package mastering

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// Profiles lists the mastered entities the tenant sees.
func (e *Engine) Profiles(ctx context.Context, tenantID string) ([]Profile, error) {
	var out []Profile
	err := e.inConfig(ctx, tenantID, func(tx *sqlx.Tx) error {
		return tx.SelectContext(ctx, &out, `SELECT DISTINCT ON (entity_cd) `+profileCols+`
			FROM mdm.mastering_entity ORDER BY entity_cd, (tenant_id::text = $1) DESC`, tenantID)
	})
	return out, err
}

func (e *Engine) profile(ctx context.Context, tenantID, entity string) (*Profile, error) {
	var p *Profile
	err := e.inConfig(ctx, tenantID, func(tx *sqlx.Tx) error {
		var err error
		p, err = getProfile(ctx, tx, tenantID, entity)
		return err
	})
	return p, err
}

// Runs lists recent runs of an entity.
func (e *Engine) Runs(ctx context.Context, tenantID, entity string, limit int) ([]Run, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	out := []Run{}
	err := e.inTenant(ctx, tenantID, func(tx *sqlx.Tx) error {
		return tx.SelectContext(ctx, &out, `SELECT `+runCols+` FROM mdm.mastering_run
			WHERE entity_cd = $1 ORDER BY started_at DESC LIMIT $2`, strings.ToUpper(entity), limit)
	})
	return out, err
}

