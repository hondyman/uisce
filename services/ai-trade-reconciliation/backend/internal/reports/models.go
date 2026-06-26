package reports

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// SemanticView represents a semantic view (not a physical model)
type SemanticView struct {
	ID              uuid.UUID       `json:"id" db:"id"`
	Name            string          `json:"name" db:"name"`
	Description     string          `json:"description" db:"description"`
	TenantID        uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	EntityType      string          `json:"entity_type" db:"entity_type"` // Portfolio, Position, Trade, etc
	SemanticContent json.RawMessage `json:"semantic_content" db:"semantic_content"`
	CreatedAt       time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at" db:"updated_at"`
}

// SemanticEntity represents an entity within a semantic view
type SemanticEntity struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`      // attribute, relationship, measure
	DataType    string    `json:"data_type"` // string, number, date, etc
	Description string    `json:"description"`
	Droppable   bool      `json:"droppable"` // Can be dragged to reports
	Path        string    `json:"path"`      // JSONPath to the entity
}

// ReportTemplate represents a template-based report built from semantic views
type ReportTemplate struct {
	ID              uuid.UUID       `json:"id" db:"id"`
	Name            string          `json:"name" db:"name"`
	Description     string          `json:"description" db:"description"`
	TenantID        uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	CreatedBy       uuid.UUID       `json:"created_by" db:"created_by"`
	Sections        []ReportSection `json:"sections" db:"sections"`
	Filters         []ReportFilter  `json:"filters" db:"filters"`
	Rules           []ReportRule    `json:"rules" db:"rules"`
	RefreshInterval int             `json:"refresh_interval" db:"refresh_interval"` // minutes
	IsActive        bool            `json:"is_active" db:"is_active"`
	CreatedAt       time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at" db:"updated_at"`
	SectionsJSON    json.RawMessage `json:"-" db:"sections"`
	FiltersJSON     json.RawMessage `json:"-" db:"filters"`
	RulesJSON       json.RawMessage `json:"-" db:"rules"`
}

// ReportSection represents a section in a report (built by dragging semantic entities)
type ReportSection struct {
	ID                uuid.UUID          `json:"id"`
	Title             string             `json:"title"`
	SectionType       string             `json:"section_type"` // table, chart, metric, summary
	DroppedEntities   []DragDropEntity   `json:"dropped_entities"`
	Visualization     VisualizationSpec  `json:"visualization"`
	Order             int                `json:"order"`
	GroupByFields     []string           `json:"group_by_fields"`
	SortByFields      []SortField        `json:"sort_by_fields"`
	AggregationFields []AggregationField `json:"aggregation_fields"`
}

// DragDropEntity represents an entity that was dragged and dropped
type DragDropEntity struct {
	EntityID      string                 `json:"entity_id"`
	EntityName    string                 `json:"entity_name"`
	EntityType    string                 `json:"entity_type"` // attribute, relationship, measure
	DataType      string                 `json:"data_type"`
	DisplayFormat string                 `json:"display_format"` // raw, formatted, calculated
	ColumnWidth   int                    `json:"column_width"`
	Alias         string                 `json:"alias"` // User-friendly name
	Metadata      map[string]interface{} `json:"metadata"`
}

// VisualizationSpec defines how a section is visualized
type VisualizationSpec struct {
	Type       string                 `json:"type"` // bar, line, pie, table, heatmap
	ChartTitle string                 `json:"chart_title"`
	XAxis      string                 `json:"x_axis"`
	YAxis      string                 `json:"y_axis"`
	Color      string                 `json:"color"`
	Options    map[string]interface{} `json:"options"`
}

// SortField represents a field to sort by
type SortField struct {
	FieldName string `json:"field_name"`
	Direction string `json:"direction"` // asc or desc
	Order     int    `json:"order"`
}

// AggregationField represents aggregation to apply
type AggregationField struct {
	FieldName       string `json:"field_name"`
	AggregationType string `json:"aggregation_type"` // sum, avg, count, min, max
	DisplayName     string `json:"display_name"`
}

// ReportFilter allows filtering data based on semantic entities
type ReportFilter struct {
	ID              uuid.UUID   `json:"id"`
	FilterType      string      `json:"filter_type"` // equals, contains, range, in, between
	EntityID        string      `json:"entity_id"`
	EntityName      string      `json:"entity_name"`
	Operator        string      `json:"operator"` // and, or
	FilterValue     interface{} `json:"value"`
	SecondValue     interface{} `json:"second_value"`      // For range/between
	ApplyToSections []string    `json:"apply_to_sections"` // Section IDs to apply filter
	DroppedFrom     string      `json:"dropped_from"`      // Which section filter was created from
}

// ReportRule represents business logic rules applied to report
type ReportRule struct {
	ID               uuid.UUID        `json:"id"`
	Name             string           `json:"name"`
	Description      string           `json:"description"`
	Condition        string           `json:"condition"` // JSONata expression
	Action           string           `json:"action"`    // JSONata expression
	Priority         int              `json:"priority"`
	EntitiesInvolved []string         `json:"entities_involved"` // Entity IDs
	CreatedFrom      []DragDropEntity `json:"created_from"`      // Entities used to create rule
	IsActive         bool             `json:"is_active"`
}

