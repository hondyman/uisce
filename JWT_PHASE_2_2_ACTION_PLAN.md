# JWT Phase 2.2 Action Plan - High-Priority Services

## 🎯 Objectives

Implement JWT authentication in the 3 highest-priority services to achieve ~50% code integration completion.

**Timeline**: 3-5 days  
**Services**: Validation Engine, Rule Engine, Analytics Engine

---

## 📋 Pre-Implementation Checklist

- [ ] Read [NODEJS_JWT_INTEGRATION_GUIDE.md](NODEJS_JWT_INTEGRATION_GUIDE.md)
- [ ] Review [JWT_QUICK_REFERENCE.md](JWT_QUICK_REFERENCE.md)
- [ ] Review [ENTITY_MANAGER_JWT_INTEGRATION.md](ENTITY_MANAGER_JWT_INTEGRATION.md) as reference
- [ ] Understand JWT token structure and claims
- [ ] Understand tenant isolation pattern

---

## 🔧 Service Implementation Template

This template is used for EACH service. Follow this exact pattern for consistency.

### Step 1: Locate Main Entry Point
```bash
# For Go services
find . -name "main.go" -path "*/[service-name]/*" -not -path "*/vendor/*"

# For Node.js services
find . -name "server.ts" -o -name "index.ts" -path "*/[service-name]/*"
```

### Step 2: Import JWT Middleware
```go
// For Go services
import "github.com/hondyman/semlayer/libs/jwt-middleware"

// For Node.js services
import { jwtMiddleware, getClaims } from '../../libs/jwt-middleware-node.js';
```

### Step 3: Configure Middleware
```go
// Go pattern (in main.go before route registration)
publicPaths := []string{
    "/health",
    "/ready",
    "/docs",
}
jwtMiddleware := jwtmiddleware.NewJWTMiddleware(publicPaths...)
router.Use(jwtMiddleware.Handler)
```

```typescript
// Node.js pattern (in server.ts before setupRoutes)
const publicPaths = ['/health', '/ready', '/docs'];
app.use(jwtMiddleware(publicPaths));
app.use(injectTenantFromClaims());
```

### Step 4: Update Route Handlers
```go
// Go pattern
func handleGetData(w http.ResponseWriter, r *http.Request) {
    claims := jwtmiddleware.GetClaimsFromContext(r)
    if claims == nil {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }
    
    // Use claims.TenantID for queries
    tenantID := claims.TenantID
}
```

```typescript
// Node.js pattern
router.get('/data', (req, res) => {
    const claims = getClaims(req);
    if (!claims) {
        return res.status(401).json({ error: 'Unauthorized' });
    }
    
    // Use claims.tenant_id for queries
    const tenantID = claims.tenant_id;
});
```

### Step 5: Update Database Queries
```sql
-- BEFORE (Vulnerable)
SELECT * FROM data WHERE id = $1

-- AFTER (Secure)
SELECT * FROM data WHERE tenant_id = $1 AND id = $2
```

### Step 6: Test
```bash
# Get token
TOKEN=$(curl -s http://localhost:8001/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"test@example.com","password":"password123"}' | jq -r '.access_token')

# Test endpoint
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:[SERVICE_PORT]/api/data

# Should return 200 OK
```

---

## 🚀 Priority 1: Validation Engine

### Basic Info
- **Language**: Go or Node.js (TBD - check codebase)
- **Location**: validation-engine/ directory
- **Main File**: main.go or server.ts
- **Port**: 8090 (from docker-compose.yml)
- **Routes**: See validation-engine route files

### Discovery Tasks
1. [ ] Determine if Go or Node.js service
2. [ ] Find main entry point
3. [ ] List all route handlers
4. [ ] Identify database usage patterns
5. [ ] Check for existing middleware setup

### Implementation Tasks
1. [ ] Import JWT middleware
2. [ ] Setup middleware in main
3. [ ] Update all GET handlers
4. [ ] Update all POST handlers
5. [ ] Update all PUT handlers
6. [ ] Update DELETE handlers
7. [ ] Add tenant filtering to queries
8. [ ] Test with JWT token

