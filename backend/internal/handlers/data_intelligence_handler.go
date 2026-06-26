package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/data_intelligence/contracts"
	"github.com/hondyman/semlayer/backend/internal/data_intelligence/indexing"
	"github.com/hondyman/semlayer/backend/internal/data_intelligence/quality"
	"github.com/hondyman/semlayer/backend/internal/data_intelligence/tiering"
)

type DataIntelligenceHandler struct {
	indexAdvisor      *indexing.IndexAdvisor
	storageTiering    *tiering.StorageTiering
	qualityMonitor    *quality.QualityMonitor
	contractGenerator *contracts.ContractGenerator
}

func NewDataIntelligenceHandler(
	idx *indexing.IndexAdvisor,
	tier *tiering.StorageTiering,
	qual *quality.QualityMonitor,
	cont *contracts.ContractGenerator,
) *DataIntelligenceHandler {
	return &DataIntelligenceHandler{
		indexAdvisor:      idx,
		storageTiering:    tier,
		qualityMonitor:    qual,
		contractGenerator: cont,
	}
}

func (h *DataIntelligenceHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Index Advisor
	r.Get("/indexes/suggestions", h.GetIndexSuggestions)
	r.Post("/indexes/apply", h.ApplyIndexSuggestion)

	// Storage Tiering
	r.Get("/tiering/plan/{tenantId}", h.GetTieringPlan)
	r.Post("/tiering/execute", h.ExecuteTieringPlan)

	// Data Quality
	r.Get("/quality/issues", h.GetQualityIssues)
	r.Get("/quality/scores/{tableName}", h.GetQualityScore)

	// Data Contracts
	r.Get("/contracts/suggestions", h.GetContractSuggestions)
	r.Post("/contracts/apply", h.ApplyContractSuggestion)

	return r
}

func (h *DataIntelligenceHandler) GetIndexSuggestions(w http.ResponseWriter, r *http.Request) {
	suggestions, _ := h.indexAdvisor.AnalyzeAndSuggest(r.Context())
	json.NewEncoder(w).Encode(suggestions)
}

func (h *DataIntelligenceHandler) ApplyIndexSuggestion(w http.ResponseWriter, r *http.Request) {
	var suggestion indexing.IndexSuggestion
	if err := json.NewDecoder(r.Body).Decode(&suggestion); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	changeset, _ := h.indexAdvisor.GenerateChangeSet(r.Context(), &suggestion)
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(changeset))
}

func (h *DataIntelligenceHandler) GetTieringPlan(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	plan, _ := h.storageTiering.GeneratePlan(r.Context(), tenantID)
	json.NewEncoder(w).Encode(plan)
}

func (h *DataIntelligenceHandler) ExecuteTieringPlan(w http.ResponseWriter, r *http.Request) {
	var plan tiering.TieringPlan
	if err := json.NewDecoder(r.Body).Decode(&plan); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err := h.storageTiering.ExecutePlan(r.Context(), &plan)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *DataIntelligenceHandler) GetQualityIssues(w http.ResponseWriter, r *http.Request) {
	issues, _ := h.qualityMonitor.DetectIssues(r.Context())
	json.NewEncoder(w).Encode(issues)
}

func (h *DataIntelligenceHandler) GetQualityScore(w http.ResponseWriter, r *http.Request) {
	tableName := chi.URLParam(r, "tableName")
	score, _ := h.qualityMonitor.ScoreTable(r.Context(), tableName)
	json.NewEncoder(w).Encode(score)
}

func (h *DataIntelligenceHandler) GetContractSuggestions(w http.ResponseWriter, r *http.Request) {
	suggestions, _ := h.contractGenerator.AnalyzeAndSuggest(r.Context())
	json.NewEncoder(w).Encode(suggestions)
}

func (h *DataIntelligenceHandler) ApplyContractSuggestion(w http.ResponseWriter, r *http.Request) {
	var suggestion contracts.ContractSuggestion
	if err := json.NewDecoder(r.Body).Decode(&suggestion); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	changeset, _ := h.contractGenerator.GenerateChangeSet(r.Context(), &suggestion)
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(changeset))
}
