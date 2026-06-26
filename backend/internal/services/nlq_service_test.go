package services

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/security"
	"github.com/hondyman/semlayer/backend/internal/testutils"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

// MockSearchService is not easily mockable because SearchService is a struct, not an interface.
// However, we can mock the DB calls made by SearchService if we use the real one,
// OR we can refactor NLQService to take an interface.
// For now, let's use the real SearchService with mocked DB and LLM.

func TestNLQAsk(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "postgres")

	// Initialize services with mocks
	mockProvider := &testutils.MockLLMProvider{}
	searchSvc := NewSearchService(sqlxDB, mockProvider) // Use sqlxDB here
	reasoningEngine := NewReasoningEngine(mockProvider)
	financialTools := NewFinancialToolService(db)

	nlqSvc := NewNLQService(sqlxDB, mockProvider, searchSvc, reasoningEngine, financialTools) // Use sqlxDB here

	ctx := context.Background()
	tenantID := uuid.New().String()
	datasourceID := uuid.New().String()
	secCtx := &security.Context{
		TenantID:     tenantID,
		DatasourceID: datasourceID,
		Region:       "us-east-1",
	}

	t.Run("Ask with Target Path", func(t *testing.T) {
		req := AskRequest{
			Question:         "Explain revenue",
			TargetEntityPath: "finance.revenue",
		}

		// Mock DAG retrieval
		dagJSON := `{"nodes": [{"name": "Revenue", "path": "finance.revenue", "type": "metric"}], "edges": []}`
		mock.ExpectQuery(`SELECT get_calc_dag_with_metadata\(\$1, \$2\)`).
			WithArgs(req.TargetEntityPath, secCtx.TenantID).
			WillReturnRows(sqlmock.NewRows([]string{"get_calc_dag_with_metadata"}).AddRow([]byte(dagJSON)))

		// Mock LLM generation
		// Note: MockLLMProvider.GenerateResponse returns "Mock Response"

		resp, err := nlqSvc.Ask(ctx, secCtx, req)
		require.NoError(t, err)
		require.Equal(t, "Mock Response", resp.Answer)
		require.Equal(t, "finance.revenue", resp.ResolvedEntityPath)
	})

	t.Run("Ask without Target Path (Auto-discovery)", func(t *testing.T) {
		req := AskRequest{
			Question: "What is revenue?",
		}

		// Mock Hybrid Search (via SearchService)
		// 1. Embed query
		mockProvider.EmbedFunc = func(ctx context.Context, text string) ([]float32, error) {
			return []float32{0.1, 0.2, 0.3}, nil
		}

		// 2. DB Query for Hybrid Search
		rows := sqlmock.NewRows([]string{"id", "name", "description", "type", "qualified_path", "text_score", "vector_score", "final_score"}).
			AddRow(uuid.New(), "Revenue", "Total Revenue", "catalog_node", "finance.revenue", 0.9, 0.9, 0.9)

		mock.ExpectQuery(`WITH catalog_results AS`).
			WithArgs(req.Question, "[0.100000,0.200000,0.300000]", secCtx.TenantID, secCtx.DatasourceID).
			WillReturnRows(rows)

		// Mock DAG retrieval
		dagJSON := `{"nodes": [{"name": "Revenue", "path": "finance.revenue", "type": "metric"}], "edges": []}`
		mock.ExpectQuery(`SELECT get_calc_dag_with_metadata\(\$1, \$2\)`).
			WithArgs("finance.revenue", secCtx.TenantID).
			WillReturnRows(sqlmock.NewRows([]string{"get_calc_dag_with_metadata"}).AddRow([]byte(dagJSON)))

		resp, err := nlqSvc.Ask(ctx, secCtx, req)
		require.NoError(t, err)
		require.Equal(t, "Mock Response", resp.Answer)
		require.Equal(t, "finance.revenue", resp.ResolvedEntityPath)
	})
}
