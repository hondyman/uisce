# Entity Schema Restructuring: From JSON Blob to Robust Individual Rows

## Overview

The entity schema has been restructured to replace the monolithic JSON blob storage (`entity_schema` table) with a robust, normalized table (`entity_attribute`) that stores each entity as its own row with:

- **Individual rows per entity** - Each entity, parent or subtype, gets its own row
- **Parent-child relationships** - Subtypes linked via `parent_id` self-reference
- **Semantic term linking** - `catalog_node_id` FK to `catalog_node` table (immutable semantic definitions)
- **Proper constraints** - Uniqueness, referential integrity, and cascade rules

## Problem Statement

### Old Approach (entity_schema)
```
Single row per datasource storing entire entity hierarchy as JSON:
{
  "order": {
    "key": "order",
    "name": "Order",
    "isCore": true,
    "subtypes": {
      "rush_order": {...},
      "standard_order": {...}
    }
  }
}
```

**Issues:**
- ❌ Entire entity tree in one JSONB blob - no indexing per entity
- ❌ String-based "name" references that can become stale
- ❌ No FK to semantic catalog - breaks when entities change names
- ❌ Difficult to query individual entities or subtypes
- ❌ Revalidation needed after any change
- ❌ No audit trail per entity

### New Approach (entity_attribute)
```
One row per entity with proper relationships:

id                | tenant_id | datasource_id | parent_id | catalog_node_id | entity_key | name
uuid              | uuid      | uuid          | NULL      | uuid            | order      | Order
uuid              | uuid      | uuid          | order_id  | uuid            | rush_order | Rush Order
uuid              | uuid      | uuid          | order_id  | uuid            | std_order  | Standard Order
```

**Benefits:**
- ✅ Each entity independently queryable and indexable
- ✅ Strong FK to catalog_node (semantic terms are immutable)
- ✅ Parent-child hierarchy via self-reference `parent_id`
- ✅ Proper timestamps (`created_at`, `updated_at`) per entity
- ✅ Clear constraints preventing data corruption
- ✅ Full audit trail per entity change

---

## Database Schema (DDL)

### New Table: `entity_attribute`

```sql
CREATE TABLE public.entity_attribute (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    tenant_datasource_id uuid NOT NULL,
    parent_id uuid,
    -- Link to the semantic term (catalog_node) - NOT a string name that can change
    catalog_node_id uuid,
    entity_key text NOT NULL,
    name text NOT NULL,
    is_core boolean DEFAULT false NOT NULL,
    business_name text,
    technical_name text,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    CONSTRAINT entity_attribute_pk PRIMARY KEY (id),
    CONSTRAINT entity_attribute_parent_fk FOREIGN KEY (parent_id) 
        REFERENCES public.entity_attribute(id) 
        ON DELETE CASCADE 
        ON UPDATE CASCADE 
        DEFERRABLE INITIALLY DEFERRED,
    CONSTRAINT entity_attribute_catalog_node_fk FOREIGN KEY (catalog_node_id) 
        REFERENCES public.catalog_node(id) 
        ON DELETE SET NULL 
        ON UPDATE CASCADE 
        DEFERRABLE INITIALLY DEFERRED,
    CONSTRAINT entity_attribute_tenant_fk FOREIGN KEY (tenant_id) 
        REFERENCES public.tenants(id) 
        ON DELETE CASCADE 
        ON UPDATE CASCADE 
        DEFERRABLE INITIALLY DEFERRED,
    CONSTRAINT entity_attribute_tenant_datasource_fk FOREIGN KEY (tenant_datasource_id) 
        REFERENCES public.tenant_product_datasource(id) 
        ON DELETE CASCADE 
        ON UPDATE CASCADE 
        DEFERRABLE INITIALLY DEFERRED,
    -- Ensure entity_key uniqueness within a datasource
    CONSTRAINT entity_attribute_key_datasource_unique UNIQUE (tenant_datasource_id, entity_key),
    -- Prevent self-parent-reference
    CONSTRAINT entity_attribute_no_self_parent CHECK (id != parent_id)
);
```

