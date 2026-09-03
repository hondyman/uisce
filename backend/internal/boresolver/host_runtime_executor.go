package boresolver

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hondyman/uisce/backend/internal/boresolver/finlib"
)

// CalcRow is one row of resolved base-field values for a host-runtime calc,
// keyed by term key.
type CalcRow map[string]any

// RowSource fetches the rows a host-runtime CalcNode needs, scoped to one
// tenant and grouped by entity (whatever grain the calc is defined against —
// a fund, a position, a customer). Implementations MUST enforce the same
// tenant isolation boundary the pushdown SQL layer enforces (Rule 7) —
// HostRuntimeExecutor does not itself filter by tenant.
type RowSource interface {
	// FetchRows returns rows grouped by entity ID. termKeys are the base
	// field dependencies the calc's formula needs (e.g. cashflow_amount,
	// cashflow_date), resolved the same way the pushdown layer resolves
	// base fields.
	FetchRows(ctx context.Context, tenantID string, termKeys []string) (map[string][]CalcRow, error)
}

// HostRuntimeResult is the outcome of evaluating one host-runtime CalcNode
// for one entity. Err is per-entity (e.g. a fund with no sign change in its
// cash flows) rather than failing the whole batch.
type HostRuntimeResult struct {
	TermKey  string
	EntityID string
	Value    float64
	Err      error
}

// hostRuntimeAdapter is declared in calc_functions.go (as part of
// FunctionSpec); implementations follow.

func xirrAdapter(rows []CalcRow, argNames []string) (float64, error) {
	if len(argNames) != 2 {
		return 0, fmt.Errorf("xirr expects 2 arguments (amount, date), got %d", len(argNames))
	}
	amountKey, dateKey := argNames[0], argNames[1]

	flows := make([]finlib.Cashflow, 0, len(rows))
	for _, row := range rows {
		amount, err := toFloat64(row[amountKey])
		if err != nil {
			return 0, fmt.Errorf("xirr: %s: %w", amountKey, err)
		}
		date, err := toTime(row[dateKey])
		if err != nil {
			return 0, fmt.Errorf("xirr: %s: %w", dateKey, err)
		}
		flows = append(flows, finlib.Cashflow{Date: date, Amount: amount})
	}
	return finlib.XIRR(flows, 0.1)
}

func irrAdapter(rows []CalcRow, argNames []string) (float64, error) {
	if len(argNames) != 1 {
		return 0, fmt.Errorf("irr expects 1 argument (amount), got %d", len(argNames))
	}
	amountKey := argNames[0]

	flows := make([]float64, 0, len(rows))
	for _, row := range rows {
		amount, err := toFloat64(row[amountKey])
		if err != nil {
			return 0, fmt.Errorf("irr: %s: %w", amountKey, err)
		}
		flows = append(flows, amount)
	}
	return finlib.IRR(flows, 0.1)
}

// HostRuntimeExecutor evaluates the host-runtime CalcNodes cut out of
// CompileDeepCalculations (see calc_compiler.go). It is the single
// implementation of how a host-runtime calc actually runs: both the
// on-demand (interactive, per-request) caller and the precalc (pipeline,
// batch/materialization) caller must go through this executor, so an
// interactive preview and a pipeline-published "official" value are always
// computed identically.
type HostRuntimeExecutor struct {
	Rows RowSource
}

// Execute evaluates each host-runtime node against its dependency rows,
// grouped by entity, IN THE ORDER GIVEN (callers must pass nodes in
// dependency order — CompileDeepCalculations already does, since it walks
// CalcGraph's topologically sorted layers). This is what makes "a calc in a
// calc" work for host-runtime formulas: once a node's per-entity scalar
// result is computed, it's kept in resultsByEntity so a LATER node in the
// same call can reference it by TermRef (e.g. "adjusted_xirr" referencing
// "${customer_xirr} - ${hurdle_rate}") without re-fetching or re-running
// the earlier calc.
//
// tenantID scopes RowSource.FetchRows — callers must never share a
// RowSource across tenants without tenant scoping enforced inside it.
func (x *HostRuntimeExecutor) Execute(ctx context.Context, tenantID string, nodes []*CalcNode) ([]HostRuntimeResult, error) {
	if x.Rows == nil {
		return nil, fmt.Errorf("HostRuntimeExecutor: RowSource is required")
	}
	if tenantID == "" {
		return nil, fmt.Errorf("HostRuntimeExecutor: tenantID is required")
	}

	// resultsByEntity[entityID][termKey] holds every earlier node's scalar
	// result for that entity, so later nodes can reference them.
	resultsByEntity := make(map[string]map[string]float64)
	priorTermKeys := make(map[string]bool)

	var results []HostRuntimeResult
	for _, node := range nodes {
		expr := node.ParsedExpr
		if expr == nil {
			var err error
			expr, err = ParseCalcFormula(node.Formula)
			if err != nil {
				return nil, fmt.Errorf("failed to parse formula for %s: %w", node.TermKey, err)
			}
		}

		// Only fetch what isn't already a computed result from an earlier
		// node in this same call — that's the "calc in a calc" wiring.
		var toFetch []string
		for _, ref := range collectTermRefs(expr) {
			if !priorTermKeys[ref] {
				toFetch = append(toFetch, ref)
			}
		}

		var rowsByEntity map[string][]CalcRow
		if len(toFetch) > 0 {
			var err error
			rowsByEntity, err = x.Rows.FetchRows(ctx, tenantID, toFetch)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch rows for %s: %w", node.TermKey, err)
			}
		}

		for entityID := range unionEntityKeys(rowsByEntity, resultsByEntity) {
			value, evalErr := EvalHostExpr(expr, rowsByEntity[entityID], resultsByEntity[entityID])
			results = append(results, HostRuntimeResult{
				TermKey:  node.TermKey,
				EntityID: entityID,
				Value:    value,
				Err:      evalErr,
			})
			if evalErr == nil {
				if resultsByEntity[entityID] == nil {
					resultsByEntity[entityID] = make(map[string]float64)
				}
				resultsByEntity[entityID][node.TermKey] = value
			}
		}

		priorTermKeys[node.TermKey] = true
	}
	return results, nil
}

