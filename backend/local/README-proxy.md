Local Proxy (developer)
=======================

This small proxy helps local development by exposing a single HTTP port
(`:29080` by default) and forwarding requests to the appropriate local
services. It centralizes routes so the frontend can use a single
`VITE_API_BASE_URL` (`http://localhost:29080`).

Default routing
- `/api/validation-rules` -> rule-engine service (default `http://localhost:8083`)
- `/v1/graphql` -> Hasura (default `http://localhost:8080`)
- `/api/*` -> backend API gateway (default `http://localhost:8080`)
- `/health` -> proxy health

Configuration
-------------
Override targets using environment variables or flags:

- `PROXY_LISTEN` (default `:29080`)
- `RULE_ENGINE_URL` (default `http://localhost:8083`)
- `BACKEND_URL` (default `http://localhost:8080`)
- `HASURA_URL` (default `http://localhost:8080`)

Run (dev)
---------
From the repo root you can start the proxy with the provided Make target:

```bash
make start-proxy
```

Or run directly (override targets inline):

```bash
PROXY_LISTEN=:29080 RULE_ENGINE_URL=http://localhost:8083 BACKEND_URL=http://localhost:8080 HASURA_URL=http://localhost:8080 go run ./backend/local/cmd/proxy
```

Helper scripts
--------------
There are simple scripts in `backend/local/scripts/` to start/stop the proxy
in the background and write a PID file.

Notes
-----
- The proxy sets the upstream `Host` header and URL host so services that
  rely on Host-based routing or require WebSocket upgrades will behave
  correctly.
- The proxy returns `502` if an upstream service is unreachable.
- For a fully integrated dev experience, start any required backends (Hasura,
  rule-engine) or run the project's docker-compose setup and then start the
  proxy.
