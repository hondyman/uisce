// GraphQL model types - moved to gqlmodels to avoid import cycles

package gqlmodels

import (
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/google/uuid"
)

type AISuggestRuleInput struct {
	TenantID      uuid.UUID  `json:"tenantId"`
	Entity        string     `json:"entity"`
	Intent        string     `json:"intent"`
	TargetField   *string    `json:"targetField,omitempty"`
	ContextRuleID *uuid.UUID `json:"contextRuleId,omitempty"`
}

type AISuggestedRule struct {
	ConditionJSON *string `json:"conditionJson,omitempty"`
	// StarlarkSrc removed
	Description           string      `json:"description"`
	Severity              string      `json:"severity"`
	InheritModeSuggestion string      `json:"inheritModeSuggestion"`
	ConflictsWith         []uuid.UUID `json:"conflictsWith"`
	TestFailureRate       float64     `json:"testFailureRate"`
	RuntimeOk             bool        `json:"runtimeOk"`
}

type ApplyRelationshipSuggestionInput struct {
	SuggestionID uuid.UUID `json:"suggestionId"`
}

// Payload for closePosition
type ClosePositionPayload struct {
	Success  bool      `json:"success"`
	Position *Position `json:"position,omitempty"`
	Errors   []string  `json:"errors"`
}

// Input for creating an attribute
type CreateAttributeInput struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type CreateCustomModelInput struct {
	EntityID   uuid.UUID `json:"entityId"`
	ModelName  string    `json:"modelName"`
	Expression string    `json:"expression"`
	SourceKeys []string  `json:"sourceKeys"`
}

type CreateCustomViewInput struct {
	EntityID   uuid.UUID `json:"entityId"`
	ViewName   string    `json:"viewName"`
	Expression string    `json:"expression"`
	SourceKeys []string  `json:"sourceKeys"`
}

// Input for creating an entity
type CreateEntityInput struct {
	ModelType      string                  `json:"modelType"`
	OriginalName   string                  `json:"originalName"`
	DisplayName    *string                 `json:"displayName,omitempty"`
	OwnershipType  OwnershipType           `json:"ownershipType"`
	CurrencyFactor string                  `json:"currencyFactor"`
	Attributes     []*CreateAttributeInput `json:"attributes,omitempty"`
}

// Payload for createEntity
type CreateEntityPayload struct {
	Success bool     `json:"success"`
	Entity  *Entity  `json:"entity,omitempty"`
	Errors  []string `json:"errors"`
}

// Input for creating a position
type CreatePositionInput struct {
	OwnerID             uuid.UUID     `json:"ownerId"`
	OwnedID             uuid.UUID     `json:"ownedId"`
	OwnershipType       OwnershipType `json:"ownershipType"`
	OwnershipPercentage *float64      `json:"ownershipPercentage,omitempty"`
	Shares              *float64      `json:"shares,omitempty"`
	Value               *float64      `json:"value,omitempty"`
	InceptingDate       *string       `json:"inceptingDate,omitempty"`
}

// Payload for createPosition
type CreatePositionPayload struct {
	Success  bool      `json:"success"`
	Position *Position `json:"position,omitempty"`
	Errors   []string  `json:"errors"`
}

type CreateRuleScenarioInput struct {
	TenantID    uuid.UUID  `json:"tenantId"`
	BaseRuleID  *uuid.UUID `json:"baseRuleId,omitempty"`
	Name        string     `json:"name"`
	Description *string    `json:"description,omitempty"`
}

// Payload for deleteEntity
type DeleteEntityPayload struct {
	Success bool     `json:"success"`
	Errors  []string `json:"errors"`
}

