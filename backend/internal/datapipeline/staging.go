package datapipeline

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"

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
	skip   bool              // run_ref already COMPLETED, or a dry run
	cols   map[string]string // row field -> staging column
	valid  map[string]bool   // real columns of the table
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
		rows, err := tx.QueryContext(ctx, `SELECT column_name FROM information_schema.columns WHERE table_schema = $1 AND table_name = $2`, schema, table)
		if err != nil {
			return err
		}
		s.valid = map[string]bool{}
		for rows.Next() {
			var c string
			if err := rows.Scan(&c); err != nil {
				rows.Close()
				return err
			}
			s.valid[c] = true
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
		s.cols = s.cfg.Columns
		if rc.DryRun {
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
	sort.Strings(fields)
	cols := make([]string, len(fields))
	for i, f := range fields {
		cols[i] = s.cols[f]
	}
	return fields, cols, nil
}

func (s *stagingSink) Process(ctx context.Context, rows []Row) (Result, error) {
	if s.skip || len(rows) == 0 {
		return Result{Out: rows}, nil
	}
	fields, cols, err := s.columnsFor(rows)
	if err != nil {
		return Result{}, err
	}
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
	return Result{Out: rows}, nil
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
