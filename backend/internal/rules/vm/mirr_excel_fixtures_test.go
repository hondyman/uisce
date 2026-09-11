package vm

import "testing"

// Compatibility fixture: real values from Microsoft's own published MIRR
// documentation, fetched directly for this test rather than recalled -
// https://support.microsoft.com/en-us/office/mirr-function-b020f038-7492-4fb4-93c1-35c345b53524
//
// (Note for continuity: an earlier pass guessed this URL's last hex digit
// wrong - ...53482 instead of ...53524 - and got a 404, then gave up after
// one bad search. Re-searching rather than trusting the first 404 found it.)
//
// The example: a $120,000 initial cost financed at 10%/year, five years of
// income (39000/30000/21000/37000/46000), profits reinvested at 12%/year.
// Tolerance is loose (5e-3) because Microsoft's own docs only publish
// whole-percent results ("13%", "-5%").
func TestMIRR_ExcelFixture_MicrosoftDocsExample(t *testing.T) {
	cashFlows := []float64{-120000, 39000, 30000, 21000, 37000, 46000}

	cases := []struct {
		label        string
		cfs          []float64
		financeRate  float64
		reinvestRate float64
		expected     float64 // as published by Microsoft (rounded to whole %)
	}{
		{"MIRR(A2:A7,A8,A9) - five years, 10%/12%", cashFlows, 0.10, 0.12, 0.13},
		{"MIRR(A2:A5,A8,A9) - three years, 10%/12%", cashFlows[:4], 0.10, 0.12, -0.05},
		{"MIRR(A2:A7,A8,14%) - five years, 10%/14%", cashFlows, 0.10, 0.14, 0.13},
	}
	for _, c := range cases {
		got, err := solveMIRR(c.cfs, c.financeRate, c.reinvestRate)
		if err != nil {
			t.Fatalf("%s: %v", c.label, err)
		}
		assertNear(t, got, c.expected, 5e-3, c.label+" (vs. Microsoft's published MIRR example)")
	}
}
