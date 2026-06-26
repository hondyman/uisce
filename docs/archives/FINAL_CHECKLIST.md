# ✅ Final Checklist - System Ready

## Pre-Startup Verification

- [x] Backend build tag added (`// +build ignore`)
- [x] Docker Compose services configured
- [x] Frontend environment variables updated
- [x] All 8001 hardcoded references removed
- [x] Environment-driven configuration implemented

---

## Startup Sequence

### Phase 1: Docker Services (5 minutes)

```bash
cd /Users/eganpj/GitHub/semlayer
docker compose -f docker-compose.backend.yml up -d
```

**Verify**:
```bash
docker compose -f docker-compose.backend.yml ps
# Should show 3 containers in "Up" status:
# - semlayer-rabbitmq (Up, healthy)
# - semlayer-event-router (Up, healthy)  
# - semlayer-graphql-engine (Up, healthy)
```

✅ **Status**: [  ] Docker services ready

---

### Phase 2: Backend (2 minutes)

```bash
cd /Users/eganpj/GitHub/semlayer
PORT=29080 go run ./cmd/server
```

**Verify**:
```bash
curl -s 'http://localhost:29080/api/entity_registry?tenant_id=910638ba-a459-4a3f-bb2d-78391b0595f6&datasource_id=982aef38-418f-46dc-acd0-35fe8f3b97b0' | head -c 100
# Should return JSON starting with: {"entity_registry":[...
```

✅ **Status**: [  ] Backend running

---

### Phase 3: Frontend (3 minutes)

```bash
cd /Users/eganpj/GitHub/semlayer/frontend
npm run dev
```

**Verify**:
```bash
curl -s http://localhost:5173 | head -c 50
# Should return HTML starting with: <!doctype html>
```

✅ **Status**: [  ] Frontend running

---

## Post-Startup Verification

### Console Check (Open http://localhost:5173, press F12)

- [ ] No errors about `localhost:8001`
- [ ] Message shows: `[apollo] graphqlEndpoint = http://localhost:8080/v1/graphql`
- [ ] No warnings in console
- [ ] No network errors shown

### Network Tab Check (F12 → Network, refresh page)

- [ ] GraphQL requests go to `http://localhost:8080/v1/graphql`
- [ ] REST API calls go to `http://localhost:29080/api/*`
- [ ] All requests show status 200 or 204 (success)
- [ ] No failed requests to port 8001

### Functionality Check

- [ ] Can select a tenant from UI dropdown
- [ ] Can see entity registry data
- [ ] Can navigate between pages without errors
- [ ] Real-time updates working (if applicable)

---

## Port Verification

```bash
# Run this quick check:
echo "Frontend:" && curl -s http://localhost:5173 | head -c 50 && echo ""
echo "Backend:" && curl -s 'http://localhost:29080/api/entity_registry?tenant_id=910638ba-a459-4a3f-bb2d-78391b0595f6&datasource_id=982aef38-418f-46dc-acd0-35fe8f3b97b0' | head -c 50 && echo ""
echo "GraphQL:" && curl -s http://localhost:8080/healthz && echo ""
echo "RabbitMQ:" && curl -s -u guest:guest http://localhost:15673/api/whoami | jq . && echo ""
```

| Port | Service | Status |
|------|---------|--------|
| 5173 | Frontend | [  ] ✅ |
| 29080 | Backend REST | [  ] ✅ |
| 8080 | GraphQL | [  ] ✅ |
| 5673 | RabbitMQ | [  ] ✅ |

---

## Configuration Verification

### .env File Check

```bash
cat /Users/eganpj/GitHub/semlayer/frontend/.env
```

**Must contain**:
- [ ] `VITE_API_BASE_URL=http://localhost:29080`
- [ ] `VITE_GRAPHQL_ENDPOINT=http://localhost:8080/v1/graphql`
- [ ] `VITE_GRAPHQL_WS_ENDPOINT=ws://localhost:8080/v1/graphql`
- [ ] `VITE_BACKEND_TARGET=http://localhost:29080`

