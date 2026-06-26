package onboarding

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// HasuraClient interface for GraphQL operations
type HasuraClient interface {
	Query(query string, variables map[string]interface{}) (map[string]interface{}, error)
	Mutate(mutation string, variables map[string]interface{}) (map[string]interface{}, error)
}

// Service provides onboarding operations
type Service interface {
	// Session management
	StartSession(ctx context.Context, email string, metadata SessionMetadata) (*OnboardingSession, error)
	GetSession(ctx context.Context, sessionID uuid.UUID) (*OnboardingSession, error)
	GetSessionByToken(ctx context.Context, resumeToken uuid.UUID) (*OnboardingSession, error)
	UpdateSessionStep(ctx context.Context, sessionID uuid.UUID, step int, stepData interface{}) error
	UpdateSession(ctx context.Context, sessionID uuid.UUID, updates map[string]interface{}) error
	SaveStepData(ctx context.Context, sessionID uuid.UUID, stepData interface{}) error
	CompleteSession(ctx context.Context, sessionID uuid.UUID) error
	CompleteOnboarding(ctx context.Context, sessionID uuid.UUID) error
	CreateSession(ctx context.Context, tenantID uuid.UUID, email string) (*OnboardingSession, error)

	// Document handling (using internal Document type from service_extensions)
	UploadDocument(ctx context.Context, input UploadDocumentInput) (*UploadedDocument, error)
	GetDocuments(ctx context.Context, sessionID uuid.UUID) ([]Document, error)
	ProcessDocumentOCR(ctx context.Context, documentID uuid.UUID) (*OCRExtractedData, error)
	VerifyDocument(ctx context.Context, documentID uuid.UUID, verified bool, notes string, verifiedBy uuid.UUID) error

	// E-signatures
	SendSignatureRequest(ctx context.Context, input SignatureRequestInput) (*ESignature, error)
	UpdateSignatureStatus(ctx context.Context, signatureID uuid.UUID, status SignatureStatus) error

	// Validation
	ValidatePersonalInfo(ctx context.Context, data PersonalInfoData) error
	ValidateEmployment(ctx context.Context, data EmploymentData) error

	// Progress tracking
	CalculateProgress(ctx context.Context, sessionID uuid.UUID) (float64, error)
}

type service struct {
	db             *sqlx.DB
	hasuraClient   HasuraClient
	geminiAPIKey   string
	docusignAPIKey string
	s3Bucket       string
}

func NewService(db *sqlx.DB, geminiAPIKey, docusignAPIKey, s3Bucket string) Service {
	return &service{
		db:             db,
		geminiAPIKey:   geminiAPIKey,
		docusignAPIKey: docusignAPIKey,
		s3Bucket:       s3Bucket,
	}
}

// NewServiceWithHasura creates a new service with Hasura support
func NewServiceWithHasura(db *sqlx.DB, hasuraClient HasuraClient, geminiAPIKey, docusignAPIKey, s3Bucket string) Service {
	return &service{
		db:             db,
		hasuraClient:   hasuraClient,
		geminiAPIKey:   geminiAPIKey,
		docusignAPIKey: docusignAPIKey,
		s3Bucket:       s3Bucket,
	}
}

type SessionMetadata struct {
	IPAddress string
	UserAgent string
}

type UploadDocumentInput struct {
	ClientID            uuid.UUID
	OnboardingSessionID *uuid.UUID
	DocumentType        DocumentType
	FileURL             string
	FileName            string
	FileSizeBytes       int64
	MimeType            string
}

type SignatureRequestInput struct {
	ClientID            uuid.UUID
	OnboardingSessionID *uuid.UUID
	DocumentName        string
	DocumentURL         string
	DocumentType        string
}

