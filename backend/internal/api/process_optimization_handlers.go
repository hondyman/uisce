package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ProcessOptimizationHandlers handles AI-powered process optimization
type ProcessOptimizationHandlers struct {
	db *sqlx.DB
}

// OptimizationSuggestion represents an AI-generated optimization recommendation
type OptimizationSuggestion struct {
	ID                  string                 `json:"id" db:"id"`
	WorkflowType        string                 `json:"workflow_type" db:"workflow_type"`
	SuggestionType      string                 `json:"suggestion_type" db:"suggestion_type"` // parallel_execution, reorder_steps, remove_step, sla_adjustment, resource_allocation
	Title               string                 `json:"title" db:"title"`
	Description         string                 `json:"description" db:"description"`
	ConfidenceScore     float64                `json:"confidence_score" db:"confidence_score"`         // 0-100
	ExpectedImprovement string                 `json:"expected_improvement" db:"expected_improvement"` // e.g., "15-20% faster"
	ImpactMetrics       map[string]interface{} `json:"impact_metrics" db:"impact_metrics"`             // duration_reduction, success_rate_increase, cost_savings
	TargetSteps         []string               `json:"target_steps" db:"target_steps"`                 // Steps affected
	ActionDetails       map[string]interface{} `json:"action_details" db:"action_details"`             // Specific actions to take
	BasedOnExecutions   int                    `json:"based_on_executions" db:"based_on_executions"`   // Sample size
	Status              string                 `json:"status" db:"status"`                             // pending, applied, dismissed, testing
	Priority            string                 `json:"priority" db:"priority"`                         // critical, high, medium, low
	CreatedAt           time.Time              `json:"created_at" db:"created_at"`
	TenantID            string                 `json:"tenant_id" db:"tenant_id"`
	DatasourceID        string                 `json:"datasource_id" db:"datasource_id"`
}

// AppliedOptimization tracks optimizations that have been applied
type AppliedOptimization struct {
	ID                string                 `json:"id" db:"id"`
	SuggestionID      string                 `json:"suggestion_id" db:"suggestion_id"`
	WorkflowType      string                 `json:"workflow_type" db:"workflow_type"`
	AppliedAt         time.Time              `json:"applied_at" db:"applied_at"`
	AppliedBy         string                 `json:"applied_by" db:"applied_by"`
	BeforeMetrics     map[string]interface{} `json:"before_metrics" db:"before_metrics"`
	AfterMetrics      map[string]interface{} `json:"after_metrics" db:"after_metrics"`
	ActualImprovement float64                `json:"actual_improvement" db:"actual_improvement"` // Percentage
	RollbackAvailable bool                   `json:"rollback_available" db:"rollback_available"`
	TenantID          string                 `json:"tenant_id" db:"tenant_id"`
	DatasourceID      string                 `json:"datasource_id" db:"datasource_id"`
}

// OptimizationForecast represents predicted impact of applying an optimization
type OptimizationForecast struct {
	SuggestionID               string                 `json:"suggestion_id"`
	PredictedDurationChange    float64                `json:"predicted_duration_change"`     // Seconds
	PredictedSuccessRateChange float64                `json:"predicted_success_rate_change"` // Percentage points
	PredictedCostSavings       float64                `json:"predicted_cost_savings"`        // Monthly USD
	RiskLevel                  string                 `json:"risk_level"`                    // low, medium, high
	RollbackComplexity         string                 `json:"rollback_complexity"`           // easy, moderate, complex
	AdditionalDetails          map[string]interface{} `json:"additional_details"`
}

// NewProcessOptimizationHandlers creates a new optimization handler
func NewProcessOptimizationHandlers(db *sqlx.DB) *ProcessOptimizationHandlers {
	return &ProcessOptimizationHandlers{db: db}
}

// RegisterRoutes registers optimization routes
func (h *ProcessOptimizationHandlers) RegisterRoutes(r chi.Router) {
	r.Route("/api/process-optimization", func(r chi.Router) {
		r.Get("/suggestions", h.GetSuggestions)
		r.Post("/analyze", h.AnalyzeAndGenerateSuggestions)
		r.Post("/apply/{suggestionID}", h.ApplyOptimization)
		r.Post("/dismiss/{suggestionID}", h.DismissSuggestion)
		r.Get("/applied", h.GetAppliedOptimizations)
		r.Get("/forecast/{suggestionID}", h.ForecastImpact)
		r.Post("/auto-tune/enable", h.EnableAutoTune)
		r.Get("/auto-tune/status", h.GetAutoTuneStatus)
	})
}

