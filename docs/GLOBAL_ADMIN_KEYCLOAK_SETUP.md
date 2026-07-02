# Global Admin Setup — `john.b@example.com` via Keycloak Group-to-Role Inheritance

## TL;DR

Make **john.b@example.com** a **Global Admin** with cross-tenant visibility using the standard enterprise **Group-to-Role inheritance chain** in Keycloak. The application code is already wired for this pattern — only the Keycloak realm configuration is missing.

```bash
# Idempotent — safe to re-run.
./scripts/setup-keycloak-realm.sh

# Optional: also provision the john.b user account and add him to Uisce-Global-Admins.
BOOTSTRAP_USERS=1 JOHN_B_PASSWORD='A-strong-password-here' \
  ./scripts/setup-keycloak-realm.sh
```

Once it completes, log in at `http://localhost:5173` as john.b — the **AccessContext** will route him to `/api/tenants/all` (all tenants), and the backend's `AuthContextMiddleware` will set `X-Is-Core-Admin: true` on every request.

---

## Background — Why This Pattern?

`john.b@example.com` is associated with the `global_admin` role through a
**hierarchical inheritance chain** rather than direct assignment:

```
john.b@example.com
        │  (member of)
        ▼
Uisce-Global-Admins         ←─── Keycloak Group
        │  (carries Realm Role)
        ▼
global_admin                ←─── Keycloak Realm Role
```

When john.b authenticates, Keycloak processes his group memberships and
automatically resolves the roles associated with those groups. Because the
role is mapped to the group, Keycloak includes `global_admin` in his JWT's
`realm_access.roles` claim — and the **Go backend only ever asks "does this
token contain the `global_admin` role?"**, so it's immune to changes in the
Keycloak group structure.

### What john.b's JWT will look like

```json
{
  "sub": "113d...",
  "email": "john.b@example.com",
  "realm_access": {
    "roles": [
      "global_admin",
      "uma_authorization",
      "..."
    ]
  },
  "groups": [
    "/Uisce-Global-Admins"
  ],
  "uisce_metadata": {
    "operator_role": "global_admin"
  }
}
```

The `groups` claim is optional for the role-detection path — the frontend
and backend both recognise `global_admin` in `realm_access.roles` directly.
The `groups` claim is kept as defence-in-depth (see the backend's
`security_manager.go:706-720` fallback mapping).

---

## The Three-Step Checklist (matches Keycloak UI)

### 1. Group Mapping — Done by the script (idempotent)

The script creates `Uisce-Global-Admins` if it does not yet exist. If you've
already added john.b to the group in the Keycloak UI, that membership is
preserved — the script only adds the group if missing.

Equivalent Keycloak UI path: **Realm → Groups → Uisce-Global-Admins → Members → Add user**

### 2. Role Assignment — Done by the script (idempotent)

The script maps the `global_admin` Realm Role onto the `Uisce-Global-Admins`
group via the Admin REST API:

```
POST /admin/realms/{realm}/groups/{group-id}/role-mappings/realm
[{"id": "<role-uuid>", "name": "global_admin"}]
```

Equivalent Keycloak UI path: **Realm → Groups → Uisce-Global-Admins → Role mapping → Assign role → global_admin**

### 3. Token Mapping — Already done by the existing script

`setup-keycloak-realm.sh` (sections 5 and 5b) ensures the `roles` and `groups`
client scopes are in `default-default-client-scopes` and that a working
`oidc-group-membership-mapper` is present, so the `realm_access.roles` and
`groups` claims are emitted in every ID/access token.

Equivalent Keycloak UI path: **Realm → Client scopes → roles → Mappers → User Realm Role** (already present by default in Keycloak).

---

## What the Script Provisions

| Object | Type | Keycloak UI Path | Created by Section |
|--------|------|------------------|-------------------|
| `uisce` | Realm | (root) | pre-existing section 3 |
| `semlayer-frontend` | Client (SPA, PKCE S256) | Realm → Clients | pre-existing section 4 |
| `roles` | Default Client Scope | Realm → Client scopes | pre-existing section 5 |
| `groups` | Default Client Scope + Mapper | Realm → Client scopes | pre-existing section 5b |
| `global_admin` | Realm Role | Realm → Roles | **new section 6** |
| `global_ops` | Realm Role | Realm → Roles | **new section 6** |
| `professional_services` | Realm Role | Realm → Roles | **new section 6** |
| `helpdesk` | Realm Role | Realm → Roles | **new section 6** |
| `Uisce-Global-Admins` | Group | Realm → Groups | **new section 7** |
| `Uisce-Professional-Services` | Group | Realm → Groups | **new section 7** |
| `Uisce-Helpdesk` | Group | Realm → Groups | **new section 7** |
| `global_admin → Uisce-Global-Admins` | Role→Group map | Group → Role mapping | **new section 8** |
| `john.b@example.com` | User | Realm → Users | **new section 10** (opt-in) |
| `john.b → Uisce-Global-Admins` | Group membership | User → Groups | **new section 10** (opt-in) |

