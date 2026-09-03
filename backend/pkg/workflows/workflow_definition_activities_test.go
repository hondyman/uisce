package workflows

import (
	"context"
	"database/sql"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func TestActivityLoadWorkflowDefinition_RequiresWorkflowKey(t *testing.T) {
	a := NewWorkflowDefinitionActivities(nil)
	_, err := a.ActivityLoadWorkflowDefinition(context.Background(), "", "")
	if err == nil {
		t.Fatal("expected an error when workflowKey is empty")
	}
}

func TestActivityLoadWorkflowDefinition_LoadsCoreDefinition(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	a := NewWorkflowDefinitionActivities(sqlx.NewDb(db, "postgres"))

	def := []byte(`{"name":"Core Demo","startNodeId":"n1","nodes":{"n1":{"id":"n1","type":"END"}}}`)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT definition FROM workflow_definitions
			WHERE workflow_key = $1 AND tenant_id IS NULL AND is_active = TRUE`)).
		WithArgs("bp_risk_demo").
		WillReturnRows(sqlmock.NewRows([]string{"definition"}).AddRow(def))

	dsl, err := a.ActivityLoadWorkflowDefinition(context.Background(), "", "bp_risk_demo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dsl.Name != "Core Demo" {
		t.Errorf("expected Name=Core Demo, got %q", dsl.Name)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestActivityLoadWorkflowDefinition_PrefersTenantOverride(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	a := NewWorkflowDefinitionActivities(sqlx.NewDb(db, "postgres"))

	def := []byte(`{"name":"Tenant Override","startNodeId":"n1","nodes":{"n1":{"id":"n1","type":"END"}}}`)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT definition FROM workflow_definitions
			WHERE workflow_key = $1 AND tenant_id = $2 AND is_active = TRUE`)).
		WithArgs("bp_risk_demo", "tenant-a").
		WillReturnRows(sqlmock.NewRows([]string{"definition"}).AddRow(def))

	dsl, err := a.ActivityLoadWorkflowDefinition(context.Background(), "tenant-a", "bp_risk_demo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dsl.Name != "Tenant Override" {
		t.Errorf("expected the tenant-scoped row to win, got %q", dsl.Name)
	}
	// The core-row query must never run once the tenant-scoped lookup hits.
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestActivityLoadWorkflowDefinition_FallsBackToDefault(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	a := NewWorkflowDefinitionActivities(sqlx.NewDb(db, "postgres"))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT definition FROM workflow_definitions
			WHERE workflow_key = $1 AND tenant_id IS NULL AND is_active = TRUE`)).
		WithArgs("some_unknown_key").
		WillReturnError(sql.ErrNoRows)

	def := []byte(`{"name":"__default__","startNodeId":"n1","nodes":{"n1":{"id":"n1","type":"END"}}}`)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT definition FROM workflow_definitions
				WHERE workflow_key = '__default__' AND tenant_id IS NULL AND is_active = TRUE`)).
		WillReturnRows(sqlmock.NewRows([]string{"definition"}).AddRow(def))

	dsl, err := a.ActivityLoadWorkflowDefinition(context.Background(), "", "some_unknown_key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dsl.Name != "__default__" {
		t.Errorf("expected fallback to __default__, got %q", dsl.Name)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}
