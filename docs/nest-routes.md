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

All paths outside first-pass modules are **source inventory only**, not registered endpoints in the Go application yet.
