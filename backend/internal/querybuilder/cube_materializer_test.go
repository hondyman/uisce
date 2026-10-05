package querybuilder

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCubeMaterializer_RejectsFederation(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	sqlxDB := sqlx.NewDb(db, "postgres")
	m := NewCubeMaterializer(sqlxDB, nil)

	cubeID := uuid.New().String()
	tenantID := "99e99e99-99e9-49e9-89e9-99e99e99e999"
	dims, _ := json.Marshal([]CubeDimension{{TermNodeID: "account_id"}})
	metrics, _ := json.Marshal([]string{"m1"})
	grains, _ := json.Marshal([][]string{{"account_id"}})
	mat, _ := json.Marshal(CubeMaterializationConfig{Strategy: "starrocks_mv"})
	fed, _ := json.Marshal(CubeFederation{
		Sources: []CubeFederationSource{{BOID: "account", Alias: "a"}, {BOID: "position", Alias: "p"}},
		Joins:   []CubeFederationJoin{{LeftAlias: "a", RightAlias: "p", KeyKind: "common", LeftTermIDs: []string{"account_id"}, RightTermIDs: []string{"account_id"}}},
	})

	expectFedCube := func() {
		rows := sqlmock.NewRows([]string{
			"id", "tenant_id", "name", "description", "bo_id",
			"dimensions", "time_dimension", "metric_ids", "grains", "materialization",
			"federation", "contract_version", "content_hash", "is_core", "status",
			"archived_at", "created_by", "created_at", "updated_at",
		}).AddRow(
			cubeID, tenantID, "Fed Cube", "", "account",
			dims, nil, metrics, grains, mat,
			fed, 1, "hash", false, "active",
			nil, nil, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		)
		mock.ExpectQuery(`SELECT .+ FROM data_explorer.cube_definition`).
			WithArgs(cubeID, tenantID).
			WillReturnRows(rows)
	}

	t.Run("no samples fail-closed orphan gate", func(t *testing.T) {
		expectFedCube()
		_, err = m.ValidateAndPlan(context.Background(), CubeMaterializeRequest{
			TenantID: tenantID,
			CubeID:   cubeID,
			Grain:    []string{"account_id"},
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrCubeFederationOrphanGate)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("passing samples still require CUBE-2.2 materialize", func(t *testing.T) {
		expectFedCube()
		_, err = m.ValidateAndPlan(context.Background(), CubeMaterializeRequest{
			TenantID: tenantID,
			CubeID:   cubeID,
			Grain:    []string{"account_id"},
			FederationKeySamples: []FederationKeySample{{
				LeftAlias: "a", RightAlias: "p",
				LeftKeys: 1000, RightKeys: 1000, Matched: 995,
			}},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "CUBE-2.2")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestCubeMaterializer_ApplyHotRequiresStarRocks(t *testing.T) {
	m := NewCubeMaterializer(nil, nil)
	_, err := m.ApplyHot(context.Background(), &CubeMaterializePlan{
		TargetDatabase:      "tenant_t",
		MaterializationName: "cube_x",
		DDL:                 "CREATE MATERIALIZED VIEW cube_x AS SELECT 1;",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "starrocks connection is not available")
}

func TestGrainCovered(t *testing.T) {
	grains := [][]string{{"b", "a"}, {"day"}}
	assert.True(t, grainCovered(grains, []string{"a", "b"}))
	assert.False(t, grainCovered(grains, []string{"missing"}))
}
