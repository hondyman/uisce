# Relationship Discovery & Self-Service Reporting Guide

**Date:** November 7, 2025  
**Feature:** Entity Relationship Discovery with Semantic Linking

---

## 🎯 What You're Building

When you click **"Add Relationship"** on an entity, you want to discover and display:

1. **Related entities** (via foreign key chains)
2. **How they're related** (the connection path)
3. **Which keys allow linking** (FK columns)
4. **Semantic context** (what the relationships mean)
5. **Self-service reporting capability** (use for dashboards)

---

## 🔗 The Relationship Chain You Described

```
┌──────────────────────────────────────────────────────────────────────┐
│                    RELATIONSHIP DISCOVERY CHAIN                      │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Entity A (e.g., "Customer")                                         │
│  └─ has Attribute (e.g., "customer_id")                             │
│     └─ linked to Semantic Term (e.g., "customer_semantic")          │
│        └─ related to Column (e.g., "customers.id")                  │
│           └─ FK Parent Table (e.g., "payments.customer_id →         │
│              └─ Parent Table: "customers")                          │
│                 └─ Column linked to Semantic Term                   │
│                    └─ Used in Entity B (e.g., "Payment")            │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
```

---

## 📊 Current Implementation (What Exists)

### File: `/backend/internal/api/relationships_discovery.go`

**Main Service:** `RelationshipDiscoveryService`

**Key Function:** `DiscoverLinkableEntities(ctx, tenantID, datasourceID, entityName)`

**What it does:**
1. Finds the source table node for entity
2. Discovers direct FK relationships (both outbound and inbound)
3. Returns related entities with linking information

---

## 🔄 How It Works Now

### Step 1: Find Source Entity
```sql
SELECT DISTINCT cn.id as table_id, cn.node_name as table_name
FROM catalog_node cn
WHERE cn.node_name = 'customer'
  AND cn.tenant_datasource_id = $datasource_id
```

### Step 2: Find Foreign Key Relationships
```sql
-- Outbound FKs: "customer" table → other tables
SELECT ce.target_node_id, ct.node_name as target_table
FROM catalog_edge ce
WHERE ce.source_node_id = customer_table_id
  AND ce.relationship_type = 'foreign_key'

-- Inbound FKs: other tables → "customer" table
SELECT ce.source_node_id, cs.node_name as source_table
FROM catalog_edge ce
WHERE ce.target_node_id = customer_table_id
  AND ce.relationship_type = 'foreign_key'
```

### Step 3: Return Related Entities
```
{
  "sourceEntity": "customer",
  "relationships": [
    {
      "targetEntity": "payment",
      "cardinality": "one-to-many",
      "linkType": "outbound",
      "description": "payment table has foreign key to customer table",
      "keyFields": {
        "source": "customer(id)",
        "target": "payment(customer_id)"
      }
    },
    {
      "targetEntity": "order",
      "cardinality": "one-to-many",
      "linkType": "outbound",
      "description": "order table has foreign key to customer table",
      "keyFields": {
        "source": "customer(id)",
        "target": "order(customer_id)"
      }
    }
  ]
}
```

---

## 🚀 What Needs Enhancement

Based on your description, enhance the discovery to include:

### 1. Semantic Term Linking in Relationships
```go
type RelatedEntity struct {
    // Existing fields...
    EntityID       string
    EntityName     string
    
    // NEW: Semantic context
    SemanticTermID   string `json:"semantic_term_id"`
    SemanticTermName string `json:"semantic_term_name"`
    
    // NEW: Attribute mapping
    SourceAttribute   string `json:"source_attribute"`
    TargetAttribute   string `json:"target_attribute"`
    
    // NEW: Column hierarchy
    SourceColumn      string `json:"source_column"`
    TargetColumn      string `json:"target_column"`
    ColumnParentTable string `json:"column_parent_table"`
}
```

