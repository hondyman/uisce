package reporting

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/ext"
)

// ─────────────────────────────────────────────────────────────────────────────
// HybridEvaluator evaluates non-pushable ExprNode predicates against row data
// in-memory using the Google CEL (Common Expression Language) runtime.
//
// Architecture:
//  1. CompileExpression(node) — converts ExprNode to CEL source string, compiles
//     to a CEL program once (thread-safe for repeated evaluation).
//  2. Evaluate(program, row) — evaluates the compiled program against a row map.
//
// The CEL environment declares the full set of supported functions and variable
// types, enabling safe sandboxed expression evaluation without SQL injection risk.
// ─────────────────────────────────────────────────────────────────────────────

// RowMap is a string-keyed map of arbitrary column values from a fetched row.
type RowMap map[string]interface{}

// HybridEvaluator compiles non-pushable ExprNode predicates for in-memory evaluation.
type HybridEvaluator struct {
	env *cel.Env
}

// NewHybridEvaluator creates a CEL evaluator with the Uisce expression environment.
func NewHybridEvaluator(fieldNames []string) (*HybridEvaluator, error) {
	// Build variable declarations from field names
	vars := make([]cel.EnvOption, 0, len(fieldNames)+5)
	for _, fn := range fieldNames {
		vars = append(vars, cel.Variable(fn, cel.DynType))
	}
	// Standard session variables always available
	vars = append(vars,
		cel.Variable("_tenant_id", cel.StringType),
		cel.Variable("_user_id", cel.StringType),
		cel.Variable("_as_of_date", cel.StringType),
		cel.Variable("_now", cel.StringType),
		ext.Strings(),
	)

	env, err := cel.NewEnv(vars...)
	if err != nil {
		return nil, fmt.Errorf("hybrid evaluator: build CEL env: %w", err)
	}
	return &HybridEvaluator{env: env}, nil
}

// CompiledPredicate is a pre-compiled, thread-safe CEL predicate.
type CompiledPredicate struct {
	program cel.Program
	celExpr string
}

// CompileExpression converts an ExprNode predicate to a CompiledPredicate.
func (h *HybridEvaluator) CompileExpression(node *ExprNode) (*CompiledPredicate, error) {
	celExpr := exprToCEL(node)
	ast, issues := h.env.Compile(celExpr)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("hybrid evaluator: compile %q: %w", celExpr, issues.Err())
	}
	prg, err := h.env.Program(ast)
	if err != nil {
		return nil, fmt.Errorf("hybrid evaluator: create program: %w", err)
	}
	return &CompiledPredicate{program: prg, celExpr: celExpr}, nil
}

// Evaluate runs a compiled predicate against a row map.
// Returns true if the row passes the predicate, false if it is filtered out.
// Errors are treated conservatively — the row is excluded on evaluation error.
func (h *HybridEvaluator) Evaluate(pred *CompiledPredicate, row RowMap, ctx *ExecutionContext) (bool, error) {
	activation := buildActivation(row, ctx)
	out, _, err := pred.program.Eval(activation)
	if err != nil {
		return false, fmt.Errorf("hybrid eval: %w", err)
	}
	boolVal, ok := out.(types.Bool)
	if !ok {
		// Non-boolean result — treat as falsy
		return false, nil
	}
	return bool(boolVal), nil
}

