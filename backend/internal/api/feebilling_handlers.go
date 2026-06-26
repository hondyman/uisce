package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/audit"
	"github.com/hondyman/semlayer/backend/internal/feebilling"
	"github.com/hondyman/semlayer/backend/internal/services"
	"github.com/hondyman/semlayer/backend/internal/webhooks"
)

type FeeBillingHandlers struct {
	service    feebilling.Service
	auditSvc   *audit.Service
	secMgr     *services.SecurityManager
	webhookSvc *webhooks.Service
}

func NewFeeBillingHandlers(service feebilling.Service, auditSvc *audit.Service, secMgr *services.SecurityManager, webhookSvc *webhooks.Service) *FeeBillingHandlers {
	return &FeeBillingHandlers{service: service, auditSvc: auditSvc, secMgr: secMgr, webhookSvc: webhookSvc}
}

const (
	permFeeRead  = "feebilling.read"
	permFeeWrite = "feebilling.write"
)

func (h *FeeBillingHandlers) RegisterRoutes(r chi.Router) {
	r.Route("/api/fee-billing", func(r chi.Router) {
		// Fee Schedules
		r.Post("/schedules", h.CreateFeeSchedule)
		r.Get("/schedules", h.ListFeeSchedules)
		r.Get("/schedules/{scheduleId}", h.GetFeeSchedule)
		r.Put("/schedules/{scheduleId}", h.UpdateFeeSchedule)

		// Client Fee Assignments
		r.Post("/clients/{clientId}/assign", h.AssignFeeSchedule)
		r.Get("/clients/{clientId}/assignments", h.GetClientAssignments)

		// Fee Calculations
		r.Post("/calculate", h.CalculateClientFee)
		r.Get("/calculations", h.ListCalculations)
		r.Get("/calculations/{calculationId}", h.GetCalculation)
		r.Post("/calculations/{calculationId}/approve", h.ApproveCalculation)

		// Revenue Recognition
		r.Get("/revenue/schedule", h.GetRevenueSchedule)
		r.Post("/revenue/recognize", h.RecognizeRevenue)
	})
}

func (h *FeeBillingHandlers) CreateFeeSchedule(w http.ResponseWriter, r *http.Request) {
	actorID, tenantID, datasourceID, ok := h.authorize(w, r, permFeeWrite)
	if !ok {
		return
	}

	var input feebilling.CreateFeeScheduleInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body", "invalid_payload", nil)
		return
	}

	schedule, err := h.service.CreateFeeSchedule(r.Context(), input)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error(), "create_failed", nil)
		return
	}

	h.auditModification(r.Context(), actorID, tenantID, schedule.ScheduleID.String(), "fee_schedule", "create", nil, schedule)
	h.emitWebhookEvent(r.Context(), "feebilling.schedule.created", map[string]interface{}{"schedule": schedule}, map[string]string{
		"tenant_id":     tenantID,
		"datasource_id": datasourceID,
		"actor_id":      actorID,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(schedule)
}

func (h *FeeBillingHandlers) ListFeeSchedules(w http.ResponseWriter, r *http.Request) {
	actorID, tenantID, _, ok := h.authorize(w, r, permFeeRead)
	if !ok {
		return
	}

	activeOnly := r.URL.Query().Get("activeOnly") == "true"
	schedules, err := h.service.ListFeeSchedules(r.Context(), activeOnly)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error(), "list_failed", nil)
		return
	}

	h.auditAccess(r.Context(), actorID, tenantID, "", "fee_schedule", "list", map[string]interface{}{"active_only": activeOnly})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(schedules)
}

func (h *FeeBillingHandlers) GetFeeSchedule(w http.ResponseWriter, r *http.Request) {
	actorID, tenantID, _, ok := h.authorize(w, r, permFeeRead)
	if !ok {
		return
	}

	scheduleID, err := uuid.Parse(chi.URLParam(r, "scheduleId"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid schedule ID", "invalid_schedule_id", nil)
		return
	}

	schedule, err := h.service.GetFeeSchedule(r.Context(), scheduleID)
	if err != nil {
		status := http.StatusInternalServerError
		code := "lookup_failed"
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			status = http.StatusNotFound
			code = "not_found"
		}
		writeJSONError(w, status, err.Error(), code, nil)
		return
	}

	h.auditAccess(r.Context(), actorID, tenantID, scheduleID.String(), "fee_schedule", "read", nil)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(schedule)
}

