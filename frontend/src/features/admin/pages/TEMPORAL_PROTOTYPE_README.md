Temporal Prototype pages
=======================

Location
- `frontend/src/features/admin/pages/TemporalPrototypePage.tsx` — host page at `/admin/temporal-prototype`

Components
- `frontend/src/components/temporal/WorkflowDesigner.tsx` — ReactFlow-based designer (prototype)
- `frontend/src/components/temporal/ExecutionMonitor.tsx` — lists recent executions via `temporalService.listExecutions()`
- `frontend/src/components/temporal/DebugPanel.tsx` — debug controls (AMQP metrics, publish test event)
- `frontend/src/components/temporal/LiveEventsWidget.tsx` — live trigger events list

How it wires to the backend
- All HTTP calls use tenant-scoped headers via `frontend/src/services/temporalService.ts`.
- Key backend endpoints used by the UI:
  - `GET /api/temporal/executions` — lists recent executions (admin-only)
  - `GET /api/_debug/amqp-metrics` — AMQP metrics (dev/debug)
  - `POST /api/_debug/publish-event` — publish a test event (dev/debug)
  - `GET /api/v1/triggers/events` — list trigger events

Running locally (frontend)
1. Install deps and start dev server:
```bash
cd frontend
npm ci
npm run dev
```
2. Open the app and navigate to `/admin/temporal-prototype`.

Notes and next steps
- ABAC: UI uses the app `useABAC()` for gating (e.g. publishing test events and saving workflows).
- Designer: `WorkflowDesigner` is a prototype. It requires `reactflow` (already in `package.json`) for the visual editor. The Save action currently validates ABAC and logs the nodes/edges; integrate with `temporalService.signalWorkflow` or a backend save endpoint to persist workflows.
- Backend: Ensure debug endpoints are enabled in dev and tenant headers are set (see Tenant Runbook in `agents.md`).
