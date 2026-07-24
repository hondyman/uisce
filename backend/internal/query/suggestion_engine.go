package query

import (
	"fmt"
	"strings"
	"time"
)

// SuggestionEngine generates governance-aware suggestions for query refinement
type SuggestionEngine struct {
	// Configuration for suggestion generation
}

// NewSuggestionEngine creates a new suggestion engine
func NewSuggestionEngine() *SuggestionEngine {
	return &SuggestionEngine{}
}

// GenerateSuggestions analyzes the current intent, query, and governance context to generate helpful suggestions
func (se *SuggestionEngine) GenerateSuggestions(intent *ParsedIntent, govCtx *GovernanceContext, query *GeneratedQuery) []RefinementSuggestion {
	var suggestions []RefinementSuggestion

	// Generate metric-related suggestions
	metricSuggestions := se.generateMetricSuggestions(intent, govCtx, query)
	suggestions = append(suggestions, metricSuggestions...)

	// Generate dimension-related suggestions
	dimensionSuggestions := se.generateDimensionSuggestions(intent, govCtx, query)
	suggestions = append(suggestions, dimensionSuggestions...)

	// Generate filter suggestions
	filterSuggestions := se.generateFilterSuggestions(intent, govCtx, query)
	suggestions = append(suggestions, filterSuggestions...)

	// Generate time range suggestions
	timeSuggestions := se.generateTimeRangeSuggestions(intent, govCtx, query)
	suggestions = append(suggestions, timeSuggestions...)

	// Generate aggregation suggestions
	aggregationSuggestions := se.generateAggregationSuggestions(intent, govCtx, query)
	suggestions = append(suggestions, aggregationSuggestions...)

	return suggestions
}

// generateMetricSuggestions generates suggestions related to metrics
func (se *SuggestionEngine) generateMetricSuggestions(intent *ParsedIntent, govCtx *GovernanceContext, _ *GeneratedQuery) []RefinementSuggestion {
	var suggestions []RefinementSuggestion

	// Suggest certified alternatives for uncertified metrics
	for _, metric := range intent.Metrics {
		if !se.isMetricAllowed(metric, govCtx) {
			alternatives := se.getCertifiedAlternatives(metric, govCtx)
			for _, alt := range alternatives {
				suggestions = append(suggestions, RefinementSuggestion{
					ID:          generateSuggestionID(),
					Type:        "metric",
					Description: fmt.Sprintf("Replace '%s' with certified metric '%s'", metric, alt),
					Action:      "replace",
					Value:       alt,
					Reason:      "This metric is certified and available for your role",
					Confidence:  0.9,
				})
			}
		}
	}

	// Suggest related metrics that might be useful
	if len(intent.Metrics) > 0 {
		relatedMetrics := se.getRelatedMetrics(intent.Metrics[0], govCtx)
		for _, related := range relatedMetrics {
			if !se.containsMetric(intent.Metrics, related) {
				suggestions = append(suggestions, RefinementSuggestion{
					ID:          generateSuggestionID(),
					Type:        "metric",
					Description: fmt.Sprintf("Add related metric '%s' for better analysis", related),
					Action:      "add",
					Value:       related,
					Reason:      "This metric is often analyzed together with your selected metrics",
					Confidence:  0.7,
				})
			}
		}
	}

	return suggestions
}

// generateDimensionSuggestions generates suggestions related to dimensions
func (se *SuggestionEngine) generateDimensionSuggestions(intent *ParsedIntent, govCtx *GovernanceContext, _ *GeneratedQuery) []RefinementSuggestion {
	var suggestions []RefinementSuggestion

	// Suggest certified alternatives for blocked dimensions
	for _, dimension := range intent.Dimensions {
		if !se.isDimensionAllowed(dimension, govCtx) {
			alternatives := se.getCertifiedDimensionAlternatives(dimension, govCtx)
			for _, alt := range alternatives {
				suggestions = append(suggestions, RefinementSuggestion{
					ID:          generateSuggestionID(),
					Type:        "dimension",
					Description: fmt.Sprintf("Replace '%s' with certified dimension '%s'", dimension, alt),
					Action:      "replace",
					Value:       alt,
					Reason:      "This dimension is certified and available for your role",
					Confidence:  0.9,
				})
			}
		}
	}

	// Suggest grouping dimensions for better analysis
	if len(intent.Metrics) > 0 && len(intent.Dimensions) == 0 {
		suggestions = append(suggestions, RefinementSuggestion{
			ID:          generateSuggestionID(),
			Type:        "dimension",
			Description: "Add a dimension to group your results (e.g., by region, category, or time period)",
			Action:      "add",
			Value:       se.suggestGroupingDimension(intent.Metrics, govCtx),
			Reason:      "Grouping by dimensions provides more meaningful insights",
			Confidence:  0.8,
		})
	}

	return suggestions
}

