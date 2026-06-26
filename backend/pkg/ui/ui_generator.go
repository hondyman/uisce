package ui

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/hondyman/semlayer/backend/internal/rules"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// ============================================================================
// CORE UI GENERATION ENGINE
// ============================================================================

// UIGenerator generates dynamic forms from metadata
type UIGenerator struct {
	db              *sqlx.DB
	hierarchyEngine *rules.ValidationEngineWithHierarchy
}

// NewUIGenerator creates a new UI generator
func NewUIGenerator(db *sqlx.DB) *UIGenerator {
	logger := log.New(os.Stdout, "hierarchical-validator: ", log.LstdFlags)
	return &UIGenerator{
		db:              db,
		hierarchyEngine: rules.NewValidationEngineWithHierarchy(db.DB, logger),
	}
}

// ============================================================================
// DATA STRUCTURES
// ============================================================================

// BusinessObject represents a BO metadata definition
type BusinessObject struct {
	ID                 string     `db:"id" json:"id"`
	TenantID           string     `db:"tenant_id" json:"tenant_id"`
	BOName             string     `db:"bo_name" json:"bo_name"`
	BODescription      string     `db:"bo_description" json:"bo_description"`
	EntityType         string     `db:"entity_type" json:"entity_type"`
	AllowCustomFields  bool       `db:"allow_custom_fields" json:"allow_custom_fields"`
	AllowFieldDeletion bool       `db:"allow_field_deletion" json:"allow_field_deletion"`
	IsSystemBO         bool       `db:"is_system_bo" json:"is_system_bo"`
	IsActive           bool       `db:"is_active" json:"is_active"`
	Fields             []*BOField `json:"fields,omitempty"`
}

// BOField represents a field in a Business Object
type BOField struct {
	ID                    string         `db:"id" json:"id"`
	BOID                  string         `db:"bo_id" json:"bo_id"`
	FieldName             string         `db:"field_name" json:"field_name"`
	FieldType             string         `db:"field_type" json:"field_type"`
	DisplayLabel          string         `db:"display_label" json:"display_label"`
	DisplayOrder          int            `db:"display_order" json:"display_order"`
	SectionName           string         `db:"section_name" json:"section_name"`
	HelpText              string         `db:"help_text" json:"help_text"`
	PlaceholderText       string         `db:"placeholder_text" json:"placeholder_text"`
	IsRequired            bool           `db:"is_required" json:"is_required"`
	IsReadonly            bool           `db:"is_readonly" json:"is_readonly"`
	IsSearchable          bool           `db:"is_searchable" json:"is_searchable"`
	IsSortable            bool           `db:"is_sortable" json:"is_sortable"`
	MaxLength             sql.NullInt64  `db:"max_length" json:"max_length,omitempty"`
	MinValue              sql.NullString `db:"min_value" json:"min_value,omitempty"`
	MaxValue              sql.NullString `db:"max_value" json:"max_value,omitempty"`
	DecimalPlaces         sql.NullInt64  `db:"decimal_places" json:"decimal_places,omitempty"`
	DateFormat            string         `db:"date_format" json:"date_format"`
	ReferenceBoID         sql.NullString `db:"reference_bo_id" json:"reference_bo_id,omitempty"`
	ReferenceDisplayField string         `db:"reference_display_field" json:"reference_display_field"`
	PicklistValues        pq.StringArray `db:"picklist_values" json:"picklist_values"`
	DefaultValue          string         `db:"default_value" json:"default_value"`
	IsSystemField         bool           `db:"is_system_field" json:"is_system_field"`
	IsCustomField         bool           `db:"is_custom_field" json:"is_custom_field"`
	ValidationRuleIds     pq.StringArray `db:"validation_rule_ids" json:"validation_rule_ids"`
}

