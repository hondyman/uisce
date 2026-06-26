package workflows

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/rag"
	"github.com/hondyman/semlayer/backend/internal/tenant"
)

// DocumentActivities holds the activities for document processing
type DocumentActivities struct {
	TenantManager    *tenant.TenantManager
	IngestionService *rag.IngestionService
	EmbeddingService rag.EmbeddingService
	ConfigService    *rag.ConfigService
}

// NewDocumentActivities creates a new DocumentActivities instance
func NewDocumentActivities(tm *tenant.TenantManager, is *rag.IngestionService, es rag.EmbeddingService, cs *rag.ConfigService) *DocumentActivities {
	return &DocumentActivities{
		TenantManager:    tm,
		IngestionService: is,
		EmbeddingService: es,
		ConfigService:    cs,
	}
}

// ExtractTextActivity extracts text from a document (mocked for now)
func (a *DocumentActivities) ExtractTextActivity(ctx context.Context, sourcePath string) (string, error) {
	// TODO: Implement actual text extraction (e.g., from PDF/Docx)
	// For now, return dummy text
	return fmt.Sprintf("Content extracted from %s. This is a sample document for RAG testing.", sourcePath), nil
}

// ChunkDocumentActivity splits text into chunks based on tenant config
func (a *DocumentActivities) ChunkDocumentActivity(ctx context.Context, tenantID uuid.UUID, documentType string, text string) ([]string, error) {
	// 1. Get chunking config
	config, err := a.ConfigService.GetDocumentTypeConfig(ctx, tenantID, documentType)
	if err != nil {
		return nil, fmt.Errorf("failed to get chunking config: %w", err)
	}

	// 2. Apply strategy
	// For now, we support "fixed" (simple split) and "semantic" (mocked)
	// Real implementation would use the MaxChunkSize and OverlapTokens from config
	
	var chunks []string
	if config.ChunkingStrategy.Method == "fixed" {
		// Simple fixed-size chunking (mocked by splitting by periods)
		rawChunks := strings.Split(text, ". ")
		for _, c := range rawChunks {
			trimmed := strings.TrimSpace(c)
			if len(trimmed) > 0 {
				chunks = append(chunks, trimmed)
			}
		}
	} else {
		// Default/Semantic: mock implementation
		// In reality, this would use a tokenizer and sliding window
		rawChunks := strings.Split(text, ". ")
		for _, c := range rawChunks {
			trimmed := strings.TrimSpace(c)
			if len(trimmed) > 0 {
				chunks = append(chunks, trimmed)
			}
		}
	}
	
	return chunks, nil
}

// GenerateEmbeddingsActivity generates embeddings for chunks
func (a *DocumentActivities) GenerateEmbeddingsActivity(ctx context.Context, chunks []string) ([][]float32, error) {
	return a.EmbeddingService.EmbedBatch(ctx, chunks)
}

// StoreChunksActivity stores the chunks and embeddings in the database
func (a *DocumentActivities) StoreChunksActivity(ctx context.Context, tenantID uuid.UUID, documentID uuid.UUID, chunks []string, embeddings [][]float32) error {
	if len(chunks) != len(embeddings) {
		return fmt.Errorf("chunk count %d does not match embedding count %d", len(chunks), len(embeddings))
	}

	// 1. Get tenant connection
	conn, err := a.TenantManager.GetTenantConnection(ctx, tenantID)
	if err != nil {
		return err
	}
	defer conn.Close()

	// 2. Prepare chunk objects
	var docChunks []rag.DocumentChunk
	for i, content := range chunks {
		docChunks = append(docChunks, rag.DocumentChunk{
			ChunkID:    uuid.New(),
			DocumentID: documentID,
			ChunkIndex: i,
			Content:    content,
			Embedding:  embeddings[i],
			TokenCount: len(content) / 4, // Rough estimate
		})
	}

	// 3. Store in DB
	return a.IngestionService.StoreChunks(ctx, conn, docChunks)
}
