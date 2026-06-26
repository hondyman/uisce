# Quick Command Reference - E2E Testing & Deployment

**Purpose:** Copy-paste commands for rapid testing and deployment  
**Date:** October 21, 2024

---

## Environment Setup (Required First)

```bash
# Set environment variables
export TENANT_ID="00000000-0000-0000-0000-000000000001"
export DATASOURCE_ID="00000000-0000-0000-0000-000000000001"
export API_BASE="http://localhost:8080"
export DB_URL="postgres://postgres:postgres@host.docker.internal:5432/alpha?sslmode=disable"

# Verify environment
echo "Tenant: $TENANT_ID"
echo "API Base: $API_BASE"
echo "Database: $DB_URL"
```

---

## E2E Testing - Quick Commands (25 min)

### Test 1: List All Triggers
```bash
curl -X GET \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "X-Tenant-Datasource-ID: $DATASOURCE_ID" \
  "$API_BASE/api/workflow-timeout-triggers" | jq '.'

# Expected: 3 triggers (HireEmployee, OrderApproval, InvoiceProcessing)
```

### Test 2: Create New Trigger
```bash
NEW_TRIGGER=$(curl -s -X POST \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "X-Tenant-Datasource-ID: $DATASOURCE_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "workflow_name": "ApprovalProcess",
    "step_name": "VPApproval",
    "due_hours": 36,
    "trigger_percentages": [75, 90, 100],
    "actions": [
      {"percent": 75, "type": "notify", "target": "assignee", "message": "60% deadline"},
      {"percent": 90, "type": "notify", "target": "manager", "message": "90% deadline"},
      {"percent": 100, "type": "escalate", "target": "vp", "message": "Overdue"}
    ]
  }' \
  "$API_BASE/api/workflow-timeout-triggers")

TRIGGER_ID=$(echo $NEW_TRIGGER | jq -r '.id')
echo "Created trigger: $TRIGGER_ID"

# Expected: HTTP 201, new trigger ID returned
```

### Test 3: Get Specific Trigger
```bash
curl -X GET \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "X-Tenant-Datasource-ID: $DATASOURCE_ID" \
  "$API_BASE/api/workflow-timeout-triggers/$TRIGGER_ID" | jq '.'

# Expected: Single trigger object
```

### Test 4: Update Trigger
```bash
curl -X PUT \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "X-Tenant-Datasource-ID: $DATASOURCE_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "workflow_name": "ApprovalProcess",
    "step_name": "VPApproval",
    "due_hours": 48,
    "trigger_percentages": [70, 85, 100],
    "actions": [
      {"percent": 70, "type": "notify", "target": "assignee", "message": "Updated: 70%"},
      {"percent": 85, "type": "notify", "target": "manager", "message": "Updated: 85%"},
      {"percent": 100, "type": "escalate", "target": "vp", "message": "Updated: Overdue"}
    ]
  }' \
  "$API_BASE/api/workflow-timeout-triggers/$TRIGGER_ID" | jq '.'

# Expected: Updated trigger with new due_hours
```

### Test 5: Test Trigger Execution
```bash
curl -X POST \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "X-Tenant-Datasource-ID: $DATASOURCE_ID" \
  "$API_BASE/api/workflow-timeout-triggers/$TRIGGER_ID/test" | jq '.'

# Expected: Success message with action count
```

### Test 6: Delete Trigger
```bash
curl -X DELETE \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "X-Tenant-Datasource-ID: $DATASOURCE_ID" \
  "$API_BASE/api/workflow-timeout-triggers/$TRIGGER_ID" | jq '.'

# Expected: Success message
```

### Test 7: Verify Deletion (Soft Delete)
```bash
# Try to get deleted trigger (should fail)
curl -X GET \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "X-Tenant-Datasource-ID: $DATASOURCE_ID" \
  "$API_BASE/api/workflow-timeout-triggers/$TRIGGER_ID" | jq '.'

# Expected: 404 Not Found

# But verify it still exists in database with is_active=false
psql "$DB_URL" -c "SELECT id, is_active FROM workflow_timeout_triggers WHERE id = '$TRIGGER_ID';"
```

