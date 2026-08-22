package drift

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/audit"
	"github.com/jmoiron/sqlx"
)

type PatchService struct {
	db        *sqlx.DB
	outboxMgr *audit.TransactionalOutboxManager
}

func NewPatchService(db *sqlx.DB, outboxMgr *audit.TransactionalOutboxManager) *PatchService {
	return &PatchService{
		db:        db,
		outboxMgr: outboxMgr,
	}
}

// ApplyHotSwapPatch updates field bindings atomically and recalculates BO binding status
func (s *PatchService) ApplyHotSwapPatch(
	ctx context.Context,
	tenantID, proposalID uuid.UUID,
	appliedByUserID string,
) error {
	if tenantID == uuid.Nil {
		return fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var prop struct {
		BOID                 uuid.UUID `db:"bo_id"`
		BindingID            uuid.UUID `db:"binding_id"`
		FieldID              uuid.UUID `db:"field_id"`
		FieldName            string    `db:"field_name"`
		ProposedSourceNodeID uuid.UUID `db:"proposed_source_node_id"`
		ProposedColumnName   string    `db:"proposed_column_name"`
		ConfidenceScore      float64   `db:"confidence_score"`
		Status               string    `db:"status"`
	}

	fetchQuery := `
		SELECT bo_id, binding_id, field_id, field_name, proposed_source_node_id, 
		       proposed_column_name, confidence_score, status
		FROM catalog_drift.schema_drift_proposals
		WHERE proposal_id = $1 AND tenant_id = $2 FOR UPDATE;
	`
	err = tx.GetContext(ctx, &prop, fetchQuery, proposalID, tenantID)
	if err != nil {
		return fmt.Errorf("proposal not found: %w", err)
	}

	if prop.Status != "PENDING" {
		return fmt.Errorf("proposal is already in %s status", prop.Status)
	}

	updateBindingSQL := `
		UPDATE public.field_bindings
		SET source_node_id = $1,
		    binding_status = 'RESOLVED',
		    updated_at = NOW()
		WHERE bo_id = $2 AND binding_id = $3 AND field_id = $4 AND tenant_id = $5;
	`
	_, err = tx.ExecContext(ctx, updateBindingSQL, prop.ProposedSourceNodeID, prop.BOID, prop.BindingID, prop.FieldID, tenantID)
	if err != nil {
		return fmt.Errorf("failed hot-swapping field binding: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE catalog_drift.schema_drift_proposals
		SET status = 'APPLIED', applied_by_user_id = $1, applied_at = NOW()
		WHERE proposal_id = $2 AND tenant_id = $3;
	`, appliedByUserID, proposalID, tenantID)
	if err != nil {
		return err
	}

	if s.outboxMgr != nil {
		auditPayload := map[string]interface{}{
			"proposal_id":           proposalID.String(),
			"bo_id":                 prop.BOID.String(),
			"binding_id":            prop.BindingID.String(),
			"field_id":              prop.FieldID.String(),
			"field_name":            prop.FieldName,
			"new_source_node_id":    prop.ProposedSourceNodeID.String(),
			"new_column_name":       prop.ProposedColumnName,
			"confidence_score":      prop.ConfidenceScore,
			"applied_by_user":       appliedByUserID,
			"patched_timestamp_utc": time.Now().UTC().Format(time.RFC3339),
		}
		_ = s.outboxMgr.StageOutboxEventAtomic(
			ctx, tx, tenantID, proposalID,
			"CATALOG_DRIFT", "SCHEMA_HOTSWAP_APPLIED",
			"SCHEMA_SENTINEL", appliedByUserID, auditPayload,
		)
	}

	return tx.Commit()
}
