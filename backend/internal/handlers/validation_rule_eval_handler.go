package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/analytics"
	"github.com/hondyman/uisce/backend/internal/logging"
)

func parseUUIDOrNil(s string) uuid.UUID {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil
	}
	return id
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

func (h *ValidationRuleHandler) handleEvaluateRecord(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := mustTenantID(r)
	if !ok {
		// Fallback for tests or unsecured local calls if tenant is passed in body or query
		tStr := r.URL.Query().Get("tenant_id")
		if tStr == "" {
			tStr = r.Header.Get("X-Tenant-ID")
		}
		if tStr == "" {
			tStr = "00000000-0000-0000-0000-000000000000"
		}
		tenantID = parseUUIDOrNil(tStr)
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
		tStr := r.URL.Query().Get("tenant_id")
		if tStr == "" {
			tStr = r.Header.Get("X-Tenant-ID")
		}
		if tStr == "" {
			tStr = "00000000-0000-0000-0000-000000000000"
		}
		tenantID = parseUUIDOrNil(tStr)
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

func (h *ValidationRuleHandler) handleLoadSnapshot(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := mustTenantID(r)
	if !ok {
		tStr := r.URL.Query().Get("tenant_id")
		if tStr == "" {
			tStr = r.Header.Get("X-Tenant-ID")
		}
		if tStr == "" {
			tStr = "00000000-0000-0000-0000-000000000000"
		}
		tenantID = parseUUIDOrNil(tStr)
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

