# Phase 3: Architecture Guide - Semantic Rules Engine

## System Overview

The Semantic Rules Engine is a production-ready governance layer that enables business users to design, test, and manage data quality rules for the Calendar MDM system. It provides a visual interface for building priority-based rules without coding, complete workflow approvals, and comprehensive version control.

```
┌──────────────────────────────────────────────────────────────────┐
│  User Tier (React/TypeScript + Material-UI)                      │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │ SemanticRuleBuilder                                          │ │
│  │  ├── SemanticCatalog (Drag semantic terms)                  │ │
│  │  ├── PriorityHierarchyEditor (Build conditions)             │ │
│  │  ├── SimulationPanel (Test rules)                           │ │
│  │  └── RuleVersionControl (Governance)                        │ │
│  │                                                             │ │
│  │ Supporting Hooks:                                           │ │
│  │  ├── useRuleBuilder() - State management                   │ │
│  │  ├── useSemanticTerms() - Term discovery                   │ │
│  │  └── useSimulation() - Dry-run execution                   │ │
│  │                                                             │ │
│  │ API Service Layer:                                          │ │
│  │  └── ruleService.ts (13 HTTP clients)                      │ │
│  └─────────────────────────────────────────────────────────────┘ │
└────────────────────────────┬─────────────────────────────────────┘
                             │
                    HTTP REST API
                    /api/v1/rules/*
                             │
┌────────────────────────────┴─────────────────────────────────────┐
│  Application Tier (Go)                                           │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │ RuleHandler (13 endpoints)                                  │ │
│  │  ├── CRUD: Create, Read, Update, Delete                   │ │
│  │  ├── Publishing: Draft → Testing                           │ │
│  │  ├── Promotion: Testing → Staging → Production             │ │
│  │  ├── Simulation: Execute rules on test data               │ │
│  │  ├── Versioning: History & Diffs                          │ │
│  │  └── Approvals: Request & Track                           │ │
│  │                                                             │ │
│  │ Business Logic:                                             │ │
│  │  ├── RuleExecutionEngine (Simulate rule matching)         │ │
│  │  ├── ApprovalWorkflow (Route to appropriate roles)        │ │
│  │  ├── VersionControl (Track changes)                       │ │
│  │  └── AuditLog (Track mutations)                           │ │
│  └─────────────────────────────────────────────────────────────┘ │
└────────────────────────────┬─────────────────────────────────────┘
                             │
                        PostgreSQL
                    Connection Pool
                             │
┌────────────────────────────┴─────────────────────────────────────┐
│  Data Tier (PostgreSQL in 'alpha' database)                      │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │ edm.rules                                                   │ │
│  │  ├── Main rule definitions                                │ │
│  │  ├── Status: draft → testing → staging → production       │ │
│  │  ├── Version tracking                                      │ │
│  │  └── Tenant isolation (RLS)                               │ │
│  │                                                             │ │
│  │ edm.rule_steps                                             │ │
│  │  ├── Individual priority conditions                       │ │
│  │  ├── IF (Condition) clause definition                     │ │
│  │  ├── Confidence scores                                     │ │
│  │  └── Foreign key to rules                                 │ │
│  │                                                             │ │
│  │ edm.rule_versions                                          │ │
│  │  ├── Version history                                       │ │
│  │  ├── Promotion audit trail                                │ │
│  │  └── Rollback capability                                  │ │
│  │                                                             │ │
│  │ edm.rule_approvals                                         │ │
│  │  ├── Approval records                                      │ │
│  │  ├── Multi-role workflow                                   │ │
│  │  └── Audit trail                                          │ │
│  │                                                             │ │
│  │ edm.semantic_terms                                         │ │
│  │  ├── Business dimension catalog                           │ │
│  │  ├── By category & data type                              │ │
│  │  └── Governance status                                    │ │
│  │  └── **(deprecated)** semantic catalog moves to `public.catalog_node` ─ see integration guide
│  │                                                             │ │
│  │ edm.rule_execution_history                                │ │
│  │  ├── Simulation records                                    │ │
│  │  ├── Performance metrics                                   │ │
│  │  └── Audit trail                                          │ │
│  │                                                             │ │
│  │ edm.approval_workflows                                     │ │
│  │  ├── Approval requirements per stage                      │ │
│  │  └── Role definitions                                     │ │
│  └─────────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────────┘
```

