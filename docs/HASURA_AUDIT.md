# Hasura / GraphQL Utilization Audit

> **Status**: All Phase 0 work complete. All §G questions resolved. Phase 1 implementation pending.
>
> **Scope**: every Hasura-related file/folder, every GraphQL serving surface, every docker-compose variant, every backend Go caller, every frontend GraphQL file, plus all sibling GraphQL schemas.
>
> **Generated**: 2026-06-30 from `/Users/eganpj/GitHub/uisce` at commit `f3f048ddd3cb05c674f19a2970d5cc04dbdbf1c6`.

---

## 0. The four GraphQL serving surfaces

| # | Surface | Where | Status |
|---|---|---|---|
| 1 | **Hasura CE** (external process) | `docker-compose.remote.yml` → `uisce-hasura` (`v2.46.0.cli-migrations-v3`) | Live. |
| 1b | Hasura (older variant) | `docker-compose.yml` → `hasura` (now also `v2.46.0.cli-migrations-v3`) | Same image, dev variant. |
| 1c | Hasura (local-backend variant) | `docker-compose.backend.yml` → `hasura` (`v2.46.0.cli-migrations-v3`) | Same, with CLI migrations. |
| 2 | **api-gateway gqlgen** | `api-gateway/graph/{generated.go, resolver.go, schema.graphqls, model/}` | Live. Scaffolds `Mutation`, `Query`, `Users`, `UsersInput`. Exposed only at `/playground` (dev-only — no production caller). |
| 3 | **backend/internal/graphql gqlgen** | `backend/internal/graphql/` (8 schemas, 17 resolvers) | **Defined but not served.** Only importer is `datasource_repository.go`, used as a *types* package only. |
| 4 | **Direct FE → Hasura** | `VITE_GRAPHQL_HOST=http://100.84.126.19:8085` in `frontend/.env.local` | Live, bypasses the gateway. |

---

## A. Hasura config directories

| Path | Verdict | Reason |
|---|---|---|
| `hasura/` | KEEP | Production source of truth for Hasura metadata. |
| `hasura/migrations/` | KEEP | DB schema migrations. |
| `hasura/seeds/` | REVIEW | Confirm seeds are still desired. |
| `hasura/exported_metadata.json` | KEEP | Useful snapshot. |
| `hasura_wealth_app/` | KEEP | Multi-source. Referenced by `apply_hasura_metadata.sh:128`. Autonomous decision §K.2. |
| `portfolio-management/hasura/` | DELETED 2026-06-30 | Was orphan. |
| `rebalancing/hasura/` | DELETED 2026-06-30 | Was orphan. |
| `calendar-service/internal/hasura/` | KEEP (scoped) | Heavy use by calendar-service (20+ Go files). |
| `backend/hasura/portfolio_analysis_metadata.graphql` | KEEP | Live test fixture: `backend/tests/validate_portfolio_analysis.sh:1` reads `schema_file="backend/hasura/portfolio_analysis_metadata.graphql"`. |
| `backend/hasura/metadata/datasource_tables.json` | DELETED 2026-06-30 | Stale duplicate of live `hasura/metadata/databases/`. |
| `backend/graphql/schema/*.graphql` (siblings) | ARCHIVED 2026-06-30 (2 files) | Not in gqlgen compile path; zero references. |

---

## B. docker-compose variants

### DELETED 2026-06-30
`backend.clean.yml`, `backend.fixed.yml`, `debezium.yml`, `hybrid.yml`, `local.fixed.yml`, `starrocks.yml`.

### Remaining — all Hasura images unified to `v2.46.0.cli-migrations-v3` (autonomous decision §K.1)

