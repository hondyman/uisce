# Audit Explorer - Deployment & Integration Checklist

## Pre-Integration Verification

- [ ] All 6 backend Go files created without errors
  - [ ] `explorer_models.go` (240 lines)
  - [ ] `explorer_repository.go` (380 lines)
  - [ ] `explorer_service.go` (180 lines)
  - [ ] `explorer_handler.go` (340 lines)
  - [ ] `explorer_rbac.go` (310 lines)
  - [ ] `trino_queries.go` (450 lines)

- [ ] All 7 frontend React components created without errors
  - [ ] `AuditExplorer.tsx` (main container)
  - [ ] `FilterBar.tsx` (filters)
  - [ ] `TimelineView.tsx` (timeline tab)
  - [ ] `EntitiesView.tsx` (entities tab)
  - [ ] `IncidentsView.tsx` (incidents tab)
  - [ ] `ComplianceView.tsx` (compliance tab)
  - [ ] `AIPanel.tsx` (AI explanations)

- [ ] Custom hook created
  - [ ] `useAuditExplorer.ts`

- [ ] Documentation complete
  - [ ] `AUDIT_EXPLORER_GUIDE.md` (full guide)
  - [ ] `AUDIT_EXPLORER_SUMMARY.md` (summary)
  - [ ] `AUDIT_EXPLORER_QUICK_INTEGRATION.md` (integration steps)

## Backend Integration Steps

### Step 1: File Placement
- [ ] Copy `explorer_models.go` to `/backend/internal/audit/`
- [ ] Copy `explorer_repository.go` to `/backend/internal/audit/`
- [ ] Copy `explorer_service.go` to `/backend/internal/audit/`
- [ ] Copy `explorer_handler.go` to `/backend/internal/audit/`
- [ ] Copy `explorer_rbac.go` to `/backend/internal/audit/`
- [ ] Copy `trino_queries.go` to `/backend/internal/audit/`

### Step 2: API Integration
- [ ] Update `/backend/internal/api/api.go`:
  - [ ] Add imports: `audit`, `auth`
  - [ ] Add `aiClient` field to `APIServer` struct
  - [ ] Add `registerAuditExplorerRoutes()` method
  - [ ] Call `a.registerAuditExplorerRoutes(r)` in `setupRoutes()`

### Step 3: AI Client Setup
- [ ] Choose AI vendor (Anthropic, OpenAI, or custom)
- [ ] Install AI client library:
  - [ ] `go get github.com/anthropics/...` (if Anthropic)
  - [ ] `go get github.com/sashabaranov/go-openai` (if OpenAI)
- [ ] Implement `audit.AIClient` interface in `/backend/internal/audit/ai_client.go`
- [ ] Configure API key in environment or `.env` file

### Step 4: Trino Connection
- [ ] Verify Trino driver registered: `import _ "github.com/trinodb/trino-go-client/trino"`
- [ ] Verify database connection URL format: `http://host:8080/default/iceberg?user=root`
- [ ] Test Trino connection:
  ```go
  rows, err := db.Query("SELECT 1")
  if err != nil {
      log.Fatalf("Trino connection failed: %v", err)
  }
  ```

### Step 5: Database Tables
- [ ] Verify these tables exist in `iceberg.audit` schema:
  - [ ] `scheduler_job_runs`
  - [ ] `scheduler_dag_runs`
  - [ ] `governance_changesets`
  - [ ] `semantic_snapshots`
  - [ ] `compliance_violations`
  - [ ] `orchestration_events` (optional)
  
- [ ] Verify table structure matches expected columns:
  - [ ] `tenant_id` (partition key)
  - [ ] `date` or `timestamp` (partition key)
  - [ ] `id`, `type`, `status`, `timestamp`, `actor`

### Step 6: Build Verification
- [ ] Run `go mod tidy`
- [ ] Run `go build ./backend/...` (no errors)
- [ ] Run `go test ./backend/internal/audit/...` (all pass)

## Frontend Integration Steps

