package api

import (
	"fmt"
	"strings"
)

// ============================================================================
// Reporting Query Generator
// ============================================================================
// Generates SQL queries for self-service reporting using discovered relationships

// ReportingQueryGenerator generates SQL for multi-entity queries
type ReportingQueryGenerator struct {
	tenantID     string
	datasourceID string
}

// NewReportingQueryGenerator creates a new reporting query generator
func NewReportingQueryGenerator(tenantID, datasourceID string) *ReportingQueryGenerator {
	return &ReportingQueryGenerator{
		tenantID:     tenantID,
		datasourceID: datasourceID,
	}
}

// ReportQuery represents a generated report query
type ReportQuery struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	SQL         string   `json:"sql"`
	JoinPaths   []string `json:"join_paths"`
	Metrics     []string `json:"metrics"`
	Dimensions  []string `json:"dimensions"`
	Filters     []string `json:"filters"`
	Confidence  float64  `json:"confidence"`
}

// ReportQueryBuilder builds complex report queries
type ReportQueryBuilder struct {
	baseEntity  string
	baseTable   string
	baseColumns []string
	joins       []JoinClause
	metrics     []MetricDefinition
	dimensions  []DimensionDefinition
	filters     []FilterCondition
	confidence  float64
}

// JoinClause represents a JOIN in the query
type JoinClause struct {
	SourceTable  string
	SourceColumn string
	TargetTable  string
	TargetColumn string
	JoinType     string // "INNER", "LEFT", "RIGHT"
	Cardinality  string // "1:1", "1:N", "N:M"
}

// MetricDefinition represents an aggregated metric
type MetricDefinition struct {
	Name         string // e.g., "total_orders", "avg_order_value"
	Expression   string // e.g., "COUNT(*)", "AVG(order_amount)"
	SourceTable  string
	SourceColumn string
	AggFunction  string // SUM, AVG, COUNT, MIN, MAX
}

// DimensionDefinition represents a grouping dimension
type DimensionDefinition struct {
	Name        string // e.g., "customer_segment"
	Table       string
	Column      string
	DataType    string // "STRING", "DATE", "NUMBER"
	DisplayName string
}

// FilterCondition represents a WHERE clause condition
type FilterCondition struct {
	Table    string
	Column   string
	Operator string // "=", ">", "<", "LIKE", "IN"
	Value    string
}

// GenerateMultiEntityQuery generates a query joining multiple entities
func (gen *ReportingQueryGenerator) GenerateMultiEntityQuery(
	baseEntity string,
	baseTable string,
	joins []JoinClause,
	metrics []MetricDefinition,
	dimensions []DimensionDefinition,
	filters []FilterCondition,
) *ReportQuery {

	builder := &ReportQueryBuilder{
		baseEntity: baseEntity,
		baseTable:  baseTable,
		joins:      joins,
		metrics:    metrics,
		dimensions: dimensions,
		filters:    filters,
	}

	sql := builder.buildSQL()
	confidence := builder.calculateConfidence()

	return &ReportQuery{
		Title:       fmt.Sprintf("Multi-Entity Report: %s", baseEntity),
		Description: fmt.Sprintf("Report joining %s with related entities", baseEntity),
		SQL:         sql,
		JoinPaths:   builder.buildJoinPaths(),
		Metrics:     builder.buildMetricsList(),
		Dimensions:  builder.buildDimensionsList(),
		Confidence:  confidence,
	}
}

