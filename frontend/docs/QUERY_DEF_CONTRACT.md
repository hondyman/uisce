# Alpha Query Builder — QueryDef Contract

This document defines the JSON contract between the Alpha Query Builder UI and the backend Resolution Engine / SQL Generator.

## Principle

The UI never constructs SQL. It sends a **Query Definition (QueryDef)**. The backend is the only place that translates semantic intent into dialect-specific SQL.

## Endpoints

### 1. List semantic terms for a binding

```http
GET /api/business-objects/{boId}/terms?bindingId={bindingId}
```

**Headers**

- `Authorization: Bearer <jwt>`
- `X-Tenant-ID: <tenantId>`
- `X-Tenant-Datasource-ID: <datasourceId>` (optional, legacy)

**Response**

```json
{
  "terms": [
    {
      "termNodeId": "order_date",
      "termKey": "order_date",
      "termName": "Order Date",
      "displayName": "Order Date",
      "description": "Date the order was placed",
      "dataType": "date",
      "role": "DIMENSION",
      "bindingStatus": "RESOLVED",
      "defaultAggregation": null
    },
    {
      "termNodeId": "order_total",
      "termKey": "order_total",
      "termName": "Order Total",
      "displayName": "Order Total",
      "dataType": "decimal",
      "role": "MEASURE",
      "bindingStatus": "RESOLVED",
      "defaultAggregation": "SUM"
    }
  ]
}
```

Rules:

- Only return terms whose `field_binding` status is `RESOLVED` for the requested `bindingId`.
- Categorize by `role`: `DIMENSION`, `MEASURE`, `CALCULATED`.

### 2. Preview SQL

```http
POST /api/query/preview
```

**Request body** — a `QueryDef` (see schema below).

**Response**

```json
{
  "sql": "SELECT \"order_date\" AS \"date\", SUM(\"order_total\") AS \"revenue\" FROM \"orders\" WHERE \"tenant_id\" = ? GROUP BY \"date\" LIMIT 1000;",
  "dialect": "postgres",
  "parameters": ["tenant_123"]
}
```

### 3. Execute query

```http
POST /api/query/execute
```

**Request body** — a `QueryDef`.

**Response**

```json
{
  "sql": "SELECT ...",
  "columns": [
    { "name": "date", "type": "date" },
    { "name": "revenue", "type": "decimal" }
  ],
  "rows": [
    { "date": "2026-01-01", "revenue": 12500.00 }
  ],
  "rowCount": 1,
  "executionTimeMs": 42
}
```

## QueryDef JSON Schema

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "QueryDef",
  "type": "object",
  "required": ["context", "query"],
  "properties": {
    "context": {
      "type": "object",
      "required": ["boId", "bindingId", "tenantId"],
      "properties": {
        "boId": {
          "type": "string",
          "format": "uuid",
          "description": "Business Object that provides the semantic entry point"
        },
        "bindingId": {
          "type": "string",
          "format": "uuid",
          "description": "Resolved binding for the selected tenant/product/datasource"
        },
        "tenantId": {
          "type": "string",
          "format": "uuid",
          "description": "Tenant that owns the binding and the data"
        }
      }
    },
    "query": {
      "type": "object",
      "required": ["dimensions", "measures", "filters"],
      "properties": {
        "dimensions": {
          "type": "array",
          "items": {
            "type": "object",
            "required": ["termNodeId", "alias"],
            "properties": {
              "termNodeId": { "type": "string" },
              "alias": { "type": "string" }
            }
          }
        },
        "measures": {
          "type": "array",
          "items": {
            "type": "object",
            "required": ["termNodeId", "alias", "agg"],
            "properties": {
              "termNodeId": { "type": "string" },
              "alias": { "type": "string" },
              "agg": {
                "type": "string",
                "enum": ["SUM", "AVG", "MIN", "MAX", "COUNT", "COUNT_DISTINCT", "NONE"]
              }
            }
          }
        },
        "filters": {
          "type": "array",
          "items": {
            "type": "object",
            "required": ["termNodeId", "operator"],
            "properties": {
              "termNodeId": { "type": "string" },
              "operator": {
                "type": "string",
                "enum": [
                  "eq", "neq", "gt", "gte", "lt", "lte",
                  "contains", "starts_with", "ends_with",
                  "in", "not_in", "is_null", "is_not_null", "between"
                ]
              },
              "value": {
                "oneOf": [
                  { "type": "string" },
                  { "type": "number" },
                  { "type": "boolean" },
                  {
                    "type": "array",
                    "items": { "type": ["string", "number"] }
                  }
                ]
              }
            }
          }
        },
        "groupBy": {
          "type": "array",
          "items": { "type": "string" },
          "description": "Aliases to group by. Defaults to all dimensions."
        },
        "limit": {
          "type": "integer",
          "minimum": 1,
          "default": 1000
        }
      }
    }
  }
}
```

## Backend Responsibilities

### 1. Binding Resolution

The user context provides (Tenant, Product, Datasource). The API must resolve the unique `bindingId` that maps this triplet to a physical backend.

### 2. Tenant Ownership Check

Before executing or previewing any query, call:

```go
RequireTenantOwnership(ctx, pool, tenantId, bindingId, "binding")
```

This ensures the user cannot query a datasource they do not own.

### 3. Tenant Filter Injection

The SQL Generator must prepend the tenant filter to every query. Example:

```sql
WHERE "tenant_id" = ?
```

Use parameterized queries; never concatenate the tenant id into the SQL string.

### 4. Semantic Expansion

For every `termNodeId` in the QueryDef, look up its `field_binding` for the resolved `bindingId`. This yields the physical column, table, and SQL expression.

### 5. Join Traversal

If the QueryDef selects terms from multiple Business Objects, the generator must traverse the `relationship_binding` graph and incorporate the pre-defined join SQL into the `FROM/JOIN` clauses.

### 6. Dialect Translation

Apply the correct identifier quoting, bind placeholders, and function translations for the target backend (Postgres, Snowflake, Iceberg, etc.).

## UI Guarantees

- The UI only sends `QueryDef` objects.
- The UI never concatenates SQL.
- The UI only shows terms with `bindingStatus === "RESOLVED"`.
- The UI resolves the `bindingId` dynamically from the tenant/product/datasource context.

## Security Rules

| Rule | Owner |
|------|-------|
| Binding resolution | Backend |
| Tenant ownership enforcement | Backend (`RequireTenantOwnership`) |
| Tenant filter injection | Backend SQL Generator |
| SQL construction | Backend only |
| Term visibility filtering | Backend (only RESOLVED terms) + UI |
