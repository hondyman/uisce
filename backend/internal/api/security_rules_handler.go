package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/backend/internal/security"
	"go.uber.org/zap"
)

// SecurityRulesHandler manages access rule HTTP endpoints.
type SecurityRulesHandler struct {
	service *security.AccessRuleService
}

// NewSecurityRulesHandler creates a handler for security rule endpoints.
func NewSecurityRulesHandler(service *security.AccessRuleService) *SecurityRulesHandler {
	return &SecurityRulesHandler{service: service}
}

// RegisterRoutes wires all security rule endpoints to the router.
func (h *SecurityRulesHandler) RegisterRoutes(r chi.Router) {
	r.Get("/security/rules", h.ListRules)
	r.Post("/security/rules", h.CreateRule)
	r.Get("/security/rules/{ruleId}", h.GetRule)
	r.Put("/security/rules/{ruleId}", h.UpdateRule)
	r.Post("/security/rules/validate", h.ValidateRuleDsl)
	r.Get("/security/rules/{ruleId}/impact", h.GetRuleImpact)
}

// ListRules handles [GET] /security/rules
func (h *SecurityRulesHandler) ListRules(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLogger()
	ctx := r.Context()
	filters := security.AccessRuleFilters{
		TenantID:         r.URL.Query().Get("tenantId"),
		BusinessObjectID: r.URL.Query().Get("businessObjectId"),
		Status:           r.URL.Query().Get("status"),
	}

	rules, err := h.service.List(ctx, filters)
	if err != nil {
		logger.Error("Failed to list access rules", zap.Error(err))
		http.Error(w, "Failed to list rules", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rules)
}

// GetRule handles [GET] /security/rules/{ruleId}
func (h *SecurityRulesHandler) GetRule(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLogger()
	ctx := r.Context()
	ruleID := chi.URLParam(r, "ruleId")

	rule, err := h.service.Get(ctx, ruleID)
	if err != nil {
		logger.Error("Failed to get access rule", zap.String("ruleId", ruleID), zap.Error(err))
		http.Error(w, "Rule not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rule)
}

// CreateRule handles [POST] /security/rules
func (h *SecurityRulesHandler) CreateRule(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLogger()
	ctx := r.Context()
	var req models.AccessRule
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	created, err := h.service.Create(ctx, &req)
	if err != nil {
		logger.Error("Failed to create access rule", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}

// UpdateRule handles [PUT] /security/rules/{ruleId}
func (h *SecurityRulesHandler) UpdateRule(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLogger()
	ctx := r.Context()
	ruleID := chi.URLParam(r, "ruleId")

	var req models.AccessRule
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	updated, err := h.service.Update(ctx, ruleID, &req)
	if err != nil {
		logger.Error("Failed to update access rule", zap.String("ruleId", ruleID), zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updated)
}

// ValidateRuleDslRequest represents DSL validation input.
type ValidateRuleDslRequest struct {
	BusinessObjectID string `json:"businessObjectId"`
	RowFilterDsl     string `json:"rowFilterDsl"`
}

// ValidateRuleDsl handles [POST] /security/rules/validate
func (h *SecurityRulesHandler) ValidateRuleDsl(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLogger()
	ctx := r.Context()
	var req ValidateRuleDslRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	result, err := h.service.ValidateDsl(ctx, req.BusinessObjectID, req.RowFilterDsl)
	if err != nil {
		logger.Error("Failed to validate DSL", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GetRuleImpact handles [GET] /security/rules/{ruleId}/impact
func (h *SecurityRulesHandler) GetRuleImpact(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLogger()
	ctx := r.Context()
	ruleID := chi.URLParam(r, "ruleId")

	impact, err := h.service.GetImpact(ctx, ruleID)
	if err != nil {
		logger.Error("Failed to get rule impact", zap.String("ruleId", ruleID), zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(impact)
}
