# Production Deployment Guide - Workflow Timeout Triggers

**Duration:** 30 minutes  
**Date:** October 21, 2024  
**Status:** ✅ Ready to Deploy

---

## Overview

This guide provides step-by-step procedures for deploying the Workflow Timeout Triggers feature to production, including:
- Pre-deployment verification
- Database migration execution
- Backend compilation and deployment
- Frontend build and deployment
- Post-deployment verification
- Rollback procedures

---

## Phase 1: Pre-Deployment Verification (5 min)

### 1.1: Verify Current System State

```bash
# Check all services running
echo "=== Backend Service ==="
lsof -i :8080 | head -3

echo "=== Frontend Service ==="
lsof -i :3000 | head -3

echo "=== Database Service ==="
psql postgres://postgres:postgres@host.docker.internal:5432/alpha?sslmode=disable \
  -c "SELECT version();" | head -5

echo "=== Git Status ==="
cd /Users/eganpj/GitHub/semlayer
git status
git branch

echo "=== Environment Variables ==="
env | grep -E "ENV|NODE_ENV|GOENV" || echo "No env vars set"
```

### 1.2: Verify Build Artifacts Exist

```bash
# Check backend dependencies
cd /Users/eganpj/GitHub/semlayer/backend
go mod verify
# Should return: all verified

# Check frontend dependencies
cd /Users/eganpj/GitHub/semlayer/frontend
npm ls | head -20
# Should show package tree without errors

# Verify handler file exists and is current
ls -lh /Users/eganpj/GitHub/semlayer/backend/internal/handlers/timeout_triggers_handler.go
# Should show recent modification date

# Verify database migration file
ls -lh /Users/eganpj/GitHub/semlayer/backend/db/migrations/2025_10_20_workflow_timeout_triggers.sql
# Should exist and be readable
```

### 1.3: Backup Current Database

```bash
# Create backup before migration
DB_BACKUP="/tmp/alpha_backup_$(date +%Y%m%d_%H%M%S).sql"
pg_dump postgres://postgres:postgres@host.docker.internal:5432/alpha \
  --no-password > "$DB_BACKUP"

echo "Database backed up to: $DB_BACKUP"
ls -lh "$DB_BACKUP"

# Verify backup is valid
psql -f "$DB_BACKUP" -c "SELECT COUNT(*) FROM workflow_timeout_triggers;" 2>&1 | tail -5
```

### 1.4: Document Current Versions

```bash
# Record current versions for rollback
echo "=== Version Information ===" > /tmp/deployment_versions.txt

echo "Go Version:" >> /tmp/deployment_versions.txt
go version >> /tmp/deployment_versions.txt

echo "Node Version:" >> /tmp/deployment_versions.txt
node --version >> /tmp/deployment_versions.txt
npm --version >> /tmp/deployment_versions.txt

echo "Database Version:" >> /tmp/deployment_versions.txt
psql --version >> /tmp/deployment_versions.txt

echo "Git Commit:" >> /tmp/deployment_versions.txt
git log -1 --format="%h %s" >> /tmp/deployment_versions.txt

echo "Git Branch:" >> /tmp/deployment_versions.txt
git branch >> /tmp/deployment_versions.txt

cat /tmp/deployment_versions.txt
```

---

## Phase 2: Database Migration (5 min)

### 2.1: Execute Migration

```bash
# Set database connection string
DB_URL="postgres://postgres:postgres@host.docker.internal:5432/alpha?sslmode=disable"

# Apply migration using golang-migrate (if installed)
# OR manually using psql:

psql "$DB_URL" << 'EOF'

-- Create table if not exists
CREATE TABLE IF NOT EXISTS workflow_timeout_triggers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    workflow_name VARCHAR(255) NOT NULL,
    step_name VARCHAR(255) NOT NULL,
    due_hours INTEGER NOT NULL CHECK (due_hours > 0 AND due_hours <= 999),
    trigger_percentages JSONB DEFAULT '[80, 100]'::jsonb,
    actions_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_timeout_triggers_tenant 
    ON workflow_timeout_triggers(tenant_id);

CREATE INDEX IF NOT EXISTS idx_timeout_triggers_tenant_active 
    ON workflow_timeout_triggers(tenant_id, is_active);

CREATE INDEX IF NOT EXISTS idx_timeout_triggers_workflow 
    ON workflow_timeout_triggers(tenant_id, workflow_name, step_name);

-- Verify table created
SELECT tablename FROM pg_tables 
WHERE tablename = 'workflow_timeout_triggers';

-- Verify indexes created
SELECT indexname FROM pg_indexes 
WHERE tablename = 'workflow_timeout_triggers';

EOF

echo "✓ Migration complete"
```

