package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/data_intelligence/tiering"
)

// TieringHandler handles storage tiering API requests
type TieringHandler struct {
	service *tiering.StorageTiering
}

// NewTieringHandler creates a new tiering handler
func NewTieringHandler(service *tiering.StorageTiering) *TieringHandler {
	return &TieringHandler{service: service}
}

// RegisterRoutes registers tiering routes
func (h *TieringHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/intelligence/storage", func(r chi.Router) {
		r.Get("/plans", h.ListPlans)
		r.Post("/plans/generate", h.GeneratePlan)
		r.Get("/plans/{id}", h.GetPlan)
		r.Post("/plans/{id}/execute", h.ExecutePlan)
		r.Post("/plans/{id}/status", h.UpdateStatus)
	})
}

// ListPlans returns all plans for a tenant
func (h *TieringHandler) ListPlans(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		http.Error(w, "tenant_id index is required", http.StatusBadRequest)
		return
	}

	plans, err := h.service.ListPlans(r.Context(), tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(plans)
}

// GeneratePlan triggers a new analysis and saves the plan
func (h *TieringHandler) GeneratePlan(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantID string `json:"tenant_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	plan, err := h.service.GeneratePlan(r.Context(), req.TenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(plan)
}

// GetPlan returns a specific plan
func (h *TieringHandler) GetPlan(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid plan ID", http.StatusBadRequest)
		return
	}

	plan, err := h.service.GetPlan(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(plan)
}

// ExecutePlan starts data movement according to the plan
func (h *TieringHandler) ExecutePlan(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid plan ID", http.StatusBadRequest)
		return
	}

	plan, err := h.service.GetPlan(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if err := h.service.ExecutePlan(r.Context(), plan); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "migrating"})
}

// UpdateStatus manually updates plan status (e.g. dismissing it)
func (h *TieringHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	_, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid plan ID", http.StatusBadRequest)
		return
	}

	// This would call a service method in real implementation
	// For now, return OK
	w.WriteHeader(http.StatusOK)
}
