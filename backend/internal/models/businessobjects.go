package models

import (
	"database/sql"
	"encoding/json"
	"time"
)

// BUSINESS OBJECT DEFINITIONS
// ============================================================================

// HistoryMode represents the strategy for tracking temporal data
type HistoryMode string

const (
	HistoryModeExplicitRange HistoryMode = "EXPLICIT_RANGE"
	HistoryModeEventLog      HistoryMode = "EVENT_LOG"
)

// ============================================================================

// FieldDefinition represents a single field in a BO or subtype
type FieldDefinition struct {
	ID              string    `db:"id" json:"id"`
	Key             string    `db:"key" json:"key"`
	Name            string    `db:"name" json:"name"`
	DisplayName     string    `db:"display_name" json:"displayName"`
	TechnicalName   string    `db:"technical_name" json:"technicalName"`
	Type            string    `db:"type" json:"type"` // text, number, date, etc.
	IsCore          bool      `db:"is_core" json:"isCore"`
	IsRequired      bool      `db:"is_required" json:"isRequired"`
	IsSystem        bool      `db:"is_system" json:"isSystem"`
	Description     string    `db:"description" json:"description"`
	Role            FieldRole `json:"role"`
	SemanticTermID  string    `db:"semantic_term_id" json:"semanticTermId"`
	ReferenceEntity string    `db:"reference_entity" json:"referenceEntity"`
	Sequence        int       `db:"sequence" json:"sequence"`
	CoreID          string    `db:"core_id" json:"coreId,omitempty"` // Links to gold copy source field
	CreatedAt       time.Time `db:"created_at" json:"createdAt"`
	CreatedBy       string    `db:"created_by" json:"createdBy"`
	LastModifiedAt  time.Time `db:"last_modified_at" json:"lastModifiedAt"`
	LastModifiedBy  string    `db:"last_modified_by" json:"lastModifiedBy"`
}

// SubtypeDefinition represents a subtype within a BO
type SubtypeDefinition struct {
	ID             string            `db:"id" json:"id"`
	Key            string            `db:"key" json:"key"`
	Name           string            `db:"name" json:"name"`
	DisplayName    string            `db:"display_name" json:"displayName"`
	TechnicalName  string            `db:"technical_name" json:"technicalName"`
	Description    string            `db:"description" json:"description"`
	SubtypeFields  []FieldDefinition `json:"subtypeFields"`
	IsCore         bool              `db:"is_core" json:"isCore"`
	BasedOnEntity  string            `db:"based_on_entity" json:"basedOnEntity"`
	CloneParentKey string            `db:"clone_parent_key" json:"cloneParentKey"`
	CoreID         string            `db:"core_id" json:"coreId,omitempty"` // Links to gold copy source subtype
	Sequence       int               `db:"sequence" json:"sequence"`
	CreatedAt      time.Time         `db:"created_at" json:"createdAt"`
	CreatedBy      string            `db:"created_by" json:"createdBy"`
	LastModifiedAt time.Time         `db:"last_modified_at" json:"lastModifiedAt"`
	LastModifiedBy string            `db:"last_modified_by" json:"lastModifiedBy"`
}

// BusinessObjectDefinition represents a complete Business Object
type BusinessObjectDefinition struct {
	ID                     string                       `db:"id" json:"id"`
	Key                    string                       `db:"key" json:"key"`
	Name                   string                       `db:"name" json:"name"`
	DisplayName            string                       `db:"display_name" json:"displayName"`
	TechnicalName          string                       `db:"technical_name" json:"technicalName"`
	Description            string                       `db:"description" json:"description"`
	Icon                   string                       `db:"icon" json:"icon"`
	IsCore                 bool                         `db:"is_core" json:"isCore"`
	CoreFields             []FieldDefinition            `json:"coreFields"`
	CustomFields           []FieldDefinition            `json:"customFields"`
	Subtypes               map[string]SubtypeDefinition `json:"subtypes" db:"-"`
	Config                 json.RawMessage              `db:"config" json:"config"`
	ClonesFrom             string                       `db:"clones_from" json:"clonesFrom"`
	CloneParentKey         string                       `db:"clone_parent_key" json:"cloneParentKey"`
	CloneParentDisplayName string                       `db:"clone_parent_display_name" json:"cloneParentDisplayName"`
	Category               string                       `db:"category" json:"category"`
	ParentID               sql.NullString               `db:"parent_id" json:"parentId,omitempty"`
	InstanceCount          int                          `db:"instance_count" json:"instanceCount"`
	IsActive               bool                         `db:"is_active" json:"isActive"`
	EnableHistory          bool                         `db:"enable_history" json:"enableHistory"`
	HistoryMode            HistoryMode                  `db:"history_mode" json:"historyMode"`
	CreatedAt              time.Time                    `db:"created_at" json:"createdAt"`
	CreatedBy              string                       `db:"created_by" json:"createdBy"`
	LastModifiedAt         time.Time                    `db:"last_modified_at" json:"lastModifiedAt"`
	LastModifiedBy         string                       `db:"last_modified_by" json:"lastModifiedBy"`
	DriverTableID          sql.NullString               `db:"driver_table_id" json:"driverTableId,omitempty"`
	DriverTableName        string                       `db:"driver_table_name" json:"driverTableName"`
	TenantID               string                       `db:"tenant_id" json:"tenantId"`
	DatasourceID           sql.NullString               `db:"datasource_id" json:"datasourceId,omitempty"` // Nullable for global BOs
	CoreID                 sql.NullString               `db:"core_id" json:"coreId,omitempty"`             // Links to gold copy source BO (Workday-style extension)
}

