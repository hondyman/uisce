package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"strings" // Added missing import

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// SemanticTermProperties mirrors the JSONB structure in catalog_node.properties
type SemanticTermProperties struct {
	Type        string                 `json:"type"`                 // "calculated", "physical", etc.
	DataType    string                 `json:"data_type"`            // "number", "string", "boolean"
	Expression  string                 `json:"expression"`           // The SQL or Formula
	DisplayName string                 `json:"display_name"`         // Optional, if we want it here
	Tags        []string               `json:"tags"`                 // [Category, Subcategory]
	Attributes  map[string]interface{} `json:"attributes,omitempty"` // Extra metadata
}

type LibraryItem struct {
	Name          string
	Title         string
	Description   string
	SQL           string
	ReturnType    string // "percent", "currency", "number", "string"
	Category      string
	Subcategory   string
	FinancialCalc map[string]interface{}
}

// Full library ported from financialCalculations.ts + Investment ORM & Northwind Domains
var library = []LibraryItem{
	// --- INVESTMENT ORM DOMAIN CALCULATIONS ---
	{
		Name:        "net_fund_yield",
		Title:       "Net Fund Yield",
		SQL:         "${gross_return} - ${management_fee}",
		Description: "Net annual yield of a fund after deducting annual management fees.",
		Category:    "Investment ORM",
		Subcategory: "Fund Performance",
		ReturnType:  "percent",
		FinancialCalc: map[string]interface{}{
			"term_type":              "calculated",
			"execution_strategy":     "sql",
			"formula":                 "${gross_return} - ${management_fee}",
			"aggregation_function":   "AVG",
			"drill_down_dimensions":  "fund_id, fund_type, region, portfolio_manager",
		},
	},
	{
		Name:        "gross_return",
		Title:       "Gross Fund Return",
		SQL:         "(${total_revenue} / NULLIF(${total_aum}, 0)) * 100",
		Description: "Gross percentage return generated relative to total assets under management.",
		Category:    "Investment ORM",
		Subcategory: "Fund Performance",
		ReturnType:  "percent",
		FinancialCalc: map[string]interface{}{
			"term_type":              "calculated",
			"execution_strategy":     "sql",
			"formula":                 "(${total_revenue} / NULLIF(${total_aum}, 0)) * 100",
			"aggregation_function":   "AVG",
			"drill_down_dimensions":  "fund_id, asset_class",
		},
	},
	{
		Name:        "total_portfolio_exposure",
		Title:       "Total Portfolio Exposure",
		SQL:         "SUM(${position_market_value}) + SUM(${derivative_notional})",
		Description: "Aggregate aggregate economic exposure across cash securities and derivative contracts.",
		Category:    "Investment ORM",
		Subcategory: "Risk & Exposure",
		ReturnType:  "currency",
		FinancialCalc: map[string]interface{}{
			"term_type":              "preaggregated",
			"execution_strategy":     "preaggregated",
			"formula":                 "SUM(${position_market_value}) + SUM(${derivative_notional})",
			"aggregation_function":   "SUM",
			"drill_down_dimensions":  "portfolio_id, asset_class, counterparty, sector",
		},
	},
	{
		Name:        "equity_risk_weight",
		Title:       "Equity Risk Weight",
		SQL:         "(${position_market_value} / NULLIF(${total_portfolio_value}, 0)) * ${beta_coefficient}",
		Description: "Weighted systematic equity risk contribution per asset position.",
		Category:    "Investment ORM",
		Subcategory: "Risk Analytics",
		ReturnType:  "number",
		FinancialCalc: map[string]interface{}{
			"term_type":              "calculated",
			"execution_strategy":     "on_the_fly",
			"formula":                 "(${position_market_value} / NULLIF(${total_portfolio_value}, 0)) * ${beta_coefficient}",
			"aggregation_function":   "SUM",
			"drill_down_dimensions":  "security_id, sector, country",
		},
	},

	// --- NORTHWIND COMMERCE DOMAIN CALCULATIONS ---
	{
		Name:        "line_item_net_total",
		Title:       "Line Item Net Total",
		SQL:         "(${unit_price} * ${quantity}) * (1.0 - ${discount})",
		Description: "Total billed amount for an order line item after applying unit discounts.",
		Category:    "Northwind",
		Subcategory: "Order Fulfillment",
		ReturnType:  "currency",
		FinancialCalc: map[string]interface{}{
			"term_type":              "calculated",
			"execution_strategy":     "sql",
			"formula":                 "(${unit_price} * ${quantity}) * (1.0 - ${discount})",
			"aggregation_function":   "SUM",
			"drill_down_dimensions":  "order_id, product_id, customer_id, employee_id",
		},
	},
	{
		Name:        "order_discount_amount",
		Title:       "Order Discount Amount",
		SQL:         "(${unit_price} * ${quantity}) * ${discount}",
		Description: "Total value subtracted from gross price via promotional discounts.",
		Category:    "Northwind",
		Subcategory: "Pricing & Promotions",
		ReturnType:  "currency",
		FinancialCalc: map[string]interface{}{
			"term_type":              "calculated",
			"execution_strategy":     "sql",
			"formula":                 "(${unit_price} * ${quantity}) * ${discount}",
			"aggregation_function":   "SUM",
			"drill_down_dimensions":  "order_id, product_id, customer_id",
		},
	},
	{
		Name:        "gross_profit_margin",
		Title:       "Gross Profit Margin %",
		SQL:         "((${revenue} - ${cogs}) / NULLIF(${revenue}, 0)) * 100",
		Description: "Percentage of revenue retained after deducting cost of goods sold.",
		Category:    "Northwind",
		Subcategory: "Financial Metrics",
		ReturnType:  "percent",
		FinancialCalc: map[string]interface{}{
			"term_type":              "calculated",
			"execution_strategy":     "sql",
			"formula":                 "((${revenue} - ${cogs}) / NULLIF(${revenue}, 0)) * 100",
			"aggregation_function":   "AVG",
			"drill_down_dimensions":  "category_id, supplier_id, product_id, region",
		},
	},
	{
		Name:        "monthly_order_velocity",
		Title:       "Monthly Order Velocity",
		SQL:         "COUNT(${order_id}) / NULLIF(COUNT(DISTINCT ${customer_id}), 0)",
		Description: "Pre-aggregated average orders placed per customer per monthly cycle.",
		Category:    "Northwind",
		Subcategory: "Customer Analytics",
		ReturnType:  "number",
		FinancialCalc: map[string]interface{}{
			"term_type":              "preaggregated",
			"execution_strategy":     "preaggregated",
			"formula":                 "COUNT(${order_id}) / NULLIF(COUNT(DISTINCT ${customer_id}), 0)",
			"aggregation_function":   "AVG",
			"drill_down_dimensions":  "customer_id, country, territory, year_month",
		},
	},

	// --- STANDARD QUANT & FINANCIAL LIBRARY ---
	{Name: "investment_xirr", Title: "Investment XIRR", SQL: "{{ xirr(ARRAY_AGG(${pre_agg_name}.cash_flow), ARRAY_AGG(${pre_agg_name}.transaction_date)) }}", Description: "Calculate the internal rate of return for a series of cash flows that is not necessarily periodic.", Category: "Private Markets", Subcategory: "IRR", ReturnType: "percent"},
	{Name: "net_present_value", Title: "Net Present Value", SQL: "NPV(discount_rate, value1, value2)", Description: "Calculate the net present value of an investment using a discount rate and a series of future payments", Category: "Performance", Subcategory: "Valuation", ReturnType: "currency"},
	{Name: "sharpe_ratio", Title: "Sharpe Ratio", SQL: "(AVG(returns) - risk_free_rate) / STDDEV(returns)", Description: "Measure risk-adjusted return by calculating excess return per unit of risk", Category: "Risk", Subcategory: "Volatility", ReturnType: "number"},
	{Name: "value_at_risk", Title: "Value at Risk (VaR)", SQL: "PERCENTILE(portfolio_returns, 0.05) * portfolio_value", Description: "Estimate the potential loss in value of a portfolio over a defined period for a given confidence interval", Category: "Risk", Subcategory: "Market Risk", ReturnType: "currency"},
	{Name: "total_return", Title: "Total Return", SQL: "((ending_value + dividends) / beginning_value) - 1", Description: "Calculate the actual rate of return including capital appreciation and income", Category: "Performance", Subcategory: "Returns", ReturnType: "percent"},
	{Name: "compound_annual_growth", Title: "CAGR", SQL: "POWER(ending_value / beginning_value, 1.0 / years) - 1", Description: "Compound Annual Growth Rate", Category: "Performance", Subcategory: "Growth", ReturnType: "percent"},
	{Name: "beta_coefficient", Title: "Beta Coefficient", SQL: "COVAR(stock_returns, market_returns) / VAR(market_returns)", Description: "Measure of systematic risk", Category: "Risk", Subcategory: "Correlation", ReturnType: "number"},
	{Name: "portfolio_allocation", Title: "Portfolio Allocation %", SQL: "(position_value / total_portfolio_value) * 100", Description: "Calculate the percentage allocation of each position", Category: "Wealth", Subcategory: "Allocation", ReturnType: "percent"},
	{Name: "drawdown_max", Title: "Maximum Drawdown", SQL: "MIN((current_value - peak_value) / peak_value)", Description: "Measure the largest peak-to-trough decline", Category: "Risk", Subcategory: "Drawdown", ReturnType: "percent"},
	{Name: "irr_calculation", Title: "Internal Rate of Return", SQL: "{{ irr(ARRAY_AGG(${pre_agg_name}.cash_flow)) }}", Description: "Calculate the discount rate that makes NPV equal to zero", Category: "Private Markets", Subcategory: "IRR", ReturnType: "percent"},
	{Name: "multiple_invested_capital", Title: "Multiple of Invested Capital", SQL: "total_distributions / total_contributions", Description: "Private equity metric showing total value returned relative to capital invested", Category: "Private Markets", Subcategory: "Multiples", ReturnType: "number"},
	{Name: "distributed_paid_in", Title: "DPI Ratio", SQL: "cumulative_distributions / paid_in_capital", Description: "Distributed to Paid-In capital ratio", Category: "Private Markets", Subcategory: "Ratios", ReturnType: "number"},
	{Name: "net_present_value_calc", Title: "Net Present Value (NPV)", SQL: "{{ npv(0.08, ARRAY_AGG(${pre_agg_name}.cash_flow)) }}", Description: "Calculates the present value of future cash flows minus initial investment", Category: "Performance", Subcategory: "Valuation", ReturnType: "currency"},
	{Name: "modified_irr_calc", Title: "Modified IRR (MIRR)", SQL: "{{ mirr(ARRAY_AGG(${pre_agg_name}.cash_flow), 0.07, 0.05) }}", Description: "Calculates MIRR, accounting for cost of capital and reinvestment", Category: "Performance", Subcategory: "IRR", ReturnType: "percent"},
	{Name: "payback_period_calc", Title: "Payback Period", SQL: "{{ payback_period(ARRAY_AGG(${pre_agg_name}.cash_flow)) }}", Description: "Calculates the time required to recover the initial investment", Category: "Performance", Subcategory: "Valuation", ReturnType: "number"},
	{Name: "weighted_irr_calc", Title: "Weighted IRR (WIRR)", SQL: "{{ wirr(ARRAY_AGG(${pre_agg_name}.cash_flow), ARRAY_AGG(${pre_agg_name}.weight)) }}", Description: "Calculates the portfolio IRR weighted by investment size", Category: "Performance", Subcategory: "IRR", ReturnType: "percent"},
	{Name: "cash_on_cash_return", Title: "Cash-on-Cash Return", SQL: "SUM(annual_cash_flow) / SUM(total_cash_invested)", Description: "Measures the annual pre-tax cash flow as a percentage of total cash invested", Category: "Performance", Subcategory: "Ratios", ReturnType: "percent"},
	{Name: "equity_multiple", Title: "Equity Multiple", SQL: "SUM(total_distributions) / SUM(total_invested)", Description: "Measures the total cash returned relative to total cash invested", Category: "Private Markets", Subcategory: "Multiples", ReturnType: "number"},
	{Name: "sharpe_ratio_calc", Title: "Sharpe Ratio (Calc)", SQL: "({{ avg_return }} - {{ risk_free_rate }}) / {{ std_dev }}", Description: "Measures risk-adjusted return", Category: "Risk", Subcategory: "Volatility", ReturnType: "number"},
	{Name: "loss_ratio", Title: "Loss Ratio", SQL: "SUM(claim_amount) / SUM(premium_amount)", Description: "Claims paid out as % of premiums earned", Category: "Insurance", Subcategory: "Profitability", ReturnType: "percent"},
	{Name: "combined_ratio_calc", Title: "Combined Ratio", SQL: "{{ sum_of_ratios(SUM(claim_amount), SUM(premium_amount), SUM(expenses), SUM(premium_amount)) }}", Description: "Combined ratio (Loss Ratio + Expense Ratio)", Category: "Insurance", Subcategory: "Profitability", ReturnType: "percent"},
	{Name: "claims_reserve_adequacy", Title: "Claims Reserve Adequacy", SQL: "SUM(reserve_amount) / SUM(outstanding_claims)", Description: "Measures sufficiency of funds for future claims", Category: "Insurance", Subcategory: "Risk", ReturnType: "percent"},
	{Name: "loan_to_value_ratio", Title: "Loan-to-Value (LTV)", SQL: "SUM(outstanding_balance) / SUM(appraised_value)", Description: "Lending risk ratio", Category: "Banking", Subcategory: "Risk", ReturnType: "percent"},
	{Name: "net_interest_margin", Title: "Net Interest Margin (NIM)", SQL: "(SUM(interest_income) - SUM(interest_expense)) / AVG(earning_assets)", Description: "Difference between interest income and expense", Category: "Banking", Subcategory: "Profitability", ReturnType: "percent"},
	{Name: "capital_adequacy_ratio", Title: "Capital Adequacy Ratio (CAR)", SQL: "(SUM(tier1_capital) + SUM(tier2_capital)) / SUM(risk_weighted_assets)", Description: "Capital in relation to risk-weighted assets", Category: "Banking", Subcategory: "Regulatory", ReturnType: "percent"},
	{Name: "portfolio_volatility_calc", Title: "Portfolio Volatility (Markowitz)", SQL: "{{ portfolio_volatility(ARRAY_AGG(weights), ARRAY_AGG(volatilities), CORR_MATRIX(asset_id)) }}", Description: "Calculates portfolio volatility using covariance matrix", Category: "Quant Finance", Subcategory: "Portfolio Analytics", ReturnType: "percent"},
	{Name: "tracking_error_calc", Title: "Tracking Error", SQL: "{{ tracking_error(ARRAY_AGG(asset_return), ARRAY_AGG(benchmark_return)) }}", Description: "Measures active risk vs benchmark", Category: "Quant Finance", Subcategory: "Portfolio Analytics", ReturnType: "percent"},
	{Name: "black_scholes_calc", Title: "Black-Scholes Option Price", SQL: "{{ black_scholes('call', 100, 105, 0.5, 0.02, 0.25) }}", Description: "Calculates price and Greeks of European option", Category: "Quant Finance", Subcategory: "Derivatives Pricing", ReturnType: "currency"},
	{Name: "credit_var_calc", Title: "Credit VaR", SQL: "{{ credit_var(0.99, ARRAY_AGG(exposure), ARRAY_AGG(pd), ARRAY_AGG(lgd)) }}", Description: "Estimates potential loss on credit portfolio", Category: "Risk", Subcategory: "Credit Risk", ReturnType: "currency"},
	{Name: "quant_market_cvar", Title: "Conditional VaR (CVaR)", SQL: "AVG(returns[returns < PERCENTILE(returns, 1 - confidence_level)])", Description: "Expected shortfall beyond VaR", Category: "Quant Finance", Subcategory: "Market Risk", ReturnType: "currency"},
	{Name: "insurance_loss_ratio", Title: "Loss Ratio (Ins)", SQL: "SUM(claim_amount) / SUM(premium_amount)", Description: "Underwriting profitability", Category: "Insurance", Subcategory: "Underwriting", ReturnType: "percent"},
	{Name: "insurance_combined_ratio", Title: "Combined Ratio (Ins)", SQL: "(SUM(claim_amount) + SUM(expenses)) / SUM(premium_amount)", Description: "Losses + Expenses / Premiums", Category: "Insurance", Subcategory: "Underwriting", ReturnType: "percent"},
	{Name: "private_markets_tvpi", Title: "TVPI", SQL: "(SUM(cumulative_distributions) + SUM(remaining_value)) / SUM(paid_in_capital)", Description: "Total Value to Paid-In", Category: "Private Markets", Subcategory: "Performance", ReturnType: "number"},
	{Name: "private_markets_nav", Title: "Net Asset Value (NAV)", SQL: "SUM(current_fair_value)", Description: "Net asset value of portfolio holdings", Category: "Private Markets", Subcategory: "Valuation", ReturnType: "currency"},
	{Name: "excel_xirr", Title: "Excel XIRR", SQL: "{{ excel_formula('=XIRR({cash_flows}, {dates})') }}", Description: "XIRR using Excel function", Category: "Private Markets", Subcategory: "IRR", ReturnType: "percent"},
	{Name: "excel_npv", Title: "Excel NPV", SQL: "{{ excel_formula('=NPV({rate}, {cash_flows})') }}", Description: "NPV using Excel function", Category: "Performance", Subcategory: "Valuation", ReturnType: "currency"},
}

