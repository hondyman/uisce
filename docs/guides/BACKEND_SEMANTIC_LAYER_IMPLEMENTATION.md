# Backend Implementation: Business Entity Semantic Layer

**Status**: ✅ Phase 1 Complete - API Handlers & Database Schema Ready  
**Date**: January 2025  
**Component**: Semantic Layer Backend Integration

---

## 1. Overview

This document captures the backend implementation of the Business Entity Semantic Layer feature, fully integrated with the existing Fabric Builder stack and connected to the React frontend built in previous sessions.

### Architecture

```
┌─────────────────────────────────────────────────────────┐
│ Frontend (React + Apollo)                               │
│ ├─ SemanticAssetsTab, RelationshipSuggestionPanel      │
│ ├─ RelatedObjectsNavigator, EntityDetailsPage          │
│ └─ businessEntitySemanticService (HTTP client)         │
└──────────────────┬──────────────────────────────────────┘
                   │ HTTP Requests with Tenant Headers
                   ▼
┌─────────────────────────────────────────────────────────┐
│ Backend (Go + Chi Router) - semantic_layer_chi.go       │
│ ├─ 8 REST Endpoints (all tenant-scoped)                │
│ ├─ Database Operations (semantic_assets table)         │
│ └─ Relationship Suggestion Logic                        │
└──────────────────┬──────────────────────────────────────┘
                   │ SQL Queries
                   ▼
┌─────────────────────────────────────────────────────────┐
│ PostgreSQL Database                                     │
│ ├─ semantic_assets (core/custom models & views)        │
│ ├─ relationship_suggestions (AI suggestions)           │
│ ├─ relationship_suggestion_audit (audit trail)         │
│ └─ catalog_node/catalog_edge (existing tables)         │
└─────────────────────────────────────────────────────────┘
```

---

## 2. Implementation Status

### ✅ Completed

#### Database Schema (semantic_layer_tables.sql)
- **3 Tables**: semantic_assets, relationship_suggestions, relationship_suggestion_audit
- **8 Performance Indexes**: For common queries
- **Constraints**: Foreign keys to catalog_node, tenants, datasources
- **File**: `/backend/internal/api/migrations/semantic_layer_tables.sql`

#### API Handlers (semantic_layer_chi.go)
- **8 REST Endpoints**: Fully implemented with Chi routing
- **Request/Response Types**: Proper Go structs with JSON tags
- **Tenant Isolation**: All endpoints enforce tenant scoping
- **Error Handling**: Structured JSON error responses
- **File**: `/backend/internal/api/semantic_layer_chi.go` (430+ LOC)

#### Routes Registration
- All endpoints mounted under `/api/business-entities/...`
- Can be registered via `RegisterSemanticLayerRoutes(router)` in main API setup

### ⏳ Ready for Next Phase

- GraphQL Resolver wiring (mutations & queries)
- Integration testing (E2E validation)
- Performance optimization & caching
- Deployment to staging/production

---

## 3. API Endpoints

### Core Model Generation
```
POST /api/business-entities/{entityID}/generate-core-model
```

**Request Body**:
```json
{
  "model_name": "Customer_CoreModel",
  "source_keys": ["customer_id", "customer_name"]
}
```

**Response** (201 Created):
```json
{
  "model_id": "550e8400-e29b-41d4-a716-446655440000",
  "model_name": "Customer_CoreModel"
}
```

**Description**: Creates a catalog node of type "model" linked to the entity's semantic assets. The model automatically gets a "core" designation.

---

### Core View Generation
```
POST /api/business-entities/{entityID}/generate-core-view
```

**Request Body**:
```json
{
  "view_name": "Customer_CoreView",
  "selected_columns": ["customer_id", "email", "created_date"]
}
```

**Response** (201 Created):
```json
{
  "view_id": "660e8400-e29b-41d4-a716-446655440000",
  "view_name": "Customer_CoreView"
}
```

**Description**: Creates a catalog node of type "view" for selected columns. Linked in semantic_assets table.

---

### Custom Model Creation
```
POST /api/business-entities/{entityID}/create-custom-model
```

**Request Body**:
```json
{
  "model_name": "CustomerAnalytics",
  "expression": "SELECT * FROM customers WHERE status = 'active'",
  "source_keys": ["customer_id", "total_purchases"]
}
```

