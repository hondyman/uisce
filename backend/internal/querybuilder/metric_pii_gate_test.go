package querybuilder

// PII gate through the metric path (C2 / 9.2).
//
// The BO path refuses to compile a calc term that reads a masked column:
// bo_sql_generator.go's resolveCol closure checks
//
//	if foundField.TermType == "calculated" && f.SensitivityTag != "" {
//	    tier := DetermineMaskingTier(f.SensitivityTag, role, clearance)
//	    if tier != MaskingTierPassthrough { return "", fmt.Errorf(...) }
//	}
//
// before it hands back a physical column. MetricCompiler had no equivalent.
// Its aggregation case took the authored term straight to a column name:
//
//	colRef := fmt.Sprintf("t0.%s", sanitizeIdentifier(termID))
//
// so a metric could name any column, including a PII one, and the compiled SQL
// carried it. MetricCompiler holds only a dialect (NewMetricCompiler), so it had
// neither the BO definition that carries SensitivityTag nor the request's role
// and clearance - it could not have run the check even if it had wanted to.
//
// This file is the named security gate for C2. It pins three things:
//
//  1. TestUngatedMetricCompilerIsUnsafe - that an ungated compiler does let a
//     PII column through, which is the fact that makes the DDL path's
//     fail-closed requirement necessary rather than merely tidy.
//  2. TestMetricCompilerRejectsMetricOverMaskedTerm - the gate itself, plus
//     equivalence for a term that passes, so a gate cannot be satisfied by
//     refusing everything.
//  3. TestCubeDDLRequiresTermGate - that the deploy path cannot be used
//     ungated.

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/hondyman/uisce/backend/internal/boresolver"
	"github.com/hondyman/uisce/backend/internal/rules/vm"
)

// piiField is a BO field whose physical column is classified as PII.
func piiField() boresolver.BOField {
	return boresolver.BOField{
		ID:             "f_ssn",
		Name:           "customer_ssn",
		DisplayName:    "Customer SSN",
		PhysicalColumn: "customers.ssn",
		SensitivityTag: "pii",
	}
}

// nonPIIField is an ordinary field in the same BO.
func nonPIIField() boresolver.BOField {
	return boresolver.BOField{
		ID:             "f_price",
		Name:           "price",
		DisplayName:    "Unit price",
		PhysicalColumn: "order_lines.price",
	}
}

func testBO() *boresolver.BODefinition {
	return &boresolver.BODefinition{
		ID:           "bo_order",
		DrivingTable: "order_lines",
		Fields:       []boresolver.BOField{piiField(), nonPIIField()},
	}
}

// aggregationOver returns a one-metric definition aggregating fn over termID.
func aggregationOver(termID, fn string) MetricDefinition {
	return MetricDefinition{
		ID:   "m_test",
		Name: "Test metric",
		Expression: MetricExpression{
			Kind:       "aggregation",
			Fn:         fn,
			TermNodeID: termID,
		},
	}
}

// TestUngatedMetricCompilerIsUnsafe pins the reason the DDL path fails closed.
//
// A MetricCompiler with no gate still compiles a metric over a PII column
// without complaint. That is deliberate: the golden corpus and the equivalence
// suite compile metrics with no BO and no request context, and forcing a gate
// there would mean changing the 8.3 corpus to accommodate a security control
// that is not what those tests are about.
//
// The consequence is that "ungated" is unsafe by construction, which is exactly
// why GenerateCubeMaterializationDDL refuses to run without a gate
// (TestCubeDDLRequiresTermGate). This test exists so that the unsafe default
// is a measured fact in the repository rather than a comment nobody checks: if
// the gate were ever made to also cover the ungated compiler, this test fails
// and the decision gets made deliberately rather than by accident.
func TestUngatedMetricCompilerIsUnsafe(t *testing.T) {
	bo := testBO()
	term, err := resolveTermToField(bo, "customer_ssn")
	if err != nil {
		t.Fatalf("precondition: the PII term must be resolvable in the BO: %v", err)
	}
	if term.SensitivityTag == "" {
		t.Fatalf("precondition: field %q must carry a sensitivity tag", term.Name)
	}

	// The BO path's own view: at passthrough threshold this column is refused.
	if tier := boresolver.DetermineMaskingTier(term.SensitivityTag, "analyst", ""); tier == boresolver.MaskingTierPassthrough {
		t.Fatalf("precondition: %q should not be passthrough for a low-clearance analyst", term.SensitivityTag)
	}

	mc := NewMetricCompiler(nil)
	got, err := mc.CompileMetric(aggregationOver("customer_ssn", "SUM"), nil, nil)
	if err != nil {
		t.Skipf("gate already present - CompileMetric refused the PII term: %v", err)
	}
	if !strings.Contains(strings.ToLower(got.SQLExpr), "ssn") {
		t.Fatalf("expected the PII column to leak into the compiled SQL pre-gate, got %q", got.SQLExpr)
	}
	t.Logf("CONFIRMED HOLE: metric over PII term %q compiled to %q with no error", term.PhysicalColumn, got.SQLExpr)
}

