# Entity Relationship Discovery via Foreign Keys

## Overview

This guide documents how to discover relationships between **business entities** by analyzing the **foreign keys (FKs)** in the database tables that back each entity.

### Core Concept

```
Business Entity                Database Table              Foreign Key Relationship
┌─────────────────┐           ┌──────────────┐           ┌──────────────────┐
│  Customer       │──backs──→ │  customers   │─ has FK ─→│  accounts        │
│  (Entity)       │           │  (Table)     │           │  (Target Table)  │
└─────────────────┘           └──────────────┘           └──────────────────┘
                                      │
                                 stored in edge properties
                                      ↓
                    ┌──────────────────────────────┐
                    │  FK Details (edge.properties)│
                    │  - fk_table: "customers"     │
                    │  - fk_column: "account_id"   │
                    │  - target_table: "accounts"  │
                    │  - target_column: "id"       │
                    │  - cardinality: "many-to-one"│
                    └──────────────────────────────┘
                                      ↓
                    If there's an entity backed by
                    "accounts" table, we've found a
                    relationship: Customer → Account
```

## Architecture

### 1. Current State: Table-to-Table Foreign Keys

The `catalog_edge` table in your database already stores foreign key relationships:

```sql
CREATE TABLE catalog_edge (
    id UUID PRIMARY KEY,
    source_node_id UUID,           -- FK: source table
    target_node_id UUID,           -- FK: target table
    relationship_type VARCHAR(100), -- 'foreign_key', etc.
    properties JSONB,              -- ← Contains FK details
    -- ...
);
```

**Edge properties structure** (stored as JSONB):

```json
{
  "foreign_key_constraints": ["fk_customers_accounts"],
  "foreign_key_target_table": "public.accounts",
  "foreign_key_target_column": "id",
  "source_column": "account_id",
  "columns": [
    {
      "source_column": "account_id",
      "target_column": "id"
    }
  ]
}
```

### 2. Entity-to-Table Mapping

Entities are backed by tables via a mapping (conceptual):

```
Entity Model                 Database Mapping
┌──────────────┐            ┌─────────────────────────┐
│ Entity: {    │     maps    │ schema: "public"        │
│   id: "e1",  │     ←───→   │ table: "customers"      │
│   name: "...",            │ entity_id: "e1"        │
│ }            │            │ tenant_id: "t1"        │
└──────────────┘            └─────────────────────────┘
```

This mapping typically exists as:
- Properties on the entity object
- A separate `entity_table_mapping` table
- Configuration in the bundle schema

## Implementation Strategy

### Phase 1: Build FK Discovery Engine

**Goal**: Given an entity, find all FKs (inbound and outbound) from its backing table(s).

#### 1.1 Core Data Structures

```go
// ForeignKeyRelationship represents a single FK relationship
type ForeignKeyRelationship struct {
    // Source information
    SourceTable  string `json:"source_table"`   // e.g., "customers"
    SourceColumn string `json:"source_column"`  // e.g., "account_id"
    
    // Target information
    TargetTable  string `json:"target_table"`   // e.g., "accounts"
    TargetColumn string `json:"target_column"`  // e.g., "id"
    
    // Relationship metadata
    Cardinality string `json:"cardinality"`     // "many-to-one", "one-to-many"
    Constraint  string `json:"constraint_name"` // FK name
    
    // Catalog edge reference
    EdgeID      string `json:"edge_id"`        // catalog_edge.id
    Properties  map[string]interface{} `json:"properties"` // Full edge properties
}

// EntityBackingTable represents a table that backs an entity
type EntityBackingTable struct {
    EntityID   string
    EntityName string
    TableName  string
    SchemaName string
}

// EntityRelationshipPair represents a discovered relationship
type EntityRelationshipPair struct {
    SourceEntity  string                      `json:"source_entity"`
    TargetEntity  string                      `json:"target_entity"`
    ForeignKey    ForeignKeyRelationship     `json:"foreign_key"`
    Confidence    float64                     `json:"confidence"`    // 0.0-1.0
    RelationType  string                      `json:"relation_type"` // "composition", "reference"
}
```

#### 1.2 FK Discovery Queries

**Query 1: Get outbound FKs from a table**

```sql
SELECT
    ce.id as edge_id,
    source_table.node_name as source_table,
    target_table.node_name as target_table,
    ce.properties,
    ce.relationship_type
FROM catalog_edge ce
JOIN catalog_node source_table ON ce.source_node_id = source_table.id
JOIN catalog_node target_table ON ce.target_node_id = target_table.id
WHERE ce.relationship_type = 'foreign_key'
  AND source_table.node_name = $1            -- input table name
  AND ce.tenant_datasource_id = $2           -- tenant_datasource_id
  AND source_table.node_type_id = (
      SELECT id FROM node_type WHERE name = 'table'
  );
```

