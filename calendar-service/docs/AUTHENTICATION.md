# Calendar Service Authentication & Security

This document outlines the authentication and security patterns used in the Calendar Service, aligned with the platform-wide JWT session security standards.

## Overview

The Calendar Service uses JWT (JSON Web Token) based authentication with the following features:

- **JWT Bearer Tokens** - Standard `Authorization: Bearer <token>` format
- **Multi-tenancy Support** - Tenant isolation via JWT claims and headers
- **Role-Based Access Control (RBAC)** - User roles define permission scopes
- **Token Claims** - Rich claims including user info, tenant, roles, and permissions
- **Hasura Integration** - JWT claims include Hasura-specific claims for RLS (Row Level Security)

## JWT Token Structure

### Standard Claims

All JWT tokens include the following claims aligned with the platform:

```json
{
  "user_id": "uuid-string",          // Unique user identifier (required)
  "email": "user@example.com",       // User email address
  "name": "User Name",               // Full user name
  "role": "admin",                   // Primary role
  "roles": ["admin", "user"],        // List of all user roles
  "tenant_id": "tenant-uuid",        // Primary tenant ID
  "tenant_ids": ["t1", "t2"],        // All tenant IDs user has access to
  "organization": "org-name",        // Organization name
  "permissions": ["read:calendar"],  // User permissions
  "is_core_admin": false,            // Core platform admin flag
  "jti": "jwt-unique-id",            // JWT ID for revocation tracking
  "iat": 1676000000,                 // Issued at (Unix timestamp)
  "exp": 1676003600,                 // Expiration (Unix timestamp)
  "https://hasura.io/jwt/claims": {  // Hasura-specific claims
    "x-hasura-allowed-roles": ["admin", "user"],
    "x-hasura-default-role": "user",
    "x-hasura-user-id": "uuid-string",
    "x-hasura-tenant-id": "tenant-uuid"
  }
}
```

## Authentication Flow

### 1. Token Acquisition

Users obtain JWT tokens through the Auth Service via login endpoint:

```bash
curl -X POST https://api.example.com/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password"
  }'

# Response
{
  "access_token": "eyJhbGc...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "user": { /* user info */ }
}
```

### 2. Token Transmission

Include token in Authorization header for all requests:

```bash
curl -X GET https://api.example.com/api/v1/calendars \
  -H "Authorization: Bearer eyJhbGc..." \
  -H "X-Tenant-ID: tenant-uuid" \
  -H "X-Tenant-Region: us-east-1"
```

### 3. Token Validation

The Calendar Service validates tokens using JWTMiddleware:

1. Extract Bearer token from Authorization header
2. Parse JWT with HS256 signature verification
3. Validate token expiration
4. Extract and validate required claims (user_id, tenant_id)
5. Store claims in request context
6. Enforce tenant access control

## Middleware Stack

### JWTMiddleware

Validates JWT tokens and extracts claims:

```go
// Applied to all protected routes
middleware.JWTMiddleware(jwtSecret, logger)
```

**Behavior:**
- Extracts and validates Bearer token
- Validates signature using JWT_SECRET
- Validates token expiration
- Requires user_id and tenant claims
- Supports development mode unauth with X-User-ID header
- Adds claims to request context

**Error Cases:**
- Missing Authorization header → 401 Unauthorized
- Invalid format (not Bearer token) → 401 Unauthorized
- Invalid signature → 401 Unauthorized
- Expired token → 401 Unauthorized
- Missing required claims → 401 Unauthorized

### TenantGuardMiddleware

Enforces tenant isolation:

```go
// Applied to all API routes
middleware.TenantGuardMiddleware(logger)
```

**Behavior:**
- Validates user specified X-Tenant-ID matches JWT claims
- Ensures user only accesses their authorized tenants
- Falls back to JWT tenant if header not provided
- Prevents cross-tenant data access

**Error Cases:**
- Missing tenant identification → 403 Forbidden
- User not authorized for tenant → 403 Forbidden

## Context Keys

JWT claims are added to request context with standardized keys:

```go
// In request handlers
import "calendar-service/internal/middleware"

userID := middleware.ExtractUserIDFromContext(ctx)
tenantID := middleware.ExtractTenantIDFromContext(ctx)
roles := middleware.ExtractRolesFromContext(ctx)

// Check for specific role
if middleware.HasRole(ctx, "admin") {
  // Admin-only logic
}
```

## Security Configuration

