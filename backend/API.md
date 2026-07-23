# Backend API Documentation

## Catalog Scan

Endpoint: POST /api/catalog/scan

Description: Triggers catalog metadata scans for all datasources or a single datasource when `datasource_id` query param is provided.

Responses:

- 200 OK
  - When all requested datasource scans completed successfully.
  - Body:
    {
      "status": "success",
      "message": "All datasource scans completed successfully",
      "results": [
        { "datasource_id": "<uuid>", "name": "<name>", "success": true }
      ]
    }

- 207 Multi-Status
  - When some datasource scans succeeded and some failed.
  - Body includes per-datasource results, plus counts:
    {
      "status": "partial",
      "message": "Some datasource scans failed",
      "success_count": <n>,
      "failure_count": <m>,
      "results": [
        { "datasource_id": "<uuid>", "name": "<name>", "success": true },
        { "datasource_id": "<uuid>", "name": "<name>", "success": false, "error": "<err>" }
      ]
    }

- 500 Internal Server Error
  - When all requested datasource scans failed, the response includes details and results.
    {
      "status": "failure",
      "message": "All datasource scans failed",
      "details": "<aggregate error>",
      "results": [ ... ]
    }

Notes:

- The response always includes a `results` array with per-datasource status. This allows clients to programmatically inspect which scans failed and why.
- You can request a single datasource scan by calling: POST /api/catalog/scan?datasource_id=<uuid>

## Alpha Link (Tenant-scoped linking to gold copy)

Link non-gold catalog nodes to their gold-copy counterparts (sets core_id and is_alpha=true) for a single tenant, matched by qualified_path and node_type.

Endpoints:

- POST /api/tenants/:tenant_id/catalog/link-alpha
  - Links all node types for the tenant.
  - Response: { tenant_id, rows_affected }

- POST /api/tenants/:tenant_id/catalog/link-alpha/:node_type_id
  - Links only the specified node type.
  - Response: { tenant_id, node_type_id, rows_affected }

- Convenience POST endpoints for common types:
  - /api/tenants/:tenant_id/catalog/link-alpha/semantic-model
  - /api/tenants/:tenant_id/catalog/link-alpha/semantic-column
  - /api/tenants/:tenant_id/catalog/link-alpha/schema
  - /api/tenants/:tenant_id/catalog/link-alpha/database-column

Preview (dry-run) endpoints return which rows would be linked without updating:

- GET /api/tenants/:tenant_id/catalog/link-alpha/preview
- GET /api/tenants/:tenant_id/catalog/link-alpha/:node_type_id/preview
- Convenience GET previews mirror the POSTs above with /preview suffix.

Query params:
- limit (default 100), offset (default 0), count_only=true to return just a total.

Preview item shape: { id, node_type_id, qualified_path, candidate_core_id, gold_tenant_id }

Configuration:
- Convenience node type IDs can be overridden via environment variables:
  - SEMLAYER_NODETYPE_SEMANTIC_MODEL (default c53f9e99-8d02-4dfb-bc1b-914747d35edb)
  - SEMLAYER_NODETYPE_SEMANTIC_COLUMN (default 1439f761-606a-44cb-b4f8-7aa6b27a9bf5)
  - SEMLAYER_NODETYPE_SCHEMA (default 68d6d495-0992-4d92-ad2f-7f66dc1e7d78)
  - SEMLAYER_NODETYPE_DATABASE_COLUMN (default a64c1011-16e8-4ddf-b447-363bf8e15c9a)

## Business Terms Upsert (gold -> non-gold)

Behavior:
- After each successful catalog merge for a non-gold datasource, the backend upserts Business Term nodes from the gold copy into the target tenant/datasource.
- Matching is by natural key (qualified_path) and the Business Term node type.
- For new terms, a new node is created with core_id pointing to the gold term and is_alpha=true; for existing terms, name/description/properties are updated and core_id refreshed.
- Gold tenants are skipped.

Configuration:
- SEMLAYER_NODETYPE_BUSINESS_TERM can override the Business Term node type UUID (default 21645d21-de5f-4feb-af99-99273ea75626).

Safety:
- Catalog merges restrict deletions to technical node types only (schemas, tables, and columns). Semantic/business nodes are not deleted by merges.
