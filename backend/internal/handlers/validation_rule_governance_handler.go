package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/logging"
	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/hondyman/uisce/backend/internal/temporal/workflows"
	"go.temporal.io/sdk/client"
)

type SubmitReviewRequest struct {
	AuthorID  string `json:"author_id"`
	ReviewSLA string `json:"review_sla,omitempty"` // e.g. "72h"
}

type ApproveRuleRequest struct {
	ApproverID string `json:"approver_id"`
}

type RejectRuleRequest struct {
	RejecterID string `json:"rejecter_id"`
	Reason     string `json:"reason"`
}

// handleSubmitReview transitions a rule into 'submitted_for_review' and starts a RuleReviewWorkflow.
func (h *ValidationRuleHandler) handleSubmitReview(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := mustTenantID(r)
	if !ok {
		http.Error(w, "tenant_id is required", http.StatusUnauthorized)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var req SubmitReviewRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	authorID := req.AuthorID
	if authorID == "" {
		authorID = "unknown_author"
	}

	// 1. Update rule properties to submitted_for_review
	var row struct {
		Properties json.RawMessage `db:"properties"`
	}
	err = h.db.GetContext(r.Context(), &row, `
		SELECT properties FROM catalog_node
		WHERE id = $1 AND tenant_id = $2::uuid
	`, id, tenantID)
	if err != nil {
		http.Error(w, "validation rule not found", http.StatusNotFound)
		return
	}

	props, _ := models.ParseValidationRuleProperties(row.Properties)
	props.GovernanceStatus = models.ValidationRuleGovernanceSubmittedForReview
	props.AuthorID = authorID
	now := time.Now().UTC()
	props.SubmittedAt = &now

	updatedProps, err := json.Marshal(props)
	if err != nil {
		http.Error(w, "failed to marshal properties", http.StatusInternalServerError)
		return
	}

	_, err = h.db.ExecContext(r.Context(), `
		UPDATE catalog_node
		SET properties = $1, updated_at = NOW()
		WHERE id = $2 AND tenant_id = $3::uuid
	`, updatedProps, id, tenantID)
	if err != nil {
		http.Error(w, "failed to update rule status: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 2. Start Temporal RuleReviewWorkflow if TemporalClient is available
	workflowID := fmt.Sprintf("rule-review/%s/%d", id.String(), props.Version)
	if h.temporalClient != nil {
		sla := 72 * time.Hour
		if req.ReviewSLA != "" {
			if parsed, err := time.ParseDuration(req.ReviewSLA); err == nil {
				sla = parsed
			}
		}

		workflowOptions := client.StartWorkflowOptions{
			ID:        workflowID,
			TaskQueue: workflows.RuleGovernanceTaskQueue,
		}

		input := workflows.RuleReviewWorkflowInput{
			TenantID:   tenantID.String(),
			RuleNodeID: id.String(),
			Version:    props.Version,
			AuthorID:   authorID,
			ReviewSLA:  sla,
		}

		_, err = h.temporalClient.ExecuteWorkflow(r.Context(), workflowOptions, workflows.RuleReviewWorkflow, input)
		if err != nil {
			logging.GetLogger().Sugar().Warnf("failed to start RuleReviewWorkflow for %s: %v", id, err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "submitted_for_review",
		"workflow_id": workflowID,
		"rule_id":     id.String(),
	})
}

// handleApproveRule dispatches the Approve signal to the active RuleReviewWorkflow.
func (h *ValidationRuleHandler) handleApproveRule(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := mustTenantID(r)
	if !ok {
		http.Error(w, "tenant_id is required", http.StatusUnauthorized)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var req ApproveRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ApproverID == "" {
		http.Error(w, "approver_id is required", http.StatusBadRequest)
		return
	}

	var row struct {
		Properties json.RawMessage `db:"properties"`
	}
	err = h.db.GetContext(r.Context(), &row, `
		SELECT properties FROM catalog_node
		WHERE id = $1 AND tenant_id = $2::uuid
	`, id, tenantID)
	if err != nil {
		http.Error(w, "validation rule not found", http.StatusNotFound)
		return
	}

	props, _ := models.ParseValidationRuleProperties(row.Properties)
	if props.AuthorID != "" && props.AuthorID == req.ApproverID {
		http.Error(w, "four-eyes violation: author cannot approve their own rule", http.StatusForbidden)
		return
	}

	workflowID := fmt.Sprintf("rule-review/%s/%d", id.String(), props.Version)
	if h.temporalClient != nil {
		err = h.temporalClient.SignalWorkflow(r.Context(), workflowID, "", workflows.SignalRuleApprove, workflows.ApprovalSignalPayload{
			ApproverID: req.ApproverID,
		})
		if err != nil {
			logging.GetLogger().Sugar().Warnf("failed to signal RuleReviewWorkflow: %v", err)
		}
	} else {
		// Fallback: direct DB transition when running without Temporal server
		props.GovernanceStatus = models.ValidationRuleGovernanceApproved
		props.ApprovedBy = req.ApproverID
		now := time.Now().UTC()
		props.ApprovedAt = &now
		updatedProps, _ := json.Marshal(props)
		_, _ = h.db.ExecContext(r.Context(), `
			UPDATE catalog_node SET properties = $1, updated_at = NOW() WHERE id = $2 AND tenant_id = $3::uuid
		`, updatedProps, id, tenantID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "approved",
		"workflow_id": workflowID,
		"rule_id":     id.String(),
	})
}

// handleRejectRule dispatches the Reject signal to the active RuleReviewWorkflow.
func (h *ValidationRuleHandler) handleRejectRule(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := mustTenantID(r)
	if !ok {
		http.Error(w, "tenant_id is required", http.StatusUnauthorized)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var req RejectRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Reason == "" {
		req.Reason = "rejected by reviewer"
	}

	var row struct {
		Properties json.RawMessage `db:"properties"`
	}
	err = h.db.GetContext(r.Context(), &row, `
		SELECT properties FROM catalog_node
		WHERE id = $1 AND tenant_id = $2::uuid
	`, id, tenantID)
	if err != nil {
		http.Error(w, "validation rule not found", http.StatusNotFound)
		return
	}

	props, _ := models.ParseValidationRuleProperties(row.Properties)
	workflowID := fmt.Sprintf("rule-review/%s/%d", id.String(), props.Version)

	if h.temporalClient != nil {
		err = h.temporalClient.SignalWorkflow(r.Context(), workflowID, "", workflows.SignalRuleReject, workflows.RejectionSignalPayload{
			RejecterID: req.RejecterID,
			Reason:     req.Reason,
		})
		if err != nil {
			logging.GetLogger().Sugar().Warnf("failed to signal RuleReviewWorkflow rejection: %v", err)
		}
	} else {
		props.GovernanceStatus = models.ValidationRuleGovernanceRejected
		props.RejectionReason = req.Reason
		props.RejectedBy = req.RejecterID
		now := time.Now().UTC()
		props.RejectedAt = &now
		updatedProps, _ := json.Marshal(props)
		_, _ = h.db.ExecContext(r.Context(), `
			UPDATE catalog_node SET properties = $1, updated_at = NOW() WHERE id = $2 AND tenant_id = $3::uuid
		`, updatedProps, id, tenantID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "rejected",
		"workflow_id": workflowID,
		"rule_id":     id.String(),
	})
}
