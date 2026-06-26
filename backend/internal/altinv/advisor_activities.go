package altinv

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.temporal.io/sdk/activity"
)

// ============================================================================
// ADVISOR WORKFLOW ACTIVITIES
// Activities for the Alternative Investment Advisor Workflows
// ============================================================================

// AdvisorWorkflowActivities contains all activity implementations
type AdvisorWorkflowActivities struct {
	db *sqlx.DB
}

// NewAdvisorWorkflowActivities creates a new activities instance
func NewAdvisorWorkflowActivities(db *sqlx.DB) *AdvisorWorkflowActivities {
	return &AdvisorWorkflowActivities{db: db}
}

// ============================================================================
// SCREENING ACTIVITIES
// ============================================================================

// RunAutomatedScreeningActivity performs automated screening on an opportunity
func (a *AdvisorWorkflowActivities) RunAutomatedScreeningActivity(ctx context.Context, opportunityID uuid.UUID) (*ScreeningResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Running automated screening", "opportunity_id", opportunityID)

	result := &ScreeningResult{
		Passed:  true,
		Score:   100,
		Reasons: []string{},
	}

	// Fetch opportunity details
	var opp struct {
		MinimumCommitment   float64   `db:"minimum_commitment"`
		VintageYear         *int      `db:"vintage_year"`
		TargetIRRMin        *float64  `db:"target_irr_min"`
		TrackRecordYearsMin *int      `db:"track_record_years_min"`
		MaxLeverageRatio    *float64  `db:"max_leverage_ratio"`
		ClientID            uuid.UUID `db:"client_id"`
	}

	err := a.db.GetContext(ctx, &opp, `
		SELECT minimum_commitment, vintage_year, target_irr_min, 
		       track_record_years_min, max_leverage_ratio, client_id
		FROM investment_opportunities
		WHERE opportunity_id = $1
	`, opportunityID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch opportunity: %w", err)
	}

	// Get client's total alternative AUM
	var clientAltAUM float64
	err = a.db.GetContext(ctx, &clientAltAUM, `
		SELECT COALESCE(SUM(current_nav), 0)
		FROM alternative_investments
		WHERE client_id = $1
	`, opp.ClientID)
	if err != nil {
		logger.Warn("Could not fetch client AUM", "error", err)
	}

	// Screening Check 1: Minimum commitment vs client capacity (10% rule)
	if clientAltAUM > 0 && opp.MinimumCommitment > clientAltAUM*0.10 {
		result.Reasons = append(result.Reasons,
			fmt.Sprintf("Minimum commitment ($%.2f) exceeds 10%% of alternative AUM ($%.2f)",
				opp.MinimumCommitment, clientAltAUM*0.10))
		result.Score -= 25
	}

	// Screening Check 2: Vintage year (prefer 2025-2028)
	if opp.VintageYear != nil && (*opp.VintageYear < 2025 || *opp.VintageYear > 2028) {
		result.Reasons = append(result.Reasons,
			fmt.Sprintf("Vintage year %d outside preferred 2025-2028 range", *opp.VintageYear))
		result.Score -= 10
	}

	// Screening Check 3: Track record (minimum 5 years)
	if opp.TrackRecordYearsMin != nil && *opp.TrackRecordYearsMin < 5 {
		result.Reasons = append(result.Reasons,
			fmt.Sprintf("Manager track record (%d years) below 5-year minimum", *opp.TrackRecordYearsMin))
		result.Score -= 15
	}

	// Screening Check 4: Target IRR reasonableness
	if opp.TargetIRRMin != nil && *opp.TargetIRRMin > 35 {
		result.Reasons = append(result.Reasons,
			fmt.Sprintf("Target IRR (%.1f%%) appears unrealistically high", *opp.TargetIRRMin))
		result.Score -= 10
	}

	// Screening Check 5: Leverage
	if opp.MaxLeverageRatio != nil && *opp.MaxLeverageRatio > 2.0 {
		result.Reasons = append(result.Reasons,
			fmt.Sprintf("Leverage ratio (%.1fx) exceeds 2x limit", *opp.MaxLeverageRatio))
		result.Score -= 20
	}

	// Determine pass/fail based on score
	result.Passed = result.Score >= 60

	// Update screening results in database
	_, err = a.db.ExecContext(ctx, `
		UPDATE investment_opportunities
		SET screening_passed = $2,
		    screening_score = $3,
		    screening_reasons = $4,
		    screening_completed_at = NOW(),
		    updated_at = NOW()
		WHERE opportunity_id = $1
	`, opportunityID, result.Passed, result.Score, result.Reasons)
	if err != nil {
		logger.Warn("Failed to update screening results", "error", err)
	}

	return result, nil
}

// ============================================================================
// NOTIFICATION ACTIVITIES
// ============================================================================

// NotifyAdvisorScreeningFailedActivity notifies advisor of failed screening
func (a *AdvisorWorkflowActivities) NotifyAdvisorScreeningFailedActivity(
	ctx context.Context,
	opportunityID uuid.UUID,
	advisorID uuid.UUID,
	result ScreeningResult,
) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Notifying advisor of screening failure",
		"opportunity_id", opportunityID,
		"advisor_id", advisorID,
	)

	// Create notification record
	_, err := a.db.ExecContext(ctx, `
		INSERT INTO notifications (
			user_id, notification_type, title, message, 
			related_entity_type, related_entity_id, metadata
		) VALUES (
			$1, 'SCREENING_FAILED', 'Investment Opportunity Screening Failed',
			$2, 'INVESTMENT_OPPORTUNITY', $3, $4
		)
	`, advisorID,
		fmt.Sprintf("Screening failed with score %.0f. Reasons: %v", result.Score, result.Reasons),
		opportunityID,
		result)

	if err != nil {
		logger.Warn("Failed to create notification", "error", err)
	}

	return nil
}

// NotifyAdvisorForReviewActivity notifies advisor to review an opportunity
func (a *AdvisorWorkflowActivities) NotifyAdvisorForReviewActivity(
	ctx context.Context,
	opportunityID uuid.UUID,
	advisorID uuid.UUID,
) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Notifying advisor for review",
		"opportunity_id", opportunityID,
		"advisor_id", advisorID,
	)

	// Get opportunity details
	var fundName string
	err := a.db.GetContext(ctx, &fundName, `
		SELECT fund_name FROM investment_opportunities WHERE opportunity_id = $1
	`, opportunityID)
	if err != nil {
		fundName = "Unknown Fund"
	}

	// Create notification
	_, err = a.db.ExecContext(ctx, `
		INSERT INTO notifications (
			user_id, notification_type, title, message,
			related_entity_type, related_entity_id
		) VALUES (
			$1, 'REVIEW_REQUIRED', 'Investment Opportunity Ready for Review',
			$2, 'INVESTMENT_OPPORTUNITY', $3
		)
	`, advisorID,
		fmt.Sprintf("Investment opportunity '%s' has passed automated screening and is ready for your review.", fundName),
		opportunityID)

	if err != nil {
		logger.Warn("Failed to create notification", "error", err)
	}

	return nil
}

// NotifyCommitteeRejectionActivity notifies of committee review issues
func (a *AdvisorWorkflowActivities) NotifyCommitteeRejectionActivity(
	ctx context.Context,
	opportunityID uuid.UUID,
	failures []string,
) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Notifying of due diligence issues", "opportunity_id", opportunityID)

	// Store the failures for committee package
	_, err := a.db.ExecContext(ctx, `
		UPDATE investment_opportunities
		SET risk_assessment_notes = $2,
		    updated_at = NOW()
		WHERE opportunity_id = $1
	`, opportunityID, fmt.Sprintf("DD Issues: %v", failures))

	if err != nil {
		logger.Warn("Failed to update risk notes", "error", err)
	}

	return nil
}