// Represents a business entity in the ownership hierarchy.
// Examples: Household, Client, Trust, Financial Account, Stock, Bond, etc.
type Entity struct {
	// Unique identifier
	ID uuid.UUID `json:"id"`
	// Model type discriminator (e.g., 'household', 'stock', 'trust')
	// Determines the entity's role in the ownership hierarchy
	ModelType string `json:"modelType"`
	// Tenant ID for multi-tenant isolation.
	// Used implicitly in ABAC and RLS policies.
	TenantID uuid.UUID `json:"tenantId"`
	// Original name from source system
	// (e.g., legacy ticker, original account ID)
	OriginalName string `json:"originalName"`
	// User-friendly display name (e.g., "Growth Portfolio 2025")
	DisplayName *string `json:"displayName,omitempty"`
	// Currency factor for multi-currency support.
	// Typically base currency code (USD, EUR, GBP, etc.)
	CurrencyFactor string `json:"currencyFactor"`
	// Ownership model: PERCENT_BASED, SHARE_BASED, or VALUE_BASED
	OwnershipType OwnershipType `json:"ownershipType"`
	// Status: ACTIVE, INACTIVE, CLOSED, PENDING
	Status EntityStatus `json:"status"`
	// Is this entity actively managed?
	IsActive bool `json:"isActive"`
	// Dynamic attributes stored as JSONB.
	// Examples:
	//   - bond: { cusip, maturity_date, coupon_rate }
	//   - stock: { ticker, sector, market_cap }
	//   - household: { preferred_currency, domicile }
	Attributes []*EntityAttribute `json:"attributes"`
	// Positions where this entity is the owner.
	// Returns all active child positions.
	Owned []*Position `json:"owned"`
	// Positions where this entity is owned.
	// Returns all active parent positions.
	Owners []*Position `json:"owners"`
	// Creation timestamp (UTC)
	CreatedAt time.Time `json:"createdAt"`
	// Last update timestamp (UTC)
	UpdatedAt time.Time `json:"updatedAt"`
	// User who created this entity
	CreatedBy uuid.UUID `json:"createdBy"`
	// User who last updated this entity
	UpdatedBy *uuid.UUID `json:"updatedBy,omitempty"`
	// Soft delete timestamp (if entity is deleted)
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
}

// Aggregate statistics on entity query results
type EntityAggregate struct {
	Count            int32             `json:"count"`
	MaxCreatedAt     *time.Time        `json:"maxCreatedAt,omitempty"`
	MinCreatedAt     *time.Time        `json:"minCreatedAt,omitempty"`
	CountByModelType []*ModelTypeCount `json:"countByModelType"`
}

// Represents an attribute (key-value pair) of an entity.
// Stored in entity_attributes table, supports flexible typing.
type EntityAttribute struct {
	// Unique identifier
	ID uuid.UUID `json:"id"`
	// Parent entity ID
	EntityID uuid.UUID `json:"entityId"`
	// Attribute key (e.g., 'cusip', 'maturity_date', 'sector')
	Key string `json:"key"`
	// Attribute value – can be string, number, boolean, object, or array.
	// Type is inferred from JSON structure or validated against model metadata.
	Value string `json:"value"`
	// Optional: Human-readable description
	Description *string `json:"description,omitempty"`
	// Creation timestamp
	CreatedAt time.Time `json:"createdAt"`
	// Last update timestamp
	UpdatedAt time.Time `json:"updatedAt"`
}

// Filter by attribute (for JSONB entity_attributes)
type EntityAttributeFilter struct {
	// Attribute key
	Key *StringFilter `json:"key"`
	// Attribute value filter
	Value *JSONFilter `json:"value"`
}

// Filter entities by various criteria
type EntityFilter struct {
	// Model type (exact match or in list)
	ModelType *StringFilter `json:"modelType,omitempty"`
	// Ownership type
	OwnershipType *OwnershipTypeFilter `json:"ownershipType,omitempty"`
	// Entity status
	Status *EntityStatusFilter `json:"status,omitempty"`
	// Filter by attribute key-value pair
	Attribute *EntityAttributeFilter `json:"attribute,omitempty"`
	// Logical AND – all conditions must be true
	And []*EntityFilter `json:"AND,omitempty"`
	// Logical OR – at least one condition must be true
	Or []*EntityFilter `json:"OR,omitempty"`
	// Logical NOT
	Not *EntityFilter `json:"NOT,omitempty"`
}

