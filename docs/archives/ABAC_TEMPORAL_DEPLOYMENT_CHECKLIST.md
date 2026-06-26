# ABAC + Temporal Deployment Checklist

Complete pre-deployment verification steps for ABAC and Temporal workflow system.

## ✅ Pre-Deployment Requirements

### Database Setup
- [ ] Verify `abac_policies` table exists in PostgreSQL
- [ ] Verify `abac_delegations` table exists
- [ ] Verify `audit_log` table exists and is scoped by `tenant_id` + `datasource_id`
- [ ] Verify `step_timeouts` table has timeout configuration
- [ ] Run all migrations: `psql -f migrations/006_complete_trigger_system_schema.sql`

### Backend Setup
- [ ] Go backend compiles without errors: `go build ./backend/...`
- [ ] ABAC handlers registered in main.go with `httpapi.RegisterABACRoutes(router, db)`
- [ ] PostgreSQL connection working with DSN in config.yaml
- [ ] Test trigger routes still functional

### Frontend Setup
- [ ] React 18+ installed and compiling: `npm run build`
- [ ] @tanstack/react-query v4+ available (not v3)
- [ ] Ant Design v5+ available
- [ ] `setupTenantFetch.ts` configured to patch fetch with tenant headers

### Temporal Setup
- [ ] Temporal Server running locally: `temporal server start-dev` (port 7233)
- [ ] Temporal CLI available: `temporal --version`
- [ ] Node.js 18+ available for TypeScript workflows
- [ ] Dependencies installable in temporal/: `npm install` successful

## 🔍 Component Verification

### React Components
- [ ] `frontend/src/components/abac/ABACProvider.tsx` - compiles without errors
- [ ] `frontend/src/components/abac/PolicyBuilder.tsx` - compiles without errors
- [ ] `frontend/src/components/abac/DelegationManager.tsx` - compiles without errors
- [ ] `frontend/src/components/abac/AuditLogViewer.tsx` - compiles without errors
- [ ] `frontend/src/components/abac/index.ts` - exports all components correctly

### Temporal Workflows
- [ ] `temporal/workflows/ClientOnboardingWorkflow.ts` - structural syntax valid
- [ ] `temporal/workflows/TimeoutEscalationWorkflow.ts` - structural syntax valid
- [ ] Workflow signal/query handlers defined
- [ ] Workflow error handling implements escalation

### Temporal Activities
- [ ] `temporal/activities/clientOnboardingActivities.ts` - all 7 activities defined
- [ ] `temporal/activities/timeoutEscalationActivities.ts` - all 5 activities defined
- [ ] Activities make HTTP calls with proper error handling
- [ ] Activity retry configurations match workflow expectations

### Temporal Integration
- [ ] `temporal/worker.ts` - worker creation and startup logic complete
- [ ] `temporal/client.ts` - client initialization and workflow starters implemented
- [ ] Graceful shutdown handlers configured

### Backend Handlers
- [ ] `backend/internal/api/abac.go` - all 9 endpoint handlers implemented
- [ ] Tenant scope enforcement on every endpoint (X-Tenant-ID + X-Tenant-Datasource-ID)
- [ ] Audit logging on policy create/update/delete/evaluate
- [ ] Database queries properly parameterized to prevent SQL injection

## 🧪 Functional Testing

### Unit Tests
- [ ] Run React component tests: `npm test` (if tests exist)
- [ ] Run Go handler tests: `go test ./backend/internal/api/...`
- [ ] Verify zero test failures

### Integration Tests - Backend
- [ ] POST `/api/abac/policies` with tenant headers → 201 Created
- [ ] GET `/api/abac/policies` → 200 OK with policies list
- [ ] PUT `/api/abac/policies/{id}` → 200 OK with updated policy
- [ ] DELETE `/api/abac/policies/{id}` → 200 OK with deleted policy
- [ ] POST `/api/abac/evaluate` → 200 OK with decision (allow/deny)
- [ ] POST `/api/abac/delegations` → 201 Created
- [ ] GET `/api/abac/delegations` → 200 OK with active delegations
- [ ] DELETE `/api/abac/delegations/{id}` → 200 OK with revoked delegation
- [ ] GET `/api/abac/audit` → 200 OK with filtered logs

### Integration Tests - Tenant Isolation
- [ ] Create policy with tenant A
- [ ] Attempt to read with tenant B → 404 or empty list
- [ ] Missing X-Tenant-ID header → 400 Bad Request
- [ ] Missing X-Tenant-Datasource-ID header → 400 Bad Request

### Integration Tests - Frontend
- [ ] ABACProvider renders without errors
- [ ] useABAC hook returns expected methods
- [ ] PolicyBuilder form submits with tenant headers
- [ ] DelegationManager loads delegations for tenant
- [ ] AuditLogViewer filters and exports CSV
- [ ] Tenant context properly reads localStorage

### Integration Tests - Workflows
- [ ] Start ClientOnboardingWorkflow → returns workflow ID
- [ ] Query workflow status → returns step + details
- [ ] Send approval signal → workflow processes it
- [ ] Workflow completes all 6 steps successfully
- [ ] Start TimeoutEscalationWorkflow → triggers escalation action
- [ ] Timeout escalation logs event to audit trail

### End-to-End Test Scenario
1. [ ] Tenant A creates admin user + finance user
2. [ ] Admin creates policy: "Finance can view reports"
3. [ ] Finance user logs in and sees available policies
4. [ ] Admin delegates finance policy to temp user (1-week expiry)
5. [ ] Temp user can execute delegated actions
6. [ ] Audit log shows all decisions and delegations
7. [ ] Admin starts client onboarding workflow
8. [ ] Manager approves via signal
9. [ ] Workflow completes → final notification sent
10. [ ] All steps logged to audit trail

