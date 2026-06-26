# API Endpoints Catalog: Quick Reference

## Overview

Self-documenting API system that catalogs all API endpoints with metadata, relationships, and discovery capabilities. Enables dynamic endpoint discovery and context-aware API browsing.

## Core Concepts

### Three-Tier Architecture
```
Catalog (API metadata store)
  ↓
Mappings (Relationships)
  ↓
Discovery (Context-aware lookup)
```

## Quick Start

### 1. Get All Validation Endpoints
```bash
curl -X GET \
  "http://localhost:8080/api-endpoints?category=validation&tenant_id=550e8400-e29b-41d4-a716-446655440000" \
  -H "X-Tenant-ID: 550e8400-e29b-41d4-a716-446655440000"
```

### 2. Get Endpoints for a Specific Entity
```bash
curl -X GET \
  "http://localhost:8080/entities/e29b-41d4-a716-446655440000/api-endpoints?tenant_id=550e8400-e29b-41d4-a716-446655440000" \
  -H "X-Tenant-ID: 550e8400-e29b-41d4-a716-446655440000"
```

### 3. Search Endpoints
```bash
curl -X GET \
  "http://localhost:8080/api-endpoints/search?tenant_id=550e8400-e29b-41d4-a716-446655440000&q=validation&method=POST" \
  -H "X-Tenant-ID: 550e8400-e29b-41d4-a716-446655440000"
```

### 4. Get OpenAPI Specification
```bash
curl -X GET \
  "http://localhost:8080/api-endpoints/openapi?tenant_id=550e8400-e29b-41d4-a716-446655440000" \
  -H "X-Tenant-ID: 550e8400-e29b-41d4-a716-446655440000"
```

## Validation Rules Endpoints (Pre-Seeded)

| Operation | Method | Path | Purpose |
|-----------|--------|------|---------|
| List Rules | GET | `/validation-rules` | Retrieve all rules with pagination |
| Create Rule | POST | `/validation-rules` | Create new rule |
| Get Rule | GET | `/validation-rules/{id}` | Retrieve specific rule |
| Update Rule | PATCH | `/validation-rules/{id}` | Update rule definition |
| Delete Rule | DELETE | `/validation-rules/{id}` | Delete rule |
| Execute | POST | `/validation-rules/{id}/execute` | Execute rule against data |
| Batch Execute | POST | `/validation-rules/execute-batch` | Execute rules on multiple records |
| Audit Trail | GET | `/validation-rules/{id}/audit` | View rule change history |

## API Response Examples

### List Endpoints Response
```json
{
  "data": [
    {
      "id": "12345678-1234-1234-1234-123456789012",
      "endpoint_name": "List Validation Rules",
      "description": "Retrieve all validation rules for the tenant",
      "http_method": "GET",
      "url_path": "/validation-rules",
      "category": "validation",
      "subcategory": "rules",
      "purpose": "read",
      "version": "1.0.0",
      "is_active": true,
      "requires_auth": true,
      "parameters": [
        {
          "name": "page",
          "in": "query",
          "required": false,
          "data_type": "integer",
          "description": "Page number for pagination"
        },
        {
          "name": "limit",
          "in": "query",
          "required": false,
          "data_type": "integer",
          "description": "Items per page (max 200)"
        }
      ],
      "response_schema": {
        "type": "object",
        "properties": {
          "data": {"type": "array"},
          "pagination": {"type": "object"}
        }
      },
      "tags": ["validation", "rules", "listing"],
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 50,
    "total": 8
  }
}
```

### Entity Endpoints Response
```json
{
  "entity_id": "e29b-41d4-a716-446655440000",
  "data": [
    {
      "id": "12345678-1234-1234-1234-123456789012",
      "endpoint_name": "List Validation Rules",
      "http_method": "GET",
      "url_path": "/validation-rules",
      "category": "validation",
      "purpose": "read",
      "relationship_type": "can_read",
      "version": "1.0.0"
    },
    {
      "id": "87654321-4321-4321-4321-210987654321",
      "endpoint_name": "Execute Single Validation Rule",
      "http_method": "POST",
      "url_path": "/validation-rules/{id}/execute",
      "category": "validation",
      "purpose": "execute",
      "relationship_type": "can_execute",
      "version": "1.0.0"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 50,
    "total": 5
  }
}
```

