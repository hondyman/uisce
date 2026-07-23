package bp

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func TestEvaluateAIModels_HasuraSuccess(t *testing.T) {
	ResetHasuraFallbacks()
	fh := &fakeHasura{}
	// Hasura returns model performance and mutate succeeds
	fh.queryFn = func(q string, vars map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"bp_ai_models": []interface{}{map[string]interface{}{"model_id": "m-x", "last_accuracy": 0.92, "predictions_count": float64(10), "drift_detected": false}}}, nil
	}
	mutCalled := false
	fh.mutateFn = func(m string, vars map[string]interface{}) (map[string]interface{}, error) {
		mutCalled = true
		return map[string]interface{}{"update_bp_ai_models": map[string]interface{}{"affected_rows": 1}}, nil
	}

	eval := NewBranchEvaluatorWithHasura(nil, fh)

	// Compose config with AvailableModels to select
	cfg := map[string]interface{}{"available_models": []map[string]interface{}{{"model_id": "m-x", "last_accuracy": 0.9, "accuracy_threshold": 0.5}}, "auto_switch_enabled": false}
	cfgB, _ := json.Marshal(cfg)

	branch, err := eval.EvaluateAIModels(context.Background(), cfgB, "tenant-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branch != "m-x_predicted_branch" {
		t.Fatalf("unexpected predicted branch: %s", branch)
	}
	if !mutCalled {
		t.Fatalf("expected Hasura mutate to be invoked")
	}
}

func TestEvaluateAIModels_HasuraDown_SQLFallback(t *testing.T) {
	ResetHasuraFallbacks()
	dbSQL, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	sqlxDB := sqlx.NewDb(dbSQL, "sqlmock")
	defer sqlxDB.Close()

	// Expect select to return a row
	mock.ExpectQuery("SELECT model_id, last_accuracy, predictions_count, drift_detected").WithArgs("m-sql", "tenant-1").WillReturnRows(sqlmock.NewRows([]string{"model_id", "last_accuracy", "predictions_count", "drift_detected"}).AddRow("m-sql", 0.8, int64(2), false))
	// Expect update to be executed as fallback to increment count
	mock.ExpectExec("UPDATE bp_ai_models").WithArgs("m-sql", "tenant-1").WillReturnResult(sqlmock.NewResult(0, 1))

	fh := &fakeHasura{queryFn: func(q string, vars map[string]interface{}) (map[string]interface{}, error) {
		return nil, errors.New("hasura down")
	}}

	eval := NewBranchEvaluatorWithHasura(sqlxDB, fh)

	cfg := map[string]interface{}{"available_models": []map[string]interface{}{{"model_id": "m-sql", "last_accuracy": 0.85, "accuracy_threshold": 0.5}}, "auto_switch_enabled": false}
	cfgB, _ := json.Marshal(cfg)

	branch, err := eval.EvaluateAIModels(context.Background(), cfgB, "tenant-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branch != "m-sql_predicted_branch" {
		t.Fatalf("unexpected predicted branch (sql): %s", branch)
	}

	// Hasura query + mutate both fail -> 2 fallbacks expected
	if got := GetHasuraFallbackCount("branch_evaluator"); got != 2 {
		t.Fatalf("expected branch_evaluator fallback count 2, got %d", got)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock expectations not met: %v", err)
	}
}

func TestEvaluateSemanticIntent_BranchEvaluator_HasuraSuccess(t *testing.T) {
	ResetHasuraFallbacks()
	fh := &fakeHasura{}
	called := false
	fh.mutateFn = func(m string, vars map[string]interface{}) (map[string]interface{}, error) {
		called = true
		return map[string]interface{}{"insert_bp_semantic_intents_one": map[string]interface{}{"intent_id": "intent-1"}}, nil
	}

	eval := NewBranchEvaluatorWithHasura(nil, fh)

	// Use same embedding as calculateEmbedding (0.5 per element) so similarity threshold is met
	vec := make([]float64, 384)
	for i := range vec {
		vec[i] = 0.5
	}
	cfg := map[string]interface{}{"intent_description": "Approve invoice", "intents": []map[string]interface{}{{"intent_id": "intent-1", "target_branch": "branch-ok", "similarity_threshold": 0.5, "vector": vec}}}
	cfgB, _ := json.Marshal(cfg)

	branch, err := eval.EvaluateSemanticIntent(context.Background(), cfgB, "entity-1", "tenant-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branch != "branch-ok" {
		t.Fatalf("unexpected branch: %s", branch)
	}
	if !called {
		t.Fatalf("expected Hasura mutate for semantic intent logging")
	}
}

