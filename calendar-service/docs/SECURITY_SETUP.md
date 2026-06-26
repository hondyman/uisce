# Security Implementation Checklist & Integration Guide

## Overview

This guide documents the JWT authentication and security implementation for the Calendar Service, aligned with platform-wide security standards.

## Security Architecture

### Components

1. **JWTMiddleware** (`internal/middleware/jwt_auth.go`)
   - Validates Bearer tokens
   - Extracts claims into context
   - Enforces token expiration
   - Validates required claims

2. **TenantGuardMiddleware** (`internal/middleware/jwt_auth.go`)
   - Enforces tenant isolation
   - Validates tenant access
   - Prevents cross-tenant data access

3. **SecurityManager** (`internal/security/manager.go`)
   - Token validation and parsing
   - Role-based access checks
   - Permission validation
   - Tenant access verification

## Integration Checklist

### ✅ Phase 1: Authentication Infrastructure (COMPLETED)

- [x] Create JWT middleware with Bearer token support
- [x] Implement tenant guard middleware
- [x] Create SecurityManager for token operations
- [x] Add claims extraction utilities
- [x] Support multi-tenancy validation
- [x] Add context propagation helpers

### ⏳ Phase 2: API Integration (NEXT)

- [ ] Update all handlers to extract user/tenant from context
- [ ] Add audit logging to all endpoints
- [ ] Implement role-based access control in handlers
- [ ] Add request validation with tenant context
- [ ] Create integration tests with JWT tokens

### ⏳ Phase 3: Advanced Security

- [ ] Implement token revocation via JTI tracking
- [ ] Add rate limiting per tenant
- [ ] Implement request signing for webhooks
- [ ] Add security headers (CSP, X-Content-Type-Options, etc.)
- [ ] Implement request logging and audit trail

### ⏳ Phase 4: Monitoring & Compliance

- [ ] Add security event logging
- [ ] Implement metrics for failed authentications
- [ ] Add alerting for suspicious patterns
- [ ] Create security audit reports
- [ ] Document compliance mappings

## Configuration

### Environment Variables

```bash
# REQUIRED - JWT secret for token validation
# Must be at least 32 characters
# Use strong random string: $(openssl rand -hex 32)
JWT_SECRET="your-secure-random-secret-key-min-32-chars"

# OPTIONAL - Development mode
# Set to "true" ONLY for local development
# NEVER set to "true" in production
DEV_ALLOW_UNAUTH_XUSER="false"

# OPTIONAL - Server configuration  
PORT="8080"
LOG_LEVEL="info"
```

### Docker Compose Example

```yaml
version: '3.8'
services:
  calendar-service:
    build: .
    ports:
      - "8080:8080"
    environment:
      JWT_SECRET: ${JWT_SECRET}
      DEV_ALLOW_UNAUTH_XUSER: "false"
      PORT: "8080"
      LOG_LEVEL: "info"
    depends_on:
      - postgres
      - redis
```

## Handler Implementation Pattern

All handlers should follow this pattern to use authenticated context:

```go
package api

import (
	"calendar-service/internal/middleware"
)

func (h *YourHandler) YourEndpoint(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract authentication context
	userID := middleware.ExtractUserIDFromContext(ctx)
	tenantID := middleware.ExtractTenantIDFromContext(ctx)
	roles := middleware.ExtractRolesFromContext(ctx)

	// Validate context
	if userID == "" || tenantID == "" {
		http.Error(w, "Authentication context missing", http.StatusUnauthorized)
		return
	}

	// Check for admin role if needed
	if middleware.HasRole(ctx, "admin") {
		// Admin-specific logic
	}

	// Use tenant context for data operations
	// Pass to service layer
	result, err := h.service.DoSomething(ctx, userID, tenantID)
	if err != nil {
		// Handle error
		return
	}

	// Return result
	// ...
}
```

## Testing JWT Tokens

### Generate Test Token

```go
package api_test

import (
	"time"
	"github.com/golang-jwt/jwt/v5"
)

func generateTestToken(userID, tenantID, secret string) string {
	claims := jwt.MapClaims{
		"user_id":   userID,
		"email":     "test@example.com",
		"tenant_id": tenantID,
		"roles":     []string{"user", "admin"},
		"permissions": []string{"read:calendar"},
		"iat":       time.Now().Unix(),
		"exp":       time.Now().Add(time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(secret))
	return tokenString
}
```

### Integration Test Example

