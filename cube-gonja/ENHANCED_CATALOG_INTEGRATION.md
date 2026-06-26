# Enhanced Catalog Integration

This document describes the enhanced catalog integration that automatically detects and catalogs model relationships, views, and dependencies.

## Overview

The enhanced catalog service now automatically detects:
- **Model Relationships**: Joins between semantic models
- **SQL References**: References to other cubes in measures and dimensions
- **View Detection**: Identifies semantic views vs. regular models
- **Dependency Tracking**: Creates edges between related models

## New Node Types

### semantic_view
Represents a semantic view that combines multiple models through joins or references.

**Detection Criteria:**
- Contains joins with other cubes
- SQL table contains keywords like 'view', 'union', 'join'
- Measures/dimensions reference other cubes

## New Edge Types

### joins
Links semantic models that have explicit join relationships.

**Properties:**
```json
{
  "relationship": "many_to_one|one_to_many|many_to_many",
  "sql": "JOIN clause",
  "join_type": "join"
}
```

### references
Links semantic models where one references another in SQL expressions.

**Properties:**
```json
{
  "element_name": "measure_or_dimension_name",
  "element_type": "measure|dimension",
  "reference_type": "sql_reference"
}
```

# Enhanced Catalog Integration

This document describes the enhanced catalog integration that automatically detects and catalogs model relationships, views, and business terms.

## Overview

The enhanced catalog service now automatically detects:
- **Model Relationships**: Joins between semantic models
- **SQL References**: References to other cubes in measures and dimensions
- **View Detection**: Identifies semantic views vs. regular models
- **Business Term Integration**: Links semantic models to business glossary terms
- **Dependency Tracking**: Creates edges between related models

## New Node Types

### semantic_view
Represents a semantic view that combines multiple models through joins or references.

**Detection Criteria:**
- Contains joins with other cubes
- SQL table contains keywords like 'view', 'union', 'join'
- Measures/dimensions reference other cubes

### business_term
Represents a business term or concept from the business glossary.

**Properties:**
```json
{
  "business_term_id": "BT-CUST-001",
  "name": "Customer",
  "description": "A customer entity representing a business client",
  "category": "Business Entity"
}
```

## New Edge Types

### joins
Links semantic models that have explicit join relationships.

**Properties:**
```json
{
  "relationship": "many_to_one|one_to_many|many_to_many",
  "sql": "JOIN clause",
  "join_type": "join"
}
```

### references
Links semantic models where one references another in SQL expressions.

**Properties:**
```json
{
  "element_name": "measure_or_dimension_name",
  "element_type": "measure|dimension",
  "reference_type": "sql_reference"
}
```

### has_semantic
Links business terms to semantic models (model-level business terms).

**Properties:**
```json
{
  "business_term_id": "BT-CUST-001",
  "relationship_type": "semantic_mapping",
  "category": "Business Entity"
}
```

### member_of
Links business terms to individual semantic columns (measure/dimension level).

**Properties:**
```json
{
  "business_term_id": "BT-CUST-001",
  "element_type": "measure|dimension",
  "element_name": "column_name",
  "relationship_type": "semantic_mapping",
  "category": "Business Entity"
}
```

## Business Term Integration

### Model-Level Business Terms
Business terms can be associated with entire semantic models:

```yaml
cubes:
  - name: customers
    title: Customers
    description: "Customer information and metrics"
    business_terms:
      - id: "BT-CUST-001"
        name: "Customer"
        description: "A customer entity representing a business client"
        category: "Business Entity"
      - id: "BT-REVENUE-002"
        name: "Revenue"
        description: "Financial revenue metrics"
        category: "Financial"
```

### Column-Level Business Terms
Business terms can be associated with individual measures and dimensions:

```yaml
measures:
  - name: count
    type: count
    description: "Total number of customers"
    business_terms:
      - id: "BT-CUST-001"
        name: "Customer Count"
        description: "Count of customer entities"
        category: "Business Entity"
```

## Enhanced Metadata

### Semantic Model Properties
Models now include business term information in their metadata:

```json
{
  "sql_table": "public.customers",
  "data_source": "default",
  "is_view": false,
  "description": "Customer information and metrics",
  "business_term_ids": ["BT-CUST-001", "BT-REVENUE-002"],
  "business_terms": ["Customer", "Revenue"]
}
```

### Semantic Column Properties
Measures and dimensions include business term information:

```json
{
  "type": "measure",
  "measure_type": "count",
  "sql": "COUNT(*)",
  "description": "Total number of customers",
  "business_term_ids": ["BT-CUST-001"],
  "business_terms": ["Customer Count"]
}
```

## Example Usage

