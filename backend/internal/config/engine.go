package config

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"github.com/google/uuid"
)

// ConfigurationEngine manages tenant-specific configurations
type ConfigurationEngine struct {
	db *sql.DB
}

// NewConfigurationEngine creates a new ConfigurationEngine
func NewConfigurationEngine(db *sql.DB) *ConfigurationEngine {
	return &ConfigurationEngine{db: db}
}

// TenantConfig represents a generic configuration record
type TenantConfig struct {
	ConfigID   uuid.UUID              `json:"config_id"`
	TenantID   uuid.UUID              `json:"tenant_id"`
	ConfigType string                 `json:"config_type"`
	ConfigData map[string]interface{} `json:"config_data"`
	Version    int                    `json:"version"`
	Active     bool                   `json:"active"`
}

// GetConfig retrieves the active configuration of a specific type for a tenant
func (ce *ConfigurationEngine) GetConfig(ctx context.Context, tenantID uuid.UUID, configType string) (*TenantConfig, error) {
	var config TenantConfig
	var configDataJSON []byte

	err := ce.db.QueryRowContext(ctx, `
		SELECT config_id, tenant_id, config_type, config_data, version, active
		FROM tenant_configs
		WHERE tenant_id = $1 AND config_type = $2 AND active = true
		ORDER BY version DESC
		LIMIT 1
	`, tenantID, configType).Scan(
		&config.ConfigID,
		&config.TenantID,
		&config.ConfigType,
		&configDataJSON,
		&config.Version,
		&config.Active,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// Return nil if no config found, caller handles defaults
			return nil, nil
		}
		return nil, fmt.Errorf("failed to fetch config %s for tenant %s: %w", configType, tenantID, err)
	}

	if err := json.Unmarshal(configDataJSON, &config.ConfigData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config data: %w", err)
	}

	return &config, nil
}

// SaveConfig saves a new version of a configuration for a tenant
func (ce *ConfigurationEngine) SaveConfig(ctx context.Context, tenantID uuid.UUID, configType string, configData map[string]interface{}) error {
	configDataJSON, err := json.Marshal(configData)
	if err != nil {
		return fmt.Errorf("failed to marshal config data: %w", err)
	}

	tx, err := ce.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Deactivate current active config (if any)
	_, err = tx.ExecContext(ctx, `
		UPDATE tenant_configs 
		SET active = false 
		WHERE tenant_id = $1 AND config_type = $2 AND active = true
	`, tenantID, configType)
	if err != nil {
		return fmt.Errorf("failed to deactivate old config: %w", err)
	}

	// 2. Insert new config version
	_, err = tx.ExecContext(ctx, `
		INSERT INTO tenant_configs (tenant_id, config_type, config_data, version, active)
		SELECT $1, $2, $3::jsonb, COALESCE(MAX(version), 0) + 1, true
		FROM tenant_configs
		WHERE tenant_id = $1 AND config_type = $2
	`, tenantID, configType, string(configDataJSON))
	
	// Handle case where it's the first config (MAX(version) returns null, but COALESCE handles it, 
	// but we need to make sure the FROM clause returns a row even if empty. 
	// Actually, standard SQL might return null for MAX on empty set. 
	// A safer way is to query max version first or use a separate INSERT logic.
	// Let's use a simpler approach: Get max version first.
	
	// Re-do logic inside transaction for safety
	var maxVersion int
	err = tx.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(version), 0) FROM tenant_configs WHERE tenant_id = $1 AND config_type = $2
	`, tenantID, configType).Scan(&maxVersion)
	if err != nil {
		return fmt.Errorf("failed to get max version: %w", err)
	}

	newVersion := maxVersion + 1
	
	_, err = tx.ExecContext(ctx, `
		INSERT INTO tenant_configs (tenant_id, config_type, config_data, version, active)
		VALUES ($1, $2, $3::jsonb, $4, true)
	`, tenantID, configType, string(configDataJSON), newVersion)
	if err != nil {
		return fmt.Errorf("failed to insert new config: %w", err)
	}

	// 3. Log deployment
	_, err = tx.ExecContext(ctx, `
		INSERT INTO config_deployments (tenant_id, config_type, version, deployed_by, changelog)
		VALUES ($1, $2, $3, 'system', 'Configuration update')
	`, tenantID, configType, newVersion)
	if err != nil {
		log.Printf("Warning: failed to log deployment: %v", err)
		// Non-fatal
	}

	return tx.Commit()
}

// RAGConfig represents the specific configuration for RAG operations
type RAGConfig struct {
	EmbeddingModel   EmbeddingModelConfig   `json:"embedding_model"`
	ChunkingStrategy ChunkingStrategyConfig `json:"chunking_strategy"`
	RetrievalConfig  RetrievalConfig        `json:"retrieval_config"`
	HybridSearch     HybridSearchConfig     `json:"hybrid_search"`
}

type EmbeddingModelConfig struct {
	Provider   string `json:"provider"`
	Model      string `json:"model"`
	Dimensions int    `json:"dimensions"`
}

type ChunkingStrategyConfig struct {
	Method        string   `json:"method"`
	MaxChunkSize  int      `json:"max_chunk_size"`
	OverlapTokens int      `json:"overlap_tokens"`
	Separators    []string `json:"separators"`
}

type RetrievalConfig struct {
	TopK                int     `json:"top_k"`
	SimilarityThreshold float64 `json:"similarity_threshold"`
	Rerank              bool    `json:"rerank"`
}

type HybridSearchConfig struct {
	Enabled        bool    `json:"enabled"`
	SemanticWeight float64 `json:"semantic_weight"`
	KeywordWeight  float64 `json:"keyword_weight"`
}

// GetRAGConfig is a helper to get and parse RAG configuration
func (ce *ConfigurationEngine) GetRAGConfig(ctx context.Context, tenantID uuid.UUID) (*RAGConfig, error) {
	config, err := ce.GetConfig(ctx, tenantID, "rag")
	if err != nil {
		return nil, err
	}
	
	// Default configuration if none exists
	if config == nil {
		return &RAGConfig{
			EmbeddingModel: EmbeddingModelConfig{
				Provider:   "openai",
				Model:      "text-embedding-ada-002",
				Dimensions: 1536,
			},
			ChunkingStrategy: ChunkingStrategyConfig{
				Method:        "semantic",
				MaxChunkSize:  512,
				OverlapTokens: 50,
				Separators:    []string{"\n\n", "\n", ". ", " "},
			},
			RetrievalConfig: RetrievalConfig{
				TopK:                10,
				SimilarityThreshold: 0.7,
				Rerank:              false,
			},
			HybridSearch: HybridSearchConfig{
				Enabled:        true,
				SemanticWeight: 0.7,
				KeywordWeight:  0.3,
			},
		}, nil
	}

	// Convert map to struct
	configJSON, _ := json.Marshal(config.ConfigData)
	var ragConfig RAGConfig
	if err := json.Unmarshal(configJSON, &ragConfig); err != nil {
		return nil, fmt.Errorf("failed to parse RAG config: %w", err)
	}

	return &ragConfig, nil
}
