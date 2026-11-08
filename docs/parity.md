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
| Orders | orders | PR #3 PORT + TESTS | Explicit Go HTTP checkout/list/detail surface, atomic Order/OrderItem/Payment/Shipment writes, stock locks, cart decrement, discount and voucher validation; original Nest has CQRS steps but no HTTP Orders route. |
| Payments | payments | PR #4 SCAFFOLD PARITY + TESTS | NestJS PaymentService only returns literal strings; matching endpoints never charge or mutate payment records. |
| Shipments | shipments | PR #4 PORT + TESTS | List/detail, status PATCH, public cost proxy via RajaOngkir with Redis per-IP and global provider quotas, explicit trusted proxies, positive integer provider inputs, and Prisma millisecond UTC timestamps; owner/admin ACL deliberately fixes unauthenticated legacy reads/writes. External carrier API needs real-credential verification. |
| Reviews | reviews | PR #5 PORT + TESTS | Create/list/detail/product/me/update/delete, including source global-count pagination. |
| Recommendations | recommendations | PR #5 PORT + TESTS | Classic, categories, offers, related; original ranking/discount formulas. |
| Product Analytics | productanalytics | PR #5 PORT + TESTS | Create, product/user views and count; original global-count pagination. |
| Notifications | notifications | PR #5 PORT + TESTS | CRUD and existing DTO constraints. |
| User Notifications | usernotifications | PR #5 PORT + TESTS | CRUD subset, feed, read and counts with nested Notification JSON. |

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

## PR #5 — Remaining five NestJS domains

All 28 remaining NestJS controller routes are registered in the Go branch:
Reviews (7), Recommendations (4), Product Analytics (4), Notifications (5),
and User Notifications (8). Each module ports the existing service's query,
response and mutation semantics; this **does not certify whole-app 1:1 parity**.

The source has behaviors that appear defective but were deliberately retained
for the requested source-first rewrite:

- `ReviewsService.findByUser` and `.findByProduct` use a **global** Review count
  in pagination, even though rows are scoped.
- Product Analytics pages also use a **global** count.
- `UserNotificationsService.findByNotification` and `.findByNotificationCount`
  filter `user_id` with the **notification ID**, not `notification_id`.
- User notification count includes read records; it is not an unread count.
- Recommendations deliberately use their own historical discount computation
  rather than the corrected Catalog/Checkout shared price calculation. The
  offers percentage-ranking expression uses `discount.value / 100`, and the
  source takes its first five offers **before** sorting them.
- Nullable Product Analytics IDs still use the source's misspelled `unkown`
  fallback, which is invalid for the UUID database columns. No schema change
  or silent rewrite was introduced.
- Nest's Notification Zod title length allows 100 characters although the
  underlying Prisma VARCHAR(50) column is narrower.

All five modules expose the source's public mutation routes as-is. The `me`
endpoints use the existing Go verified-current-actor middleware rather than
Nest's signature-free JWT decode; this is an inherited, documented security
deviation. A separate authorization design change should not be mixed with
this compatibility port.

Integration tests exercise all five domains against the existing disposable
PostgreSQL schema. They also pin source-specific counts, nesting and filters.
The end-to-end harness now includes all 28 routes from these five domains with
deterministic seeded data and a separate NestJS server in the CI `nest-parity`
job, in addition to the original Promotions replay. These are selected route
cases, **not** an exhaustive full-application business-parity proof. Before
cutover, checkout/auth/stock edge cases and real external providers still
require independent verification.


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


### Security and voucher eligibility hardening on PR #2 (2026-11-05)

