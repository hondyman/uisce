package bundles

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func TestEvaluateGuardrails_DBFallback(t *testing.T) {
	// Guardrails are DB-only. This test used to write a guardrails.yaml into
	// the working directory and assert the YAML fallback picked it up, which
	// made the suite order-dependent: the file it wrote could still be present
	// when a sibling test asserted no file existed, and os.Remove failures were
	// silently discarded. The DB row below is the whole configuration path.
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "postgres")

	rows := sqlmock.NewRows([]string{"type", "data"}).
		AddRow("sod", []byte(`{"pairs":[["orders_write","billing_view"],["orders_write","customer_segment"]]}`)).
		AddRow("certified", []byte(`{"claims":["billing_view"]}`))
	mock.ExpectQuery(`FROM guardrail_rules`).WillReturnRows(rows)

	cfg, src, err := loadGuardrails(sqlxDB)
	if err != nil {
		t.Fatalf("loadGuardrails error: %v", err)
	}
	if src != "db" {
		t.Fatalf("expected source db, got %q", src)
	}
	if len(cfg.SoDPairs) != 2 || len(cfg.Certified) != 1 {
		t.Fatalf("unexpected config: %+v", cfg)
	}

	// build a proposal details payload with conflicting claims
	details := map[string]interface{}{
		"claims":      []string{"orders_write", "billing_view"},
		"description": "test",
	}
	b, _ := json.Marshal(details)

	// evaluateGuardrails reads the process-global guardrails cache, so seed it
	// explicitly rather than relying on a previous test's load. A test that
	// depends on the order other tests happened to run in is the same class of
	// defect as the filesystem coupling this file used to have.
	guardrailsMutex.Lock()
	guardrailsCache = &GuardrailCache{Config: cfg, LastLoaded: time.Now(), Source: src}
	guardrailsMutex.Unlock()

	ok, reasons, err := evaluateGuardrails(sqlxDB, b)
	if err != nil {
		t.Fatalf("evaluateGuardrails returned error: %v", err)
	}
	if ok {
		t.Fatalf("expected guardrail to fail but passed")
	}
	if len(reasons) == 0 {
		t.Fatalf("expected reasons for failure")
	}
}

func TestLoadGuardrails_NoDB(t *testing.T) {
	// No DB means no guardrails, and nothing is read from disk. The previous
	// version of this test deleted a guardrails.yaml and asserted it was
	// absent, which coupled the test to the working directory and to whether
	// os.Remove was permitted.
	cfg, src, err := loadGuardrails(nil)
	if err != nil {
		t.Fatalf("loadGuardrails error: %v", err)
	}
	if src != "none" {
		t.Fatalf("expected source %q with no DB, got %q", "none", src)
	}
	if cfg == nil {
		t.Fatalf("expected non-nil cfg")
	}
	if len(cfg.SoDPairs) != 0 || len(cfg.Certified) != 0 {
		t.Fatalf("expected empty guardrails when no config present")
	}
}
