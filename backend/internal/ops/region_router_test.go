package ops

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// ============================================================================
// Phase 3.6/3.7: Region Router Tests
// ============================================================================

func TestRegionRouterInitialization(t *testing.T) {
	store := &TestStore{}
	router := NewInMemoryRegionRegistry(store, 5*time.Minute)
	ctx := context.Background()

	err := router.Initialize(ctx)
	assert.NoError(t, err)

	// Verify default regions are registered
	targets, err := router.ListRegionTargets(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, targets)
}

func TestTenantRegionRouting(t *testing.T) {
	store := &TestStore{}
	router := NewInMemoryRegionRegistry(store, 5*time.Minute)
	ctx := context.Background()

	router.Initialize(ctx)

	// Register test regions
	router.RegisterRegionTarget(ctx, &RegionTarget{
		Region:            "us-east-1",
		StarRocksCluster:  "starrocks-us-east-1",
		RedpandaBroker:    "redpanda-us-east-1:9092",
		TemporalNamespace: "us-east-1",
		OpsWorkerPool:     "worker-pool-us-east-1",
		IsActive:          true,
	})
	router.RegisterRegionTarget(ctx, &RegionTarget{
		Region:            "eu-west-1",
		StarRocksCluster:  "starrocks-eu-west-1",
		RedpandaBroker:    "redpanda-eu-west-1:9092",
		TemporalNamespace: "eu-west-1",
		OpsWorkerPool:     "worker-pool-eu-west-1",
		IsActive:          true,
	})

	tenantID := uuid.New()

	// Set tenant to eu-west-1
	err := router.SetTenantRegion(ctx, tenantID, "eu-west-1", []string{"eu-west-1", "us-east-1"})
	assert.NoError(t, err)

	// Get tenant's region
	region, err := router.GetTenantRegion(ctx, tenantID)
	assert.NoError(t, err)
	assert.Equal(t, "eu-west-1", region)

	// Get allowed regions
	allowedRegions, err := router.GetTenantAllowedRegions(ctx, tenantID)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(allowedRegions))
}

func TestDefaultTenantRegion(t *testing.T) {
	store := &TestStore{}
	router := NewInMemoryRegionRegistry(store, 5*time.Minute)
	ctx := context.Background()

	router.Initialize(ctx)

	unknownTenant := uuid.New()

	// Unknown tenant should default to us-east-1
	region, err := router.GetTenantRegion(ctx, unknownTenant)
	assert.NoError(t, err)
	assert.Equal(t, "us-east-1", region)
}

func TestRegionTargetResolution(t *testing.T) {
	store := &TestStore{}
	router := NewInMemoryRegionRegistry(store, 5*time.Minute)
	ctx := context.Background()

	router.Initialize(ctx)

	// Register a custom region target
	target := &RegionTarget{
		Region:            "ap-southeast-1",
		StarRocksCluster:  "starrocks-ap-1",
		RedpandaBroker:    "redpanda-ap-1:9092",
		TemporalNamespace: "ap-southeast-1",
		OpsWorkerPool:     "worker-pool-ap-1",
		IsActive:          true,
	}

	err := router.RegisterRegionTarget(ctx, target)
	assert.NoError(t, err)

	// Resolve the target
	resolved, err := router.GetRegionTarget(ctx, "ap-southeast-1")
	assert.NoError(t, err)
	assert.Equal(t, "starrocks-ap-1", resolved.StarRocksCluster)
	assert.Equal(t, "redpanda-ap-1:9092", resolved.RedpandaBroker)
}

func TestRouteForTenant(t *testing.T) {
	store := &TestStore{}
	router := NewInMemoryRegionRegistry(store, 5*time.Minute)
	ctx := context.Background()

	router.Initialize(ctx)

	// Register us-east-1
	router.RegisterRegionTarget(ctx, &RegionTarget{
		Region:            "us-east-1",
		StarRocksCluster:  "starrocks-us-east-1",
		RedpandaBroker:    "redpanda-us-east-1:9092",
		TemporalNamespace: "us-east-1",
		OpsWorkerPool:     "worker-pool-us-east-1",
		IsActive:          true,
	})

	tenantID := uuid.New()
	err := router.SetTenantRegion(ctx, tenantID, "us-east-1", []string{"us-east-1"})
	assert.NoError(t, err)

	// Route for tenant
	target, err := router.RouteForTenant(ctx, tenantID)
	assert.NoError(t, err)
	assert.Equal(t, "us-east-1", target.Region)
}

