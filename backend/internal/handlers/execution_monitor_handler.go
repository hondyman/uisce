package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/services"
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
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 50
	if l, err := strconv.Atoi(limitStr); err == nil {
		limit = l
	}

	offset := 0
	if o, err := strconv.Atoi(offsetStr); err == nil {
		offset = o
	}

	logs, err := h.service.QueryLogs(r.Context(), limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}
