package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/hondyman/uisce/backend/internal/analytics"
	"github.com/hondyman/uisce/backend/internal/logging"
	"golang.org/x/time/rate"
)

var (
	pushdownSemaphore = make(chan struct{}, 10) // max 10 concurrent pushdowns cluster-wide
	batchLimiterMu    sync.Mutex
	batchLimiters     = make(map[string]*rate.Limiter)
)

func getBatchLimiter(tenantID string) *rate.Limiter {
	batchLimiterMu.Lock()
	defer batchLimiterMu.Unlock()
	lim, ok := batchLimiters[tenantID]
	if !ok {
		lim = rate.NewLimiter(rate.Limit(60.0/60.0), 60) // 60 batch requests per minute
		batchLimiters[tenantID] = lim
	}
	return lim
}

// EvalService is the interface required for rule evaluation endpoints.
type EvalService interface {
	LoadRuleSnapshot(ctx context.Context, tenantID, boName, domain, timing string) (*analytics.RuleSnapshot, error)
	EvaluateBatch(ctx context.Context, snap *analytics.RuleSnapshot, records []map[string]any, loader analytics.ContextLoader) (*analytics.EvaluateBatchResult, error)
}

// EvaluateRecordRequest specifies parameters for evaluating a single record.
type EvaluateRecordRequest struct {
	BOName string         `json:"bo_name"`
	Domain string         `json:"domain,omitempty"`
	Timing string         `json:"timing,omitempty"`
	Record map[string]any `json:"record"`
}

// EvaluateBatchRequest specifies parameters for batch evaluation.
type EvaluateBatchRequest struct {
	BOName  string           `json:"bo_name"`
	Domain  string           `json:"domain,omitempty"`
	Timing  string           `json:"timing,omitempty"`
	Records []map[string]any `json:"records"`
}

// EvaluatePushdownRequest specifies parameters for pushdown evaluation.
type EvaluatePushdownRequest struct {
	BOName        string    `json:"bo_name"`
	Table         string    `json:"table"`
	PKColumn      string    `json:"pk_column"`
	Domain        string    `json:"domain,omitempty"`
	Timing        string    `json:"timing,omitempty"`
	MaxDetailRows int       `json:"max_detail_rows,omitempty"`
	AsOf          time.Time `json:"as_of,omitempty"`
}

func (h *ValidationRuleHandler) handleEvaluateRecord(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := mustTenantID(r)
	if !ok {
		http.Error(w, "unauthorized: no tenant established for this request", http.StatusUnauthorized)
		return
	}

	var req EvaluateRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.BOName == "" {
		http.Error(w, "bo_name is required", http.StatusBadRequest)
		return
	}

	evaluator := h.evaluator
	if evaluator == nil {
		evaluator = h.svc
	}
	if evaluator == nil {
		http.Error(w, "evaluation service not configured", http.StatusInternalServerError)
		return
	}

	snap, err := evaluator.LoadRuleSnapshot(r.Context(), tenantID.String(), req.BOName, req.Domain, req.Timing)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("evaluate-record: load snapshot failed: %v", err)
		http.Error(w, "load rule snapshot failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	evalRes, err := analytics.EvaluateRecord(r.Context(), snap, req.Record, nil)
	if err != nil {
		if scre, ok := err.(*analytics.ServerContextRequiredError); ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK) // 200 with domain error payload
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error":                  "ERR_SERVER_CONTEXT_REQUIRED",
				"missing_context_fields": scre.MissingFields,
				"hint":                   scre.Hint,
			})
			return
		}
		http.Error(w, "evaluate record failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(evalRes)
}