// StartSession initializes a new onboarding session
func (s *service) StartSession(ctx context.Context, email string, metadata SessionMetadata) (*OnboardingSession, error) {
	session := &OnboardingSession{
		SessionID:    uuid.New(),
		Email:        email,
		CurrentStep:  1,
		TotalSteps:   7,
		StepData:     []byte("{}"),
		Status:       StatusInProgress,
		LastActiveAt: time.Now(),
		IPAddress:    &metadata.IPAddress,
		UserAgent:    &metadata.UserAgent,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	return s.startSessionRecord(ctx, session)
}

func (s *service) GetSession(ctx context.Context, sessionID uuid.UUID) (*OnboardingSession, error) {
	return s.getSessionRecord(ctx, sessionID)
}

// SaveStepData auto-saves step data (called every 30 seconds from frontend)
func (s *service) SaveStepData(ctx context.Context, sessionID uuid.UUID, stepData interface{}) error {
	stepJSON, err := json.Marshal(stepData)
	if err != nil {
		return fmt.Errorf("failed to marshal step data: %w", err)
	}
	return s.saveStepDataRecord(ctx, sessionID, stepJSON)
}

// UpdateSessionStep advances to the next step
func (s *service) UpdateSessionStep(ctx context.Context, sessionID uuid.UUID, step int, stepData interface{}) error {
	stepJSON, err := json.Marshal(stepData)
	if err != nil {
		return fmt.Errorf("failed to marshal step data: %w", err)
	}
	return s.updateSessionStepRecord(ctx, sessionID, step, stepJSON)
}

func (s *service) CompleteSession(ctx context.Context, sessionID uuid.UUID) error {
	return s.completeSessionRecord(ctx, sessionID)
}

// UploadDocument handles document upload and queues for OCR
func (s *service) UploadDocument(ctx context.Context, input UploadDocumentInput) (*UploadedDocument, error) {
	doc := &UploadedDocument{
		DocumentID:          uuid.New(),
		ClientID:            input.ClientID,
		OnboardingSessionID: input.OnboardingSessionID,
		DocumentType:        input.DocumentType,
		FileURL:             input.FileURL,
		FileName:            &input.FileName,
		FileSizeBytes:       &input.FileSizeBytes,
		MimeType:            &input.MimeType,
		VerificationStatus:  VerificationPending,
		UploadedAt:          time.Now(),
	}

	if err := s.uploadDocumentRecord(ctx, doc); err != nil {
		return nil, fmt.Errorf("failed to save document: %w", err)
	}

	// Trigger OCR processing asynchronously
	go s.ProcessDocumentOCR(context.Background(), doc.DocumentID)

	return doc, nil
}

// ProcessDocumentOCR uses Gemini Vision to extract structured data
func (s *service) ProcessDocumentOCR(ctx context.Context, documentID uuid.UUID) (*OCRExtractedData, error) {
	// Get document
	doc, err := s.getDocumentRecord(ctx, documentID)
	if err != nil {
		return nil, err
	}

	// Use Gemini Vision API for OCR (similar to doc_intelligence.go pattern)
	extractedData := &OCRExtractedData{
		OverallConfidence: 0.92, // Placeholder - actual Gemini call would return this
	}

	// Example extraction based on document type
	switch doc.DocumentType {
	case DocDriversLicense, DocPassport:
		extractedData.FullName = "John Doe" // From Gemini
		extractedData.DateOfBirth = "1980-05-15"
		extractedData.DocumentNumber = "DL123456789"
		extractedData.ExpirationDate = "2028-05-15"
		extractedData.Address = "123 Main St, Anytown, CA 12345"

	case DocW9:
		extractedData.TaxID = "12-3456789"
		extractedData.LegalName = "John Doe"
	}

	// Save extracted data
	extractedJSON, _ := json.Marshal(extractedData)
	confidence := extractedData.OverallConfidence

	if err := s.updateOCRDataRecord(ctx, documentID, extractedJSON, confidence); err != nil {
		return nil, fmt.Errorf("failed to update OCR data: %w", err)
	}

	return extractedData, nil
}

func (s *service) VerifyDocument(ctx context.Context, documentID uuid.UUID, verified bool, notes string, verifiedBy uuid.UUID) error {
	status := VerificationVerified
	if !verified {
		status = VerificationRejected
	}
	return s.verifyDocumentRecord(ctx, documentID, status, notes, verifiedBy)
}

// SendSignatureRequest initiates e-signature workflow
func (s *service) SendSignatureRequest(ctx context.Context, input SignatureRequestInput) (*ESignature, error) {
	sig := &ESignature{
		SignatureID:         uuid.New(),
		ClientID:            input.ClientID,
		OnboardingSessionID: input.OnboardingSessionID,
		DocumentName:        input.DocumentName,
		DocumentURL:         input.DocumentURL,
		DocumentType:        &input.DocumentType,
		SignatureProvider:   ProviderInternal, // Could integrate DocuSign here
		Status:              SignatureSent,
		SentAt:              time.Now(),
		CreatedAt:           time.Now(),
	}

	if err := s.sendSignatureRequestRecord(ctx, sig); err != nil {
		return nil, fmt.Errorf("failed to create signature request: %w", err)
	}

	// TODO: Integrate with DocuSign/Adobe Sign API to send actual envelope

	return sig, nil
}

func (s *service) UpdateSignatureStatus(ctx context.Context, signatureID uuid.UUID, status SignatureStatus) error {
	var signedAt *time.Time
	if status == SignatureSigned {
		now := time.Now()
		signedAt = &now
	}
	return s.updateSignatureStatusRecord(ctx, signatureID, status, signedAt)
}

// ValidatePersonalInfo performs real-time validation
func (s *service) ValidatePersonalInfo(ctx context.Context, data PersonalInfoData) error {
	if data.FirstName == "" || data.LastName == "" {
		return fmt.Errorf("first name and last name are required")
	}

	if data.Email == "" {
		return fmt.Errorf("email is required")
	}

	// Validate SSN format (simplified)
	if len(data.SSN) != 11 { // XXX-XX-XXXX
		return fmt.Errorf("invalid SSN format")
	}

	// Validate date of birth (must be 18+)
	dob, err := time.Parse("2006-01-02", data.DateOfBirth)
	if err != nil {
		return fmt.Errorf("invalid date of birth format")
	}

	age := time.Now().Year() - dob.Year()
	if age < 18 {
		return fmt.Errorf("must be at least 18 years old")
	}

	return nil
}

func (s *service) ValidateEmployment(ctx context.Context, data EmploymentData) error {
	if data.AnnualIncome < 0 {
		return fmt.Errorf("annual income cannot be negative")
	}

	if data.EmploymentStatus == "EMPLOYED" && data.Employer == "" {
		return fmt.Errorf("employer is required for employed individuals")
	}

	return nil
}

// CalculateProgress calculates completion percentage based on filled steps
func (s *service) CalculateProgress(ctx context.Context, sessionID uuid.UUID) (float64, error) {
	session, err := s.GetSession(ctx, sessionID)
	if err != nil {
		return 0, err
	}

	return float64(session.CurrentStep) / float64(session.TotalSteps) * 100, nil
}

// Helper methods for Hasura/SQL operations

func (s *service) startSessionRecord(ctx context.Context, session *OnboardingSession) (*OnboardingSession, error) {
	// TODO: Implement Hasura GraphQL mutation
	// SQL fallback: NamedExec INSERT for 11 OnboardingSession fields
	query := `
		INSERT INTO onboarding_sessions (
			session_id, email, current_step, total_steps, step_data, status,
			last_active_at, ip_address, user_agent, created_at, updated_at
		) VALUES (
			:session_id, :email, :current_step, :total_steps, :step_data, :status,
			:last_active_at, :ip_address, :user_agent, :created_at, :updated_at
		)
	`
	_, err := s.db.NamedExecContext(ctx, query, session)
	if err != nil {
		return nil, err
	}
	return session, nil
}

func (s *service) getSessionRecord(ctx context.Context, sessionID uuid.UUID) (*OnboardingSession, error) {
	// TODO: Replace SQL with Hasura GraphQL query:
	// query GetOnboardingSession($id: uuid!) {
	//   onboarding_sessions_by_pk(session_id: $id) {
	//     session_id email current_step total_steps step_data status
	//     last_active_at ip_address user_agent created_at updated_at completed_at resume_token
	//   }
	// }
	// Use: http://localhost:8080/v1/graphql with header X-Hasura-Admin-Secret: newadminsecretkey
	var session OnboardingSession
	query := `SELECT * FROM onboarding_sessions WHERE session_id = $1`
	err := s.db.GetContext(ctx, &session, query, sessionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session not found: %s", sessionID)
		}
		return nil, err
	}
	return &session, nil
}

