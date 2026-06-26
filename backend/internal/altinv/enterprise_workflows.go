package altinv

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// ============================================================================
// ENTERPRISE ADVISOR WORKFLOWS - BEST IN CLASS
// ============================================================================
// Features:
// - Parallel processing with configurable gates
// - AI-powered decision support
// - Real-time collaboration & notifications
// - Comprehensive audit trail
// - Escalation policies
// - SLA monitoring
// - Rollback capabilities
// ============================================================================

// ============================================================================
// ENHANCED DATA STRUCTURES
// ============================================================================

// WorkflowConfig contains configurable workflow parameters
type WorkflowConfig struct {
	// Timeouts
	ScreeningTimeout       time.Duration `json:"screening_timeout"`
	AdvisorReviewTimeout   time.Duration `json:"advisor_review_timeout"`
	CommitteeReviewTimeout time.Duration `json:"committee_review_timeout"`
	DocumentSigningTimeout time.Duration `json:"document_signing_timeout"`

	// Escalation
	EscalationEnabled   bool          `json:"escalation_enabled"`
	EscalationThreshold time.Duration `json:"escalation_threshold"`
	EscalatorUserIDs    []uuid.UUID   `json:"escalator_user_ids"`

	// SLA
	SLAEnabled          bool          `json:"sla_enabled"`
	TargetCycleTime     time.Duration `json:"target_cycle_time"`
	SLAWarningThreshold float64       `json:"sla_warning_threshold"` // e.g., 0.8 = warn at 80%

	// Parallel processing
	MaxParallelReviews int  `json:"max_parallel_reviews"`
	RequireAllReviews  bool `json:"require_all_reviews"`

	// AI settings
	AIScreeningEnabled    bool    `json:"ai_screening_enabled"`
	AIConfidenceThreshold float64 `json:"ai_confidence_threshold"`
	AutoApproveHighScore  bool    `json:"auto_approve_high_score"`
	AutoApproveThreshold  float64 `json:"auto_approve_threshold"`

	// Notifications
	NotifyOnStageChange    bool `json:"notify_on_stage_change"`
	NotifyOnEscalation     bool `json:"notify_on_escalation"`
	NotifyOnSLAWarning     bool `json:"notify_on_sla_warning"`
	NotifyClientOnApproval bool `json:"notify_client_on_approval"`
}

// DefaultWorkflowConfig returns sensible defaults
func DefaultWorkflowConfig() WorkflowConfig {
	return WorkflowConfig{
		ScreeningTimeout:       1 * time.Hour,
		AdvisorReviewTimeout:   14 * 24 * time.Hour,
		CommitteeReviewTimeout: 30 * 24 * time.Hour,
		DocumentSigningTimeout: 14 * 24 * time.Hour,
		EscalationEnabled:      true,
		EscalationThreshold:    7 * 24 * time.Hour,
		SLAEnabled:             true,
		TargetCycleTime:        45 * 24 * time.Hour,
		SLAWarningThreshold:    0.8,
		MaxParallelReviews:     5,
		RequireAllReviews:      false, // Allow proceeding with conditions
		AIScreeningEnabled:     true,
		AIConfidenceThreshold:  0.85,
		AutoApproveHighScore:   false,
		AutoApproveThreshold:   0.95,
		NotifyOnStageChange:    true,
		NotifyOnEscalation:     true,
		NotifyOnSLAWarning:     true,
		NotifyClientOnApproval: true,
	}
}

