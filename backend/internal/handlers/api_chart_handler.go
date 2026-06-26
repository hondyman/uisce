// api_chart_handler.go - API endpoints for serving chart data
package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/db"
)

// Enhanced chart response with better error handling
type ChartAPIResponse struct {
	Success   bool                   `json:"success"`
	Data      interface{}            `json:"data,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Metadata  map[string]interface{} `json:"metadata"`
	Timestamp string                 `json:"timestamp"`
	Debug     map[string]interface{} `json:"debug,omitempty"`
}

// ChartHandler handles API requests related to charts.
type ChartHandler struct {
	db *sql.DB
}

// NewChartHandler creates a new ChartHandler.
func NewChartHandler(db *sql.DB) *ChartHandler {
	return &ChartHandler{db: db}
}

// RegisterRoutes registers the routes for ChartHandler.
// RegisterRoutes registers the routes for ChartHandler.
func (h *ChartHandler) RegisterRoutes(r chi.Router) {
	r.Get("/chart/{datasourceId}/{chartType}", h.GetChartDataV2)
	r.Get("/charts/{datasourceId}", h.GetAllCharts)
	r.Post("/charts/{datasourceId}/refresh", h.RefreshCharts)
	r.Get("/chart/{datasourceId}/health", h.GetChartHealth)
	r.Get("/chart/{datasourceId}/debug", h.DebugChartDataAPI)

	// Legacy support
	r.Get("/lineage/{datasourceId}/{lineageType}", func(w http.ResponseWriter, r *http.Request) {
		datasourceId := chi.URLParam(r, "datasourceId")
		lineageType := chi.URLParam(r, "lineageType")
		http.Redirect(w, r, fmt.Sprintf("/api/chart/%s/%s", datasourceId, lineageType), http.StatusMovedPermanently)
	})
}

// GetChartDataV2 - Enhanced endpoint with better error handling and debugging
func (h *ChartHandler) GetChartDataV2(w http.ResponseWriter, r *http.Request) {
	datasourceId := chi.URLParam(r, "datasourceId")
	chartType := chi.URLParam(r, "chartType")
	debug := r.URL.Query().Get("debug") == "true"

	if datasourceId == "" || chartType == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ChartAPIResponse{
			Success:   false,
			Error:     "datasourceId and chartType are required",
			Timestamp: time.Now().Format(time.RFC3339),
			Metadata:  map[string]interface{}{},
		})
		return
	}

	// Validate and normalize chart type
	normalizedType, valid := validateAndNormalizeChartType(chartType)
	if !valid {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ChartAPIResponse{
			Success:   false,
			Error:     fmt.Sprintf("invalid chart type: %s. Valid types: erd, enhanced, technical, semantic, technical_lineage, semantic_lineage", chartType),
			Timestamp: time.Now().Format(time.RFC3339),
			Metadata:  map[string]interface{}{"validTypes": []string{"erd", "enhanced", "technical", "semantic", "technical_lineage", "semantic_lineage"}},
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// Debug info
	debugInfo := map[string]interface{}{
		"originalChartType": chartType,
		"normalizedType":    normalizedType,
		"datasourceId":      datasourceId,
		"requestTimestamp":  time.Now().Format(time.RFC3339),
	}

	// Get compressed chart data from database
	compressedData, err := db.GetLineageData(ctx, h.db, datasourceId, normalizedType)
	if err != nil {
		debugInfo["error"] = err.Error()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ChartAPIResponse{
			Success:   false,
			Error:     fmt.Sprintf("chart not found: %v", err),
			Timestamp: time.Now().Format(time.RFC3339),
			Metadata: map[string]interface{}{
				"chartType":  chartType,
				"dataSource": datasourceId,
			},
			Debug: debugInfo,
		})
		return
	}

	debugInfo["compressedSize"] = len(compressedData)

	// Decompress and parse chart data
	chartData, err := db.ParseChartData(compressedData, normalizedType)
	if err != nil {
		debugInfo["parseError"] = err.Error()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ChartAPIResponse{
			Success:   false,
			Error:     fmt.Sprintf("failed to parse chart data: %v", err),
			Timestamp: time.Now().Format(time.RFC3339),
			Debug:     debugInfo,
		})
		return
	}

	// Extract metadata and add debug info
	metadata := extractChartMetadata(chartData, chartType, datasourceId)

	if debug {
		metadata["debug"] = debugInfo
	}

	response := ChartAPIResponse{
		Success:   true,
		Data:      chartData,
		Timestamp: time.Now().Format(time.RFC3339),
		Metadata:  metadata,
	}

	if debug {
		response.Debug = debugInfo
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// validateAndNormalizeChartType validates and normalizes chart type names
func validateAndNormalizeChartType(chartType string) (string, bool) {
	switch chartType {
	case "erd", "erd_chart":
		return "erd", true
	case "enhanced", "enhanced_erd", "enhanced_erd_chart":
		return "enhanced", true
	case "technical", "technical_lineage", "technical_lineage_chart":
		return "technical", true
	case "semantic", "semantic_lineage", "semantic_lineage_chart":
		return "semantic", true
	case "semantic_raw", "semantic_lineage_raw":
		return "semantic_raw", true
	default:
		return "", false
	}
}

// extractChartMetadata extracts metadata from chart data
func extractChartMetadata(chartData interface{}, chartType, datasourceId string) map[string]interface{} {
	metadata := map[string]interface{}{
		"chartType":  chartType,
		"dataSource": datasourceId,
	}

	switch chart := chartData.(type) {
	case db.TechnicalLineageChart:
		metadata["nodeCount"] = len(chart.Nodes)
		metadata["edgeCount"] = len(chart.Edges)
		metadata["format"] = "reactflow"

		// Copy original metadata
		for k, v := range chart.Metadata {
			metadata[k] = v
		}

	case db.SemanticLineageChart:
		totalNodes := len(chart.BusinessTerms) + len(chart.SemanticTerms) +
			len(chart.SemanticColumns) + len(chart.DatabaseColumns)
		metadata["nodeCount"] = totalNodes
		metadata["edgeCount"] = len(chart.Edges)
		metadata["format"] = "semantic"
		metadata["businessTermCount"] = len(chart.BusinessTerms)
		metadata["semanticTermCount"] = len(chart.SemanticTerms)
		metadata["semanticColumnCount"] = len(chart.SemanticColumns)
		metadata["databaseColumnCount"] = len(chart.DatabaseColumns)

		// Copy original metadata
		for k, v := range chart.Metadata {
			metadata[k] = v
		}
	}

	return metadata
}

// GetAllCharts handles GET /api/charts/:datasourceId
func (h *ChartHandler) GetAllCharts(w http.ResponseWriter, r *http.Request) {
	datasourceId := chi.URLParam(r, "datasourceId")

	if datasourceId == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ChartAPIResponse{
			Success:   false,
			Error:     "datasourceId is required",
			Timestamp: time.Now().Format(time.RFC3339),
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// List all charts for this datasource
	charts, err := db.ListChartsForDatasource(ctx, h.db, datasourceId)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ChartAPIResponse{
			Success:   false,
			Error:     fmt.Sprintf("failed to list charts: %v", err),
			Timestamp: time.Now().Format(time.RFC3339),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ChartAPIResponse{
		Success:   true,
		Data:      charts,
		Timestamp: time.Now().Format(time.RFC3339),
		Metadata: map[string]interface{}{
			"dataSource": datasourceId,
		},
	})
}

// RefreshCharts handles POST /api/charts/:datasourceId/refresh
func (h *ChartHandler) RefreshCharts(w http.ResponseWriter, r *http.Request) {
	datasourceId := chi.URLParam(r, "datasourceId")

	if datasourceId == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ChartAPIResponse{
			Success:   false,
			Error:     "datasourceId is required",
			Timestamp: time.Now().Format(time.RFC3339),
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute) // Longer timeout for refresh
	defer cancel()

	// Refresh all charts
	err := db.RefreshAllCharts(ctx, h.db, datasourceId, false)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ChartAPIResponse{
			Success:   false,
			Error:     fmt.Sprintf("failed to refresh charts: %v", err),
			Timestamp: time.Now().Format(time.RFC3339),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ChartAPIResponse{
		Success:   true,
		Timestamp: time.Now().Format(time.RFC3339),
		Metadata: map[string]interface{}{
			"dataSource": datasourceId,
		},
	})
}

// GetChartHealth handles GET /api/chart/:datasourceId/health
func (h *ChartHandler) GetChartHealth(w http.ResponseWriter, r *http.Request) {
	datasourceId := chi.URLParam(r, "datasourceId")

	if datasourceId == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ChartAPIResponse{
			Success:   false,
			Error:     "datasourceId is required",
			Timestamp: time.Now().Format(time.RFC3339),
			Metadata:  map[string]interface{}{},
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// Validate chart integrity
	integrity, err := db.ValidateChartIntegrity(ctx, h.db, datasourceId)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ChartAPIResponse{
			Success:   false,
			Error:     fmt.Sprintf("failed to validate charts: %v", err),
			Timestamp: time.Now().Format(time.RFC3339),
		})
		return
	}

	// Get chart list with sizes
	charts, err := db.ListChartsForDatasource(ctx, h.db, datasourceId)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ChartAPIResponse{
			Success:   false,
			Error:     fmt.Sprintf("failed to list charts: %v", err),
			Timestamp: time.Now().Format(time.RFC3339),
		})
		return
	}

	healthData := map[string]interface{}{
		"integrity": integrity,
		"charts":    charts,
		"summary": map[string]interface{}{
			"totalCharts":   len(charts),
			"healthyCharts": countHealthyCharts(integrity),
			"datasourceId":  datasourceId,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ChartAPIResponse{
		Success:   true,
		Data:      healthData,
		Timestamp: time.Now().Format(time.RFC3339),
		Metadata: map[string]interface{}{
			"dataSource": datasourceId,
		},
	})
}

// countHealthyCharts counts how many charts are healthy
func countHealthyCharts(integrity map[string]bool) int {
	count := 0
	for _, healthy := range integrity {
		if healthy {
			count++
		}
	}
	return count
}

// DebugChartDataAPI handles GET /api/chart/:datasourceId/debug
func (h *ChartHandler) DebugChartDataAPI(w http.ResponseWriter, r *http.Request) {
	datasourceId := chi.URLParam(r, "datasourceId")

	if datasourceId == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ChartAPIResponse{
			Success:   false,
			Error:     "datasourceId is required",
			Timestamp: time.Now().Format(time.RFC3339),
			Metadata:  map[string]interface{}{},
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// Run debug analysis
	err := db.DebugChartData(ctx, h.db, datasourceId)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ChartAPIResponse{
			Success:   false,
			Error:     fmt.Sprintf("debug failed: %v", err),
			Timestamp: time.Now().Format(time.RFC3339),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ChartAPIResponse{
		Success:   true,
		Data:      "Debug output written to logs",
		Timestamp: time.Now().Format(time.RFC3339),
		Metadata: map[string]interface{}{
			"dataSource": datasourceId,
			"message":    "Check server logs for detailed debug information",
		},
	})
}
