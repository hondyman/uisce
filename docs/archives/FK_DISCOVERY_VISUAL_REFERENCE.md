# Entity Relationship Discovery via Foreign Keys - Visual Reference

## Architecture Diagram

```
┌──────────────────────────────────────────────────────────────────────┐
│                                                                      │
│                    FRONTEND LAYER                                    │
│                    ───────────────                                   │
│                                                                      │
│  Entity Relationships Tab                                           │
│  ├─ Related Objects List                                            │
│  ├─ Suggested Relationships                                         │
│  └─ Create/Edit Relationship UI                                     │
│                                                                      │
│  Calls: GET /entities/{id}/foreign-keys                            │
│         POST /entities/{id}/discover-and-link-relationships        │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
                              ▼ HTTP
┌──────────────────────────────────────────────────────────────────────┐
│                                                                      │
│                     API LAYER                                        │
│                     ─────────                                        │
│                                                                      │
│  ┌────────────────────────────────────────────────────────────┐   │
│  │  RelationshipService                                       │   │
│  ├─ GetRelationshipSuggestions() [EXISTING]                  │   │
│  ├─ GetRelationshipSuggestionsWithFK() [NEW]                 │   │
│  └─ convertFKToSuggestions()                                 │   │
│  └─ mergeSuggestions()                                       │   │
│  └─ fkEngine: *ForeignKeyDiscoveryEngine                     │   │
│  └────────────────────────────────────────────────────────────┘   │
│                           │                                         │
│                           ├─ calls ─────────────────────┐         │
│                           │                            ▼         │
│  ┌────────────────────────────────────────────────────────────┐   │
│  │  ForeignKeyDiscoveryEngine [NEW]                           │   │
│  │  (fk_discovery_engine.go)                                  │   │
│  │                                                             │   │
│  │  ├─ DiscoverForeignKeysForTable()                         │   │
│  │  │  └─ Queries: FK edges from catalog_edge               │   │
│  │  │  └─ Handles: outbound + inbound FKs                   │   │
│  │  │  └─ Returns: []ForeignKeyRelationship                 │   │
│  │  │                                                         │   │
│  │  ├─ DiscoverEntityRelationshipsFromFK()                   │   │
│  │  │  ├─ Gets entity backing table(s)                      │   │
│  │  │  ├─ Queries FKs for that table                        │   │
│  │  │  ├─ For each FK: finds target entity                  │   │
│  │  │  └─ Returns: []EntityRelationshipFromFK               │   │
│  │  │                                                         │   │
│  │  ├─ CreateEntityRelationshipEdgeFromFK()                  │   │
│  │  │  ├─ Creates edge in catalog_edge                      │   │
│  │  │  ├─ Stores FK details in properties JSON              │   │
│  │  │  └─ Returns: edge ID                                  │   │
│  │  │                                                         │   │
│  │  ├─ Helper Functions:                                     │   │
│  │  │  ├─ extractColumnMappings()                           │   │
│  │  │  ├─ inferCardinality()                                │   │
│  │  │  ├─ inferRelationType()                               │   │
│  │  │  ├─ getEntityBackingTables()                          │   │
│  │  │  ├─ findEntityByBackingTable()                        │   │
│  │  │  └─ getEdgeTypeID()                                   │   │
│  │                                                             │   │
│  └────────────────────────────────────────────────────────────┘   │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
                              ▼ SQL
┌──────────────────────────────────────────────────────────────────────┐
│                                                                      │
│                   DATABASE LAYER                                     │
│                   ───────────────                                    │
│                                                                      │
│  catalog_edge (Foreign Key Edges)                                   │
│  ├─ id: UUID                                                        │
│  ├─ source_node_id: UUID (source table node)                       │
│  ├─ target_node_id: UUID (target table node)                       │
│  ├─ relationship_type: 'foreign_key'                               │
│  ├─ properties: {                                                   │
│  │   "foreign_key_constraints": ["fk_name"],                       │
│  │   "source_column": "account_id",                                │
│  │   "target_column": "id",                                        │
│  │   "columns": [                                                  │
│  │     {"source_column": "account_id", "target_column": "id"}     │
│  │   ]                                                              │
│  │ }                                                                │
│  └─ created_at, updated_at                                         │
│                                                                      │
│  catalog_node (Table Nodes)                                         │
│  ├─ id: UUID                                                        │
│  ├─ node_name: 'customers', 'accounts', etc.                       │
│  ├─ node_type_id: table type UUID                                  │
│  └─ properties: {...}                                              │
│                                                                      │
│  entities (Business Entities)                                       │
│  ├─ id: UUID                                                        │
│  ├─ name: 'Customer', 'Account', etc.                              │
│  ├─ table_name: 'customers', 'accounts'   ◄─── KEY LINK           │
│  ├─ schema_name: 'public'                                          │
│  └─ properties: {...}                                              │
│                                                                      │
│  catalog_edge (Entity Relationship Edges) [CREATED BY FK DISCOVERY] │
│  ├─ id: UUID                                                        │
│  ├─ source_node_id: UUID (source entity)                           │
│  ├─ target_node_id: UUID (target entity)                           │
│  ├─ relationship_type: 'entity_relationship_fk'                    │
│  ├─ properties: {                                                   │
│  │   "discovery_method": "foreign_key_analysis",                   │
│  │   "source_table": "customers",                                  │
│  │   "target_table": "accounts",                                   │
│  │   "fk_edge_id": "<uuid>",                ◄─── Links to FK edge  │
│  │   "cardinality": "many-to-one",                                 │
│  │   "relation_type": "reference"                                  │
│  │ }                                                                │
│  └─ created_at, updated_at                                         │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
```

