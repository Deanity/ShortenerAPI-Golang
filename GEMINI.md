# GEMINI.md — ShortenerAPI

> Important: Instructions in this file are written in English for optimal AI processing.
> This file serves as the main project reference document (PRD) for the AI coding assistant.

---

## 1. Project Overview

- **Name**        : ShortenerAPI
- **Description** : A high-performance, production-grade URL Shortener REST API built with Go and MongoDB. Provides full link management, deep analytics, advanced routing, and developer tooling (webhooks, rate limiting, etc.).
- **Goal**        : Solve the need for a self-hosted, scalable, and feature-rich link shortening service — including bulk shortening, branded domains, geo-targeting, A/B testing, and real-time analytics — that can be consumed by any client application or third-party system.
- **Target Users**: Developers, marketers, and businesses needing programmatic short-link management with rich analytics and advanced routing capabilities.
- **Version**     : v0.1.0
- **Status**      : Active Development

---

## 2. Tech Stack

- **Language**       : Go (Golang) 1.22+
- **Framework**      : Fiber v2 (fast, Express-inspired HTTP framework for Go)
- **Database**       : MongoDB (primary store for links, analytics, users)
- **Cache**          : Redis (redirect cache, rate limiting counters, session storage)
- **ODM / Driver**   : mongo-driver/mongo (official Go MongoDB driver)
- **Auth**           : JWT (golang-jwt/jwt) — API key + Bearer token
- **Validation**     : go-playground/validator v10
- **Config**         : godotenv + viper
- **Logging**        : zerolog (structured JSON logging)
- **Testing**        : Go standard `testing` + testify + mockery (mocks)
- **HTTP Client**    : net/http (built-in, for outbound webhook calls & malware checks)
- **Package Manager**: Go modules (`go mod`)
- **Deployment**     : VPS / Cloud Run (binary deploy, no Docker)

> Never use `dep` or `glide`. Always use Go modules (`go mod tidy`, `go get`).
> Do NOT introduce Docker or containerization — run MongoDB and Redis locally.

---

## 3. Commands

```bash
# Development
go run ./cmd/api/main.go          # Run dev server
air                                # Hot reload (requires cosmtrek/air)

# Build
go build -o bin/shortener-api ./cmd/api/main.go   # Build binary

# Testing
go test ./...                      # Run all tests
go test ./... -v                   # Verbose test output
go test ./... -cover               # With coverage report
go test -run TestXxx ./pkg/...     # Run specific test

# Linting & Formatting
go vet ./...                       # Go vet checks
gofmt -w .                         # Format all Go files
golangci-lint run                  # Full linter suite (requires golangci-lint)

# Dependencies
go mod tidy                        # Clean up go.mod / go.sum
go get [package]                   # Add new dependency

# Database / Seeding
go run ./cmd/seed/main.go          # Seed initial data (plans, admin user)

# Local Services (run natively, not via Docker)
mongod --dbpath ./data/db          # Start MongoDB locally (install via mongodb.com)
redis-server                       # Start Redis locally (install via redis.io)
```

---

## 4. Project Structure

**Architecture**: Clean Architecture (Domain → Repository → Use Case → Handler)

