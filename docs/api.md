# Creamy Cream Go — HTTP API

All routes are relative to the backend root. List endpoints accept page (default 1) and limit (default 10). JSON responses use message and optional data/paging.

## Core modules

| Route group | Features |
|---|---|
| /auth | Login, logout and logout-all |
| /sessions | Access-token refresh |
| /roles | CRUD, get by name |
| /users | Profile, update, password, image, delete |
| /addresses | Create, list, user addresses, update, delete |
| /categories | CRUD and Redis-cached list |
| /products | CRUD, catalog and images |
| /product | Detail and product information |
| /carts | Current-user cart, add, remove |
| /inventory | List, lookup and update |
| /health | HTTP and PostgreSQL status |
| /media/:scope/:id/:filename | Public image delivery from private storage |

## Discounts

| Method | Path | Behavior |
|---|---|---|
| POST | /discounts | Create discount |
| GET | /discounts | Paginated list |
| GET | /discounts/:id | Get discount |
| PATCH | /discounts/:id | Update discount |
| DELETE | /discounts/:id | Hard delete |

Discount types: PERCENTAGE, FIXED, PRICE, FREE_ITEM. Nullable discount values, descriptions, product references and quantities must be present on creation. Current list response labels use User and Users for the list and empty error respectively.

## Product discounts

| Method | Path | Behavior |
|---|---|---|
| POST | /product-discounts | Assign discount to product |
| GET | /product-discounts | Paginated list |
| GET | /product-discounts/:id | Get assignment |
| GET | /product-discounts/product/:productId | List by product |
| DELETE | /product-discounts/:id | Set isActive=false |

Product discount assignments have one row per product (unique product ID). Disabled assignments remain queryable. The list totalItems currently counts user records.

## Vouchers

| Method | Path | Behavior |
|---|---|---|
| POST | /vouchers | Create voucher |
| GET | /vouchers | Paginated list |
| GET | /vouchers/:id | Get voucher |
| PATCH | /vouchers/:id | Update voucher |
| DELETE | /vouchers/:id | Set isActive=false |

Voucher types: PERCENTAGE, FIXED and FREE_ITEM. Groups: CHECKOUT and SHIPPING. Soft-disabled vouchers remain readable.

## User vouchers

| Method | Path | Behavior |
|---|---|---|
| POST | /user-vouchers | Create a user claim |
| GET | /user-vouchers | Paginated list |
| GET | /user-vouchers/user | List current user's claims; Bearer token |
| GET | /user-vouchers/voucher/:id | Claims by voucher |
| GET | /user-vouchers/:id | Get claim |
| PATCH | /user-vouchers/:id | Set status to USED |

Claims start as CLAIMED. The /user endpoint returns each claim with a full associated voucher in its vouchers field. The use endpoint sets USED without checking expiration or redemption count.

## General contracts

- Creation returns 201; read/update/delete returns 200 on success.
- Validation errors return 400; missing rows 404; unique constraint conflicts 409.
- No-op updates are rejected on endpoints with an update guard.
- Datetime fields are serialized as UTC ISO timestamps with milliseconds.
