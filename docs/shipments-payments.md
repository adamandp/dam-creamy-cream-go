# PR #4: Shipments and Payments compatibility

Source of truth: adamandp/dam-creamy-cream-be (NestJS ShipmentsController, ShipmentsService, PaymentsService). This PR stacks on PR #3 (feat/orders-checkout) and leaves previous branches and the Prisma database schema intact.

## HTTP contract

| Method | Route | Authorization | Behavior |
|---|---|---|---|
| GET | /shipments?page=1&limit=10 | Admin access JWT | Paginated shipment rows; 404 on empty collection as NestJS |
| GET | /shipments/:id | Owner or admin access JWT | Shipment JSON in data; UUID validation |
| PATCH | /shipments/:id/status | Admin access JWT | Status PENDING, SHIPPED, DELIVERED, or CANCELLED |
| POST | /shipments/cost | Public, as NestJS | Origin/destination/weight body, RajaOngkir URL-encoded quote, response with message/status/data |
| POST | /payments | Public inert scaffold | HTTP 201, literal "This action adds a new payment" |
| GET | /payments | Public inert scaffold | HTTP 200, literal "This action returns all payments" |
| GET | /payments/:id | Public inert scaffold | HTTP 200, literal "This action returns a #N payment" |
| PATCH | /payments/:id | Public inert scaffold | HTTP 200, literal "This action updates a #N payment" |
| DELETE | /payments/:id | Public inert scaffold | HTTP 200, literal "This action removes a #N payment" |

NestJS PaymentsService contains only those placeholder strings. It has no actual persisted CRUD, gateway, capture, settlement or refunds. Exposing the same responses does not implement payment processing. Checkout PR #3 creates a separate PENDING Payment record.

## RajaOngkir integration

Configure both RAJAONGKIR_BASE_URL and RAJAONGKIR_API_KEY, or neither (quote route returns 503). In production the base URL must be HTTPS.

The backend sends application/x-www-form-urlencoded POST to the configured base URL plus /calculate/domestic-cost with origin, destination, weight and the original courier list:

jne:sicepat:ide:sap:jnt:ninja:tiki:lion:anteraja:pos:ncs:rex:rpx:sentral:star:wahana:dse

The API credential is sent in the upstream key header, never to the browser. HTTP timeout is 10 seconds and response payload is capped at 1 MiB. Upstream HTTP/network/JSON failures return generic HTTP 502 without leaking the credential. JSON numbers and numeric strings are accepted, while nonpositive values, NaN/Infinity and unknown fields are rejected. This positive-input rule is deliberately stricter than NestJS numeric coercion.

Unit tests use a fake HTTP server to verify outbound request, form fields, header and response. Live RajaOngkir verification is NOT asserted without staging credentials.

## Authorization and production boundaries

NestJS ShipmentsController does not authenticate list/detail/status routes. PR #4 deliberately requires signed current access JWT with owner/admin checks on reads and admin-only writes. Status updates do not automatically mutate Order.status or Payment.status, preserving NestJS mutation scope.

Important production limitations:

1. Checkout PR #3 still accepts caller-supplied shipment.cost. The quote route is informational only and not linked to a signed/time-limited quote ID binding user, address, courier, weight and expiry.
2. No payment gateway, reconciliation or actual PaymentsService processing exists. The /payments routes remain inert.
3. No automatic stock release upon order expiry/cancellation, shipment booking, tracking reconciliation or refunds.
4. The three PR #3 review findings (fractional-price rounding, FREE_ITEM quantity cap, and missing required shipment fields) are not fixed in this PR.

No schema migration, Cloudinary change or dummy-login modification.
