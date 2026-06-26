# Semantic Model Edge Types Documentation

## Overview

Two new edge types have been added to the catalog system to support semantic model relationships:

1. **semantic_model_extends** - Model Inheritance
2. **semantic_model_links_to** - Semantic Term Relationships

---

## 1. EXTENDS Edge (Inheritance)

**Edge Type Name:** `semantic_model_extends`  
**Edge Type ID:** `semantic_model_extends_edge`

### Purpose
Establishes the inheritance relationship between a custom semantic model and its base (core) model.

### Relationship
- **From Node:** Custom Semantic Model (e.g., `Customer_LTV`)
- **To Node:** Core Semantic Model (e.g., `Customer`)

### Semantics
- The custom model **extends** the core model
- Inherits all properties (semantic terms) from the base model
- Can add new terms or override inherited ones
- Establishes the specialization path

### Example Usage

```sql
-- Create an extends edge from Customer_LTV (custom) to Customer (core)
INSERT INTO public.catalog_edge (
    id, tenant_id, tenant_datasource_id, 
    source_node_id, target_node_id,
    edge_type_id, relationship_type, 
    properties, created_at, updated_at
) VALUES (
    gen_random_uuid(),
    'default',
    '<tenant_datasource_id>',
    '<customer_ltv_model_id>',  -- Custom model
    '<customer_core_model_id>',  -- Core model it extends
    'semantic_model_extends_edge',
    'semantic_model_extends',
    '{"inheritance_type": "extension", "inherits_all_terms": true}'::jsonb,
    NOW(),
    NOW()
);
```

### Properties (JSONB)
Suggested properties for the edge:
- `inheritance_type`: "extension" | "specialization"
- `inherits_all_terms`: boolean (typically true)
- `override_count`: number of overridden terms
- `new_term_count`: number of new terms added

---

## 2. LINKS_TO Edge (Content Definition)

**Edge Type Name:** `semantic_model_links_to`  
**Edge Type ID:** `semantic_model_links_to_edge`

### Purpose
Documents which semantic terms (dimensions and measures) a semantic model utilizes.

### Relationship
- **From Node:** Semantic Model (e.g., `Customer_LTV`)
- **To Node:** Semantic Term (e.g., `Revenue`, `CustomerID`, `OrderDate`)

### Semantics
- The model **uses** or **exposes** this semantic term
- Multiple links from one model to many terms
- Enforces "single source of truth" - term definition lives in the Semantic Term node
- Model simply declares "I use this term"

### Example Usage

```sql
-- Link Customer_LTV model to Revenue semantic term
INSERT INTO public.catalog_edge (
    id, tenant_id, tenant_datasource_id,
    source_node_id, target_node_id,
    edge_type_id, relationship_type,
    properties, created_at, updated_at
) VALUES (
    gen_random_uuid(),
    'default',
    '<tenant_datasource_id>',
    '<customer_ltv_model_id>',     -- Semantic Model
    '<revenue_semantic_term_id>',  -- Semantic Term
    'semantic_model_links_to_edge',
    'semantic_model_links_to',
    '{"term_type": "measure", "usage": "calculation", "is_inherited": false}'::jsonb,
    NOW(),
    NOW()
);
```

### Properties (JSONB)
Suggested properties for the edge:
- `term_type`: "dimension" | "measure" | "filter"
- `usage`: "direct" | "calculation" | "filter"
- `is_inherited`: boolean (true if from parent model)
- `is_overridden`: boolean (true if term definition is customized)
- `aggregation_override`: string (if different from term default)

---

## Database Schema

### Edge Type Definitions

```sql
-- In catalog_edge_types table
INSERT INTO public.catalog_edge_types (
    id, tenant_id, edge_type_name, description, 
    source_node_type_id, target_node_type_id
) VALUES
('semantic_model_extends_edge', 'default', 
 'semantic_model_extends', 
 'Semantic Model Extends (Inheritance)', 
 'semantic_model_type', 'semantic_model_type'),
 
('semantic_model_links_to_edge', 'default', 
 'semantic_model_links_to', 
 'Semantic Model Links To Semantic Term', 
 'semantic_model_type', 'semantic_term_type');
```

---

## Use Cases

### 1. Model Inheritance Tracking
Query all custom models that extend a core model:

```sql
SELECT 
    cn_custom.node_name as custom_model,
    cn_core.node_name as extends_from_core_model,
    ce.properties->>'inheritance_type' as inheritance_type
FROM catalog_edge ce
JOIN catalog_node cn_custom ON ce.source_node_id = cn_custom.id
JOIN catalog_node cn_core ON ce.target_node_id = cn_core.id
WHERE ce.relationship_type = 'semantic_model_extends';
```

### 2. Term Usage Analysis
Find all models using a specific semantic term:

```sql
SELECT 
    cn_model.node_name as model_name,
    cn_term.node_name as semantic_term,
    ce.properties->>'term_type' as term_type,
    ce.properties->>'usage' as usage
FROM catalog_edge ce
JOIN catalog_node cn_model ON ce.source_node_id = cn_model.id
JOIN catalog_node cn_term ON ce.target_node_id = cn_term.id
WHERE ce.relationship_type = 'semantic_model_links_to'
  AND cn_term.node_name = 'Revenue';
```

### 3. Model Lineage
Trace the full inheritance chain:

```sql
WITH RECURSIVE model_lineage AS (
    -- Base case: start with a specific custom model
    SELECT 
        id, node_name, 0 as depth
    FROM catalog_node
    WHERE node_name = 'Customer_LTV'
    
    UNION ALL
    
    -- Recursive case: follow extends edges
    SELECT 
        cn.id, cn.node_name, ml.depth + 1
    FROM model_lineage ml
    JOIN catalog_edge ce ON ce.source_node_id = ml.id
    JOIN catalog_node cn ON ce.target_node_id = cn.id
    WHERE ce.relationship_type = 'semantic_model_extends'
)
SELECT * FROM model_lineage ORDER BY depth;
```

---

## Integration with Existing System

These edge types integrate with:
- The `extends_model_id` property on semantic models (stores the parent model ID)
- The `linked_semantic_terms` property (stores array of term IDs)
- The semantic model service for auto-creating edges during model generation
- The frontend lineage visualization (SemanticFlow component)

---

## Migration Applied

Run the migration script to add these edge types:
```bash
psql -h localhost -U postgres -d <database_name> -f migrations/add_semantic_model_properties.sql
```

This will:
1. Add property schema to semantic_model node type ✓
2. Create the two new edge type definitions ✓
3. Verify both were created successfully ✓
