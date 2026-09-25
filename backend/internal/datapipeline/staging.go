package datapipeline

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/lib/pq"
)

// stagingSink bulk-loads rows into a staging.* table with COPY - no business
// object involved. Every load is a staging._load_run row; (tenant, source,
// domain, run_ref) is unique, so re-running a completed run_ref loads
// nothing. Tables are row-level-secured by app.current_tenant, which every
// transaction sets.
type stagingSink struct {
	cfg    StagingSinkConfig
	db     *sql.DB
	tenant string
	runID  string
	skip   bool // run_ref already COMPLETED, or a dry run
	dryRun bool
	cols   map[string]string // row field -> staging column
	valid  map[string]bool   // real columns of the table
	shape  map[string]colShape
	loaded int
}

func newStagingSink(n Node, db *sql.DB) (Processor, error) {
	var c StagingSinkConfig
	if err := decodeConfig(n, &c); err != nil {
		return nil, err
	}
	if db == nil {
		return nil, fmt.Errorf("the staging database is not configured for this environment")
	}
	return &stagingSink{cfg: c, db: db}, nil
}

func splitTable(t string) (schema, table string) {
	parts := strings.SplitN(t, ".", 2)
	return parts[0], parts[1]
}

// inTenantTx runs fn in a transaction scoped to the tenant for RLS.
func (s *stagingSink) inTenantTx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `SELECT set_config('app.current_tenant', $1, true)`, s.tenant); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (s *stagingSink) Open(ctx context.Context, rc *RunContext) error {
	s.tenant = rc.TenantID
	runRef := s.cfg.RunRef
	if runRef == "" {
		runRef = rc.RunID
	}
	schema, table := splitTable(s.cfg.Table)
	return s.inTenantTx(ctx, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, `SELECT column_name, data_type, character_maximum_length, numeric_precision, numeric_scale
			FROM information_schema.columns WHERE table_schema = $1 AND table_name = $2`, schema, table)
		if err != nil {
			return err
		}
		s.valid, s.shape = map[string]bool{}, map[string]colShape{}
		for rows.Next() {
			var c, typ string
			var maxLen, prec, scale sql.NullInt64
			if err := rows.Scan(&c, &typ, &maxLen, &prec, &scale); err != nil {
				rows.Close()
				return err
			}
			s.valid[c] = true
			s.shape[c] = colShape{typ: typ, maxLen: int(maxLen.Int64), precision: int(prec.Int64), scale: int(scale.Int64)}
		}
		rows.Close()
		if len(s.valid) == 0 {
			return fmt.Errorf("staging table %s does not exist", s.cfg.Table)
		}
		for _, c := range []string{"_load_run_id", "_source_row_num", "tenant_id"} {
			if !s.valid[c] {
				return fmt.Errorf("staging table %s has no %s column (see the staging table template)", s.cfg.Table, c)
			}
		}
		for field, col := range s.cfg.Columns {
			if !s.valid[col] || strings.HasPrefix(col, "_") || col == "tenant_id" {
				return fmt.Errorf("%s: %q is not a loadable column of %s", field, col, s.cfg.Table)
			}
		}
		if len(s.cfg.Columns) > 0 { // empty mapping = same-name columns
			s.cols = s.cfg.Columns
		}
		if rc.DryRun {
			s.dryRun = true
			// Preview: table and mapping are checked; nothing is claimed or written.
			s.skip = true
			return nil
		}

		// Claim the run.
		err = tx.QueryRowContext(ctx, `
			INSERT INTO staging._load_run (source_system_cd, domain, run_ref, tenant_id, status)
			VALUES ($1, $2, $3, $4, 'RUNNING')
			ON CONFLICT (tenant_id, source_system_cd, domain, run_ref) DO NOTHING
			RETURNING id`, s.cfg.SourceCd, s.cfg.Domain, runRef, s.tenant).Scan(&s.runID)
		if err == nil {
			return nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		var status string
		if err := tx.QueryRowContext(ctx, `
			SELECT id, status FROM staging._load_run
			WHERE tenant_id = $1 AND source_system_cd = $2 AND domain = $3 AND run_ref = $4
			FOR UPDATE`, s.tenant, s.cfg.SourceCd, s.cfg.Domain, runRef).Scan(&s.runID, &status); err != nil {
			return err
		}
		switch status {
		case "COMPLETED":
			s.skip = true
			return nil
		case "RUNNING":
			return fmt.Errorf("a load for run_ref %q is already running", runRef)
		}
		// FAILED/PARTIAL/CANCELLED: clear what it staged and retry under the same run.
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(`DELETE FROM %s WHERE _load_run_id = $1`, pq.QuoteIdentifier(schema)+"."+pq.QuoteIdentifier(table)), s.runID); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `UPDATE staging._load_run SET status = 'RUNNING', started_at = now(), completed_at = NULL, error_summary = NULL WHERE id = $1`, s.runID)
		return err
	})
}

