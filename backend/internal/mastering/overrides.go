package mastering

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Policy is how an entity's golden values may be overridden: with the
// approval of N people other than the proposer (more for high-risk
// attributes), or directly. A tenant's own policy wins over the gold
// copy's; with neither, approval by one.
type Policy struct {
	EntityCd           string    `db:"entity_cd" json:"entity_cd"`
	Mode               string    `db:"override_mode" json:"mode"` // APPROVAL | DIRECT
	ApprovalsRequired  int       `db:"approvals_required" json:"approvals_required"`
	HighRiskAttributes []string  `db:"-" json:"high_risk_attributes"`
	HighRiskApprovals  int       `db:"high_risk_approvals" json:"high_risk_approvals"`
	UpdatedBy          *string   `db:"updated_by" json:"updated_by,omitempty"`
	UpdatedAt          time.Time `db:"updated_at" json:"updated_at"`
	Inherited          bool      `db:"inherited" json:"inherited"`
	RawHighRisk        string    `db:"high_risk" json:"-"`
	// Attributes the entity masters (for choosing high-risk ones).
	Attributes []string `db:"-" json:"attributes,omitempty"`
}

const (
	ModeApproval = "APPROVAL"
	ModeDirect   = "DIRECT"
)

// required is how many approvals an override of attr needs (0: direct).
func (p *Policy) required(attr string) int {
	if p.Mode == ModeDirect {
		return 0
	}
	for _, a := range p.HighRiskAttributes {
		// A price override's attribute is TYPE@date: a high-risk price type
		// covers every date.
		if a == attr || strings.HasPrefix(attr, a+"@") {
			return max(p.HighRiskApprovals, p.ApprovalsRequired)
		}
	}
	return p.ApprovalsRequired
}

func defaultPolicy(entity string) *Policy {
	return &Policy{EntityCd: strings.ToUpper(entity), Mode: ModeApproval, ApprovalsRequired: 1, HighRiskApprovals: 2, HighRiskAttributes: []string{}, Inherited: true}
}

func getPolicy(ctx context.Context, tx *sqlx.Tx, tenantID, entity string) (*Policy, error) {
	var p Policy
	err := tx.GetContext(ctx, &p, `SELECT entity_cd, override_mode, approvals_required, array_to_string(high_risk_attributes, ',') AS high_risk,
			high_risk_approvals, updated_by, updated_at, tenant_id::text <> $1 AS inherited
		FROM mdm.mastering_policy WHERE entity_cd = $2 ORDER BY (tenant_id::text = $1) DESC LIMIT 1`, tenantID, strings.ToUpper(entity))
	if errors.Is(err, sql.ErrNoRows) {
		return defaultPolicy(entity), nil
	}
	if err != nil {
		return nil, err
	}
	p.HighRiskAttributes = []string{}
	for _, a := range strings.Split(p.RawHighRisk, ",") {
		if a = strings.TrimSpace(a); a != "" {
			p.HighRiskAttributes = append(p.HighRiskAttributes, a)
		}
	}
	return &p, nil
}

// GetPolicy returns the entity's override policy as the tenant sees it,
// with the attributes the entity masters.
func (e *Engine) GetPolicy(ctx context.Context, tenantID, entity string) (*Policy, error) {
	cfg, err := e.loadConfig(ctx, tenantID, entity)
	if err != nil {
		return nil, err
	}
	var p *Policy
	var anchor map[string]string
	err = e.inConfig(ctx, tenantID, func(tx *sqlx.Tx) error {
		var err error
		if p, err = getPolicy(ctx, tx, tenantID, entity); err != nil {
			return err
		}
		anchor, err = columns(ctx, tx, cfg.profile.AnchorTable)
		return err
	})
	if err != nil {
		return nil, err
	}
	if cfg.profile.timeSeries() {
		// A price is overridden per price type (and date).
		err = e.inTenant(ctx, tenantID, func(tx *sqlx.Tx) error {
			return tx.SelectContext(ctx, &p.Attributes, `SELECT price_type_cd FROM mdm.price_type WHERE is_active ORDER BY display_order, price_type_cd`)
		})
		return p, err
	}
	seen := map[string]bool{}
	for a := range cfg.survival {
		seen[a] = true
	}
	if e.Fields != nil {
		if fa, err := e.Fields.AttrFields(ctx, tenantID, cfg.profile.BOKey, cfg.profile.AnchorTable, cfg.profile.Settings.BOBinding); err == nil {
			for _, a := range fa {
				if ref, ok := cfg.profile.referenceFor(a); ok {
					a = ref.Attribute
				}
				seen[a] = true
			}
		}
	}
	for a := range seen {
		if overridable(cfg.profile, anchor, a) {
			p.Attributes = append(p.Attributes, a)
		}
	}
	sort.Strings(p.Attributes)
	return p, nil
}

