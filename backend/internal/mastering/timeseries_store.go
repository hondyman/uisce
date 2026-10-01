package mastering

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

// The console's view of a time-series (price) master: golden prices by
// valuation date, one price's versions and provenance, its exceptions.

// PriceFilter narrows the golden price list.
type PriceFilter struct {
	Date      string // valuation date (default: the latest with golden prices)
	Q         string // instrument code or name contains
	Status    string
	PriceType string
	Limit     int
}

// PriceSummary is one golden price: its latest version.
type PriceSummary struct {
	ID         string          `db:"id" json:"id"`
	EntityID   string          `db:"entity_id" json:"entity_id"`
	Code       *string         `db:"code" json:"code,omitempty"`
	Name       *string         `db:"name" json:"name,omitempty"`
	PriceType  string          `db:"price_type_cd" json:"price_type"`
	Date       string          `db:"price_date" json:"date"`
	Value      float64         `db:"golden_value" json:"value"`
	Currency   *string         `db:"currency" json:"currency,omitempty"`
	Winner     *string         `db:"winner" json:"winner,omitempty"`
	Sources    int             `db:"source_count" json:"sources"`
	Variance   *float64        `db:"variance_pct" json:"variance_pct,omitempty"`
	ChangePct  *float64        `db:"change_pct" json:"change_pct,omitempty"`
	Status     string          `db:"status" json:"status"`
	Version    int             `db:"golden_version" json:"version"`
	IsCurrent  bool            `db:"is_current" json:"is_current"`
	IsStale    bool            `db:"is_stale" json:"is_stale"`
	Confidence *float64        `db:"confidence" json:"confidence,omitempty"`
	UpdatedAt  time.Time       `db:"knowledge_timestamp" json:"updated_at"`
	Attributes json.RawMessage `db:"-" json:"-"`
}

// PriceList is a valuation date's golden prices and the dates there are.
type PriceList struct {
	Date   string         `json:"date"`
	Dates  []string       `json:"dates"`
	Prices []PriceSummary `json:"prices"`
}

// PriceVersion is one version of a golden price, with its provenance.
type PriceVersion struct {
	ID         string          `db:"id" json:"id"`
	Version    int             `db:"golden_version" json:"version"`
	Value      float64         `db:"golden_value" json:"value"`
	Currency   *string         `db:"currency" json:"currency,omitempty"`
	Winner     *string         `db:"winner" json:"winner,omitempty"`
	Status     string          `db:"status" json:"status"`
	IsCurrent  bool            `db:"is_current" json:"is_current"`
	IsStale    bool            `db:"is_stale" json:"is_stale"`
	Confidence *float64        `db:"confidence" json:"confidence,omitempty"`
	DQ         *float64        `db:"dq_score" json:"dq_score,omitempty"`
	Variance   *float64        `db:"variance_pct" json:"variance_pct,omitempty"`
	KnownAt    time.Time       `db:"knowledge_timestamp" json:"knowledge_at"`
	Attributes json.RawMessage `db:"golden_attributes" json:"provenance"`
}

// VarianceEvent is two sources disagreeing on a price beyond a threshold.
type VarianceEvent struct {
	ID       string  `db:"id" json:"id"`
	SourceA  *string `db:"source_a" json:"source_a,omitempty"`
	SourceB  *string `db:"source_b" json:"source_b,omitempty"`
	PriceA   float64 `db:"price_a" json:"price_a"`
	PriceB   float64 `db:"price_b" json:"price_b"`
	Pct      float64 `db:"variance_pct" json:"variance_pct"`
	Severity string  `db:"severity" json:"severity"`
	Status   string  `db:"status" json:"status"`
}

// PriceDetail is a golden price: the key, every version, exceptions and
// variance events.
type PriceDetail struct {
	ID         string          `json:"id"`
	EntityID   string          `json:"entity_id"`
	Code       *string         `json:"code,omitempty"`
	Name       *string         `json:"name,omitempty"`
	PriceType  string          `json:"price_type"`
	Date       string          `json:"date"`
	Versions   []PriceVersion  `json:"versions"`
	Exceptions []ExceptionRow  `json:"exceptions"`
	Variances  []VarianceEvent `json:"variances"`
}

