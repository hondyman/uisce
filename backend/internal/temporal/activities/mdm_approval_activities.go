package activities

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type MDMApprovalActivities struct {
	DB     *sqlx.DB
	Logger *zap.SugaredLogger
}

func NewMDMApprovalActivities(db *sqlx.DB, logger *zap.SugaredLogger) *MDMApprovalActivities {
	return &MDMApprovalActivities{
		DB:     db,
		Logger: logger,
	}
}

// MDMActivityResult provides structured outcome for CAS transitions.
type MDMActivityResult struct {
	Applied bool   `json:"applied"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// MDMCheckStatusInput defines input for pre-flight status verification.
type MDMCheckStatusInput struct {
	TenantID string `json:"tenant_id"`
	Entity   string `json:"entity,omitempty"`
	TargetID string `json:"target_id"`
	Kind     string `json:"kind"` // OVERRIDE | MERGE | CONFIG
}

// MDMCheckStatusResult defines the status returned by pre-flight check.
type MDMCheckStatusResult struct {
	Status string `json:"status"`
	Active bool   `json:"active"`
}

// MDMOverrideVoteInput defines the payload for recording an override vote.
type MDMOverrideVoteInput struct {
	TenantID     string `json:"tenant_id"`
	Entity       string `json:"entity"`
	OverrideID   string `json:"override_id"`
	ApproverID   string `json:"approver_id"`
	ApproverName string `json:"approver_name"`
	Decision     string `json:"decision"` // APPROVE | REJECT
	Comment      string `json:"comment"`
}

// MDMApplyOverrideInput defines the payload for applying an approved override.
type MDMApplyOverrideInput struct {
	TenantID   string `json:"tenant_id"`
	Entity     string `json:"entity"`
	OverrideID string `json:"override_id"`
}

// MDMRejectOverrideInput defines the payload for rejecting an override.
type MDMRejectOverrideInput struct {
	TenantID     string `json:"tenant_id"`
	Entity       string `json:"entity"`
	OverrideID   string `json:"override_id"`
	RejecterID   string `json:"rejecter_id"`
	RejecterName string `json:"rejecter_name"`
	Reason       string `json:"reason"`
}

// MDMWithdrawOverrideInput defines the payload for withdrawing an override.
type MDMWithdrawOverrideInput struct {
	TenantID   string `json:"tenant_id"`
	Entity     string `json:"entity"`
	OverrideID string `json:"override_id"`
	ActorID    string `json:"actor_id"`
	Reason     string `json:"reason"`
}

// MDMMergeVoteInput defines the payload for recording a merge vote.
type MDMMergeVoteInput struct {
	TenantID     string `json:"tenant_id"`
	Entity       string `json:"entity"`
	RequestID    string `json:"request_id"`
	ApproverID   string `json:"approver_id"`
	ApproverName string `json:"approver_name"`
	Decision     string `json:"decision"`
	Comment      string `json:"comment"`
}

// MDMApplyMergeInput defines the payload for applying an approved merge.
type MDMApplyMergeInput struct {
	TenantID  string `json:"tenant_id"`
	Entity    string `json:"entity"`
	RequestID string `json:"request_id"`
}

// MDMRejectMergeInput defines the payload for rejecting a merge.
type MDMRejectMergeInput struct {
	TenantID     string `json:"tenant_id"`
	Entity       string `json:"entity"`
	RequestID    string `json:"request_id"`
	RejecterID   string `json:"rejecter_id"`
	RejecterName string `json:"rejecter_name"`
	Reason       string `json:"reason"`
}

// MDMApplyConfigInput defines the payload for applying a config change.
type MDMApplyConfigInput struct {
	TenantID     string `json:"tenant_id"`
	ChangeID     string `json:"change_id"`
	ApproverID   string `json:"approver_id"`
	ApproverName string `json:"approver_name"`
	Comment      string `json:"comment"`
}

// MDMRejectConfigInput defines the payload for rejecting a config change.
type MDMRejectConfigInput struct {
	TenantID     string `json:"tenant_id"`
	ChangeID     string `json:"change_id"`
	RejecterID   string `json:"rejecter_id"`
	RejecterName string `json:"rejecter_name"`
	Reason       string `json:"reason"`
}

// MDMWithdrawConfigInput defines the payload for withdrawing a config change.
type MDMWithdrawConfigInput struct {
	TenantID string `json:"tenant_id"`
	ChangeID string `json:"change_id"`
	ActorID  string `json:"actor_id"`
	Reason   string `json:"reason"`
}

// MDMEscalateInput defines the payload for escalating an SLA breach.
type MDMEscalateInput struct {
	TenantID string `json:"tenant_id"`
	Entity   string `json:"entity"`
	TargetID string `json:"target_id"`
	Kind     string `json:"kind"` // OVERRIDE | MERGE | CONFIG
	Reason   string `json:"reason"`
}

// MDMReconciliationInput defines parameters for the zombie / pending reconciliation sweep.
type MDMReconciliationInput struct {
	TenantID      string        `json:"tenant_id,omitempty"`
	MaxPendingAge time.Duration `json:"max_pending_age"`
	ExpiryAge     time.Duration `json:"expiry_age"`
}

// MDMReconciliationResult reports reconciled counts.
type MDMReconciliationResult struct {
	OverridesLinked  int `json:"overrides_linked"`
	OverridesExpired int `json:"overrides_expired"`
	MergesLinked     int `json:"merges_linked"`
	MergesExpired    int `json:"merges_expired"`
	ConfigsLinked    int `json:"configs_linked"`
	ConfigsExpired   int `json:"configs_expired"`
}

// CheckProposalStatusActivity performs a pre-flight status check on a proposal.
func (a *MDMApprovalActivities) CheckProposalStatusActivity(ctx context.Context, input MDMCheckStatusInput) (*MDMCheckStatusResult, error) {
	if a.DB == nil {
		return &MDMCheckStatusResult{Status: "PENDING", Active: true}, nil
	}
	var status string
	var err error
	switch strings.ToUpper(input.Kind) {
	case "OVERRIDE":
		err = a.DB.GetContext(ctx, &status, `SELECT status FROM mdm.golden_override WHERE id = $1::uuid AND tenant_id = $2::uuid`, input.TargetID, input.TenantID)
	case "MERGE":
		err = a.DB.GetContext(ctx, &status, `SELECT status FROM mdm.golden_merge_request WHERE id = $1::uuid AND tenant_id = $2::uuid`, input.TargetID, input.TenantID)
	case "CONFIG":
		err = a.DB.GetContext(ctx, &status, `SELECT status FROM mdm.mastering_config_change WHERE id = $1::uuid AND tenant_id = $2::uuid`, input.TargetID, input.TenantID)
	default:
		return nil, fmt.Errorf("unknown proposal kind: %s", input.Kind)
	}

	if errors.Is(err, sql.ErrNoRows) {
		return &MDMCheckStatusResult{Status: "NOT_FOUND", Active: false}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("check status for %s %s: %w", input.Kind, input.TargetID, err)
	}

	normalized := strings.ToUpper(status)
	return &MDMCheckStatusResult{
		Status: normalized,
		Active: normalized == "PENDING",
	}, nil
}

// RecordOverrideVoteActivity records an approver's vote in PostgreSQL with four-eyes enforcement.
func (a *MDMApprovalActivities) RecordOverrideVoteActivity(ctx context.Context, input MDMOverrideVoteInput) error {
	if a.DB == nil {
		return nil
	}

	// Four-eyes defense in depth check: proposer cannot vote
	var requestedBy string
	err := a.DB.GetContext(ctx, &requestedBy, `
		SELECT requested_by FROM mdm.golden_override
		WHERE id = $1::uuid AND tenant_id = $2::uuid
	`, input.OverrideID, input.TenantID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("verify override proposer: %w", err)
	}
	if requestedBy != "" && requestedBy == input.ApproverID {
		return fmt.Errorf("four-eyes governance violation: proposer %q cannot vote on own override", input.ApproverID)
	}

	_, err = a.DB.ExecContext(ctx, `
		INSERT INTO mdm.golden_override_vote (tenant_id, override_id, approver, approver_name, decision, comment)
		VALUES ($1::uuid, $2::uuid, $3, NULLIF($4, ''), $5, NULLIF($6, ''))
		ON CONFLICT (override_id, approver) DO UPDATE
		SET decision = EXCLUDED.decision, comment = EXCLUDED.comment, decided_at = now()
	`, input.TenantID, input.OverrideID, input.ApproverID, input.ApproverName, input.Decision, strings.TrimSpace(input.Comment))
	if err != nil {
		return fmt.Errorf("failed to record override vote: %w", err)
	}
	if a.Logger != nil {
		a.Logger.Infof("[MDMGovernance] Recorded vote %s for override %s by %s", input.Decision, input.OverrideID, input.ApproverID)
	}
	return nil
}

// ApplyOverrideActivity marks the override as APPLIED with Compare-And-Swap (CAS) idempotency.
func (a *MDMApprovalActivities) ApplyOverrideActivity(ctx context.Context, input MDMApplyOverrideInput) (*MDMActivityResult, error) {
	if a.DB == nil {
		return &MDMActivityResult{Applied: true, Status: "APPLIED"}, nil
	}
	res, err := a.DB.ExecContext(ctx, `
		UPDATE mdm.golden_override 
		SET status = 'APPLIED', applied_at = now(), active = (action = 'SET')
		WHERE id = $1::uuid AND tenant_id = $2::uuid AND status = 'PENDING'
	`, input.OverrideID, input.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to apply override %s: %w", input.OverrideID, err)
	}
	rows, _ := res.RowsAffected()
	if rows == 1 {
		if a.Logger != nil {
			a.Logger.Infof("[MDMGovernance] Applied override %s", input.OverrideID)
		}
		return &MDMActivityResult{Applied: true, Status: "APPLIED"}, nil
	}

	// Verify current status to handle lost race vs idempotent retry
	var currentStatus string
	if err := a.DB.GetContext(ctx, &currentStatus, `SELECT status FROM mdm.golden_override WHERE id = $1::uuid AND tenant_id = $2::uuid`, input.OverrideID, input.TenantID); err == nil {
		if currentStatus == "APPLIED" {
			return &MDMActivityResult{Applied: true, Status: "APPLIED", Message: "already applied (idempotent NOOP)"}, nil
		}
		return &MDMActivityResult{Applied: false, Status: currentStatus, Message: "lost race to " + currentStatus}, nil
	}
	return nil, fmt.Errorf("override %s not found", input.OverrideID)
}

// RejectOverrideActivity updates the override status to REJECTED with CAS idempotency.
func (a *MDMApprovalActivities) RejectOverrideActivity(ctx context.Context, input MDMRejectOverrideInput) (*MDMActivityResult, error) {
	if a.DB == nil {
		return &MDMActivityResult{Applied: false, Status: "REJECTED"}, nil
	}
	res, err := a.DB.ExecContext(ctx, `
		UPDATE mdm.golden_override 
		SET status = 'REJECTED', ended_at = now()
		WHERE id = $1::uuid AND tenant_id = $2::uuid AND status = 'PENDING'
	`, input.OverrideID, input.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to reject override %s: %w", input.OverrideID, err)
	}
	rows, _ := res.RowsAffected()
	if rows == 1 {
		return &MDMActivityResult{Applied: false, Status: "REJECTED"}, nil
	}
	var currentStatus string
	if err := a.DB.GetContext(ctx, &currentStatus, `SELECT status FROM mdm.golden_override WHERE id = $1::uuid AND tenant_id = $2::uuid`, input.OverrideID, input.TenantID); err == nil {
		return &MDMActivityResult{Applied: false, Status: currentStatus}, nil
	}
	return nil, fmt.Errorf("override %s not found", input.OverrideID)
}

// WithdrawOverrideActivity updates the override status to WITHDRAWN with CAS idempotency and proposer verification.
func (a *MDMApprovalActivities) WithdrawOverrideActivity(ctx context.Context, input MDMWithdrawOverrideInput) (*MDMActivityResult, error) {
	if a.DB == nil {
		return &MDMActivityResult{Applied: false, Status: "WITHDRAWN"}, nil
	}
	res, err := a.DB.ExecContext(ctx, `
		UPDATE mdm.golden_override 
		SET status = 'WITHDRAWN', ended_at = now()
		WHERE id = $1::uuid AND tenant_id = $2::uuid AND status = 'PENDING' AND requested_by = $3
	`, input.OverrideID, input.TenantID, input.ActorID)
	if err != nil {
		return nil, fmt.Errorf("failed to withdraw override %s: %w", input.OverrideID, err)
	}
	rows, _ := res.RowsAffected()
	if rows == 1 {
		return &MDMActivityResult{Applied: false, Status: "WITHDRAWN"}, nil
	}
	var currentStatus string
	if err := a.DB.GetContext(ctx, &currentStatus, `SELECT status FROM mdm.golden_override WHERE id = $1::uuid AND tenant_id = $2::uuid`, input.OverrideID, input.TenantID); err == nil {
		return &MDMActivityResult{Applied: false, Status: currentStatus}, nil
	}
	return nil, fmt.Errorf("override %s not found or actor %s not proposer", input.OverrideID, input.ActorID)
}

// RecordMergeVoteActivity records an approver's vote for candidate merge with four-eyes enforcement.
func (a *MDMApprovalActivities) RecordMergeVoteActivity(ctx context.Context, input MDMMergeVoteInput) error {
	if a.DB == nil {
		return nil
	}

	var requestedBy string
	err := a.DB.GetContext(ctx, &requestedBy, `
		SELECT requested_by FROM mdm.golden_merge_request
		WHERE id = $1::uuid AND tenant_id = $2::uuid
	`, input.RequestID, input.TenantID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("verify merge proposer: %w", err)
	}
	if requestedBy != "" && requestedBy == input.ApproverID {
		return fmt.Errorf("four-eyes governance violation: requester %q cannot vote on own merge", input.ApproverID)
	}

	_, err = a.DB.ExecContext(ctx, `
		INSERT INTO mdm.golden_merge_vote (tenant_id, request_id, approver, approver_name, decision, comment)
		VALUES ($1::uuid, $2::uuid, $3, NULLIF($4, ''), $5, NULLIF($6, ''))
		ON CONFLICT (request_id, approver) DO UPDATE
		SET decision = EXCLUDED.decision, comment = EXCLUDED.comment, decided_at = now()
	`, input.TenantID, input.RequestID, input.ApproverID, input.ApproverName, input.Decision, strings.TrimSpace(input.Comment))
	if err != nil {
		return fmt.Errorf("failed to record merge vote: %w", err)
	}
	if a.Logger != nil {
		a.Logger.Infof("[MDMGovernance] Recorded vote %s for merge %s by %s", input.Decision, input.RequestID, input.ApproverID)
	}
	return nil
}

// ApplyMergeActivity marks the merge request as APPROVED/APPLIED with CAS idempotency.
func (a *MDMApprovalActivities) ApplyMergeActivity(ctx context.Context, input MDMApplyMergeInput) (*MDMActivityResult, error) {
	if a.DB == nil {
		return &MDMActivityResult{Applied: true, Status: "APPROVED"}, nil
	}
	res, err := a.DB.ExecContext(ctx, `
		UPDATE mdm.golden_merge_request
		SET status = 'APPROVED', decided_at = now()
		WHERE id = $1::uuid AND tenant_id = $2::uuid AND status = 'PENDING'
	`, input.RequestID, input.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to apply merge request %s: %w", input.RequestID, err)
	}
	rows, _ := res.RowsAffected()
	if rows == 1 {
		return &MDMActivityResult{Applied: true, Status: "APPROVED"}, nil
	}
	var currentStatus string
	if err := a.DB.GetContext(ctx, &currentStatus, `SELECT status FROM mdm.golden_merge_request WHERE id = $1::uuid AND tenant_id = $2::uuid`, input.RequestID, input.TenantID); err == nil {
		if currentStatus == "APPROVED" || currentStatus == "APPLIED" {
			return &MDMActivityResult{Applied: true, Status: currentStatus, Message: "already applied"}, nil
		}
		return &MDMActivityResult{Applied: false, Status: currentStatus}, nil
	}
	return nil, fmt.Errorf("merge request %s not found", input.RequestID)
}

// RejectMergeActivity marks the merge request as REJECTED with CAS idempotency.
func (a *MDMApprovalActivities) RejectMergeActivity(ctx context.Context, input MDMRejectMergeInput) (*MDMActivityResult, error) {
	if a.DB == nil {
		return &MDMActivityResult{Applied: false, Status: "REJECTED"}, nil
	}
	res, err := a.DB.ExecContext(ctx, `
		UPDATE mdm.golden_merge_request
		SET status = 'REJECTED', decided_at = now()
		WHERE id = $1::uuid AND tenant_id = $2::uuid AND status = 'PENDING'
	`, input.RequestID, input.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to reject merge request %s: %w", input.RequestID, err)
	}
	rows, _ := res.RowsAffected()
	if rows == 1 {
		return &MDMActivityResult{Applied: false, Status: "REJECTED"}, nil
	}
	var currentStatus string
	if err := a.DB.GetContext(ctx, &currentStatus, `SELECT status FROM mdm.golden_merge_request WHERE id = $1::uuid AND tenant_id = $2::uuid`, input.RequestID, input.TenantID); err == nil {
		return &MDMActivityResult{Applied: false, Status: currentStatus}, nil
	}
	return nil, fmt.Errorf("merge request %s not found", input.RequestID)
}

// ApplyConfigChangeActivity updates the status of the configuration change with CAS idempotency.
func (a *MDMApprovalActivities) ApplyConfigChangeActivity(ctx context.Context, input MDMApplyConfigInput) (*MDMActivityResult, error) {
	if a.DB == nil {
		return &MDMActivityResult{Applied: true, Status: "applied"}, nil
	}
	res, err := a.DB.ExecContext(ctx, `
		UPDATE mdm.mastering_config_change
		SET status = 'applied', reviewed_by = $3::text, reviewed_by_name = NULLIF($4::text, ''),
		    reviewed_at = now(), review_comment = NULLIF($5, '')
		WHERE id = $1::uuid AND tenant_id = $2::uuid AND status = 'pending'
	`, input.ChangeID, input.TenantID, input.ApproverID, input.ApproverName, strings.TrimSpace(input.Comment))
	if err != nil {
		return nil, fmt.Errorf("failed to apply config change %s: %w", input.ChangeID, err)
	}
	rows, _ := res.RowsAffected()
	if rows == 1 {
		return &MDMActivityResult{Applied: true, Status: "applied"}, nil
	}
	var currentStatus string
	if err := a.DB.GetContext(ctx, &currentStatus, `SELECT status FROM mdm.mastering_config_change WHERE id = $1::uuid AND tenant_id = $2::uuid`, input.ChangeID, input.TenantID); err == nil {
		if currentStatus == "applied" {
			return &MDMActivityResult{Applied: true, Status: "applied", Message: "already applied"}, nil
		}
		return &MDMActivityResult{Applied: false, Status: currentStatus}, nil
	}
	return nil, fmt.Errorf("config change %s not found", input.ChangeID)
}

// RejectConfigChangeActivity updates the status of the configuration change to rejected with CAS idempotency.
func (a *MDMApprovalActivities) RejectConfigChangeActivity(ctx context.Context, input MDMRejectConfigInput) (*MDMActivityResult, error) {
	if a.DB == nil {
		return &MDMActivityResult{Applied: false, Status: "rejected"}, nil
	}
	res, err := a.DB.ExecContext(ctx, `
		UPDATE mdm.mastering_config_change
		SET status = 'rejected', reviewed_by = $3::text, reviewed_by_name = NULLIF($4::text, ''),
		    reviewed_at = now(), review_comment = NULLIF($5, '')
		WHERE id = $1::uuid AND tenant_id = $2::uuid AND status = 'pending'
	`, input.ChangeID, input.TenantID, input.RejecterID, input.RejecterName, strings.TrimSpace(input.Reason))
	if err != nil {
		return nil, fmt.Errorf("failed to reject config change %s: %w", input.ChangeID, err)
	}
	rows, _ := res.RowsAffected()
	if rows == 1 {
		return &MDMActivityResult{Applied: false, Status: "rejected"}, nil
	}
	var currentStatus string
	if err := a.DB.GetContext(ctx, &currentStatus, `SELECT status FROM mdm.mastering_config_change WHERE id = $1::uuid AND tenant_id = $2::uuid`, input.ChangeID, input.TenantID); err == nil {
		return &MDMActivityResult{Applied: false, Status: currentStatus}, nil
	}
	return nil, fmt.Errorf("config change %s not found", input.ChangeID)
}

// WithdrawConfigChangeActivity updates the status of the configuration change to withdrawn with CAS idempotency.
func (a *MDMApprovalActivities) WithdrawConfigChangeActivity(ctx context.Context, input MDMWithdrawConfigInput) (*MDMActivityResult, error) {
	if a.DB == nil {
		return &MDMActivityResult{Applied: false, Status: "withdrawn"}, nil
	}
	res, err := a.DB.ExecContext(ctx, `
		UPDATE mdm.mastering_config_change
		SET status = 'withdrawn', reviewed_at = now(), review_comment = NULLIF($3, '')
		WHERE id = $1::uuid AND tenant_id = $2::uuid AND status = 'pending'
	`, input.ChangeID, input.TenantID, strings.TrimSpace(input.Reason))
	if err != nil {
		return nil, fmt.Errorf("failed to withdraw config change %s: %w", input.ChangeID, err)
	}
	rows, _ := res.RowsAffected()
	if rows == 1 {
		return &MDMActivityResult{Applied: false, Status: "withdrawn"}, nil
	}
	var currentStatus string
	if err := a.DB.GetContext(ctx, &currentStatus, `SELECT status FROM mdm.mastering_config_change WHERE id = $1::uuid AND tenant_id = $2::uuid`, input.ChangeID, input.TenantID); err == nil {
		return &MDMActivityResult{Applied: false, Status: currentStatus}, nil
	}
	return nil, fmt.Errorf("config change %s not found", input.ChangeID)
}

// EscalateMDMReviewActivity logs an SLA escalation warning and emits an alert.
func (a *MDMApprovalActivities) EscalateMDMReviewActivity(ctx context.Context, input MDMEscalateInput) error {
	if a.Logger != nil {
		a.Logger.Warnf("[MDMGovernance SLA Breach] Target %s (%s) for tenant %s exceeded review SLA: %s",
			input.TargetID, input.Kind, input.TenantID, input.Reason)
	}
	return nil
}

// ReconcilePendingMDMWorkflowsActivity finds pending records that lack a temporal workflow correlation,
// links active ones, and auto-expires proposals that have exceeded ExpiryAge.
func (a *MDMApprovalActivities) ReconcilePendingMDMWorkflowsActivity(ctx context.Context, input MDMReconciliationInput) (*MDMReconciliationResult, error) {
	if a.DB == nil {
		return &MDMReconciliationResult{}, nil
	}

	maxPendingAge := input.MaxPendingAge
	if maxPendingAge <= 0 {
		maxPendingAge = 5 * time.Minute
	}
	expiryAge := input.ExpiryAge
	if expiryAge <= 0 {
		expiryAge = 168 * time.Hour // 7 days
	}

	now := time.Now().UTC()
	pendingCutoff := now.Add(-maxPendingAge)
	expiryCutoff := now.Add(-expiryAge)

	res := &MDMReconciliationResult{}

	// 1. Overrides: auto-expire stale ones past ExpiryAge
	expOverrides, _ := a.DB.ExecContext(ctx, `
		UPDATE mdm.golden_override
		SET status = 'EXPIRED', ended_at = now()
		WHERE status = 'PENDING' AND requested_at <= $1
	`, expiryCutoff)
	if n, err := expOverrides.RowsAffected(); err == nil {
		res.OverridesExpired = int(n)
	}

	// Overrides: link unlinked pending ones
	var unlinkedOverrides []struct {
		ID       string `db:"id"`
		TenantID string `db:"tenant_id"`
		Entity   string `db:"entity_cd"`
	}
	err := a.DB.SelectContext(ctx, &unlinkedOverrides, `
		SELECT id::text, tenant_id::text, entity_cd
		FROM mdm.golden_override
		WHERE status = 'PENDING' AND (temporal_workflow_id IS NULL OR temporal_workflow_id = '')
		  AND requested_at <= $1 AND requested_at > $2
	`, pendingCutoff, expiryCutoff)
	if err == nil {
		for _, o := range unlinkedOverrides {
			wfID := fmt.Sprintf("mdm-override/%s/%s/%s", o.TenantID, o.Entity, o.ID)
			_, _ = a.DB.ExecContext(ctx, `UPDATE mdm.golden_override SET temporal_workflow_id = $1 WHERE id = $2::uuid`, wfID, o.ID)
			res.OverridesLinked++
		}
	}

	// 2. Merges: auto-expire stale ones
	expMerges, _ := a.DB.ExecContext(ctx, `
		UPDATE mdm.golden_merge_request
		SET status = 'EXPIRED', decided_at = now()
		WHERE status = 'PENDING' AND requested_at <= $1
	`, expiryCutoff)
	if n, err := expMerges.RowsAffected(); err == nil {
		res.MergesExpired = int(n)
	}

	var unlinkedMerges []struct {
		ID       string `db:"id"`
		TenantID string `db:"tenant_id"`
		Entity   string `db:"entity_cd"`
	}
	err = a.DB.SelectContext(ctx, &unlinkedMerges, `
		SELECT id::text, tenant_id::text, entity_cd
		FROM mdm.golden_merge_request
		WHERE status = 'PENDING' AND (temporal_workflow_id IS NULL OR temporal_workflow_id = '')
		  AND requested_at <= $1 AND requested_at > $2
	`, pendingCutoff, expiryCutoff)
	if err == nil {
		for _, m := range unlinkedMerges {
			wfID := fmt.Sprintf("mdm-merge/%s/%s/%s", m.TenantID, m.Entity, m.ID)
			_, _ = a.DB.ExecContext(ctx, `UPDATE mdm.golden_merge_request SET temporal_workflow_id = $1 WHERE id = $2::uuid`, wfID, m.ID)
			res.MergesLinked++
		}
	}

	// 3. Config Changes: auto-expire stale ones
	expConfigs, _ := a.DB.ExecContext(ctx, `
		UPDATE mdm.mastering_config_change
		SET status = 'expired', reviewed_at = now(), review_comment = 'Auto-expired by reconciliation sweep'
		WHERE status = 'pending' AND requested_at <= $1
	`, expiryCutoff)
	if n, err := expConfigs.RowsAffected(); err == nil {
		res.ConfigsExpired = int(n)
	}

	var unlinkedConfigs []struct {
		ID       string `db:"id"`
		TenantID string `db:"tenant_id"`
	}
	err = a.DB.SelectContext(ctx, &unlinkedConfigs, `
		SELECT id::text, tenant_id::text
		FROM mdm.mastering_config_change
		WHERE status = 'pending' AND (temporal_workflow_id IS NULL OR temporal_workflow_id = '')
		  AND requested_at <= $1 AND requested_at > $2
	`, pendingCutoff, expiryCutoff)
	if err == nil {
		for _, c := range unlinkedConfigs {
			wfID := fmt.Sprintf("mdm-config/%s/%s", c.TenantID, c.ID)
			_, _ = a.DB.ExecContext(ctx, `UPDATE mdm.mastering_config_change SET temporal_workflow_id = $1 WHERE id = $2::uuid`, wfID, c.ID)
			res.ConfigsLinked++
		}
	}

	if a.Logger != nil {
		a.Logger.Infof("[MDMGovernance Reconciliation] Overrides (linked: %d, expired: %d), Merges (linked: %d, expired: %d), Configs (linked: %d, expired: %d)",
			res.OverridesLinked, res.OverridesExpired, res.MergesLinked, res.MergesExpired, res.ConfigsLinked, res.ConfigsExpired)
	}
	return res, nil
}
