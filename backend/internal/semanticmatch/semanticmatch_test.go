package semanticmatch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenizeAndExpand(t *testing.T) {
	toks := Tokenize("cash_acct_no")
	assert.Equal(t, []string{"cash", "acct", "no"}, toks)

	expanded := ExpandTokens(toks)
	assert.Equal(t, []string{"cash", "account", "number"}, expanded)

	toksCamel := Tokenize("settleDateUSD")
	assert.Equal(t, []string{"settle", "date", "usd"}, toksCamel)
	expandedCamel := ExpandTokens(toksCamel)
	assert.Equal(t, []string{"settlement", "date", "usd"}, expandedCamel)
}

func TestParseAliasesAndRange(t *testing.T) {
	aliases := ParseAliases("broker_cd / bkr_cd; est_settle_cash0…est_settle_cash3")
	expected := []string{
		"broker_cd",
		"bkr_cd",
		"est_settle_cash0",
		"est_settle_cash1",
		"est_settle_cash2",
		"est_settle_cash3",
	}
	assert.Equal(t, expected, aliases)
}

func TestTypeFamilyCompatibility(t *testing.T) {
	assert.Equal(t, FamilyNumeric, ParseTypeFamily("NUMERIC(18,4) NOT NULL"))
	assert.Equal(t, FamilyString, ParseTypeFamily("VARCHAR(100)"))
	assert.Equal(t, FamilyDateTime, ParseTypeFamily("TIMESTAMPTZ"))
	assert.Equal(t, FamilyBool, ParseTypeFamily("BOOLEAN"))

	assert.Equal(t, 1.0, TypeCompatibility(FamilyNumeric, FamilyNumeric))
	assert.Equal(t, 0.9, TypeCompatibility(FamilyNumeric, FamilyInteger))
	assert.Equal(t, 0.6, TypeCompatibility(FamilyBool, FamilyString))
}

func TestSuggestTermName(t *testing.T) {
	assert.Equal(t, "Cash Reserve Percent", SuggestTermName("cash_rsrv_pct"))
	assert.Equal(t, "Broker Identifier", SuggestTermName("bkr_id"))
}

func TestPipelineLexicalOnly(t *testing.T) {
	reg := NewTermRegistry()

	// Seed term
	reg.AddTerm(&SemanticTerm{
		ID:       "glossary:1",
		Source:   "glossary",
		Name:     "Trade Cash Account Number",
		Aliases:  []string{"cash_acct_no"},
		Domain:   "Trading",
		RawType:  "VARCHAR(40)",
	}, []string{"trade_order"})

	// Another term
	reg.AddTerm(&SemanticTerm{
		ID:          "bb:PX_LAST",
		Source:      "bloomberg",
		Mnemonic:    "PX_LAST",
		Name:        "Last Price",
		Description: "Last traded price of the security",
		Domain:      "Pricing",
		RawType:     "Real",
	}, nil)

	cfg := DefaultConfig()
	pipe := NewPipeline(cfg, reg, nil)

	cols := []ColumnMeta{
		{
			Table:    "trade_order",
			Column:   "cash_acct_no",
			DataType: "VARCHAR(40)",
		},
		{
			Table:    "quote",
			Column:   "px_last",
			DataType: "NUMERIC(18,4)",
		},
		{
			Table:    "unknown",
			Column:   "completely_random_col_xyz",
			DataType: "TEXT",
		},
	}

	outcomes, err := pipe.Run(context.Background(), cols)
	require.NoError(t, err)
	require.Len(t, outcomes, 3)

	// col 0: Curated seed match
	assert.Equal(t, StatusAutoLinked, outcomes[0].Status)
	assert.Equal(t, MethodSeed, outcomes[0].Method)
	assert.Equal(t, "glossary:1", outcomes[0].Term.ID)

	// col 1: Exact mnemonic / lexical match
	assert.Equal(t, StatusAutoLinked, outcomes[1].Status)
	assert.Equal(t, "bb:PX_LAST", outcomes[1].Term.ID)

	// col 2: No match
	assert.Equal(t, StatusNoMatch, outcomes[2].Status)
	assert.Equal(t, "Completely Random Col Xyz", outcomes[2].SuggestedNewTerm)
}

func TestSuggestAndHandler(t *testing.T) {
	reg := NewTermRegistry()
	reg.AddTerm(&SemanticTerm{
		ID:       "glossary:1",
		Source:   "glossary",
		Name:     "Broker Code",
		Aliases:  []string{"bkr_cd"},
		Domain:   "Trading",
		RawType:  "VARCHAR(10)",
	}, []string{"ts_order_broker"})

	pipe := NewPipeline(DefaultConfig(), reg, nil)

	cands, oc, err := pipe.Suggest(context.Background(), ColumnMeta{
		Table:    "ts_order_broker",
		Column:   "bkr_cd",
		DataType: "VARCHAR(10)",
	})
	require.NoError(t, err)
	assert.Equal(t, StatusAutoLinked, oc.Status)
	assert.NotEmpty(t, cands)

	handler := SuggestHandler(pipe)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/semantic-match/suggest",
		strings.NewReader(`{"table":"ts_order_broker","column":"bkr_cd","data_type":"VARCHAR(10)"}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Broker Code")
}