---

## Component Deep Dives

### 1. Frontend Architecture

#### Component Hierarchy
```
SemanticRuleBuilder (Orchestrator)
├── DndContext (Drag-drop container)
├── AppBar (Header)
│   └── Tabs (Builder | Governance | Versions)
│
├── Grid Container (3-column layout)
│   │
│   ├── Column 1 (3 width) - SemanticCatalog
│   │   ├── TextField (Search)
│   │   └── Collapse sections
│   │       └── Draggable Cards per term
│   │
│   ├── Column 2 (6 width) - Tab Content
│   │   ├── Builder Tab
│   │   │   ├── SortableContext (for steps)
│   │   │   └── PriorityHierarchyEditor[] (Steps)
│   │   │       ├── CardHeader (Draggable)
│   │   │       ├── Collapse (Expandable)
│   │   │       │   ├── FormControl (Semantic Term)
│   │   │       │   ├── FormControl (Operator)
│   │   │       │   ├── TextField (Value)
│   │   │       │   └── Slider (Confidence)
│   │   │       └── Button (Delete, Duplicate)
│   │   │
│   │   ├── Governance Tab (RuleVersionControl)
│   │   │   ├── Stepper (Workflow stages)
│   │   │   └── Accordion (Version history)
│   │   │
│   │   └── Versions Tab (Diff viewer)
│   │       └── Version comparison
│   │
│   └── Column 3 (3 width) - SimulationPanel
│       ├── Header (Scenario selector)
│       ├── Tabs
│       │   ├── Test Data Tab
│       │   ├── Execution Trace Tab
│       │   └── Impact Analysis Tab
│       └── Footer (Action buttons)
```

#### Data Flow
```
User Action
    ↓
Event Handler (onClick, onDragEnd, etc.)
    ↓
Hook Method (addStep, updateStep, saveRule, etc.)
    ↓
API Service Call (POST /api/v1/rules, etc.)
    ↓
Backend HTTP Handler
    ↓
Database Write/Read
    ↓
Response JSON
    ↓
Hook State Update
    ↓
Component Re-render
    ↓
UI Updates
```

#### Example: Create Rule Flow
```
1. User clicks [+ Add Priority]
   ↓
2. React-dnd detects drop event
   ↓
3. SemanticRuleBuilder.handleDragEnd() fires
   ↓
4. useRuleBuilder.addStep() called
   ↓
5. Hook optimistically updates rule.steps
   ↓
6. PriorityHierarchyEditor renders new step
   ↓
7. User enters condition details
   ↓
8. FormControl onChange → updateStep()
   ↓
9. User clicks Save
   ↓
10. useRuleBuilder.saveRule() → PUT /api/v1/rules/{id}
    ↓
11. Backend validates & saves
    ↓
12. Returns updated rule
    ↓
13. Hook updates state
    ↓
14. UI reflects saved rule
```

---

### 2. Backend Architecture

#### Request Pipeline
```
HTTP REQUEST
    ↓
[Middleware Layer]
    ├── CORSMiddleware (Allow frontend)
    ├── AuthMiddleware (Validate JWT)
    ├── TenantMiddleware (Extract X-Tenant-ID)
    ├── LoggingMiddleware (Structured logs)
    └── RateLimitMiddleware (Rate limiting)
    ↓
[RuleHandler]
    ├── Validate request (required fields, types)
    ├── Extract tenantID from context
    ├── Business logic (state transitions, etc.)
    ├── Database operations (INSERT/UPDATE/DELETE)
    ├── Error handling (try-catch)
    ├── Audit logging (mutation details)
    └── Response marshaling (JSON)
    ↓
HTTP RESPONSE (JSON)
```

#### Database Transaction Pattern
```go
// Pseudo-code for typical handler
func (h *RuleHandler) PublishRule(w, r) {
    // 1. Begin transaction
    tx := h.db.BeginTx(ctx)
    
    // 2. Fetch rule (with lock)
    rule := GetRuleForUpdate(tx, ruleID)
    
    // 3. Validate status
    if rule.Status != "draft" {
        tx.Rollback()
        ErrorResponse(400)
    }
    
    // 4. Update rule
    rule.Status = "testing"
    rule.Version += 1
    SaveRule(tx, rule)
    
    // 5. Create version record
    CreateRuleVersion(tx, rule)
    
    // 6. Log audit event
    LogAudit(tx, "RULE_PUBLISHED", rule.ID, userID)
    
    // 7. Publish event (Redpanda)
    PublishEvent("rule-published", rule)
    
    // 8. Commit
    tx.Commit()
    
    // 9. Response
    SuccessResponse(rule)
}
```

