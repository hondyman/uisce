# Distributed Platform - First-Time Setup Verification Checklist

## Pre-Launch Checklist

Before running the platform, verify all prerequisites are met:

### ✓ Remote Machine (100.84.126.19) Setup

- [ ] PostgreSQL running natively
  ```bash
  ssh user@100.84.126.19
  sudo systemctl status postgresql
  # or
  psql -V
  psql postgresql://postgres:postgres@localhost:5432/alpha -c "SELECT 1"
  ```

- [ ] Docker & Docker Compose installed
  ```bash
  docker -v
  docker compose version
  ```

- [ ] Remote services started
  ```bash
  docker compose -f docker-compose.remote.yml ps
  ```

- [ ] Firewall allows incoming connections on ports:
  - 5432 (PostgreSQL)
  - 8085 (Hasura)
  - 19092 (Redpanda external)
  - 7233 (Temporal)
  - 8083 (Debezium)
  - 9010 (MinIO API)
  - 9011 (MinIO Console)

### ✓ MacBook Setup

- [ ] Docker Desktop installed and running
  ```bash
  docker ps
  docker compose version
  ```

- [ ] Network connectivity to remote
  ```bash
  ping -c 3 100.84.126.19
  ```

- [ ] Node.js 18+ installed
  ```bash
  node -v
  npm -v
  ```

- [ ] Git cloned and updated
  ```bash
  cd /path/to/semlayer
  git status
  ```

- [ ] .env file configured
  ```bash
  grep "DB_HOST" .env
  # Should show: DB_HOST=100.84.126.19
  ```

---

## Step-by-Step First Run

### Step 1: Connectivity Test (5 minutes)

```bash
cd /path/to/semlayer

# Run connectivity test
./test-distributed-connectivity.sh 100.84.126.19
```

**Expected Output:**
```
✓ PostgreSQL (100.84.126.19:5432)... OK
✓ Hasura GraphQL (100.84.126.19:8085)... OK
✓ Redpanda Kafka (100.84.126.19:19092)... OK
✓ Temporal (100.84.126.19:7233)... OK
...
✓ All tests passed! Your platform is ready to start.
```

**If tests fail:**
- Check .env has remote IP: `100.84.126.19`
- Verify remote services running: `ssh user@100.84.126.19 docker compose -f docker-compose.remote.yml ps`
- Check network connectivity: `ping 100.84.126.19`
- Check firewall isn't blocking ports

### Step 2: Build Backend Image (10 minutes)

```bash
# Build the backend Docker image
docker compose -f docker-compose.mac-distributed.yml build --no-cache
```

**Expected:**
```
[+] Building 2.3s (15/15) FINISHED
 => [internal] load build definition from backend/Dockerfile
 => [external] base image golang:1.21
 => ...
 => => writing image sha256:abc123...
 => => naming to docker.io/library/semlayer-backend:latest
```

**If build fails:**
- Check Go version: `go version` (should be 1.21+)
- Check Dockerfile exists: `ls -la backend/Dockerfile`
- View full logs: `docker compose -f docker-compose.mac-distributed.yml build --no-cache 2>&1 | tail -50`

### Step 3: Start Backend (5 minutes)

```bash
# Start backend container
docker compose -f docker-compose.mac-distributed.yml up -d

# Monitor startup
docker compose -f docker-compose.mac-distributed.yml logs -f backend
```

**Expected logs:**
```
backend    | 2026/02/22 10:30:15 Backend server starting on port 8080
backend    | 2026/02/22 10:30:16 Database connected: postgres://100.84.126.19:5432/alpha
backend    | 2026/02/22 10:30:17 Hasura URL configured: http://100.84.126.19:8085
backend    | 2026/02/22 10:30:20 Health check passed
```

**If backend fails to start:**
- Check container status: `docker compose -f docker-compose.mac-distributed.yml ps`
- Check logs: `docker compose -f docker-compose.mac-distributed.yml logs backend | head -50`
- Restart: `docker compose -f docker-compose.mac-distributed.yml restart backend`

### Step 4: Verify Backend Health (2 minutes)

```bash
# Test health endpoint
curl -v http://localhost:8080/health

# Test database connectivity
curl -v http://localhost:8080/api/ping

# Or in browser
open http://localhost:8080/health
```

**Expected response:**
```json
{"status":"ok","timestamp":"2026-02-22T10:30:25Z"}
```

**If health check fails:**
- Check container is running: `docker ps | grep backend`
- Check logs: `docker compose -f docker-compose.mac-distributed.yml logs backend | tail -20`
- Verify port 8080 is accessible: `lsof -i :8080`
- Wait for startup: `sleep 10 && curl http://localhost:8080/health`

### Step 5: Start Frontend (3 minutes)

In a **NEW terminal window**:

```bash
cd /path/to/semlayer/frontend

# Install dependencies (first time only)
npm install

# Start dev server
npm run dev
```

**Expected output:**
```
  VITE v5.0.0  ready in 245 ms

  ➜  Local:   http://localhost:5173/
  ➜  press h + enter to show help
```

**If frontend fails:**
- Check Node version: `node -v` (should be 18+)
- Clear node_modules: `rm -rf node_modules && npm install`
- Check port 5173 is free: `lsof -i :5173`
- Check .env has correct API URL:
  ```bash
  grep VITE_API_BASE_URL .env*
  # Should show: VITE_API_BASE_URL=http://localhost:8080
  ```

