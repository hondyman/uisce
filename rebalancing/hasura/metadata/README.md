# Hasura Metadata

This directory contains Hasura metadata configuration.

## Permissions

Example permissions (set via Hasura Console or metadata):

```json
{
  "table": "portfolios",
  "role": "portfolio_manager",
  "permission": {
    "columns": ["id", "name", "aum", "drift", "last_rebalance", "tax_saved", "rebalance_status"],
    "filter": {
      "tenant_id": {
        "_eq": "X-Hasura-Tenant-Id"
      }
    },
    "allow_aggregations": true
  }
}
```

```json
{
  "table": "rebalance_plans",
  "role": "portfolio_manager",
  "permission": {
    "columns": "*",
    "filter": {
      "portfolio": {
        "tenant_id": {
          "_eq": "X-Hasura-Tenant-Id"
        }
      }
    }
  }
}
```

## Relationships

-   `portfolios` -> `holdings` (one-to-many)
-   `portfolios` -> `rebalance_plans` (one-to-many)
