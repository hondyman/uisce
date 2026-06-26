package reporting

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/services"
	hasuraclient "github.com/hondyman/semlayer/libs/hasura-client"
	"go.uber.org/zap"
)

// ============================================================================
// WORKDAY-STYLE REPORT BUSINESS OBJECT INTEGRATION
// ============================================================================

// WorkdayReportService integrates reporting with business objects and processes
type WorkdayReportService struct {
	reportService    *Service
	processService   *services.BusinessProcessService
	portfolioService *services.PortfolioService
	hasuraClient     *hasuraclient.HasuraClient
	logger           *zap.Logger
}

// NewWorkdayReportService creates a new Workday-style report service
func NewWorkdayReportService(
	reportService *Service,
	processService *services.BusinessProcessService,
	portfolioService *services.PortfolioService,
	hasuraClient *hasuraclient.HasuraClient,
) *WorkdayReportService {
	logger, _ := zap.NewProduction()
	return &WorkdayReportService{
		reportService:    reportService,
		processService:   processService,
		portfolioService: portfolioService,
		hasuraClient:     hasuraClient,
		logger:           logger,
	}
}

// ============================================================================
// REPORT BUSINESS OBJECT
// ============================================================================

// ReportBusinessObject represents a Workday-style report BO
type ReportBusinessObject struct {
	ID               string                `json:"id"`
	Key              string                `json:"key"`
	Name             string                `json:"name"`
	DisplayName      string                `json:"display_name"`
	Description      string                `json:"description"`
	Category         string                `json:"category"`
	ReportType       string                `json:"report_type"` // standard, composite, scheduled
	DataSource       ReportDataSource      `json:"data_source"`
	Layout           json.RawMessage       `json:"layout"`
	Parameters       []ReportParameter     `json:"parameters"`
	SemanticBindings []SemanticBinding     `json:"semantic_bindings"`
	Permissions      ReportPermissions     `json:"permissions"`
	Schedule         *ReportScheduleConfig `json:"schedule,omitempty"`
	OutputFormats    []string              `json:"output_formats"`
	IsSystem         bool                  `json:"is_system"`
	IsActive         bool                  `json:"is_active"`
	Version          int                   `json:"version"`
	CreatedAt        time.Time             `json:"created_at"`
	UpdatedAt        time.Time             `json:"updated_at"`
}

// ReportDataSource defines where report data comes from
type ReportDataSource struct {
	Type           string            `json:"type"` // cube, sql, graphql, api, composite
	CubeID         string            `json:"cube_id,omitempty"`
	Query          string            `json:"query,omitempty"`
	Dimensions     []string          `json:"dimensions,omitempty"`
	Measures       []string          `json:"measures,omitempty"`
	Filters        []DataFilter      `json:"filters,omitempty"` // Uses DataFilter from model.go
	JoinConditions []JoinCondition   `json:"join_conditions,omitempty"`
	Aggregations   map[string]string `json:"aggregations,omitempty"`
}

// JoinCondition represents a join between data sources
type JoinCondition struct {
	LeftSource  string `json:"left_source"`
	LeftField   string `json:"left_field"`
	RightSource string `json:"right_source"`
	RightField  string `json:"right_field"`
	JoinType    string `json:"join_type"` // inner, left, right, full
}

// SemanticBinding binds a report to semantic layer entities
type SemanticBinding struct {
	BusinessObjectKey string   `json:"business_object_key"`
	Fields            []string `json:"fields"`
	Relationship      string   `json:"relationship,omitempty"` // primary, referenced
	FilterField       string   `json:"filter_field,omitempty"`
}

// ReportPermissions defines access control
type ReportPermissions struct {
	ViewRoles     []string `json:"view_roles"`
	RunRoles      []string `json:"run_roles"`
	ScheduleRoles []string `json:"schedule_roles"`
	EditRoles     []string `json:"edit_roles"`
	IsPublic      bool     `json:"is_public"`
}

