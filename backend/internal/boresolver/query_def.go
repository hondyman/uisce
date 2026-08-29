package boresolver

// QueryDef is the frontend Query Builder contract. It is the only input the
// UI ever sends; SQL construction happens exclusively in the backend.
type QueryDef struct {
	Context QueryContext `json:"context"`
	Query   QueryRequest `json:"query"`
}

// QueryContext carries the security + binding scope for the query.
type QueryContext struct {
	BOID      string `json:"boId"`
	BindingID string `json:"bindingId"`
	TenantID  string `json:"tenantId"`
}

// QueryRequest is the user-intent portion of the QueryDef.
type QueryRequest struct {
	Dimensions []DimensionDef `json:"dimensions"`
	Measures   []MeasureDef   `json:"measures"`
	Filters    []FilterDef    `json:"filters"`
	GroupBy    []string       `json:"groupBy,omitempty"`
	Limit      int            `json:"limit,omitempty"`
}

// DimensionDef selects a semantic term to group/project by.
type DimensionDef struct {
	TermNodeID string `json:"termNodeId"`
	Alias      string `json:"alias"`
}

// MeasureDef selects a semantic term with an aggregation.
type MeasureDef struct {
	TermNodeID  string `json:"termNodeId"`
	Alias       string `json:"alias"`
	Aggregation string `json:"agg"`
}

// FilterDef applies a predicate to a semantic term.
type FilterDef struct {
	TermNodeID string      `json:"termNodeId"`
	Operator   string      `json:"operator"`
	Value      interface{} `json:"value,omitempty"`
}

// QueryPreviewResponse is returned by POST /api/query/preview.
type QueryPreviewResponse struct {
	SQL        string        `json:"sql"`
	Dialect    string        `json:"dialect,omitempty"`
	Parameters []interface{} `json:"parameters,omitempty"`
}

// QueryExecuteResponse is returned by POST /api/query/execute.
type QueryExecuteResponse struct {
	SQL             string                 `json:"sql"`
	Columns         []QueryResultColumn    `json:"columns"`
	Rows            []map[string]interface{} `json:"rows"`
	RowCount        int                    `json:"rowCount"`
	ExecutionTimeMs int64                  `json:"executionTimeMs"`
}

// QueryResultColumn describes one result column.
type QueryResultColumn struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// SemanticTermView is the shape exposed by GET /api/business-objects/{boId}/terms.
type SemanticTermView struct {
	TermNodeID         string `json:"termNodeId"`
	TermKey            string `json:"termKey"`
	TermName           string `json:"termName"`
	DisplayName        string `json:"displayName"`
	Description        string `json:"description,omitempty"`
	DataType           string `json:"dataType"`
	Role               string `json:"role"`
	BindingStatus      string `json:"bindingStatus"`
	DefaultAggregation string `json:"defaultAggregation,omitempty"`
}

// BOBinding captures the physical binding context for a Business Object.
// db tags are required: sqlx's default column mapper only lowercases field
// names (DrivingTable -> "drivingtable"), it doesn't insert underscores, so
// without these tags a snake_case query alias like "driving_table" never
// matches its destination field and struct-scans silently fail to populate it.
type BOBinding struct {
	ID               string `db:"id"`
	Name             string `db:"name"`
	DialectName      string `db:"dialect_name"`
	ConnectionString string `db:"connection_string"`
	BindingID        string `db:"binding_id"`
	BOID             string `db:"bo_id"`
	DatasourceID     string `db:"datasource_id"`
	DrivingTable     string `db:"driving_table"`
	DrivingTableID   string `db:"driving_table_id"`
}

