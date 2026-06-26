# Semantic Layout Builder Integration Guide

## Overview

The **Semantic Layout Builder** is a visual, no-code tool for building dashboards directly from your semantic layer (dimensions, measures, cubes). It integrates with your existing semantic views and generates JSON configurations that can be consumed by your layout rendering system.

## Key Features

✅ **Drag & Drop Interface** - Build layouts visually without code
✅ **Semantic Layer Integration** - Automatically pulls dimensions & measures from your semantic views
✅ **Component Library** - Pre-built chart types (KPI cards, tables, line/bar/pie charts)
✅ **JSON Export** - Generate configuration files compatible with your Dynamic UI Generator
✅ **Real-time Configuration** - Select dimensions & measures with checkboxes
✅ **Grid-based Layout** - Flexible 12/16 column grid system

---

## Integration with Your Existing Code

### 1. **Connect to Your Semantic Service**

Replace the mock `semanticViews` state in `SemanticLayoutBuilder.tsx` with your actual semantic layer:

```typescript
import { useSemanticViews } from '../hooks/useSemanticViews';

const SemanticLayoutBuilder: React.FC = () => {
  // Replace this mock data:
  // const [semanticViews] = useState<SemanticView[]>([...]);
  
  // With your actual semantic views hook:
  const { tenant, datasource } = useTenant();
  const { data: semanticViews, isLoading } = useSemanticViews(tenant?.id, datasource?.id);
  
  // ... rest of component
}
```

### 2. **Map Your Semantic Models to the Builder Format**

Your existing semantic models from `/backend/models/semantic.go` define:
- `SemanticViewMeta` with `Dimensions` and `Metrics`
- `SemanticMember` with `Name`, `Label`, `Description`, `Type`

Create a mapper function:

```typescript
// In frontend/src/utils/semanticMapper.ts
import { SemanticView } from '../pages/SemanticLayoutBuilder';

export function mapSemanticViewMetaToBuilderFormat(
  viewMeta: any
): SemanticView {
  return {
    id: viewMeta.id,
    name: viewMeta.name,
    title: viewMeta.name.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase()),
    description: viewMeta.description,
    dimensions: viewMeta.dimensions.map((dim: any) => ({
      id: dim.name,
      name: dim.name,
      type: mapSemanticTypeToBuilderType(dim.type),
      sql: dim.sql || `{CUBE}.${dim.name}`,
      title: dim.label || dim.name,
      description: dim.description
    })),
    measures: viewMeta.metrics.map((metric: any) => ({
      id: metric.name,
      name: metric.name,
      type: mapAggregationType(metric.type),
      sql: metric.sql || `SUM({CUBE}.${metric.name})`,
      title: metric.label || metric.name,
      description: metric.description,
      format: metric.format
    }))
  };
}

function mapSemanticTypeToBuilderType(type: string): 'string' | 'number' | 'time' | 'boolean' {
  const lowerType = type.toLowerCase();
  if (lowerType.includes('string') || lowerType.includes('text')) return 'string';
  if (lowerType.includes('number') || lowerType.includes('int') || lowerType.includes('decimal')) return 'number';
  if (lowerType.includes('time') || lowerType.includes('date')) return 'time';
  if (lowerType.includes('bool')) return 'boolean';
  return 'string';
}

function mapAggregationType(type: string): 'count' | 'sum' | 'avg' | 'min' | 'max' | 'count_distinct' {
  const lowerType = type.toLowerCase();
  if (lowerType.includes('count_distinct')) return 'count_distinct';
  if (lowerType.includes('count')) return 'count';
  if (lowerType.includes('sum')) return 'sum';
  if (lowerType.includes('avg') || lowerType.includes('average')) return 'avg';
  if (lowerType.includes('min')) return 'min';
  if (lowerType.includes('max')) return 'max';
  return 'sum';
}
```

### 3. **Integrate with Your Existing QueryBuilder**

The JSON exported from the Semantic Layout Builder can be consumed by your existing query execution system:

```typescript
// Example generated JSON structure:
{
  "layout": {
    "id": "semantic-dashboard",
    "name": "Portfolio Dashboard",
    "version": "1.0.0"
  },
  "components": [
    {
      "id": "component-12345",
      "type": "DataTable",
      "layout": { "row": 1, "col": 1, "width": 12, "height": 6 },
      "config": {
        "title": "Portfolio Holdings",
        "semanticView": "portfolio_positions",
        "dimensions": ["security_name", "sector"],
        "measures": ["market_value", "gain_loss"],
        "filters": [],
        "sort": [{ "field": "market_value", "direction": "desc" }]
      }
    }
  ]
}
```

Execute queries from this config:

