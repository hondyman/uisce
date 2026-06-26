# Semantic Types Lookup Integration Guide

## Overview

This guide explains how to use the new `semantic_types` lookup table to manage semantic type configurations for nodes and edges in your Fabric Builder graph.

## What Was Created

### 1. Database Migration
**File:** `backend/migrations/2025_11_19_create_semantic_types_lookup.sql`

This migration:
- Creates a `semantic_types` lookup entry in the `lookups` table
- Populates 35 semantic type combinations covering:
  - **Dimension** types: 11 variants (string, number, boolean, time, geo with various formats)
  - **Measure** types: 18 variants (simple types and aggregation functions with formats)
  - **Time** type: 1 dedicated semantic time variant
- Stores metadata in JSONB format including semantic_type, data_type, format, and notes
- Creates an index for efficient lookups

## Semantic Types Structure

Each semantic type entry contains:

```json
{
  "semantic_type": "Dimension|Measure|Time",
  "data_type": "string|number|boolean|time|geo|number_agg|count|count_distinct|...",
  "format": "default|imageUrl|link|currency|percent|id",
  "notes": "Descriptive notes about the type"
}
```

## How to Use in Your Application

### 1. API Access - List All Semantic Types

```bash
curl -X GET \
  "http://localhost:8080/api/lookups?tenant_id=<YOUR_TENANT_ID>&q=semantic_types" \
  -H "X-Tenant-ID: <YOUR_TENANT_ID>" \
  -H "X-Tenant-Datasource-ID: <YOUR_DATASOURCE_ID>"
```

Response:
```json
{
  "lookups": [
    {
      "id": "lookup-uuid-here",
      "tenant_id": "tenant-uuid",
      "name": "semantic_types",
      "description": "Semantic types with data types and formats for nodes and edges"
    }
  ]
}
```

### 2. API Access - Get All Semantic Type Values

```bash
curl -X GET \
  "http://localhost:8080/api/lookups/<LOOKUP_ID>/values?tenant_id=<YOUR_TENANT_ID>" \
  -H "X-Tenant-ID: <YOUR_TENANT_ID>" \
  -H "X-Tenant-Datasource-ID: <YOUR_DATASOURCE_ID>"
```

Response example:
```json
{
  "values": [
    {
      "id": "value-uuid",
      "lookup_id": "lookup-uuid",
      "tenant_id": "tenant-uuid",
      "value": "dimension_string_default",
      "label": "Dimension (string, default)",
      "metadata": {
        "semantic_type": "Dimension",
        "data_type": "string",
        "format": "default",
        "notes": ""
      }
    },
    {
      "id": "value-uuid-2",
      "lookup_id": "lookup-uuid",
      "tenant_id": "tenant-uuid",
      "value": "dimension_string_imageurl",
      "label": "Dimension (string, imageUrl)",
      "metadata": {
        "semantic_type": "Dimension",
        "data_type": "string",
        "format": "imageUrl",
        "notes": "Dimension Format"
      }
    }
    // ... 33 more entries
  ]
}
```

### 3. Using in Frontend with Property Lookups

To use semantic_types with nodes and edges, associate the `semantic_types` lookup with properties:

```javascript
// In your property definition for a node or edge:
{
  name: "semantic_type",
  label: "Semantic Type",
  lookup_id: "<semantic_types_lookup_id>",  // Reference the lookup
  data_type: "string"
}
```

Then use `usePropertyLookupMaps` to fetch available options:

```typescript
import { usePropertyLookupMaps } from '../hooks/usePropertyLookupMaps';

function MyComponent({ nodeType }) {
  const lookupMaps = usePropertyLookupMaps(nodeType, assetProperties);
  
  // lookupMaps will contain semantic_type values mapped by ID and label
  // Use in dropdown/select component
}
```

### 4. Direct SQL Queries

