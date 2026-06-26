# Workday Trigger System Implementation Guide

## Overview

This implementation provides a complete Workday-style trigger system for your application. Triggers are application-layer smart events that enforce validation rules at key points (Create, Save, Delete, Field Change, Workflow Step, etc.).

**Key principle**: Triggers are NOT database triggers. They are application logic that runs before/after data mutations, ensuring consistent validation across your entire platform.

## 13 Trigger Types (Current Coverage: 7/13)

| # | Type | When | Your Code | Status |
|---|------|------|-----------|--------|
| 1 | Save | Entity Save | POST/PATCH handler | ✅ LIVE |
| 2 | Field Change | Field Modified | Form onChange | ✅ LIVE |
| 3 | Delete | Entity Delete | DELETE handler | ✅ LIVE |
| 4 | Create | New Entity | POST handler | ✅ LIVE |
| 5 | Workflow Step | BP Step Complete | Temporal Activity | ✅ READY (Phase 6A) |
| 6 | Bulk Load | Import Batch | Future | 🔄 |
| 7 | Integration | External Event | RabbitMQ | ✅ LIVE |
| 8 | Time-Based | Scheduled | Cron | Future |
| 9 | Sub-Entity Change | Child Modified | Hierarchy handlers | ✅ LIVE |
| 10 | Relationship Change | Link Modified | FK handlers | ✅ LIVE |
| 11 | Status Change | Status Updated | Future | 🔄 |
| 12 | Calculated Field | Formula Recalc | Future | 🔄 |
| 13 | Security Role | Role Assignment | Future | 🔄 |

## Architecture

### Flow: User Action → Validation → DB Commit

```
1. User Action (Save Order)
   ↓
2. HTTP Handler receives request
   ↓
3. Trigger Engine: Fetch triggers for (tenant, "save", "orders")
   ↓
4. For each trigger → Fetch validation rules (catalog_validation_rules)
   ↓
5. Evaluate rules against payload (field_format, cardinality, FK, etc.)
   ├─ PASS → Continue to DB commit + RabbitMQ event
   └─ FAIL → Return 400 error to client
   ↓
6. DB transaction committed
   ↓
7. Publish event to RabbitMQ: orders.created / orders.updated / orders.deleted
```

## Database Schema

### 1. validation_triggers Table

```sql
CREATE TABLE validation_triggers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    trigger_type VARCHAR(50) NOT NULL,     -- "save", "create", "delete", "field_change", "workflow_step"
    target_entity VARCHAR(128) NOT NULL,   -- "orders", "customers"
    step_name VARCHAR(128),                -- optional for workflow_step triggers
    rule_ids UUID[] NOT NULL,              -- array of rule IDs to evaluate
    meta JSONB DEFAULT '{}'::jsonb,
    created_by UUID,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now()
);

CREATE INDEX idx_validation_triggers_tenant_type_entity 
  ON validation_triggers(tenant_id, trigger_type, target_entity);
```

### 2. workflow_timeout_triggers Table

```sql
CREATE TABLE workflow_timeout_triggers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    workflow_name VARCHAR(100) NOT NULL,  -- "HireEmployee"
    step_name VARCHAR(100) NOT NULL,      -- "ManagerApproval"
    due_hours INT NOT NULL,               -- 48
    trigger_percentages JSONB,            -- [80, 100]
    actions_json JSONB NOT NULL,          -- [{"percent": 100, "type": "escalate", "target": "hr_director"}]
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now()
);

CREATE INDEX idx_timeout_triggers_workflow 
  ON workflow_timeout_triggers(tenant_id, workflow_name, step_name);
```

### 3. seed: Auto-trigger all Save rules

```sql
-- Create a trigger for every existing validation rule
INSERT INTO validation_triggers (tenant_id, trigger_type, target_entity, rule_ids)
SELECT 
    tenant_id, 
    'save', 
    target_entities[1], 
    ARRAY[id]::UUID[]
FROM catalog_validation_rules 
WHERE rule_type IN ('field_format', 'cardinality', 'referential_integrity');
```

## Go Implementation