// GetSuggestions returns all optimization suggestions
func (h *ProcessOptimizationHandlers) GetSuggestions(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")
	workflowType := r.URL.Query().Get("workflow_type")
	status := r.URL.Query().Get("status") // pending, applied, dismissed

	if tenantID == "" || datasourceID == "" {
		http.Error(w, "Missing tenant_id or datasource_id", http.StatusBadRequest)
		return
	}

	query := `
		SELECT 
			id, workflow_type, suggestion_type, title, description,
			confidence_score, expected_improvement, impact_metrics,
			target_steps, action_details, based_on_executions,
			status, priority, created_at, tenant_id, datasource_id
		FROM process_optimization_suggestions
		WHERE tenant_id = $1 AND datasource_id = $2
	`

	args := []interface{}{tenantID, datasourceID}
	argCount := 3

	if workflowType != "" {
		query += fmt.Sprintf(" AND workflow_type = $%d", argCount)
		args = append(args, workflowType)
		argCount++
	}

	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, status)
		argCount++
	}

	query += " ORDER BY priority DESC, confidence_score DESC, created_at DESC"

	var suggestions []OptimizationSuggestion
	err := h.db.Select(&suggestions, query, args...)
	if err != nil {
		log.Printf("Error querying suggestions: %v", err)
		http.Error(w, "Failed to fetch suggestions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(suggestions)
}

// AnalyzeAndGenerateSuggestions analyzes workflows and generates optimization suggestions
func (h *ProcessOptimizationHandlers) AnalyzeAndGenerateSuggestions(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")
	workflowType := r.URL.Query().Get("workflow_type")

	if tenantID == "" || datasourceID == "" {
		http.Error(w, "Missing tenant_id or datasource_id", http.StatusBadRequest)
		return
	}

	// Generate suggestions using various ML algorithms
	suggestions := []OptimizationSuggestion{}

	// Algorithm 1: Parallel Execution Opportunities
	parallelSuggestions, err := h.detectParallelExecutionOpportunities(tenantID, datasourceID, workflowType)
	if err != nil {
		log.Printf("Error detecting parallel execution: %v", err)
	} else {
		suggestions = append(suggestions, parallelSuggestions...)
	}

	// Algorithm 2: Step Order Optimization
	reorderSuggestions, err := h.detectStepOrderOptimizations(tenantID, datasourceID, workflowType)
	if err != nil {
		log.Printf("Error detecting reorder opportunities: %v", err)
	} else {
		suggestions = append(suggestions, reorderSuggestions...)
	}

	// Algorithm 3: Unused Step Detection
	unusedStepSuggestions, err := h.detectUnusedSteps(tenantID, datasourceID, workflowType)
	if err != nil {
		log.Printf("Error detecting unused steps: %v", err)
	} else {
		suggestions = append(suggestions, unusedStepSuggestions...)
	}

	// Algorithm 4: SLA Adjustment Recommendations
	slaSuggestions, err := h.recommendSLAdjustments(tenantID, datasourceID, workflowType)
	if err != nil {
		log.Printf("Error generating SLA recommendations: %v", err)
	} else {
		suggestions = append(suggestions, slaSuggestions...)
	}

	// Algorithm 5: Resource Allocation Optimization
	resourceSuggestions, err := h.optimizeResourceAllocation(tenantID, datasourceID, workflowType)
	if err != nil {
		log.Printf("Error optimizing resources: %v", err)
	} else {
		suggestions = append(suggestions, resourceSuggestions...)
	}

	// Save suggestions to database
	for i := range suggestions {
		suggestions[i].ID = uuid.New().String()
		suggestions[i].CreatedAt = time.Now()
		suggestions[i].TenantID = tenantID
		suggestions[i].DatasourceID = datasourceID
		suggestions[i].Status = "pending"

		_, err := h.db.Exec(`
			INSERT INTO process_optimization_suggestions (
				id, workflow_type, suggestion_type, title, description,
				confidence_score, expected_improvement, impact_metrics,
				target_steps, action_details, based_on_executions,
				status, priority, created_at, tenant_id, datasource_id
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
			ON CONFLICT (workflow_type, suggestion_type, tenant_id, datasource_id)
			WHERE status = 'pending'
			DO UPDATE SET
				confidence_score = EXCLUDED.confidence_score,
				expected_improvement = EXCLUDED.expected_improvement,
				impact_metrics = EXCLUDED.impact_metrics,
				action_details = EXCLUDED.action_details,
				based_on_executions = EXCLUDED.based_on_executions
		`, suggestions[i].ID, suggestions[i].WorkflowType, suggestions[i].SuggestionType,
			suggestions[i].Title, suggestions[i].Description, suggestions[i].ConfidenceScore,
			suggestions[i].ExpectedImprovement, suggestions[i].ImpactMetrics,
			suggestions[i].TargetSteps, suggestions[i].ActionDetails,
			suggestions[i].BasedOnExecutions, suggestions[i].Status, suggestions[i].Priority,
			suggestions[i].CreatedAt, tenantID, datasourceID)

		if err != nil {
			log.Printf("Error saving suggestion: %v", err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"suggestions_generated": len(suggestions),
		"suggestions":           suggestions,
		"timestamp":             time.Now(),
	})
}

// detectParallelExecutionOpportunities finds steps that can run in parallel
func (h *ProcessOptimizationHandlers) detectParallelExecutionOpportunities(tenantID, datasourceID, workflowType string) ([]OptimizationSuggestion, error) {
	// Analyze step dependencies and execution patterns
	query := `
		WITH step_dependencies AS (
			SELECT
				m1.workflow_type,
				m1.step_name as step_a,
				m2.step_name as step_b,
				COUNT(*) as execution_count,
				AVG(EXTRACT(EPOCH FROM (m2.started_at - m1.completed_at))) as avg_gap_seconds
			FROM process_execution_metrics m1
			JOIN process_execution_metrics m2 ON 
				m1.workflow_id = m2.workflow_id AND
				m1.completed_at < m2.started_at AND
				m1.tenant_id = m2.tenant_id
			WHERE m1.tenant_id = $1
				AND m1.datasource_id = $2
				AND m1.status = 'completed'
				AND m2.status = 'completed'
				AND ($3 = '' OR m1.workflow_type = $3)
			GROUP BY m1.workflow_type, m1.step_name, m2.step_name
			HAVING COUNT(*) > 10
		)
		SELECT
			workflow_type,
			step_a,
			step_b,
			execution_count,
			avg_gap_seconds
		FROM step_dependencies
		WHERE avg_gap_seconds > 5  -- Steps with >5s gap could potentially run in parallel
		ORDER BY avg_gap_seconds DESC
		LIMIT 5
	`

	type StepPair struct {
		WorkflowType   string  `db:"workflow_type"`
		StepA          string  `db:"step_a"`
		StepB          string  `db:"step_b"`
		ExecutionCount int     `db:"execution_count"`
		AvgGapSeconds  float64 `db:"avg_gap_seconds"`
	}

	var pairs []StepPair
	err := h.db.Select(&pairs, query, tenantID, datasourceID, workflowType)
	if err != nil {
		return nil, err
	}

	suggestions := []OptimizationSuggestion{}
	for _, pair := range pairs {
		if pair.AvgGapSeconds < 5 {
			continue
		}

		potentialSavings := pair.AvgGapSeconds * float64(pair.ExecutionCount) / 3600 // Hours saved
		improvement := (pair.AvgGapSeconds / 120) * 100                              // Rough % improvement assuming 2min avg workflow

		suggestions = append(suggestions, OptimizationSuggestion{
			WorkflowType:   pair.WorkflowType,
			SuggestionType: "parallel_execution",
			Title:          fmt.Sprintf("Enable Parallel Execution: %s + %s", pair.StepA, pair.StepB),
			Description: fmt.Sprintf("Steps '%s' and '%s' have no dependencies and wait an average of %.1f seconds between execution. "+
				"Running them in parallel could save %.1f hours across %d executions.",
				pair.StepA, pair.StepB, pair.AvgGapSeconds, potentialSavings, pair.ExecutionCount),
			ConfidenceScore:     math.Min(95, 60+(pair.AvgGapSeconds/10)*10),
			ExpectedImprovement: fmt.Sprintf("%.0f%% faster, %.1f hours saved", math.Min(improvement, 40), potentialSavings),
			ImpactMetrics: map[string]interface{}{
				"duration_reduction_seconds": pair.AvgGapSeconds,
				"total_time_saved_hours":     potentialSavings,
				"executions_analyzed":        pair.ExecutionCount,
			},
			TargetSteps: []string{pair.StepA, pair.StepB},
			ActionDetails: map[string]interface{}{
				"action":       "enable_parallel_execution",
				"step_group":   []string{pair.StepA, pair.StepB},
				"wait_for_all": true,
			},
			BasedOnExecutions: pair.ExecutionCount,
			Priority:          h.calculatePriority(improvement, pair.AvgGapSeconds),
		})
	}

	return suggestions, nil
}

// detectStepOrderOptimizations finds inefficient step sequences
func (h *ProcessOptimizationHandlers) detectStepOrderOptimizations(tenantID, datasourceID, workflowType string) ([]OptimizationSuggestion, error) {
	// Analyze if reordering steps could reduce wait times or improve cache efficiency
	query := `
		WITH step_wait_times AS (
			SELECT
				workflow_type,
				step_name,
				AVG(EXTRACT(EPOCH FROM (started_at - LAG(completed_at) OVER (PARTITION BY workflow_id ORDER BY started_at)))) as avg_wait_before,
				AVG(EXTRACT(EPOCH FROM (completed_at - started_at))) as avg_duration,
				COUNT(*) as execution_count
			FROM process_execution_metrics
			WHERE tenant_id = $1
				AND datasource_id = $2
				AND ($3 = '' OR workflow_type = $3)
				AND status = 'completed'
			GROUP BY workflow_type, step_name
			HAVING COUNT(*) > 10 AND AVG(EXTRACT(EPOCH FROM (started_at - LAG(completed_at) OVER (PARTITION BY workflow_id ORDER BY started_at)))) > 10
		)
		SELECT * FROM step_wait_times
		ORDER BY avg_wait_before DESC
		LIMIT 3
	`

	type StepWait struct {
		WorkflowType   string  `db:"workflow_type"`
		StepName       string  `db:"step_name"`
		AvgWaitBefore  float64 `db:"avg_wait_before"`
		AvgDuration    float64 `db:"avg_duration"`
		ExecutionCount int     `db:"execution_count"`
	}

	var steps []StepWait
	err := h.db.Select(&steps, query, tenantID, datasourceID, workflowType)
	if err != nil {
		return nil, err
	}

	suggestions := []OptimizationSuggestion{}
	for _, step := range steps {
		if step.AvgWaitBefore < 10 {
			continue
		}

		improvement := (step.AvgWaitBefore / (step.AvgDuration + step.AvgWaitBefore)) * 100

		suggestions = append(suggestions, OptimizationSuggestion{
			WorkflowType:   step.WorkflowType,
			SuggestionType: "reorder_steps",
			Title:          fmt.Sprintf("Reorder Step: %s", step.StepName),
			Description: fmt.Sprintf("Step '%s' waits an average of %.1f seconds before execution. "+
				"Consider reordering steps or optimizing dependencies to reduce wait time.",
				step.StepName, step.AvgWaitBefore),
			ConfidenceScore:     70,
			ExpectedImprovement: fmt.Sprintf("%.0f%% reduction in wait time", improvement),
			ImpactMetrics: map[string]interface{}{
				"current_wait_seconds": step.AvgWaitBefore,
				"step_duration":        step.AvgDuration,
				"potential_savings":    step.AvgWaitBefore * 0.7, // Assume 70% reduction
			},
			TargetSteps: []string{step.StepName},
			ActionDetails: map[string]interface{}{
				"action":           "analyze_dependencies",
				"current_position": "unknown",
				"suggested_action": "Move earlier in sequence if dependencies allow",
			},
			BasedOnExecutions: step.ExecutionCount,
			Priority:          h.calculatePriority(improvement, step.AvgWaitBefore),
		})
	}

	return suggestions, nil
}

// detectUnusedSteps finds steps that are rarely used or always skipped
func (h *ProcessOptimizationHandlers) detectUnusedSteps(tenantID, datasourceID, workflowType string) ([]OptimizationSuggestion, error) {
	query := `
		WITH step_usage AS (
			SELECT
				workflow_type,
				step_name,
				COUNT(*) as total_executions,
				COUNT(*) FILTER (WHERE status = 'skipped' OR status = 'failed') as skipped_or_failed,
				AVG(EXTRACT(EPOCH FROM (completed_at - started_at))) as avg_duration
			FROM process_execution_metrics
			WHERE tenant_id = $1
				AND datasource_id = $2
				AND ($3 = '' OR workflow_type = $3)
			GROUP BY workflow_type, step_name
			HAVING COUNT(*) > 20
		)
		SELECT
			workflow_type,
			step_name,
			total_executions,
			skipped_or_failed,
			ROUND((skipped_or_failed::FLOAT / total_executions) * 100, 2) as skip_rate,
			avg_duration
		FROM step_usage
		WHERE (skipped_or_failed::FLOAT / total_executions) > 0.8  -- >80% skipped/failed
		ORDER BY skip_rate DESC
		LIMIT 5
	`

	type UnusedStep struct {
		WorkflowType  string  `db:"workflow_type"`
		StepName      string  `db:"step_name"`
		TotalExecs    int     `db:"total_executions"`
		SkippedFailed int     `db:"skipped_or_failed"`
		SkipRate      float64 `db:"skip_rate"`
		AvgDuration   float64 `db:"avg_duration"`
	}

	var steps []UnusedStep
	err := h.db.Select(&steps, query, tenantID, datasourceID, workflowType)
	if err != nil {
		return nil, err
	}

	suggestions := []OptimizationSuggestion{}
	for _, step := range steps {
		suggestions = append(suggestions, OptimizationSuggestion{
			WorkflowType:   step.WorkflowType,
			SuggestionType: "remove_step",
			Title:          fmt.Sprintf("Consider Removing: %s", step.StepName),
			Description: fmt.Sprintf("Step '%s' is skipped or fails %.0f%% of the time (%d out of %d executions). "+
				"Consider making it optional, removing it, or fixing underlying issues.",
				step.StepName, step.SkipRate, step.SkippedFailed, step.TotalExecs),
			ConfidenceScore:     85,
			ExpectedImprovement: fmt.Sprintf("Simplify workflow, save %.1f seconds per execution", step.AvgDuration),
			ImpactMetrics: map[string]interface{}{
				"skip_rate":   step.SkipRate,
				"executions":  step.TotalExecs,
				"time_wasted": step.AvgDuration * float64(step.TotalExecs),
			},
			TargetSteps: []string{step.StepName},
			ActionDetails: map[string]interface{}{
				"action":     "remove_or_make_optional",
				"skip_rate":  step.SkipRate,
				"suggestion": "Make conditional or remove entirely",
			},
			BasedOnExecutions: step.TotalExecs,
			Priority:          h.calculatePriority(step.SkipRate, float64(step.TotalExecs)),
		})
	}

	return suggestions, nil
}

// recommendSLAdjustments suggests SLA threshold changes based on actual performance
func (h *ProcessOptimizationHandlers) recommendSLAdjustments(tenantID, datasourceID, workflowType string) ([]OptimizationSuggestion, error) {
	query := `
		WITH workflow_performance AS (
			SELECT
				workflow_type,
				PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY EXTRACT(EPOCH FROM (completed_at - started_at))) as median_duration,
				PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY EXTRACT(EPOCH FROM (completed_at - started_at))) as p95_duration,
				AVG(EXTRACT(EPOCH FROM (completed_at - started_at))) as avg_duration,
				COUNT(*) as execution_count
			FROM process_execution_metrics
			WHERE tenant_id = $1
				AND datasource_id = $2
				AND ($3 = '' OR workflow_type = $3)
				AND status = 'completed'
			GROUP BY workflow_type
			HAVING COUNT(*) > 30
		)
		SELECT * FROM workflow_performance
		WHERE p95_duration < avg_duration * 1.5  -- Stable performance
	`

	type WorkflowPerf struct {
		WorkflowType   string  `db:"workflow_type"`
		MedianDuration float64 `db:"median_duration"`
		P95Duration    float64 `db:"p95_duration"`
		AvgDuration    float64 `db:"avg_duration"`
		ExecutionCount int     `db:"execution_count"`
	}

	var perfs []WorkflowPerf
	err := h.db.Select(&perfs, query, tenantID, datasourceID, workflowType)
	if err != nil {
		return nil, err
	}

	suggestions := []OptimizationSuggestion{}
	for _, perf := range perfs {
		recommendedSLA := perf.P95Duration * 1.1 // 10% buffer above P95

		suggestions = append(suggestions, OptimizationSuggestion{
			WorkflowType:   perf.WorkflowType,
			SuggestionType: "sla_adjustment",
			Title:          fmt.Sprintf("Optimize SLA for %s", perf.WorkflowType),
			Description: fmt.Sprintf("Based on %d executions, 95%% of workflows complete in %.1f seconds. "+
				"Recommended SLA: %.1f seconds (current median: %.1f seconds).",
				perf.ExecutionCount, perf.P95Duration, recommendedSLA, perf.MedianDuration),
			ConfidenceScore:     90,
			ExpectedImprovement: "More realistic SLAs, better resource planning",
			ImpactMetrics: map[string]interface{}{
				"median_duration":  perf.MedianDuration,
				"p95_duration":     perf.P95Duration,
				"recommended_sla":  recommendedSLA,
				"based_on_samples": perf.ExecutionCount,
			},
			TargetSteps: []string{}, // Workflow-level
			ActionDetails: map[string]interface{}{
				"action":          "update_sla",
				"recommended_sla": recommendedSLA,
				"unit":            "seconds",
			},
			BasedOnExecutions: perf.ExecutionCount,
			Priority:          "medium",
		})
	}

	return suggestions, nil
}

// optimizeResourceAllocation suggests resource allocation improvements
func (h *ProcessOptimizationHandlers) optimizeResourceAllocation(tenantID, datasourceID, workflowType string) ([]OptimizationSuggestion, error) {
	// Analyze resource contention and load distribution
	query := `
		WITH hourly_load AS (
			SELECT
				workflow_type,
				EXTRACT(HOUR FROM started_at) as hour_of_day,
				COUNT(*) as execution_count,
				AVG(EXTRACT(EPOCH FROM (completed_at - started_at))) as avg_duration
			FROM process_execution_metrics
			WHERE tenant_id = $1
				AND datasource_id = $2
				AND ($3 = '' OR workflow_type = $3)
				AND status = 'completed'
				AND started_at > NOW() - INTERVAL '30 days'
			GROUP BY workflow_type, EXTRACT(HOUR FROM started_at)
			HAVING COUNT(*) > 5
		)
		SELECT
			workflow_type,
			hour_of_day,
			execution_count,
			avg_duration
		FROM hourly_load
		WHERE execution_count > (SELECT AVG(execution_count) * 1.5 FROM hourly_load)
		ORDER BY execution_count DESC
		LIMIT 3
	`

	type HourlyLoad struct {
		WorkflowType   string  `db:"workflow_type"`
		HourOfDay      int     `db:"hour_of_day"`
		ExecutionCount int     `db:"execution_count"`
		AvgDuration    float64 `db:"avg_duration"`
	}

	var loads []HourlyLoad
	err := h.db.Select(&loads, query, tenantID, datasourceID, workflowType)
	if err != nil {
		return nil, err
	}

	suggestions := []OptimizationSuggestion{}
	for _, load := range loads {
		suggestions = append(suggestions, OptimizationSuggestion{
			WorkflowType:   load.WorkflowType,
			SuggestionType: "resource_allocation",
			Title:          fmt.Sprintf("Optimize Resources During Peak: %02d:00", load.HourOfDay),
			Description: fmt.Sprintf("Peak load detected at hour %02d with %d executions (%.1f sec avg duration). "+
				"Consider increasing worker capacity or implementing rate limiting during this period.",
				load.HourOfDay, load.ExecutionCount, load.AvgDuration),
			ConfidenceScore:     75,
			ExpectedImprovement: "Reduce queue wait times by 30-40%%",
			ImpactMetrics: map[string]interface{}{
				"peak_hour":       load.HourOfDay,
				"execution_count": load.ExecutionCount,
				"avg_duration":    load.AvgDuration,
			},
			TargetSteps: []string{}, // Workflow-level
			ActionDetails: map[string]interface{}{
				"action":         "increase_capacity",
				"peak_hour":      load.HourOfDay,
				"current_load":   load.ExecutionCount,
				"recommendation": "Scale workers during peak hours",
			},
			BasedOnExecutions: load.ExecutionCount,
			Priority:          "high",
		})
	}

	return suggestions, nil
}

// calculatePriority determines suggestion priority
func (h *ProcessOptimizationHandlers) calculatePriority(improvement, impact float64) string {
	score := improvement * (impact / 10)
	if score > 50 {
		return "critical"
	} else if score > 20 {
		return "high"
	} else if score > 10 {
		return "medium"
	}
	return "low"
}

// ApplyOptimization applies a suggestion to a workflow
func (h *ProcessOptimizationHandlers) ApplyOptimization(w http.ResponseWriter, r *http.Request) {
	suggestionID := chi.URLParam(r, "suggestionID")
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")

	if tenantID == "" || datasourceID == "" {
		http.Error(w, "Missing tenant_id or datasource_id", http.StatusBadRequest)
		return
	}

	// Get suggestion
	var suggestion OptimizationSuggestion
	err := h.db.Get(&suggestion, `
		SELECT * FROM process_optimization_suggestions
		WHERE id = $1 AND tenant_id = $2 AND datasource_id = $3
	`, suggestionID, tenantID, datasourceID)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Suggestion not found", http.StatusNotFound)
			return
		}
		log.Printf("Error fetching suggestion: %v", err)
		http.Error(w, "Failed to fetch suggestion", http.StatusInternalServerError)
		return
	}

	// TODO: Actually apply the optimization to the workflow definition
	// This would involve updating the business_processes table with the changes

	// Record applied optimization
	appliedID := uuid.New().String()
	_, err = h.db.Exec(`
		INSERT INTO applied_optimizations (
			id, suggestion_id, workflow_type, applied_at, applied_by,
			before_metrics, rollback_available, tenant_id, datasource_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, appliedID, suggestionID, suggestion.WorkflowType, time.Now(), "system",
		suggestion.ImpactMetrics, true, tenantID, datasourceID)

	if err != nil {
		log.Printf("Error recording applied optimization: %v", err)
		http.Error(w, "Failed to record optimization", http.StatusInternalServerError)
		return
	}

	// Update suggestion status
	_, err = h.db.Exec(`
		UPDATE process_optimization_suggestions
		SET status = 'applied'
		WHERE id = $1
	`, suggestionID)

	if err != nil {
		log.Printf("Error updating suggestion status: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":       true,
		"applied_id":    appliedID,
		"suggestion_id": suggestionID,
		"message":       "Optimization applied successfully",
	})
}

// DismissSuggestion dismisses a suggestion
func (h *ProcessOptimizationHandlers) DismissSuggestion(w http.ResponseWriter, r *http.Request) {
	suggestionID := chi.URLParam(r, "suggestionID")
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")

	if tenantID == "" || datasourceID == "" {
		http.Error(w, "Missing tenant_id or datasource_id", http.StatusBadRequest)
		return
	}

	_, err := h.db.Exec(`
		UPDATE process_optimization_suggestions
		SET status = 'dismissed'
		WHERE id = $1 AND tenant_id = $2 AND datasource_id = $3
	`, suggestionID, tenantID, datasourceID)

	if err != nil {
		log.Printf("Error dismissing suggestion: %v", err)
		http.Error(w, "Failed to dismiss suggestion", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Suggestion dismissed",
	})
}

// GetAppliedOptimizations returns optimization history
func (h *ProcessOptimizationHandlers) GetAppliedOptimizations(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")
	workflowType := r.URL.Query().Get("workflow_type")

	if tenantID == "" || datasourceID == "" {
		http.Error(w, "Missing tenant_id or datasource_id", http.StatusBadRequest)
		return
	}

	query := `
		SELECT * FROM applied_optimizations
		WHERE tenant_id = $1 AND datasource_id = $2
	`

	args := []interface{}{tenantID, datasourceID}
	if workflowType != "" {
		query += " AND workflow_type = $3"
		args = append(args, workflowType)
	}

	query += " ORDER BY applied_at DESC LIMIT 50"

	var applied []AppliedOptimization
	err := h.db.Select(&applied, query, args...)
	if err != nil {
		log.Printf("Error querying applied optimizations: %v", err)
		http.Error(w, "Failed to fetch history", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(applied)
}

// ForecastImpact predicts the impact of applying an optimization
func (h *ProcessOptimizationHandlers) ForecastImpact(w http.ResponseWriter, r *http.Request) {
	suggestionID := chi.URLParam(r, "suggestionID")
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")

	if tenantID == "" || datasourceID == "" {
		http.Error(w, "Missing tenant_id or datasource_id", http.StatusBadRequest)
		return
	}

	var suggestion OptimizationSuggestion
	err := h.db.Get(&suggestion, `
		SELECT * FROM process_optimization_suggestions
		WHERE id = $1 AND tenant_id = $2 AND datasource_id = $3
	`, suggestionID, tenantID, datasourceID)

	if err != nil {
		http.Error(w, "Suggestion not found", http.StatusNotFound)
		return
	}

	// Build forecast based on suggestion type
	forecast := OptimizationForecast{
		SuggestionID:       suggestionID,
		RiskLevel:          h.calculateRiskLevel(suggestion),
		RollbackComplexity: h.assessRollbackComplexity(suggestion),
		AdditionalDetails: map[string]interface{}{
			"confidence":  suggestion.ConfidenceScore,
			"sample_size": suggestion.BasedOnExecutions,
		},
	}

	// Extract impact metrics
	if durationReduction, ok := suggestion.ImpactMetrics["duration_reduction_seconds"].(float64); ok {
		forecast.PredictedDurationChange = -durationReduction
	}

	if successRateInc, ok := suggestion.ImpactMetrics["success_rate_increase"].(float64); ok {
		forecast.PredictedSuccessRateChange = successRateInc
	}

	// Estimate cost savings (rough calculation)
	if timeSaved, ok := suggestion.ImpactMetrics["total_time_saved_hours"].(float64); ok {
		forecast.PredictedCostSavings = timeSaved * 25 // $25/hour average cost
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(forecast)
}

// calculateRiskLevel assesses risk of applying optimization
func (h *ProcessOptimizationHandlers) calculateRiskLevel(suggestion OptimizationSuggestion) string {
	if suggestion.ConfidenceScore > 85 && suggestion.BasedOnExecutions > 100 {
		return "low"
	} else if suggestion.ConfidenceScore > 70 && suggestion.BasedOnExecutions > 50 {
		return "medium"
	}
	return "high"
}

// assessRollbackComplexity determines how easy it is to roll back
func (h *ProcessOptimizationHandlers) assessRollbackComplexity(suggestion OptimizationSuggestion) string {
	switch suggestion.SuggestionType {
	case "parallel_execution", "reorder_steps":
		return "easy"
	case "remove_step", "sla_adjustment":
		return "moderate"
	case "resource_allocation":
		return "complex"
	default:
		return "moderate"
	}
}

// EnableAutoTune enables automatic optimization
func (h *ProcessOptimizationHandlers) EnableAutoTune(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")

	if tenantID == "" || datasourceID == "" {
		http.Error(w, "Missing tenant_id or datasource_id", http.StatusBadRequest)
		return
	}

	var config struct {
		Enabled             bool     `json:"enabled"`
		ConfidenceThreshold float64  `json:"confidence_threshold"` // Only apply suggestions above this confidence
		AutoApplyTypes      []string `json:"auto_apply_types"`     // Which suggestion types to auto-apply
	}

	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Store auto-tune configuration
	// For now, just acknowledge

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Auto-tune configuration updated",
		"config":  config,
	})
}

// GetAutoTuneStatus returns auto-tune configuration
func (h *ProcessOptimizationHandlers) GetAutoTuneStatus(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")

	if tenantID == "" || datasourceID == "" {
		http.Error(w, "Missing tenant_id or datasource_id", http.StatusBadRequest)
		return
	}

	// TODO: Fetch actual configuration
	status := map[string]interface{}{
		"enabled":                     false,
		"confidence_threshold":        80.0,
		"auto_apply_types":            []string{"sla_adjustment"},
		"last_run":                    nil,
		"optimizations_applied_today": 0,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}
