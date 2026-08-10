package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/db"
	"github.com/hondyman/uisce/backend/internal/metadata"
	"github.com/hondyman/uisce/backend/models"
	"github.com/jmoiron/sqlx"
)

// CatalogScanHandler handles catalog scanning requests
// scanServiceIface allows tests to inject a fake implementation.
type scanServiceIface interface {
	ScanDatasources(ctx context.Context, tenantDatasourceID *uuid.UUID) ([]metadata.ScanResult, error)
	ScanWithProgress(ctx context.Context, tenantDatasourceID *uuid.UUID, progress chan<- models.ScanProgress) ([]metadata.ScanResult, error)
}

type CatalogScanHandler struct {
	scanService   scanServiceIface
	securityDeps  SecurityContextDeps
}

// NewCatalogScanHandler creates a new catalog scan handler
func NewCatalogScanHandler(scanService scanServiceIface, securityDeps SecurityContextDeps) *CatalogScanHandler {
	return &CatalogScanHandler{
		scanService:  scanService,
		securityDeps: securityDeps,
	}
}

// RegisterRoutes registers the routes for CatalogScanHandler.
func (h *CatalogScanHandler) RegisterRoutes(r chi.Router) {
	r.Post("/api/catalog/scan", h.HandleCatalogScan)
	r.Get("/api/catalog/scan/stream", h.HandleScanStream)
}

// HandleCatalogScan handles POST requests to trigger catalog scans
func (h *CatalogScanHandler) HandleCatalogScan(w http.ResponseWriter, r *http.Request) {
	secCtx, ctx, err := SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Set content type
	w.Header().Set("Content-Type", "application/json")

	// Parse request body for datasource_id
	// Hasura Action payload wraps args in "input" object
	var requestBody struct {
		DatasourceID *string `json:"datasource_id,omitempty"`
		Input        struct {
			DatasourceID *string `json:"datasource_id,omitempty"`
		} `json:"input,omitempty"`
	}

	if r.Body != nil {
		// Try to parse JSON body
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil && err.Error() != "EOF" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON in request body"})
			return
		}
	}

	// Use security context datasource ID, but allow body to override for backward compatibility
	datasourceIDParam := secCtx.DatasourceID
	if requestBody.Input.DatasourceID != nil {
		datasourceIDParam = *requestBody.Input.DatasourceID
	} else if requestBody.DatasourceID != nil {
		datasourceIDParam = *requestBody.DatasourceID
	}

	var tenantDatasourceID *uuid.UUID
	if datasourceIDParam != "" {
		parsedID, err := uuid.Parse(datasourceIDParam)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid datasource_id format"})
			return
		}
		tenantDatasourceID = &parsedID
	}

	// Start the scan process
	results, err := h.scanService.ScanDatasources(ctx, tenantDatasourceID)

	// Ensure results is an empty array rather than nil so Hasura GraphQL (non-null list) accepts it
	if results == nil {
		results = []metadata.ScanResult{}
	}

	// Build base response
	response := map[string]interface{}{
		"results": results,
	}

	if tenantDatasourceID != nil {
		response["scanned_datasource_id"] = tenantDatasourceID.String()
	} else {
		response["scanned_datasources"] = "all"
	}

	// Count successes and failures
	var successCount, failureCount int
	for _, result := range results {
		if result.Success {
			successCount++
		} else {
			failureCount++
		}
	}

	switch {
	case failureCount == 0:
		response["status"] = "success"
		response["message"] = "All datasource scans completed successfully"
		w.WriteHeader(http.StatusOK)
	case successCount == 0:
		// All failed - still return 200 for Hasura to parse the response body
		response["status"] = "failure"
		response["message"] = "All datasource scans failed"
		if err != nil {
			response["details"] = err.Error()
		}
		w.WriteHeader(http.StatusOK)
	default:
		// Mixed results - return 200 for Hasura
		response["status"] = "partial"
		response["message"] = "Some datasource scans failed"
		response["success_count"] = successCount
		response["failure_count"] = failureCount
		w.WriteHeader(http.StatusOK)
	}

	json.NewEncoder(w).Encode(response)
}

// DebugHandler handles debug requests
type DebugHandler struct {
	DB           *sqlx.DB
	securityDeps SecurityContextDeps
}

// NewDebugHandler creates a new DebugHandler
func NewDebugHandler(db *sqlx.DB, securityDeps SecurityContextDeps) *DebugHandler {
	return &DebugHandler{DB: db, securityDeps: securityDeps}
}

// RegisterRoutes registers the routes for DebugHandler.
func (h *DebugHandler) RegisterRoutes(r chi.Router) {
	r.Get("/api/debug/chart", h.DebugChart)
}

// debugChart handles requests to debug chart data
func (h *DebugHandler) DebugChart(w http.ResponseWriter, r *http.Request) {
	secCtx, ctx, err := SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	datasourceID := secCtx.DatasourceID
	if datasourceID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "datasource_id required"})
		return
	}

	// Call the DebugChartData function from the db package
	if err := db.DebugChartData(ctx, h.DB.DB, datasourceID); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"message": "Debug output written to logs"})
}
