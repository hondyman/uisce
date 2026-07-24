package query

import (
	"fmt"
	"strings"
)

// GenerationEngine generates SQL queries from parsed intent
type GenerationEngine struct {
	// Template mappings for different query types
	templates map[string]string
}

// NewGenerationEngine creates a new query generation engine
func NewGenerationEngine() *GenerationEngine {
	engine := &GenerationEngine{
		templates: make(map[string]string),
	}
	engine.initializeTemplates()
	return engine
}

// GenerateQuerySkeleton creates a query skeleton from parsed intent
func (ge *GenerationEngine) GenerateQuerySkeleton(intent *ParsedIntent, govCtx *GovernanceContext) (*QuerySkeleton, error) {
	skeleton := &QuerySkeleton{
		Measures:   []string{},
		Dimensions: []string{},
		Filters:    []QueryFilter{},
	}

	// Map semantic names to actual column names
	for _, metric := range intent.Metrics {
		if columnName, ok := govCtx.AssetMappings[metric]; ok {
			skeleton.Measures = append(skeleton.Measures, columnName)
		} else {
			// If not found in mappings, use as-is (might be direct column name)
			skeleton.Measures = append(skeleton.Measures, metric)
		}
	}

	for _, dimension := range intent.Dimensions {
		if columnName, ok := govCtx.AssetMappings[dimension]; ok {
			skeleton.Dimensions = append(skeleton.Dimensions, columnName)
		} else {
			// If not found in mappings, use as-is
			skeleton.Dimensions = append(skeleton.Dimensions, dimension)
		}
	}

	// Convert intent filters to query filters
	for _, filter := range intent.Filters {
		skeleton.Filters = append(skeleton.Filters, QueryFilter{
			Field:    filter.Field,
			Operator: filter.Operator,
			Value:    strings.Join(filter.Values, ","),
		})
	}

	// Add time range filter if present
	if intent.TimeRange != nil && intent.TimeRange.Start != "" && intent.TimeRange.End != "" {
		skeleton.Filters = append(skeleton.Filters, QueryFilter{
			Field:    "order_date", // Assuming order_date is the time field
			Operator: "BETWEEN",
			Value:    fmt.Sprintf("'%s' AND '%s'", intent.TimeRange.Start, intent.TimeRange.End),
		})
	}

	return skeleton, nil
}

// GenerateSQL generates the final SQL from query skeleton
func (ge *GenerationEngine) GenerateSQL(skeleton *QuerySkeleton, govCtx *GovernanceContext) (string, error) {
	if len(skeleton.Measures) == 0 {
		return "", fmt.Errorf("no measures specified in query")
	}

	var sql strings.Builder

	// SELECT clause
	sql.WriteString("SELECT ")
	if len(skeleton.Dimensions) > 0 {
		// Include dimensions in SELECT
		allColumns := append(skeleton.Dimensions, skeleton.Measures...)
		sql.WriteString(strings.Join(allColumns, ", "))
	} else {
		// Only measures
		sql.WriteString(strings.Join(skeleton.Measures, ", "))
	}

	// FROM clause - use a default table name based on context
	tableName := ge.determineTableName(govCtx)
	sql.WriteString(fmt.Sprintf(" FROM %s", tableName))

	// WHERE clause
	if len(skeleton.Filters) > 0 {
		sql.WriteString(" WHERE ")
		var conditions []string
		for _, filter := range skeleton.Filters {
			condition := ge.buildCondition(filter)
			if condition != "" {
				conditions = append(conditions, condition)
			}
		}
		sql.WriteString(strings.Join(conditions, " AND "))
	}

	// GROUP BY clause (if we have dimensions)
	if len(skeleton.Dimensions) > 0 {
		sql.WriteString(" GROUP BY ")
		sql.WriteString(strings.Join(skeleton.Dimensions, ", "))
	}

	// ORDER BY clause (default ordering)
	if len(skeleton.Dimensions) > 0 {
		sql.WriteString(" ORDER BY ")
		sql.WriteString(strings.Join(skeleton.Dimensions, ", "))
	}

	return sql.String(), nil
}

// determineTableName determines the appropriate table name based on context
func (ge *GenerationEngine) determineTableName(govCtx *GovernanceContext) string {
	// Simple logic: if datasource contains "orders", use orders_view, etc.
	datasource := strings.ToLower(govCtx.Datasource)
	if strings.Contains(datasource, "order") {
		return "orders_view"
	}
	if strings.Contains(datasource, "customer") {
		return "customers_view"
	}

	// Default table name
	return "data_view"
}

// buildCondition builds a SQL condition from a filter
func (ge *GenerationEngine) buildCondition(filter QueryFilter) string {
	switch filter.Operator {
	case "=":
		return fmt.Sprintf("%s = %s", filter.Field, ge.quoteValue(filter.Value))
	case "!=":
		return fmt.Sprintf("%s != %s", filter.Field, ge.quoteValue(filter.Value))
	case ">":
		return fmt.Sprintf("%s > %s", filter.Field, ge.quoteValue(filter.Value))
	case "<":
		return fmt.Sprintf("%s < %s", filter.Field, ge.quoteValue(filter.Value))
	case ">=":
		return fmt.Sprintf("%s >= %s", filter.Field, ge.quoteValue(filter.Value))
	case "<=":
		return fmt.Sprintf("%s <= %s", filter.Field, ge.quoteValue(filter.Value))
	case "LIKE":
		return fmt.Sprintf("%s LIKE %s", filter.Field, ge.quoteValue(filter.Value))
	case "IN":
		return fmt.Sprintf("%s IN (%s)", filter.Field, filter.Value)
	case "BETWEEN":
		return fmt.Sprintf("%s %s", filter.Field, filter.Value)
	default:
		return fmt.Sprintf("%s = %s", filter.Field, ge.quoteValue(filter.Value))
	}
}

// quoteValue quotes string values appropriately
func (ge *GenerationEngine) quoteValue(value string) string {
	// If it looks like a string (not a number), quote it
	if value == "" {
		return "''"
	}

	// Check if it's already quoted
	if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") {
		return value
	}

	// Check if it's a number
	if ge.isNumeric(value) {
		return value
	}

	// Quote it
	return fmt.Sprintf("'%s'", value)
}

// isNumeric checks if a string represents a number
func (ge *GenerationEngine) isNumeric(s string) bool {
	for _, r := range s {
		if (r < '0' || r > '9') && r != '.' && r != '-' {
			return false
		}
	}
	return true
}

// initializeTemplates sets up query generation templates
func (ge *GenerationEngine) initializeTemplates() {
	// Basic aggregation template
	ge.templates["aggregation"] = "SELECT {{measures}} FROM {{table}} WHERE {{conditions}} GROUP BY {{dimensions}}"

	// Time series template
	ge.templates["time_series"] = "SELECT {{time_dimension}}, {{measures}} FROM {{table}} WHERE {{conditions}} GROUP BY {{time_dimension}} ORDER BY {{time_dimension}}"

	// Simple select template
	ge.templates["simple"] = "SELECT {{columns}} FROM {{table}} WHERE {{conditions}}"
}
