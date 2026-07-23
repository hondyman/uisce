package altinv

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Service provides alternative investment management operations
type Service interface {
	// Alternative Investments
	CreateInvestment(ctx context.Context, input CreateInvestmentInput) (*AlternativeInvestment, error)
	GetInvestment(ctx context.Context, investmentID uuid.UUID) (*AlternativeInvestment, error)
	ListInvestmentsByClient(ctx context.Context, clientID uuid.UUID) ([]*AlternativeInvestment, error)
	UpdateInvestment(ctx context.Context, investmentID uuid.UUID, input UpdateInvestmentInput) (*AlternativeInvestment, error)
	DeleteInvestment(ctx context.Context, investmentID uuid.UUID) error

	// Performance
	GetInvestmentPerformance(ctx context.Context, investmentID uuid.UUID) (*InvestmentPerformance, error)
	ListInvestmentPerformances(ctx context.Context, clientID uuid.UUID) ([]*InvestmentPerformance, error)

	// Capital Calls
	CreateCapitalCall(ctx context.Context, input CreateCapitalCallInput) (*CapitalCall, error)
	GetCapitalCall(ctx context.Context, callID uuid.UUID) (*CapitalCall, error)
	ListCapitalCallsByInvestment(ctx context.Context, investmentID uuid.UUID) ([]*CapitalCall, error)
	ListUpcomingCapitalCalls(ctx context.Context, clientID *uuid.UUID) ([]*UpcomingCapitalCall, error)
	UpdateCapitalCallStatus(ctx context.Context, callID uuid.UUID, status CapitalCallStatus, amountFunded float64) error

	// Distributions
	CreateDistribution(ctx context.Context, input CreateDistributionInput) (*Distribution, error)
	GetDistribution(ctx context.Context, distributionID uuid.UUID) (*Distribution, error)
	ListDistributionsByInvestment(ctx context.Context, investmentID uuid.UUID) ([]*Distribution, error)

	// Documents
	CreateDocument(ctx context.Context, input CreateDocumentInput) (*AltInvestmentDocument, error)
	GetDocument(ctx context.Context, documentID uuid.UUID) (*AltInvestmentDocument, error)
	ListDocumentsByInvestment(ctx context.Context, investmentID uuid.UUID) ([]*AltInvestmentDocument, error)
	UpdateDocumentExtraction(ctx context.Context, documentID uuid.UUID, extractedData json.RawMessage, confidence float64, status ExtractionStatus) error
}

type service struct {
	db *sqlx.DB
}

// NewService creates a new alternative investment service
func NewService(db *sqlx.DB) Service {
	return &service{db: db}
}

// CreateInvestmentInput represents input for creating an alternative investment
type CreateInvestmentInput struct {
	ClientID              uuid.UUID
	InvestmentType        InvestmentType
	FundName              string
	GeneralPartner        *string
	VintageYear           *int
	TotalCommitmentAmount float64
	UnfundedCommitment    float64
	RedemptionFrequency   *RedemptionFrequency
	LockUpEndDate         *time.Time
	Metadata              json.RawMessage
	CreatedBy             *uuid.UUID
}

// TODO: Migrate to Hasura GraphQL mutation:
//
//	mutation CreateInvestment($object: alternative_investments_insert_input!) {
//	  insert_alternative_investments_one(object: $object) {
//	    investment_id
//	    client_id
//	    investment_type
//	    fund_name
//	    ...
//	  }
//	}
func (s *service) CreateInvestment(ctx context.Context, input CreateInvestmentInput) (*AlternativeInvestment, error) {
	inv := &AlternativeInvestment{
		InvestmentID:          uuid.New(),
		ClientID:              input.ClientID,
		InvestmentType:        input.InvestmentType,
		FundName:              input.FundName,
		GeneralPartner:        input.GeneralPartner,
		VintageYear:           input.VintageYear,
		TotalCommitmentAmount: input.TotalCommitmentAmount,
		UnfundedCommitment:    input.UnfundedCommitment,
		TotalCapitalCalled:    0,
		TotalDistributions:    0,
		RedemptionFrequency:   input.RedemptionFrequency,
		LockUpEndDate:         input.LockUpEndDate,
		Metadata:              input.Metadata,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
		CreatedBy:             input.CreatedBy,
	}

	if inv.Metadata == nil {
		inv.Metadata = json.RawMessage("{}")
	}

	query := `
		INSERT INTO alternative_investments (
			investment_id, client_id, investment_type, fund_name, general_partner, vintage_year,
			total_commitment_amount, unfunded_commitment, total_capital_called, total_distributions,
			redemption_frequency, lock_up_end_date, metadata, created_at, updated_at, created_by
		) VALUES (
			:investment_id, :client_id, :investment_type, :fund_name, :general_partner, :vintage_year,
			:total_commitment_amount, :unfunded_commitment, :total_capital_called, :total_distributions,
			:redemption_frequency, :lock_up_end_date, :metadata, :created_at, :updated_at, :created_by
		)
	`
	_, err := s.db.NamedExecContext(ctx, query, inv)
	if err != nil {
		return nil, fmt.Errorf("failed to create alternative investment: %w", err)
	}

	return inv, nil
}

