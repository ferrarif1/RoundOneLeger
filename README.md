# Ledger Platform

Go backend with a small React console for IP/personnel/system ledgers. Data is persisted to PostgreSQL; snapshots can be exported/imported for backup or migration.

## Requirements
- Go 1.23+
- PostgreSQL 15+ (service defaults to port 5433)
- Docker & Docker Compose (optional)

## Quick Start
```bash
docker-compose up --build   # Postgres on 5433 + API on 8080
# or
make build && make run      # uses local Postgres via env vars
```
Health check: <http://localhost:8080/health>

## Environment
| Var | Default | Note |
| --- | --- | --- |
| DB_HOST | localhost | |
| DB_PORT | 5433 | |
| DB_NAME | ledger | |
| DB_USER | postgres | |
| DB_PASS | postgres | |
| LEDGER_ADMIN_PASSWORD | *(optional)* | Seed password for `hzdsz_admin` |
| LEDGER_ADMIN_PASSWORD_HASH | *(optional)* | PBKDF2-HMAC-SHA256 hash to seed admin |

A default admin `hzdsz_admin` is always created. Set one of the admin password env vars before first login, then create your own account and remove the default.

## Import / Export
- Snapshots live in Postgres (`snapshots` table). `LEDGER_DATA_DIR` is used only for local asset files.
- `GET /api/v1/export/all` → ZIP with `snapshot.sql` + `assets/`.
- `POST /api/v1/import/all` accepts ZIP/SQL/JSON; assets restore when present.
- XLSX round-trips remain available for ledgers/workspaces.

## Auth
- Login: `POST /auth/password-login` with username/password.
- Send `Authorization: Bearer <token>` to `/api/v1/**`.

## Tests
```bash
go test ./...
```

See `openapi.yaml` for the full API surface.
