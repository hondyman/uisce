package analytics

import (
	"context"
	"fmt"

	"github.com/hondyman/semlayer/backend/internal/cbo"
)

// SemanticRepoAdapter adapts BOContextResolver to cbo.SemanticRepository
type SemanticRepoAdapter struct {
	resolver *BOContextResolver
}

// NewSemanticRepoAdapter creates a new adapter
func NewSemanticRepoAdapter(resolver *BOContextResolver) *SemanticRepoAdapter {
	return &SemanticRepoAdapter{resolver: resolver}
}

// ResolveBaseSQL resolves the SQL for the base plan
func (a *SemanticRepoAdapter) ResolveBaseSQL(ctx context.Context, pc cbo.PlanContext) (string, error) {
	// Convert cbo.PlanContext to BOContext
	dialect := "postgres" // Default
	boCtx := BOContext{
		BOName:   pc.BOName,
		TenantID: *pc.TenantID,
		Dialect:  dialect,
	}
	if pc.DatasourceID != nil {
		boCtx.DatasourceID = *pc.DatasourceID
	}

	// We need to look up the driving table to fully populate BOContext
	// Ideally BOContextResolver has a helper for this or we load it
	fullCtx, err := a.resolver.GetBOContext(pc.BOName, *pc.TenantID, boCtx.DatasourceID, dialect)
	if err != nil {
		return "", err
	}

	// Generate the SQL
	// Use helper to categorize fields
	termNames, calcNames, err := a.categorizeFields(fullCtx, pc.GroupBy, pc.Measures)
	if err != nil {
		return "", err
	}

	sql, err := a.resolver.GenerateBOSQL(*fullCtx, termNames, calcNames)
	if err != nil {
		return "", err
	}

	// Append filters
	sql = a.appendFilters(sql, pc.Filters)

	return sql, nil
}

// ResolvePreAggSQL resolves the SQL for a pre-aggregation plan
func (a *SemanticRepoAdapter) ResolvePreAggSQL(ctx context.Context, pc cbo.PlanContext, preAgg cbo.PreAggDescriptor) (string, error) {
	// For a pre-agg, we select from the pre-agg target table
	// We map the requested dimensions/measures to columns in the pre-agg

	// Basic implementation: SELECT <dims>, <measures> FROM <pre_agg_table>

	cols := []string{}
	cols = append(cols, pc.GroupBy...)  // Dimensions
	cols = append(cols, pc.Measures...) // Measures

	colList := "*"
	if len(cols) > 0 {
		colList = ""
		for i, c := range cols {
			if i > 0 {
				colList += ", "
			}
			colList += fmt.Sprintf("\"%s\"", c) // Quote identifiers
		}
	}

	sql := fmt.Sprintf("SELECT %s FROM %s", colList, preAgg.TargetTable)

	// Append filters
	sql = a.appendFilters(sql, pc.Filters)

	return sql, nil
}

// categorizeFields separates requested fields into terms and calculations
func (a *SemanticRepoAdapter) categorizeFields(ctx *BOContext, dimensions []string, measures []string) ([]string, []string, error) {
	// In a real implementation, we would check the graph or metadata to know if a field is a term or calculation
	// For now, let's assume dimensions are terms and measures are calculations (common pattern)
	// But actually, measures can be terms too.

	// We can try to use ResolveCalculation to check if it's a calc.
	var terms []string
	var calcs []string

	check := func(name string) error {
		// Try to resolve as calc
		_, err := a.resolver.ResolveCalculation(name, *ctx)
		if err == nil {
			calcs = append(calcs, name)
			return nil
		}

		// If not calc, assume term
		terms = append(terms, name)
		return nil
	}

	for _, d := range dimensions {
		if err := check(d); err != nil {
			return nil, nil, err
		}
	}

	for _, m := range measures {
		if err := check(m); err != nil {
			return nil, nil, err
		}
	}

	return terms, calcs, nil
}

// appendFilters adds WHERE clauses to SQL
func (a *SemanticRepoAdapter) appendFilters(sql string, filters map[string]interface{}) string {
	if len(filters) == 0 {
		return sql
	}

	clause := " WHERE "

	conditions := []string{}
	for k, v := range filters {
		conditions = append(conditions, fmt.Sprintf("%s = '%v'", k, v))
	}

	if len(conditions) > 0 {
		sql += clause + conditions[0]
		for i := 1; i < len(conditions); i++ {
			sql += " AND " + conditions[i]
		}
	}

	return sql
}
