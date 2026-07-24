package charts

// HierarchicalNode represents a node with parent-child relationships
type HierarchicalNode struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	NodeType      string                 `json:"nodeType"`
	QualifiedPath string                 `json:"qualifiedPath"`
	ParentID      string                 `json:"parentId,omitempty"`
	ParentPath    string                 `json:"parentPath,omitempty"`
	Children      []HierarchicalNode     `json:"children,omitempty"`
	Properties    map[string]interface{} `json:"properties"`
	Level         int                    `json:"level"` // 0=schema, 1=table, 2=column
}

// ContainerNodeData represents a container node that holds child nodes
type ContainerNodeData struct {
	Label         string                 `json:"label"`
	NodeType      string                 `json:"nodeType"`
	QualifiedPath string                 `json:"qualifiedPath"`
	Children      []string               `json:"children"` // IDs of child nodes
	Level         int                    `json:"level"`
	IsContainer   bool                   `json:"isContainer"`
	Properties    map[string]interface{} `json:"properties"`
}

// EnhancedReactFlowNode extends ReactFlowNode with hierarchical information
type EnhancedReactFlowNode struct {
	ReactFlowNode
	ParentNode *string `json:"parentNode,omitempty"` // ID of parent container
	Extent     *string `json:"extent,omitempty"`     // "parent" for child nodes
}

// HierarchicalLayout represents a layout with container nodes
type HierarchicalLayout struct {
	Nodes     []EnhancedReactFlowNode `json:"nodes"`
	Edges     []ReactFlowEdge         `json:"edges"`
	Viewport  map[string]interface{}  `json:"viewport"`
	Metadata  map[string]interface{}  `json:"metadata"`
	Hierarchy map[string][]string     `json:"hierarchy"` // parentId -> childIds
}

// DatabaseHierarchy represents the complete database structure
type DatabaseHierarchy struct {
	Schemas map[string]*SchemaNode `json:"schemas"`
}

type SchemaNode struct {
	ID       string                `json:"id"`
	Name     string                `json:"name"`
	Tables   map[string]*TableNode `json:"tables"`
	Position Position              `json:"position"`
}

type TableNode struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Schema   string                 `json:"schema"`
	Columns  map[string]*ColumnNode `json:"columns"`
	Position Position               `json:"position"`
}

type ColumnNode struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Table    string   `json:"table"`
	Schema   string   `json:"schema"`
	DataType string   `json:"dataType"`
	Position Position `json:"position"`
}

type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}
