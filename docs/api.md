# Creamy Cream Go — HTTP API

All routes are relative to the backend root. List endpoints accept page (default 1) and limit (default 10). JSON responses use message and optional data/paging.

## Core modules

| Route group | Features |
|---|---|
| /auth | Login, logout and logout-all |
| /sessions | Access-token refresh |
| /roles | Public reads; admin-only create, update, delete |
| /users | Public reads; profile login required; own profile update and password; image owner/admin; role/status update and delete admin only |
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

The /user endpoint returns each claim with its associated voucher under the vouchers property. The admin use endpoint marks USED; checkout transaction redemption and settlement are not ported yet. Access JWTs are signature-verified and checked against the CURRENT user, active flag and role in PostgreSQL. Admin/Administrator/Super Admin/Superadmin role names have elevated access (case-insensitive); customer access is owner-scoped. Read access to Discounts, Product Discounts and Vouchers is public; all their write operations require a verified current admin in PostgreSQL.

Role creation/renaming/deletion is admin-only. Customers can edit their own safe profile fields, but cannot set roleId or isActive or edit another user's account. Password changes require the authenticated account owner. User deletion is admin-only. Role changes invalidate access tokens with the old role claim. Admin accounts must already exist in the database or be provisioned out-of-band; there is no public bootstrap-admin endpoint.

## General contracts

- Creation returns 201; read/update/delete returns 200 on success.
- Invalid DTO/parameter errors return 400; promotion DTOs provide source-compatible Zod errors[]; missing rows return 404; unique and foreign-key constraints return 409.
- No-op updates are rejected on endpoints with an update guard.
- Datetime fields are serialized as UTC ISO timestamps with milliseconds. The GORM clock writes UTC even on non-UTC hosts, matching Prisma TIMESTAMP(3) columns.
- Zod-style pagination accepts numeric notations such as page=1.0 and limit=1e1. Noncanonical URL UUIDs are rejected with 400.
- Promotion DTO numeric fields accept fractions as source Zod does. Prisma-backed Int columns store the integer portion.
- User-voucher ACL, product-discount counts and semantic database errors are approved legacy corrections.
