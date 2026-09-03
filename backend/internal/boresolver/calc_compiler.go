package boresolver

import (
	"fmt"
	"strings"
)

// CompileDeepCalculations generates the nested CTE plan for pushdown-tier
// calculated fields (see calc_functions.go for tier classification).
//
// Host-runtime tier fields — formulas whose functions cannot compile to SQL
// (e.g. XIRR, an iterative solver over a cash-flow series) — are excluded
// from the returned query and returned separately as hostNodes, in
// dependency order. The caller runs the returned SQL, then executes
// hostNodes in order via HostRuntimeExecutor (which composes them — a
// "calc in a calc" is fine when every node in the chain is host-runtime),
// and merges the results.
//
// Tier propagates: a formula that is otherwise pure SQL but references a
// term that itself resolved to host-runtime is ALSO host-runtime — that
// referenced value was never projected into the CTE, so it can't be a SQL
// column reference. This is "poisoning," and it's why hostNodes must be
// walked in the order given rather than per-layer in isolation.
//
// A pushdown formula depending on a host-runtime formula's output (the
// reverse direction — splicing a host-runtime result back INTO a later SQL
// layer) is not supported yet; that needs the host-runtime executor to
// materialize its result into the query engine (a temp table/parameter)
// before the next layer compiles, which is bigger scope (see the
// datapipeline precalc path) than the Go-side composition above.
//
// baseQuery is the pre-compiled string containing the Hot/Cold joins and
// Rule 7 Tenant Security boundary.
func (g *BOSQLGenerator) CompileDeepCalculations(layers [][]*CalcNode, baseQuery string, requestedFields []string) (sql string, hostNodes []*CalcNode, err error) {
	dialect := g.Dialect
	if dialect == nil {
		dialect = PostgresDialect{}
	}

	if len(layers) == 0 {
		return baseQuery, nil, nil
	}

	var sb strings.Builder

	// Layer 0: Base Data Extraction (Security & Isolation boundary)
	sb.WriteString("WITH layer_0 AS (\n  ")
	sb.WriteString(baseQuery)
	sb.WriteString("\n)")

	lastLayerIdx := 0
	hostRuntimeTermKeys := make(map[string]bool)
	for depth := 1; depth < len(layers); depth++ {
		var pushdown []*CalcNode
		for _, calc := range layers[depth] {
			expr := calc.ParsedExpr
			if expr == nil {
				expr, err = ParseCalcFormula(calc.Formula)
				if err != nil {
					return "", nil, fmt.Errorf("failed to parse formula for %s: %w", calc.TermKey, err)
				}
				calc.ParsedExpr = expr
			}

			tier, tierErr := ResolveTier(expr, dialect, calc.Preference)
			if tierErr != nil {
				return "", nil, fmt.Errorf("failed to resolve execution tier for %s: %w", calc.TermKey, tierErr)
			}

			// Tier propagation: a dependency that resolved to host-runtime
			// earlier in this compile poisons this node too, since that
			// dependency's value was never projected as a SQL column.
			if tier == TierPushdown {
				for _, ref := range collectTermRefs(expr) {
					if !hostRuntimeTermKeys[ref] {
						continue
					}
					if calc.Preference == PreferPushdown {
						return "", nil, fmt.Errorf("failed to resolve execution tier for %s: depends on host-runtime term %q, cannot be SQL pushdown", calc.TermKey, ref)
					}
					tier = TierHostRuntime
					break
				}
			}
			calc.Tier = tier

			if tier == TierHostRuntime {
				hostRuntimeTermKeys[calc.TermKey] = true
				hostNodes = append(hostNodes, calc)
				continue
			}
			pushdown = append(pushdown, calc)
		}

		if len(pushdown) == 0 {
			// Nothing to project at this layer (host-runtime-only) — carry
			// the previous layer forward rather than emit an empty CTE.
			continue
		}

		sb.WriteString(fmt.Sprintf(",\nlayer_%d AS (\n", depth))
		sb.WriteString("  SELECT *") // Carry forward lower-level variables

		for _, calc := range pushdown {
			exprSQL, rErr := renderCalcExpr(calc.ParsedExpr, dialect)
			if rErr != nil {
				return "", nil, fmt.Errorf("failed to compile formula for %s: %w", calc.TermKey, rErr)
			}
			sb.WriteString(fmt.Sprintf(",\n    (%s) AS %s", exprSQL, calc.TermKey))
		}

		sb.WriteString(fmt.Sprintf("\n  FROM layer_%d\n)", lastLayerIdx))
		lastLayerIdx = depth
	}

	// Final Select Projection. Host-runtime fields never appear in the SQL
	// projection — the caller computes and merges them via hostNodes.
	hostKeys := make(map[string]bool, len(hostNodes))
	for _, n := range hostNodes {
		hostKeys[n.TermKey] = true
	}

	var sqlFields []string
	for _, f := range requestedFields {
		if !hostKeys[f] {
			sqlFields = append(sqlFields, f)
		}
	}

	switch {
	case len(requestedFields) == 0, len(sqlFields) == 0:
		// Either no explicit projection was requested, or every requested
		// field is host-runtime — either way the caller needs the full row
		// set (host-runtime fields key off base/pushdown columns).
		sb.WriteString(fmt.Sprintf("\nSELECT * FROM layer_%d;", lastLayerIdx))
	default:
		sb.WriteString(fmt.Sprintf("\nSELECT\n  %s\nFROM layer_%d;", strings.Join(sqlFields, ",\n  "), lastLayerIdx))
	}

	return sb.String(), hostNodes, nil
}

