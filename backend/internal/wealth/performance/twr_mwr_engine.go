// Package performance provides world-class performance calculation
// including Time-Weighted Return (TWR), Money-Weighted Return (MWR),
// Modified Dietz, and contribution analysis.
package performance

import (
"context"
"encoding/json"
"fmt"
"math"
"sort"
"sync"
"time"

"github.com/google/uuid"
"github.com/jmoiron/sqlx"
)

// ReturnMethod defines the performance calculation method
type ReturnMethod string

const (
TWR           ReturnMethod = "twr"
MWR           ReturnMethod = "mwr"
ModifiedDietz ReturnMethod = "modified_dietz"
LinkedTWR     ReturnMethod = "linked_twr"
TrueTWR       ReturnMethod = "true_twr"
)

// PerformanceConfig configures performance calculation
type PerformanceConfig struct {
	Method            ReturnMethod `json:"method"`
	StartDate         time.Time    `json:"start_date"`
	EndDate           time.Time    `json:"end_date"`
	Frequency         string       `json:"frequency"`
	IncludeFees       bool         `json:"include_fees"`
	AnnualizationDays int          `json:"annualization_days"`
	BenchmarkID       string       `json:"benchmark_id,omitempty"`
	Currency          string       `json:"currency"`
	GeometricLinking  bool         `json:"geometric_linking"`
}

// CashFlow represents a portfolio cash flow
type CashFlow struct {
	ID          uuid.UUID `json:"id" db:"id"`
	PortfolioID string    `json:"portfolio_id" db:"portfolio_id"`
	Date        time.Time `json:"date" db:"date"`
	Amount      float64   `json:"amount" db:"amount"`
	Type        string    `json:"type" db:"type"`
	SecurityID  *string   `json:"security_id,omitempty" db:"security_id"`
	Description string    `json:"description" db:"description"`
	IsExternal  bool      `json:"is_external" db:"is_external"`
	Weight      float64   `json:"weight"`
}

// Valuation represents a portfolio valuation at a point in time
type Valuation struct {
	ID            uuid.UUID               `json:"id" db:"id"`
	PortfolioID   string                  `json:"portfolio_id" db:"portfolio_id"`
	Date          time.Time               `json:"date" db:"date"`
	MarketValue   float64                 `json:"market_value" db:"market_value"`
	AccruedIncome float64                 `json:"accrued_income" db:"accrued_income"`
	TotalValue    float64                 `json:"total_value" db:"total_value"`
	Holdings      map[string]HoldingValue `json:"holdings,omitempty"`
}

// HoldingValue represents a single holding's value
type HoldingValue struct {
SecurityID     string  `json:"security_id"`
SecurityName   string  `json:"security_name"`
Quantity       float64 `json:"quantity"`
Price          float64 `json:"price"`
MarketValue    float64 `json:"market_value"`
CostBasis      float64 `json:"cost_basis"`
Weight         float64 `json:"weight"`
UnrealizedGain float64 `json:"unrealized_gain"`
}

// PeriodReturn represents return for a single period
type PeriodReturn struct {
PeriodStart      time.Time `json:"period_start"`
PeriodEnd        time.Time `json:"period_end"`
BeginningValue   float64   `json:"beginning_value"`
EndingValue      float64   `json:"ending_value"`
NetCashFlows     float64   `json:"net_cash_flows"`
GrossReturn      float64   `json:"gross_return"`
NetReturn        float64   `json:"net_return"`
Fees             float64   `json:"fees"`
Income           float64   `json:"income"`
CumulativeReturn float64   `json:"cumulative_return"`
TradingDays      int       `json:"trading_days"`
}

// ContributionAnalysis shows return contribution by segment
type ContributionAnalysis struct {
Segment        string  `json:"segment"`
Name           string  `json:"name"`
BeginWeight    float64 `json:"begin_weight"`
EndWeight      float64 `json:"end_weight"`
AverageWeight  float64 `json:"average_weight"`
Return         float64 `json:"return"`
Contribution   float64 `json:"contribution"`
PercentOfTotal float64 `json:"percent_of_total"`
}

