package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/domain"
	"github.com/hondyman/semlayer/backend/internal/query"
)

// QueryRewriteRequest represents a request to rewrite a query
type QueryRewriteRequest struct {
	Query         string                 `json:"query" binding:"required"`
	UserID        string                 `json:"user_id" binding:"required"`
	TenantID      string                 `json:"tenant_id" binding:"required"`
	AssetID       string                 `json:"asset_id" binding:"required"`
	UserIntent    string                 `json:"user_intent,omitempty"`
	PolicyContext map[string]interface{} `json:"policy_context,omitempty"`
}

// QueryRewriteResponse represents the response from query rewriting
type QueryRewriteResponse struct {
	RewriteID       string                    `json:"rewrite_id"`
	OriginalQuery   string                    `json:"original_query"`
	RewrittenQuery  string                    `json:"rewritten_query"`
	AppliedRules    []query.AppliedRule       `json:"applied_rules"`
	Suggestions     []query.RewriteSuggestion `json:"suggestions"`
	PerformanceTips []string                  `json:"performance_tips"`
	ComplianceNotes []string                  `json:"compliance_notes"`
	Timestamp       time.Time                 `json:"timestamp"`
}

// QuerySimulationRequest represents a request to simulate query rewriting
type QuerySimulationRequest struct {
	Query         string                 `json:"query" binding:"required"`
	UserID        string                 `json:"user_id" binding:"required"`
	TenantID      string                 `json:"tenant_id" binding:"required"`
	AssetID       string                 `json:"asset_id" binding:"required"`
	UserIntent    string                 `json:"user_intent,omitempty"`
	PolicyContext map[string]interface{} `json:"policy_context,omitempty"`
}

// QuerySimulationResponse represents the response from query simulation
type QuerySimulationResponse struct {
	SimulationID    string                    `json:"simulation_id"`
	OriginalQuery   string                    `json:"original_query"`
	SimulatedQuery  string                    `json:"simulated_query"`
	AppliedRules    []query.AppliedRule       `json:"applied_rules"`
	Suggestions     []query.RewriteSuggestion `json:"suggestions"`
	PerformanceTips []string                  `json:"performance_tips"`
	ComplianceNotes []string                  `json:"compliance_notes"`
	Timestamp       time.Time                 `json:"timestamp"`
}

// QueryRewriteHandler handles query rewriting operations
type QueryRewriteHandler struct {
	rewriteEngine *query.RewriteEngine
	evaluator     domain.Evaluator
	planner       *domain.PlannerAdapter
}

// NewQueryRewriteHandler creates a new query rewrite handler
func NewQueryRewriteHandler(
	rewriteEngine *query.RewriteEngine,
	evaluator domain.Evaluator,
	planner *domain.PlannerAdapter,
) *QueryRewriteHandler {
	return &QueryRewriteHandler{
		rewriteEngine: rewriteEngine,
		evaluator:     evaluator,
		planner:       planner,
	}
}

// RewriteQuery handles POST /query/rewrite
func (h *QueryRewriteHandler) RewriteQuery(w http.ResponseWriter, r *http.Request) {
	var req QueryRewriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "INVALID_REQUEST",
				Message: "Invalid request format",
				Details: err.Error(),
			},
		})
		return
	}

	// Evaluate access for the user/asset combination
	evalReq := domain.EvaluationRequest{
		UserID:   req.UserID,
		TenantID: req.TenantID,
		AssetID:  req.AssetID,
		Action:   "read",
		Context:  req.PolicyContext,
	}

	allowed, reason, _, err := h.evaluator.Evaluate(r.Context(), evalReq)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "EVALUATION_ERROR",
				Message: "Failed to evaluate access",
				Details: err.Error(),
			},
		})
		return
	}

	// Convert to EvaluationDecision format expected by planner
	decision := &domain.EvaluationDecision{
		Decision:      "allow",
		Reason:        reason,
		AllowedScopes: []string{}, // Extract from claims if needed
	}

	if !allowed {
		decision.Decision = "deny"
	}

	// Build pruning hints
	pruningHints, err := h.planner.BuildHints(req.AssetID, *decision, req.TenantID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "PLANNER_ERROR",
				Message: "Failed to build pruning hints",
				Details: err.Error(),
			},
		})
		return
	}

	// Create rewrite context
	rewriteCtx := &query.RewriteContext{
		UserID:        req.UserID,
		TenantID:      req.TenantID,
		AssetID:       req.AssetID,
		Decision:      *decision,
		PruningHints:  pruningHints,
		PolicyContext: req.PolicyContext,
		UserIntent:    req.UserIntent,
	}

	// Rewrite the query
	result, err := h.rewriteEngine.RewriteQuery(r.Context(), req.Query, rewriteCtx)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "REWRITE_ERROR",
				Message: "Failed to rewrite query",
				Details: err.Error(),
			},
		})
		return
	}

	response := QueryRewriteResponse{
		RewriteID:       result.RewriteID,
		OriginalQuery:   result.OriginalQuery,
		RewrittenQuery:  result.RewrittenQuery,
		AppliedRules:    result.AppliedRules,
		Suggestions:     result.Suggestions,
		PerformanceTips: result.PerformanceTips,
		ComplianceNotes: result.ComplianceNotes,
		Timestamp:       result.Timestamp,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    response,
		Meta: &Meta{
			RequestID:      r.Header.Get("X-Request-ID"),
			Timestamp:      time.Now(),
			ProcessingTime: "N/A", // Could be calculated from start time
		},
	})
}

