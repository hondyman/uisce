package mdm

import (
	"context"
	"database/sql"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func TestFetchLatestGoldenRecord_NoDBConfigured(t *testing.T) {
	engine := NewUniversalMasteringEngine(nil)
	_, err := engine.FetchLatestGoldenRecord(context.Background(), uuid.New(), "Counterparty", "CP-123")
	if err == nil {
		t.Fatal("expected an error when no database is configured")
	}
}

func TestFetchLatestGoldenRecord_ReturnsNilWhenNeverMastered(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	engine := NewUniversalMasteringEngine(sqlx.NewDb(db, "postgres"))
	tenantID := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT golden_attributes
		FROM catalog_mdm.golden_records_ledger
		WHERE tenant_id = $1 AND domain_key = $2 AND master_entity_sid = $3`)).
		WithArgs(tenantID, "Counterparty", "CP-999").
		WillReturnError(sql.ErrNoRows)

	attrs, err := engine.FetchLatestGoldenRecord(context.Background(), tenantID, "Counterparty", "CP-999")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if attrs != nil {
		t.Errorf("expected nil attrs for a never-mastered entity, got %v", attrs)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestFetchLatestGoldenRecord_ReturnsLatestAttributes(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	engine := NewUniversalMasteringEngine(sqlx.NewDb(db, "postgres"))
	tenantID := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT golden_attributes
		FROM catalog_mdm.golden_records_ledger
		WHERE tenant_id = $1 AND domain_key = $2 AND master_entity_sid = $3`)).
		WithArgs(tenantID, "Counterparty", "CP-123").
		WillReturnRows(sqlmock.NewRows([]string{"golden_attributes"}).
			AddRow([]byte(`{"risk_rating":"HIGH","country":"US"}`)))

	attrs, err := engine.FetchLatestGoldenRecord(context.Background(), tenantID, "Counterparty", "CP-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if attrs["risk_rating"] != "HIGH" {
		t.Errorf("expected risk_rating=HIGH, got %v", attrs["risk_rating"])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}
