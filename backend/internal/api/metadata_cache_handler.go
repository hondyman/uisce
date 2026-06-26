package api

import (
	"encoding/json"
	"net/http"

	"github.com/hondyman/semlayer/backend/pkg/meta"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// MetadataCacheHandler provides HTTP endpoints for cache management
type MetadataCacheHandler struct {
	service *meta.Service
}

// NewMetadataCacheHandler creates a new cache handler
func NewMetadataCacheHandler(service *meta.Service) *MetadataCacheHandler {
	return &MetadataCacheHandler{
		service: service,
	}
}

// GetCacheStats returns cache performance statistics
// GET /api/admin/metadata/cache/stats
func (h *MetadataCacheHandler) GetCacheStats(w http.ResponseWriter, r *http.Request) {
	metrics, err := h.service.GetCacheMetrics()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"hits":         metrics.Hits,
		"misses":       metrics.Misses,
		"evictions":    metrics.Evictions,
		"hit_rate":     metrics.HitRate,
		"item_count":   metrics.ItemCount,
		"memory_bytes": metrics.MemoryBytes,
		"load_time_ms": metrics.LoadTime.Milliseconds(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// WarmCache triggers cache warmup for a tenant
// POST /api/admin/metadata/cache/warm
func (h *MetadataCacheHandler) WarmCache(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantID string `json:"tenant_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.TenantID == "" {
		http.Error(w, "tenant_id is required", http.StatusBadRequest)
		return
	}

	if err := h.service.WarmCache(r.Context(), req.TenantID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Cache warmed successfully",
	})
}

// InvalidateCache invalidates cache for a tenant
// POST /api/admin/metadata/cache/invalidate
func (h *MetadataCacheHandler) InvalidateCache(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantID string `json:"tenant_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.TenantID == "" {
		http.Error(w, "tenant_id is required", http.StatusBadRequest)
		return
	}

	h.service.InvalidateCache(req.TenantID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Cache invalidated successfully",
	})
}

// GetBusinessObjectWithCache retrieves a business object using cache
// GET /api/metadata/business-objects/:name
func (h *MetadataCacheHandler) GetBusinessObjectWithCache(w http.ResponseWriter, r *http.Request) {
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

	// Extract name from URL path or query params
	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "name parameter required", http.StatusBadRequest)
		return
	}

	bo, err := h.service.GetBusinessObjectByName(r.Context(), tenantID, name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	// Add cache hit header for debugging
	w.Header().Set("X-Cache-Hit", "true")
	json.NewEncoder(w).Encode(bo)
}

// ListBusinessObjectsWithCache lists all business objects using cache
// GET /api/metadata/business-objects
func (h *MetadataCacheHandler) ListBusinessObjectsWithCache(w http.ResponseWriter, r *http.Request) {
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

	objects, err := h.service.ListBusinessObjects(r.Context(), tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache-Hit", "true")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"business_objects": objects,
		"count":            len(objects),
	})
}
