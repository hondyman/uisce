package ai

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
)

// SmartRetryOptimizer generates AI-optimized retry policies based on failure patterns
type SmartRetryOptimizer struct {
	logger *slog.Logger
}

// NewSmartRetryOptimizer creates a new smart retry optimizer
func NewSmartRetryOptimizer(logger *slog.Logger) *SmartRetryOptimizer {
	return &SmartRetryOptimizer{
		logger: logger,
	}
}

// RetryOutcome represents the result of a retry
type RetryOutcome struct {
	ExecutionID   uuid.UUID `json:"execution_id"`
	JobID         uuid.UUID `json:"job_id"`
	AttemptNumber int       `json:"attempt_number"`
	Success       bool      `json:"success"`
	DurationMS    int64     `json:"duration_ms"`
	DelayBeforeMS int64     `json:"delay_before_ms"` // Wait time before this attempt
	ErrorType     string    `json:"error_type"`
	Timestamp     time.Time `json:"timestamp"`
}

// CurrentRetryPolicy describes the existing retry configuration
type CurrentRetryPolicy struct {
	MaxAttempts        int      `json:"max_attempts"`
	InitialIntervalMS  int64    `json:"initial_interval_ms"`
	BackoffCoefficient float64  `json:"backoff_coefficient"`
	MaxIntervalMS      int64    `json:"max_interval_ms"`
	NonRetryableErrors []string `json:"non_retryable_errors,omitempty"`
}

// OptimizedRetryPolicy is the AI-recommended policy
type OptimizedRetryPolicy struct {
	MaxAttempts        int      `json:"max_attempts"`
	InitialIntervalMS  int64    `json:"initial_interval_ms"`
	BackoffCoefficient float64  `json:"backoff_coefficient"`
	MaxIntervalMS      int64    `json:"max_interval_ms"`
	NonRetryableErrors []string `json:"non_retryable_errors"`

	// Optimization metadata
	ExpectedSuccessRate  float64  `json:"expected_success_rate"`
	ExpectedAvgAttempts  float64  `json:"expected_avg_attempts"`
	ExpectedTotalLatency int64    `json:"expected_total_latency_ms"`
	Confidence           float64  `json:"confidence"`
	Reasoning            []string `json:"reasoning"`
}

// RetryOptimizationResult contains the full analysis
type RetryOptimizationResult struct {
	JobID           uuid.UUID            `json:"job_id"`
	CurrentPolicy   CurrentRetryPolicy   `json:"current_policy"`
	OptimizedPolicy OptimizedRetryPolicy `json:"optimized_policy"`
	Analysis        RetryAnalysis        `json:"analysis"`
	Improvements    []Improvement        `json:"improvements"`
	WarningsSummary []string             `json:"warnings,omitempty"`
}

// RetryAnalysis provides insights into retry patterns
type RetryAnalysis struct {
	TotalExecutions       int                   `json:"total_executions"`
	ExecutionsWithRetries int                   `json:"executions_with_retries"`
	OverallSuccessRate    float64               `json:"overall_success_rate"`
	SuccessRateByAttempt  map[int]float64       `json:"success_rate_by_attempt"`
	AvgAttemptsPerExec    float64               `json:"avg_attempts_per_execution"`
	AvgRecoveryTimeMS     int64                 `json:"avg_recovery_time_ms"`
	WastedRetries         int                   `json:"wasted_retries"` // Retries on non-recoverable errors
	ErrorTypeBreakdown    map[string]ErrorStats `json:"error_type_breakdown"`
}

// ErrorStats provides stats for an error type
type ErrorStats struct {
	Count            int     `json:"count"`
	RecoveryRate     float64 `json:"recovery_rate"` // Success after retry
	AvgAttemptsToFix float64 `json:"avg_attempts_to_fix"`
	IsRetryable      bool    `json:"is_retryable"`
}

// Improvement describes a specific optimization
type Improvement struct {
	Area       string  `json:"area"` // max_attempts, interval, backoff, errors
	Change     string  `json:"change"`
	Impact     string  `json:"impact"`
	Confidence float64 `json:"confidence"`
}