// generateFilterSuggestions generates suggestions for adding filters
func (se *SuggestionEngine) generateFilterSuggestions(intent *ParsedIntent, govCtx *GovernanceContext, query *GeneratedQuery) []RefinementSuggestion {
	var suggestions []RefinementSuggestion

	// Suggest date filters if no time range is specified
	if intent.TimeRange == nil {
		suggestions = append(suggestions, RefinementSuggestion{
			ID:          generateSuggestionID(),
			Type:        "filter",
			Description: "Add a time filter to focus on specific date ranges",
			Action:      "add",
			Value: map[string]interface{}{
				"field":    "date",
				"operator": "between",
				"value":    "last_30_days",
			},
			Reason:     "Time filters help narrow down results and improve query performance",
			Confidence: 0.8,
		})
	}

	// Suggest tenant/organization filters if not present
	hasTenantFilter := false
	for _, filter := range intent.Filters {
		if strings.Contains(strings.ToLower(filter.Field), "tenant") ||
			strings.Contains(strings.ToLower(filter.Field), "organization") {
			hasTenantFilter = true
			break
		}
	}

	if !hasTenantFilter && len(govCtx.RequiredFilters) == 0 {
		suggestions = append(suggestions, RefinementSuggestion{
			ID:          generateSuggestionID(),
			Type:        "filter",
			Description: "Add tenant filter to ensure data security",
			Action:      "add",
			Value: map[string]interface{}{
				"field":    "tenant_id",
				"operator": "equals",
				"value":    govCtx.TenantID,
			},
			Reason:     "Tenant filters ensure you only see data for your organization",
			Confidence: 0.95,
		})
	}

	// Suggest performance optimization filters
	if se.shouldSuggestPerformanceFilter(intent, query) {
		suggestions = append(suggestions, RefinementSuggestion{
			ID:          generateSuggestionID(),
			Type:        "filter",
			Description: "Add filter to improve query performance",
			Action:      "add",
			Value:       se.suggestPerformanceFilter(intent),
			Reason:      "This filter will significantly improve query execution time",
			Confidence:  0.85,
		})
	}

	return suggestions
}

// generateTimeRangeSuggestions generates suggestions for time range refinements
func (se *SuggestionEngine) generateTimeRangeSuggestions(intent *ParsedIntent, _ *GovernanceContext, _ *GeneratedQuery) []RefinementSuggestion {
	var suggestions []RefinementSuggestion

	if intent.TimeRange != nil {
		timeRangeStr := strings.ToLower(intent.TimeRange.Label)

		// Suggest more specific time ranges
		if strings.Contains(timeRangeStr, "last year") {
			suggestions = append(suggestions, RefinementSuggestion{
				ID:          generateSuggestionID(),
				Type:        "time_range",
				Description: "Consider using fiscal year instead of calendar year",
				Action:      "replace",
				Value:       "last_fiscal_year",
				Reason:      "Fiscal year aligns better with business reporting cycles",
				Confidence:  0.7,
			})
		}

		// Suggest comparison periods
		if !strings.Contains(timeRangeStr, "vs") && !strings.Contains(timeRangeStr, "compared") {
			suggestions = append(suggestions, RefinementSuggestion{
				ID:          generateSuggestionID(),
				Type:        "time_range",
				Description: "Compare with previous period for trend analysis",
				Action:      "add",
				Value:       se.suggestComparisonPeriod(intent.TimeRange),
				Reason:      "Period-over-period comparisons provide valuable insights",
				Confidence:  0.75,
			})
		}
	}

	return suggestions
}