## Discovery Flow Diagram

```
INPUT: Entity { id: "entity-1", name: "Customer" }
       tenant_id: "t1", datasource_id: "d1"
   │
   ▼
┌─────────────────────────────────────────┐
│  Step 1: Get Backing Table(s)          │
│  ───────────────────────────────────────│
│                                         │
│  Query: SELECT table_name FROM entities │
│         WHERE id = 'entity-1'           │
│                                         │
│  Result: table_name = "customers"      │
│          schema_name = "public"         │
│                                         │
└────────────┬──────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────────────────────────────┐
│  Step 2: Query All Foreign Keys (Inbound + Outbound)          │
│  ─────────────────────────────────────────────────────────────│
│                                                                 │
│  Query: SELECT * FROM catalog_edge ce                         │
│         WHERE (source.node_name = 'customers' OR              │
│                target.node_name = 'customers')                │
│         AND ce.relationship_type = 'foreign_key'             │
│                                                                 │
│  Result Set:                                                   │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │ Edge 1: customers → accounts (outbound)                  │ │
│  │ Edge 2: orders → customers (inbound)                    │ │
│  │ Edge 3: interactions → customers (inbound)              │ │
│  └──────────────────────────────────────────────────────────┘ │
│                                                                 │
└────────────┬──────────────────────────────────────────────────┘
             │
             ├─────────────────────────────────────┬────────────────┐
             │                                     │                │
             ▼                                     ▼                ▼
   ┌──────────────────┐              ┌──────────────────┐     ┌──────────────┐
   │ Edge 1: Outbound │              │ Edge 2: Inbound  │     │ Edge 3: In.  │
   │ customers →      │              │ orders →         │     │ interactions │
   │ accounts         │              │ customers        │     │ → customers  │
   │ (many-to-one)    │              │ (one-to-many)    │     │ (one-many)   │
   └────────┬─────────┘              └────────┬─────────┘     └───────┬──────┘
            │                                 │                       │
            ▼                                 ▼                       ▼
    ┌──────────────────┐          ┌─────────────────┐         ┌─────────────┐
    │ Find Entity for  │          │ Find Entity for │         │ Find Entity │
    │ Target: accounts │          │ Source: orders  │         │ Source:     │
    └────────┬─────────┘          └────────┬────────┘         │ interactions│
             │                             │                  └─────┬───────┘
             ▼                             ▼                        ▼
    Found: Entity 2                Found: Entity 3           Found: Entity 4
    Name: "Account"                Name: "Order"             Name: "Interaction"
    │                               │                         │
    ▼                               ▼                         ▼
┌─────────────────────────┐  ┌──────────────────┐    ┌────────────────────┐
│ Relationship Pair 1     │  │ Relationship 2   │    │ Relationship 3     │
├─────────────────────────┤  ├──────────────────┤    ├────────────────────┤
│ Source: Customer        │  │ Source: Customer │    │ Source: Customer   │
│ Target: Account         │  │ Target: Order    │    │ Target: Interaction│
│ Type: reference         │  │ Type: composition│    │ Type: association  │
│ Cardinality: m:1        │  │ Cardinality: 1:m │    │ Cardinality: 1:m   │
│ Confidence: 1.0         │  │ Confidence: 1.0  │    │ Confidence: 1.0    │
│ DiscoveryCode: fk_out   │  │ DiscoveryCode: fk│    │ DiscoveryCode: fk  │
└─────────────────────────┘  │ _in              │    │ _in                │
                             └──────────────────┘    └────────────────────┘
             │                        │                       │
             └────────────┬───────────┴───────────────────────┘
                          │
                          ▼
                   ┌──────────────────┐
                   │  OUTPUT:         │
                   │ []EntityRelation-│
                   │  shipFromFK with │
                   │  3 relationships │
                   └──────────────────┘
```