### 2.2: Load Sample Data (Production - Optional)

```bash
# For production deployment, typically you would NOT load sample data
# But if needed for testing/demo, uncomment the following:

# psql "$DB_URL" << 'EOF'
# 
# INSERT INTO workflow_timeout_triggers 
# (tenant_id, workflow_name, step_name, due_hours, actions_json)
# VALUES 
# (
#     '00000000-0000-0000-0000-000000000001',
#     'HireEmployee',
#     'ManagerApproval',
#     48,
#     '[
#       {"percent": 80, "type": "notify", "target": "assignee", "message": "Approval due in 10 hours"},
#       {"percent": 100, "type": "escalate", "target": "hr_director", "message": "Approval overdue, escalating"}
#     ]'::jsonb
# ),
# (
#     '00000000-0000-0000-0000-000000000001',
#     'OrderApproval',
#     'CreditApproval',
#     24,
#     '[
#       {"percent": 80, "type": "notify", "target": "assignee", "message": "Credit check needed"},
#       {"percent": 100, "type": "escalate", "target": "credit_manager", "message": "Order holds for credit"}
#     ]'::jsonb
# );
# 
# EOF

echo "Note: Sample data not loaded for production"
```

### 2.3: Verify Migration Success

```bash
# Verify table structure
psql "$DB_URL" << 'EOF'
\d workflow_timeout_triggers
EOF

# Verify indexes
psql "$DB_URL" << 'EOF'
SELECT indexname, indexdef FROM pg_indexes 
WHERE tablename = 'workflow_timeout_triggers'
ORDER BY indexname;
EOF

# Verify record count
psql "$DB_URL" -c "SELECT COUNT(*) as trigger_count FROM workflow_timeout_triggers;"

echo "✓ Database migration verified"
```

---

## Phase 3: Backend Deployment (10 min)

### 3.1: Build Backend Binary

```bash
# Navigate to backend directory
cd /Users/eganpj/GitHub/semlayer/backend

# Clean previous builds
rm -f /tmp/semlayer-server
rm -f /tmp/semlayer-server.old

# Build new binary
echo "Building backend binary..."
go build -o /tmp/semlayer-server ./cmd/server

# Verify build succeeded
if [ -f /tmp/semlayer-server ]; then
    echo "✓ Backend build successful"
    ls -lh /tmp/semlayer-server
    file /tmp/semlayer-server
else
    echo "✗ Backend build failed"
    exit 1
fi
```

### 3.2: Verify Build Dependencies

```bash
# Verify handler is included
strings /tmp/semlayer-server | grep -i "timeout" | head -5
# Should find references to timeout_triggers_handler

# Verify database package
strings /tmp/semlayer-server | grep -i "sqlx"
# Should find sqlx references

# Verify chi router
strings /tmp/semlayer-server | grep -i "chi"
# Should find chi references
```

### 3.3: Pre-Production Testing

```bash
# Run backend unit tests (if available)
cd /Users/eganpj/GitHub/semlayer/backend

echo "Running unit tests..."
go test ./...

# If tests pass, continue
# If tests fail, review test logs before proceeding
```

### 3.4: Deploy Backend Binary

```bash
# Option A: Local Development
# Stop current backend if running
pkill -f "semlayer-server" || true
sleep 2

# Copy new binary to deployment location
DEPLOY_DIR="${DEPLOY_DIR:-/opt/semlayer}"
sudo mkdir -p "$DEPLOY_DIR"
sudo cp /tmp/semlayer-server "$DEPLOY_DIR/semlayer-server"
sudo chmod +x "$DEPLOY_DIR/semlayer-server"
sudo chown -R app:app "$DEPLOY_DIR"

# Option B: Docker Deployment
# If using Docker, rebuild image:
cd /Users/eganpj/GitHub/semlayer
docker build -t semlayer:latest -f backend/Dockerfile .

# Push to registry (if applicable)
docker push your-registry.azurecr.io/semlayer:latest

# Update deployment manifest:
# kubectl set image deployment/semlayer semlayer=your-registry.azurecr.io/semlayer:latest

echo "✓ Backend binary deployed"
```

