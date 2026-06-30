package calcengine

import (
	"context"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupManager(t *testing.T, dialect QueryDialect) (*DataIntegrityManager, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	if dialect == nil {
		dialect = PostgresQueryDialect{}
	}
	manager := NewDataIntegrityManager(db, &IntegrityConfig{}, dialect)
	return manager, mock
}

func expectWatermark(mock sqlmock.Sqlmock, cutoff time.Time, state string) {
	rows := sqlmock.NewRows([]string{
		"table_name", "tenant_id", "datasource_id", "cutoff_date", "state",
		"migration_started", "migration_ended", "hot_row_count", "cold_row_count",
		"last_validated_at", "updated_at",
	}).AddRow("positions", "tenant-alpha", "ds-postgres", cutoff, state, nil, nil, 0, 0, time.Now(), time.Now())

	mock.ExpectQuery(regexp.QuoteMeta("SELECT table_name, tenant_id, datasource_id, cutoff_date, state,")).
		WithArgs("positions", "tenant-alpha", "ds-postgres").
		WillReturnRows(rows)
}

func TestBuildSafeQuery_UnionSafe_Postgres(t *testing.T) {
	cutoff := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	manager, mock := setupManager(t, PostgresQueryDialect{})
	expectWatermark(mock, cutoff, "STABLE")

	q := &TierQuery{
		TableName:     "positions",
		TenantID:      "tenant-alpha",
		DatasourceID:  "ds-postgres",
		DateColumn:    "as_of_date",
		Mode:          UnionSafe,
		SelectColumns: "account_id, balance",
	}

	sql, args, err := manager.BuildSafeQuery(context.Background(), q)
	require.NoError(t, err)

	// Hot branch is strictly greater than watermark; cold branch is less-or-equal.
	assert.Contains(t, sql, `"as_of_date" > $3`)
	assert.Contains(t, sql, `"as_of_date" <= $6`)

	// Tenant scoping on both halves.
	assert.Contains(t, sql, "tenant_id = $1")
	assert.Contains(t, sql, "tenant_id = $4")
	assert.Contains(t, sql, "datasource_id = $5")

	// Both halves emit identical projection columns and the tier discriminator.
	assert.Contains(t, sql, "SELECT account_id, balance, 'hot' AS _data_tier")
	assert.Contains(t, sql, "SELECT account_id, balance, 'cold' AS _data_tier")
	assert.Contains(t, sql, "UNION ALL")

	// Args align 1:1 with placeholders.
	require.Len(t, args, 6)
	assert.Equal(t, "tenant-alpha", args[0])
	assert.Equal(t, "ds-postgres", args[1])
	assert.Equal(t, "2024-01-15", args[2])
	assert.Equal(t, "tenant-alpha", args[3])
	assert.Equal(t, "ds-postgres", args[4])
	assert.Equal(t, "2024-01-15", args[5])
}

func TestBuildSafeQuery_HotOnly(t *testing.T) {
	cutoff := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	manager, mock := setupManager(t, PostgresQueryDialect{})
	expectWatermark(mock, cutoff, "STABLE")

	q := &TierQuery{
		TableName:     "positions",
		TenantID:      "tenant-alpha",
		DatasourceID:  "ds-postgres",
		DateColumn:    "as_of_date",
		Mode:          HotOnly,
		SelectColumns: "*",
	}

	sql, args, err := manager.BuildSafeQuery(context.Background(), q)
	require.NoError(t, err)

	assert.Contains(t, sql, `FROM semantic_hot."positions"`)
	assert.Contains(t, sql, "tenant_id = $1")
	assert.Contains(t, sql, "datasource_id = $2")
	assert.Len(t, args, 2)
	assert.NotContains(t, sql, "UNION ALL")
}

func TestBuildSafeQuery_ColdOnly(t *testing.T) {
	cutoff := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	manager, mock := setupManager(t, PostgresQueryDialect{})
	expectWatermark(mock, cutoff, "STABLE")

	q := &TierQuery{
		TableName:     "positions",
		TenantID:      "tenant-alpha",
		DatasourceID:  "ds-postgres",
		DateColumn:    "as_of_date",
		Mode:          ColdOnly,
		SelectColumns: "*",
	}

	sql, args, err := manager.BuildSafeQuery(context.Background(), q)
	require.NoError(t, err)

	assert.Contains(t, sql, `FROM semantic_cold."positions"`)
	assert.Contains(t, sql, `"as_of_date" <= $3`)
	assert.Len(t, args, 3)
}

func TestBuildSafeQuery_MigrationFallback(t *testing.T) {
	cutoff := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	manager, mock := setupManager(t, PostgresQueryDialect{})
	expectWatermark(mock, cutoff, "MIGRATING")

	q := &TierQuery{
		TableName:     "positions",
		TenantID:      "tenant-alpha",
		DatasourceID:  "ds-postgres",
		DateColumn:    "as_of_date",
		Mode:          UnionSafe, // Should be ignored during migration
		SelectColumns: "*",
	}

	sql, _, err := manager.BuildSafeQuery(context.Background(), q)
	require.NoError(t, err)

	assert.Contains(t, sql, `FROM semantic_hot."positions"`)
	assert.NotContains(t, sql, "UNION ALL")
}

func TestBuildSafeQuery_DateRange(t *testing.T) {
	cutoff := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	manager, mock := setupManager(t, PostgresQueryDialect{})
	expectWatermark(mock, cutoff, "STABLE")

	q := &TierQuery{
		TableName:     "positions",
		TenantID:      "tenant-alpha",
		DatasourceID:  "ds-postgres",
		DateColumn:    "as_of_date",
		Mode:          UnionSafe,
		SelectColumns: "*",
		StartDate:     &start,
		EndDate:       &end,
	}

	sql, args, err := manager.BuildSafeQuery(context.Background(), q)
	require.NoError(t, err)

	// Hot branch should clamp its start to the watermark.
	assert.Contains(t, sql, `"as_of_date" > $3`)
	assert.Contains(t, sql, `"as_of_date" >= $4`) // start clamped to cutoff
	assert.Contains(t, sql, `"as_of_date" <= $5`)

	// Cold branch should clamp its end to the watermark.
	assert.Contains(t, sql, `"as_of_date" <= $8`)
	assert.Contains(t, sql, `"as_of_date" >= $9`)
	assert.Contains(t, sql, `"as_of_date" <= $10`) // end clamped to cutoff

	require.Len(t, args, 10)
}

func TestBuildSafeQuery_UserWhereClause(t *testing.T) {
	cutoff := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	manager, mock := setupManager(t, PostgresQueryDialect{})
	expectWatermark(mock, cutoff, "STABLE")

	q := &TierQuery{
		TableName:     "positions",
		TenantID:      "tenant-alpha",
		DatasourceID:  "ds-postgres",
		DateColumn:    "as_of_date",
		Mode:          UnionSafe,
		SelectColumns: "*",
		WhereClause:   "status = $1",
		WhereArgs:     []interface{}{"active"},
	}

	sql, args, err := manager.BuildSafeQuery(context.Background(), q)
	require.NoError(t, err)

	// User clause is parenthesized on both branches and its placeholder is
	// offset to avoid colliding with the structural tenant/datasource/watermark
	// parameters.
	assert.Contains(t, sql, "AND (status = $4)")
	assert.Contains(t, sql, "AND (status = $8)")
	assert.Len(t, args, 8) // tenant/datasource/watermark x2 + user arg x2
}

func TestBuildSafeQuery_Dialects(t *testing.T) {
	cutoff := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		dialect  QueryDialect
		expected string
	}{
		{"postgres", PostgresQueryDialect{}, `"as_of_date" > $3`},
		{"trino", TrinoQueryDialect{}, `"as_of_date" > ?`},
		{"sqlserver", SQLServerQueryDialect{}, `[as_of_date] > @p3`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			manager, mock := setupManager(t, tc.dialect)
			expectWatermark(mock, cutoff, "STABLE")

			q := &TierQuery{
				TableName:     "positions",
				TenantID:      "tenant-alpha",
				DatasourceID:  "ds-postgres",
				DateColumn:    "as_of_date",
				Mode:          UnionSafe,
				SelectColumns: "*",
			}

			sql, _, err := manager.BuildSafeQuery(context.Background(), q)
			require.NoError(t, err)
			assert.Contains(t, sql, tc.expected)
		})
	}
}

