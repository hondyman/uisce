# HANDOFF — SWIFT Settlement Process over Data Pipeline

**Authors:** Antigravity (2026-09-17)
**Status:** Phase 1 complete (inbound/outbound execution pending pipeline engine merge — nifty-greider-015b86). Full build clean, 18/18 tests pass (5 parser + 5 tile + 8 workflow). Workflow correctly drives to FAILED at VALIDATING until pipeline is wired; no fake-green SETTLED reachable. Migrations 001–010 applied to alpha.
**Task queue:** `bp_queue` (same as FIX)

---

## TL;DR

A SWIFT trade settlement process built on top of the existing FIX-over-pipeline architecture. Tenants configure SWIFT topology (BIC routing, field mappings, compliance rules, settlement calendars, enrichment lookups) entirely through database rows. No per-tenant code, no per-tenant recompile.

---

## File Locations

### Layer 4: Migrations
```
backend/db/migrations/20261001_001_swift_tenant_config.up.sql
backend/db/migrations/20261001_002_swift_field_map.up.sql
backend/db/migrations/20261001_003_swift_field_map_seed.up.sql   ← 24 gold-copy defaults
backend/db/migrations/20261001_004_swift_session_log.up.sql
backend/db/migrations/20261001_005_swift_compliance_rule_set.up.sql
backend/db/migrations/20261001_006_swift_reconciliation_report.up.sql
backend/db/migrations/20261001_007_cash_flow_settlement_swift_subtypes.up.sql
backend/db/seeds/20261001_swift_subtype_registry.sql
```

### Layer 1: SWIFT Gateway Adapter
```
backend/internal/swift/adapter.go        ← Adapter, TenantResolver, InboundSWIFTRecord, InboundSink
backend/internal/swift/mt_parser.go      ← ISO 15022 zero-dep MT parser (blocks 1/2/4)
backend/internal/swift/mx_parser.go      ← ISO 20022 XML parser (pacs.008/009, camt.056)
backend/internal/swift/admin_server.go   ← 127.0.0.1:8982, X-Swift-Admin-Token, send/cancel/health
```

### Layer 2: Pipeline Tiles
```
backend/internal/swift/tiles/tiles.go             ← TileFunc, TenantContext, Record
backend/internal/swift/tiles/decode.go             ← swift_decode
backend/internal/swift/tiles/field_map.go          ← swift_field_map  (GSIFI gold-copy precedence)
backend/internal/swift/tiles/compliance.go         ← swift_compliance (rulefabric)
backend/internal/swift/tiles/calendar.go           ← swift_calendar   (weekend check / HTTP service)
backend/internal/swift/tiles/enrich.go             ← swift_enrich     (declarative lookup specs)
backend/internal/swift/tiles/settlement_writer.go  ← swift_settlement_writer (cash_flow.settlement STI)
backend/internal/swift/tiles/swift_sender.go       ← swift_sender     (admin server dispatch)
backend/internal/swift/tiles/instruction_emit.go   ← swift_instruction_emit  (outbound MT/MX builder)
backend/internal/swift/tiles/tiles_test.go         ← 5/5 unit tests pass
```

### Layer 3: Temporal Workflows (bp_queue)
```
backend/internal/temporal/swift_session_lifecycle.go   ← SWIFTChannelLifecycleWorkflow
backend/internal/temporal/swift_settlement_workflow.go ← SWIFTSettlementWorkflow (T+2 state machine)
backend/internal/temporal/swift_reconciliation.go      ← SWIFTReconciliationWorkflow
backend/internal/temporal/swift_large_value.go         ← SWIFTLargeValueApprovalWorkflow
backend/internal/temporal/worker.go                    ← RegisterWorkflow + RegisterActivity (4+17)
```

### API / Server
```
backend/internal/api/swift_settlement_handlers.go  ← RegisterSWIFTRoutes(), 7 endpoints
backend/internal/api/api.go                        ← RegisterSWIFTRoutes called from SetupRouter (~line 1919)
backend/cmd/server/main.go                         ← SWIFT_ENABLE=true → startSWIFTGateway()
```

---

## 4-Layer Architecture

