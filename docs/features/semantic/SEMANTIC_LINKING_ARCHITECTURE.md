# Entity-to-Semantic Linking - Visual Architecture

## Complete Data Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          FRONTEND APPLICATION                              │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                    GET /api/business-entities
                   (with tenant headers)
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                        BACKEND API ENDPOINT                                │
│  getBusinessEntities() - Query entity_attribute table                      │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                    SELECT catalog_node_id FROM entity_attribute
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                       DATABASE TABLES                                       │
│                                                                             │
│  entity_attribute (stores individual entities)                            │
│  ┌──────────┬──────────┬────────────────────┬──────────────┐              │
│  │ id       │ entity_key│ name               │catalog_node_id│              │
│  ├──────────┼──────────┼────────────────────┼──────────────┤              │
│  │ ent-123  │ order    │ Order              │ sema-111 ────────────┐      │
│  │ ent-124  │ rush_ord │ Rush Order         │ sema-222 ─┐         │      │
│  │ ent-125  │ std_ord  │ Standard Order     │ sema-333 ─┼─────┐  │      │
│  └──────────┴──────────┴────────────────────┴──────────────┘  │  │      │
│                                                               │  │      │
│  catalog_node (semantic term definitions)                    │  │      │
│  ┌──────────┬───────┬──────────────────┬──────────────┐     │  │      │
│  │ id       │ name  │ display_name     │ description  │     │  │      │
│  ├──────────┼───────┼──────────────────┼──────────────┤     │  │      │
│  │ sema-111◄┘       │ Order            │ Core order   │     │  │      │
│  │ sema-222◄────────┘ Rush Order       │ Fast order   │     │  │      │
│  │ sema-333◄────────────────────────────┘ Std order   │     │  │      │
│  └──────────┴───────┴──────────────────┴──────────────┘     │  │      │
│                                                               │  │      │
└─────────────────────────────────────────────────────────────────┼──────────┘
                                                                   │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                    JSON RESPONSE (Include catalogNodeId)                    │
│                                                                             │
│ {                                                                           │
│   "order": {                                                                │
│     "key": "order",                                                         │
│     "name": "Order",                                                        │
│     "catalogNodeId": "sema-111",  ◄─────────────────────┐                 │
│     "subtypes": {                                        │                 │
│       "rush_order": {                                    │ LINKS BACK     │
│         "key": "rush_order",                            │ TO SEMANTIC    │
│         "catalogNodeId": "sema-222"  ◄──────────────────┤ TERMS          │
│       }                                                  │                 │
│     }                                                    │                 │
│   }                                                      │                 │
│ }                                                        │                 │
└───────────────────────────────────────────────────────────┼─────────────────┘
                                                            │
                    Frontend can now:
                    ✅ Get semantic term details
                    ✅ Link to semantic term admin page
                    ✅ Validate entity matches semantic definition
                    ✅ Build breadcrumbs/navigation
```

---

## Data Structure Relationships

### Before (Problem)

```
Entity Response
└─ key: "order"
└─ name: "Order"               ⚠️  String reference (can become stale)
└─ businessName: "..."
└─ subtypes: {...}

Problem: Can't link back to semantic term!
❌ If name changes, entity reference breaks
```

### After (Solution)

```
Entity Response
├─ key: "order"
├─ name: "Order"
├─ catalogNodeId: "550e8400..."  ✅ UUID reference (immutable)
└─ subtypes:
   └─ rush_order
      ├─ key: "rush_order"
      ├─ name: "Rush Order"
      └─ catalogNodeId: "550e8400..."  ✅ Each subtype has its own link

Can now look up semantic term:
SELECT * FROM catalog_node WHERE id = '550e8400...'
✅ Always works, even if name changes
```

---

## Foreign Key Relationship

```
┌─────────────────────────────┐
│      entity_attribute       │
├─────────────────────────────┤
│ id (PK)                     │
│ entity_key                  │
│ name                        │
│ catalog_node_id (FK) ───────┼──────┐
│ parent_id (FK self-ref)     │      │
│ tenant_id (FK)              │      │
│ tenant_datasource_id (FK)   │      │
│ ...                         │      │
└─────────────────────────────┘      │
                                     │ FK Constraint
                                     │ ON DELETE SET NULL
                                     │
                                     ▼
                        ┌─────────────────────────────┐
                        │      catalog_node          │
                        ├─────────────────────────────┤
                        │ id (PK) ◄────────────────────
                        │ name                        │
                        │ display_name                │
                        │ description                 │
                        │ version                     │
                        │ is_active                   │
                        │ ...                         │
                        └─────────────────────────────┘