Get all semantic types:
```sql
SELECT 
  lv.id,
  lv.value,
  lv.label,
  lv.metadata->>'semantic_type' as semantic_type,
  lv.metadata->>'data_type' as data_type,
  lv.metadata->>'format' as format,
  lv.metadata->>'notes' as notes
FROM lookup_values lv
JOIN lookups l ON lv.lookup_id = l.id
WHERE l.name = 'semantic_types'
  AND lv.tenant_id = $1
ORDER BY 
  lv.metadata->>'semantic_type',
  lv.metadata->>'data_type',
  lv.metadata->>'format';
```

Get specific semantic type category:
```sql
SELECT * FROM lookup_values 
WHERE lookup_id = (SELECT id FROM lookups WHERE name = 'semantic_types' LIMIT 1)
  AND metadata->>'semantic_type' = 'Dimension'
  AND metadata->>'data_type' = 'string'
  AND metadata->>'format' = 'currency';
```

## Semantic Types Reference

### Dimension Types (11 total)

| Data Type | Formats | Notes |
|-----------|---------|-------|
| string | default, imageUrl, link, currency, percent | Visual dimension formats |
| number | default, id, currency, percent | Numeric dimensions |
| boolean | default | Boolean flags |
| time | default | Date/time dimensions |
| geo | default | Geographic dimensions |

### Measure Types (18 total)

| Data Type | Formats | Notes |
|-----------|---------|-------|
| string | default | Text-based measures |
| time | default | Time-based measures |
| boolean | default | Boolean measures |
| number | default, percent, currency | Numeric measures with formatting |
| number_agg | default, percent, currency | Aggregated numeric measures |
| count | default | Row count measure |
| count_distinct | default | Distinct count measure |
| count_distinct_approx | default | Approximate distinct count |
| sum | default, currency | Summation measures |
| avg | default | Average measures |
| min | default | Minimum measures |
| max | default | Maximum measures |

### Time Type (1 total)

| Data Type | Format | Notes |
|-----------|--------|-------|
| time | default | Dedicated semantic time object |

## Applying Semantic Types to Nodes and Edges

### Step 1: Configure Property on Node Type

Register a property with the semantic_types lookup:

```sql
-- In your node_type properties definition
-- This should be in your configuration or schema setup

UPDATE node_types 
SET properties = jsonb_set(
  COALESCE(properties, '{}'::jsonb),
  '{semantic_type}',
  jsonb_build_object(
    'name', 'semantic_type',
    'label', 'Semantic Type',
    'lookup_id', (SELECT id FROM lookups WHERE name = 'semantic_types' LIMIT 1)::text,
    'data_type', 'string'
  )
)
WHERE name = 'dimension';
```

### Step 2: Use in Graph Operations

When creating/updating nodes or edges, set the semantic_type:

```sql
UPDATE catalog_node 
SET properties = jsonb_set(
  COALESCE(properties, '{}'::jsonb),
  '{semantic_type}',
  '"dimension_string_currency"'  -- value from semantic_types lookup
)
WHERE id = '<node_id>';
```

### Step 3: Query by Semantic Type

Find all dimensions with currency formatting:

```sql
SELECT cn.* 
FROM catalog_node cn
WHERE cn.properties->>'semantic_type' = 'dimension_string_currency'
  AND cn.tenant_id = $1
  AND cn.tenant_datasource_id = $2;
```

## Integration with Existing Systems

### With Lookup System

The `semantic_types` lookup integrates seamlessly with your existing lookup system:

- **Hierarchical lookups**: Not needed for semantic_types (flat list of combinations)
- **Table-backed lookups**: Can be implemented later if semantic types need source table backing
- **Cascading properties**: Can cascade from other properties if needed

### With Bundle System

If using bundles with semantic types:

```sql
-- Store semantic type constraint in bundle
UPDATE bundles 
SET definition = jsonb_set(
  definition,
  '{semantic_type_constraint}',
  '"measure_number_currency"'
)
WHERE id = '<bundle_id>';
```

### With Governance/ABAC

Apply semantic type-based policies:

```sql
-- Restrict access to certain semantic types
INSERT INTO policies (
  tenant_id,
  name,
  condition,
  effect
) VALUES (
  $1,
  'Restrict Currency Measures',
  'resource.properties.semantic_type = "measure_number_currency"',
  'DENY'
);
```

## Running the Migration

### Using migrate CLI:

```bash
export DATABASE_URL='postgres://postgres:postgres@host.docker.internal:5432/alpha?sslmode=disable'
migrate -path backend/migrations -database "$DATABASE_URL" up
```

### Using psql directly:

```bash
psql "$DATABASE_URL" -f backend/migrations/2025_11_19_create_semantic_types_lookup.sql
```

### Using Docker:

```bash
docker compose exec backend psql -U user -d db -f /migrations/2025_11_19_create_semantic_types_lookup.sql
```

## Verification

Verify the semantic_types lookup was created:

```bash
export DATABASE_URL='postgres://postgres:postgres@host.docker.internal:5432/alpha?sslmode=disable'

# Check lookup exists
psql "$DATABASE_URL" -c \
  "SELECT id, name FROM lookups WHERE name = 'semantic_types';"

# Count values (should be 35)
psql "$DATABASE_URL" -c \
  "SELECT COUNT(*) FROM lookup_values 
   WHERE lookup_id = (SELECT id FROM lookups WHERE name = 'semantic_types' LIMIT 1);"

# View sample entries
psql "$DATABASE_URL" -c \
  "SELECT value, label, metadata FROM lookup_values 
   WHERE lookup_id = (SELECT id FROM lookups WHERE name = 'semantic_types' LIMIT 1)
   LIMIT 5;"
```

## Complete Semantic Types Value Map

All 35 semantic type combinations:

### Dimensions (11)
1. dimension_string_default
2. dimension_string_imageurl
3. dimension_string_link
4. dimension_string_currency
5. dimension_string_percent
6. dimension_number_default
7. dimension_number_id
8. dimension_number_currency
9. dimension_number_percent
10. dimension_boolean_default
11. dimension_time_default
12. dimension_geo_default

### Measures (18)
13. measure_string_default
14. measure_time_default
15. measure_boolean_default
16. measure_number_default
17. measure_number_percent
18. measure_number_currency
19. measure_number_agg_default
20. measure_number_agg_percent
21. measure_number_agg_currency
22. measure_count_default
23. measure_count_distinct_default
24. measure_count_distinct_approx_default
25. measure_sum_default
26. measure_sum_currency
27. measure_avg_default
28. measure_min_default
29. measure_max_default

### Time (1)
30. time_time_default

## FAQ

**Q: Can I add custom semantic types?**
A: Yes, insert new entries into the `lookup_values` table:
```sql
INSERT INTO lookup_values (
  lookup_id, tenant_id, value, label, metadata
) VALUES (
  (SELECT id FROM lookups WHERE name = 'semantic_types'),
  $1,
  'my_custom_type',
  'My Custom Type',
  '{"semantic_type":"Custom","data_type":"custom","format":"default"}'
);
```

**Q: How do I filter semantic types by data_type?**
A: Use metadata queries:
```sql
SELECT * FROM lookup_values
WHERE lookup_id = (SELECT id FROM lookups WHERE name = 'semantic_types')
  AND metadata->>'data_type' = 'number';
```

**Q: Can semantic types have hierarchies?**
A: Yes, use the optional `parent_id` column if needed, but the current implementation uses a flat structure.

**Q: How do I use semantic types in queries?**
A: Store the semantic type value (e.g., "dimension_string_currency") in node/edge properties and query by it.

## Summary

You now have:

✅ A fully populated `semantic_types` lookup table (35 entries)  
✅ JSONB metadata for each semantic type combination  
✅ Integration ready with existing lookup API (`/api/lookups`)  
✅ Frontend hook support via `usePropertyLookupMaps`  
✅ SQL query examples for common operations  
✅ Ready to apply to nodes and edges in your graph  

The semantic_types lookup is tenant-scoped and integrates seamlessly with your existing Fabric Builder infrastructure.
