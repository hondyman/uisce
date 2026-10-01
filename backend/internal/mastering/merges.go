package mastering

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
)

// requestMerge records a steward's merge as a request awaiting approval
// (the entity's policy is APPROVAL).
func (e *Engine) requestMerge(ctx context.Context, tx *sqlx.Tx, p *Profile, tenantID, candidateID string, d CandidateDecision, actorID, actorName string, need int, res *DecisionResult) error {
	var ok bool
	if err := tx.GetContext(ctx, &ok, `SELECT to_regclass('mdm.golden_merge_request') IS NOT NULL`); err != nil {
		return err
	}
	if !ok {
		return msgMergeApprovalsUnavailable()
	}
	keep := "a"
	if strings.EqualFold(d.Keep, "b") {
		keep = "b"
	}
	var id string
	err := tx.GetContext(ctx, &id, `INSERT INTO mdm.golden_merge_request (tenant_id, entity_cd, candidate_id, keep, note, approvals_required,
			requested_by, requested_by_name)
		VALUES ($1::uuid, $2, $3::uuid, $4, NULLIF($5, ''), $6, $7, NULLIF($8, '')) RETURNING id::text`,
		tenantID, p.EntityCd, candidateID, keep, strings.TrimSpace(d.Note), need, actorID, actorName)
	if isUniqueViolation(err) {
		return msgMergePending()
	}
	if err != nil {
		return err
	}
	res.Status, res.MergeRequest = "PENDING_APPROVAL", id
	return nil
}

// DecideMerge records one approver's vote on a merge request: one
// rejection ends it (the pair goes back to the review list); the Nth
// distinct approval performs the merge. The requester never votes.
func (e *Engine) DecideMerge(ctx context.Context, tenantID, entity, requestID string, approve bool, comment, actorID, actorName string) (*DecisionResult, error) {
	cfg, err := e.loadConfig(ctx, tenantID, entity)
	if err != nil {
		return nil, err
	}
	var res *DecisionResult
	err = e.inTenant(ctx, tenantID, func(tx *sqlx.Tx) error {
		var err error
		res, err = e.decideMerge(ctx, tx, cfg, tenantID, entity, requestID, approve, comment, actorID, actorName)
		return err
	})
	return res, err
}

func (e *Engine) decideMerge(ctx context.Context, tx *sqlx.Tx, cfg *config, tenantID, entity, requestID string, approve bool, comment, actorID, actorName string) (*DecisionResult, error) {
	p := cfg.profile
	var m struct {
		Candidate   string         `db:"candidate_id"`
		Keep        string         `db:"keep"`
		Note        sql.NullString `db:"note"`
		Need        int            `db:"approvals_required"`
		Status      string         `db:"status"`
		RequestedBy string         `db:"requested_by"`
		ByName      sql.NullString `db:"requested_by_name"`
	}
	err := tx.GetContext(ctx, &m, `SELECT candidate_id::text, keep, note, approvals_required, status, requested_by, requested_by_name
		FROM mdm.golden_merge_request WHERE id::text = $1 AND entity_cd = $2 FOR UPDATE`, requestID, p.EntityCd)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, msgNoMergeRequest(requestID)
	}
	if err != nil {
		return nil, err
	}
	if m.Status != "PENDING" {
		return nil, msgMergeDecided(m.Status)
	}
	if m.RequestedBy == actorID {
		return nil, msgOwnMerge()
	}
	decision := "REJECT"
	if approve {
		decision = "APPROVE"
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO mdm.golden_merge_vote (tenant_id, request_id, approver, approver_name, decision, comment)
		VALUES ($1::uuid, $2::uuid, $3, NULLIF($4, ''), $5, NULLIF($6, ''))`, tenantID, requestID, actorID, actorName, decision, strings.TrimSpace(comment)); err != nil {
		if isUniqueViolation(err) {
			return nil, msgMergeVoted()
		}
		return nil, err
	}
	res := &DecisionResult{Candidate: m.Candidate, MergeRequest: requestID}
	if !approve {
		_, err := tx.ExecContext(ctx, `UPDATE mdm.golden_merge_request SET status = 'REJECTED', decided_at = now() WHERE id::text = $1`, requestID)
		res.Status = "MERGE_REJECTED"
		return res, err
	}
	var approvers []string
	if err := tx.SelectContext(ctx, &approvers, `SELECT COALESCE(approver_name, approver) FROM mdm.golden_merge_vote
		WHERE request_id::text = $1 AND decision = 'APPROVE' ORDER BY decided_at`, requestID); err != nil {
		return nil, err
	}
	if len(approvers) < m.Need {
		res.Status = "PENDING_APPROVAL"
		return res, nil
	}
	var pair struct {
		A      string `db:"a"`
		B      string `db:"b"`
		Status string `db:"status"`
	}
	if err := tx.GetContext(ctx, &pair, fmt.Sprintf(`SELECT %s::text AS a, %s::text AS b, status FROM %s WHERE id::text = $1 FOR UPDATE`,
		qi(p.TablePrefix+"_id_a"), qi(p.TablePrefix+"_id_b"), p.table("match_candidate")), m.Candidate); err != nil {
		return nil, err
	}
	if pair.Status != "PENDING" && pair.Status != "DEFERRED" {
		return nil, msgCandidateDecided(m.Candidate, pair.Status)
	}
	if err := e.applyMerge(ctx, tx, cfg, tenantID, entity, m.Candidate, pair.A, pair.B, m.Keep, m.Note.String, m.RequestedBy, m.ByName.String,
		strings.Join(approvers, ", "), res); err != nil {
		return nil, err
	}
	_, err = tx.ExecContext(ctx, `UPDATE mdm.golden_merge_request SET status = 'APPLIED', decided_at = now() WHERE id::text = $1`, requestID)
	return res, err
}

// WithdrawMerge: the requester takes back a pending merge request.
func (e *Engine) WithdrawMerge(ctx context.Context, tenantID, entity, requestID, actorID string) error {
	p, err := e.profile(ctx, tenantID, entity)
	if err != nil {
		return err
	}
	return e.inTenant(ctx, tenantID, func(tx *sqlx.Tx) error {
		res, err := tx.ExecContext(ctx, `UPDATE mdm.golden_merge_request SET status = 'WITHDRAWN', decided_at = now()
			WHERE id::text = $1 AND entity_cd = $2 AND status = 'PENDING' AND requested_by = $3`, requestID, p.EntityCd, actorID)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return msgNotMergeRequester()
		}
		return nil
	})
}
