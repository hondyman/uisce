package rag

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/config"
	"github.com/hondyman/semlayer/backend/internal/tenant"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHybridSearch_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Load configuration
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		cfg, err := config.LoadConfig("../../config.yaml")
		if err == nil {
			dsn = cfg.DSN
		} else {
			dsn = "postgres://postgres:postgres@localhost:5432/semlayer?sslmode=disable"
		}
	}

	db, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	defer db.Close()

	err = db.Ping()
	if err != nil {
		t.Skipf("Skipping integration test: database not available: %v", err)
	}

	// Skip if the 'vector' extension is not available in this DB
	var extName string
	if err := db.QueryRowContext(context.Background(), "SELECT name FROM pg_available_extensions WHERE name = 'vector' LIMIT 1").Scan(&extName); err != nil || extName != "vector" {
		t.Skip("Skipping integration test: 'vector' extension not available in this DB")
		return
	}

	// Setup dependencies
	tm := tenant.NewTenantManager(db, nil)
	is := NewIngestionService()
	es := NewOpenAIEmbedder("dummy-key", "text-embedding-ada-002")
	ss := NewSearchService(es)

	ctx := context.Background()

	// Create a unique tenant
	tenantCode := fmt.Sprintf("test_search_%d", time.Now().Unix())
	tenantName := fmt.Sprintf("Search Test Tenant %d", time.Now().Unix())
	ten, err := tm.CreateTenant(ctx, tenantCode, tenantName)
	require.NoError(t, err)

	// Ingest sample data
	docID := uuid.New()
	doc := &Document{
		DocumentID:   docID,
		SourcePath:   "search_doc.txt",
		DocumentType: "text",
		Title:        "Search Test Document",
		FileHash:     uuid.New().String(),
		Status:       "pending",
	}

	conn, err := tm.GetTenantConnection(ctx, ten.TenantID)
	require.NoError(t, err)
	defer conn.Close()

	err = is.CreateDocumentRecord(ctx, conn, doc)
	require.NoError(t, err)

	chunks := []DocumentChunk{
		{
			ChunkID:    uuid.New(),
			DocumentID: docID,
			ChunkIndex: 0,
			Content:    "The quick brown fox jumps over the lazy dog.",
			Embedding:  mustEmbed(t, es, "The quick brown fox jumps over the lazy dog."),
			TokenCount: 9,
		},
		{
			ChunkID:    uuid.New(),
			DocumentID: docID,
			ChunkIndex: 1,
			Content:    "Artificial intelligence is transforming the world.",
			Embedding:  mustEmbed(t, es, "Artificial intelligence is transforming the world."),
			TokenCount: 7,
		},
		{
			ChunkID:    uuid.New(),
			DocumentID: docID,
			ChunkIndex: 2,
			Content:    "Go is an open source programming language supported by Google.",
			Embedding:  mustEmbed(t, es, "Go is an open source programming language supported by Google."),
			TokenCount: 10,
		},
	}

	err = is.StoreChunks(ctx, conn, chunks)
	require.NoError(t, err)

	// Test Cases
	tests := []struct {
		name           string
		query          string
		semanticWeight float64
		keywordWeight  float64
		expectedText   string
	}{
		{
			name:           "Keyword Match",
			query:          "fox",
			semanticWeight: 0.0,
			keywordWeight:  1.0,
			expectedText:   "The quick brown fox jumps over the lazy dog.",
		},
		{
			name:           "Semantic Match",
			query:          "programming",
			semanticWeight: 1.0,
			keywordWeight:  0.0,
			expectedText:   "Go is an open source programming language supported by Google.",
		},
		{
			name:           "Hybrid Match",
			query:          "intelligence",
			semanticWeight: 0.5,
			keywordWeight:  0.5,
			expectedText:   "Artificial intelligence is transforming the world.",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := SearchRequest{
				Query:          tc.query,
				Limit:          1,
				MinScore:       0.0,
				SemanticWeight: tc.semanticWeight,
				KeywordWeight:  tc.keywordWeight,
			}

			results, err := ss.HybridSearch(ctx, conn, req)
			require.NoError(t, err)
			require.NotEmpty(t, results)
			assert.Contains(t, results[0].Content, tc.expectedText)
		})
	}
}

func mustEmbed(t *testing.T, es EmbeddingService, text string) []float32 {
	emb, err := es.Embed(context.Background(), text)
	require.NoError(t, err)
	return emb
}
