package rag

import (
	"context"
	"fmt"
)

// EmbeddingService defines the interface for generating vector embeddings
type EmbeddingService interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
}

// OpenAIEmbedder implements EmbeddingService using OpenAI API
type OpenAIEmbedder struct {
	APIKey string
	Model  string
}

// NewOpenAIEmbedder creates a new OpenAI embedder
func NewOpenAIEmbedder(apiKey, model string) *OpenAIEmbedder {
	return &OpenAIEmbedder{
		APIKey: apiKey,
		Model:  model,
	}
}

// Embed generates an embedding for a single text string
func (oe *OpenAIEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	// TODO: Implement actual OpenAI API call
	// For now, return a dummy embedding of size 1536 (standard for ada-002)
	// This allows the system to compile and run without an active API key for testing
	
	dims := 1536
	embedding := make([]float32, dims)
	
	// Simple hash-based embedding generation for testing
	// Use position-dependent hash to ensure distinctness
	var seed int
	for i, c := range text {
		seed = seed*31 + int(c) + i
	}
	
	for i := 0; i < dims; i++ {
		// Create a pattern that varies based on the seed and index
		val := float32((seed + i) % 100) / 100.0
		embedding[i] = val
	}
	
	return embedding, nil
}

// EmbedBatch generates embeddings for a batch of texts
func (oe *OpenAIEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		emb, err := oe.Embed(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("failed to embed text at index %d: %w", i, err)
		}
		embeddings[i] = emb
	}
	return embeddings, nil
}