func TestEvaluateSemanticIntent_BranchEvaluator_HasuraDown_SQLFallback(t *testing.T) {
	ResetHasuraFallbacks()
	dbSQL, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	sqlxDB := sqlx.NewDb(dbSQL, "sqlmock")
	defer sqlxDB.Close()

	// Expect SQL exec to log the match
	mock.ExpectExec("INSERT INTO bp_semantic_intents").WithArgs("intent-sql", sqlmock.AnyArg(), "tenant-1").WillReturnResult(sqlmock.NewResult(1, 1))

	fh := &fakeHasura{mutateFn: func(m string, vars map[string]interface{}) (map[string]interface{}, error) {
		return nil, errors.New("hasura mutate down")
	}}

	eval := NewBranchEvaluatorWithHasura(sqlxDB, fh)

	// Use same embedding as calculateEmbedding (0.5 per element) so similarity threshold is met
	vec := make([]float64, 384)
	for i := range vec {
		vec[i] = 0.5
	}
	cfg := map[string]interface{}{"intent_description": "Approve invoice", "intents": []map[string]interface{}{{"intent_id": "intent-sql", "target_branch": "branch-sql", "similarity_threshold": 0.5, "vector": vec}}}
	cfgB, _ := json.Marshal(cfg)

	branch, err := eval.EvaluateSemanticIntent(context.Background(), cfgB, "entity-1", "tenant-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branch != "branch-sql" {
		t.Fatalf("unexpected branch: %s", branch)
	}

	if got := GetHasuraFallbackCount("branch_evaluator"); got != 1 {
		t.Fatalf("expected branch_evaluator fallback count 1, got %d", got)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock expectations not met: %v", err)
	}
}

func TestEvaluateScoringMatrix_BranchEvaluator_HasuraSuccess(t *testing.T) {
	ResetHasuraFallbacks()
	fh := &fakeHasura{}
	called := false
	fh.mutateFn = func(m string, vars map[string]interface{}) (map[string]interface{}, error) {
		called = true
		return map[string]interface{}{"insert_bp_scoring_matrices_one": map[string]interface{}{"matrix_name": "risk_score"}}, nil
	}

	eval := NewBranchEvaluatorWithHasura(nil, fh)

	// config that yields a finalScore >= 5
	cfg := map[string]interface{}{"matrix_name": "risk_score", "dimensions": []map[string]interface{}{{"name": "urgency", "weight": float64(1), "scoring_rules": []map[string]interface{}{{"condition": "x", "score": float64(6)}}}}, "routing_thresholds": []map[string]interface{}{{"min_score": float64(5), "branch_id": "branch-A"}}}
	cfgB, _ := json.Marshal(cfg)

	branch, err := eval.EvaluateScoringMatrix(context.Background(), cfgB, map[string]interface{}{"priority": "high"}, "tenant-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branch != "branch-A" {
		t.Fatalf("unexpected branch: %s", branch)
	}
	if !called {
		t.Fatalf("expected Hasura mutate to be invoked")
	}
	if got := GetHasuraFallbackCount("branch_evaluator"); got != 0 {
		t.Fatalf("expected 0 fallbacks, got %d", got)
	}
}

func TestEvaluateScoringMatrix_BranchEvaluator_HasuraDown_SQLFallback(t *testing.T) {
	ResetHasuraFallbacks()
	dbSQL, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	sqlxDB := sqlx.NewDb(dbSQL, "sqlmock")
	defer sqlxDB.Close()

	// Expect SQL exec to log the evaluation
	mock.ExpectExec("INSERT INTO bp_scoring_matrices").WithArgs("risk_score", sqlmock.AnyArg(), "tenant-1").WillReturnResult(sqlmock.NewResult(1, 1))

	fh := &fakeHasura{mutateFn: func(m string, vars map[string]interface{}) (map[string]interface{}, error) {
		return nil, errors.New("hasura mutate down")
	}}

	eval := NewBranchEvaluatorWithHasura(sqlxDB, fh)

	cfg := map[string]interface{}{"matrix_name": "risk_score", "dimensions": []map[string]interface{}{{"name": "urgency", "weight": float64(1), "scoring_rules": []map[string]interface{}{{"condition": "x", "score": float64(6)}}}}, "routing_thresholds": []map[string]interface{}{{"min_score": float64(5), "branch_id": "branch-A"}}}
	cfgB, _ := json.Marshal(cfg)

	branch, err := eval.EvaluateScoringMatrix(context.Background(), cfgB, map[string]interface{}{"priority": "high"}, "tenant-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branch != "branch-A" {
		t.Fatalf("unexpected branch: %s", branch)
	}

	if got := GetHasuraFallbackCount("branch_evaluator"); got != 1 {
		t.Fatalf("expected branch_evaluator fallback count 1, got %d", got)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock expectations not met: %v", err)
	}
}

