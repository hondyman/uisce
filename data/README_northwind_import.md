Northwind Catalog & Entities - Import Guide

Files included:
- northwind_catalog_nodes.json  -> semantic catalog nodes (semantic terms)
- northwind_catalog_edges.json  -> edges linking catalog nodes to DB columns
- northwind_entities.json       -> Entity/BO definitions with subtypes and field links

Requirements
- Backend running locally (default: http://localhost:29080)
- Tenant + datasource scope headers/params (see Agent Runbook `agents.md`) or seed localStorage for UI imports

Quick curl templates (replace TENANT_ID and DATASOURCE_ID):

# Import catalog nodes
curl -X POST \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: <TENANT_ID>" \
  -H "X-Tenant-Datasource-ID: <DATASOURCE_ID>" \
  "http://localhost:29080/api/catalog/nodes?tenant_id=<TENANT_ID>&datasource_id=<DATASOURCE_ID>" \
  --data-binary @data/northwind_catalog_nodes.json

# Import catalog edges
curl -X POST \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: <TENANT_ID>" \
  -H "X-Tenant-Datasource-ID: <DATASOURCE_ID>" \
  "http://localhost:29080/api/catalog/edges?tenant_id=<TENANT_ID>&datasource_id=<DATASOURCE_ID>" \
  --data-binary @data/northwind_catalog_edges.json

# Import entities (BOs, subtypes, fields)
curl -X POST \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: <TENANT_ID>" \
  -H "X-Tenant-Datasource-ID: <DATASOURCE_ID>" \
  "http://localhost:29080/api/bundles/entities?tenant_id=<TENANT_ID>&datasource_id=<DATASOURCE_ID>" \
  --data-binary @data/northwind_entities.json

Notes
- The API endpoints above are examples; confirm actual endpoint paths in `backend/internal/api/api.go` if different.
- The runbook requires tenant+datasource for bundle/catalog endpoints. If using the UI, set `localStorage` keys `selected_tenant` and `selected_datasource` as documented.

Next steps
- Run the curl imports. Check backend logs for errors.
- If edges require a different format, adapt `northwind_catalog_edges.json` to the backend schema.
- After import, verify entities show in UI and fields link to semantic terms in the catalog.
