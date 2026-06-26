# Cascading Filter Testing Guide

## Problem
The cascading filters in SemanticMapper should filter:
- **Tables** by selected schema (parent_id = schema.id)
- **Columns** by selected table (parent_id = table.id)

## Root Cause Analysis

The code implementation is **correct**:
- ✅ Backend SQL filters by `parent_id` when provided
- ✅ Frontend passes `scopeSchemaId` as parentId to table typeahead
- ✅ Frontend passes `scopeTableId` as parentId to column typeahead
- ✅ CatalogNodeTypeahead sends parentId in API requests
- ✅ Database has correct parent_id relationships

The issue is: **Tenant scope must be selected first!**

## Why Tenant Scope is Required

The `setupTenantFetch.ts` shim intercepts ALL `/api/catalog/nodes` requests and:
1. Checks if tenant + datasource are selected
2. If missing → **rejects the request** with error
3. If present → adds `tenant_id` and `datasource_id` query params + headers

Without tenant scope, the API requests never reach the backend!

## Testing Steps

### 1. Seed Tenant Scope

Run this in your terminal:
```bash
# Copy the seed script to clipboard
cat seed_tenant_scope.js | pbcopy
```

Then:
1. Open http://localhost:3000/semantic-mapper in your browser
2. Open DevTools Console (Cmd+Option+J on Mac)
3. Paste the script and press Enter
4. You should see: "✅ Tenant scope seeded successfully!"
5. **Reload the page** (Cmd+R)

### 2. Verify Tenant Scope

In the console, run:
```javascript
JSON.parse(localStorage.getItem('selected_tenant'))
JSON.parse(localStorage.getItem('selected_datasource'))
```

You should see:
```javascript
{id: "910638ba-a459-4a3f-bb2d-78391b0595f6", display_name: "Northwind", ...}
{id: "982aef38-418f-46dc-acd0-35fe8f3b97b0", source_name: "Northwind Database", ...}
```

### 3. Test Cascading Filters

1. Go to the **Schema** dropdown
2. Open it and select a schema (e.g., "public")
3. Watch the console logs:
   ```
   [CatalogNodeTypeahead] Mounted/updated: nodeType="schema", parentId="none"
   [CatalogNodeTypeahead] Searching: nodeType="schema", q="", parentId="none"
   [CatalogNodeTypeahead] URL: /api/catalog/nodes?q=&limit=50&type=schema
   [CatalogNodeTypeahead] Got 9 results
   ```

4. Now open the **Table** dropdown
5. You should see console logs showing parent_id is the schema you selected:
   ```
   [CatalogNodeTypeahead] Mounted/updated: nodeType="table", parentId="<schema-uuid>"
   [CatalogNodeTypeahead] Searching: nodeType="table", q="", parentId="<schema-uuid>"
   [CatalogNodeTypeahead] URL: /api/catalog/nodes?q=&limit=50&type=table&parent_id=<schema-uuid>
   [CatalogNodeTypeahead] Got 15 results
   ```

6. Select a table and open the **Columns** dropdown
7. You should see parent_id is the table you selected:
   ```
   [CatalogNodeTypeahead] Mounted/updated: nodeType="column", parentId="<table-uuid>"
   [CatalogNodeTypeahead] Searching: nodeType="column", q="", parentId="<table-uuid>"
   [CatalogNodeTypeahead] URL: /api/catalog/nodes?q=&limit=50&type=column&parent_id=<table-uuid>
   [CatalogNodeTypeahead] Got 23 results
   ```

### 4. Verify API Requests in Network Tab

1. Open DevTools Network tab
2. Filter by "catalog/nodes"
3. Select a schema and watch the API requests
4. Click on a request and check:
   - **Query String Parameters** should include:
     - `tenant_id=910638ba-a459-4a3f-bb2d-78391b0595f6`
     - `datasource_id=982aef38-418f-46dc-acd0-35fe8f3b97b0`
     - `type=table`
     - `parent_id=<schema-uuid>` (when filtering tables)
   - **Request Headers** should include:
     - `X-Tenant-ID: 910638ba-a459-4a3f-bb2d-78391b0595f6`
     - `X-Tenant-Datasource-ID: 982aef38-418f-46dc-acd0-35fe8f3b97b0`