## Classification Reference

### Categories
- **validation** - Validation rule operations
- **entity** - Entity management operations
- **relationship** - Relationship management
- **catalog** - Catalog operations
- **audit** - Audit trail operations

### Purposes
- **read** - Retrieve/list operations
- **create** - Create operations
- **update** - Update operations
- **delete** - Delete operations
- **execute** - Execution/action operations
- **search** - Search operations
- **audit** - Audit/history operations

### Relationship Types
- **can_read** - Can read from entity/datasource
- **can_create** - Can create for entity/datasource
- **can_update** - Can update entity/datasource
- **can_delete** - Can delete from entity/datasource
- **can_execute** - Can execute against entity/datasource
- **can_validate** - Can validate entity/datasource
- **can_sync** - Can synchronize with datasource

## Common Patterns

### Pattern: Get All Operations for an Entity
```typescript
// 1. Get entity ID
const entityId = 'e29b-41d4-a716-446655440000';

// 2. Fetch all available endpoints
const response = await fetch(
  `/entities/${entityId}/api-endpoints?tenant_id=${tenantId}`,
  {
    headers: { 'X-Tenant-ID': tenantId }
  }
);

// 3. Group by category
const endpointsByCategory = {};
response.data.forEach(ep => {
  if (!endpointsByCategory[ep.category]) {
    endpointsByCategory[ep.category] = [];
  }
  endpointsByCategory[ep.category].push(ep);
});

// Result: All operations available for this entity
```

### Pattern: Generate API Documentation
```typescript
// 1. Get all endpoints
const endpoints = await fetch(
  `/api-endpoints?tenant_id=${tenantId}&limit=200`,
  { headers: { 'X-Tenant-ID': tenantId } }
);

// 2. Group by category
const doc = {};
endpoints.data.forEach(ep => {
  if (!doc[ep.category]) doc[ep.category] = [];
  doc[ep.category].push({
    name: ep.endpoint_name,
    method: ep.http_method,
    path: ep.url_path,
    description: ep.description,
    parameters: ep.parameters,
    examples: ep.examples
  });
});

// 3. Export as markdown/HTML
```

### Pattern: Context-Aware API Browser
```typescript
// 1. User selects entity
const selectedEntity = getSelectedEntity();

// 2. Load available operations
const operations = await fetch(
  `/entities/${selectedEntity.id}/api-endpoints?tenant_id=${tenantId}`,
  { headers: { 'X-Tenant-ID': tenantId } }
);

// 3. Build UI based on endpoints
const operationButtons = operations.data.map(op => ({
  label: op.endpoint_name,
  action: () => executeOperation(op),
  enabled: canExecute(op)
}));
```

## HTTP Status Codes

| Code | Meaning | Example |
|------|---------|---------|
| 200 | Success | GET endpoint details |
| 201 | Created | POST new endpoint |
| 204 | No content | DELETE endpoint |
| 400 | Bad request | Missing required parameter |
| 401 | Unauthorized | Missing auth token |
| 403 | Forbidden | Insufficient permissions |
| 404 | Not found | Endpoint doesn't exist |
| 409 | Conflict | Duplicate endpoint mapping |
| 500 | Server error | Database error |

## Error Response Format

```json
{
  "error": {
    "message": "Failed to fetch endpoints",
    "code": "db_error",
    "details": "connection refused",
    "timestamp": "2024-01-01T00:00:00Z"
  }
}
```

## Query Parameters

### Common Parameters
- `tenant_id` (required) - Tenant identifier
- `datasource_id` (optional) - Filter by datasource
- `page` (optional, default: 1) - Page number
- `limit` (optional, default: 50, max: 200) - Results per page

### Filter Parameters
- `category` - Filter by category
- `method` - Filter by HTTP method (GET, POST, PATCH, DELETE)
- `active_only` - Boolean to show only active endpoints
- `q` / `search` - Full-text search

