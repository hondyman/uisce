package altinvest

import (
	"math"
	"testing"
	"time"
)

func TestComputeXIRRFromFlows(t *testing.T) {
	// Standard test case:
	// 2021-01-01: -100,000 (Initial Investment)
	// 2021-07-01:  -50,000 (Follow-on Call)
	// 2022-01-01:   20,000 (Distribution)
	// 2023-01-01:  180,000 (Terminal Valuation)
	t1, _ := time.Parse("2006-01-02", "2021-01-01")
	t2, _ := time.Parse("2006-01-02", "2021-07-01")
	t3, _ := time.Parse("2006-01-02", "2022-01-01")
	t4, _ := time.Parse("2006-01-02", "2023-01-01")

	flows := []XIRRFlow{
		{Date: t1, Amount: -100000.0, Type: "CAPITAL_CALL"},
		{Date: t2, Amount: -50000.0, Type: "CAPITAL_CALL"},
		{Date: t3, Amount: 20000.0, Type: "DISTRIBUTION"},
		{Date: t4, Amount: 180000.0, Type: "TERMINAL_NAV"},
	}

	xirr, err := ComputeXIRRFromFlows(flows)
	if err != nil {
		t.Fatalf("unexpected error in ComputeXIRRFromFlows: %v", err)
	}

	if xirr < 0.15 || xirr > 0.22 {
		t.Errorf("expected XIRR to be between 0.15 and 0.22, got %f", xirr)
	}

	// Test NPV at calculated XIRR is close to 0
	refDate := flows[0].Date
	npv := 0.0
	for _, f := range flows {
		years := f.Date.Sub(refDate).Hours() / (24 * 365.25)
		npv += f.Amount / math.Pow(1+xirr, years)
	}

	if math.Abs(npv) > 0.01 {
		t.Errorf("expected NPV at XIRR to be ~0, got %f", npv)
	}
}
