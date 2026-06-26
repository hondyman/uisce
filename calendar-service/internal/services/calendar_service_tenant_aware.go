package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// ============================================================================
// Calendar Types
// ============================================================================

// Calendar represents a calendar with holidays/blackouts
type Calendar struct {
	ID          string          `json:"id"`
	TenantID    string          `json:"tenant_id"`
	LogicalID   string          `json:"logical_id"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Region      string          `json:"region"`
	ValidFrom   time.Time       `json:"valid_from"`
	ValidTo     *time.Time      `json:"valid_to"`
	Holidays    json.RawMessage `json:"holidays"`
	CreatedAt   time.Time       `json:"created_at"`
	CreatedBy   string          `json:"created_by"`
	UpdatedAt   time.Time       `json:"updated_at"`
	UpdatedBy   string          `json:"updated_by"`
	Tags        json.RawMessage `json:"tags"`
	Metadata    json.RawMessage `json:"metadata"`
}

// ============================================================================
// Tenant-Aware Service Interfaces
// ============================================================================

// CalendarServiceTenantAware defines tenant-scoped calendar operations
type CalendarServiceTenantAware interface {
	// Create creates a calendar for a specific tenant
	Create(ctx context.Context, tenantID, userID, name, description, timezone string) (*Calendar, error)

	// GetByID retrieves a calendar, verifying tenant access
	GetByID(ctx context.Context, tenantID, calendarID string) (*Calendar, error)

	// ListByTenant lists all calendars for a tenant
	ListByTenant(ctx context.Context, tenantID string, limit, offset int) ([]Calendar, error)

	// Update updates a calendar with tenant verification
	Update(ctx context.Context, tenantID, calendarID, userID string, updates map[string]interface{}) (*Calendar, error)

	// Delete soft-deletes a calendar with tenant verification
	Delete(ctx context.Context, tenantID, calendarID, userID string) error

	// validateTenantAccess verifies resource belongs to tenant
	validateTenantAccess(ctx context.Context, tenantID, calendarID string) error
}

// AvailabilityServiceTenantAware defines tenant-scoped availability operations
type AvailabilityServiceTenantAware interface {
	// CheckAvailability checks if a time slot is available for a calendar
	CheckAvailability(ctx context.Context, tenantID, calendarID string, startTime time.Time, durationSecs int) (bool, error)

	// GetMetrics retrieves availability metrics for a calendar
	GetMetrics(ctx context.Context, tenantID, calendarID string, period string) (map[string]interface{}, error)
}

// BlackoutServiceTenantAware defines tenant-scoped blackout operations
type BlackoutServiceTenantAware interface {
	// CreateBlackout creates a blackout window for a calendar
	CreateBlackout(ctx context.Context, tenantID, calendarID, userID, name string, startTime, endTime time.Time) (string, error)

	// GetBlackouts lists blackouts for a calendar
	GetBlackouts(ctx context.Context, tenantID, calendarID string) ([]map[string]interface{}, error)

	// DeleteBlackout deletes a blackout window
	DeleteBlackout(ctx context.Context, tenantID, calendarID, blackoutID, userID string) error
}

// ============================================================================
// Tenant Context Helper
// ============================================================================

// TenantContext wraps tenant-scoped request context
type TenantContext struct {
	TenantID string
	UserID   string
	Roles    []string
	Email    string
}

// ValidateTenant ensures tenantID is valid
func (tc *TenantContext) ValidateTenant() error {
	if tc.TenantID == "" {
		return errors.New("tenant_id is required")
	}
	return nil
}

// ValidateUser ensures userID is present
func (tc *TenantContext) ValidateUser() error {
	if tc.UserID == "" {
		return errors.New("user_id is required")
	}
	return nil
}

// ============================================================================
// Enhanced CalendarService (Tenant-Aware)
// ============================================================================

// CalendarServiceImpl implements CalendarServiceTenantAware with tenant context
type CalendarServiceImpl struct {
	repo   CalendarRepository
	logger *logrus.Entry
}

// NewCalendarServiceImpl creates a new tenant-aware calendar service
func NewCalendarServiceImpl(repo CalendarRepository, logger *logrus.Entry) *CalendarServiceImpl {
	if logger == nil {
		logger = logrus.NewEntry(logrus.New())
	}
	return &CalendarServiceImpl{
		repo:   repo,
		logger: logger.WithField("service", "calendar_tenant_aware"),
	}
}

// Create creates a calendar with full tenant verification
func (s *CalendarServiceImpl) Create(ctx context.Context, tenantID, userID, name, description, timezone string) (*Calendar, error) {
	// Validate inputs
	if tenantID == "" {
		s.logger.Warn("Create calendar failed: tenant_id required")
		return nil, errors.New("tenant_id is required")
	}
	if userID == "" {
		s.logger.Warn("Create calendar failed: user_id required")
		return nil, errors.New("user_id is required")
	}
	if name == "" {
		return nil, errors.New("name is required")
	}

	// Create calendar object
	calendar := &Calendar{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		Name:        name,
		Description: description,
		Region:      timezone,
		CreatedAt:   time.Now().UTC(),
		CreatedBy:   userID,
		UpdatedAt:   time.Now().UTC(),
		UpdatedBy:   userID,
	}

	// Persist to repository
	if err := s.repo.Create(ctx, calendar); err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"tenant_id": tenantID,
			"name":      name,
		}).Error("Failed to create calendar")
		return nil, fmt.Errorf("create calendar: %w", err)
	}

	// Audit log
	s.logger.WithFields(logrus.Fields{
		"tenant_id":   tenantID,
		"user_id":     userID,
		"calendar_id": calendar.ID,
		"action":      "create_calendar",
	}).Info("Calendar created")

	return calendar, nil
}

// GetByID retrieves a calendar with tenant verification
func (s *CalendarServiceImpl) GetByID(ctx context.Context, tenantID, calendarID string) (*Calendar, error) {
	if tenantID == "" || calendarID == "" {
		return nil, errors.New("tenant_id and calendar_id are required")
	}

	// Verify tenant access
	if err := s.validateTenantAccess(ctx, tenantID, calendarID); err != nil {
		return nil, err
	}

	// Fetch from repository (already scoped to tenant)
	calendar, err := s.repo.GetByID(ctx, tenantID, calendarID)
	if err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"tenant_id":   tenantID,
			"calendar_id": calendarID,
		}).Warn("Calendar not found or access denied")
		// Don't leak whether resource exists
		return nil, errors.New("calendar not found")
	}

	return calendar, nil
}

// ListByTenant lists all calendars for a tenant
func (s *CalendarServiceImpl) ListByTenant(ctx context.Context, tenantID string, limit, offset int) ([]Calendar, error) {
	if tenantID == "" {
		return nil, errors.New("tenant_id is required")
	}

	// Fetch from repository with tenant filter
	calendars, err := s.repo.ListByTenant(ctx, tenantID, limit, offset)
	if err != nil {
		s.logger.WithError(err).WithField("tenant_id", tenantID).Error("Failed to list calendars")
		return nil, fmt.Errorf("list calendars: %w", err)
	}

	// Audit logging
	s.logger.WithFields(logrus.Fields{
		"tenant_id": tenantID,
		"count":     len(calendars),
		"action":    "list_calendars",
	}).Debug("Calendars listed")

	return calendars, nil
}

// Update updates a calendar with tenant verification
func (s *CalendarServiceImpl) Update(ctx context.Context, tenantID, calendarID, userID string, updates map[string]interface{}) (*Calendar, error) {
	if tenantID == "" || calendarID == "" || userID == "" {
		return nil, errors.New("tenant_id, calendar_id, and user_id are required")
	}

	// Verify tenant access before update
	if err := s.validateTenantAccess(ctx, tenantID, calendarID); err != nil {
		return nil, err
	}

	// Add metadata to update
	updates["updated_by"] = userID
	updates["updated_at"] = time.Now().UTC()

	// Update in repository
	calendar, err := s.repo.Update(ctx, tenantID, calendarID, updates)
	if err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"tenant_id":   tenantID,
			"calendar_id": calendarID,
		}).Error("Failed to update calendar")
		return nil, fmt.Errorf("update calendar: %w", err)
	}

	// Audit log
	s.logger.WithFields(logrus.Fields{
		"tenant_id":   tenantID,
		"user_id":     userID,
		"calendar_id": calendarID,
		"action":      "update_calendar",
	}).Info("Calendar updated")

	return calendar, nil
}

// Delete soft-deletes a calendar with tenant verification
func (s *CalendarServiceImpl) Delete(ctx context.Context, tenantID, calendarID, userID string) error {
	if tenantID == "" || calendarID == "" || userID == "" {
		return errors.New("tenant_id, calendar_id, and user_id are required")
	}

	// Verify tenant access before delete
	if err := s.validateTenantAccess(ctx, tenantID, calendarID); err != nil {
		return err
	}

	// Delete from repository
	if err := s.repo.Delete(ctx, tenantID, calendarID); err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"tenant_id":   tenantID,
			"calendar_id": calendarID,
		}).Error("Failed to delete calendar")
		return fmt.Errorf("delete calendar: %w", err)
	}

	// Audit log
	s.logger.WithFields(logrus.Fields{
		"tenant_id":   tenantID,
		"user_id":     userID,
		"calendar_id": calendarID,
		"action":      "delete_calendar",
	}).Info("Calendar deleted")

	return nil
}

// validateTenantAccess verifies a resource belongs to a tenant
func (s *CalendarServiceImpl) validateTenantAccess(ctx context.Context, tenantID, calendarID string) error {
	calendar, err := s.repo.GetByID(ctx, tenantID, calendarID)
	if err != nil {
		// Don't leak whether resource exists
		return errors.New("access denied")
	}

	// Verify tenant match
	if calendar.TenantID != tenantID {
		s.logger.WithFields(logrus.Fields{
			"tenant_id":     tenantID,
			"calendar_id":   calendarID,
			"actual_tenant": calendar.TenantID,
		}).Warn("Cross-tenant access attempt blocked")
		return errors.New("access denied")
	}

	return nil
}

// ============================================================================
// Calendar Repository Interface (Tenant-Aware)
// ============================================================================

// CalendarRepository defines tenant-scoped calendar persistence
type CalendarRepository interface {
	// Create persists a calendar
	Create(ctx context.Context, calendar *Calendar) error

	// GetByID retrieves a calendar by ID
	GetByID(ctx context.Context, tenantID, calendarID string) (*Calendar, error)

	// ListByTenant lists all calendars for a tenant
	ListByTenant(ctx context.Context, tenantID string, limit, offset int) ([]Calendar, error)

	// Update modifies a calendar
	Update(ctx context.Context, tenantID, calendarID string, updates map[string]interface{}) (*Calendar, error)

	// Delete soft-deletes a calendar
	Delete(ctx context.Context, tenantID, calendarID string) error
}

// eof
