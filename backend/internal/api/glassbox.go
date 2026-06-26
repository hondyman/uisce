package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/api/middleware"
	"github.com/hondyman/semlayer/backend/pkg/multitenancy"
	"go.temporal.io/sdk/client"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

type GlassBoxHandler struct {
	tm             *multitenancy.TenantManager
	temporalClient client.Client
}

func NewGlassBoxHandler(tm *multitenancy.TenantManager, tc client.Client) *GlassBoxHandler {
	return &GlassBoxHandler{tm: tm, temporalClient: tc}
}

func (h *GlassBoxHandler) RegisterRoutes(r chi.Router) {
	// Advisor Routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireRole(middleware.RoleAdvisor, middleware.RoleCompliance))
		r.Get("/audit/events", h.GetEvents)
		r.Post("/approvals/{workflowID}/signal", h.SignalApproval)
		r.Post("/exports/run/{runID}", h.GenerateExportBundle)
		r.Post("/replay/run/{runID}", h.TriggerReplay)
	})

	// Compliance Routes (Protected)
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireRole(middleware.RoleCompliance))
		r.Get("/compliance/summary", h.GetComplianceSummary)
		r.Get("/compliance/sec-report", h.GetSECReport)
	})
}

// GetComplianceSummary returns the daily compliance metrics
func (h *GlassBoxHandler) GetComplianceSummary(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "Missing X-Tenant-ID header", http.StatusBadRequest)
		return
	}
	db, err := h.tm.GetDB(tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	query := "SELECT * FROM compliance_summary ORDER BY date DESC LIMIT 30"
	rows, err := db.QueryxContext(r.Context(), query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var summary []map[string]interface{}
	for rows.Next() {
		row := make(map[string]interface{})
		_ = rows.MapScan(row)
		summary = append(summary, row)
	}
	json.NewEncoder(w).Encode(summary)
}

// GetSECReport generates the regulatory report
func (h *GlassBoxHandler) GetSECReport(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	db, err := h.tm.GetDB(tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Default to last 30 days
	start := time.Now().AddDate(0, 0, -30)
	end := time.Now()

	query := "SELECT * FROM generate_sec_report($1, $2)"
	rows, err := db.QueryxContext(r.Context(), query, start, end)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var report []map[string]interface{}
	for rows.Next() {
		row := make(map[string]interface{})
		_ = rows.MapScan(row)
		report = append(report, row)
	}
	json.NewEncoder(w).Encode(report)
}

// GetEvents returns the immutable audit log
func (h *GlassBoxHandler) GetEvents(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	db, err := h.tm.GetDB(tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// In a real app, pagination and filtering would be here
	query := `
		SELECT event_id, run_id, seq, event_type, payload_canon, payload_hash, parent_hash, timestamp 
		FROM events_raw 
		ORDER BY timestamp DESC 
		LIMIT 100
	`
	rows, err := db.QueryxContext(r.Context(), query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var events []map[string]interface{}
	for rows.Next() {
		evt := make(map[string]interface{})
		err := rows.MapScan(evt)
		if err != nil {
			continue
		}
		events = append(events, evt)
	}

	json.NewEncoder(w).Encode(events)
}

type ApprovalSignal struct {
	Action  string `json:"action"`
	Comment string `json:"comment"`
	ActorID string `json:"actor_id"`
}

// SignalApproval sends a signal to a running workflow
func (h *GlassBoxHandler) SignalApproval(w http.ResponseWriter, r *http.Request) {
	workflowID := chi.URLParam(r, "workflowID")
	var sig ApprovalSignal
	if err := json.NewDecoder(r.Body).Decode(&sig); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Send signal to Temporal
	err := h.temporalClient.SignalWorkflow(r.Context(), workflowID, "", "advisor-signal", sig)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Record decision in structured table
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	db, err := h.tm.GetDB(tenantID)
	if err == nil {
		_, _ = db.ExecContext(r.Context(),
			"INSERT INTO decisions (run_id, outcome, actor_id, comment) VALUES ($1, $2, $3, $4)",
			workflowID, sig.Action, sig.ActorID, sig.Comment)
	}
	if err != nil {
		// Log error but don't fail request since signal went through
	}

	w.WriteHeader(http.StatusOK)
}

// GenerateExportBundle creates a signed audit pack
func (h *GlassBoxHandler) GenerateExportBundle(w http.ResponseWriter, r *http.Request) {
	runID := chi.URLParam(r, "runID")
	format := r.URL.Query().Get("format")

	// Mock response for now
	bundle := map[string]interface{}{
		"run_id":    runID,
		"format":    format,
		"status":    "generated",
		"url":       "/download/bundles/" + runID + ".zip",
		"hash":      "sha256:mock-hash",
		"signature": "hmac:mock-signature",
	}
	json.NewEncoder(w).Encode(bundle)
}

// TriggerReplay starts a deterministic replay
func (h *GlassBoxHandler) TriggerReplay(w http.ResponseWriter, r *http.Request) {
	runID := chi.URLParam(r, "runID")

	// Mock response
	result := map[string]interface{}{
		"original_run_id":   runID,
		"replay_run_id":     "replay-" + runID,
		"status":            "started",
		"determinism_check": "pending",
	}
	json.NewEncoder(w).Encode(result)
}
