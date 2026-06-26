package cube

import (
	"fmt"
	"strings"
)

// ExampleUsage demonstrates how to use the cube package
func ExampleUsage() {
	// Example 1: Setting configuration options
	fmt.Println("=== Configuration Example ===")

	// Set base path
	GlobalConfig.Set("base_path", "/cube-api")
	fmt.Printf("Base path: %s\n", GlobalConfig.Get("base_path"))

	// Set context to app ID function
	GlobalConfig.Set("context_to_app_id", func(ctx map[string]interface{}) string {
		if securityCtx, ok := ctx["securityContext"].(map[string]interface{}); ok {
			if tenantID, ok := securityCtx["tenant_id"].(string); ok {
				return tenantID
			}
		}
		return "default"
	})

	// Set query rewrite function
	GlobalConfig.Set("query_rewrite", func(query map[string]interface{}, ctx map[string]interface{}) map[string]interface{} {
		if measures, ok := query["measures"].([]interface{}); ok {
			measures = append(measures, "orders.count")
			query["measures"] = measures
		}
		return query
	})

	// Example 2: Using decorators
	fmt.Println("\n=== Decorator Example ===")

	// Using decorator for context_to_app_id
	ConfigDecorator("context_to_app_id")(func(ctx map[string]interface{}) string {
		return "decorated_tenant"
	})

	// Using decorator for query_rewrite
	ConfigDecorator("query_rewrite")(func(query map[string]interface{}, ctx map[string]interface{}) map[string]interface{} {
		fmt.Println("Query rewrite called")
		return query
	})

	// Example 3: Template context
	fmt.Println("\n=== Template Context Example ===")

	template := NewTemplateContext()

	// Add variables
	template.AddVariable("my_var", 123)
	template.AddVariable("app_name", "My Cube App")

	// Add functions
	template.AddFunction("get_data", func() int {
		return 42
	})

	template.AddFunction("format_name", func(name string) string {
		return fmt.Sprintf("User: %s", name)
	})

	// Add filters
	template.AddFilter("wrap", func(data string) string {
		return fmt.Sprintf("< %s >", data)
	})

	template.AddFilter("uppercase", func(data string) string {
		return strings.ToUpper(data)
	})

	// Using decorators
	template.Function("get_more_data")(func() string {
		return "more data"
	})

	template.Filter("wrap_more")(func(data string) string {
		return fmt.Sprintf("<<< %s >>>", data)
	})

	// Example 4: Global template context
	fmt.Println("\n=== Global Template Context Example ===")

	globalTemplate := GetGlobalTemplateContext()
	globalTemplate.AddVariable("global_var", "I'm global")
	globalTemplate.AddFunction("global_func", func() string {
		return "Hello from global function"
	})

	fmt.Println("Cube package setup complete!")
	fmt.Printf("Config base_path: %v\n", GlobalConfig.Get("base_path"))
	fmt.Printf("Template variables count: %d\n", len(template.GetVariables()))
	fmt.Printf("Template functions count: %d\n", len(template.GetFunctions()))
	fmt.Printf("Template filters count: %d\n", len(template.GetFilters()))
}
