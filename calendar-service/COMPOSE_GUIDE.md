# 🚀 Calendar Service - Which Compose File to Use?

## TL;DR - Quick Decision

### ❓ Are you developing locally?
**YES** → Use: `docker-compose -f docker-compose.local.yml up`
- All Golang services in Docker (Redpanda, Redis, Temporal, Calendar Service)
- Your PostgreSQL and Hasura run NATIVE (not in compose)
- Frontend runs NATIVE on your machine
- Hot reload enabled via volume mounts

---

### ❓ Are you deploying to staging/production?
**YES** → Use: `docker-compose -f docker-compose.remote.yml up`
- All services except PostgreSQL in Docker
- PostgreSQL is managed (AWS RDS, Cloud SQL, etc.)
- Hasura is managed (Hasura Cloud or self-hosted)
- Production-grade configurations (2GB Redis, SMP 2, etc.)

---

## Service Composition Matrix

| Service | Local | Remote |
|---------|-------|--------|
| **Calendar Service** | 🐳 Compose | 🐳 Compose |
| **Redpanda** | 🐳 Compose | 🐳 Compose |
| **Debezium** | 🐳 Compose | 🐳 Compose |
| **Redis** | 🐳 Compose | 🐳 Compose |
| **Temporal** | 🐳 Compose | 🐳 Compose |
| **PostgreSQL** | 💻 Native | 💻 Managed (AWS/etc) |
| **Hasura** | 💻 Native | 💻 Managed |
| **Frontend** | 💻 Native | 💻 External |

Legend: 🐳 = In docker-compose | 💻 = Outside compose

---

## Quick Start

### LOCAL Development

```bash
# 1. Set up environment
cp .env.example .env.local

# 2. Edit with your local PostgreSQL details
# POSTGRES_HOST=localhost
# HASURA_ENDPOINT=http://localhost:8080/v1/graphql

# 3. Start services
docker-compose -f docker-compose.local.yml up -d

# 4. Check health
curl http://localhost:8081/health

# 5. Watch logs
docker-compose -f docker-compose.local.yml logs -f calendar-service
```

### REMOTE Deployment

```bash
# 1. Set environment variables
export POSTGRES_HOST=your-rds-endpoint.aws.amazonaws.com
export POSTGRES_PASSWORD=your-password
export POSTGRES_DB=calendar_service
export HASURA_ENDPOINT=https://hasura.example.com/v1/graphql
export HASURA_ADMIN_SECRET=your-secret

# 2. Start services
docker-compose -f docker-compose.remote.yml up -d

# 3. Verify
curl http://localhost:8081/health

# 4. Monitor
docker-compose -f docker-compose.remote.yml logs -f
```

---

## Key Differences

### LOCAL (docker-compose.local.yml)
```yaml
Services in Compose:
✅ redpanda (9092, 19092)
✅ debezium (8083)
✅ redis (6379, 256MB)
✅ temporal (7233)
✅ calendar-service (8081, hot-reload enabled)

PostgreSQL: EXTERNAL (localhost:5432)
Hasura: EXTERNAL (http://localhost:8080)
Network: semlayer-local
Volume Mounts: .:/app (hot reload)
```

### REMOTE (docker-compose.remote.yml)
```yaml
Services in Compose:
✅ redpanda (9092, 19092, SMP 2)
✅ debezium (8083)
✅ redis (6379, 2GB LRU policy)
✅ temporal (7233)
✅ calendar-service (8081)

PostgreSQL: MANAGED (AWS RDS, Cloud SQL, etc.)
Hasura: MANAGED (Hasura Cloud or self-hosted)
Network: semlayer-remote
Restart: unless-stopped
Memory Policy: allkeys-lru
```

---

## ❌ DON'T Use

❌ `docker-compose.yml` - Generic/deprecated, use local or remote instead
❌ `docker-compose.test.yml` - Integration tests only
❌ `docker-compose.hybrid.yml` - Old version, DO NOT USE
❌ `docker-compose.backend.yml` - Old version, DO NOT USE

---

## Environment Variables

### Always Required
```bash
POSTGRES_HOST=<your-postgres>
POSTGRES_PORT=5432
POSTGRES_USER=postgres
POSTGRES_PASSWORD=<your-password>
POSTGRES_DB=calendar_service

HASURA_ENDPOINT=<your-hasura>/v1/graphql
HASURA_ADMIN_SECRET=<your-secret>
```

