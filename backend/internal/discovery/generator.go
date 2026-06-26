package discovery

import (
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/hondyman/semlayer/backend/internal/models"
)

// FeatureGenerator creates derived and transformed features from candidates
type FeatureGenerator struct {
	logger *log.Logger
}

// TransformationType represents a feature transformation
type TransformationType string

const (
	// Aggregations
	AggSum    TransformationType = "sum"
	AggAvg    TransformationType = "avg"
	AggMin    TransformationType = "min"
	AggMax    TransformationType = "max"
	AggCount  TransformationType = "count"
	AggStdDev TransformationType = "stddev"
	AggMedian TransformationType = "median"

	// Time-series transformations
	TransLag      TransformationType = "lag"      // Previous value
	TransDelta    TransformationType = "delta"    // Difference from previous
	TransRate     TransformationType = "rate"     // Change per time unit
	TransRolling  TransformationType = "rolling"  // Rolling window stats
	TransFFT      TransformationType = "fft"      // Fourier features
	TransTrend    TransformationType = "trend"    // Trend extraction
	TransSeasonal TransformationType = "seasonal" // Seasonal decomposition

	// Encoding
	TransOneHot  TransformationType = "onehot"  // One-hot encoding
	TransBinary  TransformationType = "binary"  // Binary encoding
	TransOrdinal TransformationType = "ordinal" // Ordinal encoding
	TransHash    TransformationType = "hash"    // Hash encoding

	// Math operations
	TransLog        TransformationType = "log"        // Log transform
	TransSqrt       TransformationType = "sqrt"       // Square root
	TransSquare     TransformationType = "square"     // Squared
	TransReciprocal TransformationType = "reciprocal" // 1/x
	TransNormalize  TransformationType = "normalize"  // Z-score normalization

	// Interactions
	TransInteraction TransformationType = "interaction" // Feature × Feature
	TransRatio       TransformationType = "ratio"       // Feature / Feature
	TransCombination TransformationType = "combination" // Feature + Feature or other ops
)

// DerivedFeature represents a feature created from one or more source features
type DerivedFeature struct {
	Name              string
	TransformType     TransformationType
	SourceFeatures    []string
	Parameters        map[string]interface{} // windowsize, lag, threshold, etc
	Description       string
	ComputeComplexity string  // "simple", "medium", "complex"
	Stability         float64 // 0-1: how stable across time
	Importance        float64 // 0-1: expected feature importance
}

// NewFeatureGenerator creates a new feature generator
func NewFeatureGenerator(logger *log.Logger) *FeatureGenerator {
	return &FeatureGenerator{
		logger: logger,
	}
}

// GenerateAggregations creates aggregate features (sum, avg, etc)
func (fg *FeatureGenerator) GenerateAggregations(candidates []models.FeatureCandidate) []DerivedFeature {
	var derived []DerivedFeature

	// Find numeric candidates suitable for aggregation
	numericCandidates := make([]models.FeatureCandidate, 0)
	for _, c := range candidates {
		if c.DataType == "number" || c.DataType == "float" {
			numericCandidates = append(numericCandidates, c)
		}
	}

	for _, candidate := range numericCandidates {
		// Generate aggregations with different windows
		windows := []int{60, 300, 900, 3600} // 1min, 5min, 15min, 1hour

		for _, window := range windows {
			for _, aggType := range []TransformationType{AggAvg, AggSum, AggMax, AggMin, AggStdDev} {
				df := DerivedFeature{
					Name:           fmt.Sprintf("%s_%s_%dm", candidate.Name, aggType, window/60),
					TransformType:  aggType,
					SourceFeatures: []string{candidate.Name},
					Parameters: map[string]interface{}{
						"window_seconds": window,
					},
					Description:       fmt.Sprintf("%s of %s over %d seconds", aggType, candidate.Name, window),
					ComputeComplexity: "simple",
					Stability:         0.9,
					Importance:        candidate.BusinessValue * 0.8, // Slightly lower than base
				}
				derived = append(derived, df)
			}
		}
	}

	return derived
}

