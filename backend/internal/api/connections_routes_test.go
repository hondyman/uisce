package api

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
)

func signTestJWT(t *testing.T, secret string, claims jwtmiddleware.JWTClaims) string {
	t.Helper()
	if claims.RegisteredClaims.ExpiresAt == nil {
		claims.RegisteredClaims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(time.Hour))
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("failed to sign test token: %v", err)
	}
	return signed
}

// TestGetTenantIDFromRequest_TenantIsolation proves that a caller cannot use
// the X-Tenant-ID header to impersonate a tenant outside their own JWT's
// tenant list. Before the fix, getTenantIDFromRequest trusted the raw
// X-Tenant-ID header whenever the JWT was missing/invalid/tenant-less,
// letting anyone claim any tenant.
func TestGetTenantIDFromRequest_TenantIsolation(t *testing.T) {
	secret := "test-secret-with-at-least-32-characters!!"
	t.Setenv("JWT_SECRET", secret)

	t.Run("no JWT at all: header must not be trusted", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/connections", nil)
		req.Header.Set("X-Tenant-ID", "tenant-victim")

		got := getTenantIDFromRequest(req)
		if got != "" {
			t.Fatalf("expected empty tenant when no JWT present, got %q (raw header was trusted)", got)
		}
	})

	t.Run("valid JWT with no tenant claim: header must not be trusted", func(t *testing.T) {
		token := signTestJWT(t, secret, jwtmiddleware.JWTClaims{
			UserID:   "user-attacker",
			IsActive: true,
			// TenantID / TenantIDs intentionally empty
			Roles: []string{"user"},
		})
		req := httptest.NewRequest("GET", "/connections", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("X-Tenant-ID", "tenant-victim")

		got := getTenantIDFromRequest(req)
		if got != "" {
			t.Fatalf("expected empty tenant when JWT has no tenant claim, got %q (raw header was trusted)", got)
		}
	})

	t.Run("valid JWT for a different tenant: header must not override it", func(t *testing.T) {
		token := signTestJWT(t, secret, jwtmiddleware.JWTClaims{
			UserID:   "user-attacker",
			IsActive: true,
			TenantID: "tenant-attacker",
			Roles:    []string{"user"},
		})
		req := httptest.NewRequest("GET", "/connections", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("X-Tenant-ID", "tenant-victim")

		got := getTenantIDFromRequest(req)
		if got == "tenant-victim" {
			t.Fatalf("cross-tenant IDOR: X-Tenant-ID header overrode the JWT's own tenant, got %q", got)
		}
	})

	t.Run("global admin JWT may select any tenant via header", func(t *testing.T) {
		token := signTestJWT(t, secret, jwtmiddleware.JWTClaims{
			UserID:      "user-admin",
			IsActive:    true,
			IsCoreAdmin: true,
		})
		req := httptest.NewRequest("GET", "/connections", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("X-Tenant-ID", "tenant-123")

		got := getTenantIDFromRequest(req)
		if got != "tenant-123" {
			t.Fatalf("expected admin to be able to select tenant-123, got %q", got)
		}
	})
}

// TestGetTenantUUIDFromRequest_RebaseHandlers_TenantIsolation proves that
// rebase_handlers.go's getTenantUUIDFromRequest (governs 3-way graph rebase
// dry-run/apply, a write path) cannot be tricked into acting on a tenant
// outside the caller's own JWT via the X-Tenant-ID header.
func TestGetTenantUUIDFromRequest_RebaseHandlers_TenantIsolation(t *testing.T) {
	secret := "test-secret-with-at-least-32-characters!!"
	t.Setenv("JWT_SECRET", secret)

	t.Run("no JWT: header must not be trusted", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/governance/rebase/apply", nil)
		req.Header.Set("X-Tenant-ID", "11111111-1111-1111-1111-111111111111")

		if got := getTenantUUIDFromRequest(req); got != uuid.Nil {
			t.Fatalf("expected uuid.Nil with no JWT, got %v (raw header was trusted)", got)
		}
	})

	t.Run("valid JWT for a different tenant: header must not override it", func(t *testing.T) {
		token := signTestJWT(t, secret, jwtmiddleware.JWTClaims{
			UserID:   "user-attacker",
			IsActive: true,
			TenantID: "22222222-2222-2222-2222-222222222222",
			Roles:    []string{"user"},
		})
		req := httptest.NewRequest("POST", "/governance/rebase/apply", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("X-Tenant-ID", "11111111-1111-1111-1111-111111111111")

		got := getTenantUUIDFromRequest(req)
		if got.String() == "11111111-1111-1111-1111-111111111111" {
			t.Fatalf("cross-tenant IDOR: X-Tenant-ID header overrode the JWT's own tenant for a rebase-apply write, got %v", got)
		}
	})
}

// TestExtractTenantID_DataQualityHandler_TenantIsolation proves that
// data_quality_handlers.go's extractTenantID cannot be tricked into
// auditing/mutating a different tenant's data via the X-Tenant-ID header.
func TestExtractTenantID_DataQualityHandler_TenantIsolation(t *testing.T) {
	secret := "test-secret-with-at-least-32-characters!!"
	t.Setenv("JWT_SECRET", secret)
	h := &DataQualityHandler{}

	t.Run("no JWT: header must not be trusted", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/business-objects/1/quality-audit", nil)
		req.Header.Set("X-Tenant-ID", "11111111-1111-1111-1111-111111111111")

		if got := h.extractTenantID(req); got != uuid.Nil {
			t.Fatalf("expected uuid.Nil with no JWT, got %v (raw header was trusted)", got)
		}
	})

	t.Run("valid JWT for a different tenant: header must not override it", func(t *testing.T) {
		token := signTestJWT(t, secret, jwtmiddleware.JWTClaims{
			UserID:   "user-attacker",
			IsActive: true,
			TenantID: "22222222-2222-2222-2222-222222222222",
			Roles:    []string{"user"},
		})
		req := httptest.NewRequest("POST", "/business-objects/1/quality-audit", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("X-Tenant-ID", "11111111-1111-1111-1111-111111111111")

		got := h.extractTenantID(req)
		if got.String() == "11111111-1111-1111-1111-111111111111" {
			t.Fatalf("cross-tenant IDOR: X-Tenant-ID header overrode the JWT's own tenant for a quality-audit trigger, got %v", got)
		}
	})
}
