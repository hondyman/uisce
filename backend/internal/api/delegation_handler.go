package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/auth"
	"github.com/hondyman/semlayer/backend/internal/delegation"
)

type DelegationHandler struct {
	delegationService *delegation.DelegationService
}

func NewDelegationHandler(ds *delegation.DelegationService) *DelegationHandler {
	return &DelegationHandler{delegationService: ds}
}

// POST /api/delegations
func (h *DelegationHandler) CreateDelegation(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", 401)
		return
	}

	var req struct {
		ToUserID  string   `json:"to_user_id"`
		FromDate  string   `json:"from_date"` // YYYY-MM-DD
		ToDate    string   `json:"to_date"`   // YYYY-MM-DD
		Reason    string   `json:"reason"`
		Roles     []string `json:"roles"`
		Workflows []string `json:"workflows"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", 400)
		return
	}

	fromDate, _ := time.Parse("2006-01-02", req.FromDate)
	toDate, _ := time.Parse("2006-01-02", req.ToDate)

	delegationReq := delegation.CreateDelegationRequest{
		ToUserID:  req.ToUserID,
		FromDate:  fromDate,
		ToDate:    toDate,
		Reason:    req.Reason,
		Roles:     req.Roles,
		Workflows: req.Workflows,
	}

	delegationID, err := h.delegationService.CreateDelegation(
		r.Context(),
		user.TenantID,
		user.ID,
		req.ToUserID,
		delegationReq,
	)

	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"delegation_id": delegationID,
		"status":        "created",
	})
}

// GET /api/delegations/incoming
func (h *DelegationHandler) GetIncomingDelegations(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", 401)
		return
	}

	delegations, err := h.delegationService.GetActiveDelegationsForUser(r.Context(), user.TenantID, user.ID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"delegations": delegations,
	})
}

// GET /api/delegations/outgoing
func (h *DelegationHandler) GetOutgoingDelegations(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", 401)
		return
	}

	delegations, err := h.delegationService.GetOutgoingDelegationsForUser(r.Context(), user.TenantID, user.ID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"delegations": delegations,
	})
}

// POST /api/delegations/{delegationId}/revoke
func (h *DelegationHandler) RevokeDelegation(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", 401)
		return
	}
	delegationID := chi.URLParam(r, "delegationId")

	if err := h.delegationService.RevokeDelegation(r.Context(), delegationID, user.ID); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	w.WriteHeader(200)
	json.NewEncoder(w).Encode(map[string]string{"status": "revoked"})
}
