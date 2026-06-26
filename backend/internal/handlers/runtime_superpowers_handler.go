package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/runtime_superpowers/adaptive"
	"github.com/hondyman/semlayer/backend/internal/runtime_superpowers/offline"
	"github.com/hondyman/semlayer/backend/internal/runtime_superpowers/prefetch"
)

type RuntimeSuperpowersHandler struct {
	optimizer   *adaptive.Optimizer
	predictor   *prefetch.Predictor
	syncManager *offline.SyncManager
}

func NewRuntimeSuperpowersHandler(
	opt *adaptive.Optimizer,
	pred *prefetch.Predictor,
	sync *offline.SyncManager,
) *RuntimeSuperpowersHandler {
	return &RuntimeSuperpowersHandler{
		optimizer:   opt,
		predictor:   pred,
		syncManager: sync,
	}
}

func (h *RuntimeSuperpowersHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Adaptive
	r.Post("/adaptive/optimize", h.OptimizeRendering)

	// Prefetch
	r.Post("/prefetch/predict", h.PredictNext)
	r.Post("/prefetch/record", h.RecordTransition)

	// Offline
	r.Post("/offline/sync", h.SyncMutations)

	return r
}

func (h *RuntimeSuperpowersHandler) OptimizeRendering(w http.ResponseWriter, r *http.Request) {
	var profile adaptive.DeviceProfile
	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	result, _ := h.optimizer.Optimize(r.Context(), profile)
	json.NewEncoder(w).Encode(result)
}

func (h *RuntimeSuperpowersHandler) PredictNext(w http.ResponseWriter, r *http.Request) {
	// Assume current page ID is passed in body for now
	var body struct {
		CurrentPageID string `json:"current_page_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	predictions, _ := h.predictor.PredictNext(r.Context(), body.CurrentPageID)
	json.NewEncoder(w).Encode(predictions)
}

func (h *RuntimeSuperpowersHandler) RecordTransition(w http.ResponseWriter, r *http.Request) {
	var body prefetch.TransitionEvidence
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.predictor.RecordTransition(r.Context(), body.FromPageID, body.ToPageID)
	w.WriteHeader(http.StatusOK)
}

func (h *RuntimeSuperpowersHandler) SyncMutations(w http.ResponseWriter, r *http.Request) {
	var mutations []offline.Mutation
	if err := json.NewDecoder(r.Body).Decode(&mutations); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	results, _ := h.syncManager.SyncMutations(r.Context(), mutations)
	json.NewEncoder(w).Encode(results)
}