// columnsFor fixes the COPY column list from the mapping, or (identity
// mapping) from the first batch's fields that are real columns.
func (s *stagingSink) columnsFor(rows []Row) ([]string, []string, error) {
	if s.cols == nil {
		s.cols = map[string]string{}
		for f := range rows[0].Data {
			if !s.valid[f] {
				return nil, nil, fmt.Errorf("field %q has no column in %s (map it, or add the column)", f, s.cfg.Table)
			}
			if strings.HasPrefix(f, "_") || f == "tenant_id" {
				return nil, nil, fmt.Errorf("field %q would overwrite a load-tracking column", f)
			}
			s.cols[f] = f
		}
	}
	fields := make([]string, 0, len(s.cols))
	for f := range s.cols {
		fields = append(fields, f)
	}
	if len(fields) == 0 {
		// Never write rows that carry only load-tracking columns.
		return nil, nil, fmt.Errorf("no field of the incoming rows is mapped to a column of %s", s.cfg.Table)
	}
	sort.Strings(fields)
	cols := make([]string, len(fields))
	for i, f := range fields {
		cols[i] = s.cols[f]
	}
	return fields, cols, nil
}

func (s *stagingSink) Process(ctx context.Context, rows []Row) (Result, error) {
	if len(rows) == 0 {
		return Result{Out: rows}, nil
	}
	if s.skip && !s.dryRun {
		return Result{Out: rows}, nil
	}
	fields, cols, err := s.columnsFor(rows)
	if err != nil {
		return Result{}, err
	}
	// Reject, row by row, values the table cannot hold - one bad value must
	// not fail the whole load with a database error.
	var res Result
	good := rows[:0:0]
	for _, r := range rows {
		if field, reason := s.fits(fields, r); reason != "" {
			res.Rejected = append(res.Rejected, Reject{Row: r, Field: field, Reason: reason})
			continue
		}
		good = append(good, r)
	}
	res.Out = good
	if s.dryRun || len(good) == 0 {
		return res, nil
	}
	rows = good
	schema, table := splitTable(s.cfg.Table)
	target := pq.QuoteIdentifier(schema) + "." + pq.QuoteIdentifier(table)
	all := append([]string{"_load_run_id", "_source_row_num", "tenant_id"}, cols...)
	quoted := make([]string, len(all))
	for i, c := range all {
		quoted[i] = pq.QuoteIdentifier(c)
	}
	// Postgres refuses COPY into a row-level-secured table, so COPY into a
	// transaction-scoped temp table and INSERT ... SELECT from it: the
	// target's tenant policy (WITH CHECK) still judges every row.
	err = s.inTenantTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(`CREATE TEMP TABLE _pipeline_load (LIKE %s INCLUDING DEFAULTS) ON COMMIT DROP`, target)); err != nil {
			return err
		}
		stmt, err := tx.PrepareContext(ctx, pq.CopyIn("_pipeline_load", all...))
		if err != nil {
			return err
		}
		for _, r := range rows {
			args := []any{s.runID, r.Num, s.tenant}
			for _, f := range fields {
				args = append(args, r.Data[f])
			}
			if _, err := stmt.ExecContext(ctx, args...); err != nil {
				stmt.Close()
				return fmt.Errorf("row %d: %w", r.Num, err)
			}
		}
		if _, err := stmt.ExecContext(ctx); err != nil {
			stmt.Close()
			return err
		}
		if err := stmt.Close(); err != nil {
			return err
		}
		list := strings.Join(quoted, ", ")
		_, err = tx.ExecContext(ctx, fmt.Sprintf(`INSERT INTO %s (%s) SELECT %s FROM _pipeline_load`, target, list, list))
		return err
	})
	if err != nil {
		return Result{}, fmt.Errorf("loading %s: %w", s.cfg.Table, err)
	}
	s.loaded += len(rows)
	return res, nil
}

// colShape is what a staging column can hold.
type colShape struct {
	typ              string
	maxLen           int
	precision, scale int
}

