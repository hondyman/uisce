package services

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

type SemanticModelInheritanceService struct {
	logger *zap.Logger
}

func NewSemanticModelInheritanceService() *SemanticModelInheritanceService {
	logger, _ := zap.NewProduction()
	return &SemanticModelInheritanceService{
		logger: logger,
	}
}

type ModelType string

const (
	ModelTypeCore     ModelType = "core"
	ModelTypeCustom   ModelType = "custom"
	ModelTypeOverride ModelType = "override"
)

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

type SemanticDimension struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Label        string `json:"label"`
	SQL          string `json:"sql"`
	Type         string `json:"type"`
	IsInherited  bool   `json:"is_inherited"`
	IsOverridden bool   `json:"is_overridden"`
}

type SemanticMeasure struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Label        string `json:"label"`
	SQL          string `json:"sql"`
	Type         string `json:"type"`
	IsInherited  bool   `json:"is_inherited"`
	IsOverridden bool   `json:"is_overridden"`
}

func (s *SemanticModelInheritanceService) GetCoreModels(ctx context.Context) ([]*SemanticModel, error) {
	return nil, fmt.Errorf("GetCoreModels: Hasura removed from SemanticModelInheritanceService")
}

func (s *SemanticModelInheritanceService) ProvisionTenantModel(ctx context.Context, tenantID, coreCubeID string, datasourceID *string) (string, error) {
	return "", fmt.Errorf("ProvisionTenantModel: Hasura removed from SemanticModelInheritanceService")
}

func (s *SemanticModelInheritanceService) GetTenantModels(ctx context.Context, tenantID string) ([]*SemanticModel, error) {
	return nil, fmt.Errorf("GetTenantModels: Hasura removed from SemanticModelInheritanceService")
}

func (s *SemanticModelInheritanceService) GetModelWithInheritance(ctx context.Context, cubeID string) (*SemanticModel, []*SemanticDimension, []*SemanticMeasure, error) {
	return nil, nil, nil, fmt.Errorf("GetModelWithInheritance: Hasura removed from SemanticModelInheritanceService")
}

func (s *SemanticModelInheritanceService) SyncModelWithBO(ctx context.Context, cubeID string) (int, error) {
	return 0, fmt.Errorf("SyncModelWithBO: Hasura removed from SemanticModelInheritanceService")
}

func (s *SemanticModelInheritanceService) GetModelForBO(ctx context.Context, tenantID, boID string) (*SemanticModel, error) {
	return nil, fmt.Errorf("GetModelForBO: Hasura removed from SemanticModelInheritanceService")
}

func (s *SemanticModelInheritanceService) AddCustomDimension(ctx context.Context, cubeID string, dim SemanticDimension) (string, error) {
	return "", fmt.Errorf("AddCustomDimension: Hasura removed from SemanticModelInheritanceService")
}

func (s *SemanticModelInheritanceService) OverrideDimension(ctx context.Context, dimID string, newSQL, newLabel string) error {
	return fmt.Errorf("OverrideDimension: Hasura removed from SemanticModelInheritanceService")
}