### Code Verification

```bash
# Check no 8001 references remain
grep -r "localhost:8001" /Users/eganpj/GitHub/semlayer/frontend/src
# Should return: (empty - no matches)
```

- [ ] Zero hardcoded 8001 references
- [ ] All files use environment variables

---

## Build Verification

### Backend Build Check

```bash
cd /Users/eganpj/GitHub/semlayer/backend
go build -o /tmp/test ./cmd/server
```

- [ ] No compilation errors
- [ ] Binary built successfully
- [ ] No undefined symbol errors

### Frontend Build Check

```bash
cd /Users/eganpj/GitHub/semlayer/frontend
npm run build
```

- [ ] No build warnings about ports
- [ ] Build completes successfully
- [ ] Dist folder created

---

## Docker Services Detailed Check

```bash
# Check all containers
docker compose -f docker-compose.backend.yml ps -a

# Check logs for errors
docker compose -f docker-compose.backend.yml logs --tail=20
```

### RabbitMQ
- [ ] Container running
- [ ] Healthcheck passing
- [ ] Accessible at localhost:5673

### Hasura GraphQL
- [ ] Container running
- [ ] Healthcheck passing
- [ ] Accessible at http://localhost:8080
- [ ] Console loads at http://localhost:8080/console

### Event Router
- [ ] Container running
- [ ] Healthcheck passing
- [ ] Accessible at localhost:8081

---

## Application Features Check

Once frontend is loaded:

- [ ] Page loads without errors
- [ ] Tenant selector visible
- [ ] Can select a tenant
- [ ] Entity registry displays
- [ ] No red error banners
- [ ] All buttons clickable
- [ ] Search functionality works
- [ ] Pagination works (if applicable)

---

## Troubleshooting Checklist

If you encounter issues:

### Issue: "Port already in use"
- [ ] Run `lsof -i :5173 -i :29080 -i :8080` to find process
- [ ] Kill old process: `pkill -f "npm run\|go run"`
- [ ] Restart services

### Issue: "Cannot connect to database"
- [ ] Verify PostgreSQL running: `psql postgres://postgres:postgres@localhost:5432/alpha`
- [ ] Check connection string in backend
- [ ] Verify database `alpha` exists

### Issue: "Hasura segfault"
- [ ] Current fix: Using `hasura/graphql-engine:latest`
- [ ] Check image: `docker images | grep hasura`
- [ ] If old version, pull latest: `docker pull hasura/graphql-engine:latest`

### Issue: "Frontend shows 8001 errors"
- [ ] Clear browser cache: Cmd+Shift+R (Mac)
- [ ] Clear Vite cache: `rm -rf node_modules/.vite`
- [ ] Restart npm: `npm run dev`
- [ ] Check `.env` was updated: `cat frontend/.env | grep 29080`

### Issue: "Backend build fails"
- [ ] Check build tag: `head -5 backend/cmd/server/main_integration_example.go`
- [ ] Should start with: `// +build ignore`
- [ ] Clear cache: `go clean -cache`
- [ ] Try again: `PORT=29080 go run ./cmd/server`

---

## Final Checklist

System Ready When All Checked:

✅ **Build System**
- [ ] Backend compiles without errors
- [ ] Frontend builds without errors
- [ ] Docker images build successfully

✅ **Services Running**
- [ ] Frontend on :5173 ✅
- [ ] Backend on :29080 ✅
- [ ] Hasura on :8080 ✅
- [ ] RabbitMQ on :5673 ✅
- [ ] Event Router on :8081 ✅

✅ **Configuration**
- [ ] .env has correct ports
- [ ] No hardcoded 8001 in code
- [ ] Environment variables being used
- [ ] Fallbacks set to 29080 and 8080

✅ **Functionality**
- [ ] Frontend loads without errors
- [ ] No console errors about 8001
- [ ] API requests successful (200 status)
- [ ] GraphQL endpoint responsive
- [ ] Tenant scoping active

