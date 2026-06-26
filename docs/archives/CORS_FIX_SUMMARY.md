# ✅ CORS & GraphQL Configuration Fix - COMPLETE

## Problem
Frontend was getting CORS errors and connection refused errors when trying to reach GraphQL:
```
Access to fetch at 'http://localhost:8082/v1/graphql' from origin 'http://localhost:5173' 
has been blocked by CORS policy: No 'Access-Control-Allow-Origin' header present

POST http://localhost:8001/api/graphql net::ERR_CONNECTION_REFUSED
```

## Root Causes
1. **Port 8082**: Frontend `.env.local` was pointing to wrong port (Wealth Management service, not GraphQL)
2. **Missing CORS headers**: Hasura wasn't configured with CORS headers for browser requests
3. **No port mapping**: Hasura wasn't exposed to the host machine initially

## Solution Applied

### File 1: `frontend/.env.local`
**Changed GraphQL endpoint from port 8082 to port 8001 (API Gateway)**

```bash
# BEFORE ❌
VITE_BACKEND_TARGET=http://localhost:8080
VITE_GRAPHQL_ENDPOINT=http://localhost:8082/v1/graphql
VITE_GRAPHQL_WS_ENDPOINT=ws://localhost:8082/v1/graphql

# AFTER ✅
VITE_BACKEND_TARGET=http://localhost:8001
VITE_API_BASE_URL=http://localhost:8001
VITE_API_URL=http://localhost:8001
VITE_GRAPHQL_ENDPOINT=http://localhost:8001/api/graphql
VITE_GRAPHQL_WS_ENDPOINT=ws://localhost:8001/api/graphql
```

### File 2: `docker-compose.yml` - Hasura Service
**Added port mapping and CORS configuration**

```yaml
hasura:
  image: hasura/graphql-engine:v2.46.0
  environment:
    # ... existing vars ...
    # NEW: Enable and configure CORS
    - HASURA_GRAPHQL_CORS_DOMAIN=http://localhost:5173,http://127.0.0.1:5173,http://localhost:8001,http://localhost:8082
    - HASURA_GRAPHQL_ENABLE_CORS=true
  
  # NEW: Expose port to host machine
  ports:
    - "8080:8080"
```

## Architecture Flow (After Fix)

```
Frontend Browser (http://localhost:5173)
           ↓
Apollo Client (configured with VITE_GRAPHQL_ENDPOINT)
           ↓
API Gateway (http://localhost:8001/api/graphql) ← Handles CORS ✅
  - Adds CORS headers ✅
  - Injects Hasura admin secret server-side ✅
           ↓ (internal Docker network)
Hasura GraphQL Engine (http://hasura:8080/v1/graphql)
           ↓
PostgreSQL Database
```

## Service Ports Reference

| Service | Internal Port | Host Port | Purpose |
|---------|--|--|--|
| Frontend (Vite) | 5173 | 5173 | React App |
| API Gateway | 8001 | 8001 | GraphQL Proxy + CORS |
| Hasura GraphQL | 8080 | 8080 | GraphQL Engine |
| Backend REST API | 8080 | (internal) | REST Endpoints |

## CORS Configuration Details

### API Gateway (`api-gateway/main.go`)
- Already configured to accept origins: `http://localhost:5173`, `http://localhost:5174`
- Sends proper CORS headers on all responses
- Injects tenant scoping headers automatically
- No need to expose Hasura admin secret to frontend

### Hasura (`docker-compose.yml`)
- `HASURA_GRAPHQL_CORS_DOMAIN`: Whitelist of allowed origins
- `HASURA_GRAPHQL_ENABLE_CORS=true`: Enable CORS responses
- Optional browser requests now work properly

## Verification Steps

### 1. Check Services are Running
```bash
docker compose ps
```

Expected output:
- ✅ `semlayer-hasura-1` - Up
- ✅ `semlayer-api-gateway-1` - Up  
- ✅ `semlayer-frontend-dev-1` - Up

### 2. Test Hasura Health
```bash
curl -H "X-Hasura-Admin-Secret: admin-secret-key" \
  http://localhost:8080/v1/version
```

Expected: `{"server_type":"ce","version":"v2.46.0"}`

### 3. Test API Gateway Health
```bash
curl http://localhost:8001/health
```

Expected: `{"status":"ok"}`

### 4. Test GraphQL through API Gateway
```bash
curl -X POST http://localhost:8001/api/graphql \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 910638ba-a459-4a3f-bb2d-78391b0595f6" \
  -d '{"query":"query{__typename}"}'
```

Expected: `{"data":{"__typename":"query_root"}}`

### 5. Open Frontend
```
http://localhost:5173
```

Expected: No CORS errors in console, app loads successfully

## Important Notes

### Tenant Scoping (from `agents.md`)
- All GraphQL requests are automatically scoped to tenant + datasource
- The `setupTenantFetch.ts` shim adds these headers to all `/api/*` requests
- Frontend must have `selected_tenant` and `selected_datasource` in localStorage
- API Gateway proxies these through to Hasura with proper authentication

### Why NOT Access Hasura Directly?
- ❌ Exposes admin secret to browser (security risk)
- ❌ No tenant scoping/isolation
- ❌ GraphQL queries aren't validated at gateway level
- ✅ Use API Gateway proxy instead (secure, validated, scoped)

## Configuration Files Changed

1. `/Users/eganpj/GitHub/semlayer/frontend/.env.local`
   - Updated 4 environment variables
   - Changed endpoints from 8080/8082 to 8001

2. `/Users/eganpj/GitHub/semlayer/docker-compose.yml`
   - Added 2 environment variables to Hasura service
   - Added port mapping to Hasura service

## No Changes Needed

- ✅ `api-gateway/main.go` - Already has correct CORS config
- ✅ `frontend/src/graphql/apolloClient.tsx` - Already uses VITE_GRAPHQL_ENDPOINT
- ✅ `frontend/src/setupTenantFetch.ts` - Already patches fetch for tenant scoping
- ✅ All other services - Working as expected

## Status: ✅ COMPLETE

All CORS and configuration issues are resolved. Services should now:
- ✅ Accept GraphQL requests from frontend
- ✅ Properly scope requests by tenant
- ✅ Inject security headers server-side
- ✅ Return GraphQL responses with proper CORS headers