func (h *ValidationRuleHandler) handleEvaluateBatch(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := mustTenantID(r)
	if !ok {
		http.Error(w, "unauthorized: no tenant established for this request", http.StatusUnauthorized)
		return
	}

	// Rate limiting: max 60 batch evaluation requests per minute per tenant
	if !getBatchLimiter(tenantID.String()).Allow() {
		http.Error(w, "rate limit exceeded: max 60 batch evaluation requests per minute per tenant", http.StatusTooManyRequests)
		return
	}

	var req EvaluateBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.BOName == "" {
		http.Error(w, "bo_name is required", http.StatusBadRequest)
		return
	}
	if len(req.Records) > analytics.MaxBatchSize {
		http.Error(w, fmt.Sprintf("batch exceeds maximum limit of %d records", analytics.MaxBatchSize), http.StatusRequestEntityTooLarge)
		return
	}

	evaluator := h.evaluator
	if evaluator == nil {
		evaluator = h.svc
	}
	if evaluator == nil {
		http.Error(w, "evaluation service not configured", http.StatusInternalServerError)
		return
	}

	snap, err := evaluator.LoadRuleSnapshot(r.Context(), tenantID.String(), req.BOName, req.Domain, req.Timing)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("evaluate-batch: load snapshot failed: %v", err)
		http.Error(w, "load rule snapshot failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	batchRes, err := evaluator.EvaluateBatch(r.Context(), snap, req.Records, nil)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("evaluate-batch: evaluation failed: %v", err)
		http.Error(w, "evaluate batch failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(batchRes)
}

func (h *ValidationRuleHandler) handleEvaluatePushdown(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := mustTenantID(r)
	if !ok {
		http.Error(w, "unauthorized: no tenant established for this request", http.StatusUnauthorized)
		return
	}

	// Concurrency quota guard: max 10 concurrent pushdowns cluster-wide
	select {
	case pushdownSemaphore <- struct{}{}:
		defer func() { <-pushdownSemaphore }()
	default:
		http.Error(w, "pushdown capacity exceeded: max concurrent pushdown operations running (try again shortly)", http.StatusTooManyRequests)
		return
	}

	var req EvaluatePushdownRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.BOName == "" {
		http.Error(w, "bo_name is required", http.StatusBadRequest)
		return
	}
	if req.Table == "" {
		http.Error(w, "table is required", http.StatusBadRequest)
		return
	}
	if req.PKColumn == "" {
		req.PKColumn = "id"
	}
	if req.MaxDetailRows <= 0 {
		req.MaxDetailRows = 100
	}

	if h.svc == nil {
		http.Error(w, "validation rule service not configured", http.StatusInternalServerError)
		return
	}

	snap, err := h.svc.LoadRuleSnapshotAsOf(r.Context(), tenantID.String(), req.BOName, req.Domain, req.Timing, req.AsOf)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("evaluate-pushdown: load snapshot failed: %v", err)
		http.Error(w, "load rule snapshot failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	connRes := analytics.NewDirectConnResolver(h.db)
	opts := analytics.PushdownOptions{
		MaxDetailRows: req.MaxDetailRows,
	}

	res, err := h.svc.RunPushdown(r.Context(), connRes, snap, req.Table, req.PKColumn, opts)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("evaluate-pushdown: pushdown execution failed: %v", err)
		http.Error(w, "pushdown execution failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(res)
}

func (h *ValidationRuleHandler) handleLoadSnapshot(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := mustTenantID(r)
	if !ok {
		http.Error(w, "unauthorized: no tenant established for this request", http.StatusUnauthorized)
		return
	}

	boName := r.URL.Query().Get("bo_name")
	domain := r.URL.Query().Get("domain")
	timing := r.URL.Query().Get("timing")
	if boName == "" {
		http.Error(w, "bo_name is required", http.StatusBadRequest)
		return
	}

	evaluator := h.evaluator
	if evaluator == nil {
		evaluator = h.svc
	}
	if evaluator == nil {
		http.Error(w, "evaluation service not configured", http.StatusInternalServerError)
		return
	}

	snap, err := evaluator.LoadRuleSnapshot(r.Context(), tenantID.String(), boName, domain, timing)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("load snapshot failed: %v", err)
		http.Error(w, "load rule snapshot failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(snap)
}

