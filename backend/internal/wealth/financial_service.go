package wealth

import (
	"fmt"
	"math"
)

// FinancialService provides financial calculation functions
type FinancialService struct{}

// NewFinancialService creates a new financial service
func NewFinancialService() *FinancialService {
	return &FinancialService{}
}

// IRR calculates the Internal Rate of Return for a series of cash flows
func (fs *FinancialService) IRR(cashFlows []float64, guess float64) (float64, error) {
	if len(cashFlows) == 0 {
		return 0, fmt.Errorf("cash flows cannot be empty")
	}

	if guess == 0 {
		guess = 0.1 // Default 10%
	}

	// Use Newton-Raphson method to find IRR
	return fs.newtonRaphsonIRR(cashFlows, guess)
}

// XIRR calculates the Internal Rate of Return for irregular cash flows with dates
func (fs *FinancialService) XIRR(cashFlows []float64, dates []string, guess float64) (float64, error) {
	if len(cashFlows) != len(dates) {
		return 0, fmt.Errorf("cash flows and dates must have the same length")
	}

	if len(cashFlows) == 0 {
		return 0, fmt.Errorf("cash flows cannot be empty")
	}

	if guess == 0 {
		guess = 0.1 // Default 10%
	}

	// Convert dates to days since first date
	days := make([]float64, len(dates))
	if len(dates) > 0 {
		// For simplicity, assume dates are in chronological order
		// In production, you'd parse actual dates
		for i := range dates {
			days[i] = float64(i*365) / 365.0 // Assume annual periods for now
		}
	}

	return fs.newtonRaphsonXIRR(cashFlows, days, guess)
}

// NPV calculates the Net Present Value
func (fs *FinancialService) NPV(rate float64, cashFlows []float64) float64 {
	npv := 0.0
	for i, cf := range cashFlows {
		if i == 0 {
			npv += cf
		} else {
			npv += cf / math.Pow(1+rate, float64(i))
		}
	}
	return npv
}

// newtonRaphsonIRR implements Newton-Raphson method for IRR calculation
func (fs *FinancialService) newtonRaphsonIRR(cashFlows []float64, guess float64) (float64, error) {
	const maxIterations = 100
	const tolerance = 1e-6

	rate := guess

	for i := 0; i < maxIterations; i++ {
		npv := fs.NPV(rate, cashFlows)
		dnpv := fs.npvDerivative(rate, cashFlows)

		if math.Abs(dnpv) < tolerance {
			break
		}

		newRate := rate - npv/dnpv

		if math.Abs(newRate-rate) < tolerance {
			return newRate, nil
		}

		rate = newRate
	}

	return rate, nil
}

// newtonRaphsonXIRR implements Newton-Raphson method for XIRR calculation
func (fs *FinancialService) newtonRaphsonXIRR(cashFlows []float64, days []float64, guess float64) (float64, error) {
	const maxIterations = 100
	const tolerance = 1e-6

	rate := guess

	for i := 0; i < maxIterations; i++ {
		npv := fs.xnpv(rate, cashFlows, days)
		dnpv := fs.xnpvDerivative(rate, cashFlows, days)

		if math.Abs(dnpv) < tolerance {
			break
		}

		newRate := rate - npv/dnpv

		if math.Abs(newRate-rate) < tolerance {
			return newRate, nil
		}

		rate = newRate
	}

	return rate, nil
}

// npvDerivative calculates the derivative of NPV for Newton-Raphson
func (fs *FinancialService) npvDerivative(rate float64, cashFlows []float64) float64 {
	derivative := 0.0
	for i := 1; i < len(cashFlows); i++ {
		derivative -= float64(i) * cashFlows[i] / math.Pow(1+rate, float64(i+1))
	}
	return derivative
}

// xnpv calculates XNPV for irregular cash flows
func (fs *FinancialService) xnpv(rate float64, cashFlows []float64, days []float64) float64 {
	xnpv := 0.0
	for i, cf := range cashFlows {
		if i == 0 {
			xnpv += cf
		} else {
			xnpv += cf / math.Pow(1+rate, days[i])
		}
	}
	return xnpv
}

// xnpvDerivative calculates the derivative of XNPV
func (fs *FinancialService) xnpvDerivative(rate float64, cashFlows []float64, days []float64) float64 {
	derivative := 0.0
	for i := 1; i < len(cashFlows); i++ {
		derivative -= days[i] * cashFlows[i] / math.Pow(1+rate, days[i]+1)
	}
	return derivative
}

