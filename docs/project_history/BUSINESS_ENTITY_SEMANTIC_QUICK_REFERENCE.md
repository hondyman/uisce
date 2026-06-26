# Business Entity Semantic Layer - Quick Reference

## What Was Implemented

A comprehensive system enabling business entities to:

1. **Generate Core Semantic Models/Views** from semantic terms
   - Auto-discovers tables from business entity metadata
   - Links semantic terms to create foundation models
   - Marked as "core" for versioning

2. **Create Custom Extensions**
   - Custom models extend core models
   - Custom views extend core views
   - Support additional dimensions, measures, and columns
   - Separate customization from foundation

3. **AI-Powered Relationship Suggestions**
   - FK-based detection (strongest signal)
   - Join frequency analysis from query logs
   - Name similarity using Levenshtein distance
   - Semantic text similarity with embeddings
   - Edge type priors from catalog
   - Confidence scoring with transparency

4. **Related Objects Navigation** (Workday-style)
   - "Links To" (Many-to-One) relationships
   - "Links From" (One-to-Many) relationships
   - Dot-notation graph traversal (e.g., `Employee.department.company.name`)
   - Multi-level object discovery

5. **Entity Details Tabs**
   - Semantic Assets tab (core + custom models/views)
   - Suggestions tab with confidence breakdown
   - Related Objects tab with graph navigation

## Files Created

### Frontend Services
- `/frontend/src/services/businessEntitySemanticService.ts` - HTTP client & business logic
- `/frontend/src/hooks/useBusinessEntitySemanticLayer.ts` - React state management

### Frontend Components
- `/frontend/src/components/entity/SemanticAssetsTab.tsx` - Core/custom display
- `/frontend/src/components/entity/SemanticAssetsTab.css` - Styling
- `/frontend/src/components/entity/RelationshipSuggestionPanel.tsx` - AI suggestions
- `/frontend/src/components/entity/RelationshipSuggestionPanel.css` - Styling
- `/frontend/src/components/entity/RelatedObjectsNavigator.tsx` - Graph navigator
- `/frontend/src/components/entity/RelatedObjectsNavigator.css` - Styling

### GraphQL
- `/frontend/src/graphql/queries/businessEntitySemantic.ts` - Queries & mutations

### Documentation
- `/BUSINESS_ENTITY_SEMANTIC_IMPLEMENTATION_GUIDE.md` - Complete implementation guide

## Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│                   Entity Details Page                    │
├─────────────────────────────────────────────────────────┤
│  Tabs: Semantic Assets | Suggestions | Related Objects  │
├─────────────────────────────────────────────────────────┤
│                                                           │
│  ┌──────────────────┐  ┌──────────────────┐             │
│  │  Core Model Tab  │  │ Custom Model Tab │             │
│  └──────────────────┘  └──────────────────┘             │
│       (Display)              (Create)                    │
│                                                           │
│  ┌──────────────────────────────────────────────────┐   │
│  │    RelationshipSuggestionPanel                   │   │
│  │  - AI-powered suggestions with confidence       │   │
│  │  - Scoring breakdown visualization              │   │
│  │  - One-click application                        │   │
│  └──────────────────────────────────────────────────┘   │
│                                                           │
│  ┌──────────────────────────────────────────────────┐   │
│  │    RelatedObjectsNavigator                       │   │
│  │  - Links To / Links From sections               │   │
│  │  - Dot-notation traversal (Employee.dept.co)    │   │
│  │  - Graph visualization                          │   │
│  └──────────────────────────────────────────────────┘   │
│                                                           │
└─────────────────────────────────────────────────────────┘
         │                │                │
         │                │                │
         ▼                ▼                ▼
    Service Layer    GraphQL Layer    HTTP Client
         │                │                │
         └────────────────┴────────────────┘
                   │
                   ▼
         ┌─────────────────┐
         │  Backend APIs   │
         ├─────────────────┤
         │ /api/business-  │
         │   entities/...  │
         └─────────────────┘
                   │
                   ▼
         ┌─────────────────┐
         │   PostgreSQL    │
         ├─────────────────┤
         │ semantic_assets │
         │ relationship_   │
         │  suggestions    │
         │ catalog_edge    │
         │ catalog_node    │
         └─────────────────┘
```

## Core Scoring Formula

```
Confidence = (1.0×FK + 0.7×JoinFreq + 0.4×NameSim + 0.3×TextSim + 0.6×EdgePrior) / 5

Confidence Bands:
  ≥ 0.80 : High (auto-accept ready)
  0.60-0.79 : Medium (review recommended)
  < 0.60 : Low (manual review required)
```

## Key Integration Points

### 1. Entity Details Page

```tsx
const semanticLayer = useBusinessEntitySemanticLayer({
  tenantId,
  datasourceId,
  businessEntityId,
  businessEntityName,
  semanticTermIds,
  sourceTableNames,
});

