package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// ============================================================================
// CLIENT ONBOARDING SERVICE
// ============================================================================

type ClientOnboardingService struct {
	db *sqlx.DB
}

// NewClientOnboardingService creates a new service instance
func NewClientOnboardingService(db *sqlx.DB) *ClientOnboardingService {
	return &ClientOnboardingService{db: db}
}

// ============================================================================
// CLIENT OPERATIONS
// ============================================================================

// CreateClient creates a new client record
func (s *ClientOnboardingService) CreateClient(ctx context.Context, tenantID, datasourceID, userID string, req *ClientRequest) (*Client, error) {
	client := &Client{
		TenantID:             tenantID,
		DatasourceID:         datasourceID,
		FirstName:            req.FirstName,
		LastName:             req.LastName,
		Email:                req.Email,
		PhoneNumber:          req.PhoneNumber,
		IdentificationNumber: req.IdentificationNumber,
		IdentificationType:   req.IdentificationType,
		DateOfBirth:          req.DateOfBirth,
		CountryOfCitizenship: req.CountryOfCitizenship,
		TaxResidencyCountry:  req.TaxResidencyCountry,
		RiskProfile:          req.RiskProfile,
		NetWorth:             req.NetWorth,
		AnnualIncome:         req.AnnualIncome,
		InvestmentExperience: req.InvestmentExperience,
		OnboardingStatus:     "pending_validation",
		OnboardingStage:      1,
		KYCStatus:            "pending",
		AMLStatus:            "pending",
		CreatedBy:            &userID,
		IsActive:             true,
	}

	query := `
		INSERT INTO clients (
			tenant_id, datasource_id, first_name, last_name, email, phone_number,
			identification_number, identification_type, date_of_birth,
			country_of_citizenship, tax_residency_country, risk_profile,
			net_worth, annual_income, investment_experience,
			onboarding_status, onboarding_stage, kyc_status, aml_status,
			assigned_advisor_id, created_by, created_at, updated_at, is_active
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24
		) RETURNING id, created_at, updated_at
	`

	err := s.db.QueryRowxContext(ctx, query,
		tenantID, datasourceID, client.FirstName, client.LastName, client.Email,
		client.PhoneNumber, client.IdentificationNumber, client.IdentificationType,
		client.DateOfBirth, client.CountryOfCitizenship, client.TaxResidencyCountry,
		client.RiskProfile, client.NetWorth, client.AnnualIncome, client.InvestmentExperience,
		client.OnboardingStatus, client.OnboardingStage, client.KYCStatus, client.AMLStatus,
		req.AssignedAdvisorID, userID, time.Now(), time.Now(), true,
	).StructScan(client)

	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return client, nil
}

// GetClient retrieves a client by ID
func (s *ClientOnboardingService) GetClient(ctx context.Context, tenantID, clientID string) (*Client, error) {
	client := &Client{}
	query := `SELECT * FROM clients WHERE id = $1 AND tenant_id = $2`

	err := s.db.GetContext(ctx, client, query, clientID, tenantID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("client not found")
		}
		return nil, fmt.Errorf("failed to get client: %w", err)
	}

	return client, nil
}

