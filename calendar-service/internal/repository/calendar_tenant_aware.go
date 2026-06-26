package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/sirupsen/logrus"
)

// ============================================================================
// Calendar Repository Types
// ============================================================================

// TenantCalendar represents a calendar with full tenant context
type TenantCalendar struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Timezone    string    `json:"timezone"`
	CreatedAt   time.Time `json:"created_at"`
	CreatedBy   string    `json:"created_by"`
	UpdatedAt   time.Time `json:"updated_at"`
	UpdatedBy   string    `json:"updated_by"`
}

// ============================================================================
// Calendar Repository Interface
// ============================================================================

// TenantCalendarRepository defines tenant-scoped calendar operations
type TenantCalendarRepository interface {
	// Create inserts a calendar with tenant verification
	Create(ctx context.Context, tenantID string, calendar *TenantCalendar) error

	// GetByID retrieves a calendar by ID with tenant filter
	GetByID(ctx context.Context, tenantID, calendarID string) (*TenantCalendar, error)

	// ListByTenant lists all calendars for a tenant
	ListByTenant(ctx context.Context, tenantID string, limit, offset int) ([]TenantCalendar, error)

	// Update modifies a calendar with tenant verification
	Update(ctx context.Context, tenantID, calendarID string, updates map[string]interface{}) (*TenantCalendar, error)

	// Delete soft-deletes a calendar with tenant verification
	Delete(ctx context.Context, tenantID, calendarID string) error

	// CountByTenant counts calendars for a tenant
	CountByTenant(ctx context.Context, tenantID string) (int, error)

	// ExistsByID checks if a calendar exists for a tenant
	ExistsByID(ctx context.Context, tenantID, calendarID string) (bool, error)
}

// ============================================================================
// In-Memory Calendar Repository (for testing)
// ============================================================================

// InMemoryCalendarRepository implements TenantCalendarRepository in memory
type InMemoryCalendarRepository struct {
	calendars map[string]map[string]*TenantCalendar // [tenantID][calendarID]calendar
	logger    *logrus.Entry
}

// NewInMemoryCalendarRepository creates a new in-memory repository
func NewInMemoryCalendarRepository(logger *logrus.Entry) *InMemoryCalendarRepository {
	return &InMemoryCalendarRepository{
		calendars: make(map[string]map[string]*TenantCalendar),
		logger:    logger,
	}
}

// Create inserts a calendar
func (r *InMemoryCalendarRepository) Create(ctx context.Context, tenantID string, calendar *TenantCalendar) error {
	if tenantID != calendar.TenantID {
		return errors.New("tenant mismatch")
	}

	if _, exists := r.calendars[tenantID]; !exists {
		r.calendars[tenantID] = make(map[string]*TenantCalendar)
	}

	r.calendars[tenantID][calendar.ID] = calendar
	r.logger.WithFields(logrus.Fields{
		"tenant_id":   tenantID,
		"calendar_id": calendar.ID,
	}).Debug("Calendar created in memory")

	return nil
}

// GetByID retrieves a calendar by ID with tenant filter
func (r *InMemoryCalendarRepository) GetByID(ctx context.Context, tenantID, calendarID string) (*TenantCalendar, error) {
	tenantCals, exists := r.calendars[tenantID]
	if !exists {
		return nil, sql.ErrNoRows
	}

	cal, exists := tenantCals[calendarID]
	if !exists {
		return nil, sql.ErrNoRows
	}

	return cal, nil
}

// ListByTenant lists all calendars for a tenant
func (r *InMemoryCalendarRepository) ListByTenant(ctx context.Context, tenantID string, limit, offset int) ([]TenantCalendar, error) {
	tenantCals, exists := r.calendars[tenantID]
	if !exists {
		return []TenantCalendar{}, nil
	}

	var result []TenantCalendar
	i := 0
	for _, cal := range tenantCals {
		if i >= offset && len(result) < limit {
			result = append(result, *cal)
		}
		i++
	}

	return result, nil
}

// Update modifies a calendar with tenant verification
func (r *InMemoryCalendarRepository) Update(ctx context.Context, tenantID, calendarID string, updates map[string]interface{}) (*TenantCalendar, error) {
	cal, err := r.GetByID(ctx, tenantID, calendarID)
	if err != nil {
		return nil, err
	}

	if name, ok := updates["name"]; ok {
		cal.Name = name.(string)
	}
	if desc, ok := updates["description"]; ok {
		cal.Description = desc.(string)
	}
	if tz, ok := updates["timezone"]; ok {
		cal.Timezone = tz.(string)
	}
	if updated, ok := updates["updated_by"]; ok {
		cal.UpdatedBy = updated.(string)
	}
	if updatedAt, ok := updates["updated_at"]; ok {
		cal.UpdatedAt = updatedAt.(time.Time)
	}

	return cal, nil
}

// Delete soft-deletes a calendar with tenant verification
func (r *InMemoryCalendarRepository) Delete(ctx context.Context, tenantID, calendarID string) error {
	_, err := r.GetByID(ctx, tenantID, calendarID)
	if err != nil {
		return err
	}

	delete(r.calendars[tenantID], calendarID)
	r.logger.WithFields(logrus.Fields{
		"tenant_id":   tenantID,
		"calendar_id": calendarID,
	}).Debug("Calendar deleted from memory")

	return nil
}

// CountByTenant counts calendars for a tenant
func (r *InMemoryCalendarRepository) CountByTenant(ctx context.Context, tenantID string) (int, error) {
	tenantCals, exists := r.calendars[tenantID]
	if !exists {
		return 0, nil
	}
	return len(tenantCals), nil
}

// ExistsByID checks if a calendar exists for a tenant
func (r *InMemoryCalendarRepository) ExistsByID(ctx context.Context, tenantID, calendarID string) (bool, error) {
	_, err := r.GetByID(ctx, tenantID, calendarID)
	return err == nil, nil
}

// ============================================================================
// PostgreSQL Calendar Repository (for database)
// ============================================================================

// NOTE: PostgresCalendarRepository implementation moved to postgres_calendar_repository.go
// This file contains the interface and in-memory test implementation only.

// eof
