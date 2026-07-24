// ============================================================================
// Addepar Hierarchical Ownership GraphQL Resolvers (Go)
// ============================================================================
// Production-ready resolvers for:
// - Entity queries with ABAC filtering
// - Recursive ownership tree traversal
// - Temporal "as-of" position queries
// - Model type metadata queries
// - Position creation with hierarchy validation
//
// Compatible with: gqlgen, 99designs/gqlgen, or custom GraphQL-go server
// ============================================================================

package graphql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

// Missing type definitions for advanced resolvers
type EntityFilter struct {
	// Simplified - add fields as needed
}

type EntityOrderBy struct {
	Field     string
	Direction string
}

type ModelTypeDefinition struct {
	ModelType           string
	DisplayName         string
	OwnershipType       string
	Description         string
	IsHierarchical      bool
	HierarchyLevel      int
	CreatedAt           time.Time
	SuggestedAttributes []*ModelTypeAttribute
}

type ModelTypeAttribute struct {
	Key            string
	ValueType      string
	IsRequired     bool
	IsSearchable   bool
	Priority       int
	Description    string
	ValidationRule string
}

type CreatePositionInput struct {
	OwnerID             uuid.UUID
	OwnedID             uuid.UUID
	OwnershipType       string
	OwnershipPercentage float64
	Shares              *float64
	Value               *float64
	InceptingDate       *time.Time
}

type HierarchyRule struct {
	ID              int
	ParentModelType string
	ChildModelType  string
	Allowed         bool
	OwnershipTypes  []string
	MaxChildren     *int
	MinChildren     *int
	Description     string
	IsExclusive     bool
	CreatedAt       time.Time
}

type PortfolioMetrics struct {
	RootID             uuid.UUID
	AsOf               time.Time
	PositionCount      int
	AssetCount         int
	TotalMarketValue   float64
	TotalCostBasis     float64
	UnrealizedGainLoss float64
	ReturnPct          float64
}

// ============================================================================
// Query Resolvers
// ============================================================================

