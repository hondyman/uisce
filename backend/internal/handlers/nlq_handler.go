package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hondyman/semlayer/backend/internal/services"
)

// NLQHandler handles Natural Language Query requests.
type NLQHandler struct {
	nlqService   *services.NLQService
	securityDeps SecurityContextDeps
}

// NewNLQHandler creates a new NLQ handler.
func NewNLQHandler(nlqService *services.NLQService, deps SecurityContextDeps) *NLQHandler {
	return &NLQHandler{
		nlqService:   nlqService,
		securityDeps: deps,
	}
}

// HandleAsk processes a natural language question about the catalog.
// POST /api/nlq/ask
func (h *NLQHandler) HandleAsk(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var req services.AskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}
	secCtx, ctx, err := SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Question == "" {
		http.Error(w, "question field is required", http.StatusBadRequest)
		return
	}

	// Process the question
	resp, err := h.nlqService.Ask(ctx, secCtx, req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to process question: %v", err), http.StatusInternalServerError)
		return
	}

	// Return structured response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// HandleSearch searches catalog entities semantically.
// POST /api/nlq/search
func (h *NLQHandler) HandleSearch(w http.ResponseWriter, r *http.Request) {
	_, _, err := SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var req struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.Query == "" {
		http.Error(w, "query field is required", http.StatusBadRequest)
		return
	}

	if req.Limit <= 0 {
		req.Limit = 10
	}

	// This is a placeholder for semantic search across catalog
	// You would implement similar logic to findRelevantEntity but return multiple results
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"results": []map[string]string{},
		"message": "Semantic search endpoint - implementation pending",
	})
}