### 3.5: Start Backend Service

```bash
# Start backend with new binary
$DEPLOY_DIR/semlayer-server &

# Wait for service to start
sleep 5

# Verify service is running
lsof -i :8080
echo "✓ Backend service started"

# Check for errors in logs
tail -20 /var/log/semlayer/backend.log | tail -5
```

---

## Phase 4: Frontend Deployment (5 min)

### 4.1: Build Frontend Bundle

```bash
# Navigate to frontend directory
cd /Users/eganpj/GitHub/semlayer/frontend

# Clean previous builds
rm -rf dist build

# Install dependencies (if needed)
npm ci

# Build production bundle
echo "Building frontend bundle..."
npm run build

# Verify build succeeded
if [ -d dist ]; then
    echo "✓ Frontend build successful"
    du -sh dist
    find dist -name "*.js" | head -5
else
    echo "✗ Frontend build failed"
    exit 1
fi
```

### 4.2: Verify Frontend Assets

```bash
# Check for required files
if [ -f dist/index.html ]; then
    echo "✓ index.html present"
else
    echo "✗ index.html missing"
    exit 1
fi

# Verify WorkflowTimeoutTriggersPage component
grep -r "WorkflowTimeout" dist/static/js/ || \
    grep -r "workflow-timeout" dist/ || \
    echo "Note: Bundle may use code splitting"

# Check bundle size
echo "Bundle size analysis:"
du -sh dist/static/js/*

echo "✓ Frontend assets verified"
```

### 4.3: Deploy Frontend Assets

```bash
# Option A: Static File Server
# Copy build output to web server
STATIC_DIR="${STATIC_DIR:-/var/www/semlayer}"
sudo mkdir -p "$STATIC_DIR"
sudo cp -r dist/* "$STATIC_DIR/"
sudo chown -R www-data:www-data "$STATIC_DIR"

# Option B: CDN Deployment
# Upload to S3/Azure Blob Storage:
# aws s3 sync dist/ s3://my-bucket/semlayer/ --delete
# az storage blob upload-batch -s dist/ -d \$web -z "50M"

# Option C: Docker Deployment
# Build static image
cat > Dockerfile.frontend << 'DOCKERFILE_END'
FROM nginx:alpine
COPY dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/nginx.conf
EXPOSE 80
DOCKERFILE_END

docker build -t semlayer-frontend:latest -f Dockerfile.frontend .

# Push to registry
docker push your-registry.azurecr.io/semlayer-frontend:latest

echo "✓ Frontend assets deployed"
```

### 4.4: Verify Frontend Service

```bash
# If running locally
lsof -i :3000
echo "✓ Frontend service running"

# If using web server
curl -s http://localhost/index.html | head -20

# Check for 404 errors
echo "✓ Frontend assets accessible"
```

---

## Phase 5: Post-Deployment Verification (3 min)

### 5.1: Health Checks

```bash
# Check backend health endpoint
echo "=== Backend Health Check ==="
curl -s http://localhost:8080/health | jq '.'

# Check API endpoint
echo "=== API Health Check ==="
curl -s -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  http://localhost:8080/api/workflow-timeout-triggers | jq '.' | head -20

# Check frontend
echo "=== Frontend Health Check ==="
curl -s http://localhost:3000/workflow-timeouts | head -30

echo "✓ Health checks passed"
```

### 5.2: Smoke Tests

```bash
# Test 1: List triggers
echo "Test 1: List triggers"
curl -s -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  http://localhost:8080/api/workflow-timeout-triggers | jq '.length'

# Test 2: Create trigger
echo "Test 2: Create trigger"
curl -s -X POST \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -H "Content-Type: application/json" \
  -d '{"workflow_name":"Test","step_name":"Step","due_hours":24,"actions":[]}' \
  http://localhost:8080/api/workflow-timeout-triggers | jq '.id'

# Test 3: Database access
echo "Test 3: Database access"
psql "$DB_URL" -c "SELECT COUNT(*) FROM workflow_timeout_triggers;"

echo "✓ Smoke tests passed"
```

### 5.3: Verify Multi-Tenant Isolation

