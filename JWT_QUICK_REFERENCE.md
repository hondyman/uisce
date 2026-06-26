# JWT Integration - Quick Reference Card

## 🚀 30-Second Quickstart

### For Go Services
```go
import "github.com/hondyman/semlayer/libs/jwt-middleware"

// In main()
jwtMiddleware := jwtmiddleware.NewJWTMiddleware(
    "/health", "/docs", "/ready",
)
router.Use(jwtMiddleware.Handler)

// In handlers
claims := jwtmiddleware.GetClaimsFromContext(r)
fmt.Println(claims.TenantID) // Use for queries
```

### For Node.js Services
```typescript
import { jwtMiddleware, getClaims } from '../../libs/jwt-middleware-node.js'

// In express setup
app.use(jwtMiddleware(['/health', '/ready', '/docs']))

// In handlers
const claims = getClaims(req)
console.log(claims.tenant_id) // Use for queries
```

## ✅ Checklist

- [ ] Service has JWT_SECRET in docker-compose
- [ ] Service imports JWT middleware
- [ ] Middleware initialized in main/server file
- [ ] Public paths defined (/health, /ready, /docs)
- [ ] All handlers extract claims from context
- [ ] All DB queries have `WHERE tenant_id = ?`
- [ ] Role checks added where needed
- [ ] Tested with valid JWT ✅
- [ ] Tested without JWT ❌
- [ ] Tested tenant isolation ❌

## 📊 JWT Claims Cheat Sheet

```typescript
{
  user_id: "user-123",           // Current user
  email: "user@example.com",     // User email
  tenant_id: "tenant-abc",       // Primary tenant
  tenant_ids: ["tenant-abc"],    // All accessible
  roles: ["admin", "user"],      // User roles
  is_core_admin: false,          // System admin?
  is_active: true,               // Account active?
  iat: 1708600000,               // Issued at (Unix)
  exp: 1708603600                // Expires at (Unix)
}
```

## 🔐 Database Query Patterns

### Single Tenant (MOST COMMON)
```sql
-- Use this for 99% of queries
SELECT * FROM entities WHERE tenant_id = $1
```

### With User Scope
```sql
-- For user's own data
SELECT * FROM accounts 
WHERE tenant_id = $1 AND created_by = $2
```

### Multi-Tenant (Rare)
```sql
-- For users with multi-tenant access
SELECT * FROM entities 
WHERE tenant_id = ANY($1)
```

## 🩹 Common Issues & Fixes

| Problem | Cause | Fix |
|---------|-------|-----|
| 401 Unauthorized | Missing JWT | Add Authorization header |
| 401 Invalid Token | Expired or bad signature | Get fresh token from login |
| 403 Forbidden | Wrong tenant | Check X-Tenant-ID header |
| 403 Insufficient Role | Missing role | Use different account or grant role |
| Data from wrong tenant | Query not scoped | Add `WHERE tenant_id = ?` |

## 🧪 Test One-Liners

```bash
# Get token
TOKEN=$(curl -s http://localhost:8001/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"test@example.com","password":"password123"}' \
  | jq -r '.access_token')

# Test endpoint
curl -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: tenant-123" \
  http://localhost:8080/api/data

# Decode token
echo $TOKEN | jq -R 'split(".")[1] | @base64d | fromjson'
```

## 🔗 Service-to-Service JWT

```go
// When calling another service
token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
    "user_id": claims.UserID,
    "tenant_id": claims.TenantID,
})
tokenString, _ := token.SignedString([]byte(os.Getenv("JWT_SECRET")))

// Include in request
req.Header.Set("Authorization", "Bearer "+tokenString)
```

## 📚 Documentation Files

| File | Purpose |
|------|---------|
| [JWT_DEPLOYMENT_GUIDE.md](JWT_DEPLOYMENT_GUIDE.md) | Setup & deployment |
| [JWT_MICROSERVICE_INTEGRATION_CHECKLIST.md](JWT_MICROSERVICE_INTEGRATION_CHECKLIST.md) | Per-service guide |
| [NODEJS_JWT_INTEGRATION_GUIDE.md](NODEJS_JWT_INTEGRATION_GUIDE.md) | Express.js details |
| [JWT_SECURITY_IMPLEMENTATION.md](JWT_SECURITY_IMPLEMENTATION.md) | Architecture |
| [JWT_IMPLEMENTATION_SUMMARY.md](JWT_IMPLEMENTATION_SUMMARY.md) | Complete overview |

## ⚠️ Critical Don'ts

❌ **Don't**
- Skip tenant filtering in queries
- Trust tenantId from request body
- Put JWT_SECRET in code
- Log full JWT tokens
- Use HTTP in production
- Skip service-to-service JWT

✅ **Do**
- Always filter by tenant_id
- Use tenant_id from JWT claims
- Use environment variables
- Log "JWT validation: OK" instead
- Use HTTPS everywhere
- Include JWT in all service calls

## 🎯 Priority Order for Implementation

1. **First**: API Gateway + Auth (Already ✅)
2. **Second**: Backend + Core Services
3. **Third**: Analytics + Compliance Engines
4. **Fourth**: All Other Services
5. **Fifth**: Service-to-Service JWT

## 💬 Quick Support

**Q: Where do I get JWT_SECRET?**
A: Environment variable, set in docker-compose.yml

**Q: How long does JWT last?**
A: 1 hour default (configurable in auth service)

**Q: Can users access multiple tenants?**
A: Yes, if they have multiple entries in tenant_ids

**Q: What if someone has admin role?**
A: They can access any tenant they're assigned to

**Q: How do I test without JWT?**
A: Call /health or /ready endpoints (no JWT needed)

## 🚦 Go/Node.js Comparison

| Feature | Go | Node.js |
|---------|----|----|
| Middleware Setup | `router.Use()` | `app.use()` |
| Library Path | `libs/jwt-middleware/` | `libs/jwt-middleware-node.ts` |
| Get Claims | `GetClaimsFromContext(r)` | `getClaims(req)` |
| Require Role | `middleware.HasRole()` | `requireRole()` middleware |
| Chi Router | ✅ Yes | N/A (Express) |
| Express Support | N/A | ✅ Yes |

---

**Print this out or bookmark for quick reference!**

Last updated: 2026-02-23
