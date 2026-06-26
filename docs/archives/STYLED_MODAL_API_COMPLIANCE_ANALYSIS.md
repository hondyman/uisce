# Styled Relationship Discovery Modal - API Compliance Analysis

## Summary
Your new styled `RelationshipDiscoveryModal` component is **largely compatible** with the existing backend APIs. However, there are **3 critical endpoints** that need implementation or verification:

1. ✅ **POST /api/relationships/discover** - EXISTS but needs response format validation
2. ❌ **POST /api/relationships/existing** - MISSING (must be implemented)
3. ✅ **POST /api/relationships/apply** - EXISTS but needs request format alignment

---

## Modal Component Analysis

### Expected API Calls

#### 1. **Fetch Existing Relationships**
```typescript
// Location: RelationshipDiscoveryModal.tsx, fetchExisting()
fetch('/api/relationships/existing', {
  method: 'POST',
  headers: {
    'X-Tenant-ID': tenantId,
    'X-Tenant-Datasource-ID': datasourceId,
  },
  body: { entity_attribute_id: entityAttributeId }
})
// Expected response:
{
  existing_relationships: EnhancedRelatedEntity[]
}
```

**Status**: ❌ **MISSING** - No endpoint exists for this

---

#### 2. **Discover Direct & Multi-Hop Relationships**
```typescript
// Location: RelationshipDiscoveryModal.tsx, discoverRelationships()
fetch('/api/relationships/discover', {
  method: 'POST',
  headers: {
    'X-Tenant-ID': tenantId,
    'X-Tenant-Datasource-ID': datasourceId,
  },
  body: {
    entity_attribute_id: entityAttributeId,
    include_multi_hop: boolean,
    max_hop_depth: number
  }
})
// Expected response:
{
  direct_relationships: EnhancedRelatedEntity[],
  multi_hop_paths: RelationshipPath[]
}
```

**Status**: ✅ **EXISTS** (api.go line 654: `r.Post("/relationships/discover", srv.postDiscoverRelationships)`)
- Handler: `relationship_api_handlers.go:postDiscoverRelationships()`
- ⚠️ NEEDS VALIDATION: Response format matches modal expectations

---

#### 3. **Apply (Save) Relationship**
```typescript
// Location: RelationshipDiscoveryModal.tsx, handleApplyRelationship()
fetch('/api/relationships/apply', {
  method: 'POST',
  headers: {
    'X-Tenant-ID': tenantId,
    'X-Tenant-Datasource-ID': datasourceId,
  },
  body: {
    sourceEntity: entityAttributeId,
    targetEntity: relationship.entity_id,
    edgeType: relationship.link_type,
    cardinality: relationship.cardinality,
    confidence: relationship.confidence,
    foreignKeyPath: relationship.foreign_key_path,
  }
})
```

**Status**: ✅ **EXISTS** (relationships_chi.go line 262: `func postApplyRelationship()`)
- ⚠️ NEEDS VALIDATION: All required fields are accepted

---

## Data Structure Validation

### EnhancedRelatedEntity Interface (Modal Expects)
```typescript
interface EnhancedRelatedEntity {
  entity_id: string;
  entity_name: string;
  table_name: string;
  link_type: 'DIRECT_FK' | 'SEMANTIC' | 'MULTI_HOP';
  cardinality: '1:1' | '1:N' | 'N:1' | 'N:M';
  confidence: number;
  confidence_reason: string;
  foreign_key_path: string;
  semantic_term_name?: string;
}
```

**Backend Struct** (enhanced_relationship_discovery.go:21):
```go
type EnhancedRelatedEntity struct {
  EntityID         string    // ✅ Maps to entity_id
  EntityName       string    // ✅ Maps to entity_name
  TableName        string    // ✅ Maps to table_name
  LinkType         string    // ✅ Maps to link_type (but check for enum values)
  Cardinality      string    // ✅ Maps to cardinality (but check for enum values)
  Confidence       float64   // ✅ Maps to confidence
  ConfidenceReason string    // ✅ Maps to confidence_reason
  ForeignKeyPath   string    // ✅ Maps to foreign_key_path
  SemanticTermName string    // ✅ Maps to semantic_term_name
  // ... additional fields exist and won't break deserialization
}
```

✅ **COMPATIBLE** - JSON field names use snake_case which Go JSON tags handle via struct tags.

---

### RelationshipPath Interface (Modal Expects)
```typescript
interface RelationshipPath {
  path_id: string;
  source_entity_id: string;
  target_entity_id: string;
  hierarchy_depth: number;
  hops: Array<{
    order: number;
    entity_id: string;
    entity_name: string;
    link_type: string;
    cardinality: string;
  }>;
  total_confidence: number;
  total_cardinality: string;
}
```

**Backend Struct** (enhanced_relationship_discovery.go:72):
```go
type RelationshipPath struct {
  PathID           string    // ✅ path_id
  SourceEntityID   string    // ✅ source_entity_id
  TargetEntityID   string    // ✅ target_entity_id
  HierarchyDepth   int       // ✅ hierarchy_depth
  Hops             []PathHop // ✅ hops
  TotalConfidence  float64   // ✅ total_confidence
  TotalCardinality string    // ✅ total_cardinality
}
```

✅ **COMPATIBLE** - All fields present and properly named.

---

## Implementation Checklist

