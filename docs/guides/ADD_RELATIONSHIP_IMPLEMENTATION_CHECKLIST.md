# Add Relationship Feature - Implementation Checklist

**Date:** November 7, 2025  
**Feature:** Enhanced Relationship Discovery & Self-Service Reporting

---

## 📋 Complete Implementation Roadmap

### Phase 1: Database Schema Enhancements ✅ DESIGN

- [ ] **Link columns to semantic terms**
  ```sql
  ALTER TABLE catalog_column 
  ADD COLUMN catalog_node_id UUID REFERENCES catalog_node(id);
  
  -- Update existing columns where semantic link exists
  UPDATE catalog_column cc
  SET catalog_node_id = (
    SELECT catalog_node_id FROM entity_attribute ea
    WHERE ea.entity_key LIKE '%' || cc.column_name || '%'
    LIMIT 1
  );
  ```

- [ ] **Create entity_attribute_column_mapping table**
  ```sql
  CREATE TABLE entity_attribute_column_mapping (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_attribute_id UUID NOT NULL REFERENCES entity_attribute(id) ON DELETE CASCADE,
    column_name TEXT NOT NULL,
    table_name TEXT NOT NULL,
    semantic_term_id UUID REFERENCES catalog_node(id),
    confidence NUMERIC(3,2) DEFAULT 0.80,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
  );
  
  CREATE INDEX idx_eacm_entity_attr ON entity_attribute_column_mapping(entity_attribute_id);
  CREATE INDEX idx_eacm_semantic ON entity_attribute_column_mapping(semantic_term_id);
  ```

- [ ] **Create entity_relationship table for storing discovered relationships**
  ```sql
  CREATE TABLE entity_relationship (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    tenant_datasource_id UUID NOT NULL REFERENCES tenant_product_datasource(id) ON DELETE CASCADE,
    source_entity_id UUID NOT NULL REFERENCES entity_attribute(id) ON DELETE CASCADE,
    target_entity_id UUID NOT NULL REFERENCES entity_attribute(id) ON DELETE CASCADE,
    relationship_type VARCHAR(100) NOT NULL, -- 'foreign_key', 'semantic', 'column_hierarchy'
    fk_constraint TEXT,                      -- e.g., "orders.customer_id -> customers.id"
    cardinality VARCHAR(50),                 -- 'one-to-one', 'one-to-many', etc.
    source_column VARCHAR(255),
    target_column VARCHAR(255),
    description TEXT,
    confidence NUMERIC(3,2) DEFAULT 0.80,   -- Confidence score 0.0 to 1.0
    hierarchy_depth INT DEFAULT 1,
    is_user_applied BOOLEAN DEFAULT false,   -- User manually applied vs. auto-discovered
    created_at TIMESTAMP DEFAULT NOW(),
    created_by VARCHAR(255),
    updated_at TIMESTAMP DEFAULT NOW()
  );
  
  CREATE INDEX idx_er_source_target ON entity_relationship(source_entity_id, target_entity_id);
  CREATE INDEX idx_er_tenant_ds ON entity_relationship(tenant_id, tenant_datasource_id);
  CREATE UNIQUE INDEX idx_er_unique_relationship 
    ON entity_relationship(source_entity_id, target_entity_id, tenant_datasource_id);
  ```

### Phase 2: Go Backend Implementation ✅ DESIGN

- [ ] **Create enhanced_relationship_discovery.go**
  - Implement `RelatedEntity` struct with semantic context
  - Implement `RelationshipPath` struct for multi-hop paths
  - Implement `PathHop` struct for individual steps
  - Create `EnhancedRelationshipDiscoveryService`

- [ ] **Implement discovery methods**
  - `DiscoverWithSemanticContext()` - Find relationships with semantic terms
  - `DiscoverPaths()` - Find multi-hop relationship chains
  - `GetRelationshipConfidence()` - Score relationship reliability
  - `FilterByCardinality()` - Filter by 1:1, 1:N, M:N

- [ ] **Create relationship_service.go enhancements**
  - `ApplyRelationship()` - Save discovered relationship to DB
  - `GetAppliedRelationships()` - Retrieve saved relationships
  - `GetRelationshipPath()` - Get full path between two entities

- [ ] **Update api.go handlers**
  - Enhance `getRelatedObjects()` to use enhanced discovery
  - Add `getRelationshipPaths()` endpoint
  - Add `getAppliedRelationships()` endpoint
  - Add `saveRelationshipDecision()` endpoint (user accepts/rejects suggestion)

### Phase 3: Self-Service Reporting ✅ DESIGN

