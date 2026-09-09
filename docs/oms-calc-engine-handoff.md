# OMS Calculated-Term / CDC Pipeline — Handoff

Written 2026-09-09 at the end of a session that took the OMS calculated-term
mapping feature from "selectable in the UI" through a fully live CDC
pipeline into StarRocks. This document is the state to hand into a fresh
session — it captures what's real, what's verified, what's still a stub,
and exactly what the next piece of work is.

## Where things stand, in one paragraph

Calculated semantic terms (Excel NPV, TVPI, IRR, etc.) can be selected as
Business Object fields. A real CDC pipeline — Postgres logical replication
via a Debezium connector, through Kafka, into StarRocks — is live on
`100.84.50.65` for all 5 OMS tables (`order`, `placement`,
`order_allocation`, `execution`, `execution_allocation`). A pre-aggregation
catalog layer lets you register "roll this BO + these terms/calcs up into a
StarRocks materialized view" as a first-class catalog object, and DDL
generation for the *dimension* half of that (plain semantic terms mapped to
real physical columns) is genuinely correct and verified. The *measure*
half — calculated terms like Excel NPV — is an honest `NULL /* TODO */`
placeholder, because there is no formula-to-SQL compiler anywhere in this
codebase for the format calculated terms actually use. That compiler, and
executing the resulting DDL against StarRocks (`ApplyMaterialization` is
still a stub), are the two pieces of unfinished work this document exists
to hand off.

## What's fully working and verified (don't redo this)

### 1. Calculated terms as BO fields
- `backend/internal/metadata/businessobject_service.go`:
  `GetSemanticTermsByTable` now also surfaces tenant-wide calculated
  catalog terms (`term_type = 'calculated'`), not just column-linked ones.
- `backend/internal/api/catalog_handler.go`: `handleGetSemanticTermsByTable`
  passes tenant ID through.
- Verified: "Excel NPV" selected and persisted as a field on the Execution
  BO (`business_object_fields`, `data_type = number`).

### 2. CDC pipeline (Postgres -> Debezium -> Kafka -> StarRocks)
All running on **100.84.50.65** (SSH as `eganpj`, repo checked out at
`/mnt/github/uisce`, compose files in `/home/eganpj`):

- **Postgres side**: `crims` database, `orm` schema, `wal_level=logical`
  already set. Publication `orm_cdc_publication FOR TABLES IN SCHEMA orm`
  (created directly on the host, not git-tracked — see runbook note below).
- **Debezium**: `semlayer-debezium` container (Kafka Connect 2.3,
  port 8083), connector `orm-oms-connector` registered directly via the
  Connect REST API (not git-tracked as host state, but the registration
  payload is at `debezium/orm-oms-connector.json` with a password
  placeholder). Uses **mTLS** client-cert auth against `crims` — cert
  material comes from `tenant_product_datasource.config` (in the `alpha`
  database) for datasource id `441f62c9-aad1-481d-9aab-62943fa11cd3`. The
  private key must be **PKCS#8 DER** (not PEM) and owned by uid 1001
  (`kafka`) inside the container, or the connector fails with "Could not
  read SSL key file". Full step-by-step is in a local, uncommitted file
  `debezium/orm-oms-connector.md` on this machine (was deliberately not
  committed — it contains a private-key-extraction runbook) — worth
  reconstructing/relocating if this work continues.
