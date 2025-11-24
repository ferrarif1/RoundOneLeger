# Ledger Platform

This repository contains a self-contained asset ledger backend written in Go. It exposes REST APIs for IP/personnel/system management, supports public-key based authentication, enforces IP allowlists, provides undo/redo history, and can import/export Excel workbooks that describe the ledgers individually or as a Cartesian matrix.

## Features

- **Authentication** – Ed25519 enrollment/login flows that bind users to browser fingerprints and issue bearer tokens.
- **Ledger management** – CRUD endpoints for IPs, personnel, and systems with tagging, ordering, and cross-linking.
- **Excel workflows** – Import/export XLSX workbooks with automatic IP column detection and a derived matrix sheet representing all combinations.
- **Collaborative workspaces** – Create free-form tables with dynamic columns, clipboard import, Excel syncing, and an inline rich text editor for narrative context or embedded imagery.
- **Document portability** – Keep narrative workspaces in sync with offline editors by importing and exporting DOCX files.
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
| `DB_PORT`            | `5433`      | PostgreSQL port                                  |
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

> **Image mirrors**: The compose file and `Dockerfile` default to `mirror.gcr.io` mirrors (for Postgres, Go, and Alpine) to avoid Docker Hub authentication timeouts. You can override the build args `GO_BASE_IMAGE` and `RUNTIME_BASE_IMAGE` or swap the Postgres image tag if your environment requires a different registry.

Once both containers are healthy, the API is reachable at <http://localhost:8080> with identical routes as the local build.

## Windows One-Click Bundle

Use the new helper script to compile a single Windows executable that ships both the Go backend and the compiled React SPA:

```powershell
pwsh -File scripts/build-windows-bundle.ps1
```

The script performs the following steps:

1. Installs/refreshes frontend dependencies under `web/node_modules/` (skip via `-SkipNpmInstall`).
2. Builds the production SPA into `web/dist/` (skip via `-SkipFrontendBuild` if you already ran `npm run build`).
3. Cross-compiles `./cmd/server` for Windows (`GOOS=windows GOARCH=amd64 CGO_ENABLED=0`) and writes the output to `dist/ledger.exe`.

You can override the output path with `-Output C:\path\to\ledger.exe` or change the architecture with `-Arch arm64`. After the command finishes, double-click the generated `ledger.exe` to launch the entire application at `http://localhost:8080` without separate web servers or proxies.

For non-Windows environments, run `make frontend windows-exe` to achieve the same result from a Unix shell.

## Authentication Flow

1. **Password login** – POST `/auth/password-login` with `{ "username": "…", "password": "…" }`. The server validates the credentials, issues a bearer token, and returns the username together with an `admin` flag.
2. **Session usage** – Attach `Authorization: Bearer <token>` to every `/api/v1/**` request. Sessions expire automatically after the configured TTL; the frontend stores the token in `localStorage` and clears it on logout.

首次启动前请通过环境变量为默认管理员提供口令：设置 `LEDGER_ADMIN_PASSWORD`（明文）或 `LEDGER_ADMIN_PASSWORD_HASH`（PBKDF2-HMAC-SHA256 格式），系统会据此初始化 `hzdsz_admin` 账号。登录后可在“用户中心”页面增删其他操作员帐号。只有管理员才能管理用户、IP 白名单以及台账结构，普通用户会被限制为只读视图。所有新密码需至少 10 位，同时包含大写字母、小写字母、数字与特殊字符；系统会使用 PBKDF2-HMAC-SHA256（120,000 次迭代）加盐哈希后再持久化。详见 `openapi.yaml` 获取完整的请求/响应示例和错误码。

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

The `migrations/0001_init.sql` script creates PostgreSQL tables for users, allowlists, and audit logs, and seeds the `hzdsz_admin` account with the default password (already stored as a PBKDF2-HMAC-SHA256 hash with 120k iterations). Apply it with your preferred migration tool before switching the store implementation to a database-backed version.

## API Reference

Browse `openapi.yaml` or load it into Swagger UI/Postman to explore all routes, schemas, and example payloads.

## Offline Deployment Workflow

The repository ships with a repeatable workflow for preparing and operating the application in fully offline Windows environments. Use the following procedure whenever you need to refresh dependencies or set up a new air-gapped workstation.

