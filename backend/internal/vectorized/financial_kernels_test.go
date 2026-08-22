package vectorized

import (
	"math"
	"testing"
	"time"
)

func TestVectorizedFinancialKernels_CalculateXIRR(t *testing.T) {
	kernels := NewVectorizedFinancialKernels()

	// Scenario: Standard institutional cashflow profile
	// 2024-01-01: -1,000,000 (Initial Investment)
	// 2024-07-01:   -200,000 (Capital Call)
	// 2025-01-01:    400,000 (Distribution)
	// 2026-01-01:  1,100,000 (Terminal Value)
	t0 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	t1 := time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC).Unix()
	t2 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	t3 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Unix()

	dates := []int64{t0, t1, t2, t3}
	amounts := []float64{-1000000.0, -200000.0, 400000.0, 1100000.0}

	irr, err := kernels.CalculateXIRR(dates, amounts, 100, 1e-7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Expected annualized IRR ~ 14.57%
	if math.Abs(irr-0.1457) > 0.005 {
		t.Errorf("expected IRR around 0.1457, got: %.4f", irr)
	}
}

func TestVectorizedFinancialKernels_ModifiedDuration(t *testing.T) {
	kernels := NewVectorizedFinancialKernels()

	// 5-year annual bond with 5% coupon, 5% YTM
	t0 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	dates := []int64{
		t0,
		time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
		time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
		time.Date(2028, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
		time.Date(2029, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
	}
	amounts := []float64{0.0, 50.0, 50.0, 50.0, 50.0, 1050.0}

	dur := kernels.CalculateModifiedDuration(dates, amounts, 0.05)
	if math.Abs(dur-4.329) > 0.01 {
		t.Errorf("expected modified duration ~ 4.329, got: %.4f", dur)
	}
}

func TestVectorizedFinancialKernels_SharpeRatio(t *testing.T) {
	kernels := NewVectorizedFinancialKernels()

	// Monthly returns: 1%, 2%, -1%, 1.5%, 0.5%, 2.5%
	returns := []float64{0.01, 0.02, -0.01, 0.015, 0.005, 0.025}
	riskFree := 0.03
	sharpe := kernels.CalculateSharpeRatio(returns, riskFree, 12.0)

	if math.IsNaN(sharpe) || sharpe <= 0 {
		t.Errorf("expected positive valid Sharpe Ratio, got: %.4f", sharpe)
	}
}

func TestPackedCashflowVector_SerializationCycle(t *testing.T) {
	vec := &PackedCashflowVector{
		Dates:   []int64{1704067200, 1719792000, 1735689600},
		Amounts: []float64{-500000.0, -100000.0, 750000.0},
	}

	bytes := vec.Serialize()
	deser, err := DeserializeCashflowVector(bytes)
	if err != nil {
		t.Fatalf("deserialization failed: %v", err)
	}

	if len(deser.Dates) != len(vec.Dates) || deser.Amounts[2] != vec.Amounts[2] {
		t.Errorf("roundtrip data mismatch")
	}
}

func BenchmarkVectorizedXIRR_100kIter(b *testing.B) {
	kernels := NewVectorizedFinancialKernels()
	t0 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	dates := []int64{t0, t0 + 86400*180, t0 + 86400*365, t0 + 86400*730}
	amounts := []float64{-1000000.0, -200000.0, 400000.0, 1100000.0}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = kernels.CalculateXIRR(dates, amounts, 50, 1e-6)
	}
}
