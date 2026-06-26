# Validation Rules System - Architecture Diagram

## System Architecture Overview

```
┌────────────────────────────────────────────────────────────────────┐
│                         VALIDATION RULES SYSTEM                     │
├────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                        FRONTEND (React)                      │   │
│  ├─────────────────────────────────────────────────────────────┤   │
│  │                                                              │   │
│  │  http://localhost:5173/core/validation-rules               │   │
│  │                                                              │   │
│  │  ┌──────────────────────────────────────────────────────┐  │   │
│  │  │        ValidationRulesPage.tsx                        │  │   │
│  │  ├──────────────────────────────────────────────────────┤  │   │
│  │  │                                                        │  │   │
│  │  │  ┌─ List View ────────────────────────────────────┐  │  │   │
│  │  │  │ • All rules (filter by type, severity, etc.)   │  │  │   │
│  │  │  │ • Search functionality                         │  │  │   │
│  │  │  │ • Create/Edit/Delete buttons                  │  │  │   │
│  │  │  └────────────────────────────────────────────────┘  │  │   │
│  │  │                                                        │  │   │
│  │  │  ┌─ Create/Edit Dialog ──────────────────────────┐  │  │   │
│  │  │  │ • Tabs: Rule Builder | JSON Editor            │  │  │   │
│  │  │  │ • Rule Builder:                                │  │  │   │
│  │  │  │   - Form for selected rule type               │  │  │   │
│  │  │  │   - Type-specific fields                      │  │  │   │
│  │  │  │   - Validation in real-time                  │  │  │   │
│  │  │  │ • JSON Editor:                                │  │  │   │
│  │  │  │   - JSON preview                             │  │  │   │
│  │  │  │   - Direct JSON editing                      │  │  │   │
│  │  │  └────────────────────────────────────────────────┘  │  │   │
│  │  │                                                        │  │   │
│  │  └──────────────────────────────────────────────────────┘  │   │
│  │                    ↓ API Calls                             │   │
│  │              (tenant_id, auth headers)                    │   │
│  │                                                              │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                      │
│                              ↓ HTTPS                               │
│                     REST API Calls (JSON)                          │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                     BACKEND (Go/Chi)                         │   │
│  ├─────────────────────────────────────────────────────────────┤   │
│  │                                                              │   │
│  │  http://localhost:29080/api/validation-rules               │   │
│  │                                                              │   │
│  │  ┌─────────────────────────────────────────────────────┐   │   │
│  │  │         validation_rules_routes.go (600 lines)      │   │   │
│  │  ├─────────────────────────────────────────────────────┤   │   │
│  │  │                                                      │   │   │
│  │  │  ┌─ HTTP Handlers (8 total) ─────────────────────┐ │   │   │
│  │  │  │                                                 │ │   │   │
│  │  │  │  handleListValidationRules                     │ │   │   │
│  │  │  │  ├─ Query filters: type, severity, entity     │ │   │   │
│  │  │  │  ├─ Pagination support                        │ │   │   │
│  │  │  │  └─ Returns: []ValidationRule                 │ │   │   │
│  │  │  │                                                 │ │   │   │
│  │  │  │  handleGetValidationRule                       │ │   │   │
│  │  │  │  ├─ Param: rule_id (UUID)                     │ │   │   │
│  │  │  │  └─ Returns: Single rule or 404               │ │   │   │
│  │  │  │                                                 │ │   │   │
│  │  │  │  handleCreateValidationRule                    │ │   │   │
│  │  │  │  ├─ Validation: Required fields, enum values  │ │   │   │
│  │  │  │  ├─ Duplicate check: (tenant, name) unique    │ │   │   │
│  │  │  │  └─ Audit: CREATE recorded                    │ │   │   │
│  │  │  │                                                 │ │   │   │
│  │  │  │  handleUpdateValidationRule                    │ │   │   │
│  │  │  │  ├─ Partial updates allowed                   │ │   │   │
│  │  │  │  ├─ Audit: old_values, new_values recorded   │ │   │   │
│  │  │  │  └─ Returns: Updated rule                     │ │   │   │
│  │  │  │                                                 │ │   │   │
│  │  │  │  handleDeleteValidationRule                    │ │   │   │
│  │  │  │  ├─ Soft? No - Hard delete                    │ │   │   │
│  │  │  │  ├─ Cascade: Audit records CASCADE deleted    │ │   │   │
│  │  │  │  └─ Audit: DELETE recorded                    │ │   │   │
│  │  │  │                                                 │ │   │   │
│  │  │  │  handleExecuteValidationRule                   │ │   │   │
│  │  │  │  ├─ Input: rule_id, data object              │ │   │   │
│  │  │  │  └─ Output: ExecutionResult                   │ │   │   │
│  │  │  │                                                 │ │   │   │
│  │  │  │  handleExecuteValidationRulesBatch             │ │   │   │
│  │  │  │  ├─ Input: rule_ids array                     │ │   │   │
│  │  │  │  └─ Output: []ExecutionResult                 │ │   │   │
│  │  │  │                                                 │ │   │   │
│  │  │  │  handleGetValidationRuleAudit                  │ │   │   │
│  │  │  │  ├─ Param: rule_id (UUID)                     │ │   │   │
│  │  │  │  └─ Returns: Audit history ordered DESC       │ │   │   │
│  │  │  │                                                 │ │   │   │
│  │  │  └─────────────────────────────────────────────────┘ │   │   │
│  │  │              ↓ Internal Call                          │   │   │
│  │  │        All handlers call engine.Execute()            │   │   │
│  │  │              ↓                                        │   │   │
│  │  │                                                      │   │   │
│  │  │  ┌─ Tenant Scoping (Mandatory) ──────────────────┐ │   │   │
│  │  │  │ • tenant_id from query params (required)      │ │   │   │
│  │  │  │ • X-Tenant-ID header validation              │ │   │   │
│  │  │  │ • All queries filtered WHERE tenant_id = ?    │ │   │   │
│  │  │  │ • No cross-tenant access possible            │ │   │   │
│  │  │  └────────────────────────────────────────────────┘ │   │   │
│  │  │              ↓ Database Query                        │   │   │
│  │  │                                                      │   │   │
│  │  └─────────────────────────────────────────────────────┘   │   │
│  │                    ↓                                        │   │
│  │                                                              │   │
│  │  ┌─────────────────────────────────────────────────────┐   │   │
│  │  │          engine.go (Rule Execution)                 │   │   │
│  │  ├─────────────────────────────────────────────────────┤   │   │
│  │  │                                                      │   │   │
│  │  │  ValidationEngine                                  │   │   │
│  │  │  └─ Execute(ctx ExecutionContext)                 │   │   │
│  │  │     └─ Switch on rule.rule_type:                 │   │   │
│  │  │                                                      │   │   │
│  │  │     ┌─ business_logic                             │   │   │
│  │  │     │  ├─ Field: Get value from data             │   │   │
│  │  │     │  ├─ Operator: >, <, >=, <=, ==, !=        │   │   │
│  │  │     │  ├─ Type conversion: int/float/string      │   │   │
│  │  │     │  └─ Return: ExecutionResult                │   │   │
│  │  │     │                                              │   │   │
│  │  │     ├─ field_format                              │   │   │
│  │  │     │  ├─ Regex pattern from condition           │   │   │
│  │  │     │  ├─ Match against data field              │   │   │
│  │  │     │  ├─ Error handling for invalid patterns   │   │   │
│  │  │     │  └─ Return: Pass/Fail + message            │   │   │
│  │  │     │                                              │   │   │
│  │  │     ├─ cardinality                               │   │   │
│  │  │     │  ├─ Numeric threshold check               │   │   │
│  │  │     │  ├─ Get operator & value                  │   │   │
│  │  │     │  ├─ Compare with data                     │   │   │
│  │  │     │  └─ Return: Result                         │   │   │
│  │  │     │                                              │   │   │
│  │  │     ├─ uniqueness                                │   │   │
│  │  │     │  ├─ Check field uniqueness               │   │   │
│  │  │     │  ├─ Placeholder: DB integration needed   │   │   │
│  │  │     │  └─ Return: Unique or duplicate           │   │   │
│  │  │     │                                              │   │   │
│  │  │     └─ referential_integrity                     │   │   │
│  │  │        ├─ Validate FK relationship              │   │   │
│  │  │        ├─ Placeholder: DB integration needed   │   │   │
│  │  │        └─ Return: Valid or broken               │   │   │
│  │  │                                                      │   │   │
│  │  │  ExecutionResult                                   │   │   │
│  │  │  ├─ RuleID: string (UUID)                         │   │   │
│  │  │  ├─ Passed: bool (true/false)                     │   │   │
│  │  │  ├─ Message: string (descriptive)                │   │   │
│  │  │  └─ Details: map[string]interface{}              │   │   │
│  │  │                                                      │   │   │
│  │  └─────────────────────────────────────────────────────┘   │   │
│  │                                                              │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                      │
│                              ↓ SQL                                 │
│                      Parameterized Queries                         │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                   DATABASE (PostgreSQL)                      │   │
│  ├─────────────────────────────────────────────────────────────┤   │
│  │                                                              │   │
│  │  localhost:5432/alpha                                       │   │
│  │                                                              │   │
│  │  ┌──────────────────────────────────────────────────────┐   │   │
│  │  │    catalog_validation_rules (Main Table)             │   │   │
│  │  ├──────────────────────────────────────────────────────┤   │   │
│  │  │                                                        │   │   │
│  │  │  PK: id (UUID)                                       │   │   │
│  │  │  FK: tenant_id (UUID)                                │   │   │
│  │  │  ─────────────────────────────────────             │   │   │
│  │  │  rule_name (VARCHAR 255)                           │   │   │
│  │  │  rule_type (VARCHAR 50)     [CHECK: valid types]   │   │   │
│  │  │  target_entity (VARCHAR 255)                       │   │   │
│  │  │  description (TEXT)                                 │   │   │
│  │  │  condition_json (JSONB)                            │   │   │
│  │  │  severity (VARCHAR 20)      [CHECK: error|warn]    │   │   │
│  │  │  is_active (BOOLEAN)                                │   │   │
│  │  │  created_by (UUID)                                  │   │   │
│  │  │  created_at (TIMESTAMP)                             │   │   │
│  │  │  updated_at (TIMESTAMP)                             │   │   │
│  │  │  ─────────────────────────────────────             │   │   │
│  │  │  UNIQUE(tenant_id, rule_name)                      │   │   │
│  │  │  CASCADE DELETE → audit table                      │   │   │
│  │  │                                                        │   │   │
│  │  │  Indexes (7 total):                                 │   │   │
│  │  │  ├─ tenant_id (B-tree)                             │   │   │
│  │  │  ├─ rule_type (B-tree)                             │   │   │
│  │  │  ├─ target_entity (B-tree)                         │   │   │
│  │  │  ├─ severity (B-tree)                              │   │   │
│  │  │  ├─ is_active (B-tree)                             │   │   │
│  │  │  ├─ condition_json (GIN)                           │   │   │
│  │  │  └─ created_at DESC (B-tree)                       │   │   │
│  │  │                                                        │   │   │
│  │  └──────────────────────────────────────────────────────┘   │   │
│  │                          │                                    │   │
│  │                          ├─→ CASCADE DELETE                   │   │
│  │                          │                                    │   │
│  │  ┌──────────────────────────────────────────────────────┐   │   │
│  │  │    catalog_validation_rules_audit (Audit Table)     │   │   │
│  │  ├──────────────────────────────────────────────────────┤   │   │
│  │  │                                                        │   │   │
│  │  │  PK: id (UUID)                                       │   │   │
│  │  │  FK: rule_id (UUID)  → CASCADE DELETE               │   │   │
│  │  │  FK: tenant_id (UUID)                                │   │   │
│  │  │  ─────────────────────────────────────             │   │   │
│  │  │  action (VARCHAR 20) [CHECK: CREATE|UPDATE|DELETE]  │   │   │
│  │  │  old_values (JSONB)  [nullable]                    │   │   │
│  │  │  new_values (JSONB)  [nullable]                    │   │   │
│  │  │  changed_by (UUID)                                  │   │   │
│  │  │  changed_at (TIMESTAMP)                             │   │   │
│  │  │  ─────────────────────────────────────             │   │   │
│  │  │  Note: Immutable - no updates, only inserts        │   │   │
│  │  │        Retention: Keep indefinitely (or archive)   │   │   │
│  │  │                                                        │   │   │
│  │  │  Indexes:                                            │   │   │
│  │  │  ├─ rule_id (B-tree)                               │   │   │
│  │  │  ├─ tenant_id (B-tree)                             │   │   │
│  │  │  └─ changed_at DESC (B-tree)                       │   │   │
│  │  │                                                        │   │   │
│  │  └──────────────────────────────────────────────────────┘   │   │
│  │                                                              │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                      │
└────────────────────────────────────────────────────────────────────┘
```

