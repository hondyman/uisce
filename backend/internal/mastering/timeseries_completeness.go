package mastering

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

// Completeness is the missing-price control: for a valuation date, every
// instrument of the expected universe (by asset class) needs a current
// golden price of each expected price type. Each gap is a MISSING_PRICE
// exception (once while open); a gap that has since been priced is
// resolved. Run at the cutoff - from the console, the API or a schedule
// (target "price:completeness", which Tidal / Control-M can trigger).

// Completeness is the outcome of a check.
type Completeness struct {
	Date     string        `json:"date"`
	Expected int           `json:"expected"`
	Priced   int           `json:"priced"`
	Held     int           `json:"held"` // no current price, a version waiting for a steward
	Missing  int           `json:"missing"`
	Stale    int           `json:"stale"` // open stale-price events for the date
	Raised   int           `json:"raised"`
	Resolved int           `json:"resolved"`
	Gaps     []MissingItem `json:"gaps"`
}

// MissingItem is one expected price with no current golden price.
type MissingItem struct {
	EntityID  string  `db:"eid" json:"entity_id"`
	Code      *string `db:"code" json:"code,omitempty"`
	Name      *string `db:"name" json:"name,omitempty"`
	PriceType string  `db:"pt" json:"price_type"`
	Held      bool    `db:"held" json:"held"`
}

// CheckCompleteness runs the missing-price control for a valuation date.
func (e *Engine) CheckCompleteness(ctx context.Context, tenantID, entity, date, runID string) (*Completeness, error) {
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return nil, msgBadProfile(strings.ToUpper(entity), "date "+date)
	}
	p, inst, err := e.seriesProfiles(ctx, tenantID, entity)
	if err != nil {
		return nil, err
	}
	if !p.timeSeries() || inst == nil {
		return nil, msgBadProfile(p.EntityCd, "not a time-series profile")
	}
	var out *Completeness
	err = e.inTenant(ctx, tenantID, func(tx *sqlx.Tx) error {
		var err error
		out, err = checkCompleteness(ctx, tx, p, inst, tenantID, date, runID)
		return err
	})
	return out, err
}

