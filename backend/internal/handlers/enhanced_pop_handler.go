package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/dynamic"
	"github.com/hondyman/semlayer/backend/internal/query"
	"github.com/hondyman/semlayer/backend/models"
)

// EnhancedPoPHandler extends the base PoP handler with dynamic capabilities
type EnhancedPoPHandler struct {
	*PoPHandler
	dynamicEngine *dynamic.DynamicQueryEngine
	templateMgr   *query.QueryTemplateManager
	db            *sql.DB
}

// NewEnhancedPoPHandler creates a new enhanced PoP handler with dynamic capabilities
func NewEnhancedPoPHandler(db *sql.DB, dynamicEngine *dynamic.DynamicQueryEngine, templateMgr *query.QueryTemplateManager) *EnhancedPoPHandler {
	return &EnhancedPoPHandler{
		PoPHandler:    NewPoPHandler(db),
		dynamicEngine: dynamicEngine,
		templateMgr:   templateMgr,
		db:            db,
	}
}

// RegisterRoutes registers the routes for EnhancedPoPHandler.
func (h *EnhancedPoPHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/pop/enhanced", func(r chi.Router) {
		r.Post("/analysis", h.HandleDynamicPoPAnalysis)
		r.Post("/anomaly-detection", h.HandleDynamicAnomalyDetection)
		r.Post("/steward-review", h.HandleDynamicStewardReview)
		r.Post("/dashboard-analysis", h.HandleDynamicDashboardAnalysis)
		r.Post("/metric-comparison", h.HandleDynamicMetricComparison)
	})
}

// HandleDynamicPoPAnalysis performs dynamic analysis on PoP metrics
func (h *EnhancedPoPHandler) HandleDynamicPoPAnalysis(w http.ResponseWriter, r *http.Request) {
	var req DynamicPoPAnalysisRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Build dynamic query for PoP analysis
	dynamicReq := &dynamic.DynamicQueryRequest{
		BaseQuery: &models.Query{
			TableName:  "pop_computations",
			Metrics:    req.Metrics,
			Dimensions: req.Dimensions,
			Filters:    convertToFilters(req.Filters),
		},
		Parameters:      req.Parameters,
		DynamicMeasures: req.DynamicMeasures,
		TimeRange:       req.TimeRange,
		Context:         req.Context,
	}

	// Resolve parameters
	ctx := r.Context()
	resolved, err := h.dynamicEngine.ResolveParameters(ctx, dynamicReq)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Parameter resolution failed",
			"details": err.Error(),
		})
		return
	}

	// Generate and execute SQL
	sql, args := resolved.BuildSQL()

	// Execute the query
	rows, err := h.db.Query(sql, args...)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Query execution failed",
			"details": err.Error(),
			"sql":     sql,
		})
		return
	}
	defer rows.Close()

	// Process results
	results, err := h.processQueryResults(rows, resolved)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Result processing failed",
			"details": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"query":         resolved,
		"results":       results,
		"sql_generated": sql,
		"parameters":    resolved.Parameters,
		"generated_at":  time.Now(),
	})
}

// HandleDynamicAnomalyDetection performs dynamic anomaly detection
func (h *EnhancedPoPHandler) HandleDynamicAnomalyDetection(w http.ResponseWriter, r *http.Request) {
	var req DynamicAnomalyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Build dynamic query for anomaly detection
	dynamicReq := &dynamic.DynamicQueryRequest{
		BaseQuery: &models.Query{
			TableName:  "pop_anomalies",
			Metrics:    req.Metrics,
			Dimensions: req.Dimensions,
			Filters:    convertToFilters(req.Filters),
		},
		Parameters:      req.Parameters,
		DynamicMeasures: req.DynamicMeasures,
		TimeRange:       req.TimeRange,
		Context:         req.Context,
	}

	// Resolve parameters
	ctx := r.Context()
	resolved, err := h.dynamicEngine.ResolveParameters(ctx, dynamicReq)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Parameter resolution failed",
			"details": err.Error(),
		})
		return
	}

	// Generate and execute SQL
	sql, args := resolved.BuildSQL()

	// Execute the query
	rows, err := h.db.Query(sql, args...)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Query execution failed",
			"details": err.Error(),
			"sql":     sql,
		})
		return
	}
	defer rows.Close()

	// Process results
	results, err := h.processQueryResults(rows, resolved)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Result processing failed",
			"details": err.Error(),
		})
		return
	}

	// Add anomaly-specific insights
	anomalyInsights := h.generateAnomalyInsights(results, req.Parameters)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"query":         resolved,
		"results":       results,
		"insights":      anomalyInsights,
		"sql_generated": sql,
		"parameters":    resolved.Parameters,
		"generated_at":  time.Now(),
	})
}

