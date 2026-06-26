package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// ExternalSyncProvider represents a holiday provider type
type ExternalSyncProvider string

const (
	ProviderNagerDate    ExternalSyncProvider = "nager_date"
	ProviderCalendarific ExternalSyncProvider = "calendarific"
)

// SyncFrequency represents how often to sync
type SyncFrequency string

const (
	FrequencyWeekly  SyncFrequency = "weekly"
	FrequencyMonthly SyncFrequency = "monthly"
	FrequencyYearly  SyncFrequency = "yearly"
)

// ExternalSyncConfig represents configuration for external holiday sync
type ExternalSyncConfig struct {
	ID              uuid.UUID            `json:"id"`
	TenantID        uuid.UUID            `json:"tenant_id"`
	ProfileID       uuid.UUID            `json:"profile_id"`
	Provider        ExternalSyncProvider `json:"provider"`
	CountryCode     string               `json:"country_code"`
	APIKeyEncrypted string               `json:"-"` // Never expose in JSON
	SyncEnabled     bool                 `json:"sync_enabled"`
	SyncFrequency   SyncFrequency        `json:"sync_frequency"`
	LastSyncAt      *time.Time           `json:"last_sync_at"`
	NextSyncAt      *time.Time           `json:"next_sync_at"`
	CreatedAt       time.Time            `json:"created_at"`
	UpdatedAt       time.Time            `json:"updated_at"`
}

// SyncLog represents a record of a sync execution
type SyncLog struct {
	ID              uuid.UUID `json:"id"`
	ConfigID        uuid.UUID `json:"config_id"`
	Status          string    `json:"status"` // success, failed, partial
	HolidaysAdded   int       `json:"holidays_added"`
	HolidaysUpdated int       `json:"holidays_updated"`
	ErrorMessage    *string   `json:"error_message"`
	ExecutionTimeMS int       `json:"execution_time_ms"`
	ExecutedAt      time.Time `json:"executed_at"`
}

// HolidayData represents holiday information from external providers
type HolidayData struct {
	Date        string `json:"date"`
	LocalName   string `json:"local_name"`
	Name        string `json:"name"`
	CountryCode string `json:"country_code"`
	Fixed       bool   `json:"fixed"`
	Type        string `json:"type"`
}

// ExternalSyncServiceTenantAware defines operations for managing external syncs
type ExternalSyncServiceTenantAware interface {
	// Config management
	CreateSyncConfig(ctx context.Context, tenantID uuid.UUID, config *ExternalSyncConfig) (*ExternalSyncConfig, error)
	GetSyncConfig(ctx context.Context, tenantID uuid.UUID, configID uuid.UUID) (*ExternalSyncConfig, error)
	ListSyncConfigs(ctx context.Context, tenantID uuid.UUID) ([]ExternalSyncConfig, error)
	ListSyncConfigsByProfile(ctx context.Context, tenantID uuid.UUID, profileID uuid.UUID) ([]ExternalSyncConfig, error)
	UpdateSyncConfig(ctx context.Context, tenantID uuid.UUID, configID uuid.UUID, updates map[string]interface{}) (*ExternalSyncConfig, error)
	DeleteSyncConfig(ctx context.Context, tenantID uuid.UUID, configID uuid.UUID) error

	// Sync operations
	TriggerSync(ctx context.Context, tenantID uuid.UUID, configID uuid.UUID) (*SyncLog, error)
	GetSyncLogs(ctx context.Context, tenantID uuid.UUID, configID uuid.UUID, limit int, offset int) ([]SyncLog, int, error)
	GetLastSyncLog(ctx context.Context, tenantID uuid.UUID, configID uuid.UUID) (*SyncLog, error)

	// Provider operations
	ValidateProviderCredentials(ctx context.Context, provider ExternalSyncProvider, countryCode string, apiKey string) (bool, error)
	FetchHolidays(ctx context.Context, provider ExternalSyncProvider, countryCode string, year int, apiKey string) ([]HolidayData, error)
}

// ExternalSyncService implements ExternalSyncServiceTenantAware
type ExternalSyncService struct {
	repo       *RepositoryAdapter
	auditSvc   AuditService
	logger     *logrus.Entry
	httpClient *http.Client
	mu         sync.RWMutex
}