### 2. Enhanced Discovery Query
```sql
WITH source_entity AS (
  SELECT ea.id, ea.catalog_node_id, ea.entity_key
  FROM entity_attribute ea
  WHERE ea.entity_key = $entity_name
    AND ea.tenant_datasource_id = $datasource_id
),

entity_attributes AS (
  -- Attributes of the source entity linked to semantic terms
  SELECT 
    ea.id as attribute_id,
    ea.entity_key as attribute_name,
    cn.id as semantic_term_id,
    cn.name as semantic_term_name,
    cn.display_name as semantic_display_name
  FROM entity_attribute ea
  LEFT JOIN catalog_node cn ON ea.catalog_node_id = cn.id
  WHERE ea.parent_id IN (SELECT id FROM source_entity)
),

column_mappings AS (
  -- Columns that link to these semantic terms
  SELECT 
    cc.column_name,
    cc.table_name,
    cc.catalog_node_id,
    cn.name as semantic_term_name
  FROM catalog_column cc
  LEFT JOIN catalog_node cn ON cc.catalog_node_id = cn.id
  WHERE cc.catalog_node_id IN (
    SELECT semantic_term_id FROM entity_attributes
  )
),

foreign_key_paths AS (
  -- Foreign key constraints connecting these columns
  SELECT 
    tc1.table_name as source_table,
    kcu1.column_name as source_column,
    tc2.table_name as target_table,
    kcu2.column_name as target_column,
    'foreign_key' as link_type
  FROM information_schema.table_constraints tc1
  JOIN information_schema.key_column_usage kcu1 
    ON tc1.table_name = kcu1.table_name
  JOIN information_schema.referential_constraints rc
    ON tc1.constraint_name = rc.constraint_name
  JOIN information_schema.table_constraints tc2
    ON tc2.constraint_name = rc.unique_constraint_name
  JOIN information_schema.key_column_usage kcu2
    ON tc2.table_name = kcu2.table_name
  WHERE kcu1.column_name IN (SELECT column_name FROM column_mappings)
     OR kcu2.column_name IN (SELECT column_name FROM column_mappings)
),

related_entities AS (
  -- Entities used in the target tables
  SELECT DISTINCT
    ea2.id as entity_id,
    ea2.entity_key as entity_name,
    cn2.id as semantic_term_id,
    cn2.name as semantic_term_name,
    fkp.source_table,
    fkp.source_column,
    fkp.target_table,
    fkp.target_column
  FROM foreign_key_paths fkp
  LEFT JOIN entity_attribute ea2 ON ea2.entity_key = fkp.target_table
  LEFT JOIN catalog_node cn2 ON ea2.catalog_node_id = cn2.id
  WHERE ea2.tenant_datasource_id = $datasource_id
)

SELECT * FROM related_entities;
```

---

## 🔑 Key Concepts

### Relationship Types

| Type | Description | Example |
|------|-------------|---------|
| **Direct FK** | Immediate foreign key | customer.id ← payment.customer_id |
| **Semantic Link** | Via shared semantic term | Both entities use "customer_id" semantic |
| **Column Hierarchy** | Via parent table relationships | order_item.order_id → order.customer_id |
| **Multi-hop** | Multiple relationships deep | customer → order → order_item → product |

### Cardinality (How many to many)
- **one-to-one** (1:1) - Each customer has one account
- **one-to-many** (1:N) - One customer has many orders
- **many-to-one** (N:1) - Many payments belong to one customer
- **many-to-many** (N:M) - Students enrolled in many courses

---

## 💡 Frontend Integration

### When User Clicks "Add Relationship"

**Request:**
```bash
GET /api/relationships/objects?tenant_id=t1&datasource_id=d1&entity=customer
```

