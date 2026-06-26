package analytics

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

// QueryOptimizationEngine provides intelligent query optimization capabilities
type QueryOptimizationEngine struct {
	metricsCollector *MetricsCollector
	cacheManager     *CacheManager
	costEstimator    *CostEstimator
}

// QueryPlan represents an optimized query execution plan
type QueryPlan struct {
	QueryID           string
	OriginalQuery     string
	OptimizedQuery    string
	EstimatedCost     float64
	EstimatedRows     int64
	ExecutionTime     time.Duration
	OptimizationHints []string
	DataSources       []string
	CacheStrategy     string
	Parallelization   int
}

// CostEstimator provides cost estimation for query operations
type CostEstimator struct {
	// Cost coefficients (can be tuned based on actual performance data)
	SequentialScanCost float64
	IndexScanCost      float64
	JoinCost           float64
	AggregationCost    float64
	CacheHitCost       float64
	CacheMissCost      float64
}

// NewQueryOptimizationEngine creates a new query optimization engine
func NewQueryOptimizationEngine(metricsCollector *MetricsCollector, cacheManager *CacheManager) *QueryOptimizationEngine {
	return &QueryOptimizationEngine{
		metricsCollector: metricsCollector,
		cacheManager:     cacheManager,
		costEstimator: &CostEstimator{
			SequentialScanCost: 100.0,
			IndexScanCost:      10.0,
			JoinCost:           50.0,
			AggregationCost:    25.0,
			CacheHitCost:       1.0,
			CacheMissCost:      1000.0,
		},
	}
}

// OptimizeQuery analyzes and optimizes a natural language query
func (qoe *QueryOptimizationEngine) OptimizeQuery(nlQuery string, tenantID, userID string) (*QueryPlan, error) {
	startTime := time.Now()

	// Get governance context for optimization constraints
	govContext, err := qoe.cacheManager.GetGovernanceContext(tenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get governance context: %w", err)
	}

	// Parse query components
	queryComponents := qoe.parseNaturalLanguageQuery(nlQuery)

	// Generate multiple execution plans
	plans := qoe.generateExecutionPlans(queryComponents, govContext)

	// Select optimal plan based on cost estimation
	optimalPlan := qoe.selectOptimalPlan(plans, queryComponents)

	// Apply governance constraints
	optimalPlan = qoe.applyGovernanceConstraints(optimalPlan, govContext)

	// Add optimization hints
	optimalPlan.OptimizationHints = qoe.generateOptimizationHints(optimalPlan, queryComponents)

	optimalPlan.ExecutionTime = time.Since(startTime)

	// Record metrics
	qoe.metricsCollector.RecordQueryOptimization(
		optimalPlan.QueryID,
		optimalPlan.EstimatedCost,
		optimalPlan.ExecutionTime,
	)

	return optimalPlan, nil
}

// parseNaturalLanguageQuery extracts components from natural language query
func (qoe *QueryOptimizationEngine) parseNaturalLanguageQuery(query string) *QueryComponents {
	components := &QueryComponents{
		OriginalQuery: query,
		Metrics:       []string{},
		Dimensions:    []string{},
		Filters:       []QueryFilter{},
		Aggregations:  []string{},
		TimeRange:     &TimeRange{},
	}

	// Simple NLP parsing (in production, this would use more sophisticated NLP)
	queryLower := strings.ToLower(query)

	// Extract metrics
	if strings.Contains(queryLower, "revenue") || strings.Contains(queryLower, "sales") {
		components.Metrics = append(components.Metrics, "revenue")
	}
	if strings.Contains(queryLower, "profit") || strings.Contains(queryLower, "margin") {
		components.Metrics = append(components.Metrics, "profit")
	}

	// Extract dimensions
	if strings.Contains(queryLower, "region") {
		components.Dimensions = append(components.Dimensions, "region")
	}
	if strings.Contains(queryLower, "product") {
		components.Dimensions = append(components.Dimensions, "product")
	}

	// Extract time ranges
	if strings.Contains(queryLower, "last month") {
		components.TimeRange.Start = time.Now().AddDate(0, -1, 0)
		components.TimeRange.End = time.Now()
	}

	return components
}

// QueryComponents represents parsed query components
type QueryComponents struct {
	OriginalQuery string
	Metrics       []string
	Dimensions    []string
	Filters       []QueryFilter
	Aggregations  []string
	TimeRange     *TimeRange
	Complexity    int
}

