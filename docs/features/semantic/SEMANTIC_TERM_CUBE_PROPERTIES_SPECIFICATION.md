# Semantic Term Properties - Cube.dev Specification Alignment

**Date**: January 4, 2026  
**Status**: Specification Definition

---

## Overview

This document defines the property structure for semantic terms in the catalog system, aligned with Cube.dev's specification for Dimensions, Measures, Hierarchies, Segments, and Time characteristics.

The semantic term wizard creates catalog_node entries with a `properties` JSONB field. This document standardizes what those properties should contain for each semantic term type.

---

## Semantic Term Types & Properties

### 1. DIMENSION Properties

A dimension represents an attribute related to a measure.

**Required Fields**:
- `name`: string - Unique identifier within the cube
- `sql`: string - SQL expression (e.g., `{CUBE}.column_name`)
- `type`: string - Data type (string, number, time, boolean, etc.)

**Optional Fields**:
```json
{
  "name": "user_id",
  "sql": "{CUBE}.user_id",
  "type": "number",
  
  "title": "User Identifier",
  "description": "Unique identifier for users",
  "public": true,
  "format": "string",
  "meta": { "any": "value" },
  "order": "asc",
  "primary_key": false,
  "propagate_filters_to_sub_query": false,
  "sub_query": false,
  
  "case": {
    "when": [
      { "sql": "{CUBE}.type = 'premium'", "label": "Premium" },
      { "sql": "{CUBE}.type = 'standard'", "label": "Standard" }
    ],
    "else": { "label": "Unknown" }
  },
  
  "granularities": [
    {
      "name": "quarter_hour",
      "interval": "15 minutes",
      "title": "Quarter Hour"
    },
    {
      "name": "week_starting_on_sunday",
      "interval": "1 week",
      "offset": "-1 day"
    },
    {
      "name": "fiscal_year_starting_on_april_01",
      "interval": "1 year",
      "origin": "2025-04-01"
    }
  ]
}
```

**Semantic Term Example**:
```json
{
  "semantic_term_type": "Dimension",
  "data_type": "number",
  "foreign_key": true,
  "nullable": false,
  
  "cube_properties": {
    "name": "user_id",
    "sql": "{CUBE}.user_id",
    "type": "number",
    "title": "User ID",
    "description": "Unique user identifier",
    "public": true,
    "primary_key": false,
    "order": "asc"
  }
}
```

---

### 2. MEASURE Properties

A measure is an aggregation over a column.

**Required Fields**:
- `name`: string - Unique identifier within the cube
- `sql`: string - SQL expression for aggregation
- `type`: string - Aggregation type (count, sum, avg, min, max, count_distinct, etc.)

**Optional Fields**:
```json
{
  "name": "revenue",
  "sql": "{amount}",
  "type": "sum",
  
  "title": "Total Revenue",
  "description": "Sum of all order amounts",
  "public": true,
  "format": "currency",
  "meta": { "currency": "USD" },
  
  "filters": [
    { "sql": "{CUBE}.status = 'completed'" }
  ],
  
  "rolling_window": {
    "trailing": "1 month",
    "leading": "0 day",
    "offset": "end"
  },
  
  "multi_stage": false,
  
  "time_shift": [
    {
      "time_dimension": "created_at",
      "interval": "1 year",
      "type": "prior"
    }
  ],
  
  "drill_members": [
    "id",
    "status",
    "products.name"
  ]
}
```

**Semantic Term Example**:
```json
{
  "semantic_term_type": "Measure",
  "data_type": "sum",
  "nullable": true,
  
  "cube_properties": {
    "name": "total_revenue",
    "sql": "{amount}",
    "type": "sum",
    "title": "Total Revenue",
    "description": "Sum of order amounts",
    "public": true,
    "format": "currency",
    "filters": [
      { "sql": "{CUBE}.status = 'completed'" }
    ]
  }
}
```

---

### 3. TIME Properties

Time dimensions with temporal characteristics and granularities.

**Required Fields**:
- `name`: string - Unique identifier
- `sql`: string - SQL expression for date/time
- `type`: "time" - Must be "time"