func TestEvaluateTimeSeries_BranchEvaluator_HasuraSuccess(t *testing.T) {
	ResetHasuraFallbacks()
	fh := &fakeHasura{}
	fh.queryFn = func(q string, vars map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"bp_time_series_forecasts": []interface{}{map[string]interface{}{"predicted_queue_depth": float64(3), "predicted_approval_time_minutes": float64(10), "forecast_accuracy": float64(0.9)}}}, nil
	}

	eval := NewBranchEvaluatorWithHasura(nil, fh)

	cfg := map[string]interface{}{"forecast_model": "arima", "branches": []map[string]interface{}{{"condition": "queue_low", "branch_id": "branch-low"}, {"condition": "queue_high", "branch_id": "branch-high"}}}
	cfgB, _ := json.Marshal(cfg)

	branch, err := eval.EvaluateTimeSeries(context.Background(), cfgB, "tenant-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branch != "branch-low" {
		t.Fatalf("unexpected branch: %s", branch)
	}
	if got := GetHasuraFallbackCount("branch_evaluator"); got != 0 {
		t.Fatalf("expected 0 fallbacks, got %d", got)
	}
}

func TestEvaluateTimeSeries_BranchEvaluator_HasuraDown_SQLFallback(t *testing.T) {
	ResetHasuraFallbacks()
	dbSQL, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	sqlxDB := sqlx.NewDb(dbSQL, "sqlmock")
	defer sqlxDB.Close()

	// sqlmock: return a high queue depth so the 'queue_high' condition matches
	mock.ExpectQuery("SELECT predicted_queue_depth, predicted_approval_time_minutes, forecast_accuracy").WithArgs("arima", "tenant-1").WillReturnRows(sqlmock.NewRows([]string{"predicted_queue_depth", "predicted_approval_time_minutes", "forecast_accuracy"}).AddRow(25, 15, 0.8))

	fh := &fakeHasura{queryFn: func(q string, vars map[string]interface{}) (map[string]interface{}, error) {
		return nil, errors.New("hasura down")
	}}

	eval := NewBranchEvaluatorWithHasura(sqlxDB, fh)

	cfg := map[string]interface{}{"forecast_model": "arima", "branches": []map[string]interface{}{{"condition": "queue_low", "branch_id": "branch-low"}, {"condition": "queue_high", "branch_id": "branch-high"}}}
	cfgB, _ := json.Marshal(cfg)

	branch, err := eval.EvaluateTimeSeries(context.Background(), cfgB, "tenant-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branch != "branch-high" {
		t.Fatalf("unexpected branch: %s", branch)
	}

	if got := GetHasuraFallbackCount("branch_evaluator"); got != 1 {
		t.Fatalf("expected branch_evaluator fallback count 1, got %d", got)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock expectations not met: %v", err)
	}
}

func TestEvaluateAdaptive_BranchEvaluator_HasuraSuccess(t *testing.T) {
	ResetHasuraFallbacks()
	fh := &fakeHasura{}
	called := false
	fh.mutateFn = func(m string, vars map[string]interface{}) (map[string]interface{}, error) {
		called = true
		return map[string]interface{}{"insert_bp_adaptive_triggers_one": map[string]interface{}{"trigger_id": "t-1"}}, nil
	}

	eval := NewBranchEvaluatorWithHasura(nil, fh)

	cfg := map[string]interface{}{"adaptation_triggers": []map[string]interface{}{{"trigger_id": "t-1", "trigger_type": "duration", "condition": "step_too_long", "action_type": "switch_to_branch", "target_branch": "branch-ok"}}}
	cfgB, _ := json.Marshal(cfg)

	history := map[string]interface{}{"step_duration_ms": float64(40000)}

	branch, err := eval.EvaluateAdaptive(context.Background(), cfgB, history, "tenant-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branch != "branch-ok" {
		t.Fatalf("unexpected branch: %s", branch)
	}
	if !called {
		t.Fatalf("expected Hasura mutate to be invoked")
	}
	if got := GetHasuraFallbackCount("branch_evaluator"); got != 0 {
		t.Fatalf("expected 0 fallbacks, got %d", got)
	}
}

