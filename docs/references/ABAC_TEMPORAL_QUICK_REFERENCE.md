# ABAC + Temporal - Developer Quick Reference

## 🚀 Quick Start (5 minutes)

### 1. Setup Database
```bash
psql postgres://postgres:postgres@localhost:5432/alpha < migrations/006_complete_trigger_system_schema.sql
```

### 2. Start Backend
```bash
cd backend
go build -o semlayer-api ./cmd/main.go
./semlayer-api
```

### 3. Start Temporal
```bash
# Terminal 1: Temporal Server
temporal server start-dev

# Terminal 2: Temporal Worker
cd temporal
npm install @temporalio/worker @temporalio/client @temporalio/workflow @temporalio/activity
ts-node -r tsconfig-paths/register worker.ts
```

### 4. Use in React
```tsx
import { ABACProvider, useABAC, PolicyBuilder } from '@/components/abac';

function App() {
  return (
    <ABACProvider>
      <Dashboard />
    </ABACProvider>
  );
}

function Dashboard() {
  const { evaluate, createPolicy } = useABAC();
  
  const canEdit = await evaluate({
    subject: userId,
    action: 'edit_bundle',
    resource: 'bundles:' + bundleId,
  });
  
  return canEdit.decision === 'allow' ? <Editor /> : <AccessDenied />;
}
```

## 📍 File Locations

```
frontend/src/components/abac/
├── ABACProvider.tsx        # Import this for context + hooks
├── PolicyBuilder.tsx       # Admin policy editor
├── DelegationManager.tsx   # Temp role assignments
├── AuditLogViewer.tsx      # Compliance logs
└── index.ts

backend/internal/api/
└── abac.go                 # 9 API endpoints

temporal/
├── workflows/
│   ├── ClientOnboardingWorkflow.ts
│   └── TimeoutEscalationWorkflow.ts
├── activities/
│   ├── clientOnboardingActivities.ts
│   └── timeoutEscalationActivities.ts
├── worker.ts
└── client.ts
```

## 🔌 API Endpoints (All require X-Tenant-ID + X-Tenant-Datasource-ID headers)

```bash
# Policies
POST   /api/abac/policies              # Create
GET    /api/abac/policies              # List
PUT    /api/abac/policies/:id          # Update
DELETE /api/abac/policies/:id          # Delete

# Evaluation
POST   /api/abac/evaluate              # Check decision (allow/deny)

# Delegations
POST   /api/abac/delegations           # Create
GET    /api/abac/delegations           # List active
DELETE /api/abac/delegations/:id       # Revoke

# Audit
GET    /api/abac/audit                 # List logs (filters: ?days=30&action=create_policy&result=allow)
```

## 📋 Common Tasks

### Create a Policy
```bash
curl -X POST http://localhost:8080/api/abac/policies \
  -H "X-Tenant-ID: $(uuidgen)" \
  -H "X-Tenant-Datasource-ID: $(uuidgen)" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Finance Reports",
    "effect": "allow",
    "priority": 100,
    "enabled": true,
    "subject_rules": {"roles": ["finance"]},
    "action_rules": {"allowed_actions": ["view_reports"]},
    "resource_rules": {"resource_types": ["report"]}
  }'
```

### Evaluate Access
```bash
curl -X POST http://localhost:8080/api/abac/evaluate \
  -H "X-Tenant-ID: tenant-xyz" \
  -H "X-Tenant-Datasource-ID: datasource-abc" \
  -H "Content-Type: application/json" \
  -d '{
    "subject": "user-jane",
    "action": "view_reports",
    "resource": "report:monthly"
  }'
# Response: {"decision":"allow","reason":"Matched policy...","policy_id":"..."}
```

### Create Delegation
```bash
curl -X POST http://localhost:8080/api/abac/delegations \
  -H "X-Tenant-ID: tenant-xyz" \
  -H "X-Tenant-Datasource-ID: datasource-abc" \
  -H "Content-Type: application/json" \
  -d '{
    "from_user_id": "john",
    "to_user_id": "jane",
    "policy_id": "policy-xyz",
    "expires_at": "2024-12-31T23:59:59Z"
  }'
```

