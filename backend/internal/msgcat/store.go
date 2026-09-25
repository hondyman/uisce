package msgcat

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/hondyman/uisce/backend/internal/tenant"
)

// Set is a message set (a module's range of messages).
type Set struct {
	SetNbr      int    `db:"set_nbr" json:"set_nbr"`
	Name        string `db:"set_name" json:"name"`
	Module      string `db:"module" json:"module"`
	Type        string `db:"set_type" json:"type"` // core | custom | client
	Description string `db:"description" json:"description"`
}

// Entry is one message in one language, from the core catalog or a
// tenant's overrides.
type Entry struct {
	SetNbr      int       `db:"set_nbr" json:"set_nbr"`
	MessageNbr  int       `db:"message_nbr" json:"message_nbr"`
	Language    string    `db:"language_cd" json:"language"`
	Severity    string    `db:"severity" json:"severity"`
	Text        string    `db:"message_text" json:"text"`
	Description string    `db:"description" json:"description,omitempty"`
	UserAction  string    `db:"user_action" json:"user_action,omitempty"`
	UpdatedBy   string    `db:"updated_by" json:"updated_by,omitempty"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

// Change is a proposed or decided edit (the maker-checker log).
type Change struct {
	ID            string         `db:"id" json:"id"`
	TenantID      sql.NullString `db:"tenant_id" json:"-"`
	Scope         string         `db:"-" json:"scope"` // core | tenant
	SetNbr        int            `db:"set_nbr" json:"set_nbr"`
	MessageNbr    int            `db:"message_nbr" json:"message_nbr"`
	Language      string         `db:"language_cd" json:"language"`
	Action        string         `db:"action" json:"action"`
	Severity      sql.NullString `db:"severity" json:"-"`
	Text          sql.NullString `db:"message_text" json:"-"`
	Description   sql.NullString `db:"description" json:"-"`
	UserAction    sql.NullString `db:"user_action" json:"-"`
	Before        RawJSON        `db:"before" json:"before,omitempty"`
	Status        string         `db:"status" json:"status"`
	RequestedBy   string         `db:"requested_by" json:"requested_by"`
	RequestedAt   time.Time      `db:"requested_at" json:"requested_at"`
	Reason        sql.NullString `db:"reason" json:"-"`
	ReviewedBy    sql.NullString `db:"reviewed_by" json:"-"`
	ReviewedAt    sql.NullTime   `db:"reviewed_at" json:"-"`
	ReviewComment sql.NullString `db:"review_comment" json:"-"`
	AppliedAt     sql.NullTime   `db:"applied_at" json:"-"`
}

// MarshalJSON flattens the nullable columns.
func (c Change) MarshalJSON() ([]byte, error) {
	type alias Change
	scope := "core"
	if c.TenantID.Valid {
		scope = "tenant"
	}
	var reviewedAt, appliedAt *time.Time
	if c.ReviewedAt.Valid {
		reviewedAt = &c.ReviewedAt.Time
	}
	if c.AppliedAt.Valid {
		appliedAt = &c.AppliedAt.Time
	}
	a := alias(c)
	a.Scope = scope
	return json.Marshal(struct {
		alias
		Severity      string     `json:"severity,omitempty"`
		Text          string     `json:"text,omitempty"`
		Description   string     `json:"description,omitempty"`
		UserAction    string     `json:"user_action,omitempty"`
		Reason        string     `json:"reason,omitempty"`
		ReviewedBy    string     `json:"reviewed_by,omitempty"`
		ReviewedAt    *time.Time `json:"reviewed_at,omitempty"`
		ReviewComment string     `json:"review_comment,omitempty"`
		AppliedAt     *time.Time `json:"applied_at,omitempty"`
	}{a, c.Severity.String, c.Text.String, c.Description.String, c.UserAction.String,
		c.Reason.String, c.ReviewedBy.String, reviewedAt, c.ReviewComment.String, appliedAt})
}

// Store is the catalog's database access. Tenant rows are read and written
// in a transaction carrying the tenant's RLS context, and every query also
// filters on tenant_id explicitly.
type Store struct{ db *sqlx.DB }

func NewStore(db *sqlx.DB) *Store { return &Store{db: db} }

const entryCols = `set_nbr, message_nbr, language_cd, severity, message_text,
	COALESCE(description, '') AS description, COALESCE(user_action, '') AS user_action,
	COALESCE(updated_by, '') AS updated_by, updated_at`

// inTenant runs fn in a transaction scoped to tenantID ("" = core only).
func (s *Store) inTenant(ctx context.Context, tenantID string, fn func(*sqlx.Tx) error) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	if tenantID != "" {
		if err := tenant.SetRLSContext(ctx, tx, tenantID); err != nil {
			return fmt.Errorf("setting tenant context: %w", err)
		}
	}
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) Sets(ctx context.Context) ([]Set, error) {
	var out []Set
	err := s.db.SelectContext(ctx, &out, `SELECT set_nbr, set_name, module, set_type, COALESCE(description, '') AS description
		FROM public.message_sets ORDER BY set_nbr`)
	return out, err
}

func (s *Store) Set(ctx context.Context, setNbr int) (*Set, error) {
	var out Set
	err := s.db.GetContext(ctx, &out, `SELECT set_nbr, set_name, module, set_type, COALESCE(description, '') AS description
		FROM public.message_sets WHERE set_nbr = $1`, setNbr)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &out, err
}

// CoreEntries returns the whole core catalog (it is small: hundreds of rows).
func (s *Store) CoreEntries(ctx context.Context) ([]Entry, error) {
	var out []Entry
	err := s.db.SelectContext(ctx, &out, `SELECT `+entryCols+` FROM public.message_catalog ORDER BY set_nbr, message_nbr, language_cd`)
	return out, err
}

// TenantEntries returns all of one tenant's overrides and own messages.
func (s *Store) TenantEntries(ctx context.Context, tenantID string) ([]Entry, error) {
	var out []Entry
	err := s.inTenant(ctx, tenantID, func(tx *sqlx.Tx) error {
		return tx.SelectContext(ctx, &out, `SELECT `+entryCols+` FROM public.tenant_message_catalog
			WHERE tenant_id = $1 ORDER BY set_nbr, message_nbr, language_cd`, tenantID)
	})
	return out, err
}

// MakerCheckerRequired reads the tenant's setting; it is on unless the
// tenant's settings say {"msgcat": {"maker_checker": false}}.
func (s *Store) MakerCheckerRequired(ctx context.Context, tenantID string) (bool, error) {
	var off bool
	err := s.db.GetContext(ctx, &off, `SELECT COALESCE((settings -> 'msgcat' ->> 'maker_checker') = 'false', false)
		FROM public.tenants WHERE id::text = $1`, tenantID)
	if errors.Is(err, sql.ErrNoRows) {
		return true, nil
	}
	return !off, err
}

const changeCols = `id, tenant_id, set_nbr, message_nbr, language_cd, action, severity, message_text,
	description, user_action, before, status, requested_by, requested_at, reason, reviewed_by,
	reviewed_at, review_comment, applied_at`

// ChangeFilter selects changes visible to a caller.
type ChangeFilter struct {
	TenantID    string // "" = no tenant changes
	IncludeCore bool
	Status      string // "" = any
	SetNbr      int
	MessageNbr  int
	Limit       int
}

func (s *Store) Changes(ctx context.Context, f ChangeFilter) ([]Change, error) {
	var out []Change
	err := s.inTenant(ctx, f.TenantID, func(tx *sqlx.Tx) error {
		q := `SELECT ` + changeCols + ` FROM public.message_catalog_changes
			WHERE ((tenant_id IS NULL AND $1) OR (tenant_id = $2 AND $2 <> ''))
			  AND ($3 = '' OR status = $3)
			  AND ($4 = 0 OR set_nbr = $4)
			  AND ($5 = 0 OR message_nbr = $5)
			ORDER BY requested_at DESC LIMIT $6`
		limit := f.Limit
		if limit <= 0 || limit > 500 {
			limit = 200
		}
		return tx.SelectContext(ctx, &out, q, f.IncludeCore, f.TenantID, f.Status, f.SetNbr, f.MessageNbr, limit)
	})
	return out, err
}

func (s *Store) change(ctx context.Context, tx *sqlx.Tx, id string) (*Change, error) {
	var c Change
	err := tx.GetContext(ctx, &c, `SELECT `+changeCols+` FROM public.message_catalog_changes WHERE id::text = $1 FOR UPDATE`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &c, err
}

// row reads the current catalog row in a scope (tenantID "" = core).
func row(ctx context.Context, tx *sqlx.Tx, tenantID string, set, nbr int, lang string, lock bool) (*Entry, error) {
	var e Entry
	var err error
	suffix := ""
	if lock {
		suffix = " FOR UPDATE"
	}
	if tenantID == "" {
		err = tx.GetContext(ctx, &e, `SELECT `+entryCols+` FROM public.message_catalog
			WHERE set_nbr = $1 AND message_nbr = $2 AND language_cd = $3`+suffix, set, nbr, lang)
	} else {
		err = tx.GetContext(ctx, &e, `SELECT `+entryCols+` FROM public.tenant_message_catalog
			WHERE tenant_id = $1 AND set_nbr = $2 AND message_nbr = $3 AND language_cd = $4`+suffix, tenantID, set, nbr, lang)
	}
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &e, err
}

// beforeImage is the part of a row a change's approval re-checks.
type beforeImage struct {
	Severity    string `json:"severity"`
	Text        string `json:"text"`
	Description string `json:"description,omitempty"`
	UserAction  string `json:"user_action,omitempty"`
}

func imageOf(e *Entry) RawJSON {
	if e == nil {
		return nil
	}
	b, _ := json.Marshal(beforeImage{e.Severity, e.Text, e.Description, e.UserAction})
	return b
}

func sameImage(a, b RawJSON) bool {
	if len(a) == 0 || len(b) == 0 {
		return len(a) == 0 && len(b) == 0
	}
	var x, y beforeImage
	if json.Unmarshal(a, &x) != nil || json.Unmarshal(b, &y) != nil {
		return false
	}
	return x == y
}

// RawJSON is a nullable jsonb column passed through as JSON.
type RawJSON json.RawMessage

func (j *RawJSON) Scan(v any) error {
	switch b := v.(type) {
	case nil:
		*j = nil
	case []byte:
		*j = append(RawJSON(nil), b...)
	case string:
		*j = RawJSON(b)
	default:
		return fmt.Errorf("RawJSON: unsupported type %T", v)
	}
	return nil
}

func (j RawJSON) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	return j, nil
}

func nullable(s string) sql.NullString { return sql.NullString{String: s, Valid: s != ""} }
