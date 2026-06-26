package query

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/hondyman/semlayer/backend/internal/domain"
)

// SchemaLearner automatically discovers metrics and dimensions from schema
type SchemaLearner struct {
	schemaProvider    domain.SchemaProvider
	learnedMetrics    map[string]*LearnedMetric
	learnedDimensions map[string]*LearnedDimension
	lastUpdated       time.Time
}

// LearnedMetric represents a discovered metric
type LearnedMetric struct {
	Name         string
	SemanticName string
	DataType     string
	Description  string
	Confidence   float64
	LastUsed     time.Time
	UseCount     int
	Aggregations []string // SUM, AVG, COUNT, etc.
}

// LearnedDimension represents a discovered dimension
type LearnedDimension struct {
	Name         string
	SemanticName string
	DataType     string
	Description  string
	Confidence   float64
	LastUsed     time.Time
	UseCount     int
	Cardinality  int // Estimated number of distinct values
}

// SchemaLearningResult contains the results of schema learning
type SchemaLearningResult struct {
	NewMetrics        []*LearnedMetric
	NewDimensions     []*LearnedDimension
	UpdatedMetrics    []*LearnedMetric
	UpdatedDimensions []*LearnedDimension
	TotalLearned      int
	LastUpdated       time.Time
}

// NewSchemaLearner creates a new schema learner
func NewSchemaLearner(schemaProvider domain.SchemaProvider) *SchemaLearner {
	return &SchemaLearner{
		schemaProvider:    schemaProvider,
		learnedMetrics:    make(map[string]*LearnedMetric),
		learnedDimensions: make(map[string]*LearnedDimension),
	}
}

// LearnFromSchema analyzes the schema to discover metrics and dimensions
func (sl *SchemaLearner) LearnFromSchema(ctx context.Context, datasource string) (*SchemaLearningResult, error) {
	result := &SchemaLearningResult{
		NewMetrics:        []*LearnedMetric{},
		NewDimensions:     []*LearnedDimension{},
		UpdatedMetrics:    []*LearnedMetric{},
		UpdatedDimensions: []*LearnedDimension{},
		LastUpdated:       time.Now(),
	}

	// Get schema information
	schema, err := sl.schemaProvider.GetAssetSchema(datasource)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}

	// Analyze columns by scope
	for scope, columns := range schema.ColumnsByScope {
		switch scope {
		case "metrics":
			for _, column := range columns {
				metric := sl.analyzeMetricColumn(column)
				if existing, exists := sl.learnedMetrics[column]; exists {
					// Update existing metric
					sl.updateLearnedMetric(existing, metric)
					result.UpdatedMetrics = append(result.UpdatedMetrics, existing)
				} else {
					// Add new metric
					sl.learnedMetrics[column] = metric
					result.NewMetrics = append(result.NewMetrics, metric)
				}
			}
		case "dimensions":
			for _, column := range columns {
				dimension := sl.analyzeDimensionColumn(column)
				if existing, exists := sl.learnedDimensions[column]; exists {
					// Update existing dimension
					sl.updateLearnedDimension(existing, dimension)
					result.UpdatedDimensions = append(result.UpdatedDimensions, existing)
				} else {
					// Add new dimension
					sl.learnedDimensions[column] = dimension
					result.NewDimensions = append(result.NewDimensions, dimension)
				}
			}
		}
	}

	result.TotalLearned = len(sl.learnedMetrics) + len(sl.learnedDimensions)
	sl.lastUpdated = result.LastUpdated

	return result, nil
}

// analyzeMetricColumn analyzes a column to determine if it's a metric
func (sl *SchemaLearner) analyzeMetricColumn(columnName string) *LearnedMetric {
	metric := &LearnedMetric{
		Name:       columnName,
		DataType:   sl.inferDataType(columnName),
		Confidence: 0.8,
		LastUsed:   time.Now(),
		UseCount:   1,
	}

	// Generate semantic name
	metric.SemanticName = sl.generateSemanticName(columnName)

	// Infer possible aggregations based on naming patterns
	metric.Aggregations = sl.inferAggregations(columnName)

	// Generate description
	metric.Description = sl.generateMetricDescription(columnName)

	return metric
}

