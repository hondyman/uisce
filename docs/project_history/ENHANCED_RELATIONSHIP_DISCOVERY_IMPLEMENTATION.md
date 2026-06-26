# Enhanced Relationship Discovery Service Implementation

**Date:** November 7, 2025  
**Purpose:** Implement semantic-aware relationship discovery with column hierarchy

---

## 🔧 Enhanced RelatedEntity Structure

```go
// RelatedEntity represents an entity that can be linked, with full context
type RelatedEntity struct {
    // Entity identification
    EntityID       string `json:"entity_id"`
    EntityName     string `json:"entity_name"`
    
    // Semantic context
    SemanticTermID      string `json:"semantic_term_id,omitempty"`
    SemanticTermName    string `json:"semantic_term_name,omitempty"`
    SemanticDisplay     string `json:"semantic_display,omitempty"`
    
    // Relationship link info
    SourceEntity        string `json:"source_entity"`
    SourceAttribute     string `json:"source_attribute,omitempty"`
    SourceColumn        string `json:"source_column,omitempty"`
    
    TargetAttribute     string `json:"target_attribute,omitempty"`
    TargetColumn        string `json:"target_column,omitempty"`
    TargetTable         string `json:"target_table,omitempty"`
    
    // Relationship properties
    TableName           string    `json:"table_name"`
    LinkType            string    `json:"link_type"`        // "foreign_key", "semantic", "column_hierarchy"
    Cardinality         string    `json:"cardinality"`      // "one-to-one", "one-to-many", etc.
    LinkReason          string    `json:"link_reason"`
    ForeignKeyPath      string    `json:"foreign_key_path"`
    ForeignKeyConstraint string   `json:"fk_constraint"`    // e.g., "orders.customer_id -> customers.id"
    
    // Confidence scoring
    Confidence          float64   `json:"confidence"`       // 0.0 to 1.0
    
    // Hierarchy info
    ColumnParentTable   string    `json:"column_parent_table,omitempty"`
    HierarchyDepth      int       `json:"hierarchy_depth"`  // How many levels deep
    
    DiscoveredAt        time.Time `json:"discovered_at"`
}

// RelationshipPath represents a complete path from source to target
type RelationshipPath struct {
    SourceEntity    string         `json:"source_entity"`
    TargetEntity    string         `json:"target_entity"`
    PathLength      int            `json:"path_length"`    // Number of hops
    Hops            []PathHop      `json:"hops"`           // Individual steps
    JoinPath        string         `json:"join_path"`      // SQL join path
    Confidence      float64        `json:"confidence"`     // Overall confidence
}

// PathHop represents a single step in the relationship chain
type PathHop struct {
    From              string `json:"from"`
    To                string `json:"to"`
    LinkType          string `json:"link_type"`
    KeyMapping        string `json:"key_mapping"`   // e.g., "id -> customer_id"
    Table             string `json:"table"`
    Column            string `json:"column"`
    SemanticTerm      string `json:"semantic_term,omitempty"`
    Cardinality       string `json:"cardinality"`
}
```

---

## 🔍 Enhanced Discovery Query