// SetPolicy writes the tenant's own policy for the entity (administrators
// only), audited with before and after.
func (e *Engine) SetPolicy(ctx context.Context, tenantID, entity string, in Policy, by string) (*Policy, error) {
	in.Mode = strings.ToUpper(in.Mode)
	if in.Mode != ModeApproval && in.Mode != ModeDirect {
		return nil, msgBadPolicy()
	}
	if in.ApprovalsRequired == 0 {
		in.ApprovalsRequired = 1
	}
	if in.HighRiskApprovals == 0 {
		in.HighRiskApprovals = max(2, in.ApprovalsRequired)
	}
	if in.ApprovalsRequired < 1 || in.ApprovalsRequired > 5 || in.HighRiskApprovals < 1 || in.HighRiskApprovals > 5 {
		return nil, msgBadPolicy()
	}
	attrs := []string{}
	for _, a := range in.HighRiskAttributes {
		if a = strings.TrimSpace(a); a != "" {
			if !ident.MatchString(a) {
				return nil, msgUnknownAttribute(a)
			}
			attrs = append(attrs, a)
		}
	}
	sort.Strings(attrs)
	if _, err := e.profile(ctx, tenantID, entity); err != nil {
		return nil, err
	}
	var before *Policy
	err := e.inConfig(ctx, tenantID, func(tx *sqlx.Tx) error {
		var err error
		before, err = getPolicy(ctx, tx, tenantID, entity)
		return err
	})
	if err != nil {
		return nil, err
	}
	err = e.inTenant(ctx, tenantID, func(tx *sqlx.Tx) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO mdm.mastering_policy
				(tenant_id, entity_cd, override_mode, approvals_required, high_risk_attributes, high_risk_approvals, updated_by, updated_at)
			VALUES ($1::uuid, $2, $3, $4, string_to_array(NULLIF($5, ''), ','), $6, $7, now())
			ON CONFLICT (tenant_id, entity_cd) DO UPDATE SET override_mode = EXCLUDED.override_mode,
				approvals_required = EXCLUDED.approvals_required, high_risk_attributes = EXCLUDED.high_risk_attributes,
				high_risk_approvals = EXCLUDED.high_risk_approvals, updated_by = EXCLUDED.updated_by, updated_at = now()`,
			tenantID, strings.ToUpper(entity), in.Mode, in.ApprovalsRequired, strings.Join(attrs, ","), in.HighRiskApprovals, by); err != nil {
			return err
		}
		b, _ := json.Marshal(before)
		in.HighRiskAttributes = attrs
		a, _ := json.Marshal(in)
		_, err := tx.ExecContext(ctx, `INSERT INTO mdm.mastering_policy_audit (tenant_id, entity_cd, before, after, changed_by)
			VALUES ($1::uuid, $2, $3, $4, $5)`, tenantID, strings.ToUpper(entity), b, a, by)
		return err
	})
	if err != nil {
		return nil, err
	}
	return e.GetPolicy(ctx, tenantID, entity)
}

// Override is a request to set or clear one golden attribute.
type Override struct {
	ID                string          `db:"id" json:"id"`
	GoldenID          string          `db:"golden_id" json:"golden_id"`
	GoldenCode        *string         `db:"golden_code" json:"golden_code,omitempty"`
	GoldenName        *string         `db:"golden_name" json:"golden_name,omitempty"`
	Attribute         string          `db:"attribute" json:"attribute"`
	Action            string          `db:"action" json:"action"`
	Value             json.RawMessage `db:"value" json:"value,omitempty"`
	PreviousValue     json.RawMessage `db:"previous_value" json:"previous_value,omitempty"`
	Reason            string          `db:"reason" json:"reason"`
	Mode              string          `db:"mode" json:"mode"`
	ApprovalsRequired int             `db:"approvals_required" json:"approvals_required"`
	Approvals         int             `db:"approvals" json:"approvals"`
	Status            string          `db:"status" json:"status"`
	Active            bool            `db:"active" json:"active"`
	RequestedBy       string          `db:"requested_by" json:"-"`
	RequestedByName   *string         `db:"requested_by_name" json:"requested_by_name,omitempty"`
	RequestedAt       time.Time       `db:"requested_at" json:"requested_at"`
	AppliedAt         *time.Time      `db:"applied_at" json:"applied_at,omitempty"`
	AppliedVersion    *int            `db:"applied_version" json:"applied_version,omitempty"`
	Voters            *string         `db:"voters" json:"voters,omitempty"`
	// OpenID is what the console opens for it: the golden price's latest
	// version for a price override (a record override opens golden_id).
	OpenID *string `db:"open_id" json:"open_id,omitempty"`
	// Mine: the caller proposed it (may withdraw, may not vote). Voted:
	// the caller already voted.
	Mine  bool `db:"-" json:"mine"`
	Voted bool `db:"-" json:"voted"`
}

// OverrideRequest proposes an override.
type OverrideRequest struct {
	Attribute string          `json:"attribute"`
	Action    string          `json:"action"` // SET (default) | CLEAR
	Value     json.RawMessage `json:"value,omitempty"`
	Reason    string          `json:"reason"`
}

func (in *OverrideRequest) normalize() error {
	in.Action = strings.ToUpper(strings.TrimSpace(in.Action))
	if in.Action == "" {
		in.Action = "SET"
	}
	in.Reason = strings.TrimSpace(in.Reason)
	if in.Reason == "" {
		return msgOverrideReason()
	}
	if in.Action != "SET" && in.Action != "CLEAR" {
		return msgUnknownAttribute(in.Action)
	}
	return nil
}

const overrideCols = `o.id::text, o.golden_id::text, o.attribute, o.action, COALESCE(o.value, 'null'::jsonb) AS value, COALESCE(o.previous_value, 'null'::jsonb) AS previous_value, o.reason, o.mode,
	o.approvals_required, o.status, o.active, o.requested_by, o.requested_by_name, o.requested_at, o.applied_at, o.applied_version,
	(SELECT count(*) FROM mdm.golden_override_vote v WHERE v.override_id = o.id AND v.decision = 'APPROVE') AS approvals,
	(SELECT string_agg(COALESCE(v.approver_name, v.approver) || ':' || v.decision, ', ' ORDER BY v.decided_at)
	   FROM mdm.golden_override_vote v WHERE v.override_id = o.id) AS voters`

// ProposeOverride records an override request. Under a DIRECT policy it is
// applied at once (a new golden version); otherwise it waits for approvals.
func (e *Engine) ProposeOverride(ctx context.Context, tenantID, entity, goldenID string, in OverrideRequest, actorID, actorName string) (*Override, error) {
	if err := in.normalize(); err != nil {
		return nil, err
	}
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
	var out *Override
	err = e.inTenant(ctx, tenantID, func(tx *sqlx.Tx) error {
		var err error
		out, err = e.proposeOverride(ctx, tx, cfg, pol, tenantID, entity, goldenID, in, actorID, actorName)
		return err
	})
	return out, err
}

func (e *Engine) proposeOverride(ctx context.Context, tx *sqlx.Tx, cfg *config, pol *Policy, tenantID, entity, goldenID string, in OverrideRequest, actorID, actorName string) (*Override, error) {
	if err := in.normalize(); err != nil {
		return nil, err
	}
	if cfg.profile.timeSeries() {
		return e.proposePriceOverride(ctx, tx, cfg, pol, tenantID, entity, goldenID, in, actorID, actorName)
	}
	var out *Override
	err := func() error {
		r := e.stewardRunner(ctx, tx, cfg, tenantID, entity, actorID, actorName)
		if _, err := r.prepare(); err != nil {
			return err
		}
		current, err := r.currentAttrs(goldenID)
		if err != nil {
			return err
		}
		if !r.knownAttribute(in.Attribute, current) {
			return msgUnknownAttribute(in.Attribute)
		}
		var value any
		if in.Action == "SET" {
			if len(in.Value) == 0 || json.Unmarshal(in.Value, &value) != nil || value == nil {
				return msgOverrideValue(in.Attribute)
			}
			if err := r.checkOverrideValue(in.Attribute, value); err != nil {
				return err
			}
		} else {
			var active int
			if err := tx.GetContext(ctx, &active, `SELECT count(*) FROM mdm.golden_override WHERE entity_cd = $1 AND golden_id::text = $2
				AND attribute = $3 AND active`, cfg.profile.EntityCd, goldenID, in.Attribute); err != nil {
				return err
			}
			if active == 0 {
				return msgNothingToClear(in.Attribute)
			}
		}
		prev, _ := json.Marshal(current[in.Attribute])
		var val any
		if in.Action == "SET" {
			v, _ := json.Marshal(value)
			val = v
		}
		mode := pol.Mode
		need := pol.required(in.Attribute)
		var id string
		err = tx.GetContext(ctx, &id, `INSERT INTO mdm.golden_override (tenant_id, entity_cd, golden_id, attribute, action, value, previous_value,
				reason, mode, approvals_required, requested_by, requested_by_name)
			VALUES ($1::uuid, $2, $3::uuid, $4, $5, $6, $7, $8, $9, $10, $11, NULLIF($12, '')) RETURNING id::text`,
			tenantID, cfg.profile.EntityCd, goldenID, in.Attribute, in.Action, val, prev, in.Reason, mode, need, actorID, actorName)
		if err != nil {
			if isUniqueViolation(err) {
				return msgOverridePending(in.Attribute)
			}
			return err
		}
		if need == 0 {
			if err := r.applyOverride(id); err != nil {
				return err
			}
		}
		out, err = getOverride(ctx, tx, cfg.profile, instrumentOf(cfg), id, actorID)
		return err
	}()
	return out, err
}

// DecideOverride records one approver's vote. One rejection rejects the
// request; the Nth distinct approval applies it. The proposer never votes.
func (e *Engine) DecideOverride(ctx context.Context, tenantID, entity, id string, approve bool, comment, actorID, actorName string) (*Override, error) {
	cfg, err := e.loadConfig(ctx, tenantID, entity)
	if err != nil {
		return nil, err
	}
	var out *Override
	err = e.inTenant(ctx, tenantID, func(tx *sqlx.Tx) error {
		var err error
		out, err = e.decideOverride(ctx, tx, cfg, tenantID, entity, id, approve, comment, actorID, actorName)
		return err
	})
	return out, err
}

func (e *Engine) decideOverride(ctx context.Context, tx *sqlx.Tx, cfg *config, tenantID, entity, id string, approve bool, comment, actorID, actorName string) (*Override, error) {
	var out *Override
	err := func() error {
		var o struct {
			Status      string `db:"status"`
			RequestedBy string `db:"requested_by"`
			Need        int    `db:"approvals_required"`
		}
		err := tx.GetContext(ctx, &o, `SELECT status, requested_by, approvals_required FROM mdm.golden_override
			WHERE id::text = $1 AND entity_cd = $2 FOR UPDATE`, id, cfg.profile.EntityCd)
		if errors.Is(err, sql.ErrNoRows) {
			return msgNoOverride(id)
		}
		if err != nil {
			return err
		}
		if o.Status != "PENDING" {
			return msgOverrideDecided(o.Status)
		}
		if o.RequestedBy == actorID {
			return msgOwnOverride()
		}
		decision := "REJECT"
		if approve {
			decision = "APPROVE"
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO mdm.golden_override_vote (tenant_id, override_id, approver, approver_name, decision, comment)
			VALUES ($1::uuid, $2::uuid, $3, NULLIF($4, ''), $5, NULLIF($6, ''))`, tenantID, id, actorID, actorName, decision, strings.TrimSpace(comment)); err != nil {
			if isUniqueViolation(err) {
				return msgAlreadyVoted()
			}
			return err
		}
		if !approve {
			if _, err := tx.ExecContext(ctx, `UPDATE mdm.golden_override SET status = 'REJECTED' WHERE id::text = $1`, id); err != nil {
				return err
			}
		} else {
			var n int
			if err := tx.GetContext(ctx, &n, `SELECT count(*) FROM mdm.golden_override_vote WHERE override_id::text = $1 AND decision = 'APPROVE'`, id); err != nil {
				return err
			}
			if n >= o.Need {
				r := e.stewardRunner(ctx, tx, cfg, tenantID, entity, actorID, actorName)
				if !cfg.profile.timeSeries() {
					if _, err := r.prepare(); err != nil {
						return err
					}
				}
				if err := r.applyOverride(id); err != nil {
					return err
				}
			}
		}
		out, err = getOverride(ctx, tx, cfg.profile, instrumentOf(cfg), id, actorID)
		return err
	}()
	return out, err
}

