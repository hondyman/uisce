package bp

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func TestSelectAIModel_HasuraSuccess(t *testing.T) {
	fh := &fakeHasura{}
	fh.queryFn = func(q string, vars map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"bp_ai_models": []interface{}{map[string]interface{}{"model_id": "m-1", "last_accuracy": 0.98, "model_endpoint": "http://example"}}}, nil
	}
	calledMut := false
	fh.mutateFn = func(m string, vars map[string]interface{}) (map[string]interface{}, error) {
		calledMut = true
		return map[string]interface{}{"update_bp_ai_models": map[string]interface{}{"affected_rows": 1}}, nil
	}

	eval := NewCompleteABranchEvaluatorWithHasura(nil, fh, "tenant-1")
	id, err := eval.SelectAIModel(context.Background(), "step-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "m-1" {
		t.Fatalf("unexpected model id: %s", id)
	}
	if !calledMut {
		t.Fatalf("expected mutate to be called")
	}
}

func TestSelectAIModel_HasuraError_SQLFallback(t *testing.T) {
	ResetHasuraFallbacks()
	// Setup sqlmock
	dbSQL, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	sqlxDB := sqlx.NewDb(dbSQL, "sqlmock")
	defer sqlxDB.Close()

	// Expect the SELECT query and then the UPDATE logging Exec
	mock.ExpectQuery("SELECT model_id, last_accuracy, model_endpoint").WithArgs("step-abc", "tenant-1").WillReturnRows(sqlmock.NewRows([]string{"model_id", "last_accuracy", "model_endpoint"}).AddRow("m-sql", 0.75, "ep"))
	mock.ExpectExec("UPDATE bp_ai_models").WithArgs("m-sql", "tenant-1").WillReturnResult(sqlmock.NewResult(0, 1))

	fh := &fakeHasura{queryFn: func(q string, vars map[string]interface{}) (map[string]interface{}, error) {
		return nil, errors.New("hasura down")
	}}

	eval := NewCompleteABranchEvaluatorWithHasura(sqlxDB, fh, "tenant-1")

	id, err := eval.SelectAIModel(context.Background(), "step-abc")
	if err != nil {
		t.Fatalf("SelectAIModel returned error: %v", err)
	}
	if id != "m-sql" {
		t.Fatalf("unexpected model id from SQL fallback: %s", id)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock expectations not met: %v", err)
	}
	if got := GetHasuraFallbackCount("branch_evaluator"); got != 1 {
		t.Fatalf("expected branch_evaluator fallback count 1, got %d", got)
	}
}

func TestEvaluateSemanticIntent_HasuraSuccess(t *testing.T) {
	fh := &fakeHasura{}
	fh.queryFn = func(q string, vars map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"bp_semantic_intents": []interface{}{map[string]interface{}{"intent_id": "intent-1", "intent_label": "Approve", "target_branch_id": "branch-1", "similarity_threshold": float64(0.8)}}}, nil
	}
	calledMut := false
	fh.mutateFn = func(m string, vars map[string]interface{}) (map[string]interface{}, error) {
		calledMut = true
		return map[string]interface{}{"update_bp_semantic_intents": map[string]interface{}{"affected_rows": 1}}, nil
	}

	eval := NewCompleteABranchEvaluatorWithHasura(nil, fh, "tenant-1")
	// inputText matches the label => calculateSemanticSimilarity returns 1.0
	route, err := eval.EvaluateSemanticIntent(context.Background(), "step-1", "Approve")
	if err != nil {
		t.Fatalf("unexpected error from EvaluateSemanticIntent: %v", err)
	}
	if route == nil {
		t.Fatalf("expected route, got nil")
	}
	if !route.ThresholdMet {
		t.Fatalf("expected ThresholdMet true, got false")
	}
	if route.TargetBranchID != "branch-1" {
		t.Fatalf("unexpected target branch: %s", route.TargetBranchID)
	}
	if !calledMut {
		t.Fatalf("expected Hasura mutation to be called on match")
	}
}