// GetRun returns one run.
func (e *Engine) GetRun(ctx context.Context, tenantID, id string) (*Run, error) {
	var r Run
	err := e.inTenant(ctx, tenantID, func(tx *sqlx.Tx) error {
		return tx.GetContext(ctx, &r, `SELECT `+runCols+` FROM mdm.mastering_run WHERE id::text = $1`, id)
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, msgNoRun(id)
	}
	return &r, err
}

// GoldenSummary is one golden record in a list: its latest version.
type GoldenSummary struct {
	ID                 string          `db:"id" json:"id"`
	Code               string          `db:"code" json:"code"`
	Name               *string         `db:"name" json:"name,omitempty"`
	Version            int             `db:"golden_version" json:"version"`
	Status             string          `db:"status" json:"status"`
	IsCurrent          bool            `db:"is_current" json:"is_current"`
	DQScore            *float64        `db:"overall_dq_score" json:"dq_score,omitempty"`
	IdentityConfidence *float64        `db:"identity_confidence" json:"identity_confidence,omitempty"`
	Sources            int             `db:"sources" json:"sources"`
	WinningSources     json.RawMessage `db:"winning_sources" json:"winning_sources"`
	UpdatedAt          time.Time       `db:"knowledge_timestamp" json:"updated_at"`
}

// GoldenFilter narrows the golden record list.
type GoldenFilter struct {
	Q      string // code or name contains
	Status string // PUBLISHED | REVIEW | ...
	Limit  int
}

// Golden lists golden records, latest version each.
func (e *Engine) Golden(ctx context.Context, tenantID, entity string, f GoldenFilter) ([]GoldenSummary, error) {
	p, err := e.profile(ctx, tenantID, entity)
	if err != nil {
		return nil, err
	}
	if f.Limit <= 0 || f.Limit > 500 {
		f.Limit = 100
	}
	out := []GoldenSummary{}
	name := qi(p.Settings.NameAttribute)
	err = e.inTenant(ctx, tenantID, func(tx *sqlx.Tx) error {
		return tx.SelectContext(ctx, &out, fmt.Sprintf(`SELECT a.%[6]s::text AS id, a.%[1]s AS code, a.%[2]s::text AS name,
				g.golden_version, g.status, g.is_current, g.overall_dq_score, g.identity_confidence, g.winning_sources, g.knowledge_timestamp,
				(SELECT count(*) FROM mdm.entity_xref x WHERE x.entity_cd = $1 AND x.golden_id = a.%[6]s AND x.status = 'ACTIVE') AS sources
			FROM %[3]s a
			JOIN LATERAL (SELECT * FROM %[4]s g WHERE g.%[5]s = a.%[6]s ORDER BY g.golden_version DESC LIMIT 1) g ON true
			WHERE %[7]s AND ($2 = '' OR a.%[1]s ILIKE '%%' || $2 || '%%' OR a.%[2]s::text ILIKE '%%' || $2 || '%%')
			  AND ($3 = '' OR g.status = $3)
			ORDER BY g.knowledge_timestamp DESC LIMIT $4`,
			qi(p.AnchorCodeColumn), name, qi(p.AnchorTable), p.table("golden_record"), p.keyColumn(), qi(p.entityCol()), p.current("a")),
			p.EntityCd, strings.TrimSpace(f.Q), strings.ToUpper(f.Status), f.Limit)
	})
	return out, err
}

// GoldenDetail is a golden record with its provenance: every version, one
// version's fields and where each came from (the latest unless another is
// asked for), its sources and identifiers.
type GoldenDetail struct {
	ID       string          `json:"id"`
	Code     string          `json:"code"`
	Versions []GoldenVersion `json:"versions"`
	// Selected is the version the fields and decisions are for.
	Selected    int               `json:"selected_version"`
	Fields      []GoldenField     `json:"fields"`
	Sources     []Xref            `json:"sources"`
	Identifiers []Identifier      `json:"identifiers"`
	Exceptions  []ExceptionRow    `json:"exceptions"`
	Decisions   []SurvivalLogRow  `json:"decisions"`
	Anchor      map[string]string `json:"anchor,omitempty"`
	// Terms names each attribute by the business object field (semantic
	// term) mapped to it, for the side-by-side view.
	Terms map[string]string `json:"terms,omitempty"`
	// Rules names the survivorship rules the decisions cite (id -> label).
	Rules map[string]string `json:"rules,omitempty"`
}

type GoldenVersion struct {
	ID                 string          `db:"id" json:"id"`
	Version            int             `db:"golden_version" json:"version"`
	Status             string          `db:"status" json:"status"`
	IsCurrent          bool            `db:"is_current" json:"is_current"`
	Attributes         json.RawMessage `db:"golden_attributes" json:"attributes"`
	WinningSources     json.RawMessage `db:"winning_sources" json:"winning_sources"`
	DQScore            *float64        `db:"overall_dq_score" json:"dq_score,omitempty"`
	IdentityConfidence *float64        `db:"identity_confidence" json:"identity_confidence,omitempty"`
	KnowledgeAt        time.Time       `db:"knowledge_timestamp" json:"knowledge_at"`
	PublishedAt        *time.Time      `db:"published_at" json:"published_at,omitempty"`
}

type GoldenField struct {
	Name       string   `db:"field_name" json:"name"`
	Value      *string  `db:"field_value" json:"value"`
	Source     *string  `db:"source" json:"source,omitempty"`
	SourceKey  *string  `db:"source_field" json:"source_key,omitempty"`
	Confidence *float64 `db:"confidence" json:"confidence,omitempty"`
}

type Xref struct {
	Source     string    `db:"source" json:"source"`
	SourceKey  string    `db:"source_key" json:"source_key"`
	Method     string    `db:"match_method" json:"method"`
	Score      *float64  `db:"match_score" json:"score,omitempty"`
	RuleCd     *string   `db:"match_rule_cd" json:"rule,omitempty"`
	MatchedKey string    `db:"matched_keys" json:"matched_keys"`
	Status     string    `db:"status" json:"status"`
	UpdatedAt  time.Time `db:"updated_at" json:"updated_at"`
}

type Identifier struct {
	Type      string  `db:"id_type" json:"type"`
	Value     string  `db:"id_value" json:"value"`
	IsPrimary bool    `db:"is_primary" json:"is_primary"`
	Source    *string `db:"source" json:"source,omitempty"`
}

type ExceptionRow struct {
	ID          string    `db:"id" json:"id"`
	GoldenID    *string   `db:"golden_id" json:"golden_id,omitempty"`
	SourceKey   *string   `db:"identifier_value" json:"source_key,omitempty"`
	Type        string    `db:"exception_type" json:"type"`
	Severity    string    `db:"severity" json:"severity"`
	Description string    `db:"exception_description" json:"description"`
	Source      *string   `db:"source" json:"source,omitempty"`
	Status      string    `db:"status" json:"status"`
	DetectedAt  time.Time `db:"detected_at" json:"detected_at"`
}

type SurvivalLogRow struct {
	Version   int             `db:"golden_version" json:"version"`
	Field     string          `db:"field_name" json:"field"`
	Value     *string         `db:"winning_value" json:"value"`
	Source    *string         `db:"source" json:"source,omitempty"`
	Competing json.RawMessage `db:"competing_values" json:"competing"`
	Reason    *string         `db:"decision_reason" json:"reason,omitempty"`
	RuleID    *string         `db:"rule_id" json:"rule_id,omitempty"`
}

// GoldenByID returns a golden record's provenance.
// version 0 is the latest.
func (e *Engine) GoldenByID(ctx context.Context, tenantID, entity, id string, version int) (*GoldenDetail, error) {
	p, err := e.profile(ctx, tenantID, entity)
	if err != nil {
		return nil, err
	}
	d := &GoldenDetail{ID: id}
	err = e.inTenant(ctx, tenantID, func(tx *sqlx.Tx) error {
		if err := tx.GetContext(ctx, &d.Code, fmt.Sprintf(`SELECT %s FROM %s WHERE %s::text = $1 AND %s`, qi(p.AnchorCodeColumn), qi(p.AnchorTable), qi(p.entityCol()), p.current("")), id); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return msgNoGolden(id)
			}
			return err
		}
		key := p.keyColumn()
		if err := tx.SelectContext(ctx, &d.Versions, fmt.Sprintf(`SELECT id::text, golden_version, status, is_current, golden_attributes,
				winning_sources, overall_dq_score, identity_confidence, knowledge_timestamp, published_at
			FROM %s WHERE %s::text = $1 ORDER BY golden_version DESC`, p.table("golden_record"), key), id); err != nil {
			return err
		}
		if len(d.Versions) > 0 {
			sel := d.Versions[0]
			if version > 0 {
				found := false
				for _, v := range d.Versions {
					if v.Version == version {
						sel, found = v, true
					}
				}
				if !found {
					return msgNoVersion(id, version)
				}
			}
			d.Selected = sel.Version
			if err := tx.SelectContext(ctx, &d.Fields, fmt.Sprintf(`SELECT f.field_name, f.field_value, s.code AS source, f.source_field, f.confidence
				FROM %s f LEFT JOIN mdm.source_systems s ON s.id = f.source_system_id
				WHERE f.golden_record_id::text = $1 ORDER BY f.field_name`, p.table("golden_field")), sel.ID); err != nil {
				return err
			}
			if err := tx.SelectContext(ctx, &d.Decisions, fmt.Sprintf(`SELECT l.golden_version, l.field_name, l.winning_value, s.code AS source,
					l.competing_values, l.decision_reason, l.rule_id::text
				FROM %s l LEFT JOIN mdm.source_systems s ON s.id = l.winning_source_id
				WHERE l.%s::text = $1 AND l.golden_version = $2 ORDER BY l.field_name`, p.table("survivorship_log"), key), id, sel.Version); err != nil {
				return err
			}
		}
		if err := tx.SelectContext(ctx, &d.Sources, `SELECT COALESCE(s.code, x.source_system_id::text) AS source, x.source_key, x.match_method,
				x.match_score, x.match_rule_cd, x.matched_keys::text AS matched_keys, x.status, x.updated_at
			FROM mdm.entity_xref x LEFT JOIN mdm.source_systems s ON s.id = x.source_system_id
			WHERE x.entity_cd = $1 AND x.golden_id::text = $2 ORDER BY x.status, source, x.source_key`, p.EntityCd, id); err != nil {
			return err
		}
		if p.IdentifierTable != nil {
			ids := p.ident()
			primary := "false"
			if ids.PrimaryColumn != "" {
				primary = qi(ids.PrimaryColumn)
			}
			source := qi(ids.SourceColumn) + "::text"
			if ids.SourceIsID {
				source = fmt.Sprintf("(SELECT s.code FROM mdm.source_systems s WHERE s.id = i.%s)", qi(ids.SourceColumn))
			}
			if err := tx.SelectContext(ctx, &d.Identifiers, fmt.Sprintf(`SELECT i.%s AS id_type, i.%s AS id_value, %s AS is_primary, %s AS source FROM %s i
				WHERE i.%s::text = $1 AND %s ORDER BY 3 DESC, 1`, qi(ids.TypeColumn), qi(ids.ValueColumn), primary, source, qi(*p.IdentifierTable),
				qi(ids.KeyColumn), ids.identActive("i")), id); err != nil {
				return err
			}
		}
		var err error
		d.Exceptions, err = exceptions(ctx, tx, p, "", id, 50)
		return err
	})
	if err != nil {
		return d, err
	}
	d.Terms, d.Rules = e.decisionLabels(ctx, tenantID, p, d.Decisions)
	return d, err
}

