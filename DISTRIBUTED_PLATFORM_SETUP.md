# Distributed Platform Setup Guide - MacBook Architecture

## Overview

This guide walks you through setting up SemLayer with a distributed architecture:

- **Remote Machine (100.84.126.19)**: PostgreSQL, Hasura, Redpanda, Temporal, Debezium, Trino, MinIO
- **MacBook Pro**: Backend (Docker), Frontend (Native)

```
┌─────────────────────────┐
│   MACBOOK PRO           │
├─────────────────────────┤
│ Docker Services:        │
│  - Backend (8080)       │
│                         │
│ Native Services:        │
│  - Frontend (5173)      │
│  - npm dev server       │
└──────────┬──────────────┘
           │ TCP/IP Network
     ┌─────▼──────┐
     │ 100.84.126.19 (REMOTE) 
     │             │
     │ Docker Compose:
     │  - PostgreSQL (5432)
     │  - Hasura (8085)
     │  - Redpanda (19092)
     │  - Temporal (7233)
     │  - Debezium (8083)
     │  - Trino (8094)
     │  - MinIO (9010/9011)
     └─────────────┘
```

---

## Prerequisites

### On MacBook Pro
- Docker Desktop (installed and running)
- Node.js 18+ and npm
- Git
- Network access to 100.84.126.19

### On Remote Machine (100.84.126.19)
- Docker & Docker Compose
- PostgreSQL 13+ (running natively)
- All services running via `docker-compose.remote.yml`

---

## Step 1: Verify Remote Services

Ensure all services are running on the remote machine:

```bash
# SSH into remote machine and start services
ssh user@100.84.126.19

# Navigate to semlayer directory
cd /path/to/semlayer

# Start remote services (if not already running)
docker compose -f docker-compose.remote.yml up -d

# Verify services
docker compose -f docker-compose.remote.yml ps
```

Expected output:
```
SERVICE                STATUS
hasura                 Up (healthy)
redpanda               Up (healthy)
redpanda-console       Up
debezium               Up
temporal               Up
temporal-ui            Up
trino                  Up
minio                  Up
```

### Verify connectivity from MacBook:

```bash
# Test each service
telnet 100.84.126.19 5432    # PostgreSQL
telnet 100.84.126.19 8085    # Hasura
telnet 100.84.126.19 19092   # Redpanda (external Kafka)
telnet 100.84.126.19 7233    # Temporal

# Or use nc (netcat)
nc -zv 100.84.126.19 5432
nc -zv 100.84.126.19 8085
nc -zv 100.84.126.19 19092
nc -zv 100.84.126.19 7233
```

---

## Step 2: Configure Environment (.env)

Update your `.env` file on MacBook to connect to remote services:

```bash
# Database - REMOTE
DB_HOST=100.84.126.19
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=alpha
DATABASE_URL=postgresql://postgres:postgres@100.84.126.19:5432/alpha?sslmode=disable

# Hasura - REMOTE
HASURA_URL=http://100.84.126.19:8085
HASURA_ADMIN_SECRET=myadminsecret
HASURA_GRAPHQL_JWT_SECRET={"type":"HS256", "key":"dev-jwt-secret-key-change-in-production"}

# Kafka/Redpanda - REMOTE (use external port 19092, not internal 9092)
KAFKA_BROKERS=100.84.126.19:19092
REDPANDA_BROKERS=100.84.126.19:19092
KAFKA_SCHEMA_REGISTRY=http://100.84.126.19:8081

# Temporal - REMOTE
TEMPORAL_HOSTPORT=100.84.126.19:7233

# JWT
JWT_SECRET=dev-jwt-secret-key-change-in-production

# CORS - Allow frontend on MacBook
ALLOWED_ORIGINS=http://localhost:5173,http://127.0.0.1:5173

# Security
ENABLE_SECURITY=true
IP_WHITELIST_ENFORCE=false
```

---

## Step 3: Start Backend on MacBook

### Option A: Using the Startup Script (Recommended)

```bash
cd /path/to/semlayer

# Make script executable (one time)
chmod +x start-distributed-platform.sh

# Run the startup script
./start-distributed-platform.sh
```

The script will:
- ✓ Verify remote services are reachable
- ✓ Check Docker is running
- ✓ Build backend Docker image
- ✓ Start backend container
- ✓ Wait for backend to be healthy
- ✓ Provide instructions for starting frontend

### Option B: Manual Docker Compose

```bash
cd /path/to/semlayer

# Start backend
docker compose -f docker-compose.mac-distributed.yml up -d

# Monitor backend
docker compose -f docker-compose.mac-distributed.yml logs -f backend

# Wait for healthy status
docker compose -f docker-compose.mac-distributed.yml ps
```

### Verify Backend is Running

```bash
# Test health endpoint
curl http://localhost:8080/health

# Expected response:
# {"status":"ok"}

# View backend logs
docker compose -f docker-compose.mac-distributed.yml logs backend
```

