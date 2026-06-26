# Business Glossary Testing Guide

## Quick Start

### 1. Access the Business Glossary

1. Navigate to the application
2. Ensure you have selected a tenant and datasource using the tenant picker
3. Click on **Core** menu in the top navigation
4. Select **"Business Glossary"**

You should see the Business Glossary page with two tabs:
- **Semantic Terms** (left tab)
- **Business Glossary** (right tab)

### 2. Test Semantic Terms Tab

#### Expected Behavior
- Page loads with a table of semantic terms
- Columns show: Type, Description, Active status, and any custom properties
- Each term has an Edit button

#### Try It
1. Click the **Edit** button (pencil icon) on any semantic term
2. A dialog appears with editable fields
3. Modify the Description field
4. Click **Save**
5. Dialog closes and table updates

#### If It's Empty
Your database may not have semantic terms yet. You can create them by:
```sql
-- Insert a test semantic term
INSERT INTO catalog_node (
    id, 
    tenant_datasource_id, 
    catalog_type_name, 
    description, 
    is_active, 
    properties, 
    tenant_id
) VALUES (
    gen_random_uuid(),
    '982aef38-418f-46dc-acd0-35fe8f3b97b0',
    'semantic_term',
    'Test Semantic Term',
    true,
    '[{"name": "field1", "label": "Field 1", "order": 0, "nullable": true, "data_type": "string", "input_type": "text"}]',
    '910638ba-a459-4a3f-bb2d-78391b0595f6'
);
```

### 3. Test Business Glossary Tab

#### Expected Behavior
- Three-panel layout loads
- Left panel shows Business Terms cards
- Center shows ReactFlow diagram
- Right panel shows Semantic Terms cards
- Animated arrows connect related terms

#### Try It
1. Look for blue nodes (Business Terms) in the center diagram
2. Look for green nodes (Semantic Terms) in the center diagram
3. Blue arrows should connect related terms
4. You can drag nodes around to reposition them

#### If It's Empty
Business terms and edges need to exist. Create them with:

```sql
-- Create a business term
INSERT INTO catalog_node (
    id, 
    tenant_datasource_id, 
    catalog_type_name, 
    description, 
    is_active, 
    properties, 
    tenant_id
) VALUES (
    gen_random_uuid(),
    '982aef38-418f-46dc-acd0-35fe8f3b97b0',
    'business_term',
    'Test Business Term',
    true,
    '[]',
    '910638ba-a459-4a3f-bb2d-78391b0595f6'
) RETURNING id;

-- Get the IDs from both inserts, then create an edge
INSERT INTO catalog_edge (
    id,
    tenant_id,
    subject_node_type_id,
    object_node_type_id,
    predicate,
    description,
    is_active,
    properties
) VALUES (
    gen_random_uuid(),
    '910638ba-a459-4a3f-bb2d-78391b0595f6',
    '<business_term_id>',
    '<semantic_term_id>',
    'has_semantic',
    'Links business term to semantic term',
    true,
    '[]'
);
```

## Testing Individual Endpoints

### Test Semantic Terms Endpoint
```bash
curl -X GET "http://localhost:29080/api/glossary/semantic-terms" \
  -H "X-Tenant-ID: 910638ba-a459-4a3f-bb2d-78391b0595f6" \
  -H "X-Tenant-Datasource-ID: 982aef38-418f-46dc-acd0-35fe8f3b97b0" \
  -H "Content-Type: application/json"
```

**Expected Response:**
```json
[
  {
    "id": "uuid-here",
    "tenant_datasource_id": "982aef38-418f-46dc-acd0-35fe8f3b97b0",
    "catalog_type_name": "semantic_term",
    "description": "Test Semantic Term",
    "is_active": true,
    "parent_type_id": null,
    "config": "",
    "created_at": "2025-10-17T12:00:00Z",
    "updated_at": "2025-10-17T12:00:00Z",
    "tenant_id": "910638ba-a459-4a3f-bb2d-78391b0595f6",
    "core_id": null,
    "properties": [...]
  }
]
```

### Test Business Terms Endpoint
```bash
curl -X GET "http://localhost:29080/api/glossary/business-terms" \
  -H "X-Tenant-ID: 910638ba-a459-4a3f-bb2d-78391b0595f6" \
  -H "X-Tenant-Datasource-ID: 982aef38-418f-46dc-acd0-35fe8f3b97b0" \
  -H "Content-Type: application/json"
```

### Test Edges Endpoint
```bash
curl -X GET "http://localhost:29080/api/glossary/edges" \
  -H "X-Tenant-ID: 910638ba-a459-4a3f-bb2d-78391b0595f6" \
  -H "Content-Type: application/json"
```