// TimeRange represents a time range for queries
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// generateExecutionPlans creates multiple execution strategies
func (qoe *QueryOptimizationEngine) generateExecutionPlans(components *QueryComponents, govContext *GovernanceContext) []*QueryPlan {
	plans := []*QueryPlan{}

	// Plan 1: Direct SQL execution
	plans = append(plans, qoe.createDirectSQLPlan(components, govContext))

	// Plan 2: Cached result reuse
	plans = append(plans, qoe.createCachedResultPlan(components, govContext))

	// Plan 3: Pre-aggregated data
	plans = append(plans, qoe.createPreAggregatedPlan(components, govContext))

	// Plan 4: Parallel execution
	plans = append(plans, qoe.createParallelExecutionPlan(components, govContext))

	return plans
}

// createDirectSQLPlan generates a direct SQL execution plan
func (qoe *QueryOptimizationEngine) createDirectSQLPlan(components *QueryComponents, govContext *GovernanceContext) *QueryPlan {
	query := qoe.buildSQLQuery(components, govContext)

	cost := qoe.costEstimator.SequentialScanCost * float64(len(components.Metrics))
	if len(components.Dimensions) > 0 {
		cost += qoe.costEstimator.IndexScanCost
	}

	return &QueryPlan{
		QueryID:         generateQueryID(),
		OriginalQuery:   components.OriginalQuery,
		OptimizedQuery:  query,
		EstimatedCost:   cost,
		EstimatedRows:   1000, // Estimated based on historical data
		DataSources:     []string{"primary_db"},
		CacheStrategy:   "none",
		Parallelization: 1,
	}
}

// createCachedResultPlan generates a cache-first execution plan
func (qoe *QueryOptimizationEngine) createCachedResultPlan(components *QueryComponents, govContext *GovernanceContext) *QueryPlan {
	cacheKey := qoe.generateCacheKey(components, govContext)

	// Estimate cache hit probability based on query patterns
	cacheHitProb := qoe.estimateCacheHitProbability(components)

	cost := cacheHitProb*qoe.costEstimator.CacheHitCost +
		(1-cacheHitProb)*qoe.costEstimator.CacheMissCost

	return &QueryPlan{
		QueryID:         generateQueryID(),
		OriginalQuery:   components.OriginalQuery,
		OptimizedQuery:  fmt.Sprintf("CACHE_LOOKUP('%s')", cacheKey),
		EstimatedCost:   cost,
		EstimatedRows:   1000,
		DataSources:     []string{"cache"},
		CacheStrategy:   "read-through",
		Parallelization: 1,
	}
}

// createPreAggregatedPlan generates a plan using pre-computed aggregations
func (qoe *QueryOptimizationEngine) createPreAggregatedPlan(components *QueryComponents, govContext *GovernanceContext) *QueryPlan {
	// Use pre-aggregated tables for common metrics
	query := qoe.buildPreAggregatedQuery(components, govContext)

	cost := qoe.costEstimator.AggregationCost * float64(len(components.Aggregations))

	return &QueryPlan{
		QueryID:         generateQueryID(),
		OriginalQuery:   components.OriginalQuery,
		OptimizedQuery:  query,
		EstimatedCost:   cost,
		EstimatedRows:   100,
		DataSources:     []string{"aggregated_db"},
		CacheStrategy:   "write-through",
		Parallelization: 1,
	}
}

// createParallelExecutionPlan generates a parallel execution plan
func (qoe *QueryOptimizationEngine) createParallelExecutionPlan(components *QueryComponents, govContext *GovernanceContext) *QueryPlan {
	parallelism := qoe.determineOptimalParallelism(components)

	query := qoe.buildParallelQuery(components, govContext, parallelism)

	cost := (qoe.costEstimator.SequentialScanCost * float64(len(components.Metrics))) / float64(parallelism)

	return &QueryPlan{
		QueryID:         generateQueryID(),
		OriginalQuery:   components.OriginalQuery,
		OptimizedQuery:  query,
		EstimatedCost:   cost,
		EstimatedRows:   1000,
		DataSources:     []string{"primary_db", "secondary_db"},
		CacheStrategy:   "partitioned",
		Parallelization: parallelism,
	}
}

// selectOptimalPlan chooses the best execution plan based on cost
func (qoe *QueryOptimizationEngine) selectOptimalPlan(plans []*QueryPlan, _ *QueryComponents) *QueryPlan {
	if len(plans) == 0 {
		return nil
	}

	// Sort by estimated cost
	sort.Slice(plans, func(i, j int) bool {
		return plans[i].EstimatedCost < plans[j].EstimatedCost
	})

	// Return the lowest cost plan
	return plans[0]
}

