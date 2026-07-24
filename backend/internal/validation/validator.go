package validation

import (
	"fmt"
	"path/filepath"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// Validator encapsulates JSON Schema loading and validation logic
type Validator struct {
	compiler *jsonschema.Compiler
	schemas  map[string]*jsonschema.Schema
}

// NewValidator initializes the schema compiler and compiles local JSON schemas
func NewValidator(schemaDir string) (*Validator, error) {
	c := jsonschema.NewCompiler()

	// Pre-load schemas
	files := []string{
		"compliance_context.schema.json",
		"factor_model_context.schema.json",
		"var_context.schema.json",
		"scenario_context.schema.json",
	}

	schemas := make(map[string]*jsonschema.Schema)

	for _, file := range files {
		path := filepath.Join(schemaDir, file)
		schema, err := c.Compile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to compile %s: %w", file, err)
		}
		// Use filename as the key
		schemas[file] = schema
	}

	return &Validator{
		compiler: c,
		schemas:  schemas,
	}, nil
}

// Validate against a named schema
func (v *Validator) Validate(schemaName string, payload interface{}) error {
	schema, ok := v.schemas[schemaName]
	if !ok {
		return fmt.Errorf("schema %s not loaded", schemaName)
	}

	if err := schema.Validate(payload); err != nil {
		return fmt.Errorf("schema validation failed for %s: %w", schemaName, err)
	}

	return nil
}
