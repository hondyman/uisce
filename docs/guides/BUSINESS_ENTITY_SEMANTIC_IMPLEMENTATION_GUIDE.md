# Business Entity Semantic Layer Implementation Guide

## Overview

This guide details the implementation of a comprehensive system that enables business entities to automatically generate and manage semantic models and views, with support for custom extensions and AI-powered relationship suggestions.

## Architecture

### Core Components

1. **Frontend Service** (`businessEntitySemanticService.ts`)
   - HTTP client for API communication
   - Handles all CRUD operations for semantic assets
   - Manages relationship suggestions and graph traversal

2. **React Hooks** (`useBusinessEntitySemanticLayer.ts`)
   - State management for semantic assets
   - Automatic fetching and synchronization
   - Loading and error states for all operations

3. **UI Components**
   - `SemanticAssetsTab.tsx` - Core/custom model and view display
   - `RelationshipSuggestionPanel.tsx` - AI suggestions with scoring
   - `RelatedObjectsNavigator.tsx` - Object graph navigation

4. **GraphQL Integration** (`businessEntitySemantic.ts`)
   - Queries for fetching semantic assets
   - Mutations for creating/updating models and views
   - Graph traversal and relationship management

## Database Schema Extensions

### Required Tables

```sql
-- Semantic assets linking table
CREATE TABLE semantic_assets (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  datasource_id UUID NOT NULL,
  business_entity_id UUID NOT NULL,
  core_model_id UUID,
  core_view_id UUID,
  custom_model_id UUID,
  custom_view_id UUID,
  semantic_term_ids UUID[] DEFAULT '{}',
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  UNIQUE(tenant_id, datasource_id, business_entity_id),
  FOREIGN KEY (tenant_id) REFERENCES tenants(id),
  FOREIGN KEY (core_model_id) REFERENCES catalog_node(id),
  FOREIGN KEY (core_view_id) REFERENCES catalog_node(id),
  FOREIGN KEY (custom_model_id) REFERENCES catalog_node(id),
  FOREIGN KEY (custom_view_id) REFERENCES catalog_node(id)
);

-- Relationship suggestions table
CREATE TABLE relationship_suggestions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  datasource_id UUID NOT NULL,
  source_entity_id UUID NOT NULL,
  target_entity_id UUID NOT NULL,
  confidence FLOAT NOT NULL CHECK (confidence BETWEEN 0 AND 1),
  rationale TEXT,
  scoring_breakdown JSONB,
  accepted BOOLEAN DEFAULT FALSE,
  accepted_at TIMESTAMP WITH TIME ZONE,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  UNIQUE(tenant_id, datasource_id, source_entity_id, target_entity_id),
  FOREIGN KEY (tenant_id) REFERENCES tenants(id),
  FOREIGN KEY (source_entity_id) REFERENCES catalog_node(id),
  FOREIGN KEY (target_entity_id) REFERENCES catalog_node(id)
);

-- Audit trail for suggestions
CREATE TABLE relationship_suggestion_audit (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  suggestion_id UUID NOT NULL REFERENCES relationship_suggestions(id) ON DELETE CASCADE,
  action VARCHAR(50) NOT NULL, -- 'created', 'viewed', 'accepted', 'dismissed'
  user_id UUID,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  FOREIGN KEY (suggestion_id) REFERENCES relationship_suggestions(id)
);

-- Indexes
CREATE INDEX idx_semantic_assets_business_entity ON semantic_assets(business_entity_id);
CREATE INDEX idx_semantic_assets_tenant_datasource ON semantic_assets(tenant_id, datasource_id);
CREATE INDEX idx_relationship_suggestions_source ON relationship_suggestions(source_entity_id);
CREATE INDEX idx_relationship_suggestions_target ON relationship_suggestions(target_entity_id);
CREATE INDEX idx_relationship_suggestions_confidence ON relationship_suggestions(confidence DESC);
CREATE INDEX idx_relationship_suggestions_accepted ON relationship_suggestions(accepted);
```

