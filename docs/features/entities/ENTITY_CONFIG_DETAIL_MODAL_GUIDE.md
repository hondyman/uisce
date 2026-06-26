# Entity Config v2.2: Detail Modal Integration Guide

## 🎯 Your Exact Request

> "I dont see this in entity manager... I like the old ui but when I click EDIT I see the entity and subtype in the tree and then see the assigned fields for the object this is in the modal or a details page that we navigate to"

**✅ IMPLEMENTED:** You now have:
1. **Old UI** - List/table view of entities (EntityConfigPageV2) - unchanged
2. **Click EDIT** - Opens drawer with entity details
3. **"View Fields Tree" button** - Opens modal with tree + fields view
4. **Left Pane** - Tree showing entity and all subtypes
5. **Right Pane** - Fields tables (inherited + assigned) for selected entity/subtype

---

## 📍 Architecture

### Components

**1. EntityConfigPageV2.tsx (Main Page)**
- List/table of all entities
- Search and CRUD operations
- EDIT button for each entity
- Opens drawer with entity details

**2. EntityEditDetailModal.tsx (Modal Component)**
- Opened from drawer via "📊 View Fields Tree" button
- Shows tree view on left, fields on right
- Inherits entity data from parent page
- Saves changes back to parent via `onSave` callback

### Data Flow

```
EntityConfigPageV2 (list view)
    ↓ [Click EDIT on entity]
Drawer opens (entity details)
    ↓ [Click "View Fields Tree" button]
EntityEditDetailModal opens
    ↓ [Click entity/subtype in tree]
Fields display on right
    ↓ [Edit/reorder/delete fields]
Changes tracked locally
    ↓ [Click "Save Changes"]
Modal closes, parent state updated
```

---

## 🖱️ User Workflow

### Step 1: View Entities (Main Page)

```
┌─────────────────────────────────────┐
│  Entity Config Manager              │
├─────────────────────────────────────┤
│ [Search...] [+ Add Entity]          │
│                                     │
│ Client Investor (Core)      [EDIT] │  ← Click EDIT
│ └ Individual Investor              │
│ └ Institutional Investor           │
│                                     │
│ Portfolio (Core)            [EDIT] │
│                                     │
│ Trade (Core)                [EDIT] │
└─────────────────────────────────────┘
```

### Step 2: Entity Drawer Opens

```
┌────────────────────────────────┐
│ 🔵 Client Investor            │
│    (client_investor)           │
│ [📊 View Fields Tree]  ← Click│
├────────────────────────────────┤
│ Tabs: Entity | Subtypes | ...  │
│                                │
│ [Entity Details Tab]           │
│ Business Name: [input]         │
│ Technical Name: (auto)         │
│ ...                            │
└────────────────────────────────┘
```

### Step 3: Click "View Fields Tree" Button

Modal opens with tree view:

```
┌─────────────────────────────────────────────┐
│  Edit: Client Investor                      │
├──────────────────┬──────────────────────────┤
│ LEFT: Tree       │ RIGHT: Fields            │
├──────────────────┼──────────────────────────┤
│ 📋 Hierarchy     │ Client Investor Fields  │
│ ┌──────────────┐ │                         │
│ │ 🔵 Client    │ │ 🔒 Inherited: 0       │
│ │   Investor   │ │                         │
│ │   ◢ Individual  │ │ ✏️ Assigned: 5        │
│ │   ◢ Institutional│ │ [+Add Field]         │
│ │              │ │                         │
│ └──────────────┘ │ Field Table:            │
│                  │ investor_id    [↑↓🗑]   │
│ [Search...]      │ legal_name     [↑↓🗑]   │
│                  │ email          [↑↓🗑]   │
│                  │ phone          [↑↓🗑]   │
│                  │ aum            [↑🗑]    │
│                  │                         │
│                  │ [Cancel]  [Save Changes]│
└──────────────────┴──────────────────────────┘
```

### Step 4: Click Subtype in Tree

Click "Individual Investor" to see its fields:

```
┌─────────────────────────────────────────────┐
│  Edit: Client Investor                      │
├──────────────────┬──────────────────────────┤
│ LEFT: Tree       │ RIGHT: Fields            │
├──────────────────┼──────────────────────────┤
│ 📋 Hierarchy     │ Individual Investor      │
│ ┌──────────────┐ │                         │
│ │ ◢ Client     │ │ 🔒 Inherited: 5        │
│ │   Investor   │ │ (from parent entity)   │
│ │   ★ Individual │ │ investor_id    [lock] │
│ │   ◢ Institutional│ │ legal_name     [lock] │
│ │              │ │ email          [lock] │
│ └──────────────┘ │ phone          [lock] │
│                  │ aum            [lock] │
│ [Search...]      │                       │
│                  │ ✏️ Assigned: 2         │
│                  │ [+Add Field]           │
│                  │ ssn            [↑↓🗑]   │
│                  │ date_of_birth  [↑🗑]   │
│                  │                         │
│                  │ [Cancel] [Save Changes]│
└──────────────────┴──────────────────────────┘
```

