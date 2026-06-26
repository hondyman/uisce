# Entity ID-Based Validation Rules - Implementation Guide

## Problem Statement
Currently, validation rules are linked to entities by **name** (string), not by **UUID**. This creates fragility:
- If an entity name/key changes, all validation rules referencing it break
- No foreign key relationship between rules and entities in the database
- Frontend and backend both use string-based matching, which is error-prone

## Solution Overview
Implement **UUID-based** linking alongside legacy name-based support:
1. Add `target_entity_id` (UUID) and `target_entity_ids` (UUID array) columns to `catalog_validation_rules`
2. Maintain backward compatibility with existing `target_entity` (string) and `target_entities` (string array)
3. Update backend to query by UUID when available, fall back to name-based matching
4. Update frontend to send entity UUIDs instead of names

## Database Changes

### Schema Migration
File: `/backend/migrations/add_entity_uuid_to_validation_rules.sql`

New columns:
- `target_entity_id UUID`: Single entity UUID reference
- `target_entity_ids UUID[] DEFAULT ARRAY[]::UUID[]`: Multiple entity UUIDs
- `datasource_id UUID`: Datasource scoping

New indexes:
- `idx_validation_rules_entity_id`: For UUID lookups
- `idx_validation_rules_entity_ids`: GIN index for array operations
- `idx_validation_rules_datasource`: For datasource filtering

### Validation View
Created `validation_rules_with_entities` view that:
- JOINs with `fabric_defn` to resolve entity keys and names
- Handles both UUID and name-based matching
- Provides normalized entity information for queries

## Backend Changes

### API Structures (`validation_rules_routes.go`)

**ValidationRule struct** - Now includes:
```go
TargetEntityID  string         // NEW: Entity UUID
TargetEntityIDs pq.StringArray // NEW: Multi-entity UUIDs array
DatasourceID    string         // NEW: Datasource scope
```

**ValidationRuleRequest struct** - Now includes:
```go
TargetEntityID  string         // NEW: Entity UUID in requests
TargetEntityIDs pq.StringArray // NEW: Multi-entity UUIDs in requests
DatasourceID    string         // NEW: Datasource scope
```

### Query Parameter Support

The `/api/validation-rules` endpoint now accepts:

**UUID-based (PREFERRED):**
```
GET /api/validation-rules?tenant_id=...&datasource_id=...&entity_ids=<uuid1>&entity_ids=<uuid2>
```

**Name-based (LEGACY - backward compatible):**
```
GET /api/validation-rules?tenant_id=...&datasource_id=...&entities=employee&entities=account
```

**Filtering Logic:**
1. If `entity_ids` parameter provided → use UUID filtering with array overlap
2. Else if `entities` parameter provided → use name-based filtering
3. Else → return all rules for tenant (if tenant scope is selected)

### SQL WHERE Clause

UUID-based filtering:
```sql
ARRAY['<uuid1>', '<uuid2>']::uuid[] && COALESCE(target_entity_ids, ARRAY[]::uuid[])
```

Name-based filtering (fallback):
```sql
ARRAY['employee', 'account']::text[] && COALESCE(target_entities, ARRAY[target_entity])
```

## Frontend Changes

### Entity Type Enhancement
The `Entity` interface needs an `id` field to carry the UUID:

```typescript
export interface Entity {
  id?: string;  // NEW: UUID from fabric_defn
  key?: string; // Technical name (lowercase_with_underscores)
  name: string; // Display name
  // ... existing fields ...
}
```

### EntityDetailsPage Update
When fetching validation rules:

**Current (name-based):**
```typescript
const params = new URLSearchParams({
  tenant_id: tenant.id,
  datasource_id: datasource.id,
  entities: entityKey,  // Entity name/key
});
```

**Updated (UUID-based):**
```typescript
const params = new URLSearchParams({
  tenant_id: tenant.id,
  datasource_id: datasource.id,
  entity_ids: entities[entityKey].id,  // Entity UUID from fabric_defn
});
```

## Implementation Steps

1. ✅ **Create database migration** - Add UUID columns and indexes
2. ✅ **Update backend structs** - Add UUID fields to ValidationRule
3. ✅ **Update backend query logic** - Support UUID filtering with fallback to names
4. ✅ **Update backend SELECT** - Include new UUID columns in queries
5. 🔄 **Add entity ID resolution** - Need endpoint to map entity keys to UUIDs
6. 🔄 **Update frontend Entity type** - Add `id` field
7. 🔄 **Update EntityDetailsPage** - Pass entity UUIDs instead of names
8. 🔄 **Test migration path** - Verify backward compatibility with existing rules

## Entity ID Resolution

### Option A: Embed in Entity Schema (Preferred)
Modify `/entity-schema` endpoint to include fabric_defn UUID for each entity:

```typescript
{
  "employee": {
    "id": "22222222-2222-2222-2222-222222222222",  // UUID from fabric_defn
    "key": "employee",
    "name": "Employee",
    "entity_fields": [...],
    "subtypes": {...}
  }
}
```

### Option B: Separate Resolution Endpoint
Create `/api/entities/resolve` endpoint that returns:

```json
{
  "employee": {
    "id": "22222222-2222-2222-2222-222222222222",
    "key": "employee",
    "name": "Employee"
  },
  "account": {
    "id": "33333333-3333-3333-3333-333333333333",
    "key": "account",
    "name": "Account"
  }
}
```

### Option C: Add Metadata Endpoint
Create `/api/entities/{key}/metadata` endpoint for on-demand lookup.

## Benefits

1. **Durability**: Entity name changes don't break validation rules
2. **Referential Integrity**: Database-level foreign key constraints
3. **Performance**: UUID lookups via index are faster than string matching
4. **Auditability**: Clear tracing of which exact entity a rule applies to
5. **Multi-tenancy**: Datasource scoping works correctly
6. **Backward Compatible**: Existing name-based rules still work

## Migration Path

### Phase 1: Infrastructure
- ✅ Add new UUID columns (nullable, backward compatible)
- Create triggers/procedures to auto-populate UUIDs when entity names are known

### Phase 2: Gradual Adoption
- Backend supports both UUID and name-based queries
- Frontend sends both (name for compatibility, UUID when available)
- Existing rules continue using names

### Phase 3: Full UUID Adoption
- Once all rules have UUIDs, deprecate name-based filtering
- Add data migration to populate UUIDs for existing rules
- Update all code to require UUIDs

## Testing Checklist

- [ ] Database migration runs without errors
- [ ] Existing name-based rules still filter correctly
- [ ] New UUID-based rules filter correctly
- [ ] Entity name changes don't affect rule filtering
- [ ] Multiple entities per rule work correctly
- [ ] Datasource scoping works with UUIDs
- [ ] API returns all relevant fields (both UUID and name)
- [ ] Frontend can navigate and see entity-specific rules
- [ ] Backward compatibility with existing clients maintained

## Future Considerations

1. **Subtypes**: Handle subtypes with their own UUIDs if needed
2. **Cross-datasource Rules**: Support rules that apply across datasources
3. **Rule Templates**: Create rules at system level, inherit in datasources
4. **Versioning**: Track entity versions and rule applicability