// renderCalcExpr compiles a parsed calc-layer formula to dialect SQL. Unlike
// Resolver.ToSQL (used for semantic term resolution against physical
// tables), TermRef here refers to an already-projected column alias from a
// prior CTE layer, so it renders as a bare identifier rather than a
// table.column reference.
func renderCalcExpr(expr Expr, dialect Dialect) (string, error) {
	switch e := expr.(type) {
	case *NumberLiteral:
		return e.Value, nil
	case *StringLiteral:
		return dialect.QuoteLiteral(e.Value), nil
	case *TermRef:
		return e.Name, nil
	case *UnaryExpr:
		inner, err := renderCalcExpr(e.Expr, dialect)
		if err != nil {
			return "", err
		}
		switch e.Op {
		case "-":
			return fmt.Sprintf("(-%s)", inner), nil
		case "NOT":
			return fmt.Sprintf("(NOT %s)", inner), nil
		default:
			return "", fmt.Errorf("unknown unary operator: %s", e.Op)
		}
	case *BinaryExpr:
		left, err := renderCalcExpr(e.Left, dialect)
		if err != nil {
			return "", err
		}
		right, err := renderCalcExpr(e.Right, dialect)
		if err != nil {
			return "", err
		}
		if e.Op == "/" {
			return dialect.SafeDiv(left, right), nil
		}
		op := e.Op
		switch e.Op {
		case "+":
			op = dialect.OpAdd()
		case "-":
			op = dialect.OpSub()
		case "*":
			op = dialect.OpMul()
		case "!=", "<>":
			op = "<>"
		}
		return fmt.Sprintf("(%s %s %s)", left, op, right), nil
	case *FunctionCall:
		args := make([]string, len(e.Args))
		for i, a := range e.Args {
			s, err := renderCalcExpr(a, dialect)
			if err != nil {
				return "", err
			}
			args[i] = s
		}
		if spec, ok := lookupFunction(e.FunctionName); ok {
			if spec.Pushdown != nil && spec.Pushdown.Supports(dialect) {
				return spec.Pushdown.Render(dialect, args), nil
			}
			return "", fmt.Errorf("function %q has no SQL pushdown implementation for dialect %q", e.FunctionName, dialect.Name())
		}
		return dialect.Func(e.FunctionName, args...), nil
	default:
		return "", fmt.Errorf("unknown expression type: %T", expr)
	}
}
