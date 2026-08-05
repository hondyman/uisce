package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/uisce/backend/internal/services"
	"github.com/jmoiron/sqlx"
)

type ExecutionMonitorHandler struct {
	service *services.ExecutionMonitorService
}

func NewExecutionMonitorHandler(db *sqlx.DB) *ExecutionMonitorHandler {
	return &ExecutionMonitorHandler{
		service: services.NewExecutionMonitorService(db),
	}
}

func (h *ExecutionMonitorHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.ListLogs)
	return r
}

func (h *ExecutionMonitorHandler) ListLogs(w http.ResponseWriter, r *http.Request) {
	limit, offset := Paginate(r)

	logs, err := h.service.QueryLogs(r.Context(), limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	SendJSON(w, http.StatusOK, logs)
}
