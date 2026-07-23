package region

import (
	"database/sql"
	"fmt"
)

// TenantRegionResolver provides region lookup and authorization for tenants
type TenantRegionResolver struct {
	db *sql.DB
}

// NewTenantRegionResolver creates a new tenant region resolver
func NewTenantRegionResolver(db *sql.DB) *TenantRegionResolver {
	return &TenantRegionResolver{db: db}
}

// GoldCopyTenantID is the special tenant ID for globally inherited rules/terms
const GoldCopyTenantID = "99e99e99-99e9-49e9-89e9-99e99e99e999"

// InferRegionForTenant returns the home region for a given tenant
// Returns (region, true) if tenant exists and has a region configured
// Returns ("", false) if tenant doesn't exist or has no region
//
// This is pure lookup — no authorization logic.
func (r *TenantRegionResolver) InferRegionForTenant(tenantID string) (string, bool) {
	if tenantID == GoldCopyTenantID {
		// Gold Copy has no specific region — it's region-agnostic
		return "", false
	}

	if tenantID == "" {
		return "", false
	}

	var region sql.NullString
	query := `
		SELECT COALESCE(home_region, metadata->>'region')
		FROM public.tenants
		WHERE id = $1
		LIMIT 1
	`

	err := r.db.QueryRow(query, tenantID).Scan(&region)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", false
		}
		// Log the error but don't expose it
		fmt.Printf("[TenantRegionResolver] Error querying tenant region: %v\n", err)
		return "", false
	}

	if !region.Valid || region.String == "" {
		return "", false
	}

	return region.String, true
}

// IsRegionAllowedForTenant checks if a tenant is allowed to operate in a specific region
// Returns true if:
//   - tenant is Gold Copy (always allowed)
//   - region matches tenant's home region
//   - region is in tenant's allowed_regions list
//
// This is pure authorization — no lookup beyond necessary validation.
func (r *TenantRegionResolver) IsRegionAllowedForTenant(tenantID, region string) bool {
	if tenantID == "" || region == "" {
		return false
	}

	// Gold Copy bypass — always allowed in any region
	if tenantID == GoldCopyTenantID {
		return true
	}

	// Query tenant's allowed regions
	var allowedRegions sql.NullString
	query := `
		SELECT COALESCE(allowed_regions::text, 
		        metadata->>'allowed_regions',
		        home_region,
		        metadata->>'region')
		FROM public.tenants
		WHERE id = $1
		LIMIT 1
	`

	err := r.db.QueryRow(query, tenantID).Scan(&allowedRegions)
	if err != nil {
		if err == sql.ErrNoRows {
			return false
		}
		// Log the error but don't expose it
		fmt.Printf("[TenantRegionResolver] Error querying tenant regions: %v\n", err)
		return false
	}

	if !allowedRegions.Valid || allowedRegions.String == "" {
		return false
	}

	// Parse allowed regions (could be JSON array or comma-separated)
	// For now, simple exact match with the home region
	homeRegion := allowedRegions.String
	if homeRegion == region {
		return true
	}

	// TODO: In future, parse allowed_regions JSONB array for multi-region tenants:
	// var allowed []string
	// if strings.HasPrefix(homeRegion, "[") {
	//     json.Unmarshal([]byte(homeRegion), &allowed)
	//     for _, r := range allowed {
	//         if r == region { return true }
	//     }
	// }

	return false
}

// GetAllowedRegions returns all regions this tenant is allowed to access
// Returns a slice of region codes (e.g., ["us-east-1", "us-west-2"])
func (r *TenantRegionResolver) GetAllowedRegions(tenantID string) ([]string, error) {
	if tenantID == GoldCopyTenantID {
		// Gold Copy works in all regions
		return []string{}, nil
	}

	if tenantID == "" {
		return nil, fmt.Errorf("tenant_id cannot be empty")
	}

	var homeRegion sql.NullString
	query := `
		SELECT COALESCE(home_region, metadata->>'region')
		FROM public.tenants
		WHERE id = $1
		LIMIT 1
	`

	err := r.db.QueryRow(query, tenantID).Scan(&homeRegion)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("tenant not found: %s", tenantID)
		}
		return nil, fmt.Errorf("failed to query tenant regions: %w", err)
	}

	if !homeRegion.Valid || homeRegion.String == "" {
		return []string{}, nil
	}

	// Return single home region (multi-region support coming soon)
	return []string{homeRegion.String}, nil
}
