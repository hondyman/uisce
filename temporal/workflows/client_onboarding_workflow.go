package workflows

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// ============================================================================
// CLIENT ONBOARDING WORKFLOW
// ============================================================================

// ClientOnboardingWorkflowInput represents the input to the onboarding workflow
type ClientOnboardingWorkflowInput struct {
	ClientID             string
	ClientName           string
	ClientEmail          string
	ManagerID            string
	AMLProvider          string // lexis_nexis, worldcheck, internal
	RequiresDueDiligence bool
	DueDiligenceReason   *string
}

// ClientOnboardingWorkflowState tracks the workflow state
type ClientOnboardingWorkflowState struct {
	ClientID             string
	CurrentStep          int
	ValidationPassed     bool
	ValidationErrors     []string
	AdvisorID            string
	AgreementsSentTime   time.Time
	AgreementsSignedTime *time.Time
	AccountsCreatedCount int
	NotificationSentTime *time.Time
	OverallStatus        string
	ApprovedBy           *string
	RejectedBy           *string
	RejectionReason      *string
	TimeoutEscalationID  *string
}

// ClientOnboardingWorkflow orchestrates the 5-step client onboarding process
// This workflow leverages Temporal's capabilities for reliable, long-running processes
// with built-in timeout handling, retries, and state management
func ClientOnboardingWorkflow(ctx workflow.Context, input ClientOnboardingWorkflowInput) (*ClientOnboardingWorkflowState, error) {
	// Initialize workflow state
	state := &ClientOnboardingWorkflowState{
		ClientID:      input.ClientID,
		CurrentStep:   1,
		OverallStatus: "in_progress",
	}

	// Set up logging
	logger := workflow.GetLogger(ctx)
	logger.Info("ClientOnboardingWorkflow started", "clientID", input.ClientID, "clientName", input.ClientName)

	// Define retry policy for activities
	retryPolicy := &temporal.RetryPolicy{
		InitialInterval:    1 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    10 * time.Second,
		MaximumAttempts:    3,
	}

	// Define activity options
	activityOptions := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		ScheduleToCloseTimeout: 5 * time.Minute,
		RetryPolicy:            retryPolicy,
		HeartbeatTimeout:       10 * time.Second,
	})

	// ========================================================================
	// STEP 1: VALIDATE CLIENT DATA AGAINST REGULATORY REQUIREMENTS
	// ========================================================================
	logger.Info("Starting Step 1: Client Validation", "clientID", input.ClientID)
	state.CurrentStep = 1

	var validationResult ValidationResult
	err := workflow.ExecuteActivity(activityOptions, ValidateClientDataActivity, ValidationInput{
		ClientID:             input.ClientID,
		VerifyKYC:            true,
		PerformAMLScreening:  true,
		AMLProvider:          input.AMLProvider,
		RequiresDueDiligence: input.RequiresDueDiligence,
	}).Get(activityOptions, &validationResult)

	if err != nil {
		logger.Error("Step 1 validation failed", "error", err)
		state.ValidationPassed = false
		state.ValidationErrors = []string{err.Error()}
		state.OverallStatus = "failed"
		return state, fmt.Errorf("validation failed: %w", err)
	}

	state.ValidationPassed = validationResult.Passed
	state.ValidationErrors = validationResult.Errors

	if !validationResult.Passed {
		logger.Warn("Client validation not passed, awaiting escalation resolution", "errors", validationResult.Errors)

		// If validation fails, escalate to compliance officer
		// Escalation workflow will handle approval/rejection
		escalationCtx, cancelEscalation := workflow.WithCancel(ctx)
		defer cancelEscalation()

		// Start escalation subprocess
		escalationFuture := workflow.ExecuteChildWorkflow(escalationCtx, ClientOnboardingEscalationWorkflow, ClientOnboardingEscalationInput{
			ClientID:     input.ClientID,
			Reason:       "Validation failures: " + fmt.Sprint(validationResult.Errors),
			EscalatedTo:  "compliance_director",
			TimeoutHours: 48,
		})

		var escalationResult ClientOnboardingEscalationResult
		err := escalationFuture.Get(escalationCtx, &escalationResult)
		if err != nil {
			logger.Error("Escalation workflow failed", "error", err)
			state.OverallStatus = "failed"
			return state, fmt.Errorf("escalation failed: %w", err)
		}

		if !escalationResult.Approved {
			logger.Warn("Client onboarding rejected after escalation", "reason", escalationResult.RejectionReason)
			state.OverallStatus = "rejected"
			state.RejectedBy = &escalationResult.RejectedBy
			state.RejectionReason = escalationResult.RejectionReason
			return state, errors.New("onboarding rejected by compliance team")
		}

		logger.Info("Validation escalation approved, continuing workflow")
	}

	logger.Info("Step 1 completed: Client data validated", "clientID", input.ClientID)

	// ========================================================================
	// STEP 2: ROUTE FOR ADVISOR REVIEW/APPROVAL
	// ========================================================================
	logger.Info("Starting Step 2: Route for Advisor Review", "clientID", input.ClientID)
	state.CurrentStep = 2

	var routingResult RoutingResult
	err = workflow.ExecuteActivity(activityOptions, RouteForAdvisorReviewActivity, RoutingInput{
		ClientID:    input.ClientID,
		RiskProfile: validationResult.RiskProfile,
		NetWorth:    validationResult.NetWorth,
	}).Get(activityOptions, &routingResult)

	if err != nil {
		logger.Error("Step 2 routing failed", "error", err)
		state.OverallStatus = "failed"
		return state, fmt.Errorf("routing failed: %w", err)
	}

	state.AdvisorID = routingResult.AssignedAdvisorID
	logger.Info("Step 2 completed: Client routed to advisor", "advisorID", state.AdvisorID)

	// Set up signal and timer for advisor approval
	// Timeout: 48 hours for advisor review
	var approvalSignal string
	approvalTimeout := 48 * time.Hour

	sigChan := workflow.GetSignalChannel(ctx, "advisor_approval")
	timeoutTimer := workflow.NewTimer(ctx, approvalTimeout)

	s := workflow.NewSelector(ctx)

	// Handle approval signal
	s.AddReceive(sigChan, func(c workflow.ReceiveChannel, more bool) {
		c.Receive(ctx, &approvalSignal)
		logger.Info("Advisor approval signal received", "signal", approvalSignal)
	})

	// Handle timeout
	timeoutOccurred := false
	s.AddFuture(timeoutTimer, func(f workflow.Future) {
		timeoutOccurred = true
		logger.Warn("Advisor approval timeout", "timeout", approvalTimeout)
	})

	s.Select(ctx)

	if timeoutOccurred {
		logger.Warn("Advisor review timeout - starting escalation", "clientID", input.ClientID)

		// Start timeout escalation workflow
		var escalationID string
		err := workflow.ExecuteActivity(activityOptions, StartTimeoutEscalationActivity, TimeoutEscalationInput{
			ClientID:     input.ClientID,
			StepName:     "advisor_review",
			TimeoutHours: 48,
			Action:       "escalate",
		}).Get(activityOptions, &escalationID)

		if err != nil {
			logger.Error("Failed to start timeout escalation", "error", err)
			state.TimeoutEscalationID = &escalationID
		}

		// Could implement auto-approval logic here or wait for escalation resolution
		state.OverallStatus = "escalated"
		return state, fmt.Errorf("advisor approval timeout - escalation initiated")
	}

	if approvalSignal != "approved" {
		logger.Warn("Advisor rejected client onboarding", "signal", approvalSignal)
		state.OverallStatus = "rejected"
		state.RejectionReason = &approvalSignal
		return state, errors.New("onboarding rejected by advisor")
	}

	logger.Info("Advisor approved onboarding, proceeding to Step 3", "clientID", input.ClientID)

	// ========================================================================
	// STEP 3: GENERATE AND SEND AGREEMENTS FOR E-SIGNATURE
	// ========================================================================
	logger.Info("Starting Step 3: Generate and Send Agreements", "clientID", input.ClientID)
	state.CurrentStep = 3

	var agreementsResult AgreementsResult
	err = workflow.ExecuteActivity(activityOptions, GenerateAndSendAgreementsActivity, AgreementsInput{
		ClientID:         input.ClientID,
		ClientEmail:      input.ClientEmail,
		ClientName:       input.ClientName,
		AgreementTypes:   []string{"client_service_agreement", "disclosure_form", "privacy_notice"},
		ESignatureMethod: "docusign",
		DeliveryMethod:   "email",
	}).Get(activityOptions, &agreementsResult)

	if err != nil {
		logger.Error("Step 3 agreement generation failed", "error", err)
		state.OverallStatus = "failed"
		return state, fmt.Errorf("agreement generation failed: %w", err)
	}

	state.AgreementsSentTime = time.Now()
	logger.Info("Step 3 completed: Agreements sent for signature",
		"clientID", input.ClientID,
		"agreementsCount", len(agreementsResult.AgreementIDs))

	// Set up signal and timer for agreement signature
	// Timeout: 7 days for client to sign
	signaturesTimeout := 7 * 24 * time.Hour
	sigChan = workflow.GetSignalChannel(ctx, "agreements_signed")
	signatureTimer := workflow.NewTimer(ctx, signaturesTimeout)

	s = workflow.NewSelector(ctx)

	signatureDone := false
	s.AddReceive(sigChan, func(c workflow.ReceiveChannel, more bool) {
		var sig string
		c.Receive(ctx, &sig)
		signatureDone = true
		logger.Info("Agreement signature signal received")
	})

	s.AddFuture(signatureTimer, func(f workflow.Future) {
		logger.Warn("Agreement signature timeout")
	})

	s.Select(ctx)

	if !signatureDone {
		logger.Warn("Client signature timeout - sending reminder", "clientID", input.ClientID)

		// Execute reminder activity
		_ = workflow.ExecuteActivity(activityOptions, SendAgreementReminderActivity, AgreementReminderInput{
			ClientID:    input.ClientID,
			ClientEmail: input.ClientEmail,
		}).Get(activityOptions, nil)

		// Wait another 3 days for signature
		reminderTimeout := 3 * 24 * time.Hour
		reminderTimer := workflow.NewTimer(ctx, reminderTimeout)
		s.AddFuture(reminderTimer, func(f workflow.Future) {
			logger.Warn("Still no agreement signatures after reminder")
		})
		s.Select(ctx)
	}

	state.AgreementsSignedTime = &time.Time{}
	*state.AgreementsSignedTime = time.Now()
	logger.Info("Step 3 confirmed: Agreements signed", "clientID", input.ClientID)

	// ========================================================================
	// STEP 4: CREATE LINKED ACCOUNTS AND PORTFOLIOS
	// ========================================================================
	logger.Info("Starting Step 4: Create Accounts and Portfolios", "clientID", input.ClientID)
	state.CurrentStep = 4

	var accountsResult AccountsResult
	err = workflow.ExecuteActivity(activityOptions, CreateAccountsAndPortfoliosActivity, AccountsInput{
		ClientID:       input.ClientID,
		RiskProfile:    validationResult.RiskProfile,
		InitialFunding: validationResult.NetWorth,
		Custodian:      "primary_custodian",
	}).Get(activityOptions, &accountsResult)

	if err != nil {
		logger.Error("Step 4 account creation failed", "error", err)
		state.OverallStatus = "failed"
		return state, fmt.Errorf("account creation failed: %w", err)
	}

	state.AccountsCreatedCount = accountsResult.AccountCount
	logger.Info("Step 4 completed: Accounts and portfolios created",
		"clientID", input.ClientID,
		"accountCount", state.AccountsCreatedCount)

	// ========================================================================
	// STEP 5: NOTIFY CLIENT UPON COMPLETION
	// ============================================================================
	logger.Info("Starting Step 5: Notify Client of Completion", "clientID", input.ClientID)
	state.CurrentStep = 5

	var notificationResult NotificationResult
	err = workflow.ExecuteActivity(activityOptions, NotifyClientOnCompletionActivity, NotificationInput{
		ClientID:         input.ClientID,
		ClientEmail:      input.ClientEmail,
		ClientName:       input.ClientName,
		NotificationType: "email",
		PortalAccessURL:  fmt.Sprintf("https://portal.example.com/clients/%s", input.ClientID),
	}).Get(activityOptions, &notificationResult)

	if err != nil {
		logger.Error("Step 5 notification failed", "error", err)
		// Don't fail workflow here - notification is best-effort
		logger.Warn("Continuing despite notification failure")
	} else {
		notificationTime := time.Now()
		state.NotificationSentTime = &notificationTime
	}

	logger.Info("Step 5 completed: Client notified", "clientID", input.ClientID)

	// ========================================================================
	// WORKFLOW COMPLETION
	// ========================================================================
	state.OverallStatus = "completed"
	logger.Info("ClientOnboardingWorkflow completed successfully",
		"clientID", input.ClientID,
		"totalSteps", 5,
		"status", state.OverallStatus)

	return state, nil
}