```go
func TestCalendarListWithAuth(t *testing.T) {
	// Setup
	jwtSecret := "test-secret-key-min-32-characters"
	tokenString := generateTestToken("user-123", "tenant-456", jwtSecret)

	// Create request
	req, _ := http.NewRequest("GET", "/api/v1/calendars", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	req.Header.Set("X-Tenant-ID", "tenant-456")

	// Execute
	recorder := httptest.NewRecorder()
	// router.ServeHTTP(recorder, req)

	// Assert
	if recorder.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", recorder.Code)
	}
}
```

## Deployment Checklist

### Pre-Deployment

- [ ] JWT_SECRET configured (strong, random, min 32 chars)
- [ ] DEV_ALLOW_UNAUTH_XUSER set to "false"
- [ ] HTTPS enabled in production
- [ ] All environment variables set
- [ ] Security headers configured
- [ ] Database migrations applied
- [ ] Redis cache configured (if using revocation)

### Deployment

- [ ] Verify JWT_SECRET is not logged
- [ ] Verify service starts without errors
- [ ] Verify health endpoint accessible at `/api/v1/health`
- [ ] Test with valid JWT token
- [ ] Test with invalid token (should get 401)
- [ ] Test with missing token (should get 401)
- [ ] Test cross-tenant access (should get 403)

### Post-Deployment

- [ ] Monitor authentication errors in logs
- [ ] Monitor request latency (JWT validation overhead should be <1ms)
- [ ] Monitor failed auth attempts for suspicious patterns
- [ ] Verify tokens are rotating properly
- [ ] Check token expiration handling

## Troubleshooting

### Missing Authorization Header

**Error:** `Authorization header required`

**Solution:**
1. Verify request includes `Authorization: Bearer <token>` header
2. Check token is not expired
3. Verify JWT_SECRET matches token issuer

### Invalid Token

**Error:** `Invalid token` or `invalid token`

**Solution:**
1. Verify token format: `eyJhbGc...` (should have 3 parts separated by dots)
2. Verify JWT_SECRET matches Auth Service secret
3. Verify token not expired: `exp` claim should be future timestamp
4. Check token signature with jwt.io

### Missing Tenant

**Error:** `Token missing tenant claims` or `X-Tenant-ID header required`

**Solution:**
1. Verify Auth Service issues token with `tenant_id` or `tenant_ids` claim
2. Verify request includes `X-Tenant-ID` header
3. Check tenant_id is valid UUID format

### Access Denied

**Error:** `Forbidden: Access denied for requested tenant`

**Solution:**
1. Verify X-Tenant-ID header matches one of token tenant_ids
2. Check user has access to requested tenant
3. Verify tenant isolation in database (soft deletes, etc.)

## Security Best Practices

### 1. Secret Management
```bash
# Generate strong JWT secret
openssl rand -hex 32  # Creates 64-char hex string

# Store in vault/secret manager, never in code
# Rotate periodically (at least quarterly)
```

### 2. Token Validation
```go
// Always validate expiration
if claims.ExpiresAt.Before(time.Now()) {
    return fmt.Errorf("token expired")
}

// Always validate user_id and tenant_id presence
if claims.UserID == "" || claims.TenantID == "" {
    return fmt.Errorf("missing required claims")
}
```

### 3. Context Propagation
```go
// Always pass context through service layers
func (s *Service) DoSomething(ctx context.Context, ...) error {
    // Extract context values
    userID := middleware.ExtractUserIDFromContext(ctx)
    
    // Pass to database/repository
    return s.repo.Query(ctx, userID, ...)
}
```

### 4. Audit Logging
```go
// Log all authentication events
logger.WithFields(logrus.Fields{
    "event":      "auth_check",
    "user_id":    userID,
    "tenant_id":  tenantID,
    "action":     "availability_check",
    "result":     "success",
    "ip":         remoteIP,
}).Info("Authentication event")
```

### 5. Error Messages
```go
// Never expose internal details in errors
http.Error(w, "Unauthorized", http.StatusUnauthorized)  // ✓ Good
http.Error(w, "Invalid JWT signature", 401)             // ✗ Bad
http.Error(w, "User not found in token", 401)           // ✗ Bad
```

## References

- **JWT Standard:** https://tools.ietf.org/html/rfc7519
- **Backend Auth:** `/backend/internal/api/auth_handlers.go`
- **API Gateway:** `/api-gateway/middleware/tenant_guard.go`
- **Auth Service:** `/auth-service/`
- **JWT.io:** https://jwt.io
- **OWASP JWT Cheat Sheet:** https://cheatsheetseries.owasp.org/cheatsheets/JSON_Web_Token_for_Java_Cheat_Sheet.html

## Support

For authentication issues or questions:
1. Check AUTHENTICATION.md for detailed flows
2. Review handler implementation pattern above
3. Check logs for JWT validation errors
4. Verify JWT_SECRET configuration
5. Test with jwt.io token decoder
