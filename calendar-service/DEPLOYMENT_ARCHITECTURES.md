# Calendar Service - Deployment Architectures

## Architecture Overview

This Calendar Service supports three deployment scenarios, each with a different composition of containerized vs. native services.

### 1. **LOCAL Development** (docker-compose.local.yml)
- **Where**: Developer machine or lab environment
- **Services in Compose**: 🐳 Redpanda, Debezium, Redis, Temporal, Calendar Service
- **Services Native**: 💻 PostgreSQL, Hasura, Frontend
- **Use Case**: Full local development with hot reload
- **Start**: `docker-compose -f docker-compose.local.yml up`

```
┌─────────────────────────────────────────┐
│         DOCKER COMPOSE (LOCAL)          │
├─────────────────────────────────────────┤
│ • redpanda (Kafka/CDC events)           │
│ • debezium (PostgreSQL CDC connector)   │
│ • redis (Caching)                       │
│ • temporal (Workflow orchestration)     │
│ • calendar-service (API - port 8081)    │
└─────────────────────────────────────────┘
           ▼          ▼          ▼
    ┌──────────┬──────────┬──────────┐
    │          │          │          │
    ▼          ▼          ▼          ▼
PostgreSQL  Hasura   Frontend    (Your Dev Machine)
(Remote)    (Remote) (Native)
```

### 2. **REMOTE Deployment** (docker-compose.remote.yml)
- **Where**: Staging/Production infrastructure (AWS, GCP, etc.)
- **Services in Compose**: 🐳 Redpanda, Debezium, Redis, Temporal, Calendar Service
- **Services Native**: 💻 PostgreSQL, Hasura (managed services)
- **Use Case**: Containerized deployment where PostgreSQL/Hasura are managed externally
- **Start**: `docker-compose -f docker-compose.remote.yml up`

```
┌─────────────────────────────────────────┐
│      DOCKER COMPOSE (REMOTE)            │
├─────────────────────────────────────────┤
│ • redpanda (Kafka/CDC events)           │
│ • debezium (PostgreSQL CDC connector)   │
│ • redis (Caching)                       │
│ • temporal (Workflow orchestration)     │
│ • calendar-service (API - port 8081)    │
└─────────────────────────────────────────┘
           ▼          ▼          ▼
    ┌──────────┬──────────────────┐
    │          │                  │
    ▼          ▼                  ▼
PostgreSQL  Hasura          (Managed Services)
(AWS RDS)   (Hasura Cloud)
```

### 3. **Standard** (docker-compose.yml)
- **Where**: Generic local testing/debugging
- **Services in Compose**: 🐳 Redpanda, Debezium, Redis, Temporal, Calendar Service
- **Services External**: 💻 PostgreSQL, Hasura (via environment variables)
- **Deprecated**: Use docker-compose.local.yml or docker-compose.remote.yml instead

---

## Quick Start Guide

### LOCAL Development Setup

1. **Copy environment file**:
   ```bash
   cp .env.example .env.local
   ```

2. **Configure for your local PostgreSQL**:
   ```bash
   # Edit .env.local
   POSTGRES_HOST=localhost          # Your local postgres host
   POSTGRES_PORT=5432
   POSTGRES_USER=postgres
   POSTGRES_PASSWORD=yourpassword
   POSTGRES_DB=calendar_service
   
   HASURA_ENDPOINT=http://localhost:8080/v1/graphql
   HASURA_ADMIN_SECRET=your-secret-key
   ```

3. **Start all services**:
   ```bash
   docker-compose -f docker-compose.local.yml up -d
   ```

4. **Verify services are running**:
   ```bash
   # Calendar Service
   curl http://localhost:8081/health
   
   # Redis
   redis-cli ping
   
   # Redpanda
   rpk cluster info --brokers localhost:19092
   ```

5. **Check logs** (if needed):
   ```bash
   docker-compose -f docker-compose.local.yml logs -f calendar-service
   docker-compose -f docker-compose.local.yml logs -f redis
   ```

6. **Shutdown**:
   ```bash
   docker-compose -f docker-compose.local.yml down
   ```

---

### REMOTE Deployment Setup

1. **Set environment variables** before deploying:
   ```bash
   export POSTGRES_HOST=your-rds-endpoint.aws.amazonaws.com
   export POSTGRES_PORT=5432
   export POSTGRES_USER=postgres
   export POSTGRES_PASSWORD=your-rds-password
   export POSTGRES_DB=calendar_service
   
   export HASURA_ENDPOINT=https://hasura.example.com/v1/graphql
   export HASURA_ADMIN_SECRET=your-hasura-secret
   
   export REDPANDA_BROKERS=redpanda:9092
   export REDIS_URL=redis://redis:6379/0
   ```

2. **Start all services**:
   ```bash
   docker-compose -f docker-compose.remote.yml up -d
   ```

3. **Verify deployment**:
   ```bash
   # Check running containers
   docker-compose -f docker-compose.remote.yml ps
   
   # Check calendar-service health
   curl http://localhost:8081/health
   ```

4. **Monitor logs**:
   ```bash
   docker-compose -f docker-compose.remote.yml logs -f calendar-service
   ```

---

## Environment Variables Reference

