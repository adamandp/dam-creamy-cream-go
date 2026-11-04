# Migration parity tracker

Source: `adamandp/dam-creamy-cream-be` / `master`, Nov 4, 2026.

**Status definitions:** `FIRST PASS` means a Go implementation exists but regression tests against the NestJS implementation and a disposable copy of the original database have not been completed. `PENDING` means no Go endpoint or module-equivalent logic is shipped; do not treat as supported.

| NestJS module | Go module | Status | Notes |
|---|---|---|---|
| Roles | roles | FIRST PASS | CRUD and pagination. |
| Users | users | FIRST PASS | Profile, password and MinIO image; field-level DTO validation needs parity tests. |
| Session | sessions | FIRST PASS | JWT refresh, token persistence; security corrections must be reviewed. |
| Auth | auth | FIRST PASS | Existing dummy login only, logout & logout-all. |
| Addresses | addresses | FIRST PASS | Primary address transaction; address update requires behavior comparison. |
| Categories | categories | FIRST PASS | Redis caching, 1h TTL; cache invalidation unchanged. |
| Products | products | FIRST PASS | CRUD, catalog aggregation, MinIO uploads. |
| Product (singular) | products | FIRST PASS | `/product/:id`, `/product/information/:id` preserved. |
| Carts | carts | FIRST PASS | Add/remove transaction, discounted read. |
| Inventory | inventory | FIRST PASS | Query and update routes. |
| Health | health | FIRST PASS | Ping google.com and DB. |
| Discounts | discounts | PENDING | Existing business validation must be ported. |
| Product Discounts | productdiscounts | PENDING | Existing rules, active states. |
| Vouchers | vouchers | PENDING | Validation, redemption limits, checkout semantics. |
| User Vouchers | uservouchers | PENDING | Claim/use and unique constraints. |
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
