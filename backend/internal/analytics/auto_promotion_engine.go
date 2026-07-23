package analytics

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
)

// AutoPromotionConfig defines thresholds for auto-promoting pre-aggregations.
type AutoPromotionConfig struct {
	MinFreq          int64   // Minimum query frequency over the period
	MinAvgLatencyMs  float64 // Minimum average latency to be worth optimizing
	MaxCoverageRatio float64 // Only promote if coverage is below this threshold
	DryRun           bool    // If true, log what would be promoted without creating
}

// DefaultAutoPromotionConfig returns sensible defaults.
func DefaultAutoPromotionConfig() AutoPromotionConfig {
	return AutoPromotionConfig{
		MinFreq:          50,
		MinAvgLatencyMs:  500.0,
		MaxCoverageRatio: 0.3,
		DryRun:           true,
	}
}

// AutoPromotionEngine automatically creates pre-aggregations based on workload analysis.
type AutoPromotionEngine struct {
	suggestionSvc *PreAggSuggestionService
	preAggSvc     *PreAggregationService
	coverageSvc   *CoverageDashboardService
}

// NewAutoPromotionEngine creates a new auto-promotion engine.
func NewAutoPromotionEngine(
	suggestionSvc *PreAggSuggestionService,
	preAggSvc *PreAggregationService,
	coverageSvc *CoverageDashboardService,
) *AutoPromotionEngine {
	return &AutoPromotionEngine{
		suggestionSvc: suggestionSvc,
		preAggSvc:     preAggSvc,
		coverageSvc:   coverageSvc,
	}
}

// PromotionCandidate represents a suggestion that meets promotion criteria.
type PromotionCandidate struct {
	TenantID        string
	Datasource      string
	GroupBy         []string
	Measures        []string
	Filters         []string
	Freq            int64
	AvgLatency      float64
	CurrentCoverage float64
	Score           float64
}

// Run executes the auto-promotion engine for a tenant.
func (e *AutoPromotionEngine) Run(ctx context.Context, tenantID string, cfg AutoPromotionConfig) ([]PromotionCandidate, error) {
	// Step 1: Get suggestions from the suggestion engine
	suggestions, err := e.suggestionSvc.ListSuggestions(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get suggestions: %w", err)
	}

	// Step 2: Get current coverage metrics
	coverageMetrics, err := e.coverageSvc.GetCoverageByDatasource(ctx, tenantID, 7)
	if err != nil {
		return nil, fmt.Errorf("failed to get coverage: %w", err)
	}

	coverageMap := make(map[string]float64)
	for _, m := range coverageMetrics {
		coverageMap[m.Datasource] = m.CoverageRatio
	}

	// Step 3: Filter suggestions by promotion criteria
	var candidates []PromotionCandidate
	for _, sug := range suggestions {
		// Check frequency threshold
		if sug.Freq < cfg.MinFreq {
			continue
		}

		// Check latency threshold
		if sug.AvgLatency < cfg.MinAvgLatencyMs {
			continue
		}

		// Check coverage threshold
		currentCoverage := coverageMap[sug.Datasource]
		if currentCoverage >= cfg.MaxCoverageRatio {
			continue
		}

		// Check if pre-agg already exists for this pattern
		exists, err := e.preAggSvc.ExistsForPattern(ctx, tenantID, sug.Datasource, sug.GroupBy)
		if err != nil {
			log.Printf("Error checking pre-agg existence: %v", err)
			continue
		}
		if exists {
			continue
		}

		// Calculate promotion score (higher = better)
		score := float64(sug.Freq) * sug.AvgLatency * (1.0 - currentCoverage)

		candidates = append(candidates, PromotionCandidate{
			TenantID:        tenantID,
			Datasource:      sug.Datasource,
			GroupBy:         sug.GroupBy,
			Measures:        []string{"COUNT(*)", "SUM(revenue)"}, // TODO: extract from suggestions
			Filters:         sug.Filters,
			Freq:            sug.Freq,
			AvgLatency:      sug.AvgLatency,
			CurrentCoverage: currentCoverage,
			Score:           score,
		})
	}

	// Step 4: Sort by score and promote top N
	if len(candidates) == 0 {
		log.Printf("No candidates meet promotion criteria for tenant %s", tenantID)
		return candidates, nil
	}

	// Sort by score descending (highest value first)
	sortByScore(candidates)

	// Step 5: Create pre-aggregations (or log in dry-run mode)
	for i, candidate := range candidates {
		if i >= 5 { // Limit to top 5 per run
			break
		}

		if cfg.DryRun {
			log.Printf("[DRY RUN] Would promote pre-agg: tenant=%s datasource=%s group_by=%v score=%.2f",
				candidate.TenantID, candidate.Datasource, candidate.GroupBy, candidate.Score)
			continue
		}

		// Actually create the pre-aggregation
		preAggID := uuid.New()
		name := fmt.Sprintf("auto_%s_%s", candidate.Datasource, preAggID.String()[:8])

		log.Printf("Auto-promoting pre-agg: %s (score=%.2f)", name, candidate.Score)

		// TODO: Call e.preAggSvc.Create() here when ready
		// For now, just log
		_ = name
	}

	return candidates, nil
}

// sortByScore sorts candidates by score descending.
func sortByScore(candidates []PromotionCandidate) {
	// Simple bubble sort for now (replace with sort.Slice in production)
	for i := 0; i < len(candidates); i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[j].Score > candidates[i].Score {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}
}
