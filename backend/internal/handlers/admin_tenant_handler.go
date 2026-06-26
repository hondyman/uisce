package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/backend/internal/security"
	"github.com/hondyman/semlayer/backend/internal/store"
)

// AdminTenantHandler provides endpoints for tenant management
type AdminTenantHandler struct {
	tenantStore store.TenantStore
}

// NewAdminTenantHandler creates a new admin tenant handler
func NewAdminTenantHandler(tenantStore store.TenantStore) *AdminTenantHandler {
	return &AdminTenantHandler{tenantStore: tenantStore}
}

// RegisterRoutes registers all tenant-related routes
func (h *AdminTenantHandler) RegisterRoutes(r chi.Router) {
	r.Route("/admin/tenants", func(r chi.Router) {
		r.Get("/", h.ListTenants)
		r.Post("/", h.CreateTenant)
		r.Route("/{tenantID}", func(r chi.Router) {
			r.Get("/", h.GetTenant)
			r.Patch("/", h.UpdateTenant)
			r.Delete("/", h.DeleteTenant)
			r.Post("/suspend", h.SuspendTenant)
			r.Post("/unsuspend", h.UnsuspendTenant)
		})
	})
}

// ListTenants retrieves all tenants with pagination
// GET /api/admin/tenants?limit=50&offset=0
func (h *AdminTenantHandler) ListTenants(w http.ResponseWriter, r *http.Request) {
	if err := h.requireAdminAuth(r); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	offset := 0
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	tenants, total, err := h.tenantStore.ListTenants(r.Context(), limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tenants": tenants,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

// CreateTenant creates a new tenant
// POST /api/admin/tenants
func (h *AdminTenantHandler) CreateTenant(w http.ResponseWriter, r *http.Request) {
	if err := h.requireAdminAuth(r); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var req struct {
		Name          string  `json:"name"`
		Code          *string `json:"code"`
		Region        *string `json:"region"`
		Plan          string  `json:"plan"`
		MaxRequests   *int64  `json:"max_requests"`
		WindowSeconds *int    `json:"window_seconds"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if strings.TrimSpace(req.Name) == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	if req.Plan == "" {
		req.Plan = "free"
	}

	if !models.ValidateTenantPlan(req.Plan) {
		http.Error(w, "invalid plan: must be free, pro, or enterprise", http.StatusBadRequest)
		return
	}

	tenantReq := models.TenantCreateRequest{
		ID:            uuid.New(),
		Name:          strings.TrimSpace(req.Name),
		Code:          req.Code,
		Region:        req.Region,
		Plan:          req.Plan,
		MaxRequests:   req.MaxRequests,
		WindowSeconds: req.WindowSeconds,
	}

	tenant, err := h.tenantStore.CreateTenant(r.Context(), tenantReq)
	if err != nil {
		if strings.Contains(err.Error(), "unique constraint") {
			http.Error(w, "tenant code already exists", http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tenant": tenant,
	})
}

// GetTenant retrieves a single tenant by ID
// GET /api/admin/tenants/{tenantID}
func (h *AdminTenantHandler) GetTenant(w http.ResponseWriter, r *http.Request) {
	if err := h.requireAdminAuth(r); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	tenantID, err := h.parseUUID(chi.URLParam(r, "tenantID"))
	if err != nil {
		http.Error(w, "invalid tenant_id", http.StatusBadRequest)
		return
	}

	tenant, err := h.tenantStore.GetTenantByID(r.Context(), tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if tenant == nil {
		http.Error(w, "tenant not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tenant": tenant,
	})
}

// UpdateTenant updates a tenant's metadata
// PATCH /api/admin/tenants/{tenantID}
func (h *AdminTenantHandler) UpdateTenant(w http.ResponseWriter, r *http.Request) {
	if err := h.requireAdminAuth(r); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	tenantID, err := h.parseUUID(chi.URLParam(r, "tenantID"))
	if err != nil {
		http.Error(w, "invalid tenant_id", http.StatusBadRequest)
		return
	}

	var req struct {
		Name          *string `json:"name"`
		Region        *string `json:"region"`
		Plan          *string `json:"plan"`
		MaxRequests   *int64  `json:"max_requests"`
		WindowSeconds *int    `json:"window_seconds"`
		IsSuspended   *bool   `json:"is_suspended"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate plan if provided
	if req.Plan != nil && !models.ValidateTenantPlan(*req.Plan) {
		http.Error(w, "invalid plan: must be free, pro, or enterprise", http.StatusBadRequest)
		return
	}

	updateReq := models.TenantUpdateRequest{
		Name:          req.Name,
		Region:        req.Region,
		Plan:          req.Plan,
		MaxRequests:   req.MaxRequests,
		WindowSeconds: req.WindowSeconds,
		IsSuspended:   req.IsSuspended,
	}

	tenant, err := h.tenantStore.UpdateTenant(r.Context(), tenantID, updateReq)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "tenant not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tenant": tenant,
	})
}

// DeleteTenant deletes a tenant
// DELETE /api/admin/tenants/{tenantID}
func (h *AdminTenantHandler) DeleteTenant(w http.ResponseWriter, r *http.Request) {
	if err := h.requireAdminAuth(r); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	tenantID, err := h.parseUUID(chi.URLParam(r, "tenantID"))
	if err != nil {
		http.Error(w, "invalid tenant_id", http.StatusBadRequest)
		return
	}

	if err := h.tenantStore.DeleteTenant(r.Context(), tenantID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "tenant not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// SuspendTenant suspends a tenant (hard kill switch)
// POST /api/admin/tenants/{tenantID}/suspend
func (h *AdminTenantHandler) SuspendTenant(w http.ResponseWriter, r *http.Request) {
	if err := h.requireAdminAuth(r); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	tenantID, err := h.parseUUID(chi.URLParam(r, "tenantID"))
	if err != nil {
		http.Error(w, "invalid tenant_id", http.StatusBadRequest)
		return
	}

	if err := h.tenantStore.SuspendTenant(r.Context(), tenantID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "tenant not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// UnsuspendTenant unsuspends a tenant
// POST /api/admin/tenants/{tenantID}/unsuspend
func (h *AdminTenantHandler) UnsuspendTenant(w http.ResponseWriter, r *http.Request) {
	if err := h.requireAdminAuth(r); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	tenantID, err := h.parseUUID(chi.URLParam(r, "tenantID"))
	if err != nil {
		http.Error(w, "invalid tenant_id", http.StatusBadRequest)
		return
	}

	if err := h.tenantStore.UnsuspendTenant(r.Context(), tenantID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "tenant not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ============================================================================
// Helper Methods
// ============================================================================

func (h *AdminTenantHandler) requireAdminAuth(r *http.Request) error {
	actor, ok := security.AuthInfoFromContext(r.Context())
	if !ok || strings.TrimSpace(actor.UserID) == "" {
		return errors.New("unauthorized")
	}

	if !hasAdminRole(actor.Roles) {
		return errors.New("forbidden")
	}

	return nil
}

func (h *AdminTenantHandler) parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}