---

## Data Flow Diagrams

### Create Rule Flow
```
┌─ User Creates Rule ─────────────────────┐
│                                         │
│  ValidationRulesPage.tsx                │
│  └─ handleCreateRule()                  │
│     └─ POST /api/validation-rules       │
│                                         │
├─ Backend Processing ───────────────────┤
│                                         │
│  validation_rules_routes.go             │
│  └─ handleCreateValidationRule()        │
│     ├─ Validate tenant_id (required)   │
│     ├─ Validate request body            │
│     │  ├─ Required fields present?     │
│     │  ├─ Valid rule_type?             │
│     │  └─ Valid severity?              │
│     ├─ Check duplicate                  │
│     │  └─ WHERE tenant_id=? AND name=? │
│     └─ INSERT into catalog_validation_rules
│        ├─ Auto-insert: id, created_at  │
│        └─ Auto-record: created_by       │
│                                         │
├─ Database Processing ───────────────────┤
│                                         │
│  Trigger: INSERT audit record           │
│  └─ catalog_validation_rules_audit      │
│     ├─ action: 'CREATE'                │
│     ├─ old_values: null                │
│     ├─ new_values: {rule_json}         │
│     └─ changed_at: now()               │
│                                         │
└─ Response ─────────────────────────────┘
   ├─ HTTP 201 Created
   └─ Return: New rule object
```