// ValidationRule represents a validation rule
type ValidationRule struct {
	ID                   string          `db:"id" json:"id"`
	TenantID             string          `db:"tenant_id" json:"tenant_id"`
	RuleName             string          `db:"rule_name" json:"rule_name"`
	RuleDescription      string          `db:"rule_description" json:"rule_description"`
	RuleCategory         string          `db:"rule_category" json:"rule_category"`
	Severity             string          `db:"severity" json:"severity"`
	ErrorMessage         string          `db:"error_message" json:"error_message"`
	HelpMessage          string          `db:"help_message" json:"help_message"`
	ConditionType        string          `db:"condition_type" json:"condition_type"`
	ConditionJSON        json.RawMessage `db:"condition_json" json:"condition_json"`
	ExecuteClientSide    bool            `db:"execute_client_side" json:"execute_client_side"`
	ExecuteServerSide    bool            `db:"execute_server_side" json:"execute_server_side"`
	RunOnBlur            bool            `db:"run_on_blur" json:"run_on_blur"`
	RunOnChange          bool            `db:"run_on_change" json:"run_on_change"`
	RunOnSubmit          bool            `db:"run_on_submit" json:"run_on_submit"`
	RequiresDatabaseCall bool            `db:"requires_database_call" json:"requires_database_call"`
	IsActive             bool            `db:"is_active" json:"is_active"`
}

// PageLayout represents a page layout
type PageLayout struct {
	ID                string           `db:"id" json:"id"`
	TenantID          string           `db:"tenant_id" json:"tenant_id"`
	BOID              string           `db:"bo_id" json:"bo_id"`
	LayoutName        string           `db:"layout_name" json:"layout_name"`
	LayoutType        string           `db:"layout_type" json:"layout_type"`
	LayoutDescription string           `db:"layout_description" json:"layout_description"`
	DefaultColumns    int              `db:"default_columns" json:"default_columns"`
	MobileLayout      string           `db:"mobile_layout" json:"mobile_layout"`
	IsDefaultLayout   bool             `db:"is_default_layout" json:"is_default_layout"`
	IsActive          bool             `db:"is_active" json:"is_active"`
	Sections          []*LayoutSection `json:"sections,omitempty"`
	Actions           []*LayoutAction  `json:"actions,omitempty"`
}

// LayoutSection represents a section in a layout
type LayoutSection struct {
	ID                   string         `db:"id" json:"id"`
	LayoutID             string         `db:"layout_id" json:"layout_id"`
	SectionOrder         int            `db:"section_order" json:"section_order"`
	SectionTitle         string         `db:"section_title" json:"section_title"`
	SectionDescription   string         `db:"section_description" json:"section_description"`
	SectionColumns       int            `db:"section_columns" json:"section_columns"`
	IsCollapsible        bool           `db:"is_collapsible" json:"is_collapsible"`
	IsInitiallyCollapsed bool           `db:"is_initially_collapsed" json:"is_initially_collapsed"`
	HasBorder            bool           `db:"has_border" json:"has_border"`
	BackgroundColor      string         `db:"background_color" json:"background_color"`
	IsVisible            bool           `db:"is_visible" json:"is_visible"`
	HelpText             string         `db:"help_text" json:"help_text"`
	FieldIds             pq.StringArray `db:"field_ids" json:"field_ids"`
	Fields               []*BOField     `json:"fields,omitempty"`
}

// LayoutAction represents an action button on a layout
type LayoutAction struct {
	ID                   string         `db:"id" json:"id"`
	LayoutID             string         `db:"layout_id" json:"layout_id"`
	ActionOrder          int            `db:"action_order" json:"action_order"`
	ActionLabel          string         `db:"action_label" json:"action_label"`
	ActionType           string         `db:"action_type" json:"action_type"`
	ActionIcon           string         `db:"action_icon" json:"action_icon"`
	RequiresValidation   bool           `db:"requires_validation" json:"requires_validation"`
	RequiresConfirmation bool           `db:"requires_confirmation" json:"requires_confirmation"`
	ConfirmationMessage  string         `db:"confirmation_message" json:"confirmation_message"`
	TriggersBPId         sql.NullString `db:"triggers_bp_id" json:"triggers_bp_id,omitempty"`
	IsVisible            bool           `db:"is_visible" json:"is_visible"`
	IsEnabled            bool           `db:"is_enabled" json:"is_enabled"`
	ButtonStyle          string         `db:"button_style" json:"button_style"`
	ButtonSize           string         `db:"button_size" json:"button_size"`
	SuccessMessage       string         `db:"success_message" json:"success_message"`
	ErrorMessage         string         `db:"error_message" json:"error_message"`
	RedirectOnSuccess    string         `db:"redirect_on_success" json:"redirect_on_success"`
}

