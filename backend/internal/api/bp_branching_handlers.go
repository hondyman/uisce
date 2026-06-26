package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/hondyman/semlayer/backend/pkg/bp"
)

// BranchingHandlers handles BP branching API endpoints
type BranchingHandlers struct {
	db        *sqlx.DB
	evaluator *bp.BranchEvaluator
}

// NewBranchingHandlers creates new branching handlers
func NewBranchingHandlers(db *sqlx.DB) *BranchingHandlers {
	return &BranchingHandlers{
		db:        db,
		evaluator: bp.NewBranchEvaluator(db),
	}
}

// RegisterRoutes registers branching routes
func (h *BranchingHandlers) RegisterRoutes(r chi.Router) {
	r.Route("/api/bp/branching", func(r chi.Router) {
		// Branching evaluation
		r.Post("/evaluate", h.EvaluateBranches)
		r.Post("/execute", h.ExecuteBranch)
		r.Get("/history/{workflowInstanceID}", h.GetBranchHistory)

		// Metrics and analytics
		r.Get("/metrics/{stepID}", h.GetBranchMetrics)
		r.Get("/metrics/summary/{processID}", h.GetProcessMetricsSummary)
		r.Get("/branch-performance/{branchID}", h.GetBranchPerformance)

		// Configuration
		r.Post("/config/{stepID}", h.UpdateBranchingConfig)
		r.Get("/config/{stepID}", h.GetBranchingConfig)
		r.Get("/config/{stepID}/examples", h.GetBranchingExamples)

		// Join management
		r.Post("/join/create", h.CreateJoinPoint)
		r.Post("/join/{joinID}/complete", h.CompleteBranch)
		r.Get("/join/{joinID}/status", h.GetJoinStatus)

		// ML models
		r.Get("/ml-models", h.ListMLModels)
		r.Post("/ml-models", h.CreateMLModel)
		r.Get("/ml-models/{modelID}/performance", h.GetMLModelPerformance)

		// A/B tests
		r.Post("/ab-tests", h.StartABTest)
		r.Get("/ab-tests/{testID}", h.GetABTestStatus)
		r.Post("/ab-tests/{testID}/complete", h.CompleteABTest)

		// Anomalies
		r.Get("/anomalies", h.GetAnomalies)
		r.Get("/anomalies/{anomalyID}", h.GetAnomaly)
	})
}

// ============================================
// Branch Evaluation & Execution
// ============================================

// EvaluateRequest is the request to evaluate branches
type EvaluateRequest struct {
	TenantID        uuid.UUID              `json:"tenant_id"`
	DatasourceID    uuid.UUID              `json:"datasource_id"`
	WorkflowID      uuid.UUID              `json:"workflow_id"`
	StepID          uuid.UUID              `json:"step_id"`
	BranchingConfig *bp.BranchingConfig    `json:"branching_config"`
	Data            map[string]interface{} `json:"data"`
}

// EvaluateResponse is the response from branch evaluation
type EvaluateResponse struct {
	SelectedBranches []bp.Branch `json:"selected_branches"`
	EvaluationTime   int         `json:"evaluation_time_ms"`
	SelectionMethod  string      `json:"selection_method"`
}

// EvaluateBranches evaluates which branches should execute
func (h *BranchingHandlers) EvaluateBranches(w http.ResponseWriter, r *http.Request) {
	var req EvaluateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	start := time.Now()
	ctx := r.Context()

	// Evaluate branches
	branches, err := h.evaluator.EvaluateBranches(ctx, req.BranchingConfig, req.Data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	duration := int(time.Since(start).Milliseconds())

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(EvaluateResponse{
		SelectedBranches: branches,
		EvaluationTime:   duration,
		SelectionMethod:  req.BranchingConfig.Type,
	})
}

// ExecuteRequest is the request to execute a branch
type ExecuteRequest struct {
	TenantID           uuid.UUID              `json:"tenant_id"`
	DatasourceID       uuid.UUID              `json:"datasource_id"`
	WorkflowInstanceID uuid.UUID              `json:"workflow_instance_id"`
	StepID             uuid.UUID              `json:"step_id"`
	BranchID           string                 `json:"branch_id"`
	SelectedBy         string                 `json:"selected_by"`
	Data               map[string]interface{} `json:"data"`
}

// ExecuteBranch records branch execution
func (h *BranchingHandlers) ExecuteBranch(w http.ResponseWriter, r *http.Request) {
	var req ExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	execID := uuid.New()
	now := time.Now()

	result := &bp.BranchExecutionResult{
		ExecutionID:   execID,
		BranchID:      req.BranchID,
		SelectedBy:    req.SelectedBy,
		Status:        "completed",
		StartedAt:     now,
		ConditionEval: make(map[string]interface{}),
		ResultData:    req.Data,
	}

	if err := h.evaluator.LogBranchExecution(
		r.Context(),
		req.TenantID,
		req.DatasourceID,
		req.WorkflowInstanceID,
		req.StepID,
		result,
	); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"execution_id": execID,
		"branch_id":    req.BranchID,
		"status":       "recorded",
	})
}

