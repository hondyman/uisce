package finops

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/hondyman/uisce/backend/internal/identity"
	"github.com/hondyman/uisce/backend/internal/security"
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
	clock           Clock
}

// NewForecastHandler constructs a ForecastHandler wired to the given DB and the
// production RealClock.
func NewForecastHandler(db *sqlx.DB) *ForecastHandler {
	return NewForecastHandlerWithClock(db, RealClock{})
}

// NewForecastHandlerWithClock constructs a ForecastHandler using the supplied clock
// (shared with the inner forecaster and coordinator). Used by tests and by callers
// that need to pin "today" to a deterministic date.
func NewForecastHandlerWithClock(db *sqlx.DB, clock Clock) *ForecastHandler {
	if clock == nil {
		clock = RealClock{}
	}
	return &ForecastHandler{
		forecaster:      NewDemandForecasterWithClock(db, clock),
		coordinator:     NewPrewarmCoordinatorWithClock(db, clock),
		policyService:   NewSmoothingPolicyService(db),
		feedbackService: NewForecastFeedbackService(db),
		clock:           clock,
	}
}

// RegisterRoutes mounts the FinOps forecast endpoints onto a standard http.ServeMux (Go 1.22+).
func (h *ForecastHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/finops/forecast/today", h.handleForecastToday)
	mux.HandleFunc("POST /api/finops/prewarm/trigger", h.handlePrewarmTrigger)
	mux.HandleFunc("GET /api/finops/prewarm/status", h.handleGetPrewarmStatus)
	mux.HandleFunc("GET /api/finops/smoothing-policy", h.handleGetPolicy)
	mux.HandleFunc("PUT /api/finops/smoothing-policy", h.handleUpsertPolicy)
	mux.HandleFunc("POST /api/finops/forecast/{forecastId}/feedback", h.handleSubmitFeedback)
	mux.HandleFunc("GET /api/finops/forecast/{forecastId}/feedback", h.handleGetFeedback)
	mux.HandleFunc("GET /api/finops/calibration", h.handleGetCalibration)
}

// RegisterChiRoutes mounts the FinOps forecast endpoints onto a chi.Router.
// The prewarm startup sweep is owned by the composition root (cmd/server/main.go),
// not by route registration — calling this method has no side effects beyond wiring routes.
func (h *ForecastHandler) RegisterChiRoutes(r chi.Router) {
	r.Route("/finops", func(sub chi.Router) {
		sub.Get("/forecast/today", h.handleForecastToday)
		sub.Post("/prewarm/trigger", h.handlePrewarmTrigger)
		sub.Get("/prewarm/status", h.handleGetPrewarmStatus)
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

	forecast, err := h.forecaster.GenerateTenantDemandForecast(r.Context(), tenantID, h.clock.Now().UTC())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "forecast_failed", err.Error())
		return
	}

	// Best-effort persistence — do not fail the response if the write fails.
	_ = h.forecaster.PersistForecast(r.Context(), forecast)

	writeJSON(w, http.StatusOK, forecast)
}

