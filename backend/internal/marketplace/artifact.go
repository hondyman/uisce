package marketplace

import (
	"encoding/json"
	"fmt"
	"regexp"
)

// ListingKind categorises what an artifact contains.
// Corresponds to the marketplace_listings.kind column.
type ListingKind string

const (
	KindRBAC              ListingKind = "rbac"
	KindRulesCalculations ListingKind = "rules_calculations"
	KindIntegration       ListingKind = "integration"
	KindBundle            ListingKind = "bundle"
)

// ArtifactType is the fine-grained type within a listing.
// Corresponds to marketplace_artifacts.artifact_type.
type ArtifactType string

const (
	ArtifactTypeRBACRole         ArtifactType = "rbac_role"
	ArtifactTypeRBACRolePack     ArtifactType = "rbac_role_pack"
	ArtifactTypeABACPolicy      ArtifactType = "abac_policy"
	ArtifactTypeRulesCalculation ArtifactType = "rules_calculation"
	ArtifactTypeIntegration     ArtifactType = "integration"
	ArtifactTypeBundle          ArtifactType = "bundle"
)

// ListingStatus controls visibility in browse.
// Only 'published' listings appear in the public catalog.
type ListingStatus string

const (
	StatusDraft     ListingStatus = "draft"
	StatusPublished ListingStatus = "published"
	StatusSuspended ListingStatus = "suspended"
	StatusRetired   ListingStatus = "retired"
)

// BillingPeriod controls how price is displayed.
type BillingPeriod string

const (
	PeriodOneTime BillingPeriod = "one_time"
	PeriodMonthly BillingPeriod = "monthly"
	PeriodAnnual  BillingPeriod = "annual"
)

// Artifact is the versioned, self-describing payload stored in marketplace_artifacts.
// The uisce.mp/v1 schema is the only supported version in this phase.
// All fields are required unless noted.
type Artifact struct {
	SchemaVersion       string         `json:"schema_version"`        // must be "uisce.mp/v1"
	ArtifactType       ArtifactType   `json:"artifact_type"`        // rbac_role | abac_policy | ...
	ArtifactVersion    string         `json:"artifact_version"`     // semver, e.g. "1.2.0"
	MinPlatformVersion string         `json:"min_platform_version"`  // minimum uisce platform version
	Definition         ArtifactDef    `json:"definition"`           // the actual configuration
	CanonicalSHA256    string         `json:"canonical_sha256,omitempty"` // server-computed, read-only in JSON
}

// ArtifactDef carries the type-specific configuration inside an Artifact.
// It is always valid JSON Object; unknown keys are rejected by Validate().
type ArtifactDef map[string]any

var validArtifactTypes = map[ArtifactType]bool{
	ArtifactTypeRBACRole:         true,
	ArtifactTypeRBACRolePack:     true,
	ArtifactTypeABACPolicy:      true,
	ArtifactTypeRulesCalculation: true,
	ArtifactTypeIntegration:     true,
	ArtifactTypeBundle:          true,
}

var validKinds = map[ListingKind]bool{
	KindRBAC:              true,
	KindRulesCalculations:  true,
	KindIntegration:       true,
	KindBundle:            true,
}

var validStatuses = map[ListingStatus]bool{
	StatusDraft:     true,
	StatusPublished: true,
	StatusSuspended: true,
	StatusRetired:   true,
}

var validBillingPeriods = map[BillingPeriod]bool{
	PeriodOneTime: true,
	PeriodMonthly:  true,
	PeriodAnnual:  true,
}

// semverRE validates Semantic Versioning 2.0.0 strings.
var semverRE = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)

// Validate checks the artifact for structural validity.
// It does NOT connect to a database; it only validates structure and format.
// Returns a list of human-readable validation errors; empty slice means valid.
func (a *Artifact) Validate() []string {
	var errs []string

	if a.SchemaVersion != "uisce.mp/v1" {
		errs = append(errs, fmt.Sprintf("unsupported schema_version %q (only uisce.mp/v1 is supported)", a.SchemaVersion))
	}

	if !validArtifactTypes[a.ArtifactType] {
		errs = append(errs, fmt.Sprintf("unknown artifact_type %q", a.ArtifactType))
	}

	if !semverRE.MatchString(a.ArtifactVersion) {
		errs = append(errs, fmt.Sprintf("artifact_version %q is not a valid semver string", a.ArtifactVersion))
	}

	if a.Definition == nil {
		errs = append(errs, "definition is required and must be a JSON object")
	}

	// Type-specific definition validation.
	switch a.ArtifactType {
	case ArtifactTypeRBACRole, ArtifactTypeRBACRolePack:
		errs = append(errs, validateRBACDef(a.Definition)...)
	case ArtifactTypeABACPolicy:
		errs = append(errs, validateABACDef(a.Definition)...)
	}

	return errs
}

