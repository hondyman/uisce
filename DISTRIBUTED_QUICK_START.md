# DISTRIBUTED PLATFORM - QUICK START GUIDE

## 🚀 30-Second Setup (MacBook)

### Prerequisites: Remote services running on 100.84.126.19
```bash
# On remote machine
docker compose -f docker-compose.remote.yml up -d
```

### On MacBook Pro

```bash
# 1. Test connectivity
./test-distributed-connectivity.sh

# 2. Start backend (builds + runs Docker)
./start-distributed-platform.sh

# 3. Start frontend (new terminal window)
cd frontend
npm run dev

# 4. Open browser
open http://localhost:5173
```

---

## 📋 Architecture

```
MACBOOK PRO                          REMOTE (100.84.126.19)
┌─────────────────────┐              ┌──────────────────────┐
│ Frontend            │              │ PostgreSQL (5432)    │
│ (npm dev)           │              │ Hasura (8085)        │
│ :5173               │◄─────TCP────►│ Redpanda (19092)     │
│                     │              │ Temporal (7233)      │
│ Backend             │              │ Debezium (8083)      │
│ (Docker)            │              │ Trino (8094)         │
│ :8080               │              │ MinIO (9010)         │
└─────────────────────┘              └──────────────────────┘
```

---

## 🔍 Check Services

### Remote Services Status
```bash
ssh user@100.84.126.19
docker compose -f docker-compose.remote.yml ps
```

### Local Backend Status
```bash
docker compose -f docker-compose.mac-distributed.yml ps
docker compose -f docker-compose.mac-distributed.yml logs backend
```

### Test Backend Connectivity
```bash
curl http://localhost:8080/health
# Response: {"status":"ok"}
```

---

## 🌐 Service URLs

| Service | URL | 
|---------|-----|
| Frontend | http://localhost:5173 |
| Backend API | http://localhost:8080 | 
| Hasura GraphQL | http://100.84.126.19:8085 |
| Redpanda Console | http://100.84.126.19:8096 |
| Temporal UI | http://100.84.126.19:8088 |
| MinIO Console | http://100.84.126.19:9011 |

---

## 🛠 Common Commands

```bash
# Start everything
./start-distributed-platform.sh

# Stop backend
docker compose -f docker-compose.mac-distributed.yml down

# View backend logs
docker compose -f docker-compose.mac-distributed.yml logs -f backend

# Restart backend
docker compose -f docker-compose.mac-distributed.yml restart backend

# Rebuild backend image
docker compose -f docker-compose.mac-distributed.yml build --no-cache

# Test remote connectivity
./test-distributed-connectivity.sh
```

---

## ⚠️ Troubleshooting

### Backend won't start
```bash
# Check Docker is running
docker ps

# Check remote services accessible
nc -zv 100.84.126.19 5432

# Check logs
docker compose -f docker-compose.mac-distributed.yml logs backend
```

### Frontend can't reach backend
```bash
# Test from browser console
fetch('http://localhost:8080/health').then(r => r.json()).then(console.log)

# Verify backend running
curl http://localhost:8080/health
```

### Remote service not reachable
```bash
# Test connectivity
ping 100.84.126.19
telnet 100.84.126.19 5432

# Verify remote services running
ssh user@100.84.126.19
docker compose -f docker-compose.remote.yml ps
```

---

## 📝 Configuration

### .env File (MacBook)
```bash
DB_HOST=100.84.126.19
DB_PORT=5432
HASURA_URL=http://100.84.126.19:8085
KAFKA_BROKERS=100.84.126.19:19092
TEMPORAL_HOSTPORT=100.84.126.19:7233
ALLOWED_ORIGINS=http://localhost:5173,http://127.0.0.1:5173
```

---

## 📚 Full Documentation

See [DISTRIBUTED_PLATFORM_SETUP.md](./DISTRIBUTED_PLATFORM_SETUP.md) for complete setup guide.

---

## 🎯 Next Steps

1. ✅ Verify remote services running on 100.84.126.19
2. ✅ Run `./test-distributed-connectivity.sh` 
3. ✅ Run `./start-distributed-platform.sh`
4. ✅ Run `npm run dev` in frontend directory
5. ✅ Open http://localhost:5173

---

## 💡 Key Points

- **Remote IP**: 100.84.126.19 (all data services)
- **Local**: localhost (backend & frontend on Mac)
- **Network**: TCP/IP connectivity required between locations
- **Performance**: Depends on network latency to remote host

---

**Status**: ✅ Ready to deploy
**Last Updated**: February 2026
