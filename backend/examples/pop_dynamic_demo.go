package main

import (
	"context"
	"fmt"
	"log"

	"github.com/hondyman/semlayer/backend/internal/dynamic"
	"github.com/hondyman/semlayer/backend/internal/query"
	"github.com/hondyman/semlayer/backend/models"
)

func main() {
	fmt.Println("🚀 Dynamic Parameters & Measures Demo with PoP Metrics")
	fmt.Println("====================================================")

	// Initialize components
	templateMgr := query.NewQueryTemplateManager()
	dynamicEngine := dynamic.NewDynamicQueryEngine(nil, templateMgr)

	// Demo 1: Dynamic PoP Analysis with Real Metrics
	fmt.Println("\n📊 Demo 1: Dynamic PoP Analysis")
	fmt.Println("-------------------------------")

	popAnalysis := &dynamic.DynamicQueryRequest{
		BaseQuery: &models.Query{
			TableName:  "pop_computations",
			Metrics:    []string{"current_value", "percent_change"},
			Dimensions: []string{"metric_id", "period_label"},
		},
		Parameters: []dynamic.DynamicParameter{
			{
				Name:        "metric_category",
				Type:        "dimension",
				Value:       "finance",
				Required:    true,
				Description: "Category of metrics to analyze",
				Options:     []string{"finance", "operations", "compliance"},
			},
			{
				Name:         "min_change_threshold",
				Type:         "number",
				Value:        5.0,
				DefaultValue: 0.0,
				Description:  "Minimum percentage change to highlight",
			},
			{
				Name:        "analysis_period",
				Type:        "filter",
				Value:       "2024-08",
				Description: "Period to analyze",
			},
		},
		DynamicMeasures: []dynamic.DynamicMeasure{
			{
				Name: "significant_change_flag",
				Type: "boolean",
				SQL:  "ABS(percent_change) > {{min_change_threshold}}",
				Parameters: []dynamic.DynamicParameter{
					{Name: "min_change_threshold", Type: "number"},
				},
			},
			{
				Name: "change_magnitude",
				Type: "string",
				SQL: `
					CASE
						WHEN ABS(percent_change) > 20 THEN 'extreme'
						WHEN ABS(percent_change) > 10 THEN 'high'
						WHEN ABS(percent_change) > 5 THEN 'moderate'
						ELSE 'low'
					END
				`,
			},
			{
				Name: "performance_indicator",
				Type: "string",
				SQL: `
					CASE
						WHEN percent_change > 0 THEN 'positive'
						WHEN percent_change < 0 THEN 'negative'
						ELSE 'neutral'
					END
				`,
			},
		},
		Context: map[string]interface{}{
			"user_role":        "analyst",
			"department":       "finance",
			"clearance":        "confidential",
			"preferred_format": "detailed",
		},
	}

	ctx := context.Background()
	resolved, err := dynamicEngine.ResolveParameters(ctx, popAnalysis)
	if err != nil {
		log.Fatalf("Failed to resolve PoP analysis: %v", err)
	}

	fmt.Printf("✅ Resolved Parameters: %+v\n", resolved.Parameters)
	fmt.Printf("📈 Dynamic Measures Generated: %+v\n", resolved.Metrics)

	// Demo 2: Anomaly Detection with Dynamic Parameters
	fmt.Println("\n🔍 Demo 2: Dynamic Anomaly Detection")
	fmt.Println("-----------------------------------")

	anomalyAnalysis := &dynamic.DynamicQueryRequest{
		BaseQuery: &models.Query{
			TableName:  "pop_anomalies",
			Metrics:    []string{"anomaly_score", "severity", "confidence"},
			Dimensions: []string{"metric_id", "anomaly_type"},
		},
		Parameters: []dynamic.DynamicParameter{
			{
				Name:        "severity_filter",
				Type:        "filter",
				Value:       "high",
				Options:     []string{"low", "medium", "high", "critical"},
				Description: "Filter anomalies by severity level",
			},
			{
				Name:         "confidence_threshold",
				Type:         "number",
				Value:        0.8,
				DefaultValue: 0.5,
				Description:  "Minimum confidence score for anomalies",
			},
			{
				Name:        "anomaly_types",
				Type:        "filter",
				Value:       []string{"z_score", "trend_break"},
				Description: "Types of anomalies to include",
			},
		},
		DynamicMeasures: []dynamic.DynamicMeasure{
			{
				Name: "anomaly_priority_score",
				Type: "number",
				SQL: `
					CASE severity
						WHEN 'critical' THEN confidence * 100
						WHEN 'high' THEN confidence * 75
						WHEN 'medium' THEN confidence * 50
						ELSE confidence * 25
					END
				`,
			},
			{
				Name: "requires_immediate_action",
				Type: "boolean",
				SQL:  "severity IN ('critical', 'high') AND confidence > {{confidence_threshold}}",
				Parameters: []dynamic.DynamicParameter{
					{Name: "confidence_threshold", Type: "number"},
				},
			},
		},
	}

	anomalyResolved, err := dynamicEngine.ResolveParameters(ctx, anomalyAnalysis)
	if err != nil {
		log.Fatalf("Failed to resolve anomaly analysis: %v", err)
	}

	fmt.Printf("✅ Anomaly Parameters: %+v\n", anomalyResolved.Parameters)
	fmt.Printf("🚨 Dynamic Risk Measures: %+v\n", anomalyResolved.Metrics)

	// Demo 3: Steward Review Integration
	fmt.Println("\n👥 Demo 3: Dynamic Steward Review Analysis")
	fmt.Println("----------------------------------------")

	stewardAnalysis := &dynamic.DynamicQueryRequest{
		BaseQuery: &models.Query{
			TableName:  "pop_steward_reviews",
			Metrics:    []string{"overall_rating"},
			Dimensions: []string{"reviewer_user_id", "review_type"},
		},
		Parameters: []dynamic.DynamicParameter{
			{
				Name:        "review_status",
				Type:        "filter",
				Value:       "in_progress",
				Options:     []string{"in_progress", "completed", "overdue"},
				Description: "Filter reviews by status",
			},
			{
				Name:        "reviewer_department",
				Type:        "dimension",
				Value:       "risk",
				Options:     []string{"finance", "risk", "operations", "compliance"},
				Description: "Department of the reviewer",
			},
			{
				Name:        "days_overdue",
				Type:        "number",
				Value:       7,
				Description: "Days past due date to flag",
			},
		},
		DynamicMeasures: []dynamic.DynamicMeasure{
			{
				Name: "review_urgency",
				Type: "string",
				SQL: `
					CASE
						WHEN status = 'overdue' AND
							 CURRENT_DATE > due_date + INTERVAL '{{days_overdue}} days'
						THEN 'critical'
						WHEN status = 'overdue' THEN 'high'
						WHEN status = 'in_progress' AND
							 CURRENT_DATE > due_date - INTERVAL '2 days'
						THEN 'medium'
						ELSE 'low'
					END
				`,
				Parameters: []dynamic.DynamicParameter{
					{Name: "days_overdue", Type: "number"},
				},
			},
		},
	}

	stewardResolved, err := dynamicEngine.ResolveParameters(ctx, stewardAnalysis)
	if err != nil {
		log.Fatalf("Failed to resolve steward analysis: %v", err)
	}

	fmt.Printf("✅ Steward Parameters: %+v\n", stewardResolved.Parameters)
	fmt.Printf("📋 Dynamic Review Measures: %+v\n", stewardResolved.Metrics)

	// Demo 4: Dashboard Integration
	fmt.Println("\n📊 Demo 4: Dynamic Dashboard Configuration")
	fmt.Println("----------------------------------------")

	dashboardAnalysis := &dynamic.DynamicQueryRequest{
		BaseQuery: &models.Query{
			TableName:  "pop_dashboards",
			Metrics:    []string{"name"},
			Dimensions: []string{"owner_user_id"},
		},
		Parameters: []dynamic.DynamicParameter{
			{
				Name:        "dashboard_owner",
				Type:        "filter",
				Value:       "steward.risk@company.com",
				Description: "Owner of the dashboard",
			},
			{
				Name:        "include_public",
				Type:        "boolean",
				Value:       false,
				Description: "Include public dashboards",
			},
			{
				Name:        "refresh_interval",
				Type:        "number",
				Value:       300,
				Description: "Dashboard refresh interval in seconds",
			},
		},
		DynamicMeasures: []dynamic.DynamicMeasure{
			{
				Name: "dashboard_activity_score",
				Type: "number",
				SQL: `
					CASE
						WHEN refresh_interval < 60 THEN 100
						WHEN refresh_interval < 300 THEN 75
						WHEN refresh_interval < 900 THEN 50
						ELSE 25
					END
				`,
			},
			{
				Name: "needs_refresh_optimization",
				Type: "boolean",
				SQL:  "refresh_interval > {{refresh_interval}}",
				Parameters: []dynamic.DynamicParameter{
					{Name: "refresh_interval", Type: "number"},
				},
			},
		},
	}

	dashboardResolved, err := dynamicEngine.ResolveParameters(ctx, dashboardAnalysis)
	if err != nil {
		log.Fatalf("Failed to resolve dashboard analysis: %v", err)
	}

	fmt.Printf("✅ Dashboard Parameters: %+v\n", dashboardResolved.Parameters)
	fmt.Printf("🎛️ Dynamic Dashboard Measures: %+v\n", dashboardResolved.Metrics)

	// Demo 5: Multi-Metric Comparison
	fmt.Println("\n⚖️ Demo 5: Dynamic Multi-Metric Comparison")
	fmt.Println("---------------------------------------")

	comparisonAnalysis := &dynamic.DynamicQueryRequest{
		BaseQuery: &models.Query{
			TableName:  "pop_metrics",
			Metrics:    []string{"name", "display_name"},
			Dimensions: []string{"domain", "category"},
		},
		Parameters: []dynamic.DynamicParameter{
			{
				Name:        "primary_domain",
				Type:        "dimension",
				Value:       "finance",
				Options:     []string{"finance", "operations", "compliance"},
				Description: "Primary domain for comparison",
			},
			{
				Name:        "comparison_domains",
				Type:        "filter",
				Value:       []string{"operations", "compliance"},
				Description: "Additional domains to compare",
			},
			{
				Name:        "metric_status",
				Type:        "filter",
				Value:       "active",
				Options:     []string{"active", "draft", "deprecated"},
				Description: "Status of metrics to include",
			},
		},
		DynamicMeasures: []dynamic.DynamicMeasure{
			{
				Name: "domain_priority_score",
				Type: "number",
				SQL: `
					CASE domain
						WHEN {{primary_domain}} THEN 100
						WHEN 'finance' THEN 80
						WHEN 'operations' THEN 60
						WHEN 'compliance' THEN 40
						ELSE 20
					END
				`,
				Parameters: []dynamic.DynamicParameter{
					{Name: "primary_domain", Type: "string"},
				},
			},
			{
				Name: "is_comparison_domain",
				Type: "boolean",
				SQL:  "domain = ANY(ARRAY[{{comparison_domains}}])",
				Parameters: []dynamic.DynamicParameter{
					{Name: "comparison_domains", Type: "array"},
				},
			},
		},
	}

	comparisonResolved, err := dynamicEngine.ResolveParameters(ctx, comparisonAnalysis)
	if err != nil {
		log.Fatalf("Failed to resolve comparison analysis: %v", err)
	}

	fmt.Printf("✅ Comparison Parameters: %+v\n", comparisonResolved.Parameters)
	fmt.Printf("🔄 Dynamic Comparison Measures: %+v\n", comparisonResolved.Metrics)

	fmt.Println("\n🎉 Dynamic Parameters & Measures Demo Complete!")
	fmt.Println("==============================================")
	fmt.Println("✅ Successfully demonstrated:")
	fmt.Println("   • Dynamic PoP metric analysis")
	fmt.Println("   • Real-time anomaly detection")
	fmt.Println("   • Steward review workflows")
	fmt.Println("   • Dashboard configuration")
	fmt.Println("   • Multi-metric comparisons")
	fmt.Println("   • Parameter validation and resolution")
	fmt.Println("   • Dynamic measure generation")
	fmt.Println("\n🚀 Your semantic layer now supports advanced dynamic querying!")
	fmt.Println("   Ready for integration with your PoP cockpit and anomaly detection engine.")
}