// Order by clause
type EntityOrderBy struct {
	// Field to order by
	Field EntityOrderField `json:"field"`
	// Sort direction
	Direction *OrderDirection `json:"direction,omitempty"`
}

// Filter by entity status
type EntityStatusFilter struct {
	Eq *EntityStatus  `json:"eq,omitempty"`
	In []EntityStatus `json:"in,omitempty"`
}

type GenerateCoreModelInput struct {
	EntityID   uuid.UUID `json:"entityId"`
	ModelName  string    `json:"modelName"`
	SourceKeys []string  `json:"sourceKeys"`
}

type GenerateCoreViewInput struct {
	EntityID        uuid.UUID `json:"entityId"`
	ViewName        string    `json:"viewName"`
	SelectedColumns []string  `json:"selectedColumns"`
}

// Hierarchy rule: defines allowed parent → child relationships.
// Used by ABAC and entity creation validators.
type HierarchyRule struct {
	// Unique identifier
	ID uuid.UUID `json:"id"`
	// Parent model type
	ParentModelType string `json:"parentModelType"`
	// Child model type
	ChildModelType string `json:"childModelType"`
	// Is this relationship allowed?
	Allowed bool `json:"allowed"`
	// Allowed ownership types for this relationship
	OwnershipTypes []OwnershipType `json:"ownershipTypes"`
	// Maximum children of this type (null = unlimited)
	MaxChildren *int32 `json:"maxChildren,omitempty"`
	// Minimum children of this type (null = no minimum)
	MinChildren *int32 `json:"minChildren,omitempty"`
	// Description
	Description *string `json:"description,omitempty"`
	// Can child have only this parent type?
	IsExclusive bool `json:"isExclusive"`
	// Creation timestamp
	CreatedAt time.Time `json:"createdAt"`
}

// Holding value by asset type
type HoldingByType struct {
	ModelType          string  `json:"modelType"`
	Count              int32   `json:"count"`
	MarketValue        float64 `json:"marketValue"`
	CostBasis          float64 `json:"costBasis"`
	UnrealizedGainLoss float64 `json:"unrealizedGainLoss"`
	ReturnPct          float64 `json:"returnPct"`
}

// Input for bulk importing model types
type ImportModelTypesInput struct {
	JSONPayload string `json:"jsonPayload"`
}

// Payload for importModelTypes
type ImportModelTypesPayload struct {
	Success       bool     `json:"success"`
	ImportedCount int32    `json:"importedCount"`
	Errors        []string `json:"errors"`
}

// Filter JSON values (JSONB)
type JSONFilter struct {
	// Exact match
	Eq *string `json:"eq,omitempty"`
	// Contains (for objects/arrays)
	Contains *string `json:"contains,omitempty"`
	// Greater than (for numeric)
	Gt *float64 `json:"gt,omitempty"`
	// Greater than or equal
	Gte *float64 `json:"gte,omitempty"`
	// Less than
	Lt *float64 `json:"lt,omitempty"`
	// Less than or equal
	Lte *float64 `json:"lte,omitempty"`
	// In list
	In []string `json:"in,omitempty"`
}

type LogTermFeedbackInput struct {
	TenantID     uuid.UUID  `json:"tenantId"`
	DatasourceID uuid.UUID  `json:"datasourceId"`
	TermID       uuid.UUID  `json:"termId"`
	NodeID       *uuid.UUID `json:"nodeId,omitempty"`
	SuggestionID string     `json:"suggestionId"`
	Action       string     `json:"action"`
	Reason       *string    `json:"reason,omitempty"`
	OldTermID    *uuid.UUID `json:"oldTermId,omitempty"`
	Features     string     `json:"features"`
}