// NewExternalSyncService creates a new external sync service
func NewExternalSyncService(repo *RepositoryAdapter, auditSvc AuditService, logger *logrus.Entry) *ExternalSyncService {
	return &ExternalSyncService{
		repo:     repo,
		auditSvc: auditSvc,
		logger:   logger,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CreateSyncConfig creates a new external sync configuration
func (s *ExternalSyncService) CreateSyncConfig(ctx context.Context, tenantID uuid.UUID, config *ExternalSyncConfig) (*ExternalSyncConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate input
	if config.ProfileID == uuid.Nil {
		return nil, fmt.Errorf("profile_id is required")
	}
	if config.Provider != ProviderNagerDate && config.Provider != ProviderCalendarific {
		return nil, fmt.Errorf("invalid provider: %s", config.Provider)
	}
	if len(config.CountryCode) == 0 {
		return nil, fmt.Errorf("country_code is required")
	}

	// Generate ID and timestamps
	config.ID = uuid.New()
	config.TenantID = tenantID
	now := time.Now().UTC()
	config.CreatedAt = now
	config.UpdatedAt = now

	// Calculate next sync time based on frequency
	config.NextSyncAt = s.calculateNextSyncTime(now, config.SyncFrequency)

	// Save to repository
	if err := s.repo.SaveExternalSyncConfig(ctx, config); err != nil {
		s.logger.WithError(err).Error("failed to save external sync config")
		return nil, err
	}

	// Audit log
	s.auditSvc.RecordCreate(ctx, tenantID.String(), "sync_config", config.ID.String(), nil, "")

	s.logger.WithFields(logrus.Fields{
		"tenant_id":  tenantID,
		"config_id":  config.ID,
		"provider":   config.Provider,
		"profile_id": config.ProfileID,
	}).Info("external sync config created")

	return config, nil
}

// GetSyncConfig retrieves a sync configuration with tenant verification
func (s *ExternalSyncService) GetSyncConfig(ctx context.Context, tenantID uuid.UUID, configID uuid.UUID) (*ExternalSyncConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	config, err := s.repo.GetExternalSyncConfig(ctx, configID.String())
	if err != nil {
		return nil, err
	}

	// Verify tenant ownership
	if config.TenantID != tenantID {
		s.logger.WithFields(logrus.Fields{
			"tenant_id":       tenantID,
			"config_id":       configID,
			"expected_tenant": config.TenantID,
		}).Warn("cross-tenant sync config access denied")
		return nil, fmt.Errorf("sync config not found")
	}

	return config, nil
}

// ListSyncConfigs lists all sync configurations for a tenant
func (s *ExternalSyncService) ListSyncConfigs(ctx context.Context, tenantID uuid.UUID) ([]ExternalSyncConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	configs, err := s.repo.ListExternalSyncConfigs(ctx, tenantID.String())
	if err != nil {
		s.logger.WithError(err).Error("failed to list sync configs")
		return nil, err
	}

	return configs, nil
}

// ListSyncConfigsByProfile lists sync configurations for a specific profile
func (s *ExternalSyncService) ListSyncConfigsByProfile(ctx context.Context, tenantID uuid.UUID, profileID uuid.UUID) ([]ExternalSyncConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	configs, err := s.repo.ListExternalSyncConfigsByProfile(ctx, profileID.String())
	if err != nil {
		s.logger.WithError(err).Error("failed to list sync configs by profile")
		return nil, err
	}

	// Filter by tenant (security)
	var tenantConfigs []ExternalSyncConfig
	for _, config := range configs {
		if config.TenantID == tenantID {
			tenantConfigs = append(tenantConfigs, config)
		}
	}

	return tenantConfigs, nil
}

// UpdateSyncConfig updates an existing sync configuration
func (s *ExternalSyncService) UpdateSyncConfig(ctx context.Context, tenantID uuid.UUID, configID uuid.UUID, updates map[string]interface{}) (*ExternalSyncConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get existing config
	config, err := s.repo.GetExternalSyncConfig(ctx, configID.String())
	if err != nil {
		return nil, err
	}

	// Verify tenant ownership
	if config.TenantID != tenantID {
		return nil, fmt.Errorf("sync config not found")
	}

	// Store old values for audit (simplified for now)

	// Apply updates
	if syncEnabled, ok := updates["sync_enabled"].(bool); ok {
		config.SyncEnabled = syncEnabled
	}
	if frequency, ok := updates["sync_frequency"].(string); ok {
		config.SyncFrequency = SyncFrequency(frequency)
	}
	if countryCode, ok := updates["country_code"].(string); ok {
		config.CountryCode = countryCode
	}

	config.UpdatedAt = time.Now().UTC()

	// Update next sync time if frequency changed
	if _, freqChanged := updates["sync_frequency"]; freqChanged {
		config.NextSyncAt = s.calculateNextSyncTime(time.Now().UTC(), config.SyncFrequency)
	}

	// Save updates
	if err := s.repo.SaveExternalSyncConfig(ctx, config); err != nil {
		s.logger.WithError(err).Error("failed to update sync config")
		return nil, err
	}

	// Audit log
	s.auditSvc.RecordUpdate(ctx, tenantID.String(), "sync_config", config.ID.String(), nil, nil, "")

	s.logger.WithFields(logrus.Fields{
		"tenant_id": tenantID,
		"config_id": configID,
	}).Info("external sync config updated")

	return config, nil
}

// DeleteSyncConfig deletes a sync configuration
func (s *ExternalSyncService) DeleteSyncConfig(ctx context.Context, tenantID uuid.UUID, configID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get existing config
	config, err := s.repo.GetExternalSyncConfig(ctx, configID.String())
	if err != nil {
		return err
	}

	// Verify tenant ownership
	if config.TenantID != tenantID {
		return fmt.Errorf("sync config not found")
	}

	// Delete
	if err := s.repo.DeleteExternalSyncConfig(ctx, configID.String()); err != nil {
		s.logger.WithError(err).Error("failed to delete sync config")
		return err
	}

	// Audit log
	s.auditSvc.RecordDelete(ctx, tenantID.String(), "sync_config", config.ID.String(), nil, "")

	s.logger.WithFields(logrus.Fields{
		"tenant_id": tenantID,
		"config_id": configID,
	}).Info("external sync config deleted")

	return nil
}

// TriggerSync executes a sync operation immediately
func (s *ExternalSyncService) TriggerSync(ctx context.Context, tenantID uuid.UUID, configID uuid.UUID) (*SyncLog, error) {
	s.mu.Lock()
	config, err := s.repo.GetExternalSyncConfig(ctx, configID.String())
	s.mu.Unlock()

	if err != nil {
		return nil, err
	}

	if config.TenantID != tenantID {
		return nil, fmt.Errorf("sync config not found")
	}

	startTime := time.Now()

	// Fetch holidays from provider
	currentYear := time.Now().Year()
	holidays, err := s.FetchHolidays(ctx, config.Provider, config.CountryCode, currentYear, config.APIKeyEncrypted)

	executionTime := time.Since(startTime).Milliseconds()
	status := "success"
	var errorMsg *string
	added := len(holidays)
	updated := 0

	if err != nil {
		status = "failed"
		errStr := err.Error()
		errorMsg = &errStr
		added = 0
	}

	// Create sync log
	syncLog := &SyncLog{
		ID:              uuid.New(),
		ConfigID:        configID,
		Status:          status,
		HolidaysAdded:   added,
		HolidaysUpdated: updated,
		ErrorMessage:    errorMsg,
		ExecutionTimeMS: int(executionTime),
		ExecutedAt:      time.Now().UTC(),
	}

	// Save sync log
	if err := s.repo.SaveSyncLog(ctx, syncLog); err != nil {
		s.logger.WithError(err).Error("failed to save sync log")
		// Don't fail the entire operation if logging fails
	}

	// Update sync config timestamps
	s.mu.Lock()
	now := time.Now().UTC()
	config.LastSyncAt = &now
	config.NextSyncAt = s.calculateNextSyncTime(now, config.SyncFrequency)
	if err := s.repo.SaveExternalSyncConfig(ctx, config); err != nil {
		s.logger.WithError(err).Error("failed to update sync config timestamps")
	}
	s.mu.Unlock()

	s.logger.WithFields(logrus.Fields{
		"config_id":      configID,
		"status":         status,
		"execution_time": executionTime,
		"holidays_added": added,
	}).Info("sync completed")

	return syncLog, nil
}

// GetSyncLogs retrieves sync logs for a configuration with pagination
func (s *ExternalSyncService) GetSyncLogs(ctx context.Context, tenantID uuid.UUID, configID uuid.UUID, limit int, offset int) ([]SyncLog, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Verify config ownership first
	config, err := s.repo.GetExternalSyncConfig(ctx, configID.String())
	if err != nil {
		return nil, 0, err
	}

	if config.TenantID != tenantID {
		return nil, 0, fmt.Errorf("sync config not found")
	}

	logs, total, err := s.repo.GetSyncLogs(ctx, tenantID.String(), configID.String(), limit, offset)
	if err != nil {
		s.logger.WithError(err).Error("failed to get sync logs")
		return nil, 0, err
	}

	return logs, total, nil
}

// GetLastSyncLog retrieves the most recent sync log for a configuration
func (s *ExternalSyncService) GetLastSyncLog(ctx context.Context, tenantID uuid.UUID, configID uuid.UUID) (*SyncLog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Verify config ownership first
	config, err := s.repo.GetExternalSyncConfig(ctx, configID.String())
	if err != nil {
		return nil, err
	}

	if config.TenantID != tenantID {
		return nil, fmt.Errorf("sync config not found")
	}

	log, err := s.repo.GetLastSyncLog(ctx, configID.String())
	if err != nil {
		s.logger.WithError(err).Error("failed to get last sync log")
		return nil, err
	}

	return log, nil
}

// ValidateProviderCredentials checks if provider credentials are valid
func (s *ExternalSyncService) ValidateProviderCredentials(ctx context.Context, provider ExternalSyncProvider, countryCode string, apiKey string) (bool, error) {
	switch provider {
	case ProviderNagerDate:
		return s.validateNagerDateCredentials(ctx, countryCode)
	case ProviderCalendarific:
		return s.validateCalendarificCredentials(ctx, countryCode, apiKey)
	default:
		return false, fmt.Errorf("unknown provider: %s", provider)
	}
}

// FetchHolidays retrieves holidays from the specified provider
func (s *ExternalSyncService) FetchHolidays(ctx context.Context, provider ExternalSyncProvider, countryCode string, year int, apiKey string) ([]HolidayData, error) {
	switch provider {
	case ProviderNagerDate:
		return s.fetchHolidaysFromNagerDate(ctx, countryCode, year)
	case ProviderCalendarific:
		return s.fetchHolidaysFromCalendarific(ctx, countryCode, year, apiKey)
	default:
		return nil, fmt.Errorf("unknown provider: %s", provider)
	}
}

// validateNagerDateCredentials validates Nager.Date API access
func (s *ExternalSyncService) validateNagerDateCredentials(ctx context.Context, countryCode string) (bool, error) {
	url := fmt.Sprintf("https://api.nager.date/v3/available-countries")
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

// validateCalendarificCredentials validates Calendarific API access
func (s *ExternalSyncService) validateCalendarificCredentials(ctx context.Context, countryCode string, apiKey string) (bool, error) {
	url := fmt.Sprintf("https://calendarific.com/api/v2/holidays?api_key=%s&country=%s&year=%d", apiKey, countryCode, time.Now().Year())
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

// fetchHolidaysFromNagerDate fetches holidays from Nager.Date API
func (s *ExternalSyncService) fetchHolidaysFromNagerDate(ctx context.Context, countryCode string, year int) ([]HolidayData, error) {
	url := fmt.Sprintf("https://api.nager.date/v3/PublicHolidays/%d/%s", year, countryCode)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("nager.date API error: %s", string(body))
	}

	var holidays []HolidayData
	if err := json.NewDecoder(resp.Body).Decode(&holidays); err != nil {
		return nil, err
	}

	return holidays, nil
}

// fetchHolidaysFromCalendarific fetches holidays from Calendarific API
func (s *ExternalSyncService) fetchHolidaysFromCalendarific(ctx context.Context, countryCode string, year int, apiKey string) ([]HolidayData, error) {
	url := fmt.Sprintf("https://calendarific.com/api/v2/holidays?api_key=%s&country=%s&year=%d", apiKey, countryCode, year)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("calendarific API error: %s", string(body))
	}

	var response struct {
		Response struct {
			Holidays []HolidayData `json:"holidays"`
		} `json:"response"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return response.Response.Holidays, nil
}

// calculateNextSyncTime calculates the next sync time based on frequency
func (s *ExternalSyncService) calculateNextSyncTime(from time.Time, frequency SyncFrequency) *time.Time {
	var next time.Time

	switch frequency {
	case FrequencyWeekly:
		next = from.AddDate(0, 0, 7)
	case FrequencyMonthly:
		next = from.AddDate(0, 1, 0)
	case FrequencyYearly:
		next = from.AddDate(1, 0, 0)
	default:
		next = from.AddDate(0, 1, 0) // Default to monthly
	}

	return &next
}
