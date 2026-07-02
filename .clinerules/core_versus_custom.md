## Plan: Ensure the binding wizard respects the core/custom (gold copy) inheritance model

You're right — `gold_copy = true` is the **core** tenant. Every other tenant inherits everything from it and may **override** (replace by ID/qualified_path) or **add custom** nodes on top. The wizard must reflect this when the user picks a datasource and a driving table.

### Current state in `page.tsx`

There's a bug in my last edit:

```js
const goldCopyTenantId = currentTenant?.gold_copy
  ? null
  : null; // ← both branches are null!
```

This means I'm only fetching from the current tenant and dropping the gold copy merge. The `GetCatalogNodes` endpoint already merges gold-copy + custom automatically, but the `setup.datasources` list (raw SQL of `tenant_datasources`) does NOT — so I need to include the gold copy tenant ID explicitly when filtering that.

### Fix plan

**File: `uisce_frontend/apps/genui-frontend/app/core/business-objects/new/page.tsx`**

1. Resolve the gold copy tenant ID properly:
   ```js
   const goldCopyTenant = allTenants.find((t) => t.gold_copy === true);
   const goldCopyTenantId = goldCopyTenant?.id || null;
   ```
   Then `tenantIdsToInclude = [tenantId, goldCopyTenantId].filter(Boolean)` — only add gold copy when current tenant is NOT the gold copy itself.

2. Pass `goldCopyTenantId` down to `BindingsStep` (new prop) so the wizard knows which datasource entries are "inherited from core" vs "custom".

3. Keep `getCatalogNodes` calls as a single call per node-type — the backend already merges gold-copy + custom nodes, returning them with `tenant_id` so we can mark `isGoldCopy` per row.

**File: `uisce_frontend/apps/genui-frontend/app/core/business-objects/new/components/BindingsStep.tsx`**

4. When the user picks a tenant in Step 1 of the wizard, the cascade filter must use that tenant's scope:
   - If the selected tenant IS the gold copy tenant → show gold copy datasources + their tables (no "custom" since gold copy has no overrides)
   - If the selected tenant is a regular tenant → show datasources owned by that tenant OR by gold copy (since custom inherits core)

5. For the **driving-table** Autocomplete, ensure the option list is the union of:
   - Tables owned by the selected tenant (their custom + overrides)
   - Tables owned by the gold copy tenant (their core, shown with a "core" badge)

6. For the **related tables** picker, when the driving table is a gold-copy (core) table:
   - Query catalog edges from gold copy tenant
   - Otherwise query from current tenant
   - Always merge the two lists

7. Pass the resolved `goldCopyTenantId` into the edge-fetch call so related-table discovery routes to the correct scope:
   ```js
   const edgesTenantId = currentTenant.gold_copy
     ? currentTenant.id
     : (bindingTenantId === goldCopyTenantId ? goldCopyTenantId : tenantId);
   ```

### What's already correct
- The BindingsStep already accepts `tenants`, `instances`, `tenantProductDatasources` and filters cascading dropdowns.
- Tables fetched from `getCatalogNodes(tenantId, ...)` already include gold-copy merged nodes.
- The save payload already sends `backend_id` (UUID PK) which matches the FK constraint.

### Risk assessment
- **Low**: Adding the gold copy tenant ID to the filter is additive — current tenants always still see their own datasources, plus any inherited ones.
- **Medium**: Need to verify that `setup.datasources` rows for the gold copy tenant are present in the current dataset. From the curl sample I can see `"tenant_id":"99e99e99-..."` rows exist with `gold_copy: true`, so this should work.
- **Low**: For related-table discovery, calling `getCatalogEdges(goldCopyTenantId, ...)` requires the user to have read access to gold copy tables — which they should since gold copy metadata is public to all tenants.

### Verification after switching to Act mode
1. Open the wizard on a non-gold tenant (e.g., `Northwinds`). The **Tenant** dropdown should list all tenants including `northwind` (gold copy).
2. Pick a gold copy tenant → the **Datasource** dropdown shows the gold copy's datasources, marked "· gold copy".
3. Pick a regular tenant → the **Datasource** dropdown shows BOTH their own datasources AND the gold copy datasources, each marked accordingly.
4. Pick a driving table → the **Related Tables** picker fetches edges from the right tenant scope and lists the related tables.
5. Save the BO → backend stores the binding with the correct `backend_id` (UUID PK) and no FK violation.

---

## Plan: Leverage server-side core cache and eliminate redundant fetches

You're right — core (gold copy tenant) data is mostly static and only changes on upgrades. The backend **already has** the right caching primitive:

### What's already in place

In `uisce_backend/services/uisce_core/internal/handlers/setup_handler.go`:

```go
func (h *SetupHandler) loadCoreNodes(ctx context.Context, goldTenantID string) ([]metadatacache.CachedCatalogNode, error) {
    if h.cache != nil {
        nodes, err := h.cache.GetCoreNodes(ctx)   // ← read from cache first
        if err == nil && len(nodes) > 0 {
            return nodes, nil
        }
    }
    // ... only fall back to DB if cache miss ...
    if h.cache != nil && len(nodes) > 0 {
        _ = h.cache.SetCoreNodes(ctx, nodes)     // ← write-through after fetch
    }
    return nodes, nil
}
```

`metadatacache.Cache` is provided by `uisce_backend/libs/metadata-cache` and:
- Holds core catalog nodes + edges in memory
- Is invalidated via `checkAndInvalidateCore()` whenever the gold copy tenant's catalog is mutated
- Is shared across **all** tenant requests (single in-process cache)

### The bug in my last edit

In `page.tsx`, I currently make **2× N parallel** `getCatalogNodes` calls (current tenant + gold copy tenant):

```js
const tableArrays = await Promise.all(
  tenantIdsToInclude.map((tid) =>
    Promise.all([getCatalogNodes(tid, 'database_table'), getCatalogNodes(tid, 'table')]),
  ),
);
```

This is **redundant** because the backend's `GetCatalogNodes` already does the merge server-side:

```go
} else {  // non-gold tenant
    coreNodes, _ := h.loadCoreNodes(ctx, goldTenantID)   // ← from cache
    customRows, _ := h.db.QueryContext(ctx, "SELECT ... WHERE n.tenant_id = $1", tenantUUID.String())
    // ... merge with custom overriding core by qualified_path/node_name ...
}
```

So one call from the frontend for the current tenant already returns merged core+custom data.

### Fix plan

**File: `uisce_frontend/apps/genui-frontend/app/core/business-objects/new/page.tsx`**

1. **Drop the parallel gold-copy fetch for catalog nodes** — just call `getCatalogNodes(tenantId, ...)` once per node-type. The backend's cache + merge gives the same result faster.

2. **Keep the `tenantIdsToInclude` filter for `setup.datasources`** — this list is raw SQL, no server-side merge. We do need to include both tenants to see inherited gold-copy datasources.

3. **Keep schemas + databases fetch as-is** — they're needed for dbName resolution.

4. **Optional: add a session-scoped frontend cache** in the page module so revisiting the wizard doesn't re-fetch everything within the same session:
   ```js
   const sessionCache = {
     tables: null as any[] | null,
     datasources: null as any[] | null,
     tenants: null as any[] | null,
   };
   ```

### Backend validation

5. **Verify** the `metadata-cache` library is initialized with the `uisce_core.SetupHandler` and the cache size / TTL are appropriate. Check `setup_handler.go` constructor and the `libs/metadata-cache` implementation.

6. **Add a small invalidation hook** in the catalog-edge and catalog-node mutations to ensure when an upgrade ships new core content, the cache is busted automatically. Looking at the existing code, `checkAndInvalidateCore(ctx, tenantID)` is already wired into `CreateCatalogNode`, `UpdateCatalogNode`, `DeleteCatalogNode`, `CreateCatalogEdge`, `DeleteCatalogEdge` — that's correct.

7. **Optional: add a `Cache-Control: max-age=...` HTTP header** on the `/api/admin/setup/catalog/nodes` response so the browser can cache across page loads.

### Related-tables discovery

8. **Keep `edgesTenantId` per driving table** — that's still correct since core edges live with the gold copy tenant, and custom edges live with the current tenant. The backend doesn't merge edges between tenants, so we route correctly.

### Risk assessment

- **Low**: Dropping the redundant `getCatalogNodes` call — backend already merges.
- **Low**: Adding a session cache is purely additive; if invalidation matters we can key it off the URL params.
- **Medium**: Browser-level `Cache-Control` headers could cause staleness if a wizard is open during an upgrade — mitigation: only set `max-age` short (60s), and rely on the wizard's existing re-fetch on `editId`/`tenantId` change.

### Verification after switching to Act mode

1. Open DevTools Network tab on the wizard → confirm only **one** `catalog/nodes?type=database_table` call per node type per tenant load (not two).
2. Verify the response includes gold-copy tables merged in for a non-gold tenant (e.g., `Northwinds` should see `northwind` gold-copy tables).
3. Upgrade `northwind` gold-copy tenant's catalog → open wizard on `Northwinds` → confirm the new table appears (cache invalidation works).
4. Confirm backend logs show no duplicate SQL fetches of core nodes on the second request within the cache TTL window.

---