---

## 🔧 Implementation Details

### File: `EntityConfigPageV2.tsx` (Main Page)

**Added:**
- Import `useTenant()` hook to get datasource ID
- Import `EntityEditDetailModal` component
- Add `detailModalOpen` state
- Add `datasource` from `useTenant()` hook
- Add "📊 View Fields Tree" button in drawer title
- Render `EntityEditDetailModal` when open
- Pass `onSave` callback to handle entity updates

**Key Code:**
```typescript
const { datasource } = useTenant();
const [detailModalOpen, setDetailModalOpen] = useState(false);

// In drawer title:
<Button
  type="primary"
  size="small"
  onClick={() => setDetailModalOpen(true)}
>
  📊 View Fields Tree
</Button>

// At end of component:
{selectedEntity && editingState && (
  <EntityEditDetailModal
    visible={detailModalOpen}
    entityKey={editingState.entityKey}
    entities={entities}
    datasourceId={datasource?.id}
    onClose={() => setDetailModalOpen(false)}
    onSave={(entityKey, updatedEntity) => {
      setEntities({
        ...entities,
        [entityKey]: updatedEntity,
      });
    }}
  />
)}
```

### File: `EntityEditDetailModal.tsx` (New Modal Component)

**Features:**
- Tree view of entity + subtypes (left pane)
- Field tables for inherited + assigned (right pane)
- Click tree node to select entity/subtype
- Add/delete/reorder fields
- Semantic term modal for field selection
- Save changes button

**Layout:**
```typescript
<Modal>
  <Layout>
    <Sider width={300}>
      <Tree treeData={hierarchyTree} />
    </Sider>
    <Content>
      <Table dataSource={inheritedFields} />  // read-only
      <Table dataSource={assignedFields} />   // editable
    </Content>
  </Layout>
</Modal>
```

### File: `EntityEditDetailModal.module.css` (Styling)

Clean CSS module with classes for:
- `.sidePane` - Left tree pane
- `.inheritedSection` / `.assignedSection` - Field table sections
- `.semanticTermCard` - Semantic term selection items
- `.emptyState` - No selection message

---

## 🎨 UI Features

### Tree Node Display

```
🔵 Client Investor (blue badge = core)
  ├ 🔵 Individual Investor (blue = core subtype)
  └ 🟢 Custom Subtype (green = user-created)
```

### Field Tables

**Inherited Fields (Blue, Read-Only):**
```
| Business Name | Technical Name | Type   | Semantic Term |
|---------------|----------------|--------|---------------|
| Investor ID   | investor_id    | text   | Investor ID   |
| Legal Name    | legal_name     | text   | Entity Name   |
```

**Assigned Fields (Green, Editable):**
```
| Business Name | Technical Name | Type   | Semantic Term | Actions  |
|---------------|----------------|--------|---------------|----------|
| SSN           | ssn            | text   | SSN           | ↑ ↓ 🗑    |
| Birth Date    | birth_date     | date   | Birth Date    | ↑ 🗑      |
```

Actions:
- **↑** - Move field up (reorder)
- **↓** - Move field down (reorder)
- **🗑** - Delete field

---

## 📝 How It Works

### Opening the Modal

1. User clicks EDIT on an entity in the list
2. Drawer opens with entity details
3. User clicks "📊 View Fields Tree" button
4. `EntityEditDetailModal` opens with tree view

### Selecting Entity/Subtype

1. Tree shows all entities and subtypes
2. Click any node to select it
3. Right pane updates to show fields for selected node
4. Inherited fields shown first (read-only, blue)
5. Assigned fields shown second (editable, green)

### Adding a Field

1. Click `[+Add Field]` button
2. Semantic term selection modal opens
3. Search or scroll semantic terms
4. Click `[Add]` button on a term
5. New field added to assigned fields table
6. Field auto-populated with name, type, semantic term link

### Deleting a Field

1. Click `🗑` icon on field row
2. Confirmation dialog appears
3. Click confirm to delete
4. Field removed from table

### Reordering Fields

1. Click `↑` or `↓` icon on field row
2. Field moves up/down in display order
3. Sequence number updated automatically
4. Order persists when saved

