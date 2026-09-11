package vm

import "testing"

// Compatibility fixtures: values genuinely computed by Excel, not
// constructed or cross-checked against an independent solver (that's
// irr_test.go's job - correctness). This file exists for a different
// question: "does this match what a user's spreadsheet says?" - an
// interoperability surface, not just a correctness one, especially for
// XIRR where Excel's actual/365 day-count and iterative solver behavior
// on awkward cash-flow patterns matter beyond the mathematical result.
//
// Source: Microsoft's own published IRR/XIRR function documentation
// (not this repo's memory of it) -
//
//	https://support.microsoft.com/en-us/office/irr-function-64925eaa-9988-495b-b290-3ad0c163c1bc
//	https://support.microsoft.com/en-us/office/xirr-function-de1242ec-6477-445b-b11b-a303ad9adc9d
//
// fetched directly for this test rather than recalled, and reproduced
// verbatim (cash flows, dates, published results) below. Tolerances are
// loose (1e-3) where Microsoft's own docs only publish a rounded
// percentage (e.g. "8.7%"); tight (1e-6) for XIRR's example, where the
// doc publishes a 9-significant-digit result (0.373362535).
func TestIRR_ExcelFixture_MicrosoftDocsExample(t *testing.T) {
	// Business example from Microsoft's IRR docs: -$70,000 initial cost,
	// then net income of $12,000/$15,000/$18,000/$21,000/$26,000 in
	// years 1-5.
	cashFlows := []float64{-70000, 12000, 15000, 18000, 21000, 26000}

	cases := []struct {
		label    string
		cfs      []float64
		expected float64 // as published by Microsoft (rounded to 1 decimal %)
	}{
		{"IRR(A2:A6) - four years", cashFlows[:5], -0.021},
		{"IRR(A2:A7) - five years", cashFlows[:6], 0.087},
		{"IRR(A2:A4) - two years", cashFlows[:3], -0.444},
	}
	for _, c := range cases {
		got, err := solveIRR(c.cfs, integerPeriods(len(c.cfs)))
		if err != nil {
			t.Fatalf("%s: %v", c.label, err)
		}
		assertNear(t, got, c.expected, 1e-3, c.label+" (vs. Microsoft's published IRR example)")
	}
}

func TestXIRR_ExcelFixture_MicrosoftDocsExample(t *testing.T) {
	// Microsoft's XIRR docs example: an initial investment plus four
	// irregularly-dated distributions.
	cashFlows := []float64{-10000, 2750, 4250, 3250, 2750}
	// 1-Jan-08, 1-Mar-08, 30-Oct-08, 15-Feb-09, 1-Apr-09 - day offsets
	// from 1-Jan-08, computed via Python's datetime (a real calendar
	// computation, not hand arithmetic prone to off-by-one leap-year
	// errors).
	days := []float64{0, 60, 303, 411, 456}

	got, err := solveIRR(cashFlows, dayPeriods(days))
	if err != nil {
		t.Fatal(err)
	}
	// Microsoft publishes 0.373362535 (37.34%) for this exact example.
	assertNear(t, got, 0.373362535, 1e-6, "XIRR (vs. Microsoft's published XIRR example)")
}