## Backend API Endpoints

### 1. Generate Core Model

**Endpoint:** `POST /api/business-entities/generate-core-model`

**Headers:**
```
X-Tenant-ID: {tenant_id}
X-Tenant-Datasource-ID: {datasource_id}
Content-Type: application/json
```

**Request Body:**
```json
{
  "business_entity_id": "uuid",
  "business_entity_name": "Employee",
  "semantic_term_ids": ["uuid1", "uuid2"],
  "source_tables": ["employees", "departments"],
  "datasource_id": "uuid",
  "tenant_id": "uuid"
}
```

**Response:**
```json
{
  "success": true,
  "semantic_model": {
    "id": "uuid",
    "node_name": "Employee_Core",
    "description": "Core model for Employee entity",
    "properties": {
      "is_core": true,
      "business_entity_id": "uuid",
      "semantic_term_ids": ["uuid1", "uuid2"],
      "source_tables": ["employees", "departments"]
    },
    "created_at": "2025-01-15T10:00:00Z"
  }
}
```

### 2. Generate Core View

**Endpoint:** `POST /api/business-entities/generate-core-view`

**Request Body:**
```json
{
  "business_entity_id": "uuid",
  "business_entity_name": "Employee",
  "core_model_id": "uuid",
  "semantic_term_ids": ["uuid1", "uuid2"],
  "datasource_id": "uuid",
  "tenant_id": "uuid"
}
```

**Response:**
```json
{
  "success": true,
  "semantic_view": {
    "id": "uuid",
    "node_name": "Employee_View_Core",
    "description": "Core view for Employee entity",
    "properties": {
      "is_core": true,
      "business_entity_id": "uuid",
      "model_id": "uuid"
    }
  }
}
```

### 3. Create Custom Model

**Endpoint:** `POST /api/business-entities/create-custom-model`

**Request Body:**
```json
{
  "business_entity_id": "uuid",
  "core_model_id": "uuid",
  "custom_model_name": "Employee_Custom_Advanced",
  "additional_dimensions": [
    {
      "name": "seniority_level",
      "sql": "CASE WHEN years_of_service > 10 THEN 'senior' ELSE 'junior' END"
    }
  ],
  "additional_measures": [
    {
      "name": "avg_salary",
      "type": "avg",
      "sql": "salary"
    }
  ],
  "datasource_id": "uuid",
  "tenant_id": "uuid"
}
```

### 4. Create Custom View

**Endpoint:** `POST /api/business-entities/create-custom-view`

**Request Body:**
```json
{
  "business_entity_id": "uuid",
  "core_view_id": "uuid",
  "custom_view_name": "Employee_View_Custom",
  "custom_model_id": "uuid",
  "additional_columns": [
    {
      "name": "extended_salary",
      "source": "avg_salary"
    }
  ],
  "datasource_id": "uuid",
  "tenant_id": "uuid"
}
```

### 5. Get Semantic Assets

**Endpoint:** `GET /api/business-entities/{business_entity_id}/semantic-assets?datasource_id={datasource_id}`

**Response:**
```json
{
  "assets": {
    "coreModel": {...},
    "coreView": {...},
    "customModel": {...},
    "customView": {...}
  }
}
```

### 6. Get Relationship Suggestions

**Endpoint:** `POST /api/business-entities/relationship-suggestions`

**Request Body:**
```json
{
  "business_entity_id": "uuid",
  "source_tables": ["employees", "departments"],
  "datasource_id": "uuid",
  "tenant_id": "uuid",
  "limit": 5
}
```

**Response:**
```json
{
  "suggestions": [
    {
      "id": "uuid",
      "source_entity_id": "uuid",
      "target_entity_id": "uuid",
      "confidence": 0.92,
      "rationale": "Foreign key: employees.department_id → departments.id",
      "scoring_breakdown": {
        "fk_presence": 1.0,
        "join_frequency": 0.85,
        "name_similarity": 0.78,
        "text_similarity": 0.72,
        "edge_type_prior": 0.95
      }
    }
  ]
}
```