#### Endpoint Details

**Endpoint: POST /api/v1/rules (Create Rule)**
```
Request:
{
  "businessObject": "calendar",
  "name": "Weekend Override",
  "description": "Use golden record for weekends",
  "steps": [...],
  "defaultAction": "use_source_field"
}

Response: 201 Created
{
  "id": "rule_uuid",
  "businessObject": "calendar",
  "version": 1,
  "status": "draft",
  "createdAt": "2026-02-20T12:00:00Z",
  ...
}

Database Operations:
├── INSERT INTO edm.rules (id, business_object, ...)
└── INSERT INTO edm.rule_steps (rule_id, priority, ...)

Audit Log:
├── RULE_CREATED: rule_uuid, calendar, user_uuid
└── Timestamp: 2026-02-20T12:00:00Z
```

**Endpoint: POST /api/v1/rules/{id}/publish (Publish to Testing)**
```
Request:
{
  "version": 1,
  "description": "Ready for testing with 2026 calendar"
}

Response: 200 OK
{
  "id": "rule_uuid",
  "version": 2,
  "status": "testing",
  "publishedAt": "2026-02-20T12:30:00Z",
  ...
}

Database Operations:
├── UPDATE edm.rules SET status='testing', version=2
├── INSERT INTO edm.rule_versions (rule_id, version, status, ...)
└── INSERT INTO edm.rule_execution_history (...)

Events Published:
└── rule-published: {rule_id, version, status}
```

**Endpoint: POST /api/v1/rules/{id}/simulate (Execute Rule)**
```
Request:
{
  "testData": {
    "dates": ["2026-02-20", "2026-02-21", ...],
    "regions": ["GB", "US", ...]
  }
}

Response: 200 OK
{
  "executionTrace": [
    {
      "date": "2026-02-20",
      "region": "GB",
      "winningRule": "Step#1",
      "confidence": 95,
      "evaluatedRules": ["Step#1", "Step#2", "DEFAULT"]
    },
    ...
  ],
  "impactedDates": 150,
  "changedDates": 23,
  "avgConfidence": 88.5,
  "samples": [...]
}

Database Operations:
├── SELECT * FROM edm.rules WHERE id = ?
├── SELECT * FROM edm.rule_steps WHERE rule_id = ?
└── INSERT INTO edm.rule_execution_history (...)

No external calls (pure computation)
```

---

### 3. Database Architecture

> **Semantic Catalog Integration:**
> In the latest design the simple `edm.semantic_terms` table is being phased out in favour of
> a full semantic graph implemented in the `public` schema (`catalog_node_type`,
> `catalog_node`, `catalog_edge_type(s)`, `catalog_edge`).  This allows calendar and other
> business objects to reference reusable terms directly and provides lineage, multi‑tenant
> overrides, and rule resolution.  See [PHASE_3_SEMANTIC_INTEGRATION.md](./PHASE_3_SEMANTIC_INTEGRATION.md)
> for details and migration scripts.
>

#### Schema Design Rationale

**Table 1: edm.rules**
- Primary store for rule metadata
- Denormalized `current_version` for quick status lookups
- Immutable timestamps (created_at never changes)
- Updated_at tracks last modification
- Status enum enforces valid workflow states

**Table 2: edm.rule_steps**
- Represents individual conditions (IF clauses)
- Separated from rules for flexible query & versioning
- Priority field determines evaluation order
- Confidence allows for soft constraints

**Table 3: edm.rule_versions**
- Complete audit trail of all changes
- Enables rollback (reference source_version)
- Tracks promotion with promoted_at timestamp
- Supports diff calculations

**Table 4: edm.rule_approvals**
- Governance workflow tracking
- Multi-role approval (stage-specific)
- Status transitions (pending → approved/rejected)
- Comments for decision rationale

