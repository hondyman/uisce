package migrations

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// This file is the runner for a TENANT's own database (ADR-030). It is deliberately separate
// from ApplyMigrations, which owns alpha and is left exactly as it was.
//
// Differences from the alpha runner, on purpose:
//   - Drift is fatal here. The alpha runner warns and skips a file whose content changed after
//     it was applied; for a fleet of tenant databases that would let one database quietly differ
//     from the rest. Any drift stops the target and is reported.
//   - Applying is append-only: a file that sorts BEFORE one already applied is out-of-order drift,
//     not a migration to run.
//   - The result is a Report, a value the provisioning saga can read as its "migrations applied"
//     probe, not a log line.

// ErrAlphaTarget: "alpha" is migrated by ApplyMigrations, not by this runner.
var ErrAlphaTarget = errors.New("migrations: the alpha target is applied by ApplyMigrations")

var (
	// ErrDrift: an applied file changed, vanished, or a new file sorts before an applied one.
	ErrDrift = errors.New("migrations: drift detected; refusing to apply anything")
	// ErrTargetBusy: another runner holds this database's migration lock.
	ErrTargetBusy = errors.New("migrations: another runner is migrating this database")
	// ErrNoApp: the app has no migration directory.
	ErrNoApp = errors.New("migrations: no migration directory for this app")
	// ErrBaselineNotEmpty: a baseline can only be recorded on a database that has applied nothing.
	ErrBaselineNotEmpty = errors.New("migrations: baseline refused; this database already has applied migrations")
)

var appName = regexp.MustCompile(`^[a-z][a-z0-9_]{0,31}$`)

// Target names one tenant database's schema for one app: "tenant:<tenant id>:<app>".
type Target struct {
	TenantID string
	App      string
}

func (t Target) String() string { return "tenant:" + t.TenantID + ":" + t.App }

// Validate rejects anything that could name a path or a tenant other than itself. App becomes a
// directory name, so it is a strict identifier, never a path.
func (t Target) Validate() error {
	if _, err := uuid.Parse(t.TenantID); err != nil {
		return fmt.Errorf("migrations: target tenant id %q is not a uuid", t.TenantID)
	}
	if !appName.MatchString(t.App) {
		return fmt.Errorf("migrations: target app %q must match %s", t.App, appName)
	}
	return nil
}

// ParseTarget parses "tenant:<id>:<app>". "alpha" is ErrAlphaTarget so a CLI can dispatch.
func ParseTarget(s string) (Target, error) {
	if s == "alpha" {
		return Target{}, ErrAlphaTarget
	}
	parts := strings.Split(s, ":")
	if len(parts) != 3 || parts[0] != "tenant" {
		return Target{}, fmt.Errorf("migrations: target %q must be \"alpha\" or \"tenant:<tenant id>:<app>\"", s)
	}
	t := Target{TenantID: parts[1], App: parts[2]}
	return t, t.Validate()
}

// DriftKind says how an applied migration no longer matches the files on disk.
type DriftKind string

const (
	DriftChanged    DriftKind = "changed"      // applied, but the file's sha256 is different now
	DriftMissing    DriftKind = "missing"      // applied, but the file is gone
	DriftOutOfOrder DriftKind = "out_of_order" // not applied, yet it sorts before one that was
)

// Drift is one finding.
type Drift struct {
	Filename    string    `json:"filename"`
	Kind        DriftKind `json:"kind"`
	RecordedSHA string    `json:"recorded_sha256,omitempty"`
	CurrentSHA  string    `json:"current_sha256,omitempty"`
}

// AppliedFile is one row of the database's migration log.
type AppliedFile struct {
	Filename  string    `json:"filename"`
	SHA256    string    `json:"sha256"`
	Baseline  bool      `json:"baseline,omitempty"` // recorded, not executed (the cloned schema already had it)
	AppliedAt time.Time `json:"applied_at"`
}

