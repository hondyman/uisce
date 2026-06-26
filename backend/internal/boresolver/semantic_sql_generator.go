package boresolver

import (
	"context"
	"fmt"
)

// ============================================================================
// INTEGRATED SQL GENERATION WITH SEMANTIC RESOLUTION
// ============================================================================

// SQLGeneratorWithSemanticsConfig holds configuration for the semantic-aware SQL generator.
type SQLGeneratorWithSemanticsConfig struct {
	BORepository      *BusinessObjectCachedRepository
	SemanticRepo      *SemanticTermRepository
	CatalogRepo       *CatalogRepository
	FieldResolver     *FieldResolver
	JoinInference     *JoinInference // Modular join inference engine
	DefaultDialect    string
	DefaultDatasource string
}

// SQLGeneratorWithSemantics wraps the SQL generator with semantic resolution.
// This is the production-grade entry point for SQL generation that properly
// resolves BO fields through the semantic chain using the canonical
// ResolveFieldToPhysical pipeline and composite join inference (Parts A + B).
type SQLGeneratorWithSemantics struct {
	config *SQLGeneratorWithSemanticsConfig
}

// NewSQLGeneratorWithSemantics creates a new semantic-aware SQL generator.
func NewSQLGeneratorWithSemantics(config *SQLGeneratorWithSemanticsConfig) *SQLGeneratorWithSemantics {
	return &SQLGeneratorWithSemantics{config: config}
}

// GenerateSQLForBusinessObject generates SQL for a BO with full semantic resolution.
//
// This is the main entry point. It:
//  1. Validates the BO exists
//  2. Resolves all requested fields through the semantic chain (using Field UUIDs)
//  3. Builds joins based on BO relationships
//  4. Constructs the SQL query
//
// IMPORTANT: req.SelectedFields must contain Field UUIDs (from GET /api/business-objects/{id}/fields),
// NOT field names, display names, or semantic term codes.
//
// Returns the generated SQL along with lineage information for explainability.
func (g *SQLGeneratorWithSemantics) GenerateSQLForBusinessObject(
	ctx context.Context,
	req *SQLGenerationRequest,
	datasourceID string,
) (string, *GenerationExplanation, error) {
	// Use provided datasource or default
	datasource := g.config.DefaultDatasource
	if datasourceID != "" {
		datasource = datasourceID
	}

	// Step 1: Load BO metadata
	bo, err := g.config.BORepository.GetBusinessObject(ctx, req.BusinessObjectID)
	if err != nil {
		return "", nil, fmt.Errorf("failed to load BO %s: %w", req.BusinessObjectID, err)
	}

	// Step 2: Resolve all requested fields through semantic chain
	resolvedFields := make(map[string]*ResolvedField)
	for _, fieldID := range req.SelectedFields {
		resolved, err := g.config.FieldResolver.ResolveFieldToPhysical(ctx, fieldID, datasource)
		if err != nil {
			return "", nil, fmt.Errorf("failed to resolve field %s: %w", fieldID, err)
		}
		resolvedFields[fieldID] = resolved
	}

	// Step 3: Resolve filter fields the same way
	for _, filter := range req.Filters {
		resolved, err := g.config.FieldResolver.ResolveFieldToPhysical(ctx, filter.FieldID, datasource)
		if err != nil {
			return "", nil, fmt.Errorf("failed to resolve filter field %s: %w", filter.FieldID, err)
		}
		// Store for use in WHERE clause
		resolvedFields[filter.FieldID] = resolved
	}

	// Step 4: Assign table aliases
	aliasMap := make(map[string]string)
	aliasMap[bo.DrivingTable] = "t0"

	// Build SELECT clause
	selectClause := buildSelectClause(resolvedFields, aliasMap)

	// Step 5: Build FROM clause
	fromClause := fmt.Sprintf("%s AS t0", bo.DrivingTable)

	// Step 6: Infer joins from BO relationships (Part B: Composite Join Inference)
	// Uses modular JoinInference engine for clean separation of concerns
	joinResult, err := g.config.JoinInference.InferJoins(ctx, bo, req.SelectedFields, resolvedFields)
	if err != nil {
		return "", nil, fmt.Errorf("failed to infer joins: %w", err)
	}

	// Build JOIN clauses from inferred joins
	var joinClauses string
	if len(joinResult.Joins) > 0 {
		for _, j := range joinResult.Joins {
			joinClauses += fmt.Sprintf("\n%s JOIN %s AS %s ON %s",
				j.JoinType, j.TableName, j.TableAlias, j.Condition)
		}
	}

	// Update alias map with inferred aliases
	aliasMap = joinResult.AliasesByBO

	// Step 7: Build WHERE clause
	whereClause := ""
	if len(req.Filters) > 0 {
		whereClause = buildWhereClause(req.Filters, resolvedFields, aliasMap)
	}

	// Step 8: Assemble full query
	sql := fmt.Sprintf("SELECT\n  %s\nFROM %s", selectClause, fromClause)
	if joinClauses != "" {
		sql += joinClauses
	}
	if whereClause != "" {
		sql += fmt.Sprintf("\nWHERE %s", whereClause)
	}
	if req.Limit > 0 {
		sql += fmt.Sprintf("\nLIMIT %d", req.Limit)
	}

	// Generate explanation for lineage/debugging
	explanation := &GenerationExplanation{
		BoID:           req.BusinessObjectID,
		DrivingTable:   bo.DrivingTable,
		ResolvedFields: resolvedFields,
		SQL:            sql,
	}

	return sql, explanation, nil
}

