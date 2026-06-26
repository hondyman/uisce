# Port Collision Prevention - Implementation Complete

## Problem Solved

**Previous Issue**: Port collisions between backend (8080) and Hasura (8080) in Docker, plus inconsistent configurations across `.env` files and hardcoded fallbacks.

**Root Cause**: Multiple service listening on same port in different execution contexts:
- Backend running locally wanted port 8080
- Hasura Docker container also configured for port 8080
- Frontend hardcoded fallback pointed to port 29080 (then 8080)
- Multiple `.env` files with conflicting settings

## Solution Implemented

### 1. Definitive Port Allocation
Created single source of truth: **PORT_ALLOCATION_SCHEME.md**

| Service | Port | Context | Collision Risk |
|---------|------|---------|-----------------|
| Frontend | 5173 | Local (always) | ✅ Safe - never used by others |
| Backend | 8080 | Local (always) | ✅ Safe - Docker service uses 8888 |
| Hasura | 8888 | Docker (always) | ✅ Safe - not used by local services |
| Temporal | 7233 | Docker (always) | ✅ Safe - unique |
| RabbitMQ | 5672 | Docker (always) | ✅ Safe - unique |

### 2. Configuration Alignment

**Root `.env`** (`/Users/eganpj/GitHub/semlayer/.env`)
```dotenv
PORT=8080                              # Backend listens here (local)
HASURA_URL=http://localhost:8888       # Backend forwards GraphQL to Docker
HASURA_ADMIN_SECRET=adminsecret        # Shared secret
```

**Frontend `.env.local`** (`frontend/.env.local`)
```dotenv
VITE_API_BASE_URL=http://127.0.0.1:8080        # Points to local backend
VITE_GRAPHQL_ENDPOINT=http://127.0.0.1:8888/v1/graphql  # Points to Docker Hasura
VITE_GRAPHQL_ADMIN_SECRET=adminsecret           # Matches Hasura
```

**Docker Compose** (`docker-compose.backend.yml`)
```yaml
hasura:
  ports:
    - "8888:8080"  # Host 8888 → Container 8080 (avoids collision)
  environment:
    HASURA_GRAPHQL_ADMIN_SECRET: adminsecret
```

### 3. Fixed Hardcoded Fallbacks

**File**: `frontend/src/utils/api.ts`
```typescript
// BEFORE: Hardcoded to 29080
const apiBaseUrl = import.meta.env.VITE_API_BASE_URL || 'http://localhost:29080';

// AFTER: Hardcoded to 8080 (matches actual backend port)
const apiBaseUrl = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080';
```

### 4. Separation of Concerns

**Local Services** (developer machine):
- Frontend: 5173
- Backend: 8080
- Both configured to point to Docker services for GraphQL

**Docker Services** (containers):
- Hasura: 8888
- Temporal: 7233
- RabbitMQ: 5672
- Never compete with local services

### 5. Documentation

Created two comprehensive guides:
- **PORT_ALLOCATION_SCHEME.md**: Authority on all port assignments
- **DEVELOPMENT_SETUP.md**: Step-by-step startup guide with verification

## Verification

All services running without collisions:
```
✅ 5173 - Frontend (Vite dev server)
✅ 5432 - PostgreSQL
✅ 5672 - RabbitMQ
✅ 7233 - Temporal
✅ 8080 - Backend API
✅ 8888 - Hasura GraphQL
```

## Guarantees

1. **No Future Collisions**: Port allocations are centralized and documented
2. **Single Source of Truth**: PORT_ALLOCATION_SCHEME.md is the authority
3. **Environment Consistency**: `.env` and `.env.local` aligned
4. **Clear Startup Process**: DEVELOPMENT_SETUP.md provides exact commands
5. **Troubleshooting**: Detailed resolution steps included

## Key Principle

> **Local services and Docker services never use the same port.**

- Backend (8080) always runs locally
- Hasura (8888) always runs in Docker
- Frontend (5173) always runs locally
- Each service has exactly ONE port, in exactly ONE context

## For Future Development

If adding a new service:
1. Update PORT_ALLOCATION_SCHEME.md first
2. Update .env files
3. Update DEVELOPMENT_SETUP.md
4. Never reuse an allocated port
5. Test startup sequence from DEVELOPMENT_SETUP.md