```
ShortenerAPI/
├── cmd/
│   ├── api/
│   │   └── main.go                    # Application entry point — wires up Fiber, DB, Redis, routes
│   └── seed/
│       └── main.go                    # Database seeder — inserts plans, admin user into MongoDB
├── internal/
│   ├── domain/                        # Core business entities & interfaces (no dependencies)
│   │   ├── link.go                    # Link struct, LinkRepository & LinkUseCase interfaces
│   │   ├── analytics.go               # AnalyticsEvent struct, AnalyticsRepository & AnalyticsUseCase interfaces
│   │   ├── user.go                    # User struct, UserRepository & AuthUseCase interfaces
│   │   └── errors.go                  # Sentinel errors: ErrLinkNotFound, ErrSlugTaken, etc.
│   ├── repository/                    # MongoDB data access layer (implements domain interfaces)
│   │   ├── linkRepository.go          # CRUD + query operations for the links collection
│   │   ├── analyticsRepository.go     # Insert & aggregate operations for analytics_events collection
│   │   └── userRepository.go          # CRUD + API key operations for the users collection
│   ├── usecase/                       # Business logic layer (orchestrates repositories)
│   │   ├── linkUsecase.go             # Create, update, delete, list links; slug generation; expiry checks
│   │   ├── analyticsUsecase.go        # Aggregate click stats, geo, device, referrer, time-series
│   │   └── authUsecase.go             # Register, login, JWT issuance, API key management
│   ├── handler/                       # HTTP handlers — parse request, call usecase, return response
│   │   ├── linkHandler.go             # POST /links, GET|PUT|DELETE /links/:id, GET /links
│   │   ├── analyticsHandler.go        # GET /links/:id/analytics (stats, geo, device, referrer)
│   │   ├── redirectHandler.go         # GET /:shortCode — resolves redirect, fires async analytics
│   │   └── authHandler.go             # POST /auth/register, /auth/login, CRUD /auth/keys
│   ├── middleware/                    # Fiber middleware (applied per-route or globally)
│   │   ├── auth.go                    # Validates JWT Bearer token or X-API-Key header
│   │   ├── rateLimit.go               # Redis-based rate limiting per API key / IP
│   │   └── logger.go                  # Structured request/response logging via zerolog
│   └── router/
│       └── router.go                  # Registers all routes and applies middleware
├── pkg/
│   ├── config/                        # App configuration loaded from .env
│   │   └── config.go                  # Config struct, Load() via viper/godotenv
│   ├── database/                      # External service connection setup
│   │   ├── mongo.go                   # Opens & returns *mongo.Client, pings on startup
│   │   └── redis.go                   # Opens & returns *redis.Client, pings on startup
│   ├── utils/                         # Shared, reusable helper functions
│   │   ├── slug.go                    # GenerateSlug() — nanoid / base62 short code generation
│   │   ├── response.go                # Success(), Error(), Paginated() response envelope helpers
│   │   ├── hash.go                    # HashPassword(), CheckPassword() — bcrypt wrappers
│   │   └── geo.go                     # GetLocationFromIP() — IP → country/city via geolocation
│   └── validator/
│       └── validator.go               # Custom validation rules (URL format, slug charset, etc.)
├── .env.example                       # Template for required environment variables
├── go.mod                             # Go module definition and dependencies
├── go.sum                             # Dependency checksums
├── GEMINI.md                          # Project reference document for AI assistant
└── README.md                          # Project overview, setup guide, API usage examples
```

**File placement rules:**
- All domain entities/interfaces → `internal/domain/`
- All DB access code → `internal/repository/`
- All business logic → `internal/usecase/`
- All HTTP request/response handling → `internal/handler/`
- Shared, reusable utilities → `pkg/utils/`
- Do NOT put business logic inside handlers
- Do NOT create new top-level directories without confirmation

---

## 5. Naming Conventions

```
# Files & Packages
- Files          : camelCase           e.g.: linkRepository.go, authHandler.go, rateLimit.go
                   Format: <domain><Layer>.go  →  clearly signals what domain (link, auth, analytics)
                   and which layer (Repository, Usecase, Handler, Middleware) the file belongs to.
- Packages       : lowercase single word  e.g.: domain, usecase, handler, utils
- Test files     : <camelCaseName>_test.go  e.g.: linkUsecase_test.go, authHandler_test.go

# Inside Code
- Variables      : camelCase           e.g.: shortCode, totalClicks
- Constants      : UPPER_SNAKE_CASE    e.g.: MAX_SLUG_LENGTH, DEFAULT_TTL
- Functions      : PascalCase (exported) / camelCase (unexported)
                   e.g.: CreateLink(), generateSlug()
- Structs        : PascalCase          e.g.: Link, AnalyticsEvent, UserClaims
- Interfaces     : PascalCase + "er" suffix or descriptive
                   e.g.: LinkRepository, LinkUseCase
- Error vars     : Err + PascalCase    e.g.: ErrLinkNotFound, ErrSlugTaken
- MongoDB fields : camelCase in struct tags  e.g.: `bson:"shortCode"`
- JSON fields    : snake_case in struct tags e.g.: `json:"short_code"`

# Git Branches
- New feature    : feat/<feature-name>     e.g.: feat/bulk-shorten
- Bug fix        : fix/<bug-description>   e.g.: fix/duplicate-slug
- Hotfix         : hotfix/<name>
- Refactor       : refactor/<name>
- Chore          : chore/<name>            e.g.: chore/add-golangci-config
```

---

## 6. Code Conventions

