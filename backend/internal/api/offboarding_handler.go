package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/uisce/backend/internal/offboarding"
	"github.com/hondyman/uisce/backend/internal/security"
)

// requireOffboardingAdmin authenticates the caller, requires an admin role
// (offboarding a user — including reassigning their work and deactivating
// access — is a highly sensitive admin action, never a self-service one),
// and returns the caller's tenant ID (empty for global admins acting
// tenant-agnostically).
func requireOffboardingAdmin(w http.ResponseWriter, r *http.Request) (auth security.AuthInfo, tenantID string, ok bool) {
	auth, ok = security.RequireAuth(w, r)
	if !ok {
		return security.AuthInfo{}, "", false
	}
	if !isTenantOrGlobalAdmin(auth.Roles) {
		http.Error(w, "Forbidden: admin role required", http.StatusForbidden)
		return security.AuthInfo{}, "", false
	}
	if len(auth.TenantIDs) > 0 {
		tenantID = auth.TenantIDs[0]
	}
	return auth, tenantID, true
}

type OffboardingHandler struct {
	offboardingService *offboarding.OffboardingService
}

func NewOffboardingHandler(os *offboarding.OffboardingService) *OffboardingHandler {
	return &OffboardingHandler{offboardingService: os}
}

// POST /api/admin/offboard
func (h *OffboardingHandler) OffboardUser(w http.ResponseWriter, r *http.Request) {
	auth, tenantID, ok := requireOffboardingAdmin(w, r)
	if !ok {
		return
	}

	var req struct {
		UserID           string `json:"user_id"`
		ReassignToUserID string `json:"reassign_to_user_id"`
		Reason           string `json:"reason"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	id, err := h.offboardingService.InitiateOffboarding(
		r.Context(), tenantID, req.UserID, req.ReassignToUserID, auth.UserID, req.Reason,
	)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"offboarding_id": id})
}

// GET /api/admin/offboarding
func (h *OffboardingHandler) ListOffboardings(w http.ResponseWriter, r *http.Request) {
	_, tenantID, ok := requireOffboardingAdmin(w, r)
	if !ok {
		return
	}

	limit, offset := 50, 0
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		fmt.Sscanf(lStr, "%d", &limit)
	}
	if oStr := r.URL.Query().Get("offset"); oStr != "" {
		fmt.Sscanf(oStr, "%d", &offset)
	}

	obs, total, err := h.offboardingService.ListAllOffboardings(r.Context(), tenantID, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"offboardings": obs,
		"total":        total,
	})
}

func (h *OffboardingHandler) ReverseOffboarding(w http.ResponseWriter, r *http.Request) {
	auth, _, ok := requireOffboardingAdmin(w, r)
	if !ok {
		return
	}

	id := chi.URLParam(r, "offboardingId")
	if err := h.offboardingService.ReverseOffboarding(r.Context(), id, auth.UserID); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	w.WriteHeader(200)
}
