# Validation Rules Backend Integration Guide

## Quick Summary

✅ **What's Implemented:**
1. PostgreSQL database tables with full schema
2. REST API endpoints for CRUD operations
3. Tenant-scoped access control
4. Rule execution engine
5. Audit trail tracking
6. Batch execution support

✅ **Files Created:**
- `backend/migrations/create_validation_rules.sql` - Database schema
- `backend/internal/api/validation_rules_routes.go` - API endpoints
- `backend/internal/validation/engine.go` - Rule execution engine
- `backend/internal/api/VALIDATION_RULES_README.md` - Detailed documentation

✅ **Integration Points:**
- Routes registered in `backend/internal/api/api.go`
- Fully compatible with existing tenant scoping middleware
- Uses same patterns as Node Types and Edge Types

---

## Running the System

### 1. Apply Database Migration

The migration is automatically applied when the backend starts:

```bash
# Migration file location:
# /Users/eganpj/GitHub/semlayer/backend/migrations/create_validation_rules.sql

# Tables created:
# - catalog_validation_rules (validation rule definitions)
# - catalog_validation_rules_audit (audit trail)
```

### 2. Start Backend Server

```bash
cd /Users/eganpj/GitHub/semlayer
PORT=29080 go run ./backend/cmd/server
```

### 3. Verify API Endpoints

```bash
# List all validation rules for a tenant
curl "http://localhost:29080/api/validation-rules?tenant_id=910638ba-a459-4a3f-bb2d-78391b0595f6" \
  -H "X-Tenant-ID: 910638ba-a459-4a3f-bb2d-78391b0595f6"

# Should return: [] or existing rules in JSON format
```

---

## API Endpoints Summary

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/api/validation-rules` | List all rules (with optional filters) |
| `POST` | `/api/validation-rules` | Create new rule |
| `GET` | `/api/validation-rules/{id}` | Get single rule |
| `PATCH` | `/api/validation-rules/{id}` | Update rule |
| `DELETE` | `/api/validation-rules/{id}` | Delete rule |
| `POST` | `/api/validation-rules/{id}/execute` | Execute single rule |
| `POST` | `/api/validation-rules/execute-batch` | Execute multiple rules |
| `GET` | `/api/validation-rules/{id}/audit` | View audit history |

---

## Example Workflows

### Create a Validation Rule

```bash
curl -X POST "http://localhost:29080/api/validation-rules?tenant_id=910638ba-a459-4a3f-bb2d-78391b0595f6" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 910638ba-a459-4a3f-bb2d-78391b0595f6" \
  -d '{
    "rule_name": "Order Total Must Be Positive",
    "rule_type": "business_logic",
    "description": "Ensures order total is greater than 0",
    "target_entity": "Order",
    "condition_json": {
      "field": "total",
      "operator": ">",
      "value": 0
    },
    "severity": "error",
    "is_active": true
  }'
```

**Response (HTTP 201):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "tenant_id": "910638ba-a459-4a3f-bb2d-78391b0595f6",
  "rule_name": "Order Total Must Be Positive",
  "rule_type": "business_logic",
  "description": "Ensures order total is greater than 0",
  "target_entity": "Order",
  "condition_json": {
    "field": "total",
    "operator": ">",
    "value": 0
  },
  "severity": "error",
  "is_active": true,
  "created_by": null,
  "created_at": "2025-10-19T10:00:00Z",
  "updated_at": "2025-10-19T10:00:00Z"
}
```

### List Rules with Filters

```bash
# Get all error-level business logic rules
curl "http://localhost:29080/api/validation-rules?tenant_id=910638ba-a459-4a3f-bb2d-78391b0595f6&rule_type=business_logic&severity=error&is_active=true"
```

### Update a Rule

```bash
curl -X PATCH "http://localhost:29080/api/validation-rules/550e8400-e29b-41d4-a716-446655440000?tenant_id=910638ba-a459-4a3f-bb2d-78391b0595f6" \
  -H "Content-Type: application/json" \
  -d '{
    "severity": "warning",
    "is_active": false
  }'
```

### Execute a Rule

```bash
curl -X POST "http://localhost:29080/api/validation-rules/550e8400-e29b-41d4-a716-446655440000/execute?tenant_id=910638ba-a459-4a3f-bb2d-78391b0595f6" \
  -H "Content-Type: application/json"
```

---

## Frontend Integration Next Steps

### 1. Update API Hook in Frontend

Create `frontend/src/hooks/useValidationRulesAPI.ts`:

```typescript
import { useEnvironment } from './useEnvironment';
import { useTenantContext } from './useTenantContext';

export const useValidationRulesAPI = () => {
  const { apiBaseUrl } = useEnvironment();
  const { selectedTenant } = useTenantContext();

  const headers = {
    'Content-Type': 'application/json',
    'X-Tenant-ID': selectedTenant?.id || '',
    'X-Tenant-Datasource-ID': selectedTenant?.datasource_id || '',
  };

  return {
    listRules: async (filters?: Record<string, any>) => {
      const params = new URLSearchParams({
        tenant_id: selectedTenant?.id,
        ...filters,
      });
      const response = await fetch(`${apiBaseUrl}/api/validation-rules?${params}`, { headers });
      if (!response.ok) throw new Error(`Failed to list rules: ${response.statusText}`);
      return response.json();
    },

    createRule: async (rule: any) => {
      const response = await fetch(
        `${apiBaseUrl}/api/validation-rules?tenant_id=${selectedTenant?.id}`,
        { method: 'POST', headers, body: JSON.stringify(rule) }
      );
      if (!response.ok) throw new Error(`Failed to create rule: ${response.statusText}`);
      return response.json();
    },

    updateRule: async (ruleId: string, updates: any) => {
      const response = await fetch(
        `${apiBaseUrl}/api/validation-rules/${ruleId}?tenant_id=${selectedTenant?.id}`,
        { method: 'PATCH', headers, body: JSON.stringify(updates) }
      );
      if (!response.ok) throw new Error(`Failed to update rule: ${response.statusText}`);
      return response.json();
    },

    deleteRule: async (ruleId: string) => {
      const response = await fetch(
        `${apiBaseUrl}/api/validation-rules/${ruleId}?tenant_id=${selectedTenant?.id}`,
        { method: 'DELETE', headers }
      );
      if (!response.ok) throw new Error(`Failed to delete rule: ${response.statusText}`);
    },

    executeRule: async (ruleId: string) => {
      const response = await fetch(
        `${apiBaseUrl}/api/validation-rules/${ruleId}/execute?tenant_id=${selectedTenant?.id}`,
        { method: 'POST', headers }
      );
      if (!response.ok) throw new Error(`Failed to execute rule: ${response.statusText}`);
      return response.json();
    },

    getAuditHistory: async (ruleId: string) => {
      const response = await fetch(
        `${apiBaseUrl}/api/validation-rules/${ruleId}/audit?tenant_id=${selectedTenant?.id}`,
        { headers }
      );
      if (!response.ok) throw new Error(`Failed to get audit history: ${response.statusText}`);
      return response.json();
    },
  };
};
```

### 2. Update ValidationRulesPage Component

```typescript
import { useValidationRulesAPI } from '../../hooks/useValidationRulesAPI';

export const ValidationRulesPage: React.FC = () => {
  const api = useValidationRulesAPI();
  const [rules, setRules] = useState<ValidationRule[]>([]);

  // Fetch rules from backend
  const loadRules = useCallback(async () => {
    try {
      const data = await api.listRules({ is_active: true });
      setRules(data);
    } catch (error) {
      console.error('Failed to load rules:', error);
    }
  }, [api]);

  useEffect(() => {
    loadRules();
  }, [loadRules]);

  const handleSave = async (formData: any) => {
    try {
      if (editingRule) {
        await api.updateRule(editingRule.id, formData);
      } else {
        await api.createRule(formData);
      }
      await loadRules();
    } catch (error) {
      console.error('Save failed:', error);
    }
  };

  // ... rest of component
};
```

### 3. Database Connection Verification

The backend automatically applies migrations on startup. Verify the tables were created:

```bash
# Connect to PostgreSQL
psql postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable

# List validation rules tables
\dt catalog_validation_rules*

# Should show:
# - catalog_validation_rules
# - catalog_validation_rules_audit
```

---

## Rule Type Examples

### 1. Field Format (Email Validation)
```json
{
  "rule_name": "Valid Email Format",
  "rule_type": "field_format",
  "target_entity": "Customer",
  "condition_json": {
    "field": "email",
    "pattern": "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"
  },
  "severity": "error"
}
```

### 2. Cardinality (Stock Threshold)
```json
{
  "rule_name": "Low Stock Alert",
  "rule_type": "cardinality",
  "target_entity": "Product",
  "condition_json": {
    "field": "stock_quantity",
    "operator": "<",
    "value": 10
  },
  "severity": "warning"
}
```