// HandleDynamicStewardReview performs dynamic steward review analysis
func (h *EnhancedPoPHandler) HandleDynamicStewardReview(w http.ResponseWriter, r *http.Request) {
	var req DynamicStewardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Build dynamic query for steward reviews
	dynamicReq := &dynamic.DynamicQueryRequest{
		BaseQuery: &models.Query{
			TableName:  "pop_steward_reviews",
			Metrics:    req.Metrics,
			Dimensions: req.Dimensions,
			Filters:    convertToFilters(req.Filters),
		},
		Parameters:      req.Parameters,
		DynamicMeasures: req.DynamicMeasures,
		TimeRange:       req.TimeRange,
		Context:         req.Context,
	}

	// Resolve parameters
	ctx := r.Context()
	resolved, err := h.dynamicEngine.ResolveParameters(ctx, dynamicReq)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Parameter resolution failed",
			"details": err.Error(),
		})
		return
	}

	// Generate and execute SQL
	sql, args := resolved.BuildSQL()

	// Execute the query
	rows, err := h.db.Query(sql, args...)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Query execution failed",
			"details": err.Error(),
			"sql":     sql,
		})
		return
	}
	defer rows.Close()

	// Process results
	results, err := h.processQueryResults(rows, resolved)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Result processing failed",
			"details": err.Error(),
		})
		return
	}

	// Add steward-specific insights
	stewardInsights := h.generateStewardInsights(results, req.Parameters)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"query":         resolved,
		"results":       results,
		"insights":      stewardInsights,
		"sql_generated": sql,
		"parameters":    resolved.Parameters,
		"generated_at":  time.Now(),
	})
}

// HandleDynamicDashboardAnalysis performs dynamic dashboard analysis
func (h *EnhancedPoPHandler) HandleDynamicDashboardAnalysis(w http.ResponseWriter, r *http.Request) {
	var req DynamicDashboardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Build dynamic query for dashboard analysis
	dynamicReq := &dynamic.DynamicQueryRequest{
		BaseQuery: &models.Query{
			TableName:  "pop_dashboards",
			Metrics:    req.Metrics,
			Dimensions: req.Dimensions,
			Filters:    convertToFilters(req.Filters),
		},
		Parameters:      req.Parameters,
		DynamicMeasures: req.DynamicMeasures,
		TimeRange:       req.TimeRange,
		Context:         req.Context,
	}

	// Resolve parameters
	ctx := r.Context()
	resolved, err := h.dynamicEngine.ResolveParameters(ctx, dynamicReq)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Parameter resolution failed",
			"details": err.Error(),
		})
		return
	}

	// Generate and execute SQL
	sql, args := resolved.BuildSQL()

	// Execute the query
	rows, err := h.db.Query(sql, args...)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Query execution failed",
			"details": err.Error(),
			"sql":     sql,
		})
		return
	}
	defer rows.Close()

	// Process results
	results, err := h.processQueryResults(rows, resolved)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Result processing failed",
			"details": err.Error(),
		})
		return
	}

	// Add dashboard-specific insights
	dashboardInsights := h.generateDashboardInsights(results, req.Parameters)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"query":         resolved,
		"results":       results,
		"insights":      dashboardInsights,
		"sql_generated": sql,
		"parameters":    resolved.Parameters,
		"generated_at":  time.Now(),
	})
}

