package ops

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// RegionRegistry provides region-aware service discovery and routing
type RegionRegistry struct {
	store Store
}

// NewRegionRegistry creates a new region registry
func NewRegionRegistry(store Store) *RegionRegistry {
	return &RegionRegistry{store: store}
}

// GetRegionServiceEndpoints retrieves all service endpoints for a specific region
func (r *RegionRegistry) GetRegionServiceEndpoints(ctx context.Context, region string) (*RegionConfig, error) {
	return r.store.GetRegionConfig(ctx, region)
}

// GetTenantRegionRouting retrieves routing configuration for a tenant in a specific region
func (r *RegionRegistry) GetTenantRegionRouting(ctx context.Context, tenantID uuid.UUID, region string) (*RegionRouting, error) {
	// First verify region exists
	regionConfig, err := r.store.GetRegionConfig(ctx, region)
	if err != nil {
		return nil, fmt.Errorf("region lookup failed: %w", err)
	}
	if regionConfig == nil {
		return nil, fmt.Errorf("region %s not found", region)
	}

	// Then get tenant-specific routing
	routing, err := r.store.GetRegionRouting(ctx, tenantID, region)
	if err != nil {
		return nil, fmt.Errorf("region routing lookup failed: %w", err)
	}

	// Return routing even if nil (allows default behavior)
	return routing, nil
}

// ListTenantRegions retrieves all regions a tenant operates in
func (r *RegionRegistry) ListTenantRegions(ctx context.Context, tenantID uuid.UUID) ([]RegionRouting, error) {
	return r.store.ListRegionRoutings(ctx, tenantID)
}

// ListActiveRegions retrieves all active region configurations
func (r *RegionRegistry) ListActiveRegions(ctx context.Context) ([]RegionConfig, error) {
	return r.store.ListRegionConfigs(ctx, true)
}

// ValidateRegion checks if a region is valid and active
func (r *RegionRegistry) ValidateRegion(ctx context.Context, region string) (bool, error) {
	config, err := r.store.GetRegionConfig(ctx, region)
	if err != nil {
		return false, err
	}
	if config == nil {
		return false, nil
	}
	return config.IsActive, nil
}

// ====== Region Validation Helpers ======

// IsValidRegion checks if a region string is valid
func IsValidRegion(region string) bool {
	if region == "" {
		return false
	}
	// Region should be alphanumeric with hyphens (e.g., us-east-1, eu-west-1)
	for _, c := range region {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
			return false
		}
	}
	return true
}

// GetDefaultRegion returns a default region (currently us-east-1)
func GetDefaultRegion() string {
	return "us-east-1"
}

// NormalizeRegion returns the region or default if empty
func NormalizeRegion(region *string) string {
	if region == nil || *region == "" {
		return GetDefaultRegion()
	}
	return *region
}

// ValidateRegionList checks if all regions in a list are valid
func ValidateRegionList(regions []string) error {
	if len(regions) == 0 {
		return fmt.Errorf("at least one region is required")
	}

	seen := make(map[string]bool)
	for _, r := range regions {
		if !IsValidRegion(r) {
			return fmt.Errorf("invalid region format: %s", r)
		}
		if seen[r] {
			return fmt.Errorf("duplicate region: %s", r)
		}
		seen[r] = true
	}

	return nil
}

// ExtractRegionFromContext extracts region from request context
// This would be used in middleware to populate region from headers or path
func ExtractRegionFromContext(ctx context.Context) *string {
	// Check for region in context value
	if region, ok := ctx.Value("region").(string); ok && region != "" {
		return &region
	}
	return nil
}

// ====== RCA Region Scoring ======

// RegionAffinity represents how strongly events are related by region
type RegionAffinity string

const (
	RegionAffinitySameRegion     RegionAffinity = "same_region"     // Same geographic region (weight: 1.0)
	RegionAffinityAdjacentRegion RegionAffinity = "adjacent_region" // Adjacent region (weight: 0.7)
	RegionAffinityCrossRegion    RegionAffinity = "cross_region"    // Different regions (weight: 0.3)
)

// GetRegionAffinityScore returns correlation weight based on region relationship
func GetRegionAffinityScore(region1, region2 *string) float64 {
	// Normalize nil regions to default
	r1 := NormalizeRegion(region1)
	r2 := NormalizeRegion(region2)

	// Determine affinity
	affinity := GetRegionAffinity(r1, r2)
	switch affinity {
	case RegionAffinitySameRegion:
		return 1.0 // Same region: full weight
	case RegionAffinityAdjacentRegion:
		return 0.7 // Adjacent region: 70% weight
	case RegionAffinityCrossRegion:
		return 0.3 // Cross region: 30% weight
	default:
		return 0.3
	}
}

// GetRegionAffinity determines the relationship between two regions
func GetRegionAffinity(region1, region2 string) RegionAffinity {
	if region1 == region2 {
		return RegionAffinitySameRegion
	}

	// Define adjacencies (can be extended)
	adjacencies := map[string][]string{
		"us-east-1":      {"us-east-2"},
		"us-east-2":      {"us-east-1"},
		"us-west-1":      {"us-west-2"},
		"us-west-2":      {"us-west-1"},
		"eu-west-1":      {"eu-central-1", "eu-west-2"},
		"eu-central-1":   {"eu-west-1"},
		"eu-west-2":      {"eu-west-1"},
		"ap-southeast-1": {"ap-southeast-2"},
		"ap-southeast-2": {"ap-southeast-1"},
	}

	if neighbors, ok := adjacencies[region1]; ok {
		for _, n := range neighbors {
			if n == region2 {
				return RegionAffinityAdjacentRegion
			}
		}
	}

	return RegionAffinityCrossRegion
}
