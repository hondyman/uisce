# Lineage Fix & Impact Tab Removal - Complete

## Issues Fixed

### 1. ❌ 500 Error: Lineage Not Showing
**Error:** `function cypher(unknown, unknown, unknown) does not exist (SQLSTATE 42883)`

**Root Cause:** SemanticTermDetails was calling `/api/lineage/node/{id}/graph?depth=3&engine=cypher` which tried to use Apache AGE's cypher extension that was removed.

**Fix:** 
- Changed default `lineageType` from `'cypher'` to `'sql'`
- Removed `&engine=${lineageType}` parameter from API call
- Removed the SQL/Cypher toggle UI since we only use relational SQL now

### 2. ✅ Removed Impact Analysis Tab
Replaced separate "Impact Analysis" tabs with the new **UnifiedLineageTab** component that combines lineage (upstream) and impact (downstream) in one view.

## Files Modified

### 1. `/frontend/src/pages/TabbedModal/tabs/SemanticTermDetails.tsx`
**Changes:**
- Import changed: `ImpactAnalysisTab` → `UnifiedLineageTab`
- Default lineage engine: `'cypher'` → `'sql'`
- API call: Removed `&engine=${lineageType}` parameter
- Removed SQL/Cypher toggle buttons from UI
- Tab label: "Impact Analysis" → "Lineage & Impact"
- Component: `<ImpactAnalysisTab>` → `<UnifiedLineageTab initialDirection="both">`

### 2. `/frontend/src/pages/BusinessObjectDetailsPage.tsx`
**Changes:**
- Import changed: `ImpactAnalysisTab` → `UnifiedLineageTab`
- Component: `<ImpactAnalysisTab>` → `<UnifiedLineageTab initialDirection="both">`
- Tab already labeled "Lineage & Impact Analysis" (no change needed)

## How It Works Now

### Lineage API Call
**Before:**
```typescript
const url = `/api/lineage/node/${nodeId}/graph?depth=3&engine=cypher`;
```

**After:**
```typescript
const url = `/api/lineage/node/${nodeId}/graph?depth=3`;
```

Backend now uses relational SQL queries with recursive CTEs instead of AGE graph database.

### Unified Tab Features

The new **UnifiedLineageTab** provides:

1. **Direction Toggle** (3 modes):
   - **↑ Lineage** - Show only upstream dependencies
   - **⇅ Both** - Show complete bidirectional graph (default)
   - **↓ Impact** - Show only downstream impact

2. **Statistics Chips**:
   - Blue chip: X upstream nodes
   - Yellow chip: Y downstream nodes

3. **Floating Controls**:
   - [◉] Graph View
   - [📄] Explanation (adapts to direction)
   - [💬] AI Assistant (direction-aware)
   - [▣] Sidebar Toggle

4. **Client-Side Filtering**:
   - No API calls when switching directions
   - Instant filtering
   - All data fetched once

## Backend Integration

The backend's `FindBiDirectionalGraph()` now adds direction metadata to all nodes/edges:

```json
{
  "nodes": [
    {
      "id": "customer_table",
      "metadata": {
        "direction": "upstream",
        "is_lineage": true
      }
    }
  ],
  "edges": [
    {
      "from_id": "table1",
      "to_id": "table2",
      "metadata": {
        "direction": "downstream"
      }
    }
  ]
}
```

## Testing Checklist

- [x] Frontend builds successfully (39.77s)
- [ ] Navigate to semantic term details
- [ ] Verify Lineage & Relationships tab shows graph (no 500 error)
- [ ] Click "Lineage & Impact" tab (4th tab)
- [ ] Verify UnifiedLineageTab loads
- [ ] Test direction toggle (upstream/both/downstream)
- [ ] Verify statistics chips show counts
- [ ] Test explanation adapts to direction
- [ ] Test AI assistant
- [ ] Verify business object page also works

## What Was Removed

### ❌ Deleted Components
- None (kept `ImpactAnalysisTab` for backward compatibility)

### ❌ Removed UI Elements
- SQL/Cypher engine toggle (no longer needed)
- Separate "Impact Analysis" tab usage (replaced with unified tab)

### ❌ Removed Parameters
- `&engine=cypher` query parameter from lineage API calls

## Migration Notes

### Old Behavior
- Had to click between "Lineage" and "Impact Analysis" tabs
- Each tab made separate API calls
- AGE/Cypher engine option (now removed)

### New Behavior  
- Single "Lineage & Impact" tab
- One API call fetches complete graph
- Toggle between upstream/downstream/both instantly
- Relational SQL backend only

## Related Documentation

- [UNIFIED_LINEAGE_GUIDE.md](./UNIFIED_LINEAGE_GUIDE.md) - Complete feature guide
- [UNIFIED_LINEAGE_QUICKSTART.md](./UNIFIED_LINEAGE_QUICKSTART.md) - Quick start guide
- [AGE_REMOVAL_COMPLETE.md](./AGE_REMOVAL_COMPLETE.md) - AGE removal context

## Next Steps

1. Start backend: `cd backend && go run ./cmd/server`
2. Start frontend: `cd frontend && npm run dev`
3. Test semantic term lineage
4. Test business object lineage
5. Verify no 500 errors
6. Test direction toggle functionality

---

**Status:** ✅ COMPLETE - Build successful, ready for testing
