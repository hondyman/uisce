//go:build integration

package mastering

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
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
		if err := tx.SelectContext(ctx, &ids, `SELECT DISTINCT golden_id::text FROM mdm.entity_xref WHERE entity_cd = 'PRODUCT' AND status = 'ACTIVE' ORDER BY 1 LIMIT 3`); err != nil {
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
		srcs := func(id string) int {
			var n int
			if err := tx.GetContext(ctx, &n, `SELECT count(*) FROM mdm.entity_xref WHERE golden_id = $1::uuid AND status = 'ACTIVE'`, id); err != nil {
				t.Fatal(err)
			}
			return n
		}
		aSrc, bSrc := srcs(a), srcs(b)
		var aVer int
		_ = tx.GetContext(ctx, &aVer, `SELECT COALESCE(max(golden_version), 0) FROM mdm.product_golden_record WHERE product_id = $1::uuid`, a)

		rej, err := e.decide(ctx, tx, cfg, &Policy{Mode: ModeDirect}, gold, "product", ac, CandidateDecision{Merge: false, Note: "different share classes"}, "", "test")
		if err != nil || rej.Status != "REJECTED" {
			t.Fatalf("reject: %+v %v", rej, err)
		}
		m, err := e.decide(ctx, tx, cfg, &Policy{Mode: ModeDirect}, gold, "product", ab, CandidateDecision{Merge: true, Note: "same fund"}, "", "test")
		if err != nil {
			t.Fatalf("merge: %v", err)
		}
		t.Logf("merge: %+v", m)
		if m.Survivor != a || m.Merged != b || m.Moved.Sources != bSrc || !m.Published {
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
		if st.Sources != aSrc+bSrc || st.Merged != a || st.Retracted < 1 || st.BIdents != 0 || st.Version != aVer+1 || st.OtherStatus != "REJECTED" || st.Logged != 1 {
			t.Errorf("state after merge: %+v", st)
		}
		if _, err := e.decide(ctx, tx, cfg, &Policy{Mode: ModeDirect}, gold, "product", ab, CandidateDecision{Merge: true}, "", "test"); err == nil {
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
func ddl0013(t *testing.T) string { return crimsDDL(t, "0013_mastering_overrides.up.sql") }

// crimsDDL reads a crims migration for running inside a test transaction.
func crimsDDL(t *testing.T, file string) string {
	b, err := os.ReadFile("../../db/crims/" + file)
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

// TestMergeFollowsPolicy: under an approval policy a merge waits for
// another person; the requester can't approve; a rejection returns the
// pair to review, where keeping it apart still works. Rolled back.
func TestMergeFollowsPolicy(t *testing.T) {
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
		var exists bool
		_ = tx.GetContext(ctx, &exists, `SELECT to_regclass('mdm.golden_merge_request') IS NOT NULL`)
		if !exists {
			if _, err := tx.ExecContext(ctx, crimsDDL(t, "0014_mastering_merge_requests.up.sql")); err != nil {
				return fmt.Errorf("0014: %w", err)
			}
		}
		var ids []string
		if err := tx.SelectContext(ctx, &ids, `SELECT DISTINCT golden_id::text FROM mdm.entity_xref WHERE entity_cd = 'PRODUCT' AND status = 'ACTIVE' ORDER BY 1 LIMIT 3`); err != nil {
			return err
		}
		if len(ids) < 3 {
			t.Skip("needs three mastered products")
		}
		pair := func(x, y string) string {
			var id string
			if err := tx.GetContext(ctx, &id, `INSERT INTO mdm.product_match_candidate (tenant_id, match_rule_id, product_id_a, product_id_b, overall_score)
				SELECT $1::uuid, id, $2::uuid, $3::uuid, 0.85 FROM mdm.product_match_rule WHERE rule_cd = 'PRODUCT_NAME_FUZZY' LIMIT 1 RETURNING id::text`, gold, x, y); err != nil {
				t.Fatal(err)
			}
			return id
		}
		approval := &Policy{Mode: ModeApproval, ApprovalsRequired: 1}
		ab, ac := pair(ids[0], ids[1]), pair(ids[0], ids[2])
		// An expected failure must not poison the test's one transaction.
		refused := func(what string, f func() error) {
			if _, err := tx.ExecContext(ctx, "SAVEPOINT expect"); err != nil {
				t.Fatal(err)
			}
			if f() == nil {
				t.Error(what)
			}
			if _, err := tx.ExecContext(ctx, "ROLLBACK TO SAVEPOINT expect"); err != nil {
				t.Fatal(err)
			}
		}

		req, err := e.decide(ctx, tx, cfg, approval, gold, "product", ab, CandidateDecision{Merge: true, Note: "same fund"}, "user-alice", "alice")
		if err != nil || req.Status != "PENDING_APPROVAL" || req.MergeRequest == "" {
			t.Fatalf("request: %+v %v", req, err)
		}
		refused("a second merge request on a pending pair must be refused", func() error {
			_, err := e.decide(ctx, tx, cfg, approval, gold, "product", ab, CandidateDecision{Merge: true}, "user-bob", "bob")
			return err
		})
		refused("the requester must not approve their own merge", func() error {
			_, err := e.decideMerge(ctx, tx, cfg, gold, "product", req.MergeRequest, true, "", "user-alice", "alice")
			return err
		})
		done, err := e.decideMerge(ctx, tx, cfg, gold, "product", req.MergeRequest, true, "agree", "user-john", "john")
		if err != nil || done.Status != "APPROVED" || done.Survivor != ids[0] {
			t.Fatalf("approve: %+v %v", done, err)
		}
		var approvedBy, merged string
		if err := tx.QueryRowxContext(ctx, `SELECT COALESCE(custom_attributes->>'approved_by', ''),
				(SELECT COALESCE(merged_into_id::text, '') FROM mdm.product WHERE id = $1::uuid)
			FROM mdm.product_merge_log WHERE merged_product_id = $1::uuid`, ids[1]).Scan(&approvedBy, &merged); err != nil {
			return err
		}
		t.Logf("merged by alice, approved by %q; merged into %s", approvedBy, merged)
		if approvedBy != "john" || merged != ids[0] {
			t.Errorf("merge log: approved_by=%q merged_into=%q", approvedBy, merged)
		}

		// A second pair: request a merge, have it rejected, then keep the pair apart.
		req2, err := e.decide(ctx, tx, cfg, approval, gold, "product", ac, CandidateDecision{Merge: true, Keep: "b"}, "user-alice", "alice")
		if err != nil {
			return err
		}
		rej, err := e.decideMerge(ctx, tx, cfg, gold, "product", req2.MergeRequest, false, "different share class", "user-john", "john")
		if err != nil || rej.Status != "MERGE_REJECTED" {
			t.Fatalf("reject: %+v %v", rej, err)
		}
		apart, err := e.decide(ctx, tx, cfg, approval, gold, "product", ac, CandidateDecision{Merge: false}, "user-john", "john")
		if err != nil || apart.Status != "REJECTED" {
			t.Fatalf("keep apart after a rejected merge: %+v %v", apart, err)
		}
		return errPreview
	})
	if err != nil && err != errPreview {
		t.Fatal(err)
	}
}

// TestReviewCandidateInsert exercises the review band's candidate insert
// (a record scoring between review and auto-match): it must record the
// pair. Rolled back.
func TestReviewCandidateInsert(t *testing.T) {
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
		if err := tx.SelectContext(ctx, &ids, `SELECT DISTINCT golden_id::text FROM mdm.entity_xref WHERE entity_cd = 'PRODUCT' AND status = 'ACTIVE' ORDER BY 1 LIMIT 2`); err != nil {
			return err
		}
		if len(ids) < 2 {
			t.Skip("needs two mastered products")
		}
		r := e.stewardRunner(ctx, tx, cfg, gold, "product", "", "")
		r.sourceCd = "BLOOMBERG"
		if err := r.candidate(ids[0], ids[1], 0.9, "PRODUCT_NAME_FUZZY", []string{"name", "base_currency", "domicile"}); err != nil {
			t.Fatalf("candidate insert: %v", err)
		}
		var n int
		if err := tx.GetContext(ctx, &n, `SELECT count(*) FROM mdm.product_match_candidate WHERE product_id_a = $1::uuid AND product_id_b = $2::uuid AND status = 'PENDING'`, ids[0], ids[1]); err != nil {
			return err
		}
		if n != 1 {
			t.Errorf("pending pairs: %d", n)
		}
		return errPreview
	})
	if err != nil && err != errPreview {
		t.Fatal(err)
	}
}

