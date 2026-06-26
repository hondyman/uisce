# Cube.dev Properties Implementation Guide

**Date**: January 4, 2026  
**Status**: Implementation Ready  
**Priority**: High - Enables Cube.js integration and BI tool compatibility

---

## 🎯 Objective

Ensure the semantic term catalog system has comprehensive property definitions aligned with Cube.dev specifications for Dimensions, Measures, Time, Hierarchies, and Segments.

This enables:
- ✅ Automatic Cube.js configuration generation
- ✅ BI tool integration (Metabase, Tableau, etc.)
- ✅ Rich semantic metadata storage
- ✅ Advanced analytics capabilities

---

## 📋 Implementation Checklist

### Phase 1: Schema & Database (READY)
- ✅ Created `SEMANTIC_TERM_CUBE_PROPERTIES_SPECIFICATION.md` - Complete property schema
- ✅ Created `20260104_init_cube_semantic_term_types.sql` - Migration to initialize node/edge types
- ⏳ **ACTION**: Run migration to create catalog_node_type and catalog_edge_types definitions

### Phase 2: Semantic Term Wizard Enhancement (READY)
- ✅ Property inference engine completed (Jan 4, 2026)
- ✅ `inferSemanticTermProperties()` implemented in both services
- ⏳ **ACTION**: Update to populate `cube_properties` in addition to inferred properties

### Phase 3: API Enhancements (READY)
- ⏳ **ACTION**: Update glossary handler to expose cube properties
- ⏳ **ACTION**: Create endpoint to export semantic terms as Cube.js YAML/JSON
- ⏳ **ACTION**: Add validation for required Cube.dev properties

### Phase 4: Testing & Validation (READY)
- ⏳ **ACTION**: Add property validation tests
- ⏳ **ACTION**: Test Cube.js configuration export
- ⏳ **ACTION**: Verify BI tool integration

### Phase 5: Documentation & Rollout (READY)
- ✅ Property specification document created
- ✅ Migration file created
- ⏳ **ACTION**: Update user documentation
- ⏳ **ACTION**: Create admin guide for property management

---

## 🗄️ Database Changes Required

### 1. Run Migration

```bash
# Apply the migration
psql -U postgres -d alpha -f backend/db/migrations/20260104_init_cube_semantic_term_types.sql

# Verify
psql -U postgres -d alpha -c "
SELECT catalog_type_name, description 
FROM catalog_node_type 
WHERE catalog_type_name LIKE 'semantic_term_%' 
ORDER BY catalog_type_name;
"
```

Expected output:
```
        catalog_type_name        |                    description
---------------------------------+----------------------------------------------------
 semantic_term_dimension         | Cube.js Dimension - An attribute related to a...
 semantic_term_hierarchy         | Cube.js Hierarchy - Groups dimensions for drill...
 semantic_term_measure           | Cube.js Measure - An aggregation over a column...
 semantic_term_segment           | Cube.js Segment - Pre-calculated filter or coh...
 semantic_term_time              | Cube.js Time Dimension - Temporal attribute wit...
(5 rows)
```

### 2. Verify Edge Types

```sql
SELECT edge_type_name, is_active 
FROM catalog_edge_types 
WHERE edge_type_name LIKE '%dimension%' 
   OR edge_type_name LIKE '%measure%'
ORDER BY edge_type_name;
```

Expected:
```
        edge_type_name         | is_active
--------------------------------+-----------
 dimension_references_time      | t
 hierarchy_contains_dimension   | t
 hierarchy_organizes_time       | t
 measure_aggregates_dimension   | t
 measure_uses_time              | t
 segment_filters_measure        | t
(6 rows)
```

---

## 💻 Code Implementation

### 1. Update `inferSemanticTermProperties()` in Semantic Mapping Service

**File**: `backend/internal/analytics/semantic_mapping_service.go`

Add cube_properties generation:

