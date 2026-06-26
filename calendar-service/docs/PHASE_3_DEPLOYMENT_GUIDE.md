# Phase 3 Deployment Guide

**Purpose:** Step-by-step instructions for deploying tenant-aware calendar service to production

**Audience:** DevOps engineers, backend developers, platform operators

---

## Pre-Deployment Checklist

### Code Verification

```bash
# 1. Compile all services
cd calendar-service
go build ./internal/services
go build ./internal/repository
go build ./internal/api
go build ./internal/middleware

# Expected: Clean compilation with no errors
```

### Test Verification

```bash
# 2. Run all tests
go test ./internal/services/... -v
go test ./internal/repository/... -v
go test ./internal/api/... -v

# Expected: All tests passing
# Look for specific output:
# ✓ TestPhase3CalendarCreateWithTenant
# ✓ TestPhase3CrossTenantAccessDenied
# ✓ TestPhase3HandlerCrossTenanAccessBlocked
# ... (22+ tests total)
```

### Dependencies

```bash
# 3. Verify all Go modules present
go mod tidy
go mod download

# Required modules:
# - github.com/golang-jwt/jwt/v5
# - github.com/jackc/pgx/v5
# - github.com/sirupsen/logrus
# - github.com/google/uuid
```

---

## Database Setup

### 1. Connect to PostgreSQL

```bash
# Local development
psql postgresql://localhost:5432/calendar

# Production (update credentials)
psql postgresql://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:5432/${DB_NAME}
```

### 2. Create Calendar Table

Run this SQL to create the calendar table with tenant isolation:

```sql
CREATE TABLE IF NOT EXISTS calendars (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    timezone VARCHAR(64) DEFAULT 'UTC',
    created_by VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by VARCHAR(255),
    updated_at TIMESTAMP,
    deleted_at TIMESTAMP,
    
    -- Constraints
    CONSTRAINT calendars_tenant_not_null CHECK (tenant_id IS NOT NULL)
);

-- Verify table created
\dt calendars
-- Expected output: calendars | table | postgres

-- View columns
\d calendars
-- Expected: All columns listed with correct types
```

### 3. Create Indexes

These indexes are **CRITICAL** for performance and tenant isolation:

```sql
-- PRIMARY TENANT ISOLATION INDEX
-- This is the most important index - ensures logical isolation
CREATE UNIQUE INDEX idx_calendars_tenant_id 
    ON calendars(tenant_id, id) 
    WHERE deleted_at IS NULL;

-- QUERY OPTIMIZATION INDEXES
-- Speed up list operations
CREATE INDEX idx_calendars_tenant_created 
    ON calendars(tenant_id, created_at DESC);

-- Speed up recent updates
CREATE INDEX idx_calendars_tenant_updated 
    ON calendars(tenant_id, updated_at DESC);

-- Speed up soft-delete queries
CREATE INDEX idx_calendars_deleted 
    ON calendars(tenant_id, deleted_at);

-- Verify indexes created
\di
-- Expected: idx_calendars_* indexes listed
```

### 4. Enable Row-Level Security (RLS)

RLS provides database-level tenant isolation:

```sql
-- Enable RLS on calendars table
ALTER TABLE calendars ENABLE ROW LEVEL SECURITY;

-- Create tenant isolation policy
CREATE POLICY calendars_tenant_isolation ON calendars
    USING (tenant_id = current_setting('app.current_tenant_id'))
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id'));

-- Verify RLS enabled
SELECT schemaname, tablename, rowsecurity 
FROM pg_tables 
WHERE tablename = 'calendars';
-- Expected: rowsecurity = true
```

### 5. Test Database Connection

```sql
-- Insert test data
INSERT INTO calendars (tenant_id, name, description, timezone, created_by)
VALUES 
    ('test-tenant-a', 'Calendar A', 'Test calendar for A', 'UTC', 'admin'),
    ('test-tenant-b', 'Calendar B', 'Test calendar for B', 'UTC', 'admin');

-- Verify data inserted
SELECT id, tenant_id, name FROM calendars;
-- Expected: 2 rows with correct tenant_ids

-- Test RLS policy (requires connection as service user)
SET app.current_tenant_id TO 'test-tenant-a';
SELECT id, tenant_id, name FROM calendars;
-- Expected: Only 1 row (tenant-a calendar)

RESET app.current_tenant_id;
```

---

## Environment Configuration

### Required Environment Variables