// TODO: Migrate to Hasura GraphQL query:
//
//	query GetInvestment($investment_id: uuid!) {
//	  alternative_investments_by_pk(investment_id: $investment_id) { ... }
//	}
func (s *service) GetInvestment(ctx context.Context, investmentID uuid.UUID) (*AlternativeInvestment, error) {
	var inv AlternativeInvestment
	query := `SELECT * FROM alternative_investments WHERE investment_id = $1`
	err := s.db.GetContext(ctx, &inv, query, investmentID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("investment not found: %s", investmentID)
		}
		return nil, fmt.Errorf("failed to get alternative investment: %w", err)
	}
	return &inv, nil
}

// TODO: Migrate to Hasura GraphQL query:
//
//	query ListInvestmentsByClient($client_id: uuid!) {
//	  alternative_investments(where: {client_id: {_eq: $client_id}}, order_by: {fund_name: asc}) { ... }
//	}
func (s *service) ListInvestmentsByClient(ctx context.Context, clientID uuid.UUID) ([]*AlternativeInvestment, error) {
	var invs []*AlternativeInvestment
	query := `SELECT * FROM alternative_investments WHERE client_id = $1 ORDER BY fund_name`
	err := s.db.SelectContext(ctx, &invs, query, clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to list alternative investments: %w", err)
	}
	return invs, nil
}

// UpdateInvestmentInput represents fields that can be updated
type UpdateInvestmentInput struct {
	CurrentNAV         *float64
	NAVDate            *time.Time
	ValuationSource    *ValuationSource
	IRRSinceInception  *float64
	TVPI               *float64
	DPI                *float64
	RVPI               *float64
	MOIC               *float64
	UnfundedCommitment *float64
	Metadata           json.RawMessage
	UpdatedBy          *uuid.UUID
}

// TODO: Migrate to Hasura GraphQL mutation with dynamic _set:
//
//	mutation UpdateInvestment($investment_id: uuid!, $_set: alternative_investments_set_input!) {
//	  update_alternative_investments_by_pk(pk_columns: {investment_id: $investment_id}, _set: $_set) { ... }
//	}
//
// Note: Dynamic UPDATE with optional fields (NAV, IRR, TVPI, DPI, RVPI, MOIC)
func (s *service) UpdateInvestment(ctx context.Context, investmentID uuid.UUID, input UpdateInvestmentInput) (*AlternativeInvestment, error) {
	// Build dynamic update query
	query := `
		UPDATE alternative_investments SET
			updated_at = $1,
			updated_by = $2
	`
	args := []interface{}{time.Now(), input.UpdatedBy}
	argIdx := 3

	if input.CurrentNAV != nil {
		query += fmt.Sprintf(", current_nav = $%d", argIdx)
		args = append(args, input.CurrentNAV)
		argIdx++
	}
	if input.NAVDate != nil {
		query += fmt.Sprintf(", nav_date = $%d", argIdx)
		args = append(args, input.NAVDate)
		argIdx++
	}
	if input.ValuationSource != nil {
		query += fmt.Sprintf(", valuation_source = $%d", argIdx)
		args = append(args, input.ValuationSource)
		argIdx++
	}
	if input.IRRSinceInception != nil {
		query += fmt.Sprintf(", irr_since_inception = $%d", argIdx)
		args = append(args, input.IRRSinceInception)
		argIdx++
	}
	if input.TVPI != nil {
		query += fmt.Sprintf(", tvpi = $%d", argIdx)
		args = append(args, input.TVPI)
		argIdx++
	}
	if input.DPI != nil {
		query += fmt.Sprintf(", dpi = $%d", argIdx)
		args = append(args, input.DPI)
		argIdx++
	}
	if input.RVPI != nil {
		query += fmt.Sprintf(", rvpi = $%d", argIdx)
		args = append(args, input.RVPI)
		argIdx++
	}
	if input.MOIC != nil {
		query += fmt.Sprintf(", moic = $%d", argIdx)
		args = append(args, input.MOIC)
		argIdx++
	}
	if input.UnfundedCommitment != nil {
		query += fmt.Sprintf(", unfunded_commitment = $%d", argIdx)
		args = append(args, input.UnfundedCommitment)
		argIdx++
	}
	if input.Metadata != nil {
		query += fmt.Sprintf(", metadata = $%d", argIdx)
		args = append(args, input.Metadata)
		argIdx++
	}

	query += fmt.Sprintf(" WHERE investment_id = $%d", argIdx)
	args = append(args, investmentID)

	_, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update alternative investment: %w", err)
	}

	return s.GetInvestment(ctx, investmentID)
}

