# Creamy Cream Go — HTTP API

All routes are relative to the backend root. List endpoints accept page (default 1) and limit (default 10). JSON responses use message and optional data/paging.

## Core modules

| Route group | Features |
|---|---|
| /auth | Deliberate dummy customer login for testing (unchanged), logout and logout-all |
| /sessions | Access-token refresh |
| /roles | Public reads; admin-only create, update, delete |
| /users | Admin-only list; detail self/admin; profile requires active bearer; own safe profile edits; password ONLY through PUT with current password; image self/admin; role/status mutation and delete admin-only |
| /addresses | Create self/admin; global list admin-only; detail/update/delete owner/admin; at least one address and exactly one primary per user enforced by transactions; /user reads active refresh cookie; ownership transfer prohibited |
| /categories | Public reads and Redis cache; create/update/delete admin-only |
| /products | Public catalog/detail; create/update/delete/product-image changes admin-only |
| /product | Detail and product information |
| /carts | Current-user cart read/add/remove via refresh cookie verified against active PostgreSQL session |
| /inventory | Public read/lookup; admin-only PATCH with nonnegative quantity/reservedQuantity and reservedQuantity <= quantity, validated under a PostgreSQL row lock |
| /orders | POST /checkout owner-authenticated; GET /orders admin-only; GET /user owner-only; GET /:id owner/admin |
| /products/catalog, /product/:id | Public product pricing; applies discounts only when the product-discount assignment is active **and** the discount is within its inclusive start/end period |
| /health | HTTP and PostgreSQL status |
| /media/:scope/:id/:filename | Public image delivery from private storage |

## Discounts

| Method | Path | Behavior |
|---|---|---|
| POST | /discounts | Admin only: create discount |
| GET | /discounts | Paginated list |
| GET | /discounts/:id | Get discount |
| PATCH | /discounts/:id | Admin only: update discount |
| DELETE | /discounts/:id | Admin only: hard delete |

Discount types: PERCENTAGE, FIXED, PRICE, FREE_ITEM. Nullable discount values, descriptions, product references and quantities must be present on creation. Current list response labels use User and Users for the list and empty error respectively.

## Product discounts

| Method | Path | Behavior |
|---|---|---|
| POST | /product-discounts | Admin only: assign discount to product |
| GET | /product-discounts | Paginated list |
| GET | /product-discounts/:id | Get assignment |
| GET | /product-discounts/product/:productId | List by product |
| DELETE | /product-discounts/:id | Admin only: set isActive=false |

Product discount assignments have one row per product (unique product ID). Disabled assignments remain queryable. The list totalItems counts product-discount rows (fixing the NestJS bug that counted users).

## Vouchers

| Method | Path | Behavior |
|---|---|---|
| POST | /vouchers | Admin only: create voucher |
| GET | /vouchers | Paginated list |
| GET | /vouchers/:id | Get voucher |
| PATCH | /vouchers/:id | Admin only: update voucher |
| DELETE | /vouchers/:id | Admin only: set isActive=false |

Voucher types: PERCENTAGE, FIXED and FREE_ITEM. Groups: CHECKOUT and SHIPPING. Soft-disabled vouchers remain readable.

## User vouchers

| Method | Path | Behavior |
|---|---|---|
| POST | /user-vouchers | Bearer JWT; customer self-claim or admin assignment; server validates availability and capacity |
| GET | /user-vouchers | Bearer JWT; admin only, paginated |
| GET | /user-vouchers/user | Bearer JWT; current user's own claims |
| GET | /user-vouchers/voucher/:id | Bearer JWT; admin only |
| GET | /user-vouchers/:id | Bearer JWT; owner or admin only |
| PATCH | /user-vouchers/:id | Bearer JWT; admin only, marks USED |

Claims start as CLAIMED. To claim, send voucherId (userId is optional for a customer and defaults to their authenticated identity; admin may provide userId). Legacy claimedAt and expiredAt inputs may be supplied for compatibility but are **ignored**: claimedAt is the server's UTC clock; expiredAt comes from Voucher.endDate. Voucher must be active, within its start/end time window, and have capacity (maxRedeemed > 0 and max of existing claim rows and redeemedCount below maxRedeemed). Each user can claim a voucher only once, including after USED; duplicates and ineligible vouchers return HTTP 409. Voucher-level PostgreSQL FOR UPDATE locking serializes simultaneous claims without changing the existing schema.

