package genui

import (
	"context"
	"fmt"

	"github.com/hondyman/semlayer/backend/pkg/policy"
)

// VisibilityEvaluator evaluates CEL visibility rules for components
type VisibilityEvaluator struct {
	celEval *policy.CELEvaluator
}

func NewVisibilityEvaluator() (*VisibilityEvaluator, error) {
	celEval, err := policy.NewCELEvaluator()
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL evaluator: %w", err)
	}

	return &VisibilityEvaluator{celEval: celEval}, nil
}

// EvaluateVisibility checks if a component should be visible based on CEL expression
func (ve *VisibilityEvaluator) EvaluateVisibility(
	ctx context.Context,
	expression string,
	context map[string]any,
) (bool, error) {
	if expression == "" {
		return true, nil // No rule = always visible
	}

	return ve.celEval.EvalBool(ctx, expression, context)
}

// FilterComponentsByVisibility filters components based on visibility rules
func (ve *VisibilityEvaluator) FilterComponentsByVisibility(
	ctx context.Context,
	components []ComponentDef,
	context map[string]any,
) ([]ComponentDef, error) {
	var visible []ComponentDef

	for _, comp := range components {
		// Extract visibility rule if present
		visibilityExpr := ""
		if config, ok := comp.Config["visibility"]; ok {
			if expr, ok := config.(string); ok {
				visibilityExpr = expr
			}
		}

		// Evaluate
		isVisible, err := ve.EvaluateVisibility(ctx, visibilityExpr, context)
		if err != nil {
			// Log error but don't block - default to visible
			fmt.Printf("Visibility evaluation error for %s: %v\n", comp.ID, err)
			visible = append(visible, comp)
			continue
		}

		if isVisible {
			visible = append(visible, comp)
		}
	}

	return visible, nil
}

// BuildUserContext creates a context map for visibility evaluation
func BuildUserContext(userID, tenantID, role string, customAttrs map[string]any) map[string]any {
	ctx := map[string]any{
		"user": map[string]any{
			"id":       userID,
			"tenant_id": tenantID,
			"role":     role,
		},
	}

	// Merge custom attributes
	for k, v := range customAttrs {
		ctx[k] = v
	}

	return ctx
}