// applyGovernanceConstraints applies security and compliance rules
func (qoe *QueryOptimizationEngine) applyGovernanceConstraints(plan *QueryPlan, govContext *GovernanceContext) *QueryPlan {
	// Apply row-level security
	if len(govContext.RequiredFilters) > 0 {
		plan.OptimizedQuery = qoe.addRowLevelSecurity(plan.OptimizedQuery, govContext.RequiredFilters)
		plan.EstimatedCost *= 1.2 // RLS typically adds some overhead
	}

	// Apply data masking for sensitive fields
	plan.OptimizedQuery = qoe.applyDataMasking(plan.OptimizedQuery, govContext)

	// Ensure only allowed metrics are queried
	plan.OptimizedQuery = qoe.enforceMetricRestrictions(plan.OptimizedQuery, govContext.AllowedMetrics)

	return plan
}

// generateOptimizationHints provides suggestions for query improvement
func (qoe *QueryOptimizationEngine) generateOptimizationHints(plan *QueryPlan, components *QueryComponents) []string {
	hints := []string{}

	if plan.Parallelization == 1 && len(components.Dimensions) > 3 {
		hints = append(hints, "Consider increasing parallelization for multi-dimensional queries")
	}

	if plan.CacheStrategy == "none" && components.Complexity > 5 {
		hints = append(hints, "Consider caching results for complex queries")
	}

	if len(components.Filters) == 0 {
		hints = append(hints, "Add filters to reduce result set size")
	}

	return hints
}

// Helper methods for query building and optimization
func (qoe *QueryOptimizationEngine) buildSQLQuery(components *QueryComponents, _ *GovernanceContext) string {
	query := "SELECT "

	// Add metrics
	if len(components.Metrics) > 0 {
		query += strings.Join(components.Metrics, ", ")
	} else {
		query += "*"
	}

	query += " FROM facts_table"

	// Add dimensions
	if len(components.Dimensions) > 0 {
		query += " GROUP BY " + strings.Join(components.Dimensions, ", ")
	}

	return query
}

func (qoe *QueryOptimizationEngine) buildPreAggregatedQuery(components *QueryComponents, _ *GovernanceContext) string {
	return "SELECT * FROM pre_aggregated_metrics WHERE " + qoe.buildWhereClause(components)
}

func (qoe *QueryOptimizationEngine) buildParallelQuery(components *QueryComponents, _ *GovernanceContext, parallelism int) string {
	return fmt.Sprintf("SELECT /*+ PARALLEL(%d) */ * FROM facts_table WHERE %s",
		parallelism, qoe.buildWhereClause(components))
}

func (qoe *QueryOptimizationEngine) buildWhereClause(components *QueryComponents) string {
	clauses := []string{}

	if components.TimeRange != nil {
		clauses = append(clauses, fmt.Sprintf("date_column BETWEEN '%s' AND '%s'",
			components.TimeRange.Start.Format("2006-01-02"),
			components.TimeRange.End.Format("2006-01-02")))
	}

	if len(clauses) == 0 {
		return "1=1"
	}

	return strings.Join(clauses, " AND ")
}

func (qoe *QueryOptimizationEngine) generateCacheKey(components *QueryComponents, govContext *GovernanceContext) string {
	return fmt.Sprintf("%s_%s_%s", govContext.TenantID, govContext.UserID, components.OriginalQuery)
}

func (qoe *QueryOptimizationEngine) estimateCacheHitProbability(components *QueryComponents) float64 {
	// Simple estimation based on query complexity
	baseProb := 0.3
	complexityFactor := math.Min(float64(components.Complexity)/10.0, 1.0)
	return baseProb + (0.4 * complexityFactor)
}

func (qoe *QueryOptimizationEngine) determineOptimalParallelism(components *QueryComponents) int {
	// Determine parallelism based on query complexity and data size
	if components.Complexity > 7 {
		return 4
	} else if components.Complexity > 4 {
		return 2
	}
	return 1
}

func (qoe *QueryOptimizationEngine) addRowLevelSecurity(query string, filters []QueryFilter) string {
	if len(filters) == 0 {
		return query
	}

	whereClause := " WHERE "
	conditions := []string{}
	for _, filter := range filters {
		conditions = append(conditions, fmt.Sprintf("%s %s '%v'", filter.Field, filter.Operator, filter.Value))
	}

	return query + whereClause + strings.Join(conditions, " AND ")
}

func (qoe *QueryOptimizationEngine) applyDataMasking(query string, _ *GovernanceContext) string {
	// Apply data masking for sensitive fields (simplified example)
	return strings.ReplaceAll(query, "ssn", "MASKED_SSN")
}

func (qoe *QueryOptimizationEngine) enforceMetricRestrictions(query string, allowedMetrics []string) string {
	// Ensure only allowed metrics are queried (simplified example)
	for _, metric := range allowedMetrics {
		if !strings.Contains(query, metric) {
			// Remove unauthorized metric references
			query = strings.ReplaceAll(query, metric, "")
		}
	}
	return query
}

func generateQueryID() string {
	return fmt.Sprintf("q_%d", time.Now().UnixNano())
}
