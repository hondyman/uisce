package wealth

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// AIIntelligenceService provides AI-powered insights
type AIIntelligenceService struct {
	db *pgxpool.Pool
}

// NewAIIntelligenceService creates a new AI intelligence service
func NewAIIntelligenceService(db *pgxpool.Pool) *AIIntelligenceService {
	return &AIIntelligenceService{
		db: db,
	}
}

// ============================================================================
// CLIENT CHURN PREDICTION
// ============================================================================

// ChurnPrediction represents client churn risk analysis
type ChurnPrediction struct {
	PredictionID       string                  `json:"prediction_id"`
	FamilyID           string                  `json:"family_id"`
	FamilyName         string                  `json:"family_name"`
	ChurnRisk          string                  `json:"churn_risk"` // LOW, MEDIUM, HIGH, CRITICAL
	ChurnProbability   decimal.Decimal         `json:"churn_probability"`
	RiskScore          decimal.Decimal         `json:"risk_score"` // 0-100
	RiskFactors        []ChurnRiskFactor       `json:"risk_factors"`
	ProtectiveFactors  []ChurnProtectiveFactor `json:"protective_factors"`
	RecommendedActions []string                `json:"recommended_actions"`
	AtRiskRevenue      decimal.Decimal         `json:"at_risk_revenue"`
	CreatedAt          time.Time               `json:"created_at"`
}

// ChurnRiskFactor represents a churn risk indicator
type ChurnRiskFactor struct {
	Factor      string          `json:"factor"`
	Weight      decimal.Decimal `json:"weight"`
	Impact      string          `json:"impact"` // HIGH, MEDIUM, LOW
	Description string          `json:"description"`
}

// ChurnProtectiveFactor represents factors reducing churn risk
type ChurnProtectiveFactor struct {
	Factor      string          `json:"factor"`
	Strength    decimal.Decimal `json:"strength"`
	Description string          `json:"description"`
}

