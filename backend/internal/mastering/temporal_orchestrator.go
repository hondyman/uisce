package mastering

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/hondyman/uisce/backend/internal/temporal/workflows"
	"github.com/jmoiron/sqlx"
	"go.temporal.io/api/serviceerror"
	temporalclient "go.temporal.io/sdk/client"
)

// isWorkflowNotFound determines if a Temporal error is specifically due to the workflow execution not existing.
func isWorkflowNotFound(err error) bool {
	if err == nil {
		return false
	}
	var notFound *serviceerror.NotFound
	if errors.As(err, &notFound) {
		return true
	}
	// Also check string match for test environments / mock error types
	msg := err.Error()
	return msg == "workflow not found" || msg == "workflow execution not found"
}

// StartOverrideWorkflow launches a Temporal workflow to govern the override approval process
// and links the deterministic workflow ID to the PostgreSQL table.
func (e *Engine) StartOverrideWorkflow(ctx context.Context, tenantID, entity string, o *Override, actorID, actorName string) {
	if e.TemporalClient == nil || o == nil || o.Status != "PENDING" {
		return
	}

	workflowID := fmt.Sprintf("mdm-override/%s/%s/%s", tenantID, entity, o.ID)
	options := temporalclient.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: workflows.MDMGovernanceTaskQueue,
		SearchAttributes: map[string]interface{}{
			"TenantID":     tenantID,
			"BusinessUnit": entity,
		},
	}

	input := workflows.MDMOverrideWorkflowInput{
		TenantID:          tenantID,
		Entity:            entity,
		OverrideID:        o.ID,
		GoldenID:          o.GoldenID,
		Attribute:         o.Attribute,
		Action:            o.Action,
		ProposerID:        actorID,
		ProposerName:      actorName,
		ApprovalsRequired: o.ApprovalsRequired,
		ReviewSLA:         72 * time.Hour,
		ExpirySLA:         168 * time.Hour,
	}

	_, err := e.TemporalClient.ExecuteWorkflow(ctx, options, workflows.MDMOverrideApprovalWorkflow, input)
	if err != nil {
		log.Printf("[MDMGovernance] Warning: Failed to start MDMOverrideApprovalWorkflow %s: %v", workflowID, err)
		return
	}
	log.Printf("[MDMGovernance] Started MDMOverrideApprovalWorkflow: %s", workflowID)

	// Correlate temporal_workflow_id on record
	if e.Data != nil {
		_ = e.inTenant(ctx, tenantID, func(tx *sqlx.Tx) error {
			_, err := e.Data.ExecContext(ctx, `UPDATE mdm.golden_override SET temporal_workflow_id = $1 WHERE id = $2::uuid`, workflowID, o.ID)
			return err
		})
	}
}

// SignalOverrideWorkflow sends an approval, rejection, or withdrawal signal to an active override workflow.
// Returns (true, nil) if successfully signaled, (false, nil) if workflow was not found (safe direct fallback),
// or (false, err) if a transient/connectivity error occurred.
func (e *Engine) SignalOverrideWorkflow(ctx context.Context, tenantID, entity, overrideID, signalName string, payload any) (bool, error) {
	if e.TemporalClient == nil {
		return false, nil
	}
	workflowID := fmt.Sprintf("mdm-override/%s/%s/%s", tenantID, entity, overrideID)
	err := e.TemporalClient.SignalWorkflow(ctx, workflowID, "", signalName, payload)
	if err != nil {
		if isWorkflowNotFound(err) {
			log.Printf("[MDMGovernance] Workflow %s not found on server; falling back to direct CAS path", workflowID)
			return false, nil
		}
		log.Printf("[MDMGovernance] Error signaling workflow %s: %v", workflowID, err)
		return false, fmt.Errorf("temporal signal error for %s: %w", workflowID, err)
	}
	return true, nil
}

