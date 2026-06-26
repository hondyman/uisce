package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/analytics"
)

// MetricsMigrationHandler handles metrics registry migration
type MetricsMigrationHandler struct {
	MigrationService *analytics.MetricsMigrationService
}

// NewMetricsMigrationHandler creates a new migration handler
func NewMetricsMigrationHandler(migrationService *analytics.MetricsMigrationService) *MetricsMigrationHandler {
	return &MetricsMigrationHandler{MigrationService: migrationService}
}

// RegisterRoutes registers migration routes
func (h *MetricsMigrationHandler) RegisterRoutes(r chi.Router) {
	r.Post("/api/admin/migrate-metrics", h.MigrateMetrics)
	r.Post("/api/admin/migrate-metrics/dry-run", h.DryRunMigration)
	r.Post("/api/admin/convert-dsl", h.ConvertDSL)
	r.Get("/api/admin/calculations", h.GetAvailableCalculations)
	r.Post("/api/admin/bo-assign-calc", h.AssignMetricToBO)
	r.Get("/api/admin/bo/{boName}/calculations", h.GetBOCalculations)
}

// MigrateMetricsRequest contains the migration parameters
type MigrateMetricsRequest struct {
	TenantID     string `json:"tenant_id"`
	DatasourceID string `json:"datasource_id"`
	DryRun       bool   `json:"dry_run"`
}

// MigrateMetrics performs the metrics migration
// POST /api/admin/migrate-metrics
func (h *MetricsMigrationHandler) MigrateMetrics(w http.ResponseWriter, r *http.Request) {
	var req MigrateMetricsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		http.Error(w, "Invalid tenant_id", http.StatusBadRequest)
		return
	}

	datasourceID, err := uuid.Parse(req.DatasourceID)
	if err != nil {
		http.Error(w, "Invalid datasource_id", http.StatusBadRequest)
		return
	}

	result, err := h.MigrationService.Migrate(tenantID, datasourceID, req.DryRun)
	if err != nil {
		http.Error(w, "Migration failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// DryRunMigration performs a dry-run migration
// POST /api/admin/migrate-metrics/dry-run
func (h *MetricsMigrationHandler) DryRunMigration(w http.ResponseWriter, r *http.Request) {
	var req MigrateMetricsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		http.Error(w, "Invalid tenant_id", http.StatusBadRequest)
		return
	}

	datasourceID, err := uuid.Parse(req.DatasourceID)
	if err != nil {
		http.Error(w, "Invalid datasource_id", http.StatusBadRequest)
		return
	}

	// Force dry run
	result, err := h.MigrationService.Migrate(tenantID, datasourceID, true)
	if err != nil {
		http.Error(w, "Dry-run migration failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// ConvertDSLRequest contains a formula to convert
type ConvertDSLRequest struct {
	Formula     string `json:"formula"`
	FormulaType string `json:"formula_type"`
}

// ConvertDSL converts a single formula to DSL
// POST /api/admin/convert-dsl
func (h *MetricsMigrationHandler) ConvertDSL(w http.ResponseWriter, r *http.Request) {
	var req ConvertDSLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	converter := analytics.NewDslConverter()
	result := converter.Convert(req.Formula, req.FormulaType)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GetAvailableCalculations returns all calculation terms for a datasource
// GET /api/admin/calculations?datasource_id=...
func (h *MetricsMigrationHandler) GetAvailableCalculations(w http.ResponseWriter, r *http.Request) {
	datasourceID, err := uuid.Parse(r.URL.Query().Get("datasource_id"))
	if err != nil {
		http.Error(w, "Invalid datasource_id", http.StatusBadRequest)
		return
	}

	calculations, err := h.MigrationService.GetAvailableCalculations(datasourceID)
	if err != nil {
		http.Error(w, "Failed to get calculations: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(calculations)
}

// AssignMetricRequest contains the BO-calc assignment parameters
type AssignMetricRequest struct {
	BOName       string   `json:"bo_name"`
	CalcNames    []string `json:"calc_names"`
	TenantID     string   `json:"tenant_id"`
	DatasourceID string   `json:"datasource_id"`
}

// AssignMetricToBO assigns calculations to a BO (creates BO_HAS_CALC edges)
// POST /api/admin/bo-assign-calc
func (h *MetricsMigrationHandler) AssignMetricToBO(w http.ResponseWriter, r *http.Request) {
	var req AssignMetricRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		http.Error(w, "Invalid tenant_id", http.StatusBadRequest)
		return
	}

	datasourceID, err := uuid.Parse(req.DatasourceID)
	if err != nil {
		http.Error(w, "Invalid datasource_id", http.StatusBadRequest)
		return
	}

	success, failed := h.MigrationService.BulkAssignMetricsToBO(
		req.BOName, req.CalcNames, tenantID, datasourceID,
	)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": success,
		"failed":  failed,
	})
}

// GetBOCalculations returns calculations assigned to a BO
// GET /api/admin/bo/{boName}/calculations?datasource_id=...
func (h *MetricsMigrationHandler) GetBOCalculations(w http.ResponseWriter, r *http.Request) {
	boName := chi.URLParam(r, "boName")
	datasourceID, err := uuid.Parse(r.URL.Query().Get("datasource_id"))
	if err != nil {
		http.Error(w, "Invalid datasource_id", http.StatusBadRequest)
		return
	}

	calculations, err := h.MigrationService.GetBOCalculations(boName, datasourceID)
	if err != nil {
		http.Error(w, "Failed to get calculations: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(calculations)
}
