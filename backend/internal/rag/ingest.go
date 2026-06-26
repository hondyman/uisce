package rag

import (
	"context"
	"database/sql"
	"fmt"

	"time"

	"github.com/google/uuid"

	"github.com/pgvector/pgvector-go"
)

// Document represents a file ingested into the system
type Document struct {
	DocumentID   uuid.UUID              `json:"document_id"`
	SourcePath   string                 `json:"source_path"`
	DocumentType string                 `json:"document_type"`
	Title        string                 `json:"title"`
	UploadDate   time.Time              `json:"upload_date"`
	FileHash     string                 `json:"file_hash"`
	Metadata     map[string]interface{} `json:"metadata"`
	Status       string                 `json:"status"`
}

// DocumentChunk represents a segment of a document
type DocumentChunk struct {
	ChunkID    uuid.UUID              `json:"chunk_id"`
	DocumentID uuid.UUID              `json:"document_id"`
	ChunkIndex int                    `json:"chunk_index"`
	Content    string                 `json:"content"`
	Embedding  []float32              `json:"embedding"`
	TokenCount int                    `json:"token_count"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// IngestionService handles the storage of documents and chunks
type IngestionService struct {
	// Connection is injected per-request (scoped to tenant)
	// In a real service, this might be a getter that takes a context
}

// NewIngestionService creates a new IngestionService
func NewIngestionService() *IngestionService {
	return &IngestionService{}
}

// CreateDocumentRecord creates the initial document record in the database
func (is *IngestionService) CreateDocumentRecord(ctx context.Context, conn *sql.Conn, doc *Document) error {
	_, err := conn.ExecContext(ctx, `
		INSERT INTO documents (document_id, source_path, document_type, title, file_hash, metadata, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (file_hash) DO UPDATE 
		SET updated_at = NOW(), status = 'processing'
		RETURNING document_id
	`, doc.DocumentID, doc.SourcePath, doc.DocumentType, doc.Title, doc.FileHash, "{}", "processing") // simplified metadata handling
	
	if err != nil {
		return fmt.Errorf("failed to create document record: %w", err)
	}
	return nil
}

// StoreChunks batch inserts document chunks with embeddings
func (is *IngestionService) StoreChunks(ctx context.Context, conn *sql.Conn, chunks []DocumentChunk) error {
	if len(chunks) == 0 {
		return nil
	}

	// Prepare transaction
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete existing chunks for this document (idempotency)
	// Assuming all chunks belong to the same document for this batch
	docID := chunks[0].DocumentID
	_, err = tx.ExecContext(ctx, `DELETE FROM document_chunks WHERE document_id = $1`, docID)
	if err != nil {
		return fmt.Errorf("failed to clear existing chunks: %w", err)
	}

	// Use standard INSERT for compatibility with pgx/v5 stdlib
	query := `
		INSERT INTO document_chunks (chunk_id, document_id, chunk_index, content, embedding, token_count, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	for _, chunk := range chunks {
		// Convert []float32 to pgvector.Vector
		vec := pgvector.NewVector(chunk.Embedding)
		
		_, err = tx.ExecContext(ctx, query, chunk.ChunkID, chunk.DocumentID, chunk.ChunkIndex, chunk.Content, vec, chunk.TokenCount, "{}")
		if err != nil {
			return fmt.Errorf("failed to insert chunk: %w", err)
		}
	}

	// Update document status
	_, err = tx.ExecContext(ctx, `UPDATE documents SET status = 'ready' WHERE document_id = $1`, docID)
	if err != nil {
		return fmt.Errorf("failed to update document status: %w", err)
	}

	return tx.Commit()
}
