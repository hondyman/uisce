package handlers

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/hondyman/semlayer/backend/internal/compliance"
)

// ComplianceHandler defines HTTP endpoints for the Compliance domain
type ComplianceHandler struct {
	repo compliance.ComplianceRepository
}

// NewComplianceHandler creates a new ComplianceHandler
func NewComplianceHandler(repo compliance.ComplianceRepository) *ComplianceHandler {
	return &ComplianceHandler{repo: repo}
}

// RegisterRoutes registers the compliance endpoints on the given router
func (h *ComplianceHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/rules", h.ListComplianceRules).Methods("GET")
	r.HandleFunc("/evaluations", h.ListComplianceEvaluations).Methods("GET")
	r.HandleFunc("/breaches", h.ListComplianceBreaches).Methods("GET")
	r.HandleFunc("/evaluations/{evaluationId}/lineage", h.GetLineageForEvaluation).Methods("GET")
}

// ListComplianceRules returns all active compliance rules
func (h *ComplianceHandler) ListComplianceRules(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	includeInactive := r.URL.Query().Get("includeInactive") == "true"

	rules, err := h.repo.ListComplianceRules(ctx, includeInactive)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{"data": rules})
}

// ListComplianceEvaluations returns evaluation results for a portfolio
func (h *ComplianceHandler) ListComplianceEvaluations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	portfolioIDStr := r.URL.Query().Get("portfolioId")
	dateStr := r.URL.Query().Get("asOfDate")

	if portfolioIDStr == "" || dateStr == "" {
		respondWithError(w, http.StatusBadRequest, "portfolioId and asOfDate are required")
		return
	}

	portfolioID, err := uuid.Parse(portfolioIDStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid portfolioId format")
		return
	}

	asOfDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid asOfDate format")
		return
	}

	evals, err := h.repo.ListComplianceEvaluations(ctx, portfolioID, asOfDate)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{"data": evals})
}

// ListComplianceBreaches returns breaches for a portfolio
func (h *ComplianceHandler) ListComplianceBreaches(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	portfolioIDStr := r.URL.Query().Get("portfolioId")
	status := r.URL.Query().Get("status")

	if portfolioIDStr == "" {
		respondWithError(w, http.StatusBadRequest, "portfolioId is required")
		return
	}

	portfolioID, err := uuid.Parse(portfolioIDStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid portfolioId format")
		return
	}

	breaches, err := h.repo.ListComplianceBreaches(ctx, portfolioID, status)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{"data": breaches})
}

// GetLineageForEvaluation returns the execution trace for a specific evaluation
func (h *ComplianceHandler) GetLineageForEvaluation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	evalIDStr := vars["evaluationId"]

	evalID, err := uuid.Parse(evalIDStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid evaluationId format")
		return
	}

	lineages, err := h.repo.GetLineageForEvaluation(ctx, evalID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{"data": lineages})
}
