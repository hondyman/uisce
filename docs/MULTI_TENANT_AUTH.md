# Multi-Tenant JWT Authentication - Canonical Contract

## 🎯 JWT Claim Contract (Final Specification)

This is the **authoritative JWT shape** for semlayer's multi-tenant authentication system.

### JWT Payload Structure

```json
{
  "sub": "user-uuid",
  "email": "ops@semlayer.com",
  "roles": ["admin", "global_ops"],
  "scopes": ["read:tenants", "read:metrics", "manage:incidents"],
  
  "tenant_scope": "single",
  "tenant_id": "tenant-uuid",
  "tenant_ids": ["t1", "t2"],
  "org_id": "org-uuid",
  
  "iat": 1739450000,
  "exp": 1739453600,
  "jti": "uuid-of-access-token"
}
```

### Claim Definitions

| Claim | Type | Required | Description |
|-------|------|----------|-------------|
| `sub` | string | ✅ | User UUID (primary identifier) |
| `email` | string | ✅ | User email address |
| `roles` | string[] | ✅ | Coarse-grained roles (`user`, `admin`, `global_ops`) |
| `scopes` | string[] | ✅ | Fine-grained permissions (`read:tenants`, `manage:incidents`) |
| `tenant_scope` | string | ✅ | Tenant access pattern: `"single"` \| `"multi"` \| `"all"` |
| `tenant_id` | string | 🔀 | Required if `tenant_scope = "single"` |
| `tenant_ids` | string[] | 🔀 | Required if `tenant_scope = "multi"` |
| `org_id` | string | ⭕ | Optional organizational grouping |
| `iat` | number | ✅ | Issued at (Unix timestamp) |
| `exp` | number | ✅ | Expires at (Unix timestamp) |
| `jti` | string | ✅ | JWT ID (for revocation) |

---

## 👥 Three User Types

### 1. Single-Tenant User (Normal Customer)

**Use Case:** Regular customer bound to their tenant

**JWT Example:**
```json
{
  "sub": "user-123",
  "email": "alice@tenant-a.com",
  "roles": ["user"],
  "scopes": ["read:self", "write:self"],
  "tenant_scope": "single",
  "tenant_id": "tenant-a-uuid",
  "iat": 1739450000,
  "exp": 1739453600,
  "jti": "jwt-uuid-1"
}
```

**Behavior:**
- ✅ Can only access `tenant-a-uuid`
- ❌ Cannot set `X-Tenant-ID` to any other tenant
- ❌ Cannot access cross-tenant APIs
- ✅ `X-Tenant-ID: tenant-a-uuid` auto-injected by gateway

---

### 2. Multi-Tenant Ops (Regional Support)

**Use Case:** Support engineer assigned to specific customers

**JWT Example:**
```json
{
  "sub": "ops-42",
  "email": "ops@region.com",
  "roles": ["ops", "support"],
  "scopes": ["read:tenants", "read:metrics", "read:support_tickets"],
  "tenant_scope": "multi",
  "tenant_ids": ["tenant-a-uuid", "tenant-b-uuid", "tenant-c-uuid"],
  "iat": 1739450000,
  "exp": 1739453600,
  "jti": "jwt-uuid-2"
}
```

**Behavior:**
- ✅ Can access ONLY `tenant-a`, `tenant-b`, `tenant-c`
- ✅ Must explicitly set `X-Tenant-ID` in request
- ❌ Cannot access `tenant-d` or any other tenant
- ❌ Cannot access cross-tenant aggregated views
- ✅ Frontend shows tenant selector with 3 options

---

### 3. Global Ops (Platform Admin)

**Use Case:** SemLayer internal ops, SRE, platform engineering

**JWT Example:**
```json
{
  "sub": "global-1",
  "email": "admin@semlayer.com",
  "roles": ["admin", "global_ops"],
  "scopes": ["read:tenants", "read:metrics", "manage:incidents", "manage:platform"],
  "tenant_scope": "all",
  "iat": 1739450000,
  "exp": 1739453600,
  "jti": "jwt-uuid-3"
}
```

**Behavior:**
- ✅ Can access **any** tenant
- ✅ Must **explicitly** set `X-Tenant-ID` for tenant-scoped APIs
- ❌ Cannot access tenant-scoped APIs without `X-Tenant-ID`
- ✅ Can access cross-tenant aggregated metrics (`/api/platform/metrics`)
- ✅ Frontend shows tenant search/selector with all tenants

---

## 🔒 Gateway Enforcement Rules

### Rule Matrix

