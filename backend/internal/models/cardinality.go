package models

import "strings"

// Cardinality is the shape of a relationship between two tables/entities.
// Values match the entity_relationship.cardinality CHECK constraint
// (backend/internal/migrations/006_relationship_discovery_schema.sql).
type Cardinality string

const (
	CardinalityOneToOne   Cardinality = "ONE_TO_ONE"
	CardinalityOneToMany  Cardinality = "ONE_TO_MANY"
	CardinalityManyToOne  Cardinality = "MANY_TO_ONE"
	CardinalityManyToMany Cardinality = "MANY_TO_MANY"
	CardinalityUnknown    Cardinality = "UNKNOWN"
)

// Display renders the frontend wire format used across the query builder,
// page designer and report designer ("1:1", "1:M", "M:1", "M:M").
func (c Cardinality) Display() string {
	switch c {
	case CardinalityOneToOne:
		return "1:1"
	case CardinalityOneToMany:
		return "1:M"
	case CardinalityManyToOne:
		return "M:1"
	case CardinalityManyToMany:
		return "M:M"
	default:
		return ""
	}
}

// Inverse flips a cardinality to the opposite direction's perspective
// (e.g. source->target ONE_TO_MANY becomes target->source MANY_TO_ONE).
func (c Cardinality) Inverse() Cardinality {
	switch c {
	case CardinalityOneToMany:
		return CardinalityManyToOne
	case CardinalityManyToOne:
		return CardinalityOneToMany
	case CardinalityOneToOne:
		return CardinalityOneToOne
	case CardinalityManyToMany:
		return CardinalityManyToMany
	default:
		return CardinalityUnknown
	}
}

// IsToMany reports whether the "many" side is on the target — the signal
// designers use to decide between an embedded/collection UI (page designer's
// RelatedObjectsPalette) and a single reference control.
func (c Cardinality) IsToMany() bool {
	return c == CardinalityOneToMany || c == CardinalityManyToMany
}

// ParseCardinality normalizes the many loose strings that have historically
// been written by different discovery paths ("one-to-many", "1:N", "1:M",
// lowercase snake case, etc.) into the canonical enum. Unrecognized input
// returns CardinalityUnknown rather than guessing.
func ParseCardinality(raw string) Cardinality {
	s := strings.ToUpper(strings.TrimSpace(raw))
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, " ", "_")

	switch s {
	case "ONE_TO_ONE", "1:1", "1_1":
		return CardinalityOneToOne
	case "ONE_TO_MANY", "1:M", "1:N", "1_M", "1_N":
		return CardinalityOneToMany
	case "MANY_TO_ONE", "M:1", "N:1", "M_1", "N_1":
		return CardinalityManyToOne
	case "MANY_TO_MANY", "M:M", "N:M", "M:N", "N:N", "M_M", "N_M", "M_N":
		return CardinalityManyToMany
	default:
		return CardinalityUnknown
	}
}

// ComposeCardinality reduces a multi-hop relationship path to a single
// overall cardinality: any hop that fans out to many rows makes the whole
// path a "to-many" path, since a single source row can then map to multiple
// destination rows.
func ComposeCardinality(hops []Cardinality) Cardinality {
	if len(hops) == 0 {
		return CardinalityUnknown
	}

	sawToMany := false
	sawToOne := false
	for _, hop := range hops {
		switch hop {
		case CardinalityOneToMany, CardinalityManyToMany:
			sawToMany = true
		case CardinalityOneToOne, CardinalityManyToOne:
			sawToOne = true
		}
	}

	switch {
	case sawToMany:
		return CardinalityManyToMany
	case sawToOne:
		return CardinalityManyToOne
	default:
		return CardinalityUnknown
	}
}