// Exceptions lists an entity's exceptions (status "" = open and in review).
func (e *Engine) Exceptions(ctx context.Context, tenantID, entity, status string, limit int) ([]ExceptionRow, error) {
	p, err := e.profile(ctx, tenantID, entity)
	if err != nil {
		return nil, err
	}
	var out []ExceptionRow
	err = e.inTenant(ctx, tenantID, func(tx *sqlx.Tx) error {
		out, err = exceptions(ctx, tx, p, status, "", limit)
		return err
	})
	return out, err
}

func exceptions(ctx context.Context, tx *sqlx.Tx, p *Profile, status, golden string, limit int) ([]ExceptionRow, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	if p.timeSeries() {
		return priceExceptions(ctx, tx, status, limit)
	}
	out := []ExceptionRow{}
	err := tx.SelectContext(ctx, &out, fmt.Sprintf(`SELECT x.id::text, x.%[1]s::text AS golden_id, x.identifier_value, x.exception_type, x.severity,
			x.exception_description, s.code AS source, x.status, x.detected_at
		FROM %[2]s x LEFT JOIN mdm.source_systems s ON s.id = x.source_system_id
		WHERE (($1 = '' AND x.status IN ('OPEN', 'IN_REVIEW')) OR x.status = $1)
		  AND ($2 = '' OR x.%[1]s::text = $2)
		ORDER BY x.detected_at DESC LIMIT $3`, p.keyColumn(), p.table("exception")), strings.ToUpper(status), golden, limit)
	return out, err
}

