package metadata

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/hondyman/semlayer/backend/models"
	"github.com/hondyman/semlayer/backend/pkg/llm"
	"github.com/jmoiron/sqlx"
)

// CatalogEmbeddingService generates and maintains embeddings for catalog nodes.
type CatalogEmbeddingService struct {
	db          *sqlx.DB
	llmProvider llm.LLMProvider
}

// NewCatalogEmbeddingService creates a new embedding service.
func NewCatalogEmbeddingService(db *sqlx.DB, llmProvider llm.LLMProvider) *CatalogEmbeddingService {
	return &CatalogEmbeddingService{
		db:          db,
		llmProvider: llmProvider,
	}
}

// CatalogNodeForEmbedding represents a catalog node that needs an embedding.
type CatalogNodeForEmbedding struct {
	ID            string         `db:"id"`
	NodeName      string         `db:"node_name"`
	QualifiedPath string         `db:"qualified_path"`
	Description   sql.NullString `db:"description"`
	NodeType      string         `db:"node_type"`
	Properties    sql.NullString `db:"properties"`
}

// GenerateEmbeddingsForTenant generates embeddings for all catalog nodes in a tenant/datasource.
func (s *CatalogEmbeddingService) GenerateEmbeddingsForTenant(ctx context.Context, tenantID, datasourceID string) error {
	logger := logging.GetLogger().Sugar()
	logger.Infof("Starting embedding generation for tenant %s, datasource %s", tenantID, datasourceID)

	// Query for nodes that need embeddings
	query := `
		SELECT cn.id, cn.node_name, cn.qualified_path, cn.description, cnt.type_name as node_type, cn.properties::text
		FROM catalog_node cn
		JOIN catalog_node_type cnt ON cn.node_type_id = cnt.id
		WHERE cn.tenant_id = $1 
		  AND cn.tenant_datasource_id = $2
		  AND cn.embedding IS NULL
		  AND cnt.type_name IN ('calculation', 'metric', 'measure', 'dimension', 'view', 'semantic_model', 'table')
		ORDER BY cn.created_at DESC
	`

	var nodes []CatalogNodeForEmbedding
	err := s.db.SelectContext(ctx, &nodes, query, tenantID, datasourceID)
	if err != nil {
		return fmt.Errorf("failed to query nodes for embedding: %w", err)
	}

	logger.Infof("Found %d nodes requiring embeddings", len(nodes))

	// Process nodes in batches to avoid overwhelming the API
	batchSize := 10
	successCount := 0
	errorCount := 0

	for i := 0; i < len(nodes); i += batchSize {
		end := i + batchSize
		if end > len(nodes) {
			end = len(nodes)
		}

		batch := nodes[i:end]
		logger.Infof("Processing batch %d-%d of %d", i+1, end, len(nodes))

		for _, node := range batch {
			// Build text representation for embedding
			text := s.buildNodeText(node)

			// Generate embedding
			embedding, err := s.llmProvider.Embed(ctx, text)
			if err != nil {
				logger.Errorf("Failed to generate embedding for node %s: %v", node.ID, err)
				errorCount++
				continue
			}

			// Convert to PostgreSQL vector format
			embeddingStr := vectorToString(embedding)

			// Update the node with the embedding
			_, err = s.db.ExecContext(ctx, `
				UPDATE catalog_node 
				SET embedding = $1::vector, updated_at = $2
				WHERE id = $3
			`, embeddingStr, time.Now(), node.ID)

			if err != nil {
				logger.Errorf("Failed to save embedding for node %s: %v", node.ID, err)
				errorCount++
				continue
			}

			successCount++
		}

		// Rate limiting: small delay between batches
		time.Sleep(1 * time.Second)
	}

	logger.Infof("Embedding generation complete: %d successful, %d errors", successCount, errorCount)
	return nil
}

