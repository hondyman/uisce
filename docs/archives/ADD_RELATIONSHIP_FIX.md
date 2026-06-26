# Add Relationship Fix - Complete Implementation

## Problem Statement

The "Add New Relationship" feature was not working properly:
1. Users couldn't click buttons to add discovered relationships
2. The Apply button had unclear UI/UX
3. The backend wasn't receiving data in the expected format
4. Error feedback was minimal

## Solution Overview

Fixed three layers:

### 1. **Frontend API Client** (`frontend/src/api/relationships.ts`)

**Issue:** Request body was sending snake_case field names but backend expected camelCase.

**Fix:**
```typescript
// Before (wrong):
body: JSON.stringify({
  source_entity: sourceEntity,      // ❌ snake_case
  target_entity: targetEntity,      // ❌ snake_case
  relationship_type: relationshipType,
})

// After (correct):
body: JSON.stringify({
  tenantId: tenantId,              // ✅ camelCase
  datasourceId: datasourceId,      // ✅ camelCase
  sourceEntity: sourceEntity,      // ✅ camelCase
  targetEntity: targetEntity,      // ✅ camelCase
  edgeType: relationshipType,
  cardinality: cardinality,
  fkColumn: '',
  confidence: 0.8,
})
```

**Changed:**
- Updated request body to match backend struct tags
- Added `cardinality` parameter to function signature
- Added required fields: `tenantId`, `datasourceId`, `confidence`
- Better error handling with HTTP status checks

### 2. **Component UI Improvements** (`frontend/src/components/relationship/RelatedObjectsTab.tsx`)

**Issue:** 
- Apply button was unclear (small icon-only button)
- "No relationships defined yet" message didn't explain why
- No loading/applying state feedback

**Fixes:**
```tsx
// Before:
<button className="...w-8 h-8...">  {/* Small circle button */}
  <span className="material-symbols-outlined text-xl">
    {rel.isApplied ? 'check_circle' : 'link'}
  </span>
</button>

// After:
<button className="...px-3 py-2...">  {/* Larger button with text */}
  <span className="material-symbols-outlined text-lg">
    {rel.isApplied ? 'check_circle' : _applyingRelationshipId === rel.id ? 'hourglass_empty' : 'link'}
  </span>
  <span>
    {rel.isApplied ? 'Applied' : _applyingRelationshipId === rel.id ? 'Applying...' : 'Apply'}
  </span>
</button>
```

**Changes:**
- Made Apply button larger with visible text label
- Added "Applying..." state during submission
- Better color differentiation (blue for actionable, green for applied)
- Improved "no relationships" message with diagnostic hint
- Added alert feedback on success/failure

### 3. **Backend Handler** (`backend/internal/api/api.go`)

**Issues:**
- Not validating tenant/datasource exists
- Hardcoded incorrect table name (`catalog_edge_types` vs `catalog_edge_type`)
- No proper logging or error context
- Missing RETURNING clause to confirm edge creation

**Fixes:**
```go
// Validation:
if req.TenantID == "" || req.DatasourceID == "" || req.SourceEntity == "" || req.TargetEntity == "" {
  http.Error(w, "Missing required fields: tenantId, datasourceId, sourceEntity, targetEntity", http.StatusBadRequest)
  return
}

// Verify tenant + datasource:
var tenantDatasourceID string
err := s.DB.QueryRow(
  `SELECT id FROM catalog_datasource 
   WHERE id = $1 AND tenant_id = $2`,
  req.DatasourceID, req.TenantID,
).Scan(&tenantDatasourceID)

// Correct query with proper tenant scoping:
query := `
  INSERT INTO catalog_edge (
    tenant_datasource_id, source_node_id, target_node_id, edge_type_id, 
    relationship_type, cardinality, fk_column, confidence, suggested, created_by
  ) 
  SELECT $1, src.id, tgt.id, cet.id, $2, $3, $4, $5, true, 'user'
  FROM catalog_node src, catalog_node tgt, catalog_edge_type cet
  WHERE src.node_name = $6 
    AND src.tenant_datasource_id = $1     -- ✅ Tenant scoping
    AND tgt.node_name = $7 
    AND tgt.tenant_datasource_id = $1     -- ✅ Tenant scoping
    AND cet.edge_type_name = $8
  RETURNING id
