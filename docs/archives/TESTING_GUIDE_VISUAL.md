# Quick Visual Guide: Testing the Fixes

## Test 1: Search Focus Issue ✅

### BEFORE (Broken):
```
You type: L
Input shows: L
Input loses focus: ❌
You click back in
You type: A
Input shows: A    ← Lost the "L"!
Input loses focus: ❌
```

### AFTER (Fixed):
```
You type: L-A-S-T-_-U-P-D-A-T-E
Input shows: LAST_UPDATE
Focus stays: ✅✅✅✅✅✅✅✅✅✅✅
```

---

## Test 2: Edge Creation ✅

### BEFORE (Broken):
```
1. Override → Type "LAST_UPDATE"
2. Create term → ✅ Term created
3. Click "Create Edges" → Shows "Created 0 edges" ❌
4. Refresh page → Still shows "METADATA_LAST_UPDATE" ❌
```

### AFTER (Fixed):
```
1. Override → Type "LAST_UPDATE"
2. Create term → ✅ Term created
3. Console shows:
   {
     column: "METADATA_LAST_UPDATE",
     semantic_term: "LAST_UPDATE",
     semantic_term_id: "abc123...",
     has_tenant_id: true,      ← ✅ Key!
     has_datasource_id: true   ← ✅ Key!
   }
4. Click "Create Edges" → Shows "Created 1 edges" ✅
5. Refresh page → Shows "LAST_UPDATE" with green "Mapped" chip ✅
```

---

## What to Check in Browser Console

### When clicking "Create Edges", you should see:
```javascript
[SemanticMapper] Creating edges for mappings: [
  {
    column: "METADATA_LAST_UPDATE",
    semantic_term: "LAST_UPDATE",
    semantic_term_id: "4d8f234a-...",
    is_new_term: true,
    override: true,
    edge_exists: false,
    has_tenant_id: true,      // ← Must be true!
    has_datasource_id: true,  // ← Must be true!
    full_db_column: {
      schema: "public",
      table: "my_table",
      column: "METADATA_LAST_UPDATE",
      node_id: "col-123...",
      tenant_id: "00000000-0000-0000-0000-000000000000",
      tenant_datasource_id: "11111111-1111-1111-1111-111111111111"
    }
  }
]
```

---

## What to Check in Network Tab

### Request: POST /api/semantic-mappings/edges

**Payload should look like:**
```json
{
  "mappings": [
    {
      "database_column": {
        "schema": "public",
        "table": "my_table", 
        "column": "METADATA_LAST_UPDATE",
        "node_id": "col-123...",
        "tenant_id": "00000000-0000-0000-0000-000000000000",      ← ✅ Must be present
        "tenant_datasource_id": "11111111-1111-1111-1111-111111111111"  ← ✅ Must be present
      },
      "semantic_term": "LAST_UPDATE",
      "semantic_term_id": "4d8f234a-...",
      "is_new_term": true,
      "confidence": 1.0,
      "override": true
    }
  ]
}
```

**Response should show:**
```json
{
  "created_edges": 1,
  "created_terms": 0,
  "skipped_existing": 0,
  "per_mapping_results": [
    {
      "col_node_id": "col-123...",
      "semantic_term_id": "4d8f234a-...",
      "created_edge": true,
      "skipped": false
    }
  ]
}
```

---

## Visual Indicators in UI

### Ready to Create Edge:
```
┌─────────────────────────────────────────────────────────────┐
│ ☑ METADATA_LAST_UPDATE → LAST_UPDATE                       │
│                                                              │
│ 🟢 Ready to Create Edge  ← This should pulse/glow          │
│ ✏️ Override Active                                          │
└─────────────────────────────────────────────────────────────┘
```

### After Creating Edge:
```
┌─────────────────────────────────────────────────────────────┐
│ ☐ METADATA_LAST_UPDATE → LAST_UPDATE                       │
│                                                              │
│ 🟢 Mapped  ← Changed from "Ready to Create Edge"           │
│ 🔗 Link icon shown                                          │
└─────────────────────────────────────────────────────────────┘
```

---

## Quick Test Script

Copy/paste this into browser console to verify tenant scope is set:

```javascript
// Check if tenant scope exists
const tenant = localStorage.getItem('selected_tenant');
const datasource = localStorage.getItem('selected_datasource');

console.log('Tenant:', tenant ? JSON.parse(tenant) : '❌ NOT SET');
console.log('Datasource:', datasource ? JSON.parse(datasource) : '❌ NOT SET');

if (!tenant || !datasource) {
  console.error('⚠️ TENANT SCOPE NOT SET! Select tenant in UI first.');
} else {
  console.log('✅ Tenant scope is configured');
}
```

**Expected output:**
```
Tenant: {id: "00000000-0000-0000-0000-000000000000", display_name: "Demo Tenant"}
Datasource: {id: "11111111-1111-1111-1111-111111111111", source_name: "demo_db"}
✅ Tenant scope is configured
```

---

## Common Issues & Solutions

### Issue: "Created 0 edges"
**Symptom:** Success message says "Created 0 edges"

**Check:**
1. Console log shows `has_tenant_id: false` or `has_datasource_id: false`
2. Network payload missing `tenant_id` or `tenant_datasource_id`

**Solution:**
- Verify tenant selector is used (top of page)
- Check localStorage for tenant selection
- Reload page after selecting tenant

---

### Issue: Search still loses focus
**Symptom:** Can only type one character at a time

**Check:**
1. Hard reload browser (Cmd+Shift+R)
2. Clear React DevTools warnings
3. Check if any browser extensions interfere

**Solution:**
- Clear browser cache
- Try incognito mode
- Check React version compatibility
- Verify Vite HMR updated the component

---

### Issue: Term created but edge not created
**Symptom:** Term exists but mapping not updated

**Check:**
1. Row is selected (checkbox checked)
2. "Ready to Create Edge" chip shows
3. Console log shows valid `semantic_term_id`

**Solution:**
- Verify row checkbox is checked
- Click "Create Edges" button (not just the row)
- Check if edge already exists (green "Mapped" chip)

---

## Success Criteria

✅ Can type continuously in search without clicking back in
✅ Console shows tenant IDs when creating edges
✅ Success message shows "Created N edges" where N > 0
✅ After refresh, mapping shows new semantic term
✅ Green "Mapped" chip appears after edge creation
✅ No errors in browser console
✅ No errors in network tab responses
