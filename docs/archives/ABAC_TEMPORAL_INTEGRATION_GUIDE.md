# ABAC + Temporal Integration Guide

Complete integration of ABAC (Attribute-Based Access Control) and Temporal workflows with your existing 13 Workday trigger system.

## 📋 Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                     React Frontend                           │
│  ┌──────────────┬─────────────┬─────────────┬──────────────┐ │
│  │ ABACProvider │ PolicyBuilder│Delegation   │ AuditLogView │ │
│  │              │              │ Manager     │              │ │
│  └──────────────┴─────────────┴─────────────┴──────────────┘ │
└─────────────────────────────────────────────────────────────┘
           ↓ HTTP (X-Tenant-ID headers)
┌─────────────────────────────────────────────────────────────┐
│                  Go Backend (Gin)                            │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  ABAC Handlers (/api/abac/*)                        │   │
│  │  ├─ POST   /policies         (create)               │   │
│  │  ├─ GET    /policies         (list)                 │   │
│  │  ├─ PUT    /policies/:id     (update)               │   │
│  │  ├─ DELETE /policies/:id     (delete)               │   │
│  │  ├─ POST   /evaluate         (evaluate decision)    │   │
│  │  ├─ POST   /delegations      (create delegation)    │   │
│  │  ├─ GET    /delegations      (list delegations)     │   │
│  │  ├─ DELETE /delegations/:id  (revoke delegation)    │   │
│  │  └─ GET    /audit            (list audit logs)      │   │
│  └──────────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  Temporal Client Integration                        │   │
│  │  ├─ startClientOnboardingWorkflow()                 │   │
│  │  ├─ approveClientOnboarding()                       │   │
│  │  ├─ startTimeoutEscalationWorkflow()                │   │
│  │  └─ getWorkflowStatus()                             │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
           ↓ Database writes
┌─────────────────────────────────────────────────────────────┐
│                  PostgreSQL                                  │
│  ├─ abac_policies       (policy definitions)                │
│  ├─ abac_delegations    (temporary role grants)             │
│  ├─ audit_log           (all decisions logged)              │
│  └─ step_timeouts       (timeout escalation tracking)       │
└─────────────────────────────────────────────────────────────┘
           ↓ Event-driven
┌─────────────────────────────────────────────────────────────┐
│                  Temporal Server                            │
│  ├─ ClientOnboardingWorkflow (6-step orchestration)         │
│  ├─ TimeoutEscalationWorkflow (SLA enforcement)             │
│  └─ Activities (calling backend APIs)                       │
└─────────────────────────────────────────────────────────────┘
```

## 🔧 Quick Start - Backend Integration

### 1. Register ABAC Routes in api.go

```go
// In backend/internal/api/api.go or main.go

import "github.com/semlayer/backend/internal/api"

func main() {
  // ... existing setup ...
  
  router := gin.Default()
  db := setupPostgresConnection()
  
  // Register all ABAC routes
  httpapi.RegisterABACRoutes(router, db)
  
  // Keep existing trigger routes
  httpapi.RegisterBundleRoutes(router, db)
  
  router.Run(":8080")
}
```

### 2. Set Environment Variables

```bash
# .env or docker-compose
TEMPORAL_SERVER_ADDRESS=localhost:7233
TEMPORAL_TASK_QUEUE=default
TEMPORAL_NAMESPACE=default
```

### 3. Start Temporal Worker

```bash
# In temporal/ directory
npm install @temporalio/worker @temporalio/client @temporalio/workflow @temporalio/activity

# Start worker (will block waiting for workflows)
ts-node -r tsconfig-paths/register worker.ts
```

## 📱 Frontend Integration

### 1. Import ABAC Provider

```tsx
// src/App.tsx
import { ABACProvider } from './components/abac';

export default function App() {
  return (
    <ABACProvider>
      <YourRoutes />
    </ABACProvider>
  );
}
```

### 2. Use ABAC Hooks in Components

```tsx
import { useABAC } from './components/abac';

function PolicyManagement() {
  const { evaluate, createPolicy, listPolicies } = useABAC();

  // Evaluate access decision
  const decision = await evaluate({
    subject: userId,
    action: 'edit_policy',
    resource: 'abac:policies',
  });

  if (decision.decision === 'deny') {
    return <AccessDenied reason={decision.reason} />;
  }

  // Create policy with tenant scoping
  await createPolicy({
    name: 'Finance Manager Policy',
    effect: 'allow',
    subject_rules: { roles: ['finance_manager'] },
    action_rules: { allowed_actions: ['view_reports', 'export_data'] },
    // ... tenant ID added automatically by hook
  });
}
```

### 3. Add ABAC Components to UI

```tsx
import {
  PolicyBuilder,
  DelegationManager,
  AuditLogViewer,
} from './components/abac';

function AdminDashboard() {
  return (
    <div>
      <PolicyBuilder />
      <DelegationManager />
      <AuditLogViewer />
    </div>
  );
}
```

## 🚀 Workflow Integration

### Starting Client Onboarding Workflow from API

```go
// In your trigger handler or API endpoint

import "github.com/semlayer/temporal/client"

func initiateClientOnboarding(c *gin.Context) {
  var req struct {
    ClientID  string `json:"client_id"`
    ClientName string `json:"client_name"`
    Email     string `json:"email"`
    ManagerID string `json:"manager_id"`
  }
  
  // Start Temporal workflow
  handle, err := temporal.StartClientOnboardingWorkflow(
    req.ClientID,
    map[string]interface{}{
      "name": req.ClientName,
      "email": req.Email,
      "manager_id": req.ManagerID,
    },
  )
  
  if err != nil {
    c.JSON(500, gin.H{"error": err.Error()})
    return
  }
  
  c.JSON(200, gin.H{"workflow_id": handle.ID})
}
```

### Polling Workflow Status from React

```tsx
import { useQuery } from '@tanstack/react-query';

function WorkflowStatus({ workflowId }) {
  const { data: status, isLoading } = useQuery(
    ['workflow', workflowId],
    async () => {
      const res = await fetch(`/api/workflows/${workflowId}/status`);
      return res.json();
    },
    { refetchInterval: 5000 } // Poll every 5 seconds
  );

  return (
    <div>
      <h3>Status: {status?.status}</h3>
      <p>Current Step: {status?.step}</p>
      <p>Approval Details: {JSON.stringify(status?.approvalDetails)}</p>
    </div>
  );
}
```

## 🔐 Multi-Tenant Enforcement

All ABAC operations automatically enforce tenant isolation:

### Backend Level
```go
// All ABAC handlers check tenant scope
tenantID := c.GetHeader("X-Tenant-ID")
datasourceID := c.GetHeader("X-Tenant-Datasource-ID")

if tenantID == "" || datasourceID == "" {
  c.JSON(http.StatusBadRequest, gin.H{"error": "Missing tenant scope"})
  return
}

// Query is scoped to tenant
query := `
  SELECT * FROM abac_policies
  WHERE tenant_id = $1 AND datasource_id = $2
`
```

### Frontend Level (Automatic)
```tsx
// The fetch shim (from agents.md) automatically adds headers:
// X-Tenant-ID: selected_tenant.id
// X-Tenant-Datasource-ID: selected_datasource.id
// Plus query parameters

// Your components just call the API normally:
const { data } = useQuery(
  ['policies'],
  () => fetch('/api/abac/policies').then(r => r.json())
  // Headers automatically added by shim!
);
```

## 📊 Audit Trail Integration

Every ABAC decision is logged:

```sql
-- audit_log table structure
CREATE TABLE audit_log (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  datasource_id UUID NOT NULL,
  actor VARCHAR(255) NOT NULL,
  action VARCHAR(255) NOT NULL,
  resource VARCHAR(255) NOT NULL,
  decision VARCHAR(10) NOT NULL, -- 'allow' or 'deny'
  reason TEXT,
  ip_address VARCHAR(45),
  timestamp TIMESTAMP DEFAULT NOW(),
  FOREIGN KEY (tenant_id) REFERENCES tenants(id),
  FOREIGN KEY (datasource_id) REFERENCES datasources(id)
);
```

View logs via React:
```tsx
<AuditLogViewer
  filterByAction="create_policy"
  filterByDays={30}
  onExport={(csv) => downloadCSV(csv)}
/>
```

## 🔄 Trigger System Integration

The ABAC system integrates seamlessly with your 13 Workday triggers:

### Before Executing a Trigger
```go
// In trigger_handlers.go

func handleTriggerExecution(triggerID string, action string) error {
  // 1. Evaluate ABAC policy
  decision, err := abacAPI.evaluatePolicy(ABACEvaluationRequest{
    Subject: currentUser,
    Action: action,
    Resource: "triggers:" + triggerID,
  })
  
  if decision == "deny" {
    return fmt.Errorf("Access denied: %s", decision.Reason)
  }
  
  // 2. Execute trigger
  return executeTrigger(triggerID)
}
```

### Timeout Escalation for Step Approvals
```go
// In trigger_engine.go

func (e *Engine) monitorStepTimeout(bpID, stepName string, timeoutHours int) {
  // Start timeout escalation workflow
  workflowID, err := temporal.StartTimeoutEscalationWorkflow(bpID, stepName, map[string]interface{}{
    "timeout_hours": timeoutHours,
    "escalation_action": "escalate", // or "notify", "auto_approve", "auto_reject"
    "manager_id": getCurrentManager(),
  })
  
  if err != nil {
    log.Printf("Error starting timeout workflow: %v", err)
  }
}
```

## 🧪 Testing ABAC Locally

### 1. Seed Test Policies

```bash
curl -X POST http://localhost:8080/api/abac/policies \
  -H "X-Tenant-ID: test-tenant-123" \
  -H "X-Tenant-Datasource-ID: test-datasource-456" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "View Reports Policy",
    "description": "Allow finance team to view reports",
    "effect": "allow",
    "priority": 100,
    "enabled": true,
    "subject_rules": {"roles": ["finance"]},
    "action_rules": {"allowed_actions": ["view_reports"]},
    "resource_rules": {"resource_types": ["report"]},
    "environment_rules": {"allowed_locations": ["office"]}
  }'
```

### 2. Test Policy Evaluation

```bash
curl -X POST http://localhost:8080/api/abac/evaluate \
  -H "X-Tenant-ID: test-tenant-123" \
  -H "X-Tenant-Datasource-ID: test-datasource-456" \
  -H "Content-Type: application/json" \
  -d '{
    "subject": "user-jane",
    "action": "view_reports",
    "resource": "report:sales-2024"
  }'
```

### 3. Create Test Delegations

```bash
curl -X POST http://localhost:8080/api/abac/delegations \
  -H "X-Tenant-ID: test-tenant-123" \
  -H "X-Tenant-Datasource-ID: test-datasource-456" \
  -H "Content-Type: application/json" \
  -d '{
    "from_user_id": "user-john",
    "to_user_id": "user-jane",
    "policy_id": "policy-xyz",
    "expires_at": "2024-12-31T23:59:59Z"
  }'
```

### 4. Start Client Onboarding Workflow

```bash
curl -X POST http://localhost:8080/api/workflows/client-onboarding \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "client-001",
    "client_name": "Acme Corp",
    "email": "contact@acme.com",
    "manager_id": "mgr-123"
  }'
```

## 🚨 Error Handling

### Policy Evaluation Failures
- Default decision: **DENY** (fail-secure)
- Reason logged to audit trail
- HTTP 500 returned only on system errors

### Workflow Execution Errors
- Automatic retry with exponential backoff
- Director escalation on persistent failure
- All attempts logged to audit trail

### Timeout Scenarios
- Step timeout → escalate to manager
- Manager timeout → escalate to director
- Director timeout → auto-approve/reject per configuration

## 📝 Configuration Reference

### abac_policies Table
```sql
CREATE TABLE abac_policies (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  datasource_id UUID NOT NULL,
  name VARCHAR(255) NOT NULL,
  description TEXT,
  effect VARCHAR(10) NOT NULL, -- 'allow' or 'deny'
  priority INT DEFAULT 100,
  enabled BOOLEAN DEFAULT true,
  subject_rules JSONB,
  action_rules JSONB,
  resource_rules JSONB,
  environment_rules JSONB,
  created_by VARCHAR(255),
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  UNIQUE(tenant_id, datasource_id, name)
);
```

### abac_delegations Table
```sql
CREATE TABLE abac_delegations (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  datasource_id UUID NOT NULL,
  from_user_id VARCHAR(255) NOT NULL,
  to_user_id VARCHAR(255) NOT NULL,
  policy_id UUID NOT NULL REFERENCES abac_policies(id),
  expires_at TIMESTAMP NOT NULL,
  created_at TIMESTAMP DEFAULT NOW()
);
```

## 🎯 Next Steps

1. **Run migrations** to create ABAC tables (included in existing schema)
2. **Start Temporal worker** to enable workflow execution
3. **Import ABAC components** into your admin dashboard
4. **Create initial policies** for your org structure
5. **Test workflows** with sample clients
6. **Monitor audit logs** for compliance/debugging

---

**Questions?** Refer to the agents.md guide for tenant scoping details and setupTenantFetch.ts for fetch shim behavior.
