# Entity Storage Architecture - Correct Model

**Date:** November 7, 2025  
**Clarification:** You're right! This document explains the **correct separation of concerns**.

---

## 🏗️ Three-Layer Architecture (CORRECT MODEL)

Your system has a clear separation, but I had it backwards. Here's the **actual** model:

### Layer 1: **Entity Content** (The actual entities)
**Location:** `entity_attribute` table  
**Purpose:** Stores the actual entity definitions and hierarchy  
**Contents:**
- Entity types (e.g., "Customer", "Employee", "Product", "Order")
- Entity hierarchy (parent-child relationships, subtypes)
- Entity metadata (entity_key, name, business_name, technical_name)
- Tenant and datasource scope

```sql
-- Example from entity_attribute (THE ACTUAL ENTITIES)
SELECT * FROM entity_attribute WHERE entity_key = 'customer';
-- Result:
-- id: uuid-456
-- entity_key: "customer"
-- name: "Customer"
-- business_name: "Customer Entity"
-- technical_name: "customer_type"
-- parent_id: NULL (root entity)
-- catalog_node_id: uuid-123 (reference to semantic metadata)
```

### Layer 2: **Semantic Catalog** (Metadata DESCRIBING entities)
**Location:** `catalog_node` table  
**Purpose:** Describes what each entity **MEANS** from a semantic/business perspective  
**Contents:**
- Display names and descriptions
- Semantic versioning
- Business meaning and context
- Classification and categorization

```sql
-- Example from catalog_node (DESCRIBES WHAT CUSTOMER IS)
SELECT * FROM catalog_node WHERE name = 'customer';
-- Result:
-- id: uuid-123
-- name: "customer"
-- display_name: "Customer"
-- description: "External party who purchases products"
-- type: "BusinessEntity"
-- version: 1
-- is_active: true
```

---

## 📊 Correct Relationship

```
┌──────────────────────────────────────────────────────────────────────┐
│                                                                      │
│  ENTITY CONTENT LAYER (entity_attribute table)                      │
│  ┌──────────────────────────────────────────────────────┐           │
│  │ Actual Entity: "Customer"                            │           │
│  ├──────────────────────────────────────────────────────┤           │
│  │ id: 550e8400-e29b-41d4-a716-446655440000            │           │
│  │ entity_key: "customer"                               │           │
│  │ name: "Customer"                                     │           │
│  │ business_name: "Customer Entity"                     │           │
│  │ parent_id: NULL (root, no parent)                    │           │
│  │ catalog_node_id: uuid-123  ─────────┐               │           │
│  │ is_core: true                       │               │           │
│  │ tenant_id: tenant-001               │ references    │           │
│  └──────────────────────────────────────┤               │           │
│                                          │               │           │
│                                          ▼               │           │
│  SEMANTIC CATALOG LAYER (catalog_node table)            │           │
│  ┌──────────────────────────────────────────────────────┐           │
│  │ Semantic Descriptor: "Customer"                      │           │
│  ├──────────────────────────────────────────────────────┤           │
│  │ id: uuid-123                                         │           │
│  │ name: "customer"                                     │           │
│  │ display_name: "Customer"                             │           │
│  │ description: "External party who purchases"          │           │
│  │ type: "BusinessEntity"                               │           │
│  │ version: 1                                           │           │
│  │ is_active: true                                      │           │
│  └──────────────────────────────────────────────────────┘           │
│                                                                      │
│  KEY POINT:                                                         │
│  • entity_attribute = WHAT EXISTS (the actual entity)              │
│  • catalog_node = WHAT IT MEANS (semantic metadata)               │
│  • Entity points to catalog for its meaning                        │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
```

---

## 🎯 Concrete Example: Customer Entity

### In `entity_attribute` (THE CONTENT):
```sql
INSERT INTO entity_attribute (
    tenant_id,
    tenant_datasource_id,
    entity_key,
    name,
    business_name,
    technical_name,
    is_core,
    parent_id,
    catalog_node_id
) VALUES (
    'tenant-001',
    'datasource-001',
    'customer',                          -- What we call it internally
    'Customer',                          -- Display name
    'Customer',                          -- Business name
    'customer_type',                     -- Technical name
    true,                                -- Core entity
    NULL,                                -- No parent (it's a root entity)
    'uuid-123'                           -- References its semantic meaning
);

-- Result: 1 row inserted
-- This entity now EXISTS in the system
```

### In `catalog_node` (THE METADATA ABOUT IT):
```sql
INSERT INTO catalog_node (
    name,
    display_name,
    description,
    type,
    version,
    is_active
) VALUES (
    'customer',
    'Customer',
    'External party who purchases products or services from the organization',
    'BusinessEntity',
    1,
    true
);

-- Result: 1 row inserted
-- This describes WHAT "customer" means semantically
```

### Relationship:
```
entity_attribute row (Customer entity) 
  └─ catalog_node_id points to ─→ catalog_node row (Customer description)

The entity OWNS the definition, catalog_node DESCRIBES the entity.
```

---

## 📍 Current Table Locations

| What | Table | Contains | Example |
|------|-------|----------|---------|
| **Entity Definitions** | `entity_attribute` | The actual entities (Customer, Order, Product, etc.) | entity_key: 'customer' |
| **Hierarchy** | `entity_attribute.parent_id` | Parent-child relationships (Order → RushOrder) | parent_id: <order_id> |
| **Semantic Metadata** | `catalog_node` | Meaning and context of entities | description: "External party..." |
| **Entity-to-Metadata Link** | `entity_attribute.catalog_node_id` | FK connecting entity to its semantic definition | catalog_node_id: <uuid> |

