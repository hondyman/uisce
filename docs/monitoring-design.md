# Monitoring Feature — Design Document

## Pre-flight findings (already verified, locked)

| # | Finding | Implication |
|---|---|---|
| 1 | Iceberg/StarRocks pipeline exists (`cmd/audit-sink/main.go` + `internal/audit/iceberg_sink.go`); Option A (CDC) is consistent with platform. | Only need a new Debezium publication for `report_executions` + new Kafka topic `audit.report_executions.events` |
| 2 | 5 writers to `report_executions`: pending insert (`temporal_executor.go:107`), running transition (`temporal_executor.go:191`), markExecutionFailed (`temporal_executor.go:221`), StoreExecutionResultActivity (`report_activities.go:170`), SweepStaleExecutions (`reconciler.go:18`) | All 5 writers in Phase 1 |
| 3 | `RoleGlobalAdmin = "global_admin"`, `hasRole(r, "global_admin")` handler pattern, `SetGlobalAdminContext` GUC, `admin_audit_logs` table all exist. | Admin gate unambiguous |
| 4 | Phase 4's `ReportExecutionStatusChip` + `pollExecution` backoff/AbortController + query-key convention is the right pattern. `ExecutionMonitor.tsx` is a 42-line untyped stub, not a precedent. | New components generalize Phase 4's polling discipline |
| 5 | Live `report_executions`: 11 rows, 0 active schedules, 0 production traffic. Green-field scale. | 90d hot window holds everything; don't partition Postgres; partition Iceberg by `execution_date` monthly |
| 6 | Visibility predicate proven at `report_handlers.go:805-820`: `(e.tenant_id = $2 OR e.triggered_by = $3) AND (t.is_personal = false OR t.created_by_id = $3 OR $4 = true)` | Paste verbatim into tenant-list endpoint |
| 7 | Admin route convention: `/audit-explorer/dashboard/global-admin` with `hasRole(r, "global_admin")` at handler entry. | New admin endpoints mount at `/api/v1/admin/...` |
| 8 | Topic naming: `audit.<domain>.<noun>`. Existing `audit.orchestration.events` is broader workflow events; dedicated `audit.report_executions.events` for the CDC stream. | Add to `cmd/audit-sink` subscription; add Debezium publication for `public.report_executions` |
| 9 | `report_executions` is `REPLICA IDENTITY DEFAULT` (verified by FTS runbook). | Events table needs same pre-flight check |

---

## 1. The Model

**Three-tier storage**: hot (Postgres `report_executions`), warm (Iceberg via CDC), cold (StarRocks external catalog for analytics). **Two user groups**: tenant-ops with visibility predicate; alpha-admin with metadata-only redaction on personal executions.

**Audit dimensions**: per-execution lifecycle events (CREATED, STARTED, COMPLETED, FAILED, SWEEP_RECONCILED, DISPATCH_FAILED) + per-schedule schedule-execution counts + per-tenant aggregate stats.

### Decision 1 — Admin carve-out: CONFIRMED as designed

- Admins see execution **metadata** (status, timing, errors, template reference, identity triple) for all executions including personal; never `parameters` or `output_url` for personal reports.
- Field-level redaction is **server-side key exclusion** — the JSON keys are absent from the response, not present-but-empty. Tests assert `assert.NotContains(keys(resp), "parameters")` and `assert.NotContains(keys(resp), "output_url")` for personal executions viewed cross-tenant.
- Visible `ParametersRedacted: true` indicator in the response body signals the redaction to the client.

**Rationale**: operational health doesn't need report payloads. The audit trail's value is knowing *what ran and when*, not *what the data was*.

**Audit completeness invariant**: every execution has ≥3 events (created, started, terminal). Enforced by transactional writer instrumentation. Sweep reconciliation flags broken chains.

**Audit immutability**: `report_execution_events` is insert-only. `REVOKE UPDATE, DELETE ON report_execution_events FROM PUBLIC` in DDL — but see Amendment 1 on the principal and grant structure.

**Access accountability**: viewing audit data is audit-worthy for admins. Admin read endpoints write to `admin_audit_logs` with `action = 'monitoring.read.cross_tenant'`.

---

## 2. Backend Design

### 2.1 DDL — Phase 1 (new migration)

