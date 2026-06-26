# JWT Middleware Library

Shared JWT authentication and authorization middleware for all SemLayer services.

## Features

- ✅ JWT token validation using HS256
- ✅ Multi-tenant support with tenant isolation
- ✅ Role-based access control (RBAC)
- ✅ User context extraction and propagation
- ✅ Standard JWT claims across all services
- ✅ HTTP middleware for net/http and Chi routers
- ✅ Configurable skip paths for public endpoints

## Installation

All services should import this library:

```go
import "github.com/hondyman/semlayer/libs/jwt-middleware"
```

## Configuration

All services must set the `JWT_SECRET` environment variable:

```bash
export JWT_SECRET="dev-jwt-secret-key-change-in-production"
```

**IMPORTANT**: The JWT_SECRET must be the same across all services for token validation to work.

## Usage

### Basic JWT Validation

```go
package main

import (
    "net/http"
    "github.com/hondyman/semlayer/libs/jwt-middleware"
)

func getUserHandler(w http.ResponseWriter, r *http.Request) {
    // Get claims from context
    claims := jwtmiddleware.GetClaimsFromContext(r)
    if claims == nil {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }
    
    userID := claims.UserID
    tenantID := claims.TenantID
    
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    fmt.Fprintf(w, `{"user_id":"%s","tenant_id":"%s"}`, userID, tenantID)
}

func main() {
    router := http.NewServeMux()
    
    // Apply JWT middleware
    jwtMiddleware := jwtmiddleware.NewJWTMiddleware(
        "/health",
        "/api/auth/login",
    )
    
    // Wrap routes
    router.Handle("/api/user", jwtMiddleware.Handler(http.HandlerFunc(getUserHandler)))
    
    http.ListenAndServe(":8080", router)
}
```

### With Chi Router

```go
import "github.com/go-chi/chi/v5"

func main() {
    router := chi.NewRouter()
    
    // Use JWT middleware for all routes
    router.Use(jwtmiddleware.ChiMiddleware())
    
    router.Get("/api/user", getUserHandler)
    
    http.ListenAndServe(":8080", router)
}
```

### Tenant Validation

```go
func protectedHandler(w http.ResponseWriter, r *http.Request) {
    claims := jwtmiddleware.GetClaimsFromContext(r)
    tenantID := r.Header.Get("X-Tenant-ID")
    
    // Validate tenant access
    if err := jwtmiddleware.ValidateTenantAccess(claims, tenantID); err != nil {
        http.Error(w, err.Error(), http.StatusForbidden)
        return
    }
    
    // Continue with handler logic
    w.WriteHeader(http.StatusOK)
}
```

### Role-Based Access Control

```go
func adminHandler(w http.ResponseWriter, r *http.Request) {
    claims := jwtmiddleware.GetClaimsFromContext(r)
    
    // Check for specific role
    if !jwtmiddleware.HasRole(claims, "admin") {
        http.Error(w, "forbidden", http.StatusForbidden)
        return
    }
    
    // Continue with admin logic
    w.WriteHeader(http.StatusOK)
}

// Or use middleware wrapper
router.Handle("/admin", 
    jwtmiddleware.RequireRole("admin", http.HandlerFunc(adminHandler)))
```

### Optional JWT (for public endpoints with optional auth)

```go
func publicHandler(w http.ResponseWriter, r *http.Request) {
    // Try to get claims if provided
    claims := jwtmiddleware.GetClaimsFromContext(r)
    
    if claims != nil {
        // User is authenticated
        fmt.Fprintf(w, "Hello, %s", claims.UserID)
    } else {
        // User is not authenticated (but that's ok)
        fmt.Fprintf(w, "Hello, guest")
    }
}

func main() {
    router := http.NewServeMux()
    
    optionalJWT := jwtmiddleware.NewOptionalJWTMiddleware()
    
    router.Handle("/public", optionalJWT.Handler(http.HandlerFunc(publicHandler)))
    
    http.ListenAndServe(":8080", router)
}
```

## JWT Claims Structure

```json
{
  "user_id": "36f45238-bac6-4b06-a495-6155c43df552",
  "email": "user@example.com",
  "tenant_id": "910638ba-a459-4a3f-bb2d-78391b0595f6",
  "tenant_ids": ["910638ba-a459-4a3f-bb2d-78391b0595f6"],
  "roles": ["admin", "analyst"],
  "is_active": true,
  "is_core_admin": false,
  "organization_id": "org-123",
  "exp": 1771827165,
  "iat": 1771823565,
  "jti": "...",
  "aud": ["https://semlayer.local"],
  "iss": "https://semlayer.local/auth",
  "sub": "36f45238-bac6-4b06-a495-6155c43df552"
}
```

## Services Using JWT Middleware

### Core Services (Required)

- ✅ **API Gateway** - Routes all requests, validates and forwards JWTs
- ✅ **Auth Service** - Issues JWT tokens
- ✅ **Backend** - Validates JWTs for all API endpoints
- ✅ **BP Backend** - Validates JWTs for all endpoints
- ✅ **Entity Manager** - Validates JWTs for data access
- ✅ **Analytics Engine** - Validates JWTs for analytics queries
- ✅ **Compliance Engine** - Validates JWTs for compliance operations
- ✅ **Validation Engine** - Validates JWTs for validation operations
- ✅ **Notifications Service** - Validates JWTs for notification access

### Environment Variables (All Services)

Every service must set these variables:

```bash
# Common
JWT_SECRET=dev-jwt-secret-key-change-in-production
PORT=8080

# Optional: Enable strict JWT validation
JWT_ENFORCE=true
JWT_VALIDATE_TENANT=true
```

## Security Best Practices

1. **Never** hardcode JWT secrets in code
2. **Always** use environment variables for JWT_SECRET
3. **Use different secrets** for development vs production
4. **Rotate secrets** regularly
5. **Validate tenant access** for all multi-tenant endpoints
6. **Use HTTPS** in production (enforce in API Gateway)
7. **Set short expiration times** on JWTs (default 1 hour)
8. **Implement token revocation** for logout functionality
9. **Log authentication failures** for security monitoring
10. **Use strong signing algorithms** (HS256 or RS256 only)

## Troubleshooting

### "JWT_SECRET not configured"

Set the JWT_SECRET environment variable in your service:

```bash
export JWT_SECRET="your-secret-key"
```

### "JWT validation failed: invalid token"

Common causes:
- Token was modified after signing
- Token was signed with a different secret
- Token has expired

### "user does not have access to tenant"

The user's tenant_id in the JWT doesn't match the requested tenant. Verify:
- User has access to the requested tenant
- X-Tenant-ID header matches user's tenant
- User is not a cross-tenant user

## Deployment Checklist

- [ ] All services set JWT_SECRET environment variable
- [ ] JWT_SECRET is the same across all services
- [ ] API Gateway forwards Authorization header to backend services
- [ ] Auth Service issues JWTs with correct claims
- [ ] All protected endpoints validate JWT tokens
- [ ] Tenant isolation is enforced
- [ ] Role-based access control is configured
- [ ] Error handling for invalid/expired tokens
- [ ] Logging configured for security events
- [ ] JWT secret rotation policy established

## References

- [JWT RFC 7519](https://tools.ietf.org/html/rfc7519)
- [golang-jwt library](https://pkg.go.dev/github.com/golang-jwt/jwt/v5)
- [Multi-tenant Architecture](../../docs/MULTI_TENANT_ARCHITECTURE.md)
- [Security Hardening Guide](../../docs/SECURITY_HARDENING.md)