### Update Rule Flow
```
┌─ User Updates Rule ──────────────────────┐
│                                          │
│  ValidationRulesPage.tsx                 │
│  └─ handleUpdateRule(id, changes)       │
│     └─ PATCH /api/validation-rules/{id} │
│                                          │
├─ Backend Processing ────────────────────┤
│                                          │
│  validation_rules_routes.go              │
│  └─ handleUpdateValidationRule()         │
│     ├─ Validate tenant_id (required)    │
│     ├─ GET current rule (for audit)     │
│     │  └─ SELECT * FROM WHERE id=?     │
│     ├─ Validate changes                  │
│     │  ├─ Valid rule_type? (if provided)│
│     │  └─ Valid severity? (if provided) │
│     └─ UPDATE catalog_validation_rules   │
│        ├─ Set provided fields            │
│        └─ Auto-update: updated_at        │
│                                          │
├─ Database Processing ────────────────────┤
│                                          │
│  Trigger: INSERT audit record            │
│  └─ catalog_validation_rules_audit       │
│     ├─ action: 'UPDATE'                 │
│     ├─ old_values: {previous_state}     │
│     ├─ new_values: {updated_state}      │
│     └─ changed_at: now()                │
│                                          │
└─ Response ─────────────────────────────┘
   ├─ HTTP 200 OK
   └─ Return: Updated rule object
```