// ============================================================================
// STAGE UPDATE ACTIVITIES
// ============================================================================

// UpdateOpportunityStageActivity updates opportunity stage
func (a *AdvisorWorkflowActivities) UpdateOpportunityStageActivity(
	ctx context.Context,
	opportunityID uuid.UUID,
	newStage string,
	notes string,
) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Updating opportunity stage",
		"opportunity_id", opportunityID,
		"new_stage", newStage,
	)

	// Get current stage history
	var stageHistoryJSON []byte
	err := a.db.GetContext(ctx, &stageHistoryJSON, `
		SELECT COALESCE(stage_history, '[]'::jsonb)
		FROM investment_opportunities
		WHERE opportunity_id = $1
	`, opportunityID)
	if err != nil {
		return fmt.Errorf("failed to get stage history: %w", err)
	}

	var stageHistory []map[string]interface{}
	json.Unmarshal(stageHistoryJSON, &stageHistory)

	// Add new stage entry
	stageHistory = append(stageHistory, map[string]interface{}{
		"stage":     newStage,
		"timestamp": time.Now(),
		"notes":     notes,
	})

	newHistoryJSON, _ := json.Marshal(stageHistory)

	// Update the opportunity
	_, err = a.db.ExecContext(ctx, `
		UPDATE investment_opportunities
		SET current_stage = $2,
		    stage_updated_at = NOW(),
		    stage_history = $3,
		    updated_at = NOW()
		WHERE opportunity_id = $1
	`, opportunityID, newStage, newHistoryJSON)

	if err != nil {
		return fmt.Errorf("failed to update stage: %w", err)
	}

	return nil
}

// ============================================================================
// TASK ACTIVITIES
// ============================================================================

// CreateAdvisorTaskActivity creates a task for an advisor
func (a *AdvisorWorkflowActivities) CreateAdvisorTaskActivity(
	ctx context.Context,
	advisorID uuid.UUID,
	opportunityID uuid.UUID,
	taskType string,
	description string,
) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Creating advisor task",
		"advisor_id", advisorID,
		"task_type", taskType,
	)

	// Get fund name for title
	var fundName string
	a.db.GetContext(ctx, &fundName, `
		SELECT fund_name FROM investment_opportunities WHERE opportunity_id = $1
	`, opportunityID)

	title := fmt.Sprintf("Review: %s", fundName)
	if fundName == "" {
		title = "Investment Opportunity Review"
	}

	// Set due date based on task type
	dueDate := time.Now().Add(7 * 24 * time.Hour) // Default 7 days
	priority := "MEDIUM"

	switch taskType {
	case "CAPITAL_CALL_FUNDING":
		dueDate = time.Now().Add(3 * 24 * time.Hour)
		priority = "HIGH"
	case "OPPORTUNITY_REVIEW":
		dueDate = time.Now().Add(14 * 24 * time.Hour)
	}

	_, err := a.db.ExecContext(ctx, `
		INSERT INTO advisor_tasks (
			advisor_id, opportunity_id, task_type, title, description,
			priority, due_date, status, auto_generated
		) VALUES ($1, $2, $3, $4, $5, $6, $7, 'PENDING', TRUE)
	`, advisorID, opportunityID, taskType, title, description, priority, dueDate)

	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	return nil
}

// ============================================================================
// DUE DILIGENCE ACTIVITIES
// ============================================================================

// InitializeDueDiligenceChecklistActivity initializes DD checklist from template
func (a *AdvisorWorkflowActivities) InitializeDueDiligenceChecklistActivity(
	ctx context.Context,
	opportunityID uuid.UUID,
) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Initializing due diligence checklist", "opportunity_id", opportunityID)

	// Get opportunity type
	var oppType string
	err := a.db.GetContext(ctx, &oppType, `
		SELECT opportunity_type FROM investment_opportunities WHERE opportunity_id = $1
	`, opportunityID)
	if err != nil {
		return fmt.Errorf("failed to get opportunity type: %w", err)
	}

	// Get template for this type
	var templateItems json.RawMessage
	err = a.db.GetContext(ctx, &templateItems, `
		SELECT items FROM due_diligence_templates
		WHERE opportunity_type = $1 AND is_default = TRUE
		LIMIT 1
	`, oppType)
	if err != nil {
		// Use generic template
		templateItems = json.RawMessage(`[
			{"category": "LEGAL", "item_name": "Fund Documents Review", "description": "Review LPA and subscription documents", "required": true},
			{"category": "FINANCIAL", "item_name": "Performance Analysis", "description": "Analyze historical returns", "required": true},
			{"category": "OPERATIONAL", "item_name": "Operational DD", "description": "Review operations and controls", "required": true},
			{"category": "TAX", "item_name": "Tax Analysis", "description": "Review tax implications", "required": true}
		]`)
	}

	// Parse template items
	var items []map[string]interface{}
	json.Unmarshal(templateItems, &items)

	// Insert DD items
	for _, item := range items {
		_, err := a.db.ExecContext(ctx, `
			INSERT INTO due_diligence_items (
				opportunity_id, category, item_name, description, required, status
			) VALUES ($1, $2, $3, $4, $5, 'PENDING')
		`, opportunityID, item["category"], item["item_name"], item["description"], item["required"])
		if err != nil {
			logger.Warn("Failed to insert DD item", "error", err, "item", item["item_name"])
		}
	}

	return nil
}

// RiskAssessmentActivity performs risk assessment
func (a *AdvisorWorkflowActivities) RiskAssessmentActivity(
	ctx context.Context,
	opportunityID uuid.UUID,
) (*ReviewResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Performing risk assessment", "opportunity_id", opportunityID)

	result := &ReviewResult{
		Passed:    true,
		Score:     85,
		RiskLevel: "MEDIUM",
		Findings:  []string{},
	}

	// Fetch opportunity for analysis
	var opp struct {
		OpportunityType     string   `db:"opportunity_type"`
		Strategy            *string  `db:"strategy"`
		MaxLeverageRatio    *float64 `db:"max_leverage_ratio"`
		MinimumCommitment   float64  `db:"minimum_commitment"`
		ManagementFeeRate   *float64 `db:"management_fee_rate"`
		CarriedInterestRate *float64 `db:"carried_interest_rate"`
	}

	err := a.db.GetContext(ctx, &opp, `
		SELECT opportunity_type, strategy, max_leverage_ratio, 
		       minimum_commitment, management_fee_rate, carried_interest_rate
		FROM investment_opportunities
		WHERE opportunity_id = $1
	`, opportunityID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch opportunity: %w", err)
	}

	// Risk checks
	if opp.MaxLeverageRatio != nil && *opp.MaxLeverageRatio > 1.5 {
		result.Findings = append(result.Findings, "High leverage ratio detected")
		result.Score -= 15
	}

	if opp.ManagementFeeRate != nil && *opp.ManagementFeeRate > 0.02 {
		result.Findings = append(result.Findings, "Management fee above 2%")
		result.Score -= 5
	}

	// Determine risk level
	if result.Score >= 80 {
		result.RiskLevel = "LOW"
	} else if result.Score >= 60 {
		result.RiskLevel = "MEDIUM"
	} else if result.Score >= 40 {
		result.RiskLevel = "HIGH"
		result.Passed = false
	} else {
		result.RiskLevel = "CRITICAL"
		result.Passed = false
	}

	result.ReviewedAt = time.Now()
	return result, nil
}