// SimulateRewrite handles POST /query/simulate
func (h *QueryRewriteHandler) SimulateRewrite(w http.ResponseWriter, r *http.Request) {
	var req QuerySimulationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "INVALID_REQUEST",
				Message: "Invalid request format",
				Details: err.Error(),
			},
		})
		return
	}

	// Evaluate access for the user/asset combination
	evalReq := domain.EvaluationRequest{
		UserID:   req.UserID,
		TenantID: req.TenantID,
		AssetID:  req.AssetID,
		Action:   "read",
		Context:  req.PolicyContext,
	}

	allowed, reason, _, err := h.evaluator.Evaluate(r.Context(), evalReq)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "EVALUATION_ERROR",
				Message: "Failed to evaluate access",
				Details: err.Error(),
			},
		})
		return
	}

	// Convert to EvaluationDecision format expected by planner
	decision := &domain.EvaluationDecision{
		Decision:      "allow",
		Reason:        reason,
		AllowedScopes: []string{}, // Extract from claims if needed
	}

	if !allowed {
		decision.Decision = "deny"
	}

	// Build pruning hints
	pruningHints, err := h.planner.BuildHints(req.AssetID, *decision, req.TenantID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "PLANNER_ERROR",
				Message: "Failed to build pruning hints",
				Details: err.Error(),
			},
		})
		return
	}

	// Create rewrite context
	rewriteCtx := &query.RewriteContext{
		UserID:        req.UserID,
		TenantID:      req.TenantID,
		AssetID:       req.AssetID,
		Decision:      *decision,
		PruningHints:  pruningHints,
		PolicyContext: req.PolicyContext,
		UserIntent:    req.UserIntent,
	}

	// Simulate the rewrite
	result, err := h.rewriteEngine.SimulateRewrite(r.Context(), req.Query, rewriteCtx)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "SIMULATION_ERROR",
				Message: "Failed to simulate rewrite",
				Details: err.Error(),
			},
		})
		return
	}

	response := QuerySimulationResponse{
		SimulationID:    result.RewriteID,
		OriginalQuery:   result.OriginalQuery,
		SimulatedQuery:  result.RewrittenQuery,
		AppliedRules:    result.AppliedRules,
		Suggestions:     result.Suggestions,
		PerformanceTips: result.PerformanceTips,
		ComplianceNotes: result.ComplianceNotes,
		Timestamp:       result.Timestamp,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    response,
		Meta: &Meta{
			RequestID:      r.Header.Get("X-Request-ID"),
			Timestamp:      time.Now(),
			ProcessingTime: "N/A",
		},
	})
}

// GetRewriteLog handles GET /query/rewrite/:id
func (h *QueryRewriteHandler) GetRewriteLog(w http.ResponseWriter, r *http.Request) {
	rewriteID := chi.URLParam(r, "id")
	if rewriteID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "INVALID_REQUEST",
				Message: "Rewrite ID is required",
			},
		})
		return
	}

	// In a real implementation, this would retrieve from the audit logger
	// For now, return a mock response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"rewrite_id": rewriteID,
			"status":     "not_implemented",
			"message":    "Rewrite log retrieval not yet implemented",
		},
		Meta: &Meta{
			RequestID:      r.Header.Get("X-Request-ID"),
			Timestamp:      time.Now(),
			ProcessingTime: "N/A",
		},
	})
}
