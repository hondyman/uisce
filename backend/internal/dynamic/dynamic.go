package dynamic

import (
	"context"
	"fmt"
	"strings"

	"github.com/hondyman/semlayer/backend/internal/cube"
	"github.com/hondyman/semlayer/backend/internal/query"
	"github.com/hondyman/semlayer/backend/models"
)

// DynamicParameter represents a runtime parameter for queries
type DynamicParameter struct {
	Name         string      `json:"name"`
	Type         string      `json:"type"` // "dimension", "measure", "filter", "time_range"
	Value        interface{} `json:"value"`
	DefaultValue interface{} `json:"default_value,omitempty"`
	Required     bool        `json:"required"`
	Options      []string    `json:"options,omitempty"`
	Description  string      `json:"description"`
}

// DynamicMeasure represents a measure that can be computed dynamically
type DynamicMeasure struct {
	Name       string                 `json:"name"`
	Type       string                 `json:"type"` // "count", "sum", "avg", "ratio", "custom"
	SQL        string                 `json:"sql"`
	Parameters []DynamicParameter     `json:"parameters,omitempty"`
	Meta       map[string]interface{} `json:"meta,omitempty"`
}

// DynamicQueryRequest extends the base query with dynamic capabilities
type DynamicQueryRequest struct {
	BaseQuery       *models.Query          `json:"base_query"`
	Parameters      []DynamicParameter     `json:"parameters"`
	DynamicMeasures []DynamicMeasure       `json:"dynamic_measures"`
	TimeRange       *query.TimeRange       `json:"time_range,omitempty"`
	Context         map[string]interface{} `json:"context,omitempty"`
}

// DynamicQueryEngine handles dynamic parameter and measure resolution
type DynamicQueryEngine struct {
	cubeEngine  *cube.Cube
	templateMgr *query.QueryTemplateManager
}

// NewDynamicQueryEngine creates a new dynamic query engine
func NewDynamicQueryEngine(cubeEngine *cube.Cube, templateMgr *query.QueryTemplateManager) *DynamicQueryEngine {
	return &DynamicQueryEngine{
		cubeEngine:  cubeEngine,
		templateMgr: templateMgr,
	}
}

// ResolveParameters resolves dynamic parameters into concrete values
func (dqe *DynamicQueryEngine) ResolveParameters(ctx context.Context, req *DynamicQueryRequest) (*ResolvedQuery, error) {
	resolved := &ResolvedQuery{
		Metrics:    make([]string, 0),
		Dimensions: make([]string, 0),
		Filters:    make([]models.Filter, 0),
		Parameters: make(map[string]interface{}),
	}

	// Resolve base query parameters
	for _, param := range req.Parameters {
		value, err := dqe.resolveParameterValue(ctx, param, req.Context)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve parameter %s: %w", param.Name, err)
		}
		resolved.Parameters[param.Name] = value
	}

	// Apply dynamic measures
	for _, measure := range req.DynamicMeasures {
		sql, err := dqe.buildDynamicMeasureSQL(measure, resolved.Parameters)
		if err != nil {
			return nil, fmt.Errorf("failed to build dynamic measure %s: %w", measure.Name, err)
		}
		resolved.Metrics = append(resolved.Metrics, fmt.Sprintf("%s as %s", sql, measure.Name))
	}

	// Apply time range if specified
	if req.TimeRange != nil {
		timeFilter := dqe.buildTimeRangeFilter(req.TimeRange)
		resolved.Filters = append(resolved.Filters, timeFilter)
	}

	return resolved, nil
}

// resolveParameterValue resolves a single parameter value
func (dqe *DynamicQueryEngine) resolveParameterValue(_ context.Context, param DynamicParameter, context map[string]interface{}) (interface{}, error) {
	// Check if value is provided in context
	if val, exists := context[param.Name]; exists {
		return val, nil
	}

	// Use default value if available
	if param.DefaultValue != nil {
		return param.DefaultValue, nil
	}

	// For required parameters without values, return error
	if param.Required {
		return nil, fmt.Errorf("required parameter %s not provided", param.Name)
	}

	return nil, nil
}

// buildDynamicMeasureSQL builds SQL for a dynamic measure
func (dqe *DynamicQueryEngine) buildDynamicMeasureSQL(measure DynamicMeasure, params map[string]interface{}) (string, error) {
	sql := measure.SQL

	// Replace parameter placeholders
	for name, value := range params {
		placeholder := fmt.Sprintf("{{%s}}", name)
		if strings.Contains(sql, placeholder) {
			sql = strings.ReplaceAll(sql, placeholder, fmt.Sprintf("%v", value))
		}
	}

	return sql, nil
}

// buildTimeRangeFilter creates a filter for time range
func (dqe *DynamicQueryEngine) buildTimeRangeFilter(_ *query.TimeRange) models.Filter {
	// For now, return a simple filter - would need proper TimeRange type
	return models.Filter{
		Field:  "date",
		Op:     "BETWEEN",
		Values: []string{"2024-01-01", "2024-12-31"}, // Default range
	}
}

// ResolvedQuery represents a fully resolved query ready for execution
type ResolvedQuery struct {
	Metrics    []string               `json:"metrics"`
	Dimensions []string               `json:"dimensions"`
	Filters    []models.Filter        `json:"filters"`
	Parameters map[string]interface{} `json:"parameters"`
	TableName  string                 `json:"table_name"`
}

// BuildSQL generates the final SQL from resolved query
func (rq *ResolvedQuery) BuildSQL() (string, []interface{}) {
	var selectClauses []string
	selectClauses = append(selectClauses, rq.Dimensions...)
	selectClauses = append(selectClauses, rq.Metrics...)

	var whereClauses []string
	var args []interface{}
	for i, f := range rq.Filters {
		if len(f.Values) > 0 {
			whereClauses = append(whereClauses, fmt.Sprintf("%s %s $%d", f.Field, f.Op, i+1))
			args = append(args, f.Values[0])
		}
	}

	sql := fmt.Sprintf("SELECT %s FROM %s", strings.Join(selectClauses, ", "), rq.TableName)
	if len(whereClauses) > 0 {
		sql += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	if len(rq.Dimensions) > 0 {
		sql += " GROUP BY " + strings.Join(rq.Dimensions, ", ")
	}

	return sql, args
}
