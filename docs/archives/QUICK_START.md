# 🚀 Quick Start Guide - All Services

## Current System State

All services are configured and ready to start. Here's how to get everything running:

---

## 1️⃣ Start Docker Services (Background)

```bash
cd /Users/eganpj/GitHub/semlayer

# Start Redpanda (Kafka), Hasura, Event Router in Docker
docker compose -f docker-compose.backend.yml up -d

# Verify services are running
docker compose -f docker-compose.backend.yml ps
```

**Expected Output**:
```
NAME                      IMAGE                             STATUS
semlayer-redpanda         vectorized/redpanda:latest        Up (healthy)
semlayer-event-router     semlayer-event-router:latest      Up (healthy)
semlayer-graphql-engine   hasura/graphql-engine:latest      Up (healthy)
```

---

## 2️⃣ Start Backend (Native Go Process)

```bash
cd /Users/eganpj/GitHub/semlayer

# Terminal 1: Start backend on port 29080
PORT=29080 go run ./backend/cmd/server

# Or in background:
PORT=29080 go run ./backend/cmd/server > /tmp/backend.log 2>&1 &
```

**Verify Backend**:
```bash
curl -s 'http://localhost:29080/api/entity_registry?tenant_id=910638ba-a459-4a3f-bb2d-78391b0595f6&datasource_id=982aef38-418f-46dc-acd0-35fe8f3b97b0' | head -c 100
# Should return JSON data starting with: {"entity_registry":[...
```

---

## 3️⃣ Start Frontend (React + Vite)

```bash
cd /Users/eganpj/GitHub/semlayer/frontend

# Terminal 2: Start frontend dev server
npm run dev

# Or clean rebuild:
rm -rf node_modules/.vite dist
npm run dev
```

**Verify Frontend**:
- Open browser: http://localhost:5173
- Console should show: `[apollo] graphqlEndpoint = http://localhost:8080/v1/graphql`
- No errors about `localhost:8001` should appear

---

## 🔍 Quick Health Check

```bash
#!/bin/bash

echo "=== Frontend ==="
curl -s http://localhost:5173 | head -c 50 && echo "✅" || echo "❌"

echo ""
echo "=== Backend API ==="
curl -s 'http://localhost:29080/api/entity_registry?tenant_id=910638ba-a459-4a3f-bb2d-78391b0595f6&datasource_id=982aef38-418f-46dc-acd0-35fe8f3b97b0' | head -c 100 && echo "✅" || echo "❌"

echo ""
echo "=== GraphQL (Hasura) ==="
curl -s http://localhost:8080/healthz && echo "✅" || echo "❌"

echo ""
echo "=== Redpanda (Kafka) Broker ==="
# Use rpk (Redpanda CLI) or Pandaproxy for a quick check
rpk cluster info || curl -s http://localhost:8082 | head -c 200 && echo "✅" || echo "❌"

echo ""
echo "=== Event Router ==="
curl -s http://localhost:8081/health && echo "✅" || echo "❌"
```

---

## Backend Build Note

The backend path is: `./backend/cmd/server`

Always use:
```bash
PORT=29080 go run ./backend/cmd/server
```

NOT: `go run ./cmd/server` (this path doesn't exist)

---

## 📊 Port Reference

| Service | Port | Protocol | URL |
|---------|------|----------|-----|
| **Frontend** | 5173 | HTTP | http://localhost:5173 |
| **Backend API** | 29080 | HTTP | http://localhost:29080 |
| **Backend WS** | 29080 | WS | ws://localhost:29080 |
| **GraphQL** | 8080 | HTTP | http://localhost:8080 |
| **Redpanda (Kafka) Broker** | 9092 | Kafka | localhost:9092 |
| **Redpanda (Pandaproxy)** | 8082 | HTTP | http://localhost:8082 |
| **Event Router** | 8081 | HTTP | http://localhost:8081 |
| **PostgreSQL** | 5432 | TCP | localhost:5432 |

---

## 🛑 Stop All Services

```bash
# Stop frontend (Ctrl+C in terminal)
# Stop backend (Ctrl+C in terminal)

# Stop Docker services
docker compose -f docker-compose.backend.yml down

# Or keep containers but stop them
docker compose -f docker-compose.backend.yml stop
```

---

## 🔧 Troubleshooting

### Frontend won't start
```bash
# Clear cache and restart
cd /Users/eganpj/GitHub/semlayer/frontend
rm -rf node_modules/.vite dist .next
npm run dev
```

### Backend port in use
```bash
# Kill existing backend process
pkill -f "go run ./backend/cmd/server"

# Start fresh
PORT=29080 go run ./backend/cmd/server
```

### Docker services stuck
```bash
# Clean everything
docker compose -f docker-compose.backend.yml down -v

# Restart
docker compose -f docker-compose.backend.yml up -d
```

### API returns 404
- Ensure backend is running on port 29080
- Check tenant_id and datasource_id are in query string
- Example: `http://localhost:29080/api/entity_registry?tenant_id=XXX&datasource_id=YYY`

### GraphQL not responding
- Verify Hasura is running: `docker compose -f docker-compose.backend.yml ps | grep graphql`
- Check Hasura console: http://localhost:8080
- Verify database connection: `psql postgres://postgres:postgres@localhost:5432/alpha`

---

## 📝 Environment Variables (Frontend .env)

```properties
# Automatically used by frontend
VITE_API_BASE_URL=http://localhost:29080
VITE_GRAPHQL_ENDPOINT=http://localhost:8080/v1/graphql
VITE_GRAPHQL_WS_ENDPOINT=ws://localhost:8080/v1/graphql
VITE_BACKEND_TARGET=http://localhost:29080
```

---

## 🎯 Development Workflow

### Terminal Setup (3 terminals recommended)

**Terminal 1: Docker Services**
```bash
cd /Users/eganpj/GitHub/semlayer
docker compose -f docker-compose.backend.yml up -d
docker compose -f docker-compose.backend.yml logs -f
```

**Terminal 2: Backend**
```bash
cd /Users/eganpj/GitHub/semlayer
PORT=29080 go run ./backend/cmd/server
```

**Terminal 3: Frontend**
```bash
cd /Users/eganpj/GitHub/semlayer/frontend
npm run dev
```

Then open http://localhost:5173 in browser.

---

## ✅ Recent Fixes Applied

1. **Backend Build Tag** - `main_integration_example.go` excluded with `// +build ignore`
2. **All Frontend Ports** - Updated from 8001 → 29080 (backend) and 8080 (GraphQL)
3. **Environment Variables** - `.env` configured correctly
4. **Docker Services** - RabbitMQ, Hasura, Event Router all healthy

---

## 🚀 Next Steps

1. Start services following the 3 steps above
2. Open http://localhost:5173 in browser
3. Select a tenant/datasource from the UI
4. Verify API calls in Network tab show ✅ 200 responses
5. Check browser console shows no 8001 errors

**Status**: ✅ **System fully configured and ready to run**

---

Last Updated: October 19, 2025
