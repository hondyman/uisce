package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/hondyman/semlayer/backend/services/compliance-engine/internal/models"
	"github.com/hondyman/semlayer/backend/services/compliance-engine/internal/service"
)

// TradeHandler handles trade-related API endpoints
type TradeHandler struct {
	service *service.ComplianceService
}

// NewTradeHandler creates a new trade handler
func NewTradeHandler(service *service.ComplianceService) *TradeHandler {
	return &TradeHandler{service: service}
}

// RegisterRoutes registers all trade-related routes
func (h *TradeHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/compliance", func(r chi.Router) {
		r.Post("/validate", h.Validate)
		r.Post("/submit", h.Submit)
	})
}

// Validate performs pre-trade validation only (synchronous)
// POST /api/v1/compliance/validate
func (h *TradeHandler) Validate(w http.ResponseWriter, r *http.Request) {
	var trade models.TradeRequest
	if err := json.NewDecoder(r.Body).Decode(&trade); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// attach tenant from JWT claims to context (if present)
	if claims := jwtmiddleware.GetClaimsFromContext(r); claims != nil {
		ctx := context.WithValue(r.Context(), jwtmiddleware.TenantIDContextKey, claims.TenantID)
		r = r.WithContext(ctx)
	}

	// Optional version parameter
	version := r.URL.Query().Get("version")
	if version != "" {
		// TODO: Allow manual version override for testing
	}

	result, err := h.service.PreTradeValidate(r.Context(), trade)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if result.Status == "REJECTED" {
		w.WriteHeader(http.StatusBadRequest)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// Submit performs pre-trade validation and queues post-trade processing
// POST /api/v1/compliance/submit
func (h *TradeHandler) Submit(w http.ResponseWriter, r *http.Request) {
	var trade models.TradeRequest
	if err := json.NewDecoder(r.Body).Decode(&trade); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// carry tenant information into context for service/audit
	if claims := jwtmiddleware.GetClaimsFromContext(r); claims != nil {
		ctx := context.WithValue(r.Context(), jwtmiddleware.TenantIDContextKey, claims.TenantID)
		r = r.WithContext(ctx)
	}

	result, err := h.service.SubmitTrade(r.Context(), trade)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if result.Status == "REJECTED" {
		w.WriteHeader(http.StatusBadRequest)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
