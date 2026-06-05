# ShortenerAPI

A high-performance, production-grade **URL Shortener REST API** built with **Go** and **MongoDB**. Provides full link management, deep analytics, advanced routing, and developer tooling — webhooks, rate limiting, geo-targeting, A/B testing, and more.

---

## Tech Stack

| Layer | Technology |
|---|---|
| Language | Go 1.22+ |
| Framework | [Fiber v2](https://gofiber.io) |
| Database | MongoDB (Atlas / local) |
| Cache | Redis |
| Auth | JWT + API Key |
| Logging | zerolog (structured JSON) |
| Validation | go-playground/validator v10 |
| Config | godotenv + viper |

---

## Project Structure

```
ShortenerAPI/
├── cmd/
│   ├── api/main.go          # Entry point — wires Fiber, DB, Redis, routes
│   └── seed/main.go         # Database seeder (plans, admin user)
├── internal/
│   ├── domain/              # Core entities & interfaces
│   ├── repository/          # MongoDB data access layer
│   ├── usecase/             # Business logic layer
│   ├── handler/             # HTTP handlers (request → usecase → response)
│   ├── middleware/          # Auth, rate limiting, logging
│   └── router/              # Route registration
├── pkg/
│   ├── config/              # App config from .env
│   ├── database/            # MongoDB & Redis connections
│   ├── utils/               # Slug, hash, geo, response helpers
│   └── validator/           # Custom validation rules
├── .env.example             # Environment variable template
├── go.mod
└── README.md
```

---

## Getting Started

### Prerequisites

- [Go 1.22+](https://go.dev/dl/)
- [MongoDB](https://www.mongodb.com/try/download/community) (local or Atlas)
- [Redis](https://redis.io) — or via Docker:
  ```bash
  docker run -d -p 6379:6379 --name redis-shortener redis:alpine
  ```

### Installation

```bash
# Clone the repository
git clone https://github.com/your-username/ShortenerAPI.git
cd ShortenerAPI

# Install dependencies
go mod tidy

# Copy environment template
cp .env.example .env
# → Edit .env with your credentials (see Environment Variables below)
```

### Running the Server

```bash
# Development
go run ./cmd/api/main.go

# Hot reload (requires cosmtrek/air)
air

# Build binary
go build -o bin/shortener-api ./cmd/api/main.go
./bin/shortener-api
```

Server will start at `http://localhost:8080`.

### Seed Database

```bash
go run ./cmd/seed/main.go
```

---

## Environment Variables

Copy `.env.example` to `.env` and fill in the values:

```bash
# Application
APP_ENV=development
APP_PORT=8080
APP_BASE_URL=http://localhost:8080
APP_SECRET_KEY=<random-32-char-string>   # JWT signing secret

# MongoDB
MONGO_URI=mongodb+srv://<user>:<pass>@<cluster>.mongodb.net/
MONGO_DB_NAME=ShortenerAPI

# Redis
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

# Rate Limiting
RATE_LIMIT_FREE=100           # req/min (free plan)
RATE_LIMIT_PRO=1000           # req/min (pro plan)
RATE_LIMIT_REDIRECT=10000     # redirect req/min per IP

# Security
GOOGLE_SAFE_BROWSING_API_KEY= # Google Safe Browsing API key

# Webhook
WEBHOOK_TIMEOUT_SECONDS=5
WEBHOOK_MAX_RETRIES=3

# Logging
LOG_LEVEL=debug               # debug | info | warn | error
LOG_FORMAT=pretty             # pretty (dev) | json (production)
```

---

## API Reference

### Base URL
```
http://localhost:8080/api/v1
```

### Authentication

All `/api/v1/` endpoints require one of:
- `X-API-Key: <your-api-key>` header
- `Authorization: Bearer <jwt-token>` header

Public endpoints (redirect, health check) require **no auth**.

---

### Response Format

All responses follow a consistent envelope:

**Success**
```json
{
  "success": true,
  "message": "Link created successfully",
  "data": { }
}
```

**Error**
```json
{
  "success": false,
  "message": "Slug already taken",
  "error_code": "SLUG_TAKEN",
  "data": null
}
```

**Paginated**
```json
{
  "success": true,
  "message": "Links fetched",
  "data": [],
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 150,
    "total_pages": 8
  }
}
```

---

### Endpoints

#### Auth
| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/api/v1/auth/register` | Register a new user |
| `POST` | `/api/v1/auth/login` | Login, receive JWT token |
| `GET` | `/api/v1/auth/keys` | List API keys |
| `POST` | `/api/v1/auth/keys` | Create a new API key |
| `DELETE` | `/api/v1/auth/keys/:id` | Revoke an API key |

#### Links
| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/api/v1/links` | Create a short link |
| `POST` | `/api/v1/links/bulk` | Bulk shorten (up to 100 URLs) |
| `GET` | `/api/v1/links` | List all links (paginated) |
| `GET` | `/api/v1/links/:id` | Get link details |
| `PUT` | `/api/v1/links/:id` | Update link destination |
| `DELETE` | `/api/v1/links/:id` | Delete a link |

#### Analytics
| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/api/v1/links/:id/analytics` | Click stats (total, unique) |
| `GET` | `/api/v1/links/:id/analytics/geo` | Geo breakdown |
| `GET` | `/api/v1/links/:id/analytics/devices` | Device/browser/OS breakdown |
| `GET` | `/api/v1/links/:id/analytics/referrers` | Referrer traffic sources |
| `GET` | `/api/v1/links/:id/analytics/timeseries` | Clicks over time |

#### Redirect (Public)
| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/:shortCode` | Resolve and redirect |
| `POST` | `/:shortCode/unlock` | Unlock password-protected link |

#### Health
| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/healthz` | Liveness probe |

---

## Features

### Core
- ✅ Shorten single URL with optional custom slug
- ✅ Bulk shorten (up to 100 URLs per request)
- ✅ CRUD — create, read, update, delete links
- ✅ Paginated link listing with tag/date/status filters
- ✅ Dynamic destination edit without changing short code

### Analytics
- 📊 Total & unique click counts
- 🌍 Geo analytics (country, city)
- 📱 Device, browser, OS breakdown
- 🔗 Referrer & traffic source analysis
- 📈 Time-series click data (hourly/daily/weekly/monthly)

### Advanced Routing
- 🎯 Geo-targeting — redirect by user country
- 📲 Device targeting — different URL per device type
- 🧪 A/B testing — split traffic by percentage weight
- 🔗 Deep linking — iOS App Store / Android Play Store routing
- ⏱️ Link expiration — by date or click limit

### Security
- 🔐 Password-protected links
- 🛡️ Google Safe Browsing malware/spam check on creation
- 🚦 Redis-based rate limiting per API key (configurable per plan)
- 🔒 JWT + API Key authentication

### Developer
- 🪝 Webhooks — real-time click event delivery (async)
- 🏷️ Pixel retargeting — Google Tag / Facebook Pixel injection
- 🌐 Custom domain support
- 📋 Consistent JSON envelope on all responses

---

## Running Tests

```bash
# Run all tests
go test ./...

# Verbose output
go test ./... -v

# With coverage report
go test ./... -cover

# Run a specific test
go test -run TestCreateLink_SlugAlreadyTaken ./internal/usecase/...

# Integration tests (requires MongoDB)
go test -tags=integration ./...
```

---

## Development Commands

```bash
go vet ./...          # Static analysis
gofmt -w .            # Format all Go files
golangci-lint run     # Full linter suite
go mod tidy           # Clean up dependencies
```

---

## License

MIT