### Automatically Set (Usually)
```bash
REDPANDA_BROKERS=redpanda:9092
REDIS_URL=redis://redis:6379/0
TEMPORAL_HOST=temporal
TEMPORAL_PORT=7233
```

---

## Port Mapping

### LOCAL Services (docker-compose.local.yml)
```
8081:8080    Calendar Service
6379:6379    Redis
9092:9092    Redpanda (internal)
19092:19092  Redpanda (external)
8083:8083    Debezium
8081:8081    Schema Registry
7233:7233    Temporal
```

### Your Machine (Services NOT in Compose)
```
5432:5432    PostgreSQL (your machine)
8080:8080    Hasura (your machine)
3000:3000    Frontend (your machine, or wherever)
```

---

## Health Checks

```bash
# Calendar Service
curl http://localhost:8081/health

# Redis
redis-cli ping

# Redpanda
rpk cluster info --brokers localhost:19092

# Debezium
curl http://localhost:8083/

# Temporal
docker exec semlayer-temporal-local tctl workflow list
```

---

## Common Operations

### View Logs
```bash
# LOCAL
docker-compose -f docker-compose.local.yml logs -f calendar-service

# REMOTE
docker-compose -f docker-compose.remote.yml logs -f calendar-service
```

### Restart a Service
```bash
# LOCAL
docker-compose -f docker-compose.local.yml restart redis

# REMOTE
docker-compose -f docker-compose.remote.yml restart redis
```

### Stop All Services
```bash
# LOCAL
docker-compose -f docker-compose.local.yml down

# REMOTE
docker-compose -f docker-compose.remote.yml down
```

### Validate Compose File
```bash
# LOCAL
docker-compose -f docker-compose.local.yml config

# REMOTE
docker-compose -f docker-compose.remote.yml config
```

---

## Production Checklist

Before going to production with REMOTE:

- [ ] PostgreSQL endpoint verified (can connect from compose host)
- [ ] Hasura endpoint verified (can reach from compose host)
- [ ] All required secrets in environment variables
- [ ] Redis memory settings appropriate (2GB is default)
- [ ] Redpanda SMP set correctly for CPU cores
- [ ] `docker-compose -f docker-compose.remote.yml config` passes
- [ ] All health checks returning 200/OK
- [ ] CDC processor logs show "CDC processor started"
- [ ] No error rates in metrics
- [ ] Monitoring/alerting configured (if applicable)

---

## Troubleshooting

### "Environment variables not loaded"
Make sure you set them BEFORE running compose:
```bash
export POSTGRES_HOST=localhost
export POSTGRES_PASSWORD=mypassword
docker-compose -f docker-compose.local.yml up
```

### "Cannot reach PostgreSQL"
Verify `POSTGRES_HOST` is reachable from compose:
```bash
# From container
docker exec semlayer-calendar-service-local \
  pg_isready -h $POSTGRES_HOST -p $POSTGRES_PORT
```

### "Hasura endpoint not responding"
Check `HASURA_ENDPOINT` is correct:
```bash
curl -X POST $HASURA_ENDPOINT/v1/graphql \
  -H "X-Hasura-Admin-Secret: $HASURA_ADMIN_SECRET"
```

### "CDC processor won't start"
Check Redpanda is healthy:
```bash
docker exec semlayer-redpanda-local \
  rpk cluster info
```

---

## 📚 Documentation

- **LOCAL Setup**: See [DEPLOYMENT_ARCHITECTURES.md](DEPLOYMENT_ARCHITECTURES.md#local-development-setup)
- **REMOTE Setup**: See [DEPLOYMENT_ARCHITECTURES.md](DEPLOYMENT_ARCHITECTURES.md#remote-deployment-setup)
- **Implementation Details**: See [CDC_IMPLEMENTATION_COMPLETE.md](CDC_IMPLEMENTATION_COMPLETE.md)
- **Cache Design**: See [PRODUCTION_READY_CACHE_IMPLEMENTATION.md](PRODUCTION_READY_CACHE_IMPLEMENTATION.md)

---

## Questions?

1. **Is this for local development?** → Use `docker-compose.local.yml`
2. **Is this for production?** → Use `docker-compose.remote.yml`
3. **Unsure?** → Run both! They work independently on different networks.

🎉 **Let's go!**
