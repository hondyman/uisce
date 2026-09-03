package analytics

import (
	"testing"

	"github.com/hondyman/uisce/backend/models"
)

func TestResolveTier_PushdownFormula(t *testing.T) {
	calc := &models.Calculation{Formula: "gross_return - management_fee"}
	if err := resolveTier(calc); err != nil {
		t.Fatalf("resolveTier failed: %v", err)
	}
	if calc.Tier != "pushdown" {
		t.Errorf("expected tier=pushdown, got %q", calc.Tier)
	}
	if calc.ExecutionPreference != "auto" {
		t.Errorf("expected execution_preference to default to auto, got %q", calc.ExecutionPreference)
	}
}

func TestResolveTier_HostRuntimeFormula(t *testing.T) {
	calc := &models.Calculation{Formula: "xirr(${cashflow_amount}, ${cashflow_date})"}
	if err := resolveTier(calc); err != nil {
		t.Fatalf("resolveTier failed: %v", err)
	}
	if calc.Tier != "host_runtime" {
		t.Errorf("expected tier=host_runtime, got %q", calc.Tier)
	}
}

func TestResolveTier_ExplicitPushdownRejectsHostRuntimeFormula(t *testing.T) {
	calc := &models.Calculation{Formula: "xirr(${a}, ${b})", ExecutionPreference: "pushdown"}
	if err := resolveTier(calc); err == nil {
		t.Error("expected an error: xirr has no pushdown implementation")
	}
}

func TestResolveTier_InvalidFormula(t *testing.T) {
	calc := &models.Calculation{Formula: "("}
	if err := resolveTier(calc); err == nil {
		t.Error("expected a parse error for invalid formula")
	}
}

func TestResolveTier_InvalidPreference(t *testing.T) {
	calc := &models.Calculation{Formula: "a + b", ExecutionPreference: "bogus"}
	if err := resolveTier(calc); err == nil {
		t.Error("expected an error for an invalid execution_preference value")
	}
}