---

## Step 4: Start Frontend on MacBook

In a **new terminal window**:

```bash
cd /path/to/semlayer/frontend

# Install dependencies (if not already installed)
npm install

# Start dev server
npm run dev
```

Output will show:
```
  VITE v5.x.x  ready in xxx ms

  ➜  Local:   http://localhost:5173/
  ➜  press h + enter to show help
```

### Configure Frontend Environment

Ensure frontend `.env` or `.env.local` has:

```bash
# File: frontend/.env.local or frontend/.env.development

VITE_API_BASE_URL=http://localhost:8080
VITE_BACKEND_TARGET=http://localhost:8080
VITE_HASURA_URL=http://100.84.126.19:8085
VITE_JWT_SECRET=dev-jwt-secret-key-change-in-production
```

---

## Step 5: Access the Application

Open your browser:

### Dashboard & Application
```
http://localhost:5173
```

### Remote Service UIs

| Service | URL | Credentials |
|---------|-----|-------------|
| Hasura GraphQL | `http://100.84.126.19:8085` | Admin Secret: `myadminsecret` |
| Redpanda Console | `http://100.84.126.19:8096` | (none required) |
| Temporal UI | `http://100.84.126.19:8088` | (none required) |
| Trino | `http://100.84.126.19:8094` | (none required) |
| MinIO Console | `http://100.84.126.19:9011` | User: `minioadmin` / Pass: `minioadmin` |

### Local Backend API
```
http://localhost:8080
```

---

## Architecture Details

### Network Flow

```
1. Browser Request
   http://localhost:5173/dashboard
        ↓
2. Frontend (Native Node) - localhost:5173
   → Calls API at http://localhost:8080/api/...
        ↓
3. Backend (Docker) - localhost:8080
   → Connects to Remote PostgreSQL at 100.84.126.19:5432
   → Calls Hasura at 100.84.126.19:8085
   → Publishes to Redpanda at 100.84.126.19:19092
   → Updates via Debezium at 100.84.126.19:8083
        ↓
4. Response flows back → Frontend → Browser
```

### Docker Network Configuration

The backend runs in a Docker network that has access to the host network:

```yaml
networks:
  mac-backend:
    name: mac-backend
    driver: bridge
```

Key points:
- Backend container can reach `100.84.126.19` directly via TCP/IP
- Backend container port `8080` is exposed to `localhost:8080`
- Frontend (native) can reach backend at `http://localhost:8080`

### Database Configuration

Backend uses these connection settings:
```
Host: 100.84.126.19 (remote IP)
Port: 5432
User: postgres
Password: postgres
Database: alpha
SSL: disabled
```

---

## Troubleshooting

### Backend Container Won't Start

```bash
# Check logs
docker compose -f docker-compose.mac-distributed.yml logs backend

# Common issues:
# 1. Remote services not accessible
#    → Verify telnet connectivity: telnet 100.84.126.19 5432
#
# 2. Docker build fails
#    → Run: docker compose -f docker-compose.mac-distributed.yml build --no-cache
#
# 3. Port 8080 already in use
#    → Find process: lsof -i :8080
#    → Kill process: kill -9 <PID>
```

### Cannot Connect to Database

```bash
# Test PostgreSQL connection
psql postgresql://postgres:postgres@100.84.126.19:5432/alpha

# If connection fails:
# 1. Verify PostgreSQL is running on remote
#    ssh user@100.84.126.19
#    sudo systemctl status postgresql
#
# 2. Check firewall allows port 5432
#    telnet 100.84.126.19 5432
#
# 3. Verify credentials in .env
```

### Frontend Cannot Reach Backend

```bash
# Test from browser console
fetch('http://localhost:8080/health')
  .then(r => r.json())
  .then(d => console.log(d))

# If fails:
# 1. Verify backend is running: docker compose -f docker-compose.mac-distributed.yml ps
# 2. Test locally: curl http://localhost:8080/health
# 3. Check CORS configuration in backend
# 4. Check .env ALLOWED_ORIGINS includes http://localhost:5173
```

### Redpanda/Kafka Connection Issues

```bash
# Verify Kafka is accessible
nc -zv 100.84.126.19 19092

# Use Redpanda Console to verify brokers
# http://100.84.126.19:8096

# If issues persist:
# - Check kafka broker advertise address uses external IP (19092)
# - Docker on Mac may have network issues - restart Docker or use VPN
```

### Performance Issues

If the platform is slow:

1. **Docker Resource Allocation**: Increase Docker Desktop memory/CPU
   - Docker Desktop → Preferences → Resources → Increase CPU/Memory

2. **Network Latency**: Use VPN if on different network
   - Lower latency improves responsiveness

3. **Database Connection Pooling**: Check if backend is opening too many connections
   - View logs: `docker compose -f docker-compose.mac-distributed.yml logs backend | grep "connection"`

---

## Useful Commands

### Docker Management