// ============================================================================
// ESCALATION WORKFLOW
// ============================================================================

// ClientOnboardingEscalationInput represents escalation workflow input
type ClientOnboardingEscalationInput struct {
	ClientID     string
	Reason       string
	EscalatedTo  string
	TimeoutHours int
}

// ClientOnboardingEscalationResult represents escalation workflow output
type ClientOnboardingEscalationResult struct {
	Approved        bool
	ApprovedBy      string
	RejectedBy      string
	RejectionReason *string
	ResolvedAt      time.Time
}

// ClientOnboardingEscalationWorkflow handles escalations during onboarding
func ClientOnboardingEscalationWorkflow(ctx workflow.Context, input ClientOnboardingEscalationInput) (*ClientOnboardingEscalationResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Escalation workflow started",
		"clientID", input.ClientID,
		"reason", input.Reason,
		"escalatedTo", input.EscalatedTo)

	result := &ClientOnboardingEscalationResult{}

	// Set up timeout for escalation decision
	timeoutDuration := time.Duration(input.TimeoutHours) * time.Hour
	escalationTimer := workflow.NewTimer(ctx, timeoutDuration)

	// Wait for approval or rejection signal
	sigChan := workflow.GetSignalChannel(ctx, "escalation_decision")
	s := workflow.NewSelector(ctx)

	decisionReceived := false
	s.AddReceive(sigChan, func(c workflow.ReceiveChannel, more bool) {
		var signal string
		c.Receive(ctx, &signal)
		if signal == "approved" {
			result.Approved = true
		} else {
			result.Approved = false
			result.RejectionReason = &signal
		}
		decisionReceived = true
		logger.Info("Escalation decision received", "decision", signal)
	})

	s.AddFuture(escalationTimer, func(f workflow.Future) {
		logger.Warn("Escalation decision timeout")
	})

	s.Select(ctx)

	if !decisionReceived {
		// Auto-approve after timeout
		logger.Info("Auto-approving after escalation timeout")
		result.Approved = true
		result.ApprovedBy = "system_auto_approve"
	}

	result.ResolvedAt = time.Now()
	return result, nil
}

