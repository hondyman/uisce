# Validation Rules - Quick Reference Guide

## 🚀 Quick Start (5 minutes)

### 1. Start Backend
```bash
cd /Users/eganpj/GitHub/semlayer
PORT=29080 go run ./backend/cmd/server
```
✅ Backend starts on `http://localhost:29080`
✅ Database migration auto-applies
✅ Routes registered automatically

### 2. Verify Frontend
```bash
cd /Users/eganpj/GitHub/semlayer/frontend
npm run dev
```
✅ Frontend starts on `http://localhost:5173`
✅ Validation Rules page available at: `/core/validation-rules`

### 3. Test API
```bash
# Set your tenant ID
TENANT_ID="910638ba-a459-4a3f-bb2d-78391b0595f6"

# List rules
curl "http://localhost:29080/api/validation-rules?tenant_id=$TENANT_ID" \
  -H "X-Tenant-ID: $TENANT_ID"

# Create rule
curl -X POST "http://localhost:29080/api/validation-rules?tenant_id=$TENANT_ID" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "rule_name": "Test Rule",
    "rule_type": "business_logic",
    "target_entity": "Order",
    "condition_json": {"field": "total", "operator": ">", "value": 0},
    "severity": "error",
    "is_active": true
  }'
```

---

## 📋 API Endpoints (All 8)

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/api/validation-rules` | List all rules (with filters) |
| POST | `/api/validation-rules` | Create new rule |
| GET | `/api/validation-rules/{id}` | Get single rule |
| PATCH | `/api/validation-rules/{id}` | Update rule |
| DELETE | `/api/validation-rules/{id}` | Delete rule |
| POST | `/api/validation-rules/{id}/execute` | Execute single rule |
| POST | `/api/validation-rules/execute-batch` | Execute multiple rules |
| GET | `/api/validation-rules/{id}/audit` | Get audit history |

---

## 🔍 Query Parameters (Filters)

### List Endpoint Filters
```bash
# Filter by rule type
/api/validation-rules?tenant_id={id}&rule_type=business_logic

# Filter by severity
/api/validation-rules?tenant_id={id}&severity=error

# Filter by target entity
/api/validation-rules?tenant_id={id}&target_entity=Order

# Filter by active status
/api/validation-rules?tenant_id={id}&is_active=true

# Combine multiple filters
/api/validation-rules?tenant_id={id}&rule_type=cardinality&severity=warning&is_active=true
```

---

## 📦 Rule Types (5 Total)

### 1. Business Logic
Evaluate complex conditions with multiple fields
```json
{
  "rule_type": "business_logic",
  "condition_json": {
    "field": "order_total",
    "operator": ">",
    "value": 0
  }
}
```
**Operators**: `>`, `<`, `>=`, `<=`, `==`, `!=`

### 2. Field Format
Validate string format with regex pattern
```json
{
  "rule_type": "field_format",
  "condition_json": {
    "field": "email",
    "pattern": "^[^@]+@[^@]+\\.[^@]+$"
  }
}
```

### 3. Cardinality
Ensure numeric thresholds are met
```json
{
  "rule_type": "cardinality",
  "condition_json": {
    "field": "inventory_count",
    "operator": ">=",
    "value": 10
  }
}
```

### 4. Uniqueness
Enforce unique values for field
```json
{
  "rule_type": "uniqueness",
  "condition_json": {
    "field": "email",
    "unique": true
  }
}
```

### 5. Referential Integrity
Validate foreign key relationships
```json
{
  "rule_type": "referential_integrity",
  "condition_json": {
    "source_entity": "Order",
    "source_field": "customer_id",
    "target_entity": "Customer",
    "target_field": "id"
  }
}
```

---

## ⚙️ Rule Properties

| Property | Type | Required | Example |
|----------|------|----------|---------|
| `rule_name` | string (255) | ✅ | "Email Format Validation" |
| `rule_type` | enum | ✅ | "field_format" |
| `target_entity` | string (255) | ✅ | "Customer" |
| `condition_json` | JSON object | ✅ | `{"field": "email", ...}` |
| `severity` | enum | ✅ | "error" \| "warning" |
| `description` | text | ❌ | "Validates email format..." |
| `is_active` | boolean | ✅ | true |

**Severity Levels**: `error` (validation fails), `warning` (advisory), `info` (informational)

**Rule Types**: `business_logic`, `field_format`, `cardinality`, `uniqueness`, `referential_integrity`

---

## 🔐 Authentication & Scoping

All endpoints require tenant context:

### Headers (Required)
```
X-Tenant-ID: <tenant-uuid>
X-Tenant-Datasource-ID: <datasource-uuid> (for some operations)
```

### Query Parameters (Required)
```
?tenant_id=<tenant-uuid>
?datasource_id=<datasource-uuid> (for some operations)
```

### Example
```bash
curl -X GET "http://localhost:29080/api/validation-rules?tenant_id=910638ba-a459-4a3f-bb2d-78391b0595f6" \
  -H "X-Tenant-ID: 910638ba-a459-4a3f-bb2d-78391b0595f6" \
  -H "Content-Type: application/json"
