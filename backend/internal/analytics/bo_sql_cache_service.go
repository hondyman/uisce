package analytics

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// BOSQLCacheService manages caching of resolved BO SQL
type BOSQLCacheService struct {
	db           *sqlx.DB
	graphService *SemanticGraphService
	resolver     *BOContextResolver
}

// NewBOSQLCacheService creates a new cache service
func NewBOSQLCacheService(db *sqlx.DB, graphService *SemanticGraphService, resolver *BOContextResolver) *BOSQLCacheService {
	return &BOSQLCacheService{
		db:           db,
		graphService: graphService,
		resolver:     resolver,
	}
}

// BOSQLCacheEntry represents a row in semantic.bo_sql_cache
type BOSQLCacheEntry struct {
	BOID        uuid.UUID `db:"business_object_id"`
	Dialect     string    `db:"dialect"`
	VersionHash string    `db:"version_hash"`
	SQL         string    `db:"sql"`
	CreatedAt   time.Time `db:"created_at"`
}

// GetOrGenerateBOSQL tries to get SQL from cache, or generates it if missing/stale
func (s *BOSQLCacheService) GetOrGenerateBOSQL(boName string, termNames []string, calcNames []string, tenantID, datasourceID uuid.UUID, dialect string) (string, error) {
	// 1. Resolve BO Context to get ID (needed for hash)
	ctx, err := s.resolver.GetBOContext(boName, tenantID, datasourceID, dialect)
	if err != nil {
		return "", err
	}

	// 2. Compute Version Hash
	hashGen := NewHashGenerator(s.graphService)
	hash, err := hashGen.ComputeHash(ctx.BOID, datasourceID)
	if err != nil {
		return "", fmt.Errorf("failed to compute version hash: %w", err)
	}

	// 3. Check Cache
	cachedSQL, hit, err := s.get(ctx.BOID, dialect, hash)
	if err != nil {
		// Log warning but proceed to generate
		fmt.Printf("Cache get error: %v\n", err)
	}
	if hit {
		return cachedSQL, nil
	}

	// 4. Generate SQL (Cache Miss)
	sql, err := s.resolver.GenerateBOSQL(*ctx, termNames, calcNames)
	if err != nil {
		return "", err
	}

	// 5. Update Cache
	if err := s.put(ctx.BOID, dialect, hash, sql); err != nil {
		// Log warning but don't fail request
		fmt.Printf("Cache put error: %v\n", err)
	}

	return sql, nil
}

// get retrieves a cached SQL query if the hash matches (internal)
func (s *BOSQLCacheService) get(boID uuid.UUID, dialect string, currentHash string) (string, bool, error) {
	var entry BOSQLCacheEntry
	err := s.db.Get(&entry, `
		SELECT business_object_id, dialect, version_hash, sql, created_at
		FROM semantic.bo_sql_cache
		WHERE business_object_id = $1 AND dialect = $2
	`, boID, dialect)

	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}

	if entry.VersionHash != currentHash {
		return "", false, nil // Cache miss (stale)
	}

	return entry.SQL, true, nil
}

// put stores a resolved SQL query in the cache (internal)
func (s *BOSQLCacheService) put(boID uuid.UUID, dialect string, hash string, sqlQuery string) error {
	_, err := s.db.Exec(`
		INSERT INTO semantic.bo_sql_cache (business_object_id, dialect, version_hash, sql, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (business_object_id, dialect)
		DO UPDATE SET
			version_hash = EXCLUDED.version_hash,
			sql = EXCLUDED.sql,
			created_at = NOW()
	`, boID, dialect, hash, sqlQuery)
	return err
}

// Invalidate removes cache entries for specific BOs
func (s *BOSQLCacheService) Invalidate(boIDs []uuid.UUID) error {
	if len(boIDs) == 0 {
		return nil
	}

	query, args, err := sqlx.In(`DELETE FROM semantic.bo_sql_cache WHERE business_object_id IN (?)`, boIDs)
	if err != nil {
		return err
	}
	query = s.db.Rebind(query)

	_, err = s.db.Exec(query, args...)
	return err
}

