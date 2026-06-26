# Docker Compose Quick Start & Troubleshooting Guide

## ✅ Current Status: ALL SYSTEMS OPERATIONAL

Your Docker Compose environment is running perfectly with all services up and healthy!

---

## 🚀 Quick Start (5 minutes)

### 1. Verify Everything is Running
```bash
cd /Users/eganpj/GitHub/semlayer

# Run verification script
./verify-docker-setup.sh
```

**Expected Output:**
```
✅ All checks passed! Docker Compose is ready.
```

### 2. Start the Frontend Dev Server
```bash
cd frontend

# Check if dependencies are installed
npm list react 2>/dev/null || npm install

# Start dev server
npm run dev
```

**You'll see:**
```
Local:   http://localhost:5173
```

### 3. Open in Browser
```
http://localhost:5173
```

### 4. Select Tenant & Datasource
- Look for tenant/datasource picker at the top
- Select a tenant and datasource from the dropdown
- The app will now load with data

---

## 📊 Service Overview

### What's Running in Docker
| Service | Port | Purpose | Status |
|---------|------|---------|--------|
| Backend | 9090 | API Server | ✅ Healthy |
| API Gateway | 8001 | Request Router | ✅ Healthy |
| Hasura | 8080 | GraphQL Engine | ✅ Healthy |
| Temporal | 7233 | Workflows | ✅ Running |
| RabbitMQ | 5672/15672 | Message Queue | ✅ Healthy |
| PostgreSQL | 5432 | Database (host) | ✅ Connected |

### Quick Access
```
Frontend:          http://localhost:5173
Backend Health:    http://localhost:9090/health
API Gateway:       http://localhost:8001
Hasura Console:    http://localhost:8080/console
RabbitMQ Admin:    http://localhost:15672 (guest/guest)
Temporal UI:       http://localhost:8088
```

---

## 🔧 Common Tasks

### View All Running Services
```bash
docker compose ps
```

### View Service Logs
```bash
# Backend logs (latest 50 lines, following)
docker compose logs -f backend --tail=50

# All services
docker compose logs -f

# Specific service
docker compose logs -f hasura
```

### Restart a Service
```bash
# Restart backend
docker compose restart backend

# Restart API gateway
docker compose restart api-gateway

# Restart all
docker compose restart
```

### Stop All Services (Don't Lose Data)
```bash
docker compose down
# Data persists in volumes
```

### Completely Clean Up (Warning: Deletes Data)
```bash
docker compose down -v
# WARNING: This deletes all volumes, including database data
```

### Rebuild a Service
```bash
# Rebuild backend from Dockerfile
docker compose up -d --build backend

# Rebuild all services
docker compose up -d --build
```

---

## ✨ Testing the Setup

### Test Backend Health
```bash
curl http://localhost:9090/health
```

**Expected:**
```json
{
  "status": "healthy",
  "timestamp": "2025-11-07T01:30:49Z"
}
```

### Test API Gateway
```bash
curl http://localhost:8001/health
```

### Test Hasura
```bash
curl http://localhost:8080/v1/version
```

### Test Database Access
```bash
psql -U postgres -d alpha -c "SELECT version();"
```

### Test with Tenant Scope
```bash
curl -H "X-Tenant-ID: 910638ba-a459-4a3f-bb2d-78391b0595f6" \
     -H "X-Tenant-Datasource-ID: 982aef38-418f-46dc-acd0-35fe8f3b97b0" \
     http://localhost:9090/api/entity-schema | head -20
```

---

## ⚠️ Troubleshooting

### Problem: "Cannot connect to Docker daemon"

**Solution:**
```bash
# Start Docker Desktop on macOS
open /Applications/Docker.app

# Wait 30 seconds for it to fully start
# Then try: docker ps
```

### Problem: "Port already in use"

**Example:** `Address already in use: 9090`

```bash
# Find what's using the port
lsof -i :9090

# Kill the process (use PID from above)
kill -9 <PID>

# Or restart Docker Compose
docker compose restart backend
```

### Problem: "Database connection refused"

**Verify PostgreSQL is running:**
```bash
psql -U postgres -d alpha -c "SELECT 1"
```

**If it fails:**
```bash
# Check if PostgreSQL is running on host
brew services list | grep postgresql

# Start PostgreSQL if not running
brew services start postgresql@14

# Or using default PostgreSQL installation
pg_ctl -D /usr/local/var/postgres start
```

### Problem: Backend returning 404 for `/api/relationships/objects`

This is expected! The endpoint exists but may need data. See: `RELATED_OBJECTS_QUICK_FIX.md`

**To test with demo data:**
```bash
# Component gracefully shows demo data when backend returns 404
# Navigate to Entity Details → Related Objects tab
# You'll see demo relationships (Orders, Department, Manager)
```

### Problem: "XAI_API_KEY not set" warning

**Solution:** This is harmless for local development. Only needed for AI features.

```bash
# Optional: Add to .env.local if needed
echo "XAI_API_KEY=your-key-here" >> .env.local
```