```

---

## 💾 Database Schema (Quick Reference)

### `catalog_validation_rules`
```sql
id UUID (PK)
tenant_id UUID (FK) -- Multi-tenant isolation
rule_name VARCHAR(255) -- Unique per tenant
rule_type VARCHAR(50) -- CHECK constraint: valid types only
target_entity VARCHAR(255)
condition_json JSONB -- Flexible condition storage
severity VARCHAR(20) -- CHECK constraint
is_active BOOLEAN
created_by UUID
created_at TIMESTAMP
updated_at TIMESTAMP
```

**Unique Constraint**: `(tenant_id, rule_name)` - prevents duplicate rule names per tenant

### `catalog_validation_rules_audit`
```sql
id UUID (PK)
rule_id UUID (FK) -- Cascades on delete
tenant_id UUID
action VARCHAR(20) -- CREATE, UPDATE, DELETE
old_values JSONB
new_values JSONB
changed_by UUID
changed_at TIMESTAMP
```

### Indexes (Performance)
- `tenant_id` (B-tree)
- `rule_type` (B-tree)
- `target_entity` (B-tree)
- `severity` (B-tree)
- `is_active` (B-tree)
- `condition_json` (GIN - for complex queries)
- `created_at DESC` (B-tree - for audit)

---

## ✅ HTTP Status Codes

| Status | Meaning | Example |
|--------|---------|---------|
| 200 | Success - GET request | List/get/update successful |
| 201 | Created - POST request | Rule created successfully |
| 204 | No Content - DELETE | Rule deleted successfully |
| 400 | Bad Request | Missing required fields |
| 404 | Not Found | Rule ID doesn't exist |
| 409 | Conflict | Duplicate rule name for tenant |
| 500 | Server Error | Database or internal error |

---

## 🔧 Common Tasks

### Create a New Rule
```bash
curl -X POST "http://localhost:29080/api/validation-rules?tenant_id=$TENANT_ID" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "rule_name": "Product Price Validation",
    "rule_type": "business_logic",
    "description": "Product price must be positive",
    "target_entity": "Product",
    "condition_json": {
      "field": "price",
      "operator": ">",
      "value": 0
    },
    "severity": "error",
    "is_active": true
  }'
```

### Update Rule Severity
```bash
curl -X PATCH "http://localhost:29080/api/validation-rules/{rule-id}?tenant_id=$TENANT_ID" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "severity": "warning"
  }'
```

### Disable Rule
```bash
curl -X PATCH "http://localhost:29080/api/validation-rules/{rule-id}?tenant_id=$TENANT_ID" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "is_active": false
  }'
```

### Execute Single Rule
```bash
curl -X POST "http://localhost:29080/api/validation-rules/{rule-id}/execute?tenant_id=$TENANT_ID" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "data": {
      "email": "user@example.com"
    }
  }'
```

### Execute Batch of Rules
```bash
curl -X POST "http://localhost:29080/api/validation-rules/execute-batch?tenant_id=$TENANT_ID" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "rule_ids": ["id1", "id2", "id3"]
  }'
```

### View Audit History
```bash
curl "http://localhost:29080/api/validation-rules/{rule-id}/audit?tenant_id=$TENANT_ID" \
  -H "X-Tenant-ID: $TENANT_ID"
```

### Delete Rule
```bash
curl -X DELETE "http://localhost:29080/api/validation-rules/{rule-id}?tenant_id=$TENANT_ID" \
  -H "X-Tenant-ID: $TENANT_ID"
```

---

## 🧪 Run Full Test Suite

```bash
# Make script executable
chmod +x /Users/eganpj/GitHub/semlayer/test_validation_rules_api.sh

# Run tests (requires backend running on 29080)
/Users/eganpj/GitHub/semlayer/test_validation_rules_api.sh
```

This runs all 20 test cases including:
- ✅ CRUD operations (create, read, update, delete)
- ✅ Filtering (by type, severity, entity, active)
- ✅ Execution (single and batch)
- ✅ Audit trail
- ✅ Error handling (duplicates, validation, not found)
- ✅ Tenant scoping

---

## 📂 File Reference

| File | Purpose | Location |
|------|---------|----------|
| Migration | Database schema | `backend/migrations/create_validation_rules.sql` |
| Routes | REST API handlers | `backend/internal/api/validation_rules_routes.go` |
| Engine | Rule execution | `backend/internal/validation/engine.go` |
| Frontend | UI page | `frontend/src/pages/catalog/ValidationRulesPage.tsx` |
| API Docs | Full reference | `backend/internal/api/VALIDATION_RULES_README.md` |
| Integration Guide | Setup instructions | `BACKEND_VALIDATION_INTEGRATION.md` |
| Test Script | Automated tests | `test_validation_rules_api.sh` |
| Implementation Summary | Project overview | `VALIDATION_RULES_IMPLEMENTATION_SUMMARY.md` |

---

## 🐛 Troubleshooting

### Connection Refused (Port 29080)
- Backend not running
- Solution: `PORT=29080 go run ./backend/cmd/server`

### Missing X-Tenant-ID Header
- Request won't authenticate
- Solution: Add `-H "X-Tenant-ID: <uuid>"` to curl

### Invalid Rule Type
- Endpoint returns 400
- Solution: Check valid types: `business_logic`, `field_format`, `cardinality`, `uniqueness`, `referential_integrity`

### Duplicate Rule Name
- Endpoint returns 409 Conflict
- Solution: Use different rule name or delete existing rule first

### Migration Not Applied
- Tables don't exist
- Solution: Restart backend; migration runs automatically

### Frontend Won't Load Validation Rules Page
- 404 error or page not found
- Solution: Verify route exists in `App.tsx` at `/core/validation-rules`

---

## 📞 Support

For detailed documentation, see:
- **API Reference**: `backend/internal/api/VALIDATION_RULES_README.md`
- **Integration Guide**: `BACKEND_VALIDATION_INTEGRATION.md`
- **Implementation Summary**: `VALIDATION_RULES_IMPLEMENTATION_SUMMARY.md`

For issue tracking or questions about the implementation, refer to the conversation summary in context.
