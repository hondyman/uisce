# Entity ID-Based Validation Rules - Quick Start Testing Guide

## Quick Overview
Validation rules are now linked to business entities by **UUID** instead of name, making them resilient to entity name changes.

## What Changed?

### For Users
✓ Same validation rules experience in UI
✓ Rules now survive entity name changes
✓ More reliable rule assignments to entities

### For Developers
✓ New database columns: `target_entity_id`, `target_entity_ids`
✓ New backend endpoint: `GET /api/entities/resolve`
✓ New frontend hook: `useEntityResolution`
✓ Backend API accepts both `entity_ids` and `entities` parameters

## Deploy & Test (5 minutes)

### Step 1: Run Database Migration
```bash
cd /Users/eganpj/GitHub/semlayer

# Apply migration
psql postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable < backend/migrations/add_entity_uuid_to_validation_rules.sql

# Verify new columns exist
psql postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable -c "
  SELECT column_name, data_type 
  FROM information_schema.columns 
  WHERE table_name = 'catalog_validation_rules' 
  AND column_name IN ('target_entity_id', 'target_entity_ids', 'datasource_id')
"
```

Expected output:
```
      column_name      |       data_type
-----------------------+----------------------
 target_entity_id      | uuid
 target_entity_ids     | uuid[]
 datasource_id         | uuid
```

### Step 2: Restart Backend Server
```bash
# Kill existing backend
pkill -f "go run.*main.go"

# Restart
cd /Users/eganpj/GitHub/semlayer/backend/cmd/server
go run main.go &

# Verify it starts
sleep 2
curl -s http://localhost:8080/health | jq .
```

### Step 3: Test Entity Resolution Endpoint
```bash
# Get tenant/datasource IDs (update these with real values from localStorage)
TENANT_ID="910638ba-a459-4a3f-bb2d-78391b0595f6"
DATASOURCE_ID="22222222-2222-2222-2222-222222222222"

# Call entity resolution endpoint
curl -s -H "X-Tenant-ID: $TENANT_ID" \
     -H "X-Tenant-Datasource-ID: $DATASOURCE_ID" \
     http://localhost:8080/api/entities/resolve | jq .
```

Expected output:
```json
{
  "employee": {
    "id": "12345678-1234-1234-1234-123456789012",
    "key": "employee",
    "name": "Employee"
  },
  "account": {
    "id": "87654321-8765-8765-8765-876543210987",
    "key": "account",
    "name": "Account"
  }
}
```

### Step 4: Test UUID-Based Filtering
```bash
# Fetch validation rules for 'employee' entity using UUID
EMPLOYEE_UUID="12345678-1234-1234-1234-123456789012"

curl -s -H "X-Tenant-ID: $TENANT_ID" \
     -H "X-Tenant-Datasource-ID: $DATASOURCE_ID" \
     "http://localhost:8080/api/validation-rules?tenant_id=$TENANT_ID&datasource_id=$DATASOURCE_ID&entity_ids=$EMPLOYEE_UUID" | jq '.total'

# Compare with name-based filtering (should be same)
curl -s -H "X-Tenant-ID: $TENANT_ID" \
     -H "X-Tenant-Datasource-ID: $DATASOURCE_ID" \
     "http://localhost:8080/api/validation-rules?tenant_id=$TENANT_ID&datasource_id=$DATASOURCE_ID&entities=employee" | jq '.total'
```

Both should return the same number of rules.

### Step 5: Test in UI
1. Open browser to http://localhost:5173
2. Select tenant and datasource
3. Navigate to any entity details page
4. Click "Validations" tab
5. Check browser Network tab:
   - Should see `/api/entities/resolve` call
   - Should see `/api/validation-rules?...&entity_ids=<uuid>` call
   - Validation rules should display correctly

### Step 6: Verify Response Fields
```bash
curl -s "http://localhost:8080/api/validation-rules?tenant_id=$TENANT_ID&datasource_id=$DATASOURCE_ID&entities=employee&limit=1" | jq '.rules[0] | {
  id,
  rule_name,
  target_entity,
  target_entity_id,
  target_entities,
  target_entity_ids
}'
```

Expected output:
```json
{
  "id": "rule-uuid-here",
  "rule_name": "Employee Email Format",
  "target_entity": "employee",
  "target_entity_id": null,
  "target_entities": ["employee"],
  "target_entity_ids": null
}
```

Note: Existing rules won't have UUIDs populated yet (backward compat). New rules will have both.

## Troubleshooting

### Migration fails: "constraint already exists"
- Already applied? Check if columns exist:
  ```sql
  SELECT * FROM information_schema.columns 
  WHERE table_name = 'catalog_validation_rules' 
  LIMIT 5;
  ```

### `/api/entities/resolve` returns `{}`
- Check tenant/datasource IDs are correct
- Verify entities exist in fabric_defn:
  ```sql
  SELECT model_key, id, is_current FROM fabric_defn LIMIT 5;
  ```

### Validation rules not showing
- Check network tab for 404 errors
- Verify backend is running: `lsof -i :8080`
- Check browser console for errors

### Entity UUID returns null
- Entities may not have UUIDs yet (new columns are nullable)
- Falls back to name-based filtering automatically
- New entities created after migration will have UUIDs

## Key Files

| File | Purpose |
|------|---------|
| `/backend/migrations/add_entity_uuid_to_validation_rules.sql` | Database schema changes |
| `/backend/internal/api/validation_rules_routes.go` | Updated API structs & filtering |
| `/backend/internal/api/api.go` | Entity resolution endpoint |
| `/frontend/src/hooks/useEntityResolution.ts` | Frontend hook for entity ID lookup |
| `/frontend/src/pages/EntityDetailsPage.tsx` | Updated to use entity IDs |

## API Reference

### Entity Resolution
```
GET /api/entities/resolve

Headers:
  X-Tenant-ID: <tenant-uuid>
  X-Tenant-Datasource-ID: <datasource-uuid>

Response:
{
  "<entity_key>": {
    "id": "<entity-uuid>",
    "key": "<entity-key>",
    "name": "<entity-display-name>"
  }
}
```

### Validation Rules - UUID Filtering
```
GET /api/validation-rules

Query Parameters:
  tenant_id=<uuid>                 (required)
  datasource_id=<uuid>             (required)
  entity_ids=<uuid>                (new: UUID-based filtering - PREFERRED)
  entities=<name>                  (legacy: name-based filtering)
  page=1
  limit=20

Response:
{
  "rules": [
    {
      "id": "...",
      "rule_name": "...",
      "target_entity": "...",        (legacy: entity name)
      "target_entity_id": "...",     (new: entity UUID)
      "target_entities": [...],      (legacy: array of names)
      "target_entity_ids": [...]     (new: array of UUIDs)
      ...
    }
  ],
  "total": 5,
  "page": 1,
  "has_more": false
}
```

## Performance Impact

- ✅ Queries faster with indexed UUID lookups
- ✅ Smaller UUID (16 bytes) vs string entity names
- ✅ Array overlap operations cached

## Next Steps

1. Run migration on production database
2. Deploy backend changes
3. Deploy frontend changes
4. Monitor entity resolution endpoint performance
5. Eventually populate `target_entity_id` for existing rules
6. (Optional) Deprecate name-based filtering in future version

## Questions?

See `/ENTITY_ID_VALIDATION_RULES_GUIDE.md` for architecture details.
