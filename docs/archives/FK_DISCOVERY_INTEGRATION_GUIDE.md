# Foreign Key Discovery Engine - Integration Guide

## Quick Start

### 1. Add FK Discovery to Your Relationship Service

In your `relationship_suggestions.go` file, add the FK discovery method:

```go
// Add to RelationshipService struct
type RelationshipService struct {
    db *sql.DB
    fkEngine *ForeignKeyDiscoveryEngine  // ← Add this
}

// Update the constructor
func NewRelationshipService(db *sql.DB) *RelationshipService {
    return &RelationshipService{
        db: db,
        fkEngine: NewForeignKeyDiscoveryEngine(db),  // ← Initialize
    }
}

// Add this new method to include FK-based suggestions
func (s *RelationshipService) GetRelationshipSuggestionsWithFK(
    ctx context.Context,
    tenantID, datasourceID, entity string,
    limit int,
) ([]RelationshipSuggestion, error) {
    
    // 1. Get semantic/similarity-based suggestions (existing)
    semanticSuggestions, err := s.GetRelationshipSuggestions(ctx, tenantID, datasourceID, entity, limit)
    if err != nil {
        logging.GetLogger().Sugar().Warnf("Failed to get semantic suggestions: %v", err)
        semanticSuggestions = []RelationshipSuggestion{}
    }
    
    // 2. Get FK-based suggestions (NEW)
    sourceEntity := &Entity{ID: entity, Name: entity}
    fkRelationships, err := s.fkEngine.DiscoverEntityRelationshipsFromFK(ctx, tenantID, datasourceID, sourceEntity)
    if err != nil {
        logging.GetLogger().Sugar().Warnf("Failed to discover FK relationships: %v", err)
        fkRelationships = []EntityRelationshipFromFK{}
    }
    
    // 3. Convert FK relationships to suggestions
    fkSuggestions := s.convertFKToSuggestions(fkRelationships)
    
    // 4. Merge and deduplicate
    allSuggestions := s.mergeSuggestions(semanticSuggestions, fkSuggestions)
    
    // 5. Return top N
    if len(allSuggestions) > limit {
        return allSuggestions[:limit], nil
    }
    return allSuggestions, nil
}

// Helper to convert FK relationships to suggestions
func (s *RelationshipService) convertFKToSuggestions(
    fkRels []EntityRelationshipFromFK,
) []RelationshipSuggestion {
    var suggestions []RelationshipSuggestion
    
    for _, rel := range fkRels {
        suggestion := RelationshipSuggestion{
            ID:           uuid.New().String(),
            SourceEntity: rel.SourceEntityName,
            TargetEntity: rel.TargetEntityName,
            EdgeType:     RelationshipEdgeTypeForeignKey,
            Cardinality:  rel.Cardinality,
            Confidence:   1.0,  // FKs are definitive
            Reasoning: fmt.Sprintf(
                "Foreign Key: %s.%s → %s.%s (discovered via DB schema analysis)",
                rel.ForeignKey.SourceTable,
                rel.ForeignKey.Columns[0].SourceColumn,
                rel.ForeignKey.TargetTable,
                rel.ForeignKey.Columns[0].TargetColumn,
            ),
            Dismissible: false,  // Don't allow dismissal of schema-level relationships
        }
        suggestions = append(suggestions, suggestion)
    }
    
    return suggestions
}

// Helper to merge and deduplicate suggestions
func (s *RelationshipService) mergeSuggestions(
    semantic, fkBased []RelationshipSuggestion,
) []RelationshipSuggestion {
    // Create a map to track unique suggestions
    seen := make(map[string]RelationshipSuggestion)
    
    // Add semantic suggestions first (they're more frequent)
    for _, s := range semantic {
        key := fmt.Sprintf("%s->%s", s.SourceEntity, s.TargetEntity)
        if _, exists := seen[key]; !exists {
            seen[key] = s
        }
    }
    
    // Add FK suggestions (prefer FK over semantic for definitive FKs)
    for _, s := range fkBased {
        key := fmt.Sprintf("%s->%s", s.SourceEntity, s.TargetEntity)
        // FK suggestions override semantic ones (confidence 1.0 wins)
        seen[key] = s
    }
    
    // Convert map back to slice
    var result []RelationshipSuggestion
    for _, s := range seen {
        result = append(result, s)
    }
    
    // Sort by confidence descending
    sort.Slice(result, func(i, j int) bool {
        if result[i].Confidence != result[j].Confidence {
            return result[i].Confidence > result[j].Confidence
        }
        return result[i].Title < result[j].Title
    })
    
    return result
}
```

