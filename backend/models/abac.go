package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Resource represents an entity protected by ABAC policies.
type Resource struct {
	ID         uuid.UUID       `db:"id" json:"id"`
	Name       string          `db:"name" json:"name"`
	Attributes json.RawMessage `db:"attributes" json:"attributes"`
	CreatedAt  time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time       `db:"updated_at" json:"updated_at"`
}

// Policy defines a set of rules for access control.
type Policy struct {
	ID            uuid.UUID       `db:"id" json:"id"`
	Name          string          `db:"name" json:"name"`
	Rules         json.RawMessage `db:"rules" json:"rules"`
	StartDate     *time.Time      `db:"start_date" json:"start_date,omitempty"`
	EndDate       *time.Time      `db:"end_date" json:"end_date,omitempty"`
	Schedule      json.RawMessage `db:"schedule" json:"schedule,omitempty"`
	LocationRules json.RawMessage `db:"location_rules" json:"location_rules,omitempty"`
	Priority      int             `db:"priority" json:"priority"`
	Active        bool            `db:"active" json:"active"`
	CreatedAt     time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time       `db:"updated_at" json:"updated_at"`
}

// AuditEvent logs a security-relevant event in the system.
type AuditEvent struct {
	ID        uuid.UUID       `db:"id" json:"id"`
	EventType string          `db:"event_type" json:"event_type"`
	UserID    *string         `db:"user_id" json:"user_id,omitempty"`
	Details   json.RawMessage `db:"details" json:"details"`
	Timestamp time.Time       `db:"timestamp" json:"timestamp"`
}

// Delegation allows one user to temporarily grant their permissions to another.
type Delegation struct {
	ID          uuid.UUID  `db:"id" json:"id"`
	DelegatorID string     `db:"delegator_id" json:"delegator_id"`
	DelegateeID string     `db:"delegatee_id" json:"delegatee_id"`
	PolicyID    uuid.UUID  `db:"policy_id" json:"policy_id"`
	Expiration  *time.Time `db:"expiration" json:"expiration,omitempty"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
}

// PolicyVersion stores a snapshot of a policy at a point in time.
type PolicyVersion struct {
	ID        uuid.UUID       `db:"id" json:"id"`
	PolicyID  uuid.UUID       `db:"policy_id" json:"policy_id"`
	Version   int             `db:"version" json:"version"`
	Rules     json.RawMessage `db:"rules" json:"rules"`
	CreatedAt time.Time       `db:"created_at" json:"created_at"`
}

// PolicyConflict identifies two policies that are in conflict.
type PolicyConflict struct {
	ID           uuid.UUID `db:"id" json:"id"`
	PolicyID1    uuid.UUID `db:"policy_id1" json:"policy_id1"`
	PolicyID2    uuid.UUID `db:"policy_id2" json:"policy_id2"`
	ConflictType string    `db:"conflict_type" json:"conflict_type"`
	Severity     string    `db:"severity" json:"severity"`
	DetectedAt   time.Time `db:"detected_at" json:"detected_at"`
}
