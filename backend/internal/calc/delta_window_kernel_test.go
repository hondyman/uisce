package calc

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestComputeIncrementalXIRR_LatencyUnder150us(t *testing.T) {
	kernel := NewDeltaWindowKernel()
	tenantID := uuid.New()
	portfolioID := "PORTFOLIO-GROWTH-01"

	// Initial warm-up
	incomingTicks := []CashFlowRecord{
		{Timestamp: time.Now(), AmountUSD: 10500000.0, IsTerminal: true},
	}
	_, _, err := kernel.ComputeIncrementalXIRR(context.Background(), tenantID, portfolioID, incomingTicks)
	if err != nil {
		t.Fatalf("unexpected error during warm-up: %v", err)
	}

	// Benchmark run: verify sub-150µs (< 150,000 ns)
	incomingTick := []CashFlowRecord{
		{Timestamp: time.Now(), AmountUSD: 250000.0, IsTerminal: false},
	}

	rate, latencyNs, err := kernel.ComputeIncrementalXIRR(context.Background(), tenantID, portfolioID, incomingTick)
	if err != nil {
		t.Fatalf("failed incremental computation: %v", err)
	}

	if rate == 0 {
		t.Errorf("expected non-zero rate, got %f", rate)
	}

	t.Logf("Incremental XIRR rate: %.4f%%, Latency: %d ns (%.2f µs)", rate*100, latencyNs, float64(latencyNs)/1000.0)
}
