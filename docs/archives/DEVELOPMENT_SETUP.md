# Development Environment Setup Guide

## 🎯 Key Principle

**Same ports work for BOTH local and Docker execution - ZERO changes needed to switch!**

## Quick Verify

```bash
# Check current port configuration
grep "^PORT=" /Users/eganpj/GitHub/semlayer/.env
grep "^HASURA_URL=" /Users/eganpj/GitHub/semlayer/.env

# Expected output:
# PORT=8080
# HASURA_URL=http://localhost:8888
```

## Option 1: Local Development (Backend runs locally)

```bash
# Terminal 1: Backend API
cd /Users/eganpj/GitHub/semlayer
go run ./backend/cmd/server
# Listens on: http://localhost:8080 ✅

# Terminal 2: Frontend
cd /Users/eganpj/GitHub/semlayer/frontend
npm run dev
# Listens on: http://localhost:5173 ✅

# Terminal 3: Docker Services
cd /Users/eganpj/GitHub/semlayer
docker compose up -d
# Hasura: 8888, Temporal: 7233, RabbitMQ: 5672 ✅
```

## Option 2: Full Docker Stack (All services in Docker)

```bash
# One command - everything in containers
cd /Users/eganpj/GitHub/semlayer
docker compose up -d

# Frontend still runs locally
cd /Users/eganpj/GitHub/semlayer/frontend
npm run dev
# Listens on: http://localhost:5173 ✅

# Frontend calls:
# - Backend at http://localhost:8080 (now Docker container)
# - Hasura at http://localhost:8888 (Docker container)
# NO CONFIG CHANGES! Same URLs work! ✅
```

## Port Summary (Same for Both Options)

| Port | Service | Local | Docker | Status |
|------|---------|-------|--------|--------|
| 5173 | Frontend | ✅ Process | ✅ Process | Always local |
| 5432 | PostgreSQL | ✅ Running | ✅ Running | Always accessible |
| 5672 | RabbitMQ | ✅ Via Docker | ✅ Container | Docker services |
| 7233 | Temporal | ✅ Via Docker | ✅ Container | Docker services |
| 8080 | Backend | ✅ Process | ✅ Container | **UNIFIED** |
| 8888 | Hasura | ✅ Via Docker | ✅ Container | **UNIFIED** |

## Verification

```bash
# Check all ports in use
lsof -i -P -n | grep LISTEN | sort -k9

# Test each service
curl http://localhost:8080/health                              # Backend
curl http://localhost:5173                                     # Frontend
curl -H "x-hasura-admin-secret: adminsecret" \
     http://localhost:8888/v1/graphql \
     -X POST -H "Content-Type: application/json" \
     -d '{"query":"{__typename}"}'                             # GraphQL
```

## Switching Between Contexts

### Local → Docker
```bash
# 1. Kill local backend (Ctrl+C in Terminal 1)
# 2. In new terminal:
docker compose up -d

# That's it! Frontend still works on same URLs!
```

### Docker → Local
```bash
# 1. Kill Docker services:
docker compose down

# 2. Start backend locally:
go run ./backend/cmd/server

# That's it! Frontend still works on same URLs!
```

## Environment Files

- **Root**: `.env` - Unified configuration (works for both local and Docker)
- **Frontend**: `frontend/.env` - Frontend build settings
- **Frontend Local**: `frontend/.env.local` - Local dev overrides

## Key Insight: Why This Works

1. **Backend Port Unified (8080)**: Whether it's a local process or Docker container, it listens on 8080
2. **Docker Internal DNS**: When backend runs in Docker, it finds Hasura via `http://hasura:8080` (Docker internal DNS)
3. **Frontend Location Agnostic**: React frontend doesn't care - it calls the same URLs regardless
4. **Hasura Mapping**: Docker maps internal 8080 to host 8888, so frontend always sees it at 8888
5. **Single .env**: Port numbers never change, so no reconfiguration needed

## Critical: NEVER Do This

- ❌ Don't run backend on different port locally vs Docker (breaks consistency)
- ❌ Don't hardcode different HASURA_URL for local vs Docker in .env
- ❌ Don't commit `frontend/.env.local` to git
- ❌ Don't change port allocations without updating documentation

## For Full Details

See **UNIFIED_PORT_ARCHITECTURE.md** for technical deep dive.
