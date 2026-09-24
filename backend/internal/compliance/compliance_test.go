package compliance

import (
	"testing"
)

func TestPreTradeComplianceVM_ConcentrationLimitBreach(t *testing.T) {
	vm := NewPreTradeComplianceVM(nil)

	// Mandate: No single sector may exceed 20.0% of portfolio AUM
	rec := FastRecord{
		AccountAUM:        10000000.0, // $10,000,000 Total AUM
		CurrentPositionMV: 200000.0,
		ProposedOrderMV:   500000.0,  // +$500k Buy Order
		CurrentGroupMV:    1800000.0, // $1.8M existing Tech sector (18%)
	}

	// Projected Sector MV = $1.8M + $500k = $2.3M. Projected AUM = $10.5M
	// Projected Concentration = (2.3M / 10.5M) * 100 = 21.90% (Breach > 20.0%)
	res := vm.EvaluateTradeAgainstRule(rec, "LIMIT_SECTOR_MAX_20", "Max Tech Allocation", SeverityHardBlock, "<=", 20.0)

	if res.Passed {
		t.Fatalf("expected rule to breach, but passed")
	}
	if res.Severity != SeverityHardBlock {
		t.Errorf("expected HARD_BLOCK severity, got: %s", res.Severity)
	}
	if res.BreachDelta <= 0 {
		t.Errorf("expected positive breach delta, got: %.4f", res.BreachDelta)
	}
}


func BenchmarkPreTradeComplianceVM_SingleRuleCheck(b *testing.B) {
	vm := NewPreTradeComplianceVM(nil)
	rec := FastRecord{
		AccountAUM:        10000000.0,
		CurrentPositionMV: 200000.0,
		ProposedOrderMV:   100000.0,
		CurrentGroupMV:    1500000.0,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = vm.EvaluateTradeAgainstRule(rec, "BENCHMARK_VARIANCE_10", "Sector Variance", SeverityHardBlock, "<=", 20.0)
	}
}
