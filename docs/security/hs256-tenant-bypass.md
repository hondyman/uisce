# HS256 tokens bypass issuer/tenant validation entirely

**Severity: Critical.** Anyone holding `JWT_SECRET` can mint a token claiming
to be any user, in any tenant, with `global_admin` role, and the backend
accepts it as fully authenticated. No per-tenant check applies to it.

## How this was found

While verifying an unrelated frontend change, `backend/cmd/devjwt` — a
checked-in CLI tool — was run using `JWT_SECRET` from `backend/.env`. It
produced a validly-signed HS256 token:

```json
{"roles": ["global_admin"], "tenant_ids": ["99e99e99-..."], "is_active": true, ...}
```

This token was never actually used against the running app (the frontend
requires a full OIDC session, not just a bearer token, so it didn't complete
that particular verification). But reading the backend's validation code
afterward confirms the token *would* have been accepted as fully legitimate
for any API call, for any tenant, with `global_admin` privileges.

## The two validation paths

Real Keycloak logins produce RS256 tokens. Two layers protect these
correctly:

1. `JWTManager.ValidateToken` (`backend/internal/services/security_manager.go:515`)
   verifies RS256 signatures against JWKS public keys (with `kid`-based key
   lookup, supporting rotation).
2. `ValidateIssuerTenant` (`backend/internal/services/idp_refresh.go:40`),
   wired into `AuthContextMiddleware`
   (`backend/internal/middleware/auth_context.go:117`), cross-checks the
   token's `iss` claim against an `IssuerRegistry` and rejects any
   `tenant_id`/`tenant_ids` claim the issuing IdP isn't actually registered
   for. This is the correct per-tenant-IdP isolation mechanism — a
   compromised or careless tenant IdP cannot assert a `tenant_id` belonging
   to a different tenant, because the registry ties each issuer to a fixed
   set of tenants it's trusted for.

**This part of the design is sound and is exactly the protection needed for
a multi-tenant, per-tenant-IdP architecture.**

The gap is the second token type the same validator accepts. Internal
HS256 tokens (service-to-service calls, impersonation, `devjwt` for local
dev) have no `iss` claim. `ValidateIssuerTenant` explicitly skips them:

```go
// idp_refresh.go:41-44
if claims.Issuer == "" {
    return nil, nil
}
```

with the comment: *"internal HS256 tokens... skip this check entirely...
they were never issued by an external IDP to begin with, so there's no
issuer registration to check against."* That reasoning is correct in
isolation, but the practical effect is that **any HS256 token signed with
the shared `JWT_SECRET` is trusted with whatever `tenant_id`, `tenant_ids`,
and `roles` it contains — including `global_admin` — with zero further
validation.**

## Why this is live risk, not just a design gap

`JWT_SECRET` is a single static value, shared across every use of it
(service-to-service auth, impersonation, and `devjwt`), and it sits in
plaintext in `backend/.env` on shared development machines. During the
session that found this, at least 7 concurrent Claude Code sessions had
filesystem access to that file simultaneously. Anyone or any process with
read access to it can mint an unrestricted `global_admin` token for any
tenant — a full authentication bypass, independent of how well-hardened the
RS256/per-tenant-IdP path is.

## Recommended remediation

1. **Restrict who/what can mint HS256 tokens signed with `JWT_SECRET`.**
   Currently both `cmd/devjwt` and `JWTManager.SignMapClaims` can produce a
   token trusted for anything, with no scoping between "this is a
   service-to-service token" and "this is a full user session."
2. **Stop using one shared secret for every HS256 use case.** A distinct
   secret (or scoped claim, itself independently verified) per intended use
   — service-to-service, impersonation, local dev — would mean a leak of
   one doesn't grant the others.
3. **Rotate `JWT_SECRET`.** It should be treated as exposed in any shared
   development environment where it has sat in a plaintext `.env` file
   accessible to multiple concurrent sessions/users.
4. Consider whether HS256 tokens should carry *any* tenant claim without an
   equivalent authority check — e.g., scoping internally-minted tokens to
   only the operations they're actually meant for (a specific service call,
   a specific impersonation session) rather than a free-form `tenant_ids`
   claim.

## Explicitly out of scope for this finding

The separate question of how `john.b@example.com` (and other real users)
get assigned to their tenant — via Keycloak user attributes vs. an
authoritative platform-controlled mapping — is a related but distinct
concern, tracked separately. `ValidateIssuerTenant`/`IssuerRegistry` already
correctly handles the case where a per-tenant IdP is registered; the
open item there is that no IdP is currently registered/linked for the
`uisce` realm's local users, so that protection path isn't exercised yet
in this environment, not that the mechanism itself is unsound.