**Query 2: Get inbound FKs to a table**

```sql
SELECT
    ce.id as edge_id,
    source_table.node_name as source_table,
    target_table.node_name as target_table,
    ce.properties,
    ce.relationship_type
FROM catalog_edge ce
JOIN catalog_node source_table ON ce.source_node_id = source_table.id
JOIN catalog_node target_table ON ce.target_node_id = target_table.id
WHERE ce.relationship_type = 'foreign_key'
  AND target_table.node_name = $1            -- input table name
  AND ce.tenant_datasource_id = $2           -- tenant_datasource_id
  AND target_table.node_type_id = (
      SELECT id FROM node_type WHERE name = 'table'
  );
```

**Query 3: Get all FKs with edge properties (combined)**

```sql
WITH table_fks AS (
    -- Outbound FKs
    SELECT
        ce.id,
        source_table.node_name as source_table,
        target_table.node_name as target_table,
        'outbound' as direction,
        ce.properties
    FROM catalog_edge ce
    JOIN catalog_node source_table ON ce.source_node_id = source_table.id
    JOIN catalog_node target_table ON ce.target_node_id = target_table.id
    WHERE source_table.node_name = $1
      AND ce.relationship_type = 'foreign_key'
      AND ce.tenant_datasource_id = $2
    
    UNION ALL
    
    -- Inbound FKs
    SELECT
        ce.id,
        source_table.node_name as source_table,
        target_table.node_name as target_table,
        'inbound' as direction,
        ce.properties
    FROM catalog_edge ce
    JOIN catalog_node source_table ON ce.source_node_id = source_table.id
    JOIN catalog_node target_table ON ce.target_node_id = target_table.id
    WHERE target_table.node_name = $1
      AND ce.relationship_type = 'foreign_key'
      AND ce.tenant_datasource_id = $2
)
SELECT * FROM table_fks ORDER BY direction;
```

### Phase 2: Entity-to-Entity Relationship Mapper

**Goal**: Map entity FKs to relationships between other entities.

#### 2.1 Implementation Steps

```
Step 1: Get the Entity's Backing Table(s)
  Input: entityID
  Output: [{ table: "customers", schema: "public" }]
  
Step 2: Query All FKs from/to This Table
  Input: table_name = "customers"
  Output: [
    { source: "customers", target: "accounts", direction: "outbound" },
    { source: "orders", target: "customers", direction: "inbound" }
  ]
  
Step 3: For Each FK, Find Target Entity
  For each FK result:
    - Extract target/source table name
    - Query: which entity is backed by this table?
    - If found: create EntityRelationshipPair
    
Step 4: Enrich with Relationship Type
  Based on cardinality and domain knowledge:
    - Many-to-One FK → "reference" (Customer references Account)
    - One-to-Many FK → "composition" (Customer has Orders)
    - 1:1 FK → "association"
```

#### 2.2 Algorithm: FK to Entity Relationship

```go
func DiscoverEntityRelationshipsFromFK(
    ctx context.Context,
    db *sql.DB,
    tenantID, datasourceID string,
    entity *Entity,  // with backing table info
) ([]EntityRelationshipPair, error) {
    // Step 1: Get the backing table
    backingTable := entity.GetBackingTable() // "customers"
    if backingTable == "" {
        return nil, fmt.Errorf("entity has no backing table")
    }
    
    // Step 2: Query all FKs (outbound + inbound)
    fks, err := queryAllForeignKeys(ctx, db, tenantID, datasourceID, backingTable)
    if err != nil {
        return nil, err
    }
    
    var relationships []EntityRelationshipPair
    
    // Step 3: For each FK, find the target entity
    for _, fk := range fks {
        var targetTableName string
        
        if fk.Direction == "outbound" {
            // FK points FROM customers TO accounts
            targetTableName = fk.TargetTable
        } else {
            // FK points FROM orders TO customers
            targetTableName = fk.SourceTable
        }
        
        // Find which entity is backed by targetTableName
        targetEntity, err := findEntityByBackingTable(ctx, db, tenantID, datasourceID, targetTableName)
        if err != nil {
            logging.GetLogger().Sugar().Warnf(
                "Could not find entity for table %s: %v", targetTableName, err,
            )
            continue
        }
        
        // Step 4: Determine relationship type
        relType := inferRelationshipType(fk.Direction, fk.Cardinality)
        
        pair := EntityRelationshipPair{
            SourceEntity: entity.ID,
            TargetEntity: targetEntity.ID,
            ForeignKey:   fk,
            Confidence:   1.0,  // FK is definitive
            RelationType: relType,
        }
        
        relationships = append(relationships, pair)
    }
    
    return relationships, nil
}
```

