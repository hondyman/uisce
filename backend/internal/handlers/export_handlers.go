package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/backend/internal/services"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// ExportHandlers provides export API endpoints
type ExportHandlers struct {
	exportService services.ExportService
}

// NewExportHandlers creates new export handlers
func NewExportHandlers(es services.ExportService) *ExportHandlers {
	return &ExportHandlers{
		exportService: es,
	}
}

// CreateExport creates a new export job (POST /api/v1/jobs/:jobId/exports)
func (h *ExportHandlers) CreateExport(w http.ResponseWriter, r *http.Request) {
	vars := chi.URLParam(r, "jobId")
	jobID, err := uuid.Parse(vars)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid job ID")
		return
	}

	tenantNorm := normalizeTenantID(jwtmiddleware.GetClaimsFromContext(r).TenantID)
	ctx := setupAuthContext(r.Context(), tenantNorm)

	var req models.CreateExportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request: %v", err))
		return
	}

	exportID, err := h.exportService.CreateExport(ctx, jobID, services.ExportFormat(req.ExportFormat), req.FilterCriteria)
	if err != nil {
		SendErrorResponse(w, 500, "Failed to create export", err.Error())
		return
	}

	export, err := h.exportService.GetExportStatus(ctx, exportID)
	if err != nil {
		SendErrorResponse(w, 500, "Failed to retrieve export", err.Error())
		return
	}

	response := models.ExportStatusResponse{
		ID:             export.ID,
		JobID:          export.JobID,
		Status:         export.Status,
		ExportFormat:   export.ExportFormat,
		FileSize:       export.FileSize,
		RecordCount:    export.RecordCount,
		CreatedAt:      export.CreatedAt,
		IsDownloadable: export.Status == "completed",
	}

	sendJSON(w, http.StatusAccepted, response)
}

// GetExportStatus gets the status of an export (GET /api/v1/exports/:exportId)
func (h *ExportHandlers) GetExportStatus(w http.ResponseWriter, r *http.Request) {
	vars := chi.URLParam(r, "exportId")
	exportID, err := uuid.Parse(vars)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid export ID")
		return
	}

	tenantNorm := normalizeTenantID(jwtmiddleware.GetClaimsFromContext(r).TenantID)
	ctx := setupAuthContext(r.Context(), tenantNorm)

	export, err := h.exportService.GetExportStatus(ctx, exportID)
	if err != nil {
		if err.Error() == "export not found" {
			sendError(w, http.StatusNotFound, "Export not found")
		} else {
			SendErrorResponse(w, 500, "Failed to get export status", err.Error())
		}
		return
	}

	response := models.ExportStatusResponse{
		ID:                  export.ID,
		JobID:               export.JobID,
		Status:              export.Status,
		ExportFormat:        export.ExportFormat,
		FileSize:            export.FileSize,
		RecordCount:         export.RecordCount,
		CreatedAt:           export.CreatedAt,
		CompletedAt:         export.CompletedAt,
		ExpiresAt:           export.ExpiresAt,
		PresignedURL:        export.PresignedURL,
		PresignedURLExpires: export.PresignedURLExpires,
		IsDownloadable:      export.Status == "completed",
		ErrorMessage:        export.ErrorMessage,
	}

	sendJSON(w, http.StatusOK, response)
}

// ListExports lists all exports for a job (GET /api/v1/jobs/:jobId/exports)
func (h *ExportHandlers) ListExports(w http.ResponseWriter, r *http.Request) {
	vars := chi.URLParam(r, "jobId")
	jobID, err := uuid.Parse(vars)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid job ID")
		return
	}

	tenantNorm := normalizeTenantID(jwtmiddleware.GetClaimsFromContext(r).TenantID)
	ctx := setupAuthContext(r.Context(), tenantNorm)

	exports, err := h.exportService.ListExports(ctx, jobID)
	if err != nil {
		SendErrorResponse(w, 500, "Failed to list exports", err.Error())
		return
	}

	summaries := make([]*models.ExportSummary, 0, len(exports))
	for _, exp := range exports {
		summaries = append(summaries, &models.ExportSummary{
			ID:        exp.ID,
			Format:    exp.ExportFormat,
			Status:    exp.Status,
			FileSize:  exp.FileSize,
			Records:   exp.RecordCount,
			CreatedAt: exp.CreatedAt,
			ExpiresAt: exp.ExpiresAt,
		})
	}

	response := models.ListExportsResponse{
		Exports: summaries,
		Total:   len(summaries),
	}

	sendJSON(w, http.StatusOK, response)
}

// DownloadExport downloads an export file (GET /api/v1/exports/:exportId/download)
func (h *ExportHandlers) DownloadExport(w http.ResponseWriter, r *http.Request) {
	vars := chi.URLParam(r, "exportId")
	exportID, err := uuid.Parse(vars)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid export ID")
		return
	}

	tenantNorm := normalizeTenantID(jwtmiddleware.GetClaimsFromContext(r).TenantID)
	ctx := setupAuthContext(r.Context(), tenantNorm)

	file, contentType, err := h.exportService.DownloadExport(ctx, exportID)
	if err != nil {
		if err.Error() == "export not found" {
			sendError(w, http.StatusNotFound, "Export not found")
		} else {
			SendErrorResponse(w, 400, "Failed to download export", err.Error())
		}
		return
	}
	defer file.Close()

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"export-%s\"", exportID.String()[:8]))

	if _, err := io.Copy(w, file); err != nil {
		fmt.Fprintf(w, "Error reading file: %v", err)
	}
}

// GetDownloadURL generates a presigned download URL (POST /api/v1/exports/:exportId/download-url)
func (h *ExportHandlers) GetDownloadURL(w http.ResponseWriter, r *http.Request) {
	vars := chi.URLParam(r, "exportId")
	exportID, err := uuid.Parse(vars)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid export ID")
		return
	}

	tenantNorm := normalizeTenantID(jwtmiddleware.GetClaimsFromContext(r).TenantID)
	ctx := setupAuthContext(r.Context(), tenantNorm)

	var req models.DownloadURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request: %v", err))
		return
	}

	expiryHours := req.ExpiryHours
	if expiryHours == 0 {
		expiryHours = 24
	}

	url, err := h.exportService.GetDownloadURL(ctx, exportID, expiryHours)
	if err != nil {
		if err.Error() == "export not found" {
			sendError(w, http.StatusNotFound, "Export not found")
		} else {
			SendErrorResponse(w, 400, "Failed to generate download URL", err.Error())
		}
		return
	}

	response := map[string]interface{}{
		"url":    url,
		"format": "presigned",
	}

	sendJSON(w, http.StatusOK, response)
}
