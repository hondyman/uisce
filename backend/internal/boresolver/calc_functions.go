package boresolver

import (
	"fmt"
	"regexp"
)

// CalcTier classifies where a calculated field's formula executes for a
// given compile target. Tier is a function of (function, dialect), not a
// fixed property of the function alone — see FunctionSpec.
type CalcTier int

const (
	// TierPushdown formulas compile entirely to SQL and run inside the
	// query engine (StarRocks/Iceberg/Postgres/etc).
	TierPushdown CalcTier = iota
	// TierHostRuntime formulas cannot compile to SQL on the target dialect
	// (no native function/UDF, or an inherently iterative solver) and must
	// run in Go against materialized rows.
	TierHostRuntime
)

func (t CalcTier) String() string {
	switch t {
	case TierHostRuntime:
		return "host_runtime"
	default:
		return "pushdown"
	}
}

// PushdownBuilder lets a function declare a dialect-aware SQL rendering —
// e.g. once a native StarRocks XIRR UDF exists, registering one here makes
// the compiler prefer it on StarRocks while still falling back to the Go
// host-runtime adapter on dialects without that UDF (see FunctionSpec).
type PushdownBuilder struct {
	// Supports reports whether this dialect has a working SQL rendering for
	// the function at all.
	Supports func(dialect Dialect) bool
	// Render produces the SQL for a call, given already-rendered argument
	// SQL. Only called when Supports returned true.
	Render func(dialect Dialect, args []string) string
}

// hostRuntimeAdapter evaluates a host-runtime function's rows for one
// entity. argNames are the resolved term keys of the function call's
// arguments, in declared order (e.g. xirr(${amount}, ${date}) ->
// ["amount", "date"]). Defined here so FunctionSpec can reference it;
// implementations live in host_runtime_executor.go.
type hostRuntimeAdapter func(rows []CalcRow, argNames []string) (float64, error)

// FunctionSpec describes how one function name can execute: via a
// dialect-aware SQL pushdown builder, a Go host-runtime adapter, or both.
// A function with only HostRuntime always runs host-runtime, regardless of
// dialect. A function with both lets the compiler pick the best engine per
// dialect. A function absent from the registry is assumed to be a plain
// arithmetic/dialect builtin (abs, round, coalesce, ...) that always
// compiles via Dialect.Func — no entry needed.
type FunctionSpec struct {
	Pushdown    *PushdownBuilder
	HostRuntime hostRuntimeAdapter
}

// functionRegistry holds explicit execution strategies for functions that
// are NOT plain dialect-arithmetic passthroughs.
var functionRegistry = map[string]*FunctionSpec{
	"xirr": {HostRuntime: xirrAdapter},
	"irr":  {HostRuntime: irrAdapter},
}

// RegisterFunction adds or replaces a function's execution strategy. Used
// to teach the compiler about a newly available native UDF without
// touching CalcNode/CalcGraph/CompileDeepCalculations, e.g.:
//
//	RegisterFunction("xirr", &FunctionSpec{
//	    HostRuntime: xirrAdapter, // still used for dialects without the UDF
//	    Pushdown: &PushdownBuilder{
//	        Supports: func(d Dialect) bool { return d.Name() == "starrocks" },
//	        Render:   func(d Dialect, args []string) string { return fmt.Sprintf("XIRR(%s, %s)", args[0], args[1]) },
//	    },
//	})
func RegisterFunction(name string, spec *FunctionSpec) {
	functionRegistry[normalizeFuncName(name)] = spec
}

func lookupFunction(name string) (*FunctionSpec, bool) {
	spec, ok := functionRegistry[normalizeFuncName(name)]
	return spec, ok
}