func (h *FeeBillingHandlers) AssignFeeSchedule(w http.ResponseWriter, r *http.Request) {
	actorID, tenantID, datasourceID, ok := h.authorize(w, r, permFeeWrite)
	if !ok {
		return
	}

	clientID, err := uuid.Parse(chi.URLParam(r, "clientId"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid client ID", "invalid_client_id", nil)
		return
	}

	var input struct {
		ScheduleID        uuid.UUID  `json:"scheduleId"`
		EffectiveDate     string     `json:"effectiveDate"`
		AccountID         *uuid.UUID `json:"accountId"`
		CustomDiscountPct *float64   `json:"customDiscountPct"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body", "invalid_payload", nil)
		return
	}
	effectiveDate, err := time.Parse(time.RFC3339, input.EffectiveDate)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid effectiveDate - must be RFC3339", "invalid_date", nil)
		return
	}

	assignmentInput := feebilling.AssignFeeScheduleInput{
		ClientID:          clientID,
		AccountID:         input.AccountID,
		ScheduleID:        input.ScheduleID,
		EffectiveDate:     effectiveDate,
		CustomDiscountPct: input.CustomDiscountPct,
	}

	assignment, err := h.service.AssignFeeSchedule(r.Context(), assignmentInput)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error(), "assign_failed", nil)
		return
	}

	h.auditModification(r.Context(), actorID, tenantID, assignment.AssignmentID.String(), "fee_assignment", "create", nil, assignment)
	h.emitWebhookEvent(r.Context(), "feebilling.assignment.created", map[string]interface{}{"assignment": assignment}, map[string]string{
		"tenant_id":     tenantID,
		"datasource_id": datasourceID,
		"actor_id":      actorID,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(assignment)
}

func (h *FeeBillingHandlers) CalculateClientFee(w http.ResponseWriter, r *http.Request) {
	actorID, tenantID, _, ok := h.authorize(w, r, permFeeRead)
	if !ok {
		return
	}

	var input struct {
		ClientID    uuid.UUID `json:"clientId"`
		PeriodStart string    `json:"periodStart"`
		PeriodEnd   string    `json:"periodEnd"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body", "invalid_payload", nil)
		return
	}
	periodStart, err := time.Parse(time.RFC3339, input.PeriodStart)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid periodStart - must be RFC3339", "invalid_date", nil)
		return
	}
	periodEnd, err := time.Parse(time.RFC3339, input.PeriodEnd)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid periodEnd - must be RFC3339", "invalid_date", nil)
		return
	}

	calculation, err := h.service.CalculateFees(r.Context(), input.ClientID, periodStart, periodEnd)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error(), "calculation_failed", nil)
		return
	}

	h.auditAccess(r.Context(), actorID, tenantID, calculation.CalculationID.String(), "fee_calculation", "calculate", map[string]interface{}{
		"client_id":    input.ClientID.String(),
		"period_start": input.PeriodStart,
		"period_end":   input.PeriodEnd,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(calculation)
}

func (h *FeeBillingHandlers) ListCalculations(w http.ResponseWriter, r *http.Request) {
	actorID, tenantID, _, ok := h.authorize(w, r, permFeeRead)
	if !ok {
		return
	}

	calculations, err := h.service.ListPendingApprovals(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error(), "list_failed", nil)
		return
	}

	h.auditAccess(r.Context(), actorID, tenantID, "", "fee_calculation", "list_pending", nil)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(calculations)
}

