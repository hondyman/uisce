# Validation Rules Faceted Search - Complete Implementation Guide

## Overview
Successfully implemented a complete faceted search interface for validation rules with:
- Advanced filtering by multiple dimensions
- Stable facet counts
- Rule editing capabilities
- Proper chip indicators for rule metadata
- Clean, intuitive UI with pinned sidebar

## Architecture

### Frontend Components
**File:** `/frontend/src/components/ValidationRules/ValidationRulesWithFacets.tsx`
**Size:** ~686 lines

#### Key Features:
1. **Faceted Search**
   - Entity facets (Supplier, Customer, Product, etc.)
   - Rule type facets (field_format, cardinality, uniqueness, referential_integrity, business_logic)
   - Severity facets (error, warning, info)
   - Scope facets (global, specific)
   - Type facets (core, custom)

2. **Stable Facet Counts**
   - Uses `totalFacetCounts` state to maintain counts from unfiltered data
   - Counts don't change when other filters are applied
   - Only recalculated when no filters are active

3. **Rule Display**
   - Rule name, entity, and sub-entity information
   - Multiple chips showing:
     - 🌍 Global indicator
     - 🔒 Core / ✏️ Custom indicator
     - Rule type (Cardinality, Uniqueness, etc.)
   - Severity badge with icon
   - Description

4. **Actions**
   - ✎ Edit button (opens ValidationRuleEditor modal)
   - 📋 Copy button
   - 🗑 Delete button

5. **Search**
   - Real-time search with debouncing (300ms)
   - Searches rule name and description

### Backend API
**File:** `/backend/internal/api/validation_rules_routes.go`
**Endpoints:**
- `GET /validation-rules` - List rules with facets and pagination
- `POST /validation-rules` - Create new rule
- `GET /validation-rules/{id}` - Get single rule
- `PATCH /validation-rules/{id}` - Update rule
- `DELETE /validation-rules/{id}` - Delete rule
- `POST /validation-rules/{id}/execute` - Execute rule
- `POST /validation-rules/execute-batch` - Execute multiple rules
- `GET /validation-rules/{id}/audit` - Get audit history

#### Filter Parameters (Multi-value supported):
```
GET /api/validation-rules?
  tenant_id=<uuid>&
  page=1&
  limit=20&
  target_entity=Supplier&
  target_entity=Customer&
  rule_type=cardinality&
  rule_type=uniqueness&
  severity=error&
  severity=warning&
  search=keyword
```

#### Response Format:
```json
{
  "rules": [
    {
      "id": "uuid",
      "rule_name": "string",
      "rule_type": "string",
      "target_entity": "string",
      "target_entities": ["string"],
      "severity": "error|warning|info",
      "description": "string",
      "is_active": boolean,
      "is_core": boolean,
      "is_global": boolean,
      "created_at": "timestamp",
      "created_by": "string"
    }
  ],
  "total": number,
  "page": number,
  "limit": number,
  "has_more": boolean,
  "facets": {
    "rule_types": [{"value": "string", "count": number}],
    "severities": [{"value": "string", "count": number}],
    "entities": [{"value": "string", "count": number}]
  },
  "timestamp": "iso-8601"
}
```

### Editor Component
**File:** `/frontend/src/components/ValidationRules/ValidationRuleEditor.tsx`
**File:** `/frontend/src/components/ValidationRules/ValidationRuleEditor.css`

Modal-based editor for updating rules:
- Edit rule name
- Edit description
- Change severity
- Toggle active status
- Read-only display of rule type and target entity
- Proper error handling and loading states

## Implementation Details

### Facet Stability Solution

**Problem:** Facet counts changed when filters were applied, making the UX confusing.

**Solution:** Two-tier counting system:
1. **Total Facet Counts** - Cached from unfiltered data (only tenant filter)
2. **Filtered Rules** - Apply all selected filters to display results

```typescript
// Unfiltered facet query for stable counts
const facetWhereClause = "WHERE tenant_id = $1"

// Filtered query for displayed results
const whereClause = "WHERE tenant_id = $1 AND ..."
```

### Filter Parameter Handling

**Problem:** Frontend sent comma-separated values, backend expected multiple parameters.

**Solution:** Frontend now sends multiple parameters:
```typescript
filters.selectedEntities.forEach(entity => {
  params.append('target_entity', entity);  // Creates target_entity=A&target_entity=B
});
```

Backend receives as array:
```go
targetEntities := r.URL.Query()["target_entity"]  // []string{"A", "B"}
```

### Entity Filtering Logic

Complex SQL to handle multiple entity fields:
```sql
WHERE tenant_id = $1 
  AND (
    'global' = ANY(COALESCE(target_entities, ARRAY['global']))
    OR target_entity = ANY(ARRAY[$2,$3])
    OR EXISTS (
      SELECT 1 FROM unnest(COALESCE(target_entities, ARRAY[target_entity])) AS t 
      WHERE t = ANY(ARRAY[$2,$3])
    )
  )
```

