package api

import (
	"fmt"
	"strconv"
)

// Result represents the output of a calculation.
type Result struct {
	Value  float64 `json:"value"`
	Error  string  `json:"error,omitempty"`
	Status string  `json:"status"` // "success", "error", "mock"
}

// Execute runs the financial calculation defined in the template.
func Execute(tmpl *Template) (*Result, error) {
	calc := tmpl.Financial

	switch calc.Type {
	case "irr":
		flows := make([]float64, len(calc.CashFlows))
		for i, cf := range calc.CashFlows {
			flows[i] = cf.Amount
		}
		irr := CalculateIRR(flows, calc.Guess)
		return &Result{Value: irr, Status: "success"}, nil

	case "amortization":
		// Assuming monthly rate if annual is provided, a common convention.
		monthlyRate := calc.Rate / 12
		payment, err := CalculateAmortizationPayment(monthlyRate, calc.Periods, calc.Principal)
		if err != nil {
			return nil, err
		}
		return &Result{Value: payment, Status: "success"}, nil

	case "ratio":
		// For ratios, the template provides SQL snippets. A full implementation
		// would require a data connection to execute these. Here, we'll attempt
		// to parse them as floats for simple cases, otherwise return a mock response.
		num, errNum := strconv.ParseFloat(calc.Numerator, 64)
		den, errDen := strconv.ParseFloat(calc.Denominator, 64)

		if errNum == nil && errDen == nil {
			if den == 0 {
				return nil, fmt.Errorf("denominator cannot be zero")
			}
			return &Result{Value: num / den, Status: "success"}, nil
		}

		// Return a mock result for SQL-based ratios
		return &Result{Value: 0.85, Status: "mock", Error: "SQL execution not implemented; returning mock data."}, nil

	default:
		return nil, fmt.Errorf("unknown calculation type: %s", calc.Type)
	}
}
