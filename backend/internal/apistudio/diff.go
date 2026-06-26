package apistudio

import (
	"encoding/json"
	"fmt"
)

// ApiSchemaDiff represents the result of comparing two API endpoints
type ApiSchemaDiff struct {
	Breaking    []string `json:"breaking"`
	NonBreaking []string `json:"non_breaking"`
}

// DiffEndpoints compares two versions of an API endpoint and identifies breaking changes
func DiffEndpoints(old, new *APIEndpoint) ApiSchemaDiff {
	diff := ApiSchemaDiff{
		Breaking:    []string{},
		NonBreaking: []string{},
	}

	// 1. Check fundamental identity
	if old.Method != new.Method {
		diff.Breaking = append(diff.Breaking, fmt.Sprintf("HTTP Method changed: %s -> %s", old.Method, new.Method))
	}
	if old.Path != new.Path {
		diff.Breaking = append(diff.Breaking, fmt.Sprintf("Path changed: %s -> %s", old.Path, new.Path))
	}
	if old.BOName != new.BOName {
		diff.Breaking = append(diff.Breaking, fmt.Sprintf("Underlying Business Object changed: %s -> %s", old.BOName, new.BOName))
	}

	// 2. Check Fields (Response Schema)
	var oldFields, newFields []string
	json.Unmarshal(old.Fields, &oldFields)
	json.Unmarshal(new.Fields, &newFields)

	oldFieldMap := make(map[string]bool)
	for _, f := range oldFields {
		oldFieldMap[f] = true
	}

	newFieldMap := make(map[string]bool)
	for _, f := range newFields {
		newFieldMap[f] = true
	}

	// Removed fields are breaking
	for f := range oldFieldMap {
		if !newFieldMap[f] {
			diff.Breaking = append(diff.Breaking, fmt.Sprintf("Field removed: %s", f))
		}
	}

	// Added fields are non-breaking
	for f := range newFieldMap {
		if !oldFieldMap[f] {
			diff.NonBreaking = append(diff.NonBreaking, fmt.Sprintf("Field added: %s", f))
		}
	}

	// 3. Check Filters (Request Schema)
	// Currently filters are dynamic query params, but we could check for required ones in the future
	// For now, identity of the filter set is non-breaking if added, breaking if removed

	return diff
}

// IsBreaking returns true if the diff contains any breaking changes
func (d ApiSchemaDiff) IsBreaking() bool {
	return len(d.Breaking) > 0
}