// FilterRows applies all compiled predicates to a slice of rows, returning only
// rows that pass ALL predicates (AND semantics for hybrid filters).
func (h *HybridEvaluator) FilterRows(preds []*CompiledPredicate, rows []RowMap, ctx *ExecutionContext) ([]RowMap, error) {
	if len(preds) == 0 {
		return rows, nil
	}
	result := make([]RowMap, 0, len(rows))
	for _, row := range rows {
		pass := true
		for _, pred := range preds {
			ok, err := h.Evaluate(pred, row, ctx)
			if err != nil || !ok {
				pass = false
				break
			}
		}
		if pass {
			result = append(result, row)
		}
	}
	return result, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// ExprNode → CEL source string conversion
// ─────────────────────────────────────────────────────────────────────────────

func exprToCEL(node *ExprNode) string {
	if node == nil {
		return "null"
	}
	switch node.Kind {
	case ExprLiteral:
		return celLiteral(node.Literal)
	case ExprField:
		if node.FieldRef != nil {
			// CEL uses plain identifiers for variables
			return sanitizeCELIdent(node.FieldRef.Column)
		}
		return "null"
	case ExprParam:
		if node.ParamRef != nil {
			// Params already resolved to literals by MacroResolver at this point
			return sanitizeCELIdent(node.ParamRef.ParamName)
		}
		return "null"
	case ExprFunction:
		return celFunction(node)
	case ExprBinaryOp:
		return celBinaryOp(node)
	case ExprUnaryOp:
		return celUnaryOp(node)
	case ExprCase:
		return celCase(node)
	case ExprAggregate:
		// Aggregates should not reach the hybrid evaluator — but handle gracefully
		return fmt.Sprintf("/* aggregate: %s */ null", node.AggFunc)
	default:
		return "null"
	}
}

func celLiteral(lit *LiteralExpr) string {
	if lit == nil {
		return "null"
	}
	if lit.StrVal != nil {
		return fmt.Sprintf("%q", *lit.StrVal)
	}
	if lit.NumVal != nil {
		return fmt.Sprintf("%g", *lit.NumVal)
	}
	if lit.BoolVal != nil {
		if *lit.BoolVal {
			return "true"
		}
		return "false"
	}
	if lit.DateVal != nil {
		return fmt.Sprintf("%q", *lit.DateVal)
	}
	return "null"
}

func celFunction(node *ExprNode) string {
	fn := strings.ToLower(node.FuncName)
	args := make([]string, len(node.Args))
	for i, a := range node.Args {
		args[i] = exprToCEL(a)
	}
	join := strings.Join(args, ", ")
	switch fn {
	case "upper":
		if len(args) > 0 {
			return fmt.Sprintf("string(%s).upperAscii()", args[0])
		}
	case "lower":
		if len(args) > 0 {
			return fmt.Sprintf("string(%s).lowerAscii()", args[0])
		}
	case "trim":
		if len(args) > 0 {
			return fmt.Sprintf("string(%s).trim()", args[0])
		}
	case "substr", "substring":
		if len(args) == 3 {
			return fmt.Sprintf("string(%s).substring(int(%s), int(%s)+int(%s))", args[0], args[1], args[1], args[2])
		}
	case "contains":
		if len(args) == 2 {
			return fmt.Sprintf("string(%s).contains(string(%s))", args[0], args[1])
		}
	case "regexp_like":
		if len(args) == 2 {
			return fmt.Sprintf("%s.matches(%s)", args[0], args[1])
		}
	case "coalesce":
		// CEL: first non-null — approximate with ternary chain
		if len(args) == 0 {
			return "null"
		}
		result := args[len(args)-1]
		for i := len(args) - 2; i >= 0; i-- {
			result = fmt.Sprintf("(%s != null ? %s : %s)", args[i], args[i], result)
		}
		return result
	case "round":
		// CEL doesn't have a built-in round with precision, approximate
		if len(args) == 1 {
			return fmt.Sprintf("math.round(%s)", args[0])
		}
		return fmt.Sprintf("math.round(%s)", args[0]) // precision ignored in CEL
	case "abs":
		if len(args) == 1 {
			return fmt.Sprintf("math.abs(%s)", args[0])
		}
	}
	// Generic function call — CEL extension point
	return fmt.Sprintf("%s(%s)", fn, join)
}

func celBinaryOp(node *ExprNode) string {
	left := exprToCEL(node.Left)
	right := exprToCEL(node.Right)
	op := node.Op
	switch strings.ToUpper(op) {
	case "=":
		op = "=="
	case "!=", "<>":
		op = "!="
	case "AND":
		op = "&&"
	case "OR":
		op = "||"
	case "IN":
		return fmt.Sprintf("%s in [%s]", left, right)
	case "NOT IN":
		return fmt.Sprintf("!(%s in [%s])", left, right)
	case "LIKE", "ILIKE":
		// Approximate LIKE with contains (CEL doesn't have SQL LIKE)
		pattern := strings.Trim(right, `"'`)
		if strings.HasPrefix(pattern, "%") && strings.HasSuffix(pattern, "%") {
			inner := strings.Trim(pattern, "%")
			return fmt.Sprintf(`%s.contains("%s")`, left, inner)
		}
		if strings.HasPrefix(pattern, "%") {
			inner := strings.TrimPrefix(pattern, "%")
			return fmt.Sprintf(`%s.endsWith("%s")`, left, inner)
		}
		if strings.HasSuffix(pattern, "%") {
			inner := strings.TrimSuffix(pattern, "%")
			return fmt.Sprintf(`%s.startsWith("%s")`, left, inner)
		}
		return fmt.Sprintf(`%s == "%s"`, left, pattern)
	}
	return fmt.Sprintf("(%s %s %s)", left, op, right)
}

func celUnaryOp(node *ExprNode) string {
	inner := exprToCEL(node.Operand)
	switch strings.ToUpper(node.Op) {
	case "NOT":
		return fmt.Sprintf("!(%s)", inner)
	case "-":
		return fmt.Sprintf("-(%s)", inner)
	case "IS NULL":
		return fmt.Sprintf("%s == null", inner)
	case "IS NOT NULL":
		return fmt.Sprintf("%s != null", inner)
	}
	return fmt.Sprintf("%s %s", node.Op, inner)
}

func celCase(node *ExprNode) string {
	if len(node.CaseWhen) == 0 {
		return exprToCEL(node.CaseElse)
	}
	// Build nested ternary: (when ? then : (when2 ? then2 : else))
	result := "null"
	if node.CaseElse != nil {
		result = exprToCEL(node.CaseElse)
	}
	for i := len(node.CaseWhen) - 1; i >= 0; i-- {
		w := node.CaseWhen[i]
		result = fmt.Sprintf("(%s ? %s : %s)", exprToCEL(w.When), exprToCEL(w.Then), result)
	}
	return result
}

func sanitizeCELIdent(s string) string {
	// Replace non-identifier chars with underscore
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			return r
		}
		return '_'
	}, strings.TrimPrefix(s, "@"))
}

// buildActivation builds the CEL variable activation from a row map.
func buildActivation(row RowMap, ctx *ExecutionContext) map[string]interface{} {
	act := make(map[string]interface{}, len(row)+4)
	for k, v := range row {
		act[sanitizeCELIdent(k)] = toCELValue(v)
	}
	// Session context variables
	act["_tenant_id"] = ctx.TenantID
	act["_user_id"] = ctx.UserID
	act["_as_of_date"] = ctx.AsOfDate.Format("2006-01-02")
	act["_now"] = ctx.Now.Format(time.RFC3339)
	return act
}

// toCELValue converts a Go value to a CEL-compatible native value.
func toCELValue(v interface{}) interface{} {
	if v == nil {
		return ref.Val(types.NullValue)
	}
	return v
}
