package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/access"
	"github.com/hondyman/semlayer/backend/internal/auth"
)

type AccessHandler struct {
	accessService *access.AccessService
}

func NewAccessHandler(s *access.AccessService) *AccessHandler {
	return &AccessHandler{accessService: s}
}

// GET /api/workflows/initiatable
func (h *AccessHandler) ListInitiatableWorkflows(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	bpDefIDs, err := h.accessService.ListInitiatableWorkflows(
		r.Context(),
		user.ID,
		user.TenantID,
	)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"workflows": bpDefIDs,
		"count":     len(bpDefIDs),
	})
}

// POST /api/workflows/{bpDefId}/can-initiate
func (h *AccessHandler) CanInitiate(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	bpDefID := chi.URLParam(r, "bpDefId")

	var req struct {
		Role string `json:"role"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	can, reason, err := h.accessService.CanInitiateWorkflow(
		r.Context(),
		user.ID,
		req.Role,
		user.TenantID,
		bpDefID,
	)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"allowed": can,
		"reason":  reason,
	})
}
