package models

import (
	"time"

	"github.com/google/uuid"
)

// MicroBundle represents a minimal claim set for a single task, project, or asset group.
type MicroBundle struct {
	ID          uuid.UUID   `json:"id" db:"id"`
	Name        string      `json:"name" db:"name"`
	Description string      `json:"description" db:"description"`
	Claims      interface{} `json:"claims" db:"claims"` // JSONB: [{model_id, permission, scope}]
	Domain      string      `json:"domain" db:"domain"`
	CreatedBy   string      `json:"created_by" db:"created_by"`
	CreatedAt   time.Time   `json:"created_at" db:"created_at"`
	Version     int         `json:"version" db:"version"`
}

// JITAddonGrant represents a temporary claim granted to a user for a fixed time window.
type JITAddonGrant struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    string    `json:"user_id" db:"user_id"`
	BundleID  uuid.UUID `json:"bundle_id" db:"bundle_id"`
	GrantedBy string    `json:"granted_by" db:"granted_by"`
	GrantedAt time.Time `json:"granted_at" db:"granted_at"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
	Reason    string    `json:"reason" db:"reason"`
	Status    string    `json:"status" db:"status"` // 'active','expired','revoked'
}
