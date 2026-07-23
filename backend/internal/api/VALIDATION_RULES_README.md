# Backend Validation Rules Implementation

## Overview

The Validation Rules system provides comprehensive data quality and business logic validation capabilities through REST API endpoints. The system is fully tenant-scoped, audit-enabled, and includes a pluggable rule execution engine.

## Database Schema

### Tables

#### `catalog_validation_rules`
Stores validation rule definitions with tenant scoping:

```sql
CREATE TABLE catalog_validation_rules (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    rule_name VARCHAR(255) NOT NULL,
    rule_type VARCHAR(50) NOT NULL,  -- field_format, cardinality, uniqueness, referential_integrity, business_logic
    description TEXT,
    target_entity VARCHAR(255) NOT NULL,
    condition_json JSONB NOT NULL,
    severity VARCHAR(20) NOT NULL,  -- error, warning, info
    is_active BOOLEAN DEFAULT true,
    created_by UUID,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    UNIQUE(tenant_id, rule_name)
);
```

**Key Constraints:**
- `tenant_id`: Multi-tenant isolation
- `UNIQUE(tenant_id, rule_name)`: Prevent duplicate rule names per tenant
- `rule_type` CHECK constraint: Valid types only
- `severity` CHECK constraint: Valid severity levels only

**Indexes:**
- `idx_validation_rules_tenant`: Fast tenant lookups
- `idx_validation_rules_type`: Filter by rule type
- `idx_validation_rules_entity`: Filter by target entity
- `idx_validation_rules_severity`: Filter by severity
- `idx_validation_rules_active`: Filter active rules
- `idx_validation_rules_condition`: JSONB query optimization
- `idx_validation_rules_created`: Time-based filtering

#### `catalog_validation_rules_audit`
Tracks all changes to validation rules:

```sql
CREATE TABLE catalog_validation_rules_audit (
    id UUID PRIMARY KEY,
    rule_id UUID NOT NULL REFERENCES catalog_validation_rules(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    action VARCHAR(20) NOT NULL,  -- CREATE, UPDATE, DELETE
    old_values JSONB,
    new_values JSONB,
    changed_by UUID,
    changed_at TIMESTAMP
);
```

**Purpose:**
- Complete audit trail of all rule changes
- Track who made changes and when
- Facilitate rollback and compliance reporting

## API Endpoints

### List Validation Rules
**Endpoint:** `GET /api/validation-rules`

**Query Parameters:**
- `tenant_id` (required): UUID of the tenant
- `rule_type` (optional): Filter by rule type
- `severity` (optional): Filter by severity (error, warning, info)
- `target_entity` (optional): Filter by target entity name
- `is_active` (optional): Filter by active status (true/false)

**Response:**
```json
[
  {
    "id": "uuid",
    "tenant_id": "uuid",
    "rule_name": "Order Total Must Be Positive",
    "rule_type": "business_logic",
    "description": "Order total must be greater than 0",
    "target_entity": "Order",
    "condition_json": {
      "field": "total",
      "operator": ">",
      "value": 0
    },
    "severity": "error",
    "is_active": true,
    "created_by": "user-uuid",
    "created_at": "2025-10-19T10:00:00Z",
    "updated_at": "2025-10-19T10:00:00Z"
  }
]
```

**Example:**
```bash
curl "http://localhost:29080/api/validation-rules?tenant_id=910638ba-a459-4a3f-bb2d-78391b0595f6&rule_type=business_logic"
```

---

### Get Single Validation Rule
**Endpoint:** `GET /api/validation-rules/{id}`

**Query Parameters:**
- `tenant_id` (required): UUID of the tenant

**Response:**
Same as individual rule object from list endpoint

**Example:**
```bash
curl "http://localhost:29080/api/validation-rules/rule-uuid?tenant_id=910638ba-a459-4a3f-bb2d-78391b0595f6"
```

---

### Create Validation Rule
**Endpoint:** `POST /api/validation-rules`

**Query Parameters:**
- `tenant_id` (required): UUID of the tenant

