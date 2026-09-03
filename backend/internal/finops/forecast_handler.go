package finops

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
)

// ForecastHandler exposes Predictive FinOps endpoints.
//
// Routes (registered by RegisterRoutes):
//
//	GET  /api/finops/forecast/today                — Generate & persist today's demand forecast
//	POST /api/finops/prewarm/trigger               — Manually trigger off-peak pre-warming
//	GET  /api/finops/smoothing-policy              — Fetch active smoothing policy
//	PUT  /api/finops/smoothing-policy              — Create or update smoothing policy
//	POST /api/finops/forecast/{forecastId}/feedback — Submit outcome feedback for a forecast
//	GET  /api/finops/forecast/{forecastId}/feedback — Retrieve feedback for a forecast
//	GET  /api/finops/calibration                   — Get current calibration state for tenant
type ForecastHandler struct {
	forecaster      *DemandForecaster
	coordinator     *PrewarmCoordinator
	policyService   *SmoothingPolicyService
	feedbackService *ForecastFeedbackService
}

// NewForecastHandler constructs a ForecastHandler wired to the given DB.
func NewForecastHandler(db *sqlx.DB) *ForecastHandler {
	return &ForecastHandler{
		forecaster:      NewDemandForecaster(db),
		coordinator:     NewPrewarmCoordinator(db),
		policyService:   NewSmoothingPolicyService(db),
		feedbackService: NewForecastFeedbackService(db),
	}
}

// RegisterRoutes mounts the FinOps forecast endpoints onto a standard http.ServeMux (Go 1.22+).
func (h *ForecastHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/finops/forecast/today", h.handleForecastToday)
	mux.HandleFunc("POST /api/finops/prewarm/trigger", h.handlePrewarmTrigger)
	mux.HandleFunc("GET /api/finops/smoothing-policy", h.handleGetPolicy)
	mux.HandleFunc("PUT /api/finops/smoothing-policy", h.handleUpsertPolicy)
	mux.HandleFunc("POST /api/finops/forecast/{forecastId}/feedback", h.handleSubmitFeedback)
	mux.HandleFunc("GET /api/finops/forecast/{forecastId}/feedback", h.handleGetFeedback)
	mux.HandleFunc("GET /api/finops/calibration", h.handleGetCalibration)
}

// RegisterChiRoutes mounts the FinOps forecast endpoints onto a chi.Router.
func (h *ForecastHandler) RegisterChiRoutes(r chi.Router) {
	r.Route("/finops", func(sub chi.Router) {
		sub.Get("/forecast/today", h.handleForecastToday)
		sub.Post("/prewarm/trigger", h.handlePrewarmTrigger)
		sub.Get("/smoothing-policy", h.handleGetPolicy)
		sub.Put("/smoothing-policy", h.handleUpsertPolicy)
		sub.Post("/forecast/{forecastId}/feedback", h.handleSubmitFeedback)
		sub.Get("/forecast/{forecastId}/feedback", h.handleGetFeedback)
		sub.Get("/calibration", h.handleGetCalibration)
	})
}

// ── Handlers ──────────────────────────────────────────────────────────────────

// handleForecastToday generates (and persists) today's demand forecast for the
// authenticated tenant.
func (h *ForecastHandler) handleForecastToday(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := extractTenantID(w, r)
	if !ok {
		return
	}

	forecast, err := h.forecaster.GenerateTenantDemandForecast(r.Context(), tenantID, time.Now().UTC())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "forecast_failed", err.Error())
		return
	}

	// Best-effort persistence — do not fail the response if the write fails.
	_ = h.forecaster.PersistForecast(r.Context(), forecast)

	writeJSON(w, http.StatusOK, forecast)
}

// handlePrewarmTrigger manually triggers the off-peak pre-warming workflow for
// the authenticated tenant. Intended for testing and manual override.
func (h *ForecastHandler) handlePrewarmTrigger(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := extractTenantID(w, r)
	if !ok {
		return
	}

	result, err := h.coordinator.ExecuteOffPeakPrewarming(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "prewarm_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// handleGetPolicy returns the active smoothing policy for the authenticated tenant.
func (h *ForecastHandler) handleGetPolicy(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := extractTenantID(w, r)
	if !ok {
		return
	}

	policy, err := h.policyService.GetActivePolicy(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "policy_fetch_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, policy)
}

// handleUpsertPolicy creates or updates the smoothing policy for the authenticated tenant.
func (h *ForecastHandler) handleUpsertPolicy(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := extractTenantID(w, r)
	if !ok {
		return
	}

	var input WorkloadSmoothingPolicy
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	input.TenantID = tenantID // always enforce tenant from JWT, not body

	saved, err := h.policyService.UpsertPolicy(r.Context(), &input)
	if err != nil {
		writeError(w, http.StatusBadRequest, "policy_upsert_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, saved)
}

// handleSubmitFeedback records the operator outcome for a completed forecast.
// POST /api/finops/forecast/{forecastId}/feedback
//
// Body: { "outcome": "ACCURATE"|"FALSE_POSITIVE"|"MISSED_SPIKE"|"PARTIAL_SPIKE",
//         "actualCostUsd": 123.45, "actualScannedBytes": 900000000,
//         "actualCpuDurationMs": 140000, "notes": "...", "recordedBy": "<uuid>" }
func (h *ForecastHandler) handleSubmitFeedback(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := extractTenantID(w, r)
	if !ok {
		return
	}

	forecastIDStr := getURLParam(r, "forecastId")
	forecastID, err := uuid.Parse(forecastIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_forecast_id", "forecastId must be a valid UUID")
		return
	}

	var req SubmitFeedbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	req.ForecastID = forecastID // path param wins over body

	fb, err := h.feedbackService.SubmitFeedback(r.Context(), tenantID, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "feedback_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, fb)
}

// handleGetFeedback retrieves the feedback record for a specific forecast.
// GET /api/finops/forecast/{forecastId}/feedback
func (h *ForecastHandler) handleGetFeedback(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := extractTenantID(w, r)
	if !ok {
		return
	}

	forecastIDStr := getURLParam(r, "forecastId")
	forecastID, err := uuid.Parse(forecastIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_forecast_id", "forecastId must be a valid UUID")
		return
	}

	fb, err := h.feedbackService.GetFeedback(r.Context(), tenantID, forecastID)
	if err != nil || fb == nil {
		writeError(w, http.StatusNotFound, "not_found", "no feedback recorded for this forecast")
		return
	}

	writeJSON(w, http.StatusOK, fb)
}

// handleGetCalibration returns the current calibration state for the authenticated tenant.
// GET /api/finops/calibration
func (h *ForecastHandler) handleGetCalibration(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := extractTenantID(w, r)
	if !ok {
		return
	}

	state, err := h.feedbackService.GetCalibrationState(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "calibration_fetch_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, state)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// extractTenantID pulls the tenant UUID from the JWT context set by the JWT middleware.
// Writes an HTTP error and returns false on failure.
func extractTenantID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	raw := jwtmiddleware.GetTenantIDFromContext(r)
	if raw == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing or invalid JWT claims")
		return uuid.Nil, false
	}

	tenantID, err := uuid.Parse(raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_tenant", "tenant_id in JWT is not a valid UUID")
		return uuid.Nil, false
	}

	return tenantID, true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{"error": code, "message": message})
}

func getURLParam(r *http.Request, key string) string {
	if val := chi.URLParam(r, key); val != "" {
		return val
	}
	return r.PathValue(key)
}
