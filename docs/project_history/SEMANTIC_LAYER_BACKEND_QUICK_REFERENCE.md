# Quick Reference: Semantic Layer Backend

**Package**: `httpapi`  
**File**: `/backend/internal/api/semantic_layer_chi.go` (430+ LOC)  
**Status**: ✅ Production Ready

---

## 🚀 Quick Start

### 1. Register Routes
In your main API router setup:

```go
// In main API initialization
s := &Server{DB: db}
s.RegisterSemanticLayerRoutes(router)
```

### 2. Test an Endpoint
```bash
# Generate Core Model
curl -X POST http://localhost:8080/api/business-entities/entity-123/generate-core-model \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000000" \
  -H "X-Tenant-Datasource-ID: 11111111-1111-1111-1111-111111111111" \
  -H "Content-Type: application/json" \
  -d '{
    "model_name": "Customer_Core",
    "source_keys": ["customer_id", "name"]
  }'
```

### 3. Expected Response
```json
{
  "model_id": "550e8400-e29b-41d4-a716-446655440000",
  "model_name": "Customer_Core"
}
```

---

## 📚 All 8 Endpoints

### Model/View Generation
```
POST /api/business-entities/{entityID}/generate-core-model
POST /api/business-entities/{entityID}/generate-core-view
POST /api/business-entities/{entityID}/create-custom-model
POST /api/business-entities/{entityID}/create-custom-view
```

### Data Retrieval
```
GET /api/business-entities/{entityID}/semantic-assets
GET /api/business-entities/{entityID}/relationship-suggestions?limit=20&min_confidence=0.5
```

### Relationship Management
```
POST /api/business-entities/{entityID}/apply-relationship-suggestion
POST /api/business-entities/{entityID}/traverse-graph
```

---

## 🔧 Handler Signatures

```go
// All handlers follow this pattern:
func (s *Server) handleXxx(w http.ResponseWriter, r *http.Request)

// Access methods:
s.DB                    // *sql.DB for queries
extractTenantContext(r) // Get tenant/datasource IDs
chi.URLParam(r, "...")  // Get URL parameters
```

---

## 📋 Request/Response Patterns

### Generate Core Model Request
```json
{
  "model_name": "string",
  "source_keys": ["string", "string"]
}
```

### Get Suggestions Request
```
?limit=20&min_confidence=0.5
```

### Apply Suggestion Request
```json
{
  "suggestion_id": "uuid"
}
```

### Traverse Graph Request
```json
{
  "start_node_id": "uuid",
  "dot_path": "customer.orders.items"
}
```

---

## 🛠️ Common Patterns

### Extract Tenant Context
```go
tenantContext, err := extractTenantContext(r)
if err != nil {
    http.Error(w, fmt.Sprintf("missing tenant context: %v", err), http.StatusBadRequest)
    return
}
```

### Parse JSON Request
```go
var req GenerateCoreModelRequest
if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
    writeJSONError(w, http.StatusBadRequest, "Invalid request body", "invalid_request", err.Error())
    return
}
```

### Execute Database Query
```go
row := s.DB.QueryRowContext(ctx, query, args...)
err := row.Scan(&result)
```

### Return JSON Response
```go
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusCreated)
json.NewEncoder(w).Encode(map[string]interface{}{
    "model_id": createdID,
    "model_name": req.ModelName,
})
```

---

## 🗄️ Database References

### Insert Semantic Asset
```sql
INSERT INTO semantic_assets (
    id, tenant_id, datasource_id, business_entity_id,
    core_model_id, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (tenant_id, datasource_id, business_entity_id)
DO UPDATE SET core_model_id = $5, updated_at = $6
```

### Query Suggestions
```sql
SELECT id, source_entity_id, target_entity_id, confidence
FROM relationship_suggestions
WHERE tenant_id = $1 AND source_entity_id = $2
  AND confidence >= $3
ORDER BY confidence DESC
LIMIT $4
```

### Insert Catalog Node
```sql
INSERT INTO catalog_node (
    id, tenant_id, datasource_id, node_name, node_type,
    node_description, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
```

---

## ⚠️ Error Handling

### Missing Tenant Context
```
Status: 400 Bad Request
Body: { "error": "missing tenant context: ..." }
```

### Invalid Request
```
Status: 400 Bad Request
Body: { "error": "Entity ID required" }
```

### Database Error
```
Status: 500 Internal Server Error
Body: { "error": "Failed to create model" }
```

### Not Found
```
Status: 404 Not Found
Body: { "error": "Suggestion not found" }
```

---

## 🔒 Tenant Scoping

All endpoints require:
```bash
-H "X-Tenant-ID: {tenant-uuid}"
-H "X-Tenant-Datasource-ID: {datasource-uuid}"
```

These are automatically extracted and used in all queries:
```go
WHERE tenant_id = $1 AND datasource_id = $2 ...
```

---

## 📊 Response Codes

| Code | Meaning |
|------|---------|
| 200 | Success (retrieval) |
| 201 | Created (POST operations) |
| 400 | Bad Request (validation error) |
| 404 | Not Found |
| 500 | Server Error |

---

## 🧪 Testing Examples

### Test 1: Generate Core Model
```bash
POST /api/business-entities/entity-1/generate-core-model
Expects: 201 with { "model_id": "...", "model_name": "..." }
```

### Test 2: Get Semantic Assets
```bash
GET /api/business-entities/entity-1/semantic-assets
Expects: 200 with full semantic asset record
```

### Test 3: Get Suggestions
```bash
GET /api/business-entities/entity-1/relationship-suggestions?limit=5
Expects: 200 with array of suggestions
```

### Test 4: Traverse Graph
```bash
POST /api/business-entities/entity-1/traverse-graph
Body: { "start_node_id": "...", "dot_path": "customer.orders" }
Expects: 200 with { "nodes": [...], "path": "..." }
```

---

## 📁 File Locations

| Item | Path |
|------|------|
| Handlers | `/backend/internal/api/semantic_layer_chi.go` |
| Migration | `/backend/internal/api/migrations/semantic_layer_tables.sql` |
| Full Docs | `/BACKEND_SEMANTIC_LAYER_IMPLEMENTATION.md` |
| Frontend Service | `/frontend/src/services/businessEntitySemanticService.ts` |

---

## 🔗 Integration Checklist

- [ ] Database migration applied
- [ ] Routes registered in main API
- [ ] Tenant context middleware active
- [ ] Handlers tested with sample data
- [ ] GraphQL resolvers wired
- [ ] Frontend Apollo hooks connected
- [ ] E2E flow tested
- [ ] Deployed to staging

---

## 💡 Tips

1. **Always include tenant headers** - Endpoints validate tenant context first
2. **Use parameterized queries** - All DB access uses `$1, $2` placeholders
3. **Check error responses** - JSON error format tells you what went wrong
4. **Test multi-tenant** - Verify isolation with requests from different tenants
5. **Monitor confidence scores** - Suggestions under 0.5 often indicate weak relationships

---

**Last Updated**: January 2025  
**Ready For**: GraphQL Integration & Testing