**Discount pricing:** Cart, Catalog, Product Detail, and Checkout share the same active-assignment, inclusive date checks, and global redemption-quota eligibility (maxRedeemed > 0 and redeemedCount < maxRedeemed). PERCENTAGE subtracts a percentage; FIXED subtracts a fixed amount; PRICE sets the final paid price (bounded by the original price, with zero allowed); FREE_ITEM adds a separate bonus order line instead of lowering the original item price. Nil/zero for FIXED or PERCENTAGE produces null discountPrice; FREE_ITEM always produces null discountPrice. All paid prices have a zero floor and round to cents. If global discount quota is exhausted, Cart/Catalog/Detail return discountPrice: null even while ProductDiscount assignment and dates remain valid. A storefront read is only a snapshot: Checkout still locks the discount and checks actual requested quantity and per-order cap. This is a deliberate correction of inherited behavior that applied disabled discounts to Cart and expired discounts to Catalog/Detail.

**Category cache:** successful create/update/delete increments a shared Redis revision key (categories:revision); paginated list keys include that revision and expire after one hour. Old versions naturally expire. Redis failures do not block PostgreSQL-backed API reads or report an already-committed write as failed; in the exceptional case of an outage during a mutation, cached data might remain stale until its TTL after Redis recovers.

**Refresh JWT:** a random 128-bit base64url jti differentiates rapid or concurrent logins. The compact refresh JWT deliberately omits optional iat and typ fields to fit the existing Prisma user_tokens.token VARCHAR(255) column, while retaining the mandatory exp, HS256 signature and active database session checks. Access JWT format is unchanged. Dummy login credentials and behavior are unchanged.

The /user endpoint returns each claim with its associated voucher under the vouchers property. The admin use endpoint marks USED; checkout voucher redemption is ported in PR #3; payment settlement is not ported. Access JWTs are signature-verified and checked against the CURRENT user, active flag and role in PostgreSQL. Admin/Administrator/Super Admin/Superadmin role names have elevated access (case-insensitive); customer access is owner-scoped. Read access to Discounts, Product Discounts and Vouchers is public; all their write operations require a verified current admin in PostgreSQL.

Role creation/renaming/deletion is admin-only. Customers can edit their own safe profile fields, but cannot set roleId or isActive or edit another user's account. Password changes require the authenticated account owner **and the existing password** through PUT /users/:id/password; PATCH /users/:id rejects password changes for both user and admin. User directory GET /users requires admin, and GET /users/:id is self/admin only. User deletion is admin-only. Role changes invalidate access tokens with the old role claim. Admin accounts must already exist in the database or be provisioned out-of-band; there is no public bootstrap-admin endpoint.

Categories and Products mutations and product-image changes are admin-only; Inventory PATCH is admin-only. Their public read routes are preserved for storefront browsing. Address creation requires a current bearer-authenticated actor and matching userId unless admin; listing all addresses is admin-only, and by-id reads/updates/deletions require owner/admin. Address userId reassignment through PATCH is forbidden (including by admin); the existing at-least-one-address deletion rule remains. GET /addresses/user and all Cart /user endpoints keep the existing refresh-cookie interface but verify signature **and** current user/role/active account **and** the backing unexpired UserToken row; logout/logout-all immediately revokes these routes, even while the JWT signature remains valid. /sessions/refresh uses this same validation.