// MatchCandidate is a possible duplicate pair for a steward.
type MatchCandidate struct {
	ID      string          `db:"id" json:"id"`
	A       string          `db:"a" json:"a"`
	ACode   *string         `db:"a_code" json:"a_code,omitempty"`
	AName   *string         `db:"a_name" json:"a_name,omitempty"`
	B       string          `db:"b" json:"b"`
	BCode   *string         `db:"b_code" json:"b_code,omitempty"`
	BName   *string         `db:"b_name" json:"b_name,omitempty"`
	Score   float64         `db:"overall_score" json:"score"`
	Rule    *string         `db:"rule_cd" json:"rule,omitempty"`
	Matched json.RawMessage `db:"matched_keys" json:"matched_keys"`
	Status  string          `db:"status" json:"status"`
	// A merge of this pair waiting for approval, if any.
	Merge *PendingMerge `db:"-" json:"merge_request,omitempty"`
}

// PendingMerge is a merge request awaiting approval.
type PendingMerge struct {
	ID                string  `db:"id" json:"id"`
	Candidate         string  `db:"candidate_id" json:"-"`
	Keep              string  `db:"keep" json:"keep"`
	Note              *string `db:"note" json:"note,omitempty"`
	ApprovalsRequired int     `db:"approvals_required" json:"approvals_required"`
	Approvals         int     `db:"approvals" json:"approvals"`
	RequestedBy       string  `db:"requested_by" json:"-"`
	RequestedByName   *string `db:"requested_by_name" json:"requested_by_name,omitempty"`
	Mine              bool    `db:"-" json:"mine"`
	Voted             bool    `db:"voted" json:"voted"`
}

