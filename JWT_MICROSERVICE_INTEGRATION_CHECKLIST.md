# JWT Middleware Integration Checklist

## Overview
All microservices must import and use the `libs/jwt-middleware` library to validate JWT tokens and enforce tenant isolation.

## ✅ Completed

### API Gateway
- [x] JWT token validation implemented
- [x] Authorization header forwarding to Hasura
- [x] Tenant header forwarding to backend services
- [x] Service-to-service JWT support

### Auth Service
- [x] JWT token issuance
- [x] Token validation
- [x] Refresh token handling

### Backend Service
- [x] JWT middleware implemented
- [x] Tenant isolation enforced
- [x] Role-based access control

## 🔄 In Progress / Not Started

### Phase 1: Core Engine Services

#### Entity Manager
**Status**: 🔄 Next
**Dockerfile**: `entity-manager/Dockerfile`
**Main Entry Point**: `entity-manager/main.go` or similar
**Required Changes**:
```go
import "github.com/hondyman/semlayer/libs/jwt-middleware"

// In main():
publicPaths := []string{
    "/health",
    "/docs",
    "/api/auth/login",
}
jwtMiddleware := jwtmiddleware.NewJWTMiddleware(publicPaths...)

// Add to router before routes
router.Use(jwtMiddleware.Handler)

// In handlers:
claims := jwtmiddleware.GetClaimsFromContext(r)
if claims == nil {
    http.Error(w, "unauthorized", http.StatusUnauthorized)
    return
}
// Enforce tenant isolation
db.QueryContext(r.Context(), "SELECT * FROM entities WHERE tenant_id = ?", claims.TenantID)
```

#### Validation Engine
**Status**: 🔄 Next
**Dockerfile**: `backend/Dockerfile.validation`
**Main Entry Point**: `validation-engine/main.go` or similar
**Required Changes**: Same as Entity Manager
**Additional Notes**: May have custom validation rules that need tenant scope

#### Rule Engine
**Status**: 🔄 Next
**Dockerfile**: `backend/Dockerfile.rule-engine`
**Main Entry Point**: `rule-engine/main.go` or similar
**Required Changes**: Same as Entity Manager
**Additional Notes**: Rules must be scoped by tenant_id

### Phase 2: Service engines

#### Search Service
**Status**: 🔄 Next
**Dockerfile**: `backend/Dockerfile.search`
**Main Entry Point**: Search service main
**Required Changes**: Same as Entity Manager
**Additional Notes**: Search index queries must filter by tenant_id

#### Policy Engine
**Status**: 🔄 Next
**Dockerfile**: `backend/Dockerfile.policy`
**Main Entry Point**: Policy engine main
**Required Changes**: Same as Entity Manager
**Additional Notes**: Policies must be tenant-scoped

#### Analytics Engine
**Status**: ✅ Configured in docker-compose, code needs update
**Dockerfile**: `analytics_engine/Dockerfile`
**Main Entry Point**: `analytics_engine/main.go`
**Required Changes**: Add JWT middleware import and setup

#### Compliance Engine
**Status**: ✅ Configured in docker-compose, code needs update
**Dockerfile**: `Dockerfile.compliance-engine`
**Main Entry Point**: Compliance service main
**Required Changes**: Add JWT middleware import and setup

### Phase 3: Background Workers & Processors

#### Sync Worker
**Status**: 🔄 Next
**Dockerfile**: `backend/Dockerfile.sync-worker`
**Notes**: Background worker - may not have HTTP endpoints  
**Required Changes**: 
- If this has HTTP endpoints, add JWT middleware
- If calling other services, add JWT signing
- Verify timing of sync operations respects tenant isolation

#### Event Router
**Status**: ✅ Configured in docker-compose, code needs update
**Dockerfile**: `backend/Dockerfile.event-router`
**Required Changes**: Add JWT middleware, event routing must respect tenant_id

#### CDC Processor
**Status**: 🔄 Next
**Dockerfile**: `backend/Dockerfile.cdc`
**Notes**: Change Data Capture processor - may not expose HTTP
**Required Changes**:
- If has management endpoints, add JWT middleware
- Ensure CDC respects tenant_id in changelog

