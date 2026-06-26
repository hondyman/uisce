# JWT Authentication Quick Reference

## 🔐 Security at a Glance

The Calendar Service uses **JWT Bearer tokens** for authentication, aligned with platform-wide security:

```
Request: Authorization: Bearer <JWT_token>
         X-Tenant-ID: <tenant_uuid>
         ↓
Middleware validates token signature & expiration
         ↓
Extracts user_id, tenant_id, roles to context
         ↓
Handler processes authenticated request
```

## ⚙️ Configuration

```bash
# Set JWT secret (generate with: openssl rand -hex 32)
export JWT_SECRET="your-strong-random-secret-min-32-chars"

# Optional: Allow unauth requests with X-User-ID (dev only!)
export DEV_ALLOW_UNAUTH_XUSER="false"
```

## 🚀 Making Requests

### Get Token from Auth Service
```bash
curl -X POST https://auth.example.com/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password"}'

# Response:
# {
#   "access_token": "eyJhbGc...",
#   "token_type": "Bearer"
# }
```

### Call Calendar Service API
```bash
curl -X GET https://calendar.example.com/api/v1/calendars \
  -H "Authorization: Bearer eyJhbGc..." \
  -H "X-Tenant-ID: tenant-uuid"
```

### Required Headers
```
Authorization: Bearer <token>  # Required
X-Tenant-ID: <tenant-uuid>     # Required for API calls
```

## 🔑 JWT Token Contents

```json
{
  "user_id": "user-uuid",           // Who
  "email": "user@example.com",      // Contact
  "tenant_id": "tenant-uuid",       // Which tenant
  "roles": ["admin", "user"],       // What they can do
  "permissions": ["read:calendar"],
  "exp": 1676003600,                // When expires
  "iat": 1676000000                 // When issued
}
```

## 👨‍💻 In Your Handler

```go
import "calendar-service/internal/middleware"

func (h *Handler) GetCalendars(w http.ResponseWriter, r *http.Request) {
  // Extract from context (middleware does this for you)
  userID := middleware.ExtractUserIDFromContext(r.Context())
  tenantID := middleware.ExtractTenantIDFromContext(r.Context())
  
  // Use in your logic
  calendars := h.service.GetCalendars(r.Context(), userID, tenantID)
  
  // Return to client
  json.NewEncoder(w).Encode(calendars)
}
```

## 🧪 Testing

### Generate Test Token
```go
func generateTestToken(userID, tenantID string) string {
  claims := jwt.MapClaims{
    "user_id": userID,
    "tenant_id": tenantID,
    "roles": []string{"admin"},
    "exp": time.Now().Add(time.Hour).Unix(),
  }
  token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
  tokenString, _ := token.SignedString([]byte(jwtSecret))
  return tokenString
}
```

### Make Test Request
```go
token := generateTestToken("user-123", "tenant-456")
req := httptest.NewRequest("GET", "/api/v1/calendars", nil)
req.Header.Set("Authorization", "Bearer " + token)
req.Header.Set("X-Tenant-ID", "tenant-456")
```

## ❌ Error Responses

### 401 Unauthorized - Auth Failed
```bash
Missing Authorization header
Invalid token format
Invalid signature
Expired token
Missing required claims
```

**Fix:** Verify token is valid, not expired, and JWT_SECRET matches

### 403 Forbidden - Access Denied
```bash
Access denied for requested tenant
User not authorized for tenant
```

**Fix:** Verify X-Tenant-ID matches one of user's tenants in JWT

### 400 Bad Request - Bad Input
```bash
Invalid request body
Missing required fields
```

**Fix:** Check request format and required fields

## 🔍 Debugging

### Check Token Validity
Visit https://jwt.io and paste your token to see claims

### Verify JWT_SECRET
```bash
# Should match token's signature
echo $JWT_SECRET
openssl rand -hex 32  # Generate new one
```

### Check Request Headers
```bash
curl -X GET https://calendar.example.com/api/v1/calendars \
  -H "Authorization: Bearer <token>" \
  -H "X-Tenant-ID: <tenant>" \
  -v  # Shows all headers
```

### Enable Debug Logging
```bash
export LOG_LEVEL=debug
# Watch for JWT validation logs
```

## 📚 Key Context Functions

```go
// Extract values from authenticated context
middleware.ExtractUserIDFromContext(ctx)       // -> "user-uuid"
middleware.ExtractTenantIDFromContext(ctx)     // -> "tenant-uuid"
middleware.ExtractTenantsFromContext(ctx)      // -> ["t1", "t2"]
middleware.ExtractRolesFromContext(ctx)        // -> ["admin"]

// Check role
middleware.HasRole(ctx, "admin")               // -> bool
```

## 🛡️ Security Rules

✅ **DO:**
- ✅ Use strong JWT_SECRET (min 32 chars)
- ✅ Include Authorization header
- ✅ Validate tenant from X-Tenant-ID
- ✅ Log auth failures
- ✅ Rotate secrets periodically

❌ **DON'T:**
- ❌ Commit JWT_SECRET to git
- ❌ Log full tokens
- ❌ Skip tenant validation
- ❌ Use weak secrets
- ❌ Expose error details

## 🚨 Troubleshooting

| Problem | Cause | Fix |
|---------|-------|-----|
| 401 Missing header | No Authorization | Add header |
| 401 Invalid token | Bad signature | Verify JWT_SECRET |
| 401 Expired | Token old | Get new token |
| 403 Forbidden | Tenant mismatch | Check X-Tenant-ID |
| 400 Bad request | Invalid JWT format | Check token format |

## 📞 Quick Links

- Full Auth Guide: `docs/AUTHENTICATION.md`
- Setup Guide: `docs/SECURITY_SETUP.md`
- Platform Alignment: `docs/JWT_ALIGNMENT_MATRIX.md`
- Implementation: `docs/IMPLEMENTATION_COMPLETE.md`

## 🎯 One-Minute Setup

1. **Generate JWT secret:**
   ```bash
   export JWT_SECRET=$(openssl rand -hex 32)
   ```

2. **Start service:**
   ```bash
   go run ./cmd/main.go
   ```

3. **Test:**
   ```bash
   TOKEN="eyJhbGc..." # From Auth Service
   curl -H "Authorization: Bearer $TOKEN" \
        -H "X-Tenant-ID: my-tenant" \
        http://localhost:8080/api/v1/calendars
   ```

Done! 🎉 JWT authentication is active.

---

**Last Updated:** February 17, 2026  
**Status:** Production Ready ✅  
**Questions?** See full documentation in `docs/` folder
