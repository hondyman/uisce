package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/help"
)

// HelpHandler serves API documentation.
type HelpHandler struct {
	registry *help.Registry
}

// NewHelpHandler creates a new HelpHandler.
func NewHelpHandler(r *help.Registry) *HelpHandler {
	return &HelpHandler{registry: r}
}

// RegisterRoutes registers the routes for HelpHandler.
func (h *HelpHandler) RegisterRoutes(r chi.Router) {
	r.Get("/api/help", h.HandleGetHelp)
}

// HandleGetHelp returns the registered help documentation as JSON.
func (h *HelpHandler) HandleGetHelp(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.registry.GetHelp())
}
