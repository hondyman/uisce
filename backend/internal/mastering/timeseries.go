package mastering

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/hondyman/uisce/backend/internal/stagingbind"
)

// Time-series mastering (the price master, docs/mdm-price-mastering-design.md):
// one golden value per entity x price type x valuation date, mastered
// set-based over a load rather than record at a time.
//
//  1. canonicalize - one INSERT ... SELECT turns the vendor's rows into
//     observations in staging.price_incoming through the staging binding (a
//     row with several price columns is several observations) and resolves
//     each to its golden instrument by the instrument's identifiers; an
//     unresolved quote is invalid (UNRESOLVED_INSTRUMENT), never a guess;
//  2. observe - mdm.price keeps one current observation per key and source;
//     a vendor resending a different value is a restatement (the old row is
//     kept in mdm.price_history and linked as the predecessor);
//  3. survive - per key, the layered survivorship of record mastering: the
//     ranking for the key's asset class, price type and currency, staleness
//     per source against the valuation date, the attribute's selection rule
//     (which sees the key's variance thresholds as `threshold`), stewards;
//  4. control - day-over-day move against the prior date's golden value,
//     cross-source variance events, currency mismatches;
//  5. publish - a new golden version only when the value, winner, status or
//     the candidates change; compact provenance on the golden row.
//
// Reads and writes go in chunks of keys, one statement each, so the round
// trips don't grow with the number of prices.

const seriesChunk = 2000

// seriesConfig is a time-series profile's configuration.
type seriesConfig struct {
	instrument *Profile
	priority   []priorityRow
	thresholds []thresholdRow
}

type priorityRow struct {
	AssetClass  sql.NullString `db:"asset_class_cd"`
	SubType     sql.NullString `db:"sec_sub_typ_cd"`
	PriceType   sql.NullString `db:"price_type_cd"`
	Currency    sql.NullString `db:"currency"`
	SourceCd    string         `db:"source_cd"`
	Priority    int            `db:"priority"`
	Fallback    bool           `db:"is_fallback"`
	MaxStaleMin sql.NullInt64  `db:"max_staleness_minutes"`
}

type thresholdRow struct {
	ID         string          `db:"id"`
	AssetClass sql.NullString  `db:"asset_class_cd"`
	SubType    sql.NullString  `db:"sec_sub_typ_cd"`
	PriceType  sql.NullString  `db:"price_type_cd"`
	Type       string          `db:"threshold_type"`
	Warning    sql.NullFloat64 `db:"warning_threshold"`
	Error      sql.NullFloat64 `db:"error_threshold"`
	Critical   sql.NullFloat64 `db:"critical_threshold"`
}

func readSeriesConfig(ctx context.Context, tx *sqlx.Tx, tenantID string, p *Profile) (*seriesConfig, error) {
	sr := p.Settings.Series
	c := &seriesConfig{}
	var err error
	if c.instrument, err = getProfile(ctx, tx, tenantID, sr.Instrument); err != nil {
		return nil, err
	}
	if c.instrument.IdentifierTable == nil {
		return nil, msgBadProfile(p.EntityCd, "instrument "+sr.Instrument+" has no identifier table")
	}
	// The tenant's own rows win over the gold copy's for the same scope.
	if err := tx.SelectContext(ctx, &c.priority, `SELECT DISTINCT ON (x.asset_class_cd, x.sec_sub_typ_cd, x.price_type_cd, x.currency, s.code)
			x.asset_class_cd, x.sec_sub_typ_cd, x.price_type_cd, x.currency, s.code AS source_cd, x.priority, x.is_fallback, x.max_staleness_minutes
		FROM mdm.price_source_priority x JOIN mdm.source_systems s ON s.id = x.source_id
		WHERE x.is_active AND x.price_entity_type = $2
		ORDER BY x.asset_class_cd, x.sec_sub_typ_cd, x.price_type_cd, x.currency, s.code, (x.tenant_id::text = $1) DESC`,
		tenantID, sr.EntityType); err != nil {
		return nil, fmt.Errorf("price source priority: %w", err)
	}
	if err := tx.SelectContext(ctx, &c.thresholds, `SELECT DISTINCT ON (asset_class_cd, sec_sub_typ_cd, price_type_cd)
			id::text, asset_class_cd, sec_sub_typ_cd, price_type_cd, threshold_type, warning_threshold, error_threshold, critical_threshold
		FROM mdm.price_variance_threshold WHERE is_active AND threshold_type = 'PERCENTAGE'
		ORDER BY asset_class_cd, sec_sub_typ_cd, price_type_cd, (tenant_id::text = $1) DESC`, tenantID); err != nil {
		return nil, fmt.Errorf("price variance thresholds: %w", err)
	}
	return c, nil
}

// wildcard: a scope column that applies to everything.
func wildcard(v sql.NullString) bool {
	return !v.Valid || v.String == "" || v.String == "*" || strings.EqualFold(v.String, "ALL")
}

// scopeMatch: how specifically a row's scope matches the key (-1: not at all).
func scopeMatch(pairs ...[2]any) int {
	n := 0
	for _, p := range pairs {
		col, val := p[0].(sql.NullString), p[1].(string)
		if wildcard(col) {
			continue
		}
		if !strings.EqualFold(col.String, val) {
			return -1
		}
		n++
	}
	return n
}

// seriesKey is one golden price: an instrument, a price type and a date.
type seriesKey struct {
	EntityType string
	EntityID   string
	PriceType  string
	Date       string // YYYY-MM-DD
}

func (k seriesKey) String() string { return k.EntityID + "|" + k.PriceType + "|" + k.Date }

// instrumentFacts are what survivorship and controls need of the instrument.
type instrumentFacts struct {
	AssetClass string
	SubType    string
	Currency   string
}

// ranking is the sources for a key, best first: the most specific matching
// scope of each source, non-fallback sources before fallbacks. maxStale is
// each source's allowed age in minutes (0: no limit).
func (c *seriesConfig) ranking(inst instrumentFacts, priceType, currency string) (order []string, maxStale map[string]int) {
	type pick struct {
		spec     int
		priority int
		fallback bool
		stale    int
	}
	best := map[string]pick{}
	for _, r := range c.priority {
		spec := scopeMatch([2]any{r.AssetClass, inst.AssetClass}, [2]any{r.SubType, inst.SubType},
			[2]any{r.PriceType, priceType}, [2]any{r.Currency, currency})
		if spec < 0 {
			continue
		}
		code := strings.ToUpper(r.SourceCd)
		if b, ok := best[code]; ok && b.spec >= spec {
			continue
		}
		best[code] = pick{spec, r.Priority, r.Fallback, int(r.MaxStaleMin.Int64)}
	}
	// Rank within the most specific scope first: an asset-class ranking
	// beats the default for every source it names.
	codes := make([]string, 0, len(best))
	for code := range best {
		codes = append(codes, code)
	}
	sort.SliceStable(codes, func(i, j int) bool {
		a, b := best[codes[i]], best[codes[j]]
		if a.fallback != b.fallback {
			return !a.fallback
		}
		if a.spec != b.spec {
			return a.spec > b.spec
		}
		if a.priority != b.priority {
			return a.priority < b.priority
		}
		return codes[i] < codes[j]
	})
	maxStale = map[string]int{}
	for _, code := range codes {
		maxStale[code] = best[code].stale
	}
	return codes, maxStale
}

// threshold is the most specific variance threshold for a key (nil: none).
func (c *seriesConfig) threshold(inst instrumentFacts, priceType string) *thresholdRow {
	var best *thresholdRow
	bestSpec := -1
	for i := range c.thresholds {
		t := &c.thresholds[i]
		spec := scopeMatch([2]any{t.AssetClass, inst.AssetClass}, [2]any{t.SubType, inst.SubType}, [2]any{t.PriceType, priceType})
		if spec > bestSpec {
			best, bestSpec = t, spec
		}
	}
	return best
}

func (t *thresholdRow) data() map[string]any {
	if t == nil {
		return map[string]any{}
	}
	f := func(v sql.NullFloat64) any {
		if !v.Valid {
			return nil
		}
		return v.Float64
	}
	return map[string]any{"type": t.Type, "warning": f(t.Warning), "error": f(t.Error), "critical": f(t.Critical)}
}

