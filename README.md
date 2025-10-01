# Ledger Platform

This repository contains a self-contained asset ledger backend written in Go. It exposes a REST API for device/IP/personnel/system management, supports public-key based authentication, enforces IP allowlists, provides undo/redo history, and can import/export Excel workbooks that describe the ledgers individually or as a Cartesian matrix.

## Features

- **Authentication** – Ed25519 enrollment/login flows that bind users to browser/device fingerprints and issue bearer tokens.
- **Ledger management** – CRUD endpoints for IPs, devices, personnel, and systems with tagging, ordering, and cross-linking.
- **Excel workflows** – Import/export XLSX workbooks with automatic IP column detection and a derived matrix sheet representing all combinations.
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
| `FINGERPRINT_SECRET` | random      | HMAC secret for fingerprint hashing (optional)   |

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

1. **Enrollment Request** – POST `/auth/enroll-request` with `{username, device_name, public_key}` (Ed25519 public key base64). The server issues a `device_id` and `nonce`.
2. **Enrollment Complete** – POST `/auth/enroll-complete` with `{username, device_id, nonce, signature, fingerprint}` where `signature` is over the nonce and `fingerprint` is a client-generated descriptor. The server stores the fingerprint hash.
3. **Request Login Nonce** – POST `/auth/request-nonce` to fetch a short-lived nonce for the device.
4. **Login** – POST `/auth/login` with `{username, device_id, nonce, signature, fingerprint}`. Successful validation issues a bearer token used in the `Authorization: Bearer <token>` header for subsequent requests.

See `openapi.yaml` for complete request/response schemas and error codes.

## Ledgers & Excel Import/Export

- `GET /api/v1/ledgers/{type}` – Retrieve ledger entries (`type` = `ips`, `devices`, `personnel`, or `systems`).
- `POST /api/v1/ledgers/{type}` / `PUT` / `DELETE` – Manage entries with tags, attributes, and cross-ledger links.
- `POST /api/v1/ledgers/{type}/reorder` – Persist manual ordering (drag/drop style).
- `POST /api/v1/ledgers/{type}/import` – Provide a base64-encoded XLSX snippet to replace a single ledger. IP columns are auto-detected via regex even if headers are missing.
- `GET /api/v1/ledgers/export` – Download a multi-sheet workbook containing individual ledgers and a `Matrix` sheet that renders every linked combination (Cartesian product) of IP → Device → Personnel → System.
- `POST /api/v1/ledgers/import` – Replace all ledgers using a multi-sheet workbook.

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

The `migrations/0001_init.sql` script creates PostgreSQL tables for users, devices, allowlists, and audit logs, and seeds an `admin` account. Apply it with your preferred migration tool before switching the store implementation to a database-backed version.

## API Reference

Browse `openapi.yaml` or load it into Swagger UI/Postman to explore all routes, schemas, and example payloads.