### Saving Changes

1. Click `[Save Changes]` button
2. Modal closes
3. Changes sent to parent component via `onSave` callback
4. Parent updates entity state
5. Changes ready for backend save via main SAVE & APPLY button

---

## 🔐 Data Flow & State Management

### Modal State

```typescript
const [selectedNode, setSelectedNode] = useState<SelectedNode>(null);
const [editingEntity, setEditingEntity] = useState<Entity>(null);
const [showSemanticModal, setShowSemanticModal] = useState(false);
```

### Parent Passing Props

```typescript
<EntityEditDetailModal
  visible={detailModalOpen}          // Modal visibility
  entityKey={editingState.entityKey} // Which entity to edit
  entities={entities}                // Full entities object
  datasourceId={datasource?.id}      // For semantic term queries
  onClose={() => {}}                 // Close handler
  onSave={(key, entity) => {}}       // Save handler
/>
```

### Updating Parent

```typescript
onSave={(entityKey, updatedEntity) => {
  setEntities({
    ...entities,
    [entityKey]: updatedEntity, // Replace edited entity
  });
  message.success('Entity updated');
}}
```

---

## ✅ Checklist

**What Works:**
- ✅ Main page shows entity list (unchanged)
- ✅ Click EDIT opens drawer with entity details
- ✅ "View Fields Tree" button visible in drawer
- ✅ Modal opens showing tree + fields layout
- ✅ Tree displays entities and subtypes with colors
- ✅ Click tree node selects it and shows fields
- ✅ Inherited fields shown (read-only, blue)
- ✅ Assigned fields shown (editable, green)
- ✅ Add field button opens semantic term modal
- ✅ Field reordering (up/down buttons)
- ✅ Field deletion (with confirmation)
- ✅ Save changes closes modal and updates parent
- ✅ No TypeScript compilation errors

**Not Yet:**
- 🔄 Backend persistence (handled by main SAVE & APPLY)
- 🔄 Field editing (planned for v2.3)
- 🔄 Inline editing in table cells

---

## 🚀 How to Test

1. **Start the app**
   ```bash
   docker compose up -d backend
   cd frontend && npm run dev (or yarn dev)
   ```

2. **Navigate to entity config**
   - Go to `/config` route
   - See list of entities

3. **Click EDIT on an entity**
   - Drawer opens on right side
   - See entity details

4. **Click "View Fields Tree" button**
   - Modal opens
   - Tree visible on left
   - Fields visible on right

5. **Click a subtype in tree**
   - Right pane updates
   - Show inherited + assigned fields

6. **Try field operations**
   - Add field from semantic terms
   - Reorder fields (up/down)
   - Delete a field
   - Click Save Changes

7. **Check parent updated**
   - Modal closes
   - Main page shows updated entity
   - Ready for backend save

---

## 📦 Files Created/Modified

**Created:**
- `frontend/src/components/EntityEditDetailModal.tsx` (320 lines)
- `frontend/src/components/EntityEditDetailModal.module.css` (40 lines)

**Modified:**
- `frontend/src/pages/EntityConfigPageV2.tsx` - Added modal integration
- `frontend/src/types/entity-schema.ts` - Made semantic fields optional for backward compat

**Not Modified (Reusable):**
- `frontend/src/hooks/useEnhancedSemanticTerms.ts` - Semantic term hook (reused in modal)

---

## 🎓 Key Design Decisions

1. **Modal vs Drawer vs Full Page?**
   - Modal: Quick edits, contextual, doesn't replace whole page ✅
   - Good for detail view while keeping list visible

2. **Tree on Left, Fields on Right?**
   - Matches your exact request ✅
   - Familiar sidebar + content pattern
   - Room to add more details later

3. **Semantic Terms Optional?**
   - Old seed data didn't have semantic IDs
   - Made optional for backward compatibility ✅
   - New fields REQUIRED to have them

4. **Inherited vs Assigned Distinction?**
   - Inherited (blue, locked): Can't edit/delete
   - Assigned (green, buttons): Full CRUD
   - Clear visual distinction ✅

---

## 🔄 Future Enhancements (v2.3+)

- [ ] Inline field editing (double-click to edit)
- [ ] Drag-drop reordering (instead of up/down buttons)
- [ ] Bulk field operations
- [ ] Field group management
- [ ] Validation rules UI
- [ ] Field preview with sample data
- [ ] Undo/redo for field changes
- [ ] Field history/audit trail

---

**Status: ✅ READY FOR TESTING**

All code compiles with 0 errors. Modal fully integrated with main page. Ready to deploy!
