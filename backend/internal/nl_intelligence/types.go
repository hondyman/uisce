package nl_intelligence

import (
	"encoding/json"
	"time"
)

// Intent types
const (
	IntentSQLQuery            = "SQL_QUERY"
	IntentGraphQuery          = "GRAPH_QUERY"
	IntentLineageExplanation  = "LINEAGE_EXPLANATION"
	IntentIncidentExplanation = "INCIDENT_EXPLANATION"
	IntentChangeSetGeneration = "CHANGESET_GENERATION"
	IntentComplianceAnalysis  = "COMPLIANCE_ANALYSIS"
	IntentForecasting         = "FORECASTING"
	IntentAPIDesign           = "API_DESIGN"
	IntentSimulation          = "SIMULATION"
)

// NLRequest represents a natural language query request
type NLRequest struct {
	TenantScope []string        `json:"tenantScope"`
	Question    string          `json:"question"`
	Context     json.RawMessage `json:"context,omitempty"`
}

// NLResponse represents the interpretation of an NL query
type NLResponse struct {
	Intent         string          `json:"intent"`
	QueryPlan      *QueryPlan      `json:"queryPlan,omitempty"`
	Answer         string          `json:"answer,omitempty"`
	Data           json.RawMessage `json:"data,omitempty"`
	ReasoningSteps []string        `json:"reasoning_steps,omitempty"`
}

// QueryPlan defines how a query should be executed
type QueryPlan struct {
	Type       string         `json:"type"`   // "SQL" or "CYPHER" or "HYBRID"
	Engine     string         `json:"engine"` // "TRINO", "POSTGRES", "AGE", etc.
	SQL        string         `json:"sql,omitempty"`
	Cypher     string         `json:"cypher,omitempty"`
	GraphName  string         `json:"graphName,omitempty"`
	Parameters map[string]any `json:"parameters,omitempty"`
	Dialect    string         `json:"dialect,omitempty"`
}

// ChangeSetProposal represents a generated changeset
type ChangeSetProposal struct {
	Title            string           `json:"title"`
	Description      string           `json:"description"`
	ImpactedEntities []ImpactedEntity `json:"impactedEntities"`
	Risk             string           `json:"risk"` // "LOW", "MEDIUM", "HIGH"
	GovernanceMeta   json.RawMessage  `json:"governanceMeta,omitempty"`
}

type ImpactedEntity struct {
	NodeID     string `json:"nodeId"`
	EntityType string `json:"entityType"`
}

// IncidentExplanation details the root cause and blast radius
type IncidentExplanation struct {
	Narrative                 string `json:"narrative"`
	RootCause                 string `json:"rootCause"`
	BlastRadius               string `json:"blastRadius"`
	RecommendedFix            string `json:"recommendedFix"`
	SuggestedChangeSetSummary string `json:"suggestedChangeSetSummary"`
}

// ForecastResult predicts future failures
type ForecastResult struct {
	Predictions []Prediction `json:"predictions"`
}

type Prediction struct {
	Asset       string  `json:"asset"`
	Probability float64 `json:"probability"`
	Reason      string  `json:"reason"`
}

// Entity extracted from natural language
type Entity struct {
	Type  string `json:"type"`
	Value string `json:"value"`
	ID    string `json:"id,omitempty"`
}

// Filters extracted from natural language
type Filters struct {
	TimeRange  *TimeRange     `json:"timeRange,omitempty"`
	Status     string         `json:"status,omitempty"`
	Properties map[string]any `json:"properties,omitempty"`
}

type TimeRange struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}
