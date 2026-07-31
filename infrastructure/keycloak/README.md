# Keycloak realm provisioning — `uisce` realm + `semlayer-frontend` client

This directory holds a checked-in Keycloak realm export and a one-shot import
script so the OIDC flow that the frontend (`semlayer-frontend`) depends on is
reproducible across environments instead of being configured by hand in the
Keycloak Admin UI.

## What's in the realm

| Object | Purpose |
|---|---|
| Realm `uisce` | Dedicated realm (not `master`) for Uisce users. |
| Client `semlayer-frontend` | Public SPA client, `standardFlowEnabled=true`, PKCE (S256), `redirectUris=http://localhost:5173/auth/callback`. |
| Client scope `tenant-groups` | Custom optional scope that adds the `groups` claim (Keycloak group membership) to the ID token, access token, and userinfo. Added to `defaultDefaultClientScopes` so it's always emitted. |
| Realm groups `Uisce-Global-Admins`, `Uisce-Tenant-Admins`, `Uisce-Tenant-Users` | Seed groups for tenant-scoped authorisation. |
| Per-client protocol mappers | `tenant_id`, `operator_role` → top-level claims on the ID/access token (used by `AuthContext.tsx`). |
| Signing key | `rsa-generated`, RS256, 2048-bit (Keycloak-managed, rotates automatically). |

## Provisioning

```bash
# Defaults read from repo-root .env (KEYCLOAK_HOST, KEYCLOAK_ADMIN, etc.)
infrastructure/keycloak/import-realm.sh

# Preview without writing
infrastructure/keycloak/import-realm.sh --dry-run

# Explicit overrides
KEYCLOAK_HOST=kc.example.com KEYCLOAK_ADMIN=admin KEYCLOAK_ADMIN_PASS=**** \
  infrastructure/keycloak/import-realm.sh

# Force-overwrite an existing realm with state (DESTRUCTIVE)
infrastructure/keycloak/import-realm.sh --force
```

The script is idempotent — POST on first run, PUT on subsequent runs. It
authenticates as the `master` realm admin (`admin-cli` client, password
grant) and uses the Admin REST API.

**Safety:** if the realm on the server already contains `groups`, `users`,
`clients`, `identityProviders`, or `authenticationFlows`, the script refuses
to overwrite without `--force`. Keycloak's PUT is **replace, not merge** —
a blind PUT would lose any state that the export JSON doesn't capture
(`keycloakVersion`, `organizationsEnabled`, `clientProfiles`,
`authenticationFlows`, etc.).

## Current state (as of 2026-07-30)

The `uisce` realm on `https://100.84.50.65:8443` is **already fully
provisioned** by hand. It already has:

| Object | Value |
|---|---|
| Realm | `uisce`, enabled |
| Client | `semlayer-frontend` (public SPA, PKCE, redirect `http://localhost:5173/auth/callback`) |
| Client scope | `tenant-groups` with `tenant-group-mapper` (`claim.name=groups`, `id.token.claim=true`) |
| Client default scopes | includes `tenant-groups` (always emitted) |
| Protocol mappers on client | extra `groups` mapper (defence-in-depth) |

**Do not re-run the import against this server.** The realm has fields the
checked-in JSON does not capture, and a PUT would clobber them. This JSON
remains useful as a reference for a fresh Keycloak setup (e.g. CI, staging,
or a developer onboarding on a new cluster) where the realm doesn't exist
yet — `import-realm.sh` will POST cleanly.

## After provisioning

1. Update `frontend/.env.local`:
   ```
   VITE_OIDC_ISSUER=https://100.84.50.65:8443/realms/uisce
   VITE_OIDC_CLIENT_ID=semlayer-frontend
   VITE_OIDC_REDIRECT_URI=http://localhost:5173/auth/callback
   ```
2. Restart the dev server (`npm run dev` in `frontend/`).
3. Verify the client is registered: `https://${KEYCLOAK_HOST}:${KEYCLOAK_PORT}/admin/master/console/#/realms/uisce/clients`.
4. Create at least one user in the Admin UI (Users → Add user) and assign them
   to one of the seed groups so the `groups` claim is non-empty.

## Rotating the client secret (prod hardening)

The realm export sets `clientAuthenticatorType: client-secret` and ships a
placeholder secret (`CHANGE_ME_BEFORE_PROD_…`). For production:

1. Open Admin UI → Realm `uisce` → Clients → `semlayer-frontend` → Credentials → Regenerate.
2. Distribute the new secret via your secret store (not via git).
3. Or flip `publicClient` to `true` and drop the secret entirely (PKCE-only is
   the SPA-friendly pattern and is the default behaviour of the JSON shipped
   here).

## Troubleshooting

| Symptom | Likely cause |
|---|---|
| `403 Forbidden` from `POST /admin/realms` | `KEYCLOAK_ADMIN`/`KEYCLOAK_ADMIN_PASS` don't have realm-management permissions in `master`. |
| `409 Conflict` | Realm already exists but with different settings — script handles this via PUT, but only if the auth succeeds. |
| Frontend still says "Client not found" | Browser cached the old realm; hard refresh, and confirm `VITE_OIDC_ISSUER` ends with `/realms/uisce`. |
| `groups` claim missing from ID token | The user has no group membership; assign them a group in the Admin UI. |