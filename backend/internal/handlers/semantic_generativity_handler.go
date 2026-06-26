package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/mdm"
)

// SemanticGenerativityHandler exposes generative capabilities via HTTP.
type SemanticGenerativityHandler struct {
	apiEngine     *mdm.SemanticAPIEngine
	viewGenerator *mdm.SemanticViewGenerator
}

func NewSemanticGenerativityHandler(ae *mdm.SemanticAPIEngine, vg *mdm.SemanticViewGenerator) *SemanticGenerativityHandler {
	return &SemanticGenerativityHandler{
		apiEngine:     ae,
		viewGenerator: vg,
	}
}

func (h *SemanticGenerativityHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/semantic/generate", func(r chi.Router) {
		r.Get("/bos", h.ListBOs)
		r.Post("/api/{boID}", h.GenerateAPI)
		r.Post("/view/{boID}", h.GenerateView)
	})
}

func (h *SemanticGenerativityHandler) ListBOs(w http.ResponseWriter, r *http.Request) {
	bundles, err := h.apiEngine.ListAllBundles(r.Context(), uuid.Nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bundles)
}

func (h *SemanticGenerativityHandler) GenerateAPI(w http.ResponseWriter, r *http.Request) {
	boIDStr := chi.URLParam(r, "boID")
	boID, err := uuid.Parse(boIDStr)
	if err != nil {
		http.Error(w, "invalid boID", http.StatusBadRequest)
		return
	}

	bundle, err := h.apiEngine.GenerateBundleForBO(r.Context(), uuid.Nil, boID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bundle)
}

func (h *SemanticGenerativityHandler) GenerateView(w http.ResponseWriter, r *http.Request) {
	boIDStr := chi.URLParam(r, "boID")
	boID, err := uuid.Parse(boIDStr)
	if err != nil {
		http.Error(w, "invalid boID", http.StatusBadRequest)
		return
	}

	sql, err := h.viewGenerator.GenerateViewSQL(r.Context(), uuid.Nil, boID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := map[string]string{
		"sql": sql,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
