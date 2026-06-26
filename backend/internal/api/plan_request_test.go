package api

import "testing"

func TestPlanRequest_ValidateRejectsMissingRegion(t *testing.T) {
	pr := &PlanRequest{
		TenantID:       "acme",
		Region:         "",
		BusinessObject: "orders",
	}
	if err := pr.Validate(); err == nil {
		t.Fatalf("expected error when region missing")
	} else if err.Error() != "region is required for all semantic operations." {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPlanRequest_ValidateAcceptsRegion(t *testing.T) {
	pr := &PlanRequest{
		TenantID:       "acme",
		Region:         "eu-west",
		BusinessObject: "orders",
	}
	if err := pr.Validate(); err != nil {
		t.Fatalf("expected no error when region present, got: %v", err)
	}
}