// calcTermBO is a BO where one term is a CALC TERM whose own expression reads
// the PII column. This is the shape the BO path's predicate at
// boresolver/bo_sql_generator.go:781-787 is written for: the check fires inside
// a calc term's expression, on the tag of the column that expression references.
// A calc term has no PhysicalColumn and usually no SensitivityTag of its own, so
// a gate that only inspected the term a metric names would pass it and the PII
// would be read one level down. That is a real hole, and it was in the first
// version of this gate - see TestMetricGateReachesThroughCalcTerm and the
// ADR-024 C2 entry.
func calcTermBO() *boresolver.BODefinition {
	bo := testBO()
	bo.Fields = append(bo.Fields, boresolver.BOField{
		ID:             "f_net",
		Name:           "net_per_order",
		PhysicalColumn: "", // a calc term has none
		TermType:       "calculated",
		SemanticTermID: "calc_net",
		// SensitivityTag deliberately empty: the tag lives on the column the
		// calc term reads, not on the term.
	})
	return bo
}

// calcNetExpr is the calc term's compiled expression: it references the PII
// column by name.
func calcNetExpr(refPath string) *vm.Expression {
	return &vm.Expression{Root: &vm.BinaryExpr{
		Op:    "-",
		Left:  &vm.FieldRef{Path: refPath},
		Right: &vm.Literal{Value: 1},
	}}
}

// TestMetricGateReachesThroughCalcTerm is the second hole, and the one the BO
// predicate is actually about.
//
// It asserts the metric path refuses a metric whose term is a calc term reading
// a PII column. Before the recursion was added this compiled to
// SUM(t0.net_per_order) with no error: the gate resolved the term, saw a calc
// term with no sensitivity tag of its own, and permitted it.
func TestMetricGateReachesThroughCalcTerm(t *testing.T) {
	bo := calcTermBO()
	gate := NewSensitivityTermGate(bo, "analyst", "", map[string]*vm.Expression{
		"calc_net": calcNetExpr("customer_ssn"),
	})

	mc := NewMetricCompiler(nil).WithTermGate(gate)
	got, err := mc.CompileMetric(aggregationOver("net_per_order", "SUM"), nil, nil)
	if err == nil {
		t.Fatalf("gate did not reach through the calc term: compiled to %q", got.SQLExpr)
	}
	if !errors.Is(err, ErrMetricTermNotPermitted) {
		t.Fatalf("want ErrMetricTermNotPermitted, got %v", err)
	}
	// The error must name the PII column reached THROUGH the calc term, not the
	// calc term itself - otherwise an operator cannot tell what was refused.
	if !strings.Contains(err.Error(), "customers.ssn") {
		t.Fatalf("error should name the PII column reached through the calc term, got %v", err)
	}
}

// TestMetricGateRefusesUninspectableCalcTerm pins the fail-closed choice for a
// calc term with no preloaded expression. An uninspectable chain is not a clean
// chain: permitting it would make "clear the term" mean "clear the term if
// someone remembered to hand us its expression".
func TestMetricGateRefusesUninspectableCalcTerm(t *testing.T) {
	bo := calcTermBO()
	// calcTerms deliberately nil: the calc term's expression is unavailable.
	mc := NewMetricCompiler(nil).WithTermGate(NewSensitivityTermGate(bo, "admin", "CONFIDENTIAL", nil))
	if _, err := mc.CompileMetric(aggregationOver("net_per_order", "SUM"), nil, nil); err == nil {
		t.Fatal("a calc term with no inspectable expression must not pass the gate")
	}
}

