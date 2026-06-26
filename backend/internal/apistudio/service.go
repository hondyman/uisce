package apistudio

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/semantic"
)

// VersionStore interface to avoid circular dependency
type VersionStore interface {
	SaveObject(ctx context.Context, obj semantic.SemanticObject, actor string) error
}

// DesignAI defines the interface for AI-driven endpoint creation
type DesignAI interface {
	ProposeEndpoint(ctx context.Context, prompt string, tenantID string) (any, error)
}

// Service orchestrates API logical operations and governance
type Service struct {
	repo     *Repository
	versions VersionStore
	ai       DesignAI
}

// NewService creates a new API Studio service
func NewService(repo *Repository, versions VersionStore, ai DesignAI) *Service {
	return &Service{
		repo:     repo,
		versions: versions,
		ai:       ai,
	}
}

// SaveEndpoint saves the endpoint and registers it as a semantic object for governance
func (s *Service) SaveEndpoint(ctx context.Context, ep *APIEndpoint, actor string) error {
	// 1. Save to specific table for runtime lookups
	if err := s.repo.SaveEndpoint(ctx, ep); err != nil {
		return err
	}

	// 2. Wrap in SemanticObject for governance (versioning, diff, etc.)
	payload, _ := json.Marshal(ep)
	tenantID := ep.TenantID
	semObj := semantic.SemanticObject{
		ID:        ep.ID.String(),
		Env:       ep.Env,
		TenantID:  &tenantID,
		Type:      "api_endpoint",
		Payload:   payload,
		Version:   ep.Version,
		CreatedBy: actor,
	}

	return s.versions.SaveObject(ctx, semObj, actor)
}

// DeprecateEndpoint marks an endpoint as deprecated
func (s *Service) DeprecateEndpoint(ctx context.Context, id string, actor string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	ep, err := s.repo.GetEndpoint(ctx, uid)
	if err != nil {
		return err
	}

	now := time.Now()
	ep.Status = "deprecated"
	ep.DeprecatedAt = &now

	return s.SaveEndpoint(ctx, ep, actor)
}

// AIGenerateEndpoint generates a proposal for a new endpoint
func (s *Service) AIGenerateEndpoint(ctx context.Context, prompt string, tenantID string) (*APIEndpoint, error) {
	proposal, err := s.ai.ProposeEndpoint(ctx, prompt, tenantID)
	if err != nil {
		return nil, err
	}

	// Marshall/Unmarshal to convert interface{} to APIEndpoint
	// This handles the type mapping without a direct dependency in the NL service
	b, err := json.Marshal(proposal)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal AI proposal: %w", err)
	}

	var ep APIEndpoint
	if err := json.Unmarshal(b, &ep); err != nil {
		return nil, fmt.Errorf("failed to unmarshal AI proposal into APIEndpoint: %w", err)
	}

	// Set defaults/IDs
	ep.ID = uuid.New()
	ep.CreatedAt = time.Now()
	ep.TenantID = tenantID

	return &ep, nil
}

// RetireEndpoint marks an endpoint as retired
func (s *Service) RetireEndpoint(ctx context.Context, id string, actor string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	ep, err := s.repo.GetEndpoint(ctx, uid)
	if err != nil {
		return err
	}

	now := time.Now()
	ep.Status = "retired"
	ep.RetiredAt = &now

	return s.SaveEndpoint(ctx, ep, actor)
}

// GetRepository returns the underlying repository
func (s *Service) GetRepository() *Repository {
	return s.repo
}
