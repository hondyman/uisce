//go:build ignore

package main

import (
	"cube-gonja/internal/cube"
	"fmt"
)

func main() {
	// Test that the cube package is working
	fmt.Println("Testing cube package integration...")

	// Test global config
	config := cube.GetConfig()
	fmt.Printf("Config object: %+v\n", config)

	// Test global template context
	templateCtx := cube.GetGlobalTemplateContext()
	fmt.Printf("Template context has %d variables\n", len(templateCtx.GetVariables()))
	fmt.Printf("Template context has %d functions\n", len(templateCtx.GetFunctions()))
	fmt.Printf("Template context has %d filters\n", len(templateCtx.GetFilters()))

	// Test that we can access the global instances
	fmt.Printf("Global config instance: %p\n", cube.GlobalConfig)
	fmt.Printf("Global template instance: %p\n", cube.GlobalTemplate)

	fmt.Println("Cube package integration test completed successfully!")
}