---

## 🔗 Query Pattern: Get Entity with Its Semantic Meaning

```sql
-- Get the Customer entity WITH its semantic description
SELECT 
    ea.id as entity_id,
    ea.entity_key,
    ea.name,
    ea.business_name,
    
    -- Semantic metadata
    cn.id as semantic_id,
    cn.display_name,
    cn.description,
    cn.type as semantic_type,
    cn.version
FROM entity_attribute ea
LEFT JOIN catalog_node cn ON ea.catalog_node_id = cn.id
WHERE ea.entity_key = 'customer'
  AND ea.tenant_id = 'tenant-001'
  AND ea.tenant_datasource_id = 'datasource-001';

-- Result:
-- entity_id        | entity_key | name     | business_name | semantic_id | display_name | description               | semantic_type | version
-- 550e8400-...     | customer   | Customer | Customer      | uuid-123    | Customer     | External party who...    | BusinessEntity | 1
```

---

## 🏛️ Why This Separation?

### `entity_attribute` = CONTENT (What you have)
- **Why separate table:** Entity definitions are the core business objects
- **Mutable:** Can add new entities, update names, change hierarchy
- **Scoped:** Tenant/datasource specific
- **References:** Points outward to semantic catalog
- **Example:** "We have a Customer entity with subtypes"

### `catalog_node` = CATALOG (What it means)
- **Why separate table:** Semantic meaning is independent of storage
- **Stable:** Definition of what "Customer" means doesn't change frequently
- **Referenceable:** Can be referenced from multiple places
- **Shared:** Can describe many things (entities, attributes, relationships)
- **Example:** "Customer means: external party who purchases services"

### Analogy
```
Think of a physical warehouse:

entity_attribute = The ACTUAL ITEM (physical product on shelf)
                   What we have, where it's located, how it's organized

catalog_node = The LABEL/DESCRIPTION (tag describing the product)
               What it is, what it means, why we have it
```

---

## ✅ Key Principles

1. **entity_attribute holds the actual entities**
   - One row = one entity type
   - Can be organized hierarchically (parent_id)
   - Scoped to tenant/datasource

2. **catalog_node describes what entities MEAN**
   - Semantic metadata and context
   - Display names, descriptions, versioning
   - Can reference many entities if needed

3. **No instance data in either**
   - These tables define TYPES, not instances
   - "Customer" type exists, but not "Acme Corp" (that's instance data)
   - Instance data lives in business tables (client_investors, portfolios, trades, etc.)

---

## � The Full Three-Tier Model

```
TIER 1: SEMANTIC MEANING
├─ catalog_node table
├─ "What does Customer mean?"
├─ Display name, description, version
└─ Stable business definitions

TIER 2: ENTITY CONTENT  
├─ entity_attribute table
├─ "We have a Customer entity"
├─ Hierarchy, naming, business context
└─ References semantic meaning (catalog_node_id FK)

TIER 3: INSTANCE DATA
├─ Multiple tables (client_investors, portfolios, trades, etc.)
├─ "Here are 50,000 actual customers"
├─ Real business data with values
└─ Each instance knows its entity type
```

---

## �️ File Locations

| Component | File | Purpose |
|-----------|------|---------|
| **entity_attribute table** | `/backend/migrations/000030_restructure_entity_schema_robust.sql` | Migration that creates the entity content table |
| **catalog_node table** | `/backend/migrations/000032_improved_catalog_schema.up.sql` | Migration that creates semantic metadata table |
| **API endpoints** | `/backend/internal/api/api.go` | Handles entity CRUD (business_entity handlers) |
| **Instance tables** | `/db/schema.sql` | Where actual data lives (client_investors, etc.) |

---

## 📚 Related Documentation

- **SEMANTIC_TERM_LINKING_GUIDE.md** - How to link entities to semantic terms
- **SEMANTIC_LINKING_ARCHITECTURE.md** - Visual architecture diagrams
- **SEMANTIC_LINKING_COMPLETE_FIX.md** - API response structure with catalogNodeId

---

## Next Steps

1. **Verify:** Check that `entity_attribute` has the entities you expect
   ```sql
   SELECT entity_key, name, business_name 
   FROM entity_attribute 
   WHERE tenant_id = 'your-tenant-id'
   ORDER BY entity_key;
   ```

2. **Check Links:** Verify entities link to semantic definitions
   ```sql
   SELECT ea.entity_key, cn.display_name, cn.description
   FROM entity_attribute ea
   LEFT JOIN catalog_node cn ON ea.catalog_node_id = cn.id
   WHERE ea.tenant_id = 'your-tenant-id';
   ```

3. **Instance Data:** Identify where Customer/Employee/Product instances are stored
   - If using separate tables: Confirm they're linked to entity types
   - If using generic table: Confirm entity_type_id references catalog_node

---

## Summary

**You were correct:**
- ✅ `entity_attribute` = stores actual entity CONTENT (definitions, hierarchy)
- ✅ `catalog_node` = stores METADATA describing those entities
- ✅ Entity points to catalog for semantic meaning
- ✅ No actual instance data (Customer records, Employee records) in either
- ✅ Instance data lives in business-specific tables