// ListClientsInOnboarding returns all clients currently in onboarding for a tenant
func (s *ClientOnboardingService) ListClientsInOnboarding(ctx context.Context, tenantID string, limit, offset int) ([]*Client, int, error) {
	clients := []*Client{}

	// Count total
	var total int
	countQuery := `SELECT COUNT(*) FROM clients WHERE tenant_id = $1 AND onboarding_status != 'active' AND is_active = TRUE`
	err := s.db.GetContext(ctx, &total, countQuery, tenantID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count clients: %w", err)
	}

	// Fetch with pagination
	query := `
		SELECT * FROM clients 
		WHERE tenant_id = $1 AND onboarding_status != 'active' AND is_active = TRUE
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	err = s.db.SelectContext(ctx, &clients, query, tenantID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list clients: %w", err)
	}

	return clients, total, nil
}

// UpdateClientStatus updates the onboarding status and stage
func (s *ClientOnboardingService) UpdateClientStatus(ctx context.Context, clientID, status string, stage int) error {
	query := `
		UPDATE clients 
		SET onboarding_status = $1, onboarding_stage = $2, updated_at = $3
		WHERE id = $4
	`

	result, err := s.db.ExecContext(ctx, query, status, stage, time.Now(), clientID)
	if err != nil {
		return fmt.Errorf("failed to update client status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check affected rows: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("client not found")
	}

	return nil
}

// ============================================================================
// DOCUMENT OPERATIONS
// ============================================================================

// CreateDocument creates a new client document
func (s *ClientOnboardingService) CreateDocument(ctx context.Context, tenantID, datasourceID, clientID, userID string, doc *Document) (*Document, error) {
	doc.ID = ""
	doc.TenantID = tenantID
	doc.DatasourceID = datasourceID
	doc.ClientID = clientID
	doc.CreatedBy = userID
	doc.CreatedAt = time.Now()
	doc.UpdatedAt = time.Now()

	if doc.Status == "" {
		doc.Status = "pending_review"
	}
	if doc.VerificationStatus == "" {
		doc.VerificationStatus = "unverified"
	}

	query := `
		INSERT INTO client_documents (
			tenant_id, datasource_id, client_id, document_type, document_name,
			document_path, file_size, file_type, status, verification_status,
			issue_date, expiry_date, created_by, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
		) RETURNING id
	`

	err := s.db.QueryRowContext(ctx, query,
		doc.TenantID, doc.DatasourceID, doc.ClientID, doc.DocumentType,
		doc.DocumentName, doc.DocumentPath, doc.FileSize, doc.FileType,
		doc.Status, doc.VerificationStatus, doc.IssueDate, doc.ExpiryDate,
		doc.CreatedBy, doc.CreatedAt, doc.UpdatedAt,
	).Scan(&doc.ID)

	if err != nil {
		return nil, fmt.Errorf("failed to create document: %w", err)
	}

	return doc, nil
}

// GetClientDocuments retrieves all documents for a client
func (s *ClientOnboardingService) GetClientDocuments(ctx context.Context, clientID string) ([]*Document, error) {
	documents := []*Document{}
	query := `SELECT * FROM client_documents WHERE client_id = $1 ORDER BY created_at DESC`

	err := s.db.SelectContext(ctx, &documents, query, clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to get documents: %w", err)
	}

	return documents, nil
}

// ============================================================================
// ACCOUNT OPERATIONS
// ============================================================================

// CreateAccount creates a new investment account for a client
func (s *ClientOnboardingService) CreateAccount(ctx context.Context, tenantID, datasourceID, clientID, userID string, account *Account) (*Account, error) {
	account.ID = ""
	account.TenantID = tenantID
	account.DatasourceID = datasourceID
	account.ClientID = clientID
	account.CreatedBy = userID
	account.CreatedAt = time.Now()
	account.UpdatedAt = time.Now()

	if account.Status == "" {
		account.Status = "pending_funding"
	}
	if account.Currency == "" {
		account.Currency = "USD"
	}

	query := `
		INSERT INTO client_accounts (
			tenant_id, datasource_id, client_id, account_number, account_type,
			account_title, status, initial_balance, current_balance, currency,
			custodian_name, custodian_account_id, allows_margin, allows_options,
			allows_cryptocurrency, created_by, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18
		) RETURNING id
	`

	err := s.db.QueryRowContext(ctx, query,
		account.TenantID, account.DatasourceID, account.ClientID, account.AccountNumber,
		account.AccountType, account.AccountTitle, account.Status, account.InitialBalance,
		account.CurrentBalance, account.Currency, account.CustodianName, account.CustodianID,
		account.AllowsMargin, account.AllowsOptions, account.AllowsCrypto,
		account.CreatedBy, account.CreatedAt, account.UpdatedAt,
	).Scan(&account.ID)

	if err != nil {
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	return account, nil
}

// GetClientAccounts retrieves all accounts for a client
func (s *ClientOnboardingService) GetClientAccounts(ctx context.Context, clientID string) ([]*Account, error) {
	accounts := []*Account{}
	query := `SELECT * FROM client_accounts WHERE client_id = $1 ORDER BY created_at DESC`

	err := s.db.SelectContext(ctx, &accounts, query, clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to get accounts: %w", err)
	}

	return accounts, nil
}

// ============================================================================
// PORTFOLIO OPERATIONS
// ============================================================================

// CreatePortfolio creates a new portfolio for an account
func (s *ClientOnboardingService) CreatePortfolio(ctx context.Context, tenantID, datasourceID, accountID, userID string, portfolio *Portfolio) (*Portfolio, error) {
	portfolio.ID = ""
	portfolio.TenantID = tenantID
	portfolio.DatasourceID = datasourceID
	portfolio.AccountID = accountID
	portfolio.CreatedBy = userID
	portfolio.CreatedAt = time.Now()
	portfolio.UpdatedAt = time.Now()

	if portfolio.Status == "" {
		portfolio.Status = "active"
	}
	if portfolio.RebalanceFrequency == "" {
		portfolio.RebalanceFrequency = "quarterly"
	}

	allocationJSON, err := json.Marshal(portfolio.AllocationJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal allocation JSON: %w", err)
	}

	query := `
		INSERT INTO client_portfolios (
			tenant_id, datasource_id, account_id, portfolio_name, portfolio_type,
			status, allocation_json, target_return, risk_level, rebalance_frequency,
			inception_date, created_by, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		) RETURNING id
	`

	err = s.db.QueryRowContext(ctx, query,
		portfolio.TenantID, portfolio.DatasourceID, portfolio.AccountID,
		portfolio.PortfolioName, portfolio.PortfolioType, portfolio.Status,
		allocationJSON, portfolio.TargetReturn, portfolio.RiskLevel,
		portfolio.RebalanceFrequency, portfolio.InceptionDate,
		portfolio.CreatedBy, portfolio.CreatedAt, portfolio.UpdatedAt,
	).Scan(&portfolio.ID)

	if err != nil {
		return nil, fmt.Errorf("failed to create portfolio: %w", err)
	}

	return portfolio, nil
}

// ============================================================================
// ONBOARDING WORKFLOW OPERATIONS
// ============================================================================

// CreateOnboardingWorkflow creates a new workflow record
func (s *ClientOnboardingService) CreateOnboardingWorkflow(ctx context.Context, tenantID, datasourceID, clientID, workflowID string) (*OnboardingWorkflow, error) {
	workflow := &OnboardingWorkflow{
		TenantID:                tenantID,
		DatasourceID:            datasourceID,
		ClientID:                clientID,
		WorkflowID:              workflowID,
		CurrentStep:             1,
		Step1ValidationStatus:   "pending",
		Step2RoutingStatus:      "pending",
		Step3AgreementsStatus:   "pending",
		Step4AccountsStatus:     "pending",
		Step5NotificationStatus: "pending",
		OverallStatus:           "in_progress",
	}

	query := `
		INSERT INTO onboarding_workflows (
			tenant_id, datasource_id, client_id, workflow_id,
			current_step, step_1_validation_status, step_2_routing_status,
			step_3_agreements_status, step_4_accounts_status,
			step_5_notification_status, overall_status, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		) RETURNING id
	`

	err := s.db.QueryRowContext(ctx, query,
		workflow.TenantID, workflow.DatasourceID, workflow.ClientID,
		workflow.WorkflowID, workflow.CurrentStep, workflow.Step1ValidationStatus,
		workflow.Step2RoutingStatus, workflow.Step3AgreementsStatus,
		workflow.Step4AccountsStatus, workflow.Step5NotificationStatus,
		workflow.OverallStatus, time.Now(), time.Now(),
	).Scan(&workflow.ID)

	if err != nil {
		return nil, fmt.Errorf("failed to create onboarding workflow: %w", err)
	}

	return workflow, nil
}

// GetOnboardingWorkflow retrieves a workflow by ID
func (s *ClientOnboardingService) GetOnboardingWorkflow(ctx context.Context, tenantID, workflowID string) (*OnboardingWorkflow, error) {
	workflow := &OnboardingWorkflow{}
	query := `SELECT * FROM onboarding_workflows WHERE workflow_id = $1 AND tenant_id = $2`

	err := s.db.GetContext(ctx, workflow, query, workflowID, tenantID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("workflow not found")
		}
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}

	return workflow, nil
}

// UpdateWorkflowStep updates the status of a specific step
func (s *ClientOnboardingService) UpdateWorkflowStep(ctx context.Context, workflowID string, stepNum int, status string, completedAt *time.Time) error {
	stepField := fmt.Sprintf("step_%d_validation_status", stepNum)
	completedField := fmt.Sprintf("step_%d_completed_at", stepNum)

	query := fmt.Sprintf(`
		UPDATE onboarding_workflows 
		SET %s = $1, %s = $2, current_step = $3, updated_at = $4
		WHERE id = $5
	`, stepField, completedField)

	result, err := s.db.ExecContext(ctx, query, status, completedAt, stepNum+1, time.Now(), workflowID)
	if err != nil {
		return fmt.Errorf("failed to update workflow step: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check affected rows: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("workflow not found")
	}

	return nil
}

// ============================================================================
// ONBOARDING EVENTS
// ============================================================================

// RecordOnboardingEvent creates an audit trail event
func (s *ClientOnboardingService) RecordOnboardingEvent(ctx context.Context, event *OnboardingEvent) error {
	eventDataJSON, err := json.Marshal(event.EventData)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %w", err)
	}

	query := `
		INSERT INTO onboarding_events (
			tenant_id, datasource_id, workflow_id, event_type, event_data,
			triggered_by, actor_type, actor_role, step_number, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		)
	`

	_, err = s.db.ExecContext(ctx, query,
		event.TenantID, event.DatasourceID, event.WorkflowID,
		event.EventType, eventDataJSON, event.TriggeredBy,
		event.ActorType, event.ActorRole, event.StepNumber, time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to record event: %w", err)
	}

	return nil
}

// GetWorkflowEvents retrieves all events for a workflow
func (s *ClientOnboardingService) GetWorkflowEvents(ctx context.Context, workflowID string) ([]*OnboardingEvent, error) {
	events := []*OnboardingEvent{}
	query := `SELECT * FROM onboarding_events WHERE workflow_id = $1 ORDER BY created_at ASC`

	err := s.db.SelectContext(ctx, &events, query, workflowID)
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}

	return events, nil
}

// ============================================================================
// KYC/AML SCREENING
// ============================================================================

// CreateKYCAMLResult stores screening results
func (s *ClientOnboardingService) CreateKYCAMLResult(ctx context.Context, tenantID, datasourceID, clientID, userID string, result *KYCAMLResult) (*KYCAMLResult, error) {
	result.TenantID = tenantID
	result.DatasourceID = datasourceID
	result.ClientID = clientID
	result.CreatedBy = userID
	result.CreatedAt = time.Now()

	if result.ScreeningDate.IsZero() {
		result.ScreeningDate = time.Now()
	}

	findingsJSON, err := json.Marshal(result.Findings)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal findings: %w", err)
	}

	query := `
		INSERT INTO kyc_aml_results (
			tenant_id, datasource_id, client_id, screening_type, screening_provider,
			screening_date, status, risk_score, risk_level, findings, matches,
			requires_review, created_by, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		) RETURNING id
	`

	err = s.db.QueryRowContext(ctx, query,
		result.TenantID, result.DatasourceID, result.ClientID,
		result.ScreeningType, result.ScreeningProvider, result.ScreeningDate,
		result.Status, result.RiskScore, result.RiskLevel, findingsJSON,
		result.Matches, result.RequiresReview, result.CreatedBy, result.CreatedAt,
	).Scan(&result.ID)

	if err != nil {
		return nil, fmt.Errorf("failed to create KYC/AML result: %w", err)
	}

	return result, nil
}

// GetClientKYCAMLResults retrieves all screening results for a client
func (s *ClientOnboardingService) GetClientKYCAMLResults(ctx context.Context, clientID string) ([]*KYCAMLResult, error) {
	results := []*KYCAMLResult{}
	query := `SELECT * FROM kyc_aml_results WHERE client_id = $1 ORDER BY screening_date DESC`

	err := s.db.SelectContext(ctx, &results, query, clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to get KYC/AML results: %w", err)
	}

	return results, nil
}

// ============================================================================
// SUMMARY VIEW METHODS
// ============================================================================

// GetOnboardingStatus returns the current status and progress
func (s *ClientOnboardingService) GetOnboardingStatus(ctx context.Context, tenantID, clientID string) (*OnboardingStatusResponse, error) {
	client, err := s.GetClient(ctx, tenantID, clientID)
	if err != nil {
		return nil, err
	}

	// Get workflow
	var workflowID string
	query := `SELECT workflow_id FROM onboarding_workflows WHERE client_id = $1 LIMIT 1`
	err = s.db.GetContext(ctx, &workflowID, query, clientID)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}

	status := &OnboardingStatusResponse{
		ClientID:      clientID,
		OverallStatus: client.OnboardingStatus,
		CurrentStep:   client.OnboardingStage,
		WorkflowID:    workflowID,
		StepStatuses: map[string]string{
			"step_1_validation":   "pending",
			"step_2_routing":      "pending",
			"step_3_agreements":   "pending",
			"step_4_accounts":     "pending",
			"step_5_notification": "pending",
		},
	}

	// Calculate completion percentage
	completedSteps := 0
	if client.KYCStatus == "approved" {
		completedSteps++
		status.StepStatuses["step_1_validation"] = "completed"
	}
	if client.OnboardingStatus == "pending_approval" || client.OnboardingStatus == "pending_agreements" {
		completedSteps++
		status.StepStatuses["step_2_routing"] = "completed"
	}
	if client.AgreementsSignedDate != nil {
		completedSteps++
		status.StepStatuses["step_3_agreements"] = "completed"
	}

	accounts, _ := s.GetClientAccounts(ctx, clientID)
	if len(accounts) > 0 {
		completedSteps++
		status.StepStatuses["step_4_accounts"] = "completed"
	}

	status.CompletionPercent = (completedSteps * 100) / 5

	return status, nil
}
