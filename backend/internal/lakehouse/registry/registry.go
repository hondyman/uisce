// Package registry is the control-plane record of each tenant's one Iceberg
// warehouse (ADR-029, ADR-032): its per-tenant audit retention and its
// provisioning state, in public.tenant_lakehouse, with every change recorded in
// the hash-chained public.tenant_lakehouse_audit in the same transaction.
//
// Every read and write of a tenant's row runs under dbpkg.WithTenantTransaction
// for that tenant, so row-level security applies; nothing here uses a privileged
// cross-tenant role. The warehouse and bucket names are derived from the tenant id
// and are never accepted from a caller.
package registry

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	dbpkg "github.com/hondyman/uisce/backend/internal/db"
	"github.com/hondyman/uisce/backend/internal/iceberg"
)

const (
	// MinRetentionDays and MaxRetentionDays bound a per-tenant retention. There is
	// deliberately no default: compliance-mode retention cannot be shortened, so the
	// number is always an explicit decision.
	MinRetentionDays = 1
	MaxRetentionDays = 36500 // 100 years; a sanity bound, not a recommendation

	stateUnconfigured = "unconfigured"
)

var (
	ErrTenantNotFound   = errors.New("tenant not found")
	ErrInvalidRetention = fmt.Errorf("audit retention must be between %d and %d days", MinRetentionDays, MaxRetentionDays)
	ErrRetentionLowered = errors.New("audit retention can only be extended once set")

	// ErrNotConfigured: no registry row, or no audit retention set, so there is nothing
	// to provision against.
	ErrNotConfigured = errors.New("lakehouse is not configured: set the audit retention first")
	// ErrInvalidState: the tenant's lakehouse is not in a state that allows the change.
	ErrInvalidState = errors.New("lakehouse is not in a state that allows this")
	// ErrWarehouseMismatch: the tenant is already bound to a different warehouse. One
	// tenant has exactly one, and it is never silently replaced.
	ErrWarehouseMismatch = errors.New("tenant is already bound to a different warehouse")
)

// Config is a tenant's lakehouse registry entry. A tenant with no row is returned
// with Configured=false and LifecycleState "unconfigured", not as an error, so a
// form can be shown for a tenant that has never been configured.
type Config struct {
	TenantID           string `json:"tenant_id"`
	TenantName         string `json:"tenant_name,omitempty"`
	TenantCode         string `json:"tenant_code,omitempty"`
	Configured         bool   `json:"configured"`
	WarehouseName      string `json:"warehouse_name,omitempty"`
	Bucket             string `json:"bucket,omitempty"`
	AuditRetentionDays *int   `json:"audit_retention_days"`
	// RetentionAppliedDays is what the bucket enforces by default; AuditRetentionDays is
	// what the registry wants. RetentionPending is true while the bucket enforces less.
	RetentionAppliedDays *int   `json:"retention_applied_days"`
	RetentionPending     bool   `json:"retention_pending"`
	LifecycleState       string `json:"lifecycle_state"`
	Provisioned          bool   `json:"provisioned"`
	// AuditCopiedThroughID is the last audit entry known to be copied to the tenant's Iceberg
	// audit table. A status marker only: the copy resumes from the destination's own max(id).
	AuditCopiedThroughID *int64 `json:"audit_copied_through_id"`
	// CredentialIssued is true once the tenant's storage credential has been issued. After
	// that, a credential the secrets store cannot find is an error, never a reason to mint a
	// new one.
	CredentialIssued bool       `json:"credential_issued"`
	Version          int        `json:"version,omitempty"`
	UpdatedAt        *time.Time `json:"updated_at,omitempty"`
}

// Actor is who made a change; recorded in the audit trail.
type Actor struct {
	ID   string
	Role string
}

// AuditEntry is one link of a tenant's audit chain.
type AuditEntry struct {
	ID        int64           `json:"id"`
	At        time.Time       `json:"at"`
	ActorID   string          `json:"actor_id"`
	ActorRole string          `json:"actor_role"`
	Action    string          `json:"action"`
	Before    json.RawMessage `json:"before,omitempty"`
	After     json.RawMessage `json:"after,omitempty"`
	PrevHash  string          `json:"prev_hash"`
	Hash      string          `json:"hash"`
}