### Step 1: File Placement
- [ ] Copy `AuditExplorer.tsx` to `/frontend/src/components/audit/`
- [ ] Copy `FilterBar.tsx` to `/frontend/src/components/audit/`
- [ ] Copy `TimelineView.tsx` to `/frontend/src/components/audit/tabs/`
- [ ] Copy `EntitiesView.tsx` to `/frontend/src/components/audit/tabs/`
- [ ] Copy `IncidentsView.tsx` to `/frontend/src/components/audit/tabs/`
- [ ] Copy `ComplianceView.tsx` to `/frontend/src/components/audit/tabs/`
- [ ] Copy `AIPanel.tsx` to `/frontend/src/components/audit/panels/`
- [ ] Copy `useAuditExplorer.ts` to `/frontend/src/hooks/`

### Step 2: Routing
- [ ] Update `/frontend/src/App.tsx` or router config:
  ```tsx
  import { lazy } from 'react';
  const AuditExplorer = lazy(() => import('@/components/audit/AuditExplorer'));
  
  <Routes>
    {/* ... existing routes ... */}
    <Route path="/audit-explorer" element={<AuditExplorer />} />
  </Routes>
  ```

### Step 3: Navigation
- [ ] Update `/frontend/src/components/MainNavigation.tsx`:
  ```tsx
  {hasRole('global_admin', 'global_ops', 'tenant_admin', 'tenant_ops') && (
    <NavLink to="/audit-explorer" icon={<HistoryIcon />} label="Audit Explorer" />
  )}
  ```

### Step 4: Build Verification
- [ ] Run `npm install` (if MUI dependencies needed)
- [ ] Run `npm run build` (no TypeScript errors)
- [ ] Run `npm run lint` (no linting errors)

## Testing & Validation

### Unit Tests (Backend)
- [ ] Create `/backend/internal/audit/explorer_service_test.go`:
  ```go
  func TestExplorerServiceRoleEnforcement(t *testing.T) {
      // Verify tenant scope validation
  }
  ```
- [ ] Run tests: `go test -v ./backend/internal/audit/...`

### Integration Tests (API)
- [ ] Test endpoint with valid tenant:
  ```bash
  curl -X POST http://localhost:8080/api/audit-explorer/events \
    -H "X-Tenant-ID: tenant-001" \
    -H "Authorization: Bearer <token>" \
    -d '{...}'
  ```
- [ ] Verify response contains tenant-001 data only

- [ ] Test endpoint with unauthorized tenant:
  ```bash
  curl -X POST http://localhost:8080/api/audit-explorer/events \
    -H "X-Tenant-ID: tenant-002" \
    -d '{...}'
  ```
- [ ] Verify 403 Forbidden response

### Frontend Tests
- [ ] Verify component renders without errors:
  ```bash
  npm test -- components/audit/AuditExplorer
  ```

### Multi-Role Testing

**As Global Admin:**
- [ ] Can see Timeline tab ✓
- [ ] Can see Entities tab ✓
- [ ] Can see Incidents tab ✓
- [ ] Can see Compliance tab ✓
- [ ] Can view all tenants
- [ ] AI explains with cross-tenant context

**As Global Ops:**
- [ ] Can see Timeline tab ✓
- [ ] Can see Entities tab ✓
- [ ] Can see Incidents tab ✓
- [ ] Can see Compliance tab ✓
- [ ] Limited to assigned tenants
- [ ] AI explains within assigned scope

**As Tenant Admin:**
- [ ] Can see Timeline tab ✓
- [ ] Can see Entities tab ✓
- [ ] Can see Incidents tab ✓
- [ ] Can see Compliance tab ✓
- [ ] Limited to single tenant
- [ ] AI explains for single tenant

**As Tenant Ops:**
- [ ] Can see Timeline tab ✓
- [ ] Cannot see Entities tab ✓
- [ ] Can see Incidents tab ✓
- [ ] Cannot see Compliance tab ✓
- [ ] Limited to single tenant
- [ ] Limited data visibility

### Functional Testing

- [ ] **Timeline View**
  - [ ] Shows events from all 5 sources
  - [ ] Filters by time range work
  - [ ] Filters by artifact type work
  - [ ] Filters by status work
  - [ ] Filters by risk level work
  - [ ] Search functionality works
  - [ ] Expandable rows show details
  - [ ] "Explain" button triggers AI panel

- [ ] **Entities View**
  - [ ] Can search for entities
  - [ ] Shows entity audit trail
  - [ ] Displays change count, failures, compliance issues
  - [ ] Risk score calculated correctly

- [ ] **Incidents View**
  - [ ] Shows incident clusters
  - [ ] AI root cause analysis present
  - [ ] Blast radius displayed
  - [ ] SLO impact shown
  - [ ] Expandable rows work

