# Business Object Foreign Key Semantic Discovery

## Overview

This system automatically discovers semantic terms available on tables related to a Business Object's driving table through foreign key relationships, and allows those terms to be linked to Business Object fields for automated join path generation.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    BUSINESS OBJECT                              │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │ driver_table_id: "orders_table"                            │ │
│  │                                                             │ │
│  │ Fields:                                                     │ │
│  │  - order_id (core)                                         │ │
│  │  - customer (semantic_term FROM customer table via FK) ◄── ┤━━┓
│  │  - product (semantic_term FROM product table via FK)       │ │  │
│  └────────────────────────────────────────────────────────────┘ │  │
└─────────────────────────────────────────────────────────────────┘  │
                              │                                        │
                              ├─ FK edges stored in catalog_edge ─────┤
                              │  (from metadata scanner)              │
        ┌─────────────────────┴──────────────────────┐               │
        │                                             │               │
        ▼                                             ▼               │
┌──────────────────┐                      ┌──────────────────┐       │
│  CUSTOMERS       │                      │   PRODUCTS       │       │
│  TABLE           │                      │   TABLE          │       │
├──────────────────┤                      ├──────────────────┤       │
│ id (PK)          │                      │ id (PK)          │       │
│ name             │ ◄─ semantic terms ── │ name             │       │
│ email            │     available        │ category         │ ◄─────┘
│ country          │     from these       │ sku              │
└──────────────────┘     fields           │ price            │
                                          └──────────────────┘
```

## Data Model

### FK Edge Schema (catalog_edge table)
```json
{
  "id": "uuid",
  "source_node_id": "orders_table_id",
  "target_node_id": "customers_table_id",
  "edge_type_name": "foreign_key",
  "properties": {
    "edge_type_name": "foreign_key",
    "cardinality": "N:1",
    "source_table": "orders",
    "target_table": "customers",
    "source_schema": "public",
    "target_schema": "public",
    "primary_constraint_name": "fk_orders_customer",
    "columns": [
      {
        "source_column": "customer_id",
        "target_column": "id"
      }
    ],
    "on_delete": "CASCADE",
    "on_update": "CASCADE"
  }
}
```

### BO Field with Semantic Link
```json
{
  "id": "uuid",
  "business_object_id": "orders_bo_id",
  "key": "customer",
  "name": "Customer",
  "semantic_term_id": "customer_term_id",
  "fk_edge_id": "fk_edge_uuid",
  "field_type": "related_object",
  "is_core": false,
  "display_order": 2
}
```

## Usage Flows

### 1. Discover Foreign Keys for Business Object

```bash
GET /api/business-objects/{boId}/foreign-keys
Headers:
  X-Tenant-ID: {tenantId}

Response:
{
  "business_object_id": "orders_bo_id",
  "foreign_keys": [
    {
      "edge_id": "fk_edge_1",
      "related_table_id": "customers_table_id",
      "related_table_name": "customers",
      "cardinality": "N:1",
      "direction": "outbound",
      "foreign_key_fields": [
        {
          "source_column": "customer_id",
          "target_column": "id"
        }
      ],
      "properties": { ... }
    },
    {
      "edge_id": "fk_edge_2",
      "related_table_id": "products_table_id",
      "related_table_name": "products",
      "cardinality": "N:1",
      "direction": "outbound",
      "foreign_key_fields": [
        {
          "source_column": "product_id",
          "target_column": "id"
        }
      ],
      "properties": { ... }
    }
  ],
  "count": 2
}
```

### 2. Discover Available Semantic Terms from Related Tables

```bash
GET /api/business-objects/{boId}/related-semantic-terms?limit=50
Headers:
  X-Tenant-ID: {tenantId}

Response:
{
  "business_object_id": "orders_bo_id",
  "related_semantic_terms": [
    {
      "semantic_term_id": "customer_name_term",
      "semantic_term_name": "Customer Name",
      "related_table_name": "customers",
      "related_field_name": "name",
      "related_field_id": "customers.name_column_id",
      "source_fk_edge_id": "fk_edge_1",
      "join_path": "customer_id -> customers.id",
      "confidence": 0.95,
      "match_reason": "semantic_term_mapped_in_catalog"
    },
    {
      "semantic_term_id": "customer_email_term",
      "semantic_term_name": "Customer Email",
      "related_table_name": "customers",
      "related_field_name": "email",
      "related_field_id": "customers.email_column_id",
      "source_fk_edge_id": "fk_edge_1",
      "join_path": "customer_id -> customers.id",
      "confidence": 0.95,
      "match_reason": "semantic_term_mapped_in_catalog"
    }
  ],
  "count": 2,
  "message": "These semantic terms are available from related tables via foreign keys",
  "usage": "Link these to BO fields to enable joining related table data"
}
```

### 3. Link Semantic Term to Business Object

```bash
POST /api/business-objects/{boId}/link-semantic-term
Headers:
  X-Tenant-ID: {tenantId}
  Content-Type: application/json

