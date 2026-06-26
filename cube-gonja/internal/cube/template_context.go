package cube

import (
	"fmt"
	"sync"

	"github.com/nikolalohinski/gonja/v2/exec"
)

// TemplateContext manages variables, functions, and filters for Gonja templates
type TemplateContext struct {
	mu sync.RWMutex

	// Variables available in templates
	variables map[string]interface{}

	// Functions available in templates
	functions map[string]interface{}

	// Filters available in templates
	filters map[string]interface{}
}

// NewTemplateContext creates a new TemplateContext instance
func NewTemplateContext() *TemplateContext {
	return &TemplateContext{
		variables: make(map[string]interface{}),
		functions: make(map[string]interface{}),
		filters:   make(map[string]interface{}),
	}
}

// AddVariable registers a variable in the template context
func (tc *TemplateContext) AddVariable(name string, value interface{}) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.variables[name] = value
}

// AddFunction registers a function in the template context
func (tc *TemplateContext) AddFunction(name string, fn interface{}) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.functions[name] = fn
}

// AddFilter registers a filter in the template context
func (tc *TemplateContext) AddFilter(name string, filter interface{}) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.filters[name] = filter
}

// Function is a decorator for registering functions
func (tc *TemplateContext) Function(name string) func(interface{}) {
	return func(fn interface{}) {
		tc.AddFunction(name, fn)
	}
}

// Filter is a decorator for registering filters
func (tc *TemplateContext) Filter(name string) func(interface{}) {
	return func(filter interface{}) {
		tc.AddFilter(name, filter)
	}
}

// GetVariables returns a copy of the variables map
func (tc *TemplateContext) GetVariables() map[string]interface{} {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	result := make(map[string]interface{})
	for k, v := range tc.variables {
		result[k] = v
	}
	return result
}

// GetFunctions returns a copy of the functions map
func (tc *TemplateContext) GetFunctions() map[string]interface{} {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	result := make(map[string]interface{})
	for k, v := range tc.functions {
		result[k] = v
	}
	return result
}

// GetFilters returns a copy of the filters map
func (tc *TemplateContext) GetFilters() map[string]interface{} {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	result := make(map[string]interface{})
	for k, v := range tc.filters {
		result[k] = v
	}
	return result
}

// ToGonjaContext converts the TemplateContext to a Gonja context
func (tc *TemplateContext) ToGonjaContext() *exec.Context {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	ctx := exec.NewContext(tc.variables)

	// Add functions to context
	for name, fn := range tc.functions {
		ctx.Set(name, fn)
	}

	// Note: Gonja doesn't have built-in filter support like Jinja,
	// but we can expose filters as functions with a different naming convention
	for name, filter := range tc.filters {
		ctx.Set("filter_"+name, filter)
	}

	return ctx
}

// FilterParams represents the FILTER_PARAMS context variable
type FilterParams struct {
	mu     sync.RWMutex
	params map[string]interface{}
}

// NewFilterParams creates a new FilterParams instance
func NewFilterParams() *FilterParams {
	return &FilterParams{
		params: make(map[string]interface{}),
	}
}

// Get returns filter params for a specific cube
func (fp *FilterParams) Get(cubeName string) interface{} {
	fp.mu.RLock()
	defer fp.mu.RUnlock()
	if params, exists := fp.params[cubeName]; exists {
		return params
	}
	return NewCubeFilterParams(cubeName)
}

// CubeFilterParams represents filter params for a specific cube
type CubeFilterParams struct {
	CubeName string
	mu       sync.RWMutex
	filters  map[string]interface{}
}

// NewCubeFilterParams creates filter params for a specific cube
func NewCubeFilterParams(cubeName string) *CubeFilterParams {
	return &CubeFilterParams{
		CubeName: cubeName,
		filters:  make(map[string]interface{}),
	}
}

