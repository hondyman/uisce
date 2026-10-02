package archguard

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/hondyman/uisce/backend/internal/querybuilder"
)

// C0 freeze on the metric compiler.
//
// Three expression systems exist: the metric compiler (A), internal/rules/vm
// (B, the intended survivor), and internal/calcengine (C). C1-C3 unify the
// metric compiler onto vm. Until that lands, the metric compiler's expression
// surface is FROZEN: it may not gain an operator, a function, a parser, or a
// second evaluator. New expression capability belongs in internal/rules/vm.
//
// The freeze is a guard, not a comment. It has three parts:
//
//  1. TestMetricCompilerFreezeIsNotObsolete - the freeze dies when C3 lands.
//  2. TestMetricCompilerFreezeBackstopHasNotPassed - a date backstop, so the
//     freeze cannot outlive its reason without a deliberate human decision.
//  3. TestMetricCompilerExpressionSurfaceIsFrozen - the surface itself.
//
// Why the behavioural part and not just a structural one: the compiler has no
// operator dispatch to enumerate. compileFormula tokenizes on whitespace and
// substitutes @var, passing every other token straight through. So the
// enforceable invariant is the tokenization contract, pinned by golden cases.
//
// THE GOLDEN CASES PIN CURRENT BEHAVIOUR, INCLUDING BEHAVIOUR THAT IS WRONG.
// metricFormulaUnspacedOperatorIsOneToken is the sharpest one: "@a*@b" is a
// single whitespace token that starts with "@", so it is looked up as a
// variable named "a*@b", not found, and falls through to the neutral 1.0
// multiplier - the multiplication is silently dropped and the metric compiles
// to a constant. That is a real defect. It is pinned here on purpose: a freeze
// records what the code does, and changing it must be a decision (lift the
// freeze, or amend the case deliberately) rather than a side effect. Fixing it
// is C1-C3 work, not something to smuggle through this guard.

const (
	// metricCompilerPath is backend-relative.
	metricCompilerPath = "internal/querybuilder/metric_compiler.go"

	// c3GateArtifact is the evidence that lifts this freeze: the C3
	// golden-corpus byte-identical gate, by the path agreed for it. The
	// primary expiry trigger is this artifact landing, not a calendar date -
	// the same reasoning as ADR-020, applied to the freeze itself.
	c3GateArtifact = "internal/querybuilder/metric_compiler_c3_gate_test.go"

	// metricCompilerFreezeBackstop is a backstop, not the trigger. If C3 has
	// not landed by this date the guard fails, forcing someone to either
	// extend it deliberately or lift it. A freeze that expires silently is
	// worse than no freeze.
	metricCompilerFreezeBackstop = "2026-12-31"
)

var metricCompilerFreeze = allowance{
	Reason: "C0 freeze: the metric compiler's expression surface is frozen while " +
		"C1-C3 unify it onto internal/rules/vm. New operators, functions, parsers " +
		"or a second evaluator wait for C3.",
	Until: "C3",
}

// metricCompilerFuncPins is the file's declared top-level func surface.
// Methods are receiver-qualified so a method cannot be smuggled in by
// shadowing a pinned function name on another type.
var metricCompilerFuncPins = []string{
	"(*MetricCompiler).CompileMetric",
	"(*MetricCompiler).compileFormula",
	"(*MetricCompiler).compileMetricWithCycleDetection",
	"ComputeQueryAndMetricsAndCubeCacheKey",
	"ComputeQueryAndMetricsCacheKey",
	"NewMetricCompiler",
	"sanitizeIdentifier",
}

// metricCompilerImportPins is the file's import set. C2 adds the vm resolver,
// so this pin is expected to break when C2 starts - that is the freeze doing
// its job, not a bug in it.
var metricCompilerImportPins = []string{
	"crypto/sha256",
	"encoding/hex",
	"fmt",
	"github.com/hondyman/uisce/backend/internal/boresolver",
	"sort",
	"strings",
}

func TestMetricCompilerFreezeIsNotObsolete(t *testing.T) {
	root := backendRoot(t)
	gate := filepath.Join(root, c3GateArtifact)
	if _, err := os.Stat(gate); err == nil {
		t.Fatalf("C3 gate is present at %s, so the C0 freeze is obsolete - delete metricCompilerFreeze, the golden cases and the surface pins in this file, and let the C3 gate police the surface instead", c3GateArtifact)
	}
}

func TestMetricCompilerFreezeBackstopHasNotPassed(t *testing.T) {
	backstop, err := time.Parse("2006-01-02", metricCompilerFreezeBackstop)
	if err != nil {
		t.Fatalf("backstop %q is not a valid date: %v", metricCompilerFreezeBackstop, err)
	}
	if time.Now().After(backstop) {
		t.Errorf("the C0 metric-compiler freeze backstop (%s) passed without C3 landing (%s). "+
			"Do not silence this by editing the date quietly: either extend it in this commit with a reason, "+
			"or lift the freeze and record in ADR-024 why the unification is no longer the priority. "+
			"Freeze reason: %s", metricCompilerFreezeBackstop, metricCompilerFreeze.Until, metricCompilerFreeze.Reason)
	}
}

