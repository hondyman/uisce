# Delta Save Implementation - Testing Guide

## Summary of Changes

### Frontend Changes
1. **EntityConfigPage.tsx**
   - Added `initialEntities` state to track baseline
   - Added `computeChanges` function using `useMemo` to calculate diff
   - Updated `saveAndApply` to send only changed/deleted entities
   - Updated button to show count of changes and disable when none

2. **entitySchema.ts**
   - Added `EntitySchemaDelta` type with `changed` and `deleted` fields
   - Updated `saveEntitySchema` to accept `EntitySchemaPayload` (full schema OR delta)

### Backend Changes
1. **api.go** `/entity-schema` endpoint
   - Detects if payload has `changed` and `deleted` fields
   - If delta: fetches existing schema, applies changes, applies deletions
   - If full: replaces schema entirely (backward compatible)
   - Merges at database level

## Testing Steps

### Step 1: Start Fresh
```bash
# Clear browser data to reset to initial state
# Or reload the page
```

### Step 2: Add a New Entity

1. Navigate to `/config`
2. Click the **+** button next to "Entities"
3. Enter name: "test_entity"
4. Click "Create"
5. Notice the **SAVE & APPLY** button shows **(1 changes)**

Expected in Network tab:
```json
POST /api/entity-schema
{
  "changed": {
    "test_entity": {
      "name": "test_entity",
      "entity_fields": [],
      "subtypes": {}
    }
  },
  "deleted": []
}
```

**NOT** the full schema with all entities.

### Step 3: Verify Database

After clicking SAVE & APPLY:
```bash
psql "postgresql://postgres:postgres@localhost:5432/alpha?sslmode=disable" \
  -c "SELECT schema_data FROM public.entity_schema WHERE tenant_id = '910638ba-a459-4a3f-bb2d-78391b0595f6' ORDER BY updated_at DESC LIMIT 1;" | jq . | grep -A 5 test_entity
```

Should see your new entity merged with existing ones (trades, clients, portfolios, etc).

### Step 4: Add a Field to Existing Entity

1. Select "Trades" in the entity list
2. Click **+** button next to "Fields"
3. Enter:
   - Field Level: "Entity (Trades)"
   - Field Name: "my_new_field"
   - Field Type: "text"
4. Click "Create"
5. Notice the **SAVE & APPLY** button shows **(1 changes)**

Expected in Network tab:
```json
POST /api/entity-schema
{
  "changed": {
    "trades": {
      "name": "Trades",
      "entity_fields": [
        // ... existing fields ...
        { "key": "my_new_field", "name": "my_new_field", "type": "text" }
      ],
      "subtypes": { ... }
    }
  },
  "deleted": []
}
```

Only the "trades" entity is sent, not the whole schema.

### Step 5: Make Multiple Changes

1. Add another field to Trades
2. Add a new entity
3. Notice **SAVE & APPLY** shows **(3 changes)** or similar

Expected in Network tab:
```json
POST /api/entity-schema
{
  "changed": {
    "trades": { /* updated with new field */ },
    "new_entity": { /* newly created */ }
  },
  "deleted": []
}
```

### Step 6: Verify No Changes Disables Button

1. Don't make any changes
2. **SAVE & APPLY** button should be **disabled** and show **(0 changes)**

### Step 7: Check Console Logs

Open DevTools → Console and look for:
```
[EntityConfigPage.saveAndApply] Changes detected: {changed: 1, deleted: 0, changedEntities: [...]}
[saveEntitySchema] Sending delta payload... {payload: {changed: {...}, deleted: [...]}}
[setupTenantFetch] Making request: {finalUrl: "...", method: "POST", hasBody: true}
[setupTenantFetch] Response received: {url: "...", status: 200, statusText: "OK"}
[saveEntitySchema] Save successful: {result: {success: true, message: "Entity schema saved successfully"}}
[EntityConfigPage.saveAndApply] Success! Saved 1 entities
```

## What Should NOT Happen

❌ Button should NOT send all entities when you only added one
❌ Request body should NOT include "clients", "portfolios", etc. if only "trades" changed
❌ Size of request body should be much smaller now

## Verify Backward Compatibility

The backend still accepts full schemas for backward compatibility:

```bash
# Old-style full schema POST should still work
curl -X POST http://localhost:8080/api/entity-schema \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 910638ba-a459-4a3f-bb2d-78391b0595f6" \
  -H "X-Tenant-Datasource-ID: 982aef38-418f-46dc-acd0-35fe8f3b97b0" \
  -d '{
    "trades": { "name": "Trades", "entity_fields": [...], "subtypes": {...} },
    "clients": { "name": "Clients", "entity_fields": [], "subtypes": {...} }
  }'
```

This should still work and replace the entire schema.

## Payload Size Comparison

### Before (Sending Full Schema)
```
Total size: ~2KB-5KB (all entities sent every time)
```

### After (Sending Only Deltas)
```
Adding one field: ~200 bytes
Adding one entity: ~300 bytes
Changes to 1 entity: ~500 bytes
```

Check Network tab → Request size to confirm reduction.

## If Something Breaks

### Issue: "No changes to save" message when you DID make changes

**Cause**: `computeChanges` isn't detecting changes correctly

**Debug**:
```javascript
// In console:
const initial = JSON.parse(localStorage.getItem('entityInitial') || '{}');
const current = JSON.parse(localStorage.getItem('entityCurrent') || '{}');
console.log('Different?', JSON.stringify(initial) !== JSON.stringify(current));
```

### Issue: "0 changes" but you can see modifications in the UI

**Cause**: State not updating properly

**Debug**:
1. Open DevTools → React tab
2. Check EntityConfigPage component state
3. Verify `initialEntities` vs `entities` are different

### Issue: Backend receives delta but only applies changes, not full schema

**Cause**: This is correct behavior! The backend merges new/changed entities with existing ones.

**Verify**:
```bash
psql "postgresql://postgres:postgres@localhost:5432/alpha?sslmode=disable" \
  -c "SELECT json_object_keys(schema_data) FROM public.entity_schema LIMIT 1;"
```

Should list all entities (old + new), not just the changed ones.

## Success Criteria

✅ SAVE & APPLY button shows correct change count
✅ Request body only contains changed entities
✅ Request body is much smaller than before
✅ Backend successfully merges changes with existing schema
✅ All entities still present after saving partial update
✅ No errors in browser console
✅ No errors in backend logs

---

**Next Steps**: If all tests pass, the delta save implementation is complete and ready for production!
