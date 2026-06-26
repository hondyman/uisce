# Quick Fix: Get Backend Services Running

**Problem:** Frontend can't reach GraphQL at `http://localhost:8080/v1/graphql` 

**Solution:** Start the required Docker containers properly

---

## Option 1: Use Docker (Recommended for Full Stack)

```bash
cd /Users/eganpj/GitHub/semlayer

# Stop any running containers
docker compose down

# Start just the essential services:
docker compose up -d hasura rabbitmq postgres
```

**Wait 30 seconds for services to initialize**, then check:
```bash
curl http://localhost:8080/healthz
# Should return 200 OK
```

---

## Option 2: Run Backend Locally (Faster for Development)

### 1. Set Environment Variables

```bash
export DSN="postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable"
export GRAPHQL_URL="http://localhost:8080/v1/graphql"
export PORT=":8081"
```

### 2. Start Backend

```bash
cd /Users/eganpj/GitHub/semlayer/backend
go run ./cmd/server/main.go
```

Should see:
```
Database connection established successfully
Server listening on :8081
```

### 3. Update Frontend Endpoint

Edit `/Users/eganpj/GitHub/semlayer/frontend/.env`:

```properties
# Change this:
VITE_API_BASE_URL=http://localhost:8081
VITE_BACKEND_TARGET=http://localhost:8081
```

Restart frontend dev server:
```bash
cd /Users/eganpj/GitHub/semlayer/frontend
npm run dev
```

---

## Option 3: Quick Workaround (Disable GraphQL Temporarily)

If you just want to test the UI without backend:

```bash
# Edit frontend/.env and comment out GraphQL:
# VITE_GRAPHQL_ENDPOINT=http://localhost:8080/v1/graphql

# Or set a mock endpoint:
VITE_GRAPHQL_ENDPOINT=http://localhost:9999/graphql
```

The Apollo fallback link will return empty data instead of crashing.

---

## Troubleshooting

### "Port already in use"
```bash
# Find and kill process on port 8080 or 8081
lsof -i :8080 | grep LISTEN | awk '{print $2}' | xargs kill -9
lsof -i :8081 | grep LISTEN | awk '{print $2}' | xargs kill -9
```

### "Failed to connect to postgres"
```bash
# Verify PostgreSQL is running
psql -h localhost -U postgres -d alpha -c "SELECT 1;"

# Should return:
#  ?column? 
# ----------
#         1
```

### "Cannot resolve host.docker.internal"
- This is a Docker networking issue
- Use Option 2 (run backend locally) instead
- Or use `docker network` commands to fix Docker routing

---

## Recommended: Start the Full Stack

```bash
# Terminal 1: Start Docker services
cd /Users/eganpj/GitHub/semlayer
docker compose up -d hasura rabbitmq

# Terminal 2: Start backend locally (since Docker has networking issues)
cd /Users/eganpj/GitHub/semlayer/backend
export DSN="postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable"
go run ./cmd/server/main.go

# Terminal 3: Start frontend
cd /Users/eganpj/GitHub/semlayer/frontend
npm run dev

# Open browser:
open http://localhost:5173
```

When all 3 are running:
- ✅ Frontend: http://localhost:5173
- ✅ Backend API: http://localhost:8081
- ✅ GraphQL: http://localhost:8080/v1/graphql (via Hasura)
- ✅ Database: postgres://postgres:postgres@localhost:5432/alpha

---

## Next: Continue with Metrics Consolidation

Once services are running, you can now:

1. **Run database analysis:**
   ```bash
   python3 analyze_consolidation.py
   ```

2. **Find code to update:**
   ```bash
   bash find_schema_references.sh
   ```

3. **Execute migration:**
   ```bash
   psql -h localhost -U postgres -d alpha -f migrations/consolidate_metrics_and_dax.sql
   ```

See: `EXECUTE_NOW.md` for full instructions

---

Done! Services should now be running.