```
SWIFT Network (SWIFTNet FIN / GPI / file-drop)
        │ MT541/pacs.008 raw bytes
        ▼
Layer 1: SWIFT Gateway Adapter  (backend/internal/swift/adapter.go)
  • BIC pair → (tenantID, custodianID) from swift_tenant_config
  • Immediate ACK (MT0xx / pacs.002) via admin server
  • Emits InboundSWIFTRecord to per-tenant pipeline sink
  • Admin HTTP: 127.0.0.1:8982  X-Swift-Admin-Token
        │ InboundSWIFTRecord{raw_bytes, tenant_id, custodian_id, msg_type, uetr, ...}
        ▼
Layer 2: Data-Pipeline Tiles  (backend/internal/swift/tiles/)
  swift_decode → swift_field_map → swift_compliance → swift_calendar
              → swift_enrich → swift_settlement_writer
  (outbound path: swift_instruction_emit → swift_sender)
  All business logic in DB; expression engine for transform_fn and compliance rules.
        │ pipeline result + cash_flow.settlement row
        ▼
Layer 3: Temporal Workflows  (backend/internal/temporal/swift_*.go)
  SWIFTChannelLifecycleWorkflow   — long-running channel management
  SWIFTSettlementWorkflow         — T+2 settlement lifecycle state machine
  SWIFTReconciliationWorkflow     — periodic mismatch detection
  SWIFTLargeValueApprovalWorkflow — human-in-the-loop for large settlements
        │ GSIFI-scoped SQL; all on bp_queue
        ▼
Layer 4: Per-Tenant Config Tables  (all with GSIFI RLS)
  swift_tenant_config       — BIC, SAA endpoint, latency budget, recon interval
  swift_field_map           — tag→semantic; gold-copy defaults; tenant row wins
  swift_compliance_rule_set — rulefabric rule associations
  swift_session_log         — append-only audit
  swift_reconciliation_report — recon run output
```

---

## GSIFI Isolation

Read queries on `swift_*` config tables follow the gold-copy OR-clause so tenants inherit defaults:

```sql
WHERE (tenant_id = $1
   OR tenant_id = (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1))
```

`swift_field_map` additionally orders by `(tenant_id = $1) DESC NULLS LAST` so the tenant row wins over the gold-copy default. The gold-copy seed (`20261001_003_swift_field_map_seed.up.sql`) provides 24 rows covering MT541, MT543, MT548, pacs.008 — all tenants inherit these defaults and can override any mapping without code.

> **CRITICAL WRITE RULE:** The gold-copy OR-clause is for **READS ONLY**. Every `UPDATE`, `INSERT`, and `DELETE` must scope to `tenant_id = $N` exclusively. Writing with the OR-clause active means a workflow running for tenant A can mutate the gold-copy tenant's rows — that is a data-isolation violation, not a dev convenience. `PersistSettlementStatusActivity` was corrected for this in Sept 2026 (removed OR-clause from UPDATE WHERE). Never re-add it to writes.

---

## Settlement State Machine

```
RECEIVED
  → SWIFTAckActivity (MaximumAttempts: 1, 5s)
    ACK error → log warning, continue (ack ambiguity resolved by SWIFTReconciliationWorkflow)
  → VALIDATING
  → RunSWIFTPipelineDAGActivity (timeout = swift_tenant_config.pipeline_latency_budget_ms)
    Pipeline error → PersistSettlementStatusActivity(FAILED), workflow ends
  → MATCHED  (on SettlementUpdate signal status=matched)
  → PENDING_SETTLEMENT
  → SETTLED  (on SettlementUpdate signal status=settled)
           → PersistSettlementStatusActivity("SETTLED", tenant_id ONLY — no gold-copy OR)
  → FAILED   (on signal status=failed OR 72h deadline timer)
           → PersistSettlementStatusActivity("FAILED")
  → Cancel signal received
           → SWIFTRecallActivity (MaximumAttempts: 1 → MT192 / camt.056)
             Recall succeeds → PersistSettlementStatusActivity("CANCELLED")
             Recall fails    → PersistSettlementStatusActivity("CANCEL_PENDING")
                               SWIFTReconciliationWorkflow resolves CANCEL_PENDING rows
```

Workflow ID pattern: `swift-settlement-{tenantID}-{transactionRef}` — idempotent on re-submission.

---

## No-Code Configuration — Operator Reference

| Change | Mechanism | Restart required? |
|---|---|---|
| Add/modify SWIFT field mapping | INSERT/UPDATE `swift_field_map` | No |
| Add compliance/sanctions rule | INSERT `swift_compliance_rule_set` row + rule | No |
| Change settlement calendar | UPDATE `swift_tenant_config.settlement_calendar_id` | No |
| Add enrichment lookup | Edit `lookups` JSON in pipeline definition | No |
| Reorder/skip pipeline tiles | Edit pipeline DAG JSON | No |
| Add new transform function | Insert into rule engine function registry | No |
| Enable/disable SWIFT per custodian | UPDATE `swift_tenant_config.is_active` | No |
| Change large-value threshold | Update compliance rule expression | No |
| Connect new custodian | INSERT `master.vendor` + `swift_tenant_config` | No |
| Add new STI settlement subtype | INSERT `oms.subtype_registry` + run catalog sync | Migration only |

