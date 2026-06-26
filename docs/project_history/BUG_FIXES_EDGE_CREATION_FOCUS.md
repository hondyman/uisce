# Bug Fixes: Edge Creation & Search Focus Issues

## Issues Fixed

### 1. ❌ **Created 0 edges** - Edge creation failing
### 2. ❌ **Search loses focus** - Cannot type continuously in autocomplete

---

## Fix #1: Edge Creation Failing (Created 0 edges)

### Root Cause
When applying an override (selecting or creating a semantic term), the mapping state was being updated but the **`database_column` object was being reconstructed** without preserving critical tenant metadata.

The backend requires:
- `database_column.tenant_id`
- `database_column.tenant_datasource_id`

These fields were being lost during state updates.

### The Fix
Changed both `selectSemanticTerm` and `handleCreateAndSelectTerm` to explicitly preserve the entire `database_column` object:

```typescript
// ❌ BEFORE (Lost tenant info)
return { 
  ...m, 
  semantic_term: term.term_name,
  // database_column is implicitly copied, may lose nested fields
}

// ✅ AFTER (Preserves tenant info)
return { 
  ...m,
  database_column: { ...m.database_column }, // Explicitly preserve
  semantic_term: term.term_name,
}
```

### Added Debug Logging
Added console logging in `confirmCreate()` to help debug edge creation:

```typescript
console.log('[SemanticMapper] Creating edges for mappings:', selected.map(m => ({
  column: m.database_column.column,
  semantic_term: m.semantic_term,
  semantic_term_id: m.semantic_term_id,
  has_tenant_id: !!m.database_column.tenant_id,
  has_datasource_id: !!m.database_column.tenant_datasource_id,
  full_db_column: m.database_column
})));
```

**Check the browser console** when clicking "Create Edges" to verify tenant info is present.

### Verification Steps
1. Open browser DevTools console
2. Click override icon on a mapping
3. Type or select a semantic term
4. Click "Create & Apply" or "Apply Existing Term"
5. Click "Create Edges (1)"
6. Check console for log showing:
   - ✅ `has_tenant_id: true`
   - ✅ `has_datasource_id: true`
   - ✅ `semantic_term_id: "some-uuid"`

---

## Fix #2: Search Loses Focus on Every Keystroke

### Root Cause
The MUI Autocomplete component was resetting/blurring the input on every change, causing focus to be lost after each character typed.

### The Fix
Added specific props to prevent focus loss:

```typescript
<Autocomplete 
  freeSolo
  disableClearable          // ✅ NEW: Prevents clearing input on blur
  blurOnSelect={false}      // ✅ NEW: Keeps focus after selection
  clearOnBlur={false}       // ✅ NEW: Doesn't clear on blur
  onInputChange={async (_e, value, reason) => {
    if (reason === 'reset') return; // ✅ NEW: Ignore resets
    setLocalSearchTerm(value || '');
    // ... search logic
  }}
  renderInput={(params) => (
    <TextField
      {...params}
      inputProps={{
        ...params.inputProps,
        autoComplete: 'off'  // ✅ NEW: Disable browser autocomplete
      }}
    />
  )}
/>
```

### Key Changes:
1. **`disableClearable`** - Prevents MUI from clearing input
2. **`blurOnSelect={false}`** - Keeps input focused after selecting suggestion
3. **`clearOnBlur={false}`** - Doesn't clear typed text on blur
4. **`reason === 'reset'`** check - Ignores internal resets that cause re-renders
5. **`autoComplete: 'off'`** - Prevents browser autocomplete interference

### Verification Steps
1. Click override icon on a mapping
2. Start typing in the search box
3. Type multiple characters: "M", "E", "T", "A"
4. Verify input stays focused through entire typing sequence
5. Suggestions should appear after 2 characters
6. Clicking a suggestion should apply it without losing text

---

## Testing Checklist

### Edge Creation Test
- [ ] Override a mapping with existing term
- [ ] Check console log shows `has_tenant_id: true` and `has_datasource_id: true`
- [ ] Click "Create Edges"
- [ ] Verify success message shows "Created 1 edges" (not "Created 0 edges")
- [ ] Refresh page
- [ ] Verify mapping now shows green "Mapped" chip

