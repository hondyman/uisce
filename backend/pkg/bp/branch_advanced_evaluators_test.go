package bp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func TestEvaluateAIModels_Success(t *testing.T) {
	dbSQL, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	sqlxDB := sqlx.NewDb(dbSQL, "sqlmock")
	defer sqlxDB.Close()

	eval := NewBranchEvaluator(sqlxDB)

	cfg := map[string]interface{}{"available_models": []map[string]interface{}{{"model_id": "m-x", "last_accuracy": 0.9, "accuracy_threshold": 0.5}}, "auto_switch_enabled": false}
	cfgB, _ := json.Marshal(cfg)

	branch, err := eval.EvaluateAIModels(context.Background(), cfgB, "tenant-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branch == "" {
		t.Fatalf("expected non-empty branch")
	}
}
