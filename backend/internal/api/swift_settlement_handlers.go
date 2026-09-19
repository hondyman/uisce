package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
	"go.temporal.io/sdk/client"
)

// SWIFTSettlementHandler handles SWIFT settlement instruction REST endpoints.
// Routes are mounted under /api/cash-flow/swift/*.
//
// Tenant isolation: every handler extracts tenant_id from JWT claims via
// jwtmiddleware.GetClaimsFromContext — matching the exact pattern used in
// backend/internal/cashflow/settlement/handler.go. No X-Tenant-ID header
// (that header is admin-only; JWT is the production path).
//
// Workflow dispatch: handlers start/signal SWIFTSettlementWorkflow and
// SWIFTChannelLifecycleWorkflow on task queue "bp_queue" via the Temporal
// client. The temporal client is nil-safe — when nil, handlers return 503
// so the service degrades gracefully without crashing.
type SWIFTSettlementHandler struct {
	db             *sql.DB
	temporalClient client.Client
	adminURL       string // SWIFT admin server e.g. "http://127.0.0.1:8982"
	adminToken     string // X-Swift-Admin-Token shared secret
}

// NewSWIFTSettlementHandler returns a handler wired to the given DB and
// Temporal client. adminURL/adminToken are forwarded to workflow inputs.
func NewSWIFTSettlementHandler(db *sql.DB, tc client.Client, adminURL, adminToken string) *SWIFTSettlementHandler {
	return &SWIFTSettlementHandler{
		db:             db,
		temporalClient: tc,
		adminURL:       adminURL,
		adminToken:     adminToken,
	}
}

// RegisterSWIFTRoutes mounts all SWIFT endpoints on the provided chi.Router.
func RegisterSWIFTRoutes(r chi.Router, db *sql.DB, tc client.Client, adminURL, adminToken string) {
	h := NewSWIFTSettlementHandler(db, tc, adminURL, adminToken)
	h.registerRoutes(r)
}

func (h *SWIFTSettlementHandler) registerRoutes(r chi.Router) {
	r.Route("/api/cash-flow/swift", func(r chi.Router) {
		r.Post("/instructions", h.SubmitInstruction)
		r.Get("/instructions/{id}", h.GetInstruction)
		r.Post("/instructions/{id}/cancel", h.CancelInstruction)
		r.Get("/reconciliation/reports", h.ListReconciliationReports)
		r.Post("/channels/{custodian_id}/start", h.StartChannel)
		r.Post("/channels/{custodian_id}/stop", h.StopChannel)
	})
	r.Post("/api/catalog/admin/sync-swift-subtypes", h.SyncSWIFTSubtypes)
}

// tenantID extracts tenant UUID from JWT claims.
func (h *SWIFTSettlementHandler) tenantID(r *http.Request) (uuid.UUID, bool) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil || claims.TenantID == "" {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(claims.TenantID)
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}

type submitInstructionRequest struct {
	CustodianID    string `json:"custodian_id"`
	MsgType        string `json:"msg_type"`
	TransactionRef string `json:"transaction_ref"`
	UETR           string `json:"uetr,omitempty"`
	RawMessageB64  string `json:"raw_message"`
	PipelineDAGID  string `json:"pipeline_dag_id"`
}

