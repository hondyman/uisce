package models

import "time"

// RoleStatus captures the lifecycle state of a role definition.
type RoleStatus string

const (
	RoleStatusDraft     RoleStatus = "Draft"
	RoleStatusActive    RoleStatus = "Active"
	RoleStatusSuspended RoleStatus = "Suspended"
	RoleStatusRetired   RoleStatus = "Retired"
)

// RoleType distinguishes between business, technical, or system roles.
type RoleType string

const (
	RoleTypeBusiness  RoleType = "Business"
	RoleTypeSystem    RoleType = "System"
	RoleTypeTechnical RoleType = "Technical"
)

// RoleScope denotes the reach of a role across tenants or environments.
type RoleScope string

const (
	RoleScopeGlobal      RoleScope = "Global"
	RoleScopeTenant      RoleScope = "Tenant"
	RoleScopeEnvironment RoleScope = "Environment"
)

// RolePermission describes coarse or fine-grained entitlements granted to a role.
type RolePermission struct {
	Resource    string               `json:"resource"`
	Actions     []string             `json:"actions"`
	Effect      string               `json:"effect"`
	Description string               `json:"description,omitempty"`
	Conditions  []AttributeCondition `json:"conditions,omitempty"`
}

// RoleMember tracks principals that have been bound to the role.
type RoleMember struct {
	UserID     string            `json:"userId"`
	AssignedAt time.Time         `json:"assignedAt"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// RoleChangeRecord captures audit information for the role lifecycle.
type RoleChangeRecord struct {
	Version    string    `json:"version"`
	State      string    `json:"state"`
	Action     string    `json:"action"`
	Actor      string    `json:"actor"`
	Timestamp  time.Time `json:"timestamp"`
	Notes      string    `json:"notes,omitempty"`
	DiffDigest string    `json:"diffDigest,omitempty"`
}

// RoleLifecycle stores timestamps and contextual metadata for role status changes.
type RoleLifecycle struct {
	CreatedAt   time.Time  `json:"createdAt"`
	ActivatedAt *time.Time `json:"activatedAt,omitempty"`
	SuspendedAt *time.Time `json:"suspendedAt,omitempty"`
	RetiredAt   *time.Time `json:"retiredAt,omitempty"`
	LastAction  string     `json:"lastAction,omitempty"`
	LastActor   string     `json:"lastActor,omitempty"`
	LastNotes   string     `json:"lastNotes,omitempty"`
}

// RoleAuditMetadata surfaces critical stewardship details for the role record.
type RoleAuditMetadata struct {
	CreatedBy      string     `json:"createdBy"`
	CreatedAt      time.Time  `json:"createdAt"`
	LastModifiedBy string     `json:"lastModifiedBy,omitempty"`
	LastModifiedAt *time.Time `json:"lastModifiedAt,omitempty"`
	LastReviewedBy string     `json:"lastReviewedBy,omitempty"`
	LastReviewedAt *time.Time `json:"lastReviewedAt,omitempty"`
}

// Role encapsulates the governance, membership, and permission metadata for a security role.
type Role struct {
	ID                   string               `json:"id"`
	Name                 string               `json:"name"`
	DisplayName          string               `json:"displayName"`
	Description          string               `json:"description"`
	Version              string               `json:"version"`
	Status               RoleStatus           `json:"status"`
	Type                 RoleType             `json:"type"`
	Owner                string               `json:"owner"`
	Scope                RoleScope            `json:"scope,omitempty"`
	TenantID             string               `json:"tenantId,omitempty"`
	Tags                 []string             `json:"tags,omitempty"`
	Attributes           map[string]string    `json:"attributes,omitempty"`
	Policies             []Policy             `json:"policies,omitempty"`
	Permissions          []RolePermission     `json:"permissions,omitempty"`
	AttributeConstraints []AttributeCondition `json:"attributeConstraints,omitempty"`
	Members              []RoleMember         `json:"members,omitempty"`
	BundleIDs            []string             `json:"bundleIds,omitempty"`
	AuditTrail           []RoleChangeRecord   `json:"auditTrail,omitempty"`
	AuditMetadata        *RoleAuditMetadata   `json:"auditMetadata,omitempty"`
	Lifecycle            RoleLifecycle        `json:"lifecycle"`
	CreatedAt            time.Time            `json:"createdAt"`
	UpdatedAt            time.Time            `json:"updatedAt"`
}