// RegenerateEmbeddingForNode regenerates the embedding for a single node.
func (s *CatalogEmbeddingService) RegenerateEmbeddingForNode(ctx context.Context, nodeID string) error {
	var node CatalogNodeForEmbedding
	err := s.db.GetContext(ctx, &node, `
		SELECT cn.id, cn.node_name, cn.qualified_path, cn.description, cnt.type_name as node_type, cn.properties::text
		FROM catalog_node cn
		JOIN catalog_node_type cnt ON cn.node_type_id = cnt.id
		WHERE cn.id = $1
	`, nodeID)

	if err != nil {
		return fmt.Errorf("failed to fetch node: %w", err)
	}

	// Build text and generate embedding
	text := s.buildNodeText(node)
	embedding, err := s.llmProvider.Embed(ctx, text)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Save embedding
	embeddingStr := vectorToString(embedding)
	_, err = s.db.ExecContext(ctx, `
		UPDATE catalog_node 
		SET embedding = $1::vector, updated_at = $2
		WHERE id = $3
	`, embeddingStr, time.Now(), nodeID)

	if err != nil {
		return fmt.Errorf("failed to save embedding: %w", err)
	}

	return nil
}

// GenerateEmbeddingsForBusinessTerms generates embeddings for business terms.
func (s *CatalogEmbeddingService) GenerateEmbeddingsForBusinessTerms(ctx context.Context, tenantID uuid.UUID) error {
	logger := logging.GetLogger().Sugar()
	logger.Infof("Starting business term embedding generation for tenant %s", tenantID)

	query := `
		SELECT id, term, definition, synonyms::text, scope, canonical_key
		FROM business_terms
		WHERE tenant_id = $1 AND embedding IS NULL
	`

	var terms []models.BusinessTerm
	err := s.db.SelectContext(ctx, &terms, query, tenantID)
	if err != nil {
		return fmt.Errorf("failed to query terms for embedding: %w", err)
	}

	logger.Infof("Found %d terms requiring embeddings", len(terms))

	for _, term := range terms {
		text := s.buildTermText(term)
		embedding, err := s.llmProvider.Embed(ctx, text)
		if err != nil {
			logger.Errorf("Failed to generate embedding for term %s: %v", term.ID, err)
			continue
		}

		embeddingStr := vectorToString(embedding)
		_, err = s.db.ExecContext(ctx, `
			UPDATE business_terms 
			SET embedding = $1::vector, updated_at = $2
			WHERE id = $3
		`, embeddingStr, time.Now(), term.ID)

		if err != nil {
			logger.Errorf("Failed to save embedding for term %s: %v", term.ID, err)
		}
	}

	return nil
}

// buildNodeText creates a rich text representation of a catalog node for embedding.
func (s *CatalogEmbeddingService) buildNodeText(node CatalogNodeForEmbedding) string {
	var parts []string

	// Core metadata
	parts = append(parts, fmt.Sprintf("Type: %s", node.NodeType))
	parts = append(parts, fmt.Sprintf("Name: %s", node.NodeName))
	parts = append(parts, fmt.Sprintf("Path: %s", node.QualifiedPath))

	// Description
	if node.Description.Valid && node.Description.String != "" {
		parts = append(parts, fmt.Sprintf("Description: %s", node.Description.String))
	}

	// Rich metadata from properties
	if node.Properties.Valid && node.Properties.String != "" {
		// In a real implementation, we'd parse JSON and extract specific fields like "owner", "tags", "layer"
		// For now, we'll just append the raw JSON but truncated
		props := node.Properties.String
		if len(props) > 1000 {
			props = props[:1000] + "..."
		}
		parts = append(parts, fmt.Sprintf("Metadata: %s", props))
	}

	return strings.Join(parts, "\n")
}

// buildTermText creates a rich text representation of a business term.
func (s *CatalogEmbeddingService) buildTermText(term models.BusinessTerm) string {
	var parts []string
	parts = append(parts, fmt.Sprintf("Term: %s", term.Term))
	parts = append(parts, fmt.Sprintf("Definition: %s", term.Definition))
	parts = append(parts, fmt.Sprintf("Scope: %s", term.Scope))

	if term.CanonicalKey != "" {
		parts = append(parts, fmt.Sprintf("Canonical Key: %s", term.CanonicalKey))
	}

	// Synonyms are JSON array
	if len(term.Synonyms) > 0 {
		parts = append(parts, fmt.Sprintf("Synonyms: %s", string(term.Synonyms)))
	}

	return strings.Join(parts, "\n")
}

// vectorToString converts a float32 slice to PostgreSQL vector format string.
func vectorToString(vec []float32) string {
	if len(vec) == 0 {
		return "[]"
	}
	parts := make([]string, len(vec))
	for i, v := range vec {
		parts[i] = fmt.Sprintf("%f", v)
	}
	return "[" + strings.Join(parts, ",") + "]"
}
