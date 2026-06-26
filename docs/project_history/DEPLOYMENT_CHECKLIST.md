# Deployment Checklist ✅

## Phase 1: Foundation (Database & Schema)

- [ ] **Postgres running**
  ```bash
  psql postgres://postgres:postgres@host.docker.internal:5432/alpha -c "SELECT 1"
  ```
  Expected: `1` (success)

- [ ] **Run DDL script**
  ```bash
  psql postgres://postgres:postgres@host.docker.internal:5432/alpha < backend/sql/calc-engine.sql
  ```
  Expected: No errors

- [ ] **Verify tables created**
  ```bash
  psql postgres://postgres:postgres@host.docker.internal:5432/alpha -c "\dt metric_*"
  ```
  Expected: 4 tables (metric_registry, metric_values_txn, metric_job_runs, anomaly_events)

- [ ] **Verify functions created**
  ```bash
  psql postgres://postgres:postgres@host.docker.internal:5432/alpha -c "\df+ *metric*"
  ```
  Expected: 5+ functions listed

---

## Phase 2: Backend Service

- [ ] **Go build successful**
  ```bash
  cd backend && go build ./cmd/server
  ```
  Expected: No errors

- [ ] **Backend starts**
  ```bash
  cd backend && timeout 5 go run ./cmd/server/main.go || true
  ```
  Expected: Server starts (will timeout after 5s, that's OK)

- [ ] **Routes registered**
  ```bash
  # In another terminal, with backend running:
  curl -s http://localhost:8080/_routes | grep -c metrics
  ```
  Expected: 9 (nine routes)

- [ ] **Health check passes**
  ```bash
  curl -s http://localhost:8080/health | jq .status
  ```
  Expected: `"healthy"` or similar

---

## Phase 3: REST API (CRUD)

With backend running (`go run ./cmd/server/main.go` in one terminal):

- [ ] **Create metric**
  ```bash
  RESPONSE=$(curl -s -X POST -H "X-Tenant-ID: test-tenant" \
    -H "Content-Type: application/json" \
    http://localhost:8080/api/metrics \
    -d '{"name":"Revenue","domain":"finance","aggregation_function":"sum"}')
  echo "$RESPONSE" | jq .id
  ```
  Expected: UUID returned (save this as $METRIC_ID)

- [ ] **List metrics**
  ```bash
  curl -s -H "X-Tenant-ID: test-tenant" \
    http://localhost:8080/api/metrics | jq '.[] | {id, name}'
  ```
  Expected: Your metric listed

- [ ] **Get single metric**
  ```bash
  curl -s -H "X-Tenant-ID: test-tenant" \
    http://localhost:8080/api/metrics/$METRIC_ID | jq .name
  ```
  Expected: `"Revenue"`

- [ ] **Update metric**
  ```bash
  curl -s -X PUT -H "X-Tenant-ID: test-tenant" \
    -H "Content-Type: application/json" \
    http://localhost:8080/api/metrics/$METRIC_ID \
    -d '{"name":"Revenue-Updated"}' | jq .name
  ```
  Expected: `"Revenue-Updated"`

---

## Phase 4: Computation Triggers

With backend running and $METRIC_ID from Phase 3:

- [ ] **Trigger PoP compute**
  ```bash
  RUN_RESPONSE=$(curl -s -X POST -H "X-Tenant-ID: test-tenant" \
    -H "Content-Type: application/json" \
    http://localhost:8080/api/metrics/$METRIC_ID/compute/pop \
    -d '{"period_label":"2024-11"}')
  echo "$RUN_RESPONSE" | jq .run_id
  ```
  Expected: Run ID returned (save as $RUN_ID)

- [ ] **Check job run status**
  ```bash
  curl -s -H "X-Tenant-ID: test-tenant" \
    http://localhost:8080/api/metrics/$METRIC_ID/runs | jq '.[] | {run_id, status}'
  ```
  Expected: Status should be "pending" or "success"

- [ ] **Trigger anomaly compute**
  ```bash
  ANOM_RESPONSE=$(curl -s -X POST -H "X-Tenant-ID: test-tenant" \
    -H "Content-Type: application/json" \
    http://localhost:8080/api/metrics/$METRIC_ID/compute/anomaly \
    -d '{"period_label":"2024-11"}')
  echo "$ANOM_RESPONSE" | jq .run_id
  ```
  Expected: Different run ID returned

- [ ] **Check anomalies**
  ```bash
  curl -s -H "X-Tenant-ID: test-tenant" \
    http://localhost:8080/api/metrics/$METRIC_ID/anomalies | jq .
  ```
  Expected: Empty array (no anomalies yet without data) or populated array

---

## Phase 5: Data Persistence

- [ ] **Verify metrics in Postgres**
  ```bash
  psql postgres://postgres:postgres@host.docker.internal:5432/alpha \
    -c "SELECT id, name, domain FROM metric_registry WHERE tenant_id = 'test-tenant'"
  ```
  Expected: Your metric visible

- [ ] **Verify job runs in Postgres**
  ```bash
  psql postgres://postgres:postgres@host.docker.internal:5432/alpha \
    -c "SELECT run_id, metric_id, status FROM metric_job_runs WHERE tenant_id = 'test-tenant' LIMIT 5"
  ```
  Expected: Job runs visible with status

- [ ] **Delete metric**
  ```bash
  curl -s -X DELETE -H "X-Tenant-ID: test-tenant" \
    http://localhost:8080/api/metrics/$METRIC_ID | jq .
  ```
  Expected: 204 (No Content) response

- [ ] **Verify deletion in Postgres**
  ```bash
  psql postgres://postgres:postgres@host.docker.internal:5432/alpha \
    -c "SELECT COUNT(*) FROM metric_registry WHERE id = '$METRIC_ID'"
  ```
  Expected: 0

---

## Phase 6: Automated Testing

- [ ] **Run E2E test script**
  ```bash
  bash test-calc-engine.sh
  ```
  Expected: All 11 tests PASS (green checkmarks)

---

## Phase 7: Optional Integration Points

These are ready to wire but not required for basic testing:

- [ ] **Temporal integration** (optional)
  - [ ] Temporal server running on localhost:7233
  - [ ] Wire client in backend/cmd/server/main.go
  - [ ] Create worker service (backend/cmd/worker/main.go)
  - [ ] See: CALC_ENGINE_INTEGRATION_GUIDE.md

- [ ] **Trino connection** (optional)
  - [ ] Test JDBC to 192.168.86.55:8090
  - [ ] Activate actual SQL execution in activities.go
  - [ ] Verify Iceberg tables created
  - [ ] See: CALC_ENGINE_INTEGRATION_GUIDE.md

- [ ] **Redpanda (Kafka)** (optional)
  - [ ] Redpanda running (Pandaproxy: http://localhost:8082)
  - [ ] Wire publisher in PublishCompletionEvent()
  - [ ] Test event delivery
  - [ ] See: CALC_ENGINE_INTEGRATION_GUIDE.md

- [ ] **Cube.dev** (optional)
  - [ ] Cube.dev running
  - [ ] Wire API endpoint in RefreshCubePartitions()
  - [ ] Test pre-aggregation refresh
  - [ ] See: CALC_ENGINE_INTEGRATION_GUIDE.md

---

## Success Criteria

✅ All Phase 1-6 checklist items completed  
✅ E2E test script passes all tests  
✅ Data persists in Postgres  
✅ Metrics can be created, read, updated, deleted  
✅ Compute triggers create job runs  
✅ No errors in backend logs  

---

## Quick Run-Through (5 minutes)

Save this as `quick-test.sh` and run it:

```bash
#!/bin/bash
set -e

echo "🔍 Checking database..."
psql postgres://postgres:postgres@host.docker.internal:5432/alpha -c "SELECT 1" > /dev/null
echo "✅ Database OK"

echo "🔨 Building backend..."
cd backend && go build ./cmd/server > /dev/null
echo "✅ Build OK"

echo "🚀 Running E2E tests..."
cd ..
bash test-calc-engine.sh

echo ""
echo "✅ All checks passed! Your calc engine is ready to use."
echo ""
echo "Next steps:"
echo "  1. See CALC_ENGINE_QUICKSTART.md for detailed testing"
echo "  2. See CALC_ENGINE_INTEGRATION_GUIDE.md to wire Temporal/Trino"
echo "  3. Read README_CALC_ENGINE.md for architecture overview"
```

---

## Troubleshooting

### Database Connection Failed
```bash
# Check Postgres is running
psql postgres://postgres:postgres@host.docker.internal:5432/alpha -c "SELECT 1"

# If failed, ensure:
# - Postgres is running on port 5432
# - Database 'alpha' exists
# - User 'postgres' has password 'postgres'
```

### Routes Not Showing
```bash
# Ensure backend is running in another terminal
# Check with:
curl http://localhost:8080/_routes | jq . | head -20

# If empty, backend may not be running
```

### Metric Creation Failed
```bash
# Check X-Tenant-ID header is set
# Check backend logs for validation errors
# Verify JSON is valid:
echo '{"name":"test","domain":"test","aggregation_function":"sum"}' | jq .
```

### Tests Failing
```bash
# Run individual test with:
bash test-calc-engine.sh 2>&1 | head -50

# Check backend is running:
curl http://localhost:8080/health

# Check database:
psql postgres://postgres:postgres@host.docker.internal:5432/alpha -c "\dt metric_*"
```

---

## Files You'll Need

| File | Purpose |
|------|---------|
| `backend/sql/calc-engine.sql` | Database schema |
| `backend/cmd/server/main.go` | Start backend |
| `test-calc-engine.sh` | Automated tests |
| `CALC_ENGINE_QUICKSTART.md` | Detailed testing guide |
| `CALC_ENGINE_INTEGRATION_GUIDE.md` | Wiring instructions |

---

**Status**: Ready to deploy ✅  
**Time estimate**: 5-15 minutes for full verification  
**Support**: See individual docs for detailed help  

Start with: `bash test-calc-engine.sh` 🚀
