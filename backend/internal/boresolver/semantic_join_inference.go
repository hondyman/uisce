package boresolver

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ============================================================================
// JOIN INFERENCE ENGINE
// ============================================================================

// JoinInference handles composite join inference for multi-BO queries.
// This module is responsible for:
// - Building BO relationship graphs
// - Finding join paths via BFS
// - Expanding composite join keys
// - Assigning table aliases
// - Deduplicating joins
type JoinInference struct {
	BORepository *BusinessObjectCachedRepository
}

// NewJoinInference creates a new join inference engine.
func NewJoinInference(boRepo *BusinessObjectCachedRepository) *JoinInference {
	return &JoinInference{
		BORepository: boRepo,
	}
}

// ============================================================================
// Public API
// ============================================================================

// InferJoinsResult contains the output of join inference.
type InferJoinsResult struct {
	Joins       []JoinClause        // JOIN clauses
	AliasesByBO map[string]string   // BO ID → table alias (e.g., "customer_bo" → "t0")
	JoinPaths   map[string][]string // BO ID → path to reach it (for debugging)
}

// JoinClause represents a single JOIN in the generated SQL.
type JoinClause struct {
	JoinType   string // "LEFT", "INNER", "RIGHT"
	TableName  string // physical table name (e.g., "customers")
	TableAlias string // alias in SQL (e.g., "t1")
	Condition  string // ON clause (multi-column: "t0.id = t1.id AND t0.tenant_id = t1.tenant_id")
}

// InferJoins determines which JOINs are needed for a multi-BO query.
//
// This is Part B of the semantic engine: given a set of resolved fields,
// determine which BOs they come from and infer the join paths to connect them.
//
// Parameters:
// - ctx: context for cancellation
// - bo: driving BO (always gets alias "t0")
// - fieldIDs: selected field IDs (used to determine which BOs are needed)
// - resolvedFields: fields that have been resolved through Part A
//
// Returns:
// - joins: all JOIN clauses in order
// - aliasesByBO: map of BO ID → table alias
// - error: if join path cannot be found
func (j *JoinInference) InferJoins(
	ctx context.Context,
	bo *BusinessObjectWithMetadata,
	fieldIDs []string,
	resolvedFields map[string]*ResolvedField,
) (*InferJoinsResult, error) {
	// Load all relationships for this BO
	rels, err := j.BORepository.GetRelationshipsForBO(ctx, bo.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to load BO relationships: %w", err)
	}

	if len(rels) == 0 {
		// No relationships; only driving BO needed
		return &InferJoinsResult{
			Joins:       []JoinClause{},
			AliasesByBO: map[string]string{bo.ID: "t0"},
			JoinPaths:   map[string][]string{bo.ID: {bo.ID}},
		}, nil
	}

	// Build adjacency graph (bidirectional)
	graph := buildBOGraph(rels)

	// Map field ID → BO ID
	// (In current implementation, all fields come from driving BO;
	//  in future, support lookup-driven fields from related BOs)
	boForField := make(map[string]string)
	for _, fieldID := range fieldIDs {
		boForField[fieldID] = bo.ID
	}

	// Initialize alias assignments
	aliasByBO := map[string]string{bo.ID: "t0"}
	aliasCounter := 1
	joinList := []JoinClause{}
	joinPaths := map[string][]string{bo.ID: {bo.ID}}
	deduped := make(map[string]bool)

	// Find join targets (BOs other than driving BO referenced in fields)
	targets := make(map[string]bool)
	for _, targetBO := range boForField {
		if targetBO != bo.ID {
			targets[targetBO] = true
		}
	}

	// For each target BO, find join path and create joins
	for targetBO := range targets {
		if aliasByBO[targetBO] != "" {
			continue // Already joined
		}

		// Find shortest path from driving BO to target
		path := findBOJoinPath(graph, bo.ID, targetBO)
		if len(path) == 0 {
			return nil, fmt.Errorf("no join path from %s to %s", bo.ID, targetBO)
		}

		joinPaths[targetBO] = path

		// Process each relationship in the path
		for i := 0; i < len(path)-1; i++ {
			fromBO := path[i]
			toBO := path[i+1]

			// Find the relationship between these two BOs
			rel := findRelationship(graph, fromBO, toBO)
			if rel == nil {
				return nil, fmt.Errorf("no relationship from %s to %s", fromBO, toBO)
			}

			if aliasByBO[fromBO] == "" {
				aliasByBO[fromBO] = "t0"
			}
			if aliasByBO[toBO] == "" {
				aliasByBO[toBO] = fmt.Sprintf("t%d", aliasCounter)
				aliasCounter++
			}

			fromAlias := aliasByBO[fromBO]
			toAlias := aliasByBO[toBO]
			dedupKey := fmt.Sprintf("%s_%s", toAlias, toBO)

			if deduped[dedupKey] {
				continue // Skip duplicate
			}

			// Build join clause (handles composite keys)
			jc, err := j.buildJoinClause(ctx, rel, fromAlias, toAlias)
			if err != nil {
				return nil, fmt.Errorf("failed to build join clause: %w", err)
			}

			joinList = append(joinList, jc)
			deduped[dedupKey] = true
		}
	}

	return &InferJoinsResult{
		Joins:       joinList,
		AliasesByBO: aliasByBO,
		JoinPaths:   joinPaths,
	}, nil
}

