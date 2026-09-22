# Tenant-ID Resolution — Canonical Middleware Design

**Date:** 2026-09-20
**Status:** V1 and V2 both hotfixed 2026-09-20. Phase C (identity deprecation) deferred pending audit.
**Incident reference:** INCIDENT_REPORT_20260906 / hotfix 2026-09-07

---

## 1. Background

The 2026-09-06 incident exposed that the backend had **6+ independent tenant-ID resolution implementations** with different precedence orders. A security sweep on 2026-09-07 closed the most severe finding (the `SecurityContextFromRequest` vulnerability) via `security.ResolveTenantID`. Two other implementations were not fixed in that hotfix and remain vulnerable.

This document defines the canonical tenant-resolution rule and the migration plan for the remaining vulnerable implementations.

---

## 2. Current State Audit

### 2.1 The Canonical Rule Already Exists

`security.ResolveTenantID` (`internal/security/auth_context.go:51`) was added in the 2026-09-07 hotfix. It defines the correct behavior:

```go
// ResolveTenantID is the canonical tenant-resolution rule for this codebase
//
// Precedence:
//  1. If no override requested (requested == "") → auth.TenantIDs[0] from JWT
//  2. If override requested:
//     a. Caller is global_admin or global_ops → override honored
//     b. Override matches one of auth.TenantIDs → override honored
//     c. Otherwise → ("", false) — fail-closed
//
// No default tenant. No silent fallback to a guessed value.
func ResolveTenantID(auth AuthInfo, requested string) (string, bool)
```

This is **correct and must be preserved**. It is used by `SecurityContextFromRequest` (67 call sites) — the function that was the primary attack vector in the incident.

### 2.2 Vulnerable Implementations

#### V1: `rulefabric/getTenantID` — 25 call sites (CRITICAL)

**File:** `backend/internal/rulefabric/handler.go:1281–1293`

```go
func getTenantID(r *http.Request) (uuid.UUID, error) {
    tenantIDStr := auth.TenantIDs[0]   // JWT first ✓
    if tenantIDStr == "" {
        tenantIDStr = r.Header.Get("X-Tenant-ID")   // header second ✗
    }
    if tenantIDStr == "" {
        tenantIDStr = r.URL.Query().Get("tenant_id")  // query third ✗
    }
    ...
}
```

**Vulnerability:** The header and query-param fallbacks are accepted **without verifying they match the caller's JWT tenant**. Any authenticated user can send `X-Tenant-ID: <victim-tenant-uuid>` and operate as that tenant.

**Attack path:**
```bash
# Authenticate as tenant-A user
curl -H "Authorization: Bearer <jwt-for-tenant-A>" \
     -H "X-Tenant-ID: <tenant-B-uuid>" \
     https://api.example.com/api/rulefabric/...
# Now operates as tenant B despite holding a tenant-A JWT
```

**Severity:** CRITICAL — 25 call sites in production, no admin-gate, direct tenant spoofing.

**Status 2026-09-20:** HOTFIXED — rulefabric.getTenantID now delegates to
security.ResolveTenantForRequest. 25 call sites updated in one commit. Variant A
confirmed: admin override preserved. Non-admin spoof attempts (header != JWT tenant)
are now rejected fail-closed.

**Mechanism of initial misclassification:** V1 was initially marked "dead code"
because `grep -n 'rulefabric' cmd/server/main.go` returned empty. The actual
registration is transitive: `server/main.go → api.go:SetupRouter → api.go:1267
(rulefabric.RegisterRoutes)`. Grep of the entrypoint alone missed the sub-router
mount. The `route-tree.sh` script in `scripts/refactor/route-tree.sh` encodes
the correct audit discipline: transitive walk from SetupRouter, not grep of main.go.

---

#### V2: `WithTenantContext` middleware — all routes via api.go:899 (CRITICAL)

**File:** `backend/internal/middleware/tenant_context.go:16–34`

```go
func WithTenantContext(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        tenantIDStr := r.Header.Get("X-Tenant-Id")   // header FIRST ✗

        // identity.TenantIDFromContext is set from JWT by AuthContextMiddleware
        // (runs at api.go:880, before WithTenantContext at api.go:899)
        if actorTenantID, ok := identity.TenantIDFromContext(r.Context()); ok {
            tenantIDStr = actorTenantID              // JWT fallback ✓
        }

        if tenantIDStr != "" {
            if _, err := uuid.Parse(tenantIDStr); err == nil {
                ctx := db.WithTenantContextToCtx(r.Context(), tenantIDStr)
                next.ServeHTTP(w, r.WithContext(ctx))
                return
            }
        }
        next.ServeHTTP(w, r)   // passes through without tenant context
    })
}
```

