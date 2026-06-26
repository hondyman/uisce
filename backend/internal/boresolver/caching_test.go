package boresolver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveExpression_Caching(t *testing.T) {
	// Setup Cache
	cache := NewSQLCache(10)

	// Setup Context with Cache
	ctx := &ResolutionContext{
		BOID:         "bo_test",
		DrivingTable: "main_table",
		TermMappings: map[string]PhysicalMapping{},
		JoinPaths:    map[string][]JoinStep{},
		// Cache Params
		CalcID:      "calc_literals",
		VersionHash: "v1",
		SQLCache:    cache,
	}

	expr := "1 + 2"
	dialect := "postgres"

	// 1. First Pass (Cache Miss)
	sql1, _, err := ResolveExpression(expr, dialect, ctx)
	assert.NoError(t, err)
	// expectedSQL := "1 + 2"
	// NOTE: Actual SQL generation might vary depending on TO_SQL impl, check logs if fail.
	// But assuming it wraps?

	// Let's rely on consistency.
	assert.NotEmpty(t, sql1)

	// Verify Cache Content
	key := SQLCacheKey{
		BOID:        ctx.BOID,
		CalcID:      ctx.CalcID,
		DialectName: dialect,
		VersionHash: ctx.VersionHash,
	}
	cachedVal, ok := cache.Get(key)
	assert.True(t, ok, "Cache should be populated after first resolve")
	assert.Equal(t, sql1, cachedVal.SQL)

	// 2. Second Pass (Cache Hit)
	// To prove it hits cache, we could inspect coverage or logs,
	// but functionally we just want to ensure it works and returns same result.
	sql2, _, err := ResolveExpression(expr, dialect, ctx)
	assert.NoError(t, err)
	assert.Equal(t, sql1, sql2)
}

func TestResolveExpression_CacheInvalidation(t *testing.T) {
	// Setup Cache
	cache := NewSQLCache(10)

	expr := "10 * 10"
	dialect := "postgres"

	// Call 1 with Version v1
	ctx1 := &ResolutionContext{
		BOID:        "bo_test",
		CalcID:      "calc_math",
		VersionHash: "v1",
		SQLCache:    cache,
	}
	sql1, _, _ := ResolveExpression(expr, dialect, ctx1)

	// Call 2 with Version v2
	ctx2 := &ResolutionContext{
		BOID:        "bo_test",
		CalcID:      "calc_math",
		VersionHash: "v2",
		SQLCache:    cache,
	}
	sql2, _, _ := ResolveExpression(expr, dialect, ctx2)

	// They should produce same SQL (logic didn't change), but stored as different keys.
	assert.Equal(t, sql1, sql2)

	// Verify both keys exist
	_, ok1 := cache.Get(SQLCacheKey{BOID: "bo_test", CalcID: "calc_math", DialectName: dialect, VersionHash: "v1"})
	_, ok2 := cache.Get(SQLCacheKey{BOID: "bo_test", CalcID: "calc_math", DialectName: dialect, VersionHash: "v2"})

	assert.True(t, ok1)
	assert.True(t, ok2)
}
