package altinv

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// ========================================
// CAPITAL CALL MONITORING WORKFLOW
// ========================================

// CapitalCallMonitoringWorkflow monitors capital calls and sends alerts
func CapitalCallMonitoringWorkflow(ctx workflow.Context) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting capital call monitoring workflow")

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

	// Run continuously - check every 6 hours
	for {
		logger.Info("Checking for upcoming capital calls")

		// Check for calls due in next 7 days
		var upcomingCalls []UpcomingCapitalCall
		err := workflow.ExecuteActivity(ctx, CapitalCallCheckUpcomingActivity, 7).Get(ctx, &upcomingCalls)
		if err != nil {
			logger.Error("Failed to check upcoming capital calls", "error", err)
		} else if len(upcomingCalls) > 0 {
			logger.Info("Found upcoming capital calls", "count", len(upcomingCalls))

			// Process each capital call
			for _, call := range upcomingCalls {
				// Check liquidity for this call
				var liquidityOk bool
				err := workflow.ExecuteActivity(ctx, CapitalCallValidateLiquidityActivity, call.CallID).Get(ctx, &liquidityOk)
				if err != nil {
					logger.Error("Failed to validate liquidity", "call_id", call.CallID, "error", err)
					continue
				}

				// Send notifications based on urgency and liquidity
				if call.DaysUntilDue <= 3 {
					// Urgent - due within 3 days
					err = workflow.ExecuteActivity(ctx, CapitalCallSendUrgentAlertActivity, call).Get(ctx, nil)
					if err != nil {
						logger.Error("Failed to send urgent alert", "call_id", call.CallID, "error", err)
					}
				} else if call.DaysUntilDue <= 7 && !liquidityOk {
					// Liquidity issue detected
					err = workflow.ExecuteActivity(ctx, CapitalCallSendLiquidityAlertActivity, call).Get(ctx, nil)
					if err != nil {
						logger.Error("Failed to send liquidity alert", "call_id", call.CallID, "error", err)
					}
				}
			}
		}

		// Check for overdue capital calls
		var overdueCalls []UpcomingCapitalCall
		err = workflow.ExecuteActivity(ctx, CapitalCallCheckOverdueActivity).Get(ctx, &overdueCalls)
		if err != nil {
			logger.Error("Failed to check overdue capital calls", "error", err)
		} else if len(overdueCalls) > 0 {
			logger.Warn("Found overdue capital calls", "count", len(overdueCalls))

			for _, call := range overdueCalls {
				// Update status to overdue
				err = workflow.ExecuteActivity(ctx, CapitalCallUpdateOverdueActivity, call.CallID).Get(ctx, nil)
				if err != nil {
					logger.Error("Failed to update overdue status", "call_id", call.CallID, "error", err)
				}

				// Send escalation alert
				err = workflow.ExecuteActivity(ctx, CapitalCallSendOverdueAlertActivity, call).Get(ctx, nil)
				if err != nil {
					logger.Error("Failed to send overdue alert", "call_id", call.CallID, "error", err)
				}
			}
		}

		// Sleep for 6 hours before next check
		workflow.Sleep(ctx, 6*time.Hour)
	}
}

// ========================================
// QUARTERLY STATEMENT PROCESSING WORKFLOW
// ========================================

// QuarterlyStatementProcessingInput contains input for processing a GP statement
type QuarterlyStatementProcessingInput struct {
	DocumentID   uuid.UUID `json:"document_id"`
	InvestmentID uuid.UUID `json:"investment_id"`
	FileURL      string    `json:"file_url"`
}

// QuarterlyStatementProcessingOutput contains results of processing
type QuarterlyStatementProcessingOutput struct {
	Success            bool                    `json:"success"`
	ExtractedData      *ExtractedQuarterlyData `json:"extracted_data,omitempty"`
	Confidence         float64                 `json:"confidence"`
	RequiresReview     bool                    `json:"requires_review"`
	InvestmentUpdated  bool                    `json:"investment_updated"`
	CapitalCallCreated bool                    `json:"capital_call_created"`
	Error              string                  `json:"error,omitempty"`
}

