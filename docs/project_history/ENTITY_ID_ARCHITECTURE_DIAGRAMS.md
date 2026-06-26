# Entity ID-Based Validation Rules - Architecture Diagram

## System Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Frontend (React)                             │
├─────────────────────────────────────────────────────────────────────┤
│                                                                       │
│  ┌─────────────────────────────────┐                                │
│  │  EntityDetailsPage.tsx          │                                │
│  │                                 │                                │
│  │  1. Load entity resolution:     │                                │
│  │     useEntityResolution()       │                                │
│  │                                 │                                │
│  │  2. Fetch validation rules:     │                                │
│  │     GET /api/validation-rules?  │                                │
│  │       entity_ids=<uuid> OR      │                                │
│  │       entities=<name>           │                                │
│  │                                 │                                │
│  │  3. Display filtered rules      │                                │
│  └─────────────────────────────────┘                                │
│           ↑                            ↑                             │
│           │                            │                             │
└─────────────────────────────────────────────────────────────────────┘
           │                            │
           │ 1. GET /api/entities/      │ 2. GET /api/validation-
           │     resolve                │     rules?...
           │                            │
┌──────────────────────────────────────────────────────────────────────┐
│                      Backend (Go HTTP API)                            │
├──────────────────────────────────────────────────────────────────────┤
│                                                                        │
│  ┌──────────────────────────┐   ┌──────────────────────────────┐   │
│  │ Entity Resolution        │   │ Validation Rules API         │   │
│  │ GET /entities/resolve    │   │ GET /validation-rules        │   │
│  │                          │   │                              │   │
│  │ 1. Query fabric_defn:    │   │ 1. Parse parameters:        │   │
│  │    - model_key (entity   │   │    - entity_ids (UUID)      │   │
│  │      key)                │   │    - entities (name)        │   │
│  │    - id (UUID)           │   │                              │   │
│  │    - title (display)     │   │ 2. Build WHERE clause:      │   │
│  │                          │   │    IF entity_ids:           │   │
│  │ 2. Build response map:   │   │      use UUID overlap       │   │
│  │    key → {id, key, name} │   │    ELSE:                    │   │
│  │                          │   │      use name overlap       │   │
│  │ 3. Return JSON           │   │                              │   │
│  │    (cached in frontend)  │   │ 3. Query catalog_validation │   │
│  │                          │   │    _rules with filtering    │   │
│  │                          │   │                              │   │
│  │                          │   │ 4. Return paginated rules   │   │
│  └──────────────────────────┘   └──────────────────────────────┘   │
│                                                                        │
└────────────────────────┬─────────────────────────────────────────────┘
                         │
                         │ SQL Queries
                         ↓
┌──────────────────────────────────────────────────────────────────────┐
│                    PostgreSQL Database                                │
├──────────────────────────────────────────────────────────────────────┤
│                                                                        │
│  ┌────────────────────────────┐  ┌───────────────────────────────┐ │
│  │  fabric_defn               │  │ catalog_validation_rules      │ │
│  │  (Entity Definitions)      │  │ (Validation Rules)            │ │
│  │                            │  │                               │ │
│  │  id (UUID) ◄────────────┐  │  │ id                            │ │
│  │  model_key (entity_key)   │  │ tenant_id                      │ │
│  │  title (entity name)      │  │ datasource_id (NEW)           │ │
│  │  version                  │  │ rule_name                      │ │
│  │  is_current               │  │ target_entity (legacy)        │ │
│  │  tenant_id                │  │ target_entity_id (NEW) ────┐  │ │
│  │  tenant_datasource_id     │  │ target_entities (legacy)    │  │ │
│  │                           │  │ target_entity_ids (NEW) ────┼──┼─┐
│  │ PK: id                    │  │                             │  │ │
│  │ INDEX: model_key          │  │ FK: target_entity_id ───────┘  │ │
│  │ INDEX: tenant_datasource  │  │ GIN INDEX: target_entity_ids───┘ │
│  │                           │  │                               │ │
│  │                           │  │ query_condition_json          │ │
│  │                           │  │ severity                      │ │
│  │                           │  │ is_active                     │ │
│  │                           │  │ created_at, updated_at        │ │
│  └────────────────────────────┘  │                               │ │
│                                  │ PK: id                        │ │
│                                  │ INDEX: tenant_id              │ │
│                                  │ INDEX: datasource_id (NEW)    │ │
│                                  │ INDEX: target_entity_id (NEW) │ │
│                                  │ GIN INDEX: target_entity_ids  │ │
│                                  └───────────────────────────────┘ │
│                                                                        │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │ validation_rules_with_entities (VIEW - NEW)                   │ │
│  │                                                                │ │
│  │ Joins catalog_validation_rules with fabric_defn to provide:  │ │
│  │ - Resolved entity_key (from model_key)                       │ │
│  │ - Resolved entity_name (from title)                          │ │
│  │ - Entity UUID (from id)                                      │ │
│  │ - Supports both UUID and name-based matching                 │ │
│  └────────────────────────────────────────────────────────────────┘ │
│                                                                        │
└──────────────────────────────────────────────────────────────────────┘
```

## Data Flow Diagram

### A. Fetching Entity IDs

```
┌────────────────────────────┐
│  User opens entity page    │
│  (EntityDetailsPage.tsx)   │
└────────────────────────────┘
           ↓
