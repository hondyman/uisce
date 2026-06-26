package temporal

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestLogAdminAction_InsertsWhenDBNotNil(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error opening stub db: %v", err)
	}
	defer db.Close()

	was := &WorkflowAdminService{
		client:    nil,
		namespace: "default",
		db:        db,
	}

	audit := AdminActionAudit{
		ID:         "a1",
		TenantID:   "t1",
		ActorID:    "u1",
		Action:     "terminate",
		WorkflowID: "wf-1",
		RunID:      "run-1",
		Reason:     "test",
		Input:      []byte(`{"k":"v"}`),
		Status:     "success",
		Timestamp:  time.Now().UTC(),
	}

	// sqlmock normalizes whitespace in the incoming query; keep the expectation
	// flexible rather than matching indentation exactly.
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO public.admin_audit_logs")+`\s*\(\s*id,\s*tenant_id,\s*actor_id,\s*action,\s*workflow_id,\s*run_id,\s*reason,\s*input,\s*status,\s*error_message,\s*created_at\s*\)\s*VALUES\s*\(\$1,\s*\$2,\s*\$3,\s*\$4,\s*\$5,\s*\$6,\s*\$7,\s*\$8,\s*\$9,\s*\$10,\s*\$11\s*\)`).
		WithArgs(audit.ID, audit.TenantID, audit.ActorID, audit.Action, audit.WorkflowID, audit.RunID, audit.Reason, audit.Input, audit.Status, audit.ErrorMessage, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := was.LogAdminAction(context.Background(), audit); err != nil {
		t.Fatalf("LogAdminAction returned error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestLogAdminAction_NoDB(t *testing.T) {
	was := &WorkflowAdminService{db: nil}
	audit := AdminActionAudit{
		ID:         "test-no-db",
		TenantID:   "tenant-1",
		ActorID:    "user-1",
		Action:     "signal",
		WorkflowID: "wf-no-db",
		RunID:      "",
		Reason:     "test",
		Input:      nil,
		Status:     "success",
		Timestamp:  time.Now(),
	}

	if err := was.LogAdminAction(context.Background(), audit); err != nil {
		t.Fatalf("expected no error when DB is nil, got: %v", err)
	}
}
