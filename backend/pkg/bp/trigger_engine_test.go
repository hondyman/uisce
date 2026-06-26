package bp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// fakeHasura is a tiny test double for HasuraClient
type fakeHasura struct {
	queryFn  func(query string, variables map[string]interface{}) (map[string]interface{}, error)
	mutateFn func(mutation string, variables map[string]interface{}) (map[string]interface{}, error)
}

func (f *fakeHasura) Query(q string, vars map[string]interface{}) (map[string]interface{}, error) {
	if f.queryFn == nil {
		return nil, errors.New("not implemented")
	}
	return f.queryFn(q, vars)
}

func (f *fakeHasura) Mutate(m string, vars map[string]interface{}) (map[string]interface{}, error) {
	if f.mutateFn == nil {
		return nil, errors.New("not implemented")
	}
	return f.mutateFn(m, vars)
}

func TestLoadTrigger_HasuraSuccess(t *testing.T) {
	fh := &fakeHasura{}

	// Hasura returns a single trigger item
	fh.queryFn = func(query string, variables map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{
			"bp_adaptive_triggers": []interface{}{map[string]interface{}{
				"id":                "trigger-1",
				"tenant_id":         "tenant-1",
				"step_id":           "step-1",
				"trigger_name":      "My Trigger",
				"trigger_condition": "cond",
				"trigger_type":      "event",
				"action_type":       "notify",
				"action_config":     map[string]interface{}{"foo": "bar"},
				"context_variables": []interface{}{"a", "b"},
				"is_active":         true,
			}},
		}, nil
	}

	te := NewTriggerEngineWithHasura(nil, fh, nil, "tenant-1", log.New(io.Discard, "", 0))

	trg, err := te.loadTrigger(context.Background(), "trigger-1")
	if err != nil {
		t.Fatalf("loadTrigger (Hasura) returned unexpected error: %v", err)
	}

	if trg.ID != "trigger-1" || trg.TriggerName != "My Trigger" || trg.TriggerType != "event" {
		t.Fatalf("unexpected trigger contents: %#v", trg)
	}
}

func TestLoadTrigger_HasuraFailure_SQLFallback(t *testing.T) {
	ResetHasuraFallbacks()
	// Setup sqlmock DB
	dbSQL, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	sqlxDB := sqlx.NewDb(dbSQL, "sqlmock")
	defer sqlxDB.Close()

	// Expect SQL fallback query
	cols := []string{"id", "tenant_id", "step_id", "trigger_name", "trigger_condition", "trigger_type", "action_type", "action_config", "context_variables", "is_active"}

	// action_config -> bytes, context_variables -> postgres array text
	actionCfg := []byte(`{"foo":"bar"}`)
	mock.ExpectQuery("SELECT id, tenant_id, step_id, trigger_name, trigger_condition, trigger_type,").WillReturnRows(
		sqlmock.NewRows(cols).AddRow("trigger-1", "tenant-1", "step-1", "SQL Trigger", "cond", "event", "notify", actionCfg, "{a,b}", true),
	)

	fh := &fakeHasura{queryFn: func(q string, vars map[string]interface{}) (map[string]interface{}, error) {
		return nil, errors.New("hasura down")
	}}

	te := NewTriggerEngineWithHasura(sqlxDB, fh, nil, "tenant-1", log.New(io.Discard, "", 0))

	trg, err := te.loadTrigger(context.Background(), "trigger-1")
	if err != nil {
		t.Fatalf("loadTrigger SQL fallback returned error: %v", err)
	}

	if trg.TriggerName != "SQL Trigger" {
		t.Fatalf("expected SQL Trigger name, got %s", trg.TriggerName)
	}

	if got := GetHasuraFallbackCount("trigger_engine"); got != 1 {
		t.Fatalf("expected trigger_engine fallback count 1, got %d", got)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock expectations not met: %v", err)
	}
}

