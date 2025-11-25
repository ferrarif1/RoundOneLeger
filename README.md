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

## Docker image export/import
- Export:
  ```bash
  docker save -o roundoneleger-frontend.tar roundoneleger-frontend:latest
  docker save -o roundoneleger-app.tar roundoneleger-app:latest
  docker save -o postgres-15-alpine.tar mirror.gcr.io/library/postgres:15-alpine
  ```
  Expected files:
  ```
  dockerimgs/
  ├── roundoneleger-frontend.tar
  ├── roundoneleger-app.tar
  └── postgres-15-alpine.tar
  ```
- Import on a new machine:
  ```bash
  docker load -i roundoneleger-frontend.tar
  docker load -i roundoneleger-app.tar
  docker load -i postgres-15-alpine.tar
  ```
- Run (compose recommended):
  ```bash
  docker compose up -d
  ```
  Or manually:
  ```bash
  docker run -d --name db -e POSTGRES_USER=ledger -e POSTGRES_PASSWORD=ledger123 -e POSTGRES_DB=ledgerdb -p 5433:5432 mirror.gcr.io/library/postgres:15-alpine
  docker run -d --name app -p 8080:8080 --link db:db roundoneleger-app:latest
  docker run -d --name frontend -p 5173:80 roundoneleger-frontend:latest
  ```