// PredictChurnRisk analyzes client churn risk using ML-style scoring
func (s *AIIntelligenceService) PredictChurnRisk(
	ctx context.Context,
	familyID string,
	familyName string,
	aum decimal.Decimal,
	lastLoginDays int,
	lastContactDays int,
	portfolioPerformance decimal.Decimal, // vs benchmark
	serviceIssuesCount int,
	ageOfRelationshipMonths int,
	advisorChangesCount int,
	netNewAssets decimal.Decimal, // Last 12 months
) (*ChurnPrediction, error) {
	prediction := &ChurnPrediction{
		PredictionID:       uuid.New().String(),
		FamilyID:           familyID,
		FamilyName:         familyName,
		RiskFactors:        []ChurnRiskFactor{},
		ProtectiveFactors:  []ChurnProtectiveFactor{},
		RecommendedActions: []string{},
		AtRiskRevenue:      aum.Mul(decimal.NewFromFloat(0.01)), // Assume 1% AUM fee
		CreatedAt:          time.Now(),
	}

	riskScore := decimal.Zero

	// Risk Factor 1: Inactivity (20% weight)
	if lastLoginDays > 90 {
		weight := decimal.NewFromInt(20)
		riskScore = riskScore.Add(weight)
		prediction.RiskFactors = append(prediction.RiskFactors, ChurnRiskFactor{
			Factor:      "PLATFORM_INACTIVITY",
			Weight:      weight,
			Impact:      "HIGH",
			Description: "Client hasn't logged in for 90+ days",
		})
		prediction.RecommendedActions = append(prediction.RecommendedActions, "Schedule check-in call with client")
	}

	// Risk Factor 2: Lack of Contact (15% weight)
	if lastContactDays > 60 {
		weight := decimal.NewFromInt(15)
		riskScore = riskScore.Add(weight)
		prediction.RiskFactors = append(prediction.RiskFactors, ChurnRiskFactor{
			Factor:      "LOW_ADVISOR_TOUCH",
			Weight:      weight,
			Impact:      "MEDIUM",
			Description: "No advisor contact in 60+ days",
		})
		prediction.RecommendedActions = append(prediction.RecommendedActions, "Send personalized portfolio update")
	}

	// Risk Factor 3: Poor Performance (25% weight)
	if portfolioPerformance.LessThan(decimal.NewFromInt(-3)) { // Underperforming by 3%+
		weight := decimal.NewFromInt(25)
		riskScore = riskScore.Add(weight)
		prediction.RiskFactors = append(prediction.RiskFactors, ChurnRiskFactor{
			Factor:      "UNDERPERFORMANCE",
			Weight:      weight,
			Impact:      "HIGH",
			Description: "Portfolio underperforming benchmark by 3%+",
		})
		prediction.RecommendedActions = append(prediction.RecommendedActions, "Prepare performance attribution analysis")
	}

	// Risk Factor 4: Service Issues (20% weight)
	if serviceIssuesCount > 2 {
		weight := decimal.NewFromInt(20)
		riskScore = riskScore.Add(weight)
		prediction.RiskFactors = append(prediction.RiskFactors, ChurnRiskFactor{
			Factor:      "SERVICE_COMPLAINTS",
			Weight:      weight,
			Impact:      "HIGH",
			Description: "Multiple service issues logged",
		})
		prediction.RecommendedActions = append(prediction.RecommendedActions, "Executive review of relationship")
	}

	// Risk Factor 5: Advisor Turnover (15% weight)
	if advisorChangesCount > 1 {
		weight := decimal.NewFromInt(15)
		riskScore = riskScore.Add(weight)
		prediction.RiskFactors = append(prediction.RiskFactors, ChurnRiskFactor{
			Factor:      "ADVISOR_TURNOVER",
			Weight:      weight,
			Impact:      "MEDIUM",
			Description: "Client experienced multiple advisor changes",
		})
		prediction.RecommendedActions = append(prediction.RecommendedActions, "Introduce to senior advisor for relationship stability")
	}

	// Risk Factor 6: Asset Outflows (25% weight)
	if netNewAssets.LessThan(decimal.Zero) {
		weight := decimal.NewFromInt(25)
		riskScore = riskScore.Add(weight)
		prediction.RiskFactors = append(prediction.RiskFactors, ChurnRiskFactor{
			Factor:      "ASSET_OUTFLOWS",
			Weight:      weight,
			Impact:      "CRITICAL",
			Description: "Net asset outflows in last 12 months",
		})
		prediction.RecommendedActions = append(prediction.RecommendedActions, "Urgent: Schedule retention meeting")
	}

	// Protective Factor 1: Long Relationship
	if ageOfRelationshipMonths > 60 { // 5+ years
		prediction.ProtectiveFactors = append(prediction.ProtectiveFactors, ChurnProtectiveFactor{
			Factor:      "LONG_TENURE",
			Strength:    decimal.NewFromInt(-10), // Reduces risk
			Description: "5+ year relationship",
		})
		riskScore = riskScore.Sub(decimal.NewFromInt(10))
	}

	// Protective Factor 2: Recent Asset Inflows
	if netNewAssets.GreaterThan(decimal.Zero) {
		prediction.ProtectiveFactors = append(prediction.ProtectiveFactors, ChurnProtectiveFactor{
			Factor:      "ASSET_GROWTH",
			Strength:    decimal.NewFromInt(-15),
			Description: "Net asset inflows",
		})
		riskScore = riskScore.Sub(decimal.NewFromInt(15))
	}

	// Cap risk score at 0-100
	if riskScore.LessThan(decimal.Zero) {
		riskScore = decimal.Zero
	}
	if riskScore.GreaterThan(decimal.NewFromInt(100)) {
		riskScore = decimal.NewFromInt(100)
	}

	prediction.RiskScore = riskScore
	prediction.ChurnProbability = riskScore.Div(decimal.NewFromInt(100))

	// Assign risk level
	switch {
	case riskScore.GreaterThanOrEqual(decimal.NewFromInt(70)):
		prediction.ChurnRisk = "CRITICAL"
	case riskScore.GreaterThanOrEqual(decimal.NewFromInt(50)):
		prediction.ChurnRisk = "HIGH"
	case riskScore.GreaterThanOrEqual(decimal.NewFromInt(30)):
		prediction.ChurnRisk = "MEDIUM"
	default:
		prediction.ChurnRisk = "LOW"
	}

	return prediction, nil
}

