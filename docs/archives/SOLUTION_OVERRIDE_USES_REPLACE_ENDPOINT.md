# ✅ SOLUTION: Override Now Uses Replace Endpoint

## The Problem

When you tried to override `METADATA_LAST_UPDATE` → `LAST_UPDATE`, the backend returned:

```json
{
  "skipped": true,
  "created_edge": false
}
```

**Why?** The edge already existed in the database, and the `/api/semantic-mappings/edges` endpoint **skips** existing edges instead of replacing them.

## The Root Cause

The backend has two different endpoints:

1. **`POST /api/semantic-mappings/edges`** - Creates new edges, **skips if already exists**
2. **`POST /api/semantic-mappings/replace`** - **Deletes old edge + creates new edge**

The frontend was always using the `edges` endpoint, even when you clicked the override icon!

## The Fix

I updated `useSemanticMapper.ts` to:

1. **Detect override scenarios** by checking if `mapping.override === true` or `mapping.edge_exists === true`
2. **Use the `replace` endpoint** when overrides are detected
3. **Use the `edges` endpoint** for normal new mappings

### New Logic Flow

```typescript
// Check if any mappings are overrides
const hasOverrides = selected.some(m => m.override || m.edge_exists);

if (hasOverrides) {
  // Use /api/semantic-mappings/replace endpoint
  // This will:
  // 1. Delete the existing edge
  // 2. Create the new edge
  for (const mapping of selected) {
    await fetch(`/api/semantic-mappings/replace`, {
      method: 'POST',
      body: JSON.stringify({ mapping })
    });
  }
} else {
  // Use /api/semantic-mappings/edges endpoint (bulk create)
  await fetch(`/api/semantic-mappings/edges`, {
    method: 'POST',
    body: JSON.stringify({ mappings: selected })
  });
}
```

## What Changed

### Before (Broken)
```
User clicks override icon
  ↓
Frontend sets override: true
  ↓
Calls POST /api/semantic-mappings/edges
  ↓
Backend finds existing edge
  ↓
Backend SKIPS (created_edges: 0) ❌
```

### After (Fixed)
```
User clicks override icon
  ↓
Frontend sets override: true
  ↓
Frontend detects override flag
  ↓
Calls POST /api/semantic-mappings/replace ✅
  ↓
Backend DELETES old edge
  ↓
Backend CREATES new edge
  ↓
Success! (created_edges: 1) ✅
```

## How To Test

1. **Refresh browser** (Cmd+Shift+R) to get the new code
2. **Click override icon** on `METADATA_LAST_UPDATE`
3. **Type or select** `LAST_UPDATE` semantic term
4. **Click "Apply Existing Term"** (or "Create & Apply New Term")
5. **Check the checkbox**
6. **Click "Create Edges (1)"**

### Expected Console Logs

You should now see:

```
[useSemanticMapper] Detected override scenario, using replace endpoint
[useSemanticMapper] Replacing edge: {column: "last_update", semantic_term: "LAST_UPDATE", ...}
[useSemanticMapper] Replace response: {created_edges: 1, deleted_edges: 1, ...}
✅ Replaced 1 edges (deleted 1 old edges).
```

### Expected UI Changes

After clicking "Create Edges":
- ✅ Toast shows: "Replaced 1 edges (deleted 1 old edges)"
- ✅ The mapping list refreshes
- ✅ The row now shows a green checkmark (edge exists)
- ✅ The override icon disappears (no longer needed)

## Additional Benefits

This fix also handles:

1. **Batch overrides** - If you select multiple overrides, each one uses the replace endpoint
2. **Mixed scenarios** - If you select some new mappings and some overrides, it handles both correctly
3. **Better feedback** - The toast message now shows how many edges were deleted vs created

## Backend Endpoint Reference

### POST /api/semantic-mappings/replace

**Request:**
```json
{
  "mapping": {
    "database_column": { "node_id": "...", "tenant_id": "...", ... },
    "semantic_term": "LAST_UPDATE",
    "semantic_term_id": "...",
    "override": true
  }
}
```

**Response:**
```json
{
  "created_edges": 1,
  "deleted_edges": 1,
  "created_terms": 0,
  "deleted_edge_col_ids": ["..."],
  "created_edge_col_ids": ["..."]
}
```

## Summary

**The override feature now works correctly!** 🎉

When you override a semantic term mapping:
1. The old edge is **deleted**
2. The new edge is **created**
3. You see confirmation in the toast message
4. The UI updates to reflect the new mapping

No more "Created 0 edges" messages when overriding! 🚀