// TODO: Migrate to Hasura GraphQL mutation:
//
//	mutation DeleteInvestment($investment_id: uuid!) {
//	  delete_alternative_investments_by_pk(investment_id: $investment_id) { investment_id }
//	}
func (s *service) DeleteInvestment(ctx context.Context, investmentID uuid.UUID) error {
	query := `DELETE FROM alternative_investments WHERE investment_id = $1`
	_, err := s.db.ExecContext(ctx, query, investmentID)
	if err != nil {
		return fmt.Errorf("failed to delete alternative investment: %w", err)
	}
	return nil
}

// TODO: Migrate to Hasura GraphQL query:
//
//	query GetInvestmentPerformance($investment_id: uuid!) {
//	  alt_investment_performance(where: {investment_id: {_eq: $investment_id}}, limit: 1) { ... }
//	}
func (s *service) GetInvestmentPerformance(ctx context.Context, investmentID uuid.UUID) (*InvestmentPerformance, error) {
	var perf InvestmentPerformance
	query := `SELECT * FROM alt_investment_performance WHERE investment_id = $1`
	err := s.db.GetContext(ctx, &perf, query, investmentID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("investment performance not found: %s", investmentID)
		}
		return nil, fmt.Errorf("failed to get investment performance: %w", err)
	}
	return &perf, nil
}

// TODO: Migrate to Hasura GraphQL query:
//
//	query ListInvestmentPerformances($client_id: uuid!) {
//	  alt_investment_performance(where: {client_id: {_eq: $client_id}}, order_by: {fund_name: asc}) { ... }
//	}
func (s *service) ListInvestmentPerformances(ctx context.Context, clientID uuid.UUID) ([]*InvestmentPerformance, error) {
	var perfs []*InvestmentPerformance
	query := `SELECT * FROM alt_investment_performance WHERE client_id = $1 ORDER BY fund_name`
	err := s.db.SelectContext(ctx, &perfs, query, clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to list investment performances: %w", err)
	}
	return perfs, nil
}

// CreateCapitalCallInput represents input for creating a capital call
type CreateCapitalCallInput struct {
	InvestmentID         uuid.UUID
	NoticeDate           time.Time
	DueDate              time.Time
	AmountRequested      float64
	FundingSourceAccount *uuid.UUID
	AdvisorNotes         *string
	CreatedBy            *uuid.UUID
}