// analyzeDimensionColumn analyzes a column to determine if it's a dimension
func (sl *SchemaLearner) analyzeDimensionColumn(columnName string) *LearnedDimension {
	dimension := &LearnedDimension{
		Name:       columnName,
		DataType:   sl.inferDataType(columnName),
		Confidence: 0.8,
		LastUsed:   time.Now(),
		UseCount:   1,
	}

	// Generate semantic name
	dimension.SemanticName = sl.generateSemanticName(columnName)

	// Estimate cardinality based on naming patterns
	dimension.Cardinality = sl.estimateCardinality(columnName)

	// Generate description
	dimension.Description = sl.generateDimensionDescription(columnName)

	return dimension
}

// generateSemanticName creates a human-readable name from column name
func (sl *SchemaLearner) generateSemanticName(columnName string) string {
	// Convert snake_case to Title Case
	words := strings.Split(columnName, "_")
	tc := cases.Title(language.Und)
	for i, word := range words {
		words[i] = tc.String(strings.ToLower(word))
	}
	return strings.Join(words, " ")
}

// inferDataType infers the data type from column naming patterns
func (sl *SchemaLearner) inferDataType(columnName string) string {
	columnLower := strings.ToLower(columnName)

	// Numeric patterns
	if strings.Contains(columnLower, "amount") ||
		strings.Contains(columnLower, "value") ||
		strings.Contains(columnLower, "price") ||
		strings.Contains(columnLower, "cost") ||
		strings.Contains(columnLower, "revenue") ||
		strings.Contains(columnLower, "profit") ||
		strings.Contains(columnLower, "count") ||
		strings.Contains(columnLower, "quantity") ||
		strings.Contains(columnLower, "number") ||
		strings.HasPrefix(columnLower, "avg_") ||
		strings.HasPrefix(columnLower, "sum_") ||
		strings.HasPrefix(columnLower, "min_") ||
		strings.HasPrefix(columnLower, "max_") {
		return "numeric"
	}

	// Date patterns
	if strings.Contains(columnLower, "date") ||
		strings.Contains(columnLower, "time") ||
		strings.HasSuffix(columnLower, "_at") ||
		strings.HasSuffix(columnLower, "_on") {
		return "date"
	}

	// ID patterns
	if strings.HasSuffix(columnLower, "_id") ||
		strings.Contains(columnLower, "code") ||
		strings.Contains(columnLower, "key") {
		return "identifier"
	}

	// Default to text
	return "text"
}

// inferAggregations suggests possible aggregations for a metric
func (sl *SchemaLearner) inferAggregations(columnName string) []string {
	columnLower := strings.ToLower(columnName)
	aggregations := []string{}

	// Always allow COUNT
	aggregations = append(aggregations, "COUNT")

	// Numeric aggregations
	if sl.inferDataType(columnName) == "numeric" {
		if strings.Contains(columnLower, "amount") ||
			strings.Contains(columnLower, "value") ||
			strings.Contains(columnLower, "price") ||
			strings.Contains(columnLower, "cost") ||
			strings.Contains(columnLower, "revenue") ||
			strings.Contains(columnLower, "profit") {
			aggregations = append(aggregations, "SUM", "AVG", "MIN", "MAX")
		} else if strings.Contains(columnLower, "count") ||
			strings.Contains(columnLower, "quantity") ||
			strings.Contains(columnLower, "number") {
			aggregations = append(aggregations, "SUM", "AVG", "MIN", "MAX")
		}
	}

	return aggregations
}