// OptimizeRetryPolicy generates an optimized retry policy
func (s *SmartRetryOptimizer) OptimizeRetryPolicy(
	ctx context.Context,
	jobID uuid.UUID,
	outcomes []RetryOutcome,
	currentPolicy CurrentRetryPolicy,
) (*RetryOptimizationResult, error) {
	s.logger.Info("Optimizing retry policy",
		"job_id", jobID,
		"outcomes", len(outcomes),
	)

	if len(outcomes) < 50 {
		return nil, fmt.Errorf("insufficient data for optimization (min 50, got %d)", len(outcomes))
	}

	// Analyze current retry patterns
	analysis := s.analyzeRetries(outcomes)

	// Generate optimized policy
	optimized := s.generateOptimizedPolicy(analysis, currentPolicy)

	// Identify improvements
	improvements := s.identifyImprovements(currentPolicy, optimized, analysis)

	result := &RetryOptimizationResult{
		JobID:           jobID,
		CurrentPolicy:   currentPolicy,
		OptimizedPolicy: optimized,
		Analysis:        analysis,
		Improvements:    improvements,
	}

	s.logger.Info("Retry optimization complete",
		"new_max_attempts", optimized.MaxAttempts,
		"expected_success_rate", fmt.Sprintf("%.1f%%", optimized.ExpectedSuccessRate*100),
	)

	return result, nil
}

// analyzeRetries examines historical retry patterns
func (s *SmartRetryOptimizer) analyzeRetries(outcomes []RetryOutcome) RetryAnalysis {
	analysis := RetryAnalysis{
		SuccessRateByAttempt: make(map[int]float64),
		ErrorTypeBreakdown:   make(map[string]ErrorStats),
	}

	// Group by execution
	byExecution := make(map[uuid.UUID][]RetryOutcome)
	for _, o := range outcomes {
		byExecution[o.ExecutionID] = append(byExecution[o.ExecutionID], o)
	}

	analysis.TotalExecutions = len(byExecution)

	var totalAttempts int
	var successfulExecs int
	var totalRecoveryTime int64

	// Attempt-level tracking
	attemptSuccesses := make(map[int]int)
	attemptTotals := make(map[int]int)

	// Error type tracking
	errorRecoveries := make(map[string]int)
	errorTotals := make(map[string]int)
	errorAttempts := make(map[string]int)

	for _, attempts := range byExecution {
		sort.Slice(attempts, func(i, j int) bool {
			return attempts[i].AttemptNumber < attempts[j].AttemptNumber
		})

		totalAttempts += len(attempts)
		if len(attempts) > 1 {
			analysis.ExecutionsWithRetries++
		}

		// Track each attempt
		finalSuccess := false
		for _, a := range attempts {
			attemptTotals[a.AttemptNumber]++
			if a.Success {
				attemptSuccesses[a.AttemptNumber]++
				finalSuccess = true
			}

			// Track recovery time
			if a.AttemptNumber > 1 && a.Success {
				totalRecoveryTime += a.DelayBeforeMS
			}

			// Track error types
			if !a.Success && a.ErrorType != "" {
				errorTotals[a.ErrorType]++
				errorAttempts[a.ErrorType] += a.AttemptNumber
			}
		}

		if finalSuccess {
			successfulExecs++
			// Check if this was a recovery
			lastAttempt := attempts[len(attempts)-1]
			if lastAttempt.Success && len(attempts) > 1 {
				firstError := ""
				for _, a := range attempts {
					if !a.Success && a.ErrorType != "" {
						firstError = a.ErrorType
						break
					}
				}
				if firstError != "" {
					errorRecoveries[firstError]++
				}
			}
		}
	}

	analysis.OverallSuccessRate = float64(successfulExecs) / float64(analysis.TotalExecutions)
	analysis.AvgAttemptsPerExec = float64(totalAttempts) / float64(analysis.TotalExecutions)

	if analysis.ExecutionsWithRetries > 0 {
		analysis.AvgRecoveryTimeMS = totalRecoveryTime / int64(analysis.ExecutionsWithRetries)
	}

	// Calculate per-attempt success rates
	for attempt, total := range attemptTotals {
		if total > 0 {
			analysis.SuccessRateByAttempt[attempt] = float64(attemptSuccesses[attempt]) / float64(total)
		}
	}

	// Calculate error type stats
	for errType, total := range errorTotals {
		stats := ErrorStats{
			Count: total,
		}
		if total > 0 {
			stats.RecoveryRate = float64(errorRecoveries[errType]) / float64(total)
			stats.AvgAttemptsToFix = float64(errorAttempts[errType]) / float64(total)
			stats.IsRetryable = stats.RecoveryRate > 0.1 // At least 10% recovery
		}
		analysis.ErrorTypeBreakdown[errType] = stats
	}

	// Count wasted retries
	for errType, stats := range analysis.ErrorTypeBreakdown {
		if !stats.IsRetryable {
			analysis.WastedRetries += errorTotals[errType]
		}
	}

	return analysis
}