type metricSurfaceCase struct {
	name     string
	expr     querybuilder.MetricExpression
	vars     []querybuilder.MetricVariable
	bindings map[string]interface{}
	lookup   map[string]querybuilder.MetricDefinition
	wantSQL  string
	wantArgs []string
	wantErr  string
}

func required(name string) querybuilder.MetricVariable {
	return querybuilder.MetricVariable{Name: name, Type: "number", Required: true}
}

func aggMetric(id, fn, term string) querybuilder.MetricDefinition {
	return querybuilder.MetricDefinition{
		ID:         id,
		Name:       id,
		Expression: querybuilder.MetricExpression{Kind: "aggregation", Fn: fn, TermNodeID: term},
	}
}

func TestMetricCompilerExpressionSurfaceIsFrozen(t *testing.T) {
	lookup := map[string]querybuilder.MetricDefinition{
		"m_revenue": aggMetric("m_revenue", "sum", "revenue"),
		"m_cost":    aggMetric("m_cost", "sum", "cost"),
		"m_units":   aggMetric("m_units", "sum", "units"),
	}

	cases := []metricSurfaceCase{
		{
			name:     "metricFormulaVariableBindsPositionalPlaceholder",
			expr:     querybuilder.MetricExpression{Kind: "formula", Formula: "@fx_rate * 2"},
			vars:     []querybuilder.MetricVariable{required("fx_rate")},
			bindings: map[string]interface{}{"fx_rate": 1.25},
			wantSQL:  "$1 * 2",
			wantArgs: []string{"1.25"},
		},
		{
			// The load-bearing freeze case. "@a*@b" is ONE whitespace token
			// that starts with "@", so it is looked up as a variable named
			// "a*@b", misses, and falls to the neutral 1.0 multiplier. The
			// multiplication is silently dropped. See the file comment.
			name:     "metricFormulaUnspacedOperatorIsOneToken",
			expr:     querybuilder.MetricExpression{Kind: "formula", Formula: "@a*@b"},
			vars:     []querybuilder.MetricVariable{required("a"), required("b")},
			bindings: map[string]interface{}{"a": 2.0, "b": 3.0},
			wantSQL:  "$1",
			wantArgs: []string{"1"},
		},
		{
			name:     "metricFormulaUnsuppliedOptionalVarUsesNeutralOne",
			expr:     querybuilder.MetricExpression{Kind: "formula", Formula: "@a * 2"},
			vars:     []querybuilder.MetricVariable{{Name: "a", Type: "number"}},
			bindings: nil,
			wantSQL:  "$1 * 2",
			wantArgs: []string{"1"},
		},
		{
			name:     "metricFormulaDefaultValueUsedWhenUnbound",
			expr:     querybuilder.MetricExpression{Kind: "formula", Formula: "@a * 2"},
			vars:     []querybuilder.MetricVariable{{Name: "a", Type: "number", DefaultValue: 7.5}},
			bindings: nil,
			wantSQL:  "$1 * 2",
			wantArgs: []string{"7.5"},
		},
		{
			name:     "metricFormulaRequiredVarMissingIsAnError",
			expr:     querybuilder.MetricExpression{Kind: "formula", Formula: "@a * 2"},
			vars:     []querybuilder.MetricVariable{required("a")},
			bindings: nil,
			wantErr:  "required metric variable @a not provided",
		},
		{
			name:    "metricFormulaEmptyIsAnError",
			expr:    querybuilder.MetricExpression{Kind: "formula", Formula: "   "},
			wantErr: "empty formula",
		},
		{
			name:    "metricAggregationSanitizesIdentifier",
			expr:    querybuilder.MetricExpression{Kind: "aggregation", Fn: "sum", TermNodeID: "revenue"},
			wantSQL: "SUM(t0.revenue)",
		},
		{
			// Aggregation genuinely IS allowlisted - unlike formula.
			name:    "metricAggregationRejectsNonAllowlistedFunction",
			expr:    querybuilder.MetricExpression{Kind: "aggregation", Fn: "MEDIAN", TermNodeID: "revenue"},
			wantErr: `unsupported aggregation function "MEDIAN"`,
		},
		{
			// The freeze's second load-bearing case, and a real defect.
			// BaseMetricIDs are sorted alphabetically (metric_compiler.go:86),
			// so the FIRST entry is the NUMERATOR - which means numerator and
			// denominator are decided by how the metric IDs happen to be
			// spelled, not by the order the author declared. Declaring
			// [m_revenue, m_cost] yields cost/revenue. Pinned as-is; fixing
			// it is C1-C3 work, not something to slip past this guard.
			name:    "metricDerivedTwoMetricsTakeNumeratorFromAlphabeticalOrder",
			expr:    querybuilder.MetricExpression{Kind: "derived", BaseMetricIDs: []string{"m_revenue", "m_cost"}},
			lookup:  lookup,
			wantSQL: "(SUM(t0.cost)) / NULLIF((SUM(t0.revenue)), 0)",
		},
		{
			name:    "metricDerivedThreeMetricsCompileToSum",
			expr:    querybuilder.MetricExpression{Kind: "derived", BaseMetricIDs: []string{"m_revenue", "m_cost", "m_units"}},
			lookup:  lookup,
			wantSQL: "(SUM(t0.cost)) + (SUM(t0.revenue)) + (SUM(t0.units))",
		},
		{
			name:    "metricUnknownKindIsAnError",
			expr:    querybuilder.MetricExpression{Kind: "window"},
			wantErr: `unsupported expression kind "window"`,
		},
	}

	mc := querybuilder.NewMetricCompiler(nil)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := querybuilder.MetricDefinition{
				ID:         "m_probe",
				Name:       "probe",
				Expression: tc.expr,
				Variables:  tc.vars,
			}
			res, err := mc.CompileMetric(m, tc.bindings, tc.lookup)

			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected an error containing %q, got SQLExpr %q", tc.wantErr, res.SQLExpr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("expected an error containing %q, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.SQLExpr != tc.wantSQL {
				t.Errorf("SQL expression changed under the C0 freeze.\n  want: %q\n  got:  %q\n"+
					"The metric compiler's expression surface is frozen until C3 (%s). If this change is "+
					"intended, lift the freeze deliberately and record it in ADR-024 - do not just update this case.",
					tc.wantSQL, res.SQLExpr, metricCompilerFreeze.Until)
			}
			gotArgs := make([]string, 0, len(res.Args))
			for _, a := range res.Args {
				gotArgs = append(gotArgs, fmt.Sprintf("%v", a))
			}
			if strings.Join(gotArgs, ",") != strings.Join(tc.wantArgs, ",") {
				t.Errorf("bound arguments changed under the C0 freeze.\n  want: %v\n  got:  %v",
					tc.wantArgs, gotArgs)
			}
		})
	}
}

