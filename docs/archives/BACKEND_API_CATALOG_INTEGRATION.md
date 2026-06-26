# Backend Integration: API Endpoints Catalog & Validation Rules

## Overview

This document describes the backend implementation of the API Endpoints Catalog system, which provides complete visibility into all validation rule endpoints and their relationships to entities and datasources. This enables self-documenting APIs and dynamic endpoint discovery.

## Architecture

### Three Core Components

1. **API Endpoints Catalog** (`api_endpoints_catalog.go`)
   - Central repository of all available API endpoints
   - Metadata storage (schemas, parameters, examples)
   - CRUD operations with full audit trail
   - Search and filtering capabilities

2. **Endpoint Mappings** (`api_endpoint_mapping_routes.go`)
   - Relationship management between endpoints and business entities
   - Relationship management between endpoints and datasources
   - Enables context-aware endpoint discovery

3. **Seeding System** (`api_endpoints_seeder.go`)
   - Pre-populates catalog with validation rule endpoints
   - Creates automatic entity mappings
   - Ensures consistent baseline across environments

## Database Schema

### api_endpoints_catalog
Primary table for API endpoint metadata.

```sql
CREATE TABLE api_endpoints_catalog (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  datasource_id UUID,
  
  -- Endpoint Definition
  endpoint_name VARCHAR(255) NOT NULL,
  description TEXT,
  http_method VARCHAR(10),
  url_path VARCHAR(500),
  
  -- Classification
  category VARCHAR(100),          -- validation, entity, relationship, etc.
  subcategory VARCHAR(100),       -- rules, execution, audit, etc.
  purpose VARCHAR(50),            -- create, read, update, delete, execute
  
  -- Documentation
  request_schema JSONB,
  response_schema JSONB,
  parameters JSONB,
  examples JSONB,
  tags TEXT[],
  
  -- Configuration
  requires_auth BOOLEAN DEFAULT true,
  is_active BOOLEAN DEFAULT true,
  version VARCHAR(50),
  
  -- Audit
  created_by UUID,
  created_at TIMESTAMP,
  updated_at TIMESTAMP
);
```

### api_endpoint_entity_mappings
Maps API endpoints to entity types they operate on.

```sql
CREATE TABLE api_endpoint_entity_mappings (
  id UUID PRIMARY KEY,
  api_endpoint_id UUID NOT NULL,
  entity_id UUID NOT NULL,
  tenant_id UUID NOT NULL,
  relationship_type VARCHAR(50),  -- can_read, can_create, can_execute, etc.
  created_at TIMESTAMP,
  updated_at TIMESTAMP,
  
  UNIQUE(api_endpoint_id, entity_id, tenant_id, relationship_type)
);
```

### api_endpoint_datasource_mappings
Maps API endpoints to datasources they interact with.

```sql
CREATE TABLE api_endpoint_datasource_mappings (
  id UUID PRIMARY KEY,
  api_endpoint_id UUID NOT NULL,
  datasource_id UUID NOT NULL,
  tenant_id UUID NOT NULL,
  relationship_type VARCHAR(50),  -- can_read, can_write, can_validate, etc.
  created_at TIMESTAMP,
  updated_at TIMESTAMP,
  
  UNIQUE(api_endpoint_id, datasource_id, tenant_id, relationship_type)
);
```

## API Endpoints

### Endpoint Catalog Management

#### List API Endpoints
```http
GET /api-endpoints?tenant_id=<TENANT_ID>&datasource_id=<DATASOURCE_ID>&category=validation&page=1&limit=50
```

Query Parameters:
- `tenant_id` (required): Tenant identifier
- `datasource_id` (optional): Filter by datasource
- `category` (optional): Filter by category (validation, entity, etc.)
- `method` (optional): Filter by HTTP method (GET, POST, PATCH, DELETE)
- `search` (optional): Full-text search in name/description/path
- `active_only` (optional): Boolean to filter only active endpoints
- `page` (optional): Pagination page number (default: 1)
- `limit` (optional): Items per page (default: 50, max: 200)

Response:
```json
{
  "data": [
    {
      "id": "uuid",
      "endpoint_name": "List Validation Rules",
      "description": "Retrieve all validation rules...",
      "http_method": "GET",
      "url_path": "/validation-rules",
      "category": "validation",
      "subcategory": "rules",
      "purpose": "read",
      "version": "1.0.0",
      "is_active": true,
      "parameters": [...],
      "request_schema": {...},
      "response_schema": {...},
      "tags": ["validation", "rules"],
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 50,
    "total": 100
  }
}
```

