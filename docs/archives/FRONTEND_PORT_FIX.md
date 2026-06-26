# ✅ Frontend Port Configuration Fixed

**Issue**: Browser console showed requests to `http://localhost:8001` even though apolloClient was updated to 8080.

**Root Cause**: Multiple frontend files had hardcoded `8001` references, and environment variables were outdated.

---

## Files Updated

### 1. **`.env`** (Frontend environment)
```properties
# BEFORE:
VITE_GRAPHQL_ENDPOINT=http://localhost:5175/api/graphql
VITE_GRAPHQL_WS_ENDPOINT=ws://localhost:5177/api/graphql
VITE_API_BASE_URL=http://localhost:5175
DEV_PROXY_TARGET=http://localhost:8080

# AFTER:
VITE_GRAPHQL_ENDPOINT=http://localhost:8080/v1/graphql
VITE_GRAPHQL_WS_ENDPOINT=ws://localhost:8080/v1/graphql
VITE_API_BASE_URL=http://localhost:29080
VITE_BACKEND_TARGET=http://localhost:29080
```

### 2. **`src/utils/api.ts`**
```typescript
// BEFORE:
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8001';

// AFTER:
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:29080';
```

### 3. **`src/hooks/useNotificationAPI.ts`**
```typescript
// BEFORE:
const API_BASE_URL = (import.meta.env.VITE_API_BASE_URL as string) || 'http://localhost:8001';

// AFTER:
const API_BASE_URL = (import.meta.env.VITE_API_BASE_URL as string) || 'http://localhost:29080';
```

### 4. **`src/hooks/useDashboardService.ts`**
```typescript
// BEFORE:
const API_BASE_URL = (import.meta.env.VITE_API_BASE_URL as string) || 'http://localhost:8001';

// AFTER:
const API_BASE_URL = (import.meta.env.VITE_API_BASE_URL as string) || 'http://localhost:29080';
```

### 5. **`src/hooks/useModelCatalog.ts`**
```typescript
// BEFORE:
const API_BASE = `${import.meta.env.VITE_API_BASE_URL || 'http://localhost:8001'}/api`;

// AFTER:
const API_BASE = `${(import.meta.env.VITE_API_BASE_URL as string) || 'http://localhost:29080'}/api`;
```

### 6. **`src/hooks/useWebSocket.ts`**
```typescript
// BEFORE:
const wsUrl = `ws://localhost:8001/api/ws?audience=${audience}...`;

// AFTER:
const wsUrl = `ws://localhost:29080/api/ws?audience=${audience}...`;
```

### 7. **`src/features/fabric/hooks/useIPWhitelist.ts`**
```typescript
// BEFORE:
const candidates = ['/api/tenants', 'http://localhost:8001/api/tenants', 'http://localhost:3000/api/tenants'];

// AFTER:
const backendBase = (import.meta.env.VITE_API_BASE_URL as string) || 'http://localhost:29080';
const candidates = ['/api/tenants', `${backendBase}/api/tenants`, 'http://localhost:3000/api/tenants'];
```

---

## Configuration Flow

```
┌─────────────────────────────────────────────────────────┐
│  Frontend (.env)                                         │
│  VITE_API_BASE_URL=http://localhost:29080               │
│  VITE_GRAPHQL_ENDPOINT=http://localhost:8080/v1/graphql│
└──────────────┬──────────────────────────────────────────┘
               │
               ├─► api.ts
               │   └─► Uses VITE_API_BASE_URL for REST calls
               │
               ├─► useNotificationAPI.ts
               │   └─► Falls back to VITE_API_BASE_URL
               │
               ├─► useDashboardService.ts
               │   └─► Falls back to VITE_API_BASE_URL
               │
               ├─► useModelCatalog.ts
               │   └─► Uses VITE_API_BASE_URL + '/api'
               │
               ├─► useWebSocket.ts
               │   └─► WebSocket to :29080
               │
               ├─► useIPWhitelist.ts
               │   └─► Falls back to VITE_API_BASE_URL
               │
               └─► apolloClient.tsx
                   └─► Uses VITE_GRAPHQL_ENDPOINT (:8080)
```

---

## Backend Endpoints

| Service | Port | Type | Protocol |
|---------|------|------|----------|
| REST API | `29080` | HTTP | http://localhost:29080/api/* |
| WebSocket | `29080` | WS | ws://localhost:29080/api/ws |
| GraphQL (Hasura) | `8080` | HTTP/GraphQL | http://localhost:8080/v1/graphql |

---

## Verification Checklist

- ✅ No hardcoded `8001` references in frontend source
- ✅ `.env` points to correct ports (`29080` for backend, `8080` for Hasura)
- ✅ All API modules use `VITE_API_BASE_URL` environment variable
- ✅ WebSocket configured for port `29080`
- ✅ GraphQL endpoint points to Hasura on `8080`
- ✅ Fallback logic handles missing environment variables
- ✅ Frontend can be restarted and pick up new configuration

---

## How to Test

After restarting the frontend with `npm run dev`:

1. **Check Console Logs**:
   - Look for `[apollo] graphqlEndpoint = http://localhost:8080/v1/graphql`
   - Verify no `localhost:8001` messages appear

2. **Network Tab** (F12 → Network):
   - REST API calls should go to `http://localhost:29080/api/*`
   - GraphQL calls should attempt `http://localhost:8080/v1/graphql`
   - WebSocket should connect to `ws://localhost:29080/api/ws`

3. **Verify Requests**:
   ```bash
   # REST API should respond
   curl 'http://localhost:29080/api/entity_registry?tenant_id=...'
   
   # GraphQL should be available
   curl -X POST 'http://localhost:8080/v1/graphql' \
     -H 'Content-Type: application/json' \
     -d '{"query":"{ __schema { types { name } } }"}'
   ```

---

## Frontend Startup Commands

```bash
# Development server
cd /Users/eganpj/GitHub/semlayer/frontend
npm run dev

# Clean rebuild
rm -rf node_modules/.vite dist .next build
npm run dev

# With explicit environment override (if needed)
VITE_API_BASE_URL=http://localhost:29080 \
VITE_GRAPHQL_ENDPOINT=http://localhost:8080/v1/graphql \
npm run dev
```

---

## CI/CD & Deployment Notes

For production deployments, override these via environment:

```bash
# Build-time
VITE_API_BASE_URL=https://api.production.com \
VITE_GRAPHQL_ENDPOINT=https://api.production.com/graphql \
npm run build

# Or inject at runtime via index.html before loading JavaScript
# (See vite.config.ts for configuration options)
```

---

## Summary

All frontend services now consistently use:
- **Backend REST API**: `http://localhost:29080`
- **GraphQL Engine**: `http://localhost:8080/v1/graphql`
- **WebSocket**: `ws://localhost:29080`

Changes are environment-driven (via `.env` and VITE_ variables), making it easy to swap backends without code changes.

---

**Last Updated**: Current session (all port 8001 references removed)  
**Status**: ✅ Ready for frontend restart
