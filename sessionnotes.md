# Session Notes (handoff)

## What was built
- Implemented a Go backend (`backend/server.go`) using SQLite for:
  - registration/login
  - cookie-based sessions with CSRF token checks
  - logout/session revocation
  - listing create + search (with map coordinates)
  - reviews + vouch/reputation endpoint
- Added backend tests (`backend/server_test.go`) that cover:
  - auth/listing/search/review flow
  - rate limiting
  - password hashing
  - invalid image URL rejection
  - request body limits
- Added frontend scaffold (`frontend/`) with SolidJS + Vite:
  - search UI and sample listings (`src/market.ts`)
  - map embed and vouch/rating UI (`src/App.tsx`)
  - PWA files (`manifest.webmanifest`, `service-worker.js`)
  - test setup via Vitest
- Added security audit artifacts:
  - `docs/security-audit.showboat.md`
  - `docs/security-recommendations.md`

## Important implementation details
- Passwords are hashed with bcrypt before insert.
- Requests use JSON decoding with unknown-field rejection and size caps.
- SQL is parameterized.
- Rate limiting is currently in-memory and keyed by client IP.
- `Secure` cookie flag is enabled only when `APP_ENV=production`.
- Image URL validation currently requires HTTPS and blocks localhost/private-host patterns.

## Open caveats / future follow-up
- Auth/session model is basic (single-session table, no refresh/rotation strategy).
- Rate limiter is process-memory only and not suitable for horizontal scaling.
- Frontend currently uses sample data and does not yet integrate all backend endpoints.
- Consider adding stricter CSP and iframe/referrer constraints in frontend rendering.

## Common commands
- Backend test: `go test ./...`
- Backend race test: `go test -race ./backend`
- Frontend test: `cd frontend && npm test`
- Frontend dev server: `cd frontend && npm run dev`
- Backend run: `go run ./cmd/api`

## Environment assumptions
- Go module root is repository root.
- Frontend lockfile is committed (`frontend/package-lock.json`).
- SQLite DB defaults to `raiseblinds.db` in project root.
