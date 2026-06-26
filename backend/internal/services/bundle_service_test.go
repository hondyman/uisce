package services

import (
	"errors"
	"testing"

	"github.com/hondyman/semlayer/backend/internal/models"
)

func TestBundleServiceRoleAssignments(t *testing.T) {
	policySvc := NewPolicyService()
	bundleSvc, roleMgr := NewBundleService(policySvc)

	steward := models.User{
		ID:    "stew-1",
		Role:  "Steward",
		Roles: []string{"Steward"},
	}

	bundle, err := bundleSvc.CreateBundle(steward, "Front Office Performance", "Test bundle")
	if err != nil {
		t.Fatalf("CreateBundle returned error: %v", err)
	}

	if len(bundle.AllowedRoles) == 0 {
		t.Fatalf("expected bundle to have allowed roles after creation")
	}

	riskUser := models.User{ID: "risk-1", Role: "RiskOfficer", Roles: []string{"RiskOfficer"}}

	bundles, err := bundleSvc.ListBundles(riskUser)
	if err != nil {
		t.Fatalf("ListBundles returned error: %v", err)
	}
	if len(bundles) != 0 {
		t.Fatalf("expected no bundles for risk user before assignment, got %d", len(bundles))
	}

	if err := roleMgr.AssignRoleToBundle("RiskOfficer", bundle.ID); err != nil {
		t.Fatalf("AssignRoleToBundle returned error: %v", err)
	}

	bundles, err = bundleSvc.ListBundles(riskUser)
	if err != nil {
		t.Fatalf("ListBundles returned error after assignment: %v", err)
	}
	if len(bundles) != 1 {
		t.Fatalf("expected one bundle after assignment, got %d", len(bundles))
	}

	if err := roleMgr.UnassignRoleFromBundle("RiskOfficer", bundle.ID); err != nil {
		t.Fatalf("UnassignRoleFromBundle returned error: %v", err)
	}

	bundles, err = bundleSvc.ListBundles(riskUser)
	if err != nil {
		t.Fatalf("ListBundles returned error after unassignment: %v", err)
	}
	if len(bundles) != 0 {
		t.Fatalf("expected no bundles after unassignment, got %d", len(bundles))
	}
}

func TestBundleServiceLifecyclePolicies(t *testing.T) {
	policySvc := NewPolicyService()
	bundleSvc, _ := NewBundleService(policySvc)

	steward := models.User{ID: "stew-2", Role: "Steward", Roles: []string{"Steward"}}
	admin := models.User{ID: "admin-1", Role: "Admin", Roles: []string{"Admin"}}

	bundle, err := bundleSvc.CreateBundle(steward, "Risk Oversight", "Lifecycle test")
	if err != nil {
		t.Fatalf("CreateBundle returned error: %v", err)
	}

	bundle, err = bundleSvc.CertifyBundle(steward, bundle.ID)
	if err != nil {
		t.Fatalf("CertifyBundle returned error: %v", err)
	}
	if bundle.Status != models.StatusCertified {
		t.Fatalf("expected bundle status Certified, got %s", bundle.Status)
	}

	bundle, err = bundleSvc.PublishBundle(admin, bundle.ID)
	if err != nil {
		t.Fatalf("PublishBundle returned error: %v", err)
	}
	if bundle.Status != models.StatusPublished {
		t.Fatalf("expected bundle status Published, got %s", bundle.Status)
	}

	// Attempt to publish with non-admin should fail
	otherBundle, err := bundleSvc.CreateBundle(steward, "Client Reporting", "Another bundle")
	if err != nil {
		t.Fatalf("CreateBundle returned error: %v", err)
	}
	if _, err := bundleSvc.CertifyBundle(steward, otherBundle.ID); err != nil {
		t.Fatalf("CertifyBundle returned error: %v", err)
	}
	if _, err := bundleSvc.PublishBundle(steward, otherBundle.ID); err == nil {
		t.Fatalf("expected publish with steward to fail, but succeeded")
	}
}

