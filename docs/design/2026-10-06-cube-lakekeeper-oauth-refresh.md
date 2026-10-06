# Track B — Lakekeeper OAuth refresh for StarRocks Iceberg catalog

**Status:** B0 approved with B1 implementation  
**Date:** 2026-10-06  
**Depends on:** Track A merged (#411). Live catalog DDL requires explicit operator go (B2).

## Problem

CUBE-2.5 lab cold path mounts Lakekeeper as StarRocks external catalog `lakekeeper_iceberg` with a **static** `iceberg.catalog.token` from Keycloak client `uisce-provisioner`. Token TTL is ~1h. StarRocks 3.3.22 `iceberg.catalog.oauth2.credential` auto-refresh is broken in this stack, so after expiry `ApplyCold` fails against Lakekeeper REST.

## Decision

**Sidecar refresher** (not native oauth2.credential):

1. Mint access token via Keycloak `client_credentials` (reuse `iceberg.TokenManager`).
2. `ALTER CATALOG <name> SET PROPERTIES ("iceberg.catalog.token" = '…')`.
3. Run **eagerly** at the start of `starRocksColdWriter.ApplyCold` when `CUBE_ICEBERG_TOKEN_REFRESH` is enabled.
4. Fail closed if mint or ALTER fails (do not proceed to CREATE DATABASE / CTAS with a stale token).

Honest non-claim: this does not fix StarRocks’ oauth2.credential path; it is the durable workaround until a SR version/path works.

## Env

| Variable | Role |
|----------|------|
| `CUBE_ICEBERG_TOKEN_REFRESH` | `1` / `true` enables refresh (default off for hermetic tests) |
| `CUBE_ICEBERG_CATALOG` | Catalog name (lab: `lakekeeper_iceberg`) |
| `LAKEKEEPER_TOKEN_URL` | Keycloak token endpoint |
| `LAKEKEEPER_CLIENT_ID` | Default `uisce-provisioner` |
| `LAKEKEEPER_CLIENT_SECRET` | From Infisical / `.env` only — never chat or compose literals |

## ALTER shape (confirm on lab before B2)

```sql
ALTER CATALOG `lakekeeper_iceberg`
SET PROPERTIES ("iceberg.catalog.token" = '<redacted>');
```

Inventory before apply:

```sql
SHOW CREATE CATALOG lakekeeper_iceberg;
SHOW CATALOGS;
```

## B2 lab checklist (after explicit go)

1. Present redacted ALTER matching live `SHOW CREATE CATALOG`.
2. Set refresh env on API/worker; secret from Infisical.
3. Trigger Deploy/refresh (or one-shot Ensure) so ApplyCold mints + ALTERs.
4. Receipt: Temporal `CubeCompleteDualCommit` and/or `SELECT COUNT(*)` on `lakekeeper_iceberg.cubes.…` for CUBE-2.5 cube.
5. Optional negative: stale token → clear auth error → refresh → success.

## Non-goals

- Fixing SR oauth2.credential upstream.
- Per-tenant `ivy_t_*` audit catalogs (unless the same helper is reused later).
- extract-N (Track C), pagestudio delete (Track D).
