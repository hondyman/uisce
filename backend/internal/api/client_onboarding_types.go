package api

import (
	"time"

	"github.com/lib/pq"
)

// ============================================================================
// CLIENT ONBOARDING TYPES
// ============================================================================

// Client represents a wealth management client
type Client struct {
	ID                   string     `json:"id"`
	TenantID             string     `json:"tenant_id"`
	DatasourceID         string     `json:"datasource_id"`
	FirstName            string     `json:"first_name"`
	LastName             string     `json:"last_name"`
	Email                string     `json:"email"`
	PhoneNumber          *string    `json:"phone_number,omitempty"`
	IdentificationNumber *string    `json:"identification_number,omitempty"`
	IdentificationType   *string    `json:"identification_type,omitempty"`
	DateOfBirth          *string    `json:"date_of_birth,omitempty"`
	CountryOfCitizenship *string    `json:"country_of_citizenship,omitempty"`
	TaxResidencyCountry  *string    `json:"tax_residency_country,omitempty"`
	RiskProfile          string     `json:"risk_profile"` // low, moderate, high, very_high
	NetWorth             *float64   `json:"net_worth,omitempty"`
	AnnualIncome         *float64   `json:"annual_income,omitempty"`
	InvestmentExperience *string    `json:"investment_experience,omitempty"`
	OnboardingStatus     string     `json:"onboarding_status"`
	OnboardingStage      int        `json:"onboarding_stage"`
	TemporalWorkflowID   *string    `json:"temporal_workflow_id,omitempty"`
	AssignedAdvisorID    *string    `json:"assigned_advisor_id,omitempty"`
	KYCStatus            string     `json:"kyc_status"`
	AMLStatus            string     `json:"aml_status"`
	AMLScreeningProvider *string    `json:"aml_screening_provider,omitempty"`
	AgreementsSentDate   *time.Time `json:"agreements_sent_date,omitempty"`
	AgreementsSignedDate *time.Time `json:"agreements_signed_date,omitempty"`
	CreatedBy            *string    `json:"created_by,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedBy            *string    `json:"updated_by,omitempty"`
	UpdatedAt            time.Time  `json:"updated_at"`
	IsActive             bool       `json:"is_active"`
}

// ClientRequest represents request payload for creating/updating a client
type ClientRequest struct {
	FirstName            string   `json:"first_name" binding:"required"`
	LastName             string   `json:"last_name" binding:"required"`
	Email                string   `json:"email" binding:"required,email"`
	PhoneNumber          *string  `json:"phone_number"`
	IdentificationNumber *string  `json:"identification_number"`
	IdentificationType   *string  `json:"identification_type"`
	DateOfBirth          *string  `json:"date_of_birth"`
	CountryOfCitizenship *string  `json:"country_of_citizenship"`
	TaxResidencyCountry  *string  `json:"tax_residency_country"`
	RiskProfile          string   `json:"risk_profile" binding:"required"`
	NetWorth             *float64 `json:"net_worth"`
	AnnualIncome         *float64 `json:"annual_income"`
	InvestmentExperience *string  `json:"investment_experience"`
	AssignedAdvisorID    *string  `json:"assigned_advisor_id"`
}

// Document represents a client document
type Document struct {
	ID                  string     `json:"id"`
	TenantID            string     `json:"tenant_id"`
	DatasourceID        string     `json:"datasource_id"`
	ClientID            string     `json:"client_id"`
	DocumentType        string     `json:"document_type"`
	DocumentName        string     `json:"document_name"`
	DocumentPath        *string    `json:"document_path,omitempty"`
	FileSize            *int64     `json:"file_size,omitempty"`
	FileType            *string    `json:"file_type,omitempty"`
	Status              string     `json:"status"`
	VerificationStatus  string     `json:"verification_status"`
	VerifiedBy          *string    `json:"verified_by,omitempty"`
	VerifiedAt          *time.Time `json:"verified_at,omitempty"`
	VerificationNotes   *string    `json:"verification_notes,omitempty"`
	IssueDate           *string    `json:"issue_date,omitempty"`
	ExpiryDate          *string    `json:"expiry_date,omitempty"`
	IsExpired           bool       `json:"is_expired"`
	ESignatureRequestID *string    `json:"e_signature_request_id,omitempty"`
	ESignatureStatus    *string    `json:"e_signature_status,omitempty"`
	SignedAt            *time.Time `json:"signed_at,omitempty"`
	CreatedBy           string     `json:"created_by"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

// Contact represents a client contact
type Contact struct {
	ID                      string    `json:"id"`
	TenantID                string    `json:"tenant_id"`
	DatasourceID            string    `json:"datasource_id"`
	ClientID                string    `json:"client_id"`
	ContactType             string    `json:"contact_type"`
	FirstName               string    `json:"first_name"`
	LastName                string    `json:"last_name"`
	Email                   *string   `json:"email,omitempty"`
	PhoneNumber             *string   `json:"phone_number,omitempty"`
	Relationship            *string   `json:"relationship,omitempty"`
	EmployeeID              *string   `json:"employee_id,omitempty"`
	Department              *string   `json:"department,omitempty"`
	IsPrimaryAdvisor        bool      `json:"is_primary_advisor"`
	CanAccessAccounts       bool      `json:"can_access_accounts"`
	CanMakeTrades           bool      `json:"can_make_trades"`
	CanRequestDistributions bool      `json:"can_request_distributions"`
	CreatedBy               string    `json:"created_by"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
	IsActive                bool      `json:"is_active"`
}

// Account represents a client investment account
type Account struct {
	ID             string     `json:"id"`
	TenantID       string     `json:"tenant_id"`
	DatasourceID   string     `json:"datasource_id"`
	ClientID       string     `json:"client_id"`
	AccountNumber  string     `json:"account_number"`
	AccountType    string     `json:"account_type"` // brokerage, ira, sep_ira, rollover_ira, trust
	AccountTitle   *string    `json:"account_title,omitempty"`
	Status         string     `json:"status"`
	InitialBalance float64    `json:"initial_balance"`
	CurrentBalance float64    `json:"current_balance"`
	Currency       string     `json:"currency"`
	CustodianName  *string    `json:"custodian_name,omitempty"`
	CustodianID    *string    `json:"custodian_account_id,omitempty"`
	AllowsMargin   bool       `json:"allows_margin"`
	AllowsOptions  bool       `json:"allows_options"`
	AllowsCrypto   bool       `json:"allows_cryptocurrency"`
	CreatedBy      string     `json:"created_by"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	FundingDate    *time.Time `json:"funding_date,omitempty"`
}

// Portfolio represents a client portfolio
type Portfolio struct {
	ID                 string                 `json:"id"`
	TenantID           string                 `json:"tenant_id"`
	DatasourceID       string                 `json:"datasource_id"`
	AccountID          string                 `json:"account_id"`
	PortfolioName      string                 `json:"portfolio_name"`
	PortfolioType      *string                `json:"portfolio_type,omitempty"`
	Status             string                 `json:"status"`
	AllocationJSON     map[string]interface{} `json:"allocation_json"`
	TargetReturn       *float64               `json:"target_return,omitempty"`
	RiskLevel          *string                `json:"risk_level,omitempty"`
	RebalanceFrequency string                 `json:"rebalance_frequency"`
	LastRebalanceDate  *string                `json:"last_rebalance_date,omitempty"`
	NextRebalanceDate  *string                `json:"next_rebalance_date,omitempty"`
	InceptionDate      string                 `json:"inception_date"`
	TotalMarketValue   *float64               `json:"total_market_value,omitempty"`
	TotalGainLoss      *float64               `json:"total_gain_loss,omitempty"`
	YTDReturn          *float64               `json:"ytd_return,omitempty"`
	HoldingsCount      int                    `json:"holdings_count"`
	CreatedBy          string                 `json:"created_by"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
}

// OnboardingWorkflow represents the state of a client onboarding workflow
type OnboardingWorkflow struct {
	ID                          string                 `json:"id"`
	TenantID                    string                 `json:"tenant_id"`
	DatasourceID                string                 `json:"datasource_id"`
	ClientID                    string                 `json:"client_id"`
	WorkflowID                  string                 `json:"workflow_id"`
	CurrentStep                 int                    `json:"current_step"`
	Step1ValidationStatus       string                 `json:"step_1_validation_status"`
	Step2RoutingStatus          string                 `json:"step_2_routing_status"`
	Step3AgreementsStatus       string                 `json:"step_3_agreements_status"`
	Step4AccountsStatus         string                 `json:"step_4_accounts_status"`
	Step5NotificationStatus     string                 `json:"step_5_notification_status"`
	Step1CompletedAt            *time.Time             `json:"step_1_completed_at,omitempty"`
	Step2CompletedAt            *time.Time             `json:"step_2_completed_at,omitempty"`
	Step3CompletedAt            *time.Time             `json:"step_3_completed_at,omitempty"`
	Step4CompletedAt            *time.Time             `json:"step_4_completed_at,omitempty"`
	Step5CompletedAt            *time.Time             `json:"step_5_completed_at,omitempty"`
	OverallStatus               string                 `json:"overall_status"`
	ApprovedBy                  *string                `json:"approved_by,omitempty"`
	ApprovedAt                  *time.Time             `json:"approved_at,omitempty"`
	RejectedBy                  *string                `json:"rejected_by,omitempty"`
	RejectedAt                  *time.Time             `json:"rejected_at,omitempty"`
	RejectionReason             *string                `json:"rejection_reason,omitempty"`
	TimeoutEscalationWorkflowID *string                `json:"timeout_escalation_workflow_id,omitempty"`
	EscalationStatus            *string                `json:"escalation_status,omitempty"`
	EscalationAction            *string                `json:"escalation_action,omitempty"`
	ValidationErrors            map[string]interface{} `json:"validation_errors,omitempty"`
	WorkflowContext             map[string]interface{} `json:"workflow_context,omitempty"`
	CreatedAt                   time.Time              `json:"created_at"`
	UpdatedAt                   time.Time              `json:"updated_at"`
}

// OnboardingEvent represents an event in the onboarding workflow
type OnboardingEvent struct {
	ID           string                 `json:"id"`
	TenantID     string                 `json:"tenant_id"`
	DatasourceID string                 `json:"datasource_id"`
	WorkflowID   string                 `json:"workflow_id"`
	EventType    string                 `json:"event_type"`
	EventData    map[string]interface{} `json:"event_data,omitempty"`
	TriggeredBy  *string                `json:"triggered_by,omitempty"`
	ActorType    string                 `json:"actor_type"`
	ActorRole    *string                `json:"actor_role,omitempty"`
	StepNumber   *int                   `json:"step_number,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
}

// KYCAMLResult represents KYC/AML screening results
type KYCAMLResult struct {
	ID                string                 `json:"id"`
	TenantID          string                 `json:"tenant_id"`
	DatasourceID      string                 `json:"datasource_id"`
	ClientID          string                 `json:"client_id"`
	ScreeningType     string                 `json:"screening_type"`
	ScreeningProvider *string                `json:"screening_provider,omitempty"`
	ScreeningDate     time.Time              `json:"screening_date"`
	Status            string                 `json:"status"`
	RiskScore         *float64               `json:"risk_score,omitempty"`
	RiskLevel         *string                `json:"risk_level,omitempty"`
	Findings          map[string]interface{} `json:"findings"`
	Matches           *pq.StringArray        `json:"matches,omitempty"`
	RequiresReview    bool                   `json:"requires_review"`
	ReviewedBy        *string                `json:"reviewed_by,omitempty"`
	ReviewedAt        *time.Time             `json:"reviewed_at,omitempty"`
	ReviewNotes       *string                `json:"review_notes,omitempty"`
	EscalationLevel   *string                `json:"escalation_level,omitempty"`
	EscalatedTo       *string                `json:"escalated_to,omitempty"`
	EscalationReason  *string                `json:"escalation_reason,omitempty"`
	CreatedBy         string                 `json:"created_by"`
	CreatedAt         time.Time              `json:"created_at"`
}

// OnboardingNote represents an internal note on client onboarding
type OnboardingNote struct {
	ID                string    `json:"id"`
	TenantID          string    `json:"tenant_id"`
	DatasourceID      string    `json:"datasource_id"`
	ClientID          string    `json:"client_id"`
	NoteType          string    `json:"note_type"`
	Content           string    `json:"content"`
	IsInternal        bool      `json:"is_internal"`
	VisibleToClient   bool      `json:"visible_to_client"`
	RequiredRole      *string   `json:"required_role,omitempty"`
	CreatedBy         string    `json:"created_by"`
	CreatedAt         time.Time `json:"created_at"`
	RelatedStep       *int      `json:"related_step,omitempty"`
	RelatedDocumentID *string   `json:"related_document_id,omitempty"`
}

// ============================================================================
// REQUEST/RESPONSE PAYLOADS
// ============================================================================

// StartOnboardingRequest initiates the client onboarding workflow
type StartOnboardingRequest struct {
	ClientID string `json:"client_id" binding:"required"`
}

// ValidateClientDataRequest triggers step 1 validation
type ValidateClientDataRequest struct {
	ClientID             string  `json:"client_id" binding:"required"`
	VerifyKYC            bool    `json:"verify_kyc" binding:"required"`
	PerformAMLScreening  bool    `json:"perform_aml_screening" binding:"required"`
	AMLProvider          string  `json:"aml_provider"` // lexis_nexis, worldcheck
	RequiresDueDiligence bool    `json:"requires_due_diligence"`
	DueDiligenceReason   *string `json:"due_diligence_reason,omitempty"`
}

// RouteForReviewRequest routes validated client to advisor
type RouteForReviewRequest struct {
	ClientID    string  `json:"client_id" binding:"required"`
	AdvisorID   string  `json:"advisor_id" binding:"required"`
	Priority    string  `json:"priority"` // low, medium, high, urgent
	ReviewNotes *string `json:"review_notes,omitempty"`
}

// GenerateAgreementsRequest generates and sends agreements
type GenerateAgreementsRequest struct {
	ClientID         string   `json:"client_id" binding:"required"`
	AgreementTypes   []string `json:"agreement_types" binding:"required"` // client_agreement, disclosure, etc
	ESignatureMethod string   `json:"e_signature_method"`                 // docusign, hellosign
	DeliveryMethod   string   `json:"delivery_method"`                    // email, portal
}

// CreateAccountsRequest creates investment accounts
type CreateAccountsRequest struct {
	ClientID       string   `json:"client_id" binding:"required"`
	AccountTypes   []string `json:"account_types" binding:"required"` // brokerage, ira, etc
	InitialFunding *float64 `json:"initial_funding,omitempty"`
	Custodian      string   `json:"custodian" binding:"required"`
	BankingAPIRef  *string  `json:"banking_api_ref,omitempty"`
}

// CreatePortfolioRequest creates an initial portfolio for an account
type CreatePortfolioRequest struct {
	AccountID          string                 `json:"account_id" binding:"required"`
	PortfolioName      string                 `json:"portfolio_name" binding:"required"`
	RiskProfile        string                 `json:"risk_profile" binding:"required"`
	AllocationJSON     map[string]interface{} `json:"allocation_json" binding:"required"`
	TargetReturn       *float64               `json:"target_return,omitempty"`
	RebalanceFrequency string                 `json:"rebalance_frequency"`
}

// NotifyClientRequest sends completion notification
type NotifyClientRequest struct {
	ClientID         string  `json:"client_id" binding:"required"`
	NotificationType string  `json:"notification_type"` // email, sms, portal
	PortalAccessURL  *string `json:"portal_access_url,omitempty"`
}

// ApproveOnboardingRequest approves client onboarding
type ApproveOnboardingRequest struct {
	WorkflowID string  `json:"workflow_id" binding:"required"`
	AdvisorID  string  `json:"advisor_id" binding:"required"`
	Notes      *string `json:"notes,omitempty"`
}

// RejectOnboardingRequest rejects client onboarding
type RejectOnboardingRequest struct {
	WorkflowID string `json:"workflow_id" binding:"required"`
	RejectedBy string `json:"rejected_by" binding:"required"`
	Reason     string `json:"reason" binding:"required"`
}

// OnboardingStatusResponse returns onboarding progress
type OnboardingStatusResponse struct {
	ClientID            string            `json:"client_id"`
	OverallStatus       string            `json:"overall_status"`
	CurrentStep         int               `json:"current_step"`
	StepStatuses        map[string]string `json:"step_statuses"`
	CompletionPercent   int               `json:"completion_percent"`
	WorkflowID          string            `json:"workflow_id"`
	EstimatedCompletion *time.Time        `json:"estimated_completion,omitempty"`
	NextAction          string            `json:"next_action"`
	BlockingIssues      []string          `json:"blocking_issues,omitempty"`
}