**Response** (201 Created):
```json
{
  "model_id": "770e8400-e29b-41d4-a716-446655440000",
  "model_name": "CustomerAnalytics",
  "expression": "SELECT * FROM customers WHERE status = 'active'"
}
```

**Description**: Creates a custom model node with user-defined SQL expression. Expression stored in node_description.

---

### Custom View Creation
```
POST /api/business-entities/{entityID}/create-custom-view
```

**Request Body**:
```json
{
  "view_name": "HighValueCustomers",
  "expression": "SELECT customer_id, email, total_purchases FROM customers WHERE total_purchases > 10000",
  "source_keys": ["customer_id", "total_purchases"]
}
```

**Response** (201 Created):
```json
{
  "view_id": "880e8400-e29b-41d4-a716-446655440000",
  "view_name": "HighValueCustomers",
  "expression": "SELECT customer_id, email, total_purchases FROM customers WHERE total_purchases > 10000"
}
```

**Description**: Creates a custom view node with user-defined SQL expression.

---

### Get Semantic Assets
```
GET /api/business-entities/{entityID}/semantic-assets
```

**Response** (200 OK):
```json
{
  "id": "990e8400-e29b-41d4-a716-446655440000",
  "tenant_id": "00000000-0000-0000-0000-000000000000",
  "datasource_id": "11111111-1111-1111-1111-111111111111",
  "business_entity_id": "aa000000-0000-0000-0000-000000000000",
  "core_model_id": "550e8400-e29b-41d4-a716-446655440000",
  "core_view_id": "660e8400-e29b-41d4-a716-446655440000",
  "custom_model_id": "770e8400-e29b-41d4-a716-446655440000",
  "custom_view_id": "880e8400-e29b-41d4-a716-446655440000",
  "source_tables": ["customers", "orders"],
  "created_at": "2025-01-15T10:30:00Z",
  "updated_at": "2025-01-15T10:30:00Z"
}
```

**Query Parameters**: None required
**Description**: Retrieves or creates semantic assets record for an entity. Auto-creates if doesn't exist.

---

### Get Relationship Suggestions
```
GET /api/business-entities/{entityID}/relationship-suggestions?limit=20&min_confidence=0.5
```

**Query Parameters**:
- `limit` (int, default 20): Maximum number of suggestions
- `min_confidence` (float, default 0.5): Minimum confidence score (0.0-1.0)

**Response** (200 OK):
```json
{
  "suggestions": [
    {
      "id": "bb111111-1111-1111-1111-111111111111",
      "tenant_id": "00000000-0000-0000-0000-000000000000",
      "datasource_id": "11111111-1111-1111-1111-111111111111",
      "source_entity_id": "aa000000-0000-0000-0000-000000000000",
      "target_entity_id": "cc222222-2222-2222-2222-222222222222",
      "confidence": 0.85,
      "rationale": "Foreign key found in schema",
      "scoring_breakdown": {
        "foreign_key_presence": 1.0,
        "join_frequency": 0.5,
        "name_similarity": 0.6,
        "text_similarity": 0.4,
        "edge_type_prior": 0.6
      },
      "accepted": false,
      "accepted_at": null,
      "created_at": "2025-01-15T10:30:00Z",
      "updated_at": "2025-01-15T10:30:00Z"
    }
  ],
  "count": 1
}
```

**Description**: Returns AI-generated relationship suggestions with confidence scores and scoring breakdown.

---

### Apply Relationship Suggestion
```
POST /api/business-entities/{entityID}/apply-relationship-suggestion
```

**Request Body**:
```json
{
  "suggestion_id": "bb111111-1111-1111-1111-111111111111"
}
```

**Response** (200 OK):
```json
{
  "success": true,
  "message": "Relationship suggestion applied"
}
```

**Description**: Converts a suggestion into an actual catalog edge. Marks suggestion as accepted.

---

### Traverse Object Graph
```
POST /api/business-entities/{entityID}/traverse-graph
```

**Request Body**:
```json
{
  "start_node_id": "aa000000-0000-0000-0000-000000000000",
  "dot_path": "customer.orders.items"
}
```