```sql
-- Migration: 2026XXXX_create_report_execution_events.sql
CREATE TABLE IF NOT EXISTS public.report_execution_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id    UUID NOT NULL REFERENCES public.report_executions(id) ON DELETE CASCADE,
    tenant_id       UUID NOT NULL,  -- denormalized for RLS + CDC partitioning
    event           TEXT NOT NULL,  -- 'CREATED'|'STARTED'|'COMPLETED'|'FAILED'|'SWEEP_RECONCILED'|'DISPATCH_FAILED'
    from_status     TEXT NULL,      -- nullable for CREATED
    to_status       TEXT NOT NULL,
    actor_id        TEXT NOT NULL,  -- user_id or service identifier
    detail          JSONB NULL,     -- {error_message, output_url, rows_processed, execution_time_ms, ...}
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for common query shapes
CREATE INDEX idx_ree_execution_id_created_at ON public.report_execution_events(execution_id, created_at);
CREATE INDEX idx_ree_created_at ON public.report_execution_events(created_at);
CREATE INDEX idx_ree_tenant_id_created_at ON public.report_execution_events(tenant_id, created_at);

-- RLS: same policy shape as report_executions
ALTER TABLE public.report_execution_events ENABLE ROW LEVEL SECURITY;
CREATE POLICY ree_tenant_isolation ON public.report_execution_events
    USING (tenant_id = current_setting('uisce.current_tenant', true)::UUID);

-- Insert-only enforcement: app role gets INSERT+SELECT only; PUBLIC blocked
-- Migration runner must create the app role grant:
--   GRANT INSERT, SELECT ON public.report_execution_events TO :app_role;
--   REVOKE UPDATE, DELETE ON public.report_execution_events FROM PUBLIC;
--   REVOKE UPDATE, DELETE ON public.report_execution_events FROM :app_role;  -- redundant but explicit

-- Amendment 1 note: ON DELETE CASCADE from report_executions means event rows are deleted
-- when an execution is hard-deleted. This is acceptable because:
--   (a) execution hard-deletion is not currently in the data model (soft-delete only)
--   (b) if hard-deletion is added later, the cascade is intentional — orphaned events are worse
-- If this changes, remove the ON DELETE CASCADE and add a deleted_flag column instead.
```

### 2.2 Writer instrumentation — 5 writers with transaction boundaries

| Writer | File:line | Event | Transaction |
|---|---|---|---|
| 1. Pending insert | `temporal_executor.go:107` | `('CREATED', null, 'pending')` | In same transaction as the insert |
| 2. Running transition | `temporal_executor.go:191` | `('STARTED', 'pending', 'running')` | In same transaction |
| 3. markExecutionFailed | `temporal_executor.go:221` | `('FAILED', ?, 'failed')` | In same transaction |
| 4. StoreExecutionResultActivity | `report_activities.go:170` | `('COMPLETED', 'running', 'completed')` | In same transaction |
| 5. SweepStaleExecutions | `reconciler.go:18` | `('SWEEP_RECONCILED', ?, 'failed')` | Restructured — see below |

### Decision 2 — SweepStaleExecutions refactor: CONFIRMED as `UPDATE ... RETURNING` + bulk event insert, one transaction

Per-row `SELECT FOR UPDATE` adds lock overhead for no audit benefit — the RETURNING set *is* the per-execution record, and a single transaction containing both the bulk update and the per-row event inserts preserves the atomicity that matters. Broken-chain check runs in the same sweep transaction.

**Shape**:
```go
func SweepStaleExecutions(ctx context.Context, ... ) error {
    return db.WithTx(ctx, func(tx pgx.Tx) error {
        // Bulk update + RETURNING in one round-trip
        rows, err := tx.Query(ctx, `
            UPDATE report_executions
            SET status = 'failed', updated_at = NOW()
            WHERE status = 'running' AND updated_at < $1
            RETURNING id, tenant_id, status, previous_status, error_message
        `, cutoff)
        if err != nil {
            return err
        }
        defer rows.Close()

        var events []InsertEvent
        var executionIDs []uuid.UUID
        for rows.Next() {
            var e Execution
            var prevStatus string
            // scan e.ID, e.TenantID, e.Status, prevStatus, e.ErrorMessage
            events = append(events, InsertEvent{
                ExecutionID: e.ID,
                TenantID:    e.TenantID,
                Event:       "SWEEP_RECONCILED",
                FromStatus:  prevStatus,
                ToStatus:    "failed",
                ActorID:     "system:sweep",
                Detail:      jsonb.BuildObject("reason", "stale_timeout"),
            })
            executionIDs = append(executionIDs, e.ID)
        }

        // Bulk insert events
        if len(events) > 0 {
            if err := bulkInsertEvents(ctx, tx, events); err != nil {
                return err
            }
        }

        // Broken-chain check in same transaction
        // Executions with terminal status but no terminal event
        if nonZero := checkBrokenChains(ctx, tx, executionIDs); nonZero > 0 {
            // Log + alert; do not fail the sweep
            log.Warn("broken chains detected", "count", nonZero)
        }

        return nil
    })
}
```

