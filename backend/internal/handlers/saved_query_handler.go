package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/analytics"
	"github.com/hondyman/semlayer/backend/models"
)

// SavedQueryHandler handles API requests for saved queries.
type SavedQueryHandler struct {
	service      *analytics.QueryService
	securityDeps SecurityContextDeps
}

// NewSavedQueryHandler creates a new SavedQueryHandler.
func NewSavedQueryHandler(s *analytics.QueryService, deps SecurityContextDeps) *SavedQueryHandler {
	return &SavedQueryHandler{service: s, securityDeps: deps}
}

// For demonstration, we'll use a hardcoded user ID. In a real app, this would come from middleware.
const placeholderUserID = "user-123"
const placeholderTenantID = "tenant-abc"

// HandleListSavedQueries lists all saved queries for the current user.
func (h *SavedQueryHandler) HandleListSavedQueries(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	scope := query.Get("scope")
	if scope == "" {
		scope = "mine"
	}
	viewName := query.Get("view")
	search := query.Get("search")
	tags := query["tags"]

	// In a real app, this would come from auth middleware
	secCtx, ctx, err := SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	queries, err := h.service.ListSavedQueries(ctx, *secCtx, scope, viewName, search, tags)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(queries)
}

// HandleCreateSavedQuery creates a new saved query.
func (h *SavedQueryHandler) HandleCreateSavedQuery(w http.ResponseWriter, r *http.Request) {
	var req models.SavedQueryCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid request body: " + err.Error()})
		return
	}

	savedQuery, err := h.service.CreateSavedQuery(r.Context(), req, placeholderUserID, placeholderTenantID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(savedQuery)
}

// HandleUpdateSavedQuery updates an existing saved query.
func (h *SavedQueryHandler) HandleUpdateSavedQuery(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req models.SavedQueryUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid request body: " + err.Error()})
		return
	}

	err := h.service.UpdateSavedQuery(r.Context(), id, req, placeholderUserID)
	if err != nil {
		// This could be a 404 if not found, but 500 is a safe default for other errors.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// HandleDeleteSavedQuery deletes a saved query.
func (h *SavedQueryHandler) HandleDeleteSavedQuery(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	err := h.service.DeleteSavedQuery(r.Context(), id, placeholderUserID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// HandleCloneSavedQuery clones an existing saved query.
func (h *SavedQueryHandler) HandleCloneSavedQuery(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	// The new owner is the current user.
	clonedQuery, err := h.service.CloneSavedQuery(r.Context(), id, placeholderUserID, placeholderTenantID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(clonedQuery)
}

// HandleGetSavedQuery retrieves a single saved query.
func (h *SavedQueryHandler) HandleGetSavedQuery(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	query, err := h.service.GetSavedQuery(r.Context(), id, placeholderUserID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(query)
}

// HandleShareQuery creates or updates an ACL for a saved query.
func (h *SavedQueryHandler) HandleShareQuery(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req models.ShareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid request body: " + err.Error()})
		return
	}

	// TODO: Add permission check: user must have write access to the query.

	err := h.service.ShareQuery(r.Context(), id, req, placeholderUserID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// HandleGetPreview retrieves the preview data for a saved query.
func (h *SavedQueryHandler) HandleGetPreview(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	// TODO: Add permission check: user must have read access.
	preview, err := h.service.GetPreview(r.Context(), id)
	if err != nil || preview == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Preview not found or access denied"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(preview)
}

// HandleGetDiff retrieves the latest computed diff for a saved query.
func (h *SavedQueryHandler) HandleGetDiff(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	// TODO: Add permission check: user must have read access.
	diff, err := h.service.GetLatestDiff(r.Context(), id)
	if err != nil || diff == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Diff not found or access denied"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(diff)
}

// HandleGetDuplicates finds and returns clusters of duplicate queries.
func (h *SavedQueryHandler) HandleGetDuplicates(w http.ResponseWriter, r *http.Request) {
	datasourceID := r.URL.Query().Get("datasource_id")
	if datasourceID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "datasource_id is required"})
		return
	}
	// Use placeholder user for now
	clusters, err := h.service.DetectDuplicates(r.Context(), placeholderUserID, datasourceID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to detect duplicates"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(clusters)
}

// GetSecurityContext is a placeholder for auth middleware.
