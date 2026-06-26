# Implementation Guide - Business Object Foreign Key Semantic Discovery

## Quick Start

This feature allows Business Objects to automatically discover and link semantic terms from related tables via foreign key relationships.

### 5-Minute Overview

1. **Metadata Scanner** captures FK column mappings
2. **Service Layer** discovers FK relationships for a BO
3. **API Handlers** expose discovery and linking endpoints
4. **Query Builder** uses join paths to fetch semantic data

### Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                    REST API Layer                            │
│  ┌─────────────────┬─────────────────┬─────────────────┐   │
│  │ foreign-keys    │ related-semantic│ link-semantic   │   │
│  │   endpoint      │ -terms endpoint │  -term endpoint │   │
│  └────────┬────────┴────────┬────────┴────────┬────────┘   │
└───────────┼─────────────────┼─────────────────┼─────────────┘
            │                 │                 │
            └─────────────────┼─────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│            BOSemanticRelationshipsService                   │
│  ┌────────────────────────────────────────────────────┐    │
│  │ DiscoverForeignKeyRelationships()                  │    │
│  │ DiscoverSemanticTermsForRelatedTables()            │    │
│  │ LinkSemanticTermToBusinessObject()                 │    │
│  │ GetBOSemanticJoinPaths()                           │    │
│  └────────────────────────────────────────────────────┘    │
└────────────────────────┬───────────────────────────────────┘
                         │
         ┌───────────────┼───────────────┐
         │               │               │
         ▼               ▼               ▼
    ┌────────┐      ┌────────┐      ┌────────┐
    │catalog │      │catalog │      │   bo   │
    │_edge   │      │_node   │      │ fields │
    │(FK)    │      │(tables)│      │(links) │
    └────────┘      └────────┘      └────────┘
         │               │               │
         └───────────────┼───────────────┘
                         │
                         ▼
                  PostgreSQL Database
```

## File Locations

### Core Implementation Files
- **Service:** [backend/internal/api/bo_semantic_relationships.go](../../backend/internal/api/bo_semantic_relationships.go)
- **Handlers:** [backend/internal/api/bo_semantic_relationships_handler.go](../../backend/internal/api/bo_semantic_relationships_handler.go)
- **Scanner:** [backend/internal/scanner/ansi_scanner.go](../../backend/internal/scanner/ansi_scanner.go)

### Documentation
- **User Guide:** [docs/guides/BO_FK_SEMANTIC_DISCOVERY.md](./BO_FK_SEMANTIC_DISCOVERY.md)
- **API Spec:** [docs/api/BO_FK_SEMANTIC_DISCOVERY_API.md](./BO_FK_SEMANTIC_DISCOVERY_API.md)
- **This Guide:** [docs/guides/BO_FK_SEMANTIC_DISCOVERY_IMPLEMENTATION.md](./BO_FK_SEMANTIC_DISCOVERY_IMPLEMENTATION.md)

## Key Concepts

### Foreign Key Edge

Represents a database FK constraint, stored in `catalog_edge` table:

```go
type ForeignKeyEdge struct {
	EdgeID                 string               // UUID
	SourceTableID          string               // UUID
	TargetTableID          string               // UUID
	Properties             map[string]interface{} // JSON properties
	ColumnMappings         []ColumnMapping      // source_col -> target_col
	Cardinality            string               // "N:1", "1:1", etc.
	OnDeleteAction         string               // CASCADE, SET NULL, etc.
}