// PerformanceResult contains complete performance analysis
type PerformanceResult struct {
ID                   uuid.UUID              `json:"id"`
PortfolioID          string                 `json:"portfolio_id"`
TenantID             string                 `json:"tenant_id"`
Config               PerformanceConfig      `json:"config"`
TotalReturn          float64                `json:"total_return"`
AnnualizedReturn     float64                `json:"annualized_return"`
CumulativeReturn     float64                `json:"cumulative_return"`
GrossReturn          float64                `json:"gross_return"`
NetReturn            float64                `json:"net_return"`
PeriodReturns        []PeriodReturn         `json:"period_returns"`
YTDReturn            float64                `json:"ytd_return"`
MTDReturn            float64                `json:"mtd_return"`
QTDReturn            float64                `json:"qtd_return"`
OneYearReturn        float64                `json:"one_year_return,omitempty"`
ThreeYearReturn      float64                `json:"three_year_return,omitempty"`
FiveYearReturn       float64                `json:"five_year_return,omitempty"`
SinceInceptionReturn float64                `json:"since_inception_return"`
Contributions        []ContributionAnalysis `json:"contributions,omitempty"`
TotalContributions   float64                `json:"total_contributions"`
TotalWithdrawals     float64                `json:"total_withdrawals"`
NetCashFlows         float64                `json:"net_cash_flows"`
BenchmarkReturn      float64                `json:"benchmark_return,omitempty"`
ExcessReturn         float64                `json:"excess_return,omitempty"`
GeneratedAt          time.Time              `json:"generated_at"`
CalculationMethod    string                 `json:"calculation_method"`
Metadata             map[string]interface{} `json:"metadata,omitempty"`
}

// PerformanceEngine provides performance calculation capabilities
type PerformanceEngine struct {
db    *sqlx.DB
cache sync.Map
}

// NewPerformanceEngine creates a new performance engine
func NewPerformanceEngine(db *sqlx.DB) *PerformanceEngine {
return &PerformanceEngine{db: db}
}

// Calculate performs performance calculation based on config
func (e *PerformanceEngine) Calculate(ctx context.Context, portfolioID, tenantID string, config PerformanceConfig) (*PerformanceResult, error) {
valuations, err := e.getValuations(ctx, portfolioID, tenantID, config.StartDate, config.EndDate)
if err != nil {
return nil, fmt.Errorf("failed to fetch valuations: %w", err)
}

cashFlows, err := e.getCashFlows(ctx, portfolioID, tenantID, config.StartDate, config.EndDate)
if err != nil {
return nil, fmt.Errorf("failed to fetch cash flows: %w", err)
}

var result *PerformanceResult
switch config.Method {
case TWR, LinkedTWR, TrueTWR:
result, err = e.calculateTWR(ctx, portfolioID, tenantID, valuations, cashFlows, config)
case MWR:
result, err = e.calculateMWR(ctx, portfolioID, tenantID, valuations, cashFlows, config)
case ModifiedDietz:
result, err = e.calculateModifiedDietz(ctx, portfolioID, tenantID, valuations, cashFlows, config)
default:
result, err = e.calculateTWR(ctx, portfolioID, tenantID, valuations, cashFlows, config)
}

if err != nil {
return nil, err
}

e.calculateSubPeriodReturns(result, valuations, cashFlows, config)

if len(valuations) > 0 && valuations[len(valuations)-1].Holdings != nil {
result.Contributions = e.calculateContributions(valuations, config)
}

if config.BenchmarkID != "" {
benchReturn, err := e.getBenchmarkReturn(ctx, config.BenchmarkID, config.StartDate, config.EndDate)
if err == nil {
result.BenchmarkReturn = benchReturn
result.ExcessReturn = result.TotalReturn - benchReturn
}
}

return result, nil
}

