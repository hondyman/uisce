package pagestudio

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// CorePage represents a gold-copy page definition
type CorePage struct {
	ID                  uuid.UUID       `json:"id" db:"id"`
	Env                 string          `json:"env" db:"env"`
	TenantID            *string         `json:"tenant_id" db:"tenant_id"`
	Name                string          `json:"name" db:"name"`
	Slug                string          `json:"slug" db:"slug"`
	Description         string          `json:"description" db:"description"`
	Layout              json.RawMessage `json:"layout" db:"layout"`
	Components          json.RawMessage `json:"components" db:"components"`
	DataBindings        json.RawMessage `json:"data_bindings" db:"data_bindings"`
	Visibility          json.RawMessage `json:"visibility" db:"visibility"`
	SemanticFingerprint json.RawMessage `json:"semantic_fingerprint" db:"semantic_fingerprint"`
	Version             int             `json:"version" db:"version"`
	CreatedAt           time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at" db:"updated_at"`
	CreatedBy           string          `json:"created_by" db:"created_by"`
}

// VisibilityDefinition defines access rules
type VisibilityDefinition struct {
	Roles               []string `json:"roles" db:"roles"`
	EntitlementPolicies []string `json:"entitlement_policies" db:"entitlement_policies"`
}

// AIGenerateRequest defines the input for AI page generation
type AIGenerateRequest struct {
	BOName   string `json:"bo_name"`
	Intent   string `json:"intent"` // dashboard, list, detail, form
	TenantID string `json:"tenant_id"`
}

// AIGenerateResponse defines the output for AI page generation
type AIGenerateResponse struct {
	Layout       json.RawMessage `json:"layout"`
	Components   json.RawMessage `json:"components"`
	DataBindings json.RawMessage `json:"data_bindings"`
}

// PageOverlay represents a tenant-specific override
type PageOverlay struct {
	ID        uuid.UUID       `json:"id" db:"id"`
	ParentID  uuid.UUID       `json:"parent_id" db:"parent_id"`
	Env       string          `json:"env" db:"env"`
	TenantID  string          `json:"tenant_id" db:"tenant_id"`
	Overrides json.RawMessage `json:"overrides" db:"overrides"`
	Version   int             `json:"version" db:"version"`
	CreatedAt time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt time.Time       `json:"updated_at" db:"updated_at"`
	CreatedBy string          `json:"created_by" db:"created_by"`
}

// UpgradeStatus defines the state of a tenant upgrade reconciliation
type UpgradeStatus string

const (
	UpgradeStatusPending          UpgradeStatus = "pending"
	UpgradeStatusAccepted         UpgradeStatus = "accepted"
	UpgradeStatusPartiallyApplied UpgradeStatus = "partially_applied"
	UpgradeStatusDismissed        UpgradeStatus = "dismissed"
)

// ConflictItem represents a collision between core changes and tenant overrides
type ConflictItem struct {
	Type           string      `json:"type"` // componentProp, layout, binding, visibility
	ComponentID    *string     `json:"component_id,omitempty"`
	NodeID         *string     `json:"node_id,omitempty"`
	PropName       *string     `json:"prop_name,omitempty"`
	CoreBefore     interface{} `json:"core_before"`
	CoreAfter      interface{} `json:"core_after"`
	TenantOverride interface{} `json:"tenant_override"`
}

// ChangeItem represents a safe, inherited change from core
type ChangeItem struct {
	Type        string      `json:"type"`
	ComponentID *string     `json:"component_id,omitempty"`
	NodeID      *string     `json:"node_id,omitempty"`
	PropName    *string     `json:"prop_name,omitempty"`
	Before      interface{} `json:"before"`
	After       interface{} `json:"after"`
}

// UpgradeImpact records the results of a core-to-tenant diff
type UpgradeImpact struct {
	ID                    uuid.UUID       `json:"id" db:"id"`
	CorePageID            uuid.UUID       `json:"core_page_id" db:"core_page_id"`
	CoreOldVersion        int             `json:"core_old_version" db:"core_old_version"`
	CoreNewVersion        int             `json:"core_new_version" db:"core_new_version"`
	TenantID              string          `json:"tenant_id" db:"tenant_id"`
	OverlayPageID         uuid.UUID       `json:"overlay_page_id" db:"overlay_page_id"`
	Summary               string          `json:"summary" db:"summary"`
	Conflicts             json.RawMessage `json:"conflicts" db:"conflicts"`
	InheritedChanges      json.RawMessage `json:"inherited_changes" db:"inherited_changes"`
	NewCoreComponents     []string        `json:"new_core_components" db:"new_core_components"`
	RemovedCoreComponents []string        `json:"removed_core_components" db:"removed_core_components"`
	Status                UpgradeStatus   `json:"status" db:"status"`
	CreatedAt             time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at" db:"updated_at"`
}
