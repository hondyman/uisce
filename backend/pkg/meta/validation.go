package meta

import (
	"encoding/json"
	"fmt"
)

// ValidationRule represents a metadata-level validation rule
// This allows validation to be defined at the metadata level, not in code
type ValidationRule struct {
	ID           string                 `json:"id"`
	TenantID     string                 `json:"tenant_id"`
	BOKey        string                 `json:"bo_key"`
	FieldName    string                 `json:"field_name"`
	RuleType     string                 `json:"rule_type"`  // required, format, range, custom
	Expression   string                 `json:"expression"` // CEL or JSONLogic expression
	ErrorMessage string                 `json:"error_message"`
	Severity     string                 `json:"severity"` // error, warning, info
	Enabled      bool                   `json:"enabled"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ValidationRuleType constants
const (
	ValidationRuleRequired = "required"
	ValidationRuleFormat   = "format"
	ValidationRuleRange    = "range"
	ValidationRuleCustom   = "custom"
	ValidationRuleUnique   = "unique"
	ValidationRuleRegex    = "regex"
)

// ValidationSeverity constants
const (
	SeverityError   = "error"
	SeverityWarning = "warning"
	SeverityInfo    = "info"
)

// ValidateField validates a field value against metadata-defined rules
func ValidateField(
	bo *BusinessObjectDefinition,
	fieldName string,
	value interface{},
	rules []ValidationRule,
) []ValidationError {
	var errors []ValidationError

	// Find the field definition
	var field *FieldDefinition
	for i := range bo.Fields {
		if bo.Fields[i].Name == fieldName {
			field = &bo.Fields[i]
			break
		}
	}

	if field == nil {
		return []ValidationError{{
			Field:    fieldName,
			Message:  fmt.Sprintf("Field %s not found in business object %s", fieldName, bo.Name),
			Severity: SeverityError,
		}}
	}

	// Check required
	if field.IsRequired && (value == nil || value == "") {
		errors = append(errors, ValidationError{
			Field:    fieldName,
			Message:  fmt.Sprintf("Field %s is required", field.Label),
			Severity: SeverityError,
		})
	}

	// Apply metadata-defined validation rules
	for _, rule := range rules {
		if !rule.Enabled || rule.FieldName != fieldName {
			continue
		}

		if err := applyValidationRule(field, value, rule); err != nil {
			errors = append(errors, *err)
		}
	}

	// Check field-level validation JSON if present
	if len(field.ValidationJSON) > 0 {
		if err := validateAgainstSchema(field, value); err != nil {
			errors = append(errors, *err)
		}
	}

	return errors
}

// ValidationError represents a validation error
type ValidationError struct {
	Field    string `json:"field"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
	RuleID   string `json:"rule_id,omitempty"`
}

// applyValidationRule applies a single validation rule
func applyValidationRule(field *FieldDefinition, value interface{}, rule ValidationRule) *ValidationError {
	switch rule.RuleType {
	case ValidationRuleRequired:
		if value == nil || value == "" {
			return &ValidationError{
				Field:    field.Name,
				Message:  rule.ErrorMessage,
				Severity: rule.Severity,
				RuleID:   rule.ID,
			}
		}

	case ValidationRuleFormat:
		// Format validation (email, phone, etc.)
		if !validateFormat(value, rule.Expression) {
			return &ValidationError{
				Field:    field.Name,
				Message:  rule.ErrorMessage,
				Severity: rule.Severity,
				RuleID:   rule.ID,
			}
		}

	case ValidationRuleRange:
		// Range validation for numbers
		if !validateRange(value, rule.Expression) {
			return &ValidationError{
				Field:    field.Name,
				Message:  rule.ErrorMessage,
				Severity: rule.Severity,
				RuleID:   rule.ID,
			}
		}

	case ValidationRuleRegex:
		// Regex validation
		if !validateRegex(value, rule.Expression) {
			return &ValidationError{
				Field:    field.Name,
				Message:  rule.ErrorMessage,
				Severity: rule.Severity,
				RuleID:   rule.ID,
			}
		}

	case ValidationRuleCustom:
		// Custom CEL or JSONLogic expression
		// TODO: Implement CEL/JSONLogic evaluation
		// For now, just log that custom validation is not yet implemented
	}

	return nil
}

// validateAgainstSchema validates against JSON schema in field definition
func validateAgainstSchema(field *FieldDefinition, value interface{}) *ValidationError {
	// Parse the validation JSON
	var schema map[string]interface{}
	if err := json.Unmarshal(field.ValidationJSON, &schema); err != nil {
		return &ValidationError{
			Field:    field.Name,
			Message:  "Invalid validation schema",
			Severity: SeverityError,
		}
	}

	// TODO: Implement JSON schema validation
	// For now, just return nil
	return nil
}

// Helper validation functions
func validateFormat(value interface{}, format string) bool {
	// TODO: Implement format validation (email, phone, url, etc.)
	return true
}

func validateRange(value interface{}, rangeExpr string) bool {
	// TODO: Implement range validation
	return true
}

func validateRegex(value interface{}, pattern string) bool {
	// TODO: Implement regex validation
	return true
}

// ValidateBusinessObject validates an entire business object instance
func ValidateBusinessObject(
	bo *BusinessObjectDefinition,
	instanceData map[string]interface{},
	rules []ValidationRule,
) []ValidationError {
	var allErrors []ValidationError

	// Validate each field
	for _, field := range bo.Fields {
		value := instanceData[field.Name]

		// Filter rules for this field
		var fieldRules []ValidationRule
		for _, rule := range rules {
			if rule.FieldName == field.Name {
				fieldRules = append(fieldRules, rule)
			}
		}

		errors := ValidateField(bo, field.Name, value, fieldRules)
		allErrors = append(allErrors, errors...)
	}

	return allErrors
}

// GetValidationRulesForBO retrieves all validation rules for a business object
func GetValidationRulesForBO(cache *MetadataCache, tenantID, boKey string) ([]ValidationRule, error) {
	// TODO: Load validation rules from database/cache
	// For now, return empty list
	return []ValidationRule{}, nil
}
