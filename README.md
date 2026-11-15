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
on Ctrl+C or SIGTERM. No tables or migrations are created automatically.

HTTP responses use the shared `types.WebResponse[T]` contract with optional
`message`, `errors` (a string), `data`, and `paging` fields. Paging contains
`currentPage`, `pageSize`, `totalItems`, and `totalPages`. Unused optional fields
are omitted. Health endpoints use this envelope. Framework errors use Fiber’s default handler.
