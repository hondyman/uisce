# Frontend Dev Setup (quick)

This file shows the minimal commands and environment variables to run the frontend with the local dev-proxy and backend so `/api/*` requests from the Vite dev server are forwarded correctly.

Summary
- Vite (frontend) runs on port 5173 by default.
- The dev-proxy (frontend/dev-tools/dev-proxy.cjs) listens on port 5175 by default and forwards `/api` to `API_TARGET` (default `http://localhost:8001`).
- The backend (local dev) often listens on 9090 and the API gateway (docker) on 8001; start whichever you use and point dev-proxy to it.

Recommended start order

1) Start the backend and API gateway (preferred via the repo script which starts Docker services):

```bash
# from repo root
./scripts/start-services.sh
```

This starts Docker services and also starts the dev-proxy and frontend for you. If you prefer to run services manually, use the commands below.

2) Start dev-proxy (manual)

If you run the backend locally on 9090 (direct), point the dev-proxy at it. If you run the API gateway in Docker (8001), use that instead.

```bash
# Run dev-proxy and forward /api -> http://localhost:9090 (or 8001 for gateway)
API_TARGET=http://localhost:9090 PORT=5175 node frontend/dev-tools/dev-proxy.cjs
```

3) Start the frontend (Vite)

```bash
cd frontend
npm run dev
```

Notes about environment variables
- `API_TARGET` (used by the dev-proxy) — where `/api` requests should be forwarded. Default: `http://localhost:8001`.
- `PORT` (used by dev-proxy) — dev-proxy listen port. Default: `5175`.
- `DEV_PROXY_TARGET` (used by Vite config) — if you need Vite's internal proxy to point somewhere other than `http://localhost:5175`, set this before starting Vite.

Quick smoke tests (use these to verify everything is wired)

1. Dev-proxy health (direct):

```bash
curl -sS http://localhost:5175/_debug | jq
# expected: { "status": "working", "timestamp": "...", "target": "http://localhost:8001" }
```

2. Vite -> dev-proxy -> backend (BundleEditor endpoints):

```bash
# This should return JSON or an error from the backend/gateway, not a Vite 404
curl -i -m 5 http://localhost:5173/api/semantic/objects

# Example bundle fetch
curl -i -m 5 http://localhost:5173/api/bundles/lp_private_markets_bundle
```

If the Vite request times out or returns a 404:
- Confirm dev-proxy is running and `API_TARGET` points to a service that actually exposes the requested path.
- If your backend exposes `/api/semantic/objects` directly on port 9090, you can set `API_TARGET=http://localhost:9090` when starting dev-proxy.
- If you use the API gateway on 8001, ensure the gateway is configured to route `/api/semantic/objects` to the backend; otherwise point `API_TARGET` directly at the backend.

Common quick fixes
- If you get a Vite 404 for `/api/*`, start the dev-proxy and/or restart Vite after dev-proxy is running so the Vite proxy is active.
- If Vite connects but the request times out, the dev-proxy forwarded the request but the upstream (API_TARGET) is not responding; check that backend/gateway is running and reachable.

If you want, I can also:
- Add these lines to the top-level README, or
- Change the Vite/dev-proxy defaults to point at `http://localhost:9090` to match local backend dev behavior (safe change if you run backend locally), or
- Run curl tests right now and paste the exact responses.