**Optional Fields**:
```json
{
  "name": "created_at",
  "sql": "{CUBE}.created_at",
  "type": "time",
  
  "title": "Created At",
  "description": "Timestamp when record was created",
  "public": true,
  "order": "desc",
  
  "granularities": [
    {
      "name": "day",
      "sql": "DATE_TRUNC('day', {CUBE}.created_at)"
    },
    {
      "name": "month",
      "sql": "DATE_TRUNC('month', {CUBE}.created_at)"
    },
    {
      "name": "year",
      "sql": "DATE_TRUNC('year', {CUBE}.created_at)"
    },
    {
      "name": "quarter",
      "sql": "DATE_TRUNC('quarter', {CUBE}.created_at)"
    },
    {
      "name": "week",
      "sql": "DATE_TRUNC('week', {CUBE}.created_at)"
    },
    {
      "name": "hour",
      "sql": "DATE_TRUNC('hour', {CUBE}.created_at)"
    }
  ],
  
  "time_shift": [
    {
      "name": "prior_year",
      "type": "prior",
      "interval": "1 year"
    },
    {
      "name": "prior_month",
      "type": "prior",
      "interval": "1 month"
    }
  ]
}
```

**Semantic Term Example**:
```json
{
  "semantic_term_type": "Time",
  "data_type": "timestamp",
  "temporal": true,
  "nullable": false,
  
  "cube_properties": {
    "name": "created_at",
    "sql": "{CUBE}.created_at",
    "type": "time",
    "title": "Created At",
    "description": "Record creation timestamp",
    "order": "desc",
    "granularities": [
      {
        "name": "day",
        "interval": "1 day"
      },
      {
        "name": "month",
        "interval": "1 month"
      },
      {
        "name": "year",
        "interval": "1 year"
      }
    ]
  }
}
```

---

### 4. HIERARCHY Properties

A hierarchy groups dimensions together for drill-down analysis.

**Required Fields**:
- `name`: string - Unique identifier
- `levels`: string[] - Array of dimension names from least to most granular

**Optional Fields**:
```json
{
  "name": "location",
  "title": "User Location",
  "description": "Geographic hierarchy from country to city",
  "public": true,
  "levels": [
    "country",
    "state",
    "city"
  ]
}
```

**Semantic Term Example**:
```json
{
  "semantic_term_type": "Hierarchy",
  "hierarchy_name": "location",
  
  "cube_properties": {
    "name": "location",
    "title": "User Location",
    "description": "Geographic hierarchy: country → state → city",
    "public": true,
    "levels": [
      "country",
      "state",
      "city"
    ]
  },
  
  "hierarchy_metadata": {
    "type": "geographic",
    "granularity": ["country", "state", "city"],
    "drill_down_order": 0
  }
}
```

---

### 5. SEGMENT Properties

A segment is a pre-calculated filter or cohort definition.

**Required Fields**:
- `name`: string - Unique identifier
- `sql`: string - SQL condition/filter

**Optional Fields**:
```json
{
  "name": "premium_users",
  "sql": "{CUBE}.customer_type = 'premium'",
  
  "title": "Premium Users",
  "description": "Users with premium subscription",
  "public": true,
  "meta": { "team": "product" }
}
```

**Semantic Term Example**:
```json
{
  "semantic_term_type": "Segment",
  "data_type": "filter",
  "nullable": false,
  
  "cube_properties": {
    "name": "premium_users",
    "sql": "{CUBE}.customer_type = 'premium'",
    "title": "Premium Users",
    "description": "Customers with premium status",
    "public": true
  },
  
  "segment_metadata": {
    "type": "customer_segment",
    "audience": "product_team",
    "use_case": "premium_user_analytics"
  }
}
```

---

## Complete Semantic Term Property Schema

The `catalog_node.properties` JSONB field should follow this structure:

```json
{
  "semantic_term_id": "uuid",
  "semantic_term_type": "Dimension | Measure | Time | Hierarchy | Segment",
  "semantic_term_name": "string",
  
  "data_type": "string",
  "foreign_key": boolean,
  "nullable": boolean,
  "temporal": boolean,
  "status_flag": boolean,
  
  "cardinality": integer,
  "frequent_values": [string],
  "inferred_patterns": [string],
  
  "schema": "string",
  "table": "string",
  "source_column": "string",
  
  "cube_properties": {
    "name": "string (required)",
    "sql": "string (required for Dimension/Measure/Segment)",
    "type": "string (required)",
    
    "title": "string (optional)",
    "description": "string (optional)",
    "public": "boolean (optional, default: true)",
    "format": "string (optional)",
    "meta": "object (optional)",
    
    "case": {
      "when": [{
        "sql": "string",
        "label": "string | {sql: string}"
      }],
      "else": {"label": "string | {sql: string}"}
    },
    
    "order": "asc | desc (optional)",
    "primary_key": "boolean (optional, default: false)",
    "propagate_filters_to_sub_query": "boolean (optional)",
    "sub_query": "boolean (optional)",
    
    "filters": [{
      "sql": "string"
    }],
    
    "rolling_window": {
      "trailing": "string (e.g., '1 month')",
      "leading": "string (optional)",
      "offset": "start | end (default: end)"
    },
    
    "multi_stage": "boolean (optional)",
    
    "time_shift": [{
      "time_dimension": "string (optional)",
      "type": "prior | next (optional)",
      "interval": "string (e.g., '1 year')",
      "name": "string (optional)"
    }],
    
    "granularities": [{
      "name": "string",
      "interval": "string (required)",
      "offset": "string (optional)",
      "origin": "ISO 8601 date (optional)",
      "title": "string (optional)"
    }],
    
    "drill_members": ["string"],
    
    "levels": ["string"]
  },
  
  "hierarchy_metadata": {
    "type": "string",
    "granularity": ["string"],
    "drill_down_order": "integer"
  },
  
  "segment_metadata": {
    "type": "string",
    "audience": "string",
    "use_case": "string"
  },
  
  "business_term": "string",
  "domain_hierarchy": ["string"],
  "confidence": "number (0-1)",
  "reasoning": "string",
  
  "created_by": "string",
  "last_modified_by": "string",
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

---

## Implementation Guidelines

### 1. Semantic Term Wizard Enhancement

When `SuggestEnrichment()` and `ApplyEnrichment()` create semantic terms, they should populate:

```go
properties := map[string]interface{}{
  "semantic_term_type": "Dimension", // or Measure, Time, Hierarchy, Segment
  "semantic_term_name": proposal.SemanticTermName,
  "data_type": proposal.SemanticTermType,
  
  // Inferred properties
  "foreign_key": isForeignKey,
  "nullable": isNullable,
  "temporal": isTemporal,
  "status_flag": isStatusFlag,
  
  // Data characteristics
  "cardinality": column.Cardinality,
  "frequent_values": column.FrequentValues,
  "inferred_patterns": column.InferredPatterns,
  
  // Source context
  "schema": column.Schema,
  "table": column.Table,
  "source_column": column.Column,
  
  // Cube.dev properties
  "cube_properties": {
    "name": columnName,
    "sql": fmt.Sprintf("{CUBE}.%s", columnName),
    "type": determineDataType(column),
    "title": generateTitle(columnName),
    "description": proposal.Reasoning,
    "public": true,
  },
  
  // Metadata
  "business_term": proposal.BusinessTermName,
  "domain_hierarchy": proposal.DomainHierarchy,
  "confidence": proposal.Confidence,
}
```

### 2. Catalog Node Type Definition

For each semantic term type, define a catalog_node_type:

```sql
INSERT INTO catalog_node_type (catalog_type_name, description, config)
VALUES 
  ('semantic_term_dimension', 'Cube.js Dimension', '{
    "properties_schema": {
      "name": {"type": "string", "required": true},
      "sql": {"type": "string", "required": true},
      "type": {"type": "string", "enum": ["string", "number", "boolean", "time"], "required": true},
      "title": {"type": "string"},
      "description": {"type": "string"},
      "public": {"type": "boolean", "default": true},
      "primary_key": {"type": "boolean", "default": false}
    }
  }'),
  
  ('semantic_term_measure', 'Cube.js Measure', '{
    "properties_schema": {
      "name": {"type": "string", "required": true},
      "sql": {"type": "string", "required": true},
      "type": {"type": "string", "enum": ["count", "sum", "avg", "min", "max"], "required": true},
      "title": {"type": "string"},
      "description": {"type": "string"},
      "format": {"type": "string"}
    }
  }'),
  
  ('semantic_term_time', 'Cube.js Time Dimension', '{
    "properties_schema": {
      "name": {"type": "string", "required": true},
      "sql": {"type": "string", "required": true},
      "type": {"const": "time", "required": true},
      "granularities": {"type": "array"}
    }
  }'),
  
  ('semantic_term_hierarchy', 'Cube.js Hierarchy', '{
    "properties_schema": {
      "name": {"type": "string", "required": true},
      "levels": {"type": "array", "required": true},
      "title": {"type": "string"}
    }
  }'),
  
  ('semantic_term_segment', 'Cube.js Segment', '{
    "properties_schema": {
      "name": {"type": "string", "required": true},
      "sql": {"type": "string", "required": true},
      "title": {"type": "string"},
      "description": {"type": "string"}
    }
  }');
