package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/services"
	"github.com/hondyman/semlayer/backend/models"
)

// GuardrailHandler handles API requests for the access guardrails feature.
type GuardrailHandler struct {
	service *services.GuardrailService
}

// NewGuardrailHandler creates a new GuardrailHandler.
func NewGuardrailHandler(service *services.GuardrailService) *GuardrailHandler {
	return &GuardrailHandler{service: service}
}

// RegisterRoutes registers the routes for GuardrailHandler.
func (h *GuardrailHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/guardrails", func(r chi.Router) {
		r.Post("/evaluate", h.HandleEvaluateClaimRequest)
		r.Get("/rules", h.HandleListRules)
		r.Post("/rules", h.HandleUpdateRule)
		r.Get("/violations", h.HandleListViolations)
	})
}

// HandleEvaluateClaimRequest evaluates a proposed claim against guardrail rules.
func (h *GuardrailHandler) HandleEvaluateClaimRequest(w http.ResponseWriter, r *http.Request) {
	var req models.EvaluateGuardrailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid request payload: " + err.Error()})
		return
	}

	response, err := h.service.EvaluateGuardrail(r.Context(), req)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to evaluate request", "details": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleListRules retrieves all active guardrail rules.
func (h *GuardrailHandler) HandleListRules(w http.ResponseWriter, r *http.Request) {
	rules, err := h.service.ListRules(r.Context())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to list guardrail rules"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rules)
}

// HandleUpdateRule creates or updates a guardrail rule.
func (h *GuardrailHandler) HandleUpdateRule(w http.ResponseWriter, r *http.Request) {
	var rule models.GuardrailRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid rule payload: " + err.Error()})
		return
	}

	updatedRule, err := h.service.UpdateRule(r.Context(), rule)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to update rule"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedRule)
}

// HandleListViolations retrieves recent guardrail violations.
func (h *GuardrailHandler) HandleListViolations(w http.ResponseWriter, r *http.Request) {
	violations, err := h.service.ListViolations(r.Context())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to list violations"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(violations)
}