┌────────────────────────────────────────────┐
│  useEntityResolution hook initializes:     │
│  - tenantId, datasourceId passed in        │
└────────────────────────────────────────────┘
           ↓
┌────────────────────────────────────────────┐
│  Hook calls:                               │
│  GET /api/entities/resolve                 │
│  Headers: X-Tenant-ID, X-Tenant-Datasource│
└────────────────────────────────────────────┘
           ↓
┌────────────────────────────────────────────┐
│  Backend queries fabric_defn:              │
│  SELECT id, model_key, title               │
│  WHERE tenant_id = $1                      │
│    AND tenant_datasource_id = $2           │
│    AND is_current = true                   │
└────────────────────────────────────────────┘
           ↓
┌────────────────────────────────────────────┐
│  Response returned:                        │
│  {                                         │
│    "employee": {                           │
│      "id": "uuid1",                        │
│      "key": "employee",                    │
│      "name": "Employee"                    │
│    },                                      │
│    "account": {...}                        │
│  }                                         │
└────────────────────────────────────────────┘
           ↓
┌────────────────────────────────────────────┐
│  Frontend caches result in hook state      │
│  Calls getEntityId("employee") → "uuid1"   │
└────────────────────────────────────────────┘
```

### B. Fetching Validation Rules with UUID Filter

```
┌──────────────────────────────────────┐
│  User clicks "Validations" tab       │
└──────────────────────────────────────┘
           ↓
┌──────────────────────────────────────────────┐
│  fetchValidationRules called:                │
│  - entityKey = "employee"                    │
│  - entities[entityKey] = {...}               │
└──────────────────────────────────────────────┘
           ↓
┌──────────────────────────────────────────────┐
│  Get entity UUID from cache:                 │
│  entityId = getEntityId("employee")          │
│  → "22222222-2222-2222-2222-222222222222"   │
└──────────────────────────────────────────────┘
           ↓
┌──────────────────────────────────────────────┐
│  Build query parameters:                     │
│  params = URLSearchParams({                  │
│    tenant_id: "...",                         │
│    datasource_id: "...",                     │
│    entity_ids: "uuid-here",  ← PREFERRED     │
│    page: "1",                                │
│    limit: "100"                              │
│  })                                          │
└──────────────────────────────────────────────┘
           ↓
┌──────────────────────────────────────────────┐
│  Frontend calls:                             │
│  GET /api/validation-rules?...entity_ids...  │
└──────────────────────────────────────────────┘
           ↓
┌──────────────────────────────────────────────┐
│  Backend parsing:                            │
│  - Extracts entity_ids = "uuid-here"        │
│  - Builds WHERE clause:                      │
│    ARRAY['uuid-here']::uuid[]                │
│    &&                                        │
│    COALESCE(target_entity_ids, ARRAY[]) │
└──────────────────────────────────────────────┘
           ↓
