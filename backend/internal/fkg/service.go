package fkg

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Entity represents a financial entity in the knowledge graph.
type Entity struct {
	EntityID    uuid.UUID              `json:"entity_id" db:"entity_id"`
	TenantID    uuid.UUID              `json:"tenant_id" db:"tenant_id"`
	EntityType  string                 `json:"entity_type" db:"entity_type"`
	Name        string                 `json:"name" db:"name"`
	CanonicalID string                 `json:"canonical_id,omitempty" db:"canonical_id"`
	Properties  map[string]interface{} `json:"properties,omitempty" db:"properties"`
	RiskScore   float64                `json:"risk_score,omitempty" db:"risk_score"`
	Status      string                 `json:"status" db:"status"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`
}

// OwnershipRelationship represents an ownership relationship between entities.
type OwnershipRelationship struct {
	RelationshipID      uuid.UUID              `json:"relationship_id" db:"relationship_id"`
	TenantID            uuid.UUID              `json:"tenant_id" db:"tenant_id"`
	SourceEntityID      uuid.UUID              `json:"source_entity_id" db:"source_entity_id"`
	TargetEntityID      uuid.UUID              `json:"target_entity_id" db:"target_entity_id"`
	RelationshipType    string                 `json:"relationship_type" db:"relationship_type"`
	PercentageOwnership float64                `json:"percentage_ownership,omitempty" db:"percentage_ownership"`
	VotingRights        float64                `json:"voting_rights,omitempty" db:"voting_rights"`
	EffectiveDate       *time.Time             `json:"effective_date,omitempty" db:"effective_date"`
	EndDate             *time.Time             `json:"end_date,omitempty" db:"end_date"`
	Properties          map[string]interface{} `json:"properties,omitempty" db:"properties"`
	CreatedAt           time.Time              `json:"created_at" db:"created_at"`
}

// OwnershipChain represents a link in the UBO ownership chain.
type OwnershipChain struct {
	EntityID            uuid.UUID `json:"entity_id" db:"entity_id"`
	ParentEntityID      uuid.UUID `json:"parent_entity_id" db:"parent_entity_id"`
	EntityName          string    `json:"entity_name" db:"entity_name"`
	EntityType          string    `json:"entity_type" db:"entity_type"`
	PercentageOwnership float64   `json:"percentage_ownership" db:"percentage_ownership"`
	CumulativeOwnership float64   `json:"cumulative_ownership" db:"cumulative_ownership"`
	Depth               int       `json:"depth" db:"depth"`
	Path                string    `json:"path" db:"path"`
}

