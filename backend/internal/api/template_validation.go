package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	models "github.com/hondyman/semlayer/backend/internal/models"
)

// ============================================================================
// Template Validation & Parameter Processing
// ============================================================================

// ValidateTemplateSpec validates a complete template specification
// Returns validation errors including parameter type mismatches, required fields, etc.
func ValidateTemplateSpec(template *SemanticQueryTemplate, bundle *models.DataBundle) error {
	// Validate basic fields
	if template.Name == "" {
		return errors.New("template name is required")
	}
	if template.Datasource == "" {
		return errors.New("datasource is required")
	}
	if template.SemanticQuery == nil {
		return errors.New("semantic_query is required")
	}

	// Validate semantic query structure
	if err := ValidateSemanticQuery(template.SemanticQuery, bundle); err != nil {
		return fmt.Errorf("invalid semantic query: %w", err)
	}

	// Validate parameter definitions
	for _, param := range template.Parameters {
		if err := ValidateParamDefinition(param); err != nil {
			return fmt.Errorf("invalid parameter %q: %w", param.Name, err)
		}
	}

	// Validate that all placeholders can be resolved
	if err := ValidateParameterPlaceholders(template); err != nil {
		return err
	}

	// Validate visibility setting
	if !isValidVisibility(template.Visibility) {
		return fmt.Errorf("invalid visibility %q", template.Visibility)
	}

	return nil
}

// ValidateParamDefinition validates a single parameter definition
func ValidateParamDefinition(param TemplateParamDef) error {
	if param.Name == "" {
		return errors.New("parameter name is required")
	}

	// Validate parameter name (alphanumeric + underscore)
	if !regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`).MatchString(param.Name) {
		return errors.New("parameter name must be alphanumeric (with underscores)")
	}

	// Validate type
	validTypes := map[string]bool{
		"string": true,
		"number": true,
		"bool":   true,
	}
	if !validTypes[string(param.Type)] {
		return fmt.Errorf("invalid parameter type %q", param.Type)
	}

	// Validate default value matches type
	if param.Default != nil {
		if err := validateParamValue(param.Name, param.Default, string(param.Type)); err != nil {
			return fmt.Errorf("invalid default value: %w", err)
		}
	}

	return nil
}

// ValidateParameterPlaceholders ensures all {{placeholder}} references exist as parameter definitions
func ValidateParameterPlaceholders(template *SemanticQueryTemplate) error {
	// Extract all placeholders from semantic query
	placeholders := extractPlaceholders(template.SemanticQuery)

	// Check that each placeholder has a corresponding parameter definition
	paramNames := make(map[string]bool)
	for _, param := range template.Parameters {
		paramNames[param.Name] = true
	}

	for placeholder := range placeholders {
		if !paramNames[placeholder] {
			return fmt.Errorf("placeholder %q has no parameter definition", placeholder)
		}
	}

	return nil
}

// ValidateSemanticQuery validates the structure and fields of a semantic query
func ValidateSemanticQuery(query *SemanticQuery, bundle *models.DataBundle) error {
	if query.Datasource == "" {
		return errors.New("datasource is required in semantic query")
	}

	// Validate select fields exist in bundle
	if bundle != nil {
		for _, field := range query.Select {
			if !bundle.HasField(field) {
				return fmt.Errorf("field %q not found in datasource", field)
			}
		}

		// Validate filter fields
		if err := validateFilters(query.Filters, bundle); err != nil {
			return fmt.Errorf("invalid filters: %w", err)
		}
	}

	return nil
}

// ============================================================================
// Parameter Value Resolution & Injection
// ============================================================================

// ResolveTemplateParameters applies parameter values to create execution context
// Returns the resolved parameters with validation and type coercion
func ResolveTemplateParameters(ctx context.Context, template *SemanticQueryTemplate, params map[string]interface{}) (ResolvedParameters, error) {
	resolved := make(ResolvedParameters)

	for _, paramDef := range template.Parameters {
		// Get parameter value from input or use default
		var value interface{}
		if v, exists := params[paramDef.Name]; exists {
			value = v
		} else if paramDef.Default != nil {
			value = paramDef.Default
		} else if paramDef.Required {
			return nil, fmt.Errorf("required parameter %q not provided", paramDef.Name)
		} else {
			continue // Optional parameter not provided
		}

		// Validate and coerce type
		coercedValue, err := coerceParamType(paramDef.Name, value, string(paramDef.Type))
		if err != nil {
			return nil, err
		}

		resolved[paramDef.Name] = coercedValue
	}

	return resolved, nil
}

// ApplyTemplatePlaceholders substitutes {{placeholder}} with resolved parameter values
// This mutates a deep copy of the semantic query
func ApplyTemplatePlaceholders(query *SemanticQuery, params ResolvedParameters) (*SemanticQuery, error) {
	// Marshal to JSON for placeholder substitution
	queryJSON, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	// Perform placeholder substitution
	substituted := substituteAllPlaceholders(string(queryJSON), params)

	// Unmarshal back to SemanticQuery
	resultQuery := &SemanticQuery{}
	if err := json.Unmarshal([]byte(substituted), resultQuery); err != nil {
		return nil, fmt.Errorf("failed to unmarshal substituted query: %w", err)
	}

	return resultQuery, nil
}

// ============================================================================
// Internal Placeholder & Parameter Helper Functions
// ============================================================================

// extractPlaceholders finds all {{placeholder}} references in a semantic query
func extractPlaceholders(query *SemanticQuery) map[string]bool {
	queryJSON, _ := json.Marshal(query)
	placeholders := make(map[string]bool)

	// Regex to find {{placeholder}} patterns
	re := regexp.MustCompile(`\{\{([a-zA-Z_][a-zA-Z0-9_]*)\}\}`)
	matches := re.FindAllStringSubmatch(string(queryJSON), -1)

	for _, match := range matches {
		if len(match) > 1 {
			placeholders[match[1]] = true
		}
	}

	return placeholders
}

// substituteAllPlaceholders replaces all {{placeholder}} with parameter values
// Handles string quoting and type-appropriate formatting
func substituteAllPlaceholders(jsonStr string, params ResolvedParameters) string {
	re := regexp.MustCompile(`\{\{([a-zA-Z_][a-zA-Z0-9_]*)\}\}`)

	return re.ReplaceAllStringFunc(jsonStr, func(match string) string {
		// Extract parameter name
		paramName := match[2 : len(match)-2] // Remove {{ and }}

		if value, exists := params[paramName]; exists {
			// Format value based on type
			switch v := value.(type) {
			case string:
				// Escape quotes and wrap in quotes
				return fmt.Sprintf(`"%s"`, strings.ReplaceAll(v, `"`, `\"`))
			case float64:
				// Numbers don't need quotes
				return fmt.Sprintf(`%v`, v)
			case bool:
				// Booleans as lowercase
				return fmt.Sprintf(`%v`, v)
			default:
				// Fallback: JSON encode
				if jsonBytes, err := json.Marshal(v); err == nil {
					return string(jsonBytes)
				}
				return match // Return unsubstituted if encoding fails
			}
		}

		return match // Return unsubstituted if parameter not found
	})
}

