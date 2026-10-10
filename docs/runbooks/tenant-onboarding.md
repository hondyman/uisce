# Creating a tenant from parameters

```
uisce-tenant create --name "XYZ Investments" --region "US East" --product orm --label ABC
```

registers the tenant in the region, registers the ORM product for it, creates its database
`abc_orm` (label + product, lower case) on the region's Postgres cluster, builds the structure from
the gold copy's scan (ADR-050), seeds the reference rows, and activates the tenant. Anything you
leave out, the command asks for, with the allowed values, and it shows the plan and asks before it
creates anything.

## Running it

The CLI is a client of the admin API, so it runs from anywhere that can reach the backend. It signs
in as a Keycloak service account that is a **global administrator**:

| Variable | Meaning |
|---|---|
| `UISCE_URL` | the backend, e.g. `https://uisce.example.com` |
| `UISCE_TOKEN_URL` | the realm's token endpoint |
| `UISCE_CLIENT_ID` | the service account |
| `UISCE_CLIENT_SECRET_FILE` | a file only the operator can read (or `UISCE_CLIENT_SECRET`) |

```
uisce-tenant create [--name N] [--code C] [--region R] [--product P] [--label L]
                    [--instance I] [--yes] [--no-prompt] [--no-wait] [--timeout 60m]
uisce-tenant status --workflow <id>
```

- `--region` takes a code (`us-east-1`) or a name (`US East`); an ambiguous name asks which.
- `--no-prompt` never asks: it fails and lists what is missing. It needs `--yes` to create. Use both
  in a pipeline.
- Exit codes: 0 done, 1 provisioning failed and was rolled back, 2 request incomplete or invalid,
  3 timed out waiting, 4 not authorized, 5 bad usage or configuration, 6 other API error.
- Running the same command again finds the run it already started (the run id is derived from the
  request). A tenant that finished is a 409, not a second tenant.

The same checks are an API: `POST /api/system/tenants/provision/describe` takes a partial request
and returns what is missing, what is invalid (with the allowed values), and, when complete, the plan
and the normalized request. `POST /api/system/tenants/provision` takes the request and starts the run.

## Before the first run: prerequisites

Two read-only checks report what is missing. Both exit non-zero when a check fails.

**The Postgres host** (run as the administrator role the worker connects with, `DB_USER`):

```
psql "postgresql://$DB_USER@100.84.50.65:5432/postgres" -X -v ON_ERROR_STOP=1 \
  -v database=abc_orm -v role_group="$TENANT_DB_ROLE_GROUP" \
  -f backend/db/verify/tenant_cluster_preflight.sql
```

It checks that the role can create databases and roles, that no other database on the cluster lets
PUBLIC connect (otherwise the isolation probe fails the run after the database exists; the fix is
`scripts/harden-tenant-cluster.sh`), that the role group exists, that the name is free, and that
`pg_hba.conf` has a `hostssl` rule.

**Alpha** (the control database):

```
psql "$ALPHA_URL" -X -v ON_ERROR_STOP=1 -v app=orm -v region=us-east-1 \
  -f backend/db/verify/tenant_template_preflight.sql
```

It checks that exactly one gold-copy datasource is marked as the template for the app, that the gold
copy holds the product, that the product is active, and that the region has a cluster.

The worker must also be configured for the cluster it administers: `DB_HOST`, `DB_PORT`, `DB_USER`,
`DB_PASS` (all four; nothing is defaulted), the secrets provider, and optionally `DB_SSLMODE` and
`TENANT_DB_ROLE_GROUP`.

## Regions

`region_postgres_cluster` (migration `20261227_003`) maps a region to its cluster. It holds a host
and port and **no credential**. A region with no active row has no cluster and is refused. The
worker holds the administrator credential of one cluster; if a region names another cluster, the
run is refused before anything is created (`AssertRegionCluster`), so a database is never made on a
host nobody chose. To add a region: add its `region_config` row and its cluster row, and run a
worker that administers that cluster.

## What a failure leaves behind

The run compensates in reverse, for what it created: the role and binding, the cloned products, the
Lakekeeper namespace, the database (dropped even if a connection is still open), then the tenant and
instance rows. A name that is already taken (a database of that name exists on the cluster) is
refused first, before anything is created. If a compensation itself fails the result is
`rollback_incomplete` and the database may remain; the same name is then refused as taken until an
operator drops it.

## Current limits

- **One product per request.** The saga builds one tenant database per run. The request is a list so
  more fit without changing it; more than one is refused today.
- **The seed is the ORM reference rows**: the rows of the shared reference tenant, read from alpha's
  `orm` schema and verified table by table. Reference data in other schemas of the scanned structure
  (`ref`, `mdm`, ...) and the mastering DDL under `backend/db/crims` are not applied.
- A product other than the one the gold copy marks a template for (`structure_template_app`) is
  refused at the planning step.

## Tests

`go test ./internal/provisioning/ ./internal/temporal/... ./cmd/uisce-tenant/` runs without a
database. The real-cluster test needs a throwaway cluster, never alpha:

```
SAGA_TEST_PG_HOST=127.0.0.1 SAGA_TEST_PG_PORT=55432 SAGA_TEST_PG_USER=postgres SAGA_TEST_PG_PASSWORD=x \
  go test ./internal/temporal/activities/ -run RealCluster_CreateRefuse -v
```
