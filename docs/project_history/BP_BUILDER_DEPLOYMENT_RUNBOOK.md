# Business Process Builder - Deployment Runbook

**Date:** October 21, 2025  
**Estimated Duration:** 30 minutes  
**Risk Level:** Low  
**Rollback Plan:** Available (see end of document)

---

## 🎯 Pre-Deployment Checklist

### Environment Verification

```bash
# 1. PostgreSQL running
psql --version
# Expected: psql (PostgreSQL) 11.x or higher

# 2. Go installed
go version
# Expected: go1.16 or higher

# 3. Node.js installed
node --version
# Expected: v14.x or higher

# 4. Temporal Server running
curl http://localhost:7233
# Expected: Connection successful (Temporal gRPC port)

# 5. Git status clean
git status
# Expected: "working tree clean"
```

### Database Backup

```bash
# Create backup before migration
pg_dump -U postgres -d alpha > alpha_backup_$(date +%Y%m%d_%H%M%S).sql

# Verify backup size (should be >10MB for full database)
ls -lh alpha_backup_*.sql
```

### Code Compilation Verification

```bash
# Backend compilation
cd /Users/eganpj/GitHub/semlayer/backend
go build ./...
# Expected: No errors

# Frontend compilation
cd /Users/eganpj/GitHub/semlayer/frontend
npx tsc --noEmit
# Expected: No errors
```

---

## 📦 Phase 1: Database Migration (5 minutes)

### Execute Migration

```bash
# Run the migration script
psql -U postgres -d alpha -f /Users/eganpj/GitHub/semlayer/backend/db/migrations/bp_builder_schema.sql

# Output should show:
# CREATE TABLE
# CREATE INDEX
# GRANT
# ... (8 tables total)
```

### Verify Schema

```bash
# Connect to database
psql -U postgres -d alpha

# List new tables (from psql prompt)
\dt bp_*

# Expected output:
# business_processes
# bp_steps
# bp_step_validations
# bp_step_approvers
# bp_executions
# bp_execution_steps
# bp_audit_trail
# bp_notifications_log

# Check indexes
\di bp_*

# Expected: 12+ indexes

# Verify grants
\dp bp_*

# Expected: app_user has SELECT/INSERT/UPDATE permissions

# Exit psql
\q
```

### Rollback Migration (if needed)

```bash
# If migration fails, execute rollback:
psql -U postgres -d alpha -c "
DROP TABLE IF EXISTS bp_notifications_log CASCADE;
DROP TABLE IF EXISTS bp_audit_trail CASCADE;
DROP TABLE IF EXISTS bp_execution_steps CASCADE;
DROP TABLE IF EXISTS bp_executions CASCADE;
DROP TABLE IF EXISTS bp_step_approvers CASCADE;
DROP TABLE IF EXISTS bp_step_validations CASCADE;
DROP TABLE IF EXISTS bp_steps CASCADE;
DROP TABLE IF EXISTS business_processes CASCADE;
"
```

---

## 🔗 Phase 2: Backend Integration (10 minutes)

### 2.1: Copy Handler File

```bash
# Verify file exists
ls -l /Users/eganpj/GitHub/semlayer/backend/api/handlers/bp_handler.go

# File should be 453 lines, 0 compilation errors
wc -l /Users/eganpj/GitHub/semlayer/backend/api/handlers/bp_handler.go
```

### 2.2: Register Routes in Main

Edit `backend/main.go` and add the BP routes registration:

```go
package main

import (
    // ... existing imports
    handlers "github.com/eganpj/semlayer/backend/api/handlers"
)

func main() {
    // ... existing setup code
    
    // Initialize router
    router := gin.Default()
    
    // Initialize database connection
    db := setupDatabase() // your existing function
    
    // ✅ ADD THIS LINE:
    handlers.RegisterBPRoutes(router, db)
    
    // ... rest of your code
}
```

### 2.3: Register Temporal Workflow

Edit your Temporal worker setup file (typically `worker/main.go` or `worker/worker.go`):

