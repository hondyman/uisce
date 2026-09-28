package mastering

// Maker-checker editing of mastering configuration: the vendor registry
// (mdm.source_systems), each entity's source hierarchy
// (mdm.<prefix>_source_priority) and match rules (mdm.<prefix>_match_rule).
//
// Reading shows the gold copy's rows and the tenant's own together, as the
// engine sees them (readConfig): a tenant's row with the same key as a gold
// row overrides it. Changing is a proposal (mdm.mastering_config_change) that
// a second administrator approves; only then is it applied, in the approval's
// transaction. A tenant never writes a gold-copy row - it proposes its own.
//
// Columns are read from each table, not hard-coded: product scopes its
// hierarchy by product type, security by asset class, price by price type
// and currency, and the same code handles all of them.

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// Configuration kinds.
const (
	KindSourceSystem   = "source_system"
	KindSourcePriority = "source_priority"
	KindMatchRule      = "match_rule"
)

// ConfigColumn is one editable column of a configuration table.
type ConfigColumn struct {
	Name     string `json:"name"`
	Type     string `json:"type"` // text, number, integer, boolean, json, list, source
	Required bool   `json:"required"`
	// Key: part of the row's identity for overriding (a tenant row with the
	// same key replaces the gold copy's).
	Key bool `json:"key"`
}

// ConfigRow is one row as the tenant sees it.
type ConfigRow struct {
	ID string `json:"id"`
	// Origin: core (the gold copy's) or tenant (this tenant's own).
	Origin string `json:"origin"`
	// Inherited: a gold-copy row seen from another tenant - read-only there.
	Inherited bool `json:"inherited"`
	// Overridden: an inherited row this tenant replaces with its own.
	Overridden bool `json:"overridden"`
	// Pending: a proposed change to this row is waiting for approval.
	Pending bool           `json:"pending"`
	Values  map[string]any `json:"values"`
}

// ConfigTable is a configuration table for one entity (or the registry).
type ConfigTable struct {
	Kind    string         `json:"kind"`
	Entity  string         `json:"entity,omitempty"`
	Columns []ConfigColumn `json:"columns"`
	Rows    []ConfigRow    `json:"rows"`
}

// ConfigProposal is a proposed change.
type ConfigProposal struct {
	Kind   string         `json:"kind"`
	Entity string         `json:"entity"`
	Action string         `json:"action"`
	// TargetID: the tenant's own row to change or remove; empty for a new
	// row (including an override of a gold-copy row).
	TargetID string         `json:"target_id"`
	Values   map[string]any `json:"values"`
	Reason   string         `json:"reason"`
}

// ConfigChange is a proposal and what became of it.
type ConfigChange struct {
	ID              string          `db:"id" json:"id"`
	Entity          *string         `db:"entity_cd" json:"entity,omitempty"`
	Kind            string          `db:"kind" json:"kind"`
	Action          string          `db:"action" json:"action"`
	TargetID        *string         `db:"target_id" json:"target_id,omitempty"`
	Values          json.RawMessage `db:"values" json:"values"`
	Before          json.RawMessage `db:"before" json:"before,omitempty"`
	Reason          *string         `db:"reason" json:"reason,omitempty"`
	Status          string          `db:"status" json:"status"`
	RequestedBy     string          `db:"requested_by" json:"requested_by"`
	RequestedByName *string         `db:"requested_by_name" json:"requested_by_name,omitempty"`
	RequestedAt     string          `db:"requested_at" json:"requested_at"`
	ReviewedByName  *string         `db:"reviewed_by_name" json:"reviewed_by_name,omitempty"`
	ReviewedAt      *string         `db:"reviewed_at" json:"reviewed_at,omitempty"`
	ReviewComment   *string         `db:"review_comment" json:"review_comment,omitempty"`
	// Mine: the caller proposed it (so may withdraw, never approve).
	Mine bool `db:"-" json:"mine"`
}

// systemColumns are never edited through configuration changes.
var systemColumns = map[string]bool{
	"id": true, "tenant_id": true, "created_at": true, "updated_at": true, "custom_attributes": true,
	"last_sync_at": true, "last_sync_status": true, "contact_party_id": true,
}

