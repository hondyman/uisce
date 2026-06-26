package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/pagestudio"
	pagetests "github.com/hondyman/semlayer/backend/internal/testing/page_tests"
	"github.com/hondyman/semlayer/backend/internal/testing/performance"
	"github.com/hondyman/semlayer/backend/internal/testing/semantic"
	"github.com/hondyman/semlayer/backend/internal/testing/visual"
)

type TestingHandler struct {
	testGenerator      *pagetests.TestGenerator
	snapshotCapture    *visual.SnapshotCapture
	diffEngine         *visual.DiffEngine
	regressionDetector *semantic.RegressionDetector
	syntheticRunner    *performance.SyntheticRunner
	pageRepo           *pagestudio.Repository
}

func NewTestingHandler(
	gen *pagetests.TestGenerator,
	snap *visual.SnapshotCapture,
	diff *visual.DiffEngine,
	reg *semantic.RegressionDetector,
	perf *performance.SyntheticRunner,
	repo *pagestudio.Repository,
) *TestingHandler {
	return &TestingHandler{
		testGenerator:      gen,
		snapshotCapture:    snap,
		diffEngine:         diff,
		regressionDetector: reg,
		syntheticRunner:    perf,
		pageRepo:           repo,
	}
}

func (h *TestingHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Page Tests
	r.Post("/generate/{pageId}", h.GenerateTests)

	// Visual
	r.Post("/visual/capture/{pageId}", h.CaptureSnapshot)
	r.Get("/visual/diff/{beforeId}/{afterId}", h.CompareDiff)

	// Semantic
	r.Post("/semantic/detect", h.DetectRegression)

	// Performance
	r.Post("/performance/run/{pageId}", h.RunPerformanceTest)

	return r
}

func (h *TestingHandler) GenerateTests(w http.ResponseWriter, r *http.Request) {
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
	suite, _ := h.testGenerator.Generate(r.Context(), page)
	json.NewEncoder(w).Encode(suite)
}

func (h *TestingHandler) CaptureSnapshot(w http.ResponseWriter, r *http.Request) {
	pageID, err := uuid.Parse(chi.URLParam(r, "pageId"))
	if err != nil {
		http.Error(w, "invalid page id", http.StatusBadRequest)
		return
	}
	snapshot, _ := h.snapshotCapture.Capture(r.Context(), pageID)
	json.NewEncoder(w).Encode(snapshot)
}

func (h *TestingHandler) CompareDiff(w http.ResponseWriter, r *http.Request) {
	beforeID, _ := uuid.Parse(chi.URLParam(r, "beforeId"))
	afterID, _ := uuid.Parse(chi.URLParam(r, "afterId"))
	diff, _ := h.diffEngine.Compare(r.Context(), beforeID, afterID)
	json.NewEncoder(w).Encode(diff)
}

func (h *TestingHandler) DetectRegression(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ChangeType   string `json:"change_type"`
		ChangeTarget string `json:"change_target"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	report, _ := h.regressionDetector.DetectImpact(r.Context(), body.ChangeType, body.ChangeTarget)
	json.NewEncoder(w).Encode(report)
}

func (h *TestingHandler) RunPerformanceTest(w http.ResponseWriter, r *http.Request) {
	pageID, err := uuid.Parse(chi.URLParam(r, "pageId"))
	if err != nil {
		http.Error(w, "invalid page id", http.StatusBadRequest)
		return
	}
	metrics, _ := h.syntheticRunner.Run(r.Context(), pageID)
	json.NewEncoder(w).Encode(metrics)
}