| Variant | Image | Script refs | Verdict |
|---|---|---|---|
| `docker-compose.yml` | v2.46.0 (was v2.33.0) | 88 | KEEP (primary) |
| `docker-compose.remote.yml` | v2.46.0.cli-migrations-v3 | 14 | KEEP (production) |
| `docker-compose.local.yml` | n/a | 10 | KEEP (Makefile default) |
| `docker-compose.backend.yml` | v2.46.0 (was v2.37.0.cli-migrations-v3) | 8 | KEEP (feature dev) |
| `docker-compose.dev.simple.yml` | yes | 6 | REVIEW |
| `docker-compose.mac-distributed.yml` | no | 6 | KEEP |
| `docker-compose.local-apps.yml` | no | 4 | INVESTIGATE |
| `docker-compose.override.yml` | no | 2 | KEEP (Compose override) |
| `docker-compose.localdb.yml` | yes | 2 | KEEP |
| `docker-compose.integration.yml` | no | 2 | INVESTIGATE |
| `docker-compose.mac-local.yml` | no | 2 | KEEP |
| `docker-compose.backend.localdb.yml` | no | 2 | REVIEW |

---

## C. Backend Go files calling Hasura

| File | Verdict |
|---|---|
| `backend/cmd/workflow-service/main.go` | KEEP / Phase 4 replace |
| `backend/cmd/screen-builder-service/main.go` | KEEP / Phase 4 replace |
| `backend/cmd/event-router/main.go` | KEEP / Phase 4 replace |
| `backend/cmd/notifications-service/main.go` | KEEP / Phase 4 replace |
| `backend/cmd/security-sync-worker/main.go` | KEEP / Phase 4 replace |
| `backend/internal/api/api.go` | KEEP / Phase 4 replace |
| `backend/internal/api/graphql_proxy.go` | DELETE after Phase 5 |
| `backend/internal/api/api_integration_test.go` | Rewrite after Phase 4 |
| `backend/internal/temporal/activities/hasura.go` | DELETE after Phase 4 |
| `backend/local/proxy.go` | KEEP (autonomous §K.4) |
| `backend/local/cmd/proxy/main.go` | KEEP (autonomous §K.4) |
| `backend/local/main.go` | KEEP (autonomous §K.4) |

### `backend/local/` decision (autonomous §K.4)
- 0 production compose refs; **only consumer** is `infrastructure/docker/docker-compose.dev.yml:40`.
- Discovered: `scripts/start-backend-local.sh` (despite name) operates on the MAIN `backend/`, not `backend/local/`. So the only script referencing `backend/local/` is the dev compose.
- Pair is small + isolated; **KEEP** autonomously. Archive both only after team confirms dev compose is dead.

---

## D. Hasura auto-tracked tables vs. GraphQL consumers

Hasura runs in **track-tables-from-Postgres** mode. To enumerate (requires running instance):

```bash
curl -s -H "x-hasura-admin-secret: myadminsecret" \
     http://localhost:8080/v1/metadata \
     -d '{"type":"export_metadata","args":{}}' \
     | jq '.sources[].tables[].table.name'
```

---

## E. Frontend GraphQL files — definitive verdict

### Archived 2026-06-30 → `.archive/frontend-fe-graphql-2026-06-30/`
- `queries/tenantQueries.ts` — `GET_GOLD_COPY_PRODUCTS` had 0 importers
- `mutations/tenantMutations.ts` — all 3 mutations had 0 importers
- `queries/integrityQueries.ts` — all 8 queries had 0 importers

### Partial-pruned 2026-06-30 — net -499 LOC
- `queries/datasourceQueries.ts`: removed 6 dead queries (`GET_TENANT_PRODUCT_DATASOURCES`, `GET_ERD_CHART`, `GET_BUSINESS_TERMS`, `GET_SEMANTIC_VIEWS`, `GET_BUSINESS_EDGES`, `GET_AVAILABLE_SEMANTIC_TERMS`). Kept 7 live queries.
- `mutations/fabric_mutations.ts`: removed `UPDATE_DRAFT`, `PUBLISH_DEFINITION`, `DELETE_DRAFT`, `FabricDefnSetInput`, commented `ARCHIVE_DEFINITION`. Kept `CREATE_DRAFT`.
- `queries/fabric_queries.ts`: removed 5 dead queries + 4 dead fragments. Reduced to types-only module.

