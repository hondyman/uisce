package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"log"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/audit"
)

type AuditHistoryHandler struct {
	tracker      *audit.BitemporalTracker
	asyncService *audit.AsyncAuditService
}

func NewAuditHistoryHandler(tracker *audit.BitemporalTracker, asyncService *audit.AsyncAuditService) *AuditHistoryHandler {
	return &AuditHistoryHandler{
		tracker:      tracker,
		asyncService: asyncService,
	}
}

// AuditHasuraEventPayload represents the Hasura event trigger payload
// Renamed to avoid conflict with instance_clone_handler.go
type AuditHasuraEventPayload struct {
	Event struct {
		SessionVariables map[string]string `json:"session_variables"`
		Op               string            `json:"op"`
		Data             struct {
			New map[string]interface{} `json:"new"`
			Old map[string]interface{} `json:"old"`
		} `json:"data"`
	} `json:"event"`
	Table struct {
		Name   string `json:"name"`
		Schema string `json:"schema"`
	} `json:"table"`
}

// RegisterRoutes registers audit history routes
func (h *AuditHistoryHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/audit", func(r chi.Router) {
		r.Get("/history/{entityType}/{entityId}", h.HandleGetEntityHistory)
		r.Get("/history/{entityType}/{entityId}/at/{timestamp}", h.HandleGetEntityAtTime)
		r.Post("/restore/{entityType}/{entityId}", h.HandleRestoreEntity)
		r.Get("/changes", h.HandleGetAuditChanges)
		// Hasura Event Trigger Endpoint
		// Hasura needs to be configured to send POST requests to this endpoint
		r.Post("/events/tenant-update", h.HandleTenantUpdateEvent)
	})
}

// HandleGetEntityHistory retrieves the full history of an entity
// GET /api/audit/history/{entityType}/{entityId}
// Query parameters: from, to, validFrom, validTo, includeDeleted, limit, offset
func (h *AuditHistoryHandler) HandleGetEntityHistory(w http.ResponseWriter, r *http.Request) {
	entityType := chi.URLParam(r, "entityType")
	entityID := chi.URLParam(r, "entityId")

	// Parse query parameters
	filters := audit.HistoryFilters{
		IncludeDeleted: r.URL.Query().Get("includeDeleted") == "true",
		Limit:          100, // default
		Offset:         0,
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			filters.Limit = limit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			filters.Offset = offset
		}
	}

	if fromStr := r.URL.Query().Get("from"); fromStr != "" {
		if from, err := time.Parse(time.RFC3339, fromStr); err == nil {
			filters.From = &from
		}
	}

	if toStr := r.URL.Query().Get("to"); toStr != "" {
		if to, err := time.Parse(time.RFC3339, toStr); err == nil {
			filters.To = &to
		}
	}

	if validFromStr := r.URL.Query().Get("validFrom"); validFromStr != "" {
		if validFrom, err := time.Parse(time.RFC3339, validFromStr); err == nil {
			filters.ValidFrom = &validFrom
		}
	}

	if validToStr := r.URL.Query().Get("validTo"); validToStr != "" {
		if validTo, err := time.Parse(time.RFC3339, validToStr); err == nil {
			filters.ValidTo = &validTo
		}
	}

	// Get history
	snapshots, err := h.tracker.GetEntityHistory(r.Context(), entityType, entityID, filters)
	if err != nil {
		log.Printf("Failed to get entity history: %v", err)
		http.Error(w, "Failed to get entity history", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"entity_type": entityType,
		"entity_id":   entityID,
		"history":     snapshots,
		"count":       len(snapshots),
	})
}