// Candidates lists possible duplicates (status "" = PENDING).
func (e *Engine) Candidates(ctx context.Context, tenantID, entity, status, actorID string, limit int) ([]MatchCandidate, error) {
	p, err := e.profile(ctx, tenantID, entity)
	if err != nil {
		return nil, err
	}
	if p.timeSeries() {
		return []MatchCandidate{}, nil // a time series has no duplicates to review
	}
	if status == "" {
		status = "PENDING"
	}
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	out := []MatchCandidate{}
	a, b := qi(p.TablePrefix+"_id_a"), qi(p.TablePrefix+"_id_b")
	code, name := qi(p.AnchorCodeColumn), qi(p.Settings.NameAttribute)
	err = e.inTenant(ctx, tenantID, func(tx *sqlx.Tx) error {
		if err := tx.SelectContext(ctx, &out, fmt.Sprintf(`SELECT c.id::text, c.%[1]s::text AS a, pa.%[3]s AS a_code, pa.%[4]s::text AS a_name,
				c.%[2]s::text AS b, pb.%[3]s AS b_code, pb.%[4]s::text AS b_name, c.overall_score, r.rule_cd, c.matched_keys, c.status
			FROM %[5]s c
			LEFT JOIN %[6]s pa ON pa.%[8]s = c.%[1]s AND %[9]s LEFT JOIN %[6]s pb ON pb.%[8]s = c.%[2]s AND %[10]s
			LEFT JOIN %[7]s r ON r.id = c.match_rule_id
			WHERE c.status = $1 ORDER BY c.overall_score DESC LIMIT $2`,
			a, b, code, name, p.table("match_candidate"), qi(p.AnchorTable), p.table("match_rule"), qi(p.entityCol()), p.current("pa"), p.current("pb")), strings.ToUpper(status), limit); err != nil {
			return err
		}
		var ok bool
		if err := tx.GetContext(ctx, &ok, `SELECT to_regclass('mdm.golden_merge_request') IS NOT NULL`); err != nil || !ok || len(out) == 0 {
			return err
		}
		var reqs []PendingMerge
		if err := tx.SelectContext(ctx, &reqs, `SELECT m.id::text, m.candidate_id::text, m.keep, m.note, m.approvals_required,
				m.requested_by, m.requested_by_name,
				(SELECT count(*) FROM mdm.golden_merge_vote v WHERE v.request_id = m.id AND v.decision = 'APPROVE') AS approvals,
				EXISTS (SELECT 1 FROM mdm.golden_merge_vote v WHERE v.request_id = m.id AND v.approver = $2) AS voted
			FROM mdm.golden_merge_request m WHERE m.entity_cd = $1 AND m.status = 'PENDING'`, p.EntityCd, actorID); err != nil {
			return err
		}
		by := map[string]*PendingMerge{}
		for i := range reqs {
			reqs[i].Mine = reqs[i].RequestedBy == actorID
			by[reqs[i].Candidate] = &reqs[i]
		}
		for i := range out {
			out[i].Merge = by[out[i].ID]
		}
		return nil
	})
	return out, err
}

// Load is a staging load of an entity's domain and its latest mastering.
type Load struct {
	ID            string     `db:"id" json:"id"`
	Source        string     `db:"source_system_cd" json:"source"`
	RunRef        *string    `db:"run_ref" json:"run_ref,omitempty"`
	FileName      *string    `db:"file_name" json:"file_name,omitempty"`
	Status        *string    `db:"status" json:"status,omitempty"`
	ReceivedRows  *int       `db:"received_rows" json:"received_rows,omitempty"`
	StartedAt     *time.Time `db:"started_at" json:"started_at,omitempty"`
	MasteringRun  *string    `db:"mastering_run_id" json:"mastering_run_id,omitempty"`
	MasteringStat *string    `db:"mastering_status" json:"mastering_status,omitempty"`
}

