"""Exercise all 28 remaining NestJS routes against isolated Nest and Go APIs.

The runner treats source bugs as behavior, not automatic authorization to
change them. Explicitly allowlisted route corrections must be observed on
both servers; every other unexpected HTTP/JSON difference fails the audit.
"""

import base64
import hashlib
import hmac
import json
import os
import time
import urllib.error
import urllib.request

NEST = int(os.getenv("NEST_PORT", "4081"))
GO = int(os.getenv("GO_PORT", "4082"))
CUSTOMER = "22222222-2222-4222-8222-222222222222"
OTHER = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
ROLE = "11111111-1111-4111-8111-111111111111"
PRODUCT = "44444444-4444-4444-8444-444444444444"
CLASSIC = "dddddddd-dddd-4ddd-8ddd-dddddddddddd"
REVIEW = "e1111111-1111-4111-8111-111111111111"
NOTIFICATION = "e5555555-5555-4555-8555-555555555555"
DELETE_NOTIFICATION = "e6666666-6666-4666-8666-666666666666"
USER_NOTIFICATION = "e7777777-7777-4777-8777-777777777777"


def access_token():
    def b64(value):
        return base64.urlsafe_b64encode(
            json.dumps(value, separators=(",", ":")).encode()
        ).decode().rstrip("=")

    head = b64({"alg": "HS256", "typ": "JWT"})
    now = int(time.time())
    payload = b64({"sub": CUSTOMER, "role": ROLE, "iat": now, "exp": now + 900})
    signed = f"{head}.{payload}".encode()
    signature = base64.urlsafe_b64encode(
        hmac.new(b"compare-access-secret", signed, hashlib.sha256).digest()
    ).decode().rstrip("=")
    return f"{head}.{payload}.{signature}"


TOKEN = access_token()


def request(port, method, path, payload=None, authorized=True):
    body = json.dumps(payload).encode() if payload is not None else None
    headers = {"Content-Type": "application/json"}
    if authorized:
        headers["Authorization"] = "Bearer " + TOKEN
    req = urllib.request.Request(
        f"http://127.0.0.1:{port}{path}", data=body, method=method, headers=headers
    )
    try:
        with urllib.request.urlopen(req, timeout=10) as response:
            return response.status, json.loads(response.read())
    except urllib.error.HTTPError as exc:
        raw = exc.read()
        try:
            return exc.code, json.loads(raw)
        except (ValueError, UnicodeError):
            return exc.code, raw.decode("utf-8", errors="replace")


match = 0
approved = 0
failed = 0
tested = 0


def verify(label, method, path, payload=None, authorized=True, approved_difference=None):
    global match, approved, failed, tested
    tested += 1
    nest = request(NEST, method, path, payload, authorized)
    go = request(GO, method, path, payload, authorized)
    if nest == go:
        match += 1
        print(f"MATCH {label}: {nest[0]}", flush=True)
        return
    allowed = False
    if approved_difference == "missing_query":
        # ReviewsController.findAll has no @Query() argument decorator. Nest
        # passes undefined and ReviewsService destructuring crashes with 500.
        allowed = nest == (500, {"statusCode": 500, "message": "Internal server error"}) and go[0] == 200
    elif approved_difference == "review_me_shadow":
        # Nest registers /reviews/:id before /reviews/me (no UUID pipe).
        allowed = nest[0] == 500 and "invalid input syntax for type uuid" in str(nest[1]) and go[0] == 200
    elif approved_difference == "user_notification_me_shadow":
        allowed = nest[0] == 400 and go[0] == 200
    elif approved_difference == "go_error_envelope":
        # The pre-existing Go shared error handler adds `error` to 404s;
        # preserve its project-wide convention rather than changing it here.
        allowed = (nest[0] == go[0] == 404 and isinstance(nest[1], dict)
                   and isinstance(go[1], dict)
                   and {k: v for k, v in go[1].items() if k != "error"} == nest[1]
                   and go[1].get("error") == "Not Found")
    if allowed:
        approved += 1
        print(f"APPROVED {approved_difference} {label}: Nest={nest[0]} Go={go[0]}", flush=True)
        return
    failed += 1
    print(f"FAIL {label}: Nest={nest[0]} Go={go[0]}", flush=True)
    print(" Nest:", json.dumps(nest[1], ensure_ascii=False, sort_keys=True)[:1200])
    print(" Go:  ", json.dumps(go[1], ensure_ascii=False, sort_keys=True)[:1200])


# All read contracts before any writes, on identical seeded databases.
for path, difference in [
    ("/reviews", "missing_query"),
    ("/reviews/product/" + PRODUCT, None),
    ("/reviews/" + REVIEW, None),
    ("/reviews/me", "review_me_shadow"),
    ("/recommendations/classic", None),
    ("/recommendations/categories", None),
    ("/recommendations/offers", None),
    ("/recommendations/related/" + PRODUCT, None),
    ("/product-analytics/product/" + PRODUCT, None),
    ("/product-analytics/user", None),
    ("/product-analytics/product/" + PRODUCT + "/count", None),
    ("/notifications", None),
    ("/notifications/" + NOTIFICATION, None),
    ("/user-notifications", None),
    ("/user-notifications/" + USER_NOTIFICATION, None),
    ("/user-notifications/me", "user_notification_me_shadow"),
    ("/user-notifications/count/me", None),
    ("/user-notifications/notification/" + NOTIFICATION, "go_error_envelope"),
    ("/user-notifications/notification/" + NOTIFICATION + "/count", None),
]:
    verify("GET " + path, "GET", path, approved_difference=difference)


# Mutation contracts use stable seeded rows. New IDs are not compared after
# creates; source and destination generate different UUIDs by design.
for method, path, payload in [
    ("PATCH", "/reviews/" + REVIEW, {"rating": 3}),
    ("DELETE", "/reviews/" + REVIEW, None),
    ("PATCH", "/notifications/" + NOTIFICATION, {"message": "Ready to ship"}),
    ("PATCH", "/user-notifications/" + USER_NOTIFICATION, {}),
    ("DELETE", "/notifications/" + DELETE_NOTIFICATION, None),
    ("POST", "/reviews", {"userId": CUSTOMER, "productId": CLASSIC, "rating": 4, "comment": "Yum"}),
    ("POST", "/product-analytics", {"userId": CUSTOMER, "productId": PRODUCT}),
    ("POST", "/notifications", {"title": "Update", "message": "Fresh batch available"}),
    ("POST", "/user-notifications", {"userId": OTHER, "notificationId": NOTIFICATION, "isRead": False}),
]:
    verify(method + " " + path, method, path, payload)


print(f"ROUTES={tested}; MATCH={match}; APPROVED_CHANGES={approved}; UNEXPECTED={failed}")
if tested != 28 or failed:
    raise SystemExit(1)
print("REMAINING DOMAIN PARITY AUDIT PASSED", flush=True)