// Metadata about a suggested attribute for a model type
type ModelTypeAttribute struct {
	// Attribute key
	Key string `json:"key"`
	// Expected value type:
	// string, date, numeric, boolean, enum, object, array
	ValueType string `json:"valueType"`
	// Is this attribute required?
	IsRequired bool `json:"isRequired"`
	// Is this attribute searchable/filterable?
	IsSearchable bool `json:"isSearchable"`
	// Priority/display order (higher = more important)
	Priority *int32 `json:"priority,omitempty"`
	// Description
	Description *string `json:"description,omitempty"`
	// Optional validation rules (JSON schema fragment)
	ValidationRule *string `json:"validationRule,omitempty"`
}

// Count of entities by model type
type ModelTypeCount struct {
	ModelType string `json:"modelType"`
	Count     int32  `json:"count"`
}

// Metadata about a model type in the Addepar system.
// Defines entity types, hierarchy rules, and suggested attributes.
type ModelTypeDefinition struct {
	// Internal model type code (discriminator)
	ModelType string `json:"modelType"`
	// Human-readable display name
	DisplayName string `json:"displayName"`
	// Default ownership model for this type
	OwnershipType OwnershipType `json:"ownershipType"`
	// Description and usage notes
	Description *string `json:"description,omitempty"`
	// Is this type hierarchical (can own other types)?
	IsHierarchical bool `json:"isHierarchical"`
	// Level in hierarchy (0=root, 1=containers, 2=subcontainers, 3=assets)
	HierarchyLevel *int32 `json:"hierarchyLevel,omitempty"`
	// Suggested attributes for this type
	SuggestedAttributes []*ModelTypeAttribute `json:"suggestedAttributes"`
	// Allowed parent types for this entity.
	// Empty if this is a top-level or leaf type.
	AllowedParents []string `json:"allowedParents"`
	// Allowed child types for this entity.
	// Empty if this is a leaf node.
	AllowedChildren []string `json:"allowedChildren"`
	// Can this type have multiple parents?
	AllowsMultipleParents bool `json:"allowsMultipleParents"`
	// Maximum number of children allowed (null = unlimited)
	MaxChildren *int32 `json:"maxChildren,omitempty"`
}

type Mutation struct {
}

// ObjectGraphNode represents a node in the object graph
type ObjectGraphNode struct {
	ID        uuid.UUID   `json:"id"`
	Name      string      `json:"name"`
	Type      string      `json:"type"`
	LinksTo   []uuid.UUID `json:"linksTo"`
	LinksFrom []uuid.UUID `json:"linksFrom"`
}

// ObjectGraphPath represents a traversed path through the object graph
type ObjectGraphPath struct {
	Nodes []uuid.UUID `json:"nodes"`
	Path  string      `json:"path"`
}

// Recursive node for hierarchical ownership traversal.
// Represents an entity + its children at that depth level.
type OwnershipNode struct {
	// The entity at this node
	Entity *Entity `json:"entity"`
	// The position linking parent to this entity
	// (null if this is the root)
	Position *Position `json:"position,omitempty"`
	// Child nodes (entities owned by this entity).
	// Populated up to the requested depth.
	Children []*OwnershipNode `json:"children"`
	// Current depth in tree (0 = root)
	Depth int32 `json:"depth"`
	// Total child count at this node
	ChildCount int32 `json:"childCount"`
}

// Filter by ownership type
type OwnershipTypeFilter struct {
	Eq *OwnershipType  `json:"eq,omitempty"`
	In []OwnershipType `json:"in,omitempty"`
}