// ReportScheduleConfig defines scheduling options
type ReportScheduleConfig struct {
	Frequency      string                 `json:"frequency"` // daily, weekly, monthly, quarterly
	DaysOfWeek     []int                  `json:"days_of_week,omitempty"`
	DayOfMonth     int                    `json:"day_of_month,omitempty"`
	Time           string                 `json:"time"` // HH:MM
	Timezone       string                 `json:"timezone"`
	Parameters     map[string]interface{} `json:"parameters,omitempty"`
	Recipients     []RecipientConfig      `json:"recipients"`
	DeliveryMethod string                 `json:"delivery_method"` // email, portal, sftp
	IsEnabled      bool                   `json:"is_enabled"`
}

// RecipientConfig defines report recipients
type RecipientConfig struct {
	Type  string `json:"type"` // user, role, email
	Value string `json:"value"`
}

// ReportParameter defines a report input parameter
type ReportParameter struct {
	Name         string               `json:"name"`
	Label        string               `json:"label"`
	Type         string               `json:"type"` // string, number, date, daterange, select, multiselect
	Required     bool                 `json:"required"`
	DefaultValue interface{}          `json:"default_value,omitempty"`
	Options      []ParameterOption    `json:"options,omitempty"`
	DynamicLOV   *DynamicLOV          `json:"dynamic_lov,omitempty"`
	Validation   *ParameterValidation `json:"validation,omitempty"`
}

// ParameterOption for select parameters
type ParameterOption struct {
	Label string      `json:"label"`
	Value interface{} `json:"value"`
}

// DynamicLOV for dynamic list of values
type DynamicLOV struct {
	BusinessObjectKey string `json:"business_object_key"`
	LabelField        string `json:"label_field"`
	ValueField        string `json:"value_field"`
	FilterBy          string `json:"filter_by,omitempty"`
}

// ParameterValidation defines parameter validation rules
type ParameterValidation struct {
	MinValue interface{} `json:"min_value,omitempty"`
	MaxValue interface{} `json:"max_value,omitempty"`
	Pattern  string      `json:"pattern,omitempty"`
	Message  string      `json:"message,omitempty"`
}

// ============================================================================
// REPORT EXECUTION
// ============================================================================

// ReportExecutionRequest represents a request to run a report
type ReportExecutionRequest struct {
	ReportKey    string                 `json:"report_key"`
	Parameters   map[string]interface{} `json:"parameters"`
	OutputFormat string                 `json:"output_format"` // html, pdf, excel, csv, json
	Async        bool                   `json:"async"`
	Notify       bool                   `json:"notify"`
}

// ReportExecutionResult represents a report execution result
type ReportExecutionResult struct {
	ID           string                 `json:"id"`
	ReportKey    string                 `json:"report_key"`
	Status       string                 `json:"status"` // queued, running, completed, failed
	Parameters   map[string]interface{} `json:"parameters"`
	OutputFormat string                 `json:"output_format"`
	OutputURL    string                 `json:"output_url,omitempty"`
	OutputData   interface{}            `json:"output_data,omitempty"`
	RowCount     int                    `json:"row_count,omitempty"`
	GenerationMs int64                  `json:"generation_ms,omitempty"`
	StartedAt    time.Time              `json:"started_at"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty"`
	Error        string                 `json:"error,omitempty"`
	RequestedBy  string                 `json:"requested_by"`
	ProcessID    string                 `json:"process_id,omitempty"`
}

