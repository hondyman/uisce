package onboarding

import (
	"time"

	"github.com/google/uuid"
)

// OnboardingStep represents the current step in the onboarding process
type OnboardingStep int

const (
	StepPersonalInfo   OnboardingStep = 1
	StepEmployment     OnboardingStep = 2
	StepGoals          OnboardingStep = 3
	StepRiskAssessment OnboardingStep = 4
	StepDocuments      OnboardingStep = 5
	StepAccountFunding OnboardingStep = 6
	StepSignatures     OnboardingStep = 7
)

// SessionStatus represents the state of an onboarding session
type SessionStatus string

const (
	StatusInProgress SessionStatus = "IN_PROGRESS"
	StatusCompleted  SessionStatus = "COMPLETED"
	StatusAbandoned  SessionStatus = "ABANDONED"
	StatusExpired    SessionStatus = "EXPIRED"
)

// OnboardingSession tracks multi-step onboarding progress
type OnboardingSession struct {
	SessionID    uuid.UUID     `json:"session_id" db:"session_id"`
	ClientID     *uuid.UUID    `json:"client_id" db:"client_id"`
	Email        string        `json:"email" db:"email"`
	CurrentStep  int           `json:"current_step" db:"current_step"`
	TotalSteps   int           `json:"total_steps" db:"total_steps"`
	StepData     []byte        `json:"step_data" db:"step_data"` // JSONB
	Status       SessionStatus `json:"status" db:"status"`
	LastActiveAt time.Time     `json:"last_active_at" db:"last_active_at"`
	CompletedAt  *time.Time    `json:"completed_at" db:"completed_at"`
	IPAddress    *string       `json:"ip_address" db:"ip_address"`
	UserAgent    *string       `json:"user_agent" db:"user_agent"`
	CreatedAt    time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at" db:"updated_at"`
}

// DocumentType represents different document categories
type DocumentType string

const (
	DocDriversLicense DocumentType = "DRIVERS_LICENSE"
	DocPassport       DocumentType = "PASSPORT"
	DocW9             DocumentType = "W9"
	DocW8             DocumentType = "W8"
	DocBankStatement  DocumentType = "BANK_STATEMENT"
	DocProofOfAddress DocumentType = "PROOF_OF_ADDRESS"
	DocTaxReturn      DocumentType = "TAX_RETURN"
	DocOther          DocumentType = "OTHER"
)

// VerificationStatus represents document verification state
type VerificationStatus string

const (
	VerificationPending  VerificationStatus = "PENDING"
	VerificationInReview VerificationStatus = "IN_REVIEW"
	VerificationVerified VerificationStatus = "VERIFIED"
	VerificationRejected VerificationStatus = "REJECTED"
	VerificationExpired  VerificationStatus = "EXPIRED"
)

// UploadedDocument represents a client-uploaded document
type UploadedDocument struct {
	DocumentID          uuid.UUID          `json:"document_id" db:"document_id"`
	ClientID            uuid.UUID          `json:"client_id" db:"client_id"`
	OnboardingSessionID *uuid.UUID         `json:"onboarding_session_id" db:"onboarding_session_id"`
	DocumentType        DocumentType       `json:"document_type" db:"document_type"`
	FileURL             string             `json:"file_url" db:"file_url"`
	FileName            *string            `json:"file_name" db:"file_name"`
	FileSizeBytes       *int64             `json:"file_size_bytes" db:"file_size_bytes"`
	MimeType            *string            `json:"mime_type" db:"mime_type"`
	OCRExtractedData    []byte             `json:"ocr_extracted_data" db:"ocr_extracted_data"` // JSONB
	OCRConfidence       *float64           `json:"ocr_confidence" db:"ocr_confidence"`
	VerificationStatus  VerificationStatus `json:"verification_status" db:"verification_status"`
	VerificationNotes   *string            `json:"verification_notes" db:"verification_notes"`
	VerifiedBy          *uuid.UUID         `json:"verified_by" db:"verified_by"`
	VerifiedAt          *time.Time         `json:"verified_at" db:"verified_at"`
	UploadedAt          time.Time          `json:"uploaded_at" db:"uploaded_at"`
}

// ESignatureProvider represents different e-signature services
type ESignatureProvider string

const (
	ProviderDocuSign  ESignatureProvider = "DOCUSIGN"
	ProviderAdobeSign ESignatureProvider = "ADOBE_SIGN"
	ProviderInternal  ESignatureProvider = "INTERNAL"
)

// SignatureStatus represents the state of a signature request
type SignatureStatus string