// generateOptimizedPolicy creates an optimized retry configuration
func (s *SmartRetryOptimizer) generateOptimizedPolicy(analysis RetryAnalysis, current CurrentRetryPolicy) OptimizedRetryPolicy {
	optimized := OptimizedRetryPolicy{
		MaxAttempts:        current.MaxAttempts,
		InitialIntervalMS:  current.InitialIntervalMS,
		BackoffCoefficient: current.BackoffCoefficient,
		MaxIntervalMS:      current.MaxIntervalMS,
		NonRetryableErrors: current.NonRetryableErrors,
		Confidence:         0.8,
	}

	// Optimize max attempts based on marginal success rate
	optimalAttempts := s.calculateOptimalAttempts(analysis.SuccessRateByAttempt)
	if optimalAttempts != current.MaxAttempts {
		optimized.MaxAttempts = optimalAttempts
		optimized.Reasoning = append(optimized.Reasoning,
			fmt.Sprintf("Adjusted max attempts from %d to %d based on diminishing returns analysis",
				current.MaxAttempts, optimalAttempts))
	}

	// Optimize intervals based on recovery patterns
	if analysis.AvgRecoveryTimeMS > 0 {
		// Initial interval should give systems time to recover
		suggestedInitial := int64(float64(analysis.AvgRecoveryTimeMS) * 0.5)
		if suggestedInitial < 100 {
			suggestedInitial = 100
		}
		if suggestedInitial > 30000 {
			suggestedInitial = 30000
		}

		if math.Abs(float64(suggestedInitial-current.InitialIntervalMS)) > float64(current.InitialIntervalMS)*0.3 {
			optimized.InitialIntervalMS = suggestedInitial
			optimized.Reasoning = append(optimized.Reasoning,
				fmt.Sprintf("Adjusted initial interval to %dms based on observed recovery patterns", suggestedInitial))
		}
	}

	// Add non-retryable errors
	for errType, stats := range analysis.ErrorTypeBreakdown {
		if !stats.IsRetryable && stats.Count > 5 {
			// Check if not already in list
			found := false
			for _, existing := range optimized.NonRetryableErrors {
				if existing == errType {
					found = true
					break
				}
			}
			if !found {
				optimized.NonRetryableErrors = append(optimized.NonRetryableErrors, errType)
				optimized.Reasoning = append(optimized.Reasoning,
					fmt.Sprintf("Added '%s' to non-retryable errors (%.0f%% recovery rate)",
						errType, stats.RecoveryRate*100))
			}
		}
	}

	// Calculate expected outcomes
	optimized.ExpectedSuccessRate = s.estimateSuccessRate(analysis, optimized.MaxAttempts)
	optimized.ExpectedAvgAttempts = s.estimateAvgAttempts(analysis, optimized.MaxAttempts)
	optimized.ExpectedTotalLatency = s.estimateTotalLatency(optimized)

	return optimized
}