// WithdrawOverride: the proposer takes back a pending request.
func (e *Engine) WithdrawOverride(ctx context.Context, tenantID, entity, id, actorID string) (*Override, error) {
	p, inst, err := e.seriesProfiles(ctx, tenantID, entity)
	if err != nil {
		return nil, err
	}
	var out *Override
	err = e.inTenant(ctx, tenantID, func(tx *sqlx.Tx) error {
		res, err := tx.ExecContext(ctx, `UPDATE mdm.golden_override SET status = 'WITHDRAWN'
			WHERE id::text = $1 AND entity_cd = $2 AND status = 'PENDING' AND requested_by = $3`, id, p.EntityCd, actorID)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return msgNotProposer()
		}
		out, err = getOverride(ctx, tx, p, inst, id, actorID)
		return err
	})
	return out, err
}

// Overrides lists override requests: by status ("" = all) and golden record.
func (e *Engine) Overrides(ctx context.Context, tenantID, entity, status, goldenID, actorID string, limit int) ([]Override, error) {
	p, inst, err := e.seriesProfiles(ctx, tenantID, entity)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	out := []Override{}
	err = e.inTenant(ctx, tenantID, func(tx *sqlx.Tx) error {
		err := tx.SelectContext(ctx, &out, overrideSelect(p, inst)+`
			WHERE o.entity_cd = $1 AND ($2 = '' OR o.status = $2 OR ($2 = 'ACTIVE' AND o.active)) AND ($3 = '' OR o.golden_id::text = $3)
			ORDER BY (o.status = 'PENDING') DESC, o.requested_at DESC LIMIT $4`,
			p.EntityCd, strings.ToUpper(status), goldenID, limit)
		if err != nil {
			return err
		}
		return markCaller(ctx, tx, out, actorID)
	})
	return out, err
}

