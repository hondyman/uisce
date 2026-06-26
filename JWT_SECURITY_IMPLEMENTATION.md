# JWT Security & Multi-Service Authentication Implementation

## Overview

This document describes how to implement JWT-based authentication and authorization across all SemLayer services to create a secure, multi-tenant system.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     Frontend (React)                             │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ HTTP + Bearer Token
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                     API Gateway (Go)                             │
│  ✓ Validates JWT token                                          │
│  ✓ Extracts user/tenant info                                    │
│  ✓ Forwards Authorization header to backend                     │
│  ✓ Enforces rate limiting per tenant                            │
└─────────────────────────────────────────────────────────────────┘
                    │           │           │
        ┌───────────┴───────────┴───────────┴────────────┐
        │                                                 │
        ▼ HTTP + JWT                                    ▼
┌────────────────────────┐              ┌────────────────────────┐
│   Auth Service (Node)  │              │ Backend Service (Go)   │
│ ✓ Issues JWT tokens    │              │ ✓ Validates JWT token  │
│ ✓ Manages users        │              │ ✓ Extracts claims      │
│ ✓ OAuth integration    │              │ ✓ Enforces RBAC        │
└────────────────────────┘              └────────────────────────┘
                                               │
                    ┌──────────────────────────┴──────────────────────────┐
                    │ HTTP + JWT (internal service calls)                  │
        ┌───────────┴────────────┬────────────┬─────────────┬────────────┬─────────────┐
        │                        │            │             │            │             │
        ▼                        ▼            ▼             ▼            ▼             ▼
┌─────────────────┐  ┌──────────────────┐ ┌────────────┐ ┌──────────┐ ┌────────────┐ ┌──────────┐
│ Entity Manager  │  │  Analytics Eng   │ │ Compliance │ │Validation│ │Compliance  │ │Notifications│
│ ✓ JWT validation│  │ ✓ JWT validation │ │ Engine     │ │Engine    │ │Engine      │ │            │
│ ✓ Tenant filter │  │ ✓ Tenant filter  │ │ ✓ JWT ✓    │ │✓ JWT ✓   │ │✓ JWT ✓     │ │ ✓ JWT ✓    │
└─────────────────┘  └──────────────────┘ └────────────┘ └──────────┘ └────────────┘ └──────────┘
        │                    │                   │            │            │             │
        └────────────────────┴───────────────────┴────────────┴────────────┴─────────────┘
                                    │
                                    ▼
                    ┌──────────────────────────────┐
                    │   PostgreSQL (Alpha DB)      │
                    │   ✓ Multi-tenant data        │
                    │   ✓ RLS (Row-level security) │
                    └──────────────────────────────┘
```

## Implementation Steps

### 1. Shared JWT Middleware Library

✅ **COMPLETE** - Located at `libs/jwt-middleware/`

Provides:
- Token validation (HS256)
- Claims extraction
- Tenant isolation validation
- Role-based access control
- HTTP middleware components

Usage:
```go
import "github.com/hondyman/semlayer/libs/jwt-middleware"

middleware := jwtmiddleware.NewJWTMiddleware(
    "/health",
    "/api/auth/login",
)
router.Use(middleware.Handler)
```

### 2. Environment Configuration

**Required for all services:**

```bash
# JWT Configuration
JWT_SECRET=dev-jwt-secret-key-change-in-production
JWT_EXPIRY=1h
TOKEN_REVOCATION_TTL=24h

