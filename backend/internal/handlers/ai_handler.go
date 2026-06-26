package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/ai"
	"github.com/hondyman/semlayer/backend/internal/indexing"
	"github.com/hondyman/semlayer/backend/internal/values"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

type AIHandler struct {
	Service *ai.AIService
}

func NewAIHandler(service *ai.AIService) *AIHandler {
	return &AIHandler{Service: service}
}

func (h *AIHandler) AnalyzeSignal(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	signals, err := h.Service.AnalyzeText(r.Context(), input.Text)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(signals)
}

func (h *AIHandler) GenerateConstraints(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Prompt string `json:"prompt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	constraints, err := h.Service.GenerateConstraints(r.Context(), input.Prompt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(constraints)
}

func (h *AIHandler) ExplainPortfolio(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Portfolio   indexing.Portfolio  `json:"portfolio"`
		Constraints []values.Constraint `json:"constraints"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	explanation, err := h.Service.ExplainPortfolio(r.Context(), input.Portfolio, input.Constraints)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"explanation": explanation})
}

// GetRuleSuggestions returns AI-generated suggestions for a business object
func (h *AIHandler) GetRuleSuggestions(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := uuid.Parse(jwtmiddleware.GetClaimsFromContext(r).TenantID)
	bo := r.URL.Query().Get("business_object")

	suggestions, err := h.Service.GetRuleSuggestions(r.Context(), tenantID, bo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(suggestions)
}

// AnalyzeImpact performs a what-if simulation for a rule suggestion
func (h *AIHandler) AnalyzeImpact(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := uuid.Parse(jwtmiddleware.GetClaimsFromContext(r).TenantID)
	var suggestion ai.RuleSuggestion
	if err := json.NewDecoder(r.Body).Decode(&suggestion); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	report, err := h.Service.PerformImpactAnalysis(r.Context(), tenantID, suggestion)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

// GetDriftPredictions returns AI-generated drift predictions for a tenant
func (h *AIHandler) GetDriftPredictions(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := uuid.Parse(jwtmiddleware.GetClaimsFromContext(r).TenantID)
	params := ai.DriftPredictionParams{
		BusinessObject: r.URL.Query().Get("business_object"),
		SemanticTerm:   r.URL.Query().Get("semantic_term"),
		Region:         r.URL.Query().Get("region"),
	}

	predictions, err := h.Service.GetDriftPredictions(r.Context(), tenantID, params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(predictions)
}

// SuggestRuleTemplates returns AI-generated rule templates based on usage clusters
func (h *AIHandler) SuggestRuleTemplates(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := uuid.Parse(jwtmiddleware.GetClaimsFromContext(r).TenantID)

	templates, err := h.Service.SuggestRuleTemplates(r.Context(), tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(templates)
}

// SubmitFeedback processes user feedback on a rule suggestion
func (h *AIHandler) SubmitFeedback(w http.ResponseWriter, r *http.Request) {
	var feedback ai.UserFeedback
	if err := json.NewDecoder(r.Body).Decode(&feedback); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if feedback.TenantID == uuid.Nil {
		feedback.TenantID, _ = uuid.Parse(jwtmiddleware.GetClaimsFromContext(r).TenantID)
	}

	resp, err := h.Service.SubmitFeedback(r.Context(), feedback)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (h *AIHandler) RegisterRoutes(r chi.Router) {
	r.Route("/ai", func(r chi.Router) {
		r.Post("/analyze-signal", h.AnalyzeSignal)
		r.Post("/generate-constraints", h.GenerateConstraints)
		r.Post("/explain-portfolio", h.ExplainPortfolio)
		// Feature 1
		r.Get("/rule-suggestions", h.GetRuleSuggestions)
		r.Post("/impact-analysis", h.AnalyzeImpact)
		// Feature 2: Predictive Drift Detection
		r.Get("/drift-predictions", h.GetDriftPredictions)
		// Feature 3: AI Rule Templates
		r.Get("/rule-templates", h.SuggestRuleTemplates)
		// Feedback loop
		r.Post("/feedback", h.SubmitFeedback)
	})
}
