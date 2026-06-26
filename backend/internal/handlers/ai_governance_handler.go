package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/ai_governance/audit"
	"github.com/hondyman/semlayer/backend/internal/ai_governance/reviewer"
	"github.com/hondyman/semlayer/backend/internal/ai_governance/risk"
	testplans "github.com/hondyman/semlayer/backend/internal/ai_governance/test_plans"
)

type AIGovernanceHandler struct {
	aiReviewer          *reviewer.AIReviewer
	auditTrailGenerator *audit.AuditTrailGenerator
	riskScorer          *risk.RiskScorer
	testPlanGenerator   *testplans.TestPlanGenerator
}

func NewAIGovernanceHandler(
	rev *reviewer.AIReviewer,
	aud *audit.AuditTrailGenerator,
	rsk *risk.RiskScorer,
	test *testplans.TestPlanGenerator,
) *AIGovernanceHandler {
	return &AIGovernanceHandler{
		aiReviewer:          rev,
		auditTrailGenerator: aud,
		riskScorer:          rsk,
		testPlanGenerator:   test,
	}
}

func (h *AIGovernanceHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// AI Review
	r.Post("/review/{changesetId}", h.ReviewChangeSet)

	// Audit Trail
	r.Get("/audit-trail/{changesetId}", h.GetAuditTrail)
	r.Get("/audit-trail/{changesetId}/export", h.ExportAuditTrail)

	// Risk Scoring
	r.Get("/risk-score/{changesetId}", h.GetRiskScore)

	// Test Plans
	r.Post("/test-plan/generate/{changesetId}", h.GenerateTestPlan)

	return r
}

func (h *AIGovernanceHandler) ReviewChangeSet(w http.ResponseWriter, r *http.Request) {
	changesetID, err := uuid.Parse(chi.URLParam(r, "changesetId"))
	if err != nil {
		http.Error(w, "invalid changeset id", http.StatusBadRequest)
		return
	}
	report, _ := h.aiReviewer.Review(r.Context(), changesetID)
	json.NewEncoder(w).Encode(report)
}

func (h *AIGovernanceHandler) GetAuditTrail(w http.ResponseWriter, r *http.Request) {
	changesetID, err := uuid.Parse(chi.URLParam(r, "changesetId"))
	if err != nil {
		http.Error(w, "invalid changeset id", http.StatusBadRequest)
		return
	}
	trail, _ := h.auditTrailGenerator.Generate(r.Context(), changesetID)
	json.NewEncoder(w).Encode(trail)
}

func (h *AIGovernanceHandler) ExportAuditTrail(w http.ResponseWriter, r *http.Request) {
	changesetID, err := uuid.Parse(chi.URLParam(r, "changesetId"))
	if err != nil {
		http.Error(w, "invalid changeset id", http.StatusBadRequest)
		return
	}

	format := r.URL.Query().Get("format")
	trail, _ := h.auditTrailGenerator.Generate(r.Context(), changesetID)

	if format == "pdf" {
		pdfPath, _ := h.auditTrailGenerator.ExportPDF(r.Context(), trail)
		w.Header().Set("Content-Type", "application/pdf")
		w.Write([]byte(pdfPath))
	} else {
		markdown := h.auditTrailGenerator.ExportMarkdown(r.Context(), trail)
		w.Header().Set("Content-Type", "text/markdown")
		w.Write([]byte(markdown))
	}
}

func (h *AIGovernanceHandler) GetRiskScore(w http.ResponseWriter, r *http.Request) {
	changesetID, err := uuid.Parse(chi.URLParam(r, "changesetId"))
	if err != nil {
		http.Error(w, "invalid changeset id", http.StatusBadRequest)
		return
	}
	score, _ := h.riskScorer.Score(r.Context(), changesetID)
	json.NewEncoder(w).Encode(score)
}

func (h *AIGovernanceHandler) GenerateTestPlan(w http.ResponseWriter, r *http.Request) {
	changesetID, err := uuid.Parse(chi.URLParam(r, "changesetId"))
	if err != nil {
		http.Error(w, "invalid changeset id", http.StatusBadRequest)
		return
	}
	plan, _ := h.testPlanGenerator.Generate(r.Context(), changesetID)
	json.NewEncoder(w).Encode(plan)
}
