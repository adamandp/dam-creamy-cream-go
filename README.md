# dam-creamy-cream-go

Configuration is loaded from environment variables using Viper. Application settings: `APP_ENV` controls logging (`development` enables debug text logs),
and `PORT` selects the HTTP port (1–65535).

```sh
APP_ENV=development PORT=3000 go run ./cmd/api
```

Viper also loads an optional `.env` file from the current working directory:

```dotenv
APP_ENV=development
PORT=3000
```

Run `go run ./cmd/api` from the project root. Exported shell variables override
values in `.env`. Application settings must be supplied through either source.

The public `GET /health/go` endpoint returns HTTP `200` with
`{"data":{"status":"ok"}}` when the application can serve requests. Responses use
`Cache-Control: no-store` so probes receive a fresh result.

```sh
curl --fail http://localhost:3000/health/go
```

Use this endpoint for liveness probes. It does not check external dependencies
or guarantee that every business operation will succeed. The separate `GET /health/db` readiness probe checks PostgreSQL using the existing
connection pool with a two-second timeout. It returns HTTP `200` with
`{"data":{"status":"ok"}}` when reachable, or HTTP `503` with `{"errors":"Database unavailable","data":{"status":"unavailable"}}`
on failure. Both probes use `Cache-Control: no-store`; database errors and
credentials are not included in the response.

Run the test suite with `go test ./...`.

PostgreSQL is connected through GORM before the HTTP server starts. Set these
separate variables in `.env` (see `.env.example`):

```dotenv
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=
DB_NAME=dam_creamy_cream
DB_SSLMODE=disable
```

`DB_USER` and `DB_NAME` are required. An empty password is supported if your local
PostgreSQL authentication allows it. Host, port, and SSL mode default to the
values above. Use an existing database and your local PostgreSQL credentials.
Startup fails if the database cannot be reached within five seconds. The pool
allows 10 open and 5 idle connections and closes after graceful HTTP shutdown
on Ctrl+C or SIGTERM. At startup, GORM AutoMigrate checks the 23 manually
maintained models against PostgreSQL. This operation may create or alter tables,
columns, indexes, and constraints. Review model changes before deploying,
especially against an existing database. Foreign-key creation is disabled during
AutoMigrate to avoid duplicating constraints created by the Prisma migrations.

HTTP responses use the shared `types.WebResponse[T]` contract with optional
`message`, `errors` (a string), `data`, and `paging` fields. Paging contains
`currentPage`, `pageSize`, `totalItems`, and `totalPages`. Unused optional fields
are omitted. Health endpoints use this envelope. Framework errors use Fiber’s default handler.

## GORM models and typed queries

All 23 models under `internal/dal/model/` are manually maintained against the
final NestJS `prisma/schema.prisma` and its SQL migrations. Preserve the exact
column names, nullable fields, SQL types, named PostgreSQL enums, constraints,
and relationships. Never regenerate models by introspecting the database.

The GORM Gen development tool lives at `tools/gormgen/generate.go`. Whenever a
model's fields change, regenerate type-safe queries **from the repository
root**, then test and commit the generated code:

```sh
go run ./tools/gormgen
go test ./...
```

Commit the generated `internal/dal/query/*.go` files to Git; they are compiled
into the production API binary. Production does not run GORM Gen.

Example in a service using the existing `*gorm.DB`:

```go
c := query.Use(db).Category
rows, err := c.WithContext(ctx).Where(c.IsActive.Is(true)).Find()
```

`internal/infrastructure/migration.go` uses the same model registry and runs
`AutoMigrate` during `Bootstrap` after PostgreSQL initialization. It is separate
from code generation. Existing PostgreSQL enum types must already exist;
this application does not recreate them. AutoMigrate is **not** additive-only:
it can change database column definitions and indexes. Validate changes on
an isolated database before deploying, and back up important data.
Foreign-key creation is disabled because the source Prisma migrations own
the foreign-key constraints.

The migration integration test under `test/internal/infrastructure/` only runs
when `GORM_MIGRATION_TEST_DSN` points to an isolated PostgreSQL database. Never
point this variable at a development or production database containing data.