// configTarget is the table a kind lives in for an entity.
type configTarget struct {
	table     string // quoted, schema-qualified
	schema    string
	name      string
	sourceCol string // source_system_id / source_id; "" for the registry
	columns   []ConfigColumn
	dbTypes   map[string]string
}

func (e *Engine) configTarget(ctx context.Context, tx *sqlx.Tx, tenantID, kind, entity string) (*configTarget, error) {
	t := &configTarget{}
	switch kind {
	case KindSourceSystem:
		t.schema, t.name = "mdm", "source_systems"
	case KindSourcePriority, KindMatchRule:
		p, err := getProfile(ctx, tx, tenantID, entity)
		if err != nil {
			return nil, err
		}
		if kind == KindMatchRule && p.timeSeries() {
			return nil, msgTimeSeries(p.EntityCd)
		}
		// The kinds are the table suffixes: mdm.<prefix>_source_priority, mdm.<prefix>_match_rule.
		parts := strings.SplitN(p.plainTable(kind), ".", 2)
		t.schema, t.name = parts[0], parts[1]
	default:
		return nil, msgUnknownConfigKind(kind)
	}
	t.table = qi(t.schema + "." + t.name)
	var cols []struct {
		Name     string `db:"column_name"`
		DataType string `db:"data_type"`
		Required bool   `db:"required"`
	}
	if err := tx.SelectContext(ctx, &cols, `SELECT column_name, data_type,
			(is_nullable = 'NO' AND column_default IS NULL) AS required
		FROM information_schema.columns WHERE table_schema = $1 AND table_name = $2 ORDER BY ordinal_position`, t.schema, t.name); err != nil {
		return nil, err
	}
	if len(cols) == 0 {
		return nil, msgUnknownConfigKind(kind + " (" + t.schema + "." + t.name + ")")
	}
	t.dbTypes = map[string]string{}
	for _, c := range cols {
		t.dbTypes[c.Name] = c.DataType
		if systemColumns[c.Name] {
			continue
		}
		if c.Name == "source_system_id" || c.Name == "source_id" {
			t.sourceCol = c.Name
			t.columns = append(t.columns, ConfigColumn{Name: "source", Type: "source", Required: true, Key: kind == KindSourcePriority})
			continue
		}
		typ := columnType(c.DataType)
		if typ == "" {
			continue // uuid references and the like are not edited here
		}
		t.columns = append(t.columns, ConfigColumn{Name: c.Name, Type: typ, Required: c.Required, Key: isKey(kind, c.Name, typ)})
	}
	return t, nil
}

func columnType(dataType string) string {
	switch dataType {
	case "character varying", "text", "character":
		return "text"
	case "integer", "smallint", "bigint":
		return "integer"
	case "numeric", "double precision", "real":
		return "number"
	case "boolean":
		return "boolean"
	case "jsonb", "json":
		return "json"
	case "ARRAY":
		return "list"
	}
	return ""
}

// isKey: the columns that identify a row for overriding. Match rules are
// keyed by rule code (config.go reads DISTINCT ON rule_cd); the registry by
// code; a hierarchy row by its scope, group and source.
func isKey(kind, name, typ string) bool {
	switch kind {
	case KindMatchRule:
		return name == "rule_cd"
	case KindSourceSystem:
		return name == "code"
	}
	return typ == "text"
}

func (t *configTarget) column(name string) (ConfigColumn, bool) {
	for _, c := range t.columns {
		if c.Name == name {
			return c, true
		}
	}
	return ConfigColumn{}, false
}

// keyOf is a row's identity from its values.
func (t *configTarget) keyOf(values map[string]any) string {
	var parts []string
	for _, c := range t.columns {
		if c.Key {
			v := ""
			if values[c.Name] != nil {
				v = strings.ToUpper(strings.TrimSpace(fmt.Sprint(values[c.Name])))
			}
			if v == "*" || v == "ALL" {
				v = ""
			}
			parts = append(parts, v)
		}
	}
	return strings.Join(parts, "|")
}

