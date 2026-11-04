# Dam Creamy Cream — Go Rewrite (work in progress)

A staged NestJS + Prisma → Go + Fiber v3 + GORM port, based on **adamandp/dam-creamy-cream-be**, branch `master` (inspected November 4, 2026).

**IMPORTANT: NOT YET FEATURE-COMPLETE OR PRODUCTION-READY.** This is a real Go implementation of the first batch of modules, not a complete 23-controller migration. See [`docs/parity.md`](docs/parity.md) before relying on it. Do not redirect the existing frontend to this application until contract tests pass.

## Technology

- Go **1.26+** and Fiber **v3.6.1**
- GORM, PostgreSQL **existing schema (no AutoMigrate)**
- Redis (`go-redis/v9`) for category list caching
- Cloudinary signed REST API for existing image uploads/deletes
- JWT with 15-minute access tokens and 7-day refresh cookies; bcrypt password verification
- `log/slog`; explicit constructor injection in `internal/app/bootstrap.go`
- All Go test files under `tests/`, separated from production `internal/`

## Startup

```bash
cp .env.example .env
# Set real local DATABASE_URL, REDIS_URL, ACCESS_JWT_SECRET,
# REFRESH_JWT_SECRET, and Cloudinary credentials in your own .env.
# The Go app loads .env automatically in local development.

go mod tidy
go test ./...
go run ./cmd/api
```

The Go version bundled in the artifact-building environment was **1.23.2** and it had no outbound access to fetch Go modules. Consequently, `gofmt` parsing succeeded, but `go mod tidy`, compilation, database integration, and Go tests could not be executed here. `go.sum` is generated when you run `go mod tidy` in a connected Go 1.26+ environment.

> This code connects to whatever `DATABASE_URL` you supply. Use a **disposable copy of the existing PostgreSQL database**, not production. Write endpoints are implemented and can change records. The Go program does **not** run `AutoMigrate` or migrations.

## Layout

```text
cmd/api/main.go                 application entry
internal/app/bootstrap.go       composition root / dependency wiring
internal/config/                environment configuration
internal/infrastructure/        PostgreSQL, Redis, Cloudinary clients
internal/domain/models.go       GORM models mapped to existing Prisma tables
internal/httpx/                HTTP validation, paging, envelope, errors
internal/modules/              feature handlers and services
  roles/ users/ addresses/ auth/ sessions/
  categories/ products/ carts/ inventory/ health/
tests/unit/                     unit tests (separate from core)
tests/integration/              DB integration test location
tests/e2e/                      NestJS-vs-Go regression test location
docs/                           architecture, parity, source routes
```

## Implemented endpoints

- Roles: create, list, get by ID/name, update, delete
- Categories: create, cached list, get, update, delete
- Users: list, profile, get, update, update password/image, delete
- Auth: dummy-customer-login, logout, logout-all
- Sessions: refresh access token
- Addresses: create, list, user addresses, get, update, delete
- Products: create, list, catalog, update, image, delete; `/product/:id` and `/product/information/:id` are retained
- Carts: add, remove, current-user list, with discount calculations
- Inventory: list, get by ID/product ID, update
- Health: external ping and DB ping

These endpoints are **first-pass ports**, not verified behaviorally equivalent to NestJS; see pending parity checks in `docs/parity.md`. Some deliberate security corrections in JWT verification and refresh signing differ from the original application's behavior and need user approval as part of regression review.

## Important source findings

1. `Category` maps to PostgreSQL table `category`, not `categories`.
2. Prisma `UserToken.expiredAt` and `OrderItem.finalPrice` have camelCase physical column names; the Go models explicitly map them.
3. The NestJS categories list is Redis cached for one hour using keys `categories:<page>:<limit>`.
4. Product and user images use Cloudinary (2 MiB, JPEG/PNG/WebP).
5. `@CurrentUser()` in the original backend decodes a refresh cookie without verifying the signature; the Go port verifies the signed refresh JWT to avoid impersonation.
6. The original sessions refresh method appears to sign with the refresh secret instead of the access secret; this Go port deliberately issues a real access JWT. Treat this as a **contract/behavior difference to review**, not silent parity.
7. The NestJS orders controller currently declares no HTTP methods, while CQRS checkout logic exists. The Go checkout port remains pending; no incomplete checkout endpoint has been exposed.

## Development policy

- No Go `repository.go` abstraction by default: handler → service/GORM → PostgreSQL.
- Cross-module dependencies are injected in one direction by the composition root.
- Add no new DB tables or columns without an explicit migration decision.
- No fake endpoints. Modules not yet faithfully ported are intentionally unavailable.
- Never treat `go test` success alone as 1:1 parity: compare route status, body, cookie, side effects, and errors against NestJS.
