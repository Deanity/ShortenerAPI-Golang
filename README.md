# ShortenerAPI

A high-performance, production-grade URL Shortener REST API built with Go, MongoDB, and Redis. It provides comprehensive link management, deep analytics tracking, advanced routing rules, and developer-centric tooling including webhooks, rate limiting, and security scanning.

---

## Tech Stack

| Layer | Technology | Description |
|---|---|---|
| Language | Go 1.22+ | Type-safe, compiled runtime for high performance |
| Web Framework | Fiber v2 | Express-inspired high-speed router built on fasthttp |
| Database | MongoDB | Scalable document storage for links and analytics |
| Caching | Redis | High-speed cache for redirection path & rate limits |
| Security | JWT & API Key | Dual-layered authentication flow |
| Logging | Zerolog | Structured JSON logging for production monitoring |
| Validation | Go Validator v10 | Robust request payload validation |
| Configuration | Viper & GoDotEnv | Environment and file-based configuration management |

---

## Project Structure

```text
ShortenerAPI/
├── cmd/
│   ├── api/main.go          # Wires components, initializes DB/Redis and starts server
│   └── seed/main.go         # Populates development database with seed data
├── internal/
│   ├── domain/              # Business entities and interfaces
│   ├── repository/          # MongoDB data access implementations
│   ├── usecase/             # Business logic layer
│   ├── handler/             # HTTP endpoint request and response controllers
│   ├── middleware/          # Security middlewares (auth, rate limiting)
│   └── router/              # Fiber route definitions
├── pkg/
│   ├── config/              # Environment variable loading
│   ├── database/            # Database and Cache connection setups
│   ├── utils/               # Hash, geo-lookup, and response helpers
│   └── validator/           # Custom validation logic
├── .env.example             # Template for configuration settings
├── go.mod                   # Dependencies definition
└── README.md                # Documentation
```

---

## Getting Started

### Prerequisites

* Go 1.22 or higher
* MongoDB instance (local or Atlas)
* Redis server (local or via Docker)

To run Redis using Docker:
```bash
docker run -d -p 6379:6379 --name redis-shortener redis:alpine
```

### Installation

```bash
# Clone the repository
git clone https://github.com/Deanity/ShortenerAPI-Golang.git
cd ShortenerAPI-Golang

# Fetch dependencies
go mod tidy

# Set up local configuration environment
cp .env.example .env
```

*Note: Update the values in `.env` with your database credentials and configuration preferences.*

### Running the Server

```bash
# Start in development mode
go run ./cmd/api/main.go

# Start build process
go build -o bin/shortener-api ./cmd/api/main.go
./bin/shortener-api
```

The server listens on `http://localhost:8080` by default.

### Seed Database

Populate default system accounts and rate limit plans:
```bash
go run ./cmd/seed/main.go
```

---

## Environment Variables

Configure the following options in your `.env` file:

```ini
# Application configuration
APP_ENV=development
APP_PORT=8080
APP_BASE_URL=http://localhost:8080
APP_SECRET_KEY=YourSuperSecretJWTKeyhere

# Database connection
MONGO_URI=mongodb://localhost:27017
MONGO_DB_NAME=ShortenerAPI

# Caching layer
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

# Rate limits (requests per minute)
RATE_LIMIT_FREE=100
RATE_LIMIT_PRO=1000
RATE_LIMIT_REDIRECT=10000

# Google Safe Browsing API integration
GOOGLE_SAFE_BROWSING_API_KEY=

# Webhook configuration
WEBHOOK_TIMEOUT_SECONDS=5
WEBHOOK_MAX_RETRIES=3

# Logging preferences
LOG_LEVEL=debug
LOG_FORMAT=pretty
```

---

## API Reference

### Base URL
```text
http://localhost:8080/api/v1
```

### Authentication Header Setup

Secure endpoints require one of the following authentication headers:

```http
Authorization: Bearer <jwt_token>
```
OR
```http
X-API-Key: <your_api_key>
```

---

### Response Envelope Specifications

All API actions return a unified response envelope:

