package region

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/tenant/goldcopy"
)

// TenantRegionResolver provides region lookup and authorization for tenants
type TenantRegionResolver struct {
	db              *sql.DB
	goldcopyResolve *goldcopy.Resolver
}

// NewTenantRegionResolver creates a new tenant region resolver.
// The goldcopyResolve parameter is optional; if provided the resolver will
// use the shared goldcopy.Resolver to check whether a tenant is the gold copy.
// If nil, the resolver falls back to querying the DB directly.
func NewTenantRegionResolver(db *sql.DB, goldcopyResolve *goldcopy.Resolver) *TenantRegionResolver {
	return &TenantRegionResolver{
		db:              db,
		goldcopyResolve: goldcopyResolve,
	}
}

// InferRegionForTenant returns the home region for a given tenant
// Returns (region, true) if tenant exists and has a region configured
// Returns ("", false) if tenant doesn't exist or has no region
//
// This is pure lookup — no authorization logic.
func (r *TenantRegionResolver) InferRegionForTenant(tenantID string) (string, bool) {
	if r.isGoldCopyTenant(tenantID) {
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
	if r.isGoldCopyTenant(tenantID) {
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
	if r.isGoldCopyTenant(tenantID) {
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

// isGoldCopyTenant returns true if the given tenant ID matches the gold copy tenant.
// It uses the injected goldcopy.Resolver when available, or falls back to a direct
// DB query when r.goldcopyResolve is nil.
func (r *TenantRegionResolver) isGoldCopyTenant(tenantID string) bool {
	if tenantID == "" {
		return false
	}
	id, err := uuid.Parse(tenantID)
	if err != nil {
		return false
	}
	if r.goldcopyResolve != nil {
		isGold, _ := r.goldcopyResolve.IsGoldCopy(id)
		return isGold
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM public.tenants WHERE id = $1 AND gold_copy = true)`
	_ = r.db.QueryRowContext(ctx, query, tenantID).Scan(&exists)
	return exists
}
