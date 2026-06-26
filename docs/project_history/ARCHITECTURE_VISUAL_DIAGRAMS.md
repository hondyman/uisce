# Architecture Visual Diagrams

---

## 📊 Diagram 1: High-Level Architecture

```
┌──────────────────────────────────────────────────────────────────────┐
│                                                                      │
│                    SEMLAYER ENTITY ARCHITECTURE                      │
│                                                                      │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │  API Layer (/backend/internal/api/api.go)                     │ │
│  │  GET /api/business-entities                                   │ │
│  │  POST /api/business-entities (save)                           │ │
│  └────────────────────┬─────────────────────────────────────────┘ │
│                       │                                            │
│  ┌────────────────────▼─────────────────────────────────────────┐ │
│  │  Go Handlers                                                  │ │
│  │  • getBusinessEntities()                                     │ │
│  │  • saveBusinessEntities()                                    │ │
│  │  • buildResponseEntity()                                     │ │
│  └────────────────────┬─────────────────────────────────────────┘ │
│                       │                                            │
│  ┌────────────────────▼─────────────────────────────────────────┐ │
│  │  Database Layer (PostgreSQL)                                 │ │
│  │                                                               │ │
│  │  ┌──────────────────┐        ┌──────────────────┐           │ │
│  │  │entity_attribute  │        │ catalog_node     │           │ │
│  │  ├──────────────────┤        ├──────────────────┤           │ │
│  │  │id          │UUID │        │id       │ UUID   │           │ │
│  │  │entity_key  │TEXT │        │name     │ TEXT   │           │ │
│  │  │name        │TEXT │───────→│display_ │ TEXT   │           │ │
│  │  │parent_id   │UUID │ FK     │name             │           │ │
│  │  │catalog_node_id  │        │description      │           │ │
│  │  │is_core     │BOOL │        │version  │ INT    │           │ │
│  │  │tenant_id   │UUID │        │is_active│BOOL    │           │ │
│  │  └──────────────────┘        └──────────────────┘           │ │
│  │                                                               │ │
│  │  Stores:                       Catalogs:                     │ │
│  │  • Customer (entity)           • What Customer means         │ │
│  │  • Order (entity)              • What Order means            │ │
│  │  • Product (entity)            • Business context            │ │
│  │                                • Versioning                  │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
```

---

## 📈 Diagram 2: Data Flow - Single Entity

```
Request from API Client:
    GET /api/business-entities?tenant_id=t1&datasource_id=d1
           │
           ▼
    getBusinessEntities() in api.go
           │
           ├─ Read X-Tenant-ID header → tenant-001
           ├─ Read X-Tenant-Datasource-ID header → datasource-001
           │
           ▼
    Query database:
    ┌─────────────────────────────────────────────────┐
    │ SELECT id, entity_key, name, catalog_node_id   │
    │ FROM entity_attribute                           │
    │ WHERE tenant_id = $1                            │
    │   AND tenant_datasource_id = $2                │
    └─────────────────────────────────────────────────┘
           │
           ├─ Row 1: { id: uuid-1, entity_key: 'customer', 
           │           name: 'Customer', catalog_node_id: uuid-cat-1 }
           │
           ├─ Row 2: { id: uuid-2, entity_key: 'order',
           │           name: 'Order', catalog_node_id: uuid-cat-2 }
           │
           └─ Row 3: { id: uuid-3, entity_key: 'rush_order',
                       name: 'RushOrder', parent_id: uuid-2,
                       catalog_node_id: uuid-cat-3 }
           │
           ▼
    buildResponseEntity() - builds hierarchy:
    ┌─────────────────────────────────────────┐
    │ customer                                 │
    │   ├─ key: 'customer'                    │
    │   ├─ catalogNodeId: 'uuid-cat-1'   ←──┐│
    │   └─ subtypes: {}                       ││
    │                                         ││
    │ order                                   ││
    │   ├─ key: 'order'                      ││
    │   ├─ catalogNodeId: 'uuid-cat-2'       ││
    │   └─ subtypes:                         ││
    │       rush_order                       ││
    │         ├─ key: 'rush_order'           ││
    │         ├─ catalogNodeId:'uuid-cat-3'  ││
    │         └─ subtypes: {}                ││
    └─────────────────────────────────────────┘│
                                               │
    Response to Client (JSON):
    {
      "customer": {
        "key": "customer",
        "catalogNodeId": "uuid-cat-1"  ◄─────┘
      },
      "order": {
        "key": "order",
        "catalogNodeId": "uuid-cat-2",
        "subtypes": {
          "rush_order": {
            "key": "rush_order",
            "catalogNodeId": "uuid-cat-3"
          }
        }
      }
    }
```

---

## 🔄 Diagram 3: Entity Creation Flow

