package boresolver

import (
	"fmt"
)

// ResolutionContext holds the necessary metadata for resolving a BO expression
type ResolutionContext struct {
	BOID         string
	DrivingTable string
	TermMappings map[string]PhysicalMapping
	JoinPaths    map[string][]JoinStep
	// Caching Context
	CalcID      string
	VersionHash string
	SQLCache    *SQLCache
}

// ResolveExpression resolves a semantic expression string into dialect-specific SQL
func ResolveExpression(expression string, dialectName string, ctx *ResolutionContext) (string, []JoinStep, error) {
	// 0. Check Cache
	if ctx.SQLCache != nil {
		key := SQLCacheKey{
			BOID:        ctx.BOID,
			CalcID:      ctx.CalcID, // If empty, caller might be doing ad-hoc resolution without caching intent, or we should hash logic.
			DialectName: dialectName,
			VersionHash: ctx.VersionHash,
		}

		// Only cache if we have a stable ID (CalcID or TermID) to key off of
		if key.CalcID != "" {
			if val, ok := ctx.SQLCache.Get(key); ok {
				return val.SQL, val.Joins, nil
			}
		}
	}

	// 1. Select Dialect
	var dialect Dialect
	switch dialectName {
	case "postgres":
		dialect = PostgresDialect{}
	case "snowflake":
		dialect = SnowflakeDialect{}
	case "sqlserver":
		dialect = SQLServerDialect{}
	default:
		return "", nil, fmt.Errorf("unknown dialect: %s", dialectName)
	}

	// 2. Parse Expression
	expr, err := ParseExpression(expression)
	if err != nil {
		return "", nil, err
	}

	// 3. Initialize Resolver
	resolver := &Resolver{
		BOID:         ctx.BOID,
		DrivingTable: ctx.DrivingTable,
		TermMappings: ctx.TermMappings,
		JoinPaths:    ctx.JoinPaths,
		Dialect:      dialect,
	}

	// 4. Generate SQL and Joins
	sql, joins, err := resolver.ToSQL(expr)
	if err != nil {
		return "", nil, err
	}

	// 5. Deduplicate Joins
	uniqueJoins := deduplicateJoins(joins)

	// 6. Update Cache
	if ctx.SQLCache != nil && ctx.CalcID != "" {
		key := SQLCacheKey{
			BOID:        ctx.BOID,
			CalcID:      ctx.CalcID,
			DialectName: dialectName,
			VersionHash: ctx.VersionHash,
		}
		ctx.SQLCache.Set(key, SQLCacheValue{
			SQL:   sql,
			Joins: uniqueJoins,
		})
	}

	return sql, uniqueJoins, nil
}

// deduplicateJoins removes duplicate join steps and preserves order
func deduplicateJoins(steps []JoinStep) []JoinStep {
	seen := make(map[string]bool)
	unique := []JoinStep{}

	for _, step := range steps {
		// Create a unique key for the join
		key := fmt.Sprintf("%s->%s:%s", step.FromTable, step.ToTable, step.Condition)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, step)
		}
	}
	return unique
}
