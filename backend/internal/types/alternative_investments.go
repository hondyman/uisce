package types

import (
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// ALTERNATIVE INVESTMENTS
// ============================================================================

// AlternativeInvestment represents an alternative investment (PE, VC, Hedge Funds, etc.)
type AlternativeInvestment struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	TenantID  uuid.UUID  `json:"tenantId" db:"tenant_id"`
	ClientID  uuid.UUID  `json:"clientId" db:"client_id"`
	AccountID *uuid.UUID `json:"accountId,omitempty" db:"account_id"`

	// Investment identification
	FundName    string  `json:"fundName" db:"fund_name"`
	FundManager string  `json:"fundManager" db:"fund_manager"`
	AssetClass  string  `json:"assetClass" db:"asset_class"`
	SubStrategy *string `json:"subStrategy,omitempty" db:"sub_strategy"`

	// Investment terms
	VintageYear        *int    `json:"vintageYear,omitempty" db:"vintage_year"`
	CommitmentAmount   float64 `json:"commitmentAmount" db:"commitment_amount"`
	CommitmentCurrency string  `json:"commitmentCurrency" db:"commitment_currency"`
	CapitalCalled      float64 `json:"capitalCalled" db:"capital_called"`
	CapitalDistributed float64 `json:"capitalDistributed" db:"capital_distributed"`
	UnfundedCommitment float64 `json:"unfundedCommitment" db:"unfunded_commitment"`

	// Valuation
	CurrentNAV        float64    `json:"currentNav" db:"current_nav"`
	LastValuationDate *time.Time `json:"lastValuationDate,omitempty" db:"last_valuation_date"`
	ValuationMethod   *string    `json:"valuationMethod,omitempty" db:"valuation_method"`

	// Fee structure
	ManagementFeePct  *float64 `json:"managementFeePct,omitempty" db:"management_fee_pct"`
	PerformanceFeePct *float64 `json:"performanceFeePct,omitempty" db:"performance_fee_pct"`
	HurdleRatePct     *float64 `json:"hurdleRatePct,omitempty" db:"hurdle_rate_pct"`
	HasHighWaterMark  bool     `json:"hasHighWaterMark" db:"has_high_water_mark"`
	HasCatchUp        bool     `json:"hasCatchUp" db:"has_catch_up"`

	// Tax and compliance
	TaxEntityType  *string    `json:"taxEntityType,omitempty" db:"tax_entity_type"`
	K1Received     bool       `json:"k1Received" db:"k1_received"`
	K1ReceivedDate *time.Time `json:"k1ReceivedDate,omitempty" db:"k1_received_date"`

	// Metadata
	InceptionDate     time.Time  `json:"inceptionDate" db:"inception_date"`
	ExpectedTermYears *int       `json:"expectedTermYears,omitempty" db:"expected_term_years"`
	MaturityDate      *time.Time `json:"maturityDate,omitempty" db:"maturity_date"`
	Notes             *string    `json:"notes,omitempty" db:"notes"`

	CreatedAt time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time  `json:"updatedAt" db:"updated_at"`
	DeletedAt *time.Time `json:"deletedAt,omitempty" db:"deleted_at"`
}

// ============================================================================
// CAPITAL CALLS
// ============================================================================