### Phase 3: Storage in Relationship Graph

**Goal**: Persist discovered relationships as edges in `catalog_edge`.

#### 3.1 Creating Entity Relationship Edges

```go
// CreateEntityRelationshipEdge creates an edge between two entities
// based on FK discovery
func CreateEntityRelationshipEdge(
    ctx context.Context,
    db *sql.DB,
    pair EntityRelationshipPair,
    tenantID, datasourceID string,
) error {
    edgeID := uuid.New()
    edgeTypeID := getEdgeTypeIDFor("entity_to_entity")
    
    properties := map[string]interface{}{
        "discovery_method": "foreign_key_analysis",
        "source_table":     pair.ForeignKey.SourceTable,
        "source_column":    pair.ForeignKey.SourceColumn,
        "target_table":     pair.ForeignKey.TargetTable,
        "target_column":    pair.ForeignKey.TargetColumn,
        "fk_constraint":    pair.ForeignKey.Constraint,
        "fk_edge_id":       pair.ForeignKey.EdgeID,  // Link back to table-level FK
        "cardinality":      pair.ForeignKey.Cardinality,
        "relation_type":    pair.RelationType,
    }
    
    propsJSON, _ := json.Marshal(properties)
    
    query := `
        INSERT INTO catalog_edge (
            id, tenant_id, tenant_datasource_id, source_node_id, target_node_id,
            edge_type_id, relationship_type, properties, created_at, updated_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
        ON CONFLICT (tenant_datasource_id, source_node_id, edge_type_id, target_node_id)
        DO UPDATE SET
            relationship_type = EXCLUDED.relationship_type,
            properties = EXCLUDED.properties,
            updated_at = EXCLUDED.updated_at
    `
    
    _, err := db.ExecContext(ctx, query,
        edgeID,
        tenantID,
        datasourceID,
        pair.SourceEntity,
        pair.TargetEntity,
        edgeTypeID,
        "entity_relationship",
        propsJSON,
        time.Now(),
        time.Now(),
    )
    
    return err
}
```

#### 3.2 Edge Properties Schema

When you store an entity-to-entity relationship edge, include:

```json
{
  "discovery_method": "foreign_key_analysis",
  "source_table": "customers",
  "source_column": "account_id",
  "target_table": "accounts",
  "target_column": "id",
  "fk_constraint": "fk_customers_account_id",
  "fk_edge_id": "<uuid-of-table-level-fk-edge>",
  "cardinality": "many-to-one",
  "relation_type": "reference",
  "confidence": 1.0,
  "discovered_at": "2025-10-25T10:30:00Z"
}
```

## Integration Points

### 1. In Your `RelationshipService`

Add a new method to discover FK-based relationships:

```go
// In relationship_suggestions.go

type FKDiscoveryEngine struct {
    db *sql.DB
}

func (s *FKDiscoveryEngine) DiscoverForeignKeyRelationships(
    ctx context.Context,
    tenantID, datasourceID, entity string,
) ([]RelationshipSuggestion, error) {
    // 1. Get entity backing table
    backingTable, err := s.getEntityBackingTable(ctx, tenantID, datasourceID, entity)
    if err != nil {
        return nil, err
    }
    
    // 2. Query FKs
    fks, err := s.queryForeignKeysForTable(ctx, backingTable)
    if err != nil {
        return nil, err
    }
    
    var suggestions []RelationshipSuggestion
    
    // 3. Convert FKs to suggestions
    for _, fk := range fks {
        targetEntity, err := s.findEntityByTable(ctx, tenantID, datasourceID, fk.TargetTable)
        if err != nil {
            continue
        }
        
        suggestion := RelationshipSuggestion{
            ID:           uuid.New().String(),
            SourceEntity: entity,
            TargetEntity: targetEntity,
            EdgeType:     RelationshipEdgeTypeForeignKey,
            Cardinality:  fk.Cardinality,
            Confidence:   1.0,  // FKs are definitive
            Reasoning:    fmt.Sprintf(
                "Foreign key %s.%s → %s.%s",
                fk.SourceTable, fk.SourceColumn,
                fk.TargetTable, fk.TargetColumn,
            ),
        }
        suggestions = append(suggestions, suggestion)
    }
    
    return suggestions, nil
}
```

### 2. In Your Entity Definition Service

Automatically populate relationships when entities are created:

```go
// In entities_routes.go

func CreateEntityWithRelationships(
    ctx context.Context,
    db *sql.DB,
    entity *EntityDefinition,
    tenantID, datasourceID string,
) error {
    // 1. Create entity
    err := createEntity(ctx, db, entity)
    if err != nil {
        return err
    }
    
    // 2. Discover FK-based relationships
    fkEngine := &FKDiscoveryEngine{db: db}
    relationships, err := fkEngine.DiscoverForeignKeyRelationships(
        ctx, tenantID, datasourceID, entity.Name,
    )
    if err != nil {
        logging.GetLogger().Sugar().Warnf(
            "Failed to discover FK relationships for entity %s: %v",
            entity.Name, err,
        )
        // Continue without FK relationships
    }
    
    // 3. Create edges for each discovered relationship
    for _, rel := range relationships {
        err = createEntityRelationshipEdge(ctx, db, rel, tenantID, datasourceID)
        if err != nil {
            logging.GetLogger().Sugar().Warnf(
                "Failed to create relationship edge: %v", err,
            )
        }
    }
    
    return nil
}
```

### 3. In Your GraphQL Schema

Add queries for FK-discovered relationships:

```graphql
type Query {
  # Discover entity relationships via FK analysis
  discoverEntityRelationships(
    tenantId: ID!
    datasourceId: ID!
    entityId: ID!
  ): [EntityRelationshipPair!]!
  
  # Get FK details for an entity
  getEntityForeignKeys(
    tenantId: ID!
    datasourceId: ID!
    entityId: ID!
  ): EntityForeignKeyDetails!
}

type EntityForeignKeyDetails {
  entity: Entity!
  backingTable: String!
  outboundForeignKeys: [ForeignKeyRelationship!]!
  inboundForeignKeys: [ForeignKeyRelationship!]!
  discoveredRelationships: [EntityRelationshipPair!]!
}

type ForeignKeyRelationship {
  id: ID!
  sourceTable: String!
  sourceColumn: String!
  targetTable: String!
  targetColumn: String!
  cardinality: String!
  constraintName: String!
}
```

## Cardinality Detection

### Rules for Inferring Cardinality

Based on the FK direction and table structure:

```go
func InferCardinality(fk ForeignKeyRelationship) string {
    // Outbound FK = Many-to-One
    // (Many customers have One account)
    if fk.Direction == "outbound" {
        return "many-to-one"
    }
    
    // Inbound FK = One-to-Many
    // (One customer has Many orders)
    if fk.Direction == "inbound" {
        return "one-to-many"
    }
    
    // Check for unique constraint on FK columns (1:1)
    if fk.IsUniqueOnSourceColumns {
        if fk.Direction == "outbound" {
            return "one-to-one"
        }
    }
    
    return "unknown"
}
```

## Relationship Type Classification

### Inferring Relationship Semantics

```go
func InferRelationshipType(
    fk ForeignKeyRelationship,
    cardinality string,
) string {
    // Many-to-One FK = Reference
    // (Customer references Account)
    if cardinality == "many-to-one" {
        return "reference"
    }
    
    // One-to-Many FK = Composition
    // (Customer has/owns Orders)
    if cardinality == "one-to-many" {
        // Heuristic: if target table has only this FK as unique key,
        // it's likely a composition
        if isCompositionCandidate(fk) {
            return "composition"
        }
        return "association"
    }
    
    // One-to-One FK = Association
    if cardinality == "one-to-one" {
        return "association"
    }
    
    return "unknown"
}

func isCompositionCandidate(fk ForeignKeyRelationship) bool {
    // Heuristics:
    // 1. Target table has no other FKs
    // 2. Target table name suggests it's a child
    // 3. Target table is small/audit-only
    
    targetTableName := fk.TargetTable
    sourceTableName := fk.SourceTable
    
    // Name heuristic: if target has source as prefix or "_item" suffix
    if strings.HasPrefix(targetTableName, sourceTableName) {
        return true
    }
    if strings.HasSuffix(targetTableName, "_item") ||
       strings.HasSuffix(targetTableName, "_detail") ||
       strings.HasSuffix(targetTableName, "_line") {
        return true
    }
    
    return false
}
```

## Usage Examples

### Example 1: Discover Customer Entity Relationships

```
Customer Entity
├─ Backing Table: customers
├─ Outbound FK: customers.account_id → accounts.id
│  └─ Discovered Relationship: Customer → Account (many-to-one)
└─ Inbound FK: orders.customer_id → customers.id
   └─ Discovered Relationship: Customer ← Order (one-to-many)

Result Edges:
- Customer --[references]--> Account
- Customer --[owns]--> Order
```

### Example 2: Discovering Complex Relationships

```
Transaction Entity
├─ Backing Table: transactions
├─ Outbound FKs:
│  ├─ transactions.account_id → accounts.id (many-to-one)
│  ├─ transactions.counterparty_id → counterparties.id (many-to-one)
│  └─ transactions.product_id → products.id (many-to-one)
└─ Inbound FKs:
   └─ transaction_fees.transaction_id → transactions.id (one-to-many)

Result Relationships:
- Transaction [references] Account
- Transaction [references] Counterparty
- Transaction [references] Product
- Transaction [owns] TransactionFee
```

## Advanced: Multi-Table Entities

For entities backed by **multiple tables** (via joins or unions):

```go
// If an entity is composed of multiple tables
type MultiTableEntity struct {
    PrimaryTable    string   // "customers"
    JoinedTables    []string // ["customer_profiles", "customer_preferences"]
}

func DiscoverRelationshipsForMultiTableEntity(
    ctx context.Context,
    db *sql.DB,
    entity *MultiTableEntity,
) ([]EntityRelationshipPair, error) {
    var allRelationships []EntityRelationshipPair
    
    // Discover relationships from PRIMARY TABLE
    primaryRels, err := discoverFromTable(ctx, db, entity.PrimaryTable)
    if err != nil {
        return nil, err
    }
    allRelationships = append(allRelationships, primaryRels...)
    
    // Also discover from JOINED TABLES (for completeness)
    for _, joinedTable := range entity.JoinedTables {
        joinedRels, err := discoverFromTable(ctx, db, joinedTable)
        if err != nil {
            logging.GetLogger().Sugar().Warnf(
                "Failed to discover from joined table %s: %v", joinedTable, err,
            )
            continue
        }
        
        // Only include relationships NOT already found in primary table
        for _, rel := range joinedRels {
            if !relationshipExists(allRelationships, rel) {
                allRelationships = append(allRelationships, rel)
            }
        }
    }
    
    return allRelationships, nil
}
```

## Performance Considerations

### Query Optimization

1. **Index the FK edges**: Ensure `catalog_edge` has indexes on:
   - `(source_node_id, relationship_type, tenant_datasource_id)`
   - `(target_node_id, relationship_type, tenant_datasource_id)`

2. **Cache FK results**: FK relationships are relatively static
   ```go
   const FKCacheTTL = 1 * time.Hour
   ```

3. **Batch queries**: If discovering for multiple entities:
   ```go
   // Instead of N queries, use 1 query with IN clause
   WHERE source_table IN ($1, $2, $3, ...)
   ```

## Handling Edge Cases

### 1. Circular References

```
Customer ←→ Account
(Customer.account_id → Account.id AND Account.primary_customer_id → Customer.id)
```

**Solution**: Mark as bidirectional but prevent infinite recursion in UI.

### 2. Self-Referential FKs

```
Employee.manager_id → Employee.id
```

**Solution**: Handle specially in relationship discovery.

```go
if fk.SourceTable == fk.TargetTable {
    relType.SelfReferential = true
    relType.Label = "Hierarchical (Self-Reference)"
}
```

### 3. Missing Target Entities

If a table has a FK but no corresponding entity:

```go
// Option 1: Create a placeholder entity
// Option 2: Mark as "external" relationship
// Option 3: Skip with warning
```

## Validation & Testing

### Test Cases

```go
func TestFKDiscovery(t *testing.T) {
    tests := []struct{
        name           string
        sourceTable    string
        expectedTarget []string
    }{
        {
            name: "Customer to Account",
            sourceTable: "customers",
            expectedTarget: []string{"accounts"},
        },
        {
            name: "Transaction to Multiple Targets",
            sourceTable: "transactions",
            expectedTarget: []string{"accounts", "counterparties", "products"},
        },
    }
}
```

## Summary

**Key Takeaways:**

1. **FKs are definitive**: A FK relationship has confidence = 1.0
2. **Cardinality drives semantics**: Many-to-One = reference, One-to-Many = composition
3. **Properties bridge levels**: Edge properties link entity relationships to table-level FKs
4. **Multi-source discovery**: Combine FK analysis with semantic similarity for richer suggestions
5. **Performance matters**: Cache FK results and use batch queries

**Next Steps:**

- [ ] Implement `FKDiscoveryEngine` in Go
- [ ] Add queries to discover entity backing tables
- [ ] Create edges for discovered relationships
- [ ] Update relationship suggestions endpoint
- [ ] Add GraphQL mutations for FK-based discovery
- [ ] Test with sample data (Customers → Accounts → Products)
