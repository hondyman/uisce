package bp

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func TestSelectAIModel(t *testing.T) {
	dbSQL, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	sqlxDB := sqlx.NewDb(dbSQL, "sqlmock")
	defer sqlxDB.Close()

	eval := NewCompleteABranchEvaluator(sqlxDB, "tenant-1")
	id, err := eval.SelectAIModel(context.Background(), "step-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id == "" {
		t.Fatalf("expected non-empty model id")
	}
}
