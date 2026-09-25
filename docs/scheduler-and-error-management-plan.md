# One Scheduler, One Error Catalog — Plan

> **Status:** SIGNED 2026-09-25 via decision answers: D1 (a) rebuild the Scheduler
> Intelligence console on the core; D2 apply additive migrations to `alpha`, engine on
> the Docker host; D3 maker–checker per tenant, default on; D4 LLM per-tenant opt-in
> with mandatory redaction. D5 revised below, pending confirmation.

Goal (owner, 2026-09-25): one centralized, easy, visual scheduler used by reports,
the query builder, data pipelines and workflows; one centralized error catalog and
error management (PeopleSoft Message Catalog + Process Monitor style) with an
error-management bot; everything GSIFI-safe; delivered end to end on the live
backend.

---

## §1 — What exists today (evidence)

### Schedulers: ten models, none authoritative

Reachability is from `deadcode ./cmd/...`; row counts are read-only probes of `alpha`
(every schedule table is empty).

| # | Where | What it does | State |
|---|---|---|---|
| S1 | `services.PostgresSchedulerService` + `handlers.SchedulerHandlers` (`/api/v1/schedules`), `edm.scheduled_jobs` | in-process `@every 1m` poll → async job queue | **live and unsafe**: no row locking (every API replica fires the same job); `next_run_at` is only advanced when the enqueue *fails*, so a due job re-fires every minute forever |
| S2 | `api.SchedulerHandlers` (`/api/scheduler/jobs,dags,…`) + `internal/scheduler_intelligence` (`scheduled_jobs`, `scheduled_dags`, changesets+approvals, AI suggestions, blast radius, residency) | job/DAG scheduler with a governance layer; "Scheduler Intelligence" console | live (≈100 of 148 funcs reachable; `ai/` 109 of 123 dead). Reads the actor from a client-supplied `X-User-ID` header |
| S3 | `internal/reporting` report schedules (`report_schedules`, `tenant_exchange_calendars`, holidays, business-day offset, unscheduled-day behaviour, bursting) | richest timing model | CRUD live, **never fires**: `ProcessDueSchedules` has no caller. Tables not even applied on `alpha` |
| S4 | `data_explorer.saved_query_schedule` (query builder) | metadata only | migration says "dispatching is not yet wired" |
| S5 | `scheduled_reports` (semantic query + Temporal `workflow_id`) | — | table only |
| S6 | `export_schedules` (Page Studio exports) | — | table only |
| S7 | `features/scheduler` frontend (jobs, executions, calendars, notification templates, dependency view) | calls `/api/scheduler/{jobs/:id/trigger,pause,resume,clone}`, `/executions`, `/schedules`, `/next-runs` | most endpoints do not exist in S2 → the pages cannot work |
| S8 | pre-aggregation scheduler, `jobs/queue_refresh_job`, `semantic.refresh_schedules` | internal refresh loops | live, internal |
| S9 | `calcengine/lifecycle_scheduler`, `api/ai_scheduler_handlers` | — | dead |
| S10 | data pipeline schedules (this branch) | Temporal Schedules, skip-on-overlap | live on branch |

Calendars: MDM already holds governed golden-record calendars (`mdm.calendar_master`,
`calendar_day`, `holiday_definition_calendar`, `settlement_calendar`, …). S3 invented a
second calendar store (`tenant_exchange_calendars`).

### Errors: a catalog nobody uses

* `message_sets` (11 sets: System 1, Compliance 1000, Risk 2000, ETL 3000, WASM 4000,
  Delivery 5000, Lineage 6000, Reports 7000, Notification 8000, CoreCustom 10000,
  ClientCustom 20000) and `message_catalog` (83 messages, `%1` parameters, severity,
  explanation) — the PeopleSoft model, seeded — plus `tenant_message_catalog`
  (overrides, 0 rows). **No code reads any of them.**
* Error/incident tables exist but are empty and owned by a mostly-dead `internal/ops`:
  `ops_error_events`, `ops_error_fingerprints` (AI analysis, assignment, status),
  `ops_incidents`, `platform_exceptions` (autofix attempts, `closed_by_ai`),
  `exception_routes`, `exception_autofix_policy`; MDM has its own
  `catalog_mdm.universal_exception_queue`.
* Handlers largely return `err.Error()` to clients — internal detail (SQL errors,
  hostnames, identifiers) can reach end users. A GSIFI finding on its own.

---

## §2 — Target

### One scheduler (`internal/schedule`)

* **One schedule model** for every target: `target {kind, ref_id, params}` where kind ∈
  `data_pipeline | report | saved_query | page_export | workflow | job_dag`; timing =
  preset or cron + time zone, optional **business calendar** (MDM golden calendar),
  business-day offset, non-business-day rule (skip / next / previous business day),
  start/end dates, blackout windows; delivery (notification channels / recipients);
  owner; status.
* **One engine: Temporal Schedules** (already proven on this branch against a real
  Temporal server). Each firing starts `ScheduledTargetWorkflow`, which evaluates the
  calendar rule at fire time (Temporal cron cannot express holidays), records the run,
  and dispatches to the target's **runner** (pipeline runner, report executor, saved
  query export, workflow start, DAG). Overlap = skip by default. No in-process poll
  loops anywhere (S1 is deleted), so N API replicas never double-fire.
* **One run history** (`schedule_runs`): target, status, timings, row counts, error
  code (catalog), correlation id, Temporal ids — the Process Monitor.
