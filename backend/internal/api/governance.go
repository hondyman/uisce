package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/domain"
)

// (APIResponse, APIError, and Meta are defined in types.go)

// AccessEvaluationRequest represents an access evaluation request
// (governance request/response DTOs are defined in governance_types.go)

// GovernanceAPI provides HTTP API endpoints for governance operations
type GovernanceAPI struct {
	Evaluator     domain.Evaluator
	PolicyChecker domain.PolicyChecker
	Auditor       domain.AuditLogger
}

// NewGovernanceAPI creates a new GovernanceAPI instance
func NewGovernanceAPI(evaluator domain.Evaluator, checker domain.PolicyChecker, auditor domain.AuditLogger) *GovernanceAPI {
	return &GovernanceAPI{
		Evaluator:     evaluator,
		PolicyChecker: checker,
		Auditor:       auditor,
	}
}

// RegisterRoutes registers governance routes
func (api *GovernanceAPI) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/governance", func(r chi.Router) {
		r.Post("/evaluate", api.EvaluateAccess)
		r.Post("/validate", api.ValidatePolicy)
	})
	r.Get("/health", api.GetHealth)
}

// EvaluateAccess handles access evaluation requests
// @Summary Evaluate access for a user action
// @Description Evaluates whether a user has permission to perform an action on an asset
// @Tags governance
// @Accept json
// @Produce json
// @Param request body AccessEvaluationRequest true "Access evaluation request"
// @Success 200 {object} APIResponse{data=AccessEvaluationResponse}
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /api/v1/governance/evaluate [post]
// EvaluateAccess handles access evaluation requests
func (api *GovernanceAPI) EvaluateAccess(w http.ResponseWriter, r *http.Request) {
	var req AccessEvaluationRequest
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

	// Convert to domain request
	domainReq := domain.EvaluationRequest{
		UserID:   req.UserID,
		TenantID: req.TenantID,
		AssetID:  req.AssetID,
		Action:   domain.Permission(req.Action),
		Context:  req.Context,
	}

	// Evaluate access
	allowed, reason, claims, err := api.Evaluator.Evaluate(r.Context(), domainReq)
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

	response := AccessEvaluationResponse{
		Allowed: allowed,
		Reason:  reason,
		Claims:  claims,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    response,
		Meta: &Meta{
			RequestID: r.Header.Get("X-Request-ID"),
			Timestamp: time.Now(),
		},
	})
}

// ValidatePolicy handles policy validation requests
// @Summary Validate policy compliance
// @Description Validates whether an action complies with current policies
// @Tags governance
// @Accept json
// @Produce json
// @Param request body PolicyValidationRequest true "Policy validation request"
// @Success 200 {object} APIResponse{data=PolicyValidationResponse}
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /api/v1/governance/validate [post]
// ValidatePolicy handles policy validation requests
func (api *GovernanceAPI) ValidatePolicy(w http.ResponseWriter, r *http.Request) {
	var req PolicyValidationRequest
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

	// Convert to domain request
	domainReq := domain.EvaluationRequest{
		UserID:   req.UserID,
		TenantID: req.TenantID,
		AssetID:  req.AssetID,
		Action:   domain.Permission(req.Action),
		Context:  req.Context,
	}

	// Get effective claims first
	_, _, claims, err := api.Evaluator.Evaluate(r.Context(), domainReq)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "CLAIM_RETRIEVAL_ERROR",
				Message: "Failed to retrieve effective claims",
				Details: err.Error(),
			},
		})
		return
	}

	// Validate policy
	allowed, reason, _, _, err := api.PolicyChecker.Check(r.Context(), domainReq, claims)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "POLICY_CHECK_ERROR",
				Message: "Failed to validate policy",
				Details: err.Error(),
			},
		})
		return
	}

	response := PolicyValidationResponse{
		Valid:       allowed,
		Violations:  []string{reason},
		Suggestions: []string{}, // Could be populated based on policy analysis
	}

	status := http.StatusOK
	if !allowed {
		status = http.StatusForbidden
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(APIResponse{
		Success: allowed,
		Data:    response,
		Meta: &Meta{
			RequestID: r.Header.Get("X-Request-ID"),
			Timestamp: time.Now(),
		},
	})
}

// GetHealth returns service health status
// @Summary Get service health
// @Description Returns the current health status of the governance service
// @Tags health
// @Produce json
// @Success 200 {object} APIResponse
// @Router /health [get]
// GetHealth returns service health status
func (api *GovernanceAPI) GetHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"status":    "healthy",
			"service":   "governance-api",
			"version":   "1.0.0",
			"timestamp": time.Now(),
		},
	})
}
