package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/analytics"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// CubeSchemaHandler serves dynamic Cube.js schema per tenant.
type CubeSchemaHandler struct {
	preAggSvc *analytics.PreAggregationService
}

func NewCubeSchemaHandler(preAggSvc *analytics.PreAggregationService) *CubeSchemaHandler {
	return &CubeSchemaHandler{preAggSvc: preAggSvc}
}

func (h *CubeSchemaHandler) RegisterRoutes(r chi.Router) {
	r.Get("/api/cube/schema", h.GetCubeSchema)
}

// GetCubeSchema returns dynamically generated Cube.js schema for a tenant.
// Cube.js can call this endpoint to load schema dynamically.
func (h *CubeSchemaHandler) GetCubeSchema(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		tenantID = jwtmiddleware.GetClaimsFromContext(r).TenantID
	}

	if tenantID == "" {
		http.Error(w, "tenant_id required", http.StatusBadRequest)
		return
	}

	schema, err := h.preAggSvc.GenerateCubeSchema(r.Context(), tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(schema)
}
