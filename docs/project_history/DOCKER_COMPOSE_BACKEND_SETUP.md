# 🐳 Bulletproof Docker Compose Backend Setup

**Status**: ✅ Production Ready

This guide ensures reliable, consistent startup of the Semlayer backend using Docker Compose.

## Quick Start (30 seconds)

```bash
# Terminal 1: Start the backend stack
cd /Users/eganpj/GitHub/semlayer
docker compose -f docker-compose.backend.yml up

# Terminal 2: Start the frontend (in another terminal)
cd /Users/eganpj/GitHub/semlayer/frontend
npm run dev
```

**Then open your browser:**
- Frontend: http://localhost:5173
- Backend API: http://localhost:8080
- Hasura GraphQL: http://localhost:8888
- RabbitMQ Console: http://localhost:15672 (guest/guest)

---

## Service Architecture

### ✅ Enabled by Default (Minimal Core Stack)

| Service | Port | Status | Purpose |
|---------|------|--------|---------|
| **Backend API** | 8080 | Required | Go REST API server |
| **Hasura GraphQL** | 8888 | Required | GraphQL engine |
| **RabbitMQ** | 5672 / 15672 | Required | Message broker |

### 🔧 Optional Services (Profiles)

| Service | Port | Profile | Purpose |
|---------|------|---------|---------|
| **Temporal** | 7233 | `temporal` | Workflow orchestration |
| **Temporal UI** | 8082 | `temporal` | Workflow monitoring |
| **Workflow Service** | 8084 | `temporal` | Temporal integration |
| **Screen Builder** | 8086 | `services` | Dynamic UI builder |
| **Rule Engine** | 8085 | `services` | Business rule execution |

---

## Command Cheatsheet

### Minimal Stack (Recommended for Development)

```bash
# Start just the essentials
docker compose -f docker-compose.backend.yml up

# Start and run in background
docker compose -f docker-compose.backend.yml up -d

# View logs
docker compose -f docker-compose.backend.yml logs -f

# Stop everything
docker compose -f docker-compose.backend.yml down

# Clean up and restart fresh
docker compose -f docker-compose.backend.yml down -v
docker compose -f docker-compose.backend.yml up
```

### Add Optional Services

```bash
# Include Temporal (workflows)
docker compose -f docker-compose.backend.yml --profile temporal up

# Include Screen Builder + Rule Engine
docker compose -f docker-compose.backend.yml --profile services up

# Include everything
docker compose -f docker-compose.backend.yml --profile temporal --profile services up
```

---

## Frontend Configuration

The frontend `.env` is already configured to use Docker backend:

```
# frontend/.env
VITE_API_BASE_URL=http://localhost:8080
VITE_BACKEND_TARGET=http://localhost:8080
VITE_GRAPHQL_ENDPOINT=/v1/graphql
VITE_GRAPHQL_WS_ENDPOINT=ws://localhost:8888/v1/graphql
```

### Override for Local Testing

If you want to test with a local Go backend instead of Docker:

```bash
# terminal 1: Start local Go backend
cd backend
PORT=8080 go run ./cmd/server

# terminal 2: Update frontend env (optionally)
# frontend/.env can stay the same or you can use:
# VITE_API_BASE_URL=http://localhost:8080
# VITE_BACKEND_TARGET=http://localhost:8080
```

---

## Health Checks

Verify each service is running:

```bash
# Backend API
curl http://localhost:8080/health

# Hasura GraphQL  
curl http://localhost:8888/healthz

# RabbitMQ management
curl http://localhost:15672/api/overview -u guest:guest

# Check all Docker services
docker compose -f docker-compose.backend.yml ps
```

**Expected Output:**

```
STATUS
healthy
healthy
healthy
```

---

## Troubleshooting

### ❌ "Address already in use" or Port conflict

```bash
# Kill any processes using port 8080
lsof -i :8080 | grep LISTEN | awk '{print $2}' | xargs kill -9

# Or change port mapping in docker-compose.backend.yml:
# ports:
#   - "8081:8080"  # Host:Container
```

### ❌ Backend not responding / 404 errors

```bash
# Check backend is running
docker ps | grep semlayer-backend

# View backend logs
docker logs semlayer-backend-1

# Verify database connection
curl http://localhost:8080/health

# Expected: {"status":"healthy","timestamp":"..."}
```

### ❌ Hasura connection errors

Hasura attempts to connect to your **local** PostgreSQL on `host.docker.internal:5432`.

```bash
# Verify local Postgres is running:
psql postgres://postgres:postgres@localhost:5432/alpha -c "SELECT 1"

# If Postgres isn't running, start it:
# macOS with Homebrew:
brew services start postgresql@14

# Or use Docker:
docker run -d \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=alpha \
  -p 5432:5432 \
  postgres:14
```

