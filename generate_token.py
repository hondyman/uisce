import json
import time
import base64
import hmac
import hashlib

def base64url_encode(data):
    if isinstance(data, str):
        data = data.encode('utf-8')
    encoded = base64.urlsafe_b64encode(data).decode('utf-8')
    return encoded.rstrip('=')

header = {
    "alg": "HS256",
    "typ": "JWT"
}

now = int(time.time())
payload = {
    "sub": "36f45238-bac6-4b06-a495-6155c43df552",
    "user_id": "36f45238-bac6-4b06-a495-6155c43df552",
    "email": "test@example.com",
    "role": "global_ops",
    "roles": ["global_ops"],
    "scopes": [],
    "tenant_scope": "single",
    "jti": "generated-for-testing-" + str(now),
    "https://hasura.io/jwt/claims": {
        "x-hasura-allowed-roles": ["user", "global_ops"],
        "x-hasura-default-role": "user",
        "x-hasura-user-id": "36f45238-bac6-4b06-a495-6155c43df552"
    },
    "iat": now,
    "exp": now + 3600
}

secret = "dev-jwt-secret-key-change-in-production"

encoded_header = base64url_encode(json.dumps(header))
encoded_payload = base64url_encode(json.dumps(payload))

signing_input = f"{encoded_header}.{encoded_payload}"
signature = hmac.new(
    secret.encode('utf-8'),
    signing_input.encode('utf-8'),
    hashlib.sha256
).digest()

encoded_signature = base64url_encode(signature)

jwt = f"{signing_input}.{encoded_signature}"
print(jwt)