// HandleDynamicMetricComparison performs dynamic metric comparison
func (h *EnhancedPoPHandler) HandleDynamicMetricComparison(w http.ResponseWriter, r *http.Request) {
	var req DynamicComparisonRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Build dynamic query for metric comparison
	dynamicReq := &dynamic.DynamicQueryRequest{
		BaseQuery: &models.Query{
			TableName:  "pop_metrics",
			Metrics:    req.Metrics,
			Dimensions: req.Dimensions,
			Filters:    convertToFilters(req.Filters),
		},
		Parameters:      req.Parameters,
		DynamicMeasures: req.DynamicMeasures,
		TimeRange:       req.TimeRange,
		Context:         req.Context,
	}

	// Resolve parameters
	ctx := r.Context()
	resolved, err := h.dynamicEngine.ResolveParameters(ctx, dynamicReq)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Parameter resolution failed",
			"details": err.Error(),
		})
		return
	}

	// Generate and execute SQL
	sql, args := resolved.BuildSQL()

	// Execute the query
	rows, err := h.db.Query(sql, args...)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Query execution failed",
			"details": err.Error(),
			"sql":     sql,
		})
		return
	}
	defer rows.Close()

	// Process results
	results, err := h.processQueryResults(rows, resolved)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Result processing failed",
			"details": err.Error(),
		})
		return
	}

	// Add comparison-specific insights
	comparisonInsights := h.generateComparisonInsights(results, req.Parameters)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"query":         resolved,
		"results":       results,
		"insights":      comparisonInsights,
		"sql_generated": sql,
		"parameters":    resolved.Parameters,
		"generated_at":  time.Now(),
	})
}

// Helper methods

func (h *EnhancedPoPHandler) processQueryResults(rows *sql.Rows, _ *dynamic.ResolvedQuery) ([]map[string]interface{}, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			continue
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				val = string(b)
			}
			row[col] = val
		}

		results = append(results, row)
	}

	return results, nil
}

func (h *EnhancedPoPHandler) generateAnomalyInsights(results []map[string]interface{}, _ []dynamic.DynamicParameter) map[string]interface{} {
	insights := map[string]interface{}{
		"total_anomalies": len(results),
		"severity_breakdown": map[string]int{
			"critical": 0,
			"high":     0,
			"medium":   0,
			"low":      0,
		},
		"anomaly_types": make(map[string]int),
	}

	for _, result := range results {
		if severity, ok := result["severity"].(string); ok {
			if count, exists := insights["severity_breakdown"].(map[string]int)[severity]; exists {
				insights["severity_breakdown"].(map[string]int)[severity] = count + 1
			}
		}

		if anomalyType, ok := result["anomaly_type"].(string); ok {
			insights["anomaly_types"].(map[string]int)[anomalyType]++
		}
	}

	return insights
}

func (h *EnhancedPoPHandler) generateStewardInsights(results []map[string]interface{}, _ []dynamic.DynamicParameter) map[string]interface{} {
	insights := map[string]interface{}{
		"total_reviews": len(results),
		"status_breakdown": map[string]int{
			"in_progress": 0,
			"completed":   0,
			"overdue":     0,
		},
		"review_types": make(map[string]int),
	}

	for _, result := range results {
		if status, ok := result["status"].(string); ok {
			if count, exists := insights["status_breakdown"].(map[string]int)[status]; exists {
				insights["status_breakdown"].(map[string]int)[status] = count + 1
			}
		}

		if reviewType, ok := result["review_type"].(string); ok {
			insights["review_types"].(map[string]int)[reviewType]++
		}
	}

	return insights
}

func (h *EnhancedPoPHandler) generateDashboardInsights(results []map[string]interface{}, _ []dynamic.DynamicParameter) map[string]interface{} {
	insights := map[string]interface{}{
		"total_dashboards":  len(results),
		"public_dashboards": 0,
		"owner_breakdown":   make(map[string]int),
	}

	for _, result := range results {
		if isPublic, ok := result["is_public"].(bool); ok && isPublic {
			insights["public_dashboards"] = insights["public_dashboards"].(int) + 1
		}

		if owner, ok := result["owner_user_id"].(string); ok {
			insights["owner_breakdown"].(map[string]int)[owner]++
		}
	}

	return insights
}

