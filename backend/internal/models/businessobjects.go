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
	Masked          bool      `db:"masked" json:"masked,omitempty"`
	MaskingPattern  string    `db:"masking_pattern" json:"maskingPattern,omitempty"`
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
	Bindings               []map[string]interface{}     `json:"bindings"`

	// Core Identity Triple & Semantic Governance (Feature 1)
	BOTypeID             sql.NullString `db:"bo_type_id" json:"boTypeId,omitempty"`
	ModelID              string         `db:"model_id" json:"modelId,omitempty"`
	ClassificationNodeID sql.NullString `db:"classification_node_id" json:"classificationNodeId,omitempty"` // Level 3 taxonomy node
	BusinessKeyNodeID    sql.NullString `db:"business_key_node_id" json:"businessKeyNodeId,omitempty"`       // Natural identifier (e.g. customer_bk)
	SemanticIDNodeID     sql.NullString `db:"semantic_id_node_id" json:"semanticIdNodeId,omitempty"`         // Universal semantic identifier (e.g. customer_sid)
	GrainNodeID          sql.NullString `db:"grain_node_id" json:"grainNodeId,omitempty"`                   // Dimensional granularity anchor
	CoreReferenceBOID    sql.NullString `db:"core_reference_bo_id" json:"coreReferenceBoId,omitempty"`       // Reference to master gold copy BO
	Status               string         `db:"status" json:"status,omitempty"`                               // DRAFT, IN_REVIEW, APPROVED, PUBLISHED, DEPRECATED
	StiDiscriminatorColumn sql.NullString `db:"sti_discriminator_column" json:"stiDiscriminatorColumn,omitempty"`
	ActiveSubtypeFilter    sql.NullString `db:"active_subtype_filter" json:"activeSubtypeFilter,omitempty"`
}

