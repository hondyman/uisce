package mdm

import (
	"context"
	"testing"

	"github.com/hondyman/uisce/backend/internal/boresolver"
	"github.com/hondyman/uisce/backend/internal/rules"
)

func kycRule() *rules.RuleNode {
	return &rules.RuleNode{
		Type: rules.NodeTypeGroup,
		Group: &rules.RuleGroup{
			ID:       "grp-kyc-status",
			Operator: "AND",
			Conditions: []rules.RuleNode{
				{
					Type: rules.NodeTypeCondition,
					Condition: &rules.RuleCondition{
						ID:        "kyc-status-approved",
						Field:     "kyc_status",
						FieldPath: "kyc_status",
						Operator:  "==",
						Value:     "APPROVED",
					},
				},
			},
		},
	}
}

func customerBO() *boresolver.BODefinition {
	return &boresolver.BODefinition{
		ID:           "customer",
		DrivingTable: "customers",
		Fields: []boresolver.BOField{
			{Name: "kyc_status", PhysicalColumn: "customers.kyc_status"},
		},
	}
}

func TestValidateGoldenRecord_NilRulePasses(t *testing.T) {
	eng := rules.NewRuleEngine(nil)
	passed, err := ValidateGoldenRecord(context.Background(), eng, customerBO(), map[string]interface{}{}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !passed {
		t.Error("expected nil rule to always pass")
	}
}

func TestValidateGoldenRecord_PassesOnMatchingBOField(t *testing.T) {
	eng := rules.NewRuleEngine(nil)
	resolved := map[string]interface{}{"customers.kyc_status": "APPROVED"}

	passed, err := ValidateGoldenRecord(context.Background(), eng, customerBO(), resolved, kycRule())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !passed {
		t.Error("expected golden record with APPROVED kyc_status to pass the rule")
	}
}

func TestValidateGoldenRecord_FailsOnMismatch(t *testing.T) {
	eng := rules.NewRuleEngine(nil)
	resolved := map[string]interface{}{"customers.kyc_status": "PENDING"}

	passed, err := ValidateGoldenRecord(context.Background(), eng, customerBO(), resolved, kycRule())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if passed {
		t.Error("expected golden record with PENDING kyc_status to fail the rule")
	}
}
