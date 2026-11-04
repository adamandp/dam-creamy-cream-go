# Migration parity tracker

Source: `adamandp/dam-creamy-cream-be` / `master`, Nov 4, 2026.

**Status definitions:** `PORT + TESTS` means the Go routes are implemented and selected API/database integration tests pass; it does **not** establish exhaustive 1:1 parity. `PENDING` means the module is not ported yet. Security corrections and intentional behavior changes are tracked explicitly.

| NestJS module | Go module | Status | Notes |
|---|---|---|---|
| Roles | roles | PORT + TESTS | CRUD and pagination. |
| Users | users | PORT + TESTS | Profile, password and MinIO image; selected DTO rules tested, exhaustive error parity pending. |
| Session | sessions | PORT + TESTS | JWT refresh, token persistence; security corrections must be reviewed. |
| Auth | auth | PORT + TESTS | Existing dummy login only, logout & logout-all. |
| Addresses | addresses | PORT + TESTS | Primary address transaction; address update requires behavior comparison. |
| Categories | categories | PORT + TESTS | Redis caching, 1h TTL; cache invalidation unchanged. |
| Products | products | PORT + TESTS | CRUD, catalog aggregation, MinIO uploads. |
| Product (singular) | products | PORT + TESTS | `/product/:id`, `/product/information/:id` preserved. |
| Carts | carts | PORT + TESTS | Add/remove transaction, discounted read. |
| Inventory | inventory | PORT + TESTS | Query and update routes. |
| Health | health | PORT + TESTS | Ping google.com and DB. |
| Discounts | discounts | PORT + TESTS | CRUD, DTO validation, date/null rules, no-change updates, and physical FK behavior verified in PostgreSQL tests. |
| Product Discounts | productdiscounts | PORT + TESTS | Assign/lookup/list, unique product rule and soft-disable verified. |
| Vouchers | vouchers | PORT + TESTS | CRUD, validation, nullable fields and soft-disable verified; checkout redemption remains coupled to pending Orders. |
| User Vouchers | uservouchers | PORT + TESTS | Claim/read/mark USED and nested voucher verified; /user routing defect intentionally corrected. |
| Orders | orders | PENDING | CQRS checkout handler + 17 injected steps, plus query handlers. No exposed Nest controller route on master. |
| Payments | payments | PENDING | Verify existing service behavior. |
| Shipments | shipments | PENDING | Shipping cost external integration must be traced/ported. |
| Reviews | reviews | PENDING | User/product reviews. |
| Recommendations | recommendations | PENDING | Classic, category, offers, related. |
| Product Analytics | productanalytics | PENDING | Product views/counts. |
| Notifications | notifications | PENDING | CRUD. |
| User Notifications | usernotifications | PENDING | User-specific feed/read/counts. |

## Non-negotiable before cutting over

- Build and test using Go 1.26+ with actual third-party dependencies downloaded.
- Compare PostgreSQL table definitions, enum physical types, nullable columns, defaults and UUID values with Prisma migrations.
- Run all current NestJS endpoints against seeded disposable DB and replay same requests against Go.
- Check exact status codes, message strings, field omission vs `null`, pagination and date serialization.
- Test auth across NestJS/Go, JWT existing cookies and refresh token lifecycle.
- Test Redis cache cold/hit/errors and MinIO upload/read/delete/cleanup against a disposable MinIO bucket.
- Test concurrent cart add/remove and complete checkout transaction after orders module is ported.

## Known differences in the first-pass port

- Current-user cookie is JWT-verified instead of decoded without signature verification.
- Refresh endpoint issues an access JWT using access secret, correcting a likely NestJS mistake.
- Body validation is hand-written rather than a complete port of each Zod schema; error messages and nullable-field handling are not guaranteed identical.
- Cloudinary is replaced with MinIO per request. The public image URLs now point to the Go /media route; existing Cloudinary URLs remain stored and untouched. MinIO upload/read/delete and image HTTP/database integration tests now run in CI against disposable services.
- GORM model relationship tags for association preloading are not yet exhaustive; query implementations use explicit joins where required.
- Role pagination keeps the original `Category retrieved` response message from `roles.service.ts`, even though it looks incorrect.

No unimplemented business logic is represented as complete.


## PR #1 regression and real-service tests (November 4, 2026)

- Real PostgreSQL schema generated from the original five Prisma migrations, verifying all 23 GORM table and column mappings, including case-sensitive browserInfo, expiredAt and finalPrice.
- Backend routes covered by integration tests include Auth, Sessions, Users, Categories, Products, Product Detail/Information, Carts, Addresses, Inventory and Health.
- Private MinIO image upload/read/delete as well as Users/Products image routes tested with real PostgreSQL and MinIO.
- Redis category cache tested for cold reads, cache hits and one-hour TTL.
- Intentional security fixes: verify signed JWTs on current-user and refresh endpoints; issue access JWT with the access secret.
- Prisma enum for gender is MALE/FEMALE/OTHER, while NestJS Zod allows MAN/WOMAN/OTHER. The Go port accepts MAN/WOMAN as in the NestJS DTO and writes the correct enum without changing the database.
- Remaining unported domains and full server-to-server replay remain necessary before production cutover. CI success alone does not establish parity for unported modules.

## PR #2 — Promotions: explicit NestJS/Go parity result (2026-11-04)

Replayed 19 HTTP requests against **two separately seeded disposable databases** using running NestJS and Go servers (11 GET and 8 mutations).

- **18/19** had identical HTTP statuses and JSON responses.
- **1/19 deliberate correction:** `GET /user-vouchers/user` is declared after `GET /user-vouchers/:id` in the NestJS controller. Its Express router treats `user` as the UUID parameter and responds `400 Validation failed (uuid is expected)`. The Go route is registered *before* `/:id`, returning `200` and the authenticated user's voucher assignments as the endpoint was intended to do.
- This is an **intentional bug fix**, not evidence of strict 19/19 equality. Do not change the Go route back to a broken 400 response solely to obtain byte-for-byte parity.
- The test records the route exception explicitly and fails on any *other* mismatch. Reproduce with `tests/e2e/run_compare.sh`; see `tests/e2e/README.md`.
- CI checks Go module integration using disposable PostgreSQL, Redis and MinIO. The side-by-side NestJS comparison is run separately because it requires a checkout of the original NestJS repository.
- Full end-to-end business parity for unported modules, including checkout, is deferred until those modules exist in Go.

**Ported in PR #2:** Discounts (CRUD/hard delete), Product Discounts (assign/lookup/soft-disable), Vouchers (CRUD/soft-disable), User Vouchers (claim/query/mark USED). All existing SQL tables are reused; no application migrations run.
