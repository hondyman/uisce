package boresolver

// ============================================================================
// SEMANTIC LAYER TYPES
// ============================================================================

// SemanticTerm represents a canonical semantic term in the semantic layer.
// It does NOT contain physical mappings; those live in catalog edges.
type SemanticTerm struct {
	ID          string `db:"id"`
	TenantID    string `db:"tenant_id"`
	Name        string `db:"name"`
	DisplayName string `db:"display_name"`
	Description string `db:"description"`
	Category    string `db:"category"`
	IsSystem    bool   `db:"is_system"`
	CreatedAt   string `db:"created_at"`
	UpdatedAt   string `db:"updated_at"`
}

// CatalogNode represents a node in the semantic catalog (table, column, or custom entity).
type CatalogNode struct {
	ID        string  `db:"id"`
	TenantID  string  `db:"tenant_id"`
	Type      string  `db:"type"` // "table", "column", "expression", etc.
	Name      string  `db:"name"`
	ParentID  *string `db:"parent_id"` // For columns, this points to the table node
	Metadata  string  `db:"metadata"`  // JSON for additional properties
	CreatedAt string  `db:"created_at"`
	UpdatedAt string  `db:"updated_at"`
}

// CatalogEdgeType represents the type of relationship between two catalog nodes.
type CatalogEdgeType string

const (
	CatalogEdgeTypeMapsToColumn CatalogEdgeType = "TERM_MAPS_TO_COLUMN"
	CatalogEdgeTypeMapsToTable  CatalogEdgeType = "TERM_MAPS_TO_TABLE"
	CatalogEdgeTypeExpression   CatalogEdgeType = "TERM_MAPS_TO_EXPRESSION"
)

// CatalogEdge represents a relationship between a semantic term and physical resources.
type CatalogEdge struct {
	ID           string `db:"id"`
	TenantID     string `db:"tenant_id"`
	FromID       string `db:"from_id"` // semantic_term_id
	ToID         string `db:"to_id"`   // physical resource (catalog_node_id)
	Type         string `db:"type"`    // CatalogEdgeType
	DatasourceID string `db:"datasource_id"`
	Metadata     string `db:"metadata"`
	CreatedAt    string `db:"created_at"`
}

// BORelationshipRecord represents a relationship between two business objects.
type BORelationshipRecord struct {
	ID         string `db:"id"`
	TenantID   string `db:"tenant_id"`
	FromBOID   string `db:"from_bo_id"`
	ToBOID     string `db:"to_bo_id"`
	JoinType   string `db:"join_type"` // "LEFT", "INNER", "RIGHT"
	JoinOnJSON string `db:"join_on"`   // JSON array
	Metadata   string `db:"metadata"`
	IsActive   bool   `db:"is_active"`
	CreatedAt  string `db:"created_at"`
}

// JoinOnPair defines a join condition between two fields.
type JoinOnPair struct {
	FromFieldID string `json:"from_field_id"`
	ToFieldID   string `json:"to_field_id"`
}

// ResolvedField represents a fully resolved BO field with physical table and column.
type ResolvedField struct {
	FieldID        string // bo_field.id
	FieldName      string // bo_field.name
	SemanticTermID string // bo_field.semantic_term_id
	Table          string // physical table name
	Column         string // physical column name
	Alias          string // alias in the SQL query (e.g., "t0")
	SemanticName   string // display name for lineage/explanation
	SourceType     string // "OVERRIDE", "SEMANTIC", or "DRIVING_TABLE"
}
