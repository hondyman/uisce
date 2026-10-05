package querybuilder

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/analytics"
	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGrainHash_StableAcrossOrder(t *testing.T) {
	a := GrainHash([]string{"status", "account_id"})
	b := GrainHash([]string{"account_id", "status"})
	assert.Equal(t, a, b)
	assert.Len(t, a, 16)
}

func TestEnsureGrainNodes_InsertsIdleNode(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	sqlxDB := sqlx.NewDb(db, "postgres")
	reg := NewCubeMaterializationRegistry(sqlxDB, analytics.NewPreAggLifecycleService(sqlxDB))

	cube := CubeDefinition{
		ID:              uuid.New().String(),
		TenantID:        "99e99e99-99e9-49e9-89e9-99e99e99e999",
		Name:            "Account Smoke",
		BOID:            "account",
		MetricIDs:       []string{"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"},
		Grains:          [][]string{{"account_id", "status"}},
		ContractVersion: 1,
		ContentHash:     "hash1",
		Materialization: CubeMaterializationConfig{Strategy: "starrocks_mv", RefreshStrategy: "manual"},
	}
	nodeType := uuid.New()
	mock.ExpectQuery(`SELECT id::text FROM catalog_node_type WHERE catalog_type_name = 'pre_aggregation'`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(nodeType.String()))

	mock.ExpectQuery(`SELECT id, properties FROM catalog_node`).
		WithArgs(cube.TenantID, sqlmock.AnyArg()).
		WillReturnError(sql.ErrNoRows)

	mock.ExpectExec(`INSERT INTO catalog_node`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), nodeType.String(), cube.TenantID, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	nodes, err := reg.EnsureGrainNodes(context.Background(), cube)
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	assert.Equal(t, models.LifecycleIdle, nodes[0].Properties.LifecycleStatus)
	assert.Equal(t, cube.ID, nodes[0].Properties.CubeID)
	assert.Equal(t, 1, nodes[0].Properties.ContractVersion)
	assert.Equal(t, []string{"account_id", "status"}, nodes[0].Grain)
	assert.Equal(t, GrainHash(nodes[0].Grain), nodes[0].GrainHash)
	assert.Contains(t, nodes[0].NodeName, "cube_")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestEnsureGrainNodes_PreservesLifecycleOnUpdate(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	sqlxDB := sqlx.NewDb(db, "postgres")
	reg := NewCubeMaterializationRegistry(sqlxDB, nil)

	cube := CubeDefinition{
		ID:              uuid.New().String(),
		TenantID:        "99e99e99-99e9-49e9-89e9-99e99e99e999",
		Name:            "Account Smoke",
		BOID:            "account",
		MetricIDs:       []string{"m1"},
		Grains:          [][]string{{"account_id"}},
		ContractVersion: 2,
		ContentHash:     "hash2",
	}
	existingID := uuid.New()
	existingProps := models.PreAggProperties{
		LifecycleStatus: models.LifecycleActive,
		AttemptID:       "keep-me",
		CubeID:          cube.ID,
		Grain:           []string{"account_id"},
	}
	existingJSON, err := json.Marshal(existingProps)
	require.NoError(t, err)

	nodeType := uuid.New()
	mock.ExpectQuery(`SELECT id::text FROM catalog_node_type`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(nodeType.String()))
	mock.ExpectQuery(`SELECT id, properties FROM catalog_node`).
		WithArgs(cube.TenantID, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "properties"}).AddRow(existingID, existingJSON))
	mock.ExpectExec(`UPDATE catalog_node`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), existingID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	nodes, err := reg.EnsureGrainNodes(context.Background(), cube)
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	assert.Equal(t, existingID, nodes[0].ID)
	assert.Equal(t, models.LifecycleActive, nodes[0].Properties.LifecycleStatus)
	assert.Equal(t, "keep-me", nodes[0].Properties.AttemptID)
	assert.Equal(t, 2, nodes[0].Properties.ContractVersion)
	assert.Equal(t, "hash2", nodes[0].Properties.CubeContentHash)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCubeMaterializationRegistry_AttemptTransitions(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	sqlxDB := sqlx.NewDb(db, "postgres")
	reg := NewCubeMaterializationRegistry(sqlxDB, analytics.NewPreAggLifecycleService(sqlxDB))
	id := uuid.New()
	attempt := "wf-attempt-9"

	seed, err := json.Marshal(models.PreAggProperties{LifecycleStatus: models.LifecycleIdle})
	require.NoError(t, err)

	mock.ExpectQuery(`SELECT properties FROM catalog_node WHERE id = \$1`).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"properties"}).AddRow(seed))
	mock.ExpectExec(`UPDATE catalog_node SET properties`).
		WithArgs(sqlmock.AnyArg(), id).
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, reg.BeginAttempt(context.Background(), id, attempt))

	mat, err := json.Marshal(models.PreAggProperties{LifecycleStatus: models.LifecycleMaterializing, AttemptID: attempt})
	require.NoError(t, err)
	mock.ExpectQuery(`SELECT properties FROM catalog_node WHERE id = \$1`).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"properties"}).AddRow(mat))
	mock.ExpectExec(`UPDATE catalog_node SET properties`).
		WithArgs(sqlmock.AnyArg(), id).
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, reg.CompleteAttempt(context.Background(), id, attempt, nil))

	mock.ExpectQuery(`SELECT properties FROM catalog_node WHERE id = \$1`).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"properties"}).AddRow(mat))
	mock.ExpectExec(`UPDATE catalog_node SET properties`).
		WithArgs(sqlmock.AnyArg(), id).
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, reg.FailAttempt(context.Background(), id, attempt, errors.New("boom")))

	require.NoError(t, mock.ExpectationsWereMet())
}
