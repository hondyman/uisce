# Member Attributes Storage Architecture

## Overview
Now that Business Objects/Entities are separated from Catalog Nodes, member attributes are stored in a dedicated **`bo_fields`** table that links attributes to Business Objects or Subtypes, with optional semantic linkage to the catalog system.

---

## Normalization: From JSONB Fields to Relational Design

### Before (Legacy - Denormalized)
```sql
CREATE TABLE public.business_objects (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    display_name TEXT NOT NULL,
    fields JSONB NOT NULL,  -- ❌ All attributes stored as JSON array
    ...
);

-- Example fields JSONB:
{
  "fields": [
    {"key": "customer_id", "name": "Customer ID", "type": "text", "is_required": true},
    {"key": "email", "name": "Email", "type": "email", "is_required": false},
    {"key": "phone", "name": "Phone", "type": "text"}
  ]
}
```

### After (Normalized - Relational)
```sql
-- BO definition without embedded fields
CREATE TABLE public.business_objects (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    display_name TEXT NOT NULL,
    description TEXT,
    icon TEXT,
    config JSONB,  -- ✅ Only for settings/config, not attribute definitions
    ...
);

-- Attributes stored in separate normalized table
CREATE TABLE public.bo_fields (
    id UUID PRIMARY KEY,
    business_object_id UUID NOT NULL REFERENCES business_objects(id),
    key VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    is_required BOOLEAN DEFAULT FALSE,
    ...
);
```

### Benefits of Normalization ✅

| Aspect | JSONB (Before) | Relational (After) |
|--------|---|---|
| **Querying** | Requires JSONB operators (`->`, `->>`) | Standard SQL queries, indexes |
| **Indexing** | GIN indexes only (slower) | B-tree, partial indexes (faster) |
| **Validation** | Manual in application | Database constraints, FK validation |
| **Type Safety** | String parsing required | Native types |
| **Updates** | Replace entire JSONB blob | Update individual field rows |
| **Joins** | Complex JSONB unpacking | Simple joins to bo_fields |
| **Referential Integrity** | Not enforced | Foreign keys enforce consistency |
| **Performance** | Degrades with large arrays | Consistent, scalable |

### Migration Path (000031)

Run Migration 000031 to:
1. Extract all fields from `business_objects.fields` JSONB
2. Insert each field as a row in `bo_fields` table
3. Drop the `fields` column from `business_objects`

```sql
-- After migration, queries become simpler:

-- OLD (JSONB parsing):
SELECT bo.name, field->>'name', field->>'type'
FROM business_objects bo,
     jsonb_array_elements(bo.fields) as field;

-- NEW (Clean SQL):
SELECT bo.name, bf.name, bf.type
FROM business_objects bo
JOIN bo_fields bf ON bf.business_object_id = bo.id
ORDER BY bf.sequence;
```

---

### 1. **`business_objects`** (Parent Table)
Stores the master definitions of Business Objects (entities).

| Column | Type | Purpose |
|--------|------|---------|
| `id` | UUID | Primary key |
| `tenant_id` | UUID | Tenant scope |
| `name` | TEXT | Business object name (e.g., "Customer", "Order") |
| `display_name` | TEXT | User-friendly display name |
| `description` | TEXT | Business object description |
| `icon` | TEXT | Icon identifier for UI |
| `config` | JSONB | Configuration settings (future extensibility) |
| `is_system` | BOOLEAN | System vs custom BO |
| `created_at` | TIMESTAMPTZ | Audit timestamp |
| `updated_at` | TIMESTAMPTZ | Audit timestamp |

**Unique Constraint:** `(name, tenant_id)` — name must be unique per tenant

**⚠️ Note:** The `fields` JSONB column has been normalized into the `bo_fields` table (Migration 000031)

---

### 2. **`bo_subtypes`** (Optional Hierarchy)
Stores subtypes (specializations) of Business Objects.

| Column | Type | Purpose |
|--------|------|---------|
| `id` | UUID | Primary key |
| `business_object_id` | UUID | FK to parent BO |
| `tenant_id` | UUID | Tenant scope |
| `key` | VARCHAR(255) | Subtype identifier |
| `name` | VARCHAR(255) | Subtype display name |

**Unique Constraint:** `(tenant_id, business_object_id, key)`

---

### 3. **`bo_fields`** ⭐ (Member Attributes Storage)
**This is where member attributes are stored** - one row per field/attribute.

