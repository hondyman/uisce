# Redpanda Development Setup Guide

This guide covers running **SemLayer** locally with **Redpanda** (Kafka-compatible) instead of RabbitMQ. All RabbitMQ references have been removed from the codebase.

## Quick Start (Laptop Development)

### Prerequisites
- Docker & Docker Compose installed
- Go 1.21+ (for backend)
- Node.js 18+ (for frontend/UI, if applicable)

### 1. Start the Full Stack

```bash
cd /path/to/semlayer
docker-compose up -d
```

This brings up:
- **Redpanda** (Kafka-compatible broker) on port 9092 (internal) / 19092 (external)
- **Redpanda Console** UI on port 8096
- **Backend** on port 8082
- **API Gateway** on port 8001
- **Business Process Backend** on port 8086
- **Postgres** (via DATABASE_URL)
- **Redis**, **Temporal**, **Hasura**, and supporting services

### 2. Verify Services Are Healthy

```bash
# Check Redpanda health
docker exec $(docker ps -q -f "name=semlayer-redpanda$") rpk cluster info

# View Redpanda Console
open http://localhost:8096  # Kafka topics, schemas, consumer groups
```

### 3. Run Smoke Tests

#### Redpanda Core Test
```bash
bash scripts/redpanda_smoke_test.sh
# Output: "Smoke test PASSED" on success
```

#### Event Router End-to-End Test
```bash
REDPANDA_CONTAINER=semlayer-redpanda \
HASURA_URL=http://localhost:8080/v1/graphql \
EVENT_ROUTER_URL=http://localhost:8081/events \
bash scripts/event_router_smoke_test.sh
# Output: "Event Router smoke test PASSED" on success
```

## Configuration

### Environment Variables

Add these to a `.env` file or set them in your shell:

```bash
# Kafka/Redpanda
KAFKA_BROKERS=redpanda:9092          # Internal (docker network)
KAFKA_BROKERS_EXTERNAL=localhost:19092  # External (from host machine)

# Backend
DATABASE_URL=postgres://postgres:postgres@host.docker.internal:5432/alpha?sslmode=disable
HASURA_URL=http://hasura:8080
HASURA_ADMIN_SECRET=myadminsecret
REDIS_ADDR=redis:6379
TEMPORAL_HOSTPORT=temporal:7233

# Redpanda Console
REDPANDA_CONSOLE_PORT=8096

# Backend ports
BACKEND_PORT=8082
API_GATEWAY_PORT=8001
BP_BACKEND_PORT=8086
```

### Docker Compose Service Configuration

The main [docker-compose.yml](./docker-compose.yml) includes Redpanda with:

- **Kafka broker** listening on port 9092 (internal) / 19092 (external)
- **Admin API** on port 9644
- **Schema Registry** on port 8081
- **Pandaproxy** (HTTP bridge) on port 8082
- **Console UI** on port 8096

All backend services are configured with `KAFKA_BROKERS=redpanda:9092` by default.

## Local Development (Without Docker)

To run the backend locally while Redpanda runs in Docker:

### 1. Start Redpanda only
```bash
docker-compose up -d redpanda
```

### 2. Build and run backend
```bash
cd backend
export KAFKA_BROKERS=localhost:19092  # Use external Redpanda port
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable"
export REDIS_ADDR=localhost:6379
export TEMPORAL_HOSTPORT=localhost:7233
export HASURA_URL=http://localhost:8080

go run ./cmd/server
```

### 3. Test connectivity
```bash
# Check if backend can reach Redpanda
curl http://localhost:8080/health
```

## Kafka Topics & Consumer Groups

### View Topics
```bash
# Inside container
docker exec semlayer-redpanda rpk topic list

# Via Redpanda Console
open http://localhost:8096/topics
```

### Create a Topic Manually
```bash
docker exec semlayer-redpanda rpk topic create my-topic -p 1 -r 1
```

### Consume Messages
```bash
docker exec semlayer-redpanda rpk topic consume my-topic -o start -n 10
```

## Testing & CI

### Unit Tests
```bash
cd backend
go test ./...
```

### Smoke Tests
- **Redpanda smoke test**: [scripts/redpanda_smoke_test.sh](./scripts/redpanda_smoke_test.sh)
  - Creates a Redpanda topic
  - Produces and consumes a test message
  - Validates produce/consume latency

- **Event Router smoke test**: [scripts/event_router_smoke_test.sh](./scripts/event_router_smoke_test.sh)
  - Posts an event to the Event Router
  - Verifies routing to Redpanda topic
  - Validates end-to-end delivery

### CI Workflows
- **GitHub Actions**: [.github/workflows/kafka-smoke-test.yml](./.github/workflows/kafka-smoke-test.yml)
  - Runs on every push to main/develop
  - Executes redpanda_smoke_test.sh and validates build

## Troubleshooting

### "Connection refused" errors
**Cause**: Backend is trying to use internal `redpanda:9092` but Redpanda isn't running.
**Fix**:
```bash
docker-compose up -d redpanda
docker ps | grep redpanda  # Verify it's running
```

### Redpanda container exits immediately
**Check logs**:
```bash
docker logs semlayer-redpanda
```

**Common issues**:
- Port 9092 already in use → kill competing process or use different port
- Insufficient memory → increase Docker memory limit or reduce Redpanda's `--memory` setting

### Smoke test times out on consume
**Cause**: `rpk` needs proper time duration syntax (e.g., `5s` not `5000`).
**Status**: Fixed in [scripts/redpanda_smoke_test.sh](./scripts/redpanda_smoke_test.sh)

## Migration from RabbitMQ

All code has been migrated to Kafka/Redpanda:

- ✅ Environment variables: `KAFKA_BROKERS` (no `RABBITMQ_URL`)
- ✅ Event routing: topics instead of exchanges/queues
- ✅ Backend services: all use `kafka.Writer` and consumer groups
- ✅ Tests: updated to produce/consume from Redpanda topics
- ⚠️ Legacy stubs: Deprecated RabbitMQ stubs remain for compatibility but are not used

**Code changes**:
- `backend/internal/rulefabric/orchestration.go` — Kafka publishing
- `backend/internal/services/*` — Event routing via Kafka
- `backend/cmd/*` — ENV var parsing for `KAFKA_BROKERS`
- `docker-compose.yml` — Redpanda service definition

## Next Steps

1. **Run smoke tests** to validate local setup
2. **Review** [backend/README.md](./backend/README.md) for service architecture
3. **Check** [agents.md](./agents.md) for tenant-scoped Fabric Builder workflows
4. **Build frontend** (if needed) — see [frontend/](./frontend/) README

---

**Questions?** Check the conversation summary in `.github/copilot-instructions.md` or open an issue.