# Security Flags
ENABLE_SECURITY=true
JWT_ENFORCE=true
JWT_VALIDATE_TENANT=true
JWT_VALIDATE_ROLES=true
```

### 3. Implementation Checklist by Service

#### API Gateway (✅ PARTIALLY DONE)
- [x] JWT validation middleware
- [x] Authorization header forwarding to Hasura
- [x] Tenant header forwarding
- [ ] Service-to-service JWT validation (for internal calls)
- [ ] Token refresh endpoint
- [ ] Token revocation endpoint

**File:** `api-gateway/main.go`
**Changes needed:**
- Add service-to-service JWT signing
- Add token refresh logic
- Add token revocation tracking

#### Auth Service ✅ COMPLETE
- [x] JWT token issuance
- [x] User authentication
- [x] Token refresh
- [x] Password management

**File:** `auth-service/server.js`

#### Backend Service 🔄 IN PROGRESS
- [x] JWT validation for GraphQL
- [ ] JWT validation for all HTTP endpoints
- [ ] Service-to-service JWT validation
- [ ] Internal service calls with JWT

**File:** `backend/internal/api/main.go`
**Changes needed:**
- Add JWT middleware to all routes
- Update internal service calls to include JWT
- Add rolevalidation on protected endpoints

#### BP Backend (Business Process)
- [ ] JWT validation middleware
- [ ] Tenant-scoped data access
- [ ] Role-based feature access

**File:** `bp-backend/main.go`

#### Entity Manager
- [ ] JWT validation middleware
- [ ] Multi-tenant entity filtering

**File:** `entity-manager/main.go`

#### Analytics Engine
- [ ] JWT validation on analytics queries
- [ ] Tenant-isolated query results

**File:** `services/analytics-engine/main.go` (if exists)

#### Compliance Engine
- [ ] JWT validation for compliance checks
- [ ] Tenant isolation

**File:** Depends on service structure

#### Other Microservices
- Validation Engine
- Search Service
- Rule Engine
- Policy Engine
- Notifications Service`

## Security Requirements

### Authentication Flow

1. **User login at frontend**
   ```
   POST /auth/login
   Body: { email, password }
   Response: { access_token, refresh_token, expires_in, user }
   ```

2. **Frontend stores JWT in localStorage**
   ```javascript
   localStorage.setItem('auth_token', response.access_token);
   ```

3. **Frontend includes JWT in all API requests**
   ```javascript
   headers: {
     'Authorization': 'Bearer ' + token,
     'X-Tenant-ID': tenantId,     // Optional tenant override
   }
   ```

4. **API Gateway validates JWT**
   - Extracts token from Authorization header
   - Validates signature using JWT_SECRET
   - Extracts claims (user_id, tenant_id, roles)
   - Stores in request context

5. **API Gateway forwards to backend**
   ```
   Authorization: Bearer <token>
   X-Tenant-ID: <tenant_id>
   X-User-ID: <user_id>
   ```

6. **Backend service validates JWT independently**
   - Uses same JWT_SECRET
   - Validates token signature
   - Checks token expiration
   - Enforces tenant isolation
   - Validates user roles for protected operations

### Query Isolation (Multi-Tenant)

All services must filter queries by tenant:

```sql
-- Automatic tenant filtering
SELECT * FROM table WHERE tenant_id = $1
```

```go
// Service-side filtering
rows, err := db.QueryContext(ctx, 
    "SELECT * FROM users WHERE tenant_id = ?",
    tenantID,
)
```

### Authorization Rules

1. **Public Endpoints (no JWT required)**
   - `/health`
   - `/api/auth/login`
   - `/api/auth/refresh`
   - `/docs`

2. **Protected Endpoints (JWT required)**
   - All `/api/*` endpoints
   - All `/graphql` endpointsAll microservices endpoints

3. **Admin Endpoints (admin role required)**
   - User management
   - Service configuration
   - Security settings

4. **Tenant-Scoped Endpoints (tenant validation required)**
   - All data access
   - All analytics queries
   - All compliance operations

## Traffic Flow

### Frontend → API Gateway → Backend

```
1. Frontend sends:
   GET /api/business-terms
   Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
   X-Tenant-ID: 910638ba-a459-4a3f-bb2d-78391b0595f6

2. API Gateway:
   - Validates token signature (using JWT_SECRET)
   - Extracts claims: { user_id, tenant_id, roles }
   - Stores in request context
   - Forwards to backend with Authorization header

3. Backend service:
   - Extracts JWT from Authorization header
   - Validates token (using same JWT_SECRET)
   - Filters queries by tenant_id from JWT
   - Returns only tenant-scoped data
```

