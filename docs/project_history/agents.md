# Agent Runbook: Tenant-Scoped Fabric Bundles

This guide captures the minimum context every automation agent needs before working on bundle CRUD, policies, or semantic object features inside the Fabric Builder stack. Keep it handy so you never have to wait for manual instructions again.

## 🔐 Mandatory Tenant Scope

All Fabric Builder features now require a tenant and tenant datasource before they will load or mutate data. The frontend patches `window.fetch` (see `frontend/src/setupTenantFetch.ts`) to enforce the scope automatically:

- Every `/api/...` request (except auth/health/system helpers) is blocked until a tenant + datasource are cached.
- When a scope exists, the shim adds query parameters and headers:  
  `?tenant_id=<TENANT_ID>&datasource_id=<DATASOURCE_ID>` and  
  `X-Tenant-ID: <TENANT_ID>` / `X-Tenant-Datasource-ID: <DATASOURCE_ID>`.
- Requests that bypass the shim must add the same parameters and headers manually or they will be rejected by the backend.

### Selecting the Scope in the UI

1. Use the tenant picker in the Fabric Builder shell to choose the tenant, product, and datasource you want.  
2. The selection is cached in `localStorage` under the keys exported from `TenantContext`:
   - `selected_tenant`
   - `selected_product`
   - `selected_datasource`
3. Once selected, bundle features automatically read from that cache. Clearing any key forces a reselect.

> **Headless tip:** For scripted browser sessions you can pre-populate the cache:
>
> ```js
> localStorage.setItem('selected_tenant', JSON.stringify({ id: '...', display_name: '...' }));
> localStorage.setItem('selected_product', JSON.stringify({ id: '...', alpha_product: { product_name: '...' } }));
> localStorage.setItem('selected_datasource', JSON.stringify({ id: '...', source_name: '...' }));
> ```
>
> Reload after seeding to activate the scope.

### Calling APIs Directly

All bundle-, policy-, semantic-object-, and catalog-related endpoints require the scope. Example:

```bash
curl -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000000" \
     -H "X-Tenant-Datasource-ID: 11111111-1111-1111-1111-111111111111" \
     "http://localhost:8080/api/bundles?tenant_id=00000000-0000-0000-0000-000000000000&datasource_id=11111111-1111-1111-1111-111111111111"
```

## 📦 Bundle + Policy Workflows

- **Bundle list (`frontend/src/pages/bundles/BundleListPage.tsx`)** will not fetch until a tenant scope exists. The page shows a warning and disables creation until you select one.
- **Bundle editor (`frontend/src/pages/bundles/BundleEditor.tsx`)** reflects the active scope and only searches semantic objects when both IDs are present. The object browser and search inputs stay disabled otherwise.
- The backend (`backend/internal/api/api.go`) always pulls `tenant_id` / `datasource_id` from the query string; missing values short-circuit the request.

With the mandatory scope in place, bundle CRUD, policy updates, and semantic object lookups all remain tenant-safe by default.

## 🗄️ Local Postgres Reference

The development database runs locally and is already wired into `config.yaml`:

| Item              | Value                                         |
| ----------------- | --------------------------------------------- |
| Host              | `host.docker.internal` (or `localhost` inside Docker) |
| Port              | `5432`                                        |
| Database          | `alpha`                                       |
| User              | `postgres`                                    |
| Password          | `postgres`                                    |
| SSL               | Disabled (`sslmode=disable`)                  |

You can connect with psql:

```bash
psql postgres://postgres:postgres@host.docker.internal:5432/alpha?sslmode=disable
```

## ✅ Quick Checklist Before Running Automations

1. Pick tenant + datasource via the Fabric Builder selector (or seed `localStorage`).
2. Verify the scope with `window.localStorage.getItem('selected_tenant')` etc.
3. Confirm bundle pages load without the "Select a tenant" warning.
4. Run your bundle/policy steps.

Following this playbook keeps every run tenant-safe and ensures agents have the same baseline context as interactive users.
