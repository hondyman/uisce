package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/analytics"
)

// BOSearchHandler handles BO search requests
type BOSearchHandler struct {
	SearchService *analytics.BOSearchService
}

// NewBOSearchHandler creates a new search handler
func NewBOSearchHandler(searchService *analytics.BOSearchService) *BOSearchHandler {
	return &BOSearchHandler{SearchService: searchService}
}

// RegisterRoutes registers search routes
func (h *BOSearchHandler) RegisterRoutes(r chi.Router) {
	r.Get("/api/bo/search", h.Search)
}

// Search handles BO search requests
// GET /api/bo/search?q=balance&type=term&limit=50&offset=0&domain=wealth
func (h *BOSearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	searchType := analytics.SearchType(r.URL.Query().Get("type"))
	if searchType == "" {
		searchType = analytics.SearchTypeAll
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit == 0 || limit > 100 {
		limit = 50
	}

	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	domain := r.URL.Query().Get("domain")

	results, err := h.SearchService.Search(query, searchType, limit, offset, domain)
	if err != nil {
		http.Error(w, "Search failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}
