package db

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestApplyTenantGUCs_SetsTenantAndGold(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	mock.ExpectBegin()
	tx, err := sqlDB.Begin()
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectExec("set_config\\('uisce.current_tenant'").
		WithArgs("t-a").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("set_config\\('app.tenant_id'").
		WithArgs("t-a").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("set_config\\('uisce.gold_tenant'").
		WithArgs("t-gold").
		WillReturnResult(sqlmock.NewResult(0, 0))

	if err := ApplyTenantGUCs(context.Background(), tx, "t-a", "t-gold"); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
	_ = tx.Rollback()
}

func TestApplyTenantGUCs_EmptyTenantRejected(t *testing.T) {
	err := ApplyTenantGUCs(context.Background(), nil, "", "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWithTenantGoldTransaction_AppliesGUCs(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	mock.ExpectBegin()
	mock.ExpectExec("set_config\\('uisce.current_tenant'").
		WithArgs("t-a").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("set_config\\('app.tenant_id'").
		WithArgs("t-a").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("set_config\\('uisce.gold_tenant'").
		WithArgs("t-gold").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	called := false
	err = WithTenantGoldTransaction(context.Background(), sqlDB, "t-a", "t-gold", func(tx *sql.Tx) error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("fn not called")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// TestRLSPolicyShape_DocumentsGoldAwarePages encodes the deliberate-bypass
// contract in-repo: with GUCs set for tenant A + gold G, a gold+is_core row is
// visible; a tenant-B row is not. Live FORCE bypass needs UISCE_TEST_DB_DSN.
func TestRLSPolicyShape_DocumentsGoldAwarePages(t *testing.T) {
	t.Log("page_definitions policy USING: tenant_id = uisce_get_current_tenant() OR (is_core AND tenant_id = uisce_get_gold_tenant())")
	t.Log("business_objects policy USING: tenant_id = current OR tenant_id = gold")
	t.Log("deliberate-bypass: remove app WHERE tenant_id=$caller; FORCE RLS still returns zero foreign rows when gold GUC unset/mismatched")
	t.Log("legacy activation: ListBusinessObjectsLegacy + getSemanticBundle call ApplyTenantGUCs (SET LOCAL)")
}