// generateAggregationSuggestions generates suggestions for aggregation refinements
func (se *SuggestionEngine) generateAggregationSuggestions(intent *ParsedIntent, _ *GovernanceContext, _ *GeneratedQuery) []RefinementSuggestion {
	var suggestions []RefinementSuggestion

	// Suggest aggregation if no aggregation is specified and we have dimensions
	if intent.Aggregation == "" && len(intent.Dimensions) > 0 {
		suggestions = append(suggestions, RefinementSuggestion{
			ID:          generateSuggestionID(),
			Type:        "aggregation",
			Description: "Add aggregation to summarize data by your selected dimensions",
			Action:      "add",
			Value:       "sum", // Default to sum for metrics
			Reason:      "Aggregation provides summarized insights across your dimensions",
			Confidence:  0.8,
		})
	}

	// Suggest alternative aggregations
	if intent.Aggregation != "" {
		alternatives := se.getAlternativeAggregations(intent.Aggregation, intent.Metrics)
		for _, alt := range alternatives {
			suggestions = append(suggestions, RefinementSuggestion{
				ID:          generateSuggestionID(),
				Type:        "aggregation",
				Description: fmt.Sprintf("Try '%s' aggregation instead of '%s'", alt, intent.Aggregation),
				Action:      "replace",
				Value:       alt,
				Reason:      fmt.Sprintf("'%s' might provide different insights for your analysis", alt),
				Confidence:  0.6,
			})
		}
	}

	return suggestions
}

// ProcessSuggestionResponse processes user responses to suggestions
func (se *SuggestionEngine) ProcessSuggestionResponse(
	currentIntent *ParsedIntent,
	currentQuery *GeneratedQuery,
	suggestions []RefinementSuggestion,
	response string,
) (*ParsedIntent, *GeneratedQuery, error) {

	// Create copies of current intent and query
	updatedIntent := se.copyIntent(currentIntent)
	updatedQuery := se.copyQuery(currentQuery)

	// Parse user response to determine which suggestions to apply
	responseLower := strings.ToLower(response)

	// Simple response processing - in production this would use NLP
	for _, suggestion := range suggestions {
		if se.responseMatchesSuggestion(responseLower, suggestion) {
			err := se.applySuggestion(updatedIntent, updatedQuery, suggestion)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to apply suggestion %s: %w", suggestion.ID, err)
			}
		}
	}

	return updatedIntent, updatedQuery, nil
}

// Helper methods

func (se *SuggestionEngine) isMetricAllowed(metric string, govCtx *GovernanceContext) bool {
	if len(govCtx.AllowedMetrics) == 0 {
		return true // No restrictions
	}
	for _, allowed := range govCtx.AllowedMetrics {
		if strings.EqualFold(metric, allowed) {
			return true
		}
	}
	return false
}

func (se *SuggestionEngine) isDimensionAllowed(dimension string, govCtx *GovernanceContext) bool {
	if len(govCtx.AllowedDimensions) == 0 {
		return true // No restrictions
	}
	for _, allowed := range govCtx.AllowedDimensions {
		if strings.EqualFold(dimension, allowed) {
			return true
		}
	}
	return false
}

func (se *SuggestionEngine) getCertifiedAlternatives(metric string, _ *GovernanceContext) []string {
	metricLower := strings.ToLower(metric)

	// Return certified alternatives based on metric type
	switch {
	case strings.Contains(metricLower, "sales"):
		return []string{"certified_sales", "approved_revenue"}
	case strings.Contains(metricLower, "margin"):
		return []string{"certified_margin", "approved_profit_margin"}
	case strings.Contains(metricLower, "cost"):
		return []string{"certified_cost", "approved_expenses"}
	default:
		return []string{"certified_metric"}
	}
}

func (se *SuggestionEngine) getRelatedMetrics(metric string, _ *GovernanceContext) []string {
	metricLower := strings.ToLower(metric)

	switch {
	case strings.Contains(metricLower, "sales"):
		return []string{"revenue", "profit", "margin"}
	case strings.Contains(metricLower, "revenue"):
		return []string{"sales", "profit", "margin"}
	case strings.Contains(metricLower, "cost"):
		return []string{"expense", "budget", "variance"}
	default:
		return []string{"count", "total"}
	}
}

func (se *SuggestionEngine) containsMetric(metrics []string, target string) bool {
	for _, metric := range metrics {
		if strings.EqualFold(metric, target) {
			return true
		}
	}
	return false
}

