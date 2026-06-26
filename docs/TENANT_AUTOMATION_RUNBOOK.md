# Tenant Automation Runbook (Phase 5)

This guide explains how Platform Engineering and SRE operate the tenant automation job that keeps Cube schemas, resource groups, and `tenant_datasources` records in sync.

## 1. Components

| Item | Location | Purpose |
| --- | --- | --- |
| Provisioning CLI | `backend/cmd/tenant_automation` | Runs the automation end-to-end (DB sync + schema files). |
| Provisioner library | `backend/internal/tenantauto` | Query + file writer + DB upsert logic. |
| Generated artifacts | `cube/generated/tenant-scopes.json` | Snapshot consumed by Cube scheduler + QoS router. |
| Per-tenant overrides | `cube/schema/tenants/<tenant>/<datasource>/auto/` | Auto-written schema overrides + metadata. |
| Tracking tables | `tenant_datasources`, `tenant_provision_jobs` | Resource group defaults, idempotency, alerts. |

## 2. Prerequisites

1. Database migrations applied (includes `20251130_007_tenant_automation.sql`).
2. `ALPHA_DB_URL` (or `DATABASE_URL`) reachable from runner host.
3. `cube/schema/tenants` and `cube/generated` paths writable by the job.
4. PagerDuty/Grafana wired to `tenant_provision_jobs` (see Alerts section).

## 3. Running the Job

### Dry Run (safe preview)
```bash
cd backend
go run ./cmd/tenant_automation \\
  -dry-run \\
  -tenants="9106...,c52a..." \
  -datasources="982a...,f938..."
```
Outputs summary without touching DB/files—use before risky changes.

### Full Sync (all active tenants)
```bash
cd backend
go run ./cmd/tenant_automation \
  -triggered-by="nightly-sync" \
  -timeout=10m
```
The CLI:
- Pulls active `tenant_product_datasource` rows
- Writes overrides + metadata under `cube/schema/tenants`
- Upserts `tenant_datasources` (resource group, metadata, status)
- Updates `tenant_provision_jobs`
- Regenerates `cube/generated/tenant-scopes.json`

### Targeted Sync
```bash
go run ./cmd/tenant_automation \
  -tenants=910638ba-a459-4a3f-bb2d-78391b0595f6 \
  -datasources=982aef38-418f-46dc-acd0-35fe8f3b97b0
```
Use when onboarding a single tenant or datasource.

## 4. Operational Alerts

| Check | Query | Action |
| --- | --- | --- |
| Failed jobs | `SELECT * FROM tenant_provision_jobs WHERE status='failed' ORDER BY updated_at DESC LIMIT 20;` | Review `last_error`, rerun job with `-tenants/-datasources`, confirm success. |
| Stale jobs | `updated_at < now() - interval '24 hours'` | Ensure automation pipeline/cron is still executing. |
| Resource group drift | `SELECT tenant_id,datasource_id FROM tenant_datasources WHERE resource_group IS NULL;` | Rerun job or backfill defaults. |

Hook Grafana/Alertmanager to:
- Alert if `COUNT(*) FILTER (WHERE status='failed') > 0` for 10 minutes.
- Alert if `MAX(updated_at)` older than 24 hours.

## 5. Recovery Steps

1. **Job failure**: re-run CLI in dry-run to validate, then re-run with same filters. If failure persists, inspect DB row and schema overrides (likely malformed JSON in `schema_overrides`).
2. **File corruption**: remove offending `cube/schema/tenants/<tenant>/<datasource>/auto` directory and rerun job.
3. **DB constraint issue**: verify migrations, especially unique `(tenant_id, datasource_id)` enforcement on `tenant_datasources`.

## 6. Change Management

- Automation runs nightly (cron/Temporal) with `-triggered-by=nightly-run`.
- Manual runs must note change ticket in `-triggered-by` for auditing.
- Before cutting to production, run dry-run + targeted run for new tenants.

## 7. Service Ownership

| Task | Owner |
| --- | --- |
| CLI / Go code | Platform Engineering |
| Tracking dashboards + PagerDuty | SRE |
| Schema overrides authoring | Data Modeling |
| QA regression (tenant isolation) | QA / Support |

## 8. Frequently Asked Questions

**Q: Where do schema overrides come from?**  
`tenant_product_datasource.config.schema_overrides` JSON blobs. Automation writes each file under `/auto` and records paths in metadata.

**Q: Can we run in parallel with observability work?**  
Yes. Job is idempotent and re-runnable; Observability (Phase 6) only needs `tenant_provision_jobs` table filled.

**Q: How do we add a new resource group default?**  
Update `tenant_datasources.resource_group` directly or embed `resource_group` in datasource/instance config; the job keeps using that value.

---
Document owners: Platform Eng + SRE. Update whenever the CLI gains new flags or schema columns.