// ReportGeneration represents a generated report instance
type ReportGeneration struct {
	ID             uuid.UUID       `json:"id" db:"id"`
	TemplateID     uuid.UUID       `json:"template_id" db:"template_id"`
	GeneratedAt    time.Time       `json:"generated_at" db:"generated_at"`
	FiltersApplied []ReportFilter  `json:"filters_applied" db:"filters_applied"`
	DataSnapshot   json.RawMessage `json:"data_snapshot" db:"data_snapshot"`
	ExecutionTime  int             `json:"execution_time" db:"execution_time"` // ms
	RowsAffected   int             `json:"rows_affected" db:"rows_affected"`
	Status         string          `json:"status" db:"status"` // success, failed, pending
	ErrorMessage   *string         `json:"error_message" db:"error_message"`
	FiltersJSON    json.RawMessage `json:"-" db:"filters_applied"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
}

// EntityRelationship represents relationships between semantic entities
type EntityRelationship struct {
	ID              uuid.UUID `json:"id" db:"id"`
	SourceEntityID  uuid.UUID `json:"source_entity_id" db:"source_entity_id"`
	TargetEntityID  uuid.UUID `json:"target_entity_id" db:"target_entity_id"`
	RelationType    string    `json:"relation_type" db:"relation_type"` // one-to-many, many-to-many, parent-child
	RelationshipKey string    `json:"relationship_key" db:"relationship_key"`
	Cardinality     string    `json:"cardinality" db:"cardinality"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

// DragDropState represents the state of a drag-drop operation
type DragDropState struct {
	SourceEntity    DragDropEntity         `json:"source_entity"`
	TargetSectionID string                 `json:"target_section_id"`
	Action          string                 `json:"action"` // add_to_table, create_filter, create_aggregation
	Position        int                    `json:"position"`
	AllowedActions  []string               `json:"allowed_actions"` // What actions are allowed for this entity
	Metadata        map[string]interface{} `json:"metadata"`
}

// Scan implements sql.Scanner for JSON fields
func (rs *ReportSection) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("type assertion failed: %v", value)
	}
	return json.Unmarshal(bytes, &rs)
}

// Value implements sql.Valuer for JSON fields
func (rs ReportSection) Value() (driver.Value, error) {
	return json.Marshal(rs)
}

// Similar implementations for other JSON types
func (rf *ReportFilter) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("type assertion failed: %v", value)
	}
	return json.Unmarshal(bytes, &rf)
}

func (rf ReportFilter) Value() (driver.Value, error) {
	return json.Marshal(rf)
}

func (rr *ReportRule) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("type assertion failed: %v", value)
	}
	return json.Unmarshal(bytes, &rr)
}

func (rr ReportRule) Value() (driver.Value, error) {
	return json.Marshal(rr)
}

// SemanticViewWithEntities extends SemanticView with draggable entities
type SemanticViewWithEntities struct {
	ID                  uuid.UUID            `json:"id"`
	Name                string               `json:"name"`
	Description         string               `json:"description"`
	TenantID            uuid.UUID            `json:"tenant_id"`
	EntityType          string               `json:"entity_type"`
	DraggableEntities   []DraggableEntity    `json:"draggable_entities"`
	EntityRelationships []EntityRelationship `json:"entity_relationships"`
	CreatedAt           time.Time            `json:"created_at"`
	UpdatedAt           time.Time            `json:"updated_at"`
}

// DraggableEntity represents an entity that can be dragged to a report
type DraggableEntity struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	Type               string   `json:"type"`      // attribute, relationship, measure
	DataType           string   `json:"data_type"` // string, number, date, etc
	Description        string   `json:"description"`
	Path               string   `json:"path"`
	Droppable          bool     `json:"droppable"`
	AllowedDropActions []string `json:"allowed_drop_actions"` // actions allowed for this entity
	Icon               string   `json:"icon"`
	Tooltip            string   `json:"tooltip"`
	Category           string   `json:"category"` // dimensions, measures, hierarchies
	IsPrimaryKey       bool     `json:"is_primary_key"`
	IsHierarchy        bool     `json:"is_hierarchy"`
	HierarchyMembers   []string `json:"hierarchy_members"`
}

// DroppedEntity represents an entity that has been dropped onto a report
type DroppedEntity struct {
	EntityID      string                 `json:"entity_id"`
	EntityName    string                 `json:"entity_name"`
	EntityType    string                 `json:"entity_type"`
	DataType      string                 `json:"data_type"`
	DisplayFormat string                 `json:"display_format"` // raw, formatted, calculated
	ColumnWidth   int                    `json:"column_width"`
	Alias         string                 `json:"alias"`
	Order         int                    `json:"order"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// ReportData represents generated report data
type ReportData struct {
	ID          uuid.UUID              `json:"id"`
	TemplateID  uuid.UUID              `json:"template_id"`
	Sections    []SectionData          `json:"sections"`
	Metadata    map[string]interface{} `json:"metadata"`
	GeneratedAt time.Time              `json:"generated_at"`
}

// SectionData represents data for a single section
type SectionData struct {
	ID    uuid.UUID     `json:"id"`
	Title string        `json:"title"`
	Type  string        `json:"type"`
	Data  []interface{} `json:"data"`
}

// EntityCategory represents a category of entities for UI organization
type EntityCategory struct {
	Name        string            `json:"name"`
	Icon        string            `json:"icon"`
	Description string            `json:"description"`
	Entities    []DraggableEntity `json:"entities"`
}

// DropAction defines what can be done when dropping an entity
type DropAction struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Icon        string   `json:"icon"`
	Description string   `json:"description"`
	AllowedFor  []string `json:"allowed_for"` // entity types this is allowed for
}