`

// Capture edge ID for response:
var edgeID string
err = s.DB.QueryRow(query, ...).Scan(&edgeID)
```

**Changes:**
- Added full tenant scope validation
- Fixed table name typos
- Added RETURNING clause to confirm success
- Better error messages
- Set default values for optional fields

---

## File Changes Summary

### `frontend/src/api/relationships.ts`
- ✅ Updated `applyRelationship()` function
- ✅ Changed request body to use camelCase field names
- ✅ Added `cardinality` parameter to function signature
- ✅ Added all required fields: `tenantId`, `datasourceId`, `confidence`, `fkColumn`
- ✅ Better error handling and response parsing
- **Lines Changed:** 215-260

### `frontend/src/components/relationship/RelatedObjectsTab.tsx`
- ✅ Updated `handleApplyRelationship()` to pass cardinality
- ✅ Improved Apply button UI with text label
- ✅ Added "Applying..." loading state
- ✅ Enhanced empty state message
- ✅ Added alert feedback on success/failure
- **Lines Changed:** 67-93, 145-211

### `backend/internal/api/api.go`
- ✅ Added tenant validation in `applyRelationship()` handler
- ✅ Fixed table name from `catalog_edge_types` to `catalog_edge_type`
- ✅ Added tenant scoping to both node lookups
- ✅ Added RETURNING id clause
- ✅ Better error messages
- **Lines Changed:** 6421-6516

---

## User Experience Flow

### With Relationships Available

1. User navigates to Entity Details → Related Objects tab
2. System shows list of discoverable related entities as cards
3. Each card displays:
   - Target entity name
   - Cardinality badge (One-to-One, One-to-Many, etc.)
   - Key fields mapping (Source FK → Target PK)
   - Apply button (blue, with text "Apply")
4. User clicks "Apply" button
5. Button changes to "Applying..." with loading icon
6. On success: Button becomes green with "Applied" and checkmark
7. On error: Alert shows error message with details

### With No Relationships Available

1. User navigates to Related Objects tab
2. System shows message: **"No entities available to relate to"**
3. Helpful subtext: **"Verify that semantic terms are mapped to columns and foreign keys exist in the database."**
4. User can:
   - Check if semantic terms are defined
   - Verify semantic term → column mappings exist
   - Check if foreign keys exist in database schema

---

## Testing Checklist

### Test 1: Successfully Apply Relationship ✓

**Setup:**
- Entity with discoverable relationships (e.g., semantic terms → columns → FKs → other entities)
- Backend running and accessible
- Tenant scope selected

**Steps:**
1. Navigate to Related Objects tab
2. See list of available relationships
3. Click "Apply" button on a relationship card
4. Observe button state changes to "Applying..."
5. After ~1-2 seconds, button becomes green with "Applied"

**Expected Result:**
- ✅ Button updates correctly
- ✅ No errors in console (F12)
- ✅ Relationship edge appears in database

### Test 2: No Relationships Available

**Setup:**
- Entity with NO semantic terms or FK mappings
- Backend running

**Steps:**
1. Navigate to Related Objects tab
2. Wait for content to load

**Expected Result:**
- ✅ Shows "No entities available to relate to" message
- ✅ Shows diagnostic hint about semantic terms
- ✅ No error styling

### Test 3: Error Handling

**Setup:**
- Tenant scope not selected (clear localStorage)

**Steps:**
1. Click "Apply" button

**Expected Result:**
- ✅ Alert shows error message
- ✅ Button remains blue (not applied)
- ✅ Console shows error details

### Test 4: Multiple Relationships

**Setup:**
- Entity with 3+ discoverable relationships

**Steps:**
1. Apply first relationship → should turn green
2. Apply second relationship → should turn green
3. First relationship should remain green

