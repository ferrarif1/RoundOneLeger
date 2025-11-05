# Ledger Platform

This repository contains a self-contained asset ledger backend written in Go. It exposes REST APIs for IP/personnel/system management, supports public-key based authentication, enforces IP allowlists, provides undo/redo history, and can import/export Excel workbooks that describe the ledgers individually or as a Cartesian matrix.

## Features

- **Authentication** – Ed25519 enrollment/login flows that bind users to browser fingerprints and issue bearer tokens.
- **Ledger management** – CRUD endpoints for IPs, personnel, and systems with tagging, ordering, and cross-linking.
- **Excel workflows** – Import/export XLSX workbooks with automatic IP column detection and a derived matrix sheet representing all combinations.
- **Collaborative workspaces** – Create free-form tables with dynamic columns, clipboard import, Excel syncing, and an inline rich text editor for narrative context or embedded imagery.
- **Undo/redo** – Ten-level history stack for manual corrections across all ledgers.
- **Audit log chain** – Tamper-evident audit trail with verification endpoint.
- **IP allowlist** – CIDR-aware middleware that blocks requests from unapproved addresses.
- **In-memory implementation** – Runs without external services by default; optional PostgreSQL connection helpers and SQL migrations are included for production rollout.

## Project Layout

```
cmd/server            # Application entrypoint
internal/api          # HTTP handlers and routing
internal/auth         # Session token manager
internal/db           # Database connectivity helper
internal/middleware   # Shared middleware (auth, IP checks)
internal/models       # Ledger store, auth, audit, allowlist logic
internal/xlsx         # XLSX encoder/decoder used for import/export
migrations            # PostgreSQL schema bootstrap
openapi.yaml          # OpenAPI 3 definition of the REST API
third_party/gin       # Minimal Gin-compatible shim (no external deps)
web/                  # Frontend placeholder (not required to run backend)
```

## Getting Started

### Prerequisites

- Go 1.22+
- (Optional) PostgreSQL 15+ if you plan to run migrations
- Docker & Docker Compose if you prefer container-based workflows

### Environment Variables

| Variable             | Default     | Description                                      |
|----------------------|-------------|--------------------------------------------------|
| `DB_HOST`            | `localhost` | PostgreSQL host                                  |
| `DB_PORT`            | `5432`      | PostgreSQL port                                  |
| `DB_NAME`            | `ledger`    | PostgreSQL database name                         |
| `DB_USER`            | `postgres`  | PostgreSQL user                                  |
| `DB_PASS`            | `postgres`  | PostgreSQL password                              |

### Local Development

```bash
make build   # Compile the server into ./bin/server
make run     # Start the HTTP server on :8080
make test    # Run Go unit tests
make migrate # Placeholder: run migrations with your preferred tool
```

When the server is running, open <http://localhost:8080/health> to confirm it is healthy. The endpoint returns additional database status details if a PostgreSQL server is reachable.

### Docker Compose

The provided `docker-compose.yml` launches PostgreSQL and the Go API in a single command:

```bash
docker-compose up --build
```

Once both containers are healthy, the API is reachable at <http://localhost:8080> with identical routes as the local build.

## Authentication Flow

1. **Password login** – POST `/auth/password-login` with `{ "username": "…", "password": "…" }`. The server validates the credentials, issues a bearer token, and returns the username together with an `admin` flag.
2. **Session usage** – Attach `Authorization: Bearer <token>` to every `/api/v1/**` request. Sessions expire automatically after the configured TTL; the frontend stores the token in `localStorage` and clears it on logout.

A default administrator (`hzdsz_admin` / `Hzdsz@2025#`) is provisioned for first-time access. 登录后可在“用户中心”页面增删其他操作员帐号。只有管理员才能管理用户、IP 白名单以及台账结构，普通用户会被限制为只读视图。详见 `openapi.yaml` 获取完整的请求/响应示例和错误码。

## Ledgers & Excel Import/Export

- `GET /api/v1/ledgers/{type}` – Retrieve ledger entries (`type` = `ips`, `personnel`, or `systems`).
- `POST /api/v1/ledgers/{type}` / `PUT` / `DELETE` – Manage entries with tags, attributes, and cross-ledger links.
- `POST /api/v1/ledgers/{type}/reorder` – Persist manual ordering (drag/drop style).
- `POST /api/v1/ledgers/{type}/import` – Provide a base64-encoded XLSX snippet to replace a single ledger. IP columns are auto-detected via regex even if headers are missing.
- `GET /api/v1/ledgers/export` – Download a multi-sheet workbook containing individual ledgers and a `Matrix` sheet that renders every linked combination (Cartesian product) of IP → Personnel → System.
- `POST /api/v1/ledgers/import` – Replace all ledgers using a multi-sheet workbook.

## Workspace Collaboration

- `GET /api/v1/workspaces` – List flexible, spreadsheet-style ledgers with their dynamic columns, rows, and attached documentation.
- `POST /api/v1/workspaces` – Create a new workspace; the frontend offers immediate editing with column/row controls and rich text notes.
- `PUT /api/v1/workspaces/{id}` / `DELETE` – Persist structural changes or archive an obsolete workspace.
- `POST /api/v1/workspaces/{id}/import/excel` – Upload an XLSX file to replace the table data, preserving the workspace shell and document.
- `POST /api/v1/workspaces/{id}/import/text` – Paste tab/CSV content directly for quick bulk entry.
- `GET /api/v1/workspaces/{id}/export` – Download the current workspace as an Excel sheet for offline sharing.

The React console mirrors these endpoints with an interactive grid. Users can add or remove columns on demand, paste clipboard data, upload Excel files, and maintain a narrative document with formatting and inline images alongside the structured records.

## History & Audit

- `POST /api/v1/history/undo` / `redo` – Walk backward/forward up to ten steps.
- `GET /api/v1/history` – Returns undo/redo availability flags.
- `GET /api/v1/audit` – Fetch tamper-evident audit logs.
- `GET /api/v1/audit/verify` – Recomputes the audit hash chain for integrity checking.

## Testing

Run the suite locally:

```bash
go test ./...
```

The tests exercise the ledger store, authentication flow, XLSX codec, and matrix export logic to ensure the critical features remain functional.

## Migrations

The `migrations/0001_init.sql` script creates PostgreSQL tables for users, allowlists, and audit logs, and seeds the `hzdsz_admin` account with the default password. Apply it with your preferred migration tool before switching the store implementation to a database-backed version.

## API Reference

Browse `openapi.yaml` or load it into Swagger UI/Postman to explore all routes, schemas, and example payloads.