// handlePrewarmTrigger manually triggers the off-peak pre-warming workflow for
// the authenticated tenant. Enforces administrator / FinOps manager authorization.
// Rejects duplicate concurrent executions with 409 Conflict.
// Writes a synchronous PENDING ledger row before returning 202 + jobId.
func (h *ForecastHandler) handlePrewarmTrigger(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := extractTenantID(w, r)
	if !ok {
		return
	}

	claims := jwtmiddleware.GetClaimsFromContext(r)
	var roles []string
	isCoreAdmin := false
	var userIDStr string

	if claims != nil {
		roles = claims.Roles
		isCoreAdmin = claims.IsCoreAdmin
		userIDStr = claims.UserID
	} else if authInfo, ok := security.AuthInfoFromContext(r.Context()); ok {
		roles = authInfo.Roles
		isCoreAdmin = authInfo.IsGlobalAdmin
		userIDStr = authInfo.UserID
	}

	if !isCoreAdmin &&
		!slices.Contains(roles, "admin") &&
		!slices.Contains(roles, "finops_manager") &&
		!slices.Contains(roles, "finops_admin") {
		writeError(w, http.StatusForbidden, "forbidden", "only administrators or finops managers can trigger prewarming")
		return
	}

	// In-flight deduplication: acquire tenant lock before issuing 202
	if !h.coordinator.TryLockTenant(tenantID) {
		writeError(w, http.StatusConflict, "already_in_flight", "a pre-warm execution is already in flight for this tenant")
		return
	}

	var triggeredBy *uuid.UUID
	if userIDStr != "" {
		if uID, err := uuid.Parse(userIDStr); err == nil {
			triggeredBy = &uID
		}
	}

	jobID := uuid.New()

	// Synchronously persist a PENDING row in the execution ledger.
	// This acts as a persistent crash marker and ensures subsequent status polls match this specific job.
	if err := h.coordinator.CreatePendingExecution(r.Context(), tenantID, jobID, triggeredBy); err != nil {
		h.coordinator.UnlockTenant(tenantID)
		writeError(w, http.StatusInternalServerError, "ledger_init_failed", "failed to record prewarm job in execution ledger")
		return
	}

	// Run off-peak prewarming asynchronously with a detached context
	go func(tID uuid.UUID, jID uuid.UUID) {
		defer h.coordinator.UnlockTenant(tID)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		_, _ = h.coordinator.ExecuteOffPeakPrewarming(ctx, tID, &jID)
	}(tenantID, jobID)

	writeJSON(w, http.StatusAccepted, map[string]any{
		"jobId":    jobID.String(),
		"status":   "PENDING",
		"message":  "Off-peak prewarming dispatched asynchronously",
		"tenantId": tenantID,
	})
}

// handleGetPrewarmStatus retrieves the prewarm execution state from the ledger.
// If ?jobId={uuid} is specified, queries specifically for that job. Otherwise returns the latest run.
func (h *ForecastHandler) handleGetPrewarmStatus(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := extractTenantID(w, r)
	if !ok {
		return
	}

	var result *PrewarmResult
	var err error

	jobIDStr := r.URL.Query().Get("jobId")
	if jobIDStr != "" {
		if jID, parseErr := uuid.Parse(jobIDStr); parseErr == nil {
			result, err = h.coordinator.GetPrewarmExecutionByJobID(r.Context(), tenantID, jID)
		}
	}

	if result == nil && err == nil {
		result, err = h.coordinator.GetLatestPrewarmExecution(r.Context(), tenantID)
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, "status_fetch_failed", err.Error())
		return
	}
	if result == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"tenantId":  tenantID,
			"triggered": false,
			"status":    "IDLE",
		})
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
// Enforces that only administrators or FinOps managers may modify policy.
func (h *ForecastHandler) handleUpsertPolicy(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := extractTenantID(w, r)
	if !ok {
		return
	}

	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil || (!claims.IsCoreAdmin &&
		!slices.Contains(claims.Roles, "admin") &&
		!slices.Contains(claims.Roles, "finops_manager") &&
		!slices.Contains(claims.Roles, "finops_admin")) {
		writeError(w, http.StatusForbidden, "forbidden", "only administrators or finops managers can update smoothing policy")
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

// extractTenantID pulls the tenant UUID from the JWT context set by the JWT middleware,
// with fallbacks to identity context and X-Tenant-ID header.
// Writes an HTTP error and returns false on failure.
func extractTenantID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	raw := jwtmiddleware.GetTenantIDFromContext(r)
	if raw == "" {
		if tID, ok := identity.TenantIDFromContext(r.Context()); ok && tID != "" {
			raw = tID
		}
	}
	if raw == "" {
		raw = r.Header.Get("X-Tenant-ID")
	}
	if raw == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing or invalid JWT claims")
		return uuid.Nil, false
	}

	tenantID, err := uuid.Parse(raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_tenant", "tenant_id is not a valid UUID")
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