---

## Env Vars

| Var | Default | Purpose |
|---|---|---|
| `SWIFT_ENABLE` | unset | Set to `true` to boot admin server on startup |
| `SWIFT_ADMIN_ADDR` | `127.0.0.1:8982` | Admin HTTP server bind address |
| `SWIFT_ADMIN_TOKEN` | (required in prod) | Shared secret for X-Swift-Admin-Token header |
| `ENVIRONMENT` | (prod assumed) | Set to `development` for dev-only default admin token |

---

## REST Endpoints

All under `/api/cash-flow/swift/*`. Tenant extracted from JWT claims (standard path).

```
POST   /api/cash-flow/swift/instructions               → start SWIFTSettlementWorkflow
GET    /api/cash-flow/swift/instructions/{id}           → query from cash_flow.settlement
POST   /api/cash-flow/swift/instructions/{id}/cancel    → signal Cancel → MT192/camt.056 recall
GET    /api/cash-flow/swift/reconciliation/reports      → list recon reports (GSIFI-scoped)
POST   /api/cash-flow/swift/channels/{custodian}/start  → start SWIFTChannelLifecycleWorkflow [admin]
POST   /api/cash-flow/swift/channels/{custodian}/stop   → signal Stop to channel workflow [admin]
POST   /api/catalog/admin/sync-swift-subtypes           → trigger BO builder for SWIFT subtypes
```

Admin endpoints (`start`, `stop`) require `X-Swift-Admin-Token` header.

---

## Verification Commands

```bash
# Build
go build ./internal/swift/... ./internal/temporal/... ./internal/api/... ./cmd/server/...

# Tests (18/18 pass: 5 parser + 5 tile + 8 workflow)
go test ./internal/swift/... ./internal/temporal -count=1 -v

# Vet
go vet ./internal/swift/... ./internal/temporal/... ./internal/api/...

# Schema check (alpha DB)
PGPASSWORD=postgres psql -h localhost -U postgres -d alpha -c "
  SELECT tablename, schemaname, rowsecurity FROM pg_tables
  WHERE tablename LIKE 'swift_%' ORDER BY tablename;"

# Seed check
PGPASSWORD=postgres psql -h localhost -U postgres -d alpha -c "
  SELECT msg_type, count(*) FROM vend.swift_field_map GROUP BY msg_type ORDER BY msg_type;"

# Worker registrations
grep -E "SWIFT" backend/internal/temporal/worker.go | grep -v "^[[:space:]]*//"

# Routes registered at startup (check server logs for)
# ✅ SWIFT settlement routes registered (/api/cash-flow/swift/*)
```

---

## Known Gaps / Remaining Work

### 1. `RunSWIFTPipelineDAGActivity` — fail-loud stub (correct behavior)
**File:** `backend/internal/temporal/swift_settlement_workflow.go`
**Status:** Returns `ErrPipelineNotImplemented` (NonRetryable). Workflow drives to FAILED at VALIDATING; SETTLED is unreachable. Verified by `TestPipelineStubFails` (merge canary — read its comment before deleting it).
**Fix:** After `nifty-greider-015b86` merges, wire to pipeline engine `ExecuteDAG(ctx, dagID, record)`. Then delete `TestPipelineStubFails` and write the inverse test (MATCHED state reached). The tile implementations in `backend/internal/swift/tiles/` are complete.

### 2. `PersistSettlementStatusActivity` — IMPLEMENTED ✅
**File:** `backend/internal/temporal/swift_settlement_workflow.go`
**Status:** Done. Real `UPDATE cash_flow.settlement SET settlement_status = $2, updated_at = NOW() WHERE transaction_ref = $1 AND tenant_id = $3 AND valid_to IS NULL`. Tenant-scoped ONLY — the gold-copy OR-clause is intentionally absent from this write (see GSIFI Write Rule above). 0-rows-affected logs a warning (expected until pipeline wires real rows).

### 3. Frontend Studio palette
**File:** `frontend/src/features/data-pipelines/constants/pipelineTemplates.ts`
**Status:** Not touched — frontend scope.
**Fix:** Add `swift_*` tile entries under `category: "SWIFT"`. Add `SWIFT` to `PipelineMode` enum. Add trigger sources for `settlement_date`, `UETR`, `transaction_ref`.

