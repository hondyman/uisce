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

	"github.com/hondyman/uisce/backend/internal/stagingbind"
)

// BindingSource resolves a staging binding (stagingbind.Store implements it).
type BindingSource interface {
	StagingFields(ctx context.Context, tenantID, boKey, table string) (map[string]string, error)
}

// FieldMapper maps a BO's fields to the anchor table columns they master
// (their MAPS_TO columns).
type FieldMapper interface {
	AttrFields(ctx context.Context, tenantID, boKey, anchorTable string) (map[string]string, error)
}

// Engine runs mastering. Data is the tenant's data plane (crims).
type Engine struct {
	Data     *sqlx.DB
	Rules    RuleLister
	Bindings BindingSource
	Fields   FieldMapper
	GoldCopy func(ctx context.Context) (string, error)
	Now      func() time.Time
}

// RunRequest starts a run over one staging load.
type RunRequest struct {
	Entity       string `json:"-"`
	StagingTable string `json:"staging_table"`
	LoadRunID    string `json:"load_run_id"`
	// IdempotencyKey makes a retried trigger return the first run (default:
	// the load run, so a load is mastered once unless a new key is given).
	IdempotencyKey string `json:"idempotency_key,omitempty"`
	Trigger        string `json:"-"` // manual | schedule | external
	ScheduleRunID  string `json:"-"`
	WorkflowID     string `json:"-"`
	StartedBy      string `json:"-"`
	StartedByID    string `json:"-"`
	// DryRun previews the run: everything runs, nothing is kept.
	DryRun bool `json:"dry_run,omitempty"`
}

// Run is a mastering run (mdm.mastering_run).
type Run struct {
	ID             string          `db:"id" json:"id"`
	EntityCd       string          `db:"entity_cd" json:"entity_cd"`
	LoadRunID      *string         `db:"load_run_id" json:"load_run_id,omitempty"`
	SourceSystemID *string         `db:"source_system_id" json:"source_system_id,omitempty"`
	IdempotencyKey string          `db:"idempotency_key" json:"idempotency_key"`
	TriggerKind    string          `db:"trigger_kind" json:"trigger"`
	ScheduleRunID  *string         `db:"schedule_run_id" json:"schedule_run_id,omitempty"`
	WorkflowID     *string         `db:"workflow_id" json:"workflow_id,omitempty"`
	Status         string          `db:"status" json:"status"`
	Stage          string          `db:"stage" json:"stage"`
	RawCounts      json.RawMessage `db:"counts" json:"counts"`
	ErrorCode      *string         `db:"error_code" json:"error_code,omitempty"`
	ErrorDetail    *string         `db:"error_detail" json:"error_detail,omitempty"`
	StartedBy      *string         `db:"started_by" json:"started_by,omitempty"`
	StartedAt      time.Time       `db:"started_at" json:"started_at"`
	FinishedAt     *time.Time      `db:"finished_at" json:"finished_at,omitempty"`
	// Replayed: an earlier run with the same idempotency key, not a new one.
	Replayed bool `db:"-" json:"replayed,omitempty"`
}

// Counts summarises a run.
type Counts struct {
	Records       int `json:"records"`
	Valid         int `json:"valid"`
	Invalid       int `json:"invalid"`
	Xref          int `json:"xref"`
	Deterministic int `json:"deterministic"`
	Fuzzy         int `json:"fuzzy"`
	Review        int `json:"review"`
	New           int `json:"new"`
	Conflicts     int `json:"conflicts"`
	Published     int `json:"published"`
	HeldForReview int `json:"held_for_review"`
	Unchanged     int `json:"unchanged"`
	Exceptions    int `json:"exceptions"`
}

const runCols = `id::text, entity_cd, load_run_id::text, source_system_id::text, idempotency_key, trigger_kind,
	schedule_run_id::text, workflow_id, status, stage, counts, error_code, error_detail, started_by, started_at, finished_at`