// Report is the state of one target. It is the shape the provisioning saga reads: Done is true
// only when every file on disk is applied, none has drifted, and nothing went wrong.
type Report struct {
	Target  string        `json:"target"`
	Applied []AppliedFile `json:"applied"`
	Pending []string      `json:"pending"`
	Drift   []Drift       `json:"drift"`
	Ran     []string      `json:"ran"` // applied by THIS call
	Done    bool          `json:"done"`
	Error   string        `json:"error,omitempty"`
}

// TenantRunner applies a directory of per-app migrations to tenant databases.
type TenantRunner struct {
	// Root holds one directory per app: <Root>/<app>/*.up.sql.
	Root string
	// Generated, when set, supplies the migrations of an app that has no directory because they are compiled
	// from alpha's catalog scan (ADR-050). It returns (files, true, nil) for an app it owns, and (nil, false, nil) for
	// any other, which then comes from Root as before. A generated file is recorded in the same migration log by name
	// and sha256, so a database built from a different compilation is drift exactly as a changed file is.
	Generated func(Target) ([]GeneratedFile, bool, error)
}

// GeneratedFile is one migration that exists only in memory.
type GeneratedFile struct {
	Filename string // e.g. 0001_structure.up.sql; must end in .up.sql and sort in apply order
	SQL      string
}

// DefaultTenantRoot resolves backend/db/tenant_migrations from the repo root or backend/.
func DefaultTenantRoot() string {
	dir := "db/tenant_migrations"
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		dir = "../db/tenant_migrations"
	}
	return dir
}

const logDDL = `
CREATE SCHEMA IF NOT EXISTS ivy_meta;
CREATE TABLE IF NOT EXISTS ivy_meta.migration_log (
    filename   TEXT PRIMARY KEY,
    sha256     TEXT NOT NULL,
    baseline   BOOLEAN NOT NULL DEFAULT false,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
)`

func (r *TenantRunner) files(t Target) (names []string, shas map[string]string, err error) {
	names, shas, _, err = r.source(t)
	return names, shas, err
}

