# Orders End-to-End

Last verified: 2026-09-14.

A working recipe for CRUD + Report + Validation on the Order domain in
this stack — five Business Objects (`order`, `placement`, `execution`,
`order_allocation`, `execution_allocation`) backed by `alpha.orm.*` tables,
seeded against the `northwind` tenant. Built to replace a long stretch of
session-transcript-only knowledge about which endpoint actually serves what.

## TL;DR

| Capability | Endpoint | BO key for northwind | Driver table |
|---|---|---|---|
| CRUD (single record) | `GET/PUT/DELETE /api/bo/{boKey}/records/{recordId}`, `POST /api/bo/{boKey}/records` | `order`, `order_allocation`, etc. | `orm.order`, `orm.order_allocation`, etc. |
| BO self-describing schema (form fields) | `GET /api/bo/{boKey}/schema` | same | resolved via `MAPS_TO` catalog edges |
| Table data (paginated, search, master-filter) | `GET /api/business-objects/{boId}/data` | same (UUID form) | same |
| Report templates | `GET/POST /api/v1/reports/`, `PUT /api/v1/reports/{id}` | any BO resolvable by `business_objects.id` | same |
| Central validation engine (rules) | `GET/POST /api/validation-rules/` | any BO with `target_entity_id` set | rule metadata only |
| Validation firing (post-commit trigger) | `validation_triggers` rows; fired automatically after BO CRUD writes | `trigger_type='row_insert'`, `target_entity='order'` | same |

## Endpoint contracts

The system has **two parallel BO APIs**, both reachable, that overlap:

### A. `BOCRUDHandler` — `/api/bo/{boKey}/records/...`

`boKey` is `business_objects.bo_key` (e.g. `order`, `order_allocation`)
OR the BO's UUID. The handler resolves both forms in
`bo_crud_handler.go::resolveBOMetadata`. Single-record oriented.

- `POST /api/bo/{boKey}/records` — create
- `GET /api/bo/{boKey}/records/{recordId}` — read
- `PUT /api/bo/{boKey}/records/{recordId}` — update
- `DELETE /api/bo/{boKey}/records/{recordId}` — delete
- `GET /api/bo/{boKey}/schema` — self-describing field schema (the
  endpoint `BOFormWidget` consumes; the old `/api/metadata/bo/{boId}` was
  never implemented backend-wide and 404s).
- `GET /api/bo/{boKey}/records/{recordId}/relationships/{relKey}` etc.
  for child record navigation.

Tenant resolution: the BO CRUD handlers read `security.AuthInfo` via
`extractTenantUUIDFromRequest` and call `security.ResolveTenantID` —
the same canonical rule 67 other call sites use. Unauthenticated
requests fail with `401 missing or invalid JWT token`; mismatched
tenant claims fail with `403 forbidden: requested tenant does not match
caller's tenant`. The `X-Tenant-ID` header is **optional** — only used
when the JWT is silent on tenant membership.

### B. `BusinessObjectHandler` — `/api/business-objects/{boId}/data`

`boId` is the **UUID form only** (`business_objects.id`). This is what
`BODirectTable` in the Page Studio consumes — paginated, searchable,
master-filterable. The query path goes through
`metadata.BusinessObjectService.QueryBORecords` which (post-2026-09-14)
walks the semantic graph to resolve PascalCase `field_name` →
snake_case physical column, then falls back to literal name-match.

## Schema endpoint details

`GET /api/bo/{boKey}/schema` returns:

```json
{
  "id": "b611af7b-8689-407d-807a-eeb315065e7d",
  "boKey": "order",
  "drivingTable": "/orm/order",
  "fields": [
    {
      "id": "...",
      "name": "TargetQuantity",
      "displayName": "TargetQuantity",
      "type": "number",
      "required": false,
      "physicalColumn": "target_qty"
    }
    ...
  ],
  "relationships": []
}
```

- `name` — the PascalCase semantic-term name from
  `business_object_fields.field_name`. This is the key the Form widget
  submits on, and what the BO CRUD handler accepts as the JSON
  payload key.
- `physicalColumn` — the snake_case physical column name from walking
  `catalog_edge` MAPS_TO edges. Used for type lookup and (for the
  frontend) as the display hint.