---

## Running the Script

### Production / CI (no user bootstrap)

```bash
./scripts/setup-keycloak-realm.sh
```

Creates the realm, client, scopes, roles, groups, and role→group mappings.
Does **not** create john.b — leave human account management to your IdP
(Azure AD / Okta / federated LDAP).

### Dev / Staging (with user bootstrap)

```bash
BOOTSTRAP_USERS=1 \
  JOHN_B_PASSWORD='PickAStrongPassword!' \
  ./scripts/setup-keycloak-realm.sh
```

Creates john.b@example.com (if missing) with a known password and adds him
to `Uisce-Global-Admins`. Without `JOHN_B_PASSWORD`, a fresh 24-char
base64 password is generated and **printed once** to stdout — store it
securely.

If john.b already exists, his password is **not** rotated. If he's already
a member of `Uisce-Global-Admins`, that membership is **not** changed. The
script is fully idempotent.

---

## Verification

### 1. Quick programmatic checks (printed by the script)

The script prints a `cat <<EOF` block at the end with five verification
commands. The most useful one is **#5 — decode john.b's JWT**:

```bash
ACCESS_TOKEN=$(curl -sk -X POST \
  'https://100.84.50.65:8443/realms/uisce/protocol/openid-connect/token' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'grant_type=password' \
  -d 'client_id=admin-cli' \
  -d 'username=john.b@example.com' \
  -d 'password=YOUR_PASSWORD_HERE' \
  | jq -r .access_token)

echo "${ACCESS_TOKEN}" | awk -F. '{print $2}' | tr '_-' '/+' | base64 -d 2>/dev/null | jq .
```

**Expected output:**

```json
{
  "sub": "113d...",
  "email": "john.b@example.com",
  "realm_access": {
    "roles": [
      "global_admin",
      "uma_authorization",
      "..."
    ]
  },
  "groups": ["/Uisce-Global-Admins"]
}
```

If `realm_access.roles` does **not** contain `global_admin`, the group→role
mapping is missing — re-run the script (section 8 is idempotent).

### 2. Browser end-to-end (best test)

1. Open `http://localhost:5173` in a private/incognito window.
2. Click "Sign in" — you should be redirected to the Keycloak login page
   at `https://100.84.50.65:8443/realms/uisce/protocol/openid-connect/auth`.
3. Sign in as `john.b@example.com` with the bootstrap password.
4. After the OIDC callback, the UI should load with:
   - The scope badge in the top nav showing "All Tenants" (or similar global text).
   - The Tenant Picker showing **all** tenants — not just one.
5. Open the browser DevTools console — the OIDC debug log should print:

   ```js
   [OIDC] user loaded {
     sub: '113d...',
     groups: [ '/Uisce-Global-Admins' ],
     operator_role: undefined,           // (set by backend, not IdP)
     realm_access_roles: [ 'global_admin', 'uma_authorization', ... ],
     uisce_metadata: undefined           // (optional custom claim)
   }
   ```

6. Decode the `id_token` at https://jwt.io to visually confirm the
   `global_admin` claim is present.

### 3. Backend smoke test

Once john.b is logged in, every API request carries
`Authorization: Bearer <jwt>` and `X-Is-Core-Admin: true` (the header set by
`backend/internal/middleware/auth_context.go:228-230`). Verify with:

```bash
# From the browser DevTools "Network" tab, copy any /api/tenants/all response.
# The response should contain the full tenant tree, not a single-tenant projection.
```

The frontend `AccessContext` (line 163) routes john.b to
`/api/tenants/all` because his `accessLevel === 'platform_operator'`
(determined by `global_admin` in roles / `Uisce-Global-Admins` in groups).

---

## Adding the Next Global Admin

### Option A — Via the script (idempotent re-run)

Add more users by extending the `BOOTSTRAP_USERS=1` flow or by issuing
direct Admin API calls. The script's role/group infrastructure is
already in place, so you only need to:

```bash
# 1. Create user (or skip if federated from Azure AD / Okta)
curl -sk -X POST \
  -H "Authorization: Bearer ${ADMIN_TOKEN}" \
  -H 'Content-Type: application/json' \
  -d "$(jq -n \
    --arg email 'jane.smith@example.com' \
    '{username: $email, email: $email, enabled: true, emailVerified: true, \
      credentials: [{type: "password", value: "TEMP-change-me", temporary: true}]}')" \
  "https://100.84.50.65:8443/admin/realms/uisce/users"

# 2. Find the user's UUID and the Uisce-Global-Admins group's UUID
USER_ID=$(curl -sk -H "Authorization: Bearer ${ADMIN_TOKEN}" \
  "https://100.84.50.65:8443/admin/realms/uisce/users?exact=true&username=jane.smith@example.com" \
  | jq -r '.[0].id')
GROUP_ID=$(curl -sk -H "Authorization: Bearer ${ADMIN_TOKEN}" \
  "https://100.84.50.65:8443/admin/realms/uisce/groups?exact=true&name=Uisce-Global-Admins" \
  | jq -r '.[0].id')

# 3. Add the user to the group
curl -sk -X PUT \
  -H "Authorization: Bearer ${ADMIN_TOKEN}" \
  "https://100.84.50.65:8443/admin/realms/uisce/users/${USER_ID}/groups/${GROUP_ID}"
```

