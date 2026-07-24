// backend/internal/api/relationship_suggestions.go
// Improved Workday-Inspired Business Object Linking Service
// Provides enhanced relationship discovery with multi-tenant scoping, caching, and semantic matching
package api

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// EdgeType enum
type RelationshipEdgeType string

const (
	RelationshipEdgeTypeReference   RelationshipEdgeType = "REFERENCE"
	RelationshipEdgeTypeComposition RelationshipEdgeType = "COMPOSITION"
	RelationshipEdgeTypeAssociation RelationshipEdgeType = "ASSOCIATION"
	RelationshipEdgeTypeForeignKey  RelationshipEdgeType = "FOREIGN_KEY"
)

// RelationshipSuggestion struct with enhanced metadata
type RelationshipSuggestion struct {
	ID           string               `json:"id"`
	Title        string               `json:"title"`
	Description  string               `json:"description"`
	SourceEntity string               `json:"sourceEntity"`
	TargetEntity string               `json:"targetEntity"`
	EdgeType     RelationshipEdgeType `json:"edgeType"`
	Cardinality  string               `json:"cardinality,omitempty"`
	FKColumn     string               `json:"fkColumn,omitempty"`
	Confidence   float64              `json:"confidence"`
	Reasoning    string               `json:"reasoning"`
	Dismissible  bool                 `json:"dismissible"`
}