- `required` — true when `business_object_fields.is_required = true`
  OR `binding_requirement = 'REQUIRED'`. The server still enforces
  NOT NULL/check constraints on submit; client-side `required` is a
  UX nicety.
- `type` — mapped from `information_schema.columns.data_type` for
  the resolved physical column, falling back to
  `business_object_fields.data_type`. Values are constrained to the
  four widget input types the Form widget recognizes: `text`,
  `number`, `date`, `checkbox`.

The Form widget (`frontend/src/components/reporting/BOFormWidget.tsx`)
calls this endpoint on mount, then renders one input per field.
Submit hits `POST /api/bo/{boKey}/records` (create) or
`PUT /api/bo/{boKey}/records/{recordId}` (update), passing the field
values keyed by `name`.

## Cardinal Rules respected

1. **Config-Before-Code, Graph-First.** Field resolution walks
   `business_object_fields → catalog_edge MAPS_TO → catalog_node`.
   No hardcoded `BO.Key → DriverTable` maps. The same resolver
   (`analytics.ResolveSemanticFieldMap`) is used by validation rules,
   pre-aggregation, shadow evaluation, and now the schema endpoint
   and the `/data` query path.
2. **Semantic/OLTP Boundary.** Graph holds identity + semantic
   vocabulary; OLTP (`orm.*`) holds mutable state. The schema endpoint
   reads only graph tables; the CRUD handlers write only OLTP tables.
   No financial state ever lives in graph properties.
3. **Security Mandate.** All read/write paths resolve tenant via
   `security.AuthInfo` and call `security.ResolveTenantID`. Cross-tenant
   access fails closed (401/403). Graph traversals in
   `ResolveSemanticFieldMap` are constrained by `bo_id` and tenant;
   no unscoped catalog walks.

## Master-detail flow (verified 2026-09-13, last session)

`PageComponentRenderer.tsx` wires a SelectionContext that flows:

```
<BODirectTable onRowSelect> (Tab 1, Orders)
   ↓
SelectionContext.select({ boId, recordId })
   ↓
<BOFormWidget recordId={selection.recordId}> (Tab 2, Order Form)
   ↓
GET /api/bo/{boKey}/records/{recordId}
   ↓
PATCH → PUT /api/bo/{boKey}/records/{recordId}
```

The recordId threading was verified live end-to-end before the
`/api/v1/bo/` → `/api/bo/` URL fix landed; this doc covers the
URLs as they exist post-fix.

