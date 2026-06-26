package query

import (
	"fmt"
	"strings"
	"time"
)

// RefinementEngine handles the regeneration of queries after refinements
type RefinementEngine struct {
	generationEngine *GenerationEngine
}

// NewRefinementEngine creates a new refinement engine
func NewRefinementEngine() *RefinementEngine {
	return &RefinementEngine{
		generationEngine: NewGenerationEngine(),
	}
}

// RegenerateQuery regenerates a query based on updated intent and governance context
func (re *RefinementEngine) RegenerateQuery(updatedIntent *ParsedIntent, govCtx *GovernanceContext) (*GeneratedQuery, error) {
	// Generate new query skeleton
	skeleton, err := re.generationEngine.GenerateQuerySkeleton(updatedIntent, govCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query skeleton: %w", err)
	}

	// Generate final SQL
	sql, err := re.generationEngine.GenerateSQL(skeleton, govCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to generate SQL: %w", err)
	}

	// Create generated query
	generatedQuery := &GeneratedQuery{
		SQL:         sql,
		SemanticSQL: skeleton.SemanticSQL,
		Measures:    skeleton.Measures,
		Dimensions:  skeleton.Dimensions,
		Filters:     skeleton.Filters,
		OrderBy:     skeleton.OrderBy,
	}

	return generatedQuery, nil
}

// ValidateRefinement checks if a refinement is valid and won't break the query
func (re *RefinementEngine) ValidateRefinement(intent *ParsedIntent, govCtx *GovernanceContext) []string {
	var warnings []string

	// Check for empty metrics
	if len(intent.Metrics) == 0 {
		warnings = append(warnings, "Query has no metrics selected")
	}

	// Check for governance compliance
	for _, metric := range intent.Metrics {
		if !re.isMetricAllowed(metric, govCtx) {
			warnings = append(warnings, fmt.Sprintf("Metric '%s' may not be accessible", metric))
		}
	}

	for _, dimension := range intent.Dimensions {
		if !re.isDimensionAllowed(dimension, govCtx) {
			warnings = append(warnings, fmt.Sprintf("Dimension '%s' may not be accessible", dimension))
		}
	}

	// Check for potentially expensive queries
	if len(intent.Dimensions) == 0 && intent.TimeRange == nil {
		warnings = append(warnings, "Query may return too many results without filters or dimensions")
	}

	// Check for conflicting filters
	conflicts := re.detectFilterConflicts(intent.Filters)
	warnings = append(warnings, conflicts...)

	return warnings
}

// OptimizeRefinement suggests optimizations for the current query
func (re *RefinementEngine) OptimizeRefinement(intent *ParsedIntent, query *GeneratedQuery) []RefinementSuggestion {
	var optimizations []RefinementSuggestion

	// Suggest adding indexes for performance
	if re.shouldSuggestIndex(intent) {
		optimizations = append(optimizations, RefinementSuggestion{
			ID:          generateOptimizationID(),
			Type:        "optimization",
			Description: "Consider adding database indexes on frequently queried columns",
			Action:      "optimize",
			Reason:      "Indexes can significantly improve query performance",
			Confidence:  0.8,
		})
	}

	// Suggest query restructuring for better performance
	if re.shouldRestructureQuery(intent, query) {
		optimizations = append(optimizations, RefinementSuggestion{
			ID:          generateOptimizationID(),
			Type:        "optimization",
			Description: "Query could benefit from restructuring for better performance",
			Action:      "optimize",
			Reason:      "Restructured queries often execute faster",
			Confidence:  0.7,
		})
	}

	// Suggest caching for frequently run queries
	if re.shouldSuggestCaching(intent) {
		optimizations = append(optimizations, RefinementSuggestion{
			ID:          generateOptimizationID(),
			Type:        "optimization",
			Description: "Consider caching results for this frequently run query",
			Action:      "optimize",
			Reason:      "Caching can provide sub-second response times",
			Confidence:  0.6,
		})
	}

	return optimizations
}

// PreviewRefinement shows what the query will look like after applying refinements
func (re *RefinementEngine) PreviewRefinement(intent *ParsedIntent, govCtx *GovernanceContext) (*QueryPreview, error) {
	// Generate the query
	query, err := re.RegenerateQuery(intent, govCtx)
	if err != nil {
		return nil, err
	}

	// Estimate execution metrics
	estimatedRows := re.estimateResultRows(intent)
	estimatedTime := re.estimateExecutionTime(intent, estimatedRows)

	preview := &QueryPreview{
		GeneratedQuery:    query,
		EstimatedRows:     estimatedRows,
		EstimatedTime:     estimatedTime,
		PotentialWarnings: re.ValidateRefinement(intent, govCtx),
		Optimizations:     re.OptimizeRefinement(intent, query),
	}

	return preview, nil
}

// Helper methods

