package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
)

// These cases are the direct regression test for the Tier 0 write-path
// vulnerability documented in backend/docs/INCIDENT_REPORT_20260906.md:
// extractTenantUUIDFromRequest previously fell back to an unauthenticated
// caller's raw X-Tenant-ID header, or to a hardcoded phantom tenant UUID,
// letting unauthenticated writes land under a guessed tenant instead of
// being rejected.

func requestWithClaims(claims *jwtmiddleware.JWTClaims, tenantHeader string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/bo/thing/records", nil)
	if tenantHeader != "" {
		r.Header.Set("X-Tenant-ID", tenantHeader)
	}
	if claims != nil {
		ctx := context.WithValue(r.Context(), jwtmiddleware.ClaimsContextKey, claims)
		r = r.WithContext(ctx)
	}
	return r
}

func TestExtractTenantUUID_NoClaims_Rejected(t *testing.T) {
	r := requestWithClaims(nil, "11111111-1111-1111-1111-111111111111")
	_, err := extractTenantUUIDFromRequest(r)
	if err == nil {
		t.Fatal("expected rejection: no JWT claims in context")
	}
}

func TestExtractTenantUUID_NoClaims_NoHeader_Rejected(t *testing.T) {
	// This is the direct negative test for the hardcoded-phantom-tenant
	// fallback: no claims, no header - must never default to a guessed UUID.
	r := requestWithClaims(nil, "")
	got, err := extractTenantUUIDFromRequest(r)
	if err == nil {
		t.Fatalf("expected rejection, got tenant %q", got)
	}
}

func TestExtractTenantUUID_ClaimsOnly_UsesClaimTenant(t *testing.T) {
	claims := &jwtmiddleware.JWTClaims{TenantID: "22222222-2222-2222-2222-222222222222"}
	r := requestWithClaims(claims, "")
	got, err := extractTenantUUIDFromRequest(r)
	if err != nil || got.String() != claims.TenantID {
		t.Fatalf("got (%v, %v), want (%q, nil)", got, err, claims.TenantID)
	}
}

func TestExtractTenantUUID_HeaderMismatchNonAdmin_Rejected(t *testing.T) {
	// The exact vulnerable shape: an authenticated caller for tenant A
	// requesting tenant B's scope via the header, with no admin privilege.
	claims := &jwtmiddleware.JWTClaims{TenantID: "22222222-2222-2222-2222-222222222222"}
	r := requestWithClaims(claims, "33333333-3333-3333-3333-333333333333")
	_, err := extractTenantUUIDFromRequest(r)
	if err == nil {
		t.Fatal("expected rejection: header tenant does not match caller's own tenant")
	}
}

func TestExtractTenantUUID_HeaderMatchesOwnTenant_Allowed(t *testing.T) {
	claims := &jwtmiddleware.JWTClaims{TenantID: "22222222-2222-2222-2222-222222222222"}
	r := requestWithClaims(claims, claims.TenantID)
	got, err := extractTenantUUIDFromRequest(r)
	if err != nil || got.String() != claims.TenantID {
		t.Fatalf("got (%v, %v), want (%q, nil)", got, err, claims.TenantID)
	}
}

func TestExtractTenantUUID_HeaderMismatchAdmin_Allowed(t *testing.T) {
	claims := &jwtmiddleware.JWTClaims{TenantID: "22222222-2222-2222-2222-222222222222", IsCoreAdmin: true}
	r := requestWithClaims(claims, "33333333-3333-3333-3333-333333333333")
	got, err := extractTenantUUIDFromRequest(r)
	if err != nil || got.String() != "33333333-3333-3333-3333-333333333333" {
		t.Fatalf("got (%v, %v), want admin override allowed", got, err)
	}
}
