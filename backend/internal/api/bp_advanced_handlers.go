package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// ============================================================================
// Feature 1: AI-Powered Predictive Routing Handlers
// ============================================================================

// ListAIModels retrieves all registered AI models
// GET /api/bp/branching/ai-models
func (s *Server) ListAIModels(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "Missing tenant ID", http.StatusBadRequest)
		return
	}

	query := `
		SELECT model_id, model_type, accuracy_threshold, auto_switch_enabled, 
		       predictions_count, drift_detected, last_updated
		FROM bp_ai_models
		WHERE tenant_id = $1
		ORDER BY last_updated DESC
	`

	rows, err := s.DB.QueryContext(r.Context(), query, tenantID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch models: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var models []map[string]interface{}
	for rows.Next() {
		var modelID, modelType string
		var accuracy float64
		var autoSwitch, drift bool
		var predCount int64
		var updated time.Time

		if err := rows.Scan(&modelID, &modelType, &accuracy, &autoSwitch, &predCount, &drift, &updated); err != nil {
			continue
		}

		models = append(models, map[string]interface{}{
			"model_id":            modelID,
			"model_type":          modelType,
			"accuracy_threshold":  accuracy,
			"auto_switch_enabled": autoSwitch,
			"predictions_count":   predCount,
			"drift_detected":      drift,
			"last_updated":        updated,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models)
}

// CreateAIModel registers a new AI model
// POST /api/bp/branching/ai-models
func (s *Server) CreateAIModel(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "Missing tenant ID", http.StatusBadRequest)
		return
	}

	var payload struct {
		ModelID           string  `json:"model_id"`
		ModelType         string  `json:"model_type"`
		Endpoint          string  `json:"endpoint"`
		AccuracyThreshold float64 `json:"accuracy_threshold"`
		FallbackStrategy  string  `json:"fallback_strategy"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	query := `
		INSERT INTO bp_ai_models (model_id, model_type, model_endpoint, accuracy_threshold, 
		                          fallback_strategy, auto_switch_enabled, tenant_id)
		VALUES ($1, $2, $3, $4, $5, true, $6)
		RETURNING model_id
	`

	var returnedID string
	err := s.DB.QueryRowContext(r.Context(), query,
		payload.ModelID, payload.ModelType, payload.Endpoint,
		payload.AccuracyThreshold, payload.FallbackStrategy, tenantID).Scan(&returnedID)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create model: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"model_id": returnedID})
}

// ============================================================================
// Feature 2: Semantic Intent Routing Handlers
// ============================================================================

// ListSemanticIntents retrieves all semantic intents
// GET /api/bp/branching/semantic-intents
func (s *Server) ListSemanticIntents(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "Missing tenant ID", http.StatusBadRequest)
		return
	}

	query := `
		SELECT intent_id, intent_description, similarity_threshold, 
		       match_count, avg_confidence, semantic_model
		FROM bp_semantic_intents
		WHERE tenant_id = $1
		ORDER BY match_count DESC
	`

	rows, err := s.DB.QueryContext(r.Context(), query, tenantID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch intents: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var intents []map[string]interface{}
	for rows.Next() {
		var intentID, model string
		var description sql.NullString
		var threshold, confidence float64
		var matchCount int64

		if err := rows.Scan(&intentID, &description, &threshold, &matchCount, &confidence, &model); err != nil {
			continue
		}

		desc := ""
		if description.Valid {
			desc = description.String
		}

		intents = append(intents, map[string]interface{}{
			"intent_id":            intentID,
			"description":          desc,
			"similarity_threshold": threshold,
			"match_count":          matchCount,
			"avg_confidence":       confidence,
			"semantic_model":       model,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(intents)
}

// ============================================================================
// Feature 3: Scoring Matrices Handlers
// ============================================================================

// ListScoringMatrices retrieves all scoring matrices
// GET /api/bp/branching/scoring-matrices
func (s *Server) ListScoringMatrices(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "Missing tenant ID", http.StatusBadRequest)
		return
	}

	query := `
		SELECT matrix_id, matrix_name, evaluations_total, avg_score, min_score, max_score
		FROM bp_scoring_matrices
		WHERE tenant_id = $1
		ORDER BY evaluations_total DESC
	`

	rows, err := s.DB.QueryContext(r.Context(), query, tenantID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch matrices: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var matrices []map[string]interface{}
	for rows.Next() {
		var matrixID, matrixName string
		var evalCount int64
		var avg, min, max float64

		if err := rows.Scan(&matrixID, &matrixName, &evalCount, &avg, &min, &max); err != nil {
			continue
		}

		matrices = append(matrices, map[string]interface{}{
			"matrix_id":   matrixID,
			"matrix_name": matrixName,
			"evaluations": evalCount,
			"avg_score":   avg,
			"min_score":   min,
			"max_score":   max,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(matrices)
}

// ============================================================================
// Feature 4: Time-Series Forecasting Handlers
// ============================================================================

// GetLatestForecast retrieves the most recent forecast
// GET /api/bp/branching/forecasts/latest
func (s *Server) GetLatestForecast(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "Missing tenant ID", http.StatusBadRequest)
		return
	}

	query := `
		SELECT forecast_id, forecast_model, predicted_queue_depth,
		       predicted_approval_time_minutes, confidence_interval_lower,
		       confidence_interval_upper, forecast_accuracy, created_at
		FROM bp_time_series_forecasts
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	var forecast map[string]interface{}
	var forecastID, model string
	var queueDepth, approvalTime int
	var confLower, confUpper, accuracy float64
	var createdAt time.Time

	err := s.DB.QueryRowContext(r.Context(), query, tenantID).Scan(
		&forecastID, &model, &queueDepth, &approvalTime,
		&confLower, &confUpper, &accuracy, &createdAt)

	if err == sql.ErrNoRows {
		http.Error(w, "No forecast available", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}

	forecast = map[string]interface{}{
		"forecast_id":                  forecastID,
		"forecast_model":               model,
		"predicted_queue_depth":        queueDepth,
		"predicted_approval_time_mins": approvalTime,
		"confidence_interval_lower":    confLower,
		"confidence_interval_upper":    confUpper,
		"forecast_accuracy":            accuracy,
		"created_at":                   createdAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(forecast)
}

// ============================================================================
// Feature 9: Real-Time Analytics Handlers
// ============================================================================

// GetBranchAnalytics retrieves analytics for a specific branch
// GET /api/bp/branching/{branchID}/analytics
func (s *Server) GetBranchAnalytics(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	branchID := chi.URLParam(r, "branchID")

	if tenantID == "" || branchID == "" {
		http.Error(w, "Missing tenant ID or branch ID", http.StatusBadRequest)
		return
	}

	query := `
		SELECT selection_count, completion_count, abandonment_count,
		       avg_duration_ms, p95_duration_ms, p99_duration_ms,
		       success_rate, error_rate, anomaly_score, trend_direction
		FROM bp_branch_analytics_extended
		WHERE branch_id = $1 AND tenant_id = $2
		ORDER BY metric_period DESC
		LIMIT 1
	`

	var analytics map[string]interface{}
	var selectionCount, completionCount, abandonmentCount int64
	var avgDuration, p95Duration, p99Duration, successRate, errorRate, anomalyScore float64
	var trendDirection string

	err := s.DB.QueryRowContext(r.Context(), query, branchID, tenantID).Scan(
		&selectionCount, &completionCount, &abandonmentCount,
		&avgDuration, &p95Duration, &p99Duration,
		&successRate, &errorRate, &anomalyScore, &trendDirection)

	if err == sql.ErrNoRows {
		http.Error(w, "No analytics available", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}

	analytics = map[string]interface{}{
		"selection_count":   selectionCount,
		"completion_count":  completionCount,
		"abandonment_count": abandonmentCount,
		"avg_duration_ms":   avgDuration,
		"p95_duration_ms":   p95Duration,
		"p99_duration_ms":   p99Duration,
		"success_rate":      successRate,
		"error_rate":        errorRate,
		"anomaly_score":     anomalyScore,
		"trend_direction":   trendDirection,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analytics)
}

// ============================================================================
// Feature 10: Collaborative Voting Handlers
// ============================================================================

// CreateVotingDecision initiates a collaborative decision
// POST /api/bp/branching/voting-decisions
func (s *Server) CreateVotingDecision(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "Missing tenant ID", http.StatusBadRequest)
		return
	}

	var payload struct {
		DecisionType      string        `json:"decision_type"`
		Stakeholders      []interface{} `json:"stakeholders"`
		ApprovalThreshold float64       `json:"approval_threshold"`
		QuorumRequirement float64       `json:"quorum_requirement"`
		TimeoutHours      int           `json:"timeout_hours"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	stakeholdersJSON, _ := json.Marshal(payload.Stakeholders)

	query := `
		INSERT INTO bp_collaborative_decisions (decision_mechanism, stakeholders, 
		                                        approval_threshold, quorum_requirement, 
		                                        timeout_hours, decision_outcome, tenant_id)
		VALUES ($1, $2, $3, $4, $5, 'pending', $6)
		RETURNING decision_id
	`

	var decisionID string
	err := s.DB.QueryRowContext(r.Context(), query,
		payload.DecisionType, string(stakeholdersJSON),
		payload.ApprovalThreshold, payload.QuorumRequirement,
		payload.TimeoutHours, tenantID).Scan(&decisionID)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create decision: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"decision_id": decisionID})
}

// CastVote submits a vote for a decision
// POST /api/bp/branching/voting-decisions/{decisionID}/votes
func (s *Server) CastVote(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	decisionID := chi.URLParam(r, "decisionID")

	if tenantID == "" || decisionID == "" {
		http.Error(w, "Missing tenant ID or decision ID", http.StatusBadRequest)
		return
	}

	var payload struct {
		Role string `json:"role"`
		Vote bool   `json:"vote"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	// Record the vote (implementation would update stakeholder vote in JSON)
	query := `
		UPDATE bp_collaborative_decisions
		SET votes_received = votes_received + 1, last_updated = NOW()
		WHERE decision_id = $1 AND tenant_id = $2
	`

	if _, err := s.DB.ExecContext(r.Context(), query, decisionID, tenantID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to record vote: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "vote_recorded"})
}

// ============================================================================
// Feature 11: Geofencing Handlers
// ============================================================================

// ListGeofences retrieves all geofence rules
// GET /api/bp/branching/geofences
func (s *Server) ListGeofences(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "Missing tenant ID", http.StatusBadRequest)
		return
	}

	query := `
		SELECT rule_id, geofence_type, radius_km, center_lat, center_lng, branch_id
		FROM bp_geofence_rules
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`

	rows, err := s.DB.QueryContext(r.Context(), query, tenantID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch geofences: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var geofences []map[string]interface{}
	for rows.Next() {
		var ruleID, geofenceType, branchID string
		var radiusKm, centerLat, centerLng float64

		if err := rows.Scan(&ruleID, &geofenceType, &radiusKm, &centerLat, &centerLng, &branchID); err != nil {
			continue
		}

		geofences = append(geofences, map[string]interface{}{
			"rule_id":       ruleID,
			"geofence_type": geofenceType,
			"radius_km":     radiusKm,
			"center_lat":    centerLat,
			"center_lng":    centerLng,
			"branch_id":     branchID,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(geofences)
}

// ============================================================================
// Feature 12: Blockchain Audit Handlers
// ============================================================================

// GetBlockchainAudit retrieves the blockchain audit trail
// GET /api/bp/branching/blockchain-audit/{eventID}
func (s *Server) GetBlockchainAudit(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	eventID := chi.URLParam(r, "eventID")

	if tenantID == "" || eventID == "" {
		http.Error(w, "Missing tenant ID or event ID", http.StatusBadRequest)
		return
	}

	query := `
		SELECT event_id, event_type, event_hash, parent_hash, blockchain_network,
		       verification_status, tamper_detected, created_at
		FROM bp_blockchain_audit
		WHERE event_id = $1 AND tenant_id = $2
	`

	var auditRecord map[string]interface{}
	var event_id, eventType, eventHash, parentHash, network, status string
	var tampered bool
	var createdAt time.Time

	err := s.DB.QueryRowContext(r.Context(), query, eventID, tenantID).Scan(
		&event_id, &eventType, &eventHash, &parentHash, &network,
		&status, &tampered, &createdAt)

	if err == sql.ErrNoRows {
		http.Error(w, "Audit record not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}

	auditRecord = map[string]interface{}{
		"event_id":            event_id,
		"event_type":          eventType,
		"event_hash":          eventHash,
		"parent_hash":         parentHash,
		"blockchain_network":  network,
		"verification_status": status,
		"tamper_detected":     tampered,
		"created_at":          createdAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(auditRecord)
}

// ============================================================================
// Feature 13: Natural Language Configuration Handlers
// ============================================================================

// CreateNLConfig processes natural language branching request
// POST /api/bp/branching/nl-config
func (s *Server) CreateNLConfig(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "Missing tenant ID", http.StatusBadRequest)
		return
	}

	var payload struct {
		Query string `json:"query"`
		Model string `json:"model"` // gpt-4|claude
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	// In production: call LLM API (GPT-4/Claude) to generate config
	// For now: insert with pending status
	query := `
		INSERT INTO bp_nl_configurations (nl_query, llm_model, human_approval_status, tenant_id)
		VALUES ($1, $2, 'pending', $3)
		RETURNING config_id
	`

	var configID string
	err := s.DB.QueryRowContext(r.Context(), query, payload.Query, payload.Model, tenantID).Scan(&configID)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create NL config: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"config_id": configID,
		"status":    "pending_human_approval",
	})
}

// ============================================================================
// Feature 14: Resource Pool Management Handlers
// ============================================================================

// ListResourcePools retrieves all resource pools
// GET /api/bp/branching/resource-pools
func (s *Server) ListResourcePools(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "Missing tenant ID", http.StatusBadRequest)
		return
	}

	query := `
		SELECT pool_id, pool_name, resource_type, current_load, capacity_metric,
		       routing_strategy, auto_scaling_enabled, last_updated
		FROM bp_resource_pools
		WHERE tenant_id = $1
		ORDER BY current_load DESC
	`

	rows, err := s.DB.QueryContext(r.Context(), query, tenantID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch pools: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var pools []map[string]interface{}
	for rows.Next() {
		var poolID, poolName, resourceType, capacityMetric, strategy string
		var currentLoad int64
		var autoScaling bool
		var updated time.Time

		if err := rows.Scan(&poolID, &poolName, &resourceType, &currentLoad,
			&capacityMetric, &strategy, &autoScaling, &updated); err != nil {
			continue
		}

		pools = append(pools, map[string]interface{}{
			"pool_id":              poolID,
			"pool_name":            poolName,
			"resource_type":        resourceType,
			"current_load":         currentLoad,
			"capacity_metric":      capacityMetric,
			"routing_strategy":     strategy,
			"auto_scaling_enabled": autoScaling,
			"last_updated":         updated,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pools)
}

// ============================================================================
// Feature 15: Explainability Handlers
// ============================================================================

// GetExplainability retrieves explanation for a branch decision
// GET /api/bp/branching/{branchID}/explainability/{decisionID}
func (s *Server) GetExplainability(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	decisionID := chi.URLParam(r, "decisionID")

	if tenantID == "" || decisionID == "" {
		http.Error(w, "Missing tenant ID or decision ID", http.StatusBadRequest)
		return
	}

	query := `
		SELECT record_id, branch_id, feature_importance, decision_path,
		       natural_language_summary, confidence_score, created_at
		FROM bp_explainability_records
		WHERE record_id = $1 AND tenant_id = $2
	`

	var record map[string]interface{}
	var recordID, branchID, decisionPath, nlSummary string
	var importanceJSON []byte
	var confidence float64
	var createdAt time.Time

	err := s.DB.QueryRowContext(r.Context(), query, decisionID, tenantID).Scan(
		&recordID, &branchID, &importanceJSON, &decisionPath, &nlSummary, &confidence, &createdAt)

	if err == sql.ErrNoRows {
		http.Error(w, "Explainability record not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}

	var importance map[string]float64
	json.Unmarshal(importanceJSON, &importance)

	record = map[string]interface{}{
		"record_id":                recordID,
		"branch_id":                branchID,
		"feature_importance":       importance,
		"decision_path":            decisionPath,
		"natural_language_summary": nlSummary,
		"confidence_score":         confidence,
		"created_at":               createdAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(record)
}

// RegisterAdvancedHandlers registers all advanced feature handlers with the router
func (s *Server) RegisterAdvancedHandlers(r chi.Router) {
	// AI Models endpoints
	r.Get("/api/bp/branching/ai-models", s.ListAIModels)
	r.Post("/api/bp/branching/ai-models", s.CreateAIModel)

	// Semantic Intent endpoints
	r.Get("/api/bp/branching/semantic-intents", s.ListSemanticIntents)

	// Scoring Matrix endpoints
	r.Get("/api/bp/branching/scoring-matrices", s.ListScoringMatrices)

	// Time-Series Forecast endpoints
	r.Get("/api/bp/branching/forecasts/latest", s.GetLatestForecast)

	// Analytics endpoints
	r.Get("/api/bp/branching/{branchID}/analytics", s.GetBranchAnalytics)

	// Voting Decision endpoints
	r.Post("/api/bp/branching/voting-decisions", s.CreateVotingDecision)
	r.Post("/api/bp/branching/voting-decisions/{decisionID}/votes", s.CastVote)

	// Geofence endpoints
	r.Get("/api/bp/branching/geofences", s.ListGeofences)

	// Blockchain Audit endpoints
	r.Get("/api/bp/branching/blockchain-audit/{eventID}", s.GetBlockchainAudit)

	// Natural Language Config endpoints
	r.Post("/api/bp/branching/nl-config", s.CreateNLConfig)

	// Resource Pool endpoints
	r.Get("/api/bp/branching/resource-pools", s.ListResourcePools)

	// Explainability endpoints
	r.Get("/api/bp/branching/{branchID}/explainability/{decisionID}", s.GetExplainability)
}