func normalizeFuncName(name string) string {
	out := make([]byte, 0, len(name))
	for i := 0; i < len(name); i++ {
		c := name[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		out = append(out, c)
	}
	return string(out)
}

var calcTermRefRegex = regexp.MustCompile(`\$\{([a-zA-Z0-9_]+)\}`)

// ParseCalcFormula parses a CalcNode.Formula (which uses "${term}" reference
// syntax) into an Expr AST by first rewriting term references to bare
// identifiers accepted by the expression parser.
func ParseCalcFormula(formula string) (Expr, error) {
	rewritten := calcTermRefRegex.ReplaceAllString(formula, "$1")
	return ParseExpression(rewritten)
}

// ExecutionPreference lets a calc definition steer engine selection instead
// of always taking whatever ResolveTier's auto mode picks. This matters on
// a multi-tenant platform where the same term key can mean two different
// things: an interactive preview (fine with whatever's fastest) versus a
// published/"official" performance number (must always come from one
// consistent, auditable engine, even if a faster pushdown path exists).
type ExecutionPreference int

const (
	// PreferAuto picks SQL pushdown if the target dialect supports every
	// function in the formula, else falls back to host-runtime. Default.
	PreferAuto ExecutionPreference = iota
	// PreferPushdown requires SQL pushdown; ResolveTier returns an error if
	// the target dialect can't fully render the formula, rather than
	// silently downgrading to host-runtime.
	PreferPushdown
	// PreferHostRuntime always evaluates via the Go host-runtime executor,
	// even when a pushdown implementation exists for the target dialect —
	// e.g. to guarantee every tenant's published IRR was computed by the
	// same audited finlib code path regardless of which engine served it.
	PreferHostRuntime
)

// ResolveTier decides the execution tier for a parsed formula against one
// specific dialect, honoring pref. This is where "best engine" selection
// actually happens — see FunctionSpec and RegisterFunction.
func ResolveTier(expr Expr, dialect Dialect, pref ExecutionPreference) (CalcTier, error) {
	pushdownCapable := canPushdownFully(expr, dialect)

	switch pref {
	case PreferHostRuntime:
		return TierHostRuntime, nil
	case PreferPushdown:
		if !pushdownCapable {
			return TierPushdown, fmt.Errorf("formula requires SQL pushdown (PreferPushdown) but dialect %q has no implementation for one or more functions in it", dialect.Name())
		}
		return TierPushdown, nil
	default: // PreferAuto
		if pushdownCapable {
			return TierPushdown, nil
		}
		return TierHostRuntime, nil
	}
}

// CollectTermRefs returns every term key a formula references, in
// occurrence order (used both for tier propagation in calc_compiler.go and
// for deciding what a host-runtime node still needs to fetch vs. reuse
// from an earlier node's result in host_runtime_executor.go). Exported so
// callers building a CalcGraph from stored calc definitions (see
// analytics.SemanticCalculationService.BuildCalcGraph) can resolve
// calc-in-calc dependency chains the same way.
func CollectTermRefs(expr Expr) []string {
	return collectTermRefs(expr)
}

func collectTermRefs(expr Expr) []string {
	var out []string
	var walk func(Expr)
	walk = func(e Expr) {
		switch n := e.(type) {
		case *TermRef:
			out = append(out, n.Name)
		case *FunctionCall:
			for _, arg := range n.Args {
				walk(arg)
			}
		case *BinaryExpr:
			walk(n.Left)
			walk(n.Right)
		case *UnaryExpr:
			walk(n.Expr)
		}
	}
	walk(expr)
	return out
}

func canPushdownFully(expr Expr, dialect Dialect) bool {
	switch e := expr.(type) {
	case *FunctionCall:
		if spec, ok := lookupFunction(e.FunctionName); ok {
			if spec.Pushdown == nil || !spec.Pushdown.Supports(dialect) {
				return false
			}
		}
		for _, arg := range e.Args {
			if !canPushdownFully(arg, dialect) {
				return false
			}
		}
		return true
	case *BinaryExpr:
		return canPushdownFully(e.Left, dialect) && canPushdownFully(e.Right, dialect)
	case *UnaryExpr:
		return canPushdownFully(e.Expr, dialect)
	default:
		return true
	}
}
