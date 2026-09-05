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
