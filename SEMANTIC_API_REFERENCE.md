# SemLayer Semantic API Reference

Complete documentation of all semantic layer endpoints for querying, managing, and extending semantic models, cubes, and business-aligned data assets.

---

## Authentication & Headers

All endpoints require these headers (except noted otherwise):

| Header | Type | Required | Example | Purpose |
|--------|------|----------|---------|---------|
| `X-Tenant-ID` | UUID | Yes | `550e8400-e29b-41d4-a716-446655440000` | Tenant context (multi-tenancy isolation) |
| `X-Tenant-Instance-ID` | UUID | Optional | `660e8400-e29b-41d4-a716-446655440000` | Datasource/Instance context |
| `X-User-ID` | UUID | Optional | `770e8400-e29b-41d4-a716-446655440000` | User context for audit |
| `Content-Type` | Header | Varies | `application/json` | Request/response content type |

---

## Core Semantic Layer Endpoints

### 1. Semantic Bundles

#### GET /api/semantic/bundles/{domain}
**Retrieve a semantic bundle by domain (functions, metrics, assets)**

- **Path Parameters:**
  - `domain` (string, required): Domain name (e.g., "financial_services", "capital_markets")

- **Headers:** None required (fallback to empty bundle if missing)

- **Response:** 200 OK
```json
{
  "bundle_id": "capital_markets",
  "domain": "capital_markets",
  "audience": ["analysts", "traders"],
  "version": "v1.0.0",
  "owner": "patrick",
  "tags": ["trading", "equities", "fixed-income"],
  "functions": [
    {
      "name": "calc_ytm",
      "class": "FixedIncomeCalculations",
      "badge": "builtin",
      "description": "Calculate yield to maturity"
    }
  ],
  "metrics": [
    {
      "node_id": "metric-1",
      "category": "returns",
      "description": "Total return metric",
      "financial_calc": {
        "type": "formula",
        "formula": "SUM(returns) / AVG(portfolio_value)",
        "arguments": ["returns", "portfolio_value"]
      },
      "badge": "core",
      "function_class": "ReturnsCalculation",
      "functions_used": ["sum", "avg"],
      "governance": {
        "status": "approved"
      }
    }
  ]
}
```

---

### 2. Semantic Objects

#### GET /api/semantic/objects
**List all semantic objects (dimensions, measures, hierarchies) for tenant/datasource**

- **Query Parameters:**
  - `tenant_id` (string, optional): Tenant ID filter
  - `datasource_id` (string, optional): Datasource ID filter

- **Headers:** `X-Tenant-ID`, `X-Tenant-Instance-ID` (or use query params)

- **Response:** 200 OK
```json
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "customer_count",
    "type": "measure",
    "display_name": "Customer Count",
    "description": "Total number of unique customers",
    "cube_name": "customers",
    "datasource_id": "660e8400-e29b-41d4-a716-446655440000"
  },
  {
    "id": "550e8400-e29b-41d4-a716-446655440001",
    "name": "order_date",
    "type": "dimension",
    "display_name": "Order Date",
    "description": "Date of the order",
    "cube_name": "orders",
    "data_type": "date"
  }
]
```

---

## Semantic Models (Core Templates & Custom Models)

### 3. Core Models (Templates)

#### GET /api/semantic-models/core
**Retrieve all available core semantic models (templates)**

- **Headers:** None required

- **Response:** 200 OK
```json
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "customer_360",
    "display_name": "Customer 360 View",
    "description": "Complete customer profile with transactions and interactions",
    "version": "2.1.0",
    "status": "published",
    "dimensions": 15,
    "measures": 8,
    "created_at": "2024-01-15T10:30:00Z"
  }
]
```

---

### 4. Tenant Models

#### GET /api/semantic-models/tenant
**Retrieve custom models for a specific tenant**

- **Query Parameters:**
  - `tenant_id` (string, required): Tenant UUID

- **Response:** 200 OK
```json
[
  {
    "id": "770e8400-e29b-41d4-a716-446655440000",
    "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
    "core_model_id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "customer_360_extended",
    "display_name": "Customer 360 (Extended)",
    "description": "Customized version with additional fields",
    "status": "active",
    "created_at": "2024-02-01T09:15:00Z",
    "inheritance_depth": 1
  }
]
```

---

### 5. Provision Model