### 4. Subtype catalog sync
**Status:** Migration 007 applied. Catalog BO graph not yet populated.
**Fix:** POST /api/catalog/admin/sync-subtypes with the gold-copy tenant UUID after server is deployed.

### 5. `GetInstruction` endpoint — IMPLEMENTED ✅
**File:** `backend/internal/api/swift_settlement_handlers.go`
**Status:** Done. Queries `WHERE transaction_ref = $1 AND tenant_id = $2 AND valid_to IS NULL`. Returns 404 on miss. Column `transaction_ref` landed in migration 008. Returns 200 with settlement details once the pipeline writes real rows.

---

## Don't-Do List (mirroring FIX §17)

- **Don't** instantiate the SWIFT admin HTTP client inside a workflow function. Only in activities.
- **Don't** use raw TCP for SWIFT — the adapter layer owns the transport connection.
- **Don't** use the GSIFI OR-clause on writes (`UPDATE`/`INSERT`/`DELETE`). Reads may include gold-copy fallback to inherit defaults; writes must scope to `tenant_id = $N` only. Gold-copy is a read-inheritance mechanism, never a write target.
- **Don't** use `MaximumAttempts > 1` on `SWIFTAckActivity` or `SWIFTRecallActivity` — outbound sends are not idempotent without explicit dedup at the custodian.
- **Don't** register SWIFT workflows on a task queue other than `bp_queue`.
- **Don't** add per-tenant SWIFT config files — use `swift_tenant_config` rows.
- **Don't** reintroduce CEL. The expression engine is `rulefabric/vm.go` — entry point `engine.EvaluateGroup`.

---

## Pre-Merge Checklist (mirrors FIX §20)

```bash
# 1. Build
go build ./...

# 2. Tests
go test ./internal/swift/... ./internal/temporal/... -count=1

# 3. RLS on all swift_* tables
PGPASSWORD=postgres psql -h localhost -U postgres -d alpha -c "
  SELECT tablename, rowsecurity FROM pg_tables
  WHERE tablename LIKE 'swift_%';"
# Expected: 4 tables with rowsecurity=t, 1 (session_log) with f

# 4. Worker registration
grep -c "SWIFT" backend/internal/temporal/worker.go

# 5. No top-level BEGIN/COMMIT in migrations
grep -l "^BEGIN;" backend/db/migrations/20261001_*.sql   # must be empty

# 6. Verify gold-copy seed applied
PGPASSWORD=postgres psql -h localhost -U postgres -d alpha -c "
  SELECT count(*) FROM vend.swift_field_map;"
# Expected: 24
```

---

## If Starting a New Session on This

1. Read `HANDOFF_FIX_OVER_PIPELINE.md` first — the SWIFT process mirrors it in every structural decision.
2. GSIFI is the contract. Every SQL in every tile and activity must include the gold-copy OR-clause.
3. The admin server (`127.0.0.1:8982`) owns the network connection. Workflows talk to it via HTTP activities — never directly.
4. Two idempotency keys:
   - **Inbound** (`swift_settlement_writer`): `(tenant_id, transaction_ref)` — SHA-256 of raw bytes as fallback when ref is absent. Unique index `settlement_transaction_ref_tenant_uniq` is the arbiter (migration 009).
   - **Outbound** (`swift_sender`): `(tenant_id, transaction_ref, custodian_id)` — `X-Idempotency-Key` header on admin POST.
