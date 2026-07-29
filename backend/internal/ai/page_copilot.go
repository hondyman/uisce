package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/hondyman/uisce/libs/jwt-middleware"
	"github.com/jmoiron/sqlx"
)

type PageCopilotRequest struct {
	TenantID string `json:"tenant_id"`
	Prompt   string `json:"prompt"`
	Domain   string `json:"domain"`
}

type PageCopilotService struct {
	db *sqlx.DB
}

func NewPageCopilotService(db *sqlx.DB) *PageCopilotService {
	return &PageCopilotService{db: db}
}

type UserLayoutContext struct {
	FunctionalRole   string
	ClearanceLevel  string
	FrequentBOKeys  []string
	PinnedBOKeys    []string
	RecentFilters   []string
	PreferredDomain string
}

func (s *PageCopilotService) GeneratePersonalizedLayoutSpec(ctx context.Context, req PageCopilotRequest, uc *UserLayoutContext) (map[string]interface{}, error) {
	if req.Domain == "" && uc != nil && uc.PreferredDomain != "" {
		req.Domain = uc.PreferredDomain
	}
	if req.Domain == "" {
		req.Domain = "PORTFOLIO"
	}

	targetBO := "customers"
	if uc != nil && len(uc.FrequentBOKeys) > 0 {
		targetBO = uc.FrequentBOKeys[0]
	}

	isLookbackRequested := false
	lowerPrompt := strings.ToLower(req.Prompt)
	if strings.Contains(lowerPrompt, "as of") || strings.Contains(lowerPrompt, "historical") || strings.Contains(lowerPrompt, "lookback") || strings.Contains(lowerPrompt, "audit") || strings.Contains(lowerPrompt, "point in time") {
		isLookbackRequested = true
	}

	firstRowComponents := []map[string]interface{}{}

	if uc != nil && len(uc.PinnedBOKeys) > 0 {
		firstRowComponents = append(firstRowComponents, map[string]interface{}{
			"id":    "comp_pinned_bar",
			"type":  "PINNED_BO_BAR",
			"title": "Pinned Workspaces",
			"bo_keys": uc.PinnedBOKeys,
		})
	}

	firstRowComponents = append(firstRowComponents, map[string]interface{}{
		"id":       "comp_rec_bar_1",
		"type":     "AI_RECOMMENDATION_BAR",
		"title":    "AI Proactive Recommendations",
		"bo_id":    targetBO,
		"bindings": map[string]interface{}{"bo_keys": []string{targetBO}},
	})

	if uc != nil && (uc.FunctionalRole == "compliance_officer" || uc.FunctionalRole == "data_steward") {
		firstRowComponents = append(firstRowComponents, map[string]interface{}{
			"id":    "comp_audit_trail",
			"type":  "ROLE_AUDIT_TRAIL",
			"title": "Maker-Checker & Governance Status",
			"bo_id": targetBO,
		})
	}

	if uc != nil && uc.FunctionalRole == "trader" || uc != nil && uc.FunctionalRole == "portfolio_manager" {
		firstRowComponents = append(firstRowComponents, map[string]interface{}{
			"id":    "comp_rebalance_insight",
			"type":  "REBALANCE_INSIGHT",
			"title": "Rebalance Opportunities",
			"bo_id": targetBO,
		})
	}

	firstRowComponents = append(firstRowComponents, map[string]interface{}{
		"id":       "comp_chart_1",
		"type":     "BO_ANALYTICS_CHART",
		"title":    "Breakdown by Asset Class",
		"bo_id":    targetBO,
		"bindings": map[string]interface{}{"dimensions": []string{"region"}, "measures": []string{"balance"}},
		"interactions": map[string]interface{}{
			"emits_context": []map[string]string{{"source_field": "region", "target_context_key": "selected_region"}},
		},
		"config": map[string]interface{}{"chartType": "bar"},
	})

	secondRowComponents := []map[string]interface{}{
		{
			"id":       "comp_form_1",
			"type":     "BO_FORM",
			"title":    "Master Detail View",
			"bo_id":    targetBO,
			"bindings": map[string]interface{}{"fields": []string{"id", "name", "region", "status", "notes"}},
			"config":   map[string]interface{}{"is_mutable": true},
		},
	}

	if isLookbackRequested {
		secondRowComponents = append(secondRowComponents, map[string]interface{}{
			"id":       "comp_lookback_manager_1",
			"type":     "BO_LOOKBACK_MANAGER",
			"title":    "Compliance Lookback Audit Diff Matrix",
			"bo_id":    targetBO,
			"bindings": map[string]interface{}{"fields": []string{"id", "balance", "status"}},
			"config": map[string]interface{}{
				"timestamp_a": "2025-12-31T00:00:00Z",
				"timestamp_b": "2026-06-30T00:00:00Z",
			},
		})
	}

	spec := map[string]interface{}{
		"id":           fmt.Sprintf("page_pers_%s_%s", req.Domain, targetBO),
		"tenant_id":    req.TenantID,
		"key":          fmt.Sprintf("ai_generated_%s", req.Domain),
		"title":        fmt.Sprintf("AI Dashboard: %s", req.Prompt),
		"domain":       req.Domain,
		"target_bo_id": targetBO,
		"lookback": map[string]interface{}{
			"enabled":         isLookbackRequested,
			"as_of_timestamp": "2025-12-31T23:59:59Z",
		},
		"layout": []map[string]interface{}{
			{
				"id":   "region_1",
				"name": "Generated Layout Region",
				"rows": []map[string]interface{}{
					{
						"id": "row_1",
						"columns": []map[string]interface{}{
							{
								"id":         "col_1",
								"span":       12,
								"components": firstRowComponents,
							},
						},
					},
					{
						"id": "row_2",
						"columns": []map[string]interface{}{
							{
								"id":         "col_3",
								"span":       12,
								"components": secondRowComponents,
							},
						},
					},
				},
			},
		},
		"rules": []map[string]interface{}{
			{
				"id":        "rule_pending",
				"name":      "Disable if status is PENDING",
				"condition": map[string]interface{}{"field": "selected_account_status", "operator": "EQUALS", "value": "PENDING"},
				"actions":   []map[string]interface{}{{"target_component_id": "comp_form_1", "effect": "DISABLE"}},
			},
		},
	}

	return spec, nil
}

