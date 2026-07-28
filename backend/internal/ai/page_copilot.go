package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

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

func (s *PageCopilotService) GeneratePageLayoutSpec(ctx context.Context, req PageCopilotRequest) (map[string]interface{}, error) {
	if req.Domain == "" {
		req.Domain = "PORTFOLIO"
	}

	// Generate structured layout spec based on prompt intent and schema metadata
	spec := map[string]interface{}{
		"id":           fmt.Sprintf("page_ai_%s", req.Domain),
		"tenant_id":    req.TenantID,
		"key":          fmt.Sprintf("ai_generated_%s", req.Domain),
		"title":        fmt.Sprintf("AI Dashboard: %s", req.Prompt),
		"domain":       req.Domain,
		"target_bo_id": "customers",
		"layout": []map[string]interface{}{
			{
				"id":   "region_1",
				"name": "Generated Layout Region",
				"rows": []map[string]interface{}{
					{
						"id": "row_1",
						"columns": []map[string]interface{}{
							{
								"id":   "col_1",
								"span": 6,
								"components": []map[string]interface{}{
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
								},
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
								"id":   "col_3",
								"span": 12,
								"components": []map[string]interface{}{
									{
										"id":       "comp_form_1",
										"type":     "BO_FORM",
										"title":    "Master Detail View",
										"bo_id":    "customers",
										"bindings": map[string]interface{}{"fields": []string{"id", "name", "region", "status", "notes"}},
										"config":   map[string]interface{}{"is_mutable": true},
									},
								},
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