**Request Body:**
```json
{
  "rule_name": "Email Format Validation",
  "rule_type": "field_format",
  "description": "Customer email must be valid email format",
  "target_entity": "Customer",
  "condition_json": {
    "field": "email",
    "pattern": "^[^@]+@[^@]+\\.[^@]+$"
  },
  "severity": "error",
  "is_active": true
}
```

**Response:** (HTTP 201)
Same structure as individual rule object

**Error Responses:**
- `400 Bad Request`: Missing required fields or invalid rule_type/severity
- `409 Conflict`: Rule name already exists for this tenant

**Example:**
```bash
curl -X POST "http://localhost:29080/api/validation-rules?tenant_id=910638ba-a459-4a3f-bb2d-78391b0595f6" \
  -H "Content-Type: application/json" \
  -d '{
    "rule_name": "Email Format Validation",
    "rule_type": "field_format",
    "description": "Customer email must be valid format",
    "target_entity": "Customer",
    "condition_json": {
      "field": "email",
      "pattern": "^[^@]+@[^@]+\\.[^@]+$"
    },
    "severity": "error"
  }'
```

---

### Update Validation Rule
**Endpoint:** `PATCH /api/validation-rules/{id}`

**Query Parameters:**
- `tenant_id` (required): UUID of the tenant

**Request Body:**
Same as create request (all fields can be updated)

**Response:**
Updated rule object

**Example:**
```bash
curl -X PATCH "http://localhost:29080/api/validation-rules/rule-uuid?tenant_id=910638ba-a459-4a3f-bb2d-78391b0595f6" \
  -H "Content-Type: application/json" \
  -d '{
    "severity": "warning",
    "is_active": false
  }'
```

---

### Delete Validation Rule
**Endpoint:** `DELETE /api/validation-rules/{id}`

**Query Parameters:**
- `tenant_id` (required): UUID of the tenant

**Response:** (HTTP 204 No Content)

**Example:**
```bash
curl -X DELETE "http://localhost:29080/api/validation-rules/rule-uuid?tenant_id=910638ba-a459-4a3f-bb2d-78391b0595f6"
```

---

### Execute Validation Rule
**Endpoint:** `POST /api/validation-rules/{id}/execute`

**Query Parameters:**
- `tenant_id` (required): UUID of the tenant

**Request Body:** (optional)
```json
{
  "data": {
    "field_name": "value",
    "another_field": 123
  }
}
```

**Response:**
```json
{
  "rule_id": "uuid",
  "rule_name": "Email Format Validation",
  "rule_type": "field_format",
  "severity": "error",
  "status": "pass",
  "message": "Rule executed successfully",
  "timestamp": "2025-10-19T10:00:00Z"
}
```

**Example:**
```bash
curl -X POST "http://localhost:29080/api/validation-rules/rule-uuid/execute?tenant_id=910638ba-a459-4a3f-bb2d-78391b0595f6" \
  -H "Content-Type: application/json"
```

---

### Execute Multiple Validation Rules (Batch)
**Endpoint:** `POST /api/validation-rules/execute-batch`

**Query Parameters:**
- `tenant_id` (required): UUID of the tenant

**Request Body:**
```json
{
  "rule_ids": ["rule-uuid-1", "rule-uuid-2"]
}
```

**Response:**
```json
{
  "total_rules": 2,
  "results": [
    {
      "rule_id": "uuid",
      "rule_name": "Rule 1",
      "rule_type": "business_logic",
      "severity": "error",
      "status": "pass",
      "message": "Rule executed successfully",
      "timestamp": "2025-10-19T10:00:00Z"
    }
  ]
}
```

**Example:**
```bash
curl -X POST "http://localhost:29080/api/validation-rules/execute-batch?tenant_id=910638ba-a459-4a3f-bb2d-78391b0595f6" \
  -H "Content-Type: application/json" \
  -d '{
    "rule_ids": ["rule-uuid-1", "rule-uuid-2"]
  }'
```

---

### Get Validation Rule Audit Trail
**Endpoint:** `GET /api/validation-rules/{id}/audit`

**Query Parameters:**
- `tenant_id` (required): UUID of the tenant

