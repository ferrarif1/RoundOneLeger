# RoundOneLeger

RoundOneLeger is a lightweight asset ledger that ships with a Go backend and a React (Vite) console. Use it to track IP addresses, people, and systems while keeping an audit trail and collaborative spreadsheet-style workspaces.

## What you get
- REST APIs for authentication, ledgers, history, audit logs, and collaborative workspaces
- Spreadsheet import/export powered by the in-repo XLSX helpers
- Session + IP allowlist guardrails and a seeded admin account (`hzdsz_admin` / `Hzdsz@2025#`)

## Repository layout
```
cmd/server            # HTTP server entry
internal/             # Handlers, auth, storage, middleware, XLSX helpers
migrations/           # Example PostgreSQL schema
openapi.yaml          # API contract
web/                  # React admin console
```

## Quick start
### Backend
```bash
make run          # starts the API on http://localhost:8080
make test         # runs Go tests
```
Environment variables such as `DB_HOST`, `DB_PORT`, `DB_USER`, and `DB_PASS` let you connect to PostgreSQL when needed.

### Frontend
```bash
cd web
npm install
npm run dev       # Vite dev server on http://localhost:5173
```
The frontend keeps API calls on the current origin when you visit the site over an IP/hostname (ideal for Nginx or other reverse proxies). During local development on `localhost` it automatically targets port `8080`. Override everything with `VITE_API_BASE_URL` when needed.

## Docker compose
Run everything (backend + PostgreSQL) with one command:
```bash
docker-compose up --build
```

## Login flow
1. `POST /auth/password-login` with the seeded admin credentials to obtain a token.
2. Send `Authorization: Bearer <token>` with subsequent `/api/v1/**` requests.
3. Use the “用户中心 / User Center” page to manage additional operators.

For detailed endpoints, check `openapi.yaml` or inspect the React client.
