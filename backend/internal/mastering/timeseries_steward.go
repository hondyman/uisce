package mastering

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// Steward decisions on golden prices use the entity's override policy
// (approval by N others, or direct), the same requests, votes and audit as
// record overrides (mdm.golden_override). A price override is keyed by the
// instrument (golden_id) and "<PRICE TYPE>@<date>" (attribute), so a key
// has at most one pending and one active override.
//
// Applied, a SET is recorded as an observation from the MANUAL source and
// wins survivorship for its key, whatever the controls say - that is how a
// steward releases a held price (at its held value) or corrects one; the
// key's open exceptions are resolved by the decision. A CLEAR ends it and
// the vendors' prices survive again.

const manualSource = "MANUAL"

func priceAttr(priceType, date string) string { return priceType + "@" + date }

func parsePriceAttr(a string) (priceType, date string, ok bool) {
	priceType, date, ok = strings.Cut(a, "@")
	if !ok || priceType == "" {
		return "", "", false
	}
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return "", "", false
	}
	return priceType, date, true
}

// priceKeyOf is the key of a golden price version (by its id).
func priceKeyOf(ctx context.Context, tx *sqlx.Tx, id string) (seriesKey, sql.NullFloat64, error) {
	var k struct {
		EntityType string          `db:"price_entity_type"`
		EntityID   string          `db:"entity_id"`
		PriceType  string          `db:"price_type_cd"`
		Date       string          `db:"price_date"`
		Value      sql.NullFloat64 `db:"latest"`
	}
	err := tx.GetContext(ctx, &k, `SELECT g.price_entity_type, g.price_entity_id::text AS entity_id, g.price_type_cd, g.price_date::text,
			(SELECT l.golden_value::float8 FROM mdm.price_golden_record l WHERE l.price_entity_type = g.price_entity_type
				AND l.price_entity_id = g.price_entity_id AND l.price_type_cd = g.price_type_cd AND l.price_date = g.price_date
				ORDER BY l.golden_version DESC LIMIT 1) AS latest
		FROM mdm.price_golden_record g WHERE g.id::text = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return seriesKey{}, k.Value, msgNoGolden(id)
	}
	return seriesKey{k.EntityType, k.EntityID, k.PriceType, k.Date}, k.Value, err
}

// proposePriceOverride records a steward's price for a key (priceID is any
// version of the golden price).
func (e *Engine) proposePriceOverride(ctx context.Context, tx *sqlx.Tx, cfg *config, pol *Policy, tenantID, entity, priceID string,
	in OverrideRequest, actorID, actorName string) (*Override, error) {
	k, latest, err := priceKeyOf(ctx, tx, priceID)
	if err != nil {
		return nil, err
	}
	attr := priceAttr(k.PriceType, k.Date)
	if in.Attribute != "" && in.Attribute != attr {
		return nil, msgUnknownAttribute(in.Attribute)
	}
	var val any
	if in.Action == "SET" {
		v, ok := priceValue(in.Value)
		if !ok {
			return nil, msgOverrideValue(attr)
		}
		b, _ := json.Marshal(v)
		val = b
	} else {
		var active int
		if err := tx.GetContext(ctx, &active, `SELECT count(*) FROM mdm.golden_override WHERE entity_cd = $1 AND golden_id::text = $2
			AND attribute = $3 AND active`, cfg.profile.EntityCd, k.EntityID, attr); err != nil {
			return nil, err
		}
		if active == 0 {
			return nil, msgNothingToClear(attr)
		}
	}
	var prev any
	if latest.Valid {
		prev = latest.Float64
	}
	pj, _ := json.Marshal(prev)
	need := pol.required(attr)
	var id string
	err = tx.GetContext(ctx, &id, `INSERT INTO mdm.golden_override (tenant_id, entity_cd, golden_id, attribute, action, value, previous_value,
			reason, mode, approvals_required, requested_by, requested_by_name)
		VALUES ($1::uuid, $2, $3::uuid, $4, $5, $6, $7, $8, $9, $10, $11, NULLIF($12, '')) RETURNING id::text`,
		tenantID, cfg.profile.EntityCd, k.EntityID, attr, in.Action, val, pj, in.Reason, pol.Mode, need, actorID, actorName)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, msgOverridePending(attr)
		}
		return nil, err
	}
	if need == 0 {
		r := e.stewardRunner(ctx, tx, cfg, tenantID, entity, actorID, actorName)
		if err := r.applyOverride(id); err != nil {
			return nil, err
		}
	}
	return getOverride(ctx, tx, cfg.profile, instrumentOf(cfg), id, actorID)
}

// priceValue: a number, as JSON or numeric text.
func priceValue(raw json.RawMessage) (float64, bool) {
	var v any
	if len(raw) == 0 || json.Unmarshal(raw, &v) != nil {
		return 0, false
	}
	switch x := v.(type) {
	case float64:
		return x, true
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(x), 64)
		return f, err == nil
	}
	return 0, false
}

func instrumentOf(cfg *config) *Profile {
	if cfg != nil && cfg.series != nil {
		return cfg.series.instrument
	}
	return nil
}

// applyPriceOverride makes an approved (or direct) price override take
// effect and re-masters its key.
func (r *runner) applyPriceOverride(id string) error {
	ctx, tx := r.ctx, r.tx
	var o struct {
		Entity string          `db:"golden_id"`
		Attr   string          `db:"attribute"`
		Action string          `db:"action"`
		Value  json.RawMessage `db:"value"`
		Reason string          `db:"reason"`
	}
	if err := tx.GetContext(ctx, &o, `SELECT golden_id::text, attribute, action, COALESCE(value, 'null'::jsonb) AS value, reason
		FROM mdm.golden_override WHERE id::text = $1`, id); err != nil {
		return err
	}
	priceType, date, ok := parsePriceAttr(o.Attr)
	if !ok {
		return msgUnknownAttribute(o.Attr)
	}
	manual := r.cfg.sources[manualSource]
	if manual == "" {
		return msgUnknownSource(manualSource)
	}
	k := seriesKey{EntityType: r.p.Settings.Series.EntityType, EntityID: o.Entity, PriceType: priceType, Date: date}
	if _, err := tx.ExecContext(ctx, `UPDATE mdm.golden_override SET active = false, ended_by_id = $2::uuid, ended_at = now()
		WHERE entity_cd = $1 AND golden_id::text = $3 AND attribute = $4 AND active`, r.p.EntityCd, id, o.Entity, o.Attr); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE mdm.golden_override SET status = 'APPLIED', applied_at = now(), active = (action = 'SET')
		WHERE id::text = $1`, id); err != nil {
		return err
	}

	// The steward's price is an observation from MANUAL: the previous one
	// (if any) is closed and kept in history, as a vendor restatement.
	keyArgs := []any{k.EntityType, k.EntityID, k.PriceType, k.Date, manual}
	const current = `p.price_entity_type = $1 AND p.price_entity_id::text = $2 AND p.price_type_id = (SELECT id FROM mdm.price_type WHERE price_type_cd = $3)
		AND p.price_date = $4::date AND p.source_id::text = $5 AND p.is_current`
	var closed []string
	if err := tx.SelectContext(ctx, &closed, `WITH old AS (SELECT p.* FROM mdm.price p WHERE `+current+`),
		hist AS (INSERT INTO mdm.price_history (tenant_id, price_id, price_entity_type, price_entity_id, price_type_id, source_id, version_num,
				valid_from, valid_to, is_current, record_snapshot, changed_columns, change_source, change_reason, custom_attributes)
			SELECT old.tenant_id, old.id, old.price_entity_type, old.price_entity_id, old.price_type_id, old.source_id,
				COALESCE((SELECT max(h.version_num) FROM mdm.price_history h WHERE h.price_id = old.id), 0) + 1,
				COALESCE(old.created_at, now()), now(), false, to_jsonb(old), ARRAY['value'], 'MANUAL', 'STEWARD_OVERRIDE',
				jsonb_build_object('override_id', $6::text)
			FROM old RETURNING 1)
		UPDATE mdm.price p SET is_current = false, updated_at = now() FROM old WHERE p.id = old.id RETURNING p.id::text`,
		append(keyArgs, id)...); err != nil {
		return err
	}
	if o.Action == "SET" {
		v, ok := priceValue(o.Value)
		if !ok {
			return msgOverrideValue(o.Attr)
		}
		var pred any
		if len(closed) > 0 {
			pred = closed[0]
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO mdm.price (tenant_id, price_entity_type, price_entity_id, price_type_id, source_id, price_date,
				price_time, value, is_official, is_executable, is_stale, is_indicative, is_adjusted, source_timestamp, source_system_id,
				predecessor_price_id, is_current, custom_attributes)
			SELECT $6::uuid, $1, $2::uuid, pt.id, $5::uuid, $4::date, now(), $7, false, false, false, false, false, now(), $5::uuid, $8::uuid, true,
				jsonb_build_object('override_id', $9::text, 'reason', $10::text, 'source_row_id', 'override:' || $9::text)
			FROM mdm.price_type pt WHERE pt.price_type_cd = $3`,
			k.EntityType, k.EntityID, k.PriceType, k.Date, manual, r.tenant, v, pred, id, o.Reason); err != nil {
			return err
		}
	}
	if o.Action == "CLEAR" {
		// Back to the vendors: the steward's version is no longer current
		// (if the vendors' price is still held, the key has none until a
		// steward or a new price decides it).
		if _, err := tx.ExecContext(ctx, `UPDATE mdm.price_golden_record SET is_current = false, status = 'SUPERSEDED'
			WHERE price_entity_type = $1 AND price_entity_id::text = $2 AND price_type_cd = $3 AND price_date = $4::date
			  AND is_current AND winning_sources->>'value' = $5`, k.EntityType, k.EntityID, k.PriceType, k.Date, manualSource); err != nil {
			return err
		}
	}
	// The steward has decided: the key's open exceptions are closed with the decision.
	if _, err := tx.ExecContext(ctx, `UPDATE mdm.price_exception SET status = 'RESOLVED', resolved_at = now(),
			resolution_note = 'Steward override ' || $5::text,
			custom_attributes = custom_attributes || jsonb_build_object('resolved_by', $6::text)
		WHERE status IN ('OPEN', 'IN_REVIEW') AND price_entity_type = $1 AND price_entity_id::text = $2 AND price_date = $4::date
		  AND custom_attributes->>'price_type' = $3`, k.EntityType, k.EntityID, k.PriceType, k.Date, id, r.req.StartedByID); err != nil {
		return err
	}
	r.force = true
	if err := r.publishSeries([]seriesKey{k}); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `UPDATE mdm.golden_override SET applied_version = (SELECT max(golden_version) FROM mdm.price_golden_record
			WHERE price_entity_type = $2 AND price_entity_id::text = $3 AND price_type_cd = $4 AND price_date = $5::date)
		WHERE id::text = $1`, id, k.EntityType, k.EntityID, k.PriceType, k.Date)
	return err
}

// priceOverrides are the active overrides of a chunk's keys, by key.
func (r *runner) priceOverrides(keys []seriesKey) (map[string]activeOverride, error) {
	out := map[string]activeOverride{}
	if !r.overridesTable() {
		return out, nil
	}
	ids := make([]string, len(keys))
	attrs := make([]string, len(keys))
	for i, k := range keys {
		ids[i], attrs[i] = k.EntityID, priceAttr(k.PriceType, k.Date)
	}
	var rows []struct {
		activeOverride
		Golden string `db:"golden_id"`
	}
	if err := r.tx.SelectContext(r.ctx, &rows, `SELECT o.id::text, o.golden_id::text, o.attribute, o.value, o.reason, o.requested_by_name, o.mode,
			(SELECT string_agg(COALESCE(v.approver_name, v.approver), ', ' ORDER BY v.decided_at) FROM mdm.golden_override_vote v
			  WHERE v.override_id = o.id AND v.decision = 'APPROVE') AS approvers
		FROM mdm.golden_override o
		JOIN unnest($2::text[], $3::text[]) AS k(eid, attr) ON o.golden_id::text = k.eid AND o.attribute = k.attr
		WHERE o.entity_cd = $1 AND o.active`, r.p.EntityCd, pq.Array(ids), pq.Array(attrs)); err != nil {
		return nil, err
	}
	for _, x := range rows {
		pt, d, _ := parsePriceAttr(x.Attribute)
		out[x.Golden+"|"+pt+"|"+d] = x.activeOverride
	}
	return out, nil
}

// overridesTable: whether the data plane has the override tables (crims 0013).
func (r *runner) overridesTable() bool {
	if r.overridesChecked {
		return r.overrides
	}
	r.overridesChecked = true
	_ = r.tx.GetContext(r.ctx, &r.overrides, `SELECT to_regclass('mdm.golden_override') IS NOT NULL`)
	return r.overrides
}

func overrideReason(o activeOverride) string {
	by := "a steward"
	if o.By != nil {
		by = *o.By
	}
	how := "applied directly"
	if o.Mode == ModeApproval && o.Approvers != nil {
		how = "approved by " + *o.Approvers
	}
	return fmt.Sprintf("steward override by %s (%s): %s", by, how, o.Reason)
}