// GraphDataProvider defines the interface for fetching graph data
type GraphDataProvider interface {
	GetNodeByID(nodeID uuid.UUID) (*SemanticNode, error)
	GetEdgesByType(sourceNodeID uuid.UUID, edgeType EdgeType) ([]SemanticEdge, error)
	GetOutgoingEdges(nodeID uuid.UUID) ([]SemanticEdge, error)
}

// HashGenerator computes version hashes for BOs
type HashGenerator struct {
	dataProvider GraphDataProvider
}

// NewHashGenerator creates a new generator
func NewHashGenerator(dataProvider GraphDataProvider) *HashGenerator {
	return &HashGenerator{dataProvider: dataProvider}
}

// ComputeHash calculates the SHA256 version hash for a BO
func (g *HashGenerator) ComputeHash(boID uuid.UUID, datasourceID uuid.UUID) (string, error) {
	// Gather all dependencies
	deps, err := g.gatherCanonicalDependencies(boID, datasourceID)
	if err != nil {
		return "", err
	}

	// Create canonical string
	canonicalString := deps.CanonicalString()

	// Compute SHA256
	hash := sha256.Sum256([]byte(canonicalString))
	return hex.EncodeToString(hash[:]), nil
}

// CanonicalDependencies holds the dependency graph components
type CanonicalDependencies struct {
	BOMetadata   BOMetadata
	Terms        []TermDependency
	Calculations []CalcDependency
}

type BOMetadata struct {
	Name             string
	Domain           string
	DrivingTable     string
	GovernanceStatus string
}

type TermDependency struct {
	ID               string
	DataType         string
	Category         string
	GovernanceStatus string
	Mappings         []TermMapping
}

type TermMapping struct {
	Table     string
	Column    string
	Priority  int
	IsDefault bool
}

type CalcDependency struct {
	ID               string
	Category         string
	GovernanceStatus string
	ExpressionDSL    string
	Dependencies     []DepRef
}

type DepRef struct {
	Type string // "term" or "calc"
	Ref  string
}

// CanonicalString generates the string to be hashed
func (d *CanonicalDependencies) CanonicalString() string {
	bytes, err := json.Marshal(d) // Use JSON marshaling for canonical representation as requested
	if err != nil {
		// Fallback or panic? For hashing stability, failure here is critical.
		return ""
	}
	return string(bytes)
}