// GenerateTimeSeriesFeatures creates time-series specific features (lags, rolling, etc)
func (fg *FeatureGenerator) GenerateTimeSeriesFeatures(candidates []models.FeatureCandidate) []DerivedFeature {
	var derived []DerivedFeature

	// Numeric candidates suitable for time-series
	numericCandidates := make([]models.FeatureCandidate, 0)
	for _, c := range candidates {
		if c.DataType == "number" || c.DataType == "float" || c.SourceDatabase == "prometheus" {
			numericCandidates = append(numericCandidates, c)
		}
	}

	for _, candidate := range numericCandidates {
		// 1. Lag features
		for _, lag := range []int{1, 7, 30} {
			df := DerivedFeature{
				Name:           fmt.Sprintf("%s_lag_%d", candidate.Name, lag),
				TransformType:  TransLag,
				SourceFeatures: []string{candidate.Name},
				Parameters: map[string]interface{}{
					"lag_steps": lag,
				},
				Description:       fmt.Sprintf("Value of %s from %d time steps ago", candidate.Name, lag),
				ComputeComplexity: "simple",
				Stability:         0.8,
				Importance:        candidate.BusinessValue * 0.7,
			}
			derived = append(derived, df)
		}

		// 2. Delta (difference from previous)
		df := DerivedFeature{
			Name:           fmt.Sprintf("%s_delta", candidate.Name),
			TransformType:  TransDelta,
			SourceFeatures: []string{candidate.Name},
			Parameters: map[string]interface{}{
				"order": 1,
			},
			Description:       fmt.Sprintf("First-order difference of %s", candidate.Name),
			ComputeComplexity: "simple",
			Stability:         0.7,
			Importance:        candidate.BusinessValue * 0.6,
		}
		derived = append(derived, df)

		// 3. Rolling statistics
		for _, window := range []int{7, 30} {
			for _, stat := range []string{"mean", "std", "min", "max"} {
				df := DerivedFeature{
					Name:           fmt.Sprintf("%s_rolling_%s_%d", candidate.Name, stat, window),
					TransformType:  TransRolling,
					SourceFeatures: []string{candidate.Name},
					Parameters: map[string]interface{}{
						"window_size": window,
						"statistic":   stat,
					},
					Description:       fmt.Sprintf("Rolling %s of %s over %d periods", stat, candidate.Name, window),
					ComputeComplexity: "medium",
					Stability:         0.85,
					Importance:        candidate.BusinessValue * 0.75,
				}
				derived = append(derived, df)
			}
		}

		// 4. Rate of change
		df = DerivedFeature{
			Name:           fmt.Sprintf("%s_rate_5m", candidate.Name),
			TransformType:  TransRate,
			SourceFeatures: []string{candidate.Name},
			Parameters: map[string]interface{}{
				"window_minutes": 5,
			},
			Description:       fmt.Sprintf("Rate of change of %s over 5 minutes", candidate.Name),
			ComputeComplexity: "simple",
			Stability:         0.6,
			Importance:        candidate.BusinessValue * 0.8, // Useful for detecting changes
		}
		derived = append(derived, df)
	}

	return derived
}

