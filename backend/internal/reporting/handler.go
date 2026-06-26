package reporting

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// Handler handles HTTP requests for reporting
type Handler struct {
	service *Service
}

// NewHandler creates a new reporting handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers the reporting routes
func (h *Handler) RegisterRoutes(r chi.Router) {
	// Report Definitions
	r.Route("/reports/definitions", func(r chi.Router) {
		r.Get("/", h.ListDefinitions)
		r.Post("/", h.CreateDefinition)
		r.Get("/{id}", h.GetDefinition)
		r.Put("/{id}", h.UpdateDefinition)
		r.Delete("/{id}", h.DeleteDefinition)
		r.Post("/{id}/publish", h.PublishDefinition)
	})

	// Report Extensions
	r.Route("/reports/extensions", func(r chi.Router) {
		r.Get("/", h.ListExtensions)
		r.Post("/", h.CreateExtension)
		r.Get("/{id}", h.GetExtension)
		r.Put("/{id}", h.UpdateExtension)
		r.Delete("/{id}", h.DeleteExtension)
	})

	// Report Rendering
	r.Post("/reports/render", h.RenderReport)
	r.Post("/reports/render/async", h.RenderReportAsync)

	// Report Instances
	r.Route("/reports/instances", func(r chi.Router) {
		r.Get("/", h.ListInstances)
		r.Get("/{id}", h.GetInstance)
		r.Get("/{id}/download", h.DownloadInstance)
	})

	// Report Schedules
	r.Route("/reports/schedules", func(r chi.Router) {
		r.Get("/", h.ListSchedules)
		r.Post("/", h.CreateSchedule)
		r.Get("/{id}", h.GetSchedule)
		r.Put("/{id}", h.UpdateSchedule)
		r.Delete("/{id}", h.DeleteSchedule)
	})

	// Provisioning
	r.Post("/reports/provision", h.ProvisionReports)
	r.Get("/reports/packages", h.ListPackages)
}

// ============================================================================
// REPORT DEFINITIONS
// ============================================================================

func (h *Handler) ListDefinitions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, datasourceID := getTenantContext(r)

	filters := map[string]interface{}{
		"category": r.URL.Query().Get("category"),
		"status":   r.URL.Query().Get("status"),
	}

	if isCore := r.URL.Query().Get("is_core"); isCore != "" {
		filters["is_core"] = isCore == "true"
	}

	defs, err := h.service.ListDefinitions(ctx, tenantID, datasourceID, filters)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, defs)
}

func (h *Handler) CreateDefinition(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, datasourceID := getTenantContext(r)
	userID := getUserID(r)

	var req CreateReportDefinitionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	def, err := h.service.CreateDefinition(ctx, tenantID, datasourceID, req, userID)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			respondError(w, http.StatusConflict, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, def)
}