// HandleGetEntityAtTime retrieves entity state at a specific point in time
// GET /api/audit/history/{entityType}/{entityId}/at/{timestamp}
func (h *AuditHistoryHandler) HandleGetEntityAtTime(w http.ResponseWriter, r *http.Request) {
	entityType := chi.URLParam(r, "entityType")
	entityID := chi.URLParam(r, "entityId")
	timestampStr := chi.URLParam(r, "timestamp")

	// Parse timestamp
	asOf, err := time.Parse(time.RFC3339, timestampStr)
	if err != nil {
		http.Error(w, "Invalid timestamp format. Use RFC3339", http.StatusBadRequest)
		return
	}

	// Get entity at time
	snapshot, err := h.tracker.GetEntityAtTime(r.Context(), entityType, entityID, asOf)
	if err != nil {
		log.Printf("Failed to get entity at time: %v", err)
		http.Error(w, "Failed to get entity at time", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(snapshot)
}

// HandleRestoreEntity restores an entity to a previous state
// POST /api/audit/restore/{entityType}/{entityId}
// Body: { "restoreToTime": "2024-01-01T00:00:00Z", "reason": "Accidental deletion" }
func (h *AuditHistoryHandler) HandleRestoreEntity(w http.ResponseWriter, r *http.Request) {
	entityType := chi.URLParam(r, "entityType")
	entityID := chi.URLParam(r, "entityId")

	// Parse request body
	var req struct {
		RestoreToTime string `json:"restoreToTime"`
		Reason        string `json:"reason"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Reason == "" {
		http.Error(w, "Reason is required for restore operations", http.StatusBadRequest)
		return
	}

	// Parse restore time
	restoreToTime, err := time.Parse(time.RFC3339, req.RestoreToTime)
	if err != nil {
		http.Error(w, "Invalid restoreToTime format. Use RFC3339", http.StatusBadRequest)
		return
	}

	// Perform restore
	err = h.tracker.RestoreEntityToTime(r.Context(), entityType, entityID, restoreToTime, req.Reason)
	if err != nil {
		log.Printf("Failed to restore entity: %v", err)
		http.Error(w, "Failed to restore entity", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":         true,
		"entity_type":     entityType,
		"entity_id":       entityID,
		"restore_to_time": req.RestoreToTime,
		"reason":          req.Reason,
	})
}

// HandleGetAuditChanges retrieves all audit changes with filters
// GET /api/audit/changes
// Query parameters: entityType, from, to, changedBy, changeType, limit, offset
func (h *AuditHistoryHandler) HandleGetAuditChanges(w http.ResponseWriter, r *http.Request) {
	entityType := r.URL.Query().Get("entityType")
	limitStr := r.URL.Query().Get("limit")
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	limit := 100
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	var from, to *time.Time
	if fromStr != "" {
		if t, err := time.Parse(time.RFC3339, fromStr); err == nil {
			from = &t
		}
	}
	if toStr != "" {
		if t, err := time.Parse(time.RFC3339, toStr); err == nil {
			to = &t
		}
	}

	changes, err := h.tracker.GetRecentChanges(r.Context(), from, to, entityType, limit)
	if err != nil {
		log.Printf("Failed to get recent changes: %v", err)
		http.Error(w, "Failed to get recent changes", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"changes": changes,
		"count":   len(changes),
	})
}

// HandleTenantUpdateEvent handles Hasura event trigger for tenants updates
// POST /api/audit/events/tenant-update
func (h *AuditHistoryHandler) HandleTenantUpdateEvent(w http.ResponseWriter, r *http.Request) {
	var payload AuditHasuraEventPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	// Verify table
	if payload.Table.Name != "tenants" {
		w.WriteHeader(http.StatusOK) // Ack to Hasura
		return
	}

	op := payload.Event.Op
	data := payload.Event.Data.New
	if op == "DELETE" {
		data = payload.Event.Data.Old
	}

	idStr, ok := data["id"].(string)
	if !ok {
		// Log error?
		w.WriteHeader(http.StatusOK)
		return
	}

	// Determine User
	userID := "system"
	if val, ok := payload.Event.SessionVariables["x-hasura-user-id"]; ok && val != "" {
		userID = val
	}

	change := audit.EntityChange{
		EntityType:   "tenant",
		EntityID:     idStr,
		ChangeType:   op,
		ValidFrom:    time.Now(),
		EntityData:   data,
		ChangedBy:    userID,
		ChangeReason: fmt.Sprintf("Hasura Event Trigger: %s", op),
	}

	// Use generic unique id if id is not UUID (though tenant id usually is)
	if _, err := uuid.Parse(idStr); err != nil {
		// If not UUID, we might have issues if bitemporal tracker expects UUID.
		// But bitemporal tracker uses string for EntityID, so it should be fine.
	}

	h.asyncService.TrackChangeAsync(change)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}