// calculateOptimalAttempts finds the point of diminishing returns
func (s *SmartRetryOptimizer) calculateOptimalAttempts(successRateByAttempt map[int]float64) int {
	if len(successRateByAttempt) == 0 {
		return 3 // Default
	}

	// Find where marginal improvement drops below 5%
	var attempts []int
	for a := range successRateByAttempt {
		attempts = append(attempts, a)
	}
	sort.Ints(attempts)

	optimal := 1
	prevRate := 0.0
	for _, a := range attempts {
		rate := successRateByAttempt[a]
		marginalGain := rate - prevRate

		if marginalGain < 0.05 && a > 1 {
			break
		}
		optimal = a
		prevRate = rate
	}

	// Clamp to reasonable range
	if optimal < 1 {
		optimal = 1
	}
	if optimal > 10 {
		optimal = 10
	}

	return optimal
}

// estimateSuccessRate predicts success rate with the policy
func (s *SmartRetryOptimizer) estimateSuccessRate(analysis RetryAnalysis, maxAttempts int) float64 {
	if len(analysis.SuccessRateByAttempt) == 0 {
		return analysis.OverallSuccessRate
	}

	// Cumulative success probability
	cumulative := 0.0
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if rate, ok := analysis.SuccessRateByAttempt[attempt]; ok {
			cumulative = cumulative + (1-cumulative)*rate
		}
	}

	return cumulative
}

// estimateAvgAttempts predicts average attempts needed
func (s *SmartRetryOptimizer) estimateAvgAttempts(analysis RetryAnalysis, maxAttempts int) float64 {
	// Simplified estimate
	if analysis.OverallSuccessRate == 0 {
		return float64(maxAttempts)
	}
	return math.Min(analysis.AvgAttemptsPerExec, float64(maxAttempts))
}

// estimateTotalLatency estimates retry overhead
func (s *SmartRetryOptimizer) estimateTotalLatency(policy OptimizedRetryPolicy) int64 {
	// Sum of all potential delays
	totalDelay := int64(0)
	interval := policy.InitialIntervalMS

	for i := 1; i < policy.MaxAttempts; i++ {
		totalDelay += interval
		interval = int64(float64(interval) * policy.BackoffCoefficient)
		if interval > policy.MaxIntervalMS {
			interval = policy.MaxIntervalMS
		}
	}

	return totalDelay
}

// identifyImprovements describes what's changing and why
func (s *SmartRetryOptimizer) identifyImprovements(current CurrentRetryPolicy, optimized OptimizedRetryPolicy, analysis RetryAnalysis) []Improvement {
	var improvements []Improvement

	if optimized.MaxAttempts != current.MaxAttempts {
		direction := "increased"
		if optimized.MaxAttempts < current.MaxAttempts {
			direction = "reduced"
		}
		improvements = append(improvements, Improvement{
			Area:       "max_attempts",
			Change:     fmt.Sprintf("%s from %d to %d", direction, current.MaxAttempts, optimized.MaxAttempts),
			Impact:     fmt.Sprintf("Expected success rate: %.1f%%", optimized.ExpectedSuccessRate*100),
			Confidence: 0.85,
		})
	}

	if optimized.InitialIntervalMS != current.InitialIntervalMS {
		improvements = append(improvements, Improvement{
			Area:       "initial_interval",
			Change:     fmt.Sprintf("Changed from %dms to %dms", current.InitialIntervalMS, optimized.InitialIntervalMS),
			Impact:     "Better aligned with observed recovery times",
			Confidence: 0.75,
		})
	}

	if len(optimized.NonRetryableErrors) > len(current.NonRetryableErrors) {
		newErrors := len(optimized.NonRetryableErrors) - len(current.NonRetryableErrors)
		improvements = append(improvements, Improvement{
			Area:       "non_retryable_errors",
			Change:     fmt.Sprintf("Added %d error types to skip list", newErrors),
			Impact:     fmt.Sprintf("Prevents ~%d wasted retries per 100 executions", analysis.WastedRetries*100/max(analysis.TotalExecutions, 1)),
			Confidence: 0.9,
		})
	}

	return improvements
}
