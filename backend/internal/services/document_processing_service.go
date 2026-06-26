package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/hondyman/semlayer/backend/internal/types"

	"github.com/google/uuid"
)

// DocumentProcessingService handles AI-powered document processing
// This is a stub implementation - full Gemini integration to be added
type DocumentProcessingService struct {
	db *sql.DB
	// geminiClient will be added when integrating Gemini API
}

// NewDocumentProcessingService creates a new document processing service
func NewDocumentProcessingService(db *sql.DB) *DocumentProcessingService {
	return &DocumentProcessingService{
		db: db,
	}
}

// UploadDocument uploads and stores a document for processing
func (s *DocumentProcessingService) UploadDocument(ctx context.Context, doc *types.AlternativeInvestmentDocument) error {
	query := `
		INSERT INTO alternative_investment_documents (
			investment_id, document_type, file_name, file_path,
			file_size_bytes, mime_type, uploaded_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, uploaded_at, processing_status
	`

	return s.db.QueryRowContext(ctx, query,
		doc.InvestmentID, doc.DocumentType, doc.FileName, doc.FilePath,
		doc.FileSizeBytes, doc.MimeType, doc.UploadedBy,
	).Scan(&doc.ID, &doc.UploadedAt, &doc.ProcessingStatus)
}

// ProcessDocument processes a document using AI
func (s *DocumentProcessingService) ProcessDocument(ctx context.Context, documentID uuid.UUID) error {
	// Update status to processing
	_, err := s.db.ExecContext(ctx, `
		UPDATE alternative_investment_documents
		SET processing_status = 'PROCESSING'
		WHERE id = $1
	`, documentID)
	if err != nil {
		return err
	}

	// TODO: Implement Gemini API integration
	// For now, mark as needs review
	_, err = s.db.ExecContext(ctx, `
		UPDATE alternative_investment_documents
		SET processing_status = 'NEEDS_REVIEW',
			requires_review = TRUE,
			processed_at = NOW()
		WHERE id = $1
	`, documentID)

	return err
}

// GetDocument retrieves a document by ID
func (s *DocumentProcessingService) GetDocument(ctx context.Context, documentID uuid.UUID) (*types.AlternativeInvestmentDocument, error) {
	query := `
		SELECT 
			id, investment_id, document_type, file_name, file_path,
			file_size_bytes, mime_type, processing_status, processed_at,
			processing_error, extracted_data, confidence_scores,
			requires_review, reviewed_by, reviewed_at, review_notes,
			review_status, uploaded_at, uploaded_by
		FROM alternative_investment_documents
		WHERE id = $1
	`

	doc := &types.AlternativeInvestmentDocument{}
	var extractedDataJSON, confidenceScoresJSON []byte

	err := s.db.QueryRowContext(ctx, query, documentID).Scan(
		&doc.ID, &doc.InvestmentID, &doc.DocumentType, &doc.FileName, &doc.FilePath,
		&doc.FileSizeBytes, &doc.MimeType, &doc.ProcessingStatus, &doc.ProcessedAt,
		&doc.ProcessingError, &extractedDataJSON, &confidenceScoresJSON,
		&doc.RequiresReview, &doc.ReviewedBy, &doc.ReviewedAt, &doc.ReviewNotes,
		&doc.ReviewStatus, &doc.UploadedAt, &doc.UploadedBy,
	)
	if err != nil {
		return nil, err
	}

	// Parse JSONB fields
	if len(extractedDataJSON) > 0 {
		if err := json.Unmarshal(extractedDataJSON, &doc.ExtractedData); err != nil {
			return nil, err
		}
	}
	if len(confidenceScoresJSON) > 0 {
		if err := json.Unmarshal(confidenceScoresJSON, &doc.ConfidenceScores); err != nil {
			return nil, err
		}
	}

	return doc, nil
}

// ListDocumentsByInvestment lists all documents for an investment
func (s *DocumentProcessingService) ListDocumentsByInvestment(ctx context.Context, investmentID uuid.UUID) ([]*types.AlternativeInvestmentDocument, error) {
	query := `
		SELECT 
			id, investment_id, document_type, file_name, file_path,
			file_size_bytes, mime_type, processing_status, processed_at,
			requires_review, review_status, uploaded_at
		FROM alternative_investment_documents
		WHERE investment_id = $1
		ORDER BY uploaded_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, investmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var documents []*types.AlternativeInvestmentDocument
	for rows.Next() {
		doc := &types.AlternativeInvestmentDocument{}
		err := rows.Scan(
			&doc.ID, &doc.InvestmentID, &doc.DocumentType, &doc.FileName, &doc.FilePath,
			&doc.FileSizeBytes, &doc.MimeType, &doc.ProcessingStatus, &doc.ProcessedAt,
			&doc.RequiresReview, &doc.ReviewStatus, &doc.UploadedAt,
		)
		if err != nil {
			return nil, err
		}
		documents = append(documents, doc)
	}

	return documents, rows.Err()
}

// ApproveDocument approves a reviewed document
func (s *DocumentProcessingService) ApproveDocument(ctx context.Context, documentID, reviewerID uuid.UUID, notes string) error {
	query := `
		UPDATE alternative_investment_documents
		SET 
			review_status = 'APPROVED',
			reviewed_by = $1,
			reviewed_at = NOW(),
			review_notes = $2,
			processing_status = 'REVIEWED'
		WHERE id = $3
	`

	result, err := s.db.ExecContext(ctx, query, reviewerID, notes, documentID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("document not found")
	}

	return nil
}

// RejectDocument rejects a reviewed document
func (s *DocumentProcessingService) RejectDocument(ctx context.Context, documentID, reviewerID uuid.UUID, notes string) error {
	query := `
		UPDATE alternative_investment_documents
		SET 
			review_status = 'REJECTED',
			reviewed_by = $1,
			reviewed_at = NOW(),
			review_notes = $2,
			processing_status = 'FAILED'
		WHERE id = $3
	`

	result, err := s.db.ExecContext(ctx, query, reviewerID, notes, documentID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("document not found")
	}

	return nil
}
