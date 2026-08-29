package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/analytics"
	"github.com/hondyman/uisce/backend/internal/security"
)

// BOGraphHandler handles Business Object graph visualization requests
type BOGraphHandler struct {
	Service *analytics.BOGraphService
}

// NewBOGraphHandler creates a new graph handler
func NewBOGraphHandler(service *analytics.BOGraphService) *BOGraphHandler {
	return &BOGraphHandler{Service: service}
}

// RegisterRoutes registers the graph routes
func (h *BOGraphHandler) RegisterRoutes(r chi.Router) {
	r.Get("/api/bo/{boId}/graph", h.GetBOGraph)
}

// GetBOGraph generates and returns the lineage graph for a Business Object.
//
// SECURITY: previously had no authentication or tenant check at all — any
// caller could fetch any tenant's BO graph (node names, field/term names,
// physical column mappings) by ID with no session required. Every underlying
// query is now tenant-scoped via GenerateGraph's tenantID parameter, but that
// only holds if this handler always supplies the CALLER's own verified
// tenant, never one from the request.
func (h *BOGraphHandler) GetBOGraph(w http.ResponseWriter, r *http.Request) {
	auth, ok := security.RequireAuth(w, r)
	if !ok {
		return
	}

	boID := chi.URLParam(r, "boId")
	if boID == "" {
		http.Error(w, "boId is required", http.StatusBadRequest)
		return
	}

	var tenantID string
	switch {
	case len(auth.TenantIDs) > 0:
		tenantID = auth.TenantIDs[0]
	case auth.IsGlobalAdmin:
		// Global admins have no fixed tenant — the specific tenant to view
		// must be explicit, never inferred, so a mistaken/blank value can't
		// silently widen a query.
		requested := r.URL.Query().Get("tenantId")
		if _, err := uuid.Parse(requested); err != nil {
			http.Error(w, "tenantId query parameter is required for a global admin request", http.StatusBadRequest)
			return
		}
		tenantID = requested
	default:
		http.Error(w, "Forbidden: no tenant context", http.StatusForbidden)
		return
	}

	graph, err := h.Service.GenerateGraph(boID, tenantID)
	if err != nil {
		http.Error(w, "Failed to generate graph: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(graph)
}
