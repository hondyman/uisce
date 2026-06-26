# SemLayer Distributed Platform - Startup Checklist

**Print this page and check off items as you complete them!**

---

## Pre-Startup Phase

### Remote Machine (100.84.126.19) - Have Ready

- [ ] SSH access verified: `ssh user@100.84.126.19`
- [ ] Docker is running on remote machine
- [ ] Remote compose file exists: `docker-compose.remote.yml`
- [ ] Network connectivity from MacBook to 100.84.126.19: `ping 100.84.126.19`

### MacBook Pro - Have Ready

- [ ] Docker Desktop installed and accessible: `which docker`
- [ ] Docker Compose v2+: `docker compose version`
- [ ] Node.js v18+: `node --version`
- [ ] npm v9+: `npm --version`
- [ ] Terminal with access to /Users/eganpj/GitHub/semlayer

---

## Startup Phase 1: Start Remote Services

**On Remote Machine (100.84.126.19)**

```bash
cd /path/to/semlayer
docker compose -f docker-compose.remote.yml up -d
```

- [ ] Command executed successfully
- [ ] No errors in output
- [ ] Waited for command to complete (2-3 minutes)

**Verify Remote Services Started**

```bash
docker compose -f docker-compose.remote.yml ps
```

Check all services show "Up" status:

- [ ] `postgres` - Up
- [ ] `hasura` - Up
- [ ] `redpanda` - Up
- [ ] `temporal` - Up
- [ ] `debezium` - Up
- [ ] `trino` - Up (optional)
- [ ] `minio` - Up (optional)

---

## Startup Phase 2: Test MacBook Connectivity

**On MacBook Pro - Terminal 1**

```bash
cd /Users/eganpj/GitHub/semlayer
./test-distributed-connectivity.sh
```

- [ ] Script executed successfully
- [ ] Reviewed test output
- [ ] Note any failed tests: _________________

**Critical Tests to Pass**

- [ ] PostgreSQL connection: ✓ Connected
- [ ] Hasura GraphQL: ✓ Responding
- [ ] Redpanda cluster: ✓ Healthy
- [ ] Docker daemon running: ✓ or start Docker Desktop

**If Tests Failed**

If you see failures, check:

- [ ] Network connectivity: `ping 100.84.126.19` (should complete)
- [ ] Remote services status: SSH to remote and run `docker compose ps`
- [ ] Firewall allowing ports (unlikely on local network)
- [ ] Docker Desktop running on MacBook

---

## Startup Phase 3: Verify Environment Configuration

**Check .env File**

```bash
grep -E "DB_HOST|HASURA_URL|KAFKA_BROKERS|TEMPORAL_HOSTPORT" .env
```

Expected output should show:

```
DB_HOST=100.84.126.19
HASURA_URL=http://100.84.126.19:8085
KAFKA_BROKERS=100.84.126.19:19092
TEMPORAL_HOSTPORT=100.84.126.19:7233
```

- [ ] DB_HOST configured for remote IP
- [ ] HASURA_URL configured for remote IP
- [ ] KAFKA_BROKERS configured for remote IP (port 19092!)
- [ ] TEMPORAL_HOSTPORT configured for remote IP
- [ ] ALLOWED_ORIGINS includes localhost:5173

**If Configuration is Wrong**

Update `.env`:

```bash
# Edit .env and change these lines:
DB_HOST=100.84.126.19
DATABASE_URL=postgresql://postgres:postgres@100.84.126.19:5432/alpha
HASURA_URL=http://100.84.126.19:8085
KAFKA_BROKERS=100.84.126.19:19092
TEMPORAL_HOSTPORT=100.84.126.19:7233
```

- [ ] Configuration file updated
- [ ] All remote IP addresses corrected

---

## Startup Phase 4: Start Backend

**On MacBook Pro - Terminal 1**

```bash
./start-distributed-platform.sh
```

Expected flow:

```
✓ Testing connectivity to remote services...
✓ PostgreSQL connection verified
✓ Hasura verified
✓ Redpanda verified
✓ Docker daemon running
✓ Building backend image...
✓ Starting backend container...
✓ Waiting for backend to be ready...
✓ Backend is healthy and responding

Backend is ready!
Access at: http://localhost:8080

Next: Start frontend in a new terminal
```

- [ ] Script started successfully
- [ ] Building phase completed (3-5 minutes)
- [ ] Container started
- [ ] Health check passed (✓ Backend is healthy)
- [ ] Output shows: "Backend is ready!"

**If Backend Won't Start**

Check logs:

```bash
docker compose -f docker-compose.mac-distributed.yml logs backend
```

Common issues:

- [ ] Database connection error → Verify PostgreSQL reachable: `nc -zv 100.84.126.19 5432`
- [ ] Docker daemon not running → Start Docker Desktop
- [ ] Build error → Run `docker compose -f docker-compose.mac-distributed.yml rebuild --no-cache`

---

## Startup Phase 5: Start Frontend

**On MacBook Pro - Terminal 2 (New Terminal)**

```bash
cd frontend
npm install  # Only if first time
npm run dev
```

Expected output:

```
VITE v5.x.x ready in xxx ms

➜  Local:   http://localhost:5173/
➜  press h to show help
```

- [ ] npm started successfully
- [ ] Frontend bundling completed
- [ ] "Local: http://localhost:5173/" is displayed
- [ ] No errors in output

---

## Startup Phase 6: Verification in Browser

**Open Browser**

```
http://localhost:5173
```

- [ ] Page loads without errors
- [ ] Console (F12) shows no critical errors
- [ ] Browser can reach frontend
- [ ] Dashboard visible (if authenticated)

**Test API Connectivity**