**Vulnerability:** `X-Tenant-Id` is checked **before** the JWT fallback. If a non-admin sends `X-Tenant-Id: <other-tenant-uuid>`, it is accepted as a valid UUID without any check that the caller is authorized for that tenant.

**Note:** When `X-Tenant-Id` is absent, the JWT fallback (line 20–22) is safe because `identity.TenantIDFromContext` is set from the validated JWT by `AuthContextMiddleware`. The vulnerability is **only** when the header is present and non-empty.

**Severity:** CRITICAL — mounted on all routes via `api.go:899`, affects every handler that relies on `db.WithTenantContextToCtx` for RLS.

**Status 2026-09-20:** HOTFIXED — `WithTenantContext` now delegates to `security.ResolveTenantForRequest`, which calls the canonical `ResolveTenantID`. Header accepted only if it matches JWT tenant or caller is global_admin. Non-admin spoof attempts fail-closed.

---

### 2.3 Already-Safe Implementations

#### S1: `SecurityContextFromRequest` — 67 call sites (SAFE, post-2026-09-07)

**File:** `backend/internal/handlers/security_context.go:17–115`

Uses `security.ResolveTenantID` internally. Correctly rejects unauthorized tenant overrides. The primary attack vector from the 2026-09-06 incident.

#### S2: `metadata/bo_service.go` (if exists) — Phase 3 candidate

See Phase 3 plan.

---

## 3. Frontend Tenant-Header Usage

The frontend sends `X-Tenant-ID` in approximately 30 locations. All legitimate usages send the **user's own tenant ID** (from the user's JWT/claims, stored in context), not a hardcoded value:

```typescript
// Normal usage — sends user's own tenant from context
headers.set('X-Tenant-ID', parsed.id);              // MetadataContext.tsx
...(tenant?.id && { 'X-Tenant-ID': tenant.id }),   // CopilotPanel, MDMBreakManagement, etc.

// All usages follow this pattern — no cross-tenant header injection found
```

**Known exception — TemplatesTab.tsx (hardcoded `tenant-1`):**
`frontend/src/features/semantic-playground/components/TemplatesTab.tsx` sends hardcoded
`'X-Tenant-ID': 'tenant-1'` in 7 locations. This is development/test data, not a
production workflow. After the V2 hotfix, these requests will fail if sent by a
non-admin user whose JWT tenant is not `tenant-1`. Should be migrated to use the
actual user's tenant context.

**Impact of V2 hotfix:** Normal users (their JWT tenant matches their header) are
unaffected. Admin users: unchanged (global admin can specify any tenant).
Users attempting to spoof another tenant: rejected fail-closed.

---

## 4. Canonical Middleware Design

### 4.1 New Function: `RequireTenant`

```go
// package: internal/middleware

// RequireTenant extracts and validates the tenant for this request.
//
// Precedence (identical to security.ResolveTenantID):
//   1. No X-Tenant-ID header → use auth.TenantIDs[0] from JWT (caller's own tenant)
//   2. X-Tenant-ID header present:
//      a. Caller is global_admin/global_ops → honor any valid-UUID header value
//      b. Header matches one of auth.TenantIDs → honor it
//      c. Otherwise → returns ("", ErrTenantNotAuthorized)
//
// Returns error (do NOT return a wildcard or empty-tenant fallback).
// Callers must handle the error and return 401/403.
//
// For handlers that legitimately need cross-tenant access (admin tooling):
//   use RequireAnyTenant instead, which requires isGlobalAdmin == true.
func RequireTenant(r *http.Request) (tenantID string, err error)
```

### 4.2 Dev-Mode Flag

`X-Tenant-ID` header acceptance for non-JWT tenants (i.e., unauthenticated dev shortcuts) is controlled by `DEV_ALLOW_TENANT_HEADER=true`. This is **never** set in production. When `false` (default), the behavior is exactly as 4.1 above.

### 4.3 `RequireAnyTenant` for Admin Tools

Handlers that need to operate without a tenant context (e.g., `/health`, `/metrics` that aggregate across tenants) should use `RequireAnyTenant` which returns `nil` error without setting any tenant. This is explicitly different from `RequireTenant` which always requires a tenant.