**Response:**
```json
[
  {
    "id": "audit-uuid",
    "rule_id": "rule-uuid",
    "tenant_id": "tenant-uuid",
    "action": "UPDATE",
    "old_values": {
      "severity": "error"
    },
    "new_values": {
      "severity": "warning"
    },
    "changed_by": "user-uuid",
    "changed_at": "2025-10-19T10:00:00Z"
  }
]
```

**Example:**
```bash
curl "http://localhost:29080/api/validation-rules/rule-uuid/audit?tenant_id=910638ba-a459-4a3f-bb2d-78391b0595f6"
```

---

## Rule Types

### 1. Field Format
Validates that a field matches a regex pattern.

**Condition JSON:**
```json
{
  "field": "email",
  "pattern": "^[^@]+@[^@]+\\.[^@]+$"
}
```

**Example Use Cases:**
- Email validation: `^[^@]+@[^@]+\.[^@]+$`
- Phone number: `^\\+?[1-9]\\d{1,14}$`
- URL validation: `^https?://[a-zA-Z0-9.-]+`
- ZIP code: `^\\d{5}(-\\d{4})?$`

---

### 2. Cardinality
Validates count/threshold conditions with comparison operators.

**Condition JSON:**
```json
{
  "field": "stock",
  "operator": "<",
  "value": 10
}
```

**Supported Operators:** `>`, `<`, `>=`, `<=`, `==`, `!=`

**Example Use Cases:**
- Inventory low stock warning
- Performance thresholds
- Data size constraints
- Count validations

---

### 3. Uniqueness
Ensures values are unique within the dataset.

**Condition JSON:**
```json
{
  "field": "email",
  "unique": true
}
```

**Example Use Cases:**
- Email uniqueness
- Username uniqueness
- Account number uniqueness
- Product SKU uniqueness

---

### 4. Referential Integrity
Validates foreign key relationships between entities.

**Condition JSON:**
```json
{
  "source_entity": "Order",
  "source_field": "customer_id",
  "target_entity": "Customer",
  "target_field": "id"
}
```

**Example Use Cases:**
- Order → Customer relationship
- LineItem → Product relationship
- Transaction → Account relationship

---

### 5. Business Logic
Custom business rules with flexible condition evaluation.

**Condition JSON:**
```json
{
  "field": "total",
  "operator": ">",
  "value": 0
}
```

**Example Use Cases:**
- Order total must be positive
- Discount cannot exceed item price
- Start date before end date
- Age must be 18+

---

## Validation Engine

The `ValidationEngine` (`backend/internal/validation/engine.go`) provides rule execution:

### ExecutionContext
```go
type ExecutionContext struct {
    RuleID       string
    RuleType     string
    TargetEntity string
    Condition    map[string]interface{}
    Data         map[string]interface{}
}
```

### ExecutionResult
```go
type ExecutionResult struct {
    RuleID   string
    Passed   bool
    Message  string
    Details  map[string]interface{}
}
```

### Usage Example
```go
engine := validation.NewValidationEngine()

ctx := validation.ExecutionContext{
    RuleID:   "rule-uuid",
    RuleType: "field_format",
    TargetEntity: "Customer",
    Condition: map[string]interface{}{
        "field": "email",
        "pattern": "^[^@]+@[^@]+\\.[^@]+$",
    },
    Data: map[string]interface{}{
        "email": "user@example.com",
    },
}

result := engine.Execute(ctx)
if result.Passed {
    log.Println("Validation passed:", result.Message)
} else {
    log.Println("Validation failed:", result.Message)
}
```

---

## Tenant Scoping

All endpoints enforce tenant scoping:

1. **Query Parameter Requirement**: `tenant_id` must be provided in all requests
2. **Database Isolation**: All queries filter by `tenant_id` to prevent cross-tenant data access
3. **Audit Logging**: All changes are recorded with tenant context
4. **Error Handling**: Missing `tenant_id` returns `400 Bad Request`

