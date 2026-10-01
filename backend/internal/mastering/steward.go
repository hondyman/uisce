package mastering

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Decision on a possible duplicate (a match candidate).
type CandidateDecision struct {
	// Merge: the two are the same; else they are different (reject).
	Merge bool `json:"merge"`
	// Keep is which record survives a merge: "a" (the existing one, the
	// default) or "b" (the newer one).
	Keep string `json:"keep,omitempty"`
	Note string `json:"note,omitempty"`
}

// DecisionResult is what a steward's decision did.
type DecisionResult struct {
	Candidate string `json:"candidate_id"`
	Status    string `json:"status"` // APPROVED (merged) | REJECTED | PENDING_APPROVAL (a merge request)
	// MergeRequest is the request awaiting approval (PENDING_APPROVAL).
	MergeRequest string `json:"merge_request_id,omitempty"`
	Survivor     string `json:"survivor_id,omitempty"`
	Merged       string `json:"merged_id,omitempty"`
	Moved        struct {
		Sources     int `json:"sources"`
		Identifiers int `json:"identifiers"`
	} `json:"moved"`
	Published bool `json:"published"`
}

// DecideCandidate records a steward's decision on a possible duplicate.
//
// Reject: the pair is closed as different; both golden records stand.
//
// Merge: the survivor takes over the other record's source links and
// identifiers, the other is marked merged into it (its golden version
// retracted), and the survivor is re-survived from all its sources, so the
// new version's provenance shows both sides. One transaction, logged in
// mdm.<prefix>_merge_log with what is needed to reverse it.
func (e *Engine) DecideCandidate(ctx context.Context, tenantID, entity, id string, d CandidateDecision, actorID, actorName string) (*DecisionResult, error) {
	cfg, err := e.loadConfig(ctx, tenantID, entity)
	if err != nil {
		return nil, err
	}
	var pol *Policy
	if err := e.inConfig(ctx, tenantID, func(tx *sqlx.Tx) error {
		pol, err = getPolicy(ctx, tx, tenantID, entity)
		return err
	}); err != nil {
		return nil, err
	}
	var res *DecisionResult
	err = e.inTenant(ctx, tenantID, func(tx *sqlx.Tx) error {
		var err error
		res, err = e.decide(ctx, tx, cfg, pol, tenantID, entity, id, d, actorID, actorName)
		return err
	})
	return res, err
}

func (e *Engine) decide(ctx context.Context, tx *sqlx.Tx, cfg *config, pol *Policy, tenantID, entity, id string, d CandidateDecision, actorID, actorName string) (*DecisionResult, error) {
	p := cfg.profile
	res := &DecisionResult{Candidate: id}
	err := func() error {
		var c struct {
			A      string `db:"a"`
			B      string `db:"b"`
			Status string `db:"status"`
		}
		err := tx.GetContext(ctx, &c, fmt.Sprintf(`SELECT %s::text AS a, %s::text AS b, status FROM %s WHERE id::text = $1 FOR UPDATE`,
			qi(p.TablePrefix+"_id_a"), qi(p.TablePrefix+"_id_b"), p.table("match_candidate")), id)
		if errors.Is(err, sql.ErrNoRows) {
			return msgNoCandidate(id)
		}
		if err != nil {
			return err
		}
		if c.Status != "PENDING" && c.Status != "DEFERRED" {
			return msgCandidateDecided(id, c.Status)
		}
		note := strings.TrimSpace(d.Note)
		reviewer := nullUUID(actorID)
		review := func(status string) error {
			_, err := tx.ExecContext(ctx, fmt.Sprintf(`UPDATE %s SET status = $2, reviewed_by = $3::uuid, reviewed_at = now(),
					review_note = NULLIF($4, ''), custom_attributes = custom_attributes || jsonb_build_object('reviewed_by_name', $5::text)
				WHERE id::text = $1`, p.table("match_candidate")), id, status, reviewer, note, actorName)
			return err
		}
		if !d.Merge {
			res.Status = "REJECTED"
			// A merge waiting for approval on this pair is overruled.
			var hasReq bool
			if err := tx.GetContext(ctx, &hasReq, `SELECT to_regclass('mdm.golden_merge_request') IS NOT NULL`); err != nil {
				return err
			}
			if hasReq {
				if _, err := tx.ExecContext(ctx, `UPDATE mdm.golden_merge_request SET status = 'REJECTED', decided_at = now()
					WHERE candidate_id::text = $1 AND status = 'PENDING'`, id); err != nil {
					return err
				}
			}
			return review("REJECTED")
		}

		// A merge follows the entity's policy: under APPROVAL it becomes a
		// request others approve; under DIRECT it happens now.
		if pol.required("") > 0 {
			return e.requestMerge(ctx, tx, p, tenantID, id, d, actorID, actorName, pol.required(""), res)
		}
		return e.applyMerge(ctx, tx, cfg, tenantID, entity, id, c.A, c.B, d.Keep, note, actorID, actorName, "", res)
	}()
	return res, err
}