```go
func (s *SemanticMappingService) inferSemanticTermProperties(
    column *DatabaseColumn, 
    termType string, 
    columnName string,
    termName string, // NEW
) map[string]interface{} {
    properties := map[string]interface{}{
        "data_type": termType,
    }

    // Existing inference logic...
    
    // NEW: Add Cube.dev properties
    cubeProperties := s.generateCubeProperties(column, termType, columnName, termName)
    properties["cube_properties"] = cubeProperties
    
    // NEW: Add semantic term metadata
    properties["semantic_term_type"] = s.detectSemanticTermType(column, termType)
    properties["semantic_term_name"] = termName
    
    return properties
}

// NEW METHOD: Generate Cube.dev compatible properties
func (s *SemanticMappingService) generateCubeProperties(
    column *DatabaseColumn,
    termType string,
    columnName string,
    termName string,
) map[string]interface{} {
    cubeProps := map[string]interface{}{
        "name": columnName,
        "sql": fmt.Sprintf("{CUBE}.%s", columnName),
        "type": s.mapToCubeType(termType),
        "title": s.generateTitle(columnName),
        "description": fmt.Sprintf("Semantic term: %s", termName),
        "public": true,
    }
    
    // Add primary_key if it's an ID column
    if strings.HasSuffix(strings.ToUpper(columnName), "_ID") {
        cubeProps["primary_key"] = false // Could be, but don't assume
    }
    
    // Add order for temporal
    if strings.Contains(strings.ToUpper(columnName), "CREATED") ||
       strings.Contains(strings.ToUpper(columnName), "UPDATED") {
        cubeProps["order"] = "desc"
    }
    
    return cubeProps
}

// NEW METHOD: Detect semantic term type (Dimension, Measure, Time, Hierarchy, Segment)
func (s *SemanticMappingService) detectSemanticTermType(
    column *DatabaseColumn,
    dataType string,
) string {
    columnUpper := strings.ToUpper(column.Column)
    
    // Time dimension detection
    if strings.Contains(columnUpper, "DATE") ||
       strings.Contains(columnUpper, "TIME") ||
       strings.Contains(columnUpper, "TIMESTAMP") ||
       strings.Contains(columnUpper, "CREATED") ||
       strings.Contains(columnUpper, "UPDATED") {
        return "Time"
    }
    
    // Measure detection (numeric, count, sum)
    if strings.Contains(dataType, "INT") ||
       strings.Contains(dataType, "NUMERIC") ||
       strings.Contains(dataType, "DECIMAL") {
        if strings.Contains(columnUpper, "AMOUNT") ||
           strings.Contains(columnUpper, "TOTAL") ||
           strings.Contains(columnUpper, "COUNT") ||
           strings.Contains(columnUpper, "REVENUE") ||
           strings.Contains(columnUpper, "SALES") {
            return "Measure"
        }
    }
    
    // Default to Dimension
    return "Dimension"
}

// NEW METHOD: Map data type to Cube.js type
func (s *SemanticMappingService) mapToCubeType(dataType string) string {
    dtUpper := strings.ToUpper(dataType)
    
    if strings.Contains(dtUpper, "TIME") ||
       strings.Contains(dtUpper, "DATE") ||
       strings.Contains(dtUpper, "TIMESTAMP") {
        return "time"
    }
    
    if strings.Contains(dtUpper, "INT") ||
       strings.Contains(dtUpper, "NUMERIC") ||
       strings.Contains(dtUpper, "DECIMAL") ||
       strings.Contains(dtUpper, "FLOAT") {
        return "number"
    }
    
    if strings.Contains(dtUpper, "BOOL") {
        return "boolean"
    }
    
    return "string"
}
```

### 2. Update Glossary Handler

**File**: `backend/internal/handlers/glossary_handler.go`

Add property exposure endpoint:

```go
// HandleGetSemanticTermWithCubeProperties returns semantic term with full Cube.dev properties
// GET /api/glossary/semantic-terms/{id}/cube-definition
func (h *GlossaryHandler) HandleGetSemanticTermWithCubeProperties(w http.ResponseWriter, r *http.Request) {
    logger := logging.GetLogger().Sugar()
    
    termID := chi.URLParam(r, "id")
    if termID == "" {
        h.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing term ID"})
        return
    }
    
    // Fetch semantic term with properties
    var node struct {
        ID         string          `db:"id"`
        NodeName   string          `db:"node_name"`
        NodeTypeID string          `db:"node_type_id"`
        Properties json.RawMessage `db:"properties"`
    }
    
    query := `
        SELECT id, node_name, node_type_id, properties 
        FROM catalog_node 
        WHERE id = $1 AND tenant_id = $2
    `
    
    tenantID := r.Header.Get("X-Tenant-ID")
    err := h.db.Get(&node, query, termID, tenantID)
    if err != nil {
        logger.Errorf("Failed to fetch semantic term: %v", err)
        h.respondJSON(w, http.StatusNotFound, map[string]string{"error": "Semantic term not found"})
        return
    }
    
    var properties map[string]interface{}
    json.Unmarshal(node.Properties, &properties)
    
    // Extract cube_properties if available
    response := map[string]interface{}{
        "id": node.ID,
        "name": node.NodeName,
        "node_type_id": node.NodeTypeID,
        "properties": properties,
    }
    
    if cubeProps, ok := properties["cube_properties"]; ok {
        response["cube_definition"] = cubeProps
    }
    
    h.respondJSON(w, http.StatusOK, response)
}

// HandleExportSemanticTermsAsCubeYaml exports all semantic terms as Cube.js YAML
// GET /api/glossary/semantic-terms/export/cube-yaml?datasource_id=...
func (h *GlossaryHandler) HandleExportSemanticTermsAsCubeYaml(w http.ResponseWriter, r *http.Request) {
    logger := logging.GetLogger().Sugar()
    
    datasourceID := r.URL.Query().Get("datasource_id")
    tenantID := r.Header.Get("X-Tenant-ID")
    
    query := `
        SELECT 
            node_name,
            properties,
            (SELECT catalog_type_name FROM catalog_node_type WHERE id = cn.node_type_id) as node_type
        FROM catalog_node cn
        WHERE tenant_datasource_id = $1 
          AND tenant_id = $2
          AND node_type_id IN (
              SELECT id FROM catalog_node_type 
              WHERE catalog_type_name LIKE 'semantic_term_%'
          )
        ORDER BY node_name
    `
    
    var terms []struct {
        NodeName string          `db:"node_name"`
        Properties json.RawMessage `db:"properties"`
        NodeType string          `db:"node_type"`
    }
    
    err := h.db.Select(&terms, query, datasourceID, tenantID)
    if err != nil {
        logger.Errorf("Failed to fetch semantic terms: %v", err)
        h.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to export terms"})
        return
    }
    
    // Build Cube.js configuration
    cubeConfig := h.buildCubeConfig(terms)
    yamlOutput := h.convertToCubeYaml(cubeConfig)
    
    w.Header().Set("Content-Type", "application/yaml")
    w.Header().Set("Content-Disposition", "attachment; filename=cube.yml")
    w.Write([]byte(yamlOutput))
}
```

### 3. Add Property Validation

**File**: `backend/internal/analytics/semantic_mapping_service.go`

```go
// NEW: Validate semantic term properties match Cube.dev spec
func (s *SemanticMappingService) ValidateSemanticTermProperties(
    properties map[string]interface{},
    termType string,
) error {
    // Check required cube_properties
    cubeProps, ok := properties["cube_properties"].(map[string]interface{})
    if !ok {
        return fmt.Errorf("missing cube_properties in semantic term")
    }
    
    // Validate required fields for all types
    requiredFields := []string{"name", "type"}
    for _, field := range requiredFields {
        if _, ok := cubeProps[field]; !ok {
            return fmt.Errorf("missing required cube property: %s", field)
        }
    }
    
    // Validate sql field (required for Dimension, Measure, Segment; optional for Hierarchy)
    termTypeStr, _ := properties["semantic_term_type"].(string)
    if termTypeStr != "Hierarchy" {
        if _, ok := cubeProps["sql"]; !ok {
            return fmt.Errorf("missing required sql in cube_properties for %s", termTypeStr)
        }
    }
    
    return nil
}
```

---

## 🧪 Testing

### Test Cases

```go
// Test 1: Dimension properties
func TestDimensionProperties(t *testing.T) {
    props := generateCubeProperties(
        &DatabaseColumn{Column: "USER_ID"},
        "Dimension",
        "USER_ID",
        "user_id",
    )
    
    assert.Equal(t, "user_id", props["name"])
    assert.Equal(t, "{CUBE}.USER_ID", props["sql"])
    assert.Equal(t, "number", props["type"])
}

// Test 2: Measure properties
func TestMeasureProperties(t *testing.T) {
    props := generateCubeProperties(
        &DatabaseColumn{Column: "TOTAL_REVENUE"},
        "Measure",
        "TOTAL_REVENUE",
        "total_revenue",
    )
    
    assert.Equal(t, "total_revenue", props["name"])
    assert.Contains(t, props["description"], "total_revenue")
}

// Test 3: Time properties
func TestTimeProperties(t *testing.T) {
    termType := detectSemanticTermType(&DatabaseColumn{Column: "CREATED_AT"}, "timestamp")
    assert.Equal(t, "Time", termType)
}

// Test 4: Property validation
func TestPropertyValidation(t *testing.T) {
    validProps := map[string]interface{}{
        "semantic_term_type": "Dimension",
        "cube_properties": map[string]interface{}{
            "name": "user_id",
            "sql": "{CUBE}.user_id",
            "type": "number",
        },
    }
    
    err := ValidateSemanticTermProperties(validProps, "Dimension")
    assert.Nil(t, err)
}

// Test 5: Cube YAML export
func TestCubeYamlExport(t *testing.T) {
    terms := []SemanticTerm{...}
    yaml := buildCubeYaml(terms)
    
    assert.Contains(t, yaml, "cubes:")
    assert.Contains(t, yaml, "dimensions:")
    assert.Contains(t, yaml, "measures:")
}
```

