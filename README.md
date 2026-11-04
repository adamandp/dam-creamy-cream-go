# Creamy Cream Go Backend

Backend API built with Go 1.26, Fiber v3, GORM and PostgreSQL.

## Technology

- HTTP: Fiber v3, JSON, JWT authentication
- Database: PostgreSQL and GORM (no automatic schema changes)
- Caching: Redis via go-redis/v9
- Object storage: private MinIO bucket, delivered through GET /media/...
- JWT: 15-minute access tokens and 7-day refresh cookies
- Passwords: bcrypt
- Logging: slog
- Tests: Go unit and disposable PostgreSQL, Redis, MinIO integration suites

## Requirements

Go 1.26+, PostgreSQL with the Creamy Cream schema, Redis, and MinIO.

## Run locally

`bash
cp .env.example .env
# Configure PostgreSQL, Redis, MinIO and JWT credentials.
go mod download
go run ./cmd/api
`

Run validation:

`bash
go vet ./...
go test ./...
`

The application does not call AutoMigrate or alter database tables on startup. Use a separate development database for write API testing.

## Environment

| Variable | Purpose |
|---|---|
| PORT | HTTP port (default 3000) |
| NODE_ENV | development or production |
| DATABASE_URL | PostgreSQL connection URI |
| REDIS_URL | Redis connection URI |
| ACCESS_JWT_SECRET | Access JWT signing key |
| REFRESH_JWT_SECRET | Refresh JWT signing key |
| MINIO_ENDPOINT | S3 endpoint as host:port |
| MINIO_ACCESS_KEY | Storage access key |
| MINIO_SECRET_KEY | Storage secret |
| MINIO_BUCKET | Private image bucket |
| MINIO_USE_SSL | HTTPS to MinIO |
| PUBLIC_API_URL | Public API origin for image URLs |

## Modules

- Authentication, sessions, users, roles, addresses
- Categories, products, product detail/information, inventory, carts
- Discounts, product discount assignments, vouchers, user voucher claims
- Media delivery and health checks

See [API reference](docs/api.md) for supported routes, input requirements and response behavior.

## Project structure

`text
cmd/api/                     server entrypoint
internal/app/                dependency wiring
internal/config/             environment settings
internal/infrastructure/     PostgreSQL, Redis, MinIO connections
internal/domain/             GORM models
internal/httpx/              API response and request utilities
internal/modules/            feature handlers and services
tests/unit/                  unit tests
tests/integration/           live-service tests with disposable data
tests/fixtures/db-schema/    database fixtures
docs/                        API and architecture documentation
`

## Images

User and product image uploads accept JPEG, PNG or WebP up to 2 MiB. MinIO buckets remain private. The public GET /media/:scope/:id/:filename route reads files through the Go backend.

## Tests and CI

CI checks formatting, go mod tidy, go vet and go test. It runs integration tests against isolated PostgreSQL, Redis and MinIO services, covering GORM schema compatibility, JWT sessions, catalog, cart, promotions, category cache, and image persistence.

Never run destructive tests using production credentials. Before a public deployment, configure secrets, validate all required endpoints, and restrict CORS origins.