## Data Flow: Creating Relationships

```
┌─ API Endpoint Called ──────────────────────────┐
│ POST /entities/entity-1/discover-and-link     │
│ Request Body: (optional filters)              │
└────────────┬─────────────────────────────────┘
             │
             ▼
┌─ FK Discovery ────────────────────────────────────────┐
│ 1. Get entity backing table(s)                       │
│ 2. Query FK edges from catalog_edge                 │
│ 3. For each FK:                                      │
│    - Determine direction & cardinality              │
│    - Find target entity                             │
│    - Create EntityRelationshipFromFK struct         │
│                                                      │
│ Result: []EntityRelationshipFromFK                  │
└────────────┬─────────────────────────────────────┘
             │
             ▼
┌─ For Each Relationship ────────────────────────┐
│                                                │
│ Build EdgeProperties JSON:                    │
│ {                                              │
│   "discovery_method": "foreign_key_analysis", │
│   "source_table": "customers",                │
│   "target_table": "accounts",                 │
│   "fk_edge_id": "...",                       │
│   "cardinality": "many-to-one",              │
│   "relation_type": "reference",              │
│   "discovered_at": "2025-10-25T..."          │
│ }                                              │
│                                                │
│ INSERT INTO catalog_edge (                   │
│   source_node_id = target_entity_id,         │
│   target_node_id = source_entity_id,         │
│   relationship_type = "entity_relationship", │
│   properties = EdgeProperties                │
│ )                                              │
│                                                │
└────────────┬─────────────────────────────────┘
             │
             ▼
        ┌──────────┐
        │ Edge ID  │
        └──────────┘
             │
             ▼ (for each)
        ┌─────────────┐
        │ Aggregate & │
        │ Return      │
        └─────────────┘
             │
             ▼
┌─ API Response ──────────────────────┐
│ {                                   │
│   "entity_id": "entity-1",          │
│   "discovered": 3,                  │
│   "created_edges": 3,               │
│   "failed": 0,                      │
│   "failures": []                    │
│ }                                   │
└─────────────────────────────────────┘
```

## Cardinality Decision Tree

```
                        Foreign Key Found
                              │
                              ▼
                      ┌─ Direction? ─┐
                      │              │
              ┌───────┘              └────────┐
              │                               │
              ▼                               ▼
        OUTBOUND FK                    INBOUND FK
      (Source references)        (Source is referenced)
              │                               │
              ▼                               ▼
      customers.account_id        orders.customer_id
      REFERENCES accounts.id      REFERENCES customers.id
              │                               │
              ▼                               ▼
    ┌──────────────────────┐      ┌──────────────────────┐
    │ Cardinality:         │      │ Cardinality:         │
    │ Many-to-One          │      │ One-to-Many          │
    │ (Many customers have │      │ (One customer has    │
    │  one account)        │      │  many orders)        │
    └──────────┬───────────┘      └──────────┬───────────┘
               │                             │
               ▼                             ▼
       ┌────────────────────┐       ┌────────────────────┐
       │ Relation Type:     │       │ Relation Type:     │
       │ REFERENCE          │       │ COMPOSITION        │
       │                    │       │                    │
       │ (Customer refs     │       │ (Customer owns     │
       │  Account, could    │       │  Orders, likely    │
       │  exist elsewhere)  │       │  exclusive)        │
       └────────────────────┘       └────────────────────┘
```

