# Dynamic Parameters & Measures: Integration Guide

## Overview

This guide shows how to integrate the dynamic parameters and measures system into your existing semantic layer platform.

## Architecture Integration

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Frontend UI   │    │  Dynamic API     │    │  Query Engine   │
│                 │    │                  │    │                 │
│ • Parameter     │◄──►│ • Parameter      │◄──►│ • SQL Generation│
│   Controls      │    │   Resolution     │    │ • Measure       │
│ • Measure       │    │ • Measure        │    │   Builder       │
│   Discovery     │    │   Suggestions    │    │ • Validation    │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                              │                        │
                              ▼                        ▼
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Cube.js       │    │  PostgreSQL      │    │  Cache Layer    │
│   Enhanced      │    │  Database        │    │                 │
│   Config        │    │                  │    │ • Query Results │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

## Step 1: Backend Integration

### 1.1 Add Dynamic Routes to Your API

```go
// In your main API router (api.go)
func SetupDynamicRoutes(r *gin.Engine, dynamicEngine *dynamic.DynamicQueryEngine, templateMgr *query.QueryTemplateManager) {
    dynamicHandler := handlers.NewDynamicQueryHandler(dynamicEngine, templateMgr)

    dynamicGroup := r.Group("/api/dynamic")
    {
        dynamicGroup.POST("/query", dynamicHandler.HandleDynamicQuery)
        dynamicGroup.POST("/suggest-measures", dynamicHandler.HandleDynamicMeasureSuggestion)
        dynamicGroup.POST("/validate-params", dynamicHandler.HandleParameterValidation)
        dynamicGroup.POST("/cube-config", dynamicHandler.HandleCubeConfigGeneration)
    }
}
```

### 1.2 Initialize Components

```go
// In your main.go or server setup
func initializeDynamicSystem() (*dynamic.DynamicQueryEngine, *query.QueryTemplateManager, error) {
    // Initialize your existing components
    cubeEngine := cube.NewCube() // Your existing cube engine
    templateMgr := query.NewQueryTemplateManager()

    // Initialize dynamic engine
    dynamicEngine := dynamic.NewDynamicQueryEngine(cubeEngine, templateMgr)

    // Load default templates and configurations
    if err := loadDefaultDynamicTemplates(templateMgr); err != nil {
        return nil, nil, fmt.Errorf("failed to load templates: %w", err)
    }

    return dynamicEngine, templateMgr, nil
}
```

## Step 2: Frontend Integration

### 2.1 Create Dynamic Query Components

```typescript
// DynamicQueryBuilder.tsx
import React, { useState, useEffect } from 'react';
import { DynamicParameter, DynamicMeasure } from '../types/dynamic';

interface DynamicQueryBuilderProps {
  onQueryChange: (query: DynamicQueryRequest) => void;
  initialQuery?: DynamicQueryRequest;
}

export const DynamicQueryBuilder: React.FC<DynamicQueryBuilderProps> = ({
  onQueryChange,
  initialQuery
}) => {
  const [parameters, setParameters] = useState<DynamicParameter[]>(
    initialQuery?.parameters || []
  );
  const [dynamicMeasures, setDynamicMeasures] = useState<DynamicMeasure[]>(
    initialQuery?.dynamicMeasures || []
  );
  const [suggestions, setSuggestions] = useState<DynamicMeasure[]>([]);

  // Fetch measure suggestions based on current query
  useEffect(() => {
    const fetchSuggestions = async () => {
      try {
        const response = await fetch('/api/dynamic/suggest-measures', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            metrics: [], // Current metrics
            dimensions: [], // Current dimensions
            context: { user: 'analyst', domain: 'finance' }
          })
        });
        const data = await response.json();
        setSuggestions(data.suggestions);
      } catch (error) {
        console.error('Failed to fetch suggestions:', error);
      }
    };

    fetchSuggestions();
  }, [parameters, dynamicMeasures]);

  // Update parent component when query changes
  useEffect(() => {
    onQueryChange({
      parameters,
      dynamicMeasures,
      tableName: 'your_table',
      context: {}
    });
  }, [parameters, dynamicMeasures, onQueryChange]);

  return (
    <div className="dynamic-query-builder">
      <ParameterPanel
        parameters={parameters}
        onParametersChange={setParameters}
      />
      <MeasurePanel
        measures={dynamicMeasures}
        suggestions={suggestions}
        onMeasuresChange={setDynamicMeasures}
      />
      <QueryPreview
        parameters={parameters}
        measures={dynamicMeasures}
      />
    </div>
  );
};
```