```
# General
- Follow clean architecture: no business logic in handlers, no DB calls in use cases
- Apply DRY — extract shared logic into pkg/utils or internal helpers
- Prefer explicit over implicit; avoid global mutable state
- Keep functions small and focused (single responsibility)

# Error Handling
- ALWAYS handle errors; never use _ for errors from critical operations
- Use sentinel errors (ErrXxx) in domain layer for type-safe error checks
- Wrap errors with context: fmt.Errorf("createLink: %w", err)
- Return domain errors from use cases; translate to HTTP errors in handlers
- Never expose raw MongoDB/internal errors to the API consumer

# Struct & Interface Rules
- Define interfaces in the CONSUMER package (handler imports usecase interface)
- Use pointer receivers for methods that mutate or for large structs
- Always add `bson:"..."` and `json:"..."` tags to all exported struct fields
- Use `bson:"_id,omitempty"` for MongoDB IDs; use primitive.ObjectID type

# Import Order
1. Standard library     (fmt, time, net/http, ...)
2. Third-party packages (go.mongodb.org/mongo-driver, gofiber/fiber, ...)
3. Internal packages    (shortenerapi/internal/..., shortenerapi/pkg/...)

# Logging
- Use zerolog for all logging; never use fmt.Println in production code
- Log at correct levels: Debug (dev), Info (events), Warn (recoverable), Error (failures)
- Always include structured fields: log.Error().Err(err).Str("slug", slug).Msg("...")

# Context
- Always pass context.Context as first argument to repository and usecase functions
- Never store context in a struct; pass it per-call
```

---

## 7. API Design & Response Format

```
# URL Structure
- Public redirect : GET /:shortCode                    (no /api prefix)
- API v1          : /api/v1/<resource>

# Standard Response Envelope
All API endpoints must return this JSON structure:

Success:
{
  "success": true,
  "message": "Link created successfully",
  "data": { ... }
}

Error:
{
  "success": false,
  "message": "Slug already taken",
  "error_code": "SLUG_TAKEN",
  "data": null
}

Paginated:
{
  "success": true,
  "message": "Links fetched",
  "data": [...],
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 150,
    "total_pages": 8
  }
}

# HTTP Status Codes
- 200 OK              : Successful GET / update
- 201 Created         : Successful POST (resource created)
- 204 No Content      : Successful DELETE
- 301/302             : URL redirect (short link resolution)
- 400 Bad Request     : Validation errors, malformed input
- 401 Unauthorized    : Missing or invalid API key / JWT
- 403 Forbidden       : Valid auth but insufficient permission
- 404 Not Found       : Resource not found
- 409 Conflict        : Duplicate slug / resource conflict
- 422 Unprocessable   : Business rule violation
- 429 Too Many Requests: Rate limit exceeded
- 500 Internal Server Error: Unexpected server-side failure

# Authentication
- API Key: passed as header  X-API-Key: <key>
- JWT Bearer: passed as header  Authorization: Bearer <token>
- Public endpoints (redirect, health check) require NO auth
```

---

## 8. MongoDB Schema Design

```
# Collections

links {
  _id          : ObjectID
  short_code   : string (unique index)
  original_url : string
  custom_slug  : string | null
  custom_domain: string | null
  user_id      : ObjectID (ref: users)
  tags         : []string
  is_active    : bool
  password_hash: string | null       # for password-protected links
  expires_at   : timestamp | null    # link expiration date
  click_limit  : int | null          # max clicks before expiry
  click_count  : int (default: 0)
  unique_clicks: int (default: 0)
  deep_link    : { ios: string, android: string } | null
  geo_rules    : [{ country: string, url: string }]
  device_rules : [{ device: string, url: string }]
  ab_variants  : [{ url: string, weight: int }]
  pixels       : [{ type: "gtag|fbpixel", id: string }]
  webhook_url  : string | null
  created_at   : timestamp
  updated_at   : timestamp
}

analytics_events {
  _id          : ObjectID
  link_id      : ObjectID (ref: links)
  short_code   : string (indexed)
  clicked_at   : timestamp (indexed)
  ip_address   : string (hashed for privacy)
  country      : string
  city         : string
  device_type  : string  # "mobile" | "desktop" | "tablet"
  browser      : string
  os           : string
  referrer     : string
  referrer_type: string  # "social" | "email" | "direct" | "search" | "other"
  is_unique    : bool
  user_agent   : string
}

users {
  _id          : ObjectID
  email        : string (unique)
  password_hash: string
  name         : string
  plan         : string  # "free" | "pro" | "enterprise"
  api_keys     : [{ key_hash: string, label: string, created_at: timestamp }]
  custom_domains: []string
  is_active    : bool
  created_at   : timestamp
  updated_at   : timestamp
}

# Indexing Strategy
- links: unique index on short_code; index on user_id; index on custom_domain+short_code
- analytics_events: compound index on (link_id, clicked_at); index on short_code
- users: unique index on email
```