### 2. Create an API Endpoint for FK Discovery

Add this route to your `api.go`:

```go
// In SetupRouter function, add:

r.Get("/entities/{entityId}/foreign-keys", func(w http.ResponseWriter, r *http.Request) {
    entityID := chi.URLParam(r, "entityId")
    tenantID := r.URL.Query().Get("tenant_id")
    datasourceID := r.URL.Query().Get("datasource_id")
    
    if tenantID == "" || datasourceID == "" {
        writeJSONError(w, http.StatusBadRequest, "tenant_id and datasource_id required", "", "")
        return
    }
    
    engine := NewForeignKeyDiscoveryEngine(db)
    
    // Get entity details
    entity := &Entity{ID: entityID}  // Load actual entity from DB
    
    // Discover relationships
    relationships, err := engine.DiscoverEntityRelationshipsFromFK(r.Context(), tenantID, datasourceID, entity)
    if err != nil {
        writeJSONError(w, http.StatusInternalServerError, err.Error(), "discovery_error", "")
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "entity_id": entityID,
        "relationships": relationships,
        "count": len(relationships),
    })
})

// Also add an endpoint to auto-create edges from FK discovery
r.Post("/entities/{entityId}/discover-and-link-relationships", func(w http.ResponseWriter, r *http.Request) {
    entityID := chi.URLParam(r, "entityId")
    tenantID := r.URL.Query().Get("tenant_id")
    datasourceID := r.URL.Query().Get("datasource_id")
    
    if tenantID == "" || datasourceID == "" {
        writeJSONError(w, http.StatusBadRequest, "tenant_id and datasource_id required", "", "")
        return
    }
    
    engine := NewForeignKeyDiscoveryEngine(db)
    
    // Load entity
    entity := &Entity{ID: entityID}
    
    // Discover relationships
    relationships, err := engine.DiscoverEntityRelationshipsFromFK(r.Context(), tenantID, datasourceID, entity)
    if err != nil {
        writeJSONError(w, http.StatusInternalServerError, err.Error(), "discovery_error", "")
        return
    }
    
    // Create edges for each discovered relationship
    var createdEdges []string
    var failedCreations []string
    
    for _, rel := range relationships {
        edgeID, err := engine.CreateEntityRelationshipEdgeFromFK(r.Context(), tenantID, datasourceID, rel)
        if err != nil {
            failedCreations = append(failedCreations, fmt.Sprintf("%s: %v", rel.TargetEntityName, err))
        } else {
            createdEdges = append(createdEdges, edgeID)
        }
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "entity_id": entityID,
        "discovered": len(relationships),
        "created_edges": len(createdEdges),
        "failed": len(failedCreations),
        "failures": failedCreations,
    })
})
```

### 3. Update GraphQL Schema (if using)

Add to your GraphQL schema:

```graphql
type Query {
  # Discover foreign key relationships for an entity
  discoverEntityForeignKeyRelationships(
    tenantId: ID!
    datasourceId: ID!
    entityId: ID!
  ): ForeignKeyDiscoveryResult!
}

type Mutation {
  # Auto-create relationship edges from FK discovery
  createRelationshipEdgesFromForeignKeys(
    tenantId: ID!
    datasourceId: ID!
    entityId: ID!
  ): ForeignKeyCreationResult!
}

type ForeignKeyRelationship {
  edgeId: ID!
  constraintId: String!
  sourceTable: String!
  targetTable: String!
  columns: [ForeignKeyColumn!]!
  direction: String! # "outbound" or "inbound"
  cardinality: String!
  relationType: String!
}

type ForeignKeyColumn {
  sourceColumn: String!
  targetColumn: String!
}

type ForeignKeyDiscoveryResult {
  entityId: ID!
  relationships: [EntityRelationshipFromFK!]!
  count: Int!
}

type EntityRelationshipFromFK {
  sourceEntityId: ID!
  sourceEntityName: String!
  targetEntityId: ID!
  targetEntityName: String!
  foreignKey: ForeignKeyRelationship!
  cardinality: String!
  relationType: String!
  confidence: Float!
}

type ForeignKeyCreationResult {
  entityId: ID!
  discovered: Int!
  createdEdges: Int!
  failures: [String!]!
}
```

## Usage Examples

### Example 1: Get FK relationships for Customer entity

```bash
curl -X GET "http://localhost:8080/entities/customer-uuid/foreign-keys?tenant_id=t1&datasource_id=d1" \
  -H "X-Tenant-ID: t1" \
  -H "X-Tenant-Datasource-ID: d1"
```

**Response:**
```json
{
  "entity_id": "customer-uuid",
  "relationships": [
    {
      "source_entity_id": "customer-uuid",
      "source_entity_name": "Customer",
      "target_entity_id": "account-uuid",
      "target_entity_name": "Account",
      "foreign_key": {
        "edge_id": "fk-edge-1",
        "source_table": "customers",
        "target_table": "accounts",
        "columns": [
          {
            "source_column": "account_id",
            "target_column": "id"
          }
        ],
        "direction": "outbound",
        "cardinality": "many-to-one"
      },
      "cardinality": "many-to-one",
      "relation_type": "reference",
      "confidence": 1.0
    },
    {
      "source_entity_id": "customer-uuid",
      "source_entity_name": "Customer",
      "target_entity_id": "order-uuid",
      "target_entity_name": "Order",
      "foreign_key": {
        "edge_id": "fk-edge-2",
        "source_table": "orders",
        "target_table": "customers",
        "columns": [
          {
            "source_column": "customer_id",
            "target_column": "id"
          }
        ],
        "direction": "inbound",
        "cardinality": "one-to-many"
      },
      "cardinality": "one-to-many",
      "relation_type": "composition",
      "confidence": 1.0
    }
  ],
  "count": 2
}
```

### Example 2: Auto-create edges from FK discovery

```bash
curl -X POST "http://localhost:8080/entities/customer-uuid/discover-and-link-relationships?tenant_id=t1&datasource_id=d1" \
  -H "X-Tenant-ID: t1" \
  -H "X-Tenant-Datasource-ID: d1"
```

**Response:**
```json
{
  "entity_id": "customer-uuid",
  "discovered": 2,
  "created_edges": 2,
  "failed": 0,
  "failures": []
}
```

### Example 3: Get relationship suggestions including FK-based

```bash
curl -X GET "http://localhost:8080/relationships/suggestions?tenant_id=t1&datasource_id=d1&entity=Customer&include_fk=true" \
  -H "X-Tenant-ID: t1" \
  -H "X-Tenant-Datasource-ID: d1"
```

## Testing

### Unit Test Example

