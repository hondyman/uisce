package query

import (
	"fmt"
	"strings"
	"time"
)

// ClarificationEngine detects ambiguities and generates clarification questions
type ClarificationEngine struct {
	// Configuration for ambiguity detection
}

// NewClarificationEngine creates a new clarification engine
func NewClarificationEngine() *ClarificationEngine {
	return &ClarificationEngine{}
}

// DetectAmbiguities analyzes the parsed intent and governance context to identify ambiguities
func (ce *ClarificationEngine) DetectAmbiguities(intent *ParsedIntent, govCtx *GovernanceContext) []ClarificationQuestion {
	var questions []ClarificationQuestion

	// Check for ambiguous metrics
	metricQuestions := ce.detectAmbiguousMetrics(intent.Metrics, govCtx)
	questions = append(questions, metricQuestions...)

	// Check for ambiguous dimensions
	dimensionQuestions := ce.detectAmbiguousDimensions(intent.Dimensions, govCtx)
	questions = append(questions, dimensionQuestions...)

	// Check for ambiguous time ranges
	timeQuestions := ce.detectAmbiguousTimeRanges(intent.TimeRange)
	questions = append(questions, timeQuestions...)

	// Check for ambiguous filters
	filterQuestions := ce.detectAmbiguousFilters(intent.Filters)
	questions = append(questions, filterQuestions...)

	return questions
}

// detectAmbiguousMetrics identifies metrics that need clarification
func (ce *ClarificationEngine) detectAmbiguousMetrics(metrics []string, govCtx *GovernanceContext) []ClarificationQuestion {
	var questions []ClarificationQuestion

	for _, metric := range metrics {
		metricLower := strings.ToLower(metric)

		// Check for common ambiguous metric names
		switch {
		case strings.Contains(metricLower, "sales") && !strings.Contains(metricLower, "gross") && !strings.Contains(metricLower, "net"):
			questions = append(questions, ClarificationQuestion{
				ID:       generateQuestionID(),
				Question: fmt.Sprintf("For '%s', do you mean gross sales or net sales?", metric),
				Options:  []string{"gross sales", "net sales"},
				Field:    "metric",
				Required: true,
			})

		case strings.Contains(metricLower, "margin") && !strings.Contains(metricLower, "gross") && !strings.Contains(metricLower, "net"):
			questions = append(questions, ClarificationQuestion{
				ID:       generateQuestionID(),
				Question: fmt.Sprintf("For '%s', do you mean gross margin or net margin?", metric),
				Options:  []string{"gross margin", "net margin"},
				Field:    "metric",
				Required: true,
			})

		case strings.Contains(metricLower, "revenue") && !strings.Contains(metricLower, "total") && !strings.Contains(metricLower, "recurring"):
			questions = append(questions, ClarificationQuestion{
				ID:       generateQuestionID(),
				Question: fmt.Sprintf("For '%s', do you mean total revenue or recurring revenue?", metric),
				Options:  []string{"total revenue", "recurring revenue"},
				Field:    "metric",
				Required: true,
			})
		}

		// Check if metric is blocked and suggest alternatives
		if ce.isMetricBlocked(metric, govCtx) {
			questions = append(questions, ClarificationQuestion{
				ID:       generateQuestionID(),
				Question: fmt.Sprintf("'%s' is not available for your role. Would you like to use an alternative metric?", metric),
				Options:  ce.getAlternativeMetrics(metric, govCtx),
				Field:    "metric",
				Required: true,
			})
		}
	}

	return questions
}

// detectAmbiguousDimensions identifies dimensions that need clarification
func (ce *ClarificationEngine) detectAmbiguousDimensions(dimensions []string, govCtx *GovernanceContext) []ClarificationQuestion {
	var questions []ClarificationQuestion

	for _, dimension := range dimensions {
		dimensionLower := strings.ToLower(dimension)

		// Check for common ambiguous dimension names
		switch {
		case strings.Contains(dimensionLower, "region") && !strings.Contains(dimensionLower, "sales") && !strings.Contains(dimensionLower, "geo"):
			questions = append(questions, ClarificationQuestion{
				ID:       generateQuestionID(),
				Question: fmt.Sprintf("For '%s', do you mean sales region or geographic region?", dimension),
				Options:  []string{"sales region", "geographic region"},
				Field:    "dimension",
				Required: true,
			})

		case strings.Contains(dimensionLower, "category") && !strings.Contains(dimensionLower, "product") && !strings.Contains(dimensionLower, "customer"):
			questions = append(questions, ClarificationQuestion{
				ID:       generateQuestionID(),
				Question: fmt.Sprintf("For '%s', do you mean product category or customer category?", dimension),
				Options:  []string{"product category", "customer category"},
				Field:    "dimension",
				Required: true,
			})
		}

		// Check if dimension is blocked
		if ce.isDimensionBlocked(dimension, govCtx) {
			questions = append(questions, ClarificationQuestion{
				ID:       generateQuestionID(),
				Question: fmt.Sprintf("'%s' is not available for your role. Would you like to use an alternative dimension?", dimension),
				Options:  ce.getAlternativeDimensions(dimension, govCtx),
				Field:    "dimension",
				Required: true,
			})
		}
	}

	return questions
}