// overrideSelect selects overrides with the record they are about: the
// golden record, or for a price the instrument (inst) and the price to open.
func overrideSelect(p, inst *Profile) string {
	if p.timeSeries() && inst != nil {
		return fmt.Sprintf(`SELECT %s, a.%s AS golden_code, a.%s::text AS golden_name,
				(SELECT g.id::text FROM mdm.price_golden_record g WHERE g.price_entity_id = o.golden_id
					AND g.price_type_cd = split_part(o.attribute, '@', 1) AND g.price_date::text = split_part(o.attribute, '@', 2)
					ORDER BY g.golden_version DESC LIMIT 1) AS open_id
			FROM mdm.golden_override o LEFT JOIN %s a ON a.%s = o.golden_id AND %s`,
			overrideCols, qi(inst.AnchorCodeColumn), qi(inst.Settings.NameAttribute), qi(inst.AnchorTable), qi(inst.entityCol()), inst.current("a"))
	}
	return fmt.Sprintf(`SELECT %s, a.%s AS golden_code, a.%s::text AS golden_name, NULL::text AS open_id
		FROM mdm.golden_override o LEFT JOIN %s a ON a.%s = o.golden_id AND %s`,
		overrideCols, qi(p.AnchorCodeColumn), qi(p.Settings.NameAttribute), qi(p.AnchorTable), qi(p.entityCol()), p.current("a"))
}

