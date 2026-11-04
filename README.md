# Dam Creamy Cream — Go Rewrite (work in progress)

A staged NestJS + Prisma → Go + Fiber v3 + GORM port, based on **adamandp/dam-creamy-cream-be**, branch `master` (inspected November 4, 2026).

**IMPORTANT: NOT YET FEATURE-COMPLETE OR PRODUCTION-READY.** This is a real Go implementation of the first batch of modules, not a complete 23-controller migration. See [`docs/parity.md`](docs/parity.md) before relying on it. Do not redirect the existing frontend to this application until contract tests pass.

## Technology

- Go **1.26+** and Fiber **v3.6.1**
- GORM, PostgreSQL **existing schema (no AutoMigrate)**
- Redis (`go-redis/v9`) for category list caching
- Private MinIO bucket (minio-go/v7) for images; public read-only GET /media/... via the Go API
- JWT with 15-minute access tokens and 7-day refresh cookies; bcrypt password verification
- `log/slog`; explicit constructor injection in `internal/app/bootstrap.go`
- All Go test files under `tests/`, separated from production `internal/`

## Startup

```bash
cp .env.example .env
# Set real local DATABASE_URL, REDIS_URL, ACCESS_JWT_SECRET,
# REFRESH_JWT_SECRET, MINIO_* credentials and PUBLIC_API_URL in your own .env.
# The Go app loads .env automatically in local development.

go mod tidy
go test ./...
go run ./cmd/api
```

CI runs Go 1.26 with dependency resolution, vet and tests. A green CI verifies compilation and unit tests, **not** full parity against the NestJS backend and its database.

> This code connects to whatever `DATABASE_URL` you supply. Use a **disposable copy of the existing PostgreSQL database**, not production. Write endpoints are implemented and can change records. The Go program does **not** run `AutoMigrate` or migrations.

## Layout

```text
cmd/api/main.go                 application entry
internal/app/bootstrap.go       composition root / dependency wiring
internal/config/                environment configuration
internal/infrastructure/        PostgreSQL, Redis, MinIO clients
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
4. Product and user images now upload to MinIO (2 MiB, JPEG/PNG/WebP); the pre-existing NestJS backend used Cloudinary. Old Cloudinary image URLs in PostgreSQL remain readable and are not deleted automatically.
5. `@CurrentUser()` in the original backend decodes a refresh cookie without verifying the signature; the Go port verifies the signed refresh JWT to avoid impersonation.
6. The original sessions refresh method appears to sign with the refresh secret instead of the access secret; this Go port deliberately issues a real access JWT. Treat this as a **contract/behavior difference to review**, not silent parity.
7. The NestJS orders controller currently declares no HTTP methods, while CQRS checkout logic exists. The Go checkout port remains pending; no incomplete checkout endpoint has been exposed.

## Development policy

- No Go `repository.go` abstraction by default: handler → service/GORM → PostgreSQL.
- Cross-module dependencies are injected in one direction by the composition root.
- Add no new DB tables or columns without an explicit migration decision.
- No fake endpoints. Modules not yet faithfully ported are intentionally unavailable.
- Never treat `go test` success alone as 1:1 parity: compare route status, body, cookie, side effects, and errors against NestJS.

## MinIO image storage (PR #1)

- The Go server uses the S3 API at `MINIO_ENDPOINT` and the private bucket `MINIO_BUCKET`. The bucket is created automatically at startup if it does not exist and credentials have permission.
- The image endpoints remain `PUT /users/:id/image` and `PATCH /products/:id/image`; they still accept multipart `file` and return the unchanged JSON message contract.
- New uploaded files receive random versioned keys `users/<id>/<uuid>.jpg` or `products/<id>/<uuid>.png` etc, so a failed DB update cannot overwrite the previous image.
- PostgreSQL `image_url` stores an absolute URL based on `PUBLIC_API_URL`: `https://api.example.com/media/users/<id>/<uuid>.jpg`. The public `GET /media/:scope/:id/:filename` endpoint reads objects server-side; it does not expose MinIO credentials or the bucket.
- Uploaded images are constrained to 2 MiB and JPEG/PNG/WebP, checked by MIME and actual file content. Successful DB updates clean up the previous **managed MinIO** image; a DB failure removes the just-uploaded object. Legacy Cloudinary/external URLs are never deleted during this transition.
- `PUBLIC_API_URL` is required in production and must be the external HTTPS origin of this API. In local development it defaults to `http://localhost:<PORT>`. Configure your reverse proxy to expose `/media/...`.
- This changes the image-host domain without changing the `imageUrl` field. Existing Cloudinary images are not copied automatically; a bulk migration can be done separately if required.