// Portfolio-level summary metrics
type PortfolioMetrics struct {
	// Root entity ID
	RootID uuid.UUID `json:"rootId"`
	// Snapshot date
	AsOf string `json:"asOf"`
	// Total market value across all holdings
	TotalMarketValue float64 `json:"totalMarketValue"`
	// Total cost basis
	TotalCostBasis float64 `json:"totalCostBasis"`
	// Unrealized gain/loss
	UnrealizedGainLoss float64 `json:"unrealizedGainLoss"`
	// Portfolio return percentage
	PortfolioReturnPct float64 `json:"portfolioReturnPct"`
	// Number of direct positions
	PositionCount int32 `json:"positionCount"`
	// Number of underlying assets
	AssetCount int32 `json:"assetCount"`
	// Holdings breakdown by model type.
	// E.g., "STOCK: $500K, BOND: $300K, CASH: $200K"
	HoldingsByType []*HoldingByType `json:"holdingsByType"`
	// Top 10 positions by market value.
	// Useful for pie charts.
	TopHoldings []*Position `json:"topHoldings"`
}

// Represents an ownership relationship between two entities.
// Examples:
//   - Household owns Person_node (PERCENT_BASED, 100%)
//   - Trust owns Real_estate (VALUE_BASED, $500K)
//   - Fund owns Stock (SHARE_BASED, 1000 shares)
type Position struct {
	// Unique identifier
	ID uuid.UUID `json:"id"`
	// Entity that owns the position (owner)
	OwnerID uuid.UUID `json:"ownerId"`
	// Entity that is owned (child)
	OwnedID uuid.UUID `json:"ownedId"`
	// Resolved owner entity
	Owner *Entity `json:"owner"`
	// Resolved owned entity
	Owned *Entity `json:"owned"`
	// Ownership model: PERCENT_BASED, SHARE_BASED, or VALUE_BASED
	OwnershipType OwnershipType `json:"ownershipType"`
	// Ownership percentage (0-100).
	// Only populated for PERCENT_BASED positions.
	OwnershipPercentage *float64 `json:"ownershipPercentage,omitempty"`
	// Number of shares/units owned.
	// Only populated for SHARE_BASED positions.
	Shares *float64 `json:"shares,omitempty"`
	// Value in base currency.
	// Only populated for VALUE_BASED positions.
	Value *float64 `json:"value,omitempty"`
	// Average cost per unit (for calculations)
	AverageCostPerUnit *float64 `json:"averageCostPerUnit,omitempty"`
	// Average market price (for calculations)
	AverageMarketPrice *float64 `json:"averageMarketPrice,omitempty"`
	// Date position was opened
	InceptingDate *string `json:"inceptingDate,omitempty"`
	// Date position was closed (null if still open)
	TerminatingDate *string `json:"terminatingDate,omitempty"`
	// Current status: ACTIVE, INACTIVE, CLOSED, PENDING
	Status PositionStatus `json:"status"`
	// Is this position currently active?
	IsActive bool `json:"isActive"`
	// Creation timestamp
	CreatedAt time.Time `json:"createdAt"`
	// Last update timestamp
	UpdatedAt time.Time `json:"updatedAt"`
	// User who created this position
	CreatedBy uuid.UUID `json:"createdBy"`
}

// Filter by position status
type PositionStatusFilter struct {
	Eq *PositionStatus  `json:"eq,omitempty"`
	In []PositionStatus `json:"in,omitempty"`
}

type Query struct {
}

// RelationshipSuggestion represents an AI-generated relationship recommendation
type RelationshipSuggestion struct {
	ID               uuid.UUID         `json:"id"`
	TenantID         uuid.UUID         `json:"tenantId"`
	DatasourceID     uuid.UUID         `json:"datasourceId"`
	SourceEntityID   uuid.UUID         `json:"sourceEntityId"`
	TargetEntityID   uuid.UUID         `json:"targetEntityId"`
	Confidence       float64           `json:"confidence"`
	Rationale        *string           `json:"rationale,omitempty"`
	ScoringBreakdown *ScoringBreakdown `json:"scoringBreakdown"`
	Accepted         bool              `json:"accepted"`
	AcceptedAt       *time.Time        `json:"acceptedAt,omitempty"`
	CreatedAt        time.Time         `json:"createdAt"`
	UpdatedAt        time.Time         `json:"updatedAt"`
}

