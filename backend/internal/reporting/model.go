package reporting

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// REPORT DEFINITION MODELS
// ============================================================================

// ReportDefinition represents a report template (metadata-first)
type ReportDefinition struct {
	ID                 uuid.UUID `db:"id" json:"id"`
	TenantID           uuid.UUID `db:"tenant_id" json:"tenant_id"`
	TenantDatasourceID uuid.UUID `db:"tenant_datasource_id" json:"tenant_datasource_id"`

	// Identity
	ReportKey   string   `db:"report_key" json:"report_key"`
	DisplayName string   `db:"display_name" json:"display_name"`
	Description string   `db:"description" json:"description,omitempty"`
	Category    string   `db:"category" json:"category,omitempty"`
	Tags        []string `db:"-" json:"tags,omitempty"`
	TagsJSON    []byte   `db:"tags" json:"-"`

	// Type/Classification
	ReportType        string   `db:"report_type" json:"report_type"`
	OutputFormats     []string `db:"-" json:"output_formats"`
	OutputFormatsJSON []byte   `db:"output_formats" json:"-"`

	// Definition (metadata-first)
	Definition       *ReportLayout `db:"-" json:"definition"`
	DefinitionJSON   []byte        `db:"definition" json:"-"`
	ParametersSchema []Parameter   `db:"-" json:"parameters_schema,omitempty"`
	ParametersJSON   []byte        `db:"parameters_schema" json:"-"`

	// Semantic Layer Binding
	SemanticCubeID *uuid.UUID      `db:"semantic_cube_id" json:"semantic_cube_id,omitempty"`
	SemanticQuery  json.RawMessage `db:"semantic_query" json:"semantic_query,omitempty"`

	// Versioning
	Version           int        `db:"version" json:"version"`
	IsCurrent         bool       `db:"is_current" json:"is_current"`
	PreviousVersionID *uuid.UUID `db:"previous_version_id" json:"previous_version_id,omitempty"`

	// Ownership
	IsCore       bool       `db:"is_core" json:"is_core"`
	BaseReportID *uuid.UUID `db:"base_report_id" json:"base_report_id,omitempty"`

	// Lifecycle
	Status      string     `db:"status" json:"status"`
	PublishedAt *time.Time `db:"published_at" json:"published_at,omitempty"`
	PublishedBy *uuid.UUID `db:"published_by" json:"published_by,omitempty"`

	// Audit
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	CreatedBy *uuid.UUID `db:"created_by" json:"created_by,omitempty"`
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`
	UpdatedBy *uuid.UUID `db:"updated_by" json:"updated_by,omitempty"`
}

// ReportExtension represents a tenant customization of a core report
type ReportExtension struct {
	ID                 uuid.UUID `db:"id" json:"id"`
	TenantID           uuid.UUID `db:"tenant_id" json:"tenant_id"`
	TenantDatasourceID uuid.UUID `db:"tenant_datasource_id" json:"tenant_datasource_id"`

	// Link to core
	BaseReportID uuid.UUID `db:"base_report_id" json:"base_report_id"`

	// Extension definition
	ExtensionKey  string `db:"extension_key" json:"extension_key"`
	ExtensionName string `db:"extension_name" json:"extension_name,omitempty"`
	Description   string `db:"description" json:"description,omitempty"`

	// What's customized
	ExtensionDefinition json.RawMessage `db:"extension_definition" json:"extension_definition"`
	Overrides           json.RawMessage `db:"overrides" json:"overrides,omitempty"`
	Additions           json.RawMessage `db:"additions" json:"additions,omitempty"`
	Removals            json.RawMessage `db:"removals" json:"removals,omitempty"`

	// Parameter overrides
	ParameterDefaults json.RawMessage `db:"parameter_defaults" json:"parameter_defaults,omitempty"`

	// Versioning
	Version           int  `db:"version" json:"version"`
	IsCurrent         bool `db:"is_current" json:"is_current"`
	CoreVersionTarget *int `db:"core_version_target" json:"core_version_target,omitempty"`

	// Lifecycle
	Status      string     `db:"status" json:"status"`
	PublishedAt *time.Time `db:"published_at" json:"published_at,omitempty"`
	PublishedBy *uuid.UUID `db:"published_by" json:"published_by,omitempty"`

	// Audit
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	CreatedBy *uuid.UUID `db:"created_by" json:"created_by,omitempty"`
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`
	UpdatedBy *uuid.UUID `db:"updated_by" json:"updated_by,omitempty"`
}