func (se *SuggestionEngine) getCertifiedDimensionAlternatives(dimension string, _ *GovernanceContext) []string {
	dimensionLower := strings.ToLower(dimension)

	switch {
	case strings.Contains(dimensionLower, "region"):
		return []string{"certified_region", "approved_territory"}
	case strings.Contains(dimensionLower, "category"):
		return []string{"certified_category", "approved_segment"}
	default:
		return []string{"certified_dimension"}
	}
}

func (se *SuggestionEngine) suggestGroupingDimension(metrics []string, _ *GovernanceContext) string {
	// Suggest common grouping dimensions based on metrics
	for _, metric := range metrics {
		metricLower := strings.ToLower(metric)
		if strings.Contains(metricLower, "sales") || strings.Contains(metricLower, "revenue") {
			return "region"
		}
		if strings.Contains(metricLower, "cost") || strings.Contains(metricLower, "expense") {
			return "department"
		}
	}
	return "category"
}

func (se *SuggestionEngine) shouldSuggestPerformanceFilter(intent *ParsedIntent, _query *GeneratedQuery) bool {
	_ = _query // Mark parameter as intentionally used
	// Suggest performance filters if query might be too broad
	return len(intent.Dimensions) == 0 && intent.TimeRange == nil
}

func (se *SuggestionEngine) suggestPerformanceFilter(_intent *ParsedIntent) map[string]interface{} {
	_ = _intent // Mark parameter as intentionally used
	return map[string]interface{}{
		"field":    "status",
		"operator": "equals",
		"value":    "active",
	}
}

func (se *SuggestionEngine) suggestComparisonPeriod(timeRange *TimeRange) string {
	// Suggest a comparison period based on current time range
	if timeRange != nil {
		timeRangeStr := strings.ToLower(timeRange.Label)
		if strings.Contains(timeRangeStr, "last month") {
			return "previous_month"
		}
		if strings.Contains(timeRangeStr, "last quarter") {
			return "previous_quarter"
		}
		if strings.Contains(timeRangeStr, "last year") {
			return "previous_year"
		}
	}
	return "previous_period"
}

func (se *SuggestionEngine) getAlternativeAggregations(currentAgg string, _ []string) []string {
	currentLower := strings.ToLower(currentAgg)

	// Suggest alternative aggregations based on current one
	switch currentLower {
	case "sum":
		return []string{"avg", "count"}
	case "avg":
		return []string{"sum", "median"}
	case "count":
		return []string{"sum", "distinct_count"}
	default:
		return []string{"sum", "avg", "count"}
	}
}

func (se *SuggestionEngine) copyIntent(intent *ParsedIntent) *ParsedIntent {
	return &ParsedIntent{
		Metrics:     append([]string{}, intent.Metrics...),
		Dimensions:  append([]string{}, intent.Dimensions...),
		Filters:     append([]IntentFilter{}, intent.Filters...),
		TimeRange:   intent.TimeRange,
		Aggregation: intent.Aggregation,
		Confidence:  intent.Confidence,
		RawEntities: se.copyStringMap(intent.RawEntities),
	}
}

func (se *SuggestionEngine) copyQuery(query *GeneratedQuery) *GeneratedQuery {
	return &GeneratedQuery{
		SQL:         query.SQL,
		SemanticSQL: query.SemanticSQL,
		Measures:    append([]string{}, query.Measures...),
		Dimensions:  append([]string{}, query.Dimensions...),
		Filters:     append([]QueryFilter{}, query.Filters...),
		OrderBy:     append([]OrderBySpec{}, query.OrderBy...),
	}
}

func (se *SuggestionEngine) copyStringMap(original map[string]string) map[string]string {
	copy := make(map[string]string)
	for k, v := range original {
		copy[k] = v
	}
	return copy
}