// LegalComplianceReviewActivity performs legal review
func (a *AdvisorWorkflowActivities) LegalComplianceReviewActivity(
	ctx context.Context,
	opportunityID uuid.UUID,
) (*ReviewResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Performing legal compliance review", "opportunity_id", opportunityID)

	result := &ReviewResult{
		Passed:     true,
		Score:      90,
		RiskLevel:  "LOW",
		Findings:   []string{},
		ReviewedAt: time.Now(),
	}

	// Check for required documents
	var hasLPA, hasPPM bool
	err := a.db.GetContext(ctx, &hasLPA, `
		SELECT subscription_agreement_url IS NOT NULL
		FROM investment_opportunities WHERE opportunity_id = $1
	`, opportunityID)
	if err == nil && !hasLPA {
		result.Findings = append(result.Findings, "Missing subscription agreement")
		result.Score -= 20
	}

	err = a.db.GetContext(ctx, &hasPPM, `
		SELECT private_placement_memorandum_url IS NOT NULL
		FROM investment_opportunities WHERE opportunity_id = $1
	`, opportunityID)
	if err == nil && !hasPPM {
		result.Findings = append(result.Findings, "Missing private placement memorandum")
		result.Score -= 15
	}

	result.Passed = result.Score >= 60
	if result.Score < 70 {
		result.RiskLevel = "MEDIUM"
	}

	return result, nil
}

// TaxImpactAnalysisActivity performs tax analysis
func (a *AdvisorWorkflowActivities) TaxImpactAnalysisActivity(
	ctx context.Context,
	opportunityID uuid.UUID,
) (*ReviewResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Performing tax impact analysis", "opportunity_id", opportunityID)

	result := &ReviewResult{
		Passed:     true,
		Score:      85,
		RiskLevel:  "LOW",
		Findings:   []string{},
		ReviewedAt: time.Now(),
	}

	// Get opportunity type to check tax implications
	var oppType string
	a.db.GetContext(ctx, &oppType, `
		SELECT opportunity_type FROM investment_opportunities WHERE opportunity_id = $1
	`, opportunityID)

	// Tax considerations by type
	switch oppType {
	case "PRIVATE_EQUITY", "VENTURE_CAPITAL":
		result.Findings = append(result.Findings, "K-1 tax reporting required")
		result.Conditions = append(result.Conditions, "Verify client's tax-exempt status for UBTI")
	case "HEDGE_FUND":
		result.Findings = append(result.Findings, "Complex tax reporting expected")
		result.Conditions = append(result.Conditions, "Review PFIC implications if investing internationally")
	case "REAL_ESTATE":
		result.Findings = append(result.Findings, "State tax nexus may be created")
		result.Conditions = append(result.Conditions, "Verify depreciation recapture implications")
	}

	return result, nil
}

// OperationalDueDiligenceActivity performs operational DD
func (a *AdvisorWorkflowActivities) OperationalDueDiligenceActivity(
	ctx context.Context,
	opportunityID uuid.UUID,
) (*ReviewResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Performing operational due diligence", "opportunity_id", opportunityID)

	result := &ReviewResult{
		Passed:     true,
		Score:      88,
		RiskLevel:  "LOW",
		Findings:   []string{},
		ReviewedAt: time.Now(),
	}

	// Get GP/Manager info
	var gp *string
	a.db.GetContext(ctx, &gp, `
		SELECT general_partner FROM investment_opportunities WHERE opportunity_id = $1
	`, opportunityID)

	if gp == nil || *gp == "" {
		result.Findings = append(result.Findings, "General partner information incomplete")
		result.Score -= 10
	}

	// Standard ODD findings
	result.Conditions = append(result.Conditions,
		"Verify fund administrator independence",
		"Confirm SOC 1/2 report availability",
	)

	return result, nil
}

// ============================================================================
// COMMITTEE ACTIVITIES
// ============================================================================

// GenerateCommitteePackageActivity generates investment committee materials
func (a *AdvisorWorkflowActivities) GenerateCommitteePackageActivity(
	ctx context.Context,
	opportunityID uuid.UUID,
	reviewResults *ParallelReviewResults,
) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Generating committee package", "opportunity_id", opportunityID)

	// Create committee review record
	_, err := a.db.ExecContext(ctx, `
		INSERT INTO investment_committee_reviews (
			opportunity_id, package_prepared_at, risk_assessment
		) VALUES ($1, NOW(), $2)
		ON CONFLICT (opportunity_id) DO UPDATE
		SET package_prepared_at = NOW(),
		    risk_assessment = $2
	`, opportunityID, reviewResults)

	if err != nil {
		return fmt.Errorf("failed to create committee review: %w", err)
	}

	return nil
}

// ============================================================================
// DOCUMENT ACTIVITIES
// ============================================================================

// GenerateCommitmentDocumentsActivity generates commitment documents
func (a *AdvisorWorkflowActivities) GenerateCommitmentDocumentsActivity(
	ctx context.Context,
	opportunityID uuid.UUID,
	approvedAmount float64,
) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Generating commitment documents",
		"opportunity_id", opportunityID,
		"amount", approvedAmount,
	)

	// In production, this would call a document generation service
	// For now, we just update the opportunity with the approved amount

	_, err := a.db.ExecContext(ctx, `
		UPDATE investment_opportunities
		SET target_commitment = $2,
		    updated_at = NOW()
		WHERE opportunity_id = $1
	`, opportunityID, approvedAmount)

	if err != nil {
		return fmt.Errorf("failed to update commitment amount: %w", err)
	}

	return nil
}

// InitiateESignatureWorkflowActivity initiates document signing
func (a *AdvisorWorkflowActivities) InitiateESignatureWorkflowActivity(
	ctx context.Context,
	opportunityID uuid.UUID,
	clientID uuid.UUID,
) (uuid.UUID, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Initiating e-signature workflow", "opportunity_id", opportunityID)

	requestID := uuid.New()

	// Create e-signature request record
	_, err := a.db.ExecContext(ctx, `
		INSERT INTO esignature_requests (
			request_id, opportunity_id, document_type, document_name,
			signers, status, created_at
		) VALUES ($1, $2, 'SUBSCRIPTION_AGREEMENT', 'Subscription Agreement',
			$3, 'SENT', NOW())
	`, requestID, opportunityID, json.RawMessage(`[{"client_id": "`+clientID.String()+`", "status": "PENDING"}]`))

	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create e-signature request: %w", err)
	}

	return requestID, nil
}

// ============================================================================
// COMMITMENT ACTIVITIES
// ============================================================================

// ProcessCapitalCommitmentActivity records the capital commitment
func (a *AdvisorWorkflowActivities) ProcessCapitalCommitmentActivity(
	ctx context.Context,
	opportunityID uuid.UUID,
	clientID uuid.UUID,
	amount float64,
) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Processing capital commitment",
		"opportunity_id", opportunityID,
		"amount", amount,
	)

	// Get opportunity details for the new investment record
	var opp struct {
		FundName        string  `db:"fund_name"`
		OpportunityType string  `db:"opportunity_type"`
		GeneralPartner  *string `db:"general_partner"`
		VintageYear     *int    `db:"vintage_year"`
	}

	err := a.db.GetContext(ctx, &opp, `
		SELECT fund_name, opportunity_type, general_partner, vintage_year
		FROM investment_opportunities WHERE opportunity_id = $1
	`, opportunityID)
	if err != nil {
		return fmt.Errorf("failed to fetch opportunity: %w", err)
	}

	// Create alternative investment record
	_, err = a.db.ExecContext(ctx, `
		INSERT INTO alternative_investments (
			client_id, investment_type, fund_name, general_partner,
			vintage_year, total_commitment_amount, unfunded_commitment,
			total_capital_called, total_distributions, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $6, 0, 0, $7)
	`, clientID, opp.OpportunityType, opp.FundName, opp.GeneralPartner,
		opp.VintageYear, amount,
		json.RawMessage(fmt.Sprintf(`{"source_opportunity_id": "%s"}`, opportunityID)))

	if err != nil {
		return fmt.Errorf("failed to create investment record: %w", err)
	}

	return nil
}

