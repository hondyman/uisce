package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"calendar-service/internal/services"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// ExternalSyncHandler handles external sync configuration and execution
type ExternalSyncHandler struct {
	service services.ExternalSyncServiceTenantAware
	logger  *logrus.Entry
}

// NewExternalSyncHandler creates a new external sync handler
func NewExternalSyncHandler(svc services.ExternalSyncServiceTenantAware, logger *logrus.Entry) *ExternalSyncHandler {
	return &ExternalSyncHandler{
		service: svc,
		logger:  logger.WithField("handler", "external_sync"),
	}
}

// CreateSyncConfigRequest defines the request body for creating sync config
type CreateSyncConfigRequest struct {
	ProfileID     string `json:"profile_id"`
	Provider      string `json:"provider"`          // 'nager_date', 'calendarific'
	CountryCode   string `json:"country_code"`      // ISO 3166-1 alpha-2
	APIKey        string `json:"api_key,omitempty"` // Optional, for calendarific
	SyncEnabled   bool   `json:"sync_enabled"`
	SyncFrequency string `json:"sync_frequency"` // 'weekly', 'monthly', 'yearly'
}

// UpdateSyncConfigRequest defines the request body for updating sync config
type UpdateSyncConfigRequest struct {
	SyncEnabled   *bool   `json:"sync_enabled,omitempty"`
	SyncFrequency *string `json:"sync_frequency,omitempty"`
	CountryCode   *string `json:"country_code,omitempty"`
}

// CreateSyncConfig handles POST /api/v1/external-sync
// Creates a new external sync configuration for a profile
func (h *ExternalSyncHandler) CreateSyncConfig(w http.ResponseWriter, r *http.Request) {
	var req CreateSyncConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.WithError(err).Warn("Invalid request body")
		writeJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate required fields
	if req.ProfileID == "" || req.Provider == "" || req.CountryCode == "" {
		writeJSONError(w, http.StatusBadRequest, "profile_id, provider, and country_code are required")
		return
	}

	// Get tenant from header
	tenantIDStr := r.Header.Get("X-Hasura-Tenant-Id")
	if tenantIDStr == "" {
		writeJSONError(w, http.StatusBadRequest, "tenant_id required (X-Hasura-Tenant-Id header)")
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid tenant_id format")
		return
	}

	profileID, err := uuid.Parse(req.ProfileID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid profile_id format")
		return
	}

	// Prepare sync config
	config := &services.ExternalSyncConfig{
		ProfileID:       profileID,
		Provider:        services.ExternalSyncProvider(req.Provider),
		CountryCode:     req.CountryCode,
		APIKeyEncrypted: req.APIKey, // In production, encrypt this
		SyncEnabled:     req.SyncEnabled,
		SyncFrequency:   services.SyncFrequency(req.SyncFrequency),
	}

	// Create via service
	result, err := h.service.CreateSyncConfig(r.Context(), tenantID, config)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create sync config")
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create sync config: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(result)
}