```bash
# View all containers
docker compose -f docker-compose.mac-distributed.yml ps

# View logs (real-time)
docker compose -f docker-compose.mac-distributed.yml logs -f backend

# View logs (specific time range)
docker compose -f docker-compose.mac-distributed.yml logs --since 2m backend

# Restart backend
docker compose -f docker-compose.mac-distributed.yml restart backend

# Stop all
docker compose -f docker-compose.mac-distributed.yml down

# Remove volumes (careful!)
docker compose -f docker-compose.mac-distributed.yml down -v

# Rebuild image
docker compose -f docker-compose.mac-distributed.yml build --no-cache
```

### Network Testing

```bash
# Test connectivity to remote services
nc -zv 100.84.126.19 5432   # PostgreSQL
nc -zv 100.84.126.19 8085   # Hasura
nc -zv 100.84.126.19 19092  # Redpanda
nc -zv 100.84.126.19 7233   # Temporal

# DNS lookup
nslookup 100.84.126.19

# Ping
ping -c 3 100.84.126.19

# Trace route
traceroute 100.84.126.19
```

### Frontend Development

```bash
# Install dependencies
npm install

# Run dev server
npm run dev

# Build for production
npm run build

# Preview production build
npm run preview

# Check for errors
npm run lint
```

### API Testing

```bash
# Test backend health
curl http://localhost:8080/health

# Test with JWT
TOKEN="your-jwt-token"
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/dashboard

# Get Hasura schema
curl http://100.84.126.19:8085/v1/graphql -X POST \
  -H "Content-Type: application/json" \
  -H "X-Hasura-Admin-Secret: myadminsecret" \
  -d '{"query":"{ __schema { types { name } } }"}'
```

---

## Stopping the Platform

### Stop Everything (Clean)

```bash
# From the root directory
cd /path/to/semlayer

# Stop backend container
docker compose -f docker-compose.mac-distributed.yml down

# Stop frontend (press Ctrl+C in terminal where npm run dev is running)

# Verify nothing is running
docker compose -f docker-compose.mac-distributed.yml ps
netstat -tuln | grep 8080  # Should be empty
```

### Remote Services (on 100.84.126.19)

```bash
# SSH into remote
ssh user@100.84.126.19

# If needed, stop remote services
docker compose -f docker-compose.remote.yml down

# But usually leave them running
```

---

## Advanced Configuration

### Changing Database Connection

Edit `.env`:
```bash
DATABASE_URL=postgresql://postgres:password@100.84.126.19:5432/different_db?sslmode=disable
```

### Adding More CPU/Memory to Docker

```bash
# Edit docker-compose.mac-distributed.yml
services:
  backend:
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 2G
        reservations:
          cpus: '1'
          memory: 1G
```

### Custom Backend Port

```bash
# In docker-compose.mac-distributed.yml
services:
  backend:
    ports:
      - "3000:8080"  # Access via http://localhost:3000

# Update .env
VITE_API_BASE_URL=http://localhost:3000
```

### Adding SSL/TLS

```bash
# Create self-signed cert (temporary)
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes

# Configure in backend (add to environment)
TLS_CERT_PATH=/path/to/cert.pem
TLS_KEY_PATH=/path/to/key.pem
```

---

## Production Considerations

For production deployment:

1. **Use managed database**: AWS RDS, GCP Cloud SQL
2. **Update remote services**: Use production-grade Kafka, Redis
3. **Enable TLS**: Use proper certificates
4. **Set strong passwords**: Update all default credentials
5. **Use secrets management**: AWS Secrets Manager, HashiCorp Vault
6. **Monitor**: Set up Prometheus, Grafana
7. **Backup**: Regular PostgreSQL backups
8. **Scale**: Use Kubernetes instead of Docker Compose

---

## Getting Help

### Check Logs

```bash
# Backend logs
docker compose -f docker-compose.mac-distributed.yml logs -f backend

# All services (remote)
ssh user@100.84.126.19
docker compose -f docker-compose.remote.yml logs -f
```

### Common Issues Documentation

- [Backend Connection Issues](./BACKEND_CONNECTION_GUIDE.md)
- [Docker Networking](./DOCKER_NETWORKING.md)
- [PostgreSQL Troubleshooting](./DATABASE_TROUBLESHOOTING.md)
- [Hasura Setup](./HASURA_SETUP.md)

---

## Summary

✅ **You now have:**
- Scalable architecture with services split across machines
- Fast local development with backend in Docker on Mac
- Portable frontend development environment
- Easy to manage and debug
- Production-ready patterns

💡 **Key Points:**
- Remote IP: 100.84.126.19 (all data services)
- Local IP: localhost (backend & frontend on Mac)
- Network connectivity required between locations
- Services can be managed independently

🚀 **Next Steps:**
1. Deploy frontend to production CDN
2. Set up monitoring and alerting
3. Implement automated backups
4. Add security hardening (TLS, WAF)
