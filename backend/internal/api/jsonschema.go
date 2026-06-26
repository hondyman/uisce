package api

import (
	"embed"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v5"
	_ "github.com/santhosh-tekuri/jsonschema/v5/httploader" // Required for resolving remote schemas
)

//go:embed financial_template.schema.json
var schemaFS embed.FS

var compiled *jsonschema.Schema

// Init compiles the embedded JSON schema for validation.
func Init() error {
	compiler := jsonschema.NewCompiler()
	file, err := schemaFS.Open("financial_template.schema.json")
	if err != nil {
		return fmt.Errorf("failed to open schema file: %w", err)
	}
	defer file.Close()
	if err := compiler.AddResource("schema.json", file); err != nil {
		return fmt.Errorf("failed to add schema resource: %w", err)
	}
	compiled, err = compiler.Compile("schema.json")
	return err
}

// Validate checks if a given value conforms to the compiled financial template schema.
func Validate(v any) error {
	if compiled == nil {
		return fmt.Errorf("schema not initialized")
	}
	return compiled.Validate(v)
}