// detectAmbiguousTimeRanges identifies time range ambiguities
func (ce *ClarificationEngine) detectAmbiguousTimeRanges(timeRange *TimeRange) []ClarificationQuestion {
	var questions []ClarificationQuestion

	if timeRange == nil {
		return questions
	}

	timeRangeStr := strings.ToLower(timeRange.Label)

	// Check for ambiguous time references
	switch {
	case strings.Contains(timeRangeStr, "quarter") && !strings.Contains(timeRangeStr, "fiscal") && !strings.Contains(timeRangeStr, "calendar"):
		questions = append(questions, ClarificationQuestion{
			ID:       generateQuestionID(),
			Question: "Do you mean fiscal quarter or calendar quarter?",
			Options:  []string{"fiscal quarter", "calendar quarter"},
			Field:    "time_range",
			Required: true,
		})

	case strings.Contains(timeRangeStr, "year") && !strings.Contains(timeRangeStr, "fiscal") && !strings.Contains(timeRangeStr, "calendar"):
		questions = append(questions, ClarificationQuestion{
			ID:       generateQuestionID(),
			Question: "Do you mean fiscal year or calendar year?",
			Options:  []string{"fiscal year", "calendar year"},
			Field:    "time_range",
			Required: true,
		})

	case strings.Contains(timeRangeStr, "month") && !strings.Contains(timeRangeStr, "last") && !strings.Contains(timeRangeStr, "this"):
		questions = append(questions, ClarificationQuestion{
			ID:       generateQuestionID(),
			Question: "Do you mean last month or this month?",
			Options:  []string{"last month", "this month"},
			Field:    "time_range",
			Required: true,
		})
	}

	return questions
}

// detectAmbiguousFilters identifies filter ambiguities
func (ce *ClarificationEngine) detectAmbiguousFilters(filters []IntentFilter) []ClarificationQuestion {
	var questions []ClarificationQuestion

	for _, filter := range filters {
		filterFieldLower := strings.ToLower(filter.Field)

		// Check for ambiguous filter values
		switch {
		case strings.Contains(filterFieldLower, "status") && len(filter.Values) == 1:
			statusValue := strings.ToLower(filter.Values[0])
			if statusValue == "active" || statusValue == "inactive" {
				// This might be ambiguous - could mean different things in different contexts
				questions = append(questions, ClarificationQuestion{
					ID:       generateQuestionID(),
					Question: fmt.Sprintf("For status '%s', what type of status are you referring to?", filter.Values[0]),
					Options:  []string{"account status", "order status", "user status", "product status"},
					Field:    "filter",
					Required: false,
				})
			}
		}
	}

	return questions
}

// ProcessClarificationResponse processes user responses to clarification questions
func (ce *ClarificationEngine) ProcessClarificationResponse(currentIntent *ParsedIntent, questions []ClarificationQuestion, response string) (*ParsedIntent, error) {
	// Create a copy of the current intent
	updatedIntent := &ParsedIntent{
		Metrics:     make([]string, len(currentIntent.Metrics)),
		Dimensions:  make([]string, len(currentIntent.Dimensions)),
		Filters:     make([]IntentFilter, len(currentIntent.Filters)),
		TimeRange:   currentIntent.TimeRange,
		Aggregation: currentIntent.Aggregation,
		Confidence:  currentIntent.Confidence,
		RawEntities: make(map[string]string),
	}

	// Copy existing data
	copy(updatedIntent.Metrics, currentIntent.Metrics)
	copy(updatedIntent.Dimensions, currentIntent.Dimensions)
	copy(updatedIntent.Filters, currentIntent.Filters)
	for k, v := range currentIntent.RawEntities {
		updatedIntent.RawEntities[k] = v
	}

	// Parse the response and update intent accordingly
	// Simple response processing - in production this would be more sophisticated
	for _, question := range questions {
		switch question.Field {
		case "metric":
			// Replace ambiguous metrics with clarified ones
			for i, metric := range updatedIntent.Metrics {
				if ce.needsClarification(metric, question) {
					updatedIntent.Metrics[i] = ce.extractClarifiedValue(response, question.Options)
				}
			}

		case "dimension":
			// Replace ambiguous dimensions with clarified ones
			for i, dimension := range updatedIntent.Dimensions {
				if ce.needsClarification(dimension, question) {
					updatedIntent.Dimensions[i] = ce.extractClarifiedValue(response, question.Options)
				}
			}

		case "time_range":
			// Update time range based on clarification
			if updatedIntent.TimeRange != nil {
				clarifiedTimeRange := ce.extractClarifiedValue(response, question.Options)
				updatedIntent.TimeRange.Label = clarifiedTimeRange
			}
		}
	}

	// Increase confidence after clarification
	if updatedIntent.Confidence < 1.0 {
		updatedIntent.Confidence += 0.2
		if updatedIntent.Confidence > 1.0 {
			updatedIntent.Confidence = 1.0
		}
	}

	return updatedIntent, nil
}

