# Ledger Backend Scaffold

This repository contains a minimal Go backend scaffold for an asset ledger system. It wires a Gin-compatible router, exposes a `/health` endpoint, and demonstrates how to configure a PostgreSQL connection using environment variables.

## Getting Started

### Prerequisites
- Go 1.22+
- Docker and Docker Compose (for containerized development)

### Environment Variables
The server reads database settings from the following environment variables (all have sensible defaults):

| Variable  | Default   |
|-----------|-----------|
| `DB_HOST` | `localhost` |
| `DB_PORT` | `5432`    |
| `DB_NAME` | `ledger`  |
| `DB_USER` | `postgres` |
| `DB_PASS` | `postgres` |

### Useful Commands

```bash
make build   # Compile the server to ./bin/server
make run     # Run the server locally on :8080
make test    # Execute Go unit tests
make migrate # Placeholder target for future migrations
```

### Running with Docker Compose

```bash
docker-compose up --build
```

The compose stack launches PostgreSQL alongside the Go service. Once both containers are healthy, the health check becomes available at <http://localhost:8080/health> and returns `{"status":"ok"}` when the database port is reachable.

## Project Layout

```
/cmd/server        # Application entry point
/internal/api      # HTTP router setup
/internal/db       # Database configuration helper
/internal/auth     # Authentication placeholders
/internal/middleware
/internal/models   # Domain model placeholders
/migrations        # Future database migrations
/openapi.yaml      # OpenAPI specification skeleton
```

## Continuous Integration

A GitHub Actions workflow at `.github/workflows/ci.yml` exercises `go test ./...` and `go vet ./...` to keep the scaffold healthy.

## Next Steps

This scaffold intentionally keeps business logic light so that future steps (models, authentication, front-end integration, etc.) can be layered on incrementally.