// OnboardToPortfolioActivity onboards the investment to the client's portfolio
func (a *AdvisorWorkflowActivities) OnboardToPortfolioActivity(
	ctx context.Context,
	opportunityID uuid.UUID,
	clientID uuid.UUID,
) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Onboarding to portfolio", "opportunity_id", opportunityID)

	// Create initial quarterly review schedule
	now := time.Now()
	quarter := (now.Month()-1)/3 + 1
	reviewPeriod := fmt.Sprintf("%d-Q%d", now.Year(), quarter)

	_, err := a.db.ExecContext(ctx, `
		INSERT INTO quarterly_reviews (
			client_id, review_period, period_start, period_end, status
		) VALUES ($1, $2, $3, $4, 'PENDING')
		ON CONFLICT DO NOTHING
	`, clientID, reviewPeriod,
		time.Date(now.Year(), time.Month((quarter-1)*3+1), 1, 0, 0, 0, 0, time.UTC),
		time.Date(now.Year(), time.Month(quarter*3), 1, 0, 0, 0, 0, time.UTC).Add(-24*time.Hour))

	if err != nil {
		logger.Warn("Failed to create quarterly review", "error", err)
	}

	return nil
}

// ============================================================================
// QUARTERLY REVIEW ACTIVITIES
// ============================================================================

// CreateQuarterlyReviewRecordActivity creates a quarterly review record
func (a *AdvisorWorkflowActivities) CreateQuarterlyReviewRecordActivity(
	ctx context.Context,
	clientID uuid.UUID,
	reviewPeriod string,
) (uuid.UUID, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Creating quarterly review record",
		"client_id", clientID,
		"period", reviewPeriod,
	)

	reviewID := uuid.New()

	// Parse period to get dates
	var year, quarter int
	fmt.Sscanf(reviewPeriod, "%d-Q%d", &year, &quarter)

	periodStart := time.Date(year, time.Month((quarter-1)*3+1), 1, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.AddDate(0, 3, -1)

	_, err := a.db.ExecContext(ctx, `
		INSERT INTO quarterly_reviews (
			review_id, client_id, review_period, period_start, period_end, status
		) VALUES ($1, $2, $3, $4, $5, 'IN_PROGRESS')
	`, reviewID, clientID, reviewPeriod, periodStart, periodEnd)

	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create review record: %w", err)
	}

	return reviewID, nil
}

// CalculatePortfolioPerformanceActivity calculates portfolio performance
func (a *AdvisorWorkflowActivities) CalculatePortfolioPerformanceActivity(
	ctx context.Context,
	clientID uuid.UUID,
	reviewPeriod string,
) (*PerformanceData, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Calculating portfolio performance", "client_id", clientID)

	performance := &PerformanceData{}

	// Get total alternative AUM
	err := a.db.GetContext(ctx, &performance.TotalAltAUM, `
		SELECT COALESCE(SUM(current_nav), 0)
		FROM alternative_investments
		WHERE client_id = $1
	`, clientID)
	if err != nil {
		logger.Warn("Failed to get AUM", "error", err)
	}

	// Get average performance metrics
	var metrics struct {
		AvgIRR  *float64 `db:"avg_irr"`
		AvgTVPI *float64 `db:"avg_tvpi"`
		AvgDPI  *float64 `db:"avg_dpi"`
	}
	err = a.db.GetContext(ctx, &metrics, `
		SELECT 
			AVG(irr_since_inception) as avg_irr,
			AVG(tvpi) as avg_tvpi,
			AVG(dpi) as avg_dpi
		FROM alternative_investments
		WHERE client_id = $1 AND current_nav > 0
	`, clientID)

	if err == nil {
		if metrics.AvgIRR != nil {
			performance.TotalIRR = *metrics.AvgIRR
		}
		if metrics.AvgTVPI != nil {
			performance.TotalTVPI = *metrics.AvgTVPI
		}
		if metrics.AvgDPI != nil {
			performance.TotalDPI = *metrics.AvgDPI
		}
	}

	// Simplified return calculation (in production, use proper TWR/IRR)
	performance.AltPortfolioReturnPct = performance.TotalIRR
	performance.BenchmarkReturnPct = 8.0 // S&P benchmark placeholder
	performance.Alpha = performance.AltPortfolioReturnPct - performance.BenchmarkReturnPct

	return performance, nil
}

// LiquidityStressTestActivity performs liquidity analysis
func (a *AdvisorWorkflowActivities) LiquidityStressTestActivity(
	ctx context.Context,
	clientID uuid.UUID,
) (*LiquidityAnalysis, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Running liquidity stress test", "client_id", clientID)

	liquidity := &LiquidityAnalysis{}

	// Get unfunded commitments
	err := a.db.GetContext(ctx, &liquidity.TotalUnfundedCommitments, `
		SELECT COALESCE(SUM(unfunded_commitment), 0)
		FROM alternative_investments
		WHERE client_id = $1
	`, clientID)
	if err != nil {
		logger.Warn("Failed to get unfunded commitments", "error", err)
	}

	// Get upcoming capital calls (30 and 90 days)
	err = a.db.GetContext(ctx, &liquidity.UpcomingCalls30d, `
		SELECT COALESCE(SUM(amount), 0)
		FROM capital_events ce
		JOIN alternative_investments ai ON ce.investment_id = ai.investment_id
		WHERE ai.client_id = $1 
		  AND ce.event_type = 'CAPITAL_CALL'
		  AND ce.status = 'PENDING'
		  AND ce.due_date BETWEEN CURRENT_DATE AND CURRENT_DATE + INTERVAL '30 days'
	`, clientID)

	err = a.db.GetContext(ctx, &liquidity.UpcomingCalls90d, `
		SELECT COALESCE(SUM(amount), 0)
		FROM capital_events ce
		JOIN alternative_investments ai ON ce.investment_id = ai.investment_id
		WHERE ai.client_id = $1 
		  AND ce.event_type = 'CAPITAL_CALL'
		  AND ce.status = 'PENDING'
		  AND ce.due_date BETWEEN CURRENT_DATE AND CURRENT_DATE + INTERVAL '90 days'
	`, clientID)

	// Calculate dry powder percentage
	var totalCommitments float64
	a.db.GetContext(ctx, &totalCommitments, `
		SELECT COALESCE(SUM(total_commitment_amount), 0)
		FROM alternative_investments
		WHERE client_id = $1
	`, clientID)

	if totalCommitments > 0 {
		liquidity.DryPowderPct = (liquidity.TotalUnfundedCommitments / totalCommitments) * 100
	}

	// Placeholder for available liquidity (would need accounts data)
	liquidity.AvailableLiquidity = 1000000 // Placeholder
	if liquidity.UpcomingCalls90d > 0 {
		liquidity.LiquidityCoverageRatio = liquidity.AvailableLiquidity / liquidity.UpcomingCalls90d
	}

	return liquidity, nil
}