const (
	SignatureSent     SignatureStatus = "SENT"
	SignatureViewed   SignatureStatus = "VIEWED"
	SignatureSigned   SignatureStatus = "SIGNED"
	SignatureDeclined SignatureStatus = "DECLINED"
	SignatureExpired  SignatureStatus = "EXPIRED"
	SignatureVoided   SignatureStatus = "VOIDED"
)

// ESignature tracks electronic signature requests
type ESignature struct {
	SignatureID         uuid.UUID          `json:"signature_id" db:"signature_id"`
	ClientID            uuid.UUID          `json:"client_id" db:"client_id"`
	OnboardingSessionID *uuid.UUID         `json:"onboarding_session_id" db:"onboarding_session_id"`
	DocumentName        string             `json:"document_name" db:"document_name"`
	DocumentURL         string             `json:"document_url" db:"document_url"`
	DocumentType        *string            `json:"document_type" db:"document_type"`
	SignatureProvider   ESignatureProvider `json:"signature_provider" db:"signature_provider"`
	ProviderEnvelopeID  *string            `json:"provider_envelope_id" db:"provider_envelope_id"`
	ProviderMetadata    []byte             `json:"provider_metadata" db:"provider_metadata"` // JSONB
	Status              SignatureStatus    `json:"status" db:"status"`
	SentAt              time.Time          `json:"sent_at" db:"sent_at"`
	ViewedAt            *time.Time         `json:"viewed_at" db:"viewed_at"`
	SignedAt            *time.Time         `json:"signed_at" db:"signed_at"`
	IPAddress           *string            `json:"ip_address" db:"ip_address"`
	UserAgent           *string            `json:"user_agent" db:"user_agent"`
	SignatureImageURL   *string            `json:"signature_image_url" db:"signature_image_url"`
	CreatedAt           time.Time          `json:"created_at" db:"created_at"`
}

// PersonalInfoData represents Step 1 data
type PersonalInfoData struct {
	FirstName         string `json:"first_name"`
	MiddleName        string `json:"middle_name,omitempty"`
	LastName          string `json:"last_name"`
	DateOfBirth       string `json:"date_of_birth"` // YYYY-MM-DD
	SSN               string `json:"ssn"`
	Phone             string `json:"phone"`
	Email             string `json:"email"`
	AddressLine1      string `json:"address_line_1"`
	AddressLine2      string `json:"address_line_2,omitempty"`
	City              string `json:"city"`
	State             string `json:"state"`
	ZipCode           string `json:"zip_code"`
	Country           string `json:"country"`
	CitizenshipStatus string `json:"citizenship_status"` // US_CITIZEN, PERMANENT_RESIDENT, etc.
}

// EmploymentData represents Step 2 data
type EmploymentData struct {
	EmploymentStatus string  `json:"employment_status"` // EMPLOYED, SELF_EMPLOYED, RETIRED, UNEMPLOYED
	Employer         string  `json:"employer,omitempty"`
	Occupation       string  `json:"occupation,omitempty"`
	AnnualIncome     float64 `json:"annual_income"`
	NetWorth         float64 `json:"net_worth,omitempty"`
	LiquidNetWorth   float64 `json:"liquid_net_worth,omitempty"`
}

// GoalsData represents Step 3 data
type GoalsData struct {
	PrimaryGoal     string   `json:"primary_goal"` // RETIREMENT, WEALTH_ACCUMULATION, etc.
	TimeHorizon     int      `json:"time_horizon"` // Years
	AdditionalGoals []string `json:"additional_goals,omitempty"`
}

// RiskAssessmentData represents Step 4 data
type RiskAssessmentData struct {
	InvestmentExperience string                 `json:"investment_experience"` // NONE, LIMITED, MODERATE, EXTENSIVE
	RiskTolerance        string                 `json:"risk_tolerance"`        // CONSERVATIVE, MODERATE, AGGRESSIVE
	TimeHorizon          int                    `json:"time_horizon"`
	LiquidityNeeds       string                 `json:"liquidity_needs"` // LOW, MEDIUM, HIGH
	QuestionnaireAnswers map[string]interface{} `json:"questionnaire_answers"`
}

// OCRExtractedData represents data extracted from documents
type OCRExtractedData struct {
	// For IDs (driver's license, passport)
	FullName       string `json:"full_name,omitempty"`
	DateOfBirth    string `json:"date_of_birth,omitempty"`
	DocumentNumber string `json:"document_number,omitempty"`
	ExpirationDate string `json:"expiration_date,omitempty"`
	Address        string `json:"address,omitempty"`

	// For tax forms (W9)
	TaxID        string `json:"tax_id,omitempty"`
	LegalName    string `json:"legal_name,omitempty"`
	BusinessName string `json:"business_name,omitempty"`

	// Confidence scores
	OverallConfidence float64 `json:"overall_confidence"`
}
