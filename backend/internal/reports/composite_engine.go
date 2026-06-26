package reports

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

// CompositeQuery represents a query across multiple semantic views
type CompositeQuery struct {
	Views        []ViewQuery            `json:"views"`
	Joins        []JoinClause           `json:"joins"`
	Aggregations []AggregationClause    `json:"aggregations"`
	Filters      map[string]interface{} `json:"filters"`
	OrderBy      string                 `json:"order_by"`
	Limit        int                    `json:"limit"`
}

// ViewQuery represents a single semantic view in a composite query
type ViewQuery struct {
	ViewID uuid.UUID `json:"view_id"`
	Alias  string    `json:"alias"`
	Fields []string  `json:"fields"`
}

// JoinClause represents a join between two views
type JoinClause struct {
	LeftAlias  string `json:"left_alias"`
	RightAlias string `json:"right_alias"`
	JoinType   string `json:"join_type"` // inner, left, right, full
	OnField    string `json:"on_field"`
}

// AggregationClause represents an aggregation operation
type AggregationClause struct {
	Function string `json:"function"` // SUM, AVG, COUNT, etc.
	Field    string `json:"field"`
	Alias    string `json:"alias"`
}

// CompositeEngine executes queries across multiple semantic views
type CompositeEngine struct {
	db *sql.DB
}

// NewCompositeEngine creates a new composite query engine
func NewCompositeEngine(db *sql.DB) *CompositeEngine {
	return &CompositeEngine{db: db}
}

// ExecuteComposite executes a composite query
func (ce *CompositeEngine) ExecuteComposite(ctx context.Context, query *CompositeQuery) ([]map[string]interface{}, error) {
	// Build SQL query
	sqlQuery, args, err := ce.buildSQL(query)
	if err != nil {
		return nil, fmt.Errorf("failed to build SQL: %w", err)
	}

	// Execute query
	rows, err := ce.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	// Scan results
	var results []map[string]interface{}
	for rows.Next() {
		// Create a slice of interface{} to hold each column value
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Build result map
		result := make(map[string]interface{})
		for i, col := range columns {
			result[col] = values[i]
		}
		results = append(results, result)
	}

	return results, nil
}

// buildSQL constructs the SQL query from CompositeQuery spec
func (ce *CompositeEngine) buildSQL(query *CompositeQuery) (string, []interface{}, error) {
	if len(query.Views) == 0 {
		return "", nil, fmt.Errorf("no views specified")
	}

	// Start with SELECT clause
	selectClause := ce.buildSelectClause(query)

	// Build FROM clause with first view
	fromClause := fmt.Sprintf("FROM semantic_view_data_%s AS %s",
		query.Views[0].ViewID.String(), query.Views[0].Alias)

	// Build JOIN clauses
	joinClauses := ""
	for _, join := range query.Joins {
		joinType := "INNER"
		switch join.JoinType {
		case "left":
			joinType = "LEFT"
		case "right":
			joinType = "RIGHT"
		case "full":
			joinType = "FULL"
		}

		// Find the view for right alias
		var rightViewID uuid.UUID
		for _, v := range query.Views {
			if v.Alias == join.RightAlias {
				rightViewID = v.ViewID
				break
			}
		}

		joinClauses += fmt.Sprintf(" %s JOIN semantic_view_data_%s AS %s ON %s.%s = %s.%s",
			joinType,
			rightViewID.String(),
			join.RightAlias,
			join.LeftAlias,
			join.OnField,
			join.RightAlias,
			join.OnField,
		)
	}

	// Build WHERE clause
	whereClause := ""
	args := []interface{}{}
	if len(query.Filters) > 0 {
		whereClauses := []string{}
		argIndex := 1
		for field, value := range query.Filters {
			whereClauses = append(whereClauses, fmt.Sprintf("%s = $%d", field, argIndex))
			args = append(args, value)
			argIndex++
		}
		whereClause = " WHERE " + whereClauses[0]
		for i := 1; i < len(whereClauses); i++ {
			whereClause += " AND " + whereClauses[i]
		}
	}

	// Build ORDER BY clause
	orderByClause := ""
	if query.OrderBy != "" {
		orderByClause = " ORDER BY " + query.OrderBy
	}

	// Build LIMIT clause
	limitClause := ""
	if query.Limit > 0 {
		limitClause = fmt.Sprintf(" LIMIT %d", query.Limit)
	}

	// Combine all parts
	sql := selectClause + " " + fromClause + joinClauses + whereClause + orderByClause + limitClause

	return sql, args, nil
}

// buildSelectClause constructs the SELECT part of the query
func (ce *CompositeEngine) buildSelectClause(query *CompositeQuery) string {
	fields := []string{}

	// Add fields from each view
	for _, view := range query.Views {
		for _, field := range view.Fields {
			fields = append(fields, fmt.Sprintf("%s.%s", view.Alias, field))
		}
	}

	// Add aggregations
	for _, agg := range query.Aggregations {
		fields = append(fields, fmt.Sprintf("%s(%s) AS %s", agg.Function, agg.Field, agg.Alias))
	}

	if len(fields) == 0 {
		return "SELECT *"
	}

	selectClause := "SELECT " + fields[0]
	for i := 1; i < len(fields); i++ {
		selectClause += ", " + fields[i]
	}

	return selectClause
}

// ValidateQuery validates a composite query before execution
func (ce *CompositeEngine) ValidateQuery(query *CompositeQuery) error {
	if len(query.Views) == 0 {
		return fmt.Errorf("at least one view is required")
	}

	// Ensure all aliases are unique
	aliases := make(map[string]bool)
	for _, view := range query.Views {
		if aliases[view.Alias] {
			return fmt.Errorf("duplicate alias: %s", view.Alias)
		}
		aliases[view.Alias] = true
	}

	// Validate join references
	for _, join := range query.Joins {
		if !aliases[join.LeftAlias] {
			return fmt.Errorf("unknown left alias in join: %s", join.LeftAlias)
		}
		if !aliases[join.RightAlias] {
			return fmt.Errorf("unknown right alias in join: %s", join.RightAlias)
		}
	}

	return nil
}
