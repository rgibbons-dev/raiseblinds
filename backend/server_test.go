package backend

import (
  "bytes"
  "encoding/json"
  "net/http"
  "net/http/httptest"
  "testing"
)

func TestAuthListingsSearchAndVouchFlow(t *testing.T) {
  srv := NewTestServer(t)

  user := map[string]string{"email": "a@example.com", "password": "StrongPass123!", "name": "Alice"}
  rec := doJSON(t, srv, "POST", "/api/register", user, "", "")
  if rec.Code != http.StatusCreated { t.Fatalf("register=%d", rec.Code) }

  rec = doJSON(t, srv, "POST", "/api/login", map[string]string{"email":"a@example.com","password":"StrongPass123!"}, "", "")
  if rec.Code != http.StatusOK { t.Fatalf("login=%d", rec.Code) }
  cookie := rec.Result().Cookies()[0].Value
  var loginOut map[string]any
  _ = json.Unmarshal(rec.Body.Bytes(), &loginOut)
  csrf := loginOut["csrf_token"].(string)

  listing := map[string]any{"title":"Vintage Chair","description":"Great chair","price_cents":4500,"lat":37.77,"lng":-122.41,"image_url":"https://images.unsplash.com/photo-1555041469-a586c61ea9bc"}
  rec = doJSON(t, srv, "POST", "/api/listings", listing, cookie, csrf)
  if rec.Code != http.StatusCreated { t.Fatalf("listing create=%d body=%s", rec.Code, rec.Body.String()) }

  rec = doJSON(t, srv, "GET", "/api/listings?q=chair", nil, "", "")
  if rec.Code != http.StatusOK { t.Fatalf("listing search=%d", rec.Code) }
  var out map[string]any
  _ = json.Unmarshal(rec.Body.Bytes(), &out)
  items := out["items"].([]any)
  if len(items) != 1 { t.Fatalf("expected 1 item, got %d", len(items)) }

  doJSON(t, srv, "POST", "/api/register", map[string]string{"email":"b@example.com","password":"StrongPass123!","name":"Bob"}, "", "")
  rec = doJSON(t, srv, "POST", "/api/login", map[string]string{"email":"b@example.com","password":"StrongPass123!"}, "", "")
  bCookie := rec.Result().Cookies()[0].Value
  _ = json.Unmarshal(rec.Body.Bytes(), &loginOut)
  bcsrf := loginOut["csrf_token"].(string)

  review := map[string]any{"target_user_id": float64(1), "listing_id": float64(1), "rating": float64(5), "vouch": true, "comment":"Trustworthy seller"}
  rec = doJSON(t, srv, "POST", "/api/reviews", review, bCookie, bcsrf)
  if rec.Code != http.StatusCreated { t.Fatalf("review=%d body=%s", rec.Code, rec.Body.String()) }

  rec = doJSON(t, srv, "GET", "/api/users/1/reputation", nil, "", "")
  if rec.Code != http.StatusOK { t.Fatalf("reputation=%d", rec.Code) }
  _ = json.Unmarshal(rec.Body.Bytes(), &out)
  if int(out["vouches"].(float64)) != 1 { t.Fatalf("expected vouch count 1") }
}

func TestRateLimitOnLogin(t *testing.T) {
  srv := NewTestServer(t)
  doJSON(t, srv, "POST", "/api/register", map[string]string{"email":"rate@example.com","password":"StrongPass123!","name":"Rate"}, "", "")
  for i:=0;i<5;i++ {
    rec := doJSON(t, srv, "POST", "/api/login", map[string]string{"email":"rate@example.com","password":"bad"}, "", "")
    if rec.Code != http.StatusUnauthorized { t.Fatalf("expected 401 got %d", rec.Code) }
  }
  rec := doJSON(t, srv, "POST", "/api/login", map[string]string{"email":"rate@example.com","password":"bad"}, "", "")
  if rec.Code != http.StatusTooManyRequests { t.Fatalf("expected 429 got %d", rec.Code) }
}

func doJSON(t *testing.T, h http.Handler, method, path string, body any, sid, csrf string) *httptest.ResponseRecorder {
  t.Helper()
  var b []byte
  if body != nil { b, _ = json.Marshal(body) }
  req := httptest.NewRequest(method, path, bytes.NewReader(b))
  if body != nil { req.Header.Set("Content-Type", "application/json") }
  if sid != "" { req.AddCookie(&http.Cookie{Name:"session_id", Value:sid}) }
  if csrf != "" { req.Header.Set("X-CSRF-Token", csrf) }
  rec := httptest.NewRecorder()
  h.ServeHTTP(rec, req)
  return rec
}

func TestPasswordStoredHashed(t *testing.T) {
  srv := NewTestServer(t)
  rec := doJSON(t, srv, "POST", "/api/register", map[string]string{"email":"hash@example.com","password":"StrongPass123!","name":"Hash"}, "", "")
  if rec.Code != http.StatusCreated { t.Fatalf("register=%d", rec.Code) }

  s := srv.(*Server)
  var stored string
  _ = s.db.QueryRow(`select password from users where email=?`, "hash@example.com").Scan(&stored)
  if stored == "StrongPass123!" { t.Fatalf("password stored in plaintext") }
}

func TestListingsRejectInvalidImageURL(t *testing.T) {
  srv := NewTestServer(t)
  doJSON(t, srv, "POST", "/api/register", map[string]string{"email":"u@example.com","password":"StrongPass123!","name":"U"}, "", "")
  rec := doJSON(t, srv, "POST", "/api/login", map[string]string{"email":"u@example.com","password":"StrongPass123!"}, "", "")
  cookie := rec.Result().Cookies()[0].Value
  var out map[string]any
  _ = json.Unmarshal(rec.Body.Bytes(), &out)
  csrf := out["csrf_token"].(string)

  rec = doJSON(t, srv, "POST", "/api/listings", map[string]any{"title":"bad","description":"x","price_cents":100,"lat":0,"lng":0,"image_url":"http://127.0.0.1/x.png"}, cookie, csrf)
  if rec.Code != http.StatusBadRequest { t.Fatalf("expected 400 got %d", rec.Code) }
}

func TestRequestBodyLimit(t *testing.T) {
  srv := NewTestServer(t)
  huge := map[string]string{"email":"big@example.com","password":"StrongPass123!","name": string(bytes.Repeat([]byte("a"), 2<<20))}
  rec := doJSON(t, srv, "POST", "/api/register", huge, "", "")
  if rec.Code != http.StatusRequestEntityTooLarge { t.Fatalf("expected 413 got %d", rec.Code) }
}
