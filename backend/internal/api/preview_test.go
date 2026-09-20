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
