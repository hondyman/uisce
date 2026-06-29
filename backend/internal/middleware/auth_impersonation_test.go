package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/security"
	"github.com/stretchr/testify/require"
)

// recordingAudit is a minimal audit double for producing impersonation tokens.
type recordingAudit struct{}

func (r *recordingAudit) LogStart(_ context.Context, _ security.ImpersonationSession) error { return nil }
func (r *recordingAudit) LogEnd(_ context.Context, _ security.ImpersonationSession) error   { return nil }
func (r *recordingAudit) LogBreakGlassAction(_ context.Context, _ uuid.UUID, _ string, _ uuid.UUID, _ map[string]any) error {
	return nil
}
func (r *recordingAudit) LogImpersonationAction(_ context.Context, _ *sql.Tx, _ security.ImpersonationAction) error {
	return nil
}
func (r *recordingAudit) ListExpiredActiveSessions(_ context.Context) ([]security.ImpersonationSession, error) {
	return nil, nil
}
func (r *recordingAudit) LogExpired(_ context.Context, _ security.ImpersonationSession) error { return nil }

func TestAuthContextMiddleware_ImpersonationToken_PreservesRealRoles(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-for-middleware-tests")

	sm := newTestSecurityManager()
	userID := uuid.NewString()
	tenantID := uuid.NewString()

	svc := security.NewContextExchangeService(&recordingAudit{}, security.ImpersonationPolicy{})
	token, err := svc.AssumeTenantContext(
		nil,
		userID,
		"pro@example.com",
		[]string{security.RoleProfessionalServices},
		security.ImpersonationRequest{
			TargetTenantID:  uuid.MustParse(tenantID),
			Reason:          "tenant administration for ticket PS-12345",
			TicketReference: "PS-12345",
			Mode:            security.ModeBreakGlass,
			Duration:        30 * time.Minute,
		},
	)
	require.NoError(t, err)

	mw := AuthContextMiddleware(sm)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authInfo, ok := security.AuthInfoFromContext(r.Context())
		require.True(t, ok)
		require.Equal(t, userID, authInfo.UserID)
		require.Equal(t, tenantID, authInfo.TenantIDs[0])
		require.Contains(t, authInfo.Roles, security.RoleProfessionalServices)
		require.True(t, authInfo.ImpersonationActive)
		require.Equal(t, security.RoleProfessionalServices, authInfo.ImpersonationAdminRole)
		require.False(t, authInfo.IsGlobalAdmin)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
}

func TestAuthContextMiddleware_ImpersonationToken_LegacyFallback(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-for-middleware-tests")

	sm := newTestSecurityManager()
	tenantID := uuid.NewString()

	// Manually craft a legacy payload (no admin_role or real_roles) to simulate
	// tokens issued before multi-role support.
	legacyToken, err := signLegacyImpersonationToken(tenantID)
	require.NoError(t, err)

	mw := AuthContextMiddleware(sm)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authInfo, ok := security.AuthInfoFromContext(r.Context())
		require.True(t, ok)
		require.True(t, authInfo.ImpersonationActive)
		require.Equal(t, security.RoleGlobalAdmin, authInfo.ImpersonationAdminRole)
		require.True(t, authInfo.IsGlobalAdmin)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+legacyToken)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
}

// signLegacyImpersonationToken creates a token payload identical to the format
// used before multi-role support (no admin_role, real_roles, or scope fields).
func signLegacyImpersonationToken(tenantID string) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	payload := map[string]any{
		"sub":                  "legacy-admin",
		"admin_email":          "legacy@example.com",
		"tenant_id":            tenantID,
		"impersonation_active": true,
		"session_id":           uuid.NewString(),
		"mode":                 string(security.ModeReadOnly),
		"exp":                  time.Now().Add(time.Hour).Unix(),
		"iat":                  time.Now().Unix(),
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payloadBytes)
	sig := mac.Sum(nil)
	return base64.RawURLEncoding.EncodeToString(payloadBytes) + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}
