# frontend-temporal

Small Vite + React prototype for Temporal UI features (designer, executions, namespaces, debug).

Quick start

```bash
cd frontend-temporal
npm ci
npm run dev
```

Navigate to the printed Vite URL (http://localhost:5173 by default).

Seeding tenant scope for development

The main backend requires tenant scope for bundle/trigger endpoints. For local development you can seed `localStorage` as follows in the browser console before using the UI:

```js
localStorage.setItem('selected_tenant', JSON.stringify({ id: '00000000-0000-0000-0000-000000000000', display_name: 'Dev Tenant' }))
localStorage.setItem('selected_datasource', JSON.stringify({ id: '11111111-1111-1111-1111-111111111111', source_name: 'dev' }))
location.reload()
```

Tests

```bash
cd frontend-temporal
npm ci
npm test
```

CI

A GitHub Actions workflow `frontend-temporal.yml` is included in the repository to run `npm ci` and `npm run build` for the prototype. It runs on node 20 and caches node_modules for speed.

Integration notes

- `useABAC` calls `/api/abac/evaluate` and expects either a JSON `{ allowed: true|false }` or uses HTTP status codes (200 = allow, 403/401 = deny). Make sure the backend implements that contract.
- The Debug panel calls `/api/_debug/amqp-metrics`, `/api/v1/triggers/events` and `/api/_debug/publish-event` to inspect AMQP and publish test events. These endpoints exist in the backend debug adapters.

