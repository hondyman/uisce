package domain

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/hondyman/semlayer/backend/internal/calcengine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGLBalanceResolver_ResolveBalanceQuery(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	cutoff := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	rows := sqlmock.NewRows([]string{
		"table_name", "tenant_id", "datasource_id", "cutoff_date", "state",
		"migration_started", "migration_ended", "hot_row_count", "cold_row_count",
		"last_validated_at", "updated_at",
	}).AddRow("ibor_positions", "tenant-alpha", "ds-postgres", cutoff, "STABLE", nil, nil, 0, 0, time.Now(), time.Now())

	mock.ExpectQuery(regexp.QuoteMeta("SELECT table_name, tenant_id, datasource_id, cutoff_date, state,")).
		WithArgs("ibor_positions", "tenant-alpha", "ds-postgres").
		WillReturnRows(rows)

	dim := calcengine.NewDataIntegrityManager(db, &calcengine.IntegrityConfig{}, calcengine.PostgresQueryDialect{})
	resolver := NewGLBalanceResolver(dim)

	sql, args, err := resolver.ResolveBalanceQuery(context.Background(), GLBalanceRequest{
		TenantID:      "tenant-alpha",
		DatasourceID:  "ds-postgres",
		TableName:     "ibor_positions",
		AsOfDate:      time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
		SelectColumns: "account_id, balance",
	})
	require.NoError(t, err)

	assert.Contains(t, sql, "UNION ALL")
	assert.Contains(t, sql, `"as_of_date" > $3`)
	assert.Contains(t, sql, `"as_of_date" <= $4`)
	assert.Contains(t, sql, `"as_of_date" <= $7`)
	assert.Contains(t, sql, "account_id, balance")
	require.Len(t, args, 8)
}

func TestGLBalanceResolver_RequiresTenant(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	dim := calcengine.NewDataIntegrityManager(db, &calcengine.IntegrityConfig{}, calcengine.PostgresQueryDialect{})
	resolver := NewGLBalanceResolver(dim)

	_, _, err = resolver.ResolveBalanceQuery(context.Background(), GLBalanceRequest{
		TableName: "ibor_positions",
		AsOfDate:  time.Now(),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tenant_id is required")
}