// TestHierarchyLoads: the entity's source hierarchy comes from
// mdm.product_source_priority, by field group, best first.
func TestHierarchyLoads(t *testing.T) {
	alphaDSN, dataDSN := os.Getenv("MASTERING_ALPHA_DSN"), os.Getenv("MASTERING_DATA_DSN")
	if alphaDSN == "" || dataDSN == "" {
		t.Skip("set MASTERING_ALPHA_DSN and MASTERING_DATA_DSN")
	}
	alpha, data := sqlx.MustConnect("postgres", alphaDSN), sqlx.MustConnect("postgres", dataDSN)
	defer alpha.Close()
	defer data.Close()
	platform := PlatformCatalog{DB: alpha}
	e := &Engine{Data: data, GoldCopy: platform.GoldCopyTenant}
	gold, _ := platform.GoldCopyTenant(context.Background())
	cfg, err := e.loadConfig(context.Background(), gold, "product")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("hierarchy: %v", cfg.hierarchy)
	if got := strings.Join(cfg.hierarchy["NAME"], ">"); got != "REFINITIV>BLOOMBERG>FACTSET" {
		t.Errorf("NAME: %s", got)
	}
	cfg.profile.Settings.FieldGroups = map[string]string{"name": "NAME"}
	cfg.profile.Settings.DefaultFieldGroup = "IDENTITY"
	if r := cfg.rankingFor("name"); len(r) == 0 || r[0] != "REFINITIV" {
		t.Errorf("name ranking: %v", r)
	}
	if r := cfg.rankingFor("domicile"); len(r) == 0 || r[0] != "REFINITIV" {
		t.Errorf("default group ranking: %v", r)
	}
}