#### Create API Endpoint
```http
POST /api-endpoints?tenant_id=<TENANT_ID>
Content-Type: application/json

{
  "endpoint_name": "Create Validation Rule",
  "description": "Create a new validation rule",
  "http_method": "POST",
  "url_path": "/validation-rules",
  "category": "validation",
  "subcategory": "rules",
  "purpose": "create",
  "requires_auth": true,
  "is_active": true,
  "version": "1.0.0",
  "parameters": [
    {
      "name": "body",
      "in": "body",
      "required": true,
      "description": "Rule definition",
      "data_type": "object"
    }
  ],
  "request_schema": {...},
  "response_schema": {...},
  "tags": ["validation", "rules", "create"]
}
```

#### Get API Endpoint
```http
GET /api-endpoints/{id}?tenant_id=<TENANT_ID>
```

#### Update API Endpoint
```http
PATCH /api-endpoints/{id}?tenant_id=<TENANT_ID>
Content-Type: application/json

{
  "endpoint_name": "Updated Name",
  "is_active": false,
  ...
}
```

#### Delete API Endpoint
```http
DELETE /api-endpoints/{id}?tenant_id=<TENANT_ID>
```

#### List by Category
```http
GET /api-endpoints/category/{category}?tenant_id=<TENANT_ID>
```

#### Search Endpoints
```http
GET /api-endpoints/search?tenant_id=<TENANT_ID>&q=validation&category=validation&method=GET
```

#### Get OpenAPI Specification
```http
GET /api-endpoints/openapi?tenant_id=<TENANT_ID>
```

Response:
```json
{
  "openapi": "3.0.0",
  "info": {
    "title": "API Endpoints Catalog",
    "version": "1.0.0"
  },
  "paths": {
    "validation": {
      "count": 7,
      "summary": "Endpoints for validation"
    }
  }
}
```

#### Get Endpoint Documentation
```http
GET /api-endpoints/{id}/documentation?tenant_id=<TENANT_ID>
```

### Endpoint Mapping Management

#### List Entity Mappings
```http
GET /api-endpoints/{endpoint-id}/entity-mappings?tenant_id=<TENANT_ID>
```

Response:
```json
{
  "endpoint_id": "uuid",
  "data": [
    {
      "id": "uuid",
      "api_endpoint_id": "uuid",
      "entity_id": "uuid",
      "relationship_type": "can_read",
      "created_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

#### Create Entity Mapping
```http
POST /api-endpoints/{endpoint-id}/entity-mappings?tenant_id=<TENANT_ID>
Content-Type: application/json

{
  "entity_id": "uuid",
  "relationship_type": "can_read"
}
```

#### Delete Entity Mapping
```http
DELETE /api-endpoints/{endpoint-id}/entity-mappings/{entity-id}?tenant_id=<TENANT_ID>
```

#### List Datasource Mappings
```http
GET /api-endpoints/{endpoint-id}/datasource-mappings?tenant_id=<TENANT_ID>
```

#### Create Datasource Mapping
```http
POST /api-endpoints/{endpoint-id}/datasource-mappings?tenant_id=<TENANT_ID>
Content-Type: application/json