- [ ] **Create reporting_query_generator.go**
  - `GenerateMultiEntityQuery()` - Build SQL joins from relationships
  - `ValidateReportingQuery()` - Validate generated SQL
  - `ExecuteReportingQuery()` - Run generated query

- [ ] **Implement reporting endpoints**
  - `POST /api/reporting/generate` - Generate query from relationships
  - `POST /api/reporting/preview` - Preview query results
  - `GET /api/reporting/available-metrics` - Get metrics available in entities

- [ ] **Create report_builder.go**
  - `BuildReport()` - Assemble report from selected entities/metrics
  - `CacheReportDefinition()` - Store report for reuse
  - `GetReportHistory()` - Retrieve saved reports

### Phase 4: Frontend Integration 📋 DESIGN

- [ ] **Create UI components**
  - RelationshipDiscoveryModal.tsx
  - RelationshipPathVisualizer.tsx
  - RelationshipAppliedIndicator.tsx
  - ReportBuilder.tsx

- [ ] **Implement API calls**
  - `fetchRelatedObjects(entity)` - Discover relationships
  - `fetchRelationshipPaths(source, target)` - Get multi-hop paths
  - `applyRelationship(sourceEntity, targetEntity)` - Save relationship
  - `generateReportQuery(entities, metrics)` - Generate report

- [ ] **Create user workflows**
  - "Add Relationship" button flows
  - Relationship visualization
  - Path selection for multi-hop relationships
  - Report building wizard

### Phase 5: Testing & Validation 📋 DESIGN

- [ ] **Unit tests**
  - `TestDiscoverLinkableEntities()` - Basic discovery
  - `TestDiscoverPaths()` - Multi-hop path discovery
  - `TestConfidenceScoring()` - Confidence calculation
  - `TestApplyRelationship()` - Save relationship

- [ ] **Integration tests**
  - `TestEndToEndRelationshipDiscovery()` - Full flow
  - `TestReportGeneration()` - Report query generation
  - `TestMultiTenantIsolation()` - Tenant scoping

- [ ] **Data validation**
  - FK constraint validation
  - Semantic term consistency
  - Circular reference detection

---

## 🎯 Quick Implementation Guide

### Start Here (Priority 1)

```go
// 1. Enhanced discovery service
// File: backend/internal/api/enhanced_relationship_discovery.go

type RelatedEntity struct {
    EntityID              string
    EntityName            string
    SemanticTermID        string
    SemanticTermName      string
    SourceColumn          string
    TargetColumn          string
    Cardinality           string
    LinkType              string
    ForeignKeyConstraint  string
    Confidence            float64
}

type EnhancedRelationshipDiscoveryService struct {
    db *sql.DB
}

func (s *EnhancedRelationshipDiscoveryService) DiscoverWithSemanticContext(
    ctx context.Context,
    tenantID, datasourceID, entityName string,
) ([]RelatedEntity, error) {
    // Use enhanced query from ENHANCED_RELATIONSHIP_DISCOVERY_IMPLEMENTATION.md
    // Execute query and return RelatedEntity slice
}
```

### Step 2: API Endpoint

```go
// In api.go, enhance getRelatedObjects handler
func (s *Server) getRelatedObjectsWithContext(w http.ResponseWriter, r *http.Request) {
    tenantID := r.URL.Query().Get("tenant_id")
    datasourceID := r.URL.Query().Get("datasource_id")
    entity := r.URL.Query().Get("entity")
    
    discoveryService := NewEnhancedRelationshipDiscoveryService(s.DB)
    results, err := discoveryService.DiscoverWithSemanticContext(
        r.Context(), tenantID, datasourceID, entity)
    
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(results)
}
```

### Step 3: Register Endpoint

```go
// In api.go router setup
r.Get("/relationships/objects-enhanced", srv.getRelatedObjectsWithContext)
```

---

## 📊 Data Flow Diagram

```
User clicks "Add Relationship" on Entity
    │
    ▼
Frontend: GET /api/relationships/objects-enhanced?entity=customer
    │
    ▼
Backend: getRelatedObjectsWithContext()
    │
    ▼
Enhanced Discovery Service: DiscoverWithSemanticContext()
    │
    ├─ Query catalog_node (semantic terms)
    ├─ Query entity_attribute (entity definitions)
    ├─ Query catalog_column (column mappings)
    ├─ Query information_schema (FK constraints)
    └─ Calculate confidence scores
    │
    ▼
Returns RelatedEntity[] with:
    - Entity name & ID
    - Semantic term context
    - FK constraint path
    - Cardinality (1:1, 1:N, M:N)
    - Confidence score
    │
    ▼
Frontend: Display relationships in modal
    │
    ├─ Show source entity
    ├─ Show target entities
    ├─ Show FK paths
    ├─ Show cardinality
    └─ Allow user to select & apply
    │
    ▼
User clicks "Apply Relationship"
    │
    ▼
POST /api/relationships/apply
    │
    ▼
Backend: Saves to entity_relationship table
    │
    ▼
Relationship available for reporting
```