// ExecuteReport runs a report with Workday-style process integration
func (s *WorkdayReportService) ExecuteReport(ctx context.Context, req ReportExecutionRequest, requestedBy string) (*ReportExecutionResult, error) {
	// Get report definition
	report, err := s.GetReportBO(ctx, req.ReportKey)
	if err != nil {
		return nil, fmt.Errorf("report not found: %w", err)
	}

	result := &ReportExecutionResult{
		ID:           uuid.New().String(),
		ReportKey:    req.ReportKey,
		Status:       "running",
		Parameters:   req.Parameters,
		OutputFormat: req.OutputFormat,
		StartedAt:    time.Now(),
		RequestedBy:  requestedBy,
	}

	// Start business process for audit trail
	processInstance, err := s.processService.StartProcess(ctx, "report_execution", "report", result.ID, requestedBy, map[string]interface{}{
		"report_key":    req.ReportKey,
		"parameters":    req.Parameters,
		"output_format": req.OutputFormat,
	})
	if err != nil {
		s.logger.Warn("Failed to start report process", zap.Error(err))
	} else {
		result.ProcessID = processInstance.ID
	}

	// Build query from semantic bindings
	query, err := s.buildSemanticQuery(ctx, report, req.Parameters)
	if err != nil {
		result.Status = "failed"
		result.Error = err.Error()
		return result, err
	}

	// Execute query
	startTime := time.Now()
	data, rowCount, err := s.executeQuery(ctx, report.DataSource, query)
	if err != nil {
		result.Status = "failed"
		result.Error = err.Error()
		return result, err
	}

	result.OutputData = data
	result.RowCount = rowCount
	result.GenerationMs = time.Since(startTime).Milliseconds()
	result.Status = "completed"
	completedAt := time.Now()
	result.CompletedAt = &completedAt

	// Advance process to complete
	if result.ProcessID != "" {
		s.processService.AdvanceProcess(ctx, result.ProcessID, "completed", requestedBy, "", nil)
	}

	s.logger.Info("Report executed",
		zap.String("report", req.ReportKey),
		zap.Int("rows", rowCount),
		zap.Int64("ms", result.GenerationMs))

	return result, nil
}

// GetReportBO fetches a report business object
func (s *WorkdayReportService) GetReportBO(ctx context.Context, key string) (*ReportBusinessObject, error) {
	query := `
		query GetReportBO($key: String!) {
			report_business_objects(where: { key: { _eq: $key } }, limit: 1) {
				id
				key
				name
				display_name
				description
				category
				report_type
				data_source
				layout
				parameters
				semantic_bindings
				permissions
				schedule
				output_formats
				is_system
				is_active
				version
			}
		}
	`

	result, err := s.hasuraClient.Query(query, map[string]interface{}{"key": key})
	if err != nil {
		return nil, err
	}

	reports, ok := result["report_business_objects"].([]interface{})
	if !ok || len(reports) == 0 {
		return nil, fmt.Errorf("report not found: %s", key)
	}

	// Parse and return
	data := reports[0].(map[string]interface{})
	report := &ReportBusinessObject{
		ID:          data["id"].(string),
		Key:         data["key"].(string),
		Name:        getString(data, "name"),
		DisplayName: getString(data, "display_name"),
		Description: getString(data, "description"),
		Category:    getString(data, "category"),
		ReportType:  getString(data, "report_type"),
		IsSystem:    getBool(data, "is_system"),
		IsActive:    getBool(data, "is_active"),
	}

	// Parse data source
	if ds, ok := data["data_source"].(map[string]interface{}); ok {
		dsJSON, _ := json.Marshal(ds)
		json.Unmarshal(dsJSON, &report.DataSource)
	}

	// Parse semantic bindings
	if bindings, ok := data["semantic_bindings"].([]interface{}); ok {
		for _, b := range bindings {
			bJSON, _ := json.Marshal(b)
			var binding SemanticBinding
			json.Unmarshal(bJSON, &binding)
			report.SemanticBindings = append(report.SemanticBindings, binding)
		}
	}

	return report, nil
}

// ListReportBOs lists available reports by category
func (s *WorkdayReportService) ListReportBOs(ctx context.Context, category string) ([]*ReportBusinessObject, error) {
	query := `
		query ListReportBOs($category: String) {
			report_business_objects(
				where: { is_active: { _eq: true }, category: { _eq: $category } }
				order_by: { name: asc }
			) {
				id
				key
				name
				display_name
				description
				category
				report_type
				is_system
				output_formats
			}
		}
	`

	vars := map[string]interface{}{}
	if category != "" {
		vars["category"] = category
	}

	result, err := s.hasuraClient.Query(query, vars)
	if err != nil {
		return nil, err
	}

	items, ok := result["report_business_objects"].([]interface{})
	if !ok {
		return []*ReportBusinessObject{}, nil
	}

	reports := make([]*ReportBusinessObject, 0, len(items))
	for _, item := range items {
		data := item.(map[string]interface{})
		reports = append(reports, &ReportBusinessObject{
			ID:          data["id"].(string),
			Key:         getString(data, "key"),
			Name:        getString(data, "name"),
			DisplayName: getString(data, "display_name"),
			Description: getString(data, "description"),
			Category:    getString(data, "category"),
			ReportType:  getString(data, "report_type"),
			IsSystem:    getBool(data, "is_system"),
		})
	}

	return reports, nil
}

