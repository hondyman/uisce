# Semantic Mapper Override Flow Guide

## Overview
The override flow allows users to easily replace suggested or mapped semantic terms with custom terms. The UI is designed to be intuitive and clear about the state of each override.

## How to Override a Mapping

### Step 1: Enable Override Mode
1. Click the **Edit (✏️) icon** on any mapping row
2. The row will expand to show an autocomplete search box
3. The mapping is automatically marked as **selected** (checkbox checked)

### Step 2: Choose or Create a Semantic Term

You have three options:

#### Option A: Select an Existing Term
1. Start typing in the search box
2. Suggestions appear as you type (minimum 2 characters)
3. Click a suggestion from the dropdown
4. ✅ **Status**: "Ready to Create Edge" chip appears
5. The term is applied immediately and ready for edge creation

#### Option B: Type a New Term Name
1. Type the term name you want (minimum 2 characters)
2. As you type, the UI checks if the term exists
3. A yellow warning box appears: "⚠️ Unsaved Override"
4. If the term **exists**: Click "✓ Apply Existing Term"
5. If the term **doesn't exist**: Click "➕ Create & Apply New Term"
6. ✅ **Status**: "Ready to Create Edge" chip appears

#### Option C: Press Enter
1. Type your term name
2. Press **Enter**
3. If the term exists, it's automatically applied
4. If not, you'll see the create button

### Step 3: Create the Edge
1. Once the term is applied, the mapping shows a green "Ready to Create Edge" chip
2. The row remains selected (checked)
3. Click the **"Create Edges (N)"** button in the header
4. Confirm in the dialog
5. ✅ The edge is created and persisted to the knowledge graph

## Visual Indicators

### Row States
- **Normal**: Blue semantic term box, no override active
- **Override Active**: Search box visible, edit icon highlighted in orange
- **Unsaved Changes**: Yellow warning box with action buttons
- **Ready to Create**: Green "Ready to Create Edge" chip with pulse animation
- **Edge Exists**: Green "Mapped" chip with link icon

### Alert Messages
- **Success (Green)**: "Override Ready to Apply" - shows count of selected overrides ready to persist
- **Info (Blue)**: "Override in Progress" - shows count of overrides without a term selected yet
- **Warning (Orange)**: "Override Mode Active" - general notification that overrides are present

### Status Chips
- 🟢 **"Ready to Create Edge"**: Term selected, ready to create edge
- 🟢 **"Mapped"**: Edge already exists in database
- 🟢 **"NEW"**: This is a newly created semantic term
- 🟡 **"Generated ID"**: Mapping not yet persisted

## Understanding the Flow

### What Happens When You Override?
1. **Enable Override** → Row becomes editable, automatically selected
2. **Type/Select Term** → Term is staged locally in the UI
3. **Apply Term** → Term is created (if new) and assigned to mapping
4. **Create Edge** → Edge is persisted to knowledge graph

### Important Notes
- ✅ **Override automatically selects the row** for easy batch operations
- ✅ **You can override multiple mappings** before clicking "Create Edges"
- ✅ **Creating a term doesn't create the edge** - you must click "Create Edges"
- ✅ **Terms are created immediately**, but edges are batched
- ✅ **You can disable override mode** by clicking the edit icon again

## Example Workflows

### Workflow 1: Override with Existing Term
```
1. Click edit icon → Override enabled
2. Type "CUSTOMER" → Suggestions appear
3. Click "CUSTOMER_NAME" from dropdown → Applied instantly
4. See "Ready to Create Edge" chip → Confirmed ready
5. Click "Create Edges" → Edge persisted
```

### Workflow 2: Override with New Term
```
1. Click edit icon → Override enabled
2. Type "MY_CUSTOM_TERM" → No suggestions
3. Yellow warning box appears → "Create & Apply New Term" button
4. Click button → Term created, applied to mapping
5. See "Ready to Create Edge" chip → Confirmed ready
6. Click "Create Edges" → Edge persisted
```

### Workflow 3: Batch Override
```
1. Click edit on Row 1 → Type/select term → Ready
2. Click edit on Row 2 → Type/select term → Ready
3. Click edit on Row 3 → Type/select term → Ready
4. All 3 rows selected with green "Ready" chips
5. Click "Create Edges (3)" → All 3 edges persisted at once
```

## Troubleshooting

### "Create Edges" button is disabled
- **Cause**: No mappings are selected
- **Fix**: Click the checkbox on rows you want to create edges for

### Override not showing up after creating edge
- **Cause**: Page needs to refresh to show new edge state
- **Fix**: Edges are created in batch, page reloads automatically

### Term created but edge not showing
- **Cause**: You created the term but didn't click "Create Edges"
- **Fix**: Make sure the row is selected and click "Create Edges"

### Can't find my term in search
- **Cause**: Search requires 2+ characters
- **Fix**: Type at least 2 characters to trigger search

## Technical Details

### Component Updates
- `SemanticMapper.tsx`: Main coordinator, manages state sync
- `MappingRow.tsx`: Individual row UI, handles override interactions
- `useSemanticMapper.ts`: API calls for term/edge creation

### Key Functions
- `setOverride()`: Enables/disables override mode
- `selectSemanticTerm()`: Applies existing term to mapping
- `handleCreateAndSelectTerm()`: Creates new term and applies it
- `createEdges()`: Batch creates edges for selected mappings

### State Management
- Override state tracked per mapping
- Selection state synced automatically
- Unsaved changes detected by comparing current vs. staged term
- Term existence checked against search results