### Test Update Term Endpoint
```bash
curl -X PUT "http://localhost:29080/api/glossary/terms/{term_id}" \
  -H "X-Tenant-ID: 910638ba-a459-4a3f-bb2d-78391b0595f6" \
  -H "X-Tenant-Datasource-ID: 982aef38-418f-46dc-acd0-35fe8f3b97b0" \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Updated Description",
    "is_active": true
  }'
```

### Test Create Edge Endpoint
```bash
curl -X POST "http://localhost:29080/api/glossary/edges" \
  -H "X-Tenant-ID: 910638ba-a459-4a3f-bb2d-78391b0595f6" \
  -H "X-Tenant-Datasource-ID: 982aef38-418f-46dc-acd0-35fe8f3b97b0" \
  -H "Content-Type: application/json" \
  -d '{
    "subject_node_id": "<business_term_id>",
    "object_node_id": "<semantic_term_id>",
    "edge_type_id": "<edge_type_id>"
  }'
```

## Troubleshooting

### "Tenant scope not selected" message
- **Solution**: Click the tenant picker at the top of the page and select a tenant, product, and datasource

### Empty tables in both tabs
- **Possible Cause**: No semantic or business terms in database for the selected tenant
- **Solution**: Insert test data using SQL commands provided above

### Diagram shows nodes but no edges
- **Possible Cause**: No edges created between the nodes
- **Solution**: Create edges using the SQL INSERT command or the Create Edge endpoint

### "Failed to fetch" error
- **Possible Cause**: Backend endpoint not responding
- **Solution**: 
  1. Check backend is running: `docker-compose logs app` (or your backend service)
  2. Verify tenant headers are being sent correctly
  3. Check browser console for exact error message

### Properties not showing as columns
- **Possible Cause**: Properties JSON malformed or empty
- **Solution**: Verify properties are valid JSON in the database

## Performance Notes

- **Large datasets**: If you have 1000+ terms, consider adding pagination
- **ReactFlow**: Works well up to ~100 nodes. Beyond that, consider:
  - Filtering by term type
  - Adding a search/filter panel
  - Lazy loading nodes on demand

## Browser Compatibility

- Chrome/Edge: ✅ Full support
- Firefox: ✅ Full support
- Safari: ✅ Full support
- IE11: ❌ Not supported (uses modern ES6+)

## Database Queries for Debugging

### View all semantic terms
```sql
SELECT id, description, catalog_type_name, is_active 
FROM catalog_node 
WHERE catalog_type_name = 'semantic_term' 
AND tenant_id = '910638ba-a459-4a3f-bb2d-78391b0595f6'
LIMIT 10;
```

### View all business terms
```sql
SELECT id, description, catalog_type_name, is_active 
FROM catalog_node 
WHERE catalog_type_name = 'business_term' 
AND tenant_id = '910638ba-a459-4a3f-bb2d-78391b0595f6'
LIMIT 10;
```

### View edges
```sql
SELECT id, predicate, subject_node_type_id, object_node_type_id, is_active
FROM catalog_edge
WHERE tenant_id = '910638ba-a459-4a3f-bb2d-78391b0595f6'
LIMIT 10;
```

### Count terms
```sql
SELECT 
  SUM(CASE WHEN catalog_type_name = 'semantic_term' THEN 1 ELSE 0 END) as semantic_terms,
  SUM(CASE WHEN catalog_type_name = 'business_term' THEN 1 ELSE 0 END) as business_terms
FROM catalog_node
WHERE tenant_id = '910638ba-a459-4a3f-bb2d-78391b0595f6';
```

## Common Issues & Solutions

| Issue | Cause | Solution |
|-------|-------|----------|
| 401 Unauthorized | Missing auth/cookies | Ensure you're logged in and session is active |
| 400 Bad Request | Missing required headers | Add `X-Tenant-ID` and `X-Tenant-Datasource-ID` |
| 404 Not Found | Backend endpoint doesn't exist | Verify routes are registered in api.go |
| Empty page | Tenant scope not set | Select tenant from tenant picker |
| No edges in diagram | No edges created in database | Create edges using SQL or API |
| Edit dialog stuck | Network error | Check browser console for specific error |

## Next Steps

1. ✅ Access Business Glossary page
2. ✅ Create test semantic and business terms
3. ✅ Create edges between terms
4. ✅ Test edit functionality
5. ✅ Test ReactFlow diagram navigation
6. 📊 Build custom queries/dashboards using the data
7. 🚀 Deploy to production

## Support

For issues or questions:
1. Check the browser console for JavaScript errors
2. Check backend logs: `docker-compose logs app`
3. Review database queries using provided SQL snippets
4. Consult `BUSINESS_GLOSSARY_README.md` for implementation details