// source lists an app's migrations, their sha256 and a way to read each one, from memory for a generated app and from
// its directory otherwise.
func (r *TenantRunner) source(t Target) (names []string, shas map[string]string, read func(string) (string, error), err error) {
	if err := t.Validate(); err != nil {
		return nil, nil, nil, err
	}
	if r.Generated != nil {
		gen, ok, gerr := r.Generated(t)
		if gerr != nil {
			return nil, nil, nil, gerr
		}
		if ok {
			bodies := make(map[string]string, len(gen))
			shas = make(map[string]string, len(gen))
			for _, g := range gen {
				if !strings.HasSuffix(g.Filename, ".up.sql") || strings.ContainsAny(g.Filename, `/\`) {
					return nil, nil, nil, fmt.Errorf("migrations: generated file name %q must be a plain *.up.sql name", g.Filename)
				}
				if _, dup := bodies[g.Filename]; dup {
					return nil, nil, nil, fmt.Errorf("migrations: generated file %q appears twice", g.Filename)
				}
				sum := sha256.Sum256([]byte(g.SQL))
				bodies[g.Filename], shas[g.Filename] = g.SQL, hex.EncodeToString(sum[:])
				names = append(names, g.Filename)
			}
			sort.Strings(names)
			return names, shas, func(n string) (string, error) { return bodies[n], nil }, nil
		}
	}
	dir := filepath.Join(r.Root, t.App)
	if st, serr := os.Stat(dir); serr != nil || !st.IsDir() {
		return nil, nil, nil, fmt.Errorf("%w: %s", ErrNoApp, t.App)
	}
	names, err = listUpFiles(dir)
	if err != nil {
		return nil, nil, nil, err
	}
	shas = make(map[string]string, len(names))
	for _, n := range names {
		if shas[n], err = fileSHA256(dir, n); err != nil {
			return nil, nil, nil, err
		}
	}
	read = func(n string) (string, error) {
		b, err := os.ReadFile(filepath.Join(dir, n))
		return string(b), err
	}
	return names, shas, read, nil
}

type queryer interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func loadLog(ctx context.Context, q queryer) ([]AppliedFile, error) {
	var exists bool
	if err := q.QueryRowContext(ctx, `SELECT to_regclass('ivy_meta.migration_log') IS NOT NULL`).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil // never migrated: everything is pending
	}
	rows, err := q.QueryContext(ctx, `SELECT filename, sha256, baseline, applied_at FROM ivy_meta.migration_log ORDER BY filename`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AppliedFile
	for rows.Next() {
		var a AppliedFile
		if err := rows.Scan(&a.Filename, &a.SHA256, &a.Baseline, &a.AppliedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// analyze is pure: what is pending and what has drifted, given the files and the log.
func analyze(target Target, names []string, shas map[string]string, applied []AppliedFile) Report {
	rep := Report{Target: target.String(), Applied: applied, Pending: []string{}, Drift: []Drift{}, Ran: []string{}}
	if rep.Applied == nil {
		rep.Applied = []AppliedFile{}
	}
	done := make(map[string]bool, len(applied))
	last := ""
	for _, a := range applied {
		done[a.Filename] = true
		if a.Filename > last {
			last = a.Filename
		}
		cur, onDisk := shas[a.Filename]
		switch {
		case !onDisk:
			rep.Drift = append(rep.Drift, Drift{Filename: a.Filename, Kind: DriftMissing, RecordedSHA: a.SHA256})
		case cur != a.SHA256:
			rep.Drift = append(rep.Drift, Drift{Filename: a.Filename, Kind: DriftChanged, RecordedSHA: a.SHA256, CurrentSHA: cur})
		}
	}
	for _, n := range names {
		if done[n] {
			continue
		}
		if n < last {
			rep.Drift = append(rep.Drift, Drift{Filename: n, Kind: DriftOutOfOrder})
			continue
		}
		rep.Pending = append(rep.Pending, n)
	}
	sort.Slice(rep.Drift, func(i, j int) bool { return rep.Drift[i].Filename < rep.Drift[j].Filename })
	rep.Done = len(rep.Pending) == 0 && len(rep.Drift) == 0
	return rep
}

// Inspect reports a target without changing it.
func (r *TenantRunner) Inspect(ctx context.Context, db *sql.DB, t Target) (Report, error) {
	names, shas, err := r.files(t)
	if err != nil {
		return Report{Target: t.String(), Error: err.Error()}, err
	}
	applied, err := loadLog(ctx, db)
	if err != nil {
		return Report{Target: t.String(), Error: err.Error()}, fmt.Errorf("read migration log: %w", err)
	}
	return analyze(t, names, shas, applied), nil
}

// Apply applies every pending file, in order, one transaction each. It refuses to apply anything
// while drift exists, and it holds a database-wide lock so two runners cannot interleave. A file
// that fails rolls back whole and stops the run; files before it stay applied, and a later Apply
// resumes from the failed one.
func (r *TenantRunner) Apply(ctx context.Context, db *sql.DB, t Target) (Report, error) {
	names, shas, read, err := r.source(t)
	if err != nil {
		return Report{Target: t.String(), Error: err.Error()}, err
	}
	// One pinned connection: the advisory lock and search_path are per connection.
	conn, err := db.Conn(ctx)
	if err != nil {
		return Report{Target: t.String(), Error: err.Error()}, fmt.Errorf("connect: %w", err)
	}
	defer conn.Close()

	var got bool
	if err := conn.QueryRowContext(ctx, `SELECT pg_try_advisory_lock(hashtext('ivy_tenant_migrate'))`).Scan(&got); err != nil {
		return Report{Target: t.String(), Error: err.Error()}, fmt.Errorf("take migration lock: %w", err)
	}
	if !got {
		return Report{Target: t.String(), Error: ErrTargetBusy.Error()}, ErrTargetBusy
	}
	defer func() {
		_, _ = conn.ExecContext(context.Background(), `SELECT pg_advisory_unlock(hashtext('ivy_tenant_migrate'))`)
	}()

	if _, err := conn.ExecContext(ctx, `SET search_path TO public`); err != nil {
		return Report{Target: t.String(), Error: err.Error()}, fmt.Errorf("pin search_path: %w", err)
	}
	if _, err := conn.ExecContext(ctx, logDDL); err != nil {
		return Report{Target: t.String(), Error: err.Error()}, fmt.Errorf("create migration log: %w", err)
	}
	applied, err := loadLog(ctx, conn)
	if err != nil {
		return Report{Target: t.String(), Error: err.Error()}, fmt.Errorf("read migration log: %w", err)
	}
	rep := analyze(t, names, shas, applied)
	if len(rep.Drift) > 0 {
		rep.Error = ErrDrift.Error()
		return rep, ErrDrift
	}

	for _, name := range append([]string(nil), rep.Pending...) {
		if err := ctx.Err(); err != nil {
			rep.Error = err.Error()
			return rep, err
		}
		if err := applyOne(ctx, conn, read, name, shas[name]); err != nil {
			rep.Error = err.Error()
			return rep, err
		}
		rep.Ran = append(rep.Ran, name)
		rep.Pending = rep.Pending[1:]
		rep.Applied = append(rep.Applied, AppliedFile{Filename: name, SHA256: shas[name], AppliedAt: time.Now()})
	}
	rep.Done = len(rep.Pending) == 0
	return rep, nil
}

func applyOne(ctx context.Context, conn *sql.Conn, read func(string) (string, error), name, sha string) error {
	content, err := read(name)
	if err != nil {
		return fmt.Errorf("read %s: %w", name, err)
	}
	if hasTransactionControl(content) {
		return fmt.Errorf("migration %s contains transaction-control statements (COMMIT/ROLLBACK); the runner owns the transaction", name)
	}
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin %s: %w", name, err)
	}
	if _, err := tx.ExecContext(ctx, content); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("apply %s: %w", name, err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO ivy_meta.migration_log (filename, sha256) VALUES ($1, $2)`, name, sha); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("log %s: %w", name, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit %s: %w", name, err)
	}
	return nil
}

// Baseline records every file up to and including `through` as applied WITHOUT running it. The
// provisioning saga uses it right after cloning the gold-copy schema, which already contains
// those migrations' effects, so the runner knows where the clone stood. It refuses a database
// that has applied anything: a baseline must never paper over a real history.
func (r *TenantRunner) Baseline(ctx context.Context, db *sql.DB, t Target, through string) (Report, error) {
	names, shas, err := r.files(t)
	if err != nil {
		return Report{Target: t.String(), Error: err.Error()}, err
	}
	if _, ok := shas[through]; !ok {
		err := fmt.Errorf("migrations: baseline file %q is not in %s", through, t.App)
		return Report{Target: t.String(), Error: err.Error()}, err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return Report{Target: t.String(), Error: err.Error()}, err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtext('ivy_tenant_migrate'))`); err != nil {
		return Report{Target: t.String(), Error: err.Error()}, err
	}
	if _, err := tx.ExecContext(ctx, logDDL); err != nil {
		return Report{Target: t.String(), Error: err.Error()}, fmt.Errorf("create migration log: %w", err)
	}
	existing, err := loadLog(ctx, tx)
	if err != nil {
		return Report{Target: t.String(), Error: err.Error()}, err
	}
	if len(existing) > 0 {
		return analyze(t, names, shas, existing), ErrBaselineNotEmpty
	}
	for _, n := range names {
		if n > through {
			break
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO ivy_meta.migration_log (filename, sha256, baseline) VALUES ($1, $2, true)`, n, shas[n]); err != nil {
			return Report{Target: t.String(), Error: err.Error()}, fmt.Errorf("record baseline %s: %w", n, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return Report{Target: t.String(), Error: err.Error()}, err
	}
	return r.Inspect(ctx, db, t)
}