```go
package worker

import (
    // ... existing imports
    workflows "github.com/eganpj/semlayer/backend/pkg/workflows"
)

func setupWorker(client client.Client) error {
    w := worker.New(client, "bp_workflow_queue", worker.Options{})
    
    // ✅ ADD THESE LINES:
    w.RegisterWorkflow(workflows.DynamicBPWorkflow)
    
    // Create activities instance
    activities := &workflows.DynamicBPActivities{
        BPService: bpService, // your existing BP service
    }
    
    // Register activities
    w.RegisterActivity(activities.ActivityExecuteValidation)
    w.RegisterActivity(activities.ActivityExecuteApproval)
    w.RegisterActivity(activities.ActivitySendNotification)
    w.RegisterActivity(activities.ActivityCallIntegration)
    w.RegisterActivity(activities.ActivityEvaluateCondition)
    w.RegisterActivity(activities.ActivitySaveFormData)
    
    // Start worker
    return w.Run(worker.InterruptCh())
}
```

### 2.4: Compile and Test

```bash
# Compile backend
cd /Users/eganpj/GitHub/semlayer/backend
go build -o semlayer-backend ./cmd/main.go

# Expected: No errors, executable created

# If compilation fails:
# - Check import paths match exactly
# - Verify all dependencies installed: go mod download
# - Check for syntax errors in edited files
```

---

## 🎨 Phase 3: Frontend Integration (5 minutes)

### 3.1: Copy React Component

```bash
# Verify file exists
ls -l /Users/eganpj/GitHub/semlayer/frontend/src/pages/BusinessProcessListPage.tsx

# File should be 400+ lines, 0 compilation errors
wc -l /Users/eganpj/GitHub/semlayer/frontend/src/pages/BusinessProcessListPage.tsx
```

### 3.2: Add Route

Edit your frontend router configuration (typically `src/App.tsx` or `src/Router.tsx`):

```typescript
import React from 'react';
import { BrowserRouter, Routes, Route } from 'react-router-dom';

// ✅ ADD THIS IMPORT:
import BusinessProcessList from '@/pages/BusinessProcessListPage';

// Import other pages
import Dashboard from '@/pages/Dashboard';
// ... other imports

function AppRouter() {
  return (
    <BrowserRouter>
      <Routes>
        {/* Existing routes */}
        <Route path="/dashboard" element={<Dashboard />} />
        
        {/* ✅ ADD THIS ROUTE: */}
        <Route path="/processes" element={<BusinessProcessList />} />
        
        {/* ... other routes */}
      </Routes>
    </BrowserRouter>
  );
}

export default AppRouter;
```

### 3.3: Add Navigation Link

Edit your navigation component (typically `src/components/Navigation.tsx`):

```typescript
export function Navigation() {
  return (
    <nav>
      <ul>
        {/* Existing links */}
        <li><Link to="/dashboard">Dashboard</Link></li>
        
        {/* ✅ ADD THIS LINK: */}
        <li><Link to="/processes">Business Processes</Link></li>
        
        {/* ... other links */}
      </ul>
    </nav>
  );
}
```

### 3.4: Compile Frontend

```bash
# Compile frontend
cd /Users/eganpj/GitHub/semlayer/frontend
npm run build

# Expected: No errors, build artifacts in dist/

# If compilation fails:
# - Check TypeScript syntax: npm run type-check
# - Check imports are correct
# - Verify all dependencies installed: npm install
```

---

## 🧪 Phase 4: Testing (8 minutes)

### Test 1: Database Schema

```bash
psql -U postgres -d alpha -c "
SELECT COUNT(*) as table_count FROM information_schema.tables 
WHERE table_schema = 'public' AND table_name LIKE 'bp_%';
"

# Expected output: table_count = 8
```

### Test 2: API Endpoint - Save BP

```bash
TENANT_ID="00000000-0000-0000-0000-000000000000"
DATASOURCE_ID="11111111-1111-1111-1111-111111111111"

curl -X POST http://localhost:8080/api/bp/save \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "X-Tenant-Datasource-ID: $DATASOURCE_ID" \
  -d '{
    "processName": "Test Hire Employee",
    "description": "End-to-end hiring process",
    "entity": "Employee",
    "status": "draft",
    "isActive": false,
    "steps": [
      {
        "stepOrder": 1,
        "stepType": "data_entry",
        "stepName": "Collect Application",
        "durationHours": 24,
        "description": "Gather application from candidate"
      },
      {
        "stepOrder": 2,
        "stepType": "validate",
        "stepName": "Validate Data",
        "durationHours": 1,
        "description": "Verify required fields"
      },
      {
        "stepOrder": 3,
        "stepType": "approve",
        "stepName": "HR Review",
        "durationHours": 48,
        "description": "HR team approval"
      }
    ]
  }'

# Expected response (201 Created):
# {
#   "id": "550e8400-e29b-41d4-a716-446655440000",
#   "processName": "Test Hire Employee",
#   "status": "draft",
#   "versionNumber": 1,
#   "totalSteps": 3,
#   "totalDurationHours": 73,
#   "message": "Business process saved successfully"
# }

# Save the process ID for next tests
PROCESS_ID="550e8400-e29b-41d4-a716-446655440000"
```