```

### 3. Catalog Edge Type Definition

Define relationships between semantic term types:

```sql
INSERT INTO catalog_edge_types (edge_type_name, description, source_node_type_id, target_node_type_id)
SELECT 
  'hierarchy_contains_dimension',
  'Hierarchy contains dimensions for drill-down',
  (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term_hierarchy'),
  (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term_dimension')
UNION ALL
SELECT
  'measure_uses_dimension',
  'Measure aggregates over dimensions',
  (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term_measure'),
  (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term_dimension')
UNION ALL
SELECT
  'segment_filters_measure',
  'Segment provides pre-calculated filters',
  (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term_segment'),
  (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term_measure');
```

---

## Example: Complete Semantic Term

### Input Column
```
Schema: sales
Table: orders
Column: user_id
Type: bigint
Cardinality: 150,000
```

### Generated Semantic Term
```json
{
  "id": "term-uuid-123",
  "node_name": "USER_ID",
  "node_type_id": "semantic-term-dimension-type-uuid",
  "properties": {
    "semantic_term_type": "Dimension",
    "semantic_term_name": "USER_ID",
    "data_type": "number",
    "foreign_key": true,
    "nullable": false,
    "cardinality": 150000,
    "schema": "sales",
    "table": "orders",
    "source_column": "user_id",
    
    "cube_properties": {
      "name": "user_id",
      "sql": "{CUBE}.user_id",
      "type": "number",
      "title": "User ID",
      "description": "Unique identifier linking to users table",
      "public": true,
      "primary_key": false,
      "order": "asc"
    },
    
    "business_term": "User Identifier",
    "domain_hierarchy": ["Sales", "Orders"],
    "confidence": 0.95,
    "reasoning": "Foreign key with high cardinality indicates user dimension"
  }
}
```

---

## Migration Path

1. ✅ Update semantic term wizard to populate cube_properties
2. ✅ Add catalog_node_type definitions for each semantic term type
3. ✅ Add validation to ensure required fields are present
4. ✅ Update glossary handler to expose cube properties in API responses
5. ✅ Create views for Cube.js YAML/JSON generation
6. ✅ Add tests for property validation

---

## Validation Rules

### For Dimensions
- `name`: Must be unique within cube, match naming conventions
- `sql`: Must be valid SQL expression
- `type`: Must be one of (string, number, boolean, time)
- If `primary_key: true` → `public` defaults to false (unless explicitly true)

### For Measures
- `name`: Must be unique within cube
- `sql`: Depends on type (count skips sql, others require aggregate expression)
- `type`: Must be one of (count, sum, avg, min, max, count_distinct, count_distinct_approx, number, string, time, boolean)

### For Time
- `type`: Must be exactly "time"
- `sql`: Must reference timestamp/date column
- `granularities`: If present, must include interval parameter for each

### For Hierarchies
- `levels`: Must reference existing dimensions
- Order matters: least to most granular

### For Segments
- `sql`: Must be valid WHERE clause condition
- `name`: Must be unique within cube

---

## API Integration

### GET /api/glossary/semantic-terms/{id}
```json
{
  "id": "term-uuid",
  "name": "USER_ID",
  "type": "Dimension",
  "properties": { /* complete properties as above */ },
  "cube_definition": { /* cube_properties only */ }
}
```

### GET /api/glossary/semantic-terms/cube-export
```json
{
  "cubes": [
    {
      "name": "orders",
      "dimensions": [
        {
          "name": "user_id",
          "sql": "{CUBE}.user_id",
          "type": "number"
        }
      ],
      "measures": [
        {
          "name": "revenue",
          "sql": "{amount}",
          "type": "sum"
        }
      ],
      "hierarchies": [
        {
          "name": "location",
          "levels": ["country", "state", "city"]
        }
      ]
    }
  ]
}
```

---

**Status**: ✅ **SPECIFICATION COMPLETE**

This specification aligns the semantic term catalog with Cube.dev's property definitions, enabling seamless integration for BI tool configuration and automated cube generation.