func TestEvaluateAdaptive_BranchEvaluator_HasuraDown_SQLFallback(t *testing.T) {
	ResetHasuraFallbacks()
	dbSQL, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	sqlxDB := sqlx.NewDb(dbSQL, "sqlmock")
	defer sqlxDB.Close()

	// Expect SQL exec logging
	mock.ExpectExec("INSERT INTO bp_adaptive_triggers").WithArgs("t-sql", "tenant-1").WillReturnResult(sqlmock.NewResult(1, 1))

	fh := &fakeHasura{mutateFn: func(m string, vars map[string]interface{}) (map[string]interface{}, error) {
		return nil, errors.New("hasura mutate down")
	}}

	eval := NewBranchEvaluatorWithHasura(sqlxDB, fh)

	cfg := map[string]interface{}{"adaptation_triggers": []map[string]interface{}{{"trigger_id": "t-sql", "trigger_type": "duration", "condition": "step_too_long", "action_type": "switch_to_branch", "target_branch": "branch-sql"}}}
	cfgB, _ := json.Marshal(cfg)

	history := map[string]interface{}{"step_duration_ms": float64(40000)}

	branch, err := eval.EvaluateAdaptive(context.Background(), cfgB, history, "tenant-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branch != "branch-sql" {
		t.Fatalf("unexpected branch: %s", branch)
	}

	if got := GetHasuraFallbackCount("branch_evaluator"); got != 1 {
		t.Fatalf("expected branch_evaluator fallback count 1, got %d", got)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock expectations not met: %v", err)
	}
}

func TestEvaluateResilience_BranchEvaluator_HasuraSuccess_NoFallback(t *testing.T) {
	ResetHasuraFallbacks()
	fh := &fakeHasura{}
	// Hasura returns policy with threshold 2 and failure_count 1 -> no fallback
	fh.queryFn = func(q string, vars map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"bp_resilience_policies": []interface{}{map[string]interface{}{"retry_max_attempts": float64(3), "retry_initial_interval_seconds": float64(5), "circuit_breaker_failure_threshold": float64(2), "failure_count": float64(1), "fallback_branch_id": "fallback-1"}}}, nil
	}

	eval := NewBranchEvaluatorWithHasura(nil, fh)

	branch, err := eval.EvaluateResilience(context.Background(), "policy-x", "target-branch", "tenant-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branch != "target-branch" {
		t.Fatalf("expected original target branch, got %s", branch)
	}
	if got := GetHasuraFallbackCount("branch_evaluator"); got != 0 {
		t.Fatalf("expected 0 fallbacks, got %d", got)
	}
}

func TestEvaluateResilience_BranchEvaluator_HasuraDown_SQLFallback_FallbackBranch(t *testing.T) {
	ResetHasuraFallbacks()
	dbSQL, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	sqlxDB := sqlx.NewDb(dbSQL, "sqlmock")
	defer sqlxDB.Close()

	// SQL returns policy with threshold 2
	mock.ExpectQuery("SELECT retry_max_attempts, retry_initial_interval_seconds,").WithArgs("policy-x", "tenant-1").WillReturnRows(sqlmock.NewRows([]string{"retry_max_attempts", "retry_initial_interval_seconds", "circuit_breaker_failure_threshold", "fallback_branch_id"}).AddRow(3, 5, 2, "fallback-1"))
	// Then the failure_count query returns 3 (> threshold 2)
	mock.ExpectQuery("SELECT failure_count FROM bp_resilience_policies").WithArgs("policy-x", "tenant-1").WillReturnRows(sqlmock.NewRows([]string{"failure_count"}).AddRow(3))

	fh := &fakeHasura{queryFn: func(q string, vars map[string]interface{}) (map[string]interface{}, error) {
		return nil, errors.New("hasura down")
	}}

	eval := NewBranchEvaluatorWithHasura(sqlxDB, fh)

	branch, err := eval.EvaluateResilience(context.Background(), "policy-x", "target-branch", "tenant-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branch != "fallback-1" {
		t.Fatalf("expected fallback branch, got %s", branch)
	}

	if got := GetHasuraFallbackCount("branch_evaluator"); got != 1 {
		t.Fatalf("expected branch_evaluator fallback count 1, got %d", got)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock expectations not met: %v", err)
	}
}