// FetchManagerUpdatesActivity fetches recent manager communications
func (a *AdvisorWorkflowActivities) FetchManagerUpdatesActivity(
	ctx context.Context,
	clientID uuid.UUID,
	reviewPeriod string,
) ([]ManagerUpdateSummary, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Fetching manager updates", "client_id", clientID)

	var updates []ManagerUpdateSummary

	rows, err := a.db.QueryxContext(ctx, `
		SELECT 
			mu.investment_id,
			ai.fund_name,
			mu.update_type,
			mu.title,
			mu.reported_nav,
			mu.reported_irr,
			mu.document_url
		FROM manager_updates mu
		JOIN alternative_investments ai ON mu.investment_id = ai.investment_id
		WHERE ai.client_id = $1
		  AND mu.document_date >= CURRENT_DATE - INTERVAL '90 days'
		ORDER BY mu.document_date DESC
	`, clientID)

	if err != nil {
		return updates, nil // Return empty if no updates
	}
	defer rows.Close()

	for rows.Next() {
		var update ManagerUpdateSummary
		if err := rows.StructScan(&update); err == nil {
			updates = append(updates, update)
		}
	}

	return updates, nil
}

// AssessPortfolioRisksActivity assesses portfolio risks
func (a *AdvisorWorkflowActivities) AssessPortfolioRisksActivity(
	ctx context.Context,
	clientID uuid.UUID,
	performance PerformanceData,
	liquidity LiquidityAnalysis,
	managerUpdates []ManagerUpdateSummary,
) ([]RiskFlag, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Assessing portfolio risks", "client_id", clientID)

	var flags []RiskFlag

	// Check concentration risk
	rows, _ := a.db.QueryxContext(ctx, `
		SELECT investment_id, fund_name, current_nav
		FROM alternative_investments
		WHERE client_id = $1 AND current_nav IS NOT NULL
		ORDER BY current_nav DESC
	`, clientID)

	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var inv struct {
				InvestmentID uuid.UUID `db:"investment_id"`
				FundName     string    `db:"fund_name"`
				CurrentNAV   float64   `db:"current_nav"`
			}
			if err := rows.StructScan(&inv); err == nil {
				// Flag if single position > 30% of alternatives
				if performance.TotalAltAUM > 0 && (inv.CurrentNAV/performance.TotalAltAUM) > 0.30 {
					flags = append(flags, RiskFlag{
						InvestmentID: inv.InvestmentID,
						FundName:     inv.FundName,
						FlagType:     "CONCENTRATION",
						Severity:     "MEDIUM",
						Description:  fmt.Sprintf("Position represents %.1f%% of alternatives portfolio", (inv.CurrentNAV/performance.TotalAltAUM)*100),
					})
				}
			}
		}
	}

	// Check liquidity risk
	if liquidity.LiquidityCoverageRatio < 1.0 {
		flags = append(flags, RiskFlag{
			FlagType:    "LIQUIDITY_CONCERN",
			Severity:    "HIGH",
			Description: fmt.Sprintf("Liquidity coverage ratio (%.2f) below 1.0x - insufficient to meet upcoming capital calls", liquidity.LiquidityCoverageRatio),
		})
	}

	// Check underperformance
	if performance.Alpha < -5 {
		flags = append(flags, RiskFlag{
			FlagType:    "UNDERPERFORMANCE",
			Severity:    "MEDIUM",
			Description: fmt.Sprintf("Portfolio underperforming benchmark by %.1f%%", -performance.Alpha),
		})
	}

	return flags, nil
}

// GenerateQuarterlyReportActivity generates the quarterly report
func (a *AdvisorWorkflowActivities) GenerateQuarterlyReportActivity(
	ctx context.Context,
	reviewID uuid.UUID,
	clientID uuid.UUID,
	performance PerformanceData,
	liquidity LiquidityAnalysis,
	riskFlags []RiskFlag,
	managerUpdates []ManagerUpdateSummary,
) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Generating quarterly report", "review_id", reviewID)

	// In production, this would generate a PDF report
	// For now, we create a summary and store it

	reportURL := fmt.Sprintf("/reports/quarterly/%s.pdf", reviewID)

	// Update review with report URL
	_, err := a.db.ExecContext(ctx, `
		UPDATE quarterly_reviews
		SET report_url = $2,
		    report_generated_at = NOW(),
		    portfolio_return_pct = $3,
		    alt_portfolio_return_pct = $4,
		    total_alt_aum = $5,
		    risk_flags = $6,
		    updated_at = NOW()
		WHERE review_id = $1
	`, reviewID, reportURL, performance.PortfolioReturnPct,
		performance.AltPortfolioReturnPct, performance.TotalAltAUM, riskFlags)

	if err != nil {
		return "", fmt.Errorf("failed to update review with report: %w", err)
	}

	return reportURL, nil
}

// ScheduleClientReviewMeetingActivity schedules a client meeting
func (a *AdvisorWorkflowActivities) ScheduleClientReviewMeetingActivity(
	ctx context.Context,
	clientID uuid.UUID,
	advisorID uuid.UUID,
	reviewID uuid.UUID,
	reportURL string,
) (time.Time, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Scheduling client review meeting", "client_id", clientID)

	// Default to 2 weeks from now
	meetingDate := time.Now().Add(14 * 24 * time.Hour)

	// Update review with meeting info
	_, err := a.db.ExecContext(ctx, `
		UPDATE quarterly_reviews
		SET meeting_scheduled_at = $2,
		    status = 'MEETING_SCHEDULED',
		    updated_at = NOW()
		WHERE review_id = $1
	`, reviewID, meetingDate)

	if err != nil {
		return time.Time{}, fmt.Errorf("failed to schedule meeting: %w", err)
	}

	// Create task for advisor
	a.CreateAdvisorTaskActivity(ctx, advisorID, uuid.Nil, "CLIENT_MEETING",
		fmt.Sprintf("Quarterly review meeting - report available at %s", reportURL))

	return meetingDate, nil
}

// EscalateRiskPositionsActivity escalates high-risk positions
func (a *AdvisorWorkflowActivities) EscalateRiskPositionsActivity(
	ctx context.Context,
	clientID uuid.UUID,
	advisorID uuid.UUID,
	riskFlags []RiskFlag,
) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Escalating risk positions", "client_id", clientID, "count", len(riskFlags))

	for _, flag := range riskFlags {
		_, err := a.db.ExecContext(ctx, `
			INSERT INTO notifications (
				user_id, notification_type, title, message,
				related_entity_type, related_entity_id
			) VALUES ($1, 'RISK_ALERT', $2, $3, 'CLIENT', $4)
		`, advisorID,
			fmt.Sprintf("Risk Alert: %s", flag.FlagType),
			flag.Description,
			clientID)

		if err != nil {
			logger.Warn("Failed to create risk notification", "error", err)
		}
	}

	return nil
}

// UpdateInvestmentCommitteeActivity updates committee with client status
func (a *AdvisorWorkflowActivities) UpdateInvestmentCommitteeActivity(
	ctx context.Context,
	clientID uuid.UUID,
	performance PerformanceData,
	riskFlags []RiskFlag,
) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Updating investment committee", "client_id", clientID)

	// In production, this would update a committee dashboard or send a summary
	// For now, we just log it
	logger.Info("Committee update",
		"total_alt_aum", performance.TotalAltAUM,
		"irr", performance.TotalIRR,
		"risk_flags", len(riskFlags),
	)

	return nil
}

// UpdateQuarterlyReviewStatusActivity updates review status
func (a *AdvisorWorkflowActivities) UpdateQuarterlyReviewStatusActivity(
	ctx context.Context,
	reviewID uuid.UUID,
	status string,
	reportURL string,
) error {
	_, err := a.db.ExecContext(ctx, `
		UPDATE quarterly_reviews
		SET status = $2,
		    report_url = $3,
		    updated_at = NOW()
		WHERE review_id = $1
	`, reviewID, status, reportURL)

	return err
}