// calculateTWR calculates Time-Weighted Return using geometric linking
func (e *PerformanceEngine) calculateTWR(ctx context.Context, portfolioID, tenantID string,
valuations []Valuation, cashFlows []CashFlow, config PerformanceConfig) (*PerformanceResult, error) {

if len(valuations) < 2 {
return nil, fmt.Errorf("insufficient valuations for TWR calculation")
}

sort.Slice(valuations, func(i, j int) bool {
return valuations[i].Date.Before(valuations[j].Date)
})

periodReturns := make([]PeriodReturn, 0, len(valuations)-1)
cumulativeReturn := 1.0
var totalFees, totalContributions, totalWithdrawals float64

for i := 1; i < len(valuations); i++ {
prev := valuations[i-1]
curr := valuations[i]

periodCFs := e.getCashFlowsBetween(cashFlows, prev.Date, curr.Date)
netCF := 0.0
fees := 0.0
income := 0.0

for _, cf := range periodCFs {
netCF += cf.Amount
if cf.Type == "fee" {
fees += math.Abs(cf.Amount)
}
if cf.Type == "dividend" || cf.Type == "interest" {
income += cf.Amount
}
if cf.IsExternal {
if cf.Amount > 0 {
totalContributions += cf.Amount
} else {
totalWithdrawals += math.Abs(cf.Amount)
}
}
}
totalFees += fees

bmv := prev.TotalValue
emv := curr.TotalValue
totalDays := curr.Date.Sub(prev.Date).Hours() / 24
weightedCF := 0.0

for _, cf := range periodCFs {
daysRemaining := curr.Date.Sub(cf.Date).Hours() / 24
weight := daysRemaining / totalDays
weightedCF += cf.Amount * weight
}

var grossReturn float64
denominator := bmv + weightedCF
if denominator != 0 {
grossReturn = (emv - bmv - netCF) / denominator
}

netReturn := grossReturn
if config.IncludeFees && bmv > 0 {
netReturn = grossReturn - (fees / bmv)
}

cumulativeReturn *= (1 + netReturn)

periodReturns = append(periodReturns, PeriodReturn{
PeriodStart:      prev.Date,
PeriodEnd:        curr.Date,
BeginningValue:   bmv,
EndingValue:      emv,
NetCashFlows:     netCF,
GrossReturn:      grossReturn,
NetReturn:        netReturn,
Fees:             fees,
Income:           income,
CumulativeReturn: cumulativeReturn - 1,
TradingDays:      int(totalDays),
})
}

totalReturn := cumulativeReturn - 1
totalDays := config.EndDate.Sub(config.StartDate).Hours() / 24
annualizationDays := float64(config.AnnualizationDays)
if annualizationDays == 0 {
annualizationDays = 365
}
years := totalDays / annualizationDays
annualizedReturn := 0.0
if years > 0 {
annualizedReturn = math.Pow(cumulativeReturn, 1/years) - 1
}

grossReturn := 0.0
if len(periodReturns) > 0 {
grossCumulative := 1.0
for _, pr := range periodReturns {
grossCumulative *= (1 + pr.GrossReturn)
}
grossReturn = grossCumulative - 1
}

return &PerformanceResult{
ID:                 uuid.New(),
PortfolioID:        portfolioID,
TenantID:           tenantID,
Config:             config,
TotalReturn:        totalReturn,
AnnualizedReturn:   annualizedReturn,
CumulativeReturn:   cumulativeReturn - 1,
GrossReturn:        grossReturn,
NetReturn:          totalReturn,
PeriodReturns:      periodReturns,
TotalContributions: totalContributions,
TotalWithdrawals:   totalWithdrawals,
NetCashFlows:       totalContributions - totalWithdrawals,
GeneratedAt:        time.Now(),
CalculationMethod:  string(config.Method),
Metadata: map[string]interface{}{
"total_fees":    totalFees,
"periods_count": len(periodReturns),
},
}, nil
}