## TypeScript Types

```typescript
interface APIEndpoint {
  id: string;
  tenant_id: string;
  datasource_id?: string;
  endpoint_name: string;
  description: string;
  http_method: string;
  url_path: string;
  category: string;
  subcategory?: string;
  purpose?: string;
  request_schema?: Record<string, any>;
  response_schema?: Record<string, any>;
  parameters?: EndpointParameter[];
  examples?: EndpointExample[];
  tags?: string[];
  requires_auth: boolean;
  is_active: boolean;
  version: string;
  created_at: string;
  updated_at: string;
}

interface EndpointParameter {
  name: string;
  in: 'path' | 'query' | 'header' | 'body';
  required: boolean;
  description: string;
  data_type: string;
  example?: any;
}

interface EndpointEntityMapping {
  id: string;
  api_endpoint_id: string;
  entity_id: string;
  relationship_type: string;
  created_at: string;
  updated_at: string;
}
```

## Troubleshooting

### Issue: Empty Results
```bash
# Check if endpoints are seeded
curl -X GET \
  "http://localhost:8080/api-endpoints?tenant_id=TENANT_ID&category=validation" \
  -H "X-Tenant-ID: TENANT_ID"

# If empty, trigger seeding in application startup
```

### Issue: 401 Unauthorized
```bash
# Ensure auth token is included
curl -X GET \
  "http://localhost:8080/api-endpoints?tenant_id=TENANT_ID" \
  -H "X-Tenant-ID: TENANT_ID" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### Issue: Slow Response
```bash
# Check pagination
curl -X GET \
  "http://localhost:8080/api-endpoints?tenant_id=TENANT_ID&limit=10&page=1" \
  -H "X-Tenant-ID: TENANT_ID"

# Verify database indexes
SELECT * FROM pg_indexes WHERE tablename LIKE 'api_%';
```

### Issue: Cross-Tenant Data Leakage
```bash
# Verify query filters
SELECT COUNT(*) FROM api_endpoints_catalog WHERE tenant_id = 'TENANT_1';
SELECT COUNT(*) FROM api_endpoints_catalog WHERE tenant_id = 'TENANT_2';
```

## Performance Benchmarks

| Operation | Target | Typical |
|-----------|--------|---------|
| List 50 endpoints | < 200ms | 80ms |
| Search endpoints | < 300ms | 120ms |
| Get entity endpoints | < 250ms | 100ms |
| Create endpoint | < 500ms | 200ms |
| Update endpoint | < 400ms | 150ms |
| Delete endpoint | < 300ms | 100ms |
| Batch get (10 endpoints) | < 300ms | 120ms |

## Rate Limiting

- **Default**: 1000 requests/minute per tenant
- **Burst**: Up to 100 requests/second
- **Override**: Contact DevOps for custom limits

## Pagination Guidelines

- **Default page size**: 50
- **Maximum page size**: 200
- **Recommended**: 50-100 for UI
- **Recommended**: 200 for bulk operations

## Best Practices

1. **Cache Results**
   - Cache endpoint list for 5-10 minutes
   - Invalidate on deployment
   - Use ETags if supported

2. **Batch Operations**
   - Use reverse lookup endpoints
   - Avoid N+1 queries
   - Load all endpoints once

3. **Error Handling**
   - Always handle 400/500 errors
   - Implement exponential backoff retry
   - Log errors for debugging

4. **Filtering**
   - Apply category filter when possible
   - Use active_only=true for UI
   - Implement search early

5. **Documentation**
   - Keep endpoint descriptions current
   - Maintain example requests/responses
   - Document parameters completely

## Support

For issues or questions:
- Check BACKEND_API_CATALOG_INTEGRATION.md for detailed docs
- Check FRONTEND_VALIDATION_RULES_INTEGRATION.md for UI integration
- Review API_CATALOG_DEPLOYMENT_CHECKLIST.md for deployment help
- Contact backend team for API issues
- Contact frontend team for UI issues
