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

// ConnectionSyncHandler handles connection synchronization events
type ConnectionSyncHandler struct {
	DB *sqlx.DB
}

// NewConnectionSyncHandler creates a new handler
func NewConnectionSyncHandler(database *sqlx.DB) *ConnectionSyncHandler {
	return &ConnectionSyncHandler{DB: database}
}

// RegisterRoutes registers the connection sync routes
func (h *ConnectionSyncHandler) RegisterRoutes(r chi.Router) {
	r.Post("/api/events/connection-changed", h.HandleConnectionChanged)
}

// HandleConnectionChanged handles Hasura event trigger for connection INSERT/UPDATE/DELETE
func (h *ConnectionSyncHandler) HandleConnectionChanged(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLogger().Sugar()

	var payload struct {
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

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		logger.Errorf("Failed to decode event payload: %v", err)
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	// Only process events on connections table
	if payload.Table.Name != "connections" {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "skipped", "reason": "not connections table"})
		return
	}

	// Get connection data (use new for INSERT/UPDATE, old for DELETE)
	var connectionData map[string]interface{}
	if payload.Event.Op == "DELETE" {
		connectionData = payload.Event.Data.Old
	} else {
		connectionData = payload.Event.Data.New
	}

	// Extract tenant_id to check if this is a Gold Copy connection
	tenantIDStr, ok := connectionData["tenant_id"].(string)
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
		logger.Infof("Connection event for non-gold-copy tenant %s, skipping sync", tenantID)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "skipped", "reason": "not gold_copy tenant"})
		return
	}

	// Extract connection ID
	connectionIDStr, ok := connectionData["id"].(string)
	if !ok {
		logger.Error("Missing connection id in event payload")
		http.Error(w, "Missing connection id", http.StatusBadRequest)
		return
	}

	connectionID, err := uuid.Parse(connectionIDStr)
	if err != nil {
		logger.Errorf("Invalid connection id: %v", err)
		http.Error(w, "Invalid connection id", http.StatusBadRequest)
		return
	}

	logger.Infof("Gold Copy connection %s changed (op: %s), syncing to all instances", connectionID, payload.Event.Op)

	// Sync this connection to all non-Gold Copy instances
	result, err := db.SyncGoldCopyConnectionToAllInstances(r.Context(), h.DB, connectionID, payload.Event.Op)
	if err != nil {
		logger.Errorf("Failed to sync connection: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	logger.Infof("Successfully synced connection %s to %d instances", connectionID, result.InstancesSynced)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
