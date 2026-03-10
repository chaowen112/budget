# Budget Tracker

A personal finance management API with double-entry accounting, multi-currency support, and a React frontend.

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.25, gRPC, gRPC-Gateway |
| Frontend | React, TypeScript, Vite |
| Database | PostgreSQL 16 |
| Logging | [Logrus](https://github.com/sirupsen/logrus) (JSON structured) |
| Metrics | Prometheus |
| Build | Buf (protobuf), pnpm, Docker |

## Quick Start

### Prerequisites

- Go 1.25+
- Node.js 22+ & pnpm
- Docker & Docker Compose
- [Buf CLI](https://buf.build/docs/installation)

### Local Development

```bash
# 1. Copy env file and fill in values
cp .env.example .env

# 2. Start database
docker compose up db -d

# 3. Run migrations
make migrate-up

# 4. Install frontend dependencies
make ui-deps

# 5. Build frontend
make ui

# 6. Run the server
make run
```

The API is available at `http://localhost:8080` and gRPC on port `9090`.

### Docker Compose (Full Stack)

```bash
docker compose up --build -d
```

This starts the API, PostgreSQL, and runs migrations automatically.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `ENV` | `development` | Environment (`development` / `production`) |
| `DATABASE_URL` | `postgres://budget:budget@localhost:5432/budget?sslmode=disable` | PostgreSQL connection string |
| `SERVER_HTTP_PORT` | `8080` | HTTP / REST port |
| `SERVER_GRPC_PORT` | `9090` | gRPC port |
| `JWT_SECRET` | — | Secret key for JWT tokens (min 32 chars) |
| `JWT_ACCESS_EXPIRY` | `15m` | Access token expiry |
| `JWT_REFRESH_EXPIRY` | `30d` | Refresh token expiry |
| `LOG_LEVEL` | `info` | Logrus level (`debug`, `info`, `warn`, `error`) |
| `LOG_DIR` | `/var/log/budget` | Directory for log files |
| `EXCHANGE_RATE_API_KEY` | — | API key for exchange rate provider |
| `AI_PROVIDER` | `openai` | AI provider for transaction assistant |
| `AI_API_KEY` | — | AI provider API key |
| `AI_MODEL` | `gpt-4o-mini` | AI model name |
| `AI_BASE_URL` | `https://api.openai.com` | AI API base URL |

## Logging

Structured JSON logs powered by **logrus**. Every gRPC request is logged with:

```json
{
  "level": "info",
  "msg": "gRPC request completed",
  "method": "/budget.v1.AssetService/ListAssets",
  "duration": "12.345ms",
  "auth_type": "jwt",
  "user_id": "abc-123",
  "grpc_code": "OK",
  "time": "2026-03-11T00:45:00.000+08:00"
}
```

### Log Output

- **stdout** — always enabled
- **File** — written to `$LOG_DIR/budget.log` (default: `/var/log/budget/budget.log`)

In Docker Compose, the log directory is bind-mounted to `./logs/` on the host:

```yaml
volumes:
  - ./logs:/var/log/budget
```

### Log Levels

Set via `LOG_LEVEL` environment variable:

| Level | What gets logged |
|-------|-----------------|
| `debug` | Everything including verbose request details |
| `info` | Startup, request completions, auth events |
| `warn` | Auth failures, non-critical errors |
| `error` | Internal server errors, failed operations |

## Project Structure

```
├── api/proto/           # Protobuf definitions
├── cmd/server/          # Application entrypoint
│   ├── main.go
│   ├── static/          # Embedded frontend build
│   ├── swagger-ui/      # Swagger UI assets
│   └── openapi/         # Generated OpenAPI spec
├── gen/                 # Generated Go protobuf code
├── internal/
│   ├── config/          # Configuration loader
│   ├── handler/         # gRPC service handlers
│   ├── middleware/      # Auth interceptor with logging
│   ├── model/           # Domain models
│   ├── metrics/         # Prometheus collectors
│   ├── pkg/
│   │   ├── currency/    # Exchange rate client
│   │   ├── jwt/         # JWT token management
│   │   └── logger/      # Logrus initialisation
│   ├── repository/      # Database access layer
│   └── service/         # Business logic
├── migrations/          # PostgreSQL migrations
├── web/                 # React frontend (Vite + TypeScript)
├── docker-compose.yml
├── Dockerfile
├── Makefile
└── SKILL.md             # Openclaw agent skill definition
```

## Key Features

- **Transactions** — Income/expense tracking with category and asset linking
- **Double-Entry Accounting** — Full journal ledger with balanced entries
- **Multi-Currency** — Automatic exchange rate conversion
- **Assets & Liabilities** — Track net worth across bank accounts, investments, property, crypto
- **Budgets** — Category-level budget tracking with status reports
- **Transfers** — Move money between assets with cross-currency support
- **Saving Goals** — Track progress toward financial goals
- **CPF Tracking** — Singapore Central Provident Fund support
- **Reports** — Monthly, weekly, spending trends, net worth trends, budget tracking
- **API Keys** — External API access (up to 3 keys per user)
- **AI Assistant** — Natural language transaction parsing

## API Documentation

Swagger UI is available at `/swagger/` when the server is running.

The raw OpenAPI spec can be fetched from `/api/openapi.json`.

## Makefile Commands

```bash
make proto       # Regenerate protobuf code
make build       # Build frontend + backend
make build-go    # Build backend only
make ui          # Build frontend only
make ui-dev      # Start frontend dev server
make run         # Run the server locally
make test        # Run Go tests
make docker-up   # Start all containers
make docker-down # Stop all containers
make reset-db    # Reset the database
make lint        # Run golangci-lint
```
