//go:build integration

package mastering

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"github.com/hondyman/uisce/backend/internal/analytics"
)

type fixedBinding map[string]string

func (b fixedBinding) StagingFields(context.Context, string, string, string) (map[string]string, error) {
	return b, nil
}

// TestPreviewFactSetProduct masters the FactSet product load in a rolled-back
// transaction and prints what a real run would do. Nothing is kept.
//
//	MASTERING_ALPHA_DSN=... MASTERING_DATA_DSN=... MASTERING_LOAD_RUN=... go test -tags integration -run TestPreviewFactSetProduct -v ./internal/mastering
func TestPreviewFactSetProduct(t *testing.T) {
	alphaDSN, dataDSN, load := os.Getenv("MASTERING_ALPHA_DSN"), os.Getenv("MASTERING_DATA_DSN"), os.Getenv("MASTERING_LOAD_RUN")
	if alphaDSN == "" || dataDSN == "" || load == "" {
		t.Skip("set MASTERING_ALPHA_DSN, MASTERING_DATA_DSN and MASTERING_LOAD_RUN")
	}
	alpha, data := sqlx.MustConnect("postgres", alphaDSN), sqlx.MustConnect("postgres", dataDSN)
	defer alpha.Close()
	defer data.Close()
	platform := PlatformCatalog{DB: alpha}
	e := &Engine{Data: data, Rules: analytics.NewValidationRuleService(alpha), Fields: platform, GoldCopy: platform.GoldCopyTenant,
		Bindings: fixedBinding{
			"ProductName": "fund_name", "ProductShortName": "fund_name_short", "ProductBaseCurrency": "base_currency",
			"ProductDomicile": "domicile_country", "ProductInceptionDate": "inception_date", "ProductIsActive": "is_active",
			"ProductTypeId": "fund_type", "id:ISIN": "isin", "id:CUSIP": "cusip", "id:SEDOL": "sedol",
			"id:PROVIDER_CODE": "fsym_id", "@source_key": "fsym_id",
		}}
	gold, err := platform.GoldCopyTenant(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	pv, err := e.Preview(context.Background(), gold, RunRequest{Entity: "product", StagingTable: "staging.ff_product", LoadRunID: load})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.MarshalIndent(pv.Counts, "", "  ")
	t.Logf("counts: %s", b)
	t.Logf("unmastered fields: %s", strings.Join(pv.Unmastered, ", "))
	for _, is := range pv.Exceptions {
		t.Logf("%s %s %s", is.Severity, is.Code, is.Message)
	}
}

// TestPreviewRerunIsIdempotent masters the same load twice in one
// rolled-back transaction: the second pass must link every record through
// its cross-reference and publish nothing new.
func TestPreviewRerunIsIdempotent(t *testing.T) {
	alphaDSN, dataDSN, load := os.Getenv("MASTERING_ALPHA_DSN"), os.Getenv("MASTERING_DATA_DSN"), os.Getenv("MASTERING_LOAD_RUN")
	if alphaDSN == "" || dataDSN == "" || load == "" {
		t.Skip("set MASTERING_ALPHA_DSN, MASTERING_DATA_DSN and MASTERING_LOAD_RUN")
	}
	alpha, data := sqlx.MustConnect("postgres", alphaDSN), sqlx.MustConnect("postgres", dataDSN)
	defer alpha.Close()
	defer data.Close()
	platform := PlatformCatalog{DB: alpha}
	e := &Engine{Data: data, Rules: analytics.NewValidationRuleService(alpha), Fields: platform, GoldCopy: platform.GoldCopyTenant,
		Bindings: fixedBinding{"ProductName": "fund_name", "ProductBaseCurrency": "base_currency", "ProductDomicile": "domicile_country",
			"ProductTypeId": "fund_type", "id:ISIN": "isin", "id:CUSIP": "cusip", "@source_key": "fsym_id"}}
	ctx := context.Background()
	gold, _ := platform.GoldCopyTenant(ctx)
	req := RunRequest{Entity: "product", StagingTable: "staging.ff_product", LoadRunID: load}
	cfg, err := e.loadConfig(ctx, gold, "product")
	if err != nil {
		t.Fatal(err)
	}
	var first, second, rekeyed Counts
	err = e.inTenant(ctx, gold, func(tx *sqlx.Tx) error {
		for i, c := range []*Counts{&first, &second, &rekeyed} {
			if c == &rekeyed {
				// The same records under new source keys: no xref, so they
				// must be found by a shared identifier.
				b := fixedBinding{}
				for k, v := range e.Bindings.(fixedBinding) {
					b[k] = v
				}
				b["@source_key"] = "cusip"
				e.Bindings = b
			}
			stage := ""
			r := &runner{e: e, tx: tx, ctx: ctx, tenant: gold, cfg: cfg, p: cfg.profile, run: &Run{ID: fmt.Sprintf("00000000-0000-0000-0000-00000000000%d", i+1)}, req: req, counts: c, stage: &stage}
			if err := r.execute(); err != nil {
				return err
			}
		}
		return errPreview
	})
	if err != nil && err != errPreview {
		t.Fatal(err)
	}
	t.Logf("first %+v", first)
	t.Logf("second %+v", second)
	// Whether or not the products were mastered before, a second pass over
	// the same load links every valid record and publishes nothing.
	if second.Xref != first.Valid || second.New != 0 || second.Published != 0 || second.Unchanged != first.Valid {
		t.Errorf("rerun must be a no-op: first %+v second %+v", first, second)
	}
	t.Logf("rekeyed %+v", rekeyed)
	if rekeyed.Deterministic != first.Valid || rekeyed.New != 0 {
		t.Errorf("records under new keys must match by identifier: %+v", rekeyed)
	}
	if second.Exceptions != 0 {
		t.Errorf("open exceptions must not be raised twice: %d", second.Exceptions)
	}
}

// TestPendingLoads: a load already mastered under its load key is not
// pending (read-only).
func TestPendingLoads(t *testing.T) {
	alphaDSN, dataDSN := os.Getenv("MASTERING_ALPHA_DSN"), os.Getenv("MASTERING_DATA_DSN")
	if alphaDSN == "" || dataDSN == "" {
		t.Skip("set MASTERING_ALPHA_DSN and MASTERING_DATA_DSN")
	}
	alpha, data := sqlx.MustConnect("postgres", alphaDSN), sqlx.MustConnect("postgres", dataDSN)
	defer alpha.Close()
	defer data.Close()
	e := &Engine{Data: data}
	gold, err := PlatformCatalog{DB: alpha}.GoldCopyTenant(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	loads, err := e.PendingLoads(context.Background(), gold, "product", "staging.ff_product")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("pending loads: %v", loads)
}

// TestStewardMergeAndReject raises a possible-duplicate pair between two
// real golden products, merges it and rejects another, all in a rolled-back
// transaction.
func TestStewardMergeAndReject(t *testing.T) {
	alphaDSN, dataDSN := os.Getenv("MASTERING_ALPHA_DSN"), os.Getenv("MASTERING_DATA_DSN")
	if alphaDSN == "" || dataDSN == "" {
		t.Skip("set MASTERING_ALPHA_DSN and MASTERING_DATA_DSN")
	}
	alpha, data := sqlx.MustConnect("postgres", alphaDSN), sqlx.MustConnect("postgres", dataDSN)
	defer alpha.Close()
	defer data.Close()
	platform := PlatformCatalog{DB: alpha}
	e := &Engine{Data: data, Rules: analytics.NewValidationRuleService(alpha), Fields: platform, GoldCopy: platform.GoldCopyTenant}
	ctx := context.Background()
	gold, _ := platform.GoldCopyTenant(ctx)
	cfg, err := e.loadConfig(ctx, gold, "product")
	if err != nil {
		t.Fatal(err)
	}
	err = e.inTenant(ctx, gold, func(tx *sqlx.Tx) error {
		var ids []string
		if err := tx.SelectContext(ctx, &ids, `SELECT golden_id::text FROM mdm.entity_xref WHERE entity_cd = 'PRODUCT' AND status = 'ACTIVE' ORDER BY golden_id LIMIT 3`); err != nil {
			return err
		}
		if len(ids) < 3 {
			t.Skip("needs three mastered products")
		}
		a, b, c := ids[0], ids[1], ids[2]
		pair := func(x, y string) string {
			var id string
			if err := tx.GetContext(ctx, &id, `INSERT INTO mdm.product_match_candidate (tenant_id, match_rule_id, product_id_a, product_id_b, overall_score)
				SELECT $1::uuid, id, $2::uuid, $3::uuid, 0.85 FROM mdm.product_match_rule WHERE rule_cd = 'PRODUCT_NAME_FUZZY' LIMIT 1 RETURNING id::text`, gold, x, y); err != nil {
				t.Fatal(err)
			}
			return id
		}
		ab, ac, bc := pair(a, b), pair(a, c), pair(b, c)

		rej, err := e.decide(ctx, tx, cfg, gold, "product", ac, CandidateDecision{Merge: false, Note: "different share classes"}, "", "test")
		if err != nil || rej.Status != "REJECTED" {
			t.Fatalf("reject: %+v %v", rej, err)
		}
		m, err := e.decide(ctx, tx, cfg, gold, "product", ab, CandidateDecision{Merge: true, Note: "same fund"}, "", "test")
		if err != nil {
			t.Fatalf("merge: %v", err)
		}
		t.Logf("merge: %+v", m)
		if m.Survivor != a || m.Merged != b || m.Moved.Sources != 1 || !m.Published {
			t.Errorf("merge result %+v", m)
		}
		var st struct {
			Sources     int    `db:"sources"`
			Merged      string `db:"merged"`
			Retracted   int    `db:"retracted"`
			Idents      int    `db:"idents"`
			BIdents     int    `db:"bidents"`
			Version     int    `db:"version"`
			OtherStatus string `db:"other"`
			Logged      int    `db:"logged"`
		}
		if err := tx.GetContext(ctx, &st, `SELECT
			(SELECT count(*) FROM mdm.entity_xref WHERE golden_id = $1::uuid AND status = 'ACTIVE') AS sources,
			(SELECT COALESCE(merged_into_id::text, '') FROM mdm.product WHERE id = $2::uuid) AS merged,
			(SELECT count(*) FROM mdm.product_golden_record WHERE product_id = $2::uuid AND status = 'RETRACTED') AS retracted,
			(SELECT count(*) FROM mdm.product_identifier WHERE product_id = $1::uuid AND effective_to IS NULL) AS idents,
			(SELECT count(*) FROM mdm.product_identifier WHERE product_id = $2::uuid AND effective_to IS NULL) AS bidents,
			(SELECT max(golden_version) FROM mdm.product_golden_record WHERE product_id = $1::uuid AND is_current) AS version,
			(SELECT status FROM mdm.product_match_candidate WHERE id = $3::uuid) AS other,
			(SELECT count(*) FROM mdm.product_merge_log WHERE merged_product_id = $2::uuid) AS logged`, a, b, bc); err != nil {
			return err
		}
		t.Logf("after merge: %+v", st)
		if st.Sources != 2 || st.Merged != a || st.Retracted != 1 || st.BIdents != 0 || st.Version != 2 || st.OtherStatus != "REJECTED" || st.Logged != 1 {
			t.Errorf("state after merge: %+v", st)
		}
		if _, err := e.decide(ctx, tx, cfg, gold, "product", ab, CandidateDecision{Merge: true}, "", "test"); err == nil {
			t.Error("a decided pair must not be decided again")
		}
		return errPreview
	})
	if err != nil && err != errPreview {
		t.Fatal(err)
	}
}

// ddl0013 is the overrides migration without psql meta-commands or its own
// transaction, so a test can create the tables inside a rolled-back one.
func ddl0013(t *testing.T) string {
	b, err := os.ReadFile("../../db/crims/0013_mastering_overrides.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	var keep []string
	for _, l := range strings.Split(string(b), "\n") {
		if s := strings.TrimSpace(l); strings.HasPrefix(s, `\`) || s == "BEGIN;" || s == "COMMIT;" {
			continue
		}
		keep = append(keep, l)
	}
	return strings.Join(keep, "\n")
}

// TestOverridesApprovalAndDirect: an override under approval waits, can't
// be approved by its proposer, applies on another person's approval and
// sticks on a later re-mastering; a clear under a direct policy applies at
// once and the source value returns. All rolled back.
func TestOverridesApprovalAndDirect(t *testing.T) {
	alphaDSN, dataDSN := os.Getenv("MASTERING_ALPHA_DSN"), os.Getenv("MASTERING_DATA_DSN")
	if alphaDSN == "" || dataDSN == "" {
		t.Skip("set MASTERING_ALPHA_DSN and MASTERING_DATA_DSN")
	}
	alpha, data := sqlx.MustConnect("postgres", alphaDSN), sqlx.MustConnect("postgres", dataDSN)
	defer alpha.Close()
	defer data.Close()
	platform := PlatformCatalog{DB: alpha}
	e := &Engine{Data: data, Rules: analytics.NewValidationRuleService(alpha), Fields: platform, GoldCopy: platform.GoldCopyTenant}
	ctx := context.Background()
	gold, _ := platform.GoldCopyTenant(ctx)
	cfg, err := e.loadConfig(ctx, gold, "product")
	if err != nil {
		t.Fatal(err)
	}
	err = e.inTenant(ctx, gold, func(tx *sqlx.Tx) error {
		if _, err := tx.ExecContext(ctx, `SELECT to_regclass('mdm.golden_override') IS NULL`); err != nil {
			return err
		}
		var exists bool
		_ = tx.GetContext(ctx, &exists, `SELECT to_regclass('mdm.golden_override') IS NOT NULL`)
		if !exists {
			if _, err := tx.ExecContext(ctx, ddl0013(t)); err != nil {
				return fmt.Errorf("0013: %w", err)
			}
			if _, err := tx.ExecContext(ctx, `SELECT set_config('app.current_tenant', $1, true)`, gold); err != nil {
				return err
			}
		}
		var golden, sourceName string
		if err := tx.QueryRowxContext(ctx, `SELECT x.golden_id::text, p.name FROM mdm.entity_xref x JOIN mdm.product p ON p.id = x.golden_id
			WHERE x.entity_cd = 'PRODUCT' AND x.status = 'ACTIVE' ORDER BY x.golden_id LIMIT 1`).Scan(&golden, &sourceName); err != nil {
			return err
		}
		approval := &Policy{Mode: ModeApproval, ApprovalsRequired: 1}
		alice, john := "user-alice", "user-john"
		val, _ := json.Marshal("Uisce Global Equity Income Fund (Acc)")

		o, err := e.proposeOverride(ctx, tx, cfg, approval, gold, "product", golden, OverrideRequest{Attribute: "name", Value: val, Reason: "prospectus name"}, alice, "alice")
		if err != nil {
			t.Fatalf("propose: %v", err)
		}
		if o.Status != "PENDING" || o.ApprovalsRequired != 1 {
			t.Errorf("pending: %+v", o)
		}
		if _, err := e.decideOverride(ctx, tx, cfg, gold, "product", o.ID, true, "", alice, "alice"); err == nil {
			t.Error("the proposer must not approve their own override")
		}
		o, err = e.decideOverride(ctx, tx, cfg, gold, "product", o.ID, true, "checked the prospectus", john, "john")
		if err != nil {
			t.Fatalf("approve: %v", err)
		}
		var name, src string
		var ver int
		read := func() {
			if err := tx.QueryRowxContext(ctx, `SELECT p.name, gr.golden_version,
					COALESCE((SELECT l.decision_reason FROM mdm.product_survivorship_log l WHERE l.product_id = p.id AND l.field_name = 'name'
					  ORDER BY l.golden_version DESC LIMIT 1), '')
				FROM mdm.product p JOIN mdm.product_golden_record gr ON gr.product_id = p.id AND gr.is_current WHERE p.id = $1::uuid`, golden).Scan(&name, &ver, &src); err != nil {
				t.Fatal(err)
			}
		}
		read()
		t.Logf("after approval: status=%s active=%v version=%v name=%q why=%q", o.Status, o.Active, o.AppliedVersion, name, src)
		if o.Status != "APPLIED" || !o.Active || name != "Uisce Global Equity Income Fund (Acc)" || !strings.Contains(src, "approved by john") {
			t.Errorf("applied: %+v name=%q why=%q", o, name, src)
		}

		// A later re-mastering (no force) keeps the override.
		r := e.stewardRunner(ctx, tx, cfg, gold, "product", "", "")
		if _, err := r.prepare(); err != nil {
			return err
		}
		if err := r.masterOne(golden); err != nil {
			return err
		}
		read()
		if name != "Uisce Global Equity Income Fund (Acc)" || r.counts.Unchanged != 1 {
			t.Errorf("override must stick on re-mastering: name=%q counts=%+v", name, r.counts)
		}

		// Direct policy: clearing applies at once and the source value returns.
		c, err := e.proposeOverride(ctx, tx, cfg, &Policy{Mode: ModeDirect}, gold, "product", golden, OverrideRequest{Attribute: "name", Action: "CLEAR", Reason: "prospectus superseded"}, john, "john")
		if err != nil {
			t.Fatalf("clear: %v", err)
		}
		read()
		t.Logf("after direct clear: status=%s version=%d name=%q", c.Status, ver, name)
		if c.Status != "APPLIED" || c.Mode != ModeDirect || name != sourceName {
			t.Errorf("clear: %+v name=%q want %q", c, name, sourceName)
		}
		if _, err := e.proposeOverride(ctx, tx, cfg, &Policy{Mode: ModeDirect}, gold, "product", golden, OverrideRequest{Attribute: "name", Action: "CLEAR", Reason: "again"}, john, "john"); err == nil {
			t.Error("nothing left to clear must be refused")
		}
		code, _ := json.Marshal("NOT_A_TYPE")
		if _, err := e.proposeOverride(ctx, tx, cfg, approval, gold, "product", golden, OverrideRequest{Attribute: "product_type_cd", Value: code, Reason: "x"}, alice, "alice"); err == nil {
			t.Error("a reference attribute must take a known code")
		}
		return errPreview
	})
	if err != nil && err != errPreview {
		t.Fatal(err)
	}
}
