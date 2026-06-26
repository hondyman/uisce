# 🚀 Quick Start: Hasura GraphQL System

## ✅ Current Status (Verified)

- **Hasura GraphQL Engine**: Running on port 8083 ✅
- **Frontend (Vite)**: Running on port 5173 ✅
- **Redpanda (Pandaproxy)**: Running on port 9092 (Pandaproxy: 8082) ✅
- **PostgreSQL**: Running locally on port 5432 ✅

---

## Access Points

### 🎨 Frontend Application
```
http://localhost:5173
```
Vite dev server with React app, Apollo Client pre-configured for Hasura.

### 📊 Hasura GraphQL Console
```
http://localhost:8083/console
```
Manage your GraphQL schema, tables, permissions, and test queries here.

### 📝 GraphQL Endpoint (for direct API calls)
```
http://localhost:8083/v1/graphql
```
POST endpoint for GraphQL queries. Example:
```bash
curl -X POST http://localhost:8083/v1/graphql \
  -H "Content-Type: application/json" \
  -d '{"query":"{ __typename }"}'
```

### � Redpanda (Pandaproxy) UI / rpk
```
http://localhost:15672
```
Username: `guest` | Password: `guest`

---

## Frontend Configuration Summary

**Apollo Client** automatically targets Hasura in development:
- File: `frontend/src/graphql/apolloClient.tsx`
- Development endpoint: `http://localhost:8083/v1/graphql`
- Environment config: `frontend/.env.local`

**No additional setup needed** — just visit http://localhost:5173 and start using the app.

---

## Testing GraphQL Connection

### From Browser Console (F12)
1. Open http://localhost:5173
2. Press F12 (DevTools)
3. Go to **Network** tab
4. Trigger any action that uses Apollo Client
5. Look for requests to `localhost:8083/v1/graphql` — should see 200 responses

### From Terminal
```bash
# Test Hasura is responding
curl http://localhost:8083/healthz

# Execute a GraphQL query (introspection example)
curl -X POST http://localhost:8083/v1/graphql \
  -H "Content-Type: application/json" \
  -d '{"query":"{ __typename }"}'
```

---

## Important Notes

### Database
- Runs locally on your machine (not in Docker)
- PostgreSQL 15, database: `alpha`
- User: `postgres`, Password: `postgres`
- Connection from Hasura: `host.docker.internal:5432` (Docker networking)

### Microservices
- **Only Hasura + RabbitMQ are running** (other services have build issues)
- You have REST API backend at port 8080 available but not in Docker
- This is intentional for now — focus is on GraphQL via Hasura

### Schema Management
- Visit http://localhost:8083/console to manage tables
- Add tables, define relationships, set permissions
- All changes are immediately reflected in GraphQL schema

---

## Stop/Restart Services

### Stop all running services
```bash
docker compose down
```

### Start again (Hasura + RabbitMQ only)
```bash
cd /Users/eganpj/GitHub/semlayer
docker compose up -d graphql-engine rabbitmq
```

### View logs
```bash
docker compose logs -f graphql-engine   # Hasura logs
docker compose logs -f rabbitmq          # RabbitMQ logs
```

---

## Next: Configure BP Builder

To integrate Business Process Builder with Hasura:

1. **Option A**: Keep using REST API (backend on port 8080)
   - Current setup: `useBPBuilderAPI.ts` uses REST endpoints
   - Works as-is

2. **Option B**: Migrate to GraphQL
   - Would require creating `business_processes` table in Hasura
   - Writing GraphQL queries in `useBPBuilderAPI.ts`
   - Takes ~20-30 minutes

**Recommendation**: Option A for now (REST API works fine), migrate to GraphQL later if needed.

---

## Files Modified This Session

✅ `frontend/src/graphql/apolloClient.tsx` — Corrected endpoint to port 8083  
✅ `frontend/.env.local` — Added Hasura GraphQL config  
✅ `HASURA_GRAPHQL_READY.md` — This comprehensive setup guide  

---

**You're all set!** 🎉 Visit http://localhost:5173 to start using the application.

