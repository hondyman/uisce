package services

import (
	"context"
	"fmt"
	"time"

	"calendar-service/internal/repository"

	"github.com/sirupsen/logrus"
)

// ServiceRepository defines the interface for data persistence used by services
type ServiceRepository interface {
	// Calendar operations
	Create(ctx context.Context, calendar *Calendar) error
	GetByID(ctx context.Context, tenantID, calendarID string) (*Calendar, error)
	ListByTenant(ctx context.Context, tenantID string, limit, offset int) ([]Calendar, error)
	Update(ctx context.Context, tenantID, calendarID string, updates map[string]interface{}) (*Calendar, error)
	Delete(ctx context.Context, tenantID, calendarID string) error

	// Profile operations
	SaveProfile(ctx context.Context, profile *ScheduleProfile) error
	GetProfile(ctx context.Context, profileID string) (*ScheduleProfile, error)
	ListProfilesByTenant(ctx context.Context, tenantID string, onlyActive bool) ([]ScheduleProfile, error)
	ListProfilesByID(ctx context.Context, logicalID string) ([]ScheduleProfile, error)

	// External Sync Config operations
	SaveExternalSyncConfig(ctx context.Context, config *ExternalSyncConfig) error
	GetExternalSyncConfig(ctx context.Context, configID string) (*ExternalSyncConfig, error)
	ListExternalSyncConfigs(ctx context.Context, tenantID string) ([]ExternalSyncConfig, error)
	ListExternalSyncConfigsByProfile(ctx context.Context, profileID string) ([]ExternalSyncConfig, error)
	DeleteExternalSyncConfig(ctx context.Context, configID string) error

	// Sync Log operations
	SaveSyncLog(ctx context.Context, log *SyncLog) error
	GetSyncLogs(ctx context.Context, tenantID, configID string, limit, offset int) ([]SyncLog, int, error)
	GetLastSyncLog(ctx context.Context, configID string) (*SyncLog, error)

	// Recurrence Rule operations
	StoreRecurrenceRule(ctx context.Context, rule interface{}) error
	GetRecurrenceRule(ctx context.Context, id, tenantID string) (*RecurrenceRule, error)
	ListRecurrenceRules(ctx context.Context, profileID, tenantID string, limit, offset int) ([]*RecurrenceRule, int64, error)
	UpdateRecurrenceRule(ctx context.Context, rule interface{}) error
	DeleteRecurrenceRule(ctx context.Context, id, tenantID string) error

	// Recurrence Exception operations
	StoreRecurrenceException(ctx context.Context, exception *RecurrenceException) error
	GetExceptions(ctx context.Context, recurrenceID, tenantID string) ([]*RecurrenceException, error)
	DeleteRecurrenceException(ctx context.Context, exceptionID string) error

	// Blackout Period operations
	StoreBlackoutPeriod(ctx context.Context, period interface{}) error
	GetBlackoutPeriods(ctx context.Context, profileID, tenantID string, from, to interface{}) ([]*BlackoutPeriod, error)
	DeleteBlackoutPeriod(ctx context.Context, id, tenantID string) error

	// Calendar Event operations
	ListCalendars(ctx context.Context, profileID, tenantID string, limit, offset int) ([]Calendar, int, error)
	GetCalendarEvents(ctx context.Context, calendarID, tenantID string) ([]CalendarEvent, error)
}

// RepositoryAdapter adapts repository.TenantCalendarRepository to services.CalendarRepository interface
type RepositoryAdapter struct {
	repo   repository.TenantCalendarRepository
	logger *logrus.Entry
}

// NewRepositoryAdapter creates a new adapter wrapper
func NewRepositoryAdapter(repo repository.TenantCalendarRepository, logger *logrus.Entry) *RepositoryAdapter {
	return &RepositoryAdapter{
		repo:   repo,
		logger: logger,
	}
}

// Create adapts repository Create (takes tenantID, calendar) to service repo interface
func (a *RepositoryAdapter) Create(ctx context.Context, calendar *Calendar) error {
	// Convert services.Calendar to repository.TenantCalendar
	tenantCalendar := &repository.TenantCalendar{
		ID:          calendar.ID,
		TenantID:    calendar.TenantID,
		Name:        calendar.Name,
		Description: calendar.Description,
		Timezone:    calendar.Region, // Map Region to Timezone
		CreatedAt:   calendar.CreatedAt,
		CreatedBy:   calendar.CreatedBy,
		UpdatedAt:   calendar.UpdatedAt,
		UpdatedBy:   calendar.UpdatedBy,
	}

	// Call repository with tenantID
	return a.repo.Create(ctx, calendar.TenantID, tenantCalendar)
}