func TestUpdateBundlePoliciesValidationError(t *testing.T) {
	policySvc := NewPolicyService()
	bundleSvc, _ := NewBundleService(policySvc)

	steward := models.User{ID: "stew-3", Role: "Steward", Roles: []string{"Steward"}}

	bundle, err := bundleSvc.CreateBundle(steward, "Validation Target", "")
	if err != nil {
		t.Fatalf("CreateBundle returned error: %v", err)
	}

	_, err = bundleSvc.UpdateBundlePolicies(steward, bundle.ID, []models.BundleRowPolicy{{
		Name:     "",
		Member:   "",
		Operator: "equals",
	}}, nil)
	if err == nil {
		t.Fatalf("expected validation error, got nil")
	}
	var valErr *ValidationError
	if !errors.As(err, &valErr) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if len(valErr.Errors) == 0 {
		t.Fatalf("expected field errors, got none")
	}
}

func TestUpdateBundlePoliciesSanitizesInput(t *testing.T) {
	policySvc := NewPolicyService()
	bundleSvc, _ := NewBundleService(policySvc)

	steward := models.User{ID: "stew-4", Role: "Steward", Roles: []string{"Steward"}}

	bundle, err := bundleSvc.CreateBundle(steward, "Sanitization Target", "")
	if err != nil {
		t.Fatalf("CreateBundle returned error: %v", err)
	}

	rowPolicies := []models.BundleRowPolicy{{
		ID:          " rp-1 ",
		Name:        " Tenant Filter ",
		Description: "  desc  ",
		Member:      " orders.tenant_id ",
		Operator:    " equals ",
		Values:      []string{" tenant-a ", ""},
		Conditions: []models.AttributeCondition{{
			Attribute: " roles ",
			Operator:  " equals ",
			Values:    []string{" steward ", " "},
		}},
	}}

	columnPolicies := []models.BundleColumnPolicy{{
		ID:          " cp-1 ",
		Name:        " Mask PII ",
		Description: "   ",
		Columns:     []string{" ssn ", " "},
		MaskType:    " redact ",
		MaskValue:   "  ",
		Conditions: []models.AttributeCondition{{
			Attribute: " department ",
			Operator:  " equals ",
			Values:    []string{" finance "},
		}},
	}}

	if _, err := bundleSvc.UpdateBundlePolicies(steward, bundle.ID, rowPolicies, columnPolicies); err != nil {
		t.Fatalf("UpdateBundlePolicies returned error: %v", err)
	}

	updated, err := bundleSvc.GetBundle(steward, bundle.ID)
	if err != nil {
		t.Fatalf("GetBundle returned error: %v", err)
	}

	if updated.RowPolicies[0].ID != "rp-1" {
		t.Fatalf("expected row policy ID to be sanitized, got %q", updated.RowPolicies[0].ID)
	}
	if updated.RowPolicies[0].Name != "Tenant Filter" {
		t.Fatalf("expected trimmed name, got %q", updated.RowPolicies[0].Name)
	}
	if updated.RowPolicies[0].Operator != "equals" {
		t.Fatalf("expected trimmed operator, got %q", updated.RowPolicies[0].Operator)
	}
	if len(updated.RowPolicies[0].Values) != 1 || updated.RowPolicies[0].Values[0] != "tenant-a" {
		t.Fatalf("expected sanitized values, got %#v", updated.RowPolicies[0].Values)
	}
	if updated.RowPolicies[0].Conditions[0].Attribute != "roles" {
		t.Fatalf("expected trimmed condition attribute, got %q", updated.RowPolicies[0].Conditions[0].Attribute)
	}
	if len(updated.ColumnPolicies[0].Columns) != 1 || updated.ColumnPolicies[0].Columns[0] != "ssn" {
		t.Fatalf("expected sanitized columns, got %#v", updated.ColumnPolicies[0].Columns)
	}
	if updated.ColumnPolicies[0].MaskType != "redact" {
		t.Fatalf("expected trimmed mask type, got %q", updated.ColumnPolicies[0].MaskType)
	}
	if updated.ColumnPolicies[0].MaskValue != "" {
		t.Fatalf("expected empty mask value, got %q", updated.ColumnPolicies[0].MaskValue)
	}
}
