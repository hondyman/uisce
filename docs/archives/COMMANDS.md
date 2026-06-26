# 🚀 COMMAND REFERENCE - Copy & Paste Ready

## ✅ The Fix Is Complete

All console errors (`localhost:8001`) have been fixed.  
All services are configured and ready.

---

## Start Everything (3 Commands)

### Command 1: Docker Services
```bash
cd /Users/eganpj/GitHub/semlayer && docker compose -f docker-compose.backend.yml up -d
```
**What it does**: Starts RabbitMQ, Hasura, Event Router  
**Wait for**: All 3 containers in "Up" status  
**Check with**: `docker compose -f docker-compose.backend.yml ps`

### Command 2: Backend (New Terminal)
```bash
cd /Users/eganpj/GitHub/semlayer && PORT=29080 go run ./backend/cmd/server
```
**What it does**: Starts API server on port 29080  
**Wait for**: "Server starting on http://localhost:29080"  
**Check with**: `curl http://localhost:29080/health`

### Command 3: Frontend (New Terminal)
```bash
cd /Users/eganpj/GitHub/semlayer/frontend && npm run dev
```
**What it does**: Starts Vite dev server on port 5173  
**Wait for**: "Local:   http://localhost:5173/"  
**Check with**: `curl http://localhost:5173`

### Then: Open Browser
```bash
open http://localhost:5173
```

---

## Verify Everything Works

### Quick Check (Run in Terminal 1)
```bash
echo "Frontend:" && curl -s http://localhost:5173 | head -c 30 && echo " ✅"
echo "Backend:" && curl -s 'http://localhost:29080/api/entity_registry?tenant_id=910638ba-a459-4a3f-bb2d-78391b0595f6&datasource_id=982aef38-418f-46dc-acd0-35fe8f3b97b0' | head -c 50 && echo " ✅"
echo "GraphQL:" && curl -s http://localhost:8080/healthz | head -c 50 && echo " ✅"
```

### Console Check (F12 in Browser)
```
✅ Should see: [apollo] graphqlEndpoint = http://localhost:8080/v1/graphql
✅ Should NOT see: localhost:8001 anywhere
✅ Should NOT see: net::ERR_CONNECTION_REFUSED
```

### Network Tab Check (F12 → Network, refresh page)
```
✅ API calls to: http://localhost:29080/api/*
✅ GraphQL calls to: http://localhost:8080/v1/graphql
✅ All responses: Status 200 or 204
```

---

## Troubleshooting Commands

### Backend won't start ("Port already in use")
```bash
# Kill old process
pkill -f "go run ./backend/cmd/server"

# Start fresh
cd /Users/eganpj/GitHub/semlayer && PORT=29080 go run ./backend/cmd/server
```

### Frontend won't start
```bash
# Clear cache
cd /Users/eganpj/GitHub/semlayer/frontend && rm -rf node_modules/.vite

# Start fresh
npm run dev
```

### Docker services not running
```bash
# Check status
docker compose -f docker-compose.backend.yml ps

# Restart everything
docker compose -f docker-compose.backend.yml restart

# Or full rebuild
docker compose -f docker-compose.backend.yml down -v && docker compose -f docker-compose.backend.yml up -d
```

### Can't connect to database
```bash
# Check PostgreSQL is running
psql postgres://postgres:postgres@localhost:5432/alpha

# If fails, start PostgreSQL:
brew services start postgresql@15
```

### API returns 404
```bash
# Verify backend is running
curl http://localhost:29080/health

# Check with tenant IDs
curl 'http://localhost:29080/api/entity_registry?tenant_id=910638ba-a459-4a3f-bb2d-78391b0595f6&datasource_id=982aef38-418f-46dc-acd0-35fe8f3b97b0'
```

---

## Stop Everything

```bash
# Stop frontend (Ctrl+C in frontend terminal)
# Stop backend (Ctrl+C in backend terminal)

# Stop Docker services
docker compose -f docker-compose.backend.yml down

# Kill any lingering processes
pkill -f "go run ./backend/cmd/server"
pkill -f "npm run dev"
```

---

## Port Reference