Create `.env` file in project root:

```bash
# =============================================================================
# Database Configuration
# =============================================================================
DATABASE_URL=postgresql://calendar_user:secure_password@db.prod.example.com:5432/calendar_production
DB_MAX_CONNECTIONS=20
DB_MIN_CONNECTIONS=5
DB_STATEMENT_CACHE_SIZE=50

# =============================================================================
# JWT Security Configuration
# =============================================================================
JWT_SECRET=your-super-secret-key-at-least-32-characters-long
JWT_EXPIRATION_MINUTES=60
JWT_REFRESH_EXPIRATION_DAYS=7

# =============================================================================
# Server Configuration
# =============================================================================
SERVER_HOST=0.0.0.0
SERVER_PORT=8080
SERVER_TIMEOUT_SECONDS=30

# =============================================================================
# Logging Configuration
# =============================================================================
LOG_LEVEL=info
LOG_FORMAT=json
LOG_FILE=/var/log/calendar-service/service.log

# =============================================================================
# Audit Configuration
# =============================================================================
AUDIT_LOG_ENABLED=true
AUDIT_LOG_FILE=/var/log/calendar-service/audit.log
AUDIT_LOG_RETENTION_DAYS=365

# =============================================================================
# Performance Tuning
# =============================================================================
CACHE_ENABLED=true
CACHE_TTL_SECONDS=300
CACHE_MAX_SIZE_MB=256

# =============================================================================
# Monitoring & Metrics
# =============================================================================
METRICS_ENABLED=true
METRICS_PORT=9090
TRACE_ENABLED=true
TRACE_SAMPLE_RATE=0.1
```

### For Production

Use environment variables from secret manager:

```bash
# AWS Secrets Manager
aws secretsmanager get-secret-value --secret-id calendar-service/prod --region us-east-1

# Kubernetes Secrets
kubectl get secret calendar-service-secrets -o yaml

# HashiCorp Vault
vault kv get secret/calendar-service/prod
```

---

## Docker Deployment

### Dockerfile

```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY . .

# Build service
RUN go build -o calendar-service ./cmd/calendar-service

# Final image (minimal)
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/calendar-service .

# Create non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

USER appuser

EXPOSE 8080

CMD ["./calendar-service"]
```

### Docker Compose

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: calendar_production
      POSTGRES_USER: calendar_user
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./migrations:/docker-entrypoint-initdb.d
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U calendar_user"]
      interval: 10s
      timeout: 5s
      retries: 5

  calendar-service:
    build: .
    environment:
      DATABASE_URL: "postgresql://calendar_user:${DB_PASSWORD}@postgres:5432/calendar_production"
      JWT_SECRET: ${JWT_SECRET}
      LOG_LEVEL: info
    ports:
      - "8080:8080"
    depends_on:
      postgres:
        condition: service_healthy
    volumes:
      - ./logs:/app/logs

volumes:
  postgres_data:
```

### Build & Deploy

```bash
# 1. Build Docker image
docker build -t calendar-service:v3.0.0 .

# 2. Tag for registry
docker tag calendar-service:v3.0.0 registry.example.com/calendar-service:v3.0.0

# 3. Push to registry
docker push registry.example.com/calendar-service:v3.0.0

# 4. Deploy locally (development)
docker-compose up -d

# 5. Stop services
docker-compose down
```

---

## Kubernetes Deployment

### Namespace & Secrets

```bash
# 1. Create namespace
kubectl create namespace calendar-service

# 2. Create secrets
kubectl create secret generic calendar-service-secrets \
  --from-literal=db-password=secure_password \
  --from-literal=jwt-secret=your-secret-key \
  -n calendar-service
```

### Deployment Manifest

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: calendar-service
  namespace: calendar-service
spec:
  replicas: 3
  selector:
    matchLabels:
      app: calendar-service
  template:
    metadata:
      labels:
        app: calendar-service
    spec:
      containers:
      - name: calendar-service
        image: registry.example.com/calendar-service:v3.0.0
        ports:
        - containerPort: 8080
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: calendar-service-secrets
              key: db-url
        - name: JWT_SECRET
          valueFrom:
            secretKeyRef:
              name: calendar-service-secrets
              key: jwt-secret
        - name: LOG_LEVEL
          value: "info"
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 500m
            memory: 512Mi
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: calendar-service
  namespace: calendar-service
spec:
  selector:
    app: calendar-service
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8080
  type: LoadBalancer
```

