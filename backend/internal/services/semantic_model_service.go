package services

import (
	"context"
	"fmt"

	hasuraclient "github.com/hondyman/semlayer/libs/hasura-client"
	"go.uber.org/zap"
)

// ============================================================================
// SEMANTIC MODEL INHERITANCE SERVICE
// Workday-style Core → Custom model inheritance
// ============================================================================

// SemanticModelInheritanceService manages semantic cubes with BO sync
type SemanticModelInheritanceService struct {
	hasuraClient *hasuraclient.HasuraClient
	logger       *zap.Logger
}

// NewSemanticModelInheritanceService creates a new semantic model inheritance service
func NewSemanticModelInheritanceService(hasuraClient *hasuraclient.HasuraClient) *SemanticModelInheritanceService {
	logger, _ := zap.NewProduction()
	return &SemanticModelInheritanceService{
		hasuraClient: hasuraClient,
		logger:       logger,
	}
}

// ModelType defines semantic model types
type ModelType string

const (
	ModelTypeCore     ModelType = "core"
	ModelTypeCustom   ModelType = "custom"
	ModelTypeOverride ModelType = "override"
)

// SemanticModel represents a semantic cube with inheritance
type SemanticModel struct {
	ID               string    `json:"id"`
	TenantID         string    `json:"tenant_id"`
	Name             string    `json:"name"`
	Label            string    `json:"label"`
	Description      string    `json:"description,omitempty"`
	ModelType        ModelType `json:"model_type"`
	SourceCubeID     string    `json:"source_cube_id,omitempty"`
	BusinessObjectID string    `json:"business_object_id,omitempty"`
	IsSystem         bool      `json:"is_system"`
	Status           string    `json:"status"`
}

// SemanticDimension represents a cube dimension
type SemanticDimension struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Label        string `json:"label"`
	SQL          string `json:"sql"`
	Type         string `json:"type"`
	IsInherited  bool   `json:"is_inherited"`
	IsOverridden bool   `json:"is_overridden"`
}

// SemanticMeasure represents a cube measure
type SemanticMeasure struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Label        string `json:"label"`
	SQL          string `json:"sql"`
	Type         string `json:"type"`
	IsInherited  bool   `json:"is_inherited"`
	IsOverridden bool   `json:"is_overridden"`
}

// ============================================================================
// CORE MODEL OPERATIONS
// ============================================================================

// GetCoreModels returns all core semantic models (templates)
func (s *SemanticModelInheritanceService) GetCoreModels(ctx context.Context) ([]*SemanticModel, error) {
	query := `
		query GetCoreModels {
			semantic_cubes_v2(where: { model_type: { _eq: "core" }, is_system: { _eq: true } }) {
				id
				name
				label
				description
				model_type
				source_cube_id
				business_object_id
				status
			}
		}
	`

	result, err := s.hasuraClient.Query(query, nil)
	if err != nil {
		return nil, err
	}

	return s.parseModels(result, "semantic_cubes_v2")
}

// ============================================================================
// TENANT MODEL OPERATIONS
// ============================================================================

// ProvisionTenantModel creates a custom model for a tenant from a core template
func (s *SemanticModelInheritanceService) ProvisionTenantModel(ctx context.Context, tenantID, coreCubeID string, datasourceID *string) (string, error) {
	query := `
		query ProvisionModel($tenantID: uuid!, $coreCubeID: uuid!, $datasourceID: uuid) {
			provision_tenant_semantic_model(args: {
				p_tenant_id: $tenantID,
				p_core_cube_id: $coreCubeID,
				p_datasource_id: $datasourceID
			})
		}
	`

	vars := map[string]interface{}{
		"tenantID":   tenantID,
		"coreCubeID": coreCubeID,
	}
	if datasourceID != nil {
		vars["datasourceID"] = *datasourceID
	}

	result, err := s.hasuraClient.Query(query, vars)
	if err != nil {
		return "", err
	}

	if id, ok := result["provision_tenant_semantic_model"].(string); ok {
		return id, nil
	}

	return "", fmt.Errorf("failed to provision model")
}