// fits checks a row's mapped values against the target columns and returns
// the first field that does not fit, with a plain reason.
func (s *stagingSink) fits(fields []string, r Row) (string, string) {
	for _, f := range fields {
		v := r.Data[f]
		if v == nil {
			continue
		}
		col := s.cols[f]
		sh := s.shape[col]
		str := fmt.Sprint(v)
		switch {
		case sh.maxLen > 0 && utf8.RuneCountInString(str) > sh.maxLen:
			return f, fmt.Sprintf("%s: %q is longer than the %d characters %s allows", f, truncate(str, 40), sh.maxLen, col)
		case sh.typ == "numeric" || sh.typ == "integer" || sh.typ == "bigint" || sh.typ == "smallint" || sh.typ == "double precision" || sh.typ == "real":
			x, err := strconv.ParseFloat(strings.TrimSpace(str), 64)
			if err != nil {
				return f, fmt.Sprintf("%s: %q is not a number", f, truncate(str, 40))
			}
			if strings.Contains(sh.typ, "int") && x != math.Trunc(x) {
				return f, fmt.Sprintf("%s: %q is not a whole number", f, truncate(str, 40))
			}
			if sh.typ == "numeric" && sh.precision > 0 {
				intDigits := len(strings.TrimLeft(strings.Split(strings.TrimLeft(strings.TrimSpace(str), "+-"), ".")[0], "0"))
				if intDigits > sh.precision-sh.scale {
					return f, fmt.Sprintf("%s: %q is too large for %s", f, truncate(str, 40), col)
				}
			}
		case sh.typ == "date":
			if _, err := time.Parse("2006-01-02", strings.TrimSpace(str)); err != nil {
				return f, fmt.Sprintf("%s: %q is not a date (YYYY-MM-DD)", f, truncate(str, 40))
			}
		case sh.typ == "boolean":
			switch strings.ToLower(strings.TrimSpace(str)) {
			case "true", "false", "t", "f", "1", "0", "yes", "no", "y", "n":
			default:
				return f, fmt.Sprintf("%s: %q is not true/false", f, truncate(str, 40))
			}
		}
	}
	return "", ""
}

func truncate(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n]) + "…"
}

func (s *stagingSink) Close(ctx context.Context, runErr error) error {
	if s.runID == "" || s.skip {
		return nil
	}
	status, summary := "COMPLETED", sql.NullString{}
	if runErr != nil {
		status, summary = "FAILED", sql.NullString{String: runErr.Error(), Valid: true}
	}
	// Use a fresh context: the run's may already be cancelled.
	ctx = context.WithoutCancel(ctx)
	return s.inTenantTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
			UPDATE staging._load_run
			SET status = $2, completed_at = now(), accepted_rows = $3, error_summary = $4
			WHERE id = $1`, s.runID, status, s.loaded, summary)
		return err
	})
}

// StagingTable is a loadable staging table and its target columns.
type StagingTable struct {
	Table   string        `json:"table"`
	Columns []TargetField `json:"columns"`
}

// StagingTables lists staging.* tables that carry the load-tracking columns,
// with their loadable columns (load-tracking ones excluded).
func StagingTables(ctx context.Context, db *sql.DB) ([]StagingTable, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT c.table_name, c.column_name, c.data_type, c.is_nullable = 'NO'
		FROM information_schema.columns c
		WHERE c.table_schema = 'staging'
		  AND EXISTS (SELECT 1 FROM information_schema.columns k
		              WHERE k.table_schema = 'staging' AND k.table_name = c.table_name AND k.column_name = '_load_run_id')
		  AND c.column_name NOT LIKE '\_%' AND c.column_name <> 'tenant_id'
		ORDER BY c.table_name, c.ordinal_position`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []StagingTable{}
	for rows.Next() {
		var table, col, typ string
		var required bool
		if err := rows.Scan(&table, &col, &typ, &required); err != nil {
			return nil, err
		}
		if len(out) == 0 || out[len(out)-1].Table != "staging."+table {
			out = append(out, StagingTable{Table: "staging." + table})
		}
		t := &out[len(out)-1]
		t.Columns = append(t.Columns, TargetField{Name: col, Type: pgTypeName(typ), Required: required})
	}
	return out, rows.Err()
}

func pgTypeName(t string) string {
	switch {
	case strings.Contains(t, "int"):
		return "int"
	case t == "numeric" || strings.Contains(t, "double") || t == "real":
		return "decimal"
	case t == "boolean":
		return "bool"
	case t == "date":
		return "date"
	case strings.HasPrefix(t, "timestamp"):
		return "timestamp"
	}
	return "string"
}