// seriesProfiles is a time-series profile and its instrument's.
func (e *Engine) seriesProfiles(ctx context.Context, tenantID, entity string) (*Profile, *Profile, error) {
	var p, inst *Profile
	err := e.inConfig(ctx, tenantID, func(tx *sqlx.Tx) error {
		var err error
		if p, err = getProfile(ctx, tx, tenantID, entity); err != nil {
			return err
		}
		if !p.timeSeries() {
			return nil
		}
		inst, err = getProfile(ctx, tx, tenantID, p.Settings.Series.Instrument)
		return err
	})
	return p, inst, err
}

// instrumentCols selects the instrument's code and name for price rows
// (alias a, joined on price_entity_id).
func instrumentJoin(inst *Profile, alias string) (cols, join string) {
	cols = fmt.Sprintf("a.%s::text AS code, a.%s::text AS name", qi(inst.AnchorCodeColumn), qi(inst.Settings.NameAttribute))
	join = fmt.Sprintf("LEFT JOIN %s a ON a.%s = %s.price_entity_id AND %s", qi(inst.AnchorTable), qi(inst.entityCol()), alias, inst.current("a"))
	return cols, join
}

// GoldenPrices lists a valuation date's golden prices, latest version each.
func (e *Engine) GoldenPrices(ctx context.Context, tenantID, entity string, f PriceFilter) (*PriceList, error) {
	p, inst, err := e.seriesProfiles(ctx, tenantID, entity)
	if err != nil {
		return nil, err
	}
	if !p.timeSeries() {
		return nil, msgBadProfile(p.EntityCd, "not a time-series profile")
	}
	if f.Limit <= 0 || f.Limit > 1000 {
		f.Limit = 500
	}
	out := &PriceList{Dates: []string{}, Prices: []PriceSummary{}}
	cols, join := instrumentJoin(inst, "g")
	err = e.inTenant(ctx, tenantID, func(tx *sqlx.Tx) error {
		if err := tx.SelectContext(ctx, &out.Dates, `SELECT DISTINCT price_date::text FROM mdm.price_golden_record
			WHERE price_entity_type = $1 ORDER BY 1 DESC LIMIT 60`, p.Settings.Series.EntityType); err != nil {
			return err
		}
		out.Date = strings.TrimSpace(f.Date)
		if out.Date == "" && len(out.Dates) > 0 {
			out.Date = out.Dates[0]
		}
		if out.Date == "" {
			return nil
		}
		if _, err := time.Parse("2006-01-02", out.Date); err != nil {
			return msgBadProfile(p.EntityCd, "date "+out.Date)
		}
		return tx.SelectContext(ctx, &out.Prices, fmt.Sprintf(`SELECT g.id::text, g.price_entity_id::text AS entity_id, %[1]s,
				g.price_type_cd, g.price_date::text, g.golden_value, g.currency, g.winning_sources->>'value' AS winner, g.source_count,
				g.variance_pct, (g.golden_attributes->>'change_pct')::float8 AS change_pct, g.status, g.golden_version, g.is_current,
				g.is_stale, g.confidence, g.knowledge_timestamp
			FROM (SELECT DISTINCT ON (price_entity_id, price_type_cd) * FROM mdm.price_golden_record
				WHERE price_entity_type = $1 AND price_date = $2::date ORDER BY price_entity_id, price_type_cd, golden_version DESC) g
			%[2]s
			WHERE ($3 = '' OR a.%[3]s::text ILIKE '%%' || $3 || '%%' OR a.%[4]s::text ILIKE '%%' || $3 || '%%')
			  AND ($4 = '' OR g.status = $4) AND ($5 = '' OR g.price_type_cd = $5)
			ORDER BY (g.status = 'REVIEW') DESC, name NULLS LAST, g.price_type_cd LIMIT $6`,
			cols, join, qi(inst.AnchorCodeColumn), qi(inst.Settings.NameAttribute)),
			p.Settings.Series.EntityType, out.Date, strings.TrimSpace(f.Q), strings.ToUpper(f.Status), strings.ToUpper(f.PriceType), f.Limit)
	})
	return out, err
}

