package portfoliomaster

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Service orchestrates portfolio master business logic.
type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// GetSourceRegistry returns active source registry entries for the tenant.
func (s *Service) GetSourceRegistry(ctx context.Context, tenantID uuid.UUID) ([]*SourceRegistry, error) {
	sources, err := s.repo.ListRegistrySources(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("portfolio master: list registry sources: %w", err)
	}
	return sources, nil
}

// GetPortfolioGolden returns current golden records for the tenant, optionally
// filtered by account_type and scoped to asOf (defaults to now if zero).
func (s *Service) GetPortfolioGolden(ctx context.Context, tenantID uuid.UUID, accountType string, asOf time.Time) ([]*PortfolioGolden, error) {
	if asOf.IsZero() {
		asOf = time.Now()
	}
	records, err := s.repo.ListGoldenRecords(ctx, tenantID, accountType, asOf)
	if err != nil {
		return nil, fmt.Errorf("portfolio master: list golden records: %w", err)
	}
	return records, nil
}

// SimulateSourceChange computes the impact of a source preference change on
// golden records. It compares the current primary source for 'field' to
// 'proposedSource' and returns a summary of affected positions.
//
//	field          — "price" or "quantity"
//	accountType    — filters to a specific account type
//	proposedSource — the new source system code/name to simulate
//	oldConfidence  — confidence of current source
//	newConfidence  — confidence of proposed source
func (s *Service) SimulateSourceChange(
	ctx context.Context,
	tenantID uuid.UUID,
	preferenceID uuid.UUID,
	field string,
	accountType string,
	proposedSource string,
	oldConfidence int,
	newConfidence int,
) (*ImpactSimulationResult, error) {
	// 1. Fetch current golden records
	records, err := s.repo.ListGoldenRecords(ctx, tenantID, accountType, time.Now())
	if err != nil {
		return nil, fmt.Errorf("portfolio master: simulate: list golden records: %w", err)
	}

	// 2. Identify positions whose field is sourced from something other than proposedSource
	var changes []PositionChange
	for _, g := range records {
		currentSource := g.SourceSystems[field]
		if currentSource == proposedSource {
			continue // already on proposed source — no change
		}
		changes = append(changes, PositionChange{
			PortfolioID:   g.PortfolioID,
			SecurityID:    g.SecurityID,
			Field:         field,
			OldSource:     currentSource,
			NewSource:     proposedSource,
			OldConfidence: oldConfidence,
			NewConfidence: newConfidence,
		})
	}

	// 3. Compute aggregate confidence metrics
	var sumBefore, sumAfter float64
	n := len(records)
	for _, g := range records {
		sumBefore += float64(g.ConfidenceScore)
		sumAfter += float64(g.ConfidenceScore)
	}
	// Apply the delta to affected positions
	for range changes {
		sumAfter += float64(newConfidence - oldConfidence)
	}
	var avgBefore, avgAfter float64
	if n > 0 {
		avgBefore = sumBefore / float64(n)
		avgAfter = sumAfter / float64(n)
	}
	delta := avgAfter - avgBefore

	return &ImpactSimulationResult{
		PreferenceID:      preferenceID,
		AsOfDate:          time.Now(),
		AffectedPositions: len(changes),
		ConfidenceBefore:  avgBefore,
		ConfidenceAfter:   avgAfter,
		ConfidenceDelta:   delta,
		BusinessImpact:    classifyImpact(delta),
		ChangedPositions:  changes,
		SimulatedAt:       time.Now(),
	}, nil
}

// classifyImpact converts a confidence delta to a human-readable impact label.
func classifyImpact(delta float64) string {
	if delta < 0 {
		delta = -delta
	}
	switch {
	case delta == 0:
		return "none"
	case delta <= 5:
		return "low"
	case delta <= 15:
		return "moderate"
	default:
		return "high"
	}
}
