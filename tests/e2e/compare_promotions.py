"""NestJS vs Go promotion contract audit on isolated, identically seeded databases.

Behavior changes approved in PR #2 are explicitly asserted, not silently
treated as equality. Unlisted differences always fail this audit.
"""
import base64
import hashlib
import hmac
import json
from datetime import datetime
import os
import sys
import time
import urllib.error
import urllib.request

NEST = int(os.environ.get("NEST_PORT", "4081"))
GO = int(os.environ.get("GO_PORT", "4082"))
D = "55555555-5555-4555-8555-555555555555"
PD = "66666666-6666-4666-8666-666666666666"
P = "44444444-4444-4444-8444-444444444444"
V = "77777777-7777-4777-8777-777777777777"
UV = "88888888-8888-4888-8888-888888888888"
CUSTOMER = "22222222-2222-4222-8222-222222222222"
OTHER = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
ADMIN = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
CUSTOMER_ROLE = "11111111-1111-4111-8111-111111111111"
ADMIN_ROLE = "99999999-9999-4999-8999-999999999999"


def jwt(subject, role):
    encode = lambda obj: base64.urlsafe_b64encode(
        json.dumps(obj, separators=(",", ":")).encode()
    ).decode().rstrip("=")
    head = encode({"alg": "HS256", "typ": "JWT"})
    claims = encode({"sub": subject, "role": role, "iat": int(time.time()), "exp": int(time.time()) + 900})
    body = (head + "." + claims).encode()
    sig = base64.urlsafe_b64encode(
        hmac.new(b"compare-access-secret", body, hashlib.sha256).digest()
    ).decode().rstrip("=")
    return head + "." + claims + "." + sig


TOKENS = {
    "customer": jwt(CUSTOMER, CUSTOMER_ROLE),
    "admin": jwt(ADMIN, ADMIN_ROLE),
    "other": jwt(OTHER, CUSTOMER_ROLE),
}
DEFAULT_DISCOUNT = dict(
    name="New Sale", discountType="FIXED", value=12, maxApplied=1,
    maxRedeemed=2, redeemedCount=0, description=None, productId=None,
    quantity=None, startDate="2026-01-01", endDate="2027-01-01"
)
DEFAULT_VOUCHER = dict(
    code="SPRING15", name="Spring Sale", description=None, type="FIXED",
    group="CHECKOUT", value=15, productId=None, quantity=None,
    minimumPurchase=100, isActive=True, startDate="2026-01-01",
    endDate="2027-01-01", maxRedeemed=50, redeemedCount=0
)
CLAIM = dict(userId=CUSTOMER, voucherId=V, claimedAt="2026-01-01", expiredAt="2027-01-01")


def request(port, method, path, body=None, actor="customer"):
    payload = json.dumps(body).encode() if body is not None else None
    headers = {"Content-Type": "application/json"}
    if actor is not None:
        headers["Authorization"] = "Bearer " + TOKENS[actor]
    call = urllib.request.Request(
        f"http://127.0.0.1:{port}{path}", data=payload,
        method=method, headers=headers
    )
    try:
        with urllib.request.urlopen(call, timeout=8) as response:
            return response.status, json.loads(response.read())
    except urllib.error.HTTPError as error:
        try:
            return error.code, json.loads(error.read())
        except Exception:
            return error.code, None


exact = 0
intentional = 0
failures = 0
baseline = 0


def verify(name, method, path, payload=None, actor="customer", approved=None, baseline_case=False, equivalent=None):
    global exact, intentional, failures, baseline
    if baseline_case:
        baseline += 1
    nest = request(NEST, method, path, payload, actor)
    go = request(GO, method, path, payload, actor)
    if approved is None:
        success = equivalent(nest, go) if equivalent is not None else nest == go
    else:
        success = approved(nest, go)
    if not success:
        failures += 1
        print(f"FAIL {name}: Nest={nest[0]} Go={go[0]}")
        print(" Nest:", json.dumps(nest[1], ensure_ascii=False, sort_keys=True)[:900])
        print(" Go:  ", json.dumps(go[1], ensure_ascii=False, sort_keys=True)[:900])
        return
    if approved is not None:
        intentional += 1
        print(f"APPROVED {name}: Nest={nest[0]} Go={go[0]}")
    else:
        exact += 1
        print(f"MATCH {name}: {go[0]}")