// Loads lists recent staging loads for the entity, newest first.
func (e *Engine) Loads(ctx context.Context, tenantID, entity string, limit int) ([]Load, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	out := []Load{}
	err := e.inTenant(ctx, tenantID, func(tx *sqlx.Tx) error {
		return tx.SelectContext(ctx, &out, `SELECT l.id::text, l.source_system_cd, l.run_ref, l.file_name, l.status, l.received_rows, l.started_at,
				m.id::text AS mastering_run_id, m.status AS mastering_status
			FROM staging._load_run l
			LEFT JOIN LATERAL (SELECT id, status FROM mdm.mastering_run m
				WHERE m.load_run_id = l.id AND m.entity_cd = $1 ORDER BY m.started_at DESC LIMIT 1) m ON true
			WHERE upper(l.domain) = $1
			ORDER BY l.started_at DESC NULLS LAST LIMIT $2`, strings.ToUpper(entity), limit)
	})
	return out, err
}

// ResolveException closes an exception: RESOLVED (fixed) or WAIVED
// (accepted as is), with a note for the audit trail.
func (e *Engine) ResolveException(ctx context.Context, tenantID, entity, id, status, note, by string) error {
	status = strings.ToUpper(status)
	if status != "RESOLVED" && status != "WAIVED" && status != "IN_REVIEW" {
		return msgBadResolution(status)
	}
	p, err := e.profile(ctx, tenantID, entity)
	if err != nil {
		return err
	}
	return e.inTenant(ctx, tenantID, func(tx *sqlx.Tx) error {
		res, err := tx.ExecContext(ctx, fmt.Sprintf(`UPDATE %s SET status = $2,
				resolved_at = CASE WHEN $2 IN ('RESOLVED', 'WAIVED') THEN now() END,
				resolution_note = NULLIF($3, ''),
				custom_attributes = custom_attributes || jsonb_build_object('resolved_by', $4::text)
			WHERE id::text = $1 AND status IN ('OPEN', 'IN_REVIEW')`, p.table("exception")), id, status, strings.TrimSpace(note), by)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return msgNoException(id)
		}
		return nil
	})
}

// decisionLabels names what the side-by-side view shows: each attribute's
// semantic term (the BO field mapped to it) and each cited survivorship
// rule (attribute and strategy). Best effort: a missing catalog leaves the
// attribute names as they are.
func (e *Engine) decisionLabels(ctx context.Context, tenantID string, p *Profile, ds []SurvivalLogRow) (map[string]string, map[string]string) {
	terms := map[string]string{}
	if e.Fields != nil {
		if fa, err := e.Fields.AttrFields(ctx, tenantID, p.BOKey, p.AnchorTable, p.Settings.BOBinding); err == nil {
			for field, attr := range fa {
				if ref, ok := p.referenceFor(attr); ok {
					attr = ref.Attribute
				}
				if _, taken := terms[attr]; !taken || field < terms[attr] {
					terms[attr] = field
				}
			}
		}
	}
	rules := map[string]string{}
	var ids []string
	for _, d := range ds {
		if d.RuleID != nil && *d.RuleID != "" {
			ids = append(ids, *d.RuleID)
		}
	}
	if len(ids) > 0 {
		var rows []struct {
			ID       string `db:"id"`
			Attr     string `db:"attribute_name"`
			Strategy string `db:"strategy"`
		}
		_ = e.inConfig(ctx, tenantID, func(tx *sqlx.Tx) error {
			return tx.SelectContext(ctx, &rows, `SELECT id::text, attribute_name, strategy FROM mdm.survivorship_rule WHERE id::text = ANY($1)`, pq.Array(ids))
		})
		for _, r := range rows {
			rules[r.ID] = r.Strategy + " rule for " + r.Attr
		}
	}
	return terms, rules
}