```bash
# Create trigger with Tenant A
TRIGGER_ID=$(curl -s -X POST \
  -H "X-Tenant-ID: 11111111-1111-1111-1111-111111111111" \
  -H "Content-Type: application/json" \
  -d '{"workflow_name":"TenantA","step_name":"Step","due_hours":24,"actions":[]}' \
  http://localhost:8080/api/workflow-timeout-triggers | jq -r '.id')

# Try to access with Tenant B (should fail)
RESULT=$(curl -s -H "X-Tenant-ID: 22222222-2222-2222-2222-222222222222" \
  http://localhost:8080/api/workflow-timeout-triggers/$TRIGGER_ID | jq '.error')

if [[ "$RESULT" == '"Trigger not found"' ]]; then
    echo "✓ Multi-tenant isolation verified"
else
    echo "✗ Multi-tenant isolation FAILED"
    exit 1
fi
```

### 5.4: Check Application Logs

```bash
# Check for errors
echo "=== Recent Backend Errors ==="
tail -50 /var/log/semlayer/backend.log | grep -i "error" || echo "No errors found"

# Check for warnings
echo "=== Recent Backend Warnings ==="
tail -50 /var/log/semlayer/backend.log | grep -i "warn" || echo "No warnings found"

# Check for successful API calls
echo "=== Recent API Calls ==="
tail -20 /var/log/semlayer/backend.log | grep -i "workflow-timeout" || echo "No recent API calls"

echo "✓ Log verification complete"
```

---

## Phase 6: Performance Verification (2 min)

### 6.1: Load Testing (Optional)

```bash
# Install Apache Bench (if not already installed)
# brew install httpd

# Test 1: List endpoint (100 requests)
echo "Load test: List endpoint"
ab -n 100 -c 10 \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  http://localhost:8080/api/workflow-timeout-triggers | tail -20

# Test 2: Create endpoint
echo "Load test: Create endpoint"
for i in {1..10}; do
  curl -s -X POST \
    -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
    -H "Content-Type: application/json" \
    -d "{\"workflow_name\":\"LoadTest$i\",\"step_name\":\"Step\",\"due_hours\":24,\"actions\":[]}" \
    http://localhost:8080/api/workflow-timeout-triggers > /dev/null
done

echo "✓ Load testing complete"
```

### 6.2: Database Performance

```bash
# Check query performance
psql "$DB_URL" << 'EOF'
EXPLAIN ANALYZE
SELECT id, workflow_name, step_name, due_hours
FROM workflow_timeout_triggers
WHERE tenant_id = '00000000-0000-0000-0000-000000000001'
ORDER BY workflow_name;

-- Check index usage
SELECT schemaname, tablename, indexname, idx_scan, idx_tup_read
FROM pg_stat_user_indexes
WHERE tablename = 'workflow_timeout_triggers'
ORDER BY idx_scan DESC;
EOF

echo "✓ Database performance verified"
```

---

## Phase 7: Monitoring and Alerting Setup

### 7.1: Configure Monitoring

```bash
# Set up log aggregation (e.g., ELK Stack)
# Configure application logs to flow to monitoring system

# Example: Datadog integration
cat > /etc/datadog-agent/conf.d/semlayer.d/conf.yaml << 'EOF'
logs:
  - type: file
    path: /var/log/semlayer/backend.log
    service: semlayer-backend
    source: go
    tags:
      - env:production
EOF

# Example: Azure Monitor
az monitor diagnostic-settings create \
  --resource /subscriptions/[subscription-id]/resourcegroups/[rg]/providers/Microsoft.App/containerApps/semlayer \
  --name semlayer-diagnostics \
  --logs enabled=true \
  --metrics enabled=true

echo "✓ Monitoring configured"
```

### 7.2: Create Alerts

```bash
# Alert for API errors (> 5 errors per minute)
# Alert for database connection failures
# Alert for disk space issues
# Alert for memory usage > 80%

# Example: CloudWatch alarm
aws cloudwatch put-metric-alarm \
  --alarm-name semlayer-api-errors \
  --alarm-description "Alert on API errors" \
  --metric-name APIErrors \
  --namespace Semlayer \
  --statistic Sum \
  --period 300 \
  --threshold 5 \
  --comparison-operator GreaterThanThreshold

echo "✓ Alerts configured"
```

---

## Phase 8: Documentation Updates

### 8.1: Update Runbooks