Body:
{
  "semantic_term_id": "customer_name_term",
  "related_table_id": "customers_table_id",
  "foreign_key_edge_id": "fk_edge_1",
  "role": "customer"
}

Response (201 Created):
{
  "success": true,
  "message": "Semantic term linked successfully",
  "business_object_id": "orders_bo_id",
  "semantic_term_id": "customer_name_term",
  "foreign_key_edge_id": "fk_edge_1"
}

Result in Database:
- bo_fields entry created with:
  - key: "customer_customer"
  - semantic_term_id: "customer_name_term"
  - fk_edge_id: "fk_edge_1"
  - field_type: "related_object"
  - This tells the system: "To get customer_customer data, join via FK edge"
}
```

### 4. Get Join Paths for Query Reconstruction

```bash
GET /api/business-objects/{boId}/semantic-join-paths
Headers:
  X-Tenant-ID: {tenantId}

Response:
{
  "business_object_id": "orders_bo_id",
  "semantic_join_paths": {
    "customer_customer": {
      "semantic_term_id": "customer_name_term",
      "fk_edge_id": "fk_edge_1",
      "related_table": "customers",
      "fk_properties": {
        "columns": [
          {
            "source_column": "customer_id",
            "target_column": "id"
          }
        ],
        "cardinality": "N:1"
      }
    },
    "product_product": {
      "semantic_term_id": "product_name_term",
      "fk_edge_id": "fk_edge_2",
      "related_table": "products",
      "fk_properties": {
        "columns": [
          {
            "source_column": "product_id",
            "target_column": "id"
          }
        ],
        "cardinality": "N:1"
      }
    }
  },
  "count": 2,
  "message": "Use these join paths to construct queries that fetch semantic term data from related tables"
}
```

## Implementation Details

### Metadata Scanner Enhancement

The ANSI Scanner (`internal/scanner/ansi_scanner.go`) now captures enhanced FK information:

**What's Captured:**
1. **Column Mappings** - Exact mapping of source→target columns
2. **Cardinality** - Relationship type (N:1, 1:1, etc.)
3. **Edge Type** - Marked as "foreign_key"
4. **Constraint Metadata** - ON DELETE, ON UPDATE rules
5. **Schemas** - Full qualified table names

**Example FK Edge Created:**
```json
{
  "id": "uk-fk-orders-customers",
  "source_node_id": "orders_table_uuid",
  "target_node_id": "customers_table_uuid",
  "properties": {
    "edge_type_name": "foreign_key",
    "cardinality": "N:1",
    "columns": [
      {
        "source_column": "customer_id",
        "target_column": "id"
      }
    ],
    "source_table": "orders",
    "target_table": "customers",
    "on_delete": "CASCADE",
    "primary_constraint_name": "fk_orders_customer_id"
  }
}
```

### Service Logic (BOSemanticRelationshipsService)

**Key Methods:**

1. **DiscoverForeignKeyRelationshipsForBO()**
   - Finds all FK edges involving BO's driving table
   - Returns both outbound (BO refs other) and inbound (other refs BO)
   - Includes full column mappings

2. **DiscoverSemanticTermsForRelatedTables()**
   - For each related table, queries its columns
   - Finds semantic terms mapped to those columns
   - Returns ranked list of available terms

3. **LinkSemanticTermToBusinessObject()**
   - Creates/updates bo_field entry
   - Stores both semantic_term_id and fk_edge_id
   - Enables query reconstruction with joins

4. **GetBOSemanticJoinPaths()**
   - Returns all currently linked semantic terms
   - Provides join path metadata for query building

### Query Execution Example

When executing a query for an Order BO that has linked Customer semantic terms:

```sql
-- Original query (driving table)
SELECT * FROM orders WHERE id = ?

-- With semantic term expansion (after user selects customer name):
SELECT 
  o.id,
  o.customer_id,
  o.order_date,
  c.name AS customer_name    ← from related table
