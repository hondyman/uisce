# Handoff: FIX over Pipeline + Temporal

Build the FIX process **on top of** the existing data-pipeline (`backend/internal/datapipeline/`, worktree `nifty-greider-015b86`) and Temporal workflow (`backend/internal/temporal/`) infrastructure — not as a standalone FIX service. Tenants configure their own FIX topology (broker, tags, rules, latency budget) by composing pipeline tiles and pinning Temporal workflows; no per-tenant code, no per-tenant recompile.

This is the opposite of "the old way" (single hardcoded behavior in `internal/fix/`). The existing FIX files in `internal/fix/` and `internal/compliance/fix_adapter.go` are the low-level quickfix driver and **dead code at that** — verified, see §3 — and either become the new system's low-level transport layer or get deleted once the new transport is in.

---

## 1. TL;DR

- **Driver (TCP + quickfix)**: keep `backend/internal/fix/server.go`; rewrite `adapter.go` to dispatch messages by `(tenant_id, session_id)` into a per-tenant data-pipeline DAG instead of evaluating compliance inline.
- **Business logic**: expressed as **data-pipeline tiles** (`fix_listener`, `fix_decode`, `fix_tag_map`, `fix_compliance`, `fix_enrich`, `fix_execution_writer`, `fix_sender`) plus **Temporal workflows** for session lifecycle (`FIXSessionLifecycleWorkflow`), order entry (`FIXOrderEntryWorkflow`, extends `backend/internal/trading/workflow.go:OrderEntryWorkflow`), and reconciliation (`FIXReconciliationWorkflow`).
- **Tenant isolation**: **must** use the GSIFI Gold Copy pattern — `(tenant_id = $X OR tenant_id = (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1))`. No exception. The new code lives in a tenant-bounded world; cross-tenant leak is a hard fail.
- **Socket ownership rule** (Amendment 1): the acceptor process owns the TCP socket; Temporal owns state and policy. Workflows never touch a socket directly — they call an internal admin API on the acceptor.
- **Reply latency** (Amendment 2): adapter sends an immediate `PendingNew` (or `Reject`) within the broker's response window; business-level `ExecutionReport` flows back through the pipeline asynchronously. If the pipeline exceeds the latency budget, adapter sends a `BusinessReject` rather than silence.
- **Sequence numbers** (Amendment 3): `quickfix.NewMemoryStoreFactory()` is unacceptable. Postgres-backed message store is an explicit build step (#5 below), not a "gotcha for later".

## 2. Where it lives

All file references are repo-relative; on `main` the data-pipeline is in worktree `nifty-greider-015b86` and not yet merged, but the rest of the FIX integration lands on `main` directly because the pipeline + Temporal scaffolding is now the mainline shape. If the data-pipeline worktree is unmerged at the time this is built, see `HANDOFF_DATA_PIPELINES.md` for what to backport.

- **Driver** (existing, keep): `backend/internal/fix/{server.go,adapter.go,tag_to_hydration.go,rule_engine_evaluator.go,adapter_test.go,server_test.go}`
- **Compliance parser to delete** (existing, dead): `backend/internal/compliance/fix_adapter.go`
- **Trading workflow to extend**: `backend/internal/trading/workflow.go` (existing `OrderEntryWorkflow`)
- **Data-pipeline tiles to add**: `backend/internal/datapipeline/transforms.go`, `engine.go` (`executeSource`/`executeTransform`/`executeLoader` switches), `model.go` (`PipelineDefinition` JSON shape — no new types needed; tiles are subTypes within `PipelineNode.SubType`)
- **Temporal workflows to add**: `backend/internal/temporal/fix_*.go` (new package or co-located in `backend/internal/temporal/`)
- **Worker registration**: `backend/cmd/worker/main.go:70` — current worker polls task queue `"bp_queue"` (see Amendment 4 verification)
- **Migrations**: `backend/db/migrations/<timestamp>_fix_*.up.sql` (table additions; see §11)
- **Reconciliation mirror**: `services/ai-trade-reconciliation/backend/temporal/{workflows,activities}/`

## 3. What's already in the repo — verified by grep

Don't trust memory. Before deleting or extending anything, the next LLM must re-run these commands and act on the **current** output:

```bash
# Who calls the compliance FIX adapter?
grep -rn "compliance/fix_adapter\|compliance.ParseFIXNewOrderSingle\|NewRuleEngineAdapter" backend/ --include="*.go"

# Who calls the rule engine adapter (the FIX-specific one)?
grep -rn "RuleEngineToComplianceEvaluator\|RuleEngineAdapter" backend/ --include="*.go"

# Who wires up fix.NewServer?
grep -rn "fix.NewServer\|fix.NewAdapter" backend/ --include="*.go"

# What's actually registered on the Temporal worker today?
grep -n "TaskQueue\|RegisterWorkflow\|RegisterActivity" backend/cmd/worker/main.go
```

As of 2026-09-13 (the date this handoff was written), the verified results were:

- `compliance/fix_adapter.go`'s `ParseFIXNewOrderSingle`: **0 callers** in `backend/`. Safe to delete.
- `RuleEngineToComplianceEvaluator` (defined at `backend/internal/fix/rule_engine_evaluator.go:89`): **0 callers**. The rule engine is wired in elsewhere (`backend/internal/compliance/dynamic_engine.go`, `backend/internal/rulefabric/...`) — the FIX adapter is dead.
- `fix.NewServer`, `fix.NewAdapter`: only `backend/internal/fix/server_test.go` calls them. **The entire FIX driver is unreachable from any live code path.**
- Task queue used by the deployed worker: **`"bp_queue"`** at `backend/cmd/worker/main.go:70`. (The data-pipeline worktree uses `workflows.DeployedBPTaskQueue` for its pipeline workflow — that's a worktree-only name; the mainline is `bp_queue`. Do **not** import `workflows.DeployedBPTaskQueue` on `main` until that worktree merges — see Amendment 4.)

The doc cites these file:line numbers; **re-run the greps and update if they shift.**

## 4. The GSIFI tenant isolation constraint (HARD REQUIREMENT)

GSIFI (Gold Standard Institutional Financial Isolation) is this repo's non-negotiable tenant boundary. Every read, every write, every workflow input, every FIX message dispatch, and every tile execution must respect it. Specifically:

- Every FIX message is tagged with the **broker's owning tenant** at the moment it enters the system — at the adapter layer, before any pipeline dispatch. The tag comes from the quickfix session's CompID↔tenant mapping in `fix_tenant_config` (see §11).
- Every SQL query the new code issues MUST use the pattern:
  ```sql
  ... WHERE tenant_id = $1 OR tenant_id = (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1)
  ```
  See `backend/internal/compliance/dynamic_engine.go:140` and `:169` for canonical examples.
- The Gold Copy tenant id is `00000000-0000-0000-0000-000000000001`. The `gold_copy = true` flag is the single source of truth — never hardcode the UUID.
- Use `backend/internal/tenant/goldcopy/resolver.go` (`Resolver.IsGoldCopy(id)`, `Resolver.ResolveGoldCopyTenantID()`) rather than re-implementing the lookup.
- RLS: `20260906_001_force_rls_tenant_bearing.up.sql` forces tenant RLS on tenant-bearing tables. The new `fix_tenant_config`, `fix_tenant_tag_mapping`, `fix_session_log` tables must either be added **with RLS enabled in the migration** (preferred) or carry the `(tenant_id = $X OR gold_copy)` clause in every query.

**A FIX tile that writes cross-tenant is a Sev-1 incident.** Build the isolation check into the engine first; bolt compliance rules on after. There is no "we'll add tenant scoping later" path — it has to be there from the first commit.

## 5. Architecture — four layers

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ Layer 1: Acceptor process (backend/internal/fix/server.go — KEEP)          │
│   • Owns TCP socket, owns quickfix acceptor, owns quickfix MessageStore     │
│   • Rewritten adapter.go: dispatches messages by (tenant_id, msg_type)      │
│     to per-tenant pipeline DAG; sends immediate PendingNew/Reject            │
│   • Exposes internal HTTP admin API for workflow control (see §6)           │
└─────────────────────────────────────────────────────────────────────────────┘
                              │ raw FIX bytes + (tenant_id, session_id)
                              ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ Layer 2: Data-pipeline tiles (NEW subTypes in transforms.go)                 │
│   • source:    fix_listener  — emits raw FIX bytes                          │
│   • transform: fix_decode    — bytes → tag/value map                         │
│   • transform: fix_tag_map   — applies tenant tag mapping                    │
│   • validator: fix_compliance — runs tenant compliance rule set              │
│   • transform: fix_enrich    — ISIN/Symbol lookups                           │
│   • loader:    fix_execution_writer — persists to OMS                        │
│   • sink:      fix_sender     — emits outgoing FIX bytes                    │
└─────────────────────────────────────────────────────────────────────────────┘
                              │ enriched PipelineRecord OR outgoing bytes
                              ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ Layer 3: Temporal workflows                                                  │
│   • FIXSessionLifecycleWorkflow  — logon/logout/health/reconnect (state)     │
│   • FIXOrderEntryWorkflow         — replaces OrderEntryWorkflow              │
│   • FIXReconciliationWorkflow     — execution vs order match                 │
└─────────────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ Layer 4: Per-tenant config (NEW tables)                                      │
│   • fix_tenant_config       — broker CompIDs, version, heartbeat interval    │
│   • fix_tenant_tag_mapping  — tag → semantic field, per tenant               │
│   • fix_session_log         — append-only audit (not a workflow source)      │
│   • fix_compliance_rule_set — FK to existing rules.* tables                  │
└─────────────────────────────────────────────────────────────────────────────┘
```

## 6. Socket ownership — Amendment 1

> The acceptor process owns all sockets. The Temporal workflow owns *state and policy*.

This is non-negotiable. A Temporal workflow is a deterministic event-history replay; a TCP socket cannot survive a worker restart, a host failover, or a workflow replay from history. If the workflow code touches a socket, the next deploy or eviction will silently corrupt the FIX session.

### What this means in code

- `backend/internal/fix/server.go` continues to host the quickfix `Acceptor` exactly as today. **One** addition: it exposes an internal HTTP admin API on a localhost-only port (suggested `127.0.0.1:8981`) that takes JSON commands: `POST /sessions/{sessionID}/logon`, `POST /sessions/{sessionID}/logout`, `GET /sessions`, `GET /sessions/{sessionID}/health`. The acceptor executes the command and returns the result. Auth: shared-secret header (`X-Fix-Admin-Token`, value from env) — never JWT, never expose this port off `127.0.0.1`.
- `FIXSessionLifecycleWorkflow` (running on `bp_queue`) issues logon/logout/reconnect by calling an activity that POSTs to the admin API. The workflow tracks the **state machine** (`INITIALIZING → LOGGING_ON → STREAMING → RECONNECTING → LOGGING_OUT → STOPPED`), records each transition as a Temporal event, and **never** instantiates a quickfix session.
- The activity that calls the admin API is `LogonActivity`, `LogoutActivity`, `ReconnectActivity`. Each is a thin HTTP call with retries.
- Rename `HeartbeatActivity` → **`SessionLivenessCheckActivity`** (Amendment 1). This is a **workflow-level watchdog** that pings the admin API's `/sessions/{sessionID}/health` and signals the workflow if the session is unhealthy. **It is not** a FIX-protocol-level heartbeat — quickfix handles those internally and the workflow must not duplicate them.

### What this prevents

- Workflow replays don't open new sockets.
- Worker failovers don't disconnect brokers.
- The admin API can be load-balanced / scaled independently of Temporal.

## 7. Synchronous-reply hole — Amendment 2

> Adapter sends an immediate FIX-level response (PendingNew or Reject). The business-level ExecutionReport flows asynchronously through the pipeline. If the pipeline exceeds N seconds, adapter sends a BusinessReject.

FIX brokers have strict response deadlines (commonly 2–5 seconds for an order ack; many brokers disconnect at 10s of silence). A data-pipeline DAG can take longer than that — especially with `host_runtime_calc`, `bloomberg_field_mapper`, or external `api_caller` tiles. The current `internal/fix/adapter.go:107-118` synchronously calls `EvaluateTrade` and replies — that's fast because the rule engine is in-process. The new pipeline is not.

### Reply contract

When the adapter (Layer 1) receives an inbound `NewOrderSingle` (`MsgType=D`):

1. **Immediately** send an application-level `ExecutionReport` with `ExecType=0` (`PendingNew`) and `OrdStatus=0` (`PendingNew`), echoing `ClOrdID` and `OrderID`. Latency budget: < 100ms from message receipt. This satisfies the broker's ack deadline.
2. Hand the message off to Layer 2 (pipeline) tagged with the same `ClOrdID` so the eventual `ExecType=4` (`Canceled`), `ExecType=F` (`Trade`) or `ExecType=8` (`Rejected`) carries the same id.
3. The pipeline's `fix_execution_writer` tile eventually emits a `Trade` or `Canceled` ExecutionReport via the `fix_sender` tile back to the broker.
4. **Timeout fallback**: if the pipeline hasn't emitted a terminal ExecutionReport within the tenant-configured budget (suggested default 30s, range 5–120s, configurable in `fix_tenant_config.pipeline_latency_budget_ms`), the adapter emits an `ExecutionReport` with `ExecType=8` (`Rejected`) and `OrdStatus=8` (`Rejected`), Text="pipeline timeout". This is *correctness over speed* — the broker hears back either way.

### What this prevents

- Broker disconnects due to silence.
- Misattribution of execution reports to wrong ClOrdIDs (impossible if both messages share the id).
- Cascading failures when a tile misbehaves (timeout catches it).

## 8. Sequence-number persistence — Amendment 3

> Promote Postgres-backed `quickfix.MessageStoreFactory` from "gotcha" to an explicit build step (#5).

The current `adapter.go:124` uses `quickfix.NewMemoryStoreFactory()`. On any restart, message sequence numbers (MsgSeqNum) reset to 1; the broker sees `MsgSeqNum=1` after seeing `MsgSeqNum=N`, and force-logs the session out. This is **not a bug for later** — it's a build prerequisite.

### Build step #5 (in §13 order)

- Implement `PostgresMessageStoreFactory` in `backend/internal/fix/` that wraps a `*sql.DB` and persists `messages` table keyed by `(session_id, msg_seq_num)`. Mirror the shape of quickfix's `MessageStore` interface (`quickfix/store.go`) — store, get, reset, refresh, onlogon, etc.
- Migration `2026MMDD_HH_fix_message_store.up.sql` adds the `fix_message_store` table.
- Wire it into `adapter.go`'s `CreateAcceptor` instead of `quickfix.NewMemoryStoreFactory()`.
- Acceptance test: logon, send 50 messages, restart the acceptor process, logon again — broker accepts the resumed `MsgSeqNum=51`. If the broker force-logs you out, the test fails.

### Alternative (only acceptable with broker cooperation)

Some brokers tolerate MsgSeqNum reset on every logon (rare). If `fix_tenant_config.allow_seq_reset = true` is set and confirmed with the broker in writing, you may keep `MemoryStoreFactory` for that tenant only. **Default is `false`; opt-in only.**

## 9. The new pipeline tiles

Add these to `backend/internal/datapipeline/transforms.go` (transform/validator/loader switch) and `engine.go` (`executeSource` for `fix_listener`). The exact subType strings, config shapes, and behaviors are below.

### `fix_listener` — source (consumer, not a host)

`fix_listener` is a **consumer of records dispatched by `adapter.go`'s rewritten `Emit` method (§13 step 3)**, keyed by `(tenant_id, broker_id)`. It never opens, closes, or hosts a quickfix session — per §6, session lifecycle is exclusively `FIXSessionLifecycleWorkflow`'s job via the admin API on `127.0.0.1:8981`. Two controllers of session lifecycle would be a second source of truth; this tile is read-only against the acceptor's already-open sessions.

- **Config**: `{"broker_id": "<uuid>", "wait_for": "first_message"|"timeout", "timeout_ms": 5000}`
- **Output**: `PipelineRecord` per inbound message the adapter dispatched, shape:
  ```json
  {
    "raw_bytes": "<base64>",
    "tenant_id": "<uuid>",
    "broker_id": "<uuid>",
    "session_id": "FIX.4.4:SENDER->TARGET",
    "msg_type": "D",
    "received_at": "RFC3339",
    "msg_seq_num": 42
  }
  ```
- **Tenant isolation**: derived from `fix_tenant_config.broker_id → tenant_id` (the dispatch key set by the adapter). **Do not** infer from ClOrdID or any tenant-controlled field. The tile never queries `fix_tenant_config` directly for tenant scoping — the adapter pre-tags.

### `fix_decode` — transform

Parses `raw_bytes` into a tag/value map. Use the canonical `crims_fix_schema.txt` (1059 lines, in repo root) as the default dictionary.

- **Config**: `{"fix_version": "FIX.4.4", "validate_required_tags": true}`
- **Output**: record gains a `tags` field: `{"11": "ORD-001", "55": "AAPL", ...}`. `raw_bytes` retained for downstream tiles.
- **Required tags** (per FIX 4.4 spec): `8` (BeginString), `9` (BodyLength), `35` (MsgType), `49` (SenderCompID), `56` (TargetCompID), `34` (MsgSeqNum), `52` (SendingTime), `10` (CheckSum). Validation failure → record routed to error-policy (default `skip_and_log`).

### `fix_tag_map` — transform

Applies tenant-specific tag → semantic-field mapping from `fix_tenant_tag_mapping`. This is the per-tenant flexibility mechanism — one tenant may map `tag 11` (ClOrdID) to `field external_order_id`; another maps it to `field client_order_ref`. The tile does the join.

- **Config**: `{"semantic_root": "trade", "drop_unmapped": false}`
- **Output**: record gains `semantic`: `{"external_order_id": "ORD-001", "symbol": "AAPL", "side": "BUY", "quantity": 100.0, "price": 178.50, ...}`.
- **Lookup query** (must follow GSIFI pattern — note the **parenthesization**; without it, AND binds tighter than OR and tenant rows bypass the `fix_version`/`msg_type` filters):
  ```sql
  SELECT fix_tag, semantic_field, transform_fn
  FROM fix_tenant_tag_mapping
  WHERE (tenant_id = $1
         OR tenant_id = (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1))
    AND fix_version = $2
    AND msg_type = $3
  ORDER BY (tenant_id = $1) DESC NULLS LAST,  -- tenant row wins over gold copy
           semantic_field
  ```
- **Precedence rule**: when both a tenant row and a gold-copy row exist for the same `(fix_version, msg_type, fix_tag)`, **the tenant row wins**. The `ORDER BY (tenant_id = $1) DESC NULLS LAST` makes this explicit. Tenants inherit defaults from gold copy; tenant rows override per-field. The downstream code path picks the *first* row per `fix_tag` (use a `DISTINCT ON (fix_tag)` pattern or iterate and overwrite).

### `fix_compliance` — validator

Runs the tenant's compliance rule set against the decoded/typed record. Replaces `backend/internal/fix/rule_engine_evaluator.go` (which becomes reference-only, then deleted once this tile is live).

- **Config**: `{"rule_set_ids": ["<uuid>", ...], "severity_threshold": "HARD_BLOCK"|"WARNING"}`
- **Behavior**: invokes the existing `rules.RuleEngine` (`backend/internal/rulefabric/vm.go` etc.). Rules below threshold → pass; rules at/above → record routed to error-policy.
- **Determinism requirement**: rules must produce the same outcome for the same input. Outbox caveat (from `HANDOFF_DATA_PIPELINES.md` §18): the `OutboxPublisher` is at-least-once; a tile that re-runs with side-effects on duplicate inputs is a bug.

### `fix_enrich` — transform

Lookups against internal tables: ISIN/Symbol/Cusip, account → portfolio, broker → venue. Backed by the existing BO + catalog drivers (`BODriver`, `CatalogDriver` in `internal/datapipeline/bo_driver.go` and `catalog_driver.go`).

- **Config**: `{"lookups": [{"source_field": "isin", "target_table": "oms.security", "target_field": "security_id"}, ...]}`

### `fix_execution_writer` — loader

Persists the execution report to OMS (`oms.trade_order`, `oms.position`). Uses `BODriver.BulkLoadSTI`. **Idempotent on `(tenant_id, exec_id, broker_id)`** (see Amendment: exec_id may be blank on partial fills — fall back to a SHA-256 hash of the raw bytes; the doc spells this out in §14).

### `fix_sender` — sink

Emits an outgoing FIX message via quickfix `SendToTarget`. **Used by**:
- `FIXOrderEntryWorkflow` to send `NewOrderSingle` after the pipeline approves.
- The `fix_execution_writer` tile when emitting a business-level `ExecutionReport` back to the broker.

**Config**: `{"target_session_id": "FIX.4.4:SENDER->TARGET", "tenant_id": "<uuid>"}`. The session_id is looked up in `fix_tenant_config` per (tenant, broker).

### `fix_order_emit` — transform (outbound NewOrderSingle builder)

Constructs a `NewOrderSingle` (`MsgType=D`) FIX message from a typed order record, using `fix_tenant_tag_mapping` **in reverse**: takes the record's semantic fields (e.g. `external_order_id`, `symbol`, `side`, `quantity`, `price`) and looks up each semantic field in `fix_tenant_tag_mapping` to find the corresponding FIX tag. Writes the resulting tag/value pairs into a `quickfix.Message` and emits it as a `PipelineRecord` with `direction: "outbound"` for the downstream `fix_sender` to serialize and dispatch.

- **Config**: `{"msg_type": "D", "cl_ord_id_source": "external_order_id"|"client_order_ref", "required_tags": [11, 55, 54, 38, 44]}`
- **Lookup query** (reverse direction of `fix_tag_map`; same GSIFI pattern + precedence rule):
  ```sql
  SELECT semantic_field, fix_tag, transform_fn, required
  FROM fix_tenant_tag_mapping
  WHERE (tenant_id = $1
         OR tenant_id = (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1))
    AND fix_version = $2
    AND msg_type = $3
  ORDER BY (tenant_id = $1) DESC NULLS LAST
  ```
- **Output**: `PipelineRecord` with `direction: "outbound"` and a populated `tags` field; `fix_sender` consumes it.
- **Reverse-direction precedence is the same as `fix_tag_map`**: tenant row wins over gold copy. Document this in the tile's contract comment so the next LLM doesn't reinvent it.
- **Tenant isolation**: derived from the upstream record's `tenant_id` (which was tagged at adapter dispatch). The tile never queries `fix_tenant_config` directly for tenant scoping.

## 10. The new Temporal workflows

### `FIXSessionLifecycleWorkflow` (long-running, one per `(tenant, broker)`)

- **Workflow ID**: `fix-session-<tenantID>-<brokerID>`
- **Task queue**: `bp_queue` (verified §3)
- **States**: `INITIALIZING → LOGGING_ON → STREAMING → RECONNECTING → LOGGING_OUT → STOPPED`
- **Signals**: `Start`, `Stop`, `HealthCheck`, `SwitchBroker` (the last only with admin auth)
- **Activities** (each calls the Layer 1 admin API from §6):
  - `LogonActivity` — POSTs `/sessions/{sessionID}/logon`; awaits 200.
  - `LogoutActivity` — POSTs `/sessions/{sessionID}/logout`.
  - `ReconnectActivity` — exponential backoff via Temporal retry policy (do **not** reimplement backoff in the workflow).
  - `SessionLivenessCheckActivity` (renamed from `HeartbeatActivity`, per Amendment 1) — GETs `/sessions/{sessionID}/health` on a workflow timer (e.g. every 30s). Signals the workflow if unhealthy.
- **Persistence**: each state transition is a workflow event; the workflow is durable across restarts. The actual TCP session state lives in the acceptor process; the workflow's view of "are we logged on?" is whatever the admin API last reported.

### `FIXOrderEntryWorkflow` (per-order)

Extends `backend/internal/trading/workflow.go:OrderEntryWorkflow`. The current workflow calls `SendFixNewOrderSingle` directly; the new workflow:

1. Validates the order (existing code, unchanged).
2. Loads `fix_tenant_config` for the tenant — confirms a session exists (or signals `FIXSessionLifecycleWorkflow` to start one).
3. Calls `SendFixOrderActivity` — which runs a per-tenant outbound data-pipeline DAG: **`fix_order_emit` (§9)** builds the `NewOrderSingle` from the typed order via reverse `fix_tenant_tag_mapping`, then `fix_sender` (§9) writes the bytes back to the broker via the admin-routed quickfix session. **The DAG returns a `ClOrdID` and the Temporal workflow ID.**
4. Waits for an `ExecutionReport` signal. The signal is delivered by the adapter when an inbound `MsgType=8` arrives matching this `ClOrdID` (the adapter correlates by `(tenant_id, ClOrdID)`).
5. Persists via `fix_execution_writer` (already in the inbound pipeline path).

The key insight: **outbound orders and inbound executions are the same pipeline.** The pipeline is parameterized by direction (inbound adapter → pipeline → sink; sink can be either `fix_sender` or `fix_execution_writer`).

### `FIXReconciliationWorkflow` (periodic, per-tenant)

Mirror `services/ai-trade-reconciliation/backend/temporal/workflows/workflows.go` exactly. Triggered by an `events` outbox event (`Pipeline.Trigger`-style outbox pattern from `internal/datapipeline/outbox.go`) emitted when `fix_execution_writer` lands a row.

- **Schedule**: every 5 minutes per active tenant (configurable in `fix_tenant_config.reconciliation_interval_sec`).
- **Inputs**: executions since last run, expected orders from `oms.trade_order` and `oms.position`.
- **Output**: reconciliation report, persisted to the existing `services/ai-trade-reconciliation` reports infrastructure if compatible, or a new `fix_reconciliation_report` table.

## 11. Per-tenant config tables

Add these migrations **before any code that reads from them** (the existing migration runner is SHA-256 hash-checked idempotent — `backend/internal/migrations/runner.go`):

```sql
-- 2026MMDD_HH_fix_tenant_config.up.sql
CREATE TABLE IF NOT EXISTS fix_tenant_config (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,            -- not FK; GSIFI pattern uses gold_copy OR
    broker_id UUID NOT NULL,            -- FK to public.cs_broker (existing)
    sender_comp_id TEXT NOT NULL,
    target_comp_id TEXT NOT NULL,
    fix_version TEXT NOT NULL DEFAULT 'FIX.4.4',
    heartbeat_interval_sec INT NOT NULL DEFAULT 30,
    pipeline_latency_budget_ms INT NOT NULL DEFAULT 30000,
    error_policy TEXT NOT NULL DEFAULT 'skip_and_log',
    allow_seq_reset BOOLEAN NOT NULL DEFAULT FALSE,  -- see Amendment 3
    reconciliation_interval_sec INT NOT NULL DEFAULT 300,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_modified_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, broker_id)
);

ALTER TABLE fix_tenant_config ENABLE ROW LEVEL SECURITY;
CREATE POLICY fix_tenant_config_tenant_isolation ON fix_tenant_config
    USING (tenant_id = current_setting('app.tenant_id')::uuid
           OR tenant_id = (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1));

-- 2026MMDD_HH_fix_tenant_tag_mapping.up.sql
CREATE TABLE IF NOT EXISTS fix_tenant_tag_mapping (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    fix_version TEXT NOT NULL,
    msg_type TEXT NOT NULL,             -- 'D' for NewOrderSingle, '8' for ExecutionReport, etc.
    fix_tag INT NOT NULL,
    semantic_field TEXT NOT NULL,
    required BOOLEAN NOT NULL DEFAULT FALSE,
    default_value TEXT,
    transform_fn TEXT,                  -- e.g. 'upper', 'numeric', 'parse_iso_currency'
    UNIQUE (tenant_id, fix_version, msg_type, fix_tag)
);
-- seed from crims_fix_schema.txt; GSIFI RLS as above

-- 2026MMDD_HH_fix_session_log.up.sql (append-only audit, not a workflow source)
CREATE TABLE IF NOT EXISTS fix_session_log (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL,
    broker_id UUID NOT NULL,
    session_id TEXT NOT NULL,
    event_type TEXT NOT NULL,           -- logon | logout | heartbeat | resend | reject
    msg_seq_num BIGINT,
    raw_excerpt TEXT,                   -- first 512 bytes of the message
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS fix_session_log_tenant_time ON fix_session_log (tenant_id, occurred_at DESC);

-- 2026MMDD_HH_fix_message_store.up.sql (Postgres quickfix store, Amendment 3)
CREATE TABLE IF NOT EXISTS fix_message_store (
    session_id TEXT NOT NULL,
    msg_seq_num BIGINT NOT NULL,
    message BYTEA NOT NULL,
    PRIMARY KEY (session_id, msg_seq_num)
);

-- 2026MMDD_HH_fix_compliance_rule_set.up.sql (FK only — no rule definition here)
CREATE TABLE IF NOT EXISTS fix_compliance_rule_set (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    rule_id UUID NOT NULL,              -- FK to existing rules.*
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    severity_threshold TEXT NOT NULL DEFAULT 'WARNING',
    UNIQUE (tenant_id, rule_id)
);
```

All **five** migrations must enable RLS as shown. **No exception.**

### RLS policy correctness

The policy above uses `current_setting('app.tenant_id')`, which **throws** when the connection hasn't issued `SET app.tenant_id = ...`. Use the missing-OK variant and a NULL-safe cast so unset connections **fail closed (zero rows)** rather than erroring:

```sql
ALTER TABLE fix_tenant_config ENABLE ROW LEVEL SECURITY;
CREATE POLICY fix_tenant_config_tenant_isolation ON fix_tenant_config
    USING (
        tenant_id = NULLIF(current_setting('app.tenant_id', 't'), '')::uuid
        OR tenant_id = (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1)
    );
```

The `'t'` second-arg to `current_setting` is `missing_ok=true`. The `NULLIF(..., '')` ensures an unset setting becomes SQL `NULL` (and the `= NULL` comparison evaluates to UNKNOWN/false) rather than raising.

### Match the existing RLS regime

Copy the exact policy shape from `20260906_001_force_rls_tenant_bearing.up.sql` rather than inventing a new one. The `app.tenant_id` setting must be issued per connection by the data-access layer (the existing `backend/internal/db/tenant_tx.go` does this — confirm it sets `app.tenant_id` for every connection in a tenant-scoped transaction). Consistency with the existing RLS regime matters more than elegance here.

If `EXPLAIN` shows the gold-copy subquery (`SELECT id FROM public.tenants WHERE gold_copy = true`) running per row per policy check, wrap it in a `SECURITY DEFINER` lookup function and cache the result. This is the standard fix when RLS policies show up in query profiles.

## 12. Amendment 4 verifications (do these BEFORE writing any code)

The next LLM must run, capture, and re-cite the output of:

```bash
# (a) Are there ANY callers of the soon-to-be-deleted files?
grep -rn "compliance/fix_adapter\|compliance.ParseFIXNewOrderSingle" backend/ --include="*.go"
grep -rn "NewRuleEngineAdapter\|RuleEngineToComplianceEvaluator\|RuleEngineAdapter" backend/ --include="*.go"

# (b) Is the task queue still bp_queue? Cite the line.
grep -n "worker.New\|TaskQueue" backend/cmd/worker/main.go

# (c) Is the data-pipeline worktree's DeployedBPTaskQueue merged yet?
git log --oneline --all | grep -i "deployedbp\|datapipeline" | head -5
grep -rn "DeployedBPTaskQueue\|BPTaskQueue" backend/ --include="*.go"
```

If (a) returns hits: do **not** delete; investigate the callers first. If (b) returns a different queue: the doc's task-queue guidance is wrong; update §10 / §13 / §16 to match. If (c) shows the worktree is unmerged: write `// TODO: switch to workflows.DeployedBPTaskQueue post-merge` in the registration site and use `bp_queue` for now.

## 13. Build order

1. **Migrations first**: `fix_tenant_config`, `fix_tenant_tag_mapping`, `fix_session_log`, `fix_message_store`, `fix_compliance_rule_set`. All with RLS. Apply via `backend/cmd/migrate` (the canonical runner — `backend/internal/migrations/runner.go`).
2. **Delete `compliance/fix_adapter.go`**: only after §12(a) confirms zero callers. The function `compliance.ParseFIXNewOrderSingle` is the entire file.
3. **Rewrite `adapter.go`** to (a) drop the `MsgType=D` switch, (b) emit per-tenant `PipelineRecord`s via a new `Emit(tenantID, sessionID, *quickfix.Message)` method, (c) send immediate `PendingNew` per §7. **Do not** touch `server.go` in this step.
4. **Add the 6 tiles** in `transforms.go` (and `engine.go:executeSource` for `fix_listener`). Each tile gets a unit test modeled on `engine_test.go`'s pattern. Test the GSIFI clause (every tile that reads from `fix_tenant_config` or `fix_tenant_tag_mapping` must include a test where the gold copy tenant and a non-gold tenant both see the right data).
5. **Postgres message store** (`fix_message_store` table + `PostgresMessageStoreFactory`). Wire it into `adapter.go`'s `CreateAcceptor`. **This is Amendment 3** — do not skip.
6. **Layer 1 admin API**: add `127.0.0.1:8981` admin HTTP listener in `server.go`. Endpoints: `POST /sessions/{id}/logon`, `POST /sessions/{id}/logout`, `GET /sessions`, `GET /sessions/{id}/health`, `POST /sessions` (creates session from `fix_tenant_config`). Shared-secret auth header. **No JWT; localhost-only.**
7. **`SessionLivenessCheckActivity` + `LogonActivity` + `LogoutActivity`** (the activities only — no workflow yet).
8. **`FIXSessionLifecycleWorkflow`** in `backend/internal/temporal/fix_session_lifecycle.go`. Register in `backend/cmd/worker/main.go` alongside `RunPipelineDAGWorkflow` (when that lands). State machine + admin-API activities.
9. **`FIXOrderEntryWorkflow`** in `backend/internal/temporal/fix_order_entry.go`, extending the existing `OrderEntryWorkflow`. Wires through the per-tenant pipeline DAG.
10. **`FIXReconciliationWorkflow`** in `backend/internal/temporal/fix_reconciliation.go`. Mirror `services/ai-trade-reconciliation/backend/temporal/workflows/workflows.go`.
11. **Frontend Studio entries**: add `fix_*` tile entries to `frontend/src/features/data-pipelines/constants/pipelineTemplates.ts`. Add a "FIX" mode to `PipelineMode`. Add a `TriggerAuthoringPage` enhancement for FIX-specific signal sources (ClOrdID, ExecType, OrdStatus).
12. **Delete `rule_engine_evaluator.go`** once the `fix_compliance` tile is live and calling `rulefabric`'s `engine.EvaluateGroup` directly. Port its existing tests as the tile's tests in `engine_test.go`; do **not** rename `ComplianceEvaluator` or `RuleEngineToComplianceEvaluator` — they already exist per §3 and the rename instruction was an editorial error. Verify the file has zero importers via `grep -rn "RuleEngineToComplianceEvaluator\|ruleEngineAdapter" backend/ --include="*.go"` before deleting.
13. **Smoke test**: Go-based quickfix initiator in an integration test under `backend/internal/fix/integration_test.go` — **does not require Docker**. Bring up an in-process initiator pointing at the acceptor (e.g. `127.0.0.1:8980`), logon, send `NewOrderSingle`, verify the inbound pipeline records arrive in the test's PipelineRecord channel, verify the outbound `PendingNew` is received by the initiator.

14. **Data-pipeline relocation** (gated on `nifty-greider-015b86` merge). When the data-pipeline worktree lands, the tiles currently sitting in `backend/internal/fix/tiles/` must move into the data-pipeline package proper:

   - Move `FixDecode`, `FixTagMap`, `FixOrderEmit` (and the four still-missing tiles: `fix_listener`, `fix_compliance`, `fix_enrich`, `fix_execution_writer`, `fix_sender`) into `backend/internal/datapipeline/transforms.go` (or its successor after the data-pipeline rename). Register each one in `engine.go`'s `executeSource` / `executeTransform` / `executeLoader` switch keyed on `node.SubType`.
   - Replace `tiles.WithTenant` / `tiles.TenantContext` with the data-pipeline's existing tenant context plumbing (likely already there per HANDOFF_DATA_PIPELINES.md §4 GSIFI pattern).
   - Move the `TagMappingLoader` interface and `FakeTagMappingLoader` test fixture into the data-pipeline's test helpers.
   - Delete `backend/internal/fix/tiles/` entirely. The `TODO(worktree-merge)` comment at the top of `tiles.go` is the trigger for this step.
   - Without this move, the FIX flow "bypasses the data-pipeline layer entirely — which is functionally the old way with better structure" (session-2026-09-13 review). The standalone package is interim only; this step is non-optional.

15. **Latency-budget fallback verification** (Amendment 2, hard half). Before shipping, verify in a smoke test that the adapter sends `ExecType=8` (Rejected) after `pipeline_latency_budget_ms` if no business-level ExecutionReport arrives. This is the half of Amendment 2 that prevents broker disconnects on silence. The fallback is implemented in `backend/internal/fix/adapter.go:scheduleLatencyReject` + `cancelPendingReject`; the test must:
   - Send a `NewOrderSingle` (immediate `PendingNew` fires).
   - Don't deliver a business-level `MsgType=8` for the same `ClOrdID`.
   - Wait `latencyBudget + slack`.
   - Verify the broker received an `ExecutionReport` with `ExecType=8` and `Text="pipeline timeout"`.

## 14. Operational gotchas

- **`exec_id` blank on partial fills**: some broker feeds send partial-fill `ExecutionReport`s without an `ExecID`. The dedup key `(tenant_id, exec_id, broker_id)` collapses to `(tenant_id, '', broker_id)` — useless. **Fallback**: when `ExecID` is blank, hash the raw message bytes (SHA-256) and use the first 16 hex chars as `exec_id`. Document this in the tile's code comment.
- **Shared-acceptor session dictionaries**: when one quickfix acceptor hosts multiple tenants, each session block in `quickfix.SessionSettings` must carry its own `FIX44.xml` data dictionary if tenants use different `fix_version`. The current `server.go:35-36` hardcodes `BeginString=FIX.4.4` globally — for multi-tenant, change to per-session settings loaded from `fix_tenant_config.fix_version`. **Do this in build step #6**, not as an afterthought.
- **Tenant CompID collision**: two tenants may both register `SenderCompID=BRK01` (unlikely but possible with shared brokers). The admin API in §6 must validate uniqueness at session creation time and reject the second tenant with `409 Conflict`.
- **Outbox at-least-once + idempotency**: every outbound tile must be idempotent on `(tenant_id, ClOrdID, BrokerID)`. Document this in the tile's contract comment.
- **Outbound `fix_sender` from inside the workflow**: don't call it from the workflow goroutine — call it from an activity. Workflows are deterministic; `SendToTarget` over TCP is not.
- **`MaximumAttempts: 1` discipline**: pipeline runs touching external sinks are not safely re-runnable. New FIX tile activities that write to OMS must inherit this discipline (mirrors `HANDOFF_DATA_PIPELINES.md` §18).
- **`ABACEngine.Evaluate` is a stub** (per `HANDOFF_BI_WORK.md`): pipeline runs are not currently gated by field-level entitlements. The new `fix_compliance` tile must apply the tenant's rule set explicitly (passed via `fix_compliance_rule_set`), not the global one.
- **CEL is retired** (Amendment F/G in the recent rules work): do **not** import `*cel.Env` or `pkg/workflows/cel_*` paths. The `rules.RuleEngine` has been refactored to a non-CEL evaluator (`backend/internal/rulefabric/vm.go`). The new `fix_compliance` tile calls `engine.EvaluateGroup` (which exists at `rule_engine_evaluator.go:35`), not the legacy CEL path.
- **Per-tenant quickfix settings files** (if any exist on disk): inventory and delete as part of build step #6 — they conflict with `fix_tenant_config`.

## 15. Known gaps / explicitly deferred

- **FIX 5.0 support**: only FIX 4.4 initially. `fix_tenant_config.fix_version` is TEXT so 5.0 lands later via a data dictionary update + admin-API validation.
- **Resend-request handling**: quickfix handles it internally via the `MessageStore`; no app-level code needed. **Test it** in build step #13 — force a gap and verify the resend.
- **FIX-over-TLS**: plain TCP only. TLS termination at the acceptor is a deploy-time concern; not in scope.
- **Streaming market data** (`MsgType=V` MarketDataIncrementalRefresh): out of scope.
- **Per-tenant port allocation**: initial impl uses shared acceptor with session-ID dispatch (one port). Per-tenant-port is a future mode if brokers require it.
- **Admin API auth beyond shared secret**: when this becomes multi-operator, swap to mTLS or JWT with `admin` role. Not for the first ship.
- **Tile-level backpressure**: if a tenant's pipeline is slow, the §7 latency budget protects the broker, but the adapter still enqueues inbound messages. Add bounded queue + `Rejected` reply when the queue is full — deferred until measured.
- **Reconciliation report storage**: the integration with `services/ai-trade-reconciliation`'s report infrastructure is TBD. Build the workflow; emit reports to `fix_reconciliation_report` table; reconcile into the unified reports later.

## 16. Extension recipes

### Add a new FIX tile subType

1. Add the subType string to `transforms.go`'s switch in `executeTransform` (or `executeSource`/`executeLoader`).
2. Add a constructor function for the new transform/validator/loader struct in `transforms.go` (or a new file in `backend/internal/datapipeline/`).
3. Add the tile entry to `pipelineTemplates.ts` so the Studio palette shows it.
4. Add a unit test in `engine_test.go` (or a new `*_test.go` in `backend/internal/datapipeline/`) covering: GSIFI gold-copy tenant read, non-gold tenant read, malformed input, error-policy behavior.

### Add a new tenant config column

1. New migration adding the column to `fix_tenant_config`. **RLS stays on.**
2. Update `PipelineDefinition`-style DTO in `model.go` (if the field flows through pipeline JSON).
3. Update the admin API in §13 step #6 to accept the new field.
4. Update `internal/api/.../fix_*_handler.go` if any handler reads the column.

### Wire a new broker

1. Insert row into `public.cs_broker` (existing table, see `broker_schema.txt`).
2. Insert row into `fix_tenant_config` with `broker_id` and the tenant's CompIDs.
3. (Optional) insert `fix_tenant_tag_mapping` rows to override default tag mappings.
4. Start `FIXSessionLifecycleWorkflow` for `(tenant_id, broker_id)`. It does the rest.

## 17. Don't-do list

- **Don't** import `*cel.Env` or any CEL-related path. CEL is retired.
- **Don't** put a quickfix `Acceptor` instantiation inside a Temporal workflow function. Sockets are owned by `server.go`.
- **Don't** call `quickfix.SendToTarget` from a workflow goroutine. Wrap in an activity.
- **Don't** use `quickfix.NewMemoryStoreFactory()` unless `fix_tenant_config.allow_seq_reset = true` and the broker has confirmed in writing.
- **Don't** register workflows on the wrong task queue. Verify `backend/cmd/worker/main.go:70` says `bp_queue` before registering (`DeployedBPTaskQueue` is data-pipeline-worktree-only; don't import it on `main` until that worktree merges).
- **Don't** ship a tile that reads `fix_tenant_config` or `fix_tenant_tag_mapping` without the `(tenant_id = $X OR gold_copy)` clause. **GSIFI is a hard requirement.**
- **Don't** rely on workflow signals for FIX protocol-level heartbeats. quickfix handles them internally.
- **Don't** delete `internal/compliance/fix_adapter.go` until §12(a) confirms zero callers (run the grep; cite the output).
- **Don't** add per-tenant quickfix settings files. Use `fix_tenant_config`.
- **Don't** build per-tenant port allocation in the first ship. Shared acceptor with session-ID dispatch is enough.
- **Don't** ship without the §13 step #5 Postgres message store. Sequence-number loss is a broker disconnect.

## 18. Verification commands

```bash
# 1. Build & unit tests
go build ./...
go test ./internal/fix/... ./internal/datapipeline/... ./internal/temporal/... -count=1

# 2. GSIFI isolation regression tests
go test ./internal/datapipeline/... -run TestGoldCopy -count=1
go test ./internal/fix/... -run TestTenantIsolation -count=1

# 3. Schema check
psql -c '\d fix_tenant_config' -c '\d fix_tenant_tag_mapping' -c '\d fix_message_store' -c '\d fix_session_log'
psql -c "SELECT relname, relrowsecurity FROM pg_class WHERE relname LIKE 'fix_%';"

# 4. Worker registration
grep -n 'RegisterWorkflow\|RegisterActivity\|TaskQueue' backend/cmd/worker/main.go

# 5. RLS is on for all new tables
psql -c "SELECT tablename, rowsecurity FROM pg_tables WHERE schemaname='public' AND tablename LIKE 'fix_%';"

# 6. Smoke test (no Docker required)
go test ./internal/fix/... -run TestIntegration_FIXSession -count=1 -v

# 7. Confirm dead code is gone (post-cleanup)
! grep -rn "compliance/fix_adapter\|compliance.ParseFIXNewOrderSingle" backend/ --include="*.go"
! grep -rn "RuleEngineToComplianceEvaluator" backend/ --include="*.go"
```

If any of (1)–(7) fails, the build is not done.

---

## 19. If starting a new session on this

1. **Run §12 first** — re-cite the grep output. The doc's file:line refs were current as of 2026-09-13; if they shifted, update before coding.
2. **GSIFI is the contract.** Every SQL query, every tile read, every workflow input must respect the gold-copy OR-clause. Build the isolation check into the engine first.
3. **Sockets belong to `server.go`.** Workflows talk to it via the admin API on `127.0.0.1:8981`. If you're tempted to instantiate a quickfix `Initiator` from inside a workflow, stop.
4. **Two distinct idempotency keys**:
   - **Inbound** (`fix_execution_writer`, `fix_listener` consumers): dedup on `(tenant_id, exec_id, broker_id)` with a SHA-256-of-raw-bytes fallback when `ExecID` is blank (some broker feeds send partial-fill reports without one — see §14).
   - **Outbound** (`fix_order_emit` + `fix_sender`): dedup on `(tenant_id, ClOrdID, broker_id)` with a SHA-256-of-record fallback when `ClOrdID` is missing.
   - These are different keys; do not conflate. Document the key in each tile's contract comment.
5. **Pipeline latency budget** is in `fix_tenant_config.pipeline_latency_budget_ms`. Adapter sends `BusinessReject` if the budget is exceeded. Don't skip this.
6. **CEL is gone.** Don't reintroduce it.
7. The user's standing rule: **everything flows through the data-pipeline + Temporal workflow layer** (per `HANDOFF_DATA_PIPELINES.md` "If starting a new session on this"). Don't add new raw-TCP-against-quickfix shortcuts.

### Admin-API network co-location

The `127.0.0.1:8981` admin listener is host-local, **not** container-local. If the acceptor runs containerized (e.g. in `docker-compose.remote.yml`), the Temporal worker pod/process must share the acceptor's network namespace to reach it — `network_mode: "service:fix-acceptor"` in compose, or a sidecar pattern. Alternatively, parameterize the admin URL via env (`FIX_ADMIN_URL=http://fix-acceptor:8981`) and rely on the Docker network to route. **Do not** ship a config that assumes localhost from a container.