// level is the threshold a percentage breaches: CRITICAL, ERROR, WARNING or "".
func (t *thresholdRow) level(pct float64) string {
	if t == nil {
		return ""
	}
	switch {
	case t.Critical.Valid && pct > t.Critical.Float64:
		return "CRITICAL"
	case t.Error.Valid && pct > t.Error.Float64:
		return "ERROR"
	case t.Warning.Valid && pct > t.Warning.Float64:
		return "WARNING"
	}
	return ""
}

// seriesBinding is how the vendor's rows become observations.
type seriesBinding struct {
	sourceKey string
	date      string
	currency  string
	asOf      string
	values    map[string]string // price type -> column (value:<TYPE>)
	value     string            // one row per price: value column ...
	typeCol   string            // ... and its price type column
	ids       []idCol           // in resolution order
}

type idCol struct{ Type, Column string }

func (r *runner) seriesBinding(binding map[string]string, stagingCols map[string]string) (*seriesBinding, error) {
	sr := r.p.Settings.Series
	b := &seriesBinding{values: map[string]string{}, sourceKey: binding[stagingbind.SourceKey], asOf: binding[stagingbind.AsOfKey],
		date: binding[sr.DateField], currency: binding[sr.CurrencyField]}
	if sr.ValueField != "" && sr.TypeField != "" {
		b.value, b.typeCol = binding[sr.ValueField], binding[sr.TypeField]
	}
	if b.value == "" || b.typeCol == "" {
		b.value, b.typeCol = "", ""
	}
	idAt := map[string]string{}
	for k, col := range binding {
		if t, ok := stagingbind.ValueType(k); ok {
			b.values[t] = col
		}
		if t, ok := stagingbind.IdentifierType(k); ok {
			idAt[t] = col
		}
	}
	for _, t := range sr.IdentifierOrder {
		if col, ok := idAt[t]; ok {
			b.ids = append(b.ids, idCol{t, col})
			delete(idAt, t)
		}
	}
	rest := make([]string, 0, len(idAt))
	for t := range idAt {
		rest = append(rest, t)
	}
	sort.Strings(rest)
	for _, t := range rest {
		b.ids = append(b.ids, idCol{t, idAt[t]})
	}

	if b.sourceKey == "" {
		return nil, msgNoSourceKey(r.req.StagingTable)
	}
	if b.date == "" {
		return nil, msgNoValuationDate(r.req.StagingTable, sr.DateField)
	}
	if len(b.values) == 0 && b.value == "" {
		field := sr.ValueField
		if field == "" {
			field = "value:<PRICE TYPE>"
		}
		return nil, msgNoPriceValues(r.req.StagingTable, field)
	}
	// Every bound column must be a column of the staging table (and a
	// plain name: they are quoted into SQL).
	check := []string{b.sourceKey, b.date, b.currency, b.asOf, b.value, b.typeCol}
	for _, c := range b.values {
		check = append(check, c)
	}
	for _, c := range b.ids {
		check = append(check, c.Column)
	}
	for _, c := range check {
		if c == "" {
			continue
		}
		if _, ok := stagingCols[c]; !ok || !ident.MatchString(c) {
			return nil, msgNoBinding(r.req.StagingTable, r.p.BOKey)
		}
	}
	return b, nil
}

// executeSeries masters a load of a time-series profile.
func (r *runner) executeSeries() error {
	ctx, tx := r.ctx, r.tx
	var srcCd string
	err := tx.GetContext(ctx, &srcCd, `SELECT source_system_cd FROM staging._load_run WHERE id::text = $1 AND tenant_id::text = $2`, r.req.LoadRunID, r.tenant)
	if errors.Is(err, sql.ErrNoRows) {
		return msgNoLoadRun(r.req.LoadRunID)
	}
	if err != nil {
		return err
	}
	r.sourceCd = strings.ToUpper(srcCd)
	r.sourceID = r.cfg.sources[r.sourceCd]
	if r.sourceID == "" {
		return msgUnknownSource(srcCd)
	}
	binding, err := r.e.Bindings.StagingFields(ctx, r.tenant, r.p.BOKey, r.req.StagingTable)
	if err != nil {
		return err
	}
	if binding == nil {
		return msgNoBinding(r.req.StagingTable, r.p.BOKey)
	}
	stagingCols, err := columns(ctx, tx, r.req.StagingTable)
	if err != nil {
		return err
	}
	b, err := r.seriesBinding(binding, stagingCols)
	if err != nil {
		return err
	}
	if err := r.checkPriceTypes(b); err != nil {
		return err
	}
	if r.selRules, err = loadSelectionRules(ctx, r.e.Rules, r.tenant, r.p.BOKey); err != nil {
		return err
	}

	*r.stage = "CANONICALIZE"
	if err := r.canonicalizeSeries(b); err != nil {
		return fmt.Errorf("canonicalize: %w", err)
	}
	// Resolving and observing are the price master's linking stage (the run
	// table's stages are CANONICALIZE, MATCH, SURVIVE, PUBLISH, DONE).
	*r.stage = "MATCH"
	if err := r.observe(); err != nil {
		return fmt.Errorf("observe: %w", err)
	}
	*r.stage = "PUBLISH"
	keys, err := r.touchedKeys()
	if err != nil {
		return err
	}
	for i := 0; i < len(keys); i += seriesChunk {
		end := i + seriesChunk
		if end > len(keys) {
			end = len(keys)
		}
		if err := r.publishSeries(keys[i:end]); err != nil {
			return fmt.Errorf("publish: %w", err)
		}
	}
	return nil
}

// checkPriceTypes: the value:<TYPE> keys name price types that exist.
func (r *runner) checkPriceTypes(b *seriesBinding) error {
	if len(b.values) == 0 {
		return nil
	}
	want := make([]string, 0, len(b.values))
	for t := range b.values {
		want = append(want, t)
	}
	sort.Strings(want)
	var have []string
	if err := r.tx.SelectContext(r.ctx, &have, `SELECT price_type_cd FROM mdm.price_type WHERE is_active AND price_type_cd = ANY($1)`, pq.Array(want)); err != nil {
		return err
	}
	known := map[string]bool{}
	for _, h := range have {
		known[h] = true
	}
	var missing []string
	for _, t := range want {
		if !known[t] {
			missing = append(missing, t)
		}
	}
	if len(missing) > 0 {
		return msgUnknownPriceTypes(strings.Join(missing, ", "))
	}
	return nil
}