---

## 9. Feature Specification

```
# CORE FEATURES
[CORE-1] Shorten URL          : POST /api/v1/links — create one short link
[CORE-2] Bulk Shorten         : POST /api/v1/links/bulk — shorten up to 100 URLs in one request
[CORE-3] Custom Slug/Alias    : Optional "slug" field on creation (e.g., /DiskonRamadan)
[CORE-4] Redirect             : GET /:shortCode — 301/302 redirect with tracking
[CORE-5] CRUD Links           : GET|PUT|DELETE /api/v1/links/:id
[CORE-6] List Links           : GET /api/v1/links — paginated, filterable by tag/date/status
[CORE-7] Dynamic URL Edit     : PUT /api/v1/links/:id — change destination without changing short code

# MANAGEMENT FEATURES
[MGMT-1] Tagging              : Add/remove tags per link; filter by tag
[MGMT-2] Custom Domain        : Associate a custom domain; serve redirects from it
[MGMT-3] API Key Management   : CRUD user API keys with labels (GET|POST|DELETE /api/v1/auth/keys)

# ANALYTICS FEATURES
[ANAL-1] Click Stats          : GET /api/v1/links/:id/analytics — total & unique clicks
[ANAL-2] Geo Analytics        : Breakdown by country / city
[ANAL-3] Device Analytics     : Breakdown by device type, browser, OS
[ANAL-4] Referrer Analytics   : Traffic source breakdown
[ANAL-5] Time-Series          : Clicks over time (hourly/daily/weekly/monthly)

# ADVANCED FEATURES
[ADV-1]  Deep Linking         : Route mobile users to iOS App Store / Android Play Store
[ADV-2]  Geo-Targeting        : Redirect based on user country (override destination URL)
[ADV-3]  Device Targeting     : Redirect based on device type
[ADV-4]  Pixel Retargeting    : Inject Google Tag (gtag) / Facebook Pixel on redirect page
[ADV-5]  A/B Testing          : Split traffic across N URLs by percentage weight
[ADV-6]  Link Expiration      : Auto-expire by date OR click limit; return 410 when expired

# SECURITY FEATURES
[SEC-1]  Password Protection  : Lock link with bcrypt-hashed password; unlock via POST /:code/unlock
[SEC-2]  Malware/Spam Check   : On link creation, check URL against Google Safe Browsing API
[SEC-3]  SSL Enforcement      : All API and redirect traffic over HTTPS
[SEC-4]  Rate Limiting        : Per-API-key rate limiting via Redis (configurable per plan)

# DEVELOPER FEATURES
[DEV-1]  Webhooks             : POST click event payload to webhook_url on each click (async)
[DEV-2]  JSON Response Format : Consistent JSON envelope on all endpoints
[DEV-3]  Health Check         : GET /healthz — liveness probe
[DEV-4]  API Versioning       : All endpoints under /api/v1/
```

---

## 10. Security Rules

```
# Authentication
- All /api/v1/ endpoints require authentication (API Key or JWT Bearer)
- API keys stored as bcrypt hash in DB; never stored in plaintext
- JWT tokens expire in 24h; use refresh token pattern for long sessions

# Input Validation
- Validate ALL request body fields using go-playground/validator
- Sanitize and validate original_url — must be a valid HTTP/HTTPS URL
- Validate custom slug: alphanumeric + hyphens/underscores only, 3-50 chars
- Reject reserved slugs: ["api", "healthz", "static", "admin", ...]

# Rate Limiting
- Default: 100 requests/minute per API key (free plan)
- Pro plan: 1000 requests/minute
- Redirect endpoint: 10,000 requests/minute per IP (to prevent abuse)
- Respond with 429 + Retry-After header when limit exceeded

# Data Privacy
- Store only HASHED IP addresses for analytics (SHA256 + salt)
- Comply with GDPR-friendly data minimization
- Do NOT log full user-agent strings in production (only parsed info)

# MongoDB Security
- Use parameterized queries (mongo-driver handles this with BSON)
- Never construct MongoDB filter documents from raw user input strings
- Always validate ObjectIDs before use: primitive.ObjectIDFromHex()

# Never Do
- Never hardcode credentials, secrets, or API keys in source code
- Never expose stack traces or internal errors to API consumers
- Never skip validation on any incoming request body
- Never expose MongoDB error messages directly in API responses
```

