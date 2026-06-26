# Business Object Foreign Key Semantic Discovery - API Specification

## Base URL
```
http://localhost:8080/api/business-objects
```

## Authentication
All endpoints require:
- Header: `X-Tenant-ID: {tenantId}` (required)

## Endpoints

---

## 1. Discover Foreign Keys for Business Object

### Endpoint
```
GET /business-objects/{boId}/foreign-keys
```

### Description
Discovers all foreign key relationships involving the business object's driving table. Returns both outbound FKs (where the BO table references other tables) and inbound FKs (where other tables reference the BO table).

### Path Parameters
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `boId` | UUID string | Yes | The business object ID |

### Query Parameters
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| None | | | |

### Headers
```
X-Tenant-ID: {tenantId}           # Required
```

### Response Schema

#### Success Response (200 OK)
```json
{
  "business_object_id": "uuid",
  "foreign_keys": [
    {
      "edge_id": "uuid",
      "related_table_id": "uuid",
      "related_table_name": "string",
      "cardinality": "1:1|N:1|1:N|M:N",
      "direction": "outbound|inbound",
      "foreign_key_fields": [
        {
          "source_column": "string",
          "target_column": "string"
        }
      ],
      "properties": {
        "edge_type_name": "foreign_key",
        "cardinality": "string",
        "columns": [
          {
            "source_column": "string",
            "target_column": "string"
          }
        ],
        "source_table": "string",
        "target_table": "string",
        "on_delete": "CASCADE|SET NULL|NO ACTION",
        "on_update": "CASCADE|SET NULL|NO ACTION",
        "primary_constraint_name": "string"
      }
    }
  ],
  "count": 2,
  "message": "Found 2 foreign key relationships"
}
```

#### Error Response (400/500)
```json
{
  "success": false,
  "message": "Error description",
  "error": "error_code",
  "details": "Additional context"
}
```

### Error Codes
| Code | Status | Meaning |
|------|--------|---------|
| `INVALID_TENANT_ID` | 400 | X-Tenant-ID header missing or invalid |
| `INVALID_BO_ID` | 400 | boId parameter is not a valid UUID |
| `BO_NOT_FOUND` | 404 | Business object with boId not found |
| `NO_DRIVING_TABLE` | 400 | Business object has no driving_table_id set |
| `QUERY_ERROR` | 500 | Database query failed |

### Example Request
```bash
curl -X GET \
  -H "X-Tenant-ID: tenant-abc123" \
  http://localhost:8080/api/business-objects/550e8400-e29b-41d4-a716-446655440000/foreign-keys
```

### Example Response
```json
{
  "business_object_id": "550e8400-e29b-41d4-a716-446655440000",
  "foreign_keys": [
    {
      "edge_id": "fk-edge-001",
      "related_table_id": "550e8400-e29b-41d4-a716-446655440001",
      "related_table_name": "customers",
      "cardinality": "N:1",
      "direction": "outbound",
      "foreign_key_fields": [
        {
          "source_column": "customer_id",
          "target_column": "id"
        }
      ],
      "properties": {
        "edge_type_name": "foreign_key",
        "cardinality": "N:1",
        "columns": [
          {
            "source_column": "customer_id",
            "target_column": "id"
          }
        ],
        "source_table": "orders",
        "target_table": "customers",
        "on_delete": "CASCADE",
        "on_update": "CASCADE",
        "primary_constraint_name": "fk_orders_customer_id"
      }
    }
  ],
  "count": 1,
  "message": "Found 1 foreign key relationship"
}
```

---

## 2. Discover Related Semantic Terms

### Endpoint
```
GET /business-objects/{boId}/related-semantic-terms
```

### Description
Discovers semantic terms available from tables related to the BO's driving table via foreign keys. These are semantic terms that have been mapped in the catalog for columns in the related tables.

### Path Parameters
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `boId` | UUID string | Yes | The business object ID |

### Query Parameters
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `limit` | integer | 100 | Maximum number of results to return |
| `offset` | integer | 0 | Number of results to skip (for pagination) |
| `min_confidence` | float | 0.0 | Filter results by minimum confidence score (0.0-1.0) |

### Headers
```
X-Tenant-ID: {tenantId}           # Required
```

### Response Schema