// WIRR calculates Weighted Internal Rate of Return
func (fs *FinancialService) WIRR(cashFlows []float64, weights []float64, guess float64) (float64, error) {
	if len(cashFlows) != len(weights) {
		return 0, fmt.Errorf("cash flows and weights must have the same length")
	}

	// Weight the cash flows
	weightedFlows := make([]float64, len(cashFlows))
	for i := range cashFlows {
		weightedFlows[i] = cashFlows[i] * weights[i]
	}

	return fs.IRR(weightedFlows, guess)
}

// AmortizationPayment calculates the fixed periodic payment for a loan (PMT).
// rate: interest rate per period.
// periods: total number of payment periods.
// principal: the present value, or the total amount that a series of future payments is worth now.
func (fs *FinancialService) AmortizationPayment(rate float64, periods int, principal float64) (float64, error) {
	if periods <= 0 {
		return 0, fmt.Errorf("number of periods must be positive")
	}
	if rate == 0 {
		return -principal / float64(periods), nil
	}
	ratePow := math.Pow(1+rate, float64(periods))
	payment := -principal * (rate * ratePow) / (ratePow - 1)
	return payment, nil
}

// Ratio calculates a simple ratio of numerator / denominator.
func (fs *FinancialService) Ratio(numerator, denominator float64) (float64, error) {
	if denominator == 0 {
		return 0, fmt.Errorf("denominator cannot be zero")
	}
	return numerator / denominator, nil
}

// PaybackPeriod calculates the number of periods required to recover the initial investment.
// Assumes the first cash flow is the initial investment (negative).
func (fs *FinancialService) PaybackPeriod(cashFlows []float64) (float64, error) {
	if len(cashFlows) == 0 || cashFlows[0] >= 0 {
		return 0, fmt.Errorf("cash flows must start with a negative initial investment")
	}

	initialInvestment := -cashFlows[0]
	cumulativeCashFlow := 0.0
	for i := 1; i < len(cashFlows); i++ {
		cumulativeCashFlow += cashFlows[i]
		if cumulativeCashFlow >= initialInvestment {
			lastCumulative := cumulativeCashFlow - cashFlows[i]
			amountNeeded := initialInvestment - lastCumulative
			return float64(i-1) + (amountNeeded / cashFlows[i]), nil
		}
	}

	return 0, fmt.Errorf("investment not recovered within the given cash flow periods")
}

// WeightedSum calculates the sum of values multiplied by their corresponding weights.
func (fs *FinancialService) WeightedSum(values []float64, weights []float64) (float64, error) {
	if len(values) != len(weights) {
		return 0, fmt.Errorf("values and weights must have the same length")
	}

	total := 0.0
	for i := range values {
		total += values[i] * weights[i]
	}
	return total, nil
}

// MIRR calculates the Modified Internal Rate of Return.
func (fs *FinancialService) MIRR(cashFlows []float64, financeRate, reinvestRate float64) (float64, error) {
	if len(cashFlows) < 2 {
		return 0, fmt.Errorf("at least two cash flows are required for MIRR")
	}

	n := float64(len(cashFlows) - 1)
	positiveFlows := 0.0
	negativeFlows := 0.0

	for i, cf := range cashFlows {
		if cf > 0 {
			positiveFlows += cf * math.Pow(1+reinvestRate, n-float64(i))
		} else {
			negativeFlows += cf / math.Pow(1+financeRate, float64(i))
		}
	}

	if negativeFlows == 0 || positiveFlows == 0 {
		return 0, fmt.Errorf("MIRR requires both positive and negative cash flows")
	}

	mirr := math.Pow(-positiveFlows/negativeFlows, 1/n) - 1
	return mirr, nil
}

// CAGR calculates the Compound Annual Growth Rate.
func (fs *FinancialService) CAGR(startValue, endValue, years float64) (float64, error) {
	if startValue == 0 || years <= 0 {
		return 0, fmt.Errorf("start value and years must be positive")
	}
	return math.Pow(endValue/startValue, 1/years) - 1, nil
}

// SharpeRatio calculates the risk-adjusted return.
func (fs *FinancialService) SharpeRatio(averageReturn, riskFreeRate, stdDev float64) (float64, error) {
	if stdDev == 0 {
		return 0, fmt.Errorf("standard deviation cannot be zero")
	}
	return (averageReturn - riskFreeRate) / stdDev, nil
}

// SumOfRatiosComponent defines a single numerator/denominator pair.
type SumOfRatiosComponent struct {
	Numerator   float64 `json:"numerator"`
	Denominator float64 `json:"denominator"`
}

// SumOfRatios calculates the sum of multiple ratios.
func (fs *FinancialService) SumOfRatios(components []SumOfRatiosComponent) (float64, error) {
	totalRatio := 0.0
	for _, component := range components {
		if component.Denominator == 0 {
			return 0, fmt.Errorf("denominator cannot be zero in one of the components")
		}
		totalRatio += component.Numerator / component.Denominator
	}
	return totalRatio, nil
}