**Checkout foundation invariants (PR #2):** Address create/update/delete serialize on the owning User row (\`FOR UPDATE\`) so simultaneous requests cannot establish two primary addresses. The first address is automatically primary; selecting another demotes the previous primary atomically; attempting to unset the only primary returns 409. When a primary address is deleted, another existing address becomes primary; deleting the user's final address returns 409. New address creates also repair any already-invalid zero/multiple-primary state left by the earlier code, without a database migration. The \`isPrimary\` switch is a deliberate correction to legacy Nest behavior.

Inventory PATCH accepts only \`quantity >= 0\`, \`reservedQuantity >= 0\`, and \`reservedQuantity <= quantity\`. Negative inputs return 400; inconsistent available/reserved state returns 409. Updates lock the Inventory row in a PostgreSQL transaction to prevent concurrent partial updates violating these constraints. PR #3 Checkout stock reservations use the same row-locking invariant.

\`POST /auth/logout-all\` requires a **currently active refresh session**, including a matching unexpired UserToken database row, not merely a valid JWT signature. Revoked/replayed tokens return 401 and cannot delete sessions belonging to the same user. Authorized logout-all still revokes all active sessions atomically. The dedicated dummy login is unchanged.

The deliberate dummy-customer-login endpoint and its known test account credentials are **unchanged by this security hardening**. It remains a development/testing feature and must not be exposed as a real customer authentication flow in production.

## General contracts

- Creation returns 201; read/update/delete returns 200 on success.
- Invalid DTO/parameter errors return 400; promotion DTOs provide source-compatible Zod errors[]; missing rows return 404; unique and foreign-key constraints return 409.
- No-op updates are rejected on endpoints with an update guard.
- Datetime fields are serialized as UTC ISO timestamps with milliseconds. The GORM clock writes UTC even on non-UTC hosts, matching Prisma TIMESTAMP(3) columns.
- Zod-style pagination accepts numeric notations such as page=1.0 and limit=1e1. Noncanonical URL UUIDs are rejected with 400.
- Promotion DTO numeric fields accept fractions as source Zod does. Prisma-backed Int columns store the integer portion.
- User-voucher ACL, product-discount counts and semantic database errors are approved legacy corrections.

### PR #3: Orders / Checkout

POST /orders/checkout exposes an explicit authenticated API contract for the existing NestJS 17-step CQRS checkout handler (its OrdersController had no HTTP route). One GORM/PostgreSQL transaction coordinates Cart, Inventory, Discounts, Vouchers and UserVoucher claims and inserts Order/OrderItems, Payment PENDING and Shipment PENDING. New order list/detail reads are owner/admin scoped. See [full request shape, safeguards and limitations](orders-checkout.md). Payment processing and trusted courier pricing are **not** implemented; the caller provides a validated but otherwise unverified shipping cost.


### PR #4: Shipments / Payments

- GET /shipments?page=1&limit=10: current admin access JWT, paginated Shipment rows, 404 if empty (NestJS behavior).
- GET /shipments/:id: owner or admin access JWT. Invalid UUID 400, missing 404, foreign owner 403.
- PATCH /shipments/:id/status: admin access JWT, body {"status":"SHIPPED"}, enum PENDING/SHIPPED/DELIVERED/CANCELLED; changes only Shipment status.
- POST /shipments/cost: HTTP 201, public body {"origin":123,"destination":456,"weight":1000}; sends a server-only key to RajaOngkir; returns {message,status,data}. Missing config 503, upstream failure 502. Does not validate Checkout shipping fees.
- Payments POST/GET/PATCH/DELETE endpoints reproduce NestJS scaffold literal text only; there is no payment mutation. See [PR #4 contract](shipments-payments.md).
## PR #5: Reviews, Recommendations, Analytics and Notifications

The original NestJS routes are now registered in the stacked Go branch:

| Domain | Routes |
|---|---|
| Reviews | `POST /reviews`, `GET /reviews`, `GET /reviews/product/:id`, `GET /reviews/:id`, `GET /reviews/me`, `PATCH /reviews/:id`, `DELETE /reviews/:id` |
| Recommendations | `GET /recommendations/classic`, `GET /recommendations/categories`, `GET /recommendations/offers`, `GET /recommendations/related/:id` |
| Product Analytics | `POST /product-analytics`, `GET /product-analytics/product/:id`, `GET /product-analytics/user`, `GET /product-analytics/product/:id/count` |
| Notifications | `POST /notifications`, `GET /notifications`, `GET /notifications/:id`, `PATCH /notifications/:id`, `DELETE /notifications/:id` |
| User Notifications | `POST /user-notifications`, `GET /user-notifications`, `GET /user-notifications/:id`, `GET /user-notifications/me`, `GET /user-notifications/count/me`, `GET /user-notifications/notification/:id`, `GET /user-notifications/notification/:id/count`, `PATCH /user-notifications/:id` |

The public read surfaces remain available for Reviews, Recommendations,
Product Analytics by product, and Notification definitions. The correctness
follow-up introduces these explicit verified-bearer authorization rules:

- Reviews: authenticated users may create their own review; editing/deleting
  requires its owner or an admin. `GET /reviews/me` requires an actor.
- Product Analytics: creation requires an actor, and the submitted `userId`
  must match the actor unless admin. `userId` and `productId` must be real UUIDs,
  not null/missing. `GET /product-analytics/user` requires an actor.
- Notification definitions: mutations are admin-only, reads remain public.
  Titles are capped at 50 characters to match PostgreSQL `VARCHAR(50)`.
- User Notifications: creating and global/notification-wide listing/counts are
  admin-only; reading an individual record or marking it read requires its
  owner or an admin. The `me` feed/count require an actor.

Review and Product Analytics pagination totals now count the filtered rows;
notification-wide filters use `notification_id`, not the original incorrect
`user_id` predicate. See `parity.md` for the intentionally changed NestJS
behavior. An absent collection returns 404 on the corresponding paginated list.