| Port | Service | Check Command |
|------|---------|---------------|
| 5173 | Frontend | `curl http://localhost:5173` |
| 29080 | Backend | `curl http://localhost:29080/health` |
| 8080 | GraphQL | `curl http://localhost:8080/healthz` |
| 5673 | RabbitMQ | `docker compose -f docker-compose.backend.yml ps \| grep rabbitmq` |
| 15673 | RabbitMQ Mgmt | `open http://localhost:15673` (user: guest, pass: guest) |
| 8081 | Event Router | `curl http://localhost:8081/health` |
| 5432 | PostgreSQL | `psql postgres://postgres:postgres@localhost:5432/alpha` |

---

## One-Liner to Start Everything

```bash
cd /Users/eganpj/GitHub/semlayer && docker compose -f docker-compose.backend.yml up -d && (PORT=29080 go run ./backend/cmd/server &) && sleep 5 && (cd frontend && npm run dev)
```

(Then open http://localhost:5173 in browser)

---

## Important Paths

```
Backend code:     ./backend/cmd/server
Frontend code:    ./frontend/src
Docker config:    ./docker-compose.backend.yml
Frontend config:  ./frontend/.env
Backend logs:     /tmp/backend.log (if running in background)
```

---

## Key Environment Variables

These are in `frontend/.env`:

```
VITE_API_BASE_URL=http://localhost:29080
VITE_GRAPHQL_ENDPOINT=http://localhost:8080/v1/graphql
VITE_GRAPHQL_WS_ENDPOINT=ws://localhost:8080/v1/graphql
VITE_BACKEND_TARGET=http://localhost:29080
```

---

## Testing API Directly

```bash
# Get entity registry
curl 'http://localhost:29080/api/entity_registry?tenant_id=910638ba-a459-4a3f-bb2d-78391b0595f6&datasource_id=982aef38-418f-46dc-acd0-35fe8f3b97b0'

# List bundles
curl 'http://localhost:29080/api/bundles?tenant_id=910638ba-a459-4a3f-bb2d-78391b0595f6&datasource_id=982aef38-418f-46dc-acd0-35fe8f3b97b0'

# Test GraphQL
curl -X POST 'http://localhost:8080/v1/graphql' \
  -H 'Content-Type: application/json' \
  -d '{"query":"{ __schema { types { name } } }"}'
```

---

## Development Workflow

### During Development
1. **Make code changes** in your editor
2. **Frontend auto-reloads** (Vite hot module replacement)
3. **Backend requires restart** (Ctrl+C, then re-run)
4. **Check browser console** for any errors
5. **Check Network tab** for API responses

### Typical Session
```bash
# Terminal 1: Leave Docker running
docker compose -f docker-compose.backend.yml up -d

# Terminal 2: Backend (restart when code changes)
cd /Users/eganpj/GitHub/semlayer
PORT=29080 go run ./backend/cmd/server
# ... edit code ...
# (Ctrl+C to stop, run again)

# Terminal 3: Frontend (auto-reloads)
cd /Users/eganpj/GitHub/semlayer/frontend
npm run dev
# ... keep running, hot reload works automatically ...
```

---

## All 8001 References: REMOVED ✅

```bash
# Verify no 8001 in codebase
grep -r "localhost:8001" /Users/eganpj/GitHub/semlayer/frontend/src
# Should return: (nothing - no matches)
```

---

## Success Checklist

When everything is working:

- [ ] Backend running: `curl http://localhost:29080/health` returns 200
- [ ] Frontend running: `curl http://localhost:5173` returns HTML
- [ ] GraphQL running: `curl http://localhost:8080/healthz` returns OK
- [ ] RabbitMQ running: `docker compose ... ps | grep rabbitmq` shows "Up"
- [ ] Browser opens: http://localhost:5173 loads without errors
- [ ] Console shows: `[apollo] graphqlEndpoint = http://localhost:8080/v1/graphql`
- [ ] Console shows: NO errors about localhost:8001
- [ ] Network tab shows: All requests to 29080 and 8080 returning 200

---

## Fastest Way to Restart After Crash

```bash
# 1. Kill everything
pkill -f "go run.*backend"
pkill -f "npm run dev"

# 2. Start Docker (if not running)
docker compose -f docker-compose.backend.yml up -d

# 3. Start backend
cd /Users/eganpj/GitHub/semlayer && PORT=29080 go run ./backend/cmd/server &

# 4. Start frontend
cd /Users/eganpj/GitHub/semlayer/frontend && npm run dev
```

---

**Ready to go!** 🚀

Use these commands and everything will work. The console error is fixed, all services are configured correctly.

Last Updated: October 19, 2025
