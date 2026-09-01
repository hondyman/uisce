package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"

	"github.com/pgvector/pgvector-go"
)

type SemanticCache struct {
	DB               *sql.DB
	EmbeddingService *EmbeddingService
}

func NewSemanticCache(db *sql.DB, embedder *EmbeddingService) *SemanticCache {
	return &SemanticCache{
		DB:               db,
		EmbeddingService: embedder,
	}
}

// CheckCache looks for an identical intent. Distance < 0.02 is a highly confident match (>98% similarity).
func (c *SemanticCache) CheckCache(ctx context.Context, tenantID, prompt string) (*AIExplorerQueryDefinition, error) {
	if c.DB == nil || c.EmbeddingService == nil {
		return nil, nil
	}

	embedding, err := c.EmbeddingService.GenerateEmbedding(ctx, prompt)
	if err != nil || len(embedding) == 0 {
		return nil, err
	}

	query := `
		SELECT id, query_payload 
		FROM ai_semantic_cache 
		WHERE (tenant_id = $1 OR tenant_id = 'default')
		  AND expires_at > NOW()
		  AND (prompt_embedding <=> $2) < 0.02
		ORDER BY (prompt_embedding <=> $2) ASC 
		LIMIT 1
	`

	var cacheID string
	var payloadBytes []byte

	err = c.DB.QueryRowContext(ctx, query, tenantID, pgvector.NewVector(embedding)).Scan(&cacheID, &payloadBytes)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Normal cache miss
		}
		return nil, err
	}

	var cachedQuery AIExplorerQueryDefinition
	if err := json.Unmarshal(payloadBytes, &cachedQuery); err != nil {
		return nil, err
	}

	// Async hit counter update
	go func() {
		_, _ = c.DB.Exec("UPDATE ai_semantic_cache SET hits = hits + 1 WHERE id = $1", cacheID)
	}()

	return &cachedQuery, nil
}

// SetCache saves a successful query execution for future identical intents.
func (c *SemanticCache) SetCache(ctx context.Context, tenantID, prompt string, queryDef AIExplorerQueryDefinition) {
	if c.DB == nil || c.EmbeddingService == nil {
		return
	}

	embedding, err := c.EmbeddingService.GenerateEmbedding(ctx, prompt)
	if err != nil || len(embedding) == 0 {
		return
	}

	payloadBytes, err := json.Marshal(queryDef)
	if err != nil {
		return
	}

	insertSQL := `
		INSERT INTO ai_semantic_cache (tenant_id, prompt_embedding, query_payload)
		VALUES ($1, $2, $3)
	`
	_, err = c.DB.ExecContext(ctx, insertSQL, tenantID, pgvector.NewVector(embedding), payloadBytes)
	if err != nil {
		log.Printf("Semantic Cache Warning: %v", err)
	}
}
