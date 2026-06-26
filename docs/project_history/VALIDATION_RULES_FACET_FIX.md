# Validation Rules Facet Filtering Fix

## Problem
Facet filtering was not working correctly - when clicking on facets to filter (e.g., selecting "Supplier" entity), the rules were not being filtered properly.

## Root Cause
The frontend was sending filter parameters as comma-separated values in a single parameter:
```
target_entity=Supplier,Customer,Product
```

While the backend was parsing multiple URL parameters:
```
target_entity=Supplier&target_entity=Customer&target_entity=Product
```

This mismatch caused the filtering logic to break because the backend would try to split the comma-separated values and process them incorrectly.

## Solution

### Frontend Changes (`ValidationRulesWithFacets.tsx`)
Changed the `buildFilterQuery` function to send each filter value as a separate URL parameter instead of combining them:

**Before:**
```typescript
if (filters.selectedEntities.length > 0) {
  params.append('target_entity', filters.selectedEntities.join(','));
}
```

**After:**
```typescript
if (filters.selectedEntities.length > 0) {
  filters.selectedEntities.forEach(entity => {
    params.append('target_entity', entity);
  });
}
```

This applies to all filter types:
- `target_entity` (entity filters)
- `rule_type` (rule type filters)
- `severity` (severity filters)
- `scope` (scope filters)
- `type` (core/custom filters)

### Backend Changes (`validation_rules_routes.go`)
Simplified the entity filtering logic to handle the already-parsed array directly:

**Before:**
```go
entity := r.URL.Query().Get("entity")
queryEntities := entity
if queryEntities == "" && len(targetEntities) > 0 {
  queryEntities = strings.Join(targetEntities, ",")
}
// Then split it again...
entityList := strings.Split(queryEntities, ",")
```

**After:**
```go
// targetEntities is already an array from multiple URL parameters
if len(targetEntities) > 0 {
  placeholders := make([]string, len(targetEntities))
  for i, e := range targetEntities {
    placeholders[i] = "$" + fmt.Sprintf("%d", argNum)
    args = append(args, strings.TrimSpace(e))
    argNum++
  }
  // Use the placeholders directly in the WHERE clause
}
```

Removed the unused `entity` variable that was causing compilation errors.

## How It Works Now

### Example Request
When user selects "Supplier" and "Customer" entities:
```
GET /api/validation-rules?tenant_id=<uuid>&page=1&target_entity=Supplier&target_entity=Customer
```

### Backend Processing
1. Parses `r.URL.Query()["target_entity"]` which returns `[]string{"Supplier", "Customer"}`
2. Creates placeholders for SQL: `$2, $3`
3. Adds to args: `["<uuid>", "Supplier", "Customer"]`
4. Generates WHERE clause that matches rules with either entity:
```sql
WHERE tenant_id = $1 
  AND ('global' = ANY(COALESCE(target_entities, ARRAY['global'])) 
    OR target_entity = ANY(ARRAY[$2,$3]) 
    OR EXISTS (SELECT 1 FROM unnest(COALESCE(target_entities, ARRAY[target_entity])) AS t WHERE t = ANY(ARRAY[$2,$3])))
```

## Facet Counts
- Facet counts are calculated from **all rules** (only tenant filter applied)
- Filtering rules doesn't change facet counts
- Users always see the full set of available options in facets

## Testing
To test the facet filtering:

1. Start the backend: `cd backend && PORT=29080 go run ./cmd/server/main.go`
2. Build frontend: `cd frontend && npm run build`
3. Open the validation rules page
4. Click on an entity in the facet sidebar
5. Rules should filter to show only those matching the selected entity
6. Facet counts should remain stable

## Verification Checklist
- ✅ Frontend sends multiple URL parameters (not comma-separated)
- ✅ Backend correctly parses multiple URL parameters into array
- ✅ Entity filtering works correctly
- ✅ Rule type filtering works correctly
- ✅ Severity filtering works correctly
- ✅ Facet counts are stable
- ✅ Rule type chip displays (rule_type shown on each rule)
- ✅ Edit modal opens when clicking pencil icon
