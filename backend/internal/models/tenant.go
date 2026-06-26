package models

import (
	"time"

	"github.com/google/uuid"
)

// Tenant represents a first-class tenant in the system
type Tenant struct {
	ID            uuid.UUID `db:"id" json:"id"`
	Name          string    `db:"name" json:"name"`
	Code          *string   `db:"code" json:"code"`
	Region        *string   `db:"region" json:"region"`
	Plan          string    `db:"plan" json:"plan"`
	MaxRequests   *int64    `db:"max_requests" json:"max_requests"`
	WindowSeconds *int      `db:"window_seconds" json:"window_seconds"`
	IsSuspended   bool      `db:"is_suspended" json:"is_suspended"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`
}

// TenantCreateRequest represents a request to create a new tenant
type TenantCreateRequest struct {
	ID            uuid.UUID
	Name          string
	Code          *string
	Region        *string
	Plan          string
	MaxRequests   *int64
	WindowSeconds *int
}

// TenantUpdateRequest represents a request to update a tenant
type TenantUpdateRequest struct {
	Name          *string
	Region        *string
	Plan          *string
	MaxRequests   *int64
	WindowSeconds *int
	IsSuspended   *bool
}

// ValidateTenantPlan validates the plan value
func ValidateTenantPlan(plan string) bool {
	switch plan {
	case "free", "pro", "enterprise":
		return true
	default:
		return false
	}
}