func TestEvaluateSemanticIntent_HasuraDown_SQLFallback(t *testing.T) {
	ResetHasuraFallbacks()
	dbSQL, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	sqlxDB := sqlx.NewDb(dbSQL, "sqlmock")
	defer sqlxDB.Close()

	// SQL SELECT will return a row which the evaluator will match
	mock.ExpectQuery("SELECT intent_id, intent_label, target_branch_id, similarity_threshold").WithArgs("step-xyz", "tenant-1").WillReturnRows(
		sqlmock.NewRows([]string{"intent_id", "intent_label", "target_branch_id", "similarity_threshold"}).AddRow("intent-sql", "Approve", "branch-sql", 0.5),
	)
	// Should update match_count
	mock.ExpectExec("UPDATE bp_semantic_intents").WithArgs(sqlmock.AnyArg(), "intent-sql", "tenant-1").WillReturnResult(sqlmock.NewResult(0, 1))

	fh := &fakeHasura{queryFn: func(q string, vars map[string]interface{}) (map[string]interface{}, error) {
		return nil, errors.New("hasura down")
	}}

	eval := NewCompleteABranchEvaluatorWithHasura(sqlxDB, fh, "tenant-1")
	route, err := eval.EvaluateSemanticIntent(context.Background(), "step-xyz", "Approve")
	if err != nil {
		t.Fatalf("EvaluateSemanticIntent returned error: %v", err)
	}
	if route == nil || !route.ThresholdMet || route.TargetBranchID != "branch-sql" {
		t.Fatalf("unexpected route from SQL fallback: %#v", route)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock expectations not met: %v", err)
	}
	if got := GetHasuraFallbackCount("branch_evaluator"); got != 2 {
		t.Fatalf("expected branch_evaluator fallback count 2, got %d", got)
	}
}