// MarshalJSON handles custom JSON marshaling for BusinessObjectDefinition to properly serialize sql.NullString fields
func (b *BusinessObjectDefinition) MarshalJSON() ([]byte, error) {
	type Alias BusinessObjectDefinition
	aux := struct {
		ParentID             *string `json:"parentId,omitempty"`
		DriverTableID        *string `json:"driverTableId,omitempty"`
		DatasourceID         *string `json:"datasourceId,omitempty"`
		CoreID               *string `json:"coreId,omitempty"`
		BOTypeID             *string `json:"boTypeId,omitempty"`
		ClassificationNodeID *string `json:"classificationNodeId,omitempty"`
		BusinessKeyNodeID    *string `json:"businessKeyNodeId,omitempty"`
		SemanticIDNodeID     *string `json:"semanticIdNodeId,omitempty"`
		GrainNodeID          *string `json:"grainNodeId,omitempty"`
		CoreReferenceBOID    *string `json:"coreReferenceBoId,omitempty"`
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
	if b.BOTypeID.Valid && b.BOTypeID.String != "" {
		aux.BOTypeID = &b.BOTypeID.String
	}
	if b.ClassificationNodeID.Valid && b.ClassificationNodeID.String != "" {
		aux.ClassificationNodeID = &b.ClassificationNodeID.String
	}
	if b.BusinessKeyNodeID.Valid && b.BusinessKeyNodeID.String != "" {
		aux.BusinessKeyNodeID = &b.BusinessKeyNodeID.String
	}
	if b.SemanticIDNodeID.Valid && b.SemanticIDNodeID.String != "" {
		aux.SemanticIDNodeID = &b.SemanticIDNodeID.String
	}
	if b.GrainNodeID.Valid && b.GrainNodeID.String != "" {
		aux.GrainNodeID = &b.GrainNodeID.String
	}
	if b.CoreReferenceBOID.Valid && b.CoreReferenceBOID.String != "" {
		aux.CoreReferenceBOID = &b.CoreReferenceBOID.String
	}

	return json.Marshal(aux)
}

// UnmarshalJSON handles custom JSON unmarshaling for BusinessObjectDefinition to properly deserialize sql.NullString fields
func (b *BusinessObjectDefinition) UnmarshalJSON(data []byte) error {
	type Alias BusinessObjectDefinition
	aux := struct {
		ParentID             *string `json:"parentId,omitempty"`
		DriverTableID        *string `json:"driverTableId,omitempty"`
		DatasourceID         *string `json:"datasourceId,omitempty"`
		CoreID               *string `json:"coreId,omitempty"`
		BOTypeID             *string `json:"boTypeId,omitempty"`
		ClassificationNodeID *string `json:"classificationNodeId,omitempty"`
		BusinessKeyNodeID    *string `json:"businessKeyNodeId,omitempty"`
		SemanticIDNodeID     *string `json:"semanticIdNodeId,omitempty"`
		GrainNodeID          *string `json:"grainNodeId,omitempty"`
		CoreReferenceBOID    *string `json:"coreReferenceBoId,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(b),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.ParentID != nil && *aux.ParentID != "" {
		b.ParentID = sql.NullString{String: *aux.ParentID, Valid: true}
	} else {
		b.ParentID = sql.NullString{Valid: false}
	}

	if aux.DriverTableID != nil && *aux.DriverTableID != "" {
		b.DriverTableID = sql.NullString{String: *aux.DriverTableID, Valid: true}
	} else {
		b.DriverTableID = sql.NullString{Valid: false}
	}

	if aux.DatasourceID != nil && *aux.DatasourceID != "" {
		b.DatasourceID = sql.NullString{String: *aux.DatasourceID, Valid: true}
	} else {
		b.DatasourceID = sql.NullString{Valid: false}
	}

	if aux.CoreID != nil && *aux.CoreID != "" {
		b.CoreID = sql.NullString{String: *aux.CoreID, Valid: true}
	} else {
		b.CoreID = sql.NullString{Valid: false}
	}

	if aux.BOTypeID != nil && *aux.BOTypeID != "" {
		b.BOTypeID = sql.NullString{String: *aux.BOTypeID, Valid: true}
	} else {
		b.BOTypeID = sql.NullString{Valid: false}
	}

	if aux.ClassificationNodeID != nil && *aux.ClassificationNodeID != "" {
		b.ClassificationNodeID = sql.NullString{String: *aux.ClassificationNodeID, Valid: true}
	} else {
		b.ClassificationNodeID = sql.NullString{Valid: false}
	}

	if aux.BusinessKeyNodeID != nil && *aux.BusinessKeyNodeID != "" {
		b.BusinessKeyNodeID = sql.NullString{String: *aux.BusinessKeyNodeID, Valid: true}
	} else {
		b.BusinessKeyNodeID = sql.NullString{Valid: false}
	}

	if aux.SemanticIDNodeID != nil && *aux.SemanticIDNodeID != "" {
		b.SemanticIDNodeID = sql.NullString{String: *aux.SemanticIDNodeID, Valid: true}
	} else {
		b.SemanticIDNodeID = sql.NullString{Valid: false}
	}

	if aux.GrainNodeID != nil && *aux.GrainNodeID != "" {
		b.GrainNodeID = sql.NullString{String: *aux.GrainNodeID, Valid: true}
	} else {
		b.GrainNodeID = sql.NullString{Valid: false}
	}

	if aux.CoreReferenceBOID != nil && *aux.CoreReferenceBOID != "" {
		b.CoreReferenceBOID = sql.NullString{String: *aux.CoreReferenceBOID, Valid: true}
	} else {
		b.CoreReferenceBOID = sql.NullString{Valid: false}
	}

	return nil
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
	DisplayName     string                 `json:"displayName"`
	DisplayNameSnake string                `json:"display_name"`
	Description     string                 `json:"description"`
	Icon            string                 `json:"icon"`
	Category        string                 `json:"category"`
	TechnicalName   string                 `json:"technicalName"`
	TechnicalNameSnake string              `json:"technical_name"`
	DriverTableID   string                 `json:"driverTableId"`
	DriverTableIDSnake string              `json:"driver_table_id"`
	DriverTableName string                 `json:"driverTableName"`
	DriverTableNameSnake string             `json:"driver_table_name"`
	Status          string                 `json:"status"`
	CloneFromKey    string                 `json:"cloneFromKey"` // if cloning an existing BO
	CloneFromKeySnake string               `json:"clone_from_key"`
	ParentID        string                 `json:"parentId"`      // if creating a subtype
	ParentIDSnake   string                 `json:"parent_id"`
	DatasourceID    string                 `json:"datasourceId"`  // Optional: link to specific datasource
	DatasourceIDSnake string               `json:"datasource_id"`
	EnableHistory   bool                   `json:"enableHistory"`
	HistoryMode     string                 `json:"historyMode"`
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

// ============================================================================
// ORM DATASOURCE & RUNTIME DATA MODELS
// ============================================================================

// BORecordFilter represents a filter applied to physical records
type BORecordFilter struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"` // eq, neq, gt, gte, lt, lte, like, in, is_null, is_not_null
	Value    interface{} `json:"value"`
}

// BORecordQueryRequest represents a request to query physical records through a Business Object
type BORecordQueryRequest struct {
	Page                int              `json:"page"`
	Limit               int              `json:"limit"`
	Search              string           `json:"search,omitempty"`
	Filters             []BORecordFilter `json:"filters,omitempty"`
	SortBy              string           `json:"sortBy,omitempty"`
	SortDir             string           `json:"sortDir,omitempty"` // ASC or DESC
	SubtypeKey          string           `json:"subtypeKey,omitempty"`
	AsOfValidTime       *time.Time       `json:"asOfValidTime,omitempty"`
	AsOfTransactionTime *time.Time       `json:"asOfTransactionTime,omitempty"`
}

// BORecordQueryResponse represents the paginated result of querying physical records
type BORecordQueryResponse struct {
	Total           int                      `json:"total"`
	Page            int                      `json:"page"`
	Limit           int                      `json:"limit"`
	Columns         []string                 `json:"columns"`
	Rows            []map[string]interface{} `json:"rows"`
	ExecutionTimeMs int64                    `json:"executionTimeMs"`
	DriverTable     string                   `json:"driverTable"`
	DatasourceID    string                   `json:"datasourceId"`
}

// BOCrudRecordRequest represents a request to create or update a physical record via BO
type BOCrudRecordRequest struct {
	Record     map[string]interface{} `json:"record"`
	SubtypeKey string                 `json:"subtypeKey,omitempty"`
}

// TableColumnIntrospection represents an introspected database column
type TableColumnIntrospection struct {
	Name         string `json:"name"`
	DataType     string `json:"dataType"`
	IsNullable   bool   `json:"isNullable"`
	IsPrimaryKey bool   `json:"isPrimaryKey"`
	IsForeignKey bool   `json:"isForeignKey"`
	ForeignTable string `json:"foreignTable,omitempty"`
	DefaultValue string `json:"defaultValue,omitempty"`
}

// TableIntrospectionResponse represents the result of introspecting a table for BO creation
type TableIntrospectionResponse struct {
	TableID         string                     `json:"tableId"`
	TableName       string                     `json:"tableName"`
	QualifiedPath   string                     `json:"qualifiedPath"`
	Columns         []TableColumnIntrospection `json:"columns"`
	SuggestedFields []FieldDefinition          `json:"suggestedFields"`
	SuggestedName   string                     `json:"suggestedName"`
	SuggestedKey    string                     `json:"suggestedKey"`
}

// BODeltaFieldDiff represents the diff of a field between Core and Custom
type BODeltaFieldDiff struct {
	FieldKey    string                 `json:"fieldKey"`
	FieldName   string                 `json:"fieldName"`
	Status      string                 `json:"status"` // INHERITED, OVERRIDDEN, CUSTOM_ADDED, CUSTOM_REMOVED
	CoreField   *FieldDefinition       `json:"coreField,omitempty"`
	CustomField *FieldDefinition       `json:"customField,omitempty"`
	Overrides   map[string]interface{} `json:"overrides,omitempty"`
}

// BODeltaResponse represents the Workday-style delta comparison between tenant BO and gold copy Core BO
type BODeltaResponse struct {
	BOID            string              `json:"boId"`
	Key             string              `json:"key"`
	Name            string              `json:"name"`
	DisplayName     string              `json:"displayName"`
	IsCore          bool                `json:"isCore"`
	CoreID          string              `json:"coreId,omitempty"`
	FieldsDelta     []BODeltaFieldDiff  `json:"fieldsDelta"`
	InheritedCount  int                 `json:"inheritedCount"`
	OverriddenCount int                 `json:"overriddenCount"`
	CustomCount     int                 `json:"customCount"`
}

// -----------------------------------------------------------------------------
// AI Assistant & Copilot Models
// -----------------------------------------------------------------------------

type SynthesizedCalculatedField struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Type        string `json:"type"`
	Formula     string `json:"formula"`
	Description string `json:"description"`
}

type SynthesizedRule struct {
	RuleName    string `json:"ruleName"`
	Description string `json:"description"`
	Severity    string `json:"severity"` // ERROR, WARNING, INFO
	Field       string `json:"field,omitempty"`
	Script      string `json:"script"` // Starlark or expression
}

type BOAISynthesizeRequest struct {
	Prompt       string `json:"prompt"` // Natural language domain description or requirements
	TableID      string `json:"tableId,omitempty"`
	TableName    string `json:"tableName,omitempty"`
	Category     string `json:"category,omitempty"`
	IncludeRules bool   `json:"includeRules"`
	IncludeCalc  bool   `json:"includeCalculatedFields"`
}

type BOAISynthesizeResponse struct {
	SuggestedKey              string                       `json:"suggestedKey"`
	SuggestedName             string                       `json:"suggestedName"`
	SuggestedDisplayName      string                       `json:"suggestedDisplayName"`
	Description               string                       `json:"description"`
	Category                  string                       `json:"category"`
	PrimaryKey                string                       `json:"primaryKey"`
	SuggestedDriverTable      string                       `json:"suggestedDriverTable,omitempty"`
	SuggestedFields           []FieldDefinition            `json:"suggestedFields"`
	SuggestedCalculatedFields []SynthesizedCalculatedField `json:"suggestedCalculatedFields,omitempty"`
	SuggestedRules            []SynthesizedRule            `json:"suggestedRules,omitempty"`
	Reasoning                 string                       `json:"reasoning"`
}

type BOAINLQRequest struct {
	BOIDOrKey string `json:"boIdOrKey"`
	Query     string `json:"query"` // Natural language query prompt
}

type NLQFilterItem struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

type BOAINLQResponse struct {
	QueryDef     map[string]interface{} `json:"queryDef"` // Compatible with boresolver.QueryDef
	Dimensions   []string               `json:"dimensions"`
	Measures     []string               `json:"measures"`
	Filters      []NLQFilterItem        `json:"filters"`
	SortBy       string                 `json:"sortBy,omitempty"`
	SortOrder    string                 `json:"sortOrder,omitempty"`
	Limit        int                    `json:"limit,omitempty"`
	GeneratedSQL string                 `json:"generatedSql"`
	Explanation  string                 `json:"explanation"`
}

type BOAIExplainDeltaRequest struct {
	BOIDOrKey string `json:"boIdOrKey"`
}

type BOAIExplainDeltaResponse struct {
	Summary           string   `json:"summary"`
	BreakingChanges   []string `json:"breakingChanges"`
	GovernanceRisks   []string `json:"governanceRisks"`
	SuggestedActions  []string `json:"suggestedActions"`
	ImpactScore       string   `json:"impactScore"` // LOW, MEDIUM, HIGH, CRITICAL
	MarkdownNarrative string   `json:"markdownNarrative"`
}

type BODataAnomaly struct {
	Field       string `json:"field"`
	AnomalyType string `json:"anomalyType"` // NULL_SPIKE, OUTLIER, FORMAT_DRIFT, DUPLICATE_KEY
	Severity    string `json:"severity"`    // LOW, MEDIUM, HIGH
	Description string `json:"description"`
	SampleCount int    `json:"sampleCount"`
}

type BOAIAnomalyDetectRequest struct {
	BOIDOrKey  string `json:"boIdOrKey"`
	SampleSize int    `json:"sampleSize,omitempty"`
}

type BOAIAnomalyDetectResponse struct {
	Anomalies        []BODataAnomaly `json:"anomalies"`
	DataQualityScore float64         `json:"dataQualityScore"` // 0 - 100
	Summary          string          `json:"summary"`
	Recommendations  []string        `json:"recommendations"`
}

// -----------------------------------------------------------------------------
// Workflow & Lifecycle Models
// -----------------------------------------------------------------------------

type BOWorkflowLifecycleStatus string

const (
	BOWorkflowStatusDraft      BOWorkflowLifecycleStatus = "DRAFT"
	BOWorkflowStatusInReview   BOWorkflowLifecycleStatus = "IN_REVIEW"
	BOWorkflowStatusApproved   BOWorkflowLifecycleStatus = "APPROVED"
	BOWorkflowStatusPublished  BOWorkflowLifecycleStatus = "PUBLISHED"
	BOWorkflowStatusDeprecated BOWorkflowLifecycleStatus = "DEPRECATED"
)

type BOEventTrigger struct {
	ID          string `json:"id"`
	Event       string `json:"event"`      // ON_CREATE, ON_UPDATE, ON_DELETE, ON_VALIDATION_FAILURE, ON_PUBLISH
	ActionType  string `json:"actionType"` // WORKFLOW, WEBHOOK, NOTIFICATION, RECALCULATE
	Target      string `json:"target"`     // Workflow ID or Webhook URL
	Enabled     bool   `json:"enabled"`
	Description string `json:"description,omitempty"`
}

type BOPromotionProposal struct {
	ID              string                 `json:"id"`
	SourceTenantID  string                 `json:"sourceTenantId"`
	BOKey           string                 `json:"boKey"`
	ProposedChanges map[string]interface{} `json:"proposedChanges"`
	Status          string                 `json:"status"` // PENDING, APPROVED, REJECTED
	ReviewerNote    string                 `json:"reviewerNote,omitempty"`
	CreatedAt       string                 `json:"createdAt"`
	CreatedBy       string                 `json:"createdBy"`
}

type BOWorkflowActionRequest struct {
	Action       string               `json:"action"` // SUBMIT_REVIEW, APPROVE, PUBLISH, DEPRECATE, PROPOSE_PROMOTION, ADD_TRIGGER, DELETE_TRIGGER
	ReviewerNote string               `json:"reviewerNote,omitempty"`
	Trigger      *BOEventTrigger      `json:"trigger,omitempty"`
	Proposal     *BOPromotionProposal `json:"proposal,omitempty"`
}

type BOWorkflowExecution struct {
	ID          string `json:"id"`
	Workflow    string `json:"workflow"`
	TriggeredBy string `json:"triggeredBy"`
	Status      string `json:"status"` // RUNNING, COMPLETED, FAILED
	StartTime   string `json:"startTime"`
	EndTime     string `json:"endTime,omitempty"`
	Error       string `json:"error,omitempty"`
}

type BOWorkflowStatusResponse struct {
	BOID             string                    `json:"boId"`
	Key              string                    `json:"key"`
	LifecycleStatus  BOWorkflowLifecycleStatus `json:"lifecycleStatus"`
	IsCore           bool                      `json:"isCore"`
	PendingProposals []BOPromotionProposal     `json:"pendingProposals"`
	EventTriggers    []BOEventTrigger          `json:"eventTriggers"`
	RecentExecutions []BOWorkflowExecution     `json:"recentExecutions"`
}

// ============================================================================
// FEATURE 2: BINDING-AWARE DYNAMIC SCOPE & AUTO-DISCOVERY MODELS
// ============================================================================

type BOFieldEligibilityLevel string

const (
	EligibilityDirect     BOFieldEligibilityLevel = "DIRECT"
	EligibilityRelated    BOFieldEligibilityLevel = "RELATED"
	EligibilityCalculated BOFieldEligibilityLevel = "CALCULATED"
	EligibilityManual     BOFieldEligibilityLevel = "MANUAL"
)

type BOFieldResolutionStatus string

const (
	ResolutionResolved   BOFieldResolutionStatus = "RESOLVED"
	ResolutionUnresolved BOFieldResolutionStatus = "UNRESOLVED"
	ResolutionBlocked    BOFieldResolutionStatus = "BLOCKED"
)

type BOFieldEligibilityItem struct {
	FieldKey         string                  `json:"fieldKey"`
	FieldName        string                  `json:"fieldName"`
	DisplayName      string                  `json:"displayName"`
	DataType         string                  `json:"dataType"`
	Role             FieldRole               `json:"role"`
	EligibilityLevel BOFieldEligibilityLevel `json:"eligibilityLevel"`
	ResolutionStatus BOFieldResolutionStatus `json:"resolutionStatus"`
	ResolutionPath   string                  `json:"resolutionPath"` // e.g. "driving_node -> columns -> MAPS_TO"
	PhysicalTable    string                  `json:"physicalTable"`
	PhysicalColumn   string                  `json:"physicalColumn"`
	MissingInputs    []string                `json:"missingInputs,omitempty"`
	GateReason       string                  `json:"gateReason,omitempty"`
}

type BOScopeDiscoveryResponse struct {
	BOID             string                   `json:"boId"`
	DrivingNodeID    string                   `json:"drivingNodeId"`
	DrivingTableName string                   `json:"drivingTableName"`
	TotalDiscovered  int                      `json:"totalDiscovered"`
	DirectCount      int                      `json:"directCount"`
	RelatedCount     int                      `json:"relatedCount"`
	CalculatedCount  int                      `json:"calculatedCount"`
	ManualCount      int                      `json:"manualCount"`
	EligibleFields   []BOFieldEligibilityItem `json:"eligibleFields"`
	IsPublishReady   bool                     `json:"isPublishReady"`
	BlockingIssues   []string                 `json:"blockingIssues,omitempty"`
}

type BOPublishGateValidationResponse struct {
	BOID               string                   `json:"boId"`
	CanPublish         bool                     `json:"canPublish"`
	UnresolvedFields   []BOFieldEligibilityItem `json:"unresolvedFields"`
	MissingDependencies []string                `json:"missingDependencies"`
	GateSummary        string                   `json:"gateSummary"`
}

// ============================================================================
// FEATURE 3: POLYMORPHIC MULTI-BACKEND BINDINGS & STORAGE TIERING
// ============================================================================

type StorageTier string

const (
	StorageTier1Postgres     StorageTier = "TIER_1_POSTGRES"     // Control Plane / OLTP
	StorageTier2StarRocks    StorageTier = "TIER_2_STARROCKS"    // Hot Data Plane
	StorageTier3Iceberg      StorageTier = "TIER_3_ICEBERG"      // Cold Historical Data Plane
	StorageTierAPIFederation StorageTier = "API_FEDERATION"      // Parameterized REST/OpenAPI
)

type BindingRequirement string

const (
	BindingRequirementRequired        BindingRequirement = "REQUIRED"
	BindingRequirementOptional        BindingRequirement = "OPTIONAL"
	BindingRequirementBackendSpecific BindingRequirement = "BACKEND_SPECIFIC"
)

type MultiBackendBinding struct {
	ID                 string             `json:"id"`
	StorageTier        StorageTier        `json:"storageTier"`
	BackendName        string             `json:"backendName"` // postgres, starrocks, iceberg, salesforce, etc.
	DatasourceID       string             `json:"datasourceId"`
	PhysicalTarget     string             `json:"physicalTarget"` // schema.table or endpoint
	Requirement        BindingRequirement `json:"requirement"`
	IsActive           bool               `json:"isActive"`
	CoveragePercentage float64            `json:"coveragePercentage"` // % of BO fields mapped in this tier
}

type BOMultiBackendConfiguration struct {
	BOID          string                `json:"boId"`
	ActiveTier    StorageTier           `json:"activeTier"`
	Bindings      []MultiBackendBinding `json:"bindings"`
	WatermarkDate *time.Time            `json:"watermarkDate,omitempty"` // Seam date separating Hot vs Cold tier
}

// ============================================================================
// FEATURE 6: AI COGNITIVE FABRIC & GRAPHRAG MODELS
// ============================================================================

type GraphRAGContextRequest struct {
	BOIDOrKey    string `json:"boIdOrKey"`
	UserQuery    string `json:"userQuery"`
	IncludeEdges bool   `json:"includeEdges"`
	MaxDepth     int    `json:"maxDepth"`
}

type GraphRAGNode struct {
	ID          string                 `json:"id"`
	NodeType    string                 `json:"nodeType"` // BUSINESS_OBJECT, SEMANTIC_TERM, CLASSIFICATION, SYNONYM
	Name        string                 `json:"name"`
	DisplayName string                 `json:"displayName"`
	Description string                 `json:"description"`
	Similarity  float64                `json:"similarity,omitempty"`
	Properties  map[string]interface{} `json:"properties,omitempty"`
}

type GraphRAGContextResponse struct {
	BOKey          string         `json:"boKey"`
	ResolvedIntent string         `json:"resolvedIntent"`
	MatchedNodes   []GraphRAGNode `json:"matchedNodes"`
	TenantScoped   bool           `json:"tenantScoped"`
	PromptContext  string         `json:"promptContext"`
}

type UisceASTNode struct {
	NodeType   string                 `json:"nodeType"` // SELECT, FILTER, JOIN, AGGREGATE, WATERMARK_UNION
	Attributes map[string]interface{} `json:"attributes"`
	Children   []UisceASTNode         `json:"children,omitempty"`
}

type UisceSemanticAST struct {
	Version      string         `json:"version"`
	RootNode     UisceASTNode   `json:"rootNode"`
	TenantID     string         `json:"tenantId"`
	Dialects     []string       `json:"dialects"`
	GeneratedSQL map[string]string `json:"generatedSql,omitempty"` // dialect -> sql
}

type SchemaDriftProposal struct {
	ProposalID       string    `json:"proposalId"`
	BOID             string    `json:"boId"`
	DriftType        string    `json:"driftType"` // COLUMN_RENAME, TYPE_CHANGE, COLUMN_DROPPED
	SourceColumn     string    `json:"sourceColumn"`
	TargetColumn     string    `json:"targetColumn"`
	ConfidenceScore  float64   `json:"confidenceScore"` // > 0.95
	AutoRepairScript string    `json:"autoRepairScript"`
	Status           string    `json:"status"` // PENDING, APPLIED, REJECTED
	DetectedAt       time.Time `json:"detectedAt"`
}

// ============================================================================
// FEATURE 7: AUTOMATED LIFECYCLE, CI/CD GATEWAYS, & ARTIFACTS
// ============================================================================

type ImpactedAsset struct {
	AssetID      string `json:"assetId"`
	AssetName    string `json:"assetName"`
	AssetType    string `json:"assetType"` // BUSINESS_OBJECT, SEMANTIC_VIEW, VALIDATION_RULE, DASHBOARD, CUBE_MODEL
	ImpactLevel  string `json:"impactLevel"` // CRITICAL, HIGH, MEDIUM, LOW
	Relationship string `json:"relationship"` // USES_INPUT, MAPS_TO, FEEDS_INTO, BO_RELATIONSHIP
	Details      string `json:"details"`
}

type BOLineageImpactSimulationRequest struct {
	BOIDOrKey       string                 `json:"boIdOrKey"`
	ProposedChanges map[string]interface{} `json:"proposedChanges"`
}

type BOLineageImpactSimulationResponse struct {
	BOID             string          `json:"boId"`
	TotalImpacted    int             `json:"totalImpacted"`
	HighestSeverity  string          `json:"highestSeverity"` // CRITICAL, HIGH, MEDIUM, LOW
	IsBreakingChange bool            `json:"isBreakingChange"`
	ImpactedAssets   []ImpactedAsset `json:"impactedAssets"`
	BlastRadiusScore float64         `json:"blastRadiusScore"` // 0 - 100
	SimulationReport string          `json:"simulationReport"`
}

type BOArtifactGenerationResponse struct {
	BOID           string `json:"boId"`
	BOKey          string `json:"boKey"`
	OpenAPISpecJSON string `json:"openApiSpecJson"`
	CubeJSSchemaJS  string `json:"cubeJsSchemaJs"`
	StarRocksMVDDL  string `json:"starRocksMvDdl"`
	RESTEndpointURL string `json:"restEndpointUrl"`
}

// ============================================================================
// PILLAR 3: PREDICTIVE COST EVALUATION & GATEKEEPER MODELS
// ============================================================================

type QueryCostBand string

const (
	CostBandLow       QueryCostBand = "LOW"
	CostBandModerate  QueryCostBand = "MODERATE"
	CostBandExpensive QueryCostBand = "EXPENSIVE"
	CostBandForbidden QueryCostBand = "FORBIDDEN"
)

type BOQueryCostEvaluationRequest struct {
	BOIDOrKey       string            `json:"boIdOrKey"`
	SelectedFields  []string          `json:"selectedFields"`
	Filters         []BORecordFilter  `json:"filters,omitempty"`
	EstimatedLimit  int               `json:"estimatedLimit,omitempty"`
	TargetDialect   string            `json:"targetDialect,omitempty"`
}


type BOQueryCostEvaluationResponse struct {
	ComplexityScore       float64       `json:"complexityScore"` // 0 - 100
	CostBand              QueryCostBand `json:"costBand"`        // LOW, MODERATE, EXPENSIVE, FORBIDDEN
	IsForbidden           bool          `json:"isForbidden"`
	EstimatedRowsScanned  int64         `json:"estimatedRowsScanned"`
	EstimatedDurationMs   int64         `json:"estimatedDurationMs"`
	RequiresPartitionScan bool          `json:"requiresPartitionScan"`
	Violations            []string      `json:"violations,omitempty"`
	PreAggregationTips    []string      `json:"preAggregationTips,omitempty"`
	SuggestedMaterializedView string    `json:"suggestedMaterializedView,omitempty"`
}

// ============================================================================
// PILLARS 1 & 5: DRIFT REPAIR INBOX & FINANCIAL DATA QUALITY SENTINEL
// ============================================================================

type BODriftRepairPatchRequest struct {
	BOIDOrKey  string `json:"boIdOrKey"`
	ProposalID string `json:"proposalId"`
	Action     string `json:"action"` // APPROVE, REJECT
	Note       string `json:"note,omitempty"`
}

type BODriftRepairPatchResponse struct {
	ProposalID string `json:"proposalId"`
	Status     string `json:"status"` // APPLIED, REJECTED
	Message    string `json:"message"`
}

type FinancialPatternResult struct {
	FieldName    string  `json:"fieldName"`
	PatternType  string  `json:"patternType"` // ISO_6166_ISIN, CUSIP_MOD10, SEDOL_MOD10, ISO_17442_LEI
	SampleCount  int     `json:"sampleCount"`
	ValidCount   int     `json:"validCount"`
	InvalidCount int     `json:"invalidCount"`
	PassRate     float64 `json:"passRate"` // 0 - 100%
	SampleErrors []string `json:"sampleErrors,omitempty"`
}

type BODataQualitySentinelResponse struct {
	BOID                 string                   `json:"boId"`
	SampleStrategy       string                   `json:"sampleStrategy"` // TABLESAMPLE SYSTEM (0.1) LIMIT 500
	TotalSampledRows     int                      `json:"totalSampledRows"`
	OverallQualityScore  float64                  `json:"overallQualityScore"` // 0 - 100
	DistinctRatios       map[string]float64       `json:"distinctRatios"`
	NullDrift            map[string]float64       `json:"nullDrift"`
	FinancialVerifications []FinancialPatternResult `json:"financialVerifications"`
	DriftProposals       []SchemaDriftProposal    `json:"driftProposals"`
	SentinelSummary      string                   `json:"sentinelSummary"`
}

type LakehouseMaintenanceReport struct {
	TenantID            string    `json:"tenantId"`
	Table               string    `json:"table"`
	CompactedFilesCount int       `json:"compactedFilesCount"`
	BytesCompacted      int64     `json:"bytesCompacted"`
	ManifestsRewritten  int       `json:"manifestsRewritten"`
	SnapshotsExpired    int       `json:"snapshotsExpired"`
	DurationMs          int64     `json:"durationMs"`
	Status              string    `json:"status"` // COMPLETED, SKIPPED, FAILED
	ExecutedAt          time.Time `json:"executedAt"`
}