// TODO: Migrate to Hasura GraphQL with transaction (2 mutations):
//
//	mutation CreateCapitalCall($call: capital_calls_insert_input!, $investment_id: uuid!, $amount: numeric!, $date: timestamptz!) {
//	  insert_capital_calls_one(object: $call) { ... }
//	  update_alternative_investments_by_pk(pk_columns: {investment_id: $investment_id}, _set: {last_capital_call_date: $date}, _inc: {total_capital_called: $amount, unfunded_commitment: -$amount}) { ... }
//	}
//
// Note: Requires transaction or use Hasura actions for atomic update
func (s *service) CreateCapitalCall(ctx context.Context, input CreateCapitalCallInput) (*CapitalCall, error) {
	call := &CapitalCall{
		CallID:               uuid.New(),
		InvestmentID:         input.InvestmentID,
		NoticeDate:           input.NoticeDate,
		DueDate:              input.DueDate,
		AmountRequested:      input.AmountRequested,
		AmountFunded:         0,
		Status:               StatusPending,
		FundingSourceAccount: input.FundingSourceAccount,
		AdvisorNotes:         input.AdvisorNotes,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
		CreatedBy:            input.CreatedBy,
	}

	query := `
		INSERT INTO capital_calls (
			call_id, investment_id, notice_date, due_date, amount_requested, amount_funded,
			status, funding_source_account, advisor_notes, created_at, updated_at, created_by
		) VALUES (
			:call_id, :investment_id, :notice_date, :due_date, :amount_requested, :amount_funded,
			:status, :funding_source_account, :advisor_notes, :created_at, :updated_at, :created_by
		)
	`
	_, err := s.db.NamedExecContext(ctx, query, call)
	if err != nil {
		return nil, fmt.Errorf("failed to create capital call: %w", err)
	}

	// Update investment's last capital call date and total capital called
	updateInvQuery := `
		UPDATE alternative_investments
		SET last_capital_call_date = $1, 
		    total_capital_called = total_capital_called + $2,
		    unfunded_commitment = unfunded_commitment - $2
		WHERE investment_id = $3
	`
	_, err = s.db.ExecContext(ctx, updateInvQuery, input.NoticeDate, input.AmountRequested, input.InvestmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to update investment capital call tracking: %w", err)
	}

	return call, nil
}

// TODO: Migrate to Hasura GraphQL query:
//
//	query GetCapitalCall($call_id: uuid!) {
//	  capital_calls_by_pk(call_id: $call_id) { ... }
//	}
func (s *service) GetCapitalCall(ctx context.Context, callID uuid.UUID) (*CapitalCall, error) {
	var call CapitalCall
	query := `SELECT * FROM capital_calls WHERE call_id = $1`
	err := s.db.GetContext(ctx, &call, query, callID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("capital call not found: %s", callID)
		}
		return nil, fmt.Errorf("failed to get capital call: %w", err)
	}
	return &call, nil
}

// TODO: Migrate to Hasura GraphQL query:
//
//	query ListCapitalCallsByInvestment($investment_id: uuid!) {
//	  capital_calls(where: {investment_id: {_eq: $investment_id}}, order_by: {due_date: desc}) { ... }
//	}
func (s *service) ListCapitalCallsByInvestment(ctx context.Context, investmentID uuid.UUID) ([]*CapitalCall, error) {
	var calls []*CapitalCall
	query := `SELECT * FROM capital_calls WHERE investment_id = $1 ORDER BY due_date DESC`
	err := s.db.SelectContext(ctx, &calls, query, investmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list capital calls: %w", err)
	}
	return calls, nil
}

// TODO: Migrate to Hasura GraphQL query with optional where:
//
//	query ListUpcomingCapitalCalls($client_id: uuid) {
//	  upcoming_capital_calls(where: {client_id: {_eq: $client_id}}, order_by: {due_date: asc}) { ... }
//	}
//
// Note: View or materialized view for upcoming calls
func (s *service) ListUpcomingCapitalCalls(ctx context.Context, clientID *uuid.UUID) ([]*UpcomingCapitalCall, error) {
	var calls []*UpcomingCapitalCall
	query := `SELECT * FROM upcoming_capital_calls`
	args := []interface{}{}

	if clientID != nil {
		query += ` WHERE client_id = $1`
		args = append(args, clientID)
	}

	query += ` ORDER BY due_date`

	err := s.db.SelectContext(ctx, &calls, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list upcoming capital calls: %w", err)
	}
	return calls, nil
}

// TODO: Migrate to Hasura GraphQL mutation:
//
//	mutation UpdateCapitalCallStatus($call_id: uuid!, $status: String!, $amount_funded: numeric!) {
//	  update_capital_calls_by_pk(pk_columns: {call_id: $call_id}, _set: {status: $status, amount_funded: $amount_funded, updated_at: "now()"}) { ... }
//	}
func (s *service) UpdateCapitalCallStatus(ctx context.Context, callID uuid.UUID, status CapitalCallStatus, amountFunded float64) error {
	query := `
		UPDATE capital_calls
		SET status = $1, amount_funded = $2, updated_at = $3
		WHERE call_id = $4
	`
	_, err := s.db.ExecContext(ctx, query, status, amountFunded, time.Now(), callID)
	if err != nil {
		return fmt.Errorf("failed to update capital call status: %w", err)
	}
	return nil
}