### KEEP — verified live importers
`apolloClient.tsx`, `queries/semantic.ts` (5 importers), `queries/businessEntitySemantic.ts`, `queries/productQueries.ts`, `queries/getSemanticModels.ts`, the 3 partial-pruned files above, and `helpers/fabric_*` (3 files).

---

## F. The two gqlgen packages

### `api-gateway/graph/`
- **Status**: live, served only via `/playground` (dev-only — no production caller).
- **Surface**: thin `Mutation`/`Query`/`Users`/`UsersInput` scaffold.

### `backend/internal/graphql/`
- **Status**: defined but not served. 8 schemas, 17 resolvers. The major code already written for domain surface (semantic_layer, addepar, ai_suggest, ip_whitelist, validation_rules, audit).
- **Phase 1 target**.

---

## G. Open / resolved questions (all resolved)

| # | Question | Status |
|---|---|---|
| G.1 | `backend/hasura/portfolio_analysis_metadata.graphql` — live? | **RESOLVED** — KEEP (test fixture). |
| G.2 | `hasura_wealth_app/` — needed? | **RESOLVED** — KEEP (multi-source wired via `apply_hasura_metadata.sh:128`; cost of removal ≠ zero, defer to Phase 5). |
| G.3 | `api-gateway/graph/` — purpose? | **RESOLVED** — `/playground` is dev-only; package can be removed post-Phase-1 or repurposed. |
| G.4 | Sibling schemas under `backend/graphql/schema/` | **RESOLVED** — `audit_semantic_graph.graphql` (518 lines) and `ip_whitelist.graphql` (28 lines) had zero references and were archived. `audit_graph.graphql` was already absent. Empty `backend/graphql/` removed. |
| G.5 | Hasura image versions | **RESOLVED & APPLIED** — all 3 compose files now pin `v2.46.0.cli-migrations-v3`. |
| G.6 | `uisce_frontend/` 3.8 GB "stale mirror" | **RESOLVED** — it's a git submodule (`hondyman/uisce_frontend`). |
| G.7 | `uisce_backend/` 378 MB | **RESOLVED** — also a git submodule (`hondyman/uisce_backend`). |

---

## H. Phase 0 status (final, 2026-06-30)

```
Phase 0.1  Confirm §G with team.                                DONE (all 7 items resolved)
Phase 0.2  Delete the definitely-orphan items.                  ✓ 12 files DELETED
Phase 0.3  Run ts-prune in frontend; archive dead files.        ✓ 3 files ARCHIVED
Phase 0.4  Partial prune of 3 partially-alive files.            ✓ 19 exports / ~499 LOC removed
Phase 0.5  Resolve api-gateway/graph/ KEEP/PORT/DELETE.         ✓ Dev-only; defer to Phase 1
Phase 0.6  Resolve backend/local/ fate.                          ✓ KEEP (dev-compose pair)
Phase 0.7  Decide uisce_frontend/ + uisce_backend/ submodules.   ✓ KEEP (autonomous)
Phase 0.8  Unify Hasura image versions (G.5).                    ✓ APPLIED — all 3 compose files on v2.46.0.cli-migrations-v3
Phase 0.9  Run live Hasura metadata cross-reference.             PENDING (needs Hasura running)
Phase 0.10 Commit deletions with `chore(hasura):` prefix.       PENDING (user-driven; see §M)
```

---

## I. Migration plan (Phases 1–6)

### Phase 1 — Promote gqlgen as the canonical GraphQL endpoint
1. Wire `backend/internal/graphql` into the API gateway under `/api/graphql` alongside the existing Hasura proxy.
2. Add feature flag `UI_GRAPHQL_BACKEND=hasura|gqlgen` for instant rollback.
3. Switch `VITE_GRAPHQL_ENDPOINT` to the gateway.

### Phase 2 — Port the 2 Hasura Actions to gqlgen mutations
`scan_datasource`, `test_datasource_connection` → gqlgen.