// calculateMWR calculates Money-Weighted Return (IRR)
func (e *PerformanceEngine) calculateMWR(ctx context.Context, portfolioID, tenantID string,
valuations []Valuation, cashFlows []CashFlow, config PerformanceConfig) (*PerformanceResult, error) {

if len(valuations) < 2 {
return nil, fmt.Errorf("insufficient valuations for MWR calculation")
}

sort.Slice(valuations, func(i, j int) bool {
return valuations[i].Date.Before(valuations[j].Date)
})

beginValue := valuations[0].TotalValue
endValue := valuations[len(valuations)-1].TotalValue
totalDays := config.EndDate.Sub(config.StartDate).Hours() / 24

irr := e.solveIRR(beginValue, endValue, cashFlows, config.StartDate, totalDays)

annualizationDays := float64(config.AnnualizationDays)
if annualizationDays == 0 {
annualizationDays = 365
}
periodsPerYear := annualizationDays / totalDays
annualizedReturn := math.Pow(1+irr, periodsPerYear) - 1

var totalContributions, totalWithdrawals float64
for _, cf := range cashFlows {
if cf.IsExternal {
if cf.Amount > 0 {
totalContributions += cf.Amount
} else {
totalWithdrawals += math.Abs(cf.Amount)
}
}
}

return &PerformanceResult{
ID:                 uuid.New(),
PortfolioID:        portfolioID,
TenantID:           tenantID,
Config:             config,
TotalReturn:        irr,
AnnualizedReturn:   annualizedReturn,
CumulativeReturn:   irr,
GrossReturn:        irr,
NetReturn:          irr,
TotalContributions: totalContributions,
TotalWithdrawals:   totalWithdrawals,
NetCashFlows:       totalContributions - totalWithdrawals,
GeneratedAt:        time.Now(),
CalculationMethod:  string(MWR),
Metadata: map[string]interface{}{
"begin_value": beginValue,
"end_value":   endValue,
"method":      "newton_raphson_irr",
},
}, nil
}

// solveIRR uses Newton-Raphson method to find IRR
func (e *PerformanceEngine) solveIRR(beginValue, endValue float64, cashFlows []CashFlow, startDate time.Time, totalDays float64) float64 {
rate := 0.10
maxIterations := 100
tolerance := 1e-10

for i := 0; i < maxIterations; i++ {
npv, dnpv := e.calculateNPVAndDerivative(beginValue, endValue, cashFlows, startDate, totalDays, rate)

if math.Abs(npv) < tolerance {
break
}

if dnpv == 0 {
rate += 0.01
continue
}

newRate := rate - npv/dnpv
if newRate < -0.99 {
newRate = -0.99
}
if newRate > 10 {
newRate = 10
}
rate = newRate
}

return rate
}

// calculateNPVAndDerivative calculates NPV and its derivative for Newton-Raphson
func (e *PerformanceEngine) calculateNPVAndDerivative(beginValue, endValue float64, cashFlows []CashFlow, startDate time.Time, totalDays float64, rate float64) (float64, float64) {
npv := beginValue
dnpv := 0.0

for _, cf := range cashFlows {
days := cf.Date.Sub(startDate).Hours() / 24
t := days / totalDays
discount := math.Pow(1+rate, t)
npv += cf.Amount / discount
dnpv += -t * cf.Amount / (discount * (1 + rate))
}

discount := math.Pow(1+rate, 1)
npv -= endValue / discount
dnpv -= -1 * endValue / (discount * (1 + rate))

return npv, dnpv
}

// calculateModifiedDietz calculates return using Modified Dietz method
func (e *PerformanceEngine) calculateModifiedDietz(ctx context.Context, portfolioID, tenantID string,
valuations []Valuation, cashFlows []CashFlow, config PerformanceConfig) (*PerformanceResult, error) {

if len(valuations) < 2 {
return nil, fmt.Errorf("insufficient valuations for Modified Dietz calculation")
}

sort.Slice(valuations, func(i, j int) bool {
return valuations[i].Date.Before(valuations[j].Date)
})

bmv := valuations[0].TotalValue
emv := valuations[len(valuations)-1].TotalValue
totalDays := config.EndDate.Sub(config.StartDate).Hours() / 24

weightedCF := 0.0
totalCF := 0.0
var totalContributions, totalWithdrawals float64

for _, cf := range cashFlows {
daysRemaining := config.EndDate.Sub(cf.Date).Hours() / 24
weight := daysRemaining / totalDays
weightedCF += cf.Amount * weight
totalCF += cf.Amount

if cf.IsExternal {
if cf.Amount > 0 {
totalContributions += cf.Amount
} else {
totalWithdrawals += math.Abs(cf.Amount)
}
}
}

denominator := bmv + weightedCF
modDietzReturn := 0.0
if denominator != 0 {
modDietzReturn = (emv - bmv - totalCF) / denominator
}

annualizationDays := float64(config.AnnualizationDays)
if annualizationDays == 0 {
annualizationDays = 365
}
years := totalDays / annualizationDays
annualizedReturn := 0.0
if years > 0 {
annualizedReturn = math.Pow(1+modDietzReturn, 1/years) - 1
}

return &PerformanceResult{
ID:                 uuid.New(),
PortfolioID:        portfolioID,
TenantID:           tenantID,
Config:             config,
TotalReturn:        modDietzReturn,
AnnualizedReturn:   annualizedReturn,
CumulativeReturn:   modDietzReturn,
GrossReturn:        modDietzReturn,
NetReturn:          modDietzReturn,
TotalContributions: totalContributions,
TotalWithdrawals:   totalWithdrawals,
NetCashFlows:       totalCF,
GeneratedAt:        time.Now(),
CalculationMethod:  string(ModifiedDietz),
Metadata: map[string]interface{}{
"begin_value": bmv,
"end_value":   emv,
"total_cf":    totalCF,
"weighted_cf": weightedCF,
},
}, nil
}