Child tables (e.g. an Order's Order Allocations) use a separate path:
the table widget is configured with a `masterFilter` data source,
which sets `filterField` and `filterValue` (the parent's selected
`recordId`) on the table's `/data` request. Same SelectionContext,
different mechanism.

## Validation wiring

### Active enforcer: Postgres CHECK constraint

`orm.order` has `chk_order_target_qty_positive CHECK (target_qty > 0)`
from `backend/migrations/20260909_create_local_orm_schema.sql`. This
is what actually rejects bad writes at the database layer. The
constraint runs server-side on every insert/update, independent of any
backend service being up.

### Post-commit trigger engine

`backend/internal/api/trigger_engine.go::EvaluateTriggers` fires after
a BO CRUD write via `emitBORowEvent`. Each trigger is a row in
`public.validation_triggers` with `trigger_type='row_insert'`,
`target_entity='order'`, etc. The `meta` JSONB column holds
condition and action configs.

**Important:** the trigger engine is **post-commit, not pre-commit**.
It cannot block a write. It evaluates conditions and fires actions
(notifications, Temporal workflows, RabbitMQ events, webhooks). For
write-blocking validation, the rule must live in the SQL CHECK
constraint or in a pre-write validation hook (which the system does
not have for the BO CRUD path).

### Central validation engine (rules VM)

`backend/internal/validation/engine.go` is the standalone rule VM used
by pre-aggregation and shadow evaluation. Configured via
`/api/validation-rules/`. Not currently invoked by the BO CRUD path.

## Migration log

| Migration | Purpose | SHA-256 (recorded in oms.migration_log) |
|---|---|---|
| `20260909_create_local_orm_schema.sql` | ORM tables + `chk_order_target_qty_positive` | seeded by manual run, see `oms.migration_log` |
| `20260914_013_remove_mislabeled_excel_npv_from_execution_bo.up.sql` | Drop AI-leaked `Excel NPV` field from `execution` BO | `c4b33eb69063b8b1504f939c6451d678a00255da9a43ecaf9c5e4a1ac8b128a8` |
| `20260914_014_seed_order_validation_trigger.up.sql` | Seed a post-commit `row_insert` trigger on the Order BO so the trigger engine has something to evaluate | `8d1825b77cdf2d1a53591b6cfeaf61cc109347094b40857aac56b0364f21ed6c` |

## Report on Orders

Two paths exist for "report on orders":

### A. Saved Report Template (Report Builder UI)

A pre-built "Orders Report" template already exists in
`report_templates` for the northwind tenant
(id `8195b441-6096-41a9-8404-9bb9858f7d28`). It's a Table widget bound
to the Order BO with all 14 semantic-term columns, a `Year` parameter
for filtering, and standard page/footer tokens. It is reachable via
`GET /api/v1/reports/{id}` and the standard Report Builder list view.

To execute it headlessly, the canonical path is:

1. `POST /api/v1/reports/{templateId}/schedules/` — create a one-off
   schedule (the same model used by recurring reports)
2. `POST /api/v1/reports/{templateId}/schedules/{scheduleId}/run` —
   trigger the run
3. `GET /api/v1/reports/executions/{id}` — retrieve the rendered rows

Note: the schedule/burst path is tuned for multi-client distribution
("burst" rendering) — for a single-template, single-tenant render, the
overhead of a schedule is heavy. For interactive use, prefer path B.

### B. Semantic Query (the path ReportWidgetRenderer actually uses)

`ReportWidgetRenderer.tsx` (the live renderer behind every Table widget
on a Report Builder canvas) calls `executeQuery(queryDef)` which POSTs
to `/api/query/execute` with:

```json
{
  "context": {
    "boId": "b611af7b-8689-407d-807a-eeb315065e7d",
    "bindingId": "",
    "tenantId": "910638ba-a459-4a3f-bb2d-78391b0595f6"
  },
  "query": {
    "dimensions": [
      { "termNodeId": "a07f528b-f78c-4419-bd55-b0d66def5867", "alias": "SecuritiesID" },
      { "termNodeId": "ffe1b8ac-7194-4e33-82eb-3a9779e5a5f3", "alias": "Side" },
      { "termNodeId": "381758fb-7c6f-4485-80c4-cdba040c6749", "alias": "TargetQuantity" },
      { "termNodeId": "a6115401-630c-4a77-8c74-7ca81a7b1055", "alias": "Status" }
    ],
    "measures": [
      { "termNodeId": "52d56de7-e61d-41f4-8d9e-34b4c393f183", "alias": "ExecutedQty", "agg": "SUM" }
    ],
    "filters": [],
    "limit": 200
  }
}
```

Response shape (`QueryExecuteResponse`):
```json
{
  "sql": "SELECT ...",
  "columns": [{"name": "SecuritiesID", "type": "numeric", "boId": "..."}],
  "rows": [{"SecuritiesID": "12345", "Side": "BUY", "TargetQuantity": "100", ...}],
  "rowCount": 187,
  "executionTimeMs": 23
}
```

The same `QueryDef` JSON works as a body for
`POST /api/query/preview` (returns SQL only, no rows) and
`POST /api/query/execute` (returns SQL + rows).

`bindingId` can be empty if `business_object_bindings` has no rows
for the BO (which is the current state for northwind's order BO).
The boresolver falls back to the BO's `driver_table_name` directly.
This is the same path the `/api/business-objects/{id}/data` endpoint
takes, so an empty bindingId isn't blocking — it's just a hint the
UI sends when a real binding is configured.

## Validation firing (smoke test)

After the trigger engine SQL fix (commit landing alongside this doc),
every BO write fires `EvaluateTriggers`. The seed trigger
`order_inserted_new_status` matches `target_entity='order'` and
`trigger_type='row_insert'` and has a `meta.conditions` array that
matches `status == 'NEW'` AND an `meta.actions` array with a single
`webhook` action pointing at `http://localhost:9999/uisce/order-events`.

The webhook URL is intentionally not served anywhere, so the action
will fail with a logged error — but the trigger engine will run,
prove the wiring, and emit a useful log line. A real endpoint URL
turns the seed into a production-ready notification rule with one
column edit.

## Live-DB discovered BO keys (northwind tenant)

Discovered 2026-09-14. These are the values the Page Studio Form
widget should bind to.

| BO key | BO id | Driver table |
|---|---|---|
| `order` | `b611af7b-8689-407d-807a-eeb315065e7d` | `orm.order` |
| `placement` | `6610a142-3f01-4f20-817a-abcc6223c1fc` | `orm.placement` |
| `execution` | `aab56f4f-eb39-404f-8a4a-f6d15abe2530` | `orm.execution` |
| `order_allocation` | `0b5d5b5b-cef9-49d2-82ab-13457b90ee8c` | `orm.order_allocation` |
| `execution_allocation` | `d0589204-0dca-48c4-ad5e-82915ba1b82e` | `orm.execution_allocation` |

## Files changed in this work

| File | Change |
|---|---|
| `START_BACKEND.sh:69,88` | `ENVIRONMENT` defaults to `development`; `API_TOKEN_ENCRYPTION_KEY_DEV_FALLBACK` defaults to `false` |
| `backend/.env:15` | Removed dead `DEV_FALLBACK=true` line |
| `scripts/infisical-bootstrap.sh` | `generate_composite_secrets` 3-tier resolves `API_TOKEN_ENCRYPTION_KEY`; `main()` sources `backend/.env` + `rebalancing/.env` before generation so live-env tier catches the key for all 4 output files; `main()` captures `prev_api_key` alongside the DSN captures |
| `backend/internal/metadata/businessobject_service.go:3475-3500` | `QueryBORecords` calls `analytics.ResolveSemanticFieldMap` to walk MAPS_TO edges and translate PascalCase `field_name` → snake_case physical column |
| `backend/internal/api/trigger_engine.go:99-115` | Live-schema-compatible SELECT against `validation_triggers` (was referencing nonexistent `vt.trigger_type_id` column; silently broke every BO write's post-commit validation) |
| `backend/internal/api/bo_crud_handler.go` | Added `HandleGetBOSchema` + `GET /bo/{boKey}/schema` route + helpers (`splitQualifiedTable`, `normalizeFormType`); added `pq` import |
| `frontend/src/components/reporting/BOFormWidget.tsx` | `/api/v1/bo/` → `/api/bo/` (3 callsites); client-side `required` validation now reads from schema |
| `frontend/src/features/query-builder/services/queryBuilderApi.ts:111` | `fetchBOSchema` now hits `/api/bo/{boId}/schema` (was `/api/metadata/bo/{boId}`, never implemented) |
| `frontend/src/features/query-builder/types/queryDef.ts:178` | `BOSchemaField.required?: boolean` |
| `backend/db/migrations/20260914_013_remove_mislabeled_excel_npv_from_execution_bo.up.sql` | Removes the AI-leaked `Excel NPV` ghost field from `execution` BO |
| `backend/db/migrations/20260914_014_seed_order_validation_trigger.up.sql` | Seeds a `row_insert` post-commit trigger on the Order BO so the trigger engine has something to evaluate against; webhook URL is intentionally unserved |
| `.env.infisical` | Both revoked `st.31286f42-...` lines removed; `INFISICAL_TOKEN=` left empty pending regeneration |

## Post-session audit (2026-09-14 / 2026-09-15)

Follow-on work after the main handoff, across two sessions.

**1. CLI auth state (2026-09-14)** — `~/.infisical/infisical-config.json` was
populated but `infisical login status` reported `session expired`. The
CLI fell back to a local read-cache for `secrets` (which is why bootstrap
kept working offline at the time). **Write operations required a live
session and would not work until `infisical login` was run
interactively.** This was the standing-config guard, not a documentation
gap. The state has since shifted — see entry 3.

**2. Ghost-field sweep across all tenants / ORM BOs (2026-09-14)** —
the SQL audit query

```sql
SELECT t.name, bo.bo_key, count(*) FILTER (WHERE NOT EXISTS (
  SELECT 1 FROM catalog_edge ce
  JOIN catalog_edge_type et ON et.id = ce.edge_type_id
  JOIN catalog_node col ON col.id = ce.target_node_id
  WHERE ce.source_node_id = bof.term_node_id
    AND et.edge_type_name = 'MAPS_TO'
    AND col.qualified_path LIKE bo.driver_table_name || '/%'
)) AS ghost_field_count
FROM business_object_fields bof
JOIN business_objects bo ON bo.id = bof.bo_id
JOIN tenants t ON t.id = bo.tenant_id
GROUP BY t.name, bo.bo_key
HAVING ghost_field_count > 0;
```

returned zero rows. After the `Excel NPV` removal
(migration `20260914_013`), every field across every tenant's ORM
BOs resolved through a `MAPS_TO` edge to a real physical column in
its driving table. No other AI-leaked ghost fields surfaced.

**3. Infisical recovery (2026-09-15)** — `uisce-infisical` was
crash-looping on startup with `Unsupported state or unable to
authenticate data` from `Decipheriv.final` in
`/backend/src/services/kms/kms-service.ts:1043`. The instance was
crash-looping under the post-rotation key; restoring the pre-rotation
key from the value committed in `docker-compose.remote.yml`
(June 26 commit `a42e00ea56`) produced a clean boot. **Observed facts
on the recovered instance:** schema present (every table
`\d`-able), but all data tables empty — `users=0`, `organizations=0`,
`projects=0`, `secrets=0`. **The mechanism by which the post-rotation
key and the database contents diverged is not asserted here.** Several
histories are consistent with the observation (partial wipe during the
failed rotation; re-migration during the crash-loop window; etc.); an
empty DB also boots cleanly under any encryption key because there is
no wrapped-key ciphertext to fail against. **All pre-rotation Infisical
state was lost and must be re-created.** A new admin account was
provisioned via Web UI; a new project and organization were created
during recovery with **new UUIDs** that replaced the prior ones; the
Infisical user-account credential used during recovery (a temporary
bootstrap value) was rotated at the end of the browser session.

The pre-rotation value of `INFISICAL_ENCRYPTION_KEY` (committed in
`a42e00ea56`) is documented in git history and was not put into this
doc — git history is the source of record, and re-printing it here
would create a fifth exposure surface for no benefit.

**4. Postgres `admin` role rotation (2026-09-15)** — the literal
password originally committed in `DB_CONNECTION_URI` (in
`a42e00ea56`) was rotated away from the working tree. A new password
was generated, the role was altered (`ALTER ROLE admin WITH PASSWORD
…`), the compose file's DSN was rewritten to use a `${INFISICAL_ADMIN_PASSWORD}`
substitution sourced from `.env` (gitignored, mode 600), and the
container was `--force-recreate`-d. The literal value, the prior
`ENCRYPTION_KEY` literal, and the prior `AUTH_SECRET` literal were all
removed from `docker-compose.remote.yml` in the same edit batch.
The literal values remain in git history (`a42e00ea56`) permanently;
the current working tree no longer carries them. **The compose change
needs to be committed before the next `git pull`/checkout from another
machine reintroduces the literals in `origin/main`'s working tree.**

**5. SITE_URL fix (2026-09-15)** — `SITE_URL=http://localhost:8082`
in compose would have broken post-signup redirects / cookies scoped to
the host's Mac. Fixed to `http://100.84.50.65:8085` in the same edit
batch as the credential rotations; the change took effect on the
`--force-recreate`. The Infisical UI and API both serve from `:8085`
(the only host-published port — confirmed via `docker port
uisce-infisical`); `:8082` was never published.

**6. CLI auth state after recovery (2026-09-15)** — `infisical login
status` reports `session expired` again (this time across all paths:
neither the offline read-cache nor write operations work). Bootstrap
cannot pull live secrets until auth is restored via either (a)
`infisical login` run interactively, or (b) a Universal Auth
machine-identity (Client ID + Client Secret) supplied to
`INFISICAL_TOKEN` env var or to `scripts/infisical-bootstrap.sh`
directly. **The hardcoded `--projectId` in `scripts/infisical-bootstrap.sh`
was updated to the new project UUID during recovery** — see the
comment block above that variable for the current value (this doc does
not reproduce it, since it would go stale the next time the project
is recreated).

**7. Bootstrap workaround (2026-09-15)** — with CLI auth blocked,
`ENVIRONMENT=development` was manually injected into the four
gitignored `.env` files (`.env`, `backend/.env`, `calendar-service/.env`,
`rebalancing/.env`) so the backend's `START_BACKEND.sh` script default
became redundant rather than load-bearing, and the bootstrap artifact
matched Infisical's source-of-truth for the secret. When CLI auth is
restored and bootstrap runs again, the prev-capture logic will preserve
this value (not overwrite with whatever the CLI pulls, which should
be the same anyway).

**8. Bootstrap script hardening (2026-09-15)** — three cascading
defects in `scripts/infisical-bootstrap.sh` were found and patched;
the prior failure mode that produced this incident was a script
defect, not an operator mistake:
- (a) **Open-then-check truncation.** `generate_env_file` opened
  each target `.env` with `>` *before* checking whether the Infisical
  pull had produced anything usable, so a failed pull (CLI auth
  expired, Infisical unreachable, network blip) truncated the target
  down to just the header. Any subsequent step that errored (missing
  `API_TOKEN_ENCRYPTION_KEY` composite, for example) then aborted the
  script mid-write, leaving the target in a skeleton state with no
  rollback. **Patched**: writes to a temp file under
  `${output_path}.tmp.XXXXXX`, returns the temp path on stdout;
  `generate_composite_secrets` takes that temp path and appends to it;
  the final atomic `mv` into place happens only after every step has
  succeeded. `main()` treats both functions' non-zero return as
  abort-with-cleanup. `log()` was redirected to stderr so callers
  that capture stdout via `$(...)` don't accidentally absorb log
  lines into their captured value. Verified: running bootstrap with
  broken CLI auth produces a byte-identical `.env` to before the run
  (sha256 unchanged), no leftover `.env.tmp.*` files, and a clean
  stderr message.
- (b) **Empty-pull clobber via atomic mv.** The (a) patch protects
  against FAILURE clobbering, but a subsequent run *succeeded* and
  still clobbered a 115-line `.env` down to a 14-line skeleton because
  the Infisical pull returned empty `{}` and the `jq` extraction
  silently wrote zero secret lines — only the composites landed. The
  atomic mv then replaced the rich existing file with a sparse
  skeleton, no error raised. **Patched**: `generate_env_file`
  now counts the secrets via `jq` before the `mv`; if zero, the temp
  file is removed and `return 2` exits without touching the target.
  Verified: running bootstrap against an empty project leaves all
  four `.env` files byte-identical (sha256 unchanged) and prints
  `WARNING: Pulled zero secrets for $section - leaving $path untouched`.
- (c) **Missing shell-env fallback for `API_TOKEN_ENCRYPTION_KEY`.**
  The original check bailed with an error when the key wasn't in the
  pulled secrets — but that key is *deliberately* not in Infisical
  (the historical leak in `origin/main:de336a41af` was the reason
  the bootstrap never stores it there). The check now falls back to
  the shell env (which `main()` populates by sourcing `backend/.env`
  and `rebalancing/.env` before any generate calls), only failing if
  the key is in neither place.

**9. Cleanup queue (deferred until next session)** —
- `shred -u /mnt/github/uisce/.env.bak-2026-09-15-1032` (Phase 3 backup)
- `shred -u /mnt/github/uisce/.env.bak-pgrole-2026-09-15-1109` (Phase 7b backup)
- `shred -u /mnt/github/uisce/docker-compose.remote.yml.bak-2026-09-15-1109` (Phase 7b backup)
- Commit the compose change so `origin/main`'s current tree stops
carrying the literals (history retains them forever; this only affects
the current tree).
- Restore CLI auth (`infisical login` interactively, or wire
Universal Auth machine-identity into `INFISICAL_TOKEN`).
- After CLI auth is restored, run bootstrap once to confirm
the live pull produces the same value already in `.env`.

**10. Recovery postmortem (2026-09-15)** — final lessons from the
Infisical clobber-and-recovery cascade. This entry is the single point
of record for the incident; future operators reading this doc cold
should understand both what broke and why without re-running the
debugging session.

- **Root-cause chain.** The `uisce-infisical` container was wipe-
initialized during this session, leaving an empty vault. Because
`bootstrap` reads whatever Infisical returns and writes the entire
output (composites + pulled secrets + env-from-shell) into the
target `.env`, an empty pull + an existing rich file = a clobbered
file. The recovered `.env` files were the only surviving secret copy
between the wipe and re-population. **Structural lesson:** after any
vault wipe, the vault is a single point of failure until re-populated;
"bootstrap is safe to run" depends on that pre-condition, not just on
the script's own correctness.

- **Bootstrap script bug tally** (each fixed in `scripts/infisical-bootstrap.sh`):
  - (a) **Pre-pull truncation** — `generate_env_file` opened the target
    with `>` *before* confirming the pull produced anything usable, so a
    failed pull left the file as just the header. Fixed with temp-file
    redirect + atomic `mv`.
  - (b) **`set -e` exit-code capture** — assigning a variable via
    `var=$(...)` doesn't inherit `set -e` propagation correctly across
    all bash versions; using `|| var=$?` short-circuits `set -e` and
    makes `var` capture the function's real rc. Fixed by initializing
    `gen_rc=0` BEFORE the compound.
  - (c) **`|| true` and `PIPESTATUS` confusion** — using `|| true`
    makes `$?` capture 0 (the `true`'s rc) rather than the function's.
    `PIPESTATUS[0]` in this bash captures the substitution's last
    command status, not the function's. Fixed by routing the function's
    stdout through a tempfile, then reading the path string back into
    `tmp_path`.
  - (d) **KEYCLOAK_ISSUER typo** — initial composite-gen refactor
    produced `https://.../realms/uisc` (missing the trailing 'e');
    fixed by removing the six `KEYCLOAK_*` keys from Infisical so the
    composites regenerate correctly.
  - (e) **Restart launched without sourcing env** — `bash -c 'source
    ...' && ./server` does NOT propagate the sourced vars to the
    `./server` invocation; `set -a; source ...; set +a; exec ./server`
    inside one bash invocation is the correct pattern. **This was the
    agent's own launch-procedure bug; it killed a working end-to-end
    proof (PID 24533, booted at 19:55) and then fumbled the relaunch.
    The 19:55 boot had already validated the recovered files end-to-
    end before the unnecessary kill.**

- **Known limitation — partial-pull guard gap.** The empty-pull guard
added in this session fires only when `generate_env_file` returns zero
secrets. A **partial** pull (returns fewer secrets than the existing
file has keys) is not detected — the bootstrap regenerates with the
fewer secrets + composites and silently shrinks the file. The fix is
in the guard's real condition: "pull returned fewer secrets than the
existing file has keys AND existing file is substantial → warn-and-
abort." **Not implemented in this commit; flagged as known limitation.**
Will matter less once Infisical is fully populated, but the vault must
be the complete source of truth before bootstrap is trusted again.

- **Recovery method (for the record).** When a heap-resident env is the
  only secret copy, the recovery chain that worked was: (1) `vmmap`
  enumeration of writable memory regions, (2) `lldb` memory dump of
  each region with stdout redirected to a tempfile (`region_*.bin`),
  (3) `cat region_*.bin | strings -a -n 6 | grep -aE '^[A-Z][A-Z0-9_]{2,}='`
  to extract uppercase env-like pairs, (4) per-key regex cleanup with
  `PLACEHOLDER_PATTERNS` to drop `<...>`, `your-*-here`, `changeme`,
  etc., (5) cross-check against `.env.example` for keys whose recovered
  value matches the example placeholder (this dropped 26 keys), (6) bucket
  the result into "recovered clean" + "placeholder contamination" +
  "corrupted/uncertain" + "never in dump" + "recovered as empty (verify
  intentional)", and (7) verify **the four bucket sums plus the empty
  bucket equals exactly 111** (the A.5 baseline). Sum-integrity is the
  load-bearing test — if the buckets don't sum to the baseline, a key
  was silently lost.

- **Human factors.**
  - (a) **Abandoned Create Secret dialog never submitted** — the
    initial Infisical save silently failed (the form was closed without
    the Create button being clicked). The user thought the secret was
    saved; the empty vault persisted. **Mitigation:** always
    `infisical secrets --projectId=... --env=... --path=...` after a
    Web UI save to verify the round-trip.
  - (b) **"Verify the save actually happened" before downstream
    automation** — the heap dump only existed because PID 82334 was
    still running with the pre-clobber env. The 81-key recovered file
    was a lucky snapshot. If the backend had been restarted between the
    clobber and the recovery attempt, the heap would have been empty and
    the file would be unrecoverable. **The lesson: after any vault
    mutation, verify by reading the vault back through CLI before
    declaring success.**
  - (c) **Bootstrap's launch procedure is the most error-prone part.**
    Source-and-launch must be one bash invocation, not separate blocks;
    this gotcha burned the agent during the restart verification.
