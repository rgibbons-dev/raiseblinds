package backend

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"html"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

type Server struct {
	db          *sql.DB
	mux         *http.ServeMux
	rateMu      sync.Mutex
	loginFails  map[string]int
	rateBlocked map[string]time.Time
}

func NewTestServer(t *testing.T) http.Handler {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil { t.Fatal(err) }
	s := NewServer(db)
	if err := s.Migrate(); err != nil { t.Fatal(err) }
	return s
}

func NewServer(db *sql.DB) *Server {
	s := &Server{db: db, mux: http.NewServeMux(), loginFails: map[string]int{}, rateBlocked: map[string]time.Time{}}
	s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' https: data:; style-src 'self' 'unsafe-inline';")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
	if strings.HasPrefix(r.URL.Path, "/api/") && r.Method != http.MethodGet {
		if !s.csrfOK(r) { http.Error(w, "csrf token missing", http.StatusForbidden); return }
	}
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.HandleFunc("/api/register", s.handleRegister)
	s.mux.HandleFunc("/api/login", s.handleLogin)
	s.mux.HandleFunc("/api/logout", s.withAuth(s.handleLogout))
	s.mux.HandleFunc("/api/listings", s.handleListings)
	s.mux.HandleFunc("/api/reviews", s.withAuth(s.handleCreateReview))
	s.mux.HandleFunc("/api/users/", s.handleUserReputation)
}

func (s *Server) Migrate() error {
	stmts := []string{
		`create table if not exists users(id integer primary key autoincrement, email text unique, password text, name text, created_at text);`,
		`create table if not exists sessions(id text primary key, user_id integer not null, csrf_token text not null, revoked_at text, created_at text);`,
		`create table if not exists listings(id integer primary key autoincrement, user_id integer not null, title text, description text, price_cents integer, lat real, lng real, image_url text, created_at text);`,
		`create table if not exists reviews(id integer primary key autoincrement, author_id integer not null, target_user_id integer not null, listing_id integer not null, rating integer not null, vouch integer not null, comment text, created_at text);`,
		`create unique index if not exists uniq_review on reviews(author_id, listing_id);`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil { return err }
	}
	return nil
}

func jsonIn(r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

func jsonOut(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost { http.NotFound(w, r); return }
	var in struct{ Email, Password, Name string }
	if err := jsonIn(r, &in); err != nil {
		if strings.Contains(err.Error(), "http: request body too large") { http.Error(w, "too large", http.StatusRequestEntityTooLarge); return }
		http.Error(w, "invalid", http.StatusBadRequest); return
	}
	if len(in.Password) < 8 { http.Error(w, "invalid", http.StatusBadRequest); return }
	hash, _ := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	_, err := s.db.Exec(`insert into users(email,password,name,created_at) values(?,?,?,?)`, strings.ToLower(strings.TrimSpace(in.Email)), string(hash), strings.TrimSpace(in.Name), time.Now().Format(time.RFC3339))
	if err != nil { http.Error(w, "exists", http.StatusConflict); return }
	jsonOut(w, http.StatusCreated, map[string]any{"ok": true})
}

func clientIP(r *http.Request) string {
	h, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil { return r.RemoteAddr }
	return h
}

func (s *Server) tooMany(r *http.Request) bool {
	ip := clientIP(r)
	s.rateMu.Lock()
	defer s.rateMu.Unlock()
	if until, ok := s.rateBlocked[ip]; ok && time.Now().Before(until) { return true }
	return false
}

func (s *Server) markFail(r *http.Request) {
	ip := clientIP(r)
	s.rateMu.Lock(); defer s.rateMu.Unlock()
	s.loginFails[ip]++
	if s.loginFails[ip] >= 5 { s.rateBlocked[ip] = time.Now().Add(time.Minute) }
}
func (s *Server) markSuccess(r *http.Request) {
	ip := clientIP(r)
	s.rateMu.Lock(); defer s.rateMu.Unlock()
	delete(s.loginFails, ip); delete(s.rateBlocked, ip)
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost { http.NotFound(w, r); return }
	if s.tooMany(r) { http.Error(w, "rate limited", http.StatusTooManyRequests); return }
	var in struct{ Email, Password string }
	if err := jsonIn(r, &in); err != nil {
		if strings.Contains(err.Error(), "http: request body too large") { http.Error(w, "too large", http.StatusRequestEntityTooLarge); return }
		http.Error(w, "invalid", http.StatusBadRequest); return
	}
	var userID int
	var pass string
	err := s.db.QueryRow(`select id,password from users where email = ?`, strings.ToLower(strings.TrimSpace(in.Email))).Scan(&userID, &pass)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(pass), []byte(in.Password)) != nil {
		s.markFail(r)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	s.markSuccess(r)
	sid := randHex(24)
	csrf := randHex(24)
	_, _ = s.db.Exec(`insert into sessions(id,user_id,csrf_token,created_at) values(?,?,?,?)`, sid, userID, csrf, time.Now().Format(time.RFC3339))
	secureCookies := os.Getenv("APP_ENV") == "production"
	http.SetCookie(w, &http.Cookie{Name:"session_id", Value:sid, HttpOnly:true, Secure:secureCookies, SameSite:http.SameSiteLaxMode, Path:"/"})
	jsonOut(w, http.StatusOK, map[string]any{"csrf_token": csrf, "user_id": userID})
}

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *Server) session(r *http.Request) (string, int, string, error) {
	c, err := r.Cookie("session_id")
	if err != nil { return "", 0, "", err }
	var uid int
	var revoked sql.NullString
	var csrf string
	err = s.db.QueryRow(`select user_id,csrf_token,revoked_at from sessions where id=?`, c.Value).Scan(&uid, &csrf, &revoked)
	if err != nil { return "",0,"", err }
	if revoked.Valid { return "",0,"", errors.New("revoked") }
	return c.Value, uid, csrf, nil
}

