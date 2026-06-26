package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/services"
)

// AlertsHandler handles API requests for alerts.
type AlertsHandler struct {
	service *services.AlertsService
}

// NewAlertsHandler creates a new AlertsHandler.
func NewAlertsHandler(service *services.AlertsService) *AlertsHandler {
	return &AlertsHandler{service: service}
}

// RegisterRoutes registers the routes for AlertsHandler.
func (h *AlertsHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/alerts", func(r chi.Router) {
		r.Get("/", h.HandleList)
		r.Post("/{id}/read", h.HandleMarkRead)
	})
}

// HandleList retrieves alerts for a user.
func (h *AlertsHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "user_id is required"})
		return
	}
	alerts, err := h.service.List(r.Context(), userID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to list alerts"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alerts)
}

// HandleMarkRead marks an alert as read.
func (h *AlertsHandler) HandleMarkRead(w http.ResponseWriter, r *http.Request) {
	alertID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid alert ID"})
		return
	}
	// In a real app, userID would come from auth context.
	userID := "current_user"
	err = h.service.MarkRead(r.Context(), alertID, userID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to mark alert as read"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
}