// FormDefinition is what the frontend receives
type FormDefinition struct {
	ID             string                       `json:"id"`
	LayoutName     string                       `json:"layout_name"`
	LayoutType     string                       `json:"layout_type"`
	BusinessObject *BusinessObject              `json:"business_object"`
	Sections       []*FormSection               `json:"sections"`
	Actions        []*LayoutAction              `json:"actions"`
	Validations    map[string][]*ValidationRule `json:"validations"` // fieldId -> rules
}

// FormSection is a section with resolved fields
type FormSection struct {
	ID                   string     `json:"id"`
	SectionOrder         int        `json:"section_order"`
	SectionTitle         string     `json:"section_title"`
	SectionDescription   string     `json:"section_description"`
	SectionColumns       int        `json:"section_columns"`
	IsCollapsible        bool       `json:"is_collapsible"`
	IsInitiallyCollapsed bool       `json:"is_initially_collapsed"`
	HasBorder            bool       `json:"has_border"`
	BackgroundColor      string     `json:"background_color"`
	IsVisible            bool       `json:"is_visible"`
	HelpText             string     `json:"help_text"`
	Fields               []*BOField `json:"fields"`
}

// ValidationResult is returned when validating form data
type ValidationResult struct {
	Valid    bool         `json:"valid"`
	Errors   []FieldError `json:"errors,omitempty"`
	Warnings []FieldError `json:"warnings,omitempty"`
}

// FieldError represents a single field validation error
type FieldError struct {
	FieldID   string `json:"field_id"`
	FieldName string `json:"field_name"`
	Severity  string `json:"severity"`
	Message   string `json:"message"`
}

// ============================================================================
// FORM DEFINITION GENERATION
// ============================================================================

// GetFormDefinition generates a complete form definition from layout metadata
func (g *UIGenerator) GetFormDefinition(ctx context.Context, layoutID string) (*FormDefinition, error) {
	// 1. Load page layout
	layout, err := g.loadPageLayout(ctx, layoutID)
	if err != nil {
		return nil, fmt.Errorf("failed to load page layout: %w", err)
	}

	// 2. Load Business Object
	bo, err := g.loadBusinessObject(ctx, layout.BOID)
	if err != nil {
		return nil, fmt.Errorf("failed to load business object: %w", err)
	}

	// 3. Load fields for BO
	fields, err := g.loadBOFields(ctx, layout.BOID)
	if err != nil {
		return nil, fmt.Errorf("failed to load BO fields: %w", err)
	}
	bo.Fields = fields

	// 4. Load validation rules for all fields
	fieldValidations := make(map[string][]*ValidationRule)
	for _, field := range fields {
		if len(field.ValidationRuleIds) > 0 {
			rules, err := g.loadValidationRules(ctx, field.ValidationRuleIds)
			if err != nil {
				return nil, fmt.Errorf("failed to load validation rules for field %s: %w", field.FieldName, err)
			}
			fieldValidations[field.ID] = rules
		}
	}

	// 5. Load layout sections with fields
	sections, err := g.loadLayoutSections(ctx, layoutID)
	if err != nil {
		return nil, fmt.Errorf("failed to load layout sections: %w", err)
	}

	// 6. Resolve section fields
	fieldMap := make(map[string]*BOField)
	for _, field := range fields {
		fieldMap[field.ID] = field
	}

	formSections := make([]*FormSection, 0)
	for _, section := range sections {
		formSection := &FormSection{
			ID:                   section.ID,
			SectionOrder:         section.SectionOrder,
			SectionTitle:         section.SectionTitle,
			SectionDescription:   section.SectionDescription,
			SectionColumns:       section.SectionColumns,
			IsCollapsible:        section.IsCollapsible,
			IsInitiallyCollapsed: section.IsInitiallyCollapsed,
			HasBorder:            section.HasBorder,
			BackgroundColor:      section.BackgroundColor,
			IsVisible:            section.IsVisible,
			HelpText:             section.HelpText,
			Fields:               make([]*BOField, 0),
		}

		// Add fields to section
		for _, fieldID := range section.FieldIds {
			if field, ok := fieldMap[fieldID]; ok {
				formSection.Fields = append(formSection.Fields, field)
			}
		}

		formSections = append(formSections, formSection)
	}

	// 7. Load layout actions
	actions, err := g.loadLayoutActions(ctx, layoutID)
	if err != nil {
		return nil, fmt.Errorf("failed to load layout actions: %w", err)
	}

	// 8. Build form definition
	formDef := &FormDefinition{
		ID:             layout.ID,
		LayoutName:     layout.LayoutName,
		LayoutType:     layout.LayoutType,
		BusinessObject: bo,
		Sections:       formSections,
		Actions:        actions,
		Validations:    fieldValidations,
	}

	return formDef, nil
}

