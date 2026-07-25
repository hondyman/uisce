JWKS & Token Revocation Demo (api-gateway)

Overview

This document explains how to run the api-gateway locally with the JWKS endpoint (RS256) and token revocation store (Redis). The gateway supports:

- RS256 signing with an in-process KeyManager (rotate keys programmatically).
- JWKS endpoint at `GET /jwks.json` which exposes public RSA keys and their `kid`.
- Optional Redis-backed revocation store (set `REVOCATION_REDIS_ADDR`) or an in-memory store used by default for local dev.
- Admin revoke endpoint: `POST /api/tokens/revoke` with payload `{ "jti": "<jti>", "exp": <unix_ts_optional> }`.

Quick start (local, uses Docker Compose)

1. Start Redis (and optional backend/hasura) using docker-compose from the `api-gateway` directory:

```bash
# from api-gateway/
docker compose up -d --build
```

2. Run the gateway (in dev you can run locally):

```bash
# Enable RS256 tokens and point revocation at Redis inside the compose network
export ENABLE_RS256=true
export REVOCATION_REDIS_ADDR=redis:6379
# Build and run (from repo root or api-gateway dir)
cd api-gateway
go run .
```

3. Demo flow (simple):

- Fetch JWKS:

```bash
curl http://localhost:8000/jwks.json
```

- Login (gateway proxies to backend, then issues a gateway-signed token). If you run the mock backend from tests or your backend service is running, call:

```bash
curl -X POST http://localhost:8000/api/auth/login -d '{"email":"...","password":"..."}' -H 'Content-Type: application/json'
```

The response includes `access_token` and (when RS256 is enabled) `kid`.

- Call a protected endpoint with the token:

```bash
curl -H "Authorization: Bearer <token>" http://localhost:8000/api/keys
```

- Revoke the token by sending its `jti` to the revoke endpoint:

```bash
curl -X POST http://localhost:8000/api/tokens/revoke -H 'Content-Type: application/json' -d '{"jti":"<jti>", "exp": <unix_ts> }'
```

Notes & next steps

- In production, persist keys securely (KMS/HSM) and implement a key rotation policy where old keys are retired after ensuring no active tokens depend on them.
- The in-memory revocation store in this repo is intended for tests and local development only.
- Consider adding metrics around revocations and rotation events.

Files added/changed

- Updated `main.go` to initialize `KeyManager` and `RevocationStore`, add `GET /jwks.json`, support RS256 token validation and revocation checks, and add `/api/tokens/revoke`.
- Added `redis` service to `docker-compose.yml` for `api-gateway`.
- Added tests: `keymanager_test.go`, `revocation_test.go`.
- Added this `JWKS_REVOCATION_DEMO.md`.

Running tests

```bash
cd api-gateway
go test ./...
```