### Test 8: Error Handling - Missing Header
```bash
curl -X GET "$API_BASE/api/workflow-timeout-triggers" | jq '.'

# Expected: 400 "X-Tenant-ID header is required"
```

### Test 9: Error Handling - Cross-Tenant Access
```bash
OTHER_TENANT="99999999-9999-9999-9999-999999999999"

curl -X GET \
  -H "X-Tenant-ID: $OTHER_TENANT" \
  "$API_BASE/api/workflow-timeout-triggers/$TRIGGER_ID" | jq '.'

# Expected: 404 Not Found (should not return data)
```

### Test 10: Database Verification
```bash
# Verify record count
psql "$DB_URL" -c "SELECT COUNT(*) FROM workflow_timeout_triggers WHERE tenant_id = '$TENANT_ID';"

# Verify indexes exist
psql "$DB_URL" -c "SELECT indexname FROM pg_indexes WHERE tablename = 'workflow_timeout_triggers';"

# Verify audit log
psql "$DB_URL" -c "SELECT workflow_name, action, details FROM workflow_audit_log WHERE action = 'timeout_trigger_test' ORDER BY created_at DESC LIMIT 3;"
```

---

## Production Deployment - Quick Commands (30 min)

### Phase 1: Pre-Deployment (5 min)

```bash
# Verify services running
echo "=== Backend Service ==="
lsof -i :8080 | head -2

echo "=== Frontend Service ==="
lsof -i :3000 | head -2

echo "=== Database Service ==="
psql "$DB_URL" -c "SELECT version();" | head -1

# Create database backup
BACKUP_FILE="/tmp/alpha_backup_$(date +%Y%m%d_%H%M%S).sql"
pg_dump "$DB_URL" > "$BACKUP_FILE"
echo "Backup created: $BACKUP_FILE"
ls -lh "$BACKUP_FILE"
```

### Phase 2: Database Migration (5 min)

```bash
# Execute migration
psql "$DB_URL" << 'EOF'
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

CREATE INDEX IF NOT EXISTS idx_timeout_triggers_tenant 
    ON workflow_timeout_triggers(tenant_id);

CREATE INDEX IF NOT EXISTS idx_timeout_triggers_tenant_active 
    ON workflow_timeout_triggers(tenant_id, is_active);

CREATE INDEX IF NOT EXISTS idx_timeout_triggers_workflow 
    ON workflow_timeout_triggers(tenant_id, workflow_name, step_name);

SELECT 'Migration complete' as status;
EOF

# Verify migration
psql "$DB_URL" -c "SELECT tablename FROM pg_tables WHERE tablename = 'workflow_timeout_triggers';"
```

### Phase 3: Backend Deployment (10 min)

```bash
# Navigate to backend
cd /Users/eganpj/GitHub/semlayer/backend

# Clean previous builds
rm -f /tmp/semlayer-server /tmp/semlayer-server.old

# Build new binary
echo "Building backend..."
go build -o /tmp/semlayer-server ./cmd/server

# Verify build
if [ -f /tmp/semlayer-server ]; then
    echo "✓ Build successful"
    ls -lh /tmp/semlayer-server
    file /tmp/semlayer-server
else
    echo "✗ Build failed"
    exit 1
fi

# Stop current backend
pkill -f "semlayer-server" || true
sleep 2

# Deploy binary
DEPLOY_DIR="/opt/semlayer"
sudo mkdir -p "$DEPLOY_DIR"
sudo cp /tmp/semlayer-server "$DEPLOY_DIR/semlayer-server"
sudo chmod +x "$DEPLOY_DIR/semlayer-server"

# Start backend
$DEPLOY_DIR/semlayer-server &
sleep 5

# Verify backend running
lsof -i :8080
echo "✓ Backend deployed"
```

### Phase 4: Frontend Deployment (5 min)