func (s *Server) csrfOK(r *http.Request) bool {
	if strings.HasSuffix(r.URL.Path, "/login") || strings.HasSuffix(r.URL.Path, "/register") { return true }
	_, _, csrf, err := s.session(r)
	if err != nil { return false }
	return r.Header.Get("X-CSRF-Token") == csrf
}

func (s *Server) withAuth(next func(http.ResponseWriter,*http.Request,int)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, uid, _, err := s.session(r)
		if err != nil { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
		next(w, r, uid)
	}
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request, uid int) {
	if r.Method != http.MethodPost { http.NotFound(w,r); return }
	sid, _, _, _ := s.session(r)
	_, _ = s.db.Exec(`update sessions set revoked_at=? where id=?`, time.Now().Format(time.RFC3339), sid)
	jsonOut(w, http.StatusOK, map[string]any{"ok":true})
}

func (s *Server) handleListings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
		rows, _ := s.db.Query(`select id,user_id,title,description,price_cents,lat,lng,image_url,created_at from listings where lower(title) like ? or lower(description) like ? order by id desc`, "%"+q+"%", "%"+q+"%")
		defer rows.Close()
		type item struct{ ID, UserID, PriceCents int; Title, Description, ImageURL, CreatedAt string; Lat, Lng float64 }
		items := []item{}
		for rows.Next() {
			var it item
			_ = rows.Scan(&it.ID,&it.UserID,&it.Title,&it.Description,&it.PriceCents,&it.Lat,&it.Lng,&it.ImageURL,&it.CreatedAt)
			it.Title = html.EscapeString(it.Title)
			it.Description = html.EscapeString(it.Description)
			items = append(items, it)
		}
		jsonOut(w, http.StatusOK, map[string]any{"items": items})
	case http.MethodPost:
		_, uid, _, err := s.session(r)
		if err != nil { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
		var in struct{ Title string `json:"title"`; Description string `json:"description"`; ImageURL string `json:"image_url"`; PriceCents int `json:"price_cents"`; Lat float64 `json:"lat"`; Lng float64 `json:"lng"` }
		if err := jsonIn(r, &in); err != nil || strings.TrimSpace(in.Title) == "" {
		if err != nil && strings.Contains(err.Error(), "http: request body too large") { http.Error(w, "too large", http.StatusRequestEntityTooLarge); return }
		http.Error(w, "invalid", http.StatusBadRequest); return
	}
	if !isSafeImageURL(in.ImageURL) { http.Error(w, "invalid image url", http.StatusBadRequest); return }
		_, err = s.db.Exec(`insert into listings(user_id,title,description,price_cents,lat,lng,image_url,created_at) values(?,?,?,?,?,?,?,?)`, uid, strings.TrimSpace(in.Title), strings.TrimSpace(in.Description), in.PriceCents, in.Lat, in.Lng, strings.TrimSpace(in.ImageURL), time.Now().Format(time.RFC3339))
		if err != nil { http.Error(w, "db err", http.StatusInternalServerError); return }
		jsonOut(w, http.StatusCreated, map[string]any{"ok": true})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleCreateReview(w http.ResponseWriter, r *http.Request, uid int) {
	if r.Method != http.MethodPost { http.NotFound(w,r); return }
	var in struct{ TargetUserID int `json:"target_user_id"`; ListingID int `json:"listing_id"`; Rating int `json:"rating"`; Vouch bool `json:"vouch"`; Comment string `json:"comment"` }
	if err := jsonIn(r, &in); err != nil || in.Rating < 1 || in.Rating > 5 { http.Error(w, "invalid", http.StatusBadRequest); return }
	if uid == in.TargetUserID { http.Error(w, "cannot review self", http.StatusBadRequest); return }
	_, err := s.db.Exec(`insert into reviews(author_id,target_user_id,listing_id,rating,vouch,comment,created_at) values(?,?,?,?,?,?,?)`, uid, in.TargetUserID, in.ListingID, in.Rating, boolToInt(in.Vouch), html.EscapeString(strings.TrimSpace(in.Comment)), time.Now().Format(time.RFC3339))
	if err != nil { http.Error(w, "duplicate or invalid", http.StatusBadRequest); return }
	jsonOut(w, http.StatusCreated, map[string]any{"ok": true})
}

func boolToInt(v bool) int { if v { return 1 }; return 0 }

func (s *Server) handleUserReputation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet { http.NotFound(w,r); return }
	if !strings.HasSuffix(r.URL.Path, "/reputation") { http.NotFound(w,r); return }
	chunks := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(chunks) < 4 { http.NotFound(w,r); return }
	uid, err := strconv.Atoi(chunks[2])
	if err != nil { http.NotFound(w,r); return }
	var avg sql.NullFloat64
	var vouches int
	_ = s.db.QueryRow(`select avg(rating), sum(case when vouch=1 then 1 else 0 end) from reviews where target_user_id=?`, uid).Scan(&avg, &vouches)
	jsonOut(w, http.StatusOK, map[string]any{"user_id": uid, "avg_rating": avg.Float64, "vouches": vouches})
}


func isSafeImageURL(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme != "https" || u.Hostname() == "" { return false }
	h := strings.ToLower(u.Hostname())
	if h == "localhost" || h == "127.0.0.1" || strings.HasPrefix(h, "10.") || strings.HasPrefix(h, "192.168.") { return false }
	return true
}
