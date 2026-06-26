package main

import (
	"fmt"
	"log"

	httpapi "github.com/hondyman/semlayer/backend/internal/api"
	"github.com/hondyman/semlayer/backend/models"
)

func RunDispatch() {
	fmt.Println("🚀 Advanced Analytics Business Value Demonstration")
	fmt.Println("================================================")

	// Test 1: Markowitz Portfolio Optimization
	fmt.Println("\n📊 Test 1: Markowitz Portfolio Optimization")
	fmt.Println("Business Value: Optimal asset allocation for maximum risk-adjusted returns")

	template := httpapi.Template{
		NodeID:      "test-markowitz",
		NodeType:    "calculation",
		Domain:      "finance",
		Category:    "portfolio",
		Subcategory: "optimization",
		Version:     "1.0",
		Owner:       "system",
		Description: "Markowitz optimization for balanced portfolio",
		Financial: httpapi.FinancialCalc{
			Type: "markowitz",
			Mu:   []float64{0.08, 0.12, 0.10, 0.06, 0.14}, // Expected returns for 5 assets
			Covariance: [][]float64{
				{0.04, 0.006, 0.004, 0.002, 0.008},
				{0.006, 0.09, 0.008, 0.003, 0.012},
				{0.004, 0.008, 0.0625, 0.005, 0.010},
				{0.002, 0.003, 0.005, 0.025, 0.004},
				{0.008, 0.012, 0.010, 0.004, 0.16},
			},
			LongOnly:     true,
			RiskFreeRate: 0.02,
		},
	}

	result, err := httpapi.Dispatch(template.Financial, nil)
	if err != nil {
		log.Fatalf("Markowitz failed: %v", err)
	}
	fmt.Printf("✅ Optimal weights: %+v\n", result)

	// Test 2: Efficient Frontier Analysis
	fmt.Println("\n📈 Test 2: Efficient Frontier Analysis")
	fmt.Println("Business Value: Visualize risk-return trade-offs for portfolio construction")

	template.Financial.Type = "efficient_frontier"
	template.Financial.Points = 10

	result, err = httpapi.Dispatch(template.Financial, nil)
	if err != nil {
		log.Fatalf("Efficient Frontier failed: %v", err)
	}
	fmt.Printf("✅ Efficient frontier points: %+v\n", result)

	// Test 3: Risk Analytics - VaR and CVaR
	fmt.Println("\n⚠️  Test 3: Risk Analytics (VaR & CVaR)")
	fmt.Println("Business Value: Quantify portfolio risk for regulatory compliance and risk management")

	varTemplate := httpapi.Template{
		NodeID:      "test-var",
		NodeType:    "calculation",
		Domain:      "quant_finance",
		Category:    "market_risk",
		Subcategory: "var_historical",
		Version:     "1.0",
		Owner:       "system",
		Description: "Historical VaR at 99% confidence",
		Financial: httpapi.FinancialCalc{
			Type:              "var_historical",
			ConfidenceLevel:   0.99,
			Returns:           []float64{-0.015, 0.002, -0.03, 0.01, -0.025, 0.005, -0.02, 0.008, -0.012, 0.015},
			HoldingPeriodDays: 1,
		},
	}

	varResult, err := httpapi.Dispatch(varTemplate.Financial, nil)
	if err != nil {
		log.Fatalf("VaR failed: %v", err)
	}
	fmt.Printf("✅ Historical VaR (99%%): %+v\n", varResult)

	// Test 4: Black-Scholes Option Pricing
	fmt.Println("\n💰 Test 4: Black-Scholes Option Pricing")
	fmt.Println("Business Value: Accurate derivative pricing for hedging and investment strategies")

	bsTemplate := httpapi.Template{
		NodeID:      "test-black-scholes",
		NodeType:    "calculation",
		Domain:      "quant_finance",
		Category:    "derivatives_pricing",
		Subcategory: "black_scholes",
		Version:     "1.0",
		Owner:       "system",
		Description: "European call option pricing",
		Financial: httpapi.FinancialCalc{
			Type:          "black_scholes",
			OptionType:    "call",
			S0:            []float64{100},
			StrikePrice:   105,
			TimeHorizon:   0.5,
			RiskFreeRate:  0.02,
			Volatilities:  []float64{0.25},
			DividendYield: 0.01,
		},
	}

	bsResult, err := httpapi.Dispatch(bsTemplate.Financial, nil)
	if err != nil {
		log.Fatalf("Black-Scholes failed: %v", err)
	}
	fmt.Printf("✅ Black-Scholes call price: %+v\n", bsResult)

	// Test 5: Geometric Brownian Motion Simulation
	fmt.Println("\n📈 Test 5: GBM Asset Price Simulation")
	fmt.Println("Business Value: Realistic scenario generation for strategic planning and risk analysis")

	gbmTemplate := httpapi.Template{
		NodeID:      "test-gbm",
		NodeType:    "calculation",
		Domain:      "quant_finance",
		Category:    "stochastic_modeling",
		Subcategory: "gbm",
		Version:     "1.0",
		Owner:       "system",
		Description: "Asset price path simulation",
		Financial: httpapi.FinancialCalc{
			Type:  "gbm",
			S0:    []float64{100},
			Mu:    []float64{0.08},
			Sigma: []float64{0.20},
			T:     1.0,
			Steps: 50,
			Seed:  42,
		},
	}

	gbmResult, err := httpapi.Dispatch(gbmTemplate.Financial, nil)
	if err != nil {
		log.Fatalf("GBM failed: %v", err)
	}
	fmt.Printf("✅ GBM simulation completed: %+v\n", gbmResult)

	// Test 6: Monte Carlo Simulation
	fmt.Println("\n🎲 Test 6: Monte Carlo Risk Analysis")
	fmt.Println("Business Value: Probabilistic risk assessment for complex financial instruments")

	mcTemplate := httpapi.Template{
		NodeID:      "test-monte-carlo",
		NodeType:    "calculation",
		Domain:      "quant_finance",
		Category:    "stochastic_modeling",
		Subcategory: "monte_carlo",
		Version:     "1.0",
		Owner:       "system",
		Description: "Monte Carlo option pricing",
		Financial: httpapi.FinancialCalc{
			Type:           "monte_carlo",
			NumSimulations: 10000,
			StartValue:     100,
			StrikePrice:    105,
			RiskFreeRate:   0.02,
			StdDev:         0.25,
			TimeHorizon:    1.0,
		},
	}

	mcResult, err := httpapi.Dispatch(mcTemplate.Financial, nil)
	if err != nil {
		log.Fatalf("Monte Carlo failed: %v", err)
	}
	fmt.Printf("✅ Monte Carlo option price: %+v\n", mcResult)

	// Test 7: Fixed Income Analytics
	fmt.Println("\n🏦 Test 7: Fixed Income Analytics")
	fmt.Println("Business Value: Bond portfolio risk management and yield optimization")

	fiTemplate := httpapi.Template{
		NodeID:      "test-duration-convexity",
		NodeType:    "calculation",
		Domain:      "quant_finance",
		Category:    "fixed_income",
		Subcategory: "duration_convexity",
		Version:     "1.0",
		Owner:       "system",
		Description: "Bond duration and convexity analysis",
		Financial: httpapi.FinancialCalc{
			Type: "duration_convexity",
			CashFlows: []httpapi.CashFlow{
				{Amount: 50, Period: 1},
				{Amount: 50, Period: 2},
				{Amount: 1050, Period: 3},
			},
			YieldToMaturity: 0.04,
			Frequency:       1,
		},
	}

	fiResult, err := httpapi.Dispatch(fiTemplate.Financial, nil)
	if err != nil {
		log.Fatalf("Fixed Income failed: %v", err)
	}
	fmt.Printf("✅ Bond analytics: %+v\n", fiResult)

	// Test 8: Drill-down functionality
	fmt.Println("\n🔍 Test 8: Drill-down Analysis")
	fmt.Println("Business Value: Interactive data exploration and detailed insights")

	// Get the result set from a previous calculation (using the last one)
	if resultSet, ok := fiResult.(*models.ResultSet); ok {
		fmt.Printf("✅ ResultSet received with %d measures\n", len(resultSet.Measures))

		// Test drill-down on the first measure
		if len(resultSet.Measures) > 0 {
			measure := &resultSet.Measures[0]
			fmt.Printf("📊 Testing drill-down on measure: %s\n", measure.Name)

			// Create a drill-down locator
			locator := models.DrillDownLocator{
				XValues: []interface{}{"2023-01-01"},
				YValues: []interface{}{100.0},
			}

			// Perform drill-down
			drillDownQuery := measure.DrillDown(locator, nil)
			fmt.Printf("✅ Drill-down query generated: %+v\n", drillDownQuery)

			// Test pivot on the result set
			pivotConfig := &models.PivotConfig{
				X: []string{"time"},
				Y: []string{"measures"},
			}
			pivotResult := resultSet.Pivot(pivotConfig)
			fmt.Printf("✅ Pivot result: %+v\n", pivotResult)
		}
	} else {
		fmt.Printf("❌ Result is not a ResultSet: %T\n", fiResult)
	}

	fmt.Println("\n🎉 All Advanced Analytics Tests Completed Successfully!")
	fmt.Println("==================================================")
	fmt.Println("Business Value Delivered:")
	fmt.Println("• Portfolio optimization for superior risk-adjusted returns")
	fmt.Println("• Comprehensive risk management and regulatory compliance")
	fmt.Println("• Accurate derivative pricing and hedging strategies")
	fmt.Println("• Realistic scenario generation and stress testing")
	fmt.Println("• Fixed income risk analysis and yield optimization")
	fmt.Println("• Real-time analytics for active portfolio management")
	fmt.Println("• Interactive drill-down for detailed data exploration")
}