func main() {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		panic("DATABASE_URL environment variable is required")
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}

	log.Println("Connected to DB successfully.")

	// 2. Get Master / Gold Copy Tenant ID
	var tenantID string
	err = db.QueryRow("SELECT id FROM tenants WHERE gold_copy = true OR is_core = true OR name = 'core' LIMIT 1").Scan(&tenantID)
	if err != nil {
		err = db.QueryRow("SELECT id FROM tenants ORDER BY created_at ASC LIMIT 1").Scan(&tenantID)
		if err != nil {
			log.Fatalf("Fatal: No tenant found: %v", err)
		}
	}
	log.Printf("Using Master Tenant ID: %s", tenantID)

	// 3. Get Node Type ID for 'semantic_term'
	var nodeTypeID string
	err = db.QueryRow("SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term' LIMIT 1").Scan(&nodeTypeID)
	if err != nil {
		log.Fatalf("Fatal: Could not find node type 'semantic_term': %v", err)
	}
	log.Printf("Using Node Type ID (semantic_term): %s", nodeTypeID)

	// 4. Iterate and Upsert
	log.Println("Seeding Semantic Terms...")

	for _, item := range library {
		// Map ReturnType to DataType
		dataType := "number"
		if strings.Contains(strings.ToLower(item.ReturnType), "string") {
			dataType = "string"
		} else if strings.Contains(strings.ToLower(item.ReturnType), "bool") {
			dataType = "boolean"
		}

		props := map[string]interface{}{
			"type":         "calculated",
			"term_type":    "calculated",
			"data_type":    dataType,
			"expression":   item.SQL,
			"formula":      item.SQL,
			"display_name": item.Title,
			"tags":         []string{item.Category, item.Subcategory},
			"return_type":  item.ReturnType,
		}

		if item.FinancialCalc != nil {
			for k, v := range item.FinancialCalc {
				props[k] = v
			}
		}

		propsJson, _ := json.Marshal(props)

		// Generate ID
		namespace := uuid.NameSpaceURL
		nodeID := uuid.NewSHA1(namespace, []byte("node:"+item.Name)).String()
		qualifiedPath := "semantic_term/" + item.Category + "." + item.Name

		query := `
			INSERT INTO catalog_node (
				id, tenant_id, node_name, description, node_type_id, properties, qualified_path, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, NOW(), NOW()
			)
			ON CONFLICT (id) 
            DO UPDATE SET
                node_name = EXCLUDED.node_name,
				description = EXCLUDED.description,
				properties = EXCLUDED.properties,
				qualified_path = EXCLUDED.qualified_path,
				updated_at = NOW()
		`

		_, err := db.Exec(query,
			nodeID, tenantID, item.Title, item.Description, nodeTypeID, propsJson, qualifiedPath,
		)

		if err != nil {
			log.Printf("Error inserting %s: %v", item.Name, err)
		} else {
			log.Printf("Upserted %s [%s]", item.Title, item.Name)
		}
	}
	log.Println("Seeding complete.")
}