### Estimated Time: 1-2 hours

### Success Criteria
- [ ] Service starts without errors
- [ ] /health returns 200 without JWT
- [ ] /ready returns 200 without JWT
- [ ] All protected endpoints reject requests without JWT (401)
- [ ] Protected endpoints accept valid JWT (200)
- [ ] Tenant isolation verified (cross-tenant access returns 403)

---

## 🚀 Priority 2: Rule Engine

### Basic Info
- **Language**: Go or Node.js (TBD)
- **Location**: rule-engine/ directory
- **Main File**: main.go or server.ts
- **Port**: 8091 (from docker-compose.yml)
- **Routes**: See rule-engine route files

### Implementation Tasks
Same as Validation Engine (Step 1-6 above)

### Estimated Time: 1-2 hours

### Success Criteria
Same as Validation Engine

---

## 🚀 Priority 3: Analytics Engine

### Basic Info
- **Language**: Go or Node.js (TBD)
- **Location**: analytics_engine/ directory
- **Main File**: main.go or server.ts
- **Port**: 8101 (from docker-compose.yml)
- **Routes**: See analytics route files

### Implementation Tasks
Same as Validation Engine (Step 1-6 above)

### Estimated Time: 1-2 hours

### Success Criteria
Same as Validation Engine

---

## 📋 Implementation Checklist

### Before Starting
- [ ] Backup code repository
- [ ] Create feature branch
- [ ] Read all relevant documentation
- [ ] Test JWT token generation works

### Service Update
- [ ] Import JWT middleware
- [ ] Setup middleware configuration
- [ ] Update route handlers
- [ ] Update database queries
- [ ] Add error handling
- [ ] Run local tests

### Code Review
- [ ] All protected endpoints check for JWT
- [ ] All database queries include tenant filter
- [ ] Error responses correct (401, 403)
- [ ] No hardcoded tenant IDs
- [ ] Public paths properly excluded

### Testing
- [ ] Test health endpoint (no JWT needed)
- [ ] Test with valid JWT (200)
- [ ] Test without JWT (401)
- [ ] Test cross-tenant access (403)
- [ ] Test role-based access (if applicable)

### Deployment
- [ ] Build service image
- [ ] Update docker-compose.yml (if needed)
- [ ] Deploy to dev environment
- [ ] Monitor logs for errors
- [ ] Verify functionality with tests

---

## 🧪 Testing Commands (Use for Each Service)

```bash
# 1. Get JWT token
TOKEN=$(curl -s http://localhost:8001/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"test@example.com","password":"password123"}' | jq -r '.access_token')

# 2. Test health (should work without JWT)
curl -X GET http://localhost:[PORT]/health

# 3. Test protected endpoint WITH JWT
curl -X GET http://localhost:[PORT]/api/data \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: tenant-123"

# 4. Test protected endpoint WITHOUT JWT (should fail)
curl -X GET http://localhost:[PORT]/api/data
# Expected: 401 Unauthorized

# 5. Decode JWT to verify claims
echo $TOKEN | jq -R 'split(".")[1] | @base64d | fromjson'
```

---

## 🔍 Expected Differences by Service Type

### Go Services
- Use `jwtmiddleware.GetClaimsFromContext(r)` to extract claims
- Add middleware via `router.Use(jwtMiddleware.Handler)`
- Store tenant in `claims.TenantID` (capital case)
- Chi router or similar

### Node.js Services
- Use `getClaims(req)` to extract claims
- Add middleware via `app.use(jwtMiddleware(paths))`
- Store tenant in `claims.tenant_id` (lowercase)
- Express.js router

---

## 📚 Reference Files