- [ ] **Compliance View**
  - [ ] Shows compliance violations
  - [ ] Filters by violation type work
  - [ ] Severity levels color-coded
  - [ ] Remediation path shown
  - [ ] Status tracking works

- [ ] **AI Panel**
  - [ ] Shows insights when event selected
  - [ ] Displays root cause analysis
  - [ ] Lists affected systems
  - [ ] Provides recommendations
  - [ ] Shows risk assessment
  - [ ] Correlates related events

### Performance Testing

- [ ] Timeline loads in < 2 seconds with 50 events
- [ ] Entity audit loads in < 3 seconds
- [ ] Incident list loads in < 2 seconds
- [ ] AI explanation completes in < 5 seconds
- [ ] No memory leaks with extended use
- [ ] Pagination works (limit/offset)

### Security Testing

- [ ] Cannot access other tenant's data
- [ ] Cannot bypass role-based restrictions
- [ ] Cannot view Compliance tab as tenant_ops
- [ ] Cannot view Entities tab as tenant_ops
- [ ] AI explanations respect tenant scope
- [ ] API validates tenant scope in headers

## Production Deployment

### Pre-Deployment
- [ ] All tests passing
- [ ] All components build without errors
- [ ] Documentation reviewed and complete
- [ ] AI client configured and tested
- [ ] Trino tables verified and indexed
- [ ] Database backups taken

### Staging Deployment
- [ ] Deploy to staging environment
- [ ] Run smoke tests against staging
- [ ] Verify all 4 roles work correctly
- [ ] Test with actual multi-tenant data
- [ ] Verify performance with production-like data volume
- [ ] Test AI explanations with real events

### Production Deployment
- [ ] Code review completed
- [ ] Security review completed
- [ ] Performance approved
- [ ] Monitoring configured (logs, metrics)
- [ ] Rollback plan documented
- [ ] On-call rotation notified

- [ ] Deploy to production
- [ ] Monitor error rates in first hour
- [ ] Verify endpoints responding
- [ ] Verify all roles can access
- [ ] Verify no tenant data leakage
- [ ] Monitor performance metrics

### Post-Deployment
- [ ] All health checks passing
- [ ] No error spike in logs
- [ ] User feedback collected
- [ ] Documentation updated if needed
- [ ] Team trained on new feature

## Rollback Plan (If Needed)

- [ ] Revert API routes registration in api.go
- [ ] Revert MainNavigation changes
- [ ] Revert router configuration
- [ ] Restart backend service
- [ ] Clear frontend cache
- [ ] Verify previous state

## Documentation for Operations

- [ ] Create runbook: "Audit Explorer Monitoring"
  - [ ] Key metrics to monitor
  - [ ] Common issues and fixes
  - [ ] Escalation contacts

- [ ] Create runbook: "Audit Explorer Troubleshooting"
  - [ ] "No data appears" troubleshooting
  - [ ] "AI explanations slow" troubleshooting
  - [ ] "Permission denied" troubleshooting

- [ ] Add to wiki/knowledge base:
  - [ ] Feature overview for users
  - [ ] Role-based access guide
  - [ ] Common use cases and patterns

## Sign-Off

- [ ] Developer: _________________ Date: _________
- [ ] QA: _________________ Date: _________
- [ ] Product Manager: _________________ Date: _________
- [ ] Engineering Manager: _________________ Date: _________

---

## Quick Verification Checklist

Before declaring done, verify:

1. **Build Status**
   ```bash
   go build ./... && npm run build
   ```
   Result: ✓ No errors

2. **Test Status**
   ```bash
   go test ./... && npm test
   ```
   Result: ✓ All passing

3. **API Responding**
   ```bash
   curl http://localhost:8080/api/audit-explorer/events
   ```
   Result: ✓ 200 OK (or 401 for auth)

4. **Frontend Rendering**
   - Navigate to `/audit-explorer`
   - Result: ✓ Page loads, tabs visible

5. **Role Enforcement**
   - Log in as different roles
   - Result: ✓ Tab visibility changes appropriately

---

**Expected Completion Time:** 6-8 hours total
- Backend integration: 1-2 hours
- Frontend integration: 1-2 hours
- Testing: 2-3 hours
- Troubleshooting/fixes: 1-2 hours
