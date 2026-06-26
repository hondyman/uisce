package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/analytics"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/jmoiron/sqlx"
)

// AISuggestPropertiesHandler handles AI-powered semantic enrichment requests
type AISuggestPropertiesHandler struct {
	db          *sqlx.DB
	llmProvider interface{}
}

// NewAISuggestPropertiesHandler creates a new handler
func NewAISuggestPropertiesHandler(db *sqlx.DB, llmProvider interface{}) *AISuggestPropertiesHandler {
	return &AISuggestPropertiesHandler{
		db:          db,
		llmProvider: llmProvider,
	}
}

// AISuggestRequest represents the request body
type AISuggestRequest struct {
	NodeKey      string                   `json:"node_key"`
	CubeDef      analytics.CubeDefinition `json:"cube_def"`
	SemanticType string                   `json:"semantic_type,omitempty"`
}

// AISuggestResponse represents the response body
type AISuggestResponse struct {
	Enrichment *analytics.SemanticEnrichment `json:"enrichment"`
	Reasoning  string                        `json:"reasoning,omitempty"`
	Success    bool                          `json:"success"`
	Error      string                        `json:"error,omitempty"`
}

// Handle processes the AI suggestion request
// POST /api/ai-suggest-properties
func (h *AISuggestPropertiesHandler) Handle(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLogger().Sugar()

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req AISuggestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Errorf("Failed to decode request: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(AISuggestResponse{
			Success: false,
			Error:   "Invalid request body: " + err.Error(),
		})
		return
	}

	// Validate request
	if req.CubeDef.Column == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(AISuggestResponse{
			Success: false,
			Error:   "cube_def.column is required",
		})
		return
	}

	// Create enricher and get AI-enhanced enrichment
	enricher := analytics.NewSemanticEnricher(h.db, h.llmProvider)

	enrichment, err := enricher.EnrichWithAIFromColumnData(
		r.Context(),
		req.CubeDef.Column,
		req.CubeDef.Table,
		req.CubeDef.DataType,
		req.SemanticType,
		req.CubeDef.IsNullable,
		req.CubeDef.IsForeignKey,
		req.CubeDef.IsPrimaryKey,
	)

	if err != nil {
		logger.Errorf("AI enrichment failed: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(AISuggestResponse{
			Success: false,
			Error:   "AI enrichment failed: " + err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AISuggestResponse{
		Enrichment: enrichment,
		Success:    true,
	})
}

// HandleBatch processes a batch of AI suggestion requests
// POST /api/ai-suggest-properties/batch
func (h *AISuggestPropertiesHandler) HandleBatch(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLogger().Sugar()

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var requests []AISuggestRequest
	if err := json.NewDecoder(r.Body).Decode(&requests); err != nil {
		logger.Errorf("Failed to decode batch request: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	enricher := analytics.NewSemanticEnricher(h.db, h.llmProvider)
	results := make([]AISuggestResponse, len(requests))

	for i, req := range requests {
		enrichment, err := enricher.EnrichWithAIFromColumnData(
			r.Context(),
			req.CubeDef.Column,
			req.CubeDef.Table,
			req.CubeDef.DataType,
			req.SemanticType,
			req.CubeDef.IsNullable,
			req.CubeDef.IsForeignKey,
			req.CubeDef.IsPrimaryKey,
		)

		if err != nil {
			results[i] = AISuggestResponse{
				Success: false,
				Error:   err.Error(),
			}
		} else {
			results[i] = AISuggestResponse{
				Enrichment: enrichment,
				Success:    true,
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"results": results,
		"total":   len(results),
	})
}

// RegisterRoutes registers the AI suggestion routes using chi router
func (h *AISuggestPropertiesHandler) RegisterRoutes(r chi.Router) {
	r.Route("/ai-suggest", func(r chi.Router) {
		r.Post("/properties", h.Handle)
		r.Post("/properties/batch", h.HandleBatch)
	})
}