func TestEvaluateScoringMatrix_HasuraSuccess(t *testing.T) {
	fh := &fakeHasura{}
	fh.queryFn = func(q string, vars map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"bp_scoring_matrices": []interface{}{map[string]interface{}{
			"id":                 "matrix-1",
			"dimensions":         []interface{}{map[string]interface{}{"name": "urgency", "weight": float64(1)}, map[string]interface{}{"name": "complexity", "weight": float64(0.5)}},
			"routing_thresholds": []interface{}{map[string]interface{}{"min_score": float64(5), "branch_id": "branch-A"}, map[string]interface{}{"min_score": float64(3), "branch_id": "branch-B"}},
		}}}, nil
	}
	calledMut := false
	fh.mutateFn = func(m string, vars map[string]interface{}) (map[string]interface{}, error) {
		calledMut = true
		return map[string]interface{}{"update_bp_scoring_matrices": map[string]interface{}{"affected_rows": 1}}, nil
	}

	eval := NewCompleteABranchEvaluatorWithHasura(nil, fh, "tenant-1")
	result, err := eval.EvaluateScoringMatrix(context.Background(), "step-1", map[string]interface{}{"priority": "high", "complexity": "low"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// urgency: priority=high -> 10 *1 = 10 ; complexity: low -> 3 *0.5 = 1.5 => total 11.5
	if result == nil {
		t.Fatalf("expected non-nil result")
	}
	if result.TotalScore < 11.4 || result.TotalScore > 11.6 {
		t.Fatalf("unexpected total score: %v", result.TotalScore)
	}
	if result.SelectedRoute != "branch-A" {
		t.Fatalf("unexpected selected route: %s", result.SelectedRoute)
	}
	if !calledMut {
		t.Fatalf("expected Hasura mutation to be called")
	}
}

func TestEvaluateScoringMatrix_HasuraDown_SQLFallback(t *testing.T) {
	ResetHasuraFallbacks()
	dbSQL, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	sqlxDB := sqlx.NewDb(dbSQL, "sqlmock")
	defer sqlxDB.Close()

	dims := []map[string]interface{}{{"name": "urgency", "weight": 1}, {"name": "complexity", "weight": 0.5}}
	thresholds := []map[string]interface{}{{"min_score": 5, "branch_id": "branch-sql"}}

	dimsB, _ := json.Marshal(dims)
	threshB, _ := json.Marshal(thresholds)

	mock.ExpectQuery("SELECT id, dimensions, routing_thresholds").WithArgs("step-xyz", "tenant-1").WillReturnRows(
		sqlmock.NewRows([]string{"id", "dimensions", "routing_thresholds"}).AddRow("matrix-sql", dimsB, threshB),
	)
	// expect fallback update
	mock.ExpectExec("UPDATE bp_scoring_matrices").WithArgs(sqlmock.AnyArg(), "matrix-sql", "tenant-1").WillReturnResult(sqlmock.NewResult(0, 1))

	fh := &fakeHasura{queryFn: func(q string, vars map[string]interface{}) (map[string]interface{}, error) {
		return nil, errors.New("hasura down")
	}}

	eval := NewCompleteABranchEvaluatorWithHasura(sqlxDB, fh, "tenant-1")
	result, err := eval.EvaluateScoringMatrix(context.Background(), "step-xyz", map[string]interface{}{"priority": "high", "complexity": "low"})
	if err != nil {
		t.Fatalf("unexpected error from EvaluateScoringMatrix: %v", err)
	}
	if result.TotalScore < 11.4 || result.TotalScore > 11.6 {
		t.Fatalf("unexpected total score from SQL fallback: %v", result.TotalScore)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock expectations not met: %v", err)
	}
	if got := GetHasuraFallbackCount("branch_evaluator"); got != 2 {
		t.Fatalf("expected branch_evaluator fallback count 2, got %d", got)
	}
}

func TestEvaluateAdaptiveTriggers_HasuraSuccess(t *testing.T) {
	fh := &fakeHasura{}
	fh.queryFn = func(q string, vars map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"bp_adaptive_triggers": []interface{}{map[string]interface{}{"action_type": "notify", "action_config": map[string]interface{}{"foo": "bar"}, "is_active": true}}}, nil
	}
	calledMut := false
	fh.mutateFn = func(m string, vars map[string]interface{}) (map[string]interface{}, error) {
		calledMut = true
		return map[string]interface{}{"update_bp_adaptive_triggers": map[string]interface{}{"affected_rows": 1}}, nil
	}

	eval := NewCompleteABranchEvaluatorWithHasura(nil, fh, "tenant-1")
	ctxData := map[string]interface{}{"duration_ms": float64(40000)}
	res, err := eval.EvaluateAdaptiveTriggers(context.Background(), "step-x", ctxData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.TriggeredYes || res.ActionType != "notify" {
		t.Fatalf("unexpected adaptive trigger result: %#v", res)
	}
	if !calledMut {
		t.Fatalf("expected Hasura mutate to be called")
	}
}

func TestEvaluateAdaptiveTriggers_HasuraDown_SQLFallback(t *testing.T) {
	ResetHasuraFallbacks()
	dbSQL, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	sqlxDB := sqlx.NewDb(dbSQL, "sqlmock")
	defer sqlxDB.Close()

	actionCfg := []byte(`{"foo":"bar"}`)
	mock.ExpectQuery("SELECT action_type, action_config, is_active").WithArgs("step-xyz", "tenant-1").WillReturnRows(sqlmock.NewRows([]string{"action_type", "action_config", "is_active"}).AddRow("notify", actionCfg, true))
	mock.ExpectExec("UPDATE bp_adaptive_triggers").WithArgs("step-xyz", "tenant-1").WillReturnResult(sqlmock.NewResult(0, 1))

	fh := &fakeHasura{queryFn: func(q string, vars map[string]interface{}) (map[string]interface{}, error) {
		return nil, errors.New("hasura down")
	}}

	eval := NewCompleteABranchEvaluatorWithHasura(sqlxDB, fh, "tenant-1")
	ctxData := map[string]interface{}{"duration_ms": float64(40000)}
	res, err := eval.EvaluateAdaptiveTriggers(context.Background(), "step-xyz", ctxData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.TriggeredYes || res.ActionType != "notify" {
		t.Fatalf("unexpected result from SQL fallback: %#v", res)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock expectations not met: %v", err)
	}
	if got := GetHasuraFallbackCount("branch_evaluator"); got != 2 {
		t.Fatalf("expected branch_evaluator fallback count 2, got %d", got)
	}
}

func TestRecordBranchAnalytics_HasuraSuccess(t *testing.T) {
	fh := &fakeHasura{}
	called := false
	fh.mutateFn = func(m string, vars map[string]interface{}) (map[string]interface{}, error) {
		called = true
		return map[string]interface{}{"insert_bp_branch_analytics_extended_one": map[string]interface{}{"tenant_id": "tenant-1", "branch_id": "b-1"}}, nil
	}

	eval := NewCompleteABranchEvaluatorWithHasura(nil, fh, "tenant-1")
	err := eval.RecordBranchAnalytics(context.Background(), "b-1", map[string]interface{}{"selections": 1, "avg_duration": 100, "success_rate": 0.9, "anomaly_score": 0.1})
	if err != nil {
		t.Fatalf("RecordBranchAnalytics returned unexpected error: %v", err)
	}
	if !called {
		t.Fatalf("expected Hasura mutate to be called")
	}
}

func TestRecordBranchAnalytics_HasuraDown_SQLFallback(t *testing.T) {
	ResetHasuraFallbacks()
	dbSQL, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	sqlxDB := sqlx.NewDb(dbSQL, "sqlmock")
	defer sqlxDB.Close()

	// Expect INSERT...ON CONFLICT Exec
	mock.ExpectExec("INSERT INTO bp_branch_analytics_extended").WithArgs("tenant-1", "b-2", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))

	fh := &fakeHasura{mutateFn: func(m string, vars map[string]interface{}) (map[string]interface{}, error) {
		return nil, errors.New("hasura down")
	}}

	eval := NewCompleteABranchEvaluatorWithHasura(sqlxDB, fh, "tenant-1")
	err = eval.RecordBranchAnalytics(context.Background(), "b-2", map[string]interface{}{"selections": 1, "avg_duration": 100, "success_rate": 0.9, "anomaly_score": 0.1})
	if err != nil {
		t.Fatalf("RecordBranchAnalytics returned unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock expectations not met: %v", err)
	}
	if got := GetHasuraFallbackCount("branch_evaluator"); got != 1 {
		t.Fatalf("expected branch_evaluator fallback count 1, got %d", got)
	}
}

func TestGetTimeSeriesForecast_HasuraSuccess(t *testing.T) {
	fh := &fakeHasura{}
	fh.queryFn = func(q string, vars map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"bp_time_series_forecasts": []interface{}{map[string]interface{}{
			"predicted_queue_depth":           float64(100),
			"predicted_approval_time_minutes": float64(30),
			"confidence_interval_lower":       float64(0.1),
			"confidence_interval_upper":       float64(0.9),
			"low_load_branch_id":              "low-branch",
			"high_load_branch_id":             "high-branch",
		}}}, nil
	}

	eval := NewCompleteABranchEvaluatorWithHasura(nil, fh, "tenant-1")
	res, err := eval.GetTimeSeriesForecast(context.Background(), "step-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.PredictedQueueDepth != 100 {
		t.Fatalf("unexpected queue depth: %d", res.PredictedQueueDepth)
	}
	if res.RecommendedBranchID != "high-branch" {
		t.Fatalf("unexpected recommended branch: %s", res.RecommendedBranchID)
	}
}