func (h *Handler) GetDefinition(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	def, err := h.service.GetDefinition(ctx, id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if def == nil {
		respondError(w, http.StatusNotFound, "definition not found")
		return
	}

	respondJSON(w, http.StatusOK, def)
}

func (h *Handler) UpdateDefinition(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")
	userID := getUserID(r)

	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	// Get existing
	existing, err := h.service.GetDefinition(ctx, id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if existing == nil {
		respondError(w, http.StatusNotFound, "definition not found")
		return
	}

	// Decode updates
	var updates struct {
		DisplayName string        `json:"display_name"`
		Description string        `json:"description"`
		Category    string        `json:"category"`
		Tags        []string      `json:"tags"`
		Definition  *ReportLayout `json:"definition"`
		Status      string        `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Apply updates
	if updates.DisplayName != "" {
		existing.DisplayName = updates.DisplayName
	}
	if updates.Description != "" {
		existing.Description = updates.Description
	}
	if updates.Category != "" {
		existing.Category = updates.Category
	}
	if updates.Tags != nil {
		existing.Tags = updates.Tags
	}
	if updates.Definition != nil {
		existing.Definition = updates.Definition
	}
	if updates.Status != "" {
		existing.Status = updates.Status
	}

	if err := h.service.UpdateDefinition(ctx, existing, userID); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, existing)
}

func (h *Handler) DeleteDefinition(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.service.DeleteDefinition(ctx, id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) PublishDefinition(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")
	userID := getUserID(r)

	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	if userID == nil {
		respondError(w, http.StatusUnauthorized, "user id required")
		return
	}

	if err := h.service.PublishDefinition(ctx, id, *userID); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "published"})
}

// ============================================================================
// REPORT EXTENSIONS
// ============================================================================

func (h *Handler) ListExtensions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, datasourceID := getTenantContext(r)

	baseReportIDStr := r.URL.Query().Get("base_report_id")

	var exts []ReportExtension
	var err error

	if baseReportIDStr != "" {
		baseReportID, parseErr := uuid.Parse(baseReportIDStr)
		if parseErr != nil {
			respondError(w, http.StatusBadRequest, "invalid base_report_id")
			return
		}
		exts, err = h.service.ListExtensions(ctx, tenantID, datasourceID, baseReportID)
	} else {
		exts, err = h.service.ListAllExtensions(ctx, tenantID, datasourceID)
	}

	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, exts)
}

func (h *Handler) CreateExtension(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, datasourceID := getTenantContext(r)
	userID := getUserID(r)

	var req CreateReportExtensionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ext, err := h.service.CreateExtension(ctx, tenantID, datasourceID, req, userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, ext)
}

func (h *Handler) GetExtension(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	ext, err := h.service.GetExtension(ctx, id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if ext == nil {
		respondError(w, http.StatusNotFound, "extension not found")
		return
	}

	respondJSON(w, http.StatusOK, ext)
}

func (h *Handler) UpdateExtension(w http.ResponseWriter, r *http.Request) {
	// Similar to UpdateDefinition
	respondError(w, http.StatusNotImplemented, "not implemented")
}

func (h *Handler) DeleteExtension(w http.ResponseWriter, r *http.Request) {
	// Similar to DeleteDefinition
	respondError(w, http.StatusNotImplemented, "not implemented")
}

// ============================================================================
// REPORT RENDERING
// ============================================================================

func (h *Handler) RenderReport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, datasourceID := getTenantContext(r)
	userID := getUserID(r)

	var req RenderReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	inst, err := h.service.RenderReport(ctx, tenantID, datasourceID, req, userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, inst)
}

func (h *Handler) RenderReportAsync(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, datasourceID := getTenantContext(r)
	userID := getUserID(r)

	var req RenderReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	inst, err := h.service.RenderReportAsync(ctx, tenantID, datasourceID, req, userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusAccepted, inst)
}

// ============================================================================
// REPORT INSTANCES
// ============================================================================

func (h *Handler) ListInstances(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, datasourceID := getTenantContext(r)

	limit := 50 // Default
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}

	instances, err := h.service.ListInstances(ctx, tenantID, datasourceID, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, instances)
}

func (h *Handler) GetInstance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	inst, err := h.service.GetInstance(ctx, id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if inst == nil {
		respondError(w, http.StatusNotFound, "instance not found")
		return
	}

	respondJSON(w, http.StatusOK, inst)
}

func (h *Handler) DownloadInstance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	inst, err := h.service.GetInstance(ctx, id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if inst == nil {
		respondError(w, http.StatusNotFound, "instance not found")
		return
	}

	if inst.Status != "completed" {
		respondError(w, http.StatusBadRequest, "report not ready")
		return
	}

	// If we have a URL, redirect
	if inst.OutputURL != "" {
		http.Redirect(w, r, inst.OutputURL, http.StatusFound)
		return
	}

	// If we have data, serve it
	if len(inst.OutputData) > 0 {
		contentType := "application/octet-stream"
		switch inst.OutputFormat {
		case "pdf":
			contentType = "application/pdf"
		case "html":
			contentType = "text/html"
		case "excel":
			contentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
		}

		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=report.%s", inst.OutputFormat))
		w.Write(inst.OutputData)
		return
	}

	respondError(w, http.StatusNotFound, "report data not available")
}

// ============================================================================
// REPORT SCHEDULES
// ============================================================================

func (h *Handler) ListSchedules(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, datasourceID := getTenantContext(r)

	schedules, err := h.service.ListSchedules(ctx, tenantID, datasourceID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, schedules)
}

func (h *Handler) CreateSchedule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, datasourceID := getTenantContext(r)
	userID := getUserID(r)

	var sched ReportSchedule
	if err := json.NewDecoder(r.Body).Decode(&sched); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	sched.TenantID = tenantID
	sched.TenantDatasourceID = datasourceID
	sched.CreatedBy = userID

	if err := h.service.CreateSchedule(ctx, &sched); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, sched)
}

func (h *Handler) GetSchedule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	sched, err := h.service.GetSchedule(ctx, id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if sched == nil {
		respondError(w, http.StatusNotFound, "schedule not found")
		return
	}

	respondJSON(w, http.StatusOK, sched)
}

func (h *Handler) UpdateSchedule(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "not implemented")
}

func (h *Handler) DeleteSchedule(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "not implemented")
}

// ============================================================================
// PROVISIONING
// ============================================================================

func (h *Handler) ProvisionReports(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req ProvisionReportsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.service.ProvisionReports(ctx, req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, resp)
}

func (h *Handler) ListPackages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	packages, err := h.service.ListPackages(ctx)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, packages)
}

// ============================================================================
// HELPERS
// ============================================================================

func getTenantContext(r *http.Request) (uuid.UUID, uuid.UUID) {
	// Get from query params or headers (following your agents.md pattern)
	tenantIDStr := r.URL.Query().Get("tenant_id")
	if tenantIDStr == "" {
		tenantIDStr = jwtmiddleware.GetClaimsFromContext(r).TenantID
	}

	datasourceIDStr := r.URL.Query().Get("datasource_id")
	if datasourceIDStr == "" {
		datasourceIDStr = r.Header.Get("X-Tenant-Datasource-ID")
	}

	tenantID, _ := uuid.Parse(tenantIDStr)
	datasourceID, _ := uuid.Parse(datasourceIDStr)

	return tenantID, datasourceID
}

func getUserID(r *http.Request) *uuid.UUID {
	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr == "" {
		return nil
	}

	id, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil
	}

	return &id
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