type ColumnMapping struct {
	SourceColumn string
	TargetColumn string
}
```

### Related Table FK

Data structure returned when discovering FKs for a BO:

```go
type RelatedTableFK struct {
	EdgeID                string
	RelatedTableID        string
	RelatedTableName      string
	Cardinality           string
	Direction             string          // "outbound" or "inbound"
	ForeignKeyFields      []ColumnMapping
	Properties            map[string]interface{}
}
```

### Semantic Term Availability

What's discovered from related tables:

```go
type RelatedTableSemanticTerm struct {
	SemanticTermID        string          // UUID
	SemanticTermName      string          // Business readable name
	RelatedTableName      string
	RelatedFieldName      string
	RelatedFieldID        string          // UUID
	SourceFKEdgeID        string          // Which FK connects to this
	JoinPath              string          // "source_col -> target_col"
	Confidence            float32         // 0.0-1.0
	MatchReason           string          // Why this term was found
}
```

### BO Field with FK Link

How semantic terms are stored in BO:

```go
type BOFieldWithFK struct {
	BOFieldID             string          // UUID
	BusinessObjectID      string          // UUID
	FieldKey              string          // Unique within BO
	FieldName             string          // Display name
	SemanticTermID        string          // Links to semantic term
	FKEdgeID              string          // Links to FK relationship
	FieldType             string          // "related_object", "scalar", etc.
	IsCore                bool            // Core BO field vs enrichment
	DisplayOrder          int
}
```

## Service Implementation Details

### BOSemanticRelationshipsService

Main service type (in `bo_semantic_relationships.go`):

```go
type BOSemanticRelationshipsService struct {
	db *sqlx.DB
}

func NewBOSemanticRelationshipsService(db *sqlx.DB) *BOSemanticRelationshipsService {
	return &BOSemanticRelationshipsService{db: db}
}
```

### Method: DiscoverForeignKeyRelationshipsForBO

```go
func (s *BOSemanticRelationshipsService) DiscoverForeignKeyRelationshipsForBO(
	ctx context.Context,
	tenantID string,
	boID string,
) ([]RelatedTableFK, error)
```

**Flow:**
1. Retrieve BO and validate it has a `driver_table_id`
2. Query catalog_edge for all edges involving the driver table
3. For each edge, determine if it's outbound (BO refs other) or inbound (other refs BO)
4. Extract column mappings from edge properties
5. Return structured list of FK relationships

**Key SQL Query:**
```sql
WITH fk_edges AS (
  SELECT DISTINCT
    ce.id as edge_id,
    CASE 
      WHEN ce.source_node_id = $1 THEN ce.target_node_id
      ELSE ce.source_node_id
    END as related_table_id,
    CASE 
      WHEN ce.source_node_id = $1 THEN 'outbound'
      ELSE 'inbound'
    END as direction,
    ce.properties
  FROM catalog_edge ce
  WHERE ce.tenant_id = $2
    AND ce.edge_type_name = 'foreign_key'
    AND (ce.source_node_id = $1 OR ce.target_node_id = $1)
)
SELECT 
  e.edge_id,
  n.id as related_table_id,
  n.name as related_table_name,
  e.properties,
  e.direction
FROM fk_edges e
JOIN catalog_node n ON e.related_table_id = n.id
```

### Method: DiscoverSemanticTermsForRelatedTables

```go
func (s *BOSemanticRelationshipsService) DiscoverSemanticTermsForRelatedTables(
	ctx context.Context,
	tenantID string,
	boID string,
	limit int,
) ([]RelatedTableSemanticTerm, error)
```

**Flow:**
1. Call `DiscoverForeignKeyRelationshipsForBO` to get related tables
2. For each related table:
   - Query all columns in that table
   - For each column, find if there's a catalog_edge linking to a semantic_term
   - Extract confidence and match reason
3. Combine results, sort by confidence and edge depth
4. Return limited list

**Key SQL Query (per related table):**
```sql
SELECT 
  st.id as semantic_term_id,
  st.name as semantic_term_name,
  col.name as related_field_name,
  col.id as related_field_id,
  ce.id as source_fk_edge_id,
  st_link.properties->>'join_path' as join_path,
  COALESCE((st_link.properties->>'confidence')::float, 0.9) as confidence
FROM catalog_node col
LEFT JOIN catalog_edge st_link 
  ON col.id = st_link.source_node_id 
  AND st_link.edge_type_name = 'semantic_term_mapping'
  AND st_link.tenant_id = $1
LEFT JOIN catalog_node st 
  ON st_link.target_node_id = st.id 
  AND st.node_type_id IN (SELECT id FROM catalog_node_type WHERE name = 'semantic_term')