// GetSyncConfig handles GET /api/v1/external-sync/{id}
// Retrieves a sync configuration
func (h *ExternalSyncHandler) GetSyncConfig(w http.ResponseWriter, r *http.Request) {
	configIDStr := r.PathValue("id")
	configID, err := uuid.Parse(configIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid config_id format")
		return
	}

	// Get tenant from header
	tenantIDStr := r.Header.Get("X-Hasura-Tenant-Id")
	if tenantIDStr == "" {
		writeJSONError(w, http.StatusBadRequest, "tenant_id required (X-Hasura-Tenant-Id header)")
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid tenant_id format")
		return
	}

	// Get via service
	config, err := h.service.GetSyncConfig(r.Context(), tenantID, configID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get sync config")
		writeJSONError(w, http.StatusNotFound, "sync config not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(config)
}

// ListSyncConfigs handles GET /api/v1/external-sync
// Lists all sync configurations for a tenant
func (h *ExternalSyncHandler) ListSyncConfigs(w http.ResponseWriter, r *http.Request) {
	// Get tenant from header
	tenantIDStr := r.Header.Get("X-Hasura-Tenant-Id")
	if tenantIDStr == "" {
		writeJSONError(w, http.StatusBadRequest, "tenant_id required (X-Hasura-Tenant-Id header)")
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid tenant_id format")
		return
	}

	// List via service
	configs, err := h.service.ListSyncConfigs(r.Context(), tenantID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list sync configs")
		writeJSONError(w, http.StatusInternalServerError, "Failed to list sync configs")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(configs)
}

// ListSyncConfigsByProfile handles GET /api/v1/profiles/{profileId}/external-sync
// Lists sync configurations for a specific profile
func (h *ExternalSyncHandler) ListSyncConfigsByProfile(w http.ResponseWriter, r *http.Request) {
	profileIDStr := r.PathValue("profileId")
	profileID, err := uuid.Parse(profileIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid profile_id format")
		return
	}

	// Get tenant from header
	tenantIDStr := r.Header.Get("X-Hasura-Tenant-Id")
	if tenantIDStr == "" {
		writeJSONError(w, http.StatusBadRequest, "tenant_id required (X-Hasura-Tenant-Id header)")
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid tenant_id format")
		return
	}

	// List via service
	configs, err := h.service.ListSyncConfigsByProfile(r.Context(), tenantID, profileID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list sync configs")
		writeJSONError(w, http.StatusInternalServerError, "Failed to list sync configs")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(configs)
}

// UpdateSyncConfig handles PUT /api/v1/external-sync/{id}
// Updates a sync configuration
func (h *ExternalSyncHandler) UpdateSyncConfig(w http.ResponseWriter, r *http.Request) {
	configIDStr := r.PathValue("id")
	configID, err := uuid.Parse(configIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid config_id format")
		return
	}

	var req UpdateSyncConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.WithError(err).Warn("Invalid request body")
		writeJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get tenant from header
	tenantIDStr := r.Header.Get("X-Hasura-Tenant-Id")
	if tenantIDStr == "" {
		writeJSONError(w, http.StatusBadRequest, "tenant_id required (X-Hasura-Tenant-Id header)")
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid tenant_id format")
		return
	}

	// Build updates map
	updates := make(map[string]interface{})
	if req.SyncEnabled != nil {
		updates["sync_enabled"] = *req.SyncEnabled
	}
	if req.SyncFrequency != nil {
		updates["sync_frequency"] = *req.SyncFrequency
	}
	if req.CountryCode != nil {
		updates["country_code"] = *req.CountryCode
	}

	// Update via service
	config, err := h.service.UpdateSyncConfig(r.Context(), tenantID, configID, updates)
	if err != nil {
		h.logger.WithError(err).Error("Failed to update sync config")
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to update sync config: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(config)
}

// DeleteSyncConfig handles DELETE /api/v1/external-sync/{id}
// Deletes a sync configuration
func (h *ExternalSyncHandler) DeleteSyncConfig(w http.ResponseWriter, r *http.Request) {
	configIDStr := r.PathValue("id")
	configID, err := uuid.Parse(configIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid config_id format")
		return
	}

	// Get tenant from header
	tenantIDStr := r.Header.Get("X-Hasura-Tenant-Id")
	if tenantIDStr == "" {
		writeJSONError(w, http.StatusBadRequest, "tenant_id required (X-Hasura-Tenant-Id header)")
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid tenant_id format")
		return
	}

	// Delete via service
	err = h.service.DeleteSyncConfig(r.Context(), tenantID, configID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to delete sync config")
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete sync config: %v", err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// TriggerSync handles POST /api/v1/external-sync/{id}/trigger
// Manually triggers a sync for a configuration
func (h *ExternalSyncHandler) TriggerSync(w http.ResponseWriter, r *http.Request) {
	configIDStr := r.PathValue("id")
	configID, err := uuid.Parse(configIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid config_id format")
		return
	}

	// Get tenant from header
	tenantIDStr := r.Header.Get("X-Hasura-Tenant-Id")
	if tenantIDStr == "" {
		writeJSONError(w, http.StatusBadRequest, "tenant_id required (X-Hasura-Tenant-Id header)")
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid tenant_id format")
		return
	}

	// Trigger sync via service
	syncLog, err := h.service.TriggerSync(r.Context(), tenantID, configID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to trigger sync")
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to trigger sync: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(syncLog)
}

// GetSyncLogs handles GET /api/v1/external-sync/{id}/logs
// Retrieves sync execution logs
func (h *ExternalSyncHandler) GetSyncLogs(w http.ResponseWriter, r *http.Request) {
	configIDStr := r.PathValue("id")
	configID, err := uuid.Parse(configIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid config_id format")
		return
	}

	// Get pagination parameters
	limit := 50
	offset := 0

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsedLimit, err := strconv.Atoi(l); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	if o := r.URL.Query().Get("offset"); o != "" {
		if parsedOffset, err := strconv.Atoi(o); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	// Get tenant from header
	tenantIDStr := r.Header.Get("X-Hasura-Tenant-Id")
	if tenantIDStr == "" {
		writeJSONError(w, http.StatusBadRequest, "tenant_id required (X-Hasura-Tenant-Id header)")
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid tenant_id format")
		return
	}

	// Get logs via service
	logs, total, err := h.service.GetSyncLogs(r.Context(), tenantID, configID, limit, offset)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get sync logs")
		writeJSONError(w, http.StatusInternalServerError, "Failed to get sync logs")
		return
	}

	// Response with pagination metadata
	response := map[string]interface{}{
		"data":   logs,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetLastSyncLog handles GET /api/v1/external-sync/{id}/last-log
// Retrieves the most recent sync log
func (h *ExternalSyncHandler) GetLastSyncLog(w http.ResponseWriter, r *http.Request) {
	configIDStr := r.PathValue("id")
	configID, err := uuid.Parse(configIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid config_id format")
		return
	}

	// Get tenant from header
	tenantIDStr := r.Header.Get("X-Hasura-Tenant-Id")
	if tenantIDStr == "" {
		writeJSONError(w, http.StatusBadRequest, "tenant_id required (X-Hasura-Tenant-Id header)")
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid tenant_id format")
		return
	}

	// Get last log via service
	log, err := h.service.GetLastSyncLog(r.Context(), tenantID, configID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get last sync log")
		writeJSONError(w, http.StatusInternalServerError, "Failed to get last sync log")
		return
	}

	if log == nil {
		writeJSONError(w, http.StatusNotFound, "no sync logs found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(log)
}

// ValidateProviderRequest defines request for validating provider credentials
type ValidateProviderRequest struct {
	Provider    string `json:"provider"`
	CountryCode string `json:"country_code"`
	APIKey      string `json:"api_key,omitempty"`
}

// ValidateProvider handles POST /api/v1/external-sync/validate-provider
// Validates provider credentials
func (h *ExternalSyncHandler) ValidateProvider(w http.ResponseWriter, r *http.Request) {
	var req ValidateProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.WithError(err).Warn("Invalid request body")
		writeJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get tenant from header (just for logging/context)
	tenantIDStr := r.Header.Get("X-Hasura-Tenant-Id")
	if tenantIDStr == "" {
		writeJSONError(w, http.StatusBadRequest, "tenant_id required (X-Hasura-Tenant-Id header)")
		return
	}

	// Validate credentials
	valid, err := h.service.ValidateProviderCredentials(r.Context(), services.ExternalSyncProvider(req.Provider), req.CountryCode, req.APIKey)
	if err != nil {
		h.logger.WithError(err).Warn("Provider validation error")
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("Provider validation error: %v", err))
		return
	}

	response := map[string]bool{"valid": valid}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
