# Phase 3 Staging Deployment Runbook

**Objective**: Deploy Phase 3 Temporal Queue Routing to staging, validate for 24-48 hours, then proceed to Phase 4 AI

**Timeline**: 
- T+0: Begin deployment
- T+4h: Full validation complete
- T+24-48h: Monitor for stability
- T+48h: Promote to production OR rollback

---

## Pre-Deployment Checklist (Before You Start)

```bash
# ✅ Code Quality
[ ] Phase 3 code compiles: go build ./cmd/server
[ ] No test failures: go test ./...
[ ] No lint errors: golangci-lint run

# ✅ Infrastructure Ready
[ ] Temporal cluster healthy: tctl cluster describe
[ ] Redis accessible: redis-cli PING → PONG
[ ] PostgreSQL accessible: psql -c "SELECT version()"
[ ] Hasura responding: curl http://hasura:8080/v1/graphql

# ✅ Staging Environment
[ ] docker-compose.yml updated with new env vars
[ ] WORKER_REGIONS set to: us-east-1,eu-west-1,ap-southeast-1
[ ] DEFAULT_REGION set to: us-east-1
[ ] Staging database backed up

# ✅ Monitoring Setup
[ ] Prometheus scrape config added for calendar-service:9090
[ ] Alert rules loaded (see section 3 below)
[ ] Grafana dashboard imported for Phase 3 metrics
[ ] Log aggregation (ELK/Loki) configured and receiving logs
```

---

## Phase 1: Build & Deploy (T+0 to T+1h)

### Step 1.1: Build Docker Image

```bash
cd /Users/eganpj/GitHub/semlayer/calendar-service

# Build with Phase 3 code
docker build \
  --tag calendar-service:phase3 \
  --tag calendar-service:phase3-$(date +%Y%m%d-%H%M%S) \
  .

# Verify image size is reasonable (~150-200MB)
docker images | grep calendar-service:phase3

# Push to registry (if using remote registry)
docker tag calendar-service:phase3 registry.example.com/calendar-service:phase3
docker push registry.example.com/calendar-service:phase3
```

**Expected output:**
```
Successfully built abc123def456
Successfully tagged calendar-service:phase3:latest
```

### Step 1.2: Update Environment Variables

**File**: `docker-compose.yml` (staging)

```yaml
services:
  calendar-service:
    image: calendar-service:phase3
    environment:
      # Phase 1: Cache (already working)
      CACHE_ENABLED: "true"
      REDIS_URL: "redis://redis:6379/0"
      REDIS_PREFIX: "calendar"
      REDIS_CACHE_TTL: "3600"
      
      # Phase 2: Data Residency (already working)
      DATA_RESIDENCY_POLICY: "strict"
      POSTGRES_HOST: "postgres"
      POSTGRES_PORT: "5432"
      POSTGRES_USER: "postgres"
      POSTGRES_PASSWORD: "postgres"
      POSTGRES_DB: "calendar_db"
      
      # Phase 3: Temporal Queue Routing (NEW)
      WORKER_REGIONS: "us-east-1,eu-west-1,ap-southeast-1"
      DEFAULT_REGION: "us-east-1"
      TEMPORAL_HOST_PORT: "temporal:7233"
      TEMPORAL_NAMESPACE: "default"
      
      # Hasura
      HASURA_ENDPOINT: "http://hasura:8080/v1/graphql"
      HASURA_ADMIN_SECRET: "myadminsecret"
      
      # Server
      SERVER_PORT: "8081"
      LOG_LEVEL: "info"
      ENVIRONMENT: "staging"
```

### Step 1.3: Deploy to Staging

```bash
# Go to staging environment
cd /staging/docker-compose

# Stop old version
docker-compose down calendar-service

# Update image reference
docker-compose pull calendar-service

# Start new version
docker-compose up -d calendar-service

# Wait for service to be ready (30-60 seconds)
sleep 30

# Verify container is running
docker-compose ps calendar-service
# Expected: healthy status
```