---

## 5. Migration Plan

### Phase A: Fix `WithTenantContext` (V2)

**File:** `backend/internal/middleware/tenant_context.go`

**Change:** Replace the header-first-then-JWT logic with `security.ResolveTenantID` using the `security.AuthInfo` already set in the request context by `AuthContextMiddleware`.

**Scope:** All routes via `api.go:899`. Affects every handler that reads tenant from `db.WithTenantContextToCtx`.

**Status 2026-09-20:** DONE — committed as part of hotfix. `security.ResolveTenantForRequest` added; `WithTenantContext` now delegates to it. Unit tests added.

---

### Phase B: Fix `rulefabric/getTenantID` (V1)

**File:** `backend/internal/rulefabric/handler.go:1281–1293`

**Change:** Replace with a call to `security.ResolveTenantForRequest`. Signature unchanged
(`(uuid.UUID, error)`) — UUID conversion happens inside the wrapper.

**Scope:** 25 call sites in `rulefabric/handler.go` and `rulefabric/bo_policy_handler.go`.

**Status 2026-09-20:** DONE — committed as part of hotfix. `getTenantID` now
delegates to `security.ResolveTenantForRequest`. Unit tests added.

---

### Phase C: Deprecate `identity.TenantIDFromContext`

**Status 2026-09-20:** PARTIALLY DONE — `WithTenantContext` no longer reads
`identity.TenantIDFromContext`. The `identity` package is still used in
`AuthContextMiddleware` (sets actor context) and should be audited before removal.

---

### Phase 4 (deferred): TemplatesTab + RegisterTemplateRoutes dead-wire

**Finding:** `frontend/src/features/semantic-playground/components/TemplatesTab.tsx`
makes 6 raw `fetch` calls to `/api/semantic/templates` with `X-Tenant-ID: tenant-1`
set in each request. `backend/internal/api/template_handlers.go:648` defines
`Server.RegisterTemplateRoutes(router chi.Router)` but `api.go` never calls it —
the routes are not mounted. Result: all 6 calls return 404.

**Classification:** Not a security risk (endpoint doesn't exist), but broken
functionality — a user-facing tab making dead requests.

**Options:**
1. Wire the routes: mount `Server.RegisterTemplateRoutes` in `api.go` and
   fix the frontend to send the context tenant instead of hardcoded `tenant-1`.
2. Delete both halves: remove `TemplatesTab.tsx` API calls and
   `Server.RegisterTemplateRoutes` definition.

**route-tree.sh validation:** `RegisterTemplateRoutes` should appear as UNMOUNTED
once the definition-line false negative is fixed. Use it as a regression test
for the script.

---

## 6. Unit Test Coverage

Every function in `internal/middleware/tenant_context.go` and `internal/security/auth_context.go` must have table-driven tests covering:

| Test | Coverage |
|------|----------|
| `RequireTenant` — no header, JWT has tenant | returns JWT tenant |
| `RequireTenant` — header matches JWT tenant | returns header value |
| `RequireTenant` — header does NOT match JWT, non-admin | returns error |
| `RequireTenant` — header does NOT match JWT, global admin | returns header value |
| `RequireTenant` — no JWT, no header | returns error (fail-closed) |
| `RequireTenant` — invalid UUID in header | returns error |
| `RequireTenant` — empty header, JWT has multiple tenants | returns TenantIDs[0] |
| `RequireAnyTenant` — global admin, no tenant context | returns nil error |
| `RequireAnyTenant` — non-admin, no tenant context | returns error |

---

## 7. Open Questions

1. **LEGACY system calls:** Do any non-HTTP entry points (Temporal workers, stream loaders, CLI tools) call `WithTenantContext` or `getTenantID`? These should be reviewed separately — they operate with service-level credentials, not user JWTs, and may have legitimate cross-tenant requirements.

2. **`DEV_ALLOW_TENANT_HEADER` scope:** Should this be a global env var or per-route middleware configuration? Recommendation: global env var, `false` by default, logged when `true`.

3. **Frontend impact:** Confirm that all frontend `X-Tenant-ID` usage sends the caller's own tenant ID (from their JWT). If any workflow sends a different tenant ID for legitimate cross-tenant operations (e.g., global admin panel), that user must be a global admin for the override to be accepted post-fix.