```bash
# Navigate to frontend
cd /Users/eganpj/GitHub/semlayer/frontend

# Clean previous builds
rm -rf dist build

# Build production bundle
echo "Building frontend..."
npm run build

# Verify build
if [ -d dist ]; then
    echo "✓ Build successful"
    du -sh dist
else
    echo "✗ Build failed"
    exit 1
fi

# Deploy assets
STATIC_DIR="/var/www/semlayer"
sudo mkdir -p "$STATIC_DIR"
sudo cp -r dist/* "$STATIC_DIR/"
sudo chown -R www-data:www-data "$STATIC_DIR"

echo "✓ Frontend deployed"
```

### Phase 5: Verification (5 min)

```bash
# Health check
echo "=== Health Checks ==="
curl -s http://localhost:8080/health | jq '.' && echo "✓ Backend healthy" || echo "✗ Backend down"

# API test
echo "=== API Test ==="
curl -s -H "X-Tenant-ID: $TENANT_ID" \
  http://localhost:8080/api/workflow-timeout-triggers | jq 'length' && echo "✓ API working"

# Database test
echo "=== Database Test ==="
psql "$DB_URL" -c "SELECT COUNT(*) FROM workflow_timeout_triggers;" && echo "✓ Database working"

# Frontend test
echo "=== Frontend Test ==="
curl -s http://localhost:3000/index.html | grep -q "<!DOCTYPE" && echo "✓ Frontend working"

# Smoke test - Create trigger
echo "=== Smoke Test ==="
curl -s -X POST \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json" \
  -d '{"workflow_name":"Test","step_name":"Step","due_hours":24,"actions":[]}' \
  http://localhost:8080/api/workflow-timeout-triggers | jq '.id' && echo "✓ API POST working"

echo "=== All verification checks passed ==="
```

---

## Common Troubleshooting Commands

### Check Backend Logs
```bash
# View last 50 lines
tail -50 /var/log/semlayer/backend.log

# Follow logs in real-time
tail -f /var/log/semlayer/backend.log | grep -i error

# Find errors in last hour
grep "error" /var/log/semlayer/backend.log | tail -20
```

### Check Database Status
```bash
# Connection test
psql "$DB_URL" -c "SELECT 1;" && echo "✓ Connected"

# Table structure
psql "$DB_URL" -c "\d workflow_timeout_triggers"

# Record count
psql "$DB_URL" -c "SELECT COUNT(*) FROM workflow_timeout_triggers;"

# Query performance
psql "$DB_URL" -c "EXPLAIN ANALYZE SELECT * FROM workflow_timeout_triggers WHERE tenant_id = '$TENANT_ID';"
```

### Check Ports
```bash
# List listening ports
lsof -i -P -n | grep LISTEN

# Check specific port
lsof -i :8080

# Kill process on port
lsof -i :8080 | tail -1 | awk '{print $2}' | xargs kill -9
```

### Frontend Issues
```bash
# Clear browser cache
# Ctrl+Shift+Delete (Windows/Linux) or Cmd+Shift+Delete (Mac)

# Check console errors
# Open DevTools: Cmd+Option+I (Mac) or Ctrl+Shift+I (Windows/Linux)

# Check web server logs
tail -50 /var/log/nginx/access.log
tail -50 /var/log/nginx/error.log
```

---

## Quick Status Check Script

```bash
#!/bin/bash
# Save as: check-status.sh

echo "=== Workflow Timeout Triggers - Status Check ==="
echo "Time: $(date)"
echo ""

echo "Backend:"
lsof -i :8080 > /dev/null && echo "  ✓ Running" || echo "  ✗ Down"
curl -s http://localhost:8080/health > /dev/null && echo "  ✓ Healthy" || echo "  ✗ Unhealthy"

echo ""
echo "Frontend:"
lsof -i :3000 > /dev/null && echo "  ✓ Running" || echo "  ✗ Down"
curl -s http://localhost:3000 > /dev/null && echo "  ✓ Accessible" || echo "  ✗ Not accessible"

echo ""
echo "Database:"
psql "$DB_URL" -c "SELECT 1;" > /dev/null 2>&1 && echo "  ✓ Connected" || echo "  ✗ Not connected"
TRIGGER_COUNT=$(psql "$DB_URL" -t -c "SELECT COUNT(*) FROM workflow_timeout_triggers WHERE is_active = true;" 2>/dev/null)
echo "  Active triggers: $TRIGGER_COUNT"

echo ""
echo "API:"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "X-Tenant-ID: $TENANT_ID" http://localhost:8080/api/workflow-timeout-triggers)
if [ "$HTTP_CODE" = "200" ]; then
    echo "  ✓ Responding ($HTTP_CODE)"
else
    echo "  ✗ Error ($HTTP_CODE)"
fi

echo ""
echo "=== Status Check Complete ==="
```