**Expected output:**
```
NAME                    STATUS              PORTS
calendar-service        Up 15 seconds (healthy)   0.0.0.0:8081->8081/tcp
```

### Step 1.4: Verify Service Startup

```bash
# Check logs for worker registration
docker logs calendar-service 2>&1 | tail -50 | grep -E "(Registered|All.*regional|error|failed)"

# Expected to see:
# ✓ Registered workers for region us-east-1: ...
# ✓ Registered workers for region eu-west-1: ...
# ✓ Registered workers for region ap-southeast-1: ...
# ✓ All 9 regional workers running (3 regions × 3 tiers)
```

---

## Phase 2: Validation (T+1h to T+4h)

### Step 2.1: Health Checks

```bash
# API Health
curl -s http://localhost:8081/health | jq .
# Expected: {"status": "healthy", "cache": "connected", "temporal": "connected"}

# Readiness
curl -s http://localhost:8081/ready | jq .
# Expected: {"status": "ready"}

# Ping
curl -s http://localhost:8081/ping
# Expected: "pong"
```

### Step 2.2: Temporal Queue Verification

```bash
# List all task queues
temporal task-queue list

# Expected output should include:
# - us-east-1-critical-queue
# - us-east-1-standard-queue
# - us-east-1-bulk-queue
# - eu-west-1-critical-queue
# - eu-west-1-standard-queue
# - eu-west-1-bulk-queue
# - ap-southeast-1-critical-queue
# - ap-southeast-1-standard-queue
# - ap-southeast-1-bulk-queue
# (9 queues total)

# Verify each queue has pollers active
temporal task-queue describe --task-queue us-east-1-critical-queue
# Expected: PollerCount: 3

temporal task-queue describe --task-queue eu-west-1-standard-queue
# Expected: PollerCount: 2

temporal task-queue describe --task-queue ap-southeast-1-bulk-queue
# Expected: PollerCount: 1
```

### Step 2.3: Test Job Routing (Critical)

**Test Case 1: High Priority (Critical Tier)**

```bash
# Submit a high-priority job
curl -X POST http://localhost:8081/api/v1/check-availability \
  -H "Content-Type: application/json" \
  -H "X-Hasura-Tenant-Id: test-tenant-001" \
  -d '{
    "profile_name": "default",
    "region": "us-east-1",
    "priority": 2,
    "start": "2026-02-20T10:00:00Z",
    "end": "2026-02-20T11:00:00Z"
  }'

# Expected: 200 OK with availability result

# Verify it appears on the critical queue
temporal workflow list --task-queue us-east-1-critical-queue | grep "started"
```

**Test Case 2: Standard Priority (Standard Tier)**

```bash
# Submit standard-priority job
curl -X POST http://localhost:8081/api/v1/check-availability \
  -H "Content-Type: application/json" \
  -H "X-Hasura-Tenant-Id: test-tenant-001" \
  -d '{
    "profile_name": "default",
    "region": "eu-west-1",
    "priority": 5,
    "start": "2026-02-20T14:00:00Z",
    "end": "2026-02-20T15:00:00Z"
  }'

# Verify on standard queue
temporal workflow list --task-queue eu-west-1-standard-queue | grep "started"
```

**Test Case 3: Low Priority (Bulk Tier)**

```bash
# Submit bulk-priority job
curl -X POST http://localhost:8081/api/v1/check-availability \
  -H "Content-Type: application/json" \
  -H "X-Hasura-Tenant-Id: test-tenant-001" \
  -d '{
    "profile_name": "default",
    "region": "ap-southeast-1",
    "priority": 9,
    "start": "2026-02-21T10:00:00Z",
    "end": "2026-02-21T11:00:00Z"
  }'

# Verify on bulk queue
temporal workflow list --task-queue ap-southeast-1-bulk-queue | grep "started"
```

### Step 2.4: Cache Validation