### 1. Collect artifacts on an internet-connected machine

1. Clone this repository and switch to the desired tag/commit.
2. Populate Go dependencies and compile the Windows binary:
   ```powershell
   go env -w GOPROXY=https://proxy.golang.org,direct
   go mod tidy
   go mod vendor
   go build -o dist/server.exe ./cmd/server
   ```
3. Download frontend assets and produce the static build:
   ```powershell
   pushd web
   npm install
   npm run build
   popd
   ```
4. Capture the resulting directories/files and npm cache:
   - `vendor/`
   - `dist/server.exe`
   - `web/node_modules/`
   - `web/package-lock.json`
   - `web/dist/`
   - npm cache directory (obtain with `npm config get cache`)
5. Place the captured content inside a folder named `offline-artifacts/` using the structure below (archives such as `.zip` are also supported by the setup script):

   ```text
   offline-artifacts/
   ├── vendor/                  # from go mod vendor
   ├── server.exe               # from go build -o dist/server.exe …
   ├── node_modules/            # from web/node_modules
   ├── dist/                    # from web/dist
   ├── npm-cache/               # npm cache directory
   ├── web-package-lock.json    # optional backup copy
   ├── docker-images/           # will store docker save outputs (see below)
   └── installers/              # optional subfolder for MSI/EXE packages
   ```

6. Download Windows installers and place them under `offline-artifacts/installers/`:
   - Go 1.22 MSI (`go1.22.windows-amd64.msi`)
   - Node.js 18 LTS MSI (`node-v18.x64.msi`)
   - PostgreSQL 15+ (`postgresql-15.4-1-windows-x64.exe` or similar)
   - Docker Desktop (`DockerDesktopInstaller.exe`)

7. Export Docker images that you plan to use offline and store them under `offline-artifacts/docker-images/`:
   ```powershell
   docker pull postgres:15-alpine
   docker pull golang:1.22-alpine      # or any base images you rely on
   docker save postgres:15-alpine -o offline-artifacts/docker-images/postgres-15-alpine.tar
   docker save golang:1.22-alpine -o offline-artifacts/docker-images/golang-1.22-alpine.tar
   ```

8. Copy the complete repository (including `offline-artifacts/`) to removable media.

### 2. Prepare the offline workstation

1. Copy the repository to the offline machine.
2. Open PowerShell (Run as Administrator is recommended for installer execution).
3. Execute the setup script:
   ```powershell
   Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass
   pwsh -File scripts/offline-setup.ps1 -InitializeDatabase -UseDocker
   ```
   - Omit `-UseDocker` to initialize PostgreSQL via `psql` instead of Docker Compose.
   - Append `-FrontendOnly` or `-BackendOnly` to limit which services start automatically.
   - Use `-OfflineRoot C:\path\to\artifacts` when your artifacts are stored outside the repository tree.
4. The script checks for Go, Node.js/npm, PostgreSQL, and Docker Desktop. Missing components are installed silently when matching installers exist in `offline-artifacts/installers/`. Otherwise, the script reports the missing dependency so you can install it manually.
5. After the tooling check completes, the script configures `GOPATH`, restores the npm cache, copies `vendor/`, `node_modules/`, and the frontend build, and places `dist/server.exe` in the correct location.

### 3. Optional database initialization

- With PostgreSQL available locally, the script applies all SQL files in `migrations/` using `psql`.
- With `-UseDocker`, the script runs `docker load` for every archive in `offline-artifacts/docker-images/` and calls `docker compose up -d` to launch the stack defined in `docker-compose.yml`.

### 4. Launching the application

- Backend + Frontend: `dist/server.exe` now embeds the compiled SPA and serves everything from `http://localhost:8080`. Double-click the executable or run it from PowerShell/CMD; the login screen will be available at the same port without an additional static file server.

### 5. Verifying the deployment

- Visit `http://localhost:8080/health` to confirm backend readiness.
- Visit `http://localhost:3000/` (or your web server host) to reach the login page.
- Use the default administrator credentials noted earlier in this document for the first sign-in.

### 6. Refreshing offline artifacts

When dependencies change, revisit the internet-connected workstation and repeat the collection process (Steps 1–8 above). Replace the contents of `offline-artifacts/` on your offline media, then rerun the PowerShell script to update the offline environment with the new artifacts.
