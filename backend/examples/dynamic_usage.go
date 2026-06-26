//go:build ignore

package main

import (
	"context"
	"fmt"
	"log"

	"github.com/hondyman/semlayer/backend/internal/cube"
	"github.com/hondyman/semlayer/backend/internal/dynamic"
	"github.com/hondyman/semlayer/backend/internal/query"
	"github.com/hondyman/semlayer/backend/models"
)

func main() {
	// Initialize components
	cubeEngine := (*cube.Cube)(nil) // Placeholder for cube engine
	templateMgr := query.NewQueryTemplateManager()
	dynamicEngine := dynamic.NewDynamicQueryEngine(cubeEngine, templateMgr)

	// Example 1: Dynamic PoP Analysis with Parameters
	fmt.Println("=== Example 1: Dynamic PoP Analysis ===")

	popRequest := &dynamic.DynamicQueryRequest{
		BaseQuery: &models.Query{
			TableName:  "pop_computations",
			Metrics:    []string{"current_value", "percent_change"},
			Dimensions: []string{"metric_id", "period_label"},
		},
		Parameters: []dynamic.DynamicParameter{
			{
				Name:        "metric_type",
				Type:        "dimension",
				Value:       "revenue",
				Required:    true,
				Description: "Type of metric to analyze",
				Options:     []string{"revenue", "users", "orders", "profit"},
			},
			{
				Name:         "time_period",
				Type:         "filter",
				Value:        "2024-Q3",
				Required:     false,
				DefaultValue: "2024-Q3",
				Description:  "Analysis period",
			},
			{
				Name:         "threshold",
				Type:         "measure",
				Value:        5.0,
				Required:     false,
				DefaultValue: 0.0,
				Description:  "Minimum change threshold (%)",
			},
		},
		DynamicMeasures: []dynamic.DynamicMeasure{
			{
				Name: "significant_change",
				Type: "boolean",
				SQL:  "ABS(percent_change) > {{threshold}}",
				Parameters: []dynamic.DynamicParameter{
					{Name: "threshold", Type: "number"},
				},
			},
			{
				Name: "change_category",
				Type: "string",
				SQL: `
					CASE
						WHEN percent_change > 10 THEN 'strong_growth'
						WHEN percent_change > 0 THEN 'growth'
						WHEN percent_change > -10 THEN 'decline'
						ELSE 'strong_decline'
					END
				`,
			},
		},
		Context: map[string]interface{}{
			"user_role":  "analyst",
			"department": "finance",
			"clearance":  "confidential",
		},
	}

	// Resolve the dynamic query
	ctx := context.Background()
	resolvedQuery, err := dynamicEngine.ResolveParameters(ctx, popRequest)
	if err != nil {
		log.Fatalf("Failed to resolve dynamic query: %v", err)
	}

	fmt.Printf("Resolved Parameters: %+v\n", resolvedQuery.Parameters)
	fmt.Printf("Dynamic Metrics: %+v\n", resolvedQuery.Metrics)

	// Generate SQL
	sql, args := resolvedQuery.BuildSQL()
	fmt.Printf("Generated SQL: %s\n", sql)
	fmt.Printf("Query Args: %+v\n", args)

	// Example 2: Cube.js Enhancement
	fmt.Println("\n=== Example 2: Enhanced Cube.js Configuration ===")

	cubeEnhancer := dynamic.NewCubeDynamicEnhancer(cubeEngine)

	// Define dynamic parameters for Cube.js
	cubeParams := []dynamic.DynamicParameter{
		{
			Name:         "analysis_period",
			Type:         "time_range",
			DefaultValue: "last_30_days",
			Description:  "Time period for analysis",
		},
		{
			Name:        "kpi_category",
			Type:        "dimension",
			Options:     []string{"financial", "operational", "customer"},
			Description: "Category of KPIs to analyze",
		},
	}

	// Define dynamic measures
	cubeMeasures := []dynamic.DynamicMeasure{
		{
			Name: "dynamic_growth_rate",
			Type: "percentage",
			SQL: `
				CASE
					WHEN {{ growth_type }} = 'revenue' THEN
						(SUM(CASE WHEN period = 'current' THEN revenue END) -
						 SUM(CASE WHEN period = 'previous' THEN revenue END)) /
						SUM(CASE WHEN period = 'previous' THEN revenue END) * 100
					WHEN {{ growth_type }} = 'users' THEN
						(SUM(CASE WHEN period = 'current' THEN user_count END) -
						 SUM(CASE WHEN period = 'previous' THEN user_count END)) /
						SUM(CASE WHEN period = 'previous' THEN user_count END) * 100
				END
			`,
			Parameters: []dynamic.DynamicParameter{
				{
					Name:         "growth_type",
					Type:         "string",
					DefaultValue: "revenue",
					Options:      []string{"revenue", "users", "orders"},
				},
			},
		},
	}

	// Generate enhanced Cube.js configuration
	cubeConfig, err := cubeEnhancer.GenerateCubeJSConfig(cubeParams, cubeMeasures)
	if err != nil {
		log.Fatalf("Failed to generate Cube.js config: %v", err)
	}

	fmt.Println("Enhanced Cube.js Configuration:")
	fmt.Println(cubeConfig)

	// Generate parameter schema for validation
	paramSchema, err := cubeEnhancer.GenerateParameterSchema(cubeParams)
	if err != nil {
		log.Fatalf("Failed to generate parameter schema: %v", err)
	}

	fmt.Println("Parameter JSON Schema:")
	fmt.Println(paramSchema)

	// Example 3: Integration with Existing PoP System
	fmt.Println("\n=== Example 3: PoP System Integration ===")

	// Create a dynamic query for anomaly detection
	anomalyRequest := &dynamic.DynamicQueryRequest{
		BaseQuery: &models.Query{
			TableName:  "pop_anomalies",
			Metrics:    []string{"anomaly_score", "severity"},
			Dimensions: []string{"metric_id", "anomaly_type"},
		},
		Parameters: []dynamic.DynamicParameter{
			{
				Name:         "severity_filter",
				Type:         "filter",
				DefaultValue: "high",
				Options:      []string{"low", "medium", "high", "critical"},
				Description:  "Filter anomalies by severity",
			},
			{
				Name:         "anomaly_types",
				Type:         "filter",
				DefaultValue: []string{"z_score", "trend_break"},
				Description:  "Types of anomalies to include",
			},
		},
		DynamicMeasures: []dynamic.DynamicMeasure{
			{
				Name: "anomaly_impact_score",
				Type: "number",
				SQL: `
					CASE severity
						WHEN 'critical' THEN anomaly_score * 10
						WHEN 'high' THEN anomaly_score * 5
						WHEN 'medium' THEN anomaly_score * 2
						ELSE anomaly_score
					END
				`,
			},
		},
	}

	anomalyQuery, err := dynamicEngine.ResolveParameters(ctx, anomalyRequest)
	if err != nil {
		log.Fatalf("Failed to resolve anomaly query: %v", err)
	}

	fmt.Printf("Anomaly Analysis Query: %+v\n", anomalyQuery)

	fmt.Println("\n✅ Dynamic Parameters & Measures Implementation Complete!")
	fmt.Println("🎯 Your platform now supports:")
	fmt.Println("   • Runtime parameter resolution")
	fmt.Println("   • Dynamic measure generation")
	fmt.Println("   • Enhanced Cube.js integration")
	fmt.Println("   • PoP system integration")
	fmt.Println("   • Parameter validation and schemas")
}