// ============================================================================
// CAPITAL CALL ACTIVITIES
// ============================================================================

// IdentifyFundingSourceActivity identifies the best account to fund a capital call
func (a *AdvisorWorkflowActivities) IdentifyFundingSourceActivity(
	ctx context.Context,
	clientID uuid.UUID,
	amountRequired float64,
) (*FundingAccountResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Identifying funding source", "client_id", clientID, "amount", amountRequired)

	result := &FundingAccountResult{
		AccountID:         uuid.New(), // Placeholder
		AccountName:       "Primary Cash Account",
		AvailableBalance:  amountRequired * 1.5, // Placeholder - assume sufficient
		SufficientBalance: true,
	}

	// In production, query accounts table to find best funding source
	// For now, return placeholder

	return result, nil
}

// InitiateFundingTransferActivity initiates wire transfer
func (a *AdvisorWorkflowActivities) InitiateFundingTransferActivity(
	ctx context.Context,
	eventID uuid.UUID,
	accountID uuid.UUID,
	amount float64,
) (*TransferResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Initiating funding transfer", "event_id", eventID, "amount", amount)

	result := &TransferResult{
		TransferID:   uuid.New(),
		Status:       "INITIATED",
		Amount:       amount,
		InitiatedAt:  time.Now(),
		ExpectedDate: time.Now().Add(2 * 24 * time.Hour),
	}

	// Update capital event with funding source
	_, err := a.db.ExecContext(ctx, `
		UPDATE capital_events
		SET funding_source_account = $2,
		    status = 'SCHEDULED',
		    updated_at = NOW()
		WHERE event_id = $1
	`, eventID, accountID)

	if err != nil {
		return nil, fmt.Errorf("failed to update event: %w", err)
	}

	return result, nil
}

// UpdateCapitalEventStatusActivity updates capital event status
func (a *AdvisorWorkflowActivities) UpdateCapitalEventStatusActivity(
	ctx context.Context,
	eventID uuid.UUID,
	status string,
	amountFunded float64,
) error {
	_, err := a.db.ExecContext(ctx, `
		UPDATE capital_events
		SET status = $2,
		    amount_funded = $3,
		    processed_at = NOW(),
		    updated_at = NOW()
		WHERE event_id = $1
	`, eventID, status, amountFunded)

	return err
}

// SendCapitalCallConfirmationActivity sends funding confirmation
func (a *AdvisorWorkflowActivities) SendCapitalCallConfirmationActivity(
	ctx context.Context,
	eventID uuid.UUID,
	clientID uuid.UUID,
	amount float64,
) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Sending capital call confirmation", "event_id", eventID)

	// In production, this would send email/notification
	return nil
}

// ============================================================================
// DRIFT MONITORING ACTIVITIES
// ============================================================================

