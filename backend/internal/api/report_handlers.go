package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/reports"
	"github.com/hondyman/uisce/backend/internal/reporting"
	"github.com/hondyman/uisce/libs/jwt-middleware"
	"github.com/jmoiron/sqlx"
)

type ReportHandler struct {
	service       *reports.ReportService
	reportingRepo *reporting.Repository
}

func NewReportHandler(service *reports.ReportService, db *sqlx.DB) *ReportHandler {
	return &ReportHandler{
		service:       service,
		reportingRepo: reporting.NewRepository(db),
	}
}

func (h *ReportHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/reports", func(r chi.Router) {
		r.Get("/", h.ListTemplates)
		r.Post("/", h.CreateTemplate)
		r.Get("/by-key/{key}", h.GetTemplateByKey)
		r.Get("/{id}", h.GetTemplate)
		r.Put("/{id}", h.UpdateTemplate)
		r.Delete("/{id}", h.DeleteTemplate)
	})
}

func (h *ReportHandler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := h.service.ListTemplates(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(templates)
}

func (h *ReportHandler) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	var template reports.ReportTemplate
	if err := json.NewDecoder(r.Body).Decode(&template); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.service.CreateTemplate(r.Context(), &template); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(template)
}

func (h *ReportHandler) GetTemplate(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid UUID", http.StatusBadRequest)
		return
	}

	template, err := h.service.GetTemplate(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(template)
}

func (h *ReportHandler) UpdateTemplate(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid UUID", http.StatusBadRequest)
		return
	}

	var template reports.ReportTemplate
	if err := json.NewDecoder(r.Body).Decode(&template); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	template.ID = id

	if err := h.service.UpdateTemplate(r.Context(), &template); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(template)
}

func (h *ReportHandler) DeleteTemplate(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid UUID", http.StatusBadRequest)
		return
	}

	if err := h.service.DeleteTemplate(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *ReportHandler) GetTemplateByKey(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	if key == "" {
		http.Error(w, "key is required", http.StatusBadRequest)
		return
	}

	tenantID, datasourceID := getReportingTenantContext(r)

	def, err := h.reportingRepo.GetDefinitionByKey(r.Context(), tenantID, datasourceID, key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if def == nil {
		http.Error(w, "report not found", http.StatusNotFound)
		return
	}

	template, err := h.service.GetTemplate(r.Context(), def.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if template == nil {
		http.Error(w, "report template not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(template)
}

func getReportingTenantContext(r *http.Request) (uuid.UUID, uuid.UUID) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	var tenantID uuid.UUID
	if claims != nil && claims.TenantID != "" {
		tenantID, _ = uuid.Parse(claims.TenantID)
	}
	if tenantID == uuid.Nil {
		tenantIDStr := r.Header.Get("X-Tenant-ID")
		tenantID, _ = uuid.Parse(tenantIDStr)
	}
	datasourceIDStr := r.Header.Get("X-Tenant-Datasource-ID")
	datasourceID, _ := uuid.Parse(datasourceIDStr)
	return tenantID, datasourceID
}
