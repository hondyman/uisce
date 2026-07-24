// GraphQL model types - isolated to avoid import cycles
package models

import (
	"time"

	"github.com/google/uuid"
)

// These are placeholder types - actual types should be defined here
// to avoid importing the root backend package

type ChangeEvent string

type Entity struct {
	ID uuid.UUID
}

type Position struct {
	ID uuid.UUID
}

type EntityChangedSubscription struct {
	Event     ChangeEvent `json:"event"`
	Entity    *Entity     `json:"entity"`
	Timestamp time.Time   `json:"timestamp"`
}

type PositionChangedSubscription struct {
	Event     ChangeEvent `json:"event"`
	Position  *Position   `json:"position"`
	Timestamp time.Time   `json:"timestamp"`
}

// Placeholder GraphQL types to satisfy generated resolvers. These can be expanded with real fields as schema evolves.

// Ownership / entity types
type CreateEntityInput struct{}
type CreateEntityPayload struct{}
type UpdateEntityInput struct{}
type UpdateEntityPayload struct{}
type DeleteEntityPayload struct{}
type CreatePositionInput struct{}
type CreatePositionPayload struct{}
type UpdatePositionInput struct{}
type UpdatePositionPayload struct{}
type ClosePositionPayload struct{}
type ImportModelTypesInput struct{}
type ImportModelTypesPayload struct{}
type EntityFilter struct{}
type EntityOrderBy struct{}
type EntityAggregate struct{}
type OwnershipNode struct{}
type ModelTypeDefinition struct{}
type HierarchyRule struct{}
type PortfolioMetrics struct{}

// Semantic layer types
type GenerateCoreModelInput struct{}
type SemanticModel struct{}
type GenerateCoreViewInput struct{}
type SemanticView struct{}
type CreateCustomModelInput struct{}
type CreateCustomViewInput struct{}
type ApplyRelationshipSuggestionInput struct{}
type RelationshipSuggestion struct{}
type TraverseGraphInput struct{}
type ObjectGraphPath struct{}
type SemanticAsset struct{}
type RelationshipSuggestionList struct{}
type ObjectGraphNode struct{}

// AI suggest types
type AISuggestRuleInput struct{}
type AISuggestedRule struct{}
type LogTermFeedbackInput struct{}

// Validation rule scenario types
type CreateRuleScenarioInput struct{}
type RuleScenario struct{}
type SaveScenarioVersionInput struct{}
type RuleScenarioVersion struct{}
type RunScenarioInput struct{}
type RuleTestRun struct{}