// buildSemanticQuery builds a query from semantic bindings
func (s *WorkdayReportService) buildSemanticQuery(ctx context.Context, report *ReportBusinessObject, params map[string]interface{}) (string, error) {
	ds := report.DataSource

	switch ds.Type {
	case "cube":
		// Build Cube.js query
		return s.buildCubeQuery(ds, params)
	case "graphql":
		// Build GraphQL query from semantic bindings
		return s.buildGraphQLQuery(report.SemanticBindings, params)
	case "sql":
		// Return SQL with parameter substitution
		return s.buildSQLQuery(ds.Query, params)
	default:
		return "", fmt.Errorf("unsupported data source type: %s", ds.Type)
	}
}

func (s *WorkdayReportService) buildCubeQuery(ds ReportDataSource, params map[string]interface{}) (string, error) {
	query := map[string]interface{}{
		"dimensions": ds.Dimensions,
		"measures":   ds.Measures,
	}

	// Add filters from parameters
	filters := []map[string]interface{}{}
	for _, f := range ds.Filters {
		filter := map[string]interface{}{
			"member":   f.Dimension,
			"operator": f.Operator,
		}
		// Check if parameter reference
		if f.Parameter != "" {
			if paramValue, exists := params[f.Parameter]; exists {
				filter["values"] = []interface{}{paramValue}
			}
		} else if f.Value != nil {
			filter["values"] = []interface{}{f.Value}
		}
		filters = append(filters, filter)
	}
	query["filters"] = filters

	jsonQuery, _ := json.Marshal(query)
	return string(jsonQuery), nil
}

func (s *WorkdayReportService) buildGraphQLQuery(bindings []SemanticBinding, params map[string]interface{}) (string, error) {
	// Build GraphQL query from semantic bindings
	if len(bindings) == 0 {
		return "", fmt.Errorf("no semantic bindings defined")
	}

	primary := bindings[0]
	fieldList := ""
	for i, f := range primary.Fields {
		if i > 0 {
			fieldList += " "
		}
		fieldList += f
	}

	query := fmt.Sprintf(`query { %s { %s } }`, primary.BusinessObjectKey, fieldList)
	return query, nil
}

func (s *WorkdayReportService) buildSQLQuery(template string, params map[string]interface{}) (string, error) {
	// Simple parameter substitution (production should use prepared statements)
	query := template
	for k, v := range params {
		placeholder := fmt.Sprintf("${%s}", k)
		valueStr := fmt.Sprintf("'%v'", v)
		query = replaceAll(query, placeholder, valueStr)
	}
	return query, nil
}

func (s *WorkdayReportService) executeQuery(ctx context.Context, ds ReportDataSource, query string) (interface{}, int, error) {
	// This would call the appropriate data source
	// For now, return placeholder
	return map[string]interface{}{"message": "Query executed", "query": query}, 0, nil
}

// Helper functions are already defined in portfolio_service.go
func replaceAll(s, old, new string) string {
	result := s
	for {
		i := -1
		for j := 0; j <= len(result)-len(old); j++ {
			if result[j:j+len(old)] == old {
				i = j
				break
			}
		}
		if i < 0 {
			break
		}
		result = result[:i] + new + result[i+len(old):]
	}
	return result
}

// getString safely extracts a string from map
func getString(data map[string]interface{}, key string) string {
	if v, ok := data[key].(string); ok {
		return v
	}
	return ""
}

// getBool safely extracts a bool from map
func getBool(data map[string]interface{}, key string) bool {
	if v, ok := data[key].(bool); ok {
		return v
	}
	return false
}
