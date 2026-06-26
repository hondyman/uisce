package rules

import (
	"context"
	"fmt"
	"strings"
)

// InstanceProvider abstracts the fetching of business object instances
// allowing the resolver to be decoupled from the concrete service.
type InstanceProvider interface {
	GetInstanceForValidation(ctx context.Context, tenantID, instanceID string) (map[string]interface{}, error)
}

// PathResolver resolves dot-notation paths (e.g. "Manager.Location.Country")
// by traversing the relationship graph via iterative lookups.
type PathResolver struct {
	provider InstanceProvider
}

// NewPathResolver creates a new resolver
func NewPathResolver(provider InstanceProvider) *PathResolver {
	return &PathResolver{
		provider: provider,
	}
}

// ResolvePath fetches a value by traversing relationships
// startData: The initial data map of the object triggering the rule
// tenantID: Needed for fetching related instances
// path: The dot-notation path (e.g. "Manager.Location")
func (r *PathResolver) ResolvePath(ctx context.Context, tenantID string, startData map[string]interface{}, path string) (interface{}, error) {
	parts := strings.Split(path, ".")
	if len(parts) == 0 {
		return nil, nil
	}

	// 1. Initial Lookup (Local Data)
	currentData := startData
	currentVal, exists := currentData[parts[0]]

	// If single simple field
	if len(parts) == 1 {
		if exists {
			return currentVal, nil
		}
		return nil, nil // Field not found, treat as null
	}

	// 2. Traversal Loop
	// We have at least 2 parts (e.g. "Manager.Email")
	// parts[0] must be a reference (ID) to another object
	for i := 0; i < len(parts)-1; i++ {
		fieldName := parts[i]

		// Get value of the reference field
		// We expect this to be a UUID string pointing to another instance
		refID, ok := currentData[fieldName].(string)
		if !ok || refID == "" {
			// If field is missing or not a string ID, we can't traverse further
			// Retuning nil effectively means "null" which might fail the rule or pass "isEmpty"
			return nil, nil
		}

		// Fetch the referenced instance
		// This uses the "Lookup Related Value" pattern
		nextInstance, err := r.provider.GetInstanceForValidation(ctx, tenantID, refID)
		if err != nil {
			return nil, fmt.Errorf("failed to traverse relationship '%s' (id: %s): %w", fieldName, refID, err)
		}

		// Move pointer forward
		currentData = nextInstance
	}

	// 3. Final Value Lookup
	lastField := parts[len(parts)-1]
	if val, ok := currentData[lastField]; ok {
		return val, nil
	}

	return nil, nil
}
