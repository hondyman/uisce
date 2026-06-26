package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/calc-engine/workflows"
	"go.temporal.io/sdk/client"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// CalcEngineHandler handles metric computation API requests
type CalcEngineHandler struct {
	db             *sql.DB
	temporalClient client.Client
}

// NewCalcEngineHandler creates a new calc engine handler
func NewCalcEngineHandler(db *sql.DB, temporalClient client.Client) *CalcEngineHandler {
	return &CalcEngineHandler{
		db:             db,
		temporalClient: temporalClient,
	}
}

// ============================================================================
// REQUEST / RESPONSE TYPES
// ============================================================================

// CreateMetricRequest represents a request to create a metric
type CreateMetricRequest struct {
	Name                     string   `json:"name"`
	DisplayName              string   `json:"display_name"`
	Domain                   string   `json:"domain"`
	Category                 string   `json:"category"`
	Granularity              string   `json:"granularity"`
	AggregationFunction      string   `json:"aggregation_function"`
	ComparisonPeriods        []string `json:"comparison_periods"`
	BaseQuery                string   `json:"base_query"`
	ComputationType          string   `json:"computation_type"`
	ComputationLogic         string   `json:"computation_logic"`
	SLAFreshnessHours        int      `json:"sla_freshness_hours"`
	SLACompletenessThreshold float64  `json:"sla_completeness_threshold"`
}

// MetricResponse represents a metric registry entry
type MetricResponse struct {
	MetricID                 string  `json:"metric_id"`
	Name                     string  `json:"name"`
	DisplayName              string  `json:"display_name"`
	Domain                   string  `json:"domain"`
	Granularity              string  `json:"granularity"`
	AggregationFunction      string  `json:"aggregation_function"`
	GoldenPath               bool    `json:"golden_path"`
	SLAFreshnessHours        int     `json:"sla_freshness_hours"`
	SLACompletenessThreshold float64 `json:"sla_completeness_threshold"`
	OwnerUserID              string  `json:"owner_user_id"`
	CreatedAt                string  `json:"created_at"`
	UpdatedAt                string  `json:"updated_at"`
}

// ComputeRequest represents a compute trigger request
type ComputeRequest struct {
	MetricID    string `json:"metric_id"`
	PeriodLabel string `json:"period_label"`
}

// ComputeResponse represents the response to a compute trigger
type ComputeResponse struct {
	RunID  string `json:"run_id"`
	Status string `json:"status"`
}

// JobRunResponse represents a metric job run
type JobRunResponse struct {
	RunID       string                 `json:"run_id"`
	MetricID    string                 `json:"metric_id"`
	CalcType    string                 `json:"calc_type"`
	PeriodLabel string                 `json:"period_label"`
	Status      string                 `json:"status"`
	StartedAt   string                 `json:"started_at"`
	EndedAt     string                 `json:"ended_at"`
	Stats       map[string]interface{} `json:"stats"`
}

// AnomalyResponse represents an anomaly event
type AnomalyResponse struct {
	ID               string  `json:"id"`
	AnomalyType      string  `json:"anomaly_type"`
	DetectedAt       string  `json:"detected_at"`
	Severity         string  `json:"severity"`
	Confidence       float64 `json:"confidence"`
	ActualValue      float64 `json:"actual_value"`
	ExpectedValue    float64 `json:"expected_value"`
	ExpectedRangeMin float64 `json:"expected_range_min"`
	ExpectedRangeMax float64 `json:"expected_range_max"`
	Status           string  `json:"status"`
	CreatedAt        string  `json:"created_at"`
}

// ============================================================================
// ROUTES
// ============================================================================

// RegisterCalcEngineRoutes registers all calc-engine routes
func RegisterCalcEngineRoutes(router chi.Router, db *sql.DB, temporalClient client.Client) {
	handler := NewCalcEngineHandler(db, temporalClient)

	router.Route("/api/metrics", func(r chi.Router) {
		r.Post("/", handler.CreateMetric)
		r.Get("/", handler.ListMetrics)

		r.Route("/{metricID}", func(r chi.Router) {
			r.Get("/", handler.GetMetric)
			r.Put("/", handler.UpdateMetric)
			r.Delete("/", handler.DeleteMetric)

			r.Route("/compute", func(r chi.Router) {
				r.Post("/pop", handler.TriggerPopCompute)
				r.Post("/anomaly", handler.TriggerAnomalyCompute)
			})

			r.Get("/runs", handler.GetMetricRuns)
			r.Get("/anomalies", handler.GetMetricAnomalies)
		})
	})
}

