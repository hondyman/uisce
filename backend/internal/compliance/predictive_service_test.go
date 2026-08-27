package compliance

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestPredictiveComplianceService_AutoResizeAndProbability(t *testing.T) {
	svc := NewPredictiveComplianceService(nil)
	ctx := context.Background()
	tenantID := uuid.New()
	portfolioID := uuid.New()
	ruleID := uuid.New()
	secID := uuid.New()

	// 1. Test Resizing Math Formula: DeltaQ = Floor((L * AUM - Vg) / (P * (1 - L)))
	// AUM = $10,000,000, Current Group = $400,000, Limit = 5% ($500k target max)
	// Price = $100/share, Proposed = 2,000 shares ($200k) -> Projected = $600k / $10.2M = 5.88% (Breach)
	// Analytical Max: (0.05 * 10,000,000 - 400,000) / (100 * 0.95) = 100,000 / 95 = 1052.63 -> Floor = 1052 shares
	resizing, err := svc.CalculateMaxCompliantShares(
		ctx,
		tenantID,
		portfolioID,
		secID,
		"TICKET-ORD-5501",
		2000,   // Proposed
		100.00, // Price
		5.00,   // Limit %
		400000.00,
		10000000.00,
	)

	if err != nil {
		t.Fatalf("unexpected error calculating max compliant shares: %v", err)
	}

	expectedMax := math.Floor((0.05*10000000.00 - 400000.00) / (100.00 * (1.0 - 0.05)))
	if resizing.MaxCompliantShares != expectedMax {
		t.Errorf("expected max compliant shares %.0f, got %.0f", expectedMax, resizing.MaxCompliantShares)
	}

	if resizing.ReductionRequired != (2000.0 - expectedMax) {
		t.Errorf("expected reduction delta %.0f, got %.0f", 2000.0-expectedMax, resizing.ReductionRequired)
	}

	// 2. Test Logistic Probability Monotonicity
	probLow, err := svc.ForecastBreachProbability(ctx, tenantID, portfolioID, ruleID, 0.50, 0.15, 0.02, 0)
	if err != nil {
		t.Fatalf("unexpected error forecasting low prob: %v", err)
	}

	probHigh, err := svc.ForecastBreachProbability(ctx, tenantID, portfolioID, ruleID, 0.95, 0.35, 0.08, 2)
	if err != nil {
		t.Fatalf("unexpected error forecasting high prob: %v", err)
	}

	if probHigh.BreachProbability <= probLow.BreachProbability {
		t.Errorf("expected higher breach probability for high utilization (got low=%.4f, high=%.4f)",
			probLow.BreachProbability, probHigh.BreachProbability)
	}

	// 3. Sub-5ms Latency Benchmark over 1,000 iterations
	start := time.Now()
	for i := 0; i < 1000; i++ {
		_, _ = svc.CalculateMaxCompliantShares(ctx, tenantID, portfolioID, secID, "BENCH-01", 1000, 150.0, 5.0, 300000.0, 10000000.0)
	}
	elapsed := time.Since(start)
	avgMs := float64(elapsed.Microseconds()) / 1000.0 / 1000.0

	if avgMs > 5.0 {
		t.Errorf("expected sub-5ms latency, got avg %.4f ms", avgMs)
	}
}
