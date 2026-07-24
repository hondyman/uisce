package api

import (
	"fmt"
)

// ValidateSemanticQuery checks that a semantic query conforms to the bundle.
// It ensures all referenced fields exist and basic constraints are met.
func (srv *Server) ValidateSemanticQuery(
	bundle *SemanticBundle,
	q *SemanticQuery,
) error {
	// Validate datasource match
	if q.Datasource != bundle.BusinessObjectName {
		return fmt.Errorf("datasource mismatch: expected %s, got %s", bundle.BusinessObjectName, q.Datasource)
	}

	// Build field name lookup map for O(1) validation
	fieldsByName := make(map[string]*SemanticField)
	for i := range bundle.Fields {
		fm := &bundle.Fields[i]
		fieldsByName[fm.Name] = fm
	}

	// Validate select fields
	if len(q.Select) == 0 {
		return fmt.Errorf("select list cannot be empty")
	}

	for _, fieldName := range q.Select {
		if _, exists := fieldsByName[fieldName]; !exists {
			return fmt.Errorf("unknown select field: %s", fieldName)
		}
	}

	// Validate filter fields
	for _, filter := range q.Filters {
		if _, exists := fieldsByName[filter.Field]; !exists {
			return fmt.Errorf("unknown filter field: %s", filter.Field)
		}

		// Validate operator
		validOps := map[string]bool{
			"=": true, "!=": true, "<>": true,
			">": true, ">=": true, "<": true, "<=": true,
			"IN": true, "NOT IN": true,
			"LIKE": true, "ILIKE": true,
			"IS NULL": true, "IS NOT NULL": true,
		}
		if !validOps[filter.Op] {
			return fmt.Errorf("invalid filter operator: %s", filter.Op)
		}
	}

	// Validate order_by fields
	for _, ob := range q.OrderBy {
		if _, exists := fieldsByName[ob.Field]; !exists {
			return fmt.Errorf("unknown order_by field: %s", ob.Field)
		}

		// Validate direction
		if ob.Direction != "asc" && ob.Direction != "desc" {
			return fmt.Errorf("invalid order_by direction: %s (must be asc or desc)", ob.Direction)
		}
	}

	// Sanitize limit
	if q.Limit <= 0 {
		q.Limit = 100 // Default limit
	}
	if q.Limit > 100000 {
		q.Limit = 100000 // Cap at 100k rows
	}

	return nil
}
