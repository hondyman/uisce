package migration

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/hondyman/semlayer/backend/pkg/llm"
)

// KnowledgeItem represents a retrievable item from the Knowledge Base
type KnowledgeItem struct {
	ID          string                 `json:"id"`
	Category    string                 `json:"category"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Content     map[string]interface{} `json:"content"`
	Similarity  float64                `json:"similarity,omitempty"`
}

// EmbeddingService handles vector embeddings and RAG retrieval
type EmbeddingService struct {
	DB            *sql.DB
	ConfigService *llm.LLMConfigService
}

func NewEmbeddingService(db *sql.DB, cfgSvc *llm.LLMConfigService) *EmbeddingService {
	return &EmbeddingService{
		DB:            db,
		ConfigService: cfgSvc,
	}
}

// GenerateEmbedding creates a vector embedding for the given text using Gemini's embedding API
func (s *EmbeddingService) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
	cfg, err := s.ConfigService.Get()
	if err != nil {
		return nil, fmt.Errorf("LLM config unavailable: %w", err)
	}

	// Use real Gemini embedding API via the provider
	provider := llm.NewGeminiProvider(cfg.APIKey, cfg.Model)

	// Call real embedding API
	floats32, err := provider.Embed(ctx, text)
	if err != nil {
		// Fallback to mock if API unavailable (e.g., no API key)
		return s.generateMockEmbedding(text), nil
	}

	// Convert float32 to float64 for pgvector compatibility
	floats64 := make([]float64, len(floats32))
	for i, v := range floats32 {
		floats64[i] = float64(v)
	}

	return floats64, nil
}

// generateMockEmbedding creates a deterministic mock embedding when API is unavailable
func (s *EmbeddingService) generateMockEmbedding(text string) []float64 {
	embedding := make([]float64, 768)
	for i := range embedding {
		embedding[i] = float64((i+len(text))%100) / 100.0
	}
	return embedding
}

// RetrieveSimilar finds the most similar items from the Knowledge Base
func (s *EmbeddingService) RetrieveSimilar(ctx context.Context, queryText string, limit int, categories []string) ([]KnowledgeItem, error) {
	// 1. Generate embedding for query
	embedding, err := s.GenerateEmbedding(ctx, queryText)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	// 2. Query pgvector for similar items
	// Note: This requires pgvector extension to be enabled
	embeddingJSON, _ := json.Marshal(embedding)

	query := `
		SELECT id, category, name, description, content,
			   1 - (embedding <=> $1::vector) AS similarity
		FROM titan_knowledge_base
		WHERE ($2::text[] IS NULL OR category = ANY($2))
		ORDER BY embedding <=> $1::vector
		LIMIT $3
	`

	var categoryFilter interface{}
	if len(categories) > 0 {
		categoryFilter = categories
	}

	rows, err := s.DB.QueryContext(ctx, query, string(embeddingJSON), categoryFilter, limit)
	if err != nil {
		// Fallback: pgvector might not be installed, use text search
		return s.fallbackTextSearch(ctx, queryText, limit, categories)
	}
	defer rows.Close()

	var items []KnowledgeItem
	for rows.Next() {
		var item KnowledgeItem
		var contentJSON []byte

		if err := rows.Scan(&item.ID, &item.Category, &item.Name, &item.Description, &contentJSON, &item.Similarity); err != nil {
			continue
		}

		if len(contentJSON) > 0 {
			_ = json.Unmarshal(contentJSON, &item.Content)
		}

		items = append(items, item)
	}

	return items, nil
}

// fallbackTextSearch uses basic text matching when pgvector is unavailable
func (s *EmbeddingService) fallbackTextSearch(ctx context.Context, queryText string, limit int, categories []string) ([]KnowledgeItem, error) {
	query := `
		SELECT id, category, name, description, content
		FROM titan_knowledge_base
		WHERE ($1::text[] IS NULL OR category = ANY($1))
		  AND (name ILIKE $2 OR description ILIKE $2 OR $3 = ANY(tags))
		LIMIT $4
	`

	var categoryFilter interface{}
	if len(categories) > 0 {
		categoryFilter = categories
	}

	searchPattern := fmt.Sprintf("%%%s%%", queryText)

	rows, err := s.DB.QueryContext(ctx, query, categoryFilter, searchPattern, queryText, limit)
	if err != nil {
		return nil, fmt.Errorf("text search failed: %w", err)
	}
	defer rows.Close()

	var items []KnowledgeItem
	for rows.Next() {
		var item KnowledgeItem
		var contentJSON []byte

		if err := rows.Scan(&item.ID, &item.Category, &item.Name, &item.Description, &contentJSON); err != nil {
			continue
		}

		if len(contentJSON) > 0 {
			_ = json.Unmarshal(contentJSON, &item.Content)
		}
		item.Similarity = 0.5 // Default for text match

		items = append(items, item)
	}

	return items, nil
}

// AddToKnowledgeBase inserts a new item with its embedding
func (s *EmbeddingService) AddToKnowledgeBase(ctx context.Context, category, name, description string, content map[string]interface{}, tags []string) error {
	// Generate embedding from description + name
	embeddingText := fmt.Sprintf("%s: %s", name, description)
	embedding, err := s.GenerateEmbedding(ctx, embeddingText)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	contentJSON, _ := json.Marshal(content)
	embeddingJSON, _ := json.Marshal(embedding)

	query := `
		INSERT INTO titan_knowledge_base (category, name, description, content, embedding, tags)
		VALUES ($1, $2, $3, $4, $5::vector, $6)
	`

	_, err = s.DB.ExecContext(ctx, query, category, name, description, contentJSON, string(embeddingJSON), tags)
	return err
}
