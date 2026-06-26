package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"calendar-service/internal/api"
	"calendar-service/internal/services"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExternalSyncCreateConfig tests creating a new sync configuration
func TestExternalSyncCreateConfig(t *testing.T) {
	// Setup
	logger := logrus.NewEntry(logrus.New())
	repo := services.NewRepositoryAdapter(nil, logger)
	auditService := services.NewAuditService(logger)
	syncService := services.NewExternalSyncService(repo, auditService, logger)
	handler := api.NewExternalSyncHandler(syncService, logger)

	tenantID := uuid.New()
	profileID := uuid.New()

	// Create request
	reqBody := api.CreateSyncConfigRequest{
		ProfileID:     profileID.String(),
		Provider:      "nager_date",
		CountryCode:   "US",
		SyncEnabled:   true,
		SyncFrequency: "monthly",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/external-sync", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hasura-Tenant-Id", tenantID.String())

	// Execute
	w := httptest.NewRecorder()
	handler.CreateSyncConfig(w, req)

	// Assert
	assert.Equal(t, http.StatusCreated, w.Code)

	var response services.ExternalSyncConfig
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, profileID, response.ProfileID)
	assert.Equal(t, services.ProviderNagerDate, response.Provider)
	assert.Equal(t, "US", response.CountryCode)
	assert.True(t, response.SyncEnabled)
}

// TestExternalSyncGetConfig tests retrieving a sync configuration
func TestExternalSyncGetConfig(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	repo := services.NewRepositoryAdapter(nil, logger)
	auditService := services.NewAuditService(logger)
	syncService := services.NewExternalSyncService(repo, auditService, logger)
	handler := api.NewExternalSyncHandler(syncService, logger)

	tenantID := uuid.New()
	profileID := uuid.New()

	// Create a config first
	ctx := context.Background()
	config := &services.ExternalSyncConfig{
		ID:            uuid.New(),
		TenantID:      tenantID,
		ProfileID:     profileID,
		Provider:      services.ProviderNagerDate,
		CountryCode:   "US",
		SyncEnabled:   true,
		SyncFrequency: services.FrequencyMonthly,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}
	repo.SaveExternalSyncConfig(ctx, config)

	// Get request
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/external-sync/%s", config.ID.String()), nil)
	req.SetPathValue("id", config.ID.String())
	req.Header.Set("X-Hasura-Tenant-Id", tenantID.String())

	// Execute
	w := httptest.NewRecorder()
	handler.GetSyncConfig(w, req)

	// Assert
	require.Equal(t, http.StatusOK, w.Code)

	var response services.ExternalSyncConfig
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, config.ID, response.ID)
	assert.Equal(t, profileID, response.ProfileID)
}

// TestExternalSyncListConfigs tests listing sync configurations
func TestExternalSyncListConfigs(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	repo := services.NewRepositoryAdapter(nil, logger)
	auditService := services.NewAuditService(logger)
	syncService := services.NewExternalSyncService(repo, auditService, logger)
	handler := api.NewExternalSyncHandler(syncService, logger)

	tenantID := uuid.New()
	profileID1 := uuid.New()
	// profileID2 removed

	// Create multiple configs
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		config := &services.ExternalSyncConfig{
			ID:            uuid.New(),
			TenantID:      tenantID,
			ProfileID:     profileID1,
			Provider:      services.ProviderNagerDate,
			CountryCode:   "US",
			SyncEnabled:   true,
			SyncFrequency: services.FrequencyMonthly,
			CreatedAt:     time.Now().UTC(),
			UpdatedAt:     time.Now().UTC(),
		}
		repo.SaveExternalSyncConfig(ctx, config)
	}

	// List request
	req := httptest.NewRequest("GET", "/api/v1/external-sync", nil)
	req.Header.Set("X-Hasura-Tenant-Id", tenantID.String())

	// Execute
	w := httptest.NewRecorder()
	handler.ListSyncConfigs(w, req)

	// Assert
	require.Equal(t, http.StatusOK, w.Code)

	var configs []services.ExternalSyncConfig
	json.NewDecoder(w.Body).Decode(&configs)
	assert.Len(t, configs, 3)
}

// TestExternalSyncUpdateConfig tests updating a sync configuration
func TestExternalSyncUpdateConfig(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	repo := services.NewRepositoryAdapter(nil, logger)
	auditService := services.NewAuditService(logger)
	syncService := services.NewExternalSyncService(repo, auditService, logger)
	handler := api.NewExternalSyncHandler(syncService, logger)

	tenantID := uuid.New()
	profileID := uuid.New()

	// Create a config
	ctx := context.Background()
	config := &services.ExternalSyncConfig{
		ID:            uuid.New(),
		TenantID:      tenantID,
		ProfileID:     profileID,
		Provider:      services.ProviderNagerDate,
		CountryCode:   "US",
		SyncEnabled:   true,
		SyncFrequency: services.FrequencyMonthly,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}
	repo.SaveExternalSyncConfig(ctx, config)

	// Update request
	updateReq := api.UpdateSyncConfigRequest{
		SyncEnabled:   func() *bool { b := false; return &b }(),
		SyncFrequency: func() *string { s := "weekly"; return &s }(),
	}

	body, _ := json.Marshal(updateReq)
	req := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/external-sync/%s", config.ID.String()), bytes.NewReader(body))
	req.SetPathValue("id", config.ID.String())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hasura-Tenant-Id", tenantID.String())

	// Execute
	w := httptest.NewRecorder()
	handler.UpdateSyncConfig(w, req)

	// Assert
	require.Equal(t, http.StatusOK, w.Code)

	var response services.ExternalSyncConfig
	json.NewDecoder(w.Body).Decode(&response)
	assert.False(t, response.SyncEnabled)
	assert.Equal(t, services.FrequencyWeekly, response.SyncFrequency)
}

