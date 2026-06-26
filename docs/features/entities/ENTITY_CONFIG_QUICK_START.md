# 🚀 Quick Start: Entity Config Detail Modal

## What Changed?

You now have the exact workflow you requested:

```
List View (Old UI) → Click EDIT → Drawer Opens → Click "View Fields Tree" → 
Modal Opens with Tree + Fields
```

## Try It Now

### 1. Start Everything
```bash
# Terminal 1: Backend
docker compose up -d backend

# Terminal 2: Frontend
cd frontend
npm run dev  # or yarn dev
```

### 2. Open Browser
```
http://localhost:5173/config
```

### 3. See the List
You'll see a list of entities:
- Client Investor (blue - core)
- Portfolio (blue - core)
- Trade (blue - core)

### 4. Click EDIT
Click the **[EDIT]** button next to any entity.

A drawer opens on the right with entity details.

### 5. Click "View Fields Tree"
You'll see a blue button in the drawer title:
```
🔵 Client Investor (client_investor) [📊 View Fields Tree]
```

Click it.

### 6. See the Tree + Fields Modal
A modal pops up:

**Left Side:**
- Tree showing entity + subtypes
- Example:
  ```
  🔵 Client Investor
    ├ 🔵 Individual Investor
    └ 🔵 Institutional Investor
  ```

**Right Side:**
- "Select an entity or subtype from the tree" message
- (empty until you click a node)

### 7. Click a Node
Click "Individual Investor" in the tree.

**Right side now shows:**

```
Individual Investor Fields
═══════════════════════════════════════════════════════════

🔒 Inherited Fields (2)
┌────────────────────────────────────────────────────────┐
│ Business Name    │ Technical Name  │ Type  │ Sem Term  │
├────────────────────────────────────────────────────────┤
│ Investor ID      │ investor_id     │ text  │ Investor  │
│ Legal Name       │ legal_name      │ text  │ Entity    │
└────────────────────────────────────────────────────────┘
(inherited from parent entity - read-only)

✏️ Assigned Fields (2) [+Add Field]
┌────────────────────────────────────────────────────────┐
│ Business Name    │ Technical Name  │ Type  │ Sem │ Act │
├────────────────────────────────────────────────────────┤
│ SSN              │ ssn             │ text  │ SSN │↑↓🗑 │
│ Birth Date       │ birth_date      │ date  │ DOB │↑🗑  │
└────────────────────────────────────────────────────────┘
(customizations for this subtype - full edit control)
```

### 8. Try Operations

**Add a Field:**
1. Click `[+Add Field]` button
2. Semantic term modal opens
3. Search for "Tax" or scroll
4. Click `[Add]` button
5. New field added to table

**Reorder Fields:**
1. Click `↑` on SSN row
2. SSN moves above Birth Date

**Delete Field:**
1. Click `🗑` on Birth Date row
2. Confirm deletion
3. Field removed

### 9. Save Changes
Click `[Save Changes]` button at bottom of modal.

**Result:**
- Modal closes
- Changes saved to local entity state
- Main page entity updated
- Ready for backend save via main "SAVE & APPLY" button

---

## File Structure

```
frontend/src/
├── pages/
│   └── EntityConfigPageV2.tsx          ← Main page (list view)
│       └── [Click EDIT] → Opens drawer
│           └── [View Fields Tree] → Opens modal
│
├── components/
│   └── EntityEditDetailModal.tsx       ← NEW: Detail modal
│       └── EntityEditDetailModal.module.css
│
└── hooks/
    └── useEnhancedSemanticTerms.ts     ← Reused for semantic terms
```

---

## Component Props

### EntityEditDetailModal

```typescript
interface Props {
  visible: boolean;           // Is modal visible?
  entityKey: string;          // Entity to edit (e.g., "client_investor")
  entities: Entities;         // Full entities object
  datasourceId?: string;      // For semantic term queries
  onClose: () => void;        // Called when modal closes
  onSave: (key, entity) => void;  // Called when user saves
}
```

---

## State & Hooks

**Main Page Uses:**
```typescript
const { datasource } = useTenant();  // Get datasource ID
const [detailModalOpen, setDetailModalOpen] = useState(false);
```

**Modal Uses:**
```typescript
const [selectedNode, setSelectedNode] = useState<SelectedNode>(null);
const [editingEntity, setEditingEntity] = useState<Entity>(null);
const { semanticTerms } = useEnhancedSemanticTerms(datasourceId);
```

---

## Data Flow

```
User clicks EDIT
  ↓
EntityConfigPageV2: setDrawerOpen(true)
  ↓
Drawer renders with entity details
  ↓
User clicks [View Fields Tree]
  ↓
EntityConfigPageV2: setDetailModalOpen(true)
  ↓
EntityEditDetailModal opens
  ↓
User clicks tree node
  ↓
Modal: setSelectedNode({ type, entityKey, subtypeKey? })
  ↓
Right pane computes inherited + assigned fields
  ↓
Fields display in tables
  ↓
User edits fields (add/delete/reorder)
  ↓
Modal tracks changes locally
  ↓
User clicks [Save Changes]
  ↓
Modal calls onSave(entityKey, updatedEntity)
  ↓
Main page updates: setEntities({ ...entities, [key]: entity })
  ↓
Modal closes
```

---

## Colors & Badges

| Icon | Color | Meaning |
|------|-------|---------|
| 🔵 | Blue | Core business object (seed data) |
| 🟢 | Green | Custom/user-created object |
| 🔒 | Gray | Inherited fields (read-only) |
| ✏️ | Green | Assigned fields (editable) |

---

## Keyboard & Mouse

| Action | Result |
|--------|--------|
| Click tree node | Select entity/subtype, show fields |
| Click `↑` button | Move field up in order |
| Click `↓` button | Move field down in order |
| Click `🗑` button | Delete field (with confirm) |
| Click `[+Add Field]` | Open semantic term picker |
| Click `[Add]` in modal | Add selected term as field |
| Click `[Save Changes]` | Save all changes, close modal |
| Click `[Cancel]` | Discard changes, close modal |

---

## Debugging

### Check if modal opens
```javascript
// In browser console:
localStorage.getItem('selected_datasource')
// Should show: { "id": "...", "source_name": "..." }
```

### Check tree renders
Open DevTools → Elements tab → Find `<Tree />` component

### Check fields update
Click a tree node → Check if right pane updates → Fields should appear

### Check semantic terms load
- Open DevTools → Network tab
- Look for GraphQL query to `catalog_node`
- Should return semantic terms

---

## Troubleshooting

**Modal doesn't open?**
- Check if drawer is open first (click EDIT)
- Check browser console for errors
- Check `datasourceId` is being passed

**Fields don't show?**
- Click a tree node to select it
- Wait 1 second for right pane to update
- Fields should appear below

**Tree is empty?**
- Reload page
- Check entity data is loaded
- Check `entities` prop has data

**Can't add field?**
- Click `[+Add Field]` button
- Semantic term modal should open
- Search for a term
- Click `[Add]` button on a term

---

## What's Next?

The modal is fully functional for:
- ✅ Viewing entity hierarchy
- ✅ Viewing fields by entity/subtype
- ✅ Adding fields from semantic catalog
- ✅ Deleting fields
- ✅ Reordering fields
- ✅ Saving changes back to parent

Then the main page "SAVE & APPLY" button persists to backend.

---

**Status: Ready to use!** 🎉