```typescript
// In your rendering layer:
import { executeSemanticQuery } from '../api/semanticApi';

async function renderLayoutComponent(component: LayoutComponent) {
  const queryPayload = {
    dimensions: component.config.dimensions,
    metrics: component.config.measures,
    filters: component.config.filters || [],
    order: component.config.sort || [],
    limit: 100
  };
  
  const results = await executeSemanticQuery(
    component.config.semanticView,
    queryPayload
  );
  
  // Render based on component.type
  switch (component.type) {
    case 'DataTable':
      return <DataTable data={results.rows} columns={results.columns} />;
    case 'LineChart':
      return <LineChart data={results.rows} xAxis={component.config.dimensions[0]} yAxis={component.config.measures[0]} />;
    // ... other component types
  }
}
```

---

## Example: Full Integration Flow

### Step 1: Fetch Your Semantic Views

```typescript
// frontend/src/pages/SemanticLayoutBuilder.tsx
import { useQuery } from '@tanstack/react-query';
import { listSemanticViews } from '../api/semanticApi';
import { mapSemanticViewMetaToBuilderFormat } from '../utils/semanticMapper';

const SemanticLayoutBuilder: React.FC = () => {
  const { tenant, datasource } = useTenant();
  
  const { data: rawViews, isLoading } = useQuery({
    queryKey: ['semanticViews', tenant?.id, datasource?.id],
    queryFn: () => listSemanticViews(tenant!.id, datasource!.id),
    enabled: !!tenant && !!datasource
  });
  
  const semanticViews = rawViews?.map(mapSemanticViewMetaToBuilderFormat) || [];
  
  // ... rest of component
}
```

### Step 2: User Builds Dashboard Visually

1. User drags a "Data Table" component onto canvas
2. User selects "Portfolio Positions" semantic view
3. User checks dimensions: `security_name`, `sector`
4. User checks measures: `market_value`, `gain_loss`
5. User clicks "Export JSON"

### Step 3: Save & Load Layouts

```typescript
// Save the layout configuration
const saveLayout = async (config: any) => {
  await fetch('/api/layouts', {
    method: 'POST',
    headers: buildTenantHeadersFromLocalStorage(),
    body: JSON.stringify(config)
  });
};

// Load and render a saved layout
const loadLayout = async (layoutId: string) => {
  const response = await fetch(`/api/layouts/${layoutId}`, {
    headers: buildTenantHeadersFromLocalStorage()
  });
  const config = await response.json();
  
  // Render each component
  return config.components.map(comp => renderLayoutComponent(comp));
};
```

---

## Connecting to Your Backend

### Backend API Integration

Your existing semantic service already has the foundation. Extend it to support layout execution:

```go
// backend/internal/services/semantic_service.go

// ExecuteLayoutComponent runs a query defined by a layout component config
func (s *SemanticService) ExecuteLayoutComponent(ctx context.Context, config LayoutComponentConfig) (*models.ExecuteResult, error) {
    query := models.SemanticQuery{
        Dimensions: config.Dimensions,
        Metrics:    config.Measures,
        Filters:    config.Filters,
        Order:      config.Sort,
        Limit:      100,
    }
    
    return s.ExecuteSemanticQuery(ctx, config.SemanticView, query)
}
```

---

## Extending the Component Library

Add new visualization types to match your domain:

```typescript
// In SemanticLayoutBuilder.tsx
const componentLibrary = [
  // Existing components...
  
  // Investment-specific components
  { 
    type: 'CandlestickChart', 
    icon: TrendingUp, 
    label: 'Candlestick Chart', 
    defaultSize: { width: 8, height: 6 },
    requiredDimensions: ['date'],
    requiredMeasures: ['open', 'high', 'low', 'close']
  },
  { 
    type: 'PerformanceAttribution', 
    icon: BarChart3, 
    label: 'Attribution Analysis', 
    defaultSize: { width: 12, height: 8 },
    requiredMeasures: ['allocation_effect', 'selection_effect']
  },
  { 
    type: 'RiskMetricsCard', 
    icon: AlertTriangle, 
    label: 'Risk Dashboard', 
    defaultSize: { width: 6, height: 4 },
    requiredMeasures: ['sharpe_ratio', 'volatility', 'beta']
  }
];
```

---

## Advanced: Pre-aggregation Support

If you have pre-aggregated cubes, the layout builder can leverage them automatically:

```typescript
// In your semantic query execution:
async function executeWithPreaggregation(config: LayoutComponentConfig) {
  // Check if a pre-aggregation matches the query grain
  const preaggMatch = await findMatchingPreaggregation(
    config.semanticView,
    config.dimensions,
    config.measures
  );
  
  if (preaggMatch) {
    // Route to pre-aggregated table
    return executeFastQuery(preaggMatch.tableName, config);
  } else {
    // Fallback to full semantic query
    return executeSemanticQuery(config.semanticView, config);
  }
}
```