// GetByID adapts repository GetByID
func (a *RepositoryAdapter) GetByID(ctx context.Context, tenantID, calendarID string) (*Calendar, error) {
	tenantCalendar, err := a.repo.GetByID(ctx, tenantID, calendarID)
	if err != nil {
		return nil, err
	}

	// Convert repository.TenantCalendar to services.Calendar
	return &Calendar{
		ID:          tenantCalendar.ID,
		TenantID:    tenantCalendar.TenantID,
		Name:        tenantCalendar.Name,
		Description: tenantCalendar.Description,
		Region:      tenantCalendar.Timezone, // Map Timezone to Region
		CreatedAt:   tenantCalendar.CreatedAt,
		CreatedBy:   tenantCalendar.CreatedBy,
		UpdatedAt:   tenantCalendar.UpdatedAt,
		UpdatedBy:   tenantCalendar.UpdatedBy,
	}, nil
}

// ListByTenant adapts repository ListByTenant
func (a *RepositoryAdapter) ListByTenant(ctx context.Context, tenantID string, limit, offset int) ([]Calendar, error) {
	tenantCalendars, err := a.repo.ListByTenant(ctx, tenantID, limit, offset)
	if err != nil {
		return nil, err
	}

	// Convert []repository.TenantCalendar to []services.Calendar
	calendars := make([]Calendar, len(tenantCalendars))
	for i, tc := range tenantCalendars {
		calendars[i] = Calendar{
			ID:          tc.ID,
			TenantID:    tc.TenantID,
			Name:        tc.Name,
			Description: tc.Description,
			Region:      tc.Timezone, // Map Timezone to Region
			CreatedAt:   tc.CreatedAt,
			CreatedBy:   tc.CreatedBy,
			UpdatedAt:   tc.UpdatedAt,
			UpdatedBy:   tc.UpdatedBy,
		}
	}

	return calendars, nil
}

// Update adapts repository Update
func (a *RepositoryAdapter) Update(ctx context.Context, tenantID, calendarID string, updates map[string]interface{}) (*Calendar, error) {
	tenantCalendar, err := a.repo.Update(ctx, tenantID, calendarID, updates)
	if err != nil {
		return nil, err
	}

	// Convert repository.TenantCalendar to services.Calendar
	return &Calendar{
		ID:          tenantCalendar.ID,
		TenantID:    tenantCalendar.TenantID,
		Name:        tenantCalendar.Name,
		Description: tenantCalendar.Description,
		Region:      tenantCalendar.Timezone, // Map Timezone to Region
		CreatedAt:   tenantCalendar.CreatedAt,
		CreatedBy:   tenantCalendar.CreatedBy,
		UpdatedAt:   tenantCalendar.UpdatedAt,
		UpdatedBy:   tenantCalendar.UpdatedBy,
	}, nil
}

// Delete adapts repository Delete
func (a *RepositoryAdapter) Delete(ctx context.Context, tenantID, calendarID string) error {
	return a.repo.Delete(ctx, tenantID, calendarID)
}

// Profile-related methods (Phase 4.3)
var profileStore = make(map[string]*ScheduleProfile)       // In-memory store for profiles
var profilesByTenant = make(map[string][]*ScheduleProfile) // Index by tenant
var profilesByID = make(map[string][]*ScheduleProfile)     // Index by logical ID (all versions)

// SaveProfile saves a profile (new version or updated)
func (a *RepositoryAdapter) SaveProfile(ctx context.Context, profile *ScheduleProfile) error {
	// Store in in-memory maps
	profileStore[profile.ID] = profile

	// Index by tenant
	if _, exists := profilesByTenant[profile.TenantID]; !exists {
		profilesByTenant[profile.TenantID] = make([]*ScheduleProfile, 0)
	}
	profilesByTenant[profile.TenantID] = append(profilesByTenant[profile.TenantID], profile)

	// Index by profile name (for version tracking)
	profilesByID[profile.ProfileName] = append(profilesByID[profile.ProfileName], profile)

	a.logger.WithField("profile_id", profile.ID).Debug("Profile saved")
	return nil
}

// GetProfile retrieves a profile by ID
func (a *RepositoryAdapter) GetProfile(ctx context.Context, profileID string) (*ScheduleProfile, error) {
	profile := profileStore[profileID]
	return profile, nil
}

// ListProfilesByTenant lists profiles for a tenant
func (a *RepositoryAdapter) ListProfilesByTenant(ctx context.Context, tenantID string, onlyActive bool) ([]ScheduleProfile, error) {
	profiles := make([]ScheduleProfile, 0)

	for _, p := range profilesByTenant[tenantID] {
		if onlyActive && (!p.Active || p.ValidTo != nil) {
			continue
		}
		profiles = append(profiles, *p)
	}

	return profiles, nil
}