// ============================================================================
// DATA LOADING HELPERS
// ============================================================================

func (g *UIGenerator) loadPageLayout(ctx context.Context, layoutID string) (*PageLayout, error) {
	layout := &PageLayout{}
	// TODO(hasura-migration): Replace SQL query with Hasura GraphQL query
	// Example GraphQL query:
	// query LoadPageLayout($layoutId: uuid!) {
	//   page_layouts_by_pk(id: $layoutId) {
	//     id
	//     tenant_id
	//     bo_id
	//     layout_name
	//     layout_type
	//     layout_description
	//     default_columns
	//     mobile_layout
	//     is_default_layout
	//     is_active
	//   }
	// }
	query := `
		SELECT id, tenant_id, bo_id, layout_name, layout_type, layout_description,
		       default_columns, mobile_layout, is_default_layout, is_active
		FROM page_layouts
		WHERE id = $1
	`
	err := g.db.GetContext(ctx, layout, query, layoutID)
	if err != nil {
		return nil, err
	}
	return layout, nil
}

func (g *UIGenerator) loadBusinessObject(ctx context.Context, boID string) (*BusinessObject, error) {
	bo := &BusinessObject{}
	// TODO(hasura-migration): Replace SQL query with Hasura GraphQL query
	// Example GraphQL query:
	// query LoadBusinessObject($boId: uuid!) {
	//   business_objects_by_pk(id: $boId) {
	//     id
	//     tenant_id
	//     bo_name
	//     bo_description
	//     entity_type
	//     allow_custom_fields
	//     allow_field_deletion
	//     is_system_bo
	//     is_active
	//   }
	// }
	query := `
		SELECT id, tenant_id, bo_name, bo_description, entity_type,
		       allow_custom_fields, allow_field_deletion, is_system_bo, is_active
		FROM business_objects
		WHERE id = $1
	`
	err := g.db.GetContext(ctx, bo, query, boID)
	if err != nil {
		return nil, err
	}
	return bo, nil
}

func (g *UIGenerator) loadBOFields(ctx context.Context, boID string) ([]*BOField, error) {
	fields := make([]*BOField, 0)
	// TODO(hasura-migration): Replace SQL query with Hasura GraphQL query
	// Example GraphQL query:
	// query LoadBOFields($boId: uuid!) {
	//   bo_fields(
	//     where: {bo_id: {_eq: $boId}},
	//     order_by: {display_order: asc}
	//   ) {
	//     id
	//     bo_id
	//     field_name
	//     field_type
	//     display_label
	//     display_order
	//     section_name
	//     help_text
	//     placeholder_text
	//     is_required
	//     is_readonly
	//     is_searchable
	//     is_sortable
	//     max_length
	//     min_value
	//     max_value
	//     decimal_places
	//     date_format
	//     reference_bo_id
	//     reference_display_field
	//     picklist_values
	//     default_value
	//     is_system_field
	//     is_custom_field
	//     validation_rule_ids
	//   }
	// }
	query := `
		SELECT id, bo_id, field_name, field_type, display_label, display_order,
		       section_name, help_text, placeholder_text, is_required, is_readonly,
		       is_searchable, is_sortable, max_length, min_value, max_value,
		       decimal_places, date_format, reference_bo_id, reference_display_field,
		       picklist_values, default_value, is_system_field, is_custom_field,
		       validation_rule_ids
		FROM bo_fields
		WHERE bo_id = $1
		ORDER BY display_order ASC
	`
	err := g.db.SelectContext(ctx, &fields, query, boID)
	if err != nil {
		return nil, err
	}
	return fields, nil
}

