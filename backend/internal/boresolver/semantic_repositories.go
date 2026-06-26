package boresolver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// ============================================================================
// SEMANTIC TERM REPOSITORY
// ============================================================================

// SemanticTermRepository handles all semantic term queries with caching.
type SemanticTermRepository struct {
	db    *sqlx.DB
	cache Cache[string, *SemanticTerm]
}

// NewSemanticTermRepository creates a new semantic term repository with caching.
func NewSemanticTermRepository(db *sqlx.DB) *SemanticTermRepository {
	return &SemanticTermRepository{
		db:    db,
		cache: NewMapCache[string, *SemanticTerm](),
	}
}

// GetSemanticTerm retrieves a semantic term by ID with caching.
func (r *SemanticTermRepository) GetSemanticTerm(ctx context.Context, id string) (*SemanticTerm, error) {
	if v, ok := r.cache.Get(id); ok {
		return v, nil
	}

	var term SemanticTerm
	err := r.db.GetContext(ctx, &term,
		`SELECT id, tenant_id, name, display_name, description, category, is_system, created_at, updated_at
		 FROM semantic_terms
		 WHERE id = $1`,
		id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch semantic term %s: %w", id, err)
	}

	r.cache.Set(id, &term)
	return &term, nil
}

// GetSemanticTermByName retrieves a semantic term by name (not cached as it's less common).
func (r *SemanticTermRepository) GetSemanticTermByName(ctx context.Context, name string) (*SemanticTerm, error) {
	var term SemanticTerm
	err := r.db.GetContext(ctx, &term,
		`SELECT id, tenant_id, name, display_name, description, category, is_system, created_at, updated_at
		 FROM semantic_terms
		 WHERE name = $1 LIMIT 1`,
		name)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch semantic term by name %s: %w", name, err)
	}

	r.cache.Set(term.ID, &term)
	return &term, nil
}

// ClearCache clears the semantic term cache (useful for testing or cache invalidation).
func (r *SemanticTermRepository) ClearCache() {
	r.cache.Clear()
}

// ============================================================================
// CATALOG REPOSITORY
// ============================================================================

// edgeKey is used for caching catalog edges (composite key: termID + datasourceID).
type edgeKey struct {
	TermID       string
	DatasourceID string
}

// CatalogRepository handles all catalog node and edge queries with caching.
type CatalogRepository struct {
	db        *sqlx.DB
	nodeCache Cache[string, *CatalogNode]
	edgeCache Cache[edgeKey, []*CatalogEdge]
}

// NewCatalogRepository creates a new catalog repository with caching.
func NewCatalogRepository(db *sqlx.DB) *CatalogRepository {
	return &CatalogRepository{
		db:        db,
		nodeCache: NewMapCache[string, *CatalogNode](),
		edgeCache: NewMapCache[edgeKey, []*CatalogEdge](),
	}
}

// GetNode retrieves a catalog node by ID with caching.
func (r *CatalogRepository) GetNode(ctx context.Context, id string) (*CatalogNode, error) {
	if v, ok := r.nodeCache.Get(id); ok {
		return v, nil
	}

	var node CatalogNode
	err := r.db.GetContext(ctx, &node,
		`SELECT id, tenant_id, type, name, parent_id, metadata, created_at, updated_at
		 FROM catalog_nodes
		 WHERE id = $1`,
		id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch catalog node %s: %w", id, err)
	}

	r.nodeCache.Set(id, &node)
	return &node, nil
}

// GetEdges retrieves all catalog edges for a semantic term and datasource with caching.
func (r *CatalogRepository) GetEdges(ctx context.Context, termID, datasourceID string) ([]*CatalogEdge, error) {
	key := edgeKey{TermID: termID, DatasourceID: datasourceID}
	if v, ok := r.edgeCache.Get(key); ok {
		return v, nil
	}

	var edges []*CatalogEdge
	err := r.db.SelectContext(ctx, &edges,
		`SELECT id, tenant_id, from_id, to_id, type, datasource_id, metadata, created_at
		 FROM catalog_edges
		 WHERE from_id = $1 AND datasource_id = $2`,
		termID, datasourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch catalog edges for term %s / datasource %s: %w", termID, datasourceID, err)
	}

	r.edgeCache.Set(key, edges)
	return edges, nil
}

// ClearCache clears all catalog caches.
func (r *CatalogRepository) ClearCache() {
	r.nodeCache.Clear()
	r.edgeCache.Clear()
}

// ============================================================================
// BUSINESS OBJECT REPOSITORY (ENHANCED WITH CACHING)
// ============================================================================

// boFieldsKey is used for caching fields by BO.
type boFieldsKey struct {
	BOID string
}

// boRelationshipsKey is used for caching relationships by BO.
type boRelationshipsKey struct {
	BOID string
}

// BusinessObjectCachedRepository extends the BO repository with caching for fields and relationships.
type BusinessObjectCachedRepository struct {
	db                *sqlx.DB
	boCache           Cache[string, *BusinessObjectWithMetadata]
	fieldCache        Cache[string, *BOFieldWithMetadata]
	fieldsByBOCache   Cache[boFieldsKey, []*BOFieldWithMetadata]
	relationshipCache Cache[boRelationshipsKey, []*BORelationshipRecord]
}

