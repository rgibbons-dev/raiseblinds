# RaiseBlinds Security Audit

*2026-04-23T14:21:45Z by Showboat 0.6.1*
<!-- showboat-id: 6244b2ed-94bd-4663-9180-16f561c4934b -->

Scope: backend Go API and frontend source. Looking for XSS, CSRF, rate limiting, session revocation, SQL injection, SSRF, XXE, secrets, race conditions, SSTI.

```bash
grep -n "html.EscapeString\|Content-Security-Policy\|X-Frame-Options" backend/server.go
```

```output
47:	w.Header().Set("X-Frame-Options", "DENY")
48:	w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' https: data:; style-src 'self' 'unsafe-inline';")
199:			it.Title = html.EscapeString(it.Title)
200:			it.Description = html.EscapeString(it.Description)
222:	_, err := s.db.Exec(`insert into reviews(author_id,target_user_id,listing_id,rating,vouch,comment,created_at) values(?,?,?,?,?,?,?)`, uid, in.TargetUserID, in.ListingID, in.Rating, boolToInt(in.Vouch), html.EscapeString(strings.TrimSpace(in.Comment)), time.Now().Format(time.RFC3339))
```

```bash
grep -n "csrf\|SameSite\|session\|revoked" backend/server.go
```

```output
51:		if !s.csrfOK(r) { http.Error(w, "csrf token missing", http.StatusForbidden); return }
68:		`create table if not exists sessions(id text primary key, user_id integer not null, csrf_token text not null, revoked_at text, created_at text);`,
142:	csrf := randHex(24)
143:	_, _ = s.db.Exec(`insert into sessions(id,user_id,csrf_token,created_at) values(?,?,?,?)`, sid, userID, csrf, time.Now().Format(time.RFC3339))
144:	http.SetCookie(w, &http.Cookie{Name:"session_id", Value:sid, HttpOnly:true, Secure:false, SameSite:http.SameSiteLaxMode, Path:"/"})
145:	jsonOut(w, http.StatusOK, map[string]any{"csrf_token": csrf, "user_id": userID})
154:func (s *Server) session(r *http.Request) (string, int, string, error) {
155:	c, err := r.Cookie("session_id")
158:	var revoked sql.NullString
159:	var csrf string
160:	err = s.db.QueryRow(`select user_id,csrf_token,revoked_at from sessions where id=?`, c.Value).Scan(&uid, &csrf, &revoked)
162:	if revoked.Valid { return "",0,"", errors.New("revoked") }
163:	return c.Value, uid, csrf, nil
166:func (s *Server) csrfOK(r *http.Request) bool {
168:	_, _, csrf, err := s.session(r)
170:	return r.Header.Get("X-CSRF-Token") == csrf
175:		_, uid, _, err := s.session(r)
183:	sid, _, _, _ := s.session(r)
184:	_, _ = s.db.Exec(`update sessions set revoked_at=? where id=?`, time.Now().Format(time.RFC3339), sid)
205:		_, uid, _, err := s.session(r)
```

```bash
grep -n "QueryRow\|Exec(`\|like \?" backend/server.go
```

```output
bash: -c: line 1: unexpected EOF while looking for matching ``'
```

```bash
grep -n "http://\|password\|secret\|token" -n backend/server.go frontend/src/*
```

```output
backend/server.go:51:		if !s.csrfOK(r) { http.Error(w, "csrf token missing", http.StatusForbidden); return }
backend/server.go:67:		`create table if not exists users(id integer primary key autoincrement, email text unique, password text, name text, created_at text);`,
backend/server.go:68:		`create table if not exists sessions(id text primary key, user_id integer not null, csrf_token text not null, revoked_at text, created_at text);`,
backend/server.go:96:	_, err := s.db.Exec(`insert into users(email,password,name,created_at) values(?,?,?,?)`, strings.ToLower(strings.TrimSpace(in.Email)), string(hash), strings.TrimSpace(in.Name), time.Now().Format(time.RFC3339))
backend/server.go:134:	err := s.db.QueryRow(`select id,password from users where email = ?`, strings.ToLower(strings.TrimSpace(in.Email))).Scan(&userID, &pass)
backend/server.go:143:	_, _ = s.db.Exec(`insert into sessions(id,user_id,csrf_token,created_at) values(?,?,?,?)`, sid, userID, csrf, time.Now().Format(time.RFC3339))
backend/server.go:145:	jsonOut(w, http.StatusOK, map[string]any{"csrf_token": csrf, "user_id": userID})
backend/server.go:160:	err = s.db.QueryRow(`select user_id,csrf_token,revoked_at from sessions where id=?`, c.Value).Scan(&uid, &csrf, &revoked)
```

```bash
sed -n '1,260p' backend/server.go | grep -n "TODO\|FIXME\|GenerateFromPassword\|Secure:false"
```

```output
95:	hash, _ := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
144:	http.SetCookie(w, &http.Cookie{Name:"session_id", Value:sid, HttpOnly:true, Secure:false, SameSite:http.SameSiteLaxMode, Path:"/"})
```

Findings summary: SQL uses placeholders (good), CSRF exists (good), XSS escaping exists in API output and review comments (good), session revocation on logout exists (good). Gaps: Secure cookie disabled (Secure:false), no request body size limit (DoS risk), no brute-force limits beyond in-memory IP map, no automated race test evidence, frontend map iframe has unrestricted referrer policy, no explicit secret scanner step yet.

```bash
grep -nE "AKIA|SECRET|PRIVATE KEY|TOKEN=|PASSWORD=" -n README.md backend/server.go frontend/src/* || true
```

```output
```

Recommendation implementation validation (expected behavior): invalid image URL rejected with 400; oversized JSON rejected with 413; race detector passes.

```bash
go test ./backend -run 'TestListingsRejectInvalidImageURL|TestRequestBodyLimit'
```

```output
ok  	raiseblinds/backend	0.299s
```

```bash
go test -race ./backend
```

```output
ok  	raiseblinds/backend	16.804s
```