// QuarterlyStatementProcessingWorkflow orchestrates AI-powered document processing
func QuarterlyStatementProcessingWorkflow(ctx workflow.Context, input QuarterlyStatementProcessingInput) (*QuarterlyStatementProcessingOutput, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting quarterly statement processing",
		"document_id", input.DocumentID,
		"investment_id", input.InvestmentID,
	)

	output := &QuarterlyStatementProcessingOutput{}

	// Configure activity options with longer timeout for AI processing
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Minute,
		HeartbeatTimeout:    2 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    5 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    2, // AI is expensive, limit retries
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Step 1: Extract text from PDF
	logger.Info("Extracting text from PDF")
	var pdfText string
	err := workflow.ExecuteActivity(ctx, CapitalCallExtractPDFActivity, input.FileURL).Get(ctx, &pdfText)
	if err != nil {
		output.Success = false
		output.Error = fmt.Sprintf("Failed to extract PDF text: %v", err)
		return output, nil
	}

	// Step 2: Use Gemini to extract structured data
	logger.Info("Extracting structured data using Gemini AI")
	var extractedData ExtractedQuarterlyData
	var confidence float64

	extractResult := struct {
		Data       ExtractedQuarterlyData `json:"data"`
		Confidence float64                `json:"confidence"`
	}{}

	err = workflow.ExecuteActivity(ctx, CapitalCallProcessStatementActivity,
		input.DocumentID, pdfText).Get(ctx, &extractResult)
	if err != nil {
		output.Success = false
		output.Error = fmt.Sprintf("Failed to process with AI: %v", err)
		return output, nil
	}

	extractedData = extractResult.Data
	confidence = extractResult.Confidence
	output.ExtractedData = &extractedData
	output.Confidence = confidence

	// Step 3: Determine if manual review is required
	output.RequiresReview = confidence < 0.7

	if output.RequiresReview {
		logger.Warn("Low confidence extraction, flagging for manual review", "confidence", confidence)
		// Signal human-in-the-loop for review
		err = workflow.ExecuteActivity(ctx, CapitalCallNotifyForStatementReviewActivity, input.DocumentID, confidence).Get(ctx, nil)
		if err != nil {
			logger.Error("Failed to send review notification", "error", err)
		}

		output.Success = true
		return output, nil
	}

	// Step 4: Update investment record with extracted data
	logger.Info("Updating investment with extracted data")
	err = workflow.ExecuteActivity(ctx, CapitalCallUpdateInvestmentActivity,
		input.InvestmentID, extractedData).Get(ctx, nil)
	if err != nil {
		logger.Error("Failed to update investment", "error", err)
		output.Error = fmt.Sprintf("Failed to update investment: %v", err)
	} else {
		output.InvestmentUpdated = true
	}

	// Step 5: If capital was called, create capital call record
	if extractedData.CapitalCalled != nil && *extractedData.CapitalCalled > 0 {
		logger.Info("Capital called detected, creating capital call record",
			"amount", *extractedData.CapitalCalled)

		err = workflow.ExecuteActivity(ctx, CapitalCallCreateFromStatementActivity,
			input.InvestmentID, *extractedData.CapitalCalled).Get(ctx, nil)
		if err != nil {
			logger.Error("Failed to create capital call", "error", err)
		} else {
			output.CapitalCallCreated = true
		}
	}

	output.Success = true
	logger.Info("Quarterly statement processing complete",
		"success", output.Success,
		"investment_updated", output.InvestmentUpdated,
		"capital_call_created", output.CapitalCallCreated,
	)

	return output, nil
}

// ========================================
// WORKFLOW ACTIVITIES
// ========================================

type CapitalCallActivities struct {
	DB            *sqlx.DB
	AltInvService Service
	EmailService  EmailService // For sending notifications
}

// EmailService interface for sending notifications
type EmailService interface {
	SendAlert(ctx context.Context, to []string, subject, body string) error
}

func NewCapitalCallActivities(db *sqlx.DB, altInvService Service, emailService EmailService) *CapitalCallActivities {
	return &CapitalCallActivities{
		DB:            db,
		AltInvService: altInvService,
		EmailService:  emailService,
	}
}

// CapitalCallCheckUpcomingActivity finds capital calls due within N days
func (a *CapitalCallActivities) CapitalCallCheckUpcomingActivity(ctx context.Context, daysAhead int) ([]UpcomingCapitalCall, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Checking for upcoming capital calls", "days_ahead", daysAhead)

	var calls []UpcomingCapitalCall
	query := `
		SELECT * FROM upcoming_capital_calls
		WHERE days_until_due BETWEEN 0 AND $1
		  AND (status = 'PENDING' OR status = 'PARTIALLY_FUNDED')
		ORDER BY days_until_due
	`

	err := a.DB.SelectContext(ctx, &calls, query, daysAhead)
	if err != nil {
		return nil, fmt.Errorf("failed to query upcoming capital calls: %w", err)
	}

	logger.Info("Found upcoming capital calls", "count", len(calls))
	return calls, nil
}