**Response** (200 OK):
```json
{
  "nodes": [
    "aa000000-0000-0000-0000-000000000000",
    "dd333333-3333-3333-3333-333333333333",
    "ee444444-4444-4444-4444-444444444444"
  ],
  "path": "customer.orders.items"
}
```

**Description**: Traverses a dot-notation path through the semantic graph. Each segment is matched against node names.

---

## 4. Database Schema

### semantic_assets Table
```sql
CREATE TABLE semantic_assets (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    datasource_id UUID NOT NULL,
    business_entity_id UUID NOT NULL,
    core_model_id UUID REFERENCES catalog_node(id),
    core_view_id UUID REFERENCES catalog_node(id),
    custom_model_id UUID REFERENCES catalog_node(id),
    custom_view_id UUID REFERENCES catalog_node(id),
    semantic_term_ids UUID[],
    source_tables TEXT[],
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    UNIQUE(tenant_id, datasource_id, business_entity_id)
);
```

**Purpose**: Central registry mapping business entities to their semantic models and views.

**Indexes**:
- `semantic_assets_tenant_entity_idx` - For queries by tenant/entity
- `semantic_assets_models_idx` - For finding entities with specific models

---

### relationship_suggestions Table
```sql
CREATE TABLE relationship_suggestions (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    datasource_id UUID NOT NULL,
    source_entity_id UUID NOT NULL,
    target_entity_id UUID NOT NULL,
    confidence DECIMAL(5,4) CHECK (confidence >= 0 AND confidence <= 1),
    rationale TEXT,
    scoring_breakdown JSONB,
    accepted BOOLEAN DEFAULT FALSE,
    accepted_at TIMESTAMP,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    UNIQUE(tenant_id, datasource_id, source_entity_id, target_entity_id)
);
```

**Purpose**: Stores AI-generated relationship recommendations with confidence scores.

**Indexes**:
- `relationship_suggestions_source_idx` - For queries by source entity
- `relationship_suggestions_confidence_idx` - For filtering by confidence threshold

---

### relationship_suggestion_audit Table
```sql
CREATE TABLE relationship_suggestion_audit (
    id UUID PRIMARY KEY,
    suggestion_id UUID REFERENCES relationship_suggestions(id),
    tenant_id UUID NOT NULL,
    action VARCHAR(50),
    created_at TIMESTAMP
);
```

**Purpose**: Audit trail for suggestion acceptance/rejection actions.

**Indexes**:
- `relationship_suggestion_audit_suggestion_idx` - For querying audit history

---

## 5. Integration Points

### 1. Router Setup
In the main API server setup file, register the semantic layer routes:

```go
// In main API router initialization
semanticLayerHandlers := httpapi.NewSemanticLayerHandlers()
apiRouter.Route("/api/business-entities", func(r chi.Router) {
    r.Post("/{entityID}/generate-core-model", s.handleGenerateCoreModel)
    r.Post("/{entityID}/generate-core-view", s.handleGenerateCoreView)
    // ... other endpoints
})
```

### 2. Tenant Context Extraction
All handlers use existing `extractTenantContext(r)` helper:

```go
tenantContext, err := extractTenantContext(r)
if err != nil {
    http.Error(w, fmt.Sprintf("missing tenant context: %v", err), http.StatusBadRequest)
    return
}
```

### 3. Database Access
All handlers use `s.DB` (*sql.DB) for database operations:

```go
row := s.DB.QueryRowContext(ctx, query, args...)
rows, err := s.DB.QueryContext(ctx, query, args...)
_, err := s.DB.ExecContext(ctx, query, args...)
```

### 4. GraphQL Integration (Next Phase)
Handlers will be connected to GraphQL resolvers:

```go
// In graphql/resolvers/semantic_layer.go
func (r *mutationResolver) GenerateCoreModel(ctx context.Context, entityID string, ...) {
    // Call backend handler or service
}
```

---

## 6. Request/Response Patterns

### Tenant Scoping
Every request includes tenant headers (handled by frontend shim):

```bash
curl -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000000" \
     -H "X-Tenant-Datasource-ID: 11111111-1111-1111-1111-111111111111" \
     "http://localhost:8080/api/business-entities/{entityID}/semantic-assets"
```

### Error Responses
Structured JSON error format:

```json
{
  "error": "Entity ID required"
}
```

### Success Responses
Consistent JSON structure with relevant data:

```json
{
  "model_id": "uuid-string",
  "model_name": "string",
  "additional_fields": "..."
}
```

---

## 7. Data Flow Example

### End-to-End Flow: Generate Core Model

1. **User Action** (Frontend)
   - User clicks "Generate Core Model" in SemanticAssetsTab
   - Component collects: entity ID, model name, source columns

2. **HTTP Request** (Frontend Service)
   ```javascript
   POST /api/business-entities/entity-123/generate-core-model
   Headers: X-Tenant-ID, X-Tenant-Datasource-ID
   Body: { model_name: "Customer_Core", source_keys: [...] }
   ```

3. **Backend Processing** (Handler)
   - Extract tenant context from headers
   - Validate request (model name required)
   - Create catalog_node with type="model"
   - Link to semantic_assets record
   - Return new model ID

4. **Database Operations**
   ```sql
   INSERT INTO catalog_node (...) VALUES (...)
   INSERT/UPDATE semantic_assets SET core_model_id = ... WHERE ...
   ```

5. **Response** (200 Created)
   ```json
   { "model_id": "new-uuid", "model_name": "Customer_Core" }
   ```

6. **Frontend Update**
   - Hook updates state with new model ID
   - UI reflects: ✅ Core Model: Customer_Core
   - User can now perform additional operations

---

## 8. Performance Considerations

### Indexes for Common Queries
- `semantic_assets_tenant_entity_idx` - Fast lookups by tenant/entity
- `relationship_suggestions_source_idx` - Fast suggestion retrieval
- `relationship_suggestions_confidence_idx` - Fast filtering by score

### Query Optimization
- All queries include tenant/datasource filters
- Relationships table has unique constraint preventing duplicates
- Scoring breakdown stored as JSONB for flexible schema

### Caching Opportunities (Future)
- Cache semantic assets per tenant/entity
- Cache relationship suggestions with TTL
- Cache catalog node relationships

---

## 9. Deployment Checklist

- [ ] Apply database migration: `semantic_layer_tables.sql`
- [ ] Register semantic layer routes in main API
- [ ] Verify tenant context middleware is active
- [ ] Test with sample tenant/entity IDs
- [ ] Wire up GraphQL resolvers
- [ ] Load test with production data volumes
- [ ] Verify logging/monitoring hooks
- [ ] Document API in Swagger/OpenAPI

---

## 10. Testing Strategy

### Unit Tests
- Handler input validation
- Error responses for missing tenant context
- Database operation mocking

### Integration Tests
- End-to-end flow: generate model → retrieve assets → apply suggestion
- Multi-tenant isolation (requests from different tenants)
- Concurrent requests handling

### E2E Tests
- Frontend → API → Database → Frontend updates
- User workflows: model generation, suggestion application
- Performance under load

---

## 11. Files Summary

| File | Purpose | Status | Lines |
|------|---------|--------|-------|
| `semantic_layer_chi.go` | All 8 REST handlers | ✅ Ready | 430+ |
| `semantic_layer_tables.sql` | Database schema | ✅ Ready | 70+ |
| Frontend service | HTTP client (previous session) | ✅ Complete | 220 |
| Frontend components | UI display (previous session) | ✅ Complete | 950 |

---

## 12. Next Steps

### Phase 2: GraphQL Integration
- [ ] Create GraphQL resolvers in `backend/internal/graphql/resolvers/`
- [ ] Wire 8 handlers to GraphQL mutations/queries
- [ ] Test with Apollo client on frontend

### Phase 3: Testing & Optimization
- [ ] Write integration tests
- [ ] Performance benchmarks
- [ ] Add caching layer

### Phase 4: Deployment
- [ ] Staging environment testing
- [ ] Production deployment
- [ ] Monitoring & alerts

---

## Technical Stack

**Backend**:
- Go 1.21+
- Chi router (HTTP routing)
- PostgreSQL (database)
- pq driver (PostgreSQL adapter)

**Frontend** (already complete):
- React 18+
- TypeScript
- Apollo Client (GraphQL)
- Material-UI (components)

**Integration**:
- REST API over HTTP
- JSON request/response
- Tenant headers (X-Tenant-ID, X-Tenant-Datasource-ID)
- UUID-based IDs

---

**Last Updated**: January 2025  
**Backend Ready For**: GraphQL Integration & Testing
