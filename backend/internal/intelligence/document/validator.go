package document

import (
	"fmt"

	"github.com/xeipuuv/gojsonschema"
)

// ValidationResult captures the outcome of the schema check.
type ValidationResult struct {
	IsValid    bool
	Errors     []string
	ResultJSON string
}

// ValidateAgainstODS checks the raw JSON against the defined ODS schema.
func ValidateAgainstODS(rawJSON string, odsSchema string) (*ValidationResult, error) {
	// Load the schema and the document.
	schemaLoader := gojsonschema.NewStringLoader(odsSchema)
	documentLoader := gojsonschema.NewStringLoader(rawJSON)

	// Perform the validation.
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		// This indicates a systemic failure in the validation engine or schema parsing,
		// not necessarily the document content.
		return nil, fmt.Errorf("validation engine failed: %w", err)
	}

	valResult := &ValidationResult{
		IsValid:    result.Valid(),
		ResultJSON: rawJSON,
	}

	// If invalid, aggregate specific field errors for the human reviewer.
	if !result.Valid() {
		var errMsgs []string
		for _, desc := range result.Errors() {
			// Format: "Field <field> failed validation: <description>"
			msg := fmt.Sprintf("Field '%s': %s", desc.Field(), desc.Description())
			errMsgs = append(errMsgs, msg)
		}
		valResult.Errors = errMsgs
	}

	return valResult, nil
}
