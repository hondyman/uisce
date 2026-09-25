package msgcat

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/jmoiron/sqlx"
)

// Actor is who is editing: the authenticated user, the tenant they are
// working in ("" when none), and what they may administer.
type Actor struct {
	UserID        string
	TenantID      string
	PlatformAdmin bool // may edit and approve the core catalog
	TenantAdmin   bool // may edit and approve TenantID's messages
}

// CanAdminister reports whether a may edit or approve in the scope.
func (a Actor) CanAdminister(scope string) bool {
	if scope == ScopeCore {
		return a.PlatformAdmin
	}
	return a.TenantID != "" && (a.TenantAdmin || a.PlatformAdmin)
}

const (
	ScopeCore   = "core"
	ScopeTenant = "tenant"

	maxText        = 1000
	maxDescription = 4000
)

// ChangeRequest is one proposed edit, from the editor or the API.
type ChangeRequest struct {
	Scope       string `json:"scope"` // core | tenant
	SetNbr      int    `json:"set_nbr"`
	MessageNbr  int    `json:"message_nbr"`
	Language    string `json:"language"`
	Action      string `json:"action"` // upsert (default) | delete
	Severity    string `json:"severity,omitempty"`
	Text        string `json:"text,omitempty"`
	Description string `json:"description,omitempty"`
	UserAction  string `json:"user_action,omitempty"`
	Reason      string `json:"reason,omitempty"`
}

// ProposeResult says what happened to each request, in order.
type ProposeResult struct {
	Changes []Change `json:"changes"`
	// Applied is false when the changes wait for a second approver.
	Applied bool `json:"applied"`
}

// Editor proposes and decides catalog changes.
type Editor struct {
	store   *Store
	catalog *Catalog
}

func NewEditor(store *Store, catalog *Catalog) *Editor {
	return &Editor{store: store, catalog: catalog}
}

func msg(nbr int, params ...any) *Error { return New(SetMsgcat, nbr, params...) }

// scopeTenant is the tenant_id for a scope ("" = core).
func (a Actor) scopeTenant(scope string) string {
	if scope == ScopeCore {
		return ""
	}
	return a.TenantID
}