// applyMerge merges the pair: requester asked for it, approvers (if any)
// approved it; both are recorded on the merge log.
func (e *Engine) applyMerge(ctx context.Context, tx *sqlx.Tx, cfg *config, tenantID, entity, candidateID, a, b, keep, note, requesterID, requesterName, approvers string, res *DecisionResult) error {
	p := cfg.profile
	survivor, merged := a, b
	if strings.EqualFold(keep, "b") {
		survivor, merged = b, a
	}
	if survivor == merged {
		return msgCandidateDecided(candidateID, "SAME_RECORD")
	}
	res.Status, res.Survivor, res.Merged = "APPROVED", survivor, merged
	reviewer := nullUUID(requesterID)
	reviewNote := note
	if approvers != "" {
		reviewNote = strings.TrimSpace(note + " (approved by " + approvers + ")")
	}

	r := &runner{e: e, tx: tx, ctx: ctx, tenant: tenantID, cfg: cfg, p: p, run: &Run{ID: uuid.NewString()},
		req: RunRequest{Entity: entity, StartedByID: requesterID, StartedBy: requesterName}, counts: &Counts{}, stage: new(string)}
	if _, err := r.prepare(); err != nil {
		return err
	}
	r.force = true
	reversal, err := r.mergeInto(survivor, merged, res)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`UPDATE %s SET status = 'APPROVED', reviewed_by = $2::uuid, reviewed_at = now(),
			review_note = NULLIF($3, ''), custom_attributes = custom_attributes || jsonb_build_object('reviewed_by_name', $4::text, 'approved_by', $5::text)
		WHERE id::text = $1`, p.table("match_candidate")), candidateID, reviewer, reviewNote, requesterName, approvers); err != nil {
		return err
	}
	// Other open pairs with the merged record now concern the survivor.
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`UPDATE %[1]s SET status = 'REJECTED', reviewed_by = $3::uuid, reviewed_at = now(),
			review_note = 'Merged into ' || $2 || ' by candidate ' || $4
		WHERE status IN ('PENDING', 'DEFERRED') AND id::text <> $4 AND (%[2]s::text = $1 OR %[3]s::text = $1)`,
		p.table("match_candidate"), qi(p.TablePrefix+"_id_a"), qi(p.TablePrefix+"_id_b")), merged, survivor, reviewer, candidateID); err != nil {
		return err
	}
	raw, _ := json.Marshal(reversal)
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`INSERT INTO %s (tenant_id, %s, %s, merge_type, match_candidate_id, merge_reason,
			merged_by, reversible, reversal_data, custom_attributes)
		VALUES ($1::uuid, $2::uuid, $3::uuid, 'STEWARD', $4::uuid, NULLIF($5, ''), $6::uuid, true, $7,
			jsonb_build_object('merged_by_name', $8::text, 'approved_by', NULLIF($9, '')))`,
		p.table("merge_log"), qi("surviving_"+p.TablePrefix+"_id"), qi("merged_"+p.TablePrefix+"_id")),
		tenantID, survivor, merged, candidateID, truncate(note, 255), reviewer, raw, requesterName, approvers); err != nil {
		return err
	}
	// A new version from every source now linked to the survivor.
	if err := r.masterOne(survivor); err != nil {
		return err
	}
	res.Published = r.counts.Published > 0
	return nil
}

// mergeReversal is what unmerging needs: which links and identifiers moved.
type mergeReversal struct {
	Xrefs       []string `json:"xref_ids"`       // retired links of the merged record
	NewXrefs    []string `json:"new_xref_ids"`   // their replacements on the survivor
	Identifiers []string `json:"identifier_ids"` // retired identifiers of the merged record
	NewIdents   []string `json:"new_identifier_ids"`
}