- Code from PR #1 was fixed directly on the PR #2 branch, **without modifying or merging PR #1**. Shared \`sessions.Actor\` middleware verifies signed bearer access tokens against the current, active database user and actual role on every authorized route.
- Roles: creating, renaming and deleting roles requires Admin. Users: self-service edit is limited to non-privileged profile fields; only Admin may change roleId or isActive; password change is owner-only, image owner/Admin, deletion Admin-only. Customer role and user privilege escalation via anonymous or self-service APIs are now blocked.
- Discounts/Vouchers/Product Discounts: reads remain public; **all writes are admin-only**. User Vouchers use the same shared authorization, including self-only claims and admin global operations.
- Claim voucher: server validates active status, start/end dates, prior claims, user active state and capacity; claims acquire PostgreSQL voucher-row locks to prevent concurrent over-claim. The conservative capacity count uses max(redeemedCount, number of existing claims) against maxRedeemed and rejects zero-capacity vouchers. No schema migration is required. Client-supplied claimedAt/expiredAt are ignored; server writes UTC now and voucher endDate.
- Source-parity acceptance was updated to admin credentials for promotions writes and fresh claim vouchers. The 19 original baseline requests retain 15 identical + 4 documented differences; additional checks now include anonymous role writes/promo writes and server-owned claim timestamps. Latest side-by-side result: **34 matches, 13 documented differences, 0 unexpected**. Concurrency, disabled/expired/too-early/exhausted, duplicate and minimal-body cases are in the PostgreSQL integration tests.
- Auth and validation changes above are explicitly approved legacy corrections and **not** claims of exact byte-for-byte parity. Checkout/redemption transaction rules remain for a later stacked PR.

### PR #2 follow-up — inherited catalog, address, account and session authorization (2026-11-05)