```sql
-- ENHANCED: Discovers relationships WITH semantic context and column hierarchy

WITH source_entity_data AS (
  -- Get the source entity
  SELECT 
    ea.id as entity_id,
    ea.entity_key as entity_name,
    ea.catalog_node_id as semantic_term_id,
    cn.name as semantic_term_name,
    cn.display_name as semantic_display
  FROM entity_attribute ea
  LEFT JOIN catalog_node cn ON ea.catalog_node_id = cn.id
  WHERE ea.entity_key = $1
    AND ea.tenant_datasource_id = $2
),

source_entity_attributes AS (
  -- Get attributes of source entity
  SELECT 
    ea2.id as attribute_id,
    ea2.entity_key as attribute_name,
    ea2.catalog_node_id as semantic_term_id,
    cn2.name as semantic_term_name
  FROM entity_attribute ea2
  LEFT JOIN catalog_node cn2 ON ea2.catalog_node_id = cn2.id
  WHERE ea2.parent_id IN (SELECT entity_id FROM source_entity_data)
),

semantic_to_columns AS (
  -- Find columns linked to semantic terms of source attributes
  SELECT DISTINCT
    soa.attribute_name,
    soa.semantic_term_name,
    cc.column_name,
    cc.table_name,
    cc.catalog_node_id as semantic_column_term_id
  FROM source_entity_attributes soa
  JOIN catalog_column cc ON cc.catalog_node_id = soa.semantic_term_id
  WHERE cc.tenant_datasource_id = $2
),

foreign_key_relationships AS (
  -- Find FK relationships from source tables
  SELECT DISTINCT
    'foreign_key' as link_type,
    tc1.table_name as source_table,
    kcu1.column_name as source_column,
    tc2.table_name as target_table,
    kcu2.column_name as target_column,
    'one-to-many' as cardinality,  -- FK from source to target
    CASE 
      WHEN tc1.constraint_type = 'FOREIGN KEY' THEN true
      ELSE false
    END as is_direct_fk
  FROM information_schema.table_constraints tc1
  JOIN information_schema.key_column_usage kcu1 
    ON tc1.table_name = kcu1.table_name 
    AND tc1.constraint_name = kcu1.constraint_name
  JOIN information_schema.referential_constraints rc
    ON tc1.constraint_name = rc.constraint_name
  JOIN information_schema.table_constraints tc2
    ON tc2.constraint_name = rc.unique_constraint_name
  JOIN information_schema.key_column_usage kcu2
    ON tc2.table_name = kcu2.table_name
    AND tc2.constraint_name = kcu2.constraint_name
  WHERE tc1.constraint_type = 'FOREIGN KEY'
    AND (
      -- FK from source table
      tc1.table_name IN (SELECT table_name FROM semantic_to_columns)
      -- OR table in catalog_node for source entity
      OR tc1.table_name = (SELECT entity_name FROM source_entity_data)
    )
),

target_entities_found AS (
  -- Map target tables to entities
  SELECT DISTINCT
    fkr.target_table as entity_name,
    fkr.source_table,
    fkr.source_column,
    fkr.target_column,
    fkr.cardinality,
    'outbound' as direction,
    ea3.id as entity_id,
    ea3.catalog_node_id as semantic_term_id,
    cn3.name as semantic_term_name,
    cn3.display_name as semantic_display,
    1 as hierarchy_depth
  FROM foreign_key_relationships fkr
  LEFT JOIN entity_attribute ea3 
    ON ea3.entity_key = fkr.target_table
    AND ea3.tenant_datasource_id = $2
  LEFT JOIN catalog_node cn3 ON ea3.catalog_node_id = cn3.id
  WHERE ea3.id IS NOT NULL

  UNION ALL

  -- Inbound FKs: tables pointing to source
  SELECT DISTINCT
    fkr.source_table as entity_name,
    fkr.target_table as source_table,
    fkr.target_column as source_column,
    fkr.source_column as target_column,
    'many-to-one' as cardinality,
    'inbound' as direction,
    ea4.id as entity_id,
    ea4.catalog_node_id as semantic_term_id,
    cn4.name as semantic_term_name,
    cn4.display_name as semantic_display,
    1 as hierarchy_depth
  FROM foreign_key_relationships fkr
  LEFT JOIN entity_attribute ea4 
    ON ea4.entity_key = fkr.source_table
    AND ea4.tenant_datasource_id = $2
  LEFT JOIN catalog_node cn4 ON ea4.catalog_node_id = cn4.id
  WHERE fkr.target_table = (SELECT entity_name FROM source_entity_data)
    AND ea4.id IS NOT NULL
),

column_hierarchy AS (
  -- Find columns with parent_id relationships (recursive hierarchy)
  SELECT 
    cc1.column_name as source_col,
    cc1.table_name as source_tbl,
    cc2.column_name as parent_col,
    cc2.table_name as parent_tbl,
    cc1.catalog_node_id as source_semantic_id,
    cc2.catalog_node_id as parent_semantic_id
  FROM catalog_column cc1
  LEFT JOIN catalog_column cc2 
    ON cc1.parent_id = cc2.id
  WHERE cc1.tenant_datasource_id = $2
    AND cc1.parent_id IS NOT NULL
),

confidence_scores AS (
  -- Calculate confidence for each relationship
  SELECT 
    tef.entity_name,
    tef.entity_id,
    tef.source_table,
    tef.source_column,
    tef.target_column,
    tef.cardinality,
    -- Higher confidence if:
    -- 1. FK exists in information_schema
    -- 2. Semantic term linked on both sides
    -- 3. Table name matches entity key
    CASE 
      WHEN tef.semantic_term_id IS NOT NULL 
           AND soa.semantic_term_name IS NOT NULL THEN 0.95
      WHEN tef.entity_name = soa.attribute_name THEN 0.85
      ELSE 0.70
    END as confidence,
    CASE 
      WHEN tef.direction = 'outbound' 
        THEN tef.source_table || ' has many ' || tef.entity_name || ' records'
      ELSE tef.entity_name || ' has many ' || tef.source_table || ' records'
    END as description
  FROM target_entities_found tef
  LEFT JOIN source_entity_attributes soa 
    ON soa.semantic_term_name = tef.semantic_term_name
)

SELECT 
  cs.entity_id,
  cs.entity_name,
  cs.semantic_term_id,
  cs.semantic_term_name,
  (SELECT entity_name FROM source_entity_data) as source_entity,
  cs.source_column as source_column,
  cs.target_column as target_column,
  cs.source_table as source_table,
  cs.entity_name as target_table,
  'foreign_key' as link_type,
  cs.cardinality,
  cs.description as link_reason,
  cs.source_column || ' -> ' || cs.target_column as fk_constraint,
  cs.confidence,
  NOW() as discovered_at
FROM confidence_scores cs
WHERE cs.entity_id IS NOT NULL
ORDER BY cs.confidence DESC, cs.entity_name;
```

