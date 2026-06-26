package population

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.temporal.io/sdk/activity"
)

// Shared types
type ExtractedEntity struct {
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Identifier string                 `json:"identifier"`
	Attributes map[string]interface{} `json:"attributes"`
}

type LinkedEntity struct {
	ExtractedEntity
	CanonicalID uuid.UUID `json:"canonical_id"`
}

type PersistNodesInput struct {
	TenantID uuid.UUID      `json:"tenant_id"`
	Entities []LinkedEntity `json:"entities"`
}

type NodeCreationStats struct {
	NodesCreated int `json:"nodes_created"`
}

type PersistRelationshipsInput struct {
	TenantID uuid.UUID      `json:"tenant_id"`
	Entities []LinkedEntity `json:"entities"`
}

type RelationshipStats struct {
	RelationshipsCreated int `json:"relationships_created"`
}

// Activities
type PopulationActivities struct {
	DB *sqlx.DB
}

func NewPopulationActivities(db *sqlx.DB) *PopulationActivities {
	return &PopulationActivities{DB: db}
}

func (a *PopulationActivities) ExtractEntitiesActivity(ctx context.Context, tenantID uuid.UUID) ([]ExtractedEntity, error) {
	// Stub implementation - in reality this would call the NER service
	return []ExtractedEntity{
		{Name: "Apple Inc.", Type: "CORP", Identifier: "US0378331005", Attributes: map[string]interface{}{"ticker": "AAPL"}},
		{Name: "Vanguard 500 Index Fund", Type: "FUND", Identifier: "US9229087696", Attributes: map[string]interface{}{"ticker": "VFIAX"}},
	}, nil
}

func (a *PopulationActivities) DeduplicateEntitiesActivity(ctx context.Context, entities []ExtractedEntity) ([]LinkedEntity, error) {
	// Stub implementation - pass through with new UUIDs
	var linked []LinkedEntity
	for _, e := range entities {
		linked = append(linked, LinkedEntity{
			ExtractedEntity: e,
			CanonicalID:     uuid.New(),
		})
	}
	return linked, nil
}

func (a *PopulationActivities) PersistNodesPostgresActivity(ctx context.Context, input PersistNodesInput) (*NodeCreationStats, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Persisting nodes to Postgres", "count", len(input.Entities))

	tx, err := a.DB.Beginx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	count := 0
	for _, entity := range input.Entities {
		propsJSON, _ := json.Marshal(entity.Attributes)

		// Upsert logic based on canonical_id or identifier
		// TODO: Replace SQL with Hasura GraphQL mutation (upsert):
		// mutation UpsertFinancialEntity($object: financial_entities_insert_input!) {
		//   insert_financial_entities_one(
		//     object: $object,
		//     on_conflict: {
		//       constraint: financial_entities_canonical_id_key,
		//       update_columns: [properties, updated_at]
		//     }
		//   ) {
		//     entity_id
		//     canonical_id
		//   }
		// }
		// Variables: {"object": {"entity_id": "...", "canonical_id": "...", "entity_type": "CORP",
		//   "name": "...", "properties": {...}}}
		// Note: Use _append for JSONB merge or custom SQL function for properties merge
		// Use: http://localhost:8080/v1/graphql with header X-Hasura-Admin-Secret: newadminsecretkey
		query := `
			INSERT INTO financial_entities (entity_id, canonical_id, entity_type, name, properties)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (canonical_id) DO UPDATE 
			SET properties = financial_entities.properties || EXCLUDED.properties,
			    updated_at = NOW();
		`
		_, err := tx.ExecContext(ctx, query,
			entity.CanonicalID,
			entity.Identifier,
			entity.Type,
			entity.Name,
			propsJSON,
		)
		if err != nil {
			logger.Error("Failed to insert entity", "error", err)
			continue
		}
		count++
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &NodeCreationStats{NodesCreated: count}, nil
}

func (a *PopulationActivities) PersistRelationshipsPostgresActivity(ctx context.Context, input PersistRelationshipsInput) (*RelationshipStats, error) {
	// Stub implementation - would insert into ownership_relationships
	return &RelationshipStats{RelationshipsCreated: 0}, nil
}

// Package-level activity name constants for Temporal workflow registration.
// These are used as activity identifiers in workflow definitions.
var (
	ExtractEntitiesActivity              = "ExtractEntitiesActivity"
	DeduplicateEntitiesActivity          = "DeduplicateEntitiesActivity"
	PersistNodesPostgresActivity         = "PersistNodesPostgresActivity"
	PersistRelationshipsPostgresActivity = "PersistRelationshipsPostgresActivity"
)