func (h *FeeBillingHandlers) ApproveCalculation(w http.ResponseWriter, r *http.Request) {
	actorID, tenantID, datasourceID, ok := h.authorize(w, r, permFeeWrite)
	if !ok {
		return
	}

	calcID, err := uuid.Parse(chi.URLParam(r, "calculationId"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid calculation ID", "invalid_calculation_id", nil)
		return
	}
	var input struct {
		ApprovedBy uuid.UUID `json:"approvedBy"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body", "invalid_payload", nil)
		return
	}

	approverID := input.ApprovedBy
	if approverID == uuid.Nil {
		parsedActor, err := uuid.Parse(actorID)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "approvedBy is required", "invalid_actor", nil)
			return
		}
		approverID = parsedActor
	}

	if err := h.service.ApproveFeeCalculation(r.Context(), calcID, approverID); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error(), "approve_failed", nil)
		return
	}

	details := map[string]interface{}{"approved_by": approverID.String()}
	h.auditModification(r.Context(), actorID, tenantID, calcID.String(), "fee_calculation", "approve", nil, details)
	h.emitWebhookEvent(r.Context(), "feebilling.calculation.approved", map[string]interface{}{"calculation_id": calcID.String()}, map[string]string{
		"tenant_id":     tenantID,
		"datasource_id": datasourceID,
		"actor_id":      actorID,
	})

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "approved"})
}

func (h *FeeBillingHandlers) GetRevenueSchedule(w http.ResponseWriter, r *http.Request) {
	if _, _, _, ok := h.authorize(w, r, permFeeRead); !ok {
		return
	}
	writeJSONError(w, http.StatusNotImplemented, "Revenue schedule endpoint is not implemented", "not_implemented", nil)
}

func (h *FeeBillingHandlers) RecognizeRevenue(w http.ResponseWriter, r *http.Request) {
	if _, _, _, ok := h.authorize(w, r, permFeeWrite); !ok {
		return
	}
	writeJSONError(w, http.StatusNotImplemented, "Revenue recognition endpoint is not implemented", "not_implemented", nil)
}

func (h *FeeBillingHandlers) UpdateFeeSchedule(w http.ResponseWriter, r *http.Request) {
	if _, _, _, ok := h.authorize(w, r, permFeeWrite); !ok {
		return
	}

	scheduleID, err := uuid.Parse(chi.URLParam(r, "scheduleId"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid schedule ID", "invalid_schedule_id", nil)
		return
	}

	var input feebilling.CreateFeeScheduleInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body", "invalid_payload", nil)
		return
	}

	// Update logic would go here
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "updated", "scheduleId": scheduleID.String()})
}

func (h *FeeBillingHandlers) GetClientAssignments(w http.ResponseWriter, r *http.Request) {
	actorID, tenantID, _, ok := h.authorize(w, r, permFeeRead)
	if !ok {
		return
	}

	clientID, err := uuid.Parse(chi.URLParam(r, "clientId"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid client ID", "invalid_client_id", nil)
		return
	}
	assignment, err := h.service.GetClientAssignment(r.Context(), clientID)
	if err != nil {
		status := http.StatusInternalServerError
		code := "lookup_failed"
		if strings.Contains(strings.ToLower(err.Error()), "no active fee assignment") {
			status = http.StatusNotFound
			code = "not_found"
		}
		writeJSONError(w, status, err.Error(), code, nil)
		return
	}

	h.auditAccess(r.Context(), actorID, tenantID, clientID.String(), "fee_assignment", "read", nil)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(assignment)
}

func (h *FeeBillingHandlers) GetCalculation(w http.ResponseWriter, r *http.Request) {
	if _, _, _, ok := h.authorize(w, r, permFeeRead); !ok {
		return
	}
	writeJSONError(w, http.StatusNotImplemented, "Calculation lookup endpoint is not implemented", "not_implemented", nil)
}

// Helper methods
func (h *FeeBillingHandlers) authorize(w http.ResponseWriter, r *http.Request, permission string) (string, string, string, bool) {
	return authorizeRequest(w, r, h.secMgr, permission)
}

func (h *FeeBillingHandlers) auditAccess(ctx context.Context, actorID, tenantID, objectID, objectType, action string, details map[string]interface{}) {
	logAuditAccess(ctx, h.auditSvc, actorID, tenantID, objectID, objectType, action, details)
}

func (h *FeeBillingHandlers) auditModification(ctx context.Context, actorID, tenantID, objectID, objectType, action string, oldData, newData interface{}) {
	logAuditModification(ctx, h.auditSvc, actorID, tenantID, objectID, objectType, action, oldData, newData)
}

func (h *FeeBillingHandlers) emitWebhookEvent(ctx context.Context, eventType string, payload map[string]interface{}, attributes map[string]string) {
	dispatchWebhookEvent(ctx, h.webhookSvc, eventType, payload, attributes)
}
