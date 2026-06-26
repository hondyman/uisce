package reports

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/api/common"
)

// ReportHandlers contains HTTP handlers for report endpoints
type ReportHandlers struct {
	orchestrator *ReportOrchestrator
}

// NewReportHandlers creates new report HTTP handlers
func NewReportHandlers(db *sql.DB) *ReportHandlers {
	return &ReportHandlers{
		orchestrator: NewReportOrchestrator(db),
	}
}

// RegisterRoutes registers all report routes
func (h *ReportHandlers) RegisterRoutes(r chi.Router) {
	r.Route("/api/reports", func(r chi.Router) {
		// Template management
		r.Get("/templates", h.ListTemplates)
		r.Get("/templates/{templateID}", h.GetTemplate)

		// Report generation
		r.Post("/generate", h.GenerateReport)
		r.Get("/executions/{executionID}", h.GetExecution)
		r.Get("/executions/{executionID}/download", h.DownloadReport)
	})
}

// ListTemplates lists available report templates
func (h *ReportHandlers) ListTemplates(w http.ResponseWriter, r *http.Request) {
	tenantID, _, err := common.GetTenantScope(r)
	if err != nil {
		common.HandleBadRequest(w, err)
		return
	}

	tid, err := uuid.Parse(tenantID)
	if err != nil {
		common.HandleBadRequest(w, err)
		return
	}

	category := r.URL.Query().Get("category")

	templates, err := h.orchestrator.ListTemplates(r.Context(), tid, category)
	if err != nil {
		common.HandleInternalError(w, err)
		return
	}

	common.WriteSuccess(w, templates, nil)
}

// GetTemplate retrieves a specific report template
func (h *ReportHandlers) GetTemplate(w http.ResponseWriter, r *http.Request) {
	tenantID, _, err := common.GetTenantScope(r)
	if err != nil {
		common.HandleBadRequest(w, err)
		return
	}

	tid, err := uuid.Parse(tenantID)
	if err != nil {
		common.HandleBadRequest(w, err)
		return
	}

	templateIDStr := chi.URLParam(r, "templateID")
	templateID, err := uuid.Parse(templateIDStr)
	if err != nil {
		common.HandleBadRequest(w, err)
		return
	}

	template, err := h.orchestrator.GetTemplate(r.Context(), templateID, tid)
	if err != nil {
		common.HandleNotFound(w, err)
		return
	}

	common.WriteSuccess(w, template, nil)
}

// GenerateReport triggers report generation
func (h *ReportHandlers) GenerateReport(w http.ResponseWriter, r *http.Request) {
	tenantID, _, err := common.GetTenantScope(r)
	if err != nil {
		common.HandleBadRequest(w, err)
		return
	}

	tid, err := uuid.Parse(tenantID)
	if err != nil {
		common.HandleBadRequest(w, err)
		return
	}

	var req struct {
		TemplateID string                 `json:"template_id"`
		Parameters map[string]interface{} `json:"parameters"`
	}

	if err := common.ParseJSONBody(r, &req); err != nil {
		common.HandleBadRequest(w, err)
		return
	}

	templateID, err := uuid.Parse(req.TemplateID)
	if err != nil {
		common.HandleBadRequest(w, err)
		return
	}

	// Generate report (sync for now, will be async with Temporal later)
	execution, err := h.orchestrator.GenerateReport(r.Context(), templateID, tid, req.Parameters)
	if err != nil {
		common.HandleInternalError(w, err)
		return
	}

	common.WriteCreated(w, execution)
}

// GetExecution retrieves a report execution status
func (h *ReportHandlers) GetExecution(w http.ResponseWriter, r *http.Request) {
	tenantID, _, err := common.GetTenantScope(r)
	if err != nil {
		common.HandleBadRequest(w, err)
		return
	}

	tid, err := uuid.Parse(tenantID)
	if err != nil {
		common.HandleBadRequest(w, err)
		return
	}

	executionIDStr := chi.URLParam(r, "executionID")
	executionID, err := uuid.Parse(executionIDStr)
	if err != nil {
		common.HandleBadRequest(w, err)
		return
	}

	execution, err := h.orchestrator.GetExecution(r.Context(), executionID, tid)
	if err != nil {
		common.HandleNotFound(w, err)
		return
	}

	common.WriteSuccess(w, execution, nil)
}

// DownloadReport returns the generated report PDF
func (h *ReportHandlers) DownloadReport(w http.ResponseWriter, r *http.Request) {
	tenantID, _, err := common.GetTenantScope(r)
	if err != nil {
		common.HandleBadRequest(w, err)
		return
	}

	tid, err := uuid.Parse(tenantID)
	if err != nil {
		common.HandleBadRequest(w, err)
		return
	}

	executionIDStr := chi.URLParam(r, "executionID")
	executionID, err := uuid.Parse(executionIDStr)
	if err != nil {
		common.HandleBadRequest(w, err)
		return
	}

	execution, err := h.orchestrator.GetExecution(r.Context(), executionID, tid)
	if err != nil {
		common.HandleNotFound(w, err)
		return
	}

	if execution.Status != "completed" {
		common.HandleError(w, fmt.Errorf("report not ready (status: %s)", execution.Status), http.StatusConflict)
		return
	}

	if execution.OutputURL == "" {
		common.HandleError(w, fmt.Errorf("report URL not available"), http.StatusNotFound)
		return
	}

	// For MVP, redirect to the output URL
	// In production, this would:
	// 1. Generate signed S3/GCS URL
	// 2. Or stream file directly
	http.Redirect(w, r, execution.OutputURL, http.StatusFound)
}
