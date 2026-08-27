package compliance

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestMandateSMTVerifier_SatisfiableMandate(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	verifier := NewMandateSMTVerifier()

	rules := []RuleConstraintClause{
		{DimensionKey: "asset_class.fixed_income", Operator: ">=", ValuePct: 40.0},
		{DimensionKey: "asset_class.equities", Operator: "<=", ValuePct: 50.0},
		{DimensionKey: "asset_class.cash", Operator: ">=", ValuePct: 5.0},
	}

	res, err := verifier.VerifyMandateConsistency(ctx, tenantID, rules)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !res.IsSatisfiable {
		t.Errorf("expected mandate to be satisfiable, got: %s", res.DiagnosticMessage)
	}
}

func TestMandateSMTVerifier_UnsatisfiableOverAllocation(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	verifier := NewMandateSMTVerifier()

	rules := []RuleConstraintClause{
		{DimensionKey: "asset_class.fixed_income_aaa", Operator: ">=", ValuePct: 85.0},
		{DimensionKey: "asset_class.cash_equivalents", Operator: ">=", ValuePct: 20.0},
	}

	res, err := verifier.VerifyMandateConsistency(ctx, tenantID, rules)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.IsSatisfiable {
		t.Errorf("expected mandate to be unsatisfiable (85%% + 20%% > 100%%)")
	}
	if !res.ConflictDetected {
		t.Errorf("expected conflictDetected = true")
	}
}

func TestMandateSMTVerifier_ContradictoryMinMax(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	verifier := NewMandateSMTVerifier()

	rules := []RuleConstraintClause{
		{DimensionKey: "asset_class.equities", Operator: ">=", ValuePct: 60.0},
		{DimensionKey: "asset_class.equities", Operator: "<=", ValuePct: 40.0},
	}

	res, err := verifier.VerifyMandateConsistency(ctx, tenantID, rules)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.IsSatisfiable {
		t.Errorf("expected contradiction on same dimension")
	}
}