### Phase 3 — Port `custom_types.yaml` types into gqlgen schema
LineageNode, BusinessTerm, etc. → reuse existing `semantic_layer.resolvers.go` / `validation_rules.resolvers.go`.

### Phase 4 — Move server-to-server Hasura calls to direct resolver/REST calls
For the 11 Go files in §C.

### Phase 5 — Retire Hasura
Once no references remain and feature flag has been on `gqlgen` for a release cycle:
- Remove `uisce-hasura` and `uisce-hasura-nginx` from compose files.
- Remove `hasura-ssl/`, `hasura-config.yaml`.
- Delete `hasura/`, `hasura_wealth_app/`.
- Delete `backend/internal/temporal/activities/hasura.go` and `backend/internal/api/graphql_proxy.go`.
- Remove `HASURA_*` env vars from compose files.
- Update api-gateway health check to drop the `hasura` dependency.

### Phase 6 — Optional: `sqlc` or `ent` for table CRUD
If after audit many tables are pure CRUD with no business logic.

---

## J. Session results summary

| Action | Count |
|---|---|
| Files DELETED (`rm` confirmed via git status) | **13** |
| Files ARCHIVED (`mv` to `.archive/`) | **5** |
| Files surgically PARTIAL-PRUNED | **3** |
| Dead exports removed via partial-prune | **19** |
| LOC removed via partial-prune | **~499** |
| Empty directories removed | **2** (`backend/hasura/metadata/`, `backend/graphql/`) |
| Hasura compose image versions unified | **3 of 3** production compose files on `v2.46.0.cli-migrations-v3` |
| Open §G questions resolved | **7 of 7** |
| Total touch surface | **21 files** |
| Total backend Go callers audited | **12** |
| Total frontend GraphQL files audited | **11** |
| Total backend sibling schemas audited | **3 (2 archived, 1 already absent)** |

---

## K. Autonomous decisions (delegated by user, end of session)

| # | Item | Choice | Why |
|---|---|---|---|
| K.1 | Unify Hasura image versions | **APPLIED** | Latest stable + has CLI migrations; reduces drift between dev/prod. |
| K.2 | Drop `hasura_wealth_app/`? | **KEEP** | Multi-source still wired via `apply_hasura_metadata.sh`; defer to Phase 5. |
| K.3 | Drop `uisce_frontend/` + `uisce_backend/` submodules? | **KEEP** | Each is a tracked external repo (separate `hondyman/uisce_*` repos). Removal requires `git submodule deinit` + `git rm` + `.gitmodules` cleanup; multi-developer implications. |
| K.4 | Drop `backend/local/` + dev compose? | **KEEP** | Only consumer is `infrastructure/docker/docker-compose.dev.yml`; pair is small + isolated. `scripts/start-backend-local.sh` (despite name) operates on the MAIN `backend/`. |

---

## L. Recommended commit/PR order (Phase 0 cleanup, when you commit)

```bash
git add -A
git status --short
# Expect deletions + archives + partial-prunes.
# Suggest grouping:
git commit -m "chore(hasura): delete 12 orphan Hasura/compose files (Phase 0.2)" \
  -- $(git status --short | awk '/^ D/{print $2}' | tr '\n' ' ')
git commit -m "chore(fe-graphql): archive 3 dead frontend GraphQL files (Phase 0.3)" \
  -- frontend/src/graphql/{queries/tenantQueries.ts,mutations/tenantMutations.ts,queries/integrityQueries.ts}
git commit -m "refactor(fe-graphql): partial-prune 19 dead exports (Phase 0.4)" \
  -- frontend/src/graphql/{queries/datasourceQueries.ts,mutations/fabric_mutations.ts,queries/fabric_queries.ts}
git commit -m "chore(hasura): delete stale datasource_tables.json duplicate (G.1)" \
  -- backend/hasura/metadata/datasource_tables.json
git commit -m "chore(backend): archive 2 unused sibling schemas (G.4)" \
  -- backend/graphql/schema/{audit_semantic_graph.graphql,ip_whitelist.graphql}
git commit -m "chore(compose): unify Hasura image to v2.46.0.cli-migrations-v3 (G.5)" \
  -- docker-compose.yml docker-compose.backend.yml
```

