package analytics

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func newLifecycleFixture(t *testing.T) (*PreAggLifecycleService, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return NewPreAggLifecycleService(sqlx.NewDb(db, "postgres")), mock
}

type lifecyclePropsMatcher struct {
	wantStatus  string
	wantAttempt string
	wantFresh   bool
	forbidFresh bool
}

func (m lifecyclePropsMatcher) Match(v driver.Value) bool {
	var b []byte
	switch x := v.(type) {
	case []byte:
		b = x
	case string:
		b = []byte(x)
	default:
		return false
	}
	var p models.PreAggProperties
	if err := json.Unmarshal(b, &p); err != nil {
		return false
	}
	if p.LifecycleStatus != m.wantStatus {
		return false
	}
	if p.AttemptID != m.wantAttempt {
		return false
	}
	if m.wantFresh && (p.LastRefreshedAt == nil || p.LastRefreshedAt.IsZero()) {
		return false
	}
	if m.forbidFresh && p.LastRefreshedAt != nil {
		return false
	}
	return true
}

func TestLifecycle_MaterializingToActive_SetsFreshnessClock(t *testing.T) {
	svc, mock := newLifecycleFixture(t)
	id := uuid.New()
	attempt := "att-active-1"

	seed := models.PreAggProperties{
		BOName:          "account",
		TenantID:        "t1",
		LifecycleStatus: models.LifecycleIdle,
		CubeID:          "cube-1",
	}
	seedJSON, err := json.Marshal(seed)
	require.NoError(t, err)

	mock.ExpectQuery(`SELECT properties FROM catalog_node WHERE id = \$1`).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"properties"}).AddRow(seedJSON))
	mock.ExpectExec(`UPDATE catalog_node SET properties = \$1, updated_at = NOW\(\) WHERE id = \$2`).
		WithArgs(lifecyclePropsMatcher{
			wantStatus:  models.LifecycleMaterializing,
			wantAttempt: attempt,
			forbidFresh: true,
		}, id).
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, svc.MarkMaterializingAttempt(context.Background(), id, attempt))

	mat := seed
	mat.LifecycleStatus = models.LifecycleMaterializing
	mat.AttemptID = attempt
	matJSON, err := json.Marshal(mat)
	require.NoError(t, err)

	mock.ExpectQuery(`SELECT properties FROM catalog_node WHERE id = \$1`).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"properties"}).AddRow(matJSON))
	mock.ExpectExec(`UPDATE catalog_node SET properties = \$1, updated_at = NOW\(\) WHERE id = \$2`).
		WithArgs(lifecyclePropsMatcher{
			wantStatus:  models.LifecycleActive,
			wantAttempt: attempt,
			wantFresh:   true,
		}, id).
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, svc.MarkActiveAttempt(context.Background(), id, attempt, &models.PreAggStats{RowCount: 10}))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLifecycle_MaterializingToFailed_DoesNotAdvanceFreshness(t *testing.T) {
	svc, mock := newLifecycleFixture(t)
	id := uuid.New()
	attempt := "att-fail-1"

	mat := models.PreAggProperties{
		BOName:          "account",
		TenantID:        "t1",
		LifecycleStatus: models.LifecycleMaterializing,
		AttemptID:       attempt,
		CubeID:          "cube-1",
	}
	matJSON, err := json.Marshal(mat)
	require.NoError(t, err)

	mock.ExpectQuery(`SELECT properties FROM catalog_node WHERE id = \$1`).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"properties"}).AddRow(matJSON))
	mock.ExpectExec(`UPDATE catalog_node SET properties = \$1, updated_at = NOW\(\) WHERE id = \$2`).
		WithArgs(lifecyclePropsMatcher{
			wantStatus:  models.LifecycleFailed,
			wantAttempt: attempt,
			forbidFresh: true,
		}, id).
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, svc.MarkFailedAttempt(context.Background(), id, attempt, errors.New("cold sink failed")))
	require.NoError(t, mock.ExpectationsWereMet())
}