---

## 11. Performance Rules

```
# Redirect Performance (Critical Path)
- Cache shortCode → originalURL mapping in Redis (TTL: 1 hour)
- Use 301 (permanent) for static links, 302 (temporary) for links with rules/expiry
- Analytics events must be written ASYNCHRONOUSLY (goroutine / channel)
  to NOT block the redirect response

# MongoDB
- Always use indexes for queries (never do collection scans in production)
- Use projections to fetch only needed fields
- Use aggregation pipelines for analytics queries (not application-side grouping)
- Set connection pool size appropriately (default: 100)

# Redis
- Use Redis for: redirect cache, rate limit counters, deduplication (unique clicks)
- TTL all Redis keys; never store without expiry
- Use INCRBY + EXPIRE for atomic rate limit counters

# Goroutines & Concurrency
- Use goroutines for: webhook calls, analytics writes, malware checks
- Always use context with timeout for all DB and HTTP calls
- Use sync.WaitGroup or errgroup for bounded concurrent operations
- Do NOT leak goroutines — always ensure completion or cancellation

# API Performance
- Paginate all list endpoints (default: 20, max: 100 per page)
- Add gzip compression middleware on all JSON responses
```

---

## 12. Testing Rules

```
# Testing Approach
- Types    : Unit tests (use cases, utils), Integration tests (handlers + DB)
- Framework: Go standard testing + testify/assert + mockery (generated mocks)
- Target   : Minimum 70% coverage on usecase and handler packages

# What to Test
- All use case business logic (happy path + all error cases)
- All handler HTTP response codes and response body shapes
- All utility functions (slug generation, validation, hashing)
- Webhook delivery and retry logic
- Rate limiting behavior

# What NOT to Test
- MongoDB driver internals
- Third-party library behavior
- Simple one-line getter/setter functions

# Test File Rules
- One test file per source file: linkUsecase_test.go tests linkUsecase.go
- Mock all external dependencies (DB, Redis, external HTTP) using mockery
- Use table-driven tests for functions with multiple input scenarios
- Test naming: TestFunctionName_Scenario (e.g., TestCreateLink_SlugAlreadyTaken)
- Use AAA pattern: Arrange → Act → Assert

# Integration Tests
- Use a real MongoDB instance via Docker (docker compose up mongo)
- Isolate each test: create a unique DB per test run, drop after
- Tag integration tests with //go:build integration to run separately
  go test -tags=integration ./...
```

---

## 13. Git Rules

Commit after every meaningful unit of work. Keep history clean and reviewable.

```
# Commit Message Format (Conventional Commits)
feat     : short description of new feature
fix      : short description of bug fixed
refactor : code restructuring without behavior change
test     : add or update tests
docs     : documentation changes
chore    : config, deps, tooling changes
perf     : performance improvement

# Examples
feat: add bulk URL shortening endpoint
feat: implement geo-targeting redirect rules
fix: prevent duplicate slug creation under race condition
refactor: extract redirect logic into dedicated use case
test: add unit tests for A/B traffic splitter
chore: add golangci-lint config and GitHub Actions workflow

# Rules
- Never commit .env or any file containing secrets
- Never commit binaries or build artifacts (add to .gitignore)
- One logical change per commit
- Write commit messages in English, imperative mood ("add", "fix", not "added", "fixed")
```

---

## 14. Features Tracker

### ✅ Completed
*(none yet — project is initializing)*

### 🔄 In Progress — DO NOT modify without confirmation
*(none yet)*

### 📋 Planned (in priority order)

**Phase 1 — Core (MVP)**
- [ ] Project scaffold: Fiber, MongoDB, Redis, Clean Architecture setup
- [ ] Auth: User registration, login, JWT issuance
- [ ] API Key: Create/revoke API keys per user
- [ ] [CORE-1] Shorten single URL
- [ ] [CORE-4] Redirect with basic click tracking
- [ ] [CORE-5] CRUD for links
- [ ] [CORE-6] List links (paginated)