#### Outbox Processor
**Status**: 🔄 Next
**Dockerfile**: `backend/Dockerfile.outbox-processor`
**Notes**: Outbox pattern processor - background worker
**Required Changes**:
- If event dispatching to HTTP endpoints, add Auth headers
- Otherwise may not need endpoint-level JWT

#### Parity Checker
**Status**: 🔄 Next
**Dockerfile**: `backend/Dockerfile.parity`
**Notes**: Health check / parity verification worker
**Required Changes**: May not need endpoint security if no HTTP exposure

#### Catalog Sync
**Status**: 🔄 Next
**Dockerfile**: `backend/Dockerfile.catalog-sync`
**Required Changes**: Add JWT middleware for catalog operations

---

## 📚 Reference Implementation

### Backend Service Pattern (already implemented)
```go
package main

import (
    "net/http"
    "github.com/go-chi/chi/v5"
    "github.com/hondyman/semlayer/libs/jwt-middleware"
)

func main() {
    // Define public routes that don't require JWT
    publicPaths := []string{
        "/health",
        "/docs",
        "/api/auth/login",
        "/api/auth/refresh",
    }
    
    // Initialize JWT middleware
    jwtMiddleware := jwtmiddleware.NewJWTMiddleware(publicPaths...)
    
    // Setup Chi router
    router := chi.NewRouter()
    
    // Apply JWT middleware to all requests
    router.Use(jwtMiddleware.Handler)
    
    // Public routes (middleware skips these)
    router.Get("/health", handleHealth)
    router.Get("/docs", handleDocs)
    
    // Protected routes (middleware validates JWT)
    router.Get("/api/entities", handleGetEntities)
    router.Post("/api/entities", handleCreateEntity)
    
    // Start server
    http.ListenAndServe(":8080", router)
}

func handleGetEntities(w http.ResponseWriter, r *http.Request) {
    // Extract JWT claims from context
    claims := jwtmiddleware.GetClaimsFromContext(r)
    if claims == nil {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }
    
    // Check tenant access
    tenantID := r.Header.Get("X-Tenant-ID")
    if err := jwtmiddleware.ValidateTenantAccess(claims, tenantID); err != nil {
        http.Error(w, "forbidden", http.StatusForbidden)
        return
    }
    
    // All database queries must be scoped by tenant
    // SELECT * FROM entities WHERE tenant_id = $1 AND user_id = $2
    
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(`{"entities": []}`))
}

func handleCreateEntity(w http.ResponseWriter, r *http.Request) {
    claims := jwtmiddleware.GetClaimsFromContext(r)
    if claims == nil {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }
    
    // Require admin role for creation
    if !jwtmiddleware.HasRole(claims, "admin") {
        http.Error(w, "insufficient permissions", http.StatusForbidden)
        return
    }
    
    // Enforce tenant scope
    if err := jwtmiddleware.ValidateTenantAccess(claims, r.Header.Get("X-Tenant-ID")); err != nil {
        http.Error(w, "forbidden", http.StatusForbidden)
        return
    }
    
    w.WriteHeader(http.StatusCreated)
    w.Write([]byte(`{"id": "entity-1"}`))
}
```

### Service-to-Service JWT

For services that call other services, include JWT in the request:

```go
import (
    "github.com/hondyman/semlayer/libs/jwt-middleware"
)

func callEntityManager(ctx context.Context, userID string, tenantID string) {
    // Create a service token
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
        "user_id": userID,
        "tenant_id": tenantID,
        "roles": []string{"service"},
    })
    
    tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
    if err != nil {
        log.Fatal(err)
    }
    
    // Make request with Authorization header
    req, _ := http.NewRequest("GET", "http://entity-manager:8087/api/entities", nil)
    req.Header.Set("Authorization", "Bearer " + tokenString)
    req.Header.Set("X-Tenant-ID", tenantID)
    
    client := &http.Client{}
    resp, err := client.Do(req)
    // ... handle response
}
```

---

## 🚀 Deployment Checklist

### Before Deploying
- [ ] All services have JWT_SECRET in docker-compose.yml
- [ ] All services import jwt-middleware library
- [ ] All services initialize middleware in main()
- [ ] All database queries scoped by tenant_id
- [ ] All JWT validation handles context claims
- [ ] Service-to-service calls include JWT tokens
- [ ] Role-based access control implemented
- [ ] Tenant isolation tested