// GetBranchHistory retrieves branch execution history
func (h *BranchingHandlers) GetBranchHistory(w http.ResponseWriter, r *http.Request) {
	workflowInstanceID := chi.URLParam(r, "workflowInstanceID")

	var executions []struct {
		BranchID    string    `db:"branch_id"`
		BranchLabel string    `db:"branch_label"`
		SelectedBy  string    `db:"selected_by"`
		Status      string    `db:"status"`
		StartedAt   time.Time `db:"started_at"`
		DurationMs  int       `db:"duration_ms"`
	}

	err := h.db.SelectContext(r.Context(), &executions, `
		SELECT branch_id, branch_label, selected_by, status, started_at, duration_ms
		FROM bp_branch_executions
		WHERE workflow_instance_id = $1
		ORDER BY started_at DESC
		LIMIT 100
	`, workflowInstanceID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(executions)
}

// ============================================
// Metrics & Analytics
// ============================================

// GetBranchMetrics retrieves metrics for a branch
func (h *BranchingHandlers) GetBranchMetrics(w http.ResponseWriter, r *http.Request) {
	stepID := chi.URLParam(r, "stepID")

	var metrics []struct {
		BranchID       string  `db:"branch_id"`
		BranchLabel    string  `db:"branch_label"`
		ExecutionCount int     `db:"execution_count"`
		CompletionRate float64 `db:"completion_rate"`
		AvgDurationMs  int     `db:"avg_duration_ms"`
		AvgMLScore     float64 `db:"avg_ml_score"`
	}

	err := h.db.SelectContext(r.Context(), &metrics, `
		SELECT 
			branch_id, branch_label,
			COALESCE(total_executions, 0) as execution_count,
			COALESCE(completion_rate, 0) as completion_rate,
			COALESCE(avg_duration_ms, 0) as avg_duration_ms,
			COALESCE(avg_ml_score, 0) as avg_ml_score
		FROM bp_branch_metrics
		WHERE step_id = $1
		ORDER BY total_executions DESC
	`, stepID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"metrics":   metrics,
		"timestamp": time.Now(),
	})
}