// ValidateListing performs structural validation on listing metadata.
// It does NOT check database referential integrity.
func ValidateListing(kind ListingKind, status ListingStatus, billingPeriod BillingPeriod,
	title, description string, priceCents int) []string {
	var errs []string

	if !validKinds[kind] {
		errs = append(errs, fmt.Sprintf("unknown kind %q", kind))
	}
	if !validStatuses[status] {
		errs = append(errs, fmt.Sprintf("unknown status %q", status))
	}
	if !validBillingPeriods[billingPeriod] {
		errs = append(errs, fmt.Sprintf("unknown billing_period %q", billingPeriod))
	}
	if len(title) == 0 {
		errs = append(errs, "title is required")
	}
	if len(title) > 150 {
		errs = append(errs, "title must be at most 150 characters")
	}
	if len(description) > 4000 {
		errs = append(errs, "description must be at most 4000 characters")
	}
	if priceCents < 0 {
		errs = append(errs, "price_cents cannot be negative")
	}
	return errs
}

// RBAC-specific definition fields that are allowed.
// Any key NOT in this list is rejected as a security measure.
var allowedRBACDefKeys = map[string]bool{
	"role_key":        true,
	"role_name":       true,
	"description":     true,
	"permissions":     true,
	"role_type":       true,
	"role_level":      true,
	"categories":      true,
	"bp_role_pages":   true,
	"metadata":        true,
}

// maxRBACDefBytes prevents memory exhaustion: artifact definition must be < 256 KiB.
const maxRBACDefBytes = 256 * 1024

func validateRBACDef(def ArtifactDef) []string {
	var errs []string
	if def == nil {
		return []string{"definition is required for rbac_role artifact"}
	}

	// Reject unknown keys.
	for k := range def {
		if !allowedRBACDefKeys[k] {
			errs = append(errs, fmt.Sprintf("definition contains disallowed key %q", k))
		}
	}

	// role_key format check (alphanumeric + underscore, max 80).
	if v, ok := def["role_key"].(string); ok {
		if len(v) == 0 {
			errs = append(errs, "definition.role_key is required")
		}
		if len(v) > 80 {
			errs = append(errs, "definition.role_key must be at most 80 characters")
		}
		if !regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`).MatchString(v) {
			errs = append(errs, "definition.role_key must start with a letter and contain only alphanumeric or underscore characters")
		}
	}

	// permissions must be []string if present.
	if v, ok := def["permissions"]; ok {
		perms, ok := v.([]any)
		if !ok {
			errs = append(errs, "definition.permissions must be an array")
		} else {
			for i, p := range perms {
				if _, ok := p.(string); !ok {
					errs = append(errs, fmt.Sprintf("definition.permissions[%d] must be a string", i))
				}
			}
		}
	}

	// Size guard: definition must be marshalable to < 256 KiB.
	if sz, _ := json.Marshal(def); len(sz) > maxRBACDefBytes {
		errs = append(errs, fmt.Sprintf("definition exceeds maximum size of %d bytes", maxRBACDefBytes))
	}

	return errs
}

// Allowed ABAC resource types and actions (from bp_rbac_handlers.go allow-lists).
var allowedResourceTypes = map[string]bool{
	"portfolio":    true,
	"trade":        true,
	"position":     true,
	"instrument":   true,
	"account":      true,
	"report":       true,
	"user":         true,
	"role":         true,
	"workflow":     true,
	"compliance":    true,
}

var allowedActions = map[string]bool{
	"create":  true,
	"read":    true,
	"update":  true,
	"delete":  true,
	"approve": true,
	"submit":  true,
	"execute": true,
}

func validateABACDef(def ArtifactDef) []string {
	var errs []string
	if def == nil {
		return []string{"definition is required for abac_policy artifact"}
	}

	if v, ok := def["resource_type"].(string); ok && !allowedResourceTypes[v] {
		errs = append(errs, fmt.Sprintf("definition.resource_type %q is not in the allowed list", v))
	}
	if v, ok := def["action"].(string); ok && !allowedActions[v] {
		errs = append(errs, fmt.Sprintf("definition.action %q is not in the allowed list", v))
	}

	// Size guard.
	if sz, _ := json.Marshal(def); len(sz) > maxRBACDefBytes {
		errs = append(errs, fmt.Sprintf("definition exceeds maximum size of %d bytes", maxRBACDefBytes))
	}

	return errs
}
