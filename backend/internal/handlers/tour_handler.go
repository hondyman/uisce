package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/services"
)

// TourHandler handles API requests for interactive tours.
type TourHandler struct {
	service *services.TourService
}

// NewTourHandler creates a new TourHandler.
func NewTourHandler(s *services.TourService) *TourHandler {
	return &TourHandler{service: s}
}

// RegisterRoutes registers the tour routes.
func (h *TourHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/tours", func(r chi.Router) {
		r.Get("/", h.HandleListTours)
		r.Get("/{id}", h.HandleGetTour)
	})
}

// HandleListTours lists available tours.
func (h *TourHandler) HandleListTours(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id") // In a real app, get this from auth context.
	tours, err := h.service.ListTours(r.Context(), userID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to list tours"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tours)
}

// HandleGetTour retrieves the steps for a specific tour.
func (h *TourHandler) HandleGetTour(w http.ResponseWriter, r *http.Request) {
	tourID := chi.URLParam(r, "id")
	tour, err := h.service.GetTour(r.Context(), tourID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Tour not found"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tour)
}