```
POST /api/business-entities
with JSON:
{
  "customer": { "name": "Customer", "catalogNodeId": "uuid-cat-1" }
}
           │
           ▼
    saveBusinessEntities() handler
           │
           ├─ Step 1: Validate input
           │
           ├─ Step 2: Start transaction
           │
           ├─ Step 3: DELETE old entities
           │   DELETE FROM entity_attribute
           │   WHERE tenant_id = 'tenant-001'
           │     AND tenant_datasource_id = 'datasource-001'
           │
           ├─ Step 4: INSERT new entities
           │   INSERT INTO entity_attribute 
           │     (tenant_id, tenant_datasource_id, entity_key, name,
           │      catalog_node_id, parent_id, is_core)
           │   VALUES
           │     ('tenant-001', 'datasource-001', 'customer', 'Customer',
           │      'uuid-cat-1', NULL, true)
           │
           ├─ Step 5: Handle subtypes recursively
           │   (if entity has subtypes, insert each with parent_id)
           │
           ├─ Step 6: Commit transaction
           │
           └─ Step 7: Return updated entities
                (same format as GET response)
```

---

## 🎯 Diagram 4: Entity Hierarchy Example

```
Database Storage:

entity_attribute table:
┌─────────────────────────────────────────────────────────────────┐
│ id    │entity_key │name      │parent_id │catalog_node_id      │
├─────────────────────────────────────────────────────────────────┤
│uuid-1 │customer   │Customer  │NULL      │uuid-cat-1           │
│uuid-2 │order      │Order     │NULL      │uuid-cat-2           │
│uuid-3 │rush_order │RushOrder │uuid-2    │uuid-cat-3           │◄── Parent: Order
│uuid-4 │payment    │Payment   │uuid-2    │uuid-cat-4           │◄── Parent: Order
└─────────────────────────────────────────────────────────────────┘

Hierarchy representation:

    customer
    
    order
    ├── rush_order
    └── payment

In query result:
SELECT id, entity_key, parent_id
FROM entity_attribute
ORDER BY entity_key;

         │
         ▼

Built into JSON hierarchy:
{
  "customer": { "key": "customer", "subtypes": {} },
  "order": {
    "key": "order",
    "subtypes": {
      "rush_order": { "key": "rush_order", "subtypes": {} },
      "payment": { "key": "payment", "subtypes": {} }
    }
  }
}
```

---

## 🔗 Diagram 5: Semantic Linking Chain

```
Frontend Application
        │
        ├─ User sees entity in UI: "Customer"
        │
        ▼
API Response received:
{
  "key": "customer",
  "catalogNodeId": "uuid-cat-1"  ◄─── Save this!
}
        │
        ▼
Frontend can now:
1. Query database for catalog_node WHERE id = "uuid-cat-1"
2. Get semantic details:
   - display_name: "Customer"
   - description: "External party who purchases"
   - version: 1
        │
        ▼
Display to user:
┌──────────────────────────────────────────┐
│ Entity: Customer                         │
│ Display Name: Customer                   │
│ Description: External party who...       │
│ Version: 1                               │
│ Active: true                             │
└──────────────────────────────────────────┘

Query to backend (backend-driven):
SELECT cn.* FROM catalog_node cn
WHERE cn.id = $1;  ◄─── Using catalogNodeId from API

Or in Go:
query := `SELECT id, display_name, description, version
          FROM catalog_node WHERE id = $1`
rows.Scan(&catalogNode)
```

---

## 📋 Diagram 6: Multi-Tenant Scoping

```
Tenant A (tenant-001) + Datasource A (ds-001)
┌────────────────────────────────────────┐
│ entity_attribute rows:                 │
│ ├─ Customer (Tenant A specific)        │
│ ├─ Order (Tenant A specific)           │
│ └─ Product (Tenant A specific)         │
│                                        │
│ Query: tenant_id = 'tenant-001'        │
│   AND tenant_datasource_id = 'ds-001' │
└────────────────────────────────────────┘

Tenant B (tenant-002) + Datasource B (ds-002)
┌────────────────────────────────────────┐
│ entity_attribute rows:                 │
│ ├─ Account (Tenant B specific)         │
│ ├─ Deal (Tenant B specific)            │
│ └─ Client (Tenant B specific)          │
│                                        │
│ Query: tenant_id = 'tenant-002'        │
│   AND tenant_datasource_id = 'ds-002' │
└────────────────────────────────────────┘

Shared (Global)
┌────────────────────────────────────────┐
│ catalog_node rows:                     │
│ ├─ customer semantic definition        │
│ ├─ order semantic definition           │
│ ├─ product semantic definition         │
│ ├─ account semantic definition         │
│ ├─ deal semantic definition            │
│ └─ client semantic definition          │
│                                        │
│ Note: No tenant_id FK!                 │
│ Same semantic meanings across all      │
└────────────────────────────────────────┘
```