// buildSQL generates the complete SQL query
func (builder *ReportQueryBuilder) buildSQL() string {
	var sb strings.Builder

	// SELECT clause
	sb.WriteString("SELECT\n")

	// Add dimensions
	if len(builder.dimensions) > 0 {
		for i, dim := range builder.dimensions {
			if i > 0 {
				sb.WriteString(",\n")
			}
			sb.WriteString(fmt.Sprintf("  %s.%s AS %s", dim.Table, dim.Column, dim.Name))
		}
	}

	// Add metrics
	if len(builder.metrics) > 0 {
		if len(builder.dimensions) > 0 {
			sb.WriteString(",\n")
		}
		for i, metric := range builder.metrics {
			if i > 0 {
				sb.WriteString(",\n")
			}
			sb.WriteString(fmt.Sprintf("  %s AS %s", metric.Expression, metric.Name))
		}
	}

	sb.WriteString("\nFROM " + builder.baseTable)

	// JOIN clauses
	for _, join := range builder.joins {
		sb.WriteString("\n" + join.JoinType + " JOIN " + join.TargetTable +
			" ON " + join.SourceTable + "." + join.SourceColumn +
			" = " + join.TargetTable + "." + join.TargetColumn)
	}

	// WHERE clause
	if len(builder.filters) > 0 {
		sb.WriteString("\nWHERE\n")
		for i, filter := range builder.filters {
			if i > 0 {
				sb.WriteString(" AND\n")
			}
			sb.WriteString(fmt.Sprintf("  %s.%s %s %s", filter.Table, filter.Column, filter.Operator, filter.Value))
		}
	}

	// GROUP BY clause (if metrics with aggregations)
	if builder.hasAggregations() && len(builder.dimensions) > 0 {
		sb.WriteString("\nGROUP BY\n")
		for i, dim := range builder.dimensions {
			if i > 0 {
				sb.WriteString(",\n")
			}
			sb.WriteString(fmt.Sprintf("  %s.%s", dim.Table, dim.Column))
		}
	}

	// ORDER BY clause
	sb.WriteString("\nORDER BY\n")
	if len(builder.metrics) > 0 {
		sb.WriteString(fmt.Sprintf("  %s DESC", builder.metrics[0].Name))
	} else if len(builder.dimensions) > 0 {
		sb.WriteString(fmt.Sprintf("  %s", builder.dimensions[0].Name))
	}

	return sb.String()
}

// hasAggregations checks if any metrics use aggregation functions
func (builder *ReportQueryBuilder) hasAggregations() bool {
	for _, metric := range builder.metrics {
		if strings.Contains(strings.ToUpper(metric.Expression), "SUM") ||
			strings.Contains(strings.ToUpper(metric.Expression), "AVG") ||
			strings.Contains(strings.ToUpper(metric.Expression), "COUNT") ||
			strings.Contains(strings.ToUpper(metric.Expression), "MIN") ||
			strings.Contains(strings.ToUpper(metric.Expression), "MAX") {
			return true
		}
	}
	return false
}