| tenant_scope | Request X-Tenant-ID | Allowed? | Action |
|--------------|---------------------|----------|--------|
| `single` | (empty) | ✅ | Auto-inject `claims.tenant_id` |
| `single` | = `claims.tenant_id` | ✅ | Forward as-is |
| `single` | ≠ `claims.tenant_id` | ❌ | **403 Forbidden** |
| `multi` | (empty) | ❌ | **403 Forbidden** (must be explicit) |
| `multi` | ∈ `claims.tenant_ids` | ✅ | Forward as-is |
| `multi` | ∉ `claims.tenant_ids` | ❌ | **403 Forbidden** |
| `all` | (empty) | ⚠️ | **403** for tenant APIs, OK for platform APIs |
| `all` | (any value) | ✅ | Forward as-is (audit log) |

### Pseudocode

```go
func EnforceTenantContext(r *http.Request, claims *JWTClaims) error {
    requested := r.Header.Get("X-Tenant-ID")
    
    switch claims.TenantScope {
    case "single":
        if requested != "" && requested != claims.TenantID {
            return Forbidden("Cannot access other tenants")
        }
        r.Header.Set("X-Tenant-ID", claims.TenantID)
        
    case "multi":
        if requested == "" {
            return Forbidden("Must specify X-Tenant-ID")
        }
        if !contains(claims.TenantIDs, requested) {
            return Forbidden("Not authorized for this tenant")
        }
        r.Header.Set("X-Tenant-ID", requested)
        
    case "all":
        if requested == "" && isTenantScopedEndpoint(r.URL.Path) {
            return Forbidden("Must specify X-Tenant-ID for tenant APIs")
        }
        if requested != "" {
            r.Header.Set("X-Tenant-ID", requested)
            auditLog(claims.Sub, requested, r.URL.Path)
        }
    }
    
    return nil
}
```

---

## 🔗 Gateway → Hasura → Backend Header Mapping

| JWT Claim | Gateway Header | Hasura Session Var | Backend Usage |
|-----------|----------------|--------------------|---------------------------------|
| `sub` | `X-User-Id` | `x-hasura-user-id` | User identification, audit logs |
| `email` | `X-User-Email` | - | Logging, user lookup |
| `roles` | `X-Roles` | `x-hasura-role` | RBAC, permission checks |
| `scopes` | `X-Scopes` | - | Fine-grained permissions |
| `tenant_scope` | `X-Tenant-Scope` | `x-hasura-tenant-scope` | Enforcement validation |
| validated `tenant_id` | `X-Tenant-ID` | `x-hasura-tenant-id` | **RLS**, SQL filters |
| `org_id` | `X-Org-Id` | - | Organizational queries |

### Example Request Flow

**Client Request:**
```http
GET /api/business-objects HTTP/1.1
Authorization: Bearer eyJhbGci...
X-Tenant-ID: tenant-b-uuid
```

**Gateway Processing:**
1. Decode JWT → `tenant_scope: "multi"`, `tenant_ids: ["tenant-a", "tenant-b"]`
2. Validate `X-Tenant-ID: tenant-b-uuid` ∈ `tenant_ids` ✅
3. Add headers before proxying

**Proxied to Backend:**
```http
GET /api/business-objects HTTP/1.1
X-User-Id: ops-42
X-User-Email: ops@region.com
X-Roles: ops,support
X-Scopes: read:tenants,read:metrics
X-Tenant-Scope: multi
X-Tenant-ID: tenant-b-uuid  ← validated
```

**Backend SQL:**
```sql
SELECT * FROM business_objects 
WHERE tenant_id = $1  -- = 'tenant-b-uuid' from X-Tenant-ID header
```

---

## 🛡️ Hasura RLS Policies

### tenants

```sql
-- Strict tenant isolation (recommended)
CREATE POLICY tenant_isolation ON table_name
FOR ALL
USING (
  tenant_id = current_setting('hasura.user', true)::json->>'x-hasura-tenant-id'
);
```

### Alternative: Allow global ops bypass (not recommended)

```sql
CREATE POLICY tenant_with_ops_bypass ON table_name
FOR ALL
USING (
  tenant_id = current_setting('hasura.user', true)::json->>'x-hasura-tenant-id'
  OR current_setting('hasura.user', true)::json->>'x-hasura-role' = 'global_ops'
);
```

**Recommendation:** Use strict isolation. Global ops should still set explicit `X-Tenant-ID`.

---

## 📊 Example Scenarios

### Scenario 1: Tenant User Tries to Access Other Tenant

