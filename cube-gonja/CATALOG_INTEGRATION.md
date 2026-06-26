# Catalog Integration

This document describes the automatic catalog update functionality that has been added to the cube-gonja service.

## Overview

The catalog integration automatically updates the `catalog_node` and `catalog_edge` tables in the database whenever semantic models are generated or modified. This ensures that the data catalog stays synchronized with the current state of your semantic models.

## How It Works

### Automatic Updates

The catalog is automatically updated when:

1. **Single Model Rendering** (`POST /render`): When a single template is rendered
2. **All Models Rendering** (`POST /render-all`): When all templates are rendered
3. **Manual Catalog Update** (`POST /update-catalog`): Manual trigger to update catalog from existing models

### What Gets Updated

#### Catalog Nodes
- **Semantic Models**: Created for each Cube.js cube with metadata like SQL table, data source, etc.
- **Semantic Columns**: Created for each measure and dimension within the cubes
- Properties include data types, SQL expressions, primary key flags, etc.

#### Catalog Edges
- **Mapped To**: Links semantic columns to physical database columns
- **Foreign Key**: Links tables via foreign key relationships
- **Has Semantic**: Links business terms to semantic terms
- **Member Of**: Links semantic terms to semantic columns

## Configuration

### Database Configuration

Set the following environment variables to enable catalog updates:

```bash
# Database connection
DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_NAME=semlayer
DATABASE_USER=postgres
DATABASE_PASSWORD=your_password
DATABASE_SSL_MODE=disable

# Or use a complete DSN
DATABASE_DSN=postgres://user:password@localhost:5432/semlayer?sslmode=disable
```

### Required Tables

The following tables must exist in your database (they are defined in `schema.sql`):

- `public.catalog_node`
- `public.catalog_node_type`
- `public.catalog_edge`
- `public.catalog_edge_types`
- `public_temp.catalog_node`
- `public_temp.catalog_edge`

## API Endpoints

### POST /update-catalog

Manually triggers a catalog update from all models in the output directory.

**Response:**
```json
{
  "status": "Catalog updated successfully"
}
```

## Node Types

The system automatically creates the following node types:

- `schema`: Database schema
- `table`: Database table
- `column`: Database column
- `semantic_model`: Cube.js semantic model/cube
- `semantic_column`: Measure or dimension in a semantic model
- `business_term`: Business terminology
- `semantic_term`: Semantic terminology

## Edge Types

The system creates the following relationship types:

- `has_semantic`: Business term → Semantic term
- `member of`: Semantic term → Semantic column
- `mapped to`: Semantic column → Physical column
- `foreign_key`: Table → Table (foreign key relationship)

## Sample Data

The system includes sample data for testing:

## Business Term Features

### Enhanced Business Term Structure

Business terms now support additional metadata fields:

```yaml
business_terms:
  - id: "BT-CUST-001"
    name: "Customer"
    description: "A customer entity representing a business client"
    category: "Business Entity"
    sub_category: "Core Entities"
    owner: "Data Governance Team"
    steward: "John Smith"
    status: "approved"  # draft, approved, deprecated, archived
    version: "1.0.0"
    tags: ["customer", "entity", "core"]
    parent_id: "BT-ENTITY-000"  # For hierarchical relationships
```

### Business Term Validation

The system validates business terms for:
- Required fields (ID, name, category)
- ID format (must start with "BT-")
- Status values
- Parent term existence

### Business Term Inheritance

If measures or dimensions don't have explicit business terms, they can inherit from their parent cube:

```yaml
cubes:
  - name: customers
    business_terms:
      - id: "BT-CUST-001"
        name: "Customer"
        category: "Business Entity"
    
    measures:
      - name: count
        # Inherits BT-CUST-001 from cube if no explicit terms
```

### Business Term Relationships

The system creates hierarchical relationships between business terms:

- **Parent-Child**: Links business terms in inheritance hierarchies
- **Semantic Mapping**: Links business terms to semantic elements
- **Member Of**: Links semantic elements to their business terms

### API Endpoints

#### GET /business-terms

Search and filter business terms:

```bash
# Search by query
GET /business-terms?query=customer

# Filter by category
GET /business-terms?category=Financial

# Filter by status
GET /business-terms?status=approved

# Filter by tags
GET /business-terms?tags=customer,revenue

# Pagination
GET /business-terms?limit=20&offset=40
```

**Response:**
```json
{
  "business_terms": [
    {
      "id": "BT-CUST-001",
      "name": "Customer",
      "description": "A customer entity",
      "category": "Business Entity",
      "sub_category": "Core Entities",
      "owner": "Data Governance Team",
      "steward": "John Smith",
      "status": "approved",
      "version": "1.0.0",
      "tags": ["customer", "entity"],
      "parent_id": null
    }
  ],
  "total": 1,
  "limit": 50,
  "offset": 0
}
```

#### POST /business-terms/validate

Validate business terms before using them:

```bash
POST /business-terms/validate
Content-Type: application/json

{
  "business_terms": [
    {
      "id": "BT-NEW-001",
      "name": "New Business Term",
      "category": "Test",
      "status": "draft"
    }
  ]
}
```

**Response:**
```json
{
  "valid": true,
  "errors": [],
  "warnings": [
    "Business term BT-NEW-001: version not specified",
    "Business term BT-NEW-001: no tags specified"
  ]
}
```

## Error Handling

- Catalog updates are logged but don't fail the rendering process
- Database connection failures are logged as warnings
- Missing node types are handled gracefully
- Failed catalog updates don't interrupt model generation

## Monitoring

Check the service logs for catalog update activity:

```
Database connection established
Warning: Failed to update catalog for rendered model example.yml: <error>
```

## Future Enhancements

- Business term governance workflows and approval processes
- Business term impact analysis for changes
- Integration with external business glossary systems
- Business term usage analytics and reporting
- Automated business term suggestions based on data patterns</content>
<parameter name="filePath">/Users/eganpj/GitHub/semlayer/cube-gonja/CATALOG_INTEGRATION.md