// calculateConfidence calculates the confidence of the generated query
func (builder *ReportQueryBuilder) calculateConfidence() float64 {
	confidence := 0.95 // Start high if we generated it

	// Reduce confidence for each join (more joins = higher complexity = lower confidence)
	for range builder.joins {
		confidence -= 0.05 // Each join reduces confidence by 5%
	}

	// Increase confidence for direct/1:1 relationships
	for _, join := range builder.joins {
		if join.Cardinality == "1:1" {
			confidence += 0.03
		}
	}

	if confidence < 0.0 {
		confidence = 0.0
	}
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// buildJoinPaths returns a human-readable description of the join paths
func (builder *ReportQueryBuilder) buildJoinPaths() []string {
	var paths []string
	for _, join := range builder.joins {
		path := fmt.Sprintf("%s.%s -> %s.%s (%s)",
			join.SourceTable,
			join.SourceColumn,
			join.TargetTable,
			join.TargetColumn,
			join.Cardinality,
		)
		paths = append(paths, path)
	}
	return paths
}

// buildMetricsList returns the list of metric names
func (builder *ReportQueryBuilder) buildMetricsList() []string {
	var metrics []string
	for _, metric := range builder.metrics {
		metrics = append(metrics, metric.Name)
	}
	return metrics
}

// buildDimensionsList returns the list of dimension names
func (builder *ReportQueryBuilder) buildDimensionsList() []string {
	var dimensions []string
	for _, dim := range builder.dimensions {
		dimensions = append(dimensions, dim.Name)
	}
	return dimensions
}

// ============================================================================
// Convenience Builders
// ============================================================================

// BuildCustomerOrderAnalysisQuery builds a sample customer-order analysis query
func BuildCustomerOrderAnalysisQuery(baseTable, customerTable, orderTable string) *ReportQuery {
	gen := NewReportingQueryGenerator("tenant-1", "datasource-1")

	joins := []JoinClause{
		{
			SourceTable:  baseTable,
			SourceColumn: "customer_id",
			TargetTable:  customerTable,
			TargetColumn: "id",
			JoinType:     "INNER",
			Cardinality:  "1:N",
		},
		{
			SourceTable:  customerTable,
			SourceColumn: "id",
			TargetTable:  orderTable,
			TargetColumn: "customer_id",
			JoinType:     "LEFT",
			Cardinality:  "1:N",
		},
	}

	metrics := []MetricDefinition{
		{
			Name:         "total_orders",
			Expression:   "COUNT(DISTINCT " + orderTable + ".id)",
			SourceTable:  orderTable,
			SourceColumn: "id",
			AggFunction:  "COUNT",
		},
		{
			Name:         "total_order_value",
			Expression:   "SUM(" + orderTable + ".amount)",
			SourceTable:  orderTable,
			SourceColumn: "amount",
			AggFunction:  "SUM",
		},
		{
			Name:         "avg_order_value",
			Expression:   "AVG(" + orderTable + ".amount)",
			SourceTable:  orderTable,
			SourceColumn: "amount",
			AggFunction:  "AVG",
		},
	}

	dimensions := []DimensionDefinition{
		{
			Name:        "customer_name",
			Table:       customerTable,
			Column:      "name",
			DataType:    "STRING",
			DisplayName: "Customer Name",
		},
		{
			Name:        "customer_segment",
			Table:       customerTable,
			Column:      "segment",
			DataType:    "STRING",
			DisplayName: "Segment",
		},
	}

	return gen.GenerateMultiEntityQuery(
		"Customer Orders",
		baseTable,
		joins,
		metrics,
		dimensions,
		[]FilterCondition{},
	)
}

// ============================================================================
// Query Templates
// ============================================================================

// GetCTETemplate returns a Common Table Expression template for hierarchical queries
func GetCTETemplate(entityName, entityTable, parentTable string) string {
	return fmt.Sprintf(`
-- %s Hierarchy Discovery
WITH RECURSIVE %s_hierarchy AS (
	-- Base case: direct relationships
	SELECT 
		source.id,
		source.name,
		target.id as parent_id,
		target.name as parent_name,
		1 as hierarchy_depth,
		source.id::text as path
	FROM %s source
	LEFT JOIN %s target ON source.parent_id = target.id
	
	UNION ALL
	
	-- Recursive case: traverse hierarchy
	SELECT 
		base.id,
		base.name,
		parent.id,
		parent.name,
		hierarchy_depth + 1,
		path || '->' || parent.id::text
	FROM %s_hierarchy base
	JOIN %s parent ON base.parent_id = parent.id
	WHERE base.hierarchy_depth < 5  -- Prevent infinite recursion
)
SELECT 
	id,
	name,
	parent_id,
	parent_name,
	hierarchy_depth,
	path
FROM %s_hierarchy
ORDER BY id, hierarchy_depth;
`, entityName, entityName, entityTable, parentTable, entityName, parentTable, entityName)
}

// GetRelationshipSQLTemplate returns a template for discovering relationships
func GetRelationshipSQLTemplate() string {
	return `
-- Discover relationships between entities
SELECT 
	er.id,
	source_ea.name as source_entity,
	target_ea.name as target_entity,
	er.relationship_type,
	er.cardinality,
	er.hierarchy_depth,
	er.fk_constraint,
	er.source_column,
	er.target_column,
	er.confidence,
	er.source_discovery_method
FROM public.entity_relationship er
JOIN public.entity_attribute source_ea ON er.source_entity_id = source_ea.id
JOIN public.entity_attribute target_ea ON er.target_entity_id = target_ea.id
WHERE er.tenant_datasource_id = $1::uuid
	AND er.is_active = true
	AND er.confidence >= $2::numeric
ORDER BY er.confidence DESC, er.hierarchy_depth ASC;
`
}

// GetSelfServiceDashboardSQL returns SQL for a self-service dashboard
func GetSelfServiceDashboardSQL(mainEntityTable, joinedEntity1Table, joinedEntity2Table string) string {
	return fmt.Sprintf(`
-- Self-Service Dashboard: Multi-Entity Aggregation
SELECT 
	me.id,
	me.name,
	me.created_at,
	COUNT(DISTINCT je1.id) as %s_count,
	COUNT(DISTINCT je2.id) as %s_count,
	SUM(je1.amount) as total_%s_amount,
	SUM(je2.amount) as total_%s_amount,
	AVG(je1.amount) as avg_%s_amount,
	AVG(je2.amount) as avg_%s_amount,
	MAX(je1.created_at) as latest_%s_date,
	MAX(je2.created_at) as latest_%s_date
FROM %s me
LEFT JOIN %s je1 ON me.id = je1.parent_id
LEFT JOIN %s je2 ON me.id = je2.parent_id
GROUP BY me.id, me.name, me.created_at
ORDER BY total_%s_amount DESC NULLS LAST
LIMIT 1000;
`,
		joinedEntity1Table,
		joinedEntity2Table,
		joinedEntity1Table,
		joinedEntity2Table,
		joinedEntity1Table,
		joinedEntity2Table,
		joinedEntity1Table,
		joinedEntity2Table,
		mainEntityTable,
		joinedEntity1Table,
		joinedEntity2Table,
		joinedEntity1Table,
	)
}
