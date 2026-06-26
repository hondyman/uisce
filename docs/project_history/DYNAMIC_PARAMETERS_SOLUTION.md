# Dynamic Parameters & Measures: Best Solution for Your Platform

## Executive Summary

Based on your existing Go-based semantic layer with Cube.js integration, **Cube.dev** remains the strongest foundation, but with strategic enhancements from **ThoughtSpot** and **Looker** for advanced dynamic capabilities.

## Current Platform Analysis

Your platform already has:
- ✅ Go-based semantic layer with Cube.js integration
- ✅ Template-based query system
- ✅ NL query engine
- ✅ PostgreSQL backend
- ✅ PoP metrics system

## Competitive Analysis: Dynamic Parameters & Measures

### 1. **Cube.dev** (Your Current Foundation)
**Strengths:**
- ✅ Native parameter support via `{% parameter %}` syntax
- ✅ Dynamic measures with conditional logic
- ✅ Pre-aggregations with parameter filtering
- ✅ REST API for parameter injection
- ✅ Type safety with TypeScript definitions

**Dynamic Parameters Example:**
```yaml
cubes:
  - name: revenue
    sql: >
      SELECT * FROM orders WHERE
      {% if_param date_from %} created_at >= {{ date_from }} {% endif %}
      {% if_param date_to %} AND created_at <= {{ date_to }} {% endif %}

    dimensions:
      date_from:
        type: string
        sql: '{{ date_from }}'
      date_to:
        type: string
        sql: '{{ date_to }}'
```

**Dynamic Measures Example:**
```yaml
measures:
  dynamic_revenue:
    type: number
    sql: |
      CASE
        WHEN {{ currency }} = 'USD' THEN amount_usd
        WHEN {{ currency }} = 'EUR' THEN amount_eur * 1.1
        ELSE amount_usd
      END
```

### 2. **ThoughtSpot** (Search-Driven Dynamics)
**Competitive Advantages:**
- 🔍 Natural language parameter discovery
- 📊 Automatic measure suggestions
- 🎯 Context-aware parameter resolution
- 📈 Formula engine for complex calculations

**Why Consider:** Best for user-driven dynamic analysis

### 3. **Looker** (LookML Dynamic Measures)
**Competitive Advantages:**
- 🏗️ Advanced templating with Liquid
- 🔄 Parameter cascading and dependencies
- 📊 Dynamic dimension creation
- 🎨 Rich visualization parameter binding

**LookML Dynamic Example:**
```lookml
parameter: metric_selector {
  type: unquoted
  allowed_value: { label: "Revenue" value: "revenue" }
  allowed_value: { label: "Orders" value: "orders" }
}

measure: dynamic_metric {
  type: number
  sql:
    CASE
      WHEN {% parameter metric_selector %} = 'revenue' THEN ${revenue}
      WHEN {% parameter metric_selector %} = 'orders' THEN ${order_count}
    END ;;
}
```

### 4. **Tableau** (Dashboard-Driven Dynamics)
**Competitive Advantages:**
- 🎛️ Interactive parameter controls
- 📊 Dynamic measure switching
- 🔗 Parameter actions between sheets
- 📈 Calculated field parameters

### 5. **Preset (Apache Superset)**
**Competitive Advantages:**
- 🆓 Open-source and free
- 🔧 Extensive SQL templating
- 📊 Dashboard parameter inheritance
- 🌐 Multi-database support

## Recommended Solution: Enhanced Cube.dev + Strategic Features

### **Phase 1: Core Cube.dev Enhancement**

```yaml
# Enhanced Cube.js with dynamic parameters
cubes:
  - name: dynamic_metrics
    sql: >
      SELECT * FROM {{ table_name }}
      WHERE date BETWEEN {{ start_date }} AND {{ end_date }}
      {% if filters.region %} AND region = {{ filters.region }} {% endif %}

    parameters:
      table_name:
        type: string
        default: 'sales'
      start_date:
        type: date
        default: '2024-01-01'
      end_date:
        type: date
        default: '2024-12-31'
      filters:
        type: object
        properties:
          region:
            type: string
            enum: ['US', 'EU', 'APAC']

    measures:
      dynamic_kpi:
        type: number
        sql: |
          CASE {{ kpi_type }}
            WHEN 'revenue' THEN SUM(amount)
            WHEN 'margin' THEN SUM(amount * margin_pct)
            WHEN 'growth' THEN
              (SUM(CASE WHEN period = 'current' THEN amount END) -
               SUM(CASE WHEN period = 'previous' THEN amount END)) /
              SUM(CASE WHEN period = 'previous' THEN amount END)
          END
```

