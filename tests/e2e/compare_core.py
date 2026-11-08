"""Compare the original NestJS foundational read routes with the Go rewrite.

Both APIs run against separate databases loaded from the same fixture. This
complements the promotion and remaining-domain parity suites; no server
response is treated as equivalent merely because both returned HTTP 200.
"""

import base64
import hashlib
import hmac
import json
import os
import time
import urllib.error
import urllib.request

NEST = int(os.environ.get("NEST_PORT", "4081"))
GO = int(os.environ.get("GO_PORT", "4082"))
ROLE = "11111111-1111-4111-8111-111111111111"
ADMIN_ROLE = "99999999-9999-4999-8999-999999999999"
CUSTOMER = "22222222-2222-4222-8222-222222222222"
ADMIN = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
CATEGORY = "33333333-3333-4333-8333-333333333333"
PRODUCT = "44444444-4444-4444-8444-444444444444"
MISSING = "00000000-0000-4000-8000-000000000001"


def token(subject, role):
    def encode(value):
        return base64.urlsafe_b64encode(json.dumps(value, separators=(",", ":")).encode()).decode().rstrip("=")

    head = encode({"alg": "HS256", "typ": "JWT"})
    body = encode({"sub": subject, "role": role, "iat": int(time.time()), "exp": int(time.time()) + 900})
    raw = (head + "." + body).encode()
    signature = base64.urlsafe_b64encode(hmac.new(b"compare-access-secret", raw, hashlib.sha256).digest()).decode().rstrip("=")
    return head + "." + body + "." + signature


TOKENS = {"customer": token(CUSTOMER, ROLE), "admin": token(ADMIN, ADMIN_ROLE)}


def request(port, path, actor=None):
    headers = {}
    if actor:
        headers["Authorization"] = "Bearer " + TOKENS[actor]
    req = urllib.request.Request("http://127.0.0.1:%d%s" % (port, path), headers=headers)
    try:
        with urllib.request.urlopen(req, timeout=10) as response:
            return response.status, json.loads(response.read())
    except urllib.error.HTTPError as exc:
        try:
            return exc.code, json.loads(exc.read())
        except ValueError:
            return exc.code, None


cases = [
    ("/roles", None),
    ("/roles/" + ROLE, None),
    ("/roles/name/Customer", None),
    ("/categories", None),
    ("/categories/" + CATEGORY, None),
    ("/products", None),
    ("/products/catalog", None),
    ("/products/catalog?q=Strawberry", None),
    ("/products/catalog?cat=" + CATEGORY, None),
    ("/products/catalog?q=%25", None),
    ("/products/catalog?q=_", None),
    ("/products/catalog?q=%25_%25", None),
    ("/products/catalog?q=%5C", None),
    ("/product/" + PRODUCT, None),
    ("/product/information/" + PRODUCT, None),
    ("/inventory", None),
    ("/inventory/product/" + PRODUCT, None),
    ("/inventory/" + MISSING, None),
    ("/shipments", "admin"),
    ("/shipments/" + MISSING, "admin"),
    ("/payments", None),
    ("/payments/1", None),
    ("/users", "admin"),
    ("/users/" + CUSTOMER, "customer"),
    ("/addresses", "admin"),
]


def equivalent(nest, go):
    if nest == go:
        return True
    if nest[0] == go[0] == 404 and isinstance(nest[1], dict) and isinstance(go[1], dict):
        # Shared Go error middleware includes the optional HTTP error label.
        return {k: v for k, v in go[1].items() if k != "error"} == nest[1]
    return False


failed = 0
for path, actor in cases:
    nest = request(NEST, path, actor)
    go = request(GO, path, actor)
    if equivalent(nest, go):
        print(f"MATCH GET {path}: {go[0]}", flush=True)
        continue
    failed += 1
    print(f"FAIL GET {path}: Nest={nest[0]} Go={go[0]}", flush=True)
    print(" Nest:", json.dumps(nest[1], ensure_ascii=False, sort_keys=True)[:1200], flush=True)
    print(" Go:  ", json.dumps(go[1], ensure_ascii=False, sort_keys=True)[:1200], flush=True)

print(f"CORE_READS={len(cases)}; UNEXPECTED={failed}", flush=True)
if failed:
    raise SystemExit(1)
print("CORE NEST→GO PARITY PASSED", flush=True)
