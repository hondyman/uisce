package rag

import (
	"time"
)

// Chunk represents a single segment of a document
type Chunk struct {
	ChunkID          string         `json:"chunk_id"`
	DocumentID       string         `json:"document_id"`
	Index            int            `json:"index"`
	Text             string         `json:"text"`
	TokenCount       int            `json:"token_count"`
	Metadata         map[string]any `json:"metadata"`
	SourceSnapshotID string         `json:"source_snapshot_id"`
	Embedding        []float32      `json:"embedding,omitempty"` // Omitted in JSON if empty
	CreatedAt        time.Time      `json:"created_at"`
}

// ChunkHit represents a search result with similarity score
type ChunkHit struct {
	ChunkID          string         `json:"chunk_id"`
	DocumentID       string         `json:"document_id"`
	ChunkIndex       int            `json:"chunk_index"`
	Text             string         `json:"text"`
	Metadata         map[string]any `json:"metadata"`
	SourceSnapshotID string         `json:"source_snapshot_id"`
	Similarity       float64        `json:"similarity"`
}

// IngestionRequest represents a request to ingest a document
type IngestionRequest struct {
	TenantID   string         `json:"tenant_id"`
	ClientID   string         `json:"client_id"`
	DocumentID string         `json:"document_id"`
	Text       string         `json:"text"`
	Metadata   map[string]any `json:"metadata"`
}
