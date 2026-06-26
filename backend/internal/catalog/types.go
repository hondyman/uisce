package catalog

import "time"

// Graph Entity Kinds
const (
	KindTable        = "table"
	KindView         = "view"
	KindBO           = "bo"
	KindBusinessTerm = "BUSINESS_TERM"
	KindSemanticTerm = "SEMANTIC_TERM"
)

// Graph Edge Types
const (
	EdgeTypeMappedTo   = "IS_MAPPED_TO"
	EdgeTypeRelatedTo  = "IS_RELATED_TO"
	EdgeTypeClassified = "IS_CLASSIFIED_AS"
)

// CatalogNode represents a node in the graph (table, view, business_term, etc.)
type CatalogNode struct {
	ID            string                 `json:"id"`
	TenantID      string                 `json:"tenantId"`
	DatasourceID  string                 `json:"datasourceId"`
	Name          string                 `json:"name"`
	NodeType      string                 `json:"nodeType"`      // DB Column: node_type
	Kind          string                 `json:"kind"`          // Legacy aliases can be handled in logic if needed, but for now keeping to minimize breakage
	QualifiedPath string                 `json:"qualifiedPath"` // DB Column: qualified_path
	Description   string                 `json:"description"`
	Properties    map[string]interface{} `json:"properties"` // Flexible metadata (compliance, etc.)
	CreatedAt     time.Time              `json:"createdAt"`
	UpdatedAt     time.Time              `json:"updatedAt"`
}

// CatalogEdge represents a directed edge between two nodes
type CatalogEdge struct {
	ID           string                 `json:"id"`
	FromNode     string                 `json:"fromNode"` // DB Column: from_node
	ToNode       string                 `json:"toNode"`   // DB Column: to_node
	SourceID     string                 `json:"sourceId"` // Alias for older code
	TargetID     string                 `json:"targetId"` // Alias for older code
	EdgeType     string                 `json:"edgeType"`
	TenantID     string                 `json:"tenantId"`     // Added: DB Column: tenant_id
	DatasourceID string                 `json:"datasourceId"` // Added: DB Column: tenant_datasource_id
	Properties   map[string]interface{} `json:"properties"`
	Confidence   float64                `json:"confidence"`
	CreatedAt    time.Time              `json:"createdAt"`
}