// CreateDistributionInput represents input for creating a distribution
type CreateDistributionInput struct {
	InvestmentID     uuid.UUID
	DistributionDate time.Time
	Amount           float64
	DistributionType DistributionType
	Reinvested       bool
	TaxYear          *int
	TaxableAmount    *float64
	AdvisorNotes     *string
	CreatedBy        *uuid.UUID
}

// TODO: Migrate to Hasura GraphQL with transaction (2 mutations):
//
//	mutation CreateDistribution($dist: distributions_insert_input!, $investment_id: uuid!, $amount: numeric!, $date: timestamptz!) {
//	  insert_distributions_one(object: $dist) { ... }
//	  update_alternative_investments_by_pk(pk_columns: {investment_id: $investment_id}, _set: {last_distribution_date: $date}, _inc: {total_distributions: $amount}) { ... }
//	}
func (s *service) CreateDistribution(ctx context.Context, input CreateDistributionInput) (*Distribution, error) {
	dist := &Distribution{
		DistributionID:   uuid.New(),
		InvestmentID:     input.InvestmentID,
		DistributionDate: input.DistributionDate,
		Amount:           input.Amount,
		DistributionType: input.DistributionType,
		Reinvested:       input.Reinvested,
		TaxYear:          input.TaxYear,
		TaxableAmount:    input.TaxableAmount,
		AdvisorNotes:     input.AdvisorNotes,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		CreatedBy:        input.CreatedBy,
	}

	query := `
		INSERT INTO distributions (
			distribution_id, investment_id, distribution_date, amount, distribution_type,
			reinvested, tax_year, taxable_amount, advisor_notes, created_at, updated_at, created_by
		) VALUES (
			:distribution_id, :investment_id, :distribution_date, :amount, :distribution_type,
			:reinvested, :tax_year, :taxable_amount, :advisor_notes, :created_at, :updated_at, :created_by
		)
	`
	_, err := s.db.NamedExecContext(ctx, query, dist)
	if err != nil {
		return nil, fmt.Errorf("failed to create distribution: %w", err)
	}

	// Update investment's last distribution date and total distributions
	updateInvQuery := `
		UPDATE alternative_investments
		SET last_distribution_date = $1, total_distributions = total_distributions + $2
		WHERE investment_id = $3
	`
	_, err = s.db.ExecContext(ctx, updateInvQuery, input.DistributionDate, input.Amount, input.InvestmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to update investment distribution tracking: %w", err)
	}

	return dist, nil
}

// TODO: Migrate to Hasura GraphQL query:
//
//	query GetDistribution($distribution_id: uuid!) {
//	  distributions_by_pk(distribution_id: $distribution_id) { ... }
//	}
func (s *service) GetDistribution(ctx context.Context, distributionID uuid.UUID) (*Distribution, error) {
	var dist Distribution
	query := `SELECT * FROM distributions WHERE distribution_id = $1`
	err := s.db.GetContext(ctx, &dist, query, distributionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("distribution not found: %s", distributionID)
		}
		return nil, fmt.Errorf("failed to get distribution: %w", err)
	}
	return &dist, nil
}

// TODO: Migrate to Hasura GraphQL query:
//
//	query ListDistributionsByInvestment($investment_id: uuid!) {
//	  distributions(where: {investment_id: {_eq: $investment_id}}, order_by: {distribution_date: desc}) { ... }
//	}
func (s *service) ListDistributionsByInvestment(ctx context.Context, investmentID uuid.UUID) ([]*Distribution, error) {
	var dists []*Distribution
	query := `SELECT * FROM distributions WHERE investment_id = $1 ORDER BY distribution_date DESC`
	err := s.db.SelectContext(ctx, &dists, query, investmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list distributions: %w", err)
	}
	return dists, nil
}

// CreateDocumentInput represents input for creating a document
type CreateDocumentInput struct {
	InvestmentID  uuid.UUID
	DocumentType  DocumentType
	DocumentDate  *time.Time
	TaxYear       *int
	FileURL       string
	FileName      *string
	FileSizeBytes *int
	MimeType      *string
	UploadedBy    *uuid.UUID
}

