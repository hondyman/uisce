package preference

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

// SourceRanking holds aggregated ranking stats for one source system
type SourceRanking struct {
	SourceSystem           string  `json:"source_system"`
	FirstPreferenceCount   int     `json:"first_preference_count"`
	SecondPreferenceCount  int     `json:"second_preference_count"`
	ThirdPreferenceCount   int     `json:"third_preference_count"`
	OtherPreferenceCount   int     `json:"other_preference_count"`
	TotalSelections        int     `json:"total_selections"`
	FirstPreferencePercent float64 `json:"first_preference_percent"`
	AvgConfidence          float64 `json:"avg_confidence"`
}

// AnalyticsReport is the full analytics response for a query
type AnalyticsReport struct {
	Rankings       []SourceRanking `json:"rankings"`
	BusinessObject string          `json:"business_object,omitempty"`
	SemanticTerm   string          `json:"semantic_term,omitempty"`
	Region         string          `json:"region,omitempty"`
	GeneratedAt    time.Time       `json:"generated_at"`
}

// Service orchestrates source preference management business logic
type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// CreatePreference creates a new source preference record in draft status
func (s *Service) CreatePreference(ctx context.Context, p *SourcePreference) (*SourcePreference, error) {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	p.Status = "draft"
	p.Version = 1
	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()
	if p.ValidFrom.IsZero() {
		p.ValidFrom = time.Now()
	}
	if err := s.repo.CreatePreference(ctx, p); err != nil {
		return nil, fmt.Errorf("create preference: %w", err)
	}
	// Record initial version in audit trail
	impactJSON, _ := p.ImpactAnalysis.JSON()
	_ = s.repo.AppendVersion(ctx, p.ID, p.Status, "Initial creation", impactJSON, p.CreatedBy)
	return p, nil
}

// GetPreference retrieves a single preference by ID
func (s *Service) GetPreference(ctx context.Context, id uuid.UUID) (*SourcePreference, error) {
	return s.repo.GetPreference(ctx, id)
}

// ListPreferences returns all preferences for a tenant, optionally filtered
func (s *Service) ListPreferences(ctx context.Context, tenantID uuid.UUID, bo, term, region string) ([]*SourcePreference, error) {
	return s.repo.ListPreferences(ctx, tenantID, bo, term, region)
}

// RequestOverride creates a draft overlay that overrides an existing production preference.
// It also runs an impact simulation and populates ImpactAnalysis before persisting.
func (s *Service) RequestOverride(ctx context.Context, existingID uuid.UUID, managerID uuid.UUID, reason string, validTo time.Time) (*SourcePreference, error) {
	existing, err := s.repo.GetPreference(ctx, existingID)
	if err != nil {
		return nil, fmt.Errorf("get existing preference: %w", err)
	}

	override, err := existing.CreateOverrideRequest(managerID, reason, validTo)
	if err != nil {
		return nil, err
	}

	// Simulate data impact (stub: counts calendar dates in validity window)
	override.ImpactAnalysis = s.simulateImpact(existing, override)

	override.ID = uuid.New()
	override.CreatedAt = time.Now()
	override.UpdatedAt = time.Now()

	if err := s.repo.CreatePreference(ctx, override); err != nil {
		return nil, fmt.Errorf("create override preference: %w", err)
	}
	impactJSON, _ := override.ImpactAnalysis.JSON()
	_ = s.repo.AppendVersion(ctx, override.ID, "draft", reason, impactJSON, managerID)

	return override, nil
}

