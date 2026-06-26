package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/services"
)

// SnapshotHandler handles API requests for dashboard snapshots.
type SnapshotHandler struct {
	service *services.SnapshotService
}

// NewSnapshotHandler creates a new SnapshotHandler.
func NewSnapshotHandler(service *services.SnapshotService) *SnapshotHandler {
	return &SnapshotHandler{service: service}
}

// RegisterRoutes mounts snapshot routes
func (h *SnapshotHandler) RegisterRoutes(r chi.Router) {
	r.Route("/dashboards/{id}/snapshots", func(r chi.Router) {
		r.Get("/", h.HandleListSnapshots)
		r.Post("/", h.HandleCreateSnapshot)
	})
	r.Route("/snapshots/{id}", func(r chi.Router) {
		r.Get("/compare", h.HandleCompareSnapshots)
		r.Post("/restore", h.HandleRestoreSnapshot)
	})
}

// HandleListSnapshots retrieves snapshots for a dashboard.
func (h *SnapshotHandler) HandleListSnapshots(w http.ResponseWriter, r *http.Request) {
	dashboardID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid dashboard ID", http.StatusBadRequest)
		return
	}
	snapshots, err := h.service.ListSnapshots(r.Context(), dashboardID)
	if err != nil {
		http.Error(w, "Failed to list snapshots", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(snapshots)
}

// HandleCreateSnapshot creates a new snapshot.
func (h *SnapshotHandler) HandleCreateSnapshot(w http.ResponseWriter, r *http.Request) {
	dashboardID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid dashboard ID", http.StatusBadRequest)
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		http.Error(w, "Invalid request payload, 'name' is required", http.StatusBadRequest)
		return
	}
	// In a real app, createdBy would come from auth context.
	createdBy := "current_user"
	snapshot, err := h.service.CreateSnapshot(r.Context(), dashboardID, req.Name, createdBy)
	if err != nil {
		http.Error(w, "Failed to create snapshot", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(snapshot)
}

// HandleCompareSnapshots retrieves a diff between two snapshots.
func (h *SnapshotHandler) HandleCompareSnapshots(w http.ResponseWriter, r *http.Request) {
	snapshotID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid snapshot ID", http.StatusBadRequest)
		return
	}
	compareToIDStr := r.URL.Query().Get("compare_to")
	compareToID, err := uuid.Parse(compareToIDStr)
	if err != nil {
		http.Error(w, "Invalid compare_to ID", http.StatusBadRequest)
		return
	}
	diff, err := h.service.CompareSnapshots(r.Context(), snapshotID, compareToID)
	if err != nil {
		http.Error(w, "Failed to compare snapshots", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(diff)
}

// HandleRestoreSnapshot restores a dashboard to a previous state.
func (h *SnapshotHandler) HandleRestoreSnapshot(w http.ResponseWriter, r *http.Request) {
	// Mock implementation
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "restore initiated"})
}
