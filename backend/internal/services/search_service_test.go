package services

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/security"
	"github.com/hondyman/semlayer/backend/internal/testutils"
	"github.com/hondyman/semlayer/backend/models"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestHybridSearch(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "postgres")

	llmProvider := &testutils.MockLLMProvider{}
	svc := NewSearchService(sqlxDB, llmProvider)

	ctx := context.Background()
	secCtx := security.Context{TenantID: uuid.New().String(), DatasourceID: "ds-1"}
	req := models.SemanticSearchRequest{Query: "revenue", DatasourceID: "ds-1", Region: "us-east-1"}

	t.Run("Successful Search", func(t *testing.T) {
		// Mock Embed call
		llmProvider.EmbedFunc = func(ctx context.Context, text string) ([]float32, error) {
			return []float32{0.1, 0.2, 0.3}, nil
		}

		// Mock DB query
		rows := sqlmock.NewRows([]string{"id", "name", "description", "type", "qualified_path", "text_score", "vector_score", "final_score"}).
			AddRow(uuid.New(), "Revenue", "Total Revenue", "catalog_node", "finance.revenue", 0.8, 0.9, 0.87).
			AddRow(uuid.New(), "Gross Profit", "Gross Profit", "business_term", "finance.gross_profit", 0.5, 0.7, 0.64)

		mock.ExpectQuery(`WITH catalog_results AS`).
			WithArgs(req.Query, "[0.100000,0.200000,0.300000]", secCtx.TenantID, secCtx.DatasourceID).
			WillReturnRows(rows)

		results, err := svc.HybridSearch(ctx, req, secCtx)
		require.NoError(t, err)
		require.Len(t, results, 2)
		require.Equal(t, "Revenue", results[0].Name)
		require.Equal(t, "finance.revenue", results[0].QualifiedPath)
		require.Equal(t, "catalog_node", results[0].Type)

		require.Equal(t, "Gross Profit", results[1].Name)
		require.Equal(t, "finance.gross_profit", results[1].QualifiedPath)
		require.Equal(t, "business_term", results[1].Type)
	})

	t.Run("Embed Failure", func(t *testing.T) {
		llmProvider.EmbedFunc = func(ctx context.Context, text string) ([]float32, error) {
			return nil, sql.ErrConnDone // Simulate error
		}

		_, err := svc.HybridSearch(ctx, req, secCtx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to embed query")
	})
}
