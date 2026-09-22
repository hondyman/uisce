package api

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestPreview_TokenizeAndDerive(t *testing.T) {
	cases := []struct {
		name     string
		colName string
		tableCtx string
		wantSem string
		wantSrc string
	}{
		{
			name:     "address_line_1 with employee table",
			colName:  "address_line_1",
			tableCtx: `table "employee" in schema "orm"`,
			wantSem:  "EmployeeAddress1",
			wantSrc:  "addr_line_context",
		},
		{
			name:     "address_line_2 with issuer_address table",
			colName:  "address_line_2",
			tableCtx: `table "issuer_address" in schema "orm"`,
			wantSem:  "IssuerAddress2",
			wantSrc:  "addr_line_context",
		},
		{
			name:     "account_id no context no abbrev table",
			colName:  "account_id",
			tableCtx: "",
			wantSem:  "AccountId",
			wantSrc:  "pascal",
		},
		{
			name:     "plain pascal case",
			colName:  "balance_amount",
			tableCtx: "",
			wantSem:  "BalanceAmount",
			wantSrc:  "pascal",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			result := deriveTermNamesPreview(context.Background(), nil, "test-tenant", c.colName, c.tableCtx)
			if result.SemanticName != c.wantSem {
				t.Errorf("SemanticName = %q; want %q", result.SemanticName, c.wantSem)
			}
			if result.GetSource() != c.wantSrc {
				t.Errorf("source = %q; want %q", result.GetSource(), c.wantSrc)
			}
		})
	}
}

func TestPreview_WithAbbreviationExpansion(t *testing.T) {
	mockSvc := &mockAbbrevSvcForPreview{
		abbrevs: []map[string]string{
			{"abbreviation": "ID", "full_word": "IDENTIFIER"},
			{"abbreviation": "ACCT", "full_word": "ACCOUNT"},
			{"abbreviation": "ADDR", "full_word": "ADDRESS"},
		},
	}

	cases := []struct {
		name     string
		colName string
		tableCtx string
		wantSem string
		wantSrc string
	}{
		{
			name:     "account_id with ID abbreviation",
			colName:  "account_id",
			tableCtx: `table "fund" in schema "orm"`,
			wantSem:  "AccountIdentifier",
			wantSrc:  "abbrev_map",
		},
		{
			name:     "name not in abbreviation map",
			colName:  "customer_name",
			tableCtx: "",
			wantSem:  "CustomerName",
			wantSrc:  "pascal",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			result := deriveTermNamesPreview(context.Background(), mockSvc, "test-tenant", c.colName, c.tableCtx)
			if result.SemanticName != c.wantSem {
				t.Errorf("SemanticName = %q; want %q", result.SemanticName, c.wantSem)
			}
			if result.GetSource() != c.wantSrc {
				t.Errorf("source = %q; want %q", result.GetSource(), c.wantSrc)
			}
		})
	}
}

func TestPreview_NoLLMCall(t *testing.T) {
	llmCalled := false
	mockSvc := &llmRecordingAbbrevSvc{
		abbrevs: []map[string]string{
			{"abbreviation": "ID", "full_word": "IDENTIFIER"},
		},
		onSuggest: func() { llmCalled = true },
		onQualify:  func() { llmCalled = true },
	}

	deriveTermNamesPreview(context.Background(), mockSvc, "test-tenant", "account_id", "")

	if llmCalled {
		t.Fatal("LLM was called — deriveTermNamesPreview must not call SuggestExpansionsInContext or QualifyGenericWord")
	}
}

func TestPreview_PinnedEmployeeAddressLine1(t *testing.T) {
	result := deriveTermNamesPreview(context.Background(), nil, "test-tenant", "address_line_1", `table "employee" in schema "orm"`)
	if result.SemanticName != "EmployeeAddress1" {
		t.Errorf("EmployeeAddress1: got %q; want %q", result.SemanticName, "EmployeeAddress1")
	}
	if result.BusinessName != "Employee Address Line 1" {
		t.Errorf("EmployeeAddress1 business name: got %q; want %q", result.BusinessName, "Employee Address Line 1")
	}
	if result.GetSource() != "addr_line_context" {
		t.Errorf("EmployeeAddress1 source: got %q; want %q", result.GetSource(), "addr_line_context")
	}
}

