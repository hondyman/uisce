package exceptions

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestComputeFingerprint_StableAcrossEvidenceOrder(t *testing.T) {
	tenant := uuid.New()
	fp1 := ComputeFingerprint(tenant, ExceptionSLOBreach, "page:dashboard", []string{"p95: 342ms", "threshold: 300ms"})
	fp2 := ComputeFingerprint(tenant, ExceptionSLOBreach, "page:dashboard", []string{"threshold: 300ms", "p95: 342ms"})
	if fp1 != fp2 {
		t.Fatalf("expected fingerprints to match regardless of evidence order, got %q vs %q", fp1, fp2)
	}
}

func TestComputeFingerprint_StableCaseAndWhitespace(t *testing.T) {
	tenant := uuid.New()
	fp1 := ComputeFingerprint(tenant, ExceptionDataQuality, "table:positions", []string{"Missing: 12%"})
	fp2 := ComputeFingerprint(tenant, ExceptionDataQuality, "table:positions", []string{"  missing: 12%  "})
	if fp1 != fp2 {
		t.Fatalf("expected fingerprints to be case/whitespace-insensitive, got %q vs %q", fp1, fp2)
	}
}

func TestComputeFingerprint_DiffersByType(t *testing.T) {
	tenant := uuid.New()
	fp1 := ComputeFingerprint(tenant, ExceptionSLOBreach, "page:dashboard", []string{"evidence"})
	fp2 := ComputeFingerprint(tenant, ExceptionDataQuality, "page:dashboard", []string{"evidence"})
	if fp1 == fp2 {
		t.Fatalf("expected fingerprints to differ by exception type")
	}
}

func TestComputeFingerprint_DiffersByTenant(t *testing.T) {
	fp1 := ComputeFingerprint(uuid.New(), ExceptionSLOBreach, "page:dashboard", []string{"evidence"})
	fp2 := ComputeFingerprint(uuid.New(), ExceptionSLOBreach, "page:dashboard", []string{"evidence"})
	if fp1 == fp2 {
		t.Fatalf("expected fingerprints to differ by tenant (dedup must never cross tenants)")
	}
}

func TestComputeFingerprint_DiffersBySource(t *testing.T) {
	tenant := uuid.New()
	fp1 := ComputeFingerprint(tenant, ExceptionSLOBreach, "page:dashboard", []string{"evidence"})
	fp2 := ComputeFingerprint(tenant, ExceptionSLOBreach, "page:client_profile", []string{"evidence"})
	if fp1 == fp2 {
		t.Fatalf("expected fingerprints to differ by source")
	}
}

// TestPublish_NilDB_DoesNotPanic exercises the degrade path used in
// dev/test environments without Postgres wired up (per plan: "don't block
// on infra you don't have").
func TestPublish_NilDB_DoesNotPanic(t *testing.T) {
	ea := NewExceptionAggregator(nil)
	exc, err := ea.Publish(context.Background(), Exception{
		TenantID: uuid.New(),
		Type:     ExceptionSLOBreach,
		Severity: "high",
		Source:   "page:dashboard",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exc == nil {
		t.Fatalf("expected non-nil exception echoed back")
	}
}

// TestResolveAutofixPolicy_NilDB_DefaultsClosed verifies the safe default:
// with no policy configured (or no DB at all), auto-fix must resolve to
// disabled, never silently enabled.
func TestResolveAutofixPolicy_NilDB_DefaultsClosed(t *testing.T) {
	ea := NewExceptionAggregator(nil)
	tenant := uuid.New()
	user := uuid.New()

	policy, err := ea.ResolveAutofixPolicy(context.Background(), tenant, &user, ExceptionSemanticDrift)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policy.Enabled {
		t.Fatalf("expected autofix to default to disabled when unconfigured")
	}
	if !policy.RequiresApproval {
		t.Fatalf("expected default policy to require approval")
	}

	// Also verify the tenant-only (no user override) path defaults closed.
	policyNoUser, err := ea.ResolveAutofixPolicy(context.Background(), tenant, nil, ExceptionSemanticDrift)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policyNoUser.Enabled {
		t.Fatalf("expected tenant-default autofix policy to default to disabled when unconfigured")
	}
}