#### POST /api/semantic-models/provision
**Create a custom model from a core template**

- **Headers:** `X-Tenant-ID`, `X-Tenant-Instance-ID`

- **Request Body:**
```json
{
  "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
  "core_cube_id": "550e8400-e29b-41d4-a716-446655440000",
  "datasource_id": "660e8400-e29b-41d4-a716-446655440000"
}
```

- **Response:** 201 Created
```json
{
  "id": "880e8400-e29b-41d4-a716-446655440000",
  "message": "Model provisioned successfully"
}
```

---

### 6. Model Details

#### GET /api/semantic-models/{id}
**Get complete model definition with all dimensions and measures**

- **Path Parameters:**
  - `id` (string, required): Model UUID

- **Response:** 200 OK
```json
{
  "model": {
    "id": "770e8400-e29b-41d4-a716-446655440000",
    "name": "customer_360_extended",
    "display_name": "Customer 360",
    "version": "2.1.1",
    "status": "active"
  },
  "dimensions": [
    {
      "id": "d1",
      "name": "customer_id",
      "type": "number",
      "display_name": "Customer ID",
      "sql": "${customers.id}",
      "primary_key": true
    },
    {
      "id": "d2",
      "name": "customer_name",
      "type": "string",
      "display_name": "Customer Name",
      "sql": "${customers.name}"
    }
  ],
  "measures": [
    {
      "id": "m1",
      "name": "total_orders",
      "type": "count",
      "display_name": "Total Orders",
      "sql": "COUNT(*)"
    },
    {
      "id": "m2",
      "name": "revenue",
      "type": "sum",
      "display_name": "Revenue",
      "sql": "SUM(${orders.amount})"
    }
  ]
}
```

---

### 7. Sync With Business Object

#### POST /api/semantic-models/{id}/sync
**Synchronize semantic model with business object schema**

- **Path Parameters:**
  - `id` (string, required): Model ID

- **Response:** 200 OK
```json
{
  "synced_fields": 12,
  "message": "Model synced successfully"
}
```

---

### 8. Add Custom Dimension

#### POST /api/semantic-models/{id}/dimensions
**Add a custom dimension to a semantic model**

- **Path Parameters:**
  - `id` (string, required): Model ID

- **Request Body:**
```json
{
  "name": "customer_segment",
  "type": "string",
  "display_name": "Customer Segment",
  "sql": "CASE WHEN ${customers.revenue} > 100000 THEN 'VIP' ELSE 'Standard' END"
}
```

- **Response:** 201 Created
```json
{
  "dimension_id": "d10",
  "message": "Dimension added successfully"
}
```

---

### 9. Override Dimension

#### PUT /api/semantic-models/dimensions/{dimId}
**Override a dimension definition in a custom model**

- **Path Parameters:**
  - `dimId` (string, required): Dimension ID

- **Request Body:**
```json
{
  "display_name": "Customer Segment (Updated)",
  "sql": "CASE WHEN ${customers.revenue} > 150000 THEN 'Premium' WHEN ${customers.revenue} > 100000 THEN 'VIP' ELSE 'Standard' END"
}
```

- **Response:** 200 OK
```json
{
  "message": "Dimension updated successfully"
}
```

---

## Cube.js Query Engine

### 10. Execute Query

#### POST /api/cube/query
**Execute a semantic query against Cube.js**

- **Headers:** `X-Tenant-ID` (required), `X-Tenant-Instance-ID` (required)

- **Request Body:**
```json
{
  "measures": ["customers.count", "orders.revenue"],
  "dimensions": ["customers.name", "orders.status"],
  "filters": [
    {
      "member": "orders.status",
      "operator": "equals",
      "values": ["completed"]
    }
  ],
  "timeDimensions": [
    {
      "dimension": "orders.created_at",
      "granularity": "month",
      "dateRange": ["2024-01-01", "2024-12-31"]
    }
  ],
  "order": {
    "orders.revenue": "desc"
  },
  "limit": 1000,
  "offset": 0,
  "timezone": "UTC"
}
```