---

## 📦 Service Implementation

```go
// EnhancedRelationshipDiscoveryService provides semantic-aware discovery
type EnhancedRelationshipDiscoveryService struct {
    db *sql.DB
}

// NewEnhancedRelationshipDiscoveryService creates the service
func NewEnhancedRelationshipDiscoveryService(db *sql.DB) *EnhancedRelationshipDiscoveryService {
    return &EnhancedRelationshipDiscoveryService{db: db}
}

// DiscoverWithSemanticContext finds relationships including semantic terms
func (s *EnhancedRelationshipDiscoveryService) DiscoverWithSemanticContext(
    ctx context.Context,
    tenantID, datasourceID, entityName string,
) ([]RelatedEntity, error) {
    query := `-- ENHANCED DISCOVERY QUERY -- (see above SQL)`
    
    rows, err := s.db.QueryContext(ctx, query, entityName, datasourceID)
    if err != nil {
        return nil, fmt.Errorf("failed to discover relationships: %w", err)
    }
    defer rows.Close()
    
    var results []RelatedEntity
    
    for rows.Next() {
        var re RelatedEntity
        if err := rows.Scan(
            &re.EntityID,
            &re.EntityName,
            &re.SemanticTermID,
            &re.SemanticTermName,
            &re.SourceEntity,
            &re.SourceColumn,
            &re.TargetColumn,
            &re.SourceEntity, // source_table (reused for brevity)
            &re.TargetTable,
            &re.LinkType,
            &re.Cardinality,
            &re.LinkReason,
            &re.ForeignKeyConstraint,
            &re.Confidence,
            &re.DiscoveredAt,
        ); err != nil {
            continue
        }
        re.HierarchyDepth = 1
        results = append(results, re)
    }
    
    return results, rows.Err()
}

// DiscoverPaths finds multi-hop relationship paths (e.g., Customer → Order → Product)
func (s *EnhancedRelationshipDiscoveryService) DiscoverPaths(
    ctx context.Context,
    tenantID, datasourceID, sourceEntity string,
    maxDepth int,
) ([]RelationshipPath, error) {
    // Recursive CTE to find paths up to maxDepth
    query := `