// ListProfilesByID lists all versions of a profile
func (a *RepositoryAdapter) ListProfilesByID(ctx context.Context, logicalID string) ([]ScheduleProfile, error) {
	profiles := make([]ScheduleProfile, 0)

	for _, p := range profilesByID[logicalID] {
		profiles = append(profiles, *p)
	}

	return profiles, nil
}

// External sync config methods (Phase 4.5)
var syncConfigStore = make(map[string]*ExternalSyncConfig)        // In-memory store for sync configs
var syncConfigsByTenant = make(map[string][]*ExternalSyncConfig)  // Index by tenant
var syncConfigsByProfile = make(map[string][]*ExternalSyncConfig) // Index by profile

// SaveExternalSyncConfig saves or updates a sync configuration
func (a *RepositoryAdapter) SaveExternalSyncConfig(ctx context.Context, config *ExternalSyncConfig) error {
	configID := config.ID.String()
	syncConfigStore[configID] = config

	// Index by tenant
	tenantID := config.TenantID.String()
	if _, exists := syncConfigsByTenant[tenantID]; !exists {
		syncConfigsByTenant[tenantID] = make([]*ExternalSyncConfig, 0)
	}

	// Update or append
	found := false
	for i, existing := range syncConfigsByTenant[tenantID] {
		if existing.ID == config.ID {
			syncConfigsByTenant[tenantID][i] = config
			found = true
			break
		}
	}
	if !found {
		syncConfigsByTenant[tenantID] = append(syncConfigsByTenant[tenantID], config)
	}

	// Index by profile
	profileID := config.ProfileID.String()
	if _, exists := syncConfigsByProfile[profileID]; !exists {
		syncConfigsByProfile[profileID] = make([]*ExternalSyncConfig, 0)
	}

	found = false
	for i, existing := range syncConfigsByProfile[profileID] {
		if existing.ID == config.ID {
			syncConfigsByProfile[profileID][i] = config
			found = true
			break
		}
	}
	if !found {
		syncConfigsByProfile[profileID] = append(syncConfigsByProfile[profileID], config)
	}

	a.logger.WithField("config_id", configID).Debug("Sync config saved")
	return nil
}

// GetExternalSyncConfig retrieves a sync configuration by ID
func (a *RepositoryAdapter) GetExternalSyncConfig(ctx context.Context, configID string) (*ExternalSyncConfig, error) {
	if config, exists := syncConfigStore[configID]; exists {
		return config, nil
	}
	return nil, fmt.Errorf("sync config not found")
}

// ListExternalSyncConfigs lists all sync configs for a tenant
func (a *RepositoryAdapter) ListExternalSyncConfigs(ctx context.Context, tenantID string) ([]ExternalSyncConfig, error) {
	configs := make([]ExternalSyncConfig, 0)

	if tenantConfigs, exists := syncConfigsByTenant[tenantID]; exists {
		for _, c := range tenantConfigs {
			configs = append(configs, *c)
		}
	}

	return configs, nil
}

// ListExternalSyncConfigsByProfile lists sync configs for a specific profile
func (a *RepositoryAdapter) ListExternalSyncConfigsByProfile(ctx context.Context, profileID string) ([]ExternalSyncConfig, error) {
	configs := make([]ExternalSyncConfig, 0)

	if profileConfigs, exists := syncConfigsByProfile[profileID]; exists {
		for _, c := range profileConfigs {
			configs = append(configs, *c)
		}
	}

	return configs, nil
}

// DeleteExternalSyncConfig deletes a sync configuration
func (a *RepositoryAdapter) DeleteExternalSyncConfig(ctx context.Context, configID string) error {
	delete(syncConfigStore, configID)
	a.logger.WithField("config_id", configID).Debug("Sync config deleted")
	return nil
}

// Sync logs methods
var syncLogStore = make(map[string]*SyncLog)       // In-memory store for sync logs
var syncLogsByConfig = make(map[string][]*SyncLog) // Index by config

// SaveSyncLog saves a sync execution log
func (a *RepositoryAdapter) SaveSyncLog(ctx context.Context, log *SyncLog) error {
	logID := log.ID.String()
	syncLogStore[logID] = log

	// Index by config
	configID := log.ConfigID.String()
	if _, exists := syncLogsByConfig[configID]; !exists {
		syncLogsByConfig[configID] = make([]*SyncLog, 0)
	}
	syncLogsByConfig[configID] = append(syncLogsByConfig[configID], log)

	a.logger.WithField("log_id", logID).Debug("Sync log saved")
	return nil
}