┌──────────────────────────────────────────────┐
│  Database executes:                          │
│  SELECT * FROM catalog_validation_rules      │
│  WHERE tenant_id = $1                        │
│    AND datasource_id = $2                    │
│    AND ARRAY[$1]::uuid[]                     │
│        &&                                    │
│        COALESCE(target_entity_ids,          │
│                 ARRAY[]::uuid[])            │
└──────────────────────────────────────────────┘
           ↓
┌──────────────────────────────────────────────┐
│  Results returned to frontend with:          │
│  - Only rules for employee entity            │
│  - Both legacy (name) and new (UUID) fields  │
└──────────────────────────────────────────────┘
           ↓
┌──────────────────────────────────────────────┐
│  Frontend renders validation rules           │
└──────────────────────────────────────────────┘
```

### C. Fallback to Name-Based Filtering

```
┌──────────────────────────────────────┐
│  Entity resolution failed OR          │
│  Entity UUID not available            │
└──────────────────────────────────────┘
           ↓
┌──────────────────────────────────────────────┐
│  getEntityId() returns undefined:            │
│  entityId = getEntityId("employee")          │
│  → undefined                                 │
└──────────────────────────────────────────────┘
           ↓
┌──────────────────────────────────────────────┐
│  Build query parameters with fallback:       │
│  params = URLSearchParams({                  │
│    tenant_id: "...",                         │
│    datasource_id: "...",                     │
│    entities: "employee",  ← FALLBACK         │
│    page: "1",                                │
│    limit: "100"                              │
│  })                                          │
└──────────────────────────────────────────────┘
           ↓
┌──────────────────────────────────────────────┐
│  Backend parsing:                            │
│  - No entity_ids provided                    │
│  - Falls back to entities parameter          │
│  - Builds WHERE clause:                      │
│    ARRAY['employee']::text[]                 │
│    &&                                        │
│    COALESCE(target_entities,                 │
│             ARRAY[target_entity])            │
└──────────────────────────────────────────────┘
           ↓
┌──────────────────────────────────────────────┐
│  Database executes name-based query          │
│  (Backward compatible with existing rules)   │
└──────────────────────────────────────────────┘
```

## Query Execution Examples

### UUID-Based Query (New - Preferred)
```sql
-- Frontend sends: entity_ids=22222222-2222-2222-2222-222222222222
-- Backend executes:

SELECT id, tenant_id, datasource_id, rule_name, rule_type, 
       description, target_entity, target_entity_id, 
       target_entities, target_entity_ids, condition_json, 
       severity, is_active, is_core, created_by, created_at, updated_at
FROM catalog_validation_rules
WHERE tenant_id = $1
  AND datasource_id = $2
  AND ARRAY['22222222-2222-2222-2222-222222222222']::uuid[]
      &&
      COALESCE(target_entity_ids, ARRAY[]::uuid[])
ORDER BY rule_name
LIMIT 100 OFFSET 0;

-- Uses GIN index on target_entity_ids for fast execution
```

### Name-Based Query (Legacy - Fallback)
```sql
-- Frontend sends: entities=employee
-- Backend executes:

SELECT id, tenant_id, datasource_id, rule_name, rule_type, 
       description, target_entity, target_entity_id, 
       target_entities, target_entity_ids, condition_json, 
       severity, is_active, is_core, created_by, created_at, updated_at
FROM catalog_validation_rules
WHERE tenant_id = $1
  AND datasource_id = $2
  AND ARRAY['employee']::text[]
      &&
      COALESCE(target_entities, ARRAY[target_entity])
ORDER BY rule_name
LIMIT 100 OFFSET 0;