### Indexes

```sql
CREATE INDEX entity_attribute_tenant_datasource_idx 
    ON public.entity_attribute USING btree (tenant_id, tenant_datasource_id);

CREATE INDEX entity_attribute_parent_id_idx 
    ON public.entity_attribute USING btree (parent_id);

CREATE INDEX entity_attribute_catalog_node_id_idx 
    ON public.entity_attribute USING btree (catalog_node_id);

CREATE INDEX entity_attribute_entity_key_idx 
    ON public.entity_attribute USING btree (tenant_datasource_id, entity_key);
```

### Backward-Compatibility View

```sql
CREATE OR REPLACE VIEW public.entity_attribute_hierarchy AS
    SELECT 
        ea.id,
        ea.tenant_id,
        ea.tenant_datasource_id,
        ea.parent_id,
        ea.catalog_node_id,
        ea.entity_key,
        ea.name,
        ea.is_core,
        ea.business_name,
        ea.technical_name,
        CASE WHEN ea.parent_id IS NULL THEN 'root' ELSE 'child' END AS hierarchy_level,
        (SELECT COUNT(*) FROM public.entity_attribute WHERE parent_id = ea.id) AS child_count,
        ea.created_at,
        ea.updated_at
    FROM public.entity_attribute ea;
```

---

## Go Code Changes

### BusinessEntity Struct

```go
type BusinessEntity struct {
    ID                 string         `db:"id"`
    TenantID           string         `db:"tenant_id"`
    TenantDatasourceID string         `db:"tenant_datasource_id"`
    ParentID           sql.NullString `db:"parent_id"`
    CatalogNodeID      sql.NullString `db:"catalog_node_id"`  // NEW: links to semantic term
    Key                string         `db:"entity_key"`
    Name               string         `db:"name"`
    IsCore             bool           `db:"is_core"`
    BusinessName       sql.NullString `db:"business_name"`
    TechnicalName      sql.NullString `db:"technical_name"`
}
```

### Query Updates

**Get all entities for a datasource:**
```go
query := `
    SELECT id, tenant_id, tenant_datasource_id, parent_id, catalog_node_id, 
           entity_key, name, is_core, business_name, technical_name
    FROM public.entity_attribute
    WHERE tenant_id = $1 AND tenant_datasource_id = $2
    ORDER BY entity_key
`
```

**Get root entities (parents only):**
```go
query := `
    SELECT * FROM public.entity_attribute
    WHERE tenant_id = $1 AND tenant_datasource_id = $2 AND parent_id IS NULL
`
```

**Get subtypes of a parent:**
```go
query := `
    SELECT * FROM public.entity_attribute
    WHERE parent_id = $1
    ORDER BY entity_key
`
```

**Get entity by semantic term:**
```go
query := `
    SELECT * FROM public.entity_attribute
    WHERE catalog_node_id = $1 AND tenant_datasource_id = $2
`
```

### Insertion Example

```go
func (s *Server) insertEntity(ctx context.Context, tx *sql.Tx, 
    tenantID, tenantDatasourceID, key string, 
    data map[string]interface{}, parentID sql.NullString) error {
    
    name := data["name"].(string)
    isCore := data["isCore"].(bool)
    
    var businessName, technicalName, catalogNodeID sql.NullString
    
    if bn, ok := data["businessName"].(string); ok {
        businessName.String = bn
        businessName.Valid = true
    }
    if tn, ok := data["technicalName"].(string); ok {
        technicalName.String = tn
        technicalName.Valid = true
    }
    if cni, ok := data["catalogNodeId"].(string); ok {
        catalogNodeID.String = cni
        catalogNodeID.Valid = true
    }

    var newID string
    err := tx.QueryRowContext(ctx, `
        INSERT INTO public.entity_attribute 
        (tenant_id, tenant_datasource_id, parent_id, catalog_node_id, 
         entity_key, name, is_core, business_name, technical_name)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
        RETURNING id
    `, tenantID, tenantDatasourceID, parentID, catalogNodeID, 
       key, name, isCore, businessName, technicalName).Scan(&newID)

    if err != nil {
        return err
    }

    // Recursively insert subtypes
    if subtypes, ok := data["subtypes"].(map[string]interface{}); ok {
        for subKey, subEntityData := range subtypes {
            subEntityMap := subEntityData.(map[string]interface{})
            if err := s.insertEntity(ctx, tx, tenantID, tenantDatasourceID, 
                subKey, subEntityMap, 
                sql.NullString{String: newID, Valid: true}); err != nil {
                return err
            }
        }
    }
    return nil
}
```