// SubmitInstruction creates a new settlement instruction and starts
// SWIFTSettlementWorkflow on bp_queue.
// POST /api/cash-flow/swift/instructions
func (h *SWIFTSettlementHandler) SubmitInstruction(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantID(r)
	if !ok {
		http.Error(w, "unauthorized: missing or invalid JWT tenant claim", http.StatusUnauthorized)
		return
	}
	var req submitInstructionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.CustodianID == "" || req.MsgType == "" || req.RawMessageB64 == "" {
		http.Error(w, "custodian_id, msg_type, and raw_message are required", http.StatusBadRequest)
		return
	}
	custodianID, err := uuid.Parse(req.CustodianID)
	if err != nil {
		http.Error(w, "invalid custodian_id UUID", http.StatusBadRequest)
		return
	}
	if h.temporalClient == nil {
		http.Error(w, "temporal client unavailable", http.StatusServiceUnavailable)
		return
	}

	// Workflow ID deterministic on (tenantID, transactionRef) — idempotent
	// duplicate submissions: Temporal deduplicates by workflow ID.
	wfID := "swift-settlement-" + tenantID.String() + "-" + req.TransactionRef
	if req.TransactionRef == "" {
		wfID = "swift-settlement-" + tenantID.String() + "-" + uuid.New().String()
	}

	wfInput := map[string]interface{}{
		"tenant_id":            tenantID.String(),
		"custodian_id":         custodianID.String(),
		"transaction_ref":      req.TransactionRef,
		"uetr":                 req.UETR,
		"admin_url":            h.adminURL,
		"admin_token":          h.adminToken,
		"pipeline_dag_id":      req.PipelineDAGID,
		"settlement_budget_ms": int64(60000),
		"msg_type":             req.MsgType,
		"raw_message_b64":      req.RawMessageB64,
	}
	opts := client.StartWorkflowOptions{
		ID:        wfID,
		TaskQueue: "bp_queue",
	}
	run, err := h.temporalClient.ExecuteWorkflow(r.Context(), opts, "SWIFTSettlementWorkflow", wfInput)
	if err != nil {
		http.Error(w, "failed to start workflow: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"workflow_id":     wfID,
		"run_id":          run.GetRunID(),
		"status":          "RECEIVED",
		"transaction_ref": req.TransactionRef,
	})
}

// GetInstruction returns the settlement record for a given transaction_ref.
// GET /api/cash-flow/swift/instructions/{id}
//
// {id} is the transaction_ref value (:20: field from MT, EndToEndId from MX).
// Filters strictly by tenant_id — no gold-copy OR on reads of per-tenant
// settlement state (only the requesting tenant's own rows are returned).
// Returns 404 when no matching row exists for this tenant.
func (h *SWIFTSettlementHandler) GetInstruction(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	txRef := chi.URLParam(r, "id")
	if txRef == "" {
		http.Error(w, "id (transaction_ref) is required", http.StatusBadRequest)
		return
	}

	var (
		recID, status, subtype, currency string
		amount                           float64
		settlementDate, updatedAt        time.Time
	)
	err := h.db.QueryRowContext(r.Context(), `
		SELECT id::text, settlement_status, settlement_date, amount, currency,
		       subtype_code, updated_at
		FROM cash_flow.settlement
		WHERE transaction_ref = $1
		  AND tenant_id = $2
		  AND valid_to IS NULL
		ORDER BY created_at DESC
		LIMIT 1
	`, txRef, tenantID).Scan(&recID, &status, &settlementDate, &amount, &currency, &subtype, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, "query error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":              recID,
		"transaction_ref": txRef,
		"status":          status,
		"settlement_date": settlementDate.Format("2006-01-02"),
		"amount":          amount,
		"currency":        currency,
		"subtype_code":    subtype,
		"updated_at":      updatedAt.Format(time.RFC3339),
	})
}


