# Promotions side-by-side HTTP parity (PR #2)

Run a **real NestJS server** and the Go rewrite with the same deterministic promotion fixtures, but in **two distinct disposable PostgreSQL databases**. Redis and MinIO test services are also disposable. This test never uses existing application databases, user credentials, or production services.

## Requirements

- Go 1.26+, Node.js and Docker with a working daemon, and Python 3.
- A local checkout of `adamandp/dam-creamy-cream-be` (original NestJS backend).
- Prepare the NestJS checkout once:

```bash
cd /path/to/dam-creamy-cream-be
npm ci
npx prisma generate
npm run build
```

From the Go repository:

```bash
bash tests/e2e/run_compare.sh /path/to/dam-creamy-cream-be
```

The runner creates isolated Docker containers, applies the five real Prisma migration SQL files, loads `tests/e2e/fixtures/promotions.sql`, starts both backends, compares responses and cleans up resources. It refuses to replace running services occupying required ports.

## Acceptance criteria

The original suite contains **19 requests: 11 GET and 8 mutations**. The improved Go implementation matches **15** exactly (HTTP status and parsed JSON) and intentionally differs in **4** documented source defects or ACL changes:

1. GET /product-discounts: totalItems now counts product discount records, not users.
2. GET /user-vouchers: non-admin customers receive 403 instead of seeing everyone's claims.
3. GET /user-vouchers/voucher/:id: non-admin customers receive 403.
4. GET /user-vouchers/user: Go returns the intended 200 + associated vouchers; NestJS mistakenly routes /user through /:id and returns 400.

The harness uses admin credentials on write routes and fresh, unclaimed vouchers for the claim flows. The source NestJS backend trusts claimedAt from the client, while Go now uses the server UTC clock; the post-write comparator explicitly checks this intentional difference and that expiration comes from the authoritative voucher endDate.

The runner then probes **additional regression cases**: Zod-style pagination with decimal/scientific representations, invalid parameters, exact validation JSON, camelCase no-change messages, repeated date updates, fractional numbers stored in PostgreSQL Int columns, verified admin and owner access, unauthorized claims/status transitions, and FK/route errors. Normalized post-write comparisons also assert business fields and UTC createdAt/updatedAt timestamps agree within 10 seconds despite independent generated UUIDs.

A passing run reports, at minimum:

    BASELINE=19; MATCH=34; APPROVED_CHANGES=13; UNEXPECTED=0
    PARITY AUDIT PASSED — all listed source deviations are intentional and verified

All differences not explicitly allowlisted are errors; the test exits nonzero. Some approved errors intentionally differ semantically (Go uses 409 for FK conflicts and 400 for malformed path UUIDs instead of legacy 500). Additional scenarios verify unauthorized role renaming and anonymous/customer promotion writes are blocked (401/403). Duplicate claims, real concurrency and capacity windows are covered by Go's PostgreSQL integration suite.

The suite starts both servers and database/storage/cache services using **disposable** fixtures and cleans up resources. The CI `nest-parity` job checks out the original NestJS repository separately. Go unit tests and real PostgreSQL/Redis/MinIO integrations run in another job. These tests do not claim exhaustive application parity.

## PR #5 final five domains

The runner now additionally loads `fixtures/remaining_domains.sql` into both
isolated databases and invokes `compare_remaining.py` **before** the Promotions
mutations. It covers all 28 original HTTP routes across Reviews (7),
Recommendations (4), Product Analytics (4), Notifications (5) and User
Notifications (8). Stable seeded rows and JSON responses are compared exactly;
four documented differences are allowlisted with explicit assertions:
1. `GET /reviews` crashes in Nest (500) because its controller argument is
   missing `@Query()`; Go serves the intended paginated list (200).
2. `GET /reviews/me` is shadowed by `GET /reviews/:id` in Nest (500 due invalid
   UUID); Go serves the intended self-scoped feed (200).
3. `GET /user-notifications/me` is shadowed by its `/:id` handler in Nest
   (400); Go serves the intended feed (200).
4. A 404 from `GET /user-notifications/notification/:id` has the same status
   and message but the existing Go error handler adds an `error` field.

Other unapproved differences fail the audit. The observed first side-by-side
run matched 24/28 exactly; the four known differences above are tested
separately and do not imply full application equivalence.

GitHub Actions has a dedicated `nest-parity` job that builds the original
NestJS source and runs this side-by-side script against disposable containers.
The earlier Promotions contract audit remains part of the same run. These
tests verify the enumerated cases, not every edge case or production-provider
interaction.
