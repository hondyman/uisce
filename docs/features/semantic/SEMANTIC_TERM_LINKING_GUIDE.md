# Semantic Term Linking - How to Reference catalog_node from Entities

## What Changed

The `BusinessEntityResponse` struct now includes `CatalogNodeID` field so you can reference back to semantic terms in `catalog_node` table.

### Before
```go
type BusinessEntityResponse struct {
    Key           string
    Name          string
    IsCore        bool
    BusinessName  string
    TechnicalName string
    Subtypes      map[string]BusinessEntityResponse
    // ❌ CatalogNodeID missing - can't reference semantic terms
}
```

### After
```go
type BusinessEntityResponse struct {
    Key            string
    Name           string
    IsCore         bool
    CatalogNodeID  string  // ✅ NEW: Link to semantic term
    BusinessName   string
    TechnicalName  string
    Subtypes       map[string]BusinessEntityResponse
}
```

---

## JSON Response Example

When you GET `/api/business-entities`, you now get:

```json
{
  "order": {
    "key": "order",
    "name": "Order",
    "isCore": true,
    "catalogNodeId": "550e8400-e29b-41d4-a716-446655440000",
    "businessName": "Customer Order",
    "subtypes": {
      "rush_order": {
        "key": "rush_order",
        "name": "Rush Order",
        "isCore": false,
        "catalogNodeId": "550e8400-e29b-41d4-a716-446655440001"
      },
      "standard_order": {
        "key": "standard_order",
        "name": "Standard Order",
        "isCore": false,
        "catalogNodeId": "550e8400-e29b-41d4-a716-446655440002"
      }
    }
  },
  "payment": {
    "key": "payment",
    "name": "Payment",
    "isCore": true,
    "catalogNodeId": "550e8400-e29b-41d4-a716-446655440003"
  }
}
```

**Key Change:** Each entity now has `catalogNodeId` to reference back to semantic terms!

---

## Using catalogNodeId to Link Back to Semantic Terms

### Query 1: Get Semantic Definition by Entity catalogNodeId

```sql
-- Use the catalogNodeId from entity response to find semantic term
SELECT 
    id,
    name,
    display_name,
    description
FROM public.catalog_node
WHERE id = '550e8400-e29b-41d4-a716-446655440000';
```

**Result:**
```
id                                   | name  | display_name | description
550e8400-e29b-41d4-a716-446655440000 | order | Order        | Core business entity for orders
```

### Query 2: Find All Entities Linked to a Semantic Term

```sql
-- Find all entities referencing a specific semantic term
SELECT 
    entity_key,
    name,
    business_name,
    is_core,
    parent_id
FROM public.entity_attribute
WHERE catalog_node_id = '550e8400-e29b-41d4-a716-446655440000'
  AND tenant_datasource_id = 'your-datasource-id';
```

### Query 3: Get Entity with Full Semantic Definition

```sql
-- Join entity_attribute with catalog_node to get complete picture
SELECT 
    ea.entity_key,
    ea.name as entity_name,
    ea.business_name,
    cn.display_name as semantic_term,
    cn.description,
    ea.catalog_node_id
FROM public.entity_attribute ea
LEFT JOIN public.catalog_node cn ON ea.catalog_node_id = cn.id
WHERE ea.tenant_datasource_id = 'your-datasource-id'
  AND ea.parent_id IS NULL;  -- Root entities only
```

---

## Frontend Usage Examples

### Example 1: Display Entity with Semantic Link

```javascript
// Get entities from API
const response = await fetch('/api/business-entities', {
  headers: {
    'X-Tenant-ID': tenantId,
    'X-Tenant-Datasource-ID': datasourceId
  }
});
const entities = await response.json();

// For each entity, you now have catalogNodeId
const order = entities.order;
console.log(order.catalogNodeId); // "550e8400-e29b-41d4-a716-446655440000"

// Use it to reference semantic term
const semanticLink = `/api/catalog/nodes/${order.catalogNodeId}`;
```

### Example 2: Build Link Back to Semantic Definition

```javascript
function getSemanticTermLink(entity) {
  if (!entity.catalogNodeId) {
    return null;
  }
  return {
    termId: entity.catalogNodeId,
    termName: entity.name,
    link: `/admin/semantic-terms/${entity.catalogNodeId}`
  };
}

// Usage
const order = entities.order;
const semanticLink = getSemanticTermLink(order);
// Result:
// {
//   termId: "550e8400-e29b-41d4-a716-446655440000",
//   termName: "Order",
//   link: "/admin/semantic-terms/550e8400-e29b-41d4-a716-446655440000"
// }
```

