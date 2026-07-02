package security

import (
	"time"

	"github.com/google/uuid"
)

// SecurityProfile represents a tenant-scoped or global blueprint security profile.
type SecurityProfile struct {
	ProfileID       uuid.UUID  `db:"profile_id" json:"profile_id"`
	TenantID        *uuid.UUID `db:"tenant_id" json:"tenant_id"` // NULL indicates Gold Copy System Blueprint
	ProfileKey      string     `db:"profile_key" json:"profile_key"`
	ProfileName     string     `db:"profile_name" json:"profile_name"`
	ParentProfileID *uuid.UUID `db:"parent_profile_id" json:"parent_profile_id,omitempty"`
	CreatedAt       time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at" json:"updated_at"`
}

// IdentityProfileMapping translates external Identity Provider group/role claims to abstract traits.
type IdentityProfileMapping struct {
	MappingID      uuid.UUID `db:"mapping_id" json:"mapping_id"`
	TenantID       uuid.UUID `db:"tenant_id" json:"tenant_id"`
	IDPClientID    string    `db:"idp_client_id" json:"idp_client_id"`
	IDPGroupID     string    `db:"idp_group_id" json:"idp_group_id"`
	FunctionalRole string    `db:"functional_role" json:"functional_role"`
	ClearanceLevel string    `db:"clearance_level" json:"clearance_level"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
}

// ResolvedProfile compiles the active traits for a user operational context.
type ResolvedProfile struct {
	ProfileKey   string                 `json:"profile_key"`
	IsCustomized bool                   `json:"is_customized"`
	Attributes   map[string]interface{} `json:"attributes"`
}
