package services

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/hondyman/semlayer/backend/internal/analytics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecurityManager_Checklist(t *testing.T) {
	// Setup
	jwtSecret := []byte("test-secret-key")
	cacheMgr := analytics.NewCacheManager(analytics.RedisConfig{})
	metricsCollector := analytics.NewMetricsCollector(time.Hour)
	sm := NewSecurityManager(cacheMgr, metricsCollector, jwtSecret)

	// 1. Verify JWT Validation
	t.Run("JWT Validation", func(t *testing.T) {
		// Create valid token
		claims := jwt.MapClaims{
			"user_id": "user123",
			"exp":     time.Now().Add(time.Hour).Unix(),
		}
		token, err := sm.SignToken(claims)
		require.NoError(t, err)

		// Validate valid token
		parsedClaims, err := sm.ValidateToken(token)
		require.NoError(t, err)
		assert.Equal(t, "user123", parsedClaims.UserID)

		// Validate invalid token
		_, err = sm.ValidateToken("invalid-token")
		assert.Error(t, err)

		// Validate expired token
		expiredClaims := jwt.MapClaims{
			"user_id": "user123",
			"exp":     time.Now().Add(-time.Hour).Unix(),
		}
		expiredToken, _ := sm.SignToken(expiredClaims)
		_, err = sm.ValidateToken(expiredToken)
		assert.Error(t, err)
	})

	// 2. Verify API Key Validation
	t.Run("API Key Validation", func(t *testing.T) {
		userID := "api-user"
		tenantID := "test-tenant"
		permissions := []string{"read", "write"}
		apiKey := sm.GenerateAPIKey(userID, tenantID, permissions)
		require.NotEmpty(t, apiKey)

		// Validate valid key
		uid, valid := sm.ValidateAPIKey(apiKey)
		assert.True(t, valid)
		assert.Equal(t, userID, uid)

		// Validate invalid key
		_, valid = sm.ValidateAPIKey("invalid-key")
		assert.False(t, valid)
	})

	// 3. Verify Rate Limiting
	t.Run("Rate Limiting", func(t *testing.T) {
		// Create limiter with small window and limit
		limiter := NewRateLimiter(time.Second, 2, 100)
		userID := "rate-limit-user"

		// First 2 requests should pass
		assert.True(t, limiter.Allow(userID))
		assert.True(t, limiter.Allow(userID))

		// 3rd request should fail
		assert.False(t, limiter.Allow(userID))
	})

	// 4. Verify Security Middleware
	t.Run("Security Middleware", func(t *testing.T) {
		r := chi.NewRouter()
		r.Use(sm.SecurityMiddleware())
		r.Get("/secure", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		// Request without auth
		req := httptest.NewRequest("GET", "/secure", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)

		// Request with valid JWT
		claims := jwt.MapClaims{"user_id": "middleware-user"}
		token, _ := sm.SignToken(claims)
		req = httptest.NewRequest("GET", "/secure", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