#### Success Response (200 OK)
```json
{
  "business_object_id": "uuid",
  "discovered_count": 5,
  "related_semantic_terms": [
    {
      "semantic_term_id": "uuid",
      "semantic_term_name": "string",
      "related_table_name": "string",
      "related_field_name": "string",
      "related_field_id": "uuid",
      "source_fk_edge_id": "uuid",
      "join_path": "string",
      "confidence": 0.95,
      "match_reason": "semantic_term_mapped_in_catalog|inferred_from_name|fk_constraint_match"
    }
  ],
  "count": 5,
  "limit": 100,
  "offset": 0,
  "message": "Found 5 related semantic terms from 2 related tables"
}
```

#### Error Response (400/500)
```json
{
  "success": false,
  "message": "Error description",
  "error": "error_code",
  "details": "Additional context"
}
```

### Error Codes
| Code | Status | Meaning |
|------|--------|---------|
| `INVALID_TENANT_ID` | 400 | X-Tenant-ID header missing or invalid |
| `INVALID_BO_ID` | 400 | boId parameter is not a valid UUID |
| `INVALID_LIMIT` | 400 | limit parameter must be 1-1000 |
| `BO_NOT_FOUND` | 404 | Business object with boId not found |
| `NO_FOREIGN_KEYS` | 404 | BO's driving table has no foreign keys |
| `NO_SEMANTIC_TERMS` | 404 | No semantic terms found on related tables |
| `QUERY_ERROR` | 500 | Database query failed |

### Example Request
```bash
curl -X GET \
  -H "X-Tenant-ID: tenant-abc123" \
  "http://localhost:8080/api/business-objects/550e8400-e29b-41d4-a716-446655440000/related-semantic-terms?limit=50&min_confidence=0.9"
```

### Example Response
```json
{
  "business_object_id": "550e8400-e29b-41d4-a716-446655440000",
  "discovered_count": 3,
  "related_semantic_terms": [
    {
      "semantic_term_id": "st-customer-001",
      "semantic_term_name": "Customer Name",
      "related_table_name": "customers",
      "related_field_name": "name",
      "related_field_id": "550e8400-e29b-41d4-a716-446655440002",
      "source_fk_edge_id": "fk-edge-001",
      "join_path": "customer_id -> customers.id",
      "confidence": 0.95,
      "match_reason": "semantic_term_mapped_in_catalog"
    },
    {
      "semantic_term_id": "st-customer-002",
      "semantic_term_name": "Customer Email",
      "related_table_name": "customers",
      "related_field_name": "email",
      "related_field_id": "550e8400-e29b-41d4-a716-446655440003",
      "source_fk_edge_id": "fk-edge-001",
      "join_path": "customer_id -> customers.id",
      "confidence": 0.88,
      "match_reason": "semantic_term_mapped_in_catalog"
    }
  ],
  "count": 2,
  "limit": 50,
  "offset": 0,
  "message": "Found 2 related semantic terms from 1 related table"
}
```

---

## 3. Link Semantic Term to Business Object

### Endpoint
```
POST /business-objects/{boId}/link-semantic-term
```

### Description
Links a semantic term from a related table to a business object field. This establishes the connection that enables automatic join path generation for queries.

### Path Parameters
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `boId` | UUID string | Yes | The business object ID |