func getOverride(ctx context.Context, tx *sqlx.Tx, p, inst *Profile, id, actorID string) (*Override, error) {
	var o []Override
	if err := tx.SelectContext(ctx, &o, overrideSelect(p, inst)+` WHERE o.id::text = $1`, id); err != nil {
		return nil, err
	}
	if len(o) == 0 {
		return nil, msgNoOverride(id)
	}
	if err := markCaller(ctx, tx, o, actorID); err != nil {
		return nil, err
	}
	return &o[0], nil
}

func markCaller(ctx context.Context, tx *sqlx.Tx, os []Override, actorID string) error {
	if len(os) == 0 {
		return nil
	}
	ids := make([]string, len(os))
	for i := range os {
		ids[i] = os[i].ID
		os[i].Mine = os[i].RequestedBy == actorID
	}
	var voted []string
	if err := tx.SelectContext(ctx, &voted, `SELECT override_id::text FROM mdm.golden_override_vote
		WHERE approver = $1 AND override_id::text = ANY(string_to_array($2, ','))`, actorID, strings.Join(ids, ",")); err != nil {
		return err
	}
	set := map[string]bool{}
	for _, v := range voted {
		set[v] = true
	}
	for i := range os {
		os[i].Voted = set[os[i].ID]
	}
	return nil
}