// gatherCanonicalDependencies builds the dependency struct
func (g *HashGenerator) gatherCanonicalDependencies(boID uuid.UUID, datasourceID uuid.UUID) (*CanonicalDependencies, error) {
	// 1. Get BO Node
	boNode, err := g.dataProvider.GetNodeByID(boID)
	if err != nil {
		return nil, err
	}

	var props map[string]interface{}
	// Use Properties from struct
	props = boNode.Properties

	domain, _ := props["domain"].(string)
	drivingTable, _ := props["driving_table"].(string)
	govStatus, _ := props["governance_status"].(string)

	deps := &CanonicalDependencies{
		BOMetadata: BOMetadata{
			Name:             boNode.NodeName,
			Domain:           domain,
			DrivingTable:     drivingTable,
			GovernanceStatus: govStatus,
		},
	}

	// 2. Get used Terms
	termEdges, err := g.dataProvider.GetEdgesByType(boID, EdgeTypeBOHasTerm)
	if err != nil {
		return nil, err
	}

	for _, edge := range termEdges {
		termNode, err := g.dataProvider.GetNodeByID(edge.TargetNodeID)
		if err != nil {
			return nil, err
		}

		var termProps map[string]interface{}
		// SemanticNode.Properties is map[string]interface{} already
		termProps = termNode.Properties

		dataType, _ := termProps["data_type"].(string)
		category, _ := termProps["category"].(string)
		termGov, _ := termProps["governance_status"].(string)

		mappings := g.extractMappings(termNode)

		deps.Terms = append(deps.Terms, TermDependency{
			ID:               termNode.NodeName, // Using name as ID for canonical JSON
			DataType:         dataType,
			Category:         category,
			GovernanceStatus: termGov,
			Mappings:         mappings,
		})
	}
	// Sort terms for determinism
	sort.Slice(deps.Terms, func(i, j int) bool {
		return deps.Terms[i].ID < deps.Terms[j].ID
	})

	// 3. Get used Calculations
	calcEdges, err := g.dataProvider.GetEdgesByType(boID, EdgeTypeBOHasCalc)
	if err != nil {
		return nil, err
	}

	for _, edge := range calcEdges {
		calcDeps, err := g.resolveCalcDependencies(edge.TargetNodeID, datasourceID)
		if err != nil {
			return nil, err
		}
		deps.Calculations = append(deps.Calculations, calcDeps...)
	}

	// Remove duplicates and sort calculations
	uniqueCalcs := make(map[string]CalcDependency)
	for _, c := range deps.Calculations {
		uniqueCalcs[c.ID] = c
	}
	deps.Calculations = make([]CalcDependency, 0, len(uniqueCalcs))
	for _, c := range uniqueCalcs {
		deps.Calculations = append(deps.Calculations, c)
	}
	sort.Slice(deps.Calculations, func(i, j int) bool {
		return deps.Calculations[i].ID < deps.Calculations[j].ID
	})

	return deps, nil
}

// Helper to get node by ID
func (g *HashGenerator) getNodeName(id uuid.UUID) (string, error) {
	node, err := g.dataProvider.GetNodeByID(id)
	if err != nil {
		return "", err
	}
	return node.NodeName, nil
}

// resolveCalcDependencies recursively gathers calc details
func (g *HashGenerator) resolveCalcDependencies(calcID uuid.UUID, datasourceID uuid.UUID) ([]CalcDependency, error) {
	var result []CalcDependency

	calcNode, err := g.dataProvider.GetNodeByID(calcID)
	if err != nil {
		return nil, err
	}

	// Properties and Config are already maps in SemanticNode struct
	category, _ := calcNode.Properties["category"].(string)
	govStatus, _ := calcNode.Properties["governance_status"].(string)

	dsl, _ := calcNode.Config["expression_dsl"].(string)

	// Get direct dependencies
	edges, err := g.dataProvider.GetOutgoingEdges(calcID)
	if err != nil {
		return nil, err
	}

	var directDeps []DepRef
	for _, e := range edges {
		targetName, _ := g.getNodeName(e.TargetNodeID)

		// Recursive collection for parent calcs
		if e.EdgeType == EdgeTypeCalcUsesCalc {
			childDeps, err := g.resolveCalcDependencies(e.TargetNodeID, datasourceID)
			if err != nil {
				return nil, err
			}
			result = append(result, childDeps...)
			directDeps = append(directDeps, DepRef{Type: "calc", Ref: targetName})
		} else if e.EdgeType == EdgeTypeCalcUsesTerm {
			directDeps = append(directDeps, DepRef{Type: "term", Ref: targetName})
		}
	}
	// Sort deps for determinism
	sort.Slice(directDeps, func(i, j int) bool {
		if directDeps[i].Type != directDeps[j].Type {
			return directDeps[i].Type < directDeps[j].Type
		}
		return directDeps[i].Ref < directDeps[j].Ref
	})

	result = append(result, CalcDependency{
		ID:               calcNode.NodeName,
		Category:         category,
		GovernanceStatus: govStatus,
		ExpressionDSL:    dsl,
		Dependencies:     directDeps,
	})

	return result, nil
}

