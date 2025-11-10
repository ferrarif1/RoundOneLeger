# RoundOneLeger

## Overview
RoundOneLeger is a full-stack asset ledger system that centralizes inventory for IP addresses, personnel, systems, and supporting audit data while layering on collaborative workspace tooling. The backend is written in Go with a Gin-compatible router while the frontend uses React and Tailwind-style utilities.

## Key Features
- **Ledger management APIs** for IPs, personnel, and systems with support for manual CRUD, ordering, and tag metadata.
- **Excel import/export pipeline** that can normalize header variations, detect IP columns via regex, and generate multi-sheet or Cartesian reports.
- **Undo/redo history** that captures mutating operations so operators can roll back up to ten steps in either direction.
- **Authentication and allowlisting hooks** supporting SDID wallet challenge/response logins plus optional fixed-network allowlisting.
- **Tamper-evident audit log** endpoints capable of exporting signed chains and verifying historical integrity.
- **Collaborative workspace UI** offering dynamic spreadsheet-style ledgers with clipboard import, Excel syncing, and an inline rich text editor for commentary or images.

## Architecture
- **Backend (`cmd/server`, `internal/`)** – Go 1.22 service exposing REST endpoints through a vendored Gin shim, with typed ledger store, authentication helpers, and middleware.
- **Excel helpers (`internal/xlsx`)** – Lightweight XLSX reader/writer used by import/export routines without external dependencies.
- **OpenAPI contract (`openapi.yaml`)** – Documents authentication, ledger, allowlist, audit, and history endpoints for client integrations.
- **Frontend (`web/`)** – React + Vite workspace with Tailwind configuration, session hooks, ledger management screens, and Ant Design-inspired components.
- **Tooling** – Makefile targets for build/run/test, Dockerfiles for backend/frontend, docker-compose orchestration, and GitHub Actions CI stub.

## Getting Started
### Prerequisites
- Go 1.22+
- Node.js 18+ (for the web client)
- Docker & Docker Compose (optional, for containerized workflows)
- PostgreSQL instance when running against a real database (connection configured via environment variables)

### Local Backend
1. Install Go dependencies (vendored Gin shim is included, so `go mod tidy` is optional).
2. Export database environment variables if a Postgres instance is available (`DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASS`).
3. Launch the API server:
   ```bash
   make run
   ```
   The health endpoint responds at `http://localhost:8080/health` with `{"status":"ok"}`.
4. Run the test suite:
   ```bash
   make test
   ```

### Frontend Workspace
1. Navigate to `web/` and install dependencies:
   ```bash
   npm install
   ```
2. Start the development server:
   ```bash
   npm run dev
   ```
3. Configure the Vite proxy or environment variables to point API requests at the Go backend (default `http://localhost:8080`).

### Docker Compose
A combined setup is provided for local experimentation:
```bash
docker-compose up --build
```
This brings up Postgres alongside the backend container. Adjust environment variables inside `docker-compose.yml` before running if you need custom credentials.

### Authentication
- POST `/auth/password-login` with `{ "username": "…", "password": "…" }` to obtain a bearer token. The response echoes the username and an `admin` flag so the UI can tailor access.
- Include the token in `Authorization: Bearer <token>` for all authenticated API calls. Sessions automatically expire based on the server TTL and the frontend clears stored credentials on logout.
- A default administrator (`hzdsz_admin` / `Hzdsz@2025#`) is provisioned for initial access. After signing in, open the “用户中心” page to add or remove other operators. Only administrators may manage users, IP allowlists, or ledger schemas.

## Collaborative Workspaces

- `GET /api/v1/workspaces` surfaces every collaborative sheet with its dynamic columns, row data, and document metadata.
- `POST /api/v1/workspaces` / `PUT` / `DELETE` create, rename, restructure, or archive a workspace directly through the API.
- `POST /api/v1/workspaces/{id}/import/excel` ingests an uploaded XLSX file and replaces the table content while preserving the workspace shell.
- `POST /api/v1/workspaces/{id}/import/text` accepts pasted tab/CSV data for quick bulk entry without leaving the browser.
- `GET /api/v1/workspaces/{id}/export` downloads the current sheet as an Excel workbook for offline sharing or further processing.

The React console mirrors these endpoints with an interactive grid: operators can add/remove columns on demand, paste clipboard data, upload spreadsheets, and maintain a WYSIWYG-style narrative panel (including inline images) alongside the structured ledger.

## Project Layout
```
.
├── cmd/server            # API entry point and HTTP server bootstrap
├── internal/api          # HTTP handlers and router wiring
├── internal/auth         # Session token manager
├── internal/db           # Database configuration helpers
├── internal/middleware   # Shared Gin middleware (IP allowlist, etc.)
├── internal/models       # Data models and in-memory store
├── internal/xlsx         # Excel reader/writer utilities
├── migrations            # Database migration files (create your own)
├── openapi.yaml          # OpenAPI v3 specification
├── third_party/gin       # Lightweight Gin-compatible shim
└── web                   # React admin console
```

## Testing and Quality
- `make test` runs the Go unit and integration tests.
- GitHub Actions (`.github/workflows/ci.yml`) executes `go test ./...` and `go vet ./...` on every push.

## Additional Notes
- The provided `migrate` Makefile target is a placeholder; integrate your preferred migration tool (e.g., golang-migrate) as needed.