func TestGetTimeSeriesForecast_HasuraDown_SQLFallback(t *testing.T) {
	ResetHasuraFallbacks()
	dbSQL, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	sqlxDB := sqlx.NewDb(dbSQL, "sqlmock")
	defer sqlxDB.Close()

	mock.ExpectQuery("SELECT predicted_queue_depth, predicted_approval_time_minutes").WithArgs("step-xyz", "tenant-1").WillReturnRows(
		sqlmock.NewRows([]string{"predicted_queue_depth", "predicted_approval_time_minutes", "confidence_interval_lower", "confidence_interval_upper", "low_load_branch_id", "high_load_branch_id"}).AddRow(10, 5, 0.1, 0.9, "low-b", "high-b"),
	)

	fh := &fakeHasura{queryFn: func(q string, vars map[string]interface{}) (map[string]interface{}, error) {
		return nil, errors.New("hasura down")
	}}

	eval := NewCompleteABranchEvaluatorWithHasura(sqlxDB, fh, "tenant-1")
	res, err := eval.GetTimeSeriesForecast(context.Background(), "step-xyz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.PredictedQueueDepth != 10 {
		t.Fatalf("unexpected queue depth from SQL fallback: %d", res.PredictedQueueDepth)
	}
	if res.RecommendedBranchID != "low-b" {
		t.Fatalf("unexpected recommended branch from SQL fallback: %s", res.RecommendedBranchID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock expectations not met: %v", err)
	}
	if got := GetHasuraFallbackCount("branch_evaluator"); got != 1 {
		t.Fatalf("expected branch_evaluator fallback count 1, got %d", got)
	}
}

func TestCastVote_HasuraSuccess(t *testing.T) {
	fh := &fakeHasura{}
	called := false
	fh.mutateFn = func(m string, vars map[string]interface{}) (map[string]interface{}, error) {
		called = true
		return map[string]interface{}{"update_bp_collaborative_decisions": map[string]interface{}{"affected_rows": 1}}, nil
	}

	eval := NewCompleteABranchEvaluatorWithHasura(nil, fh, "tenant-1")
	if err := eval.CastVote(context.Background(), "dec-1", "role", "yes"); err != nil {
		t.Fatalf("CastVote returned unexpected error: %v", err)
	}
	if !called {
		t.Fatalf("expected Hasura mutate to be called")
	}
}

func TestCastVote_HasuraDown_SQLFallback(t *testing.T) {
	ResetHasuraFallbacks()
	dbSQL, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	sqlxDB := sqlx.NewDb(dbSQL, "sqlmock")
	defer sqlxDB.Close()

	mock.ExpectExec("UPDATE bp_collaborative_decisions").WithArgs("dec-2", "tenant-1").WillReturnResult(sqlmock.NewResult(0, 1))

	fh := &fakeHasura{mutateFn: func(m string, vars map[string]interface{}) (map[string]interface{}, error) {
		return nil, errors.New("hasura down")
	}}

	eval := NewCompleteABranchEvaluatorWithHasura(sqlxDB, fh, "tenant-1")
	if err := eval.CastVote(context.Background(), "dec-2", "role", "no"); err != nil {
		t.Fatalf("CastVote returned unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock expectations not met: %v", err)
	}
	if got := GetHasuraFallbackCount("branch_evaluator"); got != 1 {
		t.Fatalf("expected branch_evaluator fallback count 1, got %d", got)
	}
}

func TestLogBlockchainAudit_HasuraSuccess(t *testing.T) {
	fh := &fakeHasura{}
	called := false
	fh.mutateFn = func(m string, vars map[string]interface{}) (map[string]interface{}, error) {
		called = true
		return map[string]interface{}{"insert_bp_blockchain_audit_one": map[string]interface{}{"tenant_id": "tenant-1", "workflow_instance_id": "wf-1", "event_hash": "abc"}}, nil
	}

	eval := NewCompleteABranchEvaluatorWithHasura(nil, fh, "tenant-1")
	if err := eval.LogBlockchainAudit(context.Background(), "wf-1", "decision"); err != nil {
		t.Fatalf("LogBlockchainAudit returned error: %v", err)
	}
	if !called {
		t.Fatalf("expected Hasura mutate to be called")
	}
}

func TestLogBlockchainAudit_HasuraDown_SQLFallback(t *testing.T) {
	ResetHasuraFallbacks()
	dbSQL, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	sqlxDB := sqlx.NewDb(dbSQL, "sqlmock")
	defer sqlxDB.Close()

	mock.ExpectExec("INSERT INTO bp_blockchain_audit").WithArgs("tenant-1", "wf-2", sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))

	fh := &fakeHasura{mutateFn: func(m string, vars map[string]interface{}) (map[string]interface{}, error) {
		return nil, errors.New("hasura down")
	}}

	eval := NewCompleteABranchEvaluatorWithHasura(sqlxDB, fh, "tenant-1")
	if err := eval.LogBlockchainAudit(context.Background(), "wf-2", "decision"); err != nil {
		t.Fatalf("LogBlockchainAudit returned error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock expectations not met: %v", err)
	}
	if got := GetHasuraFallbackCount("branch_evaluator"); got != 1 {
		t.Fatalf("expected branch_evaluator fallback count 1, got %d", got)
	}
}