// stewardRunner is a runner for work a steward starts (no load).
func (e *Engine) stewardRunner(ctx context.Context, tx *sqlx.Tx, cfg *config, tenantID, entity, actorID, actorName string) *runner {
	return &runner{e: e, tx: tx, ctx: ctx, tenant: tenantID, cfg: cfg, p: cfg.profile, run: &Run{ID: uuid.NewString()},
		req: RunRequest{Entity: entity, StartedByID: actorID, StartedBy: actorName}, counts: &Counts{}, stage: new(string)}
}

// currentAttrs is the golden record's latest attributes.
func (r *runner) currentAttrs(golden string) (map[string]any, error) {
	var code string
	if err := r.tx.GetContext(r.ctx, &code, fmt.Sprintf(`SELECT %s FROM %s WHERE %s::text = $1 AND %s`, qi(r.p.AnchorCodeColumn), qi(r.p.AnchorTable),
		qi(r.p.entityCol()), r.p.current("")), golden); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, msgNoGolden(golden)
		}
		return nil, err
	}
	var raw json.RawMessage
	err := r.tx.GetContext(r.ctx, &raw, fmt.Sprintf(`SELECT golden_attributes FROM %s WHERE %s::text = $1 ORDER BY golden_version DESC LIMIT 1`,
		r.p.table("golden_record"), r.p.keyColumn()), golden)
	if errors.Is(err, sql.ErrNoRows) {
		return map[string]any{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	return out, json.Unmarshal(raw, &out)
}

// knownAttribute: an attribute the entity masters (on the golden record,
// in a survivorship rule, or mapped from the business object).
func (r *runner) knownAttribute(attr string, current map[string]any) bool {
	if !ident.MatchString(attr) || !overridable(r.p, r.anchorCols, attr) {
		return false
	}
	if _, ok := current[attr]; ok {
		return true
	}
	if _, ok := r.cfg.survival[attr]; ok {
		return true
	}
	_, ok := r.attrField[attr]
	return ok
}

// overridable: a steward may override business attributes, never the
// record's identity (its id, tenant, minted code) or a raw id column (a
// foreign key is overridden through its reference's code instead).
func overridable(p *Profile, anchor map[string]string, attr string) bool {
	switch attr {
	case "id", "tenant_id", p.AnchorCodeColumn, p.entityCol(), "created_at", "updated_at", "valid_from", "valid_to":
		return false
	}
	return anchor[attr] != "uuid"
}

// checkOverrideValue: a reference attribute takes an internal code; a
// numeric column a number.
func (r *runner) checkOverrideValue(attr string, v any) error {
	for _, ref := range r.p.Settings.References {
		if ref.Attribute == attr {
			if _, ok := r.refIDs[attr][normCode(text(v))]; !ok {
				return msgOverrideCode(text(v), attr)
			}
			return nil
		}
	}
	if isNumericType(r.anchorCols[attr]) {
		if _, ok := number(v); !ok {
			return msgOverrideValue(attr)
		}
	}
	return nil
}

// applyOverride makes an approved (or direct) request take effect: a SET
// becomes the attribute's active override (ending any earlier one), a CLEAR
// ends the active one; then the golden record is re-survived and a new
// version published with the override in its provenance.
func (r *runner) applyOverride(id string) error {
	if r.p.timeSeries() {
		return r.applyPriceOverride(id)
	}
	ctx, tx := r.ctx, r.tx
	var o struct {
		Golden string `db:"golden_id"`
		Attr   string `db:"attribute"`
		Action string `db:"action"`
	}
	if err := tx.GetContext(ctx, &o, `SELECT golden_id::text, attribute, action FROM mdm.golden_override WHERE id::text = $1`, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE mdm.golden_override SET active = false, ended_by_id = $2::uuid, ended_at = now()
		WHERE entity_cd = $1 AND golden_id::text = $3 AND attribute = $4 AND active`, r.p.EntityCd, id, o.Golden, o.Attr); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE mdm.golden_override SET status = 'APPLIED', applied_at = now(), active = (action = 'SET')
		WHERE id::text = $1`, id); err != nil {
		return err
	}
	r.force = true
	if err := r.masterOne(o.Golden); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, fmt.Sprintf(`UPDATE mdm.golden_override SET applied_version =
		(SELECT max(golden_version) FROM %s WHERE %s::text = $2) WHERE id::text = $1`, r.p.table("golden_record"), r.p.keyColumn()), id, o.Golden)
	return err
}