// Helper methods

func (ce *ClarificationEngine) isMetricBlocked(metric string, govCtx *GovernanceContext) bool {
	// Check if metric is in allowed metrics list
	for _, allowed := range govCtx.AllowedMetrics {
		if strings.EqualFold(metric, allowed) {
			return false
		}
	}
	// If we have allowed metrics list and metric is not in it, consider it blocked
	return len(govCtx.AllowedMetrics) > 0
}

func (ce *ClarificationEngine) isDimensionBlocked(dimension string, govCtx *GovernanceContext) bool {
	// Check if dimension is in allowed dimensions list
	for _, allowed := range govCtx.AllowedDimensions {
		if strings.EqualFold(dimension, allowed) {
			return false
		}
	}
	// If we have allowed dimensions list and dimension is not in it, consider it blocked
	return len(govCtx.AllowedDimensions) > 0
}

func (ce *ClarificationEngine) getAlternativeMetrics(metric string, govCtx *GovernanceContext) []string {
	// Return alternative metrics that are allowed for the user
	metricLower := strings.ToLower(metric)

	var alternatives []string
	switch {
	case strings.Contains(metricLower, "net"):
		alternatives = []string{"gross_sales", "total_revenue"}
	case strings.Contains(metricLower, "gross"):
		alternatives = []string{"net_sales", "revenue"}
	case strings.Contains(metricLower, "margin"):
		alternatives = []string{"profit", "revenue"}
	default:
		alternatives = []string{"revenue", "sales", "profit"}
	}

	// Filter alternatives to only include allowed metrics
	var allowedAlternatives []string
	for _, alt := range alternatives {
		if !ce.isMetricBlocked(alt, govCtx) {
			allowedAlternatives = append(allowedAlternatives, alt)
		}
	}

	// If no alternatives are allowed, return a default allowed metric
	if len(allowedAlternatives) == 0 && len(govCtx.AllowedMetrics) > 0 {
		return []string{govCtx.AllowedMetrics[0]}
	}

	return allowedAlternatives
}

func (ce *ClarificationEngine) getAlternativeDimensions(dimension string, govCtx *GovernanceContext) []string {
	// Return alternative dimensions that are allowed for the user
	dimensionLower := strings.ToLower(dimension)

	var alternatives []string
	switch {
	case strings.Contains(dimensionLower, "region"):
		alternatives = []string{"country", "territory", "market"}
	case strings.Contains(dimensionLower, "category"):
		alternatives = []string{"type", "segment", "group"}
	default:
		alternatives = []string{"category", "type", "group"}
	}

	// Filter alternatives to only include allowed dimensions
	var allowedAlternatives []string
	for _, alt := range alternatives {
		if !ce.isDimensionBlocked(alt, govCtx) {
			allowedAlternatives = append(allowedAlternatives, alt)
		}
	}

	// If no alternatives are allowed, return a default allowed dimension
	if len(allowedAlternatives) == 0 && len(govCtx.AllowedDimensions) > 0 {
		return []string{govCtx.AllowedDimensions[0]}
	}

	return allowedAlternatives
}

func (ce *ClarificationEngine) needsClarification(field string, question ClarificationQuestion) bool {
	fieldLower := strings.ToLower(field)
	questionLower := strings.ToLower(question.Question)

	// Check if the field mentioned in the question matches this field
	return strings.Contains(questionLower, fieldLower)
}

func (ce *ClarificationEngine) extractClarifiedValue(response string, options []string) string {
	responseLower := strings.ToLower(response)

	// Try to match the response to one of the options
	for _, option := range options {
		if strings.Contains(responseLower, strings.ToLower(option)) {
			return option
		}
	}

	// If no match found, return the first option as default
	if len(options) > 0 {
		return options[0]
	}

	return response
}

func generateQuestionID() string {
	return fmt.Sprintf("q_%d", time.Now().UnixNano())
}
