package rules

import (
	"testing"

	"github.com/hondyman/uisce/backend/internal/boresolver"
)

func TestBuildContextFromBORow_MapsQualifiedColumn(t *testing.T) {
	bo := &boresolver.BODefinition{
		ID:           "customer",
		DrivingTable: "customers",
		Fields: []boresolver.BOField{
			{Name: "kyc_status", PhysicalColumn: "customers.kyc_status"},
			{Name: "risk_score", PhysicalColumn: "customers.risk_score"},
		},
	}
	row := map[string]interface{}{
		"customers.kyc_status": "APPROVED",
		"customers.risk_score": 42,
		"customers.unmapped":   "passthrough",
	}

	ctx := BuildContextFromBORow(bo, row)

	if ctx["kyc_status"] != "APPROVED" {
		t.Errorf("expected kyc_status=APPROVED, got %v", ctx["kyc_status"])
	}
	if ctx["risk_score"] != 42 {
		t.Errorf("expected risk_score=42, got %v", ctx["risk_score"])
	}
	if ctx["customers.unmapped"] != "passthrough" {
		t.Errorf("expected unmapped column to pass through unchanged, got %v", ctx["customers.unmapped"])
	}
}

func TestBuildContextFromBORow_MapsBareColumnFallback(t *testing.T) {
	bo := &boresolver.BODefinition{
		ID:           "customer",
		DrivingTable: "customers",
		Fields: []boresolver.BOField{
			{Name: "kyc_status", PhysicalColumn: "customers.kyc_status"},
		},
	}
	row := map[string]interface{}{
		"kyc_status": "REJECTED",
	}

	ctx := BuildContextFromBORow(bo, row)

	if ctx["kyc_status"] != "REJECTED" {
		t.Errorf("expected kyc_status=REJECTED via bare-column fallback, got %v", ctx["kyc_status"])
	}
}

func TestBuildContextFromBORow_ThenEvaluatesRuleAgainstBOFieldName(t *testing.T) {
	bo := &boresolver.BODefinition{
		ID:           "customer",
		DrivingTable: "customers",
		Fields: []boresolver.BOField{
			{Name: "kyc_status", PhysicalColumn: "customers.kyc_status"},
		},
	}
	row := map[string]interface{}{
		"customers.kyc_status": "APPROVED",
	}
	ctx := BuildContextFromBORow(bo, row)

	ce := NewConditionEvaluator()
	passed, actual, expected, err := ce.EvaluateConditionWithValues(&RuleCondition{
		Field:    "kyc_status",
		Operator: "equals",
		Value:    "APPROVED",
	}, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !passed {
		t.Errorf("expected rule to pass against BO field name, actual=%v expected=%v", actual, expected)
	}
}