### 1. Core Trigger Engine (trigger.go)

```go
type TriggerValidationEngine struct {
    ValidationEngine *ValidationEngine  // core engine
    db *sql.DB
    logger Logger
}

// TriggerValidate is your main function to call before DB commits
func (tve *TriggerValidationEngine) TriggerValidate(ctx context.Context, tenantID uuid.UUID, triggerType, entity, stepName string, data map[string]interface{}) error
```

**Usage in handlers:**

```go
// In CreateOrderHandler:
if err := engine.TriggerValidate(ctx, tenantID, "create", "orders", "", orderData); err != nil {
    http.Error(w, err.Error(), http.StatusBadRequest)
    return
}
// Safe to save
db.ExecContext(ctx, "INSERT INTO orders...", orderData)
```

### 2. HTTP Handlers (validation_triggers_handlers.go)

#### POST /api/validate/field
Quick field validation (for onChange events, optional):

```bash
curl -X POST "http://localhost:29080/api/validate/field" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 910638ba-a459-4a3f-bb2d-78391b0595f6" \
  -d '{
    "entity": "customers",
    "field": "phone",
    "value": "+15551234567",
    "record": {}
  }'
```

Response (pass):
```json
{"status":"pass"}
```

Response (fail):
```json
{"error":"PhoneFormat failed: Phone must be valid E.164"}
```

#### POST /api/admin/validation-triggers
Create a new trigger (admin only):

```bash
curl -X POST "http://localhost:29080/api/admin/validation-triggers" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 910638ba-a459-4a3f-bb2d-78391b0595f6" \
  -d '{
    "trigger_type": "save",
    "target_entity": "orders",
    "rule_ids": ["11111111-1111-1111-1111-111111111111"]
  }'
```

#### GET /api/admin/validation-triggers?entity=orders
List triggers for an entity:

```bash
curl "http://localhost:29080/api/admin/validation-triggers?entity=orders" \
  -H "X-Tenant-ID: 910638ba-a459-4a3f-bb2d-78391b0595f6"
```

### 3. Example Integration: Orders Handler

See `backend/internal/api/orders_handlers_example.go` for:
- `HandleCreateOrder()` - runs "create" triggers
- `HandleUpdateOrder()` - runs "save" triggers
- `HandleDeleteOrder()` - runs "delete" triggers

**Pattern:**

```go
func (h *OrdersHandler) HandleCreateOrder(w http.ResponseWriter, r *http.Request) {
    // 1. Parse request
    var req CreateOrderRequest
    json.NewDecoder(r.Body).Decode(&req)
    
    // 2. Convert to validation data
    orderData := map[string]interface{}{"customer_id": req.CustomerID, "total": req.Total}
    
    // 3. Run trigger validation
    if err := h.triggerEngine.TriggerValidate(ctx, tid, "create", "orders", "", orderData); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    // 4. Safe to persist
    db.ExecContext(ctx, "INSERT INTO orders...", orderData)
    
    // 5. Publish event
    amqp.Publish("orders.created", orderData)
}
```

### 4. Timeout Triggers (timeout_monitor.go)

**Temporal workflow that runs every hour:**

```go
func TimeoutMonitorWorkflow(ctx workflow.Context, db *sql.DB) error {
    for {
        workflow.Sleep(ctx, time.Hour)
        checkWorkflowTimeouts(ctx, db)
    }
}
```

**Actions when timeout threshold is hit:**
- `escalate`: Reassign to higher role (e.g., HR Director)
- `notify`: Send email reminder
- `log`: Record audit entry
- `cancel`: Auto-cancel (for critical timeouts >7 days)

**Example:** Manager Approval due in 48h:
- At 40h (80%): Send reminder email
- At 48h (100%): Escalate to HR Director + log event

Register in worker:
```go
w.RegisterWorkflow(TimeoutMonitorWorkflow)
c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{TaskQueue: "monitor"}, TimeoutMonitorWorkflow)
```

## Testing

### Unit Tests

Run trigger validation tests:
```bash
go test ./backend/internal/validation -v
```