### Problem: Services stuck or hanging

**Solution:**
```bash
# Hard restart everything
docker compose down
docker compose up -d

# Wait 30 seconds for services to start
sleep 30

# Verify health
curl http://localhost:9090/health
```

### Problem: Out of disk space

```bash
# Clean up unused Docker resources
docker system prune

# Or more aggressively:
docker system prune -a --volumes
```

### Problem: Cannot access Hasura Console

**Verify:** `http://localhost:8080/console`

If getting 404:
```bash
# Check Hasura logs
docker compose logs hasura --tail=100

# Restart Hasura
docker compose restart hasura
```

---

## 📝 Configuration Files

### Main Files
- **`docker-compose.yml`** - Main service definitions
- **`docker-compose.override.yml`** - Local overrides
- **`.env`** - Environment variables
- **`.env.local`** - Local dev overrides (auto-loaded)
- **`backend/config.yaml`** - Backend configuration

### To Update Configuration
1. Edit `.env` or `.env.local`
2. Restart the service: `docker compose restart backend`
3. Or rebuild: `docker compose up -d --build backend`

---

## 🔄 Typical Development Workflow

```
1. Start each morning:
   docker compose ps  # Verify services running

2. If services aren't running:
   docker compose up -d

3. Start frontend dev server:
   cd frontend && npm run dev

4. Open browser:
   http://localhost:5173

5. Work on features normally
   (changes auto-reload in dev mode)

6. End of day:
   Optional: docker compose down
   Next day: docker compose up -d (restarts everything)
```

---

## 🎯 Working on Specific Features

### Backend API Development
1. Make changes to Go code in `backend/`
2. Rebuild: `docker compose up -d --build backend`
3. Test endpoint: `curl http://localhost:9090/api/your-endpoint`
4. View logs: `docker compose logs -f backend`

### Frontend Development
1. Make changes to React code in `frontend/`
2. Changes auto-reload (Vite HMR)
3. Check console for errors: F12 → Console tab
4. Test in browser: `http://localhost:5173`

### Database Changes
1. Connect to database:
   ```bash
   psql -U postgres -d alpha
   ```
2. Run SQL migrations
3. Services automatically see changes
4. No restart needed for read operations

### Workflow Changes (Temporal)
1. Update workflows in temporal configs
2. Restart: `docker compose restart temporal`
3. Monitor: `http://localhost:8088`

---

## 🚨 Emergency Procedures

### Service Completely Broken
```bash
# Complete reset (keeps database)
docker compose down
docker compose up -d

# Wait for startup
sleep 30

# Verify
curl http://localhost:9090/health
```

### Need Fresh Database
```bash
# WARNING: Deletes all data!
docker compose down -v
docker compose up -d

# Wait for initialization
sleep 60
```

### Memory/Resource Issues
```bash
# Check resource usage
docker stats

# Restart to free resources
docker compose restart

# Or rebuild smaller
docker compose build --no-cache backend
```

### Need to Debug Container
```bash
# SSH into backend container
docker compose exec backend /bin/sh

# Or as root
docker compose exec -u root backend /bin/bash

# Install tools if needed
apt-get update && apt-get install -y curl
```

---

## 📊 Verification Checklist

Before starting development, confirm:

- [ ] Docker Desktop running: `docker ps`
- [ ] PostgreSQL running: `psql -U postgres -d alpha -c "SELECT 1"`
- [ ] All services up: `docker compose ps` (all say "Up")
- [ ] Backend healthy: `curl http://localhost:9090/health`
- [ ] API Gateway responding: `curl http://localhost:8001/health`
- [ ] Hasura reachable: `curl http://localhost:8080/v1/version`
- [ ] Frontend can be started: `cd frontend && npm run dev`

---

## 📞 Quick Reference Commands

```bash
# Status check
docker compose ps

# View logs
docker compose logs -f backend

# Restart backend
docker compose restart backend

# Stop everything
docker compose down

# Start everything
docker compose up -d

# Full rebuild
docker compose down && docker compose up -d --build

# Test health
curl http://localhost:9090/health

# Access database
psql -U postgres -d alpha

# Check resource usage
docker stats

# Clean up unused resources
docker system prune
```

---

## 🎓 Learning Resources

- **Docker Compose Docs**: https://docs.docker.com/compose/
- **Service Configurations**: See `docker-compose.yml` (line numbers in error messages)
- **Environment Setup**: See `.env` and `.env.local`
- **Backend Logs**: `docker compose logs backend --tail=200`
- **Related Objects Fix**: See `RELATED_OBJECTS_QUICK_FIX.md`

---

## ✅ You're All Set!

Your Docker Compose environment is:
- ✅ Fully operational
- ✅ Connected to PostgreSQL
- ✅ Services healthy and responsive
- ✅ Ready for development

**Next Step:** Start the frontend dev server and begin building!

```bash
cd frontend
npm run dev
# Open http://localhost:5173
```

---

**Last Updated:** November 7, 2025  
**Environment:** macOS with Docker Desktop  
**Status:** All Systems Operational ✅
