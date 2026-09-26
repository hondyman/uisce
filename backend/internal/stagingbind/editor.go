package stagingbind

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"reflect"
	"sort"
	"strings"

	"github.com/jmoiron/sqlx"
)

// Actor is who is proposing or deciding: the authenticated user, the tenant
// they are working in, and what they may administer.
type Actor struct {
	UserID        string
	Name          string // readable identity (email) for the change log
	TenantID      string
	PlatformAdmin bool
	TenantAdmin   bool
}

// canAdminister: the gold copy's bindings are core - platform administrators
// only; a tenant's own need a tenant (or platform) administrator.
func (a Actor) canAdminister(goldCopy bool) bool {
	if goldCopy {
		return a.PlatformAdmin
	}
	return a.TenantAdmin || a.PlatformAdmin
}

// Columns lists a staging table's columns (the staging DB).
type Columns interface {
	Columns(ctx context.Context, table string) ([]string, error)
}

// Proposal is one requested change, from the editor, the API or MCP.
type Proposal struct {
	BOKey        string            `json:"bo_key"`
	StagingTable string            `json:"staging_table"`
	Action       string            `json:"action,omitempty"` // upsert (default) | delete
	Fields       map[string]string `json:"fields,omitempty"` // BO field -> staging column
	Reason       string            `json:"reason,omitempty"`
}

// Editor proposes and decides binding changes. Nothing is applied without a
// second person's approval.
type Editor struct {
	Store   *Store
	Columns Columns // nil: proposals are refused (the columns can't be checked)
}

func isGoldCopy(ctx context.Context, tx *sqlx.Tx, tenantID string) (bool, error) {
	var gold bool
	err := tx.GetContext(ctx, &gold, `SELECT COALESCE($1::uuid = public.uisce_gold_copy_tenant_id(), false)`, tenantID)
	return gold, err
}

// Propose records a change for approval.
func (ed *Editor) Propose(ctx context.Context, a Actor, p Proposal) (*Change, error) {
	p.StagingTable = strings.TrimSpace(p.StagingTable)
	if !stagingTable.MatchString(p.StagingTable) {
		return nil, msgNotStaging(p.StagingTable)
	}
	if p.Action == "" {
		p.Action = "upsert"
	}
	if p.Action != "upsert" && p.Action != "delete" {
		p.Action = "upsert"
	}
	if p.Action == "upsert" && len(p.Fields) == 0 {
		return nil, msgNoFields()
	}
	if p.Action == "upsert" {
		if ed.Columns == nil {
			return nil, msgNoStagingDB()
		}
		cols, err := ed.Columns.Columns(ctx, p.StagingTable)
		if err != nil {
			return nil, err
		}
		have := map[string]bool{}
		for _, c := range cols {
			have[c] = true
		}
		for _, f := range sortedKeys(p.Fields) {
			if !have[p.Fields[f]] {
				return nil, msgNoColumn(p.StagingTable, p.Fields[f])
			}
		}
	}

	var out *Change
	err := ed.Store.inTenant(ctx, a.TenantID, func(tx *sqlx.Tx) error {
		gold, err := isGoldCopy(ctx, tx, a.TenantID)
		if err != nil {
			return err
		}
		if !a.canAdminister(gold) {
			return msgNotAllowed()
		}
		// The object as the tenant sees it: its own, else the gold copy's.
		var boID string
		err = tx.GetContext(ctx, &boID, `SELECT id::text FROM public.business_objects
			WHERE bo_key = $2 AND (tenant_id::text = $1 OR tenant_id = public.uisce_gold_copy_tenant_id())
			ORDER BY (tenant_id::text = $1) DESC LIMIT 1`, a.TenantID, p.BOKey)
		if err != nil {
			return msgNoBO(p.BOKey)
		}
		if p.Action == "upsert" {
			var names []string
			if err := tx.SelectContext(ctx, &names, `SELECT field_name FROM public.business_object_fields WHERE bo_id::text = $1`, boID); err != nil {
				return err
			}
			known := map[string]bool{}
			for _, n := range names {
				known[n] = true
			}
			for _, f := range sortedKeys(p.Fields) {
				if !known[f] {
					return msgNoField(p.BOKey, f)
				}
			}
		}
		before, err := ownFields(ctx, tx, a.TenantID, boID, p.StagingTable)
		if err != nil {
			return err
		}
		if p.Action == "delete" && before == nil {
			return msgNoBinding(p.StagingTable, p.BOKey)
		}
		var id string
		if err := tx.GetContext(ctx, &id, `INSERT INTO public.staging_binding_changes
			(tenant_id, bo_id, staging_table, action, fields, before, reason, requested_by, requested_by_name)
			VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, NULLIF($7, ''), $8, NULLIF($9, '')) RETURNING id::text`,
			a.TenantID, boID, p.StagingTable, p.Action, jsonOrNil(p.Fields, p.Action == "upsert"), jsonOrNil(before, before != nil),
			p.Reason, a.UserID, a.Name); err != nil {
			return err
		}
		out, err = change(ctx, tx, a.TenantID, id)
		return err
	})
	return out, err
}