// RelationshipSuggestionList is a paginated list of suggestions
type RelationshipSuggestionList struct {
	Suggestions []*RelationshipSuggestion `json:"suggestions"`
	Count       int32                     `json:"count"`
}

type RuleScenario struct {
	ID          uuid.UUID              `json:"id"`
	TenantID    uuid.UUID              `json:"tenantId"`
	BaseRuleID  *uuid.UUID             `json:"baseRuleId,omitempty"`
	Name        string                 `json:"name"`
	Description *string                `json:"description,omitempty"`
	Status      string                 `json:"status"`
	Versions    []*RuleScenarioVersion `json:"versions"`
	CreatedAt   time.Time              `json:"createdAt"`
}

type RuleScenarioVersion struct {
	ID           uuid.UUID `json:"id"`
	Version      int32     `json:"version"`
	RuleSnapshot string    `json:"ruleSnapshot"`
	CreatedAt    time.Time `json:"createdAt"`
}

// RuleTestRun represents an execution of a rule against a sample dataset
type RuleTestRun struct {
	ID                uuid.UUID  `json:"id"`
	TenantID          uuid.UUID  `json:"tenantId"`
	ScenarioVersionID *uuid.UUID `json:"scenarioVersionId,omitempty"`
	Status            string     `json:"status"`
	SampleSize        int32      `json:"sampleSize"`
	FailureCount      int32      `json:"failureCount"`
	FailureRate       *float64   `json:"failureRate,omitempty"`
	StartedAt         time.Time  `json:"startedAt"`
	CompletedAt       *time.Time `json:"completedAt,omitempty"`
}

type RunScenarioInput struct {
	TenantID          uuid.UUID `json:"tenantId"`
	ScenarioVersionID uuid.UUID `json:"scenarioVersionId"`
	Entity            string    `json:"entity"`
	SampleSize        int32     `json:"sampleSize"`
	Filter            *string   `json:"filter,omitempty"`
}

type SaveScenarioVersionInput struct {
	ScenarioID uuid.UUID `json:"scenarioId"`
	RuleDraft  string    `json:"ruleDraft"`
}

// ScoringBreakdown contains individual signal scores for relationship confidence
type ScoringBreakdown struct {
	ForeignKeyPresence float64 `json:"foreignKeyPresence"`
	JoinFrequency      float64 `json:"joinFrequency"`
	NameSimilarity     float64 `json:"nameSimilarity"`
	TextSimilarity     float64 `json:"textSimilarity"`
	EdgeTypePrior      float64 `json:"edgeTypePrior"`
}

