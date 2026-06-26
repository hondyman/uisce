package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/aso"
)

// ASOEnhancedHandler handles Phase 7 ASO endpoints
type ASOEnhancedHandler struct {
	costService    aso.CostAttributionService
	anomalyService aso.AnomalyDetectionService
	healingService aso.SelfHealingService
}

// NewASOEnhancedHandler creates a new enhanced handler
func NewASOEnhancedHandler(
	costService aso.CostAttributionService,
	anomalyService aso.AnomalyDetectionService,
	healingService aso.SelfHealingService,
) *ASOEnhancedHandler {
	return &ASOEnhancedHandler{
		costService:    costService,
		anomalyService: anomalyService,
		healingService: healingService,
	}
}

// RegisterASOEnhancedRoutes registers enhanced ASO routes
func RegisterASOEnhancedRoutes(r chi.Router, h *ASOEnhancedHandler) {
	r.Route("/aso", func(r chi.Router) {
		// Cost Attribution
		r.Get("/costs/global", h.GetGlobalSavings)
		r.Get("/costs/tenant/{tenantId}", h.GetTenantSavings)
		r.Get("/costs/optimization/{id}", h.GetOptimizationCosts)

		// Anomaly Detection
		r.Get("/drift", h.GetOpenDriftSignals)
		r.Get("/drift/{id}", h.GetDriftSignal)
		r.Post("/drift/{id}/acknowledge", h.AcknowledgeDrift)
		r.Post("/drift/{id}/resolve", h.ResolveDrift)
		r.Post("/drift/scan", h.TriggerAnomalyScan)

		// Self-Healing
		r.Get("/healing", h.GetHealingActions)
		r.Get("/healing/{targetId}/history", h.GetHealingHistory)
		r.Post("/healing/{signalId}/trigger", h.TriggerHealing)
	})
}

// ============================================================================
// Cost Attribution Endpoints
// ============================================================================

// GetGlobalSavings returns platform-wide cost savings
func (h *ASOEnhancedHandler) GetGlobalSavings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	since := time.Now().AddDate(0, 0, -30) // Last 30 days
	if sinceStr := r.URL.Query().Get("since"); sinceStr != "" {
		if t, err := time.Parse("2006-01-02", sinceStr); err == nil {
			since = t
		}
	}

	summary, err := h.costService.GetGlobalSavings(ctx, since)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// GetTenantSavings returns cost savings for a specific tenant
func (h *ASOEnhancedHandler) GetTenantSavings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantIDStr := chi.URLParam(r, "tenantId")

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		http.Error(w, "Invalid tenant ID", http.StatusBadRequest)
		return
	}

	since := time.Now().AddDate(0, 0, -30)
	if sinceStr := r.URL.Query().Get("since"); sinceStr != "" {
		if t, err := time.Parse("2006-01-02", sinceStr); err == nil {
			since = t
		}
	}

	summary, err := h.costService.GetTenantSavings(ctx, tenantID, since)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// GetOptimizationCosts returns cost metrics for a specific optimization
func (h *ASOEnhancedHandler) GetOptimizationCosts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid optimization ID", http.StatusBadRequest)
		return
	}

	metrics, err := h.costService.CalculateCostMetrics(ctx, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// ============================================================================
// Anomaly Detection Endpoints
// ============================================================================

// GetOpenDriftSignals returns unresolved drift signals
func (h *ASOEnhancedHandler) GetOpenDriftSignals(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	env := r.URL.Query().Get("env")
	if env == "" {
		env = "prod"
	}

	var tenantID *uuid.UUID
	if tid := r.URL.Query().Get("tenant_id"); tid != "" {
		if id, err := uuid.Parse(tid); err == nil {
			tenantID = &id
		}
	}

	signals, err := h.anomalyService.GetOpenSignals(ctx, env, tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(signals)
}

// GetDriftSignal returns a specific drift signal
func (h *ASOEnhancedHandler) GetDriftSignal(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid signal ID", http.StatusBadRequest)
		return
	}

	history, err := h.anomalyService.GetSignalHistory(ctx, id, 1)
	if err != nil || len(history) == 0 {
		http.Error(w, "Signal not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history[0])
}

// AcknowledgeDrift acknowledges a drift signal
func (h *ASOEnhancedHandler) AcknowledgeDrift(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid signal ID", http.StatusBadRequest)
		return
	}

	actor := r.Header.Get("X-User-ID")
	if actor == "" {
		actor = "api_user"
	}

	if err := h.anomalyService.AcknowledgeSignal(ctx, id, actor); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ResolveDrift resolves a drift signal
func (h *ASOEnhancedHandler) ResolveDrift(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid signal ID", http.StatusBadRequest)
		return
	}

	actor := r.Header.Get("X-User-ID")
	if actor == "" {
		actor = "api_user"
	}

	if err := h.anomalyService.ResolveSignal(ctx, id, actor, false); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// TriggerAnomalyScan triggers a full anomaly scan
func (h *ASOEnhancedHandler) TriggerAnomalyScan(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	env := r.URL.Query().Get("env")
	if env == "" {
		env = "prod"
	}

	signals, err := h.anomalyService.ScanForAnomalies(ctx, env)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"signals_detected": len(signals),
		"signals":          signals,
	})
}

// ============================================================================
// Self-Healing Endpoints
// ============================================================================

// GetHealingActions returns pending/recent healing actions
func (h *ASOEnhancedHandler) GetHealingActions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	env := r.URL.Query().Get("env")
	if env == "" {
		env = "prod"
	}

	actions, err := h.healingService.GetPendingActions(ctx, env)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(actions)
}

// GetHealingHistory returns healing history for a target
func (h *ASOEnhancedHandler) GetHealingHistory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	targetIDStr := chi.URLParam(r, "targetId")

	targetID, err := uuid.Parse(targetIDStr)
	if err != nil {
		http.Error(w, "Invalid target ID", http.StatusBadRequest)
		return
	}

	actions, err := h.healingService.GetHealingHistory(ctx, targetID, 20)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(actions)
}

// TriggerHealing manually triggers self-healing for a drift signal
func (h *ASOEnhancedHandler) TriggerHealing(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	signalIDStr := chi.URLParam(r, "signalId")

	signalID, err := uuid.Parse(signalIDStr)
	if err != nil {
		http.Error(w, "Invalid signal ID", http.StatusBadRequest)
		return
	}

	// Get the signal
	history, err := h.anomalyService.GetSignalHistory(ctx, signalID, 1)
	if err != nil || len(history) == 0 {
		http.Error(w, "Signal not found", http.StatusNotFound)
		return
	}

	signal := &history[0]

	action, err := h.healingService.ProcessDriftSignal(ctx, signal)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(action)
}