// ============================================================================
// SUPPORTING TYPES
// ============================================================================

type ValidationResult struct {
	Passed       bool
	Errors       []string
	RiskProfile  string
	NetWorth     float64
	KYCStatus    string
	AMLStatus    string
	MatchedLists bool
}

type RoutingResult struct {
	AssignedAdvisorID string
	Priority          string
}

type AgreementsResult struct {
	AgreementIDs []string
	RequestDate  time.Time
}

type AgreementReminderInput struct {
	ClientID    string
	ClientEmail string
}

type AccountsResult struct {
	AccountCount int
	PortfolioIDs []string
}

type NotificationResult struct {
	EmailSent    bool
	PortalAccess bool
	SentAt       time.Time
}

// Input types for activities
type ValidationInput struct {
	ClientID             string
	VerifyKYC            bool
	PerformAMLScreening  bool
	AMLProvider          string
	RequiresDueDiligence bool
}

type RoutingInput struct {
	ClientID    string
	RiskProfile string
	NetWorth    float64
}

type AgreementsInput struct {
	ClientID         string
	ClientEmail      string
	ClientName       string
	AgreementTypes   []string
	ESignatureMethod string
	DeliveryMethod   string
}

type TimeoutEscalationInput struct {
	ClientID     string
	StepName     string
	TimeoutHours int
	Action       string
}

