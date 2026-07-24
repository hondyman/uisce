package catalog

// AttributeType defines the logical type of a product attribute
type AttributeType string

const (
	AttrString AttributeType = "string"
	AttrNumber AttributeType = "number"
	AttrBool   AttributeType = "bool"
	AttrEnum   AttributeType = "enum"
	AttrObject AttributeType = "object"
	AttrArray  AttributeType = "array"
)

// AttributeDef describes a single field/attribute on a product
type AttributeDef struct {
	Name        string        `json:"name"`
	Type        AttributeType `json:"type"`
	Required    bool          `json:"required"`
	EnumValues  []string      `json:"enumValues,omitempty"`
	Description string        `json:"description,omitempty"`
	// Original CDM metadata can be preserved here if needed
	CDMMetadata map[string]string `json:"cdmMetadata,omitempty"`
}

// ProductDef describes a product template (e.g. InterestRateSwap)
type ProductDef struct {
	ID          string         `json:"id"`          // Unique internal ID
	Label       string         `json:"label"`       // Display name
	Family      string         `json:"family"`      // e.g. "IR", "FX", "Credit"
	CdmType     string         `json:"cdmType"`     // Fully qualified CDM type name
	Description string         `json:"description"` // Derived from CDM docs
	Attributes  []AttributeDef `json:"attributes"`
}

// Catalog is the top-level container for all product definitions
type Catalog struct {
	Products []ProductDef `json:"products"`
}
