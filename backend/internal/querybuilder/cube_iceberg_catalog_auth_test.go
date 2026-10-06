package querybuilder

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/hondyman/uisce/backend/internal/iceberg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIcebergTokenRefreshEnabled(t *testing.T) {
	t.Setenv("CUBE_ICEBERG_TOKEN_REFRESH", "")
	assert.False(t, icebergTokenRefreshEnabled())
	t.Setenv("CUBE_ICEBERG_TOKEN_REFRESH", "1")
	assert.True(t, icebergTokenRefreshEnabled())
	t.Setenv("CUBE_ICEBERG_TOKEN_REFRESH", "true")
	assert.True(t, icebergTokenRefreshEnabled())
	t.Setenv("CUBE_ICEBERG_TOKEN_REFRESH", "no")
	assert.False(t, icebergTokenRefreshEnabled())
}

func TestNewDefaultIcebergCatalogTokenRefresher_GateOff(t *testing.T) {
	t.Setenv("CUBE_ICEBERG_TOKEN_REFRESH", "")
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	assert.Nil(t, newDefaultIcebergCatalogTokenRefresher(db))
}

func TestStarRocksCatalogTokenRefresher_Ensure_AlterSQL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		_ = r.ParseForm()
		assert.Equal(t, "client_credentials", r.Form.Get("grant_type"))
		assert.Equal(t, "uisce-provisioner", r.Form.Get("client_id"))
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "tok-live-secret-value",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer srv.Close()

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectExec(`(?s)ALTER CATALOG .+lakekeeper_iceberg.+ SET \("iceberg\.catalog\.token" = 'tok-live-secret-value'\)`).
		WillReturnResult(sqlmock.NewResult(0, 0))

	r := &starRocksCatalogTokenRefresher{
		db: db,
		tm: iceberg.NewTokenManager(srv.URL, "uisce-provisioner", "client-secret"),
	}
	require.NoError(t, r.Ensure(context.Background(), "lakekeeper_iceberg"))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestStarRocksCatalogTokenRefresher_Ensure_RedactsTokenOnExecError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "super-secret-token-xyz",
			"expires_in":   60,
		})
	}))
	defer srv.Close()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectExec("(?s)ALTER CATALOG.*").
		WillReturnError(fmt.Errorf("StarRocks: bad token super-secret-token-xyz in catalog"))

	r := &starRocksCatalogTokenRefresher{
		db: db,
		tm: iceberg.NewTokenManager(srv.URL, "uisce-provisioner", "client-secret"),
	}
	err = r.Ensure(context.Background(), "lakekeeper_iceberg")
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "super-secret-token-xyz")
	assert.Contains(t, err.Error(), "***")
	assert.Contains(t, err.Error(), "ALTER CATALOG lakekeeper_iceberg")
}

func TestStarRocksCatalogTokenRefresher_Ensure_MissingSecret(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	r := &starRocksCatalogTokenRefresher{db: db, tm: nil}
	err = r.Ensure(context.Background(), "lakekeeper_iceberg")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "LAKEKEEPER_CLIENT_SECRET")
}

func TestRedactCatalogAuthSecrets(t *testing.T) {
	err := redactCatalogAuthSecrets(errors.New("fail tok-abc end"), "tok-abc")
	require.Error(t, err)
	assert.Equal(t, "fail *** end", err.Error())
	assert.Nil(t, redactCatalogAuthSecrets(nil, "x"))
}

func TestApplyCold_RunsCatalogTokenRefreshFirst(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer db.Close()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "fresh-tok", "expires_in": 3600})
	}))
	defer srv.Close()

	mock.ExpectExec(`ALTER CATALOG .+ SET \("iceberg\.catalog\.token"`).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`CREATE DATABASE IF NOT EXISTS .+`).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`DROP TABLE IF EXISTS .+`).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`CREATE TABLE .+ AS SELECT \* FROM .+`).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM .+`).
		WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(3))

	m := NewCubeMaterializer(nil, db)
	m.SetCatalogTokenRefresher(&starRocksCatalogTokenRefresher{
		db: db,
		tm: iceberg.NewTokenManager(srv.URL, "uisce-provisioner", "secret"),
	})

	plan := &CubeMaterializePlan{
		IcebergCatalog:       "lakekeeper_iceberg",
		IcebergDatabase:      "cubes",
		MaterializationName:  "cube_grain_x",
		TargetDatabase:       "oms",
	}
	hot := &CubeMaterializeHotResult{AppliedDDL: true, RowCount: 3}
	cold, err := m.ApplyCold(context.Background(), plan, hot)
	require.NoError(t, err)
	require.NotNil(t, cold)
	assert.True(t, cold.Applied)
	assert.True(t, strings.HasPrefix(cold.IcebergTable, "lakekeeper_iceberg.cubes."))
	require.NoError(t, mock.ExpectationsWereMet())
}
