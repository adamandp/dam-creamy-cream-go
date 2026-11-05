# PR #3 — Orders / Checkout

This batch is stacked on PR #2 (`feat/promotions-discounts-vouchers`); do not merge the earlier branches to `main` just to review it.

## Provenance

NestJS has a `CheckoutOrderHandler` with 17 CQRS steps but its `OrdersController` exposes **no HTTP endpoints**. Go PR #3 introduces an explicit, owner-authenticated route and ports the underlying transaction semantics. Where the old CQRS steps were unsafe (cross-user address, inactive discounts, negative totals, insufficient stock, duplicated product lines, gift-item stock and voucher claims), the Go behavior intentionally fixes them.

## HTTP routes

| Route | Authorization | Behavior |
|---|---|---|
| POST /orders/checkout | Bearer access JWT, authenticated account owner | Atomically create Order, OrderItem(s), OrderVoucher(s), Payment `PENDING` and Shipment `PENDING`; update Cart, Inventory and voucher/discount redemption counts |
| GET /orders | Current Admin only | Paginated order list (page/limit) |
| GET /orders/user | Signed-in current customer | Paginated *own* order list |
| GET /orders/:id | Owner or Admin | Order, OrderItem(s), Payment, Shipment and applied voucher IDs |

Checkout request (nested shape matches the existing NestJS CQRS DTO):

```json
{
  "order": {
    "userId": "6ac3e441-d7cb-402a-8aae-e4141b5eeb16",
    "addressId": "27868b32-c4d9-4c25-95e3-d031bb5e32d8",
    "voucherIds": []
  },
  "orderItems": [
    {
      "productId": "0aa7d631-e7a6-45c1-b172-02e3a2b9194d",
      "discountId": null,
      "quantity": 2
    }
  ],
  "shipment": {
    "courier": "jne",
    "service": "regular",
    "cost": 8000,
    "estimated": "2026-12-01T12:00:00Z",
    "description": null
  },
  "payment": {
    "method": "BCA"
  }
}
```

Checkout response: HTTP 201 with `data.orderId`, `totalPrice`, `discountAmount`, `voucherAmount`, `shippingCost`, `finalPrice`, and `status=PENDING`. Prices, product details, discount amounts, payment total and voucher ownership are obtained and validated on the server. Shipping cost is presently the request value subject to numeric validation, **not** a carrier-verified price.

## Transaction and invariants

1. Authenticate against current active user and role. Require `order.userId` to be the actual actor, and ensure `order.addressId` belongs to that user. A valid session is insufficient to use another account's address.
2. Reject duplicate product/voucher IDs, invalid UUIDs, quantity outside 1..100, non-finite/negative shipping fees, unknown JSON fields, and more than two vouchers.
3. Lock the customer's Cart and CartItems using PostgreSQL `FOR UPDATE`; verify that every checkout line exists and has sufficient quantity. Read product price from the database and require active product.
4. Verify active ProductDiscount relationship for each explicitly requested discount, lock Discount rows, validate date windows, per-order limit and remaining redemption quota. Public Cart/Catalog/Detail prices use the same global quota predicate (maxRedeemed > 0 and redeemedCount < maxRedeemed), so an already-exhausted promo is not advertised; a final remaining redemption can be consumed concurrently, hence Checkout remains authoritative. Cart, Catalog, Product Detail and Checkout share one calculator for PERCENTAGE, FIXED and PRICE. PRICE is a final-price override (never an amount subtracted), including a valid zero-price override. FREE_ITEM leaves the paid product price untouched and creates a separate bonus line. The calculator rounds to currency cents so storefront quotes and persisted order-item prices match. The `PRICE` discount means price override bounded by original product price.
5. Lock Vouchers and UserVoucher claims, require current validity, ownership, `CLAIMED` status, capacity, minimum purchase, and at most one voucher for each `CHECKOUT` / `SHIPPING` group. Apply fixed/percentage rewards with zero floor.
6. Materialize `FREE_ITEM` rewards as zero-cost order lines; include reward product inventory in required quantities (even if the reward does not appear in Cart).
7. Lock inventory rows in sorted product UUID order with PostgreSQL `FOR UPDATE`, reject inactive or insufficient stock, and increment `reserved_quantity`. This matches the stock invariant enforced by Inventory PATCH in PR #2.
8. In the **same** GORM transaction, create Order/OrderItems/OrderVouchers and Payment/Shipment as `PENDING`; decrement or delete purchased CartItems, increment Discount/Voucher redemption counters, and mark used UserVoucher claims. Any error, including an insert failure at the final Shipment step, rolls the entire transaction back.