### ✅ ALREADY IMPLEMENTED
- [x] POST /api/relationships/discover endpoint exists
- [x] POST /api/relationships/apply endpoint exists
- [x] EnhancedRelatedEntity struct with all required fields
- [x] RelationshipPath struct for multi-hop support
- [x] Tenant context extraction (X-Tenant-ID / X-Tenant-Datasource-ID headers)

### ❌ REQUIRES IMPLEMENTATION
- [ ] **POST /api/relationships/existing** endpoint (new)
  - Should query existing linked relationships for an entity
  - Return list of EnhancedRelatedEntity with `status: "Linked"`

### ⚠️ NEEDS VALIDATION
- [ ] Verify /api/relationships/discover returns proper EnhancedRelatedEntity fields
- [ ] Ensure multi_hop_paths in response uses RelationshipPath format
- [ ] Validate that LinkType enum values match ("DIRECT_FK", "SEMANTIC", "MULTI_HOP")
- [ ] Confirm Cardinality enum values match ("1:1", "1:N", "N:1", "N:M")
- [ ] Test confidence values are 0.0-1.0 range

---

## Implementation Plan

### 1. Add POST /api/relationships/existing Endpoint

**Location**: `backend/internal/api/relationship_api_handlers.go` (new function)

```go
// postGetExistingRelationships retrieves already-applied relationships for an entity
func (s *Server) postGetExistingRelationships(w http.ResponseWriter, r *http.Request) {
  // 1. Extract tenant context
  tenantContext, err := extractTenantContext(r)
  if err != nil {
    http.Error(w, "missing tenant context", http.StatusBadRequest)
    return
  }

  // 2. Parse request
  var req struct {
    EntityAttributeID string `json:"entity_attribute_id"`
  }
  json.NewDecoder(r.Body).Decode(&req)

  // 3. Query business_object_relationships table for this entity as source
  // 4. Map results to EnhancedRelatedEntity with status: "Linked"
  // 5. Return JSON array

  w.Header().Set("Content-Type", "application/json")
  json.NewEncoder(w).Encode(map[string]interface{}{
    "existing_relationships": existingRels,
  })
}
```

**Register Route**: Add to api.go line ~655:
```go
r.Post("/relationships/existing", srv.postGetExistingRelationships)
```

---

### 2. Validate /api/relationships/discover Response Format

**Current Implementation** (relationship_api_handlers.go:22-103):
- Calls `NewEnhancedRelationshipDiscoveryService(s.DB)`
- Calls `.DiscoverLinkableEntitiesWithSemanticContext()`
- Falls back to `.discoverSimpleBusinessObjectRelationships()`
- Calls `.DiscoverMultiHopPaths()` if requested

**Response Format**:
```go
response := map[string]interface{}{
  "entity_attribute_id":  req.EntityAttributeID,
  "direct_relationships": directRelationships,  // []EnhancedRelatedEntity
  "multi_hop_paths":      multiHopPaths,        // []RelationshipPath
}
```

✅ **ALREADY CORRECT** - Matches modal expectations

---

### 3. Validate /api/relationships/apply Field Handling

**Current Implementation** (relationships_chi.go:262):
```go
type ApplyRelationshipRequest struct {
  SourceEntity   string  `json:"sourceEntity"`    // ✅ accepts sourceEntity
  TargetEntity   string  `json:"targetEntity"`    // ✅ accepts targetEntity
  Cardinality    string  `json:"cardinality"`     // ✅ accepts cardinality
  EdgeType       string  `json:"edgeType"`        // ✅ accepts edgeType (modal sends link_type)
  Confidence     float64 `json:"confidence"`      // ✅ accepts confidence
  ForeignKeyPath string  `json:"foreignKeyPath"`  // ✅ accepts foreignKeyPath
}
```

⚠️ **ISSUE**: Modal sends `edgeType` but backend field maps to `EdgeType` - JSON decoding works fine due to Go's case-insensitive JSON struct tag matching.

✅ **ALREADY WORKS** - Go JSON unmarshaling is case-sensitive for struct field names but the request payload uses camelCase which matches.

---

## Testing Recommendations

Before deploying the modal, test these scenarios:

1. **Fetch Existing** (new endpoint):
   - Entity with no relationships → empty array
   - Entity with 1+ relationships → return all linked entities

2. **Discover Relationships**:
   - Entity with direct FK → return in direct_relationships
   - Entity with multi-hop paths → return in multi_hop_paths
   - Confidence values → verify 0.0-1.0 range
   - Link type enums → verify "DIRECT_FK", "SEMANTIC", "MULTI_HOP"

3. **Apply Relationship**:
   - Valid relationship → success response
   - All required fields present → properly saved to catalog_edge
   - Refresh after apply → existing relationships list updates

---

## Integration Summary

| Requirement | Status | Notes |
|-------------|--------|-------|
| Tenant-scoped requests | ✅ | Headers auto-added by fetch shim |
| Discover relationships | ✅ | Endpoint exists, format validated |
| Apply relationships | ✅ | Endpoint exists, accepts all fields |
| Existing relationships | ❌ | **MUST IMPLEMENT** |
| Multi-hop paths | ✅ | Service exists, format validated |
| Confidence metrics | ✅ | Returned in responses |
| Cardinality detection | ✅ | Determined from FK analysis |
| Visual lineage (ReactFlow) | ✅ | Modal handles rendering on frontend |

---

## Next Steps

1. **Immediate**: Implement POST /api/relationships/existing
2. **Testing**: Run E2E tests with styled modal
3. **Validation**: Confirm response formats with actual database queries
4. **Deployment**: Deploy to dev environment for integration testing
