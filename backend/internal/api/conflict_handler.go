package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/repository"
	"github.com/hondyman/semlayer/backend/internal/sync"
	"github.com/sirupsen/logrus"
)

type ConflictHandler struct {
	syncRepo *repository.GoogleSyncRepo
	logger   *logrus.Entry
}

func NewConflictHandler(syncRepo *repository.GoogleSyncRepo, logger *logrus.Entry) *ConflictHandler {
	return &ConflictHandler{
		syncRepo: syncRepo,
		logger:   logger.WithField("component", "conflict_handler"),
	}
}

func (h *ConflictHandler) RegisterRoutes(r chi.Router) {
	r.Route("/sync/conflicts", func(r chi.Router) {
		r.Get("/", h.ListConflicts)
		r.Post("/{conflictID}/resolve", h.ResolveConflict)
	})
}

func (h *ConflictHandler) ListConflicts(w http.ResponseWriter, r *http.Request) {
	// Implementation would query sync_conflicts table via repo
	// For now, return mocking or empty
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"conflicts": []interface{}{},
	})
}

func (h *ConflictHandler) ResolveConflict(w http.ResponseWriter, r *http.Request) {
	conflictID := chi.URLParam(r, "conflictID")
	var req struct {
		ResolutionStrategy string `json:"resolution_strategy"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	h.logger.Infof("Resolving conflict %s with strategy %s", conflictID, req.ResolutionStrategy)

	// Call ConflictDetector
	// We need to construct it.
	// Warning: We need logger.
	cd := sync.NewConflictDetector(sync.ConflictDetectorConfig{
		SyncRepo: h.syncRepo,
		Logger:   h.logger,
	})

	if err := cd.ResolveConflict(r.Context(), conflictID, sync.ResolutionStrategy(req.ResolutionStrategy)); err != nil {
		h.logger.WithError(err).Error("Failed to resolve conflict")
		http.Error(w, "Failed to resolve conflict", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "resolved"})
}
