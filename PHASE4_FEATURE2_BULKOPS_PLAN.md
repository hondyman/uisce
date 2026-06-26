# Phase 4 Feature 2: Bulk Operations - Implementation Plan

**Status**: 🚀 STARTING NOW  
**Date**: February 20, 2026  
**Estimated Time**: 2-3 hours  
**Predecessor**: Phase 4 Feature 1 ✅ (100% Complete)

---

## Executive Summary

Phase 4 Feature 2 adds bulk operations support to the Rule Templates API, allowing users to:
- **Bulk Create**: Import multiple templates at once
- **Bulk Publish**: Publish/approve multiple templates in a single operation
- **Bulk Promote**: Promote multiple rules across environments simultaneously

These operations are critical for:
- Template library initialization (importing 100+ templates from external sources)
- Governance workflows (review and approve multiple templates at once)
- Rule promotion (move sets of related rules through dev → staging → prod)

---

## Feature Breakdown

### Feature 2.1: Bulk Create Templates
**Endpoint**: `POST /api/v1/templates/bulk-create`

**Purpose**: Import multiple templates in a single API call

**Request Body**:
```json
{
  "templates": [
    {
      "businessObject": "calendar",
      "name": "Weekend Override",
      "description": "...",
      "category": "weekend",
      "baseRuleSteps": [...],
      "parameterSchema": {},
      "isPublic": false
    },
    {
      "businessObject": "calendar",
      "name": "Holiday Override",
      "description": "...",
      "category": "holiday",
      "baseRuleSteps": [...],
      "parameterSchema": {},
      "isPublic": false
    }
    // ... up to 1000 templates
  ],
  "continueOnError": false,  // true = skip failures, false = fail entire batch
  "tags": ["import-v1", "automated"]  // Optional metadata
}
```

**Response (Success)**:
```json
{
  "status": "success",
  "created": 2,
  "failed": 0,
  "results": [
    {
      "templateName": "Weekend Override",
      "id": "uuid-1",
      "status": "created"
    },
    {
      "templateName": "Holiday Override",
      "id": "uuid-2",
      "status": "created"
    }
  ],
  "batchId": "batch-uuid",
  "timestamp": "2026-02-20T20:30:00Z"
}
```

**Response (With Errors)**:
```json
{
  "status": "partial",
  "created": 1,
  "failed": 1,
  "results": [
    {
      "templateName": "Weekend Override",
      "id": "uuid-1",
      "status": "created"
    },
    {
      "templateName": "Holiday Override",
      "error": "Invalid parameter schema: missing required property",
      "status": "failed"
    }
  ],
  "batchId": "batch-uuid",
  "timestamp": "2026-02-20T20:30:00Z"
}
```

**HTTP Status Codes**:
- `201 Created`: All templates created successfully
- `207 Multi-Status`: Some created, some failed (only if `continueOnError: true`)
- `400 Bad Request`: Validation failed, no templates created
- `403 Forbidden`: Insufficient permissions
- `413 Payload Too Large`: Request exceeds 50MB limit

**Validations**:
- Max 1000 templates per request
- Max 50MB total payload
- Each template must pass individual validation
- Business object must exist in system
- Names must be unique within tenant
- Parameter schema must be valid JSON

**Transaction Behavior**:
- If `continueOnError: false` → all-or-nothing (ACID)
- If `continueOnError: true` → best-effort (some may fail)

---

### Feature 2.2: Bulk Publish Templates
**Endpoint**: `POST /api/v1/templates/bulk-publish`

**Purpose**: Change status of multiple templates from draft → approved

**Request Body**:
```json
{
  "templateIds": ["uuid-1", "uuid-2", "uuid-3"],
  "targetStatus": "approved",  // or "archived", "deprecated"
  "requireApproval": true,     // If true, just flag for approval
  "approvalComment": "Approved by admin batch processing"
}
```

**Response**:
```json
{
  "status": "success",
  "published": 3,
  "failed": 0,
  "results": [
    {
      "id": "uuid-1",
      "name": "Template Name",
      "previousStatus": "draft",
      "newStatus": "approved",
      "status": "published"
    },
    // ...
  ],
  "batchId": "batch-uuid",
  "timestamp": "2026-02-20T20:30:00Z"
}
```

**Validations**:
- All template IDs must exist
- All templates must be in draft status
- User must have permission to publish (can add role check)
- Max 500 templates per batch

