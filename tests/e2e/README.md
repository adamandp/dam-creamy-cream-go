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

**19 total requests:** 11 GET and 8 write methods. The baseline is the original NestJS backend's actual HTTP responses.

- **18 requests must have equal status code and parsed JSON.** Includes list pagination, detail/association responses, voucher claims and updates.
- **1 intentional exception:** `GET /user-vouchers/user`. NestJS declares `@Get(':id')` above `@Get('/user')`, causing the literal path to hit the UUID parser and return HTTP 400 (`Validation failed (uuid is expected)`). Go correctly registers `/user` before `/:id` and returns HTTP 200 with the authenticated user's voucher list. The comparison explicitly asserts the NestJS bug and the corrected Go behavior.
- **Zero untracked mismatches.** Any other difference or a change to the expected route behavior makes the comparison exit nonzero.

A passing result must end with:

```text
EXACT=18; INTENTIONAL_ROUTE_FIX=1; UNEXPECTED_DIFFERENCES=0
PARITY AUDIT PASSED: 18 exact; 1 documented intentional NestJS routing bug fix
```

This is targeted PR #2 acceptance, **not** evidence that every edge case or unported module is 1:1. Go-only unit and integration tests are run by GitHub Actions; this side-by-side comparison requires the original NestJS source checkout and is executed separately.