| File | Purpose |
|------|---------|
| [NODEJS_JWT_INTEGRATION_GUIDE.md](NODEJS_JWT_INTEGRATION_GUIDE.md) | Node.js patterns |
| [JWT_MICROSERVICE_INTEGRATION_CHECKLIST.md](JWT_MICROSERVICE_INTEGRATION_CHECKLIST.md) | Service-by-service guide |
| [JWT_QUICK_REFERENCE.md](JWT_QUICK_REFERENCE.md) | Quick syntax reference |
| [ENTITY_MANAGER_JWT_INTEGRATION.md](ENTITY_MANAGER_JWT_INTEGRATION.md) | Reference implementation |

---

## ⚠️ Common Mistakes to Avoid

### ❌ WRONG
```go
// Using tenant from request body
tenantID := r.FormValue("tenant_id")

// Query without tenant filtering
db.Query("SELECT * FROM data WHERE id = ?", id)

// Forgetting to check for JWT claims
// (just proceed without validation)

// Logging full JWT tokens
log.Println("JWT: " + token)

// Hardcoding tenant IDs
if tenant == "tenant-123" { ... }
```

### ✅ CORRECT
```go
// Using tenant from JWT claims
claims := jwtmiddleware.GetClaimsFromContext(r)
tenantID := claims.TenantID

// Query with tenant filtering
db.Query("SELECT * FROM data WHERE tenant_id = ? AND id = ?", tenantID, id)

// Always check for JWT claims first
if claims == nil {
    return http.Error(w, "unauthorized", 401)
}

// Log only necessary info
log.Println("JWT validation: OK")

// Use claims for tenant logic
tenantID := claims.TenantID
```

---

## 📊 Progress Tracking

### Validation Engine
- [ ] Main entry point identified
- [ ] JWT middleware imported
- [ ] Middleware configured
- [ ] All handlers updated
- [ ] Database queries updated
- [ ] Local tests passed
- [ ] PR created
- [ ] Code review completed
- [ ] Deployed to dev
- [ ] Verified in dev

### Rule Engine
- [ ] Main entry point identified
- [ ] JWT middleware imported
- [ ] Middleware configured
- [ ] All handlers updated
- [ ] Database queries updated
- [ ] Local tests passed
- [ ] PR created
- [ ] Code review completed
- [ ] Deployed to dev
- [ ] Verified in dev

### Analytics Engine
- [ ] Main entry point identified
- [ ] JWT middleware imported
- [ ] Middleware configured
- [ ] All handlers updated
- [ ] Database queries updated
- [ ] Local tests passed
- [ ] PR created
- [ ] Code review completed
- [ ] Deployed to dev
- [ ] Verified in dev

---

## 🎯 Success Criteria for Phase 2.2

- [ ] All 3 services pass local testing
- [ ] All 3 services tested in dev environment
- [ ] All tenant isolation verified
- [ ] No breaking changes to existing APIs
- [ ] All error handling working correctly
- [ ] Logs show JWT validation happening
- [ ] Security team approved
- [ ] Ready for staging deployment

---

## 📞 Escalation Path

If you encounter issues:

1. **JWT validation errors**
   - Check JWT_SECRET matches in docker-compose.yml
   - Verify token not expired (1 hour default)
   - Check Authorization header format (Bearer <token>)

2. **Database query errors**
   - Ensure tenant_id column exists in tables
   - Check query parameter order
   - Verify tenant_id values in database

3. **Middleware loading errors**
   - Check import paths are correct
   - Verify module installed in Go mod/npm
   - Check for circular imports

4. **Service not starting**
   - Check logs: `docker logs [container-name]`
   - Verify JWT_SECRET available
   - Check port not already in use

5. **Still stuck?**
   - Review [ENTITY_MANAGER_JWT_INTEGRATION.md](ENTITY_MANAGER_JWT_INTEGRATION.md)
   - Check GitHub search for similar service
   - Post to security channel for review

---

## 📝 Notes

- Each service should follow the same pattern for consistency
- Keep changes minimal - only add JWT, don't refactor
- Test each service independently before combining
- Use the reference implementation (Entity Manager) as guide
- Document any deviations from standard pattern

---

**Created**: 2026-02-23  
**Target Start**: Immediately  
**Target Completion**: 3-5 days  
**Success Metric**: 60%+ code integration completion
