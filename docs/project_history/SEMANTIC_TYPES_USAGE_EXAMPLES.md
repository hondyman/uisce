# Semantic Types - Practical Usage Examples

This document provides real-world examples of how to use the semantic_types lookup table in your Fabric Builder platform.

## Backend Examples (Go)

### 1. Using the Type Constants

```go
package handlers

import (
    "github.com/hondyman/semlayer/backend/models"
)

func createDimensionNode(nodeID string, tenantID string) {
    // Type-safe reference to semantic type
    semanticType := models.DimensionStringCurrency
    
    if models.IsDimension(semanticType) {
        log.Println("Creating dimension node with currency format")
    }
    
    // Get metadata for display
    metadata := models.GetMetadata(semanticType)
    if metadata != nil {
        log.Printf("Semantic Type: %s, Data Type: %s, Format: %s\n",
            metadata.SemanticType, metadata.DataType, metadata.Format)
    }
}
```

### 2. Querying Nodes by Semantic Type

```go
package repositories

import (
    "database/sql"
    "github.com/hondyman/semlayer/backend/models"
)

func GetDimensionsByFormat(db *sql.DB, tenantID string, format models.Format) ([]models.CatalogNode, error) {
    query := `
        SELECT id, node_name, node_type, properties, created_at
        FROM catalog_node
        WHERE tenant_id = $1
          AND properties->>'semantic_type' LIKE 'dimension_%'
          AND properties->>'semantic_type' LIKE $2
        ORDER BY created_at DESC
    `
    
    rows, err := db.Query(query, tenantID, "%"+string(format))
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var nodes []models.CatalogNode
    for rows.Next() {
        var node models.CatalogNode
        if err := rows.Scan(&node.ID, &node.NodeName, &node.NodeType, 
            &node.Properties, &node.CreatedAt); err != nil {
            return nil, err
        }
        nodes = append(nodes, node)
    }
    return nodes, nil
}
```

### 3. API Endpoint - Get Semantic Types

```go
package httpapi

import (
    "net/http"
    "database/sql"
    "encoding/json"
)

func handleGetSemanticTypes(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        tenantID := r.URL.Query().Get("tenant_id")
        if tenantID == "" {
            http.Error(w, "tenant_id required", http.StatusBadRequest)
            return
        }
        
        query := `
            SELECT lv.id, lv.value, lv.label, lv.metadata
            FROM lookup_values lv
            WHERE lv.lookup_id = (
                SELECT id FROM lookups WHERE name = 'semantic_types' AND tenant_id = $1
            )
            ORDER BY 
                lv.metadata->>'semantic_type',
                lv.metadata->>'data_type',
                lv.metadata->>'format'
        `
        
        rows, err := db.Query(query, tenantID)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        defer rows.Close()
        
        var values []map[string]interface{}
        for rows.Next() {
            var id, value, label string
            var metadata sql.NullString
            if err := rows.Scan(&id, &value, &label, &metadata); err != nil {
                http.Error(w, err.Error(), http.StatusInternalServerError)
                return
            }
            
            var meta map[string]interface{}
            if metadata.Valid {
                json.Unmarshal([]byte(metadata.String), &meta)
            }
            
            values = append(values, map[string]interface{}{
                "id":       id,
                "value":    value,
                "label":    label,
                "metadata": meta,
            })
        }
        
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(values)
    }
}
```

### 4. Filtering Measures by Category

```go
func GetMeasuresByCategory(db *sql.DB, tenantID string) (map[string][]models.SemanticTypeLookupValue, error) {
    query := `
        SELECT lv.id, lv.lookup_id, lv.tenant_id, lv.value, lv.label, lv.metadata
        FROM lookup_values lv
        WHERE lv.lookup_id = (
            SELECT id FROM lookups WHERE name = 'semantic_types' AND tenant_id = $1
        )
        AND lv.metadata->>'semantic_type' = 'Measure'
        ORDER BY lv.metadata->>'data_type', lv.metadata->>'format'
    `
    
    rows, err := db.Query(query, tenantID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    measuresByDataType := make(map[string][]models.SemanticTypeLookupValue)
    
    for rows.Next() {
        var value models.SemanticTypeLookupValue
        var metadata json.RawMessage
        
        if err := rows.Scan(&value.ID, &value.LookupID, &value.TenantID, 
            &value.Value, &value.Label, &metadata); err != nil {
            return nil, err
        }
        
        json.Unmarshal(metadata, &value.Metadata)
        
        dataType := string(value.Metadata.DataType)
        measuresByDataType[dataType] = append(measuresByDataType[dataType], value)
    }
    
    return measuresByDataType, nil
}
```