#### Successful Action
```json
{
  "success": true,
  "message": "Action completed successfully",
  "data": {
    "id": "6a2226ac3989d55d0264c067",
    "short_code": "kRpP52Ng",
    "original_url": "https://example.com"
  }
}
```

#### Erroneous Action
```json
{
  "success": false,
  "message": "Invalid request parameters provided",
  "error_code": "INVALID_BODY",
  "data": null
}
```

#### Paginated Results
```json
{
  "success": true,
  "message": "Resources retrieved successfully",
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

### Available Endpoints

#### Authentication & User Management
* **POST** `/api/v1/auth/register` - Create a new user account
* **POST** `/api/v1/auth/login` - Authenticate and receive a JWT token
* **GET** `/api/v1/auth/keys` - List active developer API keys
* **POST** `/api/v1/auth/keys` - Generate a new API key
* **DELETE** `/api/v1/auth/keys/:id` - Revoke an existing API key

#### Link Management
* **POST** `/api/v1/links` - Create a new short link (supports custom slugs, passwords, and expiry rules)
* **POST** `/api/v1/links/bulk` - Shorten up to 100 URLs in a single request
* **GET** `/api/v1/links` - List all links registered to user (supports pagination and tag filters)
* **GET** `/api/v1/links/:id` - Retrieve full metadata for a link
* **PUT** `/api/v1/links/:id` - Update short link details
* **DELETE** `/api/v1/links/:id` - Remove a short link

#### Analytics Sub-endpoints
* **GET** `/api/v1/links/:id/analytics` - View total and unique click statistics
* **GET** `/api/v1/links/:id/analytics/geo` - Geographic location data breakdown (countries, cities)
* **GET** `/api/v1/links/:id/analytics/devices` - Device platform, browser, and OS breakdowns
* **GET** `/api/v1/links/:id/analytics/referrers` - Traffic sources and referrers breakdown
* **GET** `/api/v1/links/:id/analytics/timeseries` - Chronological clicks breakdown over a given interval

#### Public Redirection & Utilities
* **GET** `/:shortCode` - Public link redirection path (applies rate limits, validates expiration and passwords)
* **POST** `/:shortCode/unlock` - Authenticate and retrieve original destination for password-protected links
* **GET** `/healthz` - Liveness health probe check

---

## Features

### Core Management
* Shorten individual URLs with optional custom slugs (Implemented)
* Bulk shortening up to 100 URLs per payload (Implemented)
* Full CRUD endpoints for managing existing short links (Implemented)
* Paginated link index with tag and active status filters (Implemented)
* Live destination edits without changing shortcode (Implemented)

### Analytics Engine
* Real-time total and unique click count tracking (Implemented)
* Geolocation analytics extracting country and city details (Implemented)
* Device profiling including device type, browser model, and operating system (Implemented)
* Referrer classification separating traffic sources (search, social, direct, email) (Implemented)
* Time-series aggregations by hour, day, week, or month (Implemented)

### Routing Constraints
* Link expiration by specific timestamp (Implemented)
* Maximum click limits per link (Implemented)
* Geographic targeting - country-specific destination rules (Planned)
* Device targeting - platform-specific destination rules (Planned)
* Split A/B testing - weighted percentage-based routing (Planned)
* Deep linking routing for App Store or Google Play Store (Planned)

### Security Features
* Password-protected redirection gates (Implemented)
* Live malware check using Google Safe Browsing API on creation (Implemented)
* Redis-based sliding window rate limiter per API key (Implemented)
* JWT and API Key dual verification layers (Implemented)

### Developer Tooling
* Consistent envelope for all JSON api payloads (Implemented)
* Webhook notifications triggered on link clicks (Planned)
* Pixel tracking injections (Planned)
* Custom domain support (Planned)

---

## Testing

```bash
# Run all tests
go test ./...

# Run tests in verbose mode
go test ./... -v

# Run with test coverage calculations
go test ./... -cover
```

---

## Development

```bash
# Run static code verification
go vet ./...

# Format all code files
gofmt -w .
```

---

## License

_"Code is art. Make it beautiful." — De4nity_
