package analytics

import (
	"context"
	"encoding/json"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/jmoiron/sqlx"
)

// PreAggRecommendationEngine generates pre-aggregation recommendations based on workload.
type PreAggRecommendationEngine struct {
	db        *sqlx.DB
	analyzer  *WorkloadAnalyzer
	preAggSvc *PreAggregationService
}

func NewPreAggRecommendationEngine(db *sqlx.DB, analyzer *WorkloadAnalyzer, preAggSvc *PreAggregationService) *PreAggRecommendationEngine {
	return &PreAggRecommendationEngine{db: db, analyzer: analyzer, preAggSvc: preAggSvc}
}

// RecommendForBO generates recommendations for a specific BO.
func (e *PreAggRecommendationEngine) RecommendForBO(ctx context.Context, tenantID, boName string, window time.Duration) ([]models.PreAggRecommendation, error) {
	profile, err := e.analyzer.AnalyzeBO(ctx, tenantID, boName, window)
	if err != nil {
		return nil, err
	}
	if profile.TotalQueries == 0 {
		return nil, nil
	}

	// Load existing pre-aggs
	existing, _ := e.preAggSvc.ListByBO(ctx, tenantID, boName)

	var recs []models.PreAggRecommendation

	// For each hot group-by, build candidate
	for _, gb := range profile.TopGroupBys {
		grain := gb.Terms
		if len(grain) == 0 {
			continue
		}

		// Select top measures
		var measures []string
		for i, m := range profile.TopMeasures {
			if i >= 5 {
				break
			}
			measures = append(measures, m.Name)
		}

		// Suggested filters (date-based)
		var filters []models.Filter
		for _, f := range profile.TopFilters {
			if strings.Contains(strings.ToLower(f.Term), "date") {
				filters = append(filters, models.Filter{
					Term:     f.Term,
					Operator: f.Operator,
					Value:    "<window>",
				})
				break
			}
		}

		// Check if existing pre-agg covers this
		if match := e.findMatchingPreAgg(existing, grain, measures); match != nil {
			// Could suggest tuning, but skip for now
			continue
		}

		// Compute cost estimate
		cost := e.estimateCost(profile, gb, measures, 7) // 7-day window

		if cost.Score < 0.5 {
			continue // Below threshold
		}

		recs = append(recs, models.PreAggRecommendation{
			TenantID:           tenantID,
			BOName:             boName,
			Grain:              grain,
			Measures:           measures,
			SuggestedFilters:   filters,
			CostEstimate:       cost,
			RecommendationType: "new",
		})
	}

	// Sort by score descending
	sort.Slice(recs, func(i, j int) bool {
		return recs[i].CostEstimate.Score > recs[j].CostEstimate.Score
	})

	return recs, nil
}

// RecommendGlobal generates recommendations across all BOs.
func (e *PreAggRecommendationEngine) RecommendGlobal(ctx context.Context, window time.Duration) ([]models.PreAggRecommendation, error) {
	profiles, err := e.analyzer.AnalyzeAll(ctx, window)
	if err != nil {
		return nil, err
	}

	var all []models.PreAggRecommendation
	for _, p := range profiles {
		recs, err := e.RecommendForBO(ctx, p.TenantID, p.BOName, window)
		if err != nil {
			continue
		}
		all = append(all, recs...)
	}

	sort.Slice(all, func(i, j int) bool {
		return all[i].CostEstimate.Score > all[j].CostEstimate.Score
	})

	return all, nil
}

func (e *PreAggRecommendationEngine) findMatchingPreAgg(existing []models.PreAggDescriptor, grain, measures []string) *models.PreAggDescriptor {
	grainSet := make(map[string]bool)
	for _, g := range grain {
		grainSet[g] = true
	}

	for _, pa := range existing {
		// Simple heuristic: name contains grain terms
		match := true
		for g := range grainSet {
			if !strings.Contains(strings.ToLower(pa.Name), strings.ToLower(g)) {
				match = false
				break
			}
		}
		if match {
			return &pa
		}
	}
	return nil
}

func (e *PreAggRecommendationEngine) estimateCost(profile *models.BOWorkloadProfile, gb models.GroupByProfile, measures []string, windowDays int) models.PreAggCostEstimate {
	// Estimate queries per day
	estimatedQPD := int(float64(gb.QueryCount) / float64(windowDays))
	if estimatedQPD < 1 {
		estimatedQPD = 1
	}

	// Heuristic: estimate pre-agg row count based on cardinality
	// Assume 1% of scanned rows go into pre-agg (rough approximation)
	estimatedRows := int64(math.Max(1000, gb.AvgRowsScanned*0.01))

	// Row size estimate (rough: 50 bytes per column)
	rowSize := int64(len(gb.Terms)+len(measures)) * 50
	storage := estimatedRows * rowSize

	// Speedup factor
	speedup := 1.0
	if gb.AvgRowsScanned > float64(estimatedRows) {
		speedup = gb.AvgRowsScanned / float64(estimatedRows)
	}
	speedup = math.Min(speedup, 100) // Cap at 100x

	// Costs
	buildCost := float64(estimatedRows) * 0.001
	refreshCost := buildCost * 0.2

	// Score: benefit / cost
	score := (float64(estimatedQPD) * gb.AvgDurationMs * (speedup - 1.0)) /
		(float64(storage) + refreshCost + 1.0)

	return models.PreAggCostEstimate{
		TenantID:               profile.TenantID,
		BOName:                 profile.BOName,
		Grain:                  gb.Terms,
		Measures:               measures,
		EstimatedQueriesPerDay: estimatedQPD,
		AvgDurationMs:          gb.AvgDurationMs,
		P95DurationMs:          gb.P95DurationMs,
		AvgRowsScanned:         gb.AvgRowsScanned,
		EstimatedSpeedupFactor: speedup,
		EstimatedStorageBytes:  storage,
		EstimatedBuildCost:     buildCost,
		EstimatedRefreshCost:   refreshCost,
		Score:                  score,
	}
}

// --- Telemetry Ingestion ---

// TelemetryService handles ingestion of query telemetry.
type TelemetryService struct {
	db *sqlx.DB
}

func NewTelemetryService(db *sqlx.DB) *TelemetryService {
	return &TelemetryService{db: db}
}

// Ingest inserts a telemetry event.
func (s *TelemetryService) Ingest(ctx context.Context, req models.TelemetryIngestionRequest) error {
	groupByJSON, _ := json.Marshal(req.GroupByTerms)
	measuresJSON, _ := json.Marshal(req.Measures)
	filtersJSON, _ := json.Marshal(req.Filters)

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO semantic.query_telemetry (
			id, tenant_id, bo_name, cube_query_id, starrocks_query_id,
			started_at, duration_ms, rows_scanned, bytes_scanned, rows_returned,
			status, error_message, group_by_terms, measures, filters, source,
			preagg_id, preagg_hit
		) VALUES (
			gen_random_uuid(), $1, $2, $3, $4,
			now(), $5, $6, $7, $8,
			$9, $10, $11, $12, $13, $14,
			$15, $16
		)
	`,
		req.TenantID, req.BOName, req.CubeQueryID, req.StarRocksQueryID,
		req.DurationMs, req.RowsScanned, req.BytesScanned, req.RowsReturned,
		req.Status, req.ErrorMessage, groupByJSON, measuresJSON, filtersJSON, req.Source,
		req.PreAggID, req.PreAggHit,
	)
	return err
}
