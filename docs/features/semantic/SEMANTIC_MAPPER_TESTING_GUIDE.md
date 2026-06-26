# Semantic Mapper Datamart Testing Guide

## Quick Test Steps

1. **Start the application**
   - Ensure backend is running
   - Frontend should be running on http://localhost:5173

2. **Navigate to Semantic Mapper**
   - Go to `http://localhost:5173/core/semantic-mapper`

3. **Select Datamart Datasource**
   - Click the tenant/datasource picker (usually in the header)
   - Select a tenant
   - Select a product
   - Select "datamart" as the datasource
   - Confirm selection

4. **Verify Column Loading**
   - Wait for the page to load
   - You should see a list of database columns in the semantic mapper
   - If empty, check browser console for errors
   - Check backend logs for: `Mapped 'datamart' datasource to 'alpha_dwh' ID:`

5. **Test Mapping Creation**
   - Select one or more columns (checkboxes)
   - Click "Create Edges" button
   - Should see success message: "Created X edges"
   - Check the "edge_exists" column changes for selected rows

6. **Verify in Database**
   - Query the catalog_edge table:
     ```sql
     SELECT id, source_node_id, target_node_id, relationship_type 
     FROM catalog_edge 
     WHERE tenant_datasource_id = '<alpha_dwh_id>' 
     LIMIT 10;
     ```
   - Should see edges with relationship_type = "MAPS_TO"

## Expected Behavior After Fix

✅ **GenerateMappings Endpoint**
- Returns list of columns from alpha_dwh when datamart is selected
- Each mapping has database_column.node_id pointing to alpha_dwh column node

✅ **Edge Creation Endpoint**
- Resolves datamart datasource to alpha_dwh ID
- Creates edges using the resolved datasource ID
- Edges link to correct column nodes in catalog

✅ **Business Terms**
- When using the semantic wizard, business term nodes are automatically created
- HAS_BUSINESS_TERM edges link semantic terms to business terms
- Complete semantic chain: Column → Semantic Term → Business Term

## Debugging Commands

**Check datasource mappings:**
```sql
SELECT id, datasource_name FROM alpha_datasource 
WHERE datasource_name IN ('datamart', 'alpha_dwh');
```

**Check if edges were created:**
```sql
SELECT ce.*, 
       cn1.node_name as source_name,
       cn2.node_name as target_name
FROM catalog_edge ce
JOIN catalog_node cn1 ON ce.source_node_id = cn1.id
JOIN catalog_node cn2 ON ce.target_node_id = cn2.id
WHERE ce.relationship_type = 'MAPS_TO'
LIMIT 10;
```

**Check semantic terms:**
```sql
SELECT id, node_name, qualified_path FROM catalog_node 
WHERE node_type_id = '2d5f6e21-7f4c-4c4d-8c0e-0d5e5c5b5a59'  -- Semantic Term node type
LIMIT 10;
```

**Check business terms:**
```sql
SELECT id, node_name, qualified_path FROM catalog_node 
WHERE node_type_id = '8b8d8e8f-9a9b-4c4d-8e8f-0d5e5c5b5a60'  -- Business Term node type
LIMIT 10;
```

## Log Messages to Look For

**Success indicators in logs:**

- `[GenerateMappings] Mapped 'datamart' datasource to 'alpha_dwh' ID: <uuid>`
- `[edges] Resolved 'datamart' datasource to 'alpha_dwh' ID: <uuid>`
- `Created mapping edge: <semantic_term_id> -> <column_id>`
- `Successfully applied enrichment for column`

**Error indicators:**

- `Could not find 'alpha_dwh' datasource ID to map 'datamart'`
- `Failed to create mapping edge: ...`
- `Failed to fetch datasource name for ID`

## Known Limitations

- Datamart column discovery only works if there are columns in alpha_dwh with matching tenant_id
- Business terms are automatically created by the enrichment function, not shown in the UI
- Edges are stored with the alpha_dwh datasource_id in the catalog