// coerceParamType coerces a value to the expected parameter type
func coerceParamType(name string, value interface{}, paramType string) (interface{}, error) {
	switch paramType {
	case "string":
		switch v := value.(type) {
		case string:
			return v, nil
		case float64:
			return fmt.Sprintf("%v", v), nil
		case bool:
			return fmt.Sprintf("%v", v), nil
		default:
			return fmt.Sprintf("%v", v), nil
		}

	case "number":
		switch v := value.(type) {
		case float64:
			return v, nil
		case int:
			return float64(v), nil
		case string:
			// Try to parse string as number
			var f float64
			if _, err := fmt.Sscanf(v, "%f", &f); err == nil {
				return f, nil
			}
			return nil, fmt.Errorf("value %q is not a valid number", v)
		default:
			return nil, fmt.Errorf("cannot coerce %T to number", value)
		}

	case "bool":
		switch v := value.(type) {
		case bool:
			return v, nil
		case string:
			switch strings.ToLower(v) {
			case "true", "1", "yes":
				return true, nil
			case "false", "0", "no":
				return false, nil
			default:
				return nil, fmt.Errorf("value %q is not a valid boolean", v)
			}
		case float64:
			return v != 0, nil
		default:
			return nil, fmt.Errorf("cannot coerce %T to boolean", value)
		}

	default:
		return nil, fmt.Errorf("unknown parameter type %q", paramType)
	}
}

// validateParamValue validates that a value matches the expected type
func validateParamValue(name string, value interface{}, paramType string) error {
	switch paramType {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("expected string, got %T", value)
		}
	case "number":
		if _, ok := value.(float64); !ok {
			if _, ok := value.(int); !ok {
				return fmt.Errorf("expected number, got %T", value)
			}
		}
	case "bool":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("expected boolean, got %T", value)
		}
	}
	return nil
}

// validateFilters checks that all filter fields exist in the bundle
func validateFilters(filters []Filter, bundle *models.DataBundle) error {
	for _, f := range filters {
		if !bundle.HasField(f.Field) {
			return fmt.Errorf("filter field %q not found in datasource", f.Field)
		}
	}
	return nil
}

// isValidVisibility checks if a visibility setting is valid
func isValidVisibility(visibility string) bool {
	valid := map[string]bool{
		"private": true,
		"team":    true,
		"public":  true,
	}
	return valid[visibility]
}

// ============================================================================
// Template Diff & Comparison
// ============================================================================

// ComparTemplateVersions generates a diff between two template versions
type TemplateDiff struct {
	NameChanged        bool
	DescriptionChanged bool
	QueryChanged       bool
	ParametersChanged  bool
	Changes            map[string]interface{}
}

func DiffTemplateVersions(v1, v2 *TemplateVersion) *TemplateDiff {
	diff := &TemplateDiff{
		Changes: make(map[string]interface{}),
	}

	if v1.Name != v2.Name {
		diff.NameChanged = true
		diff.Changes["name"] = map[string]string{
			"from": v1.Name,
			"to":   v2.Name,
		}
	}

	// Compare query JSON
	q1JSON, _ := json.Marshal(v1.SemanticQuery)
	q2JSON, _ := json.Marshal(v2.SemanticQuery)
	if string(q1JSON) != string(q2JSON) {
		diff.QueryChanged = true
		diff.Changes["semantic_query"] = map[string]interface{}{
			"from": v1.SemanticQuery,
			"to":   v2.SemanticQuery,
		}
	}

	// Compare parameters JSON
	p1JSON, _ := json.Marshal(v1.Parameters)
	p2JSON, _ := json.Marshal(v2.Parameters)
	if string(p1JSON) != string(p2JSON) {
		diff.ParametersChanged = true
		diff.Changes["parameters"] = map[string]interface{}{
			"from": v1.Parameters,
			"to":   v2.Parameters,
		}
	}

	return diff
}

// ============================================================================
// Type Definitions
// ============================================================================

// ResolvedParameters is a map of parameter names to their resolved values
type ResolvedParameters map[string]interface{}