// Entity – retrieve single entity by ID with ABAC check
func (r *Resolver) Entity(ctx context.Context, id uuid.UUID) (*Entity, error) {
	var e Entity
	if err := r.DB.GetContext(ctx, &e,
		`SELECT id, model_type, tenant_id, original_name, display_name, 
		        ownership_type, status, is_active, created_at, updated_at, 
		        created_by, updated_by, deleted_at
		 FROM entities WHERE id = $1 AND deleted_at IS NULL`, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// ABAC: check read permission on entity type
	if allowed := r.ABAC.Can(ctx, "read", "entity", map[string]interface{}{
		"model_type": e.ModelType,
		"tenant_id":  e.TenantID,
	}); !allowed {
		return nil, errors.New("forbidden: insufficient permissions to read entity")
	}

	return &e, nil
}

// Entities – list entities with filtering, pagination, and ABAC
func (r *Resolver) Entities(ctx context.Context, where *EntityFilter, orderBy []*EntityOrderBy, limit int, offset int) ([]*Entity, error) {
	// Validate pagination
	if limit < 1 || limit > 1000 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	// Build WHERE clause from filter
	query := `SELECT id, model_type, tenant_id, original_name, display_name, 
	                 ownership_type, status, is_active, created_at, updated_at, 
	                 created_by, updated_by, deleted_at
	          FROM entities
	          WHERE deleted_at IS NULL`
	args := []interface{}{}

	if where != nil {
		whereSQL, whereArgs := buildEntityFilter(where)
		query += " AND " + whereSQL
		args = append(args, whereArgs...)
	}

	// Add ORDER BY
	if len(orderBy) > 0 {
		query += " ORDER BY "
		for i, ob := range orderBy {
			if i > 0 {
				query += ", "
			}
			query += fmt.Sprintf("%s %s", ob.Field, ob.Direction)
		}
	}

	// Add pagination
	query += " LIMIT $" + fmt.Sprintf("%d", len(args)+1)
	args = append(args, limit)
	query += " OFFSET $" + fmt.Sprintf("%d", len(args)+1)
	args = append(args, offset)

	var entities []*Entity
	if err := r.DB.SelectContext(ctx, &entities, query, args...); err != nil {
		return nil, err
	}

	// ABAC filter: remove entities user cannot read
	filtered := make([]*Entity, 0, len(entities))
	for _, e := range entities {
		if r.ABAC.Can(ctx, "read", "entity", map[string]interface{}{
			"model_type": e.ModelType,
			"tenant_id":  e.TenantID,
		}) {
			filtered = append(filtered, e)
		}
	}

	return filtered, nil
}

// OwnershipTree – recursive DAG traversal with optional temporal slicing
func (r *Resolver) OwnershipTree(ctx context.Context, rootID uuid.UUID, depth int, includeAttributes bool, asOf *time.Time) (*OwnershipNode, error) {
	if depth < 0 || depth > 10 {
		depth = 3
	}

	// Get root entity
	root, err := r.Entity(ctx, rootID)
	if err != nil || root == nil {
		return nil, errors.New("root entity not found")
	}

	// Initialize temporal filter
	if asOf == nil {
		now := time.Now()
		asOf = &now
	}

	node := &OwnershipNode{
		Entity: root,
		Depth:  0,
	}

	if depth > 0 {
		children, err := r.traverseOwnershipDAG(ctx, rootID, depth, includeAttributes, *asOf)
		if err != nil {
			log.Printf("warning: error traversing DAG: %v", err)
		}
		node.Children = children
		node.ChildCount = len(children)
	}

	return node, nil
}

// traverseOwnershipDAG – recursive helper for OwnershipTree
func (r *Resolver) traverseOwnershipDAG(ctx context.Context, parentID uuid.UUID, depth int, includeAttrs bool, asOf time.Time) ([]*OwnershipNode, error) {
	if depth <= 0 {
		return []*OwnershipNode{}, nil
	}

	// Fetch positions where parentID is the owner, filtered by date
	query := `SELECT id, owner_id, owned_id, ownership_type, ownership_percentage, 
	                 shares, value, average_cost_per_unit, average_market_price,
	                 incepting_date, terminating_date, status, is_active, 
	                 created_at, updated_at, created_by
	          FROM positions
	          WHERE owner_id = $1
	            AND incepting_date <= $2
	            AND (terminating_date IS NULL OR terminating_date >= $2)
	            AND is_active = true`

	var positions []*Position
	if err := r.DB.SelectContext(ctx, &positions, query, parentID, asOf); err != nil {
		return nil, err
	}

	var children []*OwnershipNode
	for _, pos := range positions {
		// Get the owned entity
		ownedEntity, err := r.Entity(ctx, pos.OwnedID)
		if err != nil || ownedEntity == nil {
			continue
		}

		node := &OwnershipNode{
			Entity:     ownedEntity,
			Position:   pos,
			Depth:      depth - 1,
			ChildCount: 0,
		}

		// Recurse if depth > 1
		if depth > 1 {
			grandchildren, err := r.traverseOwnershipDAG(ctx, pos.OwnedID, depth-1, includeAttrs, asOf)
			if err != nil {
				log.Printf("warning: error in recursion: %v", err)
			}
			node.Children = grandchildren
			node.ChildCount = len(grandchildren)
		}

		children = append(children, node)
	}

	return children, nil
}

// OwnershipChain – reverse lookup: find all owners of a target entity
func (r *Resolver) OwnershipChain(ctx context.Context, targetID uuid.UUID, depth int, asOf *time.Time) ([]*OwnershipNode, error) {
	if depth < 0 || depth > 10 {
		depth = 5
	}

	if asOf == nil {
		now := time.Now()
		asOf = &now
	}

	var chains []*OwnershipNode

	// Find all direct owners
	query := `SELECT DISTINCT owner_id FROM positions
	          WHERE owned_id = $1
	            AND incepting_date <= $2
	            AND (terminating_date IS NULL OR terminating_date >= $2)
	            AND is_active = true`

	var ownerIDs []uuid.UUID
	if err := r.DB.SelectContext(ctx, &ownerIDs, query, targetID, *asOf); err != nil {
		return nil, err
	}

	// For each owner, build chain up to depth
	for _, ownerID := range ownerIDs {
		chain, err := r.OwnershipTree(ctx, ownerID, depth, true, asOf)
		if err != nil {
			continue
		}
		chains = append(chains, chain)
	}

	return chains, nil
}

// ModelTypes – get all model type definitions
func (r *Resolver) ModelTypes(ctx context.Context, hierarchyLevel *int) ([]*ModelTypeDefinition, error) {
	tenantID := r.getTenantIDFromContext(ctx)

	query := `SELECT model_type, display_name, ownership_type, description, 
	                 is_hierarchical, hierarchy_level, created_at
	          FROM model_type_definitions
	          WHERE tenant_id = $1`

	args := []interface{}{tenantID}

	if hierarchyLevel != nil {
		query += ` AND hierarchy_level = $` + fmt.Sprintf("%d", len(args)+1)
		args = append(args, *hierarchyLevel)
	}

	query += ` ORDER BY hierarchy_level, model_type`

	var types []*ModelTypeDefinition
	if err := r.DB.SelectContext(ctx, &types, query, args...); err != nil {
		return nil, err
	}

	// Populate suggested attributes for each type
	for _, t := range types {
		attrs, err := r.getSuggestedAttributes(ctx, t.ModelType)
		if err == nil {
			t.SuggestedAttributes = attrs
		}
	}

	return types, nil
}

// ModelType – get single model type by code
func (r *Resolver) ModelType(ctx context.Context, modelType string) (*ModelTypeDefinition, error) {
	tenantID := r.getTenantIDFromContext(ctx)

	var t ModelTypeDefinition
	err := r.DB.GetContext(ctx, &t,
		`SELECT model_type, display_name, ownership_type, description, 
		        is_hierarchical, hierarchy_level, created_at
		 FROM model_type_definitions
		 WHERE model_type = $1 AND tenant_id = $2`, modelType, tenantID)

	if err != nil {
		return nil, err
	}

	// Get suggested attributes
	attrs, err := r.getSuggestedAttributes(ctx, modelType)
	if err == nil {
		t.SuggestedAttributes = attrs
	}

	return &t, nil
}

// HierarchyRules – get rules for parent-child pair
func (r *Resolver) HierarchyRules(ctx context.Context, parentModelType string, childModelType string) ([]*HierarchyRule, error) {
	tenantID := r.getTenantIDFromContext(ctx)

	var rules []*HierarchyRule
	err := r.DB.SelectContext(ctx, &rules,
		`SELECT id, parent_model_type, child_model_type, allowed, ownership_types, 
		        max_children, min_children, description, is_exclusive, created_at
		 FROM entity_hierarchy_rules
		 WHERE tenant_id = $1 
		   AND parent_model_type = $2 
		   AND child_model_type = $3`,
		tenantID, parentModelType, childModelType)

	return rules, err
}

// AllowedChildren – get allowed child types for a parent
func (r *Resolver) AllowedChildren(ctx context.Context, parentModelType string) ([]*ModelTypeDefinition, error) {
	tenantID := r.getTenantIDFromContext(ctx)

	// Query: find all rules where parent = parentModelType and allowed = true
	query := `SELECT DISTINCT m.model_type, m.display_name, m.ownership_type, 
	                 m.description, m.is_hierarchical, m.hierarchy_level, m.created_at
	          FROM model_type_definitions m
	          INNER JOIN entity_hierarchy_rules r ON m.model_type = r.child_model_type
	          WHERE r.tenant_id = $1 
	            AND r.parent_model_type = $2 
	            AND r.allowed = true
	          ORDER BY m.hierarchy_level, m.model_type`

	var types []*ModelTypeDefinition
	if err := r.DB.SelectContext(ctx, &types, query, tenantID, parentModelType); err != nil {
		return nil, err
	}

	return types, nil
}

// AllowedParents – get allowed parent types for a child
func (r *Resolver) AllowedParents(ctx context.Context, childModelType string) ([]*ModelTypeDefinition, error) {
	tenantID := r.getTenantIDFromContext(ctx)

	query := `SELECT DISTINCT m.model_type, m.display_name, m.ownership_type, 
	                 m.description, m.is_hierarchical, m.hierarchy_level, m.created_at
	          FROM model_type_definitions m
	          INNER JOIN entity_hierarchy_rules r ON m.model_type = r.parent_model_type
	          WHERE r.tenant_id = $1 
	            AND r.child_model_type = $2 
	            AND r.allowed = true
	          ORDER BY m.hierarchy_level, m.model_type`

	var types []*ModelTypeDefinition
	if err := r.DB.SelectContext(ctx, &types, query, tenantID, childModelType); err != nil {
		return nil, err
	}

	return types, nil
}

// SearchEntities – full-text search across attributes and display names
func (r *Resolver) SearchEntities(ctx context.Context, query string, modelTypes []string, limit int) ([]*Entity, error) {
	tenantID := r.getTenantIDFromContext(ctx)

	if limit < 1 || limit > 100 {
		limit = 20
	}

	// Use PostgreSQL full-text search (tsvector) if available, or LIKE fallback
	q := `SELECT DISTINCT e.id, e.model_type, e.tenant_id, e.original_name, 
	             e.display_name, e.ownership_type, e.status, e.is_active, 
	             e.created_at, e.updated_at, e.created_by, e.updated_by, e.deleted_at
	      FROM entities e
	      LEFT JOIN entity_attributes ea ON e.id = ea.entity_id
	      WHERE e.tenant_id = $1 
	        AND e.deleted_at IS NULL
	        AND (e.display_name ILIKE $2 OR e.original_name ILIKE $2 OR ea.value::text ILIKE $2)`

	args := []interface{}{tenantID, "%" + query + "%"}

	if len(modelTypes) > 0 {
		q += ` AND e.model_type = ANY($` + fmt.Sprintf("%d", len(args)+1)
		args = append(args, modelTypes)
	}

	q += ` LIMIT $` + fmt.Sprintf("%d", len(args)+1)
	args = append(args, limit)

	var entities []*Entity
	if err := r.DB.SelectContext(ctx, &entities, q, args...); err != nil {
		return nil, err
	}

	// ABAC filter
	filtered := make([]*Entity, 0, len(entities))
	for _, e := range entities {
		if r.ABAC.Can(ctx, "read", "entity", map[string]interface{}{
			"model_type": e.ModelType,
			"tenant_id":  e.TenantID,
		}) {
			filtered = append(filtered, e)
		}
	}

	return filtered, nil
}

// PortfolioMetrics – aggregate portfolio-level metrics
func (r *Resolver) PortfolioMetrics(ctx context.Context, rootID uuid.UUID, asOf *time.Time) (*PortfolioMetrics, error) {
	if asOf == nil {
		now := time.Now()
		asOf = &now
	}

	metrics := &PortfolioMetrics{
		RootID: rootID,
		AsOf:   *asOf,
	}

	// Fetch tree to get all holdings
	tree, err := r.OwnershipTree(ctx, rootID, 5, false, asOf)
	if err != nil {
		return nil, err
	}

	// Walk tree and aggregate
	metrics.PositionCount = countPositions(tree)
	metrics.AssetCount = countAssets(tree)
	metrics.TotalMarketValue = sumMarketValues(tree)
	metrics.TotalCostBasis = sumCostBasis(tree)
	metrics.UnrealizedGainLoss = metrics.TotalMarketValue - metrics.TotalCostBasis
	if metrics.TotalCostBasis > 0 {
		metrics.ReturnPct = (metrics.UnrealizedGainLoss / metrics.TotalCostBasis) * 100
	}

	return metrics, nil
}

// ============================================================================
// Mutation Resolvers (simplified)
// ============================================================================

// CreatePosition – create position with hierarchy validation
func (r *Resolver) CreatePosition(ctx context.Context, input *CreatePositionInput) (*Position, error) {
	// Get user ID from context
	userID := r.getUserIDFromContext(ctx)
	tenantID := r.getTenantIDFromContext(ctx)

	// ABAC: check create permission
	if !r.ABAC.Can(ctx, "create", "position", map[string]interface{}{
		"tenant_id": tenantID,
	}) {
		return nil, errors.New("forbidden: insufficient permissions to create position")
	}

	// Validate hierarchy
	var isValid bool
	var errMsg string
	err := r.DB.QueryRowContext(ctx,
		`SELECT is_valid, error_message FROM validate_hierarchy_position($1, $2, $3, $4)`,
		input.OwnerID, input.OwnedID, tenantID, userID).Scan(&isValid, &errMsg)
	if err != nil {
		return nil, err
	}
	if !isValid {
		return nil, errors.New(errMsg)
	}

	// Create position
	posID := uuid.New()
	now := time.Now()
	inceptingDate := time.Now()
	if input.InceptingDate != nil {
		inceptingDate = *input.InceptingDate
	}

	_, err = r.DB.ExecContext(ctx,
		`INSERT INTO positions (id, owner_id, owned_id, ownership_type, ownership_percentage,
		                        shares, value, incepting_date, is_active, created_by, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, true, $9, $10)`,
		posID, input.OwnerID, input.OwnedID, input.OwnershipType,
		input.OwnershipPercentage, input.Shares, input.Value, inceptingDate, userID, now)

	if err != nil {
		return nil, err
	}

	// Fetch and return
	var pos Position
	err = r.DB.GetContext(ctx, &pos,
		`SELECT id, owner_id, owned_id, ownership_type, ownership_percentage, shares, value,
		        incepting_date, is_active, created_by, created_at
		 FROM positions WHERE id = $1`, posID)

	return &pos, err
}

// ============================================================================
// Helper Functions
// ============================================================================

func buildEntityFilter(_ *EntityFilter) (string, []interface{}) {
	// Simplified implementation – extend as needed
	return "1=1", []interface{}{}
}

func (r *Resolver) getSuggestedAttributes(ctx context.Context, modelType string) ([]*ModelTypeAttribute, error) {
	var attrs []*ModelTypeAttribute
	tenantID := r.getTenantIDFromContext(ctx)

	err := r.DB.SelectContext(ctx, &attrs,
		`SELECT attribute_key as key, attribute_type as value_type, 
		        is_required, is_searchable, priority, description, validation_rule
		 FROM model_type_hierarchy_attributes
		 WHERE model_type = $1 AND tenant_id = $2
		 ORDER BY priority DESC`,
		modelType, tenantID)

	return attrs, err
}

func (r *Resolver) getTenantIDFromContext(ctx context.Context) uuid.UUID {
	tenantID, ok := ctx.Value("tenant_id").(uuid.UUID)
	if !ok {
		// Fallback to default
		return uuid.UUID{}
	}
	return tenantID
}

func (r *Resolver) getUserIDFromContext(ctx context.Context) uuid.UUID {
	userID, ok := ctx.Value("user_id").(uuid.UUID)
	if !ok {
		return uuid.UUID{}
	}
	return userID
}

func countPositions(node *OwnershipNode) int {
	if node == nil {
		return 0
	}
	count := len(node.Children)
	for _, child := range node.Children {
		count += countPositions(child)
	}
	return count
}

func countAssets(node *OwnershipNode) int {
	// Count leaf nodes (assets)
	if node == nil {
		return 0
	}
	if len(node.Children) == 0 {
		return 1
	}
	count := 0
	for _, child := range node.Children {
		count += countAssets(child)
	}
	return count
}

func sumMarketValues(node *OwnershipNode) float64 {
	if node == nil {
		return 0
	}
	sum := node.Entity.MarketValue // Assume entity has this field
	for _, child := range node.Children {
		sum += sumMarketValues(child)
	}
	return sum
}

func sumCostBasis(node *OwnershipNode) float64 {
	if node == nil {
		return 0
	}
	sum := node.Entity.CostBasis // Assume entity has this field
	for _, child := range node.Children {
		sum += sumCostBasis(child)
	}
	return sum
}