---

## Testing Your Integration

1. **Load semantic views**: Verify views appear in the sidebar
2. **Build a simple dashboard**: Drag a table, select dimensions/measures
3. **Export JSON**: Check the configuration structure
4. **Execute the query**: Manually call your semantic API with the exported config
5. **Render the results**: Display the data in your chosen visualization

---

## Comparison to Workday & Cube.dev

| Feature | Workday Studio | Cube.dev Playground | Your Semantic Layout Builder |
|---------|---------------|---------------------|------------------------------|
| **Visual Builder** | ✅ Drag & drop | ✅ Query builder UI | ✅ Drag & drop + visual config |
| **Semantic Layer** | ❌ Custom logic | ✅ Native cubes | ✅ Your semantic views |
| **Investment-specific** | ❌ Generic | ❌ Generic | ✅ Purpose-built for finance |
| **No-code** | ⚠️ Requires Studio IDE | ✅ Web UI | ✅ Pure web UI |
| **JSON Export** | ❌ Proprietary | ⚠️ Limited | ✅ Full layout config |
| **Pre-aggregation** | ❌ | ✅ | ✅ (when implemented) |

---

## Next Steps

1. **Hook up your semantic service** - Replace mock data with real API calls
2. **Add more chart types** - Candlestick, heatmaps, attribution waterfalls
3. **Build a renderer** - Create React components that consume the JSON layouts
4. **Add filtering UI** - Let users add filters visually (not just in JSON)
5. **Save/load layouts** - Persist configurations to your backend
6. **Multi-tenant support** - Ensure layouts respect tenant/datasource scope

---

## Example: Building a Portfolio Dashboard

**User Actions:**
1. Opens Semantic Layout Builder
2. Drags "KPI Card" → Selects view "portfolio_positions", measure "market_value"
3. Drags "Pie Chart" → Selects dimensions ["sector"], measures ["market_value"]
4. Drags "Data Table" → Selects dimensions ["security_name", "sector"], measures ["market_value", "gain_loss"]
5. Clicks "Export JSON"

**Generated JSON:**
```json
{
  "layout": {
    "id": "portfolio-dashboard",
    "name": "Portfolio Dashboard",
    "version": "1.0.0"
  },
  "components": [
    {
      "id": "kpi-1",
      "type": "MetricCard",
      "layout": { "row": 1, "col": 1, "width": 3, "height": 2 },
      "config": {
        "title": "Total Portfolio Value",
        "semanticView": "portfolio_positions",
        "measures": ["market_value"]
      }
    },
    {
      "id": "pie-1",
      "type": "PieChart",
      "layout": { "row": 1, "col": 4, "width": 4, "height": 4 },
      "config": {
        "title": "Allocation by Sector",
        "semanticView": "portfolio_positions",
        "dimensions": ["sector"],
        "measures": ["market_value"]
      }
    },
    {
      "id": "table-1",
      "type": "DataTable",
      "layout": { "row": 3, "col": 1, "width": 12, "height": 6 },
      "config": {
        "title": "Top Holdings",
        "semanticView": "portfolio_positions",
        "dimensions": ["security_name", "sector"],
        "measures": ["market_value", "gain_loss"],
        "sort": [{ "field": "market_value", "direction": "desc" }]
      }
    }
  ]
}
```

**Rendering:**
Your layout renderer loads this JSON and executes 3 semantic queries, then displays the results in the configured components.

---

## Troubleshooting

### "Semantic views not loading"
- Check that `listSemanticViews` API call is working
- Verify tenant/datasource headers are set correctly
- Inspect network tab for API errors

### "Dimensions/measures not appearing"
- Ensure your semantic views have `dimensions` and `measures` arrays
- Check the mapper function is correctly transforming your backend models

### "Component won't render"
- Verify the component type exists in your renderer
- Check that required dimensions/measures are selected
- Inspect the generated JSON for missing fields

---

## Summary

The **Semantic Layout Builder** bridges the gap between your powerful semantic layer and end-user dashboard creation. Unlike Workday's complex Studio or Cube.dev's developer-focused Playground, this gives business users a visual, no-code way to build investment management dashboards directly from your semantic models.

**Key advantage:** Because it's integrated with your existing semantic layer (dimensions, measures, cubes), users get:
- Type safety (only valid dimensions/measures selectable)
- Pre-aggregation benefits (fast queries)
- Governance (access control via semantic layer)
- Flexibility (JSON export for custom integrations)

Ready to deploy! 🚀