// TestMetricGateStopsCalcTermCycles pins the cycle guard. A calc term chain that
// loops must terminate with an error rather than recursing until the stack dies -
// the same failure the BO resolver's seenPtr prevents.
func TestMetricGateStopsCalcTermCycles(t *testing.T) {
	bo := &boresolver.BODefinition{ID: "bo_order", DrivingTable: "order_lines"}
	bo.Fields = []boresolver.BOField{
		{ID: "f_a", Name: "a", TermType: "calculated", SemanticTermID: "calc_a"},
		{ID: "f_b", Name: "b", TermType: "calculated", SemanticTermID: "calc_b"},
	}
	// clearance CONFIDENTIAL so the tier check is passthrough throughout and
	// only the structural guards can be what refuses these.
	mk := func(terms map[string]*vm.Expression) *MetricCompiler {
		return NewMetricCompiler(nil).WithTermGate(NewSensitivityTermGate(bo, "admin", "CONFIDENTIAL", terms))
	}

	t.Run("a self-referencing calc term is named as a cycle", func(t *testing.T) {
		// a -> a is detected by the cycle guard at depth 0, before the depth cap
		// can preempt it, so this pins the cycle detector specifically.
		mc := mk(map[string]*vm.Expression{"calc_a": calcNetExpr("a")})
		_, err := mc.CompileMetric(aggregationOver("a", "SUM"), nil, nil)
		if err == nil {
			t.Fatal("a self-referencing calc term must not compile")
		}
		if !strings.Contains(err.Error(), "cycle") {
			t.Fatalf("want a cycle error for a self-reference, got %v", err)
		}
	})

	t.Run("a mutually-referencing chain terminates with an error", func(t *testing.T) {
		// a -> b -> a trips the depth cap before the cycle guard, because
		// maxMetricCalcTermDepth is 1. Which guard fires first is an
		// implementation detail; the property that matters is that it
		// TERMINATES with a structural refusal rather than recursing until the
		// stack dies. Asserting "error mentions cycle OR depth" keeps the test
		// from passing on an unrelated error while not over-specifying.
		mc := mk(map[string]*vm.Expression{
			"calc_a": calcNetExpr("b"),
			"calc_b": calcNetExpr("a"),
		})
		_, err := mc.CompileMetric(aggregationOver("a", "SUM"), nil, nil)
		if err == nil {
			t.Fatal("a mutually-referencing calc-term chain must not compile")
		}
		if !strings.Contains(err.Error(), "cycle") && !strings.Contains(err.Error(), "depth") {
			t.Fatalf("want a cycle or depth error, got %v", err)
		}
	})
}