// Start records the run, or returns the earlier run with the same key (a
// failed one is retried under the same id).
func (e *Engine) Start(ctx context.Context, tenantID string, req RunRequest) (*Run, bool, error) {
	if req.StagingTable == "" || req.LoadRunID == "" {
		return nil, false, msgNeedTable()
	}
	if _, err := uuid.Parse(req.LoadRunID); err != nil {
		return nil, false, msgNoLoadRun(req.LoadRunID)
	}
	if req.IdempotencyKey == "" {
		req.IdempotencyKey = "load:" + req.LoadRunID
	}
	if req.Trigger == "" {
		req.Trigger = "manual"
	}
	var run Run
	fresh := false
	err := e.inTenant(ctx, tenantID, func(tx *sqlx.Tx) error {
		err := tx.GetContext(ctx, &run, `INSERT INTO mdm.mastering_run
			(tenant_id, entity_cd, load_run_id, idempotency_key, trigger_kind, schedule_run_id, workflow_id, started_by)
			VALUES ($1::uuid, $2, $3::uuid, $4, $5, NULLIF($6, '')::uuid, NULLIF($7, ''), NULLIF($8, ''))
			ON CONFLICT (tenant_id, entity_cd, idempotency_key) DO NOTHING
			RETURNING `+runCols,
			tenantID, strings.ToUpper(req.Entity), req.LoadRunID, req.IdempotencyKey, req.Trigger, req.ScheduleRunID, req.WorkflowID, req.StartedBy)
		if err == nil {
			fresh = true
			return nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		// Seen before: a failed run is retried; anything else is replayed.
		err = tx.GetContext(ctx, &run, `UPDATE mdm.mastering_run
			SET status = 'RUNNING', stage = 'CANONICALIZE', error_code = NULL, error_detail = NULL,
			    started_at = now(), finished_at = NULL, counts = '{}'::jsonb
			WHERE tenant_id::text = $1 AND entity_cd = $2 AND idempotency_key = $3 AND status = 'FAILED'
			RETURNING `+runCols, tenantID, strings.ToUpper(req.Entity), req.IdempotencyKey)
		if err == nil {
			fresh = true
			return nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		run.Replayed = true
		return tx.GetContext(ctx, &run, `SELECT `+runCols+` FROM mdm.mastering_run
			WHERE tenant_id::text = $1 AND entity_cd = $2 AND idempotency_key = $3`, tenantID, strings.ToUpper(req.Entity), req.IdempotencyKey)
	})
	return &run, fresh, err
}

// Execute runs a started run to the end and records the outcome. The whole
// run is one transaction on the data plane: a failed run leaves nothing
// half-published, and a retry starts clean.
func (e *Engine) Execute(ctx context.Context, tenantID string, run *Run, req RunRequest) (*Run, error) {
	counts := &Counts{}
	var sourceID string
	stage := "CANONICALIZE"
	err := func() error {
		cfg, err := e.loadConfig(ctx, tenantID, req.Entity)
		if err != nil {
			return err
		}
		return e.inTenant(ctx, tenantID, func(tx *sqlx.Tx) error {
			r := &runner{e: e, tx: tx, ctx: ctx, tenant: tenantID, cfg: cfg, p: cfg.profile, run: run, req: req, counts: counts, stage: &stage}
			err := r.execute()
			sourceID = r.sourceID
			return err
		})
	}()
	return e.finish(ctx, tenantID, run.ID, counts, sourceID, stage, err)
}

// Preview is what a run over the load would do: the whole run, in a
// transaction that is rolled back. No run is recorded.
type Preview struct {
	Counts     Counts   `json:"counts"`
	Exceptions []Issue  `json:"exceptions"`
	Unmastered []string `json:"unmastered_fields,omitempty"`
}

var errPreview = errors.New("preview: rolled back")

func (e *Engine) Preview(ctx context.Context, tenantID string, req RunRequest) (*Preview, error) {
	if req.StagingTable == "" || req.LoadRunID == "" {
		return nil, msgNeedTable()
	}
	cfg, err := e.loadConfig(ctx, tenantID, req.Entity)
	if err != nil {
		return nil, err
	}
	out := &Preview{Exceptions: []Issue{}}
	stage := "CANONICALIZE"
	run := &Run{ID: uuid.NewString()}
	err = e.inTenant(ctx, tenantID, func(tx *sqlx.Tx) error {
		r := &runner{e: e, tx: tx, ctx: ctx, tenant: tenantID, cfg: cfg, p: cfg.profile, run: run, req: req, counts: &out.Counts, stage: &stage}
		if err := r.execute(); err != nil {
			return err
		}
		out.Exceptions = append(out.Exceptions, r.raised...)
		out.Unmastered = r.unmastered
		return errPreview
	})
	if errors.Is(err, errPreview) {
		err = nil
	}
	return out, err
}

func (e *Engine) finish(ctx context.Context, tenantID, runID string, c *Counts, sourceID, stage string, runErr error) (*Run, error) {
	status := "COMPLETED"
	if c.Invalid > 0 || c.Conflicts > 0 || c.Review > 0 || c.HeldForReview > 0 {
		status = "PARTIAL"
	}
	var code, detail string
	if runErr != nil {
		status = "FAILED"
		detail = runErr.Error()
		code = "MASTERING"
		var ce interface{ Code() string }
		if errors.As(runErr, &ce) {
			code = ce.Code()
		}
	} else {
		stage = "DONE"
	}
	raw, _ := json.Marshal(c)
	var run Run
	err := e.inTenant(ctx, tenantID, func(tx *sqlx.Tx) error {
		return tx.GetContext(ctx, &run, `UPDATE mdm.mastering_run SET status = $2, stage = $3, counts = $4,
				error_code = NULLIF($5, ''), error_detail = NULLIF($6, ''), source_system_id = COALESCE(NULLIF($7, '')::uuid, source_system_id),
				finished_at = now()
			WHERE id::text = $1 RETURNING `+runCols, runID, status, stage, raw, code, detail, sourceID)
	})
	if err != nil {
		return nil, err
	}
	return &run, runErr
}

// runner is one run's working state.
type runner struct {
	e        *Engine
	tx       *sqlx.Tx
	ctx      context.Context
	tenant   string
	cfg      *config
	p        *Profile
	run      *Run
	req      RunRequest
	counts   *Counts
	stage    *string
	sourceID string
	sourceCd string

	attrField   map[string]string // golden attribute -> BO field
	anchorCols  map[string]string // anchor column -> data type
	refIDs      map[string]map[string]string
	touched     map[string]bool // golden ids to survive + publish
	rules       *RuleSet
	identifiers bool
	raised      []Issue // exceptions raised this run
	// force publishes a new version even when the values are unchanged
	// (a steward merge changes the sources behind them).
	force      bool
	unmastered []string // bound fields with no golden column
}

func (r *runner) execute() error {
	ctx, tx := r.ctx, r.tx

	// The load and its source system.
	var srcCd string
	err := tx.GetContext(ctx, &srcCd, `SELECT source_system_cd FROM staging._load_run WHERE id::text = $1 AND tenant_id::text = $2`, r.req.LoadRunID, r.tenant)
	if errors.Is(err, sql.ErrNoRows) {
		return msgNoLoadRun(r.req.LoadRunID)
	}
	if err != nil {
		return err
	}
	r.sourceCd = strings.ToUpper(srcCd)
	r.sourceID = r.cfg.sources[r.sourceCd]
	if r.sourceID == "" {
		return msgUnknownSource(srcCd)
	}

	// The binding, the BO's field -> golden column map, the rules.
	binding, err := r.e.Bindings.StagingFields(ctx, r.tenant, r.p.BOKey, r.req.StagingTable)
	if err != nil {
		return err
	}
	if binding == nil {
		return msgNoBinding(r.req.StagingTable, r.p.BOKey)
	}
	if _, ok := binding[stagingbind.SourceKey]; !ok {
		return msgNoSourceKey(r.req.StagingTable)
	}
	fieldAttr, err := r.prepare()
	if err != nil {
		return err
	}
	stagingCols, err := columns(ctx, tx, r.req.StagingTable)
	if err != nil {
		return err
	}
	if len(stagingCols) == 0 {
		return msgNoBinding(r.req.StagingTable, r.p.BOKey)
	}

	canon := &Canonicalizer{Profile: r.p, Binding: binding, FieldAttr: fieldAttr, ColumnTypes: stagingCols}
	r.unmastered = canon.Unmastered()
	records, err := r.canonicalize(canon)
	if err != nil {
		return fmt.Errorf("canonicalize: %w", err)
	}
	*r.stage = "MATCH"
	for i := range records {
		if !records[i].Valid() {
			continue
		}
		if err := r.matchOne(&records[i]); err != nil {
			return fmt.Errorf("match %s: %w", records[i].SourceKey, err)
		}
	}
	*r.stage = "PUBLISH"
	ids := make([]string, 0, len(r.touched))
	for id := range r.touched {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if err := r.masterOne(id); err != nil {
			return fmt.Errorf("publish %s: %w", id, err)
		}
	}
	return nil
}

// prepare loads what surviving and publishing need, whatever started the
// work (a load, or a steward's merge): the BO field <-> golden attribute
// map, the rules, the anchor's columns and the reference ids.
func (r *runner) prepare() (map[string]string, error) {
	fieldAttr, err := r.e.Fields.AttrFields(r.ctx, r.tenant, r.p.BOKey, r.p.AnchorTable)
	if err != nil {
		return nil, err
	}
	r.attrField = map[string]string{}
	for f, a := range fieldAttr {
		r.attrField[a] = f
		if ref, ok := r.p.referenceFor(a); ok {
			r.attrField[ref.Attribute] = f
		}
	}
	if r.rules, err = loadRules(r.ctx, r.e.Rules, r.tenant, r.p.BOKey); err != nil {
		return nil, err
	}
	if r.anchorCols, err = columns(r.ctx, r.tx, r.p.AnchorTable); err != nil {
		return nil, err
	}
	if err := r.loadRefIDs(); err != nil {
		return nil, err
	}
	r.identifiers = r.p.IdentifierTable != nil
	r.touched = map[string]bool{}
	return fieldAttr, nil
}

// columns lists a table's columns and data types (empty: no such table).
func columns(ctx context.Context, tx *sqlx.Tx, table string) (map[string]string, error) {
	parts := strings.SplitN(table, ".", 2)
	if len(parts) != 2 {
		return nil, nil
	}
	var rows []struct {
		Name string `db:"column_name"`
		Type string `db:"data_type"`
	}
	if err := tx.SelectContext(ctx, &rows, `SELECT column_name, data_type FROM information_schema.columns
		WHERE table_schema = $1 AND table_name = $2`, parts[0], parts[1]); err != nil {
		return nil, err
	}
	out := make(map[string]string, len(rows))
	for _, c := range rows {
		out[c.Name] = c.Type
	}
	return out, nil
}

// loadRefIDs reads each reference table's internal codes and ids (reference
// data: the tenant's own and the shared reference tenant's).
func (r *runner) loadRefIDs() error {
	r.refIDs = map[string]map[string]string{}
	for _, ref := range r.p.Settings.References {
		var rows []struct {
			Code string `db:"code"`
			ID   string `db:"id"`
		}
		q := fmt.Sprintf(`SELECT %s AS code, id::text FROM %s ORDER BY (tenant_id::text = $1)`, qi(ref.RefCodeColumn), qi(ref.RefTable))
		if err := r.tx.SelectContext(r.ctx, &rows, q, r.tenant); err != nil {
			return fmt.Errorf("reference %s: %w", ref.RefTable, err)
		}
		m := map[string]string{}
		for _, row := range rows {
			m[normCode(row.Code)] = row.ID
		}
		r.refIDs[ref.Attribute] = m
	}
	return nil
}

// canonicalize reads the load's staging rows, canonicalizes each through
// the binding, resolves codes, runs the rules the source can be held to,
// and writes the canonical records (and an exception per rejected one).
func (r *runner) canonicalize(canon *Canonicalizer) ([]Record, error) {
	ctx, tx := r.ctx, r.tx
	q := fmt.Sprintf(`SELECT * FROM %s WHERE _load_run_id::text = $1 AND tenant_id::text = $2 ORDER BY _source_row_num`, qi(r.req.StagingTable))
	rows, err := tx.QueryxContext(ctx, q, r.req.LoadRunID, r.tenant)
	if err != nil {
		return nil, err
	}
	var raw []map[string]any
	for rows.Next() {
		m := map[string]any{}
		if err := rows.MapScan(m); err != nil {
			rows.Close()
			return nil, err
		}
		raw = append(raw, m)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	incomingCols, err := columns(ctx, tx, r.p.IncomingTable)
	if err != nil {
		return nil, err
	}
	// Re-mastering a load replaces its canonical records.
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`DELETE FROM %s WHERE load_run_id::text = $1 AND source_system_id::text = $2`, qi(r.p.IncomingTable)),
		r.req.LoadRunID, r.sourceID); err != nil {
		return nil, err
	}

	out := make([]Record, 0, len(raw))
	for _, row := range raw {
		ingested, _ := asTime(row["_ingested_at"])
		rec := canon.Record(row, ingested)
		r.resolveCodes(&rec)
		r.checkIDs(&rec)
		rec.Issues = append(rec.Issues, r.rules.Canonical(inFields(rec.Attrs, r.attrField))...)
		r.counts.Records++
		if rec.Valid() {
			r.counts.Valid++
		} else {
			r.counts.Invalid++
		}
		if err := r.writeIncoming(rec, incomingCols); err != nil {
			return nil, err
		}
		for _, is := range rec.Issues {
			if is.Severity == SevError {
				if err := r.exception("", rec.SourceKey, is); err != nil {
					return nil, err
				}
			}
		}
		out = append(out, rec)
	}
	return out, nil
}

// resolveCodes maps each reference's vendor code to the internal code: the
// source's code map first, else the code itself if it is an internal code.
func (r *runner) resolveCodes(rec *Record) {
	for _, ref := range r.p.Settings.References {
		v, ok := rec.Attrs[ref.Attribute]
		if !ok {
			continue
		}
		vendor := normCode(text(v))
		code := ""
		if m := r.cfg.vendorCodes[ref.Attribute][r.sourceID]; m != nil {
			code = m[vendor]
		}
		if code == "" {
			if _, ok := r.refIDs[ref.Attribute][vendor]; ok {
				code = vendor
			}
		}
		if code == "" {
			delete(rec.Attrs, ref.Attribute)
			sev := SevWarning
			if ref.Required {
				sev = SevError
			}
			rec.add("UNMAPPED_CODE", sev, ref.Attribute,
				fmt.Sprintf("%s has no %s mapping for %q (add it to %s)", r.sourceCd, ref.Attribute, text(v), mapTableName(ref)))
			continue
		}
		rec.Attrs[ref.Attribute] = code
	}
}

// checkIDs refuses a source value for an id column that isn't an id: a
// vendor code bound to a foreign key needs a reference in the profile.
func (r *runner) checkIDs(rec *Record) {
	for a, v := range rec.Attrs {
		if r.anchorCols[a] != "uuid" {
			continue
		}
		if _, err := uuid.Parse(text(v)); err != nil {
			delete(rec.Attrs, a)
			rec.add("NOT_AN_ID", SevError, a, fmt.Sprintf("%s holds ids, not %q: add a reference for it to the %s mastering profile", a, text(v), r.p.EntityCd))
		}
	}
}

func mapTableName(ref Reference) string {
	if ref.MapTable != "" {
		return ref.MapTable
	}
	return ref.RefTable
}

func (r *runner) writeIncoming(rec Record, incomingCols map[string]string) error {
	var warnings, errs []Issue
	for _, is := range rec.Issues {
		if is.Severity == SevError {
			errs = append(errs, is)
		} else {
			warnings = append(warnings, is)
		}
	}
	payload, _ := json.Marshal(map[string]any{"attrs": rec.Attrs, "identifiers": rec.Identifiers, "as_of": rec.AsOf,
		"source_key": rec.SourceKey, "warnings": warnings, "run_id": r.run.ID})
	if errs == nil {
		errs = []Issue{}
	}
	errJSON, _ := json.Marshal(errs)

	cols := []string{"tenant_id", "source_system_id", "source_system_cd", "source_row_id", "load_run_id", "canonical_payload", "is_valid", "validation_errors", "mapped_at"}
	args := []any{r.tenant, r.sourceID, r.sourceCd, rec.SourceKey, r.req.LoadRunID, payload, rec.Valid(), errJSON, time.Now().UTC()}
	std := map[string]bool{}
	for _, c := range cols {
		std[c] = true
	}
	// Typed columns the incoming table has for the record's attributes.
	attrs := make([]string, 0, len(rec.Attrs))
	for a := range rec.Attrs {
		if _, ok := incomingCols[a]; ok && !std[a] && a != "id" {
			attrs = append(attrs, a)
		}
	}
	sort.Strings(attrs)
	for _, a := range attrs {
		cols = append(cols, a)
		args = append(args, rec.Attrs[a])
	}
	ph := make([]string, len(cols))
	quoted := make([]string, len(cols))
	for i, c := range cols {
		ph[i] = fmt.Sprintf("$%d", i+1)
		quoted[i] = qi(c)
	}
	_, err := r.tx.ExecContext(r.ctx, fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s)`, qi(r.p.IncomingTable), strings.Join(quoted, ", "), strings.Join(ph, ", ")), args...)
	return err
}

// exception raises an exception for a steward, once: an open exception
// with the same type, record and message isn't raised again.
func (r *runner) exception(goldenID, sourceKey string, is Issue) error {
	sev := is.Severity
	desc := is.Message
	typ := is.Code
	if len(typ) > 50 {
		typ = typ[:50]
	}
	if len(sourceKey) > 100 {
		sourceKey = sourceKey[:100]
	}
	custom, _ := json.Marshal(map[string]any{"run_id": r.run.ID, "rule_id": is.RuleID, "attribute": is.Attribute, "source": r.sourceCd})
	key := r.p.keyColumn()
	res, err := r.tx.ExecContext(r.ctx, fmt.Sprintf(`INSERT INTO %[1]s (tenant_id, %[2]s, identifier_value, exception_type, severity, exception_description, source_system_id, custom_attributes)
		SELECT $1::uuid, NULLIF($2::text, '')::uuid, NULLIF($3::text, ''), $4::text, $5::text, $6::text, NULLIF($7::text, '')::uuid, $8::jsonb
		WHERE NOT EXISTS (SELECT 1 FROM %[1]s x WHERE x.status IN ('OPEN', 'IN_REVIEW') AND x.exception_type = $4::text
			AND x.exception_description = $6::text AND COALESCE(x.%[2]s::text, '') = $2::text AND COALESCE(x.identifier_value, '') = $3::text)`,
		r.p.table("exception"), key),
		r.tenant, goldenID, sourceKey, typ, sev, desc, r.sourceID, custom)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n > 0 {
		r.counts.Exceptions++
	}
	is.Message = strings.TrimSpace(sourceKey + " " + is.Message)
	r.raised = append(r.raised, is)
	return nil
}

func decodeCounts(r *Run, c *Counts) error {
	if len(r.RawCounts) == 0 {
		return nil
	}
	return json.Unmarshal(r.RawCounts, c)
}