In browser console (F12):

```javascript
fetch('http://localhost:8080/health')
  .then(r => r.json())
  .then(console.log)
```

Expected result: `{status: "healthy"}`

- [ ] API call succeeds
- [ ] Backend responds with health status
- [ ] No CORS errors in console
- [ ] Network tab shows 200 responses

---

## Startup Phase 7: Full System Test

### Test Database Connectivity

```bash
curl -X GET http://localhost:8080/api/dashboard/metrics \
  -H "Authorization: Bearer YOUR_TOKEN"
```

- [ ] Request completes
- [ ] Returns JSON response (not HTML error)
- [ ] No connection errors

### Test Real API Endpoint

In browser, navigate to a dashboard page:

- [ ] Page loads
- [ ] Data displays
- [ ] Charts render
- [ ] No console errors

### Check All Service Endpoints

Test each endpoint is accessible:

- [ ] Backend health: `curl http://localhost:8080/health` → `{status: "healthy"}`
- [ ] Hasura: `curl http://100.84.126.19:8085/healthz` → Should return OK
- [ ] Kafka: `nc -zv 100.84.126.19 19092` → Should connect
- [ ] Temporal: `curl http://100.84.126.19:8088` → Should load UI

Endpoint status:

- [ ] Backend :8080 - Working
- [ ] Frontend :5173 - Working
- [ ] Hasura :8085 - Working (remote)
- [ ] Kafka :19092 - Working (remote)
- [ ] Temporal UI :8088 - Working (remote)

---

## Success Criteria

✅ **You've successfully set up the distributed platform when you have:**

- [ ] Remote services running on 100.84.126.19
- [ ] Connectivity test passed all checks
- [ ] Backend container running and healthy
- [ ] Frontend server running without errors
- [ ] Browser loads http://localhost:5173 successfully
- [ ] API calls from browser to backend succeed
- [ ] Dashboard displays data from database
- [ ] All services responding on expected ports
- [ ] Console (F12) shows no critical errors
- [ ] Network tab shows all requests succeeding (200 status)

---

## Troubleshooting Shortcuts

**If Something Fails, Try These:**

### Backend Won't Start

```bash
# Check status
docker compose -f docker-compose.mac-distributed.yml ps

# View logs
docker compose -f docker-compose.mac-distributed.yml logs backend

# Rebuild from scratch
docker compose -f docker-compose.mac-distributed.yml down
docker compose -f docker-compose.mac-distributed.yml build --no-cache
docker compose -f docker-compose.mac-distributed.yml up -d backend

# Check if port is in use
lsof -i :8080
```

- [ ] Issue resolved

### Frontend Won't Connect to Backend

```bash
# Check backend is running
curl http://localhost:8080/health

# Check browser console (F12) for CORS errors
# Check .env for ALLOWED_ORIGINS setting
grep ALLOWED_ORIGINS .env

# May need to update CORS
# ALLOWED_ORIGINS=http://localhost:5173,http://127.0.0.1:5173
```

- [ ] Issue resolved

### Can't Reach Remote Services

```bash
# Test network
ping 100.84.126.19

# Test specific ports
nc -zv 100.84.126.19 5432    # PostgreSQL
nc -zv 100.84.126.19 8085    # Hasura
nc -zv 100.84.126.19 19092   # Kafka

# If all fail, restart remote services
ssh user@100.84.126.19
docker compose -f docker-compose.remote.yml up -d
```

- [ ] Issue resolved

### Need to See What's Running

```bash
# Local MacBook
docker ps
docker stats

# Remote machine
ssh user@100.84.126.19
docker compose -f docker-compose.remote.yml ps
docker stats
```

- [ ] Got the information needed

### Need to Stop Everything

```bash
# Stop MacBook services
docker compose -f docker-compose.mac-distributed.yml down

# Stop remote services (SSH first)
ssh user@100.84.126.19
docker compose -f docker-compose.remote.yml down
```

- [ ] Services stopped

---

## Common Port Issues

If you get "port already in use" errors:

- [ ] Port 8080 (Backend): `lsof -i :8080` → `kill -9 <PID>`
- [ ] Port 5173 (Frontend): `lsof -i :5173` → `kill -9 <PID>`
- [ ] Port 5174 (Frontend alt): `lsof -i :5174` → `kill -9 <PID>`

---

## Performance Baseline

After startup, expected metrics:

- [ ] Backend health check: < 100ms
- [ ] API response time: < 500ms (first call may be 1-2s)
- [ ] Frontend page load: < 2 seconds
- [ ] Network latency to 100.84.126.19: < 50ms

---

## Final Notes

**Date Started:** ________________

**Date Completed:** ________________

**Issues Encountered:** 

```
_____________________________________________________________

_____________________________________________________________

_____________________________________________________________
```

**Resolution:** 

```
_____________________________________________________________

_____________________________________________________________

_____________________________________________________________
```

**Contacts for Help:**

- Backend Issues: Check `docker logs`
- Frontend Issues: Check browser console (F12)
- Network Issues: Use `./test-distributed-connectivity.sh`
- Reference: Run `./print-reference-card.sh`

---

## What To Do Next

Once platform is running:

- [ ] Read [DISTRIBUTED_PLATFORM_SETUP.md](DISTRIBUTED_PLATFORM_SETUP.md) for advanced configuration
- [ ] Run end-to-end tests in the backend
- [ ] Load test the platform with realistic data
- [ ] Set up monitoring and alerting
- [ ] Configure backups
- [ ] Plan security hardening for production

---

**✅ Checklist Complete!** 

Your SemLayer distributed platform is now running across your MacBook Pro and remote server. Happy coding! 🚀
