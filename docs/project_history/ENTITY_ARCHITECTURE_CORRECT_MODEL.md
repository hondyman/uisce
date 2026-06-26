# Entity Architecture - Correct Model (Quick Reference)

**The Correct Separation of Concerns**

---

## ⚡ Quick Summary

```
📦 entity_attribute TABLE = STORES ACTUAL ENTITIES
   └─ What entities exist: "Customer", "Order", "Product"
   └─ How they relate: hierarchy, parent-child
   └─ Entity naming: entity_key, business_name, technical_name

🏷️ catalog_node TABLE = DESCRIBES WHAT ENTITIES MEAN  
   └─ Semantic metadata: display_name, description
   └─ Business context: what this entity represents
   └─ Versioning: track changes to meaning over time

🔗 RELATIONSHIP = entity_attribute.catalog_node_id → catalog_node.id
   └─ Entity points to its semantic definition
   └─ One FK connection: "I'm a Customer, and here's what that means"
```

---

## 🎯 Simple Analogy

```
Think of a warehouse system:

entity_attribute = THE ITEM ON THE SHELF
                   Physical product that exists
                   Has a location, size, weight
                   Can be organized in categories

catalog_node = THE PRODUCT DESCRIPTION CARD
              Attached to the item
              Says what it is and why we have it
              Can be updated without moving the item
```

---

## 📊 Visual Comparison

```
❌ WRONG WAY (What I said initially):
┌─────────────────────────────────────────┐
│ catalog_node: Holds entity definitions  │
│ (This is backwards!)                    │
└─────────────────────────────────────────┘

✅ CORRECT WAY (Your way):
┌─────────────────────────────────────────┐
│ entity_attribute: Holds entity content  │
│   ├─ Customer (entity_key: 'customer')  │
│   ├─ Order (entity_key: 'order')        │
│   └─ Product (entity_key: 'product')    │
│                                         │
│ Points to ↓                             │
│                                         │
│ catalog_node: Describes the entity     │
│   ├─ "Customer means: external party"  │
│   ├─ "Order means: purchase request"   │
│   └─ "Product means: sellable item"    │
└─────────────────────────────────────────┘
```

---

## 🗂️ Tables Explained

### `entity_attribute` Table
**What it stores:** The actual entity definitions

```sql
Column              | Type    | Purpose
─────────────────────────────────────────────────────────
id                  | UUID    | Primary key
tenant_id           | UUID    | Tenant scope
tenant_datasource_id| UUID    | Datasource scope
entity_key          | TEXT    | Internal name: 'customer', 'order'
name                | TEXT    | Display name: 'Customer', 'Order'
business_name       | TEXT    | Business context name
technical_name      | TEXT    | Technical/system name
parent_id           | UUID    | Parent entity (for hierarchy)
catalog_node_id     | UUID    | FK → catalog_node (what it MEANS)
is_core             | BOOLEAN | Is this a core/system entity?
created_at          | TIMESTAMP
updated_at          | TIMESTAMP
```

**Example data:**
```
id                 | entity_key | name     | parent_id | catalog_node_id | business_name
────────────────────────────────────────────────────────────────────────────────────────────
550e8400-e29b-...  | customer   | Customer | NULL      | uuid-123        | Customer
550e8401-e29b-...  | order      | Order    | NULL      | uuid-124        | Purchase Order
550e8402-e29b-...  | rush_order | RushOrder| 550e8400..| uuid-125        | Rush Order
```

### `catalog_node` Table
**What it stores:** Semantic metadata describing entities

```sql
Column       | Type      | Purpose
──────────────────────────────────────────────────
id           | UUID      | Primary key
name         | TEXT      | Internal name
display_name | TEXT      | Display name
description  | TEXT      | What it means
type         | VARCHAR   | Type: 'BusinessEntity', 'Attribute', etc.
version      | INTEGER   | Version number
is_active    | BOOLEAN   | Active flag
created_at   | TIMESTAMP
updated_at   | TIMESTAMP
```

**Example data:**
```
id         | name     | display_name | description
──────────────────────────────────────────────────────────────────────
uuid-123   | customer | Customer     | External party who purchases
uuid-124   | order    | Order        | Purchase request from customer
uuid-125   | rush_ord | RushOrder    | Order with expedited fulfillment
```

---

## 🔗 How They Connect

```sql
-- The connection:
-- entity_attribute.catalog_node_id FK → catalog_node.id

SELECT 
    ea.entity_key,           -- "customer"
    ea.name,                 -- "Customer"
    cn.display_name,         -- "Customer"  
    cn.description           -- "External party who..."
FROM entity_attribute ea
LEFT JOIN catalog_node cn ON ea.catalog_node_id = cn.id;

-- Result shows entity + what it means
```

---

## ✅ The Key Insight

**`entity_attribute` owns the definition; `catalog_node` describes it.**

```
Think of it like this:

entity_attribute says:
  "I have an entity type called 'Customer'
   Its entity_key is 'customer'
   And I want to know what it MEANS"
   
catalog_node says:
  "The 'customer' semantic term means:
   External party who purchases products/services
   Versioned, stable definition"
```

---

## 📍 Where Instance Data Lives

**Neither `entity_attribute` nor `catalog_node` store actual instance data!**

```
entity_attribute + catalog_node = Entity TYPES (definitions)
                                 "What entity types exist?"

client_investors + portfolios + trades = Entity INSTANCES (data)
                                        "Here are actual customers,
                                         portfolios, trades"
```