func TestEvaluateExplainability_BranchEvaluator_HasuraSuccess(t *testing.T) {
	ResetHasuraFallbacks()
	fh := &fakeHasura{}
	fh.mutateFn = func(m string, vars map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"insert_bp_explainability_records_one": map[string]interface{}{"record_id": "rec-1"}}, nil
	}

	eval := NewBranchEvaluatorWithHasura(nil, fh)

	features := map[string]float64{"salary": 0.45, "age": 0.3}
	rec, err := eval.EvaluateExplainability(context.Background(), "branch-1", features, "tenant-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec == nil || rec.RecordID != "rec-1" {
		t.Fatalf("unexpected record returned: %#v", rec)
	}
	if got := GetHasuraFallbackCount("branch_evaluator"); got != 0 {
		t.Fatalf("expected 0 fallbacks, got %d", got)
	}
}

func TestEvaluateExplainability_BranchEvaluator_HasuraDown_SQLFallback(t *testing.T) {
	ResetHasuraFallbacks()
	dbSQL, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	sqlxDB := sqlx.NewDb(dbSQL, "sqlmock")
	defer sqlxDB.Close()

	// Expect insert to return id
	mock.ExpectQuery("INSERT INTO bp_explainability_records").WithArgs("branch-1", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), float64(0.94), "tenant-1").WillReturnRows(sqlmock.NewRows([]string{"record_id"}).AddRow("rec-sql-1"))

	fh := &fakeHasura{mutateFn: func(m string, vars map[string]interface{}) (map[string]interface{}, error) {
		return nil, errors.New("hasura mutate down")
	}}

	eval := NewBranchEvaluatorWithHasura(sqlxDB, fh)

	features := map[string]float64{"salary": 0.45, "age": 0.3}
	rec, err := eval.EvaluateExplainability(context.Background(), "branch-1", features, "tenant-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec == nil || rec.RecordID != "rec-sql-1" {
		t.Fatalf("unexpected record returned: %#v", rec)
	}
	if got := GetHasuraFallbackCount("branch_evaluator"); got != 1 {
		t.Fatalf("expected branch_evaluator fallback count 1, got %d", got)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock expectations not met: %v", err)
	}
}

func TestEvaluateAnalytics_BranchEvaluator_HasuraSuccess(t *testing.T) {
	ResetHasuraFallbacks()
	fh := &fakeHasura{}
	fh.queryFn = func(q string, vars map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"bp_branch_analytics_extended": []interface{}{map[string]interface{}{"selection_count": float64(100), "completion_count": float64(80), "abandonment_count": float64(20), "avg_duration_ms": float64(5000), "anomaly_score": float64(0.1), "trend_direction": "up"}}}, nil
	}

	eval := NewBranchEvaluatorWithHasura(nil, fh)

	analytics, err := eval.EvaluateAnalytics(context.Background(), "branch-1", "tenant-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if analytics == nil || analytics.SelectionCount != 100 || analytics.TrendDirection != "up" {
		t.Fatalf("unexpected analytics returned: %#v", analytics)
	}
	if got := GetHasuraFallbackCount("branch_evaluator"); got != 0 {
		t.Fatalf("expected 0 fallbacks, got %d", got)
	}
}

func TestEvaluateAnalytics_BranchEvaluator_HasuraDown_SQLFallback(t *testing.T) {
	ResetHasuraFallbacks()
	dbSQL, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	sqlxDB := sqlx.NewDb(dbSQL, "sqlmock")
	defer sqlxDB.Close()

	mock.ExpectQuery("SELECT selection_count, completion_count, abandonment_count,").WithArgs("branch-1", "tenant-1").WillReturnRows(sqlmock.NewRows([]string{"selection_count", "completion_count", "abandonment_count", "avg_duration_ms", "anomaly_score", "trend_direction"}).AddRow(int64(50), int64(40), int64(10), float64(3000), float64(0.05), "stable"))

	fh := &fakeHasura{queryFn: func(q string, vars map[string]interface{}) (map[string]interface{}, error) {
		return nil, errors.New("hasura down")
	}}

	eval := NewBranchEvaluatorWithHasura(sqlxDB, fh)

	analytics, err := eval.EvaluateAnalytics(context.Background(), "branch-1", "tenant-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if analytics == nil || analytics.SelectionCount != 50 || analytics.TrendDirection != "stable" {
		t.Fatalf("unexpected analytics from SQL: %#v", analytics)
	}
	if got := GetHasuraFallbackCount("branch_evaluator"); got != 1 {
		t.Fatalf("expected branch_evaluator fallback count 1, got %d", got)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock expectations not met: %v", err)
	}
}

