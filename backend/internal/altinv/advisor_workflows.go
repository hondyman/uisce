package altinv

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// ============================================================================
// ADVISOR WORKFLOWS FOR ALTERNATIVE INVESTMENT ALLOCATIONS
// ============================================================================

// ============================================================================
// 1. DUE DILIGENCE PIPELINE WORKFLOW
// ============================================================================

// DueDiligencePipelineInput contains input for the due diligence workflow
type DueDiligencePipelineInput struct {
	OpportunityID uuid.UUID `json:"opportunity_id"`
	ClientID      uuid.UUID `json:"client_id"`
	AdvisorID     uuid.UUID `json:"advisor_id"`
}

// DueDiligencePipelineOutput contains the final result of the workflow
type DueDiligencePipelineOutput struct {
	OpportunityID      uuid.UUID              `json:"opportunity_id"`
	FinalStage         string                 `json:"final_stage"`
	Success            bool                   `json:"success"`
	ScreeningPassed    bool                   `json:"screening_passed"`
	AdvisorApproved    bool                   `json:"advisor_approved"`
	CommitteeDecision  string                 `json:"committee_decision"`
	ApprovedAmount     float64                `json:"approved_amount,omitempty"`
	FailureReason      string                 `json:"failure_reason,omitempty"`
	ReviewResults      *ParallelReviewResults `json:"review_results,omitempty"`
	DocumentsCompleted bool                   `json:"documents_completed"`
	TotalDurationDays  int                    `json:"total_duration_days"`
}

// ScreeningResult contains automated screening results
type ScreeningResult struct {
	Passed          bool     `json:"passed"`
	Score           float64  `json:"score"`
	Reasons         []string `json:"reasons"`
	Recommendations []string `json:"recommendations"`
}

// AdvisorDecision contains the advisor's review decision
type AdvisorDecision struct {
	Approved       bool      `json:"approved"`
	ApprovedBy     uuid.UUID `json:"approved_by"`
	ApprovedAt     time.Time `json:"approved_at"`
	Reason         string    `json:"reason,omitempty"`
	ModifiedAmount float64   `json:"modified_amount,omitempty"`
	Notes          string    `json:"notes,omitempty"`
}

// ParallelReviewResults contains results from parallel review activities
type ParallelReviewResults struct {
	RiskAssessment ReviewResult `json:"risk_assessment"`
	LegalReview    ReviewResult `json:"legal_review"`
	TaxAnalysis    ReviewResult `json:"tax_analysis"`
	OperationalDD  ReviewResult `json:"operational_dd"`
	ESGReview      ReviewResult `json:"esg_review,omitempty"`
}

// ReviewResult contains a single review outcome
type ReviewResult struct {
	Passed     bool      `json:"passed"`
	Score      float64   `json:"score"`
	RiskLevel  string    `json:"risk_level"` // LOW, MEDIUM, HIGH, CRITICAL
	Findings   []string  `json:"findings"`
	Conditions []string  `json:"conditions,omitempty"`
	ReviewedBy uuid.UUID `json:"reviewed_by,omitempty"`
	ReviewedAt time.Time `json:"reviewed_at,omitempty"`
}

// CommitteeDecisionResult contains investment committee decision
type CommitteeDecisionResult struct {
	Decision       string     `json:"decision"` // APPROVED, APPROVED_WITH_CONDITIONS, DEFERRED, REJECTED
	ApprovedAmount float64    `json:"approved_amount,omitempty"`
	Conditions     []string   `json:"conditions,omitempty"`
	VotesPassed    bool       `json:"votes_passed"`
	DecisionDate   time.Time  `json:"decision_date"`
	NextReviewDate *time.Time `json:"next_review_date,omitempty"`
}

