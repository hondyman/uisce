package models

import "time"

// ScriptState represents the lifecycle state of a script.
type ScriptState string

const (
	ScriptStateDraft      ScriptState = "draft"
	ScriptStateCertified  ScriptState = "certified"
	ScriptStatePublished  ScriptState = "published"
	ScriptStateDeprecated ScriptState = "deprecated"
)

// ScriptSummary provides a high-level overview of a script.
type ScriptSummary struct {
	ID            string      `json:"id"`
	Name          string      `json:"name"`
	Description   string      `json:"description,omitempty"`
	DomainTags    []string    `json:"domainTags"`
	Scope         string      `json:"scope"` // "table" or "semantic"
	State         ScriptState `json:"state"`
	LatestVersion string      `json:"latestVersion"`
	Sensitivity   string      `json:"sensitivity,omitempty"` // "low", "medium", "high"
	Steward       string      `json:"steward,omitempty"`
	UpdatedAt     time.Time   `json:"updatedAt"`
}

// ScriptApproval represents a record of an approval action.
type ScriptApproval struct {
	By   string    `json:"by"`
	At   time.Time `json:"at"`
	Note string    `json:"note,omitempty"`
}

// ScriptTestResult holds the outcome of a test run.
type ScriptTestResult struct {
	Pass      bool   `json:"pass"`
	Summary   string `json:"summary"`
	ReportURL string `json:"reportUrl,omitempty"`
}

// ScriptVersion contains the content and metadata for a specific version of a script.
type ScriptVersion struct {
	Version   string            `json:"version"`
	CreatedAt time.Time         `json:"createdAt"`
	CreatedBy string            `json:"createdBy"`
	Content   string            `json:"content"`
	Hash      string            `json:"hash"`
	Tests     *ScriptTestResult `json:"tests,omitempty"`
	Approvals []ScriptApproval  `json:"approvals,omitempty"`
}

// ScriptLineage defines a script's connections to other objects.
type ScriptLineage struct {
	AttachedTo   []string `json:"attachedTo"`
	Dependencies []string `json:"dependencies,omitempty"`
}

// ScriptAttachment represents an external document attached to a script.
type ScriptAttachment struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// ScriptDetail is the full representation of a script, including all versions and metadata.
type ScriptDetail struct {
	ScriptSummary
	Versions    []ScriptVersion    `json:"versions"`
	Attachments []ScriptAttachment `json:"attachments,omitempty"`
	Lineage     ScriptLineage      `json:"lineage"`
}

// ImpactedBundle represents a bundle affected by a script change.
type ImpactedBundle struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
	State   string `json:"state"`
}

// ImpactedView represents a view affected by a script change.
type ImpactedView struct {
	Name       string `json:"name"`
	BundleID   string `json:"bundleId"`
	BundleName string `json:"bundleName"`
	State      string `json:"state"`
}

// ImpactedObject represents a generic semantic object affected by a script change.
type ImpactedObject struct {
	Type     string `json:"type"` // "measure", "dimension", etc.
	ID       string `json:"id"`
	BundleID string `json:"bundleId,omitempty"`
}

// ImpactReport summarizes the downstream dependencies of a script.
type ImpactReport struct {
	ScriptID        string           `json:"scriptId"`
	ScriptVersion   string           `json:"scriptVersion"`
	ImpactedBundles []ImpactedBundle `json:"impactedBundles"`
	ImpactedViews   []ImpactedView   `json:"impactedViews"`
	ImpactedObjects []ImpactedObject `json:"impactedObjects"`
}
