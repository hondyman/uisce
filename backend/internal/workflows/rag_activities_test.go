package workflows

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/config"
	"github.com/hondyman/semlayer/backend/internal/rag"
	"github.com/hondyman/semlayer/backend/internal/tenant"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocumentIngestion_Integration(t *testing.T) {
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
	is := rag.NewIngestionService()
	es := rag.NewOpenAIEmbedder("dummy-key", "text-embedding-ada-002")
	activities := NewDocumentActivities(tm, is, es, nil)

	ctx := context.Background()

	// Create a unique tenant
	tenantCode := fmt.Sprintf("test_rag_%d", time.Now().Unix())
	tenantName := fmt.Sprintf("RAG Test Tenant %d", time.Now().Unix())
	ten, err := tm.CreateTenant(ctx, tenantCode, tenantName)
	require.NoError(t, err)

	// Create a document record first (needed for FK constraint)
	docID := uuid.New()
	doc := &rag.Document{
		DocumentID:   docID,
		SourcePath:   "test_doc.txt",
		DocumentType: "text",
		Title:        "Test Document",
		FileHash:     uuid.New().String(),
		Status:       "pending",
	}

	// We need to insert the document using a tenant-scoped connection
	conn, err := tm.GetTenantConnection(ctx, ten.TenantID)
	require.NoError(t, err)

	err = is.CreateDocumentRecord(ctx, conn, doc)
	conn.Close() // Close connection to return to pool
	require.NoError(t, err)

	// Run Activities Sequence
	// 1. Extract Text
	text, err := activities.ExtractTextActivity(ctx, doc.SourcePath)
	require.NoError(t, err)
	assert.Contains(t, text, "Content extracted from")

	// 2. Chunk Document
	chunks, err := activities.ChunkDocumentActivity(ctx, docID, doc.SourcePath, text)
	require.NoError(t, err)
	assert.NotEmpty(t, chunks)

	// 3. Generate Embeddings
	embeddings, err := activities.GenerateEmbeddingsActivity(ctx, chunks)
	require.NoError(t, err)
	assert.Equal(t, len(chunks), len(embeddings))
	assert.Equal(t, 1536, len(embeddings[0]))

	// 4. Store Chunks
	err = activities.StoreChunksActivity(ctx, ten.TenantID, docID, chunks, embeddings)
	require.NoError(t, err)

	// Verify Data in DB
	conn, err = tm.GetTenantConnection(ctx, ten.TenantID)
	require.NoError(t, err)
	defer conn.Close()

	var count int
	err = conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM document_chunks WHERE document_id = $1", docID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, len(chunks), count)

	// Verify vector data exists
	var embeddingStr string
	err = conn.QueryRowContext(ctx, "SELECT embedding::text FROM document_chunks WHERE document_id = $1 LIMIT 1", docID).Scan(&embeddingStr)
	require.NoError(t, err)
	assert.NotEmpty(t, embeddingStr)
}
