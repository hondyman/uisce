<!-- gitnexus:start -->
# GitNexus — Code Intelligence

This project is indexed by GitNexus as **uisce** (393918 symbols, 558647 relationships, 300 execution flows). Use the GitNexus MCP tools to understand code, assess impact, and navigate safely.

> Index stale? Run `node .gitnexus/run.cjs analyze` from the project root — it auto-selects an available runner. No `.gitnexus/run.cjs` yet? `npx gitnexus analyze` (npm 11 crash → `npm i -g gitnexus`; #1939).

## Always Do

- **MUST run impact analysis before editing any symbol.** Before modifying a function, class, or method, run `impact({target: "symbolName", direction: "upstream"})` and report the blast radius (direct callers, affected processes, risk level) to the user.
- **MUST run `detect_changes()` before committing** to verify your changes only affect expected symbols and execution flows. For regression review, compare against the default branch: `detect_changes({scope: "compare", base_ref: "main"})`.
- **MUST warn the user** if impact analysis returns HIGH or CRITICAL risk before proceeding with edits.
- When exploring unfamiliar code, use `query({query: "concept"})` to find execution flows instead of grepping. It returns process-grouped results ranked by relevance.
- When you need full context on a specific symbol — callers, callees, which execution flows it participates in — use `context({name: "symbolName"})`.

## Never Do

- NEVER edit a function, class, or method without first running `impact` on it.
- NEVER ignore HIGH or CRITICAL risk warnings from impact analysis.
- NEVER rename symbols with find-and-replace — use `rename` which understands the call graph.
- NEVER commit changes without running `detect_changes()` to check affected scope.

## Resources

| Resource | Use for |
|----------|---------|
| `gitnexus://repo/uisce/context` | Codebase overview, check index freshness |
| `gitnexus://repo/uisce/clusters` | All functional areas |
| `gitnexus://repo/uisce/processes` | All execution flows |
| `gitnexus://repo/uisce/process/{name}` | Step-by-step execution trace |

## CLI

| Task | Read this skill file |
|------|---------------------|
| Understand architecture / "How does X work?" | `.claude/skills/gitnexus/gitnexus-exploring/SKILL.md` |
| Blast radius / "What breaks if I change X?" | `.claude/skills/gitnexus/gitnexus-impact-analysis/SKILL.md` |
| Trace bugs / "Why is X failing?" | `.claude/skills/gitnexus/gitnexus-debugging/SKILL.md` |
| Rename / extract / split / refactor | `.claude/skills/gitnexus/gitnexus-refactoring/SKILL.md` |
| Tools, resources, schema reference | `.claude/skills/gitnexus/gitnexus-guide/SKILL.md` |
| Index, status, clean, wiki CLI commands | `.claude/skills/gitnexus/gitnexus-cli/SKILL.md` |

<!-- gitnexus:end -->

---

# STI Implementation — Single-Table Inheritance

This project uses Single-Table Inheritance (STI) with `subtype_code TEXT NOT NULL` as the discriminator column.

## Architecture

```
Handler → Service → Repository → PostgreSQL (STI table)
```

Each entity lives in its own package under `internal/`:

| Entity | Package | Table | Subtypes |
|--------|---------|-------|----------|
| Account | `internal/oms/account` | `oms.account` | institutional, retail_wealth, sma, trust_estate, qualified_retirement, corporate_treasury |
| Position | `internal/oms/position` | `oms.position` | settled_long, short_borrowed, derivative_exposure, pledged_collateral, unsettled_pipeline |
| Security | `internal/oms/security` | `oms.security` | equity, sovereign_debt, corporate_debt, structured_abs_mbs, etd_derivative, otc_derivative |
| TradeOrder | `internal/oms/trade_order` | `oms.trade_order` | block_parent, dma_execution, otc_bilateral, fx_spot_forward, primary_auction |
| AlternativeInvestment | `internal/altinv/alternative_investment` | `altinv.alternative_investment` | private_equity, venture_capital, hedge_fund, real_estate, direct_investment, infrastructure, private_debt |
| Settlement | `internal/cashflow/settlement` | `cash_flow.settlement` | dividend, coupon_fixed_income, capital_call, lp_distribution, corporate_action, expense_fee |
| Customer | `internal/master/customer` | `master.customer` | institutional_client, private_wealth, broker_dealer, corporate_treasury |
| Vendor | `internal/master/vendor` | `master.vendor` | custodian_prime_broker, market_data, fund_admin, cloud_tech |
| Personnel | `internal/master/personnel` | `master.personnel` | portfolio_manager, trade_execution, compliance_officer, client_advisor |
| SalesLedger | `internal/master/sales_ledger` | `master.sales_ledger` | aum_management_fee, trading_commission, performance_fee, platform_subscription |

## Package Structure (per entity)

```
internal/<domain>/<entity>/
  model.go          — Record struct + Validate() + subtype constants
  errors.go         — Sentinel errors (ErrInvalidSubtype, ErrNotFound, etc.)
  repository.go      — List, Get, Create, SoftDelete (bitemporal, soft-delete)
  service.go        — Thin layer: validates, sets TenantID, delegates to repo
  handler.go        — HTTP handlers (List/Get/Create/SoftDelete + RegisterRoutes)
  handler_test.go   — httptest tests with in-memory mock service
  validate_test.go  — Unit tests for Validate() method
```

## Key Patterns

- **Bitemporal soft-delete**: `valid_to IS NULL` filter on all queries; soft-delete sets `valid_to = NOW()`
- **Tenant isolation**: All queries filter by `tenant_id`; handlers extract from JWT via `jwtmiddleware.GetClaimsFromContext`
- **Handler interfaces**: Each handler's `Service` field is an interface (e.g., `AccountServiceInterface`) enabling unit testing with in-memory mocks
- **Route prefixes**: `/api/oms/*`, `/api/altinv/*`, `/api/cash-flow/*`, `/api/master/*`
- **Subtype validation**: `Validate()` checks `subtype_code` against the registry; subtype-specific rules enforced (e.g., `institutional` requires `sponsor_id`, `short_borrowed` requires `prime_broker_id`)

## Migration Files

All migrations are in `backend/db/migrations/`:

- `20260823_001_oms_subtype_registry.up.sql` — registry table + JSON schemas
- `20260823_010_oms_investment_trading_subtypes.up.sql` — oms.account, oms.position, oms.security, oms.trade_order
- `20260823_011_oms_alternatives_and_cash_flow_subtypes.up.sql` — altinv.alternative_investment, cash_flow.settlement
- `20260823_012_master_directory_subtypes.up.sql` — master.customer, master.vendor, master.personnel, master.sales_ledger

Seeds: `backend/db/seeds/20260823_oms_subtype_registry.sql` (22 rows in `oms.subtype_registry`).

## Migration Runner

`backend/internal/migrations/runner.go` — SHA-256 hash-checked idempotent runner wired into `internal/api/api.go:SetupRouter`.

## JWT Middleware

JWT middleware lives at `github.com/hondyman/uisce/libs/jwt-middleware` (NOT `internal/middleware/jwtmiddleware`).

---

## STI → Semantic → Catalog → Tenant OLTP Pipeline

Wires `oms.subtype_registry` (JSONB `field_allowlist`) → catalog graph nodes → tenant OLTP column introspection → semantic term linking into a cohesive end-to-end pipeline.

### Source

- `oms.subtype_registry` — 22 seeded rows, each with `field_allowlist JSONB` listing allowed columns per subtype

### Stage 1: Subtype Registry Loader

**File:** `backend/internal/catalog/subtype_registry.go`

- Loads `oms.subtype_registry` rows per tenant
- 5-minute TTL in-memory cache keyed by `tenant_id`
- JSONB `field_allowlist` decoded into `[]string`

### Stage 2: Subtype BO Builder

**File:** `backend/internal/catalog/subtype_bo_builder.go`

- Creates `BUSINESS_OBJECT` catalog nodes for each `(root_object, subtype_code)` pair
- Creates `ATTRIBUTE` child nodes for each entry in `field_allowlist`
- Links attribute → BO via `ATTRIBUTE_OF` edges
- Upserts using `ON CONFLICT (tenant_id, qualified_path)` — requires `catalog_node_tenant_path_uniq` constraint

### Stage 3: STI Column Scanner

**File:** `backend/internal/catalog/sti_column_scanner.go`

- Introspects `information_schema.columns` for schemas: `oms`, `altinv`, `cash_flow`, `master`
- Emits `TABLE` nodes per STI table (10 total)
- Emits `ATTRIBUTE` nodes per physical column
- Links column → table via `COLUMN_OF` edges
- Tenant isolation via `tenant_id` binding

### Stage 4: Subtype Semantic Linker

**File:** `backend/internal/catalog/subtype_semantic_linker.go`

- Matches `ATTRIBUTE` nodes to existing `SEMANTIC_TERM` nodes by name (`node_key` or `node_name`)
- Creates `IS_CLASSIFIED_AS` edges with `{"confidence": 1.0, "source": "exact_name_match"}` properties
- `NOT EXISTS` guard prevents duplicate edges

### Admin API Endpoint

**File:** `backend/cmd/catalog-admin/main.go` (standalone HTTP server)

- `POST /api/catalog/admin/sync-subtypes`
- Requires `X-Tenant-ID` header (returns 400 if missing/invalid)
- Orchestrates Stages 1 → 2 → 3 → 4 in sequence
- Runs on port 8082 (or `PORT` env var)

### Migration

**File:** `backend/db/migrations/20260824_001_catalog_sti_unique_constraints.up.sql`

```sql
ALTER TABLE catalog_node ADD CONSTRAINT catalog_node_tenant_path_uniq UNIQUE (tenant_id, qualified_path);
```

### Qualified Path Conventions

| Source | Path Pattern | Node Type |
|--------|-------------|------------|
| `oms.subtype_registry` row | `oms.{root_object}/{subtype_code}` | `BUSINESS_OBJECT` |
| `field_allowlist` field | `oms.{root_object}/{subtype_code}/{field}` | `ATTRIBUTE` |
| STI table | `{schema}.{table}` (e.g., `oms.account`) | `TABLE` |
| Physical column | `{schema}.{table}/{column}` | `ATTRIBUTE` |

### Key Files

```
backend/internal/catalog/subtype_registry.go         — Stage 1
backend/internal/catalog/subtype_bo_builder.go       — Stage 2
backend/internal/catalog/sti_column_scanner.go       — Stage 3
backend/internal/catalog/subtype_semantic_linker.go  — Stage 4
backend/internal/api/catalog_admin_handlers.go        — HTTP handler
backend/cmd/catalog-admin/main.go                    — Standalone server
backend/db/migrations/20260824_001_catalog_sti_unique_constraints.up.sql
backend/internal/catalog/subtype_registry_test.go
backend/internal/catalog/subtype_bo_builder_test.go
backend/internal/catalog/sti_column_scanner_test.go
backend/internal/catalog/subtype_semantic_linker_test.go
backend/internal/catalog/sti_e2e_test.go
```

---

## docker-compose.remote.yml — Operational Notes (Sept 2026)

Operational gotchas discovered while bringing the remote stack up cleanly. These are non-obvious failures that will recur if not documented.

### StarRocks single-node 3.x — never pass `--helper` on first boot

The compose `command` for `starrocks-fe` must remain:

```yaml
command: >
  /opt/starrocks/fe/bin/start_fe.sh
```

Do **not** add `--helper starrocks-fe:9010`. On first boot of a single-node cluster the FE tries to contact the helper as part of bootstrap. The helper doesn't exist yet, so the FE enters a `Connection refused` retry loop and never elects itself as LEADER. Once `meta/image/ROLE` exists (i.e. second boot onward) the helper flag would be ignored anyway — `--helper` is only meaningful when joining an *existing* cluster.

If the meta dir ever needs wiping (e.g. corrupted BDB JE state), wipe the volume and start fresh without `--helper`:

```bash
docker compose -f docker-compose.remote.yml stop starrocks-fe
docker volume rm uisce_starrocks_fe_meta
docker compose -f docker-compose.remote.yml up -d starrocks-fe
```

### StarRocks `starrocks_init.sql` is stale and unused

`backend/internal/analytics/starrocks_init.sql` references services that don't exist in this stack (`nessie:19120`, `minio:9000`) and creates a `wealth_analytics` DB with materialized views on tables that aren't populated. **Do not apply it.** The bind mount into `/docker-entrypoint-initdb.d/` was removed from compose because that path is a Postgres convention and the StarRocks image ignores it anyway.

The actual DDL needed for the CDC stream-load pipeline (DB `oms`, tables `orm_execution`, `orm_order`, `orm_placement`, `orm_order_allocation`, `orm_execution_allocation`) was applied manually via:

```bash
docker exec -i starrocks-fe mysql -h 127.0.0.1 -P 9030 -u root < /tmp/oms_init.sql
```

Type mapping: postgres `uuid` → `VARCHAR(36)`, `numeric(p,s)` → `DECIMAL(p,s)`, `timestamptz` → `DATETIME` (UTC). The DDL is kept outside compose because it's tied to the source postgres schema (`alpha.orm.*`), not to the StarRocks image.

### Stream loaders — `KAFKA_BROKERS` must use the full container name

The five `uisce-stream-loader-*` services must use:

```yaml
KAFKA_BROKERS: uisce-redpanda:9092
```

The Docker-internal DNS does **not** resolve the bare `redpanda` short name on the `remote-net` bridge — only `uisce-redpanda` (the actual `container_name`) and the network alias resolve. If you see `lookup redpanda on 127.0.0.11:53: server misbehaving` in a stream loader's logs, this is why.

### Lake keeper healthcheck must use the binary's own CLI

The `quay.io/lakekeeper/catalog:latest-main` image contains only `/home/nonroot/lakekeeper` — no `/bin/sh`, no `curl`, no `wget`, no `nc`. The compose healthcheck must be:

```yaml
test: ["CMD", "/home/nonroot/lakekeeper", "healthcheck"]
```

Not `CMD-SHELL` with curl/wget — the spawn of `/bin/sh` fails with `no such file or directory` and the healthcheck will never pass.

### Lake keeper port 8181 sometimes has an orphan `java` PID

After recreating `lakekeeper` you may see `failed to bind host port for 0.0.0.0:8181`. Before retrying, check for an orphan and kill it:

```bash
ss -tlnp | grep ':8181 '
# find pid=NNN from the users:((...)) field, then kill -9 NNN
```

This happens when a previous lakekeeper container was killed but the JVM held the socket briefly.

### StarRocks BE registration must be re-run after storage wipe

When you wipe `uisce_starrocks_be_storage` (e.g. to clear a `Unmatched cluster id` error), the BE does **not** auto-register — you must run:

```sql
ALTER SYSTEM ADD BACKEND "starrocks-be:9050";
```

Then `SHOW BACKENDS;` should report `Alive=true` within ~10s of the BE heartbeat. The BE has only one IP on `remote-net` so `priority_networks` in `be.conf` is not needed.

### `uisce-infisical` has a literal DB password in compose — tech debt

`docker-compose.remote.yml` line ~246 contains:

```yaml
- DB_CONNECTION_URI=postgresql://admin:<REDACTED>@100.84.50.65:5432/infisical?sslmode=disable
```

The `admin` user was created on the local mTLS postgres specifically to dodge schema-ownership conflicts during a never-finished prior install (759 orphaned tables were dropped in `public`). The password is in the repo as a working compromise. **Migrating to a dedicated `infisical` postgres role with the password in `.env` is good backlog hygiene** but requires re-running the ownership reassignment dance against a live Infisical install, which is riskier than the least-privilege win is worth today.

### Authoritative compose file location

The Docker labels on every container point at `/mnt/github/uisce/docker-compose.remote.yml`. The copy at `/home/eganpj/uisce/docker-compose.remote.yml` (user's local dev environment on the server) is a separate file that **does not auto-sync**. After any compose edit on `/mnt/github/uisce/...`, run:

```bash
cp /mnt/github/uisce/docker-compose.remote.yml /home/eganpj/uisce/docker-compose.remote.yml
```

Edits made to the latter will silently be lost on the next `cp` (or, worse, reintroduce bugs that the upstream copy had fixed).

---

## Debezium CDC connector (Sept 2026)

The CDC pipeline runs Debezium Postgres → Redpanda topics → Go stream-loaders → StarRocks. **None of the original setup survived contact with reality** — the connector was registered with a broken config and immediately FAILED. The cluster was up but emitting zero events for weeks. Operational gotchas:

### Mounting the mTLS client certs into the Debezium container

The local Postgres at `100.84.50.65:5432` requires mTLS. Debezium needs three cert files mounted into its container:

| Host path | Container path | Notes |
|---|---|---|
| `/home/eganpj/.uisce/certs/ca.crt` | `/tmp/orm_ca.crt` | mode 644, owned by 1000:1000 → fine |
| `/home/eganpj/.uisce/certs/postgres-client.crt` | `/tmp/orm_client.crt` | mode 644, owned by 1000:1000 → fine |
| `/home/eganpj/.uisce/certs/postgres-client.pk8` | `/tmp/orm_client_der.pk8` | **mode 600, owned by eganpj:eganpj (1000:1000)** |

The third file is the problem: Debezium's Kafka Connect process runs as user `kafka` (uid **1001**) inside the container. The original `.uisce/certs/` `.pk8` is owned by `eganpj` (uid 1000), so `kafka` cannot read it even though Docker bind-mounts it.

**Working fix:** copy the `.pk8` to `/tmp/orm_client_der.pk8` on the host (world-readable) and mount that into the container. Do not try to `chown` the original — `eganpj` cannot chown to uid 1001 without sudo, and `chmod` defeats the purpose of mode 600.

```yaml
# docker-compose.remote.yml — uisce-debezium volumes:
      - /home/eganpj/.uisce/certs/ca.crt:/tmp/orm_ca.crt:ro
      - /home/eganpj/.uisce/certs/postgres-client.crt:/tmp/orm_client.crt:ro
      - /tmp/orm_client_der.pk8:/tmp/orm_client_der.pk8:ro
```

### Original connector config was wrong on three counts

1. `database.dbname: crims` — **doesn't exist.** The actual DB is `alpha`. Either was leftover from a prior project or an editing mistake.
2. No `table.include.list` — `schema.include.list: orm` would have captured **all 7** tables in `orm` (account, broker, execution, execution_allocation, order, order_allocation, placement) but the stream loaders only target 5. Events for `account` and `broker` would have been emitted but no loader exists to consume them → silent data loss into Kafka.
3. `publication.autocreate.mode: disabled` with no manual publication created. The connector would never start.

### Corrected registration recipe (verified working 2026-09-13)

```bash
# 1. Drop the stale slot and create the publication manually
PGPASSWORD=postgres psql -h localhost -U postgres -d alpha <<'SQL'
SELECT pg_drop_replication_slot('orm_oms_slot');  -- ignore "does not exist"
CREATE PUBLICATION orm_cdc_publication FOR TABLE
    orm.execution,
    orm."order",
    orm.placement,
    orm.order_allocation,
    orm.execution_allocation;
SQL

# 2. Delete the broken connector
curl -s -X DELETE http://localhost:8083/connectors/orm-oms-connector

# 3. Re-register with corrected config
curl -s -X POST http://localhost:8083/connectors -H 'Content-Type: application/json' -d '{
  "name": "orm-oms-connector",
  "config": {
    "connector.class": "io.debezium.connector.postgresql.PostgresConnector",
    "topic.prefix": "orm_oms",
    "database.hostname": "100.84.50.65",
    "database.port": "5432",
    "database.user": "postgres",
    "database.password": "postgres",
    "database.dbname": "alpha",
    "database.sslmode": "verify-full",
    "database.sslcert": "/tmp/orm_client.crt",
    "database.sslkey": "/tmp/orm_client_der.pk8",
    "database.sslrootcert": "/tmp/orm_ca.crt",
    "plugin.name": "pgoutput",
    "slot.name": "orm_oms_slot",
    "publication.name": "orm_cdc_publication",
    "publication.autocreate.mode": "disabled",
    "schema.include.list": "orm",
    "table.include.list": "orm.execution,orm.\"order\",orm.placement,orm.order_allocation,orm.execution_allocation",
    "snapshot.mode": "initial",
    "tombstones.on.delete": "false",
    "decimal.handling.mode": "double",
    "time.precision.mode": "connect"
  }
}'

# 4. Verify task is RUNNING (not RUNNING with FAILED task like before)
curl -s http://localhost:8083/connectors/orm-oms-connector/status | jq '.tasks[].state'
# All tasks must say "RUNNING"
```

### Stream loaders don't read historical messages

The Go stream-loaders use `segmentio/kafka-go` and start with **LastOffset** by default — they only consume events that arrive *after* they start. After the connector registers, the initial `snapshot.mode=initial` dumps all 99 execution rows into the topic, but the loaders (already running) skip them.

For a smoke test, INSERT a new row *after* the connector is RUNNING. ~5s later it appears in `oms.orm_execution`. Decimal decoding (123.4567 with scale 4 → DECIMAL(18,4) → 123.4567 in StarRocks) was verified end-to-end on 2026-09-13.

## uisce-infisical password migration — backlog task

The current setup uses Postgres role `admin` with literal password `<REDACTED>` committed in `docker-compose.remote.yml`. This was a working compromise to dodge schema-ownership conflicts during a prior broken install. The proper least-privilege migration is:

```bash
# 1. Create the dedicated role (run as postgres superuser)
PGPASSWORD=postgres psql -h localhost -U postgres -c \
  "CREATE ROLE infisical WITH LOGIN PASSWORD '<chosen password>';"

# 2. Reassign database ownership
PGPASSWORD=postgres psql -h localhost -U postgres -c \
  "ALTER DATABASE infisical OWNER TO infisical;"

# 3. Reassign existing objects (if you DON'T keep admin as superuser)
#    This moves every object currently owned by admin → infisical.
#    Skip this step if admin remains a superuser.
PGPASSWORD=postgres psql -h localhost -U postgres -d infisical -c \
  "REASSIGN OWNED BY admin TO infisical;"

# 4. Move the literal password out of docker-compose.remote.yml:
#      - Add INFISICAL_DB_PASSWORD=<chosen> to /mnt/github/uisce/.env
#      - Change DB_CONNECTION_URI to:
#          postgresql://infisical:${INFISICAL_DB_PASSWORD}@100.84.50.65:5432/infisical?sslmode=disable
#      - Add /mnt/github/uisce/.env to .gitignore if not already

# 5. Force container to recreate
cd /mnt/github/uisce && docker compose -f docker-compose.remote.yml up -d --force-recreate uisce-infisical
```

**Why this is non-trivial:** the prior install dropped/recreated the `public` schema and reset ownership to `admin`. If you skip step 3 and just swap the URL, infisical will fail on any preexisting table with `must be owner of X`. Decide upfront whether to keep `admin` as a superuser or fully retire it.

## Orphan host PIDs after `docker compose down`

When a long-running JVM/Rust container is stopped with `docker compose down` (or killed via `docker rm -f`), the kernel can briefly hold the published port after the process exits. Docker will then refuse to bind on the next `up -d` with:

```
failed to bind host port for 0.0.0.0:<port>: ...: address already in use
```

The process is gone from `docker ps` but a host PID still owns the socket. Watch for this specifically on the JVM-based services (lakekeeper, keycloak). Mitigations:

```bash
# Before each `up -d`, check for orphans on the service's published port:
ss -tlnp | grep ':8181 '   # lakekeeper
ss -tlnp | grep ':8443 '   # keycloak

# find pid=NNN from the users:((...)) field, then:
kill -9 NNN
```

If the service is healthy now, you can usually `up -d` it again and it'll pick up correctly — but if you hit "address already in use", assume an orphan and investigate before retrying.

---

## Stream loader semantics — INSERT-only, DELETEs silently dropped

`backend/cmd/stream_loader/main.go` lines 145-150: the Debezium-envelope decoder returns `(nil, true, nil)` — a "skip" sentinel — whenever `payload.after` is empty or `null`. Debezium emits this shape for **`op: d` (delete)** events, since they only carry `before` (the deleted row), not `after` (the post-state).

**Consequence:** the CDC pipeline in this stack is **insert/upsert-only with respect to StarRocks**. Deletes (and any event whose `payload.after` is null for whatever reason) reach Kafka, are read by the loader, and are silently discarded. They never reach StarRocks.

This was confirmed end-to-end on 2026-09-13: an INSERT into `alpha.orm.execution` propagated to `oms.orm_execution` correctly; a subsequent DELETE from `alpha.orm.execution` left the row in `oms.orm_execution` unchanged. Loader logs stayed clean — no errors, just silent skip.

For most CDC-to-OLAP patterns, INSERT-only is the correct behavior (you don't want to retroactively erase historical fact rows). But it does mean:

- **StarRocks and Postgres will diverge on deletes.** A row deleted upstream stays in StarRocks until manually reconciled.
- **`op: u` updates** work correctly — Debezium's update events carry both `before` and `after`, and stream-load treats the row in `after` as an upsert keyed on `id`.
- **No tombstone events.** We registered the connector with `tombstones.on.delete=false`, so Debezium won't emit the null-bodied DELETE tombstones either — combined with the loader skipping null-after events, deletes are doubly ignored.

If a delete-propagating pipeline is needed later (rare for OLAP), the fix is in `backend/cmd/stream_loader/main.go::decodeRecord` — branch on `env.Payload.Op` and emit a StarRocks `DELETE FROM ... WHERE id = ?` for `op: d`, plus a config flag to enable it.

---

## Secrets rotation — done 2026-09-13

**Status: completed.** Both Infisical secrets and the Lakekeeper placeholder key are now sourced from `/mnt/github/uisce/.env` (gitignored). Compose uses `${VAR}` interpolation.

| Secret | Status | Where it lives |
|---|---|---|
| `INFISICAL_ENCRYPTION_KEY` | rotated from previous plain-text in history; new value is 32 hex chars (16 bytes) — Infisical rejects anything else with `Invalid key length` | `/mnt/github/uisce/.env` (mode 600) |
| `INFISICAL_AUTH_SECRET` | rotated; new value is 32 hex chars | `/mnt/github/uisce/.env` |
| `LAKEKEEPER_PG_ENCRYPTION_KEY` | generated via `openssl rand -base64 48`; placeholder `change-this-...` is gone from compose | `/mnt/github/uisce/.env` |

### What was rotated vs. what was leaked

The previous values of `INFISICAL_ENCRYPTION_KEY` and `INFISICAL_AUTH_SECRET` were committed in `a42e00ea56` ("feat: update docker-compose configuration for remote infrastructure", 2026-06-26) and **that commit IS on `origin/main`**. So the previous values are in shared history and should be considered burned — `AUTH_SECRET` is invalidated because the value changed; `ENCRYPTION_KEY` is invalidated because the encrypted data in the running Infisical instance was wiped before restart with the new key. The instance has zero user secrets — it's effectively fresh.

### Pre-push hook installed

`.git/hooks/pre-push` on the local Mac checks every push for added lines (not removed) containing:

- `ENCRYPTION_KEY=<32 hex chars>` (literal value, not `${...}`)
- `AUTH_SECRET=<32 hex chars>`
- `PASSWORD=<something not followed by ${...}>`
- `<REDACTED>` (intentional tripwire — if you see this in a diff, you've copy-pasted from a leaked snippet and should re-look)
- `postgres:Gu1nn3ss` / `admin:Gu1nn3ss` (defensive: if the previous password gets re-introduced)

It bypasses with `git push --no-verify` for legitimate cases (e.g., you're actually rotating and want to land the secret removal + re-add). The hook only checks *added* lines — secrets being removed from compose are fine.

**To make the hook persistent across clones** for other operators, the script should be committed to `scripts/git-hooks/pre-push` and configured via:

```bash
git config core.hooksPath scripts/git-hooks
```

That's not done in this session — single-operator local install is sufficient for now. The hook won't follow a fresh `git clone`.

### History rewrite (Case B unaddressed)

The previous secret values are still discoverable in `origin/main` history via:

```bash
git log --all --oneline -S '1030410d3281d742b960cf2ff13705f6'
```

A proper `git filter-repo --invert-paths` rewrite + force-push would scrub them, but only the operator (you) can decide whether `origin` is private-to-you (in which case the leak is theoretical) or shared (in which case rotation alone may not satisfy an audit). Marked as **out of scope for this session — your call**.

---

## Chat transcripts are secrets-bearing artifacts — operational rule

**Effective 2026-09-13, after the `Gu1nn3ss!`-in-AGENTS.md incident.**

Any literal value (password, token, API key, encryption key, JWT secret, etc.) written to a session transcript is **exposed** the moment that transcript is exported, synced, copied, fed to another tool, or screenshotted. This is not hypothetical — transcripts get exported. Treat every value quoted in chat the same way you'd treat a value pasted into a public GitHub issue.

**Rules:**

1. **The agent never echoes live secret values in chat output.** Read from `.env` or a secrets file; write via API or CLI; redact the value in any tool output before printing it. If a tool surfaces a value that wasn't supposed to be visible, do not paste it back into the conversation.
2. **Generated fresh values stay in `.env` only.** When the agent needs a new random secret (e.g., a 32-byte AES key), it writes directly to `.env` and reports only "wrote 32-hex value to `.env`" — never the value itself.
3. **Backups and exports containing `.env` are mode 600.** They should never be committed, never copied into chat, never shared with anyone who doesn't need the live values. The pre-push hook enforces the in-repo side; the human enforces the off-repo side.
4. **Once a value appears in chat, treat it as burned.** Whether to rotate depends on the threat model — see the "Secrets rotation" section above. The default action when a value escapes into chat is **rotate**, not "note it for later."

**Why this rule exists.** Two compounding failures led to it:

- The very first Infisical rotation in this session put the password `Gu1nn3ss!` into `AGENTS.md` as a code example. That file is the same file that is the project context for every future session; the password lived in `.gitignore`'d `.env` originally and got committed as documentation.
- A subsequent session re-quoted four live secrets (the post-rotation `ENCRYPTION_KEY`, `AUTH_SECRET`, a backend `ENCRYPTION_KEY`, and a `JWT_SECRET`) into chat on the assumption they'd "never be exported." There is no such assumption that's safe.

The pattern to break is: **secrets flowing through prose.** The fix is: secrets flowing through file I/O and APIs, prose describing what happened to them.

---

## `API_TOKEN_ENCRYPTION_KEY` migration — done 2026-09-13

**Status: completed.** The literal default value `D+1O956T8t9zZ+w/FqK1lS9b8jJ2vR7mX4kY0uP3oN8=` is gone from both `START_BACKEND.sh` and `scripts/infisical-bootstrap.sh`. It was a hardcoded fallback that masked missing env vars with a value committed to `origin/main` since `de336a41af`.

**What changed:**

- `START_BACKEND.sh` line 76 — replaced the literal fallback with `: "${API_TOKEN_ENCRYPTION_KEY:?API_TOKEN_ENCRYPTION_KEY not set — refusing to fall back to a value that is in git history (origin/main:de336a41af)}"`. The script now exits non-zero if the env var is missing.
- `scripts/infisical-bootstrap.sh` — removed the `echo "API_TOKEN_ENCRYPTION_KEY=D+1O956T8t9zZ+w/FqK1lS9b8jJ2vR7mX4kY0uP3oN8="` fallback and replaced with an error message + `return 1`.
- Removed `API_TOKEN_ENCRYPTION_KEY_DEV_FALLBACK=true` auto-set in the bootstrap path. (The env var itself remains in the script for explicit opt-in via the existing `START_BACKEND.sh` line 75, which uses a *different* fallback — a random process-lifetime key — and is gated by the backend's own dev-only enforcement.)

**Why this matters.** Without the fail-fast guard, rotating the key in Infisical was theater: a misspelled or missing `API_TOKEN_ENCRYPTION_KEY` env var would silently fall back to the leaked value with no error. Now the service refuses to start.

**Cost of rotation: zero.** `alpha.integration_credentials` had 0 rows at the time of rotation (verified 2026-09-13) — the key was set but no data was encrypted with it. A fresh 32-byte base64 key was generated and written to the 4 `.env` files that carry it (`.env`, `backend/.env`, `calendar-service/.env`, `rebalancing/.env`; `frontend/.env.local` correctly has no such key).
