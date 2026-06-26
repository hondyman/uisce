# Related Objects Tab - Troubleshooting Guide

## Error: "Cannot read properties of null (reading 'relationships')"

### Root Cause Analysis

This error typically occurs when:
1. The API response is null or malformed
2. The backend isn't running
3. The tenant scope headers are missing
4. The backend query returns no data

### ✅ Fixes Applied

1. **Better null handling in frontend** (`relationships.ts`)
   - Added null checks before accessing `data.relationships`
   - Returns empty array gracefully on null responses
   - Better error messages with diagnostic info

2. **Enhanced error messages** (`RelatedObjectsTab.tsx`)
   - Shows detailed troubleshooting steps
   - Displays entity name being searched
   - Links to browser console logs

3. **Improved backend logging** (`api.go`)
   - Logs when no relationships are found (not an error)
   - Logs discovery parameters for debugging
   - Better error messages with context

4. **Response validation**
   - Checks if response is array or object
   - Validates relationships property is array
   - Gracefully handles edge cases

---

## Diagnostic Checklist

### 1. Backend API Running

```bash
# Check if backend is running
curl -i http://localhost:8080/health

# Should return 200 OK with status: healthy
```

### 2. Endpoint Accessible

```bash
# Check if the relationships endpoint exists
curl -i "http://localhost:8080/api/relationships/objects?tenant_id=test&datasource_id=test&entity=test"

# Should return 200 (with or without relationships)
# NOT 404 (endpoint not found)
```

### 3. Tenant Scope Selected

```javascript
// In browser console:
console.log(localStorage.getItem('selected_tenant'));
console.log(localStorage.getItem('selected_datasource'));

// Both should have values, not null
```

### 4. Entity Name Matches

The entity name must exactly match what's in the database (case-sensitive):
```javascript
// In browser console when viewing Related Objects tab:
console.log('Looking for entity:', window.__lastEntityName);
```

### 5. Semantic Terms Exist

```sql
-- Check if entity has semantic terms
SELECT ct.node_name as entity, COUNT(*) as semantic_term_count
FROM catalog_node ct
JOIN catalog_edge ce ON ce.source_node_id = ct.id
WHERE ct.catalog_type_name = 'business_term'
GROUP BY ct.node_name;
```

### 6. Semantic Terms Mapped to Columns

```sql
-- Check if semantic terms are mapped to columns
SELECT COUNT(*) as mapped_count
FROM catalog_edge ce
JOIN catalog_edge_type cet ON cet.id = ce.edge_type_id
WHERE cet.predicate = 'member of'
  AND ce.relationship_type = 'MAPS_TO';
```

### 7. Foreign Keys Exist

```sql
-- Check if there are any foreign key edges
SELECT COUNT(*) as fk_count
FROM catalog_edge ce
JOIN catalog_edge_type cet ON cet.id = ce.edge_type_id
WHERE cet.predicate = 'foreign_key'
  AND ce.relationship_type = 'foreign_key';
```

---

## Debugging Steps

### Step 1: Check Browser Console

Open developer tools (F12) and look for logs like:
```
🔗 Fetching relationships for entity: {entityName, tenantId, datasourceId}
✅ Relationships fetched: [... array of relationships]
```

Or if there's an error:
```
Error fetching relationships: [error message]
```

### Step 2: Check Backend Logs

Look for backend logs:
```
getRelatedObjects called with: tenantID=xxx, datasourceID=yyy, entity=EntityName
Discovered 5 related entities for EntityName
```

Or if there's an issue:
```
No related entities found for entity EntityName in datasource xxx
Failed to discover related objects for entity EntityName: ...
```

### Step 3: Manual API Test

```bash
# With actual tenant and datasource IDs:
curl -X GET "http://localhost:8080/api/relationships/objects?tenant_id=YOUR_TENANT_ID&datasource_id=YOUR_DATASOURCE_ID&entity=YourEntity" \
  -H "X-Tenant-ID: YOUR_TENANT_ID" \
  -H "X-Tenant-Datasource-ID: YOUR_DATASOURCE_ID" \
  -H "Content-Type: application/json"

# Should return:
# {
#   "sourceEntity": "YourEntity",
#   "relationships": [...],
#   "count": 5
# }
```

