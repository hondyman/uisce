package relationships

import (
	"context"

	"github.com/google/uuid"
)

type RelationshipType string

const (
	RelTypeOneToMany  RelationshipType = "one_to_many"
	RelTypeManyToOne  RelationshipType = "many_to_one"
	RelTypeManyToMany RelationshipType = "many_to_many"
)

type RelationshipSuggestion struct {
	ID          uuid.UUID        `json:"id"`
	SourceBO    string           `json:"source_bo"`
	TargetBO    string           `json:"target_bo"`
	Type        RelationshipType `json:"type"`
	Confidence  float64          `json:"confidence"`
	Evidence    []string         `json:"evidence"`
	Description string           `json:"description"`
}

type RelationshipDiscovery struct {
	// Integration with query logs, planner stats, data analysis
}

func NewRelationshipDiscovery() *RelationshipDiscovery {
	return &RelationshipDiscovery{}
}

func (d *RelationshipDiscovery) DiscoverRelationships(ctx context.Context) ([]RelationshipSuggestion, error) {
	suggestions := make([]RelationshipSuggestion, 0)

	// Mock: Generate sample suggestions
	// Real: Analyze query patterns, co-occurrence, key patterns, pre-agg definitions

	suggestions = append(suggestions, RelationshipSuggestion{
		ID:         uuid.New(),
		SourceBO:   "Account",
		TargetBO:   "Position",
		Type:       RelTypeOneToMany,
		Confidence: 0.92,
		Evidence: []string{
			"account_id frequently joins with position.account_id (1,245 queries/day)",
			"Referential integrity: 99.8% of positions have valid account_id",
			"Co-occurrence in 8 pages and 3 APIs",
		},
		Description: "Account.account_id frequently joins with Position.account_id; suggest relationship Account hasMany Positions.",
	})

	suggestions = append(suggestions, RelationshipSuggestion{
		ID:         uuid.New(),
		SourceBO:   "Position",
		TargetBO:   "Instrument",
		Type:       RelTypeManyToOne,
		Confidence: 0.88,
		Evidence: []string{
			"instrument_id appears in Position and matches Instrument.id",
			"Used together in 5 pre-aggregations",
			"High cardinality match (95% coverage)",
		},
		Description: "Position.instrument_id matches Instrument.id; suggest relationship Position belongsTo Instrument.",
	})

	suggestions = append(suggestions, RelationshipSuggestion{
		ID:         uuid.New(),
		SourceBO:   "Trade",
		TargetBO:   "Account",
		Type:       RelTypeManyToOne,
		Confidence: 0.75,
		Evidence: []string{
			"Implicit join pattern in 3 workflows",
			"account_id field present in Trade",
		},
		Description: "Trade.account_id suggests relationship to Account.",
	})

	return suggestions, nil
}
