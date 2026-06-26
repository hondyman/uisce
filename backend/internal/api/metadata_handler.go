package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/metadata"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

type MetadataHandler struct {
	ods    *metadata.ODSService
	layout *metadata.LayoutService
	hasura *metadata.HasuraClient
}

func NewMetadataHandler(ods *metadata.ODSService, layout *metadata.LayoutService, hasura *metadata.HasuraClient) *MetadataHandler {
	return &MetadataHandler{ods: ods, layout: layout, hasura: hasura}
}

func (h *MetadataHandler) RegisterRoutes(r chi.Router) {
	r.Route("/metadata", func(r chi.Router) {
		r.Post("/objects", h.CreateObjectDefinition)
		r.Get("/layouts/{slug}", h.GetLayout)
		r.Post("/layouts", h.CreateLayout)
	})
}

// CreateObjectDefinition handles the creation of new business objects
func (h *MetadataHandler) CreateObjectDefinition(w http.ResponseWriter, r *http.Request) {
	var input metadata.CreateObjectInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 1. Create Definition in ODS
	def, err := h.ods.CreateDefinition(r.Context(), input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 2. Automate Hasura Tracking (if table existed, but here we assume ODS creates table too - simplified)
	// In a real implementation, ODS would create the table DDL first.
	// For this MVP, we'll assume the table creation logic is inside ODS or handled separately.
	// We'll just trigger the track call as a demonstration.
	// h.hasura.TrackTable(input.Slug) 

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(def)
}

// GetLayout fetches a UI layout by slug
func (h *MetadataHandler) GetLayout(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	tenantIDStr := jwtmiddleware.GetClaimsFromContext(r).TenantID
	
	if tenantIDStr == "" {
		http.Error(w, "Missing X-Tenant-ID", http.StatusBadRequest)
		return
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		http.Error(w, "Invalid Tenant ID", http.StatusBadRequest)
		return
	}

	layout, err := h.layout.GetLayoutBySlug(r.Context(), tenantID, slug)
	if err != nil {
		http.Error(w, "Layout not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(layout)
}

// CreateLayout creates a new UI layout
func (h *MetadataHandler) CreateLayout(w http.ResponseWriter, r *http.Request) {
	type CreateLayoutInput struct {
		TenantID uuid.UUID              `json:"tenant_id"`
		Slug     string                 `json:"slug"`
		Title    string                 `json:"title"`
		Layout   map[string]interface{} `json:"layout"`
	}
	var input CreateLayoutInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	def, err := h.layout.CreateLayout(r.Context(), input.TenantID, input.Slug, input.Title, input.Layout)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(def)
}
