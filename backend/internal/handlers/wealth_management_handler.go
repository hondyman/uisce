package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// WealthManagementHandler handles API requests for wealth management metrics.
type WealthManagementHandler struct {
	db *sql.DB
}

// NewWealthManagementHandler creates a new WealthManagementHandler.
func NewWealthManagementHandler(db *sql.DB) *WealthManagementHandler {
	return &WealthManagementHandler{db: db}
}

// WealthManagementMetric represents a wealth management metric from the registry.
type WealthManagementMetric struct {
	NodeID           string            `json:"node_id"`
	Category         string            `json:"category"`
	Description      string            `json:"description"`
	GovernanceStatus string            `json:"governance_status"`
	FormulaType      string            `json:"formula_type"`
	Formula          string            `json:"formula,omitempty"`
	Arguments        map[string]string `json:"arguments,omitempty"`
	Audience         []string          `json:"audience"`
	Tags             []string          `json:"tags"`
	CreatedAt        *string           `json:"created_at,omitempty"`
	UpdatedAt        *string           `json:"updated_at,omitempty"`
}

// MetricCalculation represents a metric calculation result.
type MetricCalculation struct {
	MetricID    string                 `json:"metric_id"`
	Value       float64                `json:"value"`
	Timestamp   string                 `json:"timestamp"`
	GrainValues map[string]interface{} `json:"grain_values,omitempty"`
}

// HandleListMetrics lists all wealth management metrics for a tenant.
func (h *WealthManagementHandler) HandleListMetrics(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenantId")
	if tenantID == "" {
		http.Error(w, "tenantId query parameter is required", http.StatusBadRequest)
		return
	}

	// Parse tenant ID
	_, err := uuid.Parse(tenantID)
	if err != nil {
		http.Error(w, "Invalid tenantId format", http.StatusBadRequest)
		return
	}

	query := `
		SELECT
			node_id,
			category,
			description,
			governance_status,
			formula_type,
			formula,
			arguments,
			audience,
			tags,
			created_at,
			updated_at
		FROM public.metrics_registry
		WHERE schema_domain = 'wealth_management'
		ORDER BY category, node_id
	`

	rows, err := h.db.QueryContext(r.Context(), query)
	if err != nil {
		http.Error(w, "Failed to query metrics: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var metrics []WealthManagementMetric
	for rows.Next() {
		var metric WealthManagementMetric
		var argumentsJSON, audienceJSON, tagsJSON []byte
		var createdAt, updatedAt sql.NullString

		err := rows.Scan(
			&metric.NodeID,
			&metric.Category,
			&metric.Description,
			&metric.GovernanceStatus,
			&metric.FormulaType,
			&metric.Formula,
			&argumentsJSON,
			&audienceJSON,
			&tagsJSON,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			http.Error(w, "Failed to scan metric: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Parse JSON fields
		if len(argumentsJSON) > 0 {
			json.Unmarshal(argumentsJSON, &metric.Arguments)
		}
		if len(audienceJSON) > 0 {
			json.Unmarshal(audienceJSON, &metric.Audience)
		}
		if len(tagsJSON) > 0 {
			json.Unmarshal(tagsJSON, &metric.Tags)
		}

		if createdAt.Valid {
			metric.CreatedAt = &createdAt.String
		}
		if updatedAt.Valid {
			metric.UpdatedAt = &updatedAt.String
		}

		metrics = append(metrics, metric)
	}

	if err = rows.Err(); err != nil {
		http.Error(w, "Error iterating metrics: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"metrics": metrics,
		"count":   len(metrics),
	})
}

// HandleGetMetric retrieves a specific metric by ID.
func (h *WealthManagementHandler) HandleGetMetric(w http.ResponseWriter, r *http.Request) {
	metricID := chi.URLParam(r, "metricId")
	tenantID := r.URL.Query().Get("tenantId")

	if metricID == "" {
		http.Error(w, "metricId parameter is required", http.StatusBadRequest)
		return
	}

	if tenantID == "" {
		http.Error(w, "tenantId query parameter is required", http.StatusBadRequest)
		return
	}

	query := `
		SELECT
			node_id,
			category,
			description,
			governance_status,
			formula_type,
			formula,
			arguments,
			audience,
			tags,
			created_at,
			updated_at
		FROM public.metrics_registry
		WHERE schema_domain = 'wealth_management' AND node_id = $1
	`

	var metric WealthManagementMetric
	var argumentsJSON, audienceJSON, tagsJSON []byte
	var createdAt, updatedAt sql.NullString

	err := h.db.QueryRowContext(r.Context(), query, metricID).Scan(
		&metric.NodeID,
		&metric.Category,
		&metric.Description,
		&metric.GovernanceStatus,
		&metric.FormulaType,
		&metric.Formula,
		&argumentsJSON,
		&audienceJSON,
		&tagsJSON,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Metric not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to query metric: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Parse JSON fields
	if len(argumentsJSON) > 0 {
		json.Unmarshal(argumentsJSON, &metric.Arguments)
	}
	if len(audienceJSON) > 0 {
		json.Unmarshal(audienceJSON, &metric.Audience)
	}
	if len(tagsJSON) > 0 {
		json.Unmarshal(tagsJSON, &metric.Tags)
	}

	if createdAt.Valid {
		metric.CreatedAt = &createdAt.String
	}
	if updatedAt.Valid {
		metric.UpdatedAt = &updatedAt.String
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"metric": metric,
	})
}

// HandleGetMetricCalculations retrieves calculation results for specific metrics.
func (h *WealthManagementHandler) HandleGetMetricCalculations(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenantId")
	metricIDs := r.URL.Query()["metricIds"]
	clientID := r.URL.Query().Get("clientId")

	if tenantID == "" {
		http.Error(w, "tenantId query parameter is required", http.StatusBadRequest)
		return
	}

	if len(metricIDs) == 0 {
		http.Error(w, "metricIds query parameter is required", http.StatusBadRequest)
		return
	}

	// For now, return mock calculation data
	// In a real implementation, this would query the preaggregated results
	var calculations []MetricCalculation

	for _, metricID := range metricIDs {
		// Mock calculation result
		calc := MetricCalculation{
			MetricID:  metricID,
			Value:     0.0, // Would be calculated from preaggregated data
			Timestamp: "2024-01-01T00:00:00Z",
			GrainValues: map[string]interface{}{
				"client_id": clientID,
				"period":    "monthly",
			},
		}
		calculations = append(calculations, calc)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"calculations": calculations,
		"note":         "Mock data - implement actual calculation logic",
	})
}

// HandleRefreshMetrics triggers a refresh of metric calculations.
func (h *WealthManagementHandler) HandleRefreshMetrics(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantID  string   `json:"tenantId"`
		MetricIDs []string `json:"metricIds,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	if req.TenantID == "" {
		http.Error(w, "tenantId is required", http.StatusBadRequest)
		return
	}

	// In a real implementation, this would trigger the preaggregation scheduler
	// For now, just return success

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"message":     "Metric refresh initiated",
		"tenantId":    req.TenantID,
		"metricCount": len(req.MetricIDs),
	})
}

// RegisterRoutes registers the wealth management routes.
func (h *WealthManagementHandler) RegisterRoutes(r chi.Router) {
	r.Route("/wealth-management", func(r chi.Router) {
		r.Get("/metrics", h.HandleListMetrics)
		r.Get("/metrics/{metricId}", h.HandleGetMetric)
		r.Get("/calculations", h.HandleGetMetricCalculations)
		r.Post("/refresh", h.HandleRefreshMetrics)
	})
}
