package models

import (
	"encoding/json"
	"time"
)

// 1. Core Export Bundle

type ExportBundle struct {
	Version          string       `json:"version"`
	IcebergVersion   string       `json:"iceberg_version,omitempty"` // Added 3-way diff anchor
	ExportedAt       time.Time    `json:"exported_at"`
	TenantID         *string      `json:"tenant_id,omitempty"`
	BusinessObjects  []NodeExport `json:"business_objects"`
	SemanticTerms    []NodeExport `json:"semantic_terms"`
	CalculationTerms []NodeExport `json:"calculation_terms"`
	PreAggregations  []NodeExport `json:"pre_aggregations,omitempty"` // Added for pre-agg export/import
}

type NodeExport struct {
	Node  CatalogNodeExport `json:"node"`
	Edges []EdgeExport      `json:"edges"`
}

type CatalogNodeExport struct {
	NodeTypeID    string          `json:"node_type_id"`
	NodeName      string          `json:"node_name"`
	Description   string          `json:"description,omitempty"`
	QualifiedPath string          `json:"qualified_path"`
	Properties    json.RawMessage `json:"properties"`
	Config        json.RawMessage `json:"config"`
}

type EdgeExport struct {
	SourceName string          `json:"source_name"`
	SourceType string          `json:"source_type"` // business_object, semantic_term, calculation_term, table, column
	TargetName string          `json:"target_name"`
	TargetType string          `json:"target_type"`
	EdgeType   string          `json:"edge_type"`
	Properties json.RawMessage `json:"properties,omitempty"`
}

// 2. Import Request & Result

type ConflictStrategy string

const (
	ConflictCreate  ConflictStrategy = "create"
	ConflictReplace ConflictStrategy = "replace"
	ConflictMerge   ConflictStrategy = "merge"
)

type ImportMode string

const (
	ImportModeDryRun ImportMode = "dry_run"
	ImportModeApply  ImportMode = "apply"
)

type ImportRequest struct {
	Mode             ImportMode       `json:"mode"`
	ConflictStrategy ConflictStrategy `json:"conflict_strategy"`
	DatasourceID     string           `json:"datasource_id"`
	Region           string           `json:"region"`
	Bundle           ExportBundle     `json:"bundle"`
}

type ImportResult struct {
	Mode      ImportMode    `json:"mode"`
	Summary   ImportSummary `json:"summary"`
	NodeDiffs []NodeDiff    `json:"node_diffs"`
	EdgeDiffs []EdgeDiff    `json:"edge_diffs"`
	Errors    []string      `json:"errors,omitempty"`
}

type ImportSummary struct {
	NodesToCreate    int `json:"nodes_to_create"`
	NodesToUpdate    int `json:"nodes_to_update"`
	NodesConflicting int `json:"nodes_conflicting"`
	EdgesToCreate    int `json:"edges_to_create"`
	EdgesToUpdate    int `json:"edges_to_update"`
}

// 3. Diff Structures

type DiffStatus string

const (
	DiffMissing         DiffStatus = "missing"
	DiffExistsSame      DiffStatus = "exists_same"
	DiffExistsDifferent DiffStatus = "exists_different"
	DiffConflict        DiffStatus = "conflict" // Added for explicit conflict status
)

type NodeDiff struct {
	NodeType string            `json:"node_type"` // business_object, semantic_term, calculation_term
	NodeName string            `json:"node_name"`
	Status   DiffStatus        `json:"status"`
	Existing *NodeSnapshot     `json:"existing,omitempty"`
	Incoming *NodeSnapshot     `json:"incoming,omitempty"`
	Iceberg  *NodeSnapshot     `json:"iceberg,omitempty"` // Added 3-way comparison point
	Diff     *NodePropertyDiff `json:"diff,omitempty"`
	Errors   []string          `json:"errors,omitempty"` // Governance or validation errors specific to this node
}

type NodeSnapshot struct {
	Properties json.RawMessage `json:"properties"`
	Config     json.RawMessage `json:"config"`
}

type NodePropertyDiff struct {
	Properties ThreeWayDiff `json:"properties"`
	Config     ThreeWayDiff `json:"config"`
}

type ThreeWayDiff struct {
	Added     map[string]interface{}    `json:"added"`
	Removed   map[string]interface{}    `json:"removed"`
	Changed   map[string]FieldChange    `json:"changed"`
	Conflicts map[string]ConflictDetail `json:"conflicts"`
}

type ConflictDetail struct {
	Postgres interface{} `json:"postgres"`
	Iceberg  interface{} `json:"iceberg"`
	Incoming interface{} `json:"incoming"`
}

type JSONFieldDiff struct {
	Changed map[string]FieldChange `json:"changed"`
	Added   map[string]interface{} `json:"added"`
	Removed map[string]interface{} `json:"removed"`
}

type FieldChange struct {
	From interface{} `json:"from"`
	To   interface{} `json:"to"`
}

type EdgeDiff struct {
	EdgeType string     `json:"edge_type"`
	Source   EdgeEndRef `json:"source"`
	Target   EdgeEndRef `json:"target"`
	Status   DiffStatus `json:"status"`
}

type EdgeEndRef struct {
	Type string `json:"type"` // business_object, semantic_term, calculation_term, table, column
	Name string `json:"name"`
}