FROM orders o
LEFT JOIN customers c ON o.customer_id = c.id
WHERE o.id = ?
```

The system knows to add the JOIN because:
1. BO has a bo_field with `semantic_term_id` = "customer_name"
2. That field has `fk_edge_id` pointing to the FK relationship
3. FK edge contains column mappings (customer_id → customers.id)

## Benefits

### 1. Automatic Relationship Discovery
- No manual configuration of related tables
- Self-documenting through catalog metadata
- Scales with schema changes

### 2. Semantic Term Enrichment
- Reuse semantic mappings across tables
- One semantic term can serve multiple BOs
- Terms are business-meaningful, not technical

### 3. Query Optimization
- Know exactly which joins are needed
- Only include relevant semantic terms
- Support for complex multi-table BOs

### 4. Data Governance
- FK relationships enforced at database level
- Catalog tracks all semantic enrichments
- Audit trail of which terms used in BO

## Common Operations

### Add a Related Table's Semantic Terms to Business Object

1. **Discover available FKs:**
   ```bash
   curl -H "X-Tenant-ID: {tenantId}" \
     http://localhost:8080/api/business-objects/{boId}/foreign-keys
   ```

2. **See what semantic terms exist on related tables:**
   ```bash
   curl -H "X-Tenant-ID: {tenantId}" \
     http://localhost:8080/api/business-objects/{boId}/related-semantic-terms
   ```

3. **Link the semantic terms you want:**
   ```bash
   curl -X POST \
     -H "X-Tenant-ID: {tenantId}" \
     -H "Content-Type: application/json" \
     -d '{
       "semantic_term_id": "...",
       "foreign_key_edge_id": "...",
       "role": "customer"
     }' \
     http://localhost:8080/api/business-objects/{boId}/link-semantic-term
   ```

4. **Verify join paths:**
   ```bash
   curl -H "X-Tenant-ID: {tenantId}" \
     http://localhost:8080/api/business-objects/{boId}/semantic-join-paths
   ```

### Update a Business Object's Driving Table

When you change the driving table for a BO:

1. **Old related semantic terms become invalid**
   - FK paths from old table won't exist
   - System returns 404 on those join paths

2. **Run discovery again**
   - New driving table may have different FKs
   - New semantic terms become available
   - Manually re-link as needed

## Database Schema Extensions

### bo_fields Table (Enhancement)

Added columns to capture FK semantic linkage:
```sql
ALTER TABLE bo_fields ADD COLUMN (
  semantic_term_id UUID,     -- Links to semantic term
  fk_edge_id TEXT,           -- Links to FK edge in catalog_edge
  FOREIGN KEY (semantic_term_id) REFERENCES catalog_node(id),
  -- Note: fk_edge_id is TEXT because catalog_edge.id is UUID
);

CREATE INDEX idx_bo_fields_semantic_term ON bo_fields(semantic_term_id);
CREATE INDEX idx_bo_fields_fk_edge ON bo_fields(fk_edge_id);
```

### catalog_edge Table (No changes needed)

Already supports all necessary metadata in JSONB properties:
- `edge_type_name` - identifies as foreign key
- `columns` - stores source/target column mappings
- `cardinality` - relationship type
- Full schema and table names for qualified references

## Error Handling

| Scenario | Error | Resolution |
|----------|-------|------------|
| BO has no driving table | "business object has no driving table" | Set driver_table_id on BO first |
| FK edge not found | "failed to validate foreign key edge" | Run metadata scan to discover FKs |
| Semantic term not mapped | Empty results | Add semantic term mappings in catalog |
| Query execution fails | SQL error on join | Verify FK constraint exists in DB |

## Performance Considerations

### Discovery Queries
- **ForeignKeyRelationshipsForBO:** O(n) where n = number of FKs from driving table
- **SemanticTermsForRelatedTables:** O(n×m) where n = FKs, m = semantic terms per table
- Typically <100ms for typical schema

### Join Path Retrieval
- **GetBOSemanticJoinPaths:** Single query, O(k) where k = linked fields
- Cached at application level recommended

### Recommendations
- Cache discover results for 24 hours
- Batch semantic term links in transactions
- Index bo_fields.(business_object_id, semantic_term_id, fk_edge_id)

## Future Enhancements

1. **Transitive FK Resolution**
   - Support 2+ hop joins (orders → customers → addresses)
   - Complex multi-table BOs

2. **Cardinality Validation**
   - Detect 1:1, N:M relationships
   - Adjust join strategy (INNER vs LEFT)

3. **Circular Reference Detection**
   - Prevent infinite joins
   - Alert when cycles detected

4. **Semantic Term Inheritance**
   - Child semantic terms inherit from parent table terms
   - Automatic term propagation

5. **FK Change Detection**
   - Monitor schema changes
   - Auto-remediate broken links

---

## Related Documentation

- [FK Discovery System](./FK_DISCOVERY_SYSTEM.md)
- [Business Object Implementation](./BUSINESS_OBJECT_IMPLEMENTATION.md)
- [Metadata Scanner Architecture](./METADATA_SCANNER.md)
- [Semantic Term Mapping](./SEMANTIC_TERM_MAPPING.md)