---

## Common Scenarios and Solutions

### Scenario 1: Empty Relationships (No Error)

**What You See:**
- "No relationships defined yet" message
- No error messages

**Causes:**
- Entity has no semantic terms
- Semantic terms not mapped to columns
- No foreign keys in database
- Entity name doesn't match

**Solution:**
1. Verify entity has semantic terms created
2. Check semantic terms are mapped to columns via catalog_edge
3. Verify foreign keys exist in database schema
4. Confirm entity name matches exactly

### Scenario 2: Backend Not Responding

**What You See:**
- Error message about endpoint not found
- Network error in browser console

**Causes:**
- Backend not running
- Port 8080 not accessible
- Firewall blocking connection

**Solution:**
1. Start backend: `go run ./backend/cmd/api-gateway`
2. Verify port: `netstat -an | grep 8080`
3. Check firewall settings

### Scenario 3: Tenant Scope Not Selected

**What You See:**
- Error: "Invalid request parameters"
- No tenant/datasource in localStorage

**Causes:**
- Haven't selected tenant in UI
- Session expired
- Tenant picker not working

**Solution:**
1. Click tenant selector in top-right
2. Choose tenant, product, datasource
3. Refresh page
4. Try again

### Scenario 4: Entity Name Case Mismatch

**What You See:**
- "No relationships defined yet"
- Backend logs show no entities found

**Causes:**
- Entity name in UI doesn't match database
- Catalog uses different casing

**Solution:**
1. Check exact entity name in database
2. Verify semantic terms use same name
3. Use exact case when searching

---

## Files Updated

✅ **backend/internal/api/api.go**
- Enhanced `getRelatedObjects()` with better logging
- Added info log when no relationships found

✅ **frontend/src/api/relationships.ts**
- Better null/undefined checks
- More detailed error messages
- Response format validation
- Error re-throwing (not swallowing)

✅ **frontend/src/components/relationship/RelatedObjectsTab.tsx**
- Better error display with troubleshooting steps
- Shows entity name in error message
- Lists diagnostic steps

---

## Testing the Fix

### Test 1: With Valid Entity

1. Navigate to Entity Details Page
2. Select entity with semantic terms and FKs
3. Click "Related Objects" tab
4. Should see relationships or helpful error message

### Test 2: With Invalid Entity

1. Click entity without semantic terms
2. Click "Related Objects" tab  
3. Should see "No relationships defined yet" (not error)

### Test 3: Without Tenant Scope

1. Clear localStorage: `localStorage.clear()`
2. Refresh page
3. Click "Related Objects" tab
4. Should see helpful error about tenant scope

### Test 4: Backend Down

1. Stop backend
2. Click "Related Objects" tab
3. Should see error about API not responding
4. Check console for detailed logs

---

## Next Steps if Still Failing

If you're still seeing errors after these fixes:

1. **Check exact error message** - Copy full error from browser console
2. **Check backend logs** - Look for `Discovered X related entities`
3. **Run manual SQL** - Execute queries above to verify data
4. **Check tenant IDs** - Verify tenant_id and datasource_id match
5. **Check entity name** - Make sure it's exact match (case-sensitive)

---

## Performance Optimization

If loading is slow (>5 seconds):

```sql
-- Add recommended indexes
CREATE INDEX idx_ce_tenant_src_tgt ON catalog_edge(
  tenant_datasource_id, 
  source_node_id, 
  target_node_id
);

CREATE INDEX idx_ce_tenant_rel_type ON catalog_edge(
  tenant_datasource_id, 
  relationship_type
);
```

---

## Documentation

For more detailed information, see:
- `RELATED_OBJECTS_IMPLEMENTATION_GUIDE.md` - Full technical reference
- `RELATED_OBJECTS_TAB_COMPLETE.md` - Quick start guide
- Code comments in implementation files