### Deploy to Kubernetes

```bash
# 1. Apply deployment
kubectl apply -f deployment.yaml

# 2. Check rollout status
kubectl rollout status deployment/calendar-service -n calendar-service

# 3. View pods
kubectl get pods -n calendar-service

# 4. Check logs
kubectl logs deployment/calendar-service -n calendar-service

# 5. Port forward for testing
kubectl port-forward service/calendar-service 8080:80 -n calendar-service
```

---

## Deployment Verification

### 1. Service Health Checks

```bash
# Check service is responding
curl -v http://localhost:8080/health

# Expected:
# < HTTP/1.1 200 OK
# {"status":"healthy","version":"3.0.0"}
```

### 2. Database Connection Test

```bash
# Test via service
curl -X POST http://localhost:8080/api/v1/test-db \
  -H "Authorization: Bearer <jwt-token>"

# Expected:
# {"database":"connected","message":"Connection OK"}
```

### 3. Cross-Tenant Isolation Test

```bash
# Set up JWT tokens for two tenants
TENANT_A_TOKEN=$(./scripts/generate-jwt.sh tenant-a user-a)
TENANT_B_TOKEN=$(./scripts/generate-jwt.sh tenant-b user-b)

# Tenant A creates calendar
CALENDAR_ID=$(curl -X POST http://localhost:8080/api/v1/calendars \
  -H "Authorization: Bearer $TENANT_A_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Calendar A",
    "timezone": "UTC"
  }' | jq -r '.id')

echo "Created calendar: $CALENDAR_ID for Tenant A"

# Attempt: Tenant B tries to access Tenant A's calendar
curl -X GET http://localhost:8080/api/v1/calendars/$CALENDAR_ID \
  -H "Authorization: Bearer $TENANT_B_TOKEN"

# Expected: 403 Forbidden or 404 Not Found
# NOT 200 OK with the calendar data
```

### 4. Audit Logging Test

```bash
# Check audit logs for user attribution
tail -f /var/log/calendar-service/audit.log

# Expected output:
# {"timestamp":"2026-02-18T10:30:45Z","tenant_id":"tenant-a","user_id":"user-a","action":"create_calendar","resource_id":"...","resource_type":"calendar"}
```

### 5. Performance Testing

```bash
# Load test the service
ab -n 1000 -c 10 \
  -H "Authorization: Bearer $TENANT_A_TOKEN" \
  http://localhost:8080/api/v1/calendars

# Expected:
# Requests per second: > 100
# Average response time: < 50ms
# Failed requests: 0
```

### 6. Concurrent Multi-Tenant Test

```bash
#!/bin/bash
# Run concurrent operations from 5 different tenants

for tenant in {1..5}; do
  TOKEN=$(./scripts/generate-jwt.sh tenant-$tenant user-$tenant)
  
  # Create 10 calendars per tenant concurrently
  for i in {1..10}; do
    curl -X POST http://localhost:8080/api/v1/calendars \
      -H "Authorization: Bearer $TOKEN" \
      -H "Content-Type: application/json" \
      -d "{\"name\":\"Calendar $i\",\"timezone\":\"UTC\"}" \
      & # Run in background
  done
done

wait # Wait for all background jobs

echo "Concurrent test complete"

# Verify data isolation
for tenant in {1..5}; do
  TOKEN=$(./scripts/generate-jwt.sh tenant-$tenant user-$tenant)
  
  COUNT=$(curl -s -H "Authorization: Bearer $TOKEN" \
    http://localhost:8080/api/v1/calendars | jq '.calendars | length')
  
  echo "Tenant $tenant has $COUNT calendars (expected: 10)"
done
```

---

## Monitoring & Observability

### Logs Location

```bash
# Application logs
/var/log/calendar-service/service.log

# Audit logs
/var/log/calendar-service/audit.log

# View recent errors
grep ERROR /var/log/calendar-service/service.log | tail -20

# Watch real-time logs
tail -f /var/log/calendar-service/service.log
```

### Key Metrics to Monitor

```bash
# 1. Cross-tenant access attempts (should be zero)
grep "access denied" /var/log/calendar-service/service.log | wc -l
# Expected: 0

# 2. Tenant isolation violations (should be zero)
grep "cross-tenant" /var/log/calendar-service/service.log | wc -l
# Expected: 0

# 3. Response time by endpoint
grep "GET.*calendars" /var/log/calendar-service/service.log | \
  jq '.response_time_ms' | sort -n | tail

# 4. Active tenants
grep "tenant_id" /var/log/calendar-service/audit.log | \
  jq '.tenant_id' | sort | uniq | wc -l
```

