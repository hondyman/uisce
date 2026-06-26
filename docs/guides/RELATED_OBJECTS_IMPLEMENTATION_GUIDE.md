# Related Objects Tab - Complete Implementation Guide

## What Was Implemented

This guide captures the complete implementation of the "Related Objects" tab feature in EntityDetailsPage, allowing users to discover and visualize entity relationships based on database foreign keys and semantic term mappings.

## Architecture Overview

```
User navigates to Entity Details Page
            ↓
Related Objects tab (RelatedObjectsTab.tsx)
            ↓
Calls fetchRelatedObjects() API (relationships.ts)
            ↓
GET /api/relationships/objects (api.go handler)
            ↓
RelationshipDiscoveryService.DiscoverLinkableEntities()
            ↓
PostgreSQL query (relationships_discovery.go)
            ↓
Returns list of linkable entities
            ↓
Display in Card or Diagram view
```

## Files Created/Modified

### 1. Backend Discovery Service
**File:** `backend/internal/api/relationships_discovery.go`

```go
// Main service struct
type RelationshipDiscoveryService struct {
    db *sql.DB
}

// Main discovery function
func (s *RelationshipDiscoveryService) DiscoverLinkableEntities(
    ctx context.Context,
    tenantID, datasourceID, entityName string,
) ([]RelatedEntity, error)
```

**Key Features:**
- Discovers all entities that can be linked via foreign keys
- Uses PostgreSQL CTEs for efficient multi-step discovery
- Returns typed `RelatedEntity` structs with cardinality info

### 2. Backend API Handler
**File:** `backend/internal/api/api.go`

**Function:** `getRelatedObjects()`
- Endpoint: `GET /api/relationships/objects`
- Query params: `tenant_id`, `datasource_id`, `entity`
- Uses RelationshipDiscoveryService
- Returns JSON with relationships array

### 3. Frontend API Service
**File:** `frontend/src/api/relationships.ts`

```typescript
// Fetch related entities
export async function fetchRelatedObjects(
  tenantId: string,
  datasourceId: string,
  entityName: string
): Promise<RelatedEntity[]>

// Types exported
export interface RelatedEntity {
  id: string;
  sourceEntity: string;
  targetEntity: string;
  cardinality: string;
  keyFields: { source: string; target: string };
  description?: string;
  edgeType?: string;
}
```

### 4. Updated Component
**File:** `frontend/src/components/relationship/RelatedObjectsTab.tsx`

**Features:**
- ✅ Card view showing relationships in grid
- ✅ Diagram view with SVG visualization
- ✅ Apply relationship button (creates new edge)
- ✅ Cardinality color coding
- ✅ Error/loading states
- ✅ Empty state messaging

## Discovery Algorithm

The algorithm discovers linkable entities by:

1. **Find Semantic Terms**: Get all semantic terms associated with the entity
   ```
   Query: WHERE business_term.node_name = entityName
          JOIN semantic_term via has_semantic edge
   ```

2. **Map to Columns**: Find columns mapped to those semantic terms
   ```
   Query: WHERE semantic_term links to column
          via MAPS_TO edge
   ```

3. **Find Source Tables**: Identify which tables contain those columns
   ```
   Query: SELECT DISTINCT table_id FROM mapped_columns
   ```

4. **Discover Foreign Keys**: Find all FKs from/to source tables
   ```
   Query: Find foreign_key edges where:
          - source_table IN source_tables OR
          - target_table IN source_tables
   ```

5. **Find Target Tables**: Collect tables on other side of FKs
   ```
   Query: SELECT target_table_id FROM foreign_keys
          UNION SELECT source_table_id FROM foreign_keys
   ```

6. **Find Entities**: Identify entities backed by target tables
   ```
   Query: Find semantic terms mapped to target tables
          Find business terms linked to those semantics
   ```

## Example Discovery Flow

```
Input: entityName = "Customer"

Step 1: Semantic Terms
- Found: "Customer Identifier", "CUSTOMER_ID", "Customer.ID"

Step 2: Mapped Columns  
- Found: customer_id (customers table)
         customer_id (customer_demographics table)

Step 3: Source Tables
- customers, customer_demographics

Step 4: Foreign Keys
- customers.id → NULL (primary key)
- customer_demographics.customer_id → customers.customer_id
- orders.customer_id → customers.customer_id

Step 5: Target Tables
- customer_demographics (from inbound FK)
- orders (from outbound FK)

Step 6: Entities
- CustomerDemographic (backed by customer_demographics)
- Order (backed by orders)

Output: Return [CustomerDemographic, Order] as linkable entities
```

## Database Query

The main discovery uses a single optimized PostgreSQL query with CTEs:

```sql
WITH selected_semantic AS (...)
WITH mapped_columns AS (...)
WITH source_tables AS (...)
WITH foreign_keys_outbound AS (...)
WITH foreign_keys_inbound AS (...)
WITH all_foreign_keys AS (...)
WITH target_tables AS (...)
WITH linked_semantic_terms AS (...)
WITH business_terms_for_targets AS (...)
SELECT ... FROM business_terms_for_targets
```

Benefits:
- Single database round-trip
- Efficient join strategy
- Handles both outbound and inbound FKs
- Properly scoped to tenant