**Error Handling**:
```json
{
  "status": "error",
  "published": 0,
  "failed": 3,
  "errors": [
    {
      "templateId": "uuid-1",
      "error": "Template not found"
    },
    {
      "templateId": "uuid-2",
      "error": "Template already approved (status: approved)"
    },
    {
      "templateId": "uuid-3",
      "error": "Insufficient permissions"
    }
  ]
}
```

---

### Feature 2.3: Bulk Promote Rules
**Endpoint**: `POST /api/v1/rules/bulk-promote`

**Purpose**: Promote multiple rules across environments (dev → staging → prod)

**Request Body**:
```json
{
  "ruleIds": ["rule-uuid-1", "rule-uuid-2"],
  "fromEnvironment": "development",
  "toEnvironment": "staging",
  "includeVersionHistory": true,
  "executeTests": true,  // Run validation tests before promoting
  "notifyOnComplete": ["admin@company.com"]
}
```

**Response**:
```json
{
  "status": "success",
  "promoted": 2,
  "failed": 0,
  "promotionId": "promo-uuid",
  "results": [
    {
      "ruleId": "rule-uuid-1",
      "ruleName": "Calendar Override",
      "fromEnvironment": "development",
      "toEnvironment": "staging",
      "newVersion": 2,
      "status": "promoted"
    },
    // ...
  ],
  "timestamp": "2026-02-20T20:30:00Z"
}
```

**Validations**:
- Rules must exist in source environment
- All rules must be in publishable status
- Destination environment must exist
- User must have promotion rights
- Max 100 rules per batch

---

## Implementation Strategy

### Phase 2.1: Code Organization

**New File**: `backend/internal/handlers/bulk_operations_handler.go`
```
Structure:
├── BulkTemplateHandler struct
├── BulkCreateTemplates() → POST /api/v1/templates/bulk-create
├── BulkPublishTemplates() → POST /api/v1/templates/bulk-publish
├── BulkPromoteRules() → POST /api/v1/rules/bulk-promote
├── validateBulkRequest()
├── processBulkBatch()
└── generateBatchReport()
```

**Update**: `backend/cmd/semantic-rules-api/main.go`
- Register new bulk operation routes
- `router.HandleFunc("/api/v1/templates/bulk-create", bulkHandler.BulkCreateTemplates).Methods("POST")`
- `router.HandleFunc("/api/v1/templates/bulk-publish", bulkHandler.BulkPublishTemplates).Methods("POST")`
- `router.HandleFunc("/api/v1/rules/bulk-promote", bulkHandler.BulkPromoteRules).Methods("POST")`

**Update**: `backend/internal/handlers/templates_handler.go`
- Register BulkTemplateHandler with service
- Initialize in main.go

### Phase 2.2: Database Schema (Minimal Changes)

**New Table**: `edm.bulk_operations` (for tracking/auditing)
```sql
CREATE TABLE edm.bulk_operations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  operation_type VARCHAR(50),  -- 'bulk-create', 'bulk-publish', etc.
  status VARCHAR(20),          -- 'pending', 'running', 'completed', 'failed'
  request_count INT,
  success_count INT,
  failure_count INT,
  payload_size INT,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  created_by UUID NOT NULL,
  completed_at TIMESTAMP,
  error_summary TEXT,
  CONSTRAINT fk_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);

CREATE INDEX idx_bulk_ops_tenant ON edm.bulk_operations(tenant_id, created_at DESC);
```

### Phase 2.3: Core Implementation

**Algorithm for Bulk Create**:
```
1. Parse request JSON
2. Validate batch size (≤ 1000 templates)
3. Validate payload size (≤ 50MB)
4. For each template:
   a. Validate schema (name, description, etc.)
   b. Validate parameter schema JSON format
   c. Check business object exists
   d. Check name uniqueness within tenant
5. If continueOnError=false AND any error → return 400
6. Begin transaction
7. For each template:
   a. Insert into edm.rule_templates
   b. Log in bulk_operations table
   c. Catch individual errors if continueOnError=true
8. Commit transaction
9. Return result batch with created IDs
```

**Algorithm for Bulk Publish**:
```
1. Parse request JSON
2. Validate all template IDs exist
3. For each template:
   a. Verify status is 'draft'
   b. Check user has permission to publish
4. Begin transaction
5. For each template:
   a. UPDATE edm.rule_templates SET status='approved', updated_at=NOW()
   b. Log change in audit table
   c. Catch any errors
6. Commit
7. Return results with status updates
```