### Example 3: Validate Entity Matches Semantic Term

```javascript
async function validateEntityLink(entity, catalogNode) {
  // Check that entity.catalogNodeId matches the semantic term
  if (entity.catalogNodeId !== catalogNode.id) {
    throw new Error(
      `Entity "${entity.key}" catalogNodeId ${entity.catalogNodeId} ` +
      `does not match semantic term ${catalogNode.id}`
    );
  }
  return true;
}

// Usage
const order = entities.order;
const semanticTerm = await getSemanticTerm(order.catalogNodeId);
await validateEntityLink(order, semanticTerm);
```

---

## Database Reference

### entity_attribute table
```
Column              | Type              | Description
--------------------|-------------------|-------------------------------------------
id                  | uuid              | Entity identifier
entity_key          | text              | Entity key (e.g., "order")
name                | text              | Entity display name
is_core             | boolean           | Whether it's a core entity
business_name       | text              | Business-friendly name
technical_name      | text              | Technical name
catalog_node_id     | uuid (FK)         | ✅ LINK to semantic term
parent_id           | uuid (FK)         | Link to parent entity
tenant_id           | uuid (FK)         | Tenant
tenant_datasource_id| uuid (FK)         | Datasource
created_at          | timestamp         | Creation time
updated_at          | timestamp         | Last update time
```

### catalog_node table
```
Column              | Type              | Description
--------------------|-------------------|-------------------------------------------
id                  | uuid              | Semantic term identifier
name                | text              | Internal name
display_name        | text              | Display name
description         | text              | What this term means
type                | text              | Type of node
version             | integer           | Version number
is_active           | boolean           | Is this term active
created_at          | timestamp         | Creation time
updated_at          | timestamp         | Last update time
```

---

## Linking Flow

```
Frontend Entity Response
    │
    ├─ entity.key = "order"
    ├─ entity.name = "Order"
    └─ entity.catalogNodeId = "550e8400-e29b-41d4-a716-446655440000"  ✅ LINK!
        │
        └─ Query: SELECT * FROM catalog_node WHERE id = '550e8400-e29b-41d4-a716-446655440000'
            │
            └─ Semantic Definition
                ├─ name = "order"
                ├─ display_name = "Order"
                ├─ description = "Core business entity for customer orders"
                ├─ version = 2
                └─ is_active = true
```

---

## API Contract

### GET /api/business-entities

**Request:**
```bash
curl -H "X-Tenant-ID: abc-123" \
     -H "X-Tenant-Datasource-ID: def-456" \
     http://localhost:8080/api/business-entities
```

**Response (with catalogNodeId):**
```json
{
  "order": {
    "key": "order",
    "name": "Order",
    "isCore": true,
    "catalogNodeId": "550e8400-e29b-41d4-a716-446655440000",
    "businessName": "Customer Order",
    "subtypes": {
      "rush_order": {
        "key": "rush_order",
        "name": "Rush Order",
        "isCore": false,
        "catalogNodeId": "550e8400-e29b-41d4-a716-446655440001"
      }
    }
  }
}
```

**New Field:** `catalogNodeId` - UUID reference to semantic term in `catalog_node` table

---

## Benefits of catalogNodeId in Response

✅ **Direct Reference:** No string name matching needed
✅ **Immutable Link:** UUID never changes, semantic term name can be updated
✅ **Bidirectional Traceability:** Can find semantic term from entity or entity from semantic term
✅ **Integrity Checking:** Can validate entity matches its semantic definition
✅ **Frontend Integration:** Links directly to semantic term details or admin pages
✅ **Version Tracking:** Can track entity definition changes via semantic term versioning

---

## Summary

| Aspect | Before | After |
|--------|--------|-------|
| **Entity → Semantic Link** | String name (stale) | UUID catalogNodeId (immutable) |
| **In JSON Response** | No reference back | ✅ catalogNodeId included |
| **Frontend Usage** | Can't link back | Direct link via catalogNodeId |
| **Query by Semantic** | Not possible | WHERE catalogNodeId = ? |
| **Validation** | Manual string match | Database FK constraint |

**Result:** Entities now have a strong, immutable link to semantic terms that can be used for referencing, validation, and navigation.
