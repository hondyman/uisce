package finlib_test

import (
	"math"
	"testing"
	"time"

	"github.com/hondyman/uisce/backend/internal/boresolver/finlib"
)

func date(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

// TestXIRR_KnownFixture uses the Microsoft-documented XIRR example
// (a $10,000 investment returned in four irregular installments),
// expected result 0.373362535.
func TestXIRR_KnownFixture(t *testing.T) {
	flows := []finlib.Cashflow{
		{Date: date("2008-01-01"), Amount: -10000},
		{Date: date("2008-03-01"), Amount: 2750},
		{Date: date("2008-10-30"), Amount: 4250},
		{Date: date("2009-02-15"), Amount: 3250},
		{Date: date("2009-04-01"), Amount: 2750},
	}

	rate, err := finlib.XIRR(flows, 0.1)
	if err != nil {
		t.Fatalf("XIRR returned error: %v", err)
	}

	const expected = 0.373362535
	if math.Abs(rate-expected) > 1e-6 {
		t.Errorf("XIRR = %.9f, want %.9f", rate, expected)
	}
}

// TestXIRR_UnsortedInput verifies XIRR sorts cash flows by date internally,
// so callers don't need to pre-sort.
func TestXIRR_UnsortedInput(t *testing.T) {
	sorted := []finlib.Cashflow{
		{Date: date("2008-01-01"), Amount: -10000},
		{Date: date("2008-03-01"), Amount: 2750},
		{Date: date("2008-10-30"), Amount: 4250},
		{Date: date("2009-02-15"), Amount: 3250},
		{Date: date("2009-04-01"), Amount: 2750},
	}
	shuffled := []finlib.Cashflow{sorted[3], sorted[1], sorted[4], sorted[0], sorted[2]}

	rate, err := finlib.XIRR(shuffled, 0.1)
	if err != nil {
		t.Fatalf("XIRR returned error: %v", err)
	}
	if math.Abs(rate-0.373362535) > 1e-6 {
		t.Errorf("XIRR = %.9f, want 0.373362535", rate)
	}
}

// TestXIRR_SingleDateAllPositiveOrNegative rejects cash flows with no sign
// change — there is no rate that could make the NPV cross zero.
func TestXIRR_NoSignChange(t *testing.T) {
	flows := []finlib.Cashflow{
		{Date: date("2020-01-01"), Amount: 100},
		{Date: date("2020-06-01"), Amount: 200},
	}
	_, err := finlib.XIRR(flows, 0.1)
	if err != finlib.ErrNoSignChange {
		t.Errorf("expected ErrNoSignChange, got %v", err)
	}
}

func TestXIRR_InsufficientCashflows(t *testing.T) {
	_, err := finlib.XIRR([]finlib.Cashflow{{Date: date("2020-01-01"), Amount: -100}}, 0.1)
	if err != finlib.ErrInsufficientCashflows {
		t.Errorf("expected ErrInsufficientCashflows, got %v", err)
	}
}

// TestXIRR_AdversarialPattern has a late, large sign reversal — a pattern
// known to make Newton's method overshoot into the undefined domain
// (rate <= -1) and diverge, forcing the bisection fallback.
func TestXIRR_AdversarialPattern(t *testing.T) {
	flows := []finlib.Cashflow{
		{Date: date("2020-01-01"), Amount: -1000},
		{Date: date("2020-02-01"), Amount: -1000},
		{Date: date("2020-03-01"), Amount: -1000},
		{Date: date("2029-12-01"), Amount: 100000},
	}

	rate, err := finlib.XIRR(flows, 0.1)
	if err != nil {
		t.Fatalf("XIRR returned error: %v", err)
	}

	// Sanity check via NPV at the solved rate rather than a magic constant:
	// the defining property of an IRR is that NPV(rate) ~= 0.
	npv := 0.0
	t0 := flows[0].Date
	for _, cf := range flows {
		years := cf.Date.Sub(t0).Hours() / 24 / 365.0
		npv += cf.Amount / math.Pow(1+rate, years)
	}
	if math.Abs(npv) > 1e-4 {
		t.Errorf("NPV at solved rate %.9f = %.6f, want ~0", rate, npv)
	}
}

// TestIRR_KnownFixture uses the Microsoft-documented IRR example: an initial
// -70,000 investment followed by five annual returns, expected ~8.66%.
func TestIRR_KnownFixture(t *testing.T) {
	flows := []float64{-70000, 12000, 15000, 18000, 21000, 26000}

	rate, err := finlib.IRR(flows, 0.1)
	if err != nil {
		t.Fatalf("IRR returned error: %v", err)
	}

	const expected = 0.0866
	if math.Abs(rate-expected) > 1e-3 {
		t.Errorf("IRR = %.6f, want ~%.4f", rate, expected)
	}
}

func TestIRR_NoSignChange(t *testing.T) {
	_, err := finlib.IRR([]float64{100, 200, 300}, 0.1)
	if err != finlib.ErrNoSignChange {
		t.Errorf("expected ErrNoSignChange, got %v", err)
	}
}