func TestBuildSafeQuery_MissingTenant(t *testing.T) {
	manager, _ := setupManager(t, PostgresQueryDialect{})

	q := &TierQuery{
		TableName: "positions",
		Mode:      UnionSafe,
	}

	_, _, err := manager.BuildSafeQuery(context.Background(), q)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tenant_id is required")
}

func TestOffsetPlaceholders(t *testing.T) {
	t.Run("postgres", func(t *testing.T) {
		sql := "tenant_id = $1 AND as_of_date > $2"
		got := offsetPostgresPlaceholders(sql, 3)
		assert.Equal(t, "tenant_id = $4 AND as_of_date > $5", got)
	})

	t.Run("sqlserver", func(t *testing.T) {
		sql := "tenant_id = @p1 AND as_of_date > @p2"
		got := offsetSQLServerPlaceholders(sql, 3)
		assert.Equal(t, "tenant_id = @p4 AND as_of_date > @p5", got)
	})

	t.Run("trino positional unchanged", func(t *testing.T) {
		sql := "tenant_id = ? AND as_of_date > ?"
		got := offsetPlaceholders(sql, TrinoQueryDialect{}, 3)
		assert.Equal(t, sql, got)
	})
}

func TestBuildSafeQuery_NoOverlap(t *testing.T) {
	// Verify that the hot and cold boundaries are mutually exclusive and cover
	// the entire timeline: hot is '>' and cold is '<=' on the same column.
	cutoff := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	manager, mock := setupManager(t, PostgresQueryDialect{})
	expectWatermark(mock, cutoff, "STABLE")

	q := &TierQuery{
		TableName:     "positions",
		TenantID:      "tenant-alpha",
		DatasourceID:  "ds-postgres",
		DateColumn:    "as_of_date",
		Mode:          UnionSafe,
		SelectColumns: "*",
	}

	sql, _, err := manager.BuildSafeQuery(context.Background(), q)
	require.NoError(t, err)

	hotClause, coldClause := splitBranches(sql)

	assert.Contains(t, hotClause, `"as_of_date" > `)
	assert.NotContains(t, hotClause, `"as_of_date" <= `)
	assert.Contains(t, coldClause, `"as_of_date" <= `)
	assert.NotContains(t, coldClause, `"as_of_date" > `)
}

