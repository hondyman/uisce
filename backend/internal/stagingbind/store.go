// Package stagingbind binds staging tables to the business objects they
// feed: business object field -> staging column. A pipeline rule check in
// front of a staging load reads rows through the binding, so the object's
// rules (which read its field names) check the staging columns bound to
// them.
//
// Bindings are separate from business_object_binding on purpose: those are
// where an object's records are read and written; a staging binding is only
// a mapping for checks and lineage. Every change goes through maker-checker
// (see Editor). The gold copy's bindings are inherited read-only and a
// tenant's own binding for the same object and table takes precedence.
package stagingbind

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/hondyman/uisce/backend/internal/tenant"
)

var stagingTable = regexp.MustCompile(`^staging\.[a-z_][a-z0-9_]*$`)

// Binding is a staging table's binding to a business object.
type Binding struct {
	ID           string            `db:"id" json:"id"`
	TenantID     string            `db:"tenant_id" json:"tenant_id"`
	BOKey        string            `db:"bo_key" json:"bo_key"`
	BOName       string            `db:"bo_name" json:"bo_name"`
	StagingTable string            `db:"staging_table" json:"staging_table"`
	Fields       map[string]string `db:"-" json:"fields"` // BO field -> staging column
	RawFields    []byte            `db:"fields" json:"-"`
	Version      int               `db:"version" json:"version"`
	// Origin is "core" (inherited from the gold copy) or "tenant".
	Origin    string    `db:"origin" json:"origin"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// Change is one proposed binding change and, once decided, its audit record.
type Change struct {
	ID              string            `db:"id" json:"id"`
	TenantID        string            `db:"tenant_id" json:"tenant_id"`
	BOID            string            `db:"bo_id" json:"-"`
	BOKey           string            `db:"bo_key" json:"bo_key"`
	StagingTable    string            `db:"staging_table" json:"staging_table"`
	Action          string            `db:"action" json:"action"`
	Fields          map[string]string `db:"-" json:"fields,omitempty"`
	RawFields       []byte            `db:"fields" json:"-"`
	Before          map[string]string `db:"-" json:"before,omitempty"`
	RawBefore       []byte            `db:"before" json:"-"`
	Status          string            `db:"status" json:"status"`
	Reason          sql.NullString    `db:"reason" json:"-"`
	RequestedBy     string            `db:"requested_by" json:"requested_by"`
	RequestedByName sql.NullString    `db:"requested_by_name" json:"-"`
	RequestedAt     time.Time         `db:"requested_at" json:"requested_at"`
	ReviewedBy      sql.NullString    `db:"reviewed_by" json:"-"`
	ReviewedByName  sql.NullString    `db:"reviewed_by_name" json:"-"`
	ReviewedAt      sql.NullTime      `db:"reviewed_at" json:"-"`
	ReviewComment   sql.NullString    `db:"review_comment" json:"-"`
}

// MarshalJSON flattens the nullable audit columns.
func (c Change) MarshalJSON() ([]byte, error) {
	type plain Change
	out := struct {
		plain
		Reason          string     `json:"reason,omitempty"`
		RequestedByName string     `json:"requested_by_name,omitempty"`
		ReviewedBy      string     `json:"reviewed_by,omitempty"`
		ReviewedByName  string     `json:"reviewed_by_name,omitempty"`
		ReviewedAt      *time.Time `json:"reviewed_at,omitempty"`
		ReviewComment   string     `json:"review_comment,omitempty"`
	}{plain: plain(c), Reason: c.Reason.String, RequestedByName: c.RequestedByName.String, ReviewedBy: c.ReviewedBy.String,
		ReviewedByName: c.ReviewedByName.String, ReviewComment: c.ReviewComment.String}
	if c.ReviewedAt.Valid {
		out.ReviewedAt = &c.ReviewedAt.Time
	}
	return json.Marshal(out)
}

// Store reads and writes bindings in the metadata DB. Every statement runs in
// a transaction carrying the tenant's RLS context and also filters on it.
type Store struct{ DB *sqlx.DB }

func (s *Store) inTenant(ctx context.Context, tenantID string, fn func(*sqlx.Tx) error) error {
	if tenantID == "" {
		return msgNoTenant()
	}
	tx, err := s.DB.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	if err := tenant.SetRLSContext(ctx, tx, tenantID); err != nil {
		return fmt.Errorf("setting tenant context: %w", err)
	}
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

// visible restricts rows to the tenant's own and the gold copy's.
const visible = `(sb.tenant_id::text = $1 OR sb.tenant_id = public.uisce_gold_copy_tenant_id())`

const bindingCols = `sb.id::text, sb.tenant_id::text, bo.bo_key, COALESCE(bo.bo_name, bo.bo_key) AS bo_name, sb.staging_table,
	sb.fields, sb.version, sb.updated_at,
	CASE WHEN sb.tenant_id::text = $1 THEN 'tenant' ELSE 'core' END AS origin`

// List is every binding the tenant sees, its own first where it overrides
// the gold copy's.
func (s *Store) List(ctx context.Context, tenantID string) ([]Binding, error) {
	var rows []Binding
	err := s.inTenant(ctx, tenantID, func(tx *sqlx.Tx) error {
		return tx.SelectContext(ctx, &rows, `SELECT DISTINCT ON (bo.bo_key, sb.staging_table) `+bindingCols+`
			FROM public.staging_bindings sb JOIN public.business_objects bo ON bo.id = sb.bo_id
			WHERE `+visible+`
			ORDER BY bo.bo_key, sb.staging_table, (sb.tenant_id::text = $1) DESC`, tenantID)
	})
	if err != nil {
		return nil, err
	}
	out := make([]Binding, 0, len(rows))
	for _, b := range rows {
		if err := b.decode(); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, nil
}

// Resolve is the binding a tenant uses for boKey on table: its own, else the
// gold copy's. nil (no error) when there is none.
func (s *Store) Resolve(ctx context.Context, tenantID, boKey, table string) (*Binding, error) {
	var out *Binding
	err := s.inTenant(ctx, tenantID, func(tx *sqlx.Tx) error {
		var err error
		out, err = resolve(ctx, tx, tenantID, boKey, table)
		return err
	})
	return out, err
}

// StagingFields is Resolve's field -> column map (nil when there is no
// binding) - what a pipeline rule check reads rows through.
func (s *Store) StagingFields(ctx context.Context, tenantID, boKey, table string) (map[string]string, error) {
	b, err := s.Resolve(ctx, tenantID, boKey, table)
	if err != nil || b == nil {
		return nil, err
	}
	return b.Fields, nil
}

func resolve(ctx context.Context, tx *sqlx.Tx, tenantID, boKey, table string) (*Binding, error) {
	var b Binding
	err := tx.GetContext(ctx, &b, `SELECT `+bindingCols+`
		FROM public.staging_bindings sb JOIN public.business_objects bo ON bo.id = sb.bo_id
		WHERE `+visible+` AND bo.bo_key = $2 AND sb.staging_table = $3
		ORDER BY (sb.tenant_id::text = $1) DESC LIMIT 1`, tenantID, boKey, table)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &b, b.decode()
}

func (b *Binding) decode() error {
	b.Fields = map[string]string{}
	if len(b.RawFields) == 0 {
		return nil
	}
	return json.Unmarshal(b.RawFields, &b.Fields)
}

func (c *Change) decode() error {
	for _, p := range []struct {
		raw []byte
		dst *map[string]string
	}{{c.RawFields, &c.Fields}, {c.RawBefore, &c.Before}} {
		if len(p.raw) == 0 || string(p.raw) == "null" {
			continue
		}
		if err := json.Unmarshal(p.raw, p.dst); err != nil {
			return err
		}
	}
	return nil
}

const changeCols = `c.id::text, c.tenant_id::text, c.bo_id::text, bo.bo_key, c.staging_table, c.action, c.fields, c.before,
	c.status, c.reason, c.requested_by, c.requested_by_name, c.requested_at, c.reviewed_by, c.reviewed_by_name,
	c.reviewed_at, c.review_comment`

// Changes is the tenant's change log, newest first; status "" is all.
func (s *Store) Changes(ctx context.Context, tenantID, status string, limit int) ([]Change, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	var rows []Change
	err := s.inTenant(ctx, tenantID, func(tx *sqlx.Tx) error {
		return tx.SelectContext(ctx, &rows, `SELECT `+changeCols+`
			FROM public.staging_binding_changes c JOIN public.business_objects bo ON bo.id = c.bo_id
			WHERE c.tenant_id::text = $1 AND ($2 = '' OR c.status = $2)
			ORDER BY c.requested_at DESC LIMIT $3`, tenantID, status, limit)
	})
	if err != nil {
		return nil, err
	}
	for i := range rows {
		if err := rows[i].decode(); err != nil {
			return nil, err
		}
	}
	if rows == nil {
		rows = []Change{}
	}
	return rows, nil
}

// change loads one of the tenant's changes, locked for a decision.
func change(ctx context.Context, tx *sqlx.Tx, tenantID, id string) (*Change, error) {
	var c Change
	err := tx.GetContext(ctx, &c, `SELECT `+changeCols+`
		FROM public.staging_binding_changes c JOIN public.business_objects bo ON bo.id = c.bo_id
		WHERE c.tenant_id::text = $1 AND c.id::text = $2 FOR UPDATE OF c`, tenantID, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, msgChangeNotFound(id)
	}
	if err != nil {
		return nil, err
	}
	return &c, c.decode()
}