5. `RunSWIFTPipelineDAGActivity` is a fail-loud stub — NonRetryable error, drives workflow to FAILED at VALIDATING. Do not ship to production without wiring to the pipeline engine (§Known Gaps #1). The merge-canary test `TestPipelineStubFails` turns red the moment the stub is removed — that's correct behavior.
6. CEL is gone. Don't reintroduce it.
7. Everything flows through the data-pipeline tile layer and Temporal workflows. No shortcuts.
8. **Driver contract:** `*sql.DB` in cmd/worker and cmd/server uses `lib/pq` (driver name `"postgres"`). DB errors surface as `*pq.Error`, not `*pgconn.PgError`. The 23505 UETR retransmission guard in `adapter.go` and `tiles/session_log.go` uses `*pq.Error`. If the driver ever migrates to pgx-native, update `pgerrors.go`.
9. **Session logging is at Layer 1** (adapter, not tile). `Adapter.logSession()` writes to `vend.swift_session_log` and discards UETR retransmissions independently of the pipeline DAG engine. `NewSessionLogTile` is the post-merge tile version for DAG-based discard; both share the same constraint name `swift_session_log_uetr_tenant_uniq`.
10. **Recall failure → CANCEL_PENDING, not CANCELLED.** A workflow cannot write terminal CANCELLED without confirmation from the custodian. `TestCancelRecallFails` is the regression test — if it goes red, the "failed recall lies as CANCELLED" bug has returned.
11. **Check-constraint gap (fixed):** migration 010 extends `ck_cash_flow_subtype` to include `dvp_securities` and `free_of_payment`. Any attempt to INSERT a SWIFT settlement before 010 is applied produces a check-constraint violation. Smoke test: transactional INSERT+ON CONFLICT against alpha per §Verification Commands.

---

## Merge-Day Runbook — nifty-greider-015b86 → main

Run these steps **in order** on merge day. Each step has a verification gate; do not proceed if the gate fails.

### Step 1 — Pre-merge: verify baseline

```bash
go build ./...
go test ./internal/swift/... ./internal/temporal/... -count=1
# Expected: all packages ok. TestPipelineStubFails is PASS (green stub = unmerged).
```

### Step 2 — Merge and rebuild

```bash
git merge nifty-greider-015b86
go build ./...   # must still be exit 0
```

### Step 3 — Wire `RunSWIFTPipelineDAGActivity`

**File:** `backend/internal/temporal/swift_settlement_workflow.go`

Replace the stub body with the real call. The entry point is in the merged worktree — find it with:

```bash
grep -rn "ExecuteDAG\|RunDAG" backend/internal/ --include="*.go" | grep -v _test
```

Wire it, then:

```bash
go build ./internal/temporal/...   # gate
```

### Step 4 — Invert the merge canary test

**File:** `backend/internal/temporal/swift_settlement_workflow_test.go`

`TestPipelineStubFails` must now be deleted or inverted. Post-merge assertion:

```go
// Post-merge version of the canary:
// Pipeline succeeds → workflow enters signal loop, result is MATCHED or terminal
// (depending on which signals fire in the test).
// If this test fails after merge, the pipeline wiring is broken.
```

If this test is still **PASS** after wiring, the activity is not actually calling the engine. Fix before proceeding.

### Step 5 — Engine empty-output semantics check ⚠️

**Do this before trusting tile-based discard.**

Find the pipeline engine's tile-chain executor (the code that calls `TileFunc` in sequence) and answer:

> If a tile returns an empty `[]Record{}` output for a given input record, does the engine (a) skip downstream tiles for that record, or (b) continue with the previous/stale record?

If **(a)**: `NewSessionLogTile` retransmission-discard works correctly.

If **(b)**: discard must be signalled differently. Options:
- Tile sets `rec["__discard__"] = true` and the engine or next tile honors it.
- Tile returns a sentinel error type (`ErrRetransmissionDiscarded`) the engine treats as skip+success.

**Do not ship without answering this. Guessing wrong silently processes retransmissions.**

### Step 6 — Resolve dual session-log paths

**Current state:** `Adapter.logSession()` (Layer 1) writes to `vend.swift_session_log` before `sink.Emit`. `NewSessionLogTile` (DAG position 0) writes the same row.

**Consequence:** both paths active simultaneously = every inbound message hits `swift_session_log` twice. The second write triggers the UETR unique index (23505), gets classified as a retransmission, and the record is discarded before `swift_settlement_writer`.

Pick one:

- **Option A — Keep adapter, remove tile from DAG JSON:** Session logging stays at Layer 1. No DAG entry for `swift_session_log`. Tile code stays for future use but is never registered. Simpler; adapter path is already live.
- **Option B — Keep tile, remove adapter path (`Adapter.WithDB(nil)` or remove `logSession` call):** Session logging moves into the DAG. Requires engine empty-output semantics confirmed (Step 5). Enables no-code enable/disable of session logging per tenant via DAG JSON.

Document the decision in `HANDOFF_SWIFT_SETTLEMENT.md` §Session Logging before committing.

### Step 7 — Register tile (if Option B chosen)

Add to tile factory / DAG registry (wherever `swift_decode` is registered):

```go
"swift_session_log": func(deps TileDeps, cfg TileConfig) (TileFunc, error) {
    if deps.DB == nil {
        return nil, fmt.Errorf("swift_session_log: requires DB handle — check TileDeps injection")
    }
    return tiles.NewSessionLogTile(deps.DB), nil
},
```

Update DAG JSON position 0 in `vend.swift_pipeline_dag` (or equivalent table):

```json
{ "type": "swift_session_log", "config": {} }
```

### Step 8 — CANCEL_PENDING resolver (pre-traffic requirement)

`SWIFTReconciliationWorkflow` must query `settlement_status = 'CANCEL_PENDING'` and attempt to resolve each row before any cancel traffic can exist. See Known Gap #6 below. Wire this before enabling recall in production.

### Step 9 — End-to-end smoke test

```bash
# Boot admin server
SWIFT_ENABLE=true SWIFT_ADMIN_TOKEN=dev go run ./cmd/server

# POST a real MT541 sample
curl -s -X POST http://127.0.0.1:8982/api/channels/MT541-TEST/send \
  -H "X-Swift-Admin-Token: dev" \
  -H "Content-Type: text/plain" \
  --data-raw ':20:TXN-MERGE-TEST
:23B:CINT
:98A:SETT//20261003
:35B:ISIN US0231351067
APPLE INC
:36:1000
:97A:SAFE//ACCT-001
:19A:SETT//USD1000000,
:95P:BUYR//TESTUS33XXX
:95P:SELL//CUSTGB2LXXX' \
  -w "\nHTTP: %{http_code}\n"

# Confirm the row in cash_flow.settlement
ssh eganpj@100.84.50.65 "PGPASSWORD=postgres psql -h localhost -U postgres -d alpha \
  -c \"SELECT transaction_ref, settlement_status FROM cash_flow.settlement
        WHERE transaction_ref = 'TXN-MERGE-TEST' ORDER BY created_at DESC LIMIT 1;\""
```

### Step 10 — Verify HTTP smoke paths

```bash
# 401 without token (must not 500)
curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:8982/api/channels/X/send
# Expected: 401

# 400 on missing tenant header (admin endpoint)
curl -s -X POST http://localhost:8080/api/cash-flow/swift/instructions \
  -H "Content-Type: application/json" \
  -d '{"transaction_ref": "TEST"}' \
  -w "\nHTTP: %{http_code}\n"
# Expected: 401 (no JWT) or 400 (no body fields), not 500
```

---

## Known Gaps (Updated)

*(Gaps 1–5 above updated. New gaps added below.)*

### 6. CANCEL_PENDING has no resolver — lifecycle hole ⚠️

**Created:** Sept 2026 (CANCEL_PENDING state added to prevent lying CANCELLED on failed recall).
**Status:** `SWIFTReconciliationWorkflow` does **not** currently query `settlement_status = 'CANCEL_PENDING'`.

**Consequence:** A failed recall (today: always, because `SWIFTRecallActivity` returns `ErrRecallNotImplemented`) parks the row in `CANCEL_PENDING` permanently. No SLA, no escalation, no transition to terminal state.

**Required before recall traffic exists:**

1. `SWIFTReconciliationWorkflow` must add a query:
   ```sql
   SELECT id, transaction_ref, tenant_id, custodian_id
   FROM cash_flow.settlement
   WHERE settlement_status = 'CANCEL_PENDING'
     AND tenant_id = $1
     AND updated_at < NOW() - INTERVAL '24 hours'  -- SLA: 24h resolution window
     AND valid_to IS NULL
   ```
2. For each row: attempt custodian channel query or escalation signal. Transition to `CANCELLED` (confirmed) or `FAILED` with `failure_reason = 'recall_unresolved'` (no custodian response after SLA).
3. Verify recon runs on a schedule (not only on settlement failure) — if schedule-driven: CANCEL_PENDING rows are picked up automatically. If failure-driven: add a dedicated CANCEL_PENDING sweep activity.

**Stub branch:** `swift/stub-work` (see merge instructions).

### 7. MX parser field paths not validated against seed

`TestParseMX_Pacs008_FieldExtraction` asserts the XPath keys the parser emits for pacs.008. These must match the `vend.swift_field_map` `tag` column for ISO 20022. Verify on merge day:

```sql
SELECT tag, semantic_field FROM vend.swift_field_map
WHERE msg_type = 'pacs.008' ORDER BY tag;
```

If the XPath keys in the test don't appear in `tag`, the `swift_field_map` tile silently drops them. The test catches parser drift; the seed must match.

### 8. Frontend Studio palette (unchanged)

See Gap #3 above — not touched, frontend scope.

### 9. Subtype catalog sync (unchanged)

See Gap #4 above — migration 007 applied, BO graph not yet populated.