func TestRouteForIncident(t *testing.T) {
	store := &TestStore{}
	router := NewInMemoryRegionRegistry(store, 5*time.Minute)
	ctx := context.Background()

	router.Initialize(ctx)

	// Register eu-west-1
	router.RegisterRegionTarget(ctx, &RegionTarget{
		Region:            "eu-west-1",
		StarRocksCluster:  "starrocks-eu-west-1",
		RedpandaBroker:    "redpanda-eu-west-1:9092",
		TemporalNamespace: "eu-west-1",
		OpsWorkerPool:     "worker-pool-eu-west-1",
		IsActive:          true,
	})

	region := "eu-west-1"
	incident := &Incident{
		ID:        uuid.New(),
		Region:    &region,
		Status:    "open",
		Severity:  SeverityWarning,
		Title:     "Test",
		StartedAt: time.Now(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	target, err := router.RouteForIncident(ctx, incident)
	assert.NoError(t, err)
	assert.Equal(t, "eu-west-1", target.Region)
}

func TestRouteForEvent(t *testing.T) {
	store := &TestStore{}
	router := NewInMemoryRegionRegistry(store, 5*time.Minute)
	ctx := context.Background()

	router.Initialize(ctx)

	// Register ap-southeast-1
	router.RegisterRegionTarget(ctx, &RegionTarget{
		Region:            "ap-southeast-1",
		StarRocksCluster:  "starrocks-ap-southeast-1",
		RedpandaBroker:    "redpanda-ap-southeast-1:9092",
		TemporalNamespace: "ap-southeast-1",
		OpsWorkerPool:     "worker-pool-ap-southeast-1",
		IsActive:          true,
	})

	region := "ap-southeast-1"
	event := Event{
		ID:         uuid.New(),
		EventType:  EventLatencyAnomaly,
		Severity:   SeverityWarning,
		Region:     &region,
		Scope:      "region",
		Title:      "Test",
		Details:    []byte(`{}`),
		OccurredAt: time.Now(),
	}

	target, err := router.RouteForEvent(ctx, &event)
	assert.NoError(t, err)
	assert.Equal(t, "ap-southeast-1", target.Region)
}

func TestFailoverRouting(t *testing.T) {
	store := &TestStore{}
	router := NewInMemoryRegionRegistry(store, 5*time.Minute)
	ctx := context.Background()

	router.Initialize(ctx)

	// Set up regions with failover
	primaryRegion := "us-east-1"
	secondaryRegion := "us-west-2"

	primary := &RegionTarget{
		Region:            primaryRegion,
		StarRocksCluster:  "starrocks-us-east-1",
		RedpandaBroker:    "redpanda-us-east-1:9092",
		TemporalNamespace: "us-east-1",
		OpsWorkerPool:     "worker-pool-us-east-1",
		FailoverTarget:    &secondaryRegion,
		IsActive:          true,
	}

	secondary := &RegionTarget{
		Region:            secondaryRegion,
		StarRocksCluster:  "starrocks-us-west-2",
		RedpandaBroker:    "redpanda-us-west-2:9092",
		TemporalNamespace: "us-west-2",
		OpsWorkerPool:     "worker-pool-us-west-2",
		IsActive:          true,
	}

	router.RegisterRegionTarget(ctx, primary)
	router.RegisterRegionTarget(ctx, secondary)

	// Mark primary as down
	err := router.MarkRegionDown(ctx, primaryRegion)
	assert.NoError(t, err)

	// Route to primary should failover to secondary
	target, err := router.RouteToRegion(ctx, primaryRegion)
	assert.NoError(t, err)
	assert.Equal(t, secondaryRegion, target.Region)
}

func TestRegionDownMarking(t *testing.T) {
	store := &TestStore{}
	router := NewInMemoryRegionRegistry(store, 5*time.Minute)
	ctx := context.Background()

	router.Initialize(ctx)

	region := "us-east-1"

	// Register us-east-1
	router.RegisterRegionTarget(ctx, &RegionTarget{
		Region:            "us-east-1",
		StarRocksCluster:  "starrocks-us-east-1",
		RedpandaBroker:    "redpanda-us-east-1:9092",
		TemporalNamespace: "us-east-1",
		OpsWorkerPool:     "worker-pool-us-east-1",
		IsActive:          true,
	})

	// Initially region should be up
	assert.False(t, router.IsRegionDown(region))

	// Mark as down
	err := router.MarkRegionDown(ctx, region)
	assert.NoError(t, err)
	assert.True(t, router.IsRegionDown(region))

	// Mark as up
	err = router.MarkRegionUp(ctx, region)
	assert.NoError(t, err)
	assert.False(t, router.IsRegionDown(region))
}

func TestMultiTenantRouting(t *testing.T) {
	// Test that different tenants can route to different regions
	store := &TestStore{}
	router := NewInMemoryRegionRegistry(store, 5*time.Minute)
	ctx := context.Background()

	router.Initialize(ctx)

	// Register multiple regions
	for _, region := range []string{"us-east-1", "eu-west-1", "ap-southeast-1"} {
		router.RegisterRegionTarget(ctx, &RegionTarget{
			Region:            region,
			StarRocksCluster:  "starrocks-" + region,
			RedpandaBroker:    "redpanda-" + region + ":9092",
			TemporalNamespace: region,
			OpsWorkerPool:     "worker-pool-" + region,
			IsActive:          true,
		})
	}

	tenant1 := uuid.New()
	tenant2 := uuid.New()

	// Tenant 1 -> eu-west-1
	err := router.SetTenantRegion(ctx, tenant1, "eu-west-1", []string{"eu-west-1"})
	assert.NoError(t, err)

	// Tenant 2 -> ap-southeast-1
	err = router.SetTenantRegion(ctx, tenant2, "ap-southeast-1", []string{"ap-southeast-1"})
	assert.NoError(t, err)

	// Verify separate routing
	region1, _ := router.GetTenantRegion(ctx, tenant1)
	region2, _ := router.GetTenantRegion(ctx, tenant2)

	assert.Equal(t, "eu-west-1", region1)
	assert.Equal(t, "ap-southeast-1", region2)
}

func TestRouteConsistency(t *testing.T) {
	// Same incident should always route to same region
	store := &TestStore{}
	router := NewInMemoryRegionRegistry(store, 5*time.Minute)
	ctx := context.Background()

	router.Initialize(ctx)

	region := "us-east-1"

	// Register us-east-1
	router.RegisterRegionTarget(ctx, &RegionTarget{
		Region:            "us-east-1",
		StarRocksCluster:  "starrocks-us-east-1",
		RedpandaBroker:    "redpanda-us-east-1:9092",
		TemporalNamespace: "us-east-1",
		OpsWorkerPool:     "worker-pool-us-east-1",
		IsActive:          true,
	})

	incident := &Incident{
		ID:        uuid.New(),
		Region:    &region,
		Status:    "open",
		Severity:  SeverityWarning,
		Title:     "Test",
		StartedAt: time.Now(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Route multiple times
	target1, _ := router.RouteForIncident(ctx, incident)
	target2, _ := router.RouteForIncident(ctx, incident)
	target3, _ := router.RouteForIncident(ctx, incident)

	assert.Equal(t, target1.Region, target2.Region)
	assert.Equal(t, target2.Region, target3.Region)
}

func TestInvalidRegionRouting(t *testing.T) {
	store := &TestStore{}
	router := NewInMemoryRegionRegistry(store, 5*time.Minute)
	ctx := context.Background()

	router.Initialize(ctx)

	// Try to route to non-existent region
	_, err := router.RouteToRegion(ctx, "non-existent-region")
	assert.Error(t, err)

	// Try to set tenant to non-existent region
	tenantID := uuid.New()
	err = router.SetTenantRegion(ctx, tenantID, "invalid-region", []string{})
	assert.Error(t, err)
}
func TestRegionTargetFallback(t *testing.T) {
	// Events/incidents without region should default to us-east-1
	store := &TestStore{}
	router := NewInMemoryRegionRegistry(store, 5*time.Minute)
	ctx := context.Background()

	router.Initialize(ctx)

	// Register us-east-1
	router.RegisterRegionTarget(ctx, &RegionTarget{
		Region:            "us-east-1",
		StarRocksCluster:  "starrocks-us-east-1",
		RedpandaBroker:    "redpanda-us-east-1:9092",
		TemporalNamespace: "us-east-1",
		OpsWorkerPool:     "worker-pool-us-east-1",
		IsActive:          true,
	})

	// Event without region
	event := Event{
		ID:         uuid.New(),
		EventType:  EventLatencyAnomaly,
		Severity:   SeverityWarning,
		Region:     nil,
		Scope:      "global",
		Title:      "Test",
		Details:    []byte(`{}`),
		OccurredAt: time.Now(),
	}

	target, err := router.RouteForEvent(ctx, &event)
	assert.NoError(t, err)
	assert.Equal(t, "us-east-1", target.Region)

	// Incident without region
	incident := &Incident{
		ID:        uuid.New(),
		Region:    nil,
		Status:    "open",
		Severity:  SeverityWarning,
		Title:     "Test",
		StartedAt: time.Now(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	target, err = router.RouteForIncident(ctx, incident)
	assert.NoError(t, err)
	assert.Equal(t, "us-east-1", target.Region)
}