// PriceDetailByID is a golden price (any version's id) with every version
// of its key, its exceptions and variance events.
func (e *Engine) PriceDetailByID(ctx context.Context, tenantID, entity, id string) (*PriceDetail, error) {
	p, inst, err := e.seriesProfiles(ctx, tenantID, entity)
	if err != nil {
		return nil, err
	}
	if !p.timeSeries() {
		return nil, msgBadProfile(p.EntityCd, "not a time-series profile")
	}
	cols, join := instrumentJoin(inst, "g")
	d := &PriceDetail{ID: id, Versions: []PriceVersion{}, Exceptions: []ExceptionRow{}, Variances: []VarianceEvent{}}
	err = e.inTenant(ctx, tenantID, func(tx *sqlx.Tx) error {
		var key struct {
			EntityType string  `db:"price_entity_type"`
			EntityID   string  `db:"entity_id"`
			PriceType  string  `db:"price_type_cd"`
			Date       string  `db:"price_date"`
			Code       *string `db:"code"`
			Name       *string `db:"name"`
		}
		err := tx.GetContext(ctx, &key, fmt.Sprintf(`SELECT g.price_entity_type, g.price_entity_id::text AS entity_id, g.price_type_cd,
				g.price_date::text, %s FROM mdm.price_golden_record g %s WHERE g.id::text = $1`, cols, join), id)
		if errors.Is(err, sql.ErrNoRows) {
			return msgNoGolden(id)
		}
		if err != nil {
			return err
		}
		d.EntityID, d.PriceType, d.Date, d.Code, d.Name = key.EntityID, key.PriceType, key.Date, key.Code, key.Name
		if err := tx.SelectContext(ctx, &d.Versions, `SELECT id::text, golden_version, golden_value, currency, winning_sources->>'value' AS winner,
				status, is_current, is_stale, confidence, dq_score, variance_pct, knowledge_timestamp, golden_attributes
			FROM mdm.price_golden_record WHERE price_entity_type = $1 AND price_entity_id::text = $2 AND price_type_cd = $3 AND price_date = $4::date
			ORDER BY golden_version DESC`, key.EntityType, key.EntityID, key.PriceType, key.Date); err != nil {
			return err
		}
		if err := tx.SelectContext(ctx, &d.Exceptions, `SELECT x.id::text, x.price_entity_id::text AS golden_id,
				x.custom_attributes->>'source_entity_ref' AS identifier_value, x.exception_type, x.severity, x.exception_description,
				s.code AS source, x.status, x.detected_at
			FROM mdm.price_exception x LEFT JOIN mdm.source_systems s ON s.id = x.source_id
			WHERE x.price_entity_id::text = $1 AND x.price_date = $2::date
			  AND COALESCE(x.custom_attributes->>'price_type', $3) = $3
			ORDER BY x.detected_at DESC LIMIT 50`, key.EntityID, key.Date, key.PriceType); err != nil {
			return err
		}
		return tx.SelectContext(ctx, &d.Variances, `SELECT v.id::text, sa.code AS source_a, sb.code AS source_b, v.price_a, v.price_b,
				v.variance_pct, v.severity, v.status
			FROM mdm.price_variance_event v JOIN mdm.price_type pt ON pt.id = v.price_type_id
			LEFT JOIN mdm.source_systems sa ON sa.id = v.source_a_id LEFT JOIN mdm.source_systems sb ON sb.id = v.source_b_id
			WHERE v.price_entity_id::text = $1 AND v.price_date = $2::date AND pt.price_type_cd = $3
			ORDER BY v.variance_pct DESC`, key.EntityID, key.Date, key.PriceType)
	})
	return d, err
}

// priceExceptions lists a time-series profile's exceptions, newest first.
func priceExceptions(ctx context.Context, tx *sqlx.Tx, status string, limit int) ([]ExceptionRow, error) {
	out := []ExceptionRow{}
	// golden_id: the golden price the exception is about (its latest version), to open it.
	err := tx.SelectContext(ctx, &out, `SELECT x.id::text, (SELECT g.id::text FROM mdm.price_golden_record g
				WHERE g.price_entity_id = x.price_entity_id AND g.price_date = x.price_date
				  AND g.price_type_cd = x.custom_attributes->>'price_type' ORDER BY g.golden_version DESC LIMIT 1) AS golden_id,
			x.custom_attributes->>'source_entity_ref' AS identifier_value, x.exception_type, x.severity, x.exception_description,
			s.code AS source, x.status, x.detected_at
		FROM mdm.price_exception x LEFT JOIN mdm.source_systems s ON s.id = x.source_id
		WHERE (($1 = '' AND x.status IN ('OPEN', 'IN_REVIEW')) OR x.status = $1)
		ORDER BY x.detected_at DESC LIMIT $2`, strings.ToUpper(status), limit)
	return out, err
}