Expected output:
```
=== RUN   TestTriggerValidate_Pass
--- PASS: TestTriggerValidate_Pass (0.00s)
=== RUN   TestTriggerValidate_Fail
--- PASS: TestTriggerValidate_Fail (0.00s)
=== RUN   TestValidateField_Pass
--- PASS: TestValidateField_Pass (0.00s)
=== RUN   TestValidateField_Fail
--- PASS: TestValidateField_Fail (0.00s)
PASS
```

### Curl Integration Tests

**Test 1: Save with VALID data (pass)**
```bash
curl -i -X POST "http://localhost:29080/api/orders" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 910638ba-a459-4a3f-bb2d-78391b0595f6" \
  -d '{"customer_id": 1, "total": 100}'

# Expected: 201 Created
# Response: {"id":"order-123","status":"created"}
```

**Test 2: Save with INVALID data (fail)**
```bash
curl -i -X POST "http://localhost:29080/api/orders" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 910638ba-a459-4a3f-bb2d-78391b0595f6" \
  -d '{"customer_id": 999, "total": -50}'

# Expected: 400 Bad Request
# Response: {"error":"OrderTotalPositive failed: Total must be > 0"}
```

**Test 3: Field change validation (onChange)**
```bash
curl -i -X POST "http://localhost:29080/api/validate/field" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 910638ba-a459-4a3f-bb2d-78391b0595f6" \
  -d '{"entity":"customers","field":"phone","value":"invalid"}'

# Expected: 400 Bad Request
# Response: {"error":"PhoneFormat failed: Phone must be E.164"}
```

## Wire into Your App

### Step 1: Register routes in main API setup

```go
// In your main api.go or routes.go
import "github.com/eganpj/semlayer/backend/internal/validation"

triggerEngine := validation.NewTriggerValidationEngine(db, logger)
httpapi.RegisterValidationTriggersRoutes(r, db, triggerEngine)
```

### Step 2: Update your existing handlers

```go
// Before: Just insert to DB
// db.ExecContext(ctx, "INSERT INTO orders...", data)

// After: Add trigger validation
if err := h.triggerEngine.TriggerValidate(ctx, tenantID, "create", "orders", "", data); err != nil {
    http.Error(w, err.Error(), http.StatusBadRequest)
    return
}
db.ExecContext(ctx, "INSERT INTO orders...", data)
```

### Step 3: Create initial triggers (admin UI or SQL)

Via UI (create a rule, auto-checkbox creates trigger):
1. Create a validation rule (e.g., "Order Total Positive")
2. Rule creation auto-creates a "save" trigger
3. Next time someone creates/saves an order, the rule runs

Via SQL (manual seed):
```sql
INSERT INTO validation_triggers (tenant_id, trigger_type, target_entity, rule_ids)
VALUES ('tenant-1', 'save', 'orders', ARRAY['rule-1'::uuid]);
```

## Monitoring & Audits

### Check trigger execution (via logs)

```bash
tail -f /var/log/app.log | grep "TriggerValidate"
```

### Query admin audit log

```sql
SELECT * FROM admin_audit_logs 
WHERE action = 'create' AND status = 'failed'
ORDER BY created_at DESC;
```

### Check timeout monitor status

```bash
curl http://localhost:29080/api/admin/temporal-monitor/status
# Returns: { "running": true, "last_check": "2025-10-28T20:15:00Z", "workflows_checked": 42 }
```

## Next Steps

1. **Phase 6A (Workflow Step Triggers)**: Wire TimeoutMonitorWorkflow into your Temporal setup
2. **Phase 6B (Bulk Load)**: Add batch validation for CSV imports
3. **Phase 6C (Status/Role Changes)**: Implement remaining trigger types
4. **Analytics**: Add dashboard showing rule hit rates, validation failures by type

## Files

- `backend/internal/validation/trigger.go` - Core engine
- `backend/internal/validation/trigger_test.go` - Unit tests
- `backend/internal/api/validation_triggers_handlers.go` - HTTP endpoints
- `backend/internal/api/orders_handlers_example.go` - Example integration
- `backend/internal/temporal/timeout_monitor.go` - Timeout worker
