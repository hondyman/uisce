# Uisce Semantic OS — Production Installation Guide (RC1)

**RC1 Scope:** Path 1 (Docker Compose Staging Mesh) + Path 4 (Helm / Terraform Infrastructure). Paths 2 (Frontend HUDs) and 3 (Temporal Workflows) are RC2.

---

## Prerequisites

| Dependency | Minimum Version | Notes |
|------------|----------------|-------|
| Docker Engine | 24.0+ | With Docker Compose v2 plugin |
| Go | 1.21+ | For local binary builds |
| Python | 3.10+ | For stress test script |
| aiohttp | latest | `pip install aiohttp` |
| psql client | 15+ | `brew install postgresql` |
| Helm | 3.12+ | For Kubernetes deployment |
| Terraform | 1.5+ | For infrastructure provisioning |
| Infisical CLI | latest | `brew install infisical` (optional, for production secrets) |

---

## Quick Start — Local Staging Mesh

### 1. Clone & Navigate

```bash
git clone https://github.com/hondyman/uisce
cd uisce
```

### 2. Configure Environment

Create a `.env` file in the project root (or source `.env.infisical` for production credentials):

```bash
# Required
export POSTGRES_PASSWORD=uisce_admin_password_localdev
export JWT_SECRET=dev-jwt-secret-key-change-in-production

# Optional (Infisical for production)
export INFISICAL_TOKEN=st.YOUR_TOKEN_HERE
export INFISICAL_PROJECT_ID=YOUR_PROJECT_ID
```

### 3. Launch the Mesh

```bash
# Build & start all containers in detached mode
docker compose -f docker-compose.production.yml up --build -d

# Watch logs
docker compose -f docker-compose.production.yml logs -f uisce-core

# Verify health
curl http://localhost:8081/health
```

Expected output: `{"status":"ok"}` or similar.

### 4. Verify Services

```bash
# PostgreSQL
psql "postgres://uisce_admin:uisce_admin_password_localdev@localhost:5432/uisce_control_plane" -c "SELECT 1"

# Redis
redis-cli ping
# Expected: PONG

# Redpanda
rpk cluster health
# Expected: Cluster healthy
```

### 5. Run Migration Bootstrap

```bash
# Apply all migrations (runs idempotently — skips already-applied migrations)
export POSTGRES_DSN="postgres://uisce_admin:uisce_admin_password_localdev@localhost:5432/uisce_control_plane?sslmode=disable"
bash scripts/bootstrap-production.sh
```

Expected output:
```
[OK] Database is accepting connections.
[OK] Build complete. SHA256: abc123...
[INFO] Launching Uisce Core...
```

### 6. Run 10,000-Order Stress Test

```bash
# Install Python dependencies
pip install aiohttp

# Run the stress test
python3 scripts/stress-test-hydrate.py \
  --url http://localhost:8081/api/v1/compliance/external/evaluate-external \
  --requests 10000 \
  --concurrency 50

# Check output
# Phase A: 8,000 unique requests (establishing baseline)
# Phase B: 2,000 requests across 50 repeated idempotency keys
```

Expected (after Redis idempotency caching is wired):
```
[RESULT] PASSED
Cache hit rate: 92.3%
p50 Latency: 45.20 µs
p99 Latency: 890.15 µs
```

Current status (Redis caching not yet wired in `external_compliance_handler.go`):
```
[WARN] Cache hit rate below 80% — Redis idempotency caching not wired?
[RESULT] FAILED (cache assertion)
```
This is **expected** in RC1 — see `docs/production/POST-RC1.md`.

### 7. Tear Down

```bash
docker compose -f docker-compose.production.yml down -v
# -v removes all named volumes (postgres_data, redis_data, etc.)
```

---

## Docker Compose vs. Dev Profile

| Profile | File | When to Use |
|---------|------|-------------|
| **Staging / CI gate** | `docker-compose.production.yml` | End-to-end tests, benchmarks |
| **Daily iteration** | `docker-compose.backend.yml` | Developer loop (lighter, faster startup) |

```bash
# Dev profile (lightweight)
docker compose -f docker-compose.backend.yml up --build

# Staging profile (full mesh)
docker compose -f docker-compose.production.yml up --build
```

---

## Configuration Reference

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `POSTGRES_DSN` | `postgres://...@localhost:5432/semlayer` | Backend reads this (not `DATABASE_URL`) |
| `POSTGRES_PASSWORD` | `uisce_admin_password_localdev` | Postgres password (compose) |
| `REDIS_ADDR` | `redis:6379` | Redis cache address |
| `KAFKA_BROKERS` | `redpanda:29092` | Kafka/Redpanda broker list |
| `FLIGHT_PORT` | `8090` | Arrow Flight SQL port (aspirational stub) |
| `JWT_SECRET` | `dev-jwt-secret...` | JWT signing key |
| `EBPF_ENABLED` | `false` | eBPF requires host network + CAP_NET_ADMIN |
| `INFISICAL_TOKEN` | — | Infisical service token (production) |

### Ports

| Host Port | Container Port | Service |
|----------|----------------|---------|
| 5432 | 5432 | PostgreSQL |
| 6379 | 6379 | Redis |
| 9092 | 9092 | Redpanda (Kafka) |
| 9030 | 9030 | StarRocks FE (query) |
| 8081 | 8080 | Uisce REST API |
| 8090 | 8090 | Arrow Flight SQL (aspirational) |
| 8980 | 8980 | QuickFIX Acceptor (aspirational) |

---

## Troubleshooting

### Backend won't start

```bash
# Check postgres is up
docker compose -f docker-compose.production.yml logs postgres

# Check migrations ran
docker compose -f docker-compose.production.yml exec postgres \
  psql -U uisce_admin -d uisce_control_plane -c \
  "SELECT filename FROM schema_migrations ORDER BY applied_at DESC LIMIT 5;"
```

### Migration fails

```bash
# Run migrations manually
docker compose -f docker-compose.production.yml exec postgres \
  psql -U uisce_admin -d uisce_control_plane -f \
  /docker-entrypoint-initdb.d/YYYYMMDD_descriptive_name.up.sql
```

### Stress test fails with connection refused

```bash
# Ensure backend is listening
curl http://localhost:8081/health

# Check port mapping
docker compose -f docker-compose.production.yml port uisce-core 8080
# Expected: 0.0.0.0:8081
```

---

## Next Steps After RC1

See `docs/production/POST-RC1.md` for the aspirational roadmap:
- Wire Redis idempotency caching into `external_compliance_handler.go`
- Implement Arrow Flight SQL gRPC query handler (port 8090)
- Implement QuickFIX session and tickerplant wiring (port 8980)
- StarRocks connector activation
- eBPF XDP integration
- Frontend governance dashboards (RC2)
- Temporal regulatory pack workflows (RC2)
