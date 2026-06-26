# Port Allocation Scheme - Final Authority

This document is the **single source of truth** for all port allocations in the Semlayer project.

## Service Port Assignments

| Service | Port | Type | Description | Status |
|---------|------|------|-------------|--------|
| **Frontend (Vite Dev)** | 5173 | TCP | React development server with HMR | Always running |
| **Backend API** | 8080 | TCP | Go REST API server (running locally during dev) | Always running |
| **Hasura GraphQL** | 8888 | TCP | GraphQL engine (Docker container) | Container only |
| **Temporal Server** | 7233 | TCP | Temporal workflow engine | Container only |
| **RabbitMQ Broker** | 5672 | TCP | Message queue (AMQP) | Container only |
| **RabbitMQ Management** | 15672 | TCP | RabbitMQ web UI | Container only |
| **Temporal UI** | 8081 | TCP | Temporal workflow web UI | Container only |
| **PostgreSQL** | 5432 | TCP | Database (local/Docker) | Local or Docker |

## Key Principles

1. **No Port Sharing**: Each service has a unique, dedicated port
2. **Local Dev ≠ Docker**: Services running locally do NOT also run in Docker
3. **Environment Separation**: 
   - **Local Development**: Backend (8080) + Frontend (5173) run locally
   - **Docker Services**: Hasura (8888), Temporal (7233), RabbitMQ (5672), etc. run in containers
4. **No Port Conflicts**: Since local and Docker services never listen on the same port, there are zero collisions

## Configuration Files

### Backend Configuration
**File**: `.env` (in project root, or set at runtime)
```bash
# Backend runs locally on this port
PORT=8080
# Backend points to Hasura in Docker
HASURA_URL=http://localhost:8888
```

### Frontend Configuration
**File**: `frontend/.env.local`
```dotenv
# Frontend dev server (Vite)
# Default: http://localhost:5173

# Backend API (used by frontend)
VITE_API_BASE_URL=http://127.0.0.1:8080
VITE_BACKEND_TARGET=http://127.0.0.1:8080

# GraphQL endpoint (Hasura in Docker)
VITE_GRAPHQL_ENDPOINT=http://127.0.0.1:8888/v1/graphql
VITE_GRAPHQL_WS_ENDPOINT=ws://127.0.0.1:8888/v1/graphql
VITE_GRAPHQL_ADMIN_SECRET=adminsecret
```

### Docker Compose Configuration
**File**: `docker-compose.backend.yml`
```yaml
hasura:
  ports:
    - "8888:8080"  # Host:Container mapping
  environment:
    HASURA_GRAPHQL_ADMIN_SECRET: adminsecret

rabbitmq:
  ports:
    - "5672:5672"    # AMQP
    - "15672:15672"  # Management UI

temporal:
  ports:
    - "7233:7233"    # gRPC
    - "6933:6933"    # Additional port

temporal-ui:
  ports:
    - "8081:8080"    # Host:Container mapping
```

## Development Startup Checklist

- [ ] PostgreSQL running (local or Docker): `5432`
- [ ] Frontend dev server: `npm run dev` → listens on `5173`
- [ ] Backend API: `go run ./backend/cmd/server` → listens on `8080`
- [ ] Docker containers: `docker compose up -d` → Hasura on `8888`, etc.
- [ ] Verify no conflicts: `lsof -i -P -n | grep LISTEN`

## Verification Commands

```bash
# Check all listening ports
lsof -i -P -n | grep LISTEN

# Check specific port
lsof -i :8080  # Backend
lsof -i :5173  # Frontend
lsof -i :8888  # Hasura

# Kill port if needed (last resort)
lsof -i :8080 | grep LISTEN | awk '{print $2}' | xargs kill -9
```

## Why This Works

1. **Clear Separation**: Local services (backend, frontend) use ports 8080 and 5173
2. **Container Isolation**: Docker services use separate ports (8888, 7233, 5672, etc.)
3. **No Dual Listening**: A service never listens on multiple ports in different modes
4. **Documented**: All allocations are explicit and centralized
5. **Reproducible**: Any developer can follow this scheme and get the same setup

## Future Changes

If you need to add or change a port:
1. Update this document first
2. Update the relevant `.env` or `docker-compose.yml` files
3. Communicate the change to the team
4. NEVER reuse a port without removing it from this list