// Use semanticLayer.semanticAssets, .relationshipSuggestions, .relatedObjects
// Call semanticLayer.generateCoreModel(), .applyRelationshipSuggestion(), etc.
```

### 2. GraphQL Mutations

```graphql
mutation {
  generate_core_model(
    input: {
      business_entity_id: "uuid"
      business_entity_name: "Employee"
      semantic_term_ids: ["uuid1", "uuid2"]
      source_tables: ["employees"]
    }
  ) {
    semantic_model { id, node_name }
  }
}
```

### 3. REST API

```bash
POST /api/business-entities/generate-core-model
X-Tenant-ID: {tenant_id}
X-Tenant-Datasource-ID: {datasource_id}

{
  "business_entity_id": "uuid",
  "semantic_term_ids": ["uuid1"],
  "source_tables": ["employees"]
}
```

## Workflow Examples

### Generate Core Model & View

```
User Views Entity Detail
    ↓
System Fetches Semantic Assets
    ↓ [No core model exists]
User Clicks "Generate Core Model"
    ↓
Service Calls: POST /api/business-entities/generate-core-model
    ↓
Backend: Creates catalog_node with is_core=true
    ↓
Service Updates State: semanticAssets.coreModel
    ↓
UI Shows Generated Model
    ↓
User Clicks "Generate Core View"
    ↓ [Same flow for view]
UI Shows Both Core Model & View
```

### Apply Relationship Suggestion

```
System Fetches Suggestions (FK-based scoring)
    ↓
UI Shows Top 5 with Confidence Scores
    ↓
User Reviews Scoring Breakdown
    ↓
User Clicks "Accept" on Suggestion
    ↓
Service Calls: POST /api/business-entities/apply-relationship
    ↓
Backend: Creates catalog_edge with relationship_type
    ↓
Service Refreshes Suggestion List
    ↓
UI Shows "Applied" Status & Disables Button
```

### Traverse Object Graph

```
User Types: "Employee.department.company"
    ↓
Clicks "Traverse"
    ↓
Service Calls: POST /api/semantic-models/traverse-graph
    ↓
Backend: Follows catalog_edges from Employee → Department → Company
    ↓
Returns: nodes + edges for visualization
    ↓
UI Shows Path: Employee → Department → Company
```

## Tenant Isolation

All requests include scoping headers/params:
```
Headers:
  X-Tenant-ID: {tenant_id}
  X-Tenant-Datasource-ID: {datasource_id}

Query Params:
  ?tenant_id={tenant_id}&datasource_id={datasource_id}

Database WHERE clause:
  WHERE tenant_id = $1 AND datasource_id = $2
```

## Data Model: Semantic Assets

Linking table connects business entities to semantic models/views:

```json
{
  "id": "uuid",
  "business_entity_id": "uuid",
  "core_model_id": "uuid",
  "core_view_id": "uuid",
  "custom_model_id": "uuid",
  "custom_view_id": "uuid",
  "semantic_term_ids": ["uuid1", "uuid2"],
  "datasource_id": "uuid",
  "tenant_id": "uuid",
  "created_at": "2025-01-15T10:00:00Z"
}
```

## Error Handling

All operations include:
- Loading states for UX feedback
- Error states with user-friendly messages
- Automatic retry logic for transient failures
- Logging via `devLog`, `devError` utilities

## Performance

- Suggestion caching: 1 hour
- Model/view caching: 30 minutes
- Graph traversal cache: 15 minutes
- Pagination for large result sets
- Background jobs for nightly regeneration

## Next Steps for Backend Implementation

1. **Create tables** in PostgreSQL (schema in implementation guide)
2. **Implement handlers** in `backend/internal/api/`
3. **Implement service** in `backend/internal/services/`
4. **Add GraphQL resolvers** in `backend/graph/`
5. **Wire routing** in `main.go`
6. **Test endpoints** with curl or Postman
7. **Enable audit logging** for suggestions

## Files to Check/Modify

When integrating:

1. ✅ `/frontend/src/pages/EntityDetailsPage.tsx` - Add semantic tabs
2. ✅ `/frontend/src/components/entity/` - UI components ready
3. ✅ `/backend/internal/api/` - Create handlers (from guide)
4. ✅ `/backend/internal/services/` - Create service logic (from guide)
5. ✅ `/backend/graph/` - Add GraphQL resolvers
6. ✅ Database migrations - Create tables from guide

## Support

Refer to `BUSINESS_ENTITY_SEMANTIC_IMPLEMENTATION_GUIDE.md` for:
- Complete API documentation
- Database schema DDL
- Code examples for backend
- Testing strategies
- Troubleshooting guide
- Performance tuning