// CapitalCallCheckOverdueActivity finds overdue capital calls
func (a *CapitalCallActivities) CapitalCallCheckOverdueActivity(ctx context.Context) ([]UpcomingCapitalCall, error) {
	logger := activity.GetLogger(ctx)

	var calls []UpcomingCapitalCall
	query := `
		SELECT * FROM upcoming_capital_calls
		WHERE days_until_due < 0
		  AND status != 'OVERDUE'
		  AND status != 'FUNDED'
		ORDER BY days_until_due
	`

	err := a.DB.SelectContext(ctx, &calls, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query overdue capital calls: %w", err)
	}

	logger.Info("Found overdue capital calls", "count", len(calls))
	return calls, nil
}

// CapitalCallValidateLiquidityActivity checks if client has sufficient liquidity
func (a *CapitalCallActivities) CapitalCallValidateLiquidityActivity(ctx context.Context, callID uuid.UUID) (bool, error) {
	logger := activity.GetLogger(ctx)

	// Get capital call details
	call, err := a.AltInvService.GetCapitalCall(ctx, callID)
	if err != nil {
		return false, err
	}

	// Check client's liquid assets (cash, money market, etc.)
	// This would integrate with the portfolio service to check available liquidity
	query := `
		SELECT COALESCE(SUM(market_value), 0) as liquid_assets
		FROM portfolio_holdings
		WHERE client_id = (
			SELECT client_id FROM alternative_investments 
			WHERE investment_id = $1
		)
		AND asset_class IN ('CASH', 'MONEY_MARKET', 'SHORT_TERM_BONDS')
	`

	var liquidAssets float64
	err = a.DB.GetContext(ctx, &liquidAssets, query, call.InvestmentID)
	if err != nil {
		logger.Error("Failed to check liquidity", "error", err)
		return false, err
	}

	amountNeeded := call.AmountRequested - call.AmountFunded
	hasLiquidity := liquidAssets >= amountNeeded

	// Update capital call with liquidity check results
	checkPassed := hasLiquidity
	shortage := float64(0)
	if !hasLiquidity {
		shortage = amountNeeded - liquidAssets
	}

	updateQuery := `
		UPDATE capital_calls
		SET liquidity_check_passed = $1,
		    liquidity_check_date = NOW(),
		    liquidity_shortage_amount = $2
		WHERE call_id = $3
	`
	_, err = a.DB.ExecContext(ctx, updateQuery, checkPassed, shortage, callID)
	if err != nil {
		logger.Warn("Failed to update liquidity check", "error", err)
	}

	logger.Info("Liquidity validation complete",
		"call_id", callID,
		"amount_needed", amountNeeded,
		"liquid_assets", liquidAssets,
		"has_liquidity", hasLiquidity,
	)

	return hasLiquidity, nil
}

// CapitalCallSendUrgentAlertActivity sends urgent alert for capital calls due soon
func (a *CapitalCallActivities) CapitalCallSendUrgentAlertActivity(ctx context.Context, call UpcomingCapitalCall) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Sending urgent capital call alert", "call_id", call.CallID, "days_until_due", call.DaysUntilDue)

	subject := fmt.Sprintf("URGENT: Capital Call Due in %d Days - %s", call.DaysUntilDue, call.FundName)
	body := fmt.Sprintf(`
URGENT: Capital Call Action Required

Fund: %s
Amount Requested: $%.2f
Amount Funded: $%.2f
Balance Due: $%.2f
Due Date: %s
Days Remaining: %d

Please ensure funding is arranged immediately.
`,
		call.FundName,
		call.AmountRequested,
		call.AmountFunded,
		call.AmountRequested-call.AmountFunded,
		call.DueDate.Format("2006-01-02"),
		call.DaysUntilDue,
	)

	// Send to advisor and client
	// In production, fetch email addresses from client record
	err := a.EmailService.SendAlert(ctx, []string{"advisor@example.com"}, subject, body)
	if err != nil {
		return fmt.Errorf("failed to send urgent alert: %w", err)
	}

	// Update capital call with alert sent timestamp
	_, err = a.DB.ExecContext(ctx, `UPDATE capital_calls SET alert_sent_at = NOW() WHERE call_id = $1`, call.CallID)
	return err
}

