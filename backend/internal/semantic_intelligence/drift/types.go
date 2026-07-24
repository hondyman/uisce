package drift

// DriftType defines the category of semantic mismatch
type DriftType string

const (
	DriftTypeFieldRemoved    DriftType = "field_removed"
	DriftTypeTypeMismatch    DriftType = "type_mismatch"
	DriftTypeEndpointMissing DriftType = "endpoint_missing"
	DriftTypeArgMissing      DriftType = "arg_missing"
	DriftTypeBONotFound      DriftType = "bo_not_found"
)

// DriftEvent represents a specific detected issue
type DriftEvent struct {
	Type        DriftType `json:"type"`
	Severity    string    `json:"severity"` // critical, warning
	Description string    `json:"description"`
	ItemName    string    `json:"item_name"` // e.g. field name
	Expected    string    `json:"expected,omitempty"`
	Actual      string    `json:"actual,omitempty"`
}

// DriftReport summarizes the drift status of a page
type DriftReport struct {
	PageID     string       `json:"page_id"`
	HasDrift   bool         `json:"has_drift"`
	DetectedAt string       `json:"detected_at"`
	Events     []DriftEvent `json:"events"`
}