**Table 5: edm.approval_workflows**
- Configuration for approval requirements
- Sequence order ensures correct role sequence
- Per-stage definitions (testing vs staging vs production)

**Table 6: edm.semantic_terms**
- Business term dictionary
- Sample values help users understand meaning
- Governance status prevents using deprecated terms
- Categories organize by business meaning

#### Query Patterns

**Pattern 1: List rules for a business object**
```sql
SELECT * FROM edm.rules 
WHERE tenant_id = $1 
  AND business_object = $2 
  AND status = $3  -- Optional filter
ORDER BY created_at DESC;

Index: (tenant_id, business_object, status)
Expected latency: 10-20ms
```

**Pattern 2: Fetch rule with all steps**
```sql
SELECT r.*, s.* FROM edm.rules r
LEFT JOIN edm.rule_steps s ON r.id = s.rule_id
WHERE r.id = $1 
  AND r.tenant_id = $2 
  AND s.version = r.current_version;

Index: (rule_id, version) on rule_steps
Expected latency: 5-10ms
```

**Pattern 3: Get approval requirements**
```sql
SELECT * FROM edm.approval_workflows
WHERE business_object = $1 
  AND promotion_stage = $2
ORDER BY sequence_order;

Index: (business_object, promotion_stage)
Expected latency: 2-5ms (small table)
```

**Pattern 4: Check pending approvals for user**
```sql
SELECT ra.*, r.name, r.business_object
FROM edm.rule_approvals ra
JOIN edm.rules r ON ra.rule_id = r.id
WHERE ra.status = 'pending' 
  AND (ra.approver_id IS NULL OR ra.approver_id = $1)
  AND r.tenant_id = $2
ORDER BY ra.created_at DESC;

Index: (status, approver_id, tenant_id)
Expected latency: 20-50ms
```

#### Row-Level Security (RLS) Policy

```sql
-- Policy: users can only see rules in their tenant
CREATE POLICY rules_tenant_isolation ON edm.rules
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

-- How it works:
-- 1. Before any query, set session variable:
--    SET app.current_tenant_id TO '550e8400-e29b-41d4-a716-446655440000';
--
-- 2. Query automatically filtered:
--    SELECT * FROM rules WHERE status = 'production'
--    ↓ (becomes)
--    SELECT * FROM rules 
--    WHERE status = 'production' 
--      AND tenant_id = '550e8400-e29b-41d4-a716-446655440000'
--
-- 3. Prevents accidental data leaks across tenants
-- 4. Bypassed only by superuser (admin role)
```

---

### 4. Workflow Architecture

#### Status Workflow
```
┌─────────┐
│ DRAFT   │  Created by user
└────┬────┘  (Can edit/delete)
     │
     │ PublishRule()
     ↓
┌─────────┐
│ TESTING │  Ready for internal testing
└────┬────┘  (Can't edit, awaits approval)
     │
     │ PromoteRule() + DataSteward approval
     ↓
┌──────────┐
│ STAGING  │  Staged to production env
└────┬─────┘  (Can't edit, awaits compliance)
     │
     │ PromoteRule() + ComplianceOfficer approval
     ↓
┌────────────┐
│ PRODUCTION │  Active rule on production data
└────┬───────┘  (Can't edit, can rollback)
     │
     │ RollbackRule()
     ↓
┌─────────┐
│ DRAFT   │  New draft from rollback
└─────────┘
```

#### Approval Workflow
```
Rule Status: TESTING
    ↓
PostApprovalRequest(
  ruleId: uuid,
  version: 2,
  role: "data_steward",
  action: "approve"
)
    ↓
Insert into edm.rule_approvals
    id: uuid
    status: "approved"
    approver: current_user
    ↓
GET /api/approvals/pending (for admins)
    ├── Shows pending approvals for next stage
    ↓
Can now call PromoteRule(...)
    ├── Validates all required approvals complete
    ├── Updates status to next stage
    ├── Increments version
    ↓
New approval requirements cascade (if more stages)
    ↓
Rule in PRODUCTION
```

---

### 5. Integration Points

#### With Phase 2 (Event Streaming)

