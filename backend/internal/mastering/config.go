package mastering

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/jmoiron/sqlx"
)

// config is everything a run reads that isn't the data being mastered:
// the tenant's own configuration where it has some, else the gold copy's
// (inherited read-only).
type config struct {
	profile  *Profile
	match    []MatchRule
	survival map[string]SurvivalRule
	sources  map[string]string // source system code -> id
	// vendorCodes[ref attribute][source id][VENDOR CODE] = internal code
	vendorCodes map[string]map[string]map[string]string
	// hierarchy[FIELD_GROUP] = source codes, best first
	hierarchy map[string][]string
	// series: a time-series profile's configuration.
	series *seriesConfig
}

// rankingFor is the entity's source ranking for an attribute: its field
// group's, else the default group's.
func (c *config) rankingFor(attr string) []string {
	s := c.profile.Settings
	if g, ok := s.FieldGroups[attr]; ok {
		if r := c.hierarchy[strings.ToUpper(g)]; len(r) > 0 {
			return r
		}
	}
	return c.hierarchy[strings.ToUpper(s.DefaultFieldGroup)]
}

// inConfig runs fn with the gold copy readable as the shared reference
// tenant: the tenant sees its own rows and the gold copy's, never another
// tenant's. Configuration reads only.
func (e *Engine) inConfig(ctx context.Context, tenantID string, fn func(*sqlx.Tx) error) error {
	gold, err := e.GoldCopy(ctx)
	if err != nil {
		return err
	}
	tx, err := e.Data.BeginTxx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	if _, err := tx.ExecContext(ctx, `SELECT set_config('app.current_tenant', $1, true), set_config('app.shared_reference_tenant', $2, true)`, tenantID, gold); err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

// inTenant runs fn in a transaction under the tenant's row-level security
// (plus the shared reference tenant's reference data).
func (e *Engine) inTenant(ctx context.Context, tenantID string, fn func(*sqlx.Tx) error) error {
	tx, err := e.Data.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	if _, err := tx.ExecContext(ctx, `SELECT set_config('app.current_tenant', $1, true), set_config('uisce.current_tenant', $1, true)`, tenantID); err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

const profileCols = `id::text, tenant_id::text, entity_cd, display_name, bo_key, table_prefix, anchor_table,
	anchor_code_column, identifier_table, incoming_table, code_prefix, settings, is_active,
	COALESCE(to_jsonb(mastering_entity)->>'kind', 'RECORD') AS kind,
	tenant_id::text <> $1 AS inherited`

func getProfile(ctx context.Context, tx *sqlx.Tx, tenantID, entity string) (*Profile, error) {
	var p Profile
	err := tx.GetContext(ctx, &p, `SELECT `+profileCols+` FROM mdm.mastering_entity
		WHERE entity_cd = $2 AND is_active ORDER BY (tenant_id::text = $1) DESC LIMIT 1`, tenantID, strings.ToUpper(entity))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, msgNoProfile(entity)
	}
	if err != nil {
		return nil, err
	}
	if err := p.decode(); err != nil {
		return nil, msgBadProfile(p.EntityCd, err.Error())
	}
	return &p, nil
}

func (e *Engine) loadConfig(ctx context.Context, tenantID, entity string) (*config, error) {
	var c *config
	err := e.inConfig(ctx, tenantID, func(tx *sqlx.Tx) error {
		var err error
		c, err = readConfig(ctx, tx, tenantID, entity)
		return err
	})
	return c, err
}

// readConfig reads an entity's configuration in tx, which must see the
// tenant's rows and the gold copy's (see inConfig).
func readConfig(ctx context.Context, tx *sqlx.Tx, tenantID, entity string) (*config, error) {
	c := &config{survival: map[string]SurvivalRule{}, sources: map[string]string{}, vendorCodes: map[string]map[string]map[string]string{},
		hierarchy: map[string][]string{}}
	err := func() error {
		p, err := getProfile(ctx, tx, tenantID, entity)
		if err != nil {
			return err
		}
		c.profile = p
		if p.timeSeries() {
			if c.series, err = readSeriesConfig(ctx, tx, tenantID, p); err != nil {
				return err
			}
		}

		// Match rules: per rule code, the tenant's own over the gold copy's.
		if !p.timeSeries() {
			if err := tx.SelectContext(ctx, &c.match, `SELECT DISTINCT ON (rule_cd) id::text, rule_cd, deterministic_keys, fuzzy_keys,
				threshold_auto_match, COALESCE(threshold_review, threshold_auto_match) AS threshold_review, priority
			FROM `+p.table("match_rule")+` WHERE is_active
			ORDER BY rule_cd, (tenant_id::text = $1) DESC`, tenantID); err != nil {
				return fmt.Errorf("match rules: %w", err)
			}
			for i := range c.match {
				if err := c.match[i].decode(); err != nil {
					return msgBadProfile(p.EntityCd, fmt.Sprintf("match rule %s: %v", c.match[i].RuleCd, err))
				}
			}
			sortMatchRules(c.match)
		}

		var surv []struct {
			ID        string          `db:"id"`
			Attribute string          `db:"attribute_name"`
			Strategy  string          `db:"strategy"`
			Vendors   json.RawMessage `db:"priority_vendors"`
			Tolerance sql.NullFloat64 `db:"anomaly_tolerance_pct"`
			Staleness sql.NullInt64   `db:"staleness_max_age_sec"`
			Selection string          `db:"selection_rule_id"`
			SelMode   string          `db:"selection_mode"`
			OnNone    string          `db:"on_none_selected"`
			MinPeers  int             `db:"selection_min_peers"`
		}
		// The selection columns come from crims 0015; read through to_jsonb
		// so a data plane without them still masters.
		if err := tx.SelectContext(ctx, &surv, `SELECT DISTINCT ON (attribute_name) id::text, attribute_name, strategy,
				COALESCE(priority_vendors, '[]'::jsonb) AS priority_vendors, anomaly_tolerance_pct, staleness_max_age_sec,
				COALESCE(to_jsonb(r)->>'selection_rule_id', '') AS selection_rule_id,
				COALESCE(to_jsonb(r)->>'selection_mode', '') AS selection_mode,
				COALESCE(to_jsonb(r)->>'on_none_selected', '') AS on_none_selected,
				COALESCE((to_jsonb(r)->>'selection_min_peers')::int, 0) AS selection_min_peers
			FROM mdm.survivorship_rule r WHERE entity_type = $2 AND is_active
			ORDER BY attribute_name, (tenant_id::text = $1) DESC`, tenantID, p.EntityCd); err != nil {
			return fmt.Errorf("survivorship rules: %w", err)
		}
		for _, s := range surv {
			r := SurvivalRule{ID: s.ID, Attribute: s.Attribute, Strategy: strings.ToUpper(s.Strategy),
				TolerancePct: s.Tolerance.Float64, StalenessSec: int(s.Staleness.Int64),
				SelectionRuleID: s.Selection, SelectionMode: strings.ToUpper(s.SelMode), OnNoneSelected: strings.ToUpper(s.OnNone), SelectionMinPeers: s.MinPeers}
			_ = json.Unmarshal(s.Vendors, &r.Priority)
			c.survival[s.Attribute] = r
		}

		// The entity's source hierarchy, by field group: rows not scoped to a
		// type or class (to_jsonb keeps this generic across entities).
		// (A time-series profile ranks by its own scoped priority instead.)
		var hasPriority bool
		if err := tx.GetContext(ctx, &hasPriority, `SELECT to_regclass($1) IS NOT NULL`, p.plainTable("source_priority")); err != nil {
			return err
		}
		if hasPriority && !p.timeSeries() {
			// Order within each group by priority.
			var ordered []struct {
				Group string `db:"field_group"`
				Code  string `db:"code"`
			}
			if err := tx.SelectContext(ctx, &ordered, fmt.Sprintf(`SELECT upper(x.field_group) AS field_group, s.code
				FROM %s x JOIN mdm.source_systems s ON s.id = x.source_system_id
				WHERE COALESCE((to_jsonb(x)->>'is_active')::boolean, true)
				  AND COALESCE(to_jsonb(x)->>'product_type_cd', '') IN ('', '*', 'ALL')
				  AND COALESCE(to_jsonb(x)->>'asset_class_cd', '') IN ('', '*', 'ALL')
				ORDER BY upper(x.field_group), (x.tenant_id::text = $1) DESC, x.priority, s.code`, p.table("source_priority")), tenantID); err != nil {
				return fmt.Errorf("source priority: %w", err)
			}
			seen := map[string]bool{}
			for _, o := range ordered {
				if seen[o.Group+"|"+o.Code] {
					continue
				}
				seen[o.Group+"|"+o.Code] = true
				c.hierarchy[o.Group] = append(c.hierarchy[o.Group], strings.ToUpper(o.Code))
			}
		}

		var srcs []struct {
			ID   string `db:"id"`
			Code string `db:"code"`
		}
		if err := tx.SelectContext(ctx, &srcs, `SELECT id::text, code FROM mdm.source_systems`); err != nil {
			return fmt.Errorf("source systems: %w", err)
		}
		for _, s := range srcs {
			c.sources[strings.ToUpper(s.Code)] = s.ID
		}

		for _, ref := range p.Settings.References {
			if ref.MapTable == "" {
				continue
			}
			var rows []struct {
				Source string `db:"src"`
				Vendor string `db:"vendor"`
				Code   string `db:"code"`
			}
			q := fmt.Sprintf(`SELECT %s::text AS src, %s AS vendor, %s AS code FROM %s WHERE COALESCE((to_jsonb(m) ->> 'is_active')::boolean, true)
				ORDER BY (tenant_id::text = $1)`, qi(ref.MapSourceColumn), qi(ref.MapVendorColumn), qi(ref.MapCodeColumn), qi(ref.MapTable)+" m")
			if err := tx.SelectContext(ctx, &rows, q, tenantID); err != nil {
				return fmt.Errorf("code map %s: %w", ref.MapTable, err)
			}
			bySrc := map[string]map[string]string{}
			for _, r := range rows { // own rows last, so they win
				if bySrc[r.Source] == nil {
					bySrc[r.Source] = map[string]string{}
				}
				bySrc[r.Source][normCode(r.Vendor)] = r.Code
			}
			c.vendorCodes[ref.Attribute] = bySrc
		}
		return nil
	}()
	return c, err
}

func sortMatchRules(rs []MatchRule) {
	sort.SliceStable(rs, func(i, j int) bool { return rs[i].Priority < rs[j].Priority })
}

func normCode(s string) string { return strings.ToUpper(strings.TrimSpace(s)) }