WHERE col.parent_id = $2 -- $2 is related_table_id
  AND col.tenant_id = $1
ORDER BY confidence DESC
```

### Method: LinkSemanticTermToBusinessObject

```go
func (s *BOSemanticRelationshipsService) LinkSemanticTermToBusinessObject(
	ctx context.Context,
	tenantID string,
	req *BOSemanticLinkRequest,
) (string, error) // returns bo_field_id
```

**Request Structure:**
```go
type BOSemanticLinkRequest struct {
	SemanticTermID     string `json:"semantic_term_id"`
	RelatedTableID     string `json:"related_table_id"`
	FKEdgeID           string `json:"foreign_key_edge_id"`
	Role               string `json:"role"` // e.g., "customer", "primary_contact"
}
```

**Flow:**
1. Validate all IDs exist and relate correctly
2. Verify semantic term exists in related table
3. Verify FK edge connects BO table to related table
4. Create/update bo_field entry with:
   - `semantic_term_id` set
   - `fk_edge_id` set
   - `field_key` = role (or generated)
   - `field_type` = "related_object"
5. Return created bo_field_id

**SQL Operation:**
```sql
INSERT INTO bo_fields (
  id, 
  business_object_id, 
  semantic_term_id, 
  fk_edge_id, 
  key, 
  name, 
  field_type, 
  is_core, 
  created_at
) VALUES (
  uuid_generate_v4(),
  $1,  -- business_object_id
  $2,  -- semantic_term_id
  $3,  -- fk_edge_id
  $4,  -- role parameter
  $5,  -- derived from semantic term name
  'related_object',
  false,
  NOW()
)
ON CONFLICT (business_object_id, key) DO UPDATE SET
  semantic_term_id = $2,
  fk_edge_id = $3,
  updated_at = NOW()
RETURNING id
```

### Method: GetBOSemanticJoinPaths

```go
func (s *BOSemanticRelationshipsService) GetBOSemanticJoinPaths(
	ctx context.Context,
	tenantID string,
	boID string,
) (map[string]JoinPathInfo, error)
```

**Returns Map:**
```go
type JoinPathInfo struct {
	BOFieldID          string
	SemanticTermID     string
	FKEdgeID           string
	RelatedTableName   string
	RelatedTableID     string
	FKProperties       map[string]interface{}
	JoinSQLTemplate    string
}
```

**Flow:**
1. Query bo_fields for records with both `semantic_term_id` and `fk_edge_id` set
2. For each, look up the FK edge to get column mappings
3. Generate SQL template for the join
4. Return map keyed by bo_field.key

**SQL Query:**
```sql
SELECT 
  bf.id as bo_field_id,
  bf.semantic_term_id,
  bf.fk_edge_id,
  n.name as related_table_name,
  n.id as related_table_id,
  ce.properties as fk_properties
FROM bo_fields bf
JOIN catalog_edge ce ON bf.fk_edge_id = ce.id
JOIN catalog_node n ON ce.target_node_id = n.id
WHERE bf.business_object_id = $1
  AND bf.semantic_term_id IS NOT NULL
  AND bf.fk_edge_id IS NOT NULL
ORDER BY bf.display_order
```

## Handler Implementation Details

### BOSemanticRelationshipsHandler

HTTP handler type (in `bo_semantic_relationships_handler.go`):

```go
type BOSemanticRelationshipsHandler struct {
	service *BOSemanticRelationshipsService
}

func NewBOSemanticRelationshipsHandler(
	service *BOSemanticRelationshipsService,
) *BOSemanticRelationshipsHandler {
	return &BOSemanticRelationshipsHandler{service: service}
}

