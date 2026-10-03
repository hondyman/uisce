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
// HISTORY: this freeze found two real defects on its first run, and both were
// fixed deliberately rather than pinned:
//
//  1. "@a*@b" was one whitespace token, was looked up as a variable named
//     "a*@b", missed, and fell through to the neutral 1.0 multiplier - the
//     multiplication was silently dropped and the metric compiled to a
//     constant. Now pinned by metricFormulaUnspacedOperatorBindsBothOperands
//     and its neighbours.
//  2. BaseMetricIDs were sorted alphabetically and the first entry became the
//     numerator, so a declared revenue/cost ratio compiled to cost/revenue.
//     Now pinned by metricDerivedTwoMetricsUseDeclaredOrder and
//     metricDerivedTwoMetricsUseExplicitOperands.
//
// Both were wrong answers reaching StarRocks DDL through the cube DDL
// generator. They were fixed BEFORE the golden corpus (8.3) was built, on the
// ruling that a corpus which snapshots inverted semantics would enshrine the
// bug as the reference. The cases below are the new baseline, and the freeze
// still holds them: changing any of them again is a decision, not a side
// effect.

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
//
// C2 ADDS THREE ENTRIES, DELIBERATELY. The freeze's own failure message says
// "C2 adds the vm resolver, and that is expected to break this pin - lift the
// freeze deliberately when it does", and C2 has started. These are admitted
// because the freeze protects against a second *expression* system, and none of
// them adds expression capability:
//
//   - NewSensitivityTermGate / WithTermGate / checkTerm are the PII gate over
//     term resolution (ADR-024). They decide whether a term may be read; they
//     do not parse, compile, evaluate or emit an expression. The emitted SQL is
//     produced by the same pre-existing code path as before.
//   - collectFieldRefs walks a calc term's vm.Expression so the gate can reach
//     the columns that term reads. It is a tree walk, not an evaluator: it
//     returns field paths and interprets nothing. It panics on an unhandled
//     node type rather than skipping it, so a new AST node cannot silently
//     become an unchecked branch.
//
// The pin stays ENFORCED and is simply wider: the next func added here still
// fails this test, which is the point. Lifting a pin by deleting it would have
// turned the guard off; this turns it forward by one deliberate step.
var metricCompilerFuncPins = []string{
	"(*MetricCompiler).CompileMetric",
	"(*MetricCompiler).WithTermGate",
	"(*MetricCompiler).checkTerm",
	"(*MetricCompiler).compileFormula",
	"(*MetricCompiler).compileMetricWithCycleDetection",
	"ComputeQueryAndMetricsAndCubeCacheKey",
	"ComputeQueryAndMetricsCacheKey",
	"NewMetricCompiler",
	"NewSensitivityTermGate",
	"collectFieldRefs",
	"sanitizeIdentifier",
}