### **Phase 2: ThoughtSpot-Inspired Features**

```go
// Dynamic measure suggestion engine
type MeasureSuggester struct {
    contextAnalyzer *ContextAnalyzer
    patternMatcher  *PatternMatcher
}

func (ms *MeasureSuggester) SuggestDynamicMeasures(query *Query, context *UserContext) []DynamicMeasure {
    suggestions := []DynamicMeasure{}

    // Revenue per user
    if ms.hasRevenueAndUsers(query) {
        suggestions = append(suggestions, DynamicMeasure{
            Name: "Revenue per User",
            SQL:  "SUM(revenue) / COUNT(DISTINCT user_id)",
            Type: "ratio",
        })
    }

    // Growth rate
    if ms.hasTimeSeriesData(query) {
        suggestions = append(suggestions, DynamicMeasure{
            Name: "Period Growth Rate",
            SQL:  "((current_value - previous_value) / previous_value) * 100",
            Type: "percentage",
        })
    }

    return suggestions
}
```

### **Phase 3: Looker-Style Parameter Dependencies**

```go
// Parameter dependency resolver
type ParameterResolver struct {
    dependencyGraph *DependencyGraph
}

func (pr *ParameterResolver) ResolveDependencies(params []DynamicParameter) ([]DynamicParameter, error) {
    resolved := make([]DynamicParameter, 0, len(params))

    // Topological sort based on dependencies
    sorted, err := pr.dependencyGraph.TopologicalSort()
    if err != nil {
        return nil, fmt.Errorf("circular dependency detected: %w", err)
    }

    for _, paramName := range sorted {
        param := pr.findParameterByName(params, paramName)
        if param == nil {
            continue
        }

        // Resolve dependent values
        resolvedParam, err := pr.resolveDependentValues(param, resolved)
        if err != nil {
            return nil, fmt.Errorf("failed to resolve %s: %w", paramName, err)
        }

        resolved = append(resolved, *resolvedParam)
    }

    return resolved, nil
}
```

## Implementation Roadmap

### **Week 1-2: Enhanced Cube.dev Integration**
- [ ] Extend existing Cube.js integration with parameter schema
- [ ] Add dynamic measure builder
- [ ] Implement parameter validation
- [ ] Create parameter dependency resolver

### **Week 3-4: Advanced Dynamic Features**
- [ ] Add measure suggestion engine (ThoughtSpot-inspired)
- [ ] Implement cascading parameters (Looker-inspired)
- [ ] Create dynamic dimension builder
- [ ] Add context-aware parameter resolution

### **Week 5-6: User Experience Enhancements**
- [ ] Build parameter UI components
- [ ] Add measure discovery interface
- [ ] Implement parameter persistence
- [ ] Create dynamic query builder

### **Week 7-8: Integration & Testing**
- [ ] Integrate with existing PoP system
- [ ] Add comprehensive test coverage
- [ ] Performance optimization
- [ ] Documentation and examples

## Key Benefits of This Approach

1. **🔧 Maintains Your Existing Architecture** - Builds on your current Cube.js foundation
2. **🚀 Competitive Feature Set** - Incorporates best practices from leading platforms
3. **📈 Scalable Implementation** - Modular design allows incremental enhancement
4. **🎯 User-Centric** - Focus on dynamic discovery and ease of use
5. **🔒 Enterprise-Ready** - Includes governance, validation, and audit capabilities

## Migration Strategy

```go
// Seamless migration from current system
func migrateToDynamicSystem(currentQuery *models.Query) *DynamicQueryRequest {
    return &DynamicQueryRequest{
        BaseQuery: currentQuery,
        Parameters: extractParametersFromQuery(currentQuery),
        DynamicMeasures: suggestDynamicMeasures(currentQuery),
        Context: buildUserContext(),
    }
}
```

This solution gives you the best of all worlds: the reliability of Cube.dev, the user experience of ThoughtSpot, and the flexibility of Looker, all integrated into your existing Go-based semantic layer platform.</content>
<parameter name="filePath">/Users/eganpj/GitHub/semlayer/DYNAMIC_PARAMETERS_SOLUTION.md