func (g *HashGenerator) extractMappings(termNode *SemanticNode) []TermMapping {
	mappingsRaw, ok := termNode.Config["physical_mappings"]
	if !ok {
		return nil
	}

	// Handle if it's stored as JSON string or raw slice
	// Assuming it's unmarshaled into interface{} by sqlx/json logic

	var mappings []TermMapping

	// Ensure it's a slice
	slice, ok := mappingsRaw.([]interface{})
	if !ok {
		return nil
	}

	for _, item := range slice {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		table, _ := m["table"].(string)
		column, _ := m["column"].(string)
		priority, _ := m["priority"].(float64) // JSON numbers are floats
		isDefault, _ := m["is_default"].(bool)

		mappings = append(mappings, TermMapping{
			Table:     table,
			Column:    column,
			Priority:  int(priority),
			IsDefault: isDefault,
		})
	}

	// Sort mappings by table, column
	sort.Slice(mappings, func(i, j int) bool {
		if mappings[i].Table != mappings[j].Table {
			return mappings[i].Table < mappings[j].Table
		}
		return mappings[i].Column < mappings[j].Column
	})

	return mappings
}

// OnNodeChange handles graph change events
func (s *BOSQLCacheService) OnNodeChange(nodeID uuid.UUID) {
	impactedBOs, err := s.findImpactedBOs(nodeID)
	if err != nil {
		fmt.Printf("Error finding impacted BOs for node %s: %v\n", nodeID, err)
		return
	}

	if len(impactedBOs) > 0 {
		_ = s.Invalidate(impactedBOs)
	}
}

// findImpactedBOs finds BOs impacted by a change to nodeID
func (s *BOSQLCacheService) findImpactedBOs(nodeID uuid.UUID) ([]uuid.UUID, error) {
	nodeType, err := s.getNodeType(nodeID)
	if err != nil {
		return nil, err
	}

	var boIDs []uuid.UUID

	switch nodeType {
	case NodeTypeBusinessObject:
		boIDs = append(boIDs, nodeID)

	case NodeTypeSemanticTerm:
		// Find BOs that use this term directly
		err = s.db.Select(&boIDs, `
			SELECT DISTINCT bo.id
			FROM catalog_node bo
			JOIN catalog_edge e ON e.source_node_id = bo.id AND e.edge_type = 'BO_HAS_TERM'
			JOIN catalog_node term ON term.id = e.target_node_id
			WHERE term.id = $1
		`, nodeID)

	case NodeTypeCalculationTerm:
		// Find BOs that use this calc (directly or via parent calcs)
		// 1. Find all calcs in the dependency tree (parents that use this calc)
		// 2. Find BOs that use any of those calcs
		query := `
			WITH RECURSIVE calc_tree AS (
				-- Start with the changed calculation
				SELECT id
				FROM catalog_node
				WHERE id = $1
				
				UNION
				
				-- Find calculations that use looking-up calculations (reverse CALC_USES_CALC)
				SELECT e.source_node_id
				FROM catalog_edge e
				JOIN calc_tree ct ON e.target_node_id = ct.id
				WHERE e.edge_type = 'CALC_USES_CALC'
			)
			SELECT DISTINCT bo.id
			FROM catalog_node bo
			JOIN catalog_edge e ON e.source_node_id = bo.id AND e.edge_type = 'BO_HAS_CALC'
			WHERE e.target_node_id IN (SELECT id FROM calc_tree)
		`
		err = s.db.Select(&boIDs, query, nodeID)
	}

	return boIDs, err
}

// getNodeType - helper to get node type string (or enum)
func (s *BOSQLCacheService) getNodeType(nodeID uuid.UUID) (NodeType, error) {
	var nodeType string
	err := s.db.Get(&nodeType, `
		SELECT nt.node_type 
		FROM catalog_node n 
		JOIN catalog_node_type nt ON n.node_type_id = nt.id 
		WHERE n.id = $1
	`, nodeID)
	return NodeType(nodeType), err
}