func (h *BOSemanticRelationshipsHandler) RegisterRoutes(router *chi.Mux) {
	router.Get("/business-objects/{boId}/foreign-keys", 
		h.GetForeignKeys)
	router.Get("/business-objects/{boId}/related-semantic-terms", 
		h.GetRelatedSemanticTerms)
	router.Post("/business-objects/{boId}/link-semantic-term", 
		h.LinkSemanticTerm)
	router.Get("/business-objects/{boId}/semantic-join-paths", 
		h.GetSemanticJoinPaths)
}
```

### Validation Pattern

All handlers follow this pattern:

```go
func (h *BOSemanticRelationshipsHandler) MethodName(w http.ResponseWriter, r *http.Request) {
	// 1. Extract and validate tenant ID
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		respondError(w, 400, "INVALID_TENANT_ID", "X-Tenant-ID header required")
		return
	}
	
	// 2. Extract and validate path parameters
	boID := chi.URLParam(r, "boId")
	if !isValidUUID(boID) {
		respondError(w, 400, "INVALID_BO_ID", "boId must be valid UUID")
		return
	}
	
	// 3. Parse and validate query/body parameters
	// ...
	
	// 4. Call service method
	result, err := h.service.MethodName(r.Context(), tenantID, boID, ...)
	if err != nil {
		// Handle specific error types
		respondError(w, 500, "QUERY_ERROR", err.Error())
		return
	}
	
	// 5. Respond with structured result
	respondJSON(w, 200, result)
}
```

## Metadata Scanner Enhancement

### FK Edge Property Population

In `ansi_scanner.go`, when ForeignKeyConstraint is discovered:

**Before:**
```json
{
	"constraint_name": "fk_orders_customer",
	"columns": [{"source": "customer_id", "target": "id"}]
}
```

**After:**
```json
{
	"edge_type_name": "foreign_key",
	"cardinality": "N:1",
	"source_table": "orders",
	"target_table": "customers",
	"source_schema": "public",
	"target_schema": "public",
	"columns": [{"source_column": "customer_id", "target_column": "id"}],
	"on_delete": "CASCADE",
	"on_update": "CASCADE",
	"primary_constraint_name": "fk_orders_customer_id"
}
```

### Implementation Location

In `ansi_scanner.go` around FK processing:

```go
// Enhance FK edge properties
properties := map[string]interface{}{
	"edge_type_name": "foreign_key",
	"cardinality": inferFKCardinality(fk), // Simple heuristic
	"source_table": fk.SourceTable,
	"target_table": fk.TargetTable,
	"source_schema": fk.SourceSchema,
	"target_schema": fk.TargetSchema,
	"columns": fk.ColumnMappings,
	"on_delete": fk.OnDeleteAction,
	"on_update": fk.OnUpdateAction,
	"primary_constraint_name": fk.ConstraintName,
}

// Save as catalog_edge entry
edge := &CatalogEdge{
	ID: uuid.New().String(),
	SourceNodeID: sourceTableNode.ID,
	TargetNodeID: targetTableNode.ID,
	EdgeTypeName: "foreign_key",
	Properties: properties,
	TenantDatasourceID: tenantDatasourceID,
}
```

### Cardinality Inference

Current simple heuristic in `inferFKCardinality()`:

```go
func inferFKCardinality(fk *ForeignKeyConstraint) string {
	// Simple heuristic: most FKs are N:1
	// Could enhance by querying primary key and unique constraints
	
	// If FK references PK of target table: likely N:1
	// If FK has unique constraint in source table: likely 1:1
	// Otherwise: N:1 (default)
	
	if hasUniqueIndex(fk.SourceColumns) {
		return "1:1"
	}
	return "N:1"
}
```

**Enhancement Opportunity:**
Query `information_schema.table_constraints` to determine actual cardinality.

## Integration Points

### Initialization in Main

```go
// In your main server initialization:

// 1. Create service
boSemanticService := api.NewBOSemanticRelationshipsService(db)

// 2. Create handler
boSemanticHandler := api.NewBOSemanticRelationshipsHandler(boSemanticService)

// 3. Register routes
boSemanticHandler.RegisterRoutes(router)

// Or if using direct registration:
router.Get("/business-objects/{boId}/foreign-keys", 
	boSemanticHandler.GetForeignKeys)
// ... etc
```

### Dependency Injection

```go
// If using dependency injection framework:
container.RegisterSingleton(
	func(db *sqlx.DB) *BOSemanticRelationshipsService {
		return NewBOSemanticRelationshipsService(db)
	},
)