func unionEntityKeys(rowsByEntity map[string][]CalcRow, resultsByEntity map[string]map[string]float64) map[string]struct{} {
	out := make(map[string]struct{}, len(rowsByEntity)+len(resultsByEntity))
	for id := range rowsByEntity {
		out[id] = struct{}{}
	}
	for id := range resultsByEntity {
		out[id] = struct{}{}
	}
	return out
}

// EvalHostExpr evaluates a parsed formula for one entity, given the raw
// series rows fetched for it (used by aggregate functions like xirr/irr,
// which reduce a whole series to a scalar) and scalarCtx (earlier nodes'
// already-computed scalar results for this entity, keyed by TermKey — see
// Execute). A FunctionCall is dispatched to its registered host-runtime
// adapter over the row series; everything else (arithmetic, term
// references) is evaluated as ordinary scalar math, which is what lets a
// host-runtime formula wrap a calc-in-a-calc reference in more math, e.g.
// "${customer_xirr} - ${hurdle_rate}".
func EvalHostExpr(expr Expr, rows []CalcRow, scalarCtx map[string]float64) (float64, error) {
	switch e := expr.(type) {
	case *NumberLiteral:
		return strconv.ParseFloat(e.Value, 64)

	case *TermRef:
		if v, ok := scalarCtx[e.Name]; ok {
			return v, nil
		}
		if len(rows) > 0 {
			if v, ok := rows[0][e.Name]; ok {
				return toFloat64(v)
			}
		}
		return 0, fmt.Errorf("unresolved term reference: %s", e.Name)

	case *UnaryExpr:
		inner, err := EvalHostExpr(e.Expr, rows, scalarCtx)
		if err != nil {
			return 0, err
		}
		if e.Op == "-" {
			return -inner, nil
		}
		return 0, fmt.Errorf("unsupported unary operator in host-runtime expression: %s", e.Op)

	case *BinaryExpr:
		left, err := EvalHostExpr(e.Left, rows, scalarCtx)
		if err != nil {
			return 0, err
		}
		right, err := EvalHostExpr(e.Right, rows, scalarCtx)
		if err != nil {
			return 0, err
		}
		switch e.Op {
		case "+":
			return left + right, nil
		case "-":
			return left - right, nil
		case "*":
			return left * right, nil
		case "/":
			if right == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			return left / right, nil
		default:
			return 0, fmt.Errorf("unsupported binary operator in host-runtime expression: %s", e.Op)
		}

	case *FunctionCall:
		spec, ok := lookupFunction(e.FunctionName)
		if !ok || spec.HostRuntime == nil {
			return 0, fmt.Errorf("no host-runtime adapter registered for function %q", e.FunctionName)
		}
		argNames := make([]string, len(e.Args))
		for i, a := range e.Args {
			ref, ok := a.(*TermRef)
			if !ok {
				return 0, fmt.Errorf("function %q: argument %d must be a term reference, got %T", e.FunctionName, i, a)
			}
			argNames[i] = ref.Name
		}
		return spec.HostRuntime(rows, argNames)

	default:
		return 0, fmt.Errorf("unsupported expression type in host-runtime evaluation: %T", expr)
	}
}

func toFloat64(v any) (float64, error) {
	switch n := v.(type) {
	case float64:
		return n, nil
	case float32:
		return float64(n), nil
	case int:
		return float64(n), nil
	case int32:
		return float64(n), nil
	case int64:
		return float64(n), nil
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", v)
	}
}

func toTime(v any) (time.Time, error) {
	switch t := v.(type) {
	case time.Time:
		return t, nil
	case string:
		if parsed, err := time.Parse("2006-01-02", t); err == nil {
			return parsed, nil
		}
		if parsed, err := time.Parse(time.RFC3339, t); err == nil {
			return parsed, nil
		}
		return time.Time{}, fmt.Errorf("cannot parse %q as a date", t)
	default:
		return time.Time{}, fmt.Errorf("cannot convert %T to time.Time", v)
	}
}