// ============================================================================
// HANDLERS - METRIC CRUD
// ============================================================================

// CreateMetric creates a new metric
func (h *CalcEngineHandler) CreateMetric(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "Missing X-Tenant-ID header", http.StatusBadRequest)
		return
	}

	var req CreateMetricRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Name == "" || req.Domain == "" || req.AggregationFunction == "" {
		http.Error(w, "Missing required fields: name, domain, aggregation_function", http.StatusBadRequest)
		return
	}

	metricID := uuid.NewString()
	userID := r.Header.Get("X-User-ID")

	query := `
    INSERT INTO metric_registry(
      tenant_id, metric_id, name, display_name, domain, category, 
      granularity, aggregation_function, base_query, computation_type, 
      computation_logic, sla_freshness_hours, sla_completeness_threshold,
      owner_user_id, created_by, updated_by, created_at, updated_at
    )
    VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, now(), now())
    RETURNING metric_id, name, display_name, domain, granularity, aggregation_function,
              golden_path, sla_freshness_hours, sla_completeness_threshold, owner_user_id,
              created_at, updated_at
  `

	granularity := req.Granularity
	if granularity == "" {
		granularity = "day"
	}

	freshnessHours := req.SLAFreshnessHours
	if freshnessHours == 0 {
		freshnessHours = 24
	}

	completenessThreshold := req.SLACompletenessThreshold
	if completenessThreshold == 0 {
		completenessThreshold = 95.0
	}

	var metricResp MetricResponse
	err := h.db.QueryRowContext(r.Context(),
		query,
		tenantID, metricID, req.Name, req.DisplayName, req.Domain, req.Category,
		granularity, req.AggregationFunction, req.BaseQuery, req.ComputationType,
		req.ComputationLogic, freshnessHours, completenessThreshold, userID, userID, userID).
		Scan(&metricResp.MetricID, &metricResp.Name, &metricResp.DisplayName, &metricResp.Domain,
			&metricResp.Granularity, &metricResp.AggregationFunction, &metricResp.GoldenPath,
			&metricResp.SLAFreshnessHours, &metricResp.SLACompletenessThreshold,
			&metricResp.OwnerUserID, &metricResp.CreatedAt, &metricResp.UpdatedAt)

	if err != nil {
		log.Printf("Error creating metric: %v", err)
		http.Error(w, fmt.Sprintf("Error creating metric: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(metricResp)
}

// ListMetrics lists all metrics for a tenant
func (h *CalcEngineHandler) ListMetrics(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "Missing X-Tenant-ID header", http.StatusBadRequest)
		return
	}

	query := `
    SELECT metric_id, name, display_name, domain, granularity, aggregation_function,
           golden_path, sla_freshness_hours, sla_completeness_threshold, 
           owner_user_id, created_at, updated_at
    FROM metric_registry
    WHERE tenant_id = $1
    ORDER BY updated_at DESC
    LIMIT 100
  `

	rows, err := h.db.QueryContext(r.Context(), query, tenantID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error listing metrics: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var metrics []MetricResponse
	for rows.Next() {
		var m MetricResponse
		if err := rows.Scan(&m.MetricID, &m.Name, &m.DisplayName, &m.Domain,
			&m.Granularity, &m.AggregationFunction, &m.GoldenPath,
			&m.SLAFreshnessHours, &m.SLACompletenessThreshold,
			&m.OwnerUserID, &m.CreatedAt, &m.UpdatedAt); err != nil {
			log.Printf("Error scanning metric: %v", err)
			continue
		}
		metrics = append(metrics, m)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(metrics)
}

// GetMetric retrieves a specific metric
func (h *CalcEngineHandler) GetMetric(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	metricID := chi.URLParam(r, "metricID")

	query := `
    SELECT metric_id, name, display_name, domain, granularity, aggregation_function,
           golden_path, sla_freshness_hours, sla_completeness_threshold,
           owner_user_id, created_at, updated_at
    FROM metric_registry
    WHERE tenant_id = $1 AND metric_id = $2
  `

	var m MetricResponse
	err := h.db.QueryRowContext(r.Context(), query, tenantID, metricID).
		Scan(&m.MetricID, &m.Name, &m.DisplayName, &m.Domain,
			&m.Granularity, &m.AggregationFunction, &m.GoldenPath,
			&m.SLAFreshnessHours, &m.SLACompletenessThreshold,
			&m.OwnerUserID, &m.CreatedAt, &m.UpdatedAt)

	if err == sql.ErrNoRows {
		http.Error(w, "Metric not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, fmt.Sprintf("Error retrieving metric: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(m)
}

// UpdateMetric updates an existing metric
func (h *CalcEngineHandler) UpdateMetric(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	metricID := chi.URLParam(r, "metricID")
	userID := r.Header.Get("X-User-ID")

	var req CreateMetricRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	query := `
    UPDATE metric_registry
    SET name = $1, display_name = $2, domain = $3, category = $4,
        aggregation_function = $5, computation_logic = $6,
        sla_freshness_hours = $7, sla_completeness_threshold = $8,
        updated_by = $9, updated_at = now()
    WHERE tenant_id = $10 AND metric_id = $11
    RETURNING metric_id, name, display_name, domain, granularity, aggregation_function,
              golden_path, sla_freshness_hours, sla_completeness_threshold, owner_user_id,
              created_at, updated_at
  `

	freshnessHours := req.SLAFreshnessHours
	if freshnessHours == 0 {
		freshnessHours = 24
	}

	completenessThreshold := req.SLACompletenessThreshold
	if completenessThreshold == 0 {
		completenessThreshold = 95.0
	}

	var m MetricResponse
	err := h.db.QueryRowContext(r.Context(), query,
		req.Name, req.DisplayName, req.Domain, req.Category,
		req.AggregationFunction, req.ComputationLogic,
		freshnessHours, completenessThreshold, userID, tenantID, metricID).
		Scan(&m.MetricID, &m.Name, &m.DisplayName, &m.Domain,
			&m.Granularity, &m.AggregationFunction, &m.GoldenPath,
			&m.SLAFreshnessHours, &m.SLACompletenessThreshold,
			&m.OwnerUserID, &m.CreatedAt, &m.UpdatedAt)

	if err == sql.ErrNoRows {
		http.Error(w, "Metric not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, fmt.Sprintf("Error updating metric: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(m)
}

// DeleteMetric deletes a metric
func (h *CalcEngineHandler) DeleteMetric(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	metricID := chi.URLParam(r, "metricID")

	query := `DELETE FROM metric_registry WHERE tenant_id = $1 AND metric_id = $2`
	result, err := h.db.ExecContext(r.Context(), query, tenantID, metricID)

	if err != nil {
		http.Error(w, fmt.Sprintf("Error deleting metric: %v", err), http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Metric not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ============================================================================
// HANDLERS - COMPUTE TRIGGERS
// ============================================================================

// TriggerPopCompute triggers a PoP computation
func (h *CalcEngineHandler) TriggerPopCompute(w http.ResponseWriter, r *http.Request) {
	h.triggerCompute(w, r, "pop")
}

// TriggerAnomalyCompute triggers an anomaly computation
func (h *CalcEngineHandler) TriggerAnomalyCompute(w http.ResponseWriter, r *http.Request) {
	h.triggerCompute(w, r, "anomaly")
}

// triggerCompute is a helper to trigger either PoP or anomaly computation
func (h *CalcEngineHandler) triggerCompute(w http.ResponseWriter, r *http.Request, calcType string) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "Missing X-Tenant-ID header", http.StatusBadRequest)
		return
	}

	metricID := chi.URLParam(r, "metricID")

	var req ComputeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	if req.MetricID == "" {
		req.MetricID = metricID
	}

	if req.PeriodLabel == "" {
		// Default to current month
		now := time.Now()
		req.PeriodLabel = now.Format("2006-01")
	}

	runID := uuid.NewString()

	// Create the job run record
	query := `
    INSERT INTO metric_job_runs(
      run_id, tenant_id, metric_id, calc_type, period_label, status
    )
    VALUES($1, $2, $3, $4, $5, 'pending')
    RETURNING run_id
  `

	var returnedRunID string
	err := h.db.QueryRowContext(r.Context(), query,
		runID, tenantID, req.MetricID, calcType, req.PeriodLabel).
		Scan(&returnedRunID)

	if err != nil {
		log.Printf("Error creating job run: %v", err)
		http.Error(w, fmt.Sprintf("Error triggering compute: %v", err), http.StatusInternalServerError)
		return
	}

	// Prepare workflow request with run ID
	// Parse period_label to extract start/end dates if needed
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.AddDate(0, 1, 0)

	workflowReq := workflows.ComputeRequest{
		TenantID:    tenantID,
		MetricID:    req.MetricID,
		CalcType:    calcType,
		PeriodLabel: req.PeriodLabel,
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		RunID:       returnedRunID,
	}

	// Execute Temporal workflow asynchronously
	if h.temporalClient != nil {
		workflowOptions := client.StartWorkflowOptions{
			ID:        returnedRunID,
			TaskQueue: "metrics-compute",
		}

		_, err := h.temporalClient.ExecuteWorkflow(r.Context(), workflowOptions, workflows.MetricComputeWorkflow, workflowReq)
		if err != nil {
			log.Printf("Warning: Failed to start Temporal workflow: %v (continuing anyway)", err)
			// Don't fail the request - the job run is already created
		}
	} else {
		log.Printf("Warning: Temporal client not configured, workflow will not execute")
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(ComputeResponse{
		RunID:  returnedRunID,
		Status: "pending",
	})
}

// ============================================================================
// HANDLERS - RUNS & ANOMALIES
// ============================================================================

// GetMetricRuns retrieves recent job runs for a metric
func (h *CalcEngineHandler) GetMetricRuns(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	metricID := chi.URLParam(r, "metricID")

	query := `
    SELECT run_id, metric_id, calc_type, period_label, status, 
           started_at, ended_at, stats
    FROM metric_job_runs
    WHERE tenant_id = $1 AND metric_id = $2
    ORDER BY started_at DESC
    LIMIT 100
  `

	rows, err := h.db.QueryContext(r.Context(), query, tenantID, metricID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error listing runs: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var runs []JobRunResponse
	for rows.Next() {
		var jr JobRunResponse
		var stats json.RawMessage
		if err := rows.Scan(&jr.RunID, &jr.MetricID, &jr.CalcType, &jr.PeriodLabel,
			&jr.Status, &jr.StartedAt, &jr.EndedAt, &stats); err != nil {
			log.Printf("Error scanning run: %v", err)
			continue
		}

		if stats != nil {
			json.Unmarshal(stats, &jr.Stats)
		}

		runs = append(runs, jr)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(runs)
}

// GetMetricAnomalies retrieves recent anomalies for a metric
func (h *CalcEngineHandler) GetMetricAnomalies(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	metricID := chi.URLParam(r, "metricID")

	query := `
    SELECT id, anomaly_type, detected_at, severity, confidence,
           actual_value, expected_value, expected_range_min, expected_range_max,
           status, created_at
    FROM anomaly_events
    WHERE tenant_id = $1 AND metric_id = $2
    ORDER BY detected_at DESC
    LIMIT 100
  `

	rows, err := h.db.QueryContext(r.Context(), query, tenantID, metricID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error listing anomalies: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var anomalies []AnomalyResponse
	for rows.Next() {
		var ar AnomalyResponse
		if err := rows.Scan(&ar.ID, &ar.AnomalyType, &ar.DetectedAt, &ar.Severity,
			&ar.Confidence, &ar.ActualValue, &ar.ExpectedValue,
			&ar.ExpectedRangeMin, &ar.ExpectedRangeMax, &ar.Status, &ar.CreatedAt); err != nil {
			log.Printf("Error scanning anomaly: %v", err)
			continue
		}
		anomalies = append(anomalies, ar)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(anomalies)
}