container.RegisterSingleton(
	func(service *BOSemanticRelationshipsService) *BOSemanticRelationshipsHandler {
		return NewBOSemanticRelationshipsHandler(service)
	},
)
```

## Testing

### Unit Test Example

```go
func TestDiscoverForeignKeyRelationships(t *testing.T) {
	// Setup mock database
	db := setupTestDB(t)
	defer db.Close()
	
	// Create service
	service := NewBOSemanticRelationshipsService(db)
	
	// Insert test data (BO, driving table, FK edge)
	boID := insertTestBO(t, db, "test_bo", "orders_table_id")
	insertTestFKEdge(t, db, "orders_table_id", "customers_table_id")
	
	// Call method
	fks, err := service.DiscoverForeignKeyRelationshipsForBO(
		context.Background(),
		"test_tenant",
		boID,
	)
	
	// Assert results
	require.NoError(t, err)
	require.Len(t, fks, 1)
	require.Equal(t, "customers", fks[0].RelatedTableName)
}
```

### Integration Test Example

```go
func TestEndToEndSemanticDiscovery(t *testing.T) {
	// Setup full environment
	server := setupTestServer(t)
	defer server.Close()
	
	// 1. Discover FKs via API
	fksResp := get(t, server, 
		"/api/business-objects/order123/foreign-keys",
		"X-Tenant-ID: tenant1",
	)
	require.Equal(t, 200, fksResp.StatusCode)
	var fks ForeignKeysResponse
	json.Unmarshal(fksResp.Body, &fks)
	require.Greater(t, len(fks.ForeignKeys), 0)
	
	// 2. Discover semantic terms
	termsResp := get(t, server,
		"/api/business-objects/order123/related-semantic-terms",
		"X-Tenant-ID: tenant1",
	)
	require.Equal(t, 200, termsResp.StatusCode)
	
	// 3. Link a semantic term
	linkResp := post(t, server,
		"/api/business-objects/order123/link-semantic-term",
		"X-Tenant-ID: tenant1",
		LinkRequest{...},
	)
	require.Equal(t, 201, linkResp.StatusCode)
	
	// 4. Verify join paths
	pathsResp := get(t, server,
		"/api/business-objects/order123/semantic-join-paths",
		"X-Tenant-ID: tenant1",
	)
	require.Equal(t, 200, pathsResp.StatusCode)
}
```

## Extending the Implementation

### Add Support for Transitive FK Resolution

Allow joins through intermediate tables (2+ hops).

**Implementation:**
1. Modify `DiscoverSemanticTermsForRelatedTables` to support depth parameter
2. For each FK, recursively check FKs of related table
3. Add transitive join path to results
4. Store intermediate_edges in join path metadata

### Add Cardinality Detection

Improve cardinality inference from database constraints.

**Implementation:**
1. Query `information_schema.key_column_usage`
2. Query `information_schema.table_constraints` for UNIQUE
3. Determine actual cardinality:
   - If FK columns are UNIQUE → 1:1
   - If FK references PK → N:1
   - If FK references different column → 1:N (rare)

### Add Circular Reference Detection

Prevent infinite loops in multi-hop joins.

**Implementation:**
1. Track visited table IDs during FK traversal
2. Return early if circular reference detected
3. Mark as "requires_explicit_join_limit" in response
4. Document for frontend to add LIMIT clause

### Add Semantic Term Type Inference

When no explicit semantic term mapped, try to infer from column/table names.

**Implementation:**
1. Add fuzzy matching on column names
2. Identify common patterns (customer_id → customer name lookup)
3. Return with lower confidence score
4. Annotate as "inferred" in match_reason

## Performance Optimization

### Query Caching

Cache discovery results per BO (configurable TTL):

```go
type CachedBOSemanticService struct {
	service *BOSemanticRelationshipsService
	cache   *cache.ExpireCache
	ttl     time.Duration
}