## Example: Customer Entity Discovery

```
INPUT: Entity "Customer" (backed by customers table)
        ├─ tenant: "retail-tenant"
        └─ datasource: "erp-system"

STEP 1: Query FK edges involving "customers" table
────────────────────────────────────────────────────

  customers.account_id → accounts.id (outbound)
  customers.branch_id → branches.id (outbound)  
  orders.customer_id → customers.id (inbound)
  interactions.customer_id → customers.id (inbound)
  payments.customer_id → customers.id (inbound)

STEP 2: Determine direction & cardinality for each
────────────────────────────────────────────────────

  Edge 1: Outbound, Many-to-One
  Edge 2: Outbound, Many-to-One
  Edge 3: Inbound, One-to-Many
  Edge 4: Inbound, One-to-Many
  Edge 5: Inbound, One-to-Many

STEP 3: Find target entities
───────────────────────────────

  Edge 1 → accounts table → Account entity ✓
  Edge 2 → branches table → Branch entity ✓
  Edge 3 → orders table → Order entity ✓
  Edge 4 → interactions table → Interaction entity ✓
  Edge 5 → payments table → Payment entity ✓

STEP 4: Create relationships
──────────────────────────────

  Customer ─[references]→ Account (m:1)
  Customer ─[references]→ Branch (m:1)
  Customer ←[owns]─ Order (1:m)
  Customer ←[has]─ Interaction (1:m)
  Customer ←[receives]─ Payment (1:m)

STEP 5: Store as edges in catalog_edge
─────────────────────────────────────────

  5 new edges created with:
  - relationship_type: "entity_relationship_fk"
  - properties: { discovery_method, fk_details, cardinality, etc. }

OUTPUT: 5 discovered relationships ready for visualization
─────────────────────────────────────────────────────────
```

## Integration Points

```
┌────────────────────────────────────────────────────────┐
│              YOUR EXISTING CODE                        │
├────────────────────────────────────────────────────────┤
│                                                        │
│  ┌─ RelationshipService ────────────────────────────┐ │
│  │                                                   │ │
│  │  GetRelationshipSuggestions() ◄─ [EXISTING]     │ │
│  │  └─ Semantic similarity based                   │ │
│  │                                                   │ │
│  │  GetRelationshipSuggestionsWithFK() ◄─ [NEW]    │ │
│  │  └─ Combines semantic + FK-based                │ │
│  │                                                   │ │
│  └─┬─────────────────────────────────────────────┘ │
│    │                                                │
│    ▼ Calls ────────────────────────┐                │
│                            ┌──────────────────────┐ │
│                            │ ForeignKeyDiscovery  │ │
│                            │ Engine [NEW CODE]    │ │
│                            └──────────────────────┘ │
│                                                      │
└────────────────────────────────────────────────────────┘
                              │
                              ▼
                    ┌─────────────────────┐
                    │  Your Database      │
                    │  catalog_edge       │
                    │  entities           │
                    │  catalog_node       │
                    └─────────────────────┘
```

---

## Summary

The FK discovery system works in layers:

1. **Database Layer**: Stores FK relationships in `catalog_edge` with properties
2. **API Layer**: `ForeignKeyDiscoveryEngine` queries and interprets FK data
3. **Service Layer**: `RelationshipService` integrates FK discovery with semantic analysis
4. **HTTP Layer**: Exposes discovery via REST endpoints
5. **Frontend Layer**: Displays discovered relationships with confidence scores

All layers work together to provide **automatic entity relationship discovery** from database foreign keys.
