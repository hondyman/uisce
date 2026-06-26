package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/audit"
	"github.com/hondyman/semlayer/backend/internal/household"
	"github.com/hondyman/semlayer/backend/internal/services"
	"github.com/hondyman/semlayer/backend/internal/webhooks"
)

type HouseholdHandlers struct {
	service    household.Service
	auditSvc   *audit.Service
	secMgr     *services.SecurityManager
	webhookSvc *webhooks.Service
}

func NewHouseholdHandlers(service household.Service, auditSvc *audit.Service, secMgr *services.SecurityManager, webhookSvc *webhooks.Service) *HouseholdHandlers {
	return &HouseholdHandlers{service: service, auditSvc: auditSvc, secMgr: secMgr, webhookSvc: webhookSvc}
}

const (
	permHouseholdRead  = "households.read"
	permHouseholdWrite = "households.write"
)

func (h *HouseholdHandlers) RegisterRoutes(r chi.Router) {
	r.Route("/api/households", func(r chi.Router) {
		// Entities
		r.Post("/{householdId}/entities", h.CreateEntity)
		r.Get("/{householdId}/entities", h.ListEntities)
		r.Get("/entities/{entityId}", h.GetEntity)
		r.Put("/entities/{entityId}", h.UpdateEntity)
		r.Delete("/entities/{entityId}", h.DeleteEntity)

		// Transfers
		r.Post("/{householdId}/transfers", h.CreateTransfer)
		r.Get("/{householdId}/transfers", h.ListTransfers)

		// Hierarchy & Views
		r.Get("/{householdId}/hierarchy", h.GetHierarchy)
		r.Get("/{householdId}/consolidated", h.GetConsolidatedView)
	})
}

func (h *HouseholdHandlers) CreateEntity(w http.ResponseWriter, r *http.Request) {
	actorID, tenantID, datasourceID, ok := h.authorize(w, r, permHouseholdWrite)
	if !ok {
		return
	}

	householdID, err := uuid.Parse(chi.URLParam(r, "householdId"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid household ID", "invalid_household_id", nil)
		return
	}

	var input household.Entity
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body", "invalid_payload", nil)
		return
	}

	input.HouseholdID = householdID

	if err := h.service.CreateEntity(r.Context(), &input); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error(), "create_failed", nil)
		return
	}

	h.auditModification(r.Context(), actorID, tenantID, input.EntityID.String(), "household_entity", "create", nil, input)
	h.emitWebhookEvent(r.Context(), "households.entity.created", map[string]interface{}{"entity": input}, map[string]string{
		"tenant_id":     tenantID,
		"datasource_id": datasourceID,
		"actor_id":      actorID,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(input)
}

func (h *HouseholdHandlers) ListEntities(w http.ResponseWriter, r *http.Request) {
	actorID, tenantID, _, ok := h.authorize(w, r, permHouseholdRead)
	if !ok {
		return
	}

	householdID, err := uuid.Parse(chi.URLParam(r, "householdId"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid household ID", "invalid_household_id", nil)
		return
	}

	entities, err := h.service.GetHouseholdEntities(r.Context(), householdID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error(), "list_failed", nil)
		return
	}

	h.auditAccess(r.Context(), actorID, tenantID, "", "household_entity", "list", map[string]interface{}{
		"household_id": householdID.String(),
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entities)
}

func (h *HouseholdHandlers) GetEntity(w http.ResponseWriter, r *http.Request) {
	if _, _, _, ok := h.authorize(w, r, permHouseholdRead); !ok {
		return
	}
	writeJSONError(w, http.StatusNotImplemented, "GetEntity endpoint is not implemented", "not_implemented", nil)
}

func (h *HouseholdHandlers) UpdateEntity(w http.ResponseWriter, r *http.Request) {
	if _, _, _, ok := h.authorize(w, r, permHouseholdWrite); !ok {
		return
	}
	writeJSONError(w, http.StatusNotImplemented, "UpdateEntity endpoint is not implemented", "not_implemented", nil)
}

func (h *HouseholdHandlers) DeleteEntity(w http.ResponseWriter, r *http.Request) {
	if _, _, _, ok := h.authorize(w, r, permHouseholdWrite); !ok {
		return
	}
	writeJSONError(w, http.StatusNotImplemented, "DeleteEntity endpoint is not implemented", "not_implemented", nil)
}

func (h *HouseholdHandlers) CreateTransfer(w http.ResponseWriter, r *http.Request) {
	actorID, tenantID, datasourceID, ok := h.authorize(w, r, permHouseholdWrite)
	if !ok {
		return
	}

	if _, err := uuid.Parse(chi.URLParam(r, "householdId")); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid household ID", "invalid_household_id", nil)
		return
	}

	var input household.InterEntityTransfer
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body", "invalid_payload", nil)
		return
	}
	if err := h.service.RecordTransfer(r.Context(), &input); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error(), "create_failed", nil)
		return
	}

	h.auditModification(r.Context(), actorID, tenantID, input.TransferID.String(), "household_transfer", "create", nil, input)
	h.emitWebhookEvent(r.Context(), "households.transfer.created", map[string]interface{}{"transfer": input}, map[string]string{
		"tenant_id":     tenantID,
		"datasource_id": datasourceID,
		"actor_id":      actorID,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(input)
}

func (h *HouseholdHandlers) ListTransfers(w http.ResponseWriter, r *http.Request) {
	if _, _, _, ok := h.authorize(w, r, permHouseholdRead); !ok {
		return
	}
	writeJSONError(w, http.StatusNotImplemented, "ListTransfers endpoint is not implemented", "not_implemented", nil)
}

func (h *HouseholdHandlers) GetHierarchy(w http.ResponseWriter, r *http.Request) {
	actorID, tenantID, _, ok := h.authorize(w, r, permHouseholdRead)
	if !ok {
		return
	}

	householdID, err := uuid.Parse(chi.URLParam(r, "householdId"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid household ID", "invalid_household_id", nil)
		return
	}

	hierarchy, err := h.service.GetHouseholdHierarchy(r.Context(), householdID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error(), "lookup_failed", nil)
		return
	}

	h.auditAccess(r.Context(), actorID, tenantID, householdID.String(), "household_hierarchy", "read", map[string]interface{}{
		"household_id": householdID.String(),
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(hierarchy)
}

func (h *HouseholdHandlers) GetConsolidatedView(w http.ResponseWriter, r *http.Request) {
	if _, _, _, ok := h.authorize(w, r, permHouseholdRead); !ok {
		return
	}
	writeJSONError(w, http.StatusNotImplemented, "Consolidated view endpoint is not implemented", "not_implemented", nil)
}

// Helper methods
func (h *HouseholdHandlers) authorize(w http.ResponseWriter, r *http.Request, permission string) (string, string, string, bool) {
	return authorizeRequest(w, r, h.secMgr, permission)
}

func (h *HouseholdHandlers) auditAccess(ctx context.Context, actorID, tenantID, objectID, objectType, action string, details map[string]interface{}) {
	logAuditAccess(ctx, h.auditSvc, actorID, tenantID, objectID, objectType, action, details)
}

func (h *HouseholdHandlers) auditModification(ctx context.Context, actorID, tenantID, objectID, objectType, action string, oldData, newData interface{}) {
	logAuditModification(ctx, h.auditSvc, actorID, tenantID, objectID, objectType, action, oldData, newData)
}

func (h *HouseholdHandlers) emitWebhookEvent(ctx context.Context, eventType string, payload map[string]interface{}, attributes map[string]string) {
	dispatchWebhookEvent(ctx, h.webhookSvc, eventType, payload, attributes)
}
