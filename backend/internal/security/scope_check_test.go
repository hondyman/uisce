package security

import (
	"context"
	"errors"
	"testing"
)

// TestIsWithinScope_TenantScope exercises the scope check for tenant-wide impersonation.
func TestIsWithinScope_TenantScope(t *testing.T) {
	resolved := ResolvedDatasource{
		TenantID:     "tenant-A",
		InstanceID:   "instance-X",
		ProductID:    "product-Y",
		DatasourceID: "datasource-Z",
	}

	cases := []struct {
		name     string
		scope    string
		scopeID  string
		expected bool
	}{
		{"tenant scope allows anything", ScopeTenant, "", true},
		{"empty scope_kind defaults to tenant", "", "", true},
		{"tenant scope with non-empty scope_id still allows", ScopeTenant, "anything", true},
		{"instance scope matches by instance_id", ScopeInstance, "instance-X", true},
		{"instance scope rejects different instance", ScopeInstance, "instance-Y", false},
		{"product scope matches by product_id", ScopeProduct, "product-Y", true},
		{"product scope rejects different product", ScopeProduct, "product-Z", false},
		{"datasource scope matches by datasource_id", ScopeDatasource, "datasource-Z", true},
		{"datasource scope rejects different datasource", ScopeDatasource, "datasource-W", false},
		{"unknown scope_kind denied by default", "made_up_scope", "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsWithinScope(tc.scope, tc.scopeID, resolved)
			if got != tc.expected {
				t.Errorf("IsWithinScope(%q, %q, ...) = %v, want %v",
					tc.scope, tc.scopeID, got, tc.expected)
			}
		})
	}
}

// TestValidateScope wraps ErrImpersonationScopeViolation correctly.
func TestValidateScope(t *testing.T) {
	resolved := ResolvedDatasource{
		TenantID:     "tenant-A",
		InstanceID:   "instance-X",
		ProductID:    "product-Y",
		DatasourceID: "datasource-Z",
	}

	// Inside scope → nil.
	if err := ValidateScope(ScopeTenant, "", resolved); err != nil {
		t.Errorf("ValidateScope(tenant, ...) should return nil, got %v", err)
	}

	// Outside scope → wrapped error.
	err := ValidateScope(ScopeInstance, "instance-W", resolved)
	if err == nil {
		t.Fatal("ValidateScope(instance, wrong id) should return an error")
	}
	if !errors.Is(err, ErrImpersonationScopeViolation) {
		t.Errorf("error should wrap ErrImpersonationScopeViolation, got %v", err)
	}

	// Verify helpers.IsImpersonationScopeViolation recognises it.
	if !IsImpersonationScopeViolation(err) {
		t.Error("IsImpersonationScopeViolation should detect the wrapped error")
	}
}

// TestIsImpersonationScopeViolation_NilForOtherErrors makes sure unrelated errors
// are not misclassified as scope violations.
func TestIsImpersonationScopeViolation_NilForOtherErrors(t *testing.T) {
	if IsImpersonationScopeViolation(errors.New("some other error")) {
		t.Error("unrelated error should not be classified as a scope violation")
	}
	if IsImpersonationScopeViolation(nil) {
		t.Error("nil error should not be classified as a scope violation")
	}
}

// TestImpersonationScopeContext_RoundTrip exercises the context-plumbing helpers.
func TestImpersonationScopeContext_RoundTrip(t *testing.T) {
	ctx := context.Background()

	// Default (no scope set) → tenant-wide.
	if got := ImpersonationScopeFromContext(ctx); got.Kind != ScopeTenant {
		t.Errorf("default scope should be %q, got %q", ScopeTenant, got.Kind)
	}

	// Set a non-default scope and read it back.
	scoped := ctx
	scoped = WithImpersonationScope(scoped, ImpersonationScopeContext{Kind: ScopeInstance, ID: "inst-1"})
	got := ImpersonationScopeFromContext(scoped)
	if got.Kind != ScopeInstance || got.ID != "inst-1" {
		t.Errorf("scope round-trip failed: got %+v", got)
	}
}