// GetClientsWithAllocationTargetsActivity gets clients with targets
func (a *AdvisorWorkflowActivities) GetClientsWithAllocationTargetsActivity(
	ctx context.Context,
) ([]uuid.UUID, error) {
	var clients []uuid.UUID

	rows, err := a.db.QueryxContext(ctx, `
		SELECT DISTINCT client_id 
		FROM client_allocation_targets
		WHERE effective_date <= CURRENT_DATE
		  AND (end_date IS NULL OR end_date > CURRENT_DATE)
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var clientID uuid.UUID
		if err := rows.Scan(&clientID); err == nil {
			clients = append(clients, clientID)
		}
	}

	return clients, nil
}

// CheckClientAllocationDriftActivity checks drift for a client
func (a *AdvisorWorkflowActivities) CheckClientAllocationDriftActivity(
	ctx context.Context,
	clientID uuid.UUID,
) ([]AllocationDriftResult, error) {
	var results []AllocationDriftResult

	rows, err := a.db.QueryxContext(ctx, `SELECT * FROM check_allocation_drift() WHERE client_id = $1`, clientID)
	if err != nil {
		return results, nil
	}
	defer rows.Close()

	for rows.Next() {
		var r AllocationDriftResult
		if err := rows.StructScan(&r); err == nil {
			results = append(results, r)
		}
	}

	return results, nil
}

// CreateRebalanceTriggerActivity creates a rebalance trigger
func (a *AdvisorWorkflowActivities) CreateRebalanceTriggerActivity(
	ctx context.Context,
	clientID uuid.UUID,
	drift AllocationDriftResult,
) error {
	_, err := a.db.ExecContext(ctx, `
		INSERT INTO allocation_rebalance_triggers (
			client_id, asset_class, current_allocation_pct, target_allocation_pct,
			tolerance_band_pct, deviation_pct, trigger_type, trigger_severity,
			recommended_action, status
		) VALUES ($1, $2, $3, $4, $5, $6, 'DRIFT_EXCEEDED', 
			CASE WHEN $6 > 5 THEN 'HIGH' ELSE 'MEDIUM' END,
			$7, 'OPEN')
	`, clientID, drift.AssetClass, drift.CurrentAllocationPct, drift.TargetAllocationPct,
		drift.ToleranceBandPct, drift.DeviationPct, drift.RecommendedAction)

	return err
}

// NotifyAdvisorRebalanceActivity notifies advisor of rebalance need
func (a *AdvisorWorkflowActivities) NotifyAdvisorRebalanceActivity(
	ctx context.Context,
	clientID uuid.UUID,
	drift AllocationDriftResult,
) error {
	// Get advisor for this client (simplified - would need proper lookup)
	// For now, just log it
	activity.GetLogger(ctx).Info("Rebalance notification",
		"client_id", clientID,
		"asset_class", drift.AssetClass,
		"deviation", drift.DeviationPct,
	)

	return nil
}

// ============================================================================
// REGULATORY FILING ACTIVITIES
// ============================================================================

// CreateRegulatoryFilingRecordActivity creates a filing record
func (a *AdvisorWorkflowActivities) CreateRegulatoryFilingRecordActivity(
	ctx context.Context,
	filingType string,
	reportingPeriod string,
	dueDate time.Time,
) (uuid.UUID, error) {
	filingID := uuid.New()

	_, err := a.db.ExecContext(ctx, `
		INSERT INTO regulatory_filings (
			filing_id, filing_type, reporting_period, due_date, status
		) VALUES ($1, $2, $3, $4, 'IN_PREPARATION')
	`, filingID, filingType, reportingPeriod, dueDate)

	if err != nil {
		return uuid.Nil, err
	}

	return filingID, nil
}

// CollectRegulatoryFilingDataActivity collects data for filing
func (a *AdvisorWorkflowActivities) CollectRegulatoryFilingDataActivity(
	ctx context.Context,
	filingType string,
	reportingPeriod string,
) (*RegulatoryFilingData, error) {
	data := &RegulatoryFilingData{}

	// Collect qualified client count
	a.db.GetContext(ctx, &data.QualifiedClientsCount, `
		SELECT COUNT(DISTINCT client_id) FROM alternative_investments
	`)

	// Collect total alternative AUM
	a.db.GetContext(ctx, &data.TotalAlternativeAUM, `
		SELECT COALESCE(SUM(current_nav), 0) FROM alternative_investments
	`)

	// Collect illiquid assets
	a.db.GetContext(ctx, &data.IlliquidAssetsValue, `
		SELECT COALESCE(SUM(current_nav), 0) 
		FROM alternative_investments
		WHERE investment_type IN ('PRIVATE_EQUITY', 'VENTURE_CAPITAL', 'REAL_ESTATE')
	`)

	return data, nil
}

// GenerateRegulatoryFilingDocumentActivity generates filing document
func (a *AdvisorWorkflowActivities) GenerateRegulatoryFilingDocumentActivity(
	ctx context.Context,
	filingID uuid.UUID,
	filingType string,
	data *RegulatoryFilingData,
) (string, error) {
	// In production, this would generate actual filing document
	documentURL := fmt.Sprintf("/filings/%s/%s.pdf", filingType, filingID)

	_, err := a.db.ExecContext(ctx, `
		UPDATE regulatory_filings
		SET filing_document_url = $2,
		    filing_data = $3,
		    status = 'REVIEW',
		    updated_at = NOW()
		WHERE filing_id = $1
	`, filingID, documentURL, data)

	if err != nil {
		return "", err
	}

	return documentURL, nil
}

// CreateComplianceReviewTaskActivity creates a compliance review task
func (a *AdvisorWorkflowActivities) CreateComplianceReviewTaskActivity(
	ctx context.Context,
	filingID uuid.UUID,
	filingType string,
	documentURL string,
) error {
	// In production, create task for compliance officer
	activity.GetLogger(ctx).Info("Compliance review task created",
		"filing_id", filingID,
		"type", filingType,
	)

	return nil
}

// SendFilingReminderActivity sends filing reminder
func (a *AdvisorWorkflowActivities) SendFilingReminderActivity(
	ctx context.Context,
	filingID uuid.UUID,
	dueDate time.Time,
) error {
	activity.GetLogger(ctx).Info("Filing reminder sent",
		"filing_id", filingID,
		"due_date", dueDate,
	)

	return nil
}

// UpdateRegulatoryFilingStatusActivity updates filing status
func (a *AdvisorWorkflowActivities) UpdateRegulatoryFilingStatusActivity(
	ctx context.Context,
	filingID uuid.UUID,
	status string,
) error {
	_, err := a.db.ExecContext(ctx, `
		UPDATE regulatory_filings
		SET status = $2,
		    submitted_date = CASE WHEN $2 = 'SUBMITTED' THEN CURRENT_DATE ELSE submitted_date END,
		    updated_at = NOW()
		WHERE filing_id = $1
	`, filingID, status)

	return err
}

// ============================================================================
// STANDALONE ACTIVITY FUNCTIONS FOR TEMPORAL REGISTRATION
// These wrap the struct methods for use with workflow.ExecuteActivity()
// ============================================================================

var defaultActivities *AdvisorWorkflowActivities

// InitAdvisorActivities initializes the global activities instance
func InitAdvisorActivities(db *sqlx.DB) {
	defaultActivities = NewAdvisorWorkflowActivities(db)
}

// RunAutomatedScreeningActivity is the standalone activity function for screening
func RunAutomatedScreeningActivity(ctx context.Context, opportunityID uuid.UUID) (*ScreeningResult, error) {
	return defaultActivities.RunAutomatedScreeningActivity(ctx, opportunityID)
}

// NotifyAdvisorScreeningFailedActivity is the standalone activity function
func NotifyAdvisorScreeningFailedActivity(ctx context.Context, opportunityID, advisorID uuid.UUID, result ScreeningResult) error {
	return defaultActivities.NotifyAdvisorScreeningFailedActivity(ctx, opportunityID, advisorID, result)
}

// UpdateOpportunityStageActivity is the standalone activity function
func UpdateOpportunityStageActivity(ctx context.Context, opportunityID uuid.UUID, newStage, notes string) error {
	return defaultActivities.UpdateOpportunityStageActivity(ctx, opportunityID, newStage, notes)
}

// NotifyAdvisorForReviewActivity is the standalone activity function
func NotifyAdvisorForReviewActivity(ctx context.Context, opportunityID, advisorID uuid.UUID) error {
	return defaultActivities.NotifyAdvisorForReviewActivity(ctx, opportunityID, advisorID)
}

// CreateAdvisorTaskActivity is the standalone activity function
func CreateAdvisorTaskActivity(ctx context.Context, opportunityID, advisorID uuid.UUID, taskType, description string) error {
	return defaultActivities.CreateAdvisorTaskActivity(ctx, opportunityID, advisorID, taskType, description)
}

// InitializeDueDiligenceChecklistActivity is the standalone activity function
func InitializeDueDiligenceChecklistActivity(ctx context.Context, opportunityID uuid.UUID) error {
	return defaultActivities.InitializeDueDiligenceChecklistActivity(ctx, opportunityID)
}

// RiskAssessmentActivity is the standalone activity function
func RiskAssessmentActivity(ctx context.Context, opportunityID uuid.UUID) (*ReviewResult, error) {
	return defaultActivities.RiskAssessmentActivity(ctx, opportunityID)
}

// LegalComplianceReviewActivity is the standalone activity function
func LegalComplianceReviewActivity(ctx context.Context, opportunityID uuid.UUID) (*ReviewResult, error) {
	return defaultActivities.LegalComplianceReviewActivity(ctx, opportunityID)
}

// TaxImpactAnalysisActivity is the standalone activity function
func TaxImpactAnalysisActivity(ctx context.Context, opportunityID uuid.UUID) (*ReviewResult, error) {
	return defaultActivities.TaxImpactAnalysisActivity(ctx, opportunityID)
}

// OperationalDueDiligenceActivity is the standalone activity function
func OperationalDueDiligenceActivity(ctx context.Context, opportunityID uuid.UUID) (*ReviewResult, error) {
	return defaultActivities.OperationalDueDiligenceActivity(ctx, opportunityID)
}

// NotifyCommitteeRejectionActivity is the standalone activity function
func NotifyCommitteeRejectionActivity(ctx context.Context, opportunityID uuid.UUID, failures []string) error {
	return defaultActivities.NotifyCommitteeRejectionActivity(ctx, opportunityID, failures)
}

// GenerateCommitteePackageActivity is the standalone activity function
func GenerateCommitteePackageActivity(ctx context.Context, opportunityID uuid.UUID, reviewResults *ParallelReviewResults) error {
	return defaultActivities.GenerateCommitteePackageActivity(ctx, opportunityID, reviewResults)
}

// GenerateCommitmentDocumentsActivity is the standalone activity function
func GenerateCommitmentDocumentsActivity(ctx context.Context, opportunityID uuid.UUID, approvedAmount float64) error {
	return defaultActivities.GenerateCommitmentDocumentsActivity(ctx, opportunityID, approvedAmount)
}

// InitiateESignatureWorkflowActivity is the standalone activity function
func InitiateESignatureWorkflowActivity(ctx context.Context, opportunityID, clientID uuid.UUID) (uuid.UUID, error) {
	return defaultActivities.InitiateESignatureWorkflowActivity(ctx, opportunityID, clientID)
}

// ProcessCapitalCommitmentActivity is the standalone activity function
func ProcessCapitalCommitmentActivity(ctx context.Context, opportunityID, clientID uuid.UUID, amount float64) error {
	return defaultActivities.ProcessCapitalCommitmentActivity(ctx, opportunityID, clientID, amount)
}

// OnboardToPortfolioActivity is the standalone activity function
func OnboardToPortfolioActivity(ctx context.Context, opportunityID, clientID uuid.UUID) error {
	return defaultActivities.OnboardToPortfolioActivity(ctx, opportunityID, clientID)
}

// CreateQuarterlyReviewRecordActivity is the standalone activity function
func CreateQuarterlyReviewRecordActivity(ctx context.Context, clientID uuid.UUID, reviewPeriod string) (uuid.UUID, error) {
	return defaultActivities.CreateQuarterlyReviewRecordActivity(ctx, clientID, reviewPeriod)
}

// CalculatePortfolioPerformanceActivity is the standalone activity function
func CalculatePortfolioPerformanceActivity(ctx context.Context, clientID uuid.UUID, reviewPeriod string) (*PerformanceData, error) {
	return defaultActivities.CalculatePortfolioPerformanceActivity(ctx, clientID, reviewPeriod)
}

// LiquidityStressTestActivity is the standalone activity function
func LiquidityStressTestActivity(ctx context.Context, clientID uuid.UUID) (*LiquidityAnalysis, error) {
	return defaultActivities.LiquidityStressTestActivity(ctx, clientID)
}

// FetchManagerUpdatesActivity is the standalone activity function
func FetchManagerUpdatesActivity(ctx context.Context, clientID uuid.UUID, reviewPeriod string) ([]ManagerUpdateSummary, error) {
	return defaultActivities.FetchManagerUpdatesActivity(ctx, clientID, reviewPeriod)
}

// AssessPortfolioRisksActivity is the standalone activity function
func AssessPortfolioRisksActivity(ctx context.Context, clientID uuid.UUID, performance PerformanceData, liquidity LiquidityAnalysis, managerUpdates []ManagerUpdateSummary) ([]RiskFlag, error) {
	return defaultActivities.AssessPortfolioRisksActivity(ctx, clientID, performance, liquidity, managerUpdates)
}

// EscalateRiskPositionsActivity is the standalone activity function
func EscalateRiskPositionsActivity(ctx context.Context, clientID, advisorID uuid.UUID, riskFlags []RiskFlag) error {
	return defaultActivities.EscalateRiskPositionsActivity(ctx, clientID, advisorID, riskFlags)
}

// GenerateQuarterlyReportActivity is the standalone activity function
func GenerateQuarterlyReportActivity(ctx context.Context, reviewID, clientID uuid.UUID, performance PerformanceData, liquidity LiquidityAnalysis, riskFlags []RiskFlag, managerUpdates []ManagerUpdateSummary) (string, error) {
	return defaultActivities.GenerateQuarterlyReportActivity(ctx, reviewID, clientID, performance, liquidity, riskFlags, managerUpdates)
}

// ScheduleClientReviewMeetingActivity is the standalone activity function
func ScheduleClientReviewMeetingActivity(ctx context.Context, clientID, advisorID, reviewID uuid.UUID, reportURL string) (time.Time, error) {
	return defaultActivities.ScheduleClientReviewMeetingActivity(ctx, clientID, advisorID, reviewID, reportURL)
}

// UpdateInvestmentCommitteeActivity is the standalone activity function
func UpdateInvestmentCommitteeActivity(ctx context.Context, clientID uuid.UUID, performance PerformanceData, riskFlags []RiskFlag) error {
	return defaultActivities.UpdateInvestmentCommitteeActivity(ctx, clientID, performance, riskFlags)
}

// UpdateQuarterlyReviewStatusActivity is the standalone activity function
func UpdateQuarterlyReviewStatusActivity(ctx context.Context, reviewID uuid.UUID, status, reportURL string) error {
	return defaultActivities.UpdateQuarterlyReviewStatusActivity(ctx, reviewID, status, reportURL)
}

// IdentifyFundingSourceActivity is the standalone activity function
func IdentifyFundingSourceActivity(ctx context.Context, clientID uuid.UUID, amountRequired float64) (*FundingAccountResult, error) {
	return defaultActivities.IdentifyFundingSourceActivity(ctx, clientID, amountRequired)
}

// SendCapitalCallLiquidityAlertActivity is the standalone activity function
func SendCapitalCallLiquidityAlertActivity(ctx context.Context, clientID, advisorID uuid.UUID, eventID uuid.UUID, amount, shortfall float64) error {
	// Simplified - just log it
	return nil
}

// InitiateFundingTransferActivity is the standalone activity function
func InitiateFundingTransferActivity(ctx context.Context, eventID, accountID uuid.UUID, amount float64) (*TransferResult, error) {
	return defaultActivities.InitiateFundingTransferActivity(ctx, eventID, accountID, amount)
}

// UpdateCapitalEventStatusActivity is the standalone activity function
func UpdateCapitalEventStatusActivity(ctx context.Context, eventID uuid.UUID, status string, amountFunded float64) error {
	return defaultActivities.UpdateCapitalEventStatusActivity(ctx, eventID, status, amountFunded)
}

// SendCapitalCallConfirmationActivity is the standalone activity function
func SendCapitalCallConfirmationActivity(ctx context.Context, eventID, clientID uuid.UUID, amount float64) error {
	return defaultActivities.SendCapitalCallConfirmationActivity(ctx, eventID, clientID, amount)
}

// GetClientsWithAllocationTargetsActivity is the standalone activity function
func GetClientsWithAllocationTargetsActivity(ctx context.Context) ([]uuid.UUID, error) {
	return defaultActivities.GetClientsWithAllocationTargetsActivity(ctx)
}

// CheckClientAllocationDriftActivity is the standalone activity function
func CheckClientAllocationDriftActivity(ctx context.Context, clientID uuid.UUID) ([]AllocationDriftResult, error) {
	return defaultActivities.CheckClientAllocationDriftActivity(ctx, clientID)
}

// CreateRebalanceTriggerActivity is the standalone activity function
func CreateRebalanceTriggerActivity(ctx context.Context, clientID uuid.UUID, drift AllocationDriftResult) error {
	return defaultActivities.CreateRebalanceTriggerActivity(ctx, clientID, drift)
}

// NotifyAdvisorRebalanceActivity is the standalone activity function
func NotifyAdvisorRebalanceActivity(ctx context.Context, clientID uuid.UUID, drift AllocationDriftResult) error {
	return defaultActivities.NotifyAdvisorRebalanceActivity(ctx, clientID, drift)
}

// CreateRegulatoryFilingRecordActivity is the standalone activity function
func CreateRegulatoryFilingRecordActivity(ctx context.Context, filingType, reportingPeriod string, dueDate time.Time) (uuid.UUID, error) {
	return defaultActivities.CreateRegulatoryFilingRecordActivity(ctx, filingType, reportingPeriod, dueDate)
}

// CollectRegulatoryFilingDataActivity is the standalone activity function
func CollectRegulatoryFilingDataActivity(ctx context.Context, filingType, reportingPeriod string) (*RegulatoryFilingData, error) {
	return defaultActivities.CollectRegulatoryFilingDataActivity(ctx, filingType, reportingPeriod)
}

// GenerateRegulatoryFilingDocumentActivity is the standalone activity function
func GenerateRegulatoryFilingDocumentActivity(ctx context.Context, filingID uuid.UUID, filingType string, data *RegulatoryFilingData) (string, error) {
	return defaultActivities.GenerateRegulatoryFilingDocumentActivity(ctx, filingID, filingType, data)
}

// CreateComplianceReviewTaskActivity is the standalone activity function
func CreateComplianceReviewTaskActivity(ctx context.Context, filingID uuid.UUID, filingType, documentURL string) error {
	return defaultActivities.CreateComplianceReviewTaskActivity(ctx, filingID, filingType, documentURL)
}

// SendFilingReminderActivity is the standalone activity function
func SendFilingReminderActivity(ctx context.Context, filingID uuid.UUID, dueDate time.Time) error {
	return defaultActivities.SendFilingReminderActivity(ctx, filingID, dueDate)
}

// UpdateRegulatoryFilingStatusActivity is the standalone activity function (wrapper)
func UpdateRegulatoryFilingStatusActivity(ctx context.Context, filingID uuid.UUID, status string) error {
	return defaultActivities.UpdateRegulatoryFilingStatusActivity(ctx, filingID, status)
}
