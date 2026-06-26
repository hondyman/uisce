package api

import (
	"encoding/json"
	"net/http"

	"github.com/hondyman/semlayer/backend/pkg/meta"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// UnifiedMetadataHandler provides HTTP endpoints for unified metadata access
type UnifiedMetadataHandler struct {
	service *meta.UnifiedMetadataService
}

// NewUnifiedMetadataHandler creates a new unified metadata handler
func NewUnifiedMetadataHandler(service *meta.UnifiedMetadataService) *UnifiedMetadataHandler {
	return &UnifiedMetadataHandler{
		service: service,
	}
}

// GetAllMetadata returns all metadata (business objects + semantic views) for a tenant
// GET /api/metadata/unified
func (h *UnifiedMetadataHandler) GetAllMetadata(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "X-Tenant-ID header required", http.StatusBadRequest)
		return
	}

	metadata, err := h.service.GetAllMetadata(r.Context(), tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache-Source", "unified")
	json.NewEncoder(w).Encode(metadata)
}

// GetBusinessObject retrieves a business object from in-memory cache
// GET /api/metadata/unified/business-objects/:key
func (h *UnifiedMetadataHandler) GetBusinessObject(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "X-Tenant-ID header required", http.StatusBadRequest)
		return
	}

	boKey := r.URL.Query().Get("key")
	if boKey == "" {
		http.Error(w, "key parameter required", http.StatusBadRequest)
		return
	}

	bo, err := h.service.GetBusinessObject(r.Context(), tenantID, boKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache-Source", "in-memory")
	json.NewEncoder(w).Encode(bo)
}

// GetSemanticView retrieves a semantic view from Redis cache
// GET /api/metadata/unified/semantic-views/:id
func (h *UnifiedMetadataHandler) GetSemanticView(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "X-Tenant-ID header required", http.StatusBadRequest)
		return
	}

	viewID := r.URL.Query().Get("id")
	if viewID == "" {
		http.Error(w, "id parameter required", http.StatusBadRequest)
		return
	}

	view, err := h.service.GetSemanticView(r.Context(), tenantID, viewID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache-Source", "redis")
	json.NewEncoder(w).Encode(view)
}

// CreateBOToViewMapping creates a mapping between a business object and semantic view
// POST /api/metadata/unified/mappings
func (h *UnifiedMetadataHandler) CreateBOToViewMapping(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "X-Tenant-ID header required", http.StatusBadRequest)
		return
	}

	var req struct {
		BOKey  string `json:"bo_key"`
		ViewID string `json:"view_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	mapping, err := h.service.MapBusinessObjectToSemanticView(r.Context(), tenantID, req.BOKey, req.ViewID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(mapping)
}

// GetBOToViewMappings retrieves all mappings for a business object
// GET /api/metadata/unified/mappings
func (h *UnifiedMetadataHandler) GetBOToViewMappings(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "X-Tenant-ID header required", http.StatusBadRequest)
		return
	}

	boKey := r.URL.Query().Get("bo_key")
	if boKey == "" {
		http.Error(w, "bo_key parameter required", http.StatusBadRequest)
		return
	}

	mappings, err := h.service.GetBOToViewMappings(r.Context(), tenantID, boKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"mappings": mappings,
		"count":    len(mappings),
	})
}

// GetCombinedMetrics returns metrics from both caches
// GET /api/metadata/unified/metrics
func (h *UnifiedMetadataHandler) GetCombinedMetrics(w http.ResponseWriter, r *http.Request) {
	metrics, err := h.service.GetCombinedMetrics()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// InvalidateAllCaches invalidates both business object and semantic view caches
// POST /api/metadata/unified/invalidate
func (h *UnifiedMetadataHandler) InvalidateAllCaches(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "X-Tenant-ID header required", http.StatusBadRequest)
		return
	}

	if err := h.service.InvalidateAll(r.Context(), tenantID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "All metadata caches invalidated",
	})
}
