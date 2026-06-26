package rdl

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handler provides HTTP handlers for the RDL service
type Handler struct {
	service *RDLService
}

// NewHandler creates a new RDL handler
func NewHandler(service *RDLService) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers all RDL routes
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/api/rules", func(r chi.Router) {
		r.Get("/", h.ListRules)
		r.Post("/", h.CreateRule)
		r.Get("/{ruleID}", h.GetRule)
		r.Put("/{ruleID}", h.UpdateRule)
		r.Delete("/{ruleID}", h.DeleteRule)
		r.Post("/evaluate", h.EvaluateRule)
		r.Post("/evaluate-batch", h.EvaluateBatch)
	})
}

// ListRules returns all rules for a tenant
// GET /api/rules?tenant_id=xxx&type=tax_loss_harvesting
func (h *Handler) ListRules(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantIDStr := r.URL.Query().Get("tenant_id")
	if tenantIDStr == "" {
		http.Error(w, "tenant_id is required", http.StatusBadRequest)
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		http.Error(w, "invalid tenant_id", http.StatusBadRequest)
		return
	}

	ruleType := r.URL.Query().Get("type")

	var rules []RuleDefinition
	if ruleType != "" {
		rules, err = h.service.GetRulesByType(ctx, tenantID, RuleType(ruleType))
	} else {
		rules, err = h.service.GetRulesByTenant(ctx, tenantID)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"rules": rules,
		"count": len(rules),
	})
}

// GetRule returns a specific rule
// GET /api/rules/{ruleID}?tenant_id=xxx
func (h *Handler) GetRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantIDStr := r.URL.Query().Get("tenant_id")
	if tenantIDStr == "" {
		http.Error(w, "tenant_id is required", http.StatusBadRequest)
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		http.Error(w, "invalid tenant_id", http.StatusBadRequest)
		return
	}

	ruleID := chi.URLParam(r, "ruleID")
	if ruleID == "" {
		http.Error(w, "rule_id is required", http.StatusBadRequest)
		return
	}

	rule, err := h.service.GetRuleByID(ctx, tenantID, ruleID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rule)
}

// CreateRuleRequest represents the request body for creating a rule
type CreateRuleRequest struct {
	TenantID             string          `json:"tenant_id"`
	RuleID               string          `json:"rule_id"`
	Type                 string          `json:"type"`
	Version              string          `json:"version"`
	Name                 string          `json:"name"`
	Description          string          `json:"description"`
	Jurisdiction         string          `json:"jurisdiction,omitempty"`
	Parameters           json.RawMessage `json:"parameters"`
	Expression           string          `json:"expression"`
	ScoringFormula       string          `json:"scoring_formula,omitempty"`
	WashSaleConfig       json.RawMessage `json:"wash_sale_config,omitempty"`
	SubstituteAssetRules json.RawMessage `json:"substitute_asset_rules,omitempty"`
	Schedule             json.RawMessage `json:"schedule,omitempty"`
	Notifications        json.RawMessage `json:"notifications,omitempty"`
	Active               bool            `json:"active"`
	EffectiveFrom        string          `json:"effective_from,omitempty"`
	EffectiveTo          string          `json:"effective_to,omitempty"`
}

// CreateRule creates a new rule
// POST /api/rules
func (h *Handler) CreateRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		http.Error(w, "invalid tenant_id", http.StatusBadRequest)
		return
	}

	rule := &RuleDefinition{
		TenantID:             tenantID,
		RuleID:               req.RuleID,
		Type:                 RuleType(req.Type),
		Version:              req.Version,
		Name:                 req.Name,
		Description:          req.Description,
		Jurisdiction:         req.Jurisdiction,
		Parameters:           req.Parameters,
		Expression:           req.Expression,
		ScoringFormula:       req.ScoringFormula,
		WashSaleConfig:       req.WashSaleConfig,
		SubstituteAssetRules: req.SubstituteAssetRules,
		Schedule:             req.Schedule,
		Notifications:        req.Notifications,
		Active:               req.Active,
	}

	if err := h.service.CreateRule(ctx, rule); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(rule)
}

// UpdateRule updates an existing rule
// PUT /api/rules/{ruleID}
func (h *Handler) UpdateRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	ruleID := chi.URLParam(r, "ruleID")
	if ruleID == "" {
		http.Error(w, "rule_id is required", http.StatusBadRequest)
		return
	}

	var req CreateRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		http.Error(w, "invalid tenant_id", http.StatusBadRequest)
		return
	}

	rule := &RuleDefinition{
		TenantID:             tenantID,
		RuleID:               ruleID,
		Type:                 RuleType(req.Type),
		Version:              req.Version,
		Name:                 req.Name,
		Description:          req.Description,
		Jurisdiction:         req.Jurisdiction,
		Parameters:           req.Parameters,
		Expression:           req.Expression,
		ScoringFormula:       req.ScoringFormula,
		WashSaleConfig:       req.WashSaleConfig,
		SubstituteAssetRules: req.SubstituteAssetRules,
		Schedule:             req.Schedule,
		Notifications:        req.Notifications,
		Active:               req.Active,
	}

	if err := h.service.UpdateRule(ctx, rule); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rule)
}

