package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/services"
	"github.com/hondyman/semlayer/backend/models"
)

// FolderHandler handles API requests for folders.
type FolderHandler struct {
	service *services.FolderService
}

// NewFolderHandler creates a new FolderHandler.
func NewFolderHandler(s *services.FolderService) *FolderHandler {
	return &FolderHandler{service: s}
}

// RegisterRoutes registers the folder routes.
func (h *FolderHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/folders", func(r chi.Router) {
		r.Get("/", h.HandleListFolders)
		r.Post("/{id}/items", h.HandleAddItemToFolder)
		r.Get("/{id}/analytics", h.HandleGetFolderAnalytics)
		r.Get("/{id}/diff", h.HandleGetFolderDiff)
	})
}

// HandleListFolders lists all folders for the current user.
func (h *FolderHandler) HandleListFolders(w http.ResponseWriter, r *http.Request) {
	// Use placeholder user for now
	folders, err := h.service.ListFolders(r.Context(), placeholderUserID)
	if err != nil {
		http.Error(w, "Failed to list folders", http.StatusInternalServerError)
		return
	}
	if folders == nil {
		folders = []models.FullFolder{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(folders)
}

// HandleAddItemToFolder adds an item to a folder.
func (h *FolderHandler) HandleAddItemToFolder(w http.ResponseWriter, r *http.Request) {
	folderID := chi.URLParam(r, "id")
	var req models.AddItemToFolderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if err := h.service.AddItemToFolder(r.Context(), folderID, req.ItemID, req.ItemType, placeholderUserID); err != nil {
		http.Error(w, "Failed to add item to folder", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// HandleGetFolderAnalytics retrieves analytics for a folder.
func (h *FolderHandler) HandleGetFolderAnalytics(w http.ResponseWriter, r *http.Request) {
	folderID := chi.URLParam(r, "id")
	analytics, err := h.service.GetFolderAnalytics(r.Context(), folderID)
	if err != nil {
		http.Error(w, "Failed to get folder analytics", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analytics)
}

// HandleGetFolderDiff retrieves a diff of folder contents.
func (h *FolderHandler) HandleGetFolderDiff(w http.ResponseWriter, r *http.Request) {
	folderID := chi.URLParam(r, "id")
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	from, err := time.Parse(time.RFC3339, fromStr)
	if err != nil {
		http.Error(w, "Invalid 'from' date format", http.StatusBadRequest)
		return
	}
	to, err := time.Parse(time.RFC3339, toStr)
	if err != nil {
		http.Error(w, "Invalid 'to' date format", http.StatusBadRequest)
		return
	}

	diff, err := h.service.GetFolderDiff(r.Context(), folderID, from, to)
	if err != nil {
		http.Error(w, "Failed to generate folder diff", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(diff)
}
