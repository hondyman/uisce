package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/hondyman/semlayer/backend/internal/services"
	models "github.com/hondyman/semlayer/backend/models"
)

// SearchHandler handles API requests for search functionality.
type SearchHandler struct {
	service      *services.SearchService
	securityDeps SecurityContextDeps
}

// NewSearchHandler creates a new SearchHandler.
func NewSearchHandler(s *services.SearchService, deps SecurityContextDeps) *SearchHandler {
	return &SearchHandler{service: s, securityDeps: deps}
}

// HandleSemanticSearch performs a semantic search across assets.
func (h *SearchHandler) HandleSemanticSearch(w http.ResponseWriter, r *http.Request) {
	var req models.SemanticSearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid request body: " + err.Error()})
		return
	}

	// In a real app, this would come from auth middleware
	secCtx, ctx, err := SecurityContextFromRequest(r, req.DatasourceID, req.Region, h.securityDeps)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	req.Region = secCtx.Region
	req.DatasourceID = secCtx.DatasourceID
	results, err := h.service.SemanticSearch(ctx, req, *secCtx)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Search failed", "details": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// HandleLogFeedback logs a user's interaction with search results.
func (h *SearchHandler) HandleLogFeedback(w http.ResponseWriter, r *http.Request) {
	var req models.SearchFeedbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid request body: " + err.Error()})
		return
	}

	// Use placeholder user for now
	userID := placeholderUserID
	// Run in a goroutine so it doesn't block the user's next action.
	go func() { _ = h.service.LogFeedback(r.Context(), req, userID) }()

	w.WriteHeader(http.StatusAccepted)
}

// HandleGetSuggestions returns personalized suggestions for a user.
func (h *SearchHandler) HandleGetSuggestions(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	userID := query.Get("user_id")
	datasourceID := query.Get("datasource_id")
	if userID == "" || datasourceID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "user_id and datasource_id are required"})
		return
	}

	suggestions, err := h.service.GetSuggestions(r.Context(), userID, datasourceID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to get suggestions", "details": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(suggestions)
}
