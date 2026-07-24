package filters

import (
	"context"
	"fmt"
)

// ListLookupFilter checks if a field value exists in a reference list
type ListLookupFilter struct {
	FieldName     string   // Field to check
	ReferenceList []string // Allowed values
	AllowMissing  bool     // If true, missing field passes
}

func (f *ListLookupFilter) Name() string {
	return "List Lookup"
}

func (f *ListLookupFilter) Purify(ctx context.Context, data map[string]interface{}) error {
	value, ok := data[f.FieldName]
	if !ok {
		if f.AllowMissing {
			return nil
		}
		return fmt.Errorf("field '%s' not found in data", f.FieldName)
	}

	strValue, ok := value.(string)
	if !ok {
		return fmt.Errorf("field '%s' is not a string", f.FieldName)
	}

	for _, allowed := range f.ReferenceList {
		if strValue == allowed {
			return nil
		}
	}

	return fmt.Errorf("value '%s' not found in reference list", strValue)
}