### Request Body
```json
{
  "semantic_term_id": "uuid",
  "related_table_id": "uuid",
  "foreign_key_edge_id": "uuid",
  "role": "string"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `semantic_term_id` | UUID string | Yes | The semantic term to link (from discovery endpoint) |
| `related_table_id` | UUID string | Yes | The related table containing the semantic term |
| `foreign_key_edge_id` | UUID string | Yes | The FK edge connecting BO table to related table |
| `role` | string | Yes | Semantic role/alias for this link (e.g., "customer", "primary_contact") |

### Headers
```
X-Tenant-ID: {tenantId}           # Required
Content-Type: application/json
```

### Response Schema

#### Success Response (201 Created)
```json
{
  "success": true,
  "message": "Semantic term linked successfully",
  "business_object_id": "uuid",
  "semantic_term_id": "uuid",
  "foreign_key_edge_id": "uuid",
  "bo_field_id": "uuid",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

#### Error Response (400/500)
```json
{
  "success": false,
  "message": "Error description",
  "error": "error_code",
  "details": "Additional context"
}
```

### Error Codes
| Code | Status | Meaning |
|------|--------|---------|
| `INVALID_TENANT_ID` | 400 | X-Tenant-ID header missing or invalid |
| `INVALID_BO_ID` | 400 | boId parameter is not a valid UUID |
| `INVALID_PAYLOAD` | 400 | Request body is missing required fields |
| `INVALID_TERM_ID` | 400 | semantic_term_id is not a valid UUID |
| `INVALID_EDGE_ID` | 400 | foreign_key_edge_id is not a valid UUID |
| `BO_NOT_FOUND` | 404 | Business object with boId not found |
| `SEMANTIC_TERM_NOT_FOUND` | 404 | Semantic term not found in catalog |
| `FK_EDGE_NOT_FOUND` | 404 | Foreign key edge not found or doesn't relate to BO |
| `TERM_ALREADY_LINKED` | 409 | This semantic term is already linked to this BO |
| `TERM_NOT_IN_RELATED_TABLE` | 400 | Semantic term exists but not in the specified related_table_id |
| `INVALID_FK_RELATIONSHIP` | 400 | FK edge doesn't connect BO table to related_table_id |
| `QUERY_ERROR` | 500 | Database operation failed |

### Example Request
```bash
curl -X POST \
  -H "X-Tenant-ID: tenant-abc123" \
  -H "Content-Type: application/json" \
  -d '{
    "semantic_term_id": "st-customer-001",
    "related_table_id": "550e8400-e29b-41d4-a716-446655440001",
    "foreign_key_edge_id": "fk-edge-001",
    "role": "customer"
  }' \
  http://localhost:8080/api/business-objects/550e8400-e29b-41d4-a716-446655440000/link-semantic-term
```

### Example Response
```json
{
  "success": true,
  "message": "Semantic term linked successfully",
  "business_object_id": "550e8400-e29b-41d4-a716-446655440000",
  "semantic_term_id": "st-customer-001",
  "foreign_key_edge_id": "fk-edge-001",
  "bo_field_id": "550e8400-e29b-41d4-a716-446655440010",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

---

## 4. Get Semantic Join Paths for Business Object

### Endpoint
```
GET /business-objects/{boId}/semantic-join-paths
```

### Description
Returns all currently linked semantic terms for a business object along with their join path metadata. This information is used by the query execution layer to construct proper JOIN clauses.

### Path Parameters
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `boId` | UUID string | Yes | The business object ID |

### Query Parameters
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| None | | | |

### Headers
```
X-Tenant-ID: {tenantId}           # Required
```

### Response Schema

#### Success Response (200 OK)
```json
{
  "business_object_id": "uuid",
  "semantic_join_paths": {
    "{bo_field_key}": {
      "bo_field_id": "uuid",
      "semantic_term_id": "uuid",
      "fk_edge_id": "uuid",
      "related_table": "string",
      "related_table_id": "uuid",
      "fk_properties": {
        "columns": [
          {
            "source_column": "string",
            "target_column": "string"
          }
        ],
        "cardinality": "1:1|N:1|1:N|M:N",
        "on_delete": "CASCADE|SET NULL|NO ACTION",
        "on_update": "CASCADE|SET NULL|NO ACTION"
      },
      "join_sql_template": "string"
    }
  },
  "count": 2,
  "message": "Found 2 semantic join paths"
}
```

#### Error Response (400/500)
```json
{
  "success": false,
  "message": "Error description",
  "error": "error_code",
  "details": "Additional context"
}
```

### Error Codes
| Code | Status | Meaning |
|------|--------|---------|
| `INVALID_TENANT_ID` | 400 | X-Tenant-ID header missing or invalid |
| `INVALID_BO_ID` | 400 | boId parameter is not a valid UUID |
| `BO_NOT_FOUND` | 404 | Business object with boId not found |
| `NO_SEMANTIC_LINKS` | 404 | BO has no linked semantic terms |
| `QUERY_ERROR` | 500 | Database query failed |

### Example Request
```bash
curl -X GET \
  -H "X-Tenant-ID: tenant-abc123" \
  http://localhost:8080/api/business-objects/550e8400-e29b-41d4-a716-446655440000/semantic-join-paths
```

### Example Response
```json
{
  "business_object_id": "550e8400-e29b-41d4-a716-446655440000",
  "semantic_join_paths": {
    "customer_name": {
      "bo_field_id": "550e8400-e29b-41d4-a716-446655440010",
      "semantic_term_id": "st-customer-001",
      "fk_edge_id": "fk-edge-001",
      "related_table": "customers",
      "related_table_id": "550e8400-e29b-41d4-a716-446655440001",
      "fk_properties": {
        "columns": [
          {
            "source_column": "customer_id",
            "target_column": "id"
          }
        ],
        "cardinality": "N:1",
        "on_delete": "CASCADE",
        "on_update": "CASCADE"
      },
      "join_sql_template": "LEFT JOIN customers c ON orders.customer_id = c.id"
    },
    "product_info": {
      "bo_field_id": "550e8400-e29b-41d4-a716-446655440011",
      "semantic_term_id": "st-product-001",
      "fk_edge_id": "fk-edge-002",
      "related_table": "products",
      "related_table_id": "550e8400-e29b-41d4-a716-446655440002",
      "fk_properties": {
        "columns": [
          {
            "source_column": "product_id",
            "target_column": "id"
          }
        ],
        "cardinality": "N:1",
        "on_delete": "SET NULL",
        "on_update": "CASCADE"
      },
      "join_sql_template": "LEFT JOIN products p ON orders.product_id = p.id"
    }
  },
  "count": 2,
  "message": "Found 2 semantic join paths"
}
```

---

## Common Workflows

### Workflow 1: Enrich Business Object with Related Semantic Terms

```bash
# Step 1: Discover what foreign keys exist
curl -H "X-Tenant-ID: tenant-1" \
  http://localhost:8080/api/business-objects/{boId}/foreign-keys

# Step 2: See what semantic terms are available
curl -H "X-Tenant-ID: tenant-1" \
  http://localhost:8080/api/business-objects/{boId}/related-semantic-terms

# Step 3: Link the semantic terms you want
curl -X POST \
  -H "X-Tenant-ID: tenant-1" \
  -H "Content-Type: application/json" \
  -d '{
    "semantic_term_id": "...",
    "related_table_id": "...",
    "foreign_key_edge_id": "...",
    "role": "customer"
  }' \
  http://localhost:8080/api/business-objects/{boId}/link-semantic-term