// Store reads and writes the registry.
type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store { return &Store{db: db} }

// tenantRow reads the tenant itself. public.tenants is read directly by existing
// admin endpoints (it is the list a global admin picks from), so this is not a
// cross-tenant privilege; it only establishes that the id names a real tenant.
func (s *Store) tenantRow(ctx context.Context, tenantID uuid.UUID) (name, code string, err error) {
	var n, c sql.NullString
	err = s.db.QueryRowContext(ctx, `SELECT name, code FROM public.tenants WHERE id = $1`, tenantID).Scan(&n, &c)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", ErrTenantNotFound
	}
	if err != nil {
		return "", "", fmt.Errorf("read tenant: %w", err)
	}
	return n.String, c.String, nil
}

const selectConfig = `
	SELECT warehouse_name, bucket, audit_retention_days, retention_applied_days, lifecycle_state,
	       lakekeeper_warehouse_id IS NOT NULL, credential_issued_at IS NOT NULL, audit_copied_through_id, version, updated_at
	  FROM public.tenant_lakehouse
	 WHERE tenant_id = $1`

func scanConfig(row *sql.Row, c *Config) error {
	var days, applied sql.NullInt32
	var copied sql.NullInt64
	var updated time.Time
	if err := row.Scan(&c.WarehouseName, &c.Bucket, &days, &applied, &c.LifecycleState, &c.Provisioned, &c.CredentialIssued, &copied, &c.Version, &updated); err != nil {
		return err
	}
	if copied.Valid {
		c.AuditCopiedThroughID = &copied.Int64
	}
	if days.Valid {
		d := int(days.Int32)
		c.AuditRetentionDays = &d
	}
	if applied.Valid {
		a := int(applied.Int32)
		c.RetentionAppliedDays = &a
	}
	c.RetentionPending = c.Provisioned && c.AuditRetentionDays != nil &&
		(c.RetentionAppliedDays == nil || *c.RetentionAppliedDays < *c.AuditRetentionDays)
	c.Configured = true
	c.UpdatedAt = &updated
	return nil
}

