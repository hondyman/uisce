package scoring

import (
	"context"
	"testing"
	"time"
)

func TestPostgresRepository_SyncMultiDimensionalScorecardToStarRocks_NilDB(t *testing.T) {
	repo := NewPostgresRepository(nil)
	err := repo.SyncMultiDimensionalScorecardToStarRocks(
		context.Background(),
		time.Now(),
		"99e99e99-99e9-49e9-89e9-99e99e99e999",
		1,
		"pricing",
		[]VendorDimensionProfile{
			{
				VendorID: "BBG",
				Components: QualityComponents{
					Sufficiency: 0.95,
					Coverage:    0.98,
					SLA:         0.99,
					Stability:   0.97,
					Friction:    0.90,
					Licensing:   0.85,
				},
				CompositeQuality:    0.94,
				AnnualSpend:         2140000,
				CostPerQualityPoint: 22765.95,
			},
		},
	)
	if err != nil {
		t.Fatalf("expected nil error when starrocksDB is nil, got: %v", err)
	}
}

func TestService_SyncMultiDimensionalMart(t *testing.T) {
	repo := NewPostgresRepository(nil)
	svc := NewService(repo)

	profiles, err := svc.SyncMultiDimensionalMart(
		context.Background(),
		time.Now(),
		"99e99e99-99e9-49e9-89e9-99e99e99e999",
		1,
		"pricing",
	)
	if err != nil {
		t.Fatalf("expected successful sync, got: %v", err)
	}
	if len(profiles) == 0 {
		t.Fatalf("expected non-empty profiles returned from SyncMultiDimensionalMart")
	}
}

func TestMonitor_CheckHealth(t *testing.T) {
	mon := NewMonitor(nil, nil)
	health := mon.CheckHealth(context.Background())
	if health.Status != "HEALTHY" {
		t.Fatalf("expected HEALTHY default status, got: %s", health.Status)
	}

	mon.RecordPartialSolver()
	health2 := mon.CheckHealth(context.Background())
	if health2.PartialSolverCount != 1 {
		t.Fatalf("expected partial count 1, got: %d", health2.PartialSolverCount)
	}
	if health2.Status != "DEGRADED" {
		t.Fatalf("expected DEGRADED after solver partial fallback, got: %s", health2.Status)
	}
}