// GenerateCategoricalFeatures creates categorical encoding features
func (fg *FeatureGenerator) GenerateCategoricalFeatures(candidates []models.FeatureCandidate) []DerivedFeature {
	var derived []DerivedFeature

	// String and categorical candidates
	categoricalCandidates := make([]models.FeatureCandidate, 0)
	for _, c := range candidates {
		if c.DataType == "string" || c.DataType == "categorical" {
			categoricalCandidates = append(categoricalCandidates, c)
		}
	}

	for _, candidate := range categoricalCandidates {
		// Only one-hot encode if cardinality is reasonable (< 50)
		if candidate.Cardinality > 0 && candidate.Cardinality < 50 {
			df := DerivedFeature{
				Name:           fmt.Sprintf("%s_onehot", candidate.Name),
				TransformType:  TransOneHot,
				SourceFeatures: []string{candidate.Name},
				Parameters: map[string]interface{}{
					"max_categories": 50,
				},
				Description:       fmt.Sprintf("One-hot encoding of %s", candidate.Name),
				ComputeComplexity: "simple",
				Stability:         0.9,
				Importance:        candidate.BusinessValue * 0.6,
			}
			derived = append(derived, df)
		} else {
			// Fallback: ordinal encoding for high cardinality
			df := DerivedFeature{
				Name:           fmt.Sprintf("%s_ordinal", candidate.Name),
				TransformType:  TransOrdinal,
				SourceFeatures: []string{candidate.Name},
				Parameters: map[string]interface{}{
					"strategy": "frequency",
				},
				Description:       fmt.Sprintf("Ordinal encoding of %s by frequency", candidate.Name),
				ComputeComplexity: "medium",
				Stability:         0.7,
				Importance:        candidate.BusinessValue * 0.5,
			}
			derived = append(derived, df)
		}
	}

	return derived
}

// GenerateMathTransforms creates log, sqrt, and other math transformations
func (fg *FeatureGenerator) GenerateMathTransforms(candidates []models.FeatureCandidate) []DerivedFeature {
	var derived []DerivedFeature

	numericCandidates := make([]models.FeatureCandidate, 0)
	for _, c := range candidates {
		if c.DataType == "number" || c.DataType == "float" {
			numericCandidates = append(numericCandidates, c)
		}
	}

	for _, candidate := range numericCandidates {
		// Log transform (for right-skewed distributions)
		df := DerivedFeature{
			Name:           fmt.Sprintf("%s_log", candidate.Name),
			TransformType:  TransLog,
			SourceFeatures: []string{candidate.Name},
			Parameters: map[string]interface{}{
				"base":        math.E,
				"handle_zero": 0.0001,
			},
			Description:       fmt.Sprintf("Natural log of %s (with offset for zeros)", candidate.Name),
			ComputeComplexity: "simple",
			Stability:         0.8,
			Importance:        candidate.BusinessValue * 0.7,
		}
		derived = append(derived, df)

		// Square root transform
		df = DerivedFeature{
			Name:              fmt.Sprintf("%s_sqrt", candidate.Name),
			TransformType:     TransSqrt,
			SourceFeatures:    []string{candidate.Name},
			Parameters:        map[string]interface{}{},
			Description:       fmt.Sprintf("Square root of %s", candidate.Name),
			ComputeComplexity: "simple",
			Stability:         0.8,
			Importance:        candidate.BusinessValue * 0.6,
		}
		derived = append(derived, df)

		// Normalization (z-score)
		df = DerivedFeature{
			Name:           fmt.Sprintf("%s_normalized", candidate.Name),
			TransformType:  TransNormalize,
			SourceFeatures: []string{candidate.Name},
			Parameters: map[string]interface{}{
				"method": "zscore",
			},
			Description:       fmt.Sprintf("Z-score normalized %s", candidate.Name),
			ComputeComplexity: "simple",
			Stability:         0.85,
			Importance:        candidate.BusinessValue * 0.8,
		}
		derived = append(derived, df)
	}

	return derived
}