// inGold runs fn under the tenant's and the gold copy's visibility (the
// engine's read path).
func (e *Engine) inGold(ctx context.Context, tenantID string, fn func(tx *sqlx.Tx, gold string) error) error {
	gold, err := e.GoldCopy(ctx)
	if err != nil {
		return err
	}
	return e.inConfig(ctx, tenantID, func(tx *sqlx.Tx) error { return fn(tx, gold) })
}

// sourceCodes maps registry ids to codes (and back) for the sources the
// tenant can see.
func sourceCodes(ctx context.Context, tx *sqlx.Tx, tenantID, gold string) (map[string]string, map[string]string, error) {
	var rows []struct {
		ID   string `db:"id"`
		Code string `db:"code"`
	}
	if err := tx.SelectContext(ctx, &rows, `SELECT id::text, code FROM mdm.source_systems
		WHERE tenant_id::text IN ($1, $2) ORDER BY (tenant_id::text = $1)`, tenantID, gold); err != nil {
		return nil, nil, err
	}
	byID, byCode := map[string]string{}, map[string]string{}
	for _, r := range rows { // own rows last, so they win by code
		byID[r.ID] = r.Code
		byCode[strings.ToUpper(r.Code)] = r.ID
	}
	return byID, byCode, nil
}

// rowValues turns a table row into its editable values (source as code).
func (t *configTarget) rowValues(raw json.RawMessage, codes map[string]string) map[string]any {
	all := map[string]any{}
	_ = json.Unmarshal(raw, &all)
	out := map[string]any{}
	for _, c := range t.columns {
		if c.Type == "source" {
			if id, ok := all[t.sourceCol].(string); ok {
				out["source"] = codes[id]
			}
			continue
		}
		out[c.Name] = all[c.Name]
	}
	return out
}

// ConfigTable reads a configuration table as the tenant sees it.
func (e *Engine) ConfigTable(ctx context.Context, tenantID, kind, entity string) (*ConfigTable, error) {
	out := &ConfigTable{Kind: kind, Entity: strings.ToUpper(entity), Rows: []ConfigRow{}}
	err := e.inGold(ctx, tenantID, func(tx *sqlx.Tx, gold string) error {
		t, err := e.configTarget(ctx, tx, tenantID, kind, entity)
		if err != nil {
			return err
		}
		out.Columns = t.columns
		codes, _, err := sourceCodes(ctx, tx, tenantID, gold)
		if err != nil {
			return err
		}
		var rows []struct {
			ID     string          `db:"id"`
			Tenant string          `db:"tenant_id"`
			Row    json.RawMessage `db:"row"`
		}
		if err := tx.SelectContext(ctx, &rows, `SELECT id::text, tenant_id::text, to_jsonb(x) AS row FROM `+t.table+` x
			WHERE tenant_id::text IN ($1, $2)`, tenantID, gold); err != nil {
			return err
		}
		var pending []string
		if err := tx.SelectContext(ctx, &pending, `SELECT target_id::text FROM mdm.mastering_config_change
			WHERE tenant_id::text = $1 AND kind = $2 AND status = 'pending' AND target_id IS NOT NULL`, tenantID, kind); err != nil {
			return err
		}
		waiting := map[string]bool{}
		for _, id := range pending {
			waiting[id] = true
		}
		own := map[string]bool{}
		for _, r := range rows {
			if r.Tenant == tenantID {
				own[t.keyOf(t.rowValues(r.Row, codes))] = true
			}
		}
		for _, r := range rows {
			v := t.rowValues(r.Row, codes)
			core := r.Tenant == gold
			inherited := core && tenantID != gold
			out.Rows = append(out.Rows, ConfigRow{
				ID: r.ID, Origin: map[bool]string{true: "core", false: "tenant"}[core], Inherited: inherited,
				Overridden: inherited && own[t.keyOf(v)], Pending: waiting[r.ID], Values: v,
			})
		}
		sortConfigRows(out.Rows, t)
		return nil
	})
	return out, err
}