// GetProcessMetricsSummary gets summary metrics for a process
func (h *BranchingHandlers) GetProcessMetricsSummary(w http.ResponseWriter, r *http.Request) {
	processID := chi.URLParam(r, "processID")

	var summary struct {
		TotalExecutions int     `db:"total_executions"`
		AvgDuration     float64 `db:"avg_duration_ms"`
		CompletionRate  float64 `db:"completion_rate"`
		AvgMLScore      float64 `db:"avg_ml_score"`
	}

	err := h.db.GetContext(r.Context(), &summary, `
		SELECT 
			SUM(total_executions)::INT as total_executions,
			AVG(avg_duration_ms)::FLOAT as avg_duration_ms,
			AVG(completion_rate)::FLOAT as completion_rate,
			AVG(avg_ml_score)::FLOAT as avg_ml_score
		FROM bp_branch_metrics bm
		JOIN bp_steps bs ON bm.step_id = bs.id
		WHERE bs.process_id = $1
	`, processID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// GetBranchPerformance retrieves detailed performance for a branch
func (h *BranchingHandlers) GetBranchPerformance(w http.ResponseWriter, r *http.Request) {
	branchID := chi.URLParam(r, "branchID")

	var perf struct {
		BranchID       string  `db:"branch_id"`
		BranchLabel    string  `db:"branch_label"`
		TotalCount     int     `db:"total_executions"`
		CompletedCount int     `db:"completed_count"`
		TimeoutCount   int     `db:"timeout_count"`
		AvgDuration    int     `db:"avg_duration_ms"`
		P95Duration    int     `db:"p95_duration_ms"`
		SuccessRate    float64 `db:"success_rate"`
	}

	err := h.db.GetContext(r.Context(), &perf, `
		SELECT 
			branch_id, branch_label,
			COALESCE(total_executions, 0) as total_executions,
			COALESCE(completed_count, 0) as completed_count,
			COALESCE(timeout_count, 0) as timeout_count,
			COALESCE(avg_duration_ms, 0) as avg_duration_ms,
			COALESCE(p95_duration_ms, 0) as p95_duration_ms,
			COALESCE(success_rate, 0) as success_rate
		FROM bp_branch_metrics
		WHERE branch_id = $1
	`, branchID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(perf)
}

// ============================================
// Configuration Management
// ============================================

// UpdateBranchingConfig updates branching configuration for a step
func (h *BranchingHandlers) UpdateBranchingConfig(w http.ResponseWriter, r *http.Request) {
	stepID := chi.URLParam(r, "stepID")

	var config bp.BranchingConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	configJSON, _ := json.Marshal(config)

	_, err := h.db.ExecContext(r.Context(), `
		UPDATE bp_steps
		SET branching_config = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`, configJSON, stepID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

// GetBranchingConfig retrieves branching configuration for a step
func (h *BranchingHandlers) GetBranchingConfig(w http.ResponseWriter, r *http.Request) {
	stepID := chi.URLParam(r, "stepID")

	var configJSON []byte
	err := h.db.GetContext(r.Context(), &configJSON, `
		SELECT branching_config FROM bp_steps WHERE id = $1
	`, stepID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(configJSON)
}

// GetBranchingExamples returns branching configuration examples
func (h *BranchingHandlers) GetBranchingExamples(w http.ResponseWriter, r *http.Request) {
	examples := map[string]interface{}{
		"exclusive": map[string]interface{}{
			"type": "exclusive",
			"branches": []map[string]interface{}{
				{
					"id":    "branch-1",
					"label": "High Value",
					"condition": map[string]interface{}{
						"type": "and",
						"rules": []map[string]interface{}{
							{"field": "amount", "operator": "gte", "value": 5000},
						},
					},
				},
			},
		},
		"parallel": map[string]interface{}{
			"type": "parallel",
			"branches": []map[string]interface{}{
				{"id": "branch-1", "label": "Background Check", "steps": []string{}},
				{"id": "branch-2", "label": "Reference Check", "steps": []string{}},
			},
			"join_config": map[string]interface{}{
				"strategy":      "wait_all",
				"timeout_hours": 72,
			},
		},
		"weighted": map[string]interface{}{
			"type": "weighted",
			"branches": []map[string]interface{}{
				{"id": "branch-1", "label": "Control", "weight": 0.5},
				{"id": "branch-2", "label": "Experiment", "weight": 0.5},
			},
		},
		"ml_powered": map[string]interface{}{
			"type": "ml_powered",
			"ml_config": map[string]interface{}{
				"model_id":       "fraud-detector-v1",
				"model_endpoint": "https://ml.company.com/predict",
				"input_features": []string{"amount", "customer_age"},
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(examples)
}

// ============================================
// Join Management
// ============================================

// CreateJoinPointRequest is the request to create a join point
type CreateJoinPointRequest struct {
	TenantID           uuid.UUID `json:"tenant_id"`
	WorkflowInstanceID uuid.UUID `json:"workflow_instance_id"`
	StepID             uuid.UUID `json:"step_id"`
	Strategy           string    `json:"strategy"`
	RequiredBranches   int       `json:"required_branches"`
}

// CreateJoinPoint creates a join convergence point
func (h *BranchingHandlers) CreateJoinPoint(w http.ResponseWriter, r *http.Request) {
	var req CreateJoinPointRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	datasourceID := uuid.New() // Get from context
	joinID := uuid.New().String()

	_, err := h.evaluator.CreateJoinPoint(
		r.Context(),
		req.TenantID,
		datasourceID,
		req.WorkflowInstanceID,
		req.StepID,
		joinID,
		req.Strategy,
		req.RequiredBranches,
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"join_id": joinID})
}

// CompleteBranch marks a branch as complete in a join
func (h *BranchingHandlers) CompleteBranch(w http.ResponseWriter, r *http.Request) {
	joinID := chi.URLParam(r, "joinID")

	var req struct {
		BranchID string                 `json:"branch_id"`
		Result   map[string]interface{} `json:"result"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Record completion in database
	_, err := h.db.ExecContext(r.Context(), `
		UPDATE bp_join_convergences
		SET 
			completed_branches = completed_branches + 1,
			completed_branch_ids = array_append(completed_branch_ids, $1),
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`, req.BranchID, joinID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "recorded"})
}

// GetJoinStatus gets the status of a join point
func (h *BranchingHandlers) GetJoinStatus(w http.ResponseWriter, r *http.Request) {
	joinID := chi.URLParam(r, "joinID")

	var status struct {
		Status            string `db:"status"`
		CompletedBranches int    `db:"completed_branches"`
		RequiredBranches  int    `db:"required_branches"`
		JoinStrategy      string `db:"join_strategy"`
	}

	err := h.db.GetContext(r.Context(), &status, `
		SELECT status, completed_branches, required_branches, join_strategy
		FROM bp_join_convergences
		WHERE id = $1
	`, joinID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// ============================================
// ML Model Management
// ============================================

// ListMLModels lists available ML models
func (h *BranchingHandlers) ListMLModels(w http.ResponseWriter, r *http.Request) {
	var models []struct {
		ModelID           string  `db:"model_id"`
		ModelName         string  `db:"model_name"`
		Version           string  `db:"version"`
		AvgLatencyMs      int     `db:"avg_latency_ms"`
		SuccessRate       float64 `db:"success_rate"`
		TotalPredictions  int     `db:"total_predictions"`
		FailedPredictions int     `db:"failed_predictions"`
	}

	err := h.db.SelectContext(r.Context(), &models, `
		SELECT model_id, model_name, version, avg_latency_ms, success_rate,
		       total_predictions, failed_predictions
		FROM bp_ml_models
		WHERE is_active = true
		ORDER BY last_used_at DESC NULLS LAST
	`)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"models": models,
		"count":  len(models),
	})
}

// CreateMLModel creates a new ML model configuration
func (h *BranchingHandlers) CreateMLModel(w http.ResponseWriter, r *http.Request) {
	var model bp.MLConfig
	if err := json.NewDecoder(r.Body).Decode(&model); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Save to database
	features, _ := json.Marshal(model.InputFeatures)
	_, err := h.db.ExecContext(r.Context(), `
		INSERT INTO bp_ml_models (model_id, model_name, model_endpoint, input_features, confidence_threshold)
		VALUES ($1, $2, $3, $4, $5)
	`, model.ModelID, model.ModelID, model.ModelEndpoint, features, model.ConfidenceThreshold)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "created"})
}

// GetMLModelPerformance gets performance metrics for an ML model
func (h *BranchingHandlers) GetMLModelPerformance(w http.ResponseWriter, r *http.Request) {
	modelID := chi.URLParam(r, "modelID")

	var perf struct {
		ModelID           string  `db:"model_id"`
		TotalPredictions  int     `db:"total_predictions"`
		FailedPredictions int     `db:"failed_predictions"`
		SuccessRate       float64 `db:"success_rate"`
		AvgLatencyMs      int     `db:"avg_latency_ms"`
	}

	err := h.db.GetContext(r.Context(), &perf, `
		SELECT model_id, total_predictions, failed_predictions,
		       success_rate, avg_latency_ms
		FROM bp_ml_models
		WHERE model_id = $1
	`, modelID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(perf)
}

// ============================================
// A/B Testing
// ============================================

// StartABTest starts a new A/B test
func (h *BranchingHandlers) StartABTest(w http.ResponseWriter, r *http.Request) {
	var test struct {
		StepID             uuid.UUID `json:"step_id"`
		TestName           string    `json:"test_name"`
		ControlBranchID    string    `json:"control_branch_id"`
		ExperimentBranchID string    `json:"experiment_branch_id"`
		ControlWeight      float64   `json:"control_weight"`
		ExperimentWeight   float64   `json:"experiment_weight"`
		DurationDays       int       `json:"duration_days"`
	}

	if err := json.NewDecoder(r.Body).Decode(&test); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	endDate := time.Now().AddDate(0, 0, test.DurationDays)

	_, err := h.db.ExecContext(r.Context(), `
		INSERT INTO bp_ab_tests (
			step_id, test_name, control_branch_id, experiment_branch_id,
			control_weight, experiment_weight, start_date, end_date, status
		) VALUES ($1, $2, $3, $4, $5, $6, CURRENT_TIMESTAMP, $7, 'active')
	`, test.StepID, test.TestName, test.ControlBranchID, test.ExperimentBranchID,
		test.ControlWeight, test.ExperimentWeight, endDate)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "started"})
}

// GetABTestStatus gets status of an A/B test
func (h *BranchingHandlers) GetABTestStatus(w http.ResponseWriter, r *http.Request) {
	testID := chi.URLParam(r, "testID")

	var test struct {
		TestName                string  `db:"test_name"`
		ControlSampleSize       int     `db:"control_sample_size"`
		ExperimentSampleSize    int     `db:"experiment_sample_size"`
		ControlSuccessRate      float64 `db:"control_success_rate"`
		ExperimentSuccessRate   float64 `db:"experiment_success_rate"`
		StatisticalSignificance float64 `db:"statistical_significance"`
		Winner                  string  `db:"winner"`
		Status                  string  `db:"status"`
	}

	err := h.db.GetContext(r.Context(), &test, `
		SELECT test_name, control_sample_size, experiment_sample_size,
		       control_success_rate, experiment_success_rate,
		       statistical_significance, winner, status
		FROM bp_ab_tests
		WHERE id = $1
	`, testID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(test)
}

// CompleteABTest completes an A/B test
func (h *BranchingHandlers) CompleteABTest(w http.ResponseWriter, r *http.Request) {
	testID := chi.URLParam(r, "testID")

	_, err := h.db.ExecContext(r.Context(), `
		UPDATE bp_ab_tests
		SET status = 'completed', updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`, testID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "completed"})
}

// ============================================
// Anomaly Detection
// ============================================

// GetAnomalies retrieves detected anomalies
func (h *BranchingHandlers) GetAnomalies(w http.ResponseWriter, r *http.Request) {
	var anomalies []struct {
		ID            uuid.UUID `db:"id"`
		AnomalyType   string    `db:"anomaly_type"`
		Severity      string    `db:"severity"`
		Description   string    `db:"description"`
		AffectedCount int       `db:"affected_executions"`
		DetectedAt    time.Time `db:"detected_at"`
		Status        string    `db:"investigation_status"`
	}

	err := h.db.SelectContext(r.Context(), &anomalies, `
		SELECT id, anomaly_type, severity, description, affected_executions,
		       detected_at, investigation_status
		FROM bp_branch_anomalies
		WHERE investigation_status IN ('open', 'investigating')
		ORDER BY detected_at DESC
		LIMIT 50
	`)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"anomalies": anomalies,
		"count":     len(anomalies),
	})
}

// GetAnomaly retrieves a specific anomaly
func (h *BranchingHandlers) GetAnomaly(w http.ResponseWriter, r *http.Request) {
	anomalyID := chi.URLParam(r, "anomalyID")

	var anomaly struct {
		ID               uuid.UUID `db:"id"`
		AnomalyType      string    `db:"anomaly_type"`
		Severity         string    `db:"severity"`
		Description      string    `db:"description"`
		BaselineValue    float64   `db:"baseline_value"`
		ActualValue      float64   `db:"actual_value"`
		DeviationPercent float64   `db:"deviation_percent"`
		DetectedAt       time.Time `db:"detected_at"`
		Status           string    `db:"investigation_status"`
	}

	err := h.db.GetContext(r.Context(), &anomaly, `
		SELECT id, anomaly_type, severity, description, baseline_value,
		       actual_value, deviation_percent, detected_at, investigation_status
		FROM bp_branch_anomalies
		WHERE id = $1
	`, anomalyID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(anomaly)
}