```go
func TestFKDiscovery(t *testing.T) {
    // Setup
    db := setupTestDB(t)
    defer db.Close()
    
    engine := NewForeignKeyDiscoveryEngine(db)
    
    // Test data
    tenantID := "test-tenant"
    datasourceID := "test-datasource"
    
    // Create test entities and tables
    createTestSchema(t, db, tenantID, datasourceID)
    
    // Test: Discover FKs for customers table
    fks, err := engine.DiscoverForeignKeysForTable(
        context.Background(),
        tenantID,
        datasourceID,
        "customers",
    )
    
    require.NoError(t, err)
    require.Len(t, fks, 2)  // customers → accounts, orders → customers
    
    // Verify outbound FK
    assert.Equal(t, "customers", fks[0].SourceTable)
    assert.Equal(t, "accounts", fks[0].TargetTable)
    assert.Equal(t, "outbound", fks[0].Direction)
    assert.Equal(t, "many-to-one", fks[0].Cardinality)
}

func TestEntityRelationshipFromFK(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()
    
    engine := NewForeignKeyDiscoveryEngine(db)
    
    // Setup
    tenantID := "test-tenant"
    datasourceID := "test-datasource"
    createTestSchema(t, db, tenantID, datasourceID)
    
    // Create entity backed by customers table
    customerEntity := &Entity{
        ID: "entity-customer",
        Name: "Customer",
        TableName: "customers",
    }
    
    // Test: Discover entity relationships
    rels, err := engine.DiscoverEntityRelationshipsFromFK(
        context.Background(),
        tenantID,
        datasourceID,
        customerEntity,
    )
    
    require.NoError(t, err)
    require.Len(t, rels, 2)
    
    // Verify Customer → Account relationship
    assert.Equal(t, "Customer", rels[0].SourceEntityName)
    assert.Equal(t, "Account", rels[0].TargetEntityName)
    assert.Equal(t, "reference", rels[0].RelationType)
    assert.Equal(t, 1.0, rels[0].Confidence)
}
```

## Performance Tips

### 1. Batch Discovery for Multiple Entities

```go
func (e *ForeignKeyDiscoveryEngine) DiscoverAllEntityRelationships(
    ctx context.Context,
    tenantID, datasourceID string,
    entityIDs []string,
) (map[string][]EntityRelationshipFromFK, error) {
    results := make(map[string][]EntityRelationshipFromFK)
    
    for _, entityID := range entityIDs {
        entity := &Entity{ID: entityID}
        rels, err := e.DiscoverEntityRelationshipsFromFK(ctx, tenantID, datasourceID, entity)
        if err != nil {
            logging.GetLogger().Sugar().Warnf("Discovery failed for %s: %v", entityID, err)
            continue
        }
        results[entityID] = rels
    }
    
    return results, nil
}
```

### 2. Cache FK Results

```go
type CachedFKEngine struct {
    engine *ForeignKeyDiscoveryEngine
    cache  map[string][]ForeignKeyRelationship
    mu     sync.RWMutex
    ttl    time.Duration
}

func (c *CachedFKEngine) GetForeignKeys(ctx context.Context, table string) ([]ForeignKeyRelationship, error) {
    c.mu.RLock()
    if fks, ok := c.cache[table]; ok {
        defer c.mu.RUnlock()
        return fks, nil
    }
    c.mu.RUnlock()
    
    // Query and cache
    fks, err := c.engine.DiscoverForeignKeysForTable(ctx, table)
    if err == nil {
        c.mu.Lock()
        c.cache[table] = fks
        c.mu.Unlock()
    }
    
    return fks, err
}
```

## Troubleshooting

### Issue: No relationships discovered

**Possible causes:**
1. Entities don't have `table_name` property set
2. FK edges not stored in `catalog_edge` table
3. Entity and table names don't match exactly

**Debug:**
```bash
# Check entities have table_name
SELECT id, name, table_name FROM entities WHERE tenant_id = 't1' LIMIT 10;

# Check FK edges exist
SELECT source_node_id, target_node_id, properties FROM catalog_edge 
WHERE relationship_type = 'foreign_key' 
AND tenant_datasource_id = 'd1' LIMIT 10;
```

### Issue: Incorrect cardinality

**Solution:**
Update the `inferCardinality` method to check for unique constraints:

```go
func (e *ForeignKeyDiscoveryEngine) inferCardinalityAdvanced(
    direction string,
    fk ForeignKeyRelationship,
) string {
    if direction == "outbound" {
        // Check if source columns have unique constraint
        if fk.IsSourceUnique {
            return "one-to-one"
        }
        return "many-to-one"
    }
    return "one-to-many"
}
```

## Next Steps

1. ✅ Integrate FK discovery into your relationship service
2. ✅ Add API endpoints for FK-based discovery
3. ✅ Create/update GraphQL schema
4. ✅ Test with your actual database
5. ✅ Add caching for performance
6. ✅ Monitor FK discovery in production
