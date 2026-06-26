package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/analytics"
)

// BOGraphHandler handles Business Object graph visualization requests
type BOGraphHandler struct {
	Service *analytics.BOGraphService
}

// NewBOGraphHandler creates a new graph handler
func NewBOGraphHandler(service *analytics.BOGraphService) *BOGraphHandler {
	return &BOGraphHandler{Service: service}
}

// RegisterRoutes registers the graph routes
func (h *BOGraphHandler) RegisterRoutes(r chi.Router) {
	r.Get("/api/bo/{boId}/graph", h.GetBOGraph)
}

// GetBOGraph generates and returns the lineage graph for a Business Object
func (h *BOGraphHandler) GetBOGraph(w http.ResponseWriter, r *http.Request) {
	boID := chi.URLParam(r, "boId")
	if boID == "" {
		http.Error(w, "boId is required", http.StatusBadRequest)
		return
	}

	graph, err := h.Service.GenerateGraph(boID)
	if err != nil {
		http.Error(w, "Failed to generate graph: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(graph)
}
