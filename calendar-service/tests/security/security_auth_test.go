//go:build security

package security

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"calendar-service/internal/middleware"
	"calendar-service/internal/services"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// Helper function to create test JWT tokens
func createTestJWT(userID, tenantID, secret string, isValid bool) string {
	expTime := time.Now().Add(time.Hour)
	if !isValid {
		expTime = time.Now().Add(-time.Hour)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":   userID,
		"tenant_id": tenantID,
		"email":     "user@example.com",
		"roles":     []string{"user"},
		"exp":       expTime.Unix(),
	})

	s, _ := token.SignedString([]byte(secret))
	return s
}

// TestJWTMiddlewareSecurity validates JWT middleware security
func TestJWTMiddlewareSecurity(t *testing.T) {
	jwtSecret := "test-secret-key"
	logger := logrus.NewEntry(logrus.New())

	tests := []struct {
		name           string
		setupRequest   func(*http.Request)
		description    string
		expectedStatus int
	}{
		{
			name: "valid token",
			setupRequest: func(req *http.Request) {
				token := createTestJWT("test-user", "test-tenant", jwtSecret, true)
				req.Header.Set("Authorization", "Bearer "+token)
			},
			description:    "Valid token should be accepted",
			expectedStatus: http.StatusOK,
		},
		{
			name: "expired token",
			setupRequest: func(req *http.Request) {
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
					"user_id":   "test-user",
					"tenant_id": "test-tenant",
					"exp":       time.Now().Add(-time.Hour).Unix(),
				})
				s, _ := token.SignedString([]byte(jwtSecret))
				req.Header.Set("Authorization", "Bearer "+s)
			},
			description:    "Expired token should be rejected",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "invalid signature",
			setupRequest: func(req *http.Request) {
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
					"user_id":   "test-user",
					"tenant_id": "test-tenant",
					"exp":       time.Now().Add(time.Hour).Unix(),
				})
				s, _ := token.SignedString([]byte("wrong-secret"))
				req.Header.Set("Authorization", "Bearer "+s)
			},
			description:    "Invalid signature should be rejected",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "missing user_id claim",
			setupRequest: func(req *http.Request) {
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
					"tenant_id": "test-tenant",
					"exp":       time.Now().Add(time.Hour).Unix(),
				})
				s, _ := token.SignedString([]byte(jwtSecret))
				req.Header.Set("Authorization", "Bearer "+s)
			},
			description:    "Missing user_id claim should be rejected",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "missing tenant_id claim",
			setupRequest: func(req *http.Request) {
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
					"user_id": "test-user",
					"exp":     time.Now().Add(time.Hour).Unix(),
				})
				s, _ := token.SignedString([]byte(jwtSecret))
				req.Header.Set("Authorization", "Bearer "+s)
			},
			description:    "Missing tenant_id claim should be rejected",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "missing authorization header",
			setupRequest: func(req *http.Request) {
				// Don't set Authorization header
			},
			description:    "Missing Authorization header should be rejected",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "malformed bearer token",
			setupRequest: func(req *http.Request) {
				req.Header.Set("Authorization", "NotBearer "+createTestJWT("user", "tenant", jwtSecret, true))
			},
			description:    "Malformed Bearer token should be rejected",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			tt.setupRequest(req)

			handler := middleware.JWTMiddleware(jwtSecret, logger)(http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
				}))

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code, tt.description)

			if tt.expectedStatus == http.StatusUnauthorized {
				assert.Contains(t, rr.Body.String(), "error")
			}
		})
	}
}

// TestTenantGuardMiddlewareSecurity validates cross-tenant access prevention
func TestTenantGuardMiddlewareSecurity(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())

	tests := []struct {
		name           string
		contextClaims  map[string]interface{}
		headerTenant   string
		expectedStatus int
		description    string
	}{
		{
			name:           "matching tenant",
			contextClaims:  map[string]interface{}{"tenant_id": "tenant-a"},
			headerTenant:   "tenant-a",
			expectedStatus: http.StatusOK,
			description:    "Request with matching tenant should be allowed",
		},
		{
			name:           "mismatched tenant",
			contextClaims:  map[string]interface{}{"tenant_id": "tenant-a"},
			headerTenant:   "tenant-b",
			expectedStatus: http.StatusForbidden,
			description:    "Request with mismatched tenant should be rejected",
		},
		{
			name:           "tenant from context",
			contextClaims:  map[string]interface{}{"tenant_id": "tenant-a"},
			headerTenant:   "",
			expectedStatus: http.StatusOK,
			description:    "Request without header should use context tenant",
		},
		{
			name:           "no tenant information",
			contextClaims:  map[string]interface{}{},
			headerTenant:   "",
			expectedStatus: http.StatusForbidden,
			description:    "Request without tenant info should be rejected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			ctx := middleware.WithClaims(req.Context(), tt.contextClaims)
			req = req.WithContext(ctx)

			if tt.headerTenant != "" {
				req.Header.Set("X-Tenant-ID", tt.headerTenant)
			}

			handler := middleware.TenantGuardMiddleware(logger)(http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
				}))

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code, tt.description)
		})
	}
}

