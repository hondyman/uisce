package query

import (
	"time"

	"github.com/google/uuid"
)

// QueryUnderstanding represents the structured intent extracted from natural language
type QueryUnderstanding struct {
	OriginalQuery    string              `json:"original_query"`
	Intent           QueryIntent         `json:"intent"`
	Entities         []ExtractedEntity   `json:"entities"`
	Filters          []QueryFilter       `json:"filters"`
	Aggregations     []Aggregation       `json:"aggregations"`
	Comparisons      []Comparison        `json:"comparisons"`
	TimeConstraints  *TimeConstraint     `json:"time_constraints"`
	OutputFormat     string              `json:"output_format"`
	Confidence       float64             `json:"confidence"`
	Ambiguities      []Ambiguity         `json:"ambiguities"`
	ClarificationQs  []string            `json:"clarification_questions"`
}

type QueryIntent struct {
	PrimaryIntent    string   `json:"primary_intent"` // "search", "compare", "analyze", "summarize", "alert"
	SecondaryIntents []string `json:"secondary_intents"`
	QueryType        string   `json:"query_type"`
	Complexity       string   `json:"complexity"`
}

type ExtractedEntity struct {
	EntityID     uuid.UUID              `json:"entity_id"`
	EntityType   string                 `json:"entity_type"`
	Name         string                 `json:"name"`
	Identifier   string                 `json:"identifier"`
	Confidence   float64                `json:"confidence"`
	Metadata     map[string]interface{} `json:"metadata"`
}

type QueryFilter struct {
	Field        string      `json:"field"`
	Operator     string      `json:"operator"`
	Value        interface{} `json:"value"`
	Negated      bool        `json:"negated"`
}

type Aggregation struct {
	Function string   `json:"function"`
	Field    string   `json:"field"`
	GroupBy  []string `json:"group_by"`
}

type Comparison struct {
	CompareType string        `json:"compare_type"`
	Entity1     string        `json:"entity1"`
	Entity2     string        `json:"entity2"`
	Metrics     []string      `json:"metrics"`
}

type TimeConstraint struct {
	Type      string    `json:"type"`
	Start     time.Time `json:"start"`
	End       time.Time `json:"end"`
	Relative  string    `json:"relative"`
}

type Ambiguity struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Options     []string `json:"options"`
}

// QueryResult represents the final output of the query execution
type QueryResult struct {
	Understanding  *QueryUnderstanding `json:"understanding"`
	Results        interface{}         `json:"results"`
	Narrative      string              `json:"narrative,omitempty"`
	Confidence     float64             `json:"confidence"`
}