| Column | Type | Purpose |
|--------|------|---------|
| `id` | UUID | Primary key |
| `tenant_id` | UUID | Tenant scope |
| `business_object_id` | UUID | FK to `business_objects` (NULL if field belongs to subtype) |
| `subtype_id` | UUID | FK to `bo_subtypes` (NULL if field belongs to BO directly) |
| `key` | VARCHAR(255) | Field identifier (e.g., "customer_id", "email") |
| `name` | VARCHAR(255) | Display name |
| `technical_name` | VARCHAR(255) | Technical name for mapping |
| `type` | VARCHAR(50) | Field type (text, number, date, datetime, boolean, currency, json, array, image, reference) |
| `is_core` | BOOLEAN | Core or custom field |
| `is_required` | BOOLEAN | Required or optional |
| `is_system` | BOOLEAN | Cannot be deleted by user |
| `description` | TEXT | Field documentation |
| `reference_entity` | VARCHAR(255) | If type='reference', what entity it references |
| `sequence` | INTEGER | Display order |
| `created_at` | TIMESTAMPTZ | Audit timestamp |
| `created_by` | UUID | Audit user |
| `last_modified_at` | TIMESTAMPTZ | Audit timestamp |
| `last_modified_by` | UUID | Audit user |

**Constraints:**
- PK: `(id)`
- FK to `business_objects(id)` 
- FK to `bo_subtypes(id)`
- **CHECK:** `(business_object_id IS NOT NULL AND subtype_id IS NULL) OR (business_object_id IS NULL AND subtype_id IS NOT NULL)` — field must belong to either BO or subtype, not both

**Indexes:**
- `bo_fields_bo_idx` on `business_object_id`
- `bo_fields_subtype_idx` on `subtype_id`
- `bo_fields_tenant_idx` on `tenant_id`
- `bo_fields_key_idx` on `key`

---

### 4. **`bo_instances`** (Member Data - Attribute Values)
Stores individual records (instances) of Business Objects with their attribute values.

| Column | Type | Purpose |
|--------|------|---------|
| `id` | UUID | Primary key |
| `tenant_id` | UUID | Tenant scope |
| `datasource_id` | UUID | Datasource scope |
| `business_object_id` | UUID | FK to `business_objects` |
| `subtype_id` | UUID | FK to `bo_subtypes` (if instance is a subtype) |
| `core_field_values` | JSONB | Values for core fields (keyed by field.key) |
| `custom_field_values` | JSONB | Values for custom fields (keyed by field.key) |
| `is_deleted` | BOOLEAN | Soft delete flag |
| Audit columns | - | created_at, created_by, last_modified_at, last_modified_by, deleted_at |

**Example `core_field_values` JSONB:**
```json
{
  "customer_id": "CUST-12345",
  "email": "john@example.com",
  "created_date": "2024-01-15"
}
```

---

## Semantic Linkage to Catalog System

The `bo_fields` table does **NOT** have a direct foreign key to `catalog_node`, but semantic linkage can be established through:

### **`entity_attribute`** Table (from Migration 000030)
Maps business entity attributes to semantic terms in the catalog.

| Column | Type | Purpose |
|--------|------|---------|
| `id` | UUID | Primary key |
| `tenant_id` | UUID | Tenant scope |
| `tenant_datasource_id` | UUID | Datasource scope |
| `parent_id` | UUID | Self-reference for hierarchy (entity → subentity) |
| `catalog_node_id` | UUID | FK to `catalog_node` (semantic term) |
| `entity_key` | TEXT | Reference key (e.g., "Customer") |
| `name` | TEXT | Entity name |
| `is_core` | BOOLEAN | Core entity |
| `business_name` | TEXT | Business-friendly name |
| `technical_name` | TEXT | Technical name |

**Unique Constraint:** `(tenant_datasource_id, entity_key)`

### **`entity_attribute_column_mapping`** Table (from Migration 006)
Maps attributes to physical database columns with semantic context.

| Column | Type | Purpose |
|--------|------|---------|
| `id` | UUID | Primary key |
| `tenant_datasource_id` | UUID | Datasource scope |
| `entity_attribute_id` | UUID | FK to `entity_attribute` |
| `table_name` | VARCHAR | Physical table name |
| `column_name` | VARCHAR | Physical column name |
| `semantic_term_id` | UUID | Optional FK to `catalog_node` |
| `metadata_column_id` | UUID | Reference to metadata column |
| `confidence` | NUMERIC(3,2) | Mapping confidence (0.0-1.0) |
| `is_primary_key` | BOOLEAN | Is PK in source table |
| `is_foreign_key` | BOOLEAN | Is FK in source table |

---

## Storage Architecture Summary