// CapitalCallSendLiquidityAlertActivity sends alert about liquidity shortage
func (a *CapitalCallActivities) CapitalCallSendLiquidityAlertActivity(ctx context.Context, call UpcomingCapitalCall) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Sending liquidity shortage alert", "call_id", call.CallID)

	subject := fmt.Sprintf("Liquidity Alert: Capital Call - %s", call.FundName)
	body := fmt.Sprintf(`
Liquidity Shortage Detected

The upcoming capital call for %s may have insufficient funding.

Please review client liquidity and arrange for additional cash if needed.
`,
		call.FundName,
	)

	return a.EmailService.SendAlert(ctx, []string{"advisor@example.com"}, subject, body)
}

// CapitalCallSendOverdueAlertActivity sends escalation for overdue capital calls
func (a *CapitalCallActivities) CapitalCallSendOverdueAlertActivity(ctx context.Context, call UpcomingCapitalCall) error {
	logger := activity.GetLogger(ctx)
	logger.Error("Capital call is overdue", "call_id", call.CallID, "days_overdue", -call.DaysUntilDue)

	subject := fmt.Sprintf("OVERDUE: Capital Call Payment Missed - %s", call.FundName)
	body := fmt.Sprintf(`
CRITICAL: Capital Call Payment Overdue

This capital call payment was due %d days ago and requires immediate action.

Fund: %s
Amount Due: $%.2f
Original Due Date: %s

Please contact the GP immediately and arrange payment.
`,
		-call.DaysUntilDue,
		call.FundName,
		call.AmountRequested-call.AmountFunded,
		call.DueDate.Format("2006-01-02"),
	)

	return a.EmailService.SendAlert(ctx, []string{"advisor@example.com", "manager@example.com"}, subject, body)
}

// CapitalCallUpdateOverdueActivity marks a capital call as overdue
func (a *CapitalCallActivities) CapitalCallUpdateOverdueActivity(ctx context.Context, callID uuid.UUID) error {
	return a.AltInvService.UpdateCapitalCallStatus(ctx, callID, StatusOverdue, 0)
}

// CapitalCallExtractPDFActivity extracts text from a PDF file
func (a *CapitalCallActivities) CapitalCallExtractPDFActivity(ctx context.Context, fileURL string) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Extracting text from PDF", "url", fileURL)

	// In production, this would:
	// 1. Download file from S3/storage
	// 2. Use PDF library (pdftotext, Apache Tika, etc.) to extract text
	// 3. OCR if scanned PDF

	// For now, return simulated text
	return `
SEQUOIA CAPITAL FUND XV
QUARTERLY STATEMENT - Q4 2025

Net Asset Value as of December 31, 2025: $12,500,000
Capital Called This Quarter: $500,000
Distributions Paid This Quarter: $0
IRR Since Inception: 18.5%
TVPI: 1.45x
Unfunded Commitment Remaining: $2,000,000
`, nil
}

// CapitalCallProcessStatementActivity uses Gemini to extract structured data
func (a *CapitalCallActivities) CapitalCallProcessStatementActivity(ctx context.Context, documentID uuid.UUID, pdfText string) (map[string]interface{}, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Processing quarterly statement with AI", "document_id", documentID)

	// This would call the document intelligence service
	// For now, simulate the extraction
	nav := 12500000.0
	capitalCalled := 500000.0
	irr := 18.5
	tvpi := 1.45
	unfunded := 2000000.0
	navDate := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)

	data := ExtractedQuarterlyData{
		NAV:                &nav,
		NAVDate:            &navDate,
		CapitalCalled:      &capitalCalled,
		IRR:                &irr,
		TVPI:               &tvpi,
		UnfundedCommitment: &unfunded,
	}

	confidence := 0.92 // High confidence

	return map[string]interface{}{
		"data":       data,
		"confidence": confidence,
	}, nil
}

// CapitalCallUpdateInvestmentActivity updates investment with extracted data
func (a *CapitalCallActivities) CapitalCallUpdateInvestmentActivity(ctx context.Context, investmentID uuid.UUID, data ExtractedQuarterlyData) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Updating investment from quarterly data", "investment_id", investmentID)

	input := UpdateInvestmentInput{
		CurrentNAV:         data.NAV,
		NAVDate:            data.NAVDate,
		IRRSinceInception:  data.IRR,
		TVPI:               data.TVPI,
		UnfundedCommitment: data.UnfundedCommitment,
	}

	valSource := GPReported
	input.ValuationSource = &valSource

	_, err := a.AltInvService.UpdateInvestment(ctx, investmentID, input)
	return err
}