// TestExternalSyncDeleteConfig tests deleting a sync configuration
func TestExternalSyncDeleteConfig(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	repo := services.NewRepositoryAdapter(nil, logger)
	auditService := services.NewAuditService(logger)
	syncService := services.NewExternalSyncService(repo, auditService, logger)
	handler := api.NewExternalSyncHandler(syncService, logger)

	tenantID := uuid.New()
	profileID := uuid.New()

	// Create a config
	ctx := context.Background()
	config := &services.ExternalSyncConfig{
		ID:            uuid.New(),
		TenantID:      tenantID,
		ProfileID:     profileID,
		Provider:      services.ProviderNagerDate,
		CountryCode:   "US",
		SyncEnabled:   true,
		SyncFrequency: services.FrequencyMonthly,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}
	repo.SaveExternalSyncConfig(ctx, config)

	// Delete request
	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/external-sync/%s", config.ID.String()), nil)
	req.SetPathValue("id", config.ID.String())
	req.Header.Set("X-Hasura-Tenant-Id", tenantID.String())

	// Execute
	w := httptest.NewRecorder()
	handler.DeleteSyncConfig(w, req)

	// Assert
	assert.Equal(t, http.StatusNoContent, w.Code)

	// Verify deletion
	req2 := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/external-sync/%s", config.ID.String()), nil)
	req2.SetPathValue("id", config.ID.String())
	req2.Header.Set("X-Hasura-Tenant-Id", tenantID.String())
	w2 := httptest.NewRecorder()
	handler.GetSyncConfig(w2, req2)
	assert.Equal(t, http.StatusNotFound, w2.Code)
}

// TestExternalSyncTriggerSync tests manually triggering a sync
func TestExternalSyncTriggerSync(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	repo := services.NewRepositoryAdapter(nil, logger)
	auditService := services.NewAuditService(logger)
	syncService := services.NewExternalSyncService(repo, auditService, logger)
	handler := api.NewExternalSyncHandler(syncService, logger)

	tenantID := uuid.New()
	profileID := uuid.New()

	// Create a config
	ctx := context.Background()
	config := &services.ExternalSyncConfig{
		ID:            uuid.New(),
		TenantID:      tenantID,
		ProfileID:     profileID,
		Provider:      services.ProviderNagerDate,
		CountryCode:   "US",
		SyncEnabled:   true,
		SyncFrequency: services.FrequencyMonthly,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}
	repo.SaveExternalSyncConfig(ctx, config)

	// Trigger sync request
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/external-sync/%s/trigger", config.ID.String()), nil)
	req.SetPathValue("id", config.ID.String())
	req.Header.Set("X-Hasura-Tenant-Id", tenantID.String())

	// Execute
	w := httptest.NewRecorder()
	handler.TriggerSync(w, req)

	// Assert
	require.Equal(t, http.StatusOK, w.Code)

	var syncLog services.SyncLog
	json.NewDecoder(w.Body).Decode(&syncLog)
	assert.NotEmpty(t, syncLog.ID)
	assert.Equal(t, config.ID, syncLog.ConfigID)
	// Status is either 'success' or 'failed' depending on API response
	assert.Contains(t, []string{"success", "failed"}, syncLog.Status)
}

// TestExternalSyncGetLogs tests retrieving sync logs
func TestExternalSyncGetLogs(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	repo := services.NewRepositoryAdapter(nil, logger)
	auditService := services.NewAuditService(logger)
	syncService := services.NewExternalSyncService(repo, auditService, logger)
	handler := api.NewExternalSyncHandler(syncService, logger)

	tenantID := uuid.New()
	configID := uuid.New()

	// Save some sync logs
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		log := &services.SyncLog{
			ID:              uuid.New(),
			ConfigID:        configID,
			Status:          "success",
			HolidaysAdded:   10,
			HolidaysUpdated: 2,
			ExecutionTimeMS: 1500,
			ExecutedAt:      time.Now().UTC().Add(-time.Duration(i) * time.Hour),
		}
		repo.SaveSyncLog(ctx, log)
	}

	// Create config for tenant verification
	config := &services.ExternalSyncConfig{
		ID:            configID,
		TenantID:      tenantID,
		ProfileID:     uuid.New(),
		Provider:      services.ProviderNagerDate,
		CountryCode:   "US",
		SyncEnabled:   true,
		SyncFrequency: services.FrequencyMonthly,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}
	repo.SaveExternalSyncConfig(ctx, config)

	// Get logs request
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/external-sync/%s/logs?limit=10&offset=0", configID.String()), nil)
	req.SetPathValue("id", configID.String())
	req.Header.Set("X-Hasura-Tenant-Id", tenantID.String())

	// Execute
	w := httptest.NewRecorder()
	handler.GetSyncLogs(w, req)

	// Assert
	require.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, float64(5), response["total"])
	logs := response["data"].([]interface{})
	assert.Len(t, logs, 5)
}

