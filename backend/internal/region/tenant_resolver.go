package region

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
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
	// public.tenants has no home_region/metadata columns (that was a stale
	// assumption from an earlier schema draft); default_region/region are
	// the real columns.
	query := `
		SELECT COALESCE(default_region, region)
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

	// Query tenant's allowed regions. allowed_regions is a JSONB array
	// (e.g. ["us-west"]); default_region/region are plain strings.
	var allowedRegionsJSON sql.NullString
	var defaultRegion, homeRegion sql.NullString
	query := `
		SELECT allowed_regions::text, default_region, region
		FROM public.tenants
		WHERE id = $1
		LIMIT 1
	`

	err := r.db.QueryRow(query, tenantID).Scan(&allowedRegionsJSON, &defaultRegion, &homeRegion)
	if err != nil {
		if err == sql.ErrNoRows {
			return false
		}
		// Log the error but don't expose it
		fmt.Printf("[TenantRegionResolver] Error querying tenant regions: %v\n", err)
		return false
	}

	if allowedRegionsJSON.Valid && allowedRegionsJSON.String != "" {
		var allowed []string
		if err := json.Unmarshal([]byte(allowedRegionsJSON.String), &allowed); err == nil {
			for _, a := range allowed {
				if strings.EqualFold(strings.TrimSpace(a), region) {
					return true
				}
			}
		}
	}

	if defaultRegion.Valid && strings.EqualFold(defaultRegion.String, region) {
		return true
	}
	if homeRegion.Valid && strings.EqualFold(homeRegion.String, region) {
		return true
	}

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

	var allowedRegionsJSON sql.NullString
	var defaultRegion, homeRegion sql.NullString
	query := `
		SELECT allowed_regions::text, default_region, region
		FROM public.tenants
		WHERE id = $1
		LIMIT 1
	`

	err := r.db.QueryRow(query, tenantID).Scan(&allowedRegionsJSON, &defaultRegion, &homeRegion)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("tenant not found: %s", tenantID)
		}
		return nil, fmt.Errorf("failed to query tenant regions: %w", err)
	}

	if allowedRegionsJSON.Valid && allowedRegionsJSON.String != "" {
		var allowed []string
		if err := json.Unmarshal([]byte(allowedRegionsJSON.String), &allowed); err == nil && len(allowed) > 0 {
			return allowed, nil
		}
	}

	if defaultRegion.Valid && defaultRegion.String != "" {
		return []string{defaultRegion.String}, nil
	}
	if homeRegion.Valid && homeRegion.String != "" {
		return []string{homeRegion.String}, nil
	}

	return []string{}, nil
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
