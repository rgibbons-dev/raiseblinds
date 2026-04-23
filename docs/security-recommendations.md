# Security recommendations

1. Enable `Secure` cookies in production and keep `HttpOnly` + `SameSite=Lax`.
2. Add HTTP body-size limits for JSON endpoints (1MB cap) to reduce DoS risk.
3. Validate user-provided image URLs to only allow `https` and block localhost/private hosts to reduce SSRF-style abuse.
4. Add regression tests for these controls and run `go test -race` for race-condition checks.