// splitBranches returns the hot and cold sub-queries from a UNION ALL SQL block.
func splitBranches(sql string) (string, string) {
	idx := strings.Index(sql, "UNION ALL")
	if idx == -1 {
		return sql, ""
	}
	return sql[:idx], sql[idx+len("UNION ALL"):]
}

func TestOrderFieldsDeterministically(t *testing.T) {
	fields := []string{"gamma", "alpha", "beta"}
	got := OrderFieldsDeterministically(fields)
	assert.Equal(t, "alpha, beta, gamma", got)

	// Original slice must not be mutated.
	assert.Equal(t, "gamma", fields[0])
}

func TestBuildSafeQuery_DeterministicProjectionOrder(t *testing.T) {
	cutoff := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	manager, mock := setupManager(t, PostgresQueryDialect{})
	expectWatermark(mock, cutoff, "STABLE")

	q := &TierQuery{
		TableName:     "positions",
		TenantID:      "tenant-alpha",
		DatasourceID:  "ds-postgres",
		DateColumn:    "as_of_date",
		Mode:          UnionSafe,
		SelectColumns: "z_col, a_col, m_col",
	}

	sql, _, err := manager.BuildSafeQuery(context.Background(), q)
	require.NoError(t, err)

	hotClause, coldClause := splitBranches(sql)
	assert.Contains(t, hotClause, "SELECT a_col, m_col, z_col, 'hot' AS _data_tier")
	assert.Contains(t, coldClause, "SELECT a_col, m_col, z_col, 'cold' AS _data_tier")
}