### Execute Rule Flow
```
┌─ User Executes Rule ──────────────────────┐
│                                           │
│  ValidationRulesPage.tsx                  │
│  └─ handleExecuteRule(ruleId, data)      │
│     └─ POST /api/validation-rules/{id}/execute
│                                           │
├─ Backend Processing ────────────────────┤
│                                           │
│  validation_rules_routes.go               │
│  └─ handleExecuteValidationRule()         │
│     ├─ GET rule by id                    │
│     │  └─ SELECT * FROM WHERE id=?      │
│     ├─ Call engine.Execute()              │
│     │  └─ validation/engine.go            │
│     │     └─ Switch on rule.rule_type    │
│     │        ├─ business_logic            │
│     │        ├─ field_format              │
│     │        ├─ cardinality               │
│     │        ├─ uniqueness                │
│     │        └─ referential_integrity    │
│     └─ Return: ExecutionResult            │
│                                           │
└─ Response ─────────────────────────────┘
   ├─ HTTP 200 OK
   └─ Return: {
        "rule_id": "...",
        "passed": true/false,
        "message": "...",
        "details": {...}
      }
```

### Delete Rule Flow
```
┌─ User Deletes Rule ───────────────────────┐
│                                           │
│  ValidationRulesPage.tsx                  │
│  └─ handleDeleteRule(ruleId)             │
│     └─ DELETE /api/validation-rules/{id} │
│                                           │
├─ Backend Processing ────────────────────┤
│                                           │
│  validation_rules_routes.go               │
│  └─ handleDeleteValidationRule()          │
│     ├─ Validate tenant_id (required)     │
│     ├─ GET current rule (for audit)      │
│     │  └─ SELECT * FROM WHERE id=?      │
│     ├─ DELETE from catalog_validation_rules
│     │  └─ WHERE id=? AND tenant_id=?    │
│     └─ Verify deletion                    │
│        └─ rowsAffected == 1?              │
│                                           │
├─ Database Processing ────────────────────┤
│                                           │
│  Trigger 1: INSERT audit record           │
│  └─ catalog_validation_rules_audit        │
│     ├─ action: 'DELETE'                  │
│     ├─ old_values: {deleted_state}       │
│     ├─ new_values: null                  │
│     └─ changed_at: now()                 │
│                                           │
│  Trigger 2: CASCADE DELETE                │
│  └─ catalog_validation_rules_audit        │
│     └─ CASCADE deletes all related        │
│        audit records for this rule       │
│                                           │
└─ Response ─────────────────────────────┘
   ├─ HTTP 204 No Content (or 200 OK)
   └─ No body
```