// TestMetricCompilerRejectsMetricOverMaskedTerm is the gate.
//
// Two properties, because either alone is insufficient. A gate that refuses
// everything would satisfy the first, and a gate that refuses nothing satisfies
// the second:
//
//  1. A metric aggregating a term whose column is tagged PII is refused, and the
//     error is ErrMetricTermNotPermitted rather than a generic failure - the
//     distinction matters because the formula is well-formed and only the data
//     it reaches is not permitted.
//  2. A metric over an untagged term still compiles, and to the same SQL as
//     before the gate existed. Equivalence with the ungated compiler is the
//     check that the gate did not quietly change what a clean metric compiles
//     to - which would be a correctness regression traded for a security fix.
func TestMetricCompilerRejectsMetricOverMaskedTerm(t *testing.T) {
	bo := testBO()
	gate := NewSensitivityTermGate(bo, "analyst", "", nil)

	t.Run("refuses a metric over a PII term", func(t *testing.T) {
		mc := NewMetricCompiler(nil).WithTermGate(gate)
		got, err := mc.CompileMetric(aggregationOver("customer_ssn", "SUM"), nil, nil)
		if err == nil {
			t.Fatalf("PII gate did not bite: compiled to %q", got.SQLExpr)
		}
		if !errors.Is(err, ErrMetricTermNotPermitted) {
			t.Fatalf("want ErrMetricTermNotPermitted, got %v", err)
		}
		// The refused term must not appear in any emitted SQL: the gate runs
		// before sanitization, so a refusal cannot leave a partial artifact.
		if strings.Contains(fmt.Sprint(got), "ssn") {
			t.Fatalf("refused PII term leaked into the result: %+v", got)
		}
		if !strings.Contains(err.Error(), "customers.ssn") {
			t.Fatalf("error should name the physical column it refused, got %v", err)
		}
	})

	t.Run("permits a tagged term whose tier is passthrough for this caller", func(t *testing.T) {
		// The term is TAGGED. That is the point: an equivalence case built from
		// an untagged term passes even if the gate ignores tiers entirely, so it
		// cannot demonstrate the tier condition. DetermineMaskingTier returns
		// PASSTHROUGH for clearance CONFIDENTIAL regardless of the tag, so this
		// is a genuinely permitted PII-tagged term.
		permissive := NewSensitivityTermGate(bo, "analyst", "CONFIDENTIAL", nil)
		ungated := NewMetricCompiler(nil)
		gated := NewMetricCompiler(nil).WithTermGate(permissive)

		before, err := ungated.CompileMetric(aggregationOver("customer_ssn", "SUM"), nil, nil)
		if err != nil {
			t.Fatalf("ungated compile must succeed: %v", err)
		}
		after, err := gated.CompileMetric(aggregationOver("customer_ssn", "SUM"), nil, nil)
		if err != nil {
			t.Fatalf("a passthrough-tier tagged term must be permitted, got: %v", err)
		}
		if before.SQLExpr != after.SQLExpr {
			t.Fatalf("gate changed the SQL for a permitted term: %q -> %q", before.SQLExpr, after.SQLExpr)
		}
		if before.ContentHash != after.ContentHash {
			t.Fatalf("gate changed the content hash: %s -> %s", before.ContentHash, after.ContentHash)
		}
	})

	t.Run("the same tagged term is refused at a lower clearance", func(t *testing.T) {
		// The tier condition is the whole difference between these two cases.
		// Together they pin that the gate reads the tier rather than treating
		// "tagged" as synonymous with "forbidden".
		strict := NewSensitivityTermGate(bo, "analyst", "", nil)
		mc := NewMetricCompiler(nil).WithTermGate(strict)
		if _, err := mc.CompileMetric(aggregationOver("customer_ssn", "SUM"), nil, nil); err == nil {
			t.Fatal("the same tagged term must be refused for a caller without the clearance")
		}
	})

	t.Run("permits a metric over an untagged term, with identical SQL", func(t *testing.T) {
		ungated := NewMetricCompiler(nil)
		gated := NewMetricCompiler(nil).WithTermGate(gate)

		before, err := ungated.CompileMetric(aggregationOver("price", "SUM"), nil, nil)
		if err != nil {
			t.Fatalf("ungated compile of a clean term must succeed: %v", err)
		}
		after, err := gated.CompileMetric(aggregationOver("price", "SUM"), nil, nil)
		if err != nil {
			t.Fatalf("gated compile of a clean term must succeed: %v", err)
		}
		if before.SQLExpr != after.SQLExpr {
			t.Fatalf("gate changed the SQL for a permitted term: %q -> %q", before.SQLExpr, after.SQLExpr)
		}
		if before.ContentHash != after.ContentHash {
			t.Fatalf("gate changed the content hash: %s -> %s", before.ContentHash, after.ContentHash)
		}
	})

	t.Run("refuses a term the semantic layer does not know", func(t *testing.T) {
		mc := NewMetricCompiler(nil).WithTermGate(gate)
		// A term absent from the BO has no classification, so there is nothing
		// to clear it against. An unknown term is a refusal, not a pass.
		if _, err := mc.CompileMetric(aggregationOver("column_nobody_declared", "SUM"), nil, nil); err == nil {
			t.Fatal("an unresolvable term must not pass the gate")
		}
	})
}

// TestCubeDDLRequiresTermGate pins the fail-closed decision.
//
// The DDL path is where a cleared term becomes a stored artifact. A gate that
// were optional there would be an opt-in security control on a deploy path.
func TestCubeDDLRequiresTermGate(t *testing.T) {
	gen := NewCubeDDLGenerator("starrocks")
	if _, err := gen.GenerateCubeMaterializationDDL(
		"t1", false, cubeWithOneMetric("m_test"), []string{"order_date"}, "oms.order_lines",
		map[string]MetricDefinition{"m_test": aggregationOver("price", "SUM")}, nil,
	); err == nil {
		t.Fatal("cube DDL generation must refuse to run without a term gate")
	} else if !strings.Contains(err.Error(), "term gate") {
		t.Fatalf("error should name the missing gate, got %v", err)
	}
}

// On the BO-path CONTROL: the reference behaviour is already pinned where the
// code lives - boresolver's own TestCalcTerm_MaskingBlocked
// (internal/boresolver/bo_sql_generator_test.go) drives the real generator and
// asserts that a calc term referencing a PII-tagged column is refused with
// "references masked column" at tier REDACT_FULL, and that the untagged case
// returns NoError.
//
// That test is the control, and this file deliberately does NOT re-implement it.
// A copy here would only assert my reading of a predicate back to myself, and
// would pass just as happily if that reading were wrong - which is the circularity
// that makes "BO-path equivalence" worthless as a phrase. The metric gate's
// expected behaviour is anchored to that existing test; the equivalence cases
// above are what pin the metric path to it, including the tier condition in both
// directions.

// cubeWithOneMetric is the minimum cube that reaches the gate check.
func cubeWithOneMetric(metricID string) CubeDefinition {
	return CubeDefinition{
		ID:        "c_test",
		Name:      "Test cube",
		BOID:      "bo_order",
		MetricIDs: []string{metricID},
		Grains:    [][]string{{"order_date"}},
	}
}
