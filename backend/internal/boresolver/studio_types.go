package boresolver

import (
	"github.com/google/uuid"
)

// BindingContextRequest requests discovery metadata for a specific driving table
type BindingContextRequest struct {
	TenantID      uuid.UUID `json:"tenantId"`
	BackendID     uuid.UUID `json:"backendId"`
	DrivingNodeID uuid.UUID `json:"drivingNodeId"`
}

type ColumnMappingDescriptor struct {
	ColumnNodeID    uuid.UUID `json:"columnNodeId"`
	ColumnName      string    `json:"columnName"`
	TableName       string    `json:"tableName"`
	SourceType      string    `json:"sourceType"` // DIRECT, RELATED
	IsPrimarySource bool      `json:"isPrimarySource"`
}

type EligibleTermDescriptor struct {
	TermNodeID        uuid.UUID                 `json:"termNodeId"`
	TermKey           string                    `json:"termKey"`
	TermName          string                    `json:"termName"`
	TermType          string                    `json:"termType"`
	IdentityRole      string                    `json:"identityRole,omitempty"` // BUSINESS_KEY, SEMANTIC_ID
	SourceType        string                    `json:"sourceType"`             // DIRECT, RELATED, CALCULATED
	Mappings          []ColumnMappingDescriptor `json:"mappings"`
	CalculationInputs []string                  `json:"calculationInputs,omitempty"`
}

type RelatedTableDescriptor struct {
	TableNodeID  uuid.UUID `json:"tableNodeId"`
	TableName    string    `json:"tableName"`
	JoinEdgeType string    `json:"joinEdgeType"`
	JoinColumn   string    `json:"joinColumn"`
}

type BindingContextResponse struct {
	DrivingTable struct {
		NodeID            uuid.UUID `json:"nodeId"`
		TableName         string    `json:"tableName"`
		PrimaryKeyColumns []struct {
			ColumnNodeID    uuid.UUID `json:"columnNodeId"`
			ColumnName      string    `json:"columnName"`
			SuggestedBKTerm *struct {
				TermNodeID uuid.UUID `json:"termNodeId"`
				TermKey    string    `json:"termKey"`
			} `json:"suggestedBkTerm,omitempty"`
		} `json:"primaryKeyColumns"`
	} `json:"drivingTable"`
	RelatedTables []RelatedTableDescriptor `json:"relatedTables"`
	EligibleTerms []EligibleTermDescriptor `json:"eligibleTerms"`
}

// AtomicSaveBORequest captures the full single-screen submission payload
type AtomicSaveBORequest struct {
	TenantID       uuid.UUID `json:"tenantId"`
	ModelID        uuid.UUID `json:"modelId"`
	Publish        bool      `json:"publish"`
	BusinessObject struct {
		BOID                 *uuid.UUID `json:"boId,omitempty"`
		BOKey                string     `json:"boKey"`
		BOName               string     `json:"boName"`
		Description          string     `json:"description"`
		BOType               string     `json:"boType"` // ENTITY, FACT, DIMENSION
		ClassificationNodeID uuid.UUID  `json:"classificationNodeId"` // Level 3 Classification
		BusinessKeyNodeID    uuid.UUID  `json:"businessKeyNodeId"`
		SemanticIDNodeID     uuid.UUID  `json:"semanticIdNodeId"`
		GrainNodeID          uuid.UUID  `json:"grainNodeId"`
	} `json:"businessObject"`
	Bindings []struct {
		BindingID        *uuid.UUID `json:"bindingId,omitempty"`
		BackendID        uuid.UUID  `json:"backendId"`
		DrivingNodeID    uuid.UUID  `json:"drivingNodeId"`
		IsDefault        bool       `json:"isDefault"`
		TemporalOverride string     `json:"temporalOverride"`
		Fields           []struct {
			TermNodeID         uuid.UUID  `json:"termNodeId"`
			FieldName          string     `json:"fieldName"`
			FieldRole          string     `json:"fieldRole"`          // DIMENSION, MEASURE, ATTRIBUTE, KEY
			BindingRequirement string     `json:"bindingRequirement"`  // REQUIRED, OPTIONAL, BACKEND_SPECIFIC
			SourceNodeID       *uuid.UUID `json:"sourceNodeId,omitempty"`
			SourceType         string     `json:"sourceType"`          // COLUMN, EXPRESSION, FUNCTION
			TransformationType string     `json:"transformationType"`  // NONE, SQL, NORMALIZE
			TransformationSQL  *string    `json:"transformationSql,omitempty"`
			OverrideReason     *string    `json:"overrideReason,omitempty"`
		} `json:"fields"`
		Relationships []struct {
			ToBOID           uuid.UUID `json:"toBoId"`
			RelKey           string    `json:"relKey"`
			Cardinality      string    `json:"cardinality"`
			JoinType         string    `json:"joinType"`
			JoinConditionSQL string    `json:"joinConditionSql"`
		} `json:"relationships"`
	} `json:"bindings"`
}