---

## Tenant Scoping Architecture

```
┌──────────────────────────────────────────────────────────────┐
│              MULTI-TENANT ISOLATION LAYER                    │
├──────────────────────────────────────────────────────────────┤
│                                                               │
│  Frontend Storage (localStorage)                             │
│  ├─ selected_tenant → { id, display_name, ... }             │
│  ├─ selected_product → { id, alpha_product, ... }           │
│  └─ selected_datasource → { id, source_name, ... }          │
│                                                               │
│  ↓ setupTenantFetch.ts intercepts all /api/* calls          │
│                                                               │
│  Request Interceptor (frontend/src/setupTenantFetch.ts)     │
│  ├─ Check: Is tenant_id in localStorage?                   │
│  ├─ If NO: Block request, show "Select tenant" warning     │
│  ├─ If YES: Add to request:                                │
│  │  ├─ Query param: ?tenant_id={selected_tenant.id}        │
│  │  ├─ Query param: ?datasource_id={selected_datasource.id}│
│  │  ├─ Header: X-Tenant-ID: {selected_tenant.id}           │
│  │  └─ Header: X-Tenant-Datasource-ID: {...}              │
│  │                                                           │
│  │  Request to: /api/validation-rules?tenant_id=XXX        │
│  │  Headers: X-Tenant-ID: XXX, X-Tenant-Datasource-ID: YYY│
│                                                               │
│  ↓ Backend Validation (validation_rules_routes.go)          │
│                                                               │
│  All Handlers Enforce:                                       │
│  ├─ Query param tenant_id is REQUIRED                      │
│  ├─ If missing: HTTP 400 Bad Request                        │
│  │                                                           │
│  ├─ All SQL queries include:                                │
│  │  WHERE tenant_id = $1                                    │
│  │                                                           │
│  └─ Result: Only rules for selected tenant returned         │
│                                                               │
│  ↓ Database Level (PostgreSQL)                              │
│                                                               │
│  Foreign Key Relationship:                                   │
│  ├─ catalog_validation_rules.tenant_id → catalog_tenants.id│
│  ├─ Not enforced in migration (tenants table elsewhere)    │
│  └─ Enforced via application logic                          │
│                                                               │
│  Example Query:                                              │
│  SELECT * FROM catalog_validation_rules                     │
│  WHERE tenant_id = '910638ba-a459-4a3f-bb2d-78391b0595f6'  │
│  AND is_active = true                                       │
│  ORDER BY created_at DESC                                   │
│                                                               │
└──────────────────────────────────────────────────────────────┘
```