// GetTenantModels returns all custom models for a tenant
func (s *SemanticModelInheritanceService) GetTenantModels(ctx context.Context, tenantID string) ([]*SemanticModel, error) {
	query := `
		query GetTenantModels($tenantID: uuid!) {
			semantic_cubes_v2(where: { 
				tenant_id: { _eq: $tenantID },
				model_type: { _in: ["custom", "override"] }
			}) {
				id
				name
				label
				description
				model_type
				source_cube_id
				business_object_id
				is_system
				status
			}
		}
	`

	result, err := s.hasuraClient.Query(query, map[string]interface{}{"tenantID": tenantID})
	if err != nil {
		return nil, err
	}

	return s.parseModels(result, "semantic_cubes_v2")
}

// GetModelWithInheritance returns a model with resolved inheritance
func (s *SemanticModelInheritanceService) GetModelWithInheritance(ctx context.Context, cubeID string) (*SemanticModel, []*SemanticDimension, []*SemanticMeasure, error) {
	query := `
		query GetModelFull($cubeID: uuid!) {
			semantic_cubes_v2_by_pk(id: $cubeID) {
				id
				name
				label
				description
				model_type
				source_cube_id
				business_object_id
				is_system
				status
				cube_dimensions_v2 {
					id
					name
					label
					sql
					type
					is_inherited
					is_overridden
				}
				cube_measures_v2 {
					id
					name
					label
					sql
					type
					is_inherited
					is_overridden
				}
			}
		}
	`

	result, err := s.hasuraClient.Query(query, map[string]interface{}{"cubeID": cubeID})
	if err != nil {
		return nil, nil, nil, err
	}

	data, ok := result["semantic_cubes_v2_by_pk"].(map[string]interface{})
	if !ok {
		return nil, nil, nil, fmt.Errorf("model not found")
	}

	model := &SemanticModel{
		ID:               getString(data, "id"),
		Name:             getString(data, "name"),
		Label:            getString(data, "label"),
		Description:      getString(data, "description"),
		ModelType:        ModelType(getString(data, "model_type")),
		SourceCubeID:     getString(data, "source_cube_id"),
		BusinessObjectID: getString(data, "business_object_id"),
		IsSystem:         getBool(data, "is_system"),
		Status:           getString(data, "status"),
	}

	dimensions := []*SemanticDimension{}
	if dims, ok := data["cube_dimensions_v2"].([]interface{}); ok {
		for _, d := range dims {
			dim := d.(map[string]interface{})
			dimensions = append(dimensions, &SemanticDimension{
				ID:           getString(dim, "id"),
				Name:         getString(dim, "name"),
				Label:        getString(dim, "label"),
				SQL:          getString(dim, "sql"),
				Type:         getString(dim, "type"),
				IsInherited:  getBool(dim, "is_inherited"),
				IsOverridden: getBool(dim, "is_overridden"),
			})
		}
	}

	measures := []*SemanticMeasure{}
	if msrs, ok := data["cube_measures_v2"].([]interface{}); ok {
		for _, m := range msrs {
			msr := m.(map[string]interface{})
			measures = append(measures, &SemanticMeasure{
				ID:           getString(msr, "id"),
				Name:         getString(msr, "name"),
				Label:        getString(msr, "label"),
				SQL:          getString(msr, "sql"),
				Type:         getString(msr, "type"),
				IsInherited:  getBool(msr, "is_inherited"),
				IsOverridden: getBool(msr, "is_overridden"),
			})
		}
	}

	return model, dimensions, measures, nil
}

// ============================================================================
// SYNC WITH BUSINESS OBJECTS
// ============================================================================

