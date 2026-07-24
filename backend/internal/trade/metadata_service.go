package trade

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/google/uuid"
)

// HasuraClient defines the interface for Hasura GraphQL operations
type HasuraClient interface {
	Query(ctx context.Context, query string, variables map[string]interface{}, result interface{}) error
	Mutate(ctx context.Context, mutation string, variables map[string]interface{}, result interface{}) error
}

type MetadataService struct {
	DB           *sql.DB
	hasuraClient HasuraClient
}

func NewMetadataService(db *sql.DB) *MetadataService {
	return &MetadataService{DB: db}
}

// NewMetadataServiceWithHasura creates a new trade metadata service with Hasura support
func NewMetadataServiceWithHasura(db *sql.DB, hasuraClient HasuraClient) *MetadataService {
	return &MetadataService{DB: db, hasuraClient: hasuraClient}
}

// GetWorkflowDefinition retrieves a workflow definition by name and tenant
func (s *MetadataService) GetWorkflowDefinition(tenantID string, name string) (*WorkflowDefinition, error) {
	return s.getWorkflowDefinitionRecord(context.Background(), tenantID, name)
}

// CreateWorkflowDefinition creates a new workflow definition
func (s *MetadataService) CreateWorkflowDefinition(wd *WorkflowDefinition) error {
	return s.createWorkflowDefinitionRecord(context.Background(), wd)
}

// GetWorkflowStages retrieves stages for a workflow
func (s *MetadataService) GetWorkflowStages(workflowID uuid.UUID) ([]WorkflowStage, error) {
	return s.getWorkflowStagesRecords(context.Background(), workflowID)
}

// CreateWorkflowStage adds a stage to a workflow
func (s *MetadataService) CreateWorkflowStage(stage *WorkflowStage) error {
	return s.createWorkflowStageRecord(context.Background(), stage)
}

// GetBusinessObjects retrieves business objects for a tenant (reusing existing table)
func (s *MetadataService) GetBusinessObjects(tenantID string) ([]map[string]interface{}, error) {
	return s.getBusinessObjectsRecords(context.Background(), tenantID)
}

// Helper methods for SQL operations with Hasura fallback

// getWorkflowDefinitionRecord retrieves a workflow definition by name and tenant
// TODO: Migrate to Hasura GraphQL query:
//
//	query GetWorkflowDefinition($tenant_id: String!, $name: String!) {
//	  workflow_definitions(where: {tenant_id: {_eq: $tenant_id}, name: {_eq: $name}}, limit: 1) {
//	    id
//	    tenant_id
//	    name
//	    description
//	    status
//	    stages
//	    created_at
//	    created_by
//	    last_modified_at
//	    last_modified_by
//	  }
//	}
//
// Note: JSONB stages field with workflow configuration
func (s *MetadataService) getWorkflowDefinitionRecord(ctx context.Context, tenantID string, name string) (*WorkflowDefinition, error) {
	// Use SQL for json.RawMessage field handling
	var wd WorkflowDefinition
	var stagesJSON []byte

	query := `
		SELECT id, tenant_id, name, description, status, stages, created_at, created_by, last_modified_at, last_modified_by
		FROM workflow_definitions
		WHERE tenant_id = $1 AND name = $2
	`
	err := s.DB.QueryRow(query, tenantID, name).Scan(
		&wd.ID, &wd.TenantID, &wd.Name, &wd.Description, &wd.Status, &stagesJSON,
		&wd.CreatedAt, &wd.CreatedBy, &wd.LastModifiedAt, &wd.LastModifiedBy,
	)
	if err != nil {
		return nil, err
	}
	wd.Stages = json.RawMessage(stagesJSON)
	return &wd, nil
}