func TestBuildSafeQuery_LimitOffsetPushdown(t *testing.T) {
	cutoff := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name             string
		dialect          QueryDialect
		expectOrderBy    bool
		expectOrderByCol string
	}{
		{"postgres", PostgresQueryDialect{}, false, ""},
		{"trino", TrinoQueryDialect{}, true, `"as_of_date"`},
		{"sqlserver", SQLServerQueryDialect{}, false, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			manager, mock := setupManager(t, tc.dialect)
			expectWatermark(mock, cutoff, "STABLE")

			q := &TierQuery{
				TableName:     "positions",
				TenantID:      "tenant-alpha",
				DatasourceID:  "ds-postgres",
				DateColumn:    "as_of_date",
				Mode:          UnionSafe,
				SelectColumns: "*",
				Limit:         100,
				Offset:        50,
			}

			sql, _, err := manager.BuildSafeQuery(context.Background(), q)
			require.NoError(t, err)

			hotClause, coldClause := splitBranches(sql)

			// Both branches receive LIMIT/OFFSET pushdown.
			assert.Contains(t, hotClause, "LIMIT 100")
			assert.Contains(t, hotClause, "OFFSET 50")
			assert.Contains(t, coldClause, "LIMIT 100")
			assert.Contains(t, coldClause, "OFFSET 50")

			if tc.expectOrderBy {
				assert.Contains(t, hotClause, "ORDER BY "+tc.expectOrderByCol)
				assert.Contains(t, coldClause, "ORDER BY "+tc.expectOrderByCol)
			} else {
				assert.NotContains(t, hotClause, "ORDER BY")
				assert.NotContains(t, coldClause, "ORDER BY")
			}

			// Outer wrapper also enforces final pagination.
			assert.Contains(t, sql, "\nLIMIT 100")
			assert.Contains(t, sql, "OFFSET 50")
		})
	}
}

func TestBuildSafeQuery_ParameterPaddingAlignment(t *testing.T) {
	cutoff := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	manager, mock := setupManager(t, PostgresQueryDialect{})
	expectWatermark(mock, cutoff, "STABLE")

	// Pre-parameterized user filter with two args. This is the most dangerous
	// case for argument misalignment: the router must keep user args in their
	// original slots and append structural args (tenant, datasource, watermark)
	// only after them.
	q := &TierQuery{
		TableName:     "positions",
		TenantID:      "tenant-alpha",
		DatasourceID:  "ds-postgres",
		DateColumn:    "as_of_date",
		Mode:          UnionSafe,
		SelectColumns: "*",
		WhereClause:   "status = $1 AND region = $2",
		WhereArgs:     []interface{}{"active", "EMEA"},
	}

	sql, args, err := manager.BuildSafeQuery(context.Background(), q)
	require.NoError(t, err)

	// Hot branch layout:
	// $1 tenant, $2 datasource, $3 watermark, $4 status, $5 region
	// Cold branch layout (renumbered):
	// $6 tenant, $7 datasource, $8 watermark, $9 status, $10 region
	require.Len(t, args, 10)
	assert.Equal(t, "tenant-alpha", args[0])
	assert.Equal(t, "ds-postgres", args[1])
	assert.Equal(t, "2024-01-15", args[2])
	assert.Equal(t, "active", args[3])
	assert.Equal(t, "EMEA", args[4])
	assert.Equal(t, "tenant-alpha", args[5])
	assert.Equal(t, "ds-postgres", args[6])
	assert.Equal(t, "2024-01-15", args[7])
	assert.Equal(t, "active", args[8])
	assert.Equal(t, "EMEA", args[9])

	// Verify the watermark placeholder token matches the actual argument slot.
	assert.Contains(t, sql, `"as_of_date" > $3`)
	assert.Contains(t, sql, `"as_of_date" <= $8`)
}
