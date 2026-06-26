package pricing

import (
	"fmt"
	"strings"
)

// ISO 4217 subset for currency validation
var isoCurrencies = map[string]bool{
	"USD": true, "EUR": true, "GBP": true, "JPY": true, "CHF": true,
	"CAD": true, "AUD": true, "HKD": true, "SGD": true, "CNY": true,
	"SEK": true, "NOK": true, "DKK": true, "NZD": true, "MXN": true,
	"INR": true, "BRL": true, "ZAR": true, "KRW": true, "TRY": true,
}

// DQResult captures the outcome of a DQ validation.
type DQResult struct {
	RuleName string
	Passed   bool
	Error    string
}

// ValidatePrice validates a raw price record against all DQ rules.
// Returns a slice of DQResults and a convenience "has hard failures" bool.
func ValidatePrice(r *RawPriceRecord) ([]DQResult, bool) {
	var results []DQResult
	hardFail := false

	required := func(field, value, name string) {
		if strings.TrimSpace(value) == "" {
			results = append(results, DQResult{name, false, fmt.Sprintf("%s is required", field)})
			hardFail = true
		} else {
			results = append(results, DQResult{name, true, ""})
		}
	}

	required("security_id", r.SecurityID, "Price_Required_SecurityID")
	required("price_type", r.PriceType, "Price_Required_PriceType")
	required("price_date", r.PriceDate, "Price_Required_PriceDate")
	required("price_currency", r.PriceCurrency, "Price_Required_PriceCurrency")
	required("price_source", r.PriceSource, "Price_Required_PriceSource")

	// Positive price value
	if r.PriceValue <= 0 {
		results = append(results, DQResult{"Price_ValueValidity", false, "price_value must be positive"})
		hardFail = true
	} else {
		results = append(results, DQResult{"Price_ValueValidity", true, ""})
	}

	// ISO currency
	if _, ok := isoCurrencies[strings.ToUpper(r.PriceCurrency)]; !ok {
		results = append(results, DQResult{"Price_CurrencyValidity", false,
			fmt.Sprintf("'%s' is not a recognised ISO-4217 currency", r.PriceCurrency)})
		// Soft failure — don't set hardFail
	} else {
		results = append(results, DQResult{"Price_CurrencyValidity", true, ""})
	}

	return results, hardFail
}

// ValidateFXRate validates a raw FX record against all DQ rules.
func ValidateFXRate(r *RawFXRecord) ([]DQResult, bool) {
	var results []DQResult
	hardFail := false

	required := func(field, value, name string) {
		if strings.TrimSpace(value) == "" {
			results = append(results, DQResult{name, false, fmt.Sprintf("%s is required", field)})
			hardFail = true
		} else {
			results = append(results, DQResult{name, true, ""})
		}
	}

	required("base_currency", r.BaseCurrency, "FX_Required_BaseCurrency")
	required("quote_currency", r.QuoteCurrency, "FX_Required_QuoteCurrency")
	required("fx_rate_date", r.FXRateDate, "FX_Required_FXRateDate")
	required("fx_source", r.FXSource, "FX_Required_FXSource")

	// Positive rate
	if r.FXRate <= 0 {
		results = append(results, DQResult{"FX_RateValidity", false, "fx_rate must be positive"})
		hardFail = true
	} else {
		results = append(results, DQResult{"FX_RateValidity", true, ""})
	}

	// Currency validation
	for _, cc := range []struct{ code, rule string }{
		{r.BaseCurrency, "FX_BaseCurrencyValidity"},
		{r.QuoteCurrency, "FX_QuoteCurrencyValidity"},
	} {
		if _, ok := isoCurrencies[strings.ToUpper(cc.code)]; !ok {
			results = append(results, DQResult{cc.rule, false,
				fmt.Sprintf("'%s' is not a recognised ISO-4217 currency", cc.code)})
		} else {
			results = append(results, DQResult{cc.rule, true, ""})
		}
	}

	return results, hardFail
}

// ValidateCurve validates a raw curve record.
func ValidateCurve(r *RawCurveRecord) ([]DQResult, bool) {
	var results []DQResult
	hardFail := false

	required := func(field, value, name string) {
		if strings.TrimSpace(value) == "" {
			results = append(results, DQResult{name, false, fmt.Sprintf("%s is required", field)})
			hardFail = true
		} else {
			results = append(results, DQResult{name, true, ""})
		}
	}

	required("curve_type", r.CurveType, "Curve_Required_CurveType")
	required("curve_currency", r.CurveCurrency, "Curve_Required_CurveCurrency")
	required("curve_as_of_date", r.CurveAsOfDate, "Curve_Required_CurveAsOfDate")

	// Tenor points
	if len(r.CurveTenorPoints) == 0 {
		results = append(results, DQResult{"Curve_TenorPointsValidity", false, "curve_tenor_points must have at least one point"})
		hardFail = true
	} else {
		results = append(results, DQResult{"Curve_TenorPointsValidity", true, ""})
	}

	// Validate each tenor point has a tenor and rate
	for i, tp := range r.CurveTenorPoints {
		if tp.Tenor == "" || tp.Rate == 0 {
			results = append(results, DQResult{
				fmt.Sprintf("Curve_TenorPoint_%d_Validity", i), false,
				fmt.Sprintf("tenor point %d is missing tenor or rate", i),
			})
		}
	}

	return results, hardFail
}

// ValidateVolSurface validates a raw vol surface record.
func ValidateVolSurface(r *RawVolRecord) ([]DQResult, bool) {
	var results []DQResult
	hardFail := false

	required := func(field, value, name string) {
		if strings.TrimSpace(value) == "" {
			results = append(results, DQResult{name, false, fmt.Sprintf("%s is required", field)})
			hardFail = true
		} else {
			results = append(results, DQResult{name, true, ""})
		}
	}

	required("underlier_security_id", r.UnderlierSecurityID, "Vol_Required_UnderlierSecurityID")
	required("vol_surface_type", r.VolSurfaceType, "Vol_Required_VolSurfaceType")
	required("vol_as_of_date", r.VolAsOfDate, "Vol_Required_VolAsOfDate")

	// Grid structure validation
	grid := r.VolGrid
	if len(grid.Strikes) == 0 || len(grid.Tenors) == 0 || len(grid.Vols) == 0 {
		results = append(results, DQResult{"Vol_GridValidity", false, "vol_grid must have strikes, tenors, and vols"})
		hardFail = true
	} else if len(grid.Vols) != len(grid.Tenors) {
		results = append(results, DQResult{"Vol_GridDimensions", false,
			fmt.Sprintf("vol_grid row count (%d) must equal tenors count (%d)", len(grid.Vols), len(grid.Tenors))})
		hardFail = true
	} else {
		results = append(results, DQResult{"Vol_GridValidity", true, ""})
	}

	return results, hardFail
}

// DQSummary extracts passed and failed rule names from a DQ result slice.
func DQSummary(results []DQResult) (passed, failed []string, errors []string) {
	for _, r := range results {
		if r.Passed {
			passed = append(passed, r.RuleName)
		} else {
			failed = append(failed, r.RuleName)
			errors = append(errors, r.Error)
		}
	}
	return
}