### Start Workflow
```bash
curl -X POST http://localhost:8080/api/workflows/client-onboarding \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "acme-001",
    "client_name": "ACME Corp",
    "email": "contact@acme.com",
    "manager_id": "mgr-123"
  }'
# Response: {"workflow_id":"..."}
```

### Approve Workflow
```bash
# Via React component that calls:
const { approveClientOnboarding } = useABAC();
await approveClientOnboarding(workflowId, managerId);

# Or via API (endpoint TBD in your workflow handler)
curl -X POST http://localhost:8080/api/workflows/{id}/approve \
  -H "Content-Type: application/json" \
  -d '{"manager_id": "mgr-123"}'
```

## 🧩 Component Usage

### Use ABAC Provider
```tsx
// app.tsx
import { ABACProvider } from '@/components/abac';

export default function App() {
  return (
    <ABACProvider>
      <YourApp />
    </ABACProvider>
  );
}
```

### Use ABAC Hook
```tsx
import { useABAC } from '@/components/abac';

function MyComponent() {
  const { evaluate, canExecute, listPolicies } = useABAC();
  
  // Evaluate access
  const decision = await evaluate({
    subject: 'user-123',
    action: 'edit',
    resource: 'bundle:456'
  });
  
  // Check permission shorthand
  const canEdit = await canExecute('edit_policy', 'abac:policies');
  
  // Get all policies
  const { data: policies } = useQuery(
    ['policies'],
    () => listPolicies()
  );
}
```

### Use Components
```tsx
import {
  PolicyBuilder,
  DelegationManager,
  AuditLogViewer
} from '@/components/abac';

function AdminDashboard() {
  return (
    <div>
      <PolicyBuilder />
      <DelegationManager />
      <AuditLogViewer 
        filterByDays={30}
        onExport={handleExport}
      />
    </div>
  );
}
```

## 🔐 Tenant Headers (Mandatory)

Every API call needs:
```
X-Tenant-ID: <tenant-uuid>
X-Tenant-Datasource-ID: <datasource-uuid>
```

**In React Components**: Automatically added by fetch shim (setupTenantFetch.ts)
**In Curl/Scripts**: Add manually
**In Backend Routes**: Check headers in handlers

## 📊 Database Tables

### abac_policies
```sql
id | tenant_id | datasource_id | name | effect | priority | enabled
subject_rules (JSONB) | action_rules (JSONB) | resource_rules (JSONB)
environment_rules (JSONB) | created_by | created_at | updated_at
```

### abac_delegations
```sql
id | tenant_id | datasource_id | from_user_id | to_user_id
policy_id | expires_at | created_at
```

### audit_log
```sql
id | tenant_id | datasource_id | actor | action | resource
decision | reason | ip_address | timestamp
```

## 🛠️ Backend Integration

### Register Routes
```go
// In main.go or api.go
import "github.com/semlayer/backend/internal/api"

func main() {
  router := gin.Default()
  db := setupDatabase()
  
  // Register ABAC routes
  httpapi.RegisterABACRoutes(router, db)
  
  router.Run(":8080")
}
```

### Use in Handlers
```go
// Evaluate policy before executing
decision, err := abacAPI.evaluatePolicy(ABACEvaluationRequest{
  Subject: currentUser,
  Action: "execute_trigger",
  Resource: "trigger:" + triggerID,
})

if decision == "deny" {
  return fmt.Errorf("Access denied: %s", decision.Reason)
}

// Proceed with execution
```

## 🚦 Workflow Status

### Query Workflow Status
```bash
curl http://localhost:8080/api/workflows/{id}/status \
  -H "X-Tenant-ID: tenant-xyz" \
  -H "X-Tenant-Datasource-ID: datasource-abc"
```

### Response Structure
```json
{
  "status": "running",
  "step": "route_for_approval",
  "approval_details": {
    "manager_id": "mgr-123",
    "started_at": "2024-01-15T10:30:00Z"
  }
}
```

## 🔄 Workflow Signals

### Approve Onboarding
```tsx
const { approveClientOnboarding } = useABAC();
await approveClientOnboarding(workflowId, managerId);
```

