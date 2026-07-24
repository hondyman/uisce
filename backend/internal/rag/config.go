package rag

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// RAGConfig represents the global RAG configuration for a tenant
type RAGConfig struct {
	ConfigID        uuid.UUID       `json:"config_id"`
	TenantID        uuid.UUID       `json:"tenant_id"`
	EmbeddingModel  EmbeddingConfig `json:"embedding_model"`
	RetrievalConfig RetrievalConfig `json:"retrieval_config"`
	HybridSearch    HybridConfig    `json:"hybrid_search"`
}

type EmbeddingConfig struct {
	Provider   string `json:"provider"`
	Model      string `json:"model"`
	Dimensions int    `json:"dimensions"`
}

type RetrievalConfig struct {
	TopK                int     `json:"top_k"`
	SimilarityThreshold float64 `json:"similarity_threshold"`
	Rerank              bool    `json:"rerank"`
}

type HybridConfig struct {
	Enabled        bool    `json:"enabled"`
	SemanticWeight float64 `json:"semantic_weight"`
	KeywordWeight  float64 `json:"keyword_weight"`
}

// DocumentTypeConfig represents configuration for a specific document type
type DocumentTypeConfig struct {
	ConfigID         uuid.UUID        `json:"config_id"`
	TenantID         uuid.UUID        `json:"tenant_id"`
	TypeCode         string           `json:"type_code"`
	DisplayName      string           `json:"display_name"`
	ChunkingStrategy ChunkingStrategy `json:"chunking_strategy"`
	ExtractionRules  json.RawMessage  `json:"extraction_rules"`
}

type ChunkingStrategy struct {
	Method        string `json:"method"` // "semantic", "fixed", "paragraph"
	MaxChunkSize  int    `json:"max_chunk_size"`
	OverlapTokens int    `json:"overlap_tokens"`
}

// ConfigService manages RAG configurations
type ConfigService struct {
	db *sql.DB
}

func NewConfigService(db *sql.DB) *ConfigService {
	return &ConfigService{db: db}
}

// GetRAGConfig retrieves the RAG configuration for a tenant
func (s *ConfigService) GetRAGConfig(ctx context.Context, tenantID uuid.UUID) (*RAGConfig, error) {
	var config RAGConfig
	var embeddingJSON, retrievalJSON, hybridJSON []byte

	query := `
		SELECT config_id, tenant_id, embedding_model, retrieval_config, hybrid_search
		FROM rag_configs
		WHERE tenant_id = $1
	`

	err := s.db.QueryRowContext(ctx, query, tenantID).Scan(
		&config.ConfigID,
		&config.TenantID,
		&embeddingJSON,
		&retrievalJSON,
		&hybridJSON,
	)

	if err == sql.ErrNoRows {
		// Return default config
		return s.getDefaultRAGConfig(tenantID), nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch rag config: %w", err)
	}

	if err := json.Unmarshal(embeddingJSON, &config.EmbeddingModel); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(retrievalJSON, &config.RetrievalConfig); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(hybridJSON, &config.HybridSearch); err != nil {
		return nil, err
	}

	return &config, nil
}

// GetDocumentTypeConfig retrieves configuration for a specific document type
func (s *ConfigService) GetDocumentTypeConfig(ctx context.Context, tenantID uuid.UUID, typeCode string) (*DocumentTypeConfig, error) {
	var config DocumentTypeConfig
	var chunkingJSON []byte

	query := `
		SELECT config_id, tenant_id, type_code, display_name, chunking_strategy, extraction_rules
		FROM document_type_configs
		WHERE tenant_id = $1 AND type_code = $2
	`

	err := s.db.QueryRowContext(ctx, query, tenantID, typeCode).Scan(
		&config.ConfigID,
		&config.TenantID,
		&config.TypeCode,
		&config.DisplayName,
		&chunkingJSON,
		&config.ExtractionRules,
	)

	if err == sql.ErrNoRows {
		// Return default config
		return s.getDefaultDocumentTypeConfig(tenantID, typeCode), nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch document type config: %w", err)
	}

	if err := json.Unmarshal(chunkingJSON, &config.ChunkingStrategy); err != nil {
		return nil, err
	}

	return &config, nil
}

func (s *ConfigService) getDefaultRAGConfig(tenantID uuid.UUID) *RAGConfig {
	return &RAGConfig{
		TenantID: tenantID,
		EmbeddingModel: EmbeddingConfig{
			Provider:   "openai",
			Model:      "text-embedding-ada-002",
			Dimensions: 1536,
		},
		RetrievalConfig: RetrievalConfig{
			TopK:                10,
			SimilarityThreshold: 0.7,
			Rerank:              false,
		},
		HybridSearch: HybridConfig{
			Enabled:        true,
			SemanticWeight: 0.7,
			KeywordWeight:  0.3,
		},
	}
}

func (s *ConfigService) getDefaultDocumentTypeConfig(tenantID uuid.UUID, typeCode string) *DocumentTypeConfig {
	return &DocumentTypeConfig{
		TenantID:    tenantID,
		TypeCode:    typeCode,
		DisplayName: typeCode,
		ChunkingStrategy: ChunkingStrategy{
			Method:        "semantic",
			MaxChunkSize:  512,
			OverlapTokens: 50,
		},
	}
}