// TestRateLimitingSecurity validates tenant-scoped rate limiting
func TestRateLimitingSecurity(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())

	t.Run("burst capacity", func(t *testing.T) {
		limiter := middleware.NewTenantRateLimiter(2, 2, logger) // 2 req/s, burst 2

		makeReq := func(tenantID string) *http.Request {
			req := httptest.NewRequest("GET", "/test", nil)
			ctx := middleware.WithClaims(req.Context(), map[string]interface{}{
				"tenant_id": tenantID,
			})
			return req.WithContext(ctx)
		}

		handler := limiter.RateLimit(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

		// First 2 requests should succeed (burst)
		for i := 0; i < 2; i++ {
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, makeReq("tenant-a"))
			assert.Equal(t, http.StatusOK, rr.Code, "Request %d should succeed", i+1)
		}

		// Third request should be rate limited
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, makeReq("tenant-a"))
		assert.Equal(t, http.StatusTooManyRequests, rr.Code, "Third request should be rate limited")
	})

	t.Run("per-tenant isolation", func(t *testing.T) {
		limiter := middleware.NewTenantRateLimiter(2, 2, logger)

		makeReq := func(tenantID string) *http.Request {
			req := httptest.NewRequest("GET", "/test", nil)
			ctx := middleware.WithClaims(req.Context(), map[string]interface{}{
				"tenant_id": tenantID,
			})
			return req.WithContext(ctx)
		}

		handler := limiter.RateLimit(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

		// First tenant uses burst
		for i := 0; i < 2; i++ {
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, makeReq("tenant-a"))
			assert.Equal(t, http.StatusOK, rr.Code)
		}

		// Different tenant should have separate limit
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, makeReq("tenant-b"))
		assert.Equal(t, http.StatusOK, rr.Code, "New tenant should not be rate limited")
	})

	t.Run("rate limit response format", func(t *testing.T) {
		limiter2 := middleware.NewTenantRateLimiter(1, 1, logger)

		makeReq := func(tenantID string) *http.Request {
			req := httptest.NewRequest("GET", "/test", nil)
			ctx := middleware.WithClaims(req.Context(), map[string]interface{}{
				"tenant_id": tenantID,
			})
			return req.WithContext(ctx)
		}

		handler2 := limiter2.RateLimit(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

		// First request uses burst
		rr1 := httptest.NewRecorder()
		handler2.ServeHTTP(rr1, makeReq("tenant-c"))
		assert.Equal(t, http.StatusOK, rr1.Code)

		// Second request hits limit
		rr2 := httptest.NewRecorder()
		handler2.ServeHTTP(rr2, makeReq("tenant-c"))
		assert.Equal(t, http.StatusTooManyRequests, rr2.Code)

		// Verify response format
		assert.Equal(t, "application/json", rr2.Header().Get("Content-Type"))
		assert.Equal(t, "60", rr2.Header().Get("Retry-After"))

		var response map[string]interface{}
		json.NewDecoder(rr2.Body).Decode(&response)
		assert.Equal(t, "rate_limit_exceeded", response["error"])
	})
}

// TestAuditLoggingCompleteness validates audit service integration
func TestAuditLoggingCompleteness(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	auditSvc := services.NewAuditService(logger)
	ctx := middleware.WithClaims(context.Background(), map[string]interface{}{
		"user_id":   "test-user",
		"tenant_id": "test-tenant",
	})

	t.Run("record create", func(t *testing.T) {
		err := auditSvc.RecordCreate(ctx, "test-tenant", "calendar", "cal-123",
			map[string]interface{}{
				"name":     "Test Calendar",
				"timezone": "UTC",
			}, "test-user")
		assert.NoError(t, err)
	})

	t.Run("record update", func(t *testing.T) {
		err := auditSvc.RecordUpdate(ctx, "test-tenant", "calendar", "cal-123",
			map[string]interface{}{"name": "Old Name"},
			map[string]interface{}{"name": "New Name"},
			"test-user")
		assert.NoError(t, err)
	})

	t.Run("record delete", func(t *testing.T) {
		err := auditSvc.RecordDelete(ctx, "test-tenant", "calendar", "cal-123",
			map[string]interface{}{
				"name":     "Test Calendar",
				"timezone": "UTC",
			}, "test-user")
		assert.NoError(t, err)
	})

	t.Run("missing tenant_id rejected", func(t *testing.T) {
		err := auditSvc.RecordCreate(ctx, "", "calendar", "cal-123",
			map[string]interface{}{"name": "Test"}, "test-user")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tenant_id required")
	})

	t.Run("missing entity_type rejected", func(t *testing.T) {
		err := auditSvc.RecordCreate(ctx, "test-tenant", "", "cal-123",
			map[string]interface{}{"name": "Test"}, "test-user")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "entity_type required")
	})

	t.Run("get audit log", func(t *testing.T) {
		auditSvc.RecordCreate(ctx, "test-tenant", "calendar", "cal-1",
			map[string]interface{}{"name": "Cal 1"}, "user-1")
		auditSvc.RecordCreate(ctx, "test-tenant", "calendar", "cal-2",
			map[string]interface{}{"name": "Cal 2"}, "user-2")

		// Fetch logs
		entries, err := auditSvc.GetAuditLog(ctx, "test-tenant", 10)
		assert.NoError(t, err)
		assert.Greater(t, len(entries), 0, "Should have audit entries")
	})

	t.Run("audit log cross-tenant rejection", func(t *testing.T) {
		diffCtx := middleware.WithClaims(context.Background(), map[string]interface{}{
			"user_id":   "other-user",
			"tenant_id": "other-tenant",
		})

		_, err := auditSvc.GetAuditLog(diffCtx, "test-tenant", 10)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})
}

