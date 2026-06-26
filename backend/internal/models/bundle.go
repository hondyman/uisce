package models

import "time"

// BundleStatus represents the lifecycle state of a DataBundle.
type BundleStatus string

const (
	StatusDraft      BundleStatus = "Draft"
	StatusCertified  BundleStatus = "Certified"
	StatusPublished  BundleStatus = "Published"
	StatusDeprecated BundleStatus = "Deprecated"
)

// SemanticObjectReference is a pointer to a specific semantic object (such as a measure or dimension).
type SemanticObjectReference struct {
	ID      string `json:"id"`
	ModelID string `json:"modelId"`
	Type    string `json:"type"`
}

// AttributeCondition captures a single attribute-based requirement in a policy expression.
type AttributeCondition struct {
	Attribute string   `json:"attribute"`
	Operator  string   `json:"operator"`
	Values    []string `json:"values"`
}

// Policy defines a single access control rule for administrative actions on bundles.
type Policy struct {
	ID          string               `json:"id"`
	Effect      string               `json:"effect"`
	Actions     []string             `json:"actions"`
	Resources   []string             `json:"resources"`
	Description string               `json:"description"`
	Conditions  []AttributeCondition `json:"conditions,omitempty"`
}

// BundleRowPolicy defines attribute-conditional row filters applied at query time.
type BundleRowPolicy struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Member      string               `json:"member"`
	Operator    string               `json:"operator"`
	Values      []string             `json:"values"`
	Conditions  []AttributeCondition `json:"conditions"`
	Priority    int                  `json:"priority,omitempty"`
}

// BundleColumnPolicy defines attribute-conditional column masking rules.
type BundleColumnPolicy struct {
	ID            string               `json:"id"`
	Name          string               `json:"name"`
	Description   string               `json:"description"`
	Columns       []string             `json:"columns"`
	MaskType      string               `json:"maskType"`
	MaskValue     string               `json:"maskValue,omitempty"`
	Conditions    []AttributeCondition `json:"conditions"`
	AllowedRoles  []string             `json:"allowedRoles,omitempty"`
	Audience      []string             `json:"audience,omitempty"`
	Jurisdictions []string             `json:"jurisdictions,omitempty"`
	Tags          []string             `json:"tags,omitempty"`
	EffectiveFrom *time.Time           `json:"effectiveFrom,omitempty"`
	EffectiveTo   *time.Time           `json:"effectiveTo,omitempty"`
}

// BundleViewRef links a bundle to one or more published view definitions.
type BundleViewRef struct {
	ViewID       string      `json:"viewId"`
	ViewVersion  string      `json:"viewVersion,omitempty"`
	Primary      bool        `json:"primary"`
	Audience     []string    `json:"audience,omitempty"`
	Tags         []string    `json:"tags,omitempty"`
	AbacOverride *ABACPolicy `json:"abacOverride,omitempty"`
}

// BundleRoleAssignment describes coarse-grained entitlement for a bundle.
type BundleRoleAssignment struct {
	Role         string   `json:"role"`
	DefaultViews []string `json:"defaultViews,omitempty"`
}

// ViewPolicyBundle describes view-specific overrides enforced at runtime.
type ViewPolicyBundle struct {
	RowFilters    []string            `json:"rowFilters,omitempty"`
	ColumnMasking []ColumnMaskingRule `json:"columnMasking,omitempty"`
}

// BundlePolicySet aggregates high-level governance policies that apply to a bundle.
type BundlePolicySet struct {
	RoleAssignments  []BundleRoleAssignment      `json:"roleAssignments,omitempty"`
	AttributeFilters []AttributeCondition        `json:"attributeFilters,omitempty"`
	ViewOverrides    map[string]ViewPolicyBundle `json:"viewOverrides,omitempty"`
}

// BundleChangeRecord preserves a high-level audit of each lifecycle transition.
type BundleChangeRecord struct {
	Version    string    `json:"version"`
	State      string    `json:"state"`
	Action     string    `json:"action"`
	Actor      string    `json:"actor"`
	Timestamp  time.Time `json:"timestamp"`
	Notes      string    `json:"notes,omitempty"`
	DiffDigest string    `json:"diffDigest,omitempty"`
}

// BundleLifecycle tracks key lifecycle timestamps and metadata per bundle.
type BundleLifecycle struct {
	DraftedAt    time.Time  `json:"draftedAt"`
	CertifiedAt  *time.Time `json:"certifiedAt,omitempty"`
	PublishedAt  *time.Time `json:"publishedAt,omitempty"`
	DeprecatedAt *time.Time `json:"deprecatedAt,omitempty"`
	RetiredAt    *time.Time `json:"retiredAt,omitempty"`
	LastAction   string     `json:"lastAction,omitempty"`
	LastActor    string     `json:"lastActor,omitempty"`
	LastNotes    string     `json:"lastNotes,omitempty"`
}

// BundleMaintainer captures stewardship contact information for the bundle.
type BundleMaintainer struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
	Role  string `json:"role,omitempty"`
}

// BundleDependency tracks upstream or downstream dependencies referenced by a bundle.
type BundleDependency struct {
	ID           string `json:"id"`
	Version      string `json:"version,omitempty"`
	Type         string `json:"type,omitempty"`
	Relationship string `json:"relationship,omitempty"`
}