### Option B — Via the Keycloak UI

1. **Realm → Users → Add user** — create the user.
2. **Realm → Groups → Uisce-Global-Admins → Members → Add user** — add them to the group.
3. Done. The user logs in and Keycloak emits `global_admin` in their token.

The script does not need to be re-run.

### Option C — External federation (Azure AD / Okta)

For federated users, do **not** create them in Keycloak. Instead:

1. Configure the IdP's group attribute (`UISCE-GLOBAL-ADMIN`) to be sent in the SAML/OIDC claim.
2. In Keycloak, set up a **Group mapper** or use a **Role mapper** on the
   IdP broker to map the external group to the internal `Uisce-Global-Admins`
   group (or directly to the `global_admin` role).
3. The user's token still carries `realm_access.roles: ["global_admin", ...]`
   — the application code is unchanged.

---

## Architecture: Why the Backend Doesn't Care About Groups

The Go backend's `AuthContextMiddleware` (`backend/internal/middleware/auth_context.go`)
sets `isGlobalAdmin = true` whenever any of these are present in the JWT:

| Source | File / Line |
|--------|-------------|
| `realm_access.roles` includes `global_admin` | `security_manager.go:673-687` + `auth_context.go:215-218` |
| `realm_access.roles` includes `global_ops` | `auth_context.go:215-218` |
| `operator_role` claim is `global_admin` / `global_ops` | `auth_context.go:130-138` |
| `groups` claim contains a name matching `/uisce[-_ ]?global[-_ ]?admins?/i` | `security_manager.go:706-720` (defence-in-depth fallback) |

When `isGlobalAdmin` is true:

- The response header `X-Is-Core-Admin: true` is set on every request
  (`auth_context.go:228-230`).
- `security.BuildContext` (`security.go:98-109`) **bypasses the tenant
  filter**, so datasource lookups succeed even when `TenantIDs` is empty.
- The frontend `AccessContext.tsx:163` routes to `/api/tenants/all` and
  shows the cross-tenant picker.

You can change the Group-to-Role mapping in Keycloak (e.g. move global
admins to a group called `uisce-platform-root`), rename the group, or
switch to a federated IdP — **no backend code changes are required**.

---

## Troubleshooting

| Symptom | Cause / Fix |
|---------|-------------|
| `❌ Role 'global_admin' is NOT mapped onto group 'Uisce-Global-Admins'` | Re-run the script — section 8 is idempotent and will create the mapping if missing. |
| JWT has `global_admin` in `roles` but not in `realm_access.roles` | The `roles` client scope is missing. Re-run the script — section 5 will re-add it to `default-default-client-scopes`. |
| JWT has neither claim | The user's ID token is being issued by a different realm. Check `frontend/.env.local` — `VITE_OIDC_ISSUER` must match the `uisce` realm URL. |
| John can log in but the UI still shows only one tenant | The frontend's `AccessContext` is not detecting the role. Open DevTools → Console; the OIDC event log should print `realm_access_roles: ['global_admin', ...]`. If it does, but the UI still hides tenants, hard-reload (Cmd+Shift+R) to clear the stale `localStorage` cached user object. |
| `BOOTSTRAP_USERS=1` re-runs reset john.b's password | This is **not** the case — section 10 only sets the password on initial user creation; subsequent runs skip the credentials block entirely. |
| Need to rotate john.b's password manually | Keycloak UI: **Realm → Users → john.b → Credentials → Reset password**. |

---

## Related Documentation

- `docs/AUTH_KEYCLOAK_SETUP.md` — Realming & client provisioning (sections 1–5 of the script).
- `docs/MULTI_TENANT_AUTH.md` — JWT claim contract & multi-tenant routing.
- `backend/internal/services/security_manager.go` — JWT claim extraction (`realm_access.roles`, `groups`, `operator_role`).
- `backend/internal/middleware/auth_context.go` — `IsGlobalAdmin` detection and `X-Is-Core-Admin` header injection.
- `backend/internal/security/security.go` — `BuildContext` bypass logic for global admins.
- `frontend/src/contexts/AuthContext.tsx` — Frontend role detection (group-name regex + `realm_access.roles`).
- `frontend/src/contexts/AccessContext.tsx` — Frontend routing to `/api/tenants/all` for platform operators.

---

**Last verified:** 2026-07-01  
**Status:** ✅ Idempotent, audit-ready, federation-ready