- Applied directly to the existing PR #2 branch, including original PR #1 modules. Neither PR #1 nor \`main\` is modified.
- Categories/Products mutations including product images, and Inventory updates: verified current Admin JWT required (401 when anonymous; 403 for ordinary customer). Public catalog/inventory/category GET routes remain available to storefront clients.
- User directory is Admin-only; per-user details are accessible only to account owner/Admin. Existing Users profile endpoint still requires verified signed access JWT. Generic Users PATCH rejects \`password\` so password changes require the dedicated PUT action, an authenticated owner and the original password. Roles/users protections from the previous hardening remain intact.
- Addresses: create requires self userId or admin; all-record listing is admin-only; detail/update/delete enforce ownership/Admin; changing userId through PATCH is always forbidden. Existing /addresses/user cookie route remains, with session-backed token verification.
- The shared refresh-cookie validation is now \`sessions.ActiveRefresh\`: HS256 signature/expiration, **live matching UserToken row and DB expiration**, current active user and matching role ID. Carts and GET /addresses/user now reject revoked tokens immediately after logout/logout-all. Session refresh uses the same method. Access JWTs still have their existing 15-minute expiration.
- **Dummy customer login remains deliberately unchanged** at the user's request; it is a development/testing feature, not a production login flow.
- Regression test \`TestInheritedRouteAuthorizationOnPostgres\` covers anonymous/cross-account writes and reads, role access, catalog/inventory admin writes, password PATCH bypass rejection, legitimate owner/admin operations and revoked-cookie failures. Original Prisma schema and all PostgreSQL/Redis/MinIO integration suites remain in use; no schema migration.
- These are deliberate security corrections to inherited NestJS behavior; not byte-for-byte response parity changes.


### PR #2 follow-up — unified prices, unique refresh JWTs and category cache revisions (2026-11-05)

- **Cart vs Catalog/Detail prices:** corrected two inherited inconsistencies. Cart no longer applies an inactive ProductDiscount assignment, and Catalog/Detail no longer apply discounts before startDate or after endDate. All three share \`internal/modules/pricing\` for eligibility SQL and discount arithmetic. Nil/zero discounts do not apply; calculated prices are floored at zero. The existing per-product discount table and database schema are unchanged.
- **Duplicate refresh JWTs:** issuing two JWTs for one user within the same second formerly generated identical signed strings (second-precision iat/exp), so the second UserToken insert violated the token unique constraint. Refresh JWT now includes an unpredictable 128-bit jti. The original UserToken.token database column is VARCHAR(255), so the refresh JWT omits *optional* iat and typ fields to remain within 255 characters. Access-token iat/exp structure, refresh-token expiration and HS256 verification remain in place. Dummy customer login is intentionally untouched.
- **Category Redis consistency:** create/update/delete increments a shared persistent categories:revision key after successful PostgreSQL mutation. Paginated cache keys include the version, so reads from multiple horizontally scaled API instances cannot serve an older cached page after mutation completes. Old keys retain one-hour TTL. If Redis is unhealthy, short cache timeouts allow PostgreSQL fallback; rare Redis outages during writes can leave stale cache versions until expiry.
- **Real-service regression tests:** \`TestCartCatalogDetailDiscountEligibility\` covers disabled/expired/future/active discounts and shared Cart/Catalog/Detail calculations; \`TestRefreshJWTUniqueJTIWithinSecond\` creates 12 unique, valid, <=255-character sessions in rapid succession; \`TestCategoryCacheRevisionAfterAllMutations\` checks create/update/delete, 2+ paginated pages and multi-instance Redis invalidation. Prior PostgreSQL/Redis/MinIO integration suites and race checks remain required. Original NestJS/Go promotions HTTP replay remains 34 matches, 13 accepted changes, 0 unexpected.
- All changes are stacked on **PR #2**, with no modification to PR #1, main, source NestJS or database schema. This is scoped correctness work, not an assertion of complete production readiness.

### PR #2 follow-up — Checkout foundation invariants (2026-11-05)

- **Addresses:** create, update and delete serialize by acquiring \`SELECT ... FOR UPDATE\` on the owner User row. A user cannot have two primary addresses via HTTP races; the first address is primary regardless of the supplied \`isPrimary\`; switching primary demotes the previous primary within the transaction. Directly unsetting the current primary returns 409, deleting the primary automatically promotes a remaining address, and deleting the last address still returns 409. A new address create repairs old zero/multiple-primary states. No schema or migration changes.
- **Inventory:** admin PATCH rejects negative quantity or reservedQuantity (400), and rejects a reservedQuantity larger than quantity (409). A PostgreSQL inventory-row lock serializes concurrent updates so two individually valid partial requests cannot combine into an invalid state. Future Orders/Checkout reservation code should reuse this constraint and lock.
- **Sessions:** \`POST /auth/logout-all\` now requires a live, matching and unexpired UserToken plus valid signed refresh JWT and current active user/role; the authorizing UserToken row is locked before all sessions are deleted. A signed but already revoked token gets 401 and cannot delete other active sessions. Dummy login is deliberately untouched.
- **Regressions:** \`TestAddressPrimaryInvariantAndReplacement\` covers primary switch, delete promotion, last-address guard, parallel switches and legacy-state repair; \`TestInventoryValidationsAndConcurrentEdits\` covers negatives, capacity conflicts, and concurrent partial updates; \`TestRevokedRefreshCannotLogoutAll\` verifies denied replay and valid logout-all via real HTTP. PostgreSQL/Redis/MinIO integration, repeated race tests and NestJS-vs-Go promotions replay remain required. Corrections above deliberately differ from insecure/inconsistent source behavior.
- **Branch isolation:** all code lives in PR #2 only. PR #1, main, NestJS source, dummy login and database schema are unchanged.

### PR #3 — Orders and Checkout

Ported the existing NestJS Orders CQRS semantics to Go without inventing a NestJS route. Created owner-authenticated POST /orders/checkout and scoped reads. Server-owned prices, address ownership, Cart quantity, active product-discount association, per-discount and voucher quotas, CLAIMED voucher status, free-gift stock, and shipping/checkout voucher arithmetic are verified inside one DB transaction. Payment and Shipment are created PENDING, not sent to external gateways. Real PostgreSQL tests cover rollback at the final Shipment insert, concurrent stock and voucher contention, free reward stock and ownership. See [Orders/Checkout contract](orders-checkout.md). The previous 34-matches/13-approved/0-unexpected Promotions replay remains required and unchanged.

### PR #3 follow-up — shared PRICE and FREE_ITEM semantics (2026-11-05)

Corrected a previously untested mismatch between PR #2 Cart/Catalog/Detail and PR #3 Checkout. Discount type PRICE is a server-side *final price override*, not a fixed amount to deduct. FREE_ITEM never changes the paid line price and is materialized as a free reward with inventory reservation at Checkout. All three price-changing types use the shared calculator and consistent cent rounding. Real PostgreSQL integration regression TestAllDiscountTypesMatchCheckout compares Cart, Catalog, Detail, saved Checkout amount and reward inventory across nine scenarios; unit test TestDiscountPriceSemantics covers the same boundary logic. No database migration, dummy-login change, or change to parent PR branches.


### PR #3 follow-up — exhausted discount storefront eligibility (2026-11-05)

Public Cart/Catalog/Detail price queries now exclude discounts whose global redeemedCount >= maxRedeemed or maxRedeemed <= 0, matching the Checkout eligibility guard. Previously, storefronts advertised an exhausted promo which Checkout correctly rejected. The underlying Checkout row-lock, per-order cap and requested-quantity quota check remain authoritative; this is a deliberate inherited pricing correctness fix rather than a guaranteed quote reservation. Tests: TestCartCatalogDetailDiscountEligibility (available/exhausted/over/zero/reopened quota) and TestQuotaExhaustionAcrossStorefrontAndCheckout (all three reads, 409 without Order persistence, successful final-redemption checkout and subsequent suppression). No source NestJS, schema or dummy-login change.

### PR #3 source parity and hardening follow-up (2026-11-07)

- Checkout quantity and order-line validation now preserves the absence of arbitrary upper limits in the NestJS Zod DTO while retaining mandatory positive integer quantities and inventory/Cart/discount protection.
- Shipment estimated-date decoding accepts the source's Zod coercion variants, reusing the existing promotion date parser.
- Fixed a no-voucher fractional-shipping calculation where a raw shipping cost was incorrectly subtracted from a cent-rounded one, producing a negative voucher amount.
- Credentialed CORS no longer reflects arbitrary Origin headers; production requires a configured `FRONTEND_ORIGINS` allowlist. This deliberately hardens the inherited NestJS `origin: true` behavior.
- Request-bound operations use request contexts; detached compensating cleanup stays independent. Pricing unit tests are now under `tests/unit`.
- These are scoped corrections in PR #3 only; the previous 34 matches / 13 documented deviations are not a substitute for complete NestJS regression testing.


### PR #4: Shipments and legacy Payments endpoints (2026-11-07)

- Shipment list is admin-only, detail owner/admin, status mutation admin-only. This tightens NestJS's unauthenticated access. Mutating Shipment.status does not silently update Orders or Payments.
- Public POST /shipments/cost uses numeric origin, destination, weight and the original 17 couriers to call RajaOngkir's URL-encoded domestic-cost API from the server; it returns message, status and data. Missing provider settings yield 503, upstream errors 502. Positive finite values are an intentional stricter input rule. Fake-provider tests exist; actual provider verification requires staging credentials.
- PaymentsService in NestJS is an inert scaffold: five literal-text routes without data mutation. PR #4 reproduces it; it does NOT process capture, settlement or refunds.
- No database migration, changes to PR #1/#2/#3, or changes to the dummy login. Checkout still accepts an unverified caller-supplied shipment.cost; production quote binding is a separate concern.
- PR #4 follow-up fixes the three inherited PR #3 findings: aggregate prices reconcile from rounded unit prices, `FREE_ITEM` reward quantities follow the NestJS configuration rather than an arbitrary 100-unit cap (with inventory overflow checks), and both required shipment `cost` and nullable `description` keys are validated. Checkout quote-binding remains a separate unresolved contract/integrity issue because the schema lacks authoritative destination/weight data.
