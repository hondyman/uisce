//go:build integration

package mastering

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/hondyman/uisce/backend/internal/analytics"
)

// tableBinding is a staging binding per staging table.
type tableBinding map[string]map[string]string

func (b tableBinding) StagingFields(_ context.Context, _, _, table string) (map[string]string, error) {
	return b[table], nil
}

// TestPriceMasterEndToEnd masters end-of-day prices from Bloomberg, ICE and
// Refinitiv for a gilt and an equity of the security master over two
// valuation dates, in a rolled-back transaction (crims 0018 applied in it
// when missing). Nothing is kept.
func TestPriceMasterEndToEnd(t *testing.T) {
	alphaDSN, dataDSN := os.Getenv("MASTERING_ALPHA_DSN"), os.Getenv("MASTERING_DATA_DSN")
	if alphaDSN == "" || dataDSN == "" {
		t.Skip("set MASTERING_ALPHA_DSN and MASTERING_DATA_DSN")
	}
	alpha, data := sqlx.MustConnect("postgres", alphaDSN), sqlx.MustConnect("postgres", dataDSN)
	defer alpha.Close()
	defer data.Close()
	platform := PlatformCatalog{DB: alpha}
	ctx := context.Background()
	gold, _ := platform.GoldCopyTenant(ctx)
	e := &Engine{Data: data, Rules: analytics.NewValidationRuleService(alpha), Fields: platform, GoldCopy: platform.GoldCopyTenant,
		Bindings: tableBinding{
			"staging.bbg_price": {"@source_key": "id_bb_global", "id:FIGI": "id_bb_global", "id:ISIN": "id_isin", "id:CUSIP": "id_cusip",
				"ValuationDate": "px_date", "Currency": "crncy", "@as_of": "last_update_dt",
				"value:LAST": "px_last", "value:BID": "px_bid", "value:ASK": "px_ask", "value:MID": "px_mid"},
			"staging.ice_price": {"@source_key": "ice_id", "id:ISIN": "isin", "id:CUSIP": "cusip", "ValuationDate": "pricing_date",
				"Currency": "currency", "@as_of": "evaluated_at", "value:BID": "bid_price", "value:MID": "mid_price", "value:ASK": "ask_price"},
			"staging.rdp_price": {"@source_key": "ric", "id:ISIN": "isin", "id:CUSIP": "cusip", "id:RIC": "ric", "ValuationDate": "trade_date",
				"Currency": "currency", "@as_of": "value_ts", "value:LAST": "trdprc_1", "value:OFFICIAL_CLOSE": "official_close", "value:BID": "bid", "value:ASK": "ask", "value:MID": "mid_price"},
		}}

	tx, err := data.BeginTxx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback() //nolint:errcheck
	var have bool
	_ = tx.GetContext(ctx, &have, `SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='mdm' AND table_name='mastering_entity' AND column_name='kind')`)
	if !have {
		if _, err := tx.ExecContext(ctx, crimsDDL(t, "0018_price_master_profile.up.sql")); err != nil {
			t.Fatalf("0018: %v", err)
		}
	}
	if _, err := tx.ExecContext(ctx, `SELECT set_config('app.current_tenant', $1, true)`, gold); err != nil {
		t.Fatal(err)
	}
	const gilt, equity = "GB00BMYUK294", "GB00B0UISC06"
	var giltID, equityID string
	for isin, dst := range map[string]*string{gilt: &giltID, equity: &equityID} {
		if err := tx.GetContext(ctx, dst, `SELECT security_id::text FROM mdm.security_identifier_issuance WHERE id_type = 'ISIN' AND id_value = $1 AND is_valid LIMIT 1`, isin); err != nil {
			t.Skipf("security master has no %s: %v", isin, err)
		}
	}
	cfg, err := readConfig(ctx, tx, gold, "price")
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.profile.timeSeries() || cfg.series == nil {
		t.Fatalf("PRICE profile: kind %s", cfg.profile.Kind)
	}

	load := func(source, table string, rows ...string) string {
		var id string
		if err := tx.GetContext(ctx, &id, `INSERT INTO staging._load_run (source_system_cd, domain, run_ref, status, tenant_id, started_at, completed_at)
			VALUES ($1, 'PRICE', $2, 'COMPLETED', $3::uuid, now(), now()) RETURNING id::text`, source, "TEST-PX-"+uuid.NewString()[:8], gold); err != nil {
			t.Fatal(err)
		}
		for i, r := range rows {
			if _, err := tx.ExecContext(ctx, "INSERT INTO "+table+" "+r, id, i+1, gold); err != nil {
				t.Fatalf("%s row %d: %v", table, i+1, err)
			}
		}
		return id
	}
	run := func(label, table, loadID string) Counts {
		var c Counts
		stage := ""
		r := &runner{e: e, tx: tx, ctx: ctx, tenant: gold, cfg: cfg, p: cfg.profile, run: &Run{ID: uuid.NewString()},
			req: RunRequest{Entity: "price", StagingTable: table, LoadRunID: loadID}, counts: &c, stage: &stage}
		if err := r.execute(); err != nil {
			t.Fatalf("%s (%s): %v", label, stage, err)
		}
		t.Logf("%-22s %+v", label, c)
		for _, is := range r.raised {
			t.Logf("    %s %s: %s", is.Severity, is.Code, is.Message)
		}
		return c
	}
	type golden struct {
		Version int     `db:"golden_version"`
		Value   float64 `db:"golden_value"`
		Winner  string  `db:"winner"`
		Status  string  `db:"status"`
		Current bool    `db:"is_current"`
		Stale   bool    `db:"is_stale"`
	}
	latest := func(id, ptype, date string) golden {
		var g golden
		if err := tx.GetContext(ctx, &g, `SELECT g.golden_version, g.golden_value, COALESCE(s.code, '') AS winner, g.status, g.is_current, g.is_stale
			FROM mdm.price_golden_record g LEFT JOIN mdm.source_systems s ON s.id = g.winning_source_id
			WHERE g.price_entity_id::text = $1 AND g.price_type_cd = $2 AND g.price_date = $3::date ORDER BY g.golden_version DESC LIMIT 1`, id, ptype, date); err != nil {
			t.Fatalf("golden %s %s %s: %v", id, ptype, date, err)
		}
		return g
	}
	count := func(q string, args ...any) int {
		var n int
		if err := tx.GetContext(ctx, &n, q, args...); err != nil {
			t.Fatal(err)
		}
		return n
	}

	const bbgCols = `(_load_run_id, _source_row_num, tenant_id, id_bb_global, id_isin, px_date, crncy, px_last, px_bid, px_ask, px_mid, last_update_dt)`
	const iceCols = `(_load_run_id, _source_row_num, tenant_id, ice_id, isin, pricing_date, currency, bid_price, mid_price, ask_price, evaluated_at)`
	const rdpCols = `(_load_run_id, _source_row_num, tenant_id, ric, isin, trade_date, currency, trdprc_1, bid, ask, mid_price, value_ts, official_close)`

	// Day 1. The gilt from all three (one Refinitiv quote in the wrong
	// currency); the equity from Bloomberg and Refinitiv, 30% apart; one
	// Bloomberg quote for an instrument the master doesn't know.
	// Dates far from any real load: the transaction sees committed prices, and
	// a real price for the same key would turn a test quote into a restatement.
	d1 := "2001-06-04"
	bbg1 := load("BLOOMBERG", "staging.bbg_price",
		bbgCols+` VALUES ($1, $2, $3, 'BBGTESTGILT1', '`+gilt+`', '`+d1+`', 'GBP', NULL, 99.50, 100.00, 99.75, '`+d1+` 17:30+00')`,
		bbgCols+` VALUES ($1, $2, $3, 'BBGTESTEQTY1', '`+equity+`', '`+d1+`', 'GBP', 100.00, NULL, NULL, NULL, '`+d1+` 16:35+00')`,
		bbgCols+` VALUES ($1, $2, $3, 'BBGTESTUNKN1', 'XS9999999999', '`+d1+`', 'USD', 10.00, NULL, NULL, NULL, '`+d1+` 16:35+00')`)
	c := run("bloomberg day 1", "staging.bbg_price", bbg1)
	if c.Records != 5 || c.Invalid != 1 || c.New != 4 || c.Published != 4 {
		t.Fatalf("bloomberg day 1: %+v", c)
	}
	ice1 := load("ICE", "staging.ice_price",
		iceCols+` VALUES ($1, $2, $3, 'ICETESTGILT1', '`+gilt+`', '`+d1+`', 'GBP', 99.52, 99.76, 100.01, '`+d1+` 16:00+00')`)
	run("ice day 1", "staging.ice_price", ice1)
	rdp1 := load("REFINITIV", "staging.rdp_price",
		rdpCols+` VALUES ($1, $2, $3, 'GB5YT=RR', '`+gilt+`', '`+d1+`', 'EUR', NULL, 115.00, 116.00, 115.50, '`+d1+` 17:00+00', NULL)`,
		rdpCols+` VALUES ($1, $2, $3, 'UISC.L', '`+equity+`', '`+d1+`', 'GBP', 130.00, NULL, NULL, NULL, '`+d1+` 16:40+00', 130.00)`)
	run("refinitiv day 1", "staging.rdp_price", rdp1)

	// Fixed income ranks ICE first; equities Bloomberg first.
	if g := latest(giltID, "MID", d1); g.Winner != "ICE" || g.Value != 99.76 || !g.Current {
		t.Errorf("gilt MID day 1: %+v, want ICE 99.76 current", g)
	}
	if g := latest(equityID, "LAST", d1); g.Winner != "BLOOMBERG" || g.Value != 100 || !g.Current {
		t.Errorf("equity LAST day 1: %+v, want BLOOMBERG 100", g)
	}
	if n := count(`SELECT count(*) FROM mdm.price_variance_event WHERE price_entity_id::text = $1 AND price_date = $2::date AND severity = 'CRITICAL'`, equityID, d1); n != 1 {
		t.Errorf("equity 30%% disagreement: %d critical variance events, want 1", n)
	}
	if n := count(`SELECT count(*) FROM mdm.price_exception WHERE exception_type = 'CURRENCY_MISMATCH' AND price_entity_id::text = $1 AND status = 'OPEN'`, giltID); n == 0 {
		t.Error("the EUR quote for a GBP gilt raised no currency mismatch")
	}
	if n := count(`SELECT count(*) FROM mdm.price_exception WHERE exception_type = 'UNRESOLVED_INSTRUMENT'
			AND custom_attributes->>'source_row_id' LIKE 'BBGTESTUNKN1|%' AND exception_description LIKE '%ISIN XS9999999999%'`); n != 1 {
		t.Errorf("unresolved quote: %d exceptions, want 1", n)
	}
	// Only Refinitiv quotes the official close, 30% from the golden last
	// price: held, not published unchallenged.
	if g := latest(equityID, "OFFICIAL_CLOSE", d1); g.Status != "REVIEW" || g.Current {
		t.Errorf("single-source official close 30%% from the last price: %+v, want held", g)
	}
	if n := count(`SELECT count(*) FROM mdm.price_exception WHERE exception_type = 'RELATED_TYPE_DIVERGENCE' AND price_entity_id::text = $1 AND price_date = $2::date`, equityID, d1); n != 1 {
		t.Errorf("related-type divergence exceptions: %d, want 1", n)
	}
	var excluded bool
	_ = tx.GetContext(ctx, &excluded, `SELECT EXISTS (SELECT 1 FROM mdm.price_golden_record g, jsonb_array_elements(g.golden_attributes->'candidates') c
		WHERE g.price_entity_id::text = $1 AND g.price_type_cd = 'MID' AND g.price_date = $2::date AND g.is_current
		  AND c->>'source' = 'REFINITIV' AND c->>'excluded' LIKE 'currency%')`, giltID, d1)
	if !excluded {
		t.Error("provenance does not show the EUR quote excluded")
	}

	// Day 2: the gilt moves 4.3% (over the 3% fixed-income error threshold:
	// published, flagged); the equity 40% (critical: held for a steward).
	d2 := "2001-06-05"
	ice2 := load("ICE", "staging.ice_price",
		iceCols+` VALUES ($1, $2, $3, 'ICETESTGILT1', '`+gilt+`', '`+d2+`', 'GBP', 103.80, 104.05, 104.30, '`+d2+` 16:00+00')`)
	run("ice day 2", "staging.ice_price", ice2)
	bbg2 := load("BLOOMBERG", "staging.bbg_price",
		bbgCols+` VALUES ($1, $2, $3, 'BBGTESTEQTY1', '`+equity+`', '`+d2+`', 'GBP', 140.00, NULL, NULL, NULL, '`+d2+` 16:35+00')`)
	c = run("bloomberg day 2", "staging.bbg_price", bbg2)
	if g := latest(giltID, "MID", d2); g.Status != "PUBLISHED" || g.Value != 104.05 {
		t.Errorf("gilt MID day 2: %+v, want published 104.05 (flagged)", g)
	}
	if n := count(`SELECT count(*) FROM mdm.price_exception WHERE exception_type = 'DAY_OVER_DAY' AND price_entity_id::text = $1 AND price_date = $2::date`, giltID, d2); n == 0 {
		t.Error("gilt 4.3% move raised no day-over-day exception")
	}
	if g := latest(equityID, "LAST", d2); g.Status != "REVIEW" || g.Current {
		t.Errorf("equity LAST day 2: %+v, want held in REVIEW (not current)", g)
	}
	if c.HeldForReview != 1 {
		t.Errorf("bloomberg day 2: %+v, want 1 held", c)
	}

	// A steward releases the held price at its value (direct policy here):
	// published as the steward's price, its exceptions resolved.
	held := latest(equityID, "LAST", d2)
	var heldID string
	if err := tx.GetContext(ctx, &heldID, `SELECT id::text FROM mdm.price_golden_record WHERE price_entity_id::text = $1 AND price_type_cd = 'LAST'
		AND price_date = $2::date ORDER BY golden_version DESC LIMIT 1`, equityID, d2); err != nil {
		t.Fatal(err)
	}
	direct := &Policy{EntityCd: "PRICE", Mode: ModeDirect}
	steward := uuid.NewString()
	o, err := e.proposePriceOverride(ctx, tx, cfg, direct, gold, "price", heldID,
		OverrideRequest{Action: "SET", Value: []byte("140"), Reason: "Confirmed with the exchange: rights issue"}, steward, "Test Steward")
	if err != nil {
		t.Fatalf("accept held price: %v", err)
	}
	if o.Status != "APPLIED" || !o.Active || o.OpenID == nil || o.Attribute != "LAST@"+d2 {
		t.Errorf("override: %+v", o)
	}
	if g := latest(equityID, "LAST", d2); g.Status != "PUBLISHED" || !g.Current || g.Value != 140 || g.Winner != "MANUAL" || g.Version != held.Version+1 {
		t.Errorf("after accepting: %+v, want v%d 140 MANUAL published current", g, held.Version+1)
	}
	if n := count(`SELECT count(*) FROM mdm.price_exception WHERE price_entity_id::text = $1 AND price_date = $2::date AND status = 'OPEN'
			AND custom_attributes->>'price_type' = 'LAST'`, equityID, d2); n != 0 {
		t.Errorf("open exceptions after the steward's decision: %d", n)
	}
	// Cleared, the vendors' price is held again: nothing is current.
	if _, err := e.proposePriceOverride(ctx, tx, cfg, direct, gold, "price", heldID,
		OverrideRequest{Action: "CLEAR", Reason: "Back to the vendors"}, steward, "Test Steward"); err != nil {
		t.Fatalf("clear: %v", err)
	}
	if g := latest(equityID, "LAST", d2); g.Status != "REVIEW" || g.Current {
		t.Errorf("after clearing: %+v, want held (REVIEW, not current)", g)
	}
	if n := count(`SELECT count(*) FROM mdm.price_golden_record WHERE price_entity_id::text = $1 AND price_type_cd = 'LAST' AND price_date = $2::date AND is_current`, equityID, d2); n != 0 {
		t.Errorf("current versions after clearing: %d, want 0", n)
	}

	// Bloomberg restates day 1's equity close: a new observation linked to
	// the old one, and a new golden version.
	before := latest(equityID, "LAST", d1)
	bbgR := load("BLOOMBERG", "staging.bbg_price",
		bbgCols+` VALUES ($1, $2, $3, 'BBGTESTEQTY1', '`+equity+`', '`+d1+`', 'GBP', 101.00, NULL, NULL, NULL, '`+d1+` 18:00+00')`)
	c = run("bloomberg restatement", "staging.bbg_price", bbgR)
	if c.Restated != 1 {
		t.Errorf("restatement: %+v, want 1 restated", c)
	}
	// The next date was checked against the old close: re-checked against
	// 101 it is still critical, its exception now quoting the new prior.
	if c.Rechecked != 1 {
		t.Errorf("restatement: %+v, want day 2 re-checked", c)
	}
	var dod []string
	if err := tx.SelectContext(ctx, &dod, `SELECT exception_description FROM mdm.price_exception WHERE exception_type = 'DAY_OVER_DAY'
			AND price_entity_id::text = $1 AND price_date = $2::date AND custom_attributes->>'price_type' = 'LAST' AND status = 'OPEN'`, equityID, d2); err != nil {
		t.Fatal(err)
	}
	if len(dod) != 1 || !strings.Contains(dod[0], "from 101") {
		t.Errorf("day 2's open day-over-day exceptions after the restatement: %q, want one against 101", dod)
	}
	if g := latest(equityID, "LAST", d1); g.Version != before.Version+1 || g.Value != 101 || !g.Current {
		t.Errorf("equity LAST day 1 after restatement: %+v, want v%d 101 current", g, before.Version+1)
	}
	if n := count(`SELECT count(*) FROM mdm.price_history WHERE price_entity_id::text = $1 AND change_reason = 'RESTATED'`, equityID); n != 1 {
		t.Errorf("restatement history rows: %d, want 1", n)
	}
	if n := count(`SELECT count(*) FROM mdm.price_golden_record WHERE price_entity_id::text = $1 AND price_date = $2::date AND price_type_cd = 'LAST' AND is_current`, equityID, d1); n != 1 {
		t.Errorf("current versions: %d, want 1", n)
	}

	// A steward sets day 1's close to 135: day 2's 140 is now a 3.7% move,
	// so the re-check publishes it and resolves its exception.
	var d1ID string
	if err := tx.GetContext(ctx, &d1ID, `SELECT id::text FROM mdm.price_golden_record WHERE price_entity_id::text = $1 AND price_type_cd = 'LAST'
		AND price_date = $2::date ORDER BY golden_version DESC LIMIT 1`, equityID, d1); err != nil {
		t.Fatal(err)
	}
	if _, err := e.proposePriceOverride(ctx, tx, cfg, direct, gold, "price", d1ID,
		OverrideRequest{Action: "SET", Value: []byte("135"), Reason: "Vendor close was pre-announcement"}, steward, "Test Steward"); err != nil {
		t.Fatalf("override day 1: %v", err)
	}
	if g := latest(equityID, "LAST", d2); g.Status != "PUBLISHED" || !g.Current || g.Value != 140 {
		t.Errorf("day 2 after day 1 became 135: %+v, want 140 published and current", g)
	}
	if n := count(`SELECT count(*) FROM mdm.price_exception WHERE exception_type = 'DAY_OVER_DAY' AND price_entity_id::text = $1
			AND price_date = $2::date AND custom_attributes->>'price_type' = 'LAST' AND status = 'OPEN'`, equityID, d2); n != 0 {
		t.Errorf("day 2 still has %d open day-over-day exceptions", n)
	}

	// Running a load again changes nothing.
	c = run("ice day 1 again", "staging.ice_price", ice1)
	if c.Published != 0 || c.New != 0 || c.Unchanged == 0 {
		t.Errorf("rerun: %+v, want nothing published", c)
	}
}