// ESignatureStatus tracks document signing progress
type ESignatureStatus struct {
	AllSigned      bool       `json:"all_signed"`
	PendingSigners []string   `json:"pending_signers"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
}

// DueDiligencePipelineWorkflow orchestrates the complete due diligence process
func DueDiligencePipelineWorkflow(ctx workflow.Context, input DueDiligencePipelineInput) (*DueDiligencePipelineOutput, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting due diligence pipeline workflow",
		"opportunity_id", input.OpportunityID,
		"client_id", input.ClientID,
		"advisor_id", input.AdvisorID,
	)

	startTime := workflow.Now(ctx)
	output := &DueDiligencePipelineOutput{
		OpportunityID: input.OpportunityID,
	}

	// Configure activity options
	shortAO := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}

	longAO := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Minute,
		HeartbeatTimeout:    5 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    5 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    5 * time.Minute,
			MaximumAttempts:    2,
		},
	}

	shortCtx := workflow.WithActivityOptions(ctx, shortAO)
	longCtx := workflow.WithActivityOptions(ctx, longAO)

	// ========================================
	// Stage 1: Automated Initial Screening
	// ========================================
	logger.Info("Stage 1: Running automated screening")

	var screeningResult ScreeningResult
	err := workflow.ExecuteActivity(shortCtx, RunAutomatedScreeningActivity, input.OpportunityID).Get(ctx, &screeningResult)
	if err != nil {
		logger.Error("Automated screening failed", "error", err)
		output.FinalStage = "INTAKE"
		output.FailureReason = fmt.Sprintf("Screening activity failed: %v", err)
		return output, nil
	}

	output.ScreeningPassed = screeningResult.Passed

	if !screeningResult.Passed {
		logger.Info("Screening failed - notifying advisor",
			"score", screeningResult.Score,
			"reasons", screeningResult.Reasons,
		)

		// Notify advisor of screening failure
		err = workflow.ExecuteActivity(shortCtx, NotifyAdvisorScreeningFailedActivity,
			input.OpportunityID,
			input.AdvisorID,
			screeningResult,
		).Get(ctx, nil)
		if err != nil {
			logger.Warn("Failed to notify advisor", "error", err)
		}

		// Update opportunity stage to CLOSED_LOST
		err = workflow.ExecuteActivity(shortCtx, UpdateOpportunityStageActivity,
			input.OpportunityID,
			"CLOSED_LOST",
			"Automated screening failed: "+screeningResult.Reasons[0],
		).Get(ctx, nil)

		output.FinalStage = "CLOSED_LOST"
		output.FailureReason = "Screening failed: " + screeningResult.Reasons[0]
		output.TotalDurationDays = int(workflow.Now(ctx).Sub(startTime).Hours() / 24)
		return output, nil
	}

	logger.Info("Screening passed", "score", screeningResult.Score)

	// ========================================
	// Stage 2: Advisor Review (Human Gate)
	// ========================================
	logger.Info("Stage 2: Awaiting advisor review")

	// Notify advisor for review
	err = workflow.ExecuteActivity(shortCtx, NotifyAdvisorForReviewActivity,
		input.OpportunityID,
		input.AdvisorID,
	).Get(ctx, nil)
	if err != nil {
		logger.Warn("Failed to notify advisor for review", "error", err)
	}

	// Create advisor task
	err = workflow.ExecuteActivity(shortCtx, CreateAdvisorTaskActivity,
		input.AdvisorID,
		input.OpportunityID,
		"OPPORTUNITY_REVIEW",
		"Review investment opportunity after automated screening",
	).Get(ctx, nil)
	if err != nil {
		logger.Warn("Failed to create advisor task", "error", err)
	}

	// Wait for advisor approval (with timeout)
	var advisorDecision AdvisorDecision
	advisorApprovalSelector := workflow.NewSelector(ctx)

	// Signal channel for advisor decision
	advisorDecisionCh := workflow.GetSignalChannel(ctx, "advisor_decision")
	advisorApprovalSelector.AddReceive(advisorDecisionCh, func(c workflow.ReceiveChannel, more bool) {
		c.Receive(ctx, &advisorDecision)
	})

	// Timeout after 14 days
	advisorTimeout := workflow.NewTimer(ctx, 14*24*time.Hour)
	advisorApprovalSelector.AddFuture(advisorTimeout, func(f workflow.Future) {
		advisorDecision = AdvisorDecision{
			Approved: false,
			Reason:   "Advisor review timed out after 14 days",
		}
	})

	advisorApprovalSelector.Select(ctx)

	output.AdvisorApproved = advisorDecision.Approved

	if !advisorDecision.Approved {
		logger.Info("Advisor rejected opportunity", "reason", advisorDecision.Reason)

		err = workflow.ExecuteActivity(shortCtx, UpdateOpportunityStageActivity,
			input.OpportunityID,
			"CLOSED_LOST",
			advisorDecision.Reason,
		).Get(ctx, nil)

		output.FinalStage = "CLOSED_LOST"
		output.FailureReason = "Advisor rejected: " + advisorDecision.Reason
		output.TotalDurationDays = int(workflow.Now(ctx).Sub(startTime).Hours() / 24)
		return output, nil
	}

	logger.Info("Advisor approved - advancing to due diligence")

	// Update stage to DUE_DILIGENCE
	err = workflow.ExecuteActivity(shortCtx, UpdateOpportunityStageActivity,
		input.OpportunityID,
		"DUE_DILIGENCE",
		"Advisor approved for due diligence",
	).Get(ctx, nil)

	// ========================================
	// Stage 3: Parallel Due Diligence Reviews
	// ========================================
	logger.Info("Stage 3: Starting parallel due diligence reviews")

	// Initialize due diligence checklist
	err = workflow.ExecuteActivity(shortCtx, InitializeDueDiligenceChecklistActivity,
		input.OpportunityID,
	).Get(ctx, nil)
	if err != nil {
		logger.Warn("Failed to initialize DD checklist", "error", err)
	}

	// Execute parallel review activities
	var riskAssessment, legalReview, taxAnalysis, operationalDD ReviewResult

	riskFuture := workflow.ExecuteActivity(longCtx, RiskAssessmentActivity, input.OpportunityID)
	legalFuture := workflow.ExecuteActivity(longCtx, LegalComplianceReviewActivity, input.OpportunityID)
	taxFuture := workflow.ExecuteActivity(longCtx, TaxImpactAnalysisActivity, input.OpportunityID)
	operationalFuture := workflow.ExecuteActivity(longCtx, OperationalDueDiligenceActivity, input.OpportunityID)

	// Wait for all reviews to complete
	if err := riskFuture.Get(ctx, &riskAssessment); err != nil {
		logger.Error("Risk assessment failed", "error", err)
		riskAssessment = ReviewResult{Passed: false, RiskLevel: "CRITICAL", Findings: []string{err.Error()}}
	}

	if err := legalFuture.Get(ctx, &legalReview); err != nil {
		logger.Error("Legal review failed", "error", err)
		legalReview = ReviewResult{Passed: false, RiskLevel: "CRITICAL", Findings: []string{err.Error()}}
	}

	if err := taxFuture.Get(ctx, &taxAnalysis); err != nil {
		logger.Error("Tax analysis failed", "error", err)
		taxAnalysis = ReviewResult{Passed: false, RiskLevel: "HIGH", Findings: []string{err.Error()}}
	}

	if err := operationalFuture.Get(ctx, &operationalDD); err != nil {
		logger.Error("Operational DD failed", "error", err)
		operationalDD = ReviewResult{Passed: false, RiskLevel: "HIGH", Findings: []string{err.Error()}}
	}

	output.ReviewResults = &ParallelReviewResults{
		RiskAssessment: riskAssessment,
		LegalReview:    legalReview,
		TaxAnalysis:    taxAnalysis,
		OperationalDD:  operationalDD,
	}

	// Check if all reviews passed
	allPassed := riskAssessment.Passed && legalReview.Passed && taxAnalysis.Passed && operationalDD.Passed

	if !allPassed {
		logger.Info("One or more due diligence reviews failed")

		// Collect failures for notification
		failures := collectReviewFailures(output.ReviewResults)

		err = workflow.ExecuteActivity(shortCtx, NotifyCommitteeRejectionActivity,
			input.OpportunityID,
			failures,
		).Get(ctx, nil)

		// Allow opportunity to proceed to committee with conditions
		// (committee may still approve with conditions)
		logger.Info("Reviews had issues but proceeding to committee for decision")
	}

	// ========================================
	// Stage 4: Investment Committee Review
	// ========================================
	logger.Info("Stage 4: Investment committee review")

	// Update stage
	err = workflow.ExecuteActivity(shortCtx, UpdateOpportunityStageActivity,
		input.OpportunityID,
		"INVESTMENT_COMMITTEE",
		"Due diligence complete - submitted for committee review",
	).Get(ctx, nil)

	// Generate committee package
	err = workflow.ExecuteActivity(longCtx, GenerateCommitteePackageActivity,
		input.OpportunityID,
		output.ReviewResults,
	).Get(ctx, nil)
	if err != nil {
		logger.Warn("Failed to generate committee package", "error", err)
	}

	// Wait for committee decision (with timeout)
	var committeeDecision CommitteeDecisionResult
	committeeSelector := workflow.NewSelector(ctx)

	committeeDecisionCh := workflow.GetSignalChannel(ctx, "committee_decision")
	committeeSelector.AddReceive(committeeDecisionCh, func(c workflow.ReceiveChannel, more bool) {
		c.Receive(ctx, &committeeDecision)
	})

	// Timeout after 30 days for committee
	committeeTimeout := workflow.NewTimer(ctx, 30*24*time.Hour)
	committeeSelector.AddFuture(committeeTimeout, func(f workflow.Future) {
		committeeDecision = CommitteeDecisionResult{
			Decision: "DEFERRED",
		}
	})

	committeeSelector.Select(ctx)

	output.CommitteeDecision = committeeDecision.Decision

	if committeeDecision.Decision == "REJECTED" {
		logger.Info("Committee rejected opportunity")

		err = workflow.ExecuteActivity(shortCtx, UpdateOpportunityStageActivity,
			input.OpportunityID,
			"CLOSED_LOST",
			"Investment committee rejected",
		).Get(ctx, nil)

		output.FinalStage = "CLOSED_LOST"
		output.FailureReason = "Committee rejected"
		output.TotalDurationDays = int(workflow.Now(ctx).Sub(startTime).Hours() / 24)
		return output, nil
	}

	if committeeDecision.Decision == "DEFERRED" {
		logger.Info("Committee deferred decision")

		err = workflow.ExecuteActivity(shortCtx, UpdateOpportunityStageActivity,
			input.OpportunityID,
			"ON_HOLD",
			"Committee deferred decision",
		).Get(ctx, nil)

		output.FinalStage = "ON_HOLD"
		output.TotalDurationDays = int(workflow.Now(ctx).Sub(startTime).Hours() / 24)
		return output, nil
	}

	output.ApprovedAmount = committeeDecision.ApprovedAmount
	logger.Info("Committee approved", "amount", committeeDecision.ApprovedAmount)

	// ========================================
	// Stage 5: Documentation & E-Signature
	// ========================================
	logger.Info("Stage 5: Documentation and e-signature")

	err = workflow.ExecuteActivity(shortCtx, UpdateOpportunityStageActivity,
		input.OpportunityID,
		"DOCUMENTATION",
		"Approved - generating commitment documents",
	).Get(ctx, nil)

	// Generate commitment documents
	err = workflow.ExecuteActivity(longCtx, GenerateCommitmentDocumentsActivity,
		input.OpportunityID,
		committeeDecision.ApprovedAmount,
	).Get(ctx, nil)
	if err != nil {
		logger.Error("Failed to generate documents", "error", err)
	}

	// Initiate e-signature workflow
	var esigRequestID uuid.UUID
	err = workflow.ExecuteActivity(shortCtx, InitiateESignatureWorkflowActivity,
		input.OpportunityID,
		input.ClientID,
	).Get(ctx, &esigRequestID)
	if err != nil {
		logger.Error("Failed to initiate e-signature", "error", err)
	}

	// Wait for e-signature completion
	var esigStatus ESignatureStatus
	esigSelector := workflow.NewSelector(ctx)

	esigCompleteCh := workflow.GetSignalChannel(ctx, "esignature_complete")
	esigSelector.AddReceive(esigCompleteCh, func(c workflow.ReceiveChannel, more bool) {
		c.Receive(ctx, &esigStatus)
	})

	// Timeout after 14 days for signatures
	esigTimeout := workflow.NewTimer(ctx, 14*24*time.Hour)
	esigSelector.AddFuture(esigTimeout, func(f workflow.Future) {
		esigStatus = ESignatureStatus{AllSigned: false, PendingSigners: []string{"Timeout"}}
	})

	esigSelector.Select(ctx)

	output.DocumentsCompleted = esigStatus.AllSigned

	if !esigStatus.AllSigned {
		logger.Warn("E-signature not completed", "pending", esigStatus.PendingSigners)
		// Continue anyway - documents can be completed later
	}

	// ========================================
	// Stage 6: Commitment & Onboarding
	// ========================================
	logger.Info("Stage 6: Processing commitment and onboarding")

	err = workflow.ExecuteActivity(shortCtx, UpdateOpportunityStageActivity,
		input.OpportunityID,
		"COMMITTED",
		"Documents signed - commitment recorded",
	).Get(ctx, nil)

	// Process capital commitment
	err = workflow.ExecuteActivity(shortCtx, ProcessCapitalCommitmentActivity,
		input.OpportunityID,
		input.ClientID,
		committeeDecision.ApprovedAmount,
	).Get(ctx, nil)
	if err != nil {
		logger.Error("Failed to process commitment", "error", err)
	}

	// Onboard to portfolio
	err = workflow.ExecuteActivity(shortCtx, OnboardToPortfolioActivity,
		input.OpportunityID,
		input.ClientID,
	).Get(ctx, nil)
	if err != nil {
		logger.Error("Failed to onboard to portfolio", "error", err)
	}

	// Update final stage
	err = workflow.ExecuteActivity(shortCtx, UpdateOpportunityStageActivity,
		input.OpportunityID,
		"CLOSED_WON",
		"Investment committed and onboarded to portfolio",
	).Get(ctx, nil)

	output.Success = true
	output.FinalStage = "CLOSED_WON"
	output.TotalDurationDays = int(workflow.Now(ctx).Sub(startTime).Hours() / 24)

	logger.Info("Due diligence pipeline completed successfully",
		"opportunity_id", input.OpportunityID,
		"duration_days", output.TotalDurationDays,
		"approved_amount", output.ApprovedAmount,
	)

	return output, nil
}

// ============================================================================
// 2. QUARTERLY PORTFOLIO REVIEW WORKFLOW
// ============================================================================

// QuarterlyReviewInput contains input for the quarterly review workflow
type QuarterlyReviewInput struct {
	ClientID     uuid.UUID `json:"client_id"`
	ReviewPeriod string    `json:"review_period"` // e.g., "2025-Q4"
	AdvisorID    uuid.UUID `json:"advisor_id"`
}

// QuarterlyReviewOutput contains the results of the quarterly review
type QuarterlyReviewOutput struct {
	ReviewID          uuid.UUID         `json:"review_id"`
	ClientID          uuid.UUID         `json:"client_id"`
	ReviewPeriod      string            `json:"review_period"`
	Performance       PerformanceData   `json:"performance"`
	LiquidityAnalysis LiquidityAnalysis `json:"liquidity_analysis"`
	RiskFlags         []RiskFlag        `json:"risk_flags"`
	ReportURL         string            `json:"report_url"`
	MeetingScheduled  bool              `json:"meeting_scheduled"`
	MeetingDate       *time.Time        `json:"meeting_date,omitempty"`
	ActionItems       []string          `json:"action_items"`
}

// PerformanceData contains portfolio performance metrics
type PerformanceData struct {
	TotalAltAUM           float64 `json:"total_alt_aum"`
	PortfolioReturnPct    float64 `json:"portfolio_return_pct"`
	AltPortfolioReturnPct float64 `json:"alt_portfolio_return_pct"`
	BenchmarkReturnPct    float64 `json:"benchmark_return_pct"`
	Alpha                 float64 `json:"alpha"`
	TotalIRR              float64 `json:"total_irr"`
	TotalTVPI             float64 `json:"total_tvpi"`
	TotalDPI              float64 `json:"total_dpi"`
}

// LiquidityAnalysis contains liquidity metrics
type LiquidityAnalysis struct {
	TotalUnfundedCommitments float64 `json:"total_unfunded_commitments"`
	UpcomingCalls30d         float64 `json:"upcoming_calls_30d"`
	UpcomingCalls90d         float64 `json:"upcoming_calls_90d"`
	AvailableLiquidity       float64 `json:"available_liquidity"`
	LiquidityCoverageRatio   float64 `json:"liquidity_coverage_ratio"`
	DryPowderPct             float64 `json:"dry_powder_pct"`
}

// RiskFlag represents a flagged risk item
type RiskFlag struct {
	InvestmentID uuid.UUID `json:"investment_id"`
	FundName     string    `json:"fund_name"`
	FlagType     string    `json:"flag_type"`
	Severity     string    `json:"severity"` // LOW, MEDIUM, HIGH, CRITICAL
	Description  string    `json:"description"`
}

// QuarterlyAlternativesReviewWorkflow orchestrates quarterly portfolio monitoring
func QuarterlyAlternativesReviewWorkflow(ctx workflow.Context, input QuarterlyReviewInput) (*QuarterlyReviewOutput, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting quarterly alternatives review",
		"client_id", input.ClientID,
		"period", input.ReviewPeriod,
	)

	output := &QuarterlyReviewOutput{
		ClientID:     input.ClientID,
		ReviewPeriod: input.ReviewPeriod,
	}

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Create review record
	var reviewID uuid.UUID
	err := workflow.ExecuteActivity(ctx, CreateQuarterlyReviewRecordActivity,
		input.ClientID,
		input.ReviewPeriod,
	).Get(ctx, &reviewID)
	if err != nil {
		return nil, fmt.Errorf("failed to create review record: %w", err)
	}
	output.ReviewID = reviewID

	// ========================================
	// Parallel Data Collection
	// ========================================
	logger.Info("Collecting portfolio data in parallel")

	var performance PerformanceData
	var liquidity LiquidityAnalysis
	var managerUpdates []ManagerUpdateSummary

	perfFuture := workflow.ExecuteActivity(ctx, CalculatePortfolioPerformanceActivity, input.ClientID, input.ReviewPeriod)
	liquidityFuture := workflow.ExecuteActivity(ctx, LiquidityStressTestActivity, input.ClientID)
	managerFuture := workflow.ExecuteActivity(ctx, FetchManagerUpdatesActivity, input.ClientID, input.ReviewPeriod)

	if err := perfFuture.Get(ctx, &performance); err != nil {
		logger.Error("Failed to calculate performance", "error", err)
	}
	output.Performance = performance

	if err := liquidityFuture.Get(ctx, &liquidity); err != nil {
		logger.Error("Failed to run liquidity test", "error", err)
	}
	output.LiquidityAnalysis = liquidity

	if err := managerFuture.Get(ctx, &managerUpdates); err != nil {
		logger.Error("Failed to fetch manager updates", "error", err)
	}

	// ========================================
	// Risk Assessment
	// ========================================
	logger.Info("Running risk assessment")

	var riskFlags []RiskFlag
	err = workflow.ExecuteActivity(ctx, AssessPortfolioRisksActivity,
		input.ClientID,
		performance,
		liquidity,
		managerUpdates,
	).Get(ctx, &riskFlags)
	if err != nil {
		logger.Error("Failed to assess risks", "error", err)
	}
	output.RiskFlags = riskFlags

	// Escalate high-risk positions
	if len(riskFlags) > 0 {
		highRiskFlags := filterHighRiskFlags(riskFlags)
		if len(highRiskFlags) > 0 {
			err = workflow.ExecuteActivity(ctx, EscalateRiskPositionsActivity,
				input.ClientID,
				input.AdvisorID,
				highRiskFlags,
			).Get(ctx, nil)
			if err != nil {
				logger.Error("Failed to escalate risks", "error", err)
			}
		}
	}

	// ========================================
	// Generate Report
	// ========================================
	logger.Info("Generating quarterly report")

	var reportURL string
	err = workflow.ExecuteActivity(ctx, GenerateQuarterlyReportActivity,
		reviewID,
		input.ClientID,
		performance,
		liquidity,
		riskFlags,
		managerUpdates,
	).Get(ctx, &reportURL)
	if err != nil {
		logger.Error("Failed to generate report", "error", err)
	}
	output.ReportURL = reportURL

	// ========================================
	// Schedule Client Meeting
	// ========================================
	logger.Info("Scheduling client review meeting")

	var meetingDate time.Time
	err = workflow.ExecuteActivity(ctx, ScheduleClientReviewMeetingActivity,
		input.ClientID,
		input.AdvisorID,
		reviewID,
		reportURL,
	).Get(ctx, &meetingDate)
	if err != nil {
		logger.Warn("Failed to schedule meeting", "error", err)
		output.MeetingScheduled = false
	} else {
		output.MeetingScheduled = true
		output.MeetingDate = &meetingDate
	}

	// ========================================
	// Update Investment Committee
	// ========================================
	err = workflow.ExecuteActivity(ctx, UpdateInvestmentCommitteeActivity,
		input.ClientID,
		performance,
		riskFlags,
	).Get(ctx, nil)
	if err != nil {
		logger.Warn("Failed to update committee", "error", err)
	}

	// ========================================
	// Update Review Status
	// ========================================
	err = workflow.ExecuteActivity(ctx, UpdateQuarterlyReviewStatusActivity,
		reviewID,
		"REPORT_GENERATED",
		reportURL,
	).Get(ctx, nil)

	logger.Info("Quarterly review completed",
		"review_id", reviewID,
		"risk_flags_count", len(riskFlags),
	)

	return output, nil
}

// ============================================================================
// 3. CAPITAL CALL FUNDING WORKFLOW
// ============================================================================

// CapitalCallFundingInput contains input for the capital call workflow
type CapitalCallFundingInput struct {
	EventID        uuid.UUID `json:"event_id"`
	ClientID       uuid.UUID `json:"client_id"`
	InvestmentID   uuid.UUID `json:"investment_id"`
	AmountRequired float64   `json:"amount_required"`
	DueDate        time.Time `json:"due_date"`
}

// CapitalCallFundingOutput contains the result of the funding workflow
type CapitalCallFundingOutput struct {
	EventID              uuid.UUID `json:"event_id"`
	Success              bool      `json:"success"`
	AmountFunded         float64   `json:"amount_funded"`
	FundingSourceAccount uuid.UUID `json:"funding_source_account"`
	LiquidityCheckPassed bool      `json:"liquidity_check_passed"`
	FailureReason        string    `json:"failure_reason,omitempty"`
	FundedAt             time.Time `json:"funded_at,omitempty"`
}

// CapitalCallFundingWorkflow automates capital call funding
func CapitalCallFundingWorkflow(ctx workflow.Context, input CapitalCallFundingInput) (*CapitalCallFundingOutput, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting capital call funding workflow",
		"event_id", input.EventID,
		"client_id", input.ClientID,
		"amount", input.AmountRequired,
		"due_date", input.DueDate,
	)

	output := &CapitalCallFundingOutput{
		EventID: input.EventID,
	}

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Step 1: Identify funding source
	logger.Info("Identifying funding source account")

	var fundingAccount FundingAccountResult
	err := workflow.ExecuteActivity(ctx, IdentifyFundingSourceActivity,
		input.ClientID,
		input.AmountRequired,
	).Get(ctx, &fundingAccount)
	if err != nil {
		logger.Error("Failed to identify funding source", "error", err)
		output.FailureReason = "Could not identify funding source"
		return output, nil
	}

	output.FundingSourceAccount = fundingAccount.AccountID
	output.LiquidityCheckPassed = fundingAccount.SufficientBalance

	if !fundingAccount.SufficientBalance {
		logger.Warn("Insufficient liquidity for capital call",
			"available", fundingAccount.AvailableBalance,
			"required", input.AmountRequired,
		)

		// Send liquidity alert
		err = workflow.ExecuteActivity(ctx, SendCapitalCallLiquidityAlertActivity,
			input.EventID,
			input.ClientID,
			input.AmountRequired,
			fundingAccount.AvailableBalance,
		).Get(ctx, nil)

		// Create advisor task for manual resolution
		err = workflow.ExecuteActivity(ctx, CreateAdvisorTaskActivity,
			fundingAccount.AdvisorID,
			input.InvestmentID,
			"CAPITAL_CALL_FUNDING",
			fmt.Sprintf("Capital call requires $%.2f but only $%.2f available",
				input.AmountRequired, fundingAccount.AvailableBalance),
		).Get(ctx, nil)

		output.FailureReason = "Insufficient liquidity"
		return output, nil
	}

	// Step 2: Calculate days until due
	daysUntilDue := int(input.DueDate.Sub(workflow.Now(ctx)).Hours() / 24)

	// If due date is far out, schedule funding closer to date
	if daysUntilDue > 5 {
		// Wait until 3 business days before due date
		waitDuration := time.Duration(daysUntilDue-3) * 24 * time.Hour
		logger.Info("Scheduling funding for closer to due date",
			"wait_days", daysUntilDue-3,
		)
		workflow.Sleep(ctx, waitDuration)
	}

	// Step 3: Initiate funding transfer
	logger.Info("Initiating funding transfer")

	var transferResult TransferResult
	err = workflow.ExecuteActivity(ctx, InitiateFundingTransferActivity,
		input.EventID,
		fundingAccount.AccountID,
		input.AmountRequired,
	).Get(ctx, &transferResult)
	if err != nil {
		logger.Error("Failed to initiate transfer", "error", err)
		output.FailureReason = "Transfer failed"
		return output, nil
	}

	// Step 4: Wait for transfer confirmation
	logger.Info("Waiting for transfer confirmation")

	// Poll for confirmation or wait for signal
	transferConfirmSelector := workflow.NewSelector(ctx)

	confirmCh := workflow.GetSignalChannel(ctx, "transfer_confirmed")
	transferConfirmSelector.AddReceive(confirmCh, func(c workflow.ReceiveChannel, more bool) {
		var confirmed bool
		c.Receive(ctx, &confirmed)
		if confirmed {
			output.Success = true
			output.AmountFunded = input.AmountRequired
			output.FundedAt = workflow.Now(ctx)
		}
	})

	// Timeout after 24 hours
	confirmTimeout := workflow.NewTimer(ctx, 24*time.Hour)
	transferConfirmSelector.AddFuture(confirmTimeout, func(f workflow.Future) {
		output.FailureReason = "Transfer confirmation timed out"
	})

	transferConfirmSelector.Select(ctx)

	// Step 5: Update capital event status
	if output.Success {
		err = workflow.ExecuteActivity(ctx, UpdateCapitalEventStatusActivity,
			input.EventID,
			"FUNDED",
			output.AmountFunded,
		).Get(ctx, nil)
		if err != nil {
			logger.Error("Failed to update event status", "error", err)
		}

		// Send confirmation
		err = workflow.ExecuteActivity(ctx, SendCapitalCallConfirmationActivity,
			input.EventID,
			input.ClientID,
			output.AmountFunded,
		).Get(ctx, nil)
	}

	logger.Info("Capital call funding workflow completed",
		"success", output.Success,
		"amount_funded", output.AmountFunded,
	)

	return output, nil
}

// ============================================================================
// 4. ALLOCATION DRIFT MONITORING WORKFLOW
// ============================================================================

// AllocationDriftMonitoringWorkflow continuously monitors portfolio drift
func AllocationDriftMonitoringWorkflow(ctx workflow.Context) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting allocation drift monitoring workflow")

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Run continuously - check daily
	for {
		logger.Info("Checking allocation drift across all clients")

		// Get all clients with allocation targets
		var clientsToCheck []uuid.UUID
		err := workflow.ExecuteActivity(ctx, GetClientsWithAllocationTargetsActivity).Get(ctx, &clientsToCheck)
		if err != nil {
			logger.Error("Failed to get client list", "error", err)
			workflow.Sleep(ctx, 24*time.Hour)
			continue
		}

		// Check each client's allocation drift
		for _, clientID := range clientsToCheck {
			var driftResults []AllocationDriftResult
			err := workflow.ExecuteActivity(ctx, CheckClientAllocationDriftActivity, clientID).Get(ctx, &driftResults)
			if err != nil {
				logger.Error("Failed to check drift for client", "client_id", clientID, "error", err)
				continue
			}

			// Process any drift triggers
			for _, drift := range driftResults {
				if drift.ActionRequired {
					logger.Info("Allocation drift detected",
						"client_id", clientID,
						"asset_class", drift.AssetClass,
						"deviation", drift.DeviationPct,
					)

					// Create rebalance trigger
					err = workflow.ExecuteActivity(ctx, CreateRebalanceTriggerActivity,
						clientID,
						drift,
					).Get(ctx, nil)
					if err != nil {
						logger.Error("Failed to create rebalance trigger", "error", err)
					}

					// Notify advisor
					err = workflow.ExecuteActivity(ctx, NotifyAdvisorRebalanceActivity,
						clientID,
						drift,
					).Get(ctx, nil)
					if err != nil {
						logger.Error("Failed to notify advisor", "error", err)
					}
				}
			}
		}

		// Sleep for 24 hours before next check
		workflow.Sleep(ctx, 24*time.Hour)
	}
}

// ============================================================================
// 5. REGULATORY FILING WORKFLOW
// ============================================================================

// RegulatoryFilingInput contains input for regulatory filing workflow
type RegulatoryFilingInput struct {
	FilingType      string    `json:"filing_type"`
	ReportingPeriod string    `json:"reporting_period"`
	DueDate         time.Time `json:"due_date"`
}

// RegulatoryFilingWorkflow automates regulatory filing preparation
func RegulatoryFilingWorkflow(ctx workflow.Context, input RegulatoryFilingInput) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting regulatory filing workflow",
		"type", input.FilingType,
		"period", input.ReportingPeriod,
	)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Step 1: Create filing record
	var filingID uuid.UUID
	err := workflow.ExecuteActivity(ctx, CreateRegulatoryFilingRecordActivity,
		input.FilingType,
		input.ReportingPeriod,
		input.DueDate,
	).Get(ctx, &filingID)
	if err != nil {
		return fmt.Errorf("failed to create filing record: %w", err)
	}

	// Step 2: Collect required data
	logger.Info("Collecting filing data")

	var filingData RegulatoryFilingData
	err = workflow.ExecuteActivity(ctx, CollectRegulatoryFilingDataActivity,
		input.FilingType,
		input.ReportingPeriod,
	).Get(ctx, &filingData)
	if err != nil {
		return fmt.Errorf("failed to collect filing data: %w", err)
	}

	// Step 3: Generate filing document
	logger.Info("Generating filing document")

	var documentURL string
	err = workflow.ExecuteActivity(ctx, GenerateRegulatoryFilingDocumentActivity,
		filingID,
		input.FilingType,
		filingData,
	).Get(ctx, &documentURL)
	if err != nil {
		return fmt.Errorf("failed to generate document: %w", err)
	}

	// Step 4: Create review task
	err = workflow.ExecuteActivity(ctx, CreateComplianceReviewTaskActivity,
		filingID,
		input.FilingType,
		documentURL,
	).Get(ctx, nil)
	if err != nil {
		logger.Warn("Failed to create review task", "error", err)
	}

	// Step 5: Wait for review approval
	var approved bool
	reviewSelector := workflow.NewSelector(ctx)

	reviewCh := workflow.GetSignalChannel(ctx, "filing_approved")
	reviewSelector.AddReceive(reviewCh, func(c workflow.ReceiveChannel, more bool) {
		c.Receive(ctx, &approved)
	})

	// Calculate time until 3 days before due date for reminder
	reminderTime := input.DueDate.Add(-3 * 24 * time.Hour)
	reminderTimer := workflow.NewTimer(ctx, reminderTime.Sub(workflow.Now(ctx)))
	reviewSelector.AddFuture(reminderTimer, func(f workflow.Future) {
		// Send reminder
		workflow.ExecuteActivity(ctx, SendFilingReminderActivity, filingID, input.DueDate)
	})

	reviewSelector.Select(ctx)

	if approved {
		// Update status to submitted
		err = workflow.ExecuteActivity(ctx, UpdateRegulatoryFilingStatusActivity,
			filingID,
			"SUBMITTED",
		).Get(ctx, nil)
		if err != nil {
			logger.Error("Failed to update filing status", "error", err)
		}
	}

	logger.Info("Regulatory filing workflow completed",
		"filing_id", filingID,
		"approved", approved,
	)

	return nil
}

// ============================================================================
// HELPER TYPES
// ============================================================================

// ManagerUpdateSummary contains summary of manager communications
type ManagerUpdateSummary struct {
	InvestmentID uuid.UUID `json:"investment_id"`
	FundName     string    `json:"fund_name"`
	UpdateType   string    `json:"update_type"`
	Title        string    `json:"title"`
	ReportedNAV  float64   `json:"reported_nav,omitempty"`
	ReportedIRR  float64   `json:"reported_irr,omitempty"`
	DocumentURL  string    `json:"document_url"`
}

// FundingAccountResult contains funding source identification result
type FundingAccountResult struct {
	AccountID         uuid.UUID `json:"account_id"`
	AccountName       string    `json:"account_name"`
	AvailableBalance  float64   `json:"available_balance"`
	SufficientBalance bool      `json:"sufficient_balance"`
	AdvisorID         uuid.UUID `json:"advisor_id"`
}

// TransferResult contains transfer initiation result
type TransferResult struct {
	TransferID   uuid.UUID `json:"transfer_id"`
	Status       string    `json:"status"`
	Amount       float64   `json:"amount"`
	InitiatedAt  time.Time `json:"initiated_at"`
	ExpectedDate time.Time `json:"expected_date"`
}

// AllocationDriftResult contains drift calculation for an asset class
type AllocationDriftResult struct {
	AssetClass           string  `json:"asset_class"`
	CurrentAllocationPct float64 `json:"current_allocation_pct"`
	TargetAllocationPct  float64 `json:"target_allocation_pct"`
	DeviationPct         float64 `json:"deviation_pct"`
	ToleranceBandPct     float64 `json:"tolerance_band_pct"`
	ActionRequired       bool    `json:"action_required"`
	RecommendedAction    string  `json:"recommended_action"`
}

// RegulatoryFilingData contains collected data for regulatory filing
type RegulatoryFilingData struct {
	QualifiedClientsCount      int                    `json:"qualified_clients_count"`
	PerformanceFeeClientsCount int                    `json:"performance_fee_clients_count"`
	IlliquidAssetsValue        float64                `json:"illiquid_assets_value"`
	SidePocketValue            float64                `json:"side_pocket_value"`
	TotalAlternativeAUM        float64                `json:"total_alternative_aum"`
	AdditionalData             map[string]interface{} `json:"additional_data"`
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func collectReviewFailures(results *ParallelReviewResults) []string {
	var failures []string
	if !results.RiskAssessment.Passed {
		failures = append(failures, "Risk Assessment: "+results.RiskAssessment.Findings[0])
	}
	if !results.LegalReview.Passed {
		failures = append(failures, "Legal Review: "+results.LegalReview.Findings[0])
	}
	if !results.TaxAnalysis.Passed {
		failures = append(failures, "Tax Analysis: "+results.TaxAnalysis.Findings[0])
	}
	if !results.OperationalDD.Passed {
		failures = append(failures, "Operational DD: "+results.OperationalDD.Findings[0])
	}
	return failures
}

func filterHighRiskFlags(flags []RiskFlag) []RiskFlag {
	var highRisk []RiskFlag
	for _, f := range flags {
		if f.Severity == "HIGH" || f.Severity == "CRITICAL" {
			highRisk = append(highRisk, f)
		}
	}
	return highRisk
}
