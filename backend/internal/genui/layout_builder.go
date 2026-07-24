package genui

import (
	"context"
	"fmt"
)

// ComponentDef represents a GenUI component definition
type ComponentDef struct {
	ID       string         `json:"id"`
	Type     string         `json:"type"`
	Title    string         `json:"title,omitempty"`
	Subtitle string         `json:"subtitle,omitempty"`
	Binding  *QueryBinding  `json:"binding,omitempty"`
	Config   map[string]any `json:"config,omitempty"` // Type-specific config
}

// QueryBinding represents a GraphQL query binding
type QueryBinding struct {
	GQL       string         `json:"gql"`
	Variables map[string]any `json:"variables"`
	DataPath  string         `json:"data_path"`
}

// LayoutDef represents a complete layout
type LayoutDef struct {
	Version    int            `json:"version"`
	Title      string         `json:"title,omitempty"`
	Layout     string         `json:"layout,omitempty"` // grid, flex, stack
	Components []ComponentDef `json:"components"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

// LayoutBuilder generates UI layouts from intents
type LayoutBuilder struct {
	// In production, this would have access to metadata registry
}

func NewLayoutBuilder() *LayoutBuilder {
	return &LayoutBuilder{}
}

// Build generates a layout from an intent
func (lb *LayoutBuilder) Build(ctx context.Context, intent *Intent) (*LayoutDef, error) {
	layout := &LayoutDef{
		Version: 1,
		Layout:  "grid",
		Components: []ComponentDef{},
		Metadata: map[string]any{
			"generated_by": "layout_builder",
			"intent":       intent.Type,
		},
	}

	switch intent.Type {
	case "dashboard":
		return lb.buildDashboard(ctx, intent, layout)
	case "chart":
		return lb.buildChart(ctx, intent, layout)
	case "grid":
		return lb.buildGrid(ctx, intent, layout)
	case "approval_inbox":
		return lb.buildApprovalInbox(ctx, intent, layout)
	case "rebalancing_proposal":
		return lb.buildRebalancingProposal(ctx, intent, layout)
	default:
		return lb.buildDashboard(ctx, intent, layout)
	}
}

func (lb *LayoutBuilder) buildRebalancingProposal(ctx context.Context, intent *Intent, layout *LayoutDef) (*LayoutDef, error) {
	layout.Title = "Rebalancing Proposal"
	layout.Layout = "stack"

	// In a real implementation, we would fetch the proposal data here or bind it to a query
	// For the GenUI demo, we'll embed the full mock payload here
	
	mockProposal := map[string]any{
		"proposal_id":  "prop_genui_001",
		"portfolio_id": "pf_892",
		"generated_at": "2025-11-23T12:00:00Z",
		"advisor_view": map[string]any{
			"title":                 "Reduce US Equity overweight; harvest BND losses",
			"summary":               "Sell 50 IVV to lower drift; sell 100 BND (loss) to offset gains; buy AGG to maintain fixed-income exposure.",
			"tracking_error_before": 2.0,
			"tracking_error_after":  1.4,
			"tax_impact_usd":        -1800.0,
			"disclosures": []string{
				"Rebalancing may trigger transaction costs",
				"Loss harvesting subject to wash-sale rules",
			},
			"monte_carlo": map[string]any{
				"mean":             -1750.0,
				"median":           -1800.0,
				"pct05":            -1200.0,
				"pct95":            -2200.0,
				"confidence80_min": -1500.0,
				"confidence80_max": -2000.0,
				"runs":             1000,
			},
		},
		"orders": []map[string]any{
			{
				"side":          "SELL",
				"symbol":        "IVV",
				"qty":           50,
				"est_value_usd": 22500,
				"lots":          []any{},
				"reason":        "reduce_overweight",
			},
			{
				"side":          "SELL",
				"symbol":        "BND",
				"qty":           100,
				"est_value_usd": 7000,
				"lots": []map[string]any{
					{"lot_id": "bnd_l1", "term": "long", "unrealized_pnl": -1200},
				},
				"reason": "harvest_loss",
			},
			{
				"side":          "BUY",
				"symbol":        "AGG",
				"qty":           100,
				"est_value_usd": 10000,
				"lots":          []any{},
				"reason":        "replacement_buy",
			},
		},
		"citations": []map[string]any{
			{
				"id":          "C1",
				"source":      "snap_positions_001",
				"snapshot_id": "snap_20251123_positions",
				"excerpt":     "Current weights: IVV 35%, target 30%.",
			},
		},
		"actions": map[string]any{
			"approve": map[string]string{"label": "Approve and execute"},
			"reject":  map[string]string{"label": "Reject"},
			"clarify": map[string]string{"label": "Request clarification"},
		},
	}

	layout.Components = append(layout.Components, ComponentDef{
		ID:     "proposal_card",
		Type:   "proposal_card",
		Config: mockProposal,
	})

	return layout, nil
}

func (lb *LayoutBuilder) buildDashboard(ctx context.Context, intent *Intent, layout *LayoutDef) (*LayoutDef, error) {
	layout.Title = "Dashboard"

	// Add KPI cards for key metrics
	for i, metric := range intent.Metrics {
		layout.Components = append(layout.Components, ComponentDef{
			ID:    fmt.Sprintf("card_%d", i),
			Type:  "card",
			Title: formatMetricName(metric),
			Config: map[string]any{
				"variant": "kpi",
				"value":   metric,
			},
		})
	}

	// Add chart for primary object
	if len(intent.Objects) > 0 {
		chartComp := lb.createChartComponent(intent.Objects[0], intent.Metrics[0], intent.TimeRange)
		layout.Components = append(layout.Components, chartComp)
	}

	// Add grid for detailed data
	if len(intent.Objects) > 0 {
		gridComp := lb.createGridComponent(intent.Objects[0])
		layout.Components = append(layout.Components, gridComp)
	}

	return layout, nil
}

func (lb *LayoutBuilder) buildChart(ctx context.Context, intent *Intent, layout *LayoutDef) (*LayoutDef, error) {
	if len(intent.Objects) == 0 || len(intent.Metrics) == 0 {
		return nil, fmt.Errorf("chart requires at least one object and metric")
	}

	layout.Title = fmt.Sprintf("%s %s", intent.Objects[0], formatMetricName(intent.Metrics[0]))
	
	chartComp := lb.createChartComponent(intent.Objects[0], intent.Metrics[0], intent.TimeRange)
	layout.Components = append(layout.Components, chartComp)

	return layout, nil
}

func (lb *LayoutBuilder) buildGrid(ctx context.Context, intent *Intent, layout *LayoutDef) (*LayoutDef, error) {
	if len(intent.Objects) == 0 {
		return nil, fmt.Errorf("grid requires at least one object")
	}

	layout.Title = fmt.Sprintf("%s List", intent.Objects[0])
	
	gridComp := lb.createGridComponent(intent.Objects[0])
	layout.Components = append(layout.Components, gridComp)

	return layout, nil
}

func (lb *LayoutBuilder) createChartComponent(object string, metric string, timeRange *TimeRange) ComponentDef {
	// Build GraphQL query
	gqlQuery := fmt.Sprintf(`
		query Get%sPerformance($timeRange: String) {
			%sPerformance(timeRange: $timeRange) {
				date
				%s
			}
		}
	`, object, lowerFirst(object), metric)

	variables := map[string]any{}
	if timeRange != nil {
		variables["timeRange"] = timeRange.Start
	}

	return ComponentDef{
		ID:    "main_chart",
		Type:  "chart",
		Title: fmt.Sprintf("%s Over Time", formatMetricName(metric)),
		Binding: &QueryBinding{
			GQL:       gqlQuery,
			Variables: variables,
			DataPath:  fmt.Sprintf("data.%sPerformance", lowerFirst(object)),
		},
		Config: map[string]any{
			"chartType": "line",
			"xField":    "date",
			"yFields":   []string{metric},
			"legend":    true,
		},
	}
}

func (lb *LayoutBuilder) createGridComponent(object string) ComponentDef {
	gqlQuery := fmt.Sprintf(`
		query Get%sList {
			%sList {
				id
				name
				value
				updated_at
			}
		}
	`, object, lowerFirst(object))

	return ComponentDef{
		ID:    "main_grid",
		Type:  "grid",
		Title: fmt.Sprintf("%s List", object),
		Binding: &QueryBinding{
			GQL:      gqlQuery,
			DataPath: fmt.Sprintf("data.%sList", lowerFirst(object)),
		},
		Config: map[string]any{
			"columns": []map[string]any{
				{"field": "name", "headerName": "Name", "width": 200},
				{"field": "value", "headerName": "Value", "type": "currency"},
				{"field": "updated_at", "headerName": "Updated", "type": "date"},
			},
			"pagination": map[string]any{
				"enabled":  true,
				"pageSize": 20,
			},
		},
	}
}

func formatMetricName(metric string) string {
	// Convert snake_case to Title Case
	parts := []rune(metric)
	result := []rune{}
	capitalize := true
	
	for _, r := range parts {
		if r == '_' {
			result = append(result, ' ')
			capitalize = true
		} else if capitalize {
			result = append(result, rune(r-32))
			capitalize = false
		} else {
			result = append(result, r)
		}
	}
	
	return string(result)
}

func lowerFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return string(rune(s[0])+32) + s[1:]
}

func (lb *LayoutBuilder) buildApprovalInbox(ctx context.Context, intent *Intent, layout *LayoutDef) (*LayoutDef, error) {
	layout.Title = "Approval Inbox"
	layout.Layout = "flex"

	// Timeline widget showing pending approvals
	layout.Components = append(layout.Components, ComponentDef{
		ID:    "approval_timeline",
		Type:  "timeline",
		Title: "Pending Approvals",
		Config: map[string]any{
			"variant":     "approval_list",
			"data_source": "temporal_workflows",
			"filters": map[string]any{
				"status": "awaiting_signal",
				"signal": "AdvisorApproval",
			},
		},
	})

	// Approval decision form
	layout.Components = append(layout.Components, ComponentDef{
		ID:    "approval_form",
		Type:  "form",
		Title: "Decision",
		Config: map[string]any{
			"fields": []map[string]any{
				{
					"name":        "rationale",
					"type":        "textarea",
					"label":       "Decision Rationale",
					"placeholder": "Enter your decision rationale (required)...",
					"required":    true,
					"rows":        3,
				},
			},
			"actions": []map[string]any{
				{
					"id":      "approve",
					"label":   "Approve",
					"variant": "success",
					"signal":  "AdvisorApproval",
					"payload": map[string]any{"approved": true},
				},
				{
					"id":      "reject",
					"label":   "Reject",
					"variant": "danger",
					"signal":  "AdvisorApproval",
					"payload": map[string]any{"approved": false},
				},
				{
					"id":      "delegate",
					"label":   "Delegate",
					"variant": "secondary",
				},
			},
		},
	})

	// Audit card showing compliance info
	layout.Components = append(layout.Components, ComponentDef{
		ID:    "audit_card",
		Type:  "card",
		Title: "Audit Linkage",
		Config: map[string]any{
			"variant": "audit",
			"fields": []map[string]any{
				{"label": "Hash", "field": "uar_hash"},
				{"label": "Signature", "field": "signature"},
				{"label": "Policy Ref", "field": "policy_id"},
			},
		},
	})

	// Disclosure banner
	layout.Components = append(layout.Components, ComponentDef{
		ID:    "disclosure",
		Type:  "disclosure",
		Title: "Compliance Notice",
		Config: map[string]any{
			"message": "All approval decisions are recorded in the Universal Audit Record (UAR) and are subject to regulatory review.",
			"variant": "info",
		},
	})

	return layout, nil
}
