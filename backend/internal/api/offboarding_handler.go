package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/auth"
	"github.com/hondyman/semlayer/backend/internal/offboarding"
)

type OffboardingHandler struct {
	offboardingService *offboarding.OffboardingService
}

func NewOffboardingHandler(os *offboarding.OffboardingService) *OffboardingHandler {
	return &OffboardingHandler{offboardingService: os}
}

// POST /api/admin/offboard
func (h *OffboardingHandler) OffboardUser(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromContext(r.Context())
	// For MVP, assume everyone can call this or rely on basic auth checks somewhere.
	// Ideally check if user.IsAdmin
	if !ok {
		http.Error(w, "Unauthorized", 401)
		return
	}

	var req struct {
		UserID           string `json:"user_id"`
		ReassignToUserID string `json:"reassign_to_user_id"`
		Reason           string `json:"reason"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	id, err := h.offboardingService.InitiateOffboarding(
		r.Context(), user.TenantID, req.UserID, req.ReassignToUserID, user.ID, req.Reason,
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
	user, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", 401)
		return
	}

	limit, offset := 50, 0
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		fmt.Sscanf(lStr, "%d", &limit)
	}
	if oStr := r.URL.Query().Get("offset"); oStr != "" {
		fmt.Sscanf(oStr, "%d", &offset)
	}

	obs, total, err := h.offboardingService.ListAllOffboardings(r.Context(), user.TenantID, limit, offset)
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
	user, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", 401)
		return
	}

	id := chi.URLParam(r, "offboardingId")
	if err := h.offboardingService.ReverseOffboarding(r.Context(), id, user.ID); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	w.WriteHeader(200)
}