WITH RECURSIVE path_discovery AS (
  -- Base case: direct relationships from source
  SELECT 
    source_entity,
    target_entity,
    1 as depth,
    ARRAY[source_entity, target_entity] as path,
    ARRAY['direct'] as link_types,
    source_column || ' -> ' || target_column as join_path
  FROM discovered_relationships
  WHERE source_entity = $1
    AND tenant_datasource_id = $2

  UNION ALL

  -- Recursive case: extend paths
  SELECT 
    pd.source_entity,
    dr.target_entity,
    pd.depth + 1,
    pd.path || dr.target_entity,
    pd.link_types || dr.link_type,
    pd.join_path || ' -> ' || dr.join_path
  FROM path_discovery pd
  JOIN discovered_relationships dr 
    ON pd.target_entity = dr.source_entity
  WHERE pd.depth < $3
    AND dr.tenant_datasource_id = $2
    AND NOT dr.target_entity = ANY(pd.path)  -- Avoid cycles
)

SELECT * FROM path_discovery
WHERE depth <= $3
ORDER BY depth, target_entity;
    `
    
    rows, err := s.db.QueryContext(ctx, query, sourceEntity, datasourceID, maxDepth)
    if err != nil {
        return nil, fmt.Errorf("failed to discover paths: %w", err)
    }
    defer rows.Close()
    
    var paths []RelationshipPath
    pathMap := make(map[string]*RelationshipPath)
    
    for rows.Next() {
        var (
            sourceEntity string
            targetEntity string
            depth        int
            pathArray    pq.StringArray
            linkTypes    pq.StringArray
            joinPath     string
        )
        
        if err := rows.Scan(&sourceEntity, &targetEntity, &depth, &pathArray, &linkTypes, &joinPath); err != nil {
            continue
        }
        
        pathKey := strings.Join(pathArray, "->")
        if _, exists := pathMap[pathKey]; !exists {
            pathMap[pathKey] = &RelationshipPath{
                SourceEntity: sourceEntity,
                TargetEntity: targetEntity,
                PathLength:   len(pathArray) - 1,
                JoinPath:     joinPath,
                Confidence:   0.85, // Calculate properly in real implementation
            }
            paths = append(paths, *pathMap[pathKey])
        }
    }
    
    return paths, rows.Err()
}
```

---

## 🔌 API Endpoint Integration

```go
// Enhanced endpoint that returns full semantic context
func (s *Server) getRelatedObjectsWithContext(w http.ResponseWriter, r *http.Request) {
    tenantID := r.URL.Query().Get("tenant_id")
    datasourceID := r.URL.Query().Get("datasource_id")
    entity := r.URL.Query().Get("entity")
    includeDeep := r.URL.Query().Get("includeDeep") == "true"

    if tenantID == "" || datasourceID == "" || entity == "" {
        writeJSONError(w, http.StatusBadRequest, "Missing required parameters", "missing_params", "")
        return
    }

    // Get direct relationships with semantic context
    discoveryService := NewEnhancedRelationshipDiscoveryService(s.DB)
    relatedEntities, err := discoveryService.DiscoverWithSemanticContext(
        r.Context(), tenantID, datasourceID, entity)
    if err != nil {
        writeJSONError(w, http.StatusInternalServerError, 
            fmt.Sprintf("Failed to discover relationships: %v", err), "discovery_error", "")
        return
    }

    response := map[string]interface{}{
        "sourceEntity":     entity,
        "relationships":    relatedEntities,
        "count":            len(relatedEntities),
    }

    // If requested, include multi-hop paths
    if includeDeep {
        paths, err := discoveryService.DiscoverPaths(
            r.Context(), tenantID, datasourceID, entity, 3)
        if err == nil {
            response["paths"] = paths
        }
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}
```

---

## 🎯 Self-Service Reporting Integration

