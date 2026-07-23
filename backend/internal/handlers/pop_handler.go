package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// PoPMetric represents a period-over-period metric
type PoPMetric struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Domain      string    `json:"domain"`
	Category    string    `json:"category"`
	Status      string    `json:"status"`
	GoldenPath  bool      `json:"golden_path"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// PoPHandler handles Period-over-Period metric operations
type PoPHandler struct {
	db *sql.DB
}

// NewPoPHandler creates a new PoP handler
func NewPoPHandler(db *sql.DB) *PoPHandler {
	return &PoPHandler{db: db}
}

// GetPoPManifest returns enriched PoP manifest JSON
func (h *PoPHandler) GetPoPManifest(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT id, name, description, domain, category, status, golden_path, updated_at
		FROM public.pop_metrics
		ORDER BY domain, category, name
	`

	rows, err := h.db.Query(query)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to retrieve PoP manifest"})
		return
	}
	defer rows.Close()

	var metrics []PoPMetric
	for rows.Next() {
		var metric PoPMetric
		err := rows.Scan(&metric.ID, &metric.Name, &metric.Description, &metric.Domain,
			&metric.Category, &metric.Status, &metric.GoldenPath, &metric.UpdatedAt)
		if err != nil {
			continue
		}
		metrics = append(metrics, metric)
	}

	// Enrich with governance and contract information
	manifest := map[string]interface{}{
		"metrics":       metrics,
		"total_count":   len(metrics),
		"golden_path":   h.countGoldenPath(metrics),
		"anomaly_count": h.countAnomalies(metrics),
		"last_updated":  h.getLastUpdated(metrics),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(manifest)
}

// GetPoPMetric returns full metadata for a PoP metric
func (h *PoPHandler) GetPoPMetric(w http.ResponseWriter, r *http.Request) {
	metricID := chi.URLParam(r, "id")

	query := `
		SELECT id, name, description, domain, category, status, golden_path, updated_at
		FROM public.pop_metrics
		WHERE id = $1
	`

	var metric PoPMetric
	err := h.db.QueryRow(query, metricID).Scan(
		&metric.ID, &metric.Name, &metric.Description, &metric.Domain,
		&metric.Category, &metric.Status, &metric.GoldenPath, &metric.UpdatedAt,
	)

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]interface{}{"error": "PoP metric not found"})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to retrieve metric"})
		return
	}

	// Get anomaly count for this metric
	anomalyCount := h.getAnomalyCountForMetric(metricID)

	response := map[string]interface{}{
		"metric":        metric,
		"anomaly_count": anomalyCount,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// AnalyzePoPMetric runs anomaly detection on a metric
func (h *PoPHandler) AnalyzePoPMetric(w http.ResponseWriter, r *http.Request) {
	metricID := chi.URLParam(r, "id")

	var req struct {
		Method string `json:"method" binding:"required"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid request body"})
		return
	}

	// For now, simulate anomaly detection with a simple query
	// In a real implementation, this would run the actual detection algorithm
	anomalies := []map[string]interface{}{
		{
			"id":         "anomaly_1",
			"metric_id":  metricID,
			"severity":   "medium",
			"confidence": 0.85,
			"timestamp":  time.Now().Format(time.RFC3339),
		},
	}

	result := map[string]interface{}{
		"success":        true,
		"anomalies":      anomalies,
		"count":          len(anomalies),
		"method_used":    req.Method,
		"detection_time": time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// PromotePoPMetric marks a metric as golden path
func (h *PoPHandler) PromotePoPMetric(w http.ResponseWriter, r *http.Request) {
	metricID := chi.URLParam(r, "id")

	var req struct {
		Reason string `json:"reason" binding:"required"`
		UserID string `json:"user_id" binding:"required"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid request body"})
		return
	}

	// Update the metric to golden path
	query := `
		UPDATE public.pop_metrics
		SET golden_path = true, updated_at = NOW()
		WHERE id = $1
	`

	_, err := h.db.Exec(query, metricID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to promote metric"})
		return
	}

	// Log the promotion action using steward comments
	logQuery := `
		INSERT INTO public.pop_steward_comments (metric_id, commenter_user_id, comment_type, comment_text, created_at)
		VALUES ($1, $2, 'golden_path', $3, NOW())
	`
	_, _ = h.db.Exec(logQuery, metricID, req.UserID, "Promoted to golden path: "+req.Reason) // Ignore error for logging

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"message": "Metric promoted to golden path"})
}

// FlagPoPAnomaly flags an anomaly for steward review
func (h *PoPHandler) FlagPoPAnomaly(w http.ResponseWriter, r *http.Request) {
	metricID := chi.URLParam(r, "id")

	var req struct {
		AnomalyID string `json:"anomaly_id" binding:"required"`
		Severity  string `json:"severity" binding:"required"`
		Notes     string `json:"notes"`
		UserID    string `json:"user_id" binding:"required"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid request body"})
		return
	}

	// Log the flag action using steward comments
	query := `
		INSERT INTO public.pop_steward_comments (metric_id, commenter_user_id, comment_type, comment_text, created_at)
		VALUES ($1, $2, 'anomaly_feedback', $3, NOW())
	`

	_, err := h.db.Exec(query, metricID, req.UserID, "Anomaly flagged for review - Severity: "+req.Severity+" - Notes: "+req.Notes)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to flag anomaly"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"message": "Anomaly flagged for review"})
}

// AddPoPComment adds a steward comment to a metric
func (h *PoPHandler) AddPoPComment(w http.ResponseWriter, r *http.Request) {
	metricID := chi.URLParam(r, "id")

	var req struct {
		Comment  string `json:"comment" binding:"required"`
		UserID   string `json:"user_id" binding:"required"`
		Category string `json:"category"` // e.g., "anomaly_review", "golden_path", "general"
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid request body"})
		return
	}

	// Log the comment
	query := `
		INSERT INTO public.pop_steward_comments (metric_id, commenter_user_id, comment_type, comment_text, created_at)
		VALUES ($1, $2, $3, $4, NOW())
	`

	_, err := h.db.Exec(query, metricID, req.UserID, req.Category, req.Comment)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to add comment"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"message": "Comment added successfully"})
}

// Helper methods

func (h *PoPHandler) countGoldenPath(metrics []PoPMetric) int {
	count := 0
	for _, metric := range metrics {
		if metric.GoldenPath {
			count++
		}
	}
	return count
}

func (h *PoPHandler) countAnomalies(_ []PoPMetric) int {
	// Count anomalies from the anomalies table
	query := `
		SELECT COUNT(*) FROM public.pop_anomalies
		WHERE status = 'open' AND detected_at > NOW() - INTERVAL '30 days'
	`

	var count int
	_ = h.db.QueryRow(query).Scan(&count) // Ignore error
	return count
}

func (h *PoPHandler) getLastUpdated(metrics []PoPMetric) interface{} {
	if len(metrics) == 0 {
		return nil
	}

	latest := metrics[0].UpdatedAt
	for _, metric := range metrics[1:] {
		if metric.UpdatedAt.After(latest) {
			latest = metric.UpdatedAt
		}
	}

	return latest
}

func (h *PoPHandler) getAnomalyCountForMetric(metricID string) int {
	query := `
		SELECT COUNT(*) FROM public.pop_anomalies
		WHERE metric_id = $1 AND status = 'open' AND detected_at > NOW() - INTERVAL '30 days'
	`

	var count int
	_ = h.db.QueryRow(query, metricID).Scan(&count) // Ignore error
	return count
}
