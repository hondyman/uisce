package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/semantic_intelligence/docs"
	"github.com/hondyman/semlayer/backend/internal/semantic_intelligence/forecasting"
	"github.com/hondyman/semlayer/backend/internal/semantic_intelligence/quality"
	"github.com/hondyman/semlayer/backend/internal/semantic_intelligence/refactoring"
	"github.com/hondyman/semlayer/backend/internal/semantic_intelligence/relationships"
)

type AdvancedSemanticHandler struct {
	refactoringAnalyzer   *refactoring.RefactoringAnalyzer
	qualityScorer         *quality.QualityScorer
	docGenerator          *docs.DocumentationGenerator
	relationshipDiscovery *relationships.RelationshipDiscovery
	driftForecaster       *forecasting.DriftForecaster
}

func NewAdvancedSemanticHandler(
	refactor *refactoring.RefactoringAnalyzer,
	quality *quality.QualityScorer,
	docs *docs.DocumentationGenerator,
	rels *relationships.RelationshipDiscovery,
	forecast *forecasting.DriftForecaster,
) *AdvancedSemanticHandler {
	return &AdvancedSemanticHandler{
		refactoringAnalyzer:   refactor,
		qualityScorer:         quality,
		docGenerator:          docs,
		relationshipDiscovery: rels,
		driftForecaster:       forecast,
	}
}

func (h *AdvancedSemanticHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Refactoring
	r.Get("/refactoring/proposals", h.GetRefactoringProposals)

	// Quality
	r.Get("/quality/{boId}", h.GetQualityScore)

	// Documentation
	r.Post("/documentation/generate/{boId}", h.GenerateDocumentation)
	r.Post("/documentation/export/{boId}", h.ExportDocumentation)

	// Relationships
	r.Get("/relationships/suggestions", h.GetRelationshipSuggestions)

	// Forecasting
	r.Get("/forecasting/risk-dashboard", h.GetRiskDashboard)

	return r
}

func (h *AdvancedSemanticHandler) GetRefactoringProposals(w http.ResponseWriter, r *http.Request) {
	proposals, _ := h.refactoringAnalyzer.AnalyzeGraph(r.Context())
	json.NewEncoder(w).Encode(proposals)
}

func (h *AdvancedSemanticHandler) GetQualityScore(w http.ResponseWriter, r *http.Request) {
	boID, err := uuid.Parse(chi.URLParam(r, "boId"))
	if err != nil {
		http.Error(w, "invalid bo id", http.StatusBadRequest)
		return
	}
	report, _ := h.qualityScorer.ScoreBO(r.Context(), boID)
	json.NewEncoder(w).Encode(report)
}

func (h *AdvancedSemanticHandler) GenerateDocumentation(w http.ResponseWriter, r *http.Request) {
	boID, err := uuid.Parse(chi.URLParam(r, "boId"))
	if err != nil {
		http.Error(w, "invalid bo id", http.StatusBadRequest)
		return
	}
	doc, _ := h.docGenerator.Generate(r.Context(), boID)
	json.NewEncoder(w).Encode(doc)
}

func (h *AdvancedSemanticHandler) ExportDocumentation(w http.ResponseWriter, r *http.Request) {
	boID, err := uuid.Parse(chi.URLParam(r, "boId"))
	if err != nil {
		http.Error(w, "invalid bo id", http.StatusBadRequest)
		return
	}

	var body struct {
		Format docs.ExportFormat `json:"format"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	doc, _ := h.docGenerator.Generate(r.Context(), boID)
	exported, _ := h.docGenerator.Export(r.Context(), doc, body.Format)

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(exported))
}

func (h *AdvancedSemanticHandler) GetRelationshipSuggestions(w http.ResponseWriter, r *http.Request) {
	suggestions, _ := h.relationshipDiscovery.DiscoverRelationships(r.Context())
	json.NewEncoder(w).Encode(suggestions)
}

func (h *AdvancedSemanticHandler) GetRiskDashboard(w http.ResponseWriter, r *http.Request) {
	dashboard, _ := h.driftForecaster.ForecastDrift(r.Context())
	json.NewEncoder(w).Encode(dashboard)
}