```bash
# Submit same request twice, measure latency

# First request (cache miss)
time curl -s http://localhost:8081/api/v1/check-availability \
  -H "X-Hasura-Tenant-Id: test-tenant-001" \
  -d '{"profile_name": "default", "region": "us-east-1", ...}' > /dev/null
# Expected: ~100-150ms

# Second request (cache hit - same exact params)
time curl -s http://localhost:8081/api/v1/check-availability \
  -H "X-Hasura-Tenant-Id: test-tenant-001" \
  -d '{"profile_name": "default", "region": "us-east-1", ...}' > /dev/null
# Expected: ~5-20ms (22x faster)

# Check cache contents
redis-cli KEYS "calendar:profile:*" | wc -l
# Expected: Should have accumulated keys
```

### Step 2.5: Region Authorization Validation

**Test Case 1: Valid Region Access**

```bash
# Should succeed
curl -X POST http://localhost:8081/api/v1/check-availability \
  -H "X-Hasura-Tenant-Id: test-tenant-001" \
  -d '{"region": "us-east-1", ...}'
# Expected: 200 OK
```

**Test Case 2: Invalid Region Access**

```bash
# Should be rejected (unauthorized)
curl -X POST http://localhost:8081/api/v1/check-availability \
  -H "X-Hasura-Tenant-Id: test-tenant-001" \
  -d '{"region": "invalid-region", ...}'
# Expected: 403 Forbidden with "Unauthorized region" message
```

### Step 2.6: Concurrent Load Test

```bash
# Load test: 100 concurrent requests across all tiers
echo "Running load test..."
for i in {1..100}; do
  priority=$((RANDOM % 10 + 1))  # Random 1-10
  region=$(echo "us-east-1 eu-west-1 ap-southeast-1" | tr ' ' '\n' | shuf | head -1)
  
  curl -s -X POST http://localhost:8081/api/v1/check-availability \
    -H "X-Hasura-Tenant-Id: test-tenant-$((i % 5))" \
    -d "{\"region\": \"$region\", \"priority\": $priority, ...}" &
done
wait

echo "Load test complete"

# Check for errors in logs
docker logs calendar-service 2>&1 | grep -c "error"
# Expected: 0 errors
```

---

## Phase 3: Monitoring (T+4h to T+48h)

### Step 3.1: Set Up Prometheus Scraping

**File**: `prometheus.yml` (staging)

```yaml
scrape_configs:
  - job_name: calendar-service
    static_configs:
      - targets: ['localhost:8081']
    scrape_interval: 15s
    scrape_timeout: 10s
```

Reload Prometheus:
```bash
curl -X POST http://localhost:9090/-/reload
```

### Step 3.2: Key Metrics to Monitor

```promql
# Total workflows running
count(temporal_workflow_active_total)
# Expected: 0-10 (depending on traffic)

# Workflows per queue
temporal_workflow_active_total{task_queue=~".*-critical-queue"}
temporal_workflow_active_total{task_queue=~".*-standard-queue"}
temporal_workflow_active_total{task_queue=~".*-bulk-queue"}

# Worker utilization (should not consistently > 80%)
temporal_worker_concurrent_workflows / temporal_worker_max_concurrent_workflows

# Queue pollers active (should match expected counts)
# us-east-1-critical: 3 pollers
# eu-west-1-standard: 2 pollers
# ap-southeast-1-bulk: 1 poller

# Job latency by tier (p95)
histogram_quantile(0.95, temporal_workflow_duration_ms{tier="critical"}) < 500
histogram_quantile(0.95, temporal_workflow_duration_ms{tier="standard"}) < 1000
histogram_quantile(0.95, temporal_workflow_duration_ms{tier="bulk"}) < 2000
```

### Step 3.3: Alert Rules (Prometheus)

Create `alerts.yml`:

```yaml
groups:
  - name: phase3_temporal
    interval: 1m
    rules:
      - alert: TemporalWorkerNotRunning
        expr: count(temporal_worker_active_total) < 9
        for: 5m
        labels: { severity: critical }
        annotations:
          summary: "Fewer than 9 Temporal workers running (expected 15 with legacies)"

      - alert: QueueInitializationFailed
        expr: count(temporal_task_queue_count) == 0
        for: 2m
        labels: { severity: critical }
        annotations:
          summary: "No task queues detected (initialization failed)"

      - alert: HighQueueUtilization
        expr: max(temporal_worker_concurrent_workflows / temporal_worker_max_concurrent_workflows) > 0.9
        for: 5m
        labels: { severity: warning }
        annotations:
          summary: "Task queue {{ $labels.task_queue }} at {{ $value | humanizePercentage }} utilization"

      - alert: JobRoutingErrors
        expr: increase(temporal_workflow_failed_total{reason="routing_failed"}[5m]) > 5
        for: 5m
        labels: { severity: warning }
        annotations:
          summary: "Multiple job routing failures in last 5 minutes"
```

Load alerts:
```bash
# Stop Prometheus
docker-compose stop prometheus

# Update config with alert rules
cp alerts.yml /etc/prometheus/

# Restart
docker-compose up -d prometheus
```

### Step 3.4: Daily Validation Checklist

**Every 4 hours for 48 hours, run:**

```bash
#!/bin/bash
# validate-phase3.sh

echo "=== Phase 3 Validation Check ==="
date

echo "1. Verify all 9 queues exist"
queue_count=$(temporal task-queue list 2>/dev/null | grep -c "queue")
if [ "$queue_count" -ge 9 ]; then
    echo "✓ $queue_count queues active"
else
    echo "✗ Only $queue_count queues (expected >= 9)"
    exit 1
fi

echo "2. Check for worker errors in logs"
error_count=$(docker logs calendar-service 2>&1 | grep -c "Worker.*failed")
if [ "$error_count" -eq 0 ]; then
    echo "✓ No worker errors"
else
    echo "✗ $error_count worker errors found"
    exit 1
fi

echo "3. Verify service is responsive"
http_code=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8081/health)
if [ "$http_code" -eq 200 ]; then
    echo "✓ Service responding (HTTP $http_code)"
else
    echo "✗ Service unhealthy (HTTP $http_code)"
    exit 1
fi

echo "4. Check Temporal connectivity"
temporal workflow list --task-queue us-east-1-critical-queue > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "✓ Temporal accessible"
else
    echo "✗ Temporal not accessible"
    exit 1
fi

echo ""
echo "✅ All Phase 3 validation checks passed"
```

Run it:
```bash
chmod +x validate-phase3.sh
./validate-phase3.sh

# Schedule hourly
echo "0 * * * * /path/to/validate-phase3.sh" | crontab -
```

---

## Phase 4: Logs & Troubleshooting (If Issues Arise)

### Issue 1: Workers Not Starting

**Symptom**: No task queues visible in Temporal Web

**Debug**:
```bash
# Check logs for registration errors
docker logs calendar-service 2>&1 | grep -E "(Failed|error)" | head -20

# Check WORKER_REGIONS environment variable
docker exec calendar-service env | grep WORKER_REGIONS

# Manually verify env is set
docker exec calendar-service printenv WORKER_REGIONS
# Expected: us-east-1,eu-west-1,ap-southeast-1
```

**Fix**:
```bash
# If env var not set, update docker-compose.yml and redeploy
docker-compose down calendar-service
docker-compose up -d calendar-service

# Verify
docker logs calendar-service 2>&1 | grep "All.*regional workers"
```

### Issue 2: Job Routing to Wrong Queue

**Symptom**: Priority 5 job goes to critical queue instead of standard

**Debug**:
```bash
# Check dispatcher logic
docker logs calendar-service 2>&1 | grep "TaskQueue\|routing" | tail -20

# Verify priority classification in code
grep -A 5 "getPriorityTier" /Users/eganpj/GitHub/semlayer/calendar-service/internal/temporal/dispatcher.go
```

