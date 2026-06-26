package nl_intelligence

import (
	"fmt"
	"strings"
)

// MultiTenantSafety ensures queries are scoped to the correct tenants
type MultiTenantSafety struct {
	// Logic to validate or rewrite queries
}

// ValidateSQL checks if the generated SQL contains tenant filters
func (s *MultiTenantSafety) ValidateSQL(sql string, tenantScope []string) error {
	if len(tenantScope) == 0 {
		return nil // Global scope
	}

	// Simple check: does it contain tenant_id?
	// In production, use an SQL parser (e.g., pg_query_go or trino parser)
	lowerSQL := strings.ToLower(sql)
	if !strings.Contains(lowerSQL, "tenant_id") {
		return fmt.Errorf("generated SQL missing tenant_id filter")
	}

	return nil
}

// ValidateCypher checks if age queries respect tenant boundaries
func (s *MultiTenantSafety) ValidateCypher(cypher string, tenantScope []string) error {
	if len(tenantScope) == 0 {
		return nil
	}

	lowerCypher := strings.ToLower(cypher)
	if !strings.Contains(lowerCypher, "tenant") {
		return fmt.Errorf("generated Cypher missing tenant filter")
	}

	return nil
}

// EnforceTenantContext ensures the request context doesn't exceed authorized scope
func (s *MultiTenantSafety) EnforceTenantContext(authorized []string, requested []string) ([]string, error) {
	if len(authorized) == 0 {
		return requested, nil // Global admin
	}

	authMap := make(map[string]bool)
	for _, t := range authorized {
		authMap[t] = true
	}

	var allowed []string
	for _, r := range requested {
		if authMap[r] {
			allowed = append(allowed, r)
		} else {
			return nil, fmt.Errorf("unauthorized access to tenant: %s", r)
		}
	}

	if len(requested) == 0 {
		return authorized, nil
	}

	return allowed, nil
}
