//go:build integration

package mastering

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/hondyman/uisce/backend/internal/analytics"
	"github.com/hondyman/uisce/backend/internal/stagingbind"
)

// TestSecurityMasterEndToEnd masters a real staged SECURITY load through the
// same wiring the pipeline uses: the tenant's own staging binding and the
// platform catalog's field map, not a hand-written stub.
//
// It exists because staging.security_data names its fields by golden attribute
// ("security_name") while every other binding names them by BO field
// ("SecName"). When prepare() returned the field map in its raw orientation the
// lookup missed, mint() rejected every record for a missing security_name, and
// the run still reported COMPLETED — twelve invalid, zero published.
//
// Runs in a rolled-back transaction; nothing is kept.
func TestSecurityMasterEndToEnd(t *testing.T) {
	alphaDSN, dataDSN := os.Getenv("MASTERING_ALPHA_DSN"), os.Getenv("MASTERING_DATA_DSN")
	if alphaDSN == "" || dataDSN == "" {
		t.Skip("set MASTERING_ALPHA_DSN and MASTERING_DATA_DSN")
	}
	alpha, data := sqlx.MustConnect("postgres", alphaDSN), sqlx.MustConnect("postgres", dataDSN)
	defer alpha.Close()
	defer data.Close()

	ctx := context.Background()
	platform := PlatformCatalog{DB: alpha}
	gold, _ := platform.GoldCopyTenant(ctx)
	bindings := &stagingbind.Store{DB: alpha}
	e := &Engine{
		Data: data, Fields: platform, Bindings: bindings, GoldCopy: platform.GoldCopyTenant,
		Rules: analytics.NewValidationRuleService(alpha),
	}

	// RLS reads the tenant from the session; set it on both connections since
	// the field map is resolved on the control-plane connection.
	for _, db := range []*sqlx.DB{alpha, data} {
		if _, err := db.ExecContext(ctx, `SELECT set_config('app.current_tenant', $1, false)`, gold); err != nil {
			t.Fatal(err)
		}
	}

	tx, err := data.BeginTxx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback() //nolint:errcheck

	// The most recent committed SECURITY load, staged by the pipeline.
	var loadID string
	if err := tx.GetContext(ctx, &loadID, `
		SELECT l.id::text FROM staging._load_run l
		WHERE l.domain = 'SECURITY' AND l.status = 'COMPLETED' AND l.tenant_id = $1::uuid
		  AND EXISTS (SELECT 1 FROM staging.security_data s WHERE s._load_run_id = l.id)
		ORDER BY l.started_at DESC LIMIT 1`, gold); err != nil {
		t.Skipf("no staged SECURITY load to master: %v", err)
	}
	var staged int
	if err := tx.GetContext(ctx, &staged, `SELECT count(*) FROM staging.security_data WHERE _load_run_id = $1::uuid`, loadID); err != nil {
		t.Fatal(err)
	}
	if staged == 0 {
		t.Fatal("load claims COMPLETED but staged nothing")
	}

	cfg, err := readConfig(ctx, tx, gold, "security")
	if err != nil {
		t.Fatal(err)
	}

	var c Counts
	stage := ""
	r := &runner{e: e, tx: tx, ctx: ctx, tenant: gold, cfg: cfg, p: cfg.profile, run: &Run{ID: uuid.NewString()},
		req: RunRequest{Entity: "security", StagingTable: "staging.security_data", LoadRunID: loadID},
		counts: &c, stage: &stage}
	if err := r.execute(); err != nil {
		t.Fatalf("execute (%s): %v", stage, err)
	}

	if os.Getenv("MASTERING_DEBUG") != "" {
		for _, f := range []string{"SecId", "SecName", "SecTypCd", "instrument_type", "security_id", "security_name"} {
			t.Logf("  attrField[%q] = %q", f, r.attrField[f])
		}
		for _, ru := range r.rules.rules {
			t.Logf("  rule %q sev=%s fields=%v", ru.name, ru.severity, ru.fields)
		}
		var one string
		if err := tx.GetContext(ctx, &one, `SELECT instrument_type FROM staging.security_data WHERE _load_run_id = $1::uuid LIMIT 1`, loadID); err != nil {
			t.Logf("  staged instrument_type: %v", err)
		} else {
			t.Logf("  staged instrument_type = %q", one)
		}
	}

	t.Logf("counts: %+v", c)
	for _, is := range r.raised {
		t.Logf("issue: [%s] %s %s: %s", is.Severity, is.Code, is.Attribute, is.Message)
	}
	if c.Records != staged {
		t.Errorf("records: %d, want the %d staged", c.Records, staged)
	}
	// The regression this guards: every record was rejected for a missing
	// security_name because the binding key never reached the field map.
	if c.Invalid != 0 {
		t.Errorf("invalid: %d, want 0 — %s", c.Invalid, raisedSummary(r))
	}
	if c.Valid == 0 {
		t.Fatalf("no record was valid: %s", raisedSummary(r))
	}

	// The three required attributes must survive onto the golden record.
	for _, attr := range cfg.profile.Settings.Required {
		if _, ok := cfg.profile.referenceFor(attr); ok {
			continue
		}
		if _, ok := r.anchorCols[attr]; !ok {
			t.Errorf("required %q is not a column on the anchor", attr)
		}
	}
}

// raisedSummary renders the run's issues so a failure says why.
func raisedSummary(r *runner) string {
	if len(r.raised) == 0 {
		return "no issues raised"
	}
	s := ""
	for i, is := range r.raised {
		if i == 3 {
			return s + "..."
		}
		s += is.Message + "; "
	}
	return s
}