def statuses(n, g, old, new):
    return n[0] == old and g[0] == new


def fixed_user_route(n, g):
    return (
        statuses(n, g, 400, 200)
        and isinstance(n[1], dict)
        and n[1].get("message") == "Validation failed (uuid is expected)"
        and isinstance(g[1], dict)
        and len(g[1].get("data", [])) == 1
        and isinstance(g[1]["data"][0].get("vouchers"), dict)
    )


def records_and_utc_timestamps_match(n, g):
    # Generated database UUIDs and subsecond creation times are naturally
    # distinct between two independent servers; all stable business fields
    # must match, and both servers must write timestamps on the same UTC clock.
    if n[0] != 200 or g[0] != 200:
        return False
    if n[1].get("message") != g[1].get("message") or n[1].get("paging") != g[1].get("paging"):
        return False
    def stable(record):
        return {k: v for k, v in record.items() if k not in ("id", "createdAt", "updatedAt")}
    def sorted_records(value):
        return sorted(value, key=lambda x: (json.dumps(stable(x), sort_keys=True), x["createdAt"]))
    a, b = sorted_records(n[1].get("data", [])), sorted_records(g[1].get("data", []))
    if len(a) != len(b):
        return False
    for x, y in zip(a, b):
        if stable(x) != stable(y):
            return False
        for field in ("createdAt", "updatedAt"):
            old = datetime.fromisoformat(x[field].replace("Z", "+00:00"))
            new = datetime.fromisoformat(y[field].replace("Z", "+00:00"))
            if abs((old - new).total_seconds()) > 10:
                return False
    return True


def corrected_count(n, g):
    return (
        statuses(n, g, 200, 200)
        and isinstance(n[1], dict)
        and isinstance(g[1], dict)
        and n[1].get("data") == g[1].get("data")
        and n[1]["paging"]["totalItems"] == 3
        and g[1]["paging"]["totalItems"] == 1
        and n[1]["paging"]["totalPages"] == 1
        and g[1]["paging"]["totalPages"] == 1
    )


# Original 11 GETs. Four approved improvements no longer return exactly
# the same data/status due role restrictions, pagination fix, route fix.
for path, approved in [
    ("/discounts", None),
    ("/discounts/" + D, None),
    ("/product-discounts", corrected_count),
    ("/product-discounts/" + PD, None),
    ("/product-discounts/product/" + P, None),
    ("/vouchers", None),
    ("/vouchers/" + V, None),
    ("/user-vouchers", lambda n, g: statuses(n, g, 200, 403)),
    ("/user-vouchers/" + UV, None),
    ("/user-vouchers/voucher/" + V, lambda n, g: statuses(n, g, 200, 403)),
    ("/user-vouchers/user", fixed_user_route),
]:
    verify("GET " + path, "GET", path, approved=approved, baseline_case=True)

# Read comparisons run before new records introduce distinct IDs and timestamps.
verify("coerced page=1.0", "GET", "/discounts?page=1.0")
verify("coerced limit=1e1", "GET", "/vouchers?limit=1e1")
verify("role-filtered global list", "GET", "/user-vouchers", actor="admin")
verify("role-filtered voucher list", "GET", "/user-vouchers/voucher/" + V, actor="admin")

