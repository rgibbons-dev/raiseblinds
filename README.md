# RaiseBlinds

Craigslist/Facebook Marketplace-inspired clone with a Go + SQLite API and SolidJS frontend.

## Features
- User register/login with session cookies + CSRF token.
- Listings creation/search with geolocation for map embedding.
- Ratings/reviews and vouch counts for seller reputation.
- PWA assets (`manifest.webmanifest`, `service-worker.js`).
- Stock-image seeded sample data in frontend.
- Security audit trail in `docs/security-audit.showboat.md` and recommendations in `docs/security-recommendations.md`.

## Backend
```bash
go run ./cmd/api
```

## Frontend tests
```bash
cd frontend && npm test
```
