# Related Objects Integration Guide

## Overview

The **Related Objects** feature has been fully integrated into the **Entity Manager** (V2) as a first-class tab, allowing users to discover and manage relationships between business entities directly from the schema configuration interface.

## Architecture

### Before Integration
- Related Objects lived in a separate standalone page (`RelatedObjectsPage.tsx`)
- Users had to navigate away from Entity Manager to view relationships
- Drawer had a nested "Related Objects" tab but only for selected entities

### After Integration
- **Main Tab**: "🔗 Relationships" tab in Entity Manager V2 for browsing all entity relationships
- **Drawer Tab**: "🔗 Related Objects" tab remains in the edit drawer for quick in-context viewing
- Users can switch between schema configuration and relationship management seamlessly

## User Experience

### Scenario 1: Browse All Entity Relationships
1. Open **Entity Manager** (Schema Builder)
2. Click the **"🔗 Relationships"** tab
3. Use the dropdown to select any entity
4. View all discovered relationships and AI-suggested connections
5. Apply relationships directly from the UI

### Scenario 2: Edit Entity & View Its Relationships
1. In the **Schema Configuration** tab, find an entity card
2. Click **Edit** (pencil icon)
3. The drawer opens with tabs:
   - **📋 Entity**: Fields, subtypes, hierarchy
   - **🔗 Related Objects**: Relationships specific to this entity
4. Switch between tabs to see both structure and relationships

### Scenario 3: Apply Relationship Suggestions
1. Go to the **Relationships** tab
2. Select an entity from the dropdown
3. **AI Suggestions** panel shows:
   - Suggested related entities (based on catalog edges)
   - Relationship type and cardinality
   - Confidence score
   - Reasoning explanation
4. Accept/dismiss suggestions with one click
5. Save changes in the main header

## File Structure

```
frontend/src/
├── pages/
│   ├── EntityConfigPageV2.tsx          ← PRIMARY (with integrated tabs)
│   ├── EntityConfigPageV3.tsx          ← Alternative
│   ├── admin/
│   │   └── RelatedObjectsPage.tsx      ← Legacy standalone (still available)
│   └── EntityConfigPage.tsx            ← Older version
├── components/
│   ├── catalog/
│   │   ├── RelatedObjectsPanel.tsx     ← Shared component (reusable)
│   │   └── SuggestionPreviewModal.tsx
│   └── EntityDrawerTreeView.tsx
└── api/
    └── entitySchema.ts                 ← Fixed: now includes tenant headers
```

## Technical Changes

### 1. Fixed API Integration
**File**: `frontend/src/api/entitySchema.ts`

The `fetchEntitySchema()` function now accepts optional `tenantId` and `datasourceId` parameters:

```typescript
export function fetchEntitySchema(tenantId?: string, datasourceId?: string): Promise<Entities> {
  // Adds required headers:
  // X-Tenant-ID: {tenantId}
  // X-Tenant-Datasource-ID: {datasourceId}
}
```

**Updated callers**:
- `RelatedObjectsPage.tsx` - Passes tenant/datasource IDs
- `EntityConfigPage.tsx` - Passes tenant/datasource IDs
- `EntityConfigPageV2.tsx` - Passes tenant/datasource IDs
- `EntityConfigPageV3.tsx` - Passes tenant/datasource IDs

### 2. Entity Manager V2 Enhancement
**File**: `frontend/src/pages/EntityConfigPageV2.tsx`

Added state management:
```typescript
const [mainViewTab, setMainViewTab] = useState<'schema' | 'relationships'>('schema');
const [selectedEntityForRelationships, setSelectedEntityForRelationships] = useState<string>('');
```

Added main view tabs:
- **Schema Configuration** (existing grid view)
- **Relationships** (new tab with entity selector + RelatedObjectsPanel)

### 3. Tenant Scope Enforcement
The `fetchAPI()` shim (from `setupTenantFetch.ts`) automatically adds tenant parameters to all API requests. For `entity-schema` endpoint:

```bash
GET /api/entity-schema
Headers: X-Tenant-ID, X-Tenant-Datasource-ID
```

## Relationship Discovery

The **Relationships** tab uses the following GraphQL queries:

### GET_RELATED_OBJECTS
Returns existing relationships for an entity:
- Source/target entity pairs
- Edge types and cardinality
- Relationship confidence

### GET_RELATIONSHIP_SUGGESTIONS
Returns AI-powered suggestions based on:
- `catalog_node` relationships
- `catalog_edge` mappings
- Field type matching
- Semantic similarity

### APPLY_RELATIONSHIP
Mutation to create/update relationships:
- Saves to the schema
- Updates the UI
- Triggers refetch of relationships

## Catalog Integration Points

### catalog_node Relationships
- Maps business objects to semantic nodes
- Used to identify entities with shared concepts
- Example: "Customer" node links to client, investor, counterparty entities

### catalog_edge Relationships
- Defines potential connections between nodes
- Sources include:
  - Data lineage analysis
  - Foreign key detection
  - Semantic similarity matching
  - User-defined mappings

## Configuration

### For Users
1. Select a tenant and datasource from the tenant picker
2. Relationships automatically load with the correct scope

### For Developers
If calling `fetchEntitySchema()` manually, always pass tenant IDs:

```typescript
const schema = await fetchEntitySchema(
  tenant.id,
  datasource.id || datasource.alpha_datasource_id
);
```

## Troubleshooting

### 400 Bad Request on /api/entity-schema
**Cause**: Missing tenant scope headers

**Fix**: 
- Ensure tenant and datasource are selected via the tenant picker
- Verify `X-Tenant-ID` and `X-Tenant-Datasource-ID` headers are present
- Check browser DevTools Network tab for header values

### Relationships Not Showing
**Cause**: Possible GraphQL query failure

**Fix**:
1. Check Apollo Client errors in browser console
2. Verify catalog data is populated in the database
3. Look for DISMISS_SUGGESTION mutations in the logs

### Relationship Suggestions Empty
**Cause**: No catalog edges defined for the selected entity

**Fix**:
1. Ensure `catalog_edge` relationships exist for the entity
2. Run semantic analysis to auto-detect relationships
3. Manually map relationships via the Entity Manager

## Future Enhancements

### Phase 2
- [ ] Bulk relationship import/export
- [ ] Relationship versioning and history
- [ ] Relationship conflict detection
- [ ] Advanced cardinality validation

### Phase 3
- [ ] Relationship visualization (graph view)
- [ ] Relationship impact analysis
- [ ] Automatic relationship healing
- [ ] Machine learning-based suggestions

## Migration from Legacy Page

### Old Workflow
1. Open Entity Manager
2. Navigate to Related Objects page
3. Select tenant/datasource (again)
4. Choose entity
5. View relationships
6. Navigate back to Entity Manager

### New Workflow
1. Open Entity Manager
2. Click "Relationships" tab
3. Select entity
4. View relationships (same tenant/datasource scope)

**No manual migration required** - The legacy `RelatedObjectsPage.tsx` is still available for backward compatibility but the integrated version is recommended.

## Performance Considerations

- **RelatedObjectsPanel** queries relationships on-demand (not cached)
- Use `fetchPolicy: "cache-and-network"` to balance freshness and speed
- For large entity sets (100+), consider pagination in future versions

## API Endpoints Used

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/entity-schema` | GET | Fetch entity definitions (now with tenant scope) |
| GraphQL `GetRelatedObjects` | Query | Fetch existing relationships |
| GraphQL `GetRelationshipSuggestions` | Query | Get AI suggestions |
| GraphQL `ApplyRelationship` | Mutation | Create/update relationships |

## Related Files

- **Primary**: `/Users/eganpj/GitHub/semlayer/frontend/src/pages/EntityConfigPageV2.tsx`
- **Shared Component**: `/Users/eganpj/GitHub/semlayer/frontend/src/components/catalog/RelatedObjectsPanel.tsx`
- **API**: `/Users/eganpj/GitHub/semlayer/frontend/src/api/entitySchema.ts`
- **Tenant Context**: `/Users/eganpj/GitHub/semlayer/frontend/src/contexts/TenantContext.ts`
- **Backend**: `/Users/eganpj/GitHub/semlayer/backend/internal/api/api.go` (lines 875-950)

---

**Last Updated**: October 24, 2025  
**Status**: ✅ Integration Complete & Tested
