package household

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// HasuraClient defines the interface for Hasura GraphQL operations
type HasuraClient interface {
	Query(ctx context.Context, query string, variables map[string]interface{}, result interface{}) error
	Mutate(ctx context.Context, mutation string, variables map[string]interface{}, result interface{}) error
}

type Service struct {
	DB           *sqlx.DB
	hasuraClient HasuraClient
}

func NewService(db *sqlx.DB) *Service {
	return &Service{DB: db}
}

// NewServiceWithHasura creates a new household service with Hasura support
func NewServiceWithHasura(db *sqlx.DB, hasuraClient HasuraClient) *Service {
	return &Service{DB: db, hasuraClient: hasuraClient}
}

// CreateHousehold creates a new household
func (s *Service) CreateHousehold(ctx context.Context, name string) (*Household, error) {
	h := &Household{
		HouseholdID:   uuid.New(),
		HouseholdName: name,
		CreatedAt:     time.Now(),
	}
	err := s.createHouseholdRecord(ctx, h)
	if err != nil {
		return nil, fmt.Errorf("failed to create household: %w", err)
	}
	return h, nil
}

// CreateEntity creates a new entity within a household
func (s *Service) CreateEntity(ctx context.Context, entity *Entity) error {
	if entity.EntityID == uuid.Nil {
		entity.EntityID = uuid.New()
	}
	entity.CreatedAt = time.Now()
	entity.UpdatedAt = time.Now()

	err := s.createEntityRecord(ctx, entity)
	if err != nil {
		return fmt.Errorf("failed to create entity: %w", err)
	}
	return nil
}

// GetHouseholdEntities retrieves all entities for a household
func (s *Service) GetHouseholdEntities(ctx context.Context, householdID uuid.UUID) ([]Entity, error) {
	entities, err := s.getHouseholdEntitiesRecords(ctx, householdID)
	if err != nil {
		return nil, fmt.Errorf("failed to get entities: %w", err)
	}
	return entities, nil
}

// RecordTransfer records a transfer between entities
func (s *Service) RecordTransfer(ctx context.Context, transfer *InterEntityTransfer) error {
	if transfer.TransferID == uuid.Nil {
		transfer.TransferID = uuid.New()
	}
	transfer.CreatedAt = time.Now()

	err := s.recordTransferRecord(ctx, transfer)
	if err != nil {
		return fmt.Errorf("failed to record transfer: %w", err)
	}
	return nil
}

// GetHouseholdHierarchy returns a nested structure of entities (simplified for now)
func (s *Service) GetHouseholdHierarchy(ctx context.Context, householdID uuid.UUID) (map[string]interface{}, error) {
	entities, err := s.GetHouseholdEntities(ctx, householdID)
	if err != nil {
		return nil, err
	}

	// Build tree
	// This is a simplified representation. In a real app, we'd have a recursive struct.
	// For now, we just return the flat list which the frontend can parse into a tree.
	return map[string]interface{}{
		"household_id": householdID,
		"entities":     entities,
	}, nil
}

// Helper methods for SQL operations with Hasura fallback

// createHouseholdRecord inserts a new household
// TODO: Replace SQL with Hasura GraphQL mutation:
//
//	mutation InsertHousehold($object: households_insert_input!) {
//	  insert_households_one(object: $object) {
//	    household_id
//	    household_name
//	  }
//	}
//
// Variables: {"object": {"household_id": "...", "household_name": "...", "created_at": "..."}}
// Use: http://localhost:8080/v1/graphql with header X-Hasura-Admin-Secret: newadminsecretkey
func (s *Service) createHouseholdRecord(ctx context.Context, h *Household) error {
	query := "INSERT INTO households (household_id, household_name, created_at) VALUES ($1, $2, $3)"
	_, err := s.DB.ExecContext(ctx, query, h.HouseholdID, h.HouseholdName, h.CreatedAt)
	return err
}

// createEntityRecord inserts a new entity
// TODO: Implement Hasura GraphQL mutation
// SQL fallback: NamedExec INSERT for 14 Entity fields (trusts, foundations, LLCs)
func (s *Service) createEntityRecord(ctx context.Context, entity *Entity) error {
	query := `
		INSERT INTO entities (
entity_id, entity_type, entity_name, tax_id, parent_entity_id, household_id,
trust_type, trust_termination_date, foundation_type, annual_distribution_requirement,
ownership_structure, operating_agreement_url, created_at, updated_at
) VALUES (
:entity_id, :entity_type, :entity_name, :tax_id, :parent_entity_id, :household_id,
:trust_type, :trust_termination_date, :foundation_type, :annual_distribution_requirement,
:ownership_structure, :operating_agreement_url, :created_at, :updated_at
)`

	_, err := s.DB.NamedExecContext(ctx, query, entity)
	return err
}

// getHouseholdEntitiesRecords retrieves all entities for a household
// TODO: Replace SQL with Hasura GraphQL query:
//
//	query GetHouseholdEntities($householdId: uuid!) {
//	  entities(where: {household_id: {_eq: $householdId}}) {
//	    entity_id
//	    entity_type
//	    entity_name
//	    tax_id
//	    parent_entity_id
//	    household_id
//	    trust_type
//	    trust_termination_date
//	    foundation_type
//	    annual_distribution_requirement
//	    ownership_structure
//	    operating_agreement_url
//	    created_at
//	    updated_at
//	  }
//	}
//
// Use: http://localhost:8080/v1/graphql with header X-Hasura-Admin-Secret: newadminsecretkey
func (s *Service) getHouseholdEntitiesRecords(ctx context.Context, householdID uuid.UUID) ([]Entity, error) {
	var entities []Entity
	query := "SELECT * FROM entities WHERE household_id = $1"
	err := s.DB.SelectContext(ctx, &entities, query, householdID)
	return entities, err
}

// recordTransferRecord inserts an inter-entity transfer
// TODO: Implement Hasura GraphQL mutation
// SQL fallback: NamedExec INSERT for 11 transfer fields with gift tax tracking
func (s *Service) recordTransferRecord(ctx context.Context, transfer *InterEntityTransfer) error {
	query := `
		INSERT INTO inter_entity_transfers (
transfer_id, from_entity_id, to_entity_id, transfer_date, amount,
asset_description, transfer_reason, gift_tax_return_required,
generation_skipping_transfer, advisor_notes, created_at
) VALUES (
:transfer_id, :from_entity_id, :to_entity_id, :transfer_date, :amount,
:asset_description, :transfer_reason, :gift_tax_return_required,
:generation_skipping_transfer, :advisor_notes, :created_at
)`

	_, err := s.DB.NamedExecContext(ctx, query, transfer)
	return err
}