---

## Error Handling Flow

```
┌─────────────────────────────────────────────────────────┐
│           REQUEST ERROR HANDLING CHAIN                   │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  Request arrives at handler                             │
│  └─ GET /api/validation-rules?tenant_id=...            │
│     └─ POST /api/validation-rules (with JSON body)      │
│                                                          │
├─ Step 1: Parse Query Parameters ──────────────────────┤
│                                                          │
│  tenant_id := r.URL.Query().Get("tenant_id")           │
│  if tenant_id == "" {                                   │
│    return writeJSONError(w, http.StatusBadRequest,      │
│      "missing_tenant", "tenant_id required")           │
│  }                                                       │
│  HTTP 400 response ← Client                             │
│                                                          │
├─ Step 2: Parse Request Body ─────────────────────────┤
│                                                          │
│  var req ValidationRuleRequest                          │
│  if err := json.NewDecoder(r.Body).Decode(&req) {      │
│    return writeJSONError(w, http.StatusBadRequest,      │
│      "decode_error", "Invalid JSON")                   │
│  }                                                       │
│  HTTP 400 response ← Client                             │
│                                                          │
├─ Step 3: Validate Required Fields ────────────────────┤
│                                                          │
│  required := []string{"rule_name", "rule_type", ...}   │
│  if req.RuleName == "" || req.RuleType == "" {         │
│    return writeJSONError(w, http.StatusBadRequest,      │
│      "validation_error", "Required fields missing")    │
│  }                                                       │
│  HTTP 400 response ← Client                             │
│                                                          │
├─ Step 4: Validate Enum Values ──────────────────────┤
│                                                          │
│  validTypes := []string{"business_logic", ...}         │
│  if !contains(validTypes, req.RuleType) {              │
│    return writeJSONError(w, http.StatusBadRequest,      │
│      "validation_error", "Invalid rule_type")          │
│  }                                                       │
│  HTTP 400 response ← Client                             │
│                                                          │
├─ Step 5: Database Query Errors ──────────────────────┤
│                                                          │
│  rows, err := db.Query(query, ...)                      │
│  if err != nil {                                        │
│    if pqErr, ok := err.(*pq.Error); ok {               │
│      if pqErr.Code == "23505" {  // UNIQUE violation   │
│        return writeJSONError(w, http.StatusConflict,    │
│          "duplicate_rule", "Rule already exists")       │
│      }                                                   │
│    }                                                     │
│    return writeJSONError(w, http.StatusInternalServerError,
│      "query_error", "Database error")                  │
│  }                                                       │
│  HTTP 409 or 500 response ← Client                      │
│                                                          │
├─ Step 6: Not Found Errors ──────────────────────────┤
│                                                          │
│  row := db.QueryRow(query, ruleID, tenantID)           │
│  if err == sql.ErrNoRows {                             │
│    return writeJSONError(w, http.StatusNotFound,        │
│      "not_found", "Rule not found")                    │
│  }                                                       │
│  HTTP 404 response ← Client                             │
│                                                          │
├─ Step 7: Success Response ───────────────────────────┤
│                                                          │
│  return writeJSON(w, http.StatusOK, rule)               │
│  HTTP 200 response ← Client {rule object}               │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

---

## Performance & Indexing Strategy

```
┌──────────────────────────────────────────────────────────┐
│           QUERY OPTIMIZATION LAYERS                       │
├──────────────────────────────────────────────────────────┤
│                                                           │
│  Layer 1: Index Selection (Query Planner)               │
│  ────────────────────────────────────────────          │
│                                                           │
│  Common Queries & Their Indexes:                        │
│                                                           │
│  Query: List rules for tenant                           │
│  ├─ SQL: SELECT * FROM catalog_validation_rules        │
│  │       WHERE tenant_id = $1                           │
│  │       ORDER BY created_at DESC                       │
│  │       LIMIT 50 OFFSET 0                              │
│  └─ Index Used: (tenant_id, created_at DESC)           │
│     └─ Estimated Time: 5-20ms for 1000 rules          │
│                                                           │
│  Query: Filter by type                                  │
│  ├─ SQL: SELECT * FROM catalog_validation_rules        │
│  │       WHERE tenant_id = $1 AND rule_type = $2       │
│  └─ Index Used: (tenant_id, rule_type)                 │
│     └─ Estimated Time: 2-10ms                          │
│                                                           │
│  Query: Filter by active status                         │
│  ├─ SQL: SELECT * FROM catalog_validation_rules        │
│  │       WHERE tenant_id = $1 AND is_active = true     │
│  └─ Index Used: (tenant_id, is_active)                 │
│     └─ Estimated Time: 1-5ms                           │
│                                                           │
│  Query: Complex JSONB search                            │
│  ├─ SQL: SELECT * FROM catalog_validation_rules        │
│  │       WHERE tenant_id = $1                           │
│  │       AND condition_json @> '{"field":"email"}'     │
│  └─ Index Used: condition_json GIN                      │
│     └─ Estimated Time: 10-50ms                         │
│                                                           │
│  Query: Get audit history                               │
│  ├─ SQL: SELECT * FROM catalog_validation_rules_audit  │
│  │       WHERE rule_id = $1                             │
│  │       ORDER BY changed_at DESC                       │
│  └─ Index Used: (rule_id, changed_at DESC)             │
│     └─ Estimated Time: 5-20ms                          │
│                                                           │
├─ Layer 2: Query Execution ──────────────────────────────┤
│                                                           │
│  Prepared Statements (Preventing SQL Injection)         │
│  ├─ All queries use parameterized format: $1, $2, $3   │
│  ├─ Parameters bound at execution time                 │
│  └─ Cache-friendly for query planner                   │
│                                                           │
│  Connection Pooling (Future Optimization)               │
│  ├─ Reuse database connections                         │
│  ├─ Reduce connection overhead                         │
│  └─ Improve throughput                                  │
│                                                           │
├─ Layer 3: Caching Strategy (Future) ────────────────────┤
│                                                           │
│  Frontend Caching:                                       │
│  ├─ Cache rules list (5 min TTL)                       │
│  ├─ Invalidate on create/update/delete                │
│  └─ Improves perceived performance                      │
│                                                           │
│  Backend Caching:                                        │
│  ├─ Cache single rule by ID (10 min TTL)              │
│  ├─ Invalidate on update/delete                       │
│  └─ Reduces database load                              │
│                                                           │
└──────────────────────────────────────────────────────────┘
```

---

## Security Architecture

```
┌────────────────────────────────────────────────────────────┐
│           SECURITY LAYERS & DEFENSES                       │
├────────────────────────────────────────────────────────────┤
│                                                             │
├─ Layer 1: Authentication ──────────────────────────────────┤
│                                                             │
│  Tenant Context Requirement:                              │
│  ├─ X-Tenant-ID header required on all requests          │
│  ├─ tenant_id query parameter required                   │
│  ├─ Validated before any processing                      │
│  └─ Missing/invalid: HTTP 400 Bad Request                │
│                                                             │
├─ Layer 2: Authorization ───────────────────────────────────┤
│                                                             │
│  Tenant Isolation:                                         │
│  ├─ WHERE tenant_id = $1 on every query                  │
│  ├─ Users can only see/modify their tenant's rules       │
│  ├─ No cross-tenant access possible                      │
│  └─ Enforced at SQL layer (defense in depth)             │
│                                                             │
├─ Layer 3: Input Validation ────────────────────────────────┤
│                                                             │
│  Type Whitelist Validation:                               │
│  ├─ rule_type: "business_logic" | "field_format" | ...  │
│  ├─ severity: "error" | "warning" | "info"              │
│  └─ Checked: Backend + Database (CHECK constraint)       │
│                                                             │
│  Required Field Validation:                               │
│  ├─ rule_name: NOT NULL + VARCHAR 255                    │
│  ├─ rule_type: NOT NULL + CHECK                          │
│  ├─ target_entity: NOT NULL + VARCHAR 255                │
│  ├─ condition_json: NOT NULL + JSONB                     │
│  └─ severity: NOT NULL + CHECK                           │
│                                                             │
│  Duplicate Prevention:                                     │
│  ├─ UNIQUE(tenant_id, rule_name) constraint             │
│  ├─ Prevents duplicate rule names per tenant            │
│  ├─ Catches at database level                           │
│  └─ Returns HTTP 409 Conflict to client                 │
│                                                             │
├─ Layer 4: SQL Injection Prevention ────────────────────────┤
│                                                             │
│  Parameterized Queries:                                    │
│  ├─ ALL queries use prepared statements                 │
│  ├─ Query: SELECT * FROM table WHERE id = $1            │
│  ├─ Parameter: id=123 (bound at execution)              │
│  └─ NO string concatenation in SQL                      │
│                                                             │
│  Regex Validation (for field_format rules):              │
│  ├─ Validate pattern format before use                  │
│  ├─ Catch invalid regex: Try to compile                │
│  ├─ Error handling: Return friendly error               │
│  └─ Prevent DoS via catastrophic backtracking          │
│                                                             │
├─ Layer 5: Data Integrity ──────────────────────────────────┤
│                                                             │
│  Foreign Key Constraints:                                  │
│  ├─ tenant_id → catalog_tenants (if FK enforced)        │
│  ├─ CASCADE DELETE on audit table                       │
│  └─ Maintains referential integrity                      │
│                                                             │
│  Audit Trail:                                             │
│  ├─ All changes recorded (CREATE/UPDATE/DELETE)         │
│  ├─ Old values preserved                                 │
│  ├─ Changed by: user UUID tracked                        │
│  ├─ Changed at: timestamp recorded                       │
│  └─ Immutable: No updates to audit records               │
│                                                             │
│  Constraint Enforcement:                                   │
│  ├─ CHECK constraint: rule_type in valid_types         │
│  ├─ CHECK constraint: severity in valid_values          │
│  ├─ Enforced at database level                          │
│  └─ Bad data prevented from entering DB                 │
│                                                             │
├─ Layer 6: Error Handling ──────────────────────────────────┤
│                                                             │
│  No Sensitive Data Exposure:                              │
│  ├─ Error messages: "Rule not found" (generic)           │
│  ├─ Not: "Rule 123 in tenant 456 not found"            │
│  ├─ Stack traces: Logged server-side only               │
│  └─ Prevents information disclosure                      │
│                                                             │
│  Proper HTTP Status Codes:                                │
│  ├─ 400: Bad Request (validation errors)                 │
│  ├─ 401: Unauthorized (auth failed)                      │
│  ├─ 403: Forbidden (not allowed)                         │
│  ├─ 404: Not Found (resource doesn't exist)             │
│  ├─ 409: Conflict (duplicate)                            │
│  └─ 500: Server Error (internal)                         │
│                                                             │
└────────────────────────────────────────────────────────────┘
```

---

This comprehensive architecture diagram provides a complete visual reference for understanding how all components of the Validation Rules system interact, from the frontend UI through the backend API to the database layer.
