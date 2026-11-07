# PR #4: Shipments and Payments compatibility

Source of truth: adamandp/dam-creamy-cream-be (NestJS ShipmentsController, ShipmentsService, PaymentsService). This PR stacks on PR #3 (feat/orders-checkout) and leaves previous branches and the Prisma database schema intact.

## HTTP contract

| Method | Route | Authorization | Behavior |
|---|---|---|---|
| GET | /shipments?page=1&limit=10 | Admin access JWT | Paginated shipment rows; 404 on empty collection as NestJS |
| GET | /shipments/:id | Owner or admin access JWT | Shipment JSON in data; UUID validation |
| PATCH | /shipments/:id/status | Admin access JWT | Status PENDING, SHIPPED, DELIVERED, or CANCELLED |
| POST | /shipments/cost | Public, as NestJS | HTTP 201; Origin/destination/weight body, RajaOngkir URL-encoded quote, response with message/status/data |
| POST | /payments | Public inert scaffold | HTTP 201, literal "This action adds a new payment" |
| GET | /payments | Public inert scaffold | HTTP 200, literal "This action returns all payments" |
| GET | /payments/:id | Public inert scaffold | HTTP 200, literal "This action returns a #N payment" |
| PATCH | /payments/:id | Public inert scaffold | HTTP 200, literal "This action updates a #N payment" |
| DELETE | /payments/:id | Public inert scaffold | HTTP 200, literal "This action removes a #N payment" |

NestJS PaymentsService contains only those placeholder strings. Path parameters in its generated scaffold are coerced with JavaScript's unary plus; the Go port preserves numeric radix, infinity and exponential formatting on these inert text endpoints. It has no actual persisted CRUD, gateway, capture, settlement or refunds. Exposing the same responses does not implement payment processing. Checkout PR #3 creates a separate PENDING Payment record.

## RajaOngkir integration

Configure both RAJAONGKIR_BASE_URL and RAJAONGKIR_API_KEY, or neither (quote route returns 503). In production the base URL must be HTTPS.

The backend sends application/x-www-form-urlencoded POST to the configured base URL plus /calculate/domestic-cost with origin, destination, weight and the original courier list:

jne:sicepat:ide:sap:jnt:ninja:tiki:lion:anteraja:pos:ncs:rex:rpx:sentral:star:wahana:dse

The API credential is sent in the upstream key header, never to the browser. **HTTP redirects are never followed**, including when callers inject a custom HTTP client, because Go otherwise forwards the custom key header to redirect targets. Redirects produce a generic upstream 502 response; regression tests cover 301, 302, 303, 307 and 308 without contacting their destinations. HTTP timeout is 10 seconds and response payload is capped at 1 MiB. Upstream HTTP/network/JSON failures return generic HTTP 502 without leaking the credential. JSON numbers and numeric strings are accepted only when they coerce to positive whole numbers within signed 32-bit range; fractional, nonpositive, NaN/Infinity and unknown-field inputs are rejected before contacting the provider. RajaOngkir requires integer region IDs and weights (in grams). This integer-input rule is deliberately stricter than NestJS numeric coercion.

The public quote endpoint uses an atomic Redis **fixed-window** limiter with **two simultaneous budgets**: per client IP (default 30 requests/60s, `RAJAONGKIR_RATE_LIMIT_PER_MINUTE`, valid range 1–1000) and **global per provider API** (default 30 requests/60s across all IPs and replicas, `RAJAONGKIR_GLOBAL_RATE_LIMIT_PER_MINUTE`, valid range 1–100000). All valid, configured quote requests consume one token from both budgets. A rejected request consumes neither. Set the global rate below your RajaOngkir API-key quota; the default is intentionally conservative.

Excess traffic returns HTTP 429 with `Retry-After: 60`; Redis errors fail closed with HTTP 503. The default Fiber configuration uses the direct TCP remote IP and **ignores untrusted forwarding headers**. For Nginx/load balancers, configure `TRUSTED_PROXY_CIDRS` with only the actual proxy IP addresses or CIDRs, e.g. `10.10.0.5/32`. This explicitly enables Fiber trusted-proxy handling of `X-Forwarded-For`; never configure arbitrary client networks or `0.0.0.0/0`. Without correct deployment-specific proxy settings, multiple users behind a gateway may share one IP quota. Malformed requests are rejected before reaching Redis or RajaOngkir.

Shipment list and detail API timestamps use UTC ISO-8601 with **exactly three millisecond digits** (`.000Z` through `.999Z`), matching Prisma DateTime JSON, including nullable `deliveredAt`.

Unit tests use a fake HTTP server to verify outbound request, form fields, header and response, and rate-limit deny/error paths. Redis integration tests verify per-IP isolation, combined global budget, no quota consumption on rejected requests, and atomic concurrency. Live RajaOngkir verification is NOT asserted without staging credentials.

## Authorization and production boundaries

NestJS UpdateStatusShipmentDto strips unknown status-body keys; this port retains that acceptance while still requiring a valid status. NestJS ShipmentsController does not authenticate list/detail/status routes. PR #4 deliberately requires signed current access JWT with owner/admin checks on reads and admin-only writes. Status updates do not automatically mutate Order.status or Payment.status, preserving NestJS mutation scope.

Important production limitations:

1. Checkout PR #3 still accepts caller-supplied shipment.cost. The quote route is informational only and not linked to a signed/time-limited quote ID binding user, address, courier, weight and expiry.
2. No payment gateway, reconciliation or actual PaymentsService processing exists. The /payments routes remain inert.
3. No automatic stock release upon order expiry/cancellation, shipment booking, tracking reconciliation or refunds.
4. The three PR #3 review findings (fractional-price rounding, FREE_ITEM quantity cap, and missing required shipment fields) are not fixed in this PR.

No schema migration, Cloudinary change or dummy-login modification.
