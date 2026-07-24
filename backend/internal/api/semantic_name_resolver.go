package api

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"
)

// SemanticNameResolver provides deterministic mapping from semantic term names to field UUIDs.
// This ensures the same semantic term always maps to the same field ID, eliminating ambiguity.
type SemanticNameResolver struct {
	mu sync.RWMutex

	// termNameToFieldID maps semantic term name → field UUID
	termNameToFieldID map[string]string

	// fieldIDToTermNames is the reverse mapping (field UUID → all names/aliases for that field)
	fieldIDToTermNames map[string][]string

	// aliases tracks old names that map to field IDs (for backward compatibility)
	aliases map[string]string // oldName → fieldID

	// lastRefresh tracks when the cache was last loaded
	lastRefresh time.Time

	// db is the database connection
	db *sql.DB
}

// NewSemanticNameResolver creates a new resolver and pre-loads mappings
func NewSemanticNameResolver(db *sql.DB) *SemanticNameResolver {
	resolver := &SemanticNameResolver{
		termNameToFieldID:  make(map[string]string),
		fieldIDToTermNames: make(map[string][]string),
		aliases:            make(map[string]string),
		db:                 db,
	}
	// Pre-load on creation (errors are logged but not fatal)
	_ = resolver.Refresh(context.Background())
	return resolver
}

// Refresh reloads all semantic mappings from the database
func (r *SemanticNameResolver) Refresh(ctx context.Context) error {
	if r.db == nil {
		return fmt.Errorf("database connection is nil")
	}

	// Query all fields with their semantic term names
	query := `
		SELECT 
			id,
			field_name,
			COALESCE(semantic_term, field_name) as semantic_term
		FROM public.bo_fields
		WHERE tenant_id IS NOT NULL
		ORDER BY id
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query bo_fields: %w", err)
	}
	defer rows.Close()

	// Temporary maps to avoid holding lock during query
	newTermToField := make(map[string]string)
	newFieldToTerms := make(map[string][]string)

	for rows.Next() {
		var fieldID, fieldName, semanticTerm string
		if err := rows.Scan(&fieldID, &fieldName, &semanticTerm); err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}

		// Map semantic term name to field ID (semantic term is authoritative)
		newTermToField[semanticTerm] = fieldID

		// Also map field name as a fallback alias
		if fieldName != semanticTerm {
			newTermToField[fieldName] = fieldID
		}

		// Track reverse mapping
		newFieldToTerms[fieldID] = append(newFieldToTerms[fieldID], semanticTerm)
		if fieldName != semanticTerm {
			newFieldToTerms[fieldID] = append(newFieldToTerms[fieldID], fieldName)
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating rows: %w", err)
	}

	// Query for explicit aliases
	aliasQuery := `
		SELECT old_name, field_id
		FROM public.field_aliases
		WHERE is_active = true
	`

	aliasRows, err := r.db.QueryContext(ctx, aliasQuery)
	if err == nil {
		defer aliasRows.Close()
		newAliases := make(map[string]string)

		for aliasRows.Next() {
			var oldName, fieldID string
			if err := aliasRows.Scan(&oldName, &fieldID); err != nil {
				continue // Skip bad rows, doesn't block the refresh
			}
			newAliases[oldName] = fieldID
			newTermToField[oldName] = fieldID
		}

		// Lock and update maps atomically
		r.mu.Lock()
		r.termNameToFieldID = newTermToField
		r.fieldIDToTermNames = newFieldToTerms
		r.aliases = newAliases
		r.lastRefresh = time.Now()
		r.mu.Unlock()
	} else {
		// Aliases table might not exist; that's okay
		r.mu.Lock()
		r.termNameToFieldID = newTermToField
		r.fieldIDToTermNames = newFieldToTerms
		r.lastRefresh = time.Now()
		r.mu.Unlock()
	}

	return nil
}

// ResolveTermNameToFieldID looks up a field ID by semantic term name
// Returns the field UUID or an error if not found
func (r *SemanticNameResolver) ResolveTermNameToFieldID(termName string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if fieldID, exists := r.termNameToFieldID[termName]; exists {
		return fieldID, nil
	}

	return "", fmt.Errorf("semantic term '%s' not found in mappings", termName)
}

// ResolveFieldIDToTermNames looks up all names (including aliases) for a field ID
func (r *SemanticNameResolver) ResolveFieldIDToTermNames(fieldID string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if names, exists := r.fieldIDToTermNames[fieldID]; exists {
		return append([]string{}, names...) // Return copy
	}

	return []string{}
}

// ResolveIsAlias checks if a name is an old alias (backward compatibility)
func (r *SemanticNameResolver) ResolveIsAlias(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.aliases[name]
	return exists
}

// ResolveAlias gets the current field ID for an old alias name
func (r *SemanticNameResolver) ResolveAlias(oldName string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if fieldID, exists := r.aliases[oldName]; exists {
		return fieldID, nil
	}

	return "", fmt.Errorf("alias '%s' not found", oldName)
}

// GetAllMappings returns the complete name → UUID mapping (shallow copy)
func (r *SemanticNameResolver) GetAllMappings() map[string]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return shallow copy to prevent external modification
	result := make(map[string]string)
	for k, v := range r.termNameToFieldID {
		result[k] = v
	}
	return result
}

// GetCacheStats returns cache information for monitoring
func (r *SemanticNameResolver) GetCacheStats() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return map[string]interface{}{
		"term_count":    len(r.termNameToFieldID),
		"field_count":   len(r.fieldIDToTermNames),
		"alias_count":   len(r.aliases),
		"last_refresh":  r.lastRefresh,
		"cache_age_sec": time.Since(r.lastRefresh).Seconds(),
	}
}
