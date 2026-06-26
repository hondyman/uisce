package rag

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/pgvector/pgvector-go"
)

// Store handles RAG storage operations using PostgreSQL and pgvector
type Store struct {
	db *sql.DB
}

// NewStore creates a new RAG store
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// ProvisionTenant creates the schema and tables for a new tenant
func (s *Store) ProvisionTenant(ctx context.Context, tenantID string) error {
	schemaName := fmt.Sprintf("tenant_%s", tenantID)

	// In a real implementation, we would read the SQL template from a file
	// For now, we'll embed the SQL logic here for simplicity
	
	// 1. Create Schema
	createSchemaSQL := fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", pqQuoteIdent(schemaName))
	if _, err := s.db.ExecContext(ctx, createSchemaSQL); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	// 2. Create Chunks Table
	createTableSQL := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s.chunks (
			chunk_id TEXT PRIMARY KEY,
			document_id TEXT NOT NULL,
			chunk_index INT NOT NULL,
			text TEXT NOT NULL,
			token_count INT NOT NULL,
			metadata JSONB DEFAULT '{}'::jsonb,
			source_snapshot_id TEXT,
			embedding vector(1536),
			created_at TIMESTAMPTZ DEFAULT now()
		)`, pqQuoteIdent(schemaName))
	if _, err := s.db.ExecContext(ctx, createTableSQL); err != nil {
		return fmt.Errorf("failed to create chunks table: %w", err)
	}

	// 3. Create Indexes
	// IVFFlat index
	createIndexSQL := fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS chunks_embedding_idx 
		ON %s.chunks 
		USING ivfflat (embedding vector_cosine_ops) 
		WITH (lists = 100)`, pqQuoteIdent(schemaName))
	if _, err := s.db.ExecContext(ctx, createIndexSQL); err != nil {
		return fmt.Errorf("failed to create embedding index: %w", err)
	}

	// Metadata indexes
	createDocIndexSQL := fmt.Sprintf("CREATE INDEX IF NOT EXISTS chunks_document_idx ON %s.chunks (document_id)", pqQuoteIdent(schemaName))
	if _, err := s.db.ExecContext(ctx, createDocIndexSQL); err != nil {
		return fmt.Errorf("failed to create document index: %w", err)
	}

	return nil
}

// UpsertChunks inserts or updates chunks for a tenant
func (s *Store) UpsertChunks(ctx context.Context, tenantID string, chunks []Chunk) error {
	schemaName := fmt.Sprintf("tenant_%s", tenantID)

	return WithTenantTx(ctx, s.db, schemaName, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO chunks (chunk_id, document_id, chunk_index, text, token_count, metadata, source_snapshot_id, embedding, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now())
			ON CONFLICT (chunk_id) DO UPDATE
			SET text = EXCLUDED.text,
				token_count = EXCLUDED.token_count,
				metadata = EXCLUDED.metadata,
				source_snapshot_id = EXCLUDED.source_snapshot_id,
				embedding = EXCLUDED.embedding
		`)
		if err != nil {
			return err
		}
		defer stmt.Close()

		for _, chunk := range chunks {
			metadataJSON, err := json.Marshal(chunk.Metadata)
			if err != nil {
				return fmt.Errorf("failed to marshal metadata: %w", err)
			}

			// Convert []float32 to pgvector.Vector
			vector := pgvector.NewVector(chunk.Embedding)

			_, err = stmt.ExecContext(ctx,
				chunk.ChunkID,
				chunk.DocumentID,
				chunk.Index,
				chunk.Text,
				chunk.TokenCount,
				metadataJSON,
				chunk.SourceSnapshotID,
				vector,
			)
			if err != nil {
				return fmt.Errorf("failed to upsert chunk %s: %w", chunk.ChunkID, err)
			}
		}
		return nil
	})
}

// HybridSearch performs a search combining vector similarity and metadata filtering
func (s *Store) HybridSearch(ctx context.Context, tenantID string, queryEmbedding []float32, filters map[string]any, limit int) ([]ChunkHit, error) {
	schemaName := fmt.Sprintf("tenant_%s", tenantID)
	var hits []ChunkHit

	err := WithTenantTx(ctx, s.db, schemaName, func(tx *sql.Tx) error {
		// Build query
		// Note: <#> is cosine distance. We want similarity, so 1 - distance.
		// We order by distance ASC (most similar first).
		
		query := `
			SELECT chunk_id, document_id, chunk_index, text, metadata, source_snapshot_id, 
			       1 - (embedding <=> $1) as similarity
			FROM chunks
			WHERE 1=1
		`
		args := []any{pgvector.NewVector(queryEmbedding)}
		argIdx := 2

		// Add filters
		// This is a simple implementation. A real one would handle complex filter logic.
		for k, v := range filters {
			// specific handling for section filter as an example
			if k == "section" {
				query += fmt.Sprintf(" AND metadata->>'section' = $%d", argIdx)
				args = append(args, v)
				argIdx++
			} else if k == "document_id" {
				query += fmt.Sprintf(" AND document_id = $%d", argIdx)
				args = append(args, v)
				argIdx++
			}
		}

		query += fmt.Sprintf(" ORDER BY embedding <=> $1 ASC LIMIT $%d", argIdx)
		args = append(args, limit)

		rows, err := tx.QueryContext(ctx, query, args...)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var hit ChunkHit
			var metadataJSON []byte
			
			if err := rows.Scan(
				&hit.ChunkID,
				&hit.DocumentID,
				&hit.ChunkIndex,
				&hit.Text,
				&metadataJSON,
				&hit.SourceSnapshotID,
				&hit.Similarity,
			); err != nil {
				return err
			}

			if err := json.Unmarshal(metadataJSON, &hit.Metadata); err != nil {
				return fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
			hits = append(hits, hit)
		}
		return rows.Err()
	})

	return hits, err
}