**Response:**
```json
{
  "sourceEntity": "customer",
  "semanticContext": {
    "semanticTermId": "uuid-123",
    "semanticTermName": "customer",
    "displayName": "Customer",
    "description": "External party who purchases"
  },
  "relationships": [
    {
      "id": "rel-1",
      "targetEntity": "order",
      "targetSemanticTerm": {
        "id": "uuid-456",
        "name": "order",
        "displayName": "Order"
      },
      "cardinality": "one-to-many",
      "linkType": "foreign_key",
      "description": "Customer has many orders",
      "keyFields": {
        "source": {
          "entity": "customer",
          "attribute": "id",
          "column": "customers.id",
          "semanticTerm": "customer_id"
        },
        "target": {
          "entity": "order",
          "attribute": "customer_id",
          "column": "orders.customer_id",
          "semanticTerm": "customer_id"
        }
      },
      "fkConstraint": "orders.customer_id -> customers.id",
      "confidence": 0.95
    },
    {
      "id": "rel-2",
      "targetEntity": "payment",
      "targetSemanticTerm": {
        "id": "uuid-789",
        "name": "payment",
        "displayName": "Payment"
      },
      "cardinality": "one-to-many",
      "linkType": "foreign_key",
      "description": "Customer has many payments",
      "keyFields": {
        "source": {
          "entity": "customer",
          "attribute": "id",
          "column": "customers.id",
          "semanticTerm": "customer_id"
        },
        "target": {
          "entity": "payment",
          "attribute": "customer_id",
          "column": "payments.customer_id",
          "semanticTerm": "customer_id"
        }
      },
      "fkConstraint": "payments.customer_id -> customers.id",
      "confidence": 0.95
    }
  ],
  "count": 2
}
```

---

## 📈 Self-Service Reporting Usage

Once relationships are discovered and applied, users can:

### 1. Create Cross-Entity Reports
```
Customer → Orders → Order Items → Products

Build a report showing:
- Customer Name
- Total Orders
- Average Order Value (from orders)
- Product Count per Order (from order_items)
- Product Categories (from products)
```

### 2. Query Through Relationships
```sql
-- Self-service SQL generated from relationship chain
SELECT 
  c.name as customer_name,
  COUNT(o.id) as order_count,
  AVG(oi.quantity * oi.unit_price) as avg_item_value,
  p.product_name,
  p.category
FROM customers c
LEFT JOIN orders o ON c.id = o.customer_id
LEFT JOIN order_items oi ON o.id = oi.order_id
LEFT JOIN products p ON oi.product_id = p.id
GROUP BY c.name, p.product_name, p.category;
```

### 3. Build Dashboards
Users select relationships to include:
- ✓ Customer (root entity)
- ✓ Customer → Orders
- ✓ Orders → Order Items
- ✓ Order Items → Products

Dashboard shows related metrics in context.

---

## 🛠️ Implementation Steps

### Step 1: Enhance RelatedEntity Struct
```go
type RelatedEntity struct {
    EntityID            string `json:"entity_id"`
    EntityName          string `json:"entity_name"`
    
    // Semantic context
    SemanticTermID      string `json:"semantic_term_id"`
    SemanticTermName    string `json:"semantic_term_name"`
    SemanticDisplay     string `json:"semantic_display"`
    
    // Attributes
    SourceAttribute     string `json:"source_attribute"`
    TargetAttribute     string `json:"target_attribute"`
    
    // Columns
    SourceColumn        string `json:"source_column"`
    TargetColumn        string `json:"target_column"`
    ColumnParentTable   string `json:"column_parent_table"`
    
    // Relationship properties
    TableName           string `json:"table_name"`
    LinkType            string `json:"link_type"`
    Cardinality         string `json:"cardinality"`
    LinkReason          string `json:"link_reason"`
    ForeignKeyPath      string `json:"foreign_key_path"`
    ForeignKeyConstraint string `json:"fk_constraint"`
    
    DiscoveredAt        time.Time `json:"discovered_at"`
}
```

### Step 2: Enhance Discovery Query
Add semantic term and column hierarchy information to the discovery query.