// mergeInto moves the merged record's source links and identifiers to the
// survivor, retires its golden version and marks it merged.
func (r *runner) mergeInto(survivor, merged string, res *DecisionResult) (*mergeReversal, error) {
	ctx, tx, p := r.ctx, r.tx, r.p
	rev := &mergeReversal{}

	// Source links: retire each, re-link it to the survivor as a steward link.
	var links []struct {
		ID        string `db:"id"`
		SourceID  string `db:"source_system_id"`
		SourceKey string `db:"source_key"`
	}
	if err := tx.SelectContext(ctx, &links, `SELECT id::text, source_system_id::text, source_key FROM mdm.entity_xref
		WHERE entity_cd = $1 AND golden_id::text = $2 AND status = 'ACTIVE'`, p.EntityCd, merged); err != nil {
		return nil, err
	}
	for _, l := range links {
		if _, err := tx.ExecContext(ctx, `UPDATE mdm.entity_xref SET status = 'RETIRED', updated_at = now() WHERE id::text = $1`, l.ID); err != nil {
			return nil, err
		}
		var nid string
		if err := tx.GetContext(ctx, &nid, `INSERT INTO mdm.entity_xref
			(tenant_id, entity_cd, source_system_id, source_key, golden_id, match_method, match_score, matched_keys)
			VALUES ($1::uuid, $2, $3::uuid, $4, $5::uuid, 'STEWARD', 1, '["steward"]'::jsonb) RETURNING id::text`,
			r.tenant, p.EntityCd, l.SourceID, l.SourceKey, survivor); err != nil {
			return nil, err
		}
		rev.Xrefs = append(rev.Xrefs, l.ID)
		rev.NewXrefs = append(rev.NewXrefs, nid)
	}
	res.Moved.Sources = len(links)

	// Identifiers: the survivor gets any it doesn't hold; the merged
	// record's are ended either way (an identifier names one record).
	if p.IdentifierTable != nil {
		id := p.ident()
		var ids []struct {
			ID     string `db:"id"`
			Type   string `db:"id_type"`
			Value  string `db:"id_value"`
			Source string `db:"source"`
		}
		if err := tx.SelectContext(ctx, &ids, fmt.Sprintf(`SELECT id::text, %s AS id_type, %s AS id_value, COALESCE(%s::text, '') AS source FROM %s
			WHERE %s::text = $1 AND %s`, qi(id.TypeColumn), qi(id.ValueColumn), qi(id.SourceColumn), qi(*p.IdentifierTable),
			qi(id.KeyColumn), id.identActive("")), merged); err != nil {
			return nil, err
		}
		for _, i := range ids {
			if _, err := tx.ExecContext(ctx, fmt.Sprintf(`UPDATE %s SET %s WHERE id::text = $1`, qi(*p.IdentifierTable), id.identRetire()), i.ID); err != nil {
				return nil, err
			}
			rev.Identifiers = append(rev.Identifiers, i.ID)
			var held bool
			if err := tx.GetContext(ctx, &held, fmt.Sprintf(`SELECT EXISTS (SELECT 1 FROM %s WHERE %s::text = $1 AND %s = $2
				AND %s = $3 AND %s)`, qi(*p.IdentifierTable), qi(id.KeyColumn), qi(id.TypeColumn), qi(id.ValueColumn), id.identActive("")),
				survivor, i.Type, i.Value); err != nil {
				return nil, err
			}
			if held {
				continue
			}
			// Re-recorded against the survivor, as the same source reported it.
			args := []any{r.tenant, survivor, i.Type, i.Value, nullIfEmpty(i.Source)}
			if id.PrimaryColumn != "" {
				args = append(args, false)
			}
			var nid string
			if err := tx.GetContext(ctx, &nid, r.identInsert()+" RETURNING id::text", args...); err != nil {
				return nil, err
			}
			rev.NewIdents = append(rev.NewIdents, nid)
			res.Moved.Identifiers++
		}
	}

	// The merged record's golden version is retracted.
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`UPDATE %s SET is_current = false, status = 'RETRACTED'
		WHERE %s::text = $1 AND status IN ('PUBLISHED', 'REVIEW', 'DRAFT')`, p.table("golden_record"), p.keyColumn()), merged); err != nil {
		return nil, err
	}

	// The anchor row points at its survivor and, where the entity has a
	// status reference with a MERGED code, is marked merged.
	set := map[string]any{}
	if _, ok := r.anchorCols["merged_into_id"]; ok {
		set["merged_into_id"] = survivor
	}
	if _, ok := r.anchorCols["is_golden_record"]; ok {
		set["is_golden_record"] = false
	}
	if _, ok := r.anchorCols["is_active"]; ok {
		set["is_active"] = false
	}
	for _, ref := range p.Settings.References {
		if strings.HasPrefix(ref.Column, "status") {
			if id, ok := r.refIDs[ref.Attribute]["MERGED"]; ok {
				set[ref.Column] = id
			}
		}
	}
	for c, v := range p.Settings.MergedValues {
		if _, ok := r.anchorCols[c]; ok {
			set[c] = v
		}
	}
	if err := r.writeAnchor(merged, set); err != nil {
		return nil, err
	}
	return rev, nil
}