// createWorkflowDefinitionRecord creates a new workflow definition
// TODO: Migrate to Hasura GraphQL mutation:
//
//	mutation CreateWorkflowDefinition($object: workflow_definitions_insert_input!) {
//	  insert_workflow_definitions_one(object: $object) {
//	    id
//	    tenant_id
//	    name
//	    description
//	    status
//	    stages
//	    created_at
//	    created_by
//	    last_modified_at
//	    last_modified_by
//	  }
//	}
func (s *MetadataService) createWorkflowDefinitionRecord(ctx context.Context, wd *WorkflowDefinition) error {
	// Use SQL for json.RawMessage field handling
	if wd.ID == uuid.Nil {
		wd.ID = uuid.New()
	}

	stagesJSON, err := json.Marshal(wd.Stages)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO workflow_definitions (id, tenant_id, name, description, status, stages, created_by, last_modified_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err = s.DB.Exec(query, wd.ID, wd.TenantID, wd.Name, wd.Description, wd.Status, stagesJSON, wd.CreatedBy, wd.LastModifiedBy)
	return err
}

// getWorkflowStagesRecords retrieves stages for a workflow
// TODO: Migrate to Hasura GraphQL query:
//
//	query GetWorkflowStages($workflow_id: uuid!) {
//	  workflow_stages(where: {workflow_id: {_eq: $workflow_id}}, order_by: {order_index: asc}) {
//	    id
//	    workflow_id
//	    name
//	    order_index
//	    config_json
//	    created_at
//	  }
//	}
func (s *MetadataService) getWorkflowStagesRecords(ctx context.Context, workflowID uuid.UUID) ([]WorkflowStage, error) {
	// Use SQL for json.RawMessage field handling and ORDER BY
	query := `
		SELECT id, workflow_id, name, order_index, config_json, created_at
		FROM workflow_stages
		WHERE workflow_id = $1
		ORDER BY order_index ASC
	`
	rows, err := s.DB.Query(query, workflowID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stages []WorkflowStage
	for rows.Next() {
		var stage WorkflowStage
		var configJSON []byte
		if err := rows.Scan(&stage.ID, &stage.WorkflowID, &stage.Name, &stage.OrderIndex, &configJSON, &stage.CreatedAt); err != nil {
			return nil, err
		}
		stage.Config = json.RawMessage(configJSON)
		stages = append(stages, stage)
	}
	return stages, nil
}

// createWorkflowStageRecord adds a stage to a workflow
// TODO: Migrate to Hasura GraphQL mutation:
//
//	mutation CreateWorkflowStage($object: workflow_stages_insert_input!) {
//	  insert_workflow_stages_one(object: $object) {
//	    id
//	    workflow_id
//	    name
//	    order_index
//	    config_json
//	    created_at
//	  }
//	}
func (s *MetadataService) createWorkflowStageRecord(ctx context.Context, stage *WorkflowStage) error {
	// Use SQL for json.RawMessage field handling
	if stage.ID == uuid.Nil {
		stage.ID = uuid.New()
	}

	configJSON, err := json.Marshal(stage.Config)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO workflow_stages (id, workflow_id, name, order_index, config_json)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err = s.DB.Exec(query, stage.ID, stage.WorkflowID, stage.Name, stage.OrderIndex, configJSON)
	return err
}

// getBusinessObjectsRecords retrieves business objects for a tenant
// TODO: Migrate to Hasura GraphQL query:
//
//	query GetBusinessObjects($tenant_id: String!) {
//	  business_objects(where: {tenant_id: {_eq: $tenant_id}}) {
//	    id
//	    name
//	    display_name
//	    description
//	  }
//	}
func (s *MetadataService) getBusinessObjectsRecords(ctx context.Context, tenantID string) ([]map[string]interface{}, error) {
	// Use SQL for map[string]interface{} result type
	query := `
		SELECT id, name, display_name, description
		FROM business_objects
		WHERE tenant_id = $1
	`
	rows, err := s.DB.Query(query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var objects []map[string]interface{}
	for rows.Next() {
		var id, name, displayName, description string
		if err := rows.Scan(&id, &name, &displayName, &description); err != nil {
			return nil, err
		}
		objects = append(objects, map[string]interface{}{
			"id":           id,
			"name":         name,
			"display_name": displayName,
			"description":  description,
		})
	}
	return objects, nil
}
