# Step 0 — Pre-Flight Report

**Date:** 2026-09-23
**Status:** Step 0 COMPLETE. Deliverables 2–6 of the launch sequence produced. **Deliverable 1 (delta classification) BLOCKED** — base classification file does not exist on disk or in either database.

**Artifacts:**

| File | Contents |
|---|---|
| `preflight_alpha.sql` / `.out` | Read-only query pack run against `alpha` |
| `preflight_crims.sql` / `.out` | Read-only query pack run against `crims` |
| `preflight_crims_supplement.out` | Supplementary crims.mdm table inventory (added by agent) |
| `step0_alpha_mdm_tables.txt` | 314 alpha.mdm table names |
| `step0_alpha_oms_tables.txt` | 14 alpha.oms table names |
| `step0_alpha_edm_tables.txt` | 72 alpha.edm table names |
| `step0_crims_mdm_tables.txt` | 83 crims.mdm table names |
| `step0_name_collisions.txt` | Exact-name intersection (2 tables) |
| `step0_fk_inventory.tsv` | All 394 FKs in alpha.mdm, with definitions |

---

## Report template — filled

```
0.1  alpha.mdm table count:            314          ✓ matches assumption
0.1b alpha.mdm full table list:        314 names captured → step0_alpha_mdm_tables.txt
0.2  alpha.oms tables:                 14           (expected ~10–15)
0.3  alpha.edm tables:                 72           (expected 5–15 — MUCH larger)
0.4  classification_scheme exists:     NO (both to_regclass NULL)
0.5  crims.edm.issuer_master exists:   NO  — crims has NO `edm` schema at all
     crims.mdm.issuer_master exists:   NO
     crims.mdm.product exists:         NO
     crims.mdm.counterparty exists:    NO
     crims.mdm.benchmark_master exists:NO
     crims.mdm.calendar_master exists: NO
     crims.mdm.ca_event exists:        NO
     crims.mdm.price exists:           NO
0.6  crims.mdm.party row count:        0 rows, 0 tenants  → FULL SEED REQUIRED
0.6b crims.mdm table count:            83           ← NOT empty; has its own content
0.6c crims schemas:                    cash_flow, mdm, orm, public, ref, vend, wlth (7)
                                        ref/vend/wlth have 0 tables
                                        public=654, orm=78–80, mdm=83, cash_flow=1
0.7  alpha.mdm FK count:               394
0.8  cross-schema FK count:            8   (edm=7, public=1; oms=0, orm=0)
     target schemas (deduped):         edm, public
0.9  FDW (deferred to Step 2):         — no extension, no server exists yet
0.10 alpha.public tables:              955
```

---

## Item-by-item against the launch-sequence deliverables

### 1. Delta classification (~53 unclassified) — **BLOCKED**

The base 261-row classification file (`001_migration_plan.sql`) **does not exist**:

- Not on disk anywhere in the repo (searched `**/*001_migration*`, `migration/**`, `**/*classification*`, grep for `FABRIC_REF`, `FABRIC_RULE`, `INSERT INTO migration.plan`, `source_schema.*source_table` — all empty).
- Not in either database: **no `migration` schema exists on `alpha` or `crims`**; `to_regclass('migration.plan')` is NULL on both.
- `backend/docs/migration_classification_60.csv` is unrelated (replay results for 60 orphaned migration files, CLEAN/ERROR).
- Root-level `migration_script.sql`, `phaseb_*.sql`, `migration_to_ddl_schema.sql` are unrelated legacy artifacts.

The classification was delivered as a chat artifact in a previous session and was never persisted. The ~53-table diff cannot be computed without it.

### 2. alpha.oms additions — **PRODUCED**

`alpha.oms` has exactly 14 base tables:

```
account, allocation, execution, migration_log, order_event, order_link,
order_slice, orders, position, position_lots, security, settlement,
subtype_registry, trade_order
```

- `subtype_registry` ✓ exists → stays in `alpha.oms` (control plane), METADATA, order 100.
- `migration_log` ✓ exists → stays in `alpha.oms` (runner ledger), METADATA, order 100.
- **`subtype_registry_seed` does NOT exist.** The proposed delta row was a phantom — removed.
- STI wrapper tables (`account`, `position`, `security`, `trade_order`, `settlement`) present as expected.
- Additional unclassified oms tables to place: `allocation`, `execution`, `order_event`, `order_link`, `order_slice`, `orders`, `position_lots` (7 tables).

### 3. alpha.edm additions — **PRODUCED**

`alpha.edm` has **72** base tables (expected 5–15 — the domain is far larger than the plan assumed). Full inventory: `step0_alpha_edm_tables.txt`.