func (h *EnhancedPoPHandler) generateComparisonInsights(results []map[string]interface{}, _ []dynamic.DynamicParameter) map[string]interface{} {
	insights := map[string]interface{}{
		"total_metrics":      len(results),
		"domain_breakdown":   make(map[string]int),
		"category_breakdown": make(map[string]int),
		"golden_path_count":  0,
	}

	for _, result := range results {
		if domain, ok := result["domain"].(string); ok {
			insights["domain_breakdown"].(map[string]int)[domain]++
		}

		if category, ok := result["category"].(string); ok {
			insights["category_breakdown"].(map[string]int)[category]++
		}

		if goldenPath, ok := result["golden_path"].(bool); ok && goldenPath {
			insights["golden_path_count"] = insights["golden_path_count"].(int) + 1
		}
	}

	return insights
}

func convertToFilters(filters []Filter) []models.Filter {
	result := make([]models.Filter, len(filters))
	for i, f := range filters {
		result[i] = models.Filter{
			Field:  f.Field,
			Op:     f.Operator,
			Values: f.Values,
		}
	}
	return result
}

// Request/Response types

type DynamicPoPAnalysisRequest struct {
	Metrics         []string                   `json:"metrics"`
	Dimensions      []string                   `json:"dimensions"`
	Filters         []Filter                   `json:"filters"`
	Parameters      []dynamic.DynamicParameter `json:"parameters"`
	DynamicMeasures []dynamic.DynamicMeasure   `json:"dynamic_measures"`
	TimeRange       *query.TimeRange           `json:"time_range,omitempty"`
	Context         map[string]interface{}     `json:"context,omitempty"`
}

type DynamicAnomalyRequest struct {
	Metrics         []string                   `json:"metrics"`
	Dimensions      []string                   `json:"dimensions"`
	Filters         []Filter                   `json:"filters"`
	Parameters      []dynamic.DynamicParameter `json:"parameters"`
	DynamicMeasures []dynamic.DynamicMeasure   `json:"dynamic_measures"`
	TimeRange       *query.TimeRange           `json:"time_range,omitempty"`
	Context         map[string]interface{}     `json:"context,omitempty"`
}

type DynamicStewardRequest struct {
	Metrics         []string                   `json:"metrics"`
	Dimensions      []string                   `json:"dimensions"`
	Filters         []Filter                   `json:"filters"`
	Parameters      []dynamic.DynamicParameter `json:"parameters"`
	DynamicMeasures []dynamic.DynamicMeasure   `json:"dynamic_measures"`
	TimeRange       *query.TimeRange           `json:"time_range,omitempty"`
	Context         map[string]interface{}     `json:"context,omitempty"`
}

type DynamicDashboardRequest struct {
	Metrics         []string                   `json:"metrics"`
	Dimensions      []string                   `json:"dimensions"`
	Filters         []Filter                   `json:"filters"`
	Parameters      []dynamic.DynamicParameter `json:"parameters"`
	DynamicMeasures []dynamic.DynamicMeasure   `json:"dynamic_measures"`
	TimeRange       *query.TimeRange           `json:"time_range,omitempty"`
	Context         map[string]interface{}     `json:"context,omitempty"`
}

type DynamicComparisonRequest struct {
	Metrics         []string                   `json:"metrics"`
	Dimensions      []string                   `json:"dimensions"`
	Filters         []Filter                   `json:"filters"`
	Parameters      []dynamic.DynamicParameter `json:"parameters"`
	DynamicMeasures []dynamic.DynamicMeasure   `json:"dynamic_measures"`
	TimeRange       *query.TimeRange           `json:"time_range,omitempty"`
	Context         map[string]interface{}     `json:"context,omitempty"`
}
