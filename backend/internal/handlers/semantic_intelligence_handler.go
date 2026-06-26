package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/pagestudio"
	"github.com/hondyman/semlayer/backend/internal/semantic_intelligence/consistency"
	"github.com/hondyman/semlayer/backend/internal/semantic_intelligence/drift"
	"github.com/hondyman/semlayer/backend/internal/semantic_intelligence/healing"
	"github.com/hondyman/semlayer/backend/internal/semantic_intelligence/patterns"
)

type SemanticIntelligenceHandler struct {
	driftDetector        *drift.DriftDetector
	healingEngine        *healing.HealingEngine
	patternLearner       *patterns.Learner
	consistencyValidator *consistency.Validator
	pageRepo             *pagestudio.Repository
}

func NewSemanticIntelligenceHandler(dd *drift.DriftDetector, pageRepo *pagestudio.Repository) *SemanticIntelligenceHandler {
	return &SemanticIntelligenceHandler{
		driftDetector:        dd,
		healingEngine:        healing.NewHealingEngine(),
		patternLearner:       patterns.NewLearner(pageRepo),
		consistencyValidator: consistency.NewValidator(),
		pageRepo:             pageRepo,
	}
}

func (h *SemanticIntelligenceHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/drift/{pageId}", h.CheckPageDrift)
	r.Post("/heal/{pageId}", h.HealPage)
	r.Get("/patterns", h.ListPatterns)
	r.Get("/consistency/{pageId}", h.CheckConsistency)
	return r
}

func (h *SemanticIntelligenceHandler) CheckConsistency(w http.ResponseWriter, r *http.Request) {
	pageIdStr := chi.URLParam(r, "pageId")
	pageID, err := uuid.Parse(pageIdStr)
	if err != nil {
		http.Error(w, "invalid page id", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	page, err := h.pageRepo.GetPage(ctx, pageID)
	if err != nil {
		http.Error(w, "page not found", http.StatusNotFound)
		return
	}

	issues, err := h.consistencyValidator.Validate(ctx, page)
	if err != nil {
		http.Error(w, "validation failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(issues)
}

func (h *SemanticIntelligenceHandler) ListPatterns(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// trigger learning on the fly for MVP
	patterns, err := h.patternLearner.LearnPatterns(ctx)
	if err != nil {
		http.Error(w, "failed to learn patterns: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(patterns)
}

func (h *SemanticIntelligenceHandler) HealPage(w http.ResponseWriter, r *http.Request) {
	pageIdStr := chi.URLParam(r, "pageId")
	pageID, err := uuid.Parse(pageIdStr)
	if err != nil {
		http.Error(w, "invalid page id", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	page, err := h.pageRepo.GetPage(ctx, pageID)
	if err != nil {
		http.Error(w, "page not found", http.StatusNotFound)
		return
	}

	// 1. Get Drift
	// We need computed fingerprint again - logic duplicated from CheckPageDrift, refactor in real world
	var fp pagestudio.PageFingerprint
	if len(page.SemanticFingerprint) > 0 {
		_ = json.Unmarshal(page.SemanticFingerprint, &fp)
	} else {
		computed, _ := pagestudio.ComputeFingerprint(page)
		fp = *computed
	}

	report, err := h.driftDetector.CheckDrift(ctx, pageIdStr, &fp)
	if err != nil {
		http.Error(w, "failed to check drift: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if !report.HasDrift {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "no_drift", "message": "No drift detected, no healing needed"})
		return
	}

	// 2. Propose Fix
	proposal, err := h.healingEngine.GenerateProposal(ctx, page, report)
	if err != nil {
		http.Error(w, "failed to generate proposal: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(proposal)
}

func (h *SemanticIntelligenceHandler) CheckPageDrift(w http.ResponseWriter, r *http.Request) {
	pageIdStr := chi.URLParam(r, "pageId")
	pageID, err := uuid.Parse(pageIdStr)
	if err != nil {
		http.Error(w, "invalid page id", http.StatusBadRequest)
		return
	}

	// Fetch Page
	ctx := r.Context()
	page, err := h.pageRepo.GetPage(ctx, pageID)
	if err != nil {
		http.Error(w, "page not found", http.StatusNotFound)
		return
	}

	// Parse Fingerprint
	var fp pagestudio.PageFingerprint
	if len(page.SemanticFingerprint) > 0 {
		if err := json.Unmarshal(page.SemanticFingerprint, &fp); err != nil {
			// No fingerprint or invalid, compute on the fly?
			// For now, if invalid, treat as no fingerprint
			http.Error(w, "invalid fingerprint", http.StatusInternalServerError)
			return
		}
	} else {
		// Fingerprint missing, compute it now (and maybe save?)
		computed, err := pagestudio.ComputeFingerprint(page)
		if err != nil {
			http.Error(w, "failed to compute fingerprint", http.StatusInternalServerError)
			return
		}
		fp = *computed
	}

	// Check Drift
	report, err := h.driftDetector.CheckDrift(ctx, pageIdStr, &fp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}
