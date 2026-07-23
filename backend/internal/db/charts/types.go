package charts

import (
	"time"

	"github.com/google/uuid"
)

var (
	TABLE_NODE_TYPE_ID  = uuid.MustParse("49a50271-ae58-4d3e-ae1c-2f5b89d89192")
	COLUMN_NODE_TYPE_ID = uuid.MustParse("a64c1011-16e8-4ddf-b447-363bf8e15c9a")
	// MODEL_NODE_TYPE_ID represents semantic/semantic_model nodes used to
	// represent saved models in the catalog. The value below matches an
	// existing semantic model node type found in the database.
	MODEL_NODE_TYPE_ID = uuid.MustParse("c53f9e99-8d02-4dfb-bc1b-914747d35edb")
)

// DatabaseAsset is a generic representation of a database object like a table.
type DatabaseAsset struct {
	ID            string
	CoreID        uuid.NullUUID
	Name          string
	Schema        string
	Table         string
	QualifiedPath string
	NodeType      string
}

// ColumnData holds detailed information about a database column.
type ColumnData struct {
	ID            string
	Name          string
	Type          string
	IsCore        bool
	Nullable      bool
	Default       string
	Schema        string
	Table         string
	QualifiedPath string
	IsPrimaryKey  bool
	IsForeignKey  bool
	Properties    map[string]interface{} `json:"properties"`
}

// ChartInfo holds metadata about a saved chart.
type ChartInfo struct {
	Name      string    `db:"chart_name"`
	Type      string    `json:"type"`
	SizeBytes int64     `db:"size_bytes"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// ReactFlowNode is a struct compatible with the ReactFlow library for nodes.
type ReactFlowNode struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Position map[string]float64     `json:"position"`
	Data     map[string]interface{} `json:"data"`
}

// ReactFlowEdge is a struct compatible with the ReactFlow library for edges.
type ReactFlowEdge struct {
	ID       string                 `json:"id"`
	Source   string                 `json:"source"`
	Target   string                 `json:"target"`
	Type     string                 `json:"type"`
	Label    string                 `json:"label"`
	Data     map[string]interface{} `json:"data"`
	Animated bool                   `json:"animated,omitempty"`
}

// SemanticLineageChart represents the full structure of a semantic lineage graph.
type SemanticLineageChart struct {
	BusinessTerms   []SemanticNode         `json:"business_terms"`
	SemanticTerms   []SemanticNode         `json:"semantic_terms"`
	SemanticColumns []SemanticNode         `json:"semantic_columns"`
	DatabaseColumns []SemanticNode         `json:"database_columns"`
	Edges           []SemanticEdge         `json:"edges"`
	Viewport        map[string]interface{} `json:"viewport"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// TechnicalLineageChart represents a technical lineage graph for ReactFlow.
type TechnicalLineageChart struct {
	Nodes    []ReactFlowNode        `json:"nodes"`
	Edges    []ReactFlowEdge        `json:"edges"`
	Viewport map[string]interface{} `json:"viewport,omitempty"`
	Metadata map[string]interface{} `json:"metadata"`
}

// SemanticNode represents a node in the semantic lineage graph.
type SemanticNode struct {
	ID            uuid.UUID              `json:"id"`
	NodeName      string                 `json:"node_name"`
	NodeType      string                 `json:"node_type"`
	Description   string                 `json:"description"`
	QualifiedPath string                 `json:"qualified_path"`
	Properties    map[string]interface{} `json:"properties"`
}

// SemanticEdge represents an edge in the semantic lineage graph.
type SemanticEdge struct {
	ID               uuid.UUID              `json:"id"`
	SourceID         uuid.UUID              `json:"source_id"`
	TargetID         uuid.UUID              `json:"target_id"`
	EdgeType         string                 `json:"edge_type"`
	RelationshipType string                 `json:"relationship_type"`
	Properties       map[string]interface{} `json:"properties"`
}