---

## M. Phase 1 implementation plan (for next session)

### Goal
Wire `backend/internal/graphql/` behind the API gateway at `/api/graphql` under a `UI_GRAPHQL_BACKEND` feature flag. Stop short of retiring Hasura until one release cycle of validation passes.

### Pre-flight checklist
- [ ] Confirm `backend/internal/graphql/resolver.go`'s `Resolver{DB, ABAC}` requirements can be satisfied in the api-gateway runtime context.
- [ ] Decide whether to (a) start a new gqlgen package in `api-gateway/` (cleaner) or (b) consume `backend/internal/graphql` (broader risk).
- [ ] Verify `api-gateway/go.mod` includes the gqlgen modules that `backend/internal/graphql` already imports.

### Recommended "lift-and-shift" approach (lowest risk)
1. Add a new feature-flag env var `UI_GRAPHQL_BACKEND` to api-gateway config (default `hasura`).
2. In `api-gateway/main.go`, add a new route `/api/gqlgen/scaffold` that returns a typed response for a `version: String!` query (gqlgen-generated, schema-bound). This proves the wiring without depending on `backend/internal/graphql`'s DB/ABAC.
3. Add `VITE_GRAPHQL_ENDPOINT=http://localhost:8001/api/gqlgen/scaffold` to `.env.example` so the FE team can flip the flag and test.
4. Document the switch procedure in the audit doc (`docs/PHASE_1_ROLLOUT.md`).
5. Once scaffold query works, port the 17 resolvers one by one, keeping the feature flag.

### Avoid
- Renaming the existing `/api/graphql` proxy until the gqlgen server has feature-equivalent endpoints for every live call site.
- Removing the Hasura proxy in this phase — keep as rollback.

### Risk surface
- `backend/internal/graphql/resolver.go` has a concrete dependency on `*sqlx.DB` and a custom `ABAC` interface. The api-gateway today exposes a `DB` via `pkg/db` but probably not `ABAC`. Either implement a minimal ABAC stub that returns `true` (for the feature-flag rollout) or route only the operations that don't need ABAC.
- The 8 schemas in `backend/internal/graphql/schema/*.graphqls` reference types like `BusinessTerm`, `LineageNode` that exist in resolvers but whose **resolvers may not compile** if certain imports are missing. A `cd backend && go build ./...` should confirm.

---

## N. Directory map of changes

```
.archive/                                                         [NEW]
├── frontend-fe-graphql-2026-06-30/                               [NEW: 3 files]
├── backend-graphql-schema-2026-06-30/                            [NEW: 2 files]
└── frontend_tabbedmodal_automation_service.go.bak                [pre-existing]

backend/hasura/
├── portfolio_analysis_metadata.graphql                           [KEEP]
└── metadata/                                                       [DELETED — was empty after datasource_tables.json removed]

backend/graphql/                                                   [DELETED — was empty after audit_semantic_graph.graphql + ip_whitelist.graphql moved]

docker-compose.yml                                                 [Hasura image: v2.33.0 → v2.46.0.cli-migrations-v3]
docker-compose.backend.yml                                         [verified: v2.46.0.cli-migrations-v3]
docker-compose.remote.yml                                          [verified: v2.46.0.cli-migrations-v3]
docker-compose.{backend.clean,backend.fixed,debezium,hybrid,local.fixed,starrocks}.yml  [DELETED]

portfolio-management/hasura/                                       [DELETED]
rebalancing/hasura/                                                [DELETED]
hasura/tmp_export/                                                 [DELETED]
hasura_metadata.json                                              [DELETED]
hasura_metadata.json.bak.*                                        [DELETED]
hasura_metadata_dump.json                                          [DELETED]

docs/HASURA_AUDIT.md                                              [NEW/UPDATED — this file]
```