### Backend → Internal Services

```
1. Backend calls Entity Manager:
   POST /api/entities
   Authorization: Bearer <signed-jwt>
   Body: { entity_data }

2. Entity Manager:
   - Validates JWT signature (same secret)
   - Confirms tenant access
   - Processes entity creation
   - Returns result
```

## Deployment Checklist

- [ ] All services deployed with JWT_SECRET env var
- [ ] JWT_SECRET is 32+ characters (see `.env` file)
- [ ] JWT_SECRET is the same across all services
- [ ] All services have JWT validation enabled
- [ ] API Gateway forwards Authorization header
- [ ] All services filter by tenant_id
- [ ] Admin endpoints require admin role
- [ ] Logging configured for auth events
- [ ] Error handling for expired tokens
- [ ] Token refresh working end-to-end
- [ ] Monitoring alerts for auth failures
- [ ] Security headers enabled (CORS, CSP, etc.)

## Monitoring & Logging

### Key Metrics to Track

1. **Authentication Events**
   - Successful logins
   - Failed login attempts
   - Token refresh events

2. **Authorization Events**
   - Permission denied errors
   - Role changes
   - Tenant access changes

3. **Token Events**
   - Token expiration
   - Token revocation
   - Invalid signature errors

4. **Service Health**
   - JWT validation latency
   - Database connection errors
   - Service availability

### Logging Format

```json
{
  "timestamp": "2026-02-23T10:30:45Z",
  "level": "INFO",
  "event": "jwt_validated",
  "user_id": "36f45238-bac6-4b06-a495-6155c43df552",
  "tenant_id": "910638ba-a459-4a3f-bb2d-78391b0595f6",
  "endpoint": "/api/business-terms",
  "method": "GET",
  "status": 200,
  "duration_ms": 123
}
```

## Troubleshooting

### Issue: "JWSError JWSInvalidSignature"
- **Cause**: JWT signed with different secret
- **Fix**: Ensure JWT_SECRET is same across all services

### Issue: "Missing authorization header"
- **Cause**: Frontend not sending token
- **Fix**: Check localStorage, network tab, and Apollo client config

### Issue: "User does not have access to tenant"
- **Cause**: Token has different tenant_id
- **Fix**: Check X-Tenant-ID header in request

### Issue: "Forbidden: insufficient permissions"
- **Cause**: User role doesn't match endpoint requirement
- **Fix**: Assign appropriate role to user

## Related Documentation

- [Multi-Tenant Architecture](../../docs/MULTI_TENANT_ARCHITECTURE.md)
- [API Gateway Design](../../docs/API_GATEWAY_DESIGN.md)
- [Security Best Practices](../../docs/SECURITY_HARDENING.md)
- [JWT Middleware Library](./libs/jwt-middleware/README.md)

## Implementation Status

| Service | JWT Validation | Tenant Filtering | RBAC | Service-to-Service |
|---------|---|---|---|---|
| API Gateway | ✅ | ✅ | ✅ | 🔄 |
| Auth Service | ✅ | ✅ | ✅ | ✅ |
| Backend | ✅ | ✅ | ✅ | 🔄 |
| BP Backend | ❌ | ❌ | ❌ | ❌ |
| Entity Manager | ❌ | ❌ | ❌ | ❌ |
| Analytics Engine | ❌ | ❌ | ❌ | ❌ |
| Compliance Engine | ❌ | ❌ | ❌ | ❌ |
| Validation Engine | ❌ | ❌ | ❌ | ❌ |
| Notifications | ❌ | ❌ | ❌ | ❌ |

Legend: ✅ = Complete | 🔄 = In Progress | ❌ = Not Started

## Next Steps

1. Review this architecture with security team
2. Implement JWT validation in all services (from template)
3. Test end-to-end JWT flow
4. Deploy to staging environment
5. Run security penetration testing
6. Deploy to production with monitoring

## Questions?

Contact: security@semlayer.local
Review Date: 2026-Q2
Next Review: 2026-Q3