### Test 3: API Endpoint - List BPs

```bash
curl -X GET "http://localhost:8080/api/bp?offset=0&limit=20" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "X-Tenant-Datasource-ID: $DATASOURCE_ID"

# Expected response (200 OK):
# {
#   "processes": [
#     {
#       "id": "550e8400-e29b-41d4-a716-446655440000",
#       "processName": "Test Hire Employee",
#       "entity": "Employee",
#       "stepCount": 3,
#       "totalDurationHours": 73,
#       "status": "draft",
#       "isActive": false,
#       "createdBy": "your_email@example.com"
#     }
#   ],
#   "total": 1
# }
```

### Test 4: API Endpoint - Get Single BP

```bash
curl -X GET "http://localhost:8080/api/bp/$PROCESS_ID" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "X-Tenant-Datasource-ID: $DATASOURCE_ID"

# Expected response (200 OK):
# Complete BP with all steps, validations, approvers
```

### Test 5: API Endpoint - Simulate BP

```bash
curl -X POST http://localhost:8080/api/bp/simulate \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "X-Tenant-Datasource-ID: $DATASOURCE_ID" \
  -d "{
    \"processId\": \"$PROCESS_ID\",
    \"steps\": []
  }"

# Expected response (200 OK):
# {
#   "estimatedDurationHours": 73,
#   "stepsCount": 3,
#   "validationSteps": 1,
#   "approvalSteps": 1,
#   "notificationSteps": 0,
#   "warnings": [],
#   "status": "ready_to_execute"
# }
```

### Test 6: Frontend - Load Process List

1. Open browser: `http://localhost:3000/processes`
2. Verify:
   - [ ] Page loads without errors
   - [ ] "Test Hire Employee" process appears in table
   - [ ] Search box works (type "Hire")
   - [ ] Status filter works (select "Draft")
   - [ ] Action buttons visible (Edit, Run, Archive)
   - [ ] Pagination info shows "Showing 1 to 1 of 1 processes"

### Test 7: Frontend - Multi-Tenant Scoping

1. Verify localStorage before loading page:
   ```javascript
   // In browser console:
   localStorage.getItem('selected_tenant')
   localStorage.getItem('selected_datasource')
   // Both should return non-null values
   ```

2. Try accessing without tenant scope:
   ```javascript
   localStorage.removeItem('selected_tenant');
   // Reload page - should show warning
   ```

---

## ✅ Post-Deployment Verification

### Health Checks

```bash
# 1. API health
curl http://localhost:8080/api/health
# Expected: 200 OK

# 2. Database connection
curl http://localhost:8080/api/bp \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "X-Tenant-Datasource-ID: $DATASOURCE_ID"
# Expected: 200 OK with processes array

# 3. Temporal workflow availability
curl http://localhost:7233
# Expected: Connection to Temporal

# 4. Frontend loading
curl http://localhost:3000
# Expected: 200 OK with HTML
```

### Audit Trail Verification

```bash
# Verify audit entries were created
psql -U postgres -d alpha -c "
SELECT action_type, COUNT(*) FROM bp_audit_trail 
GROUP BY action_type;
"

# Expected output:
# action_type | count
# ------------|-------
# CREATE      |   1
# (1 row)
```

### Performance Baseline

```bash
# List endpoint response time
time curl -X GET "http://localhost:8080/api/bp?offset=0&limit=20" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "X-Tenant-Datasource-ID: $DATASOURCE_ID" > /dev/null

# Expected: < 100ms total
```

---

## 🔄 Rollback Procedure (Emergency Only)

### If Database Migration Failed

```bash
# Restore from backup
psql -U postgres -d alpha < alpha_backup_YYYYMMDD_HHMMSS.sql

# Verify tables gone
\dt bp_*
# Expected: No results
```