Notable families inside `alpha.edm`:

| Family | Count | Examples |
|---|---|---|
| `issuer_*` satellites | **0** — only `issuer_master` | (crims holds the satellites — see G19) |
| `mdm_*` golden/lineage | 6 | `mdm_calendar_golden`, `mdm_source_registry`, `mdm_survivorship_policies` |
| `portfolio_*` | 6 | `portfolio_master`, `portfolio_golden`, `portfolio_hierarchy` |
| `position_*` | 5 | `position_master`, `position_lot_master`, `position_snapshot_master` |
| `*_gold_trace` / `*_lineage` | 8 | `cash_gold_trace`, `position_gold_trace`, `price_gold_trace`, `security_gold_trace`, `transaction_gold_trace`, `gold_copy_lineage`, `rule_lineage`, `scenario_lineage` |
| `cash_*` | 5 | `cash_ledger`, `cash_balance_master`, `cash_position_master` |
| `compliance_*` | 4 | `compliance_rule`, `compliance_evaluation`, `compliance_breach` |
| masters | several | `benchmark_master`, `curve_master`, `fx_rate_master`, `security_master`, `strategy_master`, `transaction_master`, `vol_surface_master`, `mandate_master`, `price_master` |
| infra | several | `async_jobs`, `etl_run`, `scheduled_jobs`, `job_items`, `job_exports`, `wasm_module_version` |

Only `edm.issuer_master` has FKs into `alpha.mdm` (7 of them). No other edm table is referenced from mdm.

### 4. FK drop/recreate list — **PRODUCED** → `step0_fk_inventory.tsv`

- **394 total FKs** inside `alpha.mdm`.
- **8 cross-schema FKs** (the only ones that must be dropped/re-pointed):

| Constraint | From | To | Action after move |
|---|---|---|---|
| `fk_cade_issuer` | `mdm.ca_default_event` | `edm.issuer_master` | rewrite → `mdm.issuer_master` |
| `fk_cae_issuer` | `mdm.ca_event` | `edm.issuer_master` | rewrite → `mdm.issuer_master` |
| `fk_cam_issuer` | `mdm.ca_meeting` | `edm.issuer_master` | rewrite → `mdm.issuer_master` |
| `fk_cpty_issuer` | `mdm.counterparty` | `edm.issuer_master` | rewrite → `mdm.issuer_master` |
| `fk_prd_manager` | `mdm.product` | `edm.issuer_master` | rewrite → `mdm.issuer_master` |
| `fk_prd_subadv` | `mdm.product` | `edm.issuer_master` | rewrite → `mdm.issuer_master` |
| `fk_prd_dist_dist` | `mdm.product_distribution` | `edm.issuer_master` | rewrite → `mdm.issuer_master` |
| `fund_hierarchy_tenant_id_fkey` | `mdm.fund_hierarchy` | `public.tenants` | **cross-DATABASE after move — unenforceable. Decision required (G18).** |

- **Zero FKs** from `alpha.mdm` to `oms.*` or `orm.*`. The pre-registered expected list (oms.subtype_registry, orm.issuer, orm.account, orm.benchmark, orm.corporate_action) was **wrong on all five counts**.
- The remaining **386 intra-mdm FKs** move with the data and are preserved as-is (same-schema in `crims.mdm`).

### 5. Corrected `target_schema` for `issuer_master` — **DECIDED**

`crims` has **no `edm` schema** (7 schemas only: cash_flow, mdm, orm, public, ref, vend, wlth). Therefore:

**→ `crims.mdm.issuer_master`.**

Consequence: all 7 cross-schema FKs become **same-schema** (`mdm` → `mdm`) after the rewrite. One fewer schema to create, one fewer class of cross-schema constraint.

(The original rationale — "crims already has an edm schema, but it's not clear what shape it has" — is void: the schema does not exist.)

### 6. `crims.mdm.party` insert step — **REQUIRED**

`crims.mdm.party` = **0 rows, 0 tenants.** Full seed path (Step B + Step C from G4) is mandatory before any party satellite insert. `migration.party_xref` will be seeded by identity match (`tenant_id` + `party_cd`).

---

## New gaps surfaced by Step 0 (G13–G21)