**Broken-chain detection query** (in same transaction):
```sql
SELECT e.id FROM report_executions e
WHERE e.id = ANY($1)  -- the swept execution IDs
  AND e.status IN ('completed', 'failed', 'failed_dispatch')
  AND NOT EXISTS (
      SELECT 1 FROM report_execution_events ree
      WHERE ree.execution_id = e.id
        AND ree.event IN ('COMPLETED', 'FAILED', 'DISPATCH_FAILED')
  )
```

### 2.3 Endpoints

**Tenant operations** (`/api/v1/reports/`):

| Method | Path | Purpose |
|---|---|---|
| GET | `/reports/executions?template_id=&status=&from=&to=&limit=&cursor=` | Cursor-paginated list. **Predicate pasted verbatim** from `report_handlers.go:805` |
| GET | `/reports/executions/{id}` | Exists from Phase 3 |
| GET | `/reports/schedules/{id}/executions?limit=&cursor=` | Schedule-scoped history |
| GET | `/reports/executions/{id}/events` | Audit timeline |

**Alpha admin** (`/api/v1/admin/`):

| Method | Path | Purpose | Gate |
|---|---|---|---|
| GET | `/admin/report-executions?tenant_id=&status=&from=&to=&limit=&cursor=` | Cross-tenant listing. Metadata-only redaction for personal executions (server-side key exclusion — `parameters` and `output_url` absent from JSON). | `hasRole(r, "global_admin")` |
| GET | `/admin/report-executions/{id}` | Single execution cross-tenant view | `hasRole(r, "global_admin")` |
| GET | `/admin/report-execution-events?tenant_id=&from=&to=` | Cross-tenant event search | `hasRole(r, "global_admin")` |
| GET | `/admin/report-execution-analytics?...` | Aggregate analytics via StarRocks (backend-proxied, server-side filters only — see Amendment 4) | `hasRole(r, "global_admin")` |

**Admin boilerplate (every handler)**:
```go
if !hasRole(r, "global_admin") {
    http.Error(w, "insufficient permissions", http.StatusForbidden)
    return
}
auditLogAdminRead(ctx, db, "monitoring.read.cross_tenant", filtersJSON)
```

### 2.4 Hot→Cold pipeline — Option A (CDC)

1. **DDL**: add `report_executions` + `report_execution_events` to a new Debezium publication `alpha_pub_report_executions`
2. **Topic**: `audit.report_executions.events` (created by Debezium or admin)
3. **Sink**: extend `cmd/audit-sink/main.go` to subscribe to the new topic
4. **Iceberg schema**: `audit.report_executions` + `audit.report_execution_events`, partition by `execution_date` (monthly)
5. **StarRocks external catalog**: per existing pattern in `pre_aggregation_service.go`

**Amendment 3 — Replica identity pre-flight**: `report_executions` is `REPLICA IDENTITY DEFAULT` (verified by FTS runbook). Before adding `report_execution_events` to the Debezium publication, verify replica identity on the events table and document the finding in the Phase 4 pre-flight checklist, citing the FTS runbook failure mode.

---

## 3. Frontend Design

### Tenant operations view — `/reports/executions` (new route)

- `<ExecutionsListTable>` — paginated table, columns: status (chip), template, started, duration, rows_processed, error preview
- `<ExecutionDetailDrawer>` — opens on row click; shows full execution record + `<AuditTimeline>`
- `<AuditTimeline>` — vertical stepper of events per execution
- `<ExecutionFilters>` — template, status, date range

**Hooks** (extending `reportExecutionApi.ts`):
- `useExecutionList({ filters, cursor })` — TanStack Query, key `['reporting', 'executions', filters, cursor]`
- `useExecutionEvents(executionId)` — TanStack Query, key `['reporting', 'executions', executionId, 'events']`

### Alpha admin console — `/admin/monitoring` (new admin route)