This handles:
- Global rules
- Single target_entity field
- target_entities array field
- Any combination of the above

## Testing Guide

### Starting the System

**Backend:**
```bash
cd /Users/eganpj/GitHub/semlayer/backend
PORT=29080 go run ./cmd/server/main.go
```

**Frontend (in separate terminal):**
```bash
cd /Users/eganpj/GitHub/semlayer/frontend
npm run dev
# or for production build
npm run build
```

### Testing Facet Filtering

1. Open validation rules page
2. Click on an entity facet (e.g., "Supplier")
3. Verify:
   - Rules list updates to show only Supplier rules
   - Other facet counts remain the same
   - Supplier facet count doesn't change
   - URL shows: `?tenant_id=...&target_entity=Supplier`

4. Click multiple facets:
   - Add "Customer" entity
   - URL shows: `?...&target_entity=Supplier&target_entity=Customer`
   - Rules show all that match either Supplier OR Customer
   - Facet counts still stable

5. Test other dimensions:
   - Rule type facets
   - Severity facets
   - Scope facets (global/specific)
   - Type facets (core/custom)

### Testing Rule Editing

1. Click pencil icon (✎) on any rule
2. Modal opens with rule details
3. Change rule name or description
4. Change severity dropdown
5. Toggle Active checkbox
6. Click "Save Changes"
7. Verify rule updates in the list
8. Close modal

### Testing Search

1. Type in search box
2. Verify debounce (300ms delay)
3. Results filter by rule name or description
4. Search combines with other filters

## Performance Characteristics

- **Initial Load:** ~20 rules per page
- **Pagination:** Supports up to 100 rules per page
- **Search Debounce:** 300ms
- **Facet Calculation:** O(n) where n = total rules for tenant
- **Query Response:** <20ms typical

## Database Schema Requirements

**catalog_validation_rules table:**
```sql
CREATE TABLE catalog_validation_rules (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  rule_name VARCHAR(255) NOT NULL,
  rule_type VARCHAR(100) NOT NULL,
  description TEXT,
  target_entity VARCHAR(255),
  target_entities TEXT[] DEFAULT ARRAY[]::TEXT[],
  condition_json JSONB,
  severity VARCHAR(20),
  is_active BOOLEAN DEFAULT true,
  is_core BOOLEAN DEFAULT false,
  created_by VARCHAR(255),
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_validation_rules_tenant ON catalog_validation_rules(tenant_id);
CREATE INDEX idx_validation_rules_entity ON catalog_validation_rules USING GIN(target_entities);
CREATE INDEX idx_validation_rules_active ON catalog_validation_rules(is_active);
```

## Styling

**File:** `/frontend/src/components/ValidationRules/ValidationRulesWithFacets.css`
**File:** `/frontend/src/components/ValidationRules/ValidationRuleEditor.css`

- Responsive design
- Color-coded chips and badges
- Sticky header and sidebar
- Smooth transitions
- Mobile-friendly layout

## Future Enhancements

1. **Bulk Operations**
   - Select multiple rules
   - Bulk edit/delete
   - Bulk activate/deactivate

2. **Advanced Features**
   - Rule templates
   - Rule versioning
   - A/B testing rules
   - Rule execution history

3. **UI Improvements**
   - Rule builder wizard
   - Condition JSON visual editor
   - Rule preview/simulation
   - Rule dependency visualization

4. **Performance**
   - Implement rule caching
   - Async rule execution
   - Real-time rule status

## Troubleshooting

### Port Already in Use
```bash
lsof -i :29080 | grep -v COMMAND | awk '{print $2}' | xargs kill -9
```

### Compilation Errors
```bash
# Frontend
npm run build

# Backend
cd backend && go build ./cmd/server/main.go
```

### Database Connection Issues
- Verify PostgreSQL is running on localhost:5432
- Check credentials in config.yaml
- Verify database `alpha` exists

### Facet Counts Not Showing
- Check browser console for API errors
- Verify tenant_id is valid UUID
- Check backend logs for query errors

## Verification Checklist

- ✅ Multiple URL parameters sent from frontend (not comma-separated)
- ✅ Backend correctly parses array parameters
- ✅ Entity filtering works with multiple entities
- ✅ Rule type, severity filtering works
- ✅ Facet counts stable across filters
- ✅ Rule type chip displays on each rule
- ✅ Global indicator chip shows
- ✅ Core/Custom indicator chip shows
- ✅ Edit modal opens and saves
- ✅ Search debounce works
- ✅ Pagination works
- ✅ UI responsive and polished

## Deployment Notes

1. Ensure database schema is initialized
2. Set `PORT` environment variable
3. Ensure `config.yaml` has valid database credentials
4. Build frontend: `npm run build`
5. Start backend: `PORT=29080 go run ./cmd/server/main.go`
6. Serve frontend from build output or via npm dev server
7. Access at `http://localhost:3000` (frontend) / `http://localhost:29080` (backend)
