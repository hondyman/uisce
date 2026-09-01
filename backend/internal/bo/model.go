package bo

import (
	"github.com/google/uuid"
)

type BindingContextRequest struct {
	TenantID      uuid.UUID `json:"tenantId"`
	BackendID     uuid.UUID `json:"backendId"`
	DrivingNodeID uuid.UUID `json:"drivingNodeId"`
}

type ColumnMappingOption struct {
	ColumnNodeID    uuid.UUID `json:"columnNodeId"`
	ColumnName      string    `json:"columnName"`
	TableName       string    `json:"tableName"`
	SourceType      string    `json:"sourceType"` // DIRECT | RELATED
	IsPrimarySource bool      `json:"isPrimarySource"`
}

type SuggestedBKTerm struct {
	TermNodeID uuid.UUID `json:"termNodeId"`
	TermKey    string    `json:"termKey"`
}

type DiscoveredPKColumn struct {
	ColumnNodeID    uuid.UUID        `json:"columnNodeId"`
	ColumnName      string           `json:"columnName"`
	SuggestedBKTerm *SuggestedBKTerm `json:"suggestedBkTerm,omitempty"`
}

type DiscoveredRelatedTable struct {
	TableNodeID  uuid.UUID `json:"tableNodeId"`
	TableName    string    `json:"tableName"`
	JoinEdgeType string    `json:"joinEdgeType"`
	JoinColumn   string    `json:"joinColumn"`
}

type EligibleSemanticTerm struct {
	TermNodeID   uuid.UUID             `json:"termNodeId"`
	TermKey      string                `json:"termKey"`
	TermName     string                `json:"termName"`
	TermType     string                `json:"termType"`
	SourceType   string                `json:"sourceType"` // DIRECT | RELATED | CALCULATED | MANUAL
	IdentityRole string                `json:"identityRole,omitempty"`
	DataType     string                `json:"dataType"`
	Aggregation  string                `json:"aggregationType"`
	Mappings     []ColumnMappingOption `json:"mappings"`
}

type CalculatedCandidateTerm struct {
	TermNodeID               uuid.UUID `json:"termNodeId"`
	TermKey                  string    `json:"termKey"`
	Dependencies             []string  `json:"dependencies"`
	AllDependenciesAvailable bool      `json:"allDependenciesAvailable"`
}

type BindingContextResponse struct {
	DrivingTable struct {
		NodeID            uuid.UUID            `json:"nodeId"`
		TableName         string               `json:"tableName"`
		PrimaryKeyColumns []DiscoveredPKColumn `json:"primaryKeyColumns"`
	} `json:"drivingTable"`
	RelatedTables   []DiscoveredRelatedTable  `json:"relatedTables"`
	EligibleTerms   []EligibleSemanticTerm    `json:"eligibleTerms"`
	CalculatedTerms []CalculatedCandidateTerm `json:"calculatedTerms"`
}

type SaveFieldPayload struct {
	TermNodeID         uuid.UUID  `json:"termNodeId"`
	FieldName          string     `json:"fieldName"`
	FieldRole          string     `json:"fieldRole"`
	AggregationType    string     `json:"aggregationType"`
	BindingRequirement string     `json:"bindingRequirement"`
	EligibilitySource  string     `json:"eligibilitySource"`
	SubtypeScope       string     `json:"subtypeScope"`
	SourceNodeID       *uuid.UUID `json:"sourceNodeId,omitempty"`
	SourceType         string     `json:"sourceType"`
	TransformationType string     `json:"transformationType"`
	TransformationSQL  *string    `json:"transformationSql,omitempty"`
	JSONPath           *string    `json:"jsonPath,omitempty"`
}

type SaveRelationshipPayload struct {
	ToBOID           uuid.UUID `json:"toBoId"`
	RelKey           string    `json:"relKey"`
	RelName          string    `json:"relName"`
	Cardinality      string    `json:"cardinality"`
	JoinType         string    `json:"joinType"`
	JoinConditionSQL string    `json:"joinConditionSql"`
}

type SaveBindingPayload struct {
	BackendID        uuid.UUID                 `json:"backendId"`
	BackendType      string                    `json:"backendType"`
	DrivingNodeID    uuid.UUID                 `json:"drivingNodeId"`
	IsDefault        bool                      `json:"isDefault"`
	TemporalOverride string                    `json:"temporalOverride"`
	BaseSQL          *string                   `json:"baseSql,omitempty"`
	Fields           []SaveFieldPayload        `json:"fields"`
	Relationships    []SaveRelationshipPayload `json:"relationships"`
}

type SaveBusinessObjectRequest struct {
	TenantID       uuid.UUID `json:"tenantId"`
	ModelID        uuid.UUID `json:"modelId"`
	BusinessObject struct {
		ID                     *uuid.UUID `json:"id,omitempty"`
		BOKey                  string     `json:"boKey"`
		BOName                 string     `json:"boName"`
		Description            *string    `json:"description,omitempty"`
		BOType                 string     `json:"boType"`
		ClassificationNodeID   uuid.UUID  `json:"classificationNodeId"`
		BusinessKeyNodeID      uuid.UUID  `json:"businessKeyNodeId"`
		SemanticIDNodeID       uuid.UUID  `json:"semanticIdNodeId"`
		GrainNodeID            uuid.UUID  `json:"grainNodeId"`
		STIDiscriminatorColumn *string    `json:"stiDiscriminatorColumn,omitempty"`
		ActiveSubtypeFilter    *string    `json:"activeSubtypeFilter,omitempty"`
	} `json:"businessObject"`
	Bindings []SaveBindingPayload `json:"bindings"`
	Publish  bool                 `json:"publish"`
}