* **One API** (`/api/v1/schedules`, `/runs`, `/calendars`) and **one MCP tool set**.
* **Events on Redpanda** (the platform's streaming layer, Kafka API): schedule-run and
  error events are published to Redpanda topics for notification, the error bot and
  downstream consumers.
* **One UI**: a reusable `<ScheduleEditor>` (presets, calendar picker, next runs
  shown in the time zone, plain-language summary) embedded in the report builder,
  query builder, pipeline editor and workflow designer; plus one **Schedules console**:
  every schedule across kinds, calendar/timeline view, pause/resume/run-now, run
  history drill-down, failures linked to the error catalog.

### One error catalog and error management (`internal/msgcat`, `internal/errormgmt`)

* **Catalog as code-of-record**: every user-facing error is `msgcat.New(set, nbr, params…)`
  → typed `AppError{Set, Nbr, Severity, Params, CorrelationID, cause}`. Rendering uses
  the tenant override (`tenant_message_catalog`) → core catalog, by language. The
  client receives `{code: "3000-12", severity, message, correlation_id}` — **never the
  internal cause**, which is logged server-side only.
* **Error log**: every Error/Fatal occurrence is recorded (tenant, code, correlation,
  redacted context), deduplicated by fingerprint into an **error queue** with status,
  owner, severity, SLA, resolution notes, links to the run/record that failed, and
  retry/resubmit where the target supports it.
* **Error Management console**: browse/edit the catalog (core read-only for tenants;
  tenant overrides and client ranges editable with maker–checker), search the error
  log, work the queue (assign, resolve, retry), incident view, trends.
* **Error-management bot**: for each new fingerprint — explains it from the catalog
  entry + redacted context, proposes resolution steps, clusters related errors, opens
  or updates an incident, routes via `exception_routes`, notifies; autofix only where
  `exception_autofix_policy` allows and always with an audit record. Also available in
  the console as "Explain / suggest fix" and over MCP.

### GSIFI safety (applies to every slice)

1. Tenant from the authenticated token only — no header-supplied identity
   (`X-User-ID` in S2 removed), no fallback tenant; every table tenant-keyed; RLS on
   new tables.
2. Segregation of duties: schedule and catalog changes can require maker–checker per
   tenant policy (S2's changesets/approvals become the shared mechanism).
3. Immutable audit trail of every schedule/catalog/error-queue change and every run
   (who, when, before/after).
4. No internal error detail to clients; PII/secrets redacted before storage and before
   any LLM call; LLM use for the bot controlled per tenant (D4).
5. Exactly-once firing (Temporal), idempotent runners, no overlap by default, bounded
   frequency (≥ 5 min), bounded retries.
6. Data residency respected (S2's residency validator becomes a schedule check).
7. Least privilege: new capabilities `schedules:*`, `errors:*`, `msgcat:*` in the
   existing RBAC.

---

## §3 — Slices

| # | Slice | Delivers |
|---|---|---|
| 0 | **Live end to end, data pipeline** | backend + DataFusion engine + worker running against `alpha` / Temporal on 100.84.50.65; migrations applied (D2); FactSet sample loaded through the UI into staging and a BO; proof recorded |
| 1 | **msgcat** | `internal/msgcat` + error envelope middleware; pipeline, scheduler and BO write errors moved to catalog codes; guard test: no `err.Error()` to clients in migrated handlers |
| 2 | **Scheduler core** | `internal/schedule` model, Temporal engine, calendar evaluation on MDM calendars, runners for pipeline + report + saved query; `schedule_runs`; API + MCP |
| 3 | **Scheduler UI** | `<ScheduleEditor>` in report builder, query builder, pipeline editor; Schedules console |
| 4 | **Retire the rest** | S1 deleted (unsafe); S3/S4/S5/S6/S10 migrated onto the core; S7 pages rebuilt on the core API or removed; S9 deleted; S2 per D1 |
| 5 | **Workflows + DAGs** | workflow and job-DAG targets (S2's DAG model on the core) |
| 6 | **Error management** | error log + fingerprint queue + console + routing |
| 7 | **Error bot** | explain / suggest / cluster / incident / notify; autofix under policy |
| 8 | **GSIFI hardening pass** | maker–checker, audit completeness, RLS, redaction review, guard tests |

Each slice ships with tests, a live check on the running backend, and a PR.

---

## §4 — Decisions needed

* **D1 — Scheduler Intelligence (S2).** (a) Keep its console as the central Schedules
  console, rebuilt on the single core, keeping changesets/approvals, blast radius,
  residency; or (b) new console, retire S2. *Recommended: (a).*
* **D2 — Live environment.** Run the backend, worker and DataFusion engine against
  `alpha` and Temporal on 100.84.50.65, and **apply the new migrations to `alpha`**
  (additive only: new tables/columns; no drops). Where should the DataFusion engine
  and its file volume live (the Docker host is recommended)? Note: the engine's
  default port 8081 is taken there by Redpanda.
* **D3 — Maker–checker.** Required for all tenants, or a per-tenant policy (default on)?
  *Recommended: per-tenant, default on for schedule and catalog changes.*
* **D4 — Bot and LLM.** May redacted error context go to the configured LLM (Gemini
  today)? Per-tenant opt-in? *Recommended: per-tenant opt-in, redaction mandatory,
  bot works without the LLM (catalog explanation + rules).*
* **D5 — Calendars (revised 2026-09-25, pending owner confirmation).** The scheduler
  reads only **published** MDM golden calendars (`calendar_golden_record` current +
  published, materialized `calendar_day`) through a read-only calendar service — never
  MDM working tables. Standard market calendars are published once in the **gold-copy
  tenant** and inherited read-only; tenants publish their own (e.g. fund dealing
  calendars) in their tenant. Gold-copy calendars are loaded through the MDM calendar
  pipeline fed by the data pipeline. A schedule warns when its calendar's published
  horizon is too short. `tenant_exchange_calendars` retires. (MDM calendar tables are
  empty today.)

---

**Signed:** ____________ **Date:** ________