// DocumentChunk represents a chunk of document text with embedding.
type DocumentChunk struct {
	ChunkID     uuid.UUID `json:"chunk_id" db:"chunk_id"`
	TenantID    uuid.UUID `json:"tenant_id" db:"tenant_id"`
	DocumentID  uuid.UUID `json:"document_id" db:"document_id"`
	EntityID    uuid.UUID `json:"entity_id,omitempty" db:"entity_id"`
	ChunkIndex  int       `json:"chunk_index" db:"chunk_index"`
	Content     string    `json:"content" db:"content"`
	ContentHash string    `json:"content_hash" db:"content_hash"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// SearchResult represents a hybrid search result.
type SearchResult struct {
	ChunkID       uuid.UUID `json:"chunk_id" db:"chunk_id"`
	DocumentID    uuid.UUID `json:"document_id" db:"document_id"`
	EntityID      uuid.UUID `json:"entity_id" db:"entity_id"`
	Content       string    `json:"content" db:"content"`
	CombinedScore float64   `json:"combined_score" db:"combined_score"`
	KeywordRank   int       `json:"keyword_rank" db:"keyword_rank"`
	SemanticRank  int       `json:"semantic_rank" db:"semantic_rank"`
}

// SimilarEntity represents an entity found by similarity search.
type SimilarEntity struct {
	EntityID   uuid.UUID `json:"entity_id" db:"entity_id"`
	Name       string    `json:"name" db:"name"`
	EntityType string    `json:"entity_type" db:"entity_type"`
	Similarity float64   `json:"similarity" db:"similarity"`
}

// FKGService provides Financial Knowledge Graph operations.
type FKGService struct {
	db              *sqlx.DB
	embeddingClient EmbeddingClient
}

// EmbeddingClient generates embeddings for text.
type EmbeddingClient interface {
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
}

// HasuraClient interface for GraphQL operations
type HasuraClient interface {
	Query(ctx context.Context, query string, variables map[string]interface{}, result interface{}) error
	Mutate(ctx context.Context, mutation string, variables map[string]interface{}, result interface{}) error
}

// NewFKGService creates a new FKG service.
func NewFKGService(db *sqlx.DB, embeddingClient EmbeddingClient) *FKGService {
	return &FKGService{
		db:              db,
		embeddingClient: embeddingClient,
	}
}

// CreateEntity creates a new financial entity.
func (s *FKGService) CreateEntity(ctx context.Context, tenantID string, entity interface{}) (interface{}, error) {
	entityMap, ok := entity.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid entity type")
	}

	entityID := entityMap["entity_id"].(string)
	entityType := entityMap["entity_type"].(string)
	name := entityMap["name"].(string)
	canonicalID, _ := entityMap["canonical_id"].(string)
	properties, _ := entityMap["properties"].(map[string]interface{})

	propertiesJSON, err := json.Marshal(properties)
	if err != nil {
		propertiesJSON = []byte("{}")
	}

	err = s.createEntityRecord(ctx, entityID, tenantID, entityType, name, canonicalID, propertiesJSON)
	if err != nil {
		return nil, fmt.Errorf("create entity: %w", err)
	}

	return s.GetEntity(ctx, tenantID, entityID)
}

// GetEntity retrieves an entity by ID.
func (s *FKGService) GetEntity(ctx context.Context, tenantID, entityID string) (interface{}, error) {
	entity, err := s.getEntityRecord(ctx, tenantID, entityID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("entity not found")
		}
		return nil, fmt.Errorf("get entity: %w", err)
	}
	return entity, nil
}

// UpdateEntity updates an entity's properties.
func (s *FKGService) UpdateEntity(ctx context.Context, tenantID, entityID string, updates map[string]interface{}) error {
	// Build dynamic update query
	setClauses := []string{"updated_at = NOW()"}
	args := []interface{}{tenantID, entityID}
	argNum := 3

	if name, ok := updates["name"].(string); ok {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argNum))
		args = append(args, name)
		argNum++
	}

	if entityType, ok := updates["entity_type"].(string); ok {
		setClauses = append(setClauses, fmt.Sprintf("entity_type = $%d", argNum))
		args = append(args, entityType)
		argNum++
	}

	if canonicalID, ok := updates["canonical_id"].(string); ok {
		setClauses = append(setClauses, fmt.Sprintf("canonical_id = NULLIF($%d, '')", argNum))
		args = append(args, canonicalID)
		argNum++
	}

	if properties, ok := updates["properties"].(map[string]interface{}); ok {
		propertiesJSON, _ := json.Marshal(properties)
		setClauses = append(setClauses, fmt.Sprintf("properties = properties || $%d::jsonb", argNum))
		args = append(args, propertiesJSON)
		argNum++
	}

	if riskScore, ok := updates["risk_score"].(float64); ok {
		setClauses = append(setClauses, fmt.Sprintf("risk_score = $%d", argNum))
		args = append(args, riskScore)
		argNum++
	}

	query := fmt.Sprintf(`
		UPDATE financial_entities 
		SET %s 
		WHERE tenant_id = $1 AND entity_id = $2
	`, strings.Join(setClauses, ", "))

	err := s.updateEntityRecord(ctx, query, args...)
	if err != nil {
		return err
	}

	return nil
}

// DeleteEntity soft-deletes an entity.
func (s *FKGService) DeleteEntity(ctx context.Context, tenantID, entityID string) error {
	return s.deleteEntityRecord(ctx, tenantID, entityID)
}

// ListEntities lists entities for a tenant with optional filtering.
func (s *FKGService) ListEntities(ctx context.Context, tenantID string, entityType string, limit, offset int) ([]interface{}, error) {
	var entities []Entity
	var err error

	entities, err = s.listEntitiesRecords(ctx, tenantID, entityType, limit, offset)

	if err != nil {
		return nil, fmt.Errorf("list entities: %w", err)
	}

	result := make([]interface{}, len(entities))
	for i, e := range entities {
		result[i] = e
	}
	return result, nil
}

// FindSimilarEntities finds entities with similar names using pg_trgm.
func (s *FKGService) FindSimilarEntities(ctx context.Context, tenantID, name string, threshold float64) ([]interface{}, error) {
	entities, err := s.findSimilarEntitiesRecords(ctx, tenantID, name, threshold)
	if err != nil {
		return nil, fmt.Errorf("find similar entities: %w", err)
	}

	result := make([]interface{}, len(entities))
	for i, e := range entities {
		result[i] = e
	}
	return result, nil
}

// CreateRelationship creates an ownership relationship between entities.
func (s *FKGService) CreateRelationship(ctx context.Context, tenantID string, rel interface{}) error {
	relMap, ok := rel.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid relationship type")
	}

	relationshipID := relMap["relationship_id"].(string)
	sourceEntityID := relMap["source_entity_id"].(string)
	targetEntityID := relMap["target_entity_id"].(string)
	relationshipType := relMap["relationship_type"].(string)
	percentageOwnership, _ := relMap["percentage_ownership"].(float64)
	votingRights, _ := relMap["voting_rights"].(float64)
	effectiveDate, _ := relMap["effective_date"].(string)
	properties, _ := relMap["properties"].(map[string]interface{})

	propertiesJSON, err := json.Marshal(properties)
	if err != nil {
		propertiesJSON = []byte("{}")
	}

	err = s.createRelationshipRecord(ctx, relationshipID, tenantID, sourceEntityID, targetEntityID,
		relationshipType, percentageOwnership, votingRights, effectiveDate, propertiesJSON)
	if err != nil {
		return fmt.Errorf("create relationship: %w", err)
	}

	return nil
}

// GetUBOChain retrieves the Ultimate Beneficial Ownership chain for an entity.
func (s *FKGService) GetUBOChain(ctx context.Context, tenantID, entityID string, maxDepth int) ([]interface{}, error) {
	chain, err := s.getUBOChainRecords(ctx, tenantID, entityID, maxDepth)
	if err != nil {
		return nil, fmt.Errorf("get UBO chain: %w", err)
	}

	result := make([]interface{}, len(chain))
	for i, c := range chain {
		result[i] = c
	}
	return result, nil
}

// HybridSearchDocuments performs hybrid search combining keyword and semantic search.
func (s *FKGService) HybridSearchDocuments(ctx context.Context, tenantID, query string, embedding []float32, limit int) ([]interface{}, error) {
	// If no embedding provided, generate one
	if len(embedding) == 0 && s.embeddingClient != nil {
		var err error
		embedding, err = s.embeddingClient.GenerateEmbedding(ctx, query)
		if err != nil {
			// Fall back to keyword-only search
			return s.keywordSearchDocuments(ctx, tenantID, query, limit)
		}
	}

	// Convert embedding to string format for pgvector
	embeddingStr := formatEmbeddingVector(embedding)

	results, err := s.hybridSearchDocumentsRecords(ctx, tenantID, query, embeddingStr, limit)
	if err != nil {
		return nil, fmt.Errorf("hybrid search: %w", err)
	}

	result := make([]interface{}, len(results))
	for i, r := range results {
		result[i] = r
	}
	return result, nil
}

func (s *FKGService) keywordSearchDocuments(ctx context.Context, tenantID, query string, limit int) ([]interface{}, error) {
	results, err := s.keywordSearchDocumentsRecords(ctx, tenantID, query, limit)
	if err != nil {
		return nil, fmt.Errorf("keyword search: %w", err)
	}

	result := make([]interface{}, len(results))
	for i, r := range results {
		result[i] = r
	}
	return result, nil
}

func formatEmbeddingVector(embedding []float32) string {
	if len(embedding) == 0 {
		return ""
	}
	parts := make([]string, len(embedding))
	for i, v := range embedding {
		parts[i] = fmt.Sprintf("%f", v)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

// Helper methods for Hasura integration - SQL fallback for complex operations

func (s *FKGService) createEntityRecord(ctx context.Context, entityID, tenantID, entityType, name, canonicalID string, propertiesJSON []byte) error {
	// TODO: Use HasuraClient for INSERT when available
	// For now, use SQL fallback for INSERT with JSONB and NULLIF
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO financial_entities (
entity_id, tenant_id, entity_type, name, canonical_id, properties, status, created_at, updated_at
) VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, 'active', NOW(), NOW())
	`, entityID, tenantID, entityType, name, canonicalID, propertiesJSON)
	return err
}