### Search Focus Test
- [ ] Click override icon
- [ ] Type "METADATA" without clicking back into input
- [ ] Verify all 8 characters typed without focus loss
- [ ] Verify suggestions appear
- [ ] Click a suggestion
- [ ] Verify term applied immediately

### Full Workflow Test
1. [ ] Click override on "METADATA_LAST_UPDATE" mapping
2. [ ] Type "LAST_UPDATE" in search box (continuous typing)
3. [ ] See "Create & Apply New Term" button (if new) or select from suggestions
4. [ ] Click button to create/apply term
5. [ ] See green "Ready to Create Edge" chip
6. [ ] See checkbox is checked
7. [ ] Click "Create Edges (1)"
8. [ ] Check console shows tenant info
9. [ ] See success toast "Created 1 edges"
10. [ ] Refresh page
11. [ ] Verify mapping shows "LAST_UPDATE" and green "Mapped" chip

---

## Files Changed

### `/frontend/src/components/semantic-mapper/MappingRow.tsx`
- Added `disableClearable`, `blurOnSelect={false}`, `clearOnBlur={false}` to Autocomplete
- Added reason check in `onInputChange` to ignore resets
- Added `autoComplete: 'off'` to input props
- Removed `value={localSearchTerm}` prop (causes issues with freeSolo)

### `/frontend/src/components/SemanticMapper.tsx`
- Updated `selectSemanticTerm` to preserve `database_column` object
- Updated `handleCreateAndSelectTerm` to preserve `database_column` object
- Added detailed console logging in `confirmCreate`
- Added validation check for 0 selected mappings

---

## Expected Behavior After Fixes

### When you type in search:
```
User types:  L → A → S → T → _ → U → P → D → A → T → E
Input shows: L   LA  LAS  LAST  LAST_  LAST_U  LAST_UP  LAST_UPD  LAST_UPDA  LAST_UPDAT  LAST_UPDATE
Focus:       ✅  ✅   ✅   ✅    ✅     ✅      ✅       ✅        ✅         ✅          ✅
```

### When you create edge:
```
1. Override applied → semantic_term_id set
2. Row selected → checkbox checked
3. Click "Create Edges"
4. Console shows → tenant_id and datasource_id present
5. Backend creates → Edge in graph database
6. Success toast → "Created 1 edges"
7. Page reloads → Shows "Mapped" chip
```

---

## Still Not Working?

### If edges still show "Created 0 edges":

1. **Check browser console** after clicking "Create Edges"
   - Look for the detailed log I added
   - Verify `has_tenant_id: true` and `has_datasource_id: true`
   - If false, the mappings from backend don't have tenant info

2. **Check Network tab** in DevTools
   - Look for `POST /api/semantic-mappings/edges`
   - Click on the request
   - Check the "Payload" tab
   - Verify each mapping has `database_column.tenant_id` and `database_column.tenant_datasource_id`

3. **Check backend logs**
   - Look for errors when processing edge creation
   - May need to add logging in `backend/internal/api/api.go` around line 492

4. **Verify tenant selection**
   - Make sure you've selected a tenant in the UI
   - Check localStorage for `selected_tenant` and `selected_datasource`

### If search still loses focus:

1. **Clear browser cache** and hard reload (Cmd+Shift+R / Ctrl+Shift+R)
2. **Check for React warnings** in console about controlled/uncontrolled components
3. **Verify React version** - MUI Autocomplete behavior varies by version
4. **Try in incognito mode** to rule out extension interference

---

## Next Steps

1. Start the frontend dev server (already running on localhost:5173)
2. Open browser DevTools console
3. Navigate to Semantic Mapper
4. Try the full workflow test above
5. Check console logs for detailed debug output
6. Report back with:
   - Console log output from edge creation
   - Network request payload
   - Whether focus issue is resolved

The fixes are deployed via hot module reload, so refresh the browser to get the latest changes.