**Example with Headers (Optional):**
```bash
curl "http://localhost:29080/api/validation-rules?tenant_id=910638ba-a459-4a3f-bb2d-78391b0595f6" \
  -H "X-Tenant-ID: 910638ba-a459-4a3f-bb2d-78391b0595f6"
```

---

## Error Handling

### Error Response Format
```json
{
  "error": "Error message",
  "error_code": "error_code",
  "details": "Additional context"
}
```

### Common Error Codes
- `missing_tenant`: `tenant_id` not provided
- `validation_error`: Request validation failed
- `not_found`: Rule not found
- `duplicate_rule`: Rule name already exists
- `query_error`: Database query error
- `decode_error`: Request body decode error
- `create_error`: Failed to create rule
- `update_error`: Failed to update rule
- `delete_error`: Failed to delete rule
- `parse_error`: Invalid JSON in condition

---

## Migration Setup

Run the migration to create the tables:

```bash
# Migration file: backend/migrations/create_validation_rules.sql
# Automatically applied on service start if using migration system
```

---

## Frontend Integration

The frontend (`frontend/src/pages/catalog/ValidationRulesPage.tsx`) integrates with these endpoints:

### API Hook Pattern (Recommended)

```typescript
// Create new API hook in frontend
const useValidationRules = () => {
  const { apiBaseUrl } = useEnvironment();
  const tenantContext = useTenantContext();

  return {
    listRules: async (filters?: RuleFilters) => {
      const params = new URLSearchParams({
        tenant_id: tenantContext.selectedTenant.id,
        ...filters,
      });
      const response = await fetch(`${apiBaseUrl}/api/validation-rules?${params}`);
      return response.json();
    },
    createRule: async (rule: ValidationRuleRequest) => {
      const response = await fetch(`${apiBaseUrl}/api/validation-rules?tenant_id=${tenantContext.selectedTenant.id}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(rule),
      });
      return response.json();
    },
    // ... more methods
  };
};
```

---

## Performance Considerations

1. **Indexes**: All common filter columns are indexed for fast queries
2. **JSONB Queries**: Use GIN index for complex condition queries
3. **Pagination**: Consider adding pagination for large result sets (future enhancement)
4. **Caching**: Rules could be cached client-side with cache invalidation on updates
5. **Batch Operations**: Use batch execution for multiple rules to reduce API calls

---

## Future Enhancements

1. **Rule Templates**: Pre-built rule templates for common scenarios
2. **Rule Versioning**: Track and manage multiple versions of rules
3. **Scheduling**: Run rules on schedule (daily, weekly, etc.)
4. **Notifications**: Alert on rule violations
5. **Remediation**: Auto-fix for certain rule violations
6. **Analytics**: Dashboard showing rule violation trends
7. **ML Integration**: Suggest rules based on data patterns
8. **Performance Metrics**: Track rule execution performance

---

## Testing

### Manual Testing with curl

**Create a rule:**
```bash
curl -X POST "http://localhost:29080/api/validation-rules?tenant_id=910638ba-a459-4a3f-bb2d-78391b0595f6" \
  -H "Content-Type: application/json" \
  -d '{
    "rule_name": "Test Email Validation",
    "rule_type": "field_format",
    "description": "Validate email format",
    "target_entity": "Customer",
    "condition_json": {
      "field": "email",
      "pattern": "^[^@]+@[^@]+\\.[^@]+$"
    },
    "severity": "error",
    "is_active": true
  }'
```

**List rules:**
```bash
curl "http://localhost:29080/api/validation-rules?tenant_id=910638ba-a459-4a3f-bb2d-78391b0595f6"
```

**Execute a rule:**
```bash
curl -X POST "http://localhost:29080/api/validation-rules/{rule-id}/execute?tenant_id=910638ba-a459-4a3f-bb2d-78391b0595f6" \
  -H "Content-Type: application/json"
```

---

## Documentation

- **Migration**: `backend/migrations/create_validation_rules.sql`
- **Routes**: `backend/internal/api/validation_rules_routes.go`
- **Engine**: `backend/internal/validation/engine.go`
- **Frontend**: `frontend/src/pages/catalog/ValidationRulesPage.tsx`

