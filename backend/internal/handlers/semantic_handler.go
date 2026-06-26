package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/semantic"
)

// SemanticHandler handles generic semantic object operations
type SemanticHandler struct {
	versionStore *semantic.SemanticVersionStore
	// In future: semantic graph interaction
}

// NewSemanticHandler creates a new handler
func NewSemanticHandler(vs *semantic.SemanticVersionStore) *SemanticHandler {
	return &SemanticHandler{versionStore: vs}
}

// RegisterRoutes registers endpoints
func (h *SemanticHandler) RegisterRoutes(r chi.Router) {
	// Use explicit paths to avoid conflict with SemanticLayerHandler which mounts /semantic
	r.Get("/semantic/history/{id}", h.GetHistory)
	r.Get("/semantic/version/{id}/{version}", h.GetVersion)
}

// GetHistory retrieves the version history for an object
func (h *SemanticHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	history, err := h.versionStore.GetHistory(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

// GetVersion retrieves a specific version of an object
func (h *SemanticHandler) GetVersion(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	versionStr := chi.URLParam(r, "version")
	version, err := strconv.Atoi(versionStr)
	if err != nil {
		http.Error(w, "invalid version", http.StatusBadRequest)
		return
	}

	obj, err := h.versionStore.GetVersion(r.Context(), id, version)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(obj)
}