### If API Crashes

```bash
# Remove route registration from main.go
# Comment out: handlers.RegisterBPRoutes(router, db)

# Recompile and restart backend
cd /Users/eganpj/GitHub/semlayer/backend
go build -o semlayer-backend ./cmd/main.go

# Restart service
pkill -f semlayer-backend
./semlayer-backend
```

### If Frontend Issues

```bash
# Remove route from router config
# Comment out: <Route path="/processes" element={<BusinessProcessList />} />

# Rebuild frontend
cd /Users/eganpj/GitHub/semlayer/frontend
npm run build

# Restart dev server or redeploy
npm run dev
```

### Complete Rollback

```bash
# 1. Revert database
psql -U postgres -d alpha < alpha_backup_YYYYMMDD_HHMMSS.sql

# 2. Revert backend code
git checkout backend/main.go
git checkout backend/api/handlers/bp_handler.go
go build -o semlayer-backend ./cmd/main.go

# 3. Revert frontend code
git checkout frontend/src/App.tsx
npm run build

# 4. Restart services
pkill -f semlayer-backend
npm run dev

# Verify system working:
curl http://localhost:8080/api/health
# Expected: 200 OK
```

---

## 📊 Post-Deployment Monitoring

### Logs to Watch

```bash
# Backend logs
tail -f /var/log/semlayer/backend.log
# Watch for errors, API calls, database queries

# Temporal logs
tail -f /var/log/temporal/temporal.log
# Watch for workflow executions

# Database logs
tail -f /var/log/postgresql/postgresql.log
# Watch for slow queries, errors
```

### Metrics to Track

| Metric | Target | Warning |
|--------|--------|---------|
| API Response Time | < 100ms | > 500ms |
| Database Query Time | < 50ms | > 200ms |
| Workflow Execution | < 5 minutes | > 30 minutes |
| Error Rate | < 0.1% | > 1% |
| Uptime | > 99.9% | < 99% |

### Alerts to Configure

- [ ] API endpoint latency > 500ms
- [ ] Database connection errors
- [ ] Workflow execution failures
- [ ] Audit trail gaps
- [ ] Disk space below 10GB

---

## 🎉 Deployment Complete!

### Checklist Summary

- [x] Database schema deployed (8 tables)
- [x] Backend routes registered (5 endpoints)
- [x] Temporal workflow registered
- [x] Frontend route added
- [x] All compilation successful
- [x] API endpoints tested
- [x] Frontend verified
- [x] Audit trail confirmed
- [x] Multi-tenant scoping tested
- [x] Rollback procedure documented

### Success Indicators

✅ **Technical Verification**
- Database: 8 BP tables present with proper indexes
- API: All 5 endpoints responding (201, 200, 200, 200, 200)
- Frontend: List page renders with process from API
- Workflow: Temporal worker registered and ready
- Audit: Entries created for all mutations

✅ **Functional Verification**
- Can create BP via API
- Can list BPs in frontend
- Can filter and search
- Can simulate BP
- Can execute BP (workflow starts)

✅ **Security Verification**
- Multi-tenant scoping enforced
- Headers validated on all requests
- Audit trail complete
- No data leakage in errors

---

## 📞 Support Information

**Questions During Deployment?**
- Check [BP_BUILDER_COMPLETE_INTEGRATION.md](./BP_BUILDER_COMPLETE_INTEGRATION.md) for detailed examples
- Check [BP_BUILDER_BACKEND_VERIFICATION.md](./BP_BUILDER_BACKEND_VERIFICATION.md) for verification details
- Check [BP_BUILDER_QUICK_REFERENCE.md](./BP_BUILDER_QUICK_REFERENCE.md) for API reference

**Deployment Issues?**
- Database: Check `psql` command has correct password/host
- Backend: Check imports and `go mod download`
- Frontend: Check `npm install` completed
- Temporal: Check Temporal server is `started-dev`

**Performance Issues?**
- Check database indexes: `\di bp_*`
- Check API response times: Use curl with `-w "@curl-format.txt"`
- Check database connections: `SELECT * FROM pg_stat_activity`

---

**Deployment Date:** _________________  
**Deployed By:** _________________  
**Verification Sign-Off:** _________________  

✅ **DEPLOYMENT COMPLETE & VERIFIED**
