package filters

import (
	"context"
	"fmt"
)

// CrossReferenceFilter validates that a foreign key reference exists
type CrossReferenceFilter struct {
	FieldName      string                      // Field containing the FK value
	LookupTable    string                      // Table to check against
	LookupCallback func(table, id string) bool // Callback to check existence
}

func (f *CrossReferenceFilter) Name() string {
	return "Cross Reference"
}

func (f *CrossReferenceFilter) Purify(ctx context.Context, data map[string]interface{}) error {
	value, ok := data[f.FieldName]
	if !ok {
		return fmt.Errorf("field '%s' not found in data", f.FieldName)
	}

	idValue, ok := value.(string)
	if !ok {
		return fmt.Errorf("field '%s' is not a string ID", f.FieldName)
	}

	if f.LookupCallback == nil {
		// No callback provided, assume valid (for testing)
		return nil
	}

	if !f.LookupCallback(f.LookupTable, idValue) {
		return fmt.Errorf("reference '%s' not found in table '%s'", idValue, f.LookupTable)
	}

	return nil
}
