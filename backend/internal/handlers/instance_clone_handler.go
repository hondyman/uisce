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

// InstanceCloneHandler handles instance cloning requests
type InstanceCloneHandler struct {
	DB *sqlx.DB
}

// NewInstanceCloneHandler creates a new handler
func NewInstanceCloneHandler(database *sqlx.DB) *InstanceCloneHandler {
	return &InstanceCloneHandler{DB: database}
}

// RegisterRoutes registers the instance clone routes
func (h *InstanceCloneHandler) RegisterRoutes(r chi.Router) {
	r.Post("/api/instance/clone-from-goldcopy", h.HandleCloneFromGoldCopy)
	r.Post("/api/instance/sync-connections-from-goldcopy", h.HandleSyncConnectionsFromGoldCopy)
	r.Post("/api/events/tenant-instance-created", h.HandleInstanceCreatedEvent)
}

// HasuraEventPayload represents the Hasura event trigger payload
type HasuraEventPayload struct {
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

// HandleInstanceCreatedEvent handles Hasura event trigger for tenant_instance INSERT
func (h *InstanceCloneHandler) HandleInstanceCreatedEvent(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLogger().Sugar()

	var payload HasuraEventPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		logger.Errorf("Failed to decode event payload: %v", err)
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	// Only process INSERT events on tenant_instance
	if payload.Event.Op != "INSERT" || payload.Table.Name != "tenant_instance" {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "skipped", "reason": "not an INSERT on tenant_instance"})
		return
	}

	newData := payload.Event.Data.New

	// Extract instance and tenant IDs
	instanceIDStr, ok := newData["id"].(string)
	if !ok {
		logger.Error("Missing instance ID in event payload")
		http.Error(w, "Missing instance ID", http.StatusBadRequest)
		return
	}
	instanceID, err := uuid.Parse(instanceIDStr)
	if err != nil {
		logger.Errorf("Invalid instance ID: %v", err)
		http.Error(w, "Invalid instance ID", http.StatusBadRequest)
		return
	}

	tenantIDStr, ok := newData["tenant_id"].(string)
	if !ok {
		logger.Error("Missing tenant ID in event payload")
		http.Error(w, "Missing tenant ID", http.StatusBadRequest)
		return
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		logger.Errorf("Invalid tenant ID: %v", err)
		http.Error(w, "Invalid tenant ID", http.StatusBadRequest)
		return
	}

	// Check if this is a gold copy tenant (skip cloning if so)
	var isGoldCopy bool
	err = h.DB.GetContext(r.Context(), &isGoldCopy, `
		SELECT gold_copy FROM public.tenants WHERE id = $1
	`, tenantID)
	if err != nil {
		logger.Errorf("Failed to check tenant gold_copy status: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if isGoldCopy {
		logger.Infof("Skipping clone for gold copy tenant instance %s", instanceID)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "skipped", "reason": "gold_copy tenant"})
		return
	}

	// Check if instance already has a core_id (skip if already cloned)
	var existingCoreID *uuid.UUID
	err = h.DB.GetContext(r.Context(), &existingCoreID, `
		SELECT core_id FROM public.tenant_instance WHERE id = $1
	`, instanceID)
	if err != nil && err.Error() != "sql: no rows in result set" {
		logger.Errorf("Failed to check instance core_id: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if existingCoreID != nil {
		logger.Infof("Instance %s already has core_id %s, skipping clone", instanceID, *existingCoreID)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "skipped", "reason": "already cloned"})
		return
	}

	// Perform the clone
	result, err := db.CloneGoldCopyInstance(r.Context(), h.DB, tenantID, instanceID)
	if err != nil {
		logger.Errorf("Failed to clone gold copy instance: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	logger.Infof("Successfully cloned gold copy for instance %s: %d products, %d connections, %d datasources",
		instanceID, result.ProductsCloned, result.ConnectionsCloned, result.DatasourcesCloned)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// HandleCloneFromGoldCopy handles manual clone requests (for backfilling or testing)
func (h *InstanceCloneHandler) HandleCloneFromGoldCopy(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLogger().Sugar()

	var req struct {
		TenantID   string `json:"tenant_id"`
		InstanceID string `json:"instance_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		http.Error(w, "Invalid tenant_id", http.StatusBadRequest)
		return
	}

	instanceID, err := uuid.Parse(req.InstanceID)
	if err != nil {
		http.Error(w, "Invalid instance_id", http.StatusBadRequest)
		return
	}

	result, err := db.CloneGoldCopyInstance(r.Context(), h.DB, tenantID, instanceID)
	if err != nil {
		logger.Errorf("Failed to clone gold copy: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// HandleSyncConnectionsFromGoldCopy handles manual connection sync requests
func (h *InstanceCloneHandler) HandleSyncConnectionsFromGoldCopy(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLogger().Sugar()

	var req struct {
		TenantID   string `json:"tenant_id"`
		InstanceID string `json:"instance_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		http.Error(w, "Invalid tenant_id", http.StatusBadRequest)
		return
	}

	// Parse instance_id if provided, otherwise sync for all instances
	var targetInstanceIDs []uuid.UUID
	if req.InstanceID != "" {
		instanceID, err := uuid.Parse(req.InstanceID)
		if err != nil {
			http.Error(w, "Invalid instance_id", http.StatusBadRequest)
			return
		}
		targetInstanceIDs = append(targetInstanceIDs, instanceID)
	} else {
		// No instance ID provided, fetch all instances for the tenant
		err := h.DB.SelectContext(r.Context(), &targetInstanceIDs, `
			SELECT id FROM tenant_instance WHERE tenant_id = $1
		`, tenantID)
		if err != nil {
			logger.Errorf("Failed to fetch tenant instances: %v", err)
			http.Error(w, "Failed to fetch tenant instances", http.StatusInternalServerError)
			return
		}
	}

	totalResult := db.ConnectionSyncResult{}
	for _, instanceID := range targetInstanceIDs {
		result, err := db.SyncAllConnectionsForInstance(r.Context(), h.DB, tenantID, instanceID)
		if err != nil {
			// Log error but continue with other instances
			logger.Errorf("Failed to sync connections for instance %s: %v", instanceID, err)
			continue
		}
		totalResult.InstancesSynced++
		totalResult.ConnectionsCreated += result.ConnectionsCreated
		totalResult.ConnectionsUpdated += result.ConnectionsUpdated
		totalResult.ConnectionsDeleted += result.ConnectionsDeleted
		totalResult.AffectedInstanceIDs = append(totalResult.AffectedInstanceIDs, instanceID)
	}

	logger.Infof("Successfully synced connections for %d instances: %d created, %d updated",
		totalResult.InstancesSynced, totalResult.ConnectionsCreated, totalResult.ConnectionsUpdated)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(totalResult)
}