func (re *RefinementEngine) isMetricAllowed(metric string, govCtx *GovernanceContext) bool {
	if len(govCtx.AllowedMetrics) == 0 {
		return true
	}
	for _, allowed := range govCtx.AllowedMetrics {
		if strings.EqualFold(metric, allowed) {
			return true
		}
	}
	return false
}

func (re *RefinementEngine) isDimensionAllowed(dimension string, govCtx *GovernanceContext) bool {
	if len(govCtx.AllowedDimensions) == 0 {
		return true
	}
	for _, allowed := range govCtx.AllowedDimensions {
		if strings.EqualFold(dimension, allowed) {
			return true
		}
	}
	return false
}

func (re *RefinementEngine) detectFilterConflicts(filters []IntentFilter) []string {
	var conflicts []string

	// Check for conflicting date ranges
	dateFilters := 0
	for _, filter := range filters {
		if re.isDateFilter(filter) {
			dateFilters++
		}
	}

	if dateFilters > 1 {
		conflicts = append(conflicts, "Multiple date filters detected - this may cause unexpected results")
	}

	// Check for mutually exclusive filters
	for i, filter1 := range filters {
		for j, filter2 := range filters {
			if i != j && re.filtersConflict(filter1, filter2) {
				conflicts = append(conflicts, fmt.Sprintf("Filters on '%s' and '%s' may conflict", filter1.Field, filter2.Field))
			}
		}
	}

	return conflicts
}

func (re *RefinementEngine) isDateFilter(filter IntentFilter) bool {
	fieldLower := strings.ToLower(filter.Field)
	return strings.Contains(fieldLower, "date") ||
		strings.Contains(fieldLower, "time") ||
		strings.Contains(fieldLower, "created") ||
		strings.Contains(fieldLower, "updated")
}

func (re *RefinementEngine) filtersConflict(filter1, filter2 IntentFilter) bool {
	// Simple conflict detection - same field with different operators
	if strings.EqualFold(filter1.Field, filter2.Field) {
		return filter1.Operator != filter2.Operator
	}
	return false
}

func (re *RefinementEngine) shouldSuggestIndex(intent *ParsedIntent) bool {
	// Suggest indexes for queries with many filters or complex conditions
	return len(intent.Filters) > 2 || len(intent.Dimensions) > 3
}

func (re *RefinementEngine) shouldRestructureQuery(intent *ParsedIntent, _ *GeneratedQuery) bool {
	// Suggest restructuring for complex queries
	return len(intent.Metrics) > 5 || len(intent.Dimensions) > 5 || len(intent.Filters) > 5
}

func (re *RefinementEngine) shouldSuggestCaching(intent *ParsedIntent) bool {
	// Suggest caching for simple, frequently-run queries
	return len(intent.Metrics) <= 2 && len(intent.Dimensions) <= 2 && len(intent.Filters) <= 2
}

func (re *RefinementEngine) estimateResultRows(intent *ParsedIntent) int64 {
	// Simple estimation based on query complexity
	baseRows := int64(1000)

	// More dimensions = more rows
	baseRows *= int64(1 + len(intent.Dimensions))

	// Filters reduce rows
	if len(intent.Filters) > 0 {
		baseRows = baseRows / int64(len(intent.Filters)+1)
	}

	// Time ranges affect row count
	if intent.TimeRange != nil {
		timeRangeStr := strings.ToLower(intent.TimeRange.Label)
		if strings.Contains(timeRangeStr, "last week") {
			baseRows = baseRows / 4
		} else if strings.Contains(timeRangeStr, "last month") {
			baseRows = baseRows / 2
		}
	}

	return baseRows
}

func (re *RefinementEngine) estimateExecutionTime(intent *ParsedIntent, estimatedRows int64) string {
	// Estimate execution time based on complexity
	baseTime := 100 // milliseconds

	// More complex queries take longer
	baseTime += len(intent.Metrics) * 20
	baseTime += len(intent.Dimensions) * 30
	baseTime += len(intent.Filters) * 15

	// Large result sets take longer
	if estimatedRows > 10000 {
		baseTime *= 2
	} else if estimatedRows > 100000 {
		baseTime *= 5
	}

	if baseTime < 1000 {
		return fmt.Sprintf("%dms", baseTime)
	} else {
		return fmt.Sprintf("%.1fs", float64(baseTime)/1000)
	}
}

// QueryPreview represents a preview of what a query will return
type QueryPreview struct {
	GeneratedQuery    *GeneratedQuery        `json:"generated_query"`
	EstimatedRows     int64                  `json:"estimated_rows"`
	EstimatedTime     string                 `json:"estimated_time"`
	PotentialWarnings []string               `json:"potential_warnings"`
	Optimizations     []RefinementSuggestion `json:"optimizations"`
}

func generateOptimizationID() string {
	return fmt.Sprintf("opt_%d", time.Now().UnixNano())
}