// CapitalCall represents a capital call notice from a GP
type CapitalCall struct {
	ID           uuid.UUID `json:"id" db:"id"`
	InvestmentID uuid.UUID `json:"investmentId" db:"investment_id"`

	// Call details
	CallNumber      int       `json:"callNumber" db:"call_number"`
	CallDate        time.Time `json:"callDate" db:"call_date"`
	DueDate         time.Time `json:"dueDate" db:"due_date"`
	AmountRequested float64   `json:"amountRequested" db:"amount_requested"`

	// Status
	Status       string     `json:"status" db:"status"`
	AmountFunded float64    `json:"amountFunded" db:"amount_funded"`
	FundedDate   *time.Time `json:"fundedDate,omitempty" db:"funded_date"`

	// Cash management
	LiquidityCheckStatus            *string    `json:"liquidityCheckStatus,omitempty" db:"liquidity_check_status"`
	RecommendedFundingSourceAccount *uuid.UUID `json:"recommendedFundingSourceAccount,omitempty" db:"recommended_funding_source_account_id"`
	AlertSent                       bool       `json:"alertSent" db:"alert_sent"`
	AlertSentAt                     *time.Time `json:"alertSentAt,omitempty" db:"alert_sent_at"`

	// Document reference
	NoticeDocumentID *uuid.UUID `json:"noticeDocumentId,omitempty" db:"notice_document_id"`

	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

// ============================================================================
// CAPITAL DISTRIBUTIONS
// ============================================================================

// CapitalDistribution represents a distribution from an alternative investment
type CapitalDistribution struct {
	ID           uuid.UUID `json:"id" db:"id"`
	InvestmentID uuid.UUID `json:"investmentId" db:"investment_id"`

	// Distribution details
	DistributionDate time.Time `json:"distributionDate" db:"distribution_date"`
	Amount           float64   `json:"amount" db:"amount"`
	DistributionType *string   `json:"distributionType,omitempty" db:"distribution_type"`
	IsRecallable     bool      `json:"isRecallable" db:"is_recallable"`

	// Tax breakdown
	OrdinaryIncome       float64 `json:"ordinaryIncome" db:"ordinary_income"`
	LongTermCapitalGain  float64 `json:"longTermCapitalGain" db:"long_term_capital_gain"`
	ShortTermCapitalGain float64 `json:"shortTermCapitalGain" db:"short_term_capital_gain"`
	ReturnOfCapital      float64 `json:"returnOfCapital" db:"return_of_capital"`

	// Document reference
	NoticeDocumentID *uuid.UUID `json:"noticeDocumentId,omitempty" db:"notice_document_id"`

	CreatedAt time.Time `json:"createdAt" db:"created_at"`
}

// ============================================================================
// PERFORMANCE METRICS
// ============================================================================

// AlternativeInvestmentPerformance represents calculated performance metrics
type AlternativeInvestmentPerformance struct {
	ID           uuid.UUID `json:"id" db:"id"`
	InvestmentID uuid.UUID `json:"investmentId" db:"investment_id"`

	// Calculation period
	AsOfDate time.Time `json:"asOfDate" db:"as_of_date"`

	// Core metrics
	IRRSinceInception *float64 `json:"irrSinceInception,omitempty" db:"irr_since_inception"`
	TVPI              *float64 `json:"tvpi,omitempty" db:"tvpi"`
	DPI               *float64 `json:"dpi,omitempty" db:"dpi"`
	RVPI              *float64 `json:"rvpi,omitempty" db:"rvpi"`
	MOIC              *float64 `json:"moic,omitempty" db:"moic"`

	// PME benchmarking
	PMEKaplanSchoar *float64 `json:"pmeKaplanSchoar,omitempty" db:"pme_kaplan_schoar"`
	PMEDirectAlpha  *float64 `json:"pmeDirectAlpha,omitempty" db:"pme_direct_alpha"`
	BenchmarkIndex  *string  `json:"benchmarkIndex,omitempty" db:"benchmark_index"`

	// J-curve
	JCurvePosition *string `json:"jCurvePosition,omitempty" db:"j_curve_position"`

	// Peer comparison
	PeerMedianIRR      *float64 `json:"peerMedianIrr,omitempty" db:"peer_median_irr"`
	PeerTopQuartileIRR *float64 `json:"peerTopQuartileIrr,omitempty" db:"peer_top_quartile_irr"`
	PercentileRank     *int     `json:"percentileRank,omitempty" db:"percentile_rank"`

	// Cash flows
	TotalCalled      *float64 `json:"totalCalled,omitempty" db:"total_called"`
	TotalDistributed *float64 `json:"totalDistributed,omitempty" db:"total_distributed"`
	NAVValue         *float64 `json:"navValue,omitempty" db:"nav_value"`

	CreatedAt time.Time `json:"createdAt" db:"created_at"`
}

// ============================================================================
// CAPITAL CALL FORECASTS
// ============================================================================

// CapitalCallForecast represents a predicted future capital call
type CapitalCallForecast struct {
	ID           uuid.UUID `json:"id" db:"id"`
	InvestmentID uuid.UUID `json:"investmentId" db:"investment_id"`

	// Forecast
	ForecastedCallDate time.Time `json:"forecastedCallDate" db:"forecasted_call_date"`
	EstimatedAmount    float64   `json:"estimatedAmount" db:"estimated_amount"`
	ConfidenceScore    *float64  `json:"confidenceScore,omitempty" db:"confidence_score"`

	// Model metadata
	ModelVersion *string   `json:"modelVersion,omitempty" db:"model_version"`
	ModelType    *string   `json:"modelType,omitempty" db:"model_type"`
	GeneratedAt  time.Time `json:"generatedAt" db:"generated_at"`

	// Alerts
	DaysNoticeBeforeDue int        `json:"daysNoticeBeforeDue" db:"days_notice_before_due"`
	AlertTriggered      bool       `json:"alertTriggered" db:"alert_triggered"`
	AlertTriggeredAt    *time.Time `json:"alertTriggeredAt,omitempty" db:"alert_triggered_at"`

	// Actual outcome
	ActualCallID          *uuid.UUID `json:"actualCallId,omitempty" db:"actual_call_id"`
	ForecastAccuracyScore *float64   `json:"forecastAccuracyScore,omitempty" db:"forecast_accuracy_score"`
}

// ============================================================================
// DOCUMENTS
// ============================================================================

// AlternativeInvestmentDocument represents a document related to an investment
type AlternativeInvestmentDocument struct {
	ID           uuid.UUID `json:"id" db:"id"`
	InvestmentID uuid.UUID `json:"investmentId" db:"investment_id"`

	// Document metadata
	DocumentType  string  `json:"documentType" db:"document_type"`
	FileName      string  `json:"fileName" db:"file_name"`
	FilePath      string  `json:"filePath" db:"file_path"`
	FileSizeBytes *int64  `json:"fileSizeBytes,omitempty" db:"file_size_bytes"`
	MimeType      *string `json:"mimeType,omitempty" db:"mime_type"`

	// Processing status
	ProcessingStatus string     `json:"processingStatus" db:"processing_status"`
	ProcessedAt      *time.Time `json:"processedAt,omitempty" db:"processed_at"`
	ProcessingError  *string    `json:"processingError,omitempty" db:"processing_error"`

	// Extracted data
	ExtractedData    map[string]interface{} `json:"extractedData,omitempty" db:"extracted_data"`
	ConfidenceScores map[string]interface{} `json:"confidenceScores,omitempty" db:"confidence_scores"`

	// Human review
	RequiresReview bool       `json:"requiresReview" db:"requires_review"`
	ReviewedBy     *uuid.UUID `json:"reviewedBy,omitempty" db:"reviewed_by"`
	ReviewedAt     *time.Time `json:"reviewedAt,omitempty" db:"reviewed_at"`
	ReviewNotes    *string    `json:"reviewNotes,omitempty" db:"review_notes"`
	ReviewStatus   *string    `json:"reviewStatus,omitempty" db:"review_status"`

	UploadedAt time.Time  `json:"uploadedAt" db:"uploaded_at"`
	UploadedBy *uuid.UUID `json:"uploadedBy,omitempty" db:"uploaded_by"`
}

// ============================================================================
// KPIs
// ============================================================================

// AlternativeInvestmentKPI represents asset-class-specific KPIs
type AlternativeInvestmentKPI struct {
	ID            uuid.UUID              `json:"id" db:"id"`
	InvestmentID  uuid.UUID              `json:"investmentId" db:"investment_id"`
	PeriodEndDate time.Time              `json:"periodEndDate" db:"period_end_date"`
	KPIs          map[string]interface{} `json:"kpis" db:"kpis"`
	CreatedAt     time.Time              `json:"createdAt" db:"created_at"`
}

// ============================================================================
// CASH FLOW (for IRR calculation)
// ============================================================================

// CashFlow represents a cash flow for IRR/NPV calculations
type CashFlow struct {
	Date   time.Time `json:"date"`
	Amount float64   `json:"amount"` // Negative = outflow (capital call), Positive = inflow (distribution)
}

// ============================================================================
// ALTERNATIVE FEE STRUCTURES
// ============================================================================

// AlternativeFeeStructure represents fee structure for alternative investments
type AlternativeFeeStructure struct {
	ID            uuid.UUID `json:"id" db:"id"`
	FeeScheduleID uuid.UUID `json:"feeScheduleId" db:"fee_schedule_id"`

	// Management fee
	ManagementFeePct   float64 `json:"managementFeePct" db:"management_fee_pct"`
	ManagementFeeBasis string  `json:"managementFeeBasis" db:"management_fee_basis"`

	// Performance fee
	PerformanceFeePct float64 `json:"performanceFeePct" db:"performance_fee_pct"`
	HurdleRatePct     float64 `json:"hurdleRatePct" db:"hurdle_rate_pct"`
	HurdleType        string  `json:"hurdleType" db:"hurdle_type"`

	// Fee variations
	HasCatchUp       bool     `json:"hasCatchUp" db:"has_catch_up"`
	CatchUpRate      *float64 `json:"catchUpRate,omitempty" db:"catch_up_rate"`
	HasHighWaterMark bool     `json:"hasHighWaterMark" db:"has_high_water_mark"`

	// Timing
	ManagementFeeFrequency string `json:"managementFeeFrequency" db:"management_fee_frequency"`
	PerformanceFeeTiming   string `json:"performanceFeeTiming" db:"performance_fee_timing"`

	// Clawback
	HasClawback         bool `json:"hasClawback" db:"has_clawback"`
	ClawbackPeriodYears *int `json:"clawbackPeriodYears,omitempty" db:"clawback_period_years"`

	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}