---

## 🔑 Key Implementation Points

### 1. Semantic Context Integration
- Always join `entity_attribute` with `catalog_node` via `catalog_node_id`
- Include semantic term name/display in all responses
- Link columns to semantic terms for context

### 2. FK Discovery
- Use information_schema for PostgreSQL FK constraints
- Support both outbound (source has FK) and inbound (target has FK)
- Calculate cardinality based on direction

### 3. Confidence Scoring
- FK exists in information_schema: 0.95
- Semantic term on both sides: +0.10 (cap at 0.95)
- Table name matches entity key: 0.85
- Otherwise: 0.70

### 4. Multi-Hop Paths
- Use recursive CTE for deep path discovery
- Limit depth to prevent infinite recursion (default: 3)
- Include all intermediate hops

### 5. Reporting Integration
- Store applied relationships in `entity_relationship` table
- Use relationship definitions to generate JOINs
- Support multi-entity metrics aggregation

---

## 📈 Metrics to Track

```
Discovery Success Rate
├─ Entities with discovered relationships
├─ Average number of relationships per entity
└─ Confidence score distribution

Relationship Application Rate
├─ Auto-discovered vs. user-created
├─ Confidence score of applied relationships
└─ Relationship usage in reports

Report Generation
├─ Multi-entity reports created
├─ Average entities per report
├─ Query execution time
└─ Error rate
```

---

## 🛠️ Files to Create/Modify

### Create New Files
- `backend/internal/api/enhanced_relationship_discovery.go` (NEW)
- `backend/internal/api/reporting_query_generator.go` (NEW)
- `backend/migrations/000031_entity_relationship_schema.sql` (NEW)
- `RELATIONSHIP_DISCOVERY_GUIDE.md` (CREATED)
- `ENHANCED_RELATIONSHIP_DISCOVERY_IMPLEMENTATION.md` (CREATED)

### Modify Existing Files
- `backend/internal/api/api.go`
  - Add enhanced handler
  - Register endpoint
- `backend/internal/api/api.go` router setup
  - Register new endpoints
- Database migration files
  - Add schema for relationships

---

## ✅ Acceptance Criteria

- [ ] User clicks "Add Relationship" on entity
- [ ] System discovers all related entities via FK chains
- [ ] Each relationship shows:
  - [ ] Source & target entity names
  - [ ] Semantic term context
  - [ ] FK constraint path
  - [ ] Cardinality
  - [ ] Confidence score
- [ ] User can select relationships to apply
- [ ] Selected relationships saved to database
- [ ] Can use relationships for self-service reporting
- [ ] Multi-hop paths discoverable (e.g., Customer → Order → Product)
- [ ] Semantic terms properly linked and displayed
- [ ] Multi-tenant isolation maintained

---

## 🚀 Deployment Steps

1. **Database Migration**
   - Run `000031_entity_relationship_schema.sql`
   - Verify tables created

2. **Go Service Deployment**
   - Deploy enhanced_relationship_discovery.go
   - Deploy API updates
   - Test endpoints

3. **Frontend Update**
   - Update to call enhanced endpoints
   - Add relationship discovery UI
   - Add reporting builder

4. **Validation**
   - Test with sample entities
   - Verify FK discovery
   - Test multi-hop paths
   - Validate reporting queries

---

## 📞 Support & Troubleshooting

### Common Issues

**Q: Relationships not discovered**
- Check: Are FK constraints defined in database?
- Check: Is tenant_datasource_id correct?
- Check: Are entities linked to semantic terms?

**Q: Low confidence scores**
- Likely: Semantic terms not linked to columns
- Fix: Add catalog_node_id to catalog_column
- Verify: Entity names match table names

**Q: Multi-hop paths not found**
- Check: Depth limit may be too low
- Check: May have circular references
- Validate: FK chains are continuous

---

## 📝 Summary

This implementation enables:
1. **Relationship Discovery** - Find related entities automatically
2. **Semantic Context** - Show what relationships mean
3. **Visual FK Paths** - Display how entities connect
4. **Self-Service Reporting** - Build reports using relationships
5. **Confidence Scoring** - Know relationship reliability
6. **Multi-Hop Paths** - Discover indirect relationships

**Next Steps:**
- [ ] Implement enhanced discovery service
- [ ] Create API endpoints
- [ ] Build reporting query generator
- [ ] Develop frontend components
- [ ] Deploy & validate