// checkCompleteness is the check, in tx (under the tenant's row security).
func checkCompleteness(ctx context.Context, tx *sqlx.Tx, p, inst *Profile, tenantID, date, runID string) (*Completeness, error) {
	sr := p.Settings.Series
	rules, _ := json.Marshal(sr.Controls.expected())
	out := &Completeness{Date: date, Gaps: []MissingItem{}}
	err := func() error {
		// The expected keys: current instruments of each rule's asset class
		// (and not inactive) x its price types; whether each has a current
		// golden price, or only a held one.
		expected := fmt.Sprintf(`WITH rules AS (SELECT x.asset_class, pt FROM jsonb_to_recordset($1::jsonb) AS x(asset_class text, price_types text[])
				CROSS JOIN LATERAL unnest(x.price_types) AS pt),
			exp AS (SELECT DISTINCT a.%[1]s AS eid, rules.pt, a.%[2]s::text AS code, to_jsonb(a)->>%[3]s AS name
				FROM %[4]s a JOIN rules ON rules.asset_class IN ('*', to_jsonb(a)->>'asset_class')
				WHERE %[5]s AND a.%[1]s IS NOT NULL AND COALESCE(lower(to_jsonb(a)->>'status'), 'active') NOT IN ('inactive', 'matured', 'delisted', 'merged')),
			state AS (SELECT exp.*,
				EXISTS (SELECT 1 FROM mdm.price_golden_record g WHERE g.price_entity_type = $2 AND g.price_entity_id = exp.eid
					AND g.price_type_cd = exp.pt AND g.price_date = $3::date AND g.is_current) AS priced,
				EXISTS (SELECT 1 FROM mdm.price_golden_record g WHERE g.price_entity_type = $2 AND g.price_entity_id = exp.eid
					AND g.price_type_cd = exp.pt AND g.price_date = $3::date AND g.status = 'REVIEW') AS held
				FROM exp)`,
			qi(inst.entityCol()), qi(inst.AnchorCodeColumn), quoteLit(inst.Settings.NameAttribute), qi(inst.AnchorTable), inst.current("a"))
		var n struct {
			Expected int `db:"expected"`
			Priced   int `db:"priced"`
			Held     int `db:"held"`
		}
		if err := tx.GetContext(ctx, &n, expected+`
			SELECT count(*) AS expected, count(*) FILTER (WHERE priced) AS priced, count(*) FILTER (WHERE NOT priced AND held) AS held FROM state`,
			rules, sr.EntityType, date); err != nil {
			return fmt.Errorf("completeness: %w", err)
		}
		out.Expected, out.Priced, out.Held = n.Expected, n.Priced, n.Held
		out.Missing = n.Expected - n.Priced
		if err := tx.SelectContext(ctx, &out.Gaps, expected+`
			SELECT eid::text AS eid, code, name, pt, held FROM state WHERE NOT priced ORDER BY held, name, pt LIMIT 500`,
			rules, sr.EntityType, date); err != nil {
			return err
		}
		// Raise the gaps (a held price is already with a steward: its own
		// exception says why), resolve the ones priced since.
		res, err := tx.ExecContext(ctx, expected+`
			INSERT INTO mdm.price_exception (tenant_id, price_entity_type, price_entity_id, price_date, exception_type, severity,
				exception_description, detected_at, status, custom_attributes)
			SELECT $4::uuid, $2, s.eid, $3::date, 'MISSING_PRICE', 'ERROR',
				COALESCE(s.code || ' ' || s.name, s.code, s.eid::text) || ' ' || s.pt || ' ' || $3 || ': no golden price',
				now(), 'OPEN', jsonb_build_object('price_type', s.pt, 'run_id', $5::text)
			FROM state s
			WHERE NOT s.priced AND NOT s.held
			  AND NOT EXISTS (SELECT 1 FROM mdm.price_exception x WHERE x.status = 'OPEN' AND x.exception_type = 'MISSING_PRICE'
				AND x.price_entity_id = s.eid AND x.price_date = $3::date AND x.custom_attributes->>'price_type' = s.pt)`,
			rules, sr.EntityType, date, tenantID, runID)
		if err != nil {
			return err
		}
		raised, _ := res.RowsAffected()
		out.Raised = int(raised)
		res, err = tx.ExecContext(ctx, `UPDATE mdm.price_exception x SET status = 'RESOLVED', resolved_at = now(),
				resolution_note = 'Priced since (completeness check ' || $3::text || ')'
			WHERE x.status IN ('OPEN', 'IN_REVIEW') AND x.exception_type = 'MISSING_PRICE' AND x.price_date = $2::date
			  AND EXISTS (SELECT 1 FROM mdm.price_golden_record g WHERE g.price_entity_type = $1 AND g.price_entity_id = x.price_entity_id
				AND g.price_type_cd = x.custom_attributes->>'price_type' AND g.price_date = x.price_date AND g.is_current)`,
			sr.EntityType, date, runID)
		if err != nil {
			return err
		}
		resolved, _ := res.RowsAffected()
		out.Resolved = int(resolved)
		return tx.GetContext(ctx, &out.Stale, `SELECT count(*) FROM mdm.price_stale_event WHERE status = 'OPEN' AND price_date = $1::date`, date)
	}()
	return out, err
}

func quoteLit(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

// completenessDate is the valuation date a scheduled check covers:
// params.valuation_date, else the scheduled time's date in params.time_zone
// (default America/New_York) less params.days_back.
func completenessDate(scheduled time.Time, params map[string]any) string {
	if d, ok := params["valuation_date"].(string); ok {
		if _, err := time.Parse("2006-01-02", d); err == nil {
			return d
		}
	}
	loc, err := time.LoadLocation("America/New_York")
	if tz, ok := params["time_zone"].(string); ok {
		if l, e := time.LoadLocation(tz); e == nil {
			loc, err = l, nil
		}
	}
	if err != nil || loc == nil {
		loc = time.UTC
	}
	back := 0
	switch v := params["days_back"].(type) {
	case float64:
		back = int(v)
	case int:
		back = v
	}
	if scheduled.IsZero() {
		scheduled = time.Now()
	}
	return scheduled.In(loc).AddDate(0, 0, -back).Format("2006-01-02")
}