```bash
# Create/update deployment runbook
cat > /opt/semlayer/DEPLOYMENT_RUNBOOK.md << 'EOF'
# Semlayer Deployment Runbook

## Quick Start
1. Verify all services running: `make health-check`
2. Run tests: `make test`
3. Deploy: `make deploy`
4. Verify: `make verify`

## Rollback
1. Stop services: `make stop`
2. Restore backup: `./scripts/restore-backup.sh [backup-file]`
3. Start services: `make start`
EOF

echo "✓ Documentation updated"
```

### 8.2: Create Incident Response Plan

```bash
# Document incident response procedures
cat > /opt/semlayer/INCIDENT_RESPONSE.md << 'EOF'
# Incident Response Procedures

## Timeout Triggers API Down
1. Check backend health: `curl http://localhost:8080/health`
2. Check database: `psql -c "SELECT 1"`
3. Review logs: `tail -f /var/log/semlayer/backend.log`
4. Restart if needed: `systemctl restart semlayer`

## Database Issues
1. Check connections: `SELECT * FROM pg_stat_activity;`
2. Check disk space: `df -h`
3. Check slow queries: Check query logs
4. Escalate if needed

## High Latency
1. Check system load: `top`
2. Check memory: `free -h`
3. Check database performance: Run EXPLAIN ANALYZE
4. Scale if needed
EOF

echo "✓ Incident response plan created"
```

---

## Phase 9: Rollback Procedures

### 9.1: Quick Rollback (If Issues Detected)

```bash
# Step 1: Stop current services
pkill -f "semlayer-server" || true
sleep 2

# Step 2: Restore previous database backup
DB_BACKUP="/tmp/alpha_backup_[timestamp].sql"
psql "$DB_URL" << EOF
-- Drop new table if exists
DROP TABLE IF EXISTS workflow_timeout_triggers;

-- Restore from backup
$(cat $DB_BACKUP)
EOF

# Step 3: Restore previous backend binary
cp /tmp/semlayer-server.old /tmp/semlayer-server

# Step 4: Restart services
$DEPLOY_DIR/semlayer-server &
sleep 5

# Step 5: Verify
curl -s http://localhost:8080/health | jq '.'

echo "✓ Rollback complete"
```

### 9.2: Full Rollback to Previous Version

```bash
# If rollback is needed, follow these steps:

# 1. Notify stakeholders
echo "Initiating rollback..."

# 2. Stop services
systemctl stop semlayer || pkill -f "semlayer-server"

# 3. Restore database
BACKUP_FILE="/backups/alpha_backup_[previous_date].sql"
pg_restore -d alpha "$BACKUP_FILE"

# 4. Restore frontend
rm -rf /var/www/semlayer
cp -r /backups/semlayer_frontend_[previous_date] /var/www/semlayer

# 5. Restore backend
cp /backups/semlayer_server_[previous_date] $DEPLOY_DIR/semlayer-server

# 6. Start services
systemctl start semlayer

# 7. Verify
curl http://localhost:8080/health

# 8. Document incident
echo "Rollback completed at $(date)" >> /var/log/semlayer/rollback.log

echo "✓ Full rollback complete"
```

---

## Phase 10: Sign-Off and Documentation

### 10.1: Deployment Checklist

```
Pre-Deployment:
✓ Services running and accessible
✓ Git repository clean and committed
✓ Dependencies verified
✓ Database backup created

Database Migration:
✓ Migration applied successfully
✓ Table created with all columns
✓ Indexes created
✓ Schema verified

Backend Deployment:
✓ Binary compiled without errors
✓ Handler included in binary
✓ Service started and listening on port 8080
✓ No errors in logs

Frontend Deployment:
✓ Bundle built successfully
✓ Assets deployed to web server
✓ Service accessible on port 3000
✓ No 404 errors

Post-Deployment:
✓ Health checks pass
✓ Smoke tests pass
✓ Multi-tenant isolation verified
✓ Performance acceptable
✓ Monitoring alerts active

Sign-Off:
✓ Deployment complete
✓ All tests passed
✓ System stable
✓ Ready for user access
```

### 10.2: Create Deployment Report

```bash
# Generate deployment report
cat > /tmp/deployment_report.txt << 'EOF'
DEPLOYMENT REPORT
=================

Deployment Date: $(date)
Deployed By: $(whoami)
Environment: production

