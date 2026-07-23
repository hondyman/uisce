package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// ============================================================================
// SEMANTIC TERM TAGGING SUPPORT
// ============================================================================

// Tag represents a semantic term classification tag
type Tag struct {
	ID          string    `json:"id"`
	TagKey      string    `json:"tag_key"`      // Unique identifier (e.g., "sales", "financial_metric")
	TagLabel    string    `json:"tag_label"`    // Display name (e.g., "Sales", "Financial Metric")
	TagCategory string    `json:"tag_category"` // business_area, data_type, domain, usage_pattern, sensitivity, governance
	Description string    `json:"description"`
	ColorCode   string    `json:"color_code"`
	IconName    string    `json:"icon_name"`
	AutoSuggest bool      `json:"auto_suggest"` // Flag if wizard should suggest this
	SortOrder   int       `json:"sort_order"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TagCategory represents a classification of tags
type TagCategory struct {
	CategoryName string `json:"category"`
	DisplayName  string `json:"display_name"`
	Icon         string `json:"icon"`
	Tags         []*Tag `json:"tags,omitempty"`
}

// TagSuggestion represents a wizard-suggested tag with confidence score
type TagSuggestion struct {
	TagKey           string  `json:"tag_key"`
	TagLabel         string  `json:"tag_label"`
	TagCategory      string  `json:"tag_category"`
	SuggestionReason string  `json:"suggestion_reason"` // inferred_from_datatype, inferred_from_name, etc.
	ConfidenceScore  float64 `json:"confidence_score"`  // 0.0-1.0
	ColorCode        string  `json:"color_code"`
	IconName         string  `json:"icon_name"`
}

// SemanticTermTagAssignment represents the relationship between term and tags
type SemanticTermTagAssignment struct {
	ID               string    `json:"id"`
	SemanticTermID   string    `json:"semantic_term_id"`
	TagKey           string    `json:"tag_key"`
	SuggestionReason string    `json:"suggestion_reason,omitempty"` // Track how tag was added
	ConfidenceScore  float64   `json:"confidence_score,omitempty"`  // 0.0-1.0 for suggested tags
	IsAccepted       *bool     `json:"is_accepted,omitempty"`       // Track if suggestion was accepted/rejected
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// TagInput represents input for creating or updating a tag
type TagInput struct {
	TagKey      string `json:"tag_key"`
	TagLabel    string `json:"tag_label"`
	TagCategory string `json:"tag_category"`
	Description string `json:"description,omitempty"`
	ColorCode   string `json:"color_code"`
	IconName    string `json:"icon_name,omitempty"`
	AutoSuggest bool   `json:"auto_suggest"`
	SortOrder   int    `json:"sort_order"`
}

// TagSuggestionRequest represents the input for tag suggestion
type TagSuggestionRequest struct {
	NodeName        string           `json:"node_name"`        // Column/field name
	DisplayName     string           `json:"display_name"`     // Display name
	Description     string           `json:"description"`      // Field description
	DataType        SemanticDataType `json:"data_type"`        // Data type of field
	Domain          string           `json:"domain"`           // Business domain
	Expression      string           `json:"expression"`       // SQL/calculation expression
	PhysicalMapping *PhysicalMapping `json:"physical_mapping"` // Database table/column
	Relationships   []string         `json:"relationships"`    // Related business objects
	ExistingTags    []string         `json:"existing_tags"`    // Already assigned tags (for refinement)
}

// TagSuggestionResponse contains suggested tags with confidence scores
type TagSuggestionResponse struct {
	Suggestions []TagSuggestion   `json:"suggestions"`
	Reasons     map[string]string `json:"reasons"` // Explanation for each suggestion
}

// Value implements the driver.Valuer interface for JSONB storage
func (t Tag) Value() (driver.Value, error) {
	return json.Marshal(t)
}

// Value implements the driver.Valuer interface for JSONB storage
func (ts TagSuggestion) Value() (driver.Value, error) {
	return json.Marshal(ts)
}

// Value implements the driver.Valuer interface for JSONB storage
func (tta SemanticTermTagAssignment) Value() (driver.Value, error) {
	return json.Marshal(tta)
}