### Prometheus Metrics (if enabled)

```bash
# Export metrics
curl http://localhost:9090/metrics

# Key metrics:
# - calendar_service_queries_total{tenant_id="...",operation="..."}
# - calendar_service_response_time_ms{endpoint="..."}
# - calendar_service_cross_tenant_denials_total
```

---

## Rollback Plan

### If Issues Detected

```bash
# 1. Immediate rollback to previous version
docker run -d \
  --name calendar-service-v2 \
  -e DATABASE_URL=$DATABASE_URL \
  registry.example.com/calendar-service:v2.0.0

# 2. Update load balancer to route to v2
# (Update DNS/routing configuration)

# 3. Stop v3 containers
docker stop calendar-service-v3

# 4. Investigate issues in logs
docker logs calendar-service-v3 > /tmp/v3-error.log

# 5. After fix, redeploy:
docker build -t calendar-service:v3.0.1 .
docker push registry.example.com/calendar-service:v3.0.1
```

### Kubernetes Rollback

```bash
# 1. View rollout history
kubectl rollout history deployment/calendar-service -n calendar-service

# 2. Rollback to previous version
kubectl rollout undo deployment/calendar-service -n calendar-service

# 3. Verify rollback
kubectl rollout status deployment/calendar-service -n calendar-service

# 4. Check running pods
kubectl get pods -n calendar-service
```

---

## Post-Deployment Tasks

### 1. Smoke Tests

```bash
# Run smoke test suite
go test ./tests/smoke/...

# Expected: All smoke tests passing
```

### 2. Update Documentation

```bash
# Update deployment documentation
echo "Deployment date: $(date)" >> docs/DEPLOYMENT_HISTORY.md
echo "Version: v3.0.0" >> docs/DEPLOYMENT_HISTORY.md
```

### 3. Notify Team

- [ ] Backend team notified of deployment
- [ ] Monitoring team alerted to watch for issues
- [ ] Support team updated with new endpoints
- [ ] Security team notified of security updates

### 4. Enable Monitoring

```bash
# Verify monitoring is active
curl http://monitoring.example.com/api/services/calendar-service

# Expected: Service showing as "operational"
```

---

## Troubleshooting

### Issue: "access denied" errors in logs

**Likely Cause:** JWT secret mismatch

**Solution:**
```bash
# Verify JWT secret matches across services
echo $JWT_SECRET

# If incorrect, update secret and restart:
export JWT_SECRET="correct-secret-key"
docker-compose restart calendar-service
```

### Issue: "cross-tenant" access in logs

**Likely Cause:** Query missing tenant_id filter

**Solution:**
```bash
# Check recent code changes
git log -p internal/repository/ | grep -A5 "WHERE"

# Look for queries missing "tenant_id = "
# Fix by adding tenant filter

# Redeploy
docker build -t calendar-service:v3.0.1 .
```

### Issue: Database connection timeout

**Likely Cause:** Database credentials or network

**Solution:**
```bash
# Test database connectivity
psql $DATABASE_URL -c "SELECT 1"

# Check network connectivity
ping $(echo $DATABASE_URL | cut -d@ -f2 | cut -d: -f1)

# Verify credentials
# Update DATABASE_URL if incorrect
```

### Issue: Performance degradation

**Likely Cause:** Missing indexes or slow queries

**Solution:**
```sql
-- Check index usage
SELECT * FROM pg_stat_user_indexes WHERE idx_name LIKE 'idx_calendars%';

-- Add missing indexes
CREATE INDEX idx_name ON table_name(column_name);

-- Analyze query performance
EXPLAIN ANALYZE SELECT * FROM calendars WHERE tenant_id = 'test';
```

---

## Production Support Contact

| Issue | Contact | Response Time |
|-------|---------|---|
| Service Down | oncall@team | 5 minutes |
| Performance Issue | platform@team | 15 minutes |
| Cross-tenant Access | security@team | 1 hour |
| Bug Report | bugs@team | 24 hours |

---

**Deployment Status: READY FOR PRODUCTION**

Created: 2026-02-18  
Last Updated: 2026-02-18  
Version: 3.0.0