### 7. Apply Relationship Suggestion

**Endpoint:** `POST /api/business-entities/apply-relationship`

**Request Body:**
```json
{
  "source_entity_id": "uuid",
  "target_entity_id": "uuid",
  "confidence": 0.92,
  "rationale": "...",
  "scoring_breakdown": {...},
  "datasource_id": "uuid",
  "tenant_id": "uuid"
}
```

### 8. Traverse Object Graph

**Endpoint:** `POST /api/semantic-models/traverse-graph`

**Request Body:**
```json
{
  "start_model_id": "uuid",
  "dot_path": "Employee.department.company.address",
  "datasource_id": "uuid",
  "tenant_id": "uuid"
}
```

**Response:**
```json
{
  "success": true,
  "graph": {
    "nodes": [
      {
        "id": "uuid",
        "node_name": "Employee",
        "nodeType": "semantic_model"
      },
      {
        "id": "uuid",
        "node_name": "Department",
        "nodeType": "semantic_model"
      }
    ],
    "edges": [
      {
        "id": "uuid",
        "source": "uuid",
        "target": "uuid",
        "relationship_type": "references"
      }
    ]
  },
  "path_traversed": ["Employee", "Department", "Company", "Address"]
}
```

## Implementation Steps

### Step 1: Database Setup

1. Create the required tables in your PostgreSQL database
2. Add indexes for performance
3. Create audit trail tables

```bash
psql $DATABASE_URL < semantic-layer-schema.sql
```

### Step 2: Backend Implementation

Create handler functions in your Go backend (`backend/internal/api/business_entity_handlers.go`):

```go
package api

import (
  "github.com/gin-gonic/gin"
  "github.com/eganpj/semlayer/backend/internal/services"
)

type BusinessEntitySemanticHandler struct {
  service *services.BusinessEntitySemanticService
}

func (h *BusinessEntitySemanticHandler) GenerateCoreModel(c *gin.Context) {
  // Implementation
}

func (h *BusinessEntitySemanticHandler) GenerateCoreView(c *gin.Context) {
  // Implementation
}

func (h *BusinessEntitySemanticHandler) CreateCustomModel(c *gin.Context) {
  // Implementation
}

func (h *BusinessEntitySemanticHandler) CreateCustomView(c *gin.Context) {
  // Implementation
}

func (h *BusinessEntitySemanticHandler) GetRelationshipSuggestions(c *gin.Context) {
  // Implementation
}

func (h *BusinessEntitySemanticHandler) ApplyRelationshipSuggestion(c *gin.Context) {
  // Implementation
}

func (h *BusinessEntitySemanticHandler) TraverseObjectGraph(c *gin.Context) {
  // Implementation
}
```

### Step 3: Semantic Layer Service

Create business logic in `backend/internal/services/business_entity_semantic_service.go`:

```go
package services

type BusinessEntitySemanticService struct {
  db *sql.DB
  catalogService *CatalogService
}

// Helper: Calculate relationship suggestion score
func (s *BusinessEntitySemanticService) calculateScore(
  fkPresence, joinFreq, nameSim, textSim, edgePrior float64,
  weights map[string]float64,
) float64 {
  return (weights["fk"] * fkPresence +
          weights["join_freq"] * joinFreq +
          weights["name_sim"] * nameSim +
          weights["text_sim"] * textSim +
          weights["edge_prior"] * edgePrior) / 5.0
}

// Helper: Extract FK relationships from information_schema
func (s *BusinessEntitySemanticService) getForeignKeys(schema, table string) ([]ForeignKey, error) {
  // Query information_schema.referential_constraints
}

// Helper: Calculate name similarity (Levenshtein distance)
func (s *BusinessEntitySemanticService) calculateNameSimilarity(s1, s2 string) float64 {
  // Use Levenshtein or similar algorithm
}

// Helper: Calculate text similarity (semantic embeddings or TF-IDF)
func (s *BusinessEntitySemanticService) calculateTextSimilarity(desc1, desc2 string) float64 {
  // Use embeddings or TF-IDF
}
```

