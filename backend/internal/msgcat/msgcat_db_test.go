package msgcat

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// Integration tests against a real Postgres, as a non-superuser so row-level
// security applies. MSGCAT_TEST_DSN is a superuser DSN for a disposable
// database; the tests rebuild the catalog tables in it.
const (
	tenantA = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	tenantB = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
)

func testDB(t *testing.T) *sqlx.DB {
	t.Helper()
	dsn := os.Getenv("MSGCAT_TEST_DSN")
	if dsn == "" {
		t.Skip("MSGCAT_TEST_DSN not set")
	}
	admin := sqlx.MustConnect("postgres", dsn)
	defer admin.Close()
	for _, stmt := range []string{
		`DROP TABLE IF EXISTS public.message_catalog_changes, public.tenant_message_catalog, public.message_catalog, public.message_sets, public.tenants`,
		`DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'app_user') THEN CREATE ROLE app_user LOGIN PASSWORD 'app_user'; END IF; END $$`,
		`CREATE TABLE public.tenants (id uuid PRIMARY KEY, settings jsonb)`,
		`GRANT SELECT ON public.tenants TO app_user`,
		`GRANT USAGE ON SCHEMA public TO app_user`,
	} {
		admin.MustExec(stmt)
	}
	for _, f := range []string{"20261027_001_message_catalog.up.sql", "20261027_002_message_catalog_seed.up.sql"} {
		sql, err := os.ReadFile("../../db/migrations/" + f)
		if err != nil {
			t.Fatal(err)
		}
		admin.MustExec(string(sql))
	}
	// Tenant A edits without a second approver; B uses the default (on).
	admin.MustExec(`INSERT INTO public.tenants VALUES ($1, '{"msgcat": {"maker_checker": false}}'), ($2, '{}')`, tenantA, tenantB)

	u, _ := url.Parse(dsn)
	u.User = url.UserPassword("app_user", "app_user")
	db := sqlx.MustConnect("postgres", u.String())
	t.Cleanup(func() { db.Close() })
	return db
}

func setup(t *testing.T) (*Editor, *Catalog, *Store) {
	db := testDB(t)
	store := NewStore(db)
	cat := NewCatalog(store)
	return NewEditor(store, cat), cat, store
}

var (
	coreAdmin1 = Actor{UserID: "admin-1", Name: "admin1@example.com", PlatformAdmin: true}
	coreAdmin2 = Actor{UserID: "admin-2", Name: "admin2@example.com", PlatformAdmin: true}
	aAdmin     = Actor{UserID: "a-admin", TenantID: tenantA, TenantAdmin: true}
	bAdmin1    = Actor{UserID: "b-admin-1", TenantID: tenantB, TenantAdmin: true}
	bAdmin2    = Actor{UserID: "b-admin-2", TenantID: tenantB, TenantAdmin: true}
	bUser      = Actor{UserID: "b-user", TenantID: tenantB}
)

func wantCode(t *testing.T, err error, code string) {
	t.Helper()
	var me *Error
	if !errors.As(err, &me) || me.Code() != code {
		t.Fatalf("err = %v, want catalog message %s", err, code)
	}
}

func text(t *testing.T, c *Catalog, tenantID, lang string, set, nbr int) string {
	t.Helper()
	c.Invalidate("")
	c.Invalidate(tenantID)
	e, ok := c.Lookup(context.Background(), tenantID, Preferences(lang), set, nbr)
	if !ok {
		t.Fatalf("%d-%d not found", set, nbr)
	}
	return e.Text
}

func TestSeed_LegacyAndTranslations(t *testing.T) {
	_, cat, _ := setup(t)
	if got := text(t, cat, "", "fr", 1, 4); !strings.Contains(got, "erreur interne") {
		t.Errorf("1-4 fr = %q", got)
	}
	if got := text(t, cat, "", "ja", 1, 4); !strings.HasPrefix(got, "An internal error") {
		t.Errorf("1-4 ja falls back to English, got %q", got)
	}
}