```

**Constraint Details:**
```
CONSTRAINT entity_attribute_catalog_node_fk 
  FOREIGN KEY (catalog_node_id) 
  REFERENCES public.catalog_node(id) 
  ON DELETE SET NULL 
  ON UPDATE CASCADE
```

**What This Means:**
- ✅ Entity can only reference valid semantic terms
- ✅ If semantic term is deleted, entity still exists (catalog_node_id becomes NULL)
- ✅ If semantic term ID changes, entity reference updates automatically
- ✅ No orphaned references possible

---

## Query Patterns

### Pattern 1: Get Entity with Semantic Term Details

```sql
-- Join entity with its semantic term definition
SELECT 
    ea.id as entity_id,
    ea.entity_key,
    ea.name as entity_name,
    ea.business_name,
    
    -- Semantic term details
    cn.id as semantic_term_id,
    cn.name as semantic_name,
    cn.display_name,
    cn.description,
    cn.version,
    cn.is_active,
    
    -- Relationship info
    ea.catalog_node_id,
    ea.parent_id,
    ea.is_core
FROM public.entity_attribute ea
LEFT JOIN public.catalog_node cn ON ea.catalog_node_id = cn.id
WHERE ea.tenant_datasource_id = $1
ORDER BY ea.parent_id NULLS FIRST, ea.entity_key;
```

### Pattern 2: Find All Entities for a Semantic Term

```sql
-- Reverse lookup: Find entities linked to a semantic term
SELECT 
    entity_key,
    name,
    business_name,
    is_core,
    parent_id
FROM public.entity_attribute
WHERE catalog_node_id = $1
  AND tenant_datasource_id = $2
ORDER BY entity_key;
```

### Pattern 3: Find Orphaned Entities (Missing Semantic Term)

```sql
-- Identify entities without semantic term link
SELECT 
    id,
    entity_key,
    name
FROM public.entity_attribute
WHERE catalog_node_id IS NULL
  AND tenant_datasource_id = $1;
```

### Pattern 4: Validate Entity Integrity

```sql
-- Ensure entity matches semantic term properties
SELECT 
    ea.entity_key,
    ea.name as entity_name,
    cn.display_name as semantic_name,
    CASE 
        WHEN ea.name != cn.display_name THEN 'NAME_MISMATCH'
        WHEN cn.is_active = false THEN 'INACTIVE_TERM'
        ELSE 'OK'
    END as status
FROM public.entity_attribute ea
LEFT JOIN public.catalog_node cn ON ea.catalog_node_id = cn.id
WHERE ea.tenant_datasource_id = $1
  AND ea.catalog_node_id IS NOT NULL;
```

---

## API Response Structure

### Complete Response Example

```json
{
  "order": {
    "key": "order",
    "name": "Order",
    "isCore": true,
    "catalogNodeId": "550e8400-e29b-41d4-a716-446655440000",
    "businessName": "Customer Order",
    "technicalName": "orders",
    "subtypes": {
      "rush_order": {
        "key": "rush_order",
        "name": "Rush Order",
        "isCore": false,
        "catalogNodeId": "550e8400-e29b-41d4-a716-446655440001",
        "businessName": "Rush Order",
        "subtypes": {}
      },
      "standard_order": {
        "key": "standard_order",
        "name": "Standard Order",
        "isCore": false,
        "catalogNodeId": "550e8400-e29b-41d4-a716-446655440002",
        "businessName": "Standard Order",
        "subtypes": {}
      }
    }
  },
  "payment": {
    "key": "payment",
    "name": "Payment",
    "isCore": true,
    "catalogNodeId": "550e8400-e29b-41d4-a716-446655440003",
    "businessName": "Payment Method",
    "subtypes": {}
  }
}
```

**Key Points:**
- ✅ Every entity has `catalogNodeId` (or omitted if NULL)
- ✅ Subtypes inherit the same structure
- ✅ `catalogNodeId` is always a UUID
- ✅ Can be used to fetch semantic term details

---

## Usage Flow Diagram

```
┌──────────────────────────┐
│  Frontend Application    │
└──────────────────────────┘
            │
            │ 1. GET /api/business-entities
            │    (with tenant headers)
            ▼