### 2.2 Parameter Input Components

```typescript
// ParameterInput.tsx
import React from 'react';
import { DynamicParameter } from '../types/dynamic';

interface ParameterInputProps {
  parameter: DynamicParameter;
  value: any;
  onChange: (value: any) => void;
}

export const ParameterInput: React.FC<ParameterInputProps> = ({
  parameter,
  value,
  onChange
}) => {
  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) => {
    const newValue = e.target.type === 'checkbox' ? e.target.checked : e.target.value;
    onChange(newValue);
  };

  if (parameter.options && parameter.options.length > 0) {
    return (
      <select value={value || ''} onChange={handleChange}>
        <option value="">Select {parameter.name}</option>
        {parameter.options.map(option => (
          <option key={option} value={option}>{option}</option>
        ))}
      </select>
    );
  }

  switch (parameter.type) {
    case 'number':
      return (
        <input
          type="number"
          value={value || ''}
          onChange={handleChange}
          placeholder={parameter.description}
        />
      );
    case 'boolean':
      return (
        <input
          type="checkbox"
          checked={value || false}
          onChange={handleChange}
        />
      );
    case 'date':
      return (
        <input
          type="date"
          value={value || ''}
          onChange={handleChange}
        />
      );
    default:
      return (
        <input
          type="text"
          value={value || ''}
          onChange={handleChange}
          placeholder={parameter.description}
        />
      );
  }
};
```

## Step 3: Integration with Existing PoP System

### 3.1 Extend PoP Handler

```go
// In your existing pop_handler.go
func (ph *PoPHandler) HandleDynamicPoPAnalysis(c *gin.Context) {
    var req DynamicPoPRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Convert to dynamic query
    dynamicReq := &dynamic.DynamicQueryRequest{
        BaseQuery: &models.Query{
            TableName: "pop_computations",
            Metrics:   []string{"current_value", "percent_change"},
            Dimensions: []string{"metric_id", "period_label"},
        },
        Parameters: []dynamic.DynamicParameter{
            {
                Name:         "metric_filter",
                Type:         "filter",
                Value:        req.MetricType,
                Description:  "Filter by metric type",
            },
            {
                Name:         "severity_threshold",
                Type:         "number",
                Value:        req.SeverityThreshold,
                DefaultValue: 5.0,
                Description:  "Minimum severity for anomalies",
            },
        },
        DynamicMeasures: []dynamic.DynamicMeasure{
            {
                Name: "anomaly_score",
                Type: "number",
                SQL:  "CASE WHEN ABS(percent_change) > {{severity_threshold}} THEN ABS(percent_change) ELSE 0 END",
            },
        },
    }

    // Use dynamic engine
    resolved, err := ph.dynamicEngine.ResolveParameters(c.Request.Context(), dynamicReq)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Execute query and return results
    results, err := ph.executeDynamicQuery(resolved)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "query": resolved,
        "results": results,
        "generated_at": time.Now(),
    })
}
```

## Step 4: Testing & Validation

### 4.1 Unit Tests

```go
// dynamic_test.go
func TestDynamicParameterResolution(t *testing.T) {
    engine := setupTestDynamicEngine()

    req := &dynamic.DynamicQueryRequest{
        Parameters: []dynamic.DynamicParameter{
            {
                Name:         "test_param",
                Type:         "string",
                Value:        "test_value",
                Required:     true,
            },
        },
    }

    resolved, err := engine.ResolveParameters(context.Background(), req)
    assert.NoError(t, err)
    assert.Equal(t, "test_value", resolved.Parameters["test_param"])
}

func TestDynamicMeasureGeneration(t *testing.T) {
    engine := setupTestDynamicEngine()

    req := &dynamic.DynamicQueryRequest{
        DynamicMeasures: []dynamic.DynamicMeasure{
            {
                Name: "test_measure",
                Type: "number",
                SQL:  "SUM({{column}}) * {{multiplier}}",
                Parameters: []dynamic.DynamicParameter{
                    {Name: "column", Value: "revenue"},
                    {Name: "multiplier", Value: 1.1},
                },
            },
        },
    }

    resolved, err := engine.ResolveParameters(context.Background(), req)
    assert.NoError(t, err)
    assert.Contains(t, resolved.Metrics[0], "SUM(revenue) * 1.1")
}
```