### Environment Variables

```bash
# JWT secret for token validation (required for production)
JWT_SECRET="your-secure-random-secret-key-min-32-chars"

# Development mode - allow unauth requests with X-User-ID header
DEV_ALLOW_UNAUTH_XUSER="false"  # Set to "true" for local development only
```

### Best Practices

1. **Secret Management**
   - Use strong, random JWT_SECRET (min 32 characters)
   - Store secret securely (never commit to git)
   - Rotate secrets periodically
   - Different secrets for dev/staging/prod

2. **Token Lifecycle**
   - Short-lived access tokens (15-60 minutes)
   - Use refresh tokens for long-lived sessions
   - Implement token revocation via JTI tracking
   - Clear expired tokens from cache

3. **Transport Security**
   - Always use HTTPS in production
   - Enable Strict-Transport-Security header
   - Validate certificate chains

4. **Claims Validation**
   - Validate user_id and tenant_id presence
   - Validate tenant access before processing
   - Log security events for audit

## Tenant Isolation

Calendar Service enforces strict tenant isolation:

1. **At Authentication**
   - JWT claims include tenant scope
   - Cannot request tenant outside JWT claims

2. **At Request**
   - X-Tenant-ID header required
   - Validated against JWT tenant claims
   - Request fails if tenant mismatch

3. **At Data Layer**
   - All queries filtered by tenant_id
   - Repository layer enforces TenantID filtering
   - Hasura RLS enforces server-side filtering

## Integration with Auth Service

Calendar Service validates tokens issued by the Auth Service:

```bash
# Auth Service generates token
POST /auth/login
Response: { "access_token": "eyJhbGc..." }

# Calendar Service validates token
GET /api/v1/calendars
Header: "Authorization: Bearer eyJhbGc..."
→ JWTMiddleware validates signature with JWT_SECRET
→ Token valid if signed by Auth Service with same secret
```

## API Key Support (Future)

Similar to other platform services, Calendar Service can support API key authentication:

```bash
curl -X GET https://api.example.com/api/v1/calendars \
  -H "X-API-Key: api-key-value"
```

Implementation would:
1. Extract X-API-Key or Authorization: ApiKey prefix
2. Validate against API key store (Redis/database)
3. Map API key to user and tenant
4. Follow same authorization flows

## Error Handling

### 401 Unauthorized
Returned when authentication fails:
- Missing Authorization header
- Invalid token format
- Invalid signature
- Expired token
- Missing required claims

Response:
```json
{
  "error": "Unauthorized",
  "message": "Invalid token"
}
```

### 403 Forbidden
Returned when authorization fails:
- User not authorized for requested tenant
- User lacks required role
- User lacks required permission

Response:
```json
{
  "error": "Forbidden",
  "message": "Access denied"
}
```

## Audit Logging

All authentication events should be logged for security audit:

```go
logger.WithFields(logrus.Fields{
  "user_id":        userID,
  "tenant_id":      tenantID,
  "action":         "availability_check",
  "result":         "success",
  "timestamp":      time.Now().Unix(),
}).Info("Calendar action completed")
```

## Compliance

The Calendar Service authentication aligns with:

- **OWASP Top 10** - Secure authentication practices
- **NIST Standards** - Token-based authentication
- **SOC 2** - Audit logging and tenant isolation
- **Data Residency** - Region-based tenant constraints

## Testing

### Development

For local development without full Auth Service:

```bash
# Set development mode
export DEV_ALLOW_UNAUTH_XUSER=true

# Make requests with X-User-ID header (JWT validation skipped)
curl -X GET http://localhost:8080/api/v1/calendars \
  -H "X-User-ID: test-user" \
  -H "X-Tenant-ID: test-tenant"
```

### Integration Testing

Use valid JWT tokens:

```go
// Create test token with claims
token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
  "user_id":    "test-user",
  "tenant_id":  "test-tenant",
  "roles":      []string{"admin"},
  "exp":        time.Now().Add(time.Hour).Unix(),
})

tokenString, _ := token.SignedString([]byte(jwtSecret))

// Use in requests
req.Header.Set("Authorization", "Bearer "+tokenString)
```

## References

- [JWT.io](https://jwt.io) - JWT standard
- [Backend Auth Implementation](../backend/internal/api/auth_handlers.go)
- [API Gateway Security](../api-gateway/middleware/tenant_guard.go)
- [Auth Service](../auth-service/)