// activeOverride is a live override as survivorship applies it.
type activeOverride struct {
	ID        string          `db:"id"`
	Attribute string          `db:"attribute"`
	Value     json.RawMessage `db:"value"`
	Reason    string          `db:"reason"`
	By        *string         `db:"requested_by_name"`
	Mode      string          `db:"mode"`
	Approvers *string         `db:"approvers"`
}

// applyOverrides puts the golden record's active overrides over the
// surviving values: the steward's value wins, whatever the sources say,
// until the override is cleared. The sources' values stay as competing
// values in the provenance.
func (r *runner) applyOverrides(golden string, decisions map[string]Decision) error {
	if !r.overrides {
		return nil
	}
	var os []activeOverride
	if err := r.tx.SelectContext(r.ctx, &os, `SELECT o.id::text, o.attribute, o.value, o.reason, o.requested_by_name, o.mode,
			(SELECT string_agg(COALESCE(v.approver_name, v.approver), ', ' ORDER BY v.decided_at) FROM mdm.golden_override_vote v
			  WHERE v.override_id = o.id AND v.decision = 'APPROVE') AS approvers
		FROM mdm.golden_override o WHERE o.entity_cd = $1 AND o.golden_id::text = $2 AND o.active`, r.p.EntityCd, golden); err != nil {
		return err
	}
	for _, o := range os {
		var v any
		if err := json.Unmarshal(o.Value, &v); err != nil {
			return err
		}
		prev := decisions[o.Attribute]
		by := "a steward"
		if o.By != nil {
			by = *o.By
		}
		how := "applied directly"
		if o.Mode == ModeApproval && o.Approvers != nil {
			how = "approved by " + *o.Approvers
		}
		decisions[o.Attribute] = Decision{
			Attribute: o.Attribute, Value: v,
			Winner:     Candidate{SourceCd: "STEWARD", SourceKey: "override:" + o.ID, Value: v, AsOf: time.Now().UTC()},
			Strategy:   "OVERRIDE",
			Reason:     fmt.Sprintf("steward override by %s (%s): %s", by, how, o.Reason),
			Confidence: 1,
			Competing:  prev.Competing,
		}
	}
	return nil
}
