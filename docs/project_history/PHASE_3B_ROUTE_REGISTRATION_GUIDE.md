# Phase 3b.5: Route Registration Quick Start

## Location
File: `/backend/internal/api/api.go` around line 4384

## Routes to Add

Add these 4 routes to the router setup (likely in a function that registers chi routes):

```go
// Relationship Discovery endpoints
r.Post("/api/relationships/discover", srv.postDiscoverRelationships)
r.Post("/api/relationships/apply", srv.postApplyRelationship)

// Semantic Model Regeneration endpoints
r.Post("/api/models/regenerate", srv.postTriggerModelRegeneration)
r.Get("/api/models/version", srv.getModelVersion)
```

## Where to Find the Right Place

Search in `api.go` for:
- `RegisterRoutes`
- `setupRoutes`
- Line numbers around 4384 where other `.Post` routes are registered

You should see patterns like:
```go
r.Post("/path", srv.handlerName)
r.Get("/path", srv.handlerName)
```

## Tenant Context

All 4 handlers automatically extract tenant context from:
1. **Headers**: `X-Tenant-ID` and `X-Tenant-Datasource-ID`
2. **Query Params**: `tenant_id` and `datasource_id`

The tenant context is validated in each handler - no need for middleware.

## Testing the Routes

After registering, test with:

```bash
curl -X POST http://localhost:8080/api/relationships/discover \
  -H "X-Tenant-ID: your-tenant-id" \
  -H "X-Tenant-Datasource-ID: your-datasource-id" \
  -H "Content-Type: application/json" \
  -d '{
    "entity_attribute_id": "your-entity-id",
    "include_multi_hop": true,
    "max_hop_depth": 3
  }'
```

## File Locations

- **Handlers**: `/backend/internal/api/relationship_api_handlers.go`
- **API Main**: `/backend/internal/api/api.go`
- **Discovery Service**: `/backend/internal/api/enhanced_relationship_discovery.go`
- **Regeneration Service**: `/backend/internal/api/semantic_model_regeneration.go`
- **Database Schema**: `/backend/internal/migrations/006_relationship_discovery_schema.sql`
- **Regeneration Schema**: `/backend/internal/migrations/007_semantic_model_regeneration_dba.sql`

## Estimated Time
5-10 minutes to add routes + test

## Next
Once routes are registered and tested, move to Phase 4: Frontend components
