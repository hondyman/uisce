package activities

import (
	"context"
	"fmt"

	"github.com/hondyman/uisce/backend/internal/region"
)

const errTypeRegionUnknown = "RegionUnknown"

// RegionActivities resolves a tenant's region against the boot-time registry.
// It lives in its own type so that wiring it into a worker does not change
// TenantProvisioningActivities.
type RegionActivities struct {
	Registry *region.Registry
}

// ResolveRegion returns the endpoints for a region. An unknown region, or a
// missing registry, is non-retryable: retrying cannot make a region exist.
func (a *RegionActivities) ResolveRegion(ctx context.Context, code string) (region.Endpoints, error) {
	if a.Registry == nil {
		return region.Endpoints{}, nonRetryable(errTypeRegionUnknown, fmt.Errorf("region registry is not configured"))
	}
	ep, err := a.Registry.Resolve(code)
	if err != nil {
		return region.Endpoints{}, nonRetryable(errTypeRegionUnknown, err)
	}
	return ep, nil
}