- **Response:** 200 OK
```json
{
  "data": [
    {
      "customers.count": 1250,
      "orders.revenue": 125000.50,
      "customers.name": "John Doe",
      "orders.status": "completed",
      "orders.created_at": "2024-01-31"
    }
  ],
  "annotation": {
    "measures": ["customers.count", "orders.revenue"],
    "dimensions": ["customers.name", "orders.status", "orders.created_at"],
    "timeDimensions": ["orders.created_at"]
  },
  "query": {
    "measures": ["customers.count", "orders.revenue"],
    "dimensions": ["customers.name", "orders.status"],
    "timeDimensions": [
      {
        "dimension": "orders.created_at",
        "granularity": "month"
      }
    ]
  },
  "executionTime": 1250,
  "cacheHit": false,
  "preAggUsed": false
}
```

---

### 11. Generate SQL from Query

#### POST /api/cube/query/sql
**Generate SQL from semantic query without executing**

- **Headers:** `X-Tenant-ID` (required)

- **Request Body:** Same as /api/cube/query

- **Response:** 200 OK
```json
{
  "sql": "SELECT customers.id, SUM(orders.amount) as total FROM customers JOIN orders ON customers.id = orders.customer_id WHERE orders.status = 'completed' GROUP BY customers.id ORDER BY total DESC LIMIT 1000",
  "message": "SQL generated successfully"
}
```

---

### 12. Get Cube Metadata

#### GET /api/cube/meta
**Get metadata about available dimensions and measures for a cube**

- **Query Parameters:**
  - `cube_name` (string, optional): Specific cube to query

- **Response:** 200 OK
```json
{
  "cubes": [
    {
      "name": "customers",
      "title": "Customers",
      "dimensions": [
        {
          "name": "customers.id",
          "type": "string",
          "title": "Customer ID"
        },
        {
          "name": "customers.created_at",
          "type": "time",
          "title": "Created At"
        }
      ],
      "measures": [
        {
          "name": "customers.count",
          "type": "count",
          "title": "Count"
        }
      ]
    }
  ]
}
```

---

### 13. Get Pre-Aggregations

#### GET /api/cube/pre-aggregations
**Get pre-aggregation definitions and status**

- **Query Parameters:**
  - `cube_name` (string, optional): Filter by cube

- **Response:** 200 OK
```json
{
  "preAggregations": [
    {
      "id": "pa1",
      "name": "orders_by_status",
      "type": "rollup",
      "dimensions": ["orders.status"],
      "measures": ["orders.count", "orders.revenue"],
      "granularity": "month",
      "status": "active",
      "lastBuiltAt": "2024-01-20T15:30:00Z"
    }
  ]
}
```

---

### 14. Dry Run Query

#### POST /api/cube/dry-run
**Test a query without executing against the database**

- **Headers:** `X-Tenant-ID` (required), `X-Tenant-Instance-ID` (required)

- **Request Body:** Same as /api/cube/query

- **Response:** 200 OK
```json
{
  "valid": true,
  "cubes_used": ["customers", "orders"],
  "sql": "...",
  "estimated_rows": 5000,
  "message": "Query is valid"
}
```

---

### 15. Generate Cube Schema

#### POST /api/cube/generate
**Generate complete Cube.js schema**

- **Request Body:**
```json
{
  "datasource_id": "660e8400-e29b-41d4-a716-446655440000"
}
```

- **Response:** 200 OK
```json
{
  "schema": {
    "customers": {
      "sql_table": "public.customers",
      "dimensions": [...],
      "measures": [...]
    }
  },
  "message": "Schema generated successfully"
}
```

---

### 16. Generate Cube from Business Object

#### POST /api/cube/generate/{boID}
**Generate cube schema from business object definition**

- **Path Parameters:**
  - `boID` (string, required): Business Object UUID

- **Response:** 200 OK
```json
{
  "cube_name": "customer_business_object",
  "cube": {...},
  "message": "Cube generated from business object"
}
```

---

### 17. Preview Cube Schema

#### GET /api/cube/preview
**Preview cube schema without generation**

- **Query Parameters:**
  - `bo_id` (string, optional): Business object ID
  - `cube_name` (string, optional): Preview for specific cube

- **Response:** 200 OK
```json
{
  "preview": {
    "cube_name": "orders",
    "dimensions": 8,
    "measures": 5
  }
}
```

---

## Query Analytics & History

### 18. Query Execution History

#### GET /api/semantic/analytics/history
**Retrieve semantic query execution history**

