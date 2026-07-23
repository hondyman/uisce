package rules

import (
	"context"
	"fmt"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
)

// RuleEngine evaluates CEL expressions against input data.
type RuleEngine struct {
	env  *cel.Env
	repo RuleRepository
}

// NewRuleEngine creates a new RuleEngine.
func NewRuleEngine(repo RuleRepository) *RuleEngine {
	env, _ := cel.NewEnv(
		cel.Declarations(
			decls.NewVar("input", decls.NewMapType(decls.String, decls.Dyn)),
		),
	)
	// Error handling ignored for brevity in stub/refactor
	return &RuleEngine{env: env, repo: repo}
}

// Evaluate evaluates a CEL expression against the provided input.
func (e *RuleEngine) Evaluate(ctx context.Context, expression string, input map[string]interface{}) (bool, error) {
	if e.env == nil {
		return false, fmt.Errorf("engine not initialized")
	}
	ast, issues := e.env.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return false, fmt.Errorf("compile error: %w", issues.Err())
	}

	prg, err := e.env.Program(ast)
	if err != nil {
		return false, fmt.Errorf("program creation error: %w", err)
	}

	out, _, err := prg.Eval(map[string]interface{}{
		"input": input,
	})
	if err != nil {
		return false, fmt.Errorf("evaluation error: %w", err)
	}

	result, ok := out.Value().(bool)
	if !ok {
		return false, fmt.Errorf("expression did not return a boolean")
	}

	return result, nil
}

// EvaluateValue evaluates a CEL expression and returns the raw value.
func (e *RuleEngine) EvaluateValue(ctx context.Context, expression string, input map[string]interface{}) (interface{}, error) {
	if e.env == nil {
		return nil, fmt.Errorf("engine not initialized")
	}
	ast, issues := e.env.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("compile error: %w", issues.Err())
	}

	prg, err := e.env.Program(ast)
	if err != nil {
		return nil, fmt.Errorf("program creation error: %w", err)
	}

	out, _, err := prg.Eval(map[string]interface{}{
		"input": input,
	})
	if err != nil {
		return nil, fmt.Errorf("evaluation error: %w", err)
	}

	return out.Value(), nil
}
func (e *RuleEngine) EvaluateTenantRule(ctx context.Context, rule *TenantValidationRule, boCtx map[string]map[string]interface{}) (bool, error) {
	// Stub implementation - in real world this would compile ASL to CEL or use the condition_json
	// For Starlark removal, we assume legacy script_content is ignored.
	return true, nil
}

// EvaluateExpr evaluates a generic expression (formerly Starlark-heavy, now CEL)
func (e *RuleEngine) EvaluateExpr(ctx context.Context, expr string, boCtx map[string]map[string]interface{}) (bool, error) {
	// Flatten boCtx for CEL input if needed, or pass as is
	// This is a rough adaptation
	flatInput := make(map[string]interface{})
	for k, v := range boCtx {
		flatInput[k] = v
	}
	return e.Evaluate(ctx, expr, flatInput)
}

// EvaluateDurationExpr evaluates an expression expected to return a duration in seconds (int).
func (e *RuleEngine) EvaluateDurationExpr(ctx context.Context, expr string, boCtx map[string]map[string]interface{}) (int, error) {
	flatInput := make(map[string]interface{})
	for k, v := range boCtx {
		flatInput[k] = v
	}
	val, err := e.EvaluateValue(ctx, expr, flatInput)
	if err != nil {
		return 0, err
	}
	// Support int, int64, float64
	switch v := val.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	default:
		return 0, fmt.Errorf("expression returned non-numeric type: %T", val)
	}
}

type ConditionEvalTrace struct {
	Expression    string                 `json:"expr"`
	Input         map[string]interface{} `json:"input"`
	RuleMatched   bool                   `json:"matched"`
	ExecutionTime time.Duration          `json:"executionMs"`
	Explanation   string                 `json:"explanation"`
}

func (e *RuleEngine) EvaluateExprDebug(ctx context.Context, expr string, boCtx map[string]map[string]interface{}) (*ConditionEvalTrace, error) {
	return nil, fmt.Errorf("legacy Starlark evaluation is removed")
}