// TODO: Migrate to Hasura GraphQL mutation:
//
//	mutation CreateDocument($object: alt_investment_documents_insert_input!) {
//	  insert_alt_investment_documents_one(object: $object) { ... }
//	}
//
// Note: K-1 tax forms, capital call notices, quarterly reports with Gemini AI extraction
func (s *service) CreateDocument(ctx context.Context, input CreateDocumentInput) (*AltInvestmentDocument, error) {
	doc := &AltInvestmentDocument{
		DocumentID:    uuid.New(),
		InvestmentID:  input.InvestmentID,
		DocumentType:  input.DocumentType,
		DocumentDate:  input.DocumentDate,
		TaxYear:       input.TaxYear,
		FileURL:       input.FileURL,
		FileName:      input.FileName,
		FileSizeBytes: input.FileSizeBytes,
		MimeType:      input.MimeType,
		ExtractedData: json.RawMessage("{}"),
		UploadedAt:    time.Now(),
		UploadedBy:    input.UploadedBy,
	}

	pending := ExtractPending
	doc.ExtractionStatus = &pending

	query := `
		INSERT INTO alt_investment_documents (
			document_id, investment_id, document_type, document_date, tax_year,
			file_url, file_name, file_size_bytes, mime_type, extracted_data,
			extraction_status, uploaded_at, uploaded_by
		) VALUES (
			:document_id, :investment_id, :document_type, :document_date, :tax_year,
			:file_url, :file_name, :file_size_bytes, :mime_type, :extracted_data,
			:extraction_status, :uploaded_at, :uploaded_by
		)
	`
	_, err := s.db.NamedExecContext(ctx, query, doc)
	if err != nil {
		return nil, fmt.Errorf("failed to create document: %w", err)
	}

	return doc, nil
}

// TODO: Migrate to Hasura GraphQL query:
//
//	query GetDocument($document_id: uuid!) {
//	  alt_investment_documents_by_pk(document_id: $document_id) { ... }
//	}
func (s *service) GetDocument(ctx context.Context, documentID uuid.UUID) (*AltInvestmentDocument, error) {
	var doc AltInvestmentDocument
	query := `SELECT * FROM alt_investment_documents WHERE document_id = $1`
	err := s.db.GetContext(ctx, &doc, query, documentID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("document not found: %s", documentID)
		}
		return nil, fmt.Errorf("failed to get document: %w", err)
	}
	return &doc, nil
}

// TODO: Migrate to Hasura GraphQL query:
//
//	query ListDocumentsByInvestment($investment_id: uuid!) {
//	  alt_investment_documents(where: {investment_id: {_eq: $investment_id}}, order_by: {document_date: desc}) { ... }
//	}
func (s *service) ListDocumentsByInvestment(ctx context.Context, investmentID uuid.UUID) ([]*AltInvestmentDocument, error) {
	var docs []*AltInvestmentDocument
	query := `SELECT * FROM alt_investment_documents WHERE investment_id = $1 ORDER BY document_date DESC`
	err := s.db.SelectContext(ctx, &docs, query, investmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list documents: %w", err)
	}
	return docs, nil
}

// TODO: Migrate to Hasura GraphQL mutation:
//
//	mutation UpdateDocumentExtraction($document_id: uuid!, $extracted_data: jsonb!, $confidence: numeric!, $status: String!) {
//	  update_alt_investment_documents_by_pk(pk_columns: {document_id: $document_id}, _set: {extracted_data: $extracted_data, extraction_confidence: $confidence, extraction_status: $status, processed_at: "now()"}) { ... }
//	}
func (s *service) UpdateDocumentExtraction(ctx context.Context, documentID uuid.UUID, extractedData json.RawMessage, confidence float64, status ExtractionStatus) error {
	query := `
		UPDATE alt_investment_documents
		SET extracted_data = $1, extraction_confidence = $2, extraction_status = $3, processed_at = $4
		WHERE document_id = $5
	`
	_, err := s.db.ExecContext(ctx, query, extractedData, confidence, status, time.Now(), documentID)
	if err != nil {
		return fmt.Errorf("failed to update document extraction: %w", err)
	}
	return nil
}