---

## Migration Guide

### Step 1: Run Migration

The migration file `000030_restructure_entity_schema_robust.sql` will:
1. Backup old `entity_schema` data (if needed)
2. Drop the old `entity_schema` table
3. Create the new `entity_attribute` table with all constraints
4. Create supporting indexes
5. Create the backward-compatibility view

```bash
# Run migrations (tool-dependent, e.g., golang-migrate)
migrate -path backend/migrations -database "postgres://..." up
```

### Step 2: Update Application Code

The Go backend changes are already in place in `/backend/internal/api/api.go`:
- `getBusinessEntities()` - queries the new table
- `saveBusinessEntities()` - inserts into the new table
- `insertEntity()` - handles recursion with catalog_node_id support

### Step 3: Data Migration (Optional)

If you have legacy data in `entity_schema`, create a migration script:

```go
// Example migration function
func migrateEntitySchemaData(db *sql.DB) error {
    // Query old entity_schema JSONB
    rows, err := db.Query("SELECT tenant_id, tenant_datasource_id, schema_data FROM entity_schema")
    if err != nil {
        return err
    }
    defer rows.Close()

    for rows.Next() {
        var tenantID, datasourceID string
        var schemaData []byte
        
        if err := rows.Scan(&tenantID, &datasourceID, &schemaData); err != nil {
            return err
        }

        var entities map[string]interface{}
        if err := json.Unmarshal(schemaData, &entities); err != nil {
            return err
        }

        // Insert flattened entities into new table
        tx, _ := db.Begin()
        for key, entityData := range entities {
            entityMap := entityData.(map[string]interface{})
            // Use insertEntity logic to insert recursively
        }
        tx.Commit()
    }
    return nil
}
```

### Step 4: Test Endpoints

**Get all entities:**
```bash
curl -H "X-Tenant-ID: <tenant-id>" \
     -H "X-Tenant-Datasource-ID: <datasource-id>" \
     http://localhost:8080/api/business-entities
```

**Expected response (hierarchical):**
```json
{
  "order": {
    "key": "order",
    "name": "Order",
    "isCore": true,
    "businessName": "Customer Order",
    "subtypes": {
      "rush_order": {
        "key": "rush_order",
        "name": "Rush Order",
        "isCore": false
      },
      "standard_order": {
        "key": "standard_order",
        "name": "Standard Order",
        "isCore": false
      }
    }
  }
}
```

**Save entities:**
```bash
curl -X POST \
     -H "X-Tenant-ID: <tenant-id>" \
     -H "X-Tenant-Datasource-ID: <datasource-id>" \
     -H "Content-Type: application/json" \
     -d '{
       "order": {
         "name": "Order",
         "isCore": true,
         "businessName": "Customer Order",
         "catalogNodeId": "<semantic-term-uuid>",
         "subtypes": {
           "rush_order": {
             "name": "Rush Order",
             "isCore": false,
             "catalogNodeId": "<semantic-term-uuid>"
           }
         }
       }
     }' \
     http://localhost:8080/api/business-entities
```

---

## Querying the New Schema

### Get Root Entities (Parents)
```sql
SELECT * FROM public.entity_attribute
WHERE tenant_id = 'abc-123' 
  AND tenant_datasource_id = 'def-456'
  AND parent_id IS NULL;
```

