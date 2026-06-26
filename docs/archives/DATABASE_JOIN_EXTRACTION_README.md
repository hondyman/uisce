# Database Join Extraction and Cube Generation

This document describes the enhanced join extraction and automatic cube generation functionality added to the semantic layer system.

## Overview

The system now includes comprehensive database join relationship extraction and automatic Cube.js-compliant model generation based on database foreign key relationships and metadata.

## Backend Components

### 1. Join Extractor (`backend/internal/cube/join_extractor.go`)

A comprehensive service that extracts database foreign key relationships and generates Cube.js-compatible structures:

#### Key Features:
- **Foreign Key Discovery**: Extracts join relationships from `catalog_edge_vw` and `catalog_node_vw` tables
- **Join Path Building**: Constructs join paths between tables using graph traversal algorithms
- **Cube Generation**: Automatically creates complete Cube.js definitions with dimensions, measures, and joins
- **Relationship Mapping**: Maps database relationship types to Cube.js relationship types

#### Core Classes:
- `DatabaseJoinExtractor`: Main service class for join extraction
- `JoinSuggestion`: Represents a single join relationship between tables
- `TableColumn`: Represents database table column metadata
- `GeneratedCube`: Complete Cube.js definition structure

### 2. API Endpoints (`backend/internal/api/api.go`)

New REST endpoints for join extraction and cube generation:

#### Endpoints:

##### `GET /api/fabric/joins/{datasourceId}`
Extracts all join suggestions from database foreign key relationships.

**Response:**
```json
{
  "joins": [
    {
      "source_table": "orders",
      "target_table": "customers",
      "source_column": "customer_id",
      "target_column": "id",
      "relationship": "many_to_one",
      "join_sql": "{CUBE.customer_id} = {customers.id}",
      "description": "Join orders to customers via customer_id"
    }
  ],
  "count": 5,
  "datasource_id": "uuid-here"
}
```

##### `GET /api/fabric/joins/{datasourceId}/table/{tableName}`
Gets join definitions for a specific table.

**Response:**
```json
{
  "table_name": "orders",
  "joins": {
    "customers": {
      "relationship": "many_to_one",
      "sql": "{CUBE.customer_id} = {customers.id}"
    },
    "products": {
      "relationship": "many_to_many",
      "sql": "{CUBE.product_id} = {products.id}"
    }
  },
  "count": 2,
  "datasource_id": "uuid-here"
}
```

##### `POST /api/fabric/cubes/generate-from-table`
Generates a complete Cube.js definition from a database table.

**Request:**
```json
{
  "datasource_id": "uuid-here",
  "table_name": "orders"
}
```

**Response:**
```json
{
  "cube": {
    "name": "orders",
    "sql_table": "orders",
    "title": "Orders",
    "description": "Auto-generated cube for orders table",
    "public": true,
    "dimensions": {
      "id": {
        "sql": "id",
        "type": "number",
        "title": "Id",
        "primary_key": true,
        "public": true
      },
      "customer_id": {
        "sql": "customer_id",
        "type": "number",
        "title": "Customer Id",
        "public": true
      }
    },
    "measures": {
      "count": {
        "type": "count",
        "sql": "id",
        "title": "Count"
      },
      "total_amount": {
        "type": "sum",
        "sql": "amount",
        "title": "Total Amount",
        "format": "currency"
      }
    },
    "joins": {
      "customers": {
        "relationship": "many_to_one",
        "sql": "{CUBE.customer_id} = {customers.id}"
      }
    }
  },
  "table_name": "orders",
  "datasource_id": "uuid-here"
}
```

## Frontend Components

### 1. Join Extraction Service (`frontend/src/services/joinExtractionService.ts`)

A TypeScript service that interfaces with the backend join extraction APIs:

#### Key Features:
- **API Integration**: Seamless communication with backend join endpoints
- **Join Path Building**: Client-side join path discovery using BFS algorithm
- **Relationship Validation**: Validates join relationships between tables
- **Cube Conversion**: Converts database relationships to Cube.js format

#### Core Methods:
- `extractJoinSuggestions()`: Fetch all available joins for a datasource
- `getTableJoinDefinitions()`: Get joins for a specific table
- `generateCubeFromTable()`: Generate complete cube from table metadata
- `buildJoinPath()`: Find shortest path between two tables
- `validateJoinRelationship()`: Check if join exists between tables

### 2. Database Join Explorer Component (`frontend/src/components/joins/DatabaseJoinExplorer.tsx`)

A comprehensive React component for visual join exploration and cube generation:

#### Features:
- **Visual Join Explorer**: Interactive interface for browsing database relationships
- **Join Path Visualization**: Graphical representation of join paths between tables
- **Auto-Generate Cubes**: One-click cube generation from table metadata
- **Join Selection**: Multi-select interface for choosing specific joins
- **Relationship Badges**: Color-coded badges for different relationship types

#### Component Structure:
```tsx
<DatabaseJoinExplorer
  datasourceId="uuid-here"
  selectedTable="orders"
  onJoinSelect={(join) => console.log('Selected:', join)}
  onCubeGenerate={(cube) => console.log('Generated:', cube)}
/>
```

### 3. Enhanced Model Workspace (`frontend/src/components/model/ModelWorkspace.tsx`)

Updated workspace with integrated join exploration:

#### New Features:
- **Join Explorer Tab**: Dedicated tab for join relationship exploration
- **Table-based Navigation**: Automatic table name extraction from model keys
- **Integrated Workflow**: Seamless transition from join exploration to model creation

## Integration with Existing System

### Cube.js Compliance

The system ensures full compatibility with Cube.js specifications:

#### Relationship Types:
- `one_to_one`: Direct 1:1 relationships
- `one_to_many`: Parent-child relationships  
- `many_to_one`: Child-parent relationships (most common)
- `many_to_many`: Junction table relationships

#### Join SQL Format:
All generated join SQL follows Cube.js syntax:
```javascript
sql: `{CUBE.foreign_key_column} = {target_table.primary_key_column}`
```

#### Dimension and Measure Generation:
- **Dimensions**: Created for all columns with appropriate types
- **Measures**: Automatically generated for numeric columns (sum, count, avg)
- **Primary Keys**: Properly marked with `primary_key: true`
- **Data Types**: Mapped from database types to Cube.js types (string, number, time, boolean)

### Database Schema Requirements

The system relies on the existing catalog metadata structure:

#### Required Tables:
- `catalog_node_vw`: Contains table and column metadata
- `catalog_edge_vw`: Contains foreign key relationships

#### Expected Columns:
```sql
-- catalog_node_vw
table_name, column_name, data_type, description, 
is_nullable, is_primary_key, catalog_type_name

-- catalog_edge_vw  
subject_node_id, object_node_id, predicate, relationship_type
```

## Usage Examples

### 1. Basic Join Extraction

```typescript
import { joinExtractionService } from '../services/joinExtractionService';

// Get all joins for a datasource
const joins = await joinExtractionService.extractJoinSuggestions('datasource-uuid');
console.log('Available joins:', joins.joins);

// Get joins for specific table
const tableJoins = await joinExtractionService.getTableJoinDefinitions('datasource-uuid', 'orders');
console.log('Orders table joins:', tableJoins.joins);
```

### 2. Join Path Discovery

```typescript
// Find path from orders to customers
const path = await joinExtractionService.buildJoinPath('datasource-uuid', 'orders', 'customers');
console.log('Join path:', path); // ['orders', 'customers']

// Generate SQL for the path
const sqlStatements = await joinExtractionService.generateJoinSQL('datasource-uuid', path);
console.log('Join SQL:', sqlStatements);
```

### 3. Automatic Cube Generation

```typescript
// Generate complete cube from table
const cubeResponse = await joinExtractionService.generateCubeFromTable('datasource-uuid', 'orders');
console.log('Generated cube:', cubeResponse.cube);

// Use in model creation
const newModel = {
  ...cubeResponse.cube,
  // Add custom dimensions or measures
  measures: {
    ...cubeResponse.cube.measures,
    custom_measure: {
      type: 'count_distinct',
      sql: 'customer_id',
      title: 'Unique Customers'
    }
  }
};
```

### 4. React Component Usage

```tsx
import React from 'react';
import DatabaseJoinExplorer from '../components/joins/DatabaseJoinExplorer';

const ModelBuilder = () => {
  const handleJoinSelect = (join) => {
    // Add join to model definition
    console.log('Selected join:', join);
  };

  const handleCubeGenerate = (cube) => {
    // Use generated cube as starting point
    console.log('Generated cube:', cube);
  };

  return (
    <DatabaseJoinExplorer
      datasourceId={datasourceId}
      selectedTable={selectedTable}
      onJoinSelect={handleJoinSelect}
      onCubeGenerate={handleCubeGenerate}
    />
  );
};
```

## Error Handling

The system includes comprehensive error handling:

### Backend Errors:
- Invalid datasource ID format
- Missing database metadata
- SQL query failures
- Join path not found

### Frontend Errors:
- Network request failures
- Invalid response formats
- Missing required parameters
- Component state management errors

### Error Response Format:
```json
{
  "error": "error_code",
  "message": "Human readable error message",
  "details": "Additional error context"
}
```

## Performance Considerations

### Backend Optimizations:
- **Efficient Queries**: Optimized SQL queries with proper indexes
- **Caching**: In-memory caching of frequently accessed join relationships
- **Batch Processing**: Bulk operations for multiple table processing

### Frontend Optimizations:
- **Lazy Loading**: Components loaded on-demand
- **Memoization**: Cached API responses to reduce redundant requests
- **Virtual Scrolling**: Efficient rendering of large join lists

## Future Enhancements

### Planned Features:
1. **Advanced Join Types**: Support for complex join conditions and computed joins
2. **Machine Learning**: AI-powered join suggestion based on column names and data patterns
3. **Visual Graph**: Interactive graph visualization of table relationships
4. **Join Optimization**: Automatic join path optimization for performance
5. **Custom Relationships**: User-defined relationships not present in foreign keys
6. **Batch Cube Generation**: Generate multiple cubes simultaneously
7. **Export/Import**: Export join configurations and cube definitions

### Integration Roadmap:
1. **Phase 1**: Core join extraction and cube generation (✅ Complete)
2. **Phase 2**: Advanced UI components and visualizations
3. **Phase 3**: Machine learning enhancements
4. **Phase 4**: Enterprise features and scalability improvements

## Testing

### Backend Tests:
```bash
cd backend
go test ./internal/cube/
```

### Frontend Tests:
```bash
cd frontend  
npm test -- --testPathPattern=joinExtraction
```

### Integration Tests:
```bash
# Test full join extraction workflow
curl -X GET "http://localhost:8000/api/fabric/joins/test-datasource-id"
curl -X POST "http://localhost:8000/api/fabric/cubes/generate-from-table" \
  -H "Content-Type: application/json" \
  -d '{"datasource_id":"test-datasource-id","table_name":"orders"}'
```

This enhanced join extraction and cube generation system provides a powerful foundation for automatic semantic model creation, significantly reducing the manual effort required to build Cube.js models while ensuring consistency and best practices.