### 4.2 Integration Tests

```go
// integration_test.go
func TestFullDynamicQueryFlow(t *testing.T) {
    // Setup test server
    router := setupTestRouter()

    // Test dynamic query endpoint
    req := DynamicQueryRequest{
        Metrics:    []string{"revenue"},
        Dimensions: []string{"date"},
        Parameters: []dynamic.DynamicParameter{
            {
                Name:  "date_filter",
                Type:  "filter",
                Value: "2024-01-01",
            },
        },
        DynamicMeasures: []dynamic.DynamicMeasure{
            {
                Name: "growth_rate",
                SQL:  "((current - previous) / previous) * 100",
            },
        },
    }

    w := performTestRequest(router, "POST", "/api/dynamic/query", req)

    assert.Equal(t, http.StatusOK, w.Code)

    var response map[string]interface{}
    json.Unmarshal(w.Body.Bytes(), &response)

    assert.Contains(t, response, "sql")
    assert.Contains(t, response, "parameters")
}
```

## Step 5: Deployment & Monitoring

### 5.1 Configuration

```yaml
# config/dynamic.yaml
dynamic:
  enabled: true
  max_parameters: 20
  max_measures: 10
  cache_ttl: 300
  validation:
    strict_mode: true
    allow_custom_sql: false
  monitoring:
    enable_metrics: true
    log_queries: true
```

### 5.2 Monitoring

```go
// monitoring.go
func setupDynamicMonitoring() {
    // Query performance metrics
    prometheus.MustRegister(dynamicQueryDuration)
    prometheus.MustRegister(dynamicQueryCount)
    prometheus.MustRegister(parameterResolutionErrors)

    // Parameter usage analytics
    prometheus.MustRegister(parameterUsageCount)
    prometheus.MustRegister(measureSuggestionCount)
}
```

## Step 6: Documentation & Examples

### 6.1 API Documentation

```yaml
# OpenAPI spec for dynamic endpoints
paths:
  /api/dynamic/query:
    post:
      summary: Execute dynamic query with parameters
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/DynamicQueryRequest'
      responses:
        '200':
          description: Query executed successfully
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/DynamicQueryResponse'
```

### 6.2 Usage Examples

```typescript
// Example: Dynamic revenue analysis
const revenueQuery = {
  metrics: ['revenue', 'orders'],
  dimensions: ['date', 'region'],
  parameters: [
    {
      name: 'region_filter',
      type: 'filter',
      value: 'US',
      options: ['US', 'EU', 'APAC']
    },
    {
      name: 'growth_threshold',
      type: 'number',
      value: 10,
      description: 'Minimum growth percentage'
    }
  ],
  dynamic_measures: [
    {
      name: 'significant_growth',
      type: 'boolean',
      sql: 'percent_change > {{growth_threshold}}'
    }
  ]
};
```

## Benefits Achieved

✅ **Enhanced User Experience**: Dynamic parameter controls and measure discovery
✅ **Competitive Advantage**: Best practices from Cube.dev, ThoughtSpot, and Looker
✅ **Scalable Architecture**: Modular design that integrates with existing systems
✅ **Enterprise Ready**: Validation, monitoring, and governance built-in
✅ **Future Proof**: Extensible for additional dynamic features

This integration provides your platform with industry-leading dynamic capabilities while maintaining compatibility with your existing semantic layer architecture.</content>
<parameter name="filePath">/Users/eganpj/GitHub/semlayer/DYNAMIC_INTEGRATION_GUIDE.md