**Algorithm for Bulk Promote**:
```
1. Parse request JSON
2. Verify source/destination environments exist
3. Load all rules from source environment
4. For each rule:
   a. Run validation tests (if executeTests=true)
   b. Check rule is in promotable status
5. Begin transaction
6. For each rule:
   a. Copy rule to destination environment
   b. Increment version number
   c. Mark as promoted
   d. Log promotion metadata
7. Commit
8. Return promotion results
9. Send notifications (if requested)
```

### Phase 2.4: Error Handling & Resilience

**Error Categories**:
1. **Validation Errors** (400)
   - Invalid JSON format
   - Missing required fields
   - Business object doesn't exist
   - Payload too large

2. **Authorization Errors** (403)
   - Insufficient permissions
   - Tenant isolation violation
   - Cross-tenant operation attempt

3. **Conflict Errors** (409)
   - Template already exists
   - Duplicate names
   - Status transition invalid
   - Rules already in destination environment

4. **Transient Errors** (500)
   - Database connection issues
   - Timeout during bulk operation

**Retry Strategy**:
- Don't retry entire batch (user can retry)
- For individual failures with `continueOnError: true`:
  - Skip failed item
  - Continue processing remaining items
  - Report failures in response

**Logging**:
```go
log.Printf("[BULK_OP] BatchID=%s Operation=%s Status=%s Success=%d Failed=%d Duration=%dms",
  batchID, opType, status, successCount, failureCount, elapsed)
```

### Phase 2.5: Performance Optimization

**Batch Processing Options**:

**Option A: Sequential (Safe, Slower)**
```
for each template {
  INSERT into rule_templates
  INSERT into bulk_operations_log
}
// Time: ~100ms per template × 1000 = 100 seconds
```

**Option B: Buffered Inserts (Recommended)**
```
batch := []Template{}
for each template {
  batch = append(batch, template)
  if len(batch) == 100 {
    INSERT INTO rule_templates VALUES (...), (...), ...  // 100 rows at once
    batch = reset
  }
}
// Time: ~5-10ms per batch × 10 batches = 50-100ms total
```

**Option C: Prepared Statements (Most Efficient)**
```
stmt := db.Prepare("INSERT INTO rule_templates (...) VALUES (?)")
for each template {
  stmt.Exec(template.fields...)
}
stmt.Close()
// Time: ~1-2ms per template (amortized) = 1-2 seconds total
```

**Recommended**: Option B (buffered inserts) - good balance between performance and code complexity

---

## Testing Strategy

### Unit Tests

**Test 1**: BulkCreateTemplates - Success Path
```go
func TestBulkCreateTemplates_Success(t *testing.T) {
  // Setup: 10 valid templates
  // Call: BulkCreateTemplates
  // Assert: All 10 created, returned with IDs
}
```

**Test 2**: BulkCreateTemplates - Partial Failure (continueOnError: true)
```go
func TestBulkCreateTemplates_PartialFailure(t *testing.T) {
  // Setup: 10 templates, 2 with invalid schema
  // Call: BulkCreateTemplates with continueOnError=true
  // Assert: 8 created, 2 failed, results include error messages
}
```

**Test 3**: BulkCreateTemplates - Size Limits
```go
func TestBulkCreateTemplates_SizeLimits(t *testing.T) {
  // Setup: 1001 templates (exceeds limit of 1000)
  // Call: BulkCreateTemplates
  // Assert: Returns 400 Bad Request
}
```

**Test 4**: BulkPublishTemplates - Success
```go
func TestBulkPublishTemplates_Success(t *testing.T) {
  // Setup: 5 draft templates
  // Call: BulkPublishTemplates
  // Assert: All 5 now have status='approved'
}
```

**Test 5**: BulkPublishTemplates - Non-draft Templates
```go
func TestBulkPublishTemplates_NonDraft(t *testing.T) {
  // Setup: 5 templates, 2 already approved
  // Call: BulkPublishTemplates
  // Assert: Only 3 published, error for 2 already published
}
```

**Test 6**: BulkPromoteRules - Success
```go
func TestBulkPromoteRules_Success(t *testing.T) {
  // Setup: 3 development rules
  // Call: BulkPromoteRules (dev → staging)
  // Assert: Rules copied to staging with version incremented
}
```

**Test 7**: RLS Isolation
```go
func TestBulkOperations_RLSIsolation(t *testing.T) {
  // Setup: Tenant A creates templates, Tenant B calls bulk-create
  // Call: BulkCreateTemplates as Tenant B
  // Assert: Cannot see Tenant A's templates, cannot modify
}
```

### E2E Tests

**E2E Test 1**: Complete Workflow
```
1. Bulk create 50 templates
2. List and verify all 50 exist
3. Bulk publish 25 templates
4. Verify 25 have status='approved', 25 still 'draft'
5. Create rules from approved templates
6. Bulk promote 10 rules to staging
7. Verify rules exist in staging environment
```

