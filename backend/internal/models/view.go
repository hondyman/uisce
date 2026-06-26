package models

import "time"

// ColumnMaskingRule defines a single rule for masking a column.
type ColumnMaskingRule struct {
	Column    string `json:"column"`    // The column to mask
	Rule      string `json:"rule"`      // The masking rule, e.g., "hash", "redact", "nullify"
	Condition string `json:"condition"` // The condition under which to apply the mask, e.g., "user.clearance != 'high'"
}

// ABACPolicy contains the fine-grained access control rules for a view.
type ABACPolicy struct {
	RowFilters    []string            `json:"rowFilters"`
	ColumnMasking []ColumnMaskingRule `json:"columnMasking"`
}

// ViewDefinition represents a curated, queryable projection of data from a bundle.
// It is the unit of runtime ABAC enforcement.
type ViewDefinition struct {
	ID          string                    `json:"id"`
	Name        string                    `json:"name"`
	Description string                    `json:"description"`
	BundleID    string                    `json:"bundleId"` // The parent bundle this view is derived from
	Measures    []SemanticObjectReference `json:"measures"`
	Dimensions  []SemanticObjectReference `json:"dimensions"`
	ABAC        ABACPolicy                `json:"abac"`
	CreatedAt   time.Time                 `json:"createdAt"`
	UpdatedAt   time.Time                 `json:"updatedAt"`
}
