import base64, hashlib, hmac, json, os, sys, time, urllib.error, urllib.request

subject = "22222222-2222-4222-8222-222222222222"
role = "11111111-1111-4111-8111-111111111111"
secret = b"compare-access-secret"
def b64(v):
    return base64.urlsafe_b64encode(v).decode().rstrip("=")
head = b64(json.dumps({"alg":"HS256","typ":"JWT"},separators=(",",":")).encode())
payload = b64(json.dumps({"sub":subject,"role":role,"exp":int(time.time())+900,"iat":int(time.time())},separators=(",",":")).encode())
sig = b64(hmac.new(secret,(head+"."+payload).encode(),hashlib.sha256).digest())
token = ".".join((head,payload,sig))
cases = [
    "/discounts",
    "/discounts/55555555-5555-4555-8555-555555555555",
    "/product-discounts",
    "/product-discounts/66666666-6666-4666-8666-666666666666",
    "/product-discounts/product/44444444-4444-4444-8444-444444444444",
    "/vouchers",
    "/vouchers/77777777-7777-4777-8777-777777777777",
    "/user-vouchers",
    "/user-vouchers/88888888-8888-4888-8888-888888888888",
    "/user-vouchers/voucher/77777777-7777-4777-8777-777777777777",
    "/user-vouchers/user",
]
NEST_PORT = int(os.environ.get("NEST_PORT", "4081"))
GO_PORT = int(os.environ.get("GO_PORT", "4082"))

def request(port,path):
    req=urllib.request.Request(f"http://127.0.0.1:{port}{path}",headers={"Authorization":"Bearer "+token})
    try:
        with urllib.request.urlopen(req,timeout=7) as res:
            status=res.status
            data=res.read()
    except urllib.error.HTTPError as e:
        status=e.code;data=e.read()
    try:
        return status,json.loads(data)
    except Exception:
        return status,data.decode(errors="replace")
failures=0
exact=0
intentional=0
for path in cases:
    nest=request(NEST_PORT,path);go=request(GO_PORT,path)
    if path == "/user-vouchers/user":
        # Original Nest controller registers /:id before /user; the literal
        # "user" gets handled as an invalid UUID. Go intentionally fixes it.
        source_bug = (nest[0] == 400 and isinstance(nest[1],dict) and
                      nest[1].get("message") == "Validation failed (uuid is expected)")
        fixed = (go[0] == 200 and isinstance(go[1],dict) and
                 isinstance(go[1].get("data"),list) and len(go[1]["data"]) == 1 and
                 isinstance(go[1]["data"][0].get("vouchers"),dict))
        if source_bug and fixed:
            intentional += 1
            print(f"INTENTIONAL FIX {path}: Nest=400 broken route, Go=200 voucher list")
        else:
            failures += 1
            print(f"UNEXPECTED {path}: Nest={nest} Go={go}")
    elif nest==go:
        exact += 1
        print(f"PASS {path}: Nest={nest[0]} Go={go[0]}")
    else:
        failures += 1
        print(f"DIFF {path}: Nest={nest[0]} Go={go[0]}")
        print(" Nest:",json.dumps(nest[1],ensure_ascii=False,sort_keys=True)[:1400])
        print(" Go:  ",json.dumps(go[1],ensure_ascii=False,sort_keys=True)[:1400])
print(f"GET cases: {len(cases)}; exact: {exact}; intentional fixes: {intentional}; unexpected: {failures}")

def mutate(port, method, path, payload=None):
    data=json.dumps(payload).encode() if payload is not None else None
    headers={"Content-Type":"application/json"}
    req=urllib.request.Request(f"http://127.0.0.1:{port}{path}", method=method, data=data, headers=headers)
    try:
        with urllib.request.urlopen(req,timeout=7) as res:
            status=res.status;body=res.read()
    except urllib.error.HTTPError as e:
        status=e.code;body=e.read()
    try:
        return status,json.loads(body)
    except Exception:
        return status,body.decode(errors="replace")

mutations=[
 ("PATCH","/discounts/55555555-5555-4555-8555-555555555555",{"maxApplied":7}),
 ("PATCH","/vouchers/77777777-7777-4777-8777-777777777777",{"minimumPurchase":40000}),
 ("PATCH","/user-vouchers/88888888-8888-4888-8888-888888888888",None),
 ("DELETE","/product-discounts/66666666-6666-4666-8666-666666666666",None),
 ("DELETE","/vouchers/77777777-7777-4777-8777-777777777777",None),
 ("POST","/discounts",{"name":"New Sale","discountType":"FIXED","value":12,"maxApplied":1,"maxRedeemed":2,"redeemedCount":0,"description":None,"productId":None,"quantity":None,"startDate":"2026-01-01","endDate":"2027-01-01"}),
 ("POST","/vouchers",{"code":"SPRING15","name":"Spring Sale","description":None,"type":"FIXED","group":"CHECKOUT","value":15,"productId":None,"quantity":None,"minimumPurchase":100,"isActive":True,"startDate":"2026-01-01","endDate":"2027-01-01","maxRedeemed":50,"redeemedCount":0}),
 ("POST","/user-vouchers",{"userId":subject,"voucherId":"77777777-7777-4777-8777-777777777777","claimedAt":"2026-01-01","expiredAt":"2027-01-01"}),
]
for method,path,body in mutations:
    nest=mutate(NEST_PORT,method,path,body)
    go=mutate(GO_PORT,method,path,body)
    same=nest==go
    print(f"{'PASS' if same else 'DIFF'} {method} {path}: Nest={nest[0]} Go={go[0]}")
    if same:
        exact+=1
    else:
        failures+=1
        print(" Nest:",json.dumps(nest[1],ensure_ascii=False,sort_keys=True)[:1300])
        print(" Go:  ",json.dumps(go[1],ensure_ascii=False,sort_keys=True)[:1300])

print(f"COMPARED {len(cases)} GET + {len(mutations)} mutation cases")
print(f"EXACT={exact}; INTENTIONAL_ROUTE_FIX={intentional}; UNEXPECTED_DIFFERENCES={failures}")
# 18 exact + 1 deliberate improvement. This is not a claim of strict 19/19 parity.
if failures or exact != 18 or intentional != 1:
    print("PARITY AUDIT FAILED: unexpected deviation or changed source route behavior")
    sys.exit(1)
print("PARITY AUDIT PASSED: 18 exact; 1 documented intentional NestJS routing bug fix")
sys.exit(0)
