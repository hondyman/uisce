package policy

import (
	"context"
	"fmt"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
)

// CELEvaluator evaluates CEL expressions for eligibility and ranking
type CELEvaluator struct {
	env *cel.Env
}

// NewCELEvaluator creates a CEL evaluator with WealthStream host functions
func NewCELEvaluator() (*CELEvaluator, error) {
	env, err := cel.NewEnv(
		cel.Variable("client", cel.DynType),
		cel.Variable("holdings", cel.ListType(cel.DynType)),
		cel.Variable("symbol", cel.StringType),
		cel.Variable("dividend_date", cel.TimestampType),
		cel.Variable("lastDelivered", cel.TimestampType),
		cel.Variable("threshold", cel.DoubleType),
		cel.Function("recency",
			cel.Overload("recency_timestamp",
				[]*cel.Type{cel.TimestampType},
				cel.IntType,
				cel.UnaryBinding(recencyImpl))),
		cel.Function("portfolio_weight",
			cel.Overload("portfolio_weight_string_list",
				[]*cel.Type{cel.StringType, cel.ListType(cel.DynType)},
				cel.DoubleType,
				cel.BinaryBinding(portfolioWeightImpl))),
	)
	if err != nil {
		return nil, err
	}
	return &CELEvaluator{env: env}, nil
}

// EvalBool evaluates a boolean CEL expression
func (e *CELEvaluator) EvalBool(ctx context.Context, expr string, vars map[string]interface{}) (bool, error) {
	ast, issues := e.env.Compile(expr)
	if issues != nil && issues.Err() != nil {
		return false, fmt.Errorf("compile error: %w", issues.Err())
	}

	prg, err := e.env.Program(ast)
	if err != nil {
		return false, fmt.Errorf("program error: %w", err)
	}

	out, _, err := prg.Eval(vars)
	if err != nil {
		return false, fmt.Errorf("eval error: %w", err)
	}

	if out == nil {
		return false, fmt.Errorf("nil result")
	}

	b, ok := out.Value().(bool)
	if !ok {
		return false, fmt.Errorf("expr did not return bool: %T", out.Value())
	}

	return b, nil
}

// EvalNumber evaluates a numeric CEL expression for ranking
func (e *CELEvaluator) EvalNumber(ctx context.Context, expr string, vars map[string]interface{}) (float64, error) {
	ast, issues := e.env.Compile(expr)
	if issues != nil && issues.Err() != nil {
		return 0, fmt.Errorf("compile error: %w", issues.Err())
	}

	prg, err := e.env.Program(ast)
	if err != nil {
		return 0, fmt.Errorf("program error: %w", err)
	}

	out, _, err := prg.Eval(vars)
	if err != nil {
		return 0, fmt.Errorf("eval error: %w", err)
	}

	switch v := out.Value().(type) {
	case int64:
		return float64(v), nil
	case float64:
		return v, nil
	case types.Double:
		return float64(v), nil
	default:
		return 0, fmt.Errorf("expr did not return numeric: %T", out.Value())
	}
}

// recencyImpl calculates days since a timestamp (0 if future)
func recencyImpl(val ref.Val) ref.Val {
	t, ok := val.Value().(time.Time)
	if !ok {
		return types.IntZero
	}

	now := time.Now().UTC()
	days := int64(now.Sub(t).Hours() / 24.0)
	if days < 0 {
		days = 0
	}

	return types.Int(days)
}

// portfolioWeightImpl calculates the weight of a symbol in holdings
func portfolioWeightImpl(symbol, holdings ref.Val) ref.Val {
	sym, ok := symbol.Value().(string)
	if !ok {
		return types.Double(0.0)
	}

	var totalWeight, symbolWeight float64

	// Preferred: CEL list values implement traits.Lister.
	if l, ok := holdings.(traits.Lister); ok {
		it := l.Iterator()
		for it.HasNext() == types.True {
			item := it.Next()

			weight := lookupFloatField(item, "weight")
			totalWeight += weight

			if holdingSym := lookupStringField(item, "symbol"); holdingSym == sym {
				symbolWeight += weight
			}
		}

		if totalWeight == 0 {
			return types.Double(0.0)
		}
		return types.Double(symbolWeight / totalWeight)
	}

	// Fallback: native Go values (useful in unit tests / callers passing plain maps).
	switch h := holdings.Value().(type) {
	case []interface{}:
		for _, item := range h {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			weight := toFloat64(m["weight"])
			totalWeight += weight
			if holdingSym, ok := m["symbol"].(string); ok && holdingSym == sym {
				symbolWeight += weight
			}
		}
	case []map[string]interface{}:
		for _, m := range h {
			weight := toFloat64(m["weight"])
			totalWeight += weight
			if holdingSym, ok := m["symbol"].(string); ok && holdingSym == sym {
				symbolWeight += weight
			}
		}
	default:
		return types.Double(0.0)
	}

	if totalWeight == 0 {
		return types.Double(0.0)
	}
	return types.Double(symbolWeight / totalWeight)
}

func toFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case types.Double:
		return float64(val)
	case types.Int:
		return float64(val)
	case types.Uint:
		return float64(val)
	case int64:
		return float64(val)
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case uint:
		return float64(val)
	case uint64:
		return float64(val)
	default:
		return 0.0
	}
}

func lookupStringField(item ref.Val, key string) string {
	// CEL map values implement traits.Mapper.
	if m, ok := item.(traits.Mapper); ok {
		v := m.Get(types.String(key))
		if v == nil {
			return ""
		}
		if s, ok := v.Value().(string); ok {
			return s
		}
		if ss, ok := v.(types.String); ok {
			return string(ss)
		}
		return ""
	}

	// Native map fallback.
	if mm, ok := item.Value().(map[string]interface{}); ok {
		if s, ok := mm[key].(string); ok {
			return s
		}
	}
	return ""
}

func lookupFloatField(item ref.Val, key string) float64 {
	// CEL map values implement traits.Mapper.
	if m, ok := item.(traits.Mapper); ok {
		v := m.Get(types.String(key))
		if v == nil {
			return 0.0
		}
		return toFloat64(v)
	}

	// Native map fallback.
	if mm, ok := item.Value().(map[string]interface{}); ok {
		return toFloat64(mm[key])
	}
	return 0.0
}
