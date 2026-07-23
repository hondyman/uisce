package domain

import (
	"context"
	"fmt"
)

// SemanticPlannerAdapter applies partial access control decisions as pruning hints to the query engine
type SemanticPlannerAdapter struct {
	Evaluator Evaluator
	Checker   PolicyChecker
}

// PruningHint represents a hint for column or row pruning
type PruningHint struct {
	Type       string // "column" or "row"
	ColumnName string // for column pruning
	Condition  string // SQL condition for row pruning
	Value      any    // value for the condition
}

// PlanQuery applies access control decisions to generate pruning hints for the query engine
func (spa *SemanticPlannerAdapter) PlanQuery(ctx context.Context, req EvaluationRequest, originalQuery string) ([]PruningHint, error) {
	// Evaluate the request to get effective claims
	allow, _, claims, err := spa.Evaluator.Evaluate(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("evaluation failed: %w", err)
	}

	if !allow {
		return []PruningHint{{Type: "row", Condition: "1=0"}}, nil // Block all rows
	}

	// Check policies for additional constraints
	allow, _, matched, scopes, err := spa.Checker.Check(ctx, req, claims)
	if err != nil {
		return nil, fmt.Errorf("policy check failed: %w", err)
	}

	if !allow {
		return []PruningHint{{Type: "row", Condition: "1=0"}}, nil // Block all rows
	}

	// Generate pruning hints based on scopes and matched policies
	hints := []PruningHint{}

	// Example: If scope is limited, add row pruning
	if len(scopes) > 0 && scopes[0] != "all" {
		hints = append(hints, PruningHint{
			Type:      "row",
			Condition: fmt.Sprintf("tenant_id = '%s'", req.TenantID),
		})
	}

	// Example: If certain columns are restricted, add column pruning
	for _, match := range matched {
		if policyID, ok := match["policyId"].(string); ok && policyID == "restrict_sensitive_columns" {
			hints = append(hints, PruningHint{
				Type:       "column",
				ColumnName: "sensitive_data",
			})
		}
	}

	return hints, nil
}

// ApplyHintsToQuery modifies the original query with pruning hints
func (spa *SemanticPlannerAdapter) ApplyHintsToQuery(originalQuery string, hints []PruningHint) string {
	modifiedQuery := originalQuery

	for _, hint := range hints {
		switch hint.Type {
		case "row":
			// Add WHERE clause or modify existing one
			if hint.Condition != "" {
				modifiedQuery = fmt.Sprintf("SELECT * FROM (%s) WHERE %s", modifiedQuery, hint.Condition)
			}
		case "column":
			// Remove sensitive columns from SELECT
			// This is a simplified example; real implementation would parse the query
			modifiedQuery = fmt.Sprintf("SELECT * EXCEPT (%s) FROM (%s)", hint.ColumnName, modifiedQuery)
		}
	}

	return modifiedQuery
}
