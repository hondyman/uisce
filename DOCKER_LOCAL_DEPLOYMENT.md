# SemLayer Local Docker Deployment Guide

## Overview

This guide walks you through deploying SemLayer locally on your macBook for development. The setup uses:

- **Backend API** (Go) - Containerized on port 8080
- **Auth Service** (Node.js) - Containerized on port 8001  
- **Frontend** (Vite/React) - Local development server on port 5173-5174
- **External Dependencies**:
  - PostgreSQL - 100.84.126.19:5432
  - Hasura GraphQL - 100.84.126.19:8080

This hybrid approach optimizes for:
- ✅ Backend + Auth service in containers (consistent environment)
- ✅ Frontend on host (faster iteration, better npm module compatibility)

## Prerequisites

1. **Docker Desktop** installed and running on macOS
2. **Go 1.21+** (optional, for local development)
3. **Node.js 18+** (optional, for local development)
4. Access to external PostgreSQL at `100.84.126.19:5432`
5. Access to external Hasura at `100.84.126.19:8080`

## Quick Start

### 1. Start Backend and Auth Services (Docker)

```bash
cd /Users/eganpj/GitHub/semlayer

# Start backend and auth service containers
./docker-mac-local.sh
```

This will:
- 🔨 Build the Go backend binary in Docker
- 📦 Build the Node.js auth service container
- 🚀 Start both services in the background
- 📊 Display endpoint URLs and credentials

### 2. Start Frontend (Local Development)

In a **new terminal**:

```bash
cd /Users/eganpj/GitHub/semlayer/frontend

# Start the Vite dev server
sh scripts/start-dev.sh
```

This will:
- 🔧 Kill any existing process on port 5173
- ⚙️ Start Vite dev server with hot reload
- 📍 Show "Local: http://localhost:5173/"

### 3. Verify Services Are Running

**Backend & Auth:**
```bash
./docker-mac-local.sh logs
```

**Frontend:**
The startup script will show:
```
VITE v5.4.21  ready in 296 ms
➜  Local:   http://localhost:5173/
```

### 3. Access Services

**Frontend:**
```
http://localhost:5173
http://localhost:5174 (if 5173 is in use)
```

**Backend API:**
```
http://localhost:8080/api
http://localhost:8080/swagger/index.html
```

**Auth Service:**
```
http://localhost:8001/api/auth/login
```

## Login Credentials

Use these test credentials:
- **Email:** `test@example.com`
- **Password:** `password123`
- **Role:** `global_ops`

## Docker Compose Configuration

The file `docker-compose.mac-local.yml` configures two services:

### Backend Service

```yaml
environment:
  - POSTGRES_HOST=100.84.126.19      # External database IP
  - POSTGRES_PORT=5432
  - POSTGRES_DB=alpha
  - HASURA_ENDPOINT=http://100.84.126.19:8080
  - PORT=8080
```

**Key Points:**
- Connects to external PostgreSQL at 100.84.126.19:5432
- Connects to external Hasura GraphQL at 100.84.126.19:8080
- Health checks every 30 seconds
- Ports: 8080 → 8080

### Auth Service

```yaml
environment:
  - AUTH_SERVICE_PORT=8001
  - POSTGRES_HOST=100.84.126.19
  - JWT_SECRET=dev-jwt-secret-key-change-in-production
  - ALLOWED_ORIGINS=http://localhost:*
```

**Key Points:**
- Node.js 18 Alpine container
- Connects to external PostgreSQL at 100.84.126.19:5432
- JWT token management
- CORS configured for localhost
- Ports: 8001 → 8001

### Frontend Service

**NOTE:** Frontend runs on your host machine, not in Docker.

Advantages:
- ✅ Proper native modules for your OS (macOS ARM64)
- ✅ Hot reload works perfectly
- ✅ Source maps and debugging work better
- ✅ Faster npm install and Vite startup

Run locally with:
```bash
cd frontend && sh scripts/start-dev.sh
```

## Common Commands

### Docker Services (Backend & Auth)

View logs:
```bash
# All services
./docker-mac-local.sh logs

# Specific service
./docker-mac-local.sh logs backend
./docker-mac-local.sh logs auth-service
```

Restart services:
```bash
# Restart all
./docker-mac-local.sh restart

# Restart specific service
docker compose -f docker-compose.mac-local.yml restart backend
```

Stop services:
```bash
./docker-mac-local.sh down
```

### Frontend (Local Development)

The frontend runs with hot reload enabled:

```bash
# Terminal 1: Start frontend
cd frontend && sh scripts/start-dev.sh

# The terminal will show:
# VITE v5.4.21  ready in 296 ms
# ➜  Local:   http://localhost:5173/

# Make code changes and browser auto-updates!
# Press 'h' in terminal to see help
```

Kill the frontend server:
```bash
# Find Vite process
lsof -i :5173

# Kill it
kill -9 <PID>

# Or use the script which kills it automatically
sh scripts/start-dev.sh
```

## Troubleshooting

### Port Already in Use

For port 8080, 8001, or 5173:

```bash
# Find process using port
lsof -i :8080
lsof -i :8001
lsof -i :5173

# Kill process (note the PID fromabove output)
kill -9 <PID>
```

### Backend Won't Start

Check backend logs:
```bash
./docker-mac-local.sh logs backend
```

Common issues:
- **PostgreSQL not accessible**: Verify `100.84.126.19:5432` is reachable
  ```bash
  nc -zv 100.84.126.19 5432
  ```
- **Missing Go modules**: Run in backend directory
  ```bash
  go mod download
  ```
- **Compilation errors**: Review error output

### Auth Service Won't Start

Check auth service logs:
```bash
./docker-mac-local.sh logs auth-service
```

Common issues:
- **PostgreSQL not accessible**: Verify `100.84.126.19:5432`
- **JWT_SECRET not configured**: Check environment variables
- **CORS issues**: Check `ALLOWED_ORIGINS` env var matches your host

### Frontend Dev Server Won't Start

Check console output:
```bash
cd frontend && sh scripts/start-dev.sh
```

Common issues:
- **Port 5173 already in use**: The script kills existing processes
  - If still fails, manually: `lsof -i :5173` and `kill -9 <PID>`
- **Missing npm dependencies**: 
  ```bash
  cd frontend && npm install
  ```
- **Vite config issues**: Check `frontend/vite.config.ts`
- **Node modules incompatibility**: Sometimes cache needs clearing
  ```bash
  cd frontend && rm -rf node_modules package-lock.json && npm install
  ```

### External Services Not Reachable

If backend/auth can't reach the external databases/GraphQL:

```bash
# Test PostgreSQL connectivity
nc -zv 100.84.126.19 5432

# Test Hasura connectivity  
curl -I http://100.84.126.19:8080/v1/graphql

# Check DNS resolution
nslookup 100.84.126.19
```

### Docker Desktop Performance

If Docker services are slow:
1. Open Docker Desktop preferences
2. Go to Resources
3. Increase CPU cores and memory allocation
4. Set file sharing to "native" for better performance

## Development Workflow

### 1. Backend Changes

Since backend is in a container, changes require rebuild:

```bash
# Edit Go code
cd backend && vim...

# Rebuild image
docker compose -f docker-compose.mac-local.yml build backend

# Restart service
docker compose -f docker-compose.mac-local.yml restart backend

# Check logs
./docker-mac-local.sh logs backend
```

### 2. Frontend Changes

Frontend runs locally with hot reload - no restart needed:

```bash
# Edit React/TypeScript files
cd frontend && vim src/...

# Browser auto-updates instantly!
# No rebuild, no restart needed

# If hot reload doesn't work, hard refresh browser (Cmd+Shift+R)
```

### 3. Auth Service Changes

Auth service runs in Docker with volumes mounted:

```bash
# Edit Node.js code
cd auth-service && vim server.js

# Restart service to pick up changes
docker compose -f docker-compose.mac-local.yml restart auth-service

# Check logs
./docker-mac-local.sh logs auth-service
```

## Building Only (Without Docker Compose)

### Build Backend Binary Locally

```bash
cd /Users/eganpj/GitHub/semlayer/backend

# Build for macOS (ARM64)
GOOS=darwin GOARCH=arm64 go build -o server cmd/server/main.go

# Build for Linux (in container)
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o server cmd/server/main.go
```

### Build Docker Image Only

```bash
cd /Users/eganpj/GitHub/semlayer

# Build images without running
docker compose -f docker-compose.mac-local.yml build

# Build single service
docker compose -f docker-compose.mac-local.yml build backend
```

## Production Deployment

For production deployment, use:
- `docker-compose.yml` (full stack)
- `docker-compose.remote.yml` (remote deployment)
- Kubernetes manifests in `k8s/` directory

See [DEPLOYMENT.md](./DEPLOYMENT.md) for details.

## Environment Variables

Override default values by setting environment variables:

```bash
# Before starting
export POSTGRES_PASSWORD=mypassword
export JWT_SECRET=my-secret-key
export HASURA_ADMIN_SECRET=my-hasura-secret

./docker-mac-local.sh
```

Or create `.env.docker`:

```bash
POSTGRES_HOST=100.84.126.19
POSTGRES_PORT=5432
JWT_SECRET=my-jwt-secret
HASURA_ADMIN_SECRET=my-hasura-admin-secret
```

Then load it:
```bash
docker compose -f docker-compose.mac-local.yml --env-file .env.docker up -d
```

## Health Checks

Each service includes health checks:

```bash
# Check health
docker compose -f docker-compose.mac-local.yml ps

# Or manually
curl http://localhost:8080/health       # Backend
curl http://localhost:8001/health       # Auth Service
curl http://localhost:5173/            # Frontend
```

## Support

For issues:
1. Check logs: `./docker-mac-local.sh logs`
2. Verify external services are accessible
3. Ensure Docker resources are sufficient
4. Review the troubleshooting section above

## Next Steps

After deployment:
1. ✅ Access http://localhost:5173
2. ✅ Login with test@example.com / password123
3. ✅ Navigate to Glossary page
4. ✅ Select uisce tenant and northwinds datasource
5. ✅ Create semantic terms and business objects