// BundleQualityCheck represents an automated control attached to a bundle.
type BundleQualityCheck struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Category    string     `json:"category,omitempty"`
	Schedule    string     `json:"schedule,omitempty"`
	Status      string     `json:"status,omitempty"`
	LastRunAt   *time.Time `json:"lastRunAt,omitempty"`
	LastOutcome string     `json:"lastOutcome,omitempty"`
}

// BundleManifest consolidates descriptive metadata used for governance catalogs.
type BundleManifest struct {
	Identifier    string                    `json:"identifier"`
	Summary       string                    `json:"summary,omitempty"`
	Domain        string                    `json:"domain,omitempty"`
	Products      []string                  `json:"products,omitempty"`
	SourceModels  []SemanticObjectReference `json:"sourceModels,omitempty"`
	Dependencies  []BundleDependency        `json:"dependencies,omitempty"`
	Maintainers   []BundleMaintainer        `json:"maintainers,omitempty"`
	QualityChecks []BundleQualityCheck      `json:"qualityChecks,omitempty"`
	SLAs          []BundleSLA               `json:"slas,omitempty"`
	Regulatory    []string                  `json:"regulatory,omitempty"`
	LastSyncedAt  *time.Time                `json:"lastSyncedAt,omitempty"`
}

// BundleComposition captures the curated semantic selections for a bundle draft.
type BundleComposition struct {
	Measures   []SemanticObjectReference `json:"measures,omitempty"`
	Dimensions []SemanticObjectReference `json:"dimensions,omitempty"`
	Filters    []string                  `json:"filters,omitempty"`
}

// BundleAuditMetadata provides quick access to key audit fields.
type BundleAuditMetadata struct {
	CreatedBy      string     `json:"createdBy"`
	CreatedAt      time.Time  `json:"createdAt"`
	LastModifiedBy string     `json:"lastModifiedBy,omitempty"`
	LastModifiedAt *time.Time `json:"lastModifiedAt,omitempty"`
	LastReviewedBy string     `json:"lastReviewedBy,omitempty"`
	LastReviewedAt *time.Time `json:"lastReviewedAt,omitempty"`
}

// BundleDraftInput is used to author or update bundle metadata prior to certification.
type BundleDraftInput struct {
	Name            string            `json:"name"`
	Description     string            `json:"description"`
	Owner           string            `json:"owner"`
	Audience        []string          `json:"audience,omitempty"`
	Jurisdictions   []string          `json:"jurisdictions,omitempty"`
	Tags            []string          `json:"tags,omitempty"`
	EffectiveFrom   time.Time         `json:"effectiveFrom"`
	EffectiveTo     *time.Time        `json:"effectiveTo,omitempty"`
	Policies        BundlePolicySet   `json:"policies"`
	ViewRefs        []BundleViewRef   `json:"viewRefs,omitempty"`
	ResolvedLineage map[string]string `json:"resolvedLineage,omitempty"`
	Composition     BundleComposition `json:"composition"`
	Notes           string            `json:"notes,omitempty"`
}

// DataBundle represents a bundle and its rich governance metadata.
type DataBundle struct {
	ID                 string                    `json:"id"`
	Name               string                    `json:"name"`
	Description        string                    `json:"description"`
	Owner              string                    `json:"owner"`
	Version            string                    `json:"version"`
	Status             BundleStatus              `json:"status"`
	Audience           []string                  `json:"audience,omitempty"`
	Jurisdictions      []string                  `json:"jurisdictions,omitempty"`
	Tags               []string                  `json:"tags,omitempty"`
	EffectiveFrom      *time.Time                `json:"effectiveFrom,omitempty"`
	EffectiveTo        *time.Time                `json:"effectiveTo,omitempty"`
	Measures           []SemanticObjectReference `json:"measures,omitempty"`
	Dimensions         []SemanticObjectReference `json:"dimensions,omitempty"`
	Composition        BundleComposition         `json:"composition"`
	ViewRefs           []BundleViewRef           `json:"viewRefs,omitempty"`
	RowPolicies        []BundleRowPolicy         `json:"rowPolicies,omitempty"`
	ColumnPolicies     []BundleColumnPolicy      `json:"columnPolicies,omitempty"`
	Policies           BundlePolicySet           `json:"policies"`
	AllowedRoles       []string                  `json:"allowedRoles,omitempty"`
	Lifecycle          BundleLifecycle           `json:"lifecycle"`
	Manifest           BundleManifest            `json:"manifest"`
	AuditTrail         []BundleChangeRecord      `json:"auditTrail,omitempty"`
	AuditMetadata      *BundleAuditMetadata      `json:"auditMetadata,omitempty"`
	CertificationNotes string                    `json:"certificationNotes,omitempty"`
	ContentHash        string                    `json:"contentHash,omitempty"`
	ResolvedLineage    map[string]string         `json:"resolvedLineage,omitempty"`
	CreatedAt          time.Time                 `json:"createdAt"`
	UpdatedAt          time.Time                 `json:"updatedAt"`
	PublishedAt        *time.Time                `json:"publishedAt,omitempty"`
	DeprecatedAt       *time.Time                `json:"deprecatedAt,omitempty"`
}

// HasField returns true if the bundle contains a field with the given name
func (b *DataBundle) HasField(name string) bool {
	for _, m := range b.Measures {
		if m.ID == name || m.ModelID == name {
			return true
		}
	}
	for _, d := range b.Dimensions {
		if d.ID == name || d.ModelID == name {
			return true
		}
	}
	return false
}