**Example:**
```
entity_attribute row:
  entity_key: 'customer'
  catalog_node_id: uuid-123
  
catalog_node row (uuid-123):
  name: 'customer'
  description: 'External party who purchases'
  
client_investors table:
  ✅ Where actual Customer instances are:
  - id: abc-001, name: 'Acme Corp', type: 'ClientInvestor'
  - id: abc-002, name: 'TechCorp Inc', type: 'ClientInvestor'
  - ... 50,000 customer records ...
```

---

## 🏛️ Three-Tier Architecture

```
┌─────────────────────────────────────────────────────────┐
│                                                         │
│  TIER 1: SEMANTIC MEANING                              │
│  ┌──────────────────────────────────────────────────┐  │
│  │ catalog_node table                               │  │
│  │ "What does Customer mean?"                       │  │
│  │ Display name, description, version               │  │
│  └──────────────────────────────────────────────────┘  │
│                      ▲                                  │
│                      │ referenced by                    │
│  ┌──────────────────────────────────────────────────┐  │
│  │ TIER 2: ENTITY CONTENT                           │  │
│  │ ┌────────────────────────────────────────────┐  │  │
│  │ │ entity_attribute table                     │  │  │
│  │ │ "We have a Customer entity"                │  │  │
│  │ │ entity_key, name, parent_id                │  │  │
│  │ │ catalog_node_id → references TIER 1       │  │  │
│  │ └────────────────────────────────────────────┘  │  │
│  └──────────────────────────────────────────────────┘  │
│                                                         │
│  TIER 3: INSTANCE DATA                                 │
│  ┌──────────────────────────────────────────────────┐  │
│  │ client_investors, portfolios, trades, etc.       │  │
│  │ "Here are 50,000 actual customers"               │  │
│  │ Real business data with values                   │  │
│  └──────────────────────────────────────────────────┘  │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

---

## 💡 Why This Design?

| Aspect | entity_attribute | catalog_node |
|--------|------------------|--------------|
| **Contains** | Entity definitions | Semantic metadata |
| **Scope** | Tenant/Datasource | Global (shared) |
| **Mutability** | Mutable (add entities) | Stable (rarely changes) |
| **Purpose** | "What entities do we have?" | "What do they mean?" |
| **Use Case** | Entity management | Business understanding |

---

## ✨ Migration Path (If Needed)

If you previously had entities defined elsewhere, here's how they move:

```sql
-- Old way: Entities defined in some_old_table
-- New way: Entities in entity_attribute + catalog_node

-- 1. Insert into catalog_node first (semantic meaning)
INSERT INTO catalog_node (name, display_name, description, type, version, is_active)
SELECT DISTINCT 
    entity_name,
    entity_display_name,
    entity_description,
    'BusinessEntity',
    1,
    true
FROM some_old_table;

-- 2. Insert into entity_attribute (entity content), linking to catalog_node
INSERT INTO entity_attribute 
    (tenant_id, tenant_datasource_id, entity_key, name, catalog_node_id, is_core)
SELECT 
    t.tenant_id,
    t.datasource_id,
    ot.entity_name,
    ot.entity_display_name,
    cn.id,  -- Link to semantic definition
    false
FROM some_old_table ot
LEFT JOIN catalog_node cn ON ot.entity_name = cn.name
LEFT JOIN tenants t ON ot.tenant_id = t.id;
```

---

## 🔍 Verification Queries

### Check entity_attribute
```sql
-- See all entities in a datasource
SELECT entity_key, name, business_name, catalog_node_id 
FROM entity_attribute
WHERE tenant_id = 'your-tenant-id'
  AND tenant_datasource_id = 'your-datasource-id'
ORDER BY entity_key;
```

### Check catalog_node
```sql
-- See all semantic definitions
SELECT name, display_name, description, version
FROM catalog_node
WHERE is_active = true
ORDER BY name;
```

### Check the link
```sql
-- See entities with their semantic meaning
SELECT 
    ea.entity_key,
    ea.name as entity_name,
    cn.display_name as semantic_display_name,
    cn.description as semantic_meaning
FROM entity_attribute ea
LEFT JOIN catalog_node cn ON ea.catalog_node_id = cn.id
WHERE ea.tenant_id = 'your-tenant-id'
ORDER BY ea.entity_key;
```

### Check hierarchy
```sql
-- See parent-child relationships
SELECT 
    child.entity_key as child_entity,
    parent.entity_key as parent_entity,
    child.catalog_node_id
FROM entity_attribute child
LEFT JOIN entity_attribute parent ON child.parent_id = parent.id
WHERE child.tenant_id = 'your-tenant-id'
  AND child.parent_id IS NOT NULL;
```

---

## 📝 Summary

| Question | Answer |
|----------|--------|
| **Where do entity definitions go?** | `entity_attribute` table |
| **Where does semantic metadata go?** | `catalog_node` table |
| **How are they linked?** | `entity_attribute.catalog_node_id` → `catalog_node.id` |
| **Who owns whom?** | Entity owns definition, points to catalog for meaning |
| **Where do actual instances go?** | Business-specific tables (client_investors, etc.) |
| **What can I query?** | Entity + its meaning, hierarchy, relationships |

**You're right: catalog_node should only describe objects, not hold them. `entity_attribute` is the content table.**