## Frontend Examples (React/TypeScript)

### 1. Using Type Constants in Components

```typescript
import { 
  SemanticTypeValue, 
  isDimension, 
  filterByDataType, 
  DataType,
  SEMANTIC_TYPE_GROUPS 
} from '../types/semanticTypesLookup';

export function DimensionTypeSelector() {
  const allDimensionTypes = Object.values(SEMANTIC_TYPE_GROUPS.dimensions).flat();
  
  return (
    <select defaultValue={SemanticTypeValue.DIMENSION_STRING_DEFAULT}>
      {allDimensionTypes.map(type => (
        <option key={type} value={type}>
          {type.replace(/_/g, ' ')}
        </option>
      ))}
    </select>
  );
}
```

### 2. Using with Property Lookup Hook

```typescript
import { usePropertyLookupMaps } from '../hooks/usePropertyLookupMaps';
import { useTenant } from '../contexts/TenantContext';

interface NodeEditorProps {
  nodeType: any;
  assetProperties: Record<string, any>;
}

export function NodeEditor({ nodeType, assetProperties }: NodeEditorProps) {
  const { tenant } = useTenant();
  const lookupMaps = usePropertyLookupMaps(nodeType, assetProperties);
  
  return (
    <div>
      <label>Semantic Type:</label>
      <select name="semantic_type">
        {lookupMaps.semantic_type?.map((labelMap: any, idx: number) => (
          <option key={idx} value={labelMap.id}>
            {labelMap.label}
          </option>
        ))}
      </select>
    </div>
  );
}
```

### 3. Filtering Semantic Types by Category

```typescript
import { 
  SemanticTypeValue, 
  filterByCategory, 
  SemanticTypeCategory,
  SEMANTIC_TYPE_GROUPS 
} from '../types/semanticTypesLookup';

export function MeasureTypeSelector() {
  const measureTypes = SEMANTIC_TYPE_GROUPS.measures.aggregations;
  
  return (
    <div>
      <h4>Available Measures</h4>
      <select multiple>
        {measureTypes.map(type => (
          <option key={type} value={type}>
            {type.replace(/measure_/, '').replace(/_/g, ' ')}
          </option>
        ))}
      </select>
    </div>
  );
}
```

### 4. Custom Dropdown Component

```typescript
import React, { useMemo } from 'react';
import { 
  SemanticTypeValue, 
  getSemanticTypeMetadata,
  isDimension,
  isMeasure 
} from '../types/semanticTypesLookup';

interface SemanticTypeDropdownProps {
  value: SemanticTypeValue | null;
  onChange: (value: SemanticTypeValue) => void;
  filterType?: 'dimension' | 'measure' | 'all';
  allowedFormats?: string[];
}

export function SemanticTypeDropdown({ 
  value, 
  onChange, 
  filterType = 'all',
  allowedFormats 
}: SemanticTypeDropdownProps) {
  const semanticTypes: SemanticTypeValue[] = useMemo(() => {
    // Get all possible values
    const allValues = Object.values(SemanticTypeValue);
    
    // Filter by type if specified
    let filtered = allValues;
    if (filterType === 'dimension') {
      filtered = filtered.filter(isDimension);
    } else if (filterType === 'measure') {
      filtered = filtered.filter(isMeasure);
    }
    
    // Filter by allowed formats if specified
    if (allowedFormats && allowedFormats.length > 0) {
      filtered = filtered.filter(type => {
        const metadata = getSemanticTypeMetadata(type);
        return metadata && allowedFormats.includes(metadata.format);
      });
    }
    
    return filtered;
  }, [filterType, allowedFormats]);
  
  return (
    <select 
      value={value || ''} 
      onChange={(e) => onChange(e.target.value as SemanticTypeValue)}
    >
      <option value="">-- Select Semantic Type --</option>
      {semanticTypes.map(type => {
        const metadata = getSemanticTypeMetadata(type);
        const label = `${metadata?.semantic_type} - ${type.replace(/^[^_]+_/, '')}`;
        return (
          <option key={type} value={type}>
            {label}
          </option>
        );
      })}
    </select>
  );
}
```

## SQL Examples

### 1. Count Semantic Types by Category