### Step 4: Frontend Integration

1. Install the service and hook in your entity details page:

```tsx
import { useBusinessEntitySemanticLayer } from '../hooks/useBusinessEntitySemanticLayer';
import SemanticAssetsTab from '../components/entity/SemanticAssetsTab';
import RelationshipSuggestionPanel from '../components/entity/RelationshipSuggestionPanel';
import RelatedObjectsNavigator from '../components/entity/RelatedObjectsNavigator';

export default function EntityDetailsPage() {
  const { entityKey, entity } = useParams();
  const { tenant, datasource } = useTenant();

  const semanticLayer = useBusinessEntitySemanticLayer({
    tenantId: tenant.id,
    datasourceId: datasource.id,
    businessEntityId: entity.id,
    businessEntityName: entity.name,
    semanticTermIds: entity.semantic_term_ids || [],
    sourceTableNames: entity.source_tables || [],
  });

  return (
    <Tabs>
      <TabsList>
        <TabsTrigger value="semantic-assets">Semantic Assets</TabsTrigger>
        <TabsTrigger value="suggestions">Suggestions</TabsTrigger>
        <TabsTrigger value="relationships">Related Objects</TabsTrigger>
      </TabsList>

      <TabsContent value="semantic-assets">
        <SemanticAssetsTab
          semanticAssets={semanticLayer.semanticAssets}
          isLoading={semanticLayer.assetsLoading}
          error={semanticLayer.assetsError}
          onGenerateCoreModel={semanticLayer.generateCoreModel}
          onGenerateCoreView={semanticLayer.generateCoreView}
          onCreateCustomModel={semanticLayer.createCustomModel}
          onCreateCustomView={semanticLayer.createCustomView}
          businessEntityName={entity.name}
        />
      </TabsContent>

      <TabsContent value="suggestions">
        <RelationshipSuggestionPanel
          suggestions={semanticLayer.relationshipSuggestions}
          isLoading={semanticLayer.suggestionsLoading}
          error={semanticLayer.suggestionsError}
          onApplySuggestion={semanticLayer.applyRelationshipSuggestion}
          entityName={entity.name}
        />
      </TabsContent>

      <TabsContent value="relationships">
        <RelatedObjectsNavigator
          linksTo={semanticLayer.relatedObjects.linksTo}
          linksFrom={semanticLayer.relatedObjects.linksFrom}
          isLoading={semanticLayer.relatedObjectsLoading}
          error={null}
          businessEntityName={entity.name}
          onTraverse={semanticLayer.traverseObjectGraph}
        />
      </TabsContent>
    </Tabs>
  );
}
```

## Scoring Algorithm

### Relationship Confidence Formula

```
Confidence = (w1 × FK + w2 × JoinFreq + w3 × NameSim + w4 × TextSim + w5 × EdgePrior) / 5

where:
- FK: 1.0 if foreign key exists, 0.0 otherwise
- JoinFreq: Observed join frequency in query logs (0.0-1.0)
- NameSim: Similarity between entity/column names (0.0-1.0, Levenshtein-based)
- TextSim: Semantic similarity of descriptions (0.0-1.0, embedding-based)
- EdgePrior: Prior probability for edge type (0.0-1.0, from catalog_edge_types)

Recommended weights:
- w1 = 1.0  (FK presence is strongest signal)
- w2 = 0.7  (Join frequency is strong signal)
- w3 = 0.4  (Name similarity is moderate)
- w4 = 0.3  (Text similarity is weak)
- w5 = 0.6  (Edge type priors are moderately strong)
```

### Confidence Interpretation

- **≥ 0.80**: High confidence (auto-accept ready)
- **0.60-0.79**: Medium confidence (review recommended)
- **< 0.60**: Low confidence (manual review required)

