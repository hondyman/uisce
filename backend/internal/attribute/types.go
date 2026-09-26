package attribute

import (
	"time"

	"github.com/google/uuid"
)

// AttributeDef is a control-plane definition stored on alpha.
type AttributeDef struct {
	ID              uuid.UUID      `json:"id" db:"id"`
	TenantID        uuid.UUID      `json:"tenant_id" db:"tenant_id"`
	CoreID          *uuid.UUID     `json:"core_id,omitempty" db:"core_id"`
	IsShadow        bool           `json:"is_shadow" db:"is_shadow"`
	EntityType      string         `json:"entity_type" db:"entity_type"`
	TableRef        string         `json:"table_ref" db:"table_ref"`
	FieldCd         string         `json:"field_cd" db:"field_cd"`
	Name            string         `json:"name" db:"name"`
	Description     string         `json:"description,omitempty" db:"description"`
	DataType        string         `json:"data_type" db:"data_type"`
	JsonPath        string         `json:"json_path" db:"json_path"`
	IsRequired      bool           `json:"is_required" db:"is_required"`
	IsSearchable    bool           `json:"is_searchable" db:"is_searchable"`
	IsPII           bool           `json:"is_pii" db:"is_pii"`
	ValidationRules map[string]any `json:"validation_rules" db:"-"`
	PicklistValues  []any          `json:"picklist_values,omitempty" db:"-"`
	DefaultValue    *string        `json:"default_value,omitempty" db:"default_value"`
	DisplayOrder    int            `json:"display_order" db:"display_order"`
	Section         string         `json:"section,omitempty" db:"section"`
	IsActive        bool           `json:"is_active" db:"is_active"`
	Origin          string         `json:"origin,omitempty" db:"origin"` // CORE | CUSTOM
	SemanticTermID  *uuid.UUID     `json:"semantic_term_id,omitempty" db:"semantic_term_id"`
	SemanticTermName string        `json:"semantic_term_name,omitempty" db:"semantic_term_name"`
	CreatedAt       time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at" db:"updated_at"`
}

type CreateInput struct {
	EntityType      string         `json:"entity_type"`
	TableRef        string         `json:"table_ref"`
	FieldCd         string         `json:"field_cd"`
	Name            string         `json:"name"`
	Description     string         `json:"description"`
	DataType        string         `json:"data_type"`
	JsonPath        string         `json:"json_path"`
	IsRequired      bool           `json:"is_required"`
	IsSearchable    *bool          `json:"is_searchable"`
	IsPII           bool           `json:"is_pii"`
	ValidationRules map[string]any `json:"validation_rules"`
	PicklistValues  []any          `json:"picklist_values"`
	DefaultValue    *string        `json:"default_value"`
	DisplayOrder    int            `json:"display_order"`
	Section         string         `json:"section"`
	SemanticTermID  *uuid.UUID     `json:"semantic_term_id"`
}

type UpdateInput struct {
	Name            *string         `json:"name"`
	Description     *string         `json:"description"`
	DataType        *string         `json:"data_type"`
	IsRequired      *bool           `json:"is_required"`
	IsSearchable    *bool           `json:"is_searchable"`
	IsPII           *bool           `json:"is_pii"`
	ValidationRules *map[string]any `json:"validation_rules"`
	PicklistValues  *[]any          `json:"picklist_values"`
	DefaultValue    *string         `json:"default_value"`
	DisplayOrder    *int            `json:"display_order"`
	Section         *string         `json:"section"`
	SemanticTermID  *uuid.UUID      `json:"semantic_term_id"`
	ClearSemanticTerm bool          `json:"clear_semantic_term"`
}

// JSONPathBinding describes how a semantic field projects into custom_attributes.
type JSONPathBinding struct {
	FieldCd        string    `json:"field_cd"`
	EntityType     string    `json:"entity_type"`
	TableRef       string    `json:"table_ref"`
	SemanticTermID uuid.UUID `json:"semantic_term_id"`
	JSONColumn     string    `json:"json_column"` // always custom_attributes for now
	DataType       string    `json:"data_type"`
}

type EligibleEntity struct {
	EntityType   string     `json:"entity_type"`
	TableRef     string     `json:"table_ref"`
	DisplayName  string     `json:"display_name"`
	SchemaName   string     `json:"schema_name"`
	TableName    string     `json:"table_name"`
	QualifiedPath string    `json:"qualified_path"`
	DatasourceID *uuid.UUID `json:"datasource_id,omitempty"`
	FieldCount   int        `json:"field_count"`
	ColumnNodeID *uuid.UUID `json:"column_node_id,omitempty"`
	TableNodeID  *uuid.UUID `json:"table_node_id,omitempty"`
}

type PreviewRequest struct {
	EntityType string
	TableRef   string
	TenantID   uuid.UUID
	Limit      int
	Offset     int
	OrderBy    string
}

type PreviewResponse struct {
	Columns []ColumnMeta     `json:"columns"`
	Rows    []map[string]any `json:"rows"`
	Total   int64            `json:"total"`
	Showing int              `json:"showing"`
}

type ColumnMeta struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	Type    string `json:"type"`
	Source  string `json:"source"` // CORE | CUSTOM
	Section string `json:"section,omitempty"`
	FieldCd string `json:"field_cd,omitempty"`
}

type CoreColumn struct {
	Name     string
	DataType string
	Label    string
}
