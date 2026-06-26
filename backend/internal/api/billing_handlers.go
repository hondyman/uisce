package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/billing"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// BillingHandlers provides HTTP handlers for the platform billing API.
type BillingHandlers struct {
	svc *billing.PlatformBillingService
}

// NewBillingHandlers creates new billing HTTP handlers.
func NewBillingHandlers(svc *billing.PlatformBillingService) *BillingHandlers {
	return &BillingHandlers{svc: svc}
}

// RegisterRoutes mounts all billing routes on the chi router.
func (h *BillingHandlers) RegisterRoutes(r chi.Router) {
	r.Route("/billing", func(r chi.Router) {
		// ── Tenant Billing ───────────────────────────────
		// GET /api/billing/tenant/{tenantId}?window=30d
		r.Get("/tenant/{tenantId}", h.GetTenantBilling)

		// ── Platform Billing (admin only) ────────────────
		// GET /api/billing/platform?window=30d
		r.Get("/platform", h.GetPlatformBilling)

		// ── Anomaly Detection ────────────────────────────
		// GET /api/billing/anomalies
		r.Get("/anomalies", h.GetAnomalies)

		// ── Cost Forecasting ─────────────────────────────
		// GET /api/billing/forecast
		r.Get("/forecast", h.GetForecast)

		// ── Cost Simulator ───────────────────────────────
		// POST /api/billing/simulate
		r.Post("/simulate", h.SimulateCost)

		// ── Per-Table Cost Attribution ────────────────────
		// GET /api/billing/tables?window=30d
		r.Get("/tables", h.GetTableCosts)

		// ── Invoice Generation ───────────────────────────
		// GET /api/billing/invoice/{tenantId}?month=2026-01
		r.Get("/invoice/{tenantId}", h.GenerateInvoice)
	})
}

// ─── Tenant Billing ──────────────────────────────────────────────

func (h *BillingHandlers) GetTenantBilling(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	if tenantID == "" {
		// Fall back to header for multi-tenant auth
		tenantID = jwtmiddleware.GetClaimsFromContext(r).TenantID
	}
	if tenantID == "" {
		http.Error(w, "tenantId is required", http.StatusBadRequest)
		return
	}

	window := r.URL.Query().Get("window")
	if window == "" {
		window = "30d"
	}

	resp, err := h.svc.GetTenantBilling(r.Context(), tenantID, window)
	if err != nil {
		jsonError(w, fmt.Sprintf("failed to get tenant billing: %v", err), http.StatusInternalServerError)
		return
	}

	jsonOK(w, resp)
}

// ─── Platform Billing ────────────────────────────────────────────

func (h *BillingHandlers) GetPlatformBilling(w http.ResponseWriter, r *http.Request) {
	window := r.URL.Query().Get("window")
	if window == "" {
		window = "30d"
	}

	resp, err := h.svc.GetPlatformBilling(r.Context(), window)
	if err != nil {
		jsonError(w, fmt.Sprintf("failed to get platform billing: %v", err), http.StatusInternalServerError)
		return
	}

	jsonOK(w, resp)
}

// ─── Anomaly Detection ──────────────────────────────────────────

func (h *BillingHandlers) GetAnomalies(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.DetectAnomalies(r.Context())
	if err != nil {
		jsonError(w, fmt.Sprintf("failed to detect anomalies: %v", err), http.StatusInternalServerError)
		return
	}

	jsonOK(w, resp)
}

// ─── Forecasting ────────────────────────────────────────────────

func (h *BillingHandlers) GetForecast(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.ForecastCost(r.Context())
	if err != nil {
		jsonError(w, fmt.Sprintf("failed to forecast: %v", err), http.StatusInternalServerError)
		return
	}

	jsonOK(w, resp)
}

// ─── Cost Simulator ─────────────────────────────────────────────

func (h *BillingHandlers) SimulateCost(w http.ResponseWriter, r *http.Request) {
	var req billing.CostSimulationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	resp := h.svc.SimulateCost(req)
	jsonOK(w, resp)
}

// ─── Per-Table Costs ────────────────────────────────────────────

func (h *BillingHandlers) GetTableCosts(w http.ResponseWriter, r *http.Request) {
	window := r.URL.Query().Get("window")
	if window == "" {
		window = "30d"
	}

	resp, err := h.svc.GetTableCosts(r.Context(), window)
	if err != nil {
		jsonError(w, fmt.Sprintf("failed to get table costs: %v", err), http.StatusInternalServerError)
		return
	}

	jsonOK(w, resp)
}

// ─── Invoice ────────────────────────────────────────────────────

func (h *BillingHandlers) GenerateInvoice(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	if tenantID == "" {
		http.Error(w, "tenantId is required", http.StatusBadRequest)
		return
	}

	month := r.URL.Query().Get("month")
	if month == "" {
		month = "current"
	}

	resp, err := h.svc.GenerateInvoice(r.Context(), tenantID, month)
	if err != nil {
		jsonError(w, fmt.Sprintf("failed to generate invoice: %v", err), http.StatusInternalServerError)
		return
	}

	jsonOK(w, resp)
}

// ─── JSON helpers ───────────────────────────────────────────────

func jsonOK(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