// ReportInstance represents a generated report
type ReportInstance struct {
	ID                 uuid.UUID `db:"id" json:"id"`
	TenantID           uuid.UUID `db:"tenant_id" json:"tenant_id"`
	TenantDatasourceID uuid.UUID `db:"tenant_datasource_id" json:"tenant_datasource_id"`

	// Source definition
	ReportDefinitionID uuid.UUID  `db:"report_definition_id" json:"report_definition_id"`
	ReportExtensionID  *uuid.UUID `db:"report_extension_id" json:"report_extension_id,omitempty"`

	// Merged definition (snapshot at render time)
	MergedDefinition json.RawMessage `db:"merged_definition" json:"merged_definition,omitempty"`

	// Context (what entity the report is for)
	ContextType string     `db:"context_type" json:"context_type,omitempty"`
	ContextID   *uuid.UUID `db:"context_id" json:"context_id,omitempty"`
	ContextName string     `db:"context_name" json:"context_name,omitempty"`

	// Parameters used
	Parameters json.RawMessage `db:"parameters" json:"parameters,omitempty"`

	// Generated content
	OutputFormat   string          `db:"output_format" json:"output_format"`
	OutputData     []byte          `db:"output_data" json:"-"`
	OutputURL      string          `db:"output_url" json:"output_url,omitempty"`
	OutputMetadata json.RawMessage `db:"output_metadata" json:"output_metadata,omitempty"`

	// Lifecycle
	Status       string `db:"status" json:"status"`
	ErrorMessage string `db:"error_message" json:"error_message,omitempty"`

	// Timing
	RequestedAt      time.Time  `db:"requested_at" json:"requested_at"`
	StartedAt        *time.Time `db:"started_at" json:"started_at,omitempty"`
	CompletedAt      *time.Time `db:"completed_at" json:"completed_at,omitempty"`
	ExpiresAt        *time.Time `db:"expires_at" json:"expires_at,omitempty"`
	GenerationTimeMs *int       `db:"generation_time_ms" json:"generation_time_ms,omitempty"`

	// Requester
	RequestedBy *uuid.UUID `db:"requested_by" json:"requested_by,omitempty"`

	// Audit
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// ReportSchedule represents a recurring report job
type ReportSchedule struct {
	ID                 uuid.UUID `db:"id" json:"id"`
	TenantID           uuid.UUID `db:"tenant_id" json:"tenant_id"`
	TenantDatasourceID uuid.UUID `db:"tenant_datasource_id" json:"tenant_datasource_id"`

	// Report definition
	ReportDefinitionID uuid.UUID  `db:"report_definition_id" json:"report_definition_id"`
	ReportExtensionID  *uuid.UUID `db:"report_extension_id" json:"report_extension_id,omitempty"`

	// Schedule
	ScheduleName   string `db:"schedule_name" json:"schedule_name"`
	Description    string `db:"description" json:"description,omitempty"`
	CronExpression string `db:"cron_expression" json:"cron_expression"`
	Timezone       string `db:"timezone" json:"timezone"`

	// Parameters template
	ParametersTemplate json.RawMessage `db:"parameters_template" json:"parameters_template,omitempty"`

	// Context
	ContextType    string          `db:"context_type" json:"context_type,omitempty"`
	ContextQuery   json.RawMessage `db:"context_query" json:"context_query,omitempty"`
	FixedContextID *uuid.UUID      `db:"fixed_context_id" json:"fixed_context_id,omitempty"`

	// Output
	OutputFormats json.RawMessage `db:"output_formats" json:"output_formats"`

	// Delivery
	DeliveryConfig json.RawMessage `db:"delivery_config" json:"delivery_config,omitempty"`

	// State
	IsActive      bool       `db:"is_active" json:"is_active"`
	LastRunAt     *time.Time `db:"last_run_at" json:"last_run_at,omitempty"`
	LastRunStatus string     `db:"last_run_status" json:"last_run_status,omitempty"`
	LastRunError  string     `db:"last_run_error" json:"last_run_error,omitempty"`
	NextRunAt     *time.Time `db:"next_run_at" json:"next_run_at,omitempty"`
	RunCount      int        `db:"run_count" json:"run_count"`

	// Audit
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	CreatedBy *uuid.UUID `db:"created_by" json:"created_by,omitempty"`
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`
}

// ReportPackage represents a bundle of report templates for tenant provisioning
type ReportPackage struct {
	ID          uuid.UUID `db:"id" json:"id"`
	PackageKey  string    `db:"package_key" json:"package_key"`
	DisplayName string    `db:"display_name" json:"display_name"`
	Description string    `db:"description" json:"description,omitempty"`
	Category    string    `db:"category" json:"category,omitempty"`

	ReportDefinitions json.RawMessage `db:"report_definitions" json:"report_definitions"`
	DefaultSchedules  json.RawMessage `db:"default_schedules" json:"default_schedules,omitempty"`
	RequiredCubes     json.RawMessage `db:"required_cubes" json:"required_cubes,omitempty"`

	Version   string    `db:"version" json:"version"`
	IsActive  bool      `db:"is_active" json:"is_active"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// ============================================================================
// REPORT LAYOUT DEFINITION (Metadata-first schema)
// ============================================================================

// ReportLayout represents the complete report layout definition
type ReportLayout struct {
	Metadata          ReportMetadata              `json:"metadata"`
	Parameters        []Parameter                 `json:"parameters,omitempty"`
	DataBindings      map[string]DataBinding      `json:"dataBindings"`
	Layout            Layout                      `json:"layout"`
	ConditionalStyles map[string]ConditionalStyle `json:"conditionalStyles,omitempty"`
	DrillDown         map[string]DrillDownConfig  `json:"drillDown,omitempty"`
	Exports           ExportConfig                `json:"exports,omitempty"`
}

// ReportMetadata contains report identification info
type ReportMetadata struct {
	Key         string   `json:"key"`
	DisplayName string   `json:"displayName"`
	Description string   `json:"description,omitempty"`
	Category    string   `json:"category,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// Parameter defines a report input parameter
type Parameter struct {
	Name        string           `json:"name"`
	Type        string           `json:"type"` // string, number, date, dateRange, boolean, select, multiSelect
	Label       string           `json:"label"`
	Description string           `json:"description,omitempty"`
	Required    bool             `json:"required,omitempty"`
	Default     interface{}      `json:"default,omitempty"`
	Options     []SelectOption   `json:"options,omitempty"`
	DataSource  *ParamDataSource `json:"dataSource,omitempty"`
	Validation  *ParamValidation `json:"validation,omitempty"`
}

// SelectOption for dropdown parameters
type SelectOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// ParamDataSource for dynamic parameter values
type ParamDataSource struct {
	Cube             string `json:"cube"`
	Dimension        string `json:"dimension"`
	DisplayDimension string `json:"displayDimension,omitempty"`
	Filter           string `json:"filter,omitempty"`
}

// ParamValidation for parameter constraints
type ParamValidation struct {
	Min     interface{} `json:"min,omitempty"`
	Max     interface{} `json:"max,omitempty"`
	Pattern string      `json:"pattern,omitempty"`
	Message string      `json:"message,omitempty"`
}

// DataBinding defines how a report section connects to semantic data
type DataBinding struct {
	Cube          string            `json:"cube"`
	Measures      []string          `json:"measures,omitempty"`
	Dimensions    []string          `json:"dimensions,omitempty"`
	Filters       []DataFilter      `json:"filters,omitempty"`
	TimeDimension *TimeDimension    `json:"timeDimension,omitempty"`
	Order         map[string]string `json:"order,omitempty"`
	Limit         int               `json:"limit,omitempty"`
	Conditional   *ConditionalBind  `json:"conditional,omitempty"`
}

// DataFilter for cube queries
type DataFilter struct {
	Dimension string      `json:"dimension"`
	Operator  string      `json:"operator"` // equals, notEquals, contains, gt, lt, between, etc.
	Value     interface{} `json:"value,omitempty"`
	Parameter string      `json:"parameter,omitempty"` // Use parameter value
}

// TimeDimension for time-series data
type TimeDimension struct {
	Dimension   string        `json:"dimension"`
	Granularity string        `json:"granularity,omitempty"` // day, week, month, quarter, year
	DateRange   *DateRangeRef `json:"dateRange,omitempty"`
}

// DateRangeRef references a parameter for date range
type DateRangeRef struct {
	Parameter string `json:"parameter,omitempty"`
	Value     string `json:"value,omitempty"` // e.g., "last_30_days", "this_quarter"
}

// ConditionalBind for conditional data binding
type ConditionalBind struct {
	Parameter string      `json:"parameter"`
	Equals    interface{} `json:"equals"`
}

// Layout defines the report structure
type Layout struct {
	PageSettings PageSettings `json:"pageSettings"`
	Header       *Section     `json:"header,omitempty"`
	Footer       *Section     `json:"footer,omitempty"`
	Body         Body         `json:"body"`
}

// PageSettings for PDF output
type PageSettings struct {
	Size        string  `json:"size"`        // letter, a4, legal
	Orientation string  `json:"orientation"` // portrait, landscape
	Margins     Margins `json:"margins"`
}

// Margins in points (72 points = 1 inch)
type Margins struct {
	Top    int `json:"top"`
	Right  int `json:"right"`
	Bottom int `json:"bottom"`
	Left   int `json:"left"`
}

// Section for header/footer
type Section struct {
	Height   int       `json:"height"`
	Elements []Element `json:"elements"`
}

// Body contains report sections
type Body struct {
	Sections []ReportSection `json:"sections"`
}

// ReportSection is a container for report elements
type ReportSection struct {
	ID              string        `json:"id"`
	Type            string        `json:"type"` // summary, table, chart, text, group
	Title           string        `json:"title,omitempty"`
	DataBinding     string        `json:"dataBinding,omitempty"`
	DataBindingDef  *DataBinding  `json:"dataBindingDef,omitempty"` // Inline definition
	Elements        []Element     `json:"elements,omitempty"`
	Columns         []TableColumn `json:"columns,omitempty"` // For tables
	GroupBy         []string      `json:"groupBy,omitempty"`
	Subtotals       bool          `json:"subtotals,omitempty"`
	GrandTotal      bool          `json:"grandTotal,omitempty"`
	ChartConfig     *ChartConfig  `json:"chartConfig,omitempty"`
	PageBreakBefore bool          `json:"pageBreakBefore,omitempty"`
	PageBreakAfter  bool          `json:"pageBreakAfter,omitempty"`
	InsertAfter     string        `json:"insertAfter,omitempty"` // For extensions
}

// Element is a visual component in a section
type Element struct {
	Type     string           `json:"type"` // text, image, kpiCard, chart, table, rectangle, line
	Content  string           `json:"content,omitempty"`
	Src      string           `json:"src,omitempty"` // For images
	Position *ElementPosition `json:"position,omitempty"`
	Size     *Size            `json:"size,omitempty"`
	Style    json.RawMessage  `json:"style,omitempty"`

	// KPI Card specific
	Title          string `json:"title,omitempty"`
	Value          string `json:"value,omitempty"`
	Change         string `json:"change,omitempty"`
	ChangeType     string `json:"changeType,omitempty"`
	Benchmark      string `json:"benchmark,omitempty"`
	BenchmarkLabel string `json:"benchmarkLabel,omitempty"`

	// Expression/binding
	Format string `json:"format,omitempty"` // currency, percent, number, date
}

// ElementPosition for absolute positioning of report elements
type ElementPosition struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// Size for element dimensions
type Size struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// TableColumn defines a table column
type TableColumn struct {
	Dimension        string `json:"dimension,omitempty"`
	Measure          string `json:"measure,omitempty"`
	Label            string `json:"label"`
	Width            int    `json:"width,omitempty"`
	Format           string `json:"format,omitempty"`
	Alignment        string `json:"alignment,omitempty"` // left, center, right
	ConditionalStyle string `json:"conditionalStyle,omitempty"`
}

// ChartConfig for chart sections
type ChartConfig struct {
	Type   string         `json:"type"` // line, bar, pie, area, donut
	XAxis  *AxisConfig    `json:"xAxis,omitempty"`
	YAxis  *AxisConfig    `json:"yAxis,omitempty"`
	Series []SeriesConfig `json:"series,omitempty"`
	Legend bool           `json:"legend,omitempty"`
}

// AxisConfig for chart axes
type AxisConfig struct {
	Dimension string `json:"dimension,omitempty"`
	Measure   string `json:"measure,omitempty"`
	Label     string `json:"label,omitempty"`
	Format    string `json:"format,omitempty"`
}

// SeriesConfig for chart data series
type SeriesConfig struct {
	DataBinding string `json:"dataBinding,omitempty"`
	Measure     string `json:"measure"`
	Label       string `json:"label"`
	Color       string `json:"color,omitempty"`
	Conditional string `json:"conditional,omitempty"` // Parameter to check
}

// ConditionalStyle defines conditional formatting
type ConditionalStyle struct {
	Positive map[string]string `json:"positive,omitempty"`
	Negative map[string]string `json:"negative,omitempty"`
	Zero     map[string]string `json:"zero,omitempty"`
}

// DrillDownConfig for report navigation
type DrillDownConfig struct {
	TargetReport string            `json:"targetReport"`
	Parameters   map[string]string `json:"parameters"`
}

// ExportConfig for export options
type ExportConfig struct {
	PDF   *PDFExportConfig   `json:"pdf,omitempty"`
	Excel *ExcelExportConfig `json:"excel,omitempty"`
	HTML  *HTMLExportConfig  `json:"html,omitempty"`
}

// PDFExportConfig for PDF export
type PDFExportConfig struct {
	Enabled   bool   `json:"enabled"`
	Watermark string `json:"watermark,omitempty"`
}

// ExcelExportConfig for Excel export
type ExcelExportConfig struct {
	Enabled     bool `json:"enabled"`
	IncludeData bool `json:"includeData,omitempty"`
}

// HTMLExportConfig for HTML export
type HTMLExportConfig struct {
	Enabled     bool `json:"enabled"`
	Interactive bool `json:"interactive,omitempty"`
}

// ============================================================================
// REQUEST/RESPONSE TYPES
// ============================================================================

// CreateReportDefinitionRequest for creating a new report
type CreateReportDefinitionRequest struct {
	ReportKey   string        `json:"report_key" binding:"required"`
	DisplayName string        `json:"display_name" binding:"required"`
	Description string        `json:"description"`
	Category    string        `json:"category"`
	Tags        []string      `json:"tags"`
	ReportType  string        `json:"report_type"`
	Definition  *ReportLayout `json:"definition" binding:"required"`
	IsCore      bool          `json:"is_core"`
}

// CreateReportExtensionRequest for creating an extension
type CreateReportExtensionRequest struct {
	BaseReportID      uuid.UUID       `json:"base_report_id" binding:"required"`
	ExtensionKey      string          `json:"extension_key" binding:"required"`
	ExtensionName     string          `json:"extension_name"`
	Description       string          `json:"description"`
	Overrides         json.RawMessage `json:"overrides"`
	Additions         json.RawMessage `json:"additions"`
	Removals          json.RawMessage `json:"removals"`
	ParameterDefaults json.RawMessage `json:"parameter_defaults"`
}

// RenderReportRequest for generating a report
type RenderReportRequest struct {
	ReportDefinitionID uuid.UUID       `json:"report_definition_id" binding:"required"`
	ReportExtensionID  *uuid.UUID      `json:"report_extension_id"`
	OutputFormat       string          `json:"output_format" binding:"required"` // pdf, html, excel
	ContextType        string          `json:"context_type"`
	ContextID          *uuid.UUID      `json:"context_id"`
	ContextName        string          `json:"context_name"`
	Parameters         json.RawMessage `json:"parameters"`
}

// ProvisionReportsRequest for one-click tenant setup
type ProvisionReportsRequest struct {
	TenantID           uuid.UUID       `json:"tenant_id" binding:"required"`
	TenantDatasourceID uuid.UUID       `json:"tenant_datasource_id" binding:"required"`
	PackageKey         string          `json:"package_key" binding:"required"`
	CustomOptions      json.RawMessage `json:"custom_options,omitempty"`
}

// ProvisionReportsResponse returns provisioning results
type ProvisionReportsResponse struct {
	Success          bool        `json:"success"`
	ReportsCreated   int         `json:"reports_created"`
	SchedulesCreated int         `json:"schedules_created"`
	Errors           []string    `json:"errors,omitempty"`
	CreatedReportIDs []uuid.UUID `json:"created_report_ids"`
}