## 🔐 Security Verification

- [ ] All DB queries use parameterized statements (no string concatenation)
- [ ] Tenant scope checked before returning any data
- [ ] User ID from auth middleware used for audit logging
- [ ] IP address captured for audit trail
- [ ] Passwords/secrets not logged to audit trail
- [ ] Temporal connections use TLS if over network (dev: localhost OK)
- [ ] Frontend fetch shim validates tenant selection before making requests

## 📊 Performance Baseline

- [ ] Policy evaluation completes in < 100ms
- [ ] List policies (1000+) completes in < 500ms
- [ ] Audit log export (10000+ rows) completes in < 2s
- [ ] Workflow signal processing completes in < 1s
- [ ] Database connections pool configured (min 5, max 20)

## 📋 Deployment Steps

### 1. Database Migration
```bash
cd /Users/eganpj/GitHub/semlayer
psql -f migrations/006_complete_trigger_system_schema.sql
# Verify no errors
psql -c "\dt public.abac*"  # Should show 2 tables
psql -c "\dt public.audit*"  # Should show audit_log table
```

### 2. Backend Deployment
```bash
cd backend
go build -o semlayer-api ./cmd/main.go
# Test locally
./semlayer-api &
curl -H "X-Tenant-ID: test" -H "X-Tenant-Datasource-ID: test" \
  http://localhost:8080/api/abac/policies
kill %1
```

### 3. Temporal Setup
```bash
# Option A: Local dev server (for testing)
temporal server start-dev &

# Option B: Docker (production-like)
docker run --rm -d \
  -p 7233:7233 -p 8233:8233 \
  -e DB=sqlite \
  -e SQLITE_FILENAME=/var/lib/temporal/sqlite.db \
  --name temporal \
  temporalio/auto-setup:latest

# Wait 5 seconds for startup
sleep 5
curl -s http://localhost:8233/api/v1/namespaces | grep temporal
```

### 4. Temporal Worker Startup
```bash
cd temporal
npm install
# Create .env with TEMPORAL_SERVER_ADDRESS=localhost:7233
ts-node -r tsconfig-paths/register worker.ts
# Verify: "Temporal Worker created and configured" in logs
# Keep running (background process or separate terminal)
```

### 5. Frontend Deployment
```bash
cd frontend
npm run build
# Verify dist/ folder created with no errors
# Deploy to your hosting (Vercel, Netlify, S3, etc.)
```

### 6. Smoke Tests
```bash
# Test 1: Create policy
curl -X POST http://localhost:8080/api/abac/policies \
  -H "X-Tenant-ID: smoke-test" \
  -H "X-Tenant-Datasource-ID: smoke-test" \
  -H "Content-Type: application/json" \
  -d '{"name":"test","effect":"allow","priority":100,"enabled":true}' \
  | grep -q '"id"' && echo "✓ Create policy" || echo "✗ Create policy"

# Test 2: List policies
curl http://localhost:8080/api/abac/policies \
  -H "X-Tenant-ID: smoke-test" \
  -H "X-Tenant-Datasource-ID: smoke-test" \
  | grep -q '"policies"' && echo "✓ List policies" || echo "✗ List policies"

# Test 3: Evaluate policy
curl -X POST http://localhost:8080/api/abac/evaluate \
  -H "X-Tenant-ID: smoke-test" \
  -H "X-Tenant-Datasource-ID: smoke-test" \
  -H "Content-Type: application/json" \
  -d '{"subject":"user","action":"test","resource":"test"}' \
  | grep -q '"decision"' && echo "✓ Evaluate policy" || echo "✗ Evaluate policy"

# Test 4: Start workflow
curl -X POST http://localhost:8080/api/workflows/client-onboarding \
  -H "Content-Type: application/json" \
  -d '{"client_id":"test","client_name":"Test","email":"test@test.com","manager_id":"mgr1"}' \
  | grep -q '"workflow_id"' && echo "✓ Start workflow" || echo "✗ Start workflow"
```

## 🎯 Post-Deployment Validation

- [ ] Smoke tests all passing (4/4 green)
- [ ] Frontend loads ABACProvider without console errors
- [ ] React console shows no "Cannot find module" warnings
- [ ] Backend logs show "ABAC routes registered"
- [ ] Temporal worker logs show "Temporal Worker created and configured"
- [ ] Database contains test policies from smoke tests
- [ ] Audit log entries recorded for test operations
- [ ] Admin can access policy builder in UI
- [ ] Admin can see audit logs in UI
- [ ] Users without permission get "Access Denied"

## 🚀 Production Readiness

- [ ] All checklist items above completed
- [ ] Load testing shows < 500ms p95 latency for API calls
- [ ] Database backups configured (daily minimum)
- [ ] Monitoring/alerting configured for workflow failures
- [ ] Documentation reviewed and teams trained
- [ ] Rollback plan documented (database snapshots, code tags)
- [ ] Disaster recovery tested (restore from backup)

## 📞 Support Contacts

- **ABAC Questions**: See ABAC_TEMPORAL_INTEGRATION_GUIDE.md
- **Temporal Workflows**: See TEMPORAL_WORKFLOWS_GUIDE.md (if created)
- **Deployment Issues**: Check backend logs: `docker logs semlayer-api`
- **Database Issues**: Verify migrations: `psql -c "SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 5;"`

---

**Status**: Ready for deployment ✓
**Last Updated**: 2024
**Version**: 1.0