// ApproveOverride advances a preference to the next stage in the workflow:
// draft -> testing -> staging -> production
func (s *Service) ApproveOverride(ctx context.Context, prefID uuid.UUID, approverID uuid.UUID, notes string) (*SourcePreference, error) {
	p, err := s.repo.GetPreference(ctx, prefID)
	if err != nil {
		return nil, err
	}
	next, err := nextStage(p.Status)
	if err != nil {
		return nil, err
	}
	p.Status = next
	p.Version++
	uid := approverID
	p.UpdatedBy = &uid
	if err := s.repo.UpdatePreference(ctx, p); err != nil {
		return nil, fmt.Errorf("approve override: %w", err)
	}
	impactJSON, _ := p.ImpactAnalysis.JSON()
	_ = s.repo.AppendVersion(ctx, p.ID, p.Status, notes, impactJSON, approverID)
	return p, nil
}

// PromoteStage is an alias for ApproveOverride when called by the system
func (s *Service) PromoteStage(ctx context.Context, prefID uuid.UUID, promotedBy uuid.UUID) (*SourcePreference, error) {
	return s.ApproveOverride(ctx, prefID, promotedBy, "Promoted by system")
}

// simulateImpact produces a basic impact analysis comparing old vs new source
func (s *Service) simulateImpact(old, new *SourcePreference) ImpactAnalysis {
	delta := new.Confidence - old.Confidence
	if delta < 0 {
		delta = -delta
	}
	impact := ImpactAnalysis{
		AffectedDates:    365, // placeholder: count valid_from -> valid_to days
		ConfidenceDelta:  new.Confidence - old.Confidence,
		ConfidenceBefore: old.Confidence,
		ConfidenceAfter:  new.Confidence,
		BusinessImpact:   DetermineBusinessImpact(delta),
	}
	return impact
}

// nextStage returns the next workflow stage or an error if already at production
func nextStage(current string) (string, error) {
	stages := map[string]string{
		"draft":   "testing",
		"testing": "staging",
		"staging": "production",
	}
	next, ok := stages[current]
	if !ok {
		return "", fmt.Errorf("preference is already in '%s' status; cannot advance further", current)
	}
	return next, nil
}

// --- Exception Management ---

// CreateException persists an exception, marking critical path when warranted
func (s *Service) CreateException(ctx context.Context, e *SourceException) (*SourceException, error) {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	e.CriticalPath = e.IsCritical()
	e.Status = "open"
	e.CreatedAt = time.Now()

	if err := s.repo.CreateException(ctx, e); err != nil {
		return nil, fmt.Errorf("create exception: %w", err)
	}

	// Log initial history entry
	_ = s.repo.AppendExceptionHistory(ctx, e.ID, "open", "Exception registered", nil)

	// Critical exceptions trigger immediate alerting (logged here; integrate with alerting in production)
	if e.CriticalPath {
		log.Printf("[preference.Service] CRITICAL exception created: %s — %s", e.ExceptionType, e.Description)
	}

	return e, nil
}

// ListExceptions returns exceptions for a tenant, optionally filtered by status
func (s *Service) ListExceptions(ctx context.Context, tenantID uuid.UUID, status string) ([]*SourceException, error) {
	return s.repo.ListExceptions(ctx, tenantID, status)
}

// ResolveException marks an exception as resolved
func (s *Service) ResolveException(ctx context.Context, id uuid.UUID, resolvedBy uuid.UUID) error {
	if err := s.repo.ResolveException(ctx, id, resolvedBy); err != nil {
		return fmt.Errorf("resolve exception: %w", err)
	}
	_ = s.repo.AppendExceptionHistory(ctx, id, "resolved", "Exception resolved", &resolvedBy)
	return nil
}

// --- Analytics ---

// GetAnalytics returns preference ranking analytics for a tenant
func (s *Service) GetAnalytics(ctx context.Context, tenantID uuid.UUID, bo, term, region string) (*AnalyticsReport, error) {
	rankings, err := s.repo.GetRankings(ctx, tenantID, bo, term, region)
	if err != nil {
		return nil, fmt.Errorf("get rankings: %w", err)
	}
	return &AnalyticsReport{
		Rankings:       rankings,
		BusinessObject: bo,
		SemanticTerm:   term,
		Region:         region,
		GeneratedAt:    time.Now(),
	}, nil
}
