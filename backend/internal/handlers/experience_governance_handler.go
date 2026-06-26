package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/experience_governance/accessibility"
	"github.com/hondyman/semlayer/backend/internal/experience_governance/patterns"
	"github.com/hondyman/semlayer/backend/internal/experience_governance/safety"
	uxslo "github.com/hondyman/semlayer/backend/internal/experience_governance/ux_slo"
	"github.com/hondyman/semlayer/backend/internal/pagestudio"
)

type ExperienceGovernanceHandler struct {
	sloProvider     *uxslo.SLOProvider
	accessLinter    *accessibility.Linter
	safetyValidator *safety.Validator
	patternEnforcer *patterns.Enforcer
	pageRepo        *pagestudio.Repository
}

func NewExperienceGovernanceHandler(
	slo *uxslo.SLOProvider,
	acc *accessibility.Linter,
	safe *safety.Validator,
	pat *patterns.Enforcer,
	repo *pagestudio.Repository,
) *ExperienceGovernanceHandler {
	return &ExperienceGovernanceHandler{
		sloProvider:     slo,
		accessLinter:    acc,
		safetyValidator: safe,
		patternEnforcer: pat,
		pageRepo:        repo,
	}
}

func (h *ExperienceGovernanceHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/slo/{pageId}", h.GetSLOStatus)
	r.Get("/accessibility/{pageId}", h.CheckAccessibility)
	r.Get("/safety/{pageId}", h.CheckSafety)
	r.Get("/patterns/{pageId}", h.CheckPatterns)
	return r
}

func (h *ExperienceGovernanceHandler) GetSLOStatus(w http.ResponseWriter, r *http.Request) {
	pageID, err := uuid.Parse(chi.URLParam(r, "pageId"))
	if err != nil {
		http.Error(w, "invalid page id", http.StatusBadRequest)
		return
	}
	status, _ := h.sloProvider.EvaluateContracts(r.Context(), pageID)
	json.NewEncoder(w).Encode(status)
}

func (h *ExperienceGovernanceHandler) CheckAccessibility(w http.ResponseWriter, r *http.Request) {
	pageID, err := uuid.Parse(chi.URLParam(r, "pageId"))
	if err != nil {
		http.Error(w, "invalid page id", http.StatusBadRequest)
		return
	}
	page, err := h.pageRepo.GetPage(r.Context(), pageID)
	if err != nil {
		http.Error(w, "page not found", http.StatusNotFound)
		return
	}
	report, _ := h.accessLinter.LintPage(r.Context(), page)
	json.NewEncoder(w).Encode(report)
}

func (h *ExperienceGovernanceHandler) CheckSafety(w http.ResponseWriter, r *http.Request) {
	pageID, err := uuid.Parse(chi.URLParam(r, "pageId"))
	if err != nil {
		http.Error(w, "invalid page id", http.StatusBadRequest)
		return
	}
	page, err := h.pageRepo.GetPage(r.Context(), pageID)
	if err != nil {
		http.Error(w, "page not found", http.StatusNotFound)
		return
	}
	violations, _ := h.safetyValidator.Validate(r.Context(), page)
	json.NewEncoder(w).Encode(violations)
}

func (h *ExperienceGovernanceHandler) CheckPatterns(w http.ResponseWriter, r *http.Request) {
	pageID, err := uuid.Parse(chi.URLParam(r, "pageId"))
	if err != nil {
		http.Error(w, "invalid page id", http.StatusBadRequest)
		return
	}
	page, err := h.pageRepo.GetPage(r.Context(), pageID)
	if err != nil {
		http.Error(w, "page not found", http.StatusNotFound)
		return
	}
	report, _ := h.patternEnforcer.CheckCompliance(r.Context(), page)
	json.NewEncoder(w).Encode(report)
}