**Events Published by Rules Engine:**
```
Topic: rule-events
├── RuleCreatedEvent
│   ├── rule_id, business_object, name
│   └── created_by, timestamp
│
├── RulePublishedEvent
│   ├── rule_id, version, status, description
│   └── published_by, timestamp
│
├── RulePromotedEvent
│   ├── rule_id, version, from_stage, to_stage
│   └── promoted_by, timestamp
│
├── ApprovalRequestedEvent
│   ├── rule_id, version, role, approver
│   └── timestamp
│
└── ApprovalCompletedEvent
    ├── rule_id, version, role, action, approver
    └── timestamp
```

**Events Consumed:**
```
Topic: calendar-updated
├── Trigger rule re-evaluation
├── Update affected dates
└── Emit rule-impacted-dates event
```

#### With Phase 1 (MDM)

**Data Dependencies:**
```
Calendar MDM (edm.mdm_calendar)
    ├── Provides source data for simulation
    ├── Reference for business day determination
    └── Source for region-specific logic
    
User queries:
    ├── Simulation uses calendar MDM records
    ├── Tests rules against golden records
    └── Reports date counts & impacts
```

---

## Performance Characteristics

### Expected Latencies

| Operation | Latency | Notes |
|-----------|---------|-------|
| GET /api/v1/rules | 50-100ms | Depends on count, uses limit 50 |
| POST /api/v1/rules | 100-200ms | Includes audit logging |
| PUT /api/v1/rules/{id} | 80-150ms | Draft only |
| POST /api/v1/rules/{id}/publish | 200-300ms | Creates version + audit |
| POST /api/v1/rules/{id}/simulate | 500-2000ms | Depends on data size |
| GET /api/v1/rules/{id}/versions | 30-50ms | Version table |
| GET /api/v1/rules/{id}/diff | 100-300ms | Comparison computation |
| POST /api/v1/approvals | 100-150ms | Single insert |
| GET /api/v1/approvals/pending | 50-100ms | Indexed query |

### Capacity Targets

| Metric | Value | Notes |
|--------|-------|-------|
| Rules per business object | 1,000+ | Depends on retention |
| Steps per rule | 10+ | No hard limit |
| Tenants | 100+ | RLS enforced |
| Concurrent users | 50+ | Per environment |
| Requests per minute | 10,000 | Aggregate |
| Simulation/minute | 100 | Depends on data size |

### Scaling Strategies

**Horizontal:**
- Read replicas for SELECT queries
- Cache layer (Redis) for semantic_terms
- Connection pooling

**Vertical:**
- Increase DB indexes
- Tune PostgreSQL settings
- Batch audit logging

---

## Security Architecture

### Authentication
```
User Login
    ↓
Auth Provider (JWT)
    ├── Verify signature
    ├── Extract claims (user_id, roles, tenant_id)
    └── Store in request context
    ↓
Middleware Validation
    ├── Check X-Tenant-ID matches JWT
    ├── Verify user has role required for action
    └── Set session RLS variable
```

### Authorization
```
Rule: Only data_stewards can approve testing stage
    ↓
Frontend: Hide "Approve" button for non-stewards
    ↓
Backend: Validate role on POST /approve
    ├── Check user role in JWT
    ├── Check role matches rule's required approver
    ├── Reject if mismatch
    └── Return 403 Forbidden
```

### Data Isolation
```
Query:  SELECT * FROM edm.rules WHERE status = 'production'
    ↓
RLS Policy Applied:
    Implicit: AND tenant_id = current_setting('app.current_tenant_id')
    ↓
Actual Query Executed:
    SELECT * FROM edm.rules 
    WHERE status = 'production'
      AND tenant_id = $1
```

---

## Error Handling

### HTTP Error Responses
```json
400 Bad Request
{
  "error": "Invalid request",
  "message": "businessObject is required",
  "code": "VALIDATION_ERROR"
}

401 Unauthorized
{
  "error": "Authentication failed",
  "message": "Invalid token",
  "code": "AUTH_ERROR"
}

403 Forbidden
{
  "error": "Permission denied",
  "message": "User role 'analyst' cannot approve rules",
  "code": "AUTHZ_ERROR"
}

404 Not Found
{
  "error": "Resource not found",
  "message": "Rule with ID 'rule_uuid' does not exist",
  "code": "NOT_FOUND"
}

409 Conflict
{
  "error": "Invalid state transition",
  "message": "Cannot publish non-draft rule",
  "code": "STATE_ERROR"
}

500 Internal Server Error
{
  "error": "Internal server error",
  "message": "Database connection failed",
  "code": "DATABASE_ERROR",
  "requestId": "abc123def456"  # For tracing
}
```

