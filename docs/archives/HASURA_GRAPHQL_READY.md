# ✅ Hasura GraphQL System Ready

## Status

Your system is now **fully configured and running** with Hasura GraphQL as the primary data interface.

### Running Services

| Service | Port | Status | URL |
|---------|------|--------|-----|
| **Hasura GraphQL Engine** | 8083 | ✅ Running | http://localhost:8083/v1/graphql |
| **Frontend (Vite)** | 5173 | ✅ Running | http://localhost:5173 |
| **RabbitMQ** | 5672 (AMQP), 15672 (UI) | ✅ Running | http://localhost:15672 |
| **PostgreSQL** | 5432 | ✅ Running (local) | localhost:5432 |

---

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│ Frontend (React + Vite)                                  │
│ http://localhost:5173                                   │
│                                                          │
│ Apollo Client configured for Hasura                     │
│ Endpoint: http://localhost:8083/v1/graphql              │
└─────────────────┬───────────────────────────────────────┘
                  │ GraphQL Queries
                  ▼
┌─────────────────────────────────────────────────────────┐
│ Hasura GraphQL Engine v2.46.0                           │
│ http://localhost:8083                                   │
│                                                          │
│ ✅ Console: http://localhost:8083/console (if enabled) │
│ ✅ Health: http://localhost:8083/healthz               │
└─────────────────┬───────────────────────────────────────┘
                  │ Resolves to DB
                  ▼
┌─────────────────────────────────────────────────────────┐
│ PostgreSQL 15 (Local)                                    │
│ localhost:5432, Database: alpha                         │
│ User: postgres, Password: postgres                      │
└─────────────────────────────────────────────────────────┘
```

---

## Configuration Files

### ✅ Apollo Client (`frontend/src/graphql/apolloClient.tsx`)

```typescript
// Development: Uses Hasura at port 8083
const graphqlEndpoint = envEndpoint || 
  ((import.meta.env.DEV) ? 'http://localhost:8083/v1/graphql' : '/api/graphql');
```

**Status**: ✅ Correctly configured to Hasura on port 8083

### ✅ Frontend Environment (`frontend/.env.local`)

```bash
VITE_GRAPHQL_ENDPOINT=http://localhost:8083/v1/graphql
VITE_GRAPHQL_ADMIN_SECRET=dev-secret
VITE_API_BASE_URL=http://localhost:8080
```

**Status**: ✅ All variables set correctly

### ✅ Docker Compose (`docker-compose.yml`)

Hasura service definition:
- **Image**: `hasura/graphql-engine:v2.46.0`
- **Port**: 8083 (host) → 8080 (container)
- **Database**: PostgreSQL at `host.docker.internal:5432`
- **Console**: Enabled for GraphQL schema management
- **Dev Mode**: Enabled for better error messages

**Status**: ✅ Hasura configured and running

---

## Testing Hasura GraphQL

### 1. **Quick Health Check**

```bash
curl http://localhost:8083/v1/graphql \
  -X POST \
  -H "Content-Type: application/json" \
  -d '{"query":"{ __typename }"}'
```

**Expected Response**:
```json
{"data":{"__typename":"query_root"}}
```

✅ **Verified working**

### 2. **From Browser Console**

Open http://localhost:5173, then in DevTools console:

```javascript
// Apollo Client will automatically send queries to Hasura
// Check for successful connections in Network tab under GraphQL requests
console.log(window.localStorage.getItem('selected_tenant')); // Should show tenant info
```

### 3. **Hasura Console** (if needed)

Visit http://localhost:8083/console to:
- View GraphQL schema
- Add/modify tables
- Create remote schemas
- Set up permissions and roles
- Test queries directly

---

## What's Working Now

✅ **Frontend** → Vite dev server running on 5173  
✅ **Apollo Client** → Configured to point to Hasura (port 8083)  
✅ **Hasura GraphQL Engine** → Running, responding to queries  
✅ **Database Connection** → PostgreSQL connected, serving schema  
✅ **RabbitMQ** → Running for event streaming  

---

## Next Steps

### Option 1: Test GraphQL in Browser

1. Open http://localhost:5173
2. Open DevTools (F12)
3. Go to **Network** tab
4. Trigger any component that uses Apollo Client
5. You should see GraphQL requests to `localhost:8083/v1/graphql`

### Option 2: Query via curl

Test a real schema query (example assumes a `users` table exists):

```bash
curl http://localhost:8083/v1/graphql \
  -X POST \
  -H "Content-Type: application/json" \
  -d '{
    "query": "{ users { id name email } }"
  }'
```

### Option 3: Configure BP Builder in Hasura

If you want the Business Process Builder to use Hasura:

1. Visit http://localhost:8083/console
2. Click "Data" in the top menu
3. Navigate to the database and explore tables
4. Any table you want in GraphQL should be auto-exposed (or add manually)
5. Set up permissions as needed

---

## Troubleshooting

### "Cannot connect to Hasura"

**Check 1**: Verify container is running
```bash
docker ps | grep graphql-engine
```

**Check 2**: Verify port is accessible
```bash
curl http://localhost:8083/healthz
```

**Check 3**: Check Hasura logs
```bash
docker compose logs graphql-engine
```

### "GraphQL schema has no tables"

This is normal if no tables are exposed yet. Visit http://localhost:8083/console to:
- Add tables from your PostgreSQL database
- Or create new tables
- Permissions will auto-track them in GraphQL

### "Apollo Client still tries port 8080"

Clear browser cache and restart Vite:
```bash
cd frontend
npm run dev  # Will pick up .env.local changes
```

---

## Key Endpoints

| Purpose | URL | Method | Notes |
|---------|-----|--------|-------|
| GraphQL Queries | http://localhost:8083/v1/graphql | POST | Apollo Client sends here |
| Health Check | http://localhost:8083/healthz | GET | Returns 200 OK |
| Hasura Console | http://localhost:8083/console | GET | Schema management UI |
| Frontend | http://localhost:5173 | GET | Vite dev server |
| RabbitMQ UI | http://localhost:15672 | GET | Guest/guest login |

---

## Environment Variables Reference

**Frontend** (`.env.local`):
```
VITE_GRAPHQL_ENDPOINT        = Hasura GraphQL URL (port 8083)
VITE_GRAPHQL_ADMIN_SECRET    = Hasura admin secret for auth
VITE_BACKEND_TARGET          = REST API fallback (port 8080)
VITE_API_BASE_URL            = REST API base URL
```

**Docker** (`docker-compose.yml` / `.env`):
```
HASURA_GRAPHQL_DATABASE_URL       = PostgreSQL connection
HASURA_GRAPHQL_ADMIN_SECRET       = Admin authorization token
HASURA_GRAPHQL_ENABLE_CONSOLE     = true (manage schema)
HASURA_GRAPHQL_DEV_MODE           = true (detailed errors)
```

---

## Summary

Your system is **GraphQL-first** using Hasura as the primary data interface. Frontend queries go to Hasura (port 8083), which resolves them against PostgreSQL. Rest API (port 8080) remains available as a fallback if needed.

**Everything is running and ready for use.** ✅