func (s *PageCopilotService) GeneratePageLayoutSpec(ctx context.Context, req PageCopilotRequest) (map[string]interface{}, error) {
	if req.Domain == "" {
		req.Domain = "PORTFOLIO"
	}

	// Detect temporal / lookback intent in user prompt
	isLookbackRequested := false
	lowerPrompt := strings.ToLower(req.Prompt)
	if strings.Contains(lowerPrompt, "as of") || strings.Contains(lowerPrompt, "historical") || strings.Contains(lowerPrompt, "lookback") || strings.Contains(lowerPrompt, "audit") || strings.Contains(lowerPrompt, "point in time") {
		isLookbackRequested = true
	}

	// Build component list dynamically based on prompt intent
	firstRowComponents := []map[string]interface{}{
		{
			"id":       "comp_rec_bar_1",
			"type":     "AI_RECOMMENDATION_BAR",
			"title":    "AI Proactive Recommendations",
			"bo_id":    "customers",
			"bindings": map[string]interface{}{"bo_keys": []string{"customers"}},
		},
		{
			"id":       "comp_chart_1",
			"type":     "BO_ANALYTICS_CHART",
			"title":    "Breakdown by Asset Class",
			"bo_id":    "customers",
			"bindings": map[string]interface{}{"dimensions": []string{"region"}, "measures": []string{"balance"}},
			"interactions": map[string]interface{}{
				"emits_context": []map[string]string{{"source_field": "region", "target_context_key": "selected_region"}},
			},
			"config": map[string]interface{}{"chartType": "bar"},
		},
	}

	secondRowComponents := []map[string]interface{}{
		{
			"id":       "comp_form_1",
			"type":     "BO_FORM",
			"title":    "Master Detail View",
			"bo_id":    "customers",
			"bindings": map[string]interface{}{"fields": []string{"id", "name", "region", "status", "notes"}},
			"config":   map[string]interface{}{"is_mutable": true},
		},
	}

	if isLookbackRequested {
		secondRowComponents = append(secondRowComponents, map[string]interface{}{
			"id":       "comp_lookback_manager_1",
			"type":     "BO_LOOKBACK_MANAGER",
			"title":    "Compliance Lookback Audit Diff Matrix",
			"bo_id":    "customers",
			"bindings": map[string]interface{}{"fields": []string{"id", "balance", "status"}},
			"config": map[string]interface{}{
				"timestamp_a": "2025-12-31T00:00:00Z",
				"timestamp_b": "2026-06-30T00:00:00Z",
			},
		})
	}

	spec := map[string]interface{}{
		"id":           fmt.Sprintf("page_ai_%s", req.Domain),
		"tenant_id":    req.TenantID,
		"key":          fmt.Sprintf("ai_generated_%s", req.Domain),
		"title":        fmt.Sprintf("AI Dashboard: %s", req.Prompt),
		"domain":       req.Domain,
		"target_bo_id": "customers",
		"lookback": map[string]interface{}{
			"enabled":         isLookbackRequested,
			"as_of_timestamp": "2025-12-31T23:59:59Z",
		},
		"layout": []map[string]interface{}{
			{
				"id":   "region_1",
				"name": "Generated Layout Region",
				"rows": []map[string]interface{}{
					{
						"id": "row_1",
						"columns": []map[string]interface{}{
							{
								"id":         "col_1",
								"span":       6,
								"components": firstRowComponents,
							},
							{
								"id":   "col_2",
								"span": 6,
								"components": []map[string]interface{}{
									{
										"id":       "comp_grid_1",
										"type":     "BO_GRID",
										"title":    "Accounts Listing",
										"bo_id":    "customers",
										"bindings": map[string]interface{}{"fields": []string{"id", "name", "region", "status", "balance"}},
										"interactions": map[string]interface{}{
											"emits_context": []map[string]string{
												{"source_field": "id", "target_context_key": "selected_account_id"},
												{"source_field": "status", "target_context_key": "selected_account_status"},
											},
											"subscribes_to_context": []map[string]string{
												{"context_key": "selected_region", "filter_field": "region", "operator": "EQ"},
											},
										},
										"config": map[string]interface{}{},
									},
								},
							},
						},
					},
					{
						"id": "row_2",
						"columns": []map[string]interface{}{
							{
								"id":         "col_3",
								"span":       12,
								"components": secondRowComponents,
							},
						},
					},
				},
			},
		},
		"rules": []map[string]interface{}{
			{
				"id":        "rule_pending",
				"name":      "Disable if status is PENDING",
				"condition": map[string]interface{}{"field": "selected_account_status", "operator": "EQUALS", "value": "PENDING"},
				"actions":   []map[string]interface{}{{"target_component_id": "comp_form_1", "effect": "DISABLE"}},
			},
		},
	}

	return spec, nil
}

func (s *PageCopilotService) GeneratePageHandler(w http.ResponseWriter, r *http.Request) {
	var req PageCopilotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if claims := jwtmiddleware.GetClaimsFromContext(r); claims != nil && claims.TenantID != "" {
		req.TenantID = claims.TenantID
	}
	if req.TenantID == "" {
		req.TenantID = "core"
	}

	spec, err := s.GeneratePageLayoutSpec(r.Context(), req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate page layout: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(spec)
}