**E2E Test 2**: Error Recovery
```
1. Attempt bulk create with 10 invalid templates
2. Verify none created
3. Retry with 10 valid templates
4. Verify all created successfully
```

---

## Implementation Phases

### Phase 2a: Setup (30 minutes)
- Create `bulk_operations_handler.go`
- Define request/response types
- Add database table for tracking
- Register routes in main.go

### Phase 2b: Bulk Create (60 minutes)
- Implement BulkCreateTemplates endpoint
- Add validation logic
- Implement buffered insert loop
- Add error handling

### Phase 2c: Bulk Publish (45 minutes)
- Implement BulkPublishTemplates endpoint
- Add status validation
- Implement transaction wrapper
- Add permission checks

### Phase 2d: Bulk Promote (60 minutes)
- Implement BulkPromoteRules endpoint
- Add cross-environment logic
- Implement notification system
- Add test execution logic

### Phase 2e: Testing (45 minutes)
- Unit tests for all 3 endpoints
- E2E test scenarios
- Load testing (simulate 1000-template imports)
- Error path testing

**Total Estimated Time**: 4 hours (includes testing)

---

## Database Migration

**File**: `backend/migrations/007_bulk_operations.sql`

```sql
-- Create bulk operations tracking table
CREATE TABLE IF NOT EXISTS edm.bulk_operations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  operation_type VARCHAR(50) NOT NULL,
  status VARCHAR(20) NOT NULL DEFAULT 'pending',
  request_count INT NOT NULL DEFAULT 0,
  success_count INT NOT NULL DEFAULT 0,
  failure_count INT NOT NULL DEFAULT 0,
  payload_size INT,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  created_by UUID NOT NULL,
  completed_at TIMESTAMP,
  error_summary TEXT,
  
  CONSTRAINT ck_operation_type CHECK (operation_type IN ('bulk-create', 'bulk-publish', 'bulk-promote')),
  CONSTRAINT ck_status CHECK (status IN ('pending', 'running', 'completed', 'failed', 'partial'))
);

-- Index for querying batch status
CREATE INDEX IF NOT EXISTS idx_bulk_ops_tenant
  ON edm.bulk_operations(tenant_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_bulk_ops_status
  ON edm.bulk_operations(status) WHERE status IN ('pending', 'running');
```

---

## Success Criteria

- [ ] All 3 endpoints implemented and responding
- [ ] Bulk create supports up to 1000 templates
- [ ] Bulk publish transitions template statuses correctly
- [ ] Bulk promote creates rule copies with incremented versions
- [ ] Error handling returns appropriate HTTP status codes
- [ ] Transactional integrity maintained (all-or-nothing or partial)
- [ ] RLS policies enforced (cross-tenant safety)
- [ ] Database tracking table populated correctly
- [ ] Unit tests: ≥90% code coverage
- [ ] E2E tests: All critical paths tested
- [ ] Performance: 1000 templates bulk-create ≤ 5 seconds
- [ ] Documentation: API spec and usage examples

---

## Related Capabilities (Future)

### Feature 2.1+: Async Bulk Operations
- Add job queue for long-running operations
- Return job ID immediately, poll for status
- Support webhook callbacks on completion

### Feature 2.2+: Bulk Operations Scheduling
- Schedule bulk operations for off-peak hours
- Recurring bulk imports from external sources
- Batch operations with approval workflows

### Feature 2.3+: Bulk Export/Import
- Export templates to JSON format
- Import from OpenAPI/Swagger specs
- Support YAML format for GitOps workflows

---

## Rollback Strategy

If issues arise:
1. Keep old semantic-rules-api binary
2. Revert to previous version
3. Skip Feature 2 migrations if needed
4. Roll back to code before Feature 2
5. (Feature 1 remains unaffected)

---

## Next Steps

1. ✅ Create this plan (DONE)
2. ⏳ Create `bulk_operations_handler.go`
3. ⏳ Implement BulkCreateTemplates
4. ⏳ Implement BulkPublishTemplates  
5. ⏳ Implement BulkPromoteRules
6. ⏳ Add database migration
7. ⏳ Write unit tests
8. ⏳ Run E2E tests
9. ⏳ Performance testing
10. ⏳ Create documentation

---

**Plan Version**: 1.0  
**Status**: 🚀 READY TO IMPLEMENT  
**Estimated Duration**: 3-4 hours  
**Start Time**: February 20, 2026 20:30 UTC
