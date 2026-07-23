package reports

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// TypeMapping constants for consistent type determination
const (
	EntityTypeRelationship = "relationship"
	EntityTypeCollection   = "collection"
	EntityTypeMeasure      = "measure"
	EntityTypeAttribute    = "attribute"

	DataTypeObject  = "object"
	DataTypeArray   = "array"
	DataTypeNumber  = "number"
	DataTypeString  = "string"
	DataTypeBoolean = "boolean"
	DataTypeNull    = "null"
	DataTypeMixed   = "mixed"
)

// FilterTypeMapping maps data types to appropriate filter types
var FilterTypeMapping = map[string]string{
	DataTypeNumber:  "range",
	DataTypeString:  "contains",
	DataTypeBoolean: "equals",
	"date":          "between",
}

// AggregationMapping maps entity types to default aggregations
var AggregationMapping = map[string]string{
	EntityTypeMeasure:   "sum",
	EntityTypeAttribute: "count",
}

// GetDefaultFilterType returns the default filter type for a data type
func GetDefaultFilterType(dataType string) string {
	if filterType, exists := FilterTypeMapping[dataType]; exists {
		return filterType
	}
	return "equals"
}

// GetDefaultAggregation returns the default aggregation for an entity type
func GetDefaultAggregation(entityType string) string {
	if agg, exists := AggregationMapping[entityType]; exists {
		return agg
	}
	return "count"
}

// InferDataType determines the data type from a JSON value
func InferDataType(value interface{}) string {
	switch value.(type) {
	case map[string]interface{}:
		return DataTypeObject
	case []interface{}:
		return DataTypeArray
	case float64:
		return DataTypeNumber
	case string:
		return DataTypeString
	case bool:
		return DataTypeBoolean
	case nil:
		return DataTypeNull
	default:
		return DataTypeMixed
	}
}

// InferEntityType determines the entity type from a JSON value
func InferEntityType(value interface{}) string {
	switch value.(type) {
	case map[string]interface{}:
		return EntityTypeRelationship
	case []interface{}:
		return EntityTypeCollection
	case float64:
		return EntityTypeMeasure
	default:
		return EntityTypeAttribute
	}
}

// ValidateUUID checks if a string is a valid UUID
func ValidateUUID(id string) error {
	if id == "" {
		return fmt.Errorf("ID cannot be empty")
	}
	_, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid UUID format: %w", err)
	}
	return nil
}

// ValidateAndSanitizeString validates and sanitizes user input strings
func ValidateAndSanitizeString(input string, fieldName string, maxLength int) (string, error) {
	if maxLength > 0 && len([]rune(input)) > maxLength {
		return "", fmt.Errorf("%s exceeds maximum length of %d characters", fieldName, maxLength)
	}
	// Remove leading/trailing whitespace
	sanitized := strings.TrimSpace(input)
	if sanitized == "" && input != "" {
		return "", fmt.Errorf("%s contains only whitespace", fieldName)
	}
	return sanitized, nil
}

// ValidateDragDropState validates a drag-drop state object
func ValidateDragDropState(state *DragDropState) error {
	if state == nil {
		return fmt.Errorf("drag-drop state cannot be nil")
	}
	if state.SourceEntity.EntityID == "" {
		return fmt.Errorf("source entity ID cannot be empty")
	}
	if err := ValidateUUID(state.TargetSectionID); err != nil {
		return fmt.Errorf("invalid target section ID: %w", err)
	}
	if err := ValidateUUID(state.SourceEntity.EntityID); err != nil {
		return fmt.Errorf("invalid source entity ID: %w", err)
	}
	if state.Action == "" {
		return fmt.Errorf("action cannot be empty")
	}
	validActions := map[string]bool{
		"add_to_table":       true,
		"create_filter":      true,
		"create_aggregation": true,
		"create_rule":        true,
	}
	if !validActions[state.Action] {
		return fmt.Errorf("invalid action: %s", state.Action)
	}
	return nil
}

// ValidateSectionIndex checks if a section index is valid within sections slice
func ValidateSectionIndex(index int, sectionsLen int) error {
	if index < 0 || index >= sectionsLen {
		return fmt.Errorf("section index %d out of range (0-%d)", index, sectionsLen-1)
	}
	return nil
}

// FindSectionByID finds a section by ID and returns its index
func FindSectionByID(sections []ReportSection, targetID string) (int, error) {
	for i, section := range sections {
		if section.ID.String() == targetID {
			return i, nil
		}
	}
	return -1, fmt.Errorf("section not found: %s", targetID)
}

// MaxStringLength constants for validation
const (
	MaxEntityNameLength     = 255
	MaxDescriptionLength    = 2000
	MaxFilterValueLength    = 1000
	MaxRuleExpressionLength = 5000
)

// DropActionHandler defines the interface for handling different drop actions
type DropActionHandler interface {
	Handle(section *ReportSection, entity DragDropEntity, targetSectionID string) error
}

// AddToTableHandler adds a dropped entity as a table column
type AddToTableHandler struct{}

func (h *AddToTableHandler) Handle(section *ReportSection, entity DragDropEntity, targetSectionID string) error {
	section.DroppedEntities = append(
		section.DroppedEntities,
		DragDropEntity{
			EntityID:      entity.EntityID,
			EntityName:    entity.EntityName,
			EntityType:    entity.EntityType,
			DataType:      entity.DataType,
			DisplayFormat: "raw",
			ColumnWidth:   200,
		},
	)
	return nil
}

// CreateFilterHandler creates a filter from the dropped entity
type CreateFilterHandler struct{}

func (h *CreateFilterHandler) Handle(section *ReportSection, entity DragDropEntity, targetSectionID string) error {
	filter := ReportFilter{
		ID:              uuid.New(),
		FilterType:      GetDefaultFilterType(entity.DataType),
		EntityID:        entity.EntityID,
		EntityName:      entity.EntityName,
		ApplyToSections: []string{targetSectionID},
		DroppedFrom:     "drag_drop",
		Operator:        "and",
	}
	// Note: This is attached to the template, not section
	_ = filter // Will be handled in builder
	return nil
}

// CreateAggregationHandler adds aggregation fields to the section
type CreateAggregationHandler struct{}

func (h *CreateAggregationHandler) Handle(section *ReportSection, entity DragDropEntity, targetSectionID string) error {
	aggField := AggregationField{
		FieldName:       entity.EntityName,
		AggregationType: GetDefaultAggregation(entity.EntityType),
		DisplayName:     entity.EntityName,
	}
	section.AggregationFields = append(section.AggregationFields, aggField)
	return nil
}

// CreateRuleHandler creates a rule from the dropped entity
type CreateRuleHandler struct{}

func (h *CreateRuleHandler) Handle(section *ReportSection, entity DragDropEntity, targetSectionID string) error {
	rule := ReportRule{
		ID:               uuid.New(),
		Name:             fmt.Sprintf("Rule for %s", entity.EntityName),
		Description:      fmt.Sprintf("Auto-generated rule from %s", entity.EntityName),
		EntitiesInvolved: []string{entity.EntityID},
		CreatedFrom: []DragDropEntity{
			{
				EntityID:   entity.EntityID,
				EntityName: entity.EntityName,
				EntityType: entity.EntityType,
			},
		},
		IsActive: true,
	}
	// Note: This is attached to the template, not section
	_ = rule // Will be handled in builder
	return nil
}