- **Query Parameters:**
  - `limit` (integer, optional, default: 50): Number of results
  - `offset` (integer, optional, default: 0): Pagination offset
  - `cube_name` (string, optional): Filter by cube
  - `date_from` (string, optional): ISO 8601 date
  - `date_to` (string, optional): ISO 8601 date

- **Response:** 200 OK
```json
{
  "history": [
    {
      "id": "query-1",
      "query": {...},
      "executed_at": "2024-01-20T14:30:00Z",
      "execution_time_ms": 1250,
      "user_id": "user-1",
      "status": "success",
      "cache_hit": false
    }
  ],
  "total": 156
}
```

---

### 19. Performance Analytics

#### GET /api/semantic/analytics/performance
**Get performance metrics and optimization suggestions**

- **Query Parameters:**
  - `time_range` (string, optional, default: "24h"): Time window
  - `groupby` (string, optional): Group by dimension

- **Response:** 200 OK
```json
{
  "metrics": {
    "total_queries": 2450,
    "avg_execution_time_ms": 845,
    "cache_hit_rate": 0.35,
    "slow_queries": 12
  },
  "suggestions": [
    {
      "type": "add_pre_aggregation",
      "cube": "orders",
      "dimensions": ["status", "created_at"],
      "estimated_improvement": "45%"
    }
  ]
}
```

---

## Fabric Models (Extended Semantic Layer)

### 20. Get Fabric Models

#### GET /api/fabric/models
**List semantic models for a datasource**

- **Query Parameters:**
  - `datasource_id` (string, required): Datasource UUID

- **Response:** 200 OK
```json
{
  "models": [
    {
      "id": "model-1",
      "name": "orders",
      "key": "/orders",
      "title": "Orders",
      "type": "table",
      "columns": 12,
      "is_core": true
    }
  ],
  "count": 1
}
```

---

### 21. Get Model Definition

#### GET /api/fabric/models/definition
**Retrieve detailed model definition**

- **Query Parameters:**
  - `datasource_id` (string, required): Datasource UUID
  - `model_key` (string, required): Model key (e.g., "/orders")

- **Response:** 200 OK
```json
{
  "id": "model-1",
  "datasource_id": "660e8400-e29b-41d4-a716-446655440000",
  "model_key": "/orders",
  "title": "Orders",
  "description": "Customer order records",
  "resolved_config": {
    "cubes": [
      {
        "name": "orders",
        "sql": "SELECT * FROM public.orders",
        "dimensions": [...],
        "measures": [...]
      }
    ]
  }
}
```

---

### 22. Generate Models

#### POST /api/fabric/models/generate
**Generate semantic models from database schema**

- **Request Body:**
```json
{
  "datasource_id": "660e8400-e29b-41d4-a716-446655440000",
  "tables": ["customers", "orders", "products"],
  "include_joins": true,
  "auto_detect_relationships": true
}
```

- **Response:** 200 OK
```json
{
  "models_generated": 3,
  "models": [
    {
      "key": "/customers",
      "name": "customers",
      "status": "created"
    }
  ],
  "message": "Models generated successfully"
}
```

---

### 23. Generate Default Models

#### POST /api/fabric/models/generate-defaults
**Create default semantic models for a datasource**

- **Request Body:**
```json
{
  "datasource_id": "660e8400-e29b-41d4-a716-446655440000"
}
```

- **Response:** 200 OK
```json
{
  "models_created": 8,
  "message": "Default models generated",
  "models": [...]
}
```

---

### 24. Model Metadata Batch

#### POST /api/fabric/models/metadata
**Get metadata for multiple tables**

- **Request Body:**
```json
{
  "datasource_id": "660e8400-e29b-41d4-a716-446655440000",
  "table_names": ["customers", "orders", "products"]
}
```

- **Response:** 200 OK
```json
{
  "results": {
    "customers": {
      "columns": 15,
      "primary_key": "id",
      "row_count": 50000
    },
    "orders": {
      "columns": 12,
      "primary_key": "id",
      "row_count": 250000
    }
  }
}
```

---

### 25. Validate Model

#### POST /api/fabric/models/validate
**Validate a model without saving**

- **Request Body:**
```json
{
  "base_model_key": "/customers",
  "model_object": {
    "name": "customers_extended",
    "extends": "/customers",
    "dimensions": [
      {
        "name": "tier",
        "sql": "CASE WHEN revenue > 1000000 THEN 'enterprise' ELSE 'standard' END"
      }
    ]
  }
}
```