// Get returns the tenant's registry entry, or an "unconfigured" entry if it has none.
func (s *Store) Get(ctx context.Context, tenantID uuid.UUID) (*Config, error) {
	name, code, err := s.tenantRow(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	c := &Config{TenantID: tenantID.String(), TenantName: name, TenantCode: code, LifecycleState: stateUnconfigured}
	err = dbpkg.WithTenantTransaction(ctx, s.db, tenantID.String(), func(tx *sql.Tx) error {
		if e := scanConfig(tx.QueryRowContext(ctx, selectConfig, tenantID), c); e != nil && !errors.Is(e, sql.ErrNoRows) {
			return e
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("read lakehouse: %w", err)
	}
	return c, nil
}

// SetRetention sets a tenant's audit retention, creating its registry row if it has
// none. It can only extend: lowering or clearing a set retention is refused (the
// database enforces the same rule). The change and its audit record commit together.
// Setting the value it already has is a no-op and writes no audit row.
func (s *Store) SetRetention(ctx context.Context, tenantID uuid.UUID, days int, actor Actor) (*Config, error) {
	if days < MinRetentionDays || days > MaxRetentionDays {
		return nil, ErrInvalidRetention
	}
	if _, _, err := s.tenantRow(ctx, tenantID); err != nil {
		return nil, err
	}
	wh, err := iceberg.TenantWarehouseName(tenantID)
	if err != nil {
		return nil, err
	}

	err = dbpkg.WithTenantTransaction(ctx, s.db, tenantID.String(), func(tx *sql.Tx) error {
		// Create-if-absent, then lock the row. Two first-time writers race on the
		// primary key; the loser's insert is a no-op and it proceeds to the lock.
		if _, e := tx.ExecContext(ctx, `
			INSERT INTO public.tenant_lakehouse (tenant_id, warehouse_name, bucket)
			VALUES ($1, $2, $2)
			ON CONFLICT (tenant_id) DO NOTHING`, tenantID, wh); e != nil {
			return fmt.Errorf("create lakehouse row: %w", e)
		}

		var cur sql.NullInt32
		if e := tx.QueryRowContext(ctx,
			`SELECT audit_retention_days FROM public.tenant_lakehouse WHERE tenant_id = $1 FOR UPDATE`,
			tenantID).Scan(&cur); e != nil {
			return fmt.Errorf("lock lakehouse row: %w", e)
		}

		var action string
		var before []byte
		switch {
		case !cur.Valid:
			action = "configured"
		case int(cur.Int32) == days:
			return nil // already this value
		case int(cur.Int32) > days:
			return ErrRetentionLowered
		default:
			action = "retention_extended"
			before = mustJSON(map[string]int{"audit_retention_days": int(cur.Int32)})
		}

		if _, e := tx.ExecContext(ctx, `
			UPDATE public.tenant_lakehouse
			   SET audit_retention_days = $2, version = version + 1, updated_at = now()
			 WHERE tenant_id = $1`, tenantID, days); e != nil {
			return mapDBError(e)
		}
		return insertAudit(ctx, tx, tenantID, actor, action, before, mustJSON(map[string]int{"audit_retention_days": days}))
	})
	if err != nil {
		if errors.Is(err, ErrRetentionLowered) {
			return nil, err
		}
		return nil, fmt.Errorf("set retention: %w", err)
	}
	return s.Get(ctx, tenantID)
}

// MarkProvisioned records that the tenant's bucket, key and warehouse now exist: it
// binds the Lakekeeper warehouse id and KMS key and moves the tenant to "active", with
// the audit entry in the same transaction. It is idempotent for the same warehouse id,
// and refuses a different one (ErrWarehouseMismatch): a tenant's warehouse is never
// silently replaced.
func (s *Store) MarkProvisioned(ctx context.Context, tenantID, warehouseID uuid.UUID, kmsKeyID string, appliedDays int, actor Actor) error {
	if warehouseID == uuid.Nil || kmsKeyID == "" || appliedDays < MinRetentionDays {
		return errors.New("a warehouse id, a KMS key id and the retention the bucket was created with are required")
	}
	err := dbpkg.WithTenantTransaction(ctx, s.db, tenantID.String(), func(tx *sql.Tx) error {
		var state, bucket string
		var current uuid.NullUUID
		var days sql.NullInt32
		e := tx.QueryRowContext(ctx, `
			SELECT lifecycle_state, lakekeeper_warehouse_id, audit_retention_days, bucket
			  FROM public.tenant_lakehouse WHERE tenant_id = $1 FOR UPDATE`, tenantID).Scan(&state, &current, &days, &bucket)
		if errors.Is(e, sql.ErrNoRows) || (e == nil && !days.Valid) {
			return ErrNotConfigured
		}
		if e != nil {
			return fmt.Errorf("lock lakehouse row: %w", e)
		}
		if current.Valid {
			if current.UUID == warehouseID {
				return nil // already provisioned; nothing to change or audit
			}
			return ErrWarehouseMismatch
		}
		if state != "provisioning" {
			return fmt.Errorf("%w: state is %q", ErrInvalidState, state)
		}
		if _, e := tx.ExecContext(ctx, `
			UPDATE public.tenant_lakehouse
			   SET lakekeeper_warehouse_id = $2, kms_key_id = $3, lifecycle_state = 'active',
			       retention_applied_days = $4, version = version + 1, updated_at = now()
			 WHERE tenant_id = $1`, tenantID, warehouseID, kmsKeyID, appliedDays); e != nil {
			return fmt.Errorf("mark provisioned: %w", e)
		}
		return insertAudit(ctx, tx, tenantID, actor, "provisioned", nil,
			mustJSON(map[string]any{"warehouse_id": warehouseID.String(), "bucket": bucket, "audit_retention_days": int(days.Int32), "retention_applied_days": appliedDays}))
	})
	if err != nil {
		if errors.Is(err, ErrNotConfigured) || errors.Is(err, ErrWarehouseMismatch) || errors.Is(err, ErrInvalidState) {
			return err
		}
		return fmt.Errorf("mark provisioned: %w", err)
	}
	return nil
}

// MarkRetentionApplied records that the tenant's bucket now enforces `days` by default,
// with an audit entry in the same transaction. It only ever raises the applied value, never
// exceeds the retention the registry wants, and requires a provisioned tenant. Setting a
// value the bucket already has (or less) is a no-op: reconcile can be retried freely.
func (s *Store) MarkRetentionApplied(ctx context.Context, tenantID uuid.UUID, days int, actor Actor) error {
	if days < MinRetentionDays || days > MaxRetentionDays {
		return ErrInvalidRetention
	}
	err := dbpkg.WithTenantTransaction(ctx, s.db, tenantID.String(), func(tx *sql.Tx) error {
		var want, applied sql.NullInt32
		var provisioned bool
		e := tx.QueryRowContext(ctx, `
			SELECT audit_retention_days, retention_applied_days, lakekeeper_warehouse_id IS NOT NULL
			  FROM public.tenant_lakehouse WHERE tenant_id = $1 FOR UPDATE`, tenantID).Scan(&want, &applied, &provisioned)
		if errors.Is(e, sql.ErrNoRows) {
			return ErrNotConfigured
		}
		if e != nil {
			return fmt.Errorf("lock lakehouse row: %w", e)
		}
		if !provisioned {
			return fmt.Errorf("%w: nothing is provisioned to apply retention to", ErrInvalidState)
		}
		if !want.Valid || days > int(want.Int32) {
			return fmt.Errorf("%w: cannot apply %d days, the registry wants %d", ErrInvalidRetention, days, want.Int32)
		}
		if applied.Valid && int(applied.Int32) >= days {
			return nil // the bucket already enforces this much
		}
		var before []byte
		if applied.Valid {
			before = mustJSON(map[string]int{"retention_applied_days": int(applied.Int32)})
		}
		if _, e := tx.ExecContext(ctx, `
			UPDATE public.tenant_lakehouse
			   SET retention_applied_days = $2, version = version + 1, updated_at = now()
			 WHERE tenant_id = $1`, tenantID, days); e != nil {
			return fmt.Errorf("record applied retention: %w", e)
		}
		return insertAudit(ctx, tx, tenantID, actor, "retention_applied", before,
			mustJSON(map[string]int{"retention_applied_days": days}))
	})
	if err != nil {
		if errors.Is(err, ErrNotConfigured) || errors.Is(err, ErrInvalidState) || errors.Is(err, ErrInvalidRetention) {
			return err
		}
		return fmt.Errorf("mark retention applied: %w", err)
	}
	return nil
}

// AuditAfter returns up to limit audit entries with an id greater than afterID, OLDEST FIRST,
// for copying to the tenant's Iceberg audit table (ADR-036). The order is the chain order.
func (s *Store) AuditAfter(ctx context.Context, tenantID uuid.UUID, afterID int64, limit int) ([]AuditEntry, error) {
	if limit < 1 || limit > 5000 {
		limit = 500
	}
	var out []AuditEntry
	err := dbpkg.WithTenantTransaction(ctx, s.db, tenantID.String(), func(tx *sql.Tx) error {
		rows, e := tx.QueryContext(ctx, `
			SELECT id, at, actor_id, actor_role, action, before, after, prev_hash, hash
			  FROM public.tenant_lakehouse_audit
			 WHERE tenant_id = $1 AND id > $2 ORDER BY id ASC LIMIT $3`, tenantID, afterID, limit)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			var a AuditEntry
			var before, after []byte
			if e := rows.Scan(&a.ID, &a.At, &a.ActorID, &a.ActorRole, &a.Action, &before, &after, &a.PrevHash, &a.Hash); e != nil {
				return e
			}
			a.Before, a.After = before, after
			out = append(out, a)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("read audit after %d: %w", afterID, err)
	}
	return out, nil
}

// MarkAuditCopied records that the audit entries up to throughID are in the Iceberg copy. It only
// raises the marker and is a status for operators; it never decides what is shipped.
func (s *Store) MarkAuditCopied(ctx context.Context, tenantID uuid.UUID, throughID int64) error {
	if throughID < 1 {
		return errors.New("a positive audit id is required")
	}
	var found bool
	err := dbpkg.WithTenantTransaction(ctx, s.db, tenantID.String(), func(tx *sql.Tx) error {
		if e := tx.QueryRowContext(ctx, `SELECT true FROM public.tenant_lakehouse WHERE tenant_id = $1 FOR UPDATE`, tenantID).Scan(&found); e != nil {
			if errors.Is(e, sql.ErrNoRows) {
				return ErrNotConfigured
			}
			return e
		}
		_, e := tx.ExecContext(ctx, `
			UPDATE public.tenant_lakehouse SET audit_copied_through_id = $2
			 WHERE tenant_id = $1 AND (audit_copied_through_id IS NULL OR audit_copied_through_id < $2)`, tenantID, throughID)
		return e
	})
	if err != nil {
		if errors.Is(err, ErrNotConfigured) {
			return err
		}
		return fmt.Errorf("mark audit copied: %w", err)
	}
	return nil
}

// ProvisionedTenants returns the ids of every tenant that has a provisioned lakehouse, for the
// scheduled audit copy. It pages through the tenants and reads each one's entry under its own RLS
// context, like List, so no privileged cross-tenant role is involved.
func (s *Store) ProvisionedTenants(ctx context.Context) ([]uuid.UUID, error) {
	var out []uuid.UUID
	for offset := 0; ; offset += 100 {
		items, total, err := s.List(ctx, "", 100, offset)
		if err != nil {
			return nil, err
		}
		for _, c := range items {
			if c.Provisioned {
				if id, e := uuid.Parse(c.TenantID); e == nil {
					out = append(out, id)
				}
			}
		}
		if offset+100 >= total || len(items) == 0 {
			return out, nil
		}
	}
}

// MarkCredentialIssued records that the tenant's storage credential now exists. It is
// idempotent: the first call sets the time and later calls change nothing. It is the one
// write that must follow a successful issue, because it is what stops a later "credential
// not found" from being mistaken for "never issued".
func (s *Store) MarkCredentialIssued(ctx context.Context, tenantID uuid.UUID) error {
	var rows int64
	err := dbpkg.WithTenantTransaction(ctx, s.db, tenantID.String(), func(tx *sql.Tx) error {
		res, e := tx.ExecContext(ctx, `
			UPDATE public.tenant_lakehouse SET credential_issued_at = now()
			 WHERE tenant_id = $1 AND credential_issued_at IS NULL`, tenantID)
		if e != nil {
			return e
		}
		rows, e = res.RowsAffected()
		return e
	})
	if err != nil {
		return fmt.Errorf("mark credential issued: %w", err)
	}
	if rows == 0 {
		// Either already marked (fine) or there is no row at all (not fine).
		cfg, e := s.Get(ctx, tenantID)
		if e != nil {
			return e
		}
		if !cfg.Configured {
			return ErrNotConfigured
		}
	}
	return nil
}

// RecordRetentionSyncFailure appends a "retention_sync_failed" audit entry naming the step and
// the reason, so a retention reconcile that stopped is visible in the audit trail.
func (s *Store) RecordRetentionSyncFailure(ctx context.Context, tenantID uuid.UUID, actor Actor, step, reason string) error {
	if len(reason) > 500 {
		reason = reason[:500]
	}
	return s.Record(ctx, tenantID, actor, "retention_sync_failed", nil, map[string]string{"step": step, "error": reason})
}

// RecordProvisionFailure appends a "provision_failed" audit entry naming the step and
// the reason, so an admin can see in the audit trail why a tenant is still waiting.
func (s *Store) RecordProvisionFailure(ctx context.Context, tenantID uuid.UUID, actor Actor, step, reason string) error {
	if len(reason) > 500 {
		reason = reason[:500]
	}
	return s.Record(ctx, tenantID, actor, "provision_failed", nil, map[string]string{"step": step, "error": reason})
}

// Record appends an audit entry for an action that changes no registry column
// itself, such as a provisioning request.
func (s *Store) Record(ctx context.Context, tenantID uuid.UUID, actor Actor, action string, before, after any) error {
	return dbpkg.WithTenantTransaction(ctx, s.db, tenantID.String(), func(tx *sql.Tx) error {
		return insertAudit(ctx, tx, tenantID, actor, action, mustJSON(before), mustJSON(after))
	})
}

func insertAudit(ctx context.Context, tx *sql.Tx, tenantID uuid.UUID, actor Actor, action string, before, after []byte) error {
	// prev_hash and hash are computed by the table's trigger, not supplied here.
	_, err := tx.ExecContext(ctx, `
		INSERT INTO public.tenant_lakehouse_audit (tenant_id, actor_id, actor_role, action, before, after)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		tenantID, actor.ID, actor.Role, action, nullableJSON(before), nullableJSON(after))
	if err != nil {
		return fmt.Errorf("write audit: %w", err)
	}
	return nil
}

// List returns one page of tenants with their lakehouse state. Tenants are paged
// from public.tenants and each row is then read under that tenant's own RLS
// context. That costs one small transaction per row on a page, and in exchange no
// privileged cross-tenant role is needed or widened.
func (s *Store) List(ctx context.Context, q string, limit, offset int) (items []Config, total int, err error) {
	if limit < 1 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	pattern := "%" + likeEscape(strings.ToLower(strings.TrimSpace(q))) + "%"
	if err = s.db.QueryRowContext(ctx, `
		SELECT count(*) FROM public.tenants
		 WHERE $1 = '%%' OR lower(name) LIKE $1 OR lower(COALESCE(code, '')) LIKE $1`, pattern).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count tenants: %w", err)
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id FROM public.tenants
		 WHERE $1 = '%%' OR lower(name) LIKE $1 OR lower(COALESCE(code, '')) LIKE $1
		 ORDER BY name, id LIMIT $2 OFFSET $3`, pattern, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list tenants: %w", err)
	}
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if e := rows.Scan(&id); e != nil {
			rows.Close()
			return nil, 0, e
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, 0, err
	}

	items = make([]Config, 0, len(ids))
	for _, id := range ids {
		c, e := s.Get(ctx, id)
		if e != nil {
			return nil, 0, e
		}
		items = append(items, *c)
	}
	return items, total, nil
}

// Audit returns the most recent audit entries for a tenant, newest first.
func (s *Store) Audit(ctx context.Context, tenantID uuid.UUID, limit int) ([]AuditEntry, error) {
	if limit < 1 || limit > 500 {
		limit = 100
	}
	var out []AuditEntry
	err := dbpkg.WithTenantTransaction(ctx, s.db, tenantID.String(), func(tx *sql.Tx) error {
		rows, e := tx.QueryContext(ctx, `
			SELECT id, at, actor_id, actor_role, action, before, after, prev_hash, hash
			  FROM public.tenant_lakehouse_audit
			 WHERE tenant_id = $1 ORDER BY id DESC LIMIT $2`, tenantID, limit)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			var a AuditEntry
			var before, after []byte
			if e := rows.Scan(&a.ID, &a.At, &a.ActorID, &a.ActorRole, &a.Action, &before, &after, &a.PrevHash, &a.Hash); e != nil {
				return e
			}
			a.Before, a.After = before, after
			out = append(out, a)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("read audit: %w", err)
	}
	return out, nil
}

// VerifyAudit recomputes a tenant's audit chain. It returns the id of the first
// entry that does not verify, or nil if the whole chain is intact.
func (s *Store) VerifyAudit(ctx context.Context, tenantID uuid.UUID) (*int64, error) {
	var broken sql.NullInt64
	err := dbpkg.WithTenantTransaction(ctx, s.db, tenantID.String(), func(tx *sql.Tx) error {
		return tx.QueryRowContext(ctx, `SELECT public.tenant_lakehouse_audit_verify($1)`, tenantID).Scan(&broken)
	})
	if err != nil {
		return nil, fmt.Errorf("verify audit: %w", err)
	}
	if !broken.Valid {
		return nil, nil
	}
	return &broken.Int64, nil
}

// likeEscape makes user text literal inside a LIKE pattern (default escape char \).
func likeEscape(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

func mapDBError(err error) error {
	if err != nil && strings.Contains(err.Error(), "audit retention can only be extended") {
		return ErrRetentionLowered
	}
	return err
}

func mustJSON(v any) []byte {
	if v == nil {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return b
}

func nullableJSON(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	return string(b)
}
