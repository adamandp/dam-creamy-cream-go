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
- PR #1 DTO validation is hand-written and not exhaustively verified. PR #2 Promotions DTOs now implement Zod-shaped validation errors on covered cases, including nullability and field-specific paths.
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

## PR #2 — Promotions: approved Go/NestJS compatibility decisions (2026-11-04)

The side-by-side harness runs the original NestJS and Go services with separately seeded disposable PostgreSQL databases created from the original five Prisma migrations. See tests/e2e/README.md.

- **Original 19 HTTP routes/requests: 15 exact matches and 4 documented improvements.** NestJS source routes and responses are treated as the baseline, except where an explicit Go bug/security correction was approved.
- Four baseline corrections: (1) GET /user-vouchers/user returns real data (the source mistakenly routes /user through /:id and returns 400); (2) GET /user-vouchers and (3) GET /user-vouchers/voucher/:id reject non-admins with 403; and (4) GET /product-discounts counts actual ProductDiscount rows rather than all User rows.
- Additional compared cases cover customer ownership, forbidden status transitions, anonymous access, body-vs-param UUID validation, fractional Prisma Int columns, Zod numeric/date coercion, exact Zod error structure on tested inputs, original camelCase update error keys, safe foreign-key status 409, and malformed Product Discount path parameters returning 400 rather than internal 500.
- After writes, a comparator normalizes naturally generated UUIDs/subsecond timings while requiring every stable field to match and timestamps on both servers to agree within 10 seconds. Go now writes UTC through GORM NowFunc and explicit session/cart clock assignments to avoid a 7-hour offset in the existing PostgreSQL TIMESTAMP(3) schema.
- All user-voucher routes now require verified JWT bearer tokens and an active user/role in the current database. Customers can claim for themselves and read their own claims; all-user listings and USED transitions are admin-only. This is an intentional improvement over NestJS legacy endpoints.
- GitHub Actions validates Go unit and real PostgreSQL/Redis/MinIO integration tests. NestJS-vs-Go comparison requires a separate NestJS checkout and is reproducible locally using the runner.
- **No automatic migration, schema change, or production DB data modification.**

**PR #2 is the Promotions feature batch**, not full application parity. Orders/checkout, Payments, Shipments, Reviews, Recommendations, Analytics and Notifications remain unported. Uncovered edge cases may exist; treat CI success as evidence for tested behavior only.