// BusinessObjectWithMetadata wraps BO data with driving table and other metadata.
type BusinessObjectWithMetadata struct {
	ID            string
	Name          string
	TechnicalName string
	DrivingTable  string
	Description   string
	IsSystem      bool
	CreatedAt     string
}

// BOFieldWithMetadata wraps BO field with semantic term and override info.
type BOFieldWithMetadata struct {
	ID               string  `db:"id"`
	Name             string  `db:"name"`
	TechnicalName    string  `db:"technical_name"`
	BusinessObjectID string  `db:"business_object_id"`
	SemanticTermID   string  `db:"semantic_term_id"`
	PhysicalTable    *string `db:"physical_table"`  // Override
	PhysicalColumn   *string `db:"physical_column"` // Override
	Type             string  `db:"type"`
	IsRequired       bool    `db:"is_required"`
}

// NewBusinessObjectCachedRepository creates a new cached BO repository.
func NewBusinessObjectCachedRepository(db *sqlx.DB) *BusinessObjectCachedRepository {
	return &BusinessObjectCachedRepository{
		db:                db,
		boCache:           NewMapCache[string, *BusinessObjectWithMetadata](),
		fieldCache:        NewMapCache[string, *BOFieldWithMetadata](),
		fieldsByBOCache:   NewMapCache[boFieldsKey, []*BOFieldWithMetadata](),
		relationshipCache: NewMapCache[boRelationshipsKey, []*BORelationshipRecord](),
	}
}

// GetBusinessObject retrieves a BO with metadata.
func (r *BusinessObjectCachedRepository) GetBusinessObject(ctx context.Context, id string) (*BusinessObjectWithMetadata, error) {
	if v, ok := r.boCache.Get(id); ok {
		return v, nil
	}

	var bo BusinessObjectWithMetadata
	err := r.db.GetContext(ctx, &bo,
		`SELECT id, name, technical_name, is_system, created_at
		 FROM business_objects
		 WHERE id = $1`,
		id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch business object %s: %w", id, err)
	}

	// Assume driving_table is the technical_name for now (can be extended)
	bo.DrivingTable = bo.TechnicalName

	r.boCache.Set(id, &bo)
	return &bo, nil
}

// GetFieldByID retrieves a single BO field.
func (r *BusinessObjectCachedRepository) GetFieldByID(ctx context.Context, fieldID string) (*BOFieldWithMetadata, error) {
	if v, ok := r.fieldCache.Get(fieldID); ok {
		return v, nil
	}

	var field BOFieldWithMetadata
	err := r.db.GetContext(ctx, &field,
		`SELECT id, name, technical_name, business_object_id, semantic_term_id, NULL::text AS physical_table, NULL::text AS physical_column, type, is_required
		 FROM bo_fields
		 WHERE id = $1`,
		fieldID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch BO field %s: %w", fieldID, err)
	}

	r.fieldCache.Set(fieldID, &field)
	return &field, nil
}

// GetFieldsForBO retrieves all fields for a BO.
func (r *BusinessObjectCachedRepository) GetFieldsForBO(ctx context.Context, boID string) ([]*BOFieldWithMetadata, error) {
	key := boFieldsKey{BOID: boID}
	if v, ok := r.fieldsByBOCache.Get(key); ok {
		return v, nil
	}

	var fields []*BOFieldWithMetadata
	err := r.db.SelectContext(ctx, &fields,
		`SELECT id, name, technical_name, business_object_id, semantic_term_id, NULL::text AS physical_table, NULL::text AS physical_column, type, is_required
		 FROM bo_fields
		 WHERE business_object_id = $1
		 ORDER BY id`,
		boID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch BO fields for %s: %w", boID, err)
	}

	// Populate individual field cache as well
	for _, f := range fields {
		r.fieldCache.Set(f.ID, f)
	}

	r.fieldsByBOCache.Set(key, fields)
	return fields, nil
}

// GetRelationshipsForBO retrieves all relationships for a BO.
func (r *BusinessObjectCachedRepository) GetRelationshipsForBO(ctx context.Context, boID string) ([]*BORelationshipRecord, error) {
	key := boRelationshipsKey{BOID: boID}
	if v, ok := r.relationshipCache.Get(key); ok {
		return v, nil
	}

	var rels []*BORelationshipRecord
	err := r.db.SelectContext(ctx, &rels,
		`SELECT id, tenant_id, from_bo_id, to_bo_id, join_type, join_on, metadata, is_active, created_at
		 FROM bo_relationships
		 WHERE (from_bo_id = $1 OR to_bo_id = $1) AND is_active = true`,
		boID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch BO relationships for %s: %w", boID, err)
	}

	r.relationshipCache.Set(key, rels)
	return rels, nil
}

// ParseJoinOn parses the join_on JSON field into []JoinOnPair.
func ParseJoinOn(joinOnJSON string) ([]JoinOnPair, error) {
	var pairs []JoinOnPair
	if joinOnJSON == "" {
		return pairs, nil
	}
	if err := json.Unmarshal([]byte(joinOnJSON), &pairs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal join_on: %w", err)
	}
	return pairs, nil
}

// ClearCache clears all caches.
func (r *BusinessObjectCachedRepository) ClearCache() {
	r.boCache.Clear()
	r.fieldCache.Clear()
	r.fieldsByBOCache.Clear()
	r.relationshipCache.Clear()
}