// MarshalJSON handles custom JSON marshaling for BusinessObjectDefinition to properly serialize sql.NullString fields
func (b *BusinessObjectDefinition) MarshalJSON() ([]byte, error) {
	type Alias BusinessObjectDefinition
	aux := struct {
		ParentID      *string `json:"parentId,omitempty"`
		DriverTableID *string `json:"driverTableId,omitempty"`
		DatasourceID  *string `json:"datasourceId,omitempty"`
		CoreID        *string `json:"coreId,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(b),
	}

	// Only include fields if they are actually set (Valid && non-empty)
	if b.ParentID.Valid && b.ParentID.String != "" {
		aux.ParentID = &b.ParentID.String
	}
	if b.DriverTableID.Valid && b.DriverTableID.String != "" {
		aux.DriverTableID = &b.DriverTableID.String
	}
	if b.DatasourceID.Valid && b.DatasourceID.String != "" {
		aux.DatasourceID = &b.DatasourceID.String
	}
	if b.CoreID.Valid && b.CoreID.String != "" {
		aux.CoreID = &b.CoreID.String
	}

	return json.Marshal(aux)
}

// ============================================================================
// BO INSTANCES
// ============================================================================

// BusinessObjectInstance represents an individual record of a BO
type BusinessObjectInstance struct {
	ID                string                 `db:"id" json:"id"`
	TenantID          string                 `db:"tenant_id" json:"tenantId"`
	DatasourceID      string                 `db:"datasource_id" json:"datasourceId"`
	BusinessObjectID  string                 `db:"business_object_id" json:"businessObjectId"`
	BusinessObjectKey string                 `json:"businessObjectKey"`
	SubtypeID         sql.NullString         `db:"subtype_id" json:"subtypeId"`
	SubtypeKey        string                 `json:"subtypeKey"`
	CoreFieldValues   map[string]interface{} `db:"core_field_values" json:"coreFieldValues"`
	CustomFieldValues map[string]interface{} `db:"custom_field_values" json:"customFieldValues"`
	CreatedAt         time.Time              `db:"created_at" json:"createdAt"`
	CreatedBy         string                 `db:"created_by" json:"createdBy"`
	LastModifiedAt    time.Time              `db:"last_modified_at" json:"lastModifiedAt"`
	LastModifiedBy    string                 `db:"last_modified_by" json:"lastModifiedBy"`
	IsDeleted         bool                   `db:"is_deleted" json:"isDeleted"`
	DeletedAt         sql.NullTime           `db:"deleted_at" json:"deletedAt"`
}

// ============================================================================
// AUDIT LOG
// ============================================================================

// BOAuditLog tracks changes to BOs, subtypes, and fields
type BOAuditLog struct {
	ID         string                 `db:"id" json:"id"`
	TenantID   string                 `db:"tenant_id" json:"tenantId"`
	EntityType string                 `db:"entity_type" json:"entityType"` // business_object, subtype, field, instance
	EntityID   string                 `db:"entity_id" json:"entityId"`
	Action     string                 `db:"action" json:"action"` // create, update, delete, clone
	Changes    map[string]interface{} `db:"changes" json:"changes"`
	CreatedAt  time.Time              `db:"created_at" json:"createdAt"`
	CreatedBy  string                 `db:"created_by" json:"createdBy"`
}

// ============================================================================
// REQUEST/RESPONSE DTOs
// ============================================================================

// CreateBusinessObjectRequest represents a request to create a new BO
type CreateBusinessObjectRequest struct {
	Name            string                 `json:"name" validate:"required"`
	BOKey           string                 `json:"bo_key"`
	DisplayName     string                 `json:"display_name"`
	Description     string                 `json:"description"`
	Icon            string                 `json:"icon"`
	Category        string                 `json:"category"`
	TechnicalName   string                 `json:"technical_name"`
	DriverTableID   string                 `json:"driver_table_id"`
	DriverTableName string                 `json:"driver_table_name"`
	Status          string                 `json:"status"`
	CloneFromKey    string                 `json:"clone_from_key"` // if cloning an existing BO
	ParentID        string                 `json:"parent_id"`      // if creating a subtype
	DatasourceID    string                 `json:"datasource_id"`  // Optional: link to specific datasource
	EnableHistory   bool                   `json:"enable_history"`
	HistoryMode     string                 `json:"history_mode"`
	Config          map[string]interface{} `json:"config"`
}

// UpdateBusinessObjectRequest represents a request to update a BO
type UpdateBusinessObjectRequest struct {
	DisplayName     string                 `json:"displayName"`
	Description     string                 `json:"description"`
	Icon            string                 `json:"icon"`
	Category        string                 `json:"category"`
	IsActive        *bool                  `json:"isActive"`
	EnableHistory   *bool                  `json:"enableHistory"`
	HistoryMode     string                 `json:"historyMode"`
	Config          map[string]interface{} `json:"config"`
	DriverTableID   string                 `json:"driverTableId"`
	DriverTableName string                 `json:"driverTableName"`
}

// CreateSubtypeRequest represents a request to create a new subtype
type CreateSubtypeRequest struct {
	Name        string `json:"name" validate:"required"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
}

// UpdateSubtypeRequest represents a request to update a subtype
type UpdateSubtypeRequest struct {
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
}

// CreateFieldRequest represents a request to create a new field
type CreateFieldRequest struct {
	Name            string `json:"name" validate:"required"`
	DisplayName     string `json:"displayName"`
	Type            string `json:"type" validate:"required"`
	IsRequired      bool   `json:"isRequired"`
	Description     string `json:"description"`
	Role            string `json:"role"`
	SemanticTermID  string `json:"semanticTermId"`
	ReferenceEntity string `json:"referenceEntity"`
	Sequence        int    `json:"sequence"`
}

// UpdateFieldRequest represents a request to update a field
type UpdateFieldRequest struct {
	DisplayName     string `json:"displayName"`
	Description     string `json:"description"`
	Role            string `json:"role"`
	SemanticTermID  string `json:"semanticTermId"`
	IsRequired      bool   `json:"isRequired"`
	ReferenceEntity string `json:"referenceEntity"`
	Sequence        int    `json:"sequence"`
}

// CreateInstanceRequest represents a request to create a BO instance
type CreateInstanceRequest struct {
	BusinessObjectKey string                 `json:"businessObjectKey" validate:"required"`
	SubtypeKey        string                 `json:"subtypeKey"`
	CoreFieldValues   map[string]interface{} `json:"coreFieldValues"`
	CustomFieldValues map[string]interface{} `json:"customFieldValues"`
}

// UpdateInstanceRequest represents a request to update a BO instance
type UpdateInstanceRequest struct {
	CoreFieldValues   map[string]interface{} `json:"coreFieldValues"`
	CustomFieldValues map[string]interface{} `json:"customFieldValues"`
}

// CloneBORequest represents a request to clone a BO
type CloneBORequest struct {
	SourceBOKey string `json:"sourceBOKey" validate:"required"`
	NewName     string `json:"newName" validate:"required"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

// LEGACY COMPATIBILITY STRUCTURES
// ============================================================================

// BusinessObjectListField represents a field in the legacy listing format
type BusinessObjectListField struct {
	Name          string `json:"name"`
	Type          string `json:"type"`
	Label         string `json:"label"`
	ColumnName    string `json:"columnName"`              // Actual database column name
	TechnicalName string `json:"technicalName,omitempty"` // Technical/semantic identifier
	Description   string `json:"description,omitempty"`
}

// BusinessObjectListItem represents a BO in the legacy listing format
type BusinessObjectListItem struct {
	ID          string                    `json:"id" db:"id"`
	Name        string                    `json:"name" db:"name"`
	DisplayName string                    `json:"display_name" db:"display_name"`
	Description string                    `json:"description,omitempty" db:"description"`
	Fields      []BusinessObjectListField `json:"fields" db:"-"`
	Icon        string                    `json:"icon,omitempty" db:"icon"`
	Config      map[string]interface{}    `json:"config,omitempty" db:"-"`
}