// TestSecurityMasterBitemporal masters securities through the generic
// engine on the SECURITY profile (crims 0016, applied inside the rolled-back
// transaction): bitemporal anchor with a stable master_id, identifiers in the
// identifier-issuance layout, derived primary identifier, re-run no-op, and
// a changed value closing the current version and inserting the next.
func TestSecurityMasterBitemporal(t *testing.T) {
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
		Bindings: fixedBinding{"SecName": "security_name", "AssetClass": "asset_class", "AssetCrrncyCd": "currency",
			"SecStatus": "status", "Isin": "isin", "Cusip": "cusip", "SecTypCd": "asset_class",
			"id:ISIN": "isin", "id:CUSIP": "cusip", "id:FIGI": "figi", "@source_key": "source_row_id"}}

	tx, err := data.BeginTxx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback() //nolint:errcheck
	var have bool
	_ = tx.GetContext(ctx, &have, `SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='mdm' AND table_name='security_master' AND column_name='master_id')`)
	if !have {
		if _, err := tx.ExecContext(ctx, crimsDDL(t, "0016_security_master_profile.up.sql")); err != nil {
			t.Fatalf("0016: %v", err)
		}
	}
	if _, err := tx.ExecContext(ctx, `SELECT set_config('app.current_tenant', $1, true)`, gold); err != nil {
		t.Fatal(err)
	}
	// A Bloomberg load of two securities.
	var load string
	if err := tx.GetContext(ctx, &load, `INSERT INTO staging._load_run (source_system_cd, domain, run_ref, status, tenant_id, started_at, completed_at)
		VALUES ('BLOOMBERG', 'SECURITY', 'TEST-SEC-1', 'COMPLETED', $1::uuid, now(), now()) RETURNING id::text`, gold); err != nil {
		t.Fatal(err)
	}
	insert := func(row int, key, isin, name, ccy string) {
		if _, err := tx.ExecContext(ctx, `INSERT INTO staging.security_data (tenant_id, source_system, source_row_id, load_run_id, _load_run_id, _source_row_num,
				security_id, primary_identifier, isin, cusip, figi, security_name, asset_class, currency, status)
			VALUES ($1::uuid, 'BLOOMBERG', $2, $3::uuid, $3::uuid, $4, $2, $5, $5, $6, NULL, $7, 'Equity', $8, 'Active')`,
			gold, key, load, row, isin, "T"+key[len(key)-8:], name, ccy); err != nil {
			t.Fatal(err)
		}
	}
	insert(1, "BBGTEST0001", "XS0000TEST01", "Test Holdings Plc Ordinary", "GBP")
	insert(2, "BBGTEST0002", "XS0000TEST02", "Test Industries AG", "EUR")

	cfg, err := readConfig(ctx, tx, gold, "security")
	if err != nil {
		t.Fatalf("security profile: %v", err)
	}
	run := func(label string) Counts {
		var c Counts
		stage := ""
		r := &runner{e: e, tx: tx, ctx: ctx, tenant: gold, cfg: cfg, p: cfg.profile, run: &Run{ID: uuid.NewString()},
			req: RunRequest{Entity: "security", StagingTable: "staging.security_data", LoadRunID: load}, counts: &c, stage: &stage}
		if err := r.execute(); err != nil {
			t.Fatalf("%s: %v", label, err)
		}
		t.Logf("%s: %+v", label, c)
		return c
	}
	first := run("first")
	var why []string
	_ = tx.SelectContext(ctx, &why, `SELECT exception_type || ': ' || exception_description FROM mdm.security_exception WHERE detected_at > now() - interval '1 minute'`)
	for _, w := range why {
		t.Logf("exception: %s", w)
	}
	if first.New != 2 || first.Published != 2 || first.Invalid != 0 {
		t.Fatalf("first: %+v", first)
	}
	var master, code, primary string
	var versions int
	if err := tx.QueryRowxContext(ctx, `SELECT master_id::text, security_id, primary_identifier,
			(SELECT count(*) FROM mdm.security_master x WHERE x.master_id = s.master_id)
		FROM mdm.security_master s WHERE isin = 'XS0000TEST01' AND valid_to IS NULL`).Scan(&master, &code, &primary, &versions); err != nil {
		t.Fatal(err)
	}
	var idents int
	_ = tx.GetContext(ctx, &idents, `SELECT count(*) FROM mdm.security_identifier_issuance WHERE security_id = $1::uuid AND is_valid`, master)
	t.Logf("minted %s (master %s) primary=%s versions=%d identifiers=%d", code, master, primary, versions, idents)
	if !strings.HasPrefix(code, "SEC-") || primary != "XS0000TEST01" || versions != 1 || idents != 2 {
		t.Errorf("minted security: code=%s primary=%s versions=%d identifiers=%d", code, primary, versions, idents)
	}

	second := run("rerun")
	if second.Xref != 2 || second.Unchanged != 2 || second.Published != 0 {
		t.Errorf("rerun must be a no-op: %+v", second)
	}

	// Bloomberg changes a name: the current version closes, the next inserts.
	if _, err := tx.ExecContext(ctx, `UPDATE staging.security_data SET security_name = 'Test Holdings PLC' WHERE source_row_id = 'BBGTEST0001' AND _load_run_id = $1::uuid`, load); err != nil {
		t.Fatal(err)
	}
	third := run("changed name")
	if third.Published != 1 || third.Unchanged != 1 {
		t.Errorf("changed name: %+v", third)
	}
	var rows, current int
	var name string
	_ = tx.GetContext(ctx, &rows, `SELECT count(*) FROM mdm.security_master WHERE master_id = $1::uuid`, master)
	_ = tx.GetContext(ctx, &current, `SELECT count(*) FROM mdm.security_master WHERE master_id = $1::uuid AND valid_to IS NULL`, master)
	_ = tx.GetContext(ctx, &name, `SELECT security_name FROM mdm.security_master WHERE master_id = $1::uuid AND valid_to IS NULL`, master)
	t.Logf("after change: %d versions, %d current, name %q", rows, current, name)
	if rows != 2 || current != 1 || name != "Test Holdings PLC" {
		t.Errorf("bitemporal versions: rows=%d current=%d name=%q", rows, current, name)
	}
}