- **Response:** 200 OK
```json
{
  "issues": [
    {
      "type": "warning",
      "message": "SQL expression references missing column",
      "level": "warning"
    }
  ]
}
```

---

## Joins & Relationship Discovery

### 26. Extract Join Suggestions

#### GET /api/fabric/joins/{datasourceId}
**Extract join suggestions from database relationships**

- **Path Parameters:**
  - `datasourceId` (string, required): Datasource UUID

- **Response:** 200 OK
```json
{
  "joins": [
    {
      "id": "join-1",
      "left_table": "customers",
      "right_table": "orders",
      "left_key": "id",
      "right_key": "customer_id",
      "type": "one_to_many",
      "confidence": 0.95
    }
  ],
  "count": 1,
  "datasource_id": "660e8400-e29b-41d4-a716-446655440000"
}
```

---

### 27. Get Joins for Table

#### GET /api/fabric/joins/{datasourceId}/table/{tableName}
**Get join definitions for a specific table**

- **Path Parameters:**
  - `datasourceId` (string, required): Datasource UUID
  - `tableName` (string, required): Table name

- **Response:** 200 OK
```json
{
  "table_name": "orders",
  "joins": [
    {
      "join_path": "orders.customers",
      "left_table": "orders",
      "right_table": "customers",
      "join_condition": "orders.customer_id = customers.id"
    }
  ],
  "count": 1,
  "datasource_id": "660e8400-e29b-41d4-a716-446655440000"
}
```

---

### 28. Generate Cube from Table

#### POST /api/fabric/cubes/generate-from-table
**Generate complete cube schema from a database table**

- **Request Body:**
```json
{
  "datasource_id": "660e8400-e29b-41d4-a716-446655440000",
  "table_name": "orders"
}
```

- **Response:** 200 OK
```json
{
  "cube": {
    "name": "orders",
    "sql": "SELECT * FROM public.orders",
    "dimensions": [
      {
        "name": "id",
        "type": "number",
        "sql": "${TABLE}.id",
        "primary_key": true
      }
    ],
    "measures": [
      {
        "name": "count",
        "type": "count",
        "sql": "COUNT(${TABLE}.id)"
      }
    ]
  },
  "table_name": "orders",
  "datasource_id": "660e8400-e29b-41d4-a716-446655440000"
}
```

---

## Extension Models

### 29. List Extensions

#### GET /api/fabric/extensions
**List all extension models**

- **Query Parameters:**
  - `datasource_id` (string, required): Datasource UUID

- **Response:** 200 OK
```json
[
  {
    "id": "ext-1",
    "base_model_key": "/customers",
    "model_key": "/customers_extended",
    "title": "Extended Customers",
    "status": "draft"
  }
]
```

---

### 30. Save Extension

#### POST /api/fabric/extensions
**Save an extension model**

- **Query Parameters:**
  - `datasource_id` (string, required): Datasource UUID

- **Request Body:**
```json
{
  "base_model_key": "/customers",
  "model_key": "/customers_extended",
  "title": "Extended Customers",
  "description": "Customers model with additional calculations",
  "model_object": {
    "name": "customers_extended",
    "extends": "/customers",
    "dimensions": [...],
    "measures": [...]
  },
  "actor_id": "user-123"
}
```

- **Response:** 201 Created
```json
{
  "model": {
    "id": "ext-1",
    "model_key": "/customers_extended",
    "status": "saved"
  },
  "issues": []
}
```

---

### 31. Compatibility Report

#### GET /api/fabric/extensions/compatibility-report
**Get extension compatibility report**

- **Query Parameters:**
  - `datasource_id` (string, required): Datasource UUID

- **Response:** 200 OK
```json
{
  "compatible": true,
  "version": "1.0.0",
  "features": [
    "core_models",
    "custom_models",
    "semantic_layer",
    "data_catalog"
  ],
  "warnings": [],
  "datasource_id": "660e8400-e29b-41d4-a716-446655440000"
}
```

---

## Semantic Model Calculations

### 32. Add Calculation to Model

#### POST /api/fabric/models/{id}/calculations
**Add a calculation to a semantic model**