// calculateSubPeriodReturns calculates YTD, MTD, QTD, etc.
func (e *PerformanceEngine) calculateSubPeriodReturns(result *PerformanceResult, valuations []Valuation, cashFlows []CashFlow, config PerformanceConfig) {
now := config.EndDate

ytdStart := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
if ytdStart.After(config.StartDate) {
result.YTDReturn = e.calculateReturnForPeriod(valuations, cashFlows, ytdStart, now, config)
}

mtdStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
if mtdStart.After(config.StartDate) {
result.MTDReturn = e.calculateReturnForPeriod(valuations, cashFlows, mtdStart, now, config)
}

quarter := (int(now.Month()) - 1) / 3
qtdStart := time.Date(now.Year(), time.Month(quarter*3+1), 1, 0, 0, 0, 0, now.Location())
if qtdStart.After(config.StartDate) {
result.QTDReturn = e.calculateReturnForPeriod(valuations, cashFlows, qtdStart, now, config)
}

oneYearAgo := now.AddDate(-1, 0, 0)
if oneYearAgo.After(config.StartDate) || oneYearAgo.Equal(config.StartDate) {
result.OneYearReturn = e.calculateReturnForPeriod(valuations, cashFlows, oneYearAgo, now, config)
}

threeYearsAgo := now.AddDate(-3, 0, 0)
if threeYearsAgo.After(config.StartDate) || threeYearsAgo.Equal(config.StartDate) {
ret := e.calculateReturnForPeriod(valuations, cashFlows, threeYearsAgo, now, config)
result.ThreeYearReturn = math.Pow(1+ret, 1.0/3.0) - 1
}

fiveYearsAgo := now.AddDate(-5, 0, 0)
if fiveYearsAgo.After(config.StartDate) || fiveYearsAgo.Equal(config.StartDate) {
ret := e.calculateReturnForPeriod(valuations, cashFlows, fiveYearsAgo, now, config)
result.FiveYearReturn = math.Pow(1+ret, 1.0/5.0) - 1
}

result.SinceInceptionReturn = result.AnnualizedReturn
}

// calculateReturnForPeriod calculates return for a specific period
func (e *PerformanceEngine) calculateReturnForPeriod(valuations []Valuation, cashFlows []CashFlow, start, end time.Time, config PerformanceConfig) float64 {
periodVals := make([]Valuation, 0)
for _, v := range valuations {
if (v.Date.Equal(start) || v.Date.After(start)) && (v.Date.Equal(end) || v.Date.Before(end)) {
periodVals = append(periodVals, v)
}
}

if len(periodVals) < 2 {
return 0
}

periodCFs := e.getCashFlowsBetween(cashFlows, start, end)
bmv := periodVals[0].TotalValue
emv := periodVals[len(periodVals)-1].TotalValue
totalDays := end.Sub(start).Hours() / 24

weightedCF := 0.0
totalCF := 0.0
for _, cf := range periodCFs {
daysRemaining := end.Sub(cf.Date).Hours() / 24
weight := daysRemaining / totalDays
weightedCF += cf.Amount * weight
totalCF += cf.Amount
}

denominator := bmv + weightedCF
if denominator == 0 {
return 0
}

return (emv - bmv - totalCF) / denominator
}

