package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/db"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/jmoiron/sqlx"
)

// GoldCopySyncHandler handles cascading sync events for Gold Copy entities
type GoldCopySyncHandler struct {
	DB *sqlx.DB
}

// NewGoldCopySyncHandler creates a new handler
func NewGoldCopySyncHandler(database *sqlx.DB) *GoldCopySyncHandler {
	return &GoldCopySyncHandler{DB: database}
}

// RegisterRoutes registers the Gold Copy sync routes
func (h *GoldCopySyncHandler) RegisterRoutes(r chi.Router) {
	r.Post("/api/events/gold-copy-entity-changed", h.HandleGoldCopyEntityChanged)
}

// GoldCopySyncEventPayload represents the standard Hasura event trigger payload
type GoldCopySyncEventPayload struct {
	Event struct {
		Data struct {
			New map[string]interface{} `json:"new"`
			Old map[string]interface{} `json:"old"`
		} `json:"data"`
		Op string `json:"op"`
	} `json:"event"`
	Table struct {
		Name   string `json:"name"`
		Schema string `json:"schema"`
	} `json:"table"`
}

// HandleGoldCopyEntityChanged handles Hasura event trigger for Gold Copy entity DELETE
func (h *GoldCopySyncHandler) HandleGoldCopyEntityChanged(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLogger().Sugar()

	var payload GoldCopySyncEventPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		logger.Errorf("Failed to decode event payload: %v", err)
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	// Only process DELETE operations
	if payload.Event.Op != "DELETE" {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "skipped", "reason": "not a DELETE operation"})
		return
	}

	// Get the deleted entity data
	entityData := payload.Event.Data.Old
	if entityData == nil {
		logger.Error("No old data in DELETE event")
		http.Error(w, "Missing old data", http.StatusBadRequest)
		return
	}

	// Extract tenant_id to verify this is from Gold Copy
	tenantIDStr, ok := entityData["tenant_id"].(string)
	if !ok {
		logger.Error("Missing tenant_id in event payload")
		http.Error(w, "Missing tenant_id", http.StatusBadRequest)
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		logger.Errorf("Invalid tenant_id: %v", err)
		http.Error(w, "Invalid tenant_id", http.StatusBadRequest)
		return
	}

	// Check if this tenant is Gold Copy
	var isGoldCopy bool
	err = h.DB.GetContext(r.Context(), &isGoldCopy, `
		SELECT gold_copy FROM public.tenants WHERE id = $1
	`, tenantID)
	if err != nil {
		logger.Errorf("Failed to check tenant gold_copy status: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if !isGoldCopy {
		logger.Infof("Entity deleted from non-gold-copy tenant %s, skipping sync", tenantID)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "skipped", "reason": "not gold_copy tenant"})
		return
	}

	// Extract entity ID
	entityIDStr, ok := entityData["id"].(string)
	if !ok {
		logger.Error("Missing entity id in event payload")
		http.Error(w, "Missing entity id", http.StatusBadRequest)
		return
	}

	entityID, err := uuid.Parse(entityIDStr)
	if err != nil {
		logger.Errorf("Invalid entity id: %v", err)
		http.Error(w, "Invalid entity id", http.StatusBadRequest)
		return
	}

	// Route to appropriate sync function based on table
	var result interface{}
	tableName := payload.Table.Name

	logger.Infof("Processing Gold Copy DELETE for %s: %s", tableName, entityID)

	switch tableName {
	case "tenant_instance":
		result, err = db.SyncGoldCopyInstanceDeletion(r.Context(), h.DB, entityID)
	case "tenant_product":
		result, err = db.SyncGoldCopyProductDeletion(r.Context(), h.DB, entityID)
	case "connections":
		// Connections are handled by the existing connection_sync_handler
		// but we can also handle them here for completeness
		result, err = db.SyncGoldCopyConnectionToAllInstances(r.Context(), h.DB, entityID, "DELETE")
	default:
		logger.Warnf("Unknown table for Gold Copy sync: %s", tableName)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "skipped", "reason": "unknown table"})
		return
	}

	if err != nil {
		logger.Errorf("Failed to sync Gold Copy deletion: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	logger.Infof("Successfully synced Gold Copy %s deletion: %s", tableName, entityID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"table":  tableName,
		"result": result,
	})
}