func TestLoadBP_HasuraSuccess(t *testing.T) {
	// Compose fake Hasura response with one BP and one step
	id := uuid.New().String()
	tid := uuid.New().String()

	fh := &fakeHasura{queryFn: func(q string, vars map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{
			"business_processes": []interface{}{map[string]interface{}{
				"id":           id,
				"tenant_id":    tid,
				"process_name": "Proc",
				"description":  "desc",
				"is_active":    true,
				"bp_steps": []interface{}{map[string]interface{}{
					"id":             uuid.New().String(),
					"process_id":     id,
					"step_order":     float64(1),
					"step_type":      "task",
					"step_name":      "Step One",
					"description":    "step desc",
					"duration_hours": float64(2),
					"assignee_role":  "admin",
					"condition_json": map[string]interface{}{"ok": true},
				}},
			}},
		}, nil
	}}

	te := NewTriggerEngineWithHasura(nil, fh, nil, tid, log.New(io.Discard, "", 0))

	bp, err := te.loadBP(context.Background(), id)
	if err != nil {
		t.Fatalf("loadBP (Hasura) returned error: %v", err)
	}

	if bp.ProcessName != "Proc" || len(bp.Steps) != 1 {
		t.Fatalf("unexpected BP parsed: %#v", bp)
	}

	// ensure the step's Config JSON was parsed and non-empty
	var cfg map[string]interface{}
	if err := json.Unmarshal(bp.Steps[0].Config, &cfg); err != nil {
		t.Fatalf("failed to unmarshal step config: %v", err)
	}
	if _, ok := cfg["ok"]; !ok {
		t.Fatalf("missing condition_json in parsed step")
	}
}

func TestRecordTriggerSuccess_HasuraAndSQLFallback(t *testing.T) {
	// sqlmock db to assert fallback path
	dbSQL, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	sqlxDB := sqlx.NewDb(dbSQL, "sqlmock")
	defer sqlxDB.Close()

	ResetHasuraFallbacks()
	// Case 1: Hasura success should not touch DB
	fhSuccess := &fakeHasura{mutateFn: func(m string, vars map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"update_bp_trigger_events": map[string]interface{}{"affected_rows": 1}}, nil
	}}

	te1 := NewTriggerEngineWithHasura(sqlxDB, fhSuccess, nil, "tenant-1", log.New(io.Discard, "", 0))
	te1.recordTriggerSuccess(context.Background(), "evt-1", "wf-123")

	// No DB expectations set — ensure nothing was executed
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock expectations not met (should be none): %v", err)
	}

	// Case 2: Hasura fails, fall back to SQL Exec
	fhFail := &fakeHasura{mutateFn: func(m string, vars map[string]interface{}) (map[string]interface{}, error) {
		return nil, errors.New("hasura mutation failed")
	}}

	te2 := NewTriggerEngineWithHasura(sqlxDB, fhFail, nil, "tenant-1", log.New(io.Discard, "", 0))

	mock.ExpectExec("UPDATE bp_trigger_events").WithArgs("wf-456", "evt-2").WillReturnResult(sqlmock.NewResult(0, 1))

	te2.recordTriggerSuccess(context.Background(), "evt-2", "wf-456")

	if got := GetHasuraFallbackCount("trigger_engine"); got != 1 {
		t.Fatalf("expected trigger_engine fallback count 1 after mutation failure, got %d", got)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock expectations not met for fallback exec: %v", err)
	}
}

func TestRecordTriggerFailure_HasuraAndSQLFallback(t *testing.T) {
	dbSQL, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	sqlxDB := sqlx.NewDb(dbSQL, "sqlmock")
	defer sqlxDB.Close()

	ResetHasuraFallbacks()
	// Hasura success: should not cause SQL Exec
	fhSuccess := &fakeHasura{mutateFn: func(m string, vars map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"update_bp_trigger_events": map[string]interface{}{"affected_rows": 1}}, nil
	}}
	te1 := NewTriggerEngineWithHasura(sqlxDB, fhSuccess, nil, "tenant-1", log.New(io.Discard, "", 0))
	te1.recordTriggerFailure(context.Background(), "evt-3", "boom")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock expectations not met for Hasura-success path: %v", err)
	}

	// Hasura fails -> SQL fallback
	fhFail := &fakeHasura{mutateFn: func(m string, vars map[string]interface{}) (map[string]interface{}, error) {
		return nil, errors.New("hasura mutation err")
	}}
	te2 := NewTriggerEngineWithHasura(sqlxDB, fhFail, nil, "tenant-1", log.New(io.Discard, "", 0))

	// Expect update to be executed with errorMsg and id
	mock.ExpectExec("UPDATE bp_trigger_events").WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 1))

	te2.recordTriggerFailure(context.Background(), "evt-4", "err-msg")

	if got := GetHasuraFallbackCount("trigger_engine"); got != 1 {
		t.Fatalf("expected trigger_engine fallback count 1 after failure fallback, got %d", got)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock expectations not met for failure fallback: %v", err)
	}
}
