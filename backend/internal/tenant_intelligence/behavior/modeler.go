package behavior

import (
	"context"
)

type BehaviorPattern struct {
	Type        string   `json:"type"` // page_usage, workflow, data_access, performance
	Description string   `json:"description"`
	Frequency   int      `json:"frequency"`
	Evidence    []string `json:"evidence"`
}

type TenantBehaviorModel struct {
	TenantID            string            `json:"tenant_id"`
	PagePatterns        []BehaviorPattern `json:"page_patterns"`
	WorkflowPatterns    []BehaviorPattern `json:"workflow_patterns"`
	DataPatterns        []BehaviorPattern `json:"data_patterns"`
	PerformancePatterns []BehaviorPattern `json:"performance_patterns"`
}

type Suggestion struct {
	Type        string `json:"type"` // new_page, new_workflow, new_kpi, new_preagg
	Title       string `json:"title"`
	Description string `json:"description"`
	Rationale   string `json:"rationale"`
	Priority    string `json:"priority"` // high, medium, low
}

type BehaviorModeler struct{}

func NewBehaviorModeler() *BehaviorModeler {
	return &BehaviorModeler{}
}

func (bm *BehaviorModeler) Model(ctx context.Context, tenantID string) (*TenantBehaviorModel, error) {
	// Mock: Generate behavior model
	// Real: Analyze page usage, workflow execution, data access, performance metrics

	model := &TenantBehaviorModel{
		TenantID: tenantID,
		PagePatterns: []BehaviorPattern{
			{
				Type:        "page_usage",
				Description: "Positions Dashboard used heavily",
				Frequency:   847,
				Evidence:    []string{"847 views in last 7 days", "Average session: 12 minutes", "Peak usage: 9-11 AM EST"},
			},
			{
				Type:        "page_usage",
				Description: "Risk section rarely used",
				Frequency:   12,
				Evidence:    []string{"12 views in last 30 days", "No views in last 7 days"},
			},
		},
		WorkflowPatterns: []BehaviorPattern{
			{
				Type:        "workflow",
				Description: "Trade Approval workflow used frequently",
				Frequency:   234,
				Evidence:    []string{"234 executions in last 7 days", "Average completion: 45 minutes", "Drop-off at step 2: 8%"},
			},
			{
				Type:        "workflow",
				Description: "Users repeatedly perform: filter positions → export → email",
				Frequency:   127,
				Evidence:    []string{"127 occurrences in last 7 days", "Always in this sequence", "Average time: 3 minutes"},
			},
		},
		DataPatterns: []BehaviorPattern{
			{
				Type:        "data_access",
				Description: "Positions filtered by risk frequently",
				Frequency:   342,
				Evidence:    []string{"342 queries with risk filter", "Filter: risk_level > 'medium'"},
			},
			{
				Type:        "data_access",
				Description: "Positions queried by region repeatedly",
				Frequency:   189,
				Evidence:    []string{"189 queries grouping by region", "No pre-agg available"},
			},
		},
		PerformancePatterns: []BehaviorPattern{
			{
				Type:        "performance",
				Description: "Positions Dashboard SLO pressure",
				Frequency:   42,
				Evidence:    []string{"42 SLO near-violations in last 7 days", "p95 latency: 285ms (threshold: 300ms)"},
			},
		},
	}

	return model, nil
}

func (bm *BehaviorModeler) Suggest(ctx context.Context, model *TenantBehaviorModel) ([]Suggestion, error) {
	// Mock: Generate suggestions
	// Real: Analyze behavior model and propose improvements

	suggestions := []Suggestion{
		{
			Type:        "new_page",
			Title:       "Risk Dashboard",
			Description: "Create a dedicated Risk Dashboard with risk-filtered positions",
			Rationale:   "Tenant frequently filters positions by risk (342 queries in last 7 days)",
			Priority:    "high",
		},
		{
			Type:        "new_workflow",
			Title:       "Export & Email Positions",
			Description: "Create workflow: filter positions → export → email",
			Rationale:   "Users repeatedly perform this sequence (127 times in last 7 days)",
			Priority:    "medium",
		},
		{
			Type:        "new_kpi",
			Title:       "Positions Volatility KPI",
			Description: "Add volatility KPI to Positions Dashboard",
			Rationale:   "Positions volatility is a top concern based on filter patterns",
			Priority:    "medium",
		},
		{
			Type:        "new_preagg",
			Title:       "positions_by_region",
			Description: "Create pre-aggregation grouping positions by region",
			Rationale:   "Tenant repeatedly queries positions by region (189 queries, no pre-agg available)",
			Priority:    "high",
		},
	}

	return suggestions, nil
}
