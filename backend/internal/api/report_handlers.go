package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/reports"
)

type ReportHandler struct {
	service *reports.ReportService
}

func NewReportHandler(service *reports.ReportService) *ReportHandler {
	return &ReportHandler{service: service}
}

func (h *ReportHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/reports", func(r chi.Router) {
		r.Get("/", h.ListTemplates)
		r.Post("/", h.CreateTemplate)
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