func (g *UIGenerator) loadValidationRules(ctx context.Context, ruleIds pq.StringArray) ([]*ValidationRule, error) {
	rules := make([]*ValidationRule, 0)
	// TODO(hasura-migration): Replace SQL query with Hasura GraphQL query
	// Example GraphQL query:
	// query LoadValidationRules($ruleIds: [uuid!]!) {
	//   validation_rules(
	//     where: {
	//       id: {_in: $ruleIds},
	//       is_active: {_eq: true}
	//     },
	//     order_by: {rule_name: asc}
	//   ) {
	//     id
	//     tenant_id
	//     rule_name
	//     rule_description
	//     rule_category
	//     severity
	//     error_message
	//     help_message
	//     condition_type
	//     condition_json
	//     execute_client_side
	//     execute_server_side
	//     run_on_blur
	//     run_on_change
	//     run_on_submit
	//     requires_database_call
	//     is_active
	//   }
	// }
	query := `
		SELECT id, tenant_id, rule_name, rule_description, rule_category, severity,
		       error_message, help_message, condition_type, condition_json,
		       execute_client_side, execute_server_side, run_on_blur, run_on_change,
		       run_on_submit, requires_database_call, is_active
		FROM validation_rules
		WHERE id = ANY($1) AND is_active = true
		ORDER BY rule_name ASC
	`
	err := g.db.SelectContext(ctx, &rules, query, ruleIds)
	if err != nil {
		return nil, err
	}
	return rules, nil
}

func (g *UIGenerator) loadLayoutSections(ctx context.Context, layoutID string) ([]*LayoutSection, error) {
	sections := make([]*LayoutSection, 0)
	// TODO(hasura-migration): Replace SQL query with Hasura GraphQL query
	// Example GraphQL query:
	// query LoadLayoutSections($layoutId: uuid!) {
	//   layout_sections(
	//     where: {layout_id: {_eq: $layoutId}},
	//     order_by: {section_order: asc}
	//   ) {
	//     id
	//     layout_id
	//     section_order
	//     section_title
	//     section_description
	//     section_columns
	//     is_collapsible
	//     is_initially_collapsed
	//     has_border
	//     background_color
	//     is_visible
	//     help_text
	//     field_ids
	//   }
	// }
	query := `
		SELECT id, layout_id, section_order, section_title, section_description,
		       section_columns, is_collapsible, is_initially_collapsed, has_border,
		       background_color, is_visible, help_text, field_ids
		FROM layout_sections
		WHERE layout_id = $1
		ORDER BY section_order ASC
	`
	err := g.db.SelectContext(ctx, &sections, query, layoutID)
	if err != nil {
		return nil, err
	}
	return sections, nil
}

func (g *UIGenerator) loadLayoutActions(ctx context.Context, layoutID string) ([]*LayoutAction, error) {
	actions := make([]*LayoutAction, 0)
	// TODO(hasura-migration): Replace SQL query with Hasura GraphQL query
	// Example GraphQL query:
	// query LoadLayoutActions($layoutId: uuid!) {
	//   layout_actions(
	//     where: {layout_id: {_eq: $layoutId}},
	//     order_by: {action_order: asc}
	//   ) {
	//     id
	//     layout_id
	//     action_order
	//     action_label
	//     action_type
	//     action_icon
	//     requires_validation
	//     requires_confirmation
	//     confirmation_message
	//     triggers_bp_id
	//     is_visible
	//     is_enabled
	//     button_style
	//     button_size
	//     success_message
	//     error_message
	//     redirect_on_success
	//   }
	// }
	query := `
		SELECT id, layout_id, action_order, action_label, action_type, action_icon,
		       requires_validation, requires_confirmation, confirmation_message,
		       triggers_bp_id, is_visible, is_enabled, button_style, button_size,
		       success_message, error_message, redirect_on_success
		FROM layout_actions
		WHERE layout_id = $1
		ORDER BY action_order ASC
	`
	err := g.db.SelectContext(ctx, &actions, query, layoutID)
	if err != nil {
		return nil, err
	}
	return actions, nil
}