### Complete Model with Business Terms
```yaml
cubes:
  - name: customers
    sql_table: public.customers
    data_source: default
    title: Customers
    description: "Customer information and metrics"

    business_terms:
      - id: "BT-CUST-001"
        name: "Customer"
        description: "A customer entity representing a business client"
        category: "Business Entity"

    joins:
      - name: orders
        sql: "LEFT JOIN orders ON customers.customer_id = orders.customer_id"
        relationship: many_to_one

    measures:
      - name: count
        type: count
        description: "Total number of customers"
        business_terms:
          - id: "BT-CUST-001"
            name: "Customer Count"
            description: "Count of customer entities"
            category: "Business Entity"

    dimensions:
      - name: id
        sql: id
        type: number
        primary_key: true
        description: "Unique customer identifier"
        business_terms:
          - id: "BT-CUST-001"
            name: "Customer ID"
            description: "Unique identifier for customers"
            category: "Business Entity"
```

## Catalog Structure

### Nodes Created
- **semantic_model**: Regular cubes without joins/references
- **semantic_view**: Cubes with joins or complex relationships
- **semantic_column**: Measures and dimensions within models
- **business_term**: Business glossary terms

### Edges Created
- **joins**: Explicit join relationships between models
- **references**: SQL references between models
- **has_semantic**: Business terms linked to models
- **member_of**: Business terms linked to columns

## Benefits

1. **Business Glossary Integration**: Automatic linking of semantic models to business terms
2. **Enhanced Metadata**: Rich business context for all semantic elements
3. **Impact Analysis**: Better understanding of business impact when models change
4. **Data Governance**: Clear traceability from business terms to physical data
5. **Search & Discovery**: Find semantic models by business terminology

## Implementation Details

### Business Term Processing
```go
func (s *CatalogService) processCubeBusinessTerms(cube Cube, modelNodeID string) error {
    for _, businessTerm := range cube.BusinessTerms {
        // Create business term node
        businessTermNodeID, err := s.upsertBusinessTermNode(businessTerm)
        // Create edge between model and business term
        // ...
    }
}
```

### Metadata Enhancement
```go
properties := map[string]interface{}{
    "business_term_ids": s.extractBusinessTermIDs(cube.BusinessTerms),
    "business_terms": s.extractBusinessTermNames(cube.BusinessTerms),
    // ... other properties
}
```

## Future Enhancements

1. **Business Term Validation**: Ensure business terms exist in glossary
2. **Term Inheritance**: Child models inherit parent business terms
3. **Term Mapping Rules**: Automated business term assignment based on patterns
4. **Term Versioning**: Track changes in business term definitions
5. **Term Relationships**: Link related business terms together

## Example Usage

### Model with Joins
```yaml
cubes:
  - name: customers
    sql_table: public.customers
    data_source: default
    title: Customers

    joins:
      - name: orders
        sql: "LEFT JOIN orders ON customers.customer_id = orders.customer_id"
        relationship: many_to_one

    measures:
      - name: total_orders
        type: sum
        sql: orders.order_count
```

This creates:
1. A `semantic_view` node for `customers` (detected as view due to joins)
2. A `joins` edge from `customers` to `orders`
3. A `references` edge for the `total_orders` measure

### Model with References
```yaml
cubes:
  - name: orders
    sql_table: public.orders
    data_source: default
    title: Orders

    measures:
      - name: customer_count
        type: count_distinct
        sql: customers.customer_id
```

This creates:
1. A `references` edge from `orders` to `customers` for the `customer_count` measure

## Catalog Structure

### Nodes Created
- **semantic_model**: Regular cubes without joins/references
- **semantic_view**: Cubes with joins or complex relationships
- **semantic_column**: Measures and dimensions within models

### Edges Created
- **joins**: Explicit join relationships between models
- **references**: SQL references between models
- **extends**: Inheritance relationships (future enhancement)

## Benefits

1. **Automatic Discovery**: No manual configuration needed
2. **Dependency Tracking**: Clear visibility of model relationships
3. **Impact Analysis**: Easy to see what breaks when models change
4. **View Classification**: Distinguishes between simple models and complex views
5. **Lineage Tracking**: Complete data lineage from source to consumption

## Implementation Details

### Detection Logic

#### View Detection
```go
func (s *CatalogService) isCubeAView(cube Cube) bool {
    // Has joins
    if len(cube.Joins) > 0 {
        return true
    }

    // SQL contains view indicators
    if strings.Contains(sqlTable, "view|union|join") {
        return true
    }

    // References other cubes in SQL
    // ... detection logic
}
```

#### Reference Extraction
```go
func (s *CatalogService) extractReferencedCubes(sql string) []string {
    // Parse SQL for cube.column patterns
    // Return list of referenced cube names
}
```

### Edge Creation
Edges are created with conflict resolution to handle updates:
```sql
ON CONFLICT (tenant_datasource_id, source_node_id, edge_type_id, target_node_id)
DO UPDATE SET properties = EXCLUDED.properties, updated_at = EXCLUDED.updated_at
```

## Future Enhancements

1. **Advanced SQL Parsing**: More sophisticated SQL analysis for complex expressions
2. **Circular Dependency Detection**: Warn about circular references
3. **Impact Propagation**: Calculate downstream impacts of model changes
4. **Business Term Mapping**: Link semantic models to business glossary terms
5. **Versioning**: Track changes in model relationships over time</content>
<parameter name="filePath">/Users/eganpj/GitHub/semlayer/cube-gonja/ENHANCED_CATALOG_INTEGRATION.md
