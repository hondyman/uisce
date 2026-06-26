package rag_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/rag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetadataFirstRAG(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	configService := rag.NewConfigService(db)
	tenantID := uuid.New()

	t.Run("GetRAGConfig_ReturnsConfig", func(t *testing.T) {
		expectedConfig := rag.RAGConfig{
			ConfigID: uuid.New(),
			TenantID: tenantID,
			EmbeddingModel: rag.EmbeddingConfig{
				Provider:   "openai",
				Model:      "text-embedding-3-small",
				Dimensions: 1536,
			},
			RetrievalConfig: rag.RetrievalConfig{
				TopK:                20,
				SimilarityThreshold: 0.8,
				Rerank:              true,
			},
			HybridSearch: rag.HybridConfig{
				Enabled:        true,
				SemanticWeight: 0.6,
				KeywordWeight:  0.4,
			},
		}

		embeddingJSON, _ := json.Marshal(expectedConfig.EmbeddingModel)
		retrievalJSON, _ := json.Marshal(expectedConfig.RetrievalConfig)
		hybridJSON, _ := json.Marshal(expectedConfig.HybridSearch)

		rows := sqlmock.NewRows([]string{"config_id", "tenant_id", "embedding_model", "retrieval_config", "hybrid_search"}).
			AddRow(expectedConfig.ConfigID, expectedConfig.TenantID, embeddingJSON, retrievalJSON, hybridJSON)

		mock.ExpectQuery("SELECT config_id, tenant_id, embedding_model, retrieval_config, hybrid_search FROM rag_configs").
			WithArgs(tenantID).
			WillReturnRows(rows)

		config, err := configService.GetRAGConfig(context.Background(), tenantID)
		require.NoError(t, err)
		assert.Equal(t, expectedConfig.EmbeddingModel.Model, config.EmbeddingModel.Model)
		assert.Equal(t, 0.8, config.RetrievalConfig.SimilarityThreshold)
	})

	t.Run("GetDocumentTypeConfig_ReturnsConfig", func(t *testing.T) {
		typeCode := "contract"
		expectedConfig := rag.DocumentTypeConfig{
			ConfigID:    uuid.New(),
			TenantID:    tenantID,
			TypeCode:    typeCode,
			DisplayName: "Legal Contract",
			ChunkingStrategy: rag.ChunkingStrategy{
				Method:        "semantic",
				MaxChunkSize:  1000,
				OverlapTokens: 100,
			},
			ExtractionRules: json.RawMessage(`{}`),
		}

		chunkingJSON, _ := json.Marshal(expectedConfig.ChunkingStrategy)

		rows := sqlmock.NewRows([]string{"config_id", "tenant_id", "type_code", "display_name", "chunking_strategy", "extraction_rules"}).
			AddRow(expectedConfig.ConfigID, expectedConfig.TenantID, expectedConfig.TypeCode, expectedConfig.DisplayName, chunkingJSON, expectedConfig.ExtractionRules)

		mock.ExpectQuery("SELECT config_id, tenant_id, type_code, display_name, chunking_strategy, extraction_rules FROM document_type_configs").
			WithArgs(tenantID, typeCode).
			WillReturnRows(rows)

		config, err := configService.GetDocumentTypeConfig(context.Background(), tenantID, typeCode)
		require.NoError(t, err)
		assert.Equal(t, 1000, config.ChunkingStrategy.MaxChunkSize)
	})
}