func (s *FKGService) getEntityRecord(ctx context.Context, tenantID, entityID string) (*Entity, error) {
	// TODO: Use HasuraClient for SELECT when available
	// For now, use SQL fallback for SELECT with COALESCE and JSONB cast
	var entity Entity
	err := s.db.GetContext(ctx, &entity, `
		SELECT entity_id, tenant_id, entity_type, name, 
		       COALESCE(canonical_id, '') as canonical_id, 
		       COALESCE(properties::text, '{}') as properties,
		       COALESCE(risk_score, 0) as risk_score,
		       status, created_at, updated_at
		FROM financial_entities 
		WHERE tenant_id = $1 AND entity_id = $2
	`, tenantID, entityID)
	if err != nil {
		return nil, err
	}
	return &entity, nil
}

func (s *FKGService) updateEntityRecord(ctx context.Context, query string, args ...interface{}) error {
	// TODO: Use HasuraClient for UPDATE when available
	// For now, use SQL fallback for dynamic UPDATE with JSONB merge
	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update entity: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("entity not found")
	}
	return nil
}

func (s *FKGService) deleteEntityRecord(ctx context.Context, tenantID, entityID string) error {
	// TODO: Use HasuraClient for UPDATE when available
	// For now, use SQL fallback for soft delete
	result, err := s.db.ExecContext(ctx, `
		UPDATE financial_entities 
		SET status = 'deleted', updated_at = NOW() 
		WHERE tenant_id = $1 AND entity_id = $2 AND status != 'deleted'
	`, tenantID, entityID)
	if err != nil {
		return fmt.Errorf("delete entity: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("entity not found")
	}
	return nil
}

func (s *FKGService) listEntitiesRecords(ctx context.Context, tenantID, entityType string, limit, offset int) ([]Entity, error) {
	// TODO: Use HasuraClient for SELECT when available
	// For now, use SQL fallback for dynamic WHERE clause with pagination
	var entities []Entity
	var err error
	if entityType != "" {
		err = s.db.SelectContext(ctx, &entities, `
			SELECT entity_id, tenant_id, entity_type, name, 
			       COALESCE(canonical_id, '') as canonical_id, 
			       COALESCE(properties::text, '{}') as properties,
			       COALESCE(risk_score, 0) as risk_score,
			       status, created_at, updated_at
			FROM financial_entities 
			WHERE tenant_id = $1 AND entity_type = $2 AND status = 'active'
			ORDER BY created_at DESC
			LIMIT $3 OFFSET $4
		`, tenantID, entityType, limit, offset)
	} else {
		err = s.db.SelectContext(ctx, &entities, `
			SELECT entity_id, tenant_id, entity_type, name, 
			       COALESCE(canonical_id, '') as canonical_id, 
			       COALESCE(properties::text, '{}') as properties,
			       COALESCE(risk_score, 0) as risk_score,
			       status, created_at, updated_at
			FROM financial_entities 
			WHERE tenant_id = $1 AND status = 'active'
			ORDER BY created_at DESC
			LIMIT $2 OFFSET $3
		`, tenantID, limit, offset)
	}
	return entities, err
}

func (s *FKGService) findSimilarEntitiesRecords(ctx context.Context, tenantID, name string, threshold float64) ([]SimilarEntity, error) {
	// TODO: Use HasuraClient for SELECT when available
	// For now, use SQL fallback for pg_trgm similarity function
	var entities []SimilarEntity
	err := s.db.SelectContext(ctx, &entities, `
		SELECT entity_id, name, entity_type, similarity(name, $2) as similarity
		FROM financial_entities 
		WHERE tenant_id = $1 AND status = 'active' AND similarity(name, $2) > $3
		ORDER BY similarity DESC
		LIMIT 20
	`, tenantID, name, threshold)
	return entities, err
}

func (s *FKGService) createRelationshipRecord(ctx context.Context, relationshipID, tenantID, sourceEntityID, targetEntityID, relationshipType string, percentageOwnership, votingRights float64, effectiveDate string, propertiesJSON []byte) error {
	// TODO: Use HasuraClient for INSERT when available
	// For now, use SQL fallback for INSERT with date cast and JSONB
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO ownership_relationships (
relationship_id, tenant_id, source_entity_id, target_entity_id, 
relationship_type, percentage_ownership, voting_rights, 
effective_date, properties, created_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, NULLIF($8, '')::date, $9, NOW())
	`, relationshipID, tenantID, sourceEntityID, targetEntityID,
relationshipType, percentageOwnership, votingRights,
effectiveDate, propertiesJSON)
	return err
}

func (s *FKGService) getUBOChainRecords(ctx context.Context, tenantID, entityID string, maxDepth int) ([]OwnershipChain, error) {
	// TODO: Use HasuraClient for SELECT when available
	// For now, use SQL fallback for database function call
	var chain []OwnershipChain
	err := s.db.SelectContext(ctx, &chain, `
		SELECT entity_id, parent_entity_id, entity_name, entity_type, 
		       percentage_ownership, cumulative_ownership, depth, path
		FROM get_ubo_chain($1::uuid, $2::uuid, $3)
	`, tenantID, entityID, maxDepth)
	return chain, err
}

func (s *FKGService) hybridSearchDocumentsRecords(ctx context.Context, tenantID, query, embeddingStr string, limit int) ([]SearchResult, error) {
	// TODO: Use HasuraClient for SELECT when available
	// For now, use SQL fallback for hybrid search function with pgvector
	var results []SearchResult
	err := s.db.SelectContext(ctx, &results, `
		SELECT chunk_id, document_id, entity_id, content, combined_score, keyword_rank, semantic_rank
		FROM hybrid_search_documents($1::uuid, $2, $3::vector, $4)
	`, tenantID, query, embeddingStr, limit)
	return results, err
}

func (s *FKGService) keywordSearchDocumentsRecords(ctx context.Context, tenantID, query string, limit int) ([]SearchResult, error) {
	// TODO: Use HasuraClient for SELECT when available
	// For now, use SQL fallback for full-text search with ts_rank
	var results []SearchResult
	err := s.db.SelectContext(ctx, &results, `
		SELECT chunk_id, document_id, entity_id, content, 
		       ts_rank(to_tsvector('english', content), plainto_tsquery($2)) as combined_score,
		       1 as keyword_rank, 0 as semantic_rank
		FROM document_chunks 
		WHERE tenant_id = $1 AND to_tsvector('english', content) @@ plainto_tsquery($2)
		ORDER BY combined_score DESC
		LIMIT $3
	`, tenantID, query, limit)
	return results, err
}
