package domain

// PruningHints represents hints for query optimization
type PruningHints struct {
	Columns    []string // columns to project (empty = all)
	RowFilters []string // SQL predicates (ANDed)
	BindArgs   []any    // args for predicates, positional
}

// AssetSchema provides schema information for assets
type AssetSchema struct {
	ColumnsByScope map[string][]string // e.g., "metrics"->["avg_order_value","net_margin"], "dimensions"->["region","customer_id"]
	DefaultFilters []string            // e.g., tenant_id = $1
}

// TableSchema represents schema information for a table
type TableSchema struct {
	Name        string         `json:"name"`
	Columns     []ColumnSchema `json:"columns"`
	PrimaryKey  string         `json:"primary_key,omitempty"`
	ForeignKeys []string       `json:"foreign_keys,omitempty"`
}

// ColumnSchema represents schema information for a column
type ColumnSchema struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	Description  string `json:"description,omitempty"`
	IsPrimaryKey bool   `json:"is_primary_key,omitempty"`
	IsForeignKey bool   `json:"is_foreign_key,omitempty"`
}

// SchemaProvider interface for getting asset schemas
type SchemaProvider interface {
	GetAssetSchema(assetID string) (AssetSchema, error)
	GetTableSchema(assetID, tableName string) (TableSchema, error)
}

// PlannerAdapter converts evaluation results into pruning hints
type PlannerAdapter struct {
	Schema SchemaProvider
}

// BuildHints uses evaluation results to create pruning hints for the query planner
func (p *PlannerAdapter) BuildHints(assetID string, decision EvaluationDecision, tenantID string) (PruningHints, error) {
	schema, err := p.Schema.GetAssetSchema(assetID)
	if err != nil {
		return PruningHints{}, err
	}

	hints := PruningHints{}

	// Column pruning: if partial decision, map scopes to concrete columns
	if decision.Decision == "partial" && len(decision.AllowedScopes) > 0 {
		colset := map[string]struct{}{}
		for _, sc := range decision.AllowedScopes {
			if cols, ok := schema.ColumnsByScope[sc]; ok {
				for _, c := range cols {
					colset[c] = struct{}{}
				}
			}
		}
		for c := range colset {
			hints.Columns = append(hints.Columns, c)
		}
	}

	// Row pruning: always enforce tenant isolation
	if len(schema.DefaultFilters) > 0 {
		hints.RowFilters = append(hints.RowFilters, schema.DefaultFilters...)
		hints.BindArgs = append(hints.BindArgs, tenantID)
	}

	return hints, nil
}

// EvaluationDecision represents the result of an access evaluation
type EvaluationDecision struct {
	Decision      string   // "allow", "deny", "partial"
	Reason        string   // explanation
	AllowedScopes []string // for partial decisions
}
