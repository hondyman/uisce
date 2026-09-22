package metadata

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompilePolyglotQuery_Dialects(t *testing.T) {
	req := PolyglotQueryRequest{
		TenantID:  "tenant-1",
		TableName: "trades",
		Dialect:   DialectStarRocks,
		SelectColumns: []string{"id", "price"},
		WhereClause: "price > 100",
		OrderBy: "id ASC",
		Limit: 10,
	}

	// StarRocks compilation
	res, err := CompilePolyglotQuery(req)
	require.NoError(t, err)
	assert.Contains(t, res.SQL, "SELECT id, price FROM trades")
	assert.Contains(t, res.SQL, "WHERE price > 100")
	assert.Equal(t, DialectStarRocks, res.Dialect)

	// DataFusion compilation
	req.Dialect = DialectDataFusion
	res, err = CompilePolyglotQuery(req)
	require.NoError(t, err)
	assert.Contains(t, res.SQL, "SELECT id, price FROM trades")
	assert.Equal(t, DialectDataFusion, res.Dialect)

	// Iceberg compilation
	req.Dialect = DialectIceberg
	res, err = CompilePolyglotQuery(req)
	require.NoError(t, err)
	assert.Contains(t, res.SQL, "SELECT id, price FROM trades")
	assert.Equal(t, DialectIceberg, res.Dialect)
}
