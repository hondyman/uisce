package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/services"
)

// GoalHandler handles API requests for goals.
type GoalHandler struct {
	service *services.GoalService
}

// NewGoalHandler creates a new GoalHandler.
func NewGoalHandler(s *services.GoalService) *GoalHandler {
	return &GoalHandler{service: s}
}

// RegisterRoutes registers the routes for GoalHandler.
func (h *GoalHandler) RegisterRoutes(r chi.Router) {
	r.Get("/api/goals", h.HandleListGoals)
}

// HandleListGoals lists all goals for the current user.
func (h *GoalHandler) HandleListGoals(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id") // In a real app, get this from auth context.
	if userID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "user_id is required"})
		return
	}
	goals, err := h.service.ListGoals(r.Context(), userID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to list goals"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(goals)
}