### Reject Onboarding
```tsx
const { rejectClientOnboarding } = useABAC();
await rejectClientOnboarding(workflowId, managerId, 'Applicant not eligible');
```

## 📍 Error Handling

### Policy Evaluation Errors
```json
{
  "decision": "deny",
  "reason": "No matching policy",
  "policy_id": null
}
```

### API Errors
```
Missing tenant scope → 400 Bad Request
Policy not found → 404 Not Found
Database error → 500 Internal Server Error
```

### Workflow Errors
- Automatic retry with backoff
- Escalate to director on 3 failures
- All attempts logged to audit_log

## 🧪 Testing

### Test Policy Creation
```bash
# Should return 201 with policy ID
curl -X POST http://localhost:8080/api/abac/policies \
  -H "X-Tenant-ID: test-tenant" \
  -H "X-Tenant-Datasource-ID: test-datasource" \
  -H "Content-Type: application/json" \
  -d '{"name":"test","effect":"allow","priority":100,"enabled":true}' \
  | grep -q '"id"' && echo "✓ PASS" || echo "✗ FAIL"
```

### Test List Policies
```bash
# Should return policies array
curl http://localhost:8080/api/abac/policies \
  -H "X-Tenant-ID: test-tenant" \
  -H "X-Tenant-Datasource-ID: test-datasource" \
  | grep -q '"policies"' && echo "✓ PASS" || echo "✗ FAIL"
```

### Test Evaluation
```bash
# Should return decision
curl -X POST http://localhost:8080/api/abac/evaluate \
  -H "X-Tenant-ID: test-tenant" \
  -H "X-Tenant-Datasource-ID: test-datasource" \
  -H "Content-Type: application/json" \
  -d '{"subject":"user","action":"test","resource":"test"}' \
  | grep -q '"decision"' && echo "✓ PASS" || echo "✗ FAIL"
```

## 📚 Documentation Reference

| Document | Purpose |
|----------|---------|
| **ABAC_TEMPORAL_SYSTEM_INDEX.md** | Architecture + file structure |
| **ABAC_TEMPORAL_INTEGRATION_GUIDE.md** | Integration patterns + examples |
| **ABAC_TEMPORAL_DEPLOYMENT_CHECKLIST.md** | Pre-deployment + validation |
| **agents.md** | Tenant scoping rules (MUST READ) |
| **WORKDAY_TRIGGER_DEPLOYMENT_GUIDE.md** | Existing trigger system |

## 🆘 Troubleshooting

| Issue | Solution |
|-------|----------|
| "Missing tenant scope" | Add X-Tenant-ID + X-Tenant-Datasource-ID headers |
| React Query errors | Update to @tanstack/react-query v4+ (not v3) |
| Workflow not executing | Verify Temporal Worker is running |
| Policy always returns deny | Create policy with matching subject/action/resource |
| Cannot find module | SDK imports resolve at runtime (ok during dev) |

## 🎯 Common Patterns

### Permission Check in Component
```tsx
const { canExecute } = useABAC();

if (!(await canExecute('edit_policy', 'abac'))) {
  return <AccessDenied />;
}
```

### Create and Use Policy
```tsx
const { createPolicy, evaluate } = useABAC();

// 1. Admin creates policy
await createPolicy({
  name: 'View Reports',
  effect: 'allow',
  subject_rules: { roles: ['finance'] },
  action_rules: { allowed_actions: ['view_reports'] }
});

// 2. User's component evaluates
const can = await evaluate({
  subject: userId,
  action: 'view_reports',
  resource: 'report:monthly'
});

if (can.decision === 'allow') {
  return <ReportsViewer />;
}
```

### Audit Compliance
```tsx
<AuditLogViewer
  filterByDays={30}
  filterByAction="create_policy"
  onExport={(csv) => {
    const blob = new Blob([csv], { type: 'text/csv' });
    downloadBlob(blob, 'audit-log.csv');
  }}
/>
```

---

**Pro Tip**: Keep this file open while developing. All common tasks are here. 🚀

**Need More Help?** See the full guides in ABAC_TEMPORAL_SYSTEM_INDEX.md
