# JWT Remediation & Trust Inversion Fix

**Date:** 2026-09-03
**Owner:** @eganpj (Platform Security)

## Executive Summary
Our current JWT implementation suffers from a trust inversion: self-asserted tokens signed with a symmetric key (HS256) are granted highly privileged access, bypassing the stricter verification applied to asymmetric tokens. The backend currently trusts the token's `roles` and `tenant_ids` claims as a source of truth for authorization, allowing anyone with read access to the symmetric secret to forge tokens and elevate privileges (e.g., to `global_admin`).

## Core Principle
**A token proves *who authenticated*. Your database proves *what they're allowed to do*.**
Token claims will become a consistency check (mismatch = reject), rather than the source of truth for authorization.

## Findings & Status Matrix

| Finding, as discovered | Status | Details / Tracking |
|---|---|---|
| Shared-secret HS256 forgery (original Critical) | **Closed on main API bearer dispatch; open on external consumers pending audit** | Alg-gate rejects HS256 on main API; `semantic-rules-api`, `notifications-service`, `validation-service`, `rule-engine-service`, and `trace_auth_middleware` retain shared-secret HMAC paths pending migration/network gating. |
| Fail-open middleware | Closed | Structural dispatch by dot-count (1-dot / 2-dot / default), 401 on all invalid credentials. |
| Unauthenticated header impersonation (390 handler sites) | Closed at the boundary | Strip-first deletes `X-User-ID`, `X-Tenant-ID`, `X-Real-Admin-ID`, `X-Impersonation-*` before routing. Handler migration to `Actor(r)` is Phase 3. |
| SSE route bypassing all middleware | Closed | `rootMux.With(AuthContextMiddleware)` added to `/api/catalog/scan/stream`. Handler rejects missing auth context with 401. |
| Legacy impersonation → `global_admin` | Closed | Tokens missing roles now rejected with 401 Unauthorized. |
| Flat `kid` map / `iss`-spoof escalation | **Open — hard gate before first tenant IdP registration** | Namespace keys as `map[issuer]map[kid]*rsa.PublicKey`. Blocks BYOI multi-tenancy. |
| `aud` enforcement | Warn-mode | Promotes to hard rejection once auth logs confirm clean audiences and frontend switches to access tokens. |
| Impersonation server-side session validation | Tracked Phase 3 | Lookup `SessionID` in session store / audit table inside `ValidateImpersonationToken`. Owner: Platform Security. |
| Compliance-engine accepting user JWTs | Interim acceptable | Service is internal-only. Phase 3 moves to S2S tokens/mTLS. |


## Remediation Plan (This Week)

### Phase 1: Immediate Containment
1. **Rotate `JWT_SECRET`**: Rotate the `JWT_SECRET` in Infisical and redeploy to invalidate any potentially forged tokens.
2. **Implement `alg` Gate & `iss` Rejection**:
   - `JWTManager.ValidateToken` updated to reject `alg == HS256` unless `APP_ENV=development`.
   - Reject any token with an empty `iss` claim.
   - Required and verified the `aud` claim.
   - `ValidateIssuerTenant` updated to remove the `claims.Issuer == ""` early return.
3. **Auth-Event Logging**: Log `alg` and `iss` on every auth event in `AuthContextMiddleware` to discover any legitimate HS256 callers in production.

### Phase 2: Environment & Dev Hygiene
1. **Remove plaintext `.env` files**: Delete `backend/.env` from shared machines.
2. **Migrate to Infisical**: Move local development to use `infisical run -- <cmd>` to inject secrets at runtime.
3. **Audit Prod History**: Grep auth logs for HS256/empty-`iss` tokens in production history. If found, escalate to a security incident.

### Phase 3: Server-Side Authorization (Upcoming)
1. **Update `AuthContextMiddleware`**: 
   - After signature validation, resolve authorization server-side: `(iss, sub) → user record → effective roles + tenants`.
   - Token claims will only act as a consistency check.
   - This change definitively mitigates the "mint global_admin" vulnerability class.

3. **Server-Side Impersonation Session Validation**:
   - Validate `SessionID` against `platform_admin_audit` / active sessions store (active, unrevoked, unexpired) inside `ValidateImpersonationToken`.
   - Re-derive operator `RealRoles` server-side from `app_user` / role tables rather than trusting claims in the token.
   - Owner: Platform Security (Phase 3).

## Epic: Uisce Support Access & Delegation
Staff should have zero ambient tenant membership. We will implement a delegation model:
1. Staff authenticates to the uisce realm (MFA).
2. Requests tenant access via a support console (tenant + ticket/case ID).
3. Backend checks a server-side grant table (staff identity, tenant, contract tier, scope, expiry).
4. Mints a short-lived delegated token with `act` (RFC 8693 token exchange semantics).
5. All requests are audited as `(actor, subject, tenant, operation)`.

## Epic: Tenant BYOI Enhancements
1. **Claim = binding check**: Accept a registered tenant IdP's token only if the asserted `tenant_id` matches the registered tenant.
2. **JIT Provisioning**: First login creates the user mapped exclusively to that tenant.
3. **Secret Isolation**: Per-tenant IdP client secrets will live in Infisical (never a shared `IDP_CLIENT_SECRET`).

## Compliance Engine Microservice Exposure & Auth
- **Exposure**: Internal service (`backend/services/compliance-engine/cmd/compliance-engine/main.go`).
- **Auth**: Uses `jwtmiddleware.NewJWTMiddleware(...)` backed by `libs/jwt-middleware` (strict HMAC-SHA256 using `JWT_SECRET`, failing closed on invalid/missing tokens).
- **Remediation / Phase 3 Item**: It operates on its own router and validates bearer tokens directly via the shared HMAC secret rather than authenticating S2S requests from the main backend. In Phase 3, this service will be locked down behind internal Kubernetes NetworkPolicy and transition to service-to-service mTLS or dedicated S2S tokens issued by the backend platform.