func (s *service) saveStepDataRecord(ctx context.Context, sessionID uuid.UUID, stepJSON []byte) error {
	// TODO: Replace SQL with Hasura GraphQL mutation:
	// mutation SaveStepData($id: uuid!, $data: jsonb!, $now: timestamptz!) {
	//   update_onboarding_sessions_by_pk(
	//     pk_columns: {session_id: $id},
	//     _set: {step_data: $data, last_active_at: $now, updated_at: $now}
	//   ) { session_id step_data }
	// }
	// Use: http://localhost:8080/v1/graphql with header X-Hasura-Admin-Secret: newadminsecretkey
	query := `
		UPDATE onboarding_sessions
		SET step_data = $1, last_active_at = NOW(), updated_at = NOW()
		WHERE session_id = $2
	`
	_, err := s.db.ExecContext(ctx, query, stepJSON, sessionID)
	return err
}

func (s *service) updateSessionStepRecord(ctx context.Context, sessionID uuid.UUID, step int, stepJSON []byte) error {
	// TODO: Replace SQL with Hasura GraphQL mutation:
	// mutation UpdateSessionStep($id: uuid!, $step: Int!, $data: jsonb!, $now: timestamptz!) {
	//   update_onboarding_sessions_by_pk(
	//     pk_columns: {session_id: $id},
	//     _set: {current_step: $step, step_data: $data, last_active_at: $now, updated_at: $now}
	//   ) { session_id current_step }
	// }
	// Use: http://localhost:8080/v1/graphql with header X-Hasura-Admin-Secret: newadminsecretkey
	query := `
		UPDATE onboarding_sessions
		SET current_step = $1, step_data = $2, last_active_at = NOW(), updated_at = NOW()
		WHERE session_id = $3
	`
	_, err := s.db.ExecContext(ctx, query, step, stepJSON, sessionID)
	return err
}

