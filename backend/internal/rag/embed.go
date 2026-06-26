package rag

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
)

// Embedder defines the interface for generating vector embeddings
type Embedder interface {
	Embed(ctx context.Context, text string, modelID string) ([]float32, error)
}

// DemoEmbedder generates random embeddings for testing purposes
type DemoEmbedder struct {
	Dim int
}

// Embed generates a random vector of the specified dimension
func (d DemoEmbedder) Embed(ctx context.Context, text string, modelID string) ([]float32, error) {
	if d.Dim <= 0 {
		d.Dim = 1536 // Default to OpenAI ada-002 dimension
	}

	b := make([]byte, d.Dim*4)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}

	vec := make([]float32, d.Dim)
	for i := 0; i < d.Dim; i++ {
		// Convert bytes to float32 in range [0, 1]
		val := binary.LittleEndian.Uint32(b[i*4:])
		vec[i] = float32(val) / 4294967295.0
	}

	return vec, nil
}

// IngestChunks orchestrates the ingestion process: Sanitize -> Embed -> Persist
func IngestChunks(
	ctx context.Context,
	store *Store,
	tenantID string,
	clientID string,
	modelID string,
	snapshotID string,
	chunks []Chunk,
	sanitizer *SanitizationClient,
	embedder Embedder,
) error {
	
	processedChunks := make([]Chunk, len(chunks))

	for i, chunk := range chunks {
		// 1. Sanitize Text
		sanResp, err := sanitizer.SanitizeText(SanitizeRequest{
			TenantID:  tenantID,
			ClientID:  clientID,
			Text:      chunk.Text,
			RequestID: chunk.ChunkID,
		})
		if err != nil {
			return fmt.Errorf("sanitization failed for chunk %s: %w", chunk.ChunkID, err)
		}

		// 2. Generate Embedding from Sanitized Text
		vec, err := embedder.Embed(ctx, sanResp.SanitizedText, modelID)
		if err != nil {
			return fmt.Errorf("embedding failed for chunk %s: %w", chunk.ChunkID, err)
		}

		// 3. Update Chunk with Metadata
		// Create a copy to avoid mutating the original input slice if that matters
		c := chunk
		c.Embedding = vec
		c.SourceSnapshotID = snapshotID
		
		if c.Metadata == nil {
			c.Metadata = make(map[string]any)
		}
		c.Metadata["pii_map_id"] = sanResp.PIIMapID
		c.Metadata["model_id"] = modelID
		c.Metadata["sanitized"] = true

		processedChunks[i] = c
	}

	// 4. Persist to Store
	if err := store.UpsertChunks(ctx, tenantID, processedChunks); err != nil {
		return fmt.Errorf("failed to upsert chunks: %w", err)
	}

	return nil
}