### Frontend Error Handling
```typescript
try {
    const rule = await ruleService.createRule(request);
} catch (error) {
    if (error.response?.status === 400) {
        // Validation error - show form feedback
        setValidationErrors(error.response.data.fields);
    } else if (error.response?.status === 409) {
        // Conflict - state error
        showMessage("Cannot perform this action on a published rule");
    } else {
        // Network/server error
        showErrorToast("Failed to create rule. Please try again.");
        logToSentry(error);
    }
}
```

---

## Observability

### Key Metrics
```go
// Rule creation rate
rulesCreatedPerHour := prometheus.NewGauge(...)

// Rule status distribution
rulesByStatus := prometheus.NewGaugeVec(
    []string{"status", "business_object"},
)

// Approval cycle time
approvalCycleTime := prometheus.NewHistogram(...)

// Simulation execution time
simulationDuration := prometheus.NewHistogram(...)
```

### Example Grafana Dashboards

**Dashboard 1: Rules Overview**
- Rules created per day
- Rules by status (pie chart)
- Rules by business object (bar chart)
- Average steps per rule

**Dashboard 2: Workflow Health**
- Approval cycle time (p50, p95, p99)
- Pending approvals count
- Rollback frequency
- Promotion success rate

**Dashboard 3: Performance**
- API endpoint latency (per endpoint)
- Database query latency
- Simulation duration distribution
- Error rate by endpoint

### Audit Logging

Every mutation is logged:
```json
{
  "timestamp": "2026-02-20T12:00:00Z",
  "level": "info",
  "action": "RULE_PUBLISHED",
  "actor": "user_uuid",
  "actor_role": "data_steward",
  "tenant_id": "tenant_uuid",
  "resource_id": "rule_uuid",
  "changes": {
    "status": {"old": "draft", "new": "testing"},
    "version": {"old": 1, "new": 2}
  },
  "request_id": "trace_uuid"
}
```

---

## Disaster Recovery

### Backup Strategy
```
Full Backup: Daily at 02:00 UTC
    └── pg_dump with compression
    └── Stored: S3 with versioning
    └── Retention: 30 days

Incremental Backup: Hourly
    └── PostgreSQL WAL archiving
    └── Stored: S3
    └── Retention: 7 days

Test: Weekly restore from backup
    └── Verify data integrity
    └── Document recovery time
```

### Failover Procedure
```
Issue Detected
    ↓
1. Take database snapshot
2. Promote read replica to primary
3. Point application to new primary
4. Verify data integrity
5. Monitor for 30 minutes
6. Document incident

Expected Recovery Time: 5-10 minutes
Expected Data Loss: < 1 minute
```

---

## Configuration Management

### Rule-Based Configuration
```yaml
# rules-config.yaml
approval_requirements:
  testing:
    - role: data_steward
      required: true
  staging:
    - role: compliance_officer
      required: true
  production:
    - role: business_owner
      required: true

approval_timeout_days: 7

rate_limits:
  rules_per_minute: 60
  simulations_per_minute: 100

simulation:
  max_test_data_rows: 100000
  timeout_seconds: 5
  cache_ttl_minutes: 15
```

---

## Next Steps / Roadmap

**Phase 3 Complete (This Document):**
- ✅ Frontend components (5 React components)
- ✅ Backend handlers (13 endpoints)
- ✅ Database schema (6 tables)
- ✅ API contracts
- ✅ Architecture documented

**Phase 4 (Advanced Features):**
- [ ] Rule templates (reusable patterns)
- [ ] Rule composition (nested rules)
- [ ] ML-assisted suggestions
- [ ] Bulk operations
- [ ] Advanced performance metrics

**Phase 5 (Scale & Optimize):**
- [ ] Read-replica scaling
- [ ] Caching layer (Redis)
- [ ] Event-driven architecture
- [ ] GraphQL API (optional)
- [ ] Advanced search/filtering

---

**Document Version:** 1.0.0  
**Last Updated:** 2026-02-20  
**Audience:** Developers, Architects, DevOps Engineers