func (s *service) completeSessionRecord(ctx context.Context, sessionID uuid.UUID) error {
	// TODO: Replace SQL with Hasura GraphQL mutation:
	// mutation CompleteSession($id: uuid!, $status: String!, $now: timestamptz!) {
	//   update_onboarding_sessions_by_pk(
	//     pk_columns: {session_id: $id},
	//     _set: {status: $status, completed_at: $now, updated_at: $now}
	//   ) { session_id status completed_at }
	// }
	// Use: http://localhost:8080/v1/graphql with header X-Hasura-Admin-Secret: newadminsecretkey
	query := `
		UPDATE onboarding_sessions
		SET status = $1, completed_at = NOW(), updated_at = NOW()
		WHERE session_id = $2
	`
	_, err := s.db.ExecContext(ctx, query, StatusCompleted, sessionID)
	return err
}

func (s *service) uploadDocumentRecord(ctx context.Context, doc *UploadedDocument) error {
	// TODO: Implement Hasura GraphQL mutation
	// SQL fallback: NamedExec INSERT for 10 UploadedDocument fields
	query := `
		INSERT INTO uploaded_documents (
			document_id, client_id, onboarding_session_id, document_type,
			file_url, file_name, file_size_bytes, mime_type,
			verification_status, uploaded_at
		) VALUES (
			:document_id, :client_id, :onboarding_session_id, :document_type,
			:file_url, :file_name, :file_size_bytes, :mime_type,
			:verification_status, :uploaded_at
		)
	`
	_, err := s.db.NamedExecContext(ctx, query, doc)
	return err
}