// Filter creates a filter function for a member
func (cfp *CubeFilterParams) Filter(memberName string) func(sqlExpr interface{}) string {
	return func(sqlExpr interface{}) string {
		// This would be replaced with actual filter logic in a real implementation
		// For now, return a placeholder that can be processed by the query engine
		return fmt.Sprintf("{FILTER_PARAMS.%s.%s.filter(%v)}", cfp.CubeName, memberName, sqlExpr)
	}
}

// FilterGroup combines multiple filter expressions
func FilterGroup(filters ...interface{}) string {
	// Combine filters with AND logic
	result := "{FILTER_GROUP("
	for i, filter := range filters {
		if i > 0 {
			result += ", "
		}
		result += fmt.Sprintf("%v", filter)
	}
	result += ")}"
	return result
}

// SQLUtils provides SQL utility functions
type SQLUtils struct{}

// NewSQLUtils creates a new SQLUtils instance
func NewSQLUtils() *SQLUtils {
	return &SQLUtils{}
}

// ConvertTz converts timezone for timestamps
func (su *SQLUtils) ConvertTz(timestampExpr string) string {
	return fmt.Sprintf("CONVERT_TZ(%s, @@session.time_zone, '+00:00')", timestampExpr)
}

// CompileContext represents the COMPILE_CONTEXT variable
type CompileContext struct {
	mu              sync.RWMutex
	SecurityContext map[string]interface{} `json:"securityContext"`
	Extra           map[string]interface{} `json:"extra,omitempty"`
}

// NewCompileContext creates a new CompileContext instance
func NewCompileContext() *CompileContext {
	return &CompileContext{
		SecurityContext: make(map[string]interface{}),
		Extra:           make(map[string]interface{}),
	}
}

// SetSecurityContext sets the security context
func (cc *CompileContext) SetSecurityContext(key string, value interface{}) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.SecurityContext[key] = value
}

// SetExtra sets extra context variables
func (cc *CompileContext) SetExtra(key string, value interface{}) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.Extra[key] = value
}

// SecurityContext represents the SECURITY_CONTEXT variable (deprecated)
type SecurityContext struct {
	mu      sync.RWMutex
	filters map[string]interface{}
}

// NewSecurityContext creates a new SecurityContext instance
func NewSecurityContext() *SecurityContext {
	return &SecurityContext{
		filters: make(map[string]interface{}),
	}
}

// Filter creates a security filter
func (sc *SecurityContext) Filter(fieldName string) string {
	return fmt.Sprintf("{SECURITY_CONTEXT.%s.filter(%s)}", fieldName, fieldName)
}

// RequiredFilter creates a required security filter
func (sc *SecurityContext) RequiredFilter(fieldName string) string {
	return fmt.Sprintf("{SECURITY_CONTEXT.%s.requiredFilter(%s)}", fieldName, fieldName)
}

// UnsafeValue gets an unsafe value from security context
func (sc *SecurityContext) UnsafeValue() interface{} {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	// This would need to be implemented based on actual security context
	return nil
}

// Global template context instance
var globalTemplateContext = NewTemplateContext()

// Initialize global context with Cube.js context variables
func init() {
	// Initialize CUBE context variable
	globalTemplateContext.AddVariable("CUBE", "{CUBE}")

	// Initialize FILTER_PARAMS
	globalTemplateContext.AddVariable("FILTER_PARAMS", NewFilterParams())

	// Initialize FILTER_GROUP function
	globalTemplateContext.AddFunction("FILTER_GROUP", FilterGroup)

	// Initialize SQL_UTILS
	globalTemplateContext.AddVariable("SQL_UTILS", NewSQLUtils())

	// Initialize COMPILE_CONTEXT
	globalTemplateContext.AddVariable("COMPILE_CONTEXT", NewCompileContext())

	// Initialize SECURITY_CONTEXT (deprecated but included for compatibility)
	globalTemplateContext.AddVariable("SECURITY_CONTEXT", NewSecurityContext())
}

// GetGlobalTemplateContext returns the global template context
func GetGlobalTemplateContext() *TemplateContext {
	return globalTemplateContext
}