### Get Subtypes of a Parent
```sql
SELECT * FROM public.entity_attribute
WHERE parent_id = 'parent-entity-uuid'
ORDER BY entity_key;
```

### Find Entity by Semantic Term
```sql
SELECT * FROM public.entity_attribute
WHERE catalog_node_id = 'semantic-term-uuid'
  AND tenant_datasource_id = 'def-456';
```

### Get Complete Hierarchy
```sql
WITH RECURSIVE hierarchy AS (
    SELECT id, parent_id, entity_key, name, 0 as depth
    FROM public.entity_attribute
    WHERE parent_id IS NULL AND tenant_datasource_id = 'def-456'
    
    UNION ALL
    
    SELECT ea.id, ea.parent_id, ea.entity_key, ea.name, h.depth + 1
    FROM public.entity_attribute ea
    JOIN hierarchy h ON ea.parent_id = h.id
)
SELECT * FROM hierarchy ORDER BY depth, entity_key;
```

---

## Benefits Summary

| Aspect | Old (entity_schema) | New (entity_attribute) |
|--------|---------------------|----------------------|
| **Storage** | Single JSON blob | Individual rows per entity |
| **Indexing** | No entity-level indexes | Full indexing support |
| **References** | String names (stale) | UUID to catalog_node (immutable) |
| **Queries** | Must deserialize entire JSON | Direct SQL queries per entity |
| **Hierarchy** | Nested JSON | parent_id self-reference |
| **Constraints** | None | PK, FK, UNIQUE, CHECK |
| **Audit Trail** | No timestamps | created_at, updated_at per row |
| **Scalability** | Poor with many entities | Excellent with proper indexing |
| **Data Integrity** | Manual validation | DB-enforced constraints |

---

## Rollback (if needed)

To revert to the old schema:

1. Drop the new table:
   ```sql
   DROP TABLE IF EXISTS public.entity_attribute CASCADE;
   ```

2. Restore from backup or recreate `entity_schema`:
   ```sql
   CREATE TABLE public.entity_schema (
       id uuid DEFAULT gen_random_uuid() NOT NULL,
       tenant_id uuid NOT NULL,
       tenant_datasource_id uuid NOT NULL,
       schema_data jsonb NOT NULL,
       created_at timestamptz DEFAULT now() NULL,
       updated_at timestamptz DEFAULT now() NULL,
       CONSTRAINT entity_schema_pk PRIMARY KEY (id),
       CONSTRAINT entity_schema_unique UNIQUE (tenant_id, tenant_datasource_id),
       CONSTRAINT entity_schema_tenant_datasource_fk FOREIGN KEY (tenant_datasource_id) 
           REFERENCES public.tenant_product_datasource(id) ON DELETE CASCADE,
       CONSTRAINT entity_schema_tenant_fk FOREIGN KEY (tenant_id) 
           REFERENCES public.tenants(id) ON DELETE CASCADE
   );
   ```

3. Revert Go code changes to use `business_entity` or old table names.

---

## Files Changed

1. **Migration**: `/backend/migrations/000030_restructure_entity_schema_robust.sql`
   - Creates `entity_attribute` table
   - Creates indexes and backward-compatibility view
   - Drops old `entity_schema` table

2. **Go Code**: `/backend/internal/api/api.go`
   - `BusinessEntity` struct: Added `CatalogNodeID` field
   - `getBusinessEntities()`: Updated query to use `entity_attribute`
   - `saveBusinessEntities()`: Updated to use new table
   - `insertEntity()`: Added `catalogNodeID` parameter and FK support

---

## Rollout Checklist

- [ ] Run migration in development
- [ ] Test GET /api/business-entities endpoint
- [ ] Test POST /api/business-entities with sample data
- [ ] Verify hierarchy reconstruction works
- [ ] Test with catalog_node_id references
- [ ] Update frontend to send catalogNodeId in POST payloads
- [ ] Run data migration (if legacy data exists)
- [ ] Monitor logs for any schema-related errors
- [ ] Deploy to staging
- [ ] Deploy to production