**Request:**
```bash
curl -H "Authorization: Bearer <single_tenant_jwt>" \
     -H "X-Tenant-ID: other-tenant" \
     http://localhost:8001/api/data
```

**Response:**
```json
{
  "error": "Forbidden",
  "message": "Cannot access other tenants"
}
```

### Scenario 2: Global Ops Accesses Specific Tenant

**Request:**
```bash
curl -H "Authorization: Bearer <global_ops_jwt>" \
     -H "X-Tenant-ID: tenant-xyz" \
     http://localhost:8001/api/tenants/tenant-xyz/business-objects
```

**Response:**
```json
{
  "business_objects": [...]
}
```

**Audit Log:**
```json
{
  "user_id": "global-1",
  "email": "admin@semlayer.com",
  "tenant_id": "tenant-xyz",
  "path": "/api/tenants/tenant-xyz/business-objects",
  "timestamp": "2026-02-13T19:30:00Z"
}
```

### Scenario 3: Global Ops Cross-Tenant Metrics

**Request:**
```bash
curl -H "Authorization: Bearer <global_ops_jwt>" \
     http://localhost:8001/api/platform/metrics
```

**Response:**
```json
{
  "total_tenants": 42,
  "active_users": 1337,
  "total_revenue": "$1M"
}
```

*(No `X-Tenant-ID` required for platform-level APIs)*

---

## 🧪 Test Cases

### Test 1: Single-Tenant Isolation
```bash
# Login as single-tenant user
TOKEN=$(curl -X POST http://localhost:8001/api/auth/login \
  -d '{"email":"alice@tenant-a.com","password":"pass"}' \
  | jq -r .access_token)

# Should succeed
curl -H "Authorization: Bearer $TOKEN" \
     http://localhost:8001/api/data

# Should fail (403)
curl -H "Authorization: Bearer $TOKEN" \
     -H "X-Tenant-ID: tenant-b" \
     http://localhost:8001/api/data
```

### Test 2: Multi-Tenant Ops
```bash
# Login as multi-tenant ops
TOKEN=$(curl -X POST http://localhost:8001/api/auth/login \
  -d '{"email":"ops@region.com","password":"pass"}' \
  | jq -r .access_token)

# Should fail (no X-Tenant-ID)
curl -H "Authorization: Bearer $TOKEN" \
     http://localhost:8001/api/data

# Should succeed (in tenant_ids)
curl -H "Authorization: Bearer $TOKEN" \
     -H "X-Tenant-ID: tenant-a" \
     http://localhost:8001/api/data

# Should fail (not in tenant_ids)
curl -H "Authorization: Bearer $TOKEN" \
     -H "X-Tenant-ID: tenant-z" \
     http://localhost:8001/api/data
```

### Test 3: Global Ops
```bash
# Login as global ops
TOKEN=$(curl -X POST http://localhost:8001/api/auth/login \
  -d '{"email":"admin@semlayer.com","password":"Admin123!"}' \
  | jq -r .access_token)

# Should succeed (any tenant)
curl -H "Authorization: Bearer $TOKEN" \
     -H "X-Tenant-ID: any-tenant" \
     http://localhost:8001/api/data

# Should fail for tenant APIs
curl -H "Authorization: Bearer $TOKEN" \
     http://localhost:8001/api/data

# Should succeed for platform APIs
curl -H "Authorization: Bearer $TOKEN" \
     http://localhost:8001/api/platform/metrics
```

---

## 🗂️ Summary

| User Type | tenant_scope | Requires X-Tenant-ID? | Can Access |
|-----------|--------------|------------------------|------------|
| Tenant User | `"single"` | No (auto-injected) | Own tenant only |
| Multi-Tenant Ops | `"multi"` | Yes | Assigned tenants |
| Global Ops | `"all"` | Yes (for tenant APIs) | Any tenant |

**Key Principles:**
1. ✅ **Hard isolation** for tenant users
2. ✅ **Constrained access** for multi-tenant ops
3. ✅ **Explicit context** for global ops
4. ✅ **Complete audit trail** for all access
5. ✅ **Enforced at every layer** (gateway, Hasura, backend)

---

## 🔗 Related

- [`docs/security/northwind-profiles.md`](security/northwind-profiles.md) —
  Northwind ABAC Gold Copy profiles and the core/custom override pattern
  for `security.identity_profile_mappings`, `security.security_profiles`,
  and `public.abac_policies`.
- [`backend/migrations/000062_abac_security_profiles.sql`](../backend/migrations/000062_abac_security_profiles.sql) —
  Authoritative schema for the ABAC layer described above.