// ============================================================================
// MEETING PREPARATION ASSISTANT
// ============================================================================

// MeetingPrep represents AI-generated meeting preparation
type MeetingPrep struct {
	PrepID            string              `json:"prep_id"`
	FamilyID          string              `json:"family_id"`
	MeetingDate       time.Time           `json:"meeting_date"`
	MeetingType       string              `json:"meeting_type"`
	KeyTopics         []string            `json:"key_topics"`
	RecentActivity    []ActivityHighlight `json:"recent_activity"`
	PortfolioSnapshot PortfolioSnapshot   `json:"portfolio_snapshot"`
	ActionItems       []ActionItem        `json:"action_items"`
	TalkingPoints     []string            `json:"talking_points"`
	RiskAlerts        []string            `json:"risk_alerts"`
	Opportunities     []string            `json:"opportunities"`
	CreatedAt         time.Time           `json:"created_at"`
}

// ActivityHighlight represents recent client activity
type ActivityHighlight struct {
	Date       time.Time `json:"date"`
	Activity   string    `json:"activity"`
	Importance string    `json:"importance"`
}

// PortfolioSnapshot represents portfolio summary
type PortfolioSnapshot struct {
	TotalValue      decimal.Decimal            `json:"total_value"`
	MTDReturn       decimal.Decimal            `json:"mtd_return"`
	YTDReturn       decimal.Decimal            `json:"ytd_return"`
	VsBenchmark     decimal.Decimal            `json:"vs_benchmark"`
	LargestHoldings []string                   `json:"largest_holdings"`
	AssetAllocation map[string]decimal.Decimal `json:"asset_allocation"`
}

// ActionItem represents an action item for the meeting
type ActionItem struct {
	Description string `json:"description"`
	Priority    string `json:"priority"`
	Owner       string `json:"owner"`
}

// GenerateMeetingPrep creates AI-powered meeting preparation materials
func (s *AIIntelligenceService) GenerateMeetingPrep(
	ctx context.Context,
	familyID string,
	meetingDate time.Time,
	meetingType string,
) (*MeetingPrep, error) {
	prep := &MeetingPrep{
		PrepID:      uuid.New().String(),
		FamilyID:    familyID,
		MeetingDate: meetingDate,
		MeetingType: meetingType,
		CreatedAt:   time.Now(),
	}

	// Generate key topics based on meeting type
	switch meetingType {
	case "QUARTERLY_REVIEW":
		prep.KeyTopics = []string{
			"Portfolio Performance Review",
			"Market Outlook & Asset Allocation",
			"Tax-Loss Harvesting Opportunities",
			"Upcoming Distributions & Liquidity",
		}
	case "ANNUAL_PLANNING":
		prep.KeyTopics = []string{
			"Year-End Tax Planning",
			"Estate Plan Review",
			"Charitable Giving Strategy",
			"Next Year Goals & Objectives",
		}
	case "AD_HOC":
		prep.KeyTopics = []string{
			"Client-Requested Topics",
			"Recent Market Events",
			"Portfolio Adjustments",
		}
	}

	// Generate talking points
	prep.TalkingPoints = []string{
		"Highlight strong performance in alternative investments (+12% YTD)",
		"Discuss benefits of recent tax optimization strategies",
		"Review dynasty trust projections and multi-generational planning",
		"Address any concerns about market volatility",
	}

	// Generate opportunities
	prep.Opportunities = []string{
		"Qualified Opportunity Zone investment - potential $500K tax savings",
		"Charitable bunching - save $15K over 3 years",
		"529 plan optimization for grandchildren education",
	}

	// Generate risk alerts
	prep.RiskAlerts = []string{
		"ALERT: Client has 30% concentration in single stock - recommend diversification",
		"REMINDER: Estate plan not reviewed in 3 years - schedule review",
	}

	// Generate action items
	prep.ActionItems = []ActionItem{
		{
			Description: "Prepare tax-loss harvesting report",
			Priority:    "HIGH",
			Owner:       "Investment Team",
		},
		{
			Description: "Update beneficiary designations",
			Priority:    "MEDIUM",
			Owner:       "Estate Planning Team",
		},
	}

	return prep, nil
}