---

## 🎪 Diagram 7: Instance Data (Separate)

```
Note: These are NOT in entity_attribute or catalog_node!

actual_customer_instances:
┌──────────────────────────────────────────────────┐
│ Table: client_investors                         │
│ ├─ id: cust-001, name: "Acme Corp"             │
│ ├─ id: cust-002, name: "TechCorp Inc"          │
│ ├─ id: cust-003, name: "Global Partners"       │
│ └─ ... 50,000 more customer records ...         │
│                                                  │
│ These are INSTANCES of the Customer entity     │
│ Not stored in entity_attribute!                │
└──────────────────────────────────────────────────┘

entity_attribute:
┌──────────────────────────────────────────────────┐
│ id: uuid-1                                       │
│ entity_key: "customer"                           │
│ name: "Customer"                                 │
│ catalog_node_id: "uuid-cat-1"                  │
│                                                  │
│ This is the TYPE definition                    │
│ Not actual customer records!                    │
└──────────────────────────────────────────────────┘

catalog_node:
┌──────────────────────────────────────────────────┐
│ id: "uuid-cat-1"                               │
│ name: "customer"                                 │
│ display_name: "Customer"                         │
│ description: "External party who purchases"      │
│                                                  │
│ This DESCRIBES what the Customer type means    │
│ Not data storage!                               │
└──────────────────────────────────────────────────┘

CHAIN:
entity_attribute (what types exist)
    ↓
catalog_node (what they mean)
    ↓
client_investors (actual instances)
```

---

## 🌳 Diagram 8: Complete System Overview

```
┌────────────────────────────────────────────────────────────────┐
│                    COMPLETE SEMLAYER MODEL                     │
└────────────────────────────────────────────────────────────────┘

┌─────────────────────┐
│    API / Frontend   │
│  (JavaScript, etc)  │
└──────────┬──────────┘
           │
           │ HTTP Request with headers
           │ X-Tenant-ID: tenant-001
           │ X-Tenant-Datasource-ID: ds-001
           │
           ▼
┌─────────────────────────────────────────────────────────────────┐
│           Go Backend (/internal/api/api.go)                     │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  GET /api/business-entities:                                   │
│  - Query entity_attribute (scoped by tenant)                   │
│  - Build response with catalogNodeId                           │
│  - Return hierarchical JSON                                    │
│                                                                 │
│  POST /api/business-entities:                                  │
│  - Parse JSON input                                            │
│  - Delete old entities                                         │
│  - Insert new entities (with catalog_node_id FK)              │
│  - Return updated list                                         │
│                                                                 │
└──────────┬──────────────────────────────────────────────────────┘
           │
           ▼
┌─────────────────────────────────────────────────────────────────┐
│              PostgreSQL Database                                │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  LAYER 1: Entity Content (Tenant-Scoped)                       │
│  ┌──────────────────────────────────────────┐                 │
│  │ entity_attribute                         │                 │
│  │ • customer (uuid-1)                      │                 │
│  │ • order (uuid-2)                         │                 │
│  │ • rush_order (uuid-3, parent: uuid-2)   │                 │
│  │                                          │                 │
│  │ Filtered by:                             │                 │
│  │ • tenant_id = 'tenant-001'              │                 │
│  │ • tenant_datasource_id = 'ds-001'       │                 │
│  └──────────────────────────────────────────┘                 │
│                    │                                            │
│            FK Reference                                         │
│            catalog_node_id                                      │
│                    │                                            │
│                    ▼                                            │
│  LAYER 2: Semantic Catalog (Global/Shared)                    │
│  ┌──────────────────────────────────────────┐                 │
│  │ catalog_node                             │                 │
│  │ • customer (uuid-cat-1)                  │                 │
│  │   display_name: "Customer"               │                 │
│  │   description: "External party..."       │                 │
│  │ • order (uuid-cat-2)                     │                 │
│  │   display_name: "Order"                  │                 │
│  │   description: "Purchase request..."     │                 │
│  │ • rush_order (uuid-cat-3)               │                 │
│  │   display_name: "RushOrder"              │                 │
│  │   description: "Expedited order..."      │                 │
│  └──────────────────────────────────────────┘                 │
│                                                                 │
│  LAYER 3: Instance Data (Multiple Tables)                      │
│  ┌──────────────────────────────────────────┐                 │
│  │ client_investors (50,000+ records)      │                 │
│  │ portfolios (1,000+ records)              │                 │
│  │ trades (100,000+ records)                │                 │
│  │ ... other business data tables ...       │                 │
│  └──────────────────────────────────────────┘                 │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Summary

- **Blue**: Content (entity_attribute)
- **Orange**: Metadata (catalog_node)
- **Green**: Instance data (business tables)

Each layer serves a distinct purpose, with proper separation of concerns.