// TestInputValidationSecurity validates that services reject malicious input
func TestInputValidationSecurity(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	auditSvc := services.NewAuditService(logger)
	ctx := middleware.WithClaims(context.Background(), map[string]interface{}{
		"user_id":   "test-user",
		"tenant_id": "test-tenant",
	})

	tests := []struct {
		name        string
		entityType  string
		entityID    string
		fieldValue  interface{}
		shouldError bool
	}{
		{
			name:        "valid input",
			entityType:  "calendar",
			entityID:    "uuid-1234-5678-9012",
			fieldValue:  "Valid Calendar Name",
			shouldError: false,
		},
		{
			name:        "sql injection attempt with empty entity_id",
			entityType:  "calendar",
			entityID:    "",
			fieldValue:  "test",
			shouldError: true, // Should validate empty entity_id
		},
		{
			name:        "xss attempt",
			entityType:  "calendar",
			entityID:    "uuid-1234",
			fieldValue:  "<script>alert('xss')</script>",
			shouldError: false, // Audit service stores as-is, but handlers should validate
		},
		{
			name:        "empty entity_id",
			entityType:  "calendar",
			entityID:    "",
			fieldValue:  "test",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := auditSvc.RecordCreate(ctx, "test-tenant", tt.entityType, tt.entityID,
				map[string]interface{}{"name": tt.fieldValue}, "test-user")

			if tt.shouldError {
				assert.Error(t, err, "Should reject invalid input")
			} else {
				assert.NoError(t, err, "Should accept valid input")
			}
		})
	}
}

// TestContextPropagation validates that JWT claims are properly propagated
func TestContextPropagation(t *testing.T) {
	t.Run("extract user_id", func(t *testing.T) {
		ctx := middleware.WithClaims(context.Background(), map[string]interface{}{
			"user_id":   "user-123",
			"tenant_id": "tenant-456",
			"email":     "user@example.com",
			"roles":     []string{"admin", "user"},
		})

		userID := middleware.ExtractUserIDFromContext(ctx)
		assert.Equal(t, "user-123", userID)
	})

	t.Run("extract tenant_id", func(t *testing.T) {
		ctx := middleware.WithClaims(context.Background(), map[string]interface{}{
			"user_id":   "user-123",
			"tenant_id": "tenant-456",
			"email":     "user@example.com",
			"roles":     []string{"admin", "user"},
		})

		tenantID := middleware.ExtractTenantIDFromContext(ctx)
		assert.Equal(t, "tenant-456", tenantID)
	})

	t.Run("extract roles", func(t *testing.T) {
		ctx := middleware.WithClaims(context.Background(), map[string]interface{}{
			"user_id":   "user-123",
			"tenant_id": "tenant-456",
			"email":     "user@example.com",
			"roles":     []string{"admin", "user"},
		})

		roles := middleware.ExtractRolesFromContext(ctx)
		assert.ElementsMatch(t, []string{"admin", "user"}, roles)
	})

	t.Run("has role", func(t *testing.T) {
		ctx := middleware.WithClaims(context.Background(), map[string]interface{}{
			"user_id":   "user-123",
			"tenant_id": "tenant-456",
			"email":     "user@example.com",
			"roles":     []string{"admin", "user"},
		})

		assert.True(t, middleware.HasRole(ctx, "admin"))
		assert.True(t, middleware.HasRole(ctx, "user"))
		assert.False(t, middleware.HasRole(ctx, "superadmin"))
	})

	t.Run("missing context values", func(t *testing.T) {
		emptyCtx := context.Background()

		assert.Empty(t, middleware.ExtractUserIDFromContext(emptyCtx))
		assert.Empty(t, middleware.ExtractTenantIDFromContext(emptyCtx))
		assert.Empty(t, middleware.ExtractRolesFromContext(emptyCtx))
	})
}