// ============================================================================
// FORM DATA VALIDATION
// ============================================================================

// ValidateFormData validates form submission data against all rules
func (g *UIGenerator) ValidateFormData(ctx context.Context, boID string, data map[string]interface{}, tenantID string, datasourceID string) (*ValidationResult, error) {
	result := &ValidationResult{
		Valid:    true,
		Errors:   make([]FieldError, 0),
		Warnings: make([]FieldError, 0),
	}

	// Load all fields
	fields, err := g.loadBOFields(ctx, boID)
	if err != nil {
		return nil, fmt.Errorf("failed to load BO fields: %w", err)
	}

	// 3. Validate each field
	for _, field := range fields {
		value := data[field.FieldName]

		// Required check
		if field.IsRequired && isEmpty(value) {
			result.Valid = false
			result.Errors = append(result.Errors, FieldError{
				FieldID:   field.ID,
				FieldName: field.FieldName,
				Severity:  "error",
				Message:   fmt.Sprintf("%s is required", field.DisplayLabel),
			})
			continue
		}

		// Skip other validation if not provided and not required
		if isEmpty(value) {
			continue
		}

		// Type-specific validation
		if err := g.validateFieldType(ctx, field, value); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, FieldError{
				FieldID:   field.ID,
				FieldName: field.FieldName,
				Severity:  "error",
				Message:   err.Error(),
			})
			continue
		}

		// Execute validation rules
		for _, ruleID := range field.ValidationRuleIds {
			rule, err := g.loadValidationRuleByID(ctx, ruleID)
			if err != nil || rule == nil || !rule.IsActive || !rule.ExecuteServerSide {
				continue
			}

			valid, err := g.executeRule(ctx, rule, value, data)
			if err != nil {
				// Log error but continue
				continue
			}

			if !valid {
				fieldError := FieldError{
					FieldID:   field.ID,
					FieldName: field.FieldName,
					Severity:  rule.Severity,
					Message:   rule.ErrorMessage,
				}

				if rule.Severity == "error" {
					result.Valid = false
					result.Errors = append(result.Errors, fieldError)
				} else {
					result.Warnings = append(result.Warnings, fieldError)
				}
			}
		}
	}

	// Hierarchical validation
	valid, hierarchyErrors, err := g.hierarchyEngine.ValidateHierarchical(ctx, boID, data, tenantID, datasourceID)
	if err != nil {
		// Decide how to handle this error. For now, we'll just log it and continue.
		// In a real application, you might want to return an error here.
		fmt.Printf("hierarchical validation failed: %v", err)
	}

	if !valid {
		result.Valid = false
		for _, hError := range hierarchyErrors {
			fieldError := FieldError{
				FieldID:   hError.RuleID,
				FieldName: hError.RuleID, // Use RuleID as the field name since Path doesn't exist
				Severity:  hError.Severity,
				Message:   hError.Message,
			}
			if hError.Severity == "error" {
				result.Errors = append(result.Errors, fieldError)
			} else {
				result.Warnings = append(result.Warnings, fieldError)
			}
		}
	}

	return result, nil
}

// ============================================================================
// RULE EXECUTION ENGINE
// ============================================================================