| Variable | LOCAL | REMOTE | Example |
|----------|-------|--------|---------|
| `POSTGRES_HOST` | Remote | Remote (RDS) | `localhost` / `db.aws.amazonaws.com` |
| `POSTGRES_PORT` | 5432 | 5432 | `5432` |
| `POSTGRES_USER` | `postgres` | Your user | `postgres` |
| `POSTGRES_PASSWORD` | Your password | Your password | `secure-password` |
| `POSTGRES_DB` | `calendar_service` | `calendar_service` | `calendar_service` |
| `REDPANDA_BROKERS` | `redpanda:9092` | `redpanda:9092` | Internal broker |
| `REDIS_URL` | `redis://redis:6379/0` | `redis://redis:6379/0` | Internal redis |
| `HASURA_ENDPOINT` | Remote | Remote | `http://localhost:8080/v1/graphql` |
| `HASURA_ADMIN_SECRET` | Your secret | Your secret | `admin-secret-key` |
| `TEMPORAL_HOST` | `temporal` | `temporal` | Container service name |
| `TEMPORAL_PORT` | `7233` | `7233` | gRPC port |
| `SERVER_PORT` | `8080` | `8080` | Internal service port |
| `ENVIRONMENT` | `local` | `production` | Development mode |
| `LOG_LEVEL` | `debug` | `info` | Log verbosity |

---

## Service Ports (LOCAL)

| Service | Internal | External |
|---------|----------|----------|
| Calendar Service | 8080 | 8081 |
| Redis | 6379 | 6379 |
| Redpanda (Internal) | 9092 | 9092 |
| Redpanda (Schema Registry) | 8081 | 18081 |
| Debezium | 8083 | 8083 |
| Temporal | 7233 | 7233 |

---

## Key Features - CDC Integration

### What's Implemented ✅

1. **CDC Event Processing**
   - Listens to Redpanda topics via Franz-Go consumer
   - Processes Debezium CDC events from PostgreSQL
   - Supports 4 table handlers:
     - `profile_calendars` changes → L1 cache invalidation
     - `calendars` changes → L2 cache invalidation + Hasura profile lookup
     - `schedule_profiles` changes → region-aware cache invalidation
     - `blackouts` changes → profile cache invalidation

2. **L1+L2 Caching Architecture**
   - L1: In-memory sync.RWMutex (5 min TTL, <1ms lookup)
   - L2: Redis (1 hour TTL, 2-5ms lookup)
   - L3: Hasura GraphQL fallback (40-100ms query)

3. **Prometheus Metrics**
   - `calendar_profile_resolution_total` - resolutions by source
   - `calendar_profile_resolution_duration_seconds` - latency by source
   - `calendar_profile_resolution_errors_total` - errors by type

4. **Graceful Shutdown**
   - CDC processor stops cleanly
   - Leaves Redpanda consumer group
   - All goroutines terminate properly

---

## Verification Checklist

- [ ] PostgreSQL accessible on `POSTGRES_HOST:POSTGRES_PORT`
- [ ] Hasura GraphQL endpoint responds to queries
- [ ] `.env.local` or `.env` file configured
- [ ] `docker-compose config` validates without errors
- [ ] `docker-compose up -d` succeeds
- [ ] `curl http://localhost:8081/health` returns 200
- [ ] CDC processor logs show "CDC processor started"
- [ ] Redpanda topics created (`rpk topic list --brokers localhost:19092`)
- [ ] Debezium connector running (`curl http://localhost:8083/`)
- [ ] Redis responding to pings (`redis-cli ping`)

---

## Troubleshooting

### Service won't start
```bash
# Check logs
docker-compose -f docker-compose.local.yml logs service-name

# Verify compose syntax
docker-compose -f docker-compose.local.yml config

# Check image availability
docker images | grep semlayer
```

### CDC processor not consuming events
```bash
# Check consumer group status
rpk group describe calendar-cdc-group --brokers redpanda:9092

# Check topics exist
rpk topic list --brokers redpanda:9092

# Verify Debezium has created CDC topics
rpk topic list --brokers redpanda:9092 | grep cdc
```

### Redis connection issues
```bash
# Test Redis connectivity
redis-cli -h localhost ping

# Check Redis memory
redis-cli info memory
```

### Hasura connection issues
```bash
# Check endpoint responds
curl -X POST HASURA_ENDPOINT \
  -H "X-Hasura-Admin-Secret: $HASURA_ADMIN_SECRET" \
  -d '{"query":"{ __typename }"}'
```

---

## Production Deployment Checklist

- [ ] PostgreSQL on managed service (AWS RDS, Cloud SQL, etc.)
- [ ] Hasura deployed and healthy
- [ ] Docker images built and pushed to registry
- [ ] Environment variables set via secrets manager
- [ ] `docker-compose down` before deploying new version
- [ ] `docker-compose -f docker-compose.remote.yml up -d`
- [ ] Verify all health checks passing
- [ ] Monitor CDC processor logs for errors
- [ ] Verify cache hit rates in Prometheus
- [ ] Set up Grafana dashboard for monitoring
- [ ] Configure alerting for error rates

---

## Support & Documentation

- **Cache Implementation**: See [PRODUCTION_READY_CACHE_IMPLEMENTATION.md](PRODUCTION_READY_CACHE_IMPLEMENTATION.md)
- **CDC Integration**: See [CDC_INTEGRATION_GUIDE.md](CDC_INTEGRATION_GUIDE.md)
- **Architecture**: See [ARCHITECTURE.md](ARCHITECTURE.md)
