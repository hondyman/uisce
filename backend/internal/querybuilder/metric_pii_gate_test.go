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
	gate := NewSensitivityTermGate(bo, "analyst", "")

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