// DeleteRule deactivates a rule
// DELETE /api/rules/{ruleID}?tenant_id=xxx
func (h *Handler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantIDStr := r.URL.Query().Get("tenant_id")
	if tenantIDStr == "" {
		http.Error(w, "tenant_id is required", http.StatusBadRequest)
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		http.Error(w, "invalid tenant_id", http.StatusBadRequest)
		return
	}

	ruleID := chi.URLParam(r, "ruleID")
	if ruleID == "" {
		http.Error(w, "rule_id is required", http.StatusBadRequest)
		return
	}

	if err := h.service.DeactivateRule(ctx, tenantID, ruleID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// EvaluateRuleRequest represents the request body for rule evaluation
type EvaluateRuleRequest struct {
	TenantID string           `json:"tenant_id"`
	RuleID   string           `json:"rule_id"`
	Input    *EvaluationInput `json:"input"`
}

// EvaluateRule evaluates a specific rule against input
// POST /api/rules/evaluate
func (h *Handler) EvaluateRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req EvaluateRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		http.Error(w, "invalid tenant_id", http.StatusBadRequest)
		return
	}

	rule, err := h.service.GetRuleByID(ctx, tenantID, req.RuleID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	result, err := h.service.Evaluate(ctx, rule, req.Input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// EvaluateBatchRequest represents the request body for batch evaluation
type EvaluateBatchRequest struct {
	TenantID string           `json:"tenant_id"`
	RuleType string           `json:"rule_type"`
	Input    *EvaluationInput `json:"input"`
}

// EvaluateBatch evaluates all rules of a type against input
// POST /api/rules/evaluate-batch
func (h *Handler) EvaluateBatch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req EvaluateBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		http.Error(w, "invalid tenant_id", http.StatusBadRequest)
		return
	}

	results, err := h.service.EvaluateAll(ctx, tenantID, RuleType(req.RuleType), req.Input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Separate passed vs not passed
	var opportunities []EvaluationResult
	var passed []EvaluationResult
	for _, result := range results {
		if result.Passed {
			opportunities = append(opportunities, result)
		} else {
			passed = append(passed, result)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"opportunities":       opportunities,
		"passed_rules":        passed,
		"total_evaluated":     len(results),
		"opportunities_found": len(opportunities),
	})
}

// EvaluatePortfolioRequest represents the request body for portfolio evaluation
type EvaluatePortfolioRequest struct {
	TenantID    string           `json:"tenant_id"`
	PortfolioID string           `json:"portfolio_id"`
	Input       *EvaluationInput `json:"input"`
}

// EvaluatePortfolio evaluates all applicable rules for a portfolio
// POST /api/rdl/evaluate/portfolio
func (h *Handler) EvaluatePortfolio(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req EvaluatePortfolioRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		http.Error(w, "invalid tenant_id", http.StatusBadRequest)
		return
	}

	// Get all active rules for tenant
	rules, err := h.service.GetRulesByTenant(ctx, tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var results []map[string]interface{}
	for _, rule := range rules {
		if !rule.Active {
			continue
		}

		result, err := h.service.Evaluate(ctx, &rule, req.Input)
		if err != nil {
			continue
		}

		results = append(results, map[string]interface{}{
			"rule_id":     rule.RuleID,
			"rule_name":   rule.Name,
			"rule_type":   rule.Type,
			"passed":      result.Passed,
			"score":       result.Score,
			"opportunity": result,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"portfolio_id": req.PortfolioID,
		"results":      results,
		"total_rules":  len(rules),
	})
}

// EvaluateTLHRequest represents the request body for TLH evaluation
type EvaluateTLHRequest struct {
	TenantID    string           `json:"tenant_id"`
	PortfolioID string           `json:"portfolio_id"`
	Input       *EvaluationInput `json:"input"`
}

// EvaluateTLH evaluates Tax-Loss Harvesting rules specifically
// POST /api/rdl/evaluate/tlh
func (h *Handler) EvaluateTLH(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req EvaluateTLHRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		http.Error(w, "invalid tenant_id", http.StatusBadRequest)
		return
	}

	// Get only TLH rules
	rules, err := h.service.GetRulesByType(ctx, tenantID, RuleTypeTLH)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var opportunities []map[string]interface{}
	for _, rule := range rules {
		if !rule.Active {
			continue
		}

		result, err := h.service.Evaluate(ctx, &rule, req.Input)
		if err != nil {
			continue
		}

		if result.Passed {
			opportunities = append(opportunities, map[string]interface{}{
				"rule_id":        rule.RuleID,
				"rule_name":      rule.Name,
				"jurisdiction":   rule.Jurisdiction,
				"ticker":         req.Input.Ticker,
				"unrealized_pnl": req.Input.UnrealizedLossUSD,
				"score":          result.Score,
				"wash_sale_safe": result.Metadata["wash_sale_safe"],
				"substitute":     result.Metadata["substitute_ticker"],
				"actions":        result.Actions,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"portfolio_id":        req.PortfolioID,
		"opportunities":       opportunities,
		"opportunities_found": len(opportunities),
	})
}

// RuleTemplate represents a pre-defined rule template
type RuleTemplate struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Type        string          `json:"type"`
	Schema      json.RawMessage `json:"schema"`
	Example     json.RawMessage `json:"example"`
}

// ListTemplates returns available rule templates
// GET /api/rdl/templates
func (h *Handler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	templates := []RuleTemplate{
		{
			ID:          "tlh_us_standard",
			Name:        "US Tax-Loss Harvesting",
			Description: "Standard 30-day wash sale rule for US equities",
			Type:        "tax_loss_harvesting",
		},
		{
			ID:          "tlh_uk_bed_breakfast",
			Name:        "UK Bed & Breakfast Rule",
			Description: "30-day rule for UK capital gains",
			Type:        "tax_loss_harvesting",
		},
		{
			ID:          "cppi_floor",
			Name:        "CPPI Floor Protection",
			Description: "Constant Proportion Portfolio Insurance floor rule",
			Type:        "cppi",
		},
		{
			ID:          "drift_rebalance",
			Name:        "Drift-Based Rebalancing",
			Description: "Trigger rebalance when allocation drift exceeds threshold",
			Type:        "drift_constraint",
		},
		{
			ID:          "concentration_limit",
			Name:        "Concentration Limit",
			Description: "Maximum position concentration per asset",
			Type:        "portfolio_constraint",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"templates": templates,
		"count":     len(templates),
	})
}

// GetTemplate returns a specific rule template with full schema
// GET /api/rdl/templates/{templateID}
func (h *Handler) GetTemplate(w http.ResponseWriter, r *http.Request) {
	templateID := chi.URLParam(r, "templateID")
	if templateID == "" {
		http.Error(w, "template_id is required", http.StatusBadRequest)
		return
	}

	// Template schemas - in production these would come from a DB or config
	templates := map[string]RuleTemplate{
		"tlh_us_standard": {
			ID:          "tlh_us_standard",
			Name:        "US Tax-Loss Harvesting",
			Description: "Standard 30-day wash sale rule for US equities",
			Type:        "tax_loss_harvesting",
			Schema:      json.RawMessage(`{"properties":{"min_loss":{"type":"number"},"wash_sale_days":{"type":"integer"},"substitute_correlation":{"type":"number"}}}`),
			Example:     json.RawMessage(`{"min_loss":1000,"wash_sale_days":30,"substitute_correlation":0.95}`),
		},
		"cppi_floor": {
			ID:          "cppi_floor",
			Name:        "CPPI Floor Protection",
			Description: "Constant Proportion Portfolio Insurance floor rule",
			Type:        "cppi",
			Schema:      json.RawMessage(`{"properties":{"floor_percentage":{"type":"number"},"multiplier":{"type":"number"},"risk_free_ticker":{"type":"string"}}}`),
			Example:     json.RawMessage(`{"floor_percentage":0.80,"multiplier":4,"risk_free_ticker":"SHY"}`),
		},
	}

	template, ok := templates[templateID]
	if !ok {
		http.Error(w, "template not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(template)
}

// ValidateRuleRequest represents the request body for rule validation
type ValidateRuleRequest struct {
	Expression string          `json:"expression"`
	Parameters json.RawMessage `json:"parameters"`
	TestInput  json.RawMessage `json:"test_input,omitempty"`
}

// ValidateRule validates a rule expression without saving
// POST /api/rdl/validate
func (h *Handler) ValidateRule(w http.ResponseWriter, r *http.Request) {
	var req ValidateRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Try to compile the CEL expression
	isValid, errors := h.service.ValidateExpression(req.Expression)

	response := map[string]interface{}{
		"valid":      isValid,
		"expression": req.Expression,
	}

	if !isValid {
		response["errors"] = errors
	}

	// If valid and test input provided, do a test evaluation
	if isValid && len(req.TestInput) > 0 {
		var testInput EvaluationInput
		if err := json.Unmarshal(req.TestInput, &testInput); err == nil {
			testRule := &RuleDefinition{
				Expression: req.Expression,
				Parameters: req.Parameters,
			}
			if result, err := h.service.Evaluate(r.Context(), testRule, &testInput); err == nil {
				response["test_result"] = result
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
