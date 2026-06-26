# Quick Reference - BO FK Semantic Discovery

## What This Feature Does

Automatically discovers semantic terms available on related tables (via foreign keys) and links them to Business Object fields for automatic query join generation.

## Files Created/Modified

### New Files Created
| File | Lines | Purpose |
|------|-------|---------|
| `backend/internal/api/bo_semantic_relationships.go` | 373 | Core service for FK discovery and semantic linking |
| `backend/internal/api/bo_semantic_relationships_handler.go` | 168 | REST API handlers (4 endpoints) |

### Modified Files
| File | Changes | Impact |
|------|---------|--------|
| `backend/internal/scanner/ansi_scanner.go` | Enhanced FK edge properties with edge_type_name, cardinality, table names | FK metadata now complete for semantic discovery |

## API Endpoints

### 1. Discover Foreign Keys
```bash
GET /api/business-objects/{boId}/foreign-keys
Header: X-Tenant-ID: {tenant}
→ Returns: List of FK relationships with column mappings
```

### 2. Discover Related Semantic Terms
```bash
GET /api/business-objects/{boId}/related-semantic-terms?limit=50
Header: X-Tenant-ID: {tenant}
→ Returns: Available semantic terms from related tables
```

### 3. Link Semantic Term to BO
```bash
POST /api/business-objects/{boId}/link-semantic-term
Header: X-Tenant-ID: {tenant}
Body: {semantic_term_id, foreign_key_edge_id, related_table_id, role}
→ Returns: Created bo_field_id
```

### 4. Get Join Paths
```bash
GET /api/business-objects/{boId}/semantic-join-paths
Header: X-Tenant-ID: {tenant}
→ Returns: All linked semantic terms with join metadata
```

## Sample Usage Flow

```bash
# 1. See what FKs exist
curl -H "X-Tenant-ID: t1" \
  http://localhost:8080/api/business-objects/bo-123/foreign-keys
# Response: [{edge_id: "fk-1", related_table: "customers"...}]

# 2. See what semantic terms are available
curl -H "X-Tenant-ID: t1" \
  http://localhost:8080/api/business-objects/bo-123/related-semantic-terms
# Response: [{semantic_term_id: "st-cust-name", ...}]

# 3. Link the semantic term you want
curl -X POST -H "X-Tenant-ID: t1" \
  -d '{"semantic_term_id":"st-cust-name","foreign_key_edge_id":"fk-1","role":"customer"}' \
  http://localhost:8080/api/business-objects/bo-123/link-semantic-term
# Response: {bo_field_id: "bf-456"}

# 4. Get the join paths for query building
curl -H "X-Tenant-ID: t1" \
  http://localhost:8080/api/business-objects/bo-123/semantic-join-paths
# Response: {customer: {fk_edge_id: "fk-1", join_sql_template: "LEFT JOIN customers c ON o.customer_id = c.id"}}
```

## Database Schema

### FK Edge (catalog_edge)
```json
{
  "edge_type_name": "foreign_key",
  "cardinality": "N:1",
  "columns": [{source_column, target_column}],
  "source_table": "orders",
  "target_table": "customers",
  "on_delete": "CASCADE"
}
```

### BO Field with FK Link (bo_fields)
```sql
INSERT INTO bo_fields (
  business_object_id,    -- links to BO
  semantic_term_id,      -- what semantic term
  fk_edge_id,            -- how to join (FK edge)
  key,                   -- field identifier
  field_type             -- "related_object"
) VALUES (...)
```

## Service Methods

### Core Service: BOSemanticRelationshipsService

```go
// Discover all FKs for BO's driving table
func (s *BOSemanticRelationshipsService) DiscoverForeignKeyRelationshipsForBO(
	ctx, tenantID, boID) ([]RelatedTableFK, error)

// Find semantic terms on related tables
func (s *BOSemanticRelationshipsService) DiscoverSemanticTermsForRelatedTables(
	ctx, tenantID, boID, limit) ([]RelatedTableSemanticTerm, error)

// Link semantic term to BO via FK
func (s *BOSemanticRelationshipsService) LinkSemanticTermToBusinessObject(
	ctx, tenantID, req *BOSemanticLinkRequest) (string, error)

// Get materialized join paths
func (s *BOSemanticRelationshipsService) GetBOSemanticJoinPaths(
	ctx, tenantID, boID) (map[string]JoinPathInfo, error)
```

## Key Data Structures

### RelatedTableFK
```go
type RelatedTableFK struct {
	EdgeID            string
	RelatedTableID    string
	RelatedTableName  string
	Cardinality       string          // "N:1", "1:1"
	Direction         string          // "outbound", "inbound"
	ForeignKeyFields  []ColumnMapping
	Properties        map[string]interface{}
}
```