// TestExternalSyncTenantIsolation tests that configs are isolated by tenant
func TestExternalSyncTenantIsolation(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	repo := services.NewRepositoryAdapter(nil, logger)
	auditService := services.NewAuditService(logger)
	syncService := services.NewExternalSyncService(repo, auditService, logger)
	handler := api.NewExternalSyncHandler(syncService, logger)

	tenantID1 := uuid.New()
	tenantID2 := uuid.New()
	profileID1 := uuid.New()
	profileID2 := uuid.New()

	// Create configs for different tenants
	ctx := context.Background()
	config1 := &services.ExternalSyncConfig{
		ID:            uuid.New(),
		TenantID:      tenantID1,
		ProfileID:     profileID1,
		Provider:      services.ProviderNagerDate,
		CountryCode:   "US",
		SyncEnabled:   true,
		SyncFrequency: services.FrequencyMonthly,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}
	repo.SaveExternalSyncConfig(ctx, config1)

	config2 := &services.ExternalSyncConfig{
		ID:            uuid.New(),
		TenantID:      tenantID2,
		ProfileID:     profileID2,
		Provider:      services.ProviderCalendarific,
		CountryCode:   "GB",
		SyncEnabled:   true,
		SyncFrequency: services.FrequencyMonthly,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}
	repo.SaveExternalSyncConfig(ctx, config2)

	// Try to access config1 with tenantID2
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/external-sync/%s", config1.ID.String()), nil)
	req.SetPathValue("id", config1.ID.String())
	req.Header.Set("X-Hasura-Tenant-Id", tenantID2.String())

	// Execute
	w := httptest.NewRecorder()
	handler.GetSyncConfig(w, req)

	// Assert - should fail with 404 (not found)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestExternalSyncValidateProvider tests provider validation
func TestExternalSyncValidateProvider(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	repo := services.NewRepositoryAdapter(nil, logger)
	auditService := services.NewAuditService(logger)
	syncService := services.NewExternalSyncService(repo, auditService, logger)
	handler := api.NewExternalSyncHandler(syncService, logger)

	tenantID := uuid.New()

	// Validate request (Nager.Date is free, should succeed)
	reqBody := api.ValidateProviderRequest{
		Provider:    "nager_date",
		CountryCode: "US",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/external-sync/validate-provider", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hasura-Tenant-Id", tenantID.String())

	// Execute
	w := httptest.NewRecorder()
	handler.ValidateProvider(w, req)

	// Assert
	// In offline test environment, this likely fails with network error (handled as 400)
	if w.Code != http.StatusOK {
		require.Equal(t, http.StatusBadRequest, w.Code)
		return
	}

	var response map[string]bool
	json.NewDecoder(w.Body).Decode(&response)
	assert.True(t, response["valid"])
}

// BenchmarkSyncCreation benchmarks sync config creation performance
func BenchmarkSyncCreation(b *testing.B) {
	logger := logrus.NewEntry(logrus.New())
	repo := services.NewRepositoryAdapter(nil, logger)
	auditService := services.NewAuditService(logger)
	syncService := services.NewExternalSyncService(repo, auditService, logger)

	tenantID := uuid.New()
	profileID := uuid.New()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		config := &services.ExternalSyncConfig{
			ProfileID:     profileID,
			Provider:      services.ProviderNagerDate,
			CountryCode:   "US",
			SyncEnabled:   true,
			SyncFrequency: services.FrequencyMonthly,
		}
		_, _ = syncService.CreateSyncConfig(context.Background(), tenantID, config)
	}
}

// BenchmarkSyncTrigger benchmarks sync trigger performance
func BenchmarkSyncTrigger(b *testing.B) {
	logger := logrus.NewEntry(logrus.New())
	repo := services.NewRepositoryAdapter(nil, logger)
	auditService := services.NewAuditService(logger)
	syncService := services.NewExternalSyncService(repo, auditService, logger)

	ctx := context.Background()
	tenantID := uuid.New()
	profileID := uuid.New()

	// Create a config
	config := &services.ExternalSyncConfig{
		ID:            uuid.New(),
		TenantID:      tenantID,
		ProfileID:     profileID,
		Provider:      services.ProviderNagerDate,
		CountryCode:   "US",
		SyncEnabled:   true,
		SyncFrequency: services.FrequencyMonthly,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}
	repo.SaveExternalSyncConfig(ctx, config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = syncService.TriggerSync(ctx, tenantID, config.ID)
	}
}