**Fix**: Check that priority classification logic is correct (1-2: critical, 3-7: standard, 8-10: bulk)

### Issue 3: High Latency (>1s)

**Symptom**: Jobs taking >1s to route

**Debug**:
```bash
# Check worker utilization
temporal task-queue describe --task-queue us-east-1-standard-queue
# Look for: "PollerCount" and if tasks are accumulating

# Check if database queries are slow
docker logs calendar-service 2>&1 | grep -i "slow\|timeout\|latency"

# Check cache hit rate
redis-cli INFO stats | grep keyspace_hits
```

**Fix**:
- If utilization > 80%, increase worker pool sizes in `dispatcher.go`
- If cache hit rate low, check REDIS_CACHE_TTL and cache-aside logic

---

## Phase 5: Success Criteria (When to Sign Off)

### ✅ Go/No-Go Decision Matrix

| Criteria | Threshold | Status | Notes |
|----------|-----------|--------|-------|
| Workers Running | 9+ queues | ✓ | All regions × tiers |
| Queue Depth | < 100 pending | ✓ | Normal traffic volume |
| Job Routing Accuracy | 100% | ✓ | Correct priority bins |
| Latency p95 | < 500ms | ✓ | Acceptable delays |
| Cache Hit Rate | > 90% | ✓ | Repeated queries fast |
| Error Rate | < 1% | ✓ | Acceptable failure rate |
| Resource Utilization | < 80% | ✓ | Headroom for spikes |
| Uptime | > 99.9% | ✓ | 48h stable operation |
| No Worker Crashes | 0 restarts | ✓ | Stable deployment |
| Region Isolation | All queues isolated | ✓ | No cross-region bleed |

### 📋 Sign-Off Checklist

```bash
# T+48h Assessment
[ ] All 9 queues actively polling tasks
[ ] Zero worker crashes or restarts
[ ] No error log entries from Phase 3 code
[ ] Job routing 100% accurate across all tiers/regions
[ ] Cache hit rate > 90%
[ ] Latency p95 < 500ms
[ ] Load test (100 concurrent) completed without errors
[ ] Region authorization working correctly
[ ] Temporal Web shows healthy cluster
[ ] Prometheus metrics all positive
[ ] No customer complaints or issues reported

# ✅ APPROVED FOR PRODUCTION
# 👉 Next: Create production deployment runbook
```

---

## Phase 6: Production Promotion (T+48h)

Once staging validation passes all checks:

```bash
# 1. Tag image as production-ready
docker tag calendar-service:phase3 calendar-service:prod-phase3
docker push registry.example.com/calendar-service:prod-phase3

# 2. Create backup of current production
docker tag calendar-service:latest calendar-service:phase2-backup-$(date +%Y%m%d)

# 3. Deploy to production (blue-green or canary)
# See PRODUCTION_DEPLOYMENT_CHECKLIST.md

# 4. Monitor production metrics (24h)
# See PRODUCTION_MONITORING_GUIDE.md
```

---

## Rollback Plan (If Needed)

If anything goes wrong:

```bash
# Immediate rollback to Phase 2
docker tag calendar-service:phase2:latest calendar-service:latest
docker-compose restart calendar-service

# Verify rollback
docker logs calendar-service | grep "Main worker started"

# In-flight jobs will timeout and Temporal will retry them
# No data loss (all state is persisted)
```

---

## Next Steps (After T+48h Validation)

✅ **Phase 3 Validated** → Promote to production  
🔄 **Parallel Development** → Start Phase 4 AI implementation  
📊 **Monitoring** → Set up production dashboards & alerts  
🧪 **Testing** → Create load testing framework  

---

**Current Status**: Ready to execute deploy  
**Estimated Time**: 4 hours for initial validation + 48 hours monitoring  
**Risk Level**: 🟢 LOW (backward compatible, rollback available)  

👉 **Ready to proceed?** Run Section 1.1 to build the Docker image.
