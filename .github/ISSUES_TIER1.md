# GitHub Issues — Tier 1 Gap Analysis

## Issue 1: APICallerTransformer — Phase 3 stub, no HTTP call
**Labels:** area:datapipeline, priority:high

```
Title: feat(datapipeline): implement APICallerTransformer — bind to API Studio registry, no arbitrary URLs

Body:

## Problem

`APICallerTransformer` in `backend/internal/datapipeline/transforms.go` stamps `{"verified": true}` without making any HTTP call. It is registered in the transform palette. A transformer that claims to call an API but doesn't is a silent correctness failure.

## Production Impact

Pipelines using `api_caller` produce outputs claiming verification was performed when no API was contacted. Downstream decisions are based on fabricated data.

## Fix Direction

1. Bind to the server-side API Studio endpoint registry (no arbitrary URLs — SSRF prevention)
2. Support service-held auth tokens (OAuth2 client credentials, API keys)
3. Cap responses at 1MB
4. Propagate real HTTP errors through the pipeline failure path

## Acceptance Criteria

- [ ] APICallerTransformer makes real HTTP call to registered endpoint
- [ ] SSRF: arbitrary URL rejected, only registry endpoints accepted
- [ ] Service-held auth tokens used (not caller-supplied credentials)
- [ ] Response >1MB rejected with appropriate error
- [ ] HTTP errors propagate through pipeline failure path
- [ ] go test -count=1 ./internal/datapipeline/... passes

Ref: STATUS_AUDIT_BACKLOG.md → `APICallerTransformer-Unimplemented`
```

---

## Issue 2: Outbox Publish Outside BO Write Transaction
**Labels:** area:datapipeline, priority:high

```
Title: fix(datapipeline): outbox PublishPipelineTrigger must participate in BO write transaction

Body:

## Problem

`PublishPipelineTrigger` opens its own transaction, independent of the BO write transaction. A BO write can roll back *after* the trigger event commits, causing a pipeline to fire for data that doesn't exist. This is an eventual-consistency bug at the exact seam event-driven pipelines are built on.

## Production Impact

Any pipeline wired to fire on BO create/update will fire spuriously when the originating write fails after the trigger event commits. At scale this makes event-driven automation unreliable.

## Fix Direction

Refactor `PublishPipelineTrigger` to accept the BO write's `*sqlx.Tx` and participate in the same transaction. One interface change — the outbox row and the BO write commit or roll back atomically. Pipelines fire only when the event that triggered them is durable.

## Acceptance Criteria

- [ ] PublishPipelineTrigger accepts *sqlx.Tx
- [ ] BO write and trigger event commit atomically
- [ ] BO write rollback prevents trigger event from being published
- [ ] Existing trigger consumers unchanged (interface change is backward-compatible or migrated)
- [ ] go test -count=1 ./internal/datapipeline/... passes

Ref: STATUS_AUDIT_BACKLOG.md → `Outbox-Event-Transactional-Gap`
```

---

## Issue 3: ColumnMapper Delete-On-Rename — Silent Data Loss
**Labels:** area:datapipeline, priority:medium

```
Title: fix(datapipeline): ColumnMapper — remove delete-on-rename, make move semantics opt-in

Body:

## Problem

`ColumnMapper.Transform` in `backend/internal/datapipeline/transforms.go` deletes source fields when mapping to a different name:

Given `Mappings: {"out_name": "node_name"}`:
1. `transformed[targetKey] = val` — copies value to `out_name`
2. `delete(transformed, srcKey)` — deletes `node_name`

Any downstream node expecting both `out_name` AND `node_name` receives only `out_name`. The data loss is silent — no error, no warning.

## Production Impact

Every pipeline author configuring `column_mapper` with rename-style mappings will silently lose the source field. Downstream loaders and transforms get empty values.

## Fix Direction

Remove `delete(transformed, srcKey)`. Default-copy behavior (copy all fields, then apply renames) gives rename-without-loss semantics that pipeline authors would expect. If move semantics are genuinely needed, make it an explicit opt-in option.

## Acceptance Criteria

- [ ] Rename mapping copies source field without deleting it
- [ ] Downstream nodes receive both source and target fields
- [ ] Explicit move semantics available as opt-in (if implemented)
- [ ] diamond_persistence_test.go continues to pass (already uses self-referential mappings)
- [ ] go test -count=1 ./internal/datapipeline/... passes

Ref: STATUS_AUDIT_BACKLOG.md → `ColumnMapper-Delete-On-Rename`
```

---

## Issue 4: Trigger Surface — Create-Only, No Visibility or Lifecycle
**Labels:** area:studio, priority:medium

```
Title: feat(studio): trigger lifecycle — list, activate/deactivate, firing history

Body:

## Problem

The trigger-binding UI (TriggerAuthoringPage) can create bindings but cannot edit, deactivate, or view firing history. Event-driven pipelines stop being a production feature when operators cannot see which events fired which runs, and cannot control when they fire.

## Fix Direction

Add per-trigger:
1. **List view** with last-fired timestamp and run link
2. **Activate/deactivate toggle** — paused triggers don't fire pipelines
3. **Run history page** — which runs were triggered by which events

## Acceptance Criteria

- [ ] Trigger list shows all triggers for a pipeline with last-fired timestamp
- [ ] Trigger list links to the runs triggered by each event
- [ ] Activate/deactivate toggle persists and prevents firing when deactivated
- [ ] Run history page shows the triggering event payload for each run
- [ ] E2E or integration test covers the trigger lifecycle

Ref: STATUS_AUDIT_BACKLOG.md → `TriggerSurface-CreateOnly`
```
