package onboarding

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Additional interface methods implementation

// CreateSession creates a new session (wrapper around StartSession)
func (s *service) CreateSession(ctx context.Context, tenantID uuid.UUID, email string) (*OnboardingSession, error) {
	metadata := SessionMetadata{
		IPAddress: "unknown",
		UserAgent: "api",
	}
	return s.StartSession(ctx, email, metadata)
}

// GetSessionByToken retrieves a session by resume token
func (s *service) GetSessionByToken(ctx context.Context, resumeToken uuid.UUID) (*OnboardingSession, error) {
	return s.getSessionByTokenRecord(ctx, resumeToken)
}

// UpdateSession updates session with arbitrary fields
func (s *service) UpdateSession(ctx context.Context, sessionID uuid.UUID, updates map[string]interface{}) error {
	// For simplicity, update the step_data field with the updates
	updatesJSON, err := json.Marshal(updates)
	if err != nil {
		return fmt.Errorf("failed to marshal updates: %w", err)
	}
	return s.updateSessionRecord(ctx, sessionID, updatesJSON)
}

// CompleteOnboarding marks a session as complete (wrapper around CompleteSession)
func (s *service) CompleteOnboarding(ctx context.Context, sessionID uuid.UUID) error {
	return s.CompleteSession(ctx, sessionID)
}

// GetDocuments retrieves all documents for a session
func (s *service) GetDocuments(ctx context.Context, sessionID uuid.UUID) ([]Document, error) {
	return s.getDocumentsRecords(ctx, sessionID)
}

// Helper method to get documents
func (s *service) getDocumentsRecords(ctx context.Context, sessionID uuid.UUID) ([]Document, error) {
	// TODO: Replace SQL with Hasura GraphQL query:
	// query GetSessionDocuments($sessionId: uuid!) {
	//   uploaded_documents(where: {session_id: {_eq: $sessionId}}, order_by: {uploaded_at: desc}) {
	//     document_id session_id client_id document_type
	//     document_name file_size_bytes mime_type storage_path
	//     document_status uploaded_at
	//   }
	// }
	// Use: http://localhost:8080/v1/graphql with header X-Hasura-Admin-Secret: newadminsecretkey
	var docs []Document
	query := `
		SELECT document_id, session_id, client_id, document_type, 
		       document_name, file_size_bytes, mime_type, storage_path,
		       document_status, uploaded_at
		FROM uploaded_documents
		WHERE session_id = $1
		ORDER BY uploaded_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var doc Document
		err := rows.Scan(
			&doc.DocumentID,
			&doc.SessionID,
			&doc.ClientID,
			&doc.DocumentType,
			&doc.DocumentName,
			&doc.FileSizeBytes,
			&doc.MimeType,
			&doc.StoragePath,
			&doc.DocumentStatus,
			&doc.UploadedAt,
		)
		if err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}

	return docs, rows.Err()
}

// Document represents an uploaded document (matches handler expectations)
type Document struct {
	DocumentID     uuid.UUID `db:"document_id" json:"document_id"`
	SessionID      uuid.UUID `db:"session_id" json:"session_id"`
	ClientID       uuid.UUID `db:"client_id" json:"client_id"`
	DocumentType   string    `db:"document_type" json:"document_type"`
	DocumentName   string    `db:"document_name" json:"document_name"`
	FileSizeBytes  int64     `db:"file_size_bytes" json:"file_size_bytes"`
	MimeType       string    `db:"mime_type" json:"mime_type"`
	StoragePath    string    `db:"storage_path" json:"storage_path"`
	DocumentStatus string    `db:"document_status" json:"document_status"`
	UploadedAt     time.Time `db:"uploaded_at" json:"uploaded_at"`
}
