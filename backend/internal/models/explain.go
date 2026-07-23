package models

import (
	"time"
)

// ExplainResponse is the root object for the Explain API
type ExplainResponse struct {
	Header         ExplainHeader    `json:"header"`
	Summary        ExplainSummary   `json:"summary"`
	Lineage        ExplainLineage   `json:"lineage"`
	Rows           []ExplainRow     `json:"rows,omitempty"`
	EvaluationPath []ExplainStep    `json:"evaluationPath"`
	TermMetadata   SemanticTerm     `json:"termMetadata"`
	Usage          ExplainUsage     `json:"usage"`
	SQL            *ExplainSQL      `json:"sql,omitempty"`
	Anomalies      []ExplainAnomaly `json:"anomalies"`
	Actions        ExplainActions   `json:"actions"`
}

type ExplainHeader struct {
	TermID           string      `json:"termId"`
	EntityType       string      `json:"entityType"`
	EntityID         string      `json:"entityId"`
	Value            interface{} `json:"value"` // number, string, bool, json
	EvaluatedAt      time.Time   `json:"evaluatedAt"`
	EvaluatorVersion string      `json:"evaluatorVersion"`
}

type ExplainSummary struct {
	HumanReadable string                 `json:"humanReadable"`
	Stats         map[string]interface{} `json:"stats,omitempty"`
}

type ExplainLineage struct {
	SemanticTerm    SemanticTerm     `json:"semanticTerm"`
	Dependencies    []SemanticTerm   `json:"dependencies"`
	PhysicalColumns []PhysicalColumn `json:"physicalColumns"`
	JoinPath        string           `json:"joinPath,omitempty"`
}

type PhysicalColumn struct {
	Table  string `json:"table"`
	Column string `json:"column"`
}

type ExplainRow struct {
	Key      map[string]interface{} `json:"key"`
	Fields   map[string]interface{} `json:"fields"`
	Included bool                   `json:"included"`
}

type ExplainStep struct {
	Step        int                    `json:"step"`
	Action      string                 `json:"action"`
	Description string                 `json:"description"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

type ExplainUsage struct {
	Cubes             []NameFieldPair `json:"cubes"`
	BusinessProcesses []NameIDPair    `json:"businessProcesses"`
	LLMProfiles       []NameIDPair    `json:"llmProfiles"`
	Reports           []NameIDPair    `json:"reports"`
}

type NameFieldPair struct {
	Name  string `json:"name"`
	Field string `json:"field"`
}

type NameIDPair struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type,omitempty"` // For bps: condition/routing
}

type ExplainSQL struct {
	Text string `json:"text"`
}

type ExplainAnomaly struct {
	Type            string `json:"type"`
	Severity        string `json:"severity"`
	Message         string `json:"message"`
	SuggestedAction string `json:"suggestedAction"`
}

type ExplainActions struct {
	CanRecompute      bool `json:"canRecompute"`
	CanOpenTerm       bool `json:"canOpenTerm"`
	CanOpenLineage    bool `json:"canOpenLineage"`
	CanOpenCube       bool `json:"canOpenCube"`
	CanOpenBpInstance bool `json:"canOpenBpInstance"`
}
