package handlers

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/hondyman/semlayer/backend/internal/risk"
)

// RiskHandler defines HTTP endpoints for the Risk domain
type RiskHandler struct {
	repo risk.RiskRepository
}

// NewRiskHandler creates a new RiskHandler
func NewRiskHandler(repo risk.RiskRepository) *RiskHandler {
	return &RiskHandler{repo: repo}
}

// RegisterRoutes registers the risk endpoints on the given router
func (h *RiskHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/factors", h.ListRiskFactors).Methods("GET")
	r.HandleFunc("/exposures", h.GetSecurityFactorExposures).Methods("GET")
	r.HandleFunc("/portfolio-risk", h.GetPortfolioRisk).Methods("GET")
	r.HandleFunc("/scenarios", h.ListRiskScenarios).Methods("GET")
	r.HandleFunc("/scenarios/{scenarioId}/results", h.GetScenarioResult).Methods("GET")
}

// ListRiskFactors returns system risk factors
func (h *RiskHandler) ListRiskFactors(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	category := r.URL.Query().Get("category")

	factors, err := h.repo.ListRiskFactors(ctx, category)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{"data": factors})
}

// GetSecurityFactorExposures returns factor exposures for a specific security
func (h *RiskHandler) GetSecurityFactorExposures(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	securityIDStr := r.URL.Query().Get("securityId")
	dateStr := r.URL.Query().Get("asOfDate")

	if securityIDStr == "" || dateStr == "" {
		respondWithError(w, http.StatusBadRequest, "securityId and asOfDate are required")
		return
	}

	securityID, err := uuid.Parse(securityIDStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid securityId format")
		return
	}

	asOfDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid asOfDate format")
		return
	}

	exposures, err := h.repo.GetSecurityFactorExposures(ctx, securityID, asOfDate)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{"data": exposures})
}

// GetPortfolioRisk returns calculated risk measures for a portfolio
func (h *RiskHandler) GetPortfolioRisk(w http.ResponseWriter, r *http.Request) {
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

	riskMeasures, err := h.repo.GetPortfolioRisk(ctx, portfolioID, asOfDate)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{"data": riskMeasures})
}

// ListRiskScenarios returns available stress scenarios
func (h *RiskHandler) ListRiskScenarios(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	activeOnly := r.URL.Query().Get("activeOnly") != "false" // default to true

	scenarios, err := h.repo.ListRiskScenarios(ctx, activeOnly)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{"data": scenarios})
}

// GetScenarioResult returns the result of a scenario run on a portfolio
func (h *RiskHandler) GetScenarioResult(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	scenarioIDStr := vars["scenarioId"]
	portfolioIDStr := r.URL.Query().Get("portfolioId")
	dateStr := r.URL.Query().Get("asOfDate")

	if portfolioIDStr == "" || dateStr == "" {
		respondWithError(w, http.StatusBadRequest, "portfolioId and asOfDate are required")
		return
	}

	scenarioID, err := uuid.Parse(scenarioIDStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid scenarioId format")
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

	result, err := h.repo.GetScenarioResult(ctx, scenarioID, portfolioID, asOfDate)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{"data": result})
}