```sql
SELECT 
  lv.metadata->>'semantic_type' as category,
  COUNT(*) as count,
  array_agg(DISTINCT lv.metadata->>'data_type') as data_types
FROM lookup_values lv
WHERE lv.lookup_id = (
  SELECT id FROM lookups WHERE name = 'semantic_types' LIMIT 1
)
GROUP BY lv.metadata->>'semantic_type'
ORDER BY category;
```

### 2. Find All Currency Format Types

```sql
SELECT 
  lv.value,
  lv.label,
  lv.metadata->>'semantic_type' as category,
  lv.metadata->>'data_type' as data_type
FROM lookup_values lv
WHERE lv.lookup_id = (
  SELECT id FROM lookups WHERE name = 'semantic_types' LIMIT 1
)
  AND lv.metadata->>'format' = 'currency'
ORDER BY lv.metadata->>'semantic_type', lv.metadata->>'data_type';
```

### 3. Assign Semantic Type to All String Columns

```sql
UPDATE catalog_node
SET properties = jsonb_set(
  COALESCE(properties, '{}'),
  '{semantic_type}',
  '"dimension_string_default"'
)
WHERE properties->>'data_type' = 'string'
  AND properties->>'semantic_type' IS NULL
  AND tenant_id = $1;
```

### 4. Get Nodes Using Currency Measures

```sql
SELECT cn.id, cn.node_name, cn.node_type, cn.properties
FROM catalog_node cn
JOIN catalog_node_type cnt ON cn.node_type_id = cnt.id
WHERE cn.tenant_id = $1
  AND (
    cn.properties->>'semantic_type' = 'measure_number_currency'
    OR cn.properties->>'semantic_type' = 'measure_sum_currency'
    OR cn.properties->>'semantic_type' = 'measure_number_agg_currency'
  )
ORDER BY cn.node_name;
```

### 5. Migration - Add Semantic Type to Existing Nodes

```sql
-- Add semantic_type to all existing dimension columns
UPDATE catalog_node
SET properties = jsonb_set(
  COALESCE(properties, '{}'),
  '{semantic_type}',
  CASE 
    WHEN properties->>'format' = 'currency' THEN '"dimension_number_currency"'
    WHEN properties->>'format' = 'percent' THEN '"dimension_number_percent"'
    WHEN properties->>'format' = 'id' THEN '"dimension_number_id"'
    WHEN properties->>'data_type' = 'string' THEN '"dimension_string_default"'
    WHEN properties->>'data_type' = 'number' THEN '"dimension_number_default"'
    WHEN properties->>'data_type' = 'boolean' THEN '"dimension_boolean_default"'
    WHEN properties->>'data_type' = 'time' THEN '"dimension_time_default"'
    WHEN properties->>'data_type' = 'geo' THEN '"dimension_geo_default"'
    ELSE '"dimension_string_default"'
  END
)
WHERE tenant_id = $1
  AND node_type = 'column'
  AND properties->>'column_role' = 'dimension'
  AND properties->>'semantic_type' IS NULL;
```

## Real-World Scenarios

### Scenario 1: Financial Reporting Dashboard

```typescript
// Select only currency-formatted measures
const currencyMeasures = filterByFormat(
  Object.values(SEMANTIC_TYPE_GROUPS.measures.aggregations),
  Format.CURRENCY
);

// In UI: Show only currency measures for financial metrics
<SemanticTypeDropdown
  filterType="measure"
  allowedFormats={['currency']}
  onChange={(type) => updateNodeProperty('revenue_node', type)}
/>
```

### Scenario 2: Geographic Analysis

```typescript
// Filter for geo-related dimensions
const geoTypes = [SemanticTypeValue.DIMENSION_GEO_DEFAULT];

// Combine with link format for location URLs
const locationTypes = [
  SemanticTypeValue.DIMENSION_STRING_LINK,
  SemanticTypeValue.DIMENSION_GEO_DEFAULT
];
```

### Scenario 3: Time-Based Analysis

```typescript
// Get all time-related types
const timeTypes = Object.values(SEMANTIC_TYPE_GROUPS.dimensions)
  .flat()
  .concat(Object.values(SEMANTIC_TYPE_GROUPS.measures.simple))
  .filter(type => getSemanticTypeMetadata(type)?.data_type === DataType.TIME);
```

## Best Practices

1. **Always use type constants** instead of hardcoding string values
2. **Use filter functions** for complex queries
3. **Store in properties as JSONB** for efficient querying
4. **Validate semantic types** before persisting to database
5. **Use metadata** for UI display and formatting
6. **Group by category** when presenting choices to users

---

These examples show how to integrate semantic types into your data pipeline, UI, and queries across your Fabric Builder platform.