# Original 8 writes: use the correct signed identity for the new Go ACL.
for method, path, payload, actor in [
    ("PATCH", "/discounts/" + D, {"maxApplied": 7}, "customer"),
    ("PATCH", "/vouchers/" + V, {"minimumPurchase": 40000}, "customer"),
    ("PATCH", "/user-vouchers/" + UV, None, "admin"),
    ("DELETE", "/product-discounts/" + PD, None, "customer"),
    ("DELETE", "/vouchers/" + V, None, "customer"),
    ("POST", "/discounts", DEFAULT_DISCOUNT, "customer"),
    ("POST", "/vouchers", DEFAULT_VOUCHER, "customer"),
    ("POST", "/user-vouchers", CLAIM, "customer"),
]:
    verify(method + " " + path, method, path, payload, actor, baseline_case=True)

# Additional positive and negative Zod/route regressions.
verify("invalid numeric page Zod errors", "GET", "/discounts?page=abc")
verify("out-of-bounds pagination Zod errors", "GET", "/vouchers?page=0")
verify("invalid parameter UUID", "GET", "/discounts/invalid-id")
verify("bracketed UUID rejected", "GET", "/discounts/%7B" + D + "%7D")
verify("UUID URN rejected", "GET", "/discounts/urn:uuid:" + D)
verify("unchanged field name camelCase", "PATCH", "/discounts/" + D, {"name": "New Title", "maxApplied": 7})
verify("unchanged date accepted", "PATCH", "/discounts/" + D, {"startDate": "2026-01-01T00:00:00.000Z"})
verify("unchanged voucher date accepted", "PATCH", "/vouchers/" + V, {"endDate": "2027-01-01T00:00:00.000Z"})
fraction_discount = dict(DEFAULT_DISCOUNT, name="Fraction Sale", maxApplied=1.5)
verify("fractional Int discount accepted", "POST", "/discounts", fraction_discount)
fraction_voucher = dict(DEFAULT_VOUCHER, code="FRACTION15", name="Fraction Sale", maxRedeemed=1.5)
verify("fractional Int voucher accepted", "POST", "/vouchers", fraction_voucher)
invalid_number = dict(DEFAULT_DISCOUNT, name="Invalid Number", maxApplied="not a number")
verify("Zod input validation JSON", "POST", "/discounts", invalid_number)
verify("date numeric string coercion", "POST", "/user-vouchers", dict(CLAIM, claimedAt="0"))
verify("new discount records and UTC timestamps", "GET", "/discounts", equivalent=records_and_utc_timestamps_match)
verify("new voucher records and UTC timestamps", "GET", "/vouchers", equivalent=records_and_utc_timestamps_match)
verify("claim records and UTC timestamps", "GET", "/user-vouchers", actor="admin", equivalent=records_and_utc_timestamps_match)
verify("non-owner claim access blocked", "GET", "/user-vouchers/" + UV, actor="other",
       approved=lambda n, g: statuses(n, g, 200, 403))
verify("customer cannot update status", "PATCH", "/user-vouchers/" + UV, actor="customer",
       approved=lambda n, g: statuses(n, g, 200, 403))
verify("unauthenticated claim blocked", "POST", "/user-vouchers", CLAIM, actor=None,
       approved=lambda n, g: statuses(n, g, 201, 401))
verify("cross-account claim blocked", "POST", "/user-vouchers", dict(CLAIM, userId=OTHER),
       approved=lambda n, g: statuses(n, g, 201, 403))
verify("foreign-key error is semantic conflict", "POST", "/vouchers",
       dict(DEFAULT_VOUCHER, code="BADFOREIGN", productId="00000000-0000-4000-8000-000000000000"),
       approved=lambda n, g: statuses(n, g, 500, 409))
verify("invalid product-discount route UUID is 400", "GET", "/product-discounts/invalid-id",
       approved=lambda n, g: statuses(n, g, 500, 400))

print(f"BASELINE={baseline}; MATCH={exact}; APPROVED_CHANGES={intentional}; UNEXPECTED={failures}")
if failures != 0 or baseline != 19 or exact < 19 or intentional < 4:
    sys.exit("PARITY AUDIT FAILED")
print("PARITY AUDIT PASSED — all listed source deviations are intentional and verified")
