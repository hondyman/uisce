package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/hondyman/semlayer/backend/internal/scheduler_intelligence/ai"
)

// AISchedulerHandler handles AI-related scheduler requests
type AISchedulerHandler struct {
	dagGenerator     *ai.DAGGenerator
	optimizer        *ai.ScheduleOptimizer
	predictiveEngine *ai.PredictiveTriggerEngine
	dependencyHealer *ai.DependencyHealer
	incidentReporter *ai.IncidentReporter
}

// NewAISchedulerHandler creates a new AI scheduler handler
func NewAISchedulerHandler(
	dagGen *ai.DAGGenerator,
	opt *ai.ScheduleOptimizer,
	pred *ai.PredictiveTriggerEngine,
	healer *ai.DependencyHealer,
	reporter *ai.IncidentReporter,
) *AISchedulerHandler {
	return &AISchedulerHandler{
		dagGenerator:     dagGen,
		optimizer:        opt,
		predictiveEngine: pred,
		dependencyHealer: healer,
		incidentReporter: reporter,
	}
}

// RegisterRoutes registers AI scheduler routes
func (h *AISchedulerHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/scheduler/ai", func(r chi.Router) {
		// DAG Generation
		r.Post("/dag/generate", h.GenerateDAG)
		r.Post("/dag/refine", h.RefineDAG)

		// Schedule Optimization
		r.Post("/optimize", h.OptimizeSchedule)

		// Predictive Triggers
		r.Post("/predict/trigger", h.PredictOptimalTrigger)
		r.Get("/predict/capacity-windows", h.GetCapacityWindows)

		// Dependency Healing
		r.Post("/heal/analyze", h.AnalyzeDependencies)
		r.Post("/heal/validate", h.ValidateHealing)

		// Incident Reports
		r.Post("/incident/report", h.GenerateIncidentReport)

		// Suggestions
		r.Get("/suggestions", h.GetSuggestions)
		r.Post("/suggestions/{id}/accept", h.AcceptSuggestion)
		r.Post("/suggestions/{id}/dismiss", h.DismissSuggestion)
	})
}

// GenerateDAG handles DAG generation from natural language
func (h *AISchedulerHandler) GenerateDAG(w http.ResponseWriter, r *http.Request) {
	var intent ai.DAGIntent
	if err := json.NewDecoder(r.Body).Decode(&intent); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	dag, err := h.dagGenerator.GenerateDAG(r.Context(), intent)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dag)
}

// RefineDAG handles DAG refinement based on feedback
func (h *AISchedulerHandler) RefineDAG(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DAG      ai.GeneratedDAG `json:"dag"`
		Feedback string          `json:"feedback"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	refined, err := h.dagGenerator.RefineDAG(r.Context(), &req.DAG, req.Feedback)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(refined)
}

// OptimizeSchedule handles schedule optimization requests
func (h *AISchedulerHandler) OptimizeSchedule(w http.ResponseWriter, r *http.Request) {
	var req ai.OptimizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	result, err := h.optimizer.Optimize(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// PredictOptimalTrigger handles trigger prediction requests
func (h *AISchedulerHandler) PredictOptimalTrigger(w http.ResponseWriter, r *http.Request) {
	var req ai.PredictionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	prediction, err := h.predictiveEngine.PredictOptimalTrigger(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(prediction)
}

// GetCapacityWindows returns optimal execution windows
func (h *AISchedulerHandler) GetCapacityWindows(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := uuid.Parse(r.URL.Query().Get("tenant_id"))
	lookahead := 24 // hours

	windows, err := h.predictiveEngine.PredictCapacityWindow(r.Context(), tenantID, 0, lookahead)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(windows)
}

// AnalyzeDependencies handles dependency analysis requests
func (h *AISchedulerHandler) AnalyzeDependencies(w http.ResponseWriter, r *http.Request) {
	var req ai.HealingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	result, err := h.dependencyHealer.AnalyzeAndHeal(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// ValidateHealing validates proposed healing actions
func (h *AISchedulerHandler) ValidateHealing(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DAG     ai.HealingRequest  `json:"dag"`
		Actions []ai.HealingAction `json:"actions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	valid, warnings := h.dependencyHealer.ValidateHealing(r.Context(), req.DAG, req.Actions)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"valid":    valid,
		"warnings": warnings,
	})
}

// GenerateIncidentReport generates an AI incident report
func (h *AISchedulerHandler) GenerateIncidentReport(w http.ResponseWriter, r *http.Request) {
	var incident ai.IncidentContext
	if err := json.NewDecoder(r.Body).Decode(&incident); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	report, err := h.incidentReporter.GenerateReport(r.Context(), incident)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

// GetSuggestions returns AI-generated scheduling suggestions
func (h *AISchedulerHandler) GetSuggestions(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")

	// In real implementation, would fetch from database
	suggestions := []map[string]interface{}{
		{
			"id":          "sug-001",
			"type":        "schedule_optimization",
			"title":       "Stagger heavy jobs",
			"description": "Move EU Pre-Agg to 1:45 AM to reduce contention",
			"impact":      "15% latency reduction",
			"risk_level":  "low",
			"status":      "pending",
			"tenant_id":   tenantID,
		},
		{
			"id":          "sug-002",
			"type":        "dag_optimization",
			"title":       "Parallelize data load steps",
			"description": "Split serial data loads into parallel branches",
			"impact":      "30% faster DAG execution",
			"risk_level":  "medium",
			"status":      "pending",
			"tenant_id":   tenantID,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(suggestions)
}

// AcceptSuggestion marks a suggestion as accepted
func (h *AISchedulerHandler) AcceptSuggestion(w http.ResponseWriter, r *http.Request) {
	suggestionID := chi.URLParam(r, "id")

	// Would create ChangeSet and apply in real implementation
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"suggestion_id": suggestionID,
		"status":        "accepted",
		"changeset_id":  uuid.NewString(),
	})
}

// DismissSuggestion marks a suggestion as dismissed
func (h *AISchedulerHandler) DismissSuggestion(w http.ResponseWriter, r *http.Request) {
	suggestionID := chi.URLParam(r, "id")

	var req struct {
		Reason string `json:"reason"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"suggestion_id": suggestionID,
		"status":        "dismissed",
		"reason":        req.Reason,
	})
}
