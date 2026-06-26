package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/audit"
	"github.com/hondyman/semlayer/backend/internal/services"
	"github.com/hondyman/semlayer/backend/internal/succession"
	"github.com/hondyman/semlayer/backend/internal/webhooks"
)

type SuccessionHandlers struct {
	service    succession.Service
	auditSvc   *audit.Service
	secMgr     *services.SecurityManager
	webhookSvc *webhooks.Service
}

func NewSuccessionHandlers(service succession.Service, auditSvc *audit.Service, secMgr *services.SecurityManager, webhookSvc *webhooks.Service) *SuccessionHandlers {
	return &SuccessionHandlers{service: service, auditSvc: auditSvc, secMgr: secMgr, webhookSvc: webhookSvc}
}

const (
	permSuccessionRead  = "succession.read"
	permSuccessionWrite = "succession.write"
)

func (h *SuccessionHandlers) RegisterRoutes(r chi.Router) {
	r.Route("/api/succession", func(r chi.Router) {
		// Practice Metrics
		r.Get("/metrics", h.ListPracticeMetrics)
		r.Get("/metrics/{advisorId}", h.GetPracticeMetrics)
		r.Post("/metrics/{advisorId}/calculate", h.CalculatePracticeValue)

		// Succession Plans
		r.Post("/plans", h.CreateSuccessionPlan)
		r.Get("/plans", h.ListSuccessionPlans)
		r.Get("/plans/{planId}", h.GetSuccessionPlan)
		r.Put("/plans/{planId}", h.UpdateSuccessionPlan)

		// Client Transitions
		r.Post("/transitions", h.CreateClientTransition)
		r.Get("/transitions", h.ListTransitions)
		r.Put("/transitions/{transitionId}/status", h.UpdateTransitionStatus)
	})
}

func (h *SuccessionHandlers) ListPracticeMetrics(w http.ResponseWriter, r *http.Request) {
	if _, _, _, ok := h.authorize(w, r, permSuccessionRead); !ok {
		return
	}
	writeJSONError(w, http.StatusNotImplemented, "Listing practice metrics is not implemented", "not_implemented", nil)
}