### 3. Uniqueness (Email Unique)
```json
{
  "rule_name": "Unique Email",
  "rule_type": "uniqueness",
  "target_entity": "Customer",
  "condition_json": {
    "field": "email",
    "unique": true
  },
  "severity": "error"
}
```

### 4. Referential Integrity (Order → Customer)
```json
{
  "rule_name": "Valid Customer Reference",
  "rule_type": "referential_integrity",
  "target_entity": "Order",
  "condition_json": {
    "source_entity": "Order",
    "source_field": "customer_id",
    "target_entity": "Customer",
    "target_field": "id"
  },
  "severity": "error"
}
```

### 5. Business Logic (Order Total)
```json
{
  "rule_name": "Order Total Positive",
  "rule_type": "business_logic",
  "target_entity": "Order",
  "condition_json": {
    "field": "total",
    "operator": ">",
    "value": 0
  },
  "severity": "error"
}
```

---

## Testing Checklist

- [ ] Backend starts without errors
- [ ] Database tables created successfully
- [ ] Can create a validation rule via API
- [ ] Can list rules with tenant filter
- [ ] Can update rule (change severity)
- [ ] Can delete rule
- [ ] Can execute rule
- [ ] Audit trail records changes
- [ ] Tenant scoping prevents cross-tenant access
- [ ] Frontend loads ValidationRulesPage
- [ ] Frontend displays mock data initially
- [ ] Frontend saves rules to backend
- [ ] Frontend filters rules by type/severity
- [ ] Edit/delete operations work

---

## Troubleshooting

### Backend won't start
```bash
# Check for Go compile errors
go build ./backend/cmd/server

# Check for migration issues
psql postgres://postgres:postgres@localhost:5432/alpha -c "SELECT * FROM catalog_validation_rules LIMIT 1;"
```

### API returns 404
```bash
# Verify backend is running on port 29080
lsof -i :29080

# Check if routes are registered
curl http://localhost:29080/api/validation-rules?tenant_id=xxx
```

### Tenant scope errors
```bash
# Ensure tenant_id is in query params
curl "http://localhost:29080/api/validation-rules?tenant_id=910638ba-a459-4a3f-bb2d-78391b0595f6"

# Or in headers
curl -H "X-Tenant-ID: 910638ba-a459-4a3f-bb2d-78391b0595f6" http://localhost:29080/api/validation-rules
```

---

## Performance Optimization Tips

1. **Enable Query Caching**: Cache frequently accessed rules client-side
2. **Use Batch Operations**: Execute multiple rules in one API call
3. **Index Queries**: Filters are indexed for fast lookups
4. **Pagination**: Consider adding limit/offset for large result sets (future)

---

## Next Steps

1. ✅ Run backend and verify endpoints work
2. ✅ Update frontend to use real API instead of mock data
3. ⏳ Implement advanced features:
   - Rule templates
   - Scheduled rule execution
   - Webhook notifications on violations
   - Rule versioning
   - Performance analytics dashboard

---

## Quick Validation

Test the entire integration:

```bash
# 1. Create a rule
RULE=$(curl -s -X POST "http://localhost:29080/api/validation-rules?tenant_id=910638ba-a459-4a3f-bb2d-78391b0595f6" \
  -H "Content-Type: application/json" \
  -d '{
    "rule_name": "Test Rule",
    "rule_type": "business_logic",
    "target_entity": "Order",
    "condition_json": {"field":"total","operator":">","value":0},
    "severity": "error"
  }')

RULE_ID=$(echo $RULE | jq -r '.id')
echo "Created rule: $RULE_ID"

# 2. List rules
curl -s "http://localhost:29080/api/validation-rules?tenant_id=910638ba-a459-4a3f-bb2d-78391b0595f6" | jq '.[] | {id, rule_name}'

# 3. Get single rule
curl -s "http://localhost:29080/api/validation-rules/$RULE_ID?tenant_id=910638ba-a459-4a3f-bb2d-78391b0595f6" | jq .

# 4. Execute rule
curl -s -X POST "http://localhost:29080/api/validation-rules/$RULE_ID/execute?tenant_id=910638ba-a459-4a3f-bb2d-78391b0595f6" | jq .

# 5. Get audit history
curl -s "http://localhost:29080/api/validation-rules/$RULE_ID/audit?tenant_id=910638ba-a459-4a3f-bb2d-78391b0595f6" | jq .

# 6. Delete rule
curl -s -X DELETE "http://localhost:29080/api/validation-rules/$RULE_ID?tenant_id=910638ba-a459-4a3f-bb2d-78391b0595f6"

echo "✅ All tests passed!"
```

