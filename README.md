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

```bash
cp .env.example .env
# Configure PostgreSQL, Redis, MinIO and JWT credentials.
go mod download
go run ./cmd/api
```

Run validation:

```bash
go vet ./...
go test ./...
```

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

```text
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
```

## Images

User and product image uploads accept JPEG, PNG or WebP up to 2 MiB. MinIO buckets remain private. The public GET /media/:scope/:id/:filename route reads files through the Go backend.

## Tests and CI

CI checks formatting, go mod tidy, go vet and go test. It runs integration tests against isolated PostgreSQL, Redis and MinIO services, covering GORM schema compatibility, JWT sessions, catalog, cart, promotions, category cache, and image persistence. The PR #2 security follow-up adds tests for catalog/inventory admin authorization, address owner-only access, private user directory/details, password PATCH bypass prevention, and immediate refresh-cookie revocation for Carts/Addresses after logout. Dummy login is intentionally unchanged. The later PR #2 correctness batch also unifies Cart/Catalog/Detail discount eligibility, gives refresh JWTs unique compact jti identifiers under the legacy VARCHAR(255) column limit, and versions category Redis cache pages on writes. All three have real-service regression coverage. Before PR #3, another PR #2 patch serializes primary-address transitions and inventory stock changes with PostgreSQL locks and rejects logout-all using revoked session tokens. This correction adds concurrent HTTP integration tests without changing dummy login or database schema.

Never run destructive tests using production credentials. Before a public deployment, configure secrets, validate all required endpoints, and restrict CORS origins.

## Migration provenance and parity

The source of truth for existing API behavior is the NestJS/Prisma backend (`adamandp/dam-creamy-cream-be`, branch `master`). Go idioms and the agreed MinIO replacement are allowed; response/business behavior remains compatible except for documented bug/security fixes. See [the parity tracker](docs/parity.md), [NestJS route inventory](docs/nest-routes.md), and [the side-by-side test](tests/e2e/README.md). Of the 19 original promotion requests, 15 return identical HTTP status and JSON, and 4 contain approved, explicitly tested corrections (role checks, ProductDiscount pagination and the /user routing defect). More scenarios cover Zod errors, UUIDs, date/numeric coercion, ownership, foreign-key handling and UTC timestamps; see the acceptance runner. Additional PR #2 hardening protects role/user privilege changes, restricts every Promotions write to current admins, and validates voucher availability, duplicates and quota under PostgreSQL row locks. Client claim dates are never trusted.