| ID | Finding | Severity | Status |
|---|---|---|---|
| **G13** | `crims.mdm` already has **83 tables** — not an empty target. 81 have no alpha counterpart (incl. 20 `issuer_*` satellites, 38 `security_*` tables, `portfolio*`, `mandate*`, `dq_*`, `classification_scheme*`). Only **2 exact name collisions**: `party`, `rating_scale`. Two near-collisions: `source_system`/`source_systems`, `match_rule`/`match_rules`. | HIGH | Needs reconciliation rules per pair |
| **G14** | `classification_scheme` + `classification_scheme_map` exist in **crims**, not alpha (0.4 returned NULL). Any base-classification row targeting `alpha.mdm.classification_scheme` was a phantom pointing at the wrong DB. | MED | Correct target = keep in crims (no action) |
| **G15** | `oms.subtype_registry_seed` does not exist. Delta row would create a phantom plan entry. | LOW | **Row removed** |
| **G16** | `crims` has no `edm` schema → `issuer_master` target = `crims.mdm` (settles G6). | MED | **Decided** |
| **G17** | Only **8** cross-schema FKs total (7 edm + 1 public); **0** to oms/orm. Much simpler than the plan's expected list. | INFO | FK list produced |
| **G18** | `fund_hierarchy.tenant_id → public.tenants` becomes **cross-DATABASE** after the move. Postgres cannot enforce it. Options: (a) drop the FK and enforce in app, (b) mirror `tenants` into `crims.public`, (c) leave `fund_hierarchy` in alpha. | HIGH | **Decision required** |
| **G19** | The **issuer domain is split across both DBs today**: `alpha.edm.issuer_master` (master) vs `crims.mdm.issuer_*` (20 satellites, no master, no FKs by design per `0001_mdm_security.sql` header). Moving `issuer_master` to `crims.mdm` reunites them. | INFO | Supports the G6 decision |
| **G20** | **The base 261-row classification file was never persisted.** Not on disk, not in either DB. Deliverable 1 of the launch sequence is unproducible without rebuilding it. | **BLOCKER** | **Decision required** |
| **G21** | `crims.mdm.xref` already exists — may be the pre-existing cross-reference mechanism and could subsume `migration.party_xref`. | MED | Inspect before building xref |
| **G22** | `alpha.public` already has **plural** fabric tables: `business_objects`, `business_object_fields`, `business_object_relationships`, `business_object_binding`, `catalog_node`, `catalog_edge`, `semantic_terms`, `field_bindings`, `metadata_*`. The plan's Step 4 used **singular** names (`business_object`, `business_object_field`, `business_object_field_binding`, …). Collision/naming reconciliation required before Step 4. | HIGH | **Decision required** |

### Corrections to earlier assumptions

| Assumption | Reality |
|---|---|
| crims.mdm is empty | **83 tables already present** with substantive content |
| classification_scheme is in alpha.mdm | It's in **crims.mdm**; alpha has neither table |
| subtype_registry_seed exists | Does not exist |
| crims has an edm schema | Does not exist |
| ~5–15 edm tables | **72** |
| FKs to oms.subtype_registry / orm.* expected | **Zero such FKs exist** |
| ~200–400 mdm FKs | **394** (within range) |

---

## Environment readiness (observed, not modified)

- `postgres_fdw`: **not installed** on alpha or crims. No FDW server exists. Step 2 must `CREATE EXTENSION postgres_fdw` on crims (the side that hosts the server).
- `postgres` role: superuser + BYPASSRLS on alpha (FDW owner privileges available).
- mTLS certs: present at `$HOME/.uisce/certs/{ca.crt,postgres-client.crt,postgres-client.key}`; both sessions connected with `sslmode=verify-full` — **mTLS works for both DBs**. G9's cert concern is resolved pre-emptively: no SSL errors occurred on either connection.
- Both DBs live on the same host `100.84.50.65:5432`; only the dbname differs.

---

## Blocked — decision required (G20)

Deliverable 1 (the consolidated classification: 261 base rows + delta) cannot be produced because the base does not exist. Two paths:

**(a) User provides the original file** — from a chat export, another machine, a gist, or a paste. Fastest if it survives anywhere.

**(b) Rebuild the classification from scratch** — classify all 314 alpha.mdm + 14 oms + 72 edm tables using the now-complete evidence:
- full FK graph (394 edges, `step0_fk_inventory.tsv`)
- domain prefixes visible in the table names (benchmark_/ca_/calendar_/counterparty_/party_/price_/product_/rating_/recon_/settlement_/curve_/vol_/fx_/sanctions_/kyc_/fatca_/client_group_/security…)
- both DBs' inventories (collision detection built in)
- the settled rules: FABRIC_REF → `alpha.public.ref_*`, FABRIC_RULE → `alpha.public.*`, DATA → `crims.mdm`, DROP, METADATA stays in alpha.oms
- the §2/§3 additions already corrected above

Rebuild is strictly better than the original in one respect: the original had no FK graph and no crims inventory, which is how the phantom rows (G14, G15) and the wrong FK expectations (G17) got in.

Recommendation: **(b)**, unless the original file is recoverable in the next step.