- `<AdminMonitoringDashboard>` — three panels:
  - **Real-time** (Postgres-backed): status histogram, dispatch-failure count, stale-execution count
  - **Trend** (StarRocks-backed): 90d p95 duration, failure-rate-by-tenant
  - **Tenant drill-down**: metadata-only for personal executions with `<RedactedFieldIndicator>`
- `<RedactedFieldIndicator>` — badge showing "redacted — personal report"
- `<AuditOfAudit>` — "this view is access-logged" indicator in header

**Source indicator**: UI distinguishes "live (Postgres)" vs. "analytics (Iceberg/StarRocks)" so admins know when they're looking at real-time vs. cold data.

**Routing**:
```
/reports/executions              # tenant list
/reports/executions/{id}         # tenant detail
/reports/schedules/{id}          # schedule dialog gains "Execution History" tab
/admin/monitoring                # admin dashboard
/admin/monitoring/{tenantId}     # tenant drill-down
```

---

## 4. Phased Implementation

### Phase 1 — DDL + all 5 writers + broken-chain detection
- Migration for `report_execution_events` table + indexes + RLS + grants (see Amendment 1 for grant principal)
- `tenant_id` column denormalized on events table (RLS + CDC partitioning, see Amendment 2)
- Instrument writers 1–4 (same-transaction event inserts)
- Restructure SweepStaleExecutions: `UPDATE ... RETURNING` + bulk event insert + broken-chain check (Decision 2, all in one transaction)
- Tests: event chain correctness per execution lifecycle, broken-chain detection, server-side redaction (admin view asserts JSON keys absent)

### Phase 2 — Repository + tenant read paths
- `execution_repository.go`: cursor-paginated list, single get, events read
- Add handlers: `ListExecutions`, `ListScheduleExecutions`, `ListExecutionEvents`
- **Predicate pasted verbatim** from `report_handlers.go:805`

### Phase 3 — Admin surface + StarRocks proxy rules
- `admin_handlers.go`: all four admin endpoints with `hasRole` + `auditLogAdminRead`
- **Amendment 4**: StarRocks proxy constructs queries server-side with parameterized filters only; no client-supplied SQL fragments; `tenant_id` filter applied server-side even when admin is browsing cross-tenant analytics (the analytics engine trusts the API, not the client).
- Security matrix: tenant user → 403; admin sees metadata, `parameters`/`output_url` absent from personal execution responses (server-side key exclusion, not client-side hiding).

### Phase 4 — Hot→Cold pipeline
- Add topic to `cmd/audit-sink`
- **Amendment 3**: replica identity verification for `report_execution_events` (pre-flight, citing FTS runbook)
- New Debezium publication
- Iceberg schema + StarRocks external catalog registration
- Round-trip verification

### Phase 5 — Frontend + E2E
- New routes, components, hooks
- E2E spec: tenant surface + admin surface with role-gated assertions

---

## 5. The Two Risks

**Risk 1 — Admin cross-tenant read is a new attack surface.** Defense-in-depth: `hasRole` gate at handler entry; server-side field exclusion (keys absent, not empty); `admin_audit_logs` on every admin read; RLS defense-in-depth (events table now has RLS, same policy as `report_executions`). Tests prove boundary both ways.

**Risk 2 — Audit trail born incomplete.** Defense: Phase 1 ships writer instrumentation with the events table together; transactional guarantee per writer; SweepStaleExecutions refactored in the same phase so no writer produces terminal status without a terminal event (Decision 3); broken-chain detection in sweep. `REVOKE UPDATE, DELETE` makes the events table insert-only by construction.

---

## 6. Process notes (ledger cross-reference)

- **`gh pr create` backtick safety**: always use `--body-file` with a heredoc for PR bodies that include commit hashes or any text that could be interpreted as shell command substitutions. The `gh pr create` command parses `--body` through the shell, so backtick-quoted content runs as command substitutions before `gh` processes the flag. This caused empty cells in PR #55's commit table; the PR body was corrected manually.
- **`git rev-parse HEAD` before every gate run**: tree identity proof, institutionalized after the stale-branch incident. Any gate table that does not begin with `git rev-parse HEAD` + `git status` output is not a trusted result.
- **Failures ledger location**: `docs/FAILURES_LEDGER.md`

---

**Three decisions confirmed. Four amendments incorporated. Phase 1 ready to begin on verified `main`.**