func TestPreview_PinnedFundAccountId(t *testing.T) {
	mockSvc := &mockAbbrevSvcForPreview{
		abbrevs: []map[string]string{
			{"abbreviation": "ID", "full_word": "IDENTIFIER"},
			{"abbreviation": "ACCT", "full_word": "ACCOUNT"},
		},
	}
	result := deriveTermNamesPreview(context.Background(), mockSvc, "test-tenant", "account_id", `table "fund" in schema "orm"`)
	if result.SemanticName != "AccountIdentifier" {
		t.Errorf("AccountIdentifier: got %q; want %q", result.SemanticName, "AccountIdentifier")
	}
	if result.GetSource() != "abbrev_map" {
		t.Errorf("AccountIdentifier source: got %q; want %q", result.GetSource(), "abbrev_map")
	}
}

func TestPreview_CapAt2000(t *testing.T) {
	const cap = previewCap
	ids := make([]string, cap+1)
	for i := range ids {
		ids[i] = fmt.Sprintf("00000000-0000-0000-0000-000000000%03d", i)
	}
	svc := &GlossaryService{}
	_, err := svc.PreviewSemanticTerms(context.Background(), "test-tenant", ids)
	if err == nil {
		t.Fatal("expected error for >2000 column_ids")
	}
	if !strings.Contains(err.Error(), "capped") {
		t.Errorf("error = %q; want it to contain 'capped'", err.Error())
	}
}

func TestPreview_PreviewMatchesGenerator(t *testing.T) {
	tableCtx := `table "issuer_address" in schema "orm"`
	result := deriveTermNamesPreview(context.Background(), nil, "test-tenant", "address_line_1", tableCtx)
	if result.SemanticName != "IssuerAddress1" {
		t.Errorf("preview = %q; want %q (must match generator output)", result.SemanticName, "IssuerAddress1")
	}
	if result.BusinessName != "Issuer Address Line 1" {
		t.Errorf("business name = %q; want %q", result.BusinessName, "Issuer Address Line 1")
	}
}

type mockAbbrevSvcForPreview struct {
	abbrevs []map[string]string
}

func (m *mockAbbrevSvcForPreview) GetAllAbbreviations(ctx context.Context) ([]map[string]string, error) {
	return m.abbrevs, nil
}

type llmRecordingAbbrevSvc struct {
	abbrevs    []map[string]string
	onSuggest  func()
	onQualify  func()
}

func (l *llmRecordingAbbrevSvc) GetAllAbbreviations(ctx context.Context) ([]map[string]string, error) {
	return l.abbrevs, nil
}

func (l *llmRecordingAbbrevSvc) SuggestExpansionsInContext(ctx context.Context, tokens []string, rawName, tableCtx string, siblings []string) (map[string]string, error) {
	l.onSuggest()
	return nil, nil
}

func (l *llmRecordingAbbrevSvc) QualifyGenericWord(ctx context.Context, word, tableName string, siblings []string) (string, error) {
	l.onQualify()
	return "", nil
}

func TestPickFirstNonRejected_AllRejected(t *testing.T) {
	candidates := []CandidateTerm{
		{Name: "EmployeeAddress1", Source: "addr_line_context"},
		{Name: "Employee Address Line 1", Source: "addr_line_context"},
		{Name: "Address1", Source: "bare_generic"},
		{Name: "AddressLine1", Source: "pascal"},
	}
	rejections := rejectionSet{
		makeRejectionKey("ds1", "/orm/employee/address_line_1", "EmployeeAddress1"):          struct{}{},
		makeRejectionKey("ds1", "/orm/employee/address_line_1", "Employee Address Line 1"):    struct{}{},
		makeRejectionKey("ds1", "/orm/employee/address_line_1", "Address1"):                   struct{}{},
		makeRejectionKey("ds1", "/orm/employee/address_line_1", "AddressLine1"):                struct{}{},
	}

	got, src := pickFirstNonRejected(candidates, rejections, "ds1", "/orm/employee/address_line_1")
	if got != "AddressLine1" {
		t.Errorf("got %q; want pascal fallback %q", got, "AddressLine1")
	}
	if src != "pascal" {
		t.Errorf("source = %q; want %q", src, "pascal")
	}
}

func TestPickFirstNonRejected_PrimaryRejected_SecondaryWins(t *testing.T) {
	candidates := []CandidateTerm{
		{Name: "EmployeeAddress1", Source: "addr_line_context"},
		{Name: "Employee Address Line 1", Source: "addr_line_context"},
	}
	rejections := rejectionSet{
		makeRejectionKey("ds1", "/orm/employee/address_line_1", "EmployeeAddress1"): struct{}{},
	}

	got, src := pickFirstNonRejected(candidates, rejections, "ds1", "/orm/employee/address_line_1")
	if got != "Employee Address Line 1" {
		t.Errorf("got %q; want secondary %q", got, "Employee Address Line 1")
	}
	if src != "addr_line_context" {
		t.Errorf("source = %q; want %q", src, "addr_line_context")
	}
}