// Propose validates a batch of requests and records them, all or none. If
// the scope does not require a second approver they are applied at once.
// A batch is one scope: a core and a tenant edit are separate proposals.
func (ed *Editor) Propose(ctx context.Context, a Actor, reqs []ChangeRequest) (*ProposeResult, error) {
	if len(reqs) == 0 {
		return nil, InvalidInput("no changes")
	}
	if len(reqs) > 200 {
		return nil, InvalidInput("at most 200 changes per request")
	}
	for i := range reqs {
		if reqs[i].Scope == "" {
			reqs[i].Scope = ScopeTenant
		}
		if reqs[i].Action == "" {
			reqs[i].Action = "upsert"
		}
	}
	scope := reqs[0].Scope
	if scope != ScopeCore && scope != ScopeTenant {
		return nil, InvalidInput("scope must be core or tenant")
	}
	for _, r := range reqs {
		if r.Scope != scope {
			return nil, InvalidInput("all changes in one request must have the same scope")
		}
	}
	if !a.CanAdminister(scope) {
		if scope == ScopeCore {
			return nil, msg(11).WithStatus(http.StatusForbidden)
		}
		return nil, msg(12).WithStatus(http.StatusForbidden)
	}
	tenantID := a.scopeTenant(scope)
	required := true
	if scope == ScopeTenant {
		var err error
		if required, err = ed.store.MakerCheckerRequired(ctx, tenantID); err != nil {
			return nil, err
		}
	}

	res := &ProposeResult{Applied: !required}
	err := ed.store.inTenant(ctx, tenantID, func(tx *sqlx.Tx) error {
		// overlay: the batch's own edits, so a batch can add English and
		// its translations together.
		overlay := map[string]*Entry{}
		for _, r := range reqs {
			if err := ed.validate(ctx, tx, tenantID, r, overlay); err != nil {
				return err
			}
			cur, err := row(ctx, tx, tenantID, r.SetNbr, r.MessageNbr, r.Language, false)
			if err != nil {
				return err
			}
			var id string
			err = tx.GetContext(ctx, &id, `INSERT INTO public.message_catalog_changes
				(tenant_id, set_nbr, message_nbr, language_cd, action, severity, message_text, description, user_action, before, requested_by, reason)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) RETURNING id::text`,
				nullable(tenantID), r.SetNbr, r.MessageNbr, r.Language, r.Action, nullable(r.Severity), nullable(r.Text),
				nullable(r.Description), nullable(r.UserAction), nullableJSON(imageOf(cur)), a.UserID, nullable(r.Reason))
			if err != nil {
				return err
			}
			if !required {
				if err := ed.apply(ctx, tx, id, a.UserID, ""); err != nil {
					return err
				}
			}
			c, err := ed.store.change(ctx, tx, id)
			if err != nil {
				return err
			}
			res.Changes = append(res.Changes, *c)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if !required {
		ed.catalog.Invalidate(tenantID)
	}
	return res, nil
}

func overlayKey(set, nbr int, lang string) string {
	return strings.Join([]string{itoa(set), itoa(nbr), lang}, "/")
}

// current is a row as the batch so far would leave it (nil: absent).
func current(ctx context.Context, tx *sqlx.Tx, tenantID string, set, nbr int, lang string, overlay map[string]*Entry) (*Entry, error) {
	if e, ok := overlay[overlayKey(set, nbr, lang)]; ok {
		return e, nil
	}
	return row(ctx, tx, tenantID, set, nbr, lang, false)
}

func (ed *Editor) validate(ctx context.Context, tx *sqlx.Tx, tenantID string, r ChangeRequest, overlay map[string]*Entry) error {
	lang, ok := NormalizeLanguage(r.Language)
	if !ok || lang != r.Language {
		return msg(1, r.Language)
	}
	if r.MessageNbr < 1 || r.MessageNbr > 99999 {
		return msg(9)
	}
	set, err := ed.store.Set(ctx, r.SetNbr)
	if err != nil {
		return err
	}
	if set == nil {
		return msg(7, r.SetNbr).WithStatus(http.StatusNotFound)
	}
	var pending int
	if err := tx.GetContext(ctx, &pending, `SELECT count(*) FROM public.message_catalog_changes
		WHERE tenant_id IS NOT DISTINCT FROM $1 AND set_nbr = $2 AND message_nbr = $3 AND language_cd = $4 AND status = 'pending'`,
		nullable(tenantID), r.SetNbr, r.MessageNbr, r.Language); err != nil {
		return err
	}
	if pending > 0 {
		return msg(20).WithStatus(http.StatusConflict)
	}
	coreEN, err := row(ctx, tx, "", r.SetNbr, r.MessageNbr, BaseLanguage, false)
	if err != nil {
		return err
	}
	// The English text translations are checked against: the scope's own,
	// else (tenant) the core's.
	scopeEN, err := current(ctx, tx, tenantID, r.SetNbr, r.MessageNbr, BaseLanguage, overlay)
	if err != nil {
		return err
	}
	baseEN := scopeEN
	if baseEN == nil && tenantID != "" {
		baseEN = coreEN
	}
	existing, err := current(ctx, tx, tenantID, r.SetNbr, r.MessageNbr, r.Language, overlay)
	if err != nil {
		return err
	}
	k := overlayKey(r.SetNbr, r.MessageNbr, r.Language)

	switch r.Action {
	case "delete":
		if existing == nil {
			return msg(19).WithStatus(http.StatusNotFound)
		}
		if lang == BaseLanguage {
			if tenantID == "" {
				return msg(13)
			}
			// A tenant's own message (no core row) keeps English while it
			// has translations: English is their fallback.
			if coreEN == nil {
				var langs []string
				if err := tx.SelectContext(ctx, &langs, `SELECT language_cd FROM public.tenant_message_catalog
					WHERE tenant_id = $1 AND set_nbr = $2 AND message_nbr = $3 AND language_cd <> 'en'`,
					tenantID, r.SetNbr, r.MessageNbr); err != nil {
					return err
				}
				remaining := map[string]bool{}
				for _, l := range langs {
					remaining[l] = true
				}
				for _, l := range Languages {
					if e, ok := overlay[overlayKey(r.SetNbr, r.MessageNbr, l.Code)]; ok && l.Code != BaseLanguage {
						remaining[l.Code] = e != nil
					}
				}
				for _, left := range remaining {
					if left {
						return msg(21)
					}
				}
			}
		}
		overlay[k] = nil
		return nil
	case "upsert":
	default:
		return InvalidInput("action must be upsert or delete")
	}

	if strings.TrimSpace(r.Text) == "" {
		return msg(3)
	}
	if utf8.RuneCountInString(r.Text) > maxText || utf8.RuneCountInString(r.UserAction) > maxText {
		return msg(4, maxText)
	}
	if utf8.RuneCountInString(r.Description) > maxDescription {
		return msg(4, maxDescription)
	}
	if tenantID != "" && coreEN == nil && set.Type != "client" {
		return msg(10, set.SetNbr, set.Type)
	}
	if lang == BaseLanguage {
		if !ValidSeverity(r.Severity) {
			return msg(2, r.Severity)
		}
	} else {
		if baseEN == nil {
			return msg(6)
		}
		if r.Severity != "" && r.Severity != baseEN.Severity {
			return InvalidInput("severity is set on the English text and applies to every language")
		}
		if !SamePlaceholders(r.Text, baseEN.Text) {
			return msg(5, placeholderList(baseEN.Text))
		}
		if r.UserAction != "" && !subsetPlaceholders(r.UserAction, baseEN.Text) {
			return msg(5, placeholderList(baseEN.Text))
		}
	}
	overlay[k] = &Entry{SetNbr: r.SetNbr, MessageNbr: r.MessageNbr, Language: lang, Severity: r.Severity, Text: r.Text}
	return nil
}

func placeholderList(text string) string {
	p := Placeholders(text)
	if len(p) == 0 {
		return "—"
	}
	return strings.Join(p, ", ")
}

func subsetPlaceholders(text, of string) bool {
	allowed := map[string]bool{}
	for _, p := range Placeholders(of) {
		allowed[p] = true
	}
	for _, p := range Placeholders(text) {
		if !allowed[p] {
			return false
		}
	}
	return true
}

// apply writes a pending change to the catalog, in tx. The row must still
// be what the proposer saw.
func (ed *Editor) apply(ctx context.Context, tx *sqlx.Tx, id, reviewer, comment string) error {
	c, err := ed.store.change(ctx, tx, id)
	if err != nil {
		return err
	}
	if c == nil {
		return msg(14, id).WithStatus(http.StatusNotFound)
	}
	tenantID := c.TenantID.String
	cur, err := row(ctx, tx, tenantID, c.SetNbr, c.MessageNbr, c.Language, true)
	if err != nil {
		return err
	}
	if !sameImage(imageOf(cur), c.Before) {
		return msg(17).WithStatus(http.StatusConflict)
	}
	table, tcol, targ := "public.message_catalog", "", []any{}
	if tenantID != "" {
		table, tcol, targ = "public.tenant_message_catalog", "tenant_id, ", []any{tenantID}
	}
	if c.Action == "delete" {
		q := `DELETE FROM ` + table + ` WHERE set_nbr = $1 AND message_nbr = $2 AND language_cd = $3`
		args := []any{c.SetNbr, c.MessageNbr, c.Language}
		if tenantID != "" {
			q += ` AND tenant_id = $4`
			args = append(args, tenantID)
		}
		if _, err := tx.ExecContext(ctx, q, args...); err != nil {
			return err
		}
	} else {
		severity := c.Severity.String
		if c.Language != BaseLanguage {
			// Severity belongs to the message: take the scope's English, or
			// the core's for a tenant translation of a core message.
			en, err := row(ctx, tx, tenantID, c.SetNbr, c.MessageNbr, BaseLanguage, false)
			if err != nil {
				return err
			}
			if en == nil && tenantID != "" {
				en, err = row(ctx, tx, "", c.SetNbr, c.MessageNbr, BaseLanguage, false)
				if err != nil {
					return err
				}
			}
			if en == nil {
				return msg(6)
			}
			severity = en.Severity
		}
		ph := "$1, $2, $3, $4, $5, $6, $7, $8"
		if tenantID != "" {
			ph = "$1, $2, $3, $4, $5, $6, $7, $8, $9"
		}
		conflict := "set_nbr, message_nbr, language_cd"
		if tenantID != "" {
			conflict = "tenant_id, " + conflict
		}
		args := append(targ, c.SetNbr, c.MessageNbr, c.Language, severity, c.Text.String,
			c.Description, c.UserAction, c.RequestedBy)
		_, err := tx.ExecContext(ctx, `INSERT INTO `+table+` (`+tcol+`set_nbr, message_nbr, language_cd, severity, message_text, description, user_action, updated_by)
			VALUES (`+ph+`)
			ON CONFLICT (`+conflict+`) DO UPDATE SET severity = EXCLUDED.severity, message_text = EXCLUDED.message_text,
				description = EXCLUDED.description, user_action = EXCLUDED.user_action,
				updated_by = EXCLUDED.updated_by, updated_at = now()`, args...)
		if err != nil {
			return err
		}
		if c.Language == BaseLanguage {
			// One severity per message, in every language of the scope.
			q := `UPDATE ` + table + ` SET severity = $1 WHERE set_nbr = $2 AND message_nbr = $3 AND severity <> $1`
			a := []any{severity, c.SetNbr, c.MessageNbr}
			if tenantID != "" {
				q += ` AND tenant_id = $4`
				a = append(a, tenantID)
			}
			if _, err := tx.ExecContext(ctx, q, a...); err != nil {
				return err
			}
		}
	}
	_, err = tx.ExecContext(ctx, `UPDATE public.message_catalog_changes
		SET status = 'applied', reviewed_by = $2, reviewed_at = CASE WHEN $2::text IS NULL THEN NULL ELSE now() END,
			review_comment = $3, applied_at = now()
		WHERE id::text = $1`, id, nullable(reviewer), nullable(comment))
	return err
}

// Decide approves (applies) or rejects a pending change. The reviewer must
// administer the change's scope and must not be the person who proposed it.
func (ed *Editor) Decide(ctx context.Context, a Actor, id string, approve bool, comment string) (*Change, error) {
	var out *Change
	var tenantID string
	err := ed.withChange(ctx, a, id, func(tx *sqlx.Tx, c *Change) error {
		tenantID = c.TenantID.String
		if c.RequestedBy == a.UserID {
			return msg(16).WithStatus(http.StatusForbidden)
		}
		if approve {
			if err := ed.apply(ctx, tx, id, a.UserID, comment); err != nil {
				return err
			}
		} else if _, err := tx.ExecContext(ctx, `UPDATE public.message_catalog_changes
			SET status = 'rejected', reviewed_by = $2, reviewed_at = now(), review_comment = $3 WHERE id::text = $1`,
			id, a.UserID, nullable(comment)); err != nil {
			return err
		}
		var err error
		out, err = ed.store.change(ctx, tx, id)
		return err
	})
	if err != nil {
		return nil, err
	}
	if approve {
		ed.catalog.Invalidate(tenantID)
	}
	return out, nil
}

// Withdraw lets the proposer take back a pending change.
func (ed *Editor) Withdraw(ctx context.Context, a Actor, id string) (*Change, error) {
	var out *Change
	err := ed.withChange(ctx, a, id, func(tx *sqlx.Tx, c *Change) error {
		if c.RequestedBy != a.UserID {
			return msg(18).WithStatus(http.StatusForbidden)
		}
		if _, err := tx.ExecContext(ctx, `UPDATE public.message_catalog_changes SET status = 'withdrawn' WHERE id::text = $1`, id); err != nil {
			return err
		}
		var err error
		out, err = ed.store.change(ctx, tx, id)
		return err
	})
	return out, err
}

// withChange loads a pending change the actor administers, locked, and runs
// fn in the change's tenant scope. A change the actor may not see is
// reported as not found.
func (ed *Editor) withChange(ctx context.Context, a Actor, id string, fn func(*sqlx.Tx, *Change) error) error {
	// Core changes first need no tenant context; a tenant change is only
	// visible in the actor's own tenant.
	for _, tenantID := range []string{"", a.TenantID} {
		if tenantID == "" && !a.PlatformAdmin {
			continue
		}
		var found bool
		err := ed.store.inTenant(ctx, tenantID, func(tx *sqlx.Tx) error {
			c, err := ed.store.change(ctx, tx, id)
			if err != nil || c == nil || c.TenantID.String != tenantID {
				return err
			}
			found = true
			scope := ScopeTenant
			if !c.TenantID.Valid {
				scope = ScopeCore
			}
			if !a.CanAdminister(scope) {
				return msg(14, id).WithStatus(http.StatusNotFound)
			}
			if c.Status != "pending" {
				return msg(15).WithStatus(http.StatusConflict)
			}
			return fn(tx, c)
		})
		if err != nil || found {
			return err
		}
		if tenantID == "" && a.TenantID == "" {
			break
		}
	}
	return msg(14, id).WithStatus(http.StatusNotFound)
}

func nullableJSON(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	return string(b)
}

func itoa(n int) string { return strconv.Itoa(n) }
