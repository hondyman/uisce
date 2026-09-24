package services_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hondyman/uisce/backend/internal/middleware"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/hondyman/uisce/backend/internal/services"
)

const mintRoundTripSecret = "test-jwt-secret-for-mint-roundtrip-only"

func TestMintValidate_RoundTrip(t *testing.T) {
	sm := services.NewSecurityManager(nil, nil, []byte(mintRoundTripSecret))
	const (
		userID = "mint-user"
		email  = "mint@example.com"
		tenant = "99e99e99-99e9-49e9-89e9-99e99e99e999"
		role   = "portfolio_manager"
	)

	token, err := sm.MintDevToken(services.DevTokenInput{
		UserID:    userID,
		Email:     email,
		TenantIDs: []string{tenant},
		Roles:     []string{role},
	})
	if err != nil {
		t.Fatalf("MintDevToken: %v", err)
	}

	claims, err := sm.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}
	if claims.UserID != userID {
		t.Fatalf("ValidateToken UserID=%q want %q", claims.UserID, userID)
	}
	if claims.Email != email {
		t.Fatalf("ValidateToken Email=%q want %q", claims.Email, email)
	}
	if claims.TenantID != tenant {
		t.Fatalf("ValidateToken TenantID=%q want %q", claims.TenantID, tenant)
	}
	if len(claims.TenantIDs) != 1 || claims.TenantIDs[0] != tenant {
		t.Fatalf("ValidateToken TenantIDs=%v want [%s]", claims.TenantIDs, tenant)
	}
	if len(claims.Roles) != 1 || claims.Roles[0] != role {
		t.Fatalf("ValidateToken Roles=%v want [%s]", claims.Roles, role)
	}

	var got security.AuthInfo
	var ok bool
	h := middleware.AuthContextMiddleware(sm)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, ok = security.AuthInfoFromContext(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if !ok {
		t.Fatal("AuthContextMiddleware did not inject AuthInfo")
	}
	if got.UserID != userID {
		t.Fatalf("AuthInfo.UserID=%q want %q", got.UserID, userID)
	}
	if len(got.TenantIDs) != 1 || got.TenantIDs[0] != tenant {
		t.Fatalf("AuthInfo.TenantIDs=%v want [%s]", got.TenantIDs, tenant)
	}
	if len(got.Roles) != 1 || got.Roles[0] != role {
		t.Fatalf("AuthInfo.Roles=%v want [%s]", got.Roles, role)
	}
	if got.IsGlobalAdmin {
		t.Fatal("AuthInfo.IsGlobalAdmin unexpectedly true")
	}
}

func TestMintDevToken_RequiresUserID(t *testing.T) {
	sm := services.NewSecurityManager(nil, nil, []byte(mintRoundTripSecret))
	_, err := sm.MintDevToken(services.DevTokenInput{TenantIDs: []string{"t"}})
	if err == nil {
		t.Fatal("expected error for empty user_id")
	}
}