// TestMetricCompilerDeclaredSurfaceIsFrozen pins the file's func and import
// sets. The golden cases pin behaviour; this pins shape. Behavioural cases can
// miss a helper that is not yet reachable, and a second evaluator is exactly
// the thing C1-C3 are about - it would be added as a new func, in a new file,
// long before anything calls it in a way the golden set would notice.
func TestMetricCompilerDeclaredSurfaceIsFrozen(t *testing.T) {
	root := backendRoot(t)
	path := filepath.Join(root, metricCompilerPath)

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly|parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse %s: %v", metricCompilerPath, err)
	}

	var gotImports []string
	for _, imp := range f.Imports {
		p, _ := strconv.Unquote(imp.Path.Value)
		gotImports = append(gotImports, p)
	}
	sort.Strings(gotImports)
	if strings.Join(gotImports, ",") != strings.Join(metricCompilerImportPins, ",") {
		t.Errorf("metric compiler import surface changed under the C0 freeze.\n  want: %v\n  got:  %v\n"+
			"New imports are how a second expression system gets wired in. C2 adds the vm resolver, and that is "+
			"expected to break this pin - lift the freeze deliberately when it does.",
			metricCompilerImportPins, gotImports)
	}

	declared, err := topLevelFuncNames(path)
	if err != nil {
		t.Fatalf("scan %s: %v", metricCompilerPath, err)
	}
	if strings.Join(declared, ",") != strings.Join(metricCompilerFuncPins, ",") {
		t.Errorf("metric compiler func surface changed under the C0 freeze.\n  want: %v\n  got:  %v\n"+
			"A new func here is a new expression capability. Add it to internal/rules/vm instead, or lift the "+
			"freeze and record it in ADR-024.",
			metricCompilerFuncPins, declared)
	}
}

// topLevelFuncNames returns the sorted names of every func declared in a Go
// file, receiver-qualified where one is present.
func topLevelFuncNames(path string) ([]string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, d := range f.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok {
			continue
		}
		name := fn.Name.Name
		if fn.Recv != nil && len(fn.Recv.List) > 0 {
			name = "(" + exprString(fset, fn.Recv.List[0].Type) + ")." + name
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

// exprString renders an AST expression back to source.
func exprString(fset *token.FileSet, e ast.Expr) string {
	var sb strings.Builder
	if err := printer.Fprint(&sb, fset, e); err != nil {
		return "<unprintable>"
	}
	return sb.String()
}