- **Path Parameters:**
  - `id` (string, required): Semantic Model UUID

- **Request Body:**
```json
{
  "calculation_id": "calc-123",
  "argument_mapping": {
    "revenue": "orders.amount",
    "count": "customers.count"
  },
  "output_name": "revenue_per_customer",
  "is_public": true
}
```

- **Response:** 201 Created
```json
{
  "id": "smc-1",
  "semantic_model_id": "model-1",
  "calculation_id": "calc-123",
  "output_name": "revenue_per_customer"
}
```

---

### 33. Get Model Calculations

#### GET /api/fabric/models/{id}/calculations
**Get all calculations for a semantic model**

- **Path Parameters:**
  - `id` (string, required): Semantic Model UUID

- **Response:** 200 OK
```json
[
  {
    "id": "smc-1",
    "calculation_id": "calc-123",
    "output_name": "revenue_per_customer",
    "is_public": true
  }
]
```

---

### 34. Remove Calculation

#### DELETE /api/fabric/models/{id}/calculations/{calc_id}
**Remove a calculation from a semantic model**

- **Path Parameters:**
  - `id` (string, required): Semantic Model UUID
  - `calc_id` (string, required): Calculation Association UUID

- **Response:** 204 No Content

---

## Catalog & Metadata

### 35. List Catalog Tables

#### GET /api/catalog/tables
**Get all tables from a datasource with column definitions**

- **Query Parameters:**
  - `datasource_id` (string, required): Datasource UUID

- **Response:** 200 OK
```json
{
  "tables": [
    {
      "id": "table-1",
      "type": "table",
      "data": {
        "label": "orders",
        "tableName": "public.orders",
        "schemaName": "public",
        "nodeType": "table",
        "isCore": false,
        "columns": [
          {
            "id": "col-1",
            "name": "id",
            "type": "number",
            "isCore": false,
            "nullable": false,
            "isPrimaryKey": true,
            "qualifiedPath": "public.orders.id"
          }
        ],
        "columnCount": 12
      }
    }
  ],
  "count": 1
}
```

---

### 36. Get Catalog Nodes

#### GET /api/catalog/nodes
**Query catalog nodes with filters**

- **Query Parameters:**
  - `tenant_id` (string, optional): Filter by tenant
  - `tenant_datasource_id` (string, optional): Filter by datasource
  - `type` (string, optional): Node type filter (e.g., "table", "view")
  - `q` (string, optional): Search query
  - `limit` (integer, optional, default: 50): Result limit

- **Response:** 200 OK
```json
[
  {
    "id": "node-1",
    "node_id": "node-1",
    "node_name": "customers",
    "qualified_path": "public.customers",
    "catalog_type": "table",
    "node_type": "table",
    "description": "Customer master data",
    "properties": {
      "row_count": 50000,
      "columns": 15
    }
  }
]
```

---

### 37. Refresh Charts

#### POST /api/catalog/{datasourceId}/refresh-charts
**Regenerate ERD and lineage charts**

- **Path Parameters:**
  - `datasourceId` (string, required): Datasource UUID

- **Response:** 200 OK
```json
{
  "success": true,
  "message": "Charts refreshed successfully",
  "datasource_id": "660e8400-e29b-41d4-a716-446655440000"
}
```

---

## Error Responses

### Common Error Codes

| Status | Error Code | Message | Meaning |
|--------|-----------|---------|---------|
| 400 | `MISSING_TENANT_CONTEXT` | X-Tenant-ID header is required | Tenant header missing |
| 400 | `INVALID_TENANT_ID` | Invalid tenant ID format | Tenant ID not UUID |
| 400 | `INVALID_DATASOURCE_ID` | Invalid datasource ID format | Datasource ID not UUID |
| 400 | `INVALID_REQUEST` | Invalid request body | Malformed JSON or missing fields |
| 401 | `UNAUTHORIZED` | Authentication required | Token missing or invalid |
| 404 | `NOT_FOUND` | Resource not found | Cube, model, or endpoint not found |
| 500 | `INTERNAL_ERROR` | Internal server error | Server-side error occurred |

### Error Response Format

```json
{
  "error": {
    "code": "MISSING_TENANT_CONTEXT",
    "message": "X-Tenant-ID header is required",
    "details": "Detailed error information"
  }
}
```

---

## Data Type Definitions

