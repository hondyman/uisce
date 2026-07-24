package models

import (
	"time"

	"github.com/google/uuid"
)

// InternalEvent represents the system's native event format
// This corresponds to the 'internal_events' table
type InternalEvent struct {
	ID             uuid.UUID `json:"id" db:"id"`
	TenantID       uuid.UUID `json:"tenant_id" db:"tenant_id"`
	UserID         uuid.UUID `json:"user_id" db:"user_id"`
	Title          string    `json:"title" db:"title"`
	Description    *string   `json:"description,omitempty" db:"description"`
	Location       *string   `json:"location,omitempty" db:"location"`
	StartTime      time.Time `json:"start_time" db:"start_time"`
	EndTime        time.Time `json:"end_time" db:"end_time"`
	IsAllDay       bool      `json:"is_all_day" db:"is_all_day"`
	IsRecurring    bool      `json:"is_recurring" db:"is_recurring"`
	RecurrenceRule *string   `json:"recurrence_rule,omitempty" db:"recurrence_rule"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}