---

## 📊 Expected Database State After Implementation

### catalog_node_type entries:
```
semantic_term_dimension  ✓ Active
semantic_term_measure    ✓ Active
semantic_term_time       ✓ Active
semantic_term_hierarchy  ✓ Active
semantic_term_segment    ✓ Active
```

### catalog_edge_types entries:
```
hierarchy_contains_dimension     ✓ Active
measure_aggregates_dimension     ✓ Active
segment_filters_measure          ✓ Active
dimension_references_time        ✓ Active
measure_uses_time                ✓ Active
hierarchy_organizes_time         ✓ Active
```

---

## 🚀 Deployment Steps

### 1. Pre-Deployment Checklist
```bash
# Backup database
pg_dump alpha > backup_$(date +%Y%m%d).sql

# Verify migration syntax
psql -U postgres -d alpha -f backend/db/migrations/20260104_init_cube_semantic_term_types.sql --dry-run
```

### 2. Deploy Changes
```bash
# Update code (semantic mapping service)
# - Add cube_properties generation
# - Add validation logic

# Run migration
psql -U postgres -d alpha -f backend/db/migrations/20260104_init_cube_semantic_term_types.sql

# Rebuild backend
cd backend && go build -v ./cmd/api

# Test API endpoints
curl http://localhost:8080/api/glossary/semantic-terms/TERM_ID/cube-definition
curl http://localhost:8080/api/glossary/semantic-terms/export/cube-yaml?datasource_id=DS_ID
```

### 3. Verification
```bash
# Check catalog_node_type
psql -U postgres -d alpha -c "
SELECT COUNT(*) FROM catalog_node_type 
WHERE catalog_type_name LIKE 'semantic_term_%';
" # Should return: 5

# Check catalog_edge_types
psql -U postgres -d alpha -c "
SELECT COUNT(*) FROM catalog_edge_types 
WHERE edge_type_name IN (
    'hierarchy_contains_dimension',
    'measure_aggregates_dimension',
    'segment_filters_measure',
    'dimension_references_time',
    'measure_uses_time',
    'hierarchy_organizes_time'
);
" # Should return: 6

# Sample semantic term with cube_properties
psql -U postgres -d alpha -c "
SELECT properties FROM catalog_node 
WHERE node_name = 'USER_ID' LIMIT 1;
" # Should contain: cube_properties object
```

---

## 📚 Documentation Updates Needed

1. **User Guide**: How to create and manage semantic terms
2. **Admin Guide**: Property structure and validation
3. **API Documentation**: New endpoints for cube property export
4. **Integration Guide**: Using exported Cube.js config with BI tools

---

## 🎯 Success Criteria

- ✅ Migration runs successfully without errors
- ✅ All 5 semantic term node types created
- ✅ All 6 relationship edge types created
- ✅ Semantic term wizard populates cube_properties
- ✅ Cube property validation works
- ✅ API exports semantic terms as Cube.js YAML
- ✅ BI tools can import and use exported configuration
- ✅ All existing tests pass
- ✅ New property validation tests pass

---

## ⏱️ Timeline

| Phase | Task | Est. Time | Status |
|-------|------|-----------|--------|
| 1 | Run migration | 15 min | Ready |
| 2 | Update semantic mapping service | 2 hours | Ready |
| 3 | Enhance glossary handler | 1.5 hours | Ready |
| 4 | Add tests | 1 hour | Ready |
| 5 | Deploy & verify | 1 hour | Ready |
| 6 | Documentation | 1 hour | Ready |
| **Total** | | **~7 hours** | **Ready to Execute** |

---

## 📞 Support & Questions

**Specification**: See `SEMANTIC_TERM_CUBE_PROPERTIES_SPECIFICATION.md`

**Migration**: See `backend/db/migrations/20260104_init_cube_semantic_term_types.sql`

**Property Examples**: See specification document section "Complete Semantic Term Property Schema"

---

**Status**: ✅ **READY FOR IMPLEMENTATION**

All necessary documentation, specifications, and migration files are complete and ready for deployment.