// CatalogNode represents a business object in the semantic layer
type CatalogNode struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenantId"`
	DatasourceID string    `json:"datasourceId"`
	Name         string    `json:"name"`
	Kind         string    `json:"kind"` // table, view, bo
	Description  string    `json:"description"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// CatalogEdge represents a relationship between two business objects
type CatalogEdge struct {
	ID           string               `json:"id"`
	TenantID     string               `json:"tenantId"`
	DatasourceID string               `json:"datasourceId"`
	SourceID     string               `json:"sourceId"`
	TargetID     string               `json:"targetId"`
	EdgeType     RelationshipEdgeType `json:"edgeType"`
	Cardinality  string               `json:"cardinality"`
	FKTable      string               `json:"fkTable"`
	FKColumn     string               `json:"fkColumn"`
	PKTable      string               `json:"pkTable"`
	PKColumn     string               `json:"pkColumn"`
	Confidence   float64              `json:"confidence"`
	Suggested    bool                 `json:"suggested"`
	CreatedBy    string               `json:"createdBy"`
	CreatedAt    time.Time            `json:"createdAt"`
}

// Cache entry for suggestions with TTL
type cacheEntry struct {
	suggestions []RelationshipSuggestion
	timestamp   time.Time
}

var cache = sync.Map{}

const cacheTTL = 5 * time.Minute

// RelationshipService provides relationship discovery and management
type RelationshipService struct {
	db *sql.DB
}

func NewRelationshipService(db *sql.DB) *RelationshipService {
	return &RelationshipService{db: db}
}

// GetRelationshipSuggestions returns top N suggestions for a given entity
// with normalized scoring combining FK evidence, join frequency, and semantic similarity
func (s *RelationshipService) GetRelationshipSuggestions(
	ctx context.Context,
	tenantID, datasourceID, entity string,
	limit int,
) ([]RelationshipSuggestion, error) {
	if limit <= 0 || limit > 50 {
		limit = 5 // Default safe limit
	}

	// Check cache first
	cacheKey := fmt.Sprintf("%s:%s:%s", tenantID, datasourceID, entity)
	if entry, ok := cache.Load(cacheKey); ok {
		e := entry.(cacheEntry)
		if time.Since(e.timestamp) < cacheTTL {
			// Slice to requested limit from cache
			if len(e.suggestions) > limit {
				return e.suggestions[:limit], nil
			}
			return e.suggestions, nil
		}
	}

	// Get existing edges to avoid duplicates
	existingEdges, err := s.getExistingEdges(ctx, tenantID, datasourceID, entity)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch existing edges: %w", err)
	}

	existingKey := make(map[string]bool)
	for _, edge := range existingEdges {
		key := strings.ToLower(fmt.Sprintf("%s>%s", entity, edge.TargetID))
		existingKey[key] = true
	}

	// Get FK hints and relationships
	fkHints, err := s.getFKHints(ctx, tenantID, datasourceID, entity)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch FK hints: %w", err)
	}

	var suggestions []RelationshipSuggestion
	for _, hint := range fkHints {
		tgt := hint.Target
		if tgt == "" || strings.EqualFold(tgt, entity) {
			continue
		}

		key := strings.ToLower(fmt.Sprintf("%s>%s", entity, tgt))
		if existingKey[key] {
			continue // Skip existing edges
		}

		// Compute confidence using normalized scoring model
		confidence, reasoning := s.computeConfidence(ctx, tenantID, datasourceID, entity, tgt, hint)

		// Determine edge type and cardinality
		edgeType, cardinality := s.inferEdgeTypeAndCardinality(hint)

		suggestions = append(suggestions, RelationshipSuggestion{
			ID:           fmt.Sprintf("relsugg-%d", time.Now().UnixNano()),
			Title:        fmt.Sprintf("Link %s → %s", entity, tgt),
			Description:  fmt.Sprintf("Detected %s relationship via %s", edgeType, hint.FKColumn),
			SourceEntity: entity,
			TargetEntity: tgt,
			EdgeType:     edgeType,
			Cardinality:  cardinality,
			FKColumn:     hint.FKColumn,
			Confidence:   confidence,
			Reasoning:    reasoning,
			Dismissible:  true,
		})
	}

	// Sort by confidence descending
	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].Confidence > suggestions[j].Confidence
	})

	// Limit to requested count
	if len(suggestions) > limit {
		suggestions = suggestions[:limit]
	}

	// Cache result
	cache.Store(cacheKey, cacheEntry{suggestions: suggestions, timestamp: time.Now()})

	return suggestions, nil
}

// computeConfidence calculates a normalized confidence score (0-1) using multiple signals
func (s *RelationshipService) computeConfidence(
	ctx context.Context,
	tenantID, datasourceID, entity, target string,
	hint fkHint,
) (float64, string) {
	// Normalized features (0-1)
	fkFeature := 1.0
	if hint.FKColumn == "semantic" {
		fkFeature = 0.5 // Lower confidence for semantic relationships
	}

	// String and semantic similarity (0-1)
	nameSim := stringSimilarity(strings.ToLower(entity), strings.ToLower(target))
	textSim := semanticSimilarity(entity, target)

	// Join frequency estimation (0-1)
	joinFreq := s.estimatedJoinFrequency(ctx, tenantID, datasourceID, entity, target)

	// Prior probability of relationships (tunable, e.g., 0.8 for typical schemas)
	edgePrior := 0.8

	// Weights for different signals (sum to 1.0)
	w1, w2, w3, w4, w5 := 0.4, 0.2, 0.15, 0.15, 0.1

	// Weighted combination
	score := w1*fkFeature + w2*joinFreq + w3*nameSim + w4*textSim + w5*edgePrior

	// Ensure score is in [0, 1]
	if score < 0.0 {
		score = 0.0
	} else if score > 1.0 {
		score = 1.0
	}

	reasoning := fmt.Sprintf(
		"FK: %.2f, JoinFreq: %.2f, NameSim: %.2f, TextSim: %.2f, Prior: %.2f → Confidence: %.2f",
		fkFeature, joinFreq, nameSim, textSim, edgePrior, score,
	)

	return score, reasoning
}

// inferEdgeTypeAndCardinality determines edge type and cardinality from FK hint
func (s *RelationshipService) inferEdgeTypeAndCardinality(hint fkHint) (RelationshipEdgeType, string) {
	if hint.FKColumn == "semantic" {
		return RelationshipEdgeTypeAssociation, "N:N"
	}
	return RelationshipEdgeTypeForeignKey, "N:1"
}

// getExistingEdges retrieves edges already defined for the entity
func (s *RelationshipService) getExistingEdges(ctx context.Context, tenantID, datasourceID, entity string) ([]CatalogEdge, error) {
	query := `
		SELECT ce.id, ce.tenant_id, ce.tenant_datasource_id, ce.source_node_id, ce.target_node_id, ce.relationship_type, 
		       ce.cardinality, ce.fk_table, ce.fk_column, ce.pk_table, ce.pk_column, ce.confidence, ce.suggested, ce.created_by, ce.created_at
		FROM catalog_edge ce
		WHERE ce.tenant_id = $1 AND ce.tenant_datasource_id = $2
		AND ce.source_node_id = (SELECT id FROM catalog_node WHERE tenant_id = $1 AND tenant_datasource_id = $2 AND node_name = $3)
	`

	rows, err := s.db.QueryContext(ctx, query, tenantID, datasourceID, entity)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var edges []CatalogEdge
	for rows.Next() {
		var edge CatalogEdge
		var createdBy sql.NullString
		err := rows.Scan(
			&edge.ID, &edge.TenantID, &edge.DatasourceID, &edge.SourceID, &edge.TargetID,
			&edge.EdgeType, &edge.Cardinality, &edge.FKTable, &edge.FKColumn,
			&edge.PKTable, &edge.PKColumn, &edge.Confidence, &edge.Suggested, &createdBy, &edge.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		if createdBy.Valid {
			edge.CreatedBy = createdBy.String
		} else {
			edge.CreatedBy = ""
		}
		edges = append(edges, edge)
	}

	return edges, rows.Err()
}

// fkHint represents a potential foreign key relationship
type fkHint struct {
	Target   string
	FKColumn string
}

// Placeholder implementations - replace with actual logic
func (s *RelationshipService) getFKHints(ctx context.Context, _, datasourceID, entity string) ([]struct{ Target, FKColumn string }, error) {
	// Query database for FK hints from information_schema
	fkQuery := `
		SELECT 
			kcu.table_name as target_table,
			kcu.column_name as fk_column
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu ON tc.constraint_name = kcu.constraint_name
		WHERE tc.constraint_type = 'FOREIGN KEY'
		AND tc.table_name = $1
	`
	fkRows, err := s.db.QueryContext(ctx, fkQuery, entity)
	if err != nil {
		return nil, err
	}
	defer fkRows.Close()

	var hints []struct{ Target, FKColumn string }
	for fkRows.Next() {
		var hint struct{ Target, FKColumn string }
		if err := fkRows.Scan(&hint.Target, &hint.FKColumn); err != nil {
			return nil, err
		}
		hints = append(hints, hint)
	}

	// Also find semantic relationships - tables that share semantic terms
	// TODO: Fix semantic query - schema column names don't match current database structure
	/*
		semanticQuery := `
			WITH entity_semantic_terms AS (
				-- Find semantic terms mapped to the entity's columns
				SELECT DISTINCT cet.edge_type_name as edge_type, cn_tgt.node_name as semantic_term
				FROM catalog_edge ce
				JOIN catalog_edge_type cet ON cet.id = ce.edge_type_id
				JOIN catalog_node cn_src ON ce.source_node_id = cn_src.id
				JOIN catalog_node cn_tgt ON ce.target_node_id = cn_tgt.id
				JOIN catalog_node_type cnt ON cn_src.node_type_id = cnt.id
				WHERE cet.edge_type_name = 'member of'
				AND cnt.catalog_type_name = 'column'
				AND cn_src.qualified_path LIKE '/public/' || $1 || '/%'
				AND ce.tenant_datasource_id = $2
			),
			related_tables AS (
				-- Find other tables that have columns mapped to the same semantic terms
				SELECT DISTINCT cn_src.qualified_path as table_path,
					SPLIT_PART(cn_src.qualified_path, '/', 3) as table_name
				FROM catalog_edge ce
				JOIN catalog_edge_type cet ON cet.id = ce.edge_type_id
				JOIN catalog_node cn_src ON ce.source_node_id = cn_src.id
				JOIN catalog_node cn_tgt ON ce.target_node_id = cn_tgt.id
				JOIN catalog_node_type cnt ON cn_src.node_type_id = cnt.id
				WHERE cet.edge_type_name = 'member of'
				AND cnt.catalog_type_name = 'column'
				AND cn_tgt.node_name IN (SELECT semantic_term FROM entity_semantic_terms)
				AND ce.tenant_datasource_id = $2
				AND cn_src.qualified_path NOT LIKE '/public/' || $1 || '/%'
			)
			SELECT table_name as target_table, 'semantic' as fk_column
			FROM related_tables
			WHERE table_name != $1
		`

		semanticRows, err := s.db.QueryContext(ctx, semanticQuery, entity, datasourceID)
		if err != nil {
			return nil, err
		}
		defer semanticRows.Close()

		for semanticRows.Next() {
			var hint struct{ Target, FKColumn string }
			if err := semanticRows.Scan(&hint.Target, &hint.FKColumn); err != nil {
				continue
			}
			hints = append(hints, hint)
		}
	*/

	// Query explicit FK relationships from catalog_edge table
	// Include both outgoing FKs (entity references other tables) and incoming FKs (other tables reference entity)
	catalogFKQuery := `
		SELECT DISTINCT
			CASE 
				WHEN LOWER(cn_source.node_name) = LOWER($2) THEN cn_target.node_name
				ELSE cn_source.node_name
			END as target_table,
			COALESCE(
				ce.properties->'columns'->0->>'source_column',
				ce.properties->>'fk_column', 
				ce.properties->>'column', 
				'fk'
			) as fk_column
		FROM catalog_edge ce
		JOIN catalog_node cn_source ON ce.source_node_id = cn_source.id
		JOIN catalog_node cn_target ON ce.target_node_id = cn_target.id
		WHERE ce.relationship_type IN ('FOREIGN_KEY', 'foreign_key', 'reference', 'REFERENCE')
			AND ce.tenant_datasource_id = $1
			AND (LOWER(cn_source.node_name) = LOWER($2) OR LOWER(cn_target.node_name) = LOWER($2))
			AND cn_source.node_name != ''
			AND cn_target.node_name != ''
			AND cn_source.node_name IS NOT NULL
			AND cn_target.node_name IS NOT NULL
	`

	catalogFKRows, err := s.db.QueryContext(ctx, catalogFKQuery, datasourceID, entity)
	if err != nil {
		// Log warning but don't fail - this is an optional enhancement
		// The function can still return results from database-level and semantic queries
		fmt.Printf("Warning: failed to query catalog_edge for FK relationships: %v\n", err)
	} else {
		defer catalogFKRows.Close()
		for catalogFKRows.Next() {
			var hint struct{ Target, FKColumn string }
			if err := catalogFKRows.Scan(&hint.Target, &hint.FKColumn); err != nil {
				continue // Skip this row and continue processing
			}
			// Add hint only if target is different from entity
			if hint.Target != "" && !strings.EqualFold(hint.Target, entity) {
				hints = append(hints, hint)
			}
		}
	}

	return hints, nil
}

func (s *RelationshipService) estimatedJoinFrequency(ctx context.Context, _, datasourceID, entity, target string) float64 {
	// Estimate based on FK count between tables
	fkQuery := `
		SELECT COUNT(*) as fk_count
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu ON tc.constraint_name = kcu.constraint_name
		WHERE tc.constraint_type = 'FOREIGN KEY'
		AND ((tc.table_name = $1 AND kcu.table_name = $2) OR (tc.table_name = $2 AND kcu.table_name = $1))
	`
	var fkCount int
	err := s.db.QueryRowContext(ctx, fkQuery, entity, target).Scan(&fkCount)
	if err != nil {
		fkCount = 0
	}

	// Also count shared semantic terms
	semanticQuery := `
		WITH entity1_terms AS (
			SELECT DISTINCT cn_tgt.node_name as semantic_term
			FROM catalog_edge ce
			JOIN catalog_edge_type cet ON (
				(char_length(ce.edge_type_id) = 36
				 AND ce.edge_type_id ~ '^[0-9a-fA-F0-9-]{36}$'
				 AND cet.id = ce.edge_type_id::uuid)
				 OR cet.edge_type_name = ce.edge_type_id
			)
			JOIN catalog_node cn_src ON ce.source_node_id = cn_src.id
			JOIN catalog_node cn_tgt ON ce.target_node_id = cn_tgt.id
			JOIN catalog_node_type cnt ON cn_src.node_type_id = cnt.id
			WHERE cet.edge_type_name = 'member of'
			AND cnt.catalog_type_name = 'column'
			AND cn_src.qualified_path LIKE '/public/' || $1 || '/%'
			AND ce.tenant_datasource_id = $2
		),
		entity2_terms AS (
			SELECT DISTINCT cn_tgt.node_name as semantic_term
			FROM catalog_edge ce
			JOIN catalog_edge_type cet ON (
				(char_length(ce.edge_type_id) = 36
				 AND ce.edge_type_id ~ '^[0-9a-fA-F0-9-]{36}$'
				 AND cet.id = ce.edge_type_id::uuid)
				 OR cet.edge_type_name = ce.edge_type_id
			)
			JOIN catalog_node cn_src ON ce.source_node_id = cn_src.id
			JOIN catalog_node cn_tgt ON ce.target_node_id = cn_tgt.id
			JOIN catalog_node_type cnt ON cn_src.node_type_id = cnt.id
			WHERE cet.edge_type_name = 'member of'
			AND cnt.catalog_type_name = 'column'
			AND cn_src.qualified_path LIKE '/public/' || $3 || '/%'
			AND ce.tenant_datasource_id = $2
		)
		SELECT COUNT(*) as shared_terms
		FROM entity1_terms e1
		JOIN entity2_terms e2 ON e1.semantic_term = e2.semantic_term
	`
	var sharedTerms int
	err = s.db.QueryRowContext(ctx, semanticQuery, entity, datasourceID, target).Scan(&sharedTerms)
	if err != nil {
		sharedTerms = 0
	}

	// Combine FK strength and semantic similarity
	fkScore := float64(fkCount) / 10.0 // Normalize FK count
	if fkScore > 1.0 {
		fkScore = 1.0
	}

	semanticScore := float64(sharedTerms) / 5.0 // Normalize shared terms
	if semanticScore > 1.0 {
		semanticScore = 1.0
	}

	// Weight FKs more heavily than semantic similarity
	return 0.7*fkScore + 0.3*semanticScore
}

func stringSimilarity(a, b string) float64 {
	// Simple Jaccard similarity
	setA := make(map[string]bool)
	setB := make(map[string]bool)
	for _, w := range strings.Fields(a) {
		setA[strings.ToLower(w)] = true
	}
	for _, w := range strings.Fields(b) {
		setB[strings.ToLower(w)] = true
	}
	intersection := 0
	for w := range setA {
		if setB[w] {
			intersection++
		}
	}
	return float64(intersection) / float64(len(setA)+len(setB)-intersection)
}

func semanticSimilarity(a, b string) float64 {
	// Fallback to stringSimilarity
	return stringSimilarity(a, b)
}
