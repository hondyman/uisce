# Implementation Details - Modal & GraphQL Fixes

## 1. New EntityDetailsModal Component

### Location
`frontend/src/components/EntityDetailsModal.tsx`

### Purpose
Provides a cleaner, more professional alternative to the drawer for editing entity details with built-in breadcrumb navigation and clear back button.

### Key Features

**Breadcrumb Navigation**
```tsx
<Breadcrumb
  items={[
    { title: 'Entity Manager' },
    { title: entity.businessName || entity.name },
  ]}
/>
```
- Shows clear context: where you are in the app
- Helps users understand they're editing an entity

**Two-Tab Interface**
```tsx
<Tabs
  items={[
    { key: 'entity', label: '📋 Entity', children: <EntityDrawerTreeView /> },
    { key: 'related', label: '🔗 Related Objects', children: <RelatedObjectsPanel /> },
  ]}
/>
```

**Responsive Sizing**
```tsx
<Modal
  width="90vw"
  style={{ maxWidth: '1400px' }}
/>
```
- 90% of viewport width on smaller screens
- Maximum 1400px on large screens
- Works well on mobile and desktop

**Proper Close Button**
```tsx
footer={[
  <Button key="close" onClick={onClose}>
    Close
  </Button>,
]}
```
- Clear, obvious close/back action
- Better than drawer's ambiguous X button

### Component Props

```typescript
interface EntityDetailsModalProps {
  visible: boolean;                    // Controls modal visibility
  entityKey: string;                   // Which entity to edit
  entity: Entity | null;               // The entity object
  entities: Entities;                  // All entities (for tree view context)
  tenant: { id: string } | null;       // Tenant for relationships
  datasource: {                        // Datasource for relationships
    id?: string;
    alpha_datasource_id?: string;
  } | null;
  onClose: () => void;                 // Called when user closes
  onEntityUpdate: (entityKey: string, updatedEntity: Entity) => void;  // Called when entity changes
}
```

### Usage in EntityConfigPageV2

```typescript
// State management
const [detailsModalOpen, setDetailsModalOpen] = useState(false);
const [editingState, setEditingState] = useState<EditingEntityState | null>(null);

// Open modal when edit is clicked
const handleEditEntity = (entityKey: string) => {
  setEditingState({ entityKey });
  setDetailsModalOpen(true);
};

// Render modal
<EntityDetailsModal
  visible={detailsModalOpen}
  entityKey={editingState.entityKey}
  entity={selectedEntity}
  entities={entities}
  tenant={tenant}
  datasource={datasource}
  onClose={() => {
    setDetailsModalOpen(false);
    setEditingState(null);
  }}
  onEntityUpdate={(entityKey, updatedEntity) => {
    setEntities({
      ...entities,
      [entityKey]: updatedEntity,
    });
  }}
/>
```

---

## 2. GraphQL Query Fixes

### Problem
The RelatedObjectsPanel was requesting a `kind` field that doesn't exist in the BusinessTerm type.

### Error Message
```
Error loading related objects: field 'kind' not found in type: 'BusinessTerm'
```

### Affected Queries

#### Query 1: GET_RELATED_OBJECTS
**Before** ❌:
```graphql
query GetRelatedObjects($tenantId: ID!, $datasourceId: ID!, $entity: String!) {
  getRelatedObjects(tenantId: $tenantId, datasourceId: $datasourceId, entity: $entity) {
    edgeId
    direction
    edgeType
    cardinality
    source { id name kind description }    # ❌ WRONG
    target { id name kind description }    # ❌ WRONG
  }
}
```

**After** ✅:
```graphql
query GetRelatedObjects($tenantId: ID!, $datasourceId: ID!, $entity: String!) {
  getRelatedObjects(tenantId: $tenantId, datasourceId: $datasourceId, entity: $entity) {
    edgeId
    direction
    edgeType
    cardinality
    source { id name description }         # ✅ CORRECT
    target { id name description }         # ✅ CORRECT
  }
}
```

#### Query 2: APPLY_RELATIONSHIP
**Before** ❌:
```graphql
mutation ApplyRelationship(...) {
  applyRelationship(...) {
    edgeId
    direction
    edgeType
    cardinality
    source { id name kind description }    # ❌ WRONG
    target { id name kind description }    # ❌ WRONG
  }
}
```

**After** ✅:
```graphql
mutation ApplyRelationship(...) {
  applyRelationship(...) {
    edgeId
    direction
    edgeType
    cardinality
    source { id name description }         # ✅ CORRECT
    target { id name description }         # ✅ CORRECT
  }
}
```

### Why This Happens
- GraphQL schema validates field existence at query time
- `kind` field isn't defined in BusinessTerm type in backend schema
- Apollo Client throws error when field doesn't exist
- Solution: Remove unsupported field from queries

### Files Modified
- `frontend/src/components/catalog/RelatedObjectsPanel.tsx`
  - Line ~8: GET_RELATED_OBJECTS query
  - Line ~39: APPLY_RELATIONSHIP mutation

