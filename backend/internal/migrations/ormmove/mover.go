// Package ormmove copies a tenant's ORM (crims) rows out of the one shared
// database and into that tenant's own database, and then proves the copy is
// complete (ADR-043).
//
// It is deliberately not a dump-and-restore. The properties that matter:
//
//   - Dependency order. Tables are copied parents-first, from the orm->orm
//     foreign-key graph, so a placement never lands before its order.
//   - Idempotent. Every insert is ON CONFLICT DO NOTHING, so a rerun is a no-op
//     and an interrupted fleet resumes instead of dying on a duplicate key.
//   - Verified. Every table is counted on both sides with the same predicate and
//     a target is Done only when the counts agree. A partial copy that reported
//     success would be worse than no copy, because the cutover is a one-way door
//     for writes.
//   - Read-only reference data comes along. The RLS policies this replaces let a
//     tenant read its own rows *or* the shared reference tenant's, so the copy
//     takes both, and nothing in a tenant database ever writes the reference rows.
package ormmove

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ReferenceTenant is the shared-reference sentinel the RLS read policies used:
// 00000000-0000-0000-0000-000000000001. Its rows are readable by every tenant
// and writable by none, which is why they are copied but never targeted for a
// write.
const ReferenceTenant = "00000000-0000-0000-0000-000000000001"

// Table is one ORM table in the move.
type Table struct {
	// Name is the table in the orm schema, without the schema prefix.
	Name string
}

// Tables is the copy order: parents before children, derived from the orm->orm
// foreign-key graph. Three of the graph's 16 edges point at `oms` and are not
// here at all -- they are the constraints ADR-042 drops, because a foreign key
// cannot cross a database boundary. `position_lot` therefore has no internal
// parent and could be copied anywhere; it is kept in this order for readability.
//
// TestTablesRespectDeclaredEdges fails if this list and TestEdgeCoverage drift
// apart, so the order cannot rot silently.
var Tables = []Table{
	// No orm-internal parent.
	{"order"},
	{"broker"},
	{"account"},
	{"basket"},
	{"model_portfolio"},
	{"position_lot"},
	{"position_history"},
	{"cash_balance"},
	{"market_data_snapshot"},
	{"pre_trade_check"},
	{"restricted_list"},
	{"short_sell_locate"},
	{"trading_session"},
	{"trading_halt"},
	{"trading_limit"},
	{"routing_rule"},
	{"pnl_intraday"},
	{"fx_exposure"},
	{"fx_hedge"},

	// Children of orm."order".
	{"placement"},
	{"order_allocation"},
	{"order_amendment"},
	{"order_benchmark"},
	{"order_event"},
	{"order_history"},
	{"order_reject"},

	// Children of basket / model_portfolio.
	{"basket_item"},
	{"model_portfolio_target"},
	{"account_model_assignment"},

	// execution needs order AND placement; the two below need execution.
	{"execution"},
	{"execution_allocation"},
	{"execution_quality"},
}

// Edges is the orm->orm foreign-key graph, child -> parents, as declared by the
// migration set. It exists so TestTablesRespectDeclaredEdges can prove Tables is
// a valid topological order rather than asking the reader to trust it.
var Edges = map[string][]string{
	"placement":                {"order"},
	"order_allocation":         {"order"},
	"order_amendment":          {"order"},
	"order_benchmark":          {"order"},
	"order_event":              {"order"},
	"order_history":            {"order"},
	"order_reject":             {"order"},
	"pre_trade_check":          {"order"},
	"basket_item":              {"basket"},
	"model_portfolio_target":   {"model_portfolio"},
	"account_model_assignment": {"model_portfolio"},
	"execution":                {"order", "placement"},
	"execution_allocation":     {"execution", "order_allocation"},
	"execution_quality":        {"execution"},
}

// TableCount is the number of ORM tables the move copies: 32 of the 33 real
// tables. `quote` is deliberately excluded (see ExcludedTables). The schema has
// 34 names, the extra one being `quote_default`, a partition of `quote`.
const TableCount = 32

// ExcludedTables are real ORM tables the move does NOT copy, with the reason
// recorded here rather than left to whoever notices a missing table.
//
// The trade is deliberate and reversible: a tenant's order chain is its own and
// must move, while a market-data quote is a shared observation that is identical
// for every tenant. Copying it per tenant multiplies a high-volume, RANGE-
// partitioned table by the tenant count to store identical bytes. It stays
// readable from `alpha` (or the warm tier) instead. If per-tenant quote
// residency is ever wanted -- a regulated tenant that must not read another's
// market data view, or a latency requirement -- the change is to delete this
// entry and re-run, not a redesign.
var ExcludedTables = map[string]string{
	"quote": "shared market data: identical for every tenant, high volume, RANGE-partitioned by quote_time; copied per tenant it multiplies storage by the tenant count to hold the same bytes",
}