### 5. Expected Behavior

✅ **Schema dropdown**: Shows all schemas for the tenant/datasource (no parent filter)
✅ **Table dropdown**: Only shows tables belonging to the selected schema
✅ **Column dropdown**: Only shows columns belonging to the selected table
✅ **When schema changes**: Table and column selections are cleared
✅ **When table changes**: Column selection is cleared
✅ **Console logs**: Show parentId values matching the cascade hierarchy

## Troubleshooting

### No tenant warning shows
- The yellow warning banner should appear at the top if tenant scope is missing
- If it doesn't appear, check that `hasTenantScope()` is working

### API returns 400 "tenant_id and datasource_id required"
- Tenant scope is not set in localStorage
- Run the seed script again and reload

### Dropdown shows all items regardless of selection
- Check console logs for parentId values
- If parentId is "none" when it should have a UUID, the scope state is not being passed correctly
- Verify scopeSchemaId and scopeTableId are populated in ScopeContext

### Network requests missing tenant_id/datasource_id
- The setupTenantFetch shim is not loaded
- Check that `import './setupTenantFetch'` is in `main.tsx`

### Database has no parent_id values
- Run this query to verify:
  ```sql
  SELECT id, node_name, parent_id 
  FROM catalog_node cn
  JOIN catalog_node_type cnt ON cn.node_type_id = cnt.id
  WHERE cnt.catalog_type_name = 'table'
  LIMIT 10;
  ```
- If parent_id is NULL for tables, the catalog ingestion didn't set parent relationships

## Database Verification

Run these queries to verify data integrity:

```sql
-- Check tenant and datasource in catalog_node
SELECT DISTINCT tenant_id, tenant_datasource_id 
FROM catalog_node;

-- Check schemas (should have NULL parent_id)
SELECT cn.id, cn.node_name, cn.parent_id
FROM catalog_node cn
JOIN catalog_node_type cnt ON cn.node_type_id = cnt.id
WHERE cnt.catalog_type_name = 'schema'
  AND cn.tenant_id = '910638ba-a459-4a3f-bb2d-78391b0595f6'
ORDER BY cn.node_name
LIMIT 10;

-- Check tables (should have parent_id = schema.id)
SELECT cn.id, cn.node_name, cn.parent_id
FROM catalog_node cn
JOIN catalog_node_type cnt ON cn.node_type_id = cnt.id
WHERE cnt.catalog_type_name = 'table'
  AND cn.tenant_id = '910638ba-a459-4a3f-bb2d-78391b0595f6'
ORDER BY cn.node_name
LIMIT 10;

-- Check columns (should have parent_id = table.id)
SELECT cn.id, cn.node_name, cn.parent_id
FROM catalog_node cn
JOIN catalog_node_type cnt ON cn.node_type_id = cnt.id
WHERE cnt.catalog_type_name = 'column'
  AND cn.tenant_id = '910638ba-a459-4a3f-bb2d-78391b0595f6'
ORDER BY cn.node_name
LIMIT 10;
```

## Summary

The cascading filter **code is correct**. The blocker was:
1. Missing tenant scope in localStorage
2. User didn't know tenant selection was required
3. No clear error message when tenant missing

**Solutions implemented:**
- ✅ Added tenant scope warning banner at top of SemanticMapper
- ✅ Added detailed console logging to track parentId flow
- ✅ Created seed script to quickly set tenant scope for testing
- ✅ Improved error messages in API calls
- ✅ Removed invalid `disabled` props from typeahead

**Next steps:**
1. Run the seed script
2. Reload the page
3. Test the cascade behavior
4. Watch console logs to verify parentId is passed correctly