### ❌ Frontend can't reach backend (ERR_CONNECTION_REFUSED)

1. **Verify backend is running:**
   ```bash
   curl -v http://localhost:8080/health
   ```

2. **Check frontend's `.env`:**
   ```bash
   grep -E "VITE_API_BASE_URL|VITE_BACKEND_TARGET" frontend/.env
   ```
   Should show:
   ```
   VITE_API_BASE_URL=http://localhost:8080
   VITE_BACKEND_TARGET=http://localhost:8080
   ```

3. **Restart frontend dev server:**
   ```bash
   cd frontend
   npm run dev
   ```

4. **Try from browser console:**
   ```javascript
   // In browser DevTools console
   fetch('http://localhost:8080/health').then(r => r.json()).then(console.log)
   ```

---

## Docker Resources

### View Container Logs

```bash
# All services
docker compose -f docker-compose.backend.yml logs -f

# Specific service
docker compose -f docker-compose.backend.yml logs -f backend
docker compose -f docker-compose.backend.yml logs -f hasura
docker compose -f docker-compose.backend.yml logs -f rabbitmq
```

### Clean Up Everything

```bash
# Stop all containers
docker compose -f docker-compose.backend.yml down

# Remove volumes (data)
docker compose -f docker-compose.backend.yml down -v

# Remove orphaned containers
docker compose -f docker-compose.backend.yml down --remove-orphans

# Prune all Docker resources (careful!)
docker system prune -f
```

### Rebuild Images

If you modify `backend/Dockerfile` or service code:

```bash
# Rebuild backend image
docker compose -f docker-compose.backend.yml build backend

# Rebuild and restart
docker compose -f docker-compose.backend.yml up --build backend
```

---

## Performance Tips

### 1. Use `-d` Flag for Background Mode

```bash
docker compose -f docker-compose.backend.yml up -d
# Check status
docker compose -f docker-compose.backend.yml ps
```

### 2. Increase Docker Memory

Docker Desktop → Preferences → Resources → Memory: Set to at least 4GB

### 3. Exclude Temporal for Faster Startup

```bash
# Default (no Temporal) is fastest:
docker compose -f docker-compose.backend.yml up

# Add profile only when needed:
docker compose -f docker-compose.backend.yml --profile temporal up
```

### 4. Monitor Resource Usage

```bash
docker stats
```

---

## Environment Variables

Core environment variables are pre-configured in `docker-compose.backend.yml`. To override:

### Create `.env` file in repository root:

```bash
# .env
HASURA_ADMIN_SECRET=your_secret_key
POSTGRES_HOST=localhost
POSTGRES_USER=postgres
POSTGRES_PWD=postgres
```

Then restart:

```bash
docker compose -f docker-compose.backend.yml up
```

---

## Database Migrations

The backend runs migrations automatically on startup:

```bash
# View migration logs
docker logs semlayer-backend-1 | grep -i migrat

# Manual migration (if needed)
# Modify backend/migrations/... then rebuild:
docker compose -f docker-compose.backend.yml build backend
docker compose -f docker-compose.backend.yml up backend
```

---

## Production Notes

For production deployment:

1. **Use managed database** (AWS RDS, Azure Database, etc.) instead of local Postgres
2. **Change admin secrets:**
   ```bash
   HASURA_ADMIN_SECRET=<strong-random-secret>
   JWT_SECRET=<strong-random-secret>
   ```
3. **Enable SSL/TLS** for all connections
4. **Use environment-specific config files** instead of .env
5. **Set resource limits** in Docker Compose for stability
6. **Monitor logs** with centralized logging (ELK, Datadog, etc.)
7. **Use health checks** and orchestration (Kubernetes, Docker Swarm)

---

## Key Files

| File | Purpose |
|------|---------|
| `docker-compose.backend.yml` | Main Docker Compose configuration |
| `backend/Dockerfile` | Backend service image definition |
| `backend/docker-entrypoint.sh` | Startup script (migrations + server) |
| `.env` | Environment variables (ignored by git) |
| `frontend/.env` | Frontend API configuration |

---

## Support

**For issues:**
1. Check Docker logs: `docker compose -f docker-compose.backend.yml logs`
2. Verify local Postgres is running
3. Ensure port 8080 is available: `lsof -i :8080`
4. Check network: `docker network inspect backend_default`

---

## ✅ Checklist Before Shipping

- [ ] All services start without errors
- [ ] Backend responds: `curl http://localhost:8080/health`
- [ ] Hasura is accessible: `curl http://localhost:8888/healthz`
- [ ] Frontend connects to backend (no ERR_CONNECTION_REFUSED)
- [ ] Database migrations run automatically
- [ ] Optional services (Temporal, etc.) are correctly profiled

---

**Last Updated:** November 17, 2025  
**Tested On:** macOS + Docker Desktop 4.x