{
  "datasource_id": "uuid",
  "relationship_type": "can_read"
}
```

#### Delete Datasource Mapping
```http
DELETE /api-endpoints/{endpoint-id}/datasource-mappings/{datasource-id}?tenant_id=<TENANT_ID>
```

### Reverse Lookups

#### Get All Endpoints for Entity
```http
GET /entities/{entity-id}/api-endpoints?tenant_id=<TENANT_ID>&page=1&limit=50
```

Response:
```json
{
  "entity_id": "uuid",
  "data": [
    {
      "id": "uuid",
      "endpoint_name": "List Validation Rules",
      "http_method": "GET",
      "url_path": "/validation-rules",
      "category": "validation",
      "purpose": "read",
      "relationship_type": "can_read"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 50,
    "total": 5
  }
}
```

#### Get All Endpoints for Datasource
```http
GET /datasources/{datasource-id}/api-endpoints?tenant_id=<TENANT_ID>
```

## Seeding System

### Auto-Seeding Validation Endpoints

The system includes validation rule endpoints seeded automatically:

1. **List Validation Rules** - GET /validation-rules
2. **Create Validation Rule** - POST /validation-rules
3. **Get Validation Rule** - GET /validation-rules/{id}
4. **Update Validation Rule** - PATCH /validation-rules/{id}
5. **Delete Validation Rule** - DELETE /validation-rules/{id}
6. **Execute Single Rule** - POST /validation-rules/{id}/execute
7. **Execute Batch Rules** - POST /validation-rules/execute-batch
8. **Get Audit Trail** - GET /validation-rules/{id}/audit

### Trigger Seeding

```go
// In your initialization code
if err := SeedAPIEndpointsCatalog(db, tenantID); err != nil {
  log.Fatal("Failed to seed catalog:", err)
}

// Register mappings for entities
if err := RegisterValidationEndpointMappings(db, tenantID, entityID); err != nil {
  log.Fatal("Failed to register mappings:", err)
}
```

## Classification System

### Categories
- `validation` - Validation rule operations
- `entity` - Entity management
- `relationship` - Relationship management
- `catalog` - Catalog operations
- `audit` - Audit trail operations

### Subcategories
- `rules` - Rule CRUD operations
- `execution` - Rule execution and testing
- `audit` - Audit trail and history

### Purposes
- `create` - Create operations
- `read` - Read/list operations
- `update` - Update operations
- `delete` - Delete operations
- `execute` - Execution operations
- `search` - Search operations

### Relationship Types
- `can_read` - Endpoint can read this entity/datasource
- `can_create` - Endpoint can create for this entity/datasource
- `can_update` - Endpoint can update this entity/datasource
- `can_delete` - Endpoint can delete from this entity/datasource
- `can_execute` - Endpoint can execute against this entity/datasource
- `can_validate` - Endpoint can validate this entity/datasource
- `can_sync` - Endpoint can sync with this datasource

## Integration Points

### Registration in Main API Router

```go
// In your main api.go or server setup
import "backend/internal/api"

func setupRoutes(r chi.Router, db *sql.DB) {
  // Register existing routes
  api.RegisterValidationRulesRoutes(r, db)
  
  // Register new catalog routes
  api.RegisterAPIEndpointsCatalogRoutes(r, db)
  api.RegisterEndpointMappingRoutes(r, db)
  
  // Seed catalog on startup
  tenantID := "your-tenant-id"
  if err := api.SeedAPIEndpointsCatalog(db, tenantID); err != nil {
    log.Printf("Warning: Failed to seed catalog: %v", err)
  }
}
```

## Best Practices

### 1. Endpoint Documentation
Always provide:
- Clear endpoint name
- Comprehensive description
- Request schema with examples
- Response schema documentation
- Parameter specifications
- Example usage

### 2. Categorization
Use consistent categorization:
- Group related endpoints by category
- Use subcategories for logical grouping
- Tag endpoints for discovery

### 3. Version Management
- Maintain version numbers
- Track API evolution
- Support multiple versions if needed

### 4. Active Status
- Mark deprecated endpoints as `is_active = false`
- Don't delete endpoints, mark as inactive
- Enables audit trail of API changes

### 5. Relationship Management
- Create mappings when endpoints interact with entities
- Update mappings when relationships change
- Use consistent relationship type naming

## Performance Optimization

### Indexes
All critical queries are indexed:
- `idx_api_endpoints_tenant_id` - Tenant filtering
- `idx_api_endpoints_category` - Category filtering
- `idx_api_endpoints_http_method` - Method filtering
- `idx_api_endpoints_is_active` - Active status
- Mapping indexes for join operations

### Query Optimization
- Use pagination for large result sets
- Filter early in WHERE clauses
- Leverage full-text search capabilities
- LIMIT results by default

## Security

### Tenant Isolation
- All queries are tenant-scoped
- `tenant_id` is required for all operations
- No cross-tenant data leakage

### Authentication
- All endpoints require authentication
- Use X-Tenant-ID headers for validation
- User context should be verified

### Authorization
- Implement role-based access control
- Restrict catalog modifications to admins
- Allow read access to developers

## Monitoring & Maintenance

### Health Checks
Monitor endpoint catalog health:
- Total active endpoints count
- Endpoints without documentation
- Mappings consistency

### Cleanup
Regular maintenance tasks:
- Archive inactive endpoints
- Remove broken mappings
- Update stale documentation

## Future Enhancements

1. **Endpoint Usage Tracking**
   - Log endpoint invocations
   - Track error rates
   - Monitor performance metrics

2. **API Versioning**
   - Support multiple API versions
   - Track version deprecation
   - Manage version migration

3. **Dynamic Documentation Generation**
   - Auto-generate OpenAPI specs
   - Build API browsers
   - Create client SDKs

4. **Dependency Management**
   - Track endpoint dependencies
   - Identify impact of changes
   - Manage breaking changes

5. **Endpoint Analytics**
   - Most used endpoints
   - Error patterns
   - Performance metrics