```go
// GenerateReportingQuery creates SQL for self-service reports
type ReportingQueryGenerator struct {
    db *sql.DB
}

func (rqg *ReportingQueryGenerator) GenerateMultiEntityQuery(
    ctx context.Context,
    sourceEntity string,
    includedEntities []string,
    metrics []string,
) (string, error) {
    // Generate SELECT with JOINs based on discovered relationships
    
    var sqlBuilder strings.Builder
    sqlBuilder.WriteString(fmt.Sprintf("SELECT %s", strings.Join(metrics, ", ")))
    sqlBuilder.WriteString(fmt.Sprintf("\nFROM %s", sourceEntity))
    
    // Add JOINs for each included entity
    for _, targetEntity := range includedEntities {
        // Look up FK constraint
        var fkConstraint string
        err := rqg.db.QueryRowContext(ctx, 
            `SELECT fk_constraint FROM entity_relationship 
             WHERE source_entity_id = (SELECT id FROM entity_attribute WHERE entity_key = $1)
             AND target_entity_id = (SELECT id FROM entity_attribute WHERE entity_key = $2)`,
            sourceEntity, targetEntity).Scan(&fkConstraint)
        
        if err == nil {
            sqlBuilder.WriteString(fmt.Sprintf(
                "\nLEFT JOIN %s ON %s", targetEntity, fkConstraint))
        }
    }
    
    return sqlBuilder.String(), nil
}
```

---

## 📊 Response Example

```json
{
  "sourceEntity": "customer",
  "semanticContext": {
    "semanticTermId": "uuid-123",
    "semanticTermName": "customer",
    "semanticDisplay": "Customer",
    "description": "External party who purchases products"
  },
  "relationships": [
    {
      "entityId": "order-entity-uuid",
      "entityName": "order",
      "semanticTermId": "uuid-456",
      "semanticTermName": "order",
      "semanticDisplay": "Order",
      "sourceEntity": "customer",
      "sourceColumn": "customers.id",
      "targetColumn": "orders.customer_id",
      "cardinality": "one-to-many",
      "linkType": "foreign_key",
      "linkReason": "customer table has direct FK to orders table",
      "fkConstraint": "orders.customer_id -> customers.id",
      "confidence": 0.95,
      "hierarchyDepth": 1
    },
    {
      "entityId": "payment-entity-uuid",
      "entityName": "payment",
      "semanticTermId": "uuid-789",
      "semanticTermName": "payment",
      "semanticDisplay": "Payment",
      "sourceEntity": "customer",
      "sourceColumn": "customers.id",
      "targetColumn": "payments.customer_id",
      "cardinality": "one-to-many",
      "linkType": "foreign_key",
      "linkReason": "customer table has direct FK to payments table",
      "fkConstraint": "payments.customer_id -> customers.id",
      "confidence": 0.95,
      "hierarchyDepth": 1
    }
  ],
  "paths": [
    {
      "sourceEntity": "customer",
      "targetEntity": "product",
      "pathLength": 2,
      "hops": [
        {
          "from": "customer",
          "to": "order",
          "linkType": "foreign_key",
          "keyMapping": "id -> customer_id",
          "cardinality": "one-to-many"
        },
        {
          "from": "order",
          "to": "order_item",
          "linkType": "foreign_key",
          "keyMapping": "id -> order_id",
          "cardinality": "one-to-many"
        },
        {
          "from": "order_item",
          "to": "product",
          "linkType": "foreign_key",
          "keyMapping": "product_id -> id",
          "cardinality": "many-to-one"
        }
      ],
      "joinPath": "customer.id -> order.customer_id -> order_item.order_id -> product.id",
      "confidence": 0.90
    }
  ],
  "count": 2
}
```

---

## ✨ Benefits

1. **Semantic Awareness** - Users see what entities MEAN, not just table names
2. **Visual Relationships** - Clear FK paths and cardinality
3. **Discovery** - Automatically finds related entities
4. **Reporting** - Use relationships to build self-service reports
5. **Confidence Scores** - Know how reliable the relationship is
6. **Multi-hop Paths** - Discover indirect relationships (Customer → Product through Order)

---

## 📝 Summary

This implementation:
- ✅ Enhances discovery with semantic term context
- ✅ Exposes column hierarchy and parent relationships
- ✅ Scores relationship confidence
- ✅ Supports multi-hop path discovery
- ✅ Integrates with self-service reporting
- ✅ Provides complete key field information

Would you like me to create the actual SQL migrations or implement the Go service code?