**Expected Result:**
- ✅ Each button tracks its own state
- ✅ Can apply multiple independently

---

## Database Prerequisites

Verify these tables exist and have data:

```sql
-- 1. Catalog nodes (entities)
SELECT COUNT(*) as node_count 
FROM catalog_node 
WHERE catalog_type_name IN ('business_term', 'entity');

-- 2. Catalog edges (relationships/mappings)
SELECT COUNT(*) as edge_count 
FROM catalog_edge ce
JOIN catalog_edge_type cet ON ce.edge_type_id = cet.id
WHERE cet.predicate IN ('maps to', 'foreign_key');

-- 3. Edge types
SELECT COUNT(*) as edge_type_count 
FROM catalog_edge_type 
WHERE edge_type_name IN ('entity_relationship', 'foreign_key');

-- 4. Datasources
SELECT COUNT(*) as datasource_count 
FROM catalog_datasource;
```

---

## Configuration

### Field Mappings

| Frontend | Backend JSON | Backend Go Struct | Purpose |
|----------|--------------|-------------------|---------|
| `tenantId` | tenantId | TenantID | Identifies tenant |
| `datasourceId` | datasourceId | DatasourceID | Identifies data source |
| `sourceEntity` | sourceEntity | SourceEntity | Source entity name |
| `targetEntity` | targetEntity | TargetEntity | Target entity name |
| `rel.edgeType` | edgeType | EdgeType | Relationship type (default: "entity_relationship") |
| `rel.cardinality` | cardinality | Cardinality | FK cardinality (default: "One-to-Many") |
| `''` | fkColumn | FKColumn | Foreign key column name |
| `0.8` | confidence | Confidence | Confidence score (0.0-1.0) |

### Default Values

If not provided by frontend:
- `EdgeType`: "entity_relationship"
- `Cardinality`: "One-to-Many"
- `Confidence`: 0.8
- `FKColumn`: "" (empty string)

---

## Deployment Checklist

- [ ] Backend compiles without errors
- [ ] Frontend TypeScript compiles without errors
- [ ] Database tables exist and have proper schema
- [ ] Tenant and datasource selection working
- [ ] Test "Apply" button on an entity with relationships
- [ ] Verify edge was created in `catalog_edge` table
- [ ] Test error handling with invalid tenant/datasource
- [ ] Check browser console for any warnings/errors
- [ ] Verify button UI updates correctly on success

---

## Troubleshooting

### Apply button doesn't work

**Check:**
1. Backend running? `curl http://localhost:8080/health`
2. Tenant scope selected? Open DevTools → Application → localStorage
3. Entity has relationships? Check backend logs
4. Browser console errors? Press F12 and look for red messages

### Button stays in "Applying..." state

**Causes:**
- Network timeout
- Backend not responding
- Invalid tenant/datasource

**Fix:**
- Refresh page
- Check backend is running
- Verify tenant scope

### Edge not created in database

**Check:**
1. Run query to see if edge was created:
   ```sql
   SELECT * FROM catalog_edge 
   WHERE created_by = 'user' 
   ORDER BY created_at DESC LIMIT 1;
   ```

2. Check backend logs for SQL errors
3. Verify node names match exactly (case-sensitive)
4. Verify edge_type exists for relationship type

---

## Performance Optimization

For large numbers of relationships, consider:

```sql
-- Add indexes for faster lookups
CREATE INDEX idx_catalog_node_name_tenant 
ON catalog_node(node_name, tenant_datasource_id);

CREATE INDEX idx_catalog_edge_tenant_nodes 
ON catalog_edge(tenant_datasource_id, source_node_id, target_node_id);

CREATE INDEX idx_catalog_edge_type_name 
ON catalog_edge_type(edge_type_name);
```

---

## Related Documentation

- See `RELATED_OBJECTS_TROUBLESHOOTING.md` for general debugging
- See `RELATED_OBJECTS_IMPLEMENTATION_GUIDE.md` for discovery algorithm
- See agents.md for tenant scoping requirements