// StartMergeWorkflow launches a Temporal workflow to govern duplicate merge approvals.
func (e *Engine) StartMergeWorkflow(ctx context.Context, tenantID, entity, requestID, candidateID, keep, actorID, actorName string, need int) {
	if e.TemporalClient == nil || requestID == "" {
		return
	}

	workflowID := fmt.Sprintf("mdm-merge/%s/%s/%s", tenantID, entity, requestID)
	options := temporalclient.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: workflows.MDMGovernanceTaskQueue,
		SearchAttributes: map[string]interface{}{
			"TenantID":     tenantID,
			"BusinessUnit": entity,
		},
	}

	input := workflows.MDMMergeWorkflowInput{
		TenantID:          tenantID,
		Entity:            entity,
		RequestID:         requestID,
		CandidateID:       candidateID,
		Keep:              keep,
		ProposerID:        actorID,
		ProposerName:      actorName,
		ApprovalsRequired: need,
		ReviewSLA:         72 * time.Hour,
		ExpirySLA:         168 * time.Hour,
	}

	_, err := e.TemporalClient.ExecuteWorkflow(ctx, options, workflows.MDMMergeApprovalWorkflow, input)
	if err != nil {
		log.Printf("[MDMGovernance] Warning: Failed to start MDMMergeApprovalWorkflow %s: %v", workflowID, err)
		return
	}
	log.Printf("[MDMGovernance] Started MDMMergeApprovalWorkflow: %s", workflowID)

	if e.Data != nil {
		_ = e.inTenant(ctx, tenantID, func(tx *sqlx.Tx) error {
			_, err := e.Data.ExecContext(ctx, `UPDATE mdm.golden_merge_request SET temporal_workflow_id = $1 WHERE id = $2::uuid`, workflowID, requestID)
			return err
		})
	}
}

// SignalMergeWorkflow sends an approval or rejection signal to an active merge workflow.
func (e *Engine) SignalMergeWorkflow(ctx context.Context, tenantID, entity, requestID, signalName string, payload any) (bool, error) {
	if e.TemporalClient == nil {
		return false, nil
	}
	workflowID := fmt.Sprintf("mdm-merge/%s/%s/%s", tenantID, entity, requestID)
	err := e.TemporalClient.SignalWorkflow(ctx, workflowID, "", signalName, payload)
	if err != nil {
		if isWorkflowNotFound(err) {
			log.Printf("[MDMGovernance] Merge workflow %s not found on server; falling back to direct CAS path", workflowID)
			return false, nil
		}
		log.Printf("[MDMGovernance] Error signaling merge workflow %s: %v", workflowID, err)
		return false, fmt.Errorf("temporal signal error for %s: %w", workflowID, err)
	}
	return true, nil
}

// StartConfigChangeWorkflow launches a Temporal workflow for maker-checker configuration changes.
func (e *Engine) StartConfigChangeWorkflow(ctx context.Context, tenantID string, ch *ConfigChange, actorID, actorName string) {
	if e.TemporalClient == nil || ch == nil || ch.Status != "pending" {
		return
	}

	entity := ""
	if ch.Entity != nil {
		entity = *ch.Entity
	}

	workflowID := fmt.Sprintf("mdm-config/%s/%s", tenantID, ch.ID)
	options := temporalclient.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: workflows.MDMGovernanceTaskQueue,
		SearchAttributes: map[string]interface{}{
			"TenantID":     tenantID,
			"BusinessUnit": entity,
		},
	}

	input := workflows.MDMConfigWorkflowInput{
		TenantID:     tenantID,
		Entity:       entity,
		ChangeID:     ch.ID,
		Kind:         ch.Kind,
		Action:       ch.Action,
		ProposerID:   actorID,
		ProposerName: actorName,
		ReviewSLA:    72 * time.Hour,
		ExpirySLA:    168 * time.Hour,
	}

	_, err := e.TemporalClient.ExecuteWorkflow(ctx, options, workflows.MDMConfigChangeApprovalWorkflow, input)
	if err != nil {
		log.Printf("[MDMGovernance] Warning: Failed to start MDMConfigChangeApprovalWorkflow %s: %v", workflowID, err)
		return
	}
	log.Printf("[MDMGovernance] Started MDMConfigChangeApprovalWorkflow: %s", workflowID)

	if e.Data != nil {
		_ = e.inTenant(ctx, tenantID, func(tx *sqlx.Tx) error {
			_, err := e.Data.ExecContext(ctx, `UPDATE mdm.mastering_config_change SET temporal_workflow_id = $1 WHERE id = $2::uuid`, workflowID, ch.ID)
			return err
		})
	}
}

// SignalConfigChangeWorkflow sends an approval, rejection, or withdrawal signal to a config workflow.
func (e *Engine) SignalConfigChangeWorkflow(ctx context.Context, tenantID, changeID, signalName string, payload any) (bool, error) {
	if e.TemporalClient == nil {
		return false, nil
	}
	workflowID := fmt.Sprintf("mdm-config/%s/%s", tenantID, changeID)
	err := e.TemporalClient.SignalWorkflow(ctx, workflowID, "", signalName, payload)
	if err != nil {
		if isWorkflowNotFound(err) {
			log.Printf("[MDMGovernance] Config change workflow %s not found on server; falling back to direct CAS path", workflowID)
			return false, nil
		}
		log.Printf("[MDMGovernance] Error signaling config change workflow %s: %v", workflowID, err)
		return false, fmt.Errorf("temporal signal error for %s: %w", workflowID, err)
	}
	return true, nil
}