func TestRecordExplainability_HasuraSuccess(t *testing.T) {
	fh := &fakeHasura{}
	called := false
	fh.mutateFn = func(m string, vars map[string]interface{}) (map[string]interface{}, error) {
		called = true
		return map[string]interface{}{"insert_bp_explainability_records_one": map[string]interface{}{"record_id": "r-1", "selected_branch_id": "b-1", "decision_confidence": 0.9}}, nil
	}

	eval := NewCompleteABranchEvaluatorWithHasura(nil, fh, "tenant-1")
	if err := eval.RecordExplainability(context.Background(), "exec-1", "b-1", map[string]float64{"a": 1.0}); err != nil {
		t.Fatalf("RecordExplainability returned unexpected error: %v", err)
	}
	if !called {
		t.Fatalf("expected Hasura mutate to be called")
	}
}

func TestRecordExplainability_HasuraDown_SQLFallback(t *testing.T) {
	ResetHasuraFallbacks()
	dbSQL, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	sqlxDB := sqlx.NewDb(dbSQL, "sqlmock")
	defer sqlxDB.Close()

	mock.ExpectExec("INSERT INTO bp_explainability_records").WithArgs("tenant-1", "exec-2", "b-2", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))

	fh := &fakeHasura{mutateFn: func(m string, vars map[string]interface{}) (map[string]interface{}, error) {
		return nil, errors.New("hasura down")
	}}

	eval := NewCompleteABranchEvaluatorWithHasura(sqlxDB, fh, "tenant-1")
	if err := eval.RecordExplainability(context.Background(), "exec-2", "b-2", map[string]float64{"a": 1.0}); err != nil {
		t.Fatalf("RecordExplainability returned unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock expectations not met: %v", err)
	}
	if got := GetHasuraFallbackCount("branch_evaluator"); got != 1 {
		t.Fatalf("expected branch_evaluator fallback count 1, got %d", got)
	}
}
