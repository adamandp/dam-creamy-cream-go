undefined

## PR #4: Shipment boundary and Payments scaffold

The Shipments feature protects persisted metadata by admin/owner access and lets admins update the Shipment status enum. Its quote client is a bounded-timeout HTTP adapter for RajaOngkir; credentials remain on the server and the response is size-limited. The provider can be disabled in development; POST /shipments/cost then returns 503. Quotes are informational and are not attached to Checkout yet. The Payments feature remains the source-generated text-only scaffold and cannot charge or settle funds. See [Shipments / Payments](shipments-payments.md).