### RelatedTableSemanticTerm
```go
type RelatedTableSemanticTerm struct {
	SemanticTermID    string
	SemanticTermName  string
	RelatedTableName  string
	RelatedFieldName  string
	SourceFKEdgeID    string
	JoinPath          string          // "customer_id -> customers.id"
	Confidence        float32         // 0.0-1.0
	MatchReason       string
}
```

### JoinPathInfo
```go
type JoinPathInfo struct {
	BOFieldID         string
	SemanticTermID    string
	FKEdgeID          string
	RelatedTableName  string
	FKProperties      map[string]interface{}
	JoinSQLTemplate   string
}
```

## Validation Rules

- ✓ X-Tenant-ID header required on all endpoints
- ✓ boId must be valid UUID format
- ✓ BO must have driver_table_id set
- ✓ semantic_term_id must reference valid catalog_node
- ✓ fk_edge_id must reference valid catalog_edge with edge_type_name='foreign_key'
- ✓ FK edge must connect BO's driving table to related_table_id

## Error Codes

| Code | Status | Meaning |
|------|--------|---------|
| `INVALID_TENANT_ID` | 400 | Missing/invalid X-Tenant-ID header |
| `INVALID_BO_ID` | 400 | boId not valid UUID |
| `BO_NOT_FOUND` | 404 | BO not found in database |
| `NO_DRIVING_TABLE` | 400 | BO lacks driver_table_id |
| `FK_EDGE_NOT_FOUND` | 404 | FK edge not found or invalid |
| `SEMANTIC_TERM_NOT_FOUND` | 404 | Semantic term not found |
| `TERM_ALREADY_LINKED` | 409 | Term already linked to this BO |
| `INVALID_FK_RELATIONSHIP` | 400 | FK doesn't connect to related table |
| `QUERY_ERROR` | 500 | Database query failed |

## Required Indexes

```sql
CREATE INDEX idx_catalog_edge_fk_lookup
ON catalog_edge (tenant_id, edge_type_name, source_node_id, target_node_id);

CREATE INDEX idx_bo_fields_semantic_links
ON bo_fields (business_object_id, semantic_term_id, fk_edge_id)
WHERE semantic_term_id IS NOT NULL AND fk_edge_id IS NOT NULL;
```

## Integration Checklist

- [ ] Add service to dependency injection / main
- [ ] Register handler routes with router
- [ ] Create required database indexes
- [ ] Run metadata scanner to populate FK edges
- [ ] Add semantic term mappings to catalog
- [ ] Test with sample BO and related table
- [ ] Implement frontend UI components
- [ ] Add to query builder for join path use

## Query Example with Join Paths

```sql
-- Using join path metadata from API:
SELECT 
  o.id,
  o.customer_id,
  o.order_date,
  c.name AS customer_name    -- from related table
FROM orders o
LEFT JOIN customers c ON o.customer_id = c.id
WHERE o.id = ?

-- The join_sql_template from JoinPathInfo:
-- "LEFT JOIN customers c ON o.customer_id = c.id"
```

## Testing

### Quick Test
```bash
# Create test BO
curl -X POST http://localhost:8080/api/business-objects \
  -d '{"name":"orders_bo","driver_table_id":"orders_table"}'

# Discover FKs
curl -H "X-Tenant-ID: tenant1" \
  http://localhost:8080/api/business-objects/{boId}/foreign-keys

# Should return FKs from orders table
```

### Verify FK Edges in Database
```sql
SELECT id, edge_type_name, properties 
FROM catalog_edge 
WHERE edge_type_name = 'foreign_key' 
LIMIT 1;
-- Should show: edge_type_name, cardinality, columns, table names
```

## Documentation

- **User Guide:** [BO_FK_SEMANTIC_DISCOVERY.md](./BO_FK_SEMANTIC_DISCOVERY.md)
- **API Spec:** [BO_FK_SEMANTIC_DISCOVERY_API.md](../api/BO_FK_SEMANTIC_DISCOVERY_API.md)
- **Implementation Guide:** [BO_FK_SEMANTIC_DISCOVERY_IMPLEMENTATION.md](./BO_FK_SEMANTIC_DISCOVERY_IMPLEMENTATION.md)

## Performance Notes

- FK discovery: O(n) where n = # FKs from driving table
- Semantic term discovery: O(n×m) where m = semantic terms per table
- Typical query time: <100ms for normal schemas
- Recommended cache TTL: 24 hours for discovery results
- Batch index: idx_bo_fields_semantic_links for fast lookups

## Future Enhancements

- [ ] Transitive FK resolution (2+ hop joins)
- [ ] Improved cardinality detection from constraints
- [ ] Circular reference detection
- [ ] Semantic term name inference from column names
- [ ] Automatic join limit for circular cases
- [ ] FK change detection and remediation
