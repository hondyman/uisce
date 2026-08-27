package cel

import (
	"fmt"
	"sync"

	"github.com/google/cel-go/cel"
)

// CELEvaluator evaluates Google Common Expression Language (CEL) rules over typed variables and dynamic attributes.
type CELEvaluator struct {
	mu       sync.RWMutex
	env      *cel.Env
	programs map[string]cel.Program
}

// NewCELEvaluator creates a new CELEvaluator configured with standard financial variables and optional custom declarations.
func NewCELEvaluator(customFieldDecls ...cel.EnvOption) (*CELEvaluator, error) {
	standardDecls := []cel.EnvOption{
		cel.Variable("order_amount", cel.DoubleType),
		cel.Variable("account_subtype", cel.StringType),
		cel.Variable("restriction_flag", cel.BoolType),
		cel.Variable("nav", cel.DoubleType),
		cel.Variable("hurdle_rate_pct", cel.DoubleType),
		cel.Variable("esg_score", cel.DoubleType),
		cel.Variable("jurisdiction", cel.StringType),
		cel.Variable("is_qualified_investor", cel.BoolType),
		cel.Variable("liquidity_buffer_pct", cel.DoubleType),
	}

	opts := append(standardDecls, customFieldDecls...)
	env, err := cel.NewEnv(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL environment: %w", err)
	}

	return &CELEvaluator{
		env:      env,
		programs: make(map[string]cel.Program),
	}, nil
}

// CompileProgram pre-compiles and caches a CEL rule expression.
func (c *CELEvaluator) CompileProgram(ruleExpr string) (cel.Program, error) {
	c.mu.RLock()
	prog, exists := c.programs[ruleExpr]
	c.mu.RUnlock()
	if exists {
		return prog, nil
	}

	ast, issues := c.env.Compile(ruleExpr)
	if issues.Err() != nil {
		return nil, fmt.Errorf("cel compilation error: %w", issues.Err())
	}

	prog, err := c.env.Program(ast)
	if err != nil {
		return nil, fmt.Errorf("cel program generation error: %w", err)
	}

	c.mu.Lock()
	c.programs[ruleExpr] = prog
	c.mu.Unlock()

	return prog, nil
}

// EvaluateRule evaluates a boolean rule expression against dynamic input variables.
func (c *CELEvaluator) EvaluateRule(ruleExpr string, inputVars map[string]interface{}) (bool, error) {
	prog, err := c.CompileProgram(ruleExpr)
	if err != nil {
		return false, err
	}

	out, _, err := prog.Eval(inputVars)
	if err != nil {
		return false, fmt.Errorf("cel evaluation error: %w", err)
	}

	val := out.Value()
	boolResult, ok := val.(bool)
	if !ok {
		return false, fmt.Errorf("cel expression did not return a boolean, got %T (%v)", val, val)
	}

	return boolResult, nil
}

// EvaluateBatch evaluates a boolean rule across a slice of dynamic attribute records.
func (c *CELEvaluator) EvaluateBatch(ruleExpr string, records []map[string]interface{}) ([]bool, error) {
	prog, err := c.CompileProgram(ruleExpr)
	if err != nil {
		return nil, err
	}

	results := make([]bool, len(records))
	for i, row := range records {
		out, _, err := prog.Eval(row)
		if err != nil {
			return nil, fmt.Errorf("cel evaluation error at index %d: %w", i, err)
		}
		boolVal, ok := out.Value().(bool)
		if !ok {
			return nil, fmt.Errorf("cel row %d did not return boolean", i)
		}
		results[i] = boolVal
	}

	return results, nil
}