// TableCounts is one table's verification: what the source held, what the target
// now holds, and whether they agree.
//
// Error is a hard failure (unreadable source, unwritable target, a row that will
// not scan) and stops the tenant. Detail is a count disagreement, which does not:
// the run continues so that ONE pass reports every table that is wrong, and the
// tenant is still not done.
type TableCounts struct {
	Table      string `json:"table"`
	Source     int64  `json:"source"`
	Target     int64  `json:"target"`
	Inserted   int64  `json:"inserted"`
	Mismatched bool   `json:"mismatched,omitempty"`
	Detail     string `json:"detail,omitempty"`
	Error      string `json:"error,omitempty"`
}

// Report is one tenant's move.
type Report struct {
	TenantID string        `json:"tenantID"`
	Counts   []TableCounts `json:"counts"`
	// Mismatched is the number of tables whose source and target counts disagree.
	// Done is true only when it is zero and nothing errored.
	Mismatched int           `json:"mismatched"`
	Done       bool          `json:"done"`
	Duration   time.Duration `json:"duration"`
	Error      string        `json:"error,omitempty"`
}

// Mover copies one tenant's rows from a shared source database into a per-tenant
// target database. Source and Target are injected so the mover never decides how
// a database is reached or authorized: the Fleet's Connector owns that.
type Mover struct {
	// Source is the shared database holding every tenant's rows today.
	Source *sql.DB
	// ReferenceTenant is the sentinel whose rows are copied read-only. Empty means
	// ReferenceTenant's own value.
	ReferenceTenant string
	// BatchSize is how many rows go into one INSERT. 0 means 200. It bounds how
	// many bind parameters one statement carries (Postgres allows 65535).
	BatchSize int
}

// batch is the effective batch size.
func (m *Mover) batch() int {
	if m.BatchSize > 0 {
		return m.BatchSize
	}
	return 200
}

func (m *Mover) reference() string {
	if m.ReferenceTenant != "" {
		return m.ReferenceTenant
	}
	return ReferenceTenant
}

// Move copies every table for the tenant and verifies each one.
//
// A hard error stops immediately. A count disagreement does not: it is recorded
// and the run continues, so one pass reports every table that will not reconcile
// instead of only the first. Either way the tenant is not Done, and the caller
// must not cut it over on a partial move.
func (m *Mover) Move(ctx context.Context, target *sql.DB, tenantID string) (Report, error) {
	start := time.Now()
	rep := Report{TenantID: tenantID}

	for _, t := range Tables {
		c := m.moveTable(ctx, target, t.Name, tenantID)
		rep.Counts = append(rep.Counts, c)
		if c.Error != "" {
			rep.Mismatched++
			rep.Error = fmt.Sprintf("%s: %s", t.Name, c.Error)
			rep.Duration = time.Since(start)
			return rep, fmt.Errorf("ormmove: tenant %s table %s: %s", tenantID, t.Name, c.Error)
		}
		if c.Mismatched {
			rep.Mismatched++
		}
	}
	rep.Done = rep.Mismatched == 0
	rep.Duration = time.Since(start)
	if !rep.Done {
		rep.Error = fmt.Sprintf("%d of %d tables do not match the source", rep.Mismatched, len(Tables))
	}
	return rep, nil
}

func (m *Mover) moveTable(ctx context.Context, target *sql.DB, table, tenantID string) TableCounts {
	c := TableCounts{Table: table}
	ref := m.reference()

	// A constant, not a format: the tenant and the sentinel are always bound as
	// query parameters ($1, $2), never interpolated into the SQL text.
	const where = "tenant_id = $1::uuid OR tenant_id = $2::uuid"

	cols, err := m.columns(ctx, table)
	if err != nil {
		c.Error = "source columns: " + err.Error()
		return c
	}
	var src int64
	if err := m.Source.QueryRowContext(ctx,
		fmt.Sprintf("SELECT count(*) FROM orm.%s WHERE %s", q(table), where),
		tenantID, ref).Scan(&src); err != nil {
		c.Error = "count source: " + err.Error()
		return c
	}
	c.Source = src

	inserted, err := m.copyIn(ctx, target, table, cols, where, tenantID, ref)
	if err != nil {
		c.Error = "copy: " + err.Error()
		return c
	}
	c.Inserted = inserted

	var dst int64
	if err := target.QueryRowContext(ctx,
		fmt.Sprintf("SELECT count(*) FROM orm.%s WHERE %s", q(table), where),
		tenantID, ref).Scan(&dst); err != nil {
		c.Error = "count target: " + err.Error()
		return c
	}
	c.Target = dst
	c.Mismatched = dst != src
	if c.Mismatched {
		// A Detail, not an Error: the run continues so every bad table is reported.
		c.Detail = fmt.Sprintf("source has %d rows, target has %d", src, dst)
	}
	return c
}

