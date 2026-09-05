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