// calculateContributions calculates return contribution by holding
func (e *PerformanceEngine) calculateContributions(valuations []Valuation, config PerformanceConfig) []ContributionAnalysis {
if len(valuations) < 2 {
return nil
}

beginHoldings := valuations[0].Holdings
endHoldings := valuations[len(valuations)-1].Holdings

if beginHoldings == nil || endHoldings == nil {
return nil
}

beginValue := valuations[0].TotalValue
endValue := valuations[len(valuations)-1].TotalValue
totalReturn := 0.0
if beginValue > 0 {
totalReturn = (endValue - beginValue) / beginValue
}

contributions := make([]ContributionAnalysis, 0)

allSecurities := make(map[string]bool)
for id := range beginHoldings {
allSecurities[id] = true
}
for id := range endHoldings {
allSecurities[id] = true
}

for secID := range allSecurities {
beginHolding := beginHoldings[secID]
endHolding := endHoldings[secID]

beginWeight := beginHolding.Weight
endWeight := endHolding.Weight
avgWeight := (beginWeight + endWeight) / 2

holdingReturn := 0.0
if beginHolding.MarketValue > 0 {
holdingReturn = (endHolding.MarketValue - beginHolding.MarketValue) / beginHolding.MarketValue
}

contribution := avgWeight * holdingReturn

name := beginHolding.SecurityName
if name == "" {
name = endHolding.SecurityName
}

contributions = append(contributions, ContributionAnalysis{
Segment:       "security",
Name:          name,
BeginWeight:   beginWeight,
EndWeight:     endWeight,
AverageWeight: avgWeight,
Return:        holdingReturn,
Contribution:  contribution,
})
}

for i := range contributions {
if totalReturn != 0 {
contributions[i].PercentOfTotal = contributions[i].Contribution / totalReturn * 100
}
}

sort.Slice(contributions, func(i, j int) bool {
return math.Abs(contributions[i].Contribution) > math.Abs(contributions[j].Contribution)
})

return contributions
}

// getCashFlowsBetween returns cash flows between two dates
func (e *PerformanceEngine) getCashFlowsBetween(cashFlows []CashFlow, start, end time.Time) []CashFlow {
result := make([]CashFlow, 0)
for _, cf := range cashFlows {
if cf.Date.After(start) && (cf.Date.Before(end) || cf.Date.Equal(end)) {
result = append(result, cf)
}
}
return result
}

// getValuations fetches valuations from database
func (e *PerformanceEngine) getValuations(ctx context.Context, portfolioID, tenantID string, startDate, endDate time.Time) ([]Valuation, error) {
query := `
SELECT id, portfolio_id, date, market_value, accrued_income, total_value
FROM portfolio_valuations
WHERE portfolio_id = $1 AND tenant_id = $2 AND date >= $3 AND date <= $4
ORDER BY date
`
var valuations []Valuation
err := e.db.SelectContext(ctx, &valuations, query, portfolioID, tenantID, startDate, endDate)
return valuations, err
}

// getCashFlows fetches cash flows from database
func (e *PerformanceEngine) getCashFlows(ctx context.Context, portfolioID, tenantID string, startDate, endDate time.Time) ([]CashFlow, error) {
query := `
SELECT id, portfolio_id, date, amount, type, security_id, description, is_external
FROM portfolio_cash_flows
WHERE portfolio_id = $1 AND tenant_id = $2 AND date >= $3 AND date <= $4
ORDER BY date
`
var cashFlows []CashFlow
err := e.db.SelectContext(ctx, &cashFlows, query, portfolioID, tenantID, startDate, endDate)
return cashFlows, err
}

// getBenchmarkReturn fetches benchmark return for period
func (e *PerformanceEngine) getBenchmarkReturn(ctx context.Context, benchmarkID string, startDate, endDate time.Time) (float64, error) {
query := `
SELECT COALESCE(
(SELECT total_return FROM benchmark_returns WHERE benchmark_id = $1 AND start_date = $2 AND end_date = $3),
0
)
`
var benchReturn float64
err := e.db.GetContext(ctx, &benchReturn, query, benchmarkID, startDate, endDate)
return benchReturn, err
}

// ToJSON marshals result to JSON
func (r *PerformanceResult) ToJSON() ([]byte, error) {
return json.Marshal(r)
}