// Semantic Layer Schema - managing semantic models, views, and relationships
type SemanticAsset struct {
	ID               uuid.UUID  `json:"id"`
	TenantID         uuid.UUID  `json:"tenantId"`
	DatasourceID     uuid.UUID  `json:"datasourceId"`
	BusinessEntityID uuid.UUID  `json:"businessEntityId"`
	CoreModelID      *uuid.UUID `json:"coreModelId,omitempty"`
	CoreViewID       *uuid.UUID `json:"coreViewId,omitempty"`
	CustomModelID    *uuid.UUID `json:"customModelId,omitempty"`
	CustomViewID     *uuid.UUID `json:"customViewId,omitempty"`
	SourceTables     []string   `json:"sourceTables,omitempty"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

// SemanticModel represents a generated or custom model
type SemanticModel struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Description *string   `json:"description,omitempty"`
	SourceKeys  []string  `json:"sourceKeys,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

// SemanticView represents a generated or custom view
type SemanticView struct {
	ID              uuid.UUID `json:"id"`
	Name            string    `json:"name"`
	Type            string    `json:"type"`
	Description     *string   `json:"description,omitempty"`
	SelectedColumns []string  `json:"selectedColumns,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
}

// Filter by string (exact, contains, in list)
type StringFilter struct {
	// Exact match
	Eq *string `json:"eq,omitempty"`
	// Contains substring
	Contains *string `json:"contains,omitempty"`
	// Case-insensitive contains
	ContainsIgnoreCase *string `json:"containsIgnoreCase,omitempty"`
	// In list
	In []string `json:"in,omitempty"`
	// Not in list
	NotIn []string `json:"notIn,omitempty"`
	// Regex match (if DB supports)
	Regex *string `json:"regex,omitempty"`
}

type Subscription struct {
}

type TraverseGraphInput struct {
	StartNodeID uuid.UUID `json:"startNodeId"`
	DotPath     string    `json:"dotPath"`
}

// Input for updating an attribute
type UpdateAttributeInput struct {
	Key    string  `json:"key"`
	Value  *string `json:"value,omitempty"`
	Delete *bool   `json:"delete,omitempty"`
}

// Input for updating an entity
type UpdateEntityInput struct {
	DisplayName *string                 `json:"displayName,omitempty"`
	Status      *EntityStatus           `json:"status,omitempty"`
	Attributes  []*UpdateAttributeInput `json:"attributes,omitempty"`
}

// Payload for updateEntity
type UpdateEntityPayload struct {
	Success bool     `json:"success"`
	Entity  *Entity  `json:"entity,omitempty"`
	Errors  []string `json:"errors"`
}

// Input for updating a position
type UpdatePositionInput struct {
	OwnershipPercentage *float64 `json:"ownershipPercentage,omitempty"`
	Shares              *float64 `json:"shares,omitempty"`
	Value               *float64 `json:"value,omitempty"`
}

// Payload for updatePosition
type UpdatePositionPayload struct {
	Success  bool      `json:"success"`
	Position *Position `json:"position,omitempty"`
	Errors   []string  `json:"errors"`
}

// Change event type
type ChangeEvent string

const (
	ChangeEventCreated ChangeEvent = "CREATED"
	ChangeEventUpdated ChangeEvent = "UPDATED"
	ChangeEventDeleted ChangeEvent = "DELETED"
)

var AllChangeEvent = []ChangeEvent{
	ChangeEventCreated,
	ChangeEventUpdated,
	ChangeEventDeleted,
}

func (e ChangeEvent) IsValid() bool {
	switch e {
	case ChangeEventCreated, ChangeEventUpdated, ChangeEventDeleted:
		return true
	}
	return false
}

func (e ChangeEvent) String() string {
	return string(e)
}

func (e *ChangeEvent) UnmarshalGQL(v any) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = ChangeEvent(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid ChangeEvent", str)
	}
	return nil
}

func (e ChangeEvent) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

// Fields available for ordering
type EntityOrderField string

const (
	EntityOrderFieldModelType    EntityOrderField = "MODEL_TYPE"
	EntityOrderFieldOriginalName EntityOrderField = "ORIGINAL_NAME"
	EntityOrderFieldDisplayName  EntityOrderField = "DISPLAY_NAME"
	EntityOrderFieldCreatedAt    EntityOrderField = "CREATED_AT"
	EntityOrderFieldUpdatedAt    EntityOrderField = "UPDATED_AT"
)

var AllEntityOrderField = []EntityOrderField{
	EntityOrderFieldModelType,
	EntityOrderFieldOriginalName,
	EntityOrderFieldDisplayName,
	EntityOrderFieldCreatedAt,
	EntityOrderFieldUpdatedAt,
}

func (e EntityOrderField) IsValid() bool {
	switch e {
	case EntityOrderFieldModelType, EntityOrderFieldOriginalName, EntityOrderFieldDisplayName, EntityOrderFieldCreatedAt, EntityOrderFieldUpdatedAt:
		return true
	}
	return false
}

func (e EntityOrderField) String() string {
	return string(e)
}

func (e *EntityOrderField) UnmarshalGQL(v any) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = EntityOrderField(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid EntityOrderField", str)
	}
	return nil
}

func (e EntityOrderField) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

// Entity status
type EntityStatus string

const (
	EntityStatusActive   EntityStatus = "ACTIVE"
	EntityStatusInactive EntityStatus = "INACTIVE"
	EntityStatusClosed   EntityStatus = "CLOSED"
	EntityStatusPending  EntityStatus = "PENDING"
)

var AllEntityStatus = []EntityStatus{
	EntityStatusActive,
	EntityStatusInactive,
	EntityStatusClosed,
	EntityStatusPending,
}

func (e EntityStatus) IsValid() bool {
	switch e {
	case EntityStatusActive, EntityStatusInactive, EntityStatusClosed, EntityStatusPending:
		return true
	}
	return false
}

func (e EntityStatus) String() string {
	return string(e)
}

func (e *EntityStatus) UnmarshalGQL(v any) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = EntityStatus(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid EntityStatus", str)
	}
	return nil
}

func (e EntityStatus) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

// Sort direction
type OrderDirection string

const (
	OrderDirectionAsc  OrderDirection = "ASC"
	OrderDirectionDesc OrderDirection = "DESC"
)

var AllOrderDirection = []OrderDirection{
	OrderDirectionAsc,
	OrderDirectionDesc,
}

func (e OrderDirection) IsValid() bool {
	switch e {
	case OrderDirectionAsc, OrderDirectionDesc:
		return true
	}
	return false
}

func (e OrderDirection) String() string {
	return string(e)
}

func (e *OrderDirection) UnmarshalGQL(v any) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = OrderDirection(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid OrderDirection", str)
	}
	return nil
}

func (e OrderDirection) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

// Ownership model types
type OwnershipType string

const (
	// Percentage-based ownership (0-100%)
	OwnershipTypePercentBased OwnershipType = "PERCENT_BASED"
	// Share-based ownership (discrete units)
	OwnershipTypeShareBased OwnershipType = "SHARE_BASED"
	// Value-based ownership (USD amount)
	OwnershipTypeValueBased OwnershipType = "VALUE_BASED"
)

var AllOwnershipType = []OwnershipType{
	OwnershipTypePercentBased,
	OwnershipTypeShareBased,
	OwnershipTypeValueBased,
}

func (e OwnershipType) IsValid() bool {
	switch e {
	case OwnershipTypePercentBased, OwnershipTypeShareBased, OwnershipTypeValueBased:
		return true
	}
	return false
}

func (e OwnershipType) String() string {
	return string(e)
}

func (e *OwnershipType) UnmarshalGQL(v any) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = OwnershipType(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid OwnershipType", str)
	}
	return nil
}

func (e OwnershipType) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

// Position status
type PositionStatus string

const (
	PositionStatusActive   PositionStatus = "ACTIVE"
	PositionStatusInactive PositionStatus = "INACTIVE"
	PositionStatusClosed   PositionStatus = "CLOSED"
	PositionStatusPending  PositionStatus = "PENDING"
)

var AllPositionStatus = []PositionStatus{
	PositionStatusActive,
	PositionStatusInactive,
	PositionStatusClosed,
	PositionStatusPending,
}

func (e PositionStatus) IsValid() bool {
	switch e {
	case PositionStatusActive, PositionStatusInactive, PositionStatusClosed, PositionStatusPending:
		return true
	}
	return false
}

func (e PositionStatus) String() string {
	return string(e)
}

func (e *PositionStatus) UnmarshalGQL(v any) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = PositionStatus(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid PositionStatus", str)
	}
	return nil
}

func (e PositionStatus) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}