**Phase 2 — Management & Analytics**
- [ ] [CORE-2] Bulk shorten (up to 100 URLs)
- [ ] [CORE-3] Custom slug/alias support
- [ ] [MGMT-1] Tagging system
- [ ] [ANAL-1] Click stats per link
- [ ] [ANAL-2] Geo analytics
- [ ] [ANAL-3] Device/browser analytics
- [ ] [ANAL-4] Referrer analytics
- [ ] [ANAL-5] Time-series click data
- [ ] [SEC-2] Malware/spam URL check on creation

**Phase 3 — Advanced Features**
- [ ] [ADV-1] Deep linking (iOS / Android)
- [ ] [ADV-2] Geo-targeting rules
- [ ] [ADV-3] Device-based targeting
- [ ] [ADV-4] Pixel retargeting injection
- [ ] [ADV-5] A/B testing / traffic splitting
- [ ] [ADV-6] Link expiration (date + click limit)

**Phase 4 — Security & Developer**
- [ ] [SEC-1] Password-protected links
- [ ] [MGMT-2] Custom domain support
- [ ] [MGMT-3] Team API key management
- [ ] [DEV-1] Webhooks — real-time click delivery
- [ ] [SEC-4] Per-plan rate limiting via Redis

---

## 15. Environment Variables

```bash
# Copy to .env for local development
# Never commit .env to the repository

# Application
APP_ENV=development                   # development | production
APP_PORT=8080                         # HTTP server port
APP_BASE_URL=http://localhost:8080    # Base URL used to build short links
APP_SECRET_KEY=<random-32-char-string> # JWT signing secret

# MongoDB
MONGO_URI=mongodb://localhost:27017
MONGO_DB_NAME=shortener_api

# Redis
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=                       # Empty for local dev
REDIS_DB=0

# Rate Limiting
RATE_LIMIT_FREE=100                   # Requests per minute (free plan)
RATE_LIMIT_PRO=1000                   # Requests per minute (pro plan)
RATE_LIMIT_REDIRECT=10000             # Redirect endpoint per IP per minute

# Security
GOOGLE_SAFE_BROWSING_API_KEY=         # For malware/spam URL scanning

# Webhook
WEBHOOK_TIMEOUT_SECONDS=5            # Timeout for outbound webhook POST calls
WEBHOOK_MAX_RETRIES=3                # Retry attempts for failed webhook deliveries

# Logging
LOG_LEVEL=debug                      # debug | info | warn | error
LOG_FORMAT=pretty                    # pretty (dev) | json (production)
```

---

## 16. Do Not — Strict Rules for the AI Assistant

If instructions or prompts are ambiguous, **ASK FIRST** before writing code.
Do not assume and proceed without confirmation.

```
# Structure & Files
- Do NOT create new top-level directories without confirmation
- Do NOT delete files without confirmation
- Do NOT move files without confirmation

# Code
- Do NOT put business logic inside HTTP handlers — always in use cases
- Do NOT make direct DB calls from handlers — always go through use cases → repositories
- Do NOT use global variables for state (except initialized clients/config)
- Do NOT ignore returned errors from any function call
- Do NOT use panic() in request handlers (use error returns instead)
- Do NOT hardcode any value that belongs in .env

# Patterns Banned
- Do NOT use ORM/ODM libraries (use official mongo-driver/mongo directly)
- Do NOT write raw string interpolation for MongoDB queries
- Do NOT skip context propagation (always pass ctx to DB and HTTP calls)

# Database
- Do NOT run destructive operations (drop, deleteMany) without explicit confirmation
- Do NOT create MongoDB indexes manually in code outside the startup/migration script
- Do NOT expose raw MongoDB errors in API responses

# Security
- Do NOT expose any API key, JWT secret, or credential in code or logs
- Do NOT bypass input validation on any endpoint
- Do NOT skip error handling in any API route or goroutine

# Docker / Containerization
- Do NOT add Dockerfile, docker-compose.yml, or any container config
- Do NOT suggest Docker as a solution for running MongoDB or Redis
- Always assume MongoDB and Redis are running locally on the host machine
```

---

*Update this file whenever the project structure, tech decisions, or feature scope changes. The more accurate this file, the better the AI assistant performs.*