func (se *SuggestionEngine) responseMatchesSuggestion(response string, suggestion RefinementSuggestion) bool {
	// Simple matching - in production this would use more sophisticated NLP
	responseLower := strings.ToLower(response)
	descLower := strings.ToLower(suggestion.Description)

	// Check if response contains key words from suggestion
	keywords := []string{"yes", "sure", "okay", "add", "replace", "use", "try"}
	for _, keyword := range keywords {
		if strings.Contains(responseLower, keyword) {
			return true
		}
	}

	// Check if response mentions the suggestion value
	if value, ok := suggestion.Value.(string); ok {
		if strings.Contains(responseLower, strings.ToLower(value)) {
			return true
		}
	}

	// Check if response contains parts of the description
	words := strings.Fields(descLower)
	for _, word := range words {
		if len(word) > 3 && strings.Contains(responseLower, word) {
			return true
		}
	}

	return false
}

func (se *SuggestionEngine) applySuggestion(intent *ParsedIntent, query *GeneratedQuery, suggestion RefinementSuggestion) error {
	switch suggestion.Type {
	case "metric":
		return se.applyMetricSuggestion(intent, query, suggestion)
	case "dimension":
		return se.applyDimensionSuggestion(intent, query, suggestion)
	case "filter":
		return se.applyFilterSuggestion(intent, query, suggestion)
	case "time_range":
		return se.applyTimeRangeSuggestion(intent, query, suggestion)
	case "aggregation":
		return se.applyAggregationSuggestion(intent, query, suggestion)
	default:
		return fmt.Errorf("unknown suggestion type: %s", suggestion.Type)
	}
}

func (se *SuggestionEngine) applyMetricSuggestion(intent *ParsedIntent, query *GeneratedQuery, suggestion RefinementSuggestion) error {
	switch suggestion.Action {
	case "replace":
		// Replace the first metric (simplified - in production would be more sophisticated)
		if len(intent.Metrics) > 0 && suggestion.Value != nil {
			if value, ok := suggestion.Value.(string); ok {
				intent.Metrics[0] = value
				query.Measures[0] = value
			}
		}
	case "add":
		if value, ok := suggestion.Value.(string); ok {
			intent.Metrics = append(intent.Metrics, value)
			query.Measures = append(query.Measures, value)
		}
	}
	return nil
}

func (se *SuggestionEngine) applyDimensionSuggestion(intent *ParsedIntent, query *GeneratedQuery, suggestion RefinementSuggestion) error {
	switch suggestion.Action {
	case "replace":
		if len(intent.Dimensions) > 0 && suggestion.Value != nil {
			if value, ok := suggestion.Value.(string); ok {
				intent.Dimensions[0] = value
				query.Dimensions[0] = value
			}
		}
	case "add":
		if value, ok := suggestion.Value.(string); ok {
			intent.Dimensions = append(intent.Dimensions, value)
			query.Dimensions = append(query.Dimensions, value)
		}
	}
	return nil
}

func (se *SuggestionEngine) applyFilterSuggestion(intent *ParsedIntent, query *GeneratedQuery, suggestion RefinementSuggestion) error {
	if suggestion.Action == "add" && suggestion.Value != nil {
		if filterMap, ok := suggestion.Value.(map[string]interface{}); ok {
			field, _ := filterMap["field"].(string)
			operator, _ := filterMap["operator"].(string)
			value, _ := filterMap["value"].(string)

			intentFilter := IntentFilter{
				Field:    field,
				Operator: operator,
				Values:   []string{value},
			}
			intent.Filters = append(intent.Filters, intentFilter)

			queryFilter := QueryFilter{
				Field:    field,
				Operator: operator,
				Value:    value,
			}
			query.Filters = append(query.Filters, queryFilter)
		}
	}
	return nil
}

func (se *SuggestionEngine) applyTimeRangeSuggestion(intent *ParsedIntent, _ *GeneratedQuery, suggestion RefinementSuggestion) error {
	if suggestion.Value != nil {
		if value, ok := suggestion.Value.(string); ok {
			if intent.TimeRange == nil {
				intent.TimeRange = &TimeRange{Label: value}
			} else {
				intent.TimeRange.Label = value
			}
		}
	}
	return nil
}

func (se *SuggestionEngine) applyAggregationSuggestion(intent *ParsedIntent, _ *GeneratedQuery, suggestion RefinementSuggestion) error {
	if suggestion.Value != nil {
		if value, ok := suggestion.Value.(string); ok {
			intent.Aggregation = value
		}
	}
	return nil
}

func generateSuggestionID() string {
	return fmt.Sprintf("sugg_%d", time.Now().UnixNano())
}