// GenerationExplanation provides lineage and debugging info for generated SQL.
type GenerationExplanation struct {
	BoID           string
	DrivingTable   string
	ResolvedFields map[string]*ResolvedField
	SQL            string
}

// buildSelectClause constructs the SELECT part of the SQL.
func buildSelectClause(resolvedFields map[string]*ResolvedField, aliasMap map[string]string) string {
	clause := ""
	i := 0
	for _, field := range resolvedFields {
		if field.SourceType == "ERROR" {
			continue // Skip error fields
		}

		alias := aliasMap[field.Table]
		if alias == "" {
			alias = "t0"
		}

		col := fmt.Sprintf("%s.%s AS \"%s\"", alias, field.Column, field.FieldName)
		if i > 0 {
			clause += ",\n  "
		}
		clause += col
		i++
	}
	return clause
}

// joinClause represents a single JOIN in the generated SQL.
type joinClause struct {
	JoinType   string
	TableName  string
	TableAlias string
	Condition  string
}

// inferJoins delegates to the join inference service for clean separation of concerns.
func (g *SQLGeneratorWithSemantics) inferJoins(
	ctx context.Context,
	bo *BusinessObjectWithMetadata,
	selectedFieldIDs []string,
	resolvedFields map[string]*ResolvedField,
) ([]joinClause, map[string]string, error) {
	result, err := g.config.JoinInference.InferJoins(ctx, bo, selectedFieldIDs, resolvedFields)
	if err != nil {
		return nil, nil, err
	}

	// Convert JoinClause to joinClause
	joins := make([]joinClause, len(result.Joins))
	for i, jc := range result.Joins {
		joins[i] = joinClause{
			JoinType:   jc.JoinType,
			TableName:  jc.TableName,
			TableAlias: jc.TableAlias,
			Condition:  jc.Condition,
		}
	}

	return joins, result.AliasesByBO, nil
}

// buildWhereClause constructs the WHERE part of the SQL.
func buildWhereClause(filters []FilterClause, resolvedFields map[string]*ResolvedField, aliasMap map[string]string) string {
	conditions := []string{}
	for _, f := range filters {
		resolved, ok := resolvedFields[f.FieldID]
		if !ok {
			continue
		}

		alias := aliasMap[resolved.Table]
		if alias == "" {
			alias = "t0"
		}

		// Format the value
		valStr := ""
		switch v := f.Value.(type) {
		case string:
			// Escape single quotes
			escapedVal := fmt.Sprintf("'%s'", v)
			valStr = escapedVal
		default:
			valStr = fmt.Sprintf("%v", v)
		}

		op := f.Operator
		if op == "" {
			op = "="
		}

		condition := fmt.Sprintf("%s.%s %s %s", alias, resolved.Column, op, valStr)
		conditions = append(conditions, condition)
	}

	// Join conditions with AND
	if len(conditions) == 0 {
		return ""
	}

	where := ""
	for i, cond := range conditions {
		if i > 0 {
			where += " AND "
		}
		where += cond
	}
	return where
}