// EnhancedDueDiligenceInput extends the basic input with configuration
type EnhancedDueDiligenceInput struct {
	OpportunityID  uuid.UUID              `json:"opportunity_id"`
	ClientID       uuid.UUID              `json:"client_id"`
	AdvisorID      uuid.UUID              `json:"advisor_id"`
	Config         WorkflowConfig         `json:"config"`
	Priority       string                 `json:"priority"` // NORMAL, HIGH, URGENT
	RequestedBy    uuid.UUID              `json:"requested_by"`
	BusinessReason string                 `json:"business_reason"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// EnhancedDueDiligenceOutput provides comprehensive workflow results
type EnhancedDueDiligenceOutput struct {
	OpportunityID uuid.UUID `json:"opportunity_id"`
	WorkflowRunID string    `json:"workflow_run_id"`
	FinalStage    string    `json:"final_stage"`
	Success       bool      `json:"success"`

	// Screening
	ScreeningResult *AIScreeningResult `json:"screening_result"`

	// Reviews
	AdvisorDecision   *AdvisorDecision         `json:"advisor_decision"`
	ReviewResults     *EnhancedReviewResults   `json:"review_results"`
	CommitteeDecision *CommitteeDecisionResult `json:"committee_decision"`

	// Documents
	DocumentsCompleted bool    `json:"documents_completed"`
	ApprovedAmount     float64 `json:"approved_amount,omitempty"`

	// Timing
	StartedAt         time.Time                `json:"started_at"`
	CompletedAt       time.Time                `json:"completed_at"`
	TotalDurationDays int                      `json:"total_duration_days"`
	StageDurations    map[string]time.Duration `json:"stage_durations"`

	// SLA
	SLAMet          bool   `json:"sla_met"`
	SLABreachReason string `json:"sla_breach_reason,omitempty"`

	// Issues
	FailureReason        string   `json:"failure_reason,omitempty"`
	Warnings             []string `json:"warnings,omitempty"`
	EscalationsTriggered int      `json:"escalations_triggered"`

	// Audit
	AuditTrail []AuditEntry `json:"audit_trail"`
}

// AIScreeningResult contains AI-enhanced screening results
type AIScreeningResult struct {
	Passed              bool                 `json:"passed"`
	Score               float64              `json:"score"`
	AIConfidence        float64              `json:"ai_confidence"`
	RuleResults         []RuleResult         `json:"rule_results"`
	AIRecommendations   []string             `json:"ai_recommendations"`
	SimilarDeals        []SimilarDeal        `json:"similar_deals"`
	RiskSignals         []RiskSignal         `json:"risk_signals"`
	SentimentAnalysis   *SentimentAnalysis   `json:"sentiment_analysis,omitempty"`
	MarketTimingSignals *MarketTimingSignals `json:"market_timing_signals,omitempty"`
	OverallAssessment   string               `json:"overall_assessment"`
}

// RuleResult captures individual screening rule outcomes
type RuleResult struct {
	RuleName    string      `json:"rule_name"`
	RuleCode    string      `json:"rule_code"`
	Passed      bool        `json:"passed"`
	Score       float64     `json:"score"`
	MaxScore    float64     `json:"max_score"`
	ActualValue interface{} `json:"actual_value"`
	Threshold   interface{} `json:"threshold"`
	Required    bool        `json:"required"`
	Details     string      `json:"details"`
}

// SimilarDeal contains comparable historical deals
type SimilarDeal struct {
	OpportunityID   uuid.UUID `json:"opportunity_id"`
	FundName        string    `json:"fund_name"`
	SimilarityScore float64   `json:"similarity_score"`
	Outcome         string    `json:"outcome"`
	ActualReturn    float64   `json:"actual_return,omitempty"`
}

// RiskSignal represents AI-detected risk indicators
type RiskSignal struct {
	SignalType  string   `json:"signal_type"`
	Severity    string   `json:"severity"` // LOW, MEDIUM, HIGH, CRITICAL
	Description string   `json:"description"`
	Source      string   `json:"source"`
	Confidence  float64  `json:"confidence"`
	Mitigations []string `json:"mitigations,omitempty"`
}

// SentimentAnalysis from document NLP processing
type SentimentAnalysis struct {
	OverallSentiment float64            `json:"overall_sentiment"` // -1 to 1
	BySection        map[string]float64 `json:"by_section"`
	KeyPhrases       []string           `json:"key_phrases"`
	ConcernsDetected []string           `json:"concerns_detected"`
	ReadabilityScore float64            `json:"readability_score"`
}

// MarketTimingSignals contains market condition analysis
type MarketTimingSignals struct {
	MarketCyclePhase   string  `json:"market_cycle_phase"` // EARLY, MID, LATE
	ValuationLevel     string  `json:"valuation_level"`    // LOW, FAIR, HIGH
	CompetitionLevel   string  `json:"competition_level"`  // LOW, MEDIUM, HIGH
	FundraisingClimate string  `json:"fundraising_climate"`
	RecommendedAction  string  `json:"recommended_action"` // PROCEED, WAIT, PASS
	Confidence         float64 `json:"confidence"`
}

// EnhancedReviewResults contains all parallel review outcomes
type EnhancedReviewResults struct {
	RiskAssessment  *EnhancedReviewResult `json:"risk_assessment"`
	LegalReview     *EnhancedReviewResult `json:"legal_review"`
	TaxAnalysis     *EnhancedReviewResult `json:"tax_analysis"`
	OperationalDD   *EnhancedReviewResult `json:"operational_dd"`
	ESGReview       *EnhancedReviewResult `json:"esg_review,omitempty"`
	ReferenceChecks *EnhancedReviewResult `json:"reference_checks,omitempty"`
	ConflictCheck   *EnhancedReviewResult `json:"conflict_check,omitempty"`

	// Aggregate
	OverallScore     float64  `json:"overall_score"`
	OverallRiskLevel string   `json:"overall_risk_level"`
	Recommendation   string   `json:"recommendation"`
	MustAddress      []string `json:"must_address"`
	ShouldConsider   []string `json:"should_consider"`
}

// EnhancedReviewResult contains detailed review outcome
type EnhancedReviewResult struct {
	ReviewType string  `json:"review_type"`
	Passed     bool    `json:"passed"`
	Score      float64 `json:"score"`
	RiskLevel  string  `json:"risk_level"`

	// Details
	Findings     []Finding `json:"findings"`
	Conditions   []string  `json:"conditions,omitempty"`
	Requirements []string  `json:"requirements,omitempty"`

	// Reviewer
	ReviewedBy   uuid.UUID `json:"reviewed_by,omitempty"`
	ReviewerName string    `json:"reviewer_name,omitempty"`
	ReviewedAt   time.Time `json:"reviewed_at,omitempty"`

	// Timing
	StartedAt       time.Time `json:"started_at"`
	CompletedAt     time.Time `json:"completed_at"`
	DurationMinutes int       `json:"duration_minutes"`

	// AI assistance
	AIAssisted   bool    `json:"ai_assisted"`
	AIConfidence float64 `json:"ai_confidence,omitempty"`

	// Documents
	SupportingDocs []DocumentReference `json:"supporting_docs,omitempty"`
}

// Finding represents a specific due diligence finding
type Finding struct {
	FindingID      uuid.UUID `json:"finding_id"`
	Category       string    `json:"category"`
	Severity       string    `json:"severity"` // INFO, LOW, MEDIUM, HIGH, CRITICAL
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	Impact         string    `json:"impact"`
	Recommendation string    `json:"recommendation"`
	Status         string    `json:"status"` // OPEN, ACKNOWLEDGED, MITIGATED, ACCEPTED
}

// DocumentReference links to supporting documents
type DocumentReference struct {
	DocumentID    uuid.UUID `json:"document_id"`
	DocumentType  string    `json:"document_type"`
	DocumentName  string    `json:"document_name"`
	URL           string    `json:"url"`
	PageReference string    `json:"page_reference,omitempty"`
}

// AuditEntry captures workflow audit information
type AuditEntry struct {
	Timestamp       time.Time              `json:"timestamp"`
	Action          string                 `json:"action"`
	Stage           string                 `json:"stage"`
	PerformedBy     uuid.UUID              `json:"performed_by,omitempty"`
	PerformedByName string                 `json:"performed_by_name,omitempty"`
	Details         map[string]interface{} `json:"details,omitempty"`
	Duration        time.Duration          `json:"duration,omitempty"`
}

// EscalationEvent tracks escalation information
type EscalationEvent struct {
	EscalationType string      `json:"escalation_type"`
	Reason         string      `json:"reason"`
	EscalatedTo    []uuid.UUID `json:"escalated_to"`
	EscalatedAt    time.Time   `json:"escalated_at"`
	Resolved       bool        `json:"resolved"`
	ResolvedAt     *time.Time  `json:"resolved_at,omitempty"`
}

// ============================================================================
// ENTERPRISE DUE DILIGENCE WORKFLOW
// ============================================================================

// EnterpriseDueDiligenceWorkflow is the best-in-class due diligence pipeline
func EnterpriseDueDiligenceWorkflow(ctx workflow.Context, input EnhancedDueDiligenceInput) (*EnhancedDueDiligenceOutput, error) {
	logger := workflow.GetLogger(ctx)

	// Initialize output
	output := &EnhancedDueDiligenceOutput{
		OpportunityID:  input.OpportunityID,
		WorkflowRunID:  workflow.GetInfo(ctx).WorkflowExecution.RunID,
		StartedAt:      workflow.Now(ctx),
		StageDurations: make(map[string]time.Duration),
		AuditTrail:     make([]AuditEntry, 0),
	}

	// Use default config if not provided
	if input.Config.ScreeningTimeout == 0 {
		input.Config = DefaultWorkflowConfig()
	}

	// Add initial audit entry
	addAuditEntry(output, "WORKFLOW_STARTED", "INTAKE", uuid.Nil, map[string]interface{}{
		"priority":        input.Priority,
		"business_reason": input.BusinessReason,
		"config":          input.Config,
	})

	logger.Info("Starting enterprise due diligence workflow",
		"opportunity_id", input.OpportunityID,
		"priority", input.Priority,
		"ai_enabled", input.Config.AIScreeningEnabled,
	)

	// Configure activity options with different timeouts
	quickAO := workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    30 * time.Second,
			MaximumAttempts:    3,
		},
	}

	standardAO := workflow.ActivityOptions{
		StartToCloseTimeout: 15 * time.Minute,
		HeartbeatTimeout:    2 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    5 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    2 * time.Minute,
			MaximumAttempts:    3,
		},
	}

	longAO := workflow.ActivityOptions{
		StartToCloseTimeout: 1 * time.Hour,
		HeartbeatTimeout:    5 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    10 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    5 * time.Minute,
			MaximumAttempts:    2,
		},
	}

	quickCtx := workflow.WithActivityOptions(ctx, quickAO)
	standardCtx := workflow.WithActivityOptions(ctx, standardAO)
	longCtx := workflow.WithActivityOptions(ctx, longAO)

	// Track stage timing
	stageStart := workflow.Now(ctx)

	// ========================================================================
	// STAGE 1: AI-POWERED SCREENING
	// ========================================================================
	logger.Info("Stage 1: AI-powered screening")

	var screeningResult AIScreeningResult

	if input.Config.AIScreeningEnabled {
		// Run AI screening with document analysis
		err := workflow.ExecuteActivity(standardCtx, RunAIScreeningActivity,
			input.OpportunityID,
			input.Config.AIConfidenceThreshold,
		).Get(ctx, &screeningResult)

		if err != nil {
			logger.Error("AI screening failed, falling back to rule-based", "error", err)
			output.Warnings = append(output.Warnings, "AI screening failed, used rule-based fallback")

			// Fallback to basic screening
			var basicResult ScreeningResult
			err = workflow.ExecuteActivity(standardCtx, RunAutomatedScreeningActivity, input.OpportunityID).Get(ctx, &basicResult)
			if err != nil {
				output.FinalStage = "INTAKE"
				output.FailureReason = fmt.Sprintf("Screening failed: %v", err)
				addAuditEntry(output, "SCREENING_FAILED", "INTAKE", uuid.Nil, map[string]interface{}{"error": err.Error()})
				return output, nil
			}
			screeningResult = AIScreeningResult{
				Passed:       basicResult.Passed,
				Score:        basicResult.Score,
				AIConfidence: 0,
			}
		}
	} else {
		// Rule-based screening only
		var basicResult ScreeningResult
		err := workflow.ExecuteActivity(standardCtx, RunAutomatedScreeningActivity, input.OpportunityID).Get(ctx, &basicResult)
		if err != nil {
			output.FinalStage = "INTAKE"
			output.FailureReason = fmt.Sprintf("Screening failed: %v", err)
			return output, nil
		}
		screeningResult = AIScreeningResult{
			Passed:       basicResult.Passed,
			Score:        basicResult.Score,
			AIConfidence: 0,
		}
	}

	output.ScreeningResult = &screeningResult
	output.StageDurations["screening"] = workflow.Now(ctx).Sub(stageStart)

	addAuditEntry(output, "SCREENING_COMPLETED", "INITIAL_SCREEN", uuid.Nil, map[string]interface{}{
		"passed":        screeningResult.Passed,
		"score":         screeningResult.Score,
		"ai_confidence": screeningResult.AIConfidence,
		"duration":      output.StageDurations["screening"].String(),
	})

	// Handle screening failure
	if !screeningResult.Passed {
		logger.Info("Screening failed", "score", screeningResult.Score)

		// Notify advisor
		_ = workflow.ExecuteActivity(quickCtx, NotifyAdvisorScreeningFailedActivity,
			input.OpportunityID, input.AdvisorID, screeningResult,
		).Get(ctx, nil)

		// Update stage
		_ = workflow.ExecuteActivity(quickCtx, UpdateOpportunityStageActivity,
			input.OpportunityID, "CLOSED_LOST", "Failed automated screening",
		).Get(ctx, nil)

		output.FinalStage = "CLOSED_LOST"
		output.FailureReason = "Screening failed: " + screeningResult.OverallAssessment
		output.CompletedAt = workflow.Now(ctx)
		output.TotalDurationDays = int(output.CompletedAt.Sub(output.StartedAt).Hours() / 24)
		return output, nil
	}

	// Check for auto-approval on very high scores
	if input.Config.AutoApproveHighScore && screeningResult.Score >= input.Config.AutoApproveThreshold*100 {
		logger.Info("Auto-advancing due to high screening score", "score", screeningResult.Score)
		addAuditEntry(output, "AUTO_ADVANCED", "INITIAL_SCREEN", uuid.Nil, map[string]interface{}{
			"reason": "High screening score exceeded auto-approve threshold",
			"score":  screeningResult.Score,
		})
	}

	// ========================================================================
	// STAGE 2: ADVISOR REVIEW WITH ESCALATION
	// ========================================================================
	stageStart = workflow.Now(ctx)
	logger.Info("Stage 2: Advisor review")

	// Update stage
	_ = workflow.ExecuteActivity(quickCtx, UpdateOpportunityStageActivity,
		input.OpportunityID, "INITIAL_SCREEN", "Screening passed - awaiting advisor review",
	).Get(ctx, nil)

	// Notify advisor
	_ = workflow.ExecuteActivity(quickCtx, NotifyAdvisorForReviewActivity,
		input.OpportunityID, input.AdvisorID,
	).Get(ctx, nil)

	// Create task
	_ = workflow.ExecuteActivity(quickCtx, CreateAdvisorTaskActivity,
		input.AdvisorID, input.OpportunityID, "OPPORTUNITY_REVIEW",
		fmt.Sprintf("Review %s opportunity (Score: %.1f)", input.Priority, screeningResult.Score),
	).Get(ctx, nil)

	// Wait for advisor decision with escalation
	var advisorDecision AdvisorDecision
	advisorSelector := workflow.NewSelector(ctx)

	advisorDecisionCh := workflow.GetSignalChannel(ctx, "advisor_decision")
	advisorSelector.AddReceive(advisorDecisionCh, func(c workflow.ReceiveChannel, more bool) {
		c.Receive(ctx, &advisorDecision)
	})

	// Escalation timer
	escalationTriggered := false
	if input.Config.EscalationEnabled {
		escalationTimer := workflow.NewTimer(ctx, input.Config.EscalationThreshold)
		advisorSelector.AddFuture(escalationTimer, func(f workflow.Future) {
			if advisorDecision.ApprovedBy == uuid.Nil {
				escalationTriggered = true
				output.EscalationsTriggered++

				// Escalate
				_ = workflow.ExecuteActivity(quickCtx, EscalateReviewActivity,
					input.OpportunityID,
					input.Config.EscalatorUserIDs,
					"Advisor review timeout",
				).Get(ctx, nil)

				addAuditEntry(output, "ESCALATION_TRIGGERED", "INITIAL_SCREEN", uuid.Nil, map[string]interface{}{
					"reason":       "Advisor review timeout",
					"escalated_to": input.Config.EscalatorUserIDs,
				})
			}
		})
	}

	// Final timeout
	timeout := workflow.NewTimer(ctx, input.Config.AdvisorReviewTimeout)
	advisorSelector.AddFuture(timeout, func(f workflow.Future) {
		if advisorDecision.ApprovedBy == uuid.Nil {
			advisorDecision = AdvisorDecision{
				Approved: false,
				Reason:   "Advisor review timed out",
			}
		}
	})

	// Wait for first completion
	for advisorDecision.ApprovedBy == uuid.Nil && !advisorSelector.HasPending() == false {
		advisorSelector.Select(ctx)
		if advisorDecision.ApprovedBy != uuid.Nil {
			break
		}
		if escalationTriggered && advisorDecision.ApprovedBy == uuid.Nil {
			// Continue waiting after escalation
			continue
		}
	}

	output.AdvisorDecision = &advisorDecision
	output.StageDurations["advisor_review"] = workflow.Now(ctx).Sub(stageStart)

	addAuditEntry(output, "ADVISOR_DECISION", "INITIAL_SCREEN", advisorDecision.ApprovedBy, map[string]interface{}{
		"approved": advisorDecision.Approved,
		"reason":   advisorDecision.Reason,
		"duration": output.StageDurations["advisor_review"].String(),
	})

	if !advisorDecision.Approved {
		_ = workflow.ExecuteActivity(quickCtx, UpdateOpportunityStageActivity,
			input.OpportunityID, "CLOSED_LOST", "Advisor rejected: "+advisorDecision.Reason,
		).Get(ctx, nil)

		output.FinalStage = "CLOSED_LOST"
		output.FailureReason = "Advisor rejected: " + advisorDecision.Reason
		output.CompletedAt = workflow.Now(ctx)
		output.TotalDurationDays = int(output.CompletedAt.Sub(output.StartedAt).Hours() / 24)
		return output, nil
	}

	// ========================================================================
	// STAGE 3: PARALLEL DUE DILIGENCE
	// ========================================================================
	stageStart = workflow.Now(ctx)
	logger.Info("Stage 3: Parallel due diligence reviews")

	_ = workflow.ExecuteActivity(quickCtx, UpdateOpportunityStageActivity,
		input.OpportunityID, "DUE_DILIGENCE", "Advisor approved - starting due diligence",
	).Get(ctx, nil)

	// Initialize checklist
	_ = workflow.ExecuteActivity(standardCtx, InitializeDueDiligenceChecklistActivity,
		input.OpportunityID,
	).Get(ctx, nil)

	// Launch parallel reviews
	reviewResults := &EnhancedReviewResults{}

	// Core reviews (always required)
	riskFuture := workflow.ExecuteActivity(longCtx, EnhancedRiskAssessmentActivity, input.OpportunityID, input.Config)
	legalFuture := workflow.ExecuteActivity(longCtx, EnhancedLegalReviewActivity, input.OpportunityID, input.Config)
	taxFuture := workflow.ExecuteActivity(longCtx, EnhancedTaxAnalysisActivity, input.OpportunityID, input.Config)
	opsFuture := workflow.ExecuteActivity(longCtx, EnhancedOperationalDDActivity, input.OpportunityID, input.Config)

	// Optional reviews
	esgFuture := workflow.ExecuteActivity(longCtx, ESGReviewActivity, input.OpportunityID)
	refFuture := workflow.ExecuteActivity(longCtx, ReferenceChecksActivity, input.OpportunityID)
	conflictFuture := workflow.ExecuteActivity(longCtx, ConflictOfInterestCheckActivity, input.OpportunityID, input.ClientID)

	// Collect results
	var riskResult, legalResult, taxResult, opsResult EnhancedReviewResult
	var esgResult, refResult, conflictResult EnhancedReviewResult

	// Wait for core reviews
	if err := riskFuture.Get(ctx, &riskResult); err != nil {
		logger.Error("Risk assessment failed", "error", err)
		riskResult = createFailedReview("RISK", err)
	}
	reviewResults.RiskAssessment = &riskResult

	if err := legalFuture.Get(ctx, &legalResult); err != nil {
		logger.Error("Legal review failed", "error", err)
		legalResult = createFailedReview("LEGAL", err)
	}
	reviewResults.LegalReview = &legalResult

	if err := taxFuture.Get(ctx, &taxResult); err != nil {
		logger.Error("Tax analysis failed", "error", err)
		taxResult = createFailedReview("TAX", err)
	}
	reviewResults.TaxAnalysis = &taxResult

	if err := opsFuture.Get(ctx, &opsResult); err != nil {
		logger.Error("Operational DD failed", "error", err)
		opsResult = createFailedReview("OPERATIONAL", err)
	}
	reviewResults.OperationalDD = &opsResult

	// Collect optional reviews (don't fail if they error)
	if err := esgFuture.Get(ctx, &esgResult); err == nil {
		reviewResults.ESGReview = &esgResult
	}
	if err := refFuture.Get(ctx, &refResult); err == nil {
		reviewResults.ReferenceChecks = &refResult
	}
	if err := conflictFuture.Get(ctx, &conflictResult); err == nil {
		reviewResults.ConflictCheck = &conflictResult
	}

	// Calculate aggregate scores
	calculateAggregateReviewScores(reviewResults)

	output.ReviewResults = reviewResults
	output.StageDurations["due_diligence"] = workflow.Now(ctx).Sub(stageStart)

	addAuditEntry(output, "DD_COMPLETED", "DUE_DILIGENCE", uuid.Nil, map[string]interface{}{
		"overall_score":      reviewResults.OverallScore,
		"overall_risk_level": reviewResults.OverallRiskLevel,
		"recommendation":     reviewResults.Recommendation,
		"duration":           output.StageDurations["due_diligence"].String(),
	})

	// Check if critical issues block proceeding
	if reviewResults.OverallRiskLevel == "CRITICAL" && input.Config.RequireAllReviews {
		_ = workflow.ExecuteActivity(quickCtx, UpdateOpportunityStageActivity,
			input.OpportunityID, "CLOSED_LOST", "Critical issues in due diligence",
		).Get(ctx, nil)

		output.FinalStage = "CLOSED_LOST"
		output.FailureReason = "Critical issues in due diligence: " + reviewResults.Recommendation
		output.CompletedAt = workflow.Now(ctx)
		output.TotalDurationDays = int(output.CompletedAt.Sub(output.StartedAt).Hours() / 24)
		return output, nil
	}

	// ========================================================================
	// STAGE 4: INVESTMENT COMMITTEE
	// ========================================================================
	stageStart = workflow.Now(ctx)
	logger.Info("Stage 4: Investment committee review")

	_ = workflow.ExecuteActivity(quickCtx, UpdateOpportunityStageActivity,
		input.OpportunityID, "INVESTMENT_COMMITTEE", "Due diligence complete - submitted to committee",
	).Get(ctx, nil)

	// Generate committee package with AI summary
	_ = workflow.ExecuteActivity(longCtx, GenerateEnhancedCommitteePackageActivity,
		input.OpportunityID, output.ScreeningResult, output.ReviewResults,
	).Get(ctx, nil)

	// Wait for committee decision
	var committeeDecision CommitteeDecisionResult
	committeeSelector := workflow.NewSelector(ctx)

	committeeCh := workflow.GetSignalChannel(ctx, "committee_decision")
	committeeSelector.AddReceive(committeeCh, func(c workflow.ReceiveChannel, more bool) {
		c.Receive(ctx, &committeeDecision)
	})

	committeeTimeout := workflow.NewTimer(ctx, input.Config.CommitteeReviewTimeout)
	committeeSelector.AddFuture(committeeTimeout, func(f workflow.Future) {
		committeeDecision = CommitteeDecisionResult{Decision: "DEFERRED"}
	})

	committeeSelector.Select(ctx)

	output.CommitteeDecision = &committeeDecision
	output.StageDurations["committee"] = workflow.Now(ctx).Sub(stageStart)

	addAuditEntry(output, "COMMITTEE_DECISION", "INVESTMENT_COMMITTEE", uuid.Nil, map[string]interface{}{
		"decision":        committeeDecision.Decision,
		"approved_amount": committeeDecision.ApprovedAmount,
		"conditions":      committeeDecision.Conditions,
	})

	switch committeeDecision.Decision {
	case "REJECTED":
		_ = workflow.ExecuteActivity(quickCtx, UpdateOpportunityStageActivity,
			input.OpportunityID, "CLOSED_LOST", "Committee rejected",
		).Get(ctx, nil)
		output.FinalStage = "CLOSED_LOST"
		output.FailureReason = "Committee rejected"

	case "DEFERRED":
		_ = workflow.ExecuteActivity(quickCtx, UpdateOpportunityStageActivity,
			input.OpportunityID, "ON_HOLD", "Committee deferred decision",
		).Get(ctx, nil)
		output.FinalStage = "ON_HOLD"

	default: // APPROVED or APPROVED_WITH_CONDITIONS
		output.ApprovedAmount = committeeDecision.ApprovedAmount

		// ====================================================================
		// STAGE 5: DOCUMENTATION
		// ====================================================================
		stageStart = workflow.Now(ctx)
		logger.Info("Stage 5: Documentation and signatures")

		_ = workflow.ExecuteActivity(quickCtx, UpdateOpportunityStageActivity,
			input.OpportunityID, "DOCUMENTATION", "Approved - preparing documents",
		).Get(ctx, nil)

		// Generate documents
		_ = workflow.ExecuteActivity(longCtx, GenerateCommitmentDocumentsActivity,
			input.OpportunityID, committeeDecision.ApprovedAmount,
		).Get(ctx, nil)

		// E-signature workflow
		var esigRequestID uuid.UUID
		_ = workflow.ExecuteActivity(standardCtx, InitiateESignatureWorkflowActivity,
			input.OpportunityID, input.ClientID,
		).Get(ctx, &esigRequestID)

		// Wait for signatures
		var esigStatus ESignatureStatus
		esigSelector := workflow.NewSelector(ctx)

		esigCh := workflow.GetSignalChannel(ctx, "esignature_complete")
		esigSelector.AddReceive(esigCh, func(c workflow.ReceiveChannel, more bool) {
			c.Receive(ctx, &esigStatus)
		})

		esigTimeout := workflow.NewTimer(ctx, input.Config.DocumentSigningTimeout)
		esigSelector.AddFuture(esigTimeout, func(f workflow.Future) {
			esigStatus = ESignatureStatus{AllSigned: false, PendingSigners: []string{"Timeout"}}
		})

		esigSelector.Select(ctx)

		output.DocumentsCompleted = esigStatus.AllSigned
		output.StageDurations["documentation"] = workflow.Now(ctx).Sub(stageStart)

		// ====================================================================
		// STAGE 6: COMMITMENT
		// ====================================================================
		logger.Info("Stage 6: Processing commitment")

		_ = workflow.ExecuteActivity(quickCtx, UpdateOpportunityStageActivity,
			input.OpportunityID, "COMMITTED", "Documents signed - processing commitment",
		).Get(ctx, nil)

		_ = workflow.ExecuteActivity(standardCtx, ProcessCapitalCommitmentActivity,
			input.OpportunityID, input.ClientID, committeeDecision.ApprovedAmount,
		).Get(ctx, nil)

		_ = workflow.ExecuteActivity(standardCtx, OnboardToPortfolioActivity,
			input.OpportunityID, input.ClientID,
		).Get(ctx, nil)

		// Notify client if configured
		if input.Config.NotifyClientOnApproval {
			_ = workflow.ExecuteActivity(quickCtx, NotifyClientCommitmentActivity,
				input.ClientID, input.OpportunityID, committeeDecision.ApprovedAmount,
			).Get(ctx, nil)
		}

		_ = workflow.ExecuteActivity(quickCtx, UpdateOpportunityStageActivity,
			input.OpportunityID, "CLOSED_WON", "Investment committed and onboarded",
		).Get(ctx, nil)

		output.Success = true
		output.FinalStage = "CLOSED_WON"
	}

	// Calculate final metrics
	output.CompletedAt = workflow.Now(ctx)
	output.TotalDurationDays = int(output.CompletedAt.Sub(output.StartedAt).Hours() / 24)

	// Check SLA
	if input.Config.SLAEnabled {
		targetDays := int(input.Config.TargetCycleTime.Hours() / 24)
		output.SLAMet = output.TotalDurationDays <= targetDays
		if !output.SLAMet {
			output.SLABreachReason = fmt.Sprintf("Completed in %d days vs target of %d days",
				output.TotalDurationDays, targetDays)
		}
	}

	addAuditEntry(output, "WORKFLOW_COMPLETED", output.FinalStage, uuid.Nil, map[string]interface{}{
		"success":             output.Success,
		"total_duration_days": output.TotalDurationDays,
		"sla_met":             output.SLAMet,
		"approved_amount":     output.ApprovedAmount,
	})

	logger.Info("Enterprise due diligence workflow completed",
		"opportunity_id", input.OpportunityID,
		"success", output.Success,
		"final_stage", output.FinalStage,
		"duration_days", output.TotalDurationDays,
		"sla_met", output.SLAMet,
	)

	return output, nil
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func addAuditEntry(output *EnhancedDueDiligenceOutput, action, stage string, performedBy uuid.UUID, details map[string]interface{}) {
	entry := AuditEntry{
		Timestamp:   time.Now(),
		Action:      action,
		Stage:       stage,
		PerformedBy: performedBy,
		Details:     details,
	}
	output.AuditTrail = append(output.AuditTrail, entry)
}

func createFailedReview(reviewType string, err error) EnhancedReviewResult {
	return EnhancedReviewResult{
		ReviewType: reviewType,
		Passed:     false,
		Score:      0,
		RiskLevel:  "CRITICAL",
		Findings: []Finding{{
			FindingID:   uuid.New(),
			Category:    "SYSTEM",
			Severity:    "CRITICAL",
			Title:       "Review Failed",
			Description: err.Error(),
			Status:      "OPEN",
		}},
		CompletedAt: time.Now(),
	}
}

func calculateAggregateReviewScores(results *EnhancedReviewResults) {
	var totalScore float64
	var count int
	var highestRisk string = "LOW"

	riskLevels := map[string]int{"LOW": 1, "MEDIUM": 2, "HIGH": 3, "CRITICAL": 4}

	reviews := []*EnhancedReviewResult{
		results.RiskAssessment,
		results.LegalReview,
		results.TaxAnalysis,
		results.OperationalDD,
		results.ESGReview,
		results.ReferenceChecks,
		results.ConflictCheck,
	}

	for _, r := range reviews {
		if r != nil {
			totalScore += r.Score
			count++
			if riskLevels[r.RiskLevel] > riskLevels[highestRisk] {
				highestRisk = r.RiskLevel
			}
		}
	}

	if count > 0 {
		results.OverallScore = totalScore / float64(count)
	}
	results.OverallRiskLevel = highestRisk

	// Set recommendation based on aggregate
	switch {
	case highestRisk == "CRITICAL":
		results.Recommendation = "DO_NOT_PROCEED"
	case highestRisk == "HIGH":
		results.Recommendation = "PROCEED_WITH_CONDITIONS"
	case results.OverallScore >= 80:
		results.Recommendation = "STRONG_PROCEED"
	case results.OverallScore >= 60:
		results.Recommendation = "PROCEED"
	default:
		results.Recommendation = "RECONSIDER"
	}

	// Collect must-address items
	for _, r := range reviews {
		if r != nil {
			for _, f := range r.Findings {
				if f.Severity == "CRITICAL" || f.Severity == "HIGH" {
					results.MustAddress = append(results.MustAddress, f.Title)
				} else if f.Severity == "MEDIUM" {
					results.ShouldConsider = append(results.ShouldConsider, f.Title)
				}
			}
		}
	}
}