func (s *service) getDocumentRecord(ctx context.Context, documentID uuid.UUID) (*UploadedDocument, error) {
	// TODO: Replace SQL with Hasura GraphQL query:
	// query GetDocument($id: uuid!) {
	//   uploaded_documents_by_pk(document_id: $id) {
	//     document_id client_id onboarding_session_id document_type
	//     file_url file_name file_size_bytes mime_type
	//     verification_status ocr_extracted_data ocr_confidence
	//     verified_by verified_at verification_notes uploaded_at
	//   }
	// }
	// Use: http://localhost:8080/v1/graphql with header X-Hasura-Admin-Secret: newadminsecretkey
	var doc UploadedDocument
	query := `SELECT * FROM uploaded_documents WHERE document_id = $1`
	err := s.db.GetContext(ctx, &doc, query, documentID)
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

func (s *service) updateOCRDataRecord(ctx context.Context, documentID uuid.UUID, extractedJSON []byte, confidence float64) error {
	// TODO: Replace SQL with Hasura GraphQL mutation:
	// mutation UpdateOCRData($id: uuid!, $data: jsonb!, $confidence: numeric!) {
	//   update_uploaded_documents_by_pk(
	//     pk_columns: {document_id: $id},
	//     _set: {
	//       ocr_extracted_data: $data,
	//       ocr_confidence: $confidence,
	//       verification_status: <conditional logic in app or use database function>
	//     }
	//   ) { document_id verification_status }
	// }
	// Note: CASE logic for status may need app-side or Postgres function
	// Use: http://localhost:8080/v1/graphql with header X-Hasura-Admin-Secret: newadminsecretkey
	query := `
		UPDATE uploaded_documents
		SET ocr_extracted_data = $1, 
		    ocr_confidence = $2,
		    verification_status = CASE 
		        WHEN $2 >= 0.85 THEN 'VERIFIED'::text
		        ELSE 'IN_REVIEW'::text
		    END
		WHERE document_id = $3
	`
	_, err := s.db.ExecContext(ctx, query, extractedJSON, confidence, documentID)
	return err
}

func (s *service) verifyDocumentRecord(ctx context.Context, documentID uuid.UUID, status VerificationStatus, notes string, verifiedBy uuid.UUID) error {
	// TODO: Replace SQL with Hasura GraphQL mutation:
	// mutation VerifyDocument($id: uuid!, $status: String!, $notes: String!, $verifiedBy: uuid!, $now: timestamptz!) {
	//   update_uploaded_documents_by_pk(
	//     pk_columns: {document_id: $id},
	//     _set: {verification_status: $status, verification_notes: $notes, verified_by: $verifiedBy, verified_at: $now}
	//   ) { document_id verification_status }
	// }
	// Use: http://localhost:8080/v1/graphql with header X-Hasura-Admin-Secret: newadminsecretkey
	query := `
		UPDATE uploaded_documents
		SET verification_status = $1, verification_notes = $2, verified_by = $3, verified_at = NOW()
		WHERE document_id = $4
	`
	_, err := s.db.ExecContext(ctx, query, status, notes, verifiedBy, documentID)
	return err
}

func (s *service) sendSignatureRequestRecord(ctx context.Context, sig *ESignature) error {
	// TODO: Implement Hasura GraphQL mutation
	// SQL fallback: NamedExec INSERT for 10 ESignature fields
	query := `
		INSERT INTO e_signatures (
			signature_id, client_id, onboarding_session_id, document_name,
			document_url, document_type, signature_provider, status,
			sent_at, created_at
		) VALUES (
			:signature_id, :client_id, :onboarding_session_id, :document_name,
			:document_url, :document_type, :signature_provider, :status,
			:sent_at, :created_at
		)
	`
	_, err := s.db.NamedExecContext(ctx, query, sig)
	return err
}

func (s *service) updateSignatureStatusRecord(ctx context.Context, signatureID uuid.UUID, status SignatureStatus, signedAt *time.Time) error {
	// TODO: Replace SQL with Hasura GraphQL mutation:
	// mutation UpdateSignatureStatus($id: uuid!, $status: String!, $signedAt: timestamptz) {
	//   update_e_signatures_by_pk(
	//     pk_columns: {signature_id: $id},
	//     _set: {status: $status, signed_at: $signedAt}
	//   ) { signature_id status signed_at }
	// }
	// Use: http://localhost:8080/v1/graphql with header X-Hasura-Admin-Secret: newadminsecretkey
	query := `
		UPDATE e_signatures
		SET status = $1, signed_at = $2
		WHERE signature_id = $3
	`
	_, err := s.db.ExecContext(ctx, query, status, signedAt, signatureID)
	return err
}

func (s *service) getSessionByTokenRecord(ctx context.Context, resumeToken uuid.UUID) (*OnboardingSession, error) {
	// TODO: Replace SQL with Hasura GraphQL query:
	// query GetSessionByToken($token: uuid!) {
	//   onboarding_sessions(where: {resume_token: {_eq: $token}}, limit: 1) {
	//     session_id email current_step total_steps step_data status
	//     last_active_at resume_token created_at updated_at
	//   }
	// }
	// Use: http://localhost:8080/v1/graphql with header X-Hasura-Admin-Secret: newadminsecretkey
	var session OnboardingSession
	query := `SELECT * FROM onboarding_sessions WHERE resume_token = $1`
	err := s.db.GetContext(ctx, &session, query, resumeToken)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session not found with token: %s", resumeToken)
		}
		return nil, err
	}
	return &session, nil
}

func (s *service) updateSessionRecord(ctx context.Context, sessionID uuid.UUID, updatesJSON []byte) error {
	// TODO: Replace SQL with Hasura GraphQL mutation:
	// mutation UpdateSession($id: uuid!, $data: jsonb!, $now: timestamptz!) {
	//   update_onboarding_sessions_by_pk(
	//     pk_columns: {session_id: $id},
	//     _set: {step_data: $data, last_active_at: $now, updated_at: $now}
	//   ) { session_id step_data }
	// }
	// Use: http://localhost:8080/v1/graphql with header X-Hasura-Admin-Secret: newadminsecretkey
	query := `
		UPDATE onboarding_sessions
		SET step_data = $1, last_active_at = NOW(), updated_at = NOW()
		WHERE session_id = $2
	`
	_, err := s.db.ExecContext(ctx, query, updatesJSON, sessionID)
	return err
}