# Step 4: Verify the join paths
curl -H "X-Tenant-ID: tenant-1" \
  http://localhost:8080/api/business-objects/{boId}/semantic-join-paths
```

### Workflow 2: Build Query with Join Paths

```bash
# Retrieve join paths
curl -H "X-Tenant-ID: tenant-1" \
  http://localhost:8080/api/business-objects/{boId}/semantic-join-paths

# Use the join_sql_template in each path to construct query:
SELECT o.*, c.name, p.name
FROM orders o
LEFT JOIN customers c ON o.customer_id = c.id
LEFT JOIN products p ON o.product_id = p.id
WHERE o.id = ?
```

---

## Status Codes Reference

| Code | Meaning |
|------|---------|
| 200 | OK - Request succeeded |
| 201 | Created - Resource successfully created |
| 204 | No Content - Request succeeded but no content to return |
| 400 | Bad Request - Invalid parameters or request body |
| 401 | Unauthorized - Authentication failed or missing |
| 404 | Not Found - Resource not found |
| 409 | Conflict - Resource already exists or violates constraint |
| 422 | Unprocessable Entity - Semantic validation failed |
| 500 | Internal Server Error - Database or server error |
| 503 | Service Unavailable - Service temporarily down |

---

## Rate Limiting

Not currently implemented but recommended limits:
- Discovery endpoints: 1000 requests/hour per tenant
- Link endpoint: 100 requests/hour per tenant

---

## Pagination

Supported on `/related-semantic-terms` endpoint:
```
GET /business-objects/{boId}/related-semantic-terms?limit=50&offset=100
```

Returns:
- `count` - Number of results in this response
- `limit` - Requested limit
- `offset` - Requested offset
- `discovered_count` - Total available results

---

## Tenant Isolation

All endpoints enforce tenant isolation through:
1. Required `X-Tenant-ID` header
2. All queries filtered by `tenant_id`
3. BO ownership validated against tenant
4. No cross-tenant data exposure

---

## Version

**API Version:** 1.0  
**Last Updated:** 2024-01-15  
**Status:** Beta
