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


## PR #2 correction: shared pricing and cache/session consistency

Product Cart, Catalog and Detail use the common \`internal/modules/pricing\` active-association predicate, inclusive validity timestamps, and discount calculator. ProductDiscount.isActive and Discount.startDate/endDate must pass for a discount price to be shown. The same logic should be reused by future Order/Checkout pricing.

Refreshing customer sessions still uses signed HS256 JWTs and a database-backed UserToken record. Each new refresh JWT has a cryptographically random 128-bit jti, compactly serialized to remain under the original token VARCHAR(255) limit. Refresh JWTs omit optional typ and iat; access JWTs are unchanged. This avoids same-second duplicate token collisions without a database migration.

Category list cache uses Redis key \`categories:revision\` as a shared generation number. Pages are stored at \`categories:v<revision>:<page>:<limit>\` with one-hour TTL; successful writes increment revision, even across horizontally scaled instances. Cache operations are time-bounded and PostgreSQL reads remain available when Redis is down.

## PR #2 Checkout foundational consistency (pre-PR #3)

Every Address mutating route acquires a PostgreSQL row lock on its owning User (the shared serialization point even before a first Address exists), then performs the primary-address reassignment or last-address guard in the same transaction. A first Address is primary; choosing a new primary demotes the old; deleting a primary promotes another; trying to remove the sole primary without choosing another returns 409. This is an HTTP data-integrity invariant, not yet a new database constraint.

Inventory PATCH validates both stock quantities together under a PostgreSQL \`FOR UPDATE\` lock on the target inventory row. Negative or over-reserved inventory cannot be written via the API. When implementing Checkout/Orders in PR #3, keep these invariants inside the reservation transaction and lock the same row.

\`LogoutAll\` verifies the supplied refresh JWT against its current UserToken row and user state, then locks the authorizing token record and deletes the user's sessions atomically. Replay of a previously deleted token is unauthorized and cannot act on other sessions.