## Testing

### Unit Tests

```typescript
describe('useBusinessEntitySemanticLayer', () => {
  it('should generate core model', async () => {
    const { result } = renderHook(() => useBusinessEntitySemanticLayer(options));
    
    await act(async () => {
      await result.current.generateCoreModel();
    });

    expect(result.current.semanticAssets.coreModel).toBeDefined();
  });

  it('should fetch relationship suggestions', async () => {
    // Test implementation
  });

  it('should traverse object graph', async () => {
    // Test implementation
  });
});
```

### Integration Tests

1. Create a test entity with semantic terms
2. Generate core model/view
3. Create custom extensions
4. Apply relationship suggestions
5. Traverse object graph
6. Verify all catalog edges are created correctly

## Performance Considerations

1. **Caching**
   - Cache suggestion results (TTL: 1 hour)
   - Cache linked models (TTL: 30 minutes)
   - Cache relationship graphs (TTL: 15 minutes)

2. **Query Optimization**
   - Use indexes on `business_entity_id`, `datasource_id`, `confidence`
   - Batch suggestions queries
   - Paginate large result sets

3. **Background Jobs**
   - Regenerate suggestions nightly
   - Update join frequency metrics periodically
   - Clean up old audit logs (retention: 90 days)

## Tenant Isolation

All endpoints MUST include tenant context:

```
X-Tenant-ID: {tenant_id}
X-Tenant-Datasource-ID: {datasource_id}
```

Query parameters:
```
?tenant_id={tenant_id}&datasource_id={datasource_id}
```

Database filtering:
```sql
WHERE tenant_id = $1 AND datasource_id = $2
```

## Migration Path

### For Existing Entities

1. Identify business entities linked to semantic terms
2. Run batch job to generate core models/views
3. Create custom models/views as needed
4. Analyze FK relationships to generate suggestions
5. Reviewsuggestions and apply accepted ones

```sql
-- Batch core model generation
WITH entity_terms AS (
  SELECT DISTINCT 
    be.id as entity_id,
    be.name,
    ARRAY_AGG(st.id) as term_ids
  FROM business_entities be
  JOIN entity_semantic_terms est ON be.id = est.entity_id
  JOIN catalog_node st ON st.id = est.semantic_term_id
  GROUP BY be.id, be.name
)
INSERT INTO semantic_assets (
  tenant_id, datasource_id, business_entity_id, 
  semantic_term_ids, created_at
)
SELECT tenant_id, datasource_id, entity_id, term_ids, NOW()
FROM entity_terms;
```

## Future Enhancements

1. **Machine Learning**
   - Train models on historical accepted/dismissed suggestions
   - Personalize weight tuning per tenant/domain
   - Federated learning for cross-tenant patterns

2. **Semantic Enrichment**
   - Auto-generate business names/descriptions
   - Suggest synonyms and related terms
   - Pattern recognition for domain models

3. **Advanced Graph Features**
   - Multi-hop relationship discovery
   - Circular dependency detection
   - Path recommendation engine

4. **UI Enhancements**
   - Visual graph builder
   - Bulk suggestion application
   - Custom scoring rule designer

## Support & Troubleshooting

### Common Issues

1. **No suggestions generated**
   - Verify FK metadata is loaded into information_schema
   - Check semantic term mappings
   - Ensure table statistics are up-to-date

2. **Low confidence scores**
   - Review weight tuning for your domain
   - Check for data quality issues
   - Verify semantic term descriptions

3. **Performance issues**
   - Enable query result caching
   - Run background analysis jobs at off-peak times
   - Archive old audit logs

## References

- [Workday Business Object Model](https://community.workday.com/)
- [Semantic Data Modeling Best Practices](https://en.wikipedia.org/wiki/Semantic_data_model)
- [Foreign Key Analysis Patterns](https://www.postgresql.org/docs/)
- [String Similarity Algorithms](https://en.wikipedia.org/wiki/Levenshtein_distance)