✅ **Documentation**
- [ ] Read SYSTEM_FULLY_OPERATIONAL.md
- [ ] Read QUICK_START.md
- [ ] Read CONSOLE_ERROR_ANALYSIS.md
- [ ] Understand architecture

---

## One-Command Verification

```bash
#!/bin/bash

echo "=== System Status Check ==="
echo ""

echo "Frontend (5173):"
curl -s http://localhost:5173 | head -c 30 && echo "✅" || echo "❌"

echo "Backend (29080):"
curl -s 'http://localhost:29080/api/entity_registry?tenant_id=910638ba-a459-4a3f-bb2d-78391b0595f6&datasource_id=982aef38-418f-46dc-acd0-35fe8f3b97b0' | head -c 50 && echo "✅" || echo "❌"

echo "GraphQL (8080):"
curl -s http://localhost:8080/healthz && echo "✅" || echo "❌"

echo "RabbitMQ (5673):"
docker compose -f /Users/eganpj/GitHub/semlayer/docker-compose.backend.yml ps | grep rabbitmq | grep -q "Up" && echo "✅" || echo "❌"

echo ""
echo "=== Code Check ==="
echo "Hardcoded 8001 references:"
grep -r "localhost:8001" /Users/eganpj/GitHub/semlayer/frontend/src 2>/dev/null | wc -l | xargs echo "Count:"
echo "Should be: 0"

echo ""
echo "=== All Checks Complete ==="
```

Save this as `check-system.sh` and run:
```bash
chmod +x check-system.sh
./check-system.sh
```

---

## Quick Reference

| Need | Command |
|------|---------|
| Start all | `docker compose up -d && PORT=29080 go run ./cmd/server & cd frontend && npm run dev` |
| Stop all | `docker compose down && pkill -f "go run\|npm run"` |
| View logs | `docker compose logs -f` |
| Clean rebuild | `rm -rf node_modules/.vite && npm run dev` |
| Test API | `curl -s http://localhost:29080/api/entity_registry?tenant_id=XXX&datasource_id=YYY` |
| Test GraphQL | `curl -s http://localhost:8080/healthz` |
| Check ports | `lsof -i :5173 -i :29080 -i :8080 -i :5673 -i :8081` |

---

## Success Indicators

✅ **You'll know it's working when**:

1. Browser console shows NO `localhost:8001` errors
2. Console shows `[apollo] graphqlEndpoint = http://localhost:8080/v1/graphql`
3. Network tab shows API calls to `localhost:29080`
4. Network tab shows GraphQL calls to `localhost:8080`
5. All requests return status 200 or 204
6. Data displays in the UI without errors
7. Page navigation works smoothly
8. No red error banners appear

---

## Documentation Files Created

| File | Purpose |
|------|---------|
| `SYSTEM_FULLY_OPERATIONAL.md` | Complete system overview |
| `QUICK_START.md` | Quick reference for starting services |
| `FRONTEND_PORT_FIX.md` | Detailed frontend changes |
| `CONSOLE_ERROR_ANALYSIS.md` | Root cause analysis & explanation |
| `FIX_SUMMARY.md` | Summary of all changes |
| `SOLUTION_COMPLETE.md` | This solution overview |
| `FINAL_CHECKLIST.md` | This file - detailed verification steps |

---

## Support & Next Steps

**If all checks pass**: ✅ System is ready for development

**If issues remain**:
1. Review the troubleshooting section above
2. Check CONSOLE_ERROR_ANALYSIS.md for explanations
3. Verify each port with the Port Verification table
4. Run the One-Command Verification script

**For production**:
1. Review SYSTEM_FULLY_OPERATIONAL.md architecture section
2. Update .env for production URLs
3. Run `npm run build` for frontend
4. Use Docker build for backend if needed

---

**Last Updated**: October 19, 2025  
**Status**: ✅ **ALL SYSTEMS GO**

**Green Light**: You're ready to start the services and begin development! 🚀
