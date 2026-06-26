package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Tenant represents a tenant in the system
type Tenant struct {
	ID        uuid.UUID              `json:"id"`
	Name      string                 `json:"name"`
	Region    string                 `json:"region"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// CalendarProfile represents a named collection of holidays and blackout rules
type CalendarProfile struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Timezone    string    `json:"timezone"`
	Region      string    `json:"region"`
	IsActive    bool      `json:"is_active"`
	Version     string    `json:"version"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Holiday represents a specific date when business is closed
type Holiday struct {
	ID          uuid.UUID `json:"id"`
	ProfileID   uuid.UUID `json:"profile_id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	HolidayDate time.Time `json:"holiday_date"`
	Name        string    `json:"name"`
	Region      string    `json:"region,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// BlackoutWindow represents a time range when business is closed
type BlackoutWindow struct {
	ID              uuid.UUID  `json:"id"`
	ProfileID       uuid.UUID  `json:"profile_id"`
	TenantID        uuid.UUID  `json:"tenant_id"`
	StartTime       time.Time  `json:"start_time"`
	EndTime         time.Time  `json:"end_time"`
	Title           string     `json:"title"`
	Reason          string     `json:"reason,omitempty"`
	RRULE           string     `json:"rrule,omitempty"`
	IsRecurring     bool       `json:"is_recurring"`
	RecurrenceStart *time.Time `json:"recurrence_start,omitempty"`
	RecurrenceEnd   *time.Time `json:"recurrence_end,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// TenantRepository handles tenant data access
type TenantRepository interface {
	Create(ctx context.Context, tenant *Tenant) error
	GetByID(ctx context.Context, id uuid.UUID) (*Tenant, error)
	GetByName(ctx context.Context, name string) (*Tenant, error)
	List(ctx context.Context, limit, offset int) ([]*Tenant, error)
	Update(ctx context.Context, tenant *Tenant) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// CalendarProfileRepository handles calendar profile data access
type CalendarProfileRepository interface {
	Create(ctx context.Context, profile *CalendarProfile) error
	GetByID(ctx context.Context, id uuid.UUID) (*CalendarProfile, error)
	GetByName(ctx context.Context, tenantID uuid.UUID, name string) (*CalendarProfile, error)
	ListByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*CalendarProfile, error)
	Update(ctx context.Context, profile *CalendarProfile) error
	Delete(ctx context.Context, id uuid.UUID) error
	InvalidateCache(ctx context.Context, id uuid.UUID) error
}

// HolidayRepository handles holiday data access
type HolidayRepository interface {
	Create(ctx context.Context, holiday *Holiday) error
	GetByID(ctx context.Context, id uuid.UUID) (*Holiday, error)
	ListByProfile(ctx context.Context, profileID uuid.UUID) ([]*Holiday, error)
	ListByDateRange(ctx context.Context, profileID uuid.UUID, start, end time.Time) ([]*Holiday, error)
	Update(ctx context.Context, holiday *Holiday) error
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteByProfile(ctx context.Context, profileID uuid.UUID) error
	Upsert(ctx context.Context, holiday *Holiday) error
}

// BlackoutWindowRepository handles blackout window data access
type BlackoutWindowRepository interface {
	Create(ctx context.Context, blackout *BlackoutWindow) error
	GetByID(ctx context.Context, id uuid.UUID) (*BlackoutWindow, error)
	ListByProfile(ctx context.Context, profileID uuid.UUID) ([]*BlackoutWindow, error)
	ListByDateRange(ctx context.Context, profileID uuid.UUID, start, end time.Time) ([]*BlackoutWindow, error)
	Update(ctx context.Context, blackout *BlackoutWindow) error
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteByProfile(ctx context.Context, profileID uuid.UUID) error
}

// ResolvedCalendarMetadata tracks resolved calendar cache validity
type ResolvedCalendarMetadata struct {
	ID             uuid.UUID  `json:"id"`
	TenantID       uuid.UUID  `json:"tenant_id"`
	ProfileID      uuid.UUID  `json:"profile_id"`
	Region         string     `json:"region"`
	ResolvedAt     *time.Time `json:"resolved_at,omitempty"`
	Version        string     `json:"version,omitempty"`
	HolidaysCount  int        `json:"holidays_count"`
	BlackoutsCount int        `json:"blackouts_count"`
	ContentHash    string     `json:"content_hash,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// MetadataRepository handles resolved calendar metadata
type MetadataRepository interface {
	Upsert(ctx context.Context, metadata *ResolvedCalendarMetadata) error
	GetByProfile(ctx context.Context, tenantID, profileID uuid.UUID, region string) (*ResolvedCalendarMetadata, error)
	InvalidateByProfile(ctx context.Context, profileID uuid.UUID) error
}

// AuditLog represents a database change audit entry
type AuditLog struct {
	ID          uuid.UUID              `json:"id"`
	TenantID    uuid.UUID              `json:"tenant_id"`
	EntityType  string                 `json:"entity_type"`
	EntityID    *uuid.UUID             `json:"entity_id,omitempty"`
	Action      string                 `json:"action"`
	Changes     map[string]interface{} `json:"changes,omitempty"`
	PerformedBy string                 `json:"performed_by,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
}

// AuditLogRepository handles audit log storage
type AuditLogRepository interface {
	Log(ctx context.Context, log *AuditLog) error
	GetByEntity(ctx context.Context, entityType string, entityID uuid.UUID, limit int) ([]*AuditLog, error)
	GetByTenant(ctx context.Context, tenantID uuid.UUID, limit int) ([]*AuditLog, error)
}
