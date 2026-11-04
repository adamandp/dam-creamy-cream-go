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

Product discount assignments have one row per product (unique product ID). Disabled assignments remain queryable. The list totalItems counts product-discount rows (fixing the NestJS bug that counted users).

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
| POST | /user-vouchers | Bearer JWT; customer may claim for self, admin may assign |
| GET | /user-vouchers | Bearer JWT; admin only, paginated |
| GET | /user-vouchers/user | Bearer JWT; current user's own claims |
| GET | /user-vouchers/voucher/:id | Bearer JWT; admin only |
| GET | /user-vouchers/:id | Bearer JWT; owner or admin only |
| PATCH | /user-vouchers/:id | Bearer JWT; admin only, marks USED |

Claims start as CLAIMED. The /user endpoint returns each claim with its associated voucher under the vouchers property. The use endpoint marks USED without evaluating expiration/redemption limits (checkout transaction logic remains unported). Access JWTs are signature-verified and checked against the CURRENT user, active flag and role in PostgreSQL. Admin/Administrator/Super Admin/Superadmin role names have elevated access (case-insensitive); customer access is owner-scoped.

## General contracts

- Creation returns 201; read/update/delete returns 200 on success.
- Invalid DTO/parameter errors return 400; promotion DTOs provide source-compatible Zod errors[]; missing rows return 404; unique and foreign-key constraints return 409.
- No-op updates are rejected on endpoints with an update guard.
- Datetime fields are serialized as UTC ISO timestamps with milliseconds. The GORM clock writes UTC even on non-UTC hosts, matching Prisma TIMESTAMP(3) columns.\n- Zod-style pagination accepts numeric notations such as page=1.0 and limit=1e1. Noncanonical URL UUIDs are rejected with 400.\n- Promotion DTO numeric fields accept fractions as source Zod does. Prisma-backed Int columns store the integer portion.\n- User-voucher ACL, product-discount counts and semantic database errors are approved legacy corrections.
