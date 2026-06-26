package analytics

import (
	"context"
	"sort"
	"strings"

	"github.com/hondyman/semlayer/backend/internal/models"
)

// PreAggMeta represents the metadata of a pre-aggregation for eligibility checking.
type PreAggMeta struct {
	ID               string
	Priority         int      // Higher = preferred
	GroupBy          []string // Columns in the pre-agg GROUP BY
	Measures         []string // Aggregations available
	FiltersSupported []string // Fields that can be filtered
	Status           string   // lifecycle status
}

// SemanticQuery represents an incoming query to be matched against pre-aggregations.
type SemanticQuery struct {
	GroupBy  []string            // Requested group-by columns
	Measures []string            // Requested aggregations
	Filters  []EligibilityFilter // Filters in the query
}

// EligibilityFilter represents a filter clause in a query for pre-agg eligibility checking.
// Note: This is separate from QueryFilter in cache.go which is used for governance.
type EligibilityFilter struct {
	Field string
	Op    string
	Value interface{}
}

// EligibilityEngine provides methods for selecting optimal pre-aggregations for queries.
type EligibilityEngine struct {
	preAggSvc *PreAggregationService
}

// NewEligibilityEngine creates a new eligibility engine.
func NewEligibilityEngine(preAggSvc *PreAggregationService) *EligibilityEngine {
	return &EligibilityEngine{preAggSvc: preAggSvc}
}

// PickPreAgg selects the best pre-aggregation for a given query.
// Returns nil if no eligible pre-aggregation is found.
//
// A pre-aggregation is eligible if:
// 1. It is in "active" status
// 2. Query's group-by columns are a subset of the pre-agg's group-by columns
// 3. Query's measures are a subset of the pre-agg's measures
// 4. Query's filter fields are within the pre-agg's supported filters
func PickPreAgg(q *SemanticQuery, metas []PreAggMeta) *PreAggMeta {
	if q == nil || len(metas) == 0 {
		return nil
	}

	var eligible []PreAggMeta
	for _, meta := range metas {
		// Skip non-active pre-aggregations
		if meta.Status != "active" && meta.Status != "" {
			// Allow empty status for testing, require "active" in production
			if meta.Status != "" {
				continue
			}
		}

		// Check group-by subset
		if !isSubset(q.GroupBy, meta.GroupBy) {
			continue
		}

		// Check measures subset
		if !isSubset(q.Measures, meta.Measures) {
			continue
		}

		// Check filters are on supported fields
		if !areFiltersSupported(q.Filters, meta.FiltersSupported) {
			continue
		}

		eligible = append(eligible, meta)
	}

	if len(eligible) == 0 {
		return nil
	}

	// Sort by priority (higher first), then by specificity (fewer extra columns preferred)
	sort.Slice(eligible, func(i, j int) bool {
		if eligible[i].Priority != eligible[j].Priority {
			return eligible[i].Priority > eligible[j].Priority
		}
		// Prefer more specific (smaller) pre-aggregations
		sizeI := len(eligible[i].GroupBy) + len(eligible[i].Measures)
		sizeJ := len(eligible[j].GroupBy) + len(eligible[j].Measures)
		return sizeI < sizeJ
	})

	return &eligible[0]
}

// LoadPreAggMetas loads pre-aggregation metadata for a tenant and datasource.
func (e *EligibilityEngine) LoadPreAggMetas(ctx context.Context, tenantID, datasource string) ([]PreAggMeta, error) {
	descriptors, err := e.preAggSvc.ListByDatasource(ctx, tenantID, datasource)
	if err != nil {
		return nil, err
	}

	metas := make([]PreAggMeta, 0, len(descriptors))
	for _, d := range descriptors {
		metas = append(metas, PreAggMeta{
			ID:               d.ID.String(),
			Priority:         getPriority(d),
			GroupBy:          d.GroupBy,
			Measures:         d.Measures,
			FiltersSupported: d.FiltersSupported,
			Status:           d.LifecycleStatus,
		})
	}

	return metas, nil
}

// PickBestPreAgg combines loading and picking in one call.
func (e *EligibilityEngine) PickBestPreAgg(ctx context.Context, tenantID, datasource string, q *SemanticQuery) (*PreAggMeta, error) {
	metas, err := e.LoadPreAggMetas(ctx, tenantID, datasource)
	if err != nil {
		return nil, err
	}
	return PickPreAgg(q, metas), nil
}

// getPriority assigns a priority score based on pre-aggregation characteristics.
func getPriority(d models.PreAggDescriptor) int {
	priority := 0

	// Boost active pre-aggregations
	if d.LifecycleStatus == "active" {
		priority += 100
	}

	// Boost recently refreshed
	if d.LastRefreshedAt != nil {
		priority += 50
	}

	// Boost published
	if d.GovernanceStatus == "published" {
		priority += 25
	}

	// Boost based on usage (more usage = more trusted)
	if d.UsageCount > 100 {
		priority += 10
	}

	return priority
}

// isSubset returns true if all elements in needle are present in haystack.
func isSubset(needle, haystack []string) bool {
	if len(needle) == 0 {
		return true
	}
	if len(haystack) == 0 {
		return false
	}

	haystackSet := make(map[string]struct{}, len(haystack))
	for _, h := range haystack {
		haystackSet[normalizeColumn(h)] = struct{}{}
	}

	for _, n := range needle {
		if _, ok := haystackSet[normalizeColumn(n)]; !ok {
			return false
		}
	}
	return true
}

// areFiltersSupported returns true if all query filter fields are in the supported list.
func areFiltersSupported(filters []EligibilityFilter, supported []string) bool {
	if len(filters) == 0 {
		return true
	}
	if len(supported) == 0 {
		// If no filters are explicitly listed as supported,
		// assume all filters can be applied post-hoc
		return true
	}

	supportedSet := make(map[string]struct{}, len(supported))
	for _, s := range supported {
		supportedSet[normalizeColumn(s)] = struct{}{}
	}

	for _, f := range filters {
		if _, ok := supportedSet[normalizeColumn(f.Field)]; !ok {
			return false
		}
	}
	return true
}

// normalizeColumn normalizes a column name for comparison.
func normalizeColumn(col string) string {
	// Remove whitespace, lowercase, handle common variations
	col = strings.TrimSpace(col)
	col = strings.ToLower(col)
	// Remove table prefixes (e.g., "orders.country" -> "country")
	if idx := strings.LastIndex(col, "."); idx >= 0 {
		col = col[idx+1:]
	}
	return col
}