// GenerateInteractions creates feature interactions (only top pairs)
func (fg *FeatureGenerator) GenerateInteractions(candidates []models.FeatureCandidate, topN int) []DerivedFeature {
	var derived []DerivedFeature

	// Only consider top candidates
	var selected []models.FeatureCandidate
	if len(candidates) > topN {
		selected = candidates[:topN]
	} else {
		selected = candidates
	}

	// Get numeric candidates only
	numericCandidates := make([]models.FeatureCandidate, 0)
	for _, c := range selected {
		if c.DataType == "number" || c.DataType == "float" {
			numericCandidates = append(numericCandidates, c)
		}
	}

	// Generate pairwise interactions (limit to avoid explosion)
	interactionCount := 0
	maxInteractions := 20

	for i := 0; i < len(numericCandidates) && interactionCount < maxInteractions; i++ {
		for j := i + 1; j < len(numericCandidates) && interactionCount < maxInteractions; j++ {
			c1 := numericCandidates[i]
			c2 := numericCandidates[j]

			// Pair multiplication (most common)
			df := DerivedFeature{
				Name:           fmt.Sprintf("%s_x_%s", c1.Name, c2.Name),
				TransformType:  TransInteraction,
				SourceFeatures: []string{c1.Name, c2.Name},
				Parameters: map[string]interface{}{
					"operation": "multiply",
				},
				Description:       fmt.Sprintf("Interaction: %s × %s", c1.Name, c2.Name),
				ComputeComplexity: "simple",
				Stability:         math.Min(c1.BusinessValue, c2.BusinessValue) * 0.7,
				Importance:        0.6, // Interactions are exploratory
			}
			derived = append(derived, df)
			interactionCount++

			// Pair ratio
			df = DerivedFeature{
				Name:           fmt.Sprintf("%s_div_%s", c1.Name, c2.Name),
				TransformType:  TransRatio,
				SourceFeatures: []string{c1.Name, c2.Name},
				Parameters: map[string]interface{}{
					"operation":   "divide",
					"handle_zero": true,
				},
				Description:       fmt.Sprintf("Ratio: %s / %s", c1.Name, c2.Name),
				ComputeComplexity: "simple",
				Stability:         0.6,
				Importance:        0.5,
			}
			derived = append(derived, df)
			interactionCount++
		}
	}

	return derived
}

// SuggestFeatures returns recommended feature set based on use case
func (fg *FeatureGenerator) SuggestFeatures(candidates []models.FeatureCandidate, useCase string) []DerivedFeature {
	allDerived := make([]DerivedFeature, 0)

	// Base recommendations for all use cases
	allDerived = append(allDerived, fg.GenerateAggregations(candidates)...)
	allDerived = append(allDerived, fg.GenerateMathTransforms(candidates)...)

	// Use-case specific features
	switch strings.ToLower(useCase) {
	case "time-series", "forecasting":
		allDerived = append(allDerived, fg.GenerateTimeSeriesFeatures(candidates)...)
	case "classification", "regression":
		allDerived = append(allDerived, fg.GenerateInteractions(candidates, 10)...)
	case "anomaly-detection":
		allDerived = append(allDerived, fg.GenerateTimeSeriesFeatures(candidates)...)
		allDerived = append(allDerived, fg.GenerateInteractions(candidates, 5)...)
	}

	// Categorical features for all
	allDerived = append(allDerived, fg.GenerateCategoricalFeatures(candidates)...)

	return allDerived
}

// ExportAsFeatureCandidates converts derived features to candidate format
func (fg *FeatureGenerator) ExportAsFeatureCandidates(derived []DerivedFeature) []models.FeatureCandidate {
	candidates := make([]models.FeatureCandidate, len(derived))

	for i, df := range derived {
		candidates[i] = models.FeatureCandidate{
			Name:           df.Name,
			SourceDatabase: "derived",
			SourceField:    fmt.Sprintf("%v", df.SourceFeatures),
			DataType:       "float", // Most derived features are numeric
			Completeness:   0.95,
			Cardinality:    -1,
			BusinessValue:  df.Importance,
			TechnicalScore: 0.8,
			DiscoveredAt:   time.Now(),
			Status:         "candidate",
		}
	}

	return candidates
}

// GetTopDerivedFeatures returns N most promising derived features
func (fg *FeatureGenerator) GetTopDerivedFeatures(derived []DerivedFeature, topN int) []DerivedFeature {
	if len(derived) <= topN {
		return derived
	}

	// Sort by importance
	sorted := make([]DerivedFeature, len(derived))
	copy(sorted, derived)

	// Simple bubble sort for importance descending
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].Importance > sorted[i].Importance {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	return sorted[:topN]
}
