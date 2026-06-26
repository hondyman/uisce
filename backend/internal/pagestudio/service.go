package pagestudio

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/semantic"
)

// VersionStore interface to avoid circular dependency
type VersionStore interface {
	SaveObject(ctx context.Context, obj semantic.SemanticObject, actor string) error
	GetVersion(ctx context.Context, id string, version int) (*semantic.SemanticObject, error)
}

// Service orchestrates Page Studio operations and governance
type Service struct {
	repo     *Repository
	versions VersionStore
}

// NewService creates a new Page Studio service
func NewService(repo *Repository, versions VersionStore) *Service {
	return &Service{
		repo:     repo,
		versions: versions,
	}
}

// SavePage saves a core page and registers it for governance
func (s *Service) SavePage(ctx context.Context, p *CorePage, actor string) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}

	if err := s.repo.SavePage(ctx, p); err != nil {
		return err
	}

	// Compute Semantic Fingerprint
	if fp, err := ComputeFingerprint(p); err == nil {
		p.SemanticFingerprint, _ = json.Marshal(fp)
		// Update page with fingerprint (in a real DB we'd update this too, assuming repo does upsert)
		// Re-save/update repo if needed, but for now we trust it's in p for the VersionStore
		s.repo.SavePage(ctx, p)
	}

	payload, _ := json.Marshal(p)
	semObj := semantic.SemanticObject{
		ID:        p.ID.String(),
		Env:       p.Env,
		TenantID:  p.TenantID,
		Type:      "page_core",
		Payload:   payload,
		Version:   p.Version,
		CreatedBy: actor,
	}

	return s.versions.SaveObject(ctx, semObj, actor)
}

// SaveOverlay saves a page overlay and registers it for governance
func (s *Service) SaveOverlay(ctx context.Context, o *PageOverlay, actor string) error {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}

	if err := s.repo.SaveOverlay(ctx, o); err != nil {
		return err
	}

	payload, _ := json.Marshal(o)
	semObj := semantic.SemanticObject{
		ID:        o.ID.String(),
		Env:       o.Env,
		TenantID:  &o.TenantID,
		Type:      "page_overlay",
		Payload:   payload,
		Version:   o.Version,
		CreatedBy: actor,
	}

	return s.versions.SaveObject(ctx, semObj, actor)
}

// GetRepository returns the underlying repository
func (s *Service) GetRepository() *Repository {
	return s.repo
}

// GenerateLayout generates a page layout using AI/heuristics based on BO metadata
func (s *Service) GenerateLayout(ctx context.Context, req AIGenerateRequest) (*AIGenerateResponse, error) {
	// 1. Fetch BO from VersionStore
	bo, err := s.versions.GetVersion(ctx, req.BOName, -1)
	if err != nil {
		return nil, err
	}

	// 2. Call AI Generator (Mock for now)
	return GenerateAILayout(bo, req.Intent)
}

// ApplyUpgradeDecision finalizes a reconciliation by updating the overlay and impact record
func (s *Service) ApplyUpgradeDecision(ctx context.Context, impactID uuid.UUID, decisions map[string]string, actor string) error {
	impact, err := s.repo.GetUpgradeImpact(ctx, impactID)
	if err != nil {
		return err
	}

	// Fetch overlay
	// In a real system, we'd use the env from the impact or a session context
	overlay, err := s.repo.GetOverlay(ctx, impact.CorePageID, impact.TenantID, "production")
	if err != nil {
		return err
	}

	// (Simplified) Mutation logic based on decisions
	// For production, we'd iterate through decisions and modify overlay.Overrides JSON

	impact.Status = UpgradeStatusAccepted
	if err := s.repo.SaveUpgradeImpact(ctx, impact); err != nil {
		return err
	}

	overlay.Version++
	return s.SaveOverlay(ctx, overlay, actor)
}
