package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/bp"
)

type BPHandler struct {
	Service *bp.DesignerService
	Events  *ApprovalEventService
}

func NewBPHandler(svc *bp.DesignerService, events *ApprovalEventService) *BPHandler {
	return &BPHandler{Service: svc, Events: events}
}

func (h *BPHandler) RegisterRoutes(r chi.Router) {
	r.Route("/bp-designer", func(r chi.Router) {
		r.Post("/save", h.SaveDesigner)
		r.Get("/{bpDefId}", h.GetDesigner)
	})
	r.Get("/workflows/{workflowId}/events", h.Events.GetApprovalEvents)
}

func (h *BPHandler) SaveDesigner(w http.ResponseWriter, r *http.Request) {
	var in bp.SaveDesignerInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Mock tenant
	in.TenantID = "default_tenant"

	id, err := h.Service.SaveDesigner(r.Context(), in)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"bpDefId": id})
}

func (h *BPHandler) GetDesigner(w http.ResponseWriter, r *http.Request) {
	bpDefID := chi.URLParam(r, "bpDefId")
	if bpDefID == "" {
		http.Error(w, "missing bpDefId", http.StatusBadRequest)
		return
	}

	res, err := h.Service.GetDesigner(r.Context(), bpDefID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}