func (c *CachedBOSemanticService) DiscoverForeignKeyRelationshipsForBO(
	ctx context.Context,
	tenantID string,
	boID string,
) ([]RelatedTableFK, error) {
	cacheKey := fmt.Sprintf("fk:%s:%s", tenantID, boID)
	
	if val, found := c.cache.Get(cacheKey); found {
		return val.([]RelatedTableFK), nil
	}
	
	result, err := c.service.DiscoverForeignKeyRelationshipsForBO(
		ctx, tenantID, boID)
	
	if err == nil {
		c.cache.Set(cacheKey, result, c.ttl)
	}
	
	return result, err
}
```

### Index Strategy

Recommended PostgreSQL indexes:

```sql
-- Speed up FK discovery
CREATE INDEX idx_catalog_edge_fk_lookup
ON catalog_edge (tenant_id, edge_type_name, source_node_id, target_node_id);

-- Speed up semantic term discovery
CREATE INDEX idx_catalog_edge_semantic_lookup
ON catalog_edge (tenant_id, source_node_id, edge_type_name)
WHERE edge_type_name = 'semantic_term_mapping';

-- Speed up BO field lookups
CREATE INDEX idx_bo_fields_semantic_links
ON bo_fields (business_object_id, semantic_term_id, fk_edge_id)
WHERE semantic_term_id IS NOT NULL AND fk_edge_id IS NOT NULL;
```

### N+1 Query Prevention

Avoid N+1 queries in discovery:

```go
// DON'T: Query each related table separately
for _, fk := range foreignKeys {
	// This queries DB for each FK
	terms, _ := discoverTerms(fk.RelatedTableID)
}

// DO: Batch query all related tables
relatedIDs := pluck(foreignKeys, "RelatedTableID")
result := batchDiscoverTerms(ctx, tenantID, relatedIDs)
```

## Troubleshooting

### Issue: No FK Relationships Found

**Diagnosis:**
```bash
# 1. Verify BO has driving_table_id
SELECT id, driver_table_id FROM business_objects WHERE id = '{boId}';

# 2. Verify FK edges exist for that table
SELECT * FROM catalog_edge 
WHERE (source_node_id = '{driver_table_id}' 
   OR target_node_id = '{driver_table_id}')
AND edge_type_name = 'foreign_key';

# 3. Check edge properties format
SELECT id, properties FROM catalog_edge 
WHERE edge_type_name = 'foreign_key' LIMIT 1;
```

**Fix:**
- Run metadata scanner to (re)discover FKs
- Verify FK constraints exist in source database
- Check tenant_id matches in queries

### Issue: Semantic Terms Not Showing

**Diagnosis:**
```bash
# 1. Check if semantic term edges exist
SELECT * FROM catalog_edge 
WHERE edge_type_name = 'semantic_term_mapping'
AND tenant_id = '{tenantId}';

# 2. Check if terms linked to columns
SELECT ce.id, ce.properties 
FROM catalog_edge ce
WHERE ce.edge_type_name = 'semantic_term_mapping'
AND ce.source_node_id IN (
  SELECT id FROM catalog_node 
  WHERE parent_id = '{related_table_id}'
);
```

**Fix:**
- Add semantic term mappings via catalog UI
- Run semantic discovery process
- Verify edge properties contain required fields

### Issue: Join Paths Not Returning

**Diagnosis:**
```bash
# Check bo_fields with semantic links
SELECT id, semantic_term_id, fk_edge_id 
FROM bo_fields 
WHERE business_object_id = '{boId}'
AND semantic_term_id IS NOT NULL;
```

**Fix:**
- Use link-semantic-term endpoint to create links first
- Verify bo_fields.fk_edge_id references valid catalog_edge entry
- Check both semantic_term_id and fk_edge_id are not NULL

## Related Documentation

- [User Guide](./BO_FK_SEMANTIC_DISCOVERY.md)
- [API Specification](../api/BO_FK_SEMANTIC_DISCOVERY_API.md)
- [FK Discovery System](./FK_DISCOVERY_SYSTEM.md)
- [Business Object Implementation](./BUSINESS_OBJECT_IMPLEMENTATION.md)