func TestPickFirstNonRejected_NoRejections(t *testing.T) {
	candidates := []CandidateTerm{
		{Name: "EmployeeAddress1", Source: "addr_line_context"},
		{Name: "Employee Address Line 1", Source: "addr_line_context"},
	}
	rejections := rejectionSet{}

	got, _ := pickFirstNonRejected(candidates, rejections, "ds1", "/orm/employee/address_line_1")
	if got != "EmployeeAddress1" {
		t.Errorf("got %q; want primary %q", got, "EmployeeAddress1")
	}
}

func TestPickFirstNonRejected_DifferentDatasourceNotRejected(t *testing.T) {
	candidates := []CandidateTerm{
		{Name: "EmployeeAddress1", Source: "addr_line_context"},
	}
	rejections := rejectionSet{
		// Same column, different datasource — should NOT be rejected
		makeRejectionKey("ds2", "/orm/employee/address_line_1", "EmployeeAddress1"): struct{}{},
	}

	got, _ := pickFirstNonRejected(candidates, rejections, "ds1", "/orm/employee/address_line_1")
	if got != "EmployeeAddress1" {
		t.Errorf("got %q; want primary %q (rejection is ds-scoped)", got, "EmployeeAddress1")
	}
}

func TestDeriveTermNamesCandidates_EmployeeAddress1(t *testing.T) {
	// For employee.address_line_1, the ranked candidates should be:
	// 1. EmployeeAddress1 (primary, contextual)
	// 2. Employee Address Line 1 (business name)
	// 3. Address1 (base generic if different from primary — only if addr_line rule applied)
	// 4. AddressLine1 (naive pascal fallback)
	candidates := deriveTermNamesCandidates(
		[]string{"acknowledged", "line", "1"}, // resolved tokens (no abbreviations)
		"address_line_1",
		`table "employee" in schema "orm"`,
	)
	if len(candidates) < 2 {
		t.Fatalf("got %d candidates; want at least 2", len(candidates))
	}
	if candidates[0].Name != "EmployeeAddress1" {
		t.Errorf("candidates[0] = %q; want %q", candidates[0].Name, "EmployeeAddress1")
	}
	if candidates[0].Source != "addr_line_context" {
		t.Errorf("candidates[0].source = %q; want %q", candidates[0].Source, "addr_line_context")
	}
	// Last candidate must be naive pascal
	last := candidates[len(candidates)-1]
	if last.Source != "pascal" {
		t.Errorf("last candidate source = %q; want %q (must be naive pascal fallback)", last.Source, "pascal")
	}
}

func TestDeriveTermNamesCandidates_AccountIdWithAbbreviation(t *testing.T) {
	// With ID→IDENTIFIER abbreviation: account_id → AccountIdentifier
	candidates := deriveTermNamesCandidates(
		[]string{"ACCOUNT", "IDENTIFIER"},
		"account_id",
		`table "fund" in schema "orm"`,
	)
	if len(candidates) == 0 {
		t.Fatal("got 0 candidates; want at least 1")
	}
	if candidates[0].Name != "AccountIdentifier" {
		t.Errorf("candidates[0] = %q; want %q", candidates[0].Name, "AccountIdentifier")
	}
}

// --- fix: the naive fallback is built from the ORIGINAL tokens -------------------------

func TestDeriveTermNamesCandidates_NaiveFallbackSurvivesAbbreviationExpansion(t *testing.T) {
	// auditor_id with ID->IDENTIFIER. Built from the resolved tokens the naive candidate
	// equalled the primary and was de-duplicated away, leaving [AuditorIdentifier,
	// "Auditor Identifier"] exhausted after one rejection.
	candidates := deriveTermNamesCandidates(
		[]string{"auditor", "IDENTIFIER"}, "auditor_id", `table "fund" in schema "orm"`)

	var names []string
	for _, c := range candidates {
		names = append(names, c.Name)
	}
	want := []string{"AuditorIdentifier", "Auditor Identifier", "AuditorId"}
	if len(names) != len(want) {
		t.Fatalf("candidates = %q; want %q", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("candidates = %q; want %q", names, want)
		}
	}
	if last := candidates[len(candidates)-1]; last.Source != "pascal" {
		t.Errorf("last candidate source = %q; want pascal", last.Source)
	}
}