// Decide approves (applying the change) or rejects a pending change. The
// proposer can't decide their own change.
func (ed *Editor) Decide(ctx context.Context, a Actor, id string, approve bool, comment string) (*Change, error) {
	var out *Change
	err := ed.Store.inTenant(ctx, a.TenantID, func(tx *sqlx.Tx) error {
		c, err := change(ctx, tx, a.TenantID, id)
		if err != nil {
			return err
		}
		gold, err := isGoldCopy(ctx, tx, a.TenantID)
		if err != nil {
			return err
		}
		if !a.canAdminister(gold) {
			return msgChangeNotFound(id)
		}
		if c.Status != "pending" {
			return msgDecided()
		}
		if c.RequestedBy == a.UserID {
			return msgOwnChange()
		}
		status := "rejected"
		if approve {
			now, err := ownFields(ctx, tx, a.TenantID, c.BOID, c.StagingTable)
			if err != nil {
				return err
			}
			if !sameFields(now, c.Before) {
				return msgStale()
			}
			if c.Action == "delete" {
				_, err = tx.ExecContext(ctx, `DELETE FROM public.staging_bindings
					WHERE tenant_id::text = $1 AND bo_id::text = $2 AND staging_table = $3`, a.TenantID, c.BOID, c.StagingTable)
			} else {
				_, err = tx.ExecContext(ctx, `INSERT INTO public.staging_bindings (tenant_id, bo_id, staging_table, fields, applied_change_id)
					VALUES ($1::uuid, $2::uuid, $3, $4, $5::uuid)
					ON CONFLICT (tenant_id, bo_id, staging_table) DO UPDATE
					SET fields = EXCLUDED.fields, applied_change_id = EXCLUDED.applied_change_id,
					    version = staging_bindings.version + 1, updated_at = now()`,
					a.TenantID, c.BOID, c.StagingTable, jsonOrNil(c.Fields, true), id)
			}
			if err != nil {
				return err
			}
			status = "applied"
		}
		if _, err := tx.ExecContext(ctx, `UPDATE public.staging_binding_changes
			SET status = $2, reviewed_by = $3, reviewed_by_name = NULLIF($4, ''), reviewed_at = now(),
			    review_comment = NULLIF($5, ''), applied_at = CASE WHEN $2 = 'applied' THEN now() END
			WHERE tenant_id::text = $6 AND id::text = $1`, id, status, a.UserID, a.Name, comment, a.TenantID); err != nil {
			return err
		}
		out, err = change(ctx, tx, a.TenantID, id)
		return err
	})
	return out, err
}

// Withdraw lets the proposer take back a pending change.
func (ed *Editor) Withdraw(ctx context.Context, a Actor, id string) (*Change, error) {
	var out *Change
	err := ed.Store.inTenant(ctx, a.TenantID, func(tx *sqlx.Tx) error {
		c, err := change(ctx, tx, a.TenantID, id)
		if err != nil {
			return err
		}
		if c.Status != "pending" {
			return msgDecided()
		}
		if c.RequestedBy != a.UserID {
			return msgNotProposer()
		}
		if _, err := tx.ExecContext(ctx, `UPDATE public.staging_binding_changes SET status = 'withdrawn'
			WHERE tenant_id::text = $2 AND id::text = $1`, id, a.TenantID); err != nil {
			return err
		}
		out, err = change(ctx, tx, a.TenantID, id)
		return err
	})
	return out, err
}

// ownFields is the tenant's own binding for the object and table (nil when
// none): a tenant's changes only ever touch its own row.
func ownFields(ctx context.Context, tx *sqlx.Tx, tenantID, boID, table string) (map[string]string, error) {
	var raw []byte
	err := tx.GetContext(ctx, &raw, `SELECT fields FROM public.staging_bindings
		WHERE tenant_id::text = $1 AND bo_id::text = $2 AND staging_table = $3`, tenantID, boID, table)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	return out, json.Unmarshal(raw, &out)
}

func sameFields(a, b map[string]string) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	return reflect.DeepEqual(a, b)
}

func jsonOrNil(m map[string]string, present bool) any {
	if !present {
		return nil
	}
	b, _ := json.Marshal(m)
	return string(b)
}

func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