Changes Deployed:
- Backend: timeout_triggers_handler.go (335 lines)
- Frontend: WorkflowTimeoutTriggersPage.tsx (updated)
- Database: workflow_timeout_triggers table
- API Routes: 6 new endpoints

Database:
- Migration: 2025_10_20_workflow_timeout_triggers.sql
- Tables: workflow_timeout_triggers
- Indexes: 3 indexes created
- Records: [count from database]

API Endpoints:
- GET /api/workflow-timeout-triggers
- POST /api/workflow-timeout-triggers
- GET /api/workflow-timeout-triggers/{id}
- PUT /api/workflow-timeout-triggers/{id}
- DELETE /api/workflow-timeout-triggers/{id}
- POST /api/workflow-timeout-triggers/{id}/test

Test Results:
✓ Unit tests passed
✓ Smoke tests passed
✓ Load tests acceptable
✓ Multi-tenant isolation verified

Performance:
- API response time: <100ms
- Database query time: <50ms
- Frontend load time: <3s

Rollback Plan:
- Database backup: [backup file location]
- Previous binary: [backup location]
- Frontend backup: [backup location]

Sign-Off:
- Deployment Engineer: ___________
- QA Lead: ___________
- DevOps Lead: ___________
EOF

cat /tmp/deployment_report.txt
```

---

## Troubleshooting

### Issue: Backend fails to start after deployment

**Symptoms:** Port 8080 shows connection refused

**Resolution:**
```bash
# Check if old process still running
lsof -i :8080
pkill -9 -f "semlayer-server"

# Check logs for errors
tail -100 /var/log/semlayer/backend.log | grep -i error

# Verify binary is valid
file $DEPLOY_DIR/semlayer-server
ldd $DEPLOY_DIR/semlayer-server

# Try running in foreground to see errors
$DEPLOY_DIR/semlayer-server

# If error about database connection:
# - Verify database is running
# - Check connection string in config.yaml
# - Verify firewall allows database access
```

### Issue: Frontend shows blank page or 404

**Symptoms:** Website loads but shows errors in console

**Resolution:**
```bash
# Check if static files deployed
ls -la /var/www/semlayer/index.html

# Check web server logs
tail -50 /var/log/nginx/access.log | grep 404

# Check web server configuration
cat /etc/nginx/sites-enabled/semlayer.conf

# Verify CORS settings
curl -i -H "Origin: http://localhost:3000" http://localhost:8080/api/workflow-timeout-triggers

# Clear browser cache
# Ctrl+Shift+Delete to open cache clear dialog
```

### Issue: Database migration fails

**Symptoms:** Migration returns SQL error

**Resolution:**
```bash
# Check if table already exists
psql "$DB_URL" -c "\dt workflow_timeout_triggers"

# If exists, check structure
psql "$DB_URL" -c "\d workflow_timeout_triggers"

# If structure wrong, drop and recreate
psql "$DB_URL" -c "DROP TABLE IF EXISTS workflow_timeout_triggers;"

# Try migration again
psql "$DB_URL" -f /Users/eganpj/GitHub/semlayer/backend/db/migrations/2025_10_20_workflow_timeout_triggers.sql

# Check for constraint violations
psql "$DB_URL" -c "SELECT * FROM information_schema.constraint_column_usage WHERE table_name = 'workflow_timeout_triggers';"
```

---

## Success Criteria

Deployment is considered successful when:

✅ Backend service running and responding to requests  
✅ Frontend application loading and displaying correctly  
✅ All API endpoints returning correct responses  
✅ Database tables created with correct schema  
✅ Multi-tenant isolation verified  
✅ Performance metrics within acceptable ranges  
✅ No errors in application logs  
✅ All smoke tests passing  
✅ Monitoring and alerts active  

---

## Post-Deployment Tasks

1. **Monitor system for 24 hours**
   - Check for errors in logs
   - Monitor CPU and memory usage
   - Check API response times

2. **Gather user feedback**
   - Ask users for any issues
   - Collect feature requests
   - Document bug reports

3. **Update documentation**
   - Document any changes made
   - Update runbooks
   - Update architecture documentation

4. **Schedule retrospective**
   - Review deployment process
   - Identify improvements
   - Plan for next release

---

*Production Deployment Guide - Workflow Timeout Triggers*  
*Status: ✅ READY FOR DEPLOYMENT*
