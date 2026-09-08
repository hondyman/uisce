package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hondyman/uisce/backend/internal/security"
)

// These cases are the direct regression test for the Tier 0 write-path
// vulnerability documented in backend/docs/INCIDENT_REPORT_20260906.md:
// extractTenantUUIDFromRequest previously fell back to an unauthenticated
// caller's raw X-Tenant-ID header, or to a hardcoded phantom tenant UUID,
// letting unauthenticated writes land under a guessed tenant instead of
// being rejected.
//
// requestWithAuth sets security.AuthInfo directly (the context key that
// appmid.AuthContextMiddleware actually populates on this router) rather
// than jwtmiddleware claims, which an earlier revision of this fix read
// from - a context key nothing on this router ever sets, caught only by
// HTTP replay against a running server before merge.

func requestWithAuth(auth *security.AuthInfo, tenantHeader string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/bo/thing/records", nil)
	if tenantHeader != "" {
		r.Header.Set("X-Tenant-ID", tenantHeader)
	}
	if auth != nil {
		ctx := security.WithAuthInfo(r.Context(), *auth)
		r = r.WithContext(ctx)
	}
	return r
}

func statusOf(t *testing.T, err error) int {
	t.Helper()
	te, ok := err.(*tenantResolutionError)
	if !ok {
		t.Fatalf("expected *tenantResolutionError, got %T: %v", err, err)
	}
	return te.status
}

func TestExtractTenantUUID_NoAuth_Rejected(t *testing.T) {
	r := requestWithAuth(nil, "11111111-1111-1111-1111-111111111111")
	_, err := extractTenantUUIDFromRequest(r)
	if err == nil {
		t.Fatal("expected rejection: no AuthInfo in context")
	}
	if got := statusOf(t, err); got != http.StatusUnauthorized {
		t.Fatalf("no auth at all must be 401, got %d", got)
	}
}

func TestExtractTenantUUID_NoAuth_NoHeader_Rejected(t *testing.T) {
	// This is the direct negative test for the hardcoded-phantom-tenant
	// fallback: no auth, no header - must never default to a guessed UUID.
	r := requestWithAuth(nil, "")
	got, err := extractTenantUUIDFromRequest(r)
	if err == nil {
		t.Fatalf("expected rejection, got tenant %q", got)
	}
	if status := statusOf(t, err); status != http.StatusUnauthorized {
		t.Fatalf("no auth at all must be 401, got %d", status)
	}
}

func TestExtractTenantUUID_AuthOnly_UsesOwnTenant(t *testing.T) {
	auth := &security.AuthInfo{TenantIDs: []string{"22222222-2222-2222-2222-222222222222"}}
	r := requestWithAuth(auth, "")
	got, err := extractTenantUUIDFromRequest(r)
	if err != nil || got.String() != auth.TenantIDs[0] {
		t.Fatalf("got (%v, %v), want (%q, nil)", got, err, auth.TenantIDs[0])
	}
}

func TestExtractTenantUUID_HeaderMismatchNonAdmin_Rejected(t *testing.T) {
	// The exact vulnerable shape: an authenticated caller for tenant A
	// requesting tenant B's scope via the header, with no admin privilege.
	auth := &security.AuthInfo{TenantIDs: []string{"22222222-2222-2222-2222-222222222222"}}
	r := requestWithAuth(auth, "33333333-3333-3333-3333-333333333333")
	_, err := extractTenantUUIDFromRequest(r)
	if err == nil {
		t.Fatal("expected rejection: header tenant does not match caller's own tenant")
	}
	// Authenticated, just not for that tenant - must be 403, not 401.
	if status := statusOf(t, err); status != http.StatusForbidden {
		t.Fatalf("authenticated cross-tenant pivot must be 403, got %d", status)
	}
}

func TestExtractTenantUUID_HeaderMatchesOwnTenant_Allowed(t *testing.T) {
	auth := &security.AuthInfo{TenantIDs: []string{"22222222-2222-2222-2222-222222222222"}}
	r := requestWithAuth(auth, auth.TenantIDs[0])
	got, err := extractTenantUUIDFromRequest(r)
	if err != nil || got.String() != auth.TenantIDs[0] {
		t.Fatalf("got (%v, %v), want (%q, nil)", got, err, auth.TenantIDs[0])
	}
}

func TestExtractTenantUUID_HeaderMismatchAdmin_Allowed(t *testing.T) {
	auth := &security.AuthInfo{TenantIDs: []string{"22222222-2222-2222-2222-222222222222"}, IsGlobalAdmin: true}
	r := requestWithAuth(auth, "33333333-3333-3333-3333-333333333333")
	got, err := extractTenantUUIDFromRequest(r)
	if err != nil || got.String() != "33333333-3333-3333-3333-333333333333" {
		t.Fatalf("got (%v, %v), want admin override allowed", got, err)
	}
}