func (g *UIGenerator) loadValidationRuleByID(ctx context.Context, ruleID string) (*ValidationRule, error) {
	rule := &ValidationRule{}
	query := `
		SELECT id, tenant_id, rule_name, rule_description, rule_category, severity,
		       error_message, help_message, condition_type, condition_json,
		       execute_client_side, execute_server_side, run_on_blur, run_on_change,
		       run_on_submit, requires_database_call, is_active
		FROM validation_rules
		WHERE id = $1
	`
	err := g.db.GetContext(ctx, rule, query, ruleID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return rule, nil
}

// executeRule validates a value against a rule
func (g *UIGenerator) executeRule(ctx context.Context, rule *ValidationRule, value interface{}, allData map[string]interface{}) (bool, error) {
	switch rule.ConditionType {
	case "regex":
		return g.validateRegex(rule, value)
	case "compare":
		return g.validateComparison(rule, value)
	case "unique_check":
		return g.validateUniqueness(ctx, rule, value)
	case "range":
		return g.validateRange(rule, value)
	case "cross_field":
		return g.validateCrossField(rule, value, allData)
	default:
		return true, nil
	}
}

func (g *UIGenerator) validateRegex(rule *ValidationRule, value interface{}) (bool, error) {
	var condition struct {
		Pattern string `json:"pattern"`
	}
	if err := json.Unmarshal(rule.ConditionJSON, &condition); err != nil {
		return false, fmt.Errorf("invalid regex condition: %w", err)
	}

	regex, err := regexp.Compile(condition.Pattern)
	if err != nil {
		return false, fmt.Errorf("invalid regex pattern: %w", err)
	}

	valueStr := fmt.Sprintf("%v", value)
	return regex.MatchString(valueStr), nil
}

func (g *UIGenerator) validateComparison(rule *ValidationRule, value interface{}) (bool, error) {
	var condition struct {
		Operator string      `json:"operator"`
		Value    interface{} `json:"value"`
	}
	if err := json.Unmarshal(rule.ConditionJSON, &condition); err != nil {
		return false, fmt.Errorf("invalid comparison condition: %w", err)
	}

	switch condition.Operator {
	case "equals":
		return fmt.Sprintf("%v", value) == fmt.Sprintf("%v", condition.Value), nil
	case "not_equals":
		return fmt.Sprintf("%v", value) != fmt.Sprintf("%v", condition.Value), nil
	case "contains":
		return strings.Contains(fmt.Sprintf("%v", value), fmt.Sprintf("%v", condition.Value)), nil
	case "gt", "gte", "lt", "lte":
		// Numeric comparison - implement if needed
		return true, nil
	default:
		return true, nil
	}
}

func (g *UIGenerator) validateUniqueness(ctx context.Context, rule *ValidationRule, value interface{}) (bool, error) {
	var condition struct {
		Field string `json:"field"`
		Table string `json:"table"`
		Scope string `json:"scope"`
	}
	if err := json.Unmarshal(rule.ConditionJSON, &condition); err != nil {
		return false, fmt.Errorf("invalid uniqueness condition: %w", err)
	}

	// In production: Query the specified table to check uniqueness
	// For now: return true
	return true, nil
}

func (g *UIGenerator) validateRange(rule *ValidationRule, value interface{}) (bool, error) {
	// Implement range validation
	return true, nil
}

func (g *UIGenerator) validateCrossField(rule *ValidationRule, value interface{}, allData map[string]interface{}) (bool, error) {
	// Implement cross-field validation
	return true, nil
}

// ============================================================================
// TYPE VALIDATION HELPERS
// ============================================================================

func (g *UIGenerator) validateFieldType(ctx context.Context, field *BOField, value interface{}) error {
	switch field.FieldType {
	case "string":
		if str, ok := value.(string); ok {
			if field.MaxLength.Valid && int64(len(str)) > field.MaxLength.Int64 {
				return fmt.Errorf("exceeds maximum length of %d", field.MaxLength.Int64)
			}
		}
		return nil

	case "number", "decimal":
		// Validate numeric type
		return nil

	case "date":
		// Validate date format
		if s, ok := value.(string); ok {
			// Basic validation: non-empty string for now
			if strings.TrimSpace(s) == "" {
				return fmt.Errorf("invalid date value")
			}
		}
		return nil

	case "boolean":
		// Validate boolean type
		_, ok := value.(bool)
		if !ok {
			return fmt.Errorf("invalid boolean value")
		}
		return nil

	case "reference":
		// Validate reference exists
		return nil

	case "picklist":
		// Validate value is in picklist
		valueStr := fmt.Sprintf("%v", value)
		for _, v := range field.PicklistValues {
			if v == valueStr {
				return nil
			}
		}
		return fmt.Errorf("invalid picklist value: %s", valueStr)

	default:
		return nil
	}
}

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

func isEmpty(value interface{}) bool {
	if value == nil {
		return true
	}
	if str, ok := value.(string); ok {
		return strings.TrimSpace(str) == ""
	}
	return false
}