### Step 3: Add Relationship Application
```go
func (s *Server) applyRelationship(w http.ResponseWriter, r *http.Request) {
    // Save the relationship to:
    // 1. catalog_edge (FK relationship)
    // 2. entity_relationship (semantic relationship)
    // 3. semantic_link (semantic term linking)
}
```

### Step 4: Add Relationship Query for Reporting
```go
func (s *Server) queryRelationshipChain(w http.ResponseWriter, r *http.Request) {
    // Given an entity and a list of relationships to include,
    // generate SQL query or metadata for self-service reporting
}
```

---

## 📊 Database Schema Support Needed

### Existing Tables
- `entity_attribute` - Entity definitions
- `catalog_node` - Semantic terms
- `catalog_edge` - FK relationships
- `catalog_column` - Column definitions

### Needed Enhancements
```sql
-- Link columns to semantic terms
ALTER TABLE catalog_column 
ADD COLUMN catalog_node_id UUID REFERENCES catalog_node(id);

-- Link entity attributes to columns
CREATE TABLE entity_attribute_column_mapping (
    id UUID PRIMARY KEY,
    entity_attribute_id UUID REFERENCES entity_attribute(id),
    column_name TEXT,
    table_name TEXT,
    semantic_term_id UUID REFERENCES catalog_node(id),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Store applied relationships
CREATE TABLE entity_relationship (
    id UUID PRIMARY KEY,
    tenant_datasource_id UUID REFERENCES tenant_product_datasource(id),
    source_entity_id UUID REFERENCES entity_attribute(id),
    target_entity_id UUID REFERENCES entity_attribute(id),
    relationship_type VARCHAR(255),
    fk_constraint TEXT,
    cardinality VARCHAR(50),
    description TEXT,
    confidence NUMERIC(3,2),
    created_at TIMESTAMP DEFAULT NOW(),
    created_by VARCHAR(255)
);
```

---

## 🎯 Complete User Flow

### 1. User opens Entity "Customer"
```
UI loads entity details
Shows "Add Relationship" button
```

### 2. User clicks "Add Relationship"
```
API: GET /relationships/objects?entity=customer
Discovery finds: Order, Payment, Invoice entities
Response includes key fields and cardinality
```

### 3. UI displays discovered relationships
```
User sees:
- "Customer → Order" (1:N)
- "Customer → Payment" (1:N)
- "Customer → Invoice" (1:N)

Each with FK path visualization
```

### 4. User selects relationships to add
```
POST /relationships/apply
{
  "sourceEntity": "customer",
  "targetEntity": "order",
  "cardinality": "one-to-many",
  "fkColumn": "customer_id"
}
```

### 5. Relationships saved and available for reporting
```
User can now:
- Build reports using Customer + Order data
- Join on customer_id automatically
- Add metrics from both entities
- Create self-service dashboards
```

---

## 📝 Summary

**What you're building:**
- Discover related entities through FK chains
- Show how they're related (via keys)
- Link to semantic terms for context
- Enable self-service reporting on those relationships

**Current state:**
- ✅ Basic FK discovery works (`getRelatedObjects`)
- ✅ Relationship suggestions exist
- ✅ Can apply relationships
- ❌ Semantic context not fully integrated
- ❌ Column hierarchy not exposed
- ❌ Reporting queries not generated

**What's needed:**
1. Enhanced discovery with semantic + column hierarchy
2. Better visualization of relationship paths
3. Self-service reporting query generation
4. Relationship metadata storage

---

## 🔗 Related Files

- `/backend/internal/api/relationships_discovery.go` - Discovery service
- `/backend/internal/api/api.go` - API endpoints (lines 6282-6450)
- `/backend/migrations/` - Database schema
- `/db/schema.sql` - Full schema reference

Would you like me to:
1. Enhance the discovery service to include semantic terms?
2. Create the relationship reporting query generator?
3. Add the database schema for relationship storage?
4. Build the frontend integration documentation?
