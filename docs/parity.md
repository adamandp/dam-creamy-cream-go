# Migration parity tracker

Source: `adamandp/dam-creamy-cream-be` / `master`, Nov 4, 2026.

**Status definitions:** `FIRST PASS` means a Go implementation exists but regression tests against the NestJS implementation and a disposable copy of the original database have not been completed. `PENDING` means no Go endpoint or module-equivalent logic is shipped; do not treat as supported.

| NestJS module | Go module | Status | Notes |
|---|---|---|---|
| Roles | roles | FIRST PASS | CRUD and pagination. |
| Users | users | FIRST PASS | Profile, password and Cloudinary image; field-level DTO validation needs parity tests. |
| Session | sessions | FIRST PASS | JWT refresh, token persistence; security corrections must be reviewed. |
| Auth | auth | FIRST PASS | Existing dummy login only, logout & logout-all. |
| Addresses | addresses | FIRST PASS | Primary address transaction; address update requires behavior comparison. |
| Categories | categories | FIRST PASS | Redis caching, 1h TTL; cache invalidation unchanged. |
| Products | products | FIRST PASS | CRUD, catalog aggregation, Cloudinary uploads. |
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
- Test Redis cache cold/hit/errors and Cloudinary upload/destroy via a non-production cloud account.
- Test concurrent cart add/remove and complete checkout transaction after orders module is ported.

## Known differences in the first-pass port

- Current-user cookie is JWT-verified instead of decoded without signature verification.
- Refresh endpoint issues an access JWT using access secret, correcting a likely NestJS mistake.
- Body validation is hand-written rather than a complete port of each Zod schema; error messages and nullable-field handling are not guaranteed identical.
- Cloudinary uploads use signed Cloudinary REST requests rather than the Node SDK; real integration test is pending.
- GORM model relationship tags for association preloading are not yet exhaustive; query implementations use explicit joins where required.
- Role pagination keeps the original `Category retrieved` response message from `roles.service.ts`, even though it looks incorrect.

No unimplemented business logic is represented as complete.
