package validation

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/xeipuuv/gojsonschema"
)

// ValidateUpgradeArtifacts validates upgrade artifacts against the JSON schema
func ValidateUpgradeArtifacts(data []byte) error {
	schemaPath, err := filepath.Abs("schemas/upgrade-artifacts-data.schema.json")
	if err != nil {
		return fmt.Errorf("failed to get schema path: %w", err)
	}

	schemaLoader := gojsonschema.NewReferenceLoader("file://" + schemaPath)
	docLoader := gojsonschema.NewBytesLoader(data)

	result, err := gojsonschema.Validate(schemaLoader, docLoader)
	if err != nil {
		return fmt.Errorf("schema validation error: %w", err)
	}

	if !result.Valid() {
		var errors []string
		for _, err := range result.Errors() {
			errors = append(errors, err.String())
		}
		return fmt.Errorf("invalid upgrade artifact: %v", errors)
	}

	return nil
}

// ValidateUpgradeArtifactsFromStruct validates a struct by marshaling to JSON first
func ValidateUpgradeArtifactsFromStruct(artifact interface{}) error {
	data, err := json.Marshal(artifact)
	if err != nil {
		return fmt.Errorf("failed to marshal artifact: %w", err)
	}

	return ValidateUpgradeArtifacts(data)
}
