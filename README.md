# DAM Creamy Cream — Go API

[![Go](https://img.shields.io/badge/Go-1.27.3-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Fiber](https://img.shields.io/badge/Fiber-v3-00ACD7?logo=go&logoColor=white)](https://docs.gofiber.io/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-336791?logo=postgresql&logoColor=white)](https://www.postgresql.org/)
[![GORM](https://img.shields.io/badge/GORM-ORM-00ADD8)](https://gorm.io/)
[![Redis](https://img.shields.io/badge/Redis-DC382D?logo=redis&logoColor=white)](https://redis.io/)
[![MinIO](https://img.shields.io/badge/MinIO-C72E49?logo=minio&logoColor=white)](https://min.io/)

A Go-based backend for DAM Creamy Cream, an e-commerce application. This repository is an incremental rewrite of the [original NestJS backend](https://github.com/adamandp/dam-creamy-cream-be), using Fiber, PostgreSQL, Redis, and MinIO while preserving the existing database schema.

> **Project status:** Under active development. The application foundation, database models, generated queries, and infrastructure health checks are implemented. Business API endpoints and authentication are not yet exposed by this Go application.

## Table of contents

- [Features](#features)
- [Technology stack](#technology-stack)
- [Architecture](#architecture)
- [Project structure](#project-structure)
- [Getting started](#getting-started)
- [Configuration](#configuration)
- [API reference](#api-reference)
- [Database and GORM Gen](#database-and-gorm-gen)
- [Development and testing](#development-and-testing)
- [Deployment considerations](#deployment-considerations)
- [Troubleshooting](#troubleshooting)
- [Contributing](#contributing)

## Features

- HTTP server built with Fiber v3, with request IDs, access logging, and panic recovery.
- Validated environment-based configuration with optional local `.env` support.
- PostgreSQL connection pooling and startup schema migration with GORM.
- 23 manually maintained database models and generated type-safe query APIs.
- Redis connectivity and MinIO object storage initialization.
- Independent application, database, Redis, and MinIO health checks.
- Structured application logging and graceful shutdown on interrupt or SIGTERM.
- Configuration, model mapping, health handler, and opt-in migration integration tests.

The data model already covers users and roles, addresses, products and categories, inventory, carts, orders, payments, shipments, reviews, discounts, vouchers, analytics, and notifications. These are **schema-level capabilities**, not a list of available API endpoints.

## Technology stack

| Technology | Purpose |
| --- | --- |
| [Go](https://go.dev/) | Language and runtime toolchain (1.27.3 in `go.mod`) |
| [Fiber v3](https://docs.gofiber.io/) | HTTP framework |
| [PostgreSQL](https://www.postgresql.org/) | Relational database |
| [GORM](https://gorm.io/) | Object-relational mapping and schema migration |
| [GORM Gen](https://gorm.io/gen/) | Type-safe query generation |
| [go-redis](https://github.com/redis/go-redis) | Redis client |
| [MinIO Go SDK](https://github.com/minio/minio-go) | S3-compatible object storage client |
| [Viper](https://github.com/spf13/viper) | Application configuration |
| [go-playground/validator](https://github.com/go-playground/validator) | Configuration validation |
| [Logrus](https://github.com/sirupsen/logrus) | Application logging |
| [Air](https://github.com/air-verse/air) | Development hot reload |

See [go.mod](go.mod) for the exact dependency versions.

## Architecture

### Runtime

```mermaid
flowchart LR
    Client["HTTP client"] --> Fiber["Fiber v3"]
    Fiber --> Middleware["Request ID / Logger / Recovery"]
    Middleware --> Routes["Health routes"]
    Routes --> Handler["Health handler"]
    Handler --> Service["Health service"]
    Service --> PGClient["GORM / SQL pool"]
    Service --> RedisClient["go-redis"]
    Service --> MinIOClient["MinIO SDK"]
    PGClient --> PG[("PostgreSQL")]
    RedisClient --> Redis[("Redis")]
    MinIOClient --> MinIO[("Object storage")]
```

The health module separates route registration, HTTP response handling, and dependency checks. Shared clients are initialized during application bootstrap and injected into the module rather than created for each request.

### Application lifecycle

```mermaid
flowchart TD
    Start["cmd/api/main.go"] --> Config["Load and validate configuration"]
    Config --> Logging["Initialize Logrus"]
    Logging --> DB["Connect to PostgreSQL"]
    DB --> Migrate["AutoMigrate registered models"]
    Migrate --> Storage["Initialize MinIO"]
    Storage --> Cache["Connect to Redis"]
    Cache --> HTTP["Register middleware and routes"]
    HTTP --> Listen["Start HTTP server"]
    Listen --> Shutdown["Graceful HTTP shutdown"]
    Shutdown --> Cleanup["Close Redis and PostgreSQL"]
```

Bootstrap closes initialized resources if a later initialization step fails. PostgreSQL and Redis connectivity are required for startup. A MinIO bucket-check error is logged as a warning and allows startup to continue; an error creating a missing bucket fails startup.

## Project structure

```text
.
├── cmd/
│   └── api/
│       └── main.go             # Application entry point
├── internal/
│   ├── config/                # Environment loading and validation
│   ├── dal/
│   │   ├── model/             # Handwritten models, enums, registry
│   │   └── query/             # Generated GORM Gen query APIs
│   ├── health/                # Routes, handler, service, types
│   ├── infrastructure/        # Bootstrap, clients, logging, migration
│   └── types/                 # Shared HTTP response contracts
├── test/
│   └── internal/             # Tests organized by application package
├── tools/
│   └── gormgen/
│       └── generate.go        # Development-only query generator
├── .air.toml                  # Hot-reload configuration
├── .env.example               # Environment variable template
├── .gitignore
├── go.mod
├── go.sum
└── README.md
```

## Getting started

### 1. Prerequisites

- Go 1.27.3 or a compatible newer toolchain.
- A running PostgreSQL server and an existing database with the original Prisma schema.
- A running Redis server.
- A MinIO server with valid credentials and permission to access the configured bucket.
- Git. Node.js and npm are additionally required if provisioning the database using the original backend's Prisma migrations.

### 2. Clone and configure

```bash
git clone https://github.com/adamandp/dam-creamy-cream-go.git
cd dam-creamy-cream-go

go mod download
cp .env.example .env
```

Edit `.env` to match your local services. The example uses localhost, port 3000 for the API, and local-development storage credentials. Do not use those credentials in production.

### 3. Prepare PostgreSQL

The Go application expects the original Prisma database schema, including named PostgreSQL enum types and foreign-key constraints. An empty database alone is not sufficient.

If you already have a database migrated by the NestJS backend, configure the Go application to connect to it. For a new local environment, create an isolated database and apply the migrations from the original backend:

```bash
# Run these commands in a separate checkout of dam-creamy-cream-be.
npm ci
export DATABASE_URL='postgresql://USER:PASSWORD@localhost:5432/dam_creamy_cream'
npx prisma migrate deploy
```

Replace the placeholder credentials and create the target database before running the migrations. Match the same connection details in the Go application's `.env`.

> [!WARNING]
> The API runs GORM AutoMigrate at startup. Use a disposable local database for development and review schema changes before connecting to a database containing important data.

### 4. Start the API

From the Go repository root:

```bash
go run ./cmd/api
```

The HTTP server listens on the configured port (3000 in `.env.example`).

### 5. Verify the installation

```bash
curl --fail http://localhost:3000/health/go
curl --fail http://localhost:3000/health/db
curl --fail http://localhost:3000/health/redis
curl --fail http://localhost:3000/health/minio
```

A successful installation should return HTTP 200 from all four endpoints.

## Configuration

Configuration is loaded by Viper from the optional `.env` file in the working directory, with exported non-empty environment variables taking precedence. The resulting configuration is validated before dependency initialization.

### Application

| Variable | Default | Description |
| --- | --- | --- |
| `APP_ENV` | Required | `development` enables debug-level text logs; other values use info-level JSON logs. |
| `PORT` | Required | HTTP port, between 1 and 65535. |

### PostgreSQL

| Variable | Default | Description |
| --- | --- | --- |
| `DB_HOST` | `localhost` | Database host. |
| `DB_PORT` | `5432` | Database port. |
| `DB_USER` | Required | Database username. |
| `DB_PASSWORD` | Empty | Database password, if required. |
| `DB_NAME` | Required | Existing database name. |
| `DB_SSLMODE` | `disable` | PostgreSQL SSL mode. |

### Redis

| Variable | Default | Description |
| --- | --- | --- |
| `REDIS_HOST` | `localhost` | Redis host. |
| `REDIS_PORT` | `6379` | Redis port. |
| `REDIS_USERNAME` | Empty | Optional ACL username. |
| `REDIS_PASSWORD` | Empty | Optional password. |
| `REDIS_DB` | `0` | Non-negative logical database index. |

### MinIO

| Variable | Default | Description |
| --- | --- | --- |
| `MINIO_ENDPOINT` | `localhost:9000` | Object storage API endpoint (host and optional port, without URL scheme). |
| `MINIO_ACCESS_KEY` | Required | Storage access key. |
| `MINIO_SECRET_KEY` | Required | Storage secret key. |
| `MINIO_BUCKET` | Required | Bucket name; created on startup if missing and accessible. |
| `MINIO_USE_SSL` | `false` | Enable HTTPS for the storage client. |

All supported configuration keys are available in [.env.example](.env.example). Keep actual secrets in environment variables or an untracked environment file.

## API reference

Base URL for local development: `http://localhost:3000`.

### Health checks

| Method | Endpoint | Purpose | Status codes |
| --- | --- | --- | --- |
| GET | `/health/go` | Application liveness | 200 |
| GET | `/health/db` | PostgreSQL pool connectivity | 200, 503 |
| GET | `/health/redis` | Redis connectivity | 200, 503 |
| GET | `/health/minio` | Configured MinIO bucket accessibility | 200, 503 |

These endpoints are public and do not require authentication. All return `Cache-Control: no-store`. Dependency checks use a two-second context timeout and report a generic error rather than exposing internal connection errors or credentials.

Use `/health/go` for liveness. Use dependency-specific endpoints to assess readiness; application liveness alone does not guarantee that external dependencies are available.

### Response examples

**Success — `GET /health/go` (200 OK)**

```json
{
  "message": "Application is healthy",
  "data": {
    "status": "ok"
  }
}
```

**Dependency failure — `GET /health/db` (503 Service Unavailable)**

```json
{
  "message": "Database health check failed",
  "errors": "Database unavailable",
  "data": {
    "status": "unavailable"
  }
}
```

### Shared response contract

Explicit application responses use the generic [WebResponse[T]](internal/types/web_response.go) envelope.

| Field | Type | Description |
| --- | --- | --- |
| `message` | string | Human-readable message. |
| `errors` | string | Error description. |
| `data` | generic | Response payload. |
| `paging` | object | Optional pagination metadata. |

Pagination uses `currentPage`, `pageSize`, `totalItems`, and `totalPages`. Fields use Go's JSON `omitempty` behavior. Framework errors currently use Fiber's default error handler.

## Database and GORM Gen

### Domain models

| Domain | Registered models |
| --- | --- |
| Accounts | Role, User, UserToken, Address |
| Catalog | Product, Category, Inventory, ProductInformation |
| Shopping cart | Cart, CartItem |
| Orders and fulfillment | Order, OrderItem, OrderVoucher, Payment, Shipment |
| Promotions | Discount, ProductDiscount, Voucher, UserVoucher |
| Engagement | Review, ProductAnalytics, Notification, UserNotification |

The [model registry](internal/dal/model/registry.go) is the shared source of registered models for both GORM AutoMigrate and GORM Gen.

### Schema compatibility

Models are maintained manually against the original NestJS `prisma/schema.prisma` and SQL migrations. Preserve existing table and column names, nullable fields, SQL types, named enum types, constraints, and relationships.

- Never regenerate models by database introspection.
- Existing PostgreSQL enum types must be provisioned before AutoMigrate runs.
- Foreign-key creation during AutoMigrate is disabled because the original Prisma migrations own those constraints.
- AutoMigrate can create or alter schema objects; it is not an additive-only migration strategy.

### Generate typed queries

```mermaid
flowchart LR
    Models["Handwritten models"] --> Registry["model.All()"]
    Registry --> Gen["tools/gormgen"]
    Gen --> Queries["Generated query package"]
    Queries --> Binary["Compiled API binary"]
    Registry --> Migration["Runtime AutoMigrate"]
    Migration --> PostgreSQL[("PostgreSQL schema")]
```

When adding or changing a registered model, run these commands from the repository root:

```bash
go run ./tools/gormgen
go test ./...
```

No database connection is required for query generation. Review and commit generated changes under `internal/dal/query/` together with model changes. Do not edit generated files manually. Production builds compile the generated source and do not run the generator.

### Example: typed selection and filtering

Given an existing `*gorm.DB` named `db` and a request context `ctx`:

```go
c := query.Use(db).Category

rows, err := c.WithContext(ctx).
    Select(c.ID, c.Name, c.IsActive).
    Where(c.IsActive.Is(true)).
    Find()
```

This example uses the generated `dam-creamy-cream-go/internal/dal/query` package and applies only to data access, not an exposed category endpoint.

## Development and testing

### Hot reload

The repository includes an Air configuration that builds `./cmd/api` and restarts the local server when Go source files change.

```bash
go install github.com/air-verse/air@v1.67.4
air
```

Ensure the Go binary installation directory is on your `PATH`.

### Development commands

| Command | Purpose |
| --- | --- |
| `go run ./cmd/api` | Start the API. |
| `air` | Start with hot reload. |
| `go run ./tools/gormgen` | Regenerate typed queries. |
| `go fmt ./...` | Format Go source. |
| `go vet ./...` | Run static analysis. |
| `go test ./...` | Run the test suite. |
| `go test -race ./...` | Run tests with the race detector. |
| `go build -o bin/api ./cmd/api` | Build the API executable. |

### Test coverage

Tests live under `test/internal/`, mirroring the application packages. Current tests cover configuration loading and overrides, model table and column mappings, health response behavior, and migration compatibility against the original Prisma schema.

The migration integration test is skipped unless `GORM_MIGRATION_TEST_DSN` is set explicitly. To run it, provision a disposable database with the original Prisma migrations, then execute:

```bash
GORM_MIGRATION_TEST_DSN='postgresql://USER:PASSWORD@localhost:5432/dam_test?sslmode=disable' \
  go test -v ./test/internal/infrastructure
```

> [!CAUTION]
> Never point the migration integration test at a production database or any database containing important data. It runs AutoMigrate and inserts and deletes a test category.

## Deployment considerations

Build and run the compiled executable from the repository root:

```bash
mkdir -p bin
go build -o bin/api ./cmd/api
APP_ENV=production ./bin/api
```

Supply the remaining required configuration through the deployment environment.

- PostgreSQL pool: 10 maximum open connections, 5 maximum idle connections, and 30-minute connection lifetime.
- PostgreSQL and Redis startup connectivity checks use five-second contexts. MinIO bucket initialization uses a five-second context.
- Application logs are written to stdout as structured JSON outside development mode; Fiber access logs are configured separately.
- HTTP shutdown uses the signal-aware application context, followed by Redis and PostgreSQL cleanup.
- Use deployment-appropriate database SSL settings, HTTPS for object storage where supported, and restricted credentials.
- Back up the database and validate migrations against an isolated copy before deployment. AutoMigrate currently runs automatically on every startup.
- There is no authentication middleware or business API in the current implementation. Treat this version as an infrastructure foundation, not a production-ready e-commerce backend.

## Troubleshooting

| Symptom | What to check |
| --- | --- |
| Configuration validation fails | Required environment variables, port ranges, and PostgreSQL SSL mode. |
| PostgreSQL initialization fails | Service availability, database name, credentials, and SSL mode. |
| AutoMigrate reports a missing enum type | Apply the original Prisma migrations to the target database first. |
| Redis initialization fails | Service address, ACL credentials, and database index. |
| MinIO readiness returns 503 | Endpoint, credentials, TLS settings, bucket name, and bucket permissions. |
| Air command not found | Install Air and ensure its installation directory is on `PATH`. |

## Contributing

1. Create a focused feature or fix branch.
2. Keep changes consistent with the existing package boundaries and the NestJS schema contract.
3. Regenerate queries whenever registered models change.
4. Run formatting, static analysis, and relevant tests before opening a pull request.
5. Document behavior and API changes alongside their implementation.

Do not commit secrets, local configuration, build artifacts, or temporary files.

---

Maintained by [adamandp](https://github.com/adamandp).