### Dimension
```go
type Dimension struct {
  ID             string      // Unique identifier
  CubeID         string      // Parent cube
  Name           string      // Dimension name
  DisplayName    string      // User-friendly name
  Type           string      // string|number|time|geo|boolean
  SQL            string      // SQL expression
  Format         string      // Optional format hint
  CaseSensitive  bool        // Case sensitivity flag
  PrimaryKey     bool        // Is primary key
  Shown          bool        // Should be shown in UI
  Metadata       interface{} // Custom metadata
}
```

### Measure
```go
type Measure struct {
  ID           string      // Unique identifier
  CubeID       string      // Parent cube
  Name         string      // Measure name
  DisplayName  string      // User-friendly name
  Type         string      // count|sum|avg|min|max|count_distinct
  SQL          string      // SQL expression
  Format       string      // currency|percent|number
  RollingWindow int        // Optional rolling window days
  DrillMembers []string    // Drill-down dimensions
  Filters      []string    // SQL filters
  Metadata     interface{} // Custom metadata
}
```

### Cube
```go
type Cube struct {
  ID                string                    // Unique identifier
  TenantID          string                    // Tenant owner
  Name              string                    // Cube name
  DisplayName       string                    // User-friendly name
  Description       string                    // Cube description
  SQL               string                    // Base SQL table/query
  Dimensions        []Dimension               // Array of dimensions
  Measures          []Measure                 // Array of measures
  PreAggregations   []PreAggregation          // Pre-aggregation definitions
  Joins             []Join                    // Join definitions
  Metadata          interface{}               // Custom metadata
  Status            string                    // active|draft|archived
  Version           string                    // Version number
  CreatedAt         time.Time                 // Creation timestamp
  UpdatedAt         time.Time                 // Last update timestamp
}
```

### Query
```go
type Query struct {
  Measures       []string         // Measures to include
  Dimensions     []string         // Dimensions to group by
  TimeDimensions []TimeDimension  // Time dimensions with granularity
  Filters        []Filter         // WHERE clause conditions
  Segments       []string         // Pre-defined segments
  Order          map[string]string // ORDER BY specification
  Limit          int              // Row limit
  Offset         int              // Pagination offset
  Timezone       string           // Timezone for time operations
}
```

### QueryResult
```go
type QueryResult struct {
  Data           []map[string]interface{} // Result rows
  Annotation     QueryAnnotation          // Metadata about result
  ExecutionTime  int                      // Milliseconds
  CacheHit       bool                     // Was result cached?
  PreAggUsed     bool                     // Pre-aggregation used?
}
```

---

## Rate Limiting

- No explicit rate limiting implemented
- Suggested limits: 1000 requests/hour per tenant
- Long-running queries timeout after 5 minutes

---

## Caching Strategy

- **Query Results**: Cached for 5 minutes by default
- **Metadata**: Cached with stale-while-revalidate
- **Pre-Aggregations**: Cached indefinitely until invalidated
- **Cache Headers**: ETag and If-None-Match supported

---

## Pagination

Most list endpoints support pagination via:
- `limit` (default: 50, max: 500)
- `offset` (default: 0)
- `page` (alternative: 1-indexed)
- `page_size` (alternative to limit)

---

## Versioning

Current API version: **v1**

Endpoint pattern: `/api/{resource}`

No explicit version in URL path. Version management via headers coming in v2.

---

## WebSocket Endpoints

### Real-time Query Updates

#### GET /api/ws
**WebSocket connection for real-time updates**

To connect:
```javascript
const ws = new WebSocket('ws://localhost:8080/api/ws');
```

Subscribe to query updates:
```json
{
  "type": "subscribe",
  "cube": "orders",
  "event": "query_complete"
}
```

---

## Development & Debugging

### Debug Endpoints (Dev Mode Only)

#### GET /api/debug/headers
**Echo request headers for debugging**

#### GET /_routes
**List all registered routes**

#### GET /_debug/amqp-metrics
**AMQP/Kafka metrics (if enabled)**

---

## References

- **Cube.js Documentation**: https://cube.dev/docs
- **Frontend Integration**: See `semlayer-frontend` repository
- **Schema Definitions**: See `pkg/semantic/types.go`
- **Handler Implementation**: See `internal/api/semantic_layer_handler.go`