// GetSyncLogs retrieves sync logs for a config with pagination
func (a *RepositoryAdapter) GetSyncLogs(ctx context.Context, tenantID, configID string, limit, offset int) ([]SyncLog, int, error) {
	logs := make([]SyncLog, 0)

	if configLogs, exists := syncLogsByConfig[configID]; exists {
		total := len(configLogs)

		// Apply pagination
		start := offset
		end := offset + limit
		if start >= total {
			return logs, total, nil
		}
		if end > total {
			end = total
		}

		for i := start; i < end; i++ {
			logs = append(logs, *configLogs[i])
		}

		return logs, total, nil
	}

	return logs, 0, nil
}

// GetLastSyncLog retrieves the most recent sync log for a config
func (a *RepositoryAdapter) GetLastSyncLog(ctx context.Context, configID string) (*SyncLog, error) {
	if configLogs, exists := syncLogsByConfig[configID]; exists && len(configLogs) > 0 {
		// Return the last one (most recent)
		return configLogs[len(configLogs)-1], nil
	}
	return nil, nil
}

// Calendar event and conflict detection methods (Phase 5)

// CalendarEvent represents an event on a calendar
type CalendarEvent struct {
	ID         string
	StartTime  time.Time
	EndTime    time.Time
	TimezoneID string
}

// ListCalendars lists all calendars for a profile
func (a *RepositoryAdapter) ListCalendars(ctx context.Context, profileID, tenantID string, limit, offset int) ([]Calendar, int, error) {
	// Return calendars linked to this profile
	calendars, err := a.ListByTenant(ctx, tenantID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	return calendars, len(calendars), nil
}

// GetCalendarEvents retrieves events for a calendar
func (a *RepositoryAdapter) GetCalendarEvents(ctx context.Context, calendarID, tenantID string) ([]CalendarEvent, error) {
	// Return empty events list (placeholder for external calendar integration)
	return []CalendarEvent{}, nil
}

// ListRecurrenceRules retrieves recurring blackout rules for a profile
func (a *RepositoryAdapter) ListRecurrenceRules(ctx context.Context, profileID, tenantID string, limit, offset int) ([]*RecurrenceRule, int64, error) {
	// Return empty rules list (placeholder for recurrence logic)
	return []*RecurrenceRule{}, 0, nil
}

// GetBlackoutPeriods retrieves blackout periods for a time range
func (a *RepositoryAdapter) GetBlackoutPeriods(ctx context.Context, profileID, tenantID string, from, to interface{}) ([]*BlackoutPeriod, error) {
	// Return empty blackouts list (placeholder for blackout retrieval)
	return []*BlackoutPeriod{}, nil
}

// StoreBlackoutPeriod stores a new blackout period
func (a *RepositoryAdapter) StoreBlackoutPeriod(ctx context.Context, period interface{}) error {
	// Placeholder: store blackout logic would go here
	return nil
}

// DeleteBlackoutPeriod deletes a blackout period
func (a *RepositoryAdapter) DeleteBlackoutPeriod(ctx context.Context, id, tenantID string) error {
	// Placeholder: delete blackout logic would go here
	return nil
}

// Recurrence rule methods

// StoreRecurrenceRule stores a recurrence rule
func (a *RepositoryAdapter) StoreRecurrenceRule(ctx context.Context, rule interface{}) error {
	return nil
}

// GetRecurrenceRule retrieves a recurrence rule
func (a *RepositoryAdapter) GetRecurrenceRule(ctx context.Context, id, tenantID string) (*RecurrenceRule, error) {
	// Return stub rule
	return &RecurrenceRule{
		ID:       id,
		TenantID: tenantID,
	}, nil
}

// UpdateRecurrenceRule updates a recurrence rule
func (a *RepositoryAdapter) UpdateRecurrenceRule(ctx context.Context, rule interface{}) error {
	return nil
}

// DeleteRecurrenceRule deletes a recurrence rule
func (a *RepositoryAdapter) DeleteRecurrenceRule(ctx context.Context, id, tenantID string) error {
	return nil
}

// GetExceptions retrieves exceptions for a recurrence rule
func (a *RepositoryAdapter) GetExceptions(ctx context.Context, recurrenceID, tenantID string) ([]*RecurrenceException, error) {
	return []*RecurrenceException{}, nil
}

// StoreRecurrenceException stores an exception
func (a *RepositoryAdapter) StoreRecurrenceException(ctx context.Context, exception *RecurrenceException) error {
	return nil
}

// DeleteRecurrenceException deletes an exception
func (a *RepositoryAdapter) DeleteRecurrenceException(ctx context.Context, exceptionID string) error {
	return nil
}
