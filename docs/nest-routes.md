# REST controller inventory in the original NestJS backend

Extracted from `src/model/*/*.controller.ts` on `master`.

| Area | Paths |
|---|---|
| Auth | `POST /auth/dummy-customer-login`, `/auth/logout`, `/auth/logout-all` |
| Sessions | `POST /sessions/refresh` |
| Roles | `POST /roles`, `GET /roles`, `GET /roles/:id`, `GET /roles/name/:name`, `PATCH /roles/:id`, `DELETE /roles/:id` |
| Users | `GET /users`, `GET /users/profile`, `GET /users/:id`, `PATCH /users/:id`, `PUT /users/:id/password`, `PUT /users/:id/image`, `DELETE /users/:id` |
| Addresses | `POST /addresses`, `GET /addresses`, `GET /addresses/user`, `GET /addresses/:id`, `PATCH /addresses/:id`, `DELETE /addresses/:id` |
| Categories | `POST /categories`, `GET /categories`, `GET /categories/:id`, `PATCH /categories/:id`, `DELETE /categories/:id` |
| Products | `POST /products`, `GET /products`, `GET /products/catalog`, `PATCH /products/:id`, `PATCH /products/:id/image`, `DELETE /products/:id` |
| Product details | `GET /product/information/:id`, `GET /product/:id` |
| Carts | `POST /carts/user/add`, `POST /carts/user/remove`, `GET /carts/user` |
| Inventory | `GET /inventory`, `GET /inventory/:id`, `GET /inventory/product/:id`, `PATCH /inventory/:id` |
| Discounts | `POST /discounts`, `GET /discounts`, `GET /discounts/:id`, `PATCH /discounts/:id`, `DELETE /discounts/:id` |
| Product discounts | `POST /product-discounts`, `GET /product-discounts`, `GET /product-discounts/:id`, `GET /product-discounts/product/:productId`, `DELETE /product-discounts/:id` |
| Vouchers | `POST /vouchers`, `GET /vouchers`, `GET /vouchers/:id`, `PATCH /vouchers/:id`, `DELETE /vouchers/:id` |
| User vouchers | `POST /user-vouchers`, `PATCH /user-vouchers/:id`, `GET /user-vouchers`, `GET /user-vouchers/:id`, `GET /user-vouchers/user`, `GET /user-vouchers/voucher/:id` |
| Reviews | `POST /reviews`, `GET /reviews`, `GET /reviews/product/:id`, `GET /reviews/:id`, `GET /reviews/me`, `PATCH /reviews/:id`, `DELETE /reviews/:id` |
| Recommendations | `GET /recommendations/classic`, `/recommendations/categories`, `/recommendations/offers`, `/recommendations/related/:id` |
| Analytics | `POST /product-analytics`, `GET /product-analytics/product/:id`, `/product-analytics/user`, `/product-analytics/product/:id/count` |
| Notifications | `POST /notifications`, `GET /notifications`, `GET /notifications/:id`, `PATCH /notifications/:id`, `DELETE /notifications/:id` |
| User notifications | `POST /user-notifications`, `GET /user-notifications`, `/user-notifications/:id`, `/user-notifications/me`, `/user-notifications/count/me`, `/user-notifications/notification/:id`, `/user-notifications/notification/:id/count`, `PATCH /user-notifications/:id` |
| Shipments | `GET /shipments`, `GET /shipments/:id`, `PATCH /shipments/:id/status`, `POST /shipments/cost` |
| Payments | `POST /payments`, `GET /payments`, `GET /payments/:id`, `PATCH /payments/:id`, `DELETE /payments/:id` |
| Orders | NestJS controller exists but declares **no HTTP routes** on `master`. Checkout is implemented as CQRS steps. |
| Health | `GET /health`, `GET /health/db` |

As of PR #2, the Go API also registers **Discounts, Product Discounts, Vouchers and User Vouchers**. Routes for other pending modules are still source inventory only, not Go endpoints.

**Known corrected source routing defect:** NestJS has `GET /user-vouchers/:id` above `GET /user-vouchers/user`, so the literal path `/user` is treated as a UUID and returns 400. Go registers `/user` before `/:id`, returning the intended user vouchers (200). This is the single recorded mismatch among 19 side-by-side HTTP test cases. See `parity.md` and `tests/e2e/compare_promotions.py`.
\n**Further approved PR #2 corrections:** Product Discounts pagination counts its own records instead of users. User Voucher listing/claim/USED operations enforce bearer JWT authentication, owner scoping and admin authorization as applicable. Invalid Product Discount URL UUIDs return 400 instead of the legacy internal 500. Prisma constraint errors are mapped to semantic HTTP statuses.\n
PR #2 security hardening also protects legacy Roles/Users mutations and all Promotions write routes with centralized current-database authorization, and validates claim eligibility with PostgreSQL voucher-row locking. See docs/api.md and docs/parity.md.