┌──────────────────────────────────────────┐
│  Parse JSON Response                     │
│  Extract entity.catalogNodeId = UUID     │
└──────────────────────────────────────────┘
            │
            │ 2. Have UUID of semantic term
            │
            ▼
        ┌─────────────┬──────────────────┐
        │             │                  │
        ▼             ▼                  ▼
    Display Link  Query Term Details  Validation
    to Term Page  via API             Check
        │             │                  │
        │             │                  │
        │   GET /api/catalog/  │  Ensure entity
        │   nodes/UUID         │  matches term
        │             │                  │
        │             ▼                  ▼
        │     Semantic Term          Pass/Fail
        │     Details               Check
        │     ├─ name
        │     ├─ display_name
        │     ├─ description
        │     ├─ version
        │     └─ is_active
        │
        └─────────────┬──────────────────┘
                      │
                      ▼
            ┌─────────────────────┐
            │  Updated UI/State   │
            └─────────────────────┘
```

---

## Type Definitions

### Go Structs

```go
// Entity stored in database
type BusinessEntity struct {
    ID                 string         // UUID
    TenantID           string         // UUID
    TenantDatasourceID string         // UUID
    ParentID           sql.NullString // UUID (nullable)
    CatalogNodeID      sql.NullString // UUID (nullable) ✅ LINK TO SEMANTIC
    Key                string         // entity key
    Name               string         // display name
    IsCore             bool           // is core entity
    BusinessName       sql.NullString // business name
    TechnicalName      sql.NullString // technical name
}

// Response sent to frontend
type BusinessEntityResponse struct {
    Key            string                            // entity key
    Name           string                            // display name
    IsCore         bool                              // is core entity
    CatalogNodeID  string                            // ✅ UUID REFERENCE
    BusinessName   string                            // business name
    TechnicalName  string                            // technical name
    Subtypes       map[string]BusinessEntityResponse // child entities
}
```

### TypeScript Types

```typescript
interface BusinessEntity {
  key: string;
  name: string;
  isCore: boolean;
  catalogNodeId?: string;  // ✅ UUID reference to semantic term
  businessName?: string;
  technicalName?: string;
  subtypes?: Record<string, BusinessEntity>;
}

interface CatalogNode {
  id: string;
  name: string;
  displayName: string;
  description: string;
  version: number;
  isActive: boolean;
}
```

---

## Testing

### Test Case 1: Verify catalogNodeId in Response

```javascript
// GET /api/business-entities
// Verify response includes catalogNodeId

test('Entity response includes catalogNodeId', async () => {
  const response = await getBusinessEntities(tenantId, datasourceId);
  
  expect(response.order).toBeDefined();
  expect(response.order.catalogNodeId).toBeDefined();
  expect(response.order.catalogNodeId).toMatch(/^[0-9a-f-]{36}$/); // UUID format
  
  expect(response.order.subtypes.rush_order).toBeDefined();
  expect(response.order.subtypes.rush_order.catalogNodeId).toBeDefined();
});
```

### Test Case 2: Verify catalogNodeId Links to Valid Semantic Term

```javascript
test('catalogNodeId references valid semantic term', async () => {
  const entities = await getBusinessEntities(tenantId, datasourceId);
  const order = entities.order;
  
  const semanticTerm = await getCatalogNode(order.catalogNodeId);
  expect(semanticTerm).toBeDefined();
  expect(semanticTerm.id).toBe(order.catalogNodeId);
});
```

### Test Case 3: Verify Database FK Constraint

```sql
-- This should FAIL (invalid catalog_node_id)
INSERT INTO public.entity_attribute 
(tenant_id, tenant_datasource_id, entity_key, name, is_core, catalog_node_id)
VALUES (
  'abc-123',
  'def-456',
  'test',
  'Test Entity',
  true,
  '99999999-9999-9999-9999-999999999999'
)
-- ERROR: insert or update on table "entity_attribute" violates foreign key constraint
```

---

## Summary

| Aspect | Value |
|--------|-------|
| **Field Name** | `catalogNodeId` |
| **Field Type** | UUID (nullable) |
| **In JSON** | ✅ Included (omitempty if NULL) |
| **References** | `catalog_node.id` (FK constraint) |
| **Usage** | Link entity to semantic term definition |
| **Immutability** | ✅ UUID never changes |
| **Validation** | ✅ Database FK prevents invalid refs |
| **Frontend Access** | Direct in JSON response |

**Result:** Entities now have a strong, traceable link to semantic terms that can be used for all purposes!