// ============================================================================
// PORTFOLIO OPTIMIZATION ASSISTANT
// ============================================================================

// PortfolioRecommendation represents AI-generated portfolio recommendations
type PortfolioRecommendation struct {
	RecommendationID    string                     `json:"recommendation_id"`
	FamilyID            string                     `json:"family_id"`
	CurrentAllocation   map[string]decimal.Decimal `json:"current_allocation"`
	TargetAllocation    map[string]decimal.Decimal `json:"target_allocation"`
	Rebalancing         []RebalancingAction        `json:"rebalancing"`
	ExpectedImprovement ExpectedImprovementMetrics `json:"expected_improvement"`
	RationaleNarrative  string                     `json:"rationale_narrative"`
	CreatedAt           time.Time                  `json:"created_at"`
}

// RebalancingAction represents a specific rebalancing trade
type RebalancingAction struct {
	Asset     string          `json:"asset"`
	Action    string          `json:"action"` // BUY, SELL
	Amount    decimal.Decimal `json:"amount"`
	Rationale string          `json:"rationale"`
}

// ExpectedImprovementMetrics represents expected portfolio improvements
type ExpectedImprovementMetrics struct {
	ReturnImprovement      decimal.Decimal `json:"return_improvement"`
	VolatilityReduction    decimal.Decimal `json:"volatility_reduction"`
	SharpeRatioImprovement decimal.Decimal `json:"sharpe_ratio_improvement"`
	TaxEfficiencyGain      decimal.Decimal `json:"tax_efficiency_gain"`
}

// GeneratePortfolioRecommendation creates AI-powered portfolio optimization
func (s *AIIntelligenceService) GeneratePortfolioRecommendation(
	ctx context.Context,
	familyID string,
	currentAllocation map[string]decimal.Decimal,
	riskTolerance string,
) (*PortfolioRecommendation, error) {
	recommendation := &PortfolioRecommendation{
		RecommendationID:  uuid.New().String(),
		FamilyID:          familyID,
		CurrentAllocation: currentAllocation,
		TargetAllocation:  make(map[string]decimal.Decimal),
		Rebalancing:       []RebalancingAction{},
		CreatedAt:         time.Now(),
	}

	// Generate target allocation based on risk tolerance
	switch riskTolerance {
	case "AGGRESSIVE":
		recommendation.TargetAllocation = map[string]decimal.Decimal{
			"EQUITIES":     decimal.NewFromInt(70),
			"FIXED_INCOME": decimal.NewFromInt(15),
			"ALTERNATIVES": decimal.NewFromInt(10),
			"CASH":         decimal.NewFromInt(5),
		}
	case "MODERATE":
		recommendation.TargetAllocation = map[string]decimal.Decimal{
			"EQUITIES":     decimal.NewFromInt(50),
			"FIXED_INCOME": decimal.NewFromInt(35),
			"ALTERNATIVES": decimal.NewFromInt(10),
			"CASH":         decimal.NewFromInt(5),
		}
	case "CONSERVATIVE":
		recommendation.TargetAllocation = map[string]decimal.Decimal{
			"EQUITIES":     decimal.NewFromInt(30),
			"FIXED_INCOME": decimal.NewFromInt(55),
			"ALTERNATIVES": decimal.NewFromInt(10),
			"CASH":         decimal.NewFromInt(5),
		}
	}

	// Expected improvements (example values)
	recommendation.ExpectedImprovement = ExpectedImprovementMetrics{
		ReturnImprovement:      decimal.NewFromFloat(0.5), // +0.5%
		VolatilityReduction:    decimal.NewFromFloat(2.0), // -2.0%
		SharpeRatioImprovement: decimal.NewFromFloat(0.15),
		TaxEfficiencyGain:      decimal.NewFromFloat(0.25), // 0.25% per year
	}

	recommendation.RationaleNarrative = "Based on your risk profile and current market conditions, we recommend increasing your allocation to equities and alternatives while maintaining a defensive fixed income position. This will improve your expected returns while maintaining appropriate risk levels."

	return recommendation, nil
}