**Run it:**
```bash
chmod +x check-status.sh
./check-status.sh
```

---

## Performance Testing Commands

### Response Time Test
```bash
# Single request timing
time curl -s -H "X-Tenant-ID: $TENANT_ID" \
  "$API_BASE/api/workflow-timeout-triggers" > /dev/null

# 10 requests (measure average)
for i in {1..10}; do
  time curl -s -H "X-Tenant-ID: $TENANT_ID" \
    "$API_BASE/api/workflow-timeout-triggers" > /dev/null
done
```

### Load Test (if Apache Bench installed)
```bash
# 100 requests with 10 concurrent
ab -n 100 -c 10 \
  -H "X-Tenant-ID: $TENANT_ID" \
  "$API_BASE/api/workflow-timeout-triggers"
```

### Database Query Performance
```bash
# Analyze query plan
psql "$DB_URL" << 'EOF'
EXPLAIN ANALYZE
SELECT id, workflow_name, step_name
FROM workflow_timeout_triggers
WHERE tenant_id = 'UUID'
ORDER BY workflow_name;
EOF
```

---

## Rollback Commands

### Quick Rollback (5 min)
```bash
# Stop services
pkill -f "semlayer-server" || true

# Restore database
BACKUP_FILE="/tmp/alpha_backup_YYYYMMDD_HHMMSS.sql"
psql "$DB_URL" < "$BACKUP_FILE"

# Restart
cd /Users/eganpj/GitHub/semlayer/backend
go build -o /tmp/semlayer-server ./cmd/server
/tmp/semlayer-server &
```

---

## Multi-Tenant Testing

### Create Triggers for Multiple Tenants
```bash
TENANT_1="11111111-1111-1111-1111-111111111111"
TENANT_2="22222222-2222-2222-2222-222222222222"

# Create for Tenant 1
curl -s -X POST \
  -H "X-Tenant-ID: $TENANT_1" \
  -H "Content-Type: application/json" \
  -d '{"workflow_name":"Tenant1WF","step_name":"Step","due_hours":24,"actions":[]}' \
  "$API_BASE/api/workflow-timeout-triggers" | jq '.id'

# Create for Tenant 2
curl -s -X POST \
  -H "X-Tenant-ID: $TENANT_2" \
  -H "Content-Type: application/json" \
  -d '{"workflow_name":"Tenant2WF","step_name":"Step","due_hours":24,"actions":[]}' \
  "$API_BASE/api/workflow-timeout-triggers" | jq '.id'

# Verify isolation - Tenant 1 should only see Tenant 1 data
echo "Tenant 1 triggers:"
curl -s -H "X-Tenant-ID: $TENANT_1" \
  "$API_BASE/api/workflow-timeout-triggers" | jq '.[] | .workflow_name'

echo "Tenant 2 triggers:"
curl -s -H "X-Tenant-ID: $TENANT_2" \
  "$API_BASE/api/workflow-timeout-triggers" | jq '.[] | .workflow_name'
```

---

## Documentation Reference

| Document | Purpose | Time | Usage |
|----------|---------|------|-------|
| E2E_TESTING_PROCEDURES.md | Detailed test procedures | 25 min | Copy SQL/curl commands from this reference |
| PRODUCTION_DEPLOYMENT_GUIDE.md | Step-by-step deployment | 30 min | Follow phases 1-10 in order |
| WORKFLOW_TIMEOUT_TRIGGERS_SUMMARY.md | System overview | 5 min | Quick reference for API/schema |
| QUICK_COMMAND_REFERENCE.md | This file | - | Copy-paste commands for quick execution |

---

**Quick Command Reference - Ready to Use**  
**Date: October 21, 2024**