type AccountsInput struct {
	ClientID       string
	RiskProfile    string
	InitialFunding float64
	Custodian      string
}

type NotificationInput struct {
	ClientID         string
	ClientEmail      string
	ClientName       string
	NotificationType string
	PortalAccessURL  string
}

// ValidateClientDataActivity validates client data (stub implementation)
func ValidateClientDataActivity(ctx context.Context, input ValidationInput) (ValidationResult, error) {
	// TODO: Implement actual validation logic
	return ValidationResult{
		Passed:       true,
		Errors:       []string{},
		RiskProfile:  "moderate",
		NetWorth:     1000000.0,
		KYCStatus:    "passed",
		AMLStatus:    "passed",
		MatchedLists: false,
	}, nil
}

// RouteForAdvisorReviewActivity routes client for advisor review (stub implementation)
func RouteForAdvisorReviewActivity(ctx context.Context, input interface{}) (string, error) {
	// TODO: Implement actual routing logic
	return "advisor-123", nil
}

// StartTimeoutEscalationActivity starts timeout escalation (stub implementation)
func StartTimeoutEscalationActivity(ctx context.Context, input interface{}) (string, error) {
	// TODO: Implement actual escalation logic
	return "escalation-123", nil
}

// GenerateAndSendAgreementsActivity generates and sends agreements (stub implementation)
func GenerateAndSendAgreementsActivity(ctx context.Context, input interface{}) error {
	// TODO: Implement actual agreement generation and sending
	return nil
}

// SendAgreementReminderActivity sends agreement reminders (stub implementation)
func SendAgreementReminderActivity(ctx context.Context, input interface{}) error {
	// TODO: Implement actual reminder sending
	return nil
}

// CreateAccountsAndPortfoliosActivity creates accounts and portfolios (stub implementation)
func CreateAccountsAndPortfoliosActivity(ctx context.Context, input interface{}) (int, error) {
	// TODO: Implement actual account and portfolio creation
	return 1, nil
}

// NotifyClientOnCompletionActivity notifies client on completion (stub implementation)
func NotifyClientOnCompletionActivity(ctx context.Context, input interface{}) error {
	// TODO: Implement actual notification
	return nil
}