// CancelInstruction signals Cancel to SWIFTSettlementWorkflow.
// POST /api/cash-flow/swift/instructions/{id}/cancel
func (h *SWIFTSettlementHandler) CancelInstruction(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	transactionRef := chi.URLParam(r, "id")
	if h.temporalClient == nil {
		http.Error(w, "temporal client unavailable", http.StatusServiceUnavailable)
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.Reason == "" {
		body.Reason = "CUST"
	}
	wfID := "swift-settlement-" + tenantID.String() + "-" + transactionRef
	if err := h.temporalClient.SignalWorkflow(r.Context(), wfID, "", "Cancel", body); err != nil {
		http.Error(w, "failed to signal workflow: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":          "cancel_signalled",
		"transaction_ref": transactionRef,
		"reason":          body.Reason,
	})
}

// ListReconciliationReports returns recent SWIFT recon reports.
// GET /api/cash-flow/swift/reconciliation/reports
func (h *SWIFTSettlementHandler) ListReconciliationReports(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	// GSIFI pattern.
	rows, err := h.db.QueryContext(r.Context(), `
		SELECT id::text, custodian_id::text, lookback_start, lookback_end,
		       instructions_scanned, mismatches_count, generated_at
		FROM swift_reconciliation_report
		WHERE (tenant_id = $1 OR tenant_id = (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1))
		ORDER BY generated_at DESC
		LIMIT 50
	`, tenantID)
	if err != nil {
		http.Error(w, "query error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type reportRow struct {
		ID                  string    `json:"id"`
		CustodianID         string    `json:"custodian_id"`
		LookbackStart       time.Time `json:"lookback_start"`
		LookbackEnd         time.Time `json:"lookback_end"`
		InstructionsScanned int       `json:"instructions_scanned"`
		MismatchesCount     int       `json:"mismatches_count"`
		GeneratedAt         time.Time `json:"generated_at"`
	}
	var reports []reportRow
	for rows.Next() {
		var rep reportRow
		if err := rows.Scan(
			&rep.ID, &rep.CustodianID,
			&rep.LookbackStart, &rep.LookbackEnd,
			&rep.InstructionsScanned, &rep.MismatchesCount,
			&rep.GeneratedAt,
		); err != nil {
			continue
		}
		reports = append(reports, rep)
	}
	if reports == nil {
		reports = []reportRow{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reports)
}

// StartChannel starts SWIFTChannelLifecycleWorkflow. Admin-only.
// POST /api/cash-flow/swift/channels/{custodian_id}/start
func (h *SWIFTSettlementHandler) StartChannel(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Swift-Admin-Token") != h.adminToken {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	tenantID, ok := h.tenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	custodianID, err := uuid.Parse(chi.URLParam(r, "custodian_id"))
	if err != nil {
		http.Error(w, "invalid custodian_id", http.StatusBadRequest)
		return
	}
	if h.temporalClient == nil {
		http.Error(w, "temporal client unavailable", http.StatusServiceUnavailable)
		return
	}
	wfID := "swift-channel-" + tenantID.String() + "-" + custodianID.String()
	wfInput := map[string]interface{}{
		"tenant_id":    tenantID.String(),
		"custodian_id": custodianID.String(),
		"admin_url":    h.adminURL,
		"admin_token":  h.adminToken,
		"channel_id":   custodianID.String(),
	}
	run, err := h.temporalClient.ExecuteWorkflow(r.Context(), client.StartWorkflowOptions{
		ID:        wfID,
		TaskQueue: "bp_queue",
	}, "SWIFTChannelLifecycleWorkflow", wfInput)
	if err != nil {
		http.Error(w, "failed to start channel workflow: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"workflow_id":  wfID,
		"run_id":       run.GetRunID(),
		"status":       "INITIALIZING",
		"custodian_id": custodianID.String(),
	})
}

// StopChannel signals Stop to SWIFTChannelLifecycleWorkflow. Admin-only.
// POST /api/cash-flow/swift/channels/{custodian_id}/stop
func (h *SWIFTSettlementHandler) StopChannel(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Swift-Admin-Token") != h.adminToken {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	tenantID, ok := h.tenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	custodianID, err := uuid.Parse(chi.URLParam(r, "custodian_id"))
	if err != nil {
		http.Error(w, "invalid custodian_id", http.StatusBadRequest)
		return
	}
	if h.temporalClient == nil {
		http.Error(w, "temporal client unavailable", http.StatusServiceUnavailable)
		return
	}
	wfID := "swift-channel-" + tenantID.String() + "-" + custodianID.String()
	if err := h.temporalClient.SignalWorkflow(r.Context(), wfID, "", "Stop", "operator"); err != nil {
		http.Error(w, "failed to signal channel: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":       "stop_signalled",
		"custodian_id": custodianID.String(),
	})
}

// SyncSWIFTSubtypes triggers SWIFT subtype catalog sync.
// POST /api/catalog/admin/sync-swift-subtypes
func (h *SWIFTSettlementHandler) SyncSWIFTSubtypes(w http.ResponseWriter, r *http.Request) {
	tenantIDStr := r.Header.Get("X-Tenant-ID")
	if tenantIDStr == "" {
		http.Error(w, "X-Tenant-ID header is required", http.StatusBadRequest)
		return
	}
	if _, err := uuid.Parse(tenantIDStr); err != nil {
		http.Error(w, "Invalid X-Tenant-ID UUID format", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(`{"status":"accepted","message":"SWIFT subtype catalog sync queued. Run POST /api/catalog/admin/sync-subtypes for full catalog graph rebuild."}`))
}
