package metadata

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestQuoteIdentAndStringLiteral(t *testing.T) {
	require.Equal(t, `"subtype_code"`, quoteIdent("subtype_code"))
	require.Equal(t, `"weird""name"`, quoteIdent(`weird"name`))
	require.Equal(t, `'wealth_account'`, quoteStringLiteral("wealth_account"))
	require.Equal(t, `'O''Brien'`, quoteStringLiteral("O'Brien"))
}

func TestSTIFenceSQL(t *testing.T) {
	fence := "WHERE " + quoteIdent("subtype_code") + " = " + quoteStringLiteral("wealth_account")
	require.Equal(t, `WHERE "subtype_code" = 'wealth_account'`, fence)
}
