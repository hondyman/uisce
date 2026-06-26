package api

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// ============================================================================
// Template Data Models & Types
// ============================================================================

type TemplateParamType string

const (
	ParamString TemplateParamType = "string"
	ParamNumber TemplateParamType = "number"
	ParamBool   TemplateParamType = "bool"
)

type TemplateParamDef struct {
	Name     string            `json:"name" db:"name"`
	Type     TemplateParamType `json:"type" db:"type"`
	Required bool              `json:"required" db:"required"`
	Default  interface{}       `json:"default,omitempty" db:"default"`
	Help     string            `json:"help,omitempty" db:"help"` // User-facing help text
}

// SemanticQueryTemplate represents a saved, parameterized semantic query
type SemanticQueryTemplate struct {
	ID            string               `json:"id" db:"id"`
	TenantID      string               `json:"tenant_id,omitempty" db:"tenant_id"`
	Name          string               `json:"name" db:"name"`
	Description   string               `json:"description,omitempty" db:"description"`
	Datasource    string               `json:"datasource" db:"datasource"`
	Version       string               `json:"version" db:"version"`
	SemanticQuery *SemanticQuery       `json:"semantic_query" db:"-"`
	Parameters    []TemplateParamDef   `json:"parameters" db:"-"`
	CreatedBy     string               `json:"created_by" db:"created_by"`
	CreatedAt     time.Time            `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time            `json:"updated_at" db:"updated_at"`
	Visibility    string               `json:"visibility" db:"visibility"` // "private", "team", "public"
	Tags          []string             `json:"tags" db:"-"`
	Deprecated    bool                 `json:"deprecated" db:"deprecated"`
	DeprecatedAt  *time.Time           `json:"deprecated_at,omitempty" db:"deprecated_at"`
	DeprecationReason string            `json:"deprecation_reason,omitempty" db:"deprecation_reason"`
}

// TemplateVersion represents a version of a template in the versioning system
type TemplateVersion struct {
	VersionID       string             `json:"version_id" db:"version_id"`
	TemplateID      string             `json:"template_id" db:"template_id"`
	Version         int                `json:"version" db:"version"`
	Name            string             `json:"name" db:"name"`
	SemanticQuery   *SemanticQuery     `json:"semantic_query" db:"-"`
	Parameters      []TemplateParamDef `json:"parameters" db:"-"`
	CreatedAt       time.Time          `json:"created_at" db:"created_at"`
	CreatedBy       string             `json:"created_by" db:"created_by"`
	ChangeMessage   string             `json:"change_message,omitempty" db:"change_message"`
	IsPromoted      bool               `json:"is_promoted" db:"is_promoted"`
	PromotedAt      *time.Time         `json:"promoted_at,omitempty" db:"promoted_at"`
}

// TemplateRunRequest represents a request to execute a template with parameters
type TemplateRunRequest struct {
	Params map[string]interface{} `json:"params"`
}

// TemplateRunResponse represents the result of running a template
type TemplateRunResponse struct {
	Datasource string              `json:"datasource"`
	Version    string              `json:"version"`
	SQL        string              `json:"sql"`
	Rows       []map[string]interface{} `json:"rows"`
	Count      int                 `json:"count"`
	Error      string              `json:"error,omitempty"`
	ExecutedAt time.Time           `json:"executed_at"`
	Duration   int64               `json:"duration_ms"`
}

// TemplateListQueryParams holds filtering options for template listing
type TemplateListQueryParams struct {
	Datasource  string
	Version     string
	CreatedBy   string
	Tag         string
	Visibility  string
	ShowDeprecated bool
	Limit       int
	Offset      int
}

// TemplatePermission represents access control for a template
type TemplatePermission struct {
	TemplateID string   `json:"template_id" db:"template_id"`
	Role       string   `json:"role" db:"role"` // "viewer", "editor", "admin"
	CanRun     bool     `json:"can_run" db:"can_run"`
	CanEdit    bool     `json:"can_edit" db:"can_edit"`
	CanDelete  bool     `json:"can_delete" db:"can_delete"`
	CanPromote bool     `json:"can_promote" db:"can_promote"`
}

// ============================================================================
// Database Serialization (for JSONB columns)
// ============================================================================

// Value implements driver.Valuer for storing in database
func (q *SemanticQuery) Value() (driver.Value, error) {
	return json.Marshal(q)
}

// Scan implements sql.Scanner for reading from database
func (q *SemanticQuery) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("type assertion failed")
	}
	return json.Unmarshal(bytes, &q)
}

// ============================================================================
// Parameter Validation
// ============================================================================

// ValidateParam validates a parameter value against its definition
func ValidateParam(def TemplateParamDef, value interface{}) error {
	if value == nil {
		if def.Required {
			return fmt.Errorf("missing required parameter: %s", def.Name)
		}
		return nil
	}

	switch def.Type {
	case ParamString:
		if _, ok := value.(string); !ok {
			return fmt.Errorf("parameter %s: expected string, got %T", def.Name, value)
		}
	case ParamNumber:
		switch value.(type) {
		case float64, int, int64:
			// OK
		default:
			return fmt.Errorf("parameter %s: expected number, got %T", def.Name, value)
		}
	case ParamBool:
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("parameter %s: expected bool, got %T", def.Name, value)
		}
	}

	return nil
}

// ============================================================================
// Parameter Injection & Placeholder Replacement
// ============================================================================

// ApplyTemplateParams substitutes template parameters into a semantic query
// Input: template + runtime parameters
// Output: fully resolved semantic query ready for execution
func ApplyTemplateParams(t *SemanticQueryTemplate, params map[string]interface{}) (*SemanticQuery, error) {
	// 1. Build resolved parameter map with defaults
	resolved := make(map[string]interface{})
	for _, def := range t.Parameters {
		v, ok := params[def.Name]

		// Validate provided parameter
		if ok {
			if err := ValidateParam(def, v); err != nil {
				return nil, err
			}
			resolved[def.Name] = v
			continue
		}

		// Use default if available
		if def.Default != nil {
			resolved[def.Name] = def.Default
			continue
		}

		// Fail if required
		if def.Required {
			return nil, fmt.Errorf("missing required parameter: %s (type: %s)", def.Name, def.Type)
		}
	}

	// 2. Deep copy semantic query (to avoid modifying original)
	b, _ := json.Marshal(t.SemanticQuery)
	var q SemanticQuery
	json.Unmarshal(b, &q)

	// 3. Walk query structure and replace placeholders
	replacer := func(v interface{}) interface{} {
		return replaceValue(v, resolved)
	}

	// Replace in select (less common, but support it)
	for i := range q.Select {
		q.Select[i] = replacer(q.Select[i]).(string)
	}

	// Replace in filters (most common case)
	for i := range q.Filters {
		q.Filters[i].Field = replacer(q.Filters[i].Field).(string)
		q.Filters[i].Value = replacer(q.Filters[i].Value)
	}

	// Replace in order_by
	for i := range q.OrderBy {
		q.OrderBy[i].Field = replacer(q.OrderBy[i].Field).(string)
	}

	// Replace in limit
	if limitVal := replacer(q.Limit); limitVal != nil {
		if n, ok := limitVal.(float64); ok {
			q.Limit = int(n)
		}
	}

	return &q, nil
}

// replaceValue recursively replaces template placeholders in a value
func replaceValue(v interface{}, params map[string]interface{}) interface{} {
	switch val := v.(type) {
	case string:
		// Check if entire string is a placeholder: {{param_name}}
		paramName := extractPlaceholder(val)
		if paramName != "" {
			if paramVal, ok := params[paramName]; ok {
				return paramVal
			}
		}
		// String interpolation: "SELECT * FROM {{table}}" → "SELECT * FROM customers"
		return interpolateString(val, params)

	case float64:
		// Numbers can't contain placeholders, pass through
		return val

	case bool:
		return val

	case map[string]interface{}:
		for k := range val {
			val[k] = replaceValue(val[k], params)
		}
		return val

	case []interface{}:
		for i := range val {
			val[i] = replaceValue(val[i], params)
		}
		return val

	default:
		return v
	}
}

// extractPlaceholder checks if a string is a pure placeholder like "{{param}}"
// Returns the parameter name or empty string
func extractPlaceholder(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "{{") && strings.HasSuffix(s, "}}") {
		name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(s, "{{"), "}}"))
		// Validate parameter name (alphanumeric + underscore)
		if matched, _ := regexp.MatchString(`^[a-zA-Z_][a-zA-Z0-9_]*$`, name); matched {
			return name
		}
	}
	return ""
}

// interpolateString replaces {{param}} patterns within a string
// Example: "country = {{country}}" → "country = US"
func interpolateString(s string, params map[string]interface{}) string {
	pattern := regexp.MustCompile(`\{\{([a-zA-Z_][a-zA-Z0-9_]*)\}\}`)

	result := pattern.ReplaceAllStringFunc(s, func(match string) string {
		name := extractPlaceholder(match)
		if val, ok := params[name]; ok {
			return fmt.Sprintf("%v", val)
		}
		return match // Leave unchanged if param not found
	})

	return result
}

// ============================================================================
// Template Utility Functions
// ============================================================================

// ExtractParametersFromQuery analyzes a semantic query and extracts
// all parameter placeholders used ({{param_name}})
func ExtractParametersFromQuery(q *SemanticQuery) []string {
	params := make(map[string]bool)

	// Scan through all string fields
	scanString := func(s string) {
		pattern := regexp.MustCompile(`\{\{([a-zA-Z_][a-zA-Z0-9_]*)\}\}`)
		matches := pattern.FindAllStringSubmatch(s, -1)
		for _, m := range matches {
			if len(m) > 1 {
				params[m[1]] = true
			}
		}
	}

	// Scan select
	for _, field := range q.Select {
		scanString(field)
	}

	// Scan filters
	for _, f := range q.Filters {
		scanString(f.Field)
		if str, ok := f.Value.(string); ok {
			scanString(str)
		}
	}

	// Scan order_by
	for _, ob := range q.OrderBy {
		scanString(ob.Field)
	}

	// Extract unique param names
	var result []string
	for name := range params {
		result = append(result, name)
	}

	return result
}

// IsTemplateParameterized checks if a semantic query contains any parameters
func IsTemplateParameterized(q *SemanticQuery) bool {
	return len(ExtractParametersFromQuery(q)) > 0
}
