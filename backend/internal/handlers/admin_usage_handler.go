package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/security"
	"github.com/hondyman/semlayer/backend/internal/store"
)

// AdminUsageHandler provides endpoints for querying API key usage data
type AdminUsageHandler struct {
	usageStore store.APIKeyUsageStore
}

// NewAdminUsageHandler creates a new admin usage handler
func NewAdminUsageHandler(usageStore store.APIKeyUsageStore) *AdminUsageHandler {
	return &AdminUsageHandler{usageStore: usageStore}
}

// RegisterRoutes registers all usage-related routes
func (h *AdminUsageHandler) RegisterRoutes(r chi.Router) {
	r.Route("/admin", func(r chi.Router) {
		r.Get("/api-keys/{apiKeyID}/usage", h.GetAPIKeyUsage)
		r.Get("/tenants/{tenantID}/usage/daily", h.GetTenantDailyUsage)
		r.Get("/tenants/{tenantID}/usage/endpoints", h.GetTenantEndpointUsage)
		r.Get("/tenants/{tenantID}/usage/recent", h.GetTenantRecentUsage)
	})
}

// GetAPIKeyUsage retrieves usage records for a specific API key
// GET /api/admin/api-keys/{apiKeyID}/usage?limit=100
func (h *AdminUsageHandler) GetAPIKeyUsage(w http.ResponseWriter, r *http.Request) {
	actor, ok := security.AuthInfoFromContext(r.Context())
	if !ok || strings.TrimSpace(actor.UserID) == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if !hasAdminRole(actor.Roles) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	apiKeyIDStr := chi.URLParam(r, "apiKeyID")
	apiKeyID, err := uuid.Parse(apiKeyIDStr)
	if err != nil {
		http.Error(w, "invalid api_key_id", http.StatusBadRequest)
		return
	}

	limit := 100
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	usage, err := h.usageStore.GetAPIKeyUsage(r.Context(), apiKeyID, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"usage": usage,
	})
}

// GetTenantDailyUsage retrieves daily usage statistics for a tenant
// GET /api/admin/tenants/{tenantID}/usage/daily?days=30
func (h *AdminUsageHandler) GetTenantDailyUsage(w http.ResponseWriter, r *http.Request) {
	actor, ok := security.AuthInfoFromContext(r.Context())
	if !ok || strings.TrimSpace(actor.UserID) == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if !hasAdminRole(actor.Roles) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	tenantIDStr := chi.URLParam(r, "tenantID")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		http.Error(w, "invalid tenant_id", http.StatusBadRequest)
		return
	}

	days := 30
	if daysStr := r.URL.Query().Get("days"); daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 && d <= 365 {
			days = d
		}
	}

	stats, err := h.usageStore.GetDailyUsageByTenant(r.Context(), tenantID, days)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tenant_id": tenantID,
		"days":      days,
		"data":      stats,
	})
}

// GetTenantEndpointUsage retrieves endpoint usage statistics for a tenant
// GET /api/admin/tenants/{tenantID}/usage/endpoints?limit=20
func (h *AdminUsageHandler) GetTenantEndpointUsage(w http.ResponseWriter, r *http.Request) {
	actor, ok := security.AuthInfoFromContext(r.Context())
	if !ok || strings.TrimSpace(actor.UserID) == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if !hasAdminRole(actor.Roles) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	tenantIDStr := chi.URLParam(r, "tenantID")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		http.Error(w, "invalid tenant_id", http.StatusBadRequest)
		return
	}

	limit := 20
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	stats, err := h.usageStore.GetEndpointUsageByTenant(r.Context(), tenantID, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tenant_id":     tenantID,
		"top_endpoints": stats,
	})
}

// GetTenantRecentUsage retrieves recent usage records for a tenant
// GET /api/admin/tenants/{tenantID}/usage/recent?limit=100
func (h *AdminUsageHandler) GetTenantRecentUsage(w http.ResponseWriter, r *http.Request) {
	actor, ok := security.AuthInfoFromContext(r.Context())
	if !ok || strings.TrimSpace(actor.UserID) == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if !hasAdminRole(actor.Roles) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	tenantIDStr := chi.URLParam(r, "tenantID")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		http.Error(w, "invalid tenant_id", http.StatusBadRequest)
		return
	}

	limit := 100
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	usage, err := h.usageStore.GetRecentUsageByTenant(r.Context(), tenantID, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tenant_id": tenantID,
		"requests":  usage,
	})
}