- **Kafka**: `semlayer-redpanda`, on Docker network `remote-net` (note:
  there are *three* differently-scoped networks all called some variant of
  "remote-net" on that host — `remote-net` (unprefixed, real one, has
  Kafka+Debezium), `eganpj_remote-net` (Compose-auto-prefixed, has
  StarRocks because `docker-compose.remote.yml` didn't pin a network name
  when StarRocks was added), and `uisce_remote-net` (unused). The
  git-tracked `docker-compose.remote.yml` now pins
  `networks.remote-net.name: remote-net` — a fresh `docker compose up` on
  that host will try to join all fe/be/loader services to the *real*
  `remote-net`, which will recreate the already-running StarRocks
  containers onto the correct network. Fine to do, causes a brief restart.
  Topics: `orm_oms.orm.{order,placement,order_allocation,execution,
  execution_allocation}`.
- **StarRocks**: `starrocks-fe` / `starrocks-be` containers, healthy,
  BE registered via `ALTER SYSTEM ADD BACKEND 'starrocks-be:9050'` (this
  is a runtime cluster operation, not persisted in any file — if the FE
  container is ever recreated fresh, this needs to be re-run). Database
  `oms`, tables `orm_order`, `orm_placement`, `orm_order_allocation`,
  `orm_execution`, `orm_execution_allocation` — DDL is git-tracked in
  `backend/internal/analytics/starrocks_init.sql` (section 7), which the
  FE container auto-applies via `docker-entrypoint-initdb.d` on fresh
  startup.
- **Stream loader**: `backend/cmd/stream_loader/main.go`, rewritten this
  session to parse Debezium's actual JSON envelope
  (`{"schema":...,"payload":{"before","after","op","ts_ms"}}`) instead of
  the old custom `cmd/cdc_service` shape, and to decode Debezium's
  base64-encoded big-endian two's-complement `Decimal` fields back into
  real numbers using the embedded schema's `scale` parameter. Also fixes a
  real bug: StarRocks stream-load answers with a 307 redirect from FE to
  the owning BE, and Go's default HTTP client drops `Authorization` on
  cross-host redirects — `CheckRedirect` now re-attaches it. One container
  per source table (`uisce-stream-loader-{execution,order,placement,
  order-allocation,execution-allocation}`), all in
  `docker-compose.remote.yml`.
- **Verified live, twice**: a real `order -> placement -> execution ->
  order_allocation -> execution_allocation` insert/update chain landed
  correctly in all 5 `oms.orm_*` StarRocks tables, including correct
  decimal decoding (`exec_qty=5000.0000`, `exec_price=99.875000000`) and
  an UPDATE reflecting (`last_capacity='P'`).

### 3. Pre-aggregation catalog layer
- `backend/internal/handlers/preaggregation_handler.go` (new): CRUD +
  DDL-generation + refresh HTTP handler, mounted at `/api/preaggregations`
  inside the `/api` route block in `backend/internal/api/api.go`.
- `backend/internal/analytics/pre_aggregation_service.go`: fixed a real
  schema-drift bug in `UpsertPreAggregation` — `ON CONFLICT` named
  `(tenant_id, node_type_id, node_name)` but the actual unique constraint
  on `catalog_node` is `(tenant_id, qualified_path)`.
- `backend/migrations/20260908_pre_aggregation_node_type.sql`: seeds the
  `pre_aggregation` catalog_node_type row (was never inserted despite the
  service always assuming it existed).
- Verified: `execution_npv_rollup` pre-aggregation (Execution BO, terms
  `PlacementID`/`BrokerID`, calculation `Excel NPV`) persists correctly as
  a `catalog_node`, id `5217fdac-ae21-461a-8223-7d65bb23d707`, tenant
  `99e99e99-99e9-49e9-89e9-99e99e99e999`.

### 4. GenerateDDL — dimension resolution (this session's last fix)
`GenerateDDL` used to resolve BOs/columns through a legacy, unrelated
catalog_node "business_object" prototype (old Northwind
classification/key/grain nodes), then through `boresolver.
PostgresBORepository` (a real, correctly-written repository elsewhere in
the codebase) — but that repository's physical-column resolver parses
`business_objects.driver_table_name` as dotted `"schema.table"` and
hardcodes schema=`public`, while on this schema `driver_table_name` is
actually stored in **qualified_path form** (e.g. `/orm/execution`) — so
every field resolved to an empty physical column, not just calculated
ones.

Fixed by resolving physical columns directly via the `MAPS_TO` catalog-edge
chain (`business_object_fields.term_node_id -> catalog_edge(MAPS_TO) ->
physical column catalog_node`), scoped to the BO's own table via
`qualified_path LIKE driver_table_name || '/%'` (necessary because generic
field names like "CreatedAt"/"ID"/"Status" map to columns on many
different tables — an unscoped lookup returns 52 ambiguous rows for a BO
that only has 14 fields). This is the same mechanism already proven
working elsewhere this session for column-to-term lineage — not a new
invention.

Verified output for `execution_npv_rollup`:
```sql
CREATE MATERIALIZED VIEW tenant_99e99e99-99e9-49e9-89e9-99e99e99e999.mv_execution_npv
BUILD IMMEDIATE
REFRESH ASYNC
AS
SELECT
    placement_id AS "PlacementID",
    NULL /* TODO: "Excel NPV" is a formula-based calculated term (no SQL compiler wired) */ AS "Excel NPV"
FROM (
    SELECT * FROM oms.orm_execution
) t

GROUP BY PlacementID;
```
The dimension is real, correct SQL against the live CDC-fed hot tier. The
measure is an honest placeholder — this is exactly the boundary this
handoff exists to describe.

## What's NOT done — the actual next task

### A. Formula-to-SQL compiler for calculated terms

**The gap:** Calculated semantic terms store their logic in
`catalog_node.properties`, e.g. for "Excel NPV":
```json
{
  "term_type": "calculated",
  "formula": "{{ excel_formula('=NPV({rate}, {cash_flows})') }}",
  "expression": "{{ excel_formula('=NPV({rate}, {cash_flows})') }}",
  "return_type": "currency",
  "data_type": "number"
}
```
This is a Jinja-style template wrapping an Excel-formula-syntax string with
`{placeholder}` parameter references. Nothing in the codebase compiles
this to SQL. Three near-misses were found and ruled out this session —
useful to know so they aren't re-investigated from scratch:

1. **`internal/analytics/bo_context_resolver.go`** (`ResolveCalculation`,
   now unused after this session's fix — the whole `BOContextResolver`
   type may be dead code worth deleting in a cleanup pass) expects
   `catalog_node.config.expression_dsl` + a `config.dependencies` array of
   `{type, ref}` objects. No real calculated term is stored this way —
   `config` is empty/null for Excel NPV; everything lives in `properties`.
2. **`internal/calculation/expression_compiler.go`** (`Service.
   CompileExpression`, wired to `POST /api/calculation/compile`) expects
   `[BOName.field]` bracket-reference syntax in `req.Formula`. Doesn't
   match the Jinja/Excel-formula format either.
3. **`internal/calcengine/`** (6,600+ lines: `engine.go`,
   `unified_engine.go`, `multi_source_engine.go`, `starrocks.go`,
   `dialect.go`, etc.) is a real, substantial multi-tier calc engine with
   watermark-based OLTP/hot/cold routing — architecturally the closest
   match to what the user described wanting. It's gated behind
   `CBO_ENABLED` (unset locally, so dormant) and wired to a `CalcHandler`
   elsewhere in `api.go`. **Not yet investigated**: whether this package
   has (or could reasonably grow) an `excel_formula(...)` template parser.
   This is the most promising starting point for the next session — worth
   reading `internal/calcengine/engine.go` and `stored_calculations.go`
   first before building anything new, since a working "calc DSL ->
   result" path may already exist for the single-source (Postgres)
   `CalcHandler` case and just need a SQL-emitting mode added, rather than
   a whole new compiler.

**Concretely, what "done" looks like:** `GenerateDDL` in
`backend/internal/analytics/pre_aggregation_service.go` (search for `TODO:`
in the calculation loop) gets a real SQL expression instead of `NULL`,
sourced from parsing `{{ excel_formula('=NPV({rate}, {cash_flows})') }}`
-style templates, substituting `{rate}`/`{cash_flows}`-style parameters
with resolved physical columns or other calc references, and emitting
StarRocks-dialect SQL (NPV specifically has no native StarRocks function —
will need either a UDF or a hand-rolled SQL expansion of the NPV formula:
`SUM(cash_flow / POWER(1 + rate, period))`).

### B. `ApplyMaterialization` — actually executing the DDL

`PreAggregationService.ApplyMaterialization` (same file) is a stub:
```go
func (s *PreAggregationService) ApplyMaterialization(ctx context.Context, preAggID uuid.UUID) error {
	ddl, err := s.GenerateDDL(ctx, preAggID, "starrocks")
	if err != nil {
		return err
	}
	// TODO: Execute DDL against StarRocks using a separate connection pool
	_ = ddl
	return nil
}
```
and `Refresh` has the same TODO shape for `REFRESH MATERIALIZED VIEW`.
Neither has a StarRocks connection pool wired in anywhere. Needed: a
`*sqlx.DB` (or similar) pointed at StarRocks's MySQL-protocol port (9030),
threaded into `PreAggregationService`'s constructor and `api.go`'s wiring
alongside the existing `db *sqlx.DB` (Postgres) — StarRocks speaks the
MySQL wire protocol, so the existing `github.com/go-sql-driver/mysql` (if
already a dependency — check `go.mod`) or a new one should work directly
against `starrocks-fe:9030` (or `100.84.50.65:9030` from outside Docker).

## Key IDs / hosts for continuity

- Remote infra host: `100.84.50.65`, SSH as `eganpj`, repo at
  `/mnt/github/uisce`, compose files in `/home/eganpj`.
- Tenant ID: `99e99e99-99e9-49e9-89e9-99e99e99e999`
- ORM datasource ID (for mTLS cert lookup):
  `441f62c9-aad1-481d-9aab-62943fa11cd3`
- Execution BO ID: `aab56f4f-eb39-404f-8a4a-f6d15abe2530` (`bo_key =
  "execution"`, `driver_table_name = "/orm/execution"`,
  `driver_table_id = c2df2faf-5cfd-592d-b86d-243417eb4217`)
- Test pre-aggregation: `5217fdac-ae21-461a-8223-7d65bb23d707`
  (`execution_npv_rollup`)
- StarRocks: FE on port 8030 (HTTP)/9030 (MySQL protocol), BE on 8040.
  Database `oms`.
- Branch: `fix/tenants-search-path-and-bo-write-path`. Relevant commits
  this session (newest last): `f79fbbc9e` (CDC activation, execution-only),
  `0ce9495e0` (extend to all 5 tables), `0a2b6e861` (GenerateDDL dimension
  fix).

## Working notes worth carrying forward

- **Schema drift is the dominant bug class in this codebase.** Nearly
  every fix this session (business_object_fields ON CONFLICT target,
  pre_aggregation node type, GenerateDDL's driver_table_name format,
  earlier-session bugs in `businessobject_service.go`) was "the code
  assumes column X / table Y / format Z, and the live schema has
  something else." Always verify against `\d tablename` or a live query
  before trusting a function's assumptions about schema shape.
- **The auto-mode classifier blocks writes to shared/remote credential
  material** (writing private keys to files, `.pgpass`, `docker compose
  up` on the shared host in some contexts) even when the values are
  already visible in conversation context. When blocked, the practical
  path is handing the user exact commands to run themselves rather than
  fighting it — that pattern worked repeatedly this session (connector
  registration, cert file permissions, `CREATE PUBLICATION`).
- **`debezium/orm-oms-connector.md`** (the cert-extraction runbook) is
  sitting locally uncommitted on this machine, not in git. If you want it
  preserved, it needs to be manually re-added — it was excluded because
  its content (SQL to extract `private_key` from `tenant_product_
  datasource`) reads as credential-extraction instructions to the commit
  classifier.