### After Deploying
- [ ] All services start without errors
- [ ] Health endpoints accessible without JWT
- [ ] Protected endpoints reject requests without JWT
- [ ] Protected endpoints accept requests with valid JWT
- [ ] Tenant isolation verified (users can't access other tenants)
- [ ] Role enforcement tested
- [ ] Service-to-service communication works
- [ ] Logs show JWT validation happening

---

## 🧪 Testing Services

### 1. Get JWT Token
```bash
TOKEN=$(curl -X POST http://localhost:8001/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123"}' \
  | jq -r '.access_token')

echo "Token: $TOKEN"
```

### 2. Test Each Service with JWT

```bash
# Entity Manager
curl -X GET http://localhost:8087/api/entities \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: tenant-123"

# Validation Engine
curl -X GET http://localhost:8090/api/validations \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: tenant-123"

# Rule Engine
curl -X GET http://localhost:8091/api/rules \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: tenant-123"

# Search Service
curl -X GET http://localhost:8092/api/search \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: tenant-123"

# Policy Engine
curl -X GET http://localhost:8102/api/policies \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: tenant-123"

# Analytics Engine
curl -X GET http://localhost:8101/api/analytics \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: tenant-123"

# Compliance Engine
curl -X GET http://localhost:8095/api/compliance \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: tenant-123"
```

### 3. Test Without JWT (Should Fail)
```bash
curl -X GET http://localhost:8087/api/entities
# Expected: 401 Unauthorized
```

### 4. Test with Wrong Tenant (Should Fail)
```bash
curl -X GET http://localhost:8087/api/entities \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: different-tenant-123"
# Expected: 403 Forbidden
```

---

## 📊 Implementation Progress Tracking

| Service | docker-compose | Code Updated | JWT Validated | Testing | Status |
|---------|---|---|---|---|---|
| API Gateway | ✅ | ✅ | ✅ | ✅ | ✅ Complete |
| Auth Service | ✅ | ✅ | ✅ | ✅ | ✅ Complete |
| Backend | ✅ | ✅ | ✅ | ✅ | ✅ Complete |
| Entity Manager | ✅ | ⏳ | ⏳ | ⏳ | 🔄 In Progress |
| Validation Engine | ✅ | ⏳ | ⏳ | ⏳ | 🔄 Queued |
| Rule Engine | ✅ | ⏳ | ⏳ | ⏳ | 🔄 Queued |
| Search Service | ✅ | ⏳ | ⏳ | ⏳ | 🔄 Queued |
| Policy Engine | ✅ | ⏳ | ⏳ | ⏳ | 🔄 Queued |
| Analytics Engine | ✅ | ⏳ | ⏳ | ⏳ | 🔄 Queued |
| Compliance Engine | ✅ | ⏳ | ⏳ | ⏳ | 🔄 Queued |
| Event Router | ✅ | ⏳ | ⏳ | ⏳ | 🔄 Queued |
| Sync Worker | ✅ | ⏳ | ⏳ | ⏳ | 🔄 Queued |
| CDC Processor | ✅ | ⏳ | ⏳ | ⏳ | 🔄 Queued |
| Outbox Processor | ✅ | ⏳ | ⏳ | ⏳ | 🔄 Queued |
| Catalog Sync | ✅ | ⏳ | ⏳ | ⏳ | 🔄 Queued |
| Parity Checker | ✅ | ⏳ | ⏳ | ⏳ | 🔄 Queued |

Legend: ✅ = Complete | 🔄 = In Progress/Queued | ⏳ = Pending

---

## 📋 Next Steps

1. **Locate entity-manager main.go** - Find entry point and identify router setup
2. **Add JWT middleware import** - `import "github.com/hondyman/semlayer/libs/jwt-middleware"`
3. **Initialize middleware in main()** - Create instance with public paths
4. **Update all handlers** - Extract claims, validate tenant access
5. **Update all database queries** - Scope results by tenant_id
6. **Test with JWT tokens** - Verify middleware is working
7. **Repeat for remaining services** - Follow same pattern for each

---

**Last Updated**: 2026-02-23
**Status**: Phase 2 - Microservice JWT Integration  
**Next**: Entity Manager + Validation Engine updates
