package bo

import (
	"encoding/json"

	"github.com/google/uuid"
)

// SubtypeConfig holds display metadata and visual theming for polymorphic facets
type SubtypeConfig struct {
	DisplayName string `json:"displayName"`
	Color       string `json:"color"`
	Icon        string `json:"icon"`
	Description string `json:"description,omitempty"`
}

// GroupedField represents a field scoped to a specific subtype facet and taxonomy node
type GroupedField struct {
	FieldID      uuid.UUID       `json:"fieldId"`
	FieldKey     string          `json:"key"`
	DisplayName  string          `json:"displayName"`
	Role         string          `json:"role"`
	DataType     string          `json:"dataType"`
	SubtypeScope string          `json:"subtypeScope"` // 'CORE', 'EQUITY', 'FIXED_INCOME', etc.
	IsRequired   bool            `json:"isRequired"`
	IsGoverned   bool            `json:"isGoverned"`
	Formula      string          `json:"formula,omitempty"`
	Metadata     json.RawMessage `json:"metadata,omitempty"`
}

// FieldGroup represents an accordion section derived from Business Term taxonomy or UI override
type FieldGroup struct {
	GroupKey    string         `json:"groupKey"`
	GroupName   string         `json:"groupName"`
	Sequence    int            `json:"sequence"`
	SubtypeCode string         `json:"subtypeCode,omitempty"`
	Fields      []GroupedField `json:"fields"`
}

// BOLayoutResponse is the complete structured layout response for the Business Object Studio
type BOLayoutResponse struct {
	BOID               uuid.UUID                `json:"boId"`
	BOKey              string                   `json:"boKey"`
	DiscriminatorField string                   `json:"discriminatorField,omitempty"`
	Subtypes           map[string]SubtypeConfig `json:"subtypes,omitempty"`
	ActiveSubtype      string                   `json:"activeSubtype"`
	Groups             []FieldGroup             `json:"groups"`
	TotalFields        int                      `json:"totalFields"`
}
