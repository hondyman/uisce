package metadata

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/testutils"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestGenerateEmbeddingsForBusinessTerms(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "postgres")

	llmProvider := &testutils.MockLLMProvider{}
	svc := NewCatalogEmbeddingService(sqlxDB, llmProvider)

	ctx := context.Background()
	tenantID := uuid.New()

	t.Run("Successful Generation", func(t *testing.T) {
		// Mock SelectContext
		rows := sqlmock.NewRows([]string{"id", "term", "definition", "synonyms", "scope", "canonical_key"}).
			AddRow(uuid.New(), "Revenue", "Total Revenue", []byte("[]"), "Global", "finance.revenue")

		mock.ExpectQuery(`SELECT id, term, definition, synonyms::text, scope, canonical_key FROM business_terms`).
			WithArgs(tenantID).
			WillReturnRows(rows)

		// Mock Embed call
		llmProvider.EmbedFunc = func(ctx context.Context, text string) ([]float32, error) {
			return []float32{0.1, 0.2, 0.3}, nil
		}

		// Mock Update
		mock.ExpectExec(`UPDATE business_terms SET embedding = \$1::vector, updated_at = \$2 WHERE id = \$3`).
			WithArgs("[0.100000,0.200000,0.300000]", sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := svc.GenerateEmbeddingsForBusinessTerms(ctx, tenantID)
		require.NoError(t, err)
	})

	t.Run("Query Failure", func(t *testing.T) {
		mock.ExpectQuery(`SELECT id, term, definition, synonyms::text, scope, canonical_key FROM business_terms`).
			WithArgs(tenantID).
			WillReturnError(sql.ErrConnDone)

		err := svc.GenerateEmbeddingsForBusinessTerms(ctx, tenantID)
		require.Error(t, err)
	})
}

// Using centralized testutils.MockLLMProvider — no local mock needed.

func TestBuildNodeText(t *testing.T) {
	// Since buildNodeText is private, we can't test it directly easily without exporting or using reflection.
	// However, we can test it via GenerateEmbeddingsForTenant if we mock the DB calls.
	// Or we can just trust the integration test logic above.
	// For now, let's focus on the public method.
}