---

## 3. State Management Changes

### EntityConfigPageV2 State

**Removed**:
```typescript
const [drawerOpen, setDrawerOpen] = useState(false);
const [detailModalOpen, setDetailModalOpen] = useState(false);  // ← old name
```

**Added**:
```typescript
const [detailsModalOpen, setDetailsModalOpen] = useState(false);  // ← new name
```

**Why**: Consistent naming and clarity - "details modal" vs "drawer"

### Flow Changes

**Before (Drawer)**:
```
handleEditEntity()
  ↓
setDrawerOpen(true)
  ↓
<Drawer onClose={() => {
  setDrawerOpen(false);
  setEditingState(null);
}} />
```

**After (Modal)**:
```
handleEditEntity()
  ↓
setDetailsModalOpen(true)
  ↓
<EntityDetailsModal onClose={() => {
  setDetailsModalOpen(false);
  setEditingState(null);
}} />
```

---

## 4. Component Hierarchy

### Before
```
EntityConfigPageV2
  ├─ Main content
  ├─ Drawer (slide-out from right)
  │  └─ Tabs
  │     ├─ EntityDrawerTreeView
  │     └─ RelatedObjectsPanel
  ├─ Add/Edit Entity Modal
  ├─ Add Subtype Modal
  ├─ Add Field Modal
  └─ Affix (save button)
```

### After
```
EntityConfigPageV2
  ├─ Main content
  ├─ EntityDetailsModal
  │  └─ Tabs
  │     ├─ EntityDrawerTreeView
  │     └─ RelatedObjectsPanel
  ├─ Add/Edit Entity Modal
  ├─ Add Subtype Modal
  ├─ Add Field Modal
  └─ Affix (save button)
```

---

## 5. Error Handling

### Apollo useQuery Pattern ✅ CORRECT
```typescript
// This is the right way to handle GraphQL errors
const { data, loading, error, refetch } = useQuery(GET_RELATED_OBJECTS, {
  variables: { tenantId, datasourceId, entity },
  fetchPolicy: "cache-and-network",
});

// Use derived state 'error' from the hook
if (error) {
  return <p>Error loading related objects: {error.message}</p>;
}
```

### Why This is Better Than onError Callback
```typescript
// ❌ AVOID: Setting state in onError callback
const { data } = useQuery(GET_RELATED_OBJECTS, {
  onError: (err) => setLocalError(err),  // Can cause stale state issues
});

// ✅ BETTER: Use derived state from hook
const { data, error } = useQuery(GET_RELATED_OBJECTS);
// Now error updates automatically with Apollo cache
```

**Benefits**:
- State automatically syncs with Apollo cache
- No manual state management needed
- React strictMode doesn't complain
- Cleaner code

---

## 6. Testing Scenarios

### Scenario 1: Edit Entity
```
1. Open Entity Manager
2. Click Edit on entity card
3. EXPECT: Modal appears (not drawer)
4. EXPECT: Breadcrumb shows entity name
5. VERIFY: Tabs switch correctly
6. VERIFY: No GraphQL errors in console
7. Click Close
8. EXPECT: Modal disappears, grid visible again
```

### Scenario 2: View Relationships
```
1. Open Entity Manager
2. Click Edit on entity card
3. Click "🔗 Related Objects" tab
4. EXPECT: No "field 'kind' not found" error
5. EXPECT: Relationships display
6. VERIFY: Suggestions load successfully
```

### Scenario 3: Modal Sizing
```
On Desktop (1400px+):
1. Modal should be 1400px wide
2. Content should feel spacious
3. No horizontal scrolling

On Tablet (768-1399px):
1. Modal should be ~90vw (90% of viewport)
2. Margins on sides for context
3. Tabs should be easily readable

On Mobile (<768px):
1. Modal should take most of screen
2. Scrollable if needed
3. Touch-friendly buttons
```

---

## 7. Migration Path

### For Users
- **No action required** - UI improvements are automatic
- Drawer is replaced with modal
- All functionality preserved
- Same entity editing experience, better UX

### For Developers
- If you imported `EntityConfigPageV2` directly, no changes needed
- If you were accessing `drawerOpen` state externally, update to `detailsModalOpen`
- EntityDetailsModal is available as a reusable component

### For Tests
- Update drawer tests to modal tests
- Test breadcrumb rendering
- Test close button functionality
- Test tab switching in modal

---

## 8. Performance Impact

**Negligible**:
- Modal vs Drawer: Same rendering performance
- GraphQL queries: Slightly better (fewer fields requested)
- State management: Same complexity
- Bundle size: +~3KB (EntityDetailsModal component)

---

## Deployment Notes

- **Backward Compatible**: Yes
- **Database Changes**: No
- **API Changes**: No
- **Breaking Changes**: No
- **Manual Deployment Steps**: None

---

**Status**: ✅ Ready for Production