// SyncModelWithBO synchronizes a semantic model with its linked business object
func (s *SemanticModelInheritanceService) SyncModelWithBO(ctx context.Context, cubeID string) (int, error) {
	query := `
		query SyncModel($cubeID: uuid!) {
			sync_semantic_model_with_bo(args: { p_cube_id: $cubeID })
		}
	`

	result, err := s.hasuraClient.Query(query, map[string]interface{}{"cubeID": cubeID})
	if err != nil {
		return 0, err
	}

	if count, ok := result["sync_semantic_model_with_bo"].(float64); ok {
		return int(count), nil
	}

	return 0, nil
}

// GetModelForBO returns the semantic model linked to a business object
func (s *SemanticModelInheritanceService) GetModelForBO(ctx context.Context, tenantID, boID string) (*SemanticModel, error) {
	query := `
		query GetModelForBO($tenantID: uuid!, $boID: uuid!) {
			semantic_cubes_v2(
				where: {
					tenant_id: { _eq: $tenantID },
					business_object_id: { _eq: $boID },
					model_type: { _neq: "core" }
				},
				limit: 1
			) {
				id
				name
				label
				model_type
				source_cube_id
			}
		}
	`

	result, err := s.hasuraClient.Query(query, map[string]interface{}{
		"tenantID": tenantID,
		"boID":     boID,
	})
	if err != nil {
		return nil, err
	}

	models, err := s.parseModels(result, "semantic_cubes_v2")
	if err != nil || len(models) == 0 {
		return nil, nil
	}

	return models[0], nil
}

// ============================================================================
// DIMENSION/MEASURE CUSTOMIZATION
// ============================================================================

// AddCustomDimension adds a tenant-specific dimension (marks as not inherited)
func (s *SemanticModelInheritanceService) AddCustomDimension(ctx context.Context, cubeID string, dim SemanticDimension) (string, error) {
	mutation := `
		mutation AddDimension($object: cube_dimensions_v2_insert_input!) {
			insert_cube_dimensions_v2_one(object: $object) {
				id
			}
		}
	`

	object := map[string]interface{}{
		"cube_id":      cubeID,
		"name":         dim.Name,
		"label":        dim.Label,
		"sql":          dim.SQL,
		"type":         dim.Type,
		"is_inherited": false,
	}

	result, err := s.hasuraClient.Mutate(mutation, map[string]interface{}{"object": object})
	if err != nil {
		return "", err
	}

	data := result["insert_cube_dimensions_v2_one"].(map[string]interface{})
	return data["id"].(string), nil
}

// OverrideDimension overrides an inherited dimension with custom values
func (s *SemanticModelInheritanceService) OverrideDimension(ctx context.Context, dimID string, newSQL, newLabel string) error {
	mutation := `
		mutation OverrideDimension($id: uuid!, $sql: String!, $label: String!) {
			update_cube_dimensions_v2_by_pk(
				pk_columns: { id: $id },
				_set: { sql: $sql, label: $label, is_overridden: true }
			) {
				id
			}
		}
	`

	_, err := s.hasuraClient.Mutate(mutation, map[string]interface{}{
		"id":    dimID,
		"sql":   newSQL,
		"label": newLabel,
	})

	return err
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func (s *SemanticModelInheritanceService) parseModels(result map[string]interface{}, key string) ([]*SemanticModel, error) {
	items, ok := result[key].([]interface{})
	if !ok {
		return []*SemanticModel{}, nil
	}

	models := make([]*SemanticModel, 0, len(items))
	for _, item := range items {
		data := item.(map[string]interface{})
		models = append(models, &SemanticModel{
			ID:               getString(data, "id"),
			Name:             getString(data, "name"),
			Label:            getString(data, "label"),
			Description:      getString(data, "description"),
			ModelType:        ModelType(getString(data, "model_type")),
			SourceCubeID:     getString(data, "source_cube_id"),
			BusinessObjectID: getString(data, "business_object_id"),
			IsSystem:         getBool(data, "is_system"),
			Status:           getString(data, "status"),
		})
	}

	return models, nil
}