// Core edits always need a second platform administrator.
func TestCore_MakerChecker(t *testing.T) {
	ed, cat, _ := setup(t)
	ctx := context.Background()
	res, err := ed.Propose(ctx, coreAdmin1, []ChangeRequest{{Scope: ScopeCore, SetNbr: 1, MessageNbr: 6, Language: "en", Severity: "Error", Text: "We couldn't find %1."}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Applied || res.Changes[0].Status != "pending" {
		t.Fatalf("core change applied without approval: %+v", res)
	}
	if got := text(t, cat, "", "en", 1, 6); got != "%1 was not found." {
		t.Fatalf("pending change is live: %q", got)
	}
	id := res.Changes[0].ID
	_, err = ed.Decide(ctx, coreAdmin1, id, true, "")
	wantCode(t, err, "9100-16")
	_, err = ed.Decide(ctx, aAdmin, id, true, "")
	wantCode(t, err, "9100-14") // not the tenant admin's to see
	c, err := ed.Decide(ctx, coreAdmin2, id, true, "ok")
	if err != nil || c.Status != "applied" || c.ReviewedBy.String != "admin-2" ||
		c.RequestedName.String != "admin1@example.com" || c.ReviewedName.String != "admin2@example.com" {
		t.Fatalf("c=%+v err=%v", c, err)
	}
	if got := text(t, cat, "", "en", 1, 6); got != "We couldn't find %1." {
		t.Fatalf("approved change not live: %q", got)
	}
	_, err = ed.Decide(ctx, coreAdmin2, id, true, "")
	wantCode(t, err, "9100-15")
}

func TestPermissions(t *testing.T) {
	ed, _, _ := setup(t)
	ctx := context.Background()
	_, err := ed.Propose(ctx, aAdmin, []ChangeRequest{{Scope: ScopeCore, SetNbr: 1, MessageNbr: 6, Language: "en", Severity: "Error", Text: "x %1"}})
	wantCode(t, err, "9100-11")
	_, err = ed.Propose(ctx, bUser, []ChangeRequest{{SetNbr: 1, MessageNbr: 6, Language: "en", Severity: "Error", Text: "x %1"}})
	wantCode(t, err, "9100-12")
}

func TestValidation(t *testing.T) {
	ed, _, _ := setup(t)
	ctx := context.Background()
	cases := []struct {
		r    ChangeRequest
		code string
	}{
		{ChangeRequest{SetNbr: 1, MessageNbr: 6, Language: "fr", Text: "Introuvable."}, "9100-5"}, // drops %1
		{ChangeRequest{SetNbr: 1, MessageNbr: 6, Language: "xx", Text: "x"}, "9100-1"},
		{ChangeRequest{SetNbr: 1, MessageNbr: 6, Language: "en", Severity: "Critical", Text: "%1"}, "9100-2"},
		{ChangeRequest{SetNbr: 1, MessageNbr: 6, Language: "en", Severity: "Error", Text: "  "}, "9100-3"},
		{ChangeRequest{SetNbr: 1, MessageNbr: 6, Language: "en", Severity: "Error", Text: strings.Repeat("x", 1001)}, "9100-4"},
		{ChangeRequest{SetNbr: 4242, MessageNbr: 1, Language: "en", Severity: "Error", Text: "x"}, "9100-7"},
		{ChangeRequest{SetNbr: 1, MessageNbr: 500, Language: "en", Severity: "Error", Text: "new"}, "9100-10"}, // tenants add only to client sets
		{ChangeRequest{SetNbr: 20000, MessageNbr: 1, Language: "fr", Text: "Nouveau"}, "9100-6"},               // English first
		{ChangeRequest{SetNbr: 1, MessageNbr: 6, Language: "de", Action: "delete"}, "9100-19"},
	}
	for _, c := range cases {
		_, err := ed.Propose(ctx, bAdmin1, []ChangeRequest{c.r})
		wantCode(t, err, c.code)
	}
	_, err := ed.Propose(ctx, coreAdmin1, []ChangeRequest{{Scope: ScopeCore, SetNbr: 1, MessageNbr: 6, Language: "en", Action: "delete"}})
	wantCode(t, err, "9100-13")
}

// Tenant A has maker-checker off: its edits apply at once, override the core
// text for that tenant only, and a user's language wins over tenant-ness.
func TestTenant_DirectApplyAndResolution(t *testing.T) {
	ed, cat, _ := setup(t)
	ctx := context.Background()
	res, err := ed.Propose(ctx, aAdmin, []ChangeRequest{
		{SetNbr: 20000, MessageNbr: 1, Language: "en", Severity: "Warning", Text: "Trade %1 breaches the desk limit.", UserAction: "Ask the desk head to approve %1."},
		{SetNbr: 20000, MessageNbr: 1, Language: "fr", Text: "L'opération %1 dépasse la limite du desk."},
		{SetNbr: 1, MessageNbr: 6, Language: "en", Severity: "Error", Text: "Northwind could not find %1."},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Applied || res.Changes[0].Status != "applied" {
		t.Fatalf("%+v", res)
	}
	if got := text(t, cat, tenantA, "fr", 20000, 1); got != "L'opération %1 dépasse la limite du desk." {
		t.Errorf("fr = %q", got)
	}
	if got := text(t, cat, tenantA, "en", 1, 6); got != "Northwind could not find %1." {
		t.Errorf("override = %q", got)
	}
	if got := text(t, cat, tenantA, "fr", 1, 6); got != "%1 est introuvable." {
		t.Errorf("fr user gets the core French over the tenant's English, got %q", got)
	}
	if got := text(t, cat, tenantB, "en", 1, 6); got != "%1 was not found." {
		t.Errorf("tenant B sees A's override: %q", got)
	}
	// Severity is the message's: the French row took the English severity.
	e, _ := cat.Lookup(ctx, tenantA, []string{"fr"}, 20000, 1)
	if e.Severity != "Warning" {
		t.Errorf("fr severity = %q", e.Severity)
	}
	// English can't go while translations depend on it.
	_, err = ed.Propose(ctx, aAdmin, []ChangeRequest{{SetNbr: 20000, MessageNbr: 1, Language: "en", Action: "delete"}})
	wantCode(t, err, "9100-21")
	if _, err = ed.Propose(ctx, aAdmin, []ChangeRequest{
		{SetNbr: 20000, MessageNbr: 1, Language: "fr", Action: "delete"},
		{SetNbr: 20000, MessageNbr: 1, Language: "en", Action: "delete"},
	}); err != nil {
		t.Fatal(err)
	}
}

// Tenant B needs a second approver; approval re-checks the row it proposed
// against, and one pending change per message and language at a time.
func TestTenant_MakerCheckerStaleAndConflicts(t *testing.T) {
	ed, cat, store := setup(t)
	ctx := context.Background()
	res, err := ed.Propose(ctx, bAdmin1, []ChangeRequest{{SetNbr: 1, MessageNbr: 6, Language: "en", Severity: "Error", Text: "B cannot find %1."}})
	if err != nil || res.Applied {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	id := res.Changes[0].ID
	_, err = ed.Propose(ctx, bAdmin2, []ChangeRequest{{SetNbr: 1, MessageNbr: 6, Language: "en", Severity: "Error", Text: "Other %1"}})
	wantCode(t, err, "9100-20")
	_, err = ed.Withdraw(ctx, bAdmin2, id)
	wantCode(t, err, "9100-18")
	_, err = ed.Decide(ctx, aAdmin, id, true, "")
	wantCode(t, err, "9100-14") // another tenant's change is invisible

	// Someone edits the row after the proposal (here: directly).
	if err := store.inTenant(ctx, tenantB, func(tx *sqlx.Tx) error {
		_, err := tx.Exec(`INSERT INTO public.tenant_message_catalog (tenant_id, set_nbr, message_nbr, language_cd, severity, message_text) VALUES ($1, 1, 6, 'en', 'Error', 'Changed meanwhile %1')`, tenantB)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	_, err = ed.Decide(ctx, bAdmin2, id, true, "")
	wantCode(t, err, "9100-17")
	if got := text(t, cat, tenantB, "en", 1, 6); got != "Changed meanwhile %1" {
		t.Fatalf("stale change overwrote a newer edit: %q", got)
	}
	c, err := ed.Decide(ctx, bAdmin2, id, false, "stale")
	if err != nil || c.Status != "rejected" {
		t.Fatalf("c=%+v err=%v", c, err)
	}
}

// Tenant rows are protected by RLS, not only by the application's filters.
func TestRLS_TenantRowsNeedTenantContext(t *testing.T) {
	ed, _, store := setup(t)
	ctx := context.Background()
	if _, err := ed.Propose(ctx, aAdmin, []ChangeRequest{{SetNbr: 1, MessageNbr: 6, Language: "en", Severity: "Error", Text: "A %1"}}); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := store.db.Get(&n, `SELECT count(*) FROM public.tenant_message_catalog`); err != nil || n != 0 {
		t.Fatalf("rows visible without tenant context: n=%d err=%v", n, err)
	}
	if err := store.inTenant(ctx, tenantB, func(tx *sqlx.Tx) error {
		return tx.Get(&n, `SELECT count(*) FROM public.tenant_message_catalog`)
	}); err != nil || n != 0 {
		t.Fatalf("tenant B sees A's rows: n=%d err=%v", n, err)
	}
}

func TestWriteError_EnvelopeNeverLeaksCause(t *testing.T) {
	_, cat, _ := setup(t)
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Accept-Language", "fr-CA,fr;q=0.9")
	req.Header.Set("X-Request-ID", "req-123")
	rec := httptest.NewRecorder()
	cat.WriteError(rec, req, tenantB, errors.New(`pq: duplicate key value violates unique constraint "secret_idx"`))
	var body ErrorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if rec.Code != 500 || body.Code != "1-4" || body.CorrelationID != "req-123" || body.Language != "fr" ||
		!strings.Contains(body.Error, "req-123") || strings.Contains(rec.Body.String(), "secret_idx") {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req.Header.Set("Accept-Language", "es")
	req.Header.Set("X-Request-ID", "bad id with spaces")
	cat.WriteError(rec, req, tenantB, msg(7, 4242).WithStatus(404).Wrap(errors.New("internal detail")))
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if rec.Code != 404 || body.Code != "9100-7" || body.Error != "No se encontró el conjunto de mensajes 4242." ||
		body.CorrelationID == "bad id with spaces" || strings.Contains(rec.Body.String(), "internal detail") {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	// A code missing from the catalog still renders safely.
	rec = httptest.NewRecorder()
	cat.WriteError(rec, req, "", New(777, 1, "x"))
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if rec.Code != 500 || body.Code != "1-4" {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