// estimateCardinality estimates the number of distinct values
func (sl *SchemaLearner) estimateCardinality(columnName string) int {
	columnLower := strings.ToLower(columnName)

	// High cardinality patterns
	if strings.Contains(columnLower, "customer") ||
		strings.Contains(columnLower, "user") ||
		strings.Contains(columnLower, "email") ||
		strings.Contains(columnLower, "name") ||
		strings.HasSuffix(columnLower, "_id") {
		return 10000 // High cardinality
	}

	// Medium cardinality patterns
	if strings.Contains(columnLower, "category") ||
		strings.Contains(columnLower, "type") ||
		strings.Contains(columnLower, "status") ||
		strings.Contains(columnLower, "region") {
		return 50 // Medium cardinality
	}

	// Low cardinality patterns
	if strings.Contains(columnLower, "country") ||
		strings.Contains(columnLower, "state") ||
		strings.Contains(columnLower, "priority") {
		return 10 // Low cardinality
	}

	return 100 // Default
}

// generateMetricDescription creates a description for a metric
func (sl *SchemaLearner) generateMetricDescription(columnName string) string {
	semanticName := sl.generateSemanticName(columnName)
	return fmt.Sprintf("Metric representing %s", strings.ToLower(semanticName))
}

// generateDimensionDescription creates a description for a dimension
func (sl *SchemaLearner) generateDimensionDescription(columnName string) string {
	semanticName := sl.generateSemanticName(columnName)
	return fmt.Sprintf("Dimension for grouping by %s", strings.ToLower(semanticName))
}

// updateLearnedMetric updates an existing learned metric
func (sl *SchemaLearner) updateLearnedMetric(existing, new *LearnedMetric) {
	existing.UseCount++
	existing.LastUsed = time.Now()
	// Update confidence based on usage
	existing.Confidence = (existing.Confidence + new.Confidence) / 2.0
}

// updateLearnedDimension updates an existing learned dimension
func (sl *SchemaLearner) updateLearnedDimension(existing, new *LearnedDimension) {
	existing.UseCount++
	existing.LastUsed = time.Now()
	// Update confidence based on usage
	existing.Confidence = (existing.Confidence + new.Confidence) / 2.0
}

// FindMatchingMetrics finds metrics that match a semantic query
func (sl *SchemaLearner) FindMatchingMetrics(query string) []*LearnedMetric {
	var matches []*LearnedMetric
	queryLower := strings.ToLower(query)

	for _, metric := range sl.learnedMetrics {
		// Check semantic name match
		if strings.Contains(strings.ToLower(metric.SemanticName), queryLower) ||
			strings.Contains(strings.ToLower(metric.Name), queryLower) {
			matches = append(matches, metric)
		}
	}

	// Sort by confidence and usage
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Confidence != matches[j].Confidence {
			return matches[i].Confidence > matches[j].Confidence
		}
		return matches[i].UseCount > matches[j].UseCount
	})

	return matches
}

// FindMatchingDimensions finds dimensions that match a semantic query
func (sl *SchemaLearner) FindMatchingDimensions(query string) []*LearnedDimension {
	var matches []*LearnedDimension
	queryLower := strings.ToLower(query)

	for _, dimension := range sl.learnedDimensions {
		// Check semantic name match
		if strings.Contains(strings.ToLower(dimension.SemanticName), queryLower) ||
			strings.Contains(strings.ToLower(dimension.Name), queryLower) {
			matches = append(matches, dimension)
		}
	}

	// Sort by confidence and usage
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Confidence != matches[j].Confidence {
			return matches[i].Confidence > matches[j].Confidence
		}
		return matches[i].UseCount > matches[j].UseCount
	})

	return matches
}

// GetLearnedSchema returns the current learned schema
func (sl *SchemaLearner) GetLearnedSchema() map[string]interface{} {
	return map[string]interface{}{
		"metrics":       sl.learnedMetrics,
		"dimensions":    sl.learnedDimensions,
		"total_learned": len(sl.learnedMetrics) + len(sl.learnedDimensions),
		"last_updated":  sl.lastUpdated,
	}
}
