package httpapi

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	sharedtypes "github.com/hondyman/semlayer/libs/shared-types"
)

// GovernanceAPI provides HTTP API endpoints for governance operations
type GovernanceAPI struct {
	Evaluator     sharedtypes.Evaluator
	PolicyChecker sharedtypes.PolicyChecker
	Auditor       sharedtypes.AuditLogger
}

// NewGovernanceAPI creates a new GovernanceAPI instance
func NewGovernanceAPI(evaluator sharedtypes.Evaluator, checker sharedtypes.PolicyChecker, auditor sharedtypes.AuditLogger) *GovernanceAPI {
	return &GovernanceAPI{
		Evaluator:     evaluator,
		PolicyChecker: checker,
		Auditor:       auditor,
	}
}

// EvaluateAccess handles access evaluation requests
// @Summary Evaluate access for a user action
// @Description Evaluates whether a user has permission to perform an action on an asset
// @Tags governance
// @Accept json
// @Produce json
// @Param request body AccessEvaluationRequest true "Access evaluation request"
// @Success 200 {object} sharedtypes.APIResponse{data=AccessEvaluationResponse}
// @Failure 400 {object} sharedtypes.APIResponse
// @Failure 500 {object} sharedtypes.APIResponse
// @Router /api/v1/governance/evaluate [post]
func (api *GovernanceAPI) EvaluateAccess(c *gin.Context) {
	var req AccessEvaluationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, sharedtypes.APIResponse{
			Success: false,
			Error: &sharedtypes.APIError{
				Code:    "INVALID_REQUEST",
				Message: "Invalid request format",
				Details: err.Error(),
			},
		})
		return
	}

	// Convert to domain request
	domainReq := sharedtypes.EvaluationRequest{
		UserID:   req.UserID,
		TenantID: req.TenantID,
		AssetID:  req.AssetID,
		Action:   sharedtypes.Permission(req.Action),
		Context:  req.Context,
	}

	// Evaluate access
	allowed, reason, claims, err := api.Evaluator.Evaluate(c.Request.Context(), domainReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, sharedtypes.APIResponse{
			Success: false,
			Error: &sharedtypes.APIError{
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

	c.JSON(http.StatusOK, sharedtypes.APIResponse{
		Success: true,
		Data:    response,
		Meta: &sharedtypes.Meta{
			RequestID: c.GetHeader("X-Request-ID"),
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
// @Success 200 {object} sharedtypes.APIResponse{data=PolicyValidationResponse}
// @Failure 400 {object} sharedtypes.APIResponse
// @Failure 500 {object} sharedtypes.APIResponse
// @Router /api/v1/governance/validate [post]
func (api *GovernanceAPI) ValidatePolicy(c *gin.Context) {
	var req PolicyValidationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, sharedtypes.APIResponse{
			Success: false,
			Error: &sharedtypes.APIError{
				Code:    "INVALID_REQUEST",
				Message: "Invalid request format",
				Details: err.Error(),
			},
		})
		return
	}

	// Convert to domain request
	domainReq := sharedtypes.EvaluationRequest{
		UserID:   req.UserID,
		TenantID: req.TenantID,
		AssetID:  req.AssetID,
		Action:   sharedtypes.Permission(req.Action),
		Context:  req.Context,
	}

	// Get effective claims first
	_, _, claims, err := api.Evaluator.Evaluate(c.Request.Context(), domainReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, sharedtypes.APIResponse{
			Success: false,
			Error: &sharedtypes.APIError{
				Code:    "CLAIM_RETRIEVAL_ERROR",
				Message: "Failed to retrieve effective claims",
				Details: err.Error(),
			},
		})
		return
	}

	// Validate policy
	allowed, reason, _, _, err := api.PolicyChecker.Check(c.Request.Context(), domainReq, claims)
	if err != nil {
		c.JSON(http.StatusInternalServerError, sharedtypes.APIResponse{
			Success: false,
			Error: &sharedtypes.APIError{
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

	c.JSON(status, sharedtypes.APIResponse{
		Success: allowed,
		Data:    response,
		Meta: &sharedtypes.Meta{
			RequestID: c.GetHeader("X-Request-ID"),
			Timestamp: time.Now(),
		},
	})
}

// GetHealth returns service health status
// @Summary Get service health
// @Description Returns the current health status of the governance service
// @Tags health
// @Produce json
// @Success 200 {object} sharedtypes.APIResponse
// @Router /health [get]
func (api *GovernanceAPI) GetHealth(c *gin.Context) {
	c.JSON(http.StatusOK, sharedtypes.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"status":    "healthy",
			"service":   "governance-api",
			"version":   "1.0.0",
			"timestamp": time.Now(),
		},
	})
}
