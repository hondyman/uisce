package finops

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
)

func TestForecastHandler_PrewarmTrigger_WritesPendingAndReturns202(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	handler := NewForecastHandler(sqlxDB)
	tenantID := uuid.New()
	userID := uuid.New()

	// Expect the synchronous PENDING write
	mock.ExpectExec(`INSERT INTO finops\.prewarm_execution_ledger`).
		WithArgs(tenantID, sqlmock.AnyArg(), userID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	req := httptest.NewRequest(http.MethodPost, "/api/finops/prewarm/trigger", nil)
	claims := &jwtmiddleware.JWTClaims{
		TenantID:    tenantID.String(),
		UserID:      userID.String(),
		Roles:       []string{"admin"},
		IsCoreAdmin: false,
	}
	ctx := context.WithValue(req.Context(), jwtmiddleware.ClaimsContextKey, claims)
	ctx = context.WithValue(ctx, jwtmiddleware.TenantIDContextKey, tenantID.String())
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.handlePrewarmTrigger(rec, req)

	assert.Equal(t, http.StatusAccepted, rec.Code)
	var resp map[string]any
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "PENDING", resp["status"])
	assert.NotEmpty(t, resp["jobId"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestForecastHandler_PrewarmTrigger_PendingInsertFails_ReleasesLockAnd500s(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	handler := NewForecastHandler(sqlxDB)
	tenantID := uuid.New()
	userID := uuid.New()

	// First trigger: simulate DB failure on PENDING insert (e.g. missing job_id column)
	mock.ExpectExec(`INSERT INTO finops\.prewarm_execution_ledger`).
		WithArgs(tenantID, sqlmock.AnyArg(), userID).
		WillReturnError(fmt.Errorf("column job_id does not exist"))

	req1 := httptest.NewRequest(http.MethodPost, "/api/finops/prewarm/trigger", nil)
	claims := &jwtmiddleware.JWTClaims{
		TenantID: tenantID.String(),
		UserID:   userID.String(),
		Roles:    []string{"finops_manager"},
	}
	ctx1 := context.WithValue(req1.Context(), jwtmiddleware.ClaimsContextKey, claims)
	ctx1 = context.WithValue(ctx1, jwtmiddleware.TenantIDContextKey, tenantID.String())
	req1 = req1.WithContext(ctx1)
	rec1 := httptest.NewRecorder()

	handler.handlePrewarmTrigger(rec1, req1)
	assert.Equal(t, http.StatusInternalServerError, rec1.Code)

	// Second trigger: MUST NOT return 409 Conflict because lock was released on failure!
	mock.ExpectExec(`INSERT INTO finops\.prewarm_execution_ledger`).
		WithArgs(tenantID, sqlmock.AnyArg(), userID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	req2 := httptest.NewRequest(http.MethodPost, "/api/finops/prewarm/trigger", nil)
	ctx2 := context.WithValue(req2.Context(), jwtmiddleware.ClaimsContextKey, claims)
	ctx2 = context.WithValue(ctx2, jwtmiddleware.TenantIDContextKey, tenantID.String())
	req2 = req2.WithContext(ctx2)
	rec2 := httptest.NewRecorder()

	handler.handlePrewarmTrigger(rec2, req2)
	assert.Equal(t, http.StatusAccepted, rec2.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestForecastHandler_GetPrewarmStatus_FilterByTenantAndJob(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	handler := NewForecastHandler(sqlxDB)
	tenantID := uuid.New()
	otherTenantID := uuid.New()
	jobID := uuid.New()

	// 1. Query by specific jobId -> filters by tenant_id AND job_id (Rule 7 tenant isolation)
	rows := sqlmock.NewRows([]string{
		"tenant_id", "job_id", "entities_prewarmed_count", "compute_cost_incurred_usd",
		"estimated_peak_savings_usd", "peak_probability_at_trigger", "status",
	}).AddRow(tenantID, jobID, 5, 0.0, 0.0, 0.85, "SIMULATED")

	mock.ExpectQuery("SELECT (.+) FROM finops.prewarm_execution_ledger WHERE tenant_id = \\$1 AND job_id = \\$2").
		WithArgs(tenantID, jobID).
		WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/api/finops/prewarm/status?jobId="+jobID.String(), nil)
	claims := &jwtmiddleware.JWTClaims{
		TenantID: tenantID.String(),
		Roles:    []string{"user"},
	}
	ctx := context.WithValue(req.Context(), jwtmiddleware.ClaimsContextKey, claims)
	ctx = context.WithValue(ctx, jwtmiddleware.TenantIDContextKey, tenantID.String())
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.handleGetPrewarmStatus(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	var res PrewarmResult
	err = json.Unmarshal(rec.Body.Bytes(), &res)
	require.NoError(t, err)
	assert.Equal(t, tenantID, res.TenantID)
	assert.Equal(t, "SIMULATED", res.Status)

	// 2. Query without jobId -> queries latest filtered strictly by tenant_id
	rowsLatest := sqlmock.NewRows([]string{
		"tenant_id", "job_id", "entities_prewarmed_count", "compute_cost_incurred_usd",
		"estimated_peak_savings_usd", "peak_probability_at_trigger", "status",
	}).AddRow(otherTenantID, jobID, 0, 0.0, 0.0, 0.50, "SKIPPED_BELOW_THRESHOLD")

	mock.ExpectQuery("SELECT (.+) FROM finops.prewarm_execution_ledger WHERE tenant_id = \\$1 ORDER BY executed_at DESC").
		WithArgs(otherTenantID).
		WillReturnRows(rowsLatest)

	reqLatest := httptest.NewRequest(http.MethodGet, "/api/finops/prewarm/status", nil)
	claimsLatest := &jwtmiddleware.JWTClaims{
		TenantID: otherTenantID.String(),
		Roles:    []string{"user"},
	}
	ctxLatest := context.WithValue(reqLatest.Context(), jwtmiddleware.ClaimsContextKey, claimsLatest)
	ctxLatest = context.WithValue(ctxLatest, jwtmiddleware.TenantIDContextKey, otherTenantID.String())
	reqLatest = reqLatest.WithContext(ctxLatest)
	recLatest := httptest.NewRecorder()

	handler.handleGetPrewarmStatus(recLatest, reqLatest)
	assert.Equal(t, http.StatusOK, recLatest.Code)

	var resLatest PrewarmResult
	err = json.Unmarshal(recLatest.Body.Bytes(), &resLatest)
	require.NoError(t, err)
	assert.Equal(t, otherTenantID, resLatest.TenantID)
	assert.Equal(t, "SKIPPED_BELOW_THRESHOLD", resLatest.Status)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPrewarmCoordinator_RecoverStalePendingExecutions(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	mock.ExpectExec("UPDATE finops.prewarm_execution_ledger SET status = 'FAILED'").
		WillReturnResult(sqlmock.NewResult(1, 2)) // 2 abandoned executions swept

	coord := NewPrewarmCoordinator(sqlxDB)
	err = coord.RecoverStalePendingExecutions(context.Background())
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestPrewarmCoordinator_TerminalInsert_TargetMetricRunLevelAll verifies that when
// updateOrPersistLedgerEntry falls through to the INSERT path with no targets
// (run-level outcome), the INSERT is called with bo_id=nil AND target_metric='ALL'.
// Both columns describe the same row state, derived from the same targets-presence
// branch — explicit literals, no reliance on schema defaults.
func TestPrewarmCoordinator_TerminalInsert_TargetMetricRunLevelAll(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	tenantID := uuid.New()
	jobID := uuid.New()

	// The UPDATE-by-job_id path runs first (jobID is non-nil). It will affect 0 rows
	// (no prior PENDING row exists), so the code falls through to the INSERT path.
	mock.ExpectExec(`UPDATE finops\.prewarm_execution_ledger`).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// INSERT must be called with bo_id=nil AND target_metric='ALL' for run-level rows.
	mock.ExpectExec(`INSERT INTO finops\.prewarm_execution_ledger`).
		WithArgs(
			tenantID,        // $1 tenant_id
			&jobID,          // $2 job_id
			nil,             // $3 bo_id (run-level)
			"ALL",           // $4 target_metric (run-level, explicit)
			sqlmock.AnyArg(), // $5 entities_prewarmed_count
			sqlmock.AnyArg(), // $6 compute_cost
			sqlmock.AnyArg(), // $7 estimated_savings
			sqlmock.AnyArg(), // $8 peak_probability
			sqlmock.AnyArg(), // $9 policy_id
			sqlmock.AnyArg(), // $10 status
			sqlmock.AnyArg(), // $11 error_detail
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	coord := NewPrewarmCoordinator(sqlxDB)
	result := &PrewarmResult{
		TenantID:      tenantID,
		Status:        "SKIPPED_NO_TARGETS",
		PeakProbability: 0.5,
	}
	err = coord.updateOrPersistLedgerEntry(context.Background(), tenantID, &jobID, nil, result, nil, "")
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestPrewarmCoordinator_TerminalInsert_TargetMetricFromFirstTarget verifies that when
// updateOrPersistLedgerEntry falls through to the INSERT path with a primary BO
// (per-target outcome), target_metric is the first target's FormulaType.
func TestPrewarmCoordinator_TerminalInsert_TargetMetricFromFirstTarget(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	tenantID := uuid.New()
	jobID := uuid.New()
	primaryBOID := uuid.New()

	// UPDATE affects 0 rows so the code falls through to INSERT.
	mock.ExpectExec(`UPDATE finops\.prewarm_execution_ledger`).
		WillReturnResult(sqlmock.NewResult(0, 0))

	mock.ExpectExec(`INSERT INTO finops\.prewarm_execution_ledger`).
		WithArgs(
			tenantID,        // $1 tenant_id
			&jobID,          // $2 job_id
			&primaryBOID,    // $3 bo_id (first target's BO)
			"XIRR",          // $4 target_metric (first target's FormulaType; hardcoded in stub)
			sqlmock.AnyArg(), // $5 entities_prewarmed_count
			sqlmock.AnyArg(), // $6 compute_cost
			sqlmock.AnyArg(), // $7 estimated_savings
			sqlmock.AnyArg(), // $8 peak_probability
			sqlmock.AnyArg(), // $9 policy_id
			sqlmock.AnyArg(), // $10 status
			sqlmock.AnyArg(), // $11 error_detail
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	coord := NewPrewarmCoordinator(sqlxDB)
	targets := []hotTarget{
		{BOID: primaryBOID, FieldID: uuid.New(), HitCount: 100, FormulaType: "XIRR"},
	}
	result := &PrewarmResult{
		TenantID:        tenantID,
		Status:          "SIMULATED",
		TargetsSeeded:   1,
		PeakProbability: 0.95,
	}
	err = coord.updateOrPersistLedgerEntry(context.Background(), tenantID, &jobID, targets, result, nil, "")
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestPrewarmCoordinator_TerminalUpdate_TargetMetricRunLevelAll verifies the
// run-level × UPDATE cell of the matrix: UPDATE path with no targets passes
// bo_id=nil AND target_metric='ALL' together. Guards the inverse direction
// of the original mixed-semantics bug (per-target bo_id + run-level metric).
func TestPrewarmCoordinator_TerminalUpdate_TargetMetricRunLevelAll(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	tenantID := uuid.New()
	jobID := uuid.New()

	// UPDATE path runs first (jobID non-nil) and matches the existing PENDING row.
	mock.ExpectExec(`UPDATE finops\.prewarm_execution_ledger`).
		WithArgs(
			nil,             // $1 bo_id (run-level)
			"ALL",           // $2 target_metric (run-level, explicit)
			sqlmock.AnyArg(), // $3 entities_prewarmed_count
			sqlmock.AnyArg(), // $4 compute_cost
			sqlmock.AnyArg(), // $5 estimated_savings
			sqlmock.AnyArg(), // $6 peak_probability
			sqlmock.AnyArg(), // $7 policy_id
			sqlmock.AnyArg(), // $8 status
			sqlmock.AnyArg(), // $9 error_detail
			tenantID,        // $10
			&jobID,          // $11
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	coord := NewPrewarmCoordinator(sqlxDB)
	result := &PrewarmResult{
		TenantID:      tenantID,
		Status:        "SKIPPED_BELOW_THRESHOLD",
		PeakProbability: 0.3,
	}
	err = coord.updateOrPersistLedgerEntry(context.Background(), tenantID, &jobID, nil, result, nil, "")
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
