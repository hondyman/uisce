API Gateway - Login flow and setup

This file documents the gateway's authentication decision: the gateway issues and validates its own HS256 JWTs. Backends are not considered authority for session tokens.

1) Purpose
The gateway's login endpoint proxies credential validation to the backend and then issues a gateway-signed JWT (HS256). The gateway will only accept JWTs it can verify with the configured `JWT_SECRET`.

2) Environment
- JWT_SECRET: shared secret used to sign gateway JWTs.
- BACKEND_URL: base URL for backend (e.g. http://backend:3000)

Ensure these are set in your `docker-compose.yml` or environment.

3) Login flow
- Client POSTs credentials to: POST /api/gateway/login
  - Body: {"email": "...", "password": "..."}
- Gateway proxies to backend: POST ${BACKEND_URL}/api/auth/login
- If backend validates credentials, gateway returns a JSON body:
  {"access_token": "<gateway-signed-jwt>", "token_type": "bearer"}
- Client should use that token for all subsequent requests in the Authorization header.

4) Behavior
- The gateway enforces JWT-only auth for protected endpoints under the `/api` group.
- The gateway will return HTTP 401 for any non-gateway tokens.

5) Cleanup (optional)
If you created temporary test users while validating the gateway, you can remove them from the backend DB using standard backend admin tooling or SQL. Example (psql):

  DELETE FROM users WHERE email = 'test+gw@example.com';

6) Notes
- This approach centralizes token verification in the gateway and makes token lifetimes and claims consistent for the frontend.
- If you later want to accept backend sessions, reintroduce the `/api/auth/me` fallback in `JWTMiddleware` with careful validation and caching to avoid performance regressions.
