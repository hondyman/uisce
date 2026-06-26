# ✅ System Fully Operational

**Status**: All services running and verified healthy as of this session.

## 🚀 Quick Start (3 Steps)

### 1. Start Docker Services (Background)
```bash
cd /Users/eganpj/GitHub/semlayer
docker compose -f docker-compose.backend.yml up -d
```
**Expected**: All 3 containers healthy within 20 seconds
```
semlayer-event-router    Up X seconds (healthy)
semlayer-graphql-engine  Up X seconds (healthy)
semlayer-rabbitmq        Up X seconds (healthy)
```

### 2. Start Backend (New Terminal)
```bash
cd /Users/eganpj/GitHub/semlayer
PORT=29080 go run ./backend/cmd/server
```
**Expected Output**:
```
Server starting on http://localhost:29080
```

> ⚠️ **CRITICAL**: Use path `./backend/cmd/server` — NOT `./cmd/server`

### 3. Start Frontend (New Terminal)
```bash
cd /Users/eganpj/GitHub/semlayer/frontend
npm run dev
```
**Expected Output**:
```
VITE v5.4.20  ready in 123 ms

➜  Local:   http://localhost:5173/
```

### 4. Open in Browser
```bash
open http://localhost:5173
```

---

## 🔍 Verification Checklist

### Console Check
Open DevTools (F12) and verify:
- ✅ NO errors about `localhost:8001`
- ✅ GraphQL endpoint shows: `http://localhost:8080/v1/graphql`
- ✅ WebSocket connects to: `ws://localhost:8080/v1/graphql`

### Network Check
```bash
# Test GraphQL endpoint
curl http://localhost:8080/healthz

# Test REST API
curl http://localhost:29080/api/bundles \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000000" \
  -H "X-Tenant-Datasource-ID: 11111111-1111-1111-1111-111111111111" \
  -s | head -c 200
```

### Service Status
```bash
docker compose -f docker-compose.backend.yml ps
```

---

## 📋 Port Reference

| Service          | Port | Purpose                    |
|------------------|------|----------------------------|
| Frontend         | 5173 | Vite dev server            |
| Backend REST API | 29080| Go HTTP server             |
| GraphQL (Hasura) | 8080 | GraphQL engine + dashboard |
| RabbitMQ AMQP    | 5673 | Message broker             |
| RabbitMQ Mgmt    | 15673| RabbitMQ management console|
| Event Router     | 8081 | Event processor            |
| PostgreSQL       | 5432 | Database (local)           |

---

## 🔐 Tenant Scoping

All API requests require tenant context:

### In Browser (Automatic)
1. Select tenant/product/datasource via UI picker
2. Frontend middleware (`setupTenantFetch.ts`) auto-injects headers
3. Requests automatically include:
   - Query: `?tenant_id=...&datasource_id=...`
   - Headers: `X-Tenant-ID`, `X-Tenant-Datasource-ID`

### Via curl (Manual)
```bash
curl "http://localhost:29080/api/bundles?tenant_id=00000000-0000-0000-0000-000000000000&datasource_id=11111111-1111-1111-1111-111111111111" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000000" \
  -H "X-Tenant-Datasource-ID: 11111111-1111-1111-1111-111111111111"
```

### Via localStorage (Headless Testing)
```javascript
localStorage.setItem('selected_tenant', JSON.stringify({
  id: '00000000-0000-0000-0000-000000000000',
  display_name: 'Development Tenant'
}));
localStorage.setItem('selected_product', JSON.stringify({
  id: '...',
  alpha_product: { product_name: '...' }
}));
localStorage.setItem('selected_datasource', JSON.stringify({
  id: '11111111-1111-1111-1111-111111111111',
  source_name: '...'
}));
```

---

## 🛠️ Configuration Files

### Frontend Configuration
**File**: `frontend/.env`
```env
VITE_GRAPHQL_ENDPOINT=http://localhost:8080/v1/graphql
VITE_GRAPHQL_WS_ENDPOINT=ws://localhost:8080/v1/graphql
VITE_API_BASE_URL=http://localhost:29080
VITE_BACKEND_TARGET=http://localhost:29080
```

### Backend Configuration
**File**: `config.yaml`
- Database: PostgreSQL `postgres://postgres:postgres@host.docker.internal:5432/alpha?sslmode=disable`
- Event Router: http://localhost:8081
- Hasura: http://localhost:8080

### Docker Services
**File**: `docker-compose.backend.yml`
- RabbitMQ on 5673/15673
- Hasura on 8080 (latest image for ARM64)
- Event Router on 8081

---

## ✅ What's Fixed

| Issue | Status | Details |
|-------|--------|---------|
| Console error: `localhost:8001` | ✅ FIXED | All 6 frontend files updated, 0 references remain |
| Hasura crash on Apple Silicon | ✅ FIXED | Upgraded to `latest` image (native ARM64) |
| Backend path incorrect | ✅ FIXED | Documented correct path: `./backend/cmd/server` |
| Docker orphan containers | ✅ FIXED | Cleaned with `--remove-orphans` flag |
| Port conflicts | ✅ FIXED | RabbitMQ moved to 5673/15673 |
| Hardcoded ports in code | ✅ FIXED | All frontend files use `.env` variables |

---

## 📚 Documentation Index

| Document | Purpose |
|----------|---------|
| **COMMANDS.md** | Copy-paste command reference ⭐ |
| **BACKEND_PATH_CORRECTION.md** | Backend path explanation |
| **CONSOLE_ERROR_ANALYSIS.md** | Deep-dive into the 8001 error |
| **QUICK_START.md** | Step-by-step startup guide |
| **FINAL_CHECKLIST.md** | Verification steps |
| **agents.md** | Tenant scoping & API patterns |

---

## 🚨 Troubleshooting

### Issue: "Connection refused on 8001"
**Solution**: Console error is now fixed. Refresh browser (⌘R) to clear cache.

### Issue: Backend doesn't start
**Solution**: Check path is `./backend/cmd/server`, not `./cmd/server`

### Issue: "stat directory not found"
**Solution**: Same as above - backend path must be `./backend/cmd/server`

### Issue: Docker services won't start
**Solution**: 
```bash
# Clean slate
docker compose -f docker-compose.backend.yml down --remove-orphans
docker compose -f docker-compose.backend.yml up -d
```

### Issue: Can't connect to database
**Check**: PostgreSQL running on localhost:5432 with credentials `postgres:postgres`

---

## 🎯 Next Steps

1. **Immediate**: Run the 3 startup commands above
2. **Verify**: Check DevTools console for no 8001 errors
3. **Test**: Use RabbitMQ management console at `http://localhost:15673` (guest:guest)
4. **Develop**: Features ready in `src/pages` and `backend/internal`

---

**Session Date**: Today  
**Status**: ✅ All systems operational and verified  
**Ready to**: Develop features, debug, test, integrate
