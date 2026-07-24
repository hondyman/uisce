package evaluation

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// Phase 7: Evaluation & Safety

// FinancialEvalCase represents a financial calculation test case
type FinancialEvalCase struct {
	ID              string                 `json:"id"`
	Category        string                 `json:"category"` // nav, pricing, attribution, etc.
	Question        string                 `json:"question"`
	ExpectedAnswer  string                 `json:"expected_answer,omitempty"`
	ExpectedNumeric *NumericExpectation    `json:"expected_numeric,omitempty"`
	Context         map[string]interface{} `json:"context"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// NumericExpectation defines expected numeric result with tolerance
type NumericExpectation struct {
	Value     float64 `json:"value"`
	Tolerance float64 `json:"tolerance"` // Absolute tolerance
	Precision int     `json:"precision"` // Decimal places for comparison
}

// EvalResult represents the outcome of an evaluation
type EvalResult struct {
	CaseID           string    `json:"case_id"`
	IsCorrect        bool      `json:"is_correct"`
	NumericAccuracy  *float64  `json:"numeric_accuracy,omitempty"` // Percentage accuracy
	SourcesGrounded  bool      `json:"sources_grounded"` // All sources in catalog?
	HallucinationRisk string   `json:"hallucination_risk"` // low, medium, high
	LatencyMs        int64     `json:"latency_ms"`
	ActualAnswer     string    `json:"actual_answer"`
	ErrorMessage     string    `json:"error_message,omitempty"`
	Timestamp        time.Time `json:"timestamp"`
}

// GroundingValidator checks if answer is grounded in catalog
type GroundingValidator struct {
	RetrievalThreshold float64 // Minimum similarity score
}

// ValidateGrounding checks if sources are properly cited and exist in catalog
func (v *GroundingValidator) ValidateGrounding(answer string, sources []string, catalogPaths []string) (bool, string) {
	// Check 1: Are all cited sources in the catalog?
	catalogSet := make(map[string]bool)
	for _, path := range catalogPaths {
		catalogSet[path] = true
	}
	
	for _, source := range sources {
		if !catalogSet[source] {
			return false, fmt.Sprintf("hallucination")
		}
	}
	
	// Check 2: Does answer contain fabricated numbers not in sources?
	// Simplified check - real implementation would parse numbers and verify
	if strings.Contains(answer, "approximately") && len(sources) == 0 {
		return false, "low_confidence_fabrication"
	}
	
	return true, "grounded"
}

// NumericValidator validates numeric accuracy
type NumericValidator struct{}

// ValidateNumeric compares actual vs expected numeric result
func (v *NumericValidator) ValidateNumeric(expected NumericExpectation, actual float64) (bool, float64) {
	diff := math.Abs(expected.Value - actual)
	
	// Check absolute tolerance
	if diff <= expected.Tolerance {
		accuracy := 100.0 * (1.0 - diff/math.Max(math.Abs(expected.Value), 0.01))
		return true, accuracy
	}
	
	// Calculate accuracy percentage
	accuracy := 100.0 * (1.0 - diff/math.Max(math.Abs(expected.Value), 0.01))
	return false, accuracy
}

// ScenarioType defines types of edge case scenarios
type ScenarioType string

const (
	ScenarioMissingPrices     ScenarioType = "missing_prices"
	ScenarioStaleBenchmark    ScenarioType = "stale_benchmark"
	ScenarioFXHoliday         ScenarioType = "fx_holiday"
	ScenarioLateCorporateAction ScenarioType = "late_corporate_action"
	ScenarioDataGap           ScenarioType = "data_gap"
)

// EdgeCaseScenario represents a stress test scenario
type EdgeCaseScenario struct {
	Type        ScenarioType           `json:"type"`
	Description string                 `json:"description"`
	Setup       map[string]interface{} `json:"setup"`
	ExpectedBehavior string             `json:"expected_behavior"`
}

// SafetyConfig defines safety parameters for LLM responses
type SafetyConfig struct {
	MinRetrievalScore    float64 `json:"min_retrieval_score"`
	MaxResponseLength    int     `json:"max_response_length"`
	RequireSourceCitation bool   `json:"require_source_citation"`
	DeterministicMode    bool   `json:"deterministic_mode"` // Low temperature for regulated outputs
	CacheResponses       bool   `json:"cache_responses"`  // Cache canonical answers per version
}

// HallucinationDetector identifies potential hallucinations
type HallucinationDetector struct {
	groundingValidator *GroundingValidator
}

// Detect analyzes response for hallucination indicators
func (d *HallucinationDetector) Detect(answer string, sources []string, confidence string) string {
	// Low confidence with few sources = high risk
	if confidence == "low" && len(sources) < 2 {
		return "high"
	}
	
	// Hedge words without sources = medium risk
	hedgeWords := []string{"approximately", "around", "roughly", "about"}
	for _, hedge := range hedgeWords {
		if strings.Contains(strings.ToLower(answer), hedge) && len(sources) == 0 {
			return "medium"
		}
	}
	
	// Well-sourced, high confidence = low risk
	if confidence == "high" && len(sources) >= 2 {
		return "low"
	}
	
	return "medium"
}
