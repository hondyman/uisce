package rag

import (
	"context"
	"database/sql"
	"fmt"


	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
)

// SearchResult represents a single hit from the search engine
type SearchResult struct {
	ChunkID    uuid.UUID `json:"chunk_id"`
	DocumentID uuid.UUID `json:"document_id"`
	Content    string    `json:"content"`
	Score      float64   `json:"score"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// SearchRequest defines the parameters for a search operation
type SearchRequest struct {
	Query           string    `json:"query"`
	QueryEmbedding  []float32 `json:"-"` // Populated by service
	Limit           int       `json:"limit"`
	MinScore        float64   `json:"min_score"`
	SemanticWeight  float64   `json:"semantic_weight"`
	KeywordWeight   float64   `json:"keyword_weight"`
}

// SearchService handles hybrid search operations
type SearchService struct {
	embedder EmbeddingService
}

// NewSearchService creates a new SearchService
func NewSearchService(embedder EmbeddingService) *SearchService {
	return &SearchService{embedder: embedder}
}

// HybridSearch performs a combination of vector similarity and keyword search
func (ss *SearchService) HybridSearch(ctx context.Context, conn *sql.Conn, req SearchRequest) ([]SearchResult, error) {
	// 1. Generate embedding for the query
	embedding, err := ss.embedder.Embed(ctx, req.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}
	req.QueryEmbedding = embedding

	// 2. Execute Hybrid Search Query
	// This query combines cosine similarity (vector) with ts_rank (keyword)
	// using Reciprocal Rank Fusion (RRF) or a weighted sum.
	// For simplicity, we'll use a weighted sum here.
	
	querySQL := `
		WITH vector_search AS (
			SELECT chunk_id, document_id, content, metadata,
				   1 - (embedding <=> $1::public.vector) AS vector_score
			FROM document_chunks
			ORDER BY embedding <=> $1::public.vector
			LIMIT $2 * 2
		),
		keyword_search AS (
			SELECT chunk_id, document_id, content, metadata,
				   ts_rank_cd(to_tsvector('english', content), plainto_tsquery('english', $3)) AS keyword_score
			FROM document_chunks
			WHERE to_tsvector('english', content) @@ plainto_tsquery('english', $3)
			LIMIT $2 * 2
		)
		SELECT 
			COALESCE(v.chunk_id, k.chunk_id) AS chunk_id,
			COALESCE(v.document_id, k.document_id) AS document_id,
			COALESCE(v.content, k.content) AS content,
			(COALESCE(v.vector_score, 0) * $4 + COALESCE(k.keyword_score, 0) * $5) AS final_score
		FROM vector_search v
		FULL OUTER JOIN keyword_search k ON v.chunk_id = k.chunk_id
		ORDER BY final_score DESC
		LIMIT $2
	`

	rows, err := conn.QueryContext(ctx, querySQL, 
		pgvector.NewVector(req.QueryEmbedding), 
		req.Limit, 
		req.Query, 
		req.SemanticWeight, 
		req.KeywordWeight,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search query: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.ChunkID, &r.DocumentID, &r.Content, &r.Score); err != nil {
			return nil, fmt.Errorf("failed to scan search result: %w", err)
		}
		// Filter by min score
		if r.Score >= req.MinScore {
			results = append(results, r)
		}
	}

	return results, nil
}
