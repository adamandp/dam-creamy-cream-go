# Application architecture

Creamy Cream Go uses a modular monolith with explicit constructor injection.

```text
cmd/api/main.go
   |
internal/app/bootstrap.go
   |-- infrastructure.NewPostgres  -> shared GORM DB pool
   |-- infrastructure.NewRedis     -> shared cache client
   |-- infrastructure.NewMinIO     -> private image store
   |-- sessions.NewService(db, JWT secrets)
   |-- roles / users / auth / addresses
   |-- categories / products / carts / inventory
   |-- discounts / productdiscounts / vouchers / uservouchers
   |-- media / health
```

Each feature defines its own HTTP routes and services. The composition root constructs dependencies once, shares connections, and handles shutdown. GORM models use explicit table and column names from the application schema. Tests live under tests/, outside application source.

Feature packages should not create independent DB/Redis/MinIO connections. Use transactions for multi-step writes. The application does not automatically change schema definitions on startup.

## Time handling and permissions

The existing Prisma migration creates timestamp-without-time-zone columns. The GORM connection uses a UTC NowFunc; manual session/cart clock writes also use UTC, preventing a 7-hour drift on WIB development hosts. Never assume the local system timezone should be stored in these columns.

The User Vouchers feature resolves JWT bearer claims into the current, active PostgreSQL user and role on each request. Customers may claim/read only their own vouchers, while global listings and USED transitions require an admin role. The authorization code lives in the feature module, not a duplicate database connector.

Authorization is implemented in the shared sessions service (Actor/RequireActor/RequireAdmin/RequireSelfOrAdmin), which checks current active database user and role against signed access JWT claims. Roles, Users, and Promotions use this shared check; User Vouchers also serialize eligibility checks and claim insertion inside a PostgreSQL FOR UPDATE transaction. No schema migration is performed.

## Inherited route security after PR #2 hardening

The shared sessions service owns JWT verification and active refresh-cookie session validation. When a handler uses \`CookieUser\`, it now checks the signed refresh token, its backing UserToken database row, account active state and role, so logout revocation applies immediately to cookie-based Cart and address reads. Access-token middleware independently verifies current user and role. Admin-only mutations protect catalog/category/stock and promotions; addresses are owner-scoped; user list and detail are access controlled. The dummy login is intentionally preserved as a dev/test fixture.