// canonicalizeSeries writes the load's observations to the incoming table,
// resolved to instruments, in one statement.
func (r *runner) canonicalizeSeries(b *seriesBinding) error {
	ctx, tx := r.ctx, r.tx
	sr := r.p.Settings.Series
	inst := r.cfg.series.instrument
	id := inst.ident()
	text := func(col string) string {
		if col == "" {
			return "NULL::text"
		}
		return fmt.Sprintf("NULLIF(trim(s.%s::text), '')", qi(col))
	}
	idPairs := make([]string, 0, len(b.ids))
	order := make([]string, 0, len(b.ids))
	for _, c := range b.ids {
		idPairs = append(idPairs, fmt.Sprintf("%s, upper(%s)", pq.QuoteLiteral(c.Type), text(c.Column)))
		order = append(order, c.Type)
	}
	ids := "'{}'::jsonb"
	if len(idPairs) > 0 {
		ids = "jsonb_strip_nulls(jsonb_build_object(" + strings.Join(idPairs, ", ") + "))"
	}
	asOf := "NULL::timestamptz"
	if b.asOf != "" {
		asOf = fmt.Sprintf("s.%s::timestamptz", qi(b.asOf))
	}
	// Observations: one per bound price column, or the row's own type.
	var obs, longCols string
	if len(b.values) > 0 {
		types := make([]string, 0, len(b.values))
		for t := range b.values {
			types = append(types, t)
		}
		sort.Strings(types)
		vals := make([]string, 0, len(types))
		for i, t := range types {
			// Carried through src as v<i>, typed once.
			longCols += fmt.Sprintf(", %s::numeric AS v%d", text(b.values[t]), i)
			vals = append(vals, fmt.Sprintf("(%s, src.v%d)", pq.QuoteLiteral(t), i))
		}
		obs = fmt.Sprintf(`SELECT src.*, v.ptype, v.val FROM src CROSS JOIN LATERAL (VALUES %s) v(ptype, val) WHERE v.val IS NOT NULL`, strings.Join(vals, ", "))
	} else {
		obs = `SELECT src.*, COALESCE(m.internal_price_type_cd, upper(src.vendor_type)) AS ptype, src.vendor_value AS val FROM src
			LEFT JOIN LATERAL (SELECT internal_price_type_cd FROM mdm.price_type_mapping m
				WHERE m.is_active AND m.source_system_id::text = $4 AND upper(m.vendor_type_cd) = upper(src.vendor_type) LIMIT 1) m ON true
			WHERE src.vendor_value IS NOT NULL`
	}
	if b.value != "" {
		longCols += fmt.Sprintf(", %s AS vendor_type, %s::numeric AS vendor_value", text(b.typeCol), text(b.value))
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM staging.price_incoming WHERE load_run_id::text = $1 AND source_system_id::text = $2`,
		r.req.LoadRunID, r.sourceID); err != nil {
		return err
	}
	q := fmt.Sprintf(`WITH src AS (
			SELECT s._source_row_num AS rn, %[1]s AS skey, %[2]s::date AS pdate, upper(%[3]s) AS ccy, %[4]s AS as_of, %[5]s AS ids%[6]s
			FROM %[7]s s WHERE s._load_run_id::text = $1 AND s.tenant_id::text = $2
		), obs AS (%[8]s
		), res AS (
			SELECT obs.*, r.entity_id, r.via, pt.id AS type_id, pt.is_official
			FROM obs
			LEFT JOIN mdm.price_type pt ON pt.price_type_cd = obs.ptype AND pt.is_active
			LEFT JOIN LATERAL (
				SELECT i.%[9]s::uuid AS entity_id, k.t AS via
				FROM jsonb_each_text(obs.ids) k(t, v)
				JOIN %[10]s i ON i.%[11]s = k.t AND i.%[12]s = k.v AND %[13]s
				ORDER BY array_position($3::text[], k.t), k.t LIMIT 1) r ON true
		), errs AS (
			SELECT res.*, (SELECT COALESCE(jsonb_agg(e), '[]'::jsonb) FROM unnest(ARRAY[
					CASE WHEN res.pdate IS NULL THEN 'NO_VALUATION_DATE' END,
					CASE WHEN res.type_id IS NULL THEN 'UNKNOWN_PRICE_TYPE' END,
					CASE WHEN res.entity_id IS NULL THEN 'UNRESOLVED_INSTRUMENT' END]) e WHERE e IS NOT NULL) AS errors
			FROM res
		)
		INSERT INTO staging.price_incoming (source_system_id, source_system_cd, source_row_id, load_run_id, tenant_id,
			price_entity_type, price_entity_id, source_entity_ref, price_type_cd, observation_type, price_date, value, currency,
			is_official, canonical_payload, mapping_version, mapped_at, is_valid, validation_errors)
		SELECT DISTINCT ON (key) $4::uuid, $5, key, $1::uuid, $2::uuid, $6, entity_id,
			(SELECT k.t || ':' || k.v FROM jsonb_each_text(ids) k(t, v) ORDER BY array_position($3::text[], k.t), k.t LIMIT 1),
			COALESCE(ptype, '?'), $7, COALESCE(pdate, CURRENT_DATE), val, ccy, COALESCE(is_official, false),
			jsonb_build_object('ids', ids, 'source_key', skey, 'row', rn, 'as_of', as_of, 'resolved_by', via),
			1, now(), errors = '[]'::jsonb, errors
		FROM (SELECT errs.*, COALESCE(skey, 'row:' || rn) || '|' || COALESCE(pdate::text, '') || '|' || COALESCE(ptype, '?') AS key FROM errs) x
		ORDER BY key, rn DESC
		ON CONFLICT (tenant_id, source_system_id, source_row_id) DO UPDATE SET
			load_run_id = EXCLUDED.load_run_id, price_entity_id = EXCLUDED.price_entity_id, source_entity_ref = EXCLUDED.source_entity_ref,
			price_type_cd = EXCLUDED.price_type_cd, price_date = EXCLUDED.price_date, value = EXCLUDED.value, currency = EXCLUDED.currency,
			is_official = EXCLUDED.is_official, canonical_payload = EXCLUDED.canonical_payload, mapped_at = EXCLUDED.mapped_at,
			is_valid = EXCLUDED.is_valid, validation_errors = EXCLUDED.validation_errors`,
		text(b.sourceKey), text(b.date), text(b.currency), asOf, ids, longCols, qi(r.req.StagingTable), obs,
		qi(id.KeyColumn), qi(*inst.IdentifierTable), qi(id.TypeColumn), qi(id.ValueColumn), id.identActive("i"))
	if _, err := tx.ExecContext(ctx, q, r.req.LoadRunID, r.tenant, pq.Array(order), r.sourceID, r.sourceCd, sr.EntityType, sr.ObservationType); err != nil {
		return err
	}

	var n struct {
		Records int `db:"records"`
		Valid   int `db:"valid"`
	}
	if err := tx.GetContext(ctx, &n, `SELECT count(*) AS records, count(*) FILTER (WHERE is_valid) AS valid
		FROM staging.price_incoming WHERE load_run_id::text = $1 AND source_system_id::text = $2`, r.req.LoadRunID, r.sourceID); err != nil {
		return err
	}
	r.counts.Records, r.counts.Valid, r.counts.Invalid = n.Records, n.Valid, n.Records-n.Valid

	// Each invalid observation is an exception (once while it is open).
	res, err := tx.ExecContext(ctx, `INSERT INTO mdm.price_exception (tenant_id, price_entity_type, price_entity_id, price_date, source_id,
			exception_type, severity, exception_description, detected_at, status, custom_attributes)
		SELECT i.tenant_id, i.price_entity_type, i.price_entity_id, i.price_date, i.source_system_id, e, 'ERROR',
			CASE e WHEN 'UNRESOLVED_INSTRUMENT' THEN 'No golden instrument has ' || COALESCE((SELECT string_agg(k.t || ' ' || k.v, ', ' ORDER BY k.t)
					FROM jsonb_each_text(i.canonical_payload->'ids') k(t, v)), 'no identifier on this row')
			       WHEN 'UNKNOWN_PRICE_TYPE' THEN 'Price type ' || i.price_type_cd || ' is not defined'
			       ELSE 'The row has no valuation date' END || ' (' || i.source_system_cd || ' ' || i.source_row_id || ')',
			now(), 'OPEN',
			jsonb_build_object('run_id', $3::text, 'load_run_id', i.load_run_id, 'source_row_id', i.source_row_id,
				'source_entity_ref', i.source_entity_ref, 'price_type', i.price_type_cd)
		FROM staging.price_incoming i CROSS JOIN LATERAL jsonb_array_elements_text(i.validation_errors) e
		WHERE i.load_run_id::text = $1 AND i.source_system_id::text = $2 AND NOT i.is_valid
		  AND NOT EXISTS (SELECT 1 FROM mdm.price_exception x WHERE x.status = 'OPEN' AND x.exception_type = e
		                    AND x.source_id = i.source_system_id AND x.custom_attributes->>'source_row_id' = i.source_row_id)`,
		r.req.LoadRunID, r.sourceID, r.run.ID)
	if err != nil {
		return err
	}
	added, _ := res.RowsAffected()
	r.counts.Exceptions += int(added)
	return nil
}

// incomingKeys selects this load's valid observations, one per key.
const incomingKeys = `SELECT DISTINCT ON (i.price_entity_type, i.price_entity_id, pt.id, i.price_date)
		i.*, pt.id AS type_id, pt.is_executable, (pt.category = 'INDICATIVE') AS is_indicative
	FROM staging.price_incoming i JOIN mdm.price_type pt ON pt.price_type_cd = i.price_type_cd AND pt.is_active
	WHERE i.load_run_id::text = $1 AND i.source_system_id::text = $2 AND i.is_valid
	ORDER BY i.price_entity_type, i.price_entity_id, pt.id, i.price_date, (i.canonical_payload->>'row')::int DESC`

// observe records the load's observations in mdm.price: new ones are
// inserted; a changed value for a key the source already reported is a
// restatement - the current row is closed (and snapshotted to history) and
// the new one points back at it.
func (r *runner) observe() error {
	ctx, tx := r.ctx, r.tx
	if _, err := tx.ExecContext(ctx, `WITH inc AS (`+incomingKeys+`),
		old AS (SELECT p.* FROM mdm.price p JOIN inc ON p.price_entity_type = inc.price_entity_type AND p.price_entity_id = inc.price_entity_id
				AND p.price_type_id = inc.type_id AND p.source_id = inc.source_system_id AND p.price_date = inc.price_date
			WHERE p.is_current AND p.tenant_id = inc.tenant_id
			  AND (p.value IS DISTINCT FROM inc.value OR p.currency IS DISTINCT FROM inc.currency)),
		hist AS (INSERT INTO mdm.price_history (tenant_id, price_id, price_entity_type, price_entity_id, price_type_id, source_id, version_num,
				valid_from, valid_to, is_current, record_snapshot, changed_columns, change_source, change_reason, custom_attributes)
			SELECT old.tenant_id, old.id, old.price_entity_type, old.price_entity_id, old.price_type_id, old.source_id,
				COALESCE((SELECT max(h.version_num) FROM mdm.price_history h WHERE h.price_id = old.id), 0) + 1,
				COALESCE(old.created_at, now()), now(), false, to_jsonb(old), ARRAY['value', 'currency'], $3, 'RESTATED',
				jsonb_build_object('run_id', $4::text)
			FROM old RETURNING 1)
		UPDATE mdm.price p SET is_current = false, updated_at = now() FROM old WHERE p.id = old.id`,
		r.req.LoadRunID, r.sourceID, r.sourceCd, r.run.ID); err != nil {
		return err
	}
	var out []bool
	if err := tx.SelectContext(ctx, &out, `WITH inc AS (`+incomingKeys+`)
		INSERT INTO mdm.price (tenant_id, price_entity_type, price_entity_id, price_type_id, source_id, price_date, price_time, value, currency,
			is_official, is_executable, is_stale, is_indicative, is_adjusted, source_timestamp, source_system_id, predecessor_price_id,
			is_current, custom_attributes)
		SELECT inc.tenant_id, inc.price_entity_type, inc.price_entity_id, inc.type_id, inc.source_system_id, inc.price_date,
			(inc.canonical_payload->>'as_of')::timestamptz, inc.value, inc.currency, inc.is_official, inc.is_executable, false,
			inc.is_indicative, false, (inc.canonical_payload->>'as_of')::timestamptz, inc.source_system_id,
			(SELECT p.id FROM mdm.price p WHERE p.price_entity_type = inc.price_entity_type AND p.price_entity_id = inc.price_entity_id
				AND p.price_type_id = inc.type_id AND p.source_id = inc.source_system_id AND p.price_date = inc.price_date
				AND NOT p.is_current ORDER BY p.updated_at DESC NULLS LAST LIMIT 1),
			true, jsonb_build_object('run_id', $3::text, 'load_run_id', inc.load_run_id, 'source_row_id', inc.source_row_id,
				'source_entity_ref', inc.source_entity_ref)
		FROM inc
		WHERE NOT EXISTS (SELECT 1 FROM mdm.price p WHERE p.price_entity_type = inc.price_entity_type AND p.price_entity_id = inc.price_entity_id
			AND p.price_type_id = inc.type_id AND p.source_id = inc.source_system_id AND p.price_date = inc.price_date AND p.is_current)
		RETURNING predecessor_price_id IS NOT NULL`, r.req.LoadRunID, r.sourceID, r.run.ID); err != nil {
		return err
	}
	for _, restated := range out {
		if restated {
			r.counts.Restated++
		} else {
			r.counts.New++
		}
	}
	return nil
}

// touchedKeys are the golden prices this load can change.
func (r *runner) touchedKeys() ([]seriesKey, error) {
	var rows []struct {
		EntityType string `db:"price_entity_type"`
		EntityID   string `db:"price_entity_id"`
		PriceType  string `db:"price_type_cd"`
		Date       string `db:"price_date"`
	}
	if err := r.tx.SelectContext(r.ctx, &rows, `SELECT DISTINCT price_entity_type, price_entity_id::text, price_type_cd, price_date::text
		FROM staging.price_incoming WHERE load_run_id::text = $1 AND source_system_id::text = $2 AND is_valid
		ORDER BY 2, 3, 4`, r.req.LoadRunID, r.sourceID); err != nil {
		return nil, err
	}
	keys := make([]seriesKey, len(rows))
	for i, x := range rows {
		keys[i] = seriesKey{x.EntityType, x.EntityID, x.PriceType, x.Date}
	}
	return keys, nil
}

// seriesCandidate is one source's current observation for a key.
type seriesCandidate struct {
	Key        string          `db:"k"`
	SourceID   string          `db:"source_id"`
	SourceKey  string          `db:"source_key"`
	Value      float64         `db:"value"`
	Currency   sql.NullString  `db:"currency"`
	AsOf       sql.NullTime    `db:"as_of"`
	IsOfficial bool            `db:"is_official"`
	Instrument json.RawMessage `db:"instrument"`
}

// goldenState is a key's latest golden version and the prior date's value.
type goldenState struct {
	Key        string          `db:"k"`
	Version    sql.NullInt64   `db:"golden_version"`
	Value      sql.NullFloat64 `db:"golden_value"`
	Status     sql.NullString  `db:"status"`
	Attributes []byte          `db:"golden_attributes"`
	PriorValue sql.NullFloat64 `db:"prior_value"`
	PriorDate  sql.NullString  `db:"prior_date"`
	// Current is the key's current (published) golden value, if any.
	Current sql.NullFloat64 `db:"current_value"`
	// Recent current golden values before the date, newest first (for the
	// unchanged-price check), and their dates.
	Recent      pq.Float64Array `db:"recent_values"`
	RecentDates pq.StringArray  `db:"recent_dates"`
}

// staleEvent is a stale price (mdm.price_stale_event).
type staleEvent struct {
	EntityType string  `json:"et"`
	EntityID   string  `json:"eid"`
	Date       string  `json:"d"`
	PriceType  string  `json:"pt"`
	SourceID   *string `json:"src"`
	Since      *string `json:"since"`
	Days       int     `json:"days"`
	Kind       string  `json:"kind"` // UNCHANGED | OLD_QUOTE
}

// Rows written in bulk (jsonb_to_recordset).
type goldenPriceRow struct {
	EntityType string          `json:"et"`
	EntityID   string          `json:"eid"`
	Date       string          `json:"d"`
	PriceType  string          `json:"pt"`
	Version    int             `json:"v"`
	Current    bool            `json:"cur"`
	Value      float64         `json:"val"`
	Currency   *string         `json:"ccy"`
	PriceTime  *time.Time      `json:"ptime"`
	Winner     *string         `json:"win"`
	Sources    int             `json:"n"`
	Variance   *float64        `json:"var"`
	Confidence float64         `json:"conf"`
	DQ         float64         `json:"dq"`
	Stale      bool            `json:"stale"`
	Official   bool            `json:"off"`
	Attributes json.RawMessage `json:"attrs"`
	Winners    json.RawMessage `json:"wins"`
	Status     string          `json:"st"`
}

type priceIssueRow struct {
	EntityType string          `json:"et"`
	EntityID   string          `json:"eid"`
	Date       string          `json:"d"`
	SourceID   *string         `json:"src"`
	Type       string          `json:"t"`
	Severity   string          `json:"sev"`
	Message    string          `json:"msg"`
	Attributes json.RawMessage `json:"attrs"`
}

type varianceRow struct {
	EntityType string   `json:"et"`
	EntityID   string   `json:"eid"`
	Date       string   `json:"d"`
	PriceType  string   `json:"pt"`
	SourceA    string   `json:"a"`
	SourceB    string   `json:"b"`
	PriceA     float64  `json:"pa"`
	PriceB     float64  `json:"pb"`
	Abs        float64  `json:"abs"`
	Pct        float64  `json:"pct"`
	Threshold  *string  `json:"th"`
	Severity   string   `json:"sev"`
	Selected   *float64 `json:"sel"`
}

func keysJSON(keys []seriesKey) ([]byte, error) {
	out := make([]map[string]string, len(keys))
	for i, k := range keys {
		out[i] = map[string]string{"et": k.EntityType, "eid": k.EntityID, "pt": k.PriceType, "d": k.Date}
	}
	return json.Marshal(out)
}

// publishSeries survives, controls and publishes a chunk of keys.
func (r *runner) publishSeries(keys []seriesKey) error {
	ctx, tx := r.ctx, r.tx
	inst := r.cfg.series.instrument
	kj, err := keysJSON(keys)
	if err != nil {
		return err
	}
	const keyset = `k AS (SELECT et, eid::uuid AS eid, pt, d::date AS d, eid || '|' || pt || '|' || d AS key
		FROM jsonb_to_recordset($1::jsonb) AS x(et text, eid text, pt text, d text))`

	var cands []seriesCandidate
	if err := tx.SelectContext(ctx, &cands, fmt.Sprintf(`WITH %s
		SELECT k.key AS k, p.source_id::text, COALESCE(p.custom_attributes->>'source_row_id', p.id::text) AS source_key,
			p.value, p.currency, p.source_timestamp AS as_of, p.is_official,
			COALESCE((SELECT jsonb_build_object('asset_class', to_jsonb(a)->>'asset_class', 'sub_asset_class', to_jsonb(a)->>'sub_asset_class',
				'currency', to_jsonb(a)->>'currency', 'code', to_jsonb(a)->>%s, 'name', to_jsonb(a)->>%s)
				FROM %s a WHERE a.%s = k.eid AND %s LIMIT 1), '{}'::jsonb) AS instrument
		FROM k JOIN mdm.price_type pt ON pt.price_type_cd = k.pt
		JOIN mdm.price p ON p.price_entity_type = k.et AND p.price_entity_id = k.eid AND p.price_type_id = pt.id
			AND p.price_date = k.d AND p.is_current
		ORDER BY k.key, p.source_id`, keyset, pq.QuoteLiteral(inst.AnchorCodeColumn), pq.QuoteLiteral(inst.Settings.NameAttribute),
		qi(inst.AnchorTable), qi(inst.entityCol()), inst.current("a")), kj); err != nil {
		return fmt.Errorf("candidates: %w", err)
	}
	var states []goldenState
	if err := tx.SelectContext(ctx, &states, fmt.Sprintf(`WITH %s
		SELECT k.key AS k, g.golden_version, g.golden_value, g.status, g.golden_attributes, pr.golden_value AS prior_value, pr.price_date::text AS prior_date,
			(SELECT c.golden_value FROM mdm.price_golden_record c WHERE c.price_entity_type = k.et AND c.price_entity_id = k.eid
				AND c.price_type_cd = k.pt AND c.price_date = k.d AND c.is_current LIMIT 1) AS current_value,
			rc.recent_values, rc.recent_dates
		FROM k
		LEFT JOIN LATERAL (SELECT array_agg(z.v ORDER BY z.d DESC) AS recent_values, array_agg(z.d::text ORDER BY z.d DESC) AS recent_dates
			FROM (SELECT r.golden_value::float8 AS v, r.price_date AS d FROM mdm.price_golden_record r
				WHERE r.price_entity_type = k.et AND r.price_entity_id = k.eid AND r.price_type_cd = k.pt AND r.is_current
				  AND r.price_date < k.d AND r.price_date >= k.d - $3::int
				ORDER BY r.price_date DESC LIMIT $4) z) rc ON true
		LEFT JOIN LATERAL (SELECT golden_version, golden_value, status, golden_attributes FROM mdm.price_golden_record g
			WHERE g.price_entity_type = k.et AND g.price_entity_id = k.eid AND g.price_type_cd = k.pt AND g.price_date = k.d
			ORDER BY g.golden_version DESC LIMIT 1) g ON true
		LEFT JOIN LATERAL (SELECT golden_value, price_date FROM mdm.price_golden_record p
			WHERE p.price_entity_type = k.et AND p.price_entity_id = k.eid AND p.price_type_cd = k.pt AND p.price_date < k.d
			  AND p.price_date >= k.d - $2::int AND p.is_current
			ORDER BY p.price_date DESC LIMIT 1) pr ON true`, keyset), kj, r.p.Settings.Series.Controls.maxGapDays(),
		3*max(r.p.Settings.Series.Controls.unchangedDays(), 1)+7, max(r.p.Settings.Series.Controls.unchangedDays(), 1)); err != nil {
		return fmt.Errorf("golden state: %w", err)
	}
	byKey := map[string][]seriesCandidate{}
	for _, c := range cands {
		byKey[c.Key] = append(byKey[c.Key], c)
	}
	state := map[string]goldenState{}
	for _, s := range states {
		state[s.Key] = s
	}
	codeOf := map[string]string{}
	for code, id := range r.cfg.sources {
		codeOf[id] = code
	}

	overrides, err := r.priceOverrides(keys)
	if err != nil {
		return fmt.Errorf("overrides: %w", err)
	}
	// Current golden prices of the chunk's instruments and dates, for the
	// related-types check (updated as this chunk publishes).
	var sib []struct {
		K     string  `db:"k"`
		Value float64 `db:"golden_value"`
	}
	if err := tx.SelectContext(ctx, &sib, `WITH `+keyset+`
		SELECT DISTINCT g.price_entity_id::text || '|' || g.price_type_cd || '|' || g.price_date::text AS k, g.golden_value::float8 AS golden_value
		FROM (SELECT DISTINCT et, eid, d FROM k) x JOIN mdm.price_golden_record g
		  ON g.price_entity_type = x.et AND g.price_entity_id = x.eid AND g.price_date = x.d AND g.is_current`, kj); err != nil {
		return fmt.Errorf("related prices: %w", err)
	}
	r.siblings = make(map[string]float64, len(sib))
	for _, x := range sib {
		r.siblings[x.K] = x.Value
	}
	var golden []goldenPriceRow
	var issues []priceIssueRow
	var variances []varianceRow
	var changed []seriesKey
	for _, k := range keys {
		var ov *activeOverride
		if o, ok := overrides[k.String()]; ok {
			ov = &o
		}
		st := state[k.String()]
		g, is, vs := r.surviveKey(k, byKey[k.String()], st, codeOf, ov)
		if g != nil {
			golden = append(golden, *g)
			if g.Current {
				r.siblings[k.String()] = g.Value
				// The key's current value moved (or it has one for the first
				// time): later dates were checked against the old one.
				if !st.Current.Valid || st.Current.Float64 != g.Value {
					changed = append(changed, k)
				}
			}
		}
		issues = append(issues, is...)
		variances = append(variances, vs...)
	}
	if err := r.writeSeries(golden, issues, variances); err != nil {
		return err
	}
	if err := r.writeStale(); err != nil {
		return err
	}
	return r.cascade(changed)
}

// cascade re-checks the next priced date of each key whose current value
// changed: its day-over-day control was measured against the old value. A
// re-checked price keeps its value (a new version records the new prior
// and control outcome); if its own current value changes - a steward's
// decision, a vendor restatement - the date after is re-checked in turn.
func (r *runner) cascade(changed []seriesKey) error {
	if len(changed) == 0 {
		return nil
	}
	if r.cascadeDepth >= maxCascade {
		return nil
	}
	kj, err := keysJSON(changed)
	if err != nil {
		return err
	}
	var next []struct {
		EntityType string `db:"price_entity_type"`
		EntityID   string `db:"price_entity_id"`
		PriceType  string `db:"price_type_cd"`
		Date       string `db:"price_date"`
	}
	if err := r.tx.SelectContext(r.ctx, &next, `SELECT DISTINCT n.price_entity_type, n.price_entity_id::text, n.price_type_cd, n.price_date::text
		FROM jsonb_to_recordset($1::jsonb) AS x(et text, eid text, pt text, d text)
		CROSS JOIN LATERAL (SELECT g.price_entity_type, g.price_entity_id, g.price_type_cd, g.price_date FROM mdm.price_golden_record g
			WHERE g.price_entity_type = x.et AND g.price_entity_id = x.eid::uuid AND g.price_type_cd = x.pt AND g.price_date > x.d::date
			  AND g.price_date <= x.d::date + $2::int
			ORDER BY g.price_date LIMIT 1) n`, kj, r.p.Settings.Series.Controls.maxGapDays()); err != nil {
		return fmt.Errorf("cascade: %w", err)
	}
	if len(next) == 0 {
		return nil
	}
	keys := make([]seriesKey, len(next))
	for i, n := range next {
		keys[i] = seriesKey{n.EntityType, n.EntityID, n.PriceType, n.Date}
	}
	// A re-check is judged on its own: no forced version.
	force := r.force
	r.force = false
	r.cascadeDepth++
	defer func() { r.force = force; r.cascadeDepth-- }()
	r.counts.Rechecked += len(keys)
	for i := 0; i < len(keys); i += seriesChunk {
		end := min(i+seriesChunk, len(keys))
		if err := r.publishSeries(keys[i:end]); err != nil {
			return err
		}
	}
	return nil
}

// maxCascade bounds how many dates forward one change re-checks.
const maxCascade = 366

// surviveKey decides one golden price. It returns the new golden version
// (nil: unchanged, or nothing to publish), exceptions and variance events.
func (r *runner) surviveKey(k seriesKey, cands []seriesCandidate, st goldenState, codeOf map[string]string, ov *activeOverride) (*goldenPriceRow, []priceIssueRow, []varianceRow) {
	sc := r.cfg.series
	ctl := r.p.Settings.Series.Controls
	var issues []priceIssueRow
	issue := func(src *string, typ, sev, msg string) {
		if ov != nil {
			return // a steward has decided this price
		}
		attrs, _ := json.Marshal(map[string]any{"price_type": k.PriceType, "run_id": r.run.ID})
		issues = append(issues, priceIssueRow{k.EntityType, k.EntityID, k.Date, src, typ, sev, msg, attrs})
		r.raised = append(r.raised, Issue{Code: typ, Severity: sev, Attribute: k.PriceType, Message: msg}) // sev is SevError or SevWarning
	}
	if len(cands) == 0 {
		return nil, nil, nil
	}
	var facts instrumentFacts
	var raw map[string]string
	_ = json.Unmarshal(cands[0].Instrument, &raw)
	facts = instrumentFacts{AssetClass: raw["asset_class"], SubType: raw["sub_asset_class"], Currency: strings.ToUpper(raw["currency"])}
	// How messages name the price: the instrument's code and name.
	label := k.EntityID
	if raw["code"] != "" {
		label = strings.TrimSpace(raw["code"] + " " + raw["name"])
	}
	label = fmt.Sprintf("%s %s %s", label, k.PriceType, k.Date)

	// End of the valuation date: what staleness and age are measured against.
	valDate, _ := time.Parse("2006-01-02", k.Date)
	eod := valDate.Add(24 * time.Hour)
	ranking, maxStale := sc.ranking(facts, k.PriceType, facts.Currency)
	th := sc.threshold(facts, k.PriceType)

	type provCandidate struct {
		Source   string   `json:"source"`
		Value    float64  `json:"value"`
		Currency string   `json:"currency,omitempty"`
		AsOf     string   `json:"as_of,omitempty"`
		Rank     int      `json:"rank,omitempty"`
		Stale    bool     `json:"stale,omitempty"`
		Selected *bool    `json:"selected,omitempty"`
		Note     string   `json:"note,omitempty"`
		Excluded string   `json:"excluded,omitempty"`
		DiffPct  *float64 `json:"diff_pct,omitempty"`
	}
	var prov []provCandidate
	var contribs []Contribution
	info := map[string]seriesCandidate{}
	var manual *seriesCandidate
	for i, c := range cands {
		code := codeOf[c.SourceID]
		if code == "" {
			code = c.SourceID
		}
		if code == manualSource {
			// The steward's price: it wins through the override, it doesn't compete.
			manual = &cands[i]
			continue
		}
		pc := provCandidate{Source: code, Value: c.Value, Currency: c.Currency.String, Rank: rankOf(ranking, code)}
		if c.AsOf.Valid {
			pc.AsOf = c.AsOf.Time.UTC().Format(time.RFC3339)
		}
		ccy := strings.ToUpper(c.Currency.String)
		if facts.Currency != "" && ccy != "" && ccy != facts.Currency {
			msg := fmt.Sprintf("%s: %s quoted in %s, the instrument is in %s", label, code, ccy, facts.Currency)
			src := c.SourceID
			issue(&src, "CURRENCY_MISMATCH", "WARNING", msg)
			if !strings.EqualFold(ctl.CurrencyMismatch, "FLAG") {
				pc.Excluded = "currency " + ccy
				prov = append(prov, pc)
				continue
			}
		}
		var asOf time.Time
		stale := false
		if c.AsOf.Valid {
			asOf = c.AsOf.Time
			if m := maxStale[code]; m > 0 && eod.Sub(asOf) > time.Duration(m)*time.Minute {
				stale = true
			}
		}
		contribs = append(contribs, Contribution{SourceID: c.SourceID, SourceCd: code, SourceKey: c.SourceKey, AsOf: asOf,
			Attrs: map[string]any{"value": c.Value}, Stale: stale})
		info[code] = c
		prov = append(prov, pc)
	}

	rule, ok := r.cfg.survival["value"]
	if !ok {
		rule = SurvivalRule{Attribute: "value", Strategy: "SOURCE_PRIORITY"}
	}
	rule.TolerancePct = 0 // day-over-day moves are the thresholds' job, below
	var previous map[string]any
	if st.PriorValue.Valid {
		previous = map[string]any{"value": st.PriorValue.Float64}
	}
	decisions, survIssues := Survive(contribs, map[string]SurvivalRule{"value": rule}, previous, eod,
		&SurviveOptions{Hierarchy: func(string) []string { return ranking }, Select: selector(r.selRules),
			Context: map[string]any{"threshold": th.data(), "price_type": k.PriceType, "asset_class": facts.AssetClass}})
	d, decided := decisions["value"]
	if ov != nil {
		var v float64
		if manual != nil {
			v = manual.Value
		} else if x, ok := priceValue(ov.Value); ok {
			v = x
		}
		d = Decision{Attribute: "value", Value: v, Strategy: "OVERRIDE", Reason: overrideReason(*ov), Confidence: 1, Competing: d.Competing,
			Winner: Candidate{SourceCd: manualSource, SourceKey: "override:" + ov.ID, Value: v}}
		decided = true
		info[manualSource] = seriesCandidate{SourceID: r.cfg.sources[manualSource], Value: v}
		if manual != nil {
			info[manualSource] = *manual
		}
		prov = append(prov, provCandidate{Source: manualSource, Value: v, Note: "steward override " + ov.ID})
	}
	status := "PUBLISHED"
	for _, is := range survIssues {
		sev := "WARNING"
		if is.Severity == SevError {
			sev = "ERROR"
		}
		issue(nil, is.Code, sev, fmt.Sprintf("%s: %s", label, is.Message))
		if ov == nil && (is.Code == IssueSelectionHold || is.Code == IssueAnomaly || is.Severity == SevError) {
			status = "REVIEW"
		}
	}
	if !decided || d.Value == nil {
		// Every quote excluded, or held with no prior price to keep.
		r.counts.HeldForReview++
		return nil, issues, nil
	}
	value, _ := number(d.Value)
	winner := d.Winner.SourceCd
	for i := range prov {
		for _, c := range d.Competing {
			if c.SourceCd == prov[i].Source {
				prov[i].Stale, prov[i].Selected, prov[i].Note = c.Stale, c.Selected, c.Note
			}
		}
	}

	// Controls.
	var controls []map[string]any
	var changePct *float64
	if st.PriorValue.Valid && st.PriorValue.Float64 != 0 {
		pct := math.Abs(value-st.PriorValue.Float64) / math.Abs(st.PriorValue.Float64) * 100
		changePct = &pct
		if lvl := th.level(pct); lvl != "" {
			action := "FLAG"
			switch lvl {
			case "ERROR":
				if strings.EqualFold(ctl.DayOverDay.Error, "HOLD") {
					action = "HOLD"
				}
			case "CRITICAL":
				action = "HOLD"
				if strings.EqualFold(ctl.DayOverDay.Critical, "FLAG") {
					action = "FLAG"
				}
			}
			sev := "WARNING"
			if lvl != "WARNING" {
				sev = "ERROR"
			}
			msg := fmt.Sprintf("%s: %.4g moved %.2f%% from %.4g on %s (%s threshold)", label,
				value, pct, st.PriorValue.Float64, st.PriorDate.String, strings.ToLower(lvl))
			if ov != nil {
				action = "OVERRIDDEN"
			}
			if action == "HOLD" {
				status = "REVIEW"
				msg += "; held for a steward"
			}
			issue(nil, "DAY_OVER_DAY", sev, msg)
			controls = append(controls, map[string]any{"control": "DAY_OVER_DAY", "level": lvl, "action": action, "pct": round4(pct)})
		}
	}
	var variances []varianceRow
	var maxVar *float64
	winnerID := ""
	if w, ok := info[winner]; ok {
		winnerID = w.SourceID
	}
	for i, pc := range prov {
		if pc.Excluded != "" || pc.Source == winner || value == 0 {
			continue
		}
		pct := math.Abs(pc.Value-value) / math.Abs(value) * 100
		p := round4(pct)
		prov[i].DiffPct = &p
		if maxVar == nil || pct > *maxVar {
			v := pct
			maxVar = &v
		}
		if lvl := th.level(pct); ctl.CrossSource && lvl != "" && winnerID != "" {
			var thID *string
			if th != nil {
				thID = &th.ID
			}
			sel := value
			variances = append(variances, varianceRow{k.EntityType, k.EntityID, k.Date, k.PriceType, winnerID, info[pc.Source].SourceID,
				value, pc.Value, math.Abs(pc.Value - value), round4(pct), thID, lvl, &sel})
		}
	}

	// Stale: the winning quote older than its source allows, or the golden
	// value unchanged for N valuation dates in a row (a price nobody is
	// really re-marking).
	if ov == nil && !d.Held {
		staleAction := strings.ToUpper(ctl.Stale.Action)
		var srcID *string
		if w, ok := info[winner]; ok {
			id := w.SourceID
			srcID = &id
		}
		flag := func(kind, msg string, days int, since *string) {
			if staleAction == "HOLD" {
				status = "REVIEW"
				msg += "; held for a steward"
			}
			issue(srcID, "STALE_PRICE", "WARNING", msg)
			controls = append(controls, map[string]any{"control": "STALE_PRICE", "level": "WARNING", "action": map[bool]string{true: "HOLD", false: "FLAG"}[staleAction == "HOLD"], "kind": kind, "days": days})
			r.staleEvents = append(r.staleEvents, staleEvent{k.EntityType, k.EntityID, k.Date, k.PriceType, srcID, since, days, kind})
		}
		if d.Winner.Stale && !d.Winner.AsOf.IsZero() {
			days := int(eod.Sub(d.Winner.AsOf).Hours() / 24)
			since := d.Winner.AsOf.UTC().Format("2006-01-02")
			flag("OLD_QUOTE", fmt.Sprintf("%s: the winning quote from %s is from %s, older than it is allowed to be", label, winner, since), days, &since)
		}
		if n := ctl.unchangedDays(); n > 1 {
			run := 1
			for _, v := range st.Recent {
				if v != value {
					break
				}
				run++
			}
			if run >= n {
				since := st.RecentDates[run-2]
				flag("UNCHANGED", fmt.Sprintf("%s: %.6g unchanged for %d valuation dates, since %s", label, value, run, since), run, &since)
			}
		}
	}
	// Related types: a price only one source quotes has no peers to agree
	// with, so it is checked against the golden prices of the types that
	// should agree with it (a lone official close far from the last price).
	if len(contribs) == 1 && value != 0 {
		for _, rt := range ctl.related(k.PriceType) {
			other, ok := r.siblings[k.EntityID+"|"+rt+"|"+k.Date]
			if !ok || other == 0 {
				continue
			}
			pct := math.Abs(value-other) / math.Abs(other) * 100
			lvl := th.level(pct)
			if lvl == "" || lvl == "WARNING" {
				continue
			}
			action := "FLAG"
			if lvl == "CRITICAL" {
				action = "HOLD"
			}
			if ov != nil {
				action = "OVERRIDDEN"
			}
			msg := fmt.Sprintf("%s: only %s quotes it, at %.4g - %.2f%% from the golden %s %.4g (%s threshold)", label, winner, value, pct, rt, other, strings.ToLower(lvl))
			if action == "HOLD" {
				status = "REVIEW"
				msg += "; held for a steward"
			}
			issue(nil, "RELATED_TYPE_DIVERGENCE", "ERROR", msg)
			controls = append(controls, map[string]any{"control": "RELATED_TYPE_DIVERGENCE", "level": lvl, "action": action, "pct": round4(pct), "against": rt})
			break
		}
	}
	// Confidence: the share of the sources within the warning threshold of
	// the golden value (exact agreement is too strict for prices).
	if ov == nil && len(contribs) > 0 && value != 0 {
		tol := 0.5
		if th != nil && th.Warning.Valid {
			tol = th.Warning.Float64
		}
		agree := 0
		for _, c := range contribs {
			if x, ok := number(c.Attrs["value"]); ok && math.Abs(x-value)/math.Abs(value)*100 <= tol {
				agree++
			}
		}
		d.Confidence = float64(agree) / float64(len(contribs))
		if d.Winner.Stale {
			d.Confidence /= 2
		}
	}
	sort.Slice(prov, func(i, j int) bool { return prov[i].Source < prov[j].Source })
	sigParts := make([]string, len(prov))
	for i, p := range prov {
		sigParts[i] = fmt.Sprintf("%s|%v|%s|%s|%v|%s", p.Source, p.Value, p.Currency, p.AsOf, p.Stale, p.Excluded)
	}
	ovID := ""
	if ov != nil {
		ovID = ov.ID
	}
	prior := ""
	if st.PriorValue.Valid {
		prior = fmt.Sprint(st.PriorDate.String, st.PriorValue.Float64)
	}
	// The prior value is part of what was decided: a changed prior (an
	// earlier date restated) is a re-check worth a version.
	sum := sha256.Sum256([]byte(strings.Join(sigParts, ";") + "#" + winner + "#" + status + fmt.Sprint(value) + "#" + ovID + "#" + prior))
	signature := hex.EncodeToString(sum[:8])

	// Unchanged: the same candidates, winner, value and status as the latest version.
	if st.Version.Valid && !r.force {
		var before struct {
			Signature string `json:"signature"`
		}
		_ = json.Unmarshal(st.Attributes, &before)
		if before.Signature == signature {
			r.counts.Unchanged++
			return nil, issues, variances
		}
	}

	attrs := map[string]any{
		"price_type": k.PriceType, "strategy": d.Strategy, "reason": d.Reason, "rule_id": d.RuleID, "ranking": ranking,
		"candidates": prov, "threshold": th.data(), "controls": controls, "signature": signature,
		"instrument": map[string]any{"asset_class": facts.AssetClass, "sub_asset_class": facts.SubType, "currency": facts.Currency},
	}
	if st.PriorValue.Valid {
		attrs["prior"] = map[string]any{"date": st.PriorDate.String, "value": st.PriorValue.Float64}
	}
	if changePct != nil {
		attrs["change_pct"] = round4(*changePct)
	}
	if d.Held {
		attrs["held"] = true
	}
	aj, _ := json.Marshal(attrs)
	wj, _ := json.Marshal(map[string]string{"value": winner})
	dq := 100.0
	for _, is := range issues {
		if is.Severity == "ERROR" {
			dq -= 25
		} else {
			dq -= 10
		}
	}
	if dq < 0 {
		dq = 0
	}
	row := &goldenPriceRow{EntityType: k.EntityType, EntityID: k.EntityID, Date: k.Date, PriceType: k.PriceType, Version: 1,
		Current: status == "PUBLISHED", Value: value, Sources: len(contribs), Variance: maxVar, Confidence: d.Confidence, DQ: dq,
		Stale: d.Winner.Stale || d.Held, Attributes: aj, Winners: wj, Status: status}
	if st.Version.Valid {
		row.Version = int(st.Version.Int64) + 1
	}
	if w, ok := info[winner]; ok && !d.Held {
		id := w.SourceID
		row.Winner = &id
		if w.Currency.Valid {
			c := strings.ToUpper(w.Currency.String)
			row.Currency = &c
		}
		if w.AsOf.Valid {
			t := w.AsOf.Time
			row.PriceTime = &t
		}
		row.Official = w.IsOfficial
	}
	if row.Currency == nil && facts.Currency != "" {
		c := facts.Currency
		row.Currency = &c
	}
	if status == "PUBLISHED" {
		r.counts.Published++
	} else {
		r.counts.HeldForReview++
	}
	return row, issues, variances
}

func round4(f float64) float64 { return math.Round(f*10000) / 10000 }

// writeSeries writes a chunk's golden versions, exceptions and variance
// events: one statement each.
func (r *runner) writeSeries(golden []goldenPriceRow, issues []priceIssueRow, variances []varianceRow) error {
	ctx, tx := r.ctx, r.tx
	if len(golden) > 0 {
		gj, err := json.Marshal(golden)
		if err != nil {
			return err
		}
		// Publishing supersedes the key's current version; a version held
		// for review is recorded, not current.
		if _, err := tx.ExecContext(ctx, `UPDATE mdm.price_golden_record g
			SET is_current = false, status = CASE WHEN g.status = 'PUBLISHED' THEN 'SUPERSEDED' ELSE g.status END
			FROM jsonb_to_recordset($1::jsonb) AS x(et text, eid text, d text, pt text, cur boolean)
			WHERE x.cur AND g.is_current AND g.price_entity_type = x.et AND g.price_entity_id = x.eid::uuid
			  AND g.price_date = x.d::date AND g.price_type_cd = x.pt`, gj); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO mdm.price_golden_record (tenant_id, price_entity_type, price_entity_id, price_date,
				price_time, golden_version, is_current, price_type_cd, golden_value, currency, winning_source_id, source_count, variance_pct,
				confidence, dq_score, is_stale, is_official, effective_date, knowledge_timestamp, golden_attributes, winning_sources, status,
				published_at, published_by)
			SELECT $2::uuid, x.et, x.eid::uuid, x.d::date, x.ptime, x.v, x.cur, x.pt, x.val, x.ccy, x.win::uuid, x.n, x.var,
				x.conf, x.dq, x.stale, x.off, x.d::date, now(), x.attrs, x.wins, x.st, CASE WHEN x.cur THEN now() END, $3::uuid
			FROM jsonb_to_recordset($1::jsonb) AS x(et text, eid text, d text, pt text, v int, cur boolean, val numeric, ccy text,
				ptime timestamptz, win text, n int, var numeric, conf numeric, dq numeric, stale boolean, off boolean, attrs jsonb,
				wins jsonb, st text)`, gj, r.tenant, nullUUID(r.req.StartedByID)); err != nil {
			return err
		}
	}
	if len(golden) > 0 {
		// A key with a new version was checked again: its open control
		// exceptions are superseded by what this check raises.
		gj, err := json.Marshal(golden)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE mdm.price_exception e SET status = 'RESOLVED', resolved_at = now(),
				resolution_note = 'Superseded: re-checked in version ' || x.v || ' (run ' || $2::text || ')'
			FROM jsonb_to_recordset($1::jsonb) AS x(et text, eid text, d text, pt text, v int)
			WHERE e.status IN ('OPEN', 'IN_REVIEW') AND e.exception_type IN ('DAY_OVER_DAY', 'RELATED_TYPE_DIVERGENCE', 'STALE_PRICE')
			  AND e.price_entity_type = x.et AND e.price_entity_id = x.eid::uuid AND e.price_date = x.d::date
			  AND e.custom_attributes->>'price_type' = x.pt`, gj, r.run.ID); err != nil {
			return err
		}
	}
	if len(golden) > 0 {
		// A missing price that has now arrived is no longer missing.
		gj, err := json.Marshal(golden)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE mdm.price_exception e SET status = 'RESOLVED', resolved_at = now(),
				resolution_note = 'Priced (run ' || $2::text || ')'
			FROM jsonb_to_recordset($1::jsonb) AS x(et text, eid text, d text, pt text, cur boolean)
			WHERE x.cur AND e.status IN ('OPEN', 'IN_REVIEW') AND e.exception_type = 'MISSING_PRICE'
			  AND e.price_entity_type = x.et AND e.price_entity_id = x.eid::uuid AND e.price_date = x.d::date
			  AND e.custom_attributes->>'price_type' = x.pt`, gj, r.run.ID); err != nil {
			return err
		}
	}
	if len(issues) > 0 {
		ij, err := json.Marshal(issues)
		if err != nil {
			return err
		}
		res, err := tx.ExecContext(ctx, `INSERT INTO mdm.price_exception (tenant_id, price_entity_type, price_entity_id, price_date, source_id,
				exception_type, severity, exception_description, detected_at, status, custom_attributes)
			SELECT DISTINCT ON (x.et, x.eid, x.d, x.t, x.attrs->>'price_type', x.src) $2::uuid, x.et, x.eid::uuid, x.d::date, x.src::uuid,
				x.t, x.sev, x.msg, now(), 'OPEN', x.attrs
			FROM jsonb_to_recordset($1::jsonb) AS x(et text, eid text, d text, src text, t text, sev text, msg text, attrs jsonb)
			WHERE NOT EXISTS (SELECT 1 FROM mdm.price_exception e WHERE e.status = 'OPEN' AND e.exception_type = x.t
				AND e.price_entity_id = x.eid::uuid AND e.price_date = x.d::date
				AND e.custom_attributes->>'price_type' = x.attrs->>'price_type' AND e.source_id IS NOT DISTINCT FROM x.src::uuid)`,
			ij, r.tenant)
		if err != nil {
			return err
		}
		n, _ := res.RowsAffected()
		r.counts.Exceptions += int(n)
	}
	if len(variances) > 0 {
		vj, err := json.Marshal(variances)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO mdm.price_variance_event (tenant_id, price_entity_type, price_entity_id, price_type_id,
				price_date, source_a_id, source_b_id, price_a, price_b, variance_absolute, variance_pct, variance_bps, threshold_id, severity,
				status, selected_source_id, selected_price, custom_attributes)
			SELECT $2::uuid, x.et, x.eid::uuid, pt.id, x.d::date, x.a::uuid, x.b::uuid, x.pa, x.pb, x.abs, x.pct, x.pct * 100, x.th::uuid,
				x.sev, 'OPEN', x.a::uuid, x.sel, jsonb_build_object('price_type', x.pt, 'run_id', $3::text)
			FROM jsonb_to_recordset($1::jsonb) AS x(et text, eid text, d text, pt text, a text, b text, pa numeric, pb numeric,
				abs numeric, pct numeric, th text, sev text, sel numeric)
			JOIN mdm.price_type pt ON pt.price_type_cd = x.pt
			WHERE NOT EXISTS (SELECT 1 FROM mdm.price_variance_event e WHERE e.status = 'OPEN' AND e.price_entity_id = x.eid::uuid
				AND e.price_date = x.d::date AND e.price_type_id = pt.id AND e.source_b_id = x.b::uuid AND e.price_b = x.pb)`,
			vj, r.tenant, r.run.ID); err != nil {
			return err
		}
	}
	return nil
}

// writeStale records the chunk's stale prices (once while open).
func (r *runner) writeStale() error {
	if len(r.staleEvents) == 0 {
		return nil
	}
	sj, err := json.Marshal(r.staleEvents)
	r.staleEvents = nil
	if err != nil {
		return err
	}
	_, err = r.tx.ExecContext(r.ctx, `INSERT INTO mdm.price_stale_event (tenant_id, price_entity_type, price_entity_id, source_id, price_date,
			last_valid_price_date, staleness_days, severity, status, custom_attributes)
		SELECT $2::uuid, x.et, x.eid::uuid, x.src::uuid, x.d::date, x.since::date, x.days, 'WARNING', 'OPEN',
			jsonb_build_object('price_type', x.pt, 'kind', x.kind, 'run_id', $3::text)
		FROM jsonb_to_recordset($1::jsonb) AS x(et text, eid text, d text, pt text, src text, since text, days int, kind text)
		WHERE NOT EXISTS (SELECT 1 FROM mdm.price_stale_event e WHERE e.status = 'OPEN' AND e.price_entity_id = x.eid::uuid
			AND e.price_date = x.d::date AND e.custom_attributes->>'price_type' = x.pt AND e.custom_attributes->>'kind' = x.kind)`,
		sj, r.tenant, r.run.ID)
	return err
}