func sortConfigRows(rows []ConfigRow, t *configTarget) {
	num := func(v any) float64 {
		f, _ := v.(float64)
		return f
	}
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i].Values, rows[j].Values
		ka, kb := t.keyOf(withoutSource(a)), t.keyOf(withoutSource(b))
		if ka != kb {
			return ka < kb
		}
		if pa, pb := num(a["priority"]), num(b["priority"]); pa != pb {
			return pa < pb
		}
		return rows[i].Origin > rows[j].Origin // the tenant's own first
	})
}

func withoutSource(v map[string]any) map[string]any {
	out := make(map[string]any, len(v))
	for k, x := range v {
		if k != "source" {
			out[k] = x
		}
	}
	return out
}

// ProposeConfig records a proposed change; nothing is applied yet.
func (e *Engine) ProposeConfig(ctx context.Context, a Actor, in ConfigProposal) (*ConfigChange, error) {
	in.Action = strings.ToLower(strings.TrimSpace(in.Action))
	if in.Action == "" {
		in.Action = "upsert"
	}
	if in.Action != "upsert" && in.Action != "delete" {
		return nil, msgUnknownConfigKind(in.Kind + "/" + in.Action)
	}
	if in.Kind == KindSourceSystem {
		in.Entity = ""
	}
	var before json.RawMessage
	var values map[string]any
	err := e.inGold(ctx, a.TenantID, func(tx *sqlx.Tx, gold string) error {
		t, err := e.configTarget(ctx, tx, a.TenantID, in.Kind, in.Entity)
		if err != nil {
			return err
		}
		codes, byCode, err := sourceCodes(ctx, tx, a.TenantID, gold)
		if err != nil {
			return err
		}
		if in.TargetID != "" {
			var tenant string
			var raw json.RawMessage
			err := tx.QueryRowxContext(ctx, `SELECT tenant_id::text, to_jsonb(x) FROM `+t.table+` x WHERE id::text = $1 AND tenant_id::text IN ($2, $3)`,
				in.TargetID, a.TenantID, gold).Scan(&tenant, &raw)
			if errors.Is(err, sql.ErrNoRows) {
				return msgConfigRowNotFound(in.TargetID)
			}
			if err != nil {
				return err
			}
			if tenant != a.TenantID {
				return msgConfigCoreRow()
			}
			before, _ = json.Marshal(t.rowValues(raw, codes))
		} else if in.Action == "delete" {
			return msgConfigRowNotFound("")
		}
		if in.Action == "delete" {
			values = map[string]any{}
			return nil
		}
		values = map[string]any{}
		for k, v := range in.Values {
			c, ok := t.column(k)
			if !ok {
				return msgConfigBadColumn(k, t.schema+"."+t.name)
			}
			if c.Type == "source" {
				code := strings.ToUpper(strings.TrimSpace(fmt.Sprint(v)))
				if _, ok := byCode[code]; !ok {
					return msgConfigUnknownSource(code)
				}
				v = code
			}
			values[k] = v
		}
		if in.Kind == KindMatchRule {
			// Check the keys the way the engine will read them: a rule it
			// cannot decode would stop mastering for the whole entity.
			m := MatchRule{RuleCd: fmt.Sprint(values["rule_cd"])}
			m.RawDeterministic, _ = json.Marshal(values["deterministic_keys"])
			m.RawFuzzy, _ = json.Marshal(values["fuzzy_keys"])
			if values["deterministic_keys"] == nil {
				m.RawDeterministic = nil
			}
			if values["fuzzy_keys"] == nil {
				m.RawFuzzy = nil
			}
			if err := m.decode(); err != nil {
				return msgConfigBadColumn("deterministic_keys / fuzzy_keys ("+err.Error()+")", t.schema+"."+t.name)
			}
		}
		if in.TargetID == "" {
			for _, c := range t.columns {
				if c.Required && (values[c.Name] == nil || fmt.Sprint(values[c.Name]) == "") {
					return msgConfigBadColumn(c.Name+" (required)", t.schema+"."+t.name)
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	v, _ := json.Marshal(values)
	var ch ConfigChange
	err = e.inTenant(ctx, a.TenantID, func(tx *sqlx.Tx) error {
		return tx.GetContext(ctx, &ch, `INSERT INTO mdm.mastering_config_change
				(tenant_id, entity_cd, kind, action, target_id, "values", before, reason, requested_by, requested_by_name)
			VALUES ($1::uuid, NULLIF($2, ''), $3, $4, NULLIF($5, '')::uuid, $6, $7, NULLIF($8, ''), $9, NULLIF($10, ''))
			RETURNING `+configChangeCols, a.TenantID, strings.ToUpper(in.Entity), in.Kind, in.Action, in.TargetID, v, nullJSON(before),
			strings.TrimSpace(in.Reason), a.UserID, a.Name)
	})
	if err != nil {
		return nil, err
	}
	ch.Mine = true
	return &ch, nil
}

const configChangeCols = `id::text, entity_cd, kind, action, target_id::text, "values", COALESCE(before, 'null'::jsonb) AS before, reason, status,
	requested_by, requested_by_name, requested_at::text, reviewed_by_name, reviewed_at::text, review_comment`

func nullJSON(b json.RawMessage) any {
	if len(b) == 0 {
		return nil
	}
	return []byte(b)
}

// ConfigChanges lists proposals: by kind and entity (either may be empty),
// status pending or empty for all.
func (e *Engine) ConfigChanges(ctx context.Context, a Actor, kind, entity, status string) ([]ConfigChange, error) {
	out := []ConfigChange{}
	err := e.inTenant(ctx, a.TenantID, func(tx *sqlx.Tx) error {
		return tx.SelectContext(ctx, &out, `SELECT `+configChangeCols+` FROM mdm.mastering_config_change
			WHERE tenant_id = $1::uuid AND ($2 = '' OR kind = $2) AND ($3 = '' OR entity_cd = $3) AND ($4 = '' OR status = $4)
			ORDER BY requested_at DESC LIMIT 200`, a.TenantID, kind, strings.ToUpper(entity), strings.ToLower(status))
	})
	for i := range out {
		out[i].Mine = out[i].RequestedBy == a.UserID
	}
	return out, err
}

// DecideConfig approves, rejects or withdraws a pending proposal. Approving
// applies it in the same transaction.
func (e *Engine) DecideConfig(ctx context.Context, a Actor, id, decision, comment string) (*ConfigChange, error) {
	gold, err := e.GoldCopy(ctx)
	if err != nil {
		return nil, err
	}
	var out ConfigChange
	err = e.inTenant(ctx, a.TenantID, func(tx *sqlx.Tx) error {
		// The gold copy's rows are visible for source lookups while applying.
		if _, err := tx.ExecContext(ctx, `SELECT set_config('app.shared_reference_tenant', $1, true)`, gold); err != nil {
			return err
		}
		var ch ConfigChange
		err := tx.GetContext(ctx, &ch, `SELECT `+configChangeCols+` FROM mdm.mastering_config_change
			WHERE id::text = $1 AND tenant_id = $2::uuid FOR UPDATE`, id, a.TenantID)
		if errors.Is(err, sql.ErrNoRows) {
			return msgConfigChangeNotFound(id)
		}
		if err != nil {
			return err
		}
		if ch.Status != "pending" {
			return msgConfigChangeClosed(id, ch.Status)
		}
		mine := ch.RequestedBy == a.UserID
		status := ""
		switch decision {
		case "withdraw":
			if !mine {
				return msgNotAllowed()
			}
			status = "withdrawn"
		case "approve", "reject":
			if !a.Admin {
				return msgNotAdmin()
			}
			if mine {
				return msgConfigOwnChange()
			}
			status = map[string]string{"approve": "applied", "reject": "rejected"}[decision]
		default:
			return msgUnknownConfigKind(decision)
		}
		var applied any
		if status == "applied" {
			entity := ""
			if ch.Entity != nil {
				entity = *ch.Entity
			}
			t, err := e.configTarget(ctx, tx, a.TenantID, ch.Kind, entity)
			if err != nil {
				return err
			}
			if applied, err = t.apply(ctx, tx, a.TenantID, gold, &ch); err != nil {
				return err
			}
		}
		return tx.GetContext(ctx, &out, `UPDATE mdm.mastering_config_change SET status = $2::text,
				reviewed_by = CASE WHEN $2::text = 'withdrawn' THEN NULL ELSE $3::text END,
				reviewed_by_name = CASE WHEN $2::text = 'withdrawn' THEN NULL ELSE NULLIF($4::text, '') END,
				reviewed_at = now(), review_comment = NULLIF($5, ''), applied_id = $6::uuid
			WHERE id = $1::uuid RETURNING `+configChangeCols, ch.ID, status, a.UserID, a.Name, strings.TrimSpace(comment), applied)
	})
	if err != nil {
		return nil, err
	}
	out.Mine = out.RequestedBy == a.UserID
	return &out, nil
}

// apply writes an approved change into its table and returns the row id.
func (t *configTarget) apply(ctx context.Context, tx *sqlx.Tx, tenantID, gold string, ch *ConfigChange) (any, error) {
	if ch.Action == "delete" {
		res, err := tx.ExecContext(ctx, `DELETE FROM `+t.table+` WHERE id::text = $1 AND tenant_id = $2::uuid`, *ch.TargetID, tenantID)
		if err != nil {
			return nil, configWriteError(err, t)
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return nil, msgConfigRowNotFound(*ch.TargetID)
		}
		return *ch.TargetID, nil
	}
	values := map[string]any{}
	if err := json.Unmarshal(ch.Values, &values); err != nil {
		return nil, err
	}
	row := map[string]any{}
	var cols []string
	for k, v := range values {
		if k == "source" {
			_, byCode, err := sourceCodes(ctx, tx, tenantID, gold)
			if err != nil {
				return nil, err
			}
			id, ok := byCode[strings.ToUpper(fmt.Sprint(v))]
			if !ok {
				return nil, msgConfigUnknownSource(fmt.Sprint(v))
			}
			row[t.sourceCol] = id
			cols = append(cols, t.sourceCol)
			continue
		}
		if _, ok := t.column(k); !ok {
			return nil, msgConfigBadColumn(k, t.schema+"."+t.name)
		}
		row[k] = v
		cols = append(cols, k)
	}
	sort.Strings(cols)
	quoted := make([]string, len(cols))
	for i, c := range cols {
		quoted[i] = qi(c)
	}
	list := strings.Join(quoted, ", ")
	body, _ := json.Marshal(row)
	var id string
	var err error
	if ch.TargetID != nil && *ch.TargetID != "" {
		set := "(" + list + ") = (SELECT " + list + " FROM jsonb_populate_record(NULL::" + t.table + ", $1::jsonb))"
		if _, ok := t.dbTypes["updated_at"]; ok {
			set += ", updated_at = now()"
		}
		if len(cols) == 0 {
			return *ch.TargetID, nil
		}
		err = tx.GetContext(ctx, &id, `UPDATE `+t.table+` SET `+set+` WHERE id::text = $2 AND tenant_id = $3::uuid RETURNING id::text`,
			body, *ch.TargetID, tenantID)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, msgConfigRowNotFound(*ch.TargetID)
		}
	} else {
		err = tx.GetContext(ctx, &id, `INSERT INTO `+t.table+` (id, tenant_id, `+list+`)
			SELECT gen_random_uuid(), $2::uuid, `+list+` FROM jsonb_populate_record(NULL::`+t.table+`, $1::jsonb)
			RETURNING id::text`, body, tenantID)
	}
	if err != nil {
		return nil, configWriteError(err, t)
	}
	return id, nil
}

func configWriteError(err error, t *configTarget) error {
	var pe *pq.Error
	if errors.As(err, &pe) {
		switch pe.Code {
		case "23505":
			return msgConfigKeyExists(t.schema + "." + t.name)
		case "23503":
			return msgConfigInUse(pe.Detail)
		case "22P02", "23502", "22003", "23514":
			return msgConfigBadColumn(pe.Column+" ("+pe.Message+")", t.schema+"."+t.name).WithStatus(http.StatusBadRequest)
		}
	}
	return err
}