// ============================================================================
// Private: Graph Construction
// ============================================================================

// buildBOGraph creates a bidirectional adjacency list of BO relationships.
func buildBOGraph(rels []*BORelationshipRecord) map[string][]*BORelationshipRecord {
	graph := make(map[string][]*BORelationshipRecord)

	for _, rel := range rels {
		graph[rel.FromBOID] = append(graph[rel.FromBOID], rel)

		// Add reverse relationship for bidirectional search
		reverse := &BORelationshipRecord{
			ID:         rel.ID,
			TenantID:   rel.TenantID,
			FromBOID:   rel.ToBOID,
			ToBOID:     rel.FromBOID,
			JoinType:   rel.JoinType,
			JoinOnJSON: rel.JoinOnJSON,
			IsActive:   rel.IsActive,
			CreatedAt:  rel.CreatedAt,
		}
		graph[rel.ToBOID] = append(graph[rel.ToBOID], reverse)
	}

	return graph
}

// findBOJoinPath finds the shortest path between two BOs using BFS.
// Returns slice of BO IDs representing the path (empty if no path exists).
func findBOJoinPath(
	graph map[string][]*BORelationshipRecord,
	fromBO string,
	toBO string,
) []string {
	if fromBO == toBO {
		return []string{fromBO}
	}

	type node struct {
		BOID string
		Path []string
	}

	queue := []node{{BOID: fromBO, Path: []string{fromBO}}}
	visited := make(map[string]bool)
	visited[fromBO] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current.BOID == toBO {
			return current.Path
		}

		for _, rel := range graph[current.BOID] {
			nextBO := rel.ToBOID
			if nextBO == current.BOID {
				nextBO = rel.FromBOID
			}

			if visited[nextBO] {
				continue
			}

			visited[nextBO] = true
			newPath := append(current.Path, nextBO)
			queue = append(queue, node{
				BOID: nextBO,
				Path: newPath,
			})
		}
	}

	return nil // No path found
}

// findRelationship finds the direct relationship between two BOs.
func findRelationship(
	graph map[string][]*BORelationshipRecord,
	fromBO string,
	toBO string,
) *BORelationshipRecord {
	for _, rel := range graph[fromBO] {
		if (rel.FromBOID == fromBO && rel.ToBOID == toBO) ||
			(rel.ToBOID == fromBO && rel.FromBOID == toBO) {
			return rel
		}
	}
	return nil
}

// ============================================================================
// Private: Join Clause Builder
// ============================================================================

// buildJoinClause constructs a JOIN ON clause for a BO relationship.
// Handles multi-column joins via composite keys in JoinOnPair.
func (j *JoinInference) buildJoinClause(
	ctx context.Context,
	rel *BORelationshipRecord,
	fromAlias string,
	toAlias string,
) (JoinClause, error) {
	var conditions []string

	// Unmarshal JoinOn pairs (composite key support)
	var pairs []JoinOnPair
	if err := json.Unmarshal([]byte(rel.JoinOnJSON), &pairs); err != nil {
		return JoinClause{}, fmt.Errorf("invalid join_on JSON: %w", err)
	}

	// For each pair, resolve field columns
	for _, pair := range pairs {
		fromField, err := j.BORepository.GetFieldByID(ctx, pair.FromFieldID)
		if err != nil {
			return JoinClause{}, fmt.Errorf("failed to load from field %s: %w", pair.FromFieldID, err)
		}

		toField, err := j.BORepository.GetFieldByID(ctx, pair.ToFieldID)
		if err != nil {
			return JoinClause{}, fmt.Errorf("failed to load to field %s: %w", pair.ToFieldID, err)
		}

		// Use physical columns if available
		fromCol := fromField.Name
		toCol := toField.Name

		if fromField.PhysicalColumn != nil && *fromField.PhysicalColumn != "" {
			fromCol = *fromField.PhysicalColumn
		}
		if toField.PhysicalColumn != nil && *toField.PhysicalColumn != "" {
			toCol = *toField.PhysicalColumn
		}

		conditions = append(conditions, fmt.Sprintf(
			"%s.%s = %s.%s",
			fromAlias, fromCol,
			toAlias, toCol,
		))
	}

	if len(conditions) == 0 {
		return JoinClause{}, fmt.Errorf("relationship %s has no join pairs", rel.ID)
	}

	// Determine join type
	joinType := rel.JoinType
	if joinType == "" {
		joinType = "LEFT"
	}

	// Get target BO for driving table name
	targetBO, err := j.BORepository.GetBusinessObject(ctx, rel.ToBOID)
	if err != nil {
		return JoinClause{}, fmt.Errorf("failed to load target BO %s: %w", rel.ToBOID, err)
	}

	return JoinClause{
		JoinType:   joinType,
		TableName:  targetBO.DrivingTable,
		TableAlias: toAlias,
		Condition:  strings.Join(conditions, " AND "),
	}, nil
}
