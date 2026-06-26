package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/cbo"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// SLOHandler handles CRUD operations for SLOs
type SLOHandler struct {
	provider *cbo.DBSLOProvider
}

// NewSLOHandler creates a new SLO handler
func NewSLOHandler(provider *cbo.DBSLOProvider) *SLOHandler {
	return &SLOHandler{provider: provider}
}

// RegisterRoutes registers the SLO routes
func (h *SLOHandler) RegisterRoutes(r chi.Router) {
	r.Get("/api/slos", h.ListSLOs)
	r.Post("/api/slos", h.CreateSLO)
	r.Get("/api/slos/{id}", h.GetSLO)
	r.Put("/api/slos/{id}", h.UpdateSLO)
	r.Delete("/api/slos/{id}", h.DeleteSLO)
}

// ListSLOs lists SLOs filtered by query parameters
func (h *SLOHandler) ListSLOs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get query params
	env := r.URL.Query().Get("env")
	if env == "" {
		env = "dev" // Default
	}

	scopeType := r.URL.Query().Get("scope_type")
	scopeID := r.URL.Query().Get("scope_id")

	// Get tenant from context (middleware extracted)
	// For now, assume tenant ID passed in header or query if not in context
	var tenantID *uuid.UUID
	tenantStr := jwtmiddleware.GetClaimsFromContext(r).TenantID
	if tenantStr != "" {
		if id, err := uuid.Parse(tenantStr); err == nil {
			tenantID = &id
		}
	}

	slos, err := h.provider.ListSLOs(ctx, env, tenantID, scopeType, scopeID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(slos)
}

// CreateSLO creates a new SLO
func (h *SLOHandler) CreateSLO(w http.ResponseWriter, r *http.Request) {
	var slo cbo.SLODefinition
	if err := json.NewDecoder(r.Body).Decode(&slo); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Defaults/Validation
	if slo.Env == "" {
		slo.Env = "dev"
	}
	if slo.TimeWindow == "" {
		slo.TimeWindow = "7d"
	}
	slo.Enabled = true

	// Assuming tenant ID from context overrides body if present, or validated

	if err := h.provider.CreateSLO(r.Context(), &slo); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(slo)
}

// GetSLO gets a single SLO by ID
func (h *SLOHandler) GetSLO(w http.ResponseWriter, r *http.Request) {
	// Not directly implemented in provider, but can use ListSLOs with ID filtering conceptually
	// But ListSLOs takes scopeType/ID, not UUID of SLO.
	// For now, return 501 Not Implemented or implement GetById in Provider
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

// UpdateSLO updates an SLO
func (h *SLOHandler) UpdateSLO(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var slo cbo.SLODefinition
	if err := json.NewDecoder(r.Body).Decode(&slo); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	slo.ID = id

	if err := h.provider.UpdateSLO(r.Context(), &slo); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(slo)
}

// DeleteSLO deletes an SLO
func (h *SLOHandler) DeleteSLO(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := h.provider.DeleteSLO(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
