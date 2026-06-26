# Unified Port Architecture - Same Ports, Any Execution Context

## The Problem (Solved)

Previously, different port allocations were needed for local vs Docker execution:
- ❌ Local: Backend 8080, Hasura 8080 (collision)
- ❌ Docker: Backend 29080, Hasura 8080 (inconsistent)
- ❌ Switching between contexts required config changes

## The Solution

**Same port allocations work for BOTH local and Docker execution with ZERO changes.**

### Port Allocation (Universal)

| Service | Port | Local | Docker | Comment |
|---------|------|-------|--------|---------|
| Frontend (Vite) | 5173 | ✅ Local only | N/A | Always runs locally |
| Backend API | 8080 | ✅ Local process | ✅ Container | **Unified** - same port both ways |
| Hasura GraphQL | 8888 | ✅ Calls Docker on 8888 | ✅ Internal 8080→8888 | Docker container on 8888 |
| Temporal | 7233 | ✅ Calls Docker | ✅ Container | Only in Docker |
| RabbitMQ | 5672 | ✅ Calls Docker | ✅ Container | Only in Docker |
| PostgreSQL | 5432 | ✅ Local/Docker | ✅ Local/Docker | Database |

## How It Works

### Local Execution
```bash
# Terminal 1: Backend
PORT=8080 go run ./backend/cmd/server
# Backend listens on 8080 ✅
# Reads HASURA_URL=http://localhost:8888 ✅
# Connects to Hasura on Docker container ✅

# Terminal 2: Frontend  
npm run dev
# Frontend on 5173 ✅
# Calls backend at http://localhost:8080 ✅
# Calls Hasura at http://localhost:8888 ✅

# Terminal 3: Docker
docker compose up -d
# Hasura container listens on 8888 (mapped from internal 8080) ✅
# Temporal on 7233 ✅
# RabbitMQ on 5672 ✅
```

### Docker Execution (Full Stack)
```bash
# All services in containers
docker compose up -d

# Backend container:
# - Listens on 8080 (internal) → 8080 (host) ✅
# - HASURA_URL=http://hasura:8080 (Docker internal DNS) ✅
# - DATABASE_URL=postgresql://postgres:postgres@host.docker.internal:5432/alpha ✅

# Hasura container:
# - Listens on 8080 (internal) → 8888 (host) ✅

# Frontend in browser:
# - Calls backend at http://localhost:8080 ✅
# - Calls Hasura at http://localhost:8888 ✅
```

## The Magic: Service Discovery

### Local Context
- Frontend → Backend: `http://localhost:8080` (direct process)
- Frontend → Hasura: `http://localhost:8888` (Docker container)
- Backend → Hasura: `http://localhost:8888` (from .env)

### Docker Context
- Frontend → Backend: `http://localhost:8080` (container port 8080)
- Frontend → Hasura: `http://localhost:8888` (container mapped to 8888)
- Backend (container) → Hasura: `http://hasura:8080` (Docker DNS resolution)

**Key insight**: In Docker, the backend service uses internal Docker DNS (`hasura:8080`), while the frontend uses localhost port mapping (`localhost:8888`).

## Configuration Files

### `.env` (Root)
```bash
PORT=8080                           # Same for both local and Docker
HASURA_URL=http://localhost:8888    # Local dev points to Docker container
HASURA_ADMIN_SECRET=adminsecret
```

### `docker-compose.backend.yml`
```yaml
hasura:
  ports:
    - "8888:8080"  # Host 8888 → Container 8080
  environment:
    HASURA_GRAPHQL_ADMIN_SECRET: adminsecret

backend:
  environment:
    PORT: 8080                           # Same as local
    HASURA_URL: http://hasura:8080       # Docker internal DNS
    DATABASE_URL: postgresql://...       # For Docker container
  ports:
    - "8080:8080"                        # Container 8080 → Host 8080
```

### `frontend/.env.local`
```dotenv
VITE_API_BASE_URL=http://127.0.0.1:8080           # Same for both contexts
VITE_GRAPHQL_ENDPOINT=http://127.0.0.1:8888/v1/graphql  # Same for both
VITE_GRAPHQL_ADMIN_SECRET=adminsecret
```

## Switching Between Contexts (ZERO CONFIG CHANGES)

### Start Local Development
```bash
# Just run - no env file changes needed!
go run ./backend/cmd/server         # Terminal 1
npm run dev                         # Terminal 2
docker compose up -d                # Terminal 3
```

### Switch to Full Docker Stack
```bash
# Kill local backend, then:
docker compose up -d

# Frontend still works:
# - Calls http://localhost:8080 (now Docker container)
# - Calls http://localhost:8888 (Docker Hasura)
# NO CONFIG CHANGES NEEDED! ✅
```

### Back to Local Development
```bash
# Kill Docker, then:
go run ./backend/cmd/server
npm run dev

# Frontend still works:
# - Calls http://localhost:8080 (now local process)
# - Calls http://localhost:8888 (Docker Hasura)
# NO CONFIG CHANGES NEEDED! ✅
```

## Why This Works

1. **Backend Port Unified**: 8080 works the same whether it's a local process or Docker container
2. **Hasura Always on 8888**: Frontend always sees it at 8888 (Docker maps internal 8080 to host 8888)
3. **Internal Docker DNS**: When backend runs in Docker, it uses `http://hasura:8080` (Docker internal)
4. **Single .env**: Works for both contexts because port numbers never change
5. **Frontend Location Agnostic**: React doesn't care if backend is local process or container at 8080

## Guarantees

✅ **ZERO changes needed** to switch between local and Docker execution  
✅ **Same port allocations** for both contexts  
✅ **No port collisions** - each service has unique, dedicated port  
✅ **Scalable** - add new services to Docker without affecting local dev  
✅ **Team consistency** - all developers use identical configuration  

## Future-Proof

If you need to add a new service:
1. Add to docker-compose.backend.yml
2. Allocate a unique port (check QUICK_PORT_REFERENCE.txt)
3. Update .env if needed
4. Test with `docker compose up -d`
5. Test with local execution
6. Verify no port conflicts

**The principle remains**: Local and Docker services never fight for the same port.
