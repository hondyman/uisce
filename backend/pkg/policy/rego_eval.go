package policy

import (
	"context"

	"github.com/open-policy-agent/opa/rego"
)

// RegoEvaluator evaluates Rego policies for complex rules
type RegoEvaluator struct{}

// NewRegoEvaluator creates a new Rego evaluator
func NewRegoEvaluator() *RegoEvaluator {
	return &RegoEvaluator{}
}

// EvalBool evaluates a Rego policy that returns a boolean
func (r *RegoEvaluator) EvalBool(ctx context.Context, module string, input map[string]interface{}, query string) (bool, error) {
	regoObj := rego.New(
		rego.Module("policy.rego", module),
		rego.Query(query),
		rego.Input(input),
	)

	rs, err := regoObj.Eval(ctx)
	if err != nil {
		return false, err
	}

	if len(rs) == 0 || len(rs[0].Expressions) == 0 {
		return false, nil
	}

	if v, ok := rs[0].Expressions[0].Value.(bool); ok {
		return v, nil
	}

	return false, nil
}

// EvalArray evaluates a Rego policy that returns an array (e.g., disclosures)
func (r *RegoEvaluator) EvalArray(ctx context.Context, module string, input map[string]interface{}, query string) ([]interface{}, error) {
	regoObj := rego.New(
		rego.Module("policy.rego", module),
		rego.Query(query),
		rego.Input(input),
	)

	rs, err := regoObj.Eval(ctx)
	if err != nil {
		return nil, err
	}

	if len(rs) == 0 || len(rs[0].Expressions) == 0 {
		return nil, nil
	}

	if arr, ok := rs[0].Expressions[0].Value.([]interface{}); ok {
		return arr, nil
	}

	return nil, nil
}
