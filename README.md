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

1. **Request nonce** – POST `/auth/request-nonce`. The response contains a unique `nonce` plus a readable `message` that the SDID 钱包 should sign. When the UI is hosted from a static share and cannot reach this endpoint, it mints a local challenge before invoking the extension.
2. **Wallet signature** – 浏览器插件调用 `requestLogin` 后会返回完整的认证结果，其中包含 DID、带 ECDSA P-256 公钥的 `publicKeyJwk`，以及 `signature`/`proof.signatureValue` 中的签名和 canonical 请求体。
3. **Login** – POST `/auth/login` with `{nonce, response}`，其中 `response` 为插件返回的原始对象。服务端会重建 canonical 请求、基于提供的 JWK 校验 DER 编码的签名，并在成功后签发令牌；如果身份不包含管理员角色，则还需在 SDID 侧完成管理员认证（例如返回 `authorized: true`），否则接口会返回 `identity_not_approved`。

前端登录页会优先使用 WebCrypto 验证签名；在仅 HTTP 的内网环境下若 `crypto.subtle` 被禁用，则自动切换到 TypeScript 实现的 P-256 验证，不会阻断流程。登录成功后，“连接 SDID 登录”按钮会替换为“`${用户名} 已登陆`”，并在页面展示 SDID 响应的摘要与原始 JSON 供核对。

All 身份管理逻辑由 SDID 完成，本系统仅校验签名并确认非管理员账号已经获得管理员认证。可选的 IP 白名单仍可在控制台中维护以限制访问来源。详见 `openapi.yaml` 获取完整的请求/响应示例和错误码。

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