func TestPickFirstNonRejected_ExhaustedAbbreviatedColumnFallsBackToNaive(t *testing.T) {
	candidates := deriveTermNamesCandidates(
		[]string{"auditor", "IDENTIFIER"}, "auditor_id", `table "fund" in schema "orm"`)
	rejections := rejectionSet{
		makeRejectionKey("ds1", "/orm/fund/auditor_id", "AuditorIdentifier"):  struct{}{},
		makeRejectionKey("ds1", "/orm/fund/auditor_id", "Auditor Identifier"): struct{}{},
		makeRejectionKey("ds1", "/orm/fund/auditor_id", "AuditorId"):          struct{}{},
	}
	got, src := pickFirstNonRejected(candidates, rejections, "ds1", "/orm/fund/auditor_id")
	if got != "AuditorId" || src != "pascal" {
		t.Errorf("got (%q, %q); want (AuditorId, pascal)", got, src)
	}
}

func TestPickFirstNonRejected_ExhaustedNeverReturnsSpacedName(t *testing.T) {
	// The naive candidate was de-duplicated away, so the list ends with a spaced business
	// name. Exhausting it must not hand that back as the last resort.
	candidates := []CandidateTerm{
		{Name: "CountryCode", Source: "abbrev_map"},
		{Name: "Country Code", Source: "abbrev_map"},
	}
	rejections := rejectionSet{
		makeRejectionKey("ds1", "/orm/issuer/country_cd", "CountryCode"):  struct{}{},
		makeRejectionKey("ds1", "/orm/issuer/country_cd", "Country Code"): struct{}{},
	}
	got, src := pickFirstNonRejected(candidates, rejections, "ds1", "/orm/issuer/country_cd")
	if got != "CountryCode" || src != "pascal" {
		t.Errorf("got (%q, %q); want (CountryCode, pascal)", got, src)
	}
}

// --- fix: the abbreviation table is loaded once per request, not once per column -------

type countingAbbrevSvc struct {
	calls   int
	abbrevs []map[string]string
	err     error
}

func (c *countingAbbrevSvc) GetAllAbbreviations(ctx context.Context) ([]map[string]string, error) {
	c.calls++
	return c.abbrevs, c.err
}

func TestBuildAbbreviationMap_LoadsOnceForManyColumns(t *testing.T) {
	svc := &countingAbbrevSvc{abbrevs: []map[string]string{
		{"abbreviation": "ID", "full_word": "IDENTIFIER"},
		{"abbreviation": "CD", "full_word": "CODE"},
	}}
	abbrMap := buildAbbreviationMap(context.Background(), svc, "tenant-1")
	cols := []string{"account_id", "country_cd", "fund_id", "auditor_id", "address_line_1"}
	for i := 0; i < 2000; i++ {
		_ = deriveTermNamesPreviewCandidatesWithMap(abbrMap, cols[i%len(cols)], `table "fund" in schema "orm"`)
	}
	if svc.calls != 1 {
		t.Errorf("abbreviations loaded %d times for 2000 columns; want exactly 1", svc.calls)
	}
}

func TestPreviewCandidatesWithMap_MatchesPerColumnLoad(t *testing.T) {
	svc := &countingAbbrevSvc{abbrevs: []map[string]string{
		{"abbreviation": "ID", "full_word": "IDENTIFIER"},
		{"abbreviation": "QTY", "full_word": "QUANTITY"},
	}}
	abbrMap := buildAbbreviationMap(context.Background(), svc, "t")
	for _, col := range []string{"account_id", "min_order_qty", "address_line_1", "city", "id"} {
		want := deriveTermNamesPreviewCandidates(context.Background(), svc, "t", col, `table "x" in schema "orm"`)
		got := deriveTermNamesPreviewCandidatesWithMap(abbrMap, col, `table "x" in schema "orm"`)
		if len(got) != len(want) {
			t.Fatalf("%s: got %d candidates, want %d", col, len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("%s candidate %d: got %+v, want %+v", col, i, got[i], want[i])
			}
		}
	}
}

func TestBuildAbbreviationMap_NilServiceOrErrorDegradesGracefully(t *testing.T) {
	if m := buildAbbreviationMap(context.Background(), nil, "t"); len(m) != 0 {
		t.Errorf("nil service: got %d entries, want 0", len(m))
	}
	errSvc := &countingAbbrevSvc{err: context.DeadlineExceeded}
	if m := buildAbbreviationMap(context.Background(), errSvc, "t"); len(m) != 0 {
		t.Errorf("failing service: got %d entries, want 0", len(m))
	}
	// Unexpanded tokens still yield a usable name.
	c := deriveTermNamesPreviewCandidatesWithMap(nil, "account_id", `table "fund" in schema "orm"`)
	if len(c) == 0 || c[0].Name == "" {
		t.Errorf("no candidates without an abbreviation map: %+v", c)
	}
}