// copyIn streams the tenant's rows from the source into the target.
//
// It reads on the source connection and writes on the target connection, in
// batches inside one transaction per table. It cannot be a single
// `INSERT INTO target SELECT ... FROM source`: that statement would resolve BOTH
// table names against the one connection it runs on, so it would read the
// target's own (empty) table and copy nothing while reporting success. The two
// databases may also be on different clusters, which no single-server statement
// could span anyway.
//
// Batching here is safe in a way it was not with ON CONFLICT and LIMIT/OFFSET:
// the row set is read once, up front, so a batch can never be short because a
// neighbour conflicted.
func (m *Mover) copyIn(ctx context.Context, target *sql.DB, table string, cols []string, where, tenantID, ref string) (int64, error) {
	colList := strings.Join(quoteAll(cols), ", ")
	rows, err := m.Source.QueryContext(ctx,
		fmt.Sprintf("SELECT %s FROM orm.%s WHERE %s", colList, q(table), where),
		tenantID, ref)
	if err != nil {
		return 0, fmt.Errorf("read source: %w", err)
	}
	defer rows.Close()

	tx, err := target.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin: %w", err)
	}
	// One table per transaction: a bad row rolls that table back whole, and the
	// count check then reports the tenant as not done rather than half-copied.
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	var (
		total int64
		batch [][]any
	)
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		holders := make([]string, len(batch))
		args := make([]any, 0, len(batch)*len(cols))
		n := 1
		for i, r := range batch {
			holders[i] = "(" + placeholders(n, len(cols)) + ")"
			n += len(cols)
			args = append(args, r...)
		}
		res, err := tx.ExecContext(ctx, fmt.Sprintf(
			"INSERT INTO orm.%s (%s) VALUES %s ON CONFLICT DO NOTHING",
			q(table), colList, strings.Join(holders, ",")), args...)
		if err != nil {
			return err
		}
		affected, err := res.RowsAffected()
		if err == nil {
			total += affected
		}
		batch = batch[:0]
		return nil
	}

	for rows.Next() {
		row := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range row {
			ptrs[i] = &row[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return total, fmt.Errorf("scan source row: %w", err)
		}
		batch = append(batch, row)
		if len(batch) >= m.batch() {
			if err := flush(); err != nil {
				return total, fmt.Errorf("write batch: %w", err)
			}
		}
	}
	if err := rows.Err(); err != nil {
		return total, fmt.Errorf("read source: %w", err)
	}
	if err := flush(); err != nil {
		return total, fmt.Errorf("write batch: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return total, fmt.Errorf("commit: %w", err)
	}
	committed = true
	return total, nil
}

// columns reads the source table's column list, so the INSERT names the same
// columns on both sides instead of trusting a hand-maintained list that would
// drift the first time alpha gains a column.
func (m *Mover) columns(ctx context.Context, table string) ([]string, error) {
	rows, err := m.Source.QueryContext(ctx, `
		SELECT column_name
		FROM information_schema.columns
		WHERE table_schema = 'orm' AND table_name = $1
		ORDER BY ordinal_position
	`, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cols []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		cols = append(cols, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(cols) == 0 {
		// An empty list would silently copy nothing and then "verify" 0 == 0.
		return nil, fmt.Errorf("orm.%s has no columns in the source database", table)
	}
	return cols, nil
}

func quoteAll(cols []string) []string {
	out := make([]string, len(cols))
	for i, c := range cols {
		out[i] = `"` + strings.ReplaceAll(c, `"`, `""`) + `"`
	}
	return out
}

// placeholders renders n consecutive bind markers starting at from, so a
// multi-row VALUES lists every column of every row and never reuses a $n.
func placeholders(from, n int) string {
	parts := make([]string, n)
	for i := 0; i < n; i++ {
		parts[i] = "$" + strconv.Itoa(from+i)
	}
	return strings.Join(parts, ",")
}

func q(ident string) string {
	return `"` + strings.ReplaceAll(ident, `"`, `""`) + `"`
}
