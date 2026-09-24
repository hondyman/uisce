package api

import (
	"context"
	"database/sql/driver"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

var testAbbr = map[string]string{"ID": "IDENTIFIER", "CD": "CODE", "ACCT": "ACCOUNT", "TYP": "TYPE", "SEC": "SECURITY"}

func TestCanonicalTermKey_SameMeaningSameKey(t *testing.T) {
	same := [][]string{
		{"TenantId", "TenantIdentifier", "tenant_id", "Tenant_ID", "TENANT_IDENTIFIER"},
		{"IssuerId", "IssuerIdentifier"},
		{"AcctCd", "AccountCode", "account_cd"},
		{"SecTypCd", "SecurityTypeCode"},
	}
	for _, group := range same {
		want := canonicalTermKey(group[0], testAbbr)
		for _, name := range group[1:] {
			if got := canonicalTermKey(name, testAbbr); got != want {
				t.Errorf("%q key %q; want the same as %q (%q)", name, got, group[0], want)
			}
		}
	}
}

func TestCanonicalTermKey_DifferentMeaningsStayApart(t *testing.T) {
	pairs := [][2]string{
		{"TenantId", "Tenant"},
		{"IssuerId", "IssuerCode"},
		{"SecuritiesTypeCode", "SecuritiesSubordinatedTypeCode"},
		{"AccountId", "AccountName"},
	}
	for _, p := range pairs {
		if canonicalTermKey(p[0], testAbbr) == canonicalTermKey(p[1], testAbbr) {
			t.Errorf("%q and %q must not be treated as the same term", p[0], p[1])
		}
	}
}

// Without a dictionary the two most common spellings still compare equal.
func TestCanonicalTermKey_FallsBackToBuiltinSynonyms(t *testing.T) {
	if canonicalTermKey("TenantId", nil) != canonicalTermKey("TenantIdentifier", nil) {
		t.Error("Id and Identifier must match even with no abbreviation dictionary")
	}
	if canonicalTermKey("AcctCd", nil) == canonicalTermKey("AccountCode", nil) {
		t.Error("only id/cd are built in; Acct is dictionary-only")
	}
}

func TestTermIndex_FirstAddedWinsAndReuseMapsTwinsOntoIt(t *testing.T) {
	ix := newTermIndex()
	ix.add("TenantId", testAbbr) // best first: the term with the most mapped columns
	ix.add("TenantIdentifier", testAbbr)
	ix.add("IssuerIdentifier", testAbbr)

	if got, ok := ix.reuse("TenantIdentifier", testAbbr); !ok || got != "TenantId" {
		t.Errorf("reuse(TenantIdentifier) = %q, %v; want TenantId", got, ok)
	}
	if got, ok := ix.reuse("tenant_id", testAbbr); !ok || got != "TenantId" {
		t.Errorf("reuse(tenant_id) = %q, %v; want TenantId", got, ok)
	}
	if got, ok := ix.reuse("IssuerId", testAbbr); !ok || got != "IssuerIdentifier" {
		t.Errorf("reuse(IssuerId) = %q, %v; want the only existing issuer term", got, ok)
	}
	if _, ok := ix.reuse("BenchmarkId", testAbbr); ok {
		t.Error("a name with no existing equivalent must not be reused")
	}
	var nilIndex *termIndex
	if _, ok := nilIndex.reuse("X", nil); ok {
		t.Error("a nil index reuses nothing")
	}
}

func TestLoadTermIndex_UsesTheOrderTheDatabaseReturns(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	// the query orders by mapped-column count, so the best term comes first
	mock.ExpectQuery(`(?s)FROM catalog_node n.*catalog_type_name = 'semantic_term'.*ORDER BY COALESCE\(m\.mapped, 0\) DESC`).
		WithArgs("t1").
		WillReturnRows(sqlmock.NewRows([]string{"node_name"}).AddRow("TenantId").AddRow("TenantIdentifier"))

	ix := (&GlossaryService{db: db}).loadTermIndex(context.Background(), "t1", testAbbr)
	if got, _ := ix.reuse("TenantIdentifier", testAbbr); got != "TenantId" {
		t.Errorf("reuse = %q; want TenantId", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestLoadTermIndex_FailureFallsBackToNoReuseNotAnError(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	mock.ExpectQuery(`FROM catalog_node`).WillReturnError(errBoom{})
	ix := (&GlossaryService{db: db}).loadTermIndex(context.Background(), "t1", nil)
	if _, ok := ix.reuse("TenantId", nil); ok {
		t.Error("a failed load must yield an empty index")
	}
	if got := (&GlossaryService{}).loadTermIndex(context.Background(), "t1", nil); got == nil {
		t.Error("no database must yield an empty index, not nil")
	}
}

// The wizard proposes an existing term instead of a twin.
func TestPreviewSemanticTerms_ProposesTheExistingTermInsteadOfATwin(t *testing.T) {
	// the production driver accepts a []string for ANY($1); sqlmock's default converter does not
	db, mock, err := sqlmock.New(sqlmock.ValueConverterOption(passThroughConverter{}))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	const col = "11111111-1111-1111-1111-111111111111"
	mock.ExpectQuery(`FROM catalog_node\s+WHERE id = ANY`).WillReturnRows(
		sqlmock.NewRows([]string{"id", "node_name", "path", "ds"}).AddRow(col, "tenant_id", "/mdm/security_type/tenant_id", "22222222-2222-2222-2222-222222222222"))
	mock.ExpectQuery(`semantic_term_rejections`).WillReturnRows(sqlmock.NewRows([]string{"a", "b", "c"}))
	// The existing term is spelled the other way from what the deterministic path derives for tenant_id
	// ("TenantId"): the wizard must still propose the existing term, not a second spelling of it.
	mock.ExpectQuery(`catalog_type_name = 'semantic_term'`).WillReturnRows(sqlmock.NewRows([]string{"node_name"}).AddRow("TenantIdentifier"))

	res, err := (&GlossaryService{db: db}).PreviewSemanticTerms(context.Background(), "t1", []string{col})
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 || res[0].SemanticName != "TenantIdentifier" {
		t.Fatalf("got %+v; want the existing term TenantIdentifier", res)
	}
	if res[0].Source != "existing_term" {
		t.Errorf("source = %q; want existing_term", res[0].Source)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

type errBoom struct{}

func (errBoom) Error() string { return "boom" }

type passThroughConverter struct{}

func (passThroughConverter) ConvertValue(v interface{}) (driver.Value, error) { return v, nil }

func TestSanitizeExpansion_KeepsNamesPlainPascalCase(t *testing.T) {
	cases := map[string]string{
		"Service_level_agreement": "ServiceLevelAgreement",
		"service level agreement": "ServiceLevelAgreement",
		"Corporate-Action":        "CorporateAction",
		"Security":                "Security",
		"  Type ":                 "Type",
		"":                        "",
		"___":                     "",
	}
	for in, want := range cases {
		if got := sanitizeExpansion(in); got != want {
			t.Errorf("sanitizeExpansion(%q) = %q; want %q", in, got, want)
		}
	}
}
