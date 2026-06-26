package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/mdm"
)

type SecurityLineageHandler struct {
	svc *mdm.SecurityLineageService
}

func NewSecurityLineageHandler(svc *mdm.SecurityLineageService) *SecurityLineageHandler {
	return &SecurityLineageHandler{svc: svc}
}

func (h *SecurityLineageHandler) RegisterRoutes(r chi.Router) {
	r.Route("/v1/security", func(r chi.Router) {
		r.Get("/{securityId}/lineage", h.GetLineage)
		r.Post("/{securityId}/impact", h.SimulateImpact)
	})
}

func (h *SecurityLineageHandler) GetLineage(w http.ResponseWriter, r *http.Request) {
	securityID := chi.URLParam(r, "securityId")
	tenantID := mustTenantID(r)

	lineage, err := h.svc.GetSecurityLineage(r.Context(), tenantID, securityID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, lineage)
}

func (h *SecurityLineageHandler) SimulateImpact(w http.ResponseWriter, r *http.Request) {
	securityID := chi.URLParam(r, "securityId")
	tenantID := mustTenantID(r)

	var changes map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&changes); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	impact, err := h.svc.GetSecurityImpactAnalysis(r.Context(), tenantID, securityID, changes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, impact)
}
