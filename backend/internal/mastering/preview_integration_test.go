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
	if second.Xref != first.New+first.Deterministic+first.Fuzzy || second.New != 0 || second.Published != 0 || second.Unchanged != first.Published {
		t.Errorf("rerun must be a no-op: first %+v second %+v", first, second)
	}
	t.Logf("rekeyed %+v", rekeyed)
	if rekeyed.Deterministic != first.New || rekeyed.New != 0 {
		t.Errorf("records under new keys must match by identifier: %+v", rekeyed)
	}
	if second.Exceptions != 0 {
		t.Errorf("open exceptions must not be raised twice: %d", second.Exceptions)
	}
}