## Known boundaries, not implemented in PR #3

- No external payment provider, transaction capture, order settlement/cancel/refund, payment expiration cleanup or automatic release of reserved stock. Payment and Shipment are **pending records**, not evidence of payment or dispatch.
- No carrier-quoted shipping prices. The current `shipment.cost` is supplied by the caller as in the old CQRS DTO; do not treat this endpoint as production-ready for final shipping charges until a trusted shipping quote is integrated.
- No durable client idempotency key in the old schema. A consumed Cart line or already-USED voucher cannot be replayed, but two distinct checkout requests with sufficiently many Cart units can still represent two genuine orders. Introduce durable idempotency for at-least-once clients in a subsequent production-hardening iteration.
- Other modules (Payments API, Shipments API, reviews, notifications, recommendations, etc.) remain outside PR #3.

## Verification

- `TestCheckoutAtomicPricingVoucherInventoryAndOwner`: owner checks, rejection paths, item prices, shipping/checkout vouchers, payment/shipment records, Cart/Inventory updates, voucher status, replay, privacy, and a forced late Shipment insert exception with full rollback.
- `TestCheckoutConcurrentBuyersCannotOversell`: two distinct DB transactions compete for insufficient stock; exactly one Order commits.
- `TestCheckoutConcurrentBuyersCannotDoubleRedeemVoucher`: a limited voucher is claimed by two distinct buyers; exactly one Order commits and the claim/counter is consistent.
- `TestCheckoutFreeGiftRewardsAlsoReserveStock`: a free reward can be granted from both discount and voucher; insufficient gift stock rejects checkout, and successful checkout reserves all free and paid units.
- Existing PR #1/2 tests and independent NestJS-vs-Go promotion parity runner are retained. No database migrations and no changes to the deliberate dummy-customer-login.


## PR #3 follow-up: PRICE / FREE_ITEM consistency (2026-11-05)

- Fixed an HTTP/PostgreSQL-reproduced price mismatch: for an original price of 10,000 with discount type PRICE and value 7,000, Cart/Catalog/Detail formerly displayed 3,000 while Checkout charged 7,000. All four paths now use the same shared pricing calculation and return 7,000.
- FREE_ITEM is a separate, zero-cost reward order line with real inventory reservation; its optional value is *never* subtracted from the purchased product price.
- Shared price semantics: PERCENTAGE subtracts a percentage; FIXED subtracts a fixed amount (floor zero); PRICE overrides the final paid price (capped by the original price, zero allowed); FREE_ITEM has no per-line discountPrice.
- Regression test TestAllDiscountTypesMatchCheckout exercises all four paths against a real PostgreSQL schema for nine cases, including fractional percentages, price overrides of 0 or above the original price, and FREE_ITEM with both present and null values. The pricing helper has unit tests independent of PostgreSQL.


## PR #3 follow-up: exhausted discount storefront consistency (2026-11-05)

- Fixed an HTTP/PostgreSQL-reproduced mismatch: Cart, Catalog and Product Detail previously advertised an active, in-date discount even when its global redeemedCount had already reached maxRedeemed; Checkout correctly returned 409. Shared SQL active-discount predicate now additionally requires maxRedeemed > 0 and redeemedCount < maxRedeemed.
- Checkout row lock and quantity-aware quota validation remain unchanged. Showing an available per-unit discount does not guarantee enough redemptions for an arbitrary cart quantity or availability after concurrent checkouts.
- Real PostgreSQL tests cover one remaining redemption, exhausted quota, over-redeemed count, zero quota, reopening quota, Checkout rejection without a partial Order, and a successful Checkout consuming the last redemption and immediately hiding the promo again (including a freshly added CartItem). No schema migration.