### Step 6: Access Application (1 minute)

Open browser:
```
http://localhost:5173
```

**Expected:**
- Login page or dashboard loads
- No console errors in browser dev tools
- Network requests to `http://localhost:8080` succeed

**If page doesn't load:**
- Check browser console for errors (F12)
- Check network tab for failed requests
- Verify backend is running: `curl http://localhost:8080/health`
- Check CORS configuration
- Try hard refresh: `Cmd+Shift+R` (Mac)

---

## Verification Checklist - After Launch

Once everything is running, verify all components are working:

### Backend Verification
```bash
# Should return 200 OK
curl http://localhost:8080/health

# Should return 200 OK
curl http://localhost:8080/api/ping

# Check database connection
docker compose -f docker-compose.mac-distributed.yml logs backend | grep -i "database\|connected"
```

### Frontend Verification
```bash
# Check if dev server is running
curl http://localhost:5173 | head -20

# Should show HTML response
```

### Remote Services Verification
```bash
# PostgreSQL
PGPASSWORD=postgres psql -h 100.84.126.19 -U postgres -d alpha -c "SELECT 1"
# Expected: (1 row) 1

# Hasura
curl -s http://100.84.126.19:8085/v1/version | jq .
# Expected: {"version": "v2.x.x"}

# Redpanda
curl -s http://100.84.126.19:8082/brokers | head -20
# Expected: broker info in JSON

# Temporal
curl -s http://100.84.126.19:7233/api/v1/namespaces/default | head -20
# Expected: namespace info
```

---

## Common First-Time Issues & Fixes

### Issue: "Connection refused" to 100.84.126.19
**Cause:** Network connectivity issue or remote services not running

**Fix:**
```bash
# Test connectivity
ping 100.84.126.19

# Verify remote services running
ssh user@100.84.126.19 docker compose -f docker-compose.remote.yml ps

# If not running, start them
ssh user@100.84.126.19
cd /path/to/semlayer
docker compose -f docker-compose.remote.yml up -d
```

### Issue: Backend container exits immediately
**Cause:** Database connection failed

**Fix:**
```bash
# Check logs
docker compose -f docker-compose.mac-distributed.yml logs backend

# Verify database is accessible
psql postgresql://postgres:postgres@100.84.126.19:5432/alpha

# Check .env DATABASE_URL
grep DATABASE_URL .env

# If connection string wrong, update .env and rebuild
docker compose -f docker-compose.mac-distributed.yml build --no-cache
docker compose -f docker-compose.mac-distributed.yml up -d
```

### Issue: Frontend shows blank page
**Cause:** Backend not accessible from frontend

**Fix:**
```bash
# Verify backend is running
docker ps | grep backend

# Test connectivity from browser console
fetch('http://localhost:8080/health').then(r => r.json()).then(console.log)

# Check CORS settings in backend
docker compose -f docker-compose.mac-distributed.yml logs backend | grep -i cors

# Verify .env has correct origins
grep ALLOWED_ORIGINS .env
```

### Issue: Port 8080 already in use
**Cause:** Another service using port 8080

**Fix:**
```bash
# Find what's using port 8080
lsof -i :8080

# Kill the process
kill -9 <PID>

# Or change port in docker-compose.mac-distributed.yml
# Edit: ports: ["3000:8080"] (use local port 3000 instead)
```

### Issue: npm install hangs or fails
**Cause:** Network issue or locked package file

**Fix:**
```bash
# Clear cache
npm cache clean --force

# Remove lock file
rm package-lock.json

# Reinstall
npm install --legacy-peer-deps
```

---

## Performance Baselines

After successful startup, here are expected performance metrics:

| Operation | Expected Time |
|-----------|---------|
| Backend startup | 10-20 seconds |
| Frontend startup | 5-10 seconds |
| Page load | 1-3 seconds |
| API response | 100-500ms (depends on network latency) |
| Database query | 50-200ms |

If significantly slower, check:
- Network latency to 100.84.126.19 (use `ping`)
- Docker resource limits (increase CPU/Memory)
- Database query performance (check query logs)

---

## Advanced Troubleshooting

### View all container details
```bash
docker compose -f docker-compose.mac-distributed.yml config | head -50
```

### Check Docker network
```bash
docker network ls
docker network inspect mac-backend
```

### Enable verbose logging
```bash
# Set log level to debug
docker compose -f docker-compose.mac-distributed.yml up -d
RUST_LOG=debug docker logs -f semlayer-backend
```

### Test specific endpoints
```bash
# With authorization
TOKEN="your-jwt-token"
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/dashboard

# With debug info
curl -v -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/dashboard
```

---

## Success Criteria

You're ready when:

✅ `./test-distributed-connectivity.sh` passes all tests  
✅ Backend container is running and healthy  
✅ Backend health endpoint returns 200 OK  
✅ Frontend dev server starts without errors  
✅ Application loads in browser at http://localhost:5173  
✅ Can interact with the application  
✅ Network requests succeed without CORS errors  

---

## Support

If you encounter other issues:

1. **Check the full setup guide**: `DISTRIBUTED_PLATFORM_SETUP.md`
2. **Review logs**: `docker compose -f docker-compose.mac-distributed.yml logs -f`
3. **Test connectivity**: `./test-distributed-connectivity.sh`
4. **Verify remote services**: `ssh user@100.84.126.19 docker compose -f docker-compose.remote.yml ps`

---

**Keep this document handy for future reference!**