-- Uses standard text array index, still efficient
```

## Index Strategy

```
┌────────────────────────────────────────────────┐
│  catalog_validation_rules Indexes              │
├────────────────────────────────────────────────┤
│  PRIMARY: id (UUID)                            │
│  BTREE: idx_validation_rules_tenant            │
│         → for tenant_id filtering              │
│  BTREE: idx_validation_rules_datasource        │
│         → for datasource_id filtering (NEW)    │
│  BTREE: idx_validation_rules_type              │
│         → for rule_type filtering              │
│  BTREE: idx_validation_rules_entity            │
│         → for target_entity filtering (legacy) │
│  BTREE: idx_validation_rules_entity_id (NEW)   │
│         → for target_entity_id filtering       │
│  GIN:   idx_validation_rules_entity_ids (NEW)  │
│         → for target_entity_ids array overlap  │
│  JSONB: idx_validation_rules_condition         │
│         → for condition_json filtering         │
└────────────────────────────────────────────────┘
```

## Component Lifecycle

```
┌─────────────────────────────────────────────────────────┐
│  EntityDetailsPage Component                            │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  useEffect (on mount):                                  │
│  ├─ useEntityResolution(tenant.id, datasource.id)      │
│  │  └─ Fetches entity mappings from backend             │
│  │  └─ Caches in hook state                             │
│  └─ fetchValidationRules()                              │
│     ├─ Gets entity UUID via getEntityId()               │
│     ├─ Calls API with entity_ids parameter              │
│     └─ Updates validationRules state                    │
│                                                          │
│  Render:                                                │
│  ├─ ValidationRulesContainer component                  │
│  │  └─ Receives filtered rules as prop                  │
│  │  └─ Displays by category (global/direct/mixed)       │
│  └─ RuleCard components                                 │
│     └─ Display individual rule details                  │
│                                                          │
│  useEffect (when tab changes):                          │
│  └─ If tab === 'validations'                            │
│     └─ Refetch rules (with cached entity IDs)           │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

## Error Handling Flow

```
┌──────────────────────────────┐
│ Missing tenant/datasource    │
└──────────────────────────────┘
           ↓
┌──────────────────────────────────────────────┐
│  Entity resolution skipped:                  │
│  - Check: if (!tenantId || !datasourceId)    │
│  - Return early with empty entityMap         │
│  - Frontend renders empty state              │
└──────────────────────────────────────────────┘

┌──────────────────────────────┐
│ Entity resolution fails       │
└──────────────────────────────┘
           ↓
┌──────────────────────────────────────────────┐
│  Hook sets error state:                      │
│  - setError(errorMsg)                        │
│  - getEntityId() returns undefined           │
│  - Validation rules fetch falls back to      │
│    name-based filtering                      │
│  - Rules still display (via fallback)        │
└──────────────────────────────────────────────┘

┌──────────────────────────────┐
│ Validation rules fetch fails  │
└──────────────────────────────┘
           ↓
┌──────────────────────────────────────────────┐
│  Catch block handles:                        │
│  - devError() logs to console                │
│  - Validation rules remain empty             │
│  - User sees "No validation rules" message   │
└──────────────────────────────────────────────┘
```

## Performance Characteristics

```
┌─────────────────────────────────┐
│  Operation                      │  Complexity
├─────────────────────────────────┼───────────────────────┐
│ Get entity resolutions          │  O(n) where n = # of  │
│ - Query fabric_defn by tenant   │    entities (fast due  │
│                                 │    to index on        │
│                                 │    tenant_datasource) │
│                                 │                       │
│ Get validation rules by UUID    │  O(log m) where m =   │
│ - Index lookup: GIN array index │    # of rules (very   │
│ - Array overlap: &&             │    fast with GIN)     │
│                                 │                       │
│ Get validation rules by name    │  O(log m) where m =   │
│ - Index lookup: B-tree array    │    # of rules (fast   │
│                                 │    but slower than    │
│                                 │    UUID due to string │
│                                 │    comparison)        │
│                                 │                       │
│ Parse response & render         │  O(r) where r =       │
│ - Filter client-side            │    # of rules         │
│ - Render components             │    returned           │
│                                 │                       │
└─────────────────────────────────────────────────────────┘
```

## Summary

- **Frontend** requests entity IDs via resolution endpoint
- **Backend** resolves entity keys to fabric_defn UUIDs
- **Validation Rules API** accepts both UUID and name filters
- **Database** queries with array overlap operators using indexes
- **Fallback** to name-based filtering ensures backward compatibility
- **Result**: Rules survive entity name changes, UUID-based linking is future-proof