func TestEvaluateVoting_BranchEvaluator_HasuraSuccess(t *testing.T) {
	ResetHasuraFallbacks()
	fh := &fakeHasura{}
	fh.queryFn = func(q string, vars map[string]interface{}) (map[string]interface{}, error) {
		// Stakeholders JSON with 2 voters: both approve with weight 1 each => 100% approval
		trueVal := true
		stakeholdersData := []map[string]interface{}{
			{"role": "manager", "vote_weight": 1.0, "vote": &trueVal},
			{"role": "director", "vote_weight": 1.0, "vote": &trueVal},
		}
		stakeholdersJSON, _ := json.Marshal(stakeholdersData)
		return map[string]interface{}{"bp_collaborative_decisions": []interface{}{map[string]interface{}{"decision_id": "dec-1", "stakeholders": string(stakeholdersJSON), "votes_received": float64(2), "total_weight": float64(2), "outcome": "approved"}}}, nil
	}

	eval := NewBranchEvaluatorWithHasura(nil, fh)

	branch, err := eval.EvaluateVoting(context.Background(), "dec-1", "tenant-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branch != "approved_branch" {
		t.Fatalf("unexpected branch: %s", branch)
	}
	if got := GetHasuraFallbackCount("branch_evaluator"); got != 0 {
		t.Fatalf("expected 0 fallbacks, got %d", got)
	}
}

func TestEvaluateVoting_BranchEvaluator_HasuraDown_SQLFallback(t *testing.T) {
	ResetHasuraFallbacks()
	dbSQL, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	sqlxDB := sqlx.NewDb(dbSQL, "sqlmock")
	defer sqlxDB.Close()

	mock.ExpectQuery("SELECT decision_id, stakeholders, votes_received, total_weight, outcome").WithArgs("dec-1", "tenant-1").WillReturnRows(sqlmock.NewRows([]string{"decision_id", "stakeholders", "votes_received", "total_weight", "outcome"}).AddRow("dec-1", []byte("[]"), 3, float64(10), "rejected"))

	fh := &fakeHasura{queryFn: func(q string, vars map[string]interface{}) (map[string]interface{}, error) {
		return nil, errors.New("hasura down")
	}}

	eval := NewBranchEvaluatorWithHasura(sqlxDB, fh)

	branch, err := eval.EvaluateVoting(context.Background(), "dec-1", "tenant-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branch != "rejected_branch" {
		t.Fatalf("unexpected branch from SQL: %s", branch)
	}
	if got := GetHasuraFallbackCount("branch_evaluator"); got != 1 {
		t.Fatalf("expected branch_evaluator fallback count 1, got %d", got)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock expectations not met: %v", err)
	}
}

func TestLogBlockchainAudit_BranchEvaluator_HasuraSuccess(t *testing.T) {
	ResetHasuraFallbacks()
	fh := &fakeHasura{}
	fh.mutateFn = func(m string, vars map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"insert_bp_blockchain_audit_one": map[string]interface{}{"id": "audit-1"}}, nil
	}

	eval := NewBranchEvaluatorWithHasura(nil, fh)

	if err := eval.LogBlockchainAudit(context.Background(), "event-1", "decision-x", "tenant-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := GetHasuraFallbackCount("branch_evaluator"); got != 0 {
		t.Fatalf("expected 0 fallbacks, got %d", got)
	}
}

func TestLogBlockchainAudit_BranchEvaluator_HasuraDown_SQLFallback(t *testing.T) {
	ResetHasuraFallbacks()
	dbSQL, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	sqlxDB := sqlx.NewDb(dbSQL, "sqlmock")
	defer sqlxDB.Close()

	mock.ExpectExec("INSERT INTO bp_blockchain_audit").WithArgs("event-1", sqlmock.AnyArg(), "tenant-1").WillReturnResult(sqlmock.NewResult(1, 1))

	fh := &fakeHasura{mutateFn: func(m string, vars map[string]interface{}) (map[string]interface{}, error) {
		return nil, errors.New("hasura mutate down")
	}}

	eval := NewBranchEvaluatorWithHasura(sqlxDB, fh)

	if err := eval.LogBlockchainAudit(context.Background(), "event-1", "decision-x", "tenant-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := GetHasuraFallbackCount("branch_evaluator"); got != 1 {
		t.Fatalf("expected branch_evaluator fallback count 1, got %d", got)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock expectations not met: %v", err)
	}
}
