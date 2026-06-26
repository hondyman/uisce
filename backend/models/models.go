package models

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/cube"
)

// Query represents a user's request for data from the frontend.
type Query struct {
	Metrics    []string `json:"metrics"`
	Dimensions []string `json:"dimensions"`
	Filters    []Filter `json:"filters"`
	TableName  string   `json:"table_name"`
}

// BuildSQL constructs a SQL query string and its arguments from a Query struct.
func (q *Query) BuildSQL() (string, []interface{}) {
	var selectClauses []string
	if len(q.Dimensions) == 0 && len(q.Metrics) == 0 {
		selectClauses = append(selectClauses, "*")
	} else {
		selectClauses = append(selectClauses, q.Dimensions...)
		selectClauses = append(selectClauses, q.Metrics...)
	}

	var whereClauses []string
	var args []interface{}
	for i, f := range q.Filters {
		// This logic is simplified to handle the common case for the old query model.
		// A full implementation would handle IN, BETWEEN, etc.
		if len(f.Values) > 0 {
			// Use placeholder based on argument position
			whereClauses = append(whereClauses, fmt.Sprintf("%s %s $%d", f.Field, f.Op, i+1))
			args = append(args, f.Values[0])
		}
	}

	sql := fmt.Sprintf("SELECT %s FROM %s", strings.Join(selectClauses, ", "), q.TableName)
	if len(whereClauses) > 0 {
		sql += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	if len(q.Dimensions) > 0 {
		sql += " GROUP BY " + strings.Join(q.Dimensions, ", ")
	}

	return sql, args
}

// CubeConfigYAML is the top-level structure for a semantic layer configuration file.
type CubeConfigYAML struct {
	Cubes []CubeYAML `yaml:"cubes"`
}

// CubeYAML corresponds to a single "cube" in the YAML, defining a logical data model.
type CubeYAML struct {
	Name       string          `yaml:"name"`
	SQL        string          `yaml:"sql"`
	Joins      []JoinYAML      `yaml:"joins"`
	Dimensions []DimensionYAML `yaml:"dimensions"`
	Measures   []MeasureYAML   `yaml:"measures"`
}

// JoinYAML corresponds to a join in a cube.
type JoinYAML struct {
	Name string `yaml:"name"`
	SQL  string `yaml:"sql"`
}

// DimensionYAML corresponds to a dimension in a cube.
type DimensionYAML struct {
	Name string `yaml:"name"`
	SQL  string `yaml:"sql"`
	Type string `yaml:"type"`
}

// MeasureYAML corresponds to a measure in a cube.
type MeasureYAML struct {
	Name string `yaml:"name"`
	SQL  string `yaml:"sql"`
	Type string `yaml:"type"`
}

// RawTableSchema defines the structure for representing a raw database table.
type RawTableSchema struct {
	Name    string            `json:"name"`
	Columns []RawColumnSchema `json:"columns"`
}

// RawColumnSchema defines the structure for a column in a raw database table.
type RawColumnSchema struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// ConnectionDetails holds all the necessary fields for testing a database connection.
type ConnectionDetails struct {
	Type   string `json:"type"`
	Host   string `json:"host"`
	Port   string `json:"port"`
	User   string `json:"user"`
	Pass   string `json:"pass"`
	DBName string `json:"dbname"`
	SSL    string `json:"ssl"`
}

// TenantProductDatasource represents a configured datasource for a tenant's product.
type TenantProductDatasource struct {
	ID                uuid.UUID `json:"id" db:"id"`
	DatasourceID      uuid.UUID `json:"datasource_id" db:"datasource_id"`
	TenantID          uuid.UUID `json:"tenant_id" db:"tenant_id"`
	AlphaProductID    uuid.UUID `json:"alpha_product_id" db:"alpha_product_id"`
	AlphaDatasourceID uuid.UUID `json:"alpha_datasource_id" db:"alpha_datasource_id"`
	Name              string    `json:"name" db:"name"`
	DatasourceCode    string    `json:"datasource_code" db:"datasource_code"`
	Config            []byte    `json:"config" db:"config"` // For jsonb
	// TenantGoldCopy is populated from the parent tenant's `gold_copy` flag.
	TenantGoldCopy bool `json:"tenant_gold_copy" db:"tenant_gold_copy"`
}

// Auth holds authentication details, typically nested within a connection config.
type Auth struct {
	Basic struct {
		Username string `json:"username"`
		Password string `json:"password"`
	} `json:"basic"`
}

// --- Upgrade-Safe Modeling Blueprint Structs ---

// --- Diff and Comparison Structs ---

// ChangeSeverity defines the impact of a change.
type ChangeSeverity string

const (
	SeverityLow      ChangeSeverity = "low"
	SeverityMedium   ChangeSeverity = "medium"
	SeverityBreaking ChangeSeverity = "breaking"
)

const (
	StatusDraft     = "draft"
	StatusPublished = "published"
	StatusArchived  = "archived"
)

// Change represents a single, classified modification between two snapshots.
type Change struct {
	NodeType      string         `json:"node_type"`
	QualifiedPath string         `json:"qualified_path"`
	ChangeType    string         `json:"change_type"`
	Severity      ChangeSeverity `json:"severity"`
	Details       string         `json:"details"`
}

// -- Upgrade & Diff Reporting Types --

// FieldChange represents a single field-level change detected during a model diff.
type FieldChange struct {
	ID             string `json:"id"`
	Model          string `json:"model,omitempty"`
	Path           string `json:"path"`
	ChangeType     string `json:"change_type,omitempty"`
	Before         string `json:"before,omitempty"`
	After          string `json:"after,omitempty"`
	RuleID         string `json:"rule_id,omitempty"`
	Provenance     string `json:"provenance,omitempty"`
	SelectionState string `json:"selection_state,omitempty"`
	RuleLink       string `json:"rule_link,omitempty"`
	Reason         string `json:"reason,omitempty"`
}

// ModelDiff describes the set of field changes for a single semantic model.
type ModelDiff struct {
	Model        string        `json:"model"`
	ChangeType   string        `json:"change_type"`
	FieldChanges []FieldChange `json:"field_changes"`
}

// ModelUpgradeAudit represents an audit record for a reviewer decision on a field change.
type ModelUpgradeAudit struct {
	ID         uuid.UUID `json:"id" db:"id"`
	DiffID     uuid.UUID `json:"diff_id" db:"diff_id"`
	ModelName  string    `json:"model_name" db:"model_name"`
	FieldPath  string    `json:"field_path" db:"field_path"`
	RuleID     string    `json:"rule_id" db:"rule_id"`
	Provenance string    `json:"provenance" db:"provenance"`
	Decision   string    `json:"decision" db:"decision"`
	Reviewer   string    `json:"reviewer" db:"reviewer"`
	Reason     *string   `json:"reason,omitempty" db:"reason"`
	DecidedAt  time.Time `json:"decided_at" db:"decided_at"`
}

// ResolvedModelConfig is the in-memory form of a FabricDefn.resolved_config used for diffing.
type ResolvedModelConfig struct {
	ModelKey string      `json:"model_key"`
	Cubes    []cube.Cube `json:"cubes"`
	Views    []any       `json:"views,omitempty"`
}

// ChangeAnnotation is used by tuning and reporting logic to surface rule metadata.
type ChangeAnnotation struct {
	Path       string `json:"path"`
	ChangeType string `json:"change_type"`
	RuleID     string `json:"rule_id"`
	Provenance string `json:"provenance"`
	Reason     string `json:"reason"`
}

// CatalogNode represents a node in the data catalog, based on public.catalog_node.
type CatalogNode struct {
	ID                 uuid.UUID       `json:"id" db:"id" gorm:"type:uuid;primary_key"`
	CoreID             uuid.NullUUID   `json:"core_id" db:"core_id" gorm:"type:uuid;column:core_id"`
	TenantID           uuid.UUID       `json:"tenant_id" db:"tenant_id" gorm:"type:uuid;column:tenant_id"`
	TenantDatasourceId uuid.UUID       `json:"tenant_datasource_id" db:"tenant_datasource_id" gorm:"column:tenant_datasource_id"`
	NodeTypeID         uuid.UUID       `json:"node_type_id" db:"node_type_id" gorm:"column:node_type_id"`
	NodeName           string          `json:"node_name" db:"node_name" gorm:"column:node_name"`
	Description        string          `json:"description" db:"description" gorm:"column:description"`
	Properties         json.RawMessage `json:"properties" db:"properties" gorm:"type:jsonb"`
	CreatedAt          time.Time       `json:"created_at" db:"created_at" gorm:"column:created_at"`
	UpdatedAt          time.Time       `json:"updated_at" db:"updated_at" gorm:"column:updated_at"`
	ParentID           uuid.NullUUID   `json:"parent_id" db:"parent_id" gorm:"type:uuid;column:parent_id"`
	ParentTypeID       uuid.NullUUID   `json:"parent_type_id" db:"parent_type_id" gorm:"type:uuid;column:parent_type_id"`
	QualifiedPath      string          `json:"qualified_path" db:"qualified_path" gorm:"column:qualified_path"`
	IsAlpha            bool            `json:"is_alpha" db:"is_alpha" gorm:"column:is_alpha"`
	NodeType           string          `json:"node_type" db:"node_type"`
	Config             interface{}     `json:"config" db:"config"`
}

// CatalogEdge represents a relationship between two nodes, based on public.catalog_edge.
type CatalogEdge struct {
	ID                 uuid.UUID       `json:"id" db:"id"`
	CoreID             uuid.NullUUID   `json:"core_id" db:"core_id"`
	TenantID           uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	TenantDatasourceId uuid.UUID       `json:"tenant_datasource_id" db:"tenant_datasource_id"`
	SourceNodeID       uuid.UUID       `json:"source_node_id" db:"source_node_id"`
	TargetNodeID       uuid.UUID       `json:"target_node_id" db:"target_node_id"`
	Properties         json.RawMessage `json:"properties" db:"properties"` // For jsonb
	CreatedAt          time.Time       `json:"created_at" db:"created_at"`
	EdgeTypeID         uuid.UUID       `json:"edge_type_id" db:"edge_type_id"`
	EdgeTypeName       string          `json:"edge_type_name" db:"edge_type_name"`
}

// DrillDownLocator specifies the coordinates for drill-down
type DrillDownLocator struct {
	XValues []interface{} `json:"xValues"`
	YValues []interface{} `json:"yValues"`
}

// PivotConfig specifies the pivot configuration
type PivotConfig struct {
	X []string `json:"x,omitempty"`
	Y []string `json:"y,omitempty"`
}

// DrillDownQuery represents a query for drill-down
type DrillDownQuery struct {
	Measures       []string        `json:"measures"`
	Dimensions     []string        `json:"dimensions"`
	Filters        []Filter        `json:"filters"`
	TimeDimensions []TimeDimension `json:"timeDimensions"`
}

// TimeDimension represents a time dimension in a query
type TimeDimension struct {
	Dimension   string   `json:"dimension"`
	DateRange   []string `json:"dateRange,omitempty"`
	Granularity string   `json:"granularity,omitempty"`
}

// PivotRow represents a row in pivoted data
type PivotRow struct {
	XValues      []interface{}   `json:"xValues"`
	YValuesArray [][]interface{} `json:"yValuesArray"`
}

// ResultMeasure represents a measure in the result set with drill-down capabilities
type ResultMeasure struct {
	Name         string      `json:"name"`
	Value        interface{} `json:"value"`
	DrillMembers []string    `json:"drillMembers,omitempty"`
}

// ResultSet represents the result of a calculation with drill-down and pivot capabilities
type ResultSet struct {
	Data     interface{}     `json:"data"`
	Measures []ResultMeasure `json:"measures"`
}

// DrillDown performs drill-down on a measure
func (m *ResultMeasure) DrillDown(locator DrillDownLocator, pivotConfig *PivotConfig) *DrillDownQuery {
	// Implementation for drill-down
	query := &DrillDownQuery{
		Measures:       []string{m.Name},
		Dimensions:     m.DrillMembers,
		Filters:        []Filter{},
		TimeDimensions: []TimeDimension{},
	}

	// Add filters based on locator
	for _, xVal := range locator.XValues {
		// Add dimension filter based on xVal
		filter := Filter{
			Field:  "dimension", // This should be the actual dimension name
			Op:     "equals",
			Values: []string{fmt.Sprintf("%v", xVal)},
		}
		query.Filters = append(query.Filters, filter)
	}

	for _, yVal := range locator.YValues {
		// Add measure filter based on yVal
		filter := Filter{
			Field:  m.Name,
			Op:     "equals",
			Values: []string{fmt.Sprintf("%v", yVal)},
		}
		query.Filters = append(query.Filters, filter)
	}

	return query
}

// Pivot performs pivoting on the result set
func (rs *ResultSet) Pivot(pivotConfig *PivotConfig) []PivotRow {
	// Implementation for pivoting
	var result []PivotRow

	// This is a simplified implementation
	// In a real implementation, you would process the data according to pivotConfig
	// For now, return empty result
	return result
}

// Calculation represents a financial or analytical calculation definition from the catalog.
type Calculation struct {
	ID             uuid.UUID       `json:"id" db:"id"`
	NodeID         uuid.UUID       `json:"node_id" db:"node_id"`
	Name           string          `json:"name" db:"name"`
	Title          string          `json:"title" db:"title"`
	Description    string          `json:"description" db:"description"`
	Formula        string          `json:"formula" db:"formula"`
	EngineType     string          `json:"engine_type" db:"engine_type"` // postgres, cube, python, excel
	ReturnType     string          `json:"return_type" db:"return_type"`
	Arguments      json.RawMessage `json:"arguments" db:"arguments"` // JSONB
	Category       string          `json:"category" db:"category"`
	Subcategory    string          `json:"subcategory" db:"subcategory"`
	DomainID       *uuid.UUID      `json:"domain_id" db:"domain_id"`           // Link to data_domain
	ExecutionType  string          `json:"execution_type" db:"execution_type"` // realtime, batch
	Engine         string          `json:"engine" db:"engine"`                 // internal, cube, spark
	IsMaterialized bool            `json:"is_materialized" db:"is_materialized"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at" db:"updated_at"`
}

// SemanticModelCalculation represents the link between a semantic model and a calculation.
type SemanticModelCalculation struct {
	ID              uuid.UUID       `json:"id" db:"id"`
	SemanticModelID uuid.UUID       `json:"semantic_model_id" db:"semantic_model_id"`
	CalculationID   uuid.UUID       `json:"calculation_id" db:"calculation_id"`
	ArgumentMapping json.RawMessage `json:"argument_mapping" db:"argument_mapping"` // JSONB: {"arg_name": "column_name"}
	OutputName      string          `json:"output_name" db:"output_name"`
	IsPublic        bool            `json:"is_public" db:"is_public"`
	CreatedAt       time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at" db:"updated_at"`
	CreatedBy       *uuid.UUID      `json:"created_by" db:"created_by"`
	UpdatedBy       *uuid.UUID      `json:"updated_by" db:"updated_by"`
	// Joined fields
	CalculationName string `json:"calculation_name,omitempty" db:"calculation_name"`
}

// CatalogAlias represents a mapping from a friendly name to a canonical key.
type CatalogAlias struct {
	ID           uuid.UUID `json:"id" db:"id"`
	Alias        string    `json:"alias" db:"alias"`
	CanonicalKey string    `json:"canonical_key" db:"canonical_key"`
	TenantID     uuid.UUID `json:"tenant_id" db:"tenant_id"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// --- NLQ Data Contracts ---

// BusinessTerm represents a business glossary term.
type BusinessTerm struct {
	ID           uuid.UUID       `json:"id" db:"id"`
	Term         string          `json:"term" db:"term"`
	Definition   string          `json:"definition" db:"definition"`
	Synonyms     json.RawMessage `json:"synonyms" db:"synonyms"` // JSON array of strings
	Scope        string          `json:"scope" db:"scope"`
	CanonicalKey string          `json:"canonical_key" db:"canonical_key"`
	TenantID     uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	CreatedAt    time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at" db:"updated_at"`
}

// DataProfile represents statistical profile of a data entity.
type DataProfile struct {
	ID            uuid.UUID       `json:"id" db:"id"`
	EntityID      uuid.UUID       `json:"entity_id" db:"entity_id"` // Link to catalog_node
	RowCount      int64           `json:"row_count" db:"row_count"`
	Freshness     time.Time       `json:"freshness" db:"freshness"`
	NullRates     json.RawMessage `json:"null_rates" db:"null_rates"`       // JSON map[col]float
	Distincts     json.RawMessage `json:"distincts" db:"distincts"`         // JSON map[col]int
	Distributions json.RawMessage `json:"distributions" db:"distributions"` // JSON map[col]histogram
	TenantID      uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	CreatedAt     time.Time       `json:"created_at" db:"created_at"`
}

// SemanticModelMetadata extends the basic model definition with NLQ-specific fields.
// This might be stored in catalog_node.properties or a separate table.
// For now, we'll define the struct to be used in the NLQ pipeline.
type SemanticModelMetadata struct {
	ModelKey    string   `json:"model_key"`
	Metrics     []string `json:"metrics"`
	Dimensions  []string `json:"dimensions"`
	Grain       string   `json:"grain"`
	Constraints []string `json:"constraints"`
}