func (h *SuccessionHandlers) GetPracticeMetrics(w http.ResponseWriter, r *http.Request) {
	actorID, tenantID, _, ok := h.authorize(w, r, permSuccessionRead)
	if !ok {
		return
	}

	advisorID, err := uuid.Parse(chi.URLParam(r, "advisorId"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid advisor ID", "invalid_advisor_id", nil)
		return
	}

	metrics, err := h.service.CalculatePracticeMetrics(r.Context(), advisorID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error(), "metrics_failed", nil)
		return
	}

	h.auditAccess(r.Context(), actorID, tenantID, advisorID.String(), "practice_metrics", "read", nil)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

func (h *SuccessionHandlers) CalculatePracticeValue(w http.ResponseWriter, r *http.Request) {
	actorID, tenantID, datasourceID, ok := h.authorize(w, r, permSuccessionWrite)
	if !ok {
		return
	}

	advisorID, err := uuid.Parse(chi.URLParam(r, "advisorId"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid advisor ID", "invalid_advisor_id", nil)
		return
	}

	valuation, err := h.service.CalculatePracticeMetrics(r.Context(), advisorID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error(), "calculation_failed", nil)
		return
	}

	h.auditModification(r.Context(), actorID, tenantID, advisorID.String(), "practice_metrics", "calculate", nil, valuation)
	h.emitWebhookEvent(r.Context(), "succession.practice.calculated", map[string]interface{}{"valuation": valuation}, map[string]string{
		"tenant_id":     tenantID,
		"datasource_id": datasourceID,
		"actor_id":      actorID,
		"advisor_id":    advisorID.String(),
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(valuation)
}

func (h *SuccessionHandlers) CreateSuccessionPlan(w http.ResponseWriter, r *http.Request) {
	actorID, tenantID, datasourceID, ok := h.authorize(w, r, permSuccessionWrite)
	if !ok {
		return
	}

	var input succession.SuccessionPlan
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body", "invalid_payload", nil)
		return
	}

	if err := h.service.CreateSuccessionPlan(r.Context(), &input); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error(), "create_failed", nil)
		return
	}

	h.auditModification(r.Context(), actorID, tenantID, input.PlanID.String(), "succession_plan", "create", nil, input)
	h.emitWebhookEvent(r.Context(), "succession.plan.created", map[string]interface{}{"plan": input}, map[string]string{
		"tenant_id":     tenantID,
		"datasource_id": datasourceID,
		"actor_id":      actorID,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(input)
}

func (h *SuccessionHandlers) ListSuccessionPlans(w http.ResponseWriter, r *http.Request) {
	actorID, tenantID, _, ok := h.authorize(w, r, permSuccessionRead)
	if !ok {
		return
	}

	advisorID := r.URL.Query().Get("advisorId")
	if advisorID == "" {
		writeJSONError(w, http.StatusBadRequest, "advisorId query parameter is required", "missing_advisor_id", nil)
		return
	}
	advisorUUID, err := uuid.Parse(advisorID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid advisorId", "invalid_advisor_id", nil)
		return
	}

	plans, err := h.service.GetAdvisorPlans(r.Context(), advisorUUID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error(), "list_failed", nil)
		return
	}

	h.auditAccess(r.Context(), actorID, tenantID, advisorUUID.String(), "succession_plan", "list", map[string]interface{}{
		"advisor_id": advisorID,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(plans)
}

func (h *SuccessionHandlers) GetSuccessionPlan(w http.ResponseWriter, r *http.Request) {
	if _, _, _, ok := h.authorize(w, r, permSuccessionRead); !ok {
		return
	}
	writeJSONError(w, http.StatusNotImplemented, "Plan lookup is not implemented", "not_implemented", nil)
}

func (h *SuccessionHandlers) UpdateSuccessionPlan(w http.ResponseWriter, r *http.Request) {
	if _, _, _, ok := h.authorize(w, r, permSuccessionWrite); !ok {
		return
	}
	writeJSONError(w, http.StatusNotImplemented, "Plan update is not implemented", "not_implemented", nil)
}

func (h *SuccessionHandlers) CreateClientTransition(w http.ResponseWriter, r *http.Request) {
	if _, _, _, ok := h.authorize(w, r, permSuccessionWrite); !ok {
		return
	}
	writeJSONError(w, http.StatusNotImplemented, "Client transition creation is not implemented", "not_implemented", nil)
}

func (h *SuccessionHandlers) ListTransitions(w http.ResponseWriter, r *http.Request) {
	if _, _, _, ok := h.authorize(w, r, permSuccessionRead); !ok {
		return
	}
	writeJSONError(w, http.StatusNotImplemented, "Listing transitions is not implemented", "not_implemented", nil)
}

func (h *SuccessionHandlers) UpdateTransitionStatus(w http.ResponseWriter, r *http.Request) {
	if _, _, _, ok := h.authorize(w, r, permSuccessionWrite); !ok {
		return
	}
	writeJSONError(w, http.StatusNotImplemented, "Transition updates are not implemented", "not_implemented", nil)
}

// Helper methods
func (h *SuccessionHandlers) authorize(w http.ResponseWriter, r *http.Request, permission string) (string, string, string, bool) {
	return authorizeRequest(w, r, h.secMgr, permission)
}

func (h *SuccessionHandlers) auditAccess(ctx context.Context, actorID, tenantID, objectID, objectType, action string, details map[string]interface{}) {
	logAuditAccess(ctx, h.auditSvc, actorID, tenantID, objectID, objectType, action, details)
}

func (h *SuccessionHandlers) auditModification(ctx context.Context, actorID, tenantID, objectID, objectType, action string, oldData, newData interface{}) {
	logAuditModification(ctx, h.auditSvc, actorID, tenantID, objectID, objectType, action, oldData, newData)
}

func (h *SuccessionHandlers) emitWebhookEvent(ctx context.Context, eventType string, payload map[string]interface{}, attributes map[string]string) {
	dispatchWebhookEvent(ctx, h.webhookSvc, eventType, payload, attributes)
}
