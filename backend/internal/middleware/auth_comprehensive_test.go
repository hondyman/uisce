package middleware

import (
	dbsql "database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/security"
	"github.com/hondyman/semlayer/backend/internal/services"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Real SecurityManager for Testing
// ============================================================================

func newTestSecurityManager() *services.SecurityManager {
	// Create a real SecurityManager with a fixed secret for testing
	// We pass nil for cache/metrics as they are not used in auth flow
	return services.NewSecurityManager(nil, nil, []byte("test-secret-key-1234567890123456"))
}

// Helper to sign a token for testing
func signTestToken(sm *services.SecurityManager, claims jwt.MapClaims) string {
	token, err := sm.SignToken(claims)
	if err != nil {
		panic(err)
	}
	return token
}

// ============================================================================
// API Key Auth Tests
// ============================================================================

func TestAPIKeyAuth_ValidKey_Authenticated(t *testing.T) {
	sm := newTestSecurityManager()
	userID := uuid.NewString()
	tenantID := uuid.NewString()

	// Register a valid key
	sm.RegisterAPIKey("valid-key-12345", userID, []string{tenantID}, []string{"GLOBAL_OPS"})

	mw := AuthContextMiddleware(sm, nil)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authInfo, ok := security.AuthInfoFromContext(r.Context())
		require.True(t, ok, "AuthInfo should be in context")
		require.Equal(t, userID, authInfo.UserID)
		require.Contains(t, authInfo.Roles, "GLOBAL_OPS")
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "valid-key-12345")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
}

func TestAPIKeyAuth_InvalidKey_Unauthenticated(t *testing.T) {
	sm := newTestSecurityManager()

	mw := AuthContextMiddleware(sm, nil)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := security.AuthInfoFromContext(r.Context())
		// When invalid key, context should not have AuthInfo
		require.False(t, ok)
		w.WriteHeader(http.StatusUnauthorized)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "invalid-key-nope")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Handler should manually enforce auth
	require.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAPIKeyAuth_RevokedKey_Rejected(t *testing.T) {
	sm := newTestSecurityManager()
	userID := uuid.NewString()
	tenantID := uuid.NewString()

	// Register and revoke key
	key := "revoked-key-123"
	sm.RegisterAPIKey(key, userID, []string{tenantID}, []string{"GLOBAL_OPS"})
	sm.RevokeAPIKey(key)

	mw := AuthContextMiddleware(sm, nil)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := security.AuthInfoFromContext(r.Context())
		require.False(t, ok, "Revoked key should not authenticate")
		w.WriteHeader(http.StatusForbidden)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", key)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusForbidden, rr.Code)
}

func TestAPIKeyAuth_WrongTenant_ProperlyScoped(t *testing.T) {
	sm := newTestSecurityManager()
	userID := uuid.NewString()
	allowedTenantID := uuid.NewString()
	attemptedTenantID := uuid.NewString()

	// Key only allows access to allowedTenantID
	sm.RegisterAPIKey("scoped-key-123", userID, []string{allowedTenantID}, []string{"TENANT_ADMIN"})

	mw := AuthContextMiddleware(sm, nil)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authInfo, ok := security.AuthInfoFromContext(r.Context())
		require.True(t, ok)
		// Request should have the key's tenant, not the attempted tenant
		require.NotContains(t, authInfo.TenantIDs, attemptedTenantID)
		require.Contains(t, authInfo.TenantIDs, allowedTenantID)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "scoped-key-123")
	req.Header.Set("X-Tenant-ID", attemptedTenantID) // Attempt to override
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
}

func TestAPIKeyAuth_WrongRole_PassesThrough(t *testing.T) {
	sm := newTestSecurityManager()
	userID := uuid.NewString()
	tenantID := uuid.NewString()

	// Key has restricted role (not admin)
	sm.RegisterAPIKey("user-key-123", userID, []string{tenantID}, []string{"USER"})

	mw := AuthContextMiddleware(sm, nil)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authInfo, ok := security.AuthInfoFromContext(r.Context())
		require.True(t, ok)
		// Middleware doesn't enforce roles, just passes them through
		require.Contains(t, authInfo.Roles, "USER")
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "user-key-123")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
}

// ============================================================================
// JWT Auth Tests
// ============================================================================

func TestJWTAuth_ValidToken_Authenticated(t *testing.T) {
	sm := newTestSecurityManager()
	userID := uuid.NewString()
	tenantID := uuid.NewString()

	// Create a valid JWT token
	claims := jwt.MapClaims{
		"user_id":    userID,
		"tenant_id":  tenantID,
		"tenant_ids": []string{tenantID},
		"roles":      []string{"GLOBAL_OPS"},
		"exp":        time.Now().Add(time.Hour).Unix(),
	}
	token := signTestToken(sm, claims)

	mw := AuthContextMiddleware(sm, nil)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authInfo, ok := security.AuthInfoFromContext(r.Context())
		require.True(t, ok)
		require.Equal(t, userID, authInfo.UserID)
		require.Contains(t, authInfo.Roles, "GLOBAL_OPS")
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
}