```
BUSINESS OBJECT HIERARCHY:
┌──────────────────────────────────┐
│    business_objects              │ (BO definitions)
│  (Customer, Order, Product)      │
└────────┬─────────────────────────┘
         │
         ├─── bo_subtypes (optional)
         │    (VIP Customer, Standard Customer)
         │
         └─── bo_fields ⭐ (MEMBER ATTRIBUTES)
              (customer_id, email, phone, address, etc.)
              For each attribute:
              - Store field definition
              - Link to semantic term (catalog_node) via
                entity_attribute_column_mapping

INSTANCE DATA:
┌──────────────────────────────────┐
│    bo_instances                  │ (Individual records)
│  (CUST-001, CUST-002, etc.)     │
│  - core_field_values: JSONB      │ (values keyed by field.key)
│  - custom_field_values: JSONB    │
└──────────────────────────────────┘
```

---

## Access Patterns

### Get all member attributes for a Business Object:
```sql
SELECT bf.* FROM bo_fields bf
WHERE bf.business_object_id = $1
  AND bf.tenant_id = $2
ORDER BY bf.sequence;
```

### Get all member attributes for a Subtype:
```sql
SELECT bf.* FROM bo_fields bf
WHERE bf.subtype_id = $1
  AND bf.tenant_id = $2
ORDER BY bf.sequence;
```

### Get a member attribute with semantic context:
```sql
SELECT 
  bf.*,
  cn.id as semantic_term_id,
  cn.node_name as semantic_term_name,
  eacm.table_name,
  eacm.column_name
FROM bo_fields bf
LEFT JOIN entity_attribute_column_mapping eacm 
  ON eacm.entity_attribute_id = bf.id (if linked)
LEFT JOIN catalog_node cn 
  ON cn.id = eacm.semantic_term_id
WHERE bf.business_object_id = $1
  AND bf.tenant_id = $2;
```

### Get instance values with field metadata:
```sql
SELECT 
  bi.id,
  bi.core_field_values,
  bi.custom_field_values,
  bf.key,
  bf.name,
  bf.type
FROM bo_instances bi
JOIN business_objects bo ON bo.id = bi.business_object_id
LEFT JOIN bo_fields bf ON bf.business_object_id = bo.id
WHERE bi.business_object_id = $1
  AND bi.tenant_id = $2;
```

---

## Key Design Decisions

### ✅ Separation of Concerns
- **`business_objects` & `bo_fields`**: Store BO definitions and attribute schemas
- **`bo_instances`**: Store actual data values
- **`catalog_node`**: Store semantic/business definitions (independent)
- **`entity_attribute_column_mapping`**: Link BOs to semantic layer and physical columns

### ✅ Flexibility
- Fields stored in `bo_fields` table (not embedded JSON)
- Instance values in JSONB `core_field_values` and `custom_field_values`
- Allows easy addition of new attributes without schema migration

### ✅ Semantic Integration
- BOs can optionally link to catalog nodes via `entity_attribute` table
- No hard dependency — BOs work standalone or integrated

### ✅ Multi-tenancy & Scoping
- Every table has `tenant_id`
- Instance data also scoped by `datasource_id`
- Ensures complete data isolation

---

## Example: Customer Business Object

```sql
-- 1. Define the Customer BO
INSERT INTO business_objects (tenant_id, key, name, display_name, is_core)
VALUES ('tenant-001', 'customer', 'Customer', 'Customer', false);

-- 2. Define member attributes (fields)
INSERT INTO bo_fields (tenant_id, business_object_id, key, name, type, is_core, is_required)
SELECT bo.id, 'customer_id', 'Customer ID', 'text', true, true
FROM business_objects bo
WHERE bo.key = 'customer' AND bo.tenant_id = 'tenant-001';

INSERT INTO bo_fields (tenant_id, business_object_id, key, name, type, is_core)
SELECT bo.id, 'email', 'Email', 'email', false, false
FROM business_objects bo
WHERE bo.key = 'customer' AND bo.tenant_id = 'tenant-001';

-- 3. Create instances with member attribute values
INSERT INTO bo_instances (tenant_id, datasource_id, business_object_id, core_field_values)
SELECT 
  'tenant-001', 
  'ds-001', 
  bo.id,
  '{"customer_id": "CUST-001", "email": "john@example.com"}'::jsonb
FROM business_objects bo
WHERE bo.key = 'customer' AND bo.tenant_id = 'tenant-001';
```

---

## Related Concepts

- **`entity_attribute`**: Links BO definitions to semantic terms (catalog nodes)
- **`catalog_node`**: Independent semantic/business definitions
- **`entity_attribute_column_mapping`**: Maps BO attributes to physical database columns
- **Subtypes**: Use `bo_subtypes` + subtype-specific fields in `bo_fields` (via `subtype_id`)