## API Contract

### Request
```bash
GET /api/relationships/objects?tenant_id=xxx&datasource_id=yyy&entity=Customer

Headers:
  X-Tenant-ID: xxx
  X-Tenant-Datasource-ID: yyy
```

### Response
```json
{
  "sourceEntity": "Customer",
  "relationships": [
    {
      "id": "rel-1",
      "sourceEntity": "Customer",
      "targetEntity": "Order",
      "cardinality": "one-to-many",
      "keyFields": {
        "source": "Customer(ID)",
        "target": "Order(customer_id)"
      },
      "description": "Linked via has_semantic: Can be linked via foreign key",
      "edgeType": "has_semantic",
      "tableName": "orders",
      "semanticName": "ORDER_ID"
    }
  ],
  "count": 5
}
```

## Component Integration

In `EntityDetailsPage.tsx`, the tab is already integrated:

```tsx
{
  key: 'related',
  label: '🔗 Related Objects',
  children: (
    <RelatedObjectsTab
      tenantId={tenant.id}
      datasourceId={datasource.id || datasource.alpha_datasource_id}
      entityName={entity.businessName || entity.name}
    />
  ),
}
```

No changes needed to EntityDetailsPage!

## UI Components

### Card View
- Grid layout (1-3 columns responsive)
- Each card shows:
  - Target entity name
  - Cardinality badge (color-coded)
  - Key field mappings
  - Description
  - Action buttons (apply, edit)
- Empty state with "No relationships yet" message

### Diagram View
- Central node: Source entity (blue)
- Surrounding nodes: Target entities (white with border)
- SVG lines with arrow markers showing connections
- Circular distribution algorithm for node placement
- Interactive hover effects

### Cardinality Badges
```
One-to-One    → Green  (#50E3C2)
One-to-Many   → Orange (#F5A623)
Many-to-One   → Blue   (#4A90E2)
Many-to-Many  → Purple (#9B59B6)
```

## Error Handling

### Backend
- Missing params → 400 Bad Request
- Database error → 500 Internal Server Error
- No relationships → 200 OK with empty array

### Frontend
- Network error → "Error fetching relationships"
- 404 response → "Endpoint not found"
- Parse error → Logged, defaults to empty state

### User Feedback
- Loading spinner during fetch
- Error message box with troubleshooting tips
- Empty state message when no relationships found

## Tenant Scoping

Enforced at multiple levels:

1. **Frontend (setupTenantFetch.ts)**
   - Patches fetch() to add tenant headers
   - Blocks requests without tenant scope

2. **API Handler (api.go)**
   - Validates tenant_id matches X-Tenant-ID header
   - Queries filtered by tenant_datasource_id

3. **Database**
   - All queries filtered by tenant_datasource_id column
   - Proper row-level security

## Testing Checklist

- [ ] Backend query returns correct relationships for test entity
- [ ] Frontend renders Card view without errors
- [ ] Frontend renders Diagram view without errors
- [ ] Tab loads without data before tenant selection
- [ ] Tab loads with data after tenant selection
- [ ] Apply button creates new relationship edges
- [ ] View toggle switches between card/diagram
- [ ] Error handling works for invalid entity names
- [ ] Empty state displays when no relationships found
- [ ] Cardinality badges display correct colors
- [ ] Responsive design works on mobile
- [ ] Dark mode styling works correctly

## Performance Tips

1. **Indexes to Add:**
   ```sql
   CREATE INDEX idx_catalog_edge_tenant_src_tgt 
   ON catalog_edge(tenant_datasource_id, source_node_id, target_node_id);
   
   CREATE INDEX idx_catalog_edge_tenant_rel_type
   ON catalog_edge(tenant_datasource_id, relationship_type);
   ```

2. **Caching:** Frontend could memoize relationships for re-visits

3. **Pagination:** Add limit/offset for entities with many relationships

## Troubleshooting

**Problem:** "Relationships endpoint not found"
- **Solution:** Verify backend is running and /api/relationships/objects is registered

**Problem:** Tab shows loading indefinitely
- **Solution:** Check browser console for fetch errors; verify tenant scope is selected

**Problem:** No relationships returned for known FKs
- **Solution:** Verify semantic terms are mapped to columns; check catalog_edge records

**Problem:** Cardinality looks wrong
- **Solution:** Check FK direction in discovery algorithm; may need to adjust inference logic

## Future Enhancements

1. **Relationship Details Modal**: Click to see more info about each relationship
2. **Bidirectional Navigation**: Links to target entity details
3. **Filtering**: Filter by cardinality, type, or recency
4. **Batch Import**: Import multiple relationships at once
5. **Relationship Editor**: Customize relationship metadata
6. **Suggestions**: ML-based suggestions for missing relationships
7. **Audit Trail**: See who created each relationship
8. **Strength Scoring**: Show confidence/strength of discovered relationships

## Summary

The Related Objects tab provides a complete solution for discovering, visualizing, and managing entity relationships based on database structure and semantic metadata. It works seamlessly with the existing tenant-scoped architecture and provides both technical accuracy (via FK analysis) and user-friendly visualization.
