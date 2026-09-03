package middleware

import (
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/hondyman/uisce/backend/internal/services"
	"github.com/stretchr/testify/require"
)

// TestPositiveAndNegativeAuthMatrix runs the exact verification matrix demanded
// before production deployment.
func TestPositiveAndNegativeAuthMatrix(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("JWT_SECRET", "test-secret-key-32-bytes-long-minimum!")
	t.Setenv("IMPERSONATION_TOKEN_SECRET", "test-impersonation-secret-key-32b!")
	t.Setenv("KEYCLOAK_ISSUER_URL", "http://localhost:8080/realms/uisce")

	sm := services.NewSecurityManager(nil, nil, []byte(osGetenv("JWT_SECRET", "test-secret-key-32-bytes-long-minimum!")))
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	sm.SetRSAPublicKeys(map[string]*rsa.PublicKey{
		"key-1": &privKey.PublicKey,
	})

	mw := AuthContextMiddleware(sm, nil)

	// 1. Valid dev-mode HMAC token (iss=dev://local) -> request continues, context populated
	t.Run("Valid dev-mode HMAC token", func(t *testing.T) {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user_id":    "user-hmac-1",
			"iss":        "dev://local",
			"tenant_id":  "tenant-1",
			"tenant_ids": []string{"tenant-1"},
			"roles":      []string{"analyst"},
			"exp":        time.Now().Add(time.Hour).Unix(),
		})
		signed, err := token.SignedString([]byte("test-secret-key-32-bytes-long-minimum!"))
		require.NoError(t, err)

		handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			info, ok := security.AuthInfoFromContext(r.Context())
			require.True(t, ok)
			require.Equal(t, "user-hmac-1", info.UserID)
			require.Equal(t, "tenant-1", r.Header.Get("X-Tenant-ID"))
			require.Equal(t, "user-hmac-1", r.Header.Get("X-User-ID"))
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+signed)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		require.Equal(t, http.StatusOK, rr.Code)
	})

	// 2. Valid RS256 token -> context populated, X-User-ID set
	t.Run("Valid RS256 token", func(t *testing.T) {
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
			"user_id":    "user-rsa-1",
			"iss":        "http://localhost:8080/realms/uisce",
			"tenant_id":  "tenant-rsa",
			"tenant_ids": []string{"tenant-rsa"},
			"roles":      []string{"trader"},
			"exp":        time.Now().Add(time.Hour).Unix(),
		})
		token.Header["kid"] = "key-1"
		signed, err := token.SignedString(privKey)
		require.NoError(t, err)

		handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			info, ok := security.AuthInfoFromContext(r.Context())
			require.True(t, ok)
			require.Equal(t, "user-rsa-1", info.UserID)
			require.Equal(t, "user-rsa-1", r.Header.Get("X-User-ID"))
			require.Equal(t, "tenant-rsa", r.Header.Get("X-Tenant-ID"))
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+signed)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		require.Equal(t, http.StatusOK, rr.Code)
	})

	// 3. Valid impersonation token (1 dot, IMPERSONATION_TOKEN_SECRET) -> target-tenant context, flags set
	t.Run("Valid impersonation token", func(t *testing.T) {
		svc := security.NewContextExchangeService(&recordingAudit{}, security.ImpersonationPolicy{})
		token, err := svc.AssumeTenantContext(
			nil,
			"admin-user-1",
			"admin@example.com",
			[]string{security.RoleProfessionalServices},
			security.ImpersonationRequest{
				TargetTenantID:  uuid.MustParse("11111111-1111-1111-1111-111111111111"),
				Reason:          "support ticket 123",
				TicketReference: "123",
				Mode:            security.ModeReadOnly,
				Duration:        15 * time.Minute,
			},
		)
		require.NoError(t, err)

		handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			info, ok := security.AuthInfoFromContext(r.Context())
			require.True(t, ok)
			require.True(t, info.ImpersonationActive)
			require.Equal(t, "admin-user-1", info.RealAdminUserID)
			require.Equal(t, "11111111-1111-1111-1111-111111111111", r.Header.Get("X-Tenant-ID"))
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+token.AccessToken)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		require.Equal(t, http.StatusOK, rr.Code)
	})

	// 4. Valid API key -> context populated
	t.Run("Valid API key", func(t *testing.T) {
		apiKey := sm.GenerateAPIKey("api-user-1", "api-tenant-1", []string{"service"})

		handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			info, ok := security.AuthInfoFromContext(r.Context())
			require.True(t, ok)
			require.Equal(t, "api-user-1", info.UserID)
			require.Equal(t, "api-tenant-1", r.Header.Get("X-Tenant-ID"))
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-API-Key", apiKey)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		require.Equal(t, http.StatusOK, rr.Code)
	})

	// 5. Global-admin token + captured tenant header -> tenant selected and re-injected
	t.Run("Global admin tenant selection", func(t *testing.T) {
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
			"user_id": "global-admin-user",
			"iss":     "http://localhost:8080/realms/uisce",
			"roles":   []string{"global_admin"},
			"exp":     time.Now().Add(time.Hour).Unix(),
		})
		token.Header["kid"] = "key-1"
		signed, err := token.SignedString(privKey)
		require.NoError(t, err)

		targetTenant := uuid.NewString()

		handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			info, ok := security.AuthInfoFromContext(r.Context())
			require.True(t, ok)
			require.True(t, info.IsGlobalAdmin)
			require.Equal(t, targetTenant, r.Header.Get("X-Tenant-ID"))
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+signed)
		req.Header.Set("X-Tenant-ID", targetTenant)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		require.Equal(t, http.StatusOK, rr.Code)
	})

	// 6. Spoofed X-User-ID with no credentials -> header absent downstream, anonymous
	t.Run("Spoofed X-User-ID with no credentials stripped", func(t *testing.T) {
		handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Empty(t, r.Header.Get("X-User-ID"))
			require.Empty(t, r.Header.Get("X-Tenant-ID"))
			_, ok := security.AuthInfoFromContext(r.Context())
			require.False(t, ok)
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-User-ID", "attacker-uuid")
		req.Header.Set("X-Tenant-ID", "victim-tenant-uuid")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		require.Equal(t, http.StatusOK, rr.Code)
	})

	// 7. OPTIONS -> passes through, headers stripped
	t.Run("OPTIONS preflight pass-through", func(t *testing.T) {
		handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Empty(t, r.Header.Get("X-User-ID"))
			w.WriteHeader(http.StatusNoContent)
		}))

		req := httptest.NewRequest("OPTIONS", "/test", nil)
		req.Header.Set("X-User-ID", "some-uuid")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		require.Equal(t, http.StatusNoContent, rr.Code)
	})

	// 8. Negative checks: 1-dot garbage -> 401, 2-dot garbage -> 401
	t.Run("Structural malformed tokens rejected with 401", func(t *testing.T) {
		for _, malformed := range []string{"singleDot.garbage", "two.dot.garbage", "nodots"} {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", "Bearer "+malformed)
			rr := httptest.NewRecorder()
			mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})).ServeHTTP(rr, req)
			require.Equal(t, http.StatusUnauthorized, rr.Code, "expected 401 for token %q", malformed)
		}
	})
}

func osGetenv(k, def string) string {
	return def
}