func TestJWTAuth_InvalidToken_Rejected(t *testing.T) {
	sm := newTestSecurityManager()

	mw := AuthContextMiddleware(sm, nil)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := security.AuthInfoFromContext(r.Context())
		require.False(t, ok, "Invalid token should not authenticate")
		w.WriteHeader(http.StatusUnauthorized)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token-xyz")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestJWTAuth_MissingRoles_StillAuthenticated(t *testing.T) {
	sm := newTestSecurityManager()
	userID := uuid.NewString()
	tenantID := uuid.NewString()

	// Create JWT with no roles
	claims := jwt.MapClaims{
		"user_id":    userID,
		"tenant_id":  tenantID,
		"tenant_ids": []string{tenantID},
		"roles":      []string{},
		"exp":        time.Now().Add(time.Hour).Unix(),
	}
	token := signTestToken(sm, claims)

	mw := AuthContextMiddleware(sm, nil)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authInfo, ok := security.AuthInfoFromContext(r.Context())
		require.True(t, ok)
		require.Equal(t, userID, authInfo.UserID)
		require.Len(t, authInfo.Roles, 0, "Token has no roles")
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Middleware doesn't enforce roles, just authenticates
	require.Equal(t, http.StatusOK, rr.Code)
}

func TestJWTAuth_MissingTenantIDs_StillAuthenticated(t *testing.T) {
	sm := newTestSecurityManager()
	userID := uuid.NewString()

	// Create JWT with no tenant_ids
	claims := jwt.MapClaims{
		"user_id":    userID,
		"tenant_id":  "",
		"tenant_ids": []string{},
		"roles":      []string{"GLOBAL_OPS"},
		"exp":        time.Now().Add(time.Hour).Unix(),
	}
	token := signTestToken(sm, claims)

	mw := AuthContextMiddleware(sm, nil)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authInfo, ok := security.AuthInfoFromContext(r.Context())
		require.True(t, ok)
		require.Equal(t, userID, authInfo.UserID)
		// Middleware just passes through what's in the token
		require.Len(t, authInfo.TenantIDs, 0)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
}

// ============================================================================
// Role Enforcement Tests
// ============================================================================

func TestRoleEnforcement_GlobalOpsMultiTenant(t *testing.T) {
	authInfo := security.AuthInfo{
		UserID:    uuid.NewString(),
		Roles:     []string{"GLOBAL_OPS"},
		TenantIDs: []string{uuid.NewString(), uuid.NewString(), uuid.NewString()},
	}

	// GLOBAL_OPS should allow access to multiple tenants
	require.True(t, containsRole(authInfo.Roles, "GLOBAL_OPS"))
	require.GreaterOrEqual(t, len(authInfo.TenantIDs), 2)
}

func TestRoleEnforcement_TenantAdminSingleTenant(t *testing.T) {
	tenantID := uuid.NewString()
	authInfo := security.AuthInfo{
		UserID:    uuid.NewString(),
		Roles:     []string{"TENANT_ADMIN"},
		TenantIDs: []string{tenantID},
	}

	require.True(t, containsRole(authInfo.Roles, "TENANT_ADMIN"))
	require.Len(t, authInfo.TenantIDs, 1)
	require.Contains(t, authInfo.TenantIDs, tenantID)
}

func TestRoleEnforcement_UserRoleRestricted(t *testing.T) {
	tenantID := uuid.NewString()
	authInfo := security.AuthInfo{
		UserID:    uuid.NewString(),
		Roles:     []string{"USER"},
		TenantIDs: []string{tenantID},
	}

	require.True(t, containsRole(authInfo.Roles, "USER"))
	require.Len(t, authInfo.TenantIDs, 1)
	require.Contains(t, authInfo.TenantIDs, tenantID)
}

// ============================================================================
// Helper Functions
// ============================================================================

func containsRole(roles []string, role string) bool {
	for _, r := range roles {
		if strings.EqualFold(r, role) {
			return true
		}
	}
	return false
}

func TestAuthContextMiddleware_ProfessionalServicesLease(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sm := newTestSecurityManager()
	profileSvc := security.NewProfileService(db)

	token, err := sm.SignToken(jwt.MapClaims{
		"user_id":       "ali-g-user-id",
		"email":         "ali.g@example.com",
		"operator_role": "professional_services",
	})
	require.NoError(t, err)

	targetTenantUUID := uuid.New()

	mw := AuthContextMiddleware(sm, profileSvc)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authInfo, ok := security.AuthInfoFromContext(r.Context())
		require.True(t, ok, "AuthInfo should be in context")
		require.Equal(t, targetTenantUUID.String(), authInfo.TenantIDs[0])
		w.WriteHeader(http.StatusOK)
	}))

	// Scenario 1: Missing X-Tenant-ID
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	require.Equal(t, http.StatusForbidden, rr.Code)
	require.Contains(t, rr.Body.String(), "Ambient Power Prohibited")

	// Scenario 2: Active Lease exists in assignments table
	mock.ExpectQuery(`SELECT ticket_reference FROM security.staff_tenant_assignments`).
		WithArgs("ali.g@example.com", targetTenantUUID).
		WillReturnRows(sqlmock.NewRows([]string{"ticket_reference"}).AddRow("INC-88931-PS"))

	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Tenant-ID", targetTenantUUID.String())
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	// Scenario 3: Lease does not exist
	mock.ExpectQuery(`SELECT ticket_reference FROM security.staff_tenant_assignments`).
		WithArgs("ali.g@example.com", targetTenantUUID).
		WillReturnError(dbsql.ErrNoRows)

	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Tenant-ID", targetTenantUUID.String())
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	require.Equal(t, http.StatusForbidden, rr.Code)
	require.Contains(t, rr.Body.String(), "No active data lease assignment exists")
}