// CapitalCallCreateFromStatementActivity creates a capital call from extracted data
func (a *CapitalCallActivities) CapitalCallCreateFromStatementActivity(ctx context.Context, investmentID uuid.UUID, amount float64) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Creating capital call from quarterly data", "investment_id", investmentID, "amount", amount)

	// Estimate due date (typically 30 days from notice)
	noticeDate := time.Now()
	dueDate := noticeDate.Add(30 * 24 * time.Hour)

	input := CreateCapitalCallInput{
		InvestmentID:    investmentID,
		NoticeDate:      noticeDate,
		DueDate:         dueDate,
		AmountRequested: amount,
	}

	_, err := a.AltInvService.CreateCapitalCall(ctx, input)
	return err
}

// CapitalCallNotifyForStatementReviewActivity notifies advisor to review low-confidence extraction
func (a *CapitalCallActivities) CapitalCallNotifyForStatementReviewActivity(ctx context.Context, documentID uuid.UUID, confidence float64) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Notifying advisor for manual review", "document_id", documentID, "confidence", confidence)

	subject := "Manual Review Required: Quarterly Statement Extraction"
	body := fmt.Sprintf(`
The AI extraction confidence for document %s was only %.1f%%.

Please manually review and update the investment data.
`,
		documentID, confidence*100,
	)

	return a.EmailService.SendAlert(ctx, []string{"advisor@example.com"}, subject, body)
}

// ============================================================================
// STANDALONE ACTIVITY FUNCTIONS FOR TEMPORAL REGISTRATION
// ============================================================================

var capitalCallActivities *CapitalCallActivities

// InitCapitalCallActivities sets the backing implementation for standalone activities.
func InitCapitalCallActivities(db *sqlx.DB, altInvService Service, emailService EmailService) {
	capitalCallActivities = NewCapitalCallActivities(db, altInvService, emailService)
}

func CapitalCallCheckUpcomingActivity(ctx context.Context, daysAhead int) ([]UpcomingCapitalCall, error) {
	return capitalCallActivities.CapitalCallCheckUpcomingActivity(ctx, daysAhead)
}

func CapitalCallCheckOverdueActivity(ctx context.Context) ([]UpcomingCapitalCall, error) {
	return capitalCallActivities.CapitalCallCheckOverdueActivity(ctx)
}

func CapitalCallValidateLiquidityActivity(ctx context.Context, callID uuid.UUID) (bool, error) {
	return capitalCallActivities.CapitalCallValidateLiquidityActivity(ctx, callID)
}

func CapitalCallSendUrgentAlertActivity(ctx context.Context, call UpcomingCapitalCall) error {
	return capitalCallActivities.CapitalCallSendUrgentAlertActivity(ctx, call)
}

func CapitalCallSendLiquidityAlertActivity(ctx context.Context, call UpcomingCapitalCall) error {
	return capitalCallActivities.CapitalCallSendLiquidityAlertActivity(ctx, call)
}

func CapitalCallSendOverdueAlertActivity(ctx context.Context, call UpcomingCapitalCall) error {
	return capitalCallActivities.CapitalCallSendOverdueAlertActivity(ctx, call)
}

func CapitalCallUpdateOverdueActivity(ctx context.Context, callID uuid.UUID) error {
	return capitalCallActivities.CapitalCallUpdateOverdueActivity(ctx, callID)
}

func CapitalCallExtractPDFActivity(ctx context.Context, fileURL string) (string, error) {
	return capitalCallActivities.CapitalCallExtractPDFActivity(ctx, fileURL)
}

func CapitalCallProcessStatementActivity(ctx context.Context, documentID uuid.UUID, pdfText string) (map[string]interface{}, error) {
	return capitalCallActivities.CapitalCallProcessStatementActivity(ctx, documentID, pdfText)
}

func CapitalCallUpdateInvestmentActivity(ctx context.Context, investmentID uuid.UUID, data ExtractedQuarterlyData) error {
	return capitalCallActivities.CapitalCallUpdateInvestmentActivity(ctx, investmentID, data)
}

func CapitalCallCreateFromStatementActivity(ctx context.Context, investmentID uuid.UUID, amount float64) error {
	return capitalCallActivities.CapitalCallCreateFromStatementActivity(ctx, investmentID, amount)
}

func CapitalCallNotifyForStatementReviewActivity(ctx context.Context, documentID uuid.UUID, confidence float64) error {
	return capitalCallActivities.CapitalCallNotifyForStatementReviewActivity(ctx, documentID, confidence)
}