// metricCompilerImportPins is the file's import set.
//
// C2 ADDS ONE ENTRY, DELIBERATELY - this is the break this pin was written to
// expect. Its own comment said "C2 adds the vm resolver, so this pin is expected
// to break when C2 starts - that is the freeze doing its job, not a bug in it",
// and the func-surface pin beside it said to lift the freeze deliberately when
// it does.
//
// internal/rules/vm is added for the calc-term walk: the gate has to read a calc
// term's vm.Expression to find the columns that term can reach, which is the
// whole point of the check (see ADR-024's C2 entry for why inspecting only the
// term a metric names was a hole). vm is the intended survivor of C1-C3 and
// already a dependency of this package via metric_definition.go, so the pin now
// records the dependency the C1-C3 plan intends rather than forbidding it.
//
// The pin stays ENFORCED: one more import still fails this test.
var metricCompilerImportPins = []string{
	"crypto/sha256",
	"encoding/hex",
	"fmt",
	"github.com/hondyman/uisce/backend/internal/boresolver",
	"github.com/hondyman/uisce/backend/internal/rules/vm",
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
			// The tokenizer regression, fixed. "@a*@b" is one whitespace token
			// beginning with "@"; it used to be looked up as a variable named
			// "a*@b", miss, and fall to the neutral 1.0 multiplier - dropping
			// the multiplication and compiling to a constant with no error.
			// Both operands now bind, and the "*" is preserved verbatim.
			name:     "metricFormulaUnspacedOperatorBindsBothOperands",
			expr:     querybuilder.MetricExpression{Kind: "formula", Formula: "@a*@b"},
			vars:     []querybuilder.MetricVariable{required("a"), required("b")},
			bindings: map[string]interface{}{"a": 2.0, "b": 3.0},
			wantSQL:  "$1*$2",
			wantArgs: []string{"2", "3"},
		},
		{
			// Neighbour: the spaced form must be unchanged by the fix.
			name:     "metricFormulaSpacedOperatorBindsBothOperands",
			expr:     querybuilder.MetricExpression{Kind: "formula", Formula: "@a * @b"},
			vars:     []querybuilder.MetricVariable{required("a"), required("b")},
			bindings: map[string]interface{}{"a": 2.0, "b": 3.0},
			wantSQL:  "$1 * $2",
			wantArgs: []string{"2", "3"},
		},
		{
			// Neighbour: variable adjacent to a literal.
			name:     "metricFormulaVariableAdjacentToLiteralBinds",
			expr:     querybuilder.MetricExpression{Kind: "formula", Formula: "@a*2"},
			vars:     []querybuilder.MetricVariable{required("a")},
			bindings: map[string]interface{}{"a": 5.0},
			wantSQL:  "$1*2",
			wantArgs: []string{"5"},
		},
		{
			// The reason substitution must happen IN PLACE, with no whitespace
			// inserted: a Postgres cast and a JSON operator both break if a
			// space is added at the substitution point.
			name:     "metricFormulaCastAndJsonOperatorSurviveSubstitution",
			expr:     querybuilder.MetricExpression{Kind: "formula", Formula: "@a::numeric + @a->>'k'"},
			vars:     []querybuilder.MetricVariable{required("a")},
			bindings: map[string]interface{}{"a": 1.0},
			wantSQL:  "$1::numeric + $2->>'k'",
			wantArgs: []string{"1", "1"},
		},
		{
			name:     "metricFormulaBareAtIsNotAVariable",
			expr:     querybuilder.MetricExpression{Kind: "formula", Formula: "email @ @a"},
			vars:     []querybuilder.MetricVariable{required("a")},
			bindings: map[string]interface{}{"a": 1.0},
			wantSQL:  "email @ $1",
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
			// ADR-026: the ordered form is refused, not honoured. This case
			// previously asserted that [m_revenue, m_cost] meant revenue over
			// cost; the ruling went further and made the direction data the
			// author must state, so a bare ordering is an error.
			name:    "metricDerivedOrderedRatioIsRefused",
			expr:    querybuilder.MetricExpression{Kind: "derived", BaseMetricIDs: []string{"m_revenue", "m_cost"}},
			lookup:  lookup,
			wantErr: "derived ratio requires numeratorId and denominatorId",
		},
		{
			// The inverse declaration is equally refused: the rule is about
			// ambiguity, not about a preferred direction.
			name:    "metricDerivedOrderedRatioInvertedIsRefusedToo",
			expr:    querybuilder.MetricExpression{Kind: "derived", BaseMetricIDs: []string{"m_cost", "m_revenue"}},
			lookup:  lookup,
			wantErr: "derived ratio requires numeratorId and denominatorId",
		},
		{
			// The surviving contract: direction is named, and inverting the
			// named operands inverts the measure.
			name: "metricDerivedRatioUsesNamedOperands",
			expr: querybuilder.MetricExpression{
				Kind:          "derived",
				BaseMetricIDs: []string{"m_cost", "m_revenue"},
				NumeratorID:   "m_revenue",
				DenominatorID: "m_cost",
			},
			lookup:  lookup,
			wantSQL: "(SUM(t0.revenue)) / NULLIF((SUM(t0.cost)), 0)",
		},
		{
			name: "metricDerivedRatioInvertingNamedOperandsInvertsTheMeasure",
			expr: querybuilder.MetricExpression{
				Kind:          "derived",
				BaseMetricIDs: []string{"m_revenue", "m_cost"},
				NumeratorID:   "m_cost",
				DenominatorID: "m_revenue",
			},
			lookup:  lookup,
			wantSQL: "(SUM(t0.cost)) / NULLIF((SUM(t0.revenue)), 0)",
		},
		{
			name: "metricDerivedExplicitOperandsMustBothBeSet",
			expr: querybuilder.MetricExpression{
				Kind:          "derived",
				BaseMetricIDs: []string{"m_cost", "m_revenue"},
				NumeratorID:   "m_revenue",
			},
			wantErr: "derived ratio requires numeratorId and denominatorId",
		},
		{
			name: "metricDerivedExplicitOperandsMustBeExactlyTheBaseIds",
			expr: querybuilder.MetricExpression{
				Kind:          "derived",
				BaseMetricIDs: []string{"m_revenue", "m_cost", "m_units"},
				NumeratorID:   "m_revenue",
				DenominatorID: "m_cost",
			},
			wantErr: "numeratorId/denominatorId describe a 2-metric ratio, but baseMetricIds has 3 entries",
		},
		{
			name: "metricDerivedRatioOperandsMustNotBeTheSameMetric",
			expr: querybuilder.MetricExpression{
				Kind:          "derived",
				BaseMetricIDs: []string{"m_revenue", "m_cost"},
				NumeratorID:   "m_revenue",
				DenominatorID: "m_revenue",
			},
			lookup:  lookup,
			wantErr: "numeratorId and denominatorId are the same metric",
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
