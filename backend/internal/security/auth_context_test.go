package security

import "testing"

// These cases are the canonical tenant-resolution rule this repo's
// 2026-09-07 security sweep found six-plus independent implementations of,
// including the 67-call-site SecurityContextFromRequest, which accepted a
// client-supplied tenant unconditionally. Every case here corresponds
// directly to a real finding from that sweep.

func TestResolveTenantID_NoOverride_ReturnsOwnTenant(t *testing.T) {
	auth := AuthInfo{TenantIDs: []string{"tenant-a"}}
	got, ok := ResolveTenantID(auth, "")
	if !ok || got != "tenant-a" {
		t.Fatalf("got (%q, %v), want (\"tenant-a\", true)", got, ok)
	}
}

func TestResolveTenantID_NoOverride_NoOwnTenant_Rejected(t *testing.T) {
	auth := AuthInfo{}
	_, ok := ResolveTenantID(auth, "")
	if ok {
		t.Fatal("expected rejection: caller has no tenant and requested none")
	}
}

func TestResolveTenantID_MatchingRequest_Allowed(t *testing.T) {
	auth := AuthInfo{TenantIDs: []string{"tenant-a", "tenant-b"}}
	got, ok := ResolveTenantID(auth, "tenant-b")
	if !ok || got != "tenant-b" {
		t.Fatalf("got (%q, %v), want (\"tenant-b\", true)", got, ok)
	}
}

// This is the exact shape of the 67-call-site vulnerability: an
// authenticated caller for tenant A requesting tenant B's scope, with no
// admin privilege. The old SecurityContextFromRequest accepted this
// unconditionally; the canonical rule must reject it.
func TestResolveTenantID_NonMatchingRequest_NonAdmin_Rejected(t *testing.T) {
	auth := AuthInfo{TenantIDs: []string{"tenant-a"}, IsGlobalAdmin: false}
	_, ok := ResolveTenantID(auth, "tenant-b")
	if ok {
		t.Fatal("expected rejection: non-admin caller requested a tenant not their own")
	}
}

// Legitimate cross-tenant pivot: global admin support tooling.
func TestResolveTenantID_NonMatchingRequest_GlobalAdmin_Allowed(t *testing.T) {
	auth := AuthInfo{TenantIDs: []string{"tenant-a"}, IsGlobalAdmin: true}
	got, ok := ResolveTenantID(auth, "tenant-b")
	if !ok || got != "tenant-b" {
		t.Fatalf("got (%q, %v), want (\"tenant-b\", true) for a global admin", got, ok)
	}
}

func TestResolveTenantID_NeverDefaults(t *testing.T) {
	// No auth.TenantIDs, no requested tenant, not an admin - must reject,
	// never fabricate or fall back to a default tenant. This is the direct
	// negative test for bo_crud_handler.go's hardcoded-UUID-fallback finding,
	// the single worst idiom the sweep found.
	auth := AuthInfo{}
	got, ok := ResolveTenantID(auth, "")
	if ok || got != "" {
		t.Fatalf("got (%q, %v), want (\"\", false) - must never default", got, ok)
	}